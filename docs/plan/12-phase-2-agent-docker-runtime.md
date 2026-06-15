# Phase 2：Agent Docker Runtime

> 依赖：Phase 1（数据模型 + Dry Run + ResolvedRunSpec 生成器）
> 周期：2-3 周

## 1. 目标

可以真正启动和停止 Docker 模型实例。Agent 通过 RuntimeDriver 接口执行 docker run/stop/logs，Server 管理 GpuLease 完整生命周期。

## 2. 范围

- `RuntimeDriver` 接口定义
- `DockerRuntimeDriver` 完整实现
- Agent Task 类型扩展（Start / Stop / Logs / Inspect）
- GpuLease 状态流转
- ModelInstance actual_state 更新
- 后台任务：GpuLease 过期回收
- 单元测试 + mock Docker daemon 集成测试

## 3. 明确不做什么

- ProcessRuntimeDriver 实现（只保留 stub）
- RemoteRuntimeDriver 实现（只保留 stub）
- Web 页面
- Gateway
- 自动重启
- 自动调度

## 4. 数据模型（Phase 1 已建表，Phase 2 启用状态流转）

### 4.1 GpuLease 生命周期流转

```text
reserved → active  (Agent 启动成功)
reserved → failed  (Agent 启动失败) → released
active   → released (正常停止)
reserved → expired (超时未启动) → released
```

后台任务：每 30 秒扫描 `status=reserved AND expires_at < now()`，标记为 expired。

### 4.2 ModelInstance actual_state 流转

```text
pending → starting → loading → running
                   → failed
running → unhealthy
running → stopping → stopped
任何状态 → unknown (Agent 离线)
```

### 4.3 新增 Agent Task 类型

扩展现有 `AgentTask` 模型（`08-engineering-contracts.md` §9），新增：

```text
StartModelInstance
StopModelInstance
GetModelInstanceLogs
InspectModelInstance
```

## 5. RuntimeDriver 接口

位置：`internal/agent/runtime/driver.go`

```go
type RuntimeDriver interface {
    Validate(ctx context.Context, spec ResolvedRunSpec) error
    Start(ctx context.Context, spec ResolvedRunSpec) (*StartResult, error)
    Stop(ctx context.Context, instance RuntimeInstance) error
    Restart(ctx context.Context, instance RuntimeInstance) error
    Status(ctx context.Context, instance RuntimeInstance) (*RuntimeStatus, error)
    Logs(ctx context.Context, instance RuntimeInstance, opts LogOptions) (*LogResult, error)
}
```

### 5.1 DockerRuntimeDriver

位置：`internal/agent/runtime/docker.go`

#### Validate

检查：docker 命令存在、daemon 可访问、image 存在（image_pull_policy=never 时）、model path 存在、volume host path 存在、device host path 存在、host_port 可用、GPU_IDS 非空

#### Start

使用参数数组组装 `docker run -d ...`，不得 shell 拼接。

支持的 Docker 参数：`--name`、`-e`、`-v`、`--device`、`-p`、`--group-add`、`--privileged`、`--ipc`、`--uts`、`--network`、`--shm-size`、`--ulimit`、`--security-opt`、`--gpus`、command、args

ownership labels：`lightai.managed=true`、`lightai.instance_id=<id>`、`lightai.deployment_id=<id>`、`lightai.tenant_id=<id>`（与 contracts §11 对齐）

#### Stop

`docker stop <container>`（默认不 rm）。Delete instance 时才 `docker rm -f`。

#### Status

`docker inspect`，映射 Docker 状态到 ModelInstance actual_state：

| Docker 状态 | actual_state |
|---|---|
| created | starting |
| running | running |
| exited (0) | stopped |
| exited (非0) | failed |
| dead | failed |
| restarting | starting |
| not found | unknown |

#### Logs

`docker logs --tail N`。Phase 2 不要求 follow。

## 6. API（新增）

```text
POST /api/model-deployments/{id}/start
POST /api/model-deployments/{id}/stop
GET  /api/model-instances/{id}/logs?tail=100
```

