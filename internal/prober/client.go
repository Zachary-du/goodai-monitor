package prober

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
	"github.com/Zachary-du/goodai-monitor/internal/security"
)

const (
	// MaxKeywordScanBytes defines the strict 64 KB read cap for response bodies.
	MaxKeywordScanBytes = 64 * 1024
)

// Prober defines the contract for executing availability probes against a monitor target.
type Prober interface {
	ProbeTarget(ctx context.Context, m *model.Monitor) *model.CheckResult
}

// Client executes synthetic HTTP/HTTPS checks with strict SSRF defense and detailed latency tracing.
type Client struct {
	dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	httpClient  *http.Client
}

// NewClient constructs a standard production Prober Client with dial-time SSRF validation.
func NewClient() *Client {
	safeDial := func(ctx context.Context, network, address string) (net.Conn, error) {
		host, portStr, err := net.SplitHostPort(address)
		if err != nil {
			return nil, security.ErrRestrictedTargetURL
		}

		port, err := strconv.Atoi(portStr)
		if err != nil || !security.AllowedPorts[port] {
			return nil, fmt.Errorf("%w: port %s is restricted", security.ErrRestrictedTargetURL, portStr)
		}

		dialer := security.NewSafeDialer(10 * time.Second)

		// Direct IP check
		if ip := net.ParseIP(host); ip != nil {
			if security.IsRestrictedIP(ip) {
				return nil, fmt.Errorf("%w: IP %s is restricted", security.ErrRestrictedTargetURL, host)
			}
			return dialer.DialContext(ctx, network, address)
		}

		// Domain resolution with httptrace callback invocation
		trace := httptrace.ContextClientTrace(ctx)
		if trace != nil && trace.DNSStart != nil {
			trace.DNSStart(httptrace.DNSStartInfo{Host: host})
		}

		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if trace != nil && trace.DNSDone != nil {
			trace.DNSDone(httptrace.DNSDoneInfo{Addrs: ips, Err: err})
		}

		if err != nil {
			return nil, fmt.Errorf("%w: lookup failed: %v", security.ErrRestrictedTargetURL, err)
		}
		if len(ips) == 0 {
			return nil, security.ErrRestrictedTargetURL
		}

		for _, resolvedIP := range ips {
			if security.IsRestrictedIP(resolvedIP) {
				return nil, fmt.Errorf("%w: resolved to restricted IP %s", security.ErrRestrictedTargetURL, resolvedIP)
			}
		}

		// Connect to the verified IP directly to prevent TOCTOU DNS Rebinding
		verifiedAddr := net.JoinHostPort(ips[0].String(), portStr)
		return dialer.DialContext(ctx, network, verifiedAddr)
	}

	transport := &http.Transport{
		DialContext:           safeDial,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Prohibit automatic redirects to prevent target SSRF bypasses
			return http.ErrUseLastResponse
		},
	}

	return &Client{
		dialContext: safeDial,
		httpClient:  httpClient,
	}
}

// NewTestClient constructs a Prober Client with custom HTTP transport (e.g. for httptest.Server).
func NewTestClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		return NewClient()
	}
	// Enforce ErrUseLastResponse redirect policy
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{
		httpClient: httpClient,
	}
}

