package alert

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// mockEmailSender records sent outage and recovery emails in-memory.
type mockEmailSender struct {
	mu             sync.Mutex
	outageAlerts   []string
	recoveryAlerts []string
	shouldFail     bool
}

func (m *mockEmailSender) SendOutageAlert(ctx context.Context, recipient string, mon *model.Monitor, inc *model.Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outageAlerts = append(m.outageAlerts, recipient)
	if m.shouldFail {
		return context.DeadlineExceeded
	}
	return nil
}

func (m *mockEmailSender) SendRecoveryAlert(ctx context.Context, recipient string, mon *model.Monitor, inc *model.Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recoveryAlerts = append(m.recoveryAlerts, recipient)
	if m.shouldFail {
		return context.DeadlineExceeded
	}
	return nil
}

func setupTestAlertManager(t *testing.T) (*storage.DB, *storage.UserStore, *storage.MonitorStore, *storage.IncidentStore, *mockEmailSender, *AlertManager) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_alert_manager.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	userStore, err := storage.NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to initialize user store: %v", err)
	}

	monitorStore, err := storage.NewMonitorStore(db)
	if err != nil {
		t.Fatalf("failed to initialize monitor store: %v", err)
	}

	incidentStore, err := storage.NewIncidentStore(db)
	if err != nil {
		t.Fatalf("failed to initialize incident store: %v", err)
	}

	mockSender := &mockEmailSender{}
	mgr := NewAlertManager(incidentStore, userStore, monitorStore, mockSender)

	return db, userStore, monitorStore, incidentStore, mockSender, mgr
}

