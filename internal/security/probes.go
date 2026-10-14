package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

var (
	envVarRegex = regexp.MustCompile(`(?m)^[A-Z0-9_]{3,}=.+`)
	gitShaRegex = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// AuditExposures runs SSRF-safe HTTP probes to detect leakage of sensitive files or dev interfaces.
func AuditExposures(ctx context.Context, client *http.Client, targetURL *url.URL) ([]model.FindingItem, error) {
	if client == nil {
		client = NewSafeHTTPClient(DefaultCheckTimeout)
	}

	var findings []model.FindingItem

	// 1. Probe /.env
	envFinding := probeEnvFile(ctx, client, targetURL)
	findings = append(findings, envFinding)

	// 2. Probe /.git/HEAD
	gitFinding := probeGitHEAD(ctx, client, targetURL)
	findings = append(findings, gitFinding)

	// 3. Probe /openapi.json & /swagger.json
	openapiFinding := probeOpenAPISpec(ctx, client, targetURL)
	findings = append(findings, openapiFinding)

	// 4. Probe /swagger & /swagger/index.html
	swaggerFinding := probeSwaggerUI(ctx, client, targetURL)
	findings = append(findings, swaggerFinding)

	return findings, nil
}

func probeEnvFile(ctx context.Context, client *http.Client, targetURL *url.URL) model.FindingItem {
	u := *targetURL
	u.Path = "/.env"
	u.RawQuery = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return model.FindingItem{
			ID:             "exp-env-file",
			Domain:         string(model.DomainExposure),
			CheckName:      "Environment Configuration Exposure (/.env)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  "Not exposed (Request error)",
			Recommendation: "Environment files remain inaccessible.",
			Remediation:    "",
		}
	}
	req.Header.Set("User-Agent", "GoodAI-Security-Audit/1.0 (+https://goodai.co.nz)")

	resp, err := client.Do(req)
	if err != nil {
		return model.FindingItem{
			ID:             "exp-env-file",
			Domain:         string(model.DomainExposure),
			CheckName:      "Environment Configuration Exposure (/.env)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  "Not exposed (Connection refused or safe status)",
			Recommendation: "Environment files remain inaccessible.",
			Remediation:    "",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
		bodyStr := string(bodyBytes)
		lower := strings.ToLower(bodyStr)

		// Strict anti-false-positive check: Must NOT be an HTML error page and MUST match KEY=VAL pattern
		if !strings.Contains(lower, "<html") && !strings.Contains(lower, "<!doctype") && envVarRegex.MatchString(bodyStr) {
			return model.FindingItem{
				ID:             "exp-env-file",
				Domain:         string(model.DomainExposure),
				CheckName:      "Environment Configuration Exposure (/.env)",
				Severity:       model.SeverityCritical,
				ScoreDeduction: 20,
				ObservedValue:  "CRITICAL: Active environment configuration file exposed to the public",
				Recommendation: "Immediately restrict HTTP access to all dotfiles (.env, .git) in web server configurations.",
				Remediation:    "location ~ /\\.(?!well-known) {\n    deny all;\n    return 404;\n}",
			}
		}
	}

	return model.FindingItem{
		ID:             "exp-env-file",
		Domain:         string(model.DomainExposure),
		CheckName:      "Environment Configuration Exposure (/.env)",
		Severity:       model.SeverityPass,
		ScoreDeduction: 0,
		ObservedValue:  fmt.Sprintf("HTTP %d (Safe - No environment variables detected)", resp.StatusCode),
		Recommendation: "Environment files remain protected.",
		Remediation:    "",
	}
}

func probeGitHEAD(ctx context.Context, client *http.Client, targetURL *url.URL) model.FindingItem {
	u := *targetURL
	u.Path = "/.git/HEAD"
	u.RawQuery = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return model.FindingItem{
			ID:             "exp-git-repo",
			Domain:         string(model.DomainExposure),
			CheckName:      "Git Repository Metadata (/.git/HEAD)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  "Not exposed",
			Recommendation: "Source control metadata remains inaccessible.",
			Remediation:    "",
		}
	}
	req.Header.Set("User-Agent", "GoodAI-Security-Audit/1.0 (+https://goodai.co.nz)")

	resp, err := client.Do(req)
	if err != nil {
		return model.FindingItem{
			ID:             "exp-git-repo",
			Domain:         string(model.DomainExposure),
			CheckName:      "Git Repository Metadata (/.git/HEAD)",
			Severity:       model.SeverityPass,
			ScoreDeduction: 0,
			ObservedValue:  "Not exposed",
			Recommendation: "Source control metadata remains inaccessible.",
			Remediation:    "",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
		bodyStr := strings.TrimSpace(string(bodyBytes))

		// Strict anti-false-positive check: Must begin with ref: refs/ or be a 40-char SHA commit
		if strings.HasPrefix(bodyStr, "ref: refs/") || gitShaRegex.MatchString(bodyStr) {
			return model.FindingItem{
				ID:             "exp-git-repo",
				Domain:         string(model.DomainExposure),
				CheckName:      "Git Repository Metadata (/.git/HEAD)",
				Severity:       model.SeverityCritical,
				ScoreDeduction: 20,
				ObservedValue:  "CRITICAL: Git version control metadata (HEAD) is publicly exposed",
				Recommendation: "Block all HTTP requests to .git directories in web server configurations.",
				Remediation:    "location ~ /\\.git {\n    deny all;\n    return 404;\n}",
			}
		}
	}

	return model.FindingItem{
		ID:             "exp-git-repo",
		Domain:         string(model.DomainExposure),
		CheckName:      "Git Repository Metadata (/.git/HEAD)",
		Severity:       model.SeverityPass,
		ScoreDeduction: 0,
		ObservedValue:  fmt.Sprintf("HTTP %d (Safe - Git metadata not exposed)", resp.StatusCode),
		Recommendation: "Version control metadata is not publicly accessible.",
		Remediation:    "",
	}
}

func probeOpenAPISpec(ctx context.Context, client *http.Client, targetURL *url.URL) model.FindingItem {
	candidatePaths := []string{"/openapi.json", "/swagger.json"}

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
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
			var doc map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &doc); err == nil {
				if _, hasOpenAPI := doc["openapi"]; hasOpenAPI {
					return model.FindingItem{
						ID:             "exp-openapi-spec",
						Domain:         string(model.DomainExposure),
						CheckName:      "API Specification Contract (/openapi.json)",
						Severity:       model.SeverityMedium,
						ScoreDeduction: 5,
						ObservedValue:  fmt.Sprintf("Exposed OpenAPI JSON contract at %s", p),
						Recommendation: "Restrict public exposure of internal API specifications to authenticated users.",
						Remediation:    "location " + p + " {\n    auth_basic \"Internal Docs\";\n    auth_basic_user_file /etc/nginx/.htpasswd;\n}",
					}
				}
				if _, hasSwagger := doc["swagger"]; hasSwagger {
					return model.FindingItem{
						ID:             "exp-openapi-spec",
						Domain:         string(model.DomainExposure),
						CheckName:      "API Specification Contract (/openapi.json)",
						Severity:       model.SeverityMedium,
						ScoreDeduction: 5,
						ObservedValue:  fmt.Sprintf("Exposed Swagger 2.0 JSON contract at %s", p),
						Recommendation: "Restrict public exposure of internal API specifications to authenticated users.",
						Remediation:    "location " + p + " {\n    auth_basic \"Internal Docs\";\n    auth_basic_user_file /etc/nginx/.htpasswd;\n}",
					}
				}
			}
		}
	}

	return model.FindingItem{
		ID:             "exp-openapi-spec",
		Domain:         string(model.DomainExposure),
		CheckName:      "API Specification Contract (/openapi.json)",
		Severity:       model.SeverityPass,
		ScoreDeduction: 0,
		ObservedValue:  "Not publicly exposed (Clean)",
		Recommendation: "API contracts remain protected.",
		Remediation:    "",
	}
}

