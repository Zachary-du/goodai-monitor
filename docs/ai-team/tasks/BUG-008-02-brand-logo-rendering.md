# Task Specification: BUG-008-02 Brand Logo Rendering & CI/CD Deployment Synchronization

## Metadata
- **Task ID**: `BUG-008-02`
- **Title**: Brand Logo Rendering & CI/CD Deployment Synchronization
- **Current Status**: `DONE` <!-- Options: BACKLOG | READY | IN_PROGRESS | REVIEW | BLOCKED | DONE -->
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-15
- **Target Branch**: `main`
- **Dependencies**: `TASK-008`

---

## 1. Requirement & Objective
- **PRD / Defect Reference**: User-reported defect with screenshot (`media_1792029015043_418112cf.png`), showing broken image placeholder in the brand header.
- **Root Cause Analysis (Defect Report & Incident Context)**:
  1. **Premature Pull Race Condition**: In the previous deployment run (`task-1758`), a fixed `sleep 55` was used while GitHub Actions CI required ~90–120 seconds to compile Go binaries and build/push multi-stage Docker images to GHCR. Consequently, the Melbourne VPS executed `docker compose pull` before the new image was published, leaving the running container on the legacy image (`14dfc9b`).
  2. **Negative Edge Caching Collision**: The legacy container continued serving `index.html` referencing `logo.svg?v=20261015-01`. During previous container recreation, this specific query URL was requested and returned an HTTP 404, which Cloudflare cached with `age: 13577` and `cf-cache-status: HIT`.
  3. **Visual Regression**: Browsers loading `https://goodai.co.nz` received HTML pointing to `logo.svg?v=20261015-01`, received Cloudflare's cached 404, and rendered a broken image icon with alt text.
  4. **Multi-Agent Protocol Deviation**: The previous deployment was executed without waiting for explicit QA Agent verification of the live HTTP status codes of the rendered image.
- **Remediation Plan**:
  1. Wait for GitHub Actions Run `38647808839` to complete successfully.
  2. Pull the verified image on the Melbourne VPS and recreate container `goodai-monitor` (Container ID `8aea1b9924e2`).
  3. Verify that `index.html` on both local container (`http://127.0.0.1:8090/`) and public domain (`https://goodai.co.nz/`) serves `<img src="logo.svg?v=20261015-02">`.
  4. Verify that `https://goodai.co.nz/logo.svg?v=20261015-02` returns `HTTP 200` with `Content-Type: image/svg+xml`.
  5. QA Agent audits live production and generates formal report.

---

## 2. Scope Whitelist (Allowed Files & Modules)
- [x] `web/index.html`
- [x] `web/logo.svg`
- [x] `web/embed.go`
- [x] `web/style.css`
- [x] `scripts/run.sh`
- [x] `docs/ai-team/tasks/BUG-008-02-brand-logo-rendering.md`
- [x] `docs/ai-team/tasks/BUG-008-02-brand-logo-rendering-qa-report.md`

---

## 3. Acceptance Criteria
- [x] **AC-1 (Cache-Busting Image Tag)**: `web/index.html` must reference `logo.svg?v=20261015-02` in both `<link rel="icon">` and `<img class="brand-logo-img">`.
- [x] **AC-2 (HTTP 200 SVG Content-Type)**: `GET https://goodai.co.nz/logo.svg?v=20261015-02` must return `HTTP/2 200` with `Content-Type: image/svg+xml` and `Content-Length: 1421`.
- [x] **AC-3 (Live VPS Container Health)**: Container `goodai-monitor` must be running the latest image with `(healthy)` status.
- [x] **AC-4 (Zero Chinese Characters)**: 100% English compliance across all assets, HTML, CSS, and logs.
- [x] **AC-5 (Strict No-Build Rule)**: All builds compiled remotely via GitHub Actions.
- [x] **AC-6 (Zero Popup Rule)**: All deployment steps packaged in `scripts/run.sh`.
- [x] **AC-7 (Independent QA Audit)**: QA Agent conducts independent live verification and issues audit report.

---

## 4. Developer Implementation Log
- Updated `web/index.html` cache-busting version parameter to `20261015-02`.
- Pushed commit `1282058` to `main`.
- Monitored GitHub Actions Run `38647808839` until completion (`conclusion: success`).
- Pulled latest image on Melbourne VPS (`8aea1b9924e2`).
- Verified local container and public edge responses.
