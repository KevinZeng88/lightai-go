# 01 — Current Code Findings to Verify and Use

This document records the starting findings and turns them into concrete verification tasks. Claude must verify every item locally before implementation.

## F-001: Runtime concepts are exposed with inconsistent user wording

### Evidence to verify

Run:

```bash
rg -n "RunnerConfig|runner-config|runnerConfigs|NodeBackendRuntime|backend runtime|BackendRuntime|runtime template|RunPlan|ConfigSet|node_backend_runtime|source_node_backend_runtime_id" web internal docs configs
```

Inspect:

```text
web/src/router/index.ts
web/src/layouts/ConsoleLayout.vue
web/src/pages/BackendRuntimesPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/pages/ModelDeploymentsPage.vue
web/src/pages/ModelInstancesPage.vue
web/src/locales/zh-CN.*
web/src/locales/en-US.*
web/tests/*
docs/**/*.md
```

### Concrete problem

The current product has these layers:

```text
Backend
BackendVersion
BackendRuntime
NodeBackendRuntime
ModelDeployment
ModelInstance
ResolvedRunPlan
ConfigSet
```

But UI may show terms such as "runtime", "runner config", "runtime config", and "backend runtime" without clearly distinguishing template vs node-specific deployable config.

### Required fix

Create a single vocabulary and apply it everywhere user-facing.

Target wording:

| Internal | zh-CN UI | en-US UI | Where visible |
| --- | --- | --- | --- |
| Backend | 推理后端 | Backend | menu/page/table |
| BackendVersion | 后端版本 | Backend Version | backend detail/version selector |
| BackendRuntime | 运行模板 | Runtime Template | `/runtimes` page |
| NodeBackendRuntime | 节点运行配置 | Node Runtime Config | `/runner-configs` page, deployment selector |
| ModelArtifact | 模型 | Model | model library |
| ModelLocation | 模型位置 | Model Location | model detail |
| ModelDeployment | 模型部署 | Deployment | deployment page |
| ModelInstance | 模型实例 | Instance | instance page |
| ResolvedRunPlan | 运行计划 | Run Plan | preview/detail |
| ConfigSet | 配置集 | ConfigSet | technical JSON label only |

Do not expose "NBR" as the primary UI label. It can appear in technical details.

## F-002: Deployment creation UI is not a safe product workflow

### Evidence to verify

Inspect:

```text
web/src/pages/ModelDeploymentsPage.vue
web/src/api/deployments.ts
internal/server/api/router.go
internal/server/api/*deployment*
internal/server/runplan/*
docs/api/openapi.yaml
```

Run:

```bash
rg -n "HandleCreateDeployment|HandlePreflightDeployments|HandleDeploymentDryRun|HandleStartDeployment|dryRunDeployment|preflightDeployment|createDeployment|node_backend_runtime_id|backend_runtime_id|service_json|config_overrides" internal web docs
```

### Concrete problem

Deployment create must not be just a few inputs. It must show:

- selected model facts;
- selected node runtime config status;
- deployment overrides;
- final RunPlan preview;
- equivalent Docker command;
- lint/preflight/resource findings;
- start blockers before save/start.

### Required fix

Implement a guided create/edit workflow or a structured single-page wizard with these sections:

1. Select model.
2. Select node runtime config.
3. Configure service and resources.
4. Configure deployment overrides.
5. Preview final RunPlan.
6. Save or save-and-start.

## F-003: Runtime parameter editing exists in design/catalog but is not fully surfaced

### Evidence to verify

Inspect:

```text
configs/backend-catalog/versions/vllm/*.yaml
configs/backend-catalog/versions/sglang/*.yaml
configs/backend-catalog/versions/llamacpp/*.yaml
configs/backend-catalog/help/**
web/src/components/**/*Parameter*
web/src/pages/BackendRuntimesPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/pages/ModelDeploymentsPage.vue
internal/server/catalog/*
internal/server/runplan/*
```

Run:

```bash
rg -n "default_args_schema|resource_controls|parameter_values|parameter_schema|RuntimeParameterEditor|config_overrides|ConfigSet|vendor_options|gpu_memory|mem-fraction|ctx-size|n-gpu-layers" configs web internal docs
```

### Concrete problem

The product needs the same structured parameter semantics across:

```text
BackendVersion defaults
BackendRuntime template
NodeBackendRuntime node-specific config
Deployment override
RunPlan final args/env/docker spec
```

### Required fix

Implement or complete common RuntimeParameterEditor and resolver integration:

- required fields locked-on;
- optional fields have enabled/value;
- disabled values saved but not applied;
- help text visible;
- backend/vendor applicability enforced;
- duplicate/conflicting args rejected;
- final RunPlan source trace visible in preview/debug.

## F-004: ModelArtifact UI must not hold runtime serving parameters

### Evidence to verify

Inspect:

```text
web/src/pages/ModelArtifactsPage.vue
internal/server/api/*artifact*
internal/server/catalog/*
```

Run:

```bash
rg -n "parameterDefaults|servingParams|max-model-len|served-model-name|gpu-memory-utilization|extra_args|Docker|env|devices|privileged" web/src/pages/ModelArtifactsPage.vue internal docs
```

### Concrete problem

Model facts must not include Docker/backend serving args. Model layer may contain model facts and safe hints only.

### Required fix

Remove backend serve args from model edit surface. If model-level hints are retained, they must be structured and explicitly model facts, such as:

- max context detected/recommended;
- architecture;
- quantization;
- task type;
- capabilities;
- tokenizer/chat template hints.

Do not store:

- `--max-model-len`;
- `--served-model-name`;
- `--gpu-memory-utilization`;
- Docker args;
- env;
- devices;
- host ports.

## F-005: OpenAI-compatible gateway is a product gap

### Evidence to verify

Run:

```bash
rg -n "openai|chat/completions|completions|embeddings|api key|apikey|usage|token|meter|audit|HandleModelInstanceTest|endpoint_url" internal web docs configs
```

Inspect:

```text
internal/server/api/router.go
internal/server/api/*audit*
internal/server/db/db.go
internal/server/auth/*
internal/server/api/*instance*
internal/server/api/*deployment*
docs/api/openapi.yaml
```

### Concrete problem

Direct model instance endpoints exist, but product users need a stable tenant-scoped entrypoint with API key, audit, usage, and controlled routing.

### Required fix

Implement minimal gateway:

```text
GET  /v1/models
POST /v1/chat/completions
```

Optionally include:

```text
POST /v1/completions
POST /v1/embeddings
```

Only if the current backend capability model supports them safely.

Add:

- API key table;
- usage records table;
- gateway request audit;
- Bearer auth middleware;
- deployment/model routing;
- backend proxy;
- redaction;
- tests.

## F-006: Current regression evidence must be made authoritative

### Evidence to verify

Run:

```bash
find docs/reports -maxdepth 5 -type f | sort
find scripts -maxdepth 3 -type f | sort
rg -n "HISTORICAL|final-runtime-smoke|e2e|smoke|BLOCKED|PASS|SGLang|vLLM|llama.cpp" docs scripts
```

### Required fix

Create a current evidence directory:

```text
docs/reports/product-hardening-20260626/evidence/<timestamp>/
```

Do not rely on historical evidence without rerun or explicit historical label.
