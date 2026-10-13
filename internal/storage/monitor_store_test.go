package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func setupTestDBWithUsers(t *testing.T) (*DB, *UserStore, *MonitorStore, int64, int64) {
	t.Helper()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	userStore, err := NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to create user store: %v", err)
	}

	monitorStore, err := NewMonitorStore(db)
	if err != nil {
		t.Fatalf("failed to create monitor store: %v", err)
	}

	ctx := context.Background()
	user1, err := userStore.CreateUser(ctx, &model.User{
		Email:        "user1@goodai.co.nz",
		PasswordHash: "hash1",
		Name:         "User One",
	})
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	user2, err := userStore.CreateUser(ctx, &model.User{
		Email:        "user2@goodai.co.nz",
		PasswordHash: "hash2",
		Name:         "User Two",
	})
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	return db, userStore, monitorStore, user1.ID, user2.ID
}

func TestMonitorStore_CreateAndGet(t *testing.T) {
	db, _, store, user1ID, _ := setupTestDBWithUsers(t)
	defer db.Close()

	ctx := context.Background()

	m := &model.Monitor{
		UserID:             user1ID,
		Name:               "Test API",
		URL:                "https://api.goodai.co.nz",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
		Keyword:            "OK",
	}

	created, err := store.CreateMonitor(ctx, m)
	if err != nil {
		t.Fatalf("CreateMonitor failed: %v", err)
	}
	if created.ID <= 0 {
		t.Errorf("expected positive monitor ID, got %d", created.ID)
	}
	if created.Status != model.StatusPending {
		t.Errorf("expected status %s, got %s", model.StatusPending, created.Status)
	}
	if created.UptimePercent != 100.0 {
		t.Errorf("expected default uptime 100.0, got %f", created.UptimePercent)
	}

	fetched, err := store.GetMonitorByIDAndUser(ctx, created.ID, user1ID)
	if err != nil {
		t.Fatalf("GetMonitorByIDAndUser failed: %v", err)
	}
	if fetched.Name != "Test API" {
		t.Errorf("expected name 'Test API', got '%s'", fetched.Name)
	}
	if fetched.URL != "https://api.goodai.co.nz" {
		t.Errorf("expected url 'https://api.goodai.co.nz', got '%s'", fetched.URL)
	}
	if fetched.Keyword != "OK" {
		t.Errorf("expected keyword 'OK', got '%s'", fetched.Keyword)
	}
}

func TestMonitorStore_MultiTenantIsolation(t *testing.T) {
	db, _, store, user1ID, user2ID := setupTestDBWithUsers(t)
	defer db.Close()

	ctx := context.Background()

	// Create monitor for User 1
	m1, err := store.CreateMonitor(ctx, &model.Monitor{
		UserID:             user1ID,
		Name:               "User 1 Monitor",
		URL:                "https://u1.goodai.co.nz",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("CreateMonitor for user1 failed: %v", err)
	}

	// User 2 cannot access User 1's monitor
	_, err = store.GetMonitorByIDAndUser(ctx, m1.ID, user2ID)
	if !errors.Is(err, ErrMonitorNotFound) {
		t.Fatalf("expected ErrMonitorNotFound when user2 queries user1's monitor, got: %v", err)
	}

	// User 2 cannot update User 1's monitor
	_, err = store.UpdateMonitor(ctx, user2ID, &model.Monitor{
		ID:                 m1.ID,
		Name:               "Hacked Name",
		URL:                "https://hacked.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if !errors.Is(err, ErrMonitorNotFound) {
		t.Fatalf("expected ErrMonitorNotFound when user2 updates user1's monitor, got: %v", err)
	}

	// User 2 cannot toggle pause on User 1's monitor
	_, err = store.ToggleMonitorPause(ctx, m1.ID, user2ID)
	if !errors.Is(err, ErrMonitorNotFound) {
		t.Fatalf("expected ErrMonitorNotFound when user2 toggles user1's monitor, got: %v", err)
	}

	// User 2 cannot delete User 1's monitor
	err = store.DeleteMonitor(ctx, m1.ID, user2ID)
	if !errors.Is(err, ErrMonitorNotFound) {
		t.Fatalf("expected ErrMonitorNotFound when user2 deletes user1's monitor, got: %v", err)
	}

	// Verify User 1's monitor is still intact
	fetched, err := store.GetMonitorByIDAndUser(ctx, m1.ID, user1ID)
	if err != nil {
		t.Fatalf("User 1 should still own their monitor: %v", err)
	}
	if fetched.Name != "User 1 Monitor" {
		t.Errorf("Monitor was modified unexpectedly: %v", fetched.Name)
	}
}

func TestMonitorStore_ListMonitorsByUser(t *testing.T) {
	db, _, store, user1ID, user2ID := setupTestDBWithUsers(t)
	defer db.Close()

	ctx := context.Background()

	// Initial empty check
	list1, err := store.ListMonitorsByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("ListMonitorsByUser failed: %v", err)
	}
	if len(list1) != 0 {
		t.Errorf("expected 0 monitors, got %d", len(list1))
	}

	// Add 2 monitors for User 1 and 1 for User 2
	_, _ = store.CreateMonitor(ctx, &model.Monitor{UserID: user1ID, Name: "M1", URL: "https://m1.com"})
	time.Sleep(10 * time.Millisecond)
	_, _ = store.CreateMonitor(ctx, &model.Monitor{UserID: user1ID, Name: "M2", URL: "https://m2.com"})
	_, _ = store.CreateMonitor(ctx, &model.Monitor{UserID: user2ID, Name: "M3", URL: "https://m3.com"})

	user1Monitors, err := store.ListMonitorsByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("ListMonitorsByUser for user1 failed: %v", err)
	}
	if len(user1Monitors) != 2 {
		t.Fatalf("expected 2 monitors for user1, got %d", len(user1Monitors))
	}

	user2Monitors, err := store.ListMonitorsByUser(ctx, user2ID)
	if err != nil {
		t.Fatalf("ListMonitorsByUser for user2 failed: %v", err)
	}
	if len(user2Monitors) != 1 {
		t.Fatalf("expected 1 monitor for user2, got %d", len(user2Monitors))
	}
	if user2Monitors[0].Name != "M3" {
		t.Errorf("expected user2 monitor name 'M3', got '%s'", user2Monitors[0].Name)
	}
}

