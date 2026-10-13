package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// MonitorMetricsHandler handles retrieving SLA uptime stats and 24-hour latency samples.
// GET /api/v1/monitors/:id/metrics
func MonitorMetricsHandler(monitorStore *storage.MonitorStore, checkStore *storage.CheckStore) http.HandlerFunc {
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

		monitorID, err := parseMetricsMonitorID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid monitor ID")
			return
		}

		// Enforce strict multi-tenant ownership
		_, err = monitorStore.GetMonitorByIDAndUser(r.Context(), monitorID, userID)
		if err != nil {
			if errors.Is(err, storage.ErrMonitorNotFound) {
				writeJSONError(w, http.StatusNotFound, "Monitor not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to verify monitor ownership")
			return
		}

		stats, err := checkStore.CalculateUptimeStats(r.Context(), monitorID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to calculate monitor telemetry metrics")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(stats)
	}
}

// MonitorChecksHandler handles retrieving the recent 50 check execution logs for a target.
// GET /api/v1/monitors/:id/checks
func MonitorChecksHandler(monitorStore *storage.MonitorStore, checkStore *storage.CheckStore) http.HandlerFunc {
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

		monitorID, err := parseMetricsMonitorID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid monitor ID")
			return
		}

		// Enforce strict multi-tenant ownership
		_, err = monitorStore.GetMonitorByIDAndUser(r.Context(), monitorID, userID)
		if err != nil {
			if errors.Is(err, storage.ErrMonitorNotFound) {
				writeJSONError(w, http.StatusNotFound, "Monitor not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to verify monitor ownership")
			return
		}

		limit := 50
		if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		checks, err := checkStore.GetRecentChecksByMonitor(r.Context(), monitorID, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve recent check records")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"checks": checks,
			"total":  len(checks),
		})
	}
}

func parseMetricsMonitorID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		path := r.URL.Path
		path = strings.TrimPrefix(path, "/api/v1/monitors/")
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			idStr = parts[0]
		}
	}
	return strconv.ParseInt(idStr, 10, 64)
}
