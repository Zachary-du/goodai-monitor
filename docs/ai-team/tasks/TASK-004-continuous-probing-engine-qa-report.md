# QA & Security Audit Report: TASK-004 Continuous Background Probing Engine, Telemetry & Rolling Retention

## Audit Metadata
- **Task ID**: `TASK-004`
- **Target Commit**: `007d0767d7fa4848d1759eca666e6497783f8417` (`007d076`)
- **Feature Branch**: `task/TASK-004-continuous-probing-engine`
- **Auditor**: QA / Security Agent
- **Audit Date**: 2026-10-14 (Re-audit #2)
- **Final Verdict**: `REJECTED`

---

## 1. Executive Summary
An independent, zero-trust QA & Security re-audit was conducted on branch `task/TASK-004-continuous-probing-engine` targeting commit `007d076` following the Developer Agent's remediation attempts for `BUG-01` and `BUG-02` (commits `182ebf2` and `007d076`).

The re-audit findings are as follows:
1. **BUG-01 Remediation Verification**:
   - In `internal/prober/client.go:67-70`, `net.DefaultResolver.LookupIPAddr(ctx, host)` is now used, returning `[]net.IPAddr`.
   - `trace.DNSDone(httptrace.DNSDoneInfo{Addrs: ipAddrs, Err: err})` now perfectly matches Go's `httptrace.DNSDoneInfo` struct field definition.
   - Dial-time SSRF validation correctly inspects `addr.IP` with `security.IsRestrictedIP(addr.IP)`.
   - The specific compiler error reported in BUG-01 for `internal/prober/client.go` is **RESOLVED**.
2. **BUG-03 (New Compiler & CI Failure Blocker)**:
   - In `internal/prober/client_test.go:5:2`, `"fmt"` was imported but is not used anywhere in the file.
   - Go compiler rejects unused imports with: `"fmt" imported and not used`.
   - This defect caused remote GitHub Actions CI Run [`38469018380`](https://github.com/Zachary-du/goodai-monitor/actions/runs/38469018380) to fail on Job `Lint & Unit Tests` (exit code 1) on head commit `007d076`.
   - Because the test suite cannot compile, CI pipeline is red and static binary compilation was skipped.
3. **BUG-02 Scope Remediation Verification**:
   - In commit `182ebf2`, the Developer Agent attempted to restore `scripts/run.sh`.
   - However, `git diff origin/main..HEAD scripts/run.sh` shows that `scripts/run.sh` is **still modified**: line 9 still appends `git push origin task/TASK-004-continuous-probing-engine` and the original script comment is missing.
   - `scripts/run.sh` remains outside the allowed Section 2 Scope Whitelist of `TASK-004-continuous-probing-engine.md`.
   - Clean restoration requires reverting `scripts/run.sh` so `git diff origin/main..HEAD scripts/run.sh` is completely empty.

Due to CI pipeline failure (**BUG-03**) and the remaining scope modification on `scripts/run.sh` (**BUG-02**), the task is **REJECTED** and returned to the Developer Agent for immediate resolution.

---

## 2. Acceptance Criteria Verification

> [!IMPORTANT]
> The Zero-Trust principle applies: Every criterion must be verified with concrete evidence. Any item not directly verified or failing build/execution must be recorded as `FAIL` or `NOT VERIFIED`.

| ID | Requirement Description | Verification Method / Evidence | Result | Notes / Logs |
|---|---|---|---|---|
| **AC-1** | Active monitors are probed periodically at their configured intervals (e.g. every 60s) completely independent of user browser presence. | Inspection of `internal/prober/scheduler.go` (`runMonitorJob` ticker loop) and `scheduler_test.go`. | `FAIL` | Blocked by compilation failure in `internal/prober/client_test.go:5:2`; CI Run 38469018380 failed. |
| **AC-2** | Prober measures accurate latency breakdown (DNS, TCP, TLS, TTFB, Total) and stores check records. | `httptrace.ClientTrace` in `internal/prober/client.go`; `check_store.go` schema migration and `RecordCheck`. | `FAIL` | Package compilation blocked by unused import in `internal/prober/client_test.go`. |
| **AC-3** | Failed checks accurately record specific failure reason (e.g. `HTTP 502 Bad Gateway`, `Connection timeout after 5000ms`, `Keyword 'Welcome' missing`). | Error formatting in `internal/prober/client.go` lines 220-226, 258-261, 274; unit tests in `client_test.go`. | `FAIL` | Blocked by package compile failure in `internal/prober/client_test.go:5:2`. |
| **AC-4** | Paused monitors are excluded from probing; resuming dynamically schedules the probe without requiring server restart. | Lifecycle methods in `internal/prober/scheduler.go` (`AddOrUpdateMonitor`, `SyncMonitors`) and `scheduler_test.go` (`TestScheduler_PauseAndResume`). | `FAIL` | Verification blocked by project-wide CI test failure. |
| **AC-5** | `SafeDialContext` prevents DNS rebinding by resolving IP at dial time and connecting directly to verified IP. | Safe dial closure in `internal/prober/client.go` lines 40-88 resolves IP at dial time, checks `security.IsRestrictedIP(addr.IP)`, and dials verified IP directly. Type mismatch in `DNSDone` resolved. | `FAIL` | Verification blocked by CI build failure caused by `client_test.go`. |
| **AC-6** | Automated retention job purges check records older than 14 days every 24 hours. | `internal/storage/check_store.go` (`PruneOldChecks`) and `scheduler.go` (startup + 24h ticker); `check_store_test.go` (`TestCheckStore_PruneOldChecks`). | `FAIL` | Verification blocked by CI failure. |
| **AC-7** | Telemetry API returns 24-hour uptime % and historical latency points within < 30ms. | Endpoints `/api/v1/monitors/{id}/metrics` and `/checks` in `metrics_handlers.go`, indexed SQLite query `idx_checks_monitor_created`. | `FAIL` | Verification blocked by CI failure. |
| **AC-8** | Process physical RAM remains under 25 MiB RSS with 20 active monitors probed continuously. | End-to-end memory stress test. | `NOT VERIFIED` | Cannot run compiled daemon binary due to CI build failure. |

---

## 3. Security & Resilience Audit Checklist

### 3.1 Credential & Secret Protection
- [x] **Check 1**: Verified zero hardcoded API keys, tokens, or passwords in committed files.
- [x] **Check 2**: Verified `.gitignore` blocks sensitive artifacts (`.env`, `*.db`, `*.key`, `*.token`).
- [x] **Check 3**: Verified no secrets or session tokens leak into console logs or error outputs.

### 3.2 Input Validation, SSRF & Attack Defenses
- [x] **Check 1**: URLs provided to synthetic prober are validated against private IP / SSRF filters in dialer (`security.IsRestrictedIP(addr.IP)` at dial time).
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
- [x] **Check 2**: **100% English Backend verified**: Zero Chinese characters in Go source code, logs, and comments (verified via AST/regex scanner over all modified files).

---

## 4. Discovered Defects & Blocker Log

| Bug ID | Severity | Status | Summary | Steps to Reproduce | Expected vs Actual | Remediation |
|---|---|---|---|---|---|---|
| **BUG-01** | Critical (CI Blocker) | **Resolved** | Type mismatch in `internal/prober/client.go:69:47` (`[]net.IP` passed to `[]net.IPAddr`) prevented Go compiler from building `internal/prober` package. | In commit `182ebf2`, replaced `LookupIP` with `net.DefaultResolver.LookupIPAddr(ctx, host)`. Checked `addr.IP` with `security.IsRestrictedIP`. | **Expected**: `trace.DNSDone` receives `[]net.IPAddr`.<br>**Actual**: Correctly resolved in commit `182ebf2`. | Verified resolved. |
| **BUG-02** | Low / Scope Blocker | **Open** | `scripts/run.sh` remains modified on feature branch; not in Section 2 Scope Whitelist. | Run `git diff origin/main..HEAD scripts/run.sh`. Line 9 still contains `git push origin task/TASK-004-continuous-probing-engine`. | **Expected**: `scripts/run.sh` identical to `origin/main` (`git diff origin/main..HEAD scripts/run.sh` is empty).<br>**Actual**: File is still modified in branch diff. | Run `git checkout origin/main -- scripts/run.sh` and commit. |
| **BUG-03** | Critical (CI Blocker) | **Open** | Unused import `"fmt"` in `internal/prober/client_test.go:5:2` fails compilation and breaks GitHub Actions CI. | 1. Inspect `internal/prober/client_test.go:5`.<br>2. Check GitHub Actions Run `38469018380` Job `Lint & Unit Tests` (exit code 1). Annotation: `"fmt" imported and not used`. | **Expected**: Test package compiles and runs clean.<br>**Actual**: Go compiler fails with `"fmt" imported and not used`. CI job `Lint & Unit Tests` fails. | Remove `"fmt"` from the import declaration in `internal/prober/client_test.go`. |

---

## 5. Merge & Deployment Recommendation
- **Recommendation**: `RETURN_TO_DEVELOPER`
- **Residual Risks**: Test suite does not compile due to unused import in `internal/prober/client_test.go`; remote CI pipeline is red; `scripts/run.sh` violates scope whitelist.
- **Action Required by Developer Agent**:
  1. Remove `"fmt"` from `internal/prober/client_test.go` (resolving BUG-03).
  2. Restore `scripts/run.sh` cleanly using `git checkout origin/main -- scripts/run.sh` (resolving BUG-02).
  3. Push commits to `task/TASK-004-continuous-probing-engine`.
  4. Verify that remote GitHub Actions CI run succeeds (green checkmark on `Lint & Unit Tests` and `Build Linux Static Binary`).
  5. Transition task status back to `REVIEW`.
