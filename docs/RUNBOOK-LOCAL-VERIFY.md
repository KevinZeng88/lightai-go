> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference or historical compatibility document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go Local Verification Runbook

> Last updated: 2026-06-14
> Applicable: Phase 0 through RC1 Hotfix

## Quick Reference — Ports

| Service | Default address |
|---------|----------------|
| Server API + Web | `http://127.0.0.1:18080` |
| Agent metrics | `http://127.0.0.1:19091` |
| Vite dev server | `http://127.0.0.1:15173` |
| Prometheus (future) | `http://127.0.0.1:19090` |
| Grafana (future) | `http://127.0.0.1:13000` |

## 1. Prerequisites

```bash
go version  # >= 1.21
cd ~/projects/ai-platform-study/lightai-go
go build ./cmd/server
go build ./cmd/agent
```

## 2. Start Server

```bash
# Local dev (127.0.0.1 only)
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='Admin@123456'
./server -config configs/server.dev.yaml
# Server listens at http://127.0.0.1:18080
```

## 3. Start Agent

Agent defaults to auto-detect GPU vendor mode (`gpu.collector_mode: auto`).

```bash
# Development (mock GPU + nvidia probe)
./agent -config configs/agent.dev.yaml
# Agent metrics at http://127.0.0.1:19091

# Production auto-detect (recommended)
./agent -config configs/agent.yaml

# Explicit vendor (NVIDIA only, MetaX only)
./agent -config configs/agent.nvidia.yaml
./agent -config configs/agent.metax.yaml

# Disable GPU collectors (system resources only)
# Set gpu.collector_mode: disabled in config
```

### GPU Collector Modes

| Mode | Config | Behavior |
|------|--------|----------|
| `auto` (default) | `gpu.collector_mode: auto` | Probe each vendor's discover.sh; enable those with GPUs |
| `explicit` | `gpu.collector_mode: explicit` | Only collectors explicitly `enabled: true` |
| `disabled` | `gpu.collector_mode: disabled` | No GPU collectors, system resources only |

Auto-detect probes run at startup. Check logs for:
- `auto-detect probe found GPUs vendor=nvidia device_count=N`
- `auto-detect GPU collectors complete enabled_vendors=[nvidia,metax]`
- `auto-detect probe: vendor not available` (exit 10 — not an error)

## 4. Web Dev Server

```bash
cd web && npm install && npm run dev
# Vite at http://127.0.0.1:15173, proxies /api to 18080
```

## 5. Embedded Web Build

```bash
cd web && npm run build
cd ..
go build -tags web -o bin/lightai-server ./cmd/server
LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='Admin@123456' ./bin/lightai-server --config configs/server.dev.yaml
# Visit http://127.0.0.1:18080
```

## 6. Bootstrap Admin

First start creates admin user. Password from:
- `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` env var (preferred)
- Auto-generated (written to `runtime/initial-credentials.txt`, 0600 permissions)

Default username: `admin`. First login requires password change.

Credentials file: `runtime/initial-credentials.txt` — not overwritten on subsequent starts.

To reset admin password:
```bash
./scripts/reset-password.sh                    # auto-generate
./scripts/reset-password.sh --password '<pw>'  # specify
./scripts/reset-password.sh --interactive      # prompt (no shell history)
```
New credentials saved to `runtime/reset-credentials.txt` (0600).

## 7. Login and Password Change

```bash
# Login
curl -c cookies.txt -X POST http://127.0.0.1:18080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"Admin@123456"}'

# Check current user
curl -s -b cookies.txt http://127.0.0.1:18080/api/auth/me

# Change password (required on first login)
CSRF=$(curl -s -b cookies.txt http://127.0.0.1:18080/api/auth/csrf-token | jq -r '.csrf_token')
curl -X POST http://127.0.0.1:18080/api/auth/change-password \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"current_password":"Admin@123456","new_password":"NewPass123"}'
```

## 8. Verify API

