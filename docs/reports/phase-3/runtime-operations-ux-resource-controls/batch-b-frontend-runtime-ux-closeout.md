# Batch B Closeout — Frontend Runtime UX & Diagnostics

> Date: 2026-06-23
> Status: Complete

## 1. 实现摘要

### 1.1 ModelInstancesPage 自动刷新

- 使用 `useInstanceStatusPolling` composable（自管理 timer，非 useAutoRefresh 包装）。
- transitional states（pending/starting/stopping 等）→ 3s 轮询。
- stable states（running/failed/stopped）→ 15s 轮询。
- document hidden → 暂停；window focus → 立即刷新；route leave → 停止。
- 页面 header 显示 last refreshed 时间和 stale data 警告。
- 保留手动刷新按钮。
- 未新增 status-summary API，先使用现有 list endpoint。

### 1.2 JsonViewer

新建 `web/src/components/common/JsonViewer.vue`，支持：
- scroll（max-height 限制）
- fullscreen expand
- copy full content
- download
- search（高亮匹配）
- line wrap toggle
- malformed JSON fallback（raw text 展示）

已接入：
- ModelDeploymentsPage dry-run dialog（替换 `<pre>`）
- RunnerConfigsPage detail drawer advanced diagnostic JSON（替换 `<pre>`）
- RunnerConfigsPage detail drawer health check 展示

### 1.3 HealthCheckEditor

新建 `web/src/components/common/HealthCheckEditor.vue`，结构化字段：
- path
- method（GET/POST select）
- timeout_seconds
- interval_seconds
- expected_status
- expected_body_contains
- readiness_grace_seconds

已接入：
- RunnerConfigsPage edit dialog（替换 textarea raw JSON）
- Raw JSON 折叠在高级区

### 1.4 classified_log_events 展示

- ModelInstancesPage logs drawer 中新增分类事件展示区。
- 读取 `classified_log_events` 字段。
- 按 severity 显示不同颜色标签（error=danger, warning=warning, advisory=info）。
- noise/advisory 不显示成失败。
- 展示 rule_id、message、suggestion、occurrences。

### 1.5 resource_controls 前端展示

- dry-run command_preview 和 warnings 已在 ModelDeploymentsPage 展示。
- lint 子对象通过 JsonViewer 可查看。
- 未做复杂资源调度 UI。

### 1.6 useInstanceStatusPolling

新建 `web/src/composables/useInstanceStatusPolling.ts`：
- 自管理 timer（不依赖 useAutoRefresh 包装，避免动态 interval 问题）。
- 根据实例状态动态调整 interval：transitional=3s, stable=15s。
- 通过 `watch(intervalMs)` 在状态变化时自动重启 timer。
- 已接入 ModelInstancesPage。

## 2. 修改文件

### 新增文件

| 文件 | 说明 |
|------|------|
| `web/src/components/common/JsonViewer.vue` | JSON 查看器组件 |
| `web/src/components/common/HealthCheckEditor.vue` | 健康检查结构化编辑器 |
| `web/src/composables/useInstanceStatusPolling.ts` | 状态感知轮询 composable |
| `docs/reports/phase-3/runtime-operations-ux-resource-controls/batch-b-frontend-runtime-ux-closeout.md` | 本 closeout |

### 修改文件

| 文件 | 变更 |
|------|------|
| `web/src/pages/ModelInstancesPage.vue` | useInstanceStatusPolling 替代手动 refresh（transitional 3s / stable 15s）；classified_log_events 展示；last refreshed 显示 |
| `web/src/pages/ModelDeploymentsPage.vue` | JsonViewer 替换 dry-run `<pre>` |
| `web/src/pages/RunnerConfigsPage.vue` | JsonViewer 替换 diagnostic `<pre>`；HealthCheckEditor 替换 health check textarea |
| `web/src/locales/en-US.ts` | 新增 i18n keys |
| `web/src/locales/zh-CN.ts` | 新增 i18n keys |

## 3. 测试命令和结果

```
$ cd web && npm run build
✓ built in 3.41s (vue-tsc + vite build pass)

$ cd web && npm test
0 FAIL, all tests pass

$ git diff --check
(no output)
```

## 4. 未做事项

| 事项 | 原因 |
|------|------|
| status-summary API | Batch B 不新增后端 API，先用现有 list endpoint |
| complete ConfigEditorLayout | Deferred，后续单独立项 |
| shared GPU admission | DOCUMENTED_BLOCKER |
| vitest introduction | 使用现有 node test 模式 |
| BackendsPage HealthCheckEditor | BackendsPage 的 health_check_json 是 raw JSON textarea，风险较大，先只做 RunnerConfigsPage |

## 5. Commit 信息

- **commit id**: (待提交后填写)
- **push result**: (待推送后填写)
- **git status**: `M VERSION`（既有修改，本轮未处理）、`?? .mimocode/skills/`（MiMoCode 内部目录，未入库）
