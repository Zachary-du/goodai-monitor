package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

var (
	// ErrSecurityReportNotFound is returned when a security report is not found or not owned by user.
	ErrSecurityReportNotFound = errors.New("security report not found")
)

// SecurityStore manages persistence of security audit reports in SQLite.
type SecurityStore struct {
	db *DB
}

// NewSecurityStore initializes a SecurityStore and executes necessary migrations.
func NewSecurityStore(db *DB) (*SecurityStore, error) {
	if db == nil {
		return nil, errors.New("database connection cannot be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.MigrateSecuritySchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to run security schema migration: %w", err)
	}

	return &SecurityStore{db: db}, nil
}

// MigrateSecuritySchema creates the security_reports table and indexes.
func (db *DB) MigrateSecuritySchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS security_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			target_url TEXT NOT NULL,
			target_host TEXT NOT NULL,
			target_ip TEXT NOT NULL,
			composite_score INTEGER NOT NULL,
			grade TEXT NOT NULL,
			grade_label TEXT NOT NULL,
			execution_ms INTEGER NOT NULL,
			domain_scores_json TEXT NOT NULL,
			findings_json TEXT NOT NULL,
			attestation_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_security_reports_user ON security_reports(user_id, created_at DESC);`,
	}

	for _, q := range queries {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return err
		}
	}

	return nil
}

// SaveReport stores a new SecurityAuditReport in the database.
func (s *SecurityStore) SaveReport(ctx context.Context, report *model.SecurityAuditReport) error {
	if report == nil {
		return errors.New("report cannot be nil")
	}

	domainJSON, err := json.Marshal(report.DomainScores)
	if err != nil {
		return fmt.Errorf("failed to serialize domain scores: %w", err)
	}

	findingsJSON, err := json.Marshal(report.Findings)
	if err != nil {
		return fmt.Errorf("failed to serialize findings: %w", err)
	}

	createdAtStr := report.CreatedAt.UTC().Format(time.RFC3339Nano)

	query := `
		INSERT INTO security_reports (
			user_id, target_url, target_host, target_ip, composite_score,
			grade, grade_label, execution_ms, domain_scores_json, findings_json,
			attestation_hash, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := s.db.ExecContext(ctx, query,
		report.UserID,
		report.TargetURL,
		report.TargetHost,
		report.TargetIP,
		report.CompositeScore,
		report.Grade,
		report.GradeLabel,
		report.ExecutionMs,
		string(domainJSON),
		string(findingsJSON),
		report.Attestation,
		createdAtStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert security report: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to retrieve last insert id: %w", err)
	}

	report.ID = id
	return nil
}

// GetReport retrieves a single report by ID ensuring user isolation.
func (s *SecurityStore) GetReport(ctx context.Context, id int64, userID int64) (*model.SecurityAuditReport, error) {
	query := `
		SELECT id, user_id, target_url, target_host, target_ip, composite_score,
		       grade, grade_label, execution_ms, domain_scores_json, findings_json,
		       attestation_hash, created_at
		FROM security_reports
		WHERE id = ? AND user_id = ?
	`

	var (
		report       model.SecurityAuditReport
		domainJSON   string
		findingsJSON string
		createdAtStr string
	)

	err := s.db.QueryRowContext(ctx, query, id, userID).Scan(
		&report.ID,
		&report.UserID,
		&report.TargetURL,
		&report.TargetHost,
		&report.TargetIP,
		&report.CompositeScore,
		&report.Grade,
		&report.GradeLabel,
		&report.ExecutionMs,
		&domainJSON,
		&findingsJSON,
		&report.Attestation,
		&createdAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSecurityReportNotFound
		}
		return nil, fmt.Errorf("failed to query security report: %w", err)
	}

	t, err := time.Parse(time.RFC3339Nano, createdAtStr)
	if err != nil {
		// Fallback parse
		t, _ = time.Parse(time.RFC3339, createdAtStr)
	}
	report.CreatedAt = t

	if err := json.Unmarshal([]byte(domainJSON), &report.DomainScores); err != nil {
		return nil, fmt.Errorf("failed to parse domain scores: %w", err)
	}

	if err := json.Unmarshal([]byte(findingsJSON), &report.Findings); err != nil {
		return nil, fmt.Errorf("failed to parse findings: %w", err)
	}

	return &report, nil
}

// ListReports retrieves previous audit reports for a specific user.
func (s *SecurityStore) ListReports(ctx context.Context, userID int64, limit int) ([]*model.SecurityAuditReport, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	query := `
		SELECT id, user_id, target_url, target_host, target_ip, composite_score,
		       grade, grade_label, execution_ms, domain_scores_json, findings_json,
		       attestation_hash, created_at
		FROM security_reports
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query security reports: %w", err)
	}
	defer rows.Close()

	var reports []*model.SecurityAuditReport
	for rows.Next() {
		var (
			report       model.SecurityAuditReport
			domainJSON   string
			findingsJSON string
			createdAtStr string
		)

		err := rows.Scan(
			&report.ID,
			&report.UserID,
			&report.TargetURL,
			&report.TargetHost,
			&report.TargetIP,
			&report.CompositeScore,
			&report.Grade,
			&report.GradeLabel,
			&report.ExecutionMs,
			&domainJSON,
			&findingsJSON,
			&report.Attestation,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan security report: %w", err)
		}

		t, err := time.Parse(time.RFC3339Nano, createdAtStr)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, createdAtStr)
		}
		report.CreatedAt = t

		_ = json.Unmarshal([]byte(domainJSON), &report.DomainScores)
		_ = json.Unmarshal([]byte(findingsJSON), &report.Findings)

		reports = append(reports, &report)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating security reports: %w", err)
	}

	return reports, nil
}

// DeleteReport deletes a report by ID ensuring user isolation.
func (s *SecurityStore) DeleteReport(ctx context.Context, id int64, userID int64) error {
	query := `DELETE FROM security_reports WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete security report: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSecurityReportNotFound
	}

	return nil
}
