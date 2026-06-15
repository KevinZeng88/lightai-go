# LightAI Go Development Phase Status

> Last updated: 2026-06-14
> Current: RC1 Hotfix — credentials, logging, password reset, patch tooling

## Phase Summary

| Phase | Name | Status | Commit |
|-------|------|--------|--------|
| 0 | Server/Agent skeleton | Done | `afa14d9` |
| 0.5 | Auth, tenant, RBAC | Done | `ab774d2` |
| 1 | Agent register & heartbeat | Done | `d3e8edb` |
| 2A | System/Registry/Mock | Done | `f259e42` |
| 2B | NVIDIA Collector | Done | `7b0e039` |
| 2B+ | node_id hardening | Done | `7649383` |
| 2C | MetaX Collector | **Scripts Ready (mock verified, hardware pending)** | RC1 |
| 3W | Web Console MVP | Done | `5689c21` |
| 3W+ | Collector/Observability/Network | Done | `588e479` |
| 3W+ hotfix | Agent/Server metrics snapshot | Done | `1a1e374` |
| 3W+ hotfix | Web Observability pages | Done | `e870ba6` |
| 3W+ finalization | Server gauges, API middleware | Done | `34a3ac5` |
| 3W+ finalization | API metrics exclude /metrics | Done | `28dc77b` |
| 3W+ closure | Documentation | Done | current |
| 3W+ closure | Hotfix: GPU gauge panels + dashboard polish | Done | `2d06174` |
| RC1 hotfix | v0.1.1 version-based cumulative patch system | Done | `832c9c5` |
| RC1 hotfix | v0.1.2 nvidia default + stable node identity | Done | `40a1048` |
| RC1 hotfix | v0.1.3 strict node_id, no agent_id fallback | Done | `c45e2af` |
| RC1 hotfix | Persist credentials + reset password + detailed logs | Done | `063670c` |
| RC1 hotfix | Standardize log files + protect credentials output | Done | `95710a0` |
| RC1 hotfix | Fix patch manifest name/format mismatch | Done | `10a6d02` |
| RC1 hotfix | Remove python3 dependency from patch apply | Done | `f629311` |
| RC1 hotfix | Fix Grafana password reset (CLI flag order for v13+, credentials sync) | Done | current |
| RC1 hotfix | GPU collector auto-detect (auto/explicit/disabled modes) | Done | current |

## Key Architecture Decisions (Current State)

### GPU Collector
- **Default**: ExternalCommandCollector (vendor scripts → protocol).
- NVIDIA: `deploy/collectors/gpu/nvidia/discover.sh` + `metrics.sh`.
- Built-in NvidiaCollector: deprecated, not default path.
- MetaX: Scripts ready (mock verified, hardware pending). Scripts at `deploy/collectors/gpu/metax/`.

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

## RC1 Hotfix Deliverables

### Credentials
- `runtime/initial-credentials.txt` (0600): auto-generated on first start, never overwritten.
- `runtime/reset-credentials.txt` (0600): written on password reset.
- Passwords never logged to stdout, stderr, or structured log files.
- `scripts/reset-password.sh`: auto-generate / --password / --interactive modes.
- `scripts/reset-grafana-password.sh`: Grafana-only, same modes.

### Logging
- Server main log: `logs/lightai-server.log`, Agent: `logs/lightai-agent.log`.
- Dual-write: stdout (nohup wrapper) + dedicated file.
- Configurable: level, dir, file, stdout, file_enabled, append, max_size_mb, max_files, retention_days.
- Log rotation by size; retention-based cleanup on startup.

### Patch Tooling
- `scripts/package-patch.sh --from X --to Y`: generates incremental patch tarball.
- `apply-patch.sh`: shell-native (no python3 required). Uses `patch-files.tsv` (tab-separated).
- SHA256 verification, permission restoration, backup before overwrite.
- `patch-manifest.json` retained as optional audit metadata.

## Remaining Known Limitations

1. Prometheus/Grafana binaries not included in dev repository (bundled mode needs download/preparation).
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
