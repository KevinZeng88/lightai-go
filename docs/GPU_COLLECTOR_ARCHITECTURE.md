# GPU Collector Architecture

## Layers

```
┌─────────────────────────────────────────────────────┐
│ GPU Source / Collector Implementation                │
│ ScriptGPUCollector (current RC1)                     │
│ Future: APIGPUCollector, SDKGPUCollector, etc.       │
├─────────────────────────────────────────────────────┤
│ Vendor Adapter / Parser                              │
│ ParseProtocolOutput() → GPUDeviceInfo + GPUMetricInfo│
├─────────────────────────────────────────────────────┤
│ Normalizer                                           │
│ NormalizeGPUs() → []GPUResource                      │
├─────────────────────────────────────────────────────┤
│ CurrentNodeSnapshot                                  │
│ metrics.Snapshot.GPUResources (single source)        │
├─────────────────────────────────────────────────────┤
│ Output Sinks (all read from same GPUResources)       │
│ - Agent /metrics (Prometheus gpuCollector)           │
│ - Agent report payload (ResourceReport.GPUResources) │
│ - Server resource storage (gpu_devices table)        │
│ - Server API (GET /api/gpus)                         │
│ - Web (vendor-neutral)                               │
└─────────────────────────────────────────────────────┘
```

## Key Types

### GPUCollector Interface

```go
type GPUCollector interface {
    Name() string
    Vendor() string
    Discover(ctx) ([]GPUDeviceInfo, *CollectorDiagnosis)
    Metrics(ctx) ([]GPUMetricInfo, *CollectorDiagnosis)
}
```

Script collectors implement this interface. Future API/SDK collectors do too.

### GPUResource (Unified Model)

Single vendor-neutral struct. All sinks consume this type.

```go
type GPUResource struct {
    Vendor, Index, UUID, Name, PCIBusID, DriverVersion string
    MemoryTotalBytes, MemoryUsedBytes, MemoryFreeBytes uint64
    GPUUtilization, MemUtilization, Temperature, PowerDraw *float64
    Health, Status string
    CollectedAt time.Time
}
```

### CurrentNodeSnapshot

`metrics.Snapshot` holds the current unified state:

```go
type Snapshot struct {
    GPUResources []GPUResource  // ← single source of truth
    System       *SystemSnapshot
    // ...
}
```

## Data Flow

1. **Collect**: ScriptGPUCollector runs shell scripts → ParseProtocolOutput → GPUDeviceInfo + GPUMetricInfo
2. **Normalize**: NormalizeGPUs(devices, metrics) → []GPUResource (merge by vendor+uuid)
3. **Snapshot**: snap.SetGPUResources(resources) → atomically replaces current state
4. **Export**: Single loop over GPUResources drives all outputs

## Adding a New GPU Vendor

1. Write a discover script and metrics script following the LightAI GPU Collector Protocol
2. Add an ExternalCollectorDef in the agent config
3. That's it — Server, API, Web auto-support the new vendor

The protocol format is:
```
STATUS vendor=<name> ok=true|false message="..."
DEVICE vendor=<name> index=<n> uuid=<id> name="..." pci_bus_id=... driver_version=... memory_total_bytes=<n|"null">
METRIC vendor=<name> index=<n> uuid=<id> name="..." memory_total_bytes=<n> memory_used_bytes=<n> memory_free_bytes=<n> gpu_utilization_percent=<n> memory_utilization_percent=<n> temperature_celsius=<n> power_draw_watts=<n> health=<healthy|degraded|error|unknown> status=<available|unavailable>
```

## Future Collector Implementations

Script collectors are the RC1 implementation. The architecture supports pluggable collectors:

```go
// Future example — same interface, different implementation:
type APIGPUCollector struct { ... }
func (a *APIGPUCollector) Discover(ctx) ([]GPUDeviceInfo, *CollectorDiagnosis) { ... }
func (a *APIGPUCollector) Metrics(ctx) ([]GPUMetricInfo, *CollectorDiagnosis) { ... }
```

All must output the same `GPUResource` via `NormalizeGPUs()`.

## GPU Collector Auto-Detect

LightAI Agent defaults to automatic GPU vendor detection.

### Modes

| Mode | Behavior | Config |
|------|----------|--------|
| `auto` (default) | Probe each vendor. Enable collectors that return DEVICE. | `gpu.collector_mode: auto` |
| `explicit` | Only collectors explicitly `enabled: true` in config. | `gpu.collector_mode: explicit` |
| `disabled` | No GPU collectors. System resources only. | `gpu.collector_mode: disabled` |

### Auto-Detect Probe Rules

1. Agent iterates over built-in probe list (nvidia, metax) or custom `gpu.auto_detect.probes`.
2. Each probe runs `deploy/collectors/gpu/<vendor>/discover.sh`.
3. Exit code semantics:
   - **0 + DEVICE lines** → enable this vendor collector.
   - **0 + no DEVICE** → skip, record diagnostic.
   - **10 (not_available)** → skip, info log (not an error).
   - **>= 30** → warn log, skip, other collectors continue unaffected.
4. Enriched probe output is logged at startup: `enabled_vendors: [nvidia, metax]`.

### Multi-Vendor Merge

When both NVIDIA and MetaX are discovered on the same host:
- Both collectors run in the same collection cycle.
- Results are merged using `vendor + uuid` as the key.
- `index` is not a cross-vendor unique identifier.
- One collector's failure does not clear another collector's results (the registry retains the previous successful state per the `Collect()` contract).
- Server `/api/gpus` returns all GPUs from all vendors.
- Agent `/metrics` exposes all GPU metrics with `vendor` as a label.

### Probe Example

```yaml
# Built-in probes (default):
gpu:
  collector_mode: auto
  # auto_detect.probes can override:
  # auto_detect:
  #   probes:
  #     - name: "nvidia"
  #       vendor: "nvidia"
  #       discover_cmd: "deploy/collectors/gpu/nvidia/discover.sh"
  #       metrics_cmd: "deploy/collectors/gpu/nvidia/metrics.sh"
```
