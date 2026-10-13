package alert

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/storage"
)

const (
	// RequiredFailuresForOutage defines N=2 consecutive failures to confirm an incident.
	RequiredFailuresForOutage = 2
	// RequiredSuccessesForRecovery defines M=2 consecutive successes to confirm resolution.
	RequiredSuccessesForRecovery = 2
)

// MonitorState tracks the dynamic anti-flapping state machine counters for a monitor.
type MonitorState struct {
	ConsecutiveFailures   int
	ConsecutiveSuccesses  int
	ActiveIncidentID      int64
	OutageAlertDispatched bool
}

// AlertManager coordinates the incident lifecycle state machine and dispatches alerts.
type AlertManager struct {
	incidentStore *storage.IncidentStore
	userStore     *storage.UserStore
	monitorStore  *storage.MonitorStore
	emailSender   EmailSender

	mu     sync.RWMutex
	states map[int64]*MonitorState
}

// NewAlertManager initializes a new incident and alert lifecycle manager.
func NewAlertManager(
	incidentStore *storage.IncidentStore,
	userStore *storage.UserStore,
	monitorStore *storage.MonitorStore,
	emailSender EmailSender,
) *AlertManager {
	return &AlertManager{
		incidentStore: incidentStore,
		userStore:     userStore,
		monitorStore:  monitorStore,
		emailSender:   emailSender,
		states:        make(map[int64]*MonitorState),
	}
}

// getOrCreateState retrieves the in-memory state or syncs with the database if an open incident exists.
func (am *AlertManager) getOrCreateState(ctx context.Context, monitorID int64) *MonitorState {
	am.mu.Lock()
	defer am.mu.Unlock()

	state, exists := am.states[monitorID]
	if !exists {
		state = &MonitorState{}
		am.states[monitorID] = state

		// Recover open incident from database if server restarted during an outage
		if am.incidentStore != nil {
			if openInc, err := am.incidentStore.GetOpenIncidentByMonitor(ctx, monitorID); err == nil && openInc != nil {
				state.ActiveIncidentID = openInc.ID
				state.ConsecutiveFailures = RequiredFailuresForOutage
				state.OutageAlertDispatched = true
			}
		}
	}

	return state
}

