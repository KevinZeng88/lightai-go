# 10. Codex Review and Execution Plan

> Status: REVIEW_PLAN_READY
> Scope: Review and planning only. No functional code changes.
> Date: 2026-06-27
> Branch: current branch (`main`)

## 1. Executive Summary

The root cause is not a single page bug. Runtime/model/deployment configuration is currently represented by several overlapping key families:

- catalog and ConfigSet keys such as `backend.common.host`, `backend.common.port`, `launcher.docker_options`, `runtime.health`, `runtime.model_mount`
- UI canonical aliases such as `service.listen_host` and `service.container_port`
- deployment service fields such as `host_port`, `container_port`, `served_model_name`
- RunPlan parameter names such as `max_model_len`, `gpu_memory_utilization`, and CLI flags such as `--max-model-len`
- model facts such as `default_context_length`, `capability_set_json.parameter_defaults`

The current implementation already has useful copy-on-create behavior for `BackendRuntime -> NodeBackendRuntime -> Deployment`, but the semantic contract is spread across catalog materialization, `internal/server/configedit`, API handlers, Vue pages, and RunPlan resolver code. `ConfigEditView` reduces UI duplication, but it currently relies on projection-time alias merging and hardcoded layer rules rather than a single semantic registry/normalizer/snapshot/resolver pipeline.

The execution plan should preserve the accepted snapshot boundaries while moving ownership, key normalization, display, validation, warnings, and backend CLI mapping into common modules.

## 2. Audit Evidence

### 2.1 Keyword Inventory

Search scope: `web/src`, `internal`, `configs`.

| Keyword | Matches | Files | Interpretation |
| --- | ---: | ---: | --- |
| `backend.common.host` | 3 | 3 | Legacy container listen host still generated and tested. |
| `backend.common.port` | 4 | 4 | Legacy container port still generated and tested. |
| `launcher.listen_host` | 1 | 1 | UI alias only; no storage prevalence. |
| `launcher.container_port` | 1 | 1 | UI alias only; no storage prevalence. |
| `service.listen_host` | 3 | 3 | Projection alias target exists but is not catalog/storage source. |
| `service.container_port` | 5 | 3 | Projection alias target exists but is not catalog/storage source. |
| `host_port` | 104 | 31 | Deployment/service/model-instance external port is widely used. |
| `container_port` | 77 | 34 | Service, template variable, tests, and runtime output overlap. |
| `backend.arg.` | 83 | 8 | Backend CLI parameters leak as user-facing ConfigSet keys. |
| `max_model_len` | 30 | 12 | Model runtime value appears as raw parameter name. |
| `max-model-len` | 35 | 7 | Backend CLI flag appears in catalog, resolver, and tests. |
| `context_length` | 53 | 13 | Model scan fact and backend runtime parameter overlap. |
| `served_model_name` | 48 | 14 | Deployment/service and backend CLI parameter overlap. |
| `gpu_memory_utilization` | 12 | 7 | Model runtime parameter appears as raw backend parameter. |
| `runtime.env` | 23 | 15 | Canonical-ish env key exists and should remain normalized. |
| `launcher.env` | 0 | 0 | No current storage key. |
| `docker_options` | 126 | 34 | Docker settings stored as a grouped launcher object. |
| `devices` | 245 | 63 | GPU discovery and Docker device mapping share generic term. |
| `optional_devices` | 10 | 6 | Docker option under launcher/runtime catalog. |
| `group_add` | 18 | 11 | Docker option under launcher/runtime catalog. |
| `health` | 437 | 73 | GPU health, runtime health check, and endpoint health are overloaded. |
| `model_mount` | 65 | 42 | Runtime model mount and RunPlan mount resolution overlap. |
| `RuntimeParameterEditor` | 0 | 0 | Component file exists but name is not imported; candidate for removal/isolation. |
| `ConfigEditView` | 60 | 14 | Common renderer already used by several pages. |
| `config_set_json` | 125 | 18 | Main persisted config surface across backend/version/runtime/NBR/deployment. |
| `source_metadata_json` | 57 | 11 | Source/copy metadata persisted separately from ConfigSet. |

### 2.2 Current Code Findings

| Area | Evidence | Finding |
| --- | --- | --- |
| Catalog materialization | `internal/server/catalog/loader.go` sets `backend.common.host`, `backend.common.port`, `runtime.model_mount`, `runtime.health`, `launcher.docker_options`, and dynamic `backend.arg.*`. | Duplicate modeling begins before UI; normalizing only frontend fields is insufficient. |
| Config registry | `configs/config-registry/items.yaml` defines `backend.common.*`, `launcher.*`, `runtime.*`. | Registry is technical-key based, not semantic-owner based. |
| ConfigEdit projector | `internal/server/configedit/project.go`, `taxonomy.go` merge aliases and hardcode layer visibility. | Useful stopgap, but page safety depends on UI projection rather than storage normalization. |
| ConfigEdit validator | `internal/server/configedit/validate.go` rejects unknown/hidden/read-only fields. | Causes hard errors like unknown canonical keys if projector emits a key absent from storage. |
| Runtime API | `internal/server/api/runtime_handlers.go`, `node_runtime_handlers.go` patch `launcher.image`, `launcher.docker_options`, `runtime.env`, `runtime.model_mount`, `runtime.health` directly. | API write paths bypass any future semantic registry unless adapted. |
| Deployment API | `internal/server/api/deployment_lifecycle_handlers.go`, `deployment_preview_handlers.go`, `configset_helpers.go` apply `config_overrides`, `editable_config_patch`, and service JSON separately. | Deployment parameters and service fields are split outside one snapshot builder. |
| RunPlan | `internal/server/runplan/resolver.go` uses NBR snapshot, but maps raw parameter keys/CLI names and has service overlay logic. | Snapshot boundary exists; semantic adapter mapping is missing. |
| Web runtime template page | `web/src/pages/BackendRuntimesPage.vue` uses `ConfigEditView` and raw JSON diagnostics. | Good direction, but relies on backend projection aliasing. |
| Web NBR pages | `web/src/pages/RunnerConfigsPage.vue`, `web/src/components/deployments/NodeRuntimeConfigWizard.vue` use `ConfigEditView` against `backend_runtime`/`node_backend_runtime` layers. | Good direction, but creation path sends image and editable patch separately. |
| Web deployment wizard | `DeploymentWizard.vue`, `DeploymentServiceEditor.vue`, `DeploymentOverrideEditor.vue` mix private service editor fields with ConfigEditView and inject `backend.common.served_model_name`. | Direct example of semantic leakage and page-private modeling. |
| Backend pages | `BackendsPage.vue` allows adding arbitrary `backend.arg.*` parameters. | BackendVersion currently edits CLI-facing user keys rather than semantic definitions plus adapter mappings. |
| Model pages | `ModelArtifactsPage.vue`, `artifact_handlers.go`, scanner code store context length, quantization, capabilities, and `parameter_defaults`. | Model facts exist, but they are not projected into deployment semantic snapshots by a common builder. |
| Diagnostics | `JsonViewer` shows raw ConfigSet/source metadata on runtime, runner config, backend, deployment, preview. | Keep, but route to diagnostic tier/read-only view. |

## 3. Entrypoint Inventory

Found entrypoints: 31.

