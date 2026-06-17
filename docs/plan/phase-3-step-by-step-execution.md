# LightAI Go Phase 3 Step-by-Step Execution Manual

> Claude 执行操作手册 — Phase 0.1 修订版。
> 日期: 2026-06-16
> 版本: v2.0

**重要：此文件是 Claude 在执行 Phase 1-6 时的强制操作手册。每一步都必须严格遵循。不允许跳过任何步骤或验证。**

> **自动连续执行模式**：
> 如果用户明确要求 Phase 2-6 一次性自动完成，则必须同时遵循
> `docs/plan/phase-3-auto-execution-guard.md`。
> 在自动连续执行模式下，各 Phase 的"停止点"改为"质量门禁"：
> 验证失败必须修复，验证通过后自动进入下一 Phase；
> 只有 auto-execution guard 中定义的重大问题才允许停止。

---

## Phase 0（含 0.1）：文档化

### 已完成的步骤

- [x] 阅读 GPUStack 参考代码
- [x] 阅读 LightAI Go 当前实现
- [x] 编写 GPUStack 参考调研文档
- [x] 编写 Backend/Runtime/RunPlan 设计文档
- [x] 编写 Phase 3 重构计划
- [x] 编写测试计划
- [x] 编写本执行手册
- [x] Phase 0.1 修订：纠正对象边界

### 🛑 停止点：等待人工审核 Phase 0.1 修订

---

## Phase 1: 删除旧代码 + 建新表

### Step 1.1: 删除旧 Server handler 文件

```bash
rm internal/server/api/model_handlers.go
rm internal/server/api/deployment_lifecycle.go
rm internal/server/api/instance_state.go
rm internal/server/api/lease.go
rm internal/server/api/task_handlers.go
rm internal/server/api/task_constants.go
rm internal/server/api/sweep.go
rm internal/server/api/resolve_helper.go
rm internal/server/resolver/resolver.go
```

**禁止删除**：agent_handlers.go, resource_handlers.go, router.go, db.go, auth/, rbac/

### Step 1.2: 从 models.go 删除旧 struct

删除：`ModelArtifact`, `RuntimeEnvironment`, `RuntimeEnvironmentDockerSpec`, `RunTemplate`, `ModelDeployment`, `ModelInstance`, `GpuLease`, `AgentTask`, `EnabledValue[T]`

保留：`Node`, `Tenant`, `User`, `Role`, `Permission`, `Session`, `AuditLog`

### Step 1.3: 从 router.go 删除旧路由

删除所有 Phase 1 model runtime serving 路由（model-artifacts, runtime-environments, run-templates, model-deployments, model-instances, gpu-leases, agent tasks）。

保留：Agent routes, Node routes, GPU routes, Auth routes, RBAC routes, Observability routes。

### Step 1.4-1.6: 删除旧 Web 文件、路由、i18n

```bash
# 页面
rm web/src/pages/RuntimeEnvironmentsPage.vue
rm web/src/pages/RunTemplatesPage.vue
rm web/src/pages/ModelArtifactsPage.vue
rm web/src/pages/ModelDeploymentsPage.vue
rm web/src/pages/ModelInstancesPage.vue

# API clients
rm web/src/api/runtimeEnvironments.ts
rm web/src/api/runTemplates.ts
rm web/src/api/modelArtifacts.ts
rm web/src/api/modelDeployments.ts
rm web/src/api/modelInstances.ts
```

从 `web/src/router/index.ts` 删除对应路由。

从 `web/src/locales/zh-CN.ts` 和 `en-US.ts` 删除旧 i18n 键块。

### Step 1.7: 删除旧配置

```bash
rm -rf configs/templates/runtime/
rm -rf configs/templates/run/
rm configs/templates/docker-images.json
rm docs/templates-config.md
```

### Step 1.8: 创建新配置目录和文件

```bash
mkdir -p configs/model-runtime/backends
mkdir -p configs/model-runtime/backend-versions/vllm
mkdir -p configs/model-runtime/backend-versions/sglang
mkdir -p configs/model-runtime/backend-versions/llamacpp
mkdir -p configs/model-runtime/backend-runtime-templates
```

