# Claude Review Report — Runtime Operations UX & Resource Controls

> Date: 2026-06-23
> Reviewer: Claude (design review, no code changes)
> Input: design doc, known issues, implementation plan, verification plan, review prompt, user decisions
> Status: Review complete. Accepted with corrections. See `05-revised-execution-plan.md` for final plan.
> Scope: This report re-verifies 6 key findings from prior round against current codebase. No code/config/schema changes made.

## A. 总体结论

**是否认可设计方向：是。** 8 个已知问题都是真实的用户痛点，设计提出的分层方案（RunPlan lint → resource controls → log classifier → polling → JsonViewer → ConfigEditorLayout）逻辑清晰。

**是否建议执行：是，按修正后计划执行。** 存在若干与现有代码不一致的假设以及过度设计的部分，修正后可以安全执行。

**是否存在硬阻塞：无硬阻塞。** 有 1 个 DOCUMENTED_BLOCKER（shared-GPU admission），其余可本批实施或 deferred。

**本批可做：** RunPlan lint（两阶段）、resource_controls 建模（vendor_options_json）、runtime log classifier、model instance auto-refresh（复用 useAutoRefresh）、JsonViewer + diagnostics UX、HealthCheckEditor。

**必须 deferred：** shared GPU admission（需 schema change）、完整 ConfigEditorLayout（Phase 6b）、status-summary API（conditional）。

## B. Evidence-backed findings

### B-1. LLAMA_ARG_HOST / LLAMA_ARG_PORT 来源

- **问题**：设计文档将 llama.cpp `LLAMA_ARG_HOST` + `--host` 冲突定性为"RunPlan 生成错误"。
- **Evidence**：
  - `grep -r LLAMA_ARG internal/` — **零匹配**。平台 Go 代码不设置 `LLAMA_ARG_HOST` 或 `LLAMA_ARG_PORT`。
  - `configs/backend-catalog/runtimes/llamacpp/nvidia-cuda13.yaml:37` — YAML 注释明确写到：`# --host and --port come from version default_args; not duplicated here / # to avoid triggering LLAMA_ARG_HOST / --host warnings in upstream llama-server.`
  - `docs/reports/phase-3/web-ai-config-review/21-phase-2-runtime-command-cleanup-closeout.md:48` — "The llama.cpp Docker image (`ghcr.io/ggml-org/llama.cpp:server-cuda13`) has a built-in `LLAMA_ARG_HOST` environment variable. The platform adds `--host 0.0.0.0` which correctly overrides it."
  - `internal/server/db/db.go:1382` — llama.cpp seed `default_args_json` 包含 `"--host","0.0.0.0","--port","{{container_port}}"`，但不设置任何 `LLAMA_ARG_*` env。
- **严重性**：中。设计方向正确（lint 检测冲突），但实现策略必须区分 image-provided env 与 user-provided env。
- **影响**：Phase 1 RunPlan lint 和 Phase 3 log classifier 的设计。
- **处理**：lint 区分来源，image-provided 先 warning 不 block，user-provided 可 error。log classifier 分类该 warning。不未经验证就写"unset 能消除 warning"。

### B-2. gpu_leases 唯一索引已实现独占 GPU 语义

- **问题**：设计文档 §9 提出"existing exclusive instance blocks new placement on same GPU"作为新 admission rule。
- **Evidence**：
  - `internal/server/db/db.go:715-716` — `CREATE UNIQUE INDEX IF NOT EXISTS idx_gpu_leases_reserved_active ON gpu_leases(gpu_id) WHERE status IN ('reserved','active')`
  - `internal/server/db/db.go:712` — 注释："migrateV8 adds a partial unique index on gpu_leases to prevent concurrent"
  - `internal/server/runplan/dryrun.go:66` — `SELECT id FROM gpu_leases WHERE gpu_id = ? AND status IN ('reserved','active')` 用于检查 GPU 是否已被占用。
  - 同一 GPU 上不可能有两个 active/reserved 的 lease，数据库层面已强制执行独占语义。
