# 01 - 当前代码结构与问题审查

## 1. 当前已有能力

### 1.1 后端 catalog 与 ConfigSet 基础已经存在

当前数据库 schema 中，以下核心表都已经有 `config_set_json` 和 `source_metadata_json`：

- `inference_backends`
- `backend_versions`
- `backend_runtimes`
- `node_backend_runtimes`
- `model_deployments`

这说明系统已经具备“逐层配置快照”的基础，不需要从零重建。

当前 `Migrate()` 采用 fresh ConfigSet schema，明确不兼容旧 DB；这与当前项目原则一致：表结构变化允许重建 DB，避免脏兼容逻辑。

### 1.2 BackendVersion API 已经存在

`internal/server/api/router.go` 已经注册：

- `GET /api/v1/backends`
- `GET /api/v1/backends/{id}`
- `GET /api/v1/backends/{id}/versions`
- `POST /api/v1/backends/{id}/versions`
- `GET /api/v1/backend-versions`
- `PATCH /api/v1/backend-versions/{version_id}`
- `POST /api/v1/backend-versions/{version_id}/clone`
- `DELETE /api/v1/backend-versions/{version_id}`
- `POST /api/v1/backend-catalog/reload`

也就是说，后端能力已经存在，主要缺口在前端管理入口、schema 语义完整性和边界治理。

### 1.3 RuntimeParameterEditor 已经具备 schema-driven 基础

`RuntimeParameterEditor.vue` 不是硬编码 vLLM/SGLang/llama.cpp 参数，而是读取：

```ts
sourceConfigSet.value?.items
```

然后按 `category` 分组、按 `type` 渲染输入控件，并输出：

```ts
config_set
config_overrides.parameter_values
```

这意味着“BackendVersion 新增参数后，界面自动多出输入框”是可行的，不需要重写整套前端，只需要把所有人类友好参数编辑入口统一切到 `config_set.items` 驱动。

### 1.4 RunPlan 参数解析已经接近 snapshot-only

`runplan.Resolve()` 中 `buildArgs()` 明确要求：

```go
if in.NBRConfigSnapshot == nil {
    errors = append(errors, fmt.Errorf("node backend runtime parameter snapshot is missing"))
    return args, errors
}
```

并按以下层次构造命令：

1. NBR args snapshot
2. NBR structured parameter values
3. Deployment parameter overrides
4. disabled tombstones
5. service args

这基本符合“NodeBackendRuntime 是真实运行配置来源”的方向。

---

## 2. 当前主要问题

## P0 / P1 级问题

### P1-001：BackendVersion 有 API，但没有真正的 UI 管理入口

当前 `/backends` 页面只展示 Backend 列表，详情 drawer 只显示 JSON ConfigSet 和 Source Metadata。没有版本列表，没有新增/复制/编辑版本，没有参数 schema 编辑器。

影响：

- BackendVersion 变成内部表。
- 业务无法新增版本参数。
- “某个版本新增参数，界面自动多输入框”没有入口实现。
- 用户只能修改 Runtime 或 NBR，无法在版本层沉淀参数 schema。

修复：

- 在 `BackendsPage.vue` 中增加 BackendVersion tab，或新增页面 `/backends/:backendId/versions`。
- 支持 list/create/clone/patch/delete。
- 系统版本只读，用户 clone 后可编辑。
- 版本详情中使用 schema-driven 参数编辑器编辑 `config_set.items`。

---

### P1-002：HumanRuntimeParameterForm 仍然硬编码参数字段

`HumanRuntimeParameterForm.vue` 调用：

```ts
getHumanFieldsForBackend(props.backendName)
```

而 `runtimeParameterViewModel.ts` 中 `HUMAN_FIELDS` 写死了 vLLM、SGLang、llama.cpp 的一批字段。

影响：

- 后端版本新增参数后，`RuntimeParameterEditor` 可以显示，但 `HumanRuntimeParameterForm` 不会自动显示。
- “普通用户友好表单”和“高级 ConfigSet 编辑器”能力不一致。
- 后续每次新增参数还要改前端代码，违背 schema-driven 目标。

修复：

- 用 `SchemaDrivenParameterForm` 替代 `HumanRuntimeParameterForm`。
- 直接从 `config_set.items` 渲染输入项。
- 根据 `kind/category/render/extensions/order/constraints` 决定展示、分组、类型、校验。
- 保留隐藏内部字段策略，但隐藏规则也应来自 schema 或统一 helper。

---

