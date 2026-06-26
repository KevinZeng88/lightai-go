# 02. Cross Entrypoint Audit Scope

## 1. 审计目标

本次不是修一个页面，而是审计所有参数输入入口，确保它们全部复用同一套 Semantic Config 基础能力。

禁止模式：

```text
BackendRuntimesPage 自己判断哪些字段显示
RunnerConfigsPage 自己判断哪些字段显示
DeploymentWizard 自己判断哪些字段显示
BackendsPage 自己判断哪些字段显示
```

正确模式：

```text
SemanticConfigRegistry
  -> ConfigSnapshotBuilder
  -> ConfigProjector
  -> ConfigValidator / WarningEngine
  -> ConfigResolver
  -> ConfigEditView
所有页面只声明 object_kind / mode / display context
```

## 2. 必须审计的入口

### 2.1 Backend / BackendVersion 页面

文件：

```text
web/src/pages/BackendsPage.vue
internal/server/api/backend*_handlers.go
internal/server/configedit/*
configs/backend-catalog/**
```

定位：后端能力、参数定义、adapter mapping 管理入口。

应该做：

- 定义 semantic parameter schema。
- 定义 backend 支持哪些 semantic keys。
- 定义 semantic key 到 CLI/env 的映射。
- 只作为管理员/开发者入口。

不应该做：

- 存具体运行值。
- 让普通用户配置 Docker image / host_port / model runtime value。
- 生成长期 `backend.arg.*` 用户配置项。

### 2.2 BackendRuntime / 运行模板页面

文件：

```text
web/src/pages/BackendRuntimesPage.vue
web/src/api/runtimes.ts
internal/server/api/runtime_handlers.go
internal/server/configedit/*
configs/backend-catalog/runtimes/**
```

定位：运行环境模板。

应显示普通区：

- `runtime.image_ref`
- `service.listen_host`
- `service.container_port`
- `launcher.command` 或新的 `runtime.command`
- `docker.shm_size`
- `docker.ipc_mode`
- `runtime.model_mount.container_path`
- `runtime.health.path`，有默认值时
- 常用 env

应放高级折叠：

- `docker.privileged`
- `docker.security_options`
- `docker.ulimits`
- `docker.devices`
- `docker.optional_devices`
- `docker.group_add`
- `docker.network_mode`
- `docker.uts_mode`
- `launcher.entrypoint` 或新的 `runtime.entrypoint`
- 其他 env

不应出现：

- `model_runtime.max_model_len`
- `deployment.host_port`
- `backend.common.*`
- `launcher.listen_host`
- `launcher.container_port`
- raw `backend.arg.*` 用户字段

### 2.3 NodeBackendRuntime / 添加节点运行配置

文件：

```text
web/src/components/deployments/NodeRuntimeConfigWizard.vue
web/src/pages/RunnerConfigsPage.vue
internal/server/api/*node*runtime*handlers.go
internal/server/configedit/*
```

定位：节点上启用运行模板后的本地配置快照。

应显示：

- 节点 Docker image 选择 / 手工输入。
- runtime image copied snapshot。
- 节点本地 Docker 参数。
- 节点本地设备绑定。
- 节点本地 env。
- 节点本地 health 覆盖。

不应出现：

- `model_runtime.max_model_len`
- `model_runtime.gpu_memory_utilization`
- `deployment.host_port`
- `deployment.served_model_name`

### 2.4 Deployment Wizard

文件：

```text
web/src/components/deployments/DeploymentWizard.vue
web/src/components/deployments/DeploymentOverrideEditor.vue
web/src/components/deployments/DeploymentServiceEditor.vue
web/src/components/deployments/DeploymentPreviewPanel.vue
internal/server/api/deployment*_handlers.go
internal/server/runplan/**
```

定位：部署配置快照和最终运行计划。

应显示：

- 模型选择。
- NBR 选择。
- `deployment.host_port`。
- `deployment.served_model_name`。
- Deployment 快照中的 `model_runtime.*` 参数，默认折叠为高级。
- 本次部署级 resource/model warnings。

不应出现：

- BackendVersion schema 编辑。
- runtime source metadata。
- backend capabilities raw。
- 基础 Docker image 修改，除非明确作为高级拷贝参数存在。

### 2.5 Model Library / Model Artifact

文件需由 Codex 搜索确认，至少包括：

```text
web/src/pages/*Model*.vue
internal/server/api/model*_handlers.go
internal/server/model*/**
```

定位：模型资产、模型默认参数、建议值、模型能力。

应持有：

- model format。
- model path/location。
- context length / max supported tokens，如能发现。
- dtype / quantization metadata。
- recommended model_runtime defaults。

不应持有：

- Docker image。
- host_port。
- node devices。

### 2.6 RunPlan / Resolver

文件需由 Codex 搜索确认，至少包括：

```text
internal/server/runplan/**
internal/server/runtime/**
internal/agent/**docker**
```

定位：把 semantic config snapshot 解析成实际运行计划。

要求：

- 不读取旧 alias key 作为主要配置。
- 从 semantic keys 生成 CLI/env/docker args。
- 对 missing required / incompatible values 给硬错误或 warning。
- 对 warnings 贯穿到 preview / check / deploy。

## 3. 全仓库搜索关键字

Codex 必须搜索：

```text
backend.common.host
backend.common.port
launcher.listen_host
launcher.container_port
service.listen_host
service.container_port
host_port
container_port
backend.arg.
max_model_len
max-model-len
context_length
served_model_name
gpu_memory_utilization
runtime.env
launcher.env
docker_options
devices
optional_devices
group_add
health
model_mount
RuntimeParameterEditor
ConfigEditView
JsonViewer
config_set_json
source_metadata_json
```

## 4. 输出要求

Codex 第一阶段必须输出：

1. 所有参数输入入口列表。
2. 每个入口当前使用的组件/API。
3. 每个入口存在的重复建模、字段泄露、保存风险。
4. 参数 key 归属表。
5. 需要删除/替换/normalize 的旧 key 列表。
6. 需要新增的 common 程序模块。
7. 分批实施计划。

## 5. 不允许的做法

- 不允许只修 `BackendRuntimesPage.vue`。
- 不允许只补 i18n key。
- 不允许继续靠 alias 在 UI 层遮丑。
- 不允许每个页面单独维护字段显示列表。
- 不允许保留长期 `backend.arg.*` 作为普通用户配置 key。
- 不允许把 backend CLI flag 名直接暴露为参数 owner。
