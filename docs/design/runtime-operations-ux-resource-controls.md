# Runtime Operations UX & Resource Controls — Design

> Date: 2026-06-23  
> Status: Design proposal for review. Do not implement before review approval.  
> Scope: LightAI Go runtime operations, model instance UX, RunPlan diagnostics, resource controls, and testability.

## 1. Purpose

This design defines a runtime operations quality layer for LightAI Go. The goal is not to patch individual UI/log issues one by one, but to establish a repeatable mechanism for:

- detecting RunPlan argument conflicts before container start;
- classifying runtime logs after container start;
- modeling GPU/resource controls according to each backend's real semantics;
- allowing shared-GPU deployments with explicit admission rules;
- improving model instance status freshness without manual browser refresh;
- making diagnostic JSON readable and copyable;
- unifying configuration editor layouts around structured fields, preview, lint, and diagnostics;
- adding tests that catch these classes of issues before manual verification.

## 2. Source issues motivating this design

Observed issues:

1. SGLang container logs include:
   - `torchao/quantization/quant_api.py:1731: SyntaxWarning: invalid escape sequence '\.'`
   - `Attention backend not specified. Use flashinfer backend by default.`
2. Model instance page status does not automatically update; manual refresh is required.
3. Model test result page has an "Advanced diagnostic JSON" section whose copied content is complete, but the on-page display is truncated or hard to read.
4. Runtime configuration page exposes "Health Check JSON" and "Advanced Diagnostic JSON" in a way that makes users unclear whether those are inputs, previews, or generated evidence.
5. Runtime configuration and model deployment pages should follow the "Copy as user configuration" layout: easy to scan, easy to edit, and clearly separated into common fields, advanced fields, preview, and diagnostics.
6. llama.cpp container logs include:
   - `warn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host`
7. GPU memory limit/resource control settings are not clearly exposed, and the platform needs to clarify whether multiple Docker containers can share one GPU.

## 3. Official backend semantics used by this design

The platform must not invent a uniform GPU memory limit where the backend does not provide one.

### 3.1 Docker/NVIDIA GPU binding

NVIDIA Container Toolkit supports exposing GPUs to containers using Docker `--gpus` or `NVIDIA_VISIBLE_DEVICES`. This controls device visibility, not a hard VRAM partition by itself.

Design consequence:

- GPU binding means "which GPU devices are visible".
- It must not be presented as "VRAM hard isolation".
- Shared-GPU deployments need LightAI Go admission checks plus backend memory-budget parameters where available.
- For hard isolation, technologies such as MIG/vGPU/HAMi-style partitioning are separate capabilities and should not be implied by Docker `--gpus`.

Reference:
- NVIDIA Container Toolkit, Specialized Configurations with Docker: https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/docker-specialized.html

### 3.2 vLLM

vLLM exposes `--gpu-memory-utilization`, the fraction of GPU memory to be used for the model executor. It can be used as a per-instance memory budget knob when multiple vLLM instances share a GPU, although it is still a backend allocation budget, not Docker-level hard isolation.

Relevant controls to expose:

- `--gpu-memory-utilization`
- `--max-model-len`
- `--max-num-seqs`
- `--max-num-batched-tokens`
- KV cache related controls where supported by the version

References:
- vLLM engine arguments: https://docs.vllm.ai/en/v0.6.4/models/engine_args.html
- vLLM optimization guide: https://docs.vllm.ai/en/stable/configuration/optimization/

### 3.3 SGLang

SGLang exposes `--mem-fraction-static`, described as the fraction of memory used for static allocation, including model weights and KV cache memory pool. Its tuning guidance recommends adjusting the value based on available GPU memory and OOM risk.

Relevant controls to expose:

- `--mem-fraction-static`
- `--max-running-requests`
- `--max-total-tokens` where supported by the version
- `--chunked-prefill-size`
- `--kv-cache-dtype`
- `--attention-backend`

References:
- SGLang server arguments: https://docs.sglang.ai/advanced_features/server_arguments.html
- SGLang hyperparameter tuning: https://sgl-project.github.io/advanced_features/hyperparameter_tuning.html

### 3.4 llama.cpp

llama.cpp does not provide a vLLM/SGLang-style GPU memory fraction knob. GPU memory usage is controlled indirectly by model offload, context, batch, KV cache, and split settings.

Relevant controls to expose:

- `--n-gpu-layers` / `--gpu-layers`
- `--ctx-size`
- `--batch-size`
- `--ubatch-size`
- `--cache-type-k`
- `--cache-type-v`
- `--split-mode`
- `--main-gpu`
- `--tensor-split`
- `--host`, `--port` as platform-owned serving parameters

Important warning:

- Do not show a fake "GPU memory fraction" for llama.cpp.
- Do not set both `LLAMA_ARG_HOST` and `--host`, or both `LLAMA_ARG_PORT` and `--port`.

Reference:
- llama.cpp server README: https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md

## 4. Design principles

1. **Backend semantics must be truthful.** vLLM, SGLang, and llama.cpp expose different resource controls. The UI and RunPlan must reflect that.
2. **Docker device visibility is not memory isolation.** Do not imply shared GPU safety unless admission checks and backend budgets support it.
3. **RunPlan must be linted before execution.** Duplicate flags, env/CLI conflicts, and high-risk settings should be visible before start.
4. **Generated diagnostics should be read-only by default.** Users configure health-check rules, not system-generated diagnostic JSON.
5. **Configuration editing should be structured.** Raw JSON is acceptable only in expert mode with schema validation.
6. **Known warning patterns must be classified.** Users should not manually read Docker logs to decide whether a warning is fatal.
7. **Status pages should reflect current state.** Use state-sensitive polling before adding WebSocket complexity.
8. **Tests should capture real failure signatures.** Every observed warning/conflict in this batch should become a fixture test.
9. **No old compatibility clutter.** If old config/template paths are unused, they should be removed rather than preserved as fallback.

## 5. Architecture overview

```text
BackendVersion catalog / seed
    ├── resource_controls JSON
    ├── parameter ownership specs
    └── log rule metadata, initially Go-built-in

RunPlan resolver
    ├── produces DockerSpec / RunPlan
    ├── injects backend-specific resource args
    ├── runs RunPlan lint
    └── returns lint + equivalent command + resource admission

Runtime log reader / model test result
    ├── reads raw Docker logs
    ├── applies RuntimeLogClassifier
    └── returns classified_log_events

Frontend
    ├── ConfigEditorLayout
    ├── RunPlanPreviewPanel
    ├── JsonViewer
    ├── useInstanceStatusPolling
    └── resource/admission warnings
```

## 6. Runtime log classifier

### 6.1 Purpose

Classify known runtime log patterns into user-actionable categories. This prevents non-fatal dependency warnings from being confused with instance failures.

### 6.2 Types

```go
type RuntimeLogRule struct {
    ID          string
    Backend    string // vllm, sglang, llamacpp, ollama, *
    Version    string // optional
    Pattern    string
    Severity   string // fatal, error, warning, advisory, noise
    Category   string // dependency_warning, default_selection, arg_conflict, oom, health, startup
    Message    string
    Suggestion string
}

type RuntimeLogEvent struct {
    RuleID      string
    Severity    string
    Category    string
    Message     string
    Suggestion  string
    RawLine      string
    Occurrences int
}
```

### 6.3 Initial built-in rules

| Rule ID | Backend | Pattern | Severity | Category | User-facing result |
|---|---|---|---|---|---|
| `sglang.torchao.syntax_warning` | sglang | `torchao/quantization/.*SyntaxWarning: invalid escape sequence` | `noise` or `advisory` | dependency_warning | Upstream dependency warning; does not affect status |
| `sglang.attention_backend.default` | sglang | `Attention backend not specified. Use flashinfer backend by default` | `advisory` | default_selection | SGLang used default attention backend |
| `llamacpp.env_overwritten.host` | llamacpp | `LLAMA_ARG_HOST environment variable is set.*overwritten.*--host` | `warning` | arg_conflict | Remove env or CLI duplicate |
| `llamacpp.env_overwritten.port` | llamacpp | `LLAMA_ARG_PORT environment variable is set.*overwritten.*--port` | `warning` | arg_conflict | Remove env or CLI duplicate |
| `cuda.oom` | `*` | `CUDA out of memory|out of memory` | `error` | oom | Reduce model size, context, batch, or memory budget |
| `container.startup.failed` | `*` | `Traceback|panic:|fatal error|failed to start` | `error` or `fatal` | startup | Instance likely failed |

### 6.4 Integration

First iteration:

