# Task Specification: TASK-010 Customer Fleet Telemetry & Elimination of Internal Debugging Widgets

## Metadata
- **Task ID**: `TASK-010`
- **Title**: Customer Fleet Telemetry Dashboard & Total Elimination of Internal Debugging Widgets
- **Current Status**: `DONE`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **QA Report**: `docs/ai-team/tasks/TASK-010-fleet-dashboard-cleanup-qa-report.md` (APPROVED)
- **Created Date**: 2026-10-15
- **Target Branch**: `main`
- **Dependencies**: `TASK-001` through `TASK-009`

---

## 1. Requirement & Objective
- **Problem Statement**:
  - The customer dashboard previously exposed low-level internal server telemetry to logged-in users:
    1. Container restart uptime ("Server Uptime: 12m 54s") instead of customer fleet uptime.
    2. Physical memory RSS ("System Resource Load: 20.0 MiB") instead of customer service resource metrics.
    3. Storage engine details ("Database Reliability: Active WAL") instead of service health.
    4. Go goroutine counts ("Concurrent Workers: 16") instead of active targets.
    5. A hardcoded dummy row (`GoodAI Monitor Health / GET /healthz`) in the monitors table.
    6. Landing page "Live Platform Telemetry Proof" section reflecting raw container stats rather than commercial enterprise guarantees.
- **Business Goal**:
  - Transform GoodAI Monitor into a true commercial-grade SaaS dashboard by completely eradicating all internal debugging widgets, memory numbers, database internals, and dummy rows.
  - Deliver genuine **Customer Fleet Telemetry** reflecting the user's actual monitored endpoints, active outages, rolling SLA, and synthetic response times.

---

## 2. 3-Pass Codebase Audit Log (Pre-Implementation)

### Audit Pass 1: Customer-Facing Content & HTML Semantics (`web/index.html`)
- **Finding 1.1**: Lines 851-901 `.metrics-grid` rendered 4 internal container telemetry cards: `Server Uptime`, `System Resource Load`, `Database Reliability`, `Concurrent Workers`.
- **Finding 1.2**: Lines 931-947 rendered hardcoded initial row `GoodAI Monitor Health / GET /healthz`.
- **Finding 1.3**: Lines 776-805 on landing page rendered internal container uptime and worker counts.

### Audit Pass 2: Frontend Data Flow & State Logic (`web/app.js`)
- **Finding 2.1**: `pollHealth()` and `updateUIWithHealthData()` fetched `/healthz` every 5 seconds and continuously injected container memory (20.0 MiB), uptime, WAL status, and goroutine count into customer-facing cards.
- **Finding 2.2**: `renderGuestMonitors()` injected a dummy row `GoodAI Monitor Health / GET /healthz` into the monitors table.
- **Finding 2.3**: `loadMonitors()` did not calculate fleet-wide aggregate metrics (SLA, active targets, average latency, active outages).

### Audit Pass 3: Backend API & Architectural Boundaries
- **Finding 3.1**: `/healthz` endpoint in `internal/api/healthz.go` is designed for infrastructure orchestration (Docker / Kubernetes probes), but was improperly coupled directly to customer UI cards.

---

## 3. Scope Whitelist
- [x] `web/index.html`
- [x] `web/app.js`
- [x] `web/style.css`
- [x] `docs/ai-team/tasks/TASK-010-fleet-dashboard-cleanup.md`
- [x] `docs/ai-team/tasks/TASK-010-fleet-dashboard-cleanup-qa-report.md`
- [x] `docs/ai-team/tasks/README.md`
- [x] `scripts/run.sh`

---

## 4. Acceptance Criteria (AC)
- [x] **AC-1 (Zero Internal Debug Junk)**: All mentions of container uptime, RSS memory (20.0 MiB), WAL mode, and goroutine worker counts are completely removed from all customer-facing screens.
- [x] **AC-2 (Fleet SLA Card)**: Displays rolling 24h average SLA across user's active targets (`--` when 0 targets).
- [x] **AC-3 (Monitored Services Card)**: Displays active vs total targets quota (`X / 10 Active`).
- [x] **AC-4 (Avg Response Latency Card)**: Displays fleet-wide average synthetic TTFB latency (`--` when no probes).
- [x] **AC-5 (Active Outages Card)**: Displays live count of active confirmed outages.
- [x] **AC-6 (System Banner Fleet Awareness)**: Banner reflects fleet state (`STANDBY` when 0 monitors, `OPERATIONAL` when all UP, `OUTAGE DETECTED` when any DOWN).
- [x] **AC-7 (Zero Mock/Dummy Rows)**: No hardcoded `/healthz` rows in the monitors table.
- [x] **AC-8 (Commercial Landing Page)**: Landing page displays enterprise SLA guarantees and edge capabilities instead of raw container metrics.
- [x] **AC-9 (100% English & No-Build Compliance)**: Strictly zero Chinese in source code and web assets; zero local builds.
