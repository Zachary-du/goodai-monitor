package security

import (
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestCalculateScores_Perfect(t *testing.T) {
	findings := []model.FindingItem{
		{Domain: "ai_governance", Severity: model.SeverityPass, ScoreDeduction: 0},
		{Domain: "headers", Severity: model.SeverityPass, ScoreDeduction: 0},
		{Domain: "tls", Severity: model.SeverityPass, ScoreDeduction: 0},
		{Domain: "exposure", Severity: model.SeverityPass, ScoreDeduction: 0},
	}

	score, grade, label, domains := CalculateScores(findings)

	if score != 100 {
		t.Errorf("expected score 100, got %d", score)
	}
	if grade != "A+" {
		t.Errorf("expected grade A+, got %s", grade)
	}
	if label != "Hardened Defense" {
		t.Errorf("expected label Hardened Defense, got %s", label)
	}
	if len(domains) != 4 {
		t.Fatalf("expected 4 domains, got %d", len(domains))
	}
	for _, d := range domains {
		if d.Score != 25 {
			t.Errorf("expected domain %s score 25, got %d", d.DomainName, d.Score)
		}
		if d.Status != "Hardened" {
			t.Errorf("expected domain %s status Hardened, got %s", d.DomainName, d.Status)
		}
	}
}

func TestCalculateScores_DeductionsAndClamping(t *testing.T) {
	findings := []model.FindingItem{
		// D1 deductions = 10 -> Score 15
		{Domain: "ai_governance", CheckName: "Robots.txt", Severity: model.SeverityHigh, ScoreDeduction: 10},
		// D2 deductions = 30 -> Clamped to 0
		{Domain: "headers", CheckName: "CSP", Severity: model.SeverityCritical, ScoreDeduction: 20},
		{Domain: "headers", CheckName: "HSTS", Severity: model.SeverityHigh, ScoreDeduction: 10},
		// D3 deductions = 0 -> Score 25
		{Domain: "tls", CheckName: "TLS 1.3", Severity: model.SeverityPass, ScoreDeduction: 0},
		// D4 deductions = 20 -> Score 5
		{Domain: "exposure", CheckName: "Env File", Severity: model.SeverityCritical, ScoreDeduction: 20},
	}

	score, grade, _, domains := CalculateScores(findings)

	// D1=15, D2=0, D3=25, D4=5 -> Total = 45 -> Grade F
	if score != 45 {
		t.Errorf("expected score 45, got %d", score)
	}
	if grade != "F" {
		t.Errorf("expected grade F, got %s", grade)
	}

	for _, d := range domains {
		switch d.DomainKey {
		case "ai_governance":
			if d.Score != 15 {
				t.Errorf("expected ai_governance score 15, got %d", d.Score)
			}
		case "headers":
			if d.Score != 0 {
				t.Errorf("expected headers score 0 (clamped), got %d", d.Score)
			}
		case "tls":
			if d.Score != 25 {
				t.Errorf("expected tls score 25, got %d", d.Score)
			}
		case "exposure":
			if d.Score != 5 {
				t.Errorf("expected exposure score 5, got %d", d.Score)
			}
		}
	}
}

func TestEvaluateGrade(t *testing.T) {
	tests := []struct {
		score     int
		wantGrade string
	}{
		{100, "A+"},
		{95, "A+"},
		{94, "A"},
		{85, "A"},
		{84, "B"},
		{70, "B"},
		{69, "C"},
		{50, "C"},
		{49, "F"},
		{0, "F"},
	}

	for _, tt := range tests {
		got, _ := EvaluateGrade(tt.score)
		if got != tt.wantGrade {
			t.Errorf("EvaluateGrade(%d) = %s, want %s", tt.score, got, tt.wantGrade)
		}
	}
}
