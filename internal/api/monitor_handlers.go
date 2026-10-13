package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/security"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// ListMonitorsHandler handles retrieving all monitors for the authenticated user.
// GET /api/v1/monitors
func ListMonitorsHandler(store *storage.MonitorStore) http.HandlerFunc {
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

		monitors, err := store.ListMonitorsByUser(r.Context(), userID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve monitors")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"monitors": monitors,
			"total":    len(monitors),
		})
	}
}

// CreateMonitorHandler handles creating a new monitor target with SSRF checks and quota enforcement.
// POST /api/v1/monitors
func CreateMonitorHandler(store *storage.MonitorStore) http.HandlerFunc {
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

		// Enforce per-user quota (max 10 monitors)
		count, err := store.CountMonitorsByUser(r.Context(), userID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to check monitor quota")
			return
		}
		if count >= model.MaxMonitorsPerUser {
			writeJSONError(w, http.StatusForbidden, "Monitor quota exceeded (maximum 10 monitors allowed)")
			return
		}

		var req model.CreateMonitorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		cleanName := strings.TrimSpace(req.Name)
		if cleanName == "" {
			writeJSONError(w, http.StatusBadRequest, "Monitor name is required")
			return
		}
		if len(cleanName) > 100 {
			writeJSONError(w, http.StatusBadRequest, "Monitor name exceeds maximum length of 100 characters")
			return
		}

		cleanURL := strings.TrimSpace(req.URL)
		if cleanURL == "" {
			writeJSONError(w, http.StatusBadRequest, "Monitor target URL is required")
			return
		}

		// Validate URL against SSRF and port whitelisting
		if err := security.ValidateTargetURL(cleanURL); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid or restricted target URL")
			return
		}

		interval := req.IntervalSeconds
		if interval <= 0 {
			interval = model.DefaultIntervalSeconds
		} else if interval < 10 {
			writeJSONError(w, http.StatusBadRequest, "Interval must be at least 10 seconds")
			return
		}

		timeout := req.TimeoutSeconds
		if timeout <= 0 {
			timeout = model.DefaultTimeoutSeconds
		} else if timeout < 1 || timeout > 30 {
			writeJSONError(w, http.StatusBadRequest, "Timeout must be between 1 and 30 seconds")
			return
		}

		statusCode := req.ExpectedStatusCode
		if statusCode <= 0 {
			statusCode = model.DefaultExpectedStatusCode
		} else if statusCode < 100 || statusCode > 599 {
			writeJSONError(w, http.StatusBadRequest, "Expected status code must be a valid HTTP status (100-599)")
			return
		}

		cleanKeyword := strings.TrimSpace(req.Keyword)
		if len(cleanKeyword) > 100 {
			writeJSONError(w, http.StatusBadRequest, "Keyword assertion exceeds maximum length of 100 characters")
			return
		}

		monitor := &model.Monitor{
			UserID:             userID,
			Name:               cleanName,
			URL:                cleanURL,
			IntervalSeconds:    interval,
			TimeoutSeconds:     timeout,
			ExpectedStatusCode: statusCode,
			Keyword:            cleanKeyword,
			Status:             model.StatusPending,
		}

		created, err := store.CreateMonitor(r.Context(), monitor)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to create monitor")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Monitor created successfully",
			"monitor": created,
		})
	}
}

// GetMonitorHandler handles fetching a single monitor by ID for the authenticated owner.
// GET /api/v1/monitors/:id
func GetMonitorHandler(store *storage.MonitorStore) http.HandlerFunc {
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

		id, err := parseMonitorID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid monitor ID")
			return
		}

		monitor, err := store.GetMonitorByIDAndUser(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, storage.ErrMonitorNotFound) {
				writeJSONError(w, http.StatusNotFound, "Monitor not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve monitor")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"monitor": monitor,
		})
	}
}

