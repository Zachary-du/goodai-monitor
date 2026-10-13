package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

func TestHealthzHandler_Success(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "health_test.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	router := NewRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got '%s'", contentType)
	}

	nosniff := rec.Header().Get("X-Content-Type-Options")
	if nosniff != "nosniff" {
		t.Errorf("expected X-Content-Type-Options 'nosniff', got '%s'", nosniff)
	}

	var resp HealthzResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
	if resp.Database != "connected" {
		t.Errorf("expected database 'connected', got '%s'", resp.Database)
	}
	if resp.UptimeSeconds < 0 {
		t.Errorf("expected non-negative uptime, got %d", resp.UptimeSeconds)
	}
	if resp.MemoryRSSBytes == 0 {
		t.Errorf("expected positive MemoryRSSBytes, got %d", resp.MemoryRSSBytes)
	}
	if resp.GoGoroutineCount <= 0 {
		t.Errorf("expected positive GoGoroutineCount, got %d", resp.GoGoroutineCount)
	}
}

func TestStaticFiles_Serving(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "static_test.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	router := NewRouter(db)

	testCases := []struct {
		name         string
		path         string
		expectedCode int
		bodySubstr   string
	}{
		{
			name:         "Root index.html",
			path:         "/",
			expectedCode: http.StatusOK,
			bodySubstr:   "GoodAI Monitor",
		},
		{
			name:         "Stylesheet style.css",
			path:         "/style.css",
			expectedCode: http.StatusOK,
			bodySubstr:   "status-banner",
		},
		{
			name:         "Javascript app.js",
			path:         "/app.js",
			expectedCode: http.StatusOK,
			bodySubstr:   "pollHealth",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tc.expectedCode {
				t.Fatalf("expected HTTP status %d, got %d", tc.expectedCode, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.bodySubstr) {
				t.Errorf("expected body to contain '%s'", tc.bodySubstr)
			}
		})
	}
}

func TestHealthzHandler_DBDisconnected(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "disconnect_test.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	// Forcefully close DB to simulate failure
	_ = db.Close()

	router := NewRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected HTTP 503 for disconnected DB, got %d", rec.Code)
	}

	var resp HealthzResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp.Status != "degraded" {
		t.Errorf("expected status 'degraded', got '%s'", resp.Status)
	}
	if resp.Database != "disconnected" {
		t.Errorf("expected database 'disconnected', got '%s'", resp.Database)
	}
}
