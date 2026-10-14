package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestAuditHeaderMap_FullyHardened(t *testing.T) {
	h := make(http.Header)
	h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'")
	h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

	findings := AuditHeaderMap(h)
	if len(findings) != 6 {
		t.Fatalf("expected 6 findings, got %d", len(findings))
	}

	for _, f := range findings {
		if f.Severity != model.SeverityPass {
			t.Errorf("expected finding %s to be PASS, got %s", f.ID, f.Severity)
		}
		if f.ScoreDeduction != 0 {
			t.Errorf("expected 0 deduction for %s, got %d", f.ID, f.ScoreDeduction)
		}
	}
}

func TestAuditHeaderMap_MissingAll(t *testing.T) {
	h := make(http.Header)
	findings := AuditHeaderMap(h)

	var cspFound, hstsFound, xfoFound, xctoFound, refFound, permFound bool
	for _, f := range findings {
		switch f.ID {
		case "hdr-csp":
			cspFound = true
			if f.Severity != model.SeverityMedium {
				t.Errorf("expected CSP to be MEDIUM, got %s", f.Severity)
			}
		case "hdr-hsts":
			hstsFound = true
			if f.Severity != model.SeverityHigh {
				t.Errorf("expected HSTS to be HIGH, got %s", f.Severity)
			}
		case "hdr-xfo":
			xfoFound = true
			if f.Severity != model.SeverityMedium {
				t.Errorf("expected XFO to be MEDIUM, got %s", f.Severity)
			}
		case "hdr-xcto":
			xctoFound = true
			if f.Severity != model.SeverityMedium {
				t.Errorf("expected XCTO to be MEDIUM, got %s", f.Severity)
			}
		case "hdr-referrer-policy":
			refFound = true
			if f.Severity != model.SeverityLow {
				t.Errorf("expected Referrer-Policy to be LOW, got %s", f.Severity)
			}
		case "hdr-permissions-policy":
			permFound = true
			if f.Severity != model.SeverityLow {
				t.Errorf("expected Permissions-Policy to be LOW, got %s", f.Severity)
			}
		}
	}

	if !cspFound || !hstsFound || !xfoFound || !xctoFound || !refFound || !permFound {
		t.Errorf("expected all 6 header findings to be recorded when empty")
	}
}

func TestAuditHeaders_ServerIntegration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=86400") // Less than 6 months
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}

	findings, err := AuditHeaders(context.Background(), ts.Client(), u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range findings {
		if f.ID == "hdr-hsts" {
			if f.Severity != model.SeverityMedium {
				t.Errorf("expected short HSTS max-age to be MEDIUM, got %s", f.Severity)
			}
		}
		if f.ID == "hdr-xcto" {
			if f.Severity != model.SeverityPass {
				t.Errorf("expected nosniff to be PASS, got %s", f.Severity)
			}
		}
	}
}