func TestAlertManager_AntiFlappingAndAlertingLifecycle(t *testing.T) {
	db, userStore, monitorStore, incidentStore, mockSender, mgr := setupTestAlertManager(t)
	defer db.Close()
	ctx := context.Background()

	// Create user & monitor
	u, err := userStore.CreateUser(ctx, &model.User{
		Email:        "ops@goodai.co.nz",
		PasswordHash: "pwd123",
		Name:         "Ops Lead",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	mon, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             u.ID,
		Name:               "GoodAI Web Portal",
		URL:                "https://goodai.co.nz",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	baseTime := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)

	// --- STEP 1: Single failure (N=1) -> Degraded, NO incident, NO email (AC-1) ---
	check1 := &model.Check{
		MonitorID:    mon.ID,
		Status:       model.StatusDown,
		ErrorMessage: "504 Gateway Timeout",
		CreatedAt:    baseTime,
	}
	mgr.ProcessCheckResult(ctx, mon, check1)

	state, ok := mgr.GetMonitorState(mon.ID)
	if !ok {
		t.Fatal("expected state for monitor")
	}
	if state.ConsecutiveFailures != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", state.ConsecutiveFailures)
	}
	if state.ActiveIncidentID != 0 {
		t.Errorf("expected NO active incident on 1st failure, got ID %d", state.ActiveIncidentID)
	}
	if len(mockSender.outageAlerts) != 0 {
		t.Errorf("expected 0 outage emails on 1st failure, got %d", len(mockSender.outageAlerts))
	}

	// Verify no incident in DB
	activeCount, err := incidentStore.CountActiveIncidentsByUser(ctx, u.ID)
	if err != nil || activeCount != 0 {
		t.Fatalf("expected 0 active incidents in DB, got %d", activeCount)
	}

	// --- STEP 2: Second consecutive failure (N=2) -> Confirmed DOWN, creates OPEN incident, dispatches email (AC-2, AC-3) ---
	check2 := &model.Check{
		MonitorID:    mon.ID,
		Status:       model.StatusDown,
		ErrorMessage: "504 Gateway Timeout",
		CreatedAt:    baseTime.Add(1 * time.Minute),
	}
	mgr.ProcessCheckResult(ctx, mon, check2)

	state, _ = mgr.GetMonitorState(mon.ID)
	if state.ConsecutiveFailures != 2 {
		t.Errorf("expected 2 consecutive failures, got %d", state.ConsecutiveFailures)
	}
	if state.ActiveIncidentID == 0 {
		t.Fatal("expected active incident ID to be set after 2nd failure")
	}
	if !state.OutageAlertDispatched {
		t.Error("expected OutageAlertDispatched to be true")
	}

	// Verify email dispatched to user
	if len(mockSender.outageAlerts) != 1 {
		t.Fatalf("expected 1 outage email dispatched, got %d", len(mockSender.outageAlerts))
	}
	if mockSender.outageAlerts[0] != "ops@goodai.co.nz" {
		t.Errorf("expected email to 'ops@goodai.co.nz', got '%s'", mockSender.outageAlerts[0])
	}

	// Verify open incident in DB
	openInc, err := incidentStore.GetOpenIncidentByMonitor(ctx, mon.ID)
	if err != nil {
		t.Fatalf("failed to query open incident from DB: %v", err)
	}
	if openInc.Status != model.IncidentStatusOpen {
		t.Errorf("expected incident status OPEN, got %s", openInc.Status)
	}
	if openInc.Cause != "504 Gateway Timeout" {
		t.Errorf("expected cause '504 Gateway Timeout', got '%s'", openInc.Cause)
	}

	// --- STEP 3: Continued failures (N=3, N=4) -> Ongoing outage, deduplicated (AC-5) ---
	check3 := &model.Check{MonitorID: mon.ID, Status: model.StatusDown, ErrorMessage: "504 Gateway Timeout", CreatedAt: baseTime.Add(2 * time.Minute)}
	check4 := &model.Check{MonitorID: mon.ID, Status: model.StatusDown, ErrorMessage: "504 Gateway Timeout", CreatedAt: baseTime.Add(3 * time.Minute)}
	mgr.ProcessCheckResult(ctx, mon, check3)
	mgr.ProcessCheckResult(ctx, mon, check4)

	// Outage email count must remain 1
	if len(mockSender.outageAlerts) != 1 {
		t.Errorf("expected deduplication to prevent extra emails; got %d emails", len(mockSender.outageAlerts))
	}

	// --- STEP 4: First recovery check (M=1) -> Still recovering, incident remains OPEN (AC-4) ---
	check5 := &model.Check{
		MonitorID: mon.ID,
		Status:    model.StatusUp,
		CreatedAt: baseTime.Add(4 * time.Minute),
	}
	mgr.ProcessCheckResult(ctx, mon, check5)

	state, _ = mgr.GetMonitorState(mon.ID)
	if state.ConsecutiveSuccesses != 1 {
		t.Errorf("expected 1 consecutive success, got %d", state.ConsecutiveSuccesses)
	}
	if state.ActiveIncidentID == 0 {
		t.Error("expected incident to still be OPEN after only 1 success")
	}
	if len(mockSender.recoveryAlerts) != 0 {
		t.Errorf("expected 0 recovery emails on 1st success, got %d", len(mockSender.recoveryAlerts))
	}

	// DB incident should still be OPEN
	openInc, err = incidentStore.GetOpenIncidentByMonitor(ctx, mon.ID)
	if err != nil || openInc.Status != model.IncidentStatusOpen {
		t.Fatalf("expected incident to remain OPEN in DB, got err: %v", err)
	}

	// --- STEP 5: Second consecutive recovery check (M=2) -> Confirmed RESOLVED, recovery email sent (AC-4) ---
	resolveTime := baseTime.Add(5 * time.Minute)
	check6 := &model.Check{
		MonitorID: mon.ID,
		Status:    model.StatusUp,
		CreatedAt: resolveTime,
	}
	mgr.ProcessCheckResult(ctx, mon, check6)

	state, _ = mgr.GetMonitorState(mon.ID)
	if state.ActiveIncidentID != 0 {
		t.Errorf("expected active incident ID to be reset to 0, got %d", state.ActiveIncidentID)
	}
	if state.OutageAlertDispatched {
		t.Error("expected OutageAlertDispatched to be reset to false")
	}

	// Verify recovery email sent
	if len(mockSender.recoveryAlerts) != 1 {
		t.Fatalf("expected 1 recovery email dispatched, got %d", len(mockSender.recoveryAlerts))
	}
	if mockSender.recoveryAlerts[0] != "ops@goodai.co.nz" {
		t.Errorf("expected recovery email to 'ops@goodai.co.nz', got '%s'", mockSender.recoveryAlerts[0])
	}

	// Verify DB incident is RESOLVED with duration
	incInDB, err := incidentStore.GetIncidentByID(ctx, openInc.ID)
	if err != nil {
		t.Fatalf("failed to query resolved incident: %v", err)
	}
	if incInDB.Status != model.IncidentStatusResolved {
		t.Errorf("expected incident status RESOLVED, got %s", incInDB.Status)
	}
	if incInDB.ResolvedAt == nil {
		t.Error("expected resolved_at to be populated")
	}
	// Downtime between check2 (10:01) and check6 (10:05) = 4 minutes = 240 seconds
	if incInDB.DurationSeconds != 240 {
		t.Errorf("expected duration 240 seconds, got %d", incInDB.DurationSeconds)
	}
}

func TestAlertManager_EmailFailureDoesNotCrash(t *testing.T) {
	db, userStore, monitorStore, incidentStore, mockSender, mgr := setupTestAlertManager(t)
	defer db.Close()
	ctx := context.Background()

	mockSender.shouldFail = true // simulate network or SMTP auth error

	u, _ := userStore.CreateUser(ctx, &model.User{Email: "fail@goodai.co.nz", PasswordHash: "h", Name: "Fail User"})
	mon, _ := monitorStore.CreateMonitor(ctx, &model.Monitor{UserID: u.ID, Name: "Fail Target", URL: "https://fail.test", IntervalSeconds: 60, TimeoutSeconds: 5, ExpectedStatusCode: 200})

	// Process 2 failures -> should log warning and not panic
	mgr.ProcessCheckResult(ctx, mon, &model.Check{MonitorID: mon.ID, Status: model.StatusDown, ErrorMessage: "Err", CreatedAt: time.Now()})
	mgr.ProcessCheckResult(ctx, mon, &model.Check{MonitorID: mon.ID, Status: model.StatusDown, ErrorMessage: "Err", CreatedAt: time.Now()})

	// Incident should still be recorded in DB
	activeCount, err := incidentStore.CountActiveIncidentsByUser(ctx, u.ID)
	if err != nil || activeCount != 1 {
		t.Errorf("expected incident recorded in DB despite SMTP failure, got %d (err: %v)", activeCount, err)
	}
}
