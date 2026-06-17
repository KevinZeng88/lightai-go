# GPUStack Runtime Reference Review

> LightAI Go Phase 0 — GPUStack model runtime chain research for upcoming refactor.
> Date: 2026-06-16

## 1. GPUStack Model Runtime Chain Overview

GPUStack's model runtime chain uses these core concepts:

```
InferenceBackend → Model (with backend/backend_version refs) → ModelInstance → Worker/ServeManager → Container Workload
```

The scheduler lives between Model and ModelInstance, assigning instances to workers with GPU resource checks.

### Key files reviewed

| File | Purpose |
|------|---------|
| `gpustack/schemas/inference_backend.py` | Backend + VersionConfig definitions |
| `gpustack/schemas/models.py` | Model, ModelInstance, state machine |
| `gpustack/worker/backends/base.py` | InferenceServer ABC — base class for all backends |
| `gpustack/worker/backends/vllm.py` | VLLMServer — vLLM backend implementation |
| `gpustack/scheduler/scheduler.py` | Scheduler — worker selection, GPU resource fit |
| `gpustack/worker/serve_manager.py` | ServeManager — worker-side lifecycle management |

---

## 2. InferenceBackend Design

### 2.1 What is a Backend?

A Backend in GPUStack represents an inference engine type — vLLM, SGLang, Ascend MindIE, VoxBox, or a custom backend. It is NOT a deployment, NOT a model instance, and NOT tied to a specific GPU.

### 2.2 Backend Fields

```python
class InferenceBackendBase(SQLModel):
    backend_name: str           # e.g. "vLLM", "SGLang", "Custom"
    version_configs: Dict[str, VersionConfig]  # version → image/command/env
    default_version: str
    default_backend_param: List[str]
    default_run_command: str
    default_entrypoint: str
    default_env: Dict[str, str]
    is_built_in: bool
    health_check_path: str
    parameter_format: ParameterFormatEnum  # "space" or "equal"
    common_parameters: List[str]    # UI hints
```

### 2.3 VersionConfig

Each VersionConfig maps a semantic version string to concrete run configuration:

```python
class VersionConfig(BaseModel):
    image_name: str              # Docker image
    run_command: str             # CLI command (with {{model_path}}, {{port}}, etc.)
    entrypoint: str              # Container entrypoint override
    built_in_frameworks: List[str]  # Auto-detected frameworks
    custom_framework: str        # User-provided framework
    env: Dict[str, str]          # Version-specific env vars
```

### 2.4 Key Backend Methods

- `resolve_target_version(version)`: Resolves which version config to use (exact match → default → latest for non-built-in)
- `get_version_config(version)`: Returns VersionConfig for a version
- `get_run_command(version)`: Returns run_command from version_config or default
- `get_backend_env(version)`: Merges version.env > default_env
- `replace_command_param(version, model_path, port, ...)`: Template substitution in command
- `get_container_entrypoint(version)`: Returns parsed entrypoint
- `get_image_name(version)`: Resolves container image

### 2.5 Built-in Backends

GPUStack registers 5 built-in backends:
- `vLLM`
- `SGLang`
- `MindIE` (Ascend)
- `VoxBox`
- `Custom`

---

## 3. Model Design

### 3.1 Model (User-Facing Definition)

A Model in GPUStack is what the user defines — the model to deploy:

```python
class ModelSpecBase(SQLModel, ModelSource):
    name: str
    replicas: int
    backend: str                   # "vLLM", "SGLang", etc.
    backend_version: str           # optional, for pinning version
    backend_parameters: List[str]  # e.g. ["--tensor-parallel-size", "4"]
    image_name: str                # optional override
    run_command: str               # optional override
    env: Dict[str, str]            # model-level env (HF_TOKEN, etc.)
    gpu_selector: GPUSelector      # gpu_ids, gpus_per_replica
    categories: List[str]          # llm, embedding, image, reranker, etc.
    cpu_offloading: bool
    distributed_inference_across_workers: bool
    extended_kv_cache: ExtendedKVCacheConfig
    speculative_config: SpeculativeConfig
    lora_list: List[LoraListEntry]
```

