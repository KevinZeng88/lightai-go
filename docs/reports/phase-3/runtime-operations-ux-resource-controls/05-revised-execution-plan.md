# Runtime Operations UX & Resource Controls — Revised Execution Plan

> Date: 2026-06-23
> Status: Revised plan based on code-verified review report (04-claude-review-report.md).
> Supersedes: 01-implementation-plan.md (original plan, kept for reference).

## 1. 修正后的阶段列表

| Phase | 名称 | 范围 | 是否改代码 | 是否改前端 | 是否改 catalog | 是否改文档 |
|-------|------|------|-----------|-----------|---------------|-----------|
| 0 | 现状审计与文档修正 | review report + 文档更新 | 否 | 否 | 否 | 是 |
| 1 | RunPlan lint | pre-normalization + final lint，嵌入 dry-run | 是 | 否 | 否 | 是 |
| 2a | resource_controls 建模 | vendor_options_json + YAML + 参数映射 | 是 | 否 | 是 | 是 |
| 2b | Shared GPU admission | DOCUMENTED_BLOCKER | 否 | 否 | 否 | 是 |
| 3 | Runtime log classifier | Go 内置规则 + fixture 测试 | 是 | 否 | 否 | 是 |
| 4 | 模型实例自动刷新 | 复用 useAutoRefresh + useInstanceStatusPolling | 否 | 是 | 否 | 是 |
| 5 | JsonViewer + diagnostics UX | 新建 JsonViewer + 替换 `<pre>` | 否 | 是 | 否 | 是 |
| 6a | HealthCheckEditor + RunnerConfigsPage | 结构化 health check 编辑 | 否 | 是 | 否 | 是 |
| 6b | 完整 ConfigEditorLayout | DEFERRED | - | - | - | - |
| 7 | 最终验证与 closeout | 全量回归 + closeout 文档 | 否 | 否 | 否 | 是 |

## 2. 每阶段详细计划

### Phase 0 — 现状审计与文档修正

**目标**：确认当前代码状态，输出 review report（已完成），更新设计文档。

**具体文件**：
- `docs/reports/phase-3/runtime-operations-ux-resource-controls/04-claude-review-report.md`（已完成）
- `docs/reports/phase-3/runtime-operations-ux-resource-controls/05-revised-execution-plan.md`（本文件）
- `docs/design/runtime-operations-ux-resource-controls.md`（根据 review 结果更新）

**是否改代码**：否
**是否改前端**：否
**是否改 catalog**：否
**是否改文档**：是

**测试命令**：无（纯文档）

**验收标准**：
- review report 包含所有 evidence-backed findings
- revised execution plan 包含所有阶段
- 无代码变更

---

### Phase 1 — RunPlan lint

**目标**：检测 command/env 冲突，嵌入现有 dry-run response。

**设计要点**：
- 两阶段 lint：pre-normalization（记录用户覆盖）+ final lint（检查最终 command）
- 区分 image-provided env（warning，不 block）与 user-provided env（可 error）
- lint 结果嵌入 dry-run response，不新增 API route
- 不在 deduplicateArgs 之前阻断，而是在两个时机分别检查

**具体文件**：

新建：
- `internal/server/runplan/lint.go`
- `internal/server/runplan/lint_test.go`

修改：
- `internal/server/runplan/resolver.go`（在 buildArgs 中插入 pre-normalization lint hook）
- `internal/server/api/deployment_lifecycle_handlers.go`（HandleDeploymentDryRun 中嵌入 lint 结果）

**是否改代码**：是（Go 后端）
**是否改前端**：否（前端已在 dry-run dialog 中展示 warnings/errors）
**是否改 catalog**：否
**是否改文档**：是（更新 design doc 中 lint 策略）

**测试命令**：
```bash
go test ./internal/server/runplan/ -run TestLint -v
go test ./...
go build ./...
gofmt -l internal/
```

