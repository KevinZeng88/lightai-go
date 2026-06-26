# 10. Codex Review and Execution Plan

> Status: REVIEW_PLAN_READY
> Scope: review and planning only; no functional code changes in this round.
> Date: 2026-06-27
> Branch: current `main`

## Executive Summary

本轮重新对照当前代码后，结论是：问题不是 `DeploymentWizard`、`BackendRuntimesPage` 或某个 `ConfigEditView` 字段的单点缺陷，而是语义所有权、存储 key、展示 key、保存入口和 RunPlan 解析入口没有统一程序治理。

当前代码已经有可复用基础：`config_set_json` 贯穿 BackendVersion、BackendRuntime、NodeBackendRuntime、Deployment；BackendRuntime -> NodeBackendRuntime -> Deployment 已基本按 copy-on-create 保存快照；`ConfigEditView` 已作为多入口通用 renderer；RunPlan 已有 Docker、health、mount 输出结构。但当前 semantic contract 仍散落在 catalog materialization、`internal/server/configedit`、API handlers、Vue 页面和 RunPlan resolver 中。

目标是建立 `semanticconfig` 通用模块，让每个业务参数只有一个 canonical semantic key 和一个 owner；Backend CLI flag 只存在于 adapter mapping，不作为用户配置 key；下游对象复制参数后持有自己的 snapshot，并通过 `copied_from/source_snapshot/dirty/warnings` 表达来源和差异；风险给 warning，硬错误只覆盖类型、必填、格式、enum、路径/端口不可解析、unknown canonical key、direct legacy key patch。

Codex 同意用户给出的方向。当前代码证据显示，如果继续在页面或单个 handler 内补丁式修复，会继续扩大 `service_json`、`backend.common.*`、`backend.arg.*`、RunPlan raw parameter 之间的重复建模。

## Current Code Evidence

### Keyword Audit

Search scope: `configs/backend-catalog`, `configs/config-registry/items.yaml`, `internal/server/catalog`, `internal/server/configedit`, `internal/server/api`, `internal/server/runplan`, `web/src`.

| Keyword | Matches | Files | Current meaning |
| --- | ---: | ---: | --- |
| `backend.common.host` | 3 | 3 | Catalog/materialized legacy listen host. |
| `backend.common.port` | 4 | 4 | Catalog/materialized legacy container port. |
| `launcher.listen_host` | 1 | 1 | Alias vocabulary, not a durable storage surface. |
| `launcher.container_port` | 1 | 1 | Alias vocabulary, not a durable storage surface. |
| `service.listen_host` | 3 | 3 | ConfigEdit projection target, not current storage source. |
| `service.container_port` | 5 | 3 | ConfigEdit projection target, not current storage source. |
| `host_port` | 92 | 24 | Deployment service exposure and preview/start input. |
| `container_port` | 68 | 28 | Service JSON, template variables, RunPlan and config keys overlap. |
| `backend.arg.` | 83 | 8 | Dynamic backend CLI arg keys are user-visible and persisted. |
| `max_model_len` | 28 | 10 | Raw model runtime parameter, CLI/default variable and UI value. |
| `context_length` | 44 | 7 | Model fact and runtime recommendation overlap. |
| `served_model_name` | 47 | 13 | Deployment service field and backend CLI arg overlap. |
| `gpu_memory_utilization` | 9 | 4 | Raw backend runtime parameter. |
| `runtime.health` | 19 | 11 | Runtime service health config, distinct from node/GPU health metrics. |
| `runtime.model_mount` | 19 | 12 | Runtime container model mount config. |
| `launcher.docker_options` | 84 | 16 | Grouped Docker host config storage. |
| `docker_options` | 126 | 34 | Request, catalog and RunPlan Docker config vocabulary. |
| `devices` | 159 | 43 | GPU/device discovery and Docker device binding share a generic word. |
| `optional_devices` | 10 | 6 | Docker option under launcher/runtime config. |
| `group_add` | 17 | 10 | Docker group option under launcher/runtime config. |
| `runtime.env` | 23 | 15 | Runtime env snapshot key, broadly reusable. |
| `config_set_json` | 119 | 16 | Main persisted config snapshot surface. |
| `source_metadata_json` | 52 | 10 | Object-level source/copy/probe metadata surface. |
| `service_json` | 41 | 10 | Deployment service storage separate from ConfigSet. |
| `editable_config_patch` | 15 | 7 | ConfigEdit patch transport. |
| `ConfigEditView` | 60 | 14 | Common renderer already used in runtime/NBR/deployment/backend pages. |
| `RuntimeParameterEditor` | 0 | 0 | Component exists but is not used as a normal entrypoint by name. |

### Important Current Implementations