Source information (HF, ModelScope, local_path) is embedded via `ModelSource` mixin.

### 3.2 Key: Model does NOT contain Docker config

The Model references a `backend` name and optional `backend_version`, but does NOT contain Docker image, devices, privileged, ipc, shm_size, or ulimit settings. Those belong to the Backend Version or auto-detection.

The `image_name` and `run_command` on Model are override fields for custom backends.

---

## 4. ModelInstance Design

### 4.1 ModelInstance (Runtime Entity)

A ModelInstance is the actual running (or scheduled) entity:

```python
class ModelInstanceBase(SQLModel, ModelSource):
    name: str                      # unique name
    model_id: int                  # FK → models
    model_name: str
    worker_id: int                 # assigned after scheduling
    worker_name: str
    worker_ip: str
    port: int                      # main serving port
    ports: List[int]               # all ports [api, dp_rpc, master, connecting]
    pid: int
    backend: str
    backend_version: str
    api_detected_backend_version: str  # actually running version
    state: ModelInstanceStateEnum
    state_message: str
    gpu_indexes: List[int]
    gpu_type: str
    computed_resource_claim: ComputedResourceClaim
    distributed_servers: DistributedServers
    resolved_path: str              # resolved model file path
    download_progress: float
    restart_count: int
    injected_backend_parameters: List[str]
```

### 4.2 State Machine

```
PENDING → ANALYZING → SCHEDULED → INITIALIZING → DOWNLOADING → STARTING → RUNNING
                   ↑                  ↑              ↑              ↑         ↑
                   └──────────────────┴──────────────┴──────────────┴─────────┘
                                       ERROR ⟷ (restart_on_error)
                                                                UNREACHABLE
```

States and who manages them:
- **Scheduler**: PENDING → ANALYZING → SCHEDULED
- **ServeManager**: SCHEDULED → INITIALIZING → (download) → (start) → RUNNING
- **Controller (InferenceServer)**: DOWNLOADING → STARTING → RUNNING
- **Worker**: RUNNING → UNREACHABLE (on heartbeat loss)

---

## 5. Worker Flow — How a Model is Started

### 5.1 ServeManager._start_model_instance()

1. Get Model from server, resolve `backend` name
2. Assign ports (thread-safe): serving port + DP RPC ports + connecting port
3. Resolve `InferenceBackend` from `InferenceBackendManager`
4. Determine fallback container registry
5. Launch subprocess via `multiprocessing.Process(target=ServeManager._serve_model_instance)`
6. Update model instance state → INITIALIZING
7. Start container log persistence threads

### 5.2 InferenceServer (base.py) — __init__

1. Validate worker is found
2. Get model from API server
3. Resolve inference backend (from backend_name, or create synthetic CustomBackend)
4. Watch model instance until state = STARTING (with resolved_path)
5. Set `self._model_path` from resolved path

### 5.3 VLLMServer (vllm.py) — _start()

1. Get deployment metadata (distributed, leader/follower, etc.)
2. Build environment variables (base + vLLM-specific + distributed + vendor-specific)
3. Resolve container image (model.image_name > backend_version config > auto-detect from GPU+runner)
4. Build command and entrypoint
5. Build command arguments (model_path + omni + max_model_len + auto_parallelism + speculative + lora + backend_params + --host/--port/--served-model-name)
6. Create workload plan → `create_workload()`
7. Update model instance with injected_backend_parameters

### 5.4 Image Resolution Priority

```
1) Model.image_name (explicit override)
2) Backend VersionConfig.image_name (user-configured version)
3) Auto-detected from gpustack-runner based on GPU vendor/arch + backend
```

### 5.5 Command Building

