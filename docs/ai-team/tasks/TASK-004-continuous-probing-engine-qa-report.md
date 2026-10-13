# QA & Security Audit Report: TASK-004 Continuous Background Probing Engine, Telemetry & Rolling Retention

## Audit Metadata
- **Task ID**: `TASK-004`
- **Target Commit**: `d45089f1e5a6afd30c91ee8f22ca2639b6a55789` (`d45089f`)
- **Feature Branch**: `task/TASK-004-continuous-probing-engine`
- **Auditor**: QA / Security Agent
- **Audit Date**: 2026-10-14
- **Final Verdict**: `REJECTED`

---

## 1. Executive Summary
An independent, zero-trust QA & Security audit was conducted on branch `task/TASK-004-continuous-probing-engine` targeting commit `d45089f`.

The audit identified two blocking defects requiring immediate remediation:
1. **BUG-01 (Compile Error & CI Blocker)**: In `internal/prober/client.go:69:47`, `ips` (`[]net.IP`) is passed into `httptrace.DNSDoneInfo{Addrs: ips, Err: err}`. In Go's standard library `net/http/httptrace`, `Addrs` requires type `[]net.IPAddr`. This causes Go compiler error `cannot use ips (variable of type []net.IP) as []net.IPAddr value in struct literal`, leading to immediate failure of GitHub Actions CI Run `38466602911` at step `Lint & Unit Tests` (exit code 1) and skipping binary compilation.
2. **BUG-02 (Scope Whitelist Violation)**: In commit `5cd107d`, `scripts/run.sh` was modified (replacing the unified ops runner script with a hardcoded `git push` command). `scripts/run.sh` is outside the allowed scope whitelist defined in Section 2 of `TASK-004-continuous-probing-engine.md`.

All other dimensions of the implementation were audited:
- **Probing Security**: Redirect blocking (`http.ErrUseLastResponse`) and 64KB body read truncation (`MaxKeywordScanBytes` with `io.LimitReader`) are correctly implemented.
- **Concurrency & Goroutine Safety**: Clean per-monitor ticker goroutine management, dynamic add/update/pause/resume lifecycle, and clean cancellation via context and `sync.WaitGroup`.
- **Multi-Tenant Isolation**: `GET /api/v1/monitors/{id}/metrics` and `GET /api/v1/monitors/{id}/checks` verify user ownership and return HTTP 404 on cross-tenant access.
- **Rolling Retention**: 14-day prune query (`DELETE FROM checks WHERE created_at < ?`) supported by indices and scheduled on startup + 24h ticker.
- **Language Policy**: Verified 100% English across all UI, log messages, error strings, and comments (zero Chinese characters).

Due to BUG-01 and BUG-02, the task is **REJECTED** and returned to the Developer Agent for remediation.

---

## 2. Acceptance Criteria Verification

> [!IMPORTANT]
> The Zero-Trust principle applies: Every criterion must be verified with concrete evidence. Any item not directly verified or failing build/execution must be recorded as `FAIL` or `NOT VERIFIED`.

| ID | Requirement Description | Verification Method / Evidence | Result | Notes / Logs |
|---|---|---|---|---|
| **AC-1** | Active monitors are probed periodically at their configured intervals (e.g. every 60s) completely independent of user browser presence. | Inspection of `internal/prober/scheduler.go` (`runMonitorJob` ticker loop) and `scheduler_test.go`; blocked by compile error in `client.go`. | `FAIL` | Blocked by package compile failure in `internal/prober/client.go:69:47`; CI Run 38466602911 failed. |
| **AC-2** | Prober measures accurate latency breakdown (DNS, TCP, TLS, TTFB, Total) and stores check records. | `httptrace.ClientTrace` in `internal/prober/client.go`; `check_store.go` schema migration and `RecordCheck`. | `FAIL` | Blocked by type mismatch at `client.go:69:47` (`ips` as `[]net.IPAddr`). |
| **AC-3** | Failed checks accurately record specific failure reason (e.g. `HTTP 502 Bad Gateway`, `Connection timeout after 5000ms`, `Keyword 'Welcome' missing`). | Error formatting in `internal/prober/client.go` lines 220-226, 258-261, 274; unit tests in `client_test.go`. | `FAIL` | Verification blocked by compilation failure of `internal/prober`. |
| **AC-4** | Paused monitors are excluded from probing; resuming dynamically schedules the probe without requiring server restart. | Lifecycle methods in `internal/prober/scheduler.go` (`AddOrUpdateMonitor`, `SyncMonitors`) and `scheduler_test.go` (`TestScheduler_PauseAndResume`). | `FAIL` | Verification blocked by compilation failure of `internal/prober`. |
| **AC-5** | `SafeDialContext` prevents DNS rebinding by resolving IP at dial time and connecting directly to verified IP. | Safe dial closure in `internal/prober/client.go` lines 40-88 resolves IP at dial time, checks `security.IsRestrictedIP`, and dials verified IP directly. | `FAIL` | Defect in `DNSDone` trace callback at line 69 prevents compilation. |
| **AC-6** | Automated retention job purges check records older than 14 days every 24 hours. | `internal/storage/check_store.go` (`PruneOldChecks`) and `scheduler.go` (startup + 24h ticker); `check_store_test.go` (`TestCheckStore_PruneOldChecks`). | `FAIL` | Verification blocked by project-wide CI build failure. |
| **AC-7** | Telemetry API returns 24-hour uptime % and historical latency points within < 30ms. | Endpoints `/api/v1/monitors/{id}/metrics` and `/checks` in `metrics_handlers.go`, indexed SQLite query `idx_checks_monitor_created`. | `FAIL` | Verification blocked by project-wide CI build failure. |
| **AC-8** | Process physical RAM remains under 25 MiB RSS with 20 active monitors probed continuously. | End-to-end memory stress test. | `NOT VERIFIED` | Cannot run compiled daemon binary due to CI build failure. |