| Area | Code evidence | Current responsibility |
| --- | --- | --- |
| Catalog loader | `internal/server/catalog/loader.go` | `MaterializeBackendVersion` writes `backend.common.host`, `backend.common.port`, `runtime.model_mount`, `runtime.health`; `addArgConfigItems` turns `default_args_schema` into `backend.arg.*`; `MaterializeBackendRuntime` writes `launcher.image`, `launcher.docker_options`, `runtime.env`, `runtime.model_mount`, `runtime.health`, `launcher.ports`. |
| Config registry | `configs/config-registry/items.yaml` | Defines technical keys including `launcher.*`, `runtime.*`, `backend.common.*`, `backend.extra_args`; it is not semantic-owner based. |
| ConfigEdit projection | `internal/server/configedit/project.go`, `taxonomy.go` | `ProjectConfigSetToEditView` normalizes shape, merges aliases, projects `launcher.docker_options` into subfields, and uses hardcoded layer visibility/readonly rules. |
| ConfigEdit validation/apply | `internal/server/configedit/validate.go`, `apply.go` | Validates against projected hidden/readonly/protected fields, rejects unknown internal keys, and writes values back to existing ConfigSet items. |
| Runtime APIs | `internal/server/api/runtime_handlers.go` | Runtime create/patch writes `launcher.image`, `launcher.docker_options`, `runtime.env`, `runtime.model_mount`, `runtime.health`, `launcher.entrypoint`, `launcher.command`; raw `config_set` replacement still exists. |
| Node runtime APIs | `internal/server/api/node_runtime_handlers.go` | Clone/patch copies ConfigSet; patch allows raw `config_set` and `source_metadata_json` replacement. |
| Deployment APIs | `internal/server/api/deployment_lifecycle_handlers.go`, `deployment_preview_handlers.go`, `configset_helpers.go` | Create/preview copy NBR ConfigSet, apply `config_overrides` and `editable_config_patch`, store `service_json` separately, and map ConfigSet CLI items to RunPlan parameter values. |
| RunPlan resolver | `internal/server/runplan/resolver.go` | Keeps useful Docker/health/mount structures, but reads raw parameter names and CLI flags, protects `--host/--port`, and builds vars from `container_port`, `host_port`, `served_model_name`, `max_model_len`, `gpu_memory_utilization`. |
| Web pages | `web/src/pages/**`, `web/src/components/config/**`, `web/src/components/deployments/**` | Runtime/NBR/backend pages use `ConfigEditView`; Deployment wizard still has private service model and injects `backend.common.served_model_name`. |
| Web API clients | `web/src/api/**` | Deployments and config edit clients expose `service_json`, `config_overrides`, `editable_config_patch`, raw config structures. |

## Target Model

1. 每个业务参数只能有一个 semantic key 和一个 owner。
2. Backend CLI flag 不作为用户配置 key，只作为 `BackendAdapterMapping` 输出。
3. 下游对象使用 copy-on-create snapshot。BackendRuntime 从 BackendVersion 复制，NodeBackendRuntime 从 BackendRuntime 复制，Deployment 从 NodeBackendRuntime、ModelArtifact 和 service input 复制。
4. 下游复制参数后，参数属于当前对象 snapshot，和上游解除 live 关系，只保留 `copied_from`、`source_snapshot`、`copied_at`、`dirty`、`warnings`。
5. 不使用 `override` / `editable_at` 作为核心概念；这些可以作为兼容输入名或 UI 文案迁移对象，但不能决定语义。
6. 参数是否能改不按层级硬限制；当前对象 snapshot 中存在的参数原则上可改。只有 registry 显式声明为 derived/diagnostic 的字段不可普通编辑。
7. 风险类问题给 warning，不阻断保存。
8. 硬校验只处理类型错误、required 缺失、格式非法、enum 非法、路径/端口不可解析、unknown canonical key、direct legacy key patch。
9. 页面不得维护私有参数语义判断；所有 owner、tier、label、warning、validation、resolver mapping 来自 `semanticconfig` 通用模块。
10. 不需要复杂历史兼容；必要时允许 DB rebuild / catalog reload。Legacy key 仅作为 normalizer 防御输入和诊断证据。

## Current Code Gap Matrix

