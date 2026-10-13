package api

import (
	"log"
	"net/http"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/storage"
	"github.com/Zachary-du/goodai-monitor/web"
)

// NewRouter constructs and configures the HTTP router for GoodAI Monitor.
// It registers health check telemetry endpoints, authentication endpoints,
// and serves embedded frontend assets with security headers and auth middleware.
func NewRouter(db *storage.DB) http.Handler {
	mux := http.NewServeMux()
	startTime := time.Now()

	// Health and Telemetry endpoints
	healthHandler := HealthzHandler(db, startTime)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/api/healthz", healthHandler)

	var userStore *storage.UserStore
	if db != nil {
		var err error
		userStore, err = storage.NewUserStore(db)
		if err != nil {
			log.Printf("[Router] Warning: failed to initialize user store: %v", err)
		}
	}

	// Authentication API endpoints
	if userStore != nil {
		mux.HandleFunc("/api/v1/auth/register", RegisterHandler(userStore))
		mux.HandleFunc("/api/v1/auth/login", LoginHandler(userStore))
		mux.HandleFunc("/api/v1/auth/logout", LogoutHandler(userStore))
		mux.HandleFunc("/api/v1/auth/me", RequireAuth(MeHandler(userStore)))
	}

	// Embedded web UI static assets
	fileServer := http.FileServer(http.FS(web.Files))
	mux.Handle("/", fileServer)

	var handler http.Handler = mux
	if userStore != nil {
		handler = AuthMiddleware(userStore)(handler)
	}

	return securityHeadersMiddleware(handler)
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
