# Task Specification: TASK-012 Commercial Homepage Polish, Dual-Engine Showcase & Instant Playground

## Metadata
- **Task ID**: `TASK-012`
- **Title**: Commercial Homepage Polish: Dual-Engine Interactive Showcase, Instant Security Playground & FAQ
- **Current Status**: `COMPLETED`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-16
- **Completed Date**: 2026-10-16
- **Target Branch**: `main`
- **Dependencies**: `TASK-001` through `TASK-011`

---

## 1. Requirement & Objective
- **Problem Statement**:
  - The homepage (landing view) previously had:
    1. A single-dimensional Hero showcase only highlighting synthetic latency, hiding the Web Security & AI Defense engine.
    2. Zero instant try-before-you-buy experience (all buttons immediately opened a login modal, increasing friction).
    3. Absence of an industry-standard FAQ section addressing developer questions regarding agentless probing, anti-flapping hysteresis, and SSRF security.
- **Business Goal**:
  - Elevate GoodAI Monitor's public landing page to top-tier commercial SaaS standards (like Better Stack, Datadog, Vercel):
    1. **Dual-Engine Hero Showcase Tabs**: Interactive tab switcher in the Hero fold (`⚡ Synthetic Telemetry Prober` vs `🛡️ Web Security & AI Defense`).
    2. **Instant Public Inspection Playground**: Zero-login input allowing visitors to immediately inspect any public domain's AI crawler policy, security headers, and SSL health with instant visual results.
    3. **Backend Public Audit Endpoint**: `POST /api/v1/security/public-audit` with kernel-level SSRF validation, executing real security inspection without requiring authentication.
    4. **Developer FAQ Accordion**: 5 expandable questions explaining agentless probing, hysteresis anti-flapping ($N=2$, $M=2$), AI bot governance, SSRF isolation, and public preview capacity.
    5. **Strict Security & Privacy Adherence**: Zero VPS or geographic leaks, zero fake corporate claims, 100% English compliance, and zero local builds.

---

## 2. Scope Whitelist
- [x] `internal/api/security_handlers.go`
- [x] `internal/api/security_handlers_test.go`
- [x] `internal/api/router.go`
- [x] `internal/api/router_test.go`
- [x] `web/index.html`
- [x] `web/style.css`
- [x] `web/app.js`
- [x] `docs/ai-team/tasks/TASK-012-homepage-interactive-showcase-and-playground.md`
- [x] `docs/ai-team/tasks/README.md`
- [x] `scripts/run.sh`

---

## 3. Acceptance Criteria (AC)
- [x] **AC-1 (Backend Public Audit Endpoint)**: `POST /api/v1/security/public-audit` accepts `{ "target_url": "..." }`, validates SSRF, executes multi-vector audit with `userID=0`, and returns JSON report without requiring authentication.
- [x] **AC-2 (SSRF Pre-Flight Protection)**: Public audit strictly blocks loopbacks, RFC 1918 subnets, cloud metadata (169.254.169.254), and invalid schemes.
- [x] **AC-3 (Dual-Engine Hero Tabs)**: Hero showcase includes interactive tabs switching between Synthetic Telemetry and Web Security & AI Defense.
- [x] **AC-4 (Instant Public Playground)**: Visitors can input any domain on the homepage and view live score, grade, AI bot policy, security headers, and TLS posture.
- [x] **AC-5 (Conversion Bridge)**: Inspection result includes high-converting CTA directing visitors to enable 24/7 continuous monitoring for the scanned domain.
- [x] **AC-6 (Developer FAQ Accordion)**: Interactive 5-item FAQ section addressing agentless architecture, hysteresis, AI bot governance, SSRF shield, and free tier quotas.
- [x] **AC-7 (Zero Leaks & 100% English)**: Zero mentions of VPS, Melbourne, or Oracle; zero fake "Ltd." or "SOC 2" claims; 100% English source code and assets.
- [x] **AC-8 (Strict No-Build Rule)**: Remote CI/CD compilation and verification in GitHub Actions; automated VPS deployment via `scripts/run.sh`.