| Area | Current code evidence | Current behavior | Target model | Reuse | Migrate | Remove / diagnostic-only | Batch |
| ---- | --------------------- | ---------------- | ------------ | ----- | ------- | ------------------------ | ----- |
| catalog materialization | `internal/server/catalog/loader.go`: `MaterializeBackendVersion`, `MaterializeBackendRuntime`, `addArgConfigItems` | Loader 直接生成 technical keys 和 `backend.arg.*` user config items。 | Catalog 产出 canonical semantic definitions、snapshot defaults、adapter mapping。 | 保留 YAML load、checksum、system/user catalog 合并、materialize DB 流程。 | `backend.common.*`、`backend.arg.*`、`launcher.docker_options` 输入经 normalizer/registry 生成 canonical snapshot。 | `backend.arg.*` 仅作为 legacy input；CLI flag 写入 mapping，不作为 user config key。 | 1, 2, 6 |
| config registry | `configs/config-registry/items.yaml` | Registry 按 storage key 定义 label/type/section。 | `SemanticConfigRegistry` 按 semantic key、owner、value type、tier、copy policy、mapping 定义。 | 复用 label、type、widget、section/tier 初始内容。 | YAML schema 分批扩展或新增 semantic registry 文件。 | `backend.common.*`、`backend.extra_args` 等 technical entries 降级为 legacy normalizer 输入。 | 1 |
| configedit projector | `internal/server/configedit/project.go`, `taxonomy.go` | Projection-time alias merge；`launcher.docker_options` 拆子字段；layer hidden/readonly 硬编码。 | `ConfigProjector` 从 semantic snapshot + registry 输出 edit view。 | 保留 EditView DTO、section/field 形态、Docker option 子字段投影经验。 | alias merge 前移到 normalizer；visibility/label/tier 由 registry 驱动。 | Projection-time alias merge 只保留 legacy defensive diagnostics。 | 3 |
| configedit validator/apply | `internal/server/configedit/validate.go`, `apply.go` | 按 hidden/readonly/protected/unknown internal key 阻断；apply 写回 existing item。 | `ConfigValidator` 只阻断硬错误；`ConfigWarningEngine` 产出 warning；patch changed-only。 | 保留 patch DTO 和 apply endpoint 形态。 | patch key 改为 canonical semantic key；apply 更新 snapshot item dirty/source/warnings。 | layer-protected 规则不再作为核心保存阻断。 | 3 |
| runtime API handlers | `internal/server/api/runtime_handlers.go` | create/patch 直接写 `launcher.image`、Docker/env/mount/health/command，并允许 raw `config_set` replacement。 | Runtime template write path 通过 `ConfigSnapshotBuilder` 和 semantic patch。 | 保留 create-from-template、checksum、source metadata、copy-on-create 流程。 | direct setters 改为 semantic patch/builder。 | 普通 raw `config_set` replacement 删除或 admin diagnostic-only。 | 2, 3 |
| node runtime API handlers | `internal/server/api/node_runtime_handlers.go`, `runtime_handlers.go` | enable/clone/patch 复制 ConfigSet，patch 可替换 raw config/source metadata；check/probe 独立。 | NBR 从 BackendRuntime snapshot copy-on-create，普通编辑 semantic patch。 | 保留 image picker/check/probe、needs_check、copy_on_create 元数据。 | `buildRuntimeConfigSnapshot` 迁入 builder；image 写为 `runtime.image_ref`。 | raw `config_set_json` / `source_metadata_json` 普通 patch 降级。 | 2, 3, 5 |
| deployment preview/create/patch/start handlers | `internal/server/api/deployment_lifecycle_handlers.go`, `deployment_preview_handlers.go` | Deployment 复制 NBR ConfigSet，同时存 `service_json`；preview/create 使用 `editable_config_patch`；patch 主要写 service/config_overrides；start 从 service_json + ConfigSet 拼 RunPlan。 | Deployment snapshot 是内部权威；preview/create/patch/start 都读写同一 semantic snapshot。 | 保留 deployment row、placement_json、task/start flow、host port conflict check。 | `service_json` 字段迁为 derived response/短期过渡；`config_overrides` 迁为 semantic patch。 | `service_json` 普通写 path、raw config overrides 降级；legacy keys direct patch 阻断。 | 2, 3, 4, 5 |
| RunPlan resolver | `internal/server/runplan/resolver.go`, `internal/server/api/configset_helpers.go` | ConfigSet `cli_arg` 和 `render.flag` 直接变 RunPlan 参数；resolver 保护 raw flags 并读 raw vars。 | `RunPlanResolver` 只接收 semantic deployment snapshot，经 `BackendAdapterMapping` 输出 CLI/env/docker/health/mount。 | 保留 DockerRunSpec、health、mount、agent spec 输出和 service arg overlay 经验。 | raw key/flag lookup 移入 adapter mapping；ConfigSet helpers 改成 semantic extraction。 | `backend.arg.*`、`render.flag` 作为 user config source 删除。 | 4 |
| BackendRuntimesPage | `web/src/pages/BackendRuntimesPage.vue` | 使用 `ConfigEditView` 编辑 BackendRuntime，JsonViewer 展示 raw config/source。 | 页面只传 object context，不做语义判断；warnings/dirty/tier 来自 DTO。 | 保留表格、detail、ConfigEditView、diagnostics。 | API payload 改为 semantic patch；source summary 显示 copied_from/dirty。 | Raw JsonViewer 只保留 diagnostic collapsed。 | 5 |
| RunnerConfigsPage | `web/src/pages/RunnerConfigsPage.vue` | NBR detail 使用 `ConfigEditView`；JsonViewer 展示 config/source/probe。 | NBR snapshot 自主可改，页面显示 probe warnings。 | 保留列表/detail/check/probe 体验。 | patch 改 canonical semantic keys；needs_check 由 semantic dirty/probe 触发。 | Raw config edit 入口不暴露普通用户。 | 5 |
| NodeRuntimeConfigWizard | `web/src/components/deployments/NodeRuntimeConfigWizard.vue` | 从 runtime template 获取 edit view；image 单独字段；提交 `editable_config_patch` 和 `image_ref`。 | 创建 NBR 时 builder 从 BackendRuntime 复制，并把 image 作为 `runtime.image_ref` snapshot item。 | 保留节点选择、runtime 选择、Docker image picker/check。 | image field 纳入 semantic patch 或 builder input。 | 页面不维护哪些字段可改。 | 2, 5 |
| DeploymentWizard | `web/src/components/deployments/DeploymentWizard.vue` | 持有 private `service`、`served_model_name`，preview/create 发送 `service_json`，并注入 `backend.common.served_model_name`。 | Wizard 通过 deployment semantic edit view 编辑 `deployment.host_port`、`service.container_port`、`deployment.served_model_name` 等 snapshot fields。 | 保留步骤、NBR/model selector、preview panel。 | `buildPayload` 改为 semantic changed-only patch + builder inputs。 | 删除 `backend.common.served_model_name` 注入。 | 5 |
| DeploymentServiceEditor | `web/src/components/deployments/DeploymentServiceEditor.vue` | 私有 host/container/served model form。 | 服务字段由 `ConfigEditView` 渲染，editor 可变为薄 wrapper 或移除。 | 可复用输入控件/校验提示文案。 | 迁到 semantic field widgets。 | 私有 service semantic 判断删除。 | 5 |
| DeploymentOverrideEditor | `web/src/components/deployments/DeploymentOverrideEditor.vue` | 使用 `ConfigEditView` 的 deployment layer，但命名为 override，输出 nested override shape。 | 改为 deployment snapshot editor；不以 override 为核心概念。 | 保留 ConfigEditView 接入和 preview 联动。 | payload 改 `editable_config_patch` 或 `config_patch` changed-only。 | `override` 命名和 nested `{ editable_config_patch }` 形态删除。 | 5 |
| BackendsPage | `web/src/pages/BackendsPage.vue` | BackendVersion editor 可 Add Parameter，默认 `backend.arg.fake_new_param`，保存 raw ConfigSet。 | BackendVersion UI 不暴露普通 runtime value；adapter mapping 由 catalog/dev-file 管理。 | 保留 backend/version CRUD、ConfigEditView diagnostic。 | Add Parameter 改成 semantic definition/admin catalog 工具或移除普通入口。 | `backend.arg.*` 普通 UI 编辑降级为 dev diagnostic。 | 5, 6 |
| ModelArtifactsPage | `web/src/pages/ModelArtifactsPage.vue` and model artifact APIs | 保存/display `default_context_length`、quantization、capability_set parameter defaults。 | Model facts 作为 deployment snapshot 默认/推荐来源，不直接变 backend arg。 | 保留 scan/edit/model facts UI。 | builder 从 ModelArtifact 提取 `model_runtime.context_length` recommendation。 | 页面不直接决定 deployment `max_model_len` flag。 | 2, 5 |
| ConfigEditView renderer | `web/src/components/config/ConfigEditView.vue`, `ConfigSection.vue`, `ConfigField.vue`, `web/src/utils/configEditView.ts` | 通用 renderer，按 sections/fields 生成 patch；field widgets 支持 env/device/mount/health/port。 | 保留 renderer，DTO 增加 owner/tier/semantic_key/copied_from/dirty/warnings/diagnostic。 | 保留组件结构和 widgets。 | patch 生成 changed-only canonical keys；section 由 tier/owner 驱动。 | Renderer 不内建业务语义判断。 | 3, 5 |
| JsonViewer diagnostics | `JsonViewer` usages in runtime/backend/deployment/preview pages | 展示 raw config/source/probe/runplan。 | 只作为 diagnostic-only、read-only、collapsed view。 | 保留排障价值。 | 标注 diagnostic tier，避免作为编辑入口。 | Raw snapshot replacement 不从 JsonViewer 衍生普通操作。 | 5 |
| RuntimeParameterEditor | `web/src/components/common/RuntimeParameterEditor.vue`, `web/src/utils/runtimeParameterViewModel.ts` | 文件存在但未作为普通入口被引用；view model 映射 `backend.common.served_model_name`、`backend.arg.max_model_len` 等。 | 不作为普通入口；如保留，仅 dev/diagnostic 查看 legacy 参数。 | 可复用部分 label/format 经验。 | 需要时迁移 display 到 ConfigEditView field widgets。 | 普通入口删除；legacy mapping 停止扩散。 | 5, 6 |