- **严重性**：高。设计假设需要新建 admission check，但数据库层面已实现。
- **影响**：Phase 2 resource admission 的整个设计。
- **处理**：shared GPU 本 batch 不做 schema change。Phase 2b 改为 DOCUMENTED_BLOCKER：GPU Lease Shared Mode / Budget Admission。本 batch 只做 resource_controls 建模和参数预览。

### B-3. useAutoRefresh 已存在，ModelInstancesPage 未使用

- **问题**：设计文档 §11.1 提出创建 `useAutoRefresh.ts`。
- **Evidence**：
  - `web/src/composables/useAutoRefresh.ts` — 已存在，104 行，功能完整：visibility API 暂停（`document.hidden`）、route leave 停止（`router.beforeEach`）、inflight guard、focus 刷新、interval 轮询。
  - `web/src/composables/__tests__/useAutoRefresh.test.ts` — 已有 vitest 测试。
  - `web/src/pages/NodesPage.vue:174` — `import { useAutoRefresh } from '@/composables/useAutoRefresh'`，已使用。
  - `web/src/pages/GpusPage.vue:162` — 同上，已使用。
  - `web/src/pages/DashboardPage.vue:153` — 同上，已使用。
  - `web/src/pages/ModelInstancesPage.vue` — **未使用** useAutoRefresh。手动实现 `refresh()` + `onMounted` 单次加载，无自动轮询。logsTimer 仅用于日志刷新（3s），不用于实例列表状态。
- **严重性**：低。
- **处理**：复用现有 useAutoRefresh，不新建重复 composable。新增 useInstanceStatusPolling 作为状态感知封装（不同状态不同轮询间隔）。

### B-4. deduplicateArgs 已存在，保留最后出现

- **问题**：设计文档 §7 提出 RunPlan lint 检测 duplicate CLI flags。
- **Evidence**：
  - `internal/server/runplan/resolver.go:446-449` — `deduplicateArgs()` 注释："removes duplicate --flag value pairs, keeping the LAST occurrence (highest priority — user parameters from Layer 4 override defaults from Layer 1)."
  - `internal/server/runplan/resolver.go:394` — `args = deduplicateArgs(args)` 在 resolver 流程中调用。
  - `docs/reports/model-runtime-node-wizard/e2e-improvement/04-claude-review-and-implementation-plan.md:183` — "deduplicateArgs keeps first (00 §2.3) — Fixed in commit 015180c. Now keeps last."
- **严重性**：中。lint 如果只在 dedup 之后运行会丢失用户覆盖证据。
- **处理**：设计为两阶段 lint：(1) pre-normalization lint 记录用户输入重复/覆盖；(2) normalize/deduplicate；(3) final lint 检查最终 command 的 env/CLI conflict、高风险参数。UI 展示最终 lint，同时保留 user override evidence。

### B-5. dry-run API 已返回 valid/errors/warnings/command_preview

- **问题**：设计文档 §12.1 提出在 RunPlan preview/preflight response 中添加 lint 和 resource_admission。
- **Evidence**：
  - `internal/server/api/deployment_lifecycle_handlers.go:1822-1835` — `HandleDeploymentDryRun` 返回：
    ```go
    result := map[string]interface{}{
        "valid":  valid,
        "errors": pf.errs, "error_details": ...,
        "warnings": pf.warns,
    }
    if pf.plan != nil {
        result["command_preview"] = pf.commandPreview
        result["selected_node"] = pf.placement.NodeID
        result["selected_runtime"] = pf.runtimeID
        result["selected_model_location"] = pf.locationID
        result["resolved_image"] = pf.plan.Image
    }
    ```
  - `internal/server/api/router.go:199` — `POST /api/v1/deployments/{id}/dry-run` 路由已存在。
  - 现有 dry-run 已经使用 `preflightDeployment()` 做完整验证（BRR-RV-001）。
- **严重性**：低。不需要新增 API route。
- **处理**：lint 结果直接合并到 dry-run response 的 `warnings`/`errors` 数组中，加上 `lint` 子对象。

