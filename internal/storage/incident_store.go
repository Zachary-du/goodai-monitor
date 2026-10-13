package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

var (
	// ErrIncidentNotFound indicates that the requested incident does not exist.
	ErrIncidentNotFound = errors.New("incident record not found")
)

// IncidentStore manages persistence, state updates, and queries for downtime incidents.
type IncidentStore struct {
	db *DB
}

// NewIncidentStore initializes an IncidentStore and runs the incidents schema migration.
func NewIncidentStore(db *DB) (*IncidentStore, error) {
	if db == nil {
		return nil, errors.New("database connection cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.MigrateIncidentSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to run incident schema migration: %w", err)
	}

	return &IncidentStore{db: db}, nil
}

// MigrateIncidentSchema creates the incidents table and indexes.
func (db *DB) MigrateIncidentSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS incidents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			monitor_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL,
			resolved_at TEXT,
			status TEXT NOT NULL,
			cause TEXT NOT NULL DEFAULT '',
			duration_seconds INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_monitor_id ON incidents(monitor_id);`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_user_id ON incidents(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_started_at ON incidents(started_at);`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute incident migration query: %w", err)
		}
	}

	return nil
}

// CreateIncident persists a new downtime incident in the OPEN state.
func (s *IncidentStore) CreateIncident(ctx context.Context, inc *model.Incident) (*model.Incident, error) {
	if inc == nil {
		return nil, errors.New("incident cannot be nil")
	}
	if inc.StartedAt.IsZero() {
		inc.StartedAt = time.Now().UTC()
	}
	if inc.Status == "" {
		inc.Status = model.IncidentStatusOpen
	}

	startedAtStr := inc.StartedAt.Format(time.RFC3339)

	query := `
		INSERT INTO incidents (
			monitor_id, user_id, started_at, resolved_at, status, cause, duration_seconds
		) VALUES (?, ?, ?, NULL, ?, ?, 0)
	`

	res, err := s.db.ExecContext(ctx, query,
		inc.MonitorID,
		inc.UserID,
		startedAtStr,
		inc.Status,
		inc.Cause,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert incident: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	inc.ID = id

	return inc, nil
}

// ResolveIncident marks an open incident as RESOLVED, records resolved_at, and calculates duration_seconds.
func (s *IncidentStore) ResolveIncident(ctx context.Context, id int64, resolvedAt time.Time, durationSeconds int64) error {
	if resolvedAt.IsZero() {
		resolvedAt = time.Now().UTC()
	}
	resolvedAtStr := resolvedAt.Format(time.RFC3339)

	query := `
		UPDATE incidents
		SET status = ?,
		    resolved_at = ?,
		    duration_seconds = ?
		WHERE id = ? AND status = ?
	`

	res, err := s.db.ExecContext(ctx, query,
		model.IncidentStatusResolved,
		resolvedAtStr,
		durationSeconds,
		id,
		model.IncidentStatusOpen,
	)
	if err != nil {
		return fmt.Errorf("failed to update incident resolution: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrIncidentNotFound
	}

	return nil
}

// GetOpenIncidentByMonitor retrieves the current active OPEN incident for a monitor, if any exists.
func (s *IncidentStore) GetOpenIncidentByMonitor(ctx context.Context, monitorID int64) (*model.Incident, error) {
	query := `
		SELECT i.id, i.monitor_id, i.user_id, i.started_at, i.resolved_at, i.status, i.cause, i.duration_seconds,
		       m.name, m.url
		FROM incidents i
		JOIN monitors m ON i.monitor_id = m.id
		WHERE i.monitor_id = ? AND i.status = ?
		ORDER BY i.id DESC
		LIMIT 1
	`

	var inc model.Incident
	var startedAtStr string
	var resolvedAtStr sql.NullString
	var resolvedSec sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, monitorID, model.IncidentStatusOpen).Scan(
		&inc.ID,
		&inc.MonitorID,
		&inc.UserID,
		&startedAtStr,
		&resolvedAtStr,
		&inc.Status,
		&inc.Cause,
		&resolvedSec,
		&inc.MonitorName,
		&inc.MonitorURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIncidentNotFound
		}
		return nil, fmt.Errorf("failed to query open incident: %w", err)
	}

	inc.StartedAt = parseTimeOrNow(startedAtStr)
	if resolvedAtStr.Valid {
		t := parseTimeOrNow(resolvedAtStr.String)
		inc.ResolvedAt = &t
	}
	if resolvedSec.Valid {
		inc.DurationSeconds = resolvedSec.Int64
	}

	return &inc, nil
}

