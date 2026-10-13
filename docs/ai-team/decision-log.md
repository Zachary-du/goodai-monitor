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