## Duplicate Semantic Groups

| Group | Current duplicated forms | Canonical semantic model |
| --- | --- | --- |
| host/listen_host/container_port/host_port | `backend.common.host`, `backend.common.port`, `launcher.listen_host`, `launcher.container_port`, `service.listen_host`, `service.container_port`, `service_json.host_port`, `service_json.container_port`, raw `host_port`, raw `container_port` | `service.listen_host`, `service.container_port`, `deployment.host_port`; `service_json` becomes derived response or short transition storage. |
| model_runtime parameters | `backend.arg.max_model_len`, raw `max_model_len`, `--max-model-len`, `context_length`, `backend.arg.gpu_memory_utilization`, `served_model_name` | `model_runtime.max_model_len`, `model_runtime.context_length` recommendation, `model_runtime.gpu_memory_utilization`, `deployment.served_model_name`; flags only in adapter mapping. |
| health check | catalog `default_health_check`, `runtime.health`, request `health_check`, RunPlan health, node/GPU `health` metrics | `runtime.health.*` for runtime service health; node/GPU health remains metrics/diagnostic, not config. |
| model mount | catalog `default_model_mount`, `runtime.model_mount`, RunPlan model host/container vars | `runtime.model_mount.container_path` and `runtime.model_mount.readonly`; ModelLocation owns host path/source. |
| runtime env | `runtime.env`, `env`, `env_schema`, `env_overrides`, env-targeted parameters | `runtime.env` snapshot object; deployment env additions are snapshot items, not ad hoc overrides. |
| docker options / device binding | `launcher.docker_options`, `docker_options`, `devices`, `optional_devices`, `group_add`, scheduler/GPU device terms | `docker.*` for Docker host config/device mappings; scheduler placement keys remain scheduling layer. |
| backend capabilities / runtime requirements | `backend.capabilities`, `backend.supported_config_items`, model `capability_set_json`, probe backend match, supported model formats | `backend_capability.*` for backend support/mapping; `model.capability.*` for model facts and recommendations. |

## Canonical Semantic Key Table

