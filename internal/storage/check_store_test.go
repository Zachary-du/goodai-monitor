package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func setupTestCheckStore(t *testing.T) (*DB, *MonitorStore, *CheckStore) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_check_store.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}

	monitorStore, err := NewMonitorStore(db)
	if err != nil {
		t.Fatalf("failed to initialize monitor store: %v", err)
	}

	checkStore, err := NewCheckStore(db)
	if err != nil {
		t.Fatalf("failed to initialize check store: %v", err)
	}

	return db, monitorStore, checkStore
}

func createTestMonitor(t *testing.T, monitorStore *MonitorStore, userID int64, name, url string) *model.Monitor {
	t.Helper()
	m, err := monitorStore.CreateMonitor(context.Background(), &model.Monitor{
		UserID:             userID,
		Name:               name,
		URL:                url,
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("failed to create test monitor: %v", err)
	}
	return m
}

func TestCheckStore_RecordCheckAndSync(t *testing.T) {
	db, monitorStore, checkStore := setupTestCheckStore(t)
	defer db.Close()

	ctx := context.Background()
	m := createTestMonitor(t, monitorStore, 1, "API Target", "https://goodai.co.nz")

	days := 45
	check := &model.Check{
		MonitorID:      m.ID,
		Status:         model.StatusUp,
		StatusCode:     200,
		LatencyMs:      120,
		DNSLookupMs:    15,
		TCPConnectMs:   25,
		TLSHandshakeMs: 40,
		TTFBMs:         35,
		CertExpiryDays: &days,
		CreatedAt:      time.Now().UTC(),
	}

	recorded, err := checkStore.RecordCheck(ctx, check)
	if err != nil {
		t.Fatalf("failed to record check: %v", err)
	}

	if recorded.ID <= 0 {
		t.Errorf("expected positive check ID, got %d", recorded.ID)
	}

	// Verify monitor was updated with the latest telemetry
	updatedMonitor, err := monitorStore.GetMonitorByIDAndUser(ctx, m.ID, 1)
	if err != nil {
		t.Fatalf("failed to retrieve updated monitor: %v", err)
	}

	if updatedMonitor.Status != model.StatusUp {
		t.Errorf("expected monitor status UP, got %s", updatedMonitor.Status)
	}
	if updatedMonitor.LastLatencyMs != 120 {
		t.Errorf("expected last latency 120ms, got %d", updatedMonitor.LastLatencyMs)
	}
	if updatedMonitor.LastCheckAt == nil {
		t.Error("expected non-nil last_check_at timestamp")
	}
	if updatedMonitor.UptimePercent != 100.0 {
		t.Errorf("expected uptime 100%%, got %.2f%%", updatedMonitor.UptimePercent)
	}
}

func TestCheckStore_GetRecentChecks(t *testing.T) {
	db, monitorStore, checkStore := setupTestCheckStore(t)
	defer db.Close()

	ctx := context.Background()
	m := createTestMonitor(t, monitorStore, 1, "Recent Checks Target", "https://example.com")

	// Insert 10 checks
	baseTime := time.Now().UTC().Add(-10 * time.Minute)
	for i := 1; i <= 10; i++ {
		_, err := checkStore.RecordCheck(ctx, &model.Check{
			MonitorID:    m.ID,
			Status:       model.StatusUp,
			StatusCode:   200,
			LatencyMs:    int64(i * 10),
			ErrorMessage: "",
			CreatedAt:    baseTime.Add(time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatalf("failed to insert check %d: %v", i, err)
		}
	}

	// Query with limit 5
	recent, err := checkStore.GetRecentChecksByMonitor(ctx, m.ID, 5)
	if err != nil {
		t.Fatalf("failed to query recent checks: %v", err)
	}

	if len(recent) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(recent))
	}

	// Ensure descending ID order
	if recent[0].LatencyMs != 100 {
		t.Errorf("expected first check latency to be 100 (latest), got %d", recent[0].LatencyMs)
	}
	if recent[4].LatencyMs != 60 {
		t.Errorf("expected fifth check latency to be 60, got %d", recent[4].LatencyMs)
	}
}

func TestCheckStore_CalculateUptimeStats(t *testing.T) {
	db, monitorStore, checkStore := setupTestCheckStore(t)
	defer db.Close()

	ctx := context.Background()
	m := createTestMonitor(t, monitorStore, 1, "Uptime Target", "https://example.org")

	// When no checks exist, uptime should default to 100.0%
	emptyStats, err := checkStore.CalculateUptimeStats(ctx, m.ID)
	if err != nil {
		t.Fatalf("failed to calculate stats for empty monitor: %v", err)
	}
	if emptyStats.Uptime24h != 100.0 || emptyStats.Uptime7d != 100.0 || emptyStats.Uptime30d != 100.0 {
		t.Errorf("expected 100%% uptime for empty monitor, got 24h=%.2f 7d=%.2f 30d=%.2f",
			emptyStats.Uptime24h, emptyStats.Uptime7d, emptyStats.Uptime30d)
	}

	now := time.Now().UTC()

	// Insert: 3 UP checks and 1 DOWN check within the last 2 hours
	for i := 0; i < 3; i++ {
		_, err := checkStore.RecordCheck(ctx, &model.Check{
			MonitorID: m.ID,
			Status:    model.StatusUp,
			LatencyMs: 50,
			CreatedAt: now.Add(-time.Duration(i*30) * time.Minute),
		})
		if err != nil {
			t.Fatalf("failed to record UP check: %v", err)
		}
	}

	_, err = checkStore.RecordCheck(ctx, &model.Check{
		MonitorID:    m.ID,
		Status:       model.StatusDown,
		LatencyMs:    5000,
		ErrorMessage: "Connection timeout after 5000ms",
		CreatedAt:    now.Add(-10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("failed to record DOWN check: %v", err)
	}

	// 3 UP out of 4 total checks = 75.0%
	stats, err := checkStore.CalculateUptimeStats(ctx, m.ID)
	if err != nil {
		t.Fatalf("failed to calculate uptime stats: %v", err)
	}

	if stats.Uptime24h != 75.0 {
		t.Errorf("expected 24h uptime 75.00%%, got %.2f%%", stats.Uptime24h)
	}
	if stats.TotalChecks24h != 4 {
		t.Errorf("expected 4 total checks in 24h, got %d", stats.TotalChecks24h)
	}
	// Average latency: (50*3 + 5000) / 4 = 5150 / 4 = 1287.5
	if stats.AvgLatency24h != 1287.5 {
		t.Errorf("expected avg latency 1287.5ms, got %.1fms", stats.AvgLatency24h)
	}
	if len(stats.LatencyHistory) != 4 {
		t.Errorf("expected 4 latency history points, got %d", len(stats.LatencyHistory))
	}
}

func TestCheckStore_PruneOldChecks(t *testing.T) {
	db, monitorStore, checkStore := setupTestCheckStore(t)
	defer db.Close()

	ctx := context.Background()
	m := createTestMonitor(t, monitorStore, 1, "Prune Target", "https://example.com")

	now := time.Now().UTC()

	// 1 check from 20 days ago (should be pruned)
	_, err := checkStore.RecordCheck(ctx, &model.Check{
		MonitorID: m.ID,
		Status:    model.StatusUp,
		LatencyMs: 100,
		CreatedAt: now.Add(-20 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to insert old check: %v", err)
	}

	// 1 check from 15 days ago (should be pruned)
	_, err = checkStore.RecordCheck(ctx, &model.Check{
		MonitorID: m.ID,
		Status:    model.StatusUp,
		LatencyMs: 100,
		CreatedAt: now.Add(-15 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to insert 15-day check: %v", err)
	}

	// 1 check from 5 days ago (should be retained)
	_, err = checkStore.RecordCheck(ctx, &model.Check{
		MonitorID: m.ID,
		Status:    model.StatusUp,
		LatencyMs: 100,
		CreatedAt: now.Add(-5 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to insert recent check: %v", err)
	}

	// Prune records older than 14 days
	pruned, err := checkStore.PruneOldChecks(ctx, 14*24*time.Hour)
	if err != nil {
		t.Fatalf("failed to prune old checks: %v", err)
	}

	if pruned != 2 {
		t.Errorf("expected 2 pruned rows, got %d", pruned)
	}

	// Only 1 check should remain
	remaining, err := checkStore.GetRecentChecksByMonitor(ctx, m.ID, 10)
	if err != nil {
		t.Fatalf("failed to query remaining checks: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining check, got %d", len(remaining))
	}
}

func TestCheckStore_ActiveAndPausedMonitors(t *testing.T) {
	db, monitorStore, checkStore := setupTestCheckStore(t)
	defer db.Close()

	ctx := context.Background()
	m1 := createTestMonitor(t, monitorStore, 1, "Active 1", "https://goodai.co.nz")
	m2 := createTestMonitor(t, monitorStore, 1, "Active 2", "https://example.com")

	// Pause m2
	_, err := monitorStore.ToggleMonitorPause(ctx, m2.ID, 1)
	if err != nil {
		t.Fatalf("failed to pause monitor m2: %v", err)
	}

	active, err := checkStore.GetAllActiveMonitors(ctx)
	if err != nil {
		t.Fatalf("failed to get active monitors: %v", err)
	}

	if len(active) != 1 {
		t.Fatalf("expected 1 active monitor, got %d", len(active))
	}
	if active[0].ID != m1.ID {
		t.Errorf("expected active monitor ID %d, got %d", m1.ID, active[0].ID)
	}
}
