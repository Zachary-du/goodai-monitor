package security

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestAuditTLS_PlaintextHTTP(t *testing.T) {
	u, _ := url.Parse("http://example.com/api")
	findings, err := AuditTLS(context.Background(), u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var foundCritical bool
	for _, f := range findings {
		if f.ID == "tls-protocol" && f.Severity == model.SeverityCritical {
			foundCritical = true
		}
	}

	if !foundCritical {
		t.Errorf("expected plaintext HTTP to yield CRITICAL severity for tls-protocol")
	}
}

func TestAuditTLS_TLSServer(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}

	// Use standard dialer since httptest runs on 127.0.0.1
	dialer := &net.Dialer{}
	findings, err := AuditTLSWithDialer(context.Background(), u, dialer.DialContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var foundProtocol, foundSAN bool
	for _, f := range findings {
		if f.ID == "tls-protocol" && f.Severity == model.SeverityPass {
			foundProtocol = true
		}
		if f.ID == "tls-cert-san" && f.Severity == model.SeverityPass {
			foundSAN = true
		}
	}

	if !foundProtocol {
		t.Errorf("expected tls-protocol to be PASS on httptest TLS server")
	}
	if !foundSAN {
		t.Errorf("expected tls-cert-san to be PASS on httptest TLS server matching 127.0.0.1")
	}
}