### B-6. 前端测试框架现状

- **问题**：设计文档中的前端测试假设 vitest 可用。
- **Evidence**：
  - `web/package.json:11` — `"test": "node tests/apiClientPaths.test.mjs && node tests/formatters.test.mjs && node tests/i18nKeys.test.mjs && ..."` — 主测试命令使用 `node tests/*.test.mjs`。
  - `web/package.json:21-27` — devDependencies 中**没有 vitest**。只有 `@playwright/test`, `@vitejs/plugin-vue`, `typescript`, `vite`, `vue-tsc`。
  - `web/src/composables/__tests__/useAutoRefresh.test.ts:5` — `import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'` — 该文件引用 vitest 但 vitest 未在 devDependencies 中。
  - `web/src/stores/__tests__/auth.test.ts:8` — 同上，引用 vitest。
  - `web/src/pages/__tests__/dashboard.test.ts:5` — 同上。
  - **结论**：vitest 测试文件存在但 vitest 未在 package.json 中声明为依赖。主测试命令使用 node 直接运行 `.test.mjs` 文件。
- **严重性**：中。
- **处理**：前端测试优先使用现有 `node web/tests/*.test.mjs` 模式，不引入 vitest。如果发现必须引入 vitest，单独提出计划，不在本 batch 直接做。

### B-7. JsonViewer 组件不存在

- **问题**：设计文档 §11.2 提出创建 `JsonViewer.vue`。
- **Evidence**：
  - `glob web/src/**/*JsonViewer*` — 零匹配。该组件不存在。
  - `grep JsonViewer web/src/` — 零匹配。无任何引用。
  - 当前 JSON 显示方式：各页面使用 `<el-input type="textarea">` 或 `<pre>` 标签。
- **严重性**：低。需要新建。
- **处理**：Phase 5 新建 `web/src/components/common/JsonViewer.vue`。

### B-8. health check JSON / advanced diagnostics 当前页面位置

- **问题**：设计文档提到 health check 和 diagnostic JSON 边界不清。
- **Evidence**：
  - `web/src/pages/RunnerConfigsPage.vue:247` — health check 使用 `<el-input v-model="editHealthText" type="textarea" :rows="4" />`，原始 JSON textarea。
  - `web/src/pages/RunnerConfigsPage.vue:180` — health check 在详情面板中显示为 `runParamSummary(selected).health`。
  - `web/src/pages/BackendsPage.vue:66` — backend version 表单中 health check 使用 `<el-input v-model="versionForm.health_check_json" type="textarea" :rows="3" />`。
  - `web/src/pages/ModelDeploymentsPage.vue:107` — dry-run 结果中显示 health check。
  - `web/src/pages/BackendRuntimesPage.vue:400` — runtime 详情 JSON dump 包含 `health_check_override_json`。
  - **没有找到** "Advanced Diagnostic JSON" 独立展示区域。诊断 JSON 可能在 instance detail 或 test result 中。
- **严重性**：中。health check 当前是原始 JSON textarea，需要结构化编辑器。
- **处理**：Phase 6a 做 HealthCheckEditor + RunnerConfigsPage 结构化 health check 编辑。

### B-9. resource_controls 放置字段：vendor_options_json

- **问题**：设计建议 resource_controls 放在 catalog/seed JSON 中，但未明确字段。
- **Evidence**：
  - `internal/server/db/db.go:1395` — seedTargetBackendCatalog INSERT/UPDATE 都硬编码 `vendor_options_json='{}'`。
  - `internal/server/db/db.go:1694` — `ALTER TABLE backend_versions ADD COLUMN vendor_options_json TEXT NOT NULL DEFAULT '{}'` — V12 migration。
  - `internal/server/api/backend_handlers.go:938` — API 返回 `VendorOptions: jsonToAny(rawJSONString(get("vendor_options_json", ...), "{}"))`。
  - `internal/server/api/runtime_handlers.go:1129` — runtime API 也返回 `vendor_options_json`。
  - `capabilities_json` 当前包含 `supported_formats`, `supported_tasks`, `supported_capabilities`, `model_path_modes`, `test_endpoints`, `blocked_architectures`。不适合混入 resource_controls。
