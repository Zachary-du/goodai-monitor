# Task Specification: TASK-008 Web Security & AI Defense Audit with Standalone Executive Report

## Metadata
- **Task ID**: `TASK-008`
- **Title**: Web Security & AI Defense Audit with Standalone Executive Report
- **Current Status**: `REVIEW` <!-- Options: BACKLOG | READY | IN_PROGRESS | REVIEW | BLOCKED | DONE -->
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-14
- **Target Branch**: `task/TASK-008-security-audit-and-ai-defense-report`
- **Dependencies**: `TASK-001`, `TASK-002`, `TASK-003`

---

## 1. Requirement & Objective
- **PRD / Prototype Reference**:
  - `docs/prototypes/security-audit-prototype.md`
  - `docs/prd.md` Section 3 & 4
  - `docs/architecture.md`
- **Business Goal**: Modern web infrastructure faces threats from both traditional security vulnerabilities (exposed `.env`/`.git`, missing HSTS/CSP, expired certificates) and modern automated AI crawlers (uncontrolled LLM scraping by GPTBot/ClaudeBot/PerplexityBot without `robots.txt` policy or `llms.txt` context). This task delivers an automated, SSRF-safe diagnostic engine that audits targets across 4 key security domains, renders an interactive score gauge and findings register in the Cockpit, and exports a print/PDF-optimized executive HTML report.
- **Detailed Description**:
  1. Implement a Go backend inspection engine under `internal/security/` that evaluates:
     - AI Bot & Scraper Governance (`robots.txt` AI user-agents, `/llms.txt`, `/.well-known/security.txt`).
     - HTTP Security Headers (`Content-Security-Policy`, `Strict-Transport-Security`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`).
     - Transport & TLS Cryptography (TLS 1.3 protocol validation, Certificate valid days remaining, Issuer CA, SAN matching).
     - Sensitive Exposure Probing (SSRF-safe probing for `/.env`, `/.git/HEAD`, `/swagger`, `/openapi.json` with strict content validation to eliminate false positives from custom 404 pages).
  2. Implement a 0–100 composite scoring algorithm with letter grades (`A+`, `A`, `B`, `C`, `F`) partitioned across 4 domains (25 points each) with severity deductions (`CRITICAL -20`, `HIGH -10`, `MEDIUM -5`, `LOW -2`, `PASS 0`).
  3. Implement SQLite persistence in `internal/storage/security_store.go` storing audit reports with user isolation.
  4. Implement REST API endpoints in `internal/api/security_handlers.go`:
     - `POST /api/v1/security/audit` (initiates audit scan)
     - `GET /api/v1/security/reports` (lists user audit history)
     - `GET /api/v1/security/reports/{id}` (retrieves full report JSON)
     - `GET /api/v1/security/reports/{id}/export` (downloads standalone self-contained HTML report)
  5. Extend the authenticated Operations Cockpit in `web/`:
     - URL input + quick-select dropdown for existing monitored endpoints + "Run Security Audit" button.
     - Dynamic SVG score wheel, Grade badge, and 4-domain posture breakdown progress bars.
     - Severity-tagged findings table with expandable remediation code drawers (Nginx, Caddy, Cloudflare, robots.txt).
     - "Export Executive Security Report" button triggering standalone HTML export.
  6. Enforce strict SSRF defense: All scans **MUST** use `security.NewSafeHTTPClient` or `security.SafeDialContext` to eliminate internal network access and TOCTOU DNS rebinding.
  7. Enforce 100% English UI, error messages, and code comments.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [x] `internal/security/audit.go`
- [x] `internal/security/audit_test.go`
- [x] `internal/security/ai_governance.go`
- [x] `internal/security/ai_governance_test.go`
- [x] `internal/security/headers.go`
- [x] `internal/security/headers_test.go`
- [x] `internal/security/tls.go`
- [x] `internal/security/tls_test.go`
- [x] `internal/security/probes.go`
- [x] `internal/security/probes_test.go`
- [x] `internal/security/score.go`
- [x] `internal/security/score_test.go`
- [x] `internal/security/report_html.go`
- [x] `internal/security/report_html_test.go`
- [x] `internal/model/security_audit.go`
- [x] `internal/storage/security_store.go`
- [x] `internal/storage/security_store_test.go`
- [x] `internal/api/security_handlers.go`
- [x] `internal/api/security_handlers_test.go`
- [x] `internal/api/router.go`
- [x] `internal/api/router_test.go`
- [x] `web/index.html`
- [x] `web/style.css`
- [x] `web/app.js`
- [x] `scripts/run.sh`
- [x] `docs/ai-team/tasks/TASK-008-security-audit-and-ai-defense-report.md`
- [x] `docs/ai-team/tasks/TASK-008-security-audit-and-ai-defense-report-qa-report.md`

---

## 3. Acceptance Criteria

- [x] **AC-1: AI Crawler & Scraper Governance Prober**:
  - Correctly fetches and parses `robots.txt` targeting: `GPTBot`, `ClaudeBot` / `anthropic-ai`, `PerplexityBot`, `Google-Extended`, `Bytespider`, and `CCBot`.
  - Identifies whether directives allow, disallow, or lack explicit policy for each AI crawler.
  - Probes for `/llms.txt` and `/.well-known/llms.txt` (AI context curation).
  - Probes for `/.well-known/security.txt` and `/security.txt` per RFC 9116 (verifying `Contact:` and `Expires:`).

- [x] **AC-2: HTTP Security Headers Posture Engine**:
  - Assesses `Content-Security-Policy` (rewards strict policy, flags missing or `unsafe-inline` wildcards).
  - Assesses `Strict-Transport-Security` (verifies `max-age >= 15768000`, rewards `includeSubDomains` and `preload`).
  - Assesses `X-Frame-Options` (`DENY` or `SAMEORIGIN`).
  - Assesses `X-Content-Type-Options` (`nosniff`).
  - Assesses `Referrer-Policy` (e.g., `strict-origin-when-cross-origin`, `no-referrer`).
  - Assesses `Permissions-Policy` (camera, microphone, geolocation restrictions).

- [x] **AC-3: Transport & TLS Cryptographic Posture**:
  - Connects via TLS and validates negotiated protocol version (rewards TLS 1.3, accepts TLS 1.2, fails TLS 1.0/1.1 or plaintext HTTP).
  - Calculates remaining certificate validity days (> 30 days = PASS, 15–30 days = MEDIUM, < 15 days = HIGH, Expired = CRITICAL).
  - Validates certificate issuer CA and SAN (Subject Alternative Names) matching target hostname.

- [x] **AC-4: SSRF-Safe Sensitive Exposure Probes**:
  - Probes for high-risk exposures: `/.env`, `/.git/HEAD`, `/swagger`, `/swagger/index.html`, `/openapi.json`.
  - Implements strict anti-false-positive pattern validation (e.g., checks for `^[A-Z0-9_]{3,}=` without HTML tags for `.env`, `ref: refs/` for `.git/HEAD`, valid JSON schema for `/openapi.json`) to prevent false alarms on custom 404 HTML pages.
  - Uses `io.LimitReader` capped at 16 KiB per probe to prevent memory DoS.
  - Strictly routes all probes through `security.SafeDialer` / `security.SafeDialContext`.

- [x] **AC-5: Multi-Domain Scoring & Grading Algorithm**:
  - Computes composite 0–100 integer score across 4 domains (AI Governance: 25 pts, Security Headers: 25 pts, TLS Transport: 25 pts, Surface Exposure: 25 pts).
  - Applies standardized deductions: CRITICAL (-20), HIGH (-10), MEDIUM (-5), LOW (-2), PASS (0).
  - Assigns letter grades: `A+` (95–100), `A` (85–94), `B` (70–84), `C` (50–69), `F` (0–49).

- [x] **AC-6: Interactive Operations Cockpit UI**:
  - Adds a dedicated "Security & AI Defense Audit" section in `web/index.html` within `#dashboard-view`.
  - Supports manual URL input and a one-click dropdown to select from user's monitored targets.
  - Displays dynamic SVG Score Gauge, Grade Badge, and 4 domain breakdown progress bars.
  - Renders a filterable findings table (Filter by: All, Critical, High, Medium, Low, Pass) with expandable remediation drawers containing actionable Nginx/Caddy/Cloudflare snippets.

- [x] **AC-7: Standalone Executive Security Report HTML Exporter**:
  - Implements `GET /api/v1/security/reports/{id}/export` returning a self-contained, standalone HTML file.
  - Report features dark high-tech styling for screen viewing and crisp, professional black-and-white layout for `@media print` (PDF export ready with zero page-break card clipping).
  - Includes target metadata, executive scorecard, domain breakdowns, full findings table, remediation roadmap, and cryptographic attestation hash.

- [x] **AC-8: Security, Memory & Localization Safeguards**:
  - Zero SSRF vulnerability: Attempts to audit private IP addresses (`10.0.0.0/8`, `127.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `169.254.169.254`) or restricted ports are immediately rejected with `400 Bad Request`.
  - 100% English Compliance: All UI text, table labels, error messages, and code comments must be in English.
  - Memory Overhead: Bounded memory footprint strictly within the project's `< 25 MiB RSS` budget.
  - Multi-tenant isolation: All reports scoped to `user_id` in SQLite database.

---

## 4. Testing Requirements
- **Unit Tests**:
  - `internal/security/ai_governance_test.go`: Test `robots.txt` parsing against various user-agent blocks and RFC 9116 `security.txt`.
  - `internal/security/headers_test.go`: Test header extraction and deduction logic across varied simulated HTTP responses.
  - `internal/security/tls_test.go`: Test TLS handshake inspection, certificate days calculation, and SAN matching.
  - `internal/security/probes_test.go`: Test exposure probe pattern matching and false-positive filtering.
  - `internal/security/score_test.go`: Test scoring math, domain clamping, and grade thresholds.
  - `internal/security/report_html_test.go`: Test standalone HTML report generation and escaping.
  - `internal/storage/security_store_test.go`: Test report CRUD operations and user isolation.
  - `internal/api/security_handlers_test.go`: Test SSRF rejection, audit execution, and export handlers.
- **Static Analysis**: `go vet ./...` clean with zero warnings.
- **Strict No-Build Rule**: Automated test execution must run within lightweight scope or GitHub Actions CI.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**: `696fe4690377cf5aef6849dcce21421559376484`
- **Files Modified**:
  - `internal/model/security_audit.go`: Domain models, findings, domain breakdown, and report structures.
  - `internal/security/ai_governance.go` & `ai_governance_test.go`: Robots.txt AI bot parser (GPTBot, ClaudeBot, PerplexityBot, Google-Extended, Bytespider, CCBot), llms.txt, security.txt (RFC 9116).
  - `internal/security/headers.go` & `headers_test.go`: CSP, HSTS, X-Frame-Options, nosniff, Referrer-Policy, Permissions-Policy.
  - `internal/security/tls.go` & `tls_test.go`: TLS 1.3 protocol verification, remaining days calculation, SAN matching, issuer CA.
  - `internal/security/probes.go` & `probes_test.go`: Anti-false-positive probes for /.env, /.git/HEAD, /openapi.json, /swagger with 16KB LimitReader and SafeDialer.
  - `internal/security/score.go` & `score_test.go`: 0-100 score engine partitioned across 4 domains (25 pts each) with A+, A, B, C, F grading.
  - `internal/security/report_html.go` & `report_html_test.go`: Standalone executive HTML report generator with dark theme and @media print layout.
  - `internal/security/audit.go` & `audit_test.go`: Concurrent audit orchestrator with SSRF pre-flight validation.
  - `internal/storage/security_store.go` & `security_store_test.go`: SQLite security_reports table with tenant isolation.
  - `internal/api/security_handlers.go` & `security_handlers_test.go`: REST endpoints for audit trigger, report listing, details, and HTML export.
  - `internal/api/router.go` & `router_test.go`: Route registrations protected by RequireAuth.
  - `web/index.html`: Web Security & AI Defense Audit section in dashboard-view with dynamic SVG score wheel, domain cards, findings table, remediation drawers.
  - `web/style.css`: Responsive CSS styling for audit scoreboard, score wheel SVG, grade pills, and remediation snippets.
  - `web/app.js`: Interactive audit trigger, dynamic score wheel animation, findings filtering, and report export trigger.
  - `scripts/run.sh`: Push feature branch to origin and check GitHub Actions CI runs.
- **Automated Test Execution**:
  - 100% English Compliance Python audit: Zero Chinese characters in all modified files.
  - HTML syntax validation: Verified clean.
  - Comprehensive unit test suites authored and passing in GitHub Actions CI workflow:
    - `https://github.com/Zachary-du/goodai-monitor/actions/runs/38516559217`
    - Steps Verified:
      - `Verify 100% English Policy`: Success (0 non-English characters)
      - `Static Analysis (go vet ./...)`: Success (0 warnings)
      - `Run Automated Unit Tests (CGO_ENABLED=0 go test -v -cover ./...)`: Success
      - `Compile Linux AMD64 Binary`: Success
      - `Build & Push Container Image`: Success (published to GHCR)
- **Known Limitations & Caveats**:
  - Plaintext HTTP targets will fail TLS handshake checks and yield CRITICAL for TLS domain as intended by security baseline.
  - Local environment enforces strict no-build rule; test execution is handled by GitHub Actions CI runner.
- **Potential Regression Risks**:
  - Minimal: Security audit endpoints are additive and scoped under `/api/v1/security/*`.
- **Handoff Date**: 2026-10-14

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**: `docs/ai-team/tasks/TASK-008-security-audit-and-ai-defense-report-qa-report.md`
- **Verdict**: `APPROVED`
- **Audit Date**: 2026-10-14
- **Auditor**: QA / Security Agent
- **Target Commit**: `94ef537ec0ad4b0850132e924803aeb5ebb7fada`
- **CI Workflow Run**: [Run 38516817356](https://github.com/Zachary-du/goodai-monitor/actions/runs/38516817356) (`success`)
- **Human Approval Request**: Ready for Human Lead merge approval. Recommendation is `PROCEED_WITH_MERGE`. All 8 Acceptance Criteria verified PASS. SSRF defense, tenant isolation, zero secret leakage, and 100% English compliance confirmed.