| # | Entrypoint | Files | Current role | Required governance change |
| ---: | --- | --- | --- | --- |
| 1 | Backend list/detail | `web/src/pages/BackendsPage.vue`, `internal/server/api/backend_handlers.go` | Displays backend ConfigSet/source metadata. | Diagnostic-only for raw config; semantic capability view for supported keys. |
| 2 | BackendVersion add/edit/clone/delete | `BackendsPage.vue`, `backend_handlers.go`, catalog user YAML write paths | Edits version metadata and ConfigSet, can add `backend.arg.*`. | Replace with semantic definition + adapter mapping editor; block runtime values. |
| 3 | Backend catalog reload/seed | `backend_handlers.go`, `internal/server/catalog/loader.go` | Materializes catalog YAML into DB ConfigSet projections. | Normalize catalog YAML into canonical semantic keys during load. |
| 4 | System/user backend YAML | `configs/backend-catalog/backends/**`, `versions/**` | Defines health, default host/port, args schema, capabilities. | Move from user keys/CLI flags to semantic definitions and mappings. |
| 5 | System/user runtime YAML | `configs/backend-catalog/runtimes/**` | Defines image, Docker options, env, model mount, ports, health. | Normalize into runtime/service/docker semantic keys. |
| 6 | Config registry YAML | `configs/config-registry/items.yaml` | Technical ConfigSet base registry. | Replace/extend with `SemanticConfigRegistry`. |
| 7 | BackendRuntime list/detail | `BackendRuntimesPage.vue`, `runtime_handlers.go` | Runtime template display/edit with ConfigEditView. | View fields from `ConfigProjector` only. |
| 8 | BackendRuntime create from template | `HandleCreateBackendRuntimeFromTemplate` | Copies BackendVersion ConfigSet and patches direct fields. | Use `ConfigSnapshotBuilder` from BackendVersion semantic defaults. |
| 9 | BackendRuntime patch | `HandlePatchBackendRuntime` | Directly writes launcher/runtime keys or whole ConfigSet. | Route all config writes through semantic normalizer/validator. |
| 10 | BackendRuntime clone | `HandleCloneBackendRuntime` | Copies ConfigSet and applies direct overrides. | Preserve copy-on-create with copied_from metadata per item. |
| 11 | NodeBackendRuntime wizard select runtime | `NodeRuntimeConfigWizard.vue` | Reads runtime template and asks ConfigEditView for NBR layer. | Keep page thin; layer/context should be enough. |
| 12 | NodeBackendRuntime image picker/manual input | `NodeRuntimeConfigWizard.vue`, node Docker image APIs | Separate form field for image. | Treat as `runtime.image_ref` snapshot write. |
| 13 | NodeBackendRuntime enable | `HandleEnableNodeBackendRuntime`, `buildRuntimeConfigSnapshot` | Copies BackendRuntime ConfigSet at creation, applies image and patch. | Replace with `ConfigSnapshotBuilder` from BackendRuntime. |
| 14 | NodeBackendRuntime detail edit | `RunnerConfigsPage.vue`, `HandleConfigEditApply` | Edits NBR ConfigSet and marks needs_check. | Use semantic patch + warning/error response. |
| 15 | NodeBackendRuntime patch API | `HandlePatchNodeBackendRuntime` | Allows raw `config_set`/`config_set_json` replacement. | Remove/guard raw replacement; require semantic patch except admin diagnostic route. |
| 16 | NodeBackendRuntime check/probe | `HandleRequestNodeBackendRuntimeCheck` | Verifies image/Docker/backend match without mutating config snapshot. | Preserve; feed warning engine with probe warnings. |
| 17 | Deployment model selection | `DeploymentWizard.vue`, model APIs | Selects ModelArtifact. | Deployment snapshot builder must copy model recommendations. |
| 18 | Deployment NBR selection | `DeploymentWizard.vue`, `NodeRuntimeSelector.vue` | Selects deployable NodeBackendRuntime. | Snapshot source should be NBR, not live runtime. |
| 19 | Deployment service editor | `DeploymentServiceEditor.vue` | Private host/container/served name form. | Replace with projected semantic deployment/service fields. |
| 20 | Deployment override editor | `DeploymentOverrideEditor.vue` | ConfigEditView over NBR with deployment layer. | Rename model from override to deployment snapshot edit. |
| 21 | Deployment preview | `DeploymentPreviewPanel.vue`, `HandleDeploymentPreview` | Applies config overrides/patches and calls RunPlan. | Build transient deployment snapshot through common builder. |
| 22 | Deployment create | `HandleCreateDeployment` | Copies NBR ConfigSet and service JSON into deployment row. | Use `ConfigSnapshotBuilder` and semantic validator/warning engine. |
| 23 | Deployment patch | `HandlePatchDeployment` | Patches service JSON, placement JSON, overrides, and ConfigSet. | Single semantic patch path; service JSON only generated/projection as storage if kept. |
| 24 | Deployment start | `HandleStartDeployment`, `prepareDeploymentRun`, `runplan.Resolve` | Resolves stored deployment/NBR snapshots to task/run plan. | Resolve from deployment semantic snapshot + adapter mapping. |
| 25 | Deployment dry run/template sync | `HandleDeploymentDryRun`, template sync preview/apply | Compares ConfigSets and can sync deployment from templates. | Remove live-template sync for copied snapshots or reframe as explicit recopy operation. |
| 26 | ModelArtifact create/edit | `ModelArtifactsPage.vue`, `artifact_handlers.go` | Stores model facts and parameter defaults. | Define model-owned recommendation keys and copy policy to deployment. |
| 27 | Model scan/discover | `model_browser_handlers.go`, `artifact_handlers.go`, agent model scanner | Produces context length/quantization/capability facts. | Normalize scan facts into model recommendation source. |
| 28 | ModelLocation create/rescan/attest | model location handlers/pages | Node model path/location facts. | Keep model path outside runtime template; resolver builds model mount from location + mount key. |
| 29 | ConfigEditView renderer | `web/src/components/config/**`, `web/src/utils/configEditView.ts` | Renders projected sections/fields and patches all fields. | Add owner/tier/source/warnings/dirty; patch changed fields only. |
| 30 | RuntimeParameterEditor | `web/src/components/common/RuntimeParameterEditor.vue`, `runtimeParameterViewModel.ts` | Legacy human parameter editor not imported by name. | Remove from normal flows or isolate as diagnostic/dev-only. |
| 31 | Raw JSON diagnostics | `JsonViewer` usages | Shows raw config/source/probe/runplan JSON. | Keep behind diagnostic tier, read-only, collapsed. |

## 4. Duplicate Semantic Groups

Found duplicate semantic groups: 7.

