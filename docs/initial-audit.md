# Initial Environment & VPS Infrastructure Audit

## Executive Summary
This document provides a comprehensive audit of the local development workspace and the remote production server (Oracle Cloud Melbourne VPS `158.179.29.27`). The audit covers system resources, running services, network topology, firewall rules, reverse proxy routing, and CI/CD constraints.

---

## 1. Local Workspace Audit
- **Project Directory**: `/Users/dylan/Downloads/goodai_monitor`
- **Git Repository**: Initialized on branch `main`
- **Remote Origin**: `git@github.com:Zachary-du/goodai-monitor.git`
- **GitHub SSH Connectivity**: Successfully authenticated as `Zachary-du` with remote read/write access.
- **Ignored Files**: Sensitive keys (`*.key`, `key/`), API tokens (`*.token`), and local environment files (`.env`) are strictly excluded in `.gitignore`.

---

## 2. Oracle Cloud Melbourne VPS Audit (`158.179.29.27`)

### 2.1 Operating System & Hardware Resources
- **OS**: Ubuntu 24.04.5 LTS (Noble Numbat)
- **Kernel**: `6.17.0-1020-oracle #20-Ubuntu SMP x86_64`
- **CPU**: 2 vCPUs (AMD EPYC 7551 32-Core Processor)
- **Memory**:
  - Total RAM: **954 MiB** (~1.0 GB)
  - Used: ~545 MiB
  - Free: ~64 MiB
  - Available: ~408 MiB
  - Swap: 2.0 GiB total (195 MiB currently utilized)
  - **Critical Constraint**: With less than 1GB RAM, memory is severely constrained. Compiling, bundling, or running heavy runtimes (e.g., standard Node/Java/Rust builds) on the VPS or local low-spec environment will cause Out-Of-Memory (OOM) kernel panics and kill live production services.
- **Disk Storage**:
  - Partition: `/dev/sda1` mounted on `/`
  - Total Size: 96 GB
  - Used: 14 GB (15% utilization)
  - Available: 82 GB

### 2.2 Container Runtime & Active Containers
- **Container Engine**: Docker version `29.8.1`, build `4a63305`
- **Compose Project Path**: `/opt/family-hub-vps/docker-compose.yml`
- **Active Docker Containers**:
  | Container Name | Image | Command | Port / Network Mode | Purpose |
  |---|---|---|---|---|
  | `caddy` | `caddy:alpine` | `caddy run --config ...` | host mode (`:80`, `:443` TCP/UDP) | Reverse proxy, TLS termination & path-based routing |
  | `grocery-server` | `family-hub-vps-grocery` | `/app/grocery-server` | host mode (`:8088`) | GoodAI NZ grocery price comparison backend & API |
  | `alist-bridge` | `family-hub-vps-bridge` | `python3 -u service.py` | host mode (`:5245`) | Storage sync bridge daemon |
  | `alist` | `xhofe/alist:latest` | `/entrypoint.sh` | host mode (`:5244`) | AList cloud drive management server |

### 2.3 Network Ports & Process Listening Table
| Protocol | Port | Process / Service | Scope / Binding |
|---|---|---|---|
| TCP | `60324` | `sshd` | Non-standard SSH port (OpenBSD Secure Shell) |
| TCP / UDP | `80`, `443` | `caddy` | Public Web & TLS (HTTP/1.1, HTTP/2, HTTP/3 QUIC) |
| TCP | `8088` | `grocery-server` | Internal backend service (`0.0.0.0:8088`) |
| TCP | `5244` | `alist` | Internal/External drive manager (`0.0.0.0:5244`) |
| TCP | `5245` | `python3` (alist-bridge) | Storage bridge service (`0.0.0.0:5245`) |
| TCP | `2019` | `caddy` | Caddy administrative API (`127.0.0.1:2019`) |

### 2.4 Firewall Configuration (`iptables` / Oracle Cloud Security List)
- **Active Inbound Rules**:
  - `tcp/60324` (SSH) - ACCEPT
  - `tcp/80` (HTTP) - ACCEPT
  - `tcp/443` & `udp/443` (HTTPS & HTTP/3) - ACCEPT
  - `tcp/5244` (AList) - ACCEPT
  - `tcp/5245` (Bridge) - ACCEPT
  - `tcp/22` (Legacy SSH port) - ACCEPT

### 2.5 Reverse Proxy & TLS Configuration (`Caddyfile`)
- **Config File**: `/opt/family-hub-vps/Caddyfile`
- **Global Settings**: Email registered for Let's Encrypt / ZeroSSL notifications.
- **Routing Rules**:
  - Domain: `price.goodai.co.nz`
  - Upstream: `127.0.0.1:8088` (`reverse_proxy 127.0.0.1:8088`)
  - Security Rules:
    - Custom security block `@sync_block` returning `403 Forbidden` for public requests matching `/api/grocery/sync*` and `/api/grocery/pns-deals/sync*` to protect Pak'nSave upstream from rate-limiting.

### 2.6 Automation Jobs & Scheduled Timers
- **Systemd Timer**: `grocery-morning.timer` -> triggers `grocery-morning.service`
- **Schedule**: Daily at `07:50:00 NZDT` (Pacific/Auckland timezone)
- **Script Executed**: `/opt/family-hub-vps/scripts/morning_pipeline.sh`
- **Workflow**:
  1. Verifies `grocery-server` container health.
  2. Probes `http://127.0.0.1:8088/api/grocery/status`.
  3. Checks pipeline status via `/api/grocery/fresh/pipeline/status`.
  4. Triggers pipeline run if not yet completed for today (`POST /api/grocery/fresh/pipeline/run`).
  5. Monitors execution loop until completion or 1200-second timeout.
  6. Evaluates raw products, canonical products, canonical members, produce audit, and meat audit metrics.

### 2.7 Deployment Pattern
- Existing deployment uses `update-grocery.sh`:
  - Pulls precompiled release binary (`grocery-server-linux-amd64`) and static assets (`web.tar.gz`) from GitHub Releases using a GitHub API token.
  - Builds an ultra-minimal Alpine runtime container without compiling code on the server.
  - Restarts services via `docker compose up -d`.

---

## 3. Key Findings & Constraints
1. **Strict No-Build Rule**: Neither local machine nor the VPS must ever execute build/compilation tasks. All builds must run inside GitHub Actions CI/CD.
2. **Resource Budget**: `goodai-monitor` must be engineered to consume < 25 MiB of RAM and < 1% idle CPU.
3. **Language Policy**: **Zero Chinese characters allowed anywhere in the frontend, UI, or server logs**. The entire user-facing interface, documentation for the software itself, and code comments must be strictly in English. User communication and status reports remain in Chinese.
4. **Zero Downtime Constraint**: The monitoring service must run alongside existing services (`grocery-server`, `alist`, `caddy`) without interfering with their ports, storage volumes, or Caddy certificates.
