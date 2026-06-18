> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference or historical compatibility document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go Resource Monitoring Design

> Last updated: 2026-06-13. Covers Phase 0 through Phase 3W+.

## 1. Collector Architecture

### GPUCollector Interface

All GPU vendors implement `GPUCollector`:
- `Name() string`
- `Vendor() string`
- `Discover(ctx) ([]GPUDeviceInfo, *CollectorDiagnosis)`
- `Metrics(ctx) ([]GPUMetricInfo, *CollectorDiagnosis)`

### Default Product Path: ExternalCommandCollector

The default GPU collector runs external scripts that output **LightAI GPU Collector Protocol**.

- Go Agent parses only the protocol — never vendor CLI output directly.
- Scripts handle vendor tool output format differences.
- Adding a new vendor requires only new scripts, not Go Agent changes.

### NVIDIA

Active path: `deploy/collectors/gpu/nvidia/discover.sh` + `metrics.sh`.
Scripts call `nvidia-smi --query-gpu=... --format=csv,noheader,nounits` and convert output to protocol.

### Built-in NvidiaCollector — DEPRECATED

The old `NvidiaCollector` (Go code parsing nvidia-smi CSV directly) is kept for reference only.
It is NOT the default product path. New vendors must NOT add Go-native parsers.

### MetaX — Scripts Ready (mock verified, hardware pending)

Scripts prepared at `deploy/collectors/gpu/metax/`. Require real MetaX hardware.
When hardware is available: adapt `discover.sh` and `metrics.sh` for `mx-smi` output.
No Go Agent changes needed.

## 2. LightAI GPU Collector Protocol

All vendor scripts output this protocol. Three line types:

```
STATUS vendor=<v> ok=true|false message="..."
DEVICE vendor=<v> index=<n> uuid=<u> name="<n>" pci_bus_id=<p> driver_version=<d> memory_total_bytes=<b>
METRIC vendor=<v> index=<n> uuid=<u> name="<n>" memory_total_bytes=<b> memory_used_bytes=<b> memory_free_bytes=<b> gpu_utilization_percent=<p> memory_utilization_percent=<p> temperature_celsius=<t> power_draw_watts=<w> health=<h> status=<s>
```

Rules:
- Key=value, quoted values for strings with spaces.
- Memory: bytes (MB from nvidia-smi → multiply by 1024×1024).
- Utilization: 0-100 percent.
- Temperature: Celsius. Power: Watts.
- null/N/A for missing optional fields.
- All output in English. No Chinese in protocol.

## 3. Script Exit Codes

| Code | Meaning |
|------|---------|
| 0 | success |
| 10 | not_available (command/tool missing) |
| 20 | partial_success (some fields missing) |
| 30 | command_failed |
| 40 | parse_failed |
| 50 | permission_denied |

Agent behavior:
- exit 0: parse stdout normally.
- exit 10: log "collector not_available", not an Agent error.
- exit 20: parse best-effort, log warning.
- exit >=30: log error, retain stderr summary. Does NOT crash Agent.

## 4. /metrics and Latest Snapshot

- Agent maintains a **latest snapshot** updated by the collector loop.
- `/metrics` reads ONLY the latest snapshot — never triggers nvidia-smi.
- Prometheus scrape frequency is independent of collect frequency.
- `collect_interval` (5s) ≠ `report_interval` (5s) ≠ `scrape_interval` (Prometheus default 5s).

## 5. /api/gpus and Server Current State

- `/api/gpus` queries **Server current state** from SQLite.
- Server state is updated by Agent resource reports.
- Server does NOT query Prometheus for GPU state.
- Server does NOT execute nvidia-smi.

## 6. Agent lightai_gpu_* Metrics

Exposed on Agent `:19091/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `lightai_gpu_memory_total_bytes` | gauge | Total GPU memory |
| `lightai_gpu_memory_used_bytes` | gauge | Used GPU memory |
| `lightai_gpu_memory_free_bytes` | gauge | Free GPU memory |
| `lightai_gpu_utilization_percent` | gauge | GPU utilization 0-100 |
| `lightai_gpu_memory_utilization_percent` | gauge | Memory utilization 0-100 |
| `lightai_gpu_temperature_celsius` | gauge | Temperature |
| `lightai_gpu_power_draw_watts` | gauge | Power draw |
| `lightai_gpu_health_status` | gauge | 1=healthy, 0=not |
| `lightai_gpu_available_status` | gauge | 1=available, 0=not |
| `lightai_agent_collector_last_success_timestamp_seconds` | gauge | Last success |
| `lightai_agent_collector_errors_total` | counter | Collector errors |
| `lightai_agent_report_success_total` | counter | Report successes |
| `lightai_agent_report_errors_total` | counter | Report errors |
| `lightai_node_online` | gauge | 1=online |

## 7. Data Units

- API/DB/Collector: bytes (e.g., `memory_total_bytes`).
- API/DB: percent 0-100 (e.g., `gpu_utilization_percent`).
- Prometheus ratio 0-1: not currently used (stored as 0-100 percent; Prometheus queries can divide by 100).
- Vendor MB must be multiplied by 1024×1024 to bytes INSIDE the collector.

## 8. Future Evolution

The current external script approach is the first-stage field adaptation method.
Long-term: learn from GPUStack's detector/provider architecture, gradually implement
Native/API/C library providers (NVML, ROCm, etc.) under the same GPUCollector interface.
