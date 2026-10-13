package alert

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestSMTPEmailSender_IsConfigured(t *testing.T) {
	// Empty config
	s1 := NewSMTPEmailSender(SMTPConfig{})
	if s1.IsConfigured() {
		t.Errorf("expected IsConfigured to be false for empty config")
	}

	// Host without from
	s2 := NewSMTPEmailSender(SMTPConfig{Host: "smtp.example.com"})
	if s2.IsConfigured() {
		t.Errorf("expected IsConfigured to be false when From is empty")
	}

	// Valid config
	s3 := NewSMTPEmailSender(SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "alerts@goodai.co.nz",
	})
	if !s3.IsConfigured() {
		t.Errorf("expected IsConfigured to be true for valid config")
	}
}

func TestSMTPEmailSender_UnconfiguredGraceful(t *testing.T) {
	sender := NewSMTPEmailSender(SMTPConfig{})
	ctx := context.Background()

	m := &model.Monitor{
		ID:   1,
		Name: "Test Target",
		URL:  "https://test.example.com",
	}
	inc := &model.Incident{
		ID:        100,
		StartedAt: time.Now().UTC(),
		Cause:     "Connection refused",
	}

	// Should not panic or return error
	if err := sender.SendOutageAlert(ctx, "admin@example.com", m, inc); err != nil {
		t.Errorf("expected nil error on unconfigured sender, got %v", err)
	}

	if err := sender.SendRecoveryAlert(ctx, "admin@example.com", m, inc); err != nil {
		t.Errorf("expected nil error on unconfigured sender, got %v", err)
	}
}

func TestEmail_Masking(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"admin@goodai.co.nz", "ad***@goodai.co.nz"},
		{"a@example.com", "a***@example.com"},
		{"ab@example.com", "a***@example.com"},
		{"invalid-email", "***"},
	}

	for _, tt := range tests {
		got := maskEmail(tt.input)
		if got != tt.expected {
			t.Errorf("maskEmail(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestEmail_FormatDualTime(t *testing.T) {
	sampleTime := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	formatted := formatDualTime(sampleTime)

	// Must contain both UTC and NZDT/NZST
	if !strings.Contains(formatted, "UTC") {
		t.Errorf("expected dual time to contain 'UTC', got: %s", formatted)
	}
	if !strings.Contains(formatted, "NZ") {
		t.Errorf("expected dual time to contain New Zealand zone, got: %s", formatted)
	}
}

func TestEmail_FormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int64
		expected string
	}{
		{0, "0s"},
		{-5, "0s"},
		{45, "45s"},
		{125, "2m 5s"},
		{3665, "1h 1m 5s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.seconds)
		if got != tt.expected {
			t.Errorf("formatDuration(%d) = %q, expected %q", tt.seconds, got, tt.expected)
		}
	}
}