| Group | Current duplicated forms | Canonical direction |
| --- | --- | --- |
| Listen host / container port / host port | `backend.common.host`, `launcher.listen_host`, `service.listen_host`, `backend.common.port`, `launcher.container_port`, `service.container_port`, `service_json.host_port`, `service_json.container_port`, template `{{container_port}}` | `service.listen_host`, `service.container_port`, `deployment.host_port`; health port references `service.container_port` unless explicitly overridden. |
| Model runtime parameters | `backend.arg.max_model_len`, `max_model_len`, `--max-model-len`, `context_length`, `backend.arg.gpu_memory_utilization`, `served_model_name` | `model_runtime.*` and `deployment.served_model_name`; backend flags only in adapter mapping. |
| Health check | `default_health_check`, `health_check`, `runtime.health`, `HealthCheckInput`, GPU `health` | `runtime.health.*` for service health; GPU health remains node metric, not ConfigSet field. |
| Model mount | `default_model_mount`, `model_mount`, `runtime.model_mount`, RunPlan model path variables | `runtime.model_mount.container_path` owns container base; ModelLocation owns host root/path. |
| Runtime env | `runtime.env`, `env`, `env_schema`, `env_overrides`, `ParameterValue target=env` | `runtime.env` snapshot for runtime/NBR; deployment env copied as own snapshot item or deployment env group. |
| Docker options / device binding | `launcher.docker_options`, `devices`, `optional_devices`, `group_add`, `launcher.devices`, `NodeOverrideInfo.DockerOverride` | `docker.*` semantic keys under runtime environment; scheduler GPU selection separate from Docker device mapping. |
| Backend capabilities / runtime requirements | `backend.capabilities`, `backend.supported_config_items`, `capability_set_json`, `supported_model_formats`, image probe backend match | `backend_capability.*` for backend support/mapping; `model.capability.*` for model facts; probe results diagnostic. |

## 5. Canonical Semantic Key Table