| Semantic key | Owner | Value type | Copied to | Default/recommended source | Hard validation | Warning rules | Resolver mapping | Display tier | Legacy keys to remove/normalize |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `runtime.image_ref` | runtime_environment | string | BackendRuntime, NBR | Runtime YAML image candidates; node image picker at NBR create | required for Docker runtime; non-empty string | image not inspected or probe stale | Docker image | required | `launcher.image`, request `image_ref` |
| `runtime.command` | runtime_environment | string array | BackendRuntime, NBR | Backend/runtime YAML command/profile | array parseable as argv | raw command edit may break backend adapter expectations | Docker Cmd/process args base | advanced/diagnostic | `launcher.command`, `default_args` |
| `runtime.entrypoint` | runtime_environment | string array | BackendRuntime, NBR | Backend/runtime YAML entrypoint | array parseable as argv | overriding image entrypoint may break startup | Docker Entrypoint | advanced/diagnostic | `launcher.entrypoint`, `default_entrypoint` |
| `runtime.env` | runtime_environment | object string map | BackendRuntime, NBR, Deployment snapshot if copied | Runtime YAML env/env_schema | keys strings; values scalar/stringable | secret-like values redacted; unsupported env warning | Docker env | common/advanced | direct `env`, `env_overrides` |
| `service.listen_host` | runtime_service | string | BackendRuntime, NBR, Deployment | BackendVersion default host or adapter default | valid host/IP token | non-`0.0.0.0` may make container unreachable | adapter emits `--host` or env if supported | required/common | `backend.common.host`, `launcher.listen_host` |
| `service.container_port` | runtime_service | integer | BackendRuntime, NBR, Deployment | BackendVersion default port; runtime port defaults | 1-65535 integer | image exposed port or health mismatch | adapter emits service port flag; Docker container port | required/common | `backend.common.port`, `launcher.container_port`, `service_json.container_port` |
| `deployment.host_port` | deployment_exposure | integer | Deployment | User input or allocator recommendation | 1-65535 integer; conflict check before start | privileged/unusual port warning | Docker host port binding | required/common | raw `host_port`, `service_json.host_port` |
| `deployment.served_model_name` | deployment_exposure | string | Deployment | Artifact name | non-empty when adapter requires; format safe | differs from artifact name | adapter emits served-name flag/env when supported | common | `backend.common.served_model_name`, raw `served_model_name` |
| `model_runtime.context_length` | model_artifact | integer | ModelArtifact recommendation, Deployment source snapshot | Scanner/HF/GGUF metadata, `default_context_length` | positive integer if present | unknown or inconsistent model metadata | default/recommendation input, not main deployment flag | recommended/diagnostic | raw `context_length`, artifact default column as source fact |
| `model_runtime.max_model_len` | deployment_runtime | integer | Deployment | From `model_runtime.context_length` recommendation or user edit | positive integer | above context length; estimated VRAM risk | vLLM `--max-model-len`, SGLang `--context-length`, llama.cpp `--ctx-size` | common/advanced | `backend.arg.max_model_len`, raw `max_model_len`, `--max-model-len` |
| `model_runtime.gpu_memory_utilization` | deployment_runtime | number | Deployment | Backend recommendation | number in chosen normalized range | high value OOM risk | backend adapter flag/env | advanced | `backend.arg.gpu_memory_utilization`, raw key |
| `runtime.health.path` | runtime_service | string | BackendRuntime, NBR, Deployment | Backend/runtime YAML health | absolute HTTP path when enabled | missing health check for service backend | Agent health check path | recommended | `runtime.health.path`, `health_check.path` |
| `runtime.health.timeout_seconds` | runtime_service | integer | BackendRuntime, NBR | YAML health defaults | positive integer | too low causes false failures | Agent health check timeout | advanced | `runtime.health.timeout`, `health_check.timeout_seconds` |
| `runtime.health.interval_seconds` | runtime_service | integer | BackendRuntime, NBR | YAML health defaults | positive integer | too frequent adds load | Agent health check interval | advanced | `runtime.health.interval`, `health_check.interval_seconds` |
| `runtime.model_mount.container_path` | runtime_environment | string | BackendRuntime, NBR | Runtime/backend YAML model mount | absolute path; no traversal | command template mismatch | Docker volume target and resolver vars | common | `runtime.model_mount`, `default_model_mount.container_path` |
| `runtime.model_mount.readonly` | runtime_environment | boolean | BackendRuntime, NBR | YAML default | boolean | writable model mount risk | Docker volume readonly | advanced | `runtime.model_mount.readonly` |
| `docker.shm_size` | runtime_environment | string | BackendRuntime, NBR | Runtime YAML Docker options | parseable Docker size | too small/large warning | Docker HostConfig | common | `launcher.docker_options.shm_size` |
| `docker.ipc_mode` | runtime_environment | string enum | BackendRuntime, NBR | Runtime YAML Docker options | enum/string | `host` isolation warning | Docker HostConfig IPCMode | advanced | `launcher.docker_options.ipc_mode` |
| `docker.privileged` | runtime_environment | boolean | BackendRuntime, NBR | Runtime YAML Docker options | boolean | privileged security warning | Docker HostConfig Privileged | advanced | `launcher.docker_options.privileged` |
| `docker.network_mode` | runtime_environment | string | BackendRuntime, NBR | Runtime YAML Docker options | string | host network conflicts with port mapping | Docker HostConfig NetworkMode | advanced | `launcher.docker_options.network_mode` |
| `docker.devices` | runtime_environment | array object | BackendRuntime, NBR | Runtime YAML Docker options | object shape; path strings parseable | raw device security/missing optional warning | Docker device mappings | advanced | `launcher.docker_options.devices`, top-level `devices` |
| `docker.optional_devices` | runtime_environment | array object/string | BackendRuntime, NBR | Runtime YAML Docker options | array parseable | missing optional device warning only | Docker device mappings if present | advanced | `launcher.docker_options.optional_devices` |
| `docker.group_add` | runtime_environment | string array | BackendRuntime, NBR | Runtime YAML Docker options | array strings | group may not exist on host | Docker group_add | advanced | `launcher.docker_options.group_add` |
| `scheduler.accelerator_ids` | scheduler_resource | string array | Deployment placement | User placement/scheduler | valid IDs when hard-bound | stale/occupied GPU warning | Scheduler lease and visible device env | common | `placement_json.accelerator_ids` |
| `backend_capability.resolver_mapping` | backend_capability | object | BackendVersion/catalog only | Catalog/dev file | known semantic keys only | adapter missing mapping warning | `BackendAdapterMapping` source | diagnostic/admin | `render.flag`, `cli_name`, raw args schema as user config |

## Reuse Plan

| Existing code/data | Reuse decision | Required adjustment |
| --- | --- | --- |
| `ConfigSet` / `config_set_json` | Reuse as the snapshot container. It already persists across BackendVersion, BackendRuntime, NodeBackendRuntime and Deployment. | Item identity changes to canonical semantic key. Extend item metadata with owner, copied_from, source_snapshot, copied_at, dirty, warnings and diagnostic fields. |
| `source_metadata_json` | Reuse as object-level source/copy/probe metadata. Current handlers already write `copy_semantics`, `copy_boundary`, source IDs and checksums. | Extend with snapshot-level copy lineage and summary warnings. Keep per-item lineage inside ConfigSet items so downstream edits remain local. |
| `ConfigEditView` / `ConfigSection` / `ConfigField` | Reuse as renderer. Current components already support env, device, mount, health and port-style widgets. | DTO must carry semantic key, owner, tier, copied_from, dirty, warnings and diagnostic. Renderer emits changed-only canonical patch and stops carrying business rules. |
| BackendRuntime -> NBR -> Deployment copy-on-create | Reuse the existing object boundary and source metadata intent in runtime/node/deployment handlers. | Move construction into `ConfigSnapshotBuilder`; ensure copied item belongs to destination snapshot after creation. |
| RunPlan Docker/health/mount output structures | Reuse. `runplan.Resolve` and deployment start already produce useful Docker, service, mount and health data for agent task specs. | Replace raw key/flag input with semantic resolver output and backend adapter mapping. |
| `JsonViewer` | Reuse as diagnostic-only raw state view. | Keep collapsed/read-only; never make raw JSON replacement a normal user path. |
| Node Docker image picker/check/probe | Reuse. NodeRuntime wizard and check/probe APIs are valuable operational flows. | Feed selected image into `runtime.image_ref`; feed probe results into warning engine/diagnostic metadata. |
| Model scan facts | Reuse `default_context_length`, quantization and capability data. | Treat as model-owned recommendation inputs to Deployment snapshot, not as direct backend CLI parameters. |