`start` 流程：
1. Server 校验（复用 Dry Run 校验 + 额外检查 GpuLease 冲突）
2. 创建 ModelInstance（actual_state=pending）
3. 创建 GpuLease（status=reserved）
4. 生成 ResolvedRunSpec
5. 下发 Agent Task（StartModelInstance）
6. Agent 返回结果 → Server 更新状态

`stop` 流程：
1. 下发 Agent Task（StopModelInstance）
2. Agent 执行 docker stop
3. Server 更新 actual_state=stopped、lease=released

## 7. 代码承接

| 模块 | 位置 |
|------|------|
| RuntimeDriver 接口 | `internal/agent/runtime/driver.go` |
| DockerRuntimeDriver | `internal/agent/runtime/docker.go` |
| ProcessRuntimeDriver stub | `internal/agent/runtime/process.go` |
| RemoteRuntimeDriver stub | `internal/agent/runtime/remote.go` |
| Agent Task 类型扩展 | `internal/agent/task/` |
| Server start/stop/logs handlers | `internal/server/api/deployment_handler.go`、`instance_handler.go` |
| GpuLease 后台回收 | `internal/server/scheduler/lease_reaper.go` |

## 8. 测试要求

- DockerRuntimeDriver.Validate：各项校验的错误分支覆盖
- DockerRuntimeDriver.Start：mock docker 命令，验证参数数组正确（不出现 shell 拼接）
- DockerRuntimeDriver.Stop：验证 docker stop 调用 + 不自动 rm
- DockerRuntimeDriver.Status：Docker 状态 → actual_state 映射矩阵
- GpuLease 流转：reserved→active、reserved→failed→released、expired→released
- 启动失败不遗留 active lease
- Agent 离线 → instance=unknown, lease 不释放
- 权限：start/stop 需要 deployment:write

## 9. 验收标准

```bash
# 启动
curl -X POST /api/model-deployments/{id}/start -H 'Cookie: ...'
# → 200, {"instance_id":"...","actual_state":"starting"}

# 等待几秒后检查状态
curl /api/model-instances/{id} -H 'Cookie: ...'
# → {"actual_state":"running","container_id":"abc123","endpoint_url":"http://node-ip:8001"}

# 日志
curl /api/model-instances/{id}/logs?tail=100 -H 'Cookie: ...'
# → vLLM 启动日志

# 停止
curl -X POST /api/model-deployments/{id}/stop -H 'Cookie: ...'
# → 200, {"actual_state":"stopped"}

# 验证 GpuLease 已释放
curl /api/gpu-leases?instance_id={id} -H 'Cookie: ...'
# → [{"status":"released"}]

# 启动失败
curl -X POST /api/model-deployments/{id}/start -H 'Cookie: ...'  # 镜像不存在
# → {"actual_state":"failed","last_error":"docker image not found: ..."}

# 验证 active lease 未泄露
curl /api/gpu-leases?status=active -H 'Cookie: ...'
# → 不包含该实例的 lease
```

## 10. 风险点

- Docker daemon 版本兼容性（不同版本支持的参数不同）。Validate 阶段应探测 daemon 版本
- 国产 GPU 厂商的 Docker runtime（如 nvidia-container-toolkit 替代品）可能有不同的 `--gpus` 语法
- 端口冲突：Agent 侧 `docker run` 时端口已被占用（竞态条件）。Server 侧 GpuLease 不能防止端口冲突
- 容器启动后健康检查失败（模型加载失败但容器仍在运行）。需要区分 `loading` 和 `running` 状态

## 11. 与 Phase 3 的接口约定

Phase 2 必须向 Phase 3 提供：

1. Start/Stop/Logs API 稳定
2. ModelInstance 状态实时更新（Web 轮询或后续 WebSocket）
3. GPU 占用关系可通过 API 查询（`GET /api/gpus/{id}/leases` 或等价端点）
4. 启动失败时 `last_error` 可读且准确