| Semantic key | Owner | Value type | Copied to | Default/recommended source | Hard validation | Warning rules | Resolver mapping | Display tier | Legacy keys to remove/normalize |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `runtime.image_ref` | runtime_environment | string | BackendRuntime, NBR | Runtime YAML `image_ref`/first `image_candidates`; node image picker at NBR | required for docker run; non-empty string | image not inspected/present from probe | Docker image field | required | `launcher.image`, API `image_ref` |
| `runtime.command` | runtime_environment | string array | BackendRuntime, NBR | Version/runtime YAML args/default command | array of strings; template vars parse | command differs from image default/profile | Docker Cmd / process args base | common | `launcher.command`, `default_args`, `args` |
| `runtime.entrypoint` | runtime_environment | string array | BackendRuntime, NBR | Version/runtime YAML entrypoint | array of strings | overriding image entrypoint may break image | Docker Entrypoint | advanced | `launcher.entrypoint`, `default_entrypoint` |
| `runtime.env` | runtime_environment | object string map | BackendRuntime, NBR, optionally Deployment | Runtime YAML env/env_schema | keys strings; values scalar/stringable | empty env skipped; secret-like values must be redacted | Docker env | common/advanced split | direct API `env`, `env_overrides` |
| `service.listen_host` | runtime_service | string | BackendRuntime, NBR, Deployment snapshot if needed | BackendVersion default host; adapter default `0.0.0.0` | valid host/IP/string; required when flag emitted | non-`0.0.0.0` may make container unreachable | vLLM/SGLang/llama.cpp `--host` | required/common | `backend.common.host`, `launcher.listen_host` |
| `service.container_port` | runtime_service | integer | BackendRuntime, NBR, Deployment snapshot if needed | BackendVersion `default_port`; runtime `ports.container_port` | 1-65535 integer | conflicts with image exposed ports or health port mismatch | vLLM/SGLang/llama.cpp `--port`; Docker container port | required/common | `backend.common.port`, `launcher.container_port`, `default_container_port`, `service_json.container_port` |
| `deployment.host_port` | deployment_exposure | integer | Deployment | User deployment service editor/default | 1-65535 integer; conflict check hard before start | host port uncommon/privileged range warning | Docker host port | required/common | `host_port`, `service_json.host_port` |
| `deployment.served_model_name` | deployment_exposure | string | Deployment | Artifact name or user input | string format; required only if adapter requires | differs from artifact name | vLLM/SGLang `--served-model-name`; llama.cpp adapter optional/unsupported | common | `backend.common.served_model_name`, `served_model_name` |
| `model_runtime.max_model_len` | model_runtime | integer | Deployment | Model scan context length, BackendVersion recommended default | positive integer | above model context/default; estimated VRAM exceeds available | vLLM `--max-model-len`, SGLang `--context-length`, llama.cpp `--ctx-size` | advanced | `backend.arg.max_model_len`, raw `max_model_len`, `--max-model-len` |
| `model_runtime.context_length` | model_runtime | integer | ModelArtifact recommendation, Deployment | Scanner metadata `context_length`, HF/GGUF facts | positive integer if present | unknown context length | Usually source for max length recommendation, not direct flag unless adapter needs | diagnostic/recommended | scanner `context_length`, `default_context_length` as separate DB column remains source fact |
| `model_runtime.gpu_memory_utilization` | model_runtime | number | Deployment | BackendVersion/runtime recommendation | number 0-1 or normalized percent, choose one and enforce | high value may OOM | vLLM/SGLang matching flag when supported | advanced | `backend.arg.gpu_memory_utilization`, raw `gpu_memory_utilization` |
| `model_runtime.dtype` | model_runtime | string enum | Deployment | Model metadata/backend default | enum/string accepted values | backend may ignore unsupported dtype | adapter-specific flag/env | advanced | `backend.arg.dtype`, raw dtype args |
| `model_runtime.quantization` | model_runtime | string | ModelArtifact, Deployment recommendation | Model scan quantization | string enum when known | backend may not support quantization | adapter-specific flag if supported | recommended/advanced | artifact `quantization` is source fact |
| `model_runtime.max_num_seqs` | model_runtime | integer | Deployment | BackendVersion recommended default | positive integer | high concurrency memory risk | backend adapter flag | advanced | `backend.arg.max_num_seqs` |
| `model_runtime.max_num_batched_tokens` | model_runtime | integer | Deployment | BackendVersion recommended default | positive integer | high batch token memory risk | backend adapter flag | advanced | `backend.arg.max_num_batched_tokens` |
| `runtime.health.path` | runtime_service | string | BackendRuntime, NBR, Deployment snapshot | Backend/runtimes YAML health check | valid absolute HTTP path when enabled | missing health for service-capable backend | Agent health check path | recommended | `runtime.health.path`, `health_check.path`, `default_health_check.path` |
| `runtime.health.timeout_seconds` | runtime_service | integer | BackendRuntime, NBR | YAML health timeout/default | positive integer | too low may false fail | Agent health check timeout | advanced | `runtime.health.timeout`, `health_check.timeout_seconds` |
| `runtime.health.interval_seconds` | runtime_service | integer | BackendRuntime, NBR | YAML health interval/default | positive integer | too frequent may add load | Agent health check interval | advanced | `runtime.health.interval`, `health_check.interval_seconds` |
| `runtime.health.expected_status` | runtime_service | integer | BackendRuntime, NBR | YAML health expected_status | 100-599 integer | non-200 may surprise OpenAI-compatible APIs | Agent health check expected status | advanced | `health_check.expected_status` |
| `runtime.model_mount.container_path` | runtime_environment | string | BackendRuntime, NBR | Version/runtime YAML model mount | absolute container path; no traversal | non-standard path must match command template | Docker volume container path; `MODEL_CONTAINER_PATH` base | common | `runtime.model_mount`, `default_model_mount.container_path` |
| `runtime.model_mount.readonly` | runtime_environment | boolean | BackendRuntime, NBR | YAML model mount readonly | boolean | writable model mount may mutate model files | Docker volume readonly flag | advanced | `runtime.model_mount.readonly` |
| `docker.shm_size` | runtime_environment | string | BackendRuntime, NBR | Runtime YAML Docker options | parse Docker size string | too small/large warning | Docker HostConfig ShmSize/preview | common | `launcher.docker_options.shm_size`, request `docker_options.shm_size` |
| `docker.ipc_mode` | runtime_environment | string enum | BackendRuntime, NBR | Runtime YAML Docker options | enum/string | `host` has isolation implications | Docker HostConfig IPCMode | recommended/advanced | `launcher.docker_options.ipc_mode` |
| `docker.privileged` | runtime_environment | boolean | BackendRuntime, NBR | Runtime YAML Docker options | boolean | privileged container security warning | Docker HostConfig Privileged | advanced | `launcher.docker_options.privileged` |
| `docker.network_mode` | runtime_environment | string | BackendRuntime, NBR | Runtime YAML Docker options | string | host network conflicts with port mapping | Docker HostConfig NetworkMode | advanced | `launcher.docker_options.network_mode` |
| `docker.security_options` | runtime_environment | string array | BackendRuntime, NBR | Runtime YAML Docker options | array strings | security profile may break runtime | Docker HostConfig SecurityOpt | advanced | `launcher.docker_options.security_options` |
| `docker.ulimits` | runtime_environment | object | BackendRuntime, NBR | Runtime YAML Docker options | parse ulimit object | unsupported ulimit warning | Docker HostConfig Ulimits | advanced | `launcher.docker_options.ulimits` |
| `docker.devices` | runtime_environment | array objects | BackendRuntime, NBR | Runtime YAML `docker_options.devices` / `devices` | object paths valid-ish; no path traversal | missing optional device warning; raw device security warning | Docker device mappings | advanced | `launcher.docker_options.devices`, top-level runtime `devices` |
| `docker.optional_devices` | runtime_environment | array | BackendRuntime, NBR | Runtime YAML Docker options | array | missing optional devices warning only | Docker device mappings if present | advanced | `launcher.docker_options.optional_devices` |
| `docker.group_add` | runtime_environment | string array | BackendRuntime, NBR | Runtime YAML Docker options | array strings | group may not exist on host | Docker group_add | advanced | `launcher.docker_options.group_add` |
| `scheduler.gpu_count` | scheduler_resource | integer | Deployment/placement | Model requirement/artifact `required_gpu_count`, user placement | non-negative integer | insufficient GPUs warning/admission | Scheduler/lease only; not Docker device key | common | `required_gpu_count` as source fact |
| `scheduler.accelerator_ids` | scheduler_resource | string array | Deployment/placement | User placement | existing GPU ids when binding hard | stale/occupied GPU warning | GPU lease and visible-device vars | common | `placement_json.accelerator_ids` |
| `backend_capability.supported_config_keys` | backend_capability | string array | BackendVersion only | Catalog/adapter | known semantic keys | adapter missing mapping warning | Adapter registry | diagnostic | `backend.supported_config_items` |
| `backend_capability.resolver_mapping` | backend_capability | object | BackendVersion only | Catalog/adapter | maps known semantic key to supported target | mapping missing/unsupported warning | `BackendAdapterMapping` source | diagnostic/admin | `render.flag`, `cli_name`, raw args schema as user keys |

