# QA & Security Audit Report: TASK-003 Monitor Target Management & SSRF Guard Engine

## Audit Metadata
- **Task ID**: `TASK-003`
- **Target Commit**: `bb88a024ac6991542173253923c0fc50ac505cf0`
- **Feature Branch**: `task/TASK-003-monitor-crud-and-ssrf-validation`
- **Auditor**: QA / Security Agent
- **Audit Date**: 2026-10-13 (Re-Audit)
- **Final Verdict**: `REJECTED`

---

## 1. Executive Summary
A comprehensive re-audit of TASK-003 was conducted following Developer Agent's remediation commit `bb88a02`. The audit verified the status of previously reported defects and examined remote GitHub Actions CI status on branch `task/TASK-003-monitor-crud-and-ssrf-validation`.

- **BUG-02 (Frontend quote hazard)**: **VERIFIED FIXED**. `openDeleteMonitorModal(id)` in `web/app.js` now accepts only the numeric monitor ID, queries the global `cachedMonitors` array, and completely removes inline single-quote string interpolation.
- **BUG-01 (SSRF test domain)**: **PARTIALLY ADDRESSED, BUT REVEALED UNDERLYING CRITICAL BUG (BUG-03)**. While `goodai.co.nz` was substituted with `cloudflare.com` in `internal/security/ssrf_test.go`, **remote GitHub Actions CI run `38356574339` still failed at Step 7 ("Run Automated Unit Tests")**.
- **Root Cause Analysis of CI Failure (BUG-03)**:
  In `internal/security/ssrf.go`, `restrictedIPNets` registers `"::ffff:0:0/96"` and `"::0.0.0.0/96"`. In Go's standard library `net` package, `(*net.IPNet).Contains(ip)` converts any IPv4 address to its 16-byte IPv4-mapped IPv6 representation (`ip.To16()`, where bytes 10 and 11 are `0xff`). When masked with a 96-bit mask, **every valid IPv4 address matches `"::ffff:0:0/96"`**.
  Consequently, `IsRestrictedIP()` returns `true` for **all public IPv4 destinations** (e.g., `8.8.8.8`, `1.1.1.1`, `93.184.216.34`, and any DNS A records resolved for `cloudflare.com` or `example.com`), causing `TestIsRestrictedIP`, `TestValidateTargetURL_AllowedURLs`, `TestCreateMonitorHandler_Success`, and `TestUpdateMonitorHandler` to fail.

In accordance with Zero-Trust QA protocols requiring 100% passing tests and green CI before merge, the branch is **REJECTED** and returned to Developer Agent (`IN_PROGRESS`) for resolution of BUG-03.

---

## 2. Acceptance Criteria Verification

> [!IMPORTANT]
> The Zero-Trust principle applies: Every criterion must be verified with concrete evidence. Any item not directly verified must be recorded as `NOT VERIFIED`.

| ID | Requirement Description | Verification Method / Evidence | Result | Notes / Logs |
|---|---|---|---|---|
| **AC-1** | Authenticated users can create monitors with Name, URL, Interval (60s/300s), Timeout (5s), Expected Status (200), and optional Keyword. | Code audit of `CreateMonitorHandler` in `internal/api/monitor_handlers.go`. | `PASS` | Logic and boundary validations verified. (Handler test in CI blocked by BUG-03). |
| **AC-2** | Strict SSRF Blocking: Submitting `127.0.0.1`, `localhost`, `169.254.169.254`, `10.0.0.1`, or `192.168.1.1` rejected with HTTP 400 and `Invalid or restricted target URL`. | Code audit of `ValidateTargetURL`, `IsRestrictedIP`, and tests `TestValidateTargetURL_RestrictedDestinations`. | `PASS` | Blocking of private, loopback, and metadata ranges verified. |
| **AC-3** | Forbidden port numbers (e.g. `60324`, `5244`, `22`) are rejected. | Code audit of `AllowedPorts` map (80, 443, 8080, 8443) and test suite `TestCreateMonitorHandler_SSRFRejection`. | `PASS` | Port whitelist strictly enforced. |
| **AC-4** | Per-user quota enforced: creating 11th monitor returns HTTP 403 Forbidden with clear quota exceeded explanation. | Code audit of `CountMonitorsByUser` check in `CreateMonitorHandler` and unit test `TestCreateMonitorHandler_QuotaEnforcement`. | `PASS` | HTTP 403 with `Monitor quota exceeded (maximum 10 monitors allowed)` verified. |
| **AC-5** | Multi-tenant isolation verified: User A only accesses User A's monitors; User B accessing User A's monitor returns HTTP 404 Not Found. | Code audit of row-level `WHERE id = ? AND user_id = ?` in `monitor_store.go` and cross-user API tests in `monitor_handlers_test.go`. | `PASS` | Cross-tenant GET, PUT, POST /pause, DELETE all return HTTP 404 `Monitor not found`. |
| **AC-6** | Pausing a monitor updates its status to `PAUSED`; resuming restores it to `PENDING`/`UP`. | Code audit of `ToggleMonitorPause` and API test `TestTogglePauseMonitorHandler`. | `PASS` | Status toggle between `PENDING` and `PAUSED` verified. |

