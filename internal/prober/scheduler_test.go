package prober

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

type mockProber struct {
	probeCount atomic.Int64
	mu         sync.Mutex
	lastTarget *model.Monitor
}

func (m *mockProber) ProbeTarget(ctx context.Context, mon *model.Monitor) *model.CheckResult {
	m.probeCount.Add(1)
	m.mu.Lock()
	m.lastTarget = mon
	m.mu.Unlock()

	return &model.CheckResult{
		Success:        true,
		StatusCode:     200,
		Status:         model.StatusUp,
		TotalLatencyMs: 45,
		LatencyBreakdown: model.LatencyBreakdown{
			DNSLookupMs:    5,
			TCPConnectMs:   10,
			TLSHandshakeMs: 15,
			TTFBMs:         15,
			TotalMs:        45,
		},
	}
}

func setupTestScheduler(t *testing.T) (*storage.DB, *storage.MonitorStore, *storage.CheckStore, *mockProber, *Scheduler) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "scheduler_test.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}

	monitorStore, err := storage.NewMonitorStore(db)
	if err != nil {
		t.Fatalf("failed to initialize monitor store: %v", err)
	}

	checkStore, err := storage.NewCheckStore(db)
	if err != nil {
		t.Fatalf("failed to initialize check store: %v", err)
	}

	mock := &mockProber{}
	scheduler := NewScheduler(checkStore, mock)

	return db, monitorStore, checkStore, mock, scheduler
}

func TestScheduler_StartAndInitialProbe(t *testing.T) {
	db, monitorStore, checkStore, mock, scheduler := setupTestScheduler(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Seed one active monitor
	m, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             1,
		Name:               "Test Scheduler Mon",
		URL:                "https://goodai.co.nz",
		IntervalSeconds:    1, // 1s interval for fast test
		TimeoutSeconds:     2,
		ExpectedStatusCode: 200,
		Status:             model.StatusPending,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	scheduler.Start(ctx)
	defer scheduler.Stop()

	// Wait up to 1 second for the initial probe to execute
	time.Sleep(300 * time.Millisecond)

	if mock.probeCount.Load() == 0 {
		t.Error("expected mock prober to be called at least once")
	}

	// Verify check was stored in DB
	recent, err := checkStore.GetRecentChecksByMonitor(ctx, m.ID, 5)
	if err != nil {
		t.Fatalf("failed to get recent checks: %v", err)
	}
	if len(recent) == 0 {
		t.Error("expected at least 1 check record stored in database")
	}
}

func TestScheduler_AddRemoveLifecycle(t *testing.T) {
	db, monitorStore, _, _, scheduler := setupTestScheduler(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler.Start(ctx)
	defer scheduler.Stop()

	if scheduler.ActiveJobCount() != 0 {
		t.Fatalf("expected 0 active jobs initially, got %d", scheduler.ActiveJobCount())
	}

	m, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             1,
		Name:               "Dynamically Added Target",
		URL:                "https://example.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
		Status:             model.StatusPending,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	scheduler.AddOrUpdateMonitor(m)

	if scheduler.ActiveJobCount() != 1 {
		t.Errorf("expected 1 active job after AddOrUpdateMonitor, got %d", scheduler.ActiveJobCount())
	}

	scheduler.RemoveMonitor(m.ID)

	if scheduler.ActiveJobCount() != 0 {
		t.Errorf("expected 0 active jobs after RemoveMonitor, got %d", scheduler.ActiveJobCount())
	}
}

func TestScheduler_PauseAndResume(t *testing.T) {
	db, monitorStore, _, _, scheduler := setupTestScheduler(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             1,
		Name:               "Pausable Target",
		URL:                "https://example.org",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
		Status:             model.StatusPending,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	scheduler.Start(ctx)
	defer scheduler.Stop()

	time.Sleep(100 * time.Millisecond)
	if scheduler.ActiveJobCount() != 1 {
		t.Fatalf("expected 1 active job, got %d", scheduler.ActiveJobCount())
	}

	// Pause monitor
	m.Status = model.StatusPaused
	scheduler.AddOrUpdateMonitor(m)

	if scheduler.ActiveJobCount() != 0 {
		t.Errorf("expected 0 active jobs after pausing monitor, got %d", scheduler.ActiveJobCount())
	}

	// Resume monitor
	m.Status = model.StatusUp
	scheduler.AddOrUpdateMonitor(m)

	if scheduler.ActiveJobCount() != 1 {
		t.Errorf("expected 1 active job after resuming monitor, got %d", scheduler.ActiveJobCount())
	}
}

func TestScheduler_StopCleanly(t *testing.T) {
	db, monitorStore, _, _, scheduler := setupTestScheduler(t)
	defer db.Close()

	ctx := context.Background()

	_, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             1,
		Name:               "Target 1",
		URL:                "https://example.com/1",
		IntervalSeconds:    1,
		TimeoutSeconds:     1,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	scheduler.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Stop must finish quickly without hanging
	stopped := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		// Clean termination
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler.Stop() timed out after 2 seconds")
	}

	if scheduler.ActiveJobCount() != 0 {
		t.Errorf("expected 0 jobs after Stop, got %d", scheduler.ActiveJobCount())
	}
}
