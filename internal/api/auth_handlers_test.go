package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/auth"
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
