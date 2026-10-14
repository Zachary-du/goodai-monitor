# Product Prototype: Web Security & AI Defense Audit

> **Feature Name**: Web Security & AI Defense Audit  
> **Status**: PROTOTYPE / SPECIFICATION  
> **Target Audience**: Cloud Engineers, DevOps, Security Teams, AI Platform Architects  
> **Owner**: PM Agent & Architect  
> **Related Task**: [`TASK-008: Web Security & AI Defense Audit with Standalone Executive Report`](../ai-team/tasks/TASK-008-security-audit-and-ai-defense-report.md)

---

## 1. Executive Summary & Value Proposition

Modern web infrastructure faces an evolving dual threat landscape:
1. **Traditional Web & Transport Deficiencies**: Missing cryptographic hardening, deficient HTTP security headers, outdated TLS implementations, and unintended exposure of environment files (`.env`) or Git metadata (`.git/HEAD`).
2. **Autonomous AI Crawler & Scraper Proliferation**: Unchecked automated scraping by LLM data scrapers and search bots (e.g., `GPTBot`, `ClaudeBot`, `PerplexityBot`, `Bytespider`), lack of AI scraping governance (`llms.txt`), and absence of coordinated security disclosure policies (`security.txt`).

**GoodAI Monitor's "Web Security & AI Defense Audit"** bridges this gap. It provides a lightweight, zero-agent, SSRF-safe diagnostic engine that executes comprehensive multi-vector security inspections in seconds.

Results are presented directly within the authenticated Operations Cockpit with real-time visual score gauges and severity-tagged findings, and can be exported into an **Executive Security Report**—a standalone, self-contained, print/PDF-optimized HTML document designed for CISO briefings and compliance documentation.

```
+-----------------------------------------------------------------------------------------+
|                               GoodAI Security Audit Engine                              |
+-----------------------------------------------------------------------------------------+
|                                                                                         |
|   +-----------------------+   +-----------------------+   +-----------------------+     |
|   | 1. AI Governance      |   | 2. Security Headers   |   | 3. Transport & TLS    |     |
|   | - robots.txt AI bots  |   | - CSP & HSTS          |   | - TLS 1.3 protocol    |     |
|   | - llms.txt context    |   | - X-Frame & Nosniff   |   | - Cert days remaining |     |
|   | - security.txt RFC    |   | - Referrer Policy     |   | - SAN / Issuer CA     |     |
|   +-----------+-----------+   +-----------+-----------+   +-----------+-----------+     |
|               |                           |                           |                 |
|               +---------------------+     |     +---------------------+                 |
|                                     |     |     |                                       |
|                                     v     v     v                                       |
|                       +-------------------------------+                                 |
|                       | 4. Sensitive Exposure Probes  |                                 |
|                       | - /.env, /.git/HEAD           |                                 |
|                       | - /swagger, /openapi.json     |                                 |
|                       | - SSRF SafeDialer Shield      |                                 |
|                       +---------------+---------------+                                 |
|                                       |                                                 |
|                                       v                                                 |
|                       +-------------------------------+                                 |
|                       | 5. Scoring & Grading (0-100)  |                                 |
|                       | Grades: A+, A, B, C, F        |                                 |
|                       +---------------+---------------+                                 |
|                                       |                                                 |
|                 +---------------------+---------------------+                           |
|                 v                                           v                           |
|   +---------------------------+               +---------------------------+             |
|   | Interactive Cockpit UI    |               | Standalone Exec Report    |             |
|   | - Live score gauge        |               | - Embedded CSS / SVG      |             |
|   | - Domain radar bars       |               | - High-contrast theme     |             |
|   | - Remediation code copy   |               | - Print / PDF ready       |             |
|   +---------------------------+               +---------------------------+             |
|                                                                                         |
+-----------------------------------------------------------------------------------------+
```

---

## 2. Architectural Pillars & Inspection Modules

