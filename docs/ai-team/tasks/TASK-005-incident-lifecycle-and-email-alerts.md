# Task Specification: TASK-005 Incident State Machine & SMTP Email Alerts

## Metadata
- **Task ID**: `TASK-005`
- **Title**: Incident Lifecycle State Machine & SMTP Email Alerting Dispatcher
- **Current Status**: `BACKLOG`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-13
- **Target Branch**: `task/TASK-005-incident-lifecycle-and-email-alerts`
- **Dependencies**: `TASK-001`, `TASK-002`, `TASK-003`, `TASK-004`

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: `docs/prd.md` FR-INC-01 to FR-INC-04; FR-ALERT-01 to FR-ALERT-03.
- **Business Goal**: Implement the incident management engine with anti-flapping confirmation and dispatch real email alerts upon confirmed downtime and recovery.
- **Detailed Description**:
  - Implement `incidents` table in SQLite schema with fields: `id`, `monitor_id`, `user_id`, `started_at`, `resolved_at`, `status`, `cause`.
  - Implement incident state machine in prober feedback loop:
    - Confirms `DOWN` transition only after $N=2$ consecutive failed checks, creating an `OPEN` incident.
    - Confirms `UP` transition only after $M=2$ consecutive successful checks, resolving the incident and calculating total duration.
  - Implement SMTP Email Dispatcher (`internal/alert/email.go`):
    - Connects to SMTP relay via TLS/STARTTLS using credentials loaded from `.env`.
    - Dispatches formatted incident alert email on `OPEN` (Monitor name, failure time in NZDT & UTC, error details).
    - Dispatches recovery email on `RESOLVED` (total downtime duration, recovery timestamp).
    - Masks credentials and email addresses in server logs.
    - Gracefully logs SMTP errors without interrupting monitoring operations.
  - Implement Incident Hub API (`GET /api/v1/incidents`) with status filters (`?status=open`, `?status=resolved`).
  - Render Incident Hub view and active outage alerts in embedded UI.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [ ] `internal/model/incident.go`
- [ ] `internal/storage/incident_store.go`
- [ ] `internal/storage/incident_store_test.go`
- [ ] `internal/alert/email.go`
- [ ] `internal/alert/email_test.go`
- [ ] `internal/alert/manager.go`
- [ ] `internal/alert/manager_test.go`
- [ ] `internal/api/incident_handlers.go`
- [ ] `internal/api/incident_handlers_test.go`
- [ ] `web/index.html`
- [ ] `web/app.js`

---

## 3. Acceptance Criteria
- [ ] **AC-1**: Single isolated check failure leaves monitor in `DEGRADED` state and does **not** trigger an incident or send an email.
- [ ] **AC-2**: Second consecutive check failure confirms downtime, creates an `OPEN` incident row, and dispatches an outage alert email.
- [ ] **AC-3**: Outage alert email contains monitor name, target URL, timestamp in Pacific/Auckland (NZDT) & UTC, and root cause error.
- [ ] **AC-4**: Two consecutive successful checks resolve the incident, update `resolved_at`, and dispatch a recovery email stating total downtime duration.
- [ ] **AC-5**: Anti-spam rate limiting: maximum 1 email per incident; no repeated emails during an ongoing outage.
- [ ] **AC-6**: SMTP network or authentication failures are logged gracefully with masked credentials and do not crash background probe workers.
- [ ] **AC-7**: Multi-tenant isolation verified: User A querying `/api/v1/incidents` sees strictly their own incidents.
- [ ] **AC-8**: Embedded UI displays active incident alert banner and consolidated incident history table.

---

## 4. Testing Requirements
- **Unit Tests**:
  - `internal/alert`: Test state transitions ($N=2$ failures $\rightarrow$ OPEN; $M=2$ successes $\rightarrow$ RESOLVED), deduplication, and mock SMTP sender verification.
  - `internal/storage`: Test incident creation, resolution timestamping, and multi-tenant queries.
  - `internal/api`: Test incident listing API filtering by status.

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
