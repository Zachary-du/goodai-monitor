package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestParseRobotsTXT(t *testing.T) {
	robotsContent := `
# robots.txt for GoodAI
User-agent: Googlebot
Allow: /
Disallow: /admin/

User-agent: GPTBot
User-agent: ClaudeBot
Disallow: /api/
Disallow: /secrets/

User-agent: PerplexityBot
Disallow: /

User-agent: *
Disallow: /internal/
Allow: /public/
`

	policies := ParseRobotsTXT(robotsContent)

	if p, ok := policies["gptbot"]; !ok || !p.HasExplicit {
		t.Fatalf("expected explicit policy for gptbot")
	} else if len(p.Disallows) != 2 {
		t.Errorf("expected 2 disallow rules for gptbot, got %d", len(p.Disallows))
	}

	if p, ok := policies["claudebot"]; !ok || !p.HasExplicit {
		t.Fatalf("expected explicit policy for claudebot")
	} else if len(p.Disallows) != 2 {
		t.Errorf("expected 2 disallow rules for claudebot, got %d", len(p.Disallows))
	}

	if p, ok := policies["perplexitybot"]; !ok || len(p.Disallows) != 1 || p.Disallows[0] != "/" {
		t.Errorf("expected perplexitybot to be fully disallowed, got %+v", p)
	}

	if p, ok := policies["*"]; !ok || len(p.Disallows) != 1 {
		t.Errorf("expected wildcard policy to exist with 1 disallow rule")
	}
}

func TestAuditAIGovernance_Compliant(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("User-agent: GPTBot\nDisallow: /\nUser-agent: ClaudeBot\nDisallow: /\n"))
		case "/llms.txt":
			w.Header().Set("Content-Type", "text/markdown")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# Project Overview\nCurated context for LLMs.\n"))
		case "/.well-known/security.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Contact: mailto:security@goodai.co.nz\nExpires: 2027-12-31T23:59:59Z\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	targetURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse test server url: %v", err)
	}

	findings, err := AuditAIGovernance(context.Background(), ts.Client(), targetURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var foundLLMs, foundSec, foundGPT bool
	for _, f := range findings {
		if f.ID == "ai-llms-txt" && f.Severity == model.SeverityPass {
			foundLLMs = true
		}
		if f.ID == "ai-security-txt" && f.Severity == model.SeverityPass {
			foundSec = true
		}
		if f.ID == "ai-bot-gptbot" && f.Severity == model.SeverityPass {
			foundGPT = true
		}
	}

	if !foundLLMs {
		t.Errorf("expected llms.txt finding to be PASS")
	}
	if !foundSec {
		t.Errorf("expected security.txt finding to be PASS")
	}
	if !foundGPT {
		t.Errorf("expected GPTBot policy finding to be PASS")
	}
}

func TestAuditAIGovernance_MissingFiles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	targetURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse test server url: %v", err)
	}

	findings, err := AuditAIGovernance(context.Background(), ts.Client(), targetURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var foundMissingRobots, foundMissingLLMs, foundMissingSec bool
	for _, f := range findings {
		if f.ID == "ai-robots-missing" && f.Severity == model.SeverityHigh {
			foundMissingRobots = true
		}
		if f.ID == "ai-llms-txt" && f.Severity == model.SeverityLow {
			foundMissingLLMs = true
		}
		if f.ID == "ai-security-txt" && f.Severity == model.SeverityLow {
			foundMissingSec = true
		}
	}

	if !foundMissingRobots {
		t.Errorf("expected missing robots.txt finding to have HIGH severity")
	}
	if !foundMissingLLMs {
		t.Errorf("expected missing llms.txt finding to have LOW severity")
	}
	if !foundMissingSec {
		t.Errorf("expected missing security.txt finding to have LOW severity")
	}
}
