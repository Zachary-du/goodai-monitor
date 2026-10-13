package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// ListIncidentsHandler handles GET /api/v1/incidents
// Returns paginated downtime incident records for the authenticated tenant with optional status filtering.
func ListIncidentsHandler(incidentStore *storage.IncidentStore) http.HandlerFunc {
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

		query := r.URL.Query()
		statusFilter := query.Get("status")
		limit := 50
		if lStr := query.Get("limit"); lStr != "" {
			if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		offset := 0
		if oStr := query.Get("offset"); oStr != "" {
			if parsed, err := strconv.Atoi(oStr); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		incidents, err := incidentStore.ListIncidentsByUser(r.Context(), userID, statusFilter, limit, offset)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query incidents")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"incidents": incidents,
			"count":     len(incidents),
		})
	}
}

// IncidentSummaryHandler handles GET /api/v1/incidents/summary
// Returns aggregate open, resolved, and total incident counts for the authenticated tenant.
func IncidentSummaryHandler(incidentStore *storage.IncidentStore) http.HandlerFunc {
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

		summary, err := incidentStore.GetIncidentSummaryByUser(r.Context(), userID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to calculate incident summary")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(summary)
	}
}
