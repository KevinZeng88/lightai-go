# 06 - Claude Development Prompt

请在 `/home/kzeng/projects/ai-platform-study/lightai-go` 中实现 Runtime Template / BackendVersion / Schema-driven Parameters / Copy-on-Create 重构，并顺带修复本审查文档列出的 P1/P2 问题。

先阅读以下文档：

```text
docs/reports/phase-3/runtime-template-redesign/00-index.md
docs/reports/phase-3/runtime-template-redesign/01-current-code-map-and-review.md
docs/reports/phase-3/runtime-template-redesign/02-schema-driven-parameter-ui-design.md
docs/reports/phase-3/runtime-template-redesign/03-copy-on-create-data-model-and-api-design.md
docs/reports/phase-3/runtime-template-redesign/04-runtime-template-catalog-cleanup.md
docs/reports/phase-3/runtime-template-redesign/05-implementation-plan-and-acceptance.md
```

## 目标

实现以下最终状态：

1. Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / Deployment 边界清晰。
2. BackendVersion 有前端展示、复制、新增、编辑、删除入口。
3. BackendVersion 新增参数后，界面自动多出输入框，不需要改前端代码。
4. Runtime 参数 UI 完全由 `config_set.items` 驱动。
5. 逐层 copy-on-create：
   - Backend → BackendVersion
   - BackendVersion → BackendRuntime
   - BackendRuntime → NodeBackendRuntime
   - NodeBackendRuntime → Deployment
6. 复制完成后，下游与上游脱钩。
7. 禁止 runtime dynamic inheritance。
8. RunPlan 只能使用 Deployment/NBR snapshot，不得 fallback 到 BackendRuntime/BackendVersion 当前值。
9. 普通 Runtime selector 不显示 hidden/reference/disabled/重复/占位模板。
10. 清理 `runtime.xxx`、`template-only`、`<from Metax release package>`、本地 image id 等脏内容。
11. 华为、沐曦和其他国内 GPU 模板按 visible/hidden/experimental/reference 策略处理，不污染普通 UI。

## 必须修复

### 1. BackendVersion UI

当前后端 API 已经有 BackendVersion CRUD/clone，但前端 `/backends` 只显示 Backend 和 JSON。请增加版本管理 UI。

要求：

- 展示某个 Backend 的版本列表。
- 支持 clone 系统版本。
- 支持新增用户版本。
- 支持编辑用户版本 ConfigSet。
- 支持新增参数。
- 支持删除未被 Runtime 使用的用户版本。
- 系统版本只读。
- 新增参数后参数编辑器自动显示。

### 2. Schema-driven 参数 UI

当前 `HumanRuntimeParameterForm` / `runtimeParameterViewModel.ts` 是硬编码字段。请替换为 schema-driven form，或增强 `RuntimeParameterEditor` 并统一所有入口使用它。

必须支持：

- string
- integer
- number
- boolean
- select
- multi_select
- array
- object
- lines
- path
- file

必须支持字段：

- code
- label
- help
- category
- group
- kind
- type
- required
- enabled
- value
- default_value
- order
- support_level
- visibility
- readonly
- advanced
- constraints
- render.flag
- render.env_name
- render.style
- render.target
- render.options

保留 `{ enabled, value }` 结构，禁用参数仍显示输入框但不进入最终 RunPlan。

### 3. ConfigItem schema 对齐

修改 `internal/server/catalog/types.go` 和 materializer：

- 增加 `Required` 等必要字段。
- `addArgConfigItems()` 生成的 label/group/order/constraints 必须被前端正确显示。
- 前端读取 `render.label || extensions.label || code`。
- 前端读取 `constraints || render.constraints`。
- 前端按 `order` 排序。

### 4. BackendVersion 边界

BackendVersion 只描述软件版本能力和参数 schema。

禁止 BackendVersion 接收或保存：

- image_ref
- vendor image
- docker options
- devices
- vendor env
- device binding

如 API 收到这些字段，返回 400，并提示放到 BackendRuntime。

### 5. Runtime catalog 清理

实现 visible/hidden/status/support_level 策略。

普通 selector 只展示：

```text
visibility = visible
status in active / experimental
```

初始 visible 模板建议：

```text
nvidia.vllm.compat
nvidia.sglang.compat
nvidia.llamacpp.compat
cpu.llamacpp.compat
metax.vllm.compat
huawei.vllm.compat
```

其他国产 GPU 模板可作为 hidden reference/experimental，不能进入普通选择器。

### 6. 严格 copy-on-create

必须实现并测试：

- BackendVersion 创建复制 Backend 当前 ConfigSet。
- BackendRuntime 创建复制 BackendVersion 当前 ConfigSet。
- NodeBackendRuntime enable 复制 BackendRuntime 当前 ConfigSet。
- Deployment create 复制 NodeBackendRuntime 当前 ConfigSet。
- 上游修改不影响下游。
- 下游修改不回写上游。

删除或修正：

- Deployment create fallback 到 BackendRuntime snapshot。
- RunPlan image/args/env fallback 到 BackendRuntime/BackendVersion 当前值。
- 查询时跨层动态 merge。

### 7. 代码 review 问题

顺带处理：

- `configSetParameterValues()` env kind 不可达。
- reset admin password interactive 明文输入。
- RuntimeTemplate API 返回 parsed object。
- Runtime selector 逻辑重复模板过滤。
- Deployment detail 结构化参数显示。
- router 格式整理。
- README/docs 更新。

## 验收

必须执行：

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...

cd web
npm run build
npm test
```

必须新增 API-first E2E 或 Go tests 覆盖：

1. BackendVersion 新增 fake 参数。
2. UI/API 返回中该参数可见。
3. Runtime 从 BackendVersion copy 后拥有该参数。
4. 修改 BackendVersion 后既有 Runtime 不变。
5. NBR 从 Runtime copy 后拥有该参数。
6. 修改 Runtime 后既有 NBR 不变。
7. Deployment 从 NBR copy 后拥有该参数。
8. 修改 NBR 后既有 Deployment 不变。
9. RunPlan 只包含 enabled=true 参数。
10. disabled 参数不进入命令。
11. hidden/reference templates 不在普通 selector 出现。
12. 重复 seed 不产生重复模板。

## 输出

完成后输出：

- 修改文件清单
- 关键设计实现说明
- 最终 visible runtime templates
- hidden/reference templates
- 删除/隐藏的脏模板
- 新增参数自动渲染证据
- copy-on-create API-first 证据
- RunPlan snapshot-only 证据
- 测试结果
- closeout 文档路径
- commit id
- push 结果
- `git status --short`
