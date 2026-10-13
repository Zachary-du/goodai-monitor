package model

import "time"

// Rolling retention constants
const (
	// DefaultRetentionDays defines the rolling retention threshold for check records.
	DefaultRetentionDays = 14
)

// Check represents an individual probe execution record stored in SQLite.
type Check struct {
	ID             int64     `json:"id"`
	MonitorID      int64     `json:"monitor_id"`
	Status         string    `json:"status"` // UP, DOWN, DEGRADED
	StatusCode     int       `json:"status_code"`
	LatencyMs      int64     `json:"latency_ms"`
	DNSLookupMs    int64     `json:"dns_lookup_ms"`
	TCPConnectMs   int64     `json:"tcp_connect_ms"`
	TLSHandshakeMs int64     `json:"tls_handshake_ms"`
	TTFBMs         int64     `json:"ttfb_ms"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CertExpiryDays *int      `json:"cert_expiry_days,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// LatencyBreakdown details connection and transmission latency measurements.
type LatencyBreakdown struct {
	DNSLookupMs    int64 `json:"dns_lookup_ms"`
	TCPConnectMs   int64 `json:"tcp_connect_ms"`
	TLSHandshakeMs int64 `json:"tls_handshake_ms"`
	TTFBMs         int64 `json:"ttfb_ms"`
	TotalMs        int64 `json:"total_ms"`
}

// CheckResult represents the outcome of a synthetic HTTP/HTTPS probe execution.
type CheckResult struct {
	Success          bool             `json:"success"`
	StatusCode       int              `json:"status_code"`
	Status           string           `json:"status"`
	LatencyBreakdown LatencyBreakdown `json:"latency_breakdown"`
	TotalLatencyMs   int64            `json:"total_latency_ms"`
	ErrorMessage     string           `json:"error_message,omitempty"`
	CertExpiryDays   *int             `json:"cert_expiry_days,omitempty"`
}

// LatencyPoint represents a single historical telemetry point for charting.
type LatencyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	LatencyMs int64     `json:"latency_ms"`
	Status    string    `json:"status"`
}

// MetricsSummary encapsulates SLA uptime percentages and recent telemetry points.
type MetricsSummary struct {
	MonitorID       int64          `json:"monitor_id"`
	Uptime24h       float64        `json:"uptime_24h"`
	Uptime7d        float64        `json:"uptime_7d"`
	Uptime30d       float64        `json:"uptime_30d"`
	AvgLatency24h   float64        `json:"avg_latency_24h"`
	TotalChecks24h  int64          `json:"total_checks_24h"`
	LatencyHistory  []LatencyPoint `json:"latency_history"`
}
