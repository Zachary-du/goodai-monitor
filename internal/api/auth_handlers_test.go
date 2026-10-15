package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/auth"
	"github.com/Zachary-du/goodai-monitor/internal/config"
	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestRegisterHandler_Success(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	payload := model.RegisterRequest{
		Email:    "test@example.com",
		Password: "Password123!",
		Name:     "Test User",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	RegisterHandler(store)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	userMap, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected user object in response, got %v", resp["user"])
	}
	if userMap["email"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", userMap["email"])
	}
	if _, hasHash := userMap["password_hash"]; hasHash {
		t.Error("security violation: password_hash leaked in response")
	}

	// Verify session cookie
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie in response")
	}
	if !sessionCookie.HttpOnly {
		t.Error("expected HttpOnly cookie")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite Lax, got %v", sessionCookie.SameSite)
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	payload := model.RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "Password123!",
		Name:     "Initial User",
	}
	body, _ := json.Marshal(payload)

	// First registration
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rec1 := httptest.NewRecorder()
	RegisterHandler(store)(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("initial registration failed with code %d", rec1.Code)
	}

	// Second registration with same email
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rec2 := httptest.NewRecorder()
	RegisterHandler(store)(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected HTTP 409 Conflict, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}
}

func TestRegisterHandler_ShortPassword(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	payload := model.RegisterRequest{
		Email:    "short@example.com",
		Password: "short",
		Name:     "Short Pwd User",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	RegisterHandler(store)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400 Bad Request for short password, got %d", rec.Code)
	}
}

func TestRegisterHandler_InvalidEmail(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	payload := model.RegisterRequest{
		Email:    "invalid-email-address",
		Password: "Password123!",
		Name:     "User",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	RegisterHandler(store)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400 Bad Request for invalid email, got %d", rec.Code)
	}
}

func TestLoginHandler_Success(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	// Register user
	regPayload := model.RegisterRequest{
		Email:    "loginuser@example.com",
		Password: "SuperSecurePassword123!",
		Name:     "Login User",
	}
	regBody, _ := json.Marshal(regPayload)
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regRec := httptest.NewRecorder()
	RegisterHandler(store)(regRec, regReq)

	// Attempt login
	loginPayload := model.LoginRequest{
		Email:    "loginuser@example.com",
		Password: "SuperSecurePassword123!",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	LoginHandler(store)(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d. Body: %s", loginRec.Code, loginRec.Body.String())
	}

	cookies := loginRec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie in login response")
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	// Wrong password for nonexistent user
	loginPayload := model.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "WrongPassword123!",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	LoginHandler(store)(loginRec, loginReq)

	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 Unauthorized, got %d", loginRec.Code)
	}
}

func TestLogoutHandler(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	// Register user to obtain valid cookie
	regPayload := model.RegisterRequest{
		Email:    "logoutuser@example.com",
		Password: "Password123!",
		Name:     "Logout User",
	}
	regBody, _ := json.Marshal(regPayload)
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regRec := httptest.NewRecorder()
	RegisterHandler(store)(regRec, regReq)

	cookies := regRec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("failed to get session cookie")
	}

	// Call logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRec := httptest.NewRecorder()
	LogoutHandler(store)(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", logoutRec.Code)
	}

	// Verify cleared cookie
	var clearedCookie *http.Cookie
	for _, c := range logoutRec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			clearedCookie = c
			break
		}
	}
	if clearedCookie == nil {
		t.Fatal("expected cleared cookie in response")
	}
	if clearedCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", clearedCookie.MaxAge)
	}
}

func TestMeHandler(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	// Register user
	regPayload := model.RegisterRequest{
		Email:    "meuser@example.com",
		Password: "Password123!",
		Name:     "Me User",
	}
	regBody, _ := json.Marshal(regPayload)
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	regRec := httptest.NewRecorder()
	RegisterHandler(store)(regRec, regReq)

	cookies := regRec.Result().Cookies()
	sessionCookie := cookies[0]

	// Call /api/v1/auth/me wrapped in AuthMiddleware and RequireAuth
	handler := AuthMiddleware(store)(RequireAuth(MeHandler(store)))

	// 1. With valid session
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "meuser@example.com") {
		t.Fatalf("expected response to contain meuser@example.com, got %s", rec.Body.String())
	}

	// 2. Unauthenticated
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	unauthRec := httptest.NewRecorder()
	handler.ServeHTTP(unauthRec, unauthReq)

	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 Unauthorized, got %d", unauthRec.Code)
	}
}