**验收标准**：
- llama.cpp `LLAMA_ARG_HOST`（image-provided）+ `--host` → warning（不 block）
- llama.cpp user-provided `LLAMA_ARG_HOST` + `--host` → error
- llama.cpp `LLAMA_ARG_PORT`（image-provided）+ `--port` → warning
- 重复 `--ctx-size` → error
- 重复 vLLM `--gpu-memory-utilization` → error
- 重复 SGLang `--mem-fraction-static` → error
- privileged / ipc host / unsafe security-opt → warning
- clean vLLM/SGLang/llama.cpp RunPlan → ok
- dry-run response 包含 `lint` 子对象
- lint findings 有稳定 ID、severity、category、message、suggestion、field_path、sources

**关键风险**：
- lint 误报阻断正常部署 → 缓解：默认 warning，仅明确冲突为 error
- image-provided env 判定需要验证 → 缓解：先 inspect llama.cpp image env

---

### Phase 2a — resource_controls 建模

**目标**：在 backend_versions 的 vendor_options_json 中定义各后端支持的 resource controls，实现参数映射。

**设计要点**：
- resource_controls 定义写入 catalog YAML 的 `vendor_options.resource_controls`
- 用户可调参数（如 memory_fraction）放 deployment 的 `parameters_json`，不放 backend_versions
- vendor_options_json 只放"该后端支持哪些 resource controls"的元数据定义
- 不做 shared GPU admission（Phase 2b）

**具体文件**：

修改 catalog YAML：
- `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml`（添加 vendor_options.resource_controls）
- `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml`（同上）
- `configs/backend-catalog/versions/sglang/sglang-v0.5.12.post1.yaml`（同上）
- `configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml`（同上）
- `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml`（同上）

新建：
- `internal/server/runplan/resource_controls.go`（resource_controls 解析和参数映射）
- `internal/server/runplan/resource_controls_test.go`

修改：
- `internal/server/runplan/resolver.go`（在 buildArgs 中根据 resource_controls 注入参数）

**是否改代码**：是（Go 后端）
**是否改前端**：否（前端通过 dry-run command_preview 间接验证）
**是否改 catalog**：是（YAML 添加 vendor_options.resource_controls）
**是否改文档**：是

**测试命令**：
```bash
go test ./internal/server/runplan/ -run TestResourceControls -v
go test ./...
go build ./...
gofmt -l internal/
```

**验收标准**：
- vLLM resource_controls 包含 gpu_memory_fraction, max_model_len, max_num_seqs, max_num_batched_tokens
- SGLang resource_controls 包含 gpu_memory_fraction, max_running_requests, chunked_prefill_size, attention_backend
- llama.cpp resource_controls 包含 gpu_layers, ctx_size, batch_size, ubatch_size, cache_type_k/v, split_mode, main_gpu, tensor_split
- llama.cpp gpu_memory_fraction.supported = false
- resource_controls 定义在 YAML 中，seed reload 正确写入 DB
- 用户通过 deployment parameters_json 设置的参数不被 seed 覆盖
- dry-run command_preview 显示最终 resource 参数

**关键风险**：
- vendor_options_json seed 覆盖 → 缓解：resource_controls 定义写 YAML，用户参数放 deployment
- 参数映射遗漏 → 缓解：Go 单元测试覆盖各后端

---

### Phase 2b — Shared GPU admission（DOCUMENTED_BLOCKER）

**目标**：记录为 BLOCKER，不实现。

**当前平台事实**：
- `db.go:714-716`：`CREATE UNIQUE INDEX idx_gpu_leases_reserved_active ON gpu_leases(gpu_id) WHERE status IN ('reserved','active')`
- 同一 GPU 上不可能有两个 active/reserved 的 lease
- 实现 shared-GPU 需要：删除唯一索引 + 实现 budget admission + 可能新增 resource_budget_json 列

**需要的 schema change**：
- 删除 `idx_gpu_leases_reserved_active` 唯一索引
- 可能新增 `resource_budget_json` 列到 gpu_leases 或 model_instances

**触发条件**：用户明确要求共享 GPU 支持

**文档写入**：在 closeout 或 open-issues 中记录为 DOCUMENTED_BLOCKER。

---

### Phase 3 — Runtime log classifier

**目标**：分类已知 runtime log patterns，用户不需要手动阅读 Docker logs。

**具体文件**：

