package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps standard sql.DB with GoodAI Monitor specific helpers.
type DB struct {
	*sql.DB
}

// Open initializes and configures a pure-Go SQLite database connection.
// It sets WAL journal mode, a 2MB page cache limit, and enforces a single-writer pool.
func Open(dbPath string) (*DB, error) {
	if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file:") {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory '%s': %w", dir, err)
		}
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite connection: %w", err)
	}

	// Restrict connection pool to single-writer to prevent concurrency locks
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	// Apply performance and memory budget pragmas
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA cache_size = -2000;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}

	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("failed to execute pragma '%s': %w", pragma, err)
		}
	}

	db := &DB{DB: sqlDB}

	// Run initial schema migrations
	if err := db.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return db, nil
}

// migrate creates the foundational tables if they do not exist.
func (db *DB) migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS system_metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return err
		}
	}

	// Record initial migration version 1 if absent
	_, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO schema_migrations (version, applied_at)
		VALUES (1, CURRENT_TIMESTAMP);
	`)
	return err
}

// PingContext verifies the database is responsive within the given context.
func (db *DB) PingContext(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

// Close gracefully closes the underlying database pool.
func (db *DB) Close() error {
	return db.DB.Close()
}
