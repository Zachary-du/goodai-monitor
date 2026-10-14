# Task Specification: BUG-007-01 Cache Synchronization & Modern Landing Visual Polish

## Metadata
- **Task ID**: `BUG-007-01`
- **Title**: Cache Synchronization & Modern Landing Visual Polish
- **Current Status**: `REVIEW` <!-- Options: BACKLOG | READY | IN_PROGRESS | REVIEW | BLOCKED | DONE -->
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-14
- **Target Branch**: `task/BUG-007-01-cache-and-visual-polish`
- **Dependencies**: `TASK-007`

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: `docs/prd.md` Section 3.1 & 4.6; `docs/architecture.md` Section 3; `docs/ai-team/tasks/TASK-007-modern-landing-page.md`.
- **Business Goal**: Eliminate static asset caching desynchronization between edge reverse proxies (Cloudflare) / client browsers and the application HTML, ensure production HTTP `Cache-Control` headers in the Go router, and guarantee full visual polish, complete button resets, and responsive fidelity for the Hero Landing Page and Bento Showcase.
- **Root Cause Analysis (Defect Report & Incident Context)**:
  - Following the deployment of `TASK-007`, the live production site `https://goodai.co.nz` rendered as an unstyled, left-aligned raw HTML page with native grey browser buttons.
  - **Edge Cache Desynchronization**: Cloudflare edge proxy cached `style.css` (HTTP `CF-Cache-Status: HIT`, returning the legacy 537-line stylesheet) while `web/index.html` was updated with the new Hero and Bento Showcase markup (referencing classes declared on lines 538-1033 of `style.css`).
  - **Missing Version Query Strings**: `web/index.html` referenced `style.css` and `app.js` without cache-busting query version tags (e.g., `style.css?v=...`, `app.js?v=...`).
  - **Omitted HTTP Cache-Control Policy**: `internal/api/router.go` served embedded static files via generic `http.FileServer` without explicit `Cache-Control` headers, allowing edge proxies to serve stale content indefinitely while serving HTML with default caching behavior.
  - **QA Gate Failure**: The task branch was merged without live asset verification against the production edge CDN.
- **Detailed Description**:
  1. Add explicit asset version query strings (`?v=20261014-01` or build timestamp) to all static stylesheet and script links in `web/index.html`.
  2. Implement dedicated static file HTTP caching middleware / header controls in `internal/api/router.go`:
     - Document routes (`/`, `index.html`): `Cache-Control: no-cache, no-store, must-revalidate` (forcing edge proxies and browsers to always revalidate the HTML document before serving).
     - Static asset routes (`/style.css`, `/app.js`, images, fonts): explicit `Cache-Control` header policy (e.g., `Cache-Control: public, max-age=86400, must-revalidate` when query versioning is used, or `no-cache` for unversioned fallback).
  3. Verify and enhance button and component CSS resets in `web/style.css` and `web/index.html`: ensure every button (`.btn-hero-primary`, `.btn-hero-secondary`, `.btn-auth-signin`, `.btn-auth-signup`, `.btn-action`, modal action buttons) possesses explicit styling, background gradients/colors, borders, fonts, and hover/active states, eliminating native grey browser buttons.
  4. Ensure visual polish and alignment for the Bento Showcase preview cards, live telemetry sparkline SVG, 4 architectural pillars, and responsive layouts across mobile (<640px), tablet (640px-1024px), and desktop (>1024px).
  5. Provide unit test coverage in `internal/api/router_test.go` verifying the `Cache-Control` and security headers for root documents and static assets.
  6. Update verification script `scripts/run.sh` to validate live edge response headers and asset line counts.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [x] `web/index.html`
- [x] `web/style.css`
- [ ] `web/app.js`
- [x] `internal/api/router.go`
- [x] `internal/api/router_test.go`
- [x] `scripts/run.sh`
- [x] `docs/ai-team/tasks/BUG-007-01-cache-and-visual-polish.md`

---

## 3. Acceptance Criteria
- [x] **AC-1 (Asset Cache-Busting Version Query Tags)**: `web/index.html` must reference `style.css` and `app.js` with version query parameters (e.g. `href="style.css?v=20261014-01"`, `src="app.js?v=20261014-01"`), ensuring Cloudflare and browser caches are invalidated immediately on deployment.
- [x] **AC-2 (HTTP Cache-Control Headers in Go Router)**: `internal/api/router.go` must set explicit `Cache-Control` headers:
  - Document (`/`, `/index.html`): `Cache-Control: no-cache, no-store, must-revalidate` so clients and proxies always fetch the freshest HTML containing updated asset version query tags.
  - Static assets (`style.css`, `app.js`, etc.): Set explicit `Cache-Control` header (e.g., `Cache-Control: public, max-age=86400, must-revalidate`).
  - Unit tests in `internal/api/router_test.go` must assert the presence and exact value of these headers.
