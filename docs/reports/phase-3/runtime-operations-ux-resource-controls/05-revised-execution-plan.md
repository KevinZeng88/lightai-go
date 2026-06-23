# Runtime Operations UX & Resource Controls — Revised Execution Plan

> Date: 2026-06-23
> Status: Revised plan based on code review. Ready for execution.
> Input: design doc, review report (04), user decisions.
> Constraint: No schema change in this batch unless explicitly noted.

## 1. Phase overview

| Phase | Name | Goal | Schema change | Code | Frontend | Catalog | Docs |
|-------|------|------|---------------|------|----------|---------|------|
| 0 | 现状审计与文档修正 | 确认代码现状，修正设计文档假设 | No | No | No | No | Yes |
| 1 | RunPlan lint | pre-normalization + final lint，嵌入 dry-run | No | Yes | No | No | Yes |
| 2a | resource_controls 建模 | vendor_options_json 存储，参数映射 | No | Yes | No | Yes | Yes |
| 2b | shared GPU admission | DOCUMENTED_BLOCKER | Deferred | No | No | No | Yes |
| 3 | runtime log classifier | 分类已知 warning/error 模式 | No | Yes | No | No | Yes |
| 4 | model instance auto-refresh | 复用 useAutoRefresh，新增状态感知封装 | No | Yes | Yes | No | Yes |
| 5 | JsonViewer + diagnostics UX | 新建 JsonViewer 组件，改造诊断展示 | No | Yes | Yes | No | Yes |
| 6a | HealthCheckEditor | 结构化健康检查编辑 | No | Yes | Yes | No | Yes |
| 6b | ConfigEditorLayout | DEFERRED — 后续单独立项 | — | — | — | — | — |
| 7 | 最终验证与 closeout | 全量回归，closeout 文档 | No | No | No | No | Yes |

## 2. Phase 0 — 现状审计与文档修正

### 目标

确认 review report 中的 evidence 与代码一致，修正设计文档中的错误假设。

### 具体文件

- `docs/reports/phase-3/runtime-operations-ux-resource-controls/04-claude-review-report.md` — 本次 review 输出
- `docs/reports/phase-3/runtime-operations-ux-resource-controls/05-revised-execution-plan.md` — 本文件
- `docs/design/runtime-operations-ux-resource-controls.md` — 如需修正假设，更新设计文档

### 改动类型

- 改代码：否
- 改前端：否
- 改 catalog：否
- 改文档：是（review report + execution plan）

### 测试命令

```bash
# 无代码变更，无需测试
git status --short
```

### 验收标准

- 04 和 05 文件存在且内容完整
- 所有 evidence 引用指向真实代码位置
- git status 无意外变更

## 3. Phase 1 — RunPlan lint（两阶段）

### 目标

检测 command/env 冲突，在 container start 前暴露问题。设计为两阶段：pre-normalization（记录用户输入重复/覆盖）+ final lint（检查最终 command 的 env/CLI conflict、高风险参数）。

### 具体文件

**新建：**
- `internal/server/runplan/lint.go` — lint 逻辑
- `internal/server/runplan/lint_test.go` — 测试
- `internal/server/runplan/testdata/lint/*.json` — 测试 fixture

**修改：**
- `internal/server/api/deployment_lifecycle_handlers.go` — dry-run response 中嵌入 lint 结果

### 改动类型

- 改代码：是（Go）
- 改前端：否
- 改 catalog：否
- 改文档：是（closeout section）

### lint 设计

```go
// Phase 1a: pre-normalization — 在 deduplicateArgs 之前运行
// 记录用户输入中的重复 flag 和平台参数覆盖
type PreLintResult struct {
    UserDuplicates  []LintFinding  // 用户输入中同一 flag 出现多次
    UserOverrides   []LintFinding  // 用户 extra_args 覆盖平台参数
}

// Phase 1b: final lint — 在 deduplicateArgs 之后运行
// 检查最终 command 的 env/CLI conflict、高风险参数
type RunPlanLintResult struct {
    Status   string           // ok, warning, error
    Findings []LintFinding
}

type LintFinding struct {
    ID         string   // e.g. "arg.duplicate", "arg.env_cli_conflict"
    Severity   string   // error, warning, advisory
    Category   string   // arg_conflict, duplicate_arg, high_risk, env_cli_conflict
    Message    string
    Suggestion string
    FieldPath  string
    Sources    []string // platform, template, user_extra_args, user_env, backend_default
}
```

