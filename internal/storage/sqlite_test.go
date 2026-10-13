package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpen_DiskDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "subfolder", "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("PingContext failed: %v", err)
	}

	// Verify PRAGMA journal_mode is WAL
	var journalMode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Errorf("expected journal_mode 'wal', got '%s'", journalMode)
	}

	// Verify PRAGMA cache_size is -2000
	var cacheSize int
	if err := db.QueryRowContext(ctx, "PRAGMA cache_size;").Scan(&cacheSize); err != nil {
		t.Fatalf("failed to query cache_size: %v", err)
	}
	if cacheSize != -2000 {
		t.Errorf("expected cache_size -2000, got %d", cacheSize)
	}

	// Verify PRAGMA foreign_keys is 1 (ON)
	var foreignKeys int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&foreignKeys); err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("expected foreign_keys 1, got %d", foreignKeys)
	}

	// Verify schema migration version 1 is present
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = 1;").Scan(&count); err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration record for version 1, got %d", count)
	}

	// Verify system_metadata table works
	_, err = db.ExecContext(ctx, "INSERT INTO system_metadata (key, value, updated_at) VALUES ('version', '1.0.0', CURRENT_TIMESTAMP);")
	if err != nil {
		t.Fatalf("failed to insert into system_metadata: %v", err)
	}

	var val string
	if err := db.QueryRowContext(ctx, "SELECT value FROM system_metadata WHERE key = 'version';").Scan(&val); err != nil {
		t.Fatalf("failed to select from system_metadata: %v", err)
	}
	if val != "1.0.0" {
		t.Errorf("expected '1.0.0', got '%s'", val)
	}
}

func TestOpen_ConnectionPool(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "pool_test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("expected MaxOpenConnections = 1, got %d", stats.MaxOpenConnections)
	}
}
