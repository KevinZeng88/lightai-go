# 03 - Copy-on-Create 数据模型与 API 设计

## 1. 核心原则

必须严格执行：

```text
不是继承，不是运行时 merge，不是查询时动态拼接。
每一层创建时复制上一层完整参数，再添加或覆盖本层参数。
复制完成后，下游与上游脱钩。
```

一句话：

```text
上游只作为创建来源，不作为运行依赖。
```

---

## 2. 目标层级

```text
Backend
  ↓ copy-on-create
BackendVersion
  ↓ copy-on-create
BackendRuntime / RuntimeTemplate
  ↓ copy-on-create
NodeBackendRuntime
  ↓ copy-on-create
Deployment
```

每一层都拥有独立的：

```text
config_set_json
source_metadata_json
checksum / source_config_hash
created_at / copied_at
```

---

## 3. 每层职责

## 3.1 Backend

职责：

- 描述推理后端类型。
- 保存最基础、硬件无关的通用参数。
- 例如：
  - backend.capabilities
  - runtime.health default
  - protocol
  - model format support

不应该包含：

- vendor image
- docker options
- GPU device binding
- vendor env
- runtime package

---

## 3.2 BackendVersion

创建时：

```text
复制 Backend 当前 config_set_json
再追加/修改该版本参数 schema
保存为 BackendVersion 独立 config_set_json
```

职责：

- 描述软件版本能力。
- 定义参数 schema。
- 定义 backend-level CLI 参数。
- 定义协议、能力、默认 host/port、健康检查默认值。

可以包含：

- `backend.arg.max_model_len`
- `backend.arg.gpu_memory_utilization`
- `backend.arg.dtype`
- `backend.arg.tensor_parallel_size`
- `backend.arg.enable_prefix_caching`
- `backend.common.host`
- `backend.common.port`
- `backend.capabilities`

不应该包含：

- `launcher.image`
- vendor image
- `launcher.docker_options`
- devices
- vendor-specific env
- `/dev/mxcd`
- `/dev/davinci*`
- `CUDA_VISIBLE_DEVICES`
- `ASCEND_VISIBLE_DEVICES`
- `MACA_VISIBLE_DEVICES`

当前代码中的 `upsertBackendVersionFromRequest()` 允许 `image_ref/entrypoint/command/model_mount` 写入 BackendVersion，这需要清理或限制。

---

## 3.3 BackendRuntime / RuntimeTemplate

创建时：

```text
复制 BackendVersion 当前完整 config_set_json
再追加/覆盖厂商、镜像、容器、设备、健康检查、运行参数
保存为 BackendRuntime 独立 config_set_json
```

职责：

- 某个 vendor/backend/version-policy 的最佳实践运行模板。
- 包含镜像、docker options、设备绑定、env、volume、entrypoint/command、health check 覆盖。

例如：

```text
nvidia.vllm.compat
metax.vllm.compat
huawei.vllm.compat
nvidia.sglang.compat
nvidia.llamacpp.compat
cpu.llamacpp.compat
```

---

## 3.4 NodeBackendRuntime

创建时：

```text
复制 BackendRuntime 当前完整 config_set_json
再应用节点本地覆盖：
- image_ref override
- selected accelerator/devices
- node-specific env
- check/probe metadata
保存为 NodeBackendRuntime 独立 config_set_json
```

职责：

- 某个节点上真实启用后的运行配置。
- 是唯一可部署对象。
- 必须经过 enable + check-request。
- `ready` 或 `ready_with_warnings` 才能部署。

当前代码中 NBR schema 已经有 `config_set_json`，但 Claude 需要核对 `upsertNodeBackendRuntime()` 是否完全由服务端深拷贝 BackendRuntime ConfigSet，不能只信前端传入的 `config_set`。

---

## 3.5 Deployment

创建时：

```text
复制 NodeBackendRuntime 当前完整 config_set_json
再应用 Deployment config_overrides
保存为 Deployment 独立 config_set_json
```

职责：

- 某个模型部署的最终快照。
- RunPlan 应只使用 Deployment snapshot 与部署覆盖，不再读取上游 Runtime/Version 当前值。
- 后续 NBR 修改不影响已创建 Deployment。

当前代码中 Deployment 创建仍然使用 `mergeNBRConfigSnapshot(BR, NBR, image)`，NBR 为空时 fallback 到 BackendRuntime。这需要删除。

---

## 4. source_metadata 规范

每一层创建时写清楚来源：

```json
{
  "copy_semantics": "copy_on_create",
  "source_type": "backend_version",
  "source_id": "backend-version.vllm.compat",
  "source_config_hash": "sha256:...",
  "copied_at": "2026-06-26T...",
  "copy_boundary": "detached_after_create"
}
```

示例：

### BackendVersion source_metadata

```json
{
  "copy_semantics": "copy_on_create",
  "source_type": "backend",
  "source_backend_id": "backend.vllm",
  "source_config_hash": "sha256:...",
  "copied_at": "...",
  "copy_boundary": "detached_after_create"
}
```

### BackendRuntime source_metadata

```json
{
  "copy_semantics": "copy_on_create",
  "source_type": "backend_version",
  "source_backend_version_id": "backend-version.vllm.compat",
  "source_config_hash": "sha256:...",
  "copied_at": "...",
  "copy_boundary": "detached_after_create"
}
```

### NodeBackendRuntime source_metadata

