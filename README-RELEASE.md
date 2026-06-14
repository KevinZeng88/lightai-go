# LightAI Go Release Package

**Version**: 0.1.6 (RC1)
**Built with**: glibc 2.28 baseline (linux-build:el8-glibc2.28)

LightAI Go — lightweight GPU infrastructure management platform.
Server/Agent architecture with Web Console and built-in Prometheus + Grafana.

## Supported Systems

- Rocky Linux 8 / AlmaLinux 8 / RHEL 8 (glibc 2.28, linux-amd64)
- Ubuntu 20.04+ (linux-amd64)
- Other linux-amd64 with glibc >= 2.28

NOT supported: CentOS 7, Alpine/musl, ARM, glibc < 2.28.

## Build Requirements

Release binaries are built in a controlled glibc 2.28 container:
```bash
export LIGHTAI_BUILD_IMAGE=linux-build:el8-glibc2.28
./scripts/package-release-docker.sh --no-bump
```
Do NOT build release binaries on Ubuntu 24.04 or other glibc >= 2.29 hosts directly.

### Go Build Cache

The Docker wrapper persists Go module and build caches on the host:
- `.cache/go-mod` → container `/go/pkg/mod` (downloaded modules)
- `.cache/go-build` → container `/go-cache` (compiled packages)

First build downloads all Go modules (requires internet). Subsequent builds
reuse the cache and skip `go: downloading`. To force a clean module download:
```bash
rm -rf .cache/go-mod .cache/go-build
```

For fully offline builds, pre-populate the cache or use Go vendor mode (future).

## No Docker Required

This release includes Prometheus and Grafana binaries directly.
No Docker, docker compose, or container runtime needed.
All components run as native processes managed by shell scripts.

## Included Components

| Component | Version | Port |
|-----------|---------|------|
| LightAI Server + Web | 0.1.6 | 18080 |
| LightAI Agent | 0.1.6 | 19091 |
| Prometheus | 3.12.0 | 19090 |
| Grafana OSS | 13.0.2 | 13000 |

## Quick Start

### 1. Extract

```bash
tar -xzf lightai-go-0.1.4-linux-amd64.tar.gz
cd lightai-go-0.1.4-linux-amd64
```

### 2. Start All Services

```bash
# If you want to pre-set passwords (optional — auto-generated if not set):
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='YourSecurePassword123'
export LIGHTAI_GRAFANA_ADMIN_PASSWORD='YourGrafanaPassword123'

./scripts/start-server.sh
./scripts/start-agent.sh configs/agent.nvidia.yaml     # or: configs/agent.metax.yaml
./scripts/start-observability.sh
```

Initial credentials are saved to `runtime/initial-credentials.txt` (0600 permissions).
After first login, change passwords immediately.

### 3. Verify

```bash
./scripts/status.sh
./scripts/verify-local.sh
```

### 4. Access

| Service | URL |
|---------|-----|
| LightAI Web | http://<server-ip>:18080/ |
| Prometheus | http://<server-ip>:19090/ |
| Grafana | http://<server-ip>:13000/ (credentials in runtime/initial-credentials.txt) |

## Services

### Server
- Listens on 0.0.0.0:18080
- Web Console embedded (no separate web server needed)
- Default language: Chinese

### Agent
- Listens on 0.0.0.0:19091 (metrics)
- Supports MetaX (`mx-smi`) and NVIDIA (`nvidia-smi`) GPU collectors
- GPU collector scripts: `deploy/collectors/gpu/`

### Prometheus
- Local TSDB storage: `data/prometheus/`
- Retention: 15 days
- Scrapes Server (:18080/metrics) and Agent (:19091/metrics)

### Grafana
- SQLite database: `data/grafana/grafana.db`
- Auto-provisioned Prometheus datasource
- Auto-loaded dashboards from `deploy/observability/grafana/dashboards/`

## Directory Structure

```
├── bin/                    # All binaries
│   ├── lightai-server
│   ├── lightai-agent
│   ├── prometheus
│   └── grafana/
├── configs/                # Configuration files
│   ├── server.release.yaml
│   ├── agent.metax.yaml
│   ├── agent.nvidia.yaml
│   └── observability/
├── deploy/
│   ├── collectors/gpu/     # GPU collector scripts
│   └── observability/      # Grafana dashboards, provisioning
├── scripts/                # Management scripts
├── logs/ data/ run/        # Runtime directories
├── runtime/                # Credentials file (0600, auto-generated)
├── LICENSES/               # Third-party licenses
└── README-RELEASE.md
```

## Management Scripts

```bash
# Start / Stop
./scripts/start-server.sh [config]
./scripts/stop-server.sh
./scripts/start-agent.sh [config]
./scripts/stop-agent.sh
./scripts/start-observability.sh
./scripts/stop-observability.sh
./scripts/stop-all.sh

# Password Management
./scripts/reset-password.sh             # Reset Web/Admin + Grafana (auto-generate)
./scripts/reset-password.sh --password '<new>'   # Specify password
./scripts/reset-password.sh --interactive       # Prompt (no shell history)
./scripts/reset-password.sh --web-only          # Reset Web/Admin only
./scripts/reset-grafana-password.sh             # Reset Grafana only (auto-generate)

# Diagnostics
./scripts/status.sh           # Process + health check
./scripts/verify-local.sh     # Full verification
./scripts/collect-logs.sh     # Create diagnostics bundle
```

## GPU Collector

### MetaX
- Requires `mx-smi` at `/usr/bin/mx-smi` or set `MX_SMI` env var
- Agent user must be in `video` group for `/dev/mxcd`

### NVIDIA
- Requires `nvidia-smi`

## Logs

```
logs/lightai-server.log   # Server structured business log (JSON)
logs/lightai-agent.log    # Agent structured business log (JSON)
logs/server-stdout.log    # Server stdout/stderr wrapper
logs/agent-stdout.log     # Agent stdout/stderr wrapper
logs/prometheus.log       # Prometheus stdout
logs/grafana.log          # Grafana stdout
```

Log rotation: configurable via `logging.max_size_mb`, `logging.max_files`, `logging.retention_days`.

All logs in English.

## Credentials

- Initial passwords are auto-generated if not pre-set via environment variables.
- Credentials saved to `runtime/initial-credentials.txt` (0600, not overwritten on restart).
- To reset passwords: `./scripts/reset-password.sh` or `./scripts/reset-grafana-password.sh`.
- Reset credentials saved to `runtime/reset-credentials.txt` (0600).
- Passwords are never logged to stdout/stderr or log files.

## Security

- Set `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` for admin account (or use auto-generated)
- Set `LIGHTAI_GRAFANA_ADMIN_PASSWORD` for Grafana (or use auto-generated)
- Change default `agent_token` in configs before production
- Do NOT expose 18080/19090/13000 to public internet directly
- Use VPN, bastion host, or reverse proxy

## Troubleshooting

```bash
./scripts/status.sh                    # Check all services
./scripts/verify-local.sh              # Run health checks
./scripts/collect-logs.sh              # Collect diagnostics bundle
tail -100 logs/lightai-agent.log       # View agent structured log
tail -100 logs/lightai-server.log      # View server structured log
tail -100 logs/prometheus.log          # View Prometheus output
bash deploy/collectors/gpu/nvidia/discover.sh  # Test GPU collector
```

## Build from Source

```bash
./scripts/prepare-observability-binaries.sh --download
./scripts/package-release.sh
```