```bash
# Nodes
curl -s -b cookies.txt http://127.0.0.1:18080/api/nodes | jq '.'

# GPUs (expects NVIDIA GeForce RTX 5090 Laptop GPU)
curl -s -b cookies.txt http://127.0.0.1:18080/api/gpus | jq '.'

# Metrics targets
curl -s http://127.0.0.1:18080/metrics/targets | jq '.'

# Health
curl -s http://127.0.0.1:18080/healthz
```

## 9. Verify collected_at Updates

```bash
# GPU collected_at updates every ~5 seconds
watch -n 1 "curl -s -b cookies.txt http://127.0.0.1:18080/api/gpus | jq '.[0].collected_at'"
```

## 10. Verify Agent /metrics

```bash
# GPU memory (RTX 5090: ~25.65 GB)
curl -s http://127.0.0.1:19091/metrics | grep 'lightai_gpu_memory_total_bytes'

# GPU utilization (0-100)
curl -s http://127.0.0.1:19091/metrics | grep 'lightai_gpu_utilization_percent'

# Temperature
curl -s http://127.0.0.1:19091/metrics | grep 'lightai_gpu_temperature_celsius'

# Agent collector last success
curl -s http://127.0.0.1:19091/metrics | grep 'lightai_agent_collector_last_success_timestamp_seconds'

# Node online status
curl -s http://127.0.0.1:19091/metrics | grep 'lightai_node_online'
```

## 11. Verify Server /metrics

```bash
# Node counts
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_nodes_total'
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_nodes_online'

# GPU counts
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_gpus_total'
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_gpus_healthy'

# API metrics
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_api_requests_total'
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_api_request_duration_seconds'

# Heartbeat/report counters
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_agent_heartbeats_total'
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_server_agent_reports_total'
```

## 12. Observability — Bundled Mode (default)

```bash
# Check status (includes binary detection)
bash scripts/observability-status.sh

# If Prometheus/Grafana binaries are available:
bash scripts/observability-up.sh
# Prometheus: http://127.0.0.1:19090
# Grafana:    http://127.0.0.1:13000 (admin/lightai)

# Stop
bash scripts/observability-down.sh
```

**Binary not found diagnosis:**

```
DIAGNOSIS: Prometheus binary not found.
Install prometheus or set PROMETHEUS_BIN env var.
Or switch to observability.mode=external or disabled.
```

The development repository does not include Prometheus/Grafana binaries.
Production releases should bundle them or document installation steps.

## 13. Observability — External Mode

For customers with existing Prometheus/Grafana:

1. Set `observability.mode: external` in config.
2. Configure `external.prometheus_url` and `external.grafana_url`.
3. LightAI still exposes Server /metrics and Agent /metrics.
4. Import LightAI Grafana dashboards from `deploy/observability/grafana/dashboards/`.
5. Configure your Prometheus to scrape:
   - `http://<server-ip>:18080/metrics` (Server)
   - `http://<agent-ip>:19091/metrics` (Agent)
   - Use `/metrics/targets` for HTTP SD agent discovery.

## 14. Observability — Disabled Mode

Set `observability.mode: disabled`. LightAI does not start Prometheus/Grafana.
/metrics endpoints remain available for external monitoring systems.

## 15. Grafana — Admin Account

- Username: `admin`
- Password: auto-generated on first start if `LIGHTAI_GRAFANA_ADMIN_PASSWORD` env var is not set.
- First-init credentials saved to `runtime/observability/grafana.credentials` (0600).
- **If Grafana DB already exists**, `LIGHTAI_GRAFANA_ADMIN_PASSWORD` env var will NOT modify the existing DB password. Use the reset scripts instead.

To reset Grafana admin password:
```bash
# Reset Grafana only (recommended if Grafana DB already exists)
./scripts/reset-grafana-password.sh                    # auto-generate
./scripts/reset-grafana-password.sh 'NewPass123!'      # specify password
./scripts/reset-grafana-password.sh --interactive      # prompt (no shell history)

# Reset via unified script (Grafana-only mode)
./scripts/reset-password.sh --grafana-only             # auto-generate
./scripts/reset-password.sh --grafana-only --password '<pw>'
./scripts/reset-password.sh --grafana-only --interactive
```

