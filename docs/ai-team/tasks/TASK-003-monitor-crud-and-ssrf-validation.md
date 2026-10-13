# Task Specification: TASK-003 Monitor Target Management & SSRF Guard Engine

## Metadata
- **Task ID**: `TASK-003`
- **Title**: Monitor Target Management (CRUD, Quota) & SSRF Validation Engine
- **Current Status**: `BACKLOG`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-13
- **Target Branch**: `task/TASK-003-monitor-crud-and-ssrf-validation`
- **Dependencies**: `TASK-001`, `TASK-002`

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: `docs/prd.md` FR-MON-01 to FR-MON-04; NFR-SEC-01 to NFR-SEC-04 (SSRF Defense).
- **Business Goal**: Allow authenticated users to create, view, edit, pause, and delete website monitoring targets, while strictly preventing SSRF attacks against internal network ports and cloud metadata services.
- **Detailed Description**:
  - Implement `monitors` table in SQLite schema linked to `user_id`.
  - Implement Monitor REST API (`GET /api/v1/monitors`, `POST /api/v1/monitors`, `GET /api/v1/monitors/:id`, `PUT /api/v1/monitors/:id`, `DELETE /api/v1/monitors/:id`, `POST /api/v1/monitors/:id/pause`).
  - Enforce per-user quota (maximum 10 active monitors per account).
  - Implement robust URL validation and SSRF defense package (`internal/security/ssrf.go`):
    - Rejects non-HTTP/HTTPS schemes.
    - Resolves hostname and validates against forbidden IP ranges (Loopback `127.0.0.0/8`, RFC 1918 `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, Oracle Cloud IMDS `169.254.169.254`, IPv6 Link-Local `fe80::/10`).
    - Restricts port whitelist to standard web ports (`80`, `443`, `8080`, `8443`).
  - Provide frontend UI for Monitor listing, Add Monitor form, Pause/Resume toggle, and Delete confirmation.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [ ] `internal/model/monitor.go`
- [ ] `internal/storage/monitor_store.go`
- [ ] `internal/storage/monitor_store_test.go`
- [ ] `internal/security/ssrf.go`
- [ ] `internal/security/ssrf_test.go`
- [ ] `internal/api/monitor_handlers.go`
- [ ] `internal/api/monitor_handlers_test.go`
- [ ] `web/index.html`
- [ ] `web/app.js`

---

## 3. Acceptance Criteria
- [ ] **AC-1**: Authenticated users can create monitors with Name, URL, Interval (60s/300s), Timeout (5s), Expected Status (200), and optional Keyword.
- [ ] **AC-2**: **Strict SSRF Blocking**: Submitting `http://127.0.0.1`, `http://localhost`, `http://169.254.169.254/opc/v2/instance/`, `http://10.0.0.1`, or `http://192.168.1.1` is rejected with HTTP 400 and clear English error: `Invalid or restricted target URL`.
- [ ] **AC-3**: Forbidden port numbers (e.g. `http://example.com:60324`, `:5244`, `:22`) are rejected.
- [ ] **AC-4**: Per-user quota enforced: creating an 11th monitor returns HTTP 403 Forbidden with clear quota exceeded explanation.
- [ ] **AC-5**: Multi-tenant isolation verified: User A querying `/api/v1/monitors` receives strictly User A's monitors; attempting to read or edit User B's monitor ID returns HTTP 404 Not Found.
- [ ] **AC-6**: Pausing a monitor updates its status to `PAUSED`; resuming restores it to `PENDING`/`UP`.

---

## 4. Testing Requirements
- **Unit Tests**:
  - `internal/security/ssrf`: Exhaustive test suite testing loopbacks, RFC1918, cloud metadata IPs, IPv6 mapped IPv4, valid public domains (`google.com`, `example.org`).
  - `internal/storage`: Test monitor CRUD with row-level `user_id` filtering.
  - `internal/api`: Test API validation, quota limits, and multi-tenant isolation barriers.

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
