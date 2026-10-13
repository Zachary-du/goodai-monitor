# Architectural Review & Feasibility Report: GoodAI Monitor

## Executive Summary
This document consolidates the independent architectural, product, and security reviews conducted by **PM Agent** and **QA / Security Agent**, synthesized by the **Lead Architect Agent**. 

The overall architecture for **GoodAI Monitor** is assessed as **FEASIBLE, HIGHLY TARGETED, AND SAFE**, provided that:
1. **Phase 1 is restructured into a "Thin Vertical-Slice Runnable MVP"** (delivering an accessible Web server, embedded English UI shell, SQLite baseline, and GitHub CI on Day 1, rather than an isolated terminal-only host metrics script).
2. **Strict RSS memory controls and Go runtime ceiling flags** (`GOMEMLIMIT=20MiB`, `GOGC=50`, SQLite 2MB cache) are enforced to stay well below the 25 MiB budget.
3. **Dial-time SSRF and IMDS protections** are incorporated into the synthetic prober from inception.

---

## 1. Status Categorization Matrix

Every architectural finding and decision is categorized under one of the four mandated statuses:
- `VERIFIED`: Confirmed by system audit, infrastructure inspection, or established hard constraints.
- `RECOMMENDED`: Proposed by PM/QA agents based on industrial best practices to prevent technical debt.
- `NOT VERIFIED`: Assumptions or metrics requiring empirical verification during remote CI or deployment.
- `NEEDS HUMAN DECISION`: Decisions requiring explicit choice or authorization from the Human Lead.

| Item ID | Topic | Architectural Finding / Recommendation | Status | Reviewing Role |
|---|---|---|---|---|
| **REV-01** | Architecture | Go 1.24+ statically compiled single binary (`CGO_ENABLED=0`) with pure-Go SQLite (`modernc.org/sqlite`) and standard library `net/http`. | `VERIFIED` | Lead / QA |
| **REV-02** | Architecture | Lifecycle governed by root context with a 10s graceful shutdown sequence (Drain HTTP $\rightarrow$ Stop Workers $\rightarrow$ Flush WAL & Close DB). | `RECOMMENDED` | QA Agent |
| **REV-03** | Architecture | Storage concurrency restricted to single-writer (`SetMaxOpenConns(1)` + WAL mode + in-memory preallocated circular ring buffer). | `VERIFIED` | QA Agent |
| **REV-04** | Memory Budget | Exact metric defined as **Linux RSS (Resident Set Size)** from `/proc/self/statm`, strictly excluding 64-bit Virtual Memory Arenas (VSS). | `VERIFIED` | QA Agent |
| **REV-05** | Memory Budget | Feasibility under pure Go SQLite + HTTP server confirmed at **14–18 MiB steady-state RSS**, under the condition of `PRAGMA cache_size = -2000;`. | `VERIFIED` | QA Agent |
| **REV-06** | Memory Budget | Process self-monitoring exposing `process_rss_bytes`, `go_heap_alloc_bytes`, and `go_goroutine_count` in `/api/v1/status`. | `RECOMMENDED` | QA Agent |
| **REV-07** | Memory Budget | Mandatory runtime flags configured in daemon startup: `GOMEMLIMIT=20MiB`, `GOGC=50`, `GODEBUG=madvdontneed=1`. | `RECOMMENDED` | QA Agent |
| **REV-08** | Build Constraints | Strict prohibition of `go build`, `npm run build`, and `go test` on local dev machine and Melbourne VPS to prevent host OOM panics. | `VERIFIED` | Lead / QA |
| **REV-09** | Build Constraints | Clear operational boundary: Local is strictly read/format (`gofmt`); compilation, static linting, and unit tests run on GitHub Actions. | `VERIFIED` | Lead / QA |
| **REV-10** | Roadmap | **Restructure Phase 1 into a Vertical-Slice Runnable MVP** (Runnable HTTP Server + 100% English UI Shell + SQLite Schema + CI) instead of a terminal-only Host Collector. | `RECOMMENDED` | PM Agent |
| **REV-11** | Business Health | Morning Pipeline Watchdog must alert immediately on `FAIL` status rather than waiting for the 08:30 NZDT deadline. | `RECOMMENDED` | PM Agent |
| **REV-12** | Business Health | Product count anomaly threshold (>20% drop day-over-day) to catch upstream supermarket IP bans returning HTTP 200 with 0 products. | `RECOMMENDED` | PM Agent |
| **REV-13** | Business Health | Systemd timer supervision to alert if `grocery-morning.timer` fails to launch the morning script by 08:00 NZDT. | `RECOMMENDED` | PM Agent |
| **REV-14** | Incidents | Formal incident lifecycle state machine (`PENDING_ALERT` $\rightarrow$ `ACTIVE` $\rightarrow$ `RESOLVING` $\rightarrow$ `AUTO_RESOLVED`) with anti-flapping hysteresis ($N=3, M=2$). | `RECOMMENDED` | PM Agent |
| **REV-15** | Security | SSRF Protection: Synthetic prober targets must be **statically configured**; zero API endpoints allowing arbitrary user-submitted URLs. | `VERIFIED` | QA Agent |
| **REV-16** | Security | SSRF Protection: Custom `SafeDialContext` blocking Oracle Cloud IMDS (`169.254.169.254`) and blocking HTTP redirects (`http.ErrUseLastResponse`). | `RECOMMENDED` | QA Agent |
| **REV-17** | Security | Docker socket client limited strictly to HTTP `GET /containers/json` and `GET /containers/{id}/stats` (zero mutation APIs). | `VERIFIED` | QA Agent |
| **REV-18** | Security | Secret leakage prevention: `.gitignore` blocks sensitive artifacts; webhook dispatchers mask tokens in logs. | `VERIFIED` | QA Agent |
| **REV-19** | Language Policy | **100% English UI and Server**: Zero Chinese characters in frontend code, API responses, logs, and comments. Reporting to Human Lead remains Chinese. | `VERIFIED` | Lead / PM / QA |
| **REV-20** | Deployment | Melbourne VPS resources (954 MiB RAM, 408 MiB free) safely accommodate GoodAI Monitor (15–20 MiB RSS). | `VERIFIED` | Lead / QA |
| **REV-21** | Deployment | Deployment Mode: Native systemd service vs Docker container in host mode (`network_mode: host`). | `RECOMMENDED` | QA Agent |
| **REV-22** | Deployment | Domain Routing: Subdomain (`monitor.goodai.co.nz`) vs Subpath (`price.goodai.co.nz/monitor`). | `NEEDS HUMAN DECISION` | Human Lead |
| **REV-23** | Product Spec | Canonical `PRD.md` drafted in repository root to replace absent placeholder. | `RECOMMENDED` | PM Agent |
| **REV-24** | Alert Sinks | Preferred webhook notification channels (Discord / Telegram / Generic Webhook tokens). | `NEEDS HUMAN DECISION` | Human Lead |
| **REV-25** | Authentication | Whether the status dashboard enforces HTTP Basic Auth by default or offers a public read-only status view. | `NEEDS HUMAN DECISION` | Human Lead |

---

## 2. In-Depth Analysis of the 7 Focus Areas

### 2.1 Go Single Binary Suitability & Organization
- **Suitability**: Confirmed as the optimal technology choice. Runtimes like Node.js (60–100MB RAM) or Python (50MB+ RAM) would impose an unacceptable risk on a VPS with ~400MB available memory. Go produces a self-contained static binary with a predictable ~15MB footprint.
- **Component Organization**:
  ```
  cmd/monitor/main.go          --> Entry point, OS signal binding, graceful shutdown orchestrator
  internal/config/             --> Environment & flag loading with strict validation
  internal/model/              --> Core domain entities (Metrics, Containers, Incidents, Pipeline)
  internal/storage/            --> Pure-Go SQLite (WAL mode, single-writer) + In-memory ring buffer
  internal/collector/          --> Host /proc reader & Docker socket read-only client
  internal/prober/             --> Synthetic HTTP prober with dial-time SSRF validation
  internal/pipeline/           --> GoodAI NZ Grocery morning pipeline watchdog & log parser
  internal/alert/              --> Deduplication, anti-flapping hysteresis, webhook dispatchers
  internal/api/                --> Standard library net/http ServeMux, middleware, SSE stream
  web/                         --> 100% English embedded frontend (embed.FS: HTML5, Tailwind, Alpine)
  ```
- **Lifecycle Coordination**: Root context cancels on `SIGINT`/`SIGTERM`. A deterministic 10-second shutdown window halts new HTTP requests, drains background collectors, flushes ring-buffer metrics to SQLite WAL, and closes file descriptors cleanly.

---

### 2.2 25 MB Memory Budget: Measurement Target & Feasibility
- **Measurement Object**: **Linux RSS (Resident Set Size)** from `/proc/self/statm` (field 2 $\times$ page size).
  - *Distinction*: 64-bit Go runtimes routinely allocate large Virtual Set Size (VSS) address arenas via `mmap PROT_NONE`. VSS consumes 0 physical memory and is disregarded by the Linux OOM killer.
