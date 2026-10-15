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
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")

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
	if cfg.SMTPPort != 587 {
		t.Errorf("expected default SMTPPort 587, got %d", cfg.SMTPPort)
	}
	if cfg.IsSMTPConfigured() {
		t.Errorf("expected IsSMTPConfigured to be false with empty env")
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("STORAGE_PATH", "/tmp/custom.db")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("TZ", "UTC")
	t.Setenv("DATA_RETENTION_DAYS", "30")
	t.Setenv("SMTP_HOST", "mail.example.com")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_USERNAME", "mailer")
	t.Setenv("SMTP_PASSWORD", "secret")
	t.Setenv("SMTP_FROM", "alerts@example.com")

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
	if cfg.SMTPHost != "mail.example.com" {
		t.Errorf("expected SMTPHost 'mail.example.com', got '%s'", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 465 {
		t.Errorf("expected SMTPPort 465, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPUsername != "mailer" {
		t.Errorf("expected SMTPUsername 'mailer', got '%s'", cfg.SMTPUsername)
	}
	if cfg.SMTPPassword != "secret" {
		t.Errorf("expected SMTPPassword 'secret', got '%s'", cfg.SMTPPassword)
	}
	if cfg.SMTPFrom != "alerts@example.com" {
		t.Errorf("expected SMTPFrom 'alerts@example.com', got '%s'", cfg.SMTPFrom)
	}
	if !cfg.IsSMTPConfigured() {
		t.Errorf("expected IsSMTPConfigured to be true")
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

func TestGoogleOAuthConfig(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("GOOGLE_REDIRECT_URI", "")

	cfg := Load()
	if cfg.IsGoogleOAuthConfigured() {
		t.Errorf("expected IsGoogleOAuthConfigured to be false with empty env")
	}
	if cfg.GoogleRedirectURI != "https://goodai.co.nz/api/v1/auth/google/callback" {
		t.Errorf("expected default redirect URI, got '%s'", cfg.GoogleRedirectURI)
	}

	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")
	t.Setenv("GOOGLE_REDIRECT_URI", "http://localhost:8090/callback")

	cfg2 := Load()
	if !cfg2.IsGoogleOAuthConfigured() {
		t.Errorf("expected IsGoogleOAuthConfigured to be true with credentials set")
	}
	if cfg2.GoogleClientID != "test-client-id" {
		t.Errorf("expected client id 'test-client-id', got '%s'", cfg2.GoogleClientID)
	}
	if cfg2.GoogleClientSecret != "test-client-secret" {
		t.Errorf("expected client secret 'test-client-secret', got '%s'", cfg2.GoogleClientSecret)
	}
	if cfg2.GoogleRedirectURI != "http://localhost:8090/callback" {
		t.Errorf("expected custom redirect uri, got '%s'", cfg2.GoogleRedirectURI)
	}
}
