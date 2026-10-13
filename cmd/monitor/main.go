package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/api"
	"github.com/Zachary-du/goodai-monitor/internal/config"
	"github.com/Zachary-du/goodai-monitor/internal/prober"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

func main() {
	log.Println("[INFO] Initializing GoodAI Monitor daemon...")

	// 1. Load runtime configuration
	cfg := config.Load()
	log.Printf("[INFO] Configuration loaded: port=%s, storage=%s, tz=%s", cfg.Port, cfg.StoragePath, cfg.Timezone)

	// 2. Initialize pure-Go SQLite storage engine
	db, err := storage.Open(cfg.StoragePath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize database at '%s': %v", cfg.StoragePath, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Failed to close database: %v", err)
		}
	}()
	log.Printf("[INFO] SQLite database initialized in WAL mode at '%s'", cfg.StoragePath)

	// 3. Initialize CheckStore and start continuous background probing engine
	checkStore, err := storage.NewCheckStore(db)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize check store: %v", err)
	}

	scheduler := prober.NewScheduler(checkStore, nil)
	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	scheduler.Start(schedulerCtx)
	log.Println("[INFO] Prober background scheduler and retention engine initialized.")

	// 4. Initialize HTTP router and server
	router := api.NewRouter(db)
	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 5. Start HTTP listener in background goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[INFO] GoodAI Monitor HTTP server listening on http://%s", cfg.Address())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 6. Intercept termination signals (SIGINT, SIGTERM)
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("[FATAL] HTTP server listener encountered fatal error: %v", err)

	case sig := <-shutdownSignal:
		log.Printf("[INFO] Received system signal '%s'. Initiating graceful shutdown (10s window)...", sig)

		// 7. Stop background probe scheduler routines first
		scheduler.Stop()
		log.Println("[INFO] Background probe scheduler stopped cleanly.")

		// 8. Graceful shutdown window: 10 seconds context
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[WARN] HTTP server forced shutdown error: %v", err)
			_ = server.Close()
		} else {
			log.Println("[INFO] HTTP server active listeners drained successfully.")
		}

		// 9. Flush and close database
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Database flush and close failed: %v", err)
		} else {
			log.Println("[INFO] SQLite connection pool closed cleanly.")
		}

		log.Println("[INFO] GoodAI Monitor process terminated gracefully.")
	}
}
