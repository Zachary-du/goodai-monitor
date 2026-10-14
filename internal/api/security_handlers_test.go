package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

type mockAuditRunner struct {
	report *model.SecurityAuditReport
	err    error
}

func (m *mockAuditRunner) RunAudit(ctx context.Context, rawURL string, userID int64) (*model.SecurityAuditReport, error) {
	if m.err != nil {
		return nil, m.err
	}
	r := *m.report
	r.UserID = userID
	r.TargetURL = rawURL
	return &r, nil
}

func setupTestSecurityAPI(t *testing.T) (*storage.DB, *storage.UserStore, *storage.MonitorStore, *storage.SecurityStore) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_security_api.db")

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

	secStore, err := storage.NewSecurityStore(db)
	if err != nil {
		t.Fatalf("failed to init security store: %v", err)
	}

	return db, uStore, mStore, secStore
}

func TestSecurityHandlers_AuthRequired(t *testing.T) {
	db, _, mStore, secStore := setupTestSecurityAPI(t)
	defer db.Close()

	auditHandler := RunSecurityAuditHandler(nil, secStore, mStore)
	listHandler := ListSecurityReportsHandler(secStore)
	getHandler := GetSecurityReportHandler(secStore)
	exportHandler := ExportSecurityReportHandler(secStore)

	// Audit without auth
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/security/audit", strings.NewReader(`{"target_url":"https://goodai.co.nz"}`))
	rr1 := httptest.NewRecorder()
	auditHandler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated audit, got %d", rr1.Code)
	}

	// List without auth
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/security/reports", nil)
	rr2 := httptest.NewRecorder()
	listHandler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated list, got %d", rr2.Code)
	}

	// Get without auth
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/security/reports/1", nil)
	rr3 := httptest.NewRecorder()
	getHandler.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated get, got %d", rr3.Code)
	}

	// Export without auth
	req4 := httptest.NewRequest(http.MethodGet, "/api/v1/security/reports/1/export", nil)
	rr4 := httptest.NewRecorder()
	exportHandler.ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated export, got %d", rr4.Code)
	}
}

func TestSecurityHandlers_SSRFRejection(t *testing.T) {
	db, _, mStore, secStore := setupTestSecurityAPI(t)
	defer db.Close()

	handler := RunSecurityAuditHandler(nil, secStore, mStore)

	restrictedPayloads := []string{
		`{"target_url":"http://127.0.0.1:8080/admin"}`,
		`{"target_url":"http://10.0.0.1"}`,
		`{"target_url":"http://169.254.169.254/latest/meta-data/"}`,
		`{"target_url":"http://localhost:8080"}`,
	}

	for _, p := range restrictedPayloads {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/security/audit", strings.NewReader(p))
		req = req.WithContext(context.WithValue(req.Context(), UserIDContextKey, int64(1)))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for restricted SSRF URL, got %d (body: %s)", rr.Code, rr.Body.String())
		}
	}
}

