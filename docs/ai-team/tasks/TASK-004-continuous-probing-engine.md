# Task Specification: TASK-004 Continuous Background Probing Engine & Telemetry

## Metadata
- **Task ID**: `TASK-004`
- **Title**: Continuous Background Probing Engine, Telemetry & Rolling Retention
- **Current Status**: `BACKLOG`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-13
- **Target Branch**: `task/TASK-004-continuous-probing-engine`
- **Dependencies**: `TASK-001`, `TASK-002`, `TASK-003`

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: `docs/prd.md` FR-PROBE-01 to FR-PROBE-04; FR-UI-03; NFR-PERF-01 to NFR-PERF-03.
- **Business Goal**: Implement the core background engine that continuously checks active monitors, records latency telemetry, detects status transitions, and automatically prunes stale check logs.
- **Detailed Description**:
  - Implement dynamic probe scheduler (`internal/prober/scheduler.go`) maintaining active ticker loops for non-paused monitors.
  - Implement synthetic probe client (`internal/prober/client.go`):
    - Connects using `SafeDialContext` with dial-time IP validation to prevent DNS Rebinding.
    - Measures DNS lookup, TCP connect, TLS handshake, TTFB, and total response latency.
    - Inspects HTTP status code against expected codes list.
    - Performs optional keyword scanning (reading max 64 KB).
    - Checks TLS certificate expiry days.
    - Blocks automatic HTTP redirects (`http.ErrUseLastResponse`).
  - Store check results in SQLite `checks` table.
  - Implement background retention cleaner pruning check logs older than 14 days.
  - Expose telemetry endpoint (`GET /api/v1/monitors/:id/metrics`) returning 24-hour latency samples and SLA uptime %.
  - Render response time trend chart and recent check log list in embedded UI.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [ ] `internal/model/check.go`
- [ ] `internal/storage/check_store.go`
- [ ] `internal/storage/check_store_test.go`
- [ ] `internal/prober/client.go`
- [ ] `internal/prober/client_test.go`
- [ ] `internal/prober/scheduler.go`
- [ ] `internal/prober/scheduler_test.go`
- [ ] `internal/api/metrics_handlers.go`
- [ ] `internal/api/metrics_handlers_test.go`
- [ ] `web/index.html`
- [ ] `web/app.js`

---

## 3. Acceptance Criteria
- [ ] **AC-1**: Active monitors are probed periodically at their configured intervals (e.g. every 60s) completely independent of user browser presence.
- [ ] **AC-2**: Prober measures accurate latency breakdown (DNS, TCP, TLS, TTFB, Total) and stores check records.
- [ ] **AC-3**: Failed checks accurately record specific failure reason (e.g. `HTTP 502 Bad Gateway`, `Connection timeout after 5000ms`, `Keyword 'Welcome' missing`).
- [ ] **AC-4**: Paused monitors are excluded from probing; resuming dynamically schedules the probe without requiring server restart.
- [ ] **AC-5**: `SafeDialContext` prevents DNS rebinding by resolving IP at dial time and connecting directly to verified IP.
- [ ] **AC-6**: Automated retention job purges check records older than 14 days every 24 hours.
- [ ] **AC-7**: Telemetry API returns 24-hour uptime % and historical latency points within < 30ms.
- [ ] **AC-8**: Process physical RAM remains under 25 MiB RSS with 20 active monitors probed continuously.

---

## 4. Testing Requirements
- **Unit Tests**:
  - `internal/prober/client`: Test HTTP success, 4xx/5xx responses, timeouts, keyword matching, and TLS inspection using local `httptest.Server`.
  - `internal/prober/scheduler`: Test monitor add, remove, pause, resume lifecycle with mock prober.
  - `internal/storage`: Test check log insertion, historical query aggregation, and rolling 14-day prune.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**:
- **Files Modified**:
- **Automated Test Execution**:
- **Known Limitations & Caveats**:
- **Potential Regression Risks**:
- **Handoff Date**:

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**:
- **Verdict**: `PENDING`
- **Audit Date**:
- **Human Approval Request**:
