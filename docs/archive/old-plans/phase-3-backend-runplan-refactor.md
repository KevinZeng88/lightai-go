> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# LightAI Go Phase 3：模型运行链路重构计划

> Phase 0.1 修订版 — 根据人工审核意见更新。
> 日期: 2026-06-16

## 1. 总体重构目标

废弃当前 `RuntimeEnvironment + RunTemplate + ModelDeployment + ModelInstance + AgentTask` 旧链路，替换为：

```
InferenceBackend → BackendVersion → BackendRuntimeTemplate → BackendRuntime
  → NodeRuntimeOverride → ModelArtifact → ModelDeployment → ModelInstance
  → ResolvedRunPlan (独立表) → DockerExecutor
```

---

## 2. 旧链路删除范围

### 2.1 删除的数据库表

```sql
DROP TABLE IF EXISTS runtime_environment_docker_specs;
DROP TABLE IF EXISTS runtime_environments;
DROP TABLE IF EXISTS run_templates;
DROP TABLE IF EXISTS model_deployments;
DROP TABLE IF EXISTS model_instances;
DROP TABLE IF EXISTS gpu_leases;
DROP TABLE IF EXISTS agent_tasks;
-- model_artifacts 保留
```

### 2.2 删除的 Server 文件

| 文件 | 说明 |
|------|------|
| `internal/server/api/model_handlers.go` | 全部 CRUD handler |
| `internal/server/api/deployment_lifecycle.go` | Start/Stop 生命周期 |
| `internal/server/api/instance_state.go` | Instance 状态更新 helpers |
| `internal/server/api/lease.go` | GPU lease 管理 |
| `internal/server/api/task_handlers.go` | AgentTask 结果处理 |
| `internal/server/api/task_constants.go` | Task 常量 |
| `internal/server/api/sweep.go` | 过期 task/lease 清理 |
| `internal/server/api/resolve_helper.go` | Dry-run 辅助函数 |
| `internal/server/resolver/resolver.go` | 旧 Resolve 逻辑 |

### 2.3 删除的 Web 文件

旧 RuntimeEnvironments、RunTemplates、ModelArtifacts、ModelDeployments、ModelInstances 页面及对应 API client 文件。

### 2.4 删除的配置文件

- `configs/templates/runtime/`（整个目录）
- `configs/templates/run/`（整个目录）
- `configs/templates/docker-images.json`
- `docs/templates-config.md`

### 2.5 删除的 API 路由

所有 `/api/v1/runtime-environments`、`/api/v1/run-templates`、旧 `/api/v1/model-artifacts`、旧 `/api/v1/model-deployments`、旧 `/api/v1/model-instances`、`/api/v1/gpu-leases`、`/api/v1/agent/tasks/{id}/result` 路由。

---

## 3. 新链路建设范围

### 3.1 新增/修改的数据库表

| 表 | 操作 | 说明 |
|----|------|------|
| `inference_backends` | 新建 | 后端家族，含 default_version/common_parameters/default_env |
| `backend_versions` | 新建 | 后端版本，含 is_default/default_backend_params/env_json |
| `backend_runtimes` | 新建 | 用户可编辑运行配置（vendor + image + devices + docker） |
| `node_runtime_overrides` | 新建 | 节点级 image/env/device/modelRoot 覆盖 |
| `model_artifacts` | 保留（微调） | source_type → source |
| `model_deployments` | 重建 | 新结构（backend_runtime_id，placement_json，parameters_json） |
| `model_instances` | 重建 | current_run_plan_id → resolved_run_plans |
| `resolved_run_plans` | 新建 | 独立 RunPlan 表（不可变） |
| `gpu_leases` | 重建 | 保持现有结构 |
| `agent_tasks` | 重建 | 保持现有结构 |

### 3.2 新增配置目录

```
configs/model-runtime/
  backends/
    vllm.yaml
    sglang.yaml
    llamacpp.yaml
  backend-versions/
    vllm/0.8.5.yaml, 0.10.0.yaml
    sglang/0.4.6.yaml, 0.5.0.yaml
    llamacpp/b4817.yaml
  backend-runtime-templates/
    vllm-nvidia-docker.yaml
    vllm-metax-docker.yaml
    sglang-nvidia-docker.yaml
    sglang-metax-docker.yaml
    llamacpp-nvidia-docker.yaml
```

### 3.3 新增 Server 文件

| 文件 | 说明 |
|------|------|
| `internal/server/api/backend_handlers.go` | Backend/BackendVersion/BackendRuntimeTemplate 只读 API |
| `internal/server/api/runtime_handlers.go` | BackendRuntime + NodeRuntimeOverride CRUD |
| `internal/server/api/artifact_handlers.go` | ModelArtifact CRUD |
| `internal/server/api/deployment_handlers.go` | ModelDeployment CRUD + lifecycle |
| `internal/server/api/instance_handlers.go` | ModelInstance 只读 + logs |
| `internal/server/api/runplan_handlers.go` | RunPlan preview + query |
| `internal/server/api/lease_handlers.go` | GPU lease 管理 |
| `internal/server/api/task_handlers.go` | AgentTask 管理 |
| `internal/server/runplan/resolver.go` | RunPlan Resolver（{{var}} only） |
| `internal/server/runplan/dryrun.go` | Dry Run / Preview 校验器 |

### 3.4 新增 Web 文件