新建：
- `internal/server/runplan/log_classifier.go`
- `internal/server/runplan/log_classifier_test.go`
- `internal/server/runplan/testdata/runtime-logs/sglang-torchao-syntax-warning.log`
- `internal/server/runplan/testdata/runtime-logs/sglang-attention-backend-default.log`
- `internal/server/runplan/testdata/runtime-logs/llamacpp-env-host-overwritten.log`
- `internal/server/runplan/testdata/runtime-logs/cuda-oom.log`

修改：
- `internal/server/api/deployment_lifecycle_handlers.go`（model test result API 中嵌入 classified_log_events）

**是否改代码**：是（Go 后端）
**是否改前端**：否（classified_log_events 通过 API 返回，前端后续展示）
**是否改 catalog**：否
**是否改文档**：是

**测试命令**：
```bash
go test ./internal/server/runplan/ -run TestLogClassifier -v
go test ./...
go build ./...
gofmt -l internal/
```

**验收标准**：
- SGLang torchao syntax warning → noise/advisory
- SGLang attention backend default → advisory
- llama.cpp host env overwritten → warning
- CUDA OOM → error
- fatal startup traceback → error/fatal
- noise/advisory 不改变实例状态
- classification 输出包含 rule_id, severity, category, message, suggestion, raw_line, occurrences

---

### Phase 4 — 模型实例自动刷新

**目标**：ModelInstancesPage 状态自动更新，不需要手动刷新。

**设计要点**：
- 复用现有 `useAutoRefresh` composable
- 新增 `useInstanceStatusPolling` 作为状态感知封装（不同 state 不同 interval）
- 不新增 status-summary API（先轮询现有 list endpoint）

**具体文件**：

新建：
- `web/src/composables/useInstanceStatusPolling.ts`

修改：
- `web/src/pages/ModelInstancesPage.vue`（引入 useInstanceStatusPolling 替代手动 refresh）

**是否改代码**：否
**是否改前端**：是
**是否改 catalog**：否
**是否改文档**：是

**测试命令**：
```bash
cd web && npm run build
node web/tests/modelCapabilities.test.mjs
# 如果新增 node 测试：
node web/tests/<new-test>.test.mjs
```

**验收标准**：
- transitional states (pending/starting/stopping) 每 2-3s 轮询
- stable states (running/failed/stopped) 每 15-30s 轮询
- document hidden 暂停轮询
- route leave 停止轮询
- 请求失败显示 stale-data warning 并 backoff
- 手动刷新按钮仍然可用
- 页面显示 last refreshed time
- 无 console error
- 无 i18n key leak

---

### Phase 5 — JsonViewer + diagnostics UX

**目标**：Diagnostic JSON 在页面内可读、可滚动、可复制。

**具体文件**：

新建：
- `web/src/components/common/JsonViewer.vue`

修改：
- `web/src/pages/ModelDeploymentsPage.vue`（dry-run dialog 中用 JsonViewer 替换 `<pre>`）
- `web/src/pages/RunnerConfigsPage.vue`（advanced diagnostic JSON 用 JsonViewer 替换 `<pre>`）

**是否改代码**：否
**是否改前端**：是
**是否改 catalog**：否
**是否改文档**：是

**测试命令**：
```bash
cd web && npm run build
node web/tests/modelCapabilities.test.mjs
```

**验收标准**：
- JsonViewer 支持：scroll, fullscreen, copy full content, download, search, line wrap toggle, malformed JSON fallback
- 长 JSON 不溢出页面
- copy 返回完整 JSON
- 同一组件在多个 diagnostic 位置复用
- 无 horizontal page overflow

---

### Phase 6a — HealthCheckEditor + RunnerConfigsPage 结构化编辑

**目标**：RunnerConfigsPage 的 health check 从 textarea 改为结构化编辑器。

**具体文件**：

新建：
- `web/src/components/common/HealthCheckEditor.vue`

修改：
- `web/src/pages/RunnerConfigsPage.vue`（用 HealthCheckEditor 替换 textarea）
- `web/src/pages/BackendsPage.vue`（backend version 编辑中 health_check_json 用 HealthCheckEditor）

**是否改代码**：否
**是否改前端**：是
**是否改 catalog**：否
**是否改文档**：是