func TestSecurityHandlers_AuditAndExportFlow(t *testing.T) {
	db, uStore, mStore, secStore := setupTestSecurityAPI(t)
	defer db.Close()
	ctx := context.Background()

	user1, _ := uStore.CreateUser(ctx, &model.User{Email: "user1@example.com", PasswordHash: "h1", Name: "User 1"})
	user2, _ := uStore.CreateUser(ctx, &model.User{Email: "user2@example.com", PasswordHash: "h2", Name: "User 2"})

	mockReport := &model.SecurityAuditReport{
		TargetHost:     "goodai.co.nz",
		TargetIP:       "158.179.29.27",
		CompositeScore: 95,
		Grade:          "A+",
		GradeLabel:     "Hardened Defense",
		ExecutionMs:    210,
		CreatedAt:      time.Now().UTC(),
		Attestation:    "test-hash-456",
		DomainScores: []model.CategoryBreakdown{
			{DomainKey: "ai_governance", DomainName: "AI Bot Governance", Score: 25, MaxScore: 25, Status: "Hardened"},
			{DomainKey: "headers", DomainName: "Security Headers", Score: 25, MaxScore: 25, Status: "Hardened"},
			{DomainKey: "tls", DomainName: "TLS Cryptography", Score: 25, MaxScore: 25, Status: "Hardened"},
			{DomainKey: "exposure", DomainName: "Surface Exposure", Score: 20, MaxScore: 25, Status: "Strong"},
		},
		Findings: []model.FindingItem{
			{ID: "f1", Domain: "tls", CheckName: "TLS 1.3", Severity: model.SeverityPass, ScoreDeduction: 0, ObservedValue: "Enforced"},
			{ID: "f2", Domain: "exposure", CheckName: "Swagger", Severity: model.SeverityMedium, ScoreDeduction: 5, ObservedValue: "Exposed", Recommendation: "Protect with auth", Remediation: "deny all;"},
		},
	}

	runner := &mockAuditRunner{report: mockReport}

	// 1. User 1 triggers audit
	auditHandler := RunSecurityAuditHandler(runner, secStore, mStore)
	body := bytes.NewBufferString(`{"target_url":"https://goodai.co.nz"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/security/audit", body)
	req = req.WithContext(context.WithValue(req.Context(), UserIDContextKey, user1.ID))
	rr := httptest.NewRecorder()
	auditHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from audit handler, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var createdReport model.SecurityAuditReport
	if err := json.NewDecoder(rr.Body).Decode(&createdReport); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createdReport.ID <= 0 || createdReport.CompositeScore != 95 {
		t.Errorf("unexpected report output: %+v", createdReport)
	}

	// 2. User 1 lists reports
	listHandler := ListSecurityReportsHandler(secStore)
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/security/reports", nil)
	reqList = reqList.WithContext(context.WithValue(reqList.Context(), UserIDContextKey, user1.ID))
	rrList := httptest.NewRecorder()
	listHandler.ServeHTTP(rrList, reqList)

	if rrList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from list handler, got %d", rrList.Code)
	}

	var listData struct {
		Reports []*model.SecurityAuditReport `json:"reports"`
		Total   int                          `json:"total"`
	}
	_ = json.NewDecoder(rrList.Body).Decode(&listData)
	if listData.Total != 1 || len(listData.Reports) != 1 {
		t.Errorf("expected 1 report in list, got %d", listData.Total)
	}

	// 3. User 2 tries to get User 1's report (isolation check)
	getHandler := GetSecurityReportHandler(secStore)
	reqGetU2 := httptest.NewRequest(http.MethodGet, "/api/v1/security/reports/1", nil)
	reqGetU2.SetPathValue("id", "1")
	reqGetU2 = reqGetU2.WithContext(context.WithValue(reqGetU2.Context(), UserIDContextKey, user2.ID))
	rrGetU2 := httptest.NewRecorder()
	getHandler.ServeHTTP(rrGetU2, reqGetU2)

	if rrGetU2.Code != http.StatusNotFound {
		t.Errorf("expected 404 for tenant isolation check, got %d", rrGetU2.Code)
	}

	// 4. User 1 exports report HTML
	exportHandler := ExportSecurityReportHandler(secStore)
	reqExport := httptest.NewRequest(http.MethodGet, "/api/v1/security/reports/1/export", nil)
	reqExport.SetPathValue("id", "1")
	reqExport = reqExport.WithContext(context.WithValue(reqExport.Context(), UserIDContextKey, user1.ID))
	rrExport := httptest.NewRecorder()
	exportHandler.ServeHTTP(rrExport, reqExport)

	if rrExport.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from export handler, got %d", rrExport.Code)
	}
	if !strings.Contains(rrExport.Header().Get("Content-Type"), "text/html") {
		t.Errorf("expected text/html content type, got %s", rrExport.Header().Get("Content-Type"))
	}
	if !strings.Contains(rrExport.Body.String(), "<!DOCTYPE html>") {
		t.Errorf("expected standalone HTML in export response")
	}
}