### 2.1 Pillar 1: AI Bot & Scraper Governance Prober
Autonomous LLM crawlers consume vast network bandwidth and scrape proprietary knowledge bases. This module evaluates whether a web service has established clear machine-readable boundaries:
- **`robots.txt` AI Crawler Inspection**:
  - Parses standard user-agent directives targeting prominent AI bots:
    - `GPTBot` (OpenAI training crawler)
    - `ClaudeBot` / `anthropic-ai` (Anthropic model training)
    - `PerplexityBot` (Perplexity search and extraction)
    - `Google-Extended` (Google Gemini training crawler)
    - `Bytespider` (ByteDance web crawler)
    - `CCBot` (Common Crawl archive bot)
  - Distinguishes between explicit allow policies, explicit disallow policies, and unmanaged/default states.
- **`llms.txt` Context Standard**:
  - Checks `/llms.txt` and `/.well-known/llms.txt` according to emerging LLM context standards.
  - Verifies if structured content is curated specifically for AI agents and LLM search summaries.
- **`security.txt` Vulnerability Disclosure**:
  - Validates `/.well-known/security.txt` and `/security.txt` according to RFC 9116.
  - Inspects required `Contact:` and `Expires:` directives for coordinated vulnerability reporting.

### 2.2 Pillar 2: HTTP Security Headers Posture
Verifies that reverse proxies and web application servers enforce browser defenses against Cross-Site Scripting (XSS), Clickjacking, MIME-confusion, and Man-in-the-Middle (MITM) attacks:
- **Content-Security-Policy (CSP)**:
  - Verifies presence of `Content-Security-Policy`.
  - Assesses whether `default-src` or `script-src` restricts arbitrary script execution (flags `unsafe-inline` or wildcard `*`).
- **Strict-Transport-Security (HSTS)**:
  - Verifies presence of `Strict-Transport-Security`.
  - Enforces `max-age` >= 15,768,000 (minimum 6 months, recommended 1 year = 31,536,000).
  - Rewards `includeSubDomains` and `preload` directives.
- **X-Frame-Options**:
  - Verifies setting to `DENY` or `SAMEORIGIN` to mitigate framing and clickjacking attacks.
- **X-Content-Type-Options**:
  - Enforces `nosniff` to prevent browsers from MIME-sniffing response bodies.
- **Referrer-Policy**:
  - Checks for modern privacy-preserving policies (`strict-origin-when-cross-origin`, `no-referrer`, or `same-origin`).
- **Permissions-Policy**:
  - Evaluates restriction of sensitive client APIs (`camera=(), microphone=(), geolocation=()`).

### 2.3 Pillar 3: Transport & TLS Cryptographic Posture
Evaluates the cryptographic boundary of HTTPS endpoints:
- **TLS Protocol Assertion**:
  - TLS 1.3: Optimal posture (100% domain score).
  - TLS 1.2: Strong legacy compatibility (80% domain score).
  - TLS 1.0 / 1.1 or Plaintext HTTP: CRITICAL security failure (0% domain score).
- **Certificate Validity Duration**:
  - Remaining days > 30: PASS (Normal operational window).
  - Remaining days 15–30: MEDIUM WARNING (Expiring soon).
  - Remaining days < 15: HIGH SEVERITY (Urgent renewal required).
  - Expired / Self-Signed: CRITICAL SEVERITY.
