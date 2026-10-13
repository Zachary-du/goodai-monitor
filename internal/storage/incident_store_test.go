package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func setupTestIncidentStore(t *testing.T) (*DB, *UserStore, *MonitorStore, *IncidentStore) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_incident_store.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}

	userStore, err := NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to initialize user store: %v", err)
	}

	monitorStore, err := NewMonitorStore(db)
	if err != nil {
		t.Fatalf("failed to initialize monitor store: %v", err)
	}

	incidentStore, err := NewIncidentStore(db)
	if err != nil {
		t.Fatalf("failed to initialize incident store: %v", err)
	}

	return db, userStore, monitorStore, incidentStore
}

func TestIncidentStore_Lifecycle(t *testing.T) {
	db, userStore, monitorStore, incidentStore := setupTestIncidentStore(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Create test user and monitor
	u, err := userStore.CreateUser(ctx, &model.User{
		Email:        "alice@goodai.co.nz",
		PasswordHash: "secret123",
		Name:         "Alice",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	m, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             u.ID,
		Name:               "Alice API",
		URL:                "https://api.alice.example.com/healthz",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	// 2. Initial state: no open incidents
	_, err = incidentStore.GetOpenIncidentByMonitor(ctx, m.ID)
	if err != ErrIncidentNotFound {
		t.Fatalf("expected ErrIncidentNotFound, got %v", err)
	}

	activeCount, err := incidentStore.CountActiveIncidentsByUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("failed to count active incidents: %v", err)
	}
	if activeCount != 0 {
		t.Fatalf("expected 0 active incidents, got %d", activeCount)
	}

	// 3. Create OPEN incident
	startTime := time.Now().UTC().Add(-5 * time.Minute)
	inc, err := incidentStore.CreateIncident(ctx, &model.Incident{
		MonitorID: m.ID,
		UserID:    u.ID,
		StartedAt: startTime,
		Status:    model.IncidentStatusOpen,
		Cause:     "HTTP 502 Bad Gateway: connection refused",
	})
	if err != nil {
		t.Fatalf("failed to create incident: %v", err)
	}
	if inc.ID <= 0 {
		t.Fatalf("expected valid incident ID, got %d", inc.ID)
	}

	// 4. Verify GetOpenIncidentByMonitor
	openInc, err := incidentStore.GetOpenIncidentByMonitor(ctx, m.ID)
	if err != nil {
		t.Fatalf("failed to get open incident: %v", err)
	}
	if openInc.ID != inc.ID {
		t.Errorf("expected incident ID %d, got %d", inc.ID, openInc.ID)
	}
	if openInc.Status != model.IncidentStatusOpen {
		t.Errorf("expected status OPEN, got %s", openInc.Status)
	}
	if openInc.MonitorName != "Alice API" {
		t.Errorf("expected joined MonitorName 'Alice API', got '%s'", openInc.MonitorName)
	}
	if openInc.Cause != "HTTP 502 Bad Gateway: connection refused" {
		t.Errorf("unexpected cause: %s", openInc.Cause)
	}

	// Active count should be 1
	activeCount, err = incidentStore.CountActiveIncidentsByUser(ctx, u.ID)
	if err != nil || activeCount != 1 {
		t.Fatalf("expected 1 active incident, got %d (err: %v)", activeCount, err)
	}

	// 5. Resolve incident
	resolveTime := time.Now().UTC()
	durationSec := int64(300) // 5 minutes
	err = incidentStore.ResolveIncident(ctx, inc.ID, resolveTime, durationSec)
	if err != nil {
		t.Fatalf("failed to resolve incident: %v", err)
	}

	// 6. Verify resolved state
	_, err = incidentStore.GetOpenIncidentByMonitor(ctx, m.ID)
	if err != ErrIncidentNotFound {
		t.Fatalf("expected ErrIncidentNotFound after resolution, got %v", err)
	}

	resolvedInc, err := incidentStore.GetIncidentByID(ctx, inc.ID)
	if err != nil {
		t.Fatalf("failed to get incident by id: %v", err)
	}
	if resolvedInc.Status != model.IncidentStatusResolved {
		t.Errorf("expected status RESOLVED, got %s", resolvedInc.Status)
	}
	if resolvedInc.ResolvedAt == nil {
		t.Fatal("expected non-nil ResolvedAt timestamp")
	}
	if resolvedInc.DurationSeconds != 300 {
		t.Errorf("expected duration 300 seconds, got %d", resolvedInc.DurationSeconds)
	}

	// 7. Resolving again should return ErrIncidentNotFound because status is no longer OPEN
	err = incidentStore.ResolveIncident(ctx, inc.ID, resolveTime, durationSec)
	if err != ErrIncidentNotFound {
		t.Errorf("expected ErrIncidentNotFound when re-resolving, got %v", err)
	}
}

func TestIncidentStore_MultiTenantIsolationAndFiltering(t *testing.T) {
	db, userStore, monitorStore, incidentStore := setupTestIncidentStore(t)
	defer db.Close()
	ctx := context.Background()

	// Create User A and User B
	userA, _ := userStore.CreateUser(ctx, &model.User{Email: "userA@goodai.co.nz", PasswordHash: "h1", Name: "User A"})
	userB, _ := userStore.CreateUser(ctx, &model.User{Email: "userB@goodai.co.nz", PasswordHash: "h2", Name: "User B"})

	monA, _ := monitorStore.CreateMonitor(ctx, &model.Monitor{UserID: userA.ID, Name: "Mon A", URL: "https://a.test", IntervalSeconds: 60, TimeoutSeconds: 5, ExpectedStatusCode: 200})
	monB, _ := monitorStore.CreateMonitor(ctx, &model.Monitor{UserID: userB.ID, Name: "Mon B", URL: "https://b.test", IntervalSeconds: 60, TimeoutSeconds: 5, ExpectedStatusCode: 200})

	// Create 2 incidents for User A (1 open, 1 resolved)
	incA1, _ := incidentStore.CreateIncident(ctx, &model.Incident{
		MonitorID: monA.ID,
		UserID:    userA.ID,
		StartedAt: time.Now().UTC().Add(-10 * time.Minute),
		Status:    model.IncidentStatusOpen,
		Cause:     "Timeout",
	})
	_ = incidentStore.ResolveIncident(ctx, incA1.ID, time.Now().UTC().Add(-5*time.Minute), 300)

	_, _ = incidentStore.CreateIncident(ctx, &model.Incident{
		MonitorID: monA.ID,
		UserID:    userA.ID,
		StartedAt: time.Now().UTC().Add(-2 * time.Minute),
		Status:    model.IncidentStatusOpen,
		Cause:     "HTTP 500",
	})

	// Create 1 incident for User B (open)
	_, _ = incidentStore.CreateIncident(ctx, &model.Incident{
		MonitorID: monB.ID,
		UserID:    userB.ID,
		StartedAt: time.Now().UTC().Add(-1 * time.Minute),
		Status:    model.IncidentStatusOpen,
		Cause:     "Connection reset",
	})

	// Test User A queries
	listA, err := incidentStore.ListIncidentsByUser(ctx, userA.ID, "all", 50, 0)
	if err != nil {
		t.Fatalf("failed to list User A incidents: %v", err)
	}
	if len(listA) != 2 {
		t.Fatalf("expected 2 incidents for User A, got %d", len(listA))
	}

	// Filter by open for User A
	openA, err := incidentStore.ListIncidentsByUser(ctx, userA.ID, "OPEN", 50, 0)
	if err != nil || len(openA) != 1 {
		t.Fatalf("expected 1 open incident for User A, got %d (err: %v)", len(openA), err)
	}
	if openA[0].Cause != "HTTP 500" {
		t.Errorf("expected open incident cause 'HTTP 500', got '%s'", openA[0].Cause)
	}

	// Filter by resolved for User A
	resolvedA, err := incidentStore.ListIncidentsByUser(ctx, userA.ID, "RESOLVED", 50, 0)
	if err != nil || len(resolvedA) != 1 {
		t.Fatalf("expected 1 resolved incident for User A, got %d (err: %v)", len(resolvedA), err)
	}

	// User A summary
	summaryA, err := incidentStore.GetIncidentSummaryByUser(ctx, userA.ID)
	if err != nil {
		t.Fatalf("failed to get summary for User A: %v", err)
	}
	if summaryA.OpenCount != 1 || summaryA.ResolvedCount != 1 || summaryA.TotalCount != 2 {
		t.Errorf("unexpected summary for User A: %+v", summaryA)
	}

	// Multi-tenant check: User B should only see their 1 incident
	listB, err := incidentStore.ListIncidentsByUser(ctx, userB.ID, "all", 50, 0)
	if err != nil {
		t.Fatalf("failed to list User B incidents: %v", err)
	}
	if len(listB) != 1 {
		t.Fatalf("expected 1 incident for User B, got %d", len(listB))
	}
	if listB[0].MonitorName != "Mon B" {
		t.Errorf("expected User B incident to reference 'Mon B', got '%s'", listB[0].MonitorName)
	}

	summaryB, err := incidentStore.GetIncidentSummaryByUser(ctx, userB.ID)
	if err != nil || summaryB.TotalCount != 1 || summaryB.OpenCount != 1 {
		t.Fatalf("unexpected summary for User B: %+v (err: %v)", summaryB, err)
	}
}
