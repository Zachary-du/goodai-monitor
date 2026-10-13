package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/auth"
	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

func setupTestStorage(t *testing.T) (*storage.DB, *storage.UserStore) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	store, err := storage.NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to create user store: %v", err)
	}

	return db, store
}

func TestAuthMiddleware_ValidSession(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	ctx := context.Background()
	user, err := store.CreateUser(ctx, &model.User{
		Email:        "valid@example.com",
		PasswordHash: "$2a$10$validhashedpasswordexample12345",
		Name:         "Valid User",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	token := "valid-test-token-1234567890"
	err = store.CreateSession(ctx, &model.Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok || userID != user.ID {
			t.Fatalf("expected userID %d in context, got %d (ok=%v)", user.ID, userID, ok)
		}
		u, ok := UserFromContext(r.Context())
		if !ok || u.Email != "valid@example.com" {
			t.Fatalf("expected user model in context, got %+v (ok=%v)", u, ok)
		}
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	stack := AuthMiddleware(store)(RequireAuth(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: token,
	})
	rec := httptest.NewRecorder()

	stack.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected HTTP 200, got %d", rec.Code)
	}
	if !handlerCalled {
		t.Error("expected inner handler to be called")
	}
}

func TestRequireAuth_MissingSession(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	stack := AuthMiddleware(store)(RequireAuth(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	stack.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected HTTP 401 Unauthorized, got %d", rec.Code)
	}
}

func TestRequireAuth_ForgedSession(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	stack := AuthMiddleware(store)(RequireAuth(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: "forged-invalid-session-token",
	})
	rec := httptest.NewRecorder()

	stack.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected HTTP 401 Unauthorized, got %d", rec.Code)
	}
}

func TestRequireAuth_ExpiredSession(t *testing.T) {
	db, store := setupTestStorage(t)
	defer db.Close()

	ctx := context.Background()
	user, err := store.CreateUser(ctx, &model.User{
		Email:        "expired@example.com",
		PasswordHash: "$2a$10$expiredhashedpassword12345678",
		Name:         "Expired User",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	token := "expired-test-token-9876543210"
	err = store.CreateSession(ctx, &model.Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: time.Now().UTC().Add(-48 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	stack := AuthMiddleware(store)(RequireAuth(testHandler))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: token,
	})
	rec := httptest.NewRecorder()

	stack.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected HTTP 401 Unauthorized for expired session, got %d", rec.Code)
	}
}