// ProbeTarget executes a synthetic availability check against the specified monitor.
func (c *Client) ProbeTarget(ctx context.Context, m *model.Monitor) *model.CheckResult {
	timeoutSeconds := m.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = model.DefaultTimeoutSeconds
	}
	timeoutDuration := time.Duration(timeoutSeconds) * time.Second

	probeCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	result := &model.CheckResult{
		Status:  model.StatusDown,
		Success: false,
	}

	var (
		dnsStart, dnsEnd     time.Time
		connStart, connEnd   time.Time
		tlsStart, tlsEnd     time.Time
		gotFirstByte         time.Time
		reqStart             = time.Now()
	)

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsEnd = time.Now()
		},
		ConnectStart: func(network, addr string) {
			connStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			connEnd = time.Now()
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsEnd = time.Now()
		},
		GotFirstResponseByte: func() {
			gotFirstByte = time.Now()
		},
	}

	probeCtx = httptrace.WithClientTrace(probeCtx, trace)

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, m.URL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Invalid URL or request: %v", err)
		return result
	}
	req.Header.Set("User-Agent", "GoodAI-Monitor/1.0 (+https://goodai.co.nz)")
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)

	// Compute latency breakdown
	totalLatency := time.Since(reqStart)
	totalMs := totalLatency.Milliseconds()
	if totalMs <= 0 {
		totalMs = 1
	}

	var dnsMs, tcpMs, tlsMs, ttfbMs int64
	if !dnsEnd.IsZero() && !dnsStart.IsZero() && dnsEnd.After(dnsStart) {
		dnsMs = dnsEnd.Sub(dnsStart).Milliseconds()
	}
	if !connEnd.IsZero() && !connStart.IsZero() && connEnd.After(connStart) {
		tcpMs = connEnd.Sub(connStart).Milliseconds()
	}
	if !tlsEnd.IsZero() && !tlsStart.IsZero() && tlsEnd.After(tlsStart) {
		tlsMs = tlsEnd.Sub(tlsStart).Milliseconds()
	}
	if !gotFirstByte.IsZero() && gotFirstByte.After(reqStart) {
		ttfbMs = gotFirstByte.Sub(reqStart).Milliseconds()
	}

	result.TotalLatencyMs = totalMs
	result.LatencyBreakdown = model.LatencyBreakdown{
		DNSLookupMs:    dnsMs,
		TCPConnectMs:   tcpMs,
		TLSHandshakeMs: tlsMs,
		TTFBMs:         ttfbMs,
		TotalMs:        totalMs,
	}

	if err != nil {
		if errors.Is(probeCtx.Err(), context.DeadlineExceeded) || os.IsTimeout(err) {
			result.ErrorMessage = fmt.Sprintf("Connection timeout after %dms", timeoutDuration.Milliseconds())
		} else if errors.Is(err, security.ErrRestrictedTargetURL) || strings.Contains(err.Error(), "restricted") {
			result.ErrorMessage = "Invalid or restricted target URL"
		} else {
			result.ErrorMessage = formatProbeNetworkError(err)
		}
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Inspect TLS certificate expiration for HTTPS targets
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		leaf := resp.TLS.PeerCertificates[0]
		remainingHours := time.Until(leaf.NotAfter).Hours()
		daysRemaining := int(remainingHours / 24)
		result.CertExpiryDays = &daysRemaining

		if time.Now().After(leaf.NotAfter) {
			result.ErrorMessage = "TLS certificate expired"
			return result
		}
	}

	// Validate HTTP status code
	expectedCode := m.ExpectedStatusCode
	if expectedCode <= 0 {
		expectedCode = model.DefaultExpectedStatusCode
	}

	if resp.StatusCode != expectedCode {
		statusText := http.StatusText(resp.StatusCode)
		if statusText == "" {
			statusText = "Unknown Status"
		}
		if expectedCode == model.DefaultExpectedStatusCode {
			result.ErrorMessage = fmt.Sprintf("HTTP %d %s", resp.StatusCode, statusText)
		} else {
			result.ErrorMessage = fmt.Sprintf("HTTP %d %s (expected %d)", resp.StatusCode, statusText, expectedCode)
		}
		return result
	}

	// Scan body for keyword (capped at 64 KB)
	if m.Keyword != "" {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, MaxKeywordScanBytes))
		if readErr != nil {
			result.ErrorMessage = fmt.Sprintf("Failed reading response body: %v", readErr)
			return result
		}

		if !strings.Contains(strings.ToLower(string(bodyBytes)), strings.ToLower(m.Keyword)) {
			result.ErrorMessage = fmt.Sprintf("Keyword '%s' missing", m.Keyword)
			return result
		}
	} else {
		// Discard remaining body bytes up to 64KB to release socket
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxKeywordScanBytes))
	}

	// All assertions passed
	result.Success = true
	result.Status = model.StatusUp
	return result
}

func formatProbeNetworkError(err error) string {
	if err == nil {
		return "Unknown error"
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return fmt.Sprintf("Connection failed: %v", opErr.Err)
		}
	}

	return err.Error()
}