func probeSwaggerUI(ctx context.Context, client *http.Client, targetURL *url.URL) model.FindingItem {
	candidatePaths := []string{"/swagger", "/swagger/index.html"}

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
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
			lower := strings.ToLower(string(bodyBytes))
			if strings.Contains(lower, "swagger-ui") || strings.Contains(lower, "swagger ui") {
				return model.FindingItem{
					ID:             "exp-swagger-ui",
					Domain:         string(model.DomainExposure),
					CheckName:      "Interactive Swagger UI (/swagger)",
					Severity:       model.SeverityMedium,
					ScoreDeduction: 5,
					ObservedValue:  fmt.Sprintf("Publicly accessible Swagger UI at %s", p),
					Recommendation: "Place interactive API testing consoles behind IP whitelisting or authentication.",
					Remediation:    "location /swagger {\n    allow 10.0.0.0/8;\n    deny all;\n}",
				}
			}
		}
	}

	return model.FindingItem{
		ID:             "exp-swagger-ui",
		Domain:         string(model.DomainExposure),
		CheckName:      "Interactive Swagger UI (/swagger)",
		Severity:       model.SeverityPass,
		ScoreDeduction: 0,
		ObservedValue:  "Not publicly accessible (Clean)",
		Recommendation: "Interactive API consoles remain protected.",
		Remediation:    "",
	}
}
