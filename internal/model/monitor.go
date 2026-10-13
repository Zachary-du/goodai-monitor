package model

import "time"

// Monitor status constants
const (
	StatusPending  = "PENDING"
	StatusUp       = "UP"
	StatusDown     = "DOWN"
	StatusDegraded = "DEGRADED"
	StatusPaused   = "PAUSED"
)

// Monitor configuration and quota constants
const (
	// MaxMonitorsPerUser defines the strict per-account active monitor quota.
	MaxMonitorsPerUser = 10

	// DefaultIntervalSeconds defines the default probing interval (60 seconds).
	DefaultIntervalSeconds = 60

	// DefaultTimeoutSeconds defines the default HTTP check timeout (5 seconds).
	DefaultTimeoutSeconds = 5

	// DefaultExpectedStatusCode defines the default expected response status (200 OK).
	DefaultExpectedStatusCode = 200
)

// Monitor represents a website availability monitoring target.
type Monitor struct {
	ID                 int64      `json:"id"`
	UserID             int64      `json:"user_id"`
	Name               string     `json:"name"`
	URL                string     `json:"url"`
	IntervalSeconds    int        `json:"interval_seconds"`
	TimeoutSeconds     int        `json:"timeout_seconds"`
	ExpectedStatusCode int        `json:"expected_status_code"`
	Keyword            string     `json:"keyword,omitempty"`
	Status             string     `json:"status"`
	LastCheckAt        *time.Time `json:"last_check_at,omitempty"`
	LastLatencyMs      int64      `json:"last_latency_ms"`
	UptimePercent      float64    `json:"uptime_percent"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CreateMonitorRequest defines the JSON payload for creating a new monitor.
type CreateMonitorRequest struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ExpectedStatusCode int    `json:"expected_status_code"`
	Keyword            string `json:"keyword"`
}

// UpdateMonitorRequest defines the JSON payload for updating an existing monitor.
type UpdateMonitorRequest struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ExpectedStatusCode int    `json:"expected_status_code"`
	Keyword            string `json:"keyword"`
}
