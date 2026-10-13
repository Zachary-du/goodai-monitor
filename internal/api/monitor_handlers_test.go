package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

func setupMonitorAPITest(t *testing.T) (*storage.DB, *storage.UserStore, *storage.MonitorStore, *model.User, *model.User) {
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

	return db, userStore, monitorStore, user1, user2
}

func withAuthContext(r *http.Request, user *model.User) *http.Request {
	ctx := context.WithValue(r.Context(), UserIDContextKey, user.ID)
	ctx = context.WithValue(ctx, UserContextKey, user)
	return r.WithContext(ctx)
}

func TestListMonitorsHandler(t *testing.T) {
	db, _, monitorStore, user1, user2 := setupMonitorAPITest(t)
	defer db.Close()

	ctx := context.Background()

	// Seed User 1 with 2 monitors
	_, _ = monitorStore.CreateMonitor(ctx, &model.Monitor{UserID: user1.ID, Name: "A1", URL: "https://example.com/a1"})
	_, _ = monitorStore.CreateMonitor(ctx, &model.Monitor{UserID: user1.ID, Name: "A2", URL: "https://example.com/a2"})

	// Seed User 2 with 1 monitor
	_, _ = monitorStore.CreateMonitor(ctx, &model.Monitor{UserID: user2.ID, Name: "B1", URL: "https://example.com/b1"})

	handler := RequireAuth(ListMonitorsHandler(monitorStore))

	// Test Unauthenticated
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/monitors", nil)
	unauthRec := httptest.NewRecorder()
	handler.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", unauthRec.Code)
	}

	// Test User 1 sees strictly their 2 monitors
	req1 := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/monitors", nil), user1)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 for user1 list, got %d", rec1.Code)
	}

	var res1 struct {
		Monitors []*model.Monitor `json:"monitors"`
		Total    int              `json:"total"`
	}
	if err := json.NewDecoder(rec1.Body).Decode(&res1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res1.Total != 2 || len(res1.Monitors) != 2 {
		t.Errorf("expected 2 monitors for user1, got total=%d len=%d", res1.Total, len(res1.Monitors))
	}

	// Test User 2 sees strictly their 1 monitor
	req2 := withAuthContext(httptest.NewRequest(http.MethodGet, "/api/v1/monitors", nil), user2)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for user2 list, got %d", rec2.Code)
	}

	var res2 struct {
		Monitors []*model.Monitor `json:"monitors"`
		Total    int              `json:"total"`
	}
	if err := json.NewDecoder(rec2.Body).Decode(&res2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res2.Total != 1 || len(res2.Monitors) != 1 {
		t.Errorf("expected 1 monitor for user2, got total=%d len=%d", res2.Total, len(res2.Monitors))
	}
	if res2.Monitors[0].Name != "B1" {
		t.Errorf("expected monitor B1, got %s", res2.Monitors[0].Name)
	}
}

