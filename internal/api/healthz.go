package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// HealthzResponse defines the JSON schema returned by the /healthz endpoint.
type HealthzResponse struct {
	Status           string `json:"status"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	MemoryRSSBytes   uint64 `json:"memory_rss_bytes"`
	GoHeapAllocBytes uint64 `json:"go_heap_alloc_bytes"`
	GoGoroutineCount int    `json:"go_goroutine_count"`
	Database         string `json:"database"`
	Timestamp        string `json:"timestamp"`
}

// ReadProcessRSS retrieves the resident set size (RSS) in bytes.
// On Linux systems, it extracts the resident page count from /proc/self/statm.
// On other operating systems, it falls back to the Go runtime memory allocation metric.
func ReadProcessRSS() uint64 {
	// Attempt Linux /proc/self/statm first
	data, err := os.ReadFile("/proc/self/statm")
	if err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 2 {
			if pages, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				pageSize := uint64(os.Getpagesize())
				return pages * pageSize
			}
		}
	}

	// Fallback to Go runtime Sys memory metric
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Sys
}

// HealthzHandler returns an http.HandlerFunc that verifies database connectivity
// and emits live process telemetry including uptime and physical RSS memory.
func HealthzHandler(db *storage.DB, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "connected"
		status := "ok"
		statusCode := http.StatusOK

		if db != nil {
			if err := db.PingContext(ctx); err != nil {
				dbStatus = "disconnected"
				status = "degraded"
				statusCode = http.StatusServiceUnavailable
			}
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		resp := HealthzResponse{
			Status:           status,
			UptimeSeconds:    int64(time.Since(startTime).Seconds()),
			MemoryRSSBytes:   ReadProcessRSS(),
			GoHeapAllocBytes: m.HeapAlloc,
			GoGoroutineCount: runtime.NumGoroutine(),
			Database:         dbStatus,
			Timestamp:        time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
