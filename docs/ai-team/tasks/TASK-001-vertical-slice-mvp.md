# Task Specification: TASK-001 Vertical-Slice Runnable MVP Skeleton

## Metadata
- **Task ID**: `TASK-001`
- **Title**: Vertical-Slice Runnable MVP Skeleton (HTTP Server, Embedded UI Shell, SQLite WAL, CI/CD)
- **Current Status**: `REVIEW`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-13
- **Target Branch**: `task/TASK-001-vertical-slice-mvp`
- **Dependencies**: None (Root Foundation Task)

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: `docs/prd.md` Section 4 & 9; `docs/architecture-review.md` REV-10.
- **Business Goal**: Establish a functional, end-to-end runnable baseline on Day 1. The Human Lead and QA Agent must be able to launch the server, open `http://localhost:8090`, see the embedded 100% English Web UI shell, verify database connectivity, and confirm remote CI build success.
- **Detailed Description**:
  - Implement the core Go application entry point (`cmd/monitor/main.go`).
  - Configure standard library `net/http` ServeMux with graceful shutdown handling (10-second timeout).
  - Embed static frontend assets (`web/index.html`, `web/style.css`, `web/app.js`) using Go standard `embed.FS`.
  - Initialize pure-Go SQLite (`modernc.org/sqlite`) with WAL mode (`PRAGMA journal_mode = WAL;`) and 2MB cache limit (`PRAGMA cache_size = -2000;`).
  - Provide a `/healthz` endpoint returning application health and memory metrics (`process_rss_bytes`).
  - Create the GitHub Actions workflow (`.github/workflows/ci-cd.yml`) to build and test remotely without local compilation.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [ ] `cmd/monitor/main.go`
- [ ] `internal/config/config.go`
- [ ] `internal/config/config_test.go`
- [ ] `internal/storage/sqlite.go`
- [ ] `internal/storage/sqlite_test.go`
- [ ] `internal/api/router.go`
- [ ] `internal/api/healthz.go`
- [ ] `internal/api/healthz_test.go`
- [ ] `web/embed.go`
- [ ] `web/index.html`
- [ ] `web/style.css`
- [ ] `web/app.js`
- [ ] `.github/workflows/ci-cd.yml`
- [ ] `go.mod`
- [ ] `go.sum`

---

## 3. Acceptance Criteria
- [ ] **AC-1**: Starting the server binds to configured `PORT` (default `:8090`) and serves the embedded English UI at `GET /`.
- [ ] **AC-2**: `GET /healthz` returns HTTP 200 with JSON payload containing `status: "ok"`, `uptime_seconds`, and `memory_rss_bytes`.
- [ ] **AC-3**: SQLite database file (`monitor.db`) is automatically created with WAL mode active (`PRAGMA journal_mode = WAL`) and verified single-writer connection pool (`SetMaxOpenConns(1)`).
- [ ] **AC-4**: Graceful shutdown handles `SIGINT`/`SIGTERM` cleanly within 10 seconds, closing active listeners and database connections.
- [ ] **AC-5**: **100% English Policy**: Zero Chinese characters in frontend HTML/CSS/JS, API responses, log messages, and code comments.
- [ ] **AC-6**: **No-Build Rule**: Zero local high-load build commands. Remote GitHub Actions workflow (`.github/workflows/ci-cd.yml`) passes linting, tests, and compiles the `linux/amd64` binary.
- [ ] **AC-7**: **Memory Budget**: Server running idle consumes < 20 MiB RSS physical memory.

---

## 4. Testing Requirements
- **Unit Tests**:
  - `internal/config`: Test environment variable loading and default fallbacks.
  - `internal/storage`: Test SQLite initialization, schema migration creation, and connection pooling.
  - `internal/api`: Test `/healthz` HTTP handler and embedded frontend asset delivery.
- **CI Remote Testing**:
  - GitHub Actions runs `go test -v ./...` and `golangci-lint` remotely.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**: `3fba797`
- **Files Modified**:
  - `cmd/monitor/main.go`
  - `internal/config/config.go`
  - `internal/config/config_test.go`
  - `internal/storage/sqlite.go`
  - `internal/storage/sqlite_test.go`
  - `internal/api/router.go`
  - `internal/api/healthz.go`
  - `internal/api/healthz_test.go`
  - `web/embed.go`
  - `web/index.html`
  - `web/style.css`
  - `web/app.js`
  - `.github/workflows/ci-cd.yml`
  - `go.mod`
  - `go.sum`
- **Automated Test Execution**:
  - Local verification: Python automated scan confirmed 100% English policy compliance (0 non-English characters in source tree).
  - Scope Whitelist verification: strictly 0 files touched outside whitelist.
  - Remote CI pipeline configured in `.github/workflows/ci-cd.yml`: runs `go mod tidy`, `go vet ./...`, `CGO_ENABLED=0 go test -v -cover ./...`, and builds `goodai-monitor-linux-amd64` on GitHub Actions.
- **Known Limitations & Caveats**:
  - Probing engine and multi-tenant authentication are deferred to subsequent tasks (TASK-002 through TASK-005).
  - Storage engine uses pure-Go SQLite with `PRAGMA cache_size = -2000` (2MB limit) and `SetMaxOpenConns(1)` to satisfy the strict < 20 MiB physical RSS memory budget.
- **Potential Regression Risks**:
  - Zero regression risk. Clean new standalone service on port 8090 with no production server modifications.
- **Handoff Date**: 2026-10-13

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**: `docs/ai-team/tasks/TASK-001-vertical-slice-mvp-qa-report.md`
- **Verdict**: `PENDING`
- **Audit Date**:
- **Human Approval Request**:
