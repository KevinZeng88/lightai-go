> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# Phase 1：数据模型与 Dry Run

> 依赖：无
> 周期：2-3 周
> 前置设计：[12-model-runtime-serving-design.md](../design/12-model-runtime-serving-design.md)

## 1. 目标

可以在 API 层完整走通"登记→绑定→Dry Run→校验"流程。不实现 Agent 启动、Web 页面、Gateway、自动调度。

## 2. 范围

- DB migration V3（7 张新表）
- Go struct + JSON tag
- CRUD API（不含 start/stop）
- Dry Run API
- ResolvedRunSpec 生成器（复用现有 DockerRunSpec）
- 权限校验
- 敏感字段脱敏
- 基本单元测试

## 3. 明确不做什么

- Agent DockerRuntimeDriver
- StartModelInstance / StopModelInstance
- 日志 API
- Web 页面
- Gateway
- API Key
- ModelRoute 建表
- ModelUsageRecord 建表
- 独立 ModelEvent 表（复用 audit_logs）
- ProcessRuntimeDriver / RemoteRuntimeDriver 实现
- 自动调度
- binpack / spread
- 多副本
- 抢占 / 故障迁移

## 4. 数据模型

### 4.1 新建表

| 表 | 主键 | 说明 |
|----|------|------|
| `model_artifacts` | uuid | 模型登记 |
| `runtime_environments` | uuid | 运行环境定义 |
| `runtime_environment_docker_specs` | uuid (FK→runtime_environments) | Docker 基础设施配置，1:1 |
| `run_templates` | uuid | 启动模板 |
| `model_deployments` | uuid | 部署期望状态 |
| `model_instances` | uuid | 实例实际状态 |
| `gpu_leases` | uuid | GPU 占用锁 |

### 4.2 JSON 存储字段

以下字段使用 JSON 存储在对应的表列中：

- `runtime_environment_docker_specs.devices`、`security_options`、`ulimits`、`group_add`
- `run_templates.env_mappings`、`args_template`、`volume_mappings`、`port_mappings`、`backend_flags`
- `model_deployments.gpu_ids`、`env_overrides`、`arg_overrides`、`extra_args`
- `model_instances.gpu_lease_ids`、`resolved_run_spec`

### 4.3 复用 audit_logs

Phase 1 不新建 ModelEvent 表。关键状态变更写入现有 `audit_logs`：

```text
entity_type = model_deployment | model_instance | gpu_lease
action      = created | updated | deleted | dry_run | status_changed
```

ModelEvent 作为逻辑概念保留，后续如需要 event-specific 字段再独立建表。

## 5. API

### 5.1 ModelArtifact

```text
GET    /api/v1/model-artifacts
POST   /api/v1/model-artifacts
GET    /api/v1/model-artifacts/{id}
PATCH  /api/v1/model-artifacts/{id}
DELETE /api/v1/model-artifacts/{id}
```

### 5.2 RuntimeEnvironment

```text
GET    /api/v1/runtime-environments
POST   /api/v1/runtime-environments
GET    /api/v1/runtime-environments/{id}
PATCH  /api/v1/runtime-environments/{id}
DELETE /api/v1/runtime-environments/{id}
```

### 5.3 RunTemplate

```text
GET    /api/v1/run-templates
POST   /api/v1/run-templates
GET    /api/v1/run-templates/{id}
PATCH  /api/v1/run-templates/{id}
DELETE /api/v1/run-templates/{id}
POST   /api/v1/run-templates/{id}/render-preview
```

`render-preview`：给定 ModelArtifact + node + GPU + override 变量，返回 ResolvedRunSpec 和等价命令预览。

### 5.4 ModelDeployment

```text
GET    /api/v1/model-deployments
POST   /api/v1/model-deployments
GET    /api/v1/model-deployments/{id}
PATCH  /api/v1/model-deployments/{id}
DELETE /api/v1/model-deployments/{id}
POST   /api/v1/model-deployments/{id}/dry-run
```

`POST /api/v1/model-deployments` 创建部署时不自动启动。`desired_state` 创建时默认为 `stopped`。

### 5.5 ModelInstance（只读）

```text
GET    /api/v1/model-instances
GET    /api/v1/model-instances/{id}
```

Phase 1 不提供手动创建实例的 API。实例由系统在 start 时自动创建（Phase 2）。

### 5.6 GpuLease（只读）

```text
GET    /api/v1/gpu-leases
GET    /api/v1/gpu-leases/{id}
```

GpuLease 由系统在 start/stop 时自动管理，不提供手动 CRUD。

### 5.7 Dry Run

```text
POST /api/v1/model-deployments/{id}/dry-run
```

请求：

```json
{
  "node_id": "uuid",
  "gpu_ids": ["uuid"],
  "host_port": 8001,
  "served_model_name": "qwen3-32b",
  "max_model_len": 40960,
  "gpu_memory_utilization": 0.9,
  "env_overrides": {},
  "arg_overrides": {}
}
```

响应：

```json
{
  "valid": true,
  "errors": [],
  "warnings": [],
  "resolved_run_spec": {},
  "equivalent_command_preview": "docker run -d ..."
}
```

