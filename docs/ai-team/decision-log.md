# AI Team Architectural & Product Decision Log (ADR)

## Purpose
This document logs all critical architectural, engineering, and product decisions made during the GoodAI Monitor project lifecycle. Any deviation from original requirements, resolution of technical conflicts, or introduction of new dependencies must be logged here and approved by the Human Lead.

---

## Decision Records

### ADR-001: Go Language & Single Binary Architecture for Melbourne VPS
- **Date**: 2026-10-13
- **Status**: ACCEPTED
- **Deciders**: Human Lead, AI Engineering Team
- **Context**:
  The target production server is an Oracle Cloud Melbourne VPS with only 954 MiB (~1GB) RAM running existing services (`caddy`, `grocery-server`, `alist`, `alist-bridge`) that consume ~550 MiB.
- **Decision**:
  Build GoodAI Monitor in Go (Golang) as a single statically compiled binary (`CGO_ENABLED=0`). Embed frontend assets directly using Go `embed.FS`. Prohibit runtime interpreters like Node.js or heavy Python stacks for the monitoring core.
- **Consequences**:
  - Positive: Memory footprint is constrained to 12–20 MiB RSS; zero external runtime dependencies on host.
  - Constraint: Requires GitHub Actions cross-compilation pipeline since local/host builds are strictly forbidden.

### ADR-002: Strict No-Build Constraint & Remote CI/CD Delivery
- **Date**: 2026-10-13
- **Status**: ACCEPTED
- **Deciders**: Human Lead
- **Context**:
  Local dev environment and VPS have minimal resources (1 vCPU / <1GB RAM). Running compilers (`go build`, `npm run build`, `docker build`) leads to CPU saturation, OOM crashes, and host service disruption.
- **Decision**:
  Enforce a strict zero-build policy on local and production machines. All compilation and packaging must execute remotely in GitHub Actions (`.github/workflows/ci-cd.yml`). Distribution to the VPS is handled via precompiled release assets from GitHub Releases.
- **Consequences**:
  - Positive: Complete safety against OOM crashes; reproducible automated builds.
  - Constraint: Development iteration relies on pushing to feature branches and inspecting CI status.

### ADR-003: 100% English Language Policy for Software & UI
- **Date**: 2026-10-13
- **Status**: ACCEPTED
- **Deciders**: Human Lead
- **Context**:
  Internationalization and clean professional UI presentation for GoodAI platform.
- **Decision**:
  The entire software product—including web dashboard, REST API responses, error logs, configuration keys, and code comments—must be strictly in English (zero Chinese characters). Human-to-agent communications and project reporting to the Human Lead remain in Chinese.
- **Consequences**:
  - Positive: Professional appearance, international consistency.
  - Requirement: QA / Security Agent must enforce zero Chinese characters in all code and UI audits.

### ADR-004: Multi-Agent Collaboration Framework & Worktree Isolation
- **Date**: 2026-10-13
- **Status**: ACCEPTED
- **Deciders**: Human Lead, AI Engineering Team
- **Context**:
  Establishing structured, error-resilient collaboration between PM Agent, Developer Agent, and QA/Security Agent without overlapping changes, hallucinations, or unauthorized code merges.
- **Decision**:
  Enforce strict role separation: PM specifies tasks (`BACKLOG`), Human Lead approves (`READY`), Developer implements in isolated Git branches/worktrees (`IN_PROGRESS`), QA independently verifies with zero-trust audit (`REVIEW`), Human Lead approves final merge (`DONE`).
- **Consequences**:
  - Positive: Complete traceability, zero merge conflicts, rigorous security gatekeeping.

### ADR-005: Anti-Flapping Incident State Machine (N=2, M=2) & SMTP Alerting
- **Date**: 2026-10-14
- **Status**: ACCEPTED
- **Deciders**: Human Lead, AI Engineering Team
- **Context**:
  Transient network glitches or single dropped packets can trigger false-alarm outage emails and rapid toggling (flapping) between UP and DOWN states, eroding operator trust.
- **Decision**:
  1. Enforce strict anti-flapping thresholds: $N=2$ consecutive check failures required to confirm an outage and transition to `DOWN` with an `OPEN` incident; $M=2$ consecutive check successes required to confirm recovery and transition back to `UP` with a `RESOLVED` incident.
  2. Implement outbound email alerts using standard Go `net/smtp` with STARTTLS support and dual-timezone timestamp formatting (NZDT & UTC).
  3. Enforce strict deduplication: at most 1 outage alert email per incident; no repeated emails during an ongoing outage.
  4. Ensure graceful degradation: unconfigured or failed SMTP connections log warnings with masked credentials and never interrupt probe execution or crash background workers.
- **Consequences**:
  - Positive: Zero false-alarm flapping alerts; clear downtime duration records; robust against transient glitches.
  - Requirement: Incident state is persisted in SQLite with monitor and tenant foreign keys, supporting restart recovery.

### ADR-006: Hardened Containerization & Dual Artifact Distribution Strategy
- **Date**: 2026-10-14
- **Status**: ACCEPTED
- **Deciders**: Human Lead, AI Engineering Team
- **Context**:
  Operator requested containerized deployment for GoodAI Monitor serving the root apex domain `goodai.co.nz` on the Melbourne VPS, while strictly preserving resource constraints and zero local build limits.
- **Decision**:
  1. Package application using a hardened multi-stage `Dockerfile`:
     - Stage 1 compiles the Go binary using Alpine Go 1.22 with `go mod tidy` and `-ldflags="-s -w"`.
     - Stage 2 runs Alpine 3.20 as dedicated unprivileged non-root user `monitor` (UID 10001, GID 10001).
     - Persistent database mounted at `/data/monitor.db`.
  2. Implement dual distribution in remote GitHub Actions CI:
     - Push multi-arch OCI images to GitHub Container Registry (`ghcr.io/zachary-du/goodai-monitor:latest` and tagged with commit SHA).
     - Save and upload Docker image tarball (`goodai-monitor-docker.tar.gz`) as a CI artifact, allowing direct loading via `docker load` without requiring registry credentials on the VPS.
  3. Enforce strict resource quotas in `docker-compose.yml`:
     - Hard memory limit: `32M`.
     - CPU quota: `0.2`.
     - Port binding: `127.0.0.1:8090:8090` (loopback only, exposed publicly exclusively via host Caddy reverse proxy with automatic Let's Encrypt TLS).
- **Consequences**:
  - Positive: Clean container isolation, non-root execution security, zero port conflicts with existing VPS services.
  - Verification: GitHub Actions CI Run `38476133465` successfully verified compilation, testing, and container packaging.