---

## 3. Security & Resilience Audit Checklist

### 3.1 Credential & Secret Protection
- [x] **Check 1**: Verified zero hardcoded API keys, tokens, or passwords in committed files.
- [x] **Check 2**: Verified `.gitignore` blocks sensitive artifacts (`.env`, `*.db`, `*.key`).
- [x] **Check 3**: Verified no secrets or session tokens leaked into console logs or error outputs.

### 3.2 Input Validation, SSRF & Attack Defenses
- [x] **Check 1**: URLs provided to prober are validated against private IP / SSRF filters in `ValidateTargetURL`.
- [x] **Check 2**: Safe socket dialing (`SafeDialContext` in `internal/security/ssrf.go`) connects directly to pre-verified IP addresses, eliminating DNS Rebinding (TOCTOU) exploits.
- [x] **Check 3**: Open redirect SSRF bypass prevented via `http.ErrUseLastResponse` in `NewSafeHTTPClient`.
- [x] **Check 4**: All REST API input parameters are strictly validated and sanitized (lengths capped at 100 chars, timeouts bounded 1-30s).

### 3.3 Isolation & Least Privilege
- [x] **Check 1**: Storage queries enforce row-level `user_id` filtering on all CRUD operations (`Get`, `List`, `Update`, `Delete`, `TogglePause`).
- [x] **Check 2**: Unauthenticated access to monitor endpoints is blocked by `RequireAuth` returning HTTP 401 Unauthorized.

### 3.4 Resource & Memory Footprint Budget
- [x] **Check 1**: Pure-Go SQLite in WAL mode with single-writer connection pool respects memory budget.
- [x] **Check 2**: Quota enforcement bounds per-tenant storage usage to 10 monitors.
- [x] **Check 3**: Safe dialer socket timeouts are bounded (10s dial, 3s DNS context timeout).

### 3.5 Language & Presentation Compliance
- [x] **Check 1**: **100% English UI verified**: Zero Chinese characters found across `web/index.html` and `web/app.js`.
- [x] **Check 2**: **100% English Backend verified**: Zero Chinese characters across Go handlers, models, stores, security filters, logs, and comments.

---

## 4. Discovered Defects & Blocker Log

| Bug ID | Severity | Status | Summary | Steps to Reproduce | Expected vs Actual | Remediation |
|---|---|---|---|---|---|---|
| **BUG-01** | High | Closed | Remote CI failed due to unresolvable domain `goodai.co.nz` | Inspect `internal/security/ssrf_test.go` | Domain replaced with `cloudflare.com` in commit `bb88a02`. | Verified domain replacement in `internal/security/ssrf_test.go`. |
| **BUG-02** | Low | Closed | Potential inline JS quote syntax hazard on monitor delete | Create target named `Dylan's API`, click delete | Quotes could break onclick handler string literal. | Refactored `openDeleteMonitorModal` to accept only `id` and lookup name from `cachedMonitors`. Verified in `web/app.js`. |
| **BUG-03** | Critical (CI Blocker) | **Open** | `restrictedIPNets` includes `"::ffff:0:0/96"`, which erroneously matches and blocks **ALL public IPv4 addresses** in `IsRestrictedIP()` | 1. In Go: `ipNet, _ := net.ParseCIDR("::ffff:0:0/96"); ipNet.Contains(net.ParseIP("8.8.8.8"))`<br>2. Run `go test -v ./internal/security -run TestIsRestrictedIP`<br>3. Inspect GitHub Actions Run `38356574339` Step 7 | **Expected**: Routable public IPv4 addresses (`8.8.8.8`, `1.1.1.1`, `93.184.216.34`, `example.com`) pass `IsRestrictedIP()` as `false`.<br>**Actual**: `ip.To16()` for any IPv4 address contains `::ffff:` prefix; `ipNet.Contains()` returns `true`, causing `IsRestrictedIP()` to return `true` for all IPv4 destinations. | Remove `"::ffff:0:0/96"` and `"::0.0.0.0/96"` from `restrictedIPNets` in `internal/security/ssrf.go`. IPv4-mapped IPv6 addresses are already safely parsed to 4-byte IPv4 slices via `ip.To4()` (lines 86–95) and tested against private/loopback/metadata rules. |

---

## 5. Merge & Deployment Recommendation
- **Recommendation**: `RETURN_TO_DEVELOPER` (REJECTED back to `IN_PROGRESS`)
- **Residual Risks**: Remote GitHub Actions CI run `38356574339` on commit `bb88a02` is red (`failure`). The branch cannot be merged until unit tests pass and CI turns green.
- **Action Required by Developer Agent**:
  1. Modify `internal/security/ssrf.go` to remove `"::ffff:0:0/96"` and `"::0.0.0.0/96"` from the `restrictedIPNets` initialization list.
  2. Push the fix to `task/TASK-003-monitor-crud-and-ssrf-validation`.
  3. Verify that remote GitHub Actions CI run triggers and completes with `success` (green checkmark).
  4. Transition task status back to `REVIEW`.