创建配置文件：
- `configs/model-runtime/backends/vllm.yaml`
- `configs/model-runtime/backends/sglang.yaml`
- `configs/model-runtime/backends/llamacpp.yaml`
- `configs/model-runtime/backend-versions/vllm/0.8.5.yaml`
- `configs/model-runtime/backend-versions/vllm/0.10.0.yaml`
- `configs/model-runtime/backend-versions/sglang/0.4.6.yaml`
- `configs/model-runtime/backend-versions/sglang/0.5.0.yaml`
- `configs/model-runtime/backend-versions/llamacpp/b4817.yaml`
- `configs/model-runtime/backend-runtime-templates/vllm-nvidia-docker.yaml`
- `configs/model-runtime/backend-runtime-templates/vllm-metax-docker.yaml`
- `configs/model-runtime/backend-runtime-templates/sglang-nvidia-docker.yaml`
- `configs/model-runtime/backend-runtime-templates/sglang-metax-docker.yaml`
- `configs/model-runtime/backend-runtime-templates/llamacpp-nvidia-docker.yaml`

### Step 1.8.5: 权限与租户准备



1. 补齐新表 tenant_id / owner_id / created_by / updated_by 字段。

2. 补齐新增权限常量或权限种子（见设计文档 §23.2）。

3. 角色权限映射种子（viewer / operator / admin / platform_admin）。

4. 不破坏现有 auth / RBAC / tenant / session 逻辑。



### Step 1.9: 新增 DB migration V10

修改 `internal/server/db/db.go`，新增 `migrateV10()`：

1. 删除旧表：`runtime_environment_docker_specs`, `runtime_environments`, `run_templates`, `model_deployments`, `model_instances`, `gpu_leases`, `agent_tasks`
2. 创建新表：`inference_backends`, `backend_versions`, `backend_runtimes`, `node_runtime_overrides`, `model_deployments`（新结构）, `model_instances`（新结构）, `resolved_run_plans`, `gpu_leases`（重建）, `agent_tasks`（重建）
3. 保留 `model_artifacts`（可能微调 source_type → source）
4. 从配置文件 seed inference_backends 和 backend_versions 数据；backend-runtime-templates 只由文件读取，不落库

### Step 1.10: 新增 Go structs

- `internal/server/models/backend.go` — `InferenceBackend`（含 default_version, common_parameters, default_env）, `BackendVersion`（含 is_default, default_backend_params, env_json）
- `internal/server/models/runtime.go` — `BackendRuntimeTemplate`, `BackendRuntime`, `NodeRuntimeOverride`
- `internal/server/models/artifact.go` — `ModelArtifact`
- `internal/server/models/deployment.go` — `ModelDeployment`
- `internal/server/models/instance.go` — `ModelInstance`, `GpuLease`, `AgentTask`
- `internal/server/models/runplan.go` — `ResolvedRunPlan`
- `internal/server/runplan/types.go` — RunPlan 子类型

### Step 1.11: Phase 1 验证

```bash
go build ./cmd/server/ && go build ./cmd/agent/
go test ./...
cd web && npm run build

# 确认无旧代码残留
grep -r "InferenceBackendVersion\|runtime_environment_docker_specs\|run_templates\|RuntimeEnvironment\b" internal/ --include="*.go" | grep -v "_test\|\.git"
```

### 🛑 Phase 1 停止点

---

## Phase 2: RunPlan Resolver

### Step 2.1: 创建包和类型

新增 `internal/server/runplan/` 包：

- `types.go` — `ResolvedRunPlan`, `DockerSpec`, `DockerVolume`, `DockerDevice`, `DockerPort`, `HealthCheck`, `ResolvedMount`
- `resolver.go` — `Resolve(input ResolveInput) (*ResolvedRunPlan, []error, []string)`
- `template.go` — `substituteVars`（仅 `{{var}}` 语法）, `buildVarMap`
- `dryrun.go` — `ValidateDryRun(db, input DryRunInput) DryRunResult`
- `preview.go` — `EquivalentCommandPreview(plan *ResolvedRunPlan) string`

### Step 2.2: Resolver 流程

```
1. 校验 runtime_type == "docker"
2. 查找 BackendVersion
3. 解析 image: NodeRuntimeOverride > BackendRuntime > BackendVersion.defaultImages[vendor]
4. 解析 entrypoint: BackendRuntime.override > BackendVersion.default
5. 解析 args: BackendVersion.default_args + BackendVersion.default_backend_params + BackendRuntime.args_override（append only） + Deployment.parameters
6. 仅使用 {{var}} 替换。未知变量 → error
7. 合并 env: Backend.default_env_json + BackendVersion.env_json + BackendRuntime.default_env_json + NodeRuntimeOverride.env_json + Deployment.env_overrides_json + GPU visible env（后者覆盖前者）
8. 合并 docker spec: BackendRuntime.docker_json + NodeRuntimeOverride.docker_override_json
9. 生成模型挂载
10. 生成 health check
11. 生成 docker_preview
12. 计算 input_hash / plan_hash
```

### Step 2.3: DryRun 校验

检查项：
- Node 存在且在线
- GPU 存在、健康、可用、无租约冲突
- BackendRuntime.vendor 与 GPU vendor 匹配
- Host port 冲突
- 模型路径非空

### Step 2.4: Phase 2 验证

```bash
go test ./internal/server/runplan/... -v -cover
```

### 🛑 Phase 2 停止点

---

## Phase 3: Docker Executor

### Step 3.1: 保留现有 Agent Runtime 代码

不删除 `internal/agent/runtime/` 下的任何文件。

### Step 3.2: 适配 ResolvedRunPlan

在 `internal/agent/runtime/docker.go` 中添加适配函数：

```go
func planToContainerCreateOptions(plan runplan.ResolvedRunPlan) ContainerCreateOptions
```

### Step 3.3: 验证

```bash
go test ./internal/agent/runtime/... -v
```

### 🛑 Phase 3 停止点

---

## Phase 4: API

### Step 4.1-4.8: 依次实现 Handler

**权限接入要求**：

- 所有新 API 接入 auth middleware（Session + CSRF）。

- 所有新 API 接入 RBAC 权限校验（见 API 权限矩阵）。

- 所有 list/detail 接口实现 tenant 过滤。

- API roundtrip 脚本增加 401/403/跨租户拒绝测试。

- RunPlan / env / docker_preview 返回时执行脱敏。


1. `backend_handlers.go` — Backend/BackendVersion 只读 + RuntimeTemplate 只读
2. `runtime_handlers.go` — BackendRuntime CRUD + POST from-template + NodeRuntimeOverride CRUD
3. `artifact_handlers.go` — ModelArtifact CRUD
4. `deployment_handlers.go` — ModelDeployment CRUD + start/stop（引用 backend_runtime_id）
5. `instance_handlers.go` — ModelInstance 只读 + logs
6. `runplan_handlers.go` — POST preview + GET by id + GET instance run-plans
7. `lease_handlers.go` — GPU lease 管理
8. `task_handlers.go` — AgentTask 管理

### Step 4.9: 实现 Start 事务顺序

必须严格遵循设计文档 §12.5 的 7 步事务顺序：
1. INSERT model_instance（current_run_plan_id = NULL）
2. 调用 RunPlan Resolver（带入 instance_id）
3. INSERT resolved_run_plans
4. UPDATE model_instance.current_run_plan_id
5. INSERT gpu_leases
6. INSERT agent_task
7. COMMIT

如果任一步失败，事务回滚。

### Step 4.10: 更新 router.go

注册所有新路由（不使用 backend_id + backend_version_id）。

### Step 4.10: 验证

```bash
go build ./cmd/server/
curl -s http://localhost:18080/api/v1/inference-backends | jq '.items | length'  # 应为 3
curl -s http://localhost:18080/api/v1/backend-runtime-templates | jq '.items | length'  # 应为 5
```

### 🛑 Phase 4 停止点

---

## Phase 5: Web

### Step 5.1-5.7: 依次实现页面

**权限与脱敏要求**：

- Web 菜单按权限显示。

- 无权限按钮隐藏或 disabled（但后端仍必须强制校验）。

- RunPlan 详情和 docker_preview 默认脱敏显示。

- 复制 docker_preview 使用脱敏版本。