```json
{
  "copy_semantics": "copy_on_create",
  "source_type": "backend_runtime",
  "source_backend_runtime_id": "runtime.nvidia.vllm.compat",
  "source_config_hash": "sha256:...",
  "copied_at": "...",
  "copy_boundary": "detached_after_create",
  "check_semantics": "server_verified"
}
```

### Deployment source_metadata

```json
{
  "copy_semantics": "copy_on_create",
  "source_type": "node_backend_runtime",
  "source_node_backend_runtime_id": "...",
  "source_backend_runtime_id": "...",
  "source_config_hash": "sha256:...",
  "copied_at": "...",
  "copy_boundary": "detached_after_create"
}
```

---

## 5. 禁止事项

必须在代码和测试中禁止：

1. BackendRuntime 查询时动态读取 BackendVersion 参数 merge。
2. NodeBackendRuntime 查询时动态读取 BackendRuntime 参数 merge。
3. Deployment 预览或启动时动态读取 NBR / BackendRuntime / BackendVersion 当前参数。
4. RunPlan 运行时 fallback 到 BackendVersion 或 BackendRuntime 取 image/args/env。
5. 上游修改自动影响已创建下游。
6. 下游修改回写上游。
7. 为兼容旧数据保留隐式 fallback。
8. UI 文案使用“继承自”暗示动态继承。

---

## 6. 允许的显式操作

可以提供这些操作，但必须由用户主动触发：

```text
基于当前 Backend 创建新 BackendVersion
基于当前 BackendVersion 创建新 RuntimeTemplate
基于当前 RuntimeTemplate 重新创建 NBR
基于当前 NBR 创建新 Deployment
从上游重新复制为新版本
```

不建议做“同步到已有对象”，除非显式展示 diff 并创建新 snapshot。

---

## 7. 当前代码需要修改的位置

### 7.1 BackendVersion catalog seed

当前 `MaterializeBackendVersion()` 从 registry base 生成，而不是从 Backend materialized config 复制。

建议改为：

```go
backendSet := MaterializeBackend(registry, backend)
items := cloneItems(backendSet.Items)
applyVersionItems(items, version)
```

这样 catalog seed 和 API create 都符合 Backend → BackendVersion copy-on-create。

### 7.2 BackendVersion API

`upsertBackendVersionFromRequest()`：

- 创建时无 config_set：复制 Backend config_set。当前已有。
- patch 时：只修改当前版本 config_set，不影响下游。
- 删除/拒绝 `image_ref` 等 Runtime 字段。
- clone 时：复制 source version config_set，当前已有。

### 7.3 BackendRuntime create

当前 `HandleCreateBackendRuntimeFromTemplate()` 已经从 BackendVersion 复制 config_set。需要加强：

- source_metadata 写 `copy_boundary: detached_after_create`。
- 只允许从 visible/active BackendVersion 创建。
- Runtime 专属字段写到 Runtime config_set。

### 7.4 NodeBackendRuntime enable

需要检查并确保：

- 后端读取 BackendRuntime config_set。
- 服务端 deep copy。
- 应用请求中的 image/env/docker/options 覆盖。
- 保存为 NBR config_set_json。
- 不允许 NBR 后续查询时从 BackendRuntime 动态补字段。

### 7.5 Deployment create

改为：

```go
if nbrConfigSetRaw == "" || nbrConfigSetRaw == "{}" {
    writeError(400, "node backend runtime config snapshot is missing; recreate node backend runtime")
    return
}
deploymentConfigSet := copyConfigSet(nbrConfigSetRaw)
applyConfigOverrides(deploymentConfigSet, configOverrides, "Deployment", id)
```

删除 fallback：

```go
mergeNBRConfigSnapshot(h.buildDeploymentRuntimeSnapshot(...), ...)
```

### 7.6 RunPlan resolver

目标：

- image 从 Deployment/NBR snapshot 中取。
- args 从 Deployment/NBR snapshot 中取。
- env 从 Deployment/NBR snapshot 中取。
- BackendVersion/BackendRuntime 仅作为 metadata，不再参与最终缺省补齐。

需要删除或限制：

```go
resolveImage() fallback BackendRuntime / BackendVersion
required parameter fallback BackendVersion.ParameterDefs
resource_controls fallback BackendVersion.VendorOptionsJSON
```

如果某些兼容逻辑必须保留，应改为错误提示并要求重建 NBR/Deployment。

---

## 8. 必须新增测试

### 8.1 Backend → BackendVersion copy

1. 创建 BackendVersion A，未传 config_set。
2. 确认 A 拥有 Backend 当前参数。
3. 修改 Backend 参数。
4. 确认 A 不变化。

### 8.2 BackendVersion → Runtime copy

1. 从 BackendVersion A 创建 Runtime R。
2. 修改 BackendVersion A 新增参数。
3. 确认 R 不出现新增参数。
4. 从 A 再创建 R2，R2 出现新增参数。

### 8.3 Runtime → NBR copy

1. 从 Runtime R enable NBR N。
2. 修改 R 的参数。
3. 确认 N 不变化。

### 8.4 NBR → Deployment copy

1. 从 NBR N 创建 Deployment D。
2. 修改 N 参数。
3. 确认 D 不变化。

### 8.5 RunPlan snapshot-only

1. D 创建后修改所有上游。
2. Dry-run D。
3. 确认命令仍然来自 D snapshot。
4. 如果 D snapshot 缺 image，dry-run 返回明确错误，不 fallback 上游。