// UpdateMonitorHandler handles modifying an existing monitor's configuration.
// PUT /api/v1/monitors/:id
func UpdateMonitorHandler(store *storage.MonitorStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		id, err := parseMonitorID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid monitor ID")
			return
		}

		var req model.UpdateMonitorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		cleanName := strings.TrimSpace(req.Name)
		if cleanName == "" {
			writeJSONError(w, http.StatusBadRequest, "Monitor name is required")
			return
		}
		if len(cleanName) > 100 {
			writeJSONError(w, http.StatusBadRequest, "Monitor name exceeds maximum length of 100 characters")
			return
		}

		cleanURL := strings.TrimSpace(req.URL)
		if cleanURL == "" {
			writeJSONError(w, http.StatusBadRequest, "Monitor target URL is required")
			return
		}

		// Re-validate target URL on update
		if err := security.ValidateTargetURL(cleanURL); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid or restricted target URL")
			return
		}

		interval := req.IntervalSeconds
		if interval <= 0 {
			interval = model.DefaultIntervalSeconds
		} else if interval < 10 {
			writeJSONError(w, http.StatusBadRequest, "Interval must be at least 10 seconds")
			return
		}

		timeout := req.TimeoutSeconds
		if timeout <= 0 {
			timeout = model.DefaultTimeoutSeconds
		} else if timeout < 1 || timeout > 30 {
			writeJSONError(w, http.StatusBadRequest, "Timeout must be between 1 and 30 seconds")
			return
		}

		statusCode := req.ExpectedStatusCode
		if statusCode <= 0 {
			statusCode = model.DefaultExpectedStatusCode
		} else if statusCode < 100 || statusCode > 599 {
			writeJSONError(w, http.StatusBadRequest, "Expected status code must be a valid HTTP status (100-599)")
			return
		}

		cleanKeyword := strings.TrimSpace(req.Keyword)
		if len(cleanKeyword) > 100 {
			writeJSONError(w, http.StatusBadRequest, "Keyword assertion exceeds maximum length of 100 characters")
			return
		}

		monitor := &model.Monitor{
			ID:                 id,
			UserID:             userID,
			Name:               cleanName,
			URL:                cleanURL,
			IntervalSeconds:    interval,
			TimeoutSeconds:     timeout,
			ExpectedStatusCode: statusCode,
			Keyword:            cleanKeyword,
		}

		updated, err := store.UpdateMonitor(r.Context(), userID, monitor)
		if err != nil {
			if errors.Is(err, storage.ErrMonitorNotFound) {
				writeJSONError(w, http.StatusNotFound, "Monitor not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to update monitor")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Monitor updated successfully",
			"monitor": updated,
		})
	}
}

// DeleteMonitorHandler handles deleting a monitor by ID for the authenticated owner.
// DELETE /api/v1/monitors/:id
func DeleteMonitorHandler(store *storage.MonitorStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		id, err := parseMonitorID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid monitor ID")
			return
		}

		if err := store.DeleteMonitor(r.Context(), id, userID); err != nil {
			if errors.Is(err, storage.ErrMonitorNotFound) {
				writeJSONError(w, http.StatusNotFound, "Monitor not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete monitor")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Monitor deleted successfully",
		})
	}
}

// TogglePauseMonitorHandler handles toggling the pause/resume state of a monitor.
// POST /api/v1/monitors/:id/pause
func TogglePauseMonitorHandler(store *storage.MonitorStore) http.HandlerFunc {
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

		id, err := parseMonitorID(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid monitor ID")
			return
		}

		toggled, err := store.ToggleMonitorPause(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, storage.ErrMonitorNotFound) {
				writeJSONError(w, http.StatusNotFound, "Monitor not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to toggle monitor pause state")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Monitor status updated",
			"monitor": toggled,
		})
	}
}

// MonitorsCollectionHandler dispatches requests to /api/v1/monitors based on HTTP method.
func MonitorsCollectionHandler(store *storage.MonitorStore) http.HandlerFunc {
	listHandler := ListMonitorsHandler(store)
	createHandler := CreateMonitorHandler(store)

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listHandler(w, r)
		case http.MethodPost:
			createHandler(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

// MonitorItemDispatcher dispatches requests to /api/v1/monitors/:id and /api/v1/monitors/:id/pause.
func MonitorItemDispatcher(store *storage.MonitorStore) http.HandlerFunc {
	getHandler := GetMonitorHandler(store)
	putHandler := UpdateMonitorHandler(store)
	deleteHandler := DeleteMonitorHandler(store)
	pauseHandler := TogglePauseMonitorHandler(store)

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/pause") {
			pauseHandler(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			getHandler(w, r)
		case http.MethodPut:
			putHandler(w, r)
		case http.MethodDelete:
			deleteHandler(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

func parseMonitorID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/monitors/")
		path = strings.TrimSuffix(path, "/pause")
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			idStr = parts[0]
		}
	}
	return strconv.ParseInt(idStr, 10, 64)
}
