# Implementation Plan: GoodAI Monitor

## Overview & Development Methodology
This implementation plan establishes a phased, industrial-grade development roadmap for **GoodAI Monitor**. Development strictly follows the execution guidelines:
- **No Local Compilation**: Local environment is strictly restricted to source editing, unit test definition, and static linting.
- **Remote CI/CD Automation**: All cross-compilation and release artifacts are built on GitHub Actions and distributed via GitHub Releases.
- **Language Policy**: 100% English for code, UI, comments, and server responses. Progress reporting and client communications are conducted in Chinese.
- **Safety First**: Zero-downtime, non-destructive integration with the live production VPS (`158.179.29.27`).

---

## Phase Breakdown & Milestones

### Phase 1: Project Skeleton, Host Metrics Engine & CI/CD Pipeline
- **Deliverables**:
  1. Go module structure (`go.mod`, package layout: `internal/collector`, `internal/config`, `internal/model`).
  2. Host metrics collector reading `/proc/stat`, `/proc/meminfo`, `/proc/loadavg`, and filesystem stats without third-party CGO dependencies.
  3. Docker socket client (`internal/docker`) querying container statuses via `/var/run/docker.sock` HTTP API.
  4. GitHub Actions workflow (`.github/workflows/ci-cd.yml`):
     - Automatic linting (`golangci-lint`).
     - Unit test execution (`go test -v ./...`).
     - Cross-compilation for `linux/amd64`.
     - Automatic creation of GitHub Releases with `goodai-monitor-linux-amd64` binary.
  5. Deployment script `scripts/deploy.sh` on the VPS pulling release artifacts safely without compiling on the host.
- **Verification Criteria**:
  - [ ] GitHub Actions CI runs successfully on remote runner.
  - [ ] Release binary is published to GitHub Releases.
  - [ ] Binary runs on Ubuntu 24.04 x86_64 and outputs structured JSON host metrics.
  - [ ] Docker container probe accurately discovers `caddy`, `grocery-server`, `alist`, and `alist-bridge`.
  - [ ] Local RAM consumption during daemon run is verified < 20 MiB.

---

### Phase 2: Synthetic HTTP Prober & Morning Pipeline Watchdog
- **Deliverables**:
  1. HTTP Synthetic Prober (`internal/prober`):
     - Periodic non-blocking goroutines probing configured endpoints.
     - Measures latency (DNS, TLS handshake, TTFB, total duration).
     - Validates HTTP status codes and TLS certificate expiration days.
  2. GoodAI Morning Pipeline Watchdog (`internal/pipeline`):
     - Probes `http://127.0.0.1:8088/api/grocery/fresh/pipeline/status`.
     - Extracts daily status (`PASS`/`FAIL`/`RUNNING`), counts (`raw_products`, `canonical_products`, `canonical_members`), and audit states (`produce_report`, `meat_report`).
     - Reads and parses the tail of `/opt/family-hub-vps/grocery_data/morning_pipeline.log`.
     - Evaluates 08:30 NZDT deadline watchdog logic.
- **Verification Criteria**:
  - [ ] Synthetic probe correctly detects healthy endpoints and simulated 4xx/5xx errors.
  - [ ] TLS certificate remaining validity calculated accurately for `price.goodai.co.nz`.
  - [ ] Pipeline watchdog correctly reflects live pipeline status and data metrics.

---

### Phase 3: Lightweight Embedded Storage & REST API
- **Deliverables**:
  1. Storage Engine (`internal/storage`):
     - Pure Go SQLite (`modernc.org/sqlite`) with WAL mode.
     - In-memory circular ring buffer for sub-minute high-resolution metrics.
     - Rolling 14-day retention pruning job to prevent unbounded disk growth.
     - Incident table tracking ongoing and resolved anomalies.
  2. REST API Engine (`internal/api`):
     - `GET /api/v1/status`: Comprehensive live snapshot.
     - `GET /api/v1/metrics/history`: Timeseries for CPU, RAM, disk, and response latency.
     - `GET /api/v1/incidents`: Active and resolved incidents.
     - `GET /api/v1/pipeline`: Morning Pipeline execution metrics and latest log tail.
     - `GET /healthz`: Liveness check for the monitor itself.
     - Optional HTTP Basic Auth or Bearer token middleware.
