package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

func setupMetricsAPITest(t *testing.T) (*storage.DB, *storage.MonitorStore, *storage.CheckStore, *model.User, *model.User) {
	t.Helper()

	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	userStore, err := storage.NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to create user store: %v", err)
	}

	monitorStore, err := storage.NewMonitorStore(db)
	if err != nil {
		t.Fatalf("failed to create monitor store: %v", err)
	}

	checkStore, err := storage.NewCheckStore(db)
	if err != nil {
		t.Fatalf("failed to create check store: %v", err)
	}

	ctx := context.Background()
	user1, err := userStore.CreateUser(ctx, &model.User{
		Email:        "alice@goodai.co.nz",
		PasswordHash: "dummyhash1",
		Name:         "Alice",
	})
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	user2, err := userStore.CreateUser(ctx, &model.User{
		Email:        "bob@goodai.co.nz",
		PasswordHash: "dummyhash2",
		Name:         "Bob",
	})
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	return db, monitorStore, checkStore, user1, user2
}

func TestMonitorMetricsHandler_Unauthenticated(t *testing.T) {
	db, monitorStore, checkStore, _, _ := setupMetricsAPITest(t)
	defer db.Close()

	handler := MonitorMetricsHandler(monitorStore, checkStore)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/monitors/1/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
	}
}

func TestMonitorMetricsHandler_TenantIsolation(t *testing.T) {
	db, monitorStore, checkStore, user1, user2 := setupMetricsAPITest(t)
	defer db.Close()

	ctx := context.Background()
	// User1 owns this monitor
	m, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             user1.ID,
		Name:               "Alice Service",
		URL:                "https://goodai.co.nz",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	handler := MonitorMetricsHandler(monitorStore, checkStore)

	// User2 tries to access User1's monitor metrics
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/monitors/%d/metrics", m.ID), nil)
	req = withAuthContext(req, user2)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Must return 404 Not Found to prevent ID enumeration
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 Not Found across tenant boundary, got %d", w.Code)
	}
}

func TestMonitorMetricsHandler_Success(t *testing.T) {
	db, monitorStore, checkStore, user1, _ := setupMetricsAPITest(t)
	defer db.Close()

	ctx := context.Background()
	m, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             user1.ID,
		Name:               "Telemetry Target",
		URL:                "https://goodai.co.nz",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	// Insert checks: 2 UP (50ms, 70ms) and 1 DOWN (5000ms)
	now := time.Now().UTC()
	_, _ = checkStore.RecordCheck(ctx, &model.Check{
		MonitorID: m.ID,
		Status:    model.StatusUp,
		LatencyMs: 50,
		CreatedAt: now.Add(-30 * time.Minute),
	})
	_, _ = checkStore.RecordCheck(ctx, &model.Check{
		MonitorID: m.ID,
		Status:    model.StatusUp,
		LatencyMs: 70,
		CreatedAt: now.Add(-20 * time.Minute),
	})
	_, _ = checkStore.RecordCheck(ctx, &model.Check{
		MonitorID:    m.ID,
		Status:       model.StatusDown,
		LatencyMs:    5000,
		ErrorMessage: "HTTP 502 Bad Gateway",
		CreatedAt:    now.Add(-10 * time.Minute),
	})

	handler := MonitorMetricsHandler(monitorStore, checkStore)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/monitors/%d/metrics", m.ID), nil)
	req = withAuthContext(req, user1)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d (body: %s)", w.Code, w.Body.String())
	}

	var summary model.MetricsSummary
	if err := json.Unmarshal(w.Body.Bytes(), &summary); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if summary.MonitorID != m.ID {
		t.Errorf("expected monitor ID %d, got %d", m.ID, summary.MonitorID)
	}
	// 2 UP out of 3 = 66.67%
	if summary.Uptime24h < 66.0 || summary.Uptime24h > 67.0 {
		t.Errorf("expected uptime around 66.67%%, got %.2f%%", summary.Uptime24h)
	}
	if summary.TotalChecks24h != 3 {
		t.Errorf("expected 3 total checks, got %d", summary.TotalChecks24h)
	}
	if len(summary.LatencyHistory) != 3 {
		t.Errorf("expected 3 latency history points, got %d", len(summary.LatencyHistory))
	}
}

func TestMonitorChecksHandler_Success(t *testing.T) {
	db, monitorStore, checkStore, user1, _ := setupMetricsAPITest(t)
	defer db.Close()

	ctx := context.Background()
	m, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID:             user1.ID,
		Name:               "Checks Target",
		URL:                "https://goodai.co.nz",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	now := time.Now().UTC()
	for i := 1; i <= 5; i++ {
		_, _ = checkStore.RecordCheck(ctx, &model.Check{
			MonitorID:  m.ID,
			Status:     model.StatusUp,
			StatusCode: 200,
			LatencyMs:  int64(i * 10),
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
		})
	}

	handler := MonitorChecksHandler(monitorStore, checkStore)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/monitors/%d/checks?limit=3", m.ID), nil)
	req = withAuthContext(req, user1)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Code)
	}

	var res struct {
		Checks []*model.Check `json:"checks"`
		Total  int            `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse checks JSON: %v", err)
	}

	if res.Total != 3 {
		t.Errorf("expected 3 checks with limit=3, got %d", res.Total)
	}
	if len(res.Checks) != 3 {
		t.Errorf("expected 3 check objects, got %d", len(res.Checks))
	}
	// Verify descending order
	if res.Checks[0].LatencyMs != 50 {
		t.Errorf("expected latest check latency to be 50, got %d", res.Checks[0].LatencyMs)
	}
}
