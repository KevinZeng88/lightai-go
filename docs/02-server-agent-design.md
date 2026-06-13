# LightAI Go Server / Agent Design

> Last updated: 2026-06-13. Covers Phase 0 through Phase 3W+. Current defaults are accurate.

## 1. Architecture

Server = control plane. Agent = execution plane.

```
Browser / API Client
  → Server (:18080) → SQLite
  → Agent (:19091) → OS / GPU / Docker
```

## 2. Network Addressing

Three concepts must not be confused:

| Concept | Meaning | Example (dev) | Example (LAN) |
|---------|---------|---------------|---------------|
| listen address | What the server binds to | `127.0.0.1:18080` | `0.0.0.0:18080` |
| public URL | What the browser uses | `http://127.0.0.1:18080` | `http://<ip>:18080` |
| metrics advertise addr | What Prometheus scrapes | `127.0.0.1:19091` | `<agent-ip>:19091` |

Rules:
- `127.0.0.1` is for local dev only.
- `0.0.0.0` is the listen address, NOT the browser URL.
- External browsers use the actual server IP or hostname.
- `/metrics/targets` prefers `metrics.advertise_addr`; falls back to `advertised_address + metrics_port`.
- Cookie `Domain` and CSRF `Origin` must match the public URL in LAN deployments.
- Do NOT expose ports 18080/19090/13000 directly to public internet.

## 3. Agent Registration and node_id Stability

Flow:
1. Agent sends `POST /api/agent/register` with bootstrap agent token.
2. Server upserts node by `agent_id`, returns `node_id`.
3. Agent caches `node_id` to `data/agent-state.json`.
4. On restart, Agent compares cached `node_id` with server response.
5. Match → reuse. Mismatch → update cache with WARN log. Server never creates duplicate nodes.

## 4. Heartbeat and Resource Reporting

Default intervals (all configurable):

| Interval | Default | Config |
|----------|---------|--------|
| Heartbeat | 2s | `heartbeat.interval` |
| System collect | 5s | `collectors.system.interval` |
| GPU collect | 5s | per-collector |
| Resource report | 5s | `collectors.report_interval` |
| Request timeout | 5s | `request_timeout` |
| Node offline threshold | 20s | `node_offline_threshold` |

## 5. GPU Collector Architecture

**Default product path: ExternalCommandCollector.**

- All GPU vendors share the `GPUCollector` interface.
- Scripts output **LightAI GPU Collector Protocol** (see doc 03).
- Go Agent parses only the protocol, never vendor CLI output directly.
- NVIDIA: `deploy/collectors/gpu/nvidia/discover.sh` + `metrics.sh`.
- Built-in `NvidiaCollector` is **deprecated** (kept for reference, not default path).
- MetaX: Phase 2C Deferred. Scripts at `deploy/collectors/gpu/metax/`.

## 6. Agent /metrics

- `lightai_gpu_*` and `lightai_agent_*` custom Prometheus metrics.
- Reads from **latest snapshot only** — scrape never triggers nvidia-smi.
- Labels: `node_id`, `agent_id`, `hostname`, `vendor`, `uuid`, `gpu_index`, `gpu_name`.
- Never exposes passwords, tokens, sessions, CSRF.

## 7. Server /metrics

- `lightai_server_*` custom Prometheus metrics.
- Node/GPU gauges read DB on each scrape (GaugeFunc) — consistent with `/api/nodes` and `/api/gpus`.
- API metrics middleware tracks only `/api/*` paths (not `/metrics`, `/healthz`, static assets).
- `lightai_server_api_requests_total` (counter, endpoint/method/code labels).
- `lightai_server_api_request_duration_seconds` (histogram).
- Never exposes passwords, tokens, sessions, CSRF.

## 8. Agent Token vs User Session

Strict separation:
- Agent API (`/api/agent/*`): bootstrap agent token (`Authorization: Bearer`).
- User API: server-side session cookie (`lightai_session`).
- Agent token cannot call user APIs. User session cannot call Agent APIs.
- Agent-created resources: `created_by=system`, `owner_id=null`.

## 9. Logging

- All logs in English. JSON structured format.
- Files: `logs/lightai-server.log`, `logs/lightai-agent.log`.
- Levels: debug, info, warn, error.
- Never log: passwords, session IDs, CSRF tokens, agent tokens.
