package security

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

// DefaultCheckTimeout defines the default per-probe HTTP request timeout.
const DefaultCheckTimeout = 5 * time.Second

// AIBotsOfInterest lists prominent autonomous AI crawlers and scraper agents to audit.
var AIBotsOfInterest = []struct {
	AgentKey    string
	DisplayName string
	Aliases     []string
}{
	{AgentKey: "gptbot", DisplayName: "OpenAI Scraper Policy (GPTBot)", Aliases: []string{"gptbot"}},
	{AgentKey: "claudebot", DisplayName: "Anthropic Scraper Policy (ClaudeBot)", Aliases: []string{"claudebot", "anthropic-ai"}},
	{AgentKey: "perplexitybot", DisplayName: "Perplexity Scraper Policy (PerplexityBot)", Aliases: []string{"perplexitybot"}},
	{AgentKey: "google-extended", DisplayName: "Google Extended AI Policy (Google-Extended)", Aliases: []string{"google-extended"}},
	{AgentKey: "bytespider", DisplayName: "ByteDance Scraper Policy (Bytespider)", Aliases: []string{"bytespider"}},
	{AgentKey: "ccbot", DisplayName: "Common Crawl Bot Policy (CCBot)", Aliases: []string{"ccbot"}},
}

// BotPolicy summarizes directives observed for a given crawler user-agent.
type BotPolicy struct {
	Agent       string
	Disallows   []string
	Allows      []string
	HasExplicit bool
}

// ParseRobotsTXT parses robots.txt content according to RFC 9309 rules, returning policies per user-agent.
func ParseRobotsTXT(content string) map[string]*BotPolicy {
	policies := make(map[string]*BotPolicy)

	getOrCreate := func(agent string) *BotPolicy {
		clean := strings.ToLower(strings.TrimSpace(agent))
		if p, ok := policies[clean]; ok {
			return p
		}
		p := &BotPolicy{Agent: clean}
		policies[clean] = p
		return p
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentAgents []string
	inRecord := false

	for scanner.Scan() {
		line := scanner.Text()
		// Strip comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if inRecord {
				currentAgents = nil
				inRecord = false
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		directive := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch directive {
		case "user-agent":
			if !inRecord || len(currentAgents) == 0 {
				currentAgents = []string{strings.ToLower(val)}
				inRecord = true
			} else {
				// Multiple User-agent lines in the same record header
				currentAgents = append(currentAgents, strings.ToLower(val))
			}
			for _, ag := range currentAgents {
				p := getOrCreate(ag)
				p.HasExplicit = true
			}
		case "disallow":
			if len(currentAgents) > 0 {
				for _, ag := range currentAgents {
					p := getOrCreate(ag)
					p.Disallows = append(p.Disallows, val)
				}
			}
		case "allow":
			if len(currentAgents) > 0 {
				for _, ag := range currentAgents {
					p := getOrCreate(ag)
					p.Allows = append(p.Allows, val)
				}
			}
		}
	}

	return policies
}

// AuditAIGovernance probes robots.txt, /llms.txt, and /security.txt to assess AI crawler governance.
func AuditAIGovernance(ctx context.Context, client *http.Client, targetURL *url.URL) ([]model.FindingItem, error) {
	if client == nil {
		client = NewSafeHTTPClient(DefaultCheckTimeout)
	}

	var findings []model.FindingItem

	// 1. Audit robots.txt
	robotsFindings, err := auditRobotsTXT(ctx, client, targetURL)
	if err == nil {
		findings = append(findings, robotsFindings...)
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "ai-robots-fetch-error",
			Domain:         string(model.DomainAIGovernance),
			CheckName:      "Robots.txt Policy File",
			Severity:       model.SeverityMedium,
			ScoreDeduction: 5,
			ObservedValue:  fmt.Sprintf("Failed to query /robots.txt: %v", err),
			Recommendation: "Ensure /robots.txt is accessible and serves valid HTTP status codes.",
			Remediation:    "User-agent: *\nAllow: /\n\nUser-agent: GPTBot\nDisallow: /",
		})
	}

	// 2. Audit /llms.txt standard
	llmsFinding := auditLLMsTXT(ctx, client, targetURL)
	findings = append(findings, llmsFinding)

	// 3. Audit /security.txt (RFC 9116)
	secFinding := auditSecurityTXT(ctx, client, targetURL)
	findings = append(findings, secFinding)

	return findings, nil
}