- Keep rules in Go.
- Classify logs when model test results or Docker logs are read.
- Return `classified_log_events` in API responses.
- Do not change instance state solely because of `noise` or `advisory`.
- `error`/`fatal` rules can annotate failure diagnostics, but state transition should still follow the existing health/lifecycle model.

## 7. RunPlan lint

### 7.1 Purpose

RunPlan lint checks the final resolved command before execution. It prevents hidden conflicts such as llama.cpp env vars being overwritten by CLI flags.

### 7.2 Types

```go
type RunPlanLintResult struct {
    Status   string // ok, warning, error
    Findings []RunPlanLintFinding
}

type RunPlanLintFinding struct {
    ID          string
    Severity    string // error, warning, advisory
    Category    string // arg_conflict, duplicate_arg, resource_budget, high_risk, unsupported
    Message     string
    Suggestion  string
    FieldPath   string
    Sources     []string // platform, template, user_extra_args, user_env, backend_default
}
```

### 7.3 Required lint checks

| Finding ID | Description | Default severity |
|---|---|---|
| `arg.duplicate` | Same CLI flag appears more than once | error |
| `arg.env_cli_conflict` | Env var and CLI flag express the same logical parameter | error |
| `arg.platform_overridden` | User extra args override platform-owned parameters | error |
| `arg.unsupported_backend_param` | Arg is not supported for current backend/version | warning or error |
| `resource.gpu_budget_exceeded` | Shared GPU declared budget exceeds threshold | error |
| `resource.gpu_budget_unknown` | Shared GPU includes an unknown-budget instance | warning |
| `security.privileged_enabled` | Privileged/high-risk container option enabled | warning |
| `port.conflict` | Host port conflict detected | error |
| `mount.conflict` | Multiple mounts conflict on target path | error |

### 7.4 Parameter ownership

```go
type LogicalParamSpec struct {
    Name      string
    CLIFlags  []string
    EnvVars   []string
    Owner     string // platform, user, backend_default
    Conflict  string // reject, warn, platform_wins, user_wins
}
```

Examples:

llama.cpp:

```json
{
  "host": {
    "cli_flags": ["--host"],
    "env_vars": ["LLAMA_ARG_HOST"],
    "owner": "platform",
    "conflict": "reject"
  },
  "port": {
    "cli_flags": ["--port"],
    "env_vars": ["LLAMA_ARG_PORT"],
    "owner": "platform",
    "conflict": "reject"
  }
}
```

vLLM:

```json
{
  "host": {"cli_flags": ["--host"], "owner": "platform", "conflict": "reject"},
  "port": {"cli_flags": ["--port"], "owner": "platform", "conflict": "reject"},
  "model_path": {"cli_flags": ["--model"], "owner": "platform", "conflict": "reject"},
  "gpu_memory_fraction": {"cli_flags": ["--gpu-memory-utilization"], "owner": "user", "conflict": "reject"}
}
```

SGLang:

```json
{
  "host": {"cli_flags": ["--host"], "owner": "platform", "conflict": "reject"},
  "port": {"cli_flags": ["--port"], "owner": "platform", "conflict": "reject"},
  "model_path": {"cli_flags": ["--model-path"], "owner": "platform", "conflict": "reject"},
  "gpu_memory_fraction": {"cli_flags": ["--mem-fraction-static"], "owner": "user", "conflict": "reject"}
}
```

## 8. Resource controls

### 8.1 Platform resource policy

Proposed logical policy:

```json
{
  "gpu_policy": {
    "placement": "exclusive | shared",
    "gpu_ids": ["0"],
    "memory_budget_mode": "none | fraction | mib | backend_specific",
    "memory_fraction": 0.5,
    "memory_mib": null,
    "allow_oversubscribe": false,
    "oversubscribe_reason": ""
  }
}
```

### 8.2 Backend mapping

| Backend | Platform field | Backend flag | Notes |
|---|---|---|---|
| vLLM | `memory_fraction` | `--gpu-memory-utilization` | Per-instance backend allocation budget |
| SGLang | `memory_fraction` | `--mem-fraction-static` | Static allocation for weights and KV pool |
| llama.cpp | no memory_fraction | `--n-gpu-layers`, `--ctx-size`, `--batch-size`, `--ubatch-size`, `--cache-type-k/v` | Indirect control only |
| Ollama | not supported in this batch | none | Mark as unsupported for platform budget control |

### 8.3 BackendVersion resource_controls JSON

