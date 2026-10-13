package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

var (
	// ErrCheckNotFound is returned when the queried check log does not exist.
	ErrCheckNotFound = errors.New("check record not found")
)

// CheckStore manages check telemetry records and metrics calculation in SQLite.
type CheckStore struct {
	db *DB
}

// NewCheckStore initializes a CheckStore and applies the checks schema migration.
func NewCheckStore(db *DB) (*CheckStore, error) {
	if db == nil {
		return nil, errors.New("database connection cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.MigrateCheckSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to run check schema migration: %w", err)
	}

	return &CheckStore{db: db}, nil
}

// MigrateCheckSchema creates the checks table and associated indexing.
func (db *DB) MigrateCheckSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS checks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			monitor_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			status_code INTEGER NOT NULL DEFAULT 0,
			latency_ms INTEGER NOT NULL DEFAULT 0,
			dns_lookup_ms INTEGER NOT NULL DEFAULT 0,
			tcp_connect_ms INTEGER NOT NULL DEFAULT 0,
			tls_handshake_ms INTEGER NOT NULL DEFAULT 0,
			ttfb_ms INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			cert_expiry_days INTEGER,
			created_at TEXT NOT NULL,
			FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_checks_monitor_id ON checks(monitor_id);`,
		`CREATE INDEX IF NOT EXISTS idx_checks_created_at ON checks(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_checks_status ON checks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_checks_monitor_created ON checks(monitor_id, created_at);`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute check migration query: %w", err)
		}
	}

	return nil
}

// RecordCheck persists a new probe check record and updates the monitor's live status and telemetry.
func (s *CheckStore) RecordCheck(ctx context.Context, check *model.Check) (*model.Check, error) {
	if check.CreatedAt.IsZero() {
		check.CreatedAt = time.Now().UTC()
	}
	createdAtStr := check.CreatedAt.Format(time.RFC3339)

	var certExpiryVal sql.NullInt64
	if check.CertExpiryDays != nil {
		certExpiryVal = sql.NullInt64{
			Int64: int64(*check.CertExpiryDays),
			Valid: true,
		}
	}

	insertQuery := `
		INSERT INTO checks (
			monitor_id, status, status_code, latency_ms,
			dns_lookup_ms, tcp_connect_ms, tls_handshake_ms, ttfb_ms,
			error_message, cert_expiry_days, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := s.db.ExecContext(ctx, insertQuery,
		check.MonitorID,
		check.Status,
		check.StatusCode,
		check.LatencyMs,
		check.DNSLookupMs,
		check.TCPConnectMs,
		check.TLSHandshakeMs,
		check.TTFBMs,
		check.ErrorMessage,
		certExpiryVal,
		createdAtStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert check record: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	check.ID = id

	// Calculate live 24-hour uptime percentage to sync with monitors table
	uptime24h := s.calculate24hUptime(ctx, check.MonitorID, check.CreatedAt)

	// Update the monitor's last telemetry fields while preserving PAUSED state
	updateMonitorQuery := `
		UPDATE monitors
		SET last_check_at = ?,
		    last_latency_ms = ?,
		    status = CASE WHEN status = 'PAUSED' THEN 'PAUSED' ELSE ? END,
		    uptime_percent = ?,
		    updated_at = ?
		WHERE id = ?
	`
	_, _ = s.db.ExecContext(ctx, updateMonitorQuery,
		createdAtStr,
		check.LatencyMs,
		check.Status,
		uptime24h,
		createdAtStr,
		check.MonitorID,
	)

	return check, nil
}

func (s *CheckStore) calculate24hUptime(ctx context.Context, monitorID int64, now time.Time) float64 {
	cutoff := now.Add(-24 * time.Hour).Format(time.RFC3339)
	query := `
		SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN status = 'UP' THEN 1 ELSE 0 END) AS successful
		FROM checks
		WHERE monitor_id = ? AND created_at >= ?
	`

	var total, successful sql.NullInt64
	err := s.db.QueryRowContext(ctx, query, monitorID, cutoff).Scan(&total, &successful)
	if err != nil || total.Int64 == 0 {
		return 100.0
	}

	pct := (float64(successful.Int64) / float64(total.Int64)) * 100.0
	return math.Round(pct*100) / 100
}

// GetRecentChecksByMonitor retrieves the most recent check records for a given monitor, ordered by ID descending.
func (s *CheckStore) GetRecentChecksByMonitor(ctx context.Context, monitorID int64, limit int) ([]*model.Check, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT
			id, monitor_id, status, status_code, latency_ms,
			dns_lookup_ms, tcp_connect_ms, tls_handshake_ms, ttfb_ms,
			error_message, cert_expiry_days, created_at
		FROM checks
		WHERE monitor_id = ?
		ORDER BY id DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent checks: %w", err)
	}
	defer rows.Close()

	checks := make([]*model.Check, 0, limit)
	for rows.Next() {
		var c model.Check
		var certExpiryVal sql.NullInt64
		var createdAtStr string

		err := rows.Scan(
			&c.ID,
			&c.MonitorID,
			&c.Status,
			&c.StatusCode,
			&c.LatencyMs,
			&c.DNSLookupMs,
			&c.TCPConnectMs,
			&c.TLSHandshakeMs,
			&c.TTFBMs,
			&c.ErrorMessage,
			&certExpiryVal,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan check row: %w", err)
		}

		if certExpiryVal.Valid {
			days := int(certExpiryVal.Int64)
			c.CertExpiryDays = &days
		}
		c.CreatedAt = parseTimeOrNow(createdAtStr)
		checks = append(checks, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading check rows: %w", err)
	}

	return checks, nil
}

// GetLatencyHistory retrieves historical check timestamps, latencies, and statuses since a given cutoff.
func (s *CheckStore) GetLatencyHistory(ctx context.Context, monitorID int64, since time.Time) ([]model.LatencyPoint, error) {
	sinceStr := since.Format(time.RFC3339)
	query := `
		SELECT created_at, latency_ms, status
		FROM checks
		WHERE monitor_id = ? AND created_at >= ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, monitorID, sinceStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query latency history: %w", err)
	}
	defer rows.Close()

	points := make([]model.LatencyPoint, 0)
	for rows.Next() {
		var p model.LatencyPoint
		var createdAtStr string

		if err := rows.Scan(&createdAtStr, &p.LatencyMs, &p.Status); err != nil {
			return nil, fmt.Errorf("failed to scan latency point: %w", err)
		}

		p.Timestamp = parseTimeOrNow(createdAtStr)
		points = append(points, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading latency rows: %w", err)
	}

	return points, nil
}

// CalculateUptimeStats computes 24-hour, 7-day, and 30-day SLA uptime percentages,
// average 24-hour latency, and fetches the 24-hour latency history.
func (s *CheckStore) CalculateUptimeStats(ctx context.Context, monitorID int64) (*model.MetricsSummary, error) {
	now := time.Now().UTC()
	t24h := now.Add(-24 * time.Hour).Format(time.RFC3339)
	t7d := now.Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	t30d := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)

	query := `
		SELECT
			COUNT(CASE WHEN created_at >= ? THEN 1 END) AS total_24h,
			SUM(CASE WHEN created_at >= ? AND status = 'UP' THEN 1 ELSE 0 END) AS up_24h,
			AVG(CASE WHEN created_at >= ? THEN latency_ms END) AS avg_lat_24h,
			COUNT(CASE WHEN created_at >= ? THEN 1 END) AS total_7d,
			SUM(CASE WHEN created_at >= ? AND status = 'UP' THEN 1 ELSE 0 END) AS up_7d,
			COUNT(CASE WHEN created_at >= ? THEN 1 END) AS total_30d,
			SUM(CASE WHEN created_at >= ? AND status = 'UP' THEN 1 ELSE 0 END) AS up_30d
		FROM checks
		WHERE monitor_id = ? AND created_at >= ?
	`

	var total24h, up24h, total7d, up7d, total30d, up30d sql.NullInt64
	var avgLat24h sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query,
		t24h, t24h, t24h,
		t7d, t7d,
		t30d, t30d,
		monitorID, t30d,
	).Scan(
		&total24h, &up24h, &avgLat24h,
		&total7d, &up7d,
		&total30d, &up30d,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate uptime statistics: %w", err)
	}

	computePercent := func(total, up sql.NullInt64) float64 {
		if !total.Valid || total.Int64 == 0 {
			return 100.0
		}
		var successful int64
		if up.Valid {
			successful = up.Int64
		}
		pct := (float64(successful) / float64(total.Int64)) * 100.0
		return math.Round(pct*100) / 100
	}

	summary := &model.MetricsSummary{
		MonitorID:      monitorID,
		Uptime24h:      computePercent(total24h, up24h),
		Uptime7d:       computePercent(total7d, up7d),
		Uptime30d:      computePercent(total30d, up30d),
		TotalChecks24h: total24h.Int64,
	}

	if avgLat24h.Valid {
		summary.AvgLatency24h = math.Round(avgLat24h.Float64*10) / 10
	}

	// Fetch 24-hour latency points for charting
	points, err := s.GetLatencyHistory(ctx, monitorID, now.Add(-24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latency history for metrics: %w", err)
	}
	summary.LatencyHistory = points

	return summary, nil
}

// PruneOldChecks deletes check records older than the specified retention duration.
// Returns the count of deleted rows.
func (s *CheckStore) PruneOldChecks(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	query := `DELETE FROM checks WHERE created_at < ?`

	res, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune check records: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve pruned row count: %w", err)
	}

	return rows, nil
}

// GetAllActiveMonitors retrieves all monitors that are currently active (not paused).
func (s *CheckStore) GetAllActiveMonitors(ctx context.Context) ([]*model.Monitor, error) {
	query := `
		SELECT
			id, user_id, name, url, interval_seconds, timeout_seconds,
			expected_status_code, keyword, status, last_check_at,
			last_latency_ms, uptime_percent, created_at, updated_at
		FROM monitors
		WHERE status != 'PAUSED'
		ORDER BY id ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]*model.Monitor, 0)
	for rows.Next() {
		m, err := scanMonitorRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active monitor row: %w", err)
		}
		monitors = append(monitors, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading active monitor rows: %w", err)
	}

	return monitors, nil
}

// GetAllMonitors retrieves all monitors regardless of status.
func (s *CheckStore) GetAllMonitors(ctx context.Context) ([]*model.Monitor, error) {
	query := `
		SELECT
			id, user_id, name, url, interval_seconds, timeout_seconds,
			expected_status_code, keyword, status, last_check_at,
			last_latency_ms, uptime_percent, created_at, updated_at
		FROM monitors
		ORDER BY id ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]*model.Monitor, 0)
	for rows.Next() {
		m, err := scanMonitorRow(rows)
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

// GetMonitorByID retrieves a monitor target by its unique ID.
func (s *CheckStore) GetMonitorByID(ctx context.Context, id int64) (*model.Monitor, error) {
	query := `
		SELECT
			id, user_id, name, url, interval_seconds, timeout_seconds,
			expected_status_code, keyword, status, last_check_at,
			last_latency_ms, uptime_percent, created_at, updated_at
		FROM monitors
		WHERE id = ?
	`

	row := s.db.QueryRowContext(ctx, query, id)
	m, err := scanMonitorRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMonitorNotFound
		}
		return nil, fmt.Errorf("failed to get monitor by id: %w", err)
	}

	return m, nil
}
