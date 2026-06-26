# 01 - 当前源码现状与问题边界

## 1. 已有基础

### 1.1 ConfigItem 已具备 schema 基础

`internal/server/catalog/types.go` 中 `ConfigItem` 已有：

```go
Code
Category
Kind
Type
Required
Visibility
Readonly
Advanced
Value
DefaultValue
Enabled
Render
Order
Constraints
SupportLevel
Source
Extensions
```

这说明不需要从零发明字段模型。问题是这些字段目前仍直接服务于 ConfigSet 内部项，而不是明确的外部编辑视图。

### 1.2 各层已经保存 config_set_json

DB 中以下对象都有 `config_set_json`：

- `inference_backends`
- `backend_versions`
- `backend_runtimes`
- `node_backend_runtimes`
- `model_deployments`

这是实现统一编辑模型的基础。ConfigSet 应继续作为内部 canonical storage，不应被页面直接改成多套结构。

### 1.3 RuntimeParameterEditor 已能动态渲染

`web/src/components/common/RuntimeParameterEditor.vue` 当前已经遍历 `config_set.items`，支持 required、enabled、visibility、readonly、advanced，并输出 `config_set` 与 `config_overrides.parameter_values`。

## 2. 当前问题

### 2.1 仍直接暴露内部 key

当 catalog 没有完整 render metadata 时，用户会看到：

```text
launcher.docker_options
launcher.image
runtime.env
runtime.model_mount
runtime.health
backend.arg.xxx
```

这些是内部 key，不是用户应看到的编辑语言。

### 2.2 object/json 参数仍作为普通表单项出现

例如 `launcher.docker_options` 可能包含 devices、group_add、ipc_mode、privileged、security_options、shm_size、ulimits 等。普通用户需要看到结构化字段，不应看到整块 JSON textarea。

### 2.3 多个页面各自直接编辑 config_set

当前至少这些页面/组件在直接使用 RuntimeParameterEditor 或直接 PATCH config_set：

- `BackendsPage.vue`
- `BackendRuntimesPage.vue`
- `NodeRuntimeConfigWizard.vue`
- `DeploymentOverrideEditor.vue`

这会导致每个页面各自决定分组、label、checkbox、object/json 展示、保存 payload，最终又回到“每个页面硬编码”。

### 2.4 runtimeDisplay 仍偏技术展示

`web/src/utils/runtimeDisplay.ts` 当前 displayName 仍类似 `${vendor}.${backendId}`，普通用户看到的是技术组合而不是“vLLM / MetaX”这类产品化名称。

### 2.5 clone runtime 缺少命名流程

`BackendRuntimesPage.vue` 当前 clone 只是 `POST /backend-runtimes/{id}/clone`，没有让用户输入 display_name/name。

## 3. 根因

根因不是缺一个更复杂的 Vue 组件，而是缺少投影层：

```text
内部 ConfigSet item
  -> 领域字段归类
  -> 用户可编辑字段
  -> 用户修改 patch
  -> 回写 ConfigSet item/value/enabled
```

因此建议新增 `configedit` 层，而不是继续在 RuntimeParameterEditor 或各页面里堆 if/else。