### LLAMA_ARG_HOST 处理策略

```
lint 区分 env 来源：
  - image-provided env（Docker 镜像自带）+ CLI conflict → warning，不 block
  - user-provided env + CLI conflict → 可 error

log classifier 分类该 warning（Phase 3）：
  - llamacpp.env_overwritten.host → severity: warning, category: arg_conflict

实施顺序：
  1. 先验证：grep 确认平台代码不设置 LLAMA_ARG_HOST/PORT（已在 review 中完成）
  2. lint 检测 env/CLI conflict，标记来源（image vs user）
  3. log classifier 分类运行时 warning
  4. 不声称"unset 能消除 warning"——未经运行时验证
```

### 嵌入 dry-run response

```json
{
  "valid": true,
  "errors": [],
  "warnings": [],
  "command_preview": "...",
  "lint": {
    "status": "ok|warning|error",
    "findings": []
  }
}
```

不新增 API route。lint 结果合并到现有 dry-run response。

### 测试

| 测试名 | 验证内容 |
|--------|----------|
| TestPreLintDuplicateFlag | 用户输入中同一 flag 出现两次 → 记录 |
| TestPreLintUserOverridesPlatform | 用户 extra_args 覆盖平台 --host → 记录 |
| TestFinalLintLlamaCppImageEnvConflict | LLAMA_ARG_HOST（image-provided）+ --host → warning |
| TestFinalLintLlamaCppUserEnvConflict | LLAMA_ARG_HOST（user-provided）+ --host → error |
| TestFinalLintDuplicateCtxSize | dedup 后仍有重复 → error |
| TestFinalLintDuplicateVLLMGpuMemoryUtilization | 重复 --gpu-memory-utilization → error |
| TestFinalLintDuplicateSGLangMemFractionStatic | 重复 --mem-fraction-static → error |
| TestFinalLintHighRiskContainerFlags | --privileged → warning |
| TestFinalLintCleanVLLM | 正常 vLLM RunPlan → ok |
| TestFinalLintCleanSGLang | 正常 SGLang RunPlan → ok |
| TestFinalLintCleanLlamaCpp | 正常 llama.cpp RunPlan → ok |

### 测试命令

```bash
go test ./internal/server/runplan/ -v -run TestPreLint
go test ./internal/server/runplan/ -v -run TestFinalLint
go test ./internal/server/api/ -v -run TestDryRun
go build ./...
gofmt -l internal/server/runplan/lint.go
```

### 验收标准

- 所有 lint 测试通过
- dry-run response 包含 `lint` 字段
- pre-normalization lint 记录 user override evidence
- final lint 在 dedup 之后运行
- LLAMA_ARG_HOST image-provided 冲突标记为 warning（不 block）
- 不声称 unset 一定能消除 warning

## 4. Phase 2a — resource_controls 建模

### 目标

在 vendor_options_json 中存储 backend-specific resource_controls，为 vLLM/SGLang/llama.cpp 提供参数映射。

### 具体文件

**新建：**
- `internal/server/runplan/resource_controls.go` — resource_controls 解析和参数映射
- `internal/server/runplan/resource_controls_test.go` — 测试

**修改：**
- `internal/server/db/db.go` — seedTargetBackendCatalog 中为 vLLM/SGLang/llama.cpp 的 vendor_options_json 预置 resource_controls
- `configs/backend-catalog/runtimes/llamacpp/nvidia-cuda13.yaml` — 如需同步 YAML

### 改动类型

- 改代码：是（Go）
- 改前端：否
- 改 catalog：是（seed + YAML）
- 改文档：是（说明 vendor_options_json 方案）

### resource_controls 设计

暂放 `vendor_options_json`，当前无 schema 方案。

vLLM 示例：
```json
{
  "resource_controls": {
    "gpu_memory_fraction": {
      "supported": true,
      "arg": "--gpu-memory-utilization",
      "min": 0.1, "max": 0.95, "default": 0.9,
      "semantics": "per_instance_backend_allocation_budget"
    },
    "max_model_len": {"arg": "--max-model-len", "type": "int"},
    "max_num_seqs": {"arg": "--max-num-seqs", "type": "int"}
  }
}
```

SGLang 示例：
```json
{
  "resource_controls": {
    "gpu_memory_fraction": {
      "supported": true,
      "arg": "--mem-fraction-static",
      "min": 0.1, "max": 0.95, "default": 0.9,
      "semantics": "static_weights_and_kv_pool"
    },
    "max_running_requests": {"arg": "--max-running-requests", "type": "int"},
    "attention_backend": {"arg": "--attention-backend", "type": "enum", "values": ["auto", "flashinfer", "triton", "fa3"]}
  }
}
```

