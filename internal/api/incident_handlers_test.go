package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

func setupTestIncidentAPI(t *testing.T) (*storage.DB, *storage.UserStore, *storage.MonitorStore, *storage.IncidentStore) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_incident_api.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	uStore, err := storage.NewUserStore(db)
	if err != nil {
		t.Fatalf("failed to init user store: %v", err)
	}

	mStore, err := storage.NewMonitorStore(db)
	if err != nil {
		t.Fatalf("failed to init monitor store: %v", err)
	}

	iStore, err := storage.NewIncidentStore(db)
	if err != nil {
		t.Fatalf("failed to init incident store: %v", err)
	}

	return db, uStore, mStore, iStore
}

func TestIncidentHandlers_AuthenticationRequired(t *testing.T) {
	db, _, _, iStore := setupTestIncidentAPI(t)
	defer db.Close()

	listHandler := ListIncidentsHandler(iStore)
	summaryHandler := IncidentSummaryHandler(iStore)

	// Unauthenticated GET /api/v1/incidents
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	rr1 := httptest.NewRecorder()
	listHandler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for list without auth, got %d", rr1.Code)
	}

	// Unauthenticated GET /api/v1/incidents/summary
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/summary", nil)
	rr2 := httptest.NewRecorder()
	summaryHandler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for summary without auth, got %d", rr2.Code)
	}
}

func TestIncidentHandlers_MultiTenantIsolationAndFiltering(t *testing.T) {
	db, uStore, mStore, iStore := setupTestIncidentAPI(t)
	defer db.Close()
	ctx := context.Background()

	// Create User 1 & User 2
	user1, _ := uStore.CreateUser(ctx, &model.User{Email: "user1@example.com", PasswordHash: "h1", Name: "User 1"})
	user2, _ := uStore.CreateUser(ctx, &model.User{Email: "user2@example.com", PasswordHash: "h2", Name: "User 2"})

	mon1, _ := mStore.CreateMonitor(ctx, &model.Monitor{UserID: user1.ID, Name: "Mon 1", URL: "https://mon1.test", IntervalSeconds: 60, TimeoutSeconds: 5, ExpectedStatusCode: 200})
	mon2, _ := mStore.CreateMonitor(ctx, &model.Monitor{UserID: user2.ID, Name: "Mon 2", URL: "https://mon2.test", IntervalSeconds: 60, TimeoutSeconds: 5, ExpectedStatusCode: 200})

	// User 1 has 1 open and 1 resolved incident
	inc1, _ := iStore.CreateIncident(ctx, &model.Incident{MonitorID: mon1.ID, UserID: user1.ID, StartedAt: time.Now().UTC().Add(-10 * time.Minute), Status: model.IncidentStatusOpen, Cause: "502 Bad Gateway"})
	_ = iStore.ResolveIncident(ctx, inc1.ID, time.Now().UTC(), 600)

	_, _ = iStore.CreateIncident(ctx, &model.Incident{MonitorID: mon1.ID, UserID: user1.ID, StartedAt: time.Now().UTC().Add(-1 * time.Minute), Status: model.IncidentStatusOpen, Cause: "Timeout"})

	// User 2 has 1 open incident
	_, _ = iStore.CreateIncident(ctx, &model.Incident{MonitorID: mon2.ID, UserID: user2.ID, StartedAt: time.Now().UTC(), Status: model.IncidentStatusOpen, Cause: "Connection Refused"})

	listHandler := ListIncidentsHandler(iStore)
	summaryHandler := IncidentSummaryHandler(iStore)

	// User 1 requests all incidents
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDContextKey, user1.ID))
	rr := httptest.NewRecorder()
	listHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	var resp struct {
		Incidents []*model.Incident `json:"incidents"`
		Count     int               `json:"count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Count != 2 {
		t.Fatalf("expected 2 incidents for User 1, got %d", resp.Count)
	}

	// Filter by status=open for User 1
	reqOpen := httptest.NewRequest(http.MethodGet, "/api/v1/incidents?status=open", nil)
	reqOpen = reqOpen.WithContext(context.WithValue(reqOpen.Context(), UserIDContextKey, user1.ID))
	rrOpen := httptest.NewRecorder()
	listHandler.ServeHTTP(rrOpen, reqOpen)

	var respOpen struct {
		Incidents []*model.Incident `json:"incidents"`
		Count     int               `json:"count"`
	}
	_ = json.Unmarshal(rrOpen.Body.Bytes(), &respOpen)
	if respOpen.Count != 1 || respOpen.Incidents[0].Status != model.IncidentStatusOpen {
		t.Errorf("expected 1 OPEN incident for User 1, got %d", respOpen.Count)
	}

	// User 1 summary
	reqSummary := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/summary", nil)
	reqSummary = reqSummary.WithContext(context.WithValue(reqSummary.Context(), UserIDContextKey, user1.ID))
	rrSummary := httptest.NewRecorder()
	summaryHandler.ServeHTTP(rrSummary, reqSummary)

	if rrSummary.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for summary, got %d", rrSummary.Code)
	}
	var summary model.IncidentSummary
	_ = json.Unmarshal(rrSummary.Body.Bytes(), &summary)
	if summary.OpenCount != 1 || summary.ResolvedCount != 1 || summary.TotalCount != 2 {
		t.Errorf("unexpected summary for User 1: %+v", summary)
	}

	// User 2 requests incidents (isolation check)
	reqUser2 := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	reqUser2 = reqUser2.WithContext(context.WithValue(reqUser2.Context(), UserIDContextKey, user2.ID))
	rrUser2 := httptest.NewRecorder()
	listHandler.ServeHTTP(rrUser2, reqUser2)

	var respUser2 struct {
		Incidents []*model.Incident `json:"incidents"`
		Count     int               `json:"count"`
	}
	_ = json.Unmarshal(rrUser2.Body.Bytes(), &respUser2)
	if respUser2.Count != 1 {
		t.Fatalf("expected 1 incident for User 2, got %d", respUser2.Count)
	}
	if respUser2.Incidents[0].MonitorName != "Mon 2" {
		t.Errorf("expected incident to be for Mon 2, got %s", respUser2.Incidents[0].MonitorName)
	}
}

func TestIncidentHandlers_MethodNotAllowed(t *testing.T) {
	db, _, _, iStore := setupTestIncidentAPI(t)
	defer db.Close()

	listHandler := ListIncidentsHandler(iStore)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", nil)
	rr := httptest.NewRecorder()
	listHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 Method Not Allowed, got %d", rr.Code)
	}
}
