package storage

import (
	"context"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func setupTestDB(t *testing.T) (*DB, *UserStore) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	userStore, err := NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to create user store: %v", err)
	}

	return db, userStore
}

func TestUserStore_CreateAndGetUser(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	candidate := &model.User{
		Email:        "Alice@Example.com",
		PasswordHash: "$2a$10$dummyhashedpasswordalice1234567890",
		Name:         "Alice Smith",
	}

	created, err := store.CreateUser(ctx, candidate)
	if err != nil {
		t.Fatalf("unexpected error creating user: %v", err)
	}

	if created.ID <= 0 {
		t.Errorf("expected positive ID, got %d", created.ID)
	}
	if created.Email != "alice@example.com" { // Lowercased
		t.Errorf("expected normalized email alice@example.com, got %s", created.Email)
	}
	if created.Name != "Alice Smith" {
		t.Errorf("expected name Alice Smith, got %s", created.Name)
	}

	// Fetch by email
	fetchedByEmail, err := store.GetUserByEmail(ctx, "ALICE@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	if fetchedByEmail.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, fetchedByEmail.ID)
	}

	// Fetch by ID
	fetchedByID, err := store.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get user by ID: %v", err)
	}
	if fetchedByID.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", fetchedByID.Email)
	}
}

func TestUserStore_DuplicateEmailRejection(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	u1 := &model.User{
		Email:        "bob@example.com",
		PasswordHash: "$2a$10$dummyhashedpasswordbob1234567890",
		Name:         "Bob",
	}

	if _, err := store.CreateUser(ctx, u1); err != nil {
		t.Fatalf("failed to create initial user: %v", err)
	}

	// Try registering with same email (different case)
	u2 := &model.User{
		Email:        "BOB@EXAMPLE.COM",
		PasswordHash: "$2a$10$anotherhashedpasswordbob1234567890",
		Name:         "Bob Duplicate",
	}

	_, err := store.CreateUser(ctx, u2)
	if err != ErrDuplicateEmail {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestUserStore_UserNotFound(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := store.GetUserByEmail(ctx, "nonexistent@example.com")
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}

	_, err = store.GetUserByID(ctx, 999999)
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserStore_SessionLifecycle(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	u, err := store.CreateUser(ctx, &model.User{
		Email:        "carol@example.com",
		PasswordHash: "$2a$10$dummyhashedpasswordcarol1234567890",
		Name:         "Carol",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token := "secure-test-session-token-carol-12345"
	session := &model.Session{
		Token:     token,
		UserID:    u.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(14 * 24 * time.Hour),
	}

	if err := store.CreateSession(ctx, session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	fetched, err := store.GetSessionByToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}
	if fetched.UserID != u.ID {
		t.Errorf("expected user ID %d, got %d", u.ID, fetched.UserID)
	}

	// Delete session (logout)
	if err := store.DeleteSession(ctx, token); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	_, err = store.GetSessionByToken(ctx, token)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after deletion, got %v", err)
	}
}

func TestUserStore_ExpiredSession(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	u, err := store.CreateUser(ctx, &model.User{
		Email:        "david@example.com",
		PasswordHash: "$2a$10$dummyhashedpassworddavid1234567890",
		Name:         "David",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	expiredToken := "expired-token-david-999"
	expiredSession := &model.Session{
		Token:     expiredToken,
		UserID:    u.ID,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
	}

	if err := store.CreateSession(ctx, expiredSession); err != nil {
		t.Fatalf("failed to create expired session: %v", err)
	}

	_, err = store.GetSessionByToken(ctx, expiredToken)
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}

	// Test DeleteExpiredSessions
	deletedCount, err := store.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("failed to delete expired sessions: %v", err)
	}
	if deletedCount != 1 {
		t.Errorf("expected 1 expired session deleted, got %d", deletedCount)
	}
}

func TestUserStore_GoogleOAuth(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// 1. Create a Google user
	gUser, err := store.CreateGoogleUser(ctx, "googleuser@example.com", "Google User", "google-sub-123456")
	if err != nil {
		t.Fatalf("failed to create google user: %v", err)
	}
	if gUser.GoogleID == nil || *gUser.GoogleID != "google-sub-123456" {
		t.Fatalf("expected google_id 'google-sub-123456'")
	}

	// 2. Query by Google ID
	fetched, err := store.GetUserByGoogleID(ctx, "google-sub-123456")
	if err != nil {
		t.Fatalf("failed to get user by google id: %v", err)
	}
	if fetched.ID != gUser.ID || fetched.Email != "googleuser@example.com" {
		t.Errorf("fetched user mismatch: %+v", fetched)
	}

	// 3. Duplicate Google user should fail
	_, err = store.CreateGoogleUser(ctx, "other@example.com", "Other", "google-sub-123456")
	if err == nil {
		t.Errorf("expected duplicate google id to fail")
	}

	// 4. Link existing standard password user with Google ID
	standardUser, err := store.CreateUser(ctx, &model.User{
		Email:        "existing@example.com",
		PasswordHash: "somehash",
		Name:         "Existing",
	})
	if err != nil {
		t.Fatalf("failed to create standard user: %v", err)
	}

	if err := store.LinkGoogleAccount(ctx, standardUser.ID, "google-sub-7890"); err != nil {
		t.Fatalf("failed to link google account: %v", err)
	}

	linked, err := store.GetUserByGoogleID(ctx, "google-sub-7890")
	if err != nil {
		t.Fatalf("failed to fetch linked user: %v", err)
	}
	if linked.ID != standardUser.ID {
		t.Errorf("expected linked ID %d, got %d", standardUser.ID, linked.ID)
	}
}
