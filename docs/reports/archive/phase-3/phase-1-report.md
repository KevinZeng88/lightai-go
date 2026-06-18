> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Phase 1 完成报告

> 日期：2026-06-16
> 状态：已完成

## 1. 修改的文件

| 文件 | 修改说明 |
|------|----------|
| `cmd/server/main.go` | 移除 ModelHandler 初始化 + sweep loop |
| `internal/server/api/router.go` | 移除所有旧模型运行 API 路由，保留 agent/node/gpu/auth/rbac/audit 路由 |
| `internal/server/models/models.go` | 移除所有旧模型运行 struct（9 个类型） |
| `internal/server/auth/bootstrap.go` | 替换 22 个旧权限点为 20 个新权限点，更新 3 个角色权限映射 |
| `internal/server/db/db.go` | 新增 V10 migration + migrateV10() + seedBuiltInBackends() |
| `web/src/router/index.ts` | 移除 5 个旧模型运行页面路由 |
| `web/src/locales/zh-CN.ts`, `en-US.ts` | 移除旧 i18n 键块 |

## 2. 新增的文件

**Server (10 files):**
- `internal/server/api/helpers.go` — HTTP/JSON/审计/脱敏 helper 函数
- `internal/server/api/constants.go` — Task/Instance/Lease 状态常量
- `internal/server/models/backend.go` — InferenceBackend, BackendVersion, BackendRuntimeTemplate
- `internal/server/models/runtime.go` — BackendRuntime, NodeRuntimeOverride
- `internal/server/models/artifact.go` — ModelArtifact
- `internal/server/models/deployment.go` — ModelDeployment
- `internal/server/models/instance.go` — ModelInstance, GpuLease, AgentTask
- `internal/server/models/runplan.go` — ResolvedRunPlan (DB model)
- `internal/server/runplan/types.go` — RunPlan resolver 类型定义

**Config (13 YAML files):**
- `configs/model-runtime/backends/` — vllm.yaml, sglang.yaml, llamacpp.yaml
- `configs/model-runtime/backend-versions/` — 5 files (vllm 0.8.5/0.10.0, sglang 0.4.6/0.5.0, llamacpp b4817)
- `configs/model-runtime/backend-runtime-templates/` — 5 files (vllm-nvidia/metax, sglang-nvidia/metax, llamacpp-nvidia)

## 3. 删除的文件

**Server (10 files):** model_handlers.go, deployment_lifecycle.go, instance_state.go, lease.go, task_handlers.go, task_constants.go, sweep.go, resolve_helper.go, resolver/resolver.go, model_handlers_test.go, rbac_phase2f_test.go

**Web (10 files):** 5 Vue pages + 5 API client files

**Config:** configs/templates/runtime/, configs/templates/run/, configs/templates/docker-images.json, docs/templates-config.md

## 4. 数据库 migration 变化

V10 migration：删除 7 个旧表 + 创建 9 个新表 + seed 3 个后端 + 5 个版本。不创建 backend_runtime_templates 表。

## 5. 新增表结构

10 个表全部带 tenant_id（除全局 inference_backends/backend_versions）：inference_backends, backend_versions, backend_runtimes, node_runtime_overrides, model_artifacts, model_deployments, model_instances, resolved_run_plans, gpu_leases, agent_tasks。

## 6. 配置目录

13 个 YAML 文件分布在 backends/ (3), backend-versions/ (5), backend-runtime-templates/ (5)。

## 7. 权限常量

20 个新权限点 + 3 个角色映射（viewer: 12 perms, operator: 21 perms, admin: 28 perms）。

## 8. 验证结果

- go build ./cmd/server/ — ✓
- go build ./cmd/agent/ — ✓
- go test ./... — ✓ 8 packages
- npm run build — ✓

## 9. 已知问题

1. `rbac_phase2f_test.go` 已删除。Phase 4 需新增新 RBAC 测试，覆盖 401/403/跨租户/脱敏。
2. `model_handlers_test.go` 已删除。Phase 4 需新增新 API handler 测试。
3. Web i18n 导航需在 Phase 5 更新。
4. ConsoleLayout.vue 侧边栏旧菜单项需在 Phase 5 更新。

## 10. 是否建议进入 Phase 2

是 — Phase 1 删除和新表建设已完成，可以进入 RunPlan Resolver。