vLLM command args are built in a strict order:
1. `model_path` (positional)
2. Versioned command args (from backend version run_command)
3. Omni flags
4. Max model length (derived from pretrained config, capped at 8192)
5. Auto parallelism (tp/pp/dp based on GPU count)
6. Speculative decoding flags
7. Access log filtering
8. Cache report / prompt tokens details
9. Distributed arguments (Ray or MP multi-node)
10. Extended KV cache (LMCache)
11. Ascend 310P flags
12. LoRA arguments
13. User backend_parameters (from model.backend_parameters)
14. --host, --port, --served-model-name (auto-injected)

---

## 6. Scheduler Design

### 6.1 How Scheduler Works

1. Enqueues pending instances (periodic full scan every 180s + event-driven for CREATED)
2. Evaluates model metadata (ANALYZING state):
   - For GGUF models: calculate resource claim
   - For others: evaluate pretrained config (model architecture, categories)
3. Schedule cycle picks items from queue and runs `_schedule_one()`:
   - Filter workers through chain: ClusterFilter → GPUMatchingFilter → LabelMatchingFilter → StatusFilter → BackendFrameworkFilter → LocalPathFilter
   - Select candidates via backend-specific ResourceFitSelector (VLLM, SGLang, AscendMindIE, GGUF, Custom)
   - Score candidates: PlacementScorer + ModelFileLocalityScorer
   - Pick highest score candidate → assign worker_id, gpu_indexes, distributed_servers
   - Update state → SCHEDULED

### 6.2 Relevance to LightAI Go

GPUStack's scheduler is complex and designed for multi-cluster, multi-worker deployment. LightAI Go is targeting small-medium deployments and may not need:
- Full filter chain (ClusterFilter is irrelevant for single-cluster)
- Locality scoring (model files assumed to be on shared storage or pre-placed)
- Distributed inference (Phase 1 is single-node)
- Auto-parallelism calculation

What IS relevant:
- GPU matching (vendor, health, availability)
- GPU lease conflicts
- Resource fitting (VRAM check)
- Port conflict detection

---

## 7. GPUStack UI Reference

GPUStack UI pages relevant to model deployment:
- **Backends page** (`/backends`): List inference backends, add custom backends, manage versions
- **LLModels page** (`/llmodels`): Create models, set backend/backend_version/parameters, manage replicas
- **Model detail page**: View instances, start/stop, view logs
- **Dashboard**: Overview of running instances
- **Playground**: Test inference

### Key UI patterns:
- Backend selection drives available versions, default parameters
- Model creation form shows backend-specific parameter hints (common_parameters)
- Instance state display with real-time updates
- Log viewer per instance

---

## 8. What LightAI Go Should Adopt

### 8.1 Separation of Concerns (Strongly Recommended)

| Concern | GPUStack Location | LightAI Go Target |
|---------|------------------|-------------------|
| Backend family (no vendor) | `InferenceBackend` | `InferenceBackend` |
| BackendVersion definition | `VersionConfig` | `BackendVersion` |
| Runtime config (vendor + Docker) | (in VersionConfig + auto-detect) | `BackendRuntime` |
| Node-level overrides | (not in GPUStack) | `NodeRuntimeOverride` |
| Model files & metadata | `ModelSource` + `Model` | `ModelArtifact` |
| User deployment spec | `Model` (param overrides) | `ModelDeployment` |
| Runtime entity | `ModelInstance` | `ModelInstance` |
| Frozen run plan | (generated at worker) | `ResolvedRunPlan` |
| Container execution | `WorkloadPlan` + gpustack-runtime | `DockerExecutor` |

### 8.2 Backend-Version-Runtime Split

GPUStack's `VersionConfig` under `InferenceBackend` bundles image, run_command, entrypoint, and env into a single dict per version. LightAI Go further splits this into three layers (see `docs/design/13-backend-runplan-runtime-design.md`):

1. **`BackendVersion`** — version definition: default entrypoint, args template, parameter defs, health check, recommended images (per vendor)
2. **`BackendRuntime`** — user-editable run config for a specific vendor + Docker: actual image, devices, privileged, ipc, shm, ulimits, env, model mount
3. **`NodeRuntimeOverride`** — per-node image/env/device/modelRoot override for multi-server deployments

This split is needed because LightAI Go must handle NVIDIA + MetaX multi-vendor deployments and multi-server image/model-root differences, which GPUStack handles through runtime auto-detection (gpustack-runner).

### 8.3 Template Variable Substitution

GPUStack uses `{{model_path}}`, `{{port}}`, `{{worker_ip}}`, `{{model_name}}` in commands and `${VAR}` in env. LightAI Go already uses `${VAR_NAME}` style in `resolver.go`. This alignment should be maintained but unified to `{{VAR}}` for command templates.

### 8.4 State Machine

GPUStack's state machine is a good reference:
- PENDING → ANALYZING → SCHEDULED → INITIALIZING → DOWNLOADING → STARTING → RUNNING
- ERROR state with restart_on_error

LightAI Go can simplify this since we don't do automatic model downloading:
- PENDING → SCHEDULED → INITIALIZING → STARTING → RUNNING
- ERROR with optional restart

---

## 9. What LightAI Go Should NOT Copy

### 9.1 Runtime Detection and Auto-Configuration

GPUStack has extensive auto-detection:
- Detects GPU vendor/arch at runtime
- Queries gpustack-runner for compatible images
- Automatically injects backend parameters (tp, cache-report, etc.)
- Downloads models from HuggingFace/ModelScope

LightAI Go should NOT implement these (Phase 1 scope):
- Model download — users provide pre-downloaded paths
- Auto image detection — users configure backends explicitly
- Auto parameter injection — users configure parameters in templates
- gpustack-runner integration

### 9.2 Distributed Inference

GPUStack supports multi-node distributed inference (Ray, MP). LightAI Go Phase 1 is single-node only.

### 9.3 Complex Scoring

GPUStack's PlacementScorer and ModelFileLocalityScorer are overkill for single-node deployments.

### 9.4 Subprocess Model Serving

GPUStack launches model serving as a subprocess of the worker with `multiprocessing.Process`. LightAI Go will use Docker directly — the Agent receives a frozen `ResolvedRunPlan` and executes it via Docker CLI.

---

## 10. LightAI Go Should Stay Lightweight On

### 10.1 Scheduler

- Single-node: no worker filtering chain needed
- GPU selection: check vendor match, health, availability, VRAM, lease conflicts
- No distributed server coordination

### 10.2 Backend Management

- Built-in backends: vLLM, SGLang, Custom (no Ascend MindIE or VoxBox initially)
- Versions: user-configured, not auto-detected from runners
- Presets: config files under `configs/templates/` (already exists)

### 10.3 Model Definition

- No HuggingFace/ModelScope integration
- No automatic model type detection
- No pretrained config parsing
- Just: paths, metadata, backend reference

### 10.4 Instance Lifecycle

- Simpler state machine (no ANALYZING, no DOWNLOADING)
- No distributed servers
- No subordinate worker tracking
- Direct Docker via Agent

---

## 11. LightAI Go's Further Split Beyond GPUStack

GPUStack's `InferenceBackend` → `VersionConfig` design is a strong starting point. However, GPUStack's `VersionConfig` bundles image, run_command, entrypoint, and env into a single dict per version. For LightAI Go's needs (NVIDIA + MetaX, multi-server image differences), we split this further:

### 11.1 GPUStack VersionConfig vs LightAI Go Split

| GPUStack | LightAI Go | Rationale |
|----------|-----------|-----------|
| `InferenceBackend` | `InferenceBackend` (no vendor) | Backend family only — no vendor binding |
| `VersionConfig` (per version) | `BackendVersion` | Version definition: default entrypoint, args template, parameter defs, health check, recommended images |
| (in VersionConfig) | `BackendRuntimeTemplate` | System readonly template mapping version + vendor → Docker run config |
| (in VersionConfig) | `BackendRuntime` | User-editable run config: actual image, devices, privileged, ipc, shm, ulimits, env, model mount |
| (not in GPUStack) | `NodeRuntimeOverride` | Per-node image/env/device/modelRoot override for multi-server deployments |

### 11.2 Why LightAI Go Needs This Split