Server 校验规则（§6.2 of design doc 完整列表 + vendor 匹配校验）。

## 6. 代码承接

| 模块 | 位置 |
|------|------|
| DB migration V3 | `internal/server/db/db.go` |
| Go struct 定义 | `internal/server/models/` |
| CRUD handlers | `internal/server/api/model_handler.go`、`runtime_handler.go`、`template_handler.go`、`deployment_handler.go`、`instance_handler.go`、`lease_handler.go` |
| Resolver | `internal/server/resolver/`（独立包，复用 DockerRunSpec 结构） |
| Router 注册 | `internal/server/api/router.go` 新增路由组 |
| 权限 codes | `internal/server/rbac/` 新增 `model:read/write`、`runtime:read/write`、`deployment:read/write` |

## 7. 权限

新增 permission codes：

```text
model:read        — 查看模型
model:write       — 创建/编辑/删除模型
runtime:read      — 查看运行环境
runtime:write     — 创建/编辑/删除运行环境（admin+；viewer 只读）
deployment:read   — 查看部署和实例
deployment:write  — 创建/编辑/删除部署、Dry Run
```

GpuLease 只读，复用 `gpu:read` 或 `node:read`。

## 8. 测试要求

- ModelArtifact / RuntimeEnvironment / RunTemplate / ModelDeployment CRUD round-trip
- RuntimeEnvironmentDockerSpec enabled 开关：禁用参数不进入 ResolvedRunSpec
- Dry Run：正常返回 + 错误枚举覆盖（节点不存在、GPU 不健康、端口冲突、vendor 不匹配、模型路径为空、必填变量缺失）
- 权限拒绝测试（viewer 不能写 model，operator 不能写全局 runtime）
- 敏感字段脱敏测试（env key 含 TOKEN/PASSWORD 时 API 返回 `<redacted>`）
- 租户隔离测试（租户 A 看不到租户 B 的模型/部署）

## 9. 验收标准

```bash
# 1. 登记模型
curl -X POST /api/v1/model-artifacts -H 'Cookie: ...' \
  -d '{"name":"qwen3-32b","source_type":"local_path","path":"/data/models/Qwen3-32B","format":"hf","task_type":"chat","architecture":"qwen","size_label":"32B"}'
# → 201, 返回 model artifact JSON

# 2. 登记运行环境
curl -X POST /api/v1/runtime-environments -H 'Cookie: ...' \
  -d '{"name":"nvidia-vllm","runtime_type":"docker","backend_type":"vllm","vendor":"nvidia","docker":{"image":"vllm/vllm-openai:latest","ipc_mode":{"enabled":true,"value":"host"}}}'
# → 201

# 3. 登记启动模板
curl -X POST /api/v1/run-templates -H 'Cookie: ...' \
  -d '{"name":"vllm-standard","runtime_type":"docker","vendor":"nvidia","required_variables":["MODEL_PATH","GPU_IDS"],"args_template":["--model","${MODEL_PATH}"]}'
# → 201

# 4. 创建部署
curl -X POST /api/v1/model-deployments -H 'Cookie: ...' \
  -d '{"model_artifact_id":"...","runtime_environment_id":"...","run_template_id":"...","node_id":"...","gpu_ids":["..."],"host_port":8001}'
# → 201, desired_state=stopped

# 5. Dry Run 校验错误
curl -X POST /api/v1/model-deployments/{id}/dry-run -H 'Cookie: ...' \
  -d '{"node_id":"nonexistent","gpu_ids":[],"host_port":8001}'
# → {"valid":false,"errors":["指定的节点不存在"],"warnings":[]}

# 6. Dry Run 通过
curl -X POST /api/v1/model-deployments/{id}/dry-run -H 'Cookie: ...' \
  -d '{"node_id":"valid-uuid","gpu_ids":["valid-gpu-uuid"],"host_port":8001}'
# → {"valid":true,"resolved_run_spec":{...},"equivalent_command_preview":"docker run -d ..."}
```

## 10. 风险点

- Dry Run 校验规则的完整性直接影响后续 Agent 启动成功率。遗漏的校验会在 Phase 2 暴露
- `gpu_visible_env_key` 在 RuntimeEnvironment/RunTemplate/Deployment 三层之间的覆盖逻辑需要充分测试
- RunTemplate 的 `${VAR}` 替换逻辑如果过于灵活，可能导致注入风险。Phase 1 只支持已知变量名

## 11. 与 Phase 2 的接口约定

Phase 1 必须向 Phase 2 提供：

1. 完整的 Go struct 定义（尤其是 `ResolvedRunSpec`、`DockerRunSpec`）
2. `Resolver.Resolve()` 函数签名稳定
3. `GpuLease` 表结构和 `CreateLease` / `ReleaseLease` 函数
4. `ModelInstance` 表结构和 `UpdateActualState` 函数
5. Agent Task payload 结构定义（即使 Phase 1 不下发）

Phase 2 不需要关心：
- ModelArtifact / RuntimeEnvironment / RunTemplate 的 CRUD 细节
- Dry Run 实现细节
- 权限校验细节（Agent 侧只用 agent token）