llama.cpp 示例：
```json
{
  "resource_controls": {
    "gpu_memory_fraction": {"supported": false, "reason": "llama.cpp does not expose a vLLM-style GPU memory fraction."},
    "gpu_layers": {"arg": "--n-gpu-layers", "type": "string_or_int"},
    "ctx_size": {"arg": "--ctx-size", "type": "int"},
    "batch_size": {"arg": "--batch-size", "type": "int"},
    "ubatch_size": {"arg": "--ubatch-size", "type": "int"},
    "cache_type_k": {"arg": "--cache-type-k", "type": "enum"},
    "split_mode": {"arg": "--split-mode", "type": "enum"},
    "tensor_split": {"arg": "--tensor-split", "type": "string"}
  }
}
```

### seed 覆盖风险

**关键**：`seedTargetBackendCatalog` 的 UPDATE 语句会强制重置 `vendor_options_json='{}'`（`db.go:1395`）。实施时需要：

1. 在 seed 的 vendor_options_json 中预置 resource_controls JSON；
2. 或修改 seed UPDATE 逻辑，不覆盖 vendor_options_json 中的 resource_controls 字段；
3. 测试验证：用户修改 vendor_options_json 后，重启不会丢失 resource_controls。

### 测试

| 测试名 | 验证内容 |
|--------|----------|
| TestParseResourceControlsVLLM | 解析 vLLM resource_controls JSON |
| TestParseResourceControlsSGLang | 解析 SGLang resource_controls JSON |
| TestParseResourceControlsLlamaCpp | 解析 llama.cpp resource_controls JSON，gpu_memory_fraction.supported=false |
| TestResourceControlsToArgs | resource_controls 转换为 CLI args |
| TestSeedPreservesUserVendorOptions | seed 不覆盖用户 resource_controls |

### 测试命令

```bash
go test ./internal/server/runplan/ -v -run TestResourceControls
go test ./internal/server/api/ -v -run TestBackendVersion
go build ./...
```

### 验收标准

- vendor_options_json 包含 resource_controls
- vLLM/SGLang 正确映射 memory fraction 到 backend arg
- llama.cpp 不暴露 fake memory fraction
- seed 不覆盖用户配置
- YAML 与 DB seed 一致

## 5. Phase 2b — Shared GPU Admission（DOCUMENTED_BLOCKER）

### 目标

记录为 DOCUMENTED_BLOCKER，不实施。

### Blocker 描述

- **问题**：当前 gpu_leases 唯一索引 (`idx_gpu_leases_reserved_active`) 强制同一 GPU 只能有一个 active/reserved lease，无法支持 shared GPU。
- **Evidence**：`internal/server/db/db.go:715-716`
- **需要的变更**：删除唯一索引 + 实现 budget admission 逻辑 + 可能新增 resource_budget_json 列
- **影响**：无法在本 batch 实现 shared GPU deployment
- **风险**：如果用户需要 shared GPU，必须先做 schema change
- **最小修复位置**：`internal/server/db/db.go` migrateV8 + 新增 admission 逻辑
- **建议验证命令**：`go test ./internal/server/runplan/ -v -run TestGPUAdmission`

### 文档化

写入 closeout 文档，状态为 DOCUMENTED_BLOCKER。

## 6. Phase 3 — Runtime log classifier

### 目标

分类已知运行时日志模式，用户无需手动阅读 Docker logs。

### 具体文件

**新建：**
- `internal/server/runplan/log_classifier.go` — 分类逻辑和规则
- `internal/server/runplan/log_classifier_test.go` — 测试
- `internal/server/runplan/testdata/runtime-logs/sglang-torchao-syntax-warning.log`
- `internal/server/runplan/testdata/runtime-logs/sglang-attention-backend-default.log`
- `internal/server/runplan/testdata/runtime-logs/llamacpp-env-host-overwritten.log`
- `internal/server/runplan/testdata/runtime-logs/cuda-oom.log`

**修改（如有）：**
- API response 中添加 `classified_log_events`（model test result 或 instance diagnostics）

### 改动类型

- 改代码：是（Go）
- 改前端：否
- 改 catalog：否
- 改文档：是

### 规则设计