## Migration Plan by Existing Code

| Current code | Migration action |
| --- | --- |
| `MaterializeBackendVersion` / `MaterializeBackendRuntime` in `internal/server/catalog/loader.go` | Convert materialization to call `SemanticConfigNormalizer` and registry defaults. Existing legacy YAML keys are accepted as input, but persisted ConfigSet items become canonical semantic keys. |
| `addArgConfigItems` generating `backend.arg.*` | Stop creating user config items from CLI args. Convert args schema and `render.flag` into `BackendAdapterMapping` entries keyed by canonical semantic key. Unknown raw args are diagnostic catalog warnings. |
| `configedit/project.go` alias projection | Move alias resolution into normalizer. Projector consumes already-canonical snapshots and only builds display DTO. Alias conflicts become warnings or hard errors depending on whether values conflict and whether direct legacy patch was attempted. |
| `configedit/validate.go` and `apply.go` | Replace hidden/readonly/layer protection as core save policy with hard-error validator plus warning engine. Apply updates canonical item value, dirty flag, source snapshot and item warnings. |
| `DeploymentWizard` / `DeploymentServiceEditor` page-private service model | Move service fields into deployment edit view fields: `deployment.host_port`, `service.container_port`, `deployment.served_model_name`, optional health fields. |
| `DeploymentWizard.buildPayload` injecting `backend.common.served_model_name` | Remove injection. Builder creates or updates `deployment.served_model_name` and adapter maps it to CLI only when backend supports it. |
| BackendVersion Add Parameter UI in `BackendsPage.vue` | Remove normal user flow for `backend.arg.*`. If kept, it is admin/dev catalog mapping tooling that writes semantic definition + adapter mapping, not runtime value. |
| RunPlan resolver raw key / CLI flag mapping | Introduce semantic `RunPlanResolver` input. Existing resolver output remains, but raw `max_model_len`, `served_model_name`, `--max-model-len` lookup is replaced by adapter mapping. |
| API raw `config_set` replacement / `service_json` write path | Ordinary APIs accept semantic patch/builder inputs only. Raw replacement is admin diagnostic-only or removed. `service_json` becomes derived response or short transition storage generated from semantic snapshot. |

## Removal / Diagnostic-only Plan

| Legacy logic | Decision |
| --- | --- |
| `backend.common.*` | Not a long-term storage key. Normalize to `service.listen_host`, `service.container_port`, `deployment.served_model_name` where applicable. |
| `backend.arg.*` | Not a long-term user config key. Use only as legacy normalizer input or diagnostic evidence. |
| `launcher.listen_host` / `launcher.container_port` | Not a user config key. Normalize to service semantic keys. |
| DeploymentWizard injection of `backend.common.served_model_name` | Delete. It conflicts with `deployment.served_model_name` ownership. |
| DeploymentServiceEditor private host/container/served model semantics | Migrate to semantic projection. Component may remain as a thin field group only if it consumes projected fields. |
| RuntimeParameterEditor / `runtimeParameterViewModel.ts` | Remove from normal entrypoints. If retained, mark dev/diagnostic-only because it encodes legacy key mappings. |
| Raw `config_set` replacement in ordinary APIs | Remove or restrict to admin diagnostic endpoints. Normal users patch canonical semantic keys only. |
| Projection-time alias merge | Keep only as legacy input defense in normalizer. It must not be the long-term main mechanism. |
| `render.flag` as user-facing meaning | Move to `BackendAdapterMapping`. It remains resolver metadata, not field identity. |

## Program Architecture Plan

| Module | Location | Responsibility |
| --- | --- | --- |
| `SemanticConfigRegistry` | `internal/server/semanticconfig/registry.go` | Register canonical keys, owner, value type, default tier, copy policy, display metadata, adapter support and legacy aliases. Enforce one semantic key/owner per parameter. |
| `SemanticConfigNormalizer` | `internal/server/semanticconfig/normalizer.go` | Convert legacy catalog/API/config keys into canonical keys, detect conflicts, reject direct legacy patch, and produce diagnostic warnings. |
| `ConfigSnapshotBuilder` | `internal/server/semanticconfig/snapshot.go` | Build BackendVersion -> BackendRuntime, BackendRuntime -> NBR, and NBR + ModelArtifact + service input -> Deployment snapshots with copy lineage. |
| `ConfigProjector` | `internal/server/semanticconfig/projector.go` or wrapping `internal/server/configedit` | Project semantic snapshots into ConfigEditView DTOs by object kind, owner and display tier. |
| `ConfigWarningEngine` | `internal/server/semanticconfig/warnings.go` | Produce warnings for high risk values, unsupported backend options, stale probe/image data, model length/memory risks and dirty snapshot state. |
| `ConfigValidator` | `internal/server/semanticconfig/validator.go` | Hard errors only: type, required, format, enum, path/port parse, unknown canonical key, direct legacy key patch. |
| `RunPlanResolver` / `BackendAdapterMapping` | `internal/server/semanticconfig/resolver.go`, `internal/server/runplan` integration | Convert canonical deployment snapshot to backend-specific CLI/env/docker/health/mount output. |
| ConfigEditView renderer changes | `web/src/components/config/**`, `web/src/utils/configEditView.ts` | Render owner/tier/source/dirty/warnings; emit changed-only canonical patches; remove page-private semantic decisions. |

## Batch Execution Plan

### Batch 0: Plan freeze with current-code gap matrix

| Field | Plan |
| --- | --- |
| Goal | Freeze current-code-based plan and use it as execution contract. |
| Current code touched | `docs/reports/phase-3/semantic-config-governance/10-codex-review-and-execution-plan.md` only. |
| Reuse | Existing audit docs and current code evidence. |
| Migrate | No code migration. |
| Remove / diagnostic-only | No code removal. |
| Tests | `git diff --check`; if staged, `git diff --cached --check`. |
| Acceptance | Plan answers target, current behavior, gap, reuse, migration, removal, batches, and open questions. |
| Risks | The plan may need adjustment if code changes before Batch 1 starts; re-run keyword audit before implementation. |

