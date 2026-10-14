package security

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/Zachary-du/goodai-monitor/internal/model"
)

const reportHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>GoodAI Executive Security Audit Report - {{.TargetHost}}</title>
  <style>
    :root {
      --bg: #0b0f19;
      --card-bg: #111827;
      --card-border: #1f2937;
      --text: #f9fafb;
      --text-muted: #9ca3af;
      --primary: #2563eb;
      --success: #10b981;
      --warning: #f59e0b;
      --danger: #ef4444;
      --info: #06b6d4;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      background-color: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 2.5rem 1.5rem;
    }
    .report-container {
      max-width: 1040px;
      margin: 0 auto;
    }
    .report-header {
      border-bottom: 2px solid var(--card-border);
      padding-bottom: 1.5rem;
      margin-bottom: 2rem;
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      flex-wrap: wrap;
      gap: 1rem;
    }
    .brand-title {
      font-size: 1.5rem;
      font-weight: 800;
      letter-spacing: -0.02em;
      color: #ffffff;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .brand-dot {
      width: 12px;
      height: 12px;
      border-radius: 50%;
      background-color: var(--success);
      display: inline-block;
      box-shadow: 0 0 10px rgba(16, 185, 129, 0.6);
    }
    .report-subtitle {
      font-size: 0.8125rem;
      color: var(--text-muted);
      margin-top: 0.25rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }
    .confidential-tag {
      background: rgba(239, 68, 68, 0.15);
      border: 1px solid rgba(239, 68, 68, 0.4);
      color: #fca5a5;
      font-size: 0.75rem;
      font-weight: 700;
      padding: 0.35rem 0.75rem;
      border-radius: 9999px;
      letter-spacing: 0.05em;
    }
    .meta-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 1rem;
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 0.75rem;
      padding: 1.25rem;
      margin-bottom: 2rem;
    }
    .meta-item { display: flex; flex-direction: column; }
    .meta-label { font-size: 0.75rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
    .meta-val { font-size: 0.9375rem; font-weight: 600; color: #ffffff; word-break: break-all; margin-top: 0.25rem; }

    .scorecard {
      display: grid;
      grid-template-columns: 280px 1fr;
      gap: 1.5rem;
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 0.75rem;
      padding: 1.5rem;
      margin-bottom: 2rem;
      page-break-inside: avoid;
      break-inside: avoid;
    }
    @media (max-width: 768px) {
      .scorecard { grid-template-columns: 1fr; }
    }
    .gauge-card {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      background: rgba(0, 0, 0, 0.25);
      border: 1px solid var(--card-border);
      border-radius: 0.5rem;
      padding: 1.5rem;
      text-align: center;
    }
    .grade-badge {
      font-size: 3rem;
      font-weight: 900;
      line-height: 1;
      margin-bottom: 0.5rem;
    }
    .grade-A-plus, .grade-A { color: var(--success); }
    .grade-B { color: var(--info); }
    .grade-C { color: var(--warning); }
    .grade-F { color: var(--danger); }

    .score-number {
      font-size: 1.25rem;
      font-weight: 700;
      color: #ffffff;
    }
    .grade-label {
      font-size: 0.8125rem;
      color: var(--text-muted);
      margin-top: 0.25rem;
    }

    .domain-list {
      display: flex;
      flex-direction: column;
      gap: 1rem;
      justify-content: center;
    }
    .domain-item {
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
    }
    .domain-item-top {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: 0.875rem;
      font-weight: 600;
    }
    .domain-bar-bg {
      background: #1f2937;
      height: 8px;
      border-radius: 9999px;
      overflow: hidden;
    }
    .domain-bar-fill {
      height: 100%;
      border-radius: 9999px;
      background: linear-gradient(90deg, #2563eb, #10b981);
    }
    .domain-subtext {
      font-size: 0.75rem;
      color: var(--text-muted);
    }

    .section-title {
      font-size: 1.125rem;
      font-weight: 700;
      margin-bottom: 1rem;
      color: #ffffff;
      border-left: 4px solid var(--primary);
      padding-left: 0.75rem;
    }
    .summary-box {
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 0.75rem;
      padding: 1.25rem;
      font-size: 0.9375rem;
      color: #e5e7eb;
      margin-bottom: 2rem;
      line-height: 1.6;
    }

    .table-container {
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 0.75rem;
      overflow-x: auto;
      margin-bottom: 2rem;
      page-break-inside: avoid;
      break-inside: avoid;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      text-align: left;
      font-size: 0.8125rem;
    }
    th {
      background-color: #1f2937;
      color: #d1d5db;
      font-weight: 600;
      padding: 0.75rem 1rem;
      text-transform: uppercase;
      font-size: 0.6875rem;
      letter-spacing: 0.05em;
      border-bottom: 1px solid var(--card-border);
    }
    td {
      padding: 0.875rem 1rem;
      border-bottom: 1px solid rgba(255, 255, 255, 0.05);
      color: #e5e7eb;
      vertical-align: top;
    }
    tr:last-child td { border-bottom: none; }
    .badge {
      display: inline-block;
      padding: 0.2rem 0.5rem;
      border-radius: 0.25rem;
      font-size: 0.6875rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }
    .badge-PASS { background: rgba(16, 185, 129, 0.2); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.4); }
    .badge-LOW { background: rgba(148, 163, 184, 0.2); color: #cbd5e1; border: 1px solid rgba(148, 163, 184, 0.4); }
    .badge-MEDIUM { background: rgba(245, 158, 11, 0.2); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.4); }
    .badge-HIGH { background: rgba(249, 115, 22, 0.2); color: #fb923c; border: 1px solid rgba(249, 115, 22, 0.4); }
    .badge-CRITICAL { background: rgba(239, 68, 68, 0.25); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.5); }

    .remediation-card {
      background-color: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 0.75rem;
      padding: 1.25rem;
      margin-bottom: 1.25rem;
      page-break-inside: avoid;
      break-inside: avoid;
    }
    .remediation-title {
      font-size: 0.9375rem;
      font-weight: 700;
      color: #ffffff;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 0.5rem;
    }
    .remediation-desc {
      font-size: 0.8125rem;
      color: var(--text-muted);
      margin-bottom: 0.75rem;
    }
    pre {
      background: #030712;
      border: 1px solid #1f2937;
      border-radius: 0.375rem;
      padding: 0.75rem 1rem;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 0.75rem;
      color: #a5f3fc;
      overflow-x: auto;
      white-space: pre-wrap;
      word-break: break-all;
    }

    .footer-attestation {
      margin-top: 3rem;
      border-top: 1px solid var(--card-border);
      padding-top: 1.5rem;
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      flex-wrap: wrap;
      gap: 1rem;
      font-size: 0.75rem;
      color: var(--text-muted);
    }
    .btn-print {
      background-color: var(--primary);
      color: #ffffff;
      border: none;
      padding: 0.5rem 1rem;
      font-size: 0.8125rem;
      font-weight: 600;
      border-radius: 0.375rem;
      cursor: pointer;
    }

    /* Print styles for crisp PDF export */
    @media print {
      body {
        background-color: #ffffff !important;
        color: #000000 !important;
        padding: 0 !important;
      }
      .no-print { display: none !important; }
      .brand-title, .section-title, .meta-val, .score-number, .remediation-title {
        color: #000000 !important;
      }
      .meta-grid, .scorecard, .summary-box, .table-container, .remediation-card {
        background-color: #ffffff !important;
        border: 1px solid #d1d5db !important;
        color: #000000 !important;
        box-shadow: none !important;
      }
      th {
        background-color: #f3f4f6 !important;
        color: #111827 !important;
        border-bottom: 1px solid #d1d5db !important;
      }
      td {
        color: #111827 !important;
        border-bottom: 1px solid #e5e7eb !important;
      }
      pre {
        background-color: #f8fafc !important;
        border: 1px solid #e2e8f0 !important;
        color: #0f172a !important;
      }
      .badge-PASS { background: #f0fdf4 !important; color: #166534 !important; border: 1px solid #bbf7d0 !important; }
      .badge-LOW { background: #f8fafc !important; color: #334155 !important; border: 1px solid #cbd5e1 !important; }
      .badge-MEDIUM { background: #fffbeb !important; color: #92400e !important; border: 1px solid #fde68a !important; }
      .badge-HIGH { background: #fff7ed !important; color: #9a3412 !important; border: 1px solid #fed7aa !important; }
      .badge-CRITICAL { background: #fef2f2 !important; color: #991b1b !important; border: 1px solid #fecaca !important; }
      .domain-bar-bg { background: #e5e7eb !important; }
      .domain-bar-fill { background: #2563eb !important; }
    }
  </style>
</head>
<body>
  <div class="report-container">
    <header class="report-header">
      <div>
        <div class="brand-title">
          <span class="brand-dot"></span>
          GoodAI Monitor &bull; Executive Security Report
        </div>
        <div class="report-subtitle">Synthetic Attack Surface &amp; Autonomous AI Crawler Audit</div>
      </div>
      <div style="display: flex; gap: 0.5rem; align-items: center;">
        <span class="confidential-tag">CONFIDENTIAL // CISO BRIEFING</span>
        <button class="btn-print no-print" onclick="window.print()">Print / Export PDF</button>
      </div>
    </header>

    <div class="meta-grid">
      <div class="meta-item">
        <span class="meta-label">Target Hostname</span>
        <span class="meta-val">{{.TargetHost}}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Target URL</span>
        <span class="meta-val">{{.TargetURL}}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Resolved IP Address</span>
        <span class="meta-val">{{.TargetIP}}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Audit Timestamp</span>
        <span class="meta-val">{{.CreatedAtFormatted}}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Verification Report ID</span>
        <span class="meta-val">sec-rpt-{{.ID}}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Inspection Duration</span>
        <span class="meta-val">{{.ExecutionMs}} ms</span>
      </div>
    </div>

    <div class="scorecard">
      <div class="gauge-card">
        <div class="grade-badge grade-{{.GradeSlug}}">{{.Grade}}</div>
        <div class="score-number">{{.CompositeScore}} / 100</div>
        <div class="grade-label">{{.GradeLabel}}</div>
      </div>
      <div class="domain-list">
        {{range .DomainScores}}
        <div class="domain-item">
          <div class="domain-item-top">
            <span>{{.DomainName}}</span>
            <span>{{.Score}} / {{.MaxScore}} pts ({{.Status}})</span>
          </div>
          <div class="domain-bar-bg">
            <div class="domain-bar-fill" style="width: {{domainPercentage .Score .MaxScore}}%;"></div>
          </div>
          <div class="domain-subtext">{{.Summary}}</div>
        </div>
        {{end}}
      </div>
    </div>

    <h2 class="section-title">1. Executive Summary &amp; Posture Assessment</h2>
    <div class="summary-box">
      {{.ExecutiveSummary}}
    </div>

    <h2 class="section-title">2. Comprehensive Vector Findings Register</h2>
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th style="width: 110px;">Severity</th>
            <th style="width: 170px;">Domain</th>
            <th style="width: 220px;">Vector Check</th>
            <th>Observed Result &amp; Impact</th>
          </tr>
        </thead>
        <tbody>
          {{range .Findings}}
          <tr>
            <td><span class="badge badge-{{.Severity}}">{{.Severity}}</span></td>
            <td>{{formatDomain .Domain}}</td>
            <td><strong>{{.CheckName}}</strong></td>
            <td>
              <div>{{.ObservedValue}}</div>
              {{if .Recommendation}}
              <div style="font-size: 0.75rem; color: var(--text-muted); margin-top: 0.25rem;">
                Recommendation: {{.Recommendation}}
              </div>
              {{end}}
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
    </div>

    <h2 class="section-title">3. Strategic Remediation Roadmap</h2>
    {{$hasRemediations := false}}
    {{range .Findings}}
      {{if and (ne .Severity "PASS") (ne .Remediation "")}}
        {{$hasRemediations = true}}
        <div class="remediation-card">
          <div class="remediation-title">
            <span class="badge badge-{{.Severity}}">{{.Severity}}</span>
            <span>{{.CheckName}}</span>
          </div>
          <div class="remediation-desc">{{.Recommendation}}</div>
          <pre><code>{{.Remediation}}</code></pre>
        </div>
      {{end}}
    {{end}}
    {{if not $hasRemediations}}
    <div class="summary-box" style="margin-bottom: 2rem;">
      No outstanding remediation items detected. The audited infrastructure conforms to hardened defensive baselines.
    </div>
    {{end}}

    <footer class="footer-attestation">
      <div>
        <div><strong>GoodAI Autonomous Security Engine v1.0</strong> &bull; SSRF-Protected Zero-Agent Probing</div>
        <div style="margin-top: 0.25rem;">Attestation SHA256: <code>{{.Attestation}}</code></div>
      </div>
      <div style="text-align: right;">
        <div>Standard: GoodAI Web Defense Matrix RFC-Compliant</div>
        <div style="margin-top: 0.25rem;">Confidential Document &bull; Generated for Authorized Operators</div>
      </div>
    </footer>
  </div>
</body>
</html>`

type templateData struct {
	*model.SecurityAuditReport
	CreatedAtFormatted string
	GradeSlug          string
	ExecutiveSummary   template.HTML
}

// GenerateExecutiveHTMLReport renders a self-contained, standalone HTML report for the audit.
func GenerateExecutiveHTMLReport(report *model.SecurityAuditReport) (string, error) {
	funcMap := template.FuncMap{
		"domainPercentage": func(score, maxScore int) int {
			if maxScore <= 0 {
				return 0
			}
			pct := (score * 100) / maxScore
			if pct > 100 {
				return 100
			}
			return pct
		},
		"formatDomain": func(d string) string {
			switch d {
			case string(model.DomainAIGovernance):
				return "AI Bot Governance"
			case string(model.DomainHeaders):
				return "Security Headers"
			case string(model.DomainTLS):
				return "TLS & Transport"
			case string(model.DomainExposure):
				return "Surface Exposure"
			default:
				return d
			}
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(reportHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse report template: %w", err)
	}

	slug := strings.ReplaceAll(report.Grade, "+", "-plus")

	summaryText := generateExecutiveSummary(report)

	data := templateData{
		SecurityAuditReport: report,
		CreatedAtFormatted:  report.CreatedAt.Format("2006-01-02 15:04:05 UTC"),
		GradeSlug:           slug,
		ExecutiveSummary:    template.HTML(summaryText),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute report template: %w", err)
	}

	return buf.String(), nil
}

func generateExecutiveSummary(report *model.SecurityAuditReport) string {
	var criticalCount, highCount, medCount, lowCount int
	for _, f := range report.Findings {
		switch f.Severity {
		case model.SeverityCritical:
			criticalCount++
		case model.SeverityHigh:
			highCount++
		case model.SeverityMedium:
			medCount++
		case model.SeverityLow:
			lowCount++
		}
	}

	hostEscaped := template.HTMLEscapeString(report.TargetHost)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("The target infrastructure (<strong>%s</strong>) was subjected to an autonomous multi-vector security assessment, achieving a composite score of <strong>%d/100 (Grade %s)</strong> with an operational posture classification of <strong>%s</strong>. ",
		hostEscaped, report.CompositeScore, report.Grade, report.GradeLabel))

	if criticalCount > 0 || highCount > 0 {
		sb.WriteString(fmt.Sprintf("Immediate strategic remediation is advised: <strong>%d Critical</strong> and <strong>%d High-severity</strong> findings were identified. ",
			criticalCount, highCount))
	} else if medCount > 0 {
		sb.WriteString(fmt.Sprintf("The cryptographic baseline is stable with <strong>0 Critical</strong> vulnerabilities; however, <strong>%d Medium-severity</strong> hardening opportunities were recorded. ",
			medCount))
	} else {
		sb.WriteString("All primary threat vectors are hardened with optimal transport cryptography, active defense headers, and controlled exposure boundaries. ")
	}

	if lowCount > 0 {
		sb.WriteString(fmt.Sprintf("Additionally, %d Low-severity optimizations (e.g. AI crawler context standard or vulnerability disclosure metadata) are available in the remediation roadmap.", lowCount))
	}

	return sb.String()
}
