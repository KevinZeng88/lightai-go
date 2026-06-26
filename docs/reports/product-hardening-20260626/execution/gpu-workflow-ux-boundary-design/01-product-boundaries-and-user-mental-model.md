# 01 — Product Boundaries and User Mental Model

## 1. The product must expose three separate lines

LightAI Go has three related but distinct workflows:

```text
Model line      → where the model files are and what model they represent
Runtime line    → how a node can run models using a backend and GPU environment
Deployment line → combining one model with one ready runtime config to create a service
```

These lines share the concept of `Node`, but they must not be merged into one wizard or one mental model.

---

## 2. Model line

### User intent

The user wants to answer:

```text
Which model files exist on which node, and what model facts can the platform detect?
```

### System objects

```text
ModelArtifact  = model identity and facts
ModelLocation  = the model path on a specific node
```

### User-facing flow

```text
Model Library
→ Add / scan model
→ Select node
→ Browse that node's filesystem
→ Select model directory or GGUF file
→ Scan model
→ Confirm detected facts
→ Save ModelArtifact + ModelLocation
```

### User-facing fields

Show:

- Model display name
- Format: HuggingFace / safetensors / GGUF
- Task type: chat / embedding / rerank / completion
- Architecture, quantization, context length, parameter count when detected
- Node where the model path exists
- Absolute path
- Verification status
- Scan evidence and warnings

Hide from normal model UI:

- Docker image
- Docker args
- GPU devices
- Backend serve args
- `launcher.*`
- `runtime_env.*`
- `{{MODEL_CONTAINER_PATH}}`
- Port mappings
- Privileged/security options

### Design rule

Model pages describe model facts and model locations. They must not configure how the model is served.

---

## 3. Runtime line

### User intent

The user wants to answer:

```text
Can this node run models with this GPU/backend environment, and what runtime options should be used?
```

### System objects

```text
Backend             = inference backend family, such as vLLM, SGLang, llama.cpp
BackendVersion      = backend version/capability definition
BackendRuntime      = runtime template for vendor + backend + image/profile
NodeBackendRuntime  = a runtime template enabled and checked on a specific node
```

### User-facing flow

```text
Runtime Templates
→ Choose a small set of vendor/backend runtime options

Node Runtime Configs
→ New config
→ Select node
→ Select runtime template
→ Name the node runtime config
→ Configure image and common runtime parameters
→ Save and check
→ Review readiness result
```

### Runtime Template user-facing naming

Use:

```text
<gpu_vendor>.<backend> [backend_version]
```

Examples:

```text
nvidia.vllm
nvidia.sglang
nvidia.llama.cpp b9700
metax.vllm
metax.sglang
```

### User-facing runtime parameters

Group normal settings into:

#### Basic

- Config name
- Docker image
- Shared memory (`shm_size`), e.g. `16gb`
- Container memory limit, optional
- CPU limit, optional
- Health check timeout

#### GPU

- GPU visibility mode: all / selected devices
- Selected GPU devices
- GPU count if applicable
- Vendor runtime mode if applicable

#### Backend common parameters

- Served model name
- Context length
- Max batch / max sequences when applicable

#### vLLM

- `gpu-memory-utilization`
- `max-model-len`
- `max-num-seqs`
- `max-num-batched-tokens`
- `kv-cache-dtype`
- `dtype`

#### SGLang

- `mem-fraction-static`
- `context-length`
- `max-running-requests`
- `chunked-prefill-size`
- `attention-backend`

#### llama.cpp

- `ctx-size`
- `n-gpu-layers`
- `batch-size`
- `ubatch-size`
- `threads`
- `cache-type-k`
- `cache-type-v`

#### Advanced

- Extra environment variables
- Extra volume mappings
- Extra ports
- Extra backend args
- Security / privileged options

### Internal fields hidden from normal UI

Hide from normal runtime editing:

```text
launcher.command
launcher.args
launcher.*
runtime_env.*
internal.*
resolver.*
source_metadata
{{MODEL_CONTAINER_PATH}}
{{MODEL_CONTAINER_DIR}}
raw ConfigSet item codes
```

These may appear only under:

```text
Advanced Diagnostics
RunPlan Details
Raw JSON
```

### Design rule

Runtime pages configure how a node runs models. They should not expose resolver internals or command template placeholders as ordinary user fields.

---

## 4. Deployment line

### User intent

The user wants to answer:

```text
Run this model with this ready node runtime config, expose a service, and verify the final run plan before starting.
```

### System objects

```text
ModelDeployment  = saved deployment definition
ResolvedRunPlan  = final resolved runtime plan
ModelInstance    = actual running/stopped/failed instance
```

### User-facing flow

```text
Model Deployments
→ New deployment
→ Select model
→ Select ready or ready_with_warnings node runtime config
→ Configure service port and deployment overrides
→ Preview Run Plan
→ Save or start
```

### Required deployment compatibility checks

The deployment flow must verify:

- The selected model has a location on the same node as the selected NodeBackendRuntime.
- NodeBackendRuntime status is `ready` or `ready_with_warnings`.
- Model format is compatible with backend capability.
- Model task type is compatible with backend capability.
- Deployment preview uses the same resolver path as start.
- Final payload uses `node_backend_runtime_id` only.

### Design rule

Deployment is where the Model line and Runtime line meet. Before deployment, they remain separate.

---

## 5. Shared component rule

Use shared selectors for consistency, but do not merge business flows.

Recommended shared components:

```text
NodeSelectorTable
RuntimeTemplateSelector
NodeRuntimeSelector
ModelArtifactSelector
HumanRuntimeParameterForm
AdvancedDiagnosticsPanel
```

The same `NodeSelectorTable` can be used by both Model Library and Node Runtime Configs, but its label and context must differ:

```text
Model Library: select the node where model files exist
Node Runtime Configs: select the node where the runtime environment will be enabled
```