## 6. Target Program Architecture

### 6.1 `internal/server/semanticconfig`

Add a new package rather than expanding page/API-specific logic:

| Module | Responsibility |
| --- | --- |
| `types.go` | `SemanticParamDefinition`, `Owner`, `ValueType`, `DisplayTier`, `CopyPolicy`, `ConfigSnapshotItem`, `SourceSnapshot`, `ConfigWarning`, `ValidationError`, `ResolverMapping`. |
| `registry.go` | In-memory `SemanticConfigRegistry` with canonical key registration, owner uniqueness, backend support lookup, display metadata, and mapping lookup. |
| `normalizer.go` | `SemanticConfigNormalizer` for legacy key to canonical key conversion, conflict detection, and diagnostic warnings. |
| `catalog_loader.go` | Adapts existing catalog/config-registry YAML and future semantic YAML into registry definitions and adapter mappings. |
| `snapshot.go` | `ConfigSnapshotBuilder` for BackendVersion -> BackendRuntime, BackendRuntime -> NBR, NBR/ModelArtifact -> Deployment. |
| `projector.go` | `ConfigProjector` converts semantic snapshots to edit views for object kind/mode/tier. |
| `warnings.go` | `ConfigWarningEngine` for recommendation, VRAM, backend support, advanced Docker, image/probe, and dirty warnings. |
| `validator.go` | `ConfigValidator` for hard errors only: type, required, enum, parse, path, port, unknown canonical keys, direct legacy patch. |
| `resolver.go` | Semantic RunPlan input projection and adapter dispatch. |
| `adapters/vllm.go`, `adapters/sglang.go`, `adapters/llamacpp.go` | `BackendAdapterMapping` from semantic key to CLI/env/docker output. |

### 6.2 Integration Points

| Existing area | Integration |
| --- | --- |
| `internal/server/catalog` | Materialize ConfigSets through normalizer; stop generating long-lived `backend.common.*` and `backend.arg.*` user config keys. |
| `internal/server/configedit` | Either wrap/replace with `semanticconfig.ConfigProjector` and `ConfigValidator`, or keep package as HTTP view DTO layer only. |
| `internal/server/api/config_edit_handlers.go` | Load object snapshot, normalize, project, validate patch, save canonical snapshot with warnings. |
| `runtime_handlers.go` / `node_runtime_handlers.go` | Replace direct `setConfigValue` calls with `ConfigSnapshotBuilder` and semantic patch application. |
| `deployment_lifecycle_handlers.go` / `deployment_preview_handlers.go` | Build Deployment snapshot once; RunPlan preview/start read from deployment semantic snapshot. |
| `internal/server/runplan` | Keep current Docker/health/mount execution structures, but fill them from semantic resolver output. |
| `web/src/components/config/*` | Render owner/tier/warnings/dirty/source; remove hardcoded semantic decisions from pages. |

### 6.3 ConfigEditView DTO Changes

Extend `EditField` / TS types with:

```text
owner
tier
display_label
original_label
semantic_key
copied_from
dirty
warnings[]
diagnostic
```

Replace section keys with tier-aware grouping:

```text
required, common, recommended, advanced, diagnostic
```

Pages should pass only:

```text
object_kind, object_id, layer/mode/display_context
```

and should not maintain per-page parameter visibility lists.

## 7. Batch Execution Plan

### Batch 0: Review Plan Freeze

| Item | Detail |
| --- | --- |
| Goal | Freeze the audit, semantic ownership table, architecture, and batch acceptance criteria. |
| Modify files | `docs/reports/phase-3/semantic-config-governance/10-codex-review-and-execution-plan.md`; optional closeout/open-issues doc if implementation starts. |
| Steps | Review this report with product/maintainers; confirm open questions; choose DB rebuild/catalog reload stance. |
| Tests | `git diff --check`; markdown review; no functional tests required because no code changes. |
| Acceptance | Report lists all entrypoints, duplicate groups, semantic key table, common modules, batches, and open questions. |
| Risks | Plan may miss hidden runtime paths outside `web/src`, `internal`, `configs`; mitigate by re-running keyword inventory before Batch 1. |

### Batch 1: Semantic Registry and Normalizer

