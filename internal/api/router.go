package api

import (
	"net/http"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/storage"
	"github.com/Zachary-du/goodai-monitor/web"
)

// NewRouter constructs and configures the HTTP router for GoodAI Monitor.
// It registers health check telemetry endpoints and serves embedded frontend assets.
func NewRouter(db *storage.DB) http.Handler {
	mux := http.NewServeMux()
	startTime := time.Now()

	// Health and Telemetry endpoints
	healthHandler := HealthzHandler(db, startTime)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/api/healthz", healthHandler)

	// Embedded web UI static assets
	fileServer := http.FileServer(http.FS(web.Files))
	mux.Handle("/", fileServer)

	return securityHeadersMiddleware(mux)
}

// securityHeadersMiddleware injects essential HTTP security headers into all responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
