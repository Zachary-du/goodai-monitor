# Task Directory: GoodAI Monitor

This directory holds all atomic task definitions, execution logs, and QA audit reports for the GoodAI Monitor project.

## File Naming Conventions
- Task Specification: `TASK-XXX-[short-name].md`
- QA Audit Report: `TASK-XXX-[short-name]-qa-report.md`

## State Tracking
All task files maintain their current state in the metadata header:
- `BACKLOG`: Drafted by PM Agent, waiting for Human Lead approval.
- `READY`: Approved by Human Lead, available for Developer Agent pickup.
- `IN_PROGRESS`: Currently being implemented by Developer Agent in dedicated branch/worktree.
- `REVIEW`: Implementation complete, currently under independent audit by QA / Security Agent.
- `BLOCKED`: Work halted due to dependency, ambiguity, or critical defect.
- `DONE`: Passed QA audit and approved for merge by Human Lead.

---

## Phase 1 MVP Task Index (Awaiting Human Lead Approval)

| Task ID | Title | Status | Dependencies | Core Deliverable |
|---|---|---|---|---|
| [TASK-001](TASK-001-vertical-slice-mvp.md) | Vertical-Slice Runnable MVP Skeleton | `BACKLOG` | None | Go HTTP server (:8090), embedded English UI shell, SQLite WAL engine, /healthz, GHA CI |
| [TASK-002](TASK-002-user-auth-and-isolation.md) | User Auth & Multi-Tenant Isolation | `BACKLOG` | TASK-001 | User registration, login, bcrypt password hashing, secure session cookies, row-level isolation |
| [TASK-003](TASK-003-monitor-crud-and-ssrf-validation.md) | Monitor Target Management & SSRF Guard | `BACKLOG` | TASK-001, TASK-002 | Target CRUD API, quota (max 10), strict SafeDialer blocking private/IMDS IPs |
| [TASK-004](TASK-004-continuous-probing-engine.md) | Continuous Background Probing Engine | `BACKLOG` | TASK-001..003 | Concurrent scheduler, DNS/TCP/TLS/HTTP/Keyword telemetry, 14-day retention cleaner |
| [TASK-005](TASK-005-incident-lifecycle-and-email-alerts.md) | Incident State Machine & Email Alerts | `BACKLOG` | TASK-001..004 | N=2 failure confirm, M=2 recovery, deduplication, SMTP email alert dispatcher |