| 文件 | 说明 |
|------|------|
| `web/src/pages/BackendsPage.vue` | 推理后端列表（只读，含版本列表） |
| `web/src/pages/RuntimeTemplatesPage.vue` | 运行模板列表（只读，"从模板创建 Runtime"） |
| `web/src/pages/BackendRuntimesPage.vue` | 运行配置 CRUD（image, vendor, devices, docker） |
| `web/src/pages/NodeOverridesPage.vue` | 节点覆盖管理 |
| `web/src/pages/ModelArtifactsPage.vue` | 模型工件 CRUD |
| `web/src/pages/ModelDeploymentsPage.vue` | 模型部署 + Preview RunPlan + Start/Stop |
| `web/src/pages/ModelInstancesPage.vue` | 模型实例 + RunPlan + 日志 |
| `web/tests/apiSaveRoundtrip.test.mjs` | Web 保存 roundtrip 测试 |
| `test/e2e/model-runtime-api-roundtrip.sh` | E2E API 保存 roundtrip 测试 |

---

## 4. 分 Phase 实施计划

### Phase 0.1：文档修订（当前 Phase）

修订 Phase 0 设计文档以纠正对象边界。

### Phase 1：删除旧代码 + 建新表

**步骤**：
1. 删除旧 Server handler 文件（9 files）
2. 从 `router.go` 中删除旧路由
3. 删除旧 Web 页面和 API 文件（10 files）
4. 从 Web router 删除旧路由
5. 从 i18n 删除旧键
6. 删除旧配置目录
7. 新增 DB migration V10：删除旧表 + 创建新表（9 tables）
8. 新增 Go structs 文件
9. 加载配置文件注入 DB：只 seed inference_backends 和 backend_versions；backend-runtime-templates 只由文件读取，不落库

**验证**：`go build ./cmd/server/ && go test ./...`

### Phase 2：RunPlan Resolver

**步骤**：
1. 创建 `internal/server/runplan/` 包
2. 实现 types（ResolvedRunPlan + 子类型）
3. 实现 resolver（输入 Backend + BackendVersion + BackendRuntime + NodeRuntimeOverride + Artifact + Deployment）
4. 实现模板替换（仅 `{{var}}` 语法，未知变量 error）
5. 实现 image 解析优先级
6. 实现 env 合并（Runtime → NodeOverride → Deployment overrides）
7. 实现 docker spec 合并
8. 实现 health check 生成
9. 实现 docker_preview 生成
10. 实现 input_hash / plan_hash 计算
11. 实现 DryRunInput → DryRunResult 校验

**验证**：`go test ./internal/server/runplan/... -v -cover`

### Phase 3：Docker Executor

保留现有 Agent Docker runtime 代码。适配 `ResolvedRunPlan → ContainerCreateOptions` 映射。

### Phase 4：API

实现 Backend（只读）、BackendVersion（只读）、BackendRuntimeTemplate、BackendRuntime（CRUD + from-template）、NodeRuntimeOverride、ModelArtifact、ModelDeployment、ModelInstance、RunPlan preview API。

### Phase 5：Web

实现推理后端、运行模板、运行配置、节点覆盖、模型工件、模型部署、模型实例页面。

### Phase 6：E2E 验收

---

## 5. 每个 Phase 验证命令

| Phase | 验证命令 |
|-------|----------|
| 1 | `go build ./cmd/server/ && go build ./cmd/agent/ && go test ./... && cd web && npm run build` |
| 2 | `go test ./internal/server/runplan/... -v -cover` |
| 3 | `go test ./internal/agent/runtime/... -v` |
| 4 | `go build ./cmd/server/ && curl` 测试每个端点 |
| 5 | `cd web && npm run build` + `node web/tests/apiSaveRoundtrip.test.mjs` |
| 6 | `bash test/e2e/model-runtime-test.sh` + `bash test/e2e/model-runtime-api-roundtrip.sh` |

---

## 6. 风险点

| Phase | 风险 | 缓解 |
|-------|------|------|
| 1 | 编译错误（旧 import 未清理） | 逐文件删除 + 每次 go build |
| 2 | 模板变量 `{{var}}` 语法解析 | 全面测试 + 未知变量 error |
| 3 | Docker 参数映射遗漏 | 对照 Docker CLI 手动验证 |
| 4 | 新 API 权限 model 不匹配 | 提前确认 RBAC permission code |
| 5 | 页面表单复杂度 | 复用现有组件 |

---

---

## 6.5. 权限与租户隔离

### 权限点
新增权限点：backend:read/write, backend_runtime:read/write, node_runtime_override:read/write, model_artifact:read/write, model_deployment:read/write/start/stop, model_instance:read/logs, run_plan:read/preview, gpu_lease:read/write, agent_task:read/write。

### 角色映射
viewer: 只读权限。operator: viewer + 写/start/stop/logs。admin: operator + delete + lease/task write。platform_admin: 全部。

### 租户规则
- BackendRuntime / ModelArtifact / ModelDeployment / NodeRuntimeOverride 为租户级对象。
- InferenceBackend / BackendVersion / BackendRuntimeTemplate 为全局只读对象。
- ModelInstance / ResolvedRunPlan / GpuLease / AgentTask 携带 tenant_id。
- Start/Stop 校验 artifact/runtime/node/GPU 均属于本 tenant。

### 敏感脱敏
env 中含 token/key/secret/password 等 key 时 API 返回 `****`。

---

## 7. 禁止的操作

1. 修改 `cmd/agent/main.go` Agent 注册/心跳
2. 修改 `cmd/server/main.go` Server 启动
3. 修改 `internal/agent/collector/` GPU 采集
4. 修改 `internal/server/auth/` 认证
5. 修改 `internal/server/rbac/` 权限
6. 跨 Phase 提前实现
7. 引入新的第三方依赖（除非必要）
8. 实现 `${VAR}` 语法
9. 实现 Custom Backend
10. 实现多节点分布式并行
