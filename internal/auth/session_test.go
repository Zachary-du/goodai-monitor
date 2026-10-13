package auth

import (
	"encoding/hex"
	"net/http"
	"testing"
	"time"
)

func TestGenerateSecureToken(t *testing.T) {
	token1, err := GenerateSecureToken(0)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	token2, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("unexpected error generating token 2: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected distinct cryptographic random tokens")
	}

	if len(token1) != 64 { // 32 bytes * 2 hex chars
		t.Fatalf("expected 64 hex characters, got %d", len(token1))
	}

	if _, err := hex.DecodeString(token1); err != nil {
		t.Fatalf("expected valid hex string: %v", err)
	}

	customToken, err := GenerateSecureToken(16)
	if err != nil {
		t.Fatalf("unexpected error generating 16-byte token: %v", err)
	}
	if len(customToken) != 32 {
		t.Fatalf("expected 32 hex characters for 16 bytes, got %d", len(customToken))
	}
}

func TestNewSessionCookie(t *testing.T) {
	token := "dummy-session-token-12345"
	expiresAt := time.Now().Add(14 * 24 * time.Hour)

	cookie := NewSessionCookie(token, expiresAt, false)

	if cookie.Name != SessionCookieName {
		t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
	}
	if cookie.Value != token {
		t.Errorf("expected cookie value %s, got %s", token, cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly flag to be true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite Lax, got %v", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Errorf("expected Path '/', got %s", cookie.Path)
	}
	if cookie.MaxAge <= 0 {
		t.Errorf("expected positive MaxAge, got %d", cookie.MaxAge)
	}
}

func TestClearSessionCookie(t *testing.T) {
	cookie := ClearSessionCookie()

	if cookie.Name != SessionCookieName {
		t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
	}
	if cookie.Value != "" {
		t.Errorf("expected empty cookie value, got %s", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", cookie.MaxAge)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly flag to be true")
	}
}
