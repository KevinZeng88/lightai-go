# MetaX GPU Collector

Phase 2C Initial Support. Targets mx-smi 2.3.1 output format.

## Supported Models

MetaX C-series (C500, C550, C600, C800, etc.). Model name is dynamically resolved:
- `mx-smi` summary table: full name (e.g., "MetaX C500") — preferred.
- `mx-smi -L` raw model: normalized via `collector_normalize_metax_name`.
- MXC500 → MetaX C500, MXC550 → MetaX C550, etc.
- No hardcoded model names in scripts.

## Scripts

### discover.sh
- Uses `mx-smi -L` for device list.
- Uses `mx-smi` header for `Kernel Mode Driver Version`.
- Outputs: STATUS + N× DEVICE lines.

### metrics.sh
- Uses `mx-smi` default summary for real-time metrics.
- Uses `mx-smi -L` for index → uuid mapping.
- Does NOT call mx-smi per GPU (no `mx-smi -i N` loops).
- Max 2 mx-smi calls per execution.
- Outputs: STATUS + N× METRIC lines.

## Protocol

Scripts output LightAI GPU Collector Protocol (text lines):
- `STATUS vendor=metax ok=true|false message="..."`
- `DEVICE vendor=metax index=N uuid=... name="..." ...`
- `METRIC vendor=metax index=N uuid=... name="..." ...`

Memory in bytes. Utilization in percent (0-100). Temperature in Celsius. Power in Watts.

## Agent User Permissions

Device nodes:
- `/dev/mxcd` (root:video)
- `/dev/dri/card*` (root:video)

Non-root Agent users must be in the `video` group:
```bash
usermod -aG video lightai
```

Container deployment:
```bash
--device=/dev/mxcd --device=/dev/dri --group-add video
```

## Future Native Provider

Current: external script calling mx-smi CLI.
Future: Native provider via MxSML / libmxsml.so / MxSml.h / Go binding.
The LightAI GPU Collector Protocol remains the same; only the backend changes.

## Phase 2C Deferred

Agent integration with MetaX collector awaits real hardware verification.
Once discover.sh and metrics.sh output is confirmed on the target server,
enable the metax collector in `configs/agent.dev.yaml` and run Agent.