// ProcessCheckResult transitions the monitor state machine according to the latest check telemetry.
func (am *AlertManager) ProcessCheckResult(ctx context.Context, m *model.Monitor, check *model.Check) {
	if m == nil || check == nil {
		return
	}

	// Never process incidents for PAUSED monitors
	if m.Status == model.StatusPaused {
		return
	}

	state := am.getOrCreateState(ctx, m.ID)

	am.mu.Lock()
	defer am.mu.Unlock()

	if check.Status != model.StatusUp {
		// --- FAILURE PATH ---
		state.ConsecutiveSuccesses = 0
		state.ConsecutiveFailures++

		if state.ConsecutiveFailures == 1 {
			// AC-1: Single isolated failure -> DEGRADED state, no incident created, no email
			log.Printf("[INFO] Monitor '%s' (ID: %d) entered DEGRADED state (failure 1/%d): %s",
				m.Name, m.ID, RequiredFailuresForOutage, check.ErrorMessage)
			return
		}

		if state.ConsecutiveFailures == RequiredFailuresForOutage {
			// AC-2: N=2 consecutive failures -> confirm outage, create OPEN incident, dispatch alert
			log.Printf("[ALERT] Monitor '%s' (ID: %d) confirmed DOWN (failures %d/%d). Creating incident.",
				m.Name, m.ID, state.ConsecutiveFailures, RequiredFailuresForOutage)

			var currentInc *model.Incident
			var err error

			if state.ActiveIncidentID == 0 {
				// Create new incident in database
				inc := &model.Incident{
					MonitorID: m.ID,
					UserID:    m.UserID,
					StartedAt: check.CreatedAt,
					Status:    model.IncidentStatusOpen,
					Cause:     check.ErrorMessage,
				}
				if inc.Cause == "" {
					inc.Cause = "Endpoint unreachable or returned error status"
				}

				if am.incidentStore != nil {
					currentInc, err = am.incidentStore.CreateIncident(ctx, inc)
					if err != nil {
						log.Printf("[ERROR] Failed to record open incident for monitor ID %d: %v", m.ID, err)
					} else {
						state.ActiveIncidentID = currentInc.ID
					}
				}
			} else if am.incidentStore != nil {
				currentInc, _ = am.incidentStore.GetIncidentByID(ctx, state.ActiveIncidentID)
			}

			// Dispatch outage email if not already sent for this incident (AC-5)
			if !state.OutageAlertDispatched && am.emailSender != nil && currentInc != nil {
				recipient := am.getUserEmail(ctx, m.UserID)
				if recipient != "" {
					if err := am.emailSender.SendOutageAlert(ctx, recipient, m, currentInc); err != nil {
						log.Printf("[WARN] Failed to dispatch outage email for monitor '%s': %v", m.Name, err)
					} else {
						state.OutageAlertDispatched = true
					}
				}
			}
			return
		}

		// state.ConsecutiveFailures > RequiredFailuresForOutage: Outage ongoing; deduplicate (AC-5)
		return
	}

	// --- SUCCESS PATH (check.Status == StatusUp) ---
	state.ConsecutiveFailures = 0

	if state.ActiveIncidentID > 0 {
		state.ConsecutiveSuccesses++

		if state.ConsecutiveSuccesses < RequiredSuccessesForRecovery {
			// AC-4: M=1 success -> still recovering, incident remains OPEN
			log.Printf("[INFO] Monitor '%s' (ID: %d) recovering (success 1/%d). Incident #%d remains OPEN.",
				m.Name, m.ID, RequiredSuccessesForRecovery, state.ActiveIncidentID)
			return
		}

		// AC-4: M=2 consecutive successes -> confirmed recovery!
		log.Printf("[RESOLVED] Monitor '%s' (ID: %d) confirmed RECOVERED (successes %d/%d). Resolving incident #%d.",
			m.Name, m.ID, state.ConsecutiveSuccesses, RequiredSuccessesForRecovery, state.ActiveIncidentID)

		var resolvedInc *model.Incident
		if am.incidentStore != nil {
			inc, err := am.incidentStore.GetIncidentByID(ctx, state.ActiveIncidentID)
			if err == nil && inc != nil {
				resolvedAt := check.CreatedAt
				if resolvedAt.IsZero() {
					resolvedAt = time.Now().UTC()
				}
				durationSec := int64(resolvedAt.Sub(inc.StartedAt).Seconds())
				if durationSec < 0 {
					durationSec = 0
				}

				if err := am.incidentStore.ResolveIncident(ctx, inc.ID, resolvedAt, durationSec); err != nil {
					log.Printf("[ERROR] Failed to resolve incident #%d: %v", inc.ID, err)
				} else {
					inc.Status = model.IncidentStatusResolved
					inc.ResolvedAt = &resolvedAt
					inc.DurationSeconds = durationSec
					resolvedInc = inc
				}
			}
		}

		// Dispatch recovery email (AC-4)
		if am.emailSender != nil && resolvedInc != nil {
			recipient := am.getUserEmail(ctx, m.UserID)
			if recipient != "" {
				if err := am.emailSender.SendRecoveryAlert(ctx, recipient, m, resolvedInc); err != nil {
					log.Printf("[WARN] Failed to dispatch recovery email for monitor '%s': %v", m.Name, err)
				}
			}
		}

		// Reset state counters after complete recovery
		state.ActiveIncidentID = 0
		state.OutageAlertDispatched = false
		state.ConsecutiveSuccesses = 0
	}
}

// getUserEmail retrieves the destination email address for a given user ID.
func (am *AlertManager) getUserEmail(ctx context.Context, userID int64) string {
	if am.userStore == nil {
		return ""
	}
	user, err := am.userStore.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return ""
	}
	return user.Email
}

// ResetMonitorState removes tracking state for a monitor when paused or deleted.
func (am *AlertManager) ResetMonitorState(monitorID int64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.states, monitorID)
}

// GetMonitorState returns a copy of the state for testing or inspection.
func (am *AlertManager) GetMonitorState(monitorID int64) (MonitorState, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	state, exists := am.states[monitorID]
	if !exists {
		return MonitorState{}, false
	}
	return *state, true
}