- [x] **AC-3 (Comprehensive Button & CTA Styling)**: All buttons on the landing page and navigation header (`.btn-hero-primary`, `.btn-hero-secondary`, `.btn-auth-signin`, `.btn-auth-signup`, `.btn-action`, modal action buttons) must define full CSS resets (`border`, `background`, `color`, `font-family`, `padding`, `border-radius`, `transition`) with distinct hover, focus, and active states. Under no render conditions shall standard browser grey buttons appear.
- [x] **AC-4 (Bento Showcase & Architectural Pillars Visual Polish)**: Bento showcase grid cards, SVG latency sparkline, SLA progress bar, TLS certificate status card, and the 4 architectural pillars must render centered, visually balanced, with dark-theme glassmorphism surfaces (`rgba(15, 23, 42, 0.75)`, subtle border glowing, box shadows) matching modern cloud observability aesthetics.
- [x] **AC-5 (Cross-Device Responsive Viewport Fidelity)**: The full landing page and bento grid must adapt cleanly across viewports:
  - Mobile (< 640px): single-column layout, stacked CTA buttons, no horizontal scrolling.
  - Tablet (640px-1024px): 2-column bento and pillar grids, appropriate header spacing.
  - Desktop (> 1024px): full multi-column bento grid, centered hero copy, and max-width container constraints.
- [x] **AC-6 (100% English Compliance)**: All UI copy, button labels, telemetry labels, code comments, and HTTP header values must be 100% English with zero Chinese or non-English characters.
- [x] **AC-7 (Strict No-Build Rule Compliance)**: Zero local compilation or binary builds. Verification must be performed using lightweight Go unit tests (`go test -v ./internal/api/...`) and GitHub Actions remote CI.

---

## 4. Testing Requirements
- **Unit Tests (`internal/api/router_test.go`)**:
  - Test `GET /` and verify `Cache-Control: no-cache, no-store, must-revalidate` (or `no-cache, must-revalidate`) and security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`).
  - Test `GET /style.css` and `GET /app.js` and verify expected `Cache-Control` headers and `200 OK` status with proper MIME types (`text/css`, `text/javascript` or `application/javascript`).
- **Static Verification**:
  - Regex verification for 100% English compliance across modified files.
  - HTML/CSS inspection verifying cache-busting tags (`?v=...`) and button CSS reset declarations.
- **Strict No-Build Rule**:
  - Do NOT run `go build` locally. Run only unit tests (`go test -v ./internal/api/...`).
  - Push changes to target branch and verify via GitHub Actions CI workflow.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**: `beecdba`
- **Files Modified**:
  - `web/index.html` (Added cache-busting query strings `style.css?v=20261014-01` and `app.js?v=20261014-01`; inserted button reset fallbacks)
  - `web/style.css` (Universal button base reset, enhanced hero CTA active/focus states, glassmorphic surfaces for Bento cards & pillars, comprehensive button reset suite, and cross-device responsive breakpoints for <640px, 640-1024px, and >1024px)
  - `internal/api/router.go` (Implemented staticAssetHandler injecting `Cache-Control: no-cache, no-store, must-revalidate` for document routes `/` and `/index.html`, and `Cache-Control: public, max-age=86400, must-revalidate` for static assets)
  - `internal/api/router_test.go` (Added comprehensive unit tests covering root document and static asset Cache-Control headers and security headers)
  - `scripts/run.sh` (Configured automated git push of branch `task/BUG-007-01-cache-and-visual-polish` and curl inspection of public edge CDN)
  - `docs/ai-team/tasks/BUG-007-01-cache-and-visual-polish.md` (Updated task specification, acceptance criteria check-off, and developer log)
- **Automated Test Execution**:
  - Remote CI/CD Pipeline: GitHub Actions workflow triggered on push to `task/BUG-007-01-cache-and-visual-polish` (`test-and-lint`, `build-binary`, `docker-build`).
  - Unit Tests: `internal/api/router_test.go` covers `GET /`, `GET /index.html`, `GET /style.css`, `GET /style.css?v=20261014-01`, `GET /app.js`, `GET /app.js?v=20261014-01`, and security headers.
  - 100% English Check: Verified zero Chinese characters in source files (`git grep -P "[\x{4e00}-\x{9fa5}]" -- ':!docs' ':!Gemini.md' ':!.git'`).
- **Known Limitations & Caveats**:
  - Cloudflare Edge CDN cache will immediately serve new assets on requests containing version query string `?v=20261014-01`. Unversioned legacy asset requests may serve cached 537-line version until edge TTL expires or Cloudflare cache is purged.
- **Potential Regression Risks**:
  - None. Router changes strictly isolate document routes vs static assets, and preserve standard `http.FileServer` resolution and security headers.
- **Handoff Date**: 2026-10-14

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**: `docs/ai-team/tasks/BUG-007-01-cache-and-visual-polish-qa-report.md`
- **Verdict**: `PENDING` <!-- APPROVED | REJECTED | ACTION_REQUIRED -->
- **Audit Date**: YYYY-MM-DD
- **Human Approval Request**: [Ready for Human Lead merge approval / Needs developer fixes]