- **严重性**：低。
- **处理**：暂放 `vendor_options_json`，当前无 schema 方案。不等于最终 contract 字段。后续如需查询/过滤/版本化再 schema 化。

### B-10. catalog seed 覆盖 vendor_options_json

- **问题**：seed 是否会覆盖用户对 vendor_options_json 的修改？
- **Evidence**：
  - `internal/server/db/db.go:1391-1395` — `UPDATE backend_versions SET ... vendor_options_json='{}' ... WHERE id=?` — **是的，seed UPDATE 会强制将 vendor_options_json 重置为 `'{}'`。**
  - `internal/server/db/db.go:1390` — INSERT 也硬编码 `'{}'`。
  - 但 `backend_handlers.go:789` — 用户 catalog 导入时使用 `get("vendor_options_json", "{}")` 从 YAML 读取，如果 YAML 中有值则保留。
  - `backend_handlers.go:723` — PATCH 更新时 `vendor_options_json` 在允许的 JSON 字段列表中，用户可以修改。
  - **结论**：系统内置 backend 的 vendor_options_json 会被 seed 强制重置为 `'{}'`。用户自定义 backend（managed_by='user'）不受影响。如果要给系统 backend 添加 resource_controls，需要修改 seed 逻辑。
- **严重性**：中。
- **处理**：实施时需要在 seed 中加入 resource_controls JSON，或修改 seed 逻辑不覆盖 vendor_options_json 中的 resource_controls 字段。测试验证 seed 不覆盖用户配置。

## C. 设计修正建议

### 保留的设计

| 内容 | 说明 |
|------|------|
| RunPlan lint 两阶段 | 设计合理，pre-normalization + final lint |
| Runtime log classifier | Go 内置规则，fixture 测试，不改实例状态 |
| JsonViewer 组件 | 新建，无现有可复用 |
| resource_controls vendor_options_json | 当前无 schema 方案可行 |
| useAutoRefresh 复用 | 已有完整实现，只新增状态感知封装 |

### 需要调整的设计

