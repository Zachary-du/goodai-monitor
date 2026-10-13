# Task Specification: TASK-006 Containerization & GHCR Remote Delivery

## Metadata
- **Task ID**: `TASK-006`
- **Title**: Containerization (Dockerfile, Docker Compose) & GHCR Remote Delivery
- **Current Status**: `DONE`
- **Assigned Role**: Developer Agent
- **Reviewer**: QA / Security Agent
- **Created Date**: 2026-10-14
- **Target Branch**: `task/TASK-006-containerization-and-deployment`
- **Dependencies**: `TASK-001`, `TASK-002`, `TASK-003`, `TASK-004`, `TASK-005`

---

## 1. Requirement & Objective
- **PRD / Architecture Reference**: `docs/prd.md` Deployment & Non-Functional Requirements; `docs/architecture.md` Section 6.
- **Business Goal**: Package GoodAI Monitor into an ultra-lightweight, hardened OCI container image published to GitHub Container Registry (GHCR), enabling zero-downtime containerized operation behind Caddy reverse proxy on Melbourne VPS.
- **Detailed Description**:
  - Implement multi-stage `Dockerfile`:
    - Build stage: Go 1.22 Alpine compiler with `-ldflags="-s -w"`.
    - Production runtime stage: Alpine 3.20 with `ca-certificates` and `tzdata` (Pacific/Auckland).
    - Hardened security: Non-root user `monitor` (UID 10001, GID 10001).
    - Healthcheck instruction probing `http://127.0.0.1:8090/healthz`.
    - Data volume mount point `/data` for SQLite database.
    - Image size target < 30 MiB.
  - Implement `.dockerignore` excluding `.git`, `*.key`, `*.token`, `.env`, and documentation.
  - Implement `docker-compose.yml`:
    - Service `goodai-monitor` binding strictly to `127.0.0.1:8090:8090`.
    - Volume mapping `./data:/data`.
    - Memory limit `32M` and CPU quota `0.2` to respect Melbourne VPS resource constraints.
    - Healthcheck and `restart: unless-stopped`.
  - Update `.github/workflows/ci-cd.yml`:
    - Add job to build and push container image to `ghcr.io/zachary-du/goodai-monitor:latest` and tagged with commit SHA.
    - Publish image securely using standard `GITHUB_TOKEN` permissions (`packages: write`).
  - Provide safe, non-destructive VPS deployment instructions for Caddy reverse proxy integration (`goodai.co.nz` -> `127.0.0.1:8090`).

---

## 2. Scope Whitelist (Allowed Files & Modules)
> [!IMPORTANT]
> The Developer Agent is strictly restricted to creating or editing the files listed below. Any modification outside this list will result in immediate QA rejection.

- [x] `Dockerfile`
- [x] `.dockerignore`
- [x] `docker-compose.yml`
- [x] `.github/workflows/ci-cd.yml`
- [x] `scripts/run.sh`
- [x] `docs/ai-team/tasks/TASK-006-containerization-and-deployment.md`
- [x] `docs/ai-team/tasks/TASK-006-containerization-and-deployment-qa-report.md`

---

## 3. Acceptance Criteria
- [x] **AC-1**: Multi-stage `Dockerfile` produces hardened, minimal image running as non-root user (UID 10001).
- [x] **AC-2**: Image includes `ca-certificates` for outbound TLS probing and `tzdata` for Auckland timezone support.
- [x] **AC-3**: Container healthcheck probes `http://127.0.0.1:8090/healthz` and reports healthy status.
- [x] **AC-4**: `docker-compose.yml` sets strict memory limit (32MB) and port isolation (`127.0.0.1:8090:8090`).
- [x] **AC-5**: `.dockerignore` strictly excludes secrets (`*.key`, `*.token`, `.env`), `.git`, and test fixtures.
- [x] **AC-6**: GitHub Actions workflow builds and pushes image to GHCR (`ghcr.io/zachary-du/goodai-monitor:latest`).
- [x] **AC-7**: Strict No-Build compliance: Zero local docker build executions; 100% remote CI compilation.
- [x] **AC-8**: 100% English compliance across all Docker configurations, compose definitions, and workflow steps.

---

## 4. Testing Requirements
- **Remote CI Verification**: GitHub Actions workflow successfully builds the Docker container and pushes to GHCR.
- **Security Audit**: Verify non-root execution, secret exclusions, and resource limit declarations.

---

## 5. Developer Implementation Log (Filled by Developer Agent)
*This section is filled when transitioning the task to `REVIEW`.*

- **Implementation Commit**: `9fe150a` on branch `task/TASK-006-containerization-and-deployment`
- **Files Modified**:
  - `Dockerfile`
  - `.dockerignore`
  - `docker-compose.yml`
  - `.github/workflows/ci-cd.yml`
  - `docs/ai-team/tasks/TASK-006-containerization-and-deployment.md`
  - `docs/ai-team/tasks/TASK-006-containerization-and-deployment-qa-report.md`
- **Automated Test Execution**: Verified remote CI workflow Run `38476133465` completed successfully across all 3 jobs (Lint & Unit Tests, Build Linux Static Binary, Build & Push Container Image). Dual distribution artifacts uploaded (`goodai-monitor-linux-amd64` and `goodai-monitor-docker-image`).
- **Known Limitations & Caveats**: Pulling private GHCR images on the VPS requires standard `docker login ghcr.io` or loading the downloaded image tarball directly (`docker load < goodai-monitor-docker.tar.gz`).
- **Potential Regression Risks**: None; containerization packages existing verified Go binary and assets without altering application runtime code.
- **Handoff Date**: 2026-10-14

---

## 6. QA Verification Summary (Filled by QA / Security Agent)
*This section is filled during `REVIEW` phase.*

- **Audit Report Reference**: `docs/ai-team/tasks/TASK-006-containerization-and-deployment-qa-report.md`
- **Verdict**: `APPROVED`
- **Audit Date**: 2026-10-14
- **Final Status**: `DONE` (Ready to deploy to Melbourne VPS container).
