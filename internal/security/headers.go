package security

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

var (
	hstsMaxAgeRegex = regexp.MustCompile(`(?i)max-age\s*=\s*(\d+)`)
)

// AuditHeaders requests the target URL and audits its HTTP response security headers.
func AuditHeaders(ctx context.Context, client *http.Client, targetURL *url.URL) ([]model.FindingItem, error) {
	if client == nil {
		client = NewSafeHTTPClient(DefaultCheckTimeout)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GoodAI-Security-Audit/1.0 (+https://goodai.co.nz)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Drain limited response body
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)

	return AuditHeaderMap(resp.Header), nil
}

// AuditHeaderMap evaluates a map of HTTP response headers against modern security baselines.
func AuditHeaderMap(headers http.Header) []model.FindingItem {
	var findings []model.FindingItem

	cspVal := strings.TrimSpace(headers.Get("Content-Security-Policy"))
	hstsVal := strings.TrimSpace(headers.Get("Strict-Transport-Security"))
	xfoVal := strings.TrimSpace(headers.Get("X-Frame-Options"))
	xctoVal := strings.TrimSpace(headers.Get("X-Content-Type-Options"))
	refVal := strings.TrimSpace(headers.Get("Referrer-Policy"))
	permVal := strings.TrimSpace(headers.Get("Permissions-Policy"))
	if permVal == "" {
		permVal = strings.TrimSpace(headers.Get("Feature-Policy"))
	}

	// 1. Content-Security-Policy (CSP)
	if cspVal == "" {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-csp",
			Domain:         string(model.DomainHeaders),
			CheckName:      "Content-Security-Policy (CSP)",
			Severity:       model.SeverityMedium,
			ScoreDeduction: 5,
			ObservedValue:  "Directive missing",
			Recommendation: "Implement a strict Content-Security-Policy header to prevent Cross-Site Scripting (XSS) and unauthorized framing.",
			Remediation:    "add_header Content-Security-Policy \"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'self';\" always;",
		})
	} else {
		lowerCSP := strings.ToLower(cspVal)
		if strings.Contains(lowerCSP, "'unsafe-inline'") || strings.Contains(lowerCSP, "*") {
			findings = append(findings, model.FindingItem{
				ID:             "hdr-csp",
				Domain:         string(model.DomainHeaders),
				CheckName:      "Content-Security-Policy (CSP)",
				Severity:       model.SeverityLow,
				ScoreDeduction: 2,
				ObservedValue:  "Present with permissive directive ('unsafe-inline' or wildcard '*')",
				Recommendation: "Strengthen CSP by replacing 'unsafe-inline' with cryptographic nonces or hashes.",
				Remediation:    "add_header Content-Security-Policy \"default-src 'self'; script-src 'self'; object-src 'none';\" always;",
			})
		} else {
			findings = append(findings, model.FindingItem{
				ID:             "hdr-csp",
				Domain:         string(model.DomainHeaders),
				CheckName:      "Content-Security-Policy (CSP)",
				Severity:       model.SeverityPass,
				ScoreDeduction: 0,
				ObservedValue:  "Strict CSP policy enforced",
				Recommendation: "Policy conforms to defense-in-depth guidelines.",
				Remediation:    "",
			})
		}
	}

	// 2. Strict-Transport-Security (HSTS)
	if hstsVal == "" {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-hsts",
			Domain:         string(model.DomainHeaders),
			CheckName:      "Strict-Transport-Security (HSTS)",
			Severity:       model.SeverityHigh,
			ScoreDeduction: 10,
			ObservedValue:  "Directive missing",
			Recommendation: "Enforce HTTPS transport via Strict-Transport-Security to prevent SSL-stripping MITM attacks.",
			Remediation:    "add_header Strict-Transport-Security \"max-age=31536000; includeSubDomains; preload\" always;",
		})
	} else {
		matches := hstsMaxAgeRegex.FindStringSubmatch(hstsVal)
		var maxAge int64
		if len(matches) > 1 {
			maxAge, _ = strconv.ParseInt(matches[1], 10, 64)
		}
		if maxAge < 15768000 {
			findings = append(findings, model.FindingItem{
				ID:             "hdr-hsts",
				Domain:         string(model.DomainHeaders),
				CheckName:      "Strict-Transport-Security (HSTS)",
				Severity:       model.SeverityMedium,
				ScoreDeduction: 5,
				ObservedValue:  fmt.Sprintf("max-age=%d (less than recommended 6 months / 15,768,000s)", maxAge),
				Recommendation: "Increase HSTS max-age to at least 15,768,000 (recommended 31,536,000 for 1 year).",
				Remediation:    "add_header Strict-Transport-Security \"max-age=31536000; includeSubDomains; preload\" always;",
			})
		} else {
			obs := fmt.Sprintf("max-age=%d", maxAge)
			if strings.Contains(strings.ToLower(hstsVal), "includesubdomains") {
				obs += " (includeSubDomains active)"
			}
			if strings.Contains(strings.ToLower(hstsVal), "preload") {
				obs += " (preload active)"
			}
			findings = append(findings, model.FindingItem{
				ID:             "hdr-hsts",
				Domain:         string(model.DomainHeaders),
				CheckName:      "Strict-Transport-Security (HSTS)",
				Severity:       model.SeverityPass,
				ScoreDeduction: 0,
				ObservedValue:  obs,
				Recommendation: "HSTS header properly configured.",
				Remediation:    "",
			})
		}
	}

	// 3. X-Frame-Options
	lowerXFO := strings.ToUpper(xfoVal)
	if lowerXFO == "DENY" || lowerXFO == "SAMEORIGIN" {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-xfo",
			Domain:         string(model.DomainHeaders),
			CheckName:      "X-Frame-Options (Clickjacking Defense)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  lowerXFO,
			Recommendation: "Framing protection active.",
			Remediation:    "",
		})
	} else if strings.Contains(strings.ToLower(cspVal), "frame-ancestors") {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-xfo",
			Domain:         string(model.DomainHeaders),
			CheckName:      "X-Frame-Options (Clickjacking Defense)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  "Protected via CSP frame-ancestors directive",
			Recommendation: "Modern CSP frame-ancestors in effect.",
			Remediation:    "",
		})
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-xfo",
			Domain:         string(model.DomainHeaders),
			CheckName:      "X-Frame-Options (Clickjacking Defense)",
			Severity:       model.SeverityMedium,
			ScoreDeduction: 5,
			ObservedValue:  "Directive missing",
			Recommendation: "Set X-Frame-Options to DENY or SAMEORIGIN to prevent UI clickjacking attacks.",
			Remediation:    "add_header X-Frame-Options \"DENY\" always;",
		})
	}

	// 4. X-Content-Type-Options
	if strings.EqualFold(xctoVal, "nosniff") {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-xcto",
			Domain:         string(model.DomainHeaders),
			CheckName:      "X-Content-Type-Options (MIME Sniffing)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  "nosniff enforced",
			Recommendation: "MIME sniffing protection active.",
			Remediation:    "",
		})
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-xcto",
			Domain:         string(model.DomainHeaders),
			CheckName:      "X-Content-Type-Options (MIME Sniffing)",
			Severity:       model.SeverityMedium,
			ScoreDeduction: 5,
			ObservedValue:  "Directive missing or invalid",
			Recommendation: "Set X-Content-Type-Options to 'nosniff' to prevent browsers from MIME-sniffing response payloads.",
			Remediation:    "add_header X-Content-Type-Options \"nosniff\" always;",
		})
	}

	// 5. Referrer-Policy
	if refVal != "" {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-referrer-policy",
			Domain:         string(model.DomainHeaders),
			CheckName:      "Referrer-Policy",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  refVal,
			Recommendation: "Referrer policy active.",
			Remediation:    "",
		})
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-referrer-policy",
			Domain:         string(model.DomainHeaders),
			CheckName:      "Referrer-Policy",
			Severity:       model.SeverityLow,
			ScoreDeduction: 2,
			ObservedValue:  "Directive missing",
			Recommendation: "Set Referrer-Policy to protect sensitive URLs from leaking across third-party origins.",
			Remediation:    "add_header Referrer-Policy \"strict-origin-when-cross-origin\" always;",
		})
	}

	// 6. Permissions-Policy
	if permVal != "" {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-permissions-policy",
			Domain:         string(model.DomainHeaders),
			CheckName:      "Permissions-Policy",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  permVal,
			Recommendation: "Browser API access control policy configured.",
			Remediation:    "",
		})
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "hdr-permissions-policy",
			Domain:         string(model.DomainHeaders),
			CheckName:      "Permissions-Policy",
			Severity:       model.SeverityLow,
			ScoreDeduction: 2,
			ObservedValue:  "Directive missing",
			Recommendation: "Configure Permissions-Policy to explicitly disable unnecessary client device APIs (camera, microphone, geolocation).",
			Remediation:    "add_header Permissions-Policy \"camera=(), microphone=(), geolocation=()\" always;",
		})
	}

	return findings
}
