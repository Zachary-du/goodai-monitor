package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuth_GenerateOAuthState(t *testing.T) {
	state1, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("unexpected error generating state: %v", err)
	}
	if len(state1) != 64 {
		t.Fatalf("expected 64 hex characters (32 bytes), got length %d: %s", len(state1), state1)
	}

	state2, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("unexpected error generating second state: %v", err)
	}
	if state1 == state2 {
		t.Fatalf("expected distinct random state tokens, got collision: %s", state1)
	}
}

func TestOAuth_StateCookieLifecycle(t *testing.T) {
	state := "test-state-nonce-1234567890abcdef"
	cookie := NewOAuthStateCookie(state, true)

	if cookie.Name != OAuthStateCookieName {
		t.Errorf("expected cookie name '%s', got '%s'", OAuthStateCookieName, cookie.Name)
	}
	if cookie.Value != state {
		t.Errorf("expected cookie value '%s', got '%s'", state, cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Errorf("expected HttpOnly cookie")
	}
	if !cookie.Secure {
		t.Errorf("expected Secure cookie")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax")
	}
	if cookie.MaxAge <= 0 {
		t.Errorf("expected positive MaxAge, got %d", cookie.MaxAge)
	}

	clearCookie := ClearOAuthStateCookie()
	if clearCookie.Value != "" {
		t.Errorf("expected empty value in clear cookie, got '%s'", clearCookie.Value)
	}
	if clearCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1 in clear cookie, got %d", clearCookie.MaxAge)
	}
}

func TestOAuth_BuildGoogleAuthURL(t *testing.T) {
	clientID := "test-client-id.apps.googleusercontent.com"
	redirectURI := "https://goodai.co.nz/api/v1/auth/google/callback"
	state := "random-state-123"

	authURL := BuildGoogleAuthURL(clientID, redirectURI, state)

	if !strings.HasPrefix(authURL, GoogleAuthEndpoint+"?") {
		t.Fatalf("expected URL to start with GoogleAuthEndpoint, got %s", authURL)
	}

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	q := parsed.Query()
	if q.Get("client_id") != clientID {
		t.Errorf("expected client_id %s, got %s", clientID, q.Get("client_id"))
	}
	if q.Get("redirect_uri") != redirectURI {
		t.Errorf("expected redirect_uri %s, got %s", redirectURI, q.Get("redirect_uri"))
	}
	if q.Get("state") != state {
		t.Errorf("expected state %s, got %s", state, q.Get("state"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("expected response_type=code, got %s", q.Get("response_type"))
	}
	if !strings.Contains(q.Get("scope"), "openid") || !strings.Contains(q.Get("scope"), "email") {
		t.Errorf("expected scope to contain openid and email, got %s", q.Get("scope"))
	}
	if q.Get("prompt") != "select_account" {
		t.Errorf("expected prompt=select_account, got %s", q.Get("prompt"))
	}
}

func TestOAuth_ExchangeGoogleCode(t *testing.T) {
	// Mock Google Token Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		code := r.Form.Get("code")
		clientID := r.Form.Get("client_id")
		clientSecret := r.Form.Get("client_secret")
		grantType := r.Form.Get("grant_type")

		if clientID != "mock-client" || clientSecret != "mock-secret" || grantType != "authorization_code" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}

		if code == "valid-auth-code" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(GoogleTokenResponse{
				AccessToken: "mock-access-token-999",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
				Scope:       "openid email profile",
				IDToken:     "mock-id-token",
			})
			return
		}

		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	ctx := context.Background()

	// 1. Success case
	resp, err := ExchangeGoogleCodeWithClient(ctx, server.Client(), server.URL, "mock-client", "mock-secret", "https://goodai.co.nz/callback", "valid-auth-code")
	if err != nil {
		t.Fatalf("expected successful exchange, got error: %v", err)
	}
	if resp.AccessToken != "mock-access-token-999" {
		t.Errorf("expected access token 'mock-access-token-999', got '%s'", resp.AccessToken)
	}

	// 2. Invalid code case
	_, err = ExchangeGoogleCodeWithClient(ctx, server.Client(), server.URL, "mock-client", "mock-secret", "https://goodai.co.nz/callback", "bad-code")
	if err == nil {
		t.Fatalf("expected error for invalid code, got nil")
	}

	// 3. Invalid credentials case
	_, err = ExchangeGoogleCodeWithClient(ctx, server.Client(), server.URL, "wrong-client", "wrong-secret", "https://goodai.co.nz/callback", "valid-auth-code")
	if err == nil {
		t.Fatalf("expected error for wrong credentials, got nil")
	}
}

func TestOAuth_FetchGoogleUserInfo(t *testing.T) {
	// Mock Google UserInfo Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer valid-token" && authHeader != "Bearer unverified-token" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if authHeader == "Bearer unverified-token" {
			_ = json.NewEncoder(w).Encode(GoogleUserInfo{
				Sub:           "100200300",
				Email:         "unverified@example.com",
				EmailVerified: false,
				Name:          "Unverified User",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(GoogleUserInfo{
			Sub:           "999888777",
			Email:         "alex.morgan@example.com",
			EmailVerified: true,
			Name:          "Alex Morgan",
			Picture:       "https://lh3.googleusercontent.com/photo.jpg",
		})
	}))
	defer server.Close()

	ctx := context.Background()

	// 1. Success case
	info, err := FetchGoogleUserInfoWithClient(ctx, server.Client(), server.URL, "valid-token")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if info.Sub != "999888777" || info.Email != "alex.morgan@example.com" || !info.EmailVerified {
		t.Errorf("unexpected user info: %+v", info)
	}

	// 2. Unverified email case
	_, err = FetchGoogleUserInfoWithClient(ctx, server.Client(), server.URL, "unverified-token")
	if err != ErrOAuthEmailNotVerified {
		t.Errorf("expected ErrOAuthEmailNotVerified, got: %v", err)
	}

	// 3. Unauthorized token case
	_, err = FetchGoogleUserInfoWithClient(ctx, server.Client(), server.URL, "invalid-token")
	if err == nil {
		t.Errorf("expected error for unauthorized token, got nil")
	}
}