**测试命令**：
```bash
cd web && npm run build
node web/tests/modelCapabilities.test.mjs
```

**验收标准**：
- Health check 字段结构化展示：path, method, timeout_seconds, interval_seconds, expected_status, expected_body_contains, readiness_grace_seconds
- 编辑后保存为 health_check_json
- 生成的 health result JSON 为只读展示
- JsonViewer 用于展示 health result 和 diagnostic JSON

---

### Phase 6b — 完整 ConfigEditorLayout（DEFERRED）

**目标**：统一 runtime configuration 和 deployment editing UX。

**Deferred 原因**：范围过大，涉及 BackendRuntimesPage + ModelDeploymentsPage 完整改造，需要 5 个新组件。后续单独立项。

**后续需要的组件**：
- `ConfigEditorLayout.vue`
- `ConfigSection.vue`
- `AdvancedSection.vue`
- `RunPlanPreviewPanel.vue`
- `DiffFromBase.vue`

---

### Phase 7 — 最终验证与 closeout

**目标**：全量回归测试，输出 closeout 文档。

**具体文件**：

新建：
- `docs/reports/phase-3/runtime-operations-ux-resource-controls/closeout.md`

**是否改代码**：否
**是否改前端**：否
**是否改 catalog**：否
**是否改文档**：是

**测试命令**：
```bash
go test ./...
go build ./...
gofmt -l internal/
cd web && npm run build
npm test  # 或 node tests/*.test.mjs
git diff --check
git status --short
```

**验收标准**：
- 所有已知问题 FIXED 或 DOCUMENTED_BLOCKER 或 INVALID
- 所有必跑测试通过
- closeout 文档包含：issue-by-issue status, files changed, tests run, evidence paths, known blockers/deferred items, commit id, push result, git status --short
- 无 Remaining Risk 未在 formal open-issues 文档中

## 3. Deferred / Blocker 清单

| ID | 内容 | 状态 | 触发条件 | 所需变更 |
|----|------|------|----------|----------|
| DEF-001 | Shared-GPU budget admission | DOCUMENTED_BLOCKER | 用户明确要求共享 GPU | 删除 gpu_leases 唯一索引 + budget admission + schema change |
| DEF-002 | 完整 ConfigEditorLayout | deferred | 后续单独立项 | 5 个新组件 + 2 个页面改造 |
| DEF-003 | status-summary API | conditional | 轮询性能不足 | 新增 GET /api/v1/model-instances/status-summary |
| DEF-004 | DB schema promotion of resource_controls | deferred | 需要查询/过滤/版本化 | 新增 resource_controls 列 |
| DEF-005 | llama.cpp VRAM estimator | deferred | 用户要求显存预估 | 新增估算逻辑 |
| DEF-006 | vitest 引入 | deferred | Vue composable 测试必须 | package.json + vitest config |

## 4. Closeout 要求

每个 phase 完成后需记录：
- 修改的文件列表
- 测试命令及结果
- 与 review report findings 的对应关系
- 是否引入新的 DOCUMENTED_BLOCKER

最终 closeout 需包含：
- issue-by-issue status（FIXED / DOCUMENTED_BLOCKER / INVALID）
- files changed
- tests run + evidence paths
- known blockers/deferred items
- commit id + push result
- final `git status --short`

## 5. 最终执行顺序建议

```text
Phase 0 (docs)  ──→  Phase 1 (lint, Go)
                         │
                         ├──→  Phase 2a (resource_controls, Go + YAML)
                         │
                         ├──→  Phase 3 (log classifier, Go)
                         │
                         └──→  Phase 4 (auto-refresh, Vue)
                                   │
                                   ├──→  Phase 5 (JsonViewer, Vue)
                                   │
                                   └──→  Phase 6a (HealthCheckEditor, Vue)
                                              │
                                              └──→  Phase 7 (verification + closeout)
```

Phase 1/2a/3 可以并行（Go 代码，独立包）。Phase 4/5/6a 可以并行（Vue 组件，独立页面）。Phase 7 必须在所有其他 phase 完成后执行。

Phase 2b (shared GPU) 和 Phase 6b (ConfigEditorLayout) 为 deferred，不在本 batch 执行。
