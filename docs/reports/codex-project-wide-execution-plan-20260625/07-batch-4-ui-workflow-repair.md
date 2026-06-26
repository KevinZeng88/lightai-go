# Batch 4 — UI Workflow Repair

## 目标

修复前端误导入口，使 UI 与 NBR/RunPlan/Deployment contract 一致，并补浏览器级 smoke。

覆盖：

- R-004
- R-011 UI 侧
- Q-004
- frontend review findings
- 参数/部署/运行配置体验一致性

## 任务

### 4.1 修复部署编辑 runtime selector

当前问题：部署编辑表单显示 `backend_runtime_id` 选择器，但 `doEdit()` 不提交，用户误以为 runtime/NBR 已改变。

默认处理：

- 移除部署编辑里的 runtime selector。
- 部署创建时选择 NBR / 运行配置；部署创建后，普通编辑页不再提供看似可以切换运行配置的 selector。
- 运行配置可以只读展示，例如 source NBR、BackendRuntime、snapshot 信息。
- 如果后续需要改变运行配置，可以通过删除重建部署，或将来单独实现明确的 NBR change flow。
- 当前阶段不要保留“UI 看起来能改，但后端不生效”的入口。

默认优先级：

```text
先去除误导功能 > 后续单独实现完整 NBR change flow > 保留假功能
```

显式 NBR change flow 要求：

如果 Claude 判断完整 NBR change flow 实现很简单，也可以补充实现，但必须满足：

1. 有明确入口，而不是混在普通编辑字段里。
2. API 真正支持修改 source NBR 或创建新的 deployment snapshot。
3. UI 显示当前 NBR 与目标 NBR diff。
4. 修改后 NBR 需要重新 check 或明确继承已验证状态。
5. 明确 snapshot semantics。
6. 明确不会 live mutation 已运行实例。
7. preview / dry-run / start 使用新的最终 RunPlan。
8. API / UI / E2E 测试覆盖。
9. 用户不会误以为已经修改运行中实例。

如果本批不实现完整 NBR change flow，则只移除误导字段。

本项目不允许出现“看起来可以操作，但实际没有生效”的功能。

### 4.2 清理 ID-heavy create dialog

部署创建如果已有 wizard，则普通用户路径不应要求手工输入 artifact ID/node ID。

处理：

- 将旧 create dialog 标记为 advanced/admin。
- 或隐藏旧 dialog，只保留 wizard。
- 所有 label 使用 NBR/运行配置一致术语。
- UI 不再显示 BackendRuntime 与 NodeBackendRuntime 混用字段名。

### 4.3 聚合 NBR endpoint

后端新增 tenant-scoped aggregate endpoint，例如：

```text
GET /api/v1/node-backend-runtimes
GET /api/v1/backend-runtimes/node-configs
```

返回：

- node metadata
- backend runtime metadata
- NBR id
- status
- deployable
- warnings
- disabled_reason
- image
- parameter summary

前端替换 per-node loop：

- `ModelDeploymentsPage.vue loadAllNBRs()`
- `BackendRuntimesPage.vue loadNodeRuntimes()`

### 4.4 ready_with_warnings UI 表达

UI 应：

- 允许 deployable with warnings。
- 明确显示 warnings。
- 不把 warnings 混同 error。
- preflight/start 按后端 deployable 字段展示。

### 4.5 参数编辑一致性

检查 `RuntimeParameterEditor`：

- BR/NBR/Deployment 层字段名一致。
- disabled 时仍显示输入值。
- enabled 与 value 分离保存。
- 不显示与模型页无关的 Docker 参数。
- 保存后刷新不 OOM。
- clone/copy 保留 enabled + value。

### 4.6 Playwright/browser smoke

如果项目已有 Playwright dependency，则新增：

```text
web/tests/e2e/model-runtime-workflow.spec.ts
```

覆盖：

- 模型/运行配置/NBR/deployment preview 主路径。
- non-ready NBR blocked。
- ready_with_warnings 可选但显示 warning。
- deployment edit 不再显示误导 runtime selector。
- failed logs 可查看。

如果浏览器环境不可用，则至少新增可运行的 component tests，并记录 browser smoke blocked 原因；但最终 release 前必须补 browser smoke。

## 验证命令

```bash
cd web && npm test
cd web && npm run build
# 如果已有 Playwright:
cd web && npx playwright test
go test ./internal/server/api
go test ./...
```

## 验收

- R-004 CLOSED。
- R-011 UI fan-out CLOSED 或后端聚合 endpoint 已可用。
- UI 不再展示后端忽略字段。
- 部署编辑页不再出现不会生效的 runtime selector；或者 selector 的变更真实提交、生效、可验证。
- 不存在“看似可操作但不生效”的 UI。
- 相关测试覆盖 UI 表单字段与 API payload 一致性。
- UI 术语统一。
- 浏览器 smoke 或明确 blocked evidence 存在。
- Batch closeout 记录截图/测试/构建结果。
