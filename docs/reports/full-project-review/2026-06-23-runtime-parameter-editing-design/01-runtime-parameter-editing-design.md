# Runtime Parameter Editing Design

> Date: 2026-06-23
> Purpose: Design multi-layer parameter editing for model library, runtime config, and deployment

---

## 1. Core Design Principles

### 1.1 NBR Is the Source of Truth
RunPlan must only read from NBR snapshot + ModelArtifact/ModelLocation snapshot + Deployment overrides. It must NOT dynamically read BackendVersion or BackendRuntime at resolution time.

### 1.2 Copy-on-Create, Independent After
Each layer deep-copies from its parent at creation time. Parent edits never propagate to children. Only explicit user-initiated re-sync can pull updates.

### 1.3 Every Parameter Has Enabled/Disabled State
A parameter can be present but disabled, meaning it does NOT participate in the final RunPlan output. This is different from "absent" (not configured) or "empty value" (configured with empty string).

### 1.4 Parameter Type Taxonomy
Must distinguish 8 categories that behave differently in the system.

---

## 2. Parameter Type Taxonomy

| Category | Example | Enters Docker env/args? | Stored Where |
|----------|---------|------------------------|-------------|
| Capability metadata | supported_formats, blocked_architectures | NO | capabilities_json |
| Parameter schema | {name, type, default, required} | NO | parameter_defs_json |
| Parameter values | gpu-memory-utilization=0.9 | YES (if enabled) | parameters_json |
| Env values | CUDA_VISIBLE_DEVICES=0 | YES (if enabled) | default_env_json |
| Args values | --max-model-len 4096 | YES (if enabled) | args_override_json |
| Container config | image, entrypoint, ports, volumes, devices | YES | docker_json, config_snapshot_json |
| Runtime requirements | min_gpu_memory, required_devices | NO (preflight only) | runtime_requirements_json |
| Deployment overrides | user-specified at deployment time | YES (highest priority) | parameters_json, env_overrides_json |

---

## 3. Parameter Record Structure

Each parameter at every layer should be representable as:

```json
{
  "key": "gpu-memory-utilization",
  "type": "float",
  "target": "arg",
  "cli_name": "--gpu-memory-utilization",
  "env_name": null,
  "enabled": true,
  "value": 0.9,
  "default": 0.9,
  "source": "node_backend_runtime",
  "copied_from": "backend_runtime:rt-vllm-nvidia",
  "user_override": true,
  "validation": {"min": 0.1, "max": 0.95}
}
```

Fields:
- `key`: Logical parameter name (snake_case)
- `type`: string, int, float, bool, enum
- `target`: "arg" (CLI flag), "env" (Docker env), "container" (Docker option), "metadata" (no output)
- `cli_name`: CLI flag name (--gpu-memory-utilization)
- `env_name`: Environment variable name (if target=env)
- `enabled`: Whether this parameter participates in final output
- `value`: Current value
- `default`: Default value from schema
- `source`: Which layer set this value
- `copied_from`: Parent record reference
- `user_override`: Whether user explicitly changed this value
- `validation`: Type-specific validation rules

---

## 4. Layer Model

### 4.1 BackendVersion / Catalog
**Saves**: Parameter schema, capability metadata, default value suggestions, default runtime template seed
**Does NOT save**: User runtime config, node devices, deployment overrides
**Does NOT participate in RunPlan**: Only used as template source for creating BackendRuntime

### 4.2 BackendRuntime
**Saves**: User-level runtime template snapshot — parameter schema snapshot, parameter values, env, args, container config, image, entrypoint, health check, default ports
**Used for**: Creating NodeBackendRuntime
**Does NOT participate in RunPlan**: Only used as template source

### 4.3 NodeBackendRuntime
**Saves**: Final backend runtime snapshot — parameter schema snapshot, parameter values snapshot, env, args, container config, image, entrypoint, health check, resource defaults, backend capability snapshot
**This IS the RunPlan backend-side source of truth**