### Batch 1: Semantic registry + normalizer

| Field | Plan |
| --- | --- |
| Goal | Add canonical semantic key registry and legacy-key normalizer without changing user flows. |
| Current code touched | `configs/config-registry/items.yaml`, optional new semantic registry YAML, `internal/server/catalog/**`, new `internal/server/semanticconfig/**`, tests under matching packages. |
| Reuse | Reuse item labels, value types and current config registry content as seed metadata. |
| Migrate | Map `backend.common.*`, `backend.arg.*`, `launcher.docker_options.*`, `runtime.health`, `runtime.model_mount`, `runtime.env` into canonical definitions. |
| Remove / diagnostic-only | Mark legacy keys as aliases/diagnostic input only; no UI removal yet. |
| Tests | Unit tests for registry uniqueness, alias normalization, conflict handling, direct legacy patch hard error. |
| Acceptance | A legacy ConfigSet can normalize to canonical keys with warnings; duplicate semantic owner definitions fail tests. |
| Risks | Incomplete alias coverage can break catalog reload; include fixtures from current backend catalog. |

### Batch 2: Config snapshot builder

| Field | Plan |
| --- | --- |
| Goal | Centralize copy-on-create snapshot creation for BackendRuntime, NodeBackendRuntime and Deployment. |
| Current code touched | `internal/server/api/runtime_handlers.go`, `node_runtime_handlers.go`, `deployment_lifecycle_handlers.go`, `deployment_preview_handlers.go`, `artifact_handlers.go`, new snapshot builder tests. |
| Reuse | Existing `config_set_json` storage, source metadata fields, checksum update, copy_on_create source IDs, model facts. |
| Migrate | `buildRuntimeConfigSnapshot`, `deploymentConfigSnapshotFromNBR`, direct service/model injection move into `ConfigSnapshotBuilder`. |
| Remove / diagnostic-only | Stop creating new persistent `backend.common.*` and `backend.arg.*` items on new snapshots. |
| Tests | Unit/integration handler tests for BackendRuntime -> NBR -> Deployment copy lineage and dirty independence. |
| Acceptance | Downstream snapshot edits do not mutate upstream; copied_from/source_snapshot exists; model recommendations copy into Deployment. |
| Risks | Existing DB rows with legacy keys require catalog reload or DB rebuild path; this is acceptable for this phase. |

### Batch 3: Projector + warning engine + validator + changed-only patch

| Field | Plan |
| --- | --- |
| Goal | Make ConfigEdit API the single semantic edit path with hard validation and non-blocking warnings. |
| Current code touched | `internal/server/configedit/**`, config edit API handlers, `web/src/components/config/**`, `web/src/utils/configEditView.ts`, `web/src/api/configEdit.ts`. |
| Reuse | Current EditView DTO shape, widgets, patch endpoint and renderer. |
| Migrate | Projection reads semantic snapshot; apply accepts canonical changed-only patch; warnings returned in response and projected fields. |
| Remove / diagnostic-only | Hidden/readonly/layer protection no longer decides semantic editability; raw internal key edits are blocked unless diagnostic. |
| Tests | Projector tests per object kind/tier; validator tests for hard errors; warning tests for risky but saveable values; frontend patch generation tests if available. |
| Acceptance | Pages no longer need private semantic allowlists; risky values save with warnings; invalid type/required/enum/port/path fail. |
| Risks | UI field identity changes can break patch roundtrip; add fixtures for BackendRuntime, NBR and Deployment views. |

### Batch 4: RunPlan resolver + backend adapter mapping

| Field | Plan |
| --- | --- |
| Goal | Resolve deployment semantic snapshot into backend-specific CLI/env/docker without exposing CLI flags as user keys. |
| Current code touched | `internal/server/runplan/**`, `internal/server/api/configset_helpers.go`, deployment start/preview handlers, catalog adapter mapping fixtures. |
| Reuse | Existing RunPlan output structs, Docker/health/mount resolution, agent task conversion. |
| Migrate | Raw parameter defs/values and `render.flag` logic move into `BackendAdapterMapping`; `buildVarMap` reads semantic keys. |
| Remove / diagnostic-only | `backend.arg.*` and raw CLI flag lookup are no longer resolver inputs. |
| Tests | Golden RunPlan tests for vLLM/SGLang/llama.cpp equivalent commands; host/port conflict and service mapping tests. |
| Acceptance | Same or intentionally changed run command is produced from canonical semantic snapshot; unsupported semantic keys warn rather than silently disappearing. |
| Risks | Backend-specific flag differences are easy to regress; adapter golden fixtures must be explicit. |

### Batch 5: Web entrypoint migration

| Field | Plan |
| --- | --- |
| Goal | Remove page-private parameter semantics from normal UI entrypoints. |
| Current code touched | `web/src/pages/BackendRuntimesPage.vue`, `RunnerConfigsPage.vue`, `BackendsPage.vue`, `ModelArtifactsPage.vue`, `web/src/components/deployments/NodeRuntimeConfigWizard.vue`, `DeploymentWizard.vue`, `DeploymentServiceEditor.vue`, `DeploymentOverrideEditor.vue`, `DeploymentPreviewPanel.vue`, `web/src/api/**`. |
| Reuse | Current page structure, selectors, image picker/check/probe, ConfigEditView, JsonViewer diagnostics. |
| Migrate | Deployment service fields into semantic edit view; image into `runtime.image_ref`; model facts into default/warning display; backend parameter add into dev/admin mapping workflow or remove from normal UI. |
| Remove / diagnostic-only | Remove `backend.common.served_model_name` injection, private service model semantics, normal `RuntimeParameterEditor`, normal raw JSON config editing. |
| Tests | Web typecheck/build and focused component tests if present; manual smoke for runtime template, NBR wizard, deployment preview/create. |
| Acceptance | Pages pass context only; semantic labels/tier/warnings/validation come from API DTO; Deployment preview/create payload has canonical patch and no direct legacy keys. |
| Risks | UX changes can expose too many advanced fields; use display tiers and collapsed diagnostic sections. |

