# 05 — Workstream C: Runtime Parameter Completeness

## Goal

Make runtime parameter editing complete, layered, backend-correct, and verifiable.

## Required parameter flow

```text
BackendVersion default schema
  -> BackendRuntime template values
  -> NodeBackendRuntime values
  -> Deployment override values
  -> Resolved RunPlan
  -> Docker command/spec
```

Priority:

```text
Deployment override > NodeBackendRuntime > BackendRuntime > BackendVersion defaults
```

## Step C1 — Inventory existing schema and editor code

Run:

```bash
rg -n "default_args_schema|resource_controls|parameter_values|parameter_schema|RuntimeParameterEditor|config_overrides|ConfigSet|vendor_options|gpu_memory|mem-fraction|ctx-size|n-gpu-layers|extra_args|extra_env|extra_docker" configs web internal docs
```

Inspect:

```text
configs/backend-catalog/versions/vllm/*.yaml
configs/backend-catalog/versions/sglang/*.yaml
configs/backend-catalog/versions/llamacpp/*.yaml
configs/backend-catalog/help/**
web/src/components/**
web/src/pages/BackendRuntimesPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/pages/ModelDeploymentsPage.vue
internal/server/catalog/**
internal/server/runplan/**
internal/server/api/*backend*
internal/server/api/*deployment*
```

Create:

```text
docs/reports/product-hardening-20260626/execution/runtime-parameter-inventory.md
```

For each backend:

| Backend | Param | Layer | Required | Optional | Default | Editable | CLI/env/docker target | Current UI support | Missing |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |

## Step C2 — Normalize schema contract

Each parameter schema item should support, at minimum:

```json
{
  "key": "backend.vllm.gpu_memory_utilization",
  "name": "--gpu-memory-utilization",
  "label": "GPU Memory Utilization",
  "group": "performance",
  "type": "float",
  "required": false,
  "enabled": false,
  "default": 0.9,
  "value": 0.85,
  "user_editable": true,
  "cli_flag": "--gpu-memory-utilization",
  "backend": "vllm",
  "vendor": "any",
  "layer": "deployment",
  "advanced": false,
  "risk_level": "medium",
  "allow_override": true,
  "help_ref": "vllm.gpu_memory_utilization",
  "validation": {
    "min": 0.1,
    "max": 0.95
  }
}
```

Do not over-engineer DB columns unless needed. ConfigSet JSON is acceptable if:

- API contract is stable;
- tests cover read/write/merge;
- RunPlan behavior is deterministic.

## Step C3 — Backend-specific required coverage

### vLLM

Required editable optional controls:

- `--served-model-name`
- `--tensor-parallel-size`
- `--pipeline-parallel-size`
- `--max-model-len`
- `--gpu-memory-utilization`
- `--max-num-seqs`
- `--max-num-batched-tokens`
- `--kv-cache-dtype`
- `--dtype`
- `--swap-space`
- `--cpu-offload-gb`
- `--trust-remote-code`
- `--download-dir`
- `extra_args`

Required locked/generated controls:

- `--model`
- `--host`
- `--port`

### SGLang

Required editable optional controls:

- `--served-model-name`
- `--tp`
- `--tensor-parallel-size`
- `--dp`
- `--enable-metrics`
- `--log-level`
- `--mem-fraction-static`
- `--context-length`
- `--max-running-requests`
- `--chunked-prefill-size`
- `--attention-backend`
- `--disable-cuda-graph`
- `extra_args`

Required locked/generated controls:

- `--model-path`
- `--host`
- `--port`

### llama.cpp

Required editable optional controls:

- `--ctx-size`
- `--n-gpu-layers`
- `--threads`
- `--threads-batch`
- `--batch-size`
- `--ubatch-size`
- `--cache-type-k`
- `--cache-type-v`
- `--split-mode`
- `--main-gpu`
- `--tensor-split`
- `extra_args`

Required locked/generated controls:

- `-m` / `--model`
- `--host`
- `--port`

Do not show fake GPU memory fraction for llama.cpp.

## Step C4 — RuntimeParameterEditor behavior

Implement or update:

```text
web/src/components/runtime/RuntimeParameterEditor.vue
```