**NBR Schema Snapshot Requirement**: NBR must save both parameter schema AND parameter values. RunPlan validation must NOT query BackendVersion/BackendRuntime at resolution time. The schema snapshot can be stored as:
- Option A: Separate `parameter_schema_json` column
- Option B: Embedded in `config_snapshot_json`

Either way, RunPlan must read schema directly from NBR snapshot for validation.

### 4.4 ModelArtifact
**Saves**: Model global info — format, arch, quantization, capabilities, discovered metadata, runtime requirements, model parameter defaults (served_model_name, context_length, etc.)
**Does NOT save**: Docker image, privileged, security-opt, devices, ports, shm, ipc
**Boundary**: Model parameters only participate in generating Deployment defaults. They NEVER override NBR container config.

### 4.5 ModelLocation
**Saves**: Model location on a node — path, file/directory type, size/checksum, location-specific runtime requirements, consistency attestation
**Does NOT save**: Backend image, privileged, security-opt, devices, ports, shm, ipc
**Boundary**: Model location only provides path information for mount generation. It NEVER overrides NBR container config.

### 4.6 ModelDeployment
**Saves**: Final overrides — selected NBR reference, selected ModelLocation reference, parameter overrides (with enabled/disabled), env overrides, args overrides, resource overrides
**Deployment override has highest priority**

**Disabled Override / Tombstone**: Deployment must explicitly save disabled overrides. "Absent" cannot distinguish between:
- Upstream has no parameter
- User explicitly disabled parameter
- User never set parameter

Structure for disabled parameters:
```json
{
  "key": "max-model-len",
  "enabled": false,
  "override_state": "disabled",
  "source": "deployment",
  "copied_from": "node_backend_runtime:xxx"
}
```
- Disabled parameters do NOT enter final args/env
- Disabled ≠ empty value (empty value still requires type validation)
- Re-enable can restore copied value or user re-enters

---

## 5. Copy-on-Create Flow

### 5.1 BackendVersion → BackendRuntime
- Deep copy parameter schema
- Deep copy default parameter values
- Deep copy default env/args/container template
- Record `copied_from_backend_version_id`
- Record `source_hash` for template expiry detection

### 5.2 BackendRuntime → NodeBackendRuntime
- Deep copy BackendRuntime's complete runtime snapshot
- Include enabled/disabled state for all parameters
- NBR becomes independent after creation
- Record `copied_from_backend_runtime_id`
- Record `copied_from_hash`

### 5.3 ModelArtifact/ModelLocation → Deployment
- Deep copy model default parameters
- Deep copy served_name, context_length, runtime requirements
- Deep copy location path / model container path mapping
- Deployment becomes independent after creation

### 5.4 NodeBackendRuntime → Deployment
- Deep copy NBR parameter values and enabled/disabled state
- Deep copy relevant container config defaults
- Deployment can override or disable parameters
- Deployment does NOT follow NBR changes automatically

### 5.5 Reset / Re-sync (Future)
- "Reset this parameter to copied value"
- "Reset all parameters from source snapshot"
- "Re-sync from current upstream template" (with diff preview, user confirmation)
- NOT required for first implementation

---

## 6. RunPlan Integration

### Current (Broken)
```
BackendVersion.default_args → Layer 1
BackendVersion.default_backend_params → Layer 2
BackendRuntime.args_override → Layer 3
Deployment.parameters → Layer 4
```

### Target (Correct)
```
NBR.container_config_snapshot → container config
NBR.parameter_values_snapshot → args/env
ModelArtifact.model_defaults → model path, served name, etc.
ModelLocation.path_snapshot → mount path
Deployment.overrides → final overrides (highest priority)
→ merge → ResolvedRunPlan / AgentRunSpec
```

### Merge Rules
1. NBR container config is the base
2. NBR parameter values produce args/env
3. ModelArtifact provides model-side defaults
4. Deployment overrides have highest priority
5. Deployment disabled parameters are excluded from output
6. Empty values are excluded (no `-e KEY=` or `--flag ""`)
7. Metadata never enters Docker env/args
8. High-risk NBR params are NOT blocked (logged in audit only)

---

## 7. Web UI Design

