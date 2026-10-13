# QA & Security Audit Report: TASK-XXX [Task Title]

## Audit Metadata
- **Task ID**: `TASK-XXX`
- **Target Commit**: `[commit-hash]`
- **Feature Branch**: `task/TASK-XXX-[short-name]`
- **Auditor**: QA / Security Agent
- **Audit Date**: YYYY-MM-DD
- **Final Verdict**: `APPROVED` <!-- Options: APPROVED | REJECTED | ACTION_REQUIRED -->

---

## 1. Executive Summary
[Concise executive overview of the QA audit findings, stability status, and security compliance.]

---

## 2. Acceptance Criteria Verification

> [!IMPORTANT]
> The Zero-Trust principle applies: Every criterion must be verified with concrete evidence. Any item not directly verified must be recorded as `NOT VERIFIED`.

| ID | Requirement Description | Verification Method / Evidence | Result | Notes / Logs |
|---|---|---|---|---|
| **AC-1** | [Requirement 1] | [Unit test output / API response check] | `PASS` | [Details] |
| **AC-2** | [Requirement 2] | [Mock test / Code inspection] | `PASS` | [Details] |
| **AC-3** | [Requirement 3] | [Verification step] | `NOT VERIFIED` | [Reason why unverified] |

---

## 3. Security & Resilience Audit Checklist

### 3.1 Credential & Secret Protection
- [ ] **Check 1**: Verified zero hardcoded API keys, tokens, or passwords in committed files.
- [ ] **Check 2**: Verified `.gitignore` blocks sensitive artifacts (`.env`, `*.key`, `*.token`).
- [ ] **Check 3**: Verified no secrets leaked into console logs or error outputs.

### 3.2 Input Validation, SSRF & Attack Defenses
- [ ] **Check 1**: URLs provided to synthetic probers are validated against private IP / SSRF filters.
- [ ] **Check 2**: Path traversal protections enforced on log-reading and file-access handlers.
- [ ] **Check 3**: All REST API input parameters are strictly validated and sanitized.

### 3.3 Isolation & Least Privilege
- [ ] **Check 1**: Docker socket mount is strictly read-only (`/var/run/docker.sock:ro`).
- [ ] **Check 2**: File permissions follow least-privilege principles.

### 3.4 Resource & Memory Footprint Budget
- [ ] **Check 1**: Verified daemon runtime memory remains within budget (< 25 MiB RAM).
- [ ] **Check 2**: Verified CPU usage is minimal when idle (< 0.5% CPU).
- [ ] **Check 3**: Goroutines and background probe loops properly manage contexts and timeouts (no goroutine leaks).

### 3.5 Language & Presentation Compliance
- [ ] **Check 1**: **100% English UI verified**: Zero Chinese characters in frontend HTML/CSS/JS.
- [ ] **Check 2**: **100% English Backend verified**: Zero Chinese characters in log messages, API responses, and code comments.

---

## 4. Discovered Defects & Blocker Log
*If any checks failed, document the details below.*

| Bug ID | Severity | Summary | Steps to Reproduce | Expected vs Actual | Remediation |
|---|---|---|---|---|---|
| **BUG-01** | High / Medium / Low | [Summary] | 1. Step 1<br>2. Step 2 | Expected: ...<br>Actual: ... | [Required Fix] |

---

## 5. Merge & Deployment Recommendation
- **Recommendation**: [PROCEED_WITH_MERGE / RETURN_TO_DEVELOPER / BLOCKED]
- **Residual Risks**: [Any residual risks or considerations for production]
- **Action Required by Human Lead**: [Approve merge of task branch into main]
