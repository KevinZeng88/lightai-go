# LightAI Go Release Package

Version: 0.1.0

LightAI Go is a lightweight GPU infrastructure management platform.
Server/Agent architecture with Web Console.

## Quick Start

### 1. Extract

```bash
tar -xzf lightai-go-0.1.0-linux-amd64.tar.gz
cd lightai-go-0.1.0-linux-amd64
```

### 2. Set Admin Password

```bash
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='YourSecurePassword123'
```

The password is required on first start to create the admin user.
If not set, a random password is generated and printed once to stderr.

### 3. Start Server

```bash
./scripts/start-server.sh
```

Server listens on `0.0.0.0:8080` by default.

- Web Console: `http://<server-ip>:8080/`
- API: `http://<server-ip>:8080/api/`
- Health: `http://<server-ip>:8080/healthz`
- Metrics: `http://<server-ip>:8080/metrics`

### 4. Start Agent (MetaX)

```bash
./scripts/start-agent.sh metax
```

For NVIDIA GPUs:

```bash
./scripts/start-agent.sh nvidia
```

### 5. Verify

```bash
./scripts/status.sh
./scripts/verify-local.sh
```

### 6. Open Web Console

```
http://<server-ip>:8080/
```

Default language: Chinese. Switch to English via top-right language selector.

## Directory Structure

```
lightai-go-0.1.0-linux-amd64/
├── bin/
│   ├── lightai-server      # Server binary (API + embedded Web)
│   └── lightai-agent       # Agent binary
├── configs/
│   ├── server.release.yaml # Server configuration
│   ├── agent.metax.yaml    # Agent config (MetaX GPU)
│   └── agent.nvidia.yaml   # Agent config (NVIDIA GPU)
├── deploy/
│   ├── collectors/gpu/     # GPU collector scripts
│   │   ├── common.sh
│   │   ├── nvidia/
│   │   └── metax/
│   └── observability/      # Prometheus/Grafana configs
├── scripts/
│   ├── start-server.sh     # Start server (background)
│   ├── start-agent.sh      # Start agent (background)
│   ├── stop-server.sh      # Stop server
│   ├── stop-agent.sh       # Stop agent
│   ├── status.sh           # Show running status
│   ├── verify-local.sh     # Run health checks
│   ├── collect-logs.sh     # Collect diagnostics bundle
│   ├── observability-up.sh # Start Prometheus+Grafana
│   ├── observability-down.sh
│   └── observability-status.sh
├── logs/                   # Log files (created on start)
├── data/                   # Database and agent state
├── run/                    # PID files
├── VERSION                 # Build info
└── README-RELEASE.md       # This file
```

## Logs

- `logs/server.log` — Server structured log
- `logs/agent.log` — Agent structured log
- `logs/server-stdout.log` — Server stdout/stderr
- `logs/agent-stdout.log` — Agent stdout/stderr

All logs are in English.

## GPU Collector

GPU metrics are collected via external scripts that output
LightAI GPU Collector Protocol.

- NVIDIA: `nvidia-smi` must be available.
- MetaX: `mx-smi` must be available at `/usr/bin/mx-smi`
  or set via `MX_SMI` environment variable.

MetaX agent user must be in the `video` group for `/dev/mxcd` access.

## Observability (Optional)

Bundled Prometheus + Grafana:

```bash
./scripts/observability-up.sh    # Start (requires prometheus + grafana-server)
./scripts/observability-status.sh
./scripts/observability-down.sh
```

Prometheus: `http://<server-ip>:19090`
Grafana:    `http://<server-ip>:13000` (admin/lightai)

Set `LIGHTAI_GRAFANA_ADMIN_PASSWORD` for production.

## Troubleshooting

```bash
# Check status
./scripts/status.sh

# Run verification
./scripts/verify-local.sh

# Collect diagnostics
./scripts/collect-logs.sh
# Sends lightai-go-logs-<timestamp>.tar.gz

# View recent logs
tail -100 logs/agent-stdout.log
tail -100 logs/server-stdout.log

# Check GPU collector
bash deploy/collectors/gpu/metax/discover.sh
bash deploy/collectors/gpu/metax/metrics.sh

# Reset database (WARNING: deletes all data)
rm -f data/lightai.db
# Restart server to re-initialize
```

## Web Console

The Web Console is embedded in the Server binary.
No separate web server or Node.js required on the target machine.

Default language: Chinese. Click the language selector (top-right) to switch to English.

Pages:
- Dashboard — node/GPU overview
- Nodes — node list with detail drawer
- GPUs — GPU list with filter/search/detail
- Observability — Metrics Targets, Prometheus, Grafana

## Security Notes

- Change `agent_token` in configs before production use.
- Set `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` via environment variable.
- Set `LIGHTAI_GRAFANA_ADMIN_PASSWORD` for Grafana.
- Do NOT expose ports 8080/19090/13000 directly to public internet.
- Use VPN, bastion host, or reverse proxy for production access.
- TLS/HTTPS is a future enhancement.

## Build from Source

```bash
cd web && npm install && npm run build
cd ..
go build -tags web -o bin/lightai-server ./cmd/server
go build -o bin/lightai-agent ./cmd/agent
./scripts/package-release.sh
```