```go
type RuntimeLogRule struct {
    ID         string
    Backend    string // vllm, sglang, llamacpp, *
    Pattern    string // regex
    Severity   string // fatal, error, warning, advisory, noise
    Category   string // dependency_warning, default_selection, arg_conflict, oom, startup
    Message    string
    Suggestion string
}
```

初始规则：

| Rule ID | Backend | Severity | Category |
|---------|---------|----------|----------|
| sglang.torchao.syntax_warning | sglang | noise | dependency_warning |
| sglang.attention_backend.default | sglang | advisory | default_selection |
| llamacpp.env_overwritten.host | llamacpp | warning | arg_conflict |
| llamacpp.env_overwritten.port | llamacpp | warning | arg_conflict |
| cuda.oom | * | error | oom |
| container.startup.failed | * | error/fatal | startup |

### 关键约束

- noise 和 advisory 不改变实例状态
- error/fatal 可以标注失败诊断，但状态转换仍遵循现有 health/lifecycle 模型
- 规则在 Go 中硬编码，不存 DB

### 测试命令

```bash
go test ./internal/server/runplan/ -v -run TestLogClassifier
go build ./...
```

### 验收标准

- 所有 fixture 测试通过
- noise/advisory 不标记实例为 failed
- warning/error 在诊断中可见
- 分类输出包含 rule ID、severity、message、suggestion、raw line、occurrences

## 7. Phase 4 — Model instance auto-refresh

### 目标

模型实例页面状态自动更新，无需手动刷新。

### 具体文件

**修改：**
- `web/src/pages/ModelInstancesPage.vue` — 引入 useAutoRefresh

**新建（如需）：**
- `web/src/composables/useInstanceStatusPolling.ts` — 状态感知封装，不同状态不同轮询间隔

### 改动类型

- 改代码：否
- 改前端：是（Vue）
- 改 catalog：否
- 改文档：是

### 设计

复用现有 `useAutoRefresh`（`web/src/composables/useAutoRefresh.ts`），它已提供：
- visibility API 暂停
- route leave 停止
- inflight guard
- focus 刷新
- interval 轮询

新增 `useInstanceStatusPolling` 封装：
- transitional states (pending/starting/stopping): 3s 间隔
- stable states (running/failed/stopped): 15s 间隔
- document hidden: useAutoRefresh 已处理
- request failure: useAutoRefresh 已处理（refreshError）

### 测试

使用 `node web/tests/*.test.mjs` 模式或手动验证。

### 测试命令

```bash
cd web && npm run build
# 手动验证：打开 ModelInstancesPage，观察自动刷新
```

### 验收标准

- 无需手动刷新即可看到状态变化
- 页面显示 last refreshed time
- document hidden 时暂停轮询
- 无 console error
- 无 i18n key leak

## 8. Phase 5 — JsonViewer + diagnostics UX

### 目标

诊断 JSON 可在页面中阅读、复制、展开。

### 具体文件

**新建：**
- `web/src/components/common/JsonViewer.vue`

**修改（逐步）：**
- 需要诊断 JSON 展示的页面（model test result、instance detail、dry-run result 等）

### 改动类型

- 改代码：否
- 改前端：是（Vue）
- 改 catalog：否
- 改文档：是

### 组件设计

```vue
<!-- Props -->
{
  value: unknown | string     // JSON 对象或字符串
  title?: string              // 标题
  defaultExpanded?: boolean   // 默认展开
  maxHeight?: string          // 最大高度，默认 400px
  readonly?: boolean          // 只读模式
  allowDownload?: boolean     // 允许下载
  allowCopy?: boolean         // 允许复制
  allowFullscreen?: boolean   // 允许全屏
}
```

功能：
- 固定最大高度 + 滚动
- 全屏展开
- 复制完整内容
- 下载
- 搜索
- 行 wrap 切换
- malformed JSON fallback 到 raw text
- 长字符串不破坏布局

### 测试

使用 `node web/tests/*.test.mjs` 模式。

### 测试命令

```bash
cd web && npm run build
cd web && npm test
```

### 验收标准

- 长 JSON 渲染在约束高度内
- 复制返回完整内容
- 全屏打开并显示内容
- malformed JSON 显示 raw text
- 同一组件在多个诊断位置复用

## 9. Phase 6a — HealthCheckEditor + RunnerConfigsPage

### 目标

健康检查配置结构化，区分用户配置与系统生成的诊断 JSON。

### 具体文件

**新建：**
- `web/src/components/config/HealthCheckEditor.vue`

