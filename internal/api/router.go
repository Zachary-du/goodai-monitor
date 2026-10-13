package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/storage"
	"github.com/Zachary-du/goodai-monitor/web"
)

// NewRouter constructs and configures the HTTP router for GoodAI Monitor.
// It registers health check telemetry endpoints, authentication endpoints,
// monitor management and telemetry endpoints, and serves embedded frontend assets.
func NewRouter(db *storage.DB) http.Handler {
	mux := http.NewServeMux()
	startTime := time.Now()

	// Health and Telemetry endpoints
	healthHandler := HealthzHandler(db, startTime)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/api/healthz", healthHandler)

	var userStore *storage.UserStore
	var monitorStore *storage.MonitorStore
	var checkStore *storage.CheckStore
	var incidentStore *storage.IncidentStore

	if db != nil {
		var err error
		userStore, err = storage.NewUserStore(db)
		if err != nil {
			log.Printf("[Router] Warning: failed to initialize user store: %v", err)
		}
		monitorStore, err = storage.NewMonitorStore(db)
		if err != nil {
			log.Printf("[Router] Warning: failed to initialize monitor store: %v", err)
		}
		checkStore, err = storage.NewCheckStore(db)
		if err != nil {
			log.Printf("[Router] Warning: failed to initialize check store: %v", err)
		}
		incidentStore, err = storage.NewIncidentStore(db)
		if err != nil {
			log.Printf("[Router] Warning: failed to initialize incident store: %v", err)
		}
	}

	// Authentication API endpoints
	if userStore != nil {
		mux.HandleFunc("/api/v1/auth/register", RegisterHandler(userStore))
		mux.HandleFunc("/api/v1/auth/login", LoginHandler(userStore))
		mux.HandleFunc("/api/v1/auth/logout", LogoutHandler(userStore))
		mux.HandleFunc("/api/v1/auth/me", RequireAuth(MeHandler(userStore)))
	}

	// Monitor Management & Telemetry API endpoints (protected by authentication)
	if monitorStore != nil {
		mux.HandleFunc("/api/v1/monitors", RequireAuth(MonitorsCollectionHandler(monitorStore)))

		if checkStore != nil {
			mux.HandleFunc("/api/v1/monitors/{id}/metrics", RequireAuth(MonitorMetricsHandler(monitorStore, checkStore)))
			mux.HandleFunc("/api/v1/monitors/{id}/checks", RequireAuth(MonitorChecksHandler(monitorStore, checkStore)))
		}

		mux.HandleFunc("/api/v1/monitors/", RequireAuth(MonitorItemDispatcherWithTelemetry(monitorStore, checkStore)))
	}

	// Incident Hub API endpoints (protected by authentication)
	if incidentStore != nil {
		mux.HandleFunc("/api/v1/incidents", RequireAuth(ListIncidentsHandler(incidentStore)))
		mux.HandleFunc("/api/v1/incidents/summary", RequireAuth(IncidentSummaryHandler(incidentStore)))
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

// MonitorItemDispatcherWithTelemetry routes monitor item actions including metrics, checks, pause, and CRUD.
func MonitorItemDispatcherWithTelemetry(store *storage.MonitorStore, checkStore *storage.CheckStore) http.HandlerFunc {
	getHandler := GetMonitorHandler(store)
	putHandler := UpdateMonitorHandler(store)
	deleteHandler := DeleteMonitorHandler(store)
	pauseHandler := TogglePauseMonitorHandler(store)

	var metricsHandler, checksHandler http.HandlerFunc
	if checkStore != nil {
		metricsHandler = MonitorMetricsHandler(store, checkStore)
		checksHandler = MonitorChecksHandler(store, checkStore)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasSuffix(path, "/pause") {
			pauseHandler(w, r)
			return
		}
		if strings.HasSuffix(path, "/metrics") && metricsHandler != nil {
			metricsHandler(w, r)
			return
		}
		if strings.HasSuffix(path, "/checks") && checksHandler != nil {
			checksHandler(w, r)
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

// securityHeadersMiddleware injects essential HTTP security headers into all responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