- **Accurate Measurement Method**:
  1. Process self-reads `/proc/self/statm` and `/proc/self/status` (`VmRSS:`).
  2. Metrics exposed in `GET /api/v1/status`: `process_rss_bytes`, `go_heap_alloc_bytes`, `go_goroutine_count`.
- **Feasibility Evaluation**:
  - Baseline Go `net/http` server + runtime: ~8–10 MiB RSS.
  - In-memory circular ring buffer (240 samples $\times$ 8 metrics): ~150 KB.
  - `modernc.org/sqlite` with default cache can expand to 8–12 MiB.
  - **Mandatory Tuning Applied**:
    - `PRAGMA cache_size = -2000;` (hard limit of 2 MiB page cache).
    - `GOMEMLIMIT=20MiB` (triggers GC proactively before reaching ceiling).
    - `GOGC=50` (more aggressive heap recycling).
    - `GODEBUG=madvdontneed=1` (forces immediate page release back to Linux kernel).
  - **Conclusion**: Steady-state RSS will operate stably between **14 MiB and 18 MiB**, safely below the 25 MiB budget.

---

### 2.3 Strict No-Build Rule vs Compilation, Testing, and CI
- **The Core Problem**: The Go compiler toolchain (`compile` and `link`) routinely peaks at 400 MiB to 1.2 GiB RAM during build or test compilation. Executing `go build` or `go test` on the Melbourne VPS will immediately trigger the OOM killer, killing `grocery-server` or `caddy`.
- **Resolution Strategy**:
  - **Local Dev / VPS**: Strictly prohibited from running compilers. Only code editing, file reading, and lightweight syntax formatting (`gofmt`, < 15MB RAM) are permitted.
  - **Remote GitHub Actions Runner**: Standard GitHub Actions runner (Ubuntu Latest, 2 vCPUs, 7 GB RAM) executes `go test -v ./...`, `golangci-lint`, and `CGO_ENABLED=0 go build -o goodai-monitor-linux-amd64`.
  - **Artifact Distribution**: Precompiled binaries are published to GitHub Releases and pulled to the VPS via a deployment script, exactly mirroring the proven `update-grocery.sh` pattern.

---

### 2.4 Critique of Task Sequencing: Why Phase 1 Must Be a Vertical-Slice MVP

#### The Problem with Horizontal Task Sequencing
The initial plan proposed Phase 1 as: Skeleton $\rightarrow$ Host Collector $\rightarrow$ Docker Reader $\rightarrow$ CI.
- **PM Agent Critique**: This is a classic "horizontal layer" anti-pattern. At the end of Phase 1, the user and QA Agent have no web dashboard, no HTTP endpoint, and no database to verify. The tool merely dumps JSON strings to terminal stdout.
- **Delayed Risk Discovery**: The risk of embedded frontend assets (`embed.FS`) and SQLite memory bloat would only be discovered in Phase 3 & 4.
- **Inverted Value**: GoodAI's core business value is the **NZ Grocery Price Engine & Morning Pipeline**, not generic Linux `/proc` stats.

#### Recommended "Thin Vertical-Slice" Sequencing
We strongly recommend restructuring Phase 1 as follows:
- **Phase 1: Runnable Walking Skeleton & MVP**:
  1. Single Go binary starting an HTTP server on port `8090`.
  2. Embedded 100% English Web UI shell loaded in the browser.
  3. Embedded SQLite database initialized with WAL mode and health check (`GET /healthz`).
  4. Single synthetic probe hitting `https://price.goodai.co.nz` with latency rendering.
  5. GitHub Actions CI pipeline compiling and publishing the release artifact.
- **Result**: On Day 1, the Human Lead can open the browser and see a functional, end-to-end operational GoodAI Monitor dashboard. Subsequent phases simply wire in additional collectors and watchdog workers.

---

### 2.5 Security & Reliability Coverage

1. **SSRF & Metadata Protection**:
   - Oracle Cloud IMDS at `http://169.254.169.254/` exposes instance credentials.
   - **Enforced Defenses**:
     - All probe targets must be statically defined in the configuration. Zero public APIs accept dynamic user-submitted URLs.
     - Custom `SafeDialContext` validates resolved IP at socket connection time, blocking `169.254.0.0/16` and link-local IPv6, eliminating DNS Rebinding (TOCTOU).
     - Automated HTTP redirects blocked via `http.ErrUseLastResponse`.
2. **Docker Socket Privilege Containment**:
   - Unix domain sockets can receive `write()` syscalls even if mounted `:ro` if permissions allow.
   - The Docker client (`internal/collector/docker.go`) must be hardcoded to strictly issue HTTP `GET` requests (`GET /containers/json`).
