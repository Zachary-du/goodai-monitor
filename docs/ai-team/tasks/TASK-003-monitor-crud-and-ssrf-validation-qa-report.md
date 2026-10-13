# QA & Security Audit Report: TASK-003 Monitor Target Management & SSRF Guard Engine

## Audit Metadata
- **Task ID**: `TASK-003`
- **Target Commit**: `a4e6e399b71c24fd0cae14b8cff760ce932a8d31`
- **Feature Branch**: `task/TASK-003-monitor-crud-and-ssrf-validation`
- **Auditor**: QA / Security Agent
- **Audit Date**: 2026-10-13
- **Final Verdict**: `REJECTED`

---

## 1. Executive Summary
An independent, zero-trust security and resilience verification of TASK-003 (`task/TASK-003-monitor-crud-and-ssrf-validation`) was conducted. The feature introduces SQLite monitor CRUD storage with strict user isolation, quota limits (max 10 monitors per account), an SSRF guard engine (`internal/security/ssrf.go`) enforcing port whitelisting (80, 443, 8080, 8443) and RFC 1918/IMDS/loopback blocking, and complete frontend modal workflows.

While the security architecture, multi-tenant isolation, quota protection, and 100% English compliance were verified to be exceptionally robust, **the remote GitHub Actions CI pipeline failed on run `38354994913`**. Specifically, Step 7 ("Run Automated Unit Tests") failed because `internal/security/ssrf_test.go` included `"https://goodai.co.nz"` in the `allowedURLs` test suite, but `goodai.co.nz` currently lacks active public DNS A/AAAA records, causing `net.DefaultResolver.LookupIP` to return `no such host` and triggering test failure. Additionally, a minor frontend escaping edge case was identified in `web/app.js` when deleting monitors containing single quotes.

In accordance with Zero-Trust QA protocol and strict merge policies, the branch is **REJECTED** and returned to Developer Agent (`IN_PROGRESS`) for remediation.

---

## 2. Acceptance Criteria Verification

> [!IMPORTANT]
> The Zero-Trust principle applies: Every criterion must be verified with concrete evidence. Any item not directly verified must be recorded as `NOT VERIFIED`.

| ID | Requirement Description | Verification Method / Evidence | Result | Notes / Logs |
|---|---|---|---|---|
| **AC-1** | Authenticated users can create monitors with Name, URL, Interval (60s/300s), Timeout (5s), Expected Status (200), and optional Keyword. | Code audit of `CreateMonitorHandler` & unit test `TestCreateMonitorHandler_Success` in `internal/api/monitor_handlers_test.go`. | `PASS` | Payload parsing, boundary validations (interval >= 10s, timeout 1-30s, status 100-599), and SQLite persistence verified. |
| **AC-2** | Strict SSRF Blocking: Submitting `127.0.0.1`, `localhost`, `169.254.169.254`, `10.0.0.1`, or `192.168.1.1` rejected with HTTP 400 and `Invalid or restricted target URL`. | Code audit of `ValidateTargetURL`, `IsRestrictedIP`, and unit tests `TestCreateMonitorHandler_SSRFRejection`, `TestValidateTargetURL_RestrictedDestinations`. | `PASS` | Blocks loopback, RFC 1918, Oracle Cloud IMDS (`169.254.0.0/16`), Carrier-Grade NAT, IPv6 link-local (`fe80::/10`), IPv4-mapped IPv6, and localhost hostnames with uniform error message. |
| **AC-3** | Forbidden port numbers (e.g. `60324`, `5244`, `22`) are rejected. | Code audit of `AllowedPorts` map (80, 443, 8080, 8443) and test suite `TestCreateMonitorHandler_SSRFRejection`. | `PASS` | All non-whitelisted ports are rejected immediately with HTTP 400 `Invalid or restricted target URL` prior to socket connection. |
| **AC-4** | Per-user quota enforced: creating 11th monitor returns HTTP 403 Forbidden with clear quota exceeded explanation. | Code audit of `CountMonitorsByUser` check in `CreateMonitorHandler` and unit test `TestCreateMonitorHandler_QuotaEnforcement`. | `PASS` | Returns HTTP 403 Forbidden with `Monitor quota exceeded (maximum 10 monitors allowed)`. Frontend disables `+ Add Target` button upon reaching 10. |
| **AC-5** | Multi-tenant isolation verified: User A only accesses User A's monitors; User B accessing User A's monitor returns HTTP 404 Not Found. | Code audit of row-level `WHERE id = ? AND user_id = ?` in `monitor_store.go` and cross-user API tests in `monitor_handlers_test.go`. | `PASS` | Cross-tenant GET, PUT, POST /pause, DELETE all return HTTP 404 `Monitor not found` without leaking monitor existence. |
| **AC-6** | Pausing a monitor updates its status to `PAUSED`; resuming restores it to `PENDING`/`UP`. | Code audit of `ToggleMonitorPause` and API test `TestTogglePauseMonitorHandler`. | `PASS` | Status alternates between `PENDING` and `PAUSED`. |

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

| Bug ID | Severity | Summary | Steps to Reproduce | Expected vs Actual | Remediation |
|---|---|---|---|---|---|
| **BUG-01** | High (CI Blocker) | Remote GitHub Actions CI test suite failed on `TestValidateTargetURL_AllowedURLs` due to unresolvable domain `goodai.co.nz` | 1. Run `CGO_ENABLED=0 go test -v -run TestValidateTargetURL_AllowedURLs ./internal/security`<br>2. Inspect GitHub Actions Run `38354994913` Step 7 | **Expected**: All test URLs pass.<br>**Actual**: `net.DefaultResolver.LookupIP` failed with `lookup goodai.co.nz: no such host`, causing `ValidateTargetURL` to return `ErrRestrictedTargetURL` and fail the CI build. | Remove `"https://goodai.co.nz"` from `allowedURLs` in `internal/security/ssrf_test.go` or replace with an established resolvable public test domain (e.g. `https://cloudflare.com`). |
| **BUG-02** | Low | Potential inline JavaScript syntax error when rendering delete modal trigger for monitors containing single quotes | 1. Create a monitor with single quote (e.g., `Dylan's API`).<br>2. Click Delete button in web UI. | **Expected**: Delete modal opens cleanly.<br>**Actual**: Browser HTML entity decoding turns `&#039;` into `'` inside `onclick="openDeleteMonitorModal(id, 'name')"`, causing unhandled `SyntaxError: Unexpected identifier`. | Change `openDeleteMonitorModal` to take only monitor `id` and read the name from the monitor object or DOM dataset. |

---

## 5. Merge & Deployment Recommendation
- **Recommendation**: `RETURN_TO_DEVELOPER` (REJECTED back to `IN_PROGRESS`)
- **Residual Risks**: Remote CI build is red (`failure`). Branch cannot be merged to `main` until all automated tests pass in GitHub Actions.
- **Action Required by Developer Agent**:
  1. Fix `internal/security/ssrf_test.go` to eliminate the unresolvable `goodai.co.nz` test case.
  2. Fix `web/app.js` delete modal invocation to prevent quote unescaping syntax errors.
  3. Push changes and ensure GitHub Actions CI pipeline passes with green checkmark.
  4. Transition task back to `REVIEW`.
