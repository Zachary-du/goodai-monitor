package prober

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

// monitorJob holds the state and cancellation handle for an actively scheduled monitor.
type monitorJob struct {
	monitor *model.Monitor
	cancel  context.CancelFunc
	done    chan struct{}
}

// Scheduler continuously schedules and executes synthetic probes for all active monitors,
// adapts dynamically to monitor lifecycle events (add, update, pause, resume, delete),
// and periodically executes the 14-day rolling retention pruning job.
type Scheduler struct {
	store     *storage.CheckStore
	prober    Prober
	mu        sync.RWMutex
	jobs      map[int64]*monitorJob
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
}

// NewScheduler initializes a new dynamic probe Scheduler.
// If customProber is nil, the default SSRF-defended Client is used.
func NewScheduler(store *storage.CheckStore, customProber Prober) *Scheduler {
	if customProber == nil {
		customProber = NewClient()
	}

	return &Scheduler{
		store:  store,
		prober: customProber,
		jobs:   make(map[int64]*monitorJob),
	}
}

// Start boots the background scheduling loops, loads all active monitors, and initiates the 24h retention pruner.
func (s *Scheduler) Start(parentCtx context.Context) {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.ctx, s.cancel = context.WithCancel(parentCtx)
	s.isRunning = true
	s.mu.Unlock()

	log.Println("[INFO] Background probing scheduler started.")

	// Initial sync of all active monitors from database
	s.SyncMonitors(s.ctx)

	// Background ticker for 24-hour rolling retention pruning (purges checks older than 14 days)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Run immediate prune on startup
		if pruned, err := s.store.PruneOldChecks(s.ctx, model.DefaultRetentionDays*24*time.Hour); err == nil && pruned > 0 {
			log.Printf("[INFO] Initial retention prune removed %d check records older than %d days.", pruned, model.DefaultRetentionDays)
		}

		retentionTicker := time.NewTicker(24 * time.Hour)
		defer retentionTicker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-retentionTicker.C:
				pruned, err := s.store.PruneOldChecks(s.ctx, model.DefaultRetentionDays*24*time.Hour)
				if err != nil {
					log.Printf("[ERROR] Retention prune failed: %v", err)
				} else if pruned > 0 {
					log.Printf("[INFO] 24h retention prune purged %d check records older than %d days.", pruned, model.DefaultRetentionDays)
				}
			}
		}
	}()

	// Periodic database sync loop (detects newly created, paused, resumed, or deleted monitors)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		syncTicker := time.NewTicker(5 * time.Second)
		defer syncTicker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-syncTicker.C:
				s.SyncMonitors(s.ctx)
			}
		}
	}()
}

// Stop cleanly drains and cancels all active probe goroutines and background tickers.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = false
	s.cancel()

	// Cancel all active jobs
	for id, job := range s.jobs {
		job.cancel()
		delete(s.jobs, id)
	}
	s.mu.Unlock()

	s.wg.Wait()
	log.Println("[INFO] Background probing scheduler stopped cleanly.")
}

// AddOrUpdateMonitor schedules or adjusts an individual monitor's probing routine.
// If the monitor is paused, any active probing loop is immediately terminated.
func (s *Scheduler) AddOrUpdateMonitor(m *model.Monitor) {
	if m == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	existingJob, exists := s.jobs[m.ID]

	// Handle PAUSED monitors
	if m.Status == model.StatusPaused {
		if exists {
			existingJob.cancel()
			delete(s.jobs, m.ID)
		}
		return
	}

	// If already running with identical schedule and configuration, no restart needed
	if exists {
		if existingJob.monitor.IntervalSeconds == m.IntervalSeconds &&
			existingJob.monitor.URL == m.URL &&
			existingJob.monitor.TimeoutSeconds == m.TimeoutSeconds &&
			existingJob.monitor.ExpectedStatusCode == m.ExpectedStatusCode &&
			existingJob.monitor.Keyword == m.Keyword {
			existingJob.monitor = m
			return
		}
		// Cancel old job to replace with updated settings
		existingJob.cancel()
		delete(s.jobs, m.ID)
	}

	// Schedule new probe job
	jobCtx, jobCancel := context.WithCancel(s.ctx)
	job := &monitorJob{
		monitor: m,
		cancel:  jobCancel,
		done:    make(chan struct{}),
	}
	s.jobs[m.ID] = job

	s.wg.Add(1)
	go s.runMonitorJob(jobCtx, job)
}

// RemoveMonitor halts and unregisters probing for the specified monitor ID.
func (s *Scheduler) RemoveMonitor(monitorID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.jobs[monitorID]; exists {
		job.cancel()
		delete(s.jobs, monitorID)
	}
}

// SyncMonitors reconciles the currently active probe jobs with monitors in the SQLite database.
func (s *Scheduler) SyncMonitors(ctx context.Context) {
	allMonitors, err := s.store.GetAllMonitors(ctx)
	if err != nil {
		return
	}

	s.mu.Lock()
	dbMonitorMap := make(map[int64]*model.Monitor, len(allMonitors))
	for _, m := range allMonitors {
		dbMonitorMap[m.ID] = m
	}

	// Remove running jobs that no longer exist in the database or are PAUSED
	for id, job := range s.jobs {
		dbMonitor, found := dbMonitorMap[id]
		if !found || dbMonitor.Status == model.StatusPaused {
			job.cancel()
			delete(s.jobs, id)
		}
	}
	s.mu.Unlock()

	// Add or update active monitors
	for _, m := range allMonitors {
		if m.Status != model.StatusPaused {
			s.AddOrUpdateMonitor(m)
		}
	}
}

// runMonitorJob executes the probe loop for a single monitor.
func (s *Scheduler) runMonitorJob(ctx context.Context, job *monitorJob) {
	defer s.wg.Done()
	defer close(job.done)

	// Execute initial probe immediately upon scheduling
	s.executeProbe(ctx, job.monitor)

	intervalSec := job.monitor.IntervalSeconds
	if intervalSec < 1 {
		intervalSec = model.DefaultIntervalSeconds
	}
	interval := time.Duration(intervalSec) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.executeProbe(ctx, job.monitor)
		}
	}
}

// executeProbe performs a single probe check and persists the telemetry record in the CheckStore.
func (s *Scheduler) executeProbe(ctx context.Context, m *model.Monitor) {
	// Guard against probing closed or cancelled context
	if ctx.Err() != nil {
		return
	}

	result := s.prober.ProbeTarget(ctx, m)
	if result == nil {
		return
	}

	check := &model.Check{
		MonitorID:      m.ID,
		Status:         result.Status,
		StatusCode:     result.StatusCode,
		LatencyMs:      result.TotalLatencyMs,
		DNSLookupMs:    result.LatencyBreakdown.DNSLookupMs,
		TCPConnectMs:   result.LatencyBreakdown.TCPConnectMs,
		TLSHandshakeMs: result.LatencyBreakdown.TLSHandshakeMs,
		TTFBMs:         result.LatencyBreakdown.TTFBMs,
		ErrorMessage:   result.ErrorMessage,
		CertExpiryDays: result.CertExpiryDays,
		CreatedAt:      time.Now().UTC(),
	}

	if _, err := s.store.RecordCheck(ctx, check); err != nil {
		log.Printf("[WARN] Failed to record check for monitor ID %d: %v", m.ID, err)
	}
}

// ActiveJobCount returns the number of currently scheduled active monitor routines (useful for tests).
func (s *Scheduler) ActiveJobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs)
}
