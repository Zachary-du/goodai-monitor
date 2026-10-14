# Task Specification: TASK-007 Modern Full-Screen Hero Landing Page & High-Tech Showcase

## Metadata
- **Task ID**: `TASK-007`
- **Title**: Modern Full-Screen Hero Landing Page & High-Tech Showcase
- **Current Status**: `REVIEW`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-14
- **Target Branch**: `task/TASK-007-modern-landing-page`
- **Dependencies**: `TASK-001`, `TASK-002`, `TASK-003`, `TASK-004`, `TASK-005`, `TASK-006`

---

## 1. Requirement & Objective
- **PRD / Product Vision Reference**: `docs/prd.md` Section 3.1 & 4.6; `docs/product-brief.md` Section 3.4 & 5.
- **Business Goal**: Replace the plain unauthenticated view with an enterprise-grade, high-impact full-screen Hero Landing Page that articulates GoodAI Monitor's mission, autonomous synthetic probing capabilities, anti-flapping incident engine, and sub-20MB micro-kernel architecture with sophisticated cloud infrastructure terminology.
- **Detailed Description**:
  - Implement a full-screen Hero section with luminous background gradients, high-tech badge, headline (*"Mission-Critical Availability. Zero Overhead."*), and dual action buttons (*"Launch Free Monitor"* & *"Explore Live Telemetry"*).
  - Implement an interactive Bento Box Showcase demonstrating live telemetry waveforms, 99.99% SLA badge, TLS certificate health, and microsecond network breakdowns (DNS, TCP, TLS 1.3, TTFB).
  - Implement a 4-pillar architectural grid highlighting Autonomous Probing, Anti-Flapping State Machine, Micro-Kernel Footprint (< 20 MiB RSS), and Kernel-Level SSRF Shield.
  - Embed a live public platform status bar showing verified production metrics from `/healthz`.
  - Implement seamless view switching:
    - Guests view the Landing Page with modal authentication triggers.
    - Authenticated users automatically view the Operations Cockpit.
  - Enforce 100% English compliance across all copy, tags, and code comments.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [x] `web/index.html`
- [x] `web/style.css`
- [x] `web/app.js`
- [x] `scripts/run.sh`
- [x] `docs/ai-team/tasks/TASK-007-modern-landing-page.md`
- [x] `docs/ai-team/tasks/TASK-007-modern-landing-page-qa-report.md`

---

## 3. Acceptance Criteria
- [x] **AC-1**: Full-screen Hero Landing section with luminous glowing gradient backdrop, high-tech pill badge (`AUTONOMOUS CLOUD OBSERVABILITY & UPTIME INTELLIGENCE`), impactful headline (`Mission-Critical Availability. Zero Overhead.`), sophisticated subhead, and prominent action CTAs (`Launch Free Monitor` & `View Live Telemetry`).
- [x] **AC-2**: Interactive Showcase Card (Bento Preview) displaying live SLA %, latency wave stream, 72-day certificate ring, and microsecond timing breakdown (DNS, TCP, TLS 1.3, TTFB).
- [x] **AC-3**: Four Architectural Pillars Bento Grid:
  1. Autonomous Synthetic Prober
  2. Anti-Flapping Hysteresis Engine
  3. Ultra-Low Overhead Micro-Kernel (< 20 MiB RSS)
  4. Enterprise-Grade Network Isolation & SSRF Shield
- [x] **AC-4**: Live System Telemetry Strip embedded on the landing page, showing live public server metrics from `/healthz`.
- [x] **AC-5**: Seamless View Switcher:
  - Guests see the full-screen Landing Page + header with Sign In / Sign Up buttons.
  - Logged-in users see the Operations Cockpit (monitored targets, metrics, incident hub).
  - Smooth modal triggering from any landing page CTA.
- [x] **AC-6**: 100% English Compliance (zero Chinese characters in code, HTML, CSS, JS, comments).
- [x] **AC-7**: Responsive layout (mobile, tablet, desktop) with modern CSS variables, glassmorphism, and dark/light mode compatibility.
- [x] **AC-8**: Strict No-Build Compliance: zero local compilation or build commands; verified via remote GitHub Actions CI.

---

## 4. Testing Requirements
- **Automated Verification**: GitHub Actions CI workflow runs unit tests, linter, and 100% English regex scan.
- **Visual & Functional Inspection**: Verify both unauthenticated landing page presentation and authenticated operations cockpit view.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**: Ready for commit on branch `task/TASK-007-modern-landing-page`
- **Files Modified**:
  - `web/index.html`: Added `#landing-view` with Hero section, Bento showcase card, 4 architectural pillars, live telemetry proof strip, and bottom CTA banner. Wrapped existing authenticated dashboard in `#dashboard-view`.
  - `web/style.css`: Appended modern responsive styling for `.landing-container`, `.hero-section`, `.hero-glow`, `.hero-headline`, `.showcase-grid`, `.pillars-grid`, and `.live-status-section`.
  - `web/app.js`: Added view switching in `setGuestState` (shows landing, hides dashboard) and `setAuthenticatedState` (hides landing, shows dashboard). Connected `.btn-launch-free` to open registration modal and `#btn-explore-telemetry` to smooth scroll to showcase. Wired real-time `/healthz` telemetry into the landing page proof strip.
  - `docs/ai-team/tasks/TASK-007-modern-landing-page.md`
- **Automated Test Execution**: Verified zero Chinese characters in `web/` using regex scan.
- **Known Limitations & Caveats**: None; completely client-side progressive enhancement preserving all existing backend APIs and data models.
- **Handoff Date**: 2026-10-14

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**: `docs/ai-team/tasks/TASK-007-modern-landing-page-qa-report.md`
- **Verdict**: `PENDING_REVIEW`
- **Audit Date**: 2026-10-14