| Item | Detail |
| --- | --- |
| Goal | Establish canonical keys, owner uniqueness, legacy key normalization, and conflict diagnostics. |
| Modify files | Create `internal/server/semanticconfig/types.go`, `registry.go`, `normalizer.go`, `catalog_loader.go`; modify `internal/server/catalog/loader.go`, `configs/config-registry/items.yaml`; add tests under `internal/server/semanticconfig`. |
| Steps | Define registry types; register first canonical key set; implement legacy key mapping; normalize existing catalog materialization output; produce diagnostic warnings on conflicting alias values; reject direct legacy patch after normalization boundary. |
| Tests | `go test ./internal/server/semanticconfig ./internal/server/catalog ./internal/server/configedit`; add tests for alias conflict and no legacy keys after normalize. |
| Acceptance | Normalized ConfigSet contains `service.listen_host` and `service.container_port`, not `backend.common.host` / `backend.common.port`; `backend.arg.*` not emitted as long-lived user keys. |
| Risks | Existing tests assert legacy keys; update tests with DB rebuild/catalog reload assumption. |

### Batch 2: Config Snapshot Builder

| Item | Detail |
| --- | --- |
| Goal | Replace ad hoc copy/patch paths with a single copy-on-create snapshot builder. |
| Modify files | Create `snapshot.go`, `copy_policy.go`; modify `runtime_handlers.go`, `node_runtime_handlers.go`, `deployment_lifecycle_handlers.go`, `deployment_preview_handlers.go`, `configset_helpers.go`; add focused API tests. |
| Steps | Implement BackendVersion -> BackendRuntime copy; BackendRuntime -> NBR copy; NBR + ModelArtifact -> Deployment copy; store `copied_from`, `copied_at`, `dirty=false`; treat image picker as `runtime.image_ref`; treat served name as `deployment.served_model_name`. |
| Tests | `go test ./internal/server/api -run 'RuntimeBoundary|Workflow|ConfigEdit|Deployment'`; add tests that upstream edits do not mutate existing downstream snapshots. |
| Acceptance | Existing snapshot boundary tests still pass; Deployment snapshot includes model runtime recommendations copied from artifact/backend defaults. |
| Risks | Deployment start currently derives some data from service JSON; migration must keep API response compatibility while changing internal authority. |

### Batch 3: Projector, Warning Engine, and Validator

| Item | Detail |
| --- | --- |
| Goal | Centralize display tiers, warnings, hard validation, and changed-field patching. |
| Modify files | Create/modify `internal/server/semanticconfig/projector.go`, `warnings.go`, `validator.go`; modify `internal/server/configedit/*`; update `web/src/utils/configEditView.ts`, `web/src/components/config/ConfigEditView.vue`, `ConfigSection.vue`, `ConfigField.vue`; update i18n labels. |
| Steps | Project only current-object snapshot items; add owner/tier/warnings/dirty fields; render required/common/recommended/advanced/diagnostic; return warnings with save responses; enforce hard errors only for type/required/format/parse/unknown canonical key. |
| Tests | `go test ./internal/server/semanticconfig ./internal/server/configedit ./internal/server/api -run ConfigEdit`; `cd web && npm test`; add component/unit tests for warning display and changed-only patches. |
| Acceptance | Risky but parseable values save with warnings; hard invalid values fail; BackendRuntime and NBR do not display model runtime keys; Deployment displays model runtime keys as advanced. |
| Risks | UI translation and layout churn; mitigate by keeping renderer changes small and schema-driven. |

### Batch 4: RunPlan Resolver and Backend Adapter Mapping

| Item | Detail |
| --- | --- |
| Goal | Generate CLI/env/docker/health/model mount output from semantic keys, not user-facing backend flags. |
| Modify files | Create `internal/server/semanticconfig/resolver.go`, `adapters/vllm.go`, `adapters/sglang.go`, `adapters/llamacpp.go`; modify `internal/server/runplan/resolver.go`, `lint.go`, tests; update catalog mapping fields. |
| Steps | Implement adapter lookup by backend family/version; map `service.listen_host`, `service.container_port`, `deployment.served_model_name`, and `model_runtime.*`; map `docker.*`; health defaults reference `service.container_port`; model mount resolves from ModelLocation plus container path. |
| Tests | `go test ./internal/server/runplan ./internal/server/api -run 'RunPlan|WorkflowDeployment|UIPersistence'`; add vLLM/SGLang/llama.cpp semantic mapping tests. |
| Acceptance | vLLM still emits `--host`, `--port`, `--max-model-len`; SGLang and llama.cpp emit their backend-specific equivalents; no primary read from `backend.common.*` or `backend.arg.*`. |
| Risks | Existing RunPlan tests use raw parameter names; update expected inputs to semantic snapshots while preserving final command output. |

### Batch 5: Web Entrypoint Migration