### P1-003：ConfigItem schema 与前端渲染字段不一致

当前 catalog 中 `ConfigItem` 包含：

```go
Code
Category
Kind
Type
Value
DefaultValue
Enabled
Render
Order
Constraints
SupportLevel
Source
LastModified
Extensions
```

但 `RuntimeParameterEditor.vue` 主要读取：

```ts
item.render?.label
item.render?.flag
item.render?.env_name
item.render?.constraints
item.required
```

问题：

1. `ConfigItem` 没有 `Required` 字段，但前端读取 `item.required`。
2. `addArgConfigItems()` 把 label/group 放进 `Extensions`，前端不读取 `extensions.label`。
3. `RuntimeParameterEditor` 没有按 `order` 排序，只按 category/code。
4. 校验读取的是 `render.constraints`，而 catalog 类型中约束是 top-level `Constraints`。
5. `ConfigItem` 不支持 select/options 的标准字段。

影响：

- YAML 中写了 label/order/constraints/required，也可能不生效。
- 自动新增参数可以出现输入框，但体验不好：显示 code、排序混乱、required 不生效、约束不生效。

修复：

- 给 `catalog.ConfigItem` 增加 `Required bool`、`Visibility`、`Readonly`、`Advanced` 等必要字段。
- 前端 label 解析顺序改为：

```ts
item.render?.label || item.extensions?.label || item.label || item.code
```

- group 解析顺序改为：

```ts
item.render?.group || item.extensions?.group || item.category
```

- constraints 解析顺序改为：

```ts
item.constraints || item.render?.constraints
```

- 排序改为：

```ts
categoryOrder, item.order, item.code
```

---

### P1-004：BackendVersion API 允许写入 Runtime 层字段，边界不干净

`upsertBackendVersionFromRequest()` 支持以下字段：

```go
image_ref
command
entrypoint
model_mount
health_check
```

并写入：

```go
launcher.image
launcher.command
launcher.entrypoint
runtime.model_mount
runtime.health
```

问题：

- `image_ref` 明确属于 BackendRuntime / RuntimeTemplate。
- `entrypoint/command/model_mount` 多数情况下也属于 RuntimeTemplate。
- BackendVersion 应该表达后端软件版本能力与参数 schema，不应该携带厂商镜像和容器启动细节。

建议：

- BackendVersion 允许：
  - version
  - protocol
  - capabilities
  - default args schema
  - parameter config items
  - backend.common.host/port
  - health_check 是否保留需讨论；建议作为 Backend 默认可有，Runtime 可覆盖。
- BackendVersion 不允许：
  - image_ref
  - vendor image
  - docker_options
  - devices
  - env/device binding
  - vendor-specific runtime
  - launcher.image
- 若现有 API 需要保留字段，应改为拒绝或忽略，并写明错误：

```text
image_ref belongs to BackendRuntime, not BackendVersion
```

---

### P1-005：Deployment 创建仍然 fallback 到 BackendRuntime snapshot

`HandleCreateDeployment()` 中使用：

```go
configSetRaw := mergeNBRConfigSnapshot(h.buildDeploymentRuntimeSnapshot(backendRuntimeID), nbrConfigSetRaw, nbrImageRef)
```

`mergeNBRConfigSnapshot()` 如果 NBR snapshot 为空，会 fallback 到 BackendRuntime snapshot。

这不符合严格规则：

```text
上游只作为创建来源，不作为运行依赖。
```

Deployment 必须只复制 NodeBackendRuntime 当前 `config_set_json`。如果 NBR snapshot 为空，说明 NBR 不合法，应返回错误，要求重建 NBR。

修复：

- 删除 `mergeNBRConfigSnapshot()` 对 BackendRuntime 的 fallback。
- Deployment create 逻辑：

```go
if emptyConfigSet(nbrConfigSetRaw) {
    writeError(400, "node backend runtime config snapshot is missing; recreate node backend runtime")
    return
}
deploymentConfigSet := copyConfigSet(nbrConfigSetRaw)
applyConfigOverrides(...)
```

---

### P1-006：RunPlan image 解析仍然 fallback 到 BackendRuntime / BackendVersion

`resolveImage()` 当前顺序：

1. NodeRuntimeOverride image
2. BackendRuntime image
3. BackendVersion defaultImages[vendor]
4. error

在严格 copy-on-create 模型下，最终运行不应该依赖 BackendRuntime 或 BackendVersion 当前值。镜像应该已经在 NBR / Deployment ConfigSet snapshot 中。

修复方向：

