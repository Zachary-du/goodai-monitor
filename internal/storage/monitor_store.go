package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

var (
	// ErrMonitorNotFound is returned when the queried monitor does not exist or does not belong to the user.
	ErrMonitorNotFound = errors.New("monitor not found")

	// ErrMonitorQuotaExceeded is returned when the user exceeds the maximum allowed monitor count.
	ErrMonitorQuotaExceeded = errors.New("monitor quota exceeded")
)

// MonitorStore manages monitor persistence and queries in SQLite.
type MonitorStore struct {
	db *DB
}

// NewMonitorStore initializes a MonitorStore and applies monitor schema migrations.
func NewMonitorStore(db *DB) (*MonitorStore, error) {
	if db == nil {
		return nil, errors.New("database connection cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.MigrateMonitorSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to run monitor schema migration: %w", err)
	}

	return &MonitorStore{db: db}, nil
}

// MigrateMonitorSchema creates the monitors table and associated indexes.
func (db *DB) MigrateMonitorSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS monitors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			interval_seconds INTEGER NOT NULL DEFAULT 60,
			timeout_seconds INTEGER NOT NULL DEFAULT 5,
			expected_status_code INTEGER NOT NULL DEFAULT 200,
			keyword TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'PENDING',
			last_check_at TEXT,
			last_latency_ms INTEGER NOT NULL DEFAULT 0,
			uptime_percent REAL NOT NULL DEFAULT 100.0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_monitors_user_id ON monitors(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_monitors_status ON monitors(status);`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute monitor migration query: %w", err)
		}
	}

	return nil
}

// CreateMonitor persists a new monitor record for the specified user.
func (s *MonitorStore) CreateMonitor(ctx context.Context, m *model.Monitor) (*model.Monitor, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	if m.Status == "" {
		m.Status = model.StatusPending
	}
	if m.IntervalSeconds <= 0 {
		m.IntervalSeconds = model.DefaultIntervalSeconds
	}
	if m.TimeoutSeconds <= 0 {
		m.TimeoutSeconds = model.DefaultTimeoutSeconds
	}
	if m.ExpectedStatusCode <= 0 {
		m.ExpectedStatusCode = model.DefaultExpectedStatusCode
	}
	if m.UptimePercent == 0 {
		m.UptimePercent = 100.0
	}

	query := `
		INSERT INTO monitors (
			user_id, name, url, interval_seconds, timeout_seconds,
			expected_status_code, keyword, status, last_check_at,
			last_latency_ms, uptime_percent, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastCheckStr sql.NullString
	if m.LastCheckAt != nil {
		lastCheckStr = sql.NullString{
			String: m.LastCheckAt.UTC().Format(time.RFC3339),
			Valid:  true,
		}
	}

	result, err := s.db.ExecContext(ctx, query,
		m.UserID,
		strings.TrimSpace(m.Name),
		strings.TrimSpace(m.URL),
		m.IntervalSeconds,
		m.TimeoutSeconds,
		m.ExpectedStatusCode,
		strings.TrimSpace(m.Keyword),
		m.Status,
		lastCheckStr,
		m.LastLatencyMs,
		m.UptimePercent,
		nowStr,
		nowStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert monitor: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	m.ID = id
	m.CreatedAt = now
	m.UpdatedAt = now
	return m, nil
}

// GetMonitorByIDAndUser retrieves a single monitor by ID, enforcing row-level user ownership.
func (s *MonitorStore) GetMonitorByIDAndUser(ctx context.Context, id int64, userID int64) (*model.Monitor, error) {
	query := `
		SELECT
			id, user_id, name, url, interval_seconds, timeout_seconds,
			expected_status_code, keyword, status, last_check_at,
			last_latency_ms, uptime_percent, created_at, updated_at
		FROM monitors
		WHERE id = ? AND user_id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id, userID)

	m, err := scanMonitorRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMonitorNotFound
		}
		return nil, fmt.Errorf("failed to query monitor: %w", err)
	}

	return m, nil
}

// ListMonitorsByUser retrieves all monitors owned by the specified user, ordered by creation time descending.
func (s *MonitorStore) ListMonitorsByUser(ctx context.Context, userID int64) ([]*model.Monitor, error) {
	query := `
		SELECT
			id, user_id, name, url, interval_seconds, timeout_seconds,
			expected_status_code, keyword, status, last_check_at,
			last_latency_ms, uptime_percent, created_at, updated_at
		FROM monitors
		WHERE user_id = ?
		ORDER BY id DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query monitors for user: %w", err)
	}
	defer rows.Close()

	monitors := make([]*model.Monitor, 0)
	for rows.Next() {
		m, err := scanMonitorRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monitor row: %w", err)
		}
		monitors = append(monitors, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading monitor rows: %w", err)
	}

	return monitors, nil
}

// UpdateMonitor updates a monitor's user-configurable fields, strictly enforcing ownership.
func (s *MonitorStore) UpdateMonitor(ctx context.Context, userID int64, m *model.Monitor) (*model.Monitor, error) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	query := `
		UPDATE monitors
		SET name = ?, url = ?, interval_seconds = ?, timeout_seconds = ?,
		    expected_status_code = ?, keyword = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		strings.TrimSpace(m.Name),
		strings.TrimSpace(m.URL),
		m.IntervalSeconds,
		m.TimeoutSeconds,
		m.ExpectedStatusCode,
		strings.TrimSpace(m.Keyword),
		nowStr,
		m.ID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update monitor: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return nil, ErrMonitorNotFound
	}

	return s.GetMonitorByIDAndUser(ctx, m.ID, userID)
}

// DeleteMonitor deletes a monitor by ID, strictly enforcing ownership.
func (s *MonitorStore) DeleteMonitor(ctx context.Context, id int64, userID int64) error {
	query := `DELETE FROM monitors WHERE id = ? AND user_id = ?`

	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete monitor: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrMonitorNotFound
	}

	return nil
}

// ToggleMonitorPause toggles the pause status of a monitor: PAUSED -> PENDING, otherwise -> PAUSED.
func (s *MonitorStore) ToggleMonitorPause(ctx context.Context, id int64, userID int64) (*model.Monitor, error) {
	current, err := s.GetMonitorByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	var newStatus string
	if current.Status == model.StatusPaused {
		newStatus = model.StatusPending
	} else {
		newStatus = model.StatusPaused
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	query := `
		UPDATE monitors
		SET status = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`

	result, err := s.db.ExecContext(ctx, query, newStatus, nowStr, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle monitor status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return nil, ErrMonitorNotFound
	}

	current.Status = newStatus
	current.UpdatedAt = now
	return current, nil
}

// CountMonitorsByUser returns the total number of monitors owned by the specified user.
func (s *MonitorStore) CountMonitorsByUser(ctx context.Context, userID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM monitors WHERE user_id = ?`

	var count int64
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count monitors for user: %w", err)
	}

	return count, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanMonitorRow(row rowScanner) (*model.Monitor, error) {
	var m model.Monitor
	var lastCheckStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&m.ID,
		&m.UserID,
		&m.Name,
		&m.URL,
		&m.IntervalSeconds,
		&m.TimeoutSeconds,
		&m.ExpectedStatusCode,
		&m.Keyword,
		&m.Status,
		&lastCheckStr,
		&m.LastLatencyMs,
		&m.UptimePercent,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if lastCheckStr.Valid && lastCheckStr.String != "" {
		t := parseTimeOrNow(lastCheckStr.String)
		m.LastCheckAt = &t
	}
	m.CreatedAt = parseTimeOrNow(createdAtStr)
	m.UpdatedAt = parseTimeOrNow(updatedAtStr)

	return &m, nil
}

func scanMonitorRows(rows *sql.Rows) (*model.Monitor, error) {
	return scanMonitorRow(rows)
}