// GetIncidentByID retrieves a specific incident by its unique ID.
func (s *IncidentStore) GetIncidentByID(ctx context.Context, id int64) (*model.Incident, error) {
	query := `
		SELECT i.id, i.monitor_id, i.user_id, i.started_at, i.resolved_at, i.status, i.cause, i.duration_seconds,
		       m.name, m.url
		FROM incidents i
		JOIN monitors m ON i.monitor_id = m.id
		WHERE i.id = ?
		LIMIT 1
	`

	var inc model.Incident
	var startedAtStr string
	var resolvedAtStr sql.NullString
	var resolvedSec sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&inc.ID,
		&inc.MonitorID,
		&inc.UserID,
		&startedAtStr,
		&resolvedAtStr,
		&inc.Status,
		&inc.Cause,
		&resolvedSec,
		&inc.MonitorName,
		&inc.MonitorURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIncidentNotFound
		}
		return nil, fmt.Errorf("failed to query incident by id: %w", err)
	}

	inc.StartedAt = parseTimeOrNow(startedAtStr)
	if resolvedAtStr.Valid {
		t := parseTimeOrNow(resolvedAtStr.String)
		inc.ResolvedAt = &t
	}
	if resolvedSec.Valid {
		inc.DurationSeconds = resolvedSec.Int64
	}

	return &inc, nil
}

// ListIncidentsByUser retrieves a paginated list of incidents for a specific tenant, optionally filtered by status.
func (s *IncidentStore) ListIncidentsByUser(ctx context.Context, userID int64, statusFilter string, limit int, offset int) ([]*model.Incident, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var query string
	var args []any

	if statusFilter != "" && statusFilter != "ALL" && statusFilter != "all" {
		query = `
			SELECT i.id, i.monitor_id, i.user_id, i.started_at, i.resolved_at, i.status, i.cause, i.duration_seconds,
			       m.name, m.url
			FROM incidents i
			JOIN monitors m ON i.monitor_id = m.id
			WHERE i.user_id = ? AND UPPER(i.status) = UPPER(?)
			ORDER BY i.id DESC
			LIMIT ? OFFSET ?
		`
		args = []any{userID, statusFilter, limit, offset}
	} else {
		query = `
			SELECT i.id, i.monitor_id, i.user_id, i.started_at, i.resolved_at, i.status, i.cause, i.duration_seconds,
			       m.name, m.url
			FROM incidents i
			JOIN monitors m ON i.monitor_id = m.id
			WHERE i.user_id = ?
			ORDER BY i.id DESC
			LIMIT ? OFFSET ?
		`
		args = []any{userID, limit, offset}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query incidents: %w", err)
	}
	defer rows.Close()

	incidents := make([]*model.Incident, 0, limit)
	for rows.Next() {
		var inc model.Incident
		var startedAtStr string
		var resolvedAtStr sql.NullString
		var resolvedSec sql.NullInt64

		err := rows.Scan(
			&inc.ID,
			&inc.MonitorID,
			&inc.UserID,
			&startedAtStr,
			&resolvedAtStr,
			&inc.Status,
			&inc.Cause,
			&resolvedSec,
			&inc.MonitorName,
			&inc.MonitorURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident row: %w", err)
		}

		inc.StartedAt = parseTimeOrNow(startedAtStr)
		if resolvedAtStr.Valid {
			t := parseTimeOrNow(resolvedAtStr.String)
			inc.ResolvedAt = &t
		}
		if resolvedSec.Valid {
			inc.DurationSeconds = resolvedSec.Int64
		}

		incidents = append(incidents, &inc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading incident rows: %w", err)
	}

	return incidents, nil
}

// GetIncidentSummaryByUser calculates aggregate open, resolved, and total incident counts for a tenant.
func (s *IncidentStore) GetIncidentSummaryByUser(ctx context.Context, userID int64) (*model.IncidentSummary, error) {
	query := `
		SELECT
			SUM(CASE WHEN status = 'OPEN' THEN 1 ELSE 0 END) AS open_count,
			SUM(CASE WHEN status = 'RESOLVED' THEN 1 ELSE 0 END) AS resolved_count,
			COUNT(*) AS total_count
		FROM incidents
		WHERE user_id = ?
	`

	var openCount, resolvedCount, totalCount sql.NullInt64
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&openCount, &resolvedCount, &totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident summary: %w", err)
	}

	return &model.IncidentSummary{
		OpenCount:     int(openCount.Int64),
		ResolvedCount: int(resolvedCount.Int64),
		TotalCount:    int(totalCount.Int64),
	}, nil
}

// CountActiveIncidentsByUser returns the count of currently OPEN incidents for a tenant.
func (s *IncidentStore) CountActiveIncidentsByUser(ctx context.Context, userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM incidents WHERE user_id = ? AND status = 'OPEN'`
	var count int
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active incidents: %w", err)
	}
	return count, nil
}