### Batch 6: Catalog cleanup + DB rebuild + E2E closeout

| Field | Plan |
| --- | --- |
| Goal | Remove legacy storage keys from seed catalog and close the semantic migration with rebuild/reload verification. |
| Current code touched | `configs/backend-catalog/**`, `configs/config-registry/items.yaml`, seed/reload code, docs/runbooks, E2E fixtures. |
| Reuse | Existing catalog reload and seed DB flows. |
| Migrate | YAML entries switch to canonical semantic keys and adapter mapping blocks. Existing dev DB can be rebuilt. |
| Remove / diagnostic-only | Delete old `backend.common.*`, `backend.arg.*`, user-facing `launcher.listen_host/container_port` from catalog seeds; keep normalizer tests only. |
| Tests | Catalog reload, DB rebuild, API smoke, deployment preview/start golden tests, web build. |
| Acceptance | Fresh DB/catalog contains canonical keys only; legacy keys appear only in normalizer tests/diagnostics; E2E preview/start works. |
| Risks | Rebuild invalidates local dev rows; document rebuild command and expected data loss for this phase. |

## Validation Matrix

| Requirement | Planned verification |
| --- | --- |
| One semantic key and owner per parameter | Registry unit test rejects duplicate owner/key definitions. |
| CLI flag not user config key | Catalog and UI tests assert no normal ConfigEdit field key starts with `backend.arg.`. |
| Copy-on-create snapshot independence | Handler tests edit Deployment snapshot and assert NBR/BackendRuntime snapshots are unchanged. |
| Warnings do not block save | Validator/warning tests save high-risk Docker/model values and return warnings. |
| Hard errors block save | Tests cover invalid type, missing required, enum, port/path parse, unknown canonical key, direct legacy key patch. |
| Pages have no private semantic logic | Static search for forbidden injections/key families in normal pages plus component/API tests. |
| RunPlan uses adapter mapping | Golden tests assert semantic keys resolve to backend-specific flags/env. |
| Raw JSON diagnostic only | API/UI tests assert normal roles cannot replace raw `config_set_json`; JsonViewer remains read-only. |

## Open Questions

| Question | Current code impact | Recommended answer | Reason | Alternative if Codex disagrees |
| --- | --- | --- | --- | --- |
| Q1. Should `deployment.host_port` continue to exist in `service_json`? | `deployment_lifecycle_handlers.go`, `deployment_preview_handlers.go` and `DeploymentWizard.vue` currently read/write `service_json.host_port` and `service_json.container_port`; RunPlan start combines `service_json` with ConfigSet. | Use semantic Deployment snapshot as internal authority. `service_json` should be derived response or short transition storage generated from semantic snapshot. Allow DB rebuild/catalog reload rather than complex compatibility. | A second writable service object is the main source of drift between preview/create/patch/start. | If disagreement, keep `service_json` only as a generated cache with checksum and reject direct writes when it differs from semantic snapshot. |
| Q2. Should `runtime.command` be directly user-editable? | Runtime handlers and catalog loader persist `launcher.command`; ConfigEdit taxonomy currently protects some deployment-layer command edits. | Ordinary users should not edit raw command. Use `process_start_profile` or template selection for normal UI; raw command belongs in advanced/admin diagnostic tier. | Raw argv edits bypass adapter mapping and can invalidate service/health/model mount assumptions. | If disagreement, allow editing only on BackendRuntime/NBR with explicit warning and forced re-probe before deployment start. |
| Q3. Should `model_runtime.context_length` be Deployment-editable? | `ModelArtifactsPage.vue` and artifact APIs store context length facts; RunPlan/raw params also use `max_model_len` and sometimes `context_length`. | `context_length` is model fact/recommendation source. Deployment edits `model_runtime.max_model_len`; context length drives default and warning. | The model's physical/context fact should not become a mutable deployment runtime knob. | If disagreement, allow deployment-local context override as `model_runtime.context_length_override`, but keep scanner fact immutable. |
| Q4. Should BackendVersion UI expose adapter mapping? | `BackendsPage.vue` currently allows Add Parameter with `backend.arg.*`, which mixes catalog authoring and runtime values. | This phase should keep adapter mapping in catalog/dev-file, not ordinary UI. | Mapping is backend-authoring metadata; exposing it as normal UI will recreate raw CLI key leakage. | If disagreement, expose only an admin/dev mapping editor that writes semantic key + mapping, never `backend.arg.*` values. |
| Q5. Should GPU binding enter `ConfigEditView`? | Deployment uses `placement_json`; Docker options contain `devices` and `optional_devices`; RunPlan also needs visible devices. | Scheduler placement stays in scheduler layer as `scheduler.accelerator_ids`; Docker device mapping belongs to `docker.*` semantic config. | GPU allocation and Docker device mount are related but have different owners and validation. | If disagreement, show placement summary inside ConfigEditView as diagnostic/read-only fields while actual binding remains in placement editor. |

## Final Planning Status

- Status: REVIEW_PLAN_READY
- Found entrypoint areas: 19 in the gap matrix, covering catalog, API, RunPlan, web pages/components, diagnostics and legacy parameter editor.
- Found duplicate semantic groups: 7.
- Recommended new modules: `SemanticConfigRegistry`, `SemanticConfigNormalizer`, `ConfigSnapshotBuilder`, `ConfigProjector`, `ConfigWarningEngine`, `ConfigValidator`, `RunPlanResolver`, `BackendAdapterMapping`, ConfigEditView renderer changes.
- Recommended batches: Batch 0 through Batch 6 as listed above.
- Codex independent opinion: agree with semantic snapshot as authority, adapter mapping outside user config, warning-first validation, and page-thin rendering. The strongest current-code evidence is that the same value is now represented by `service_json`, `backend.common.*`, `backend.arg.*`, raw RunPlan parameter names and page-private Vue state.
