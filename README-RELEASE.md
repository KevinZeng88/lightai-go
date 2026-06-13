# LightAI Go Release Package

LightAI Go — lightweight GPU infrastructure management platform.
Server/Agent architecture with Web Console and built-in Prometheus + Grafana.

## Supported Systems

- Rocky Linux 8 (linux-amd64)
- CentOS 8 (linux-amd64)
- Ubuntu 20.04 (linux-amd64)
- Other linux-amd64 with glibc

NOT supported: CentOS 7, Alpine/musl, ARM.

## No Docker Required

This release includes Prometheus and Grafana binaries directly.
No Docker, docker compose, or container runtime needed.
All components run as native processes managed by shell scripts.

## Included Components

| Component | Version | Port |
|-----------|---------|------|
| LightAI Server + Web | 0.1.0 | 18080 |
| LightAI Agent | 0.1.0 | 19091 |
| Prometheus | 3.12.0 | 19090 |
| Grafana OSS | 13.0.2 | 13000 |

## Quick Start

### 1. Extract

```bash
tar -xzf lightai-go-0.1.0-linux-amd64.tar.gz
cd lightai-go-0.1.0-linux-amd64
```

### 2. Set Passwords

```bash
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='YourSecurePassword123'
export LIGHTAI_GRAFANA_ADMIN_PASSWORD='YourGrafanaPassword123'
```

### 3. Start All Services

```bash
./scripts/start-server.sh
./scripts/start-agent.sh nvidia     # or: metax
./scripts/start-observability.sh
```

### 4. Verify

```bash
./scripts/status.sh
./scripts/verify-local.sh
```

### 5. Access

| Service | URL |
|---------|-----|
| LightAI Web | http://<server-ip>:18080/ |
| Prometheus | http://<server-ip>:19090/ |
| Grafana | http://<server-ip>:13000/ (admin / <LIGHTAI_GRAFANA_ADMIN_PASSWORD>) |

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
├── LICENSES/               # Third-party licenses
└── README-RELEASE.md
```

## Management Scripts

```bash
# Start / Stop
./scripts/start-server.sh
./scripts/stop-server.sh
./scripts/start-agent.sh [metax|nvidia]
./scripts/stop-agent.sh
./scripts/start-observability.sh
./scripts/stop-observability.sh

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
logs/server.log           # Server structured log
logs/agent.log            # Agent structured log
logs/server-stdout.log    # Server stdout
logs/agent-stdout.log     # Agent stdout
logs/prometheus.log       # Prometheus stdout
logs/grafana.log          # Grafana stdout
```

All logs in English.

## Security

- Set `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` for admin account
- Set `LIGHTAI_GRAFANA_ADMIN_PASSWORD` for Grafana
- Change default `agent_token` in configs before production
- Do NOT expose 18080/19090/13000 to public internet directly
- Use VPN, bastion host, or reverse proxy

## Troubleshooting

```bash
./scripts/status.sh                    # Check all services
./scripts/verify-local.sh              # Run health checks
./scripts/collect-logs.sh              # Collect diagnostics bundle
tail -100 logs/agent-stdout.log        # View agent output
tail -100 logs/prometheus.log          # View Prometheus output
bash deploy/collectors/gpu/metax/discover.sh  # Test GPU collector
```

## Build from Source

```bash
./scripts/prepare-observability-binaries.sh --download
./scripts/package-release.sh
```