- **Verification Criteria**:
  - [ ] SQLite database created and initialized without CGO dependencies.
  - [ ] Metrics saved and historical queries return within < 10ms.
  - [ ] Automated retention job purges records older than configured retention period.
  - [ ] All API responses follow standard JSON format with 100% English field names and messages.

---

### Phase 4: Embedded English Web Dashboard
- **Deliverables**:
  1. Single-page status dashboard embedded via Go `embed.FS` (`web/index.html`, `web/app.js`, `web/style.css`):
     - **100% English UI** (Strict compliance: zero Chinese characters).
     - Overall System Status Header (`All Systems Operational` / `Degraded` / `System Outage`).
     - Real-Time Infrastructure Grid: CPU gauge, Memory progress bar with swap breakdown, Disk usage, Load average.
     - Container Ecosystem Matrix: Status badges, restart counters, memory per container.
     - NZ Grocery Morning Pipeline Card: Today's date, overall status badge, product counts, Phase 2 Produce & Meat audit flags, and recent log viewer.
     - Endpoint Uptime & Latency Monitor: Live ping graph, TLS cert days remaining countdown.
     - Incidents Timeline: Detailed event records with timestamps in NZDT and UTC.
  2. Auto-refresh: Polling or Server-Sent Events (SSE) every 10 seconds.
- **Verification Criteria**:
  - [ ] Dashboard loads in browser directly from the single binary with zero external asset dependencies.
  - [ ] Zero Chinese text anywhere in UI, tooltips, or error toasts.
  - [ ] Responsive design verified on desktop and mobile viewports.
  - [ ] Visual indicators instantly reflect service state changes.

---

### Phase 5: Alerting & Webhook Notification Engine
- **Deliverables**:
  1. Alert Manager (`internal/alert`):
     - Configurable alert rules (high memory > 85%, container down, probe failure, pipeline failure/delay).
     - Anti-flapping logic: requires $N$ consecutive failed checks before firing.
     - Alert deduplication and cooldown timers (e.g., maximum 1 notification per 30 minutes for identical incidents).
     - Auto-resolved notifications when normal state is restored.
  2. Dispatchers:
     - Generic Webhook dispatcher (JSON payload).
     - Discord Webhook integration with formatted embed cards.
     - Telegram Bot integration with markdown messages.
- **Verification Criteria**:
  - [ ] Webhook triggers upon simulated container failure or simulated memory threshold breach.
  - [ ] Cooldown prevents duplicate notifications during continued outage.
  - [ ] Recovery notification correctly sent when service status returns to green.

---

### Phase 6: Production Staging, Caddy Integration & Final Acceptance
- **Deliverables**:
  1. Production packaging: Minimal Docker container (`alpine:3.20` or `scratch`) or systemd service unit (`goodai-monitor.service`).
  2. Caddy reverse proxy route configuration (`monitor.goodai.co.nz` or `/monitor` subpath on `price.goodai.co.nz`).
  3. Integration into `/opt/family-hub-vps/docker-compose.yml`.
  4. End-to-end acceptance run and verification report (`PASS`, `FAIL`, `NOT VERIFIED`).
- **Verification Criteria**:
  - [ ] Monitor deployed to Melbourne VPS via CI/CD release artifact.
  - [ ] Host memory overhead remains < 25 MiB.
  - [ ] Existing services (`grocery-server`, `alist`, `caddy`) remain 100% unaffected.
  - [ ] Status dashboard accessible via HTTPS.
  - [ ] Morning pipeline execution verified during scheduled 07:50 NZDT run.

---

## Phase 1 Detailed Task List & Acceptance Criteria

| Task ID | Task Description | Verification Method | Acceptance Standard |
|---|---|---|---|
| **P1-T1** | Initialize Go project structure, dependencies (`go.mod`), and configuration model | `go vet ./...` (local syntax check) | Clean module with zero CGO dependencies |
| **P1-T2** | Implement Host Collector (`internal/collector/host.go`) reading `/proc` stats | Unit test mock | Returns valid CPU, RAM, Disk, Load values |
| **P1-T3** | Implement Docker Socket Reader (`internal/collector/docker.go`) | Unit test mock | Returns running containers and status |
| **P1-T4** | Implement GitHub Actions workflow (`.github/workflows/ci-cd.yml`) | Push to GitHub & verify GHA execution | CI passes and builds `linux/amd64` release asset |
| **P1-T5** | Create remote release downloader script (`scripts/deploy-monitor.sh`) | Dry-run script against GitHub Releases API | Pulls prebuilt binary without local compilation |