func TestCreateMonitorHandler_Success(t *testing.T) {
	db, _, monitorStore, user1, _ := setupMonitorAPITest(t)
	defer db.Close()

	handler := RequireAuth(CreateMonitorHandler(monitorStore))

	payload := model.CreateMonitorRequest{
		Name:               "Production Site",
		URL:                "https://example.com",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		ExpectedStatusCode: 200,
		Keyword:            "Welcome",
	}
	body, _ := json.Marshal(payload)

	req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/monitors", bytes.NewReader(body)), user1)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Message string         `json:"message"`
		Monitor *model.Monitor `json:"monitor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Monitor == nil || resp.Monitor.ID <= 0 {
		t.Errorf("expected created monitor with valid ID")
	}
	if resp.Monitor.Name != "Production Site" {
		t.Errorf("expected name 'Production Site', got '%s'", resp.Monitor.Name)
	}
	if resp.Monitor.Status != model.StatusPending {
		t.Errorf("expected status %s, got %s", model.StatusPending, resp.Monitor.Status)
	}
}

func TestCreateMonitorHandler_SSRFRejection(t *testing.T) {
	db, _, monitorStore, user1, _ := setupMonitorAPITest(t)
	defer db.Close()

	handler := RequireAuth(CreateMonitorHandler(monitorStore))

	// AC-2 & AC-3 forbidden targets
	ssrfTargets := []string{
		"http://127.0.0.1",
		"http://localhost",
		"http://169.254.169.254/opc/v2/instance/",
		"http://10.0.0.1",
		"http://192.168.1.1",
		"http://172.16.0.1",
		"http://example.com:60324",
		"http://example.com:5244",
		"http://example.com:22",
	}

	for _, target := range ssrfTargets {
		t.Run("SSRF_"+target, func(t *testing.T) {
			payload := model.CreateMonitorRequest{
				Name: "Malicious Target",
				URL:  target,
			}
			body, _ := json.Marshal(payload)

			req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/monitors", bytes.NewReader(body)), user1)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for %s, got %d: %s", target, rec.Code, rec.Body.String())
			}

			var errResp map[string]string
			_ = json.NewDecoder(rec.Body).Decode(&errResp)
			if errResp["error"] != "Invalid or restricted target URL" {
				t.Errorf("expected error 'Invalid or restricted target URL', got '%s'", errResp["error"])
			}
		})
	}
}

func TestCreateMonitorHandler_QuotaEnforcement(t *testing.T) {
	db, _, monitorStore, user1, _ := setupMonitorAPITest(t)
	defer db.Close()

	ctx := context.Background()

	// Fill user1 to the quota limit (10 monitors)
	for i := 1; i <= 10; i++ {
		_, err := monitorStore.CreateMonitor(ctx, &model.Monitor{
			UserID: user1.ID,
			Name:   fmt.Sprintf("Monitor %d", i),
			URL:    fmt.Sprintf("https://example.com/%d", i),
		})
		if err != nil {
			t.Fatalf("failed to insert monitor %d: %v", i, err)
		}
	}

	handler := RequireAuth(CreateMonitorHandler(monitorStore))

	// Attempting to create the 11th monitor should be rejected with 403 Forbidden
	payload := model.CreateMonitorRequest{
		Name: "Monitor 11 (Exceed Quota)",
		URL:  "https://example.com/11",
	}
	body, _ := json.Marshal(payload)

	req := withAuthContext(httptest.NewRequest(http.MethodPost, "/api/v1/monitors", bytes.NewReader(body)), user1)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on 11th monitor, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&errResp)
	if errResp["error"] != "Monitor quota exceeded (maximum 10 monitors allowed)" {
		t.Errorf("unexpected quota error message: %s", errResp["error"])
	}
}

func TestGetMonitorHandler(t *testing.T) {
	db, _, monitorStore, user1, user2 := setupMonitorAPITest(t)
	defer db.Close()

	ctx := context.Background()
	m1, _ := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID: user1.ID,
		Name:   "User1 Target",
		URL:    "https://example.com/u1",
	})

	handler := RequireAuth(GetMonitorHandler(monitorStore))

	// User 1 gets their own monitor
	req1 := withAuthContext(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/monitors/%d", m1.ID), nil), user1)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec1.Code)
	}

	// User 2 attempts to get User 1's monitor -> 404 Not Found
	req2 := withAuthContext(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/monitors/%d", m1.ID), nil), user2)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for multi-tenant isolation, got %d", rec2.Code)
	}
}

func TestUpdateMonitorHandler(t *testing.T) {
	db, _, monitorStore, user1, user2 := setupMonitorAPITest(t)
	defer db.Close()

	ctx := context.Background()
	m1, _ := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID: user1.ID,
		Name:   "Initial Name",
		URL:    "https://example.com/initial",
	})

	handler := RequireAuth(UpdateMonitorHandler(monitorStore))

	// User 1 updates successfully
	updatePayload := model.UpdateMonitorRequest{
		Name:               "Renamed Target",
		URL:                "https://example.com/updated",
		IntervalSeconds:    300,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
	}
	body, _ := json.Marshal(updatePayload)
	req1 := withAuthContext(httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/monitors/%d", m1.ID), bytes.NewReader(body)), user1)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on update, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Update with SSRF URL should be rejected with 400
	ssrfPayload := model.UpdateMonitorRequest{
		Name: "SSRF Update",
		URL:  "http://127.0.0.1",
	}
	bodySSRF, _ := json.Marshal(ssrfPayload)
	reqSSRF := withAuthContext(httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/monitors/%d", m1.ID), bytes.NewReader(bodySSRF)), user1)
	recSSRF := httptest.NewRecorder()
	handler.ServeHTTP(recSSRF, reqSSRF)
	if recSSRF.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on SSRF update, got %d", recSSRF.Code)
	}

	// User 2 cannot update User 1's monitor -> 404
	req2 := withAuthContext(httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/monitors/%d", m1.ID), bytes.NewReader(body)), user2)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on cross-tenant update, got %d", rec2.Code)
	}
}

func TestDeleteMonitorHandler(t *testing.T) {
	db, _, monitorStore, user1, user2 := setupMonitorAPITest(t)
	defer db.Close()

	ctx := context.Background()
	m1, _ := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID: user1.ID,
		Name:   "To Delete",
		URL:    "https://example.com/delete",
	})

	handler := RequireAuth(DeleteMonitorHandler(monitorStore))

	// User 2 attempts to delete User 1's monitor -> 404
	req2 := withAuthContext(httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/monitors/%d", m1.ID), nil), user2)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on cross-tenant delete, got %d", rec2.Code)
	}

	// User 1 deletes their own monitor -> 200
	req1 := withAuthContext(httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/monitors/%d", m1.ID), nil), user1)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete, got %d", rec1.Code)
	}
}

func TestTogglePauseMonitorHandler(t *testing.T) {
	db, _, monitorStore, user1, user2 := setupMonitorAPITest(t)
	defer db.Close()

	ctx := context.Background()
	m1, _ := monitorStore.CreateMonitor(ctx, &model.Monitor{
		UserID: user1.ID,
		Name:   "Toggle Target",
		URL:    "https://example.com/toggle",
	})

	handler := RequireAuth(TogglePauseMonitorHandler(monitorStore))

	// User 2 attempts to toggle User 1's monitor -> 404
	req2 := withAuthContext(httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%d/pause", m1.ID), nil), user2)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on cross-tenant pause toggle, got %d", rec2.Code)
	}

	// User 1 toggles: PENDING -> PAUSED
	req1 := withAuthContext(httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%d/pause", m1.ID), nil), user1)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on pause toggle, got %d", rec1.Code)
	}

	var resp struct {
		Monitor *model.Monitor `json:"monitor"`
	}
	_ = json.NewDecoder(rec1.Body).Decode(&resp)
	if resp.Monitor.Status != model.StatusPaused {
		t.Errorf("expected status %s, got %s", model.StatusPaused, resp.Monitor.Status)
	}

	// User 1 toggles again: PAUSED -> PENDING
	req1Resume := withAuthContext(httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%d/pause", m1.ID), nil), user1)
	rec1Resume := httptest.NewRecorder()
	handler.ServeHTTP(rec1Resume, req1Resume)
	if rec1Resume.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on resume toggle, got %d", rec1Resume.Code)
	}

	_ = json.NewDecoder(rec1Resume.Body).Decode(&resp)
	if resp.Monitor.Status != model.StatusPending {
		t.Errorf("expected status %s, got %s", model.StatusPending, resp.Monitor.Status)
	}
}