func TestMonitorStore_UpdateMonitor(t *testing.T) {
	db, _, store, user1ID, _ := setupTestDBWithUsers(t)
	defer db.Close()

	ctx := context.Background()

	m, err := store.CreateMonitor(ctx, &model.Monitor{
		UserID:             user1ID,
		Name:               "Original Name",
		URL:                "https://orig.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("CreateMonitor failed: %v", err)
	}

	updated, err := store.UpdateMonitor(ctx, user1ID, &model.Monitor{
		ID:                 m.ID,
		Name:               "Updated Name",
		URL:                "https://updated.com",
		IntervalSeconds:    300,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 204,
		Keyword:            "Success",
	})
	if err != nil {
		t.Fatalf("UpdateMonitor failed: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", updated.Name)
	}
	if updated.URL != "https://updated.com" {
		t.Errorf("expected url 'https://updated.com', got '%s'", updated.URL)
	}
	if updated.IntervalSeconds != 300 {
		t.Errorf("expected interval 300, got %d", updated.IntervalSeconds)
	}
	if updated.TimeoutSeconds != 10 {
		t.Errorf("expected timeout 10, got %d", updated.TimeoutSeconds)
	}
	if updated.ExpectedStatusCode != 204 {
		t.Errorf("expected status code 204, got %d", updated.ExpectedStatusCode)
	}
	if updated.Keyword != "Success" {
		t.Errorf("expected keyword 'Success', got '%s'", updated.Keyword)
	}
}

func TestMonitorStore_TogglePause(t *testing.T) {
	db, _, store, user1ID, _ := setupTestDBWithUsers(t)
	defer db.Close()

	ctx := context.Background()

	m, err := store.CreateMonitor(ctx, &model.Monitor{
		UserID: user1ID,
		Name:   "Pause Target",
		URL:    "https://pause.com",
	})
	if err != nil {
		t.Fatalf("CreateMonitor failed: %v", err)
	}
	if m.Status != model.StatusPending {
		t.Fatalf("expected initial status %s, got %s", model.StatusPending, m.Status)
	}

	// First toggle: PENDING -> PAUSED
	toggled, err := store.ToggleMonitorPause(ctx, m.ID, user1ID)
	if err != nil {
		t.Fatalf("ToggleMonitorPause to PAUSED failed: %v", err)
	}
	if toggled.Status != model.StatusPaused {
		t.Errorf("expected status %s, got %s", model.StatusPaused, toggled.Status)
	}

	// Second toggle: PAUSED -> PENDING
	toggledAgain, err := store.ToggleMonitorPause(ctx, m.ID, user1ID)
	if err != nil {
		t.Fatalf("ToggleMonitorPause to PENDING failed: %v", err)
	}
	if toggledAgain.Status != model.StatusPending {
		t.Errorf("expected status %s, got %s", model.StatusPending, toggledAgain.Status)
	}
}

func TestMonitorStore_DeleteAndCount(t *testing.T) {
	db, _, store, user1ID, _ := setupTestDBWithUsers(t)
	defer db.Close()

	ctx := context.Background()

	count, err := store.CountMonitorsByUser(ctx, user1ID)
	if err != nil {
		t.Fatalf("CountMonitorsByUser failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	m, err := store.CreateMonitor(ctx, &model.Monitor{
		UserID: user1ID,
		Name:   "To Delete",
		URL:    "https://delete.com",
	})
	if err != nil {
		t.Fatalf("CreateMonitor failed: %v", err)
	}

	count, _ = store.CountMonitorsByUser(ctx, user1ID)
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	err = store.DeleteMonitor(ctx, m.ID, user1ID)
	if err != nil {
		t.Fatalf("DeleteMonitor failed: %v", err)
	}

	count, _ = store.CountMonitorsByUser(ctx, user1ID)
	if count != 0 {
		t.Errorf("expected count 0 after deletion, got %d", count)
	}

	// Deleting again should return ErrMonitorNotFound
	err = store.DeleteMonitor(ctx, m.ID, user1ID)
	if !errors.Is(err, ErrMonitorNotFound) {
		t.Errorf("expected ErrMonitorNotFound on non-existent delete, got %v", err)
	}
}
