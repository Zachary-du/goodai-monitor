package model

import "time"

const (
	IncidentStatusOpen     = "OPEN"
	IncidentStatusResolved = "RESOLVED"
)

// Incident represents a confirmed downtime incident for a monitor.
type Incident struct {
	ID              int64      `json:"id"`
	MonitorID       int64      `json:"monitor_id"`
	UserID          int64      `json:"user_id"`
	StartedAt       time.Time  `json:"started_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	Status          string     `json:"status"` // OPEN or RESOLVED
	Cause           string     `json:"cause"`
	DurationSeconds int64      `json:"duration_seconds,omitempty"`

	// Joined fields for display
	MonitorName string `json:"monitor_name,omitempty"`
	MonitorURL  string `json:"monitor_url,omitempty"`
}

// IncidentSummary provides high-level count of open vs resolved incidents.
type IncidentSummary struct {
	OpenCount     int `json:"open_count"`
	ResolvedCount int `json:"resolved_count"`
	TotalCount    int `json:"total_count"`
}
