package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestAuditExposures_FalsePositiveSPA(t *testing.T) {
	// A Single Page App (SPA) server that returns HTTP 200 with an HTML shell for ALL paths!
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body><div id=\"root\">SPA App</div></body></html>"))
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}

	findings, err := AuditExposures(context.Background(), ts.Client(), u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range findings {
		if f.Severity != model.SeverityPass {
			t.Errorf("expected probe %s to PASS on SPA HTML fallback, but got %s", f.ID, f.Severity)
		}
	}
}

func TestAuditExposures_RealExposures(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.env":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("DATABASE_URL=postgres://user:secret@db:5432/prod\nAPI_KEY=xyz987\n"))
		case "/.git/HEAD":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ref: refs/heads/main\n"))
		case "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Internal API"}}`))
		case "/swagger":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><head><title>Swagger UI</title></head><body><div id=\"swagger-ui\"></div></body></html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}

	findings, err := AuditExposures(context.Background(), ts.Client(), u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envCritical, gitCritical, openapiMed, swaggerMed bool
	for _, f := range findings {
		if f.ID == "exp-env-file" && f.Severity == model.SeverityCritical {
			envCritical = true
		}
		if f.ID == "exp-git-repo" && f.Severity == model.SeverityCritical {
			gitCritical = true
		}
		if f.ID == "exp-openapi-spec" && f.Severity == model.SeverityMedium {
			openapiMed = true
		}
		if f.ID == "exp-swagger-ui" && f.Severity == model.SeverityMedium {
			swaggerMed = true
		}
	}

	if !envCritical {
		t.Errorf("expected /.env exposure to be CRITICAL")
	}
	if !gitCritical {
		t.Errorf("expected /.git/HEAD exposure to be CRITICAL")
	}
	if !openapiMed {
		t.Errorf("expected /openapi.json exposure to be MEDIUM")
	}
	if !swaggerMed {
		t.Errorf("expected /swagger exposure to be MEDIUM")
	}
}
