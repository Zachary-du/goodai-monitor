package model

import "time"

// SeverityLevel represents the severity of a security finding.
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "CRITICAL"
	SeverityHigh     SeverityLevel = "HIGH"
	SeverityMedium   SeverityLevel = "MEDIUM"
	SeverityLow      SeverityLevel = "LOW"
	SeverityPass     SeverityLevel = "PASS"
)

// AuditDomain represents one of the four inspection pillars.
type AuditDomain string

const (
	DomainAIGovernance AuditDomain = "ai_governance"
	DomainHeaders      AuditDomain = "headers"
	DomainTLS          AuditDomain = "tls"
	DomainExposure     AuditDomain = "exposure"
)

// FindingItem describes an individual check observation and remediation guidance.
type FindingItem struct {
	ID             string        `json:"id"`
	Domain         string        `json:"domain"`          // "ai_governance", "headers", "tls", "exposure"
	CheckName      string        `json:"check_name"`
	Severity       SeverityLevel `json:"severity"`
	ScoreDeduction int           `json:"score_deduction"`
	ObservedValue  string        `json:"observed_value"`
	Recommendation string        `json:"recommendation"`
	Remediation    string        `json:"remediation"`     // Actionable code snippet
}

// AuditFinding is an alias for FindingItem to support both nomenclature conventions.
type AuditFinding = FindingItem

// CategoryBreakdown captures the score and status for a specific audit domain.
type CategoryBreakdown struct {
	DomainKey  string `json:"domain_key"`
	DomainName string `json:"domain_name"`
	Score      int    `json:"score"`      // 0 - 25
	MaxScore   int    `json:"max_score"`  // 25
	Status     string `json:"status"`     // "Hardened", "Strong", "Moderate", "Action Required"
	Summary    string `json:"summary"`
}

// DomainScore is an alias for CategoryBreakdown to support both nomenclature conventions.
type DomainScore = CategoryBreakdown

// SecurityAuditReport represents the comprehensive output of a multi-vector security audit.
type SecurityAuditReport struct {
	ID             int64          `json:"id"`
	UserID         int64          `json:"user_id"`
	TargetURL      string         `json:"target_url"`
	TargetHost     string         `json:"target_host"`
	TargetIP       string         `json:"target_ip"`
	CompositeScore int            `json:"composite_score"` // 0 - 100
	Grade          string         `json:"grade"`           // "A+", "A", "B", "C", "F"
	GradeLabel     string         `json:"grade_label"`     // "Hardened Defense", "Strong Defense", etc.
	CreatedAt      time.Time      `json:"created_at"`
	ExecutionMs    int64          `json:"execution_ms"`
	DomainScores   []DomainScore  `json:"domain_scores"`
	Findings       []AuditFinding `json:"findings"`
	Attestation    string         `json:"attestation_hash"`
}

// AuditRequest defines the payload for initiating a security audit.
type AuditRequest struct {
	TargetURL string `json:"target_url,omitempty"`
	MonitorID int64  `json:"monitor_id,omitempty"`
}
