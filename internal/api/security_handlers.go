package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/security"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// RunSecurityAuditHandler triggers a multi-vector security & AI defense inspection.
// POST /api/v1/security/audit
func RunSecurityAuditHandler(auditRunner security.AuditRunner, secStore *storage.SecurityStore, monStore *storage.MonitorStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		var req model.AuditRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		targetURL := strings.TrimSpace(req.TargetURL)
		if targetURL == "" && req.MonitorID > 0 {
			if monStore == nil {
				writeJSONError(w, http.StatusBadRequest, "Monitor store not configured")
				return
			}
			m, err := monStore.GetMonitorByIDAndUser(r.Context(), req.MonitorID, userID)
			if err != nil {
				if errors.Is(err, storage.ErrMonitorNotFound) {
					writeJSONError(w, http.StatusNotFound, "Monitor not found")
					return
				}
				writeJSONError(w, http.StatusInternalServerError, "Failed to resolve monitor target")
				return
			}
			targetURL = m.URL
		}

		if targetURL == "" {
			writeJSONError(w, http.StatusBadRequest, "Target URL or monitor ID is required")
			return
		}

		// Pre-flight SSRF check
		if err := security.ValidateTargetURL(targetURL); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid or restricted target URL")
			return
		}

		if auditRunner == nil {
			auditRunner = security.NewAuditEngine()
		}

		report, err := auditRunner.RunAudit(r.Context(), targetURL, userID)
		if err != nil {
			if errors.Is(err, security.ErrRestrictedTargetURL) {
				writeJSONError(w, http.StatusBadRequest, "Invalid or restricted target URL")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to run security audit: %v", err))
			return
		}

		if secStore != nil {
			if err := secStore.SaveReport(r.Context(), report); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to persist security audit report")
				return
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(report)
	}
}

// ListSecurityReportsHandler lists previous security audit reports for the authenticated user.
// GET /api/v1/security/reports
func ListSecurityReportsHandler(secStore *storage.SecurityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		limit := 20
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		reports, err := secStore.ListReports(r.Context(), userID, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query security reports")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"reports": reports,
			"total":   len(reports),
		})
	}
}

// GetSecurityReportHandler retrieves a single security audit report JSON by ID.
// GET /api/v1/security/reports/{id}
func GetSecurityReportHandler(secStore *storage.SecurityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		id, err := parseSecurityReportID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid security report ID")
			return
		}

		report, err := secStore.GetReport(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, storage.ErrSecurityReportNotFound) {
				writeJSONError(w, http.StatusNotFound, "Security report not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve security report")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(report)
	}
}

// ExportSecurityReportHandler renders and streams a standalone executive HTML report.
// GET /api/v1/security/reports/{id}/export
func ExportSecurityReportHandler(secStore *storage.SecurityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		id, err := parseSecurityReportID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid security report ID")
			return
		}

		report, err := secStore.GetReport(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, storage.ErrSecurityReportNotFound) {
				writeJSONError(w, http.StatusNotFound, "Security report not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve security report")
			return
		}

		htmlContent, err := security.GenerateExecutiveHTMLReport(report)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate HTML report")
			return
		}

		safeHost := strings.ReplaceAll(report.TargetHost, ":", "_")
		safeHost = strings.ReplaceAll(safeHost, "/", "_")
		dateStr := report.CreatedAt.Format("20060102")
		filename := fmt.Sprintf("GoodAI-Security-Report-%s-%s.html", safeHost, dateStr)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(htmlContent))
	}
}

// SecurityReportsDispatcher handles requests to /api/v1/security/reports/:id and /export.
func SecurityReportsDispatcher(secStore *storage.SecurityStore) http.HandlerFunc {
	getHandler := GetSecurityReportHandler(secStore)
	exportHandler := ExportSecurityReportHandler(secStore)

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/export") {
			exportHandler(w, r)
			return
		}
		getHandler(w, r)
	}
}

func parseSecurityReportID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/security/reports/")
		path = strings.TrimSuffix(path, "/export")
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			idStr = parts[0]
		}
	}
	return strconv.ParseInt(idStr, 10, 64)
}