- **Certificate Authority & SAN Verification**:
  - Validates Certificate Issuer (e.g., Let's Encrypt, Cloudflare Inc ECC CA, DigiCert).
  - Verifies Subject Alternative Names (SAN) match the probed domain or wildcard.
  - Assesses negotiated cipher suites (P-256 / X25519, AES-128-GCM, AES-256-GCM, CHACHA20-POLY1305).

### 2.4 Pillar 4: SSRF-Safe Sensitive Exposure Probes
Detects accidental leakage of operational secrets, version control artifacts, or unguarded developer interfaces without compromising internal network boundaries:
- **Probed Sensitive Endpoints**:
  1. `/.env` (leaked database credentials, API tokens, cloud keys)
  2. `/.git/HEAD` (exposed Git repository root)
  3. `/swagger` / `/swagger/index.html` (exposed interactive API documentation)
  4. `/openapi.json` / `/swagger.json` (machine-readable internal API contracts)
  5. `/.dockerenv` / `/docker-compose.yml` (exposed container definitions)
- **Strict Anti-False-Positive Content Pattern Matching**:
  - Probes do not rely solely on HTTP 200 status codes (many single-page apps return HTTP 200 with HTML error shells for missing routes).
  - `/.env` is only flagged if the body contains regex `^[A-Z0-9_]{3,}=.+` without HTML `<html` tags.
  - `/.git/HEAD` is only flagged if the body begins with `ref: refs/` or is a 40-character hex commit SHA.
  - `/openapi.json` is only flagged if valid JSON with `"openapi"` or `"swagger"` keys.
- **SSRF Safety & Resource Bounding**:
  - **All requests must pass through `security.SafeDialer` / `security.SafeDialContext`**:
    - Disallows loopback (`127.0.0.0/8`, `::1`), private subnets (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), and Cloud IMDS (`169.254.169.254`).
    - Eliminates DNS Rebinding (TOCTOU) by locking the verified IP for socket dialing.
    - Prevents redirect hops to internal infrastructure (`CheckRedirect: http.ErrUseLastResponse`).
  - **Streaming Read Limit**: Capped at 16 KiB (`io.LimitReader(resp.Body, 16384)`) to prevent memory exhaustion DoS from large files.

---

## 3. Composite Security Score Algorithm & Grading Rubric

### 3.1 Domain Categories & Weight Distribution
The composite score is calculated on a 0 to 100 integer scale across 4 distinct domains (25 points maximum per domain):

| Domain | Weight | Inspection Scope | Base Score |
|---|---|---|---|
| **D1: AI Governance & Bot Shield** | 25 pts | `robots.txt` AI crawlers, `llms.txt`, `security.txt` | 25 pts |
| **D2: HTTP Security Headers** | 25 pts | CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy | 25 pts |
| **D3: TLS & Cryptographic Transport** | 25 pts | TLS 1.3/1.2, Cert valid days, SAN matching, Cipher strength | 25 pts |
| **D4: Exposure & Surface Shield** | 25 pts | Sensitive files (`.env`, `.git`, `openapi.json`, `swagger`) | 25 pts |

### 3.2 Finding Severities & Point Deductions
Findings are categorized into 5 standardized severity levels:

| Severity Tag | Score Deduction | Description |
|---|---|---|
| `CRITICAL` | **-20 pts** | Active secret leakage (`.env`, `.git/HEAD`), plaintext HTTP/expired cert, or complete failure. |
| `HIGH` | **-10 pts** | Missing HSTS, TLS cert expiring in < 15 days, or completely open AI scraping with zero robots.txt policy. |
| `MEDIUM` | **-5 pts** | Missing CSP, missing X-Frame-Options, or TLS cert expiring in 15–30 days. |
| `LOW` | **-2 pts** | Missing Referrer-Policy, missing `llms.txt`, or missing `security.txt`. |
| `PASS` | **0 pts** | Best-practice standard fully satisfied. |

> **Score Normalization**:  
> Domain scores are clamped between `[0, 25]`.  
> Composite Score = `Domain 1 + Domain 2 + Domain 3 + Domain 4` clamped to `[0, 100]`.

### 3.3 Executive Letter Grade Scale

```
+-----------------------------------------------------------------------------------+
|  Composite Score  | Grade | Status Classification   | Recommended Action         |
+-------------------+-------+-------------------------+----------------------------+
|  95 - 100 pts     |  A+   | Hardened Defense        | Maintain continuous monitoring |
|  85 - 94 pts      |  A    | Strong Defense Posture  | Review minor low-severity tips |
|  70 - 84 pts      |  B    | Moderate Posture        | Remediation within 30 days |
|  50 - 69 pts      |  C    | Deficient Posture       | Remediation within 7 days  |
|   0 - 49 pts      |  F    | Critical Exposure       | Immediate emergency fix    |
+-----------------------------------------------------------------------------------+
```

---

## 4. User Experience & Cockpit Workflow

### 4.1 Cockpit Integration Architecture
The Web Security & AI Defense Audit interface resides inside the Authenticated Operations Cockpit (`#dashboard-view`).

```
[ Operations Cockpit Navigation ]
  |-- Active Outage Banner (if incidents)
  |-- System Operational Banner
  |-- Micro-Kernel Telemetry Cards (Uptime, RSS, WAL, Goroutines)
  |-- Monitored Services Table
  |-- Incident Hub Section
  +-- [NEW] Web Security & AI Defense Audit Section
        |-- Audit Control Bar (URL input + one-click trigger + quick target picker)
        |-- Audit Scoreboard (Score Gauge, Grade Badge, Domain Progress Cards)
        |-- Findings Table (Filterable by severity & domain)
        |-- Remediation Quick-Copy Code Snippets (Nginx, Caddy, Cloudflare, robots.txt)
        +-- "Export Executive Security Report" Modal / Action
```

### 4.2 User Journey
1. **Target Selection**:
   - User types an arbitrary external domain (e.g., `https://goodai.co.nz` or `https://api.myclient.com`).
   - *OR* User selects an existing target from the **"Audit Monitored Target"** dropdown.
2. **Scan Execution**:
   - User clicks **"Run Security Audit"**.
   - Button shows an animated spinner (`"Auditing 24 Security Vectors..."`).
   - Backend asynchronously executes the 4 inspection modules concurrently via `SafeDialer` with a 5-second total timeout.
3. **Score & Findings Rendering**:
   - Dynamic SVG Score Wheel animates to the calculated composite score (e.g., `92 / 100`).
   - Grade Badge lights up in color (`A`, `#10b981`).
   - 4 Domain breakdown cards show individual domain health (e.g., AI: 23/25, Headers: 20/25, TLS: 25/25, Surface: 24/25).
   - Findings table populates with expandable findings.
4. **Remediation Inspection**:
   - Clicking on any non-pass finding expands an actionable remediation drawer containing:
     - Clear description of the vulnerability.
     - Copyable snippet (e.g., Nginx header configuration or `robots.txt` AI disallow rule).
5. **Executive Report Generation**:
   - User clicks **"Export Executive Security Report"**.
   - System renders and opens/downloads a self-contained HTML document (`GoodAI-Security-Report-[domain]-[date].html`) suitable for printing or saving directly to PDF.

---

## 5. ASCII & Markdown Wireframes

### 5.1 Cockpit UI Wireframe: Security Audit & AI Defense Console

```text
+--------------------------------------------------------------------------------------------------------------------+
|  GOODAI MONITOR  |  Availability & Fault Diagnostics                       17:52:10 NZDT  [*] [ Alex Morgan (U) ]  |
+--------------------------------------------------------------------------------------------------------------------+
|                                                                                                                    |
|  [=== MONITORED FLEET ===]  [=== INCIDENT HUB ===]  [=== SECURITY & AI DEFENSE AUDIT ===]                          |
|                                                                                                                    |
|  +--------------------------------------------------------------------------------------------------------------+  |
|  | [Shield Icon] WEB SECURITY & AI DEFENSE AUDIT CONSOLE                                                        |  |
|  | Comprehensive multi-vector inspection for modern web infrastructure & autonomous AI crawler governance      |  |
|  +--------------------------------------------------------------------------------------------------------------+  |
|                                                                                                                    |
|  TARGET SELECTION BAR:                                                                                             |
|  +-------------------------------------------------------------------+  +--------------------+  +----------------+ |
|  | Target URL: [ https://goodai.co.nz                              ] |  | [ Quick Select v ] |  | [> Run Audit]  | |
|  +-------------------------------------------------------------------+  +--------------------+  +----------------+ |
|                                                                                                                    |
|  LATEST AUDIT RESULTS: https://goodai.co.nz (Audited: 2026-10-14 17:51:48 NZDT)                                   |
|  +--------------------------------------------------------------------------------------------------------------+  |
|  |                                                                                                              |  |
|  |   +-----------------------+   +--------------------------------------------------------------------------+   |  |
|  |   |   COMPOSITE SCORE     |   | DOMAIN POSTURE BREAKDOWN                                                 |   |  |
|  |   |                       |   |                                                                          |   |  |
|  |   |       /=======\       |   |   1. AI Bot & Scraper Governance          [==================..] 21 / 25 pts |   |  |
|  |   |      |   92    |      |   |      Status: GPTBot restricted, ClaudeBot allowed, missing llms.txt          |   |  |
|  |   |       \=======/       |   |                                                                          |   |  |
|  |   |                       |   |   2. HTTP Security Headers Posture        [================....] 20 / 25 pts |   |  |
|  |   |    [ GRADE: A ]       |   |      Status: HSTS 31536000s, Nosniff active, CSP missing unsafe-inline rule  |   |  |
|  |   |  Strong Defense       |   |                                                                          |   |  |
|  |   |                       |   |   3. Transport & TLS Cryptography         [====================] 25 / 25 pts |   |  |
|  |   |  Tested: 24 Vectors   |   |      Status: TLS 1.3 enforced, 72 days remaining, Let's Encrypt CA           |   |  |
|  |   |  SSRF Shield: ACTIVE  |   |                                                                          |   |  |
|  |   |                       |   |   4. Exposure & Attack Surface Shield     [====================] 25 / 25 pts |   |  |
|  |   | [Export Exec Report]  |   |      Status: Zero exposed .env, .git, or swagger endpoints detected          |   |  |
|  |   +-----------------------+   +--------------------------------------------------------------------------+   |  |
|  +--------------------------------------------------------------------------------------------------------------+  |
|                                                                                                                    |
|  DETAILED FINDINGS & VULNERABILITY REGISTER:                                                                       |
|  Filter by Severity: [ ALL (24) ] [ CRITICAL (0) ] [ HIGH (0) ] [ MEDIUM (2) ] [ LOW (2) ] [ PASS (20) ]           |
|                                                                                                                    |
|  +----------+--------------------+---------------------------------------+---------------------+-----------------+ |
|  | SEVERITY | DOMAIN             | VECTOR CHECK                          | OBSERVED VALUE      | ACTION          | |
|  +----------+--------------------+---------------------------------------+---------------------+-----------------+ |
|  | [PASS]   | Transport & TLS    | Protocol Version                      | TLS 1.3 Enforced    | Compliant       | |
|  | [PASS]   | Transport & TLS    | Certificate Validity Remaining        | 72 Days (Dec 2026)  | Compliant       | |
|  | [PASS]   | Surface Exposure   | Environment File Probe (/.env)        | HTTP 404 (Safe)     | Compliant       | |
|  | [PASS]   | Surface Exposure   | Git Metadata Probe (/.git/HEAD)       | HTTP 404 (Safe)     | Compliant       | |
|  | [PASS]   | Security Headers   | Strict-Transport-Security (HSTS)      | max-age=31536000    | Compliant       | |
|  | [PASS]   | Security Headers   | MIME Type Sniffing Shield             | nosniff             | Compliant       | |
|  | [MEDIUM] | Security Headers   | Content-Security-Policy (CSP)         | Directive missing   | [Remediation v] | |
|  | [LOW]    | AI Governance      | AI Context Standard (llms.txt)        | HTTP 404 Not Found  | [Remediation v] | |
|  | [PASS]   | AI Governance      | OpenAI Scraper Policy (GPTBot)        | Disallow: /api/     | Compliant       | |
|  | [PASS]   | AI Governance      | Security Disclosure (security.txt)    | RFC 9116 Present    | Compliant       | |
|  +----------+--------------------+---------------------------------------+---------------------+-----------------+ |
|                                                                                                                    |
|  [EXPANDED REMEDIATION DRAWER for: Content-Security-Policy (CSP)]                                                  |
|  +--------------------------------------------------------------------------------------------------------------+  |
|  | Recommendation: Implement a strict Content-Security-Policy header to prevent XSS and unauthorized framing.  |  |
|  | Nginx / Cloudflare Snippet:                                                                                  |  |
|  |   add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline';  |  |
|  |   object-src 'none'; base-uri 'self'; frame-ancestors 'none';" always;                                       |  |
|  | [ Copy Nginx Config ] [ Copy Caddyfile Config ] [ Close ]                                                    |  |
|  +--------------------------------------------------------------------------------------------------------------+  |
|                                                                                                                    |
+--------------------------------------------------------------------------------------------------------------------+
```

---

### 5.2 Standalone Exported Executive Security Report Wireframe (Print & PDF Optimized)

The exported report is a self-contained, standalone HTML file that renders with a dark high-tech theme on screens, and cleanly adapts to print/PDF with crisp black-and-white high contrast, zero page-break clipping on metric cards, and a formal executive sign-off section.

```text
+====================================================================================================================+
|                                    GOODAI MONITOR &bull; EXECUTIVE SECURITY REPORT                                  |
|                                    CONFIDENTIAL // SECURITY AUDIT ASSESSMENT                                       |
+====================================================================================================================+
|                                                                                                                    |
|  TARGET AUDIT INFORMATION:                                                                                         |
|  - Target Hostname:        https://goodai.co.nz                                                                    |
|  - Target Resolved IP:     158.179.29.27 (Oracle Cloud Infrastructure, Oceania Node)                               |
|  - Audit Timestamp:        2026-10-14 17:51:48 NZDT (04:51:48 UTC)                                                |
|  - Probing Engine:         GoodAI Synthetic Security Engine v1.0.0-mvp                                             |
|  - Audit Verification ID:  sec-rpt-9d4f82a170e5                                                                    |
|                                                                                                                    |
|  +--------------------------------------------------------------------------------------------------------------+  |
|  |                                         EXECUTIVE SECURITY SCORECARD                                         |  |
|  |                                                                                                              |  |
|  |     +-------------------------+      +----------------------------------------------------------------+      |  |
|  |     |   OVERALL AUDIT GRADE   |      | DOMAIN SECURITY SCORES                                         |      |  |
|  |     |                         |      |                                                                |      |  |
|  |     |       GRADE:  A         |      |  * AI Governance & Bot Shield:       21 / 25 pts  [HIGH PASS]  |      |  |
|  |     |     Score: 92 / 100     |      |  * HTTP Security Headers Posture:    20 / 25 pts  [PASS]       |      |  |
|  |     |                         |      |  * TLS & Transport Cryptography:     25 / 25 pts  [OPTIMAL]    |      |  |
|  |     |  [STRONG DEFENSE POSTURE] |    |  * Surface Exposure Shield:          25 / 25 pts  [HARDENED]   |      |  |
|  |     +-------------------------+      +----------------------------------------------------------------+      |  |
|  +--------------------------------------------------------------------------------------------------------------+  |
|                                                                                                                    |
|  1. EXECUTIVE SUMMARY & POSTURE ASSESSMENT                                                                         |
|  ----------------------------------------------------------------------------------------------------------------  |
|  The target infrastructure (https://goodai.co.nz) exhibits a strong security posture with a composite score of     |
|  92/100 (Grade A). Transport layer cryptographic controls are optimal with TLS 1.3 enforcement and an active       |
|  Let's Encrypt certificate valid for 72 days. No sensitive environment variables (.env) or Git artifacts           |
|  were detected during surface probes. Minor hardening is recommended for Content-Security-Policy (CSP) headers    |
|  and the introduction of an llms.txt standard for AI crawler context curation.                                     |
|                                                                                                                    |
|  2. COMPREHENSIVE VECTOR FINDINGS REGISTER                                                                         |
|  ----------------------------------------------------------------------------------------------------------------  |
|  +-----------+----------------------+-----------------------------+----------------------------------------------+ |
|  | Severity  | Category             | Inspection Check            | Finding Observation & Impact                 | |
|  +-----------+----------------------+-----------------------------+----------------------------------------------+ |
|  | [PASS]    | TLS & Transport      | TLS Protocol Version        | TLS 1.3 enforced. Plaintext HTTP redirected. | |
|  | [PASS]    | TLS & Transport      | Certificate Expiry          | Certificate valid for 72 days.               | |
|  | [PASS]    | TLS & Transport      | Certificate Authority       | Let's Encrypt Authority verified.            | |
|  | [PASS]    | Surface Exposure     | Environment File (.env)     | Probe returned 404; zero leaked secrets.     | |
|  | [PASS]    | Surface Exposure     | Git Repository (.git/HEAD)  | Probe returned 404; zero source repository.  | |
|  | [PASS]    | Surface Exposure     | Swagger API (/swagger)      | Probes returned safe non-exposed responses.  | |
|  | [PASS]    | Security Headers     | Strict-Transport-Security   | HSTS active with max-age=31536000.           | |
|  | [PASS]    | Security Headers     | X-Content-Type-Options      | nosniff header enforced.                     | |
|  | [PASS]    | Security Headers     | X-Frame-Options             | DENY directive active against clickjacking.  | |
|  | [MEDIUM]  | Security Headers     | Content-Security-Policy     | Missing CSP header; XSS mitigation missing.  | |
|  | [LOW]     | AI Governance        | LLM Context (llms.txt)      | No llms.txt found; LLM scrapers lack schema. | |
|  | [PASS]    | AI Governance        | Bot Policy (GPTBot)         | Explicit crawler directive detected.         | |
|  | [PASS]    | AI Governance        | Coordinated Disclosure      | /.well-known/security.txt complies RFC 9116. | |
|  +-----------+----------------------+-----------------------------+----------------------------------------------+ |
|                                                                                                                    |
|  3. STRATEGIC REMEDIATION ROADMAP                                                                                  |
|  ----------------------------------------------------------------------------------------------------------------  |
|  [HIGH PRIORITY] Content-Security-Policy Header Hardening                                                          |
|  - Impact: Mitigates Cross-Site Scripting (XSS) and data injection vectors.                                        |
|  - Implementation: Inject header in reverse proxy:                                                                 |
|      add_header Content-Security-Policy "default-src 'self'; script-src 'self'; object-src 'none';" always;        |
|                                                                                                                    |
|  [RECOMMENDED] Curate /llms.txt for Modern AI Search Engines                                                       |
|  - Impact: Guides autonomous LLM search agents (Perplexity, ChatGPT Search) to official documentation.            |
|  - Implementation: Publish markdown summary at https://goodai.co.nz/llms.txt                                       |
|                                                                                                                    |
|  4. AUDIT ATTESTATION & SIGN-OFF                                                                                   |
|  ----------------------------------------------------------------------------------------------------------------  |
|  Generated by: GoodAI Autonomous Security Prober Engine                                                           |
|  Attestation Hash: SHA256: 3c5e88414ab3e6f9a08152bc23999e527d7d23a1c17240c3cbdf6d9418659102                     |
|  Platform Host: GoodAI Monitor Node AU-01 &bull; Auckland & Melbourne Operations Node                              |
|  Report Standard: GoodAI Web Defense Matrix v1.4 // Strictly 100% Autonomous & SSRF-Protected                     |
+====================================================================================================================+
```

---

## 6. REST API & Data Model Specifications

### 6.1 Data Models (`internal/model/security_audit.go`)

```go
package model

import "time"

type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "CRITICAL"
	SeverityHigh     SeverityLevel = "HIGH"
	SeverityMedium   SeverityLevel = "MEDIUM"
	SeverityLow      SeverityLevel = "LOW"
	SeverityPass     SeverityLevel = "PASS"
)

type AuditFinding struct {
	ID             string        `json:"id"`
	Domain         string        `json:"domain"`          // "ai_governance", "headers", "tls", "exposure"
	CheckName      string        `json:"check_name"`
	Severity       SeverityLevel `json:"severity"`
	ScoreDeduction int           `json:"score_deduction"`
	ObservedValue  string        `json:"observed_value"`
	Recommendation string        `json:"recommendation"`
	Remediation    string        `json:"remediation"`     // Code snippet for remediation
}

type DomainScore struct {
	DomainName string `json:"domain_name"`
	Score      int    `json:"score"`      // 0 - 25
	MaxScore   int    `json:"max_score"`  // 25
	Status     string `json:"status"`     // "Hardened", "Strong", "Moderate", "Action Required"
	Summary    string `json:"summary"`
}

type SecurityAuditReport struct {
	ID             int64          `json:"id"`
	UserID         int64          `json:"user_id"`
	TargetURL      string         `json:"target_url"`
	TargetHost     string         `json:"target_host"`
	TargetIP       string         `json:"target_ip"`
	CompositeScore int            `json:"composite_score"` // 0 - 100
	Grade          string         `json:"grade"`           // "A+", "A", "B", "C", "F"
	GradeLabel     string         `json:"grade_label"`     // "Hardened Defense", "Strong Defense", etc.
	CreatedAt      time.Time      `json:"created_at"`
	ExecutionMs    int64          `json:"execution_ms"`
	DomainScores   []DomainScore  `json:"domain_scores"`
	Findings       []AuditFinding `json:"findings"`
	Attestation    string         `json:"attestation_hash"`
}
```

### 6.2 REST Endpoints (`internal/api/security_handlers.go`)

#### 1. Trigger Security Audit
- **Route**: `POST /api/v1/security/audit`
- **Auth**: Required (`RequireAuth`)
- **Request Body**:
  ```json
  {
    "target_url": "https://goodai.co.nz",
    "monitor_id": 1
  }
  ```
  *(Either `target_url` or `monitor_id` is required).*
- **Response** `201 Created`:
  ```json
  {
    "id": 14,
    "target_url": "https://goodai.co.nz",
    "composite_score": 92,
    "grade": "A",
    "grade_label": "Strong Defense Posture",
    "execution_ms": 340,
    "created_at": "2026-10-14T17:51:48Z",
    "domain_scores": [
      { "domain_name": "AI Bot & Scraper Governance", "score": 21, "max_score": 25, "status": "Strong" },
      { "domain_name": "HTTP Security Headers", "score": 20, "max_score": 25, "status": "Moderate" },
      { "domain_name": "Transport & TLS Cryptography", "score": 25, "max_score": 25, "status": "Hardened" },
      { "domain_name": "Exposure & Surface Shield", "score": 25, "max_score": 25, "status": "Hardened" }
    ],
    "findings": [ ... ]
  }
  ```

#### 2. List Recent Audit Reports
- **Route**: `GET /api/v1/security/reports?limit=10`
- **Auth**: Required
- **Response** `200 OK`:
  Array of previous `SecurityAuditReport` metadata (ID, TargetURL, Score, Grade, CreatedAt).

#### 3. Fetch Single Report Detail
- **Route**: `GET /api/v1/security/reports/{id}`
- **Auth**: Required (Enforces user isolation)
- **Response** `200 OK`: Full `SecurityAuditReport` JSON.

#### 4. Export Standalone Executive Report
- **Route**: `GET /api/v1/security/reports/{id}/export`
- **Auth**: Required (or signed token)
- **Response** `200 OK`:
  `Content-Type: text/html; charset=utf-8`  
  `Content-Disposition: attachment; filename="GoodAI-Security-Report-[host]-[date].html"`  
  *(Self-contained standalone HTML containing all inline styles, printable media queries, and audit findings).*

---

## 7. SSRF Defense Guardrails & Runtime Guarantees

1. **Mandatory SafeDialer Routing**:
   - Every single outgoing HTTP request generated by the audit engine (robots.txt, headers, TLS, exposure probes) **MUST** use `security.NewSafeHTTPClient(5 * time.Second)`.
   - Any attempt to target `127.0.0.1`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `169.254.169.254`, or forbidden ports (anything outside 80, 443, 8080, 8443) returns immediate `400 Bad Request` (`ErrRestrictedTargetURL`).
2. **DNS Rebinding (TOCTOU) Elimination**:
   - `SafeDialContext` resolves the domain and connects directly to the validated IP, preventing DNS rebinding during multi-step scans.
3. **Response Body Truncation**:
   - All response body reads are wrapped with `io.LimitReader(resp.Body, 16384)` (16 KiB limit) to ensure zero memory exhaustion or decompression bombs.
4. **Memory Footprint Preservation**:
   - Concurrent audit scans run in bounded goroutines.
   - Total runtime memory overhead remains strictly within GoodAI Monitor's **< 20–25 MiB RSS budget**.
5. **Localization Guarantee**:
   - 100% English across all UI labels, table headers, error responses, remediation snippets, and exported executive reports.
