# Task Specification: TASK-009 Google Account Single Sign-On (OAuth 2.0 / OIDC)

## Metadata
- **Task ID**: `TASK-009`
- **Title**: Google Account Single Sign-On (OAuth 2.0 / OpenID Connect)
- **Current Status**: `READY` <!-- Options: BACKLOG | READY | IN_PROGRESS | REVIEW | BLOCKED | DONE -->
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-15
- **Target Branch**: `task/TASK-009-google-oauth-sso`
- **Dependencies**: `TASK-001`, `TASK-002`

---

## 1. Requirement & Objective
- **PRD Reference**: `docs/prd.md` Section 3 (User Authentication & Access Control).
- **Business Goal**: Enable seamless, one-click enterprise authentication using Google Identity Services (OAuth 2.0 / OpenID Connect). Users can register or sign in instantly with their verified Google account without remembering passwords, while existing password-based users can link their Google identity if their email matches.
- **Detailed Description**:
  1. **Google OAuth 2.0 Handshake Flow**:
     - Endpoint `GET /api/v1/auth/google/login`:
       - Generates a cryptographically secure 32-byte pseudo-random `state` nonce (`crypto/rand`).
       - Writes an encrypted/HttpOnly, SameSite=Lax cookie `oauth_state` with a 10-minute TTL.
       - Returns HTTP 302 redirecting to `https://accounts.google.com/o/oauth2/v2/auth` with parameters:
         - `client_id`: Configured `GOOGLE_CLIENT_ID`
         - `redirect_uri`: Configured `GOOGLE_REDIRECT_URI` (e.g., `https://goodai.co.nz/api/v1/auth/google/callback`)
         - `response_type`: `code`
         - `scope`: `openid email profile`
         - `state`: The generated `state` nonce
         - `access_type`: `online`
         - `prompt`: `select_account`
     - Endpoint `GET /api/v1/auth/google/callback`:
       - Validates returned query param `state` against `oauth_state` cookie (strictly aborting with HTTP 400 on CSRF mismatch or expired cookie).
       - Clears `oauth_state` cookie.
       - Exchanges authorization `code` with Google's token endpoint (`POST https://oauth2.googleapis.com/token`) using TLS and `security.SafeDialer` or standard secure client.
       - Fetches user profile from `https://www.googleapis.com/oauth2/v3/userinfo` with the access token.
       - Verifies that `email_verified` is true and extracts `sub` (Google user ID), `email`, `name`, and `picture`.
  2. **Database & Account Linking**:
     - Upgrade SQLite schema in `internal/storage/user_store.go`:
       - Ensure `users` table contains `google_id TEXT UNIQUE` (nullable).
       - If a user with `google_id == sub` exists: issue session and log in.
       - Else if a user with `email == profile.email` exists: link account by setting `google_id = sub`, issue session, and log in.
       - Else: create a new user row with `email`, empty password hash (OAuth-only), `google_id = sub`, and `created_at = time.Now()`.
     - Issue a standard 30-day session token via `CreateSession`, write standard `session_token` cookie, and redirect (HTTP 302) browser to `/`.
  3. **Configuration & Graceful Degradation**:
     - Add `GoogleClientID`, `GoogleClientSecret`, `GoogleRedirectURI` to `internal/config/config.go`.
     - Read from environment variables `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`.
     - If credentials are not configured in environment, `GET /api/v1/auth/google/login` returns HTTP 503 or friendly error page, and frontend shows an informative disabled state ("Google SSO not configured").
  4. **Frontend UI Integration**:
     - Add official, high-polish "Continue with Google" buttons in both Sign In and Sign Up modals in `web/index.html`.
     - Implement standard Google branding (SVG G-logo, clean border, hover state).
     - Wire up click event to navigate to `/api/v1/auth/google/login`.
     - 100% English text across all buttons, tooltips, and error notifications.

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [ ] `internal/config/config.go`
- [ ] `internal/config/config_test.go`
- [ ] `internal/model/user.go`
- [ ] `internal/storage/user_store.go`
- [ ] `internal/storage/user_store_test.go`
- [ ] `internal/auth/oauth.go`
- [ ] `internal/auth/oauth_test.go`
- [ ] `internal/api/auth_handlers.go`
- [ ] `internal/api/auth_handlers_test.go`
- [ ] `internal/api/router.go`
- [ ] `internal/api/router_test.go`
- [ ] `web/index.html`
- [ ] `web/style.css`
- [ ] `web/app.js`
- [ ] `.env.example`
- [ ] `scripts/run.sh`
- [ ] `docs/ai-team/tasks/TASK-009-google-oauth-sso.md`
- [ ] `docs/ai-team/tasks/TASK-009-google-oauth-sso-qa-report.md`

