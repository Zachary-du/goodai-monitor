package security

import (
	"fmt"
	"strings"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

// DomainConfig defines the metadata for each inspection pillar.
type DomainConfig struct {
	Key      model.AuditDomain
	Name     string
	MaxScore int
}

var AuditDomains = []DomainConfig{
	{Key: model.DomainAIGovernance, Name: "AI Bot & Scraper Governance", MaxScore: 25},
	{Key: model.DomainHeaders, Name: "HTTP Security Headers Posture", MaxScore: 25},
	{Key: model.DomainTLS, Name: "Transport & TLS Cryptography", MaxScore: 25},
	{Key: model.DomainExposure, Name: "Exposure & Surface Shield", MaxScore: 25},
}

// CalculateScores evaluates findings across the 4 domains and computes composite score and grade.
func CalculateScores(findings []model.FindingItem) (int, string, string, []model.CategoryBreakdown) {
	domainBreakdowns := make([]model.CategoryBreakdown, len(AuditDomains))
	compositeScore := 0

	for i, cfg := range AuditDomains {
		deduction := 0
		var nonPassFindings []string

		for _, f := range findings {
			if f.Domain == string(cfg.Key) {
				deduction += f.ScoreDeduction
				if f.Severity != model.SeverityPass {
					nonPassFindings = append(nonPassFindings, f.CheckName)
				}
			}
		}

		score := cfg.MaxScore - deduction
		if score < 0 {
			score = 0
		}
		if score > cfg.MaxScore {
			score = cfg.MaxScore
		}

		compositeScore += score

		var status string
		if score == cfg.MaxScore {
			status = "Hardened"
		} else if score >= 20 {
			status = "Strong"
		} else if score >= 15 {
			status = "Moderate"
		} else {
			status = "Action Required"
		}

		var summary string
		if len(nonPassFindings) == 0 {
			summary = "All checks passed in this domain."
		} else {
			summary = fmt.Sprintf("Action required on: %s.", strings.Join(nonPassFindings, ", "))
		}

		domainBreakdowns[i] = model.CategoryBreakdown{
			DomainKey:  string(cfg.Key),
			DomainName: cfg.Name,
			Score:      score,
			MaxScore:   cfg.MaxScore,
			Status:     status,
			Summary:    summary,
		}
	}

	if compositeScore < 0 {
		compositeScore = 0
	}
	if compositeScore > 100 {
		compositeScore = 100
	}

	grade, gradeLabel := EvaluateGrade(compositeScore)

	return compositeScore, grade, gradeLabel, domainBreakdowns
}

// EvaluateGrade maps a composite 0-100 score to executive letter grade and posture label.
func EvaluateGrade(score int) (string, string) {
	switch {
	case score >= 95:
		return "A+", "Hardened Defense"
	case score >= 85:
		return "A", "Strong Defense Posture"
	case score >= 70:
		return "B", "Moderate Posture"
	case score >= 50:
		return "C", "Deficient Posture"
	default:
		return "F", "Critical Exposure"
	}
}