| 原设计 | 调整 | 原因 |
|--------|------|------|
| Phase 2 resource admission | 拆为 2a + 2b | gpu_leases 唯一索引已实现独占，shared 需 schema change |
| Phase 6 ConfigEditorLayout | 拆为 6a + 6b | 范围过大，6a 做 JsonViewer + HealthCheckEditor，6b deferred |
| useAutoRefresh 新建 | 不新建 | 已存在且功能完整 |
| dry-run 新增 lint API | 不新增 | 合并到现有 response |
| status-summary API | conditional | 先轮询现有 list endpoint |
| vitest 引入 | 不引入 | 使用 node tests/*.test.mjs |

### 需要拆分的阶段

| 原阶段 | 拆分方式 |
|--------|----------|
| Phase 2 | 2a: resource_controls 建模；2b: shared-GPU admission → DOCUMENTED_BLOCKER |
| Phase 6 | 6a: JsonViewer + HealthCheckEditor；6b: ConfigEditorLayout → deferred |

### Deferred / Blocker

| 内容 | 状态 | 说明 |
|------|------|------|
| Shared-GPU admission (budget-based) | DOCUMENTED_BLOCKER | 需要删除 gpu_leases 唯一索引 + 实现 budget admission |
| Phase 6b ConfigEditorLayout | deferred | 后续单独立项 |
| Status-summary API | conditional | 仅在轮询性能不足时添加 |

## D. Schema Change 评估

### 不需要 Schema Change

| 功能 | 实现方式 |
|------|----------|
| resource_controls | 存入现有 `vendor_options_json` 字段 |
| RunPlan lint 结果 | 不持久化，运行时计算，嵌入 dry-run response |
| Log classifier rules | Go 内置，不存 DB |
| GPU admission (独占模式) | 现有 `gpu_leases` 唯一索引已实现 |

### Deferred Schema Change

| 功能 | 需要的变更 | 触发条件 |
|------|-----------|----------|
| shared-GPU admission | 删除 `idx_gpu_leases_reserved_active` 唯一索引 | 用户明确要求共享 GPU |
| instance resource budget | 新增 `resource_budget_json` 列 | 需要持久化 budget 用于 admission |

### vendor_options_json 使用说明

- 这是当前无 schema 方案，不等于最终 contract 字段。
- 后续如果需要查询/过滤/版本化，再 schema 化。
- 需要测试 seed 不覆盖用户配置，YAML 与 DB seed 一致。
- **关键风险**：seedTargetBackendCatalog 的 UPDATE 语句会强制重置 vendor_options_json='{}'，实施时需要修改 seed 逻辑或在 seed 中预置 resource_controls。

## E. 风险与回滚点

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| lint 误报导致正常部署被阻断 | 中 | 高 | lint 默认 severity 为 warning 不阻断；error 级别需要明确的冲突规则 |
| seed 强制重置 vendor_options_json | 高 | 中 | 修改 seed 逻辑保留 resource_controls，或在 seed 中预置 |
| useAutoRefresh 高频轮询增加 server 负载 | 低 | 低 | 稳定状态 15-30s 间隔，document hidden 暂停 |
| vitest 依赖缺失导致测试不可运行 | 中 | 低 | 使用 node tests/*.test.mjs 模式 |

### 回滚点

每个 phase 独立可回滚，不影响其他 phase。

| Phase | 回滚方式 |
|-------|----------|
| Phase 1 | 删除 lint.go，revert resolver.go 变更 |
| Phase 2a | revert vendor_options_json seed 变更 |
| Phase 3 | 删除 log_classifier.go，revert API 变更 |
| Phase 4 | revert ModelInstancesPage.vue 到手动刷新 |
| Phase 5 | 删除 JsonViewer.vue，revert 页面到 `<pre>` |
| Phase 6a | 删除 HealthCheckEditor.vue，revert 到 textarea |

## F. 用户确认的决策

| 问题 | 决策 |
|------|------|
| Shared GPU 本 batch 是否做？ | 否。Phase 2b → DOCUMENTED_BLOCKER。 |
| LLAMA_ARG_HOST 处理策略？ | 选 C（生成器避免 + lint 防御），分步验证，不未经验证就写"unset 能消除 warning"。 |
| RunPlan lint 运行时机？ | 两阶段：pre-normalization + post-dedup final lint。 |
| vendor_options_json 使用？ | 可接受，补说明，测试 seed 不覆盖。 |
| useAutoRefresh 是否新建？ | 不新建，复用现有，只新增 useInstanceStatusPolling。 |
| Phase 6b 是否本 batch？ | 否。ConfigEditorLayout deferred。 |
| 前端测试框架？ | 使用 node web/tests/*.test.mjs，不引入 vitest。 |

## G. 文档引用

本次 review 基于以下文件和代码位置：

1. `docs/design/runtime-operations-ux-resource-controls.md`
2. `docs/reports/phase-3/runtime-operations-ux-resource-controls/00-known-issues-and-evidence.md`
3. `docs/reports/phase-3/runtime-operations-ux-resource-controls/01-implementation-plan.md`
4. `docs/reports/phase-3/runtime-operations-ux-resource-controls/02-verification-and-acceptance-plan.md`
5. `docs/reports/phase-3/runtime-operations-ux-resource-controls/03-claude-review-prompt.md`
6. `docs/reports/phase-3/web-ai-config-review/21-phase-2-runtime-command-cleanup-closeout.md` (WEB-AI-RC-002)
7. 代码：`internal/server/runplan/resolver.go` (deduplicateArgs), `internal/server/db/db.go` (gpu_leases, seed), `internal/server/api/deployment_lifecycle_handlers.go` (dry-run), `web/src/composables/useAutoRefresh.ts`, `web/src/pages/ModelInstancesPage.vue`, `configs/backend-catalog/runtimes/llamacpp/nvidia-cuda13.yaml`

本轮只做设计审查，不修改代码。
