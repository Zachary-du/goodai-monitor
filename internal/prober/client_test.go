package prober

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

func TestProbeTarget_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("GoodAI Service OK"))
	}))
	defer ts.Close()

	client := NewTestClient(ts.Client())
	m := &model.Monitor{
		ID:                 1,
		URL:                ts.URL,
		ExpectedStatusCode: 200,
		TimeoutSeconds:     2,
	}

	result := client.ProbeTarget(context.Background(), m)

	if !result.Success {
		t.Fatalf("expected probe to succeed, got failure: %s", result.ErrorMessage)
	}
	if result.Status != model.StatusUp {
		t.Errorf("expected status UP, got %s", result.Status)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", result.StatusCode)
	}
	if result.TotalLatencyMs <= 0 {
		t.Errorf("expected positive total latency, got %d", result.TotalLatencyMs)
	}
}

func TestProbeTarget_StatusCodeMismatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("Upstream server dead"))
	}))
	defer ts.Close()

	client := NewTestClient(ts.Client())
	m := &model.Monitor{
		ID:                 2,
		URL:                ts.URL,
		ExpectedStatusCode: 200,
		TimeoutSeconds:     2,
	}

	result := client.ProbeTarget(context.Background(), m)

	if result.Success {
		t.Fatal("expected probe to fail on 502 status code")
	}
	if result.Status != model.StatusDown {
		t.Errorf("expected status DOWN, got %s", result.Status)
	}
	if result.StatusCode != 502 {
		t.Errorf("expected status code 502, got %d", result.StatusCode)
	}
	expectedErr := "HTTP 502 Bad Gateway"
	if result.ErrorMessage != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, result.ErrorMessage)
	}
}

func TestProbeTarget_KeywordMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("System Health: ALL SYSTEMS OPERATIONAL"))
	}))
	defer ts.Close()

	client := NewTestClient(ts.Client())
	m := &model.Monitor{
		ID:                 3,
		URL:                ts.URL,
		ExpectedStatusCode: 200,
		Keyword:            "operational",
		TimeoutSeconds:     2,
	}

	result := client.ProbeTarget(context.Background(), m)

	if !result.Success {
		t.Fatalf("expected keyword match to succeed, got error: %s", result.ErrorMessage)
	}
	if result.Status != model.StatusUp {
		t.Errorf("expected status UP, got %s", result.Status)
	}
}

func TestProbeTarget_KeywordMismatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Service temporarily under maintenance"))
	}))
	defer ts.Close()

	client := NewTestClient(ts.Client())
	m := &model.Monitor{
		ID:                 4,
		URL:                ts.URL,
		ExpectedStatusCode: 200,
		Keyword:            "Welcome",
		TimeoutSeconds:     2,
	}

	result := client.ProbeTarget(context.Background(), m)

	if result.Success {
		t.Fatal("expected probe to fail due to missing keyword")
	}
	if result.Status != model.StatusDown {
		t.Errorf("expected status DOWN, got %s", result.Status)
	}
	expectedErr := "Keyword 'Welcome' missing"
	if result.ErrorMessage != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, result.ErrorMessage)
	}
}

func TestProbeTarget_KeywordTruncation64KB(t *testing.T) {
	// Generate payload with target keyword only placed AFTER 70 KB
	prefix := strings.Repeat("A", 70*1024)
	body := prefix + "HIDDEN_KEYWORD"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer ts.Close()

	client := NewTestClient(ts.Client())
	m := &model.Monitor{
		ID:                 5,
		URL:                ts.URL,
		ExpectedStatusCode: 200,
		Keyword:            "HIDDEN_KEYWORD",
		TimeoutSeconds:     2,
	}

	result := client.ProbeTarget(context.Background(), m)

	// Since prober limits reading to 64 KB, it should miss HIDDEN_KEYWORD and fail
	if result.Success {
		t.Fatal("expected keyword beyond 64KB to be ignored due to body truncation")
	}
	if result.ErrorMessage != "Keyword 'HIDDEN_KEYWORD' missing" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
}

func TestProbeTarget_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than monitor timeout
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewTestClient(ts.Client())
	m := &model.Monitor{
		ID:                 6,
		URL:                ts.URL,
		ExpectedStatusCode: 200,
		TimeoutSeconds:     1, // 1 second
	}

	// Create a short-lived context to test timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := client.ProbeTarget(ctx, m)

	if result.Success {
		t.Fatal("expected probe to fail due to timeout")
	}
	if result.Status != model.StatusDown {
		t.Errorf("expected status DOWN, got %s", result.Status)
	}
	if !strings.Contains(result.ErrorMessage, "timeout") {
		t.Errorf("expected timeout error message, got '%s'", result.ErrorMessage)
	}
}

func TestProbeTarget_RedirectBlocked(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/opc/v2/instance/", http.StatusFound)
	}))
	defer ts.Close()

	client := NewTestClient(ts.Client())
	m := &model.Monitor{
		ID:                 7,
		URL:                ts.URL,
		ExpectedStatusCode: 200,
		TimeoutSeconds:     2,
	}

	result := client.ProbeTarget(context.Background(), m)

	// Should not follow redirect, and fail because 302 != 200
	if result.Success {
		t.Fatal("expected redirect probe to fail when expecting 200")
	}
	if result.StatusCode != http.StatusFound {
		t.Errorf("expected status code 302, got %d", result.StatusCode)
	}
}

func TestProbeTarget_ProductionClient_BlocksLoopback(t *testing.T) {
	prodClient := NewClient()

	m := &model.Monitor{
		ID:                 8,
		URL:                "http://127.0.0.1:8080/metrics",
		ExpectedStatusCode: 200,
		TimeoutSeconds:     2,
	}

	result := prodClient.ProbeTarget(context.Background(), m)

	if result.Success {
		t.Fatal("expected production client to block loopback IP via SSRF guard")
	}
	if result.Status != model.StatusDown {
		t.Errorf("expected status DOWN, got %s", result.Status)
	}
	if !strings.Contains(result.ErrorMessage, "restricted") {
		t.Errorf("expected restricted target URL error, got '%s'", result.ErrorMessage)
	}
}