---

## 3. Security & Resilience Audit Checklist

### 3.1 Credential & Secret Protection
- [x] **Check 1**: Verified zero hardcoded API keys, tokens, or passwords in committed files.
- [x] **Check 2**: Verified `.gitignore` blocks sensitive artifacts (`.env`, `*.db`, `*.key`, `*.token`).
- [x] **Check 3**: Verified no secrets or session tokens leak into console logs or error outputs.

### 3.2 Input Validation, SSRF & Attack Defenses
- [ ] **Check 1**: URLs provided to synthetic prober are validated against private IP / SSRF filters in dialer (Blocked by compile defect BUG-01 in `client.go`).
- [x] **Check 2**: Open redirect SSRF bypass prevented via `http.ErrUseLastResponse` in `NewClient` and `NewTestClient`.
- [x] **Check 3**: Response body scanning capped at 64 KB (`MaxKeywordScanBytes`) via `io.LimitReader` to prevent memory exhaustion / OOM attacks.
- [x] **Check 4**: All REST API input parameters are strictly validated (`parseMetricsMonitorID` enforces numeric integer, query `limit` clamped between 1 and 100).

### 3.3 Isolation & Least Privilege
- [x] **Check 1**: Multi-tenant isolation verified: `MonitorMetricsHandler` and `MonitorChecksHandler` verify monitor ownership against `userID`, returning HTTP 404 `Monitor not found` for cross-tenant attempts.
- [x] **Check 2**: Unauthenticated access to telemetry endpoints is blocked by `RequireAuth` returning HTTP 401 Unauthorized.

### 3.4 Resource & Memory Footprint Budget
- [ ] **Check 1**: Verified daemon runtime memory remains within budget (< 25 MiB RAM). (NOT VERIFIED due to build failure).
- [x] **Check 2**: Scheduler goroutines cleanly manage contexts, tickers (`ticker.Stop()`), and cancel functions; graceful shutdown in `cmd/monitor/main.go` terminates background scheduler with `scheduler.Stop()` before HTTP server drain.
- [x] **Check 3**: Pruning query uses index `idx_checks_monitor_created` / `idx_checks_created_at`.

### 3.5 Language & Presentation Compliance
- [x] **Check 1**: **100% English UI verified**: Zero Chinese characters across `web/index.html` and `web/app.js`.
- [x] **Check 2**: **100% English Backend verified**: Zero Chinese characters in Go source code, logs, and comments (verified via `git grep -P "[\x{4e00}-\x{9fa5}]"`).

---

## 4. Discovered Defects & Blocker Log

| Bug ID | Severity | Status | Summary | Steps to Reproduce | Expected vs Actual | Remediation |
|---|---|---|---|---|---|---|
| **BUG-01** | Critical (CI Blocker) | **Open** | Type mismatch in `internal/prober/client.go:69:47` (`[]net.IP` passed to `[]net.IPAddr`) prevents Go compiler from building `internal/prober` package. | 1. Inspect `internal/prober/client.go` lines 67-70.<br>2. Run `go vet ./...` or `go test ./internal/prober/...` in Go 1.22+.<br>3. Inspect GitHub Actions Run `38466602911` Job `Lint & Unit Tests`. | **Expected**: Code compiles cleanly and all unit tests pass.<br>**Actual**: Compiler error: `cannot use ips (variable of type []net.IP) as []net.IPAddr value in struct literal`. CI job fails with exit code 1. | Replace `LookupIP` with `LookupIPAddr(ctx, host)` returning `[]net.IPAddr`, pass `Addrs: ipAddrs` to `trace.DNSDone`, and check `addr.IP` against `security.IsRestrictedIP`. |
| **BUG-02** | Low / Scope Blocker | **Open** | Modification of `scripts/run.sh` is outside the allowed Section 2 Scope Whitelist. | 1. Inspect `git diff origin/main..HEAD --name-only`.<br>2. Notice `scripts/run.sh` was modified in commit `5cd107d`.<br>3. Check Section 2 Scope Whitelist of `TASK-004-continuous-probing-engine.md`. | **Expected**: Only files in the Section 2 Scope Whitelist are modified. `scripts/run.sh` remains the project's unified ops runner script.<br>**Actual**: `scripts/run.sh` was overwritten with a hardcoded `git push` command. | Revert `scripts/run.sh` to its original contents as in `main`. |

---

## 5. Merge & Deployment Recommendation
- **Recommendation**: `RETURN_TO_DEVELOPER`
- **Residual Risks**: Probing engine cannot compile until BUG-01 is fixed; CI pipeline is failing.
- **Action Required by Developer Agent**:
  1. Resolve BUG-01 by switching to `net.DefaultResolver.LookupIPAddr(ctx, host)` in `internal/prober/client.go`.
  2. Resolve BUG-02 by restoring `scripts/run.sh` to its clean version.
  3. Push commits to `task/TASK-004-continuous-probing-engine` and verify GitHub Actions CI run succeeds.
  4. Transition task status back to `REVIEW`.
