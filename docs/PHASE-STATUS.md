# LightAI Go Development Phase Status

> Last updated: 2026-06-13
> Current: Phase 3W+ Documentation Closure

## Phase Summary

| Phase | Name | Status | Commit |
|-------|------|--------|--------|
| 0 | Server/Agent skeleton | Done | `afa14d9` |
| 0.5 | Auth, tenant, RBAC | Done | `ab774d2` |
| 1 | Agent register & heartbeat | Done | `d3e8edb` |
| 2A | System/Registry/Mock | Done | `f259e42` |
| 2B | NVIDIA Collector | Done | `7b0e039` |
| 2B+ | node_id hardening | Done | `7649383` |
| 2C | MetaX Collector | **Deferred** | — |
| 3W | Web Console MVP | Done | `5689c21` |
| 3W+ | Collector/Observability/Network | Done | `588e479` |
| 3W+ hotfix | Agent/Server metrics snapshot | Done | `1a1e374` |
| 3W+ hotfix | Web Observability pages | Done | `e870ba6` |
| 3W+ finalization | Server gauges, API middleware | Done | `34a3ac5` |
| 3W+ finalization | API metrics exclude /metrics | Done | `28dc77b` |
| 3W+ closure | Documentation | Done | current |

## Key Architecture Decisions (Current State)

### GPU Collector
- **Default**: ExternalCommandCollector (vendor scripts → protocol).
- NVIDIA: `deploy/collectors/gpu/nvidia/discover.sh` + `metrics.sh`.
- Built-in NvidiaCollector: deprecated, not default path.
- MetaX: Phase 2C Deferred. Scripts at `deploy/collectors/gpu/metax/`.

### Metrics
- Agent `/metrics`: `lightai_gpu_*`, `lightai_agent_*` from latest snapshot. Scrape never triggers nvidia-smi.
- Server `/metrics`: `lightai_server_*` gauges from DB, API counters (only `/api/*`).
- Verified: RTX 5090 data flowing correctly through both Agent and Server metrics.

### Observability
- Default: bundled mode. LightAI manages Prometheus/Grafana as subprocesses.
- Also supports: external mode (customer's existing stack), disabled mode.
- Prometheus `:19090`, Grafana `:13000` (admin/lightai dev default).
- 3 Grafana dashboards auto-provisioned.
- 7 alert rule templates.
- Docker Compose is dev/demo helper, not the only path.

### Web
- Vue 3 + Element Plus + vue-i18n (zh-CN default, en-US supported).
- Pages: Dashboard, Nodes, GPUs, Metrics Targets, Observability Overview, Prometheus, Grafana.
- Embedded via `-tags web`.
- Dev: Vite at `:15173`, proxies to `:18080`.

### Ports
- Server: `18080`, Agent metrics: `19091`, Prometheus: `19090`, Grafana: `13000`, Vite: `15173`.

### Network
- `127.0.0.1` for local dev; `0.0.0.0` for LAN deployment.
- `metrics.advertise_addr` for Prometheus scrape targets.

## Remaining Known Limitations

1. Prometheus/Grafana binaries not included in dev repository (bundled mode needs installation).
2. Go Server does not yet have full Prometheus/Grafana supervisor (subprocess management is script-based).
3. MetaX Phase 2C requires real MetaX hardware.
4. Runtime Environment (Phase 3) not yet started.
5. Model Registry (Phase 4) not yet started.
6. Instance Lifecycle (Phase 5-7) not yet started.
7. TLS/HTTPS not yet implemented.

## Next Steps

1. Phase 2C: MetaX real hardware adaptation.
2. Phase 3: Runtime Environment management.
3. Phase 4: Model definition management.
4. Phase 5-7: Instance lifecycle, Docker operations, health checks.
5. Phase 8: Web UI enhancements.
6. Phase 9: Server-managed Prometheus/Grafana lifecycle.