func auditRobotsTXT(ctx context.Context, client *http.Client, targetURL *url.URL) ([]model.FindingItem, error) {
	var findings []model.FindingItem

	u := *targetURL
	u.Path = "/robots.txt"
	u.RawQuery = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GoodAI-Security-Audit/1.0 (+https://goodai.co.nz)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		findings = append(findings, model.FindingItem{
			ID:             "ai-robots-missing",
			Domain:         string(model.DomainAIGovernance),
			CheckName:      "Robots.txt Policy File",
			Severity:       model.SeverityHigh,
			ScoreDeduction: 10,
			ObservedValue:  fmt.Sprintf("HTTP %d (Policy Missing)", resp.StatusCode),
			Recommendation: "Deploy a /robots.txt file to govern search engines and autonomous AI scrapers.",
			Remediation:    "User-agent: *\nDisallow: /api/\n\nUser-agent: GPTBot\nDisallow: /\n\nUser-agent: ClaudeBot\nDisallow: /\n\nUser-agent: PerplexityBot\nDisallow: /",
		})
		return findings, nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if err != nil {
		return nil, err
	}
	bodyStr := string(bodyBytes)

	// Filter out HTML error pages masquerading as 200 OK
	lowerBody := strings.ToLower(bodyStr)
	if strings.Contains(lowerBody, "<html") || strings.Contains(lowerBody, "<!doctype") {
		findings = append(findings, model.FindingItem{
			ID:             "ai-robots-html-spa",
			Domain:         string(model.DomainAIGovernance),
			CheckName:      "Robots.txt Policy File",
			Severity:       model.SeverityHigh,
			ScoreDeduction: 10,
			ObservedValue:  "HTTP 200 returned HTML error shell instead of text/plain",
			Recommendation: "Configure reverse proxy or web server to serve plain text robots.txt without SPA HTML fallbacks.",
			Remediation:    "location = /robots.txt {\n    default_type text/plain;\n    return 200 \"User-agent: *\\nAllow: /\\n\";\n}",
		})
		return findings, nil
	}

	policies := ParseRobotsTXT(bodyStr)
	wildcardPolicy := policies["*"]

	explicitCount := 0
	for _, bot := range AIBotsOfInterest {
		var matchedPolicy *BotPolicy
		for _, alias := range bot.Aliases {
			if p, ok := policies[alias]; ok && p.HasExplicit {
				matchedPolicy = p
				break
			}
		}

		if matchedPolicy != nil {
			explicitCount++
			obs := formatBotPolicy(matchedPolicy)
			findings = append(findings, model.FindingItem{
				ID:             fmt.Sprintf("ai-bot-%s", bot.AgentKey),
				Domain:         string(model.DomainAIGovernance),
				CheckName:      bot.DisplayName,
				Severity:       model.SeverityPass,
				ScoreDeduction: 0,
				ObservedValue:  obs,
				Recommendation: "Explicit policy active in robots.txt.",
				Remediation:    "",
			})
		} else {
			// Check if wildcard covers it
			obs := "No explicit policy defined"
			if wildcardPolicy != nil && len(wildcardPolicy.Disallows) > 0 {
				obs += fmt.Sprintf(" (Governed by wildcard User-agent: * [%d disallow rules])", len(wildcardPolicy.Disallows))
			}
			findings = append(findings, model.FindingItem{
				ID:             fmt.Sprintf("ai-bot-%s", bot.AgentKey),
				Domain:         string(model.DomainAIGovernance),
				CheckName:      bot.DisplayName,
				Severity:       model.SeverityPass,
				ScoreDeduction: 0,
				ObservedValue:  obs,
				Recommendation: "Consider specifying explicit boundaries for this AI scraper.",
				Remediation:    fmt.Sprintf("User-agent: %s\nDisallow: /", bot.Aliases[0]),
			})
		}
	}

	// Overall AI crawler governance summary finding
	if explicitCount == 0 && (wildcardPolicy == nil || len(wildcardPolicy.Disallows) == 0) {
		findings = append(findings, model.FindingItem{
			ID:             "ai-bots-unmanaged",
			Domain:         string(model.DomainAIGovernance),
			CheckName:      "AI Scraper Governance Posture",
			Severity:       model.SeverityMedium,
			ScoreDeduction: 5,
			ObservedValue:  "Zero AI crawler directives detected in robots.txt",
			Recommendation: "Specify explicit directives for autonomous scrapers (e.g. GPTBot, ClaudeBot, PerplexityBot) to protect intellectual property.",
			Remediation:    "User-agent: GPTBot\nDisallow: /\n\nUser-agent: ClaudeBot\nDisallow: /\n\nUser-agent: PerplexityBot\nDisallow: /",
		})
	} else {
		findings = append(findings, model.FindingItem{
			ID:             "ai-bots-managed",
			Domain:         string(model.DomainAIGovernance),
			CheckName:      "AI Scraper Governance Posture",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  fmt.Sprintf("Robots.txt contains crawler management directives (%d explicit AI policies)", explicitCount),
			Recommendation: "Continue reviewing autonomous AI crawler policies as new agents emerge.",
			Remediation:    "",
		})
	}

	return findings, nil
}