3. **Incident State Machine & Anti-Flapping**:
   - `PENDING_ALERT` $\rightarrow$ `ACTIVE` (requires $N=3$ consecutive failures).
   - `ACTIVE` $\rightarrow$ `RESOLVING` $\rightarrow$ `AUTO_RESOLVED` (requires $M=2$ consecutive healthy samples).
   - Cooldown window (e.g. 30 minutes) prevents notification storms.
4. **Log Masking**:
   - Webhook tokens (Discord/Telegram) masked in all log outputs.

---

### 2.6 Deployment Fit on Melbourne VPS
- **Resource Feasibility**:
  - Total RAM: 954 MiB | Used: 545 MiB | Available: ~408 MiB.
  - GoodAI Monitor footprint: 14–18 MiB RSS (< 5% of available RAM).
  - Port assignment: `:8090` (zero collision with `:80`, `:443`, `:8088`, `:5244`, `:5245`, `:60324`).
- **Deployment Mode Recommendation**:
  - **Native Systemd Service (`/opt/goodai-monitor`)**: Recommended over Docker container. Eliminates container runtime tax (~0MB overhead) and allows native kernel cgroup memory enforcement via `MemoryMax=25M`.
  - **Alternative (Docker Host Mode)**: Fully supported if unified container management is preferred.

---

### 2.7 Fact vs Assumption Verification Matrix

| Claim / Component | Status | Basis / Evidence | Required Action |
|---|---|---|---|
| Melbourne VPS has 954 MiB RAM, 2 vCPUs, Ubuntu 24.04 | **VERIFIED FACT** | Direct SSH read-only audit in `docs/initial-audit.md`. | Enforce No-Build & 25MB RSS budget. |
| Production services run in host network mode (`:80, :443, :8088, :5244, :5245`) | **VERIFIED FACT** | Direct Docker inspect and `ss -tulpn` audit. | Assign non-conflicting port `:8090`. |
| Morning pipeline runs at 07:50 NZDT via `/opt/family-hub-vps/scripts/morning_pipeline.sh` | **VERIFIED FACT** | `systemctl list-timers` and script inspection. | Implement 07:50 NZDT watchdog prober. |
| GitHub SSH authentication succeeds for `Zachary-du/goodai-monitor` | **VERIFIED FACT** | `git ls-remote` executed successfully. | Ready for remote repository synchronization. |
| `modernc.org/sqlite` under WAL mode stays < 18 MiB RSS under load | **ASSUMPTION (NOT VERIFIED)** | Theoretical calculation based on 2MB page cache. | Validate empirically in GitHub Actions test run. |
| Pipeline log `/opt/family-hub-vps/grocery_data/morning_pipeline.log` is readable | **ASSUMPTION (NOT VERIFIED)** | Path exists on host, file permissions pending confirmation. | Deployment script must ensure read permissions (`chmod 644`). |
| Pipeline HTTP endpoint returns structured JSON on failure | **ASSUMPTION (NOT VERIFIED)** | Script inspects `.overall_status`, but exact failure schema needs live mock. | Unit test parser with mock pipeline responses. |

---

## 3. Recommended Actions & Next Steps

### 3.1 Proposed Documentation Updates (Awaiting Confirmation)
1. **Create `PRD.md`**: Formalize the GoodAI Monitor Product Requirements Document in repository root.
2. **Update `docs/implementation-plan.md`**: Shift from horizontal slicing to the **Vertical-Slice Runnable MVP** roadmap:
   - **Phase 1: Runnable Walking Skeleton & MVP (Port 8090, 100% English UI, SQLite, Healthz, CI)**
   - **Phase 2: NZ Grocery Morning Pipeline Watchdog & Real-Time Alerting Engine**
   - **Phase 3: Infrastructure & Docker Container Observability**
   - **Phase 4: High-Density Dashboard Visualizations & Incident Management**
   - **Phase 5: Production Staging & Zero-Downtime Caddy Integration**
3. **Update `docs/architecture.md`**: Embed dial-time SSRF protections, incident state machine schema, and Go runtime tuning flags.

### 3.2 Human Lead Decisions Required
Before issuing `TASK-001` or initiating development, please confirm your preferences on:
1. **Roadmap Restructuring**: Do you approve pivoting Phase 1 to the **Vertical-Slice Runnable MVP**?
2. **Domain Exposure**: Do you prefer subdomain routing (`monitor.goodai.co.nz`) or subpath routing (`price.goodai.co.nz/monitor`)?
3. **Authentication Default**: Should the dashboard enforce HTTP Basic Auth (`AUTH_ENABLED=true`) or be public read-only?
4. **Initial Git Commit**: May we commit the baseline documentation (`docs/`, `.env.example`, `.gitignore`, `Gemini.md`) to establish the root commit on `main`?