---

## 3. Acceptance Criteria

- [ ] **AC-1: OAuth 2.0 State Nonce & CSRF Protection**:
  - `GET /api/v1/auth/google/login` generates a 256-bit crypto-random `state` parameter and writes it to an `HttpOnly`, `SameSite=Lax` cookie.
  - `GET /api/v1/auth/google/callback` validates that the query parameter `state` strictly matches the cookie.
  - Invalid, mismatched, or missing state returns HTTP 400 Bad Request with zero side-effects.

- [ ] **AC-2: Authorization Code Exchange & Userinfo Ingestion**:
  - Exchange endpoint calls `https://oauth2.googleapis.com/token` with `grant_type=authorization_code`, `code`, `client_id`, `client_secret`, and `redirect_uri`.
  - Ingests user profile from `https://www.googleapis.com/oauth2/v3/userinfo`.
  - Verifies `email_verified == true`. Rejects unverified accounts.

- [ ] **AC-3: Multi-Tenant Account Matching & Auto-Provisioning**:
  - If `google_id` already exists, retrieves existing `userID`.
  - If `google_id` is not found, but a user exists with matching verified email, links `google_id` to that account.
  - If neither exists, creates a new user with empty/disabled password and associates `google_id`.
  - Generates secure session and sets standard `session_token` cookie, redirecting to `/`.

- [ ] **AC-4: Graceful Degradation When Unconfigured**:
  - If `GOOGLE_CLIENT_ID` or `GOOGLE_CLIENT_SECRET` is unset, the system boots normally without crashing.
  - Endpoint `GET /api/v1/auth/google/login` returns HTTP 503 Service Unavailable with clear JSON error.
  - Frontend Google buttons display a helpful disabled tooltip or toast message when unconfigured.

- [ ] **AC-5: UI Visual Polish & Google Branding Guidelines**:
  - High-contrast, clean "Continue with Google" button added to both Sign In and Sign Up modals.
  - Proper SVG Google 4-color 'G' icon rendered.
  - Responsive on mobile, tablet, and desktop viewports.

- [ ] **AC-6: Zero Secret Leakage**:
  - Neither `GOOGLE_CLIENT_ID` nor `GOOGLE_CLIENT_SECRET` may be committed to git, logged to console, or returned in public API responses.
  - Documented in `.env.example`.

- [ ] **AC-7: 100% English Compliance**:
  - All UI elements, modal labels, error messages, logs, and comments must be strictly in English.

- [ ] **AC-8: Strict No-Build Rule Compliance**:
  - Zero local builds (`go build`, `npm run build`, `docker build`). All compilation via GitHub Actions CI.

---

## 4. Architecture & Security Design

```mermaid
sequenceDiagram
    autonumber
    actor User as User Browser
    participant App as GoodAI Monitor Frontend
    participant Srv as GoodAI Backend (/api/v1/auth)
    participant DB as SQLite (users & sessions)
    participant Google as Google Identity Server

    User->>App: Clicks "Continue with Google"
    App->>Srv: GET /api/v1/auth/google/login
    Srv->>Srv: Generate secure random state nonce
    Srv->>User: Set-Cookie: oauth_state=...; HttpOnly; SameSite=Lax<br/>302 Redirect: accounts.google.com/o/oauth2/v2/auth
    User->>Google: Authenticates with Google account
    Google->>User: 302 Redirect to callback URL with ?code=...&state=...
    User->>Srv: GET /api/v1/auth/google/callback?code=...&state=...
    Srv->>Srv: Verify state matches oauth_state cookie
    Srv->>Google: POST https://oauth2.googleapis.com/token (code, secret)
    Google-->>Srv: Return access_token & id_token
    Srv->>Google: GET https://www.googleapis.com/oauth2/v3/userinfo
    Google-->>Srv: Return {sub, email, email_verified, name}
    Srv->>DB: Find or create user by google_id / email
    DB-->>Srv: User record
    Srv->>DB: CreateSession(user.ID)
    Srv->>User: Set-Cookie: session_token=...; HttpOnly<br/>Clear-Cookie: oauth_state<br/>302 Redirect to /
    User->>App: Logged in, dashboard loads with user profile!
```

---

## 5. Developer Implementation Log
*(To be populated by Developer Agent during implementation)*

---

## 6. QA Audit & Verification Record
*(To be populated by QA / Security Agent during independent audit)*