func formatBotPolicy(p *BotPolicy) string {
	if len(p.Disallows) == 1 && p.Disallows[0] == "/" {
		return "Disallow: / (Fully Blocked)"
	}
	if len(p.Disallows) > 0 {
		return fmt.Sprintf("Restricted (%d disallow paths)", len(p.Disallows))
	}
	if len(p.Allows) > 0 {
		return "Explicitly Allowed"
	}
	return "Active policy declaration"
}

func auditLLMsTXT(ctx context.Context, client *http.Client, targetURL *url.URL) model.FindingItem {
	candidatePaths := []string{"/llms.txt", "/.well-known/llms.txt"}

	for _, p := range candidatePaths {
		u := *targetURL
		u.Path = p
		u.RawQuery = ""

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "GoodAI-Security-Audit/1.0 (+https://goodai.co.nz)")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			content := string(bodyBytes)
			lower := strings.ToLower(content)
			if !strings.Contains(lower, "<html") && !strings.Contains(lower, "<!doctype") && len(strings.TrimSpace(content)) > 0 {
				return model.FindingItem{
					ID:             "ai-llms-txt",
					Domain:         string(model.DomainAIGovernance),
					CheckName:      "AI Context Standard (llms.txt)",
					Severity:       model.SeverityPass,
					ScoreDeduction: 0,
					ObservedValue:  fmt.Sprintf("Present at %s (%d bytes verified)", p, len(content)),
					Recommendation: "Maintain curated markdown documentation for autonomous LLM agents.",
					Remediation:    "",
				}
			}
		}
	}

	return model.FindingItem{
		ID:             "ai-llms-txt",
		Domain:         string(model.DomainAIGovernance),
		CheckName:      "AI Context Standard (llms.txt)",
		Severity:       model.SeverityLow,
		ScoreDeduction: 2,
		ObservedValue:  "HTTP 404 (Missing llms.txt standard)",
		Recommendation: "Publish an /llms.txt markdown file to provide structured context for LLM search engines and AI agents.",
		Remediation:    "# Project Context for LLMs\n> Core documentation and API reference for autonomous AI search.\n\n- [API Specification](https://example.com/api): REST API endpoints\n- [Security Guide](https://example.com/security): Security architecture",
	}
}

func auditSecurityTXT(ctx context.Context, client *http.Client, targetURL *url.URL) model.FindingItem {
	candidatePaths := []string{"/.well-known/security.txt", "/security.txt"}

	for _, p := range candidatePaths {
		u := *targetURL
		u.Path = p
		u.RawQuery = ""

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "GoodAI-Security-Audit/1.0 (+https://goodai.co.nz)")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			content := string(bodyBytes)
			lower := strings.ToLower(content)
			if !strings.Contains(lower, "<html") && !strings.Contains(lower, "<!doctype") {
				if strings.Contains(lower, "contact:") {
					obs := fmt.Sprintf("RFC 9116 compliant at %s", p)
					if strings.Contains(lower, "expires:") {
						obs += " (Contact and Expires directives active)"
					} else {
						obs += " (Contact directive active)"
					}
					return model.FindingItem{
						ID:             "ai-security-txt",
						Domain:         string(model.DomainAIGovernance),
						CheckName:      "Coordinated Vulnerability Disclosure (security.txt)",
						Severity:       model.SeverityPass,
						ScoreDeduction: 0,
						ObservedValue:  obs,
						Recommendation: "Ensure security.txt expiry dates are updated annually.",
						Remediation:    "",
					}
				}
			}
		}
	}

	return model.FindingItem{
		ID:             "ai-security-txt",
		Domain:         string(model.DomainAIGovernance),
		CheckName:      "Coordinated Vulnerability Disclosure (security.txt)",
		Severity:       model.SeverityLow,
		ScoreDeduction: 2,
		ObservedValue:  "HTTP 404 (Missing security.txt RFC 9116)",
		Recommendation: "Publish /.well-known/security.txt to specify vulnerability disclosure contacts and cryptographic keys per RFC 9116.",
		Remediation:    "Contact: mailto:security@example.com\nExpires: 2027-12-31T23:59:59.000Z\nPreferred-Languages: en",
	}
}