The reset scripts:
- Stop Grafana if running
- Run `grafana --homepath <path> --config <path> cli admin reset-admin-password <new-password>` (global flags BEFORE `cli` subcommand — required for Grafana 13+)
- Update `runtime/observability/grafana.credentials` (used by `start-observability.sh` on restart)
- Write human-readable record to `runtime/reset-credentials.txt`
- Restart Grafana if it was previously running

**Important**: The `reset-password.sh` script without `--grafana-only` resets BOTH the LightAI Web admin AND Grafana admin passwords. Use `--web-only` or `--grafana-only` to target a single component.

Default dashboards:
- LightAI Overview (`/d/lightai-overview`)
- GPU Resources (`/d/lightai-gpu-resources`)
- Agent Health (`/d/lightai-agent-health`)

## 16. LAN / Server Deployment

For external browser access, use `0.0.0.0` listen addresses:

```yaml
# Server
host: "0.0.0.0"
port: 18080

# Agent metrics
metrics:
  host: "0.0.0.0"
  port: 19091
  advertise_addr: "<agent-ip>:19091"
```

Browser accesses: `http://<server-ip>:18080`.
`127.0.0.1` is for local dev only; `0.0.0.0` is the listen address, not the browser URL.

## 17. Firewall Ports

| Port | Service | Required for |
|------|---------|-------------|
| 18080/tcp | Server API + Web | Browser access |
| 19091/tcp | Agent metrics | Prometheus scrape (LAN) |
| 19090/tcp | Prometheus | Prometheus UI |
| 13000/tcp | Grafana | Grafana UI |

Security: do NOT expose 18080/19090/13000 directly to public internet.
Use VPN, bastion host, or reverse proxy. TLS/HTTPS is a future enhancement.

## 18. Debug Bundle

```bash
bash scripts/collect-debug-bundle.sh
# Output: dist/debug-bundles/lightai-debug-<timestamp>.tar.gz
```

Contains: Server/Agent logs, sanitized configs, system info, nvidia-smi output,
healthz, metrics/targets, agent metrics (first 100 lines).

All output in English. No passwords, tokens, or CSRF secrets included.

## 19. Verify GPU Collector Architecture

```bash
# Confirm external command collector is active
grep "external gpu collector enabled" logs/lightai-agent.log

# Run NVIDIA scripts directly
bash deploy/collectors/gpu/nvidia/discover.sh
bash deploy/collectors/gpu/nvidia/metrics.sh

# Confirm built-in NvidiaCollector is NOT the default path
# (Agent log should show mode=external, NOT mode=builtin)
grep "mode=external" logs/lightai-agent.log
```

## 20. Verify node_id Stability

```bash
# Start Agent, note node_id from /api/nodes
# Restart Agent
# Node count should still be 1, same node_id, no duplicate nodes
curl -s -b cookies.txt http://127.0.0.1:18080/api/nodes | jq 'length'
```

## 21. Common Issues

### Port already in use

```bash
fuser -k 18080/tcp
fuser -k 19091/tcp
```

### Agent registration failed

```bash
curl http://127.0.0.1:18080/healthz
grep agent_token configs/agent.dev.yaml
```

### GPU collector not available

```bash
which nvidia-smi
bash deploy/collectors/gpu/nvidia/discover.sh
grep "collector" logs/lightai-agent.log | grep -i error
```

### Prometheus/Grafana binaries missing

```bash
bash scripts/observability-status.sh
# Output will show MISSING and clear diagnosis.
```

### Reset database

```bash
rm -f data/lightai.db runtime/initial-credentials.txt
# Restart Server to re-initialize.
```

### Embedded web not showing

```bash
cd web && npm run build
go build -tags web -o bin/lightai-server ./cmd/server
```
