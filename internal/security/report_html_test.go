package security

import (
	"strings"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestGenerateExecutiveHTMLReport(t *testing.T) {
	report := &model.SecurityAuditReport{
		ID:             42,
		UserID:         1,
		TargetURL:      "https://goodai.co.nz",
		TargetHost:     "goodai.co.nz",
		TargetIP:       "158.179.29.27",
		CompositeScore: 92,
		Grade:          "A",
		GradeLabel:     "Strong Defense Posture",
		CreatedAt:      time.Now().UTC(),
		ExecutionMs:    280,
		Attestation:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		DomainScores: []model.CategoryBreakdown{
			{DomainKey: "ai_governance", DomainName: "AI Bot & Scraper Governance", Score: 21, MaxScore: 25, Status: "Strong", Summary: "Action required on: llms.txt."},
			{DomainKey: "headers", DomainName: "HTTP Security Headers Posture", Score: 21, MaxScore: 25, Status: "Strong", Summary: "Action required on: CSP."},
			{DomainKey: "tls", DomainName: "Transport & TLS Cryptography", Score: 25, MaxScore: 25, Status: "Hardened", Summary: "All checks passed."},
			{DomainKey: "exposure", DomainName: "Exposure & Surface Shield", Score: 25, MaxScore: 25, Status: "Hardened", Summary: "All checks passed."},
		},
		Findings: []model.FindingItem{
			{
				ID:             "ai-bot-gptbot",
				Domain:         "ai_governance",
				CheckName:      "OpenAI Scraper Policy (GPTBot)",
				Severity:       model.SeverityPass,
				ScoreDeduction: 0,
				ObservedValue:  "Disallow: /api/",
				Recommendation: "Explicit policy active.",
			},
			{
				ID:             "hdr-csp",
				Domain:         "headers",
				CheckName:      "Content-Security-Policy (CSP)",
				Severity:       model.SeverityMedium,
				ScoreDeduction: 5,
				ObservedValue:  "Directive missing",
				Recommendation: "Implement a strict Content-Security-Policy header.",
				Remediation:    `add_header Content-Security-Policy "default-src 'self';" always;`,
			},
		},
	}

	html, err := GenerateExecutiveHTMLReport(report)
	if err != nil {
		t.Fatalf("failed to generate HTML report: %v", err)
	}

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Errorf("expected HTML document doctype")
	}
	if !strings.Contains(html, "goodai.co.nz") {
		t.Errorf("expected target host in report HTML")
	}
	if !strings.Contains(html, "sec-rpt-42") {
		t.Errorf("expected report ID in HTML")
	}
	if !strings.Contains(html, "add_header Content-Security-Policy") {
		t.Errorf("expected remediation snippet in report HTML")
	}
	if !strings.Contains(html, "@media print") {
		t.Errorf("expected @media print stylesheet for PDF optimization")
	}
}
