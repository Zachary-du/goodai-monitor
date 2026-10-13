package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure relevant environment variables are cleared
	t.Setenv("PORT", "")
	t.Setenv("HOST", "")
	t.Setenv("STORAGE_PATH", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("TZ", "")
	t.Setenv("DATA_RETENTION_DAYS", "")

	cfg := Load()

	if cfg.Port != "8090" {
		t.Errorf("expected default Port '8090', got '%s'", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected default Host '0.0.0.0', got '%s'", cfg.Host)
	}
	if cfg.StoragePath != "./data/monitor.db" {
		t.Errorf("expected default StoragePath './data/monitor.db', got '%s'", cfg.StoragePath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.Timezone != "Pacific/Auckland" {
		t.Errorf("expected default Timezone 'Pacific/Auckland', got '%s'", cfg.Timezone)
	}
	if cfg.DataRetentionDays != 14 {
		t.Errorf("expected default DataRetentionDays 14, got %d", cfg.DataRetentionDays)
	}
	if cfg.Address() != ":8090" {
		t.Errorf("expected default Address ':8090', got '%s'", cfg.Address())
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("STORAGE_PATH", "/tmp/custom.db")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("TZ", "UTC")
	t.Setenv("DATA_RETENTION_DAYS", "30")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got '%s'", cfg.Port)
	}
	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected Host '127.0.0.1', got '%s'", cfg.Host)
	}
	if cfg.StoragePath != "/tmp/custom.db" {
		t.Errorf("expected StoragePath '/tmp/custom.db', got '%s'", cfg.StoragePath)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", cfg.LogLevel)
	}
	if cfg.Timezone != "UTC" {
		t.Errorf("expected Timezone 'UTC', got '%s'", cfg.Timezone)
	}
	if cfg.DataRetentionDays != 30 {
		t.Errorf("expected DataRetentionDays 30, got %d", cfg.DataRetentionDays)
	}
	if cfg.Address() != "127.0.0.1:9090" {
		t.Errorf("expected Address '127.0.0.1:9090', got '%s'", cfg.Address())
	}
}

func TestAddress_EmptyHost(t *testing.T) {
	cfg := &Config{
		Host: "",
		Port: "8090",
	}
	if cfg.Address() != ":8090" {
		t.Errorf("expected Address ':8090', got '%s'", cfg.Address())
	}
}

func TestGetEnvInt_Invalid(t *testing.T) {
	t.Setenv("DATA_RETENTION_DAYS", "not-a-number")
	cfg := Load()
	if cfg.DataRetentionDays != 14 {
		t.Errorf("expected fallback to 14 for invalid integer, got %d", cfg.DataRetentionDays)
	}
}