- RunPlan 输入应从 Deployment snapshot 或 NBR snapshot 中读取 `launcher.image`。
- 如果 Deployment/NBR snapshot 没有 image，则报错。
- BackendRuntime / BackendVersion 只能用于创建 NBR 或创建 Deployment 时复制，不参与启动时解析。

---

### P1-007：`configSetParameterValues()` 对 env kind 的逻辑不可达

当前代码：

```go
if kind != "cli_arg" && kind != "cli_args" {
    continue
}
...
if kind == "env" && pv.Target == "" {
    pv.Target = "env"
}
```

因为前面已经跳过了 `env`，后面的 env 分支不可达。

影响：

- schema-driven env 参数无法通过 `ParameterValue` 进入 RunPlan。
- 如果后续希望通过 BackendVersion 增加 env 参数，可能不会生效。

修复：

- 改为允许：

```go
kind in ("cli_arg", "cli_args", "env", "env_lines")
```

- 或者明确 env 只通过 `runtime.env` object 处理，不走 `ParameterValue`。二选一，不能保留不可达逻辑。

---

### P1-008：Catalog seed 只按 ID upsert，不能防止逻辑重复模板

`SeedCatalog()` 对 backend_runtimes 使用：

```sql
ON CONFLICT(id) DO UPDATE
```

但如果 YAML 里有两个不同 ID、同样 vendor/backend/version 的 visible runtime，就会同时进入系统。

影响：

- UI 中会继续看到多个 `nvidia.vllm`、`metax.sglang`。
- 不利于“少量最佳实践模板”。

修复：

- Catalog Validate 增加逻辑唯一检查。
- 对 visible runtime，建议唯一：

```text
vendor + backend_id + backend_version_id + runtime_distribution + visibility
```

或者直接限制：

```text
每个 vendor + backend + compatible 只允许一个 visible default template
```

- hidden reference 模板可以多个，但不能进入普通 selector。

---

## P2 级问题

### P2-001：RuntimeTemplate API 只返回 YAML 字符串

`HandleListRuntimeTemplates()` 只是读取文件并返回：

```json
{
  "name": "...",
  "source": "...",
  "content": "raw yaml"
}
```

这对 UI 不是很友好，也不利于校验。

建议：

- 返回 parsed object。
- 增加 `visibility/status/vendor/backend/image/source_note` 等字段。
- raw content 可以保留为高级诊断。

---

### P2-002：BackendRuntime / Deployment 详情页仍偏 JSON 展示

`BackendRuntimesPage` 已有较多结构化内容，但 `ModelDeploymentsPage` 详情只显示 JSON ConfigSet、Source Metadata、Dry Run。用户很难判断最终参数。

建议：

- Deployment detail 增加：
  - 参数表
  - enabled 参数
  - disabled 参数
  - 最终 command preview
  - source snapshot metadata
  - 与 NBR 的差异说明

---

### P2-003：安全细节：reset admin password interactive 明文输入

`runResetAdminPassword()` interactive 模式使用：

```go
fmt.Scanln(&input)
```

终端会回显密码。

建议：

- 使用 `golang.org/x/term.ReadPassword(int(os.Stdin.Fd()))`。
- 写入凭据文件仍为 0600，可以保留。

---

### P2-004：聚合 NBR 接口存在 N+1 查询

`HandleListAllNodeBackendRuntimes()` 先查所有 nodes，再对每个 node 查询 NBR。小规模可以接受，但后续节点多时会变慢。

建议：

- 一条 JOIN 查询解决。
- 只在 tenant filter 时按 node tenant 控制。

---

### P2-005：router 部分代码格式不利于维护

`router.go` 中系统路由有一行压了多个 route。建议拆分为多行，方便 diff 和 review。

---

## 3. 当前代码中值得保留的设计

以下部分是正确方向，开发时不要推倒重来：

1. `ConfigSet` 作为统一配置表达。
2. BackendVersion / BackendRuntime / NBR / Deployment 都保存 `config_set_json`。
3. BackendVersion API 的 clone/read-only 机制。
4. BackendRuntime 创建从 BackendVersion 复制配置。
5. NBR check-request 服务端代理 agent 验证镜像，不信任客户端 evidence。
6. Deployment 只接受 `node_backend_runtime_id`。
7. RunPlan 参数层已经基本从 NBR snapshot 和 Deployment override 构造。
8. `RuntimeParameterEditor` 的 schema-driven 基础。
9. `ready_with_warnings` 可部署的前端过滤。