Start in catalog/seed JSON to avoid schema churn. Promote to explicit DB columns only if UI/querying demands it later.

vLLM example:

```json
{
  "resource_controls": {
    "gpu_memory_fraction": {
      "supported": true,
      "arg": "--gpu-memory-utilization",
      "min": 0.1,
      "max": 0.95,
      "default": 0.9,
      "semantics": "per_instance_backend_allocation_budget"
    },
    "max_model_len": {"arg": "--max-model-len", "type": "int"},
    "max_num_seqs": {"arg": "--max-num-seqs", "type": "int"},
    "max_num_batched_tokens": {"arg": "--max-num-batched-tokens", "type": "int"}
  }
}
```

SGLang example:

```json
{
  "resource_controls": {
    "gpu_memory_fraction": {
      "supported": true,
      "arg": "--mem-fraction-static",
      "min": 0.1,
      "max": 0.95,
      "default": 0.9,
      "semantics": "static_weights_and_kv_pool"
    },
    "max_running_requests": {"arg": "--max-running-requests", "type": "int"},
    "chunked_prefill_size": {"arg": "--chunked-prefill-size", "type": "int"},
    "attention_backend": {
      "arg": "--attention-backend",
      "type": "enum",
      "values": ["auto", "flashinfer", "triton", "fa3"]
    }
  }
}
```

llama.cpp example:

```json
{
  "resource_controls": {
    "gpu_memory_fraction": {
      "supported": false,
      "reason": "llama.cpp does not expose a vLLM-style GPU memory fraction. Use gpu_layers/context/batch/cache controls."
    },
    "gpu_layers": {"arg": "--n-gpu-layers", "type": "string_or_int", "values_hint": ["auto", "all"]},
    "ctx_size": {"arg": "--ctx-size", "type": "int"},
    "batch_size": {"arg": "--batch-size", "type": "int"},
    "ubatch_size": {"arg": "--ubatch-size", "type": "int"},
    "cache_type_k": {"arg": "--cache-type-k", "type": "enum"},
    "cache_type_v": {"arg": "--cache-type-v", "type": "enum"},
    "split_mode": {"arg": "--split-mode", "type": "enum"},
    "main_gpu": {"arg": "--main-gpu", "type": "int"},
    "tensor_split": {"arg": "--tensor-split", "type": "string"}
  }
}
```

## 9. Shared GPU admission

### 9.1 Policy

Multiple Docker containers can share one GPU if the same GPU is visible to each container. LightAI Go should allow this, but not silently.

Admission rules:

1. If an existing instance on a GPU is `exclusive`, block new placements on that GPU.
2. If a requested placement is `exclusive`, block if any non-terminal instance already uses that GPU.
3. For vLLM/SGLang with declared `memory_fraction`, include the value in budget sum.
4. If total declared budget exceeds threshold, default block.
5. If an instance has unknown budget, show warning.
6. If `allow_oversubscribe=true`, allow with warning and audit event.
7. llama.cpp should usually be `unknown_budget` unless a future estimator is implemented.

### 9.2 Types

```go
type GPUAdmissionInput struct {
    NodeID             string
    GPUIDs             []string
    RequestedPlacement string // exclusive, shared
    RequestedBudget    *MemoryBudget
    ExistingInstances  []GPUInstanceBudget
    AllowOversubscribe bool
}

type MemoryBudget struct {
    Mode     string // fraction, mib, unknown
    Fraction *float64
    MiB      *int64
    Source   string // vllm, sglang, llamacpp, manual
}

type GPUAdmissionResult struct {
    Status   string // ok, warning, blocked
    Findings []RunPlanLintFinding
}
```

## 10. Health check and diagnostics boundaries

### 10.1 User-configurable health fields

Users configure structured health-check rules only:

```text
path
method
timeout_seconds
interval_seconds
expected_status
expected_body_contains
readiness_grace_seconds
```

### 10.2 Read-only generated evidence

The following are read-only by default:

- health result JSON
- advanced diagnostic JSON
- RunPlan JSON
- Docker inspect JSON
- preflight evidence JSON
- classified log event JSON

Raw JSON editing is allowed only in expert mode and must have schema validation.

## 11. Frontend UX design

### 11.1 Instance status polling

Create:

```text
web/src/composables/useAutoRefresh.ts or .js
web/src/composables/useInstanceStatusPolling.ts or .js
```