| Item | Detail |
| --- | --- |
| Goal | Remove page-private parameter modeling from runtime, NBR, deployment, backend, and model pages. |
| Modify files | `BackendRuntimesPage.vue`, `RunnerConfigsPage.vue`, `NodeRuntimeConfigWizard.vue`, `DeploymentWizard.vue`, `DeploymentServiceEditor.vue`, `DeploymentOverrideEditor.vue`, `BackendsPage.vue`, `ModelArtifactsPage.vue`, `RuntimeParameterEditor.vue`, `JsonViewer` placements, `web/src/api/*`. |
| Steps | Replace deployment service editor with semantic projected fields; remove injection of `backend.common.served_model_name`; make BackendVersion parameter editor define semantic mapping rather than raw `backend.arg.*`; keep raw JSON only diagnostic; remove or isolate RuntimeParameterEditor. |
| Tests | `cd web && npm run build && npm test`; Playwright/manual smoke for runtime template, NBR creation/edit/check, deployment preview/save. |
| Acceptance | Pages no longer hardcode field visibility or semantic mappings; they pass object kind/mode/context to ConfigEditView. |
| Risks | User-facing workflow regressions; mitigate with API-first E2E before manual UI verification. |

### Batch 6: Catalog Cleanup, DB Rebuild Path, E2E, Closeout

| Item | Detail |
| --- | --- |
| Goal | Remove legacy keys from catalog/storage generation, document rebuild/reload path, and close acceptance. |
| Modify files | `configs/backend-catalog/**`, `configs/config-registry/items.yaml`, `docs/CURRENT.md`, semantic-governance acceptance docs, E2E scripts under `scripts/e2e/`, tests. |
| Steps | Rewrite catalog YAML or loader compatibility to canonical keys; document DB rebuild/catalog reload; add semantic snapshot/runplan E2E scripts; run final build/test matrix; update formal closeout docs. |
| Tests | `go build ./cmd/server/...`; `go build ./cmd/agent/...`; `go test ./internal/server/...`; `go test ./internal/agent/...`; `cd web && npm run build`; `cd web && npm test`; semantic E2E scripts. |
| Acceptance | No long-lived `backend.common.*`, `launcher.listen_host`, `launcher.container_port`, or user-facing `backend.arg.*`; all known findings are fixed or documented with blocker status. |
| Risks | Catalog reload/DB rebuild can invalidate local data; communicate as intended breaking change for this governance refactor. |

## 8. Validation Matrix

| Layer | Required validation |
| --- | --- |
| Registry/normalizer | Legacy aliases normalize; conflicts produce warnings; unknown canonical keys fail; backend flags are mappings only. |
| Snapshot | BackendRuntime/NBR/Deployment copies are detached; source metadata records copied_from/copy time; dirty flags update on edit. |
| Projector/UI | Tiers render correctly; warnings visible; advanced collapsed; diagnostic read-only; changed-only patch. |
| API | Runtime/NBR/deployment direct patch paths all route through semantic validation; raw ConfigSet replacement removed or admin-only diagnostic. |
| RunPlan | Final commands unchanged for accepted NVIDIA vLLM/SGLang/llama.cpp path; semantic source included in preview. |
| E2E | Create runtime, create NBR, edit image, create deployment, edit model runtime param, preview RunPlan, verify warnings. |

## 9. Open Questions

Only design questions requiring user/maintainer confirmation:

1. Should `deployment.host_port` remain persisted in `service_json` for API compatibility while semantic snapshot becomes authority, or can `service_json` be rebuilt from semantic snapshot after DB rebuild?
2. Should `runtime.command` remain a user-editable semantic key, or should command/profile selection be a separate `runtime.process_start_profile` semantic key with raw command in diagnostic/admin tier?
3. Should `model_runtime.context_length` be directly editable at Deployment, or should it remain a model fact that only produces the recommended default/warning for `model_runtime.max_model_len`?
4. Should BackendVersion user editing expose adapter mappings in the UI, or should adapter mappings be catalog/dev-file only for this phase?
5. For GPU binding, should `scheduler.accelerator_ids` stay outside ConfigSet permanently, with only Docker device mapping in semantic config, or should placement also be projected through ConfigEditView?

## 10. Final Planning Status

| Item | Status |
| --- | --- |
| Review and plan file | REVIEW_PLAN_READY |
| Functional code modified | No |
| Entrypoints found | 31 |
| Duplicate semantic groups found | 7 |
| Suggested new modules | `SemanticConfigRegistry`, `SemanticConfigNormalizer`, `ConfigSnapshotBuilder`, `ConfigProjector`, `ConfigWarningEngine`, `ConfigValidator`, `RunPlanResolver`, `BackendAdapterMapping`, ConfigEditView renderer metadata changes |
| Suggested batches | Batch 0 review freeze; Batch 1 registry/normalizer; Batch 2 snapshot builder; Batch 3 projector/warnings/validator; Batch 4 resolver/adapters; Batch 5 web migration; Batch 6 cleanup/E2E/closeout |