1. **Multi-vendor**: A single vLLM version (e.g., 0.8.5) runs on both NVIDIA and MetaX, but requires different Docker images and device mappings. GPUStack auto-detects this at runtime; LightAI Go pre-configures it in `BackendRuntime`.

2. **Multi-server image differences**: Different servers may have different local Docker images (e.g., `0d307f1665d3` vs `registry.local/metax/vllm:0.8.5`). GPUStack uses a `fallback_registry`; LightAI Go uses `NodeRuntimeOverride`.

3. **Model root path differences**: Different servers may store models at different paths (`/data/models`, `/mnt/models`, `/data/part2/MX-C500/model`). `NodeRuntimeOverride.model_root_host_path` handles this.

### 11.3 Current LightAI Go Old Code Issues

Current LightAI Go `runtime_environments` + `runtime_environment_docker_specs` conflates:
- Backend definition (vLLM vs SGLang)
- Version-specific config (image, command)
- Docker run config (devices, privileged, ipc, shm)
- Vendor differences (NVIDIA vs MetaX)

In the new design these are separated into `BackendVersion` + `BackendRuntime` + `NodeRuntimeOverride`.

---

## 12. Scope Decisions for LightAI Go

### 12.1 Container-First Only

LightAI Go is strictly container-first (Docker). vLLM / SGLang / llama.cpp command lines represent container-internal entrypoint/command/args. No host process runtime. No Kubernetes. This matches GPUStack's containerized approach.

### 12.2 Single-Node Multi-GPU Only (Phase 1)

LightAI Go v1 supports only single-node deployments with manual GPU selection. Multi-node scenarios are documented but not implemented:

- **Scenario A (supported)**: Single node, multi-GPU (e.g., tensor_parallel_size=4 on one node)
- **Scenario B (reserved)**: Multi-node multi-replica (independent copies on different nodes, load-balanced by future gateway)
- **Scenario C (reserved)**: Multi-node single-model distributed parallel (Ray/torchrun across nodes — complex, future)

### 12.3 Built-in Backends

Only three inference backends in v1: vLLM, SGLang, llama.cpp. Custom backend, MindIE, VoxBox are not implemented.

### 12.4 Template Syntax

Only `{{var}}` syntax (GPUStack style). `${VAR}` shell-style syntax is not supported to avoid ambiguity.

### 12.5 ResolvedRunPlan as Independent Table

Unlike GPUStack which builds the run plan at the worker, LightAI Go generates `ResolvedRunPlan` at the server and stores it in a dedicated `resolved_run_plans` table. Each start/restart creates a new immutable RunPlan row for audit trail.

---

## 13. Key Reference Points Summary

| Aspect | GPUStack Approach | LightAI Go Approach |
|--------|------------------|---------------------|
| Backend definition | `InferenceBackend` with VersionConfig dict | `InferenceBackend` (family) → `BackendVersion` (version) |
| Runtime config | In VersionConfig + auto-detection | `BackendRuntime` (vendor + Docker) + `NodeRuntimeOverride` (node-level) |
| Image resolution | Model > VersionConfig > auto-detect (gpustack-runner) | NodeOverride > BackendRuntime > BackendVersion.defaultImages |
| Command building | Backend-specific Python builders | Template-based in Go resolver (`{{var}}` only) |
| Run plan storage | Generated at worker, not persisted standalone | `resolved_run_plans` table — immutable, per start/restart |
| Parameter format | `space` or `equal` (auto-normalized) | Space only |
| State machine | 9 states with sub-worker tracking | 5-6 states, single-node only |
| Worker scheduling | Filter chain + resource fit + scoring | Manual node/GPU selection (Phase 1) |
| Health check | `/v1/models` for built-in, configurable | Per BackendVersion, overridable at BackendRuntime |
| Multi-vendor | Auto-detection via gpustack-runner | Explicit pre-configuration via BackendRuntime per vendor |
| Logs | Container log streaming + local persistence | Same, via Docker logs |
| Multi-node | Supported (Ray/MP) | Reserved (not implemented in Phase 1) |