or current equivalent.

Required props:

```ts
{
  schema: ParameterSchema[]
  values: ParameterValue[]
  layer: 'backend_runtime' | 'node_backend_runtime' | 'deployment'
  baseValues?: ParameterValue[]
  readonly?: boolean
  showSource?: boolean
  showAdvanced?: boolean
}
```

Required emits:

```ts
update:values
validate
```

Required behavior:

- render required fields as locked enabled;
- optional fields show enable checkbox and input;
- disabled optional field input remains visible and retains value;
- disabled optional field not applied to final RunPlan;
- validation errors shown inline;
- advanced groups collapsible;
- help text accessible per parameter;
- source/diff from base layer visible;
- no watch -> emit infinite loop;
- no mutation of props in place.

## Step C5 — BackendRuntime page

Modify:

```text
web/src/pages/BackendRuntimesPage.vue
```

Required UI:

- list runtime templates;
- detail drawer shows structured editor for editable user-managed templates;
- system templates are read-only;
- clone system template to user-managed template;
- save updates ConfigSet;
- preview effective default args/resource controls;
- show backend/version/vendor/image/source.

Backend API:

- `GET /api/v1/backend-runtimes/{id}` returns schema and values.
- `PATCH /api/v1/backend-runtimes/{id}` saves parameter values safely.
- System-managed runtime is read-only or requires clone.

## Step C6 — Node runtime config page

Modify:

```text
web/src/pages/RunnerConfigsPage.vue
```

Required UI:

- rename as Node Runtime Configs;
- creation: node + runtime template + image override;
- detail: structured editor for node-specific values;
- check/probe action;
- status summary;
- image inspect evidence;
- deployable state;
- source diff from runtime template;
- RunPlan-like command preview for a sample model if available, or config preview.

Backend API:

- `PATCH /api/v1/nodes/{id}/backend-runtimes/{nbr_id}` saves node-specific parameter values.
- `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request` remains authoritative check.
- Client cannot set ready state from evidence.

## Step C7 — Deployment override

Modify:

```text
web/src/pages/ModelDeploymentsPage.vue
```

Required:

- deployment override editor uses same RuntimeParameterEditor;
- source values come from selected NBR;
- changed values shown as override;
- disabled override value retained but not applied;
- final preview shows effective result.

## Step C8 — Model page cleanup

Modify:

```text
web/src/pages/ModelArtifactsPage.vue
```

Remove or refactor runtime-serving parameter textarea.

Allowed model hints:

- recommended context length;
- task type;
- capabilities;
- architecture;
- quantization;
- tokenizer/chat template metadata;
- model-level notes.

Forbidden in ModelArtifact UI:

- `--max-model-len`;
- `--served-model-name`;
- `--gpu-memory-utilization`;
- Docker args;
- env;
- devices;
- host ports;
- backend runtime security options.

## Step C9 — RunPlan and lint

Add/verify lint checks:

- duplicate CLI flag;
- env/CLI conflict;
- user extra arg overrides platform-owned arg;
- unsupported backend param;
- disabled field applied;
- missing required field;
- vendor-incompatible field.

RunPlan must return or store enough source trace for preview/debug.

## Tests

Go tests:

```bash
go test ./internal/server/runplan/... 
go test ./internal/server/api/... -run 'Parameter|ConfigSet|BackendRuntime|NodeBackendRuntime|Deployment'
```

Frontend tests:

```text
web/tests/runtimeParameterEditor.test.mjs
web/tests/modelCapabilities.test.mjs
web/tests/runtimeBoundaryUi.test.mjs
```

Required assertions:

- required fields cannot be disabled;
- disabled optional values are preserved;
- disabled optional values do not enter final args;
- deployment override wins;
- NBR override wins over BackendRuntime;
- vLLM memory fraction renders and validates;
- SGLang memory fraction renders and validates;
- llama.cpp does not render memory fraction;
- model page does not expose runtime args.

## Acceptance

- vLLM/SGLang/llama.cpp parameters are editable at intended layers.
- Final RunPlan is deterministic and source-traceable.
- UI can explain inherited vs overridden values.
- Model layer is clean.
- Tests pass.