func TestProvidersHandler(t *testing.T) {
	// 1. Unconfigured
	cfgUnconfigured := &config.Config{}
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
	rec1 := httptest.NewRecorder()
	ProvidersHandler(cfgUnconfigured)(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	var resp1 map[string]bool
	_ = json.Unmarshal(rec1.Body.Bytes(), &resp1)
	if resp1["google"] != false {
		t.Errorf("expected google=false when unconfigured, got %v", resp1["google"])
	}

	// 2. Configured
	cfgConfigured := &config.Config{
		GoogleClientID:     "mock-client-id",
		GoogleClientSecret: "mock-secret",
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
	rec2 := httptest.NewRecorder()
	ProvidersHandler(cfgConfigured)(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
	var resp2 map[string]bool
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp2["google"] != true {
		t.Errorf("expected google=true when configured, got %v", resp2["google"])
	}
}

func TestGoogleLoginHandler_Unconfigured(t *testing.T) {
	cfg := &config.Config{} // Empty credentials
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	rec := httptest.NewRecorder()

	GoogleLoginHandler(cfg)(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected HTTP 503 Service Unavailable, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGoogleLoginHandler_Configured(t *testing.T) {
	cfg := &config.Config{
		GoogleClientID:     "client-id-12345.apps.googleusercontent.com",
		GoogleClientSecret: "secret-xyz",
		GoogleRedirectURI:  "https://goodai.co.nz/api/v1/auth/google/callback",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	rec := httptest.NewRecorder()

	GoogleLoginHandler(cfg)(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected HTTP 302 Found redirect, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, auth.GoogleAuthEndpoint) {
		t.Errorf("expected Location to point to Google auth endpoint, got: %s", location)
	}
	if !strings.Contains(location, "client-id-12345") {
		t.Errorf("expected Location to contain client ID, got: %s", location)
	}

	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.OAuthStateCookieName {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil || stateCookie.Value == "" {
		t.Fatalf("expected oauth_state cookie to be set")
	}
	if !stateCookie.HttpOnly {
		t.Errorf("expected HttpOnly state cookie")
	}
}

func TestGoogleCallbackHandler_CSRFStateMismatch(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	cfg := &config.Config{
		GoogleClientID:     "mock-client",
		GoogleClientSecret: "mock-secret",
		GoogleRedirectURI:  "https://goodai.co.nz/api/v1/auth/google/callback",
	}

	// Request with mismatched state
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=somecode&state=attack-state", nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.OAuthStateCookieName,
		Value: "legit-state",
	})
	rec := httptest.NewRecorder()

	GoogleCallbackHandler(cfg, store)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400 Bad Request on state mismatch, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify state cookie is cleared
	cookies := rec.Result().Cookies()
	var cleared *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.OAuthStateCookieName {
			cleared = c
			break
		}
	}
	if cleared == nil || cleared.MaxAge != -1 {
		t.Errorf("expected state cookie to be purged on error")
	}
}

func TestGoogleCallbackHandler_Success_NewUser(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	cfg := &config.Config{
		GoogleClientID:     "mock-client",
		GoogleClientSecret: "mock-secret",
		GoogleRedirectURI:  "https://goodai.co.nz/api/v1/auth/google/callback",
	}

	mockState := "valid-matching-state-nonce"
	mockExchange := func(ctx context.Context, clientID, clientSecret, redirectURI, code string) (*auth.GoogleTokenResponse, error) {
		if code != "valid-google-code" {
			return nil, errors.New("bad code")
		}
		return &auth.GoogleTokenResponse{
			AccessToken: "google-access-token-123",
		}, nil
	}

	mockFetchUserInfo := func(ctx context.Context, accessToken string) (*auth.GoogleUserInfo, error) {
		return &auth.GoogleUserInfo{
			Sub:           "google-sub-998877",
			Email:         "newgoogleuser@example.com",
			EmailVerified: true,
			Name:          "New Google User",
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=valid-google-code&state="+mockState, nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.OAuthStateCookieName,
		Value: mockState,
	})
	rec := httptest.NewRecorder()

	handler := GoogleCallbackHandlerWithServices(cfg, store, mockExchange, mockFetchUserInfo)
	handler(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected HTTP 302 Found redirect, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if rec.Header().Get("Location") != "/" {
		t.Errorf("expected redirect to '/', got '%s'", rec.Header().Get("Location"))
	}

	// Verify user provisioned in DB
	user, err := store.GetUserByGoogleID(context.Background(), "google-sub-998877")
	if err != nil {
		t.Fatalf("failed to find provisioned user by google id: %v", err)
	}
	if user.Email != "newgoogleuser@example.com" {
		t.Errorf("expected email 'newgoogleuser@example.com', got '%s'", user.Email)
	}

	// Verify session cookie set
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("expected valid session cookie in response")
	}
}

func TestGoogleCallbackHandler_Success_LinkExistingUser(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	ctx := context.Background()

	// Pre-create standard user with email
	existing, err := store.CreateUser(ctx, &model.User{
		Email:        "existing@goodai.co.nz",
		PasswordHash: "bcrypt-hash-123",
		Name:         "Existing Admin",
	})
	if err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}

	cfg := &config.Config{
		GoogleClientID:     "mock-client",
		GoogleClientSecret: "mock-secret",
		GoogleRedirectURI:  "https://goodai.co.nz/api/v1/auth/google/callback",
	}

	mockState := "valid-state-linking"
	mockExchange := func(ctx context.Context, clientID, clientSecret, redirectURI, code string) (*auth.GoogleTokenResponse, error) {
		return &auth.GoogleTokenResponse{
			AccessToken: "google-access-token-456",
		}, nil
	}

	mockFetchUserInfo := func(ctx context.Context, accessToken string) (*auth.GoogleUserInfo, error) {
		return &auth.GoogleUserInfo{
			Sub:           "google-sub-linked-555",
			Email:         "existing@goodai.co.nz",
			EmailVerified: true,
			Name:          "Existing Admin",
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=valid-code&state="+mockState, nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.OAuthStateCookieName,
		Value: mockState,
	})
	rec := httptest.NewRecorder()

	handler := GoogleCallbackHandlerWithServices(cfg, store, mockExchange, mockFetchUserInfo)
	handler(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected HTTP 302 Found, got %d", rec.Code)
	}

	// Verify account linked
	linkedUser, err := store.GetUserByGoogleID(ctx, "google-sub-linked-555")
	if err != nil {
		t.Fatalf("failed to find user by linked google id: %v", err)
	}
	if linkedUser.ID != existing.ID {
		t.Errorf("expected linked user ID %d, got %d", existing.ID, linkedUser.ID)
	}
}