**修改：**
- `web/src/pages/RunnerConfigsPage.vue` — 将 health check textarea 替换为结构化编辑器
- `web/src/pages/BackendsPage.vue` — 如适用

### 改动类型

- 改代码：否
- 改前端：是（Vue）
- 改 catalog：否
- 改文档：是

### 用户可配置字段

```
path: string
method: string (GET/POST)
timeout_seconds: number
interval_seconds: number
expected_status: number
expected_body_contains: string
readiness_grace_seconds: number
```

### 只读生成字段

- health result JSON
- advanced diagnostic JSON
- RunPlan JSON
- Docker inspect JSON
- preflight evidence JSON

### 测试命令

```bash
cd web && npm run build
cd web && npm test
```

### 验收标准

- health check 配置是结构化表单
- 生成的诊断 JSON 是只读的
- 用户配置与诊断边界清晰
- textarea 仅用于 expert mode（如有）

## 10. Phase 6b — ConfigEditorLayout（DEFERRED）

### 目标

完整配置编辑器布局，统一 runtime config 和 deployment 页面。

### 状态

**DEFERRED。** 后续单独立项。

### 原因

- 范围过大，涉及 5 个新组件 + 2 个页面改造
- Phase 5 + 6a 已覆盖核心需求（JsonViewer + HealthCheckEditor）
- 完整 ConfigEditorLayout 需要独立设计和实施周期

## 11. Phase 7 — 最终验证与 closeout

### 目标

全量回归测试，写 closeout 文档。

### 具体文件

**新建：**
- `docs/reports/phase-3/runtime-operations-ux-resource-controls/closeout.md`

### 改动类型

- 改代码：否
- 改前端：否
- 改 catalog：否
- 改文档：是（closeout）

### 测试命令

```bash
go test ./...
go build ./...
gofmt -l internal/
cd web && npm run build
cd web && npm test
git diff --check
git status --short
```

### 验收标准

- 所有 Go 测试通过
- 所有前端测试通过
- gofmt 无格式问题
- git diff --check 无空白问题
- git status --short 无意外变更

### closeout 内容

- issue-by-issue status（每个已知问题的状态）
- files changed
- tests run + results
- evidence paths
- known blockers/deferred items
- commit id
- push result
- final git status --short

## 12. Deferred / Blocker 清单

| 内容 | 状态 | 说明 | 触发条件 |
|------|------|------|----------|
| Shared-GPU admission | DOCUMENTED_BLOCKER | gpu_leases 唯一索引强制独占 | 用户明确要求共享 GPU |
| ConfigEditorLayout (Phase 6b) | deferred | 5 组件 + 2 页面改造 | 独立项 |
| Status-summary API | conditional | 先轮询现有 list endpoint | 轮询性能不足时 |
| vitest 引入 | deferred | 使用 node tests/*.test.mjs | 必须引入时单独计划 |
| llama.cpp VRAM estimator | deferred | 间接控制，无直接 fraction | 未来需求 |
| WebSocket 实例更新 | deferred | 轮询足够 | 轮询性能不足时 |
| DB-based log rule editor | deferred | Go 内置规则足够 | 规则数量大幅增长时 |
| MIG/vGPU/HAMi | deferred | 独立能力 | 用户明确要求 |

## 13. 最终执行顺序建议

```
Phase 0 → Phase 1 → Phase 2a → Phase 3 → Phase 4 → Phase 5 → Phase 6a → Phase 7
```

- Phase 0 先行：review report + execution plan 已就绪
- Phase 1 在 Phase 2a 之前：lint 基础设施先建好
- Phase 3 独立于 Phase 1/2a：log classifier 可并行但建议在 lint 之后
- Phase 4 独立：前端改动，不影响后端
- Phase 5 在 Phase 4 之后：JsonViewer 可能在 Phase 4 中使用
- Phase 6a 在 Phase 5 之后：HealthCheckEditor 可能复用 JsonViewer
- Phase 7 最后：全量回归

每个 phase 完成后运行 phase-specific 测试，Phase 7 运行全量回归。

## 14. Closeout 要求

- 每个 phase 完成后记录到 closeout 文档
- 最终 closeout 包含所有 phase 的状态
- 所有已知问题必须是 FIXED / DOCUMENTED_BLOCKER / INVALID 之一
- 不允许 TODO / LATER / PARTIAL / KNOWN / LOW / DEFERRED 状态
- commit 只在 closeout 完成后创建
- push 只在 commit 后执行