### 7.1 Reusable Component: RuntimeParameterEditor

Props:
- `parameters`: Array of parameter records
- `schema`: Parameter definitions (type, default, validation, group)
- `editable`: Boolean (can user modify?)
- `showSource`: Boolean (show where value came from?)

Features:
- Parameter grouping (GPU Memory, Context, Concurrency, Batch, Container Options)
- Parameter search
- Parameter description / help text
- Per-parameter: type, default value, current value, enabled checkbox
- Disabled state explanation
- Source display (from template / user override)
- Clear override button
- Validation error display
- Final args/env preview
- Dangerous options group (privileged, ipc, security-opt, devices)

### 7.2 Integration Points

| Page | Component | Parameters |
|------|-----------|-----------|
| BackendRuntimesPage | RuntimeParameterEditor | BR parameter values |
| RunnerConfigsPage | RuntimeParameterEditor | NBR parameter values |
| ModelArtifactsPage | RuntimeParameterEditor | Model default parameters |
| ModelDeploymentsPage | RuntimeParameterEditor | Deployment overrides |

### 7.3 Interaction Rules
1. Default checked state from current layer snapshot
2. Unchecked = parameter excluded from final output
3. Value change = marked as user override
4. Clear override ≠ uncheck
5. Empty value cannot silently generate empty env/args
6. Validation error shows parameter name, source layer, error reason
7. Deployment page shows final merged result

---

## 8. Backend-Specific Memory/Resource Parameters

**Important**: These parameters are defined in backend-specific parameter schema, NOT as hardcoded UI fields. UI groups them under "显存 / 上下文 / 并发 / 批处理" for display purposes only. There is NO unified `gpu_memory_limit` field.

**llama.cpp note**: llama.cpp has no direct memory fraction parameter. Memory is controlled indirectly via gpu_layers + context + batch. The UI should explain this relationship, not fake a percentage parameter.

### 8.1 vLLM

| Parameter | CLI Flag | Type | Default | Range |
|-----------|----------|------|---------|-------|
| gpu-memory-utilization | --gpu-memory-utilization | float | 0.9 | 0.1-0.95 |
| max-model-len | --max-model-len | int | (auto) | >0 |
| max-num-seqs | --max-num-seqs | int | 256 | >0 |
| max-num-batched-tokens | --max-num-batched-tokens | int | (auto) | >0 |

### 8.2 SGLang

| Parameter | CLI Flag | Type | Default | Range |
|-----------|----------|------|---------|-------|
| mem-fraction-static | --mem-fraction-static | float | 0.9 | 0.1-0.95 |
| context-length | --context-length | int | (auto) | >0 |
| max-running-requests | --max-running-requests | int | 256 | >0 |

### 8.3 llama.cpp

| Parameter | CLI Flag | Type | Default | Range |
|-----------|----------|------|---------|-------|
| n-gpu-layers | --n-gpu-layers / -ngl | int | (auto) | -1=all, >=0 |
| ctx-size | --ctx-size / -c | int | 2048 | >0 |
| batch-size | --batch-size | int | 512 | >0 |
| ubatch-size | --ubatch-size | int | 512 | >0 |

**Note**: llama.cpp has no direct memory fraction parameter. Memory is controlled indirectly via gpu_layers + context + batch. UI should explain this.

### 8.4 UI Grouping
Group these under "显存 / 上下文 / 并发 / 批处理" in the parameter editor. The grouping is UI-only — the underlying parameters are backend-specific.

---

## 9. Validation Rules

| Type | Rule | Error |
|------|------|-------|
| float | min ≤ value ≤ max | "Value must be between {min} and {max}" |
| int | value ≥ 0 | "Value must be non-negative" |
| string | non-empty if enabled | "Value cannot be empty when enabled" |
| enum | value in allowed_values | "Value must be one of: {values}" |
| bool | true/false | "Value must be true or false" |
| required | value must be present | "Required parameter missing" |

Validation runs at:
1. Edit time (immediate feedback)
2. Save time (server-side validation)
3. Preflight time (final validation before start)