Behavior:

| State | Interval |
|---|---|
| pending / scheduled / starting / loading / stopping | 2–3 seconds |
| running / failed / stopped | 15–30 seconds |
| document hidden | pause or 60 seconds |
| user refresh | immediate |

UI:

- show last refreshed time;
- preserve manual refresh;
- show stale-data warning on request failure;
- avoid noisy backend logs.

### 11.2 JsonViewer

Create:

```text
web/src/components/common/JsonViewer.vue
```

Props:

```ts
{
  value: unknown | string
  title?: string
  defaultExpanded?: boolean
  maxHeight?: string
  readonly?: boolean
  allowDownload?: boolean
  allowCopy?: boolean
  allowFullscreen?: boolean
}
```

Required features:

- scroll;
- fullscreen expand;
- copy full content;
- download;
- search;
- line wrap toggle;
- long string handling;
- malformed JSON fallback to raw text.

### 11.3 ConfigEditorLayout

Create:

```text
web/src/components/config/ConfigEditorLayout.vue
web/src/components/config/ConfigSection.vue
web/src/components/config/AdvancedSection.vue
web/src/components/config/RunPlanPreviewPanel.vue
web/src/components/config/DiffFromBase.vue
```

Layout:

```text
Top summary:
  name / source / backend / version / node / system-template or user-copy

Main structured editor:
  basic
  model path or model name
  ports
  GPU/resource controls
  health check
  security/container
  advanced args/env/volumes/devices

Preview panel:
  RunPlan summary
  equivalent Docker command
  lint warnings/errors
  resource admission
  diff from base

Advanced area:
  raw JSON
  diagnostics
  evidence
```

## 12. API design

### 12.1 RunPlan preview/preflight response

Add:

```json
{
  "lint": {
    "status": "ok | warning | error",
    "findings": []
  },
  "resource_admission": {
    "status": "ok | warning | blocked",
    "findings": []
  }
}
```

### 12.2 Status summary endpoint

Optional but recommended:

```text
GET /api/v1/model-instances/status-summary
```

Response:

```json
{
  "items": [
    {
      "id": "...",
      "state": "running",
      "health": "healthy",
      "node_id": "...",
      "gpu_ids": ["0"],
      "updated_at": "...",
      "last_error": null
    }
  ],
  "server_time": "..."
}
```

### 12.3 Model test result diagnostics

Add:

```json
{
  "classified_log_events": [
    {
      "rule_id": "sglang.attention_backend.default",
      "severity": "advisory",
      "category": "default_selection",
      "message": "...",
      "suggestion": "...",
      "occurrences": 1
    }
  ]
}
```

## 13. Deferred items

Do not implement in this batch unless the codebase proves it is required:

- full arg abstraction layer;
- full RuntimeRequirements structure;
- DB-schema promotion of `resource_controls`;
- WebSocket-based instance updates;
- real GPU-heavy E2E as mandatory default;
- llama.cpp VRAM estimator;
- MIG/vGPU/HAMi integration;
- config-driven RuntimeLogRule database.

## 14. Acceptance criteria

Functional:

1. SGLang warning samples are classified and no longer require manual interpretation.
2. llama.cpp env/CLI conflicts are caught before start or avoided by RunPlan generation.
3. Model instance status refreshes without manual browser refresh.
4. Advanced diagnostic JSON is readable, scrollable, expandable, and fully copyable.
5. Health check configuration no longer mixes with generated diagnostics.
6. vLLM/SGLang expose real memory budget fields.
7. llama.cpp does not expose a fake memory-fraction control.
8. Shared GPU deployment has warning/block/override/audit semantics.
9. Docker command preview shows final resource parameters and shared-GPU risks.

Testing:

1. Current SGLang and llama.cpp warning samples are fixture tests.
2. RunPlan arg conflicts have Go tests.
3. Resource admission has Go tests.
4. JsonViewer has frontend tests.
5. Instance polling has frontend tests.
6. Dry-run command tests detect duplicate/conflicting args.
7. `go test ./...`, `go build ./...`, `gofmt -l internal/`, frontend tests, frontend build, and `git diff --check` pass.

Documentation:

1. Backend-specific resource semantics documented.
2. Docker GPU visibility versus memory isolation documented.
3. Shared GPU admission documented.
4. Health-check config versus diagnostic JSON boundaries documented.
5. Test-discovery strategy documented.
