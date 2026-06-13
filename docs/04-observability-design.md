# LightAI Go Observability Design

> Last updated: 2026-06-13. Covers Phase 0 through Phase 3W+.

## 1. Default Integration

Prometheus + Grafana is LightAI Go's **default integrated observability stack**.

Data boundary: Prometheus is for historical metrics, Grafana for visualization.
Prometheus is NOT the resource primary data source. Server state (nodes, GPUs, instances)
comes from Agent reports → SQLite, not from Prometheus queries.

## 2. Three Modes

| Mode | Behavior |
|------|----------|
| `bundled` | LightAI manages Prometheus/Grafana as subprocesses (default) |
| `external` | Customer provides existing Prometheus/Grafana |
| `disabled` | No observability stack; /metrics remain available |

**bundled mode** (default): LightAI starts Prometheus and Grafana as managed subprocesses.
The development repository does NOT include Prometheus/Grafana binaries.
Production releases should bundle them or provide installation steps.
Missing binaries show clear English diagnosis — LightAI core functions are unaffected.

**external mode**: Set `external.prometheus_url` and `external.grafana_url`.
LightAI still exposes /metrics. Import LightAI dashboards into your Grafana.
Configure your Prometheus to scrape Server and Agent /metrics.

**disabled mode**: Prometheus/Grafana not started. /metrics endpoints remain available.

Docker Compose (`deploy/observability/docker-compose.yml`) is a dev/demo helper.
It is NOT the product's only or primary observability deployment method.

## 3. Ports

| Service | Dev address | LAN |
|---------|------------|-----|
| Server /metrics | `http://127.0.0.1:18080/metrics` | `http://<ip>:18080/metrics` |
| Agent /metrics | `http://127.0.0.1:19091/metrics` | `http://<ip>:19091/metrics` |
| Prometheus | `http://127.0.0.1:19090` | `http://<ip>:19090` |
| Grafana | `http://127.0.0.1:13000` | `http://<ip>:13000` |

## 4. Grafana Default Account

- Dev: `admin` / `lightai` (set via `LIGHTAI_GRAFANA_ADMIN_PASSWORD` for production).
- Datasource auto-provisioned to Prometheus.
- Dashboards auto-imported from `deploy/observability/grafana/dashboards/`.

Default dashboards:
1. **LightAI Overview** (`lightai-overview`) — nodes, GPU memory, utilization, API rate.
2. **GPU Resources** (`lightai-gpu-resources`) — per-GPU utilization, memory, temperature, power.
3. **Agent Health** (`lightai-agent-health`) — online status, collector success, report rate.

## 5. Prometheus Configuration

- Scrapes Server at `:18080/metrics`.
- Discovers Agents via `/metrics/targets` HTTP SD.
- Alert rules in `deploy/observability/prometheus/rules/lightai.rules.yml`.

Alert rule templates:
- `NodeOffline` — `lightai_node_online == 0`
- `AgentCollectorError` — collector error rate > 0
- `GPUUnhealthy` — `lightai_gpu_health_status == 0`
- `GPUHighUtilization` — `lightai_gpu_utilization_percent > 90`
- `GPUMemoryHighUsage` — `lightai_gpu_memory_utilization_percent > 90`
- `GPUTemperatureHigh` — `lightai_gpu_temperature_celsius > 85`
- `ServerHighErrorRate` — 5xx error rate elevated

## 6. Server /metrics

| Metric | Type |
|--------|------|
| `lightai_server_info` | gauge |
| `lightai_server_nodes_total` | gauge (DB) |
| `lightai_server_nodes_online` | gauge (DB) |
| `lightai_server_gpus_total` | gauge (DB) |
| `lightai_server_gpus_available` | gauge (DB) |
| `lightai_server_gpus_healthy` | gauge (DB) |
| `lightai_server_api_requests_total` | counter (/api/* only) |
| `lightai_server_api_request_duration_seconds` | histogram |
| `lightai_server_agent_heartbeats_total` | counter |
| `lightai_server_agent_reports_total` | counter |
| `lightai_server_auth_login_total` | counter |
| `lightai_server_auth_login_failed_total` | counter |

Node/GPU gauges read from DB on each scrape — always consistent with `/api/nodes` and `/api/gpus`.
API metrics exclude `/metrics`, `/healthz`, and static assets.

## 7. Agent /metrics

See `docs/03-resource-monitoring-design.md` for full GPU metric list.

Key principle: `/metrics` reads from latest snapshot only — never triggers nvidia-smi on scrape.

## 8. Web Observability Pages

Navigating Observability section:
- **Overview** — Prometheus/Grafana status cards, dashboard shortcuts.
- **Metrics Targets** — `/metrics/targets` table with labels and raw JSON.
- **Prometheus** — status, URL, scrape targets info, open link.
- **Grafana** — status, URL, default login, dashboard links.

Pages show clear diagnosis when services are not running.

## 9. LAN Deployment

For external browser access:
- Set `host: "0.0.0.0"` in server config.
- Set `metrics.host: "0.0.0.0"` and `metrics.advertise_addr: "<ip>:19091"` in agent config.
- Configure CSRF trusted origins to include the public URL.
- Do NOT expose 18080/19090/13000 directly to public internet.

## 10. Security

- Production: set `LIGHTAI_GRAFANA_ADMIN_PASSWORD` via env var.
- Never hardcode production passwords in configs or code.
- Use VPN, bastion host, or reverse proxy for production access.
- TLS/HTTPS is a future enhancement.