1. `BackendsPage.vue` — 只读列表（vLLM/SGLang/llamacpp）+ 版本详情
2. `RuntimeTemplatesPage.vue` — 只读模板列表 + "从模板创建 Runtime"按钮
3. `BackendRuntimesPage.vue` — CRUD（name, vendor, image, backend version, devices, docker flags）
4. `NodeOverridesPage.vue` — 按 node + runtime 配置覆盖
5. `ModelArtifactsPage.vue` — CRUD
6. `ModelDeploymentsPage.vue` — 创建部署（选择 artifact + runtime → placement → parameters → Preview RunPlan → Create → Start）
7. `ModelInstancesPage.vue` — 列表 + 详情 + RunPlan + 日志

### Step 5.8-5.11: 基础设施

- API client 文件（backends.ts, artifacts.ts, deployments.ts, instances.ts, runtimes.ts, runplans.ts）
- Router 更新
- i18n 新增键（backends, runtimeTemplates, backendRuntimes, nodeOverrides, artifacts, deployments, instances, runPlans）
- 导航侧边栏更新

### Step 5.12: 保存 roundtrip 验证

```bash
# 运行前端保存 roundtrip 测试
node web/tests/apiSaveRoundtrip.test.mjs
```

所有 CRUD + PATCH + 刷新验证必须通过。

### Step 5.13: 编译验证

```bash
cd web && npm run build
```

### 🛑 Phase 5 停止点

---

## Phase 6: E2E 验收

**权限 E2E 要求**：
- 至少覆盖 viewer / operator / admin 三类用户。
- 验证：未登录 401、viewer 写操作 403、operator 本租户成功、operator 跨租户 403、platform_admin 跨租户允许。
- 验证 RunPlan 敏感 env 脱敏。

### Step 6.1: E2E 测试脚本

**权限 E2E 要求**：

- 至少覆盖 viewer / operator / admin 三类用户。

- 验证：未登录 401、viewer 写操作 403、operator 本租户成功、operator 跨租户 403、platform_admin 跨租户允许。

- 验证 RunPlan 敏感 env 脱敏。


新增 `test/e2e/model-runtime-test.sh`：

1. 启动 server + agent
2. 验证 Backend 列表（3 个）
3. 验证 BackendVersion 列表
4. 验证 RuntimeTemplate 列表（5 个）
5. 从模板创建 BackendRuntime
6. 创建 ModelArtifact
7. 创建 ModelDeployment（引用 backend_runtime_id）
8. POST /api/v1/run-plans/preview
9. Start deployment → 验证 instance + RunPlan
10. 查看 RunPlan（GET /api/v1/run-plans/{id}）
11. 验证 Docker 容器运行
12. Health check
13. 查看日志
14. Stop → 验证容器停止
15. Restart → 验证新 RunPlan 生成
16. 查看历史 RunPlans
17. 清理

### Step 6.2: 保存 roundtrip + 刷新验证

```bash
# E2E API 保存 roundtrip
bash test/e2e/model-runtime-api-roundtrip.sh
```

验证所有实体（BackendRuntime, NodeRuntimeOverride, ModelArtifact, ModelDeployment）保存后刷新不丢失。

### Step 6.3: E2E 测试验证

```bash
bash test/e2e/model-runtime-test.sh
```

### 🛑 Phase 6 停止点 → 总体验收

---

## 完成报告格式

每个 Phase 完成后输出：

```markdown
# Phase N 完成报告

## 修改的文件
## 新增的文件
## 删除的文件
## 验证命令及结果
## 已知问题
## 需要人工确认的问题
## 下一步
```

---

## 紧急停止规则

1. `go build` 失败超过 30 分钟
2. 需要修改 auth/RBAC/node 管理代码
3. 数据库 migration 失败无法回滚
4. 设计文档未覆盖的场景需要重新设计

## 绝对禁止

1. 实现 `${VAR}` 语法
2. 实现 Custom Backend / MindIE / VoxBox
3. 实现多节点分布式并行
4. Deployment 使用 backend_id + backend_version_id（必须使用 backend_runtime_id）
5. 修改 auth/RBAC/collector
6. 跨 Phase 提前实现
7. 不经验证进入下一 Phase
8. 实现 replicas > 1
