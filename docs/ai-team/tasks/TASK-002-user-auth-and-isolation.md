# Task Specification: TASK-002 User Authentication & Multi-Tenant Data Isolation

## Metadata
- **Task ID**: `TASK-002`
- **Title**: User Authentication & Multi-Tenant Data Isolation (Bcrypt, Sessions, Auth Middleware)
- **Current Status**: `REVIEW`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-13
- **Target Branch**: `task/TASK-002-user-auth-and-isolation`
- **Dependencies**: `TASK-001`

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: `docs/prd.md` FR-AUTH-01 to FR-AUTH-04; Section 5.3 Data Protection.
- **Business Goal**: Enable secure, multi-tenant user account creation and authentication. Ensure User A can never view, mutate, or access User B's monitoring data.
- **Detailed Description**:
  - Implement `users` and `sessions` SQLite tables.
  - Implement user registration endpoint (`POST /api/v1/auth/register`) with email validation and `bcrypt` password hashing (cost $\ge 10$).
  - Implement user login endpoint (`POST /api/v1/auth/login`) generating cryptographically secure session tokens stored in `HttpOnly`, `SameSite=Lax` cookies.
  - Implement logout endpoint (`POST /api/v1/auth/logout`) and current user endpoint (`GET /api/v1/auth/me`).
  - Implement authentication middleware (`AuthMiddleware`) verifying session validity and injecting `userID` into the request context.
  - Implement frontend Login and Registration views in embedded English UI.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [x] `internal/model/user.go`
- [x] `internal/storage/user_store.go`
- [x] `internal/storage/user_store_test.go`
- [x] `internal/auth/password.go`
- [x] `internal/auth/password_test.go`
- [x] `internal/auth/session.go`
- [x] `internal/auth/session_test.go`
- [x] `internal/api/auth_handlers.go`
- [x] `internal/api/auth_handlers_test.go`
- [x] `internal/api/middleware.go`
- [x] `internal/api/middleware_test.go`
- [x] `web/index.html`
- [x] `web/app.js`

---

## 3. Acceptance Criteria
- [x] **AC-1**: Users can register with valid email and password ($\ge 8$ characters). Duplicate email registration returns clean HTTP 409 conflict error.
- [x] **AC-2**: Passwords are saved strictly as `bcrypt` hashes. Zero plaintext passwords in database, logs, or API payloads.
- [x] **AC-3**: Successful login issues a secure session cookie (`HttpOnly; Path=/; SameSite=Lax`) valid for 14 days.
- [x] **AC-4**: Unauthenticated requests to protected endpoints return HTTP 401 Unauthorized.
- [x] **AC-5**: Auth middleware injects verified `userID` into Go `context.Context` for downstream multi-tenant row-level queries.
- [x] **AC-6**: 100% English UI for registration, login forms, error messages, and session notifications.

---

## 4. Testing Requirements
- **Unit Tests**:
  - `internal/auth`: Test password hashing, verification, and failure on bad passwords.
  - `internal/storage`: Test user creation, duplicate email rejection, session storage and deletion.
  - `internal/api`: Test registration, login, logout, and auth middleware rejection of forged/expired cookies.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**: `8063ce2`
- **Files Modified**:
  - `internal/model/user.go`
  - `internal/storage/user_store.go`
  - `internal/storage/user_store_test.go`
  - `internal/auth/password.go`
  - `internal/auth/password_test.go`
  - `internal/auth/session.go`
  - `internal/auth/session_test.go`
  - `internal/api/auth_handlers.go`
  - `internal/api/auth_handlers_test.go`
  - `internal/api/middleware.go`
  - `internal/api/middleware_test.go`
  - `internal/api/router.go`
  - `web/index.html`
  - `web/app.js`
  - `go.mod`
- **Automated Test Execution**:
  - `internal/auth`: `TestHashPassword_Success`, `TestHashPassword_TooShort`, `TestVerifyPassword_Success`, `TestVerifyPassword_Mismatch`, `TestValidateEmail`, `TestGenerateSecureToken`, `TestNewSessionCookie`, `TestClearSessionCookie`
  - `internal/storage`: `TestUserStore_CreateAndGetUser`, `TestUserStore_DuplicateEmailRejection`, `TestUserStore_UserNotFound`, `TestUserStore_SessionLifecycle`, `TestUserStore_ExpiredSession`
  - `internal/api`: `TestRegisterHandler_Success`, `TestRegisterHandler_DuplicateEmail`, `TestRegisterHandler_ShortPassword`, `TestRegisterHandler_InvalidEmail`, `TestLoginHandler_Success`, `TestLoginHandler_InvalidCredentials`, `TestLogoutHandler`, `TestMeHandler`, `TestAuthMiddleware_ValidSession`, `TestRequireAuth_MissingSession`, `TestRequireAuth_ForgedSession`, `TestRequireAuth_ExpiredSession`
- **Known Limitations & Caveats**:
  - Session cleanup relies on periodic calls to `DeleteExpiredSessions` or checks on active token lookup; background probe worker or scheduled runner can be configured to periodically prune expired sessions.
- **Potential Regression Risks**:
  - Existing public `/healthz` endpoints and static frontend assets are preserved and verified functional.
- **Handoff Date**: 2026-10-13

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**: `docs/ai-team/tasks/TASK-002-user-auth-and-isolation-qa-report.md`
- **Verdict**: `APPROVED`
- **Audit Date**: 2026-10-13
- **Human Approval Request**: Ready for Human Lead review and merge into `main`. All 6 ACs passed, remote CI/CD pipeline succeeded (Run ID `38353050346`), zero security defects, and 100% English compliance verified.
