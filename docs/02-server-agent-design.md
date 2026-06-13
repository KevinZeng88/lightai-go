# LightAI Go Server / Agent 通信设计

## 1. 设计目标

LightAI Go 第一阶段采用 Server / Agent 架构。

Server 是控制面，负责集中管理节点、资源、运行环境、模型、模型实例、任务、Web/API 和监控组件配置。

Agent 是执行面，运行在每台 GPU 服务器上，负责本机操作系统资源采集、GPU 采集、Docker 操作、模型实例启停、实例状态检查、日志回报和 `/metrics` 指标暴露。

第一阶段通信目标：

1. Agent 可以注册到 Server；
2. Server 可以保存和识别 Agent 节点；
3. Agent 可以周期性发送心跳；
4. Server 可以判断节点在线 / 离线；
5. Agent 可以周期性上报本机资源；
6. Server 可以向 Agent 下发任务；
7. Agent 可以执行任务并回报结果；
8. Agent 可以暴露本机 `/metrics`；
9. Server 可以暴露自身 `/metrics`；
10. Server 可以提供 Prometheus 动态发现接口 `/metrics/targets`；
11. 通信逻辑简单、稳定、现场易排障。

第一阶段不使用复杂消息队列、gRPC、WebSocket 或服务发现系统。优先使用 HTTP Pull 模式。

---

## 2. 总体通信模式

第一阶段采用 Agent 主动访问 Server 的 Pull 模式。

通信方向：

```text
Agent → Server
```

Agent 主动完成：

```text
注册
心跳
资源上报
任务拉取
任务结果回报
实例状态回报
```

Server 不主动连接 Agent。

Prometheus 监控链路是旁路能力：

```text
Prometheus → Server /metrics
Prometheus → Server /metrics/targets
Prometheus → Agent /metrics
Grafana → Prometheus
LightAI Web → Grafana
```

Prometheus 只用于时序指标采集、趋势展示和告警，不作为业务状态同步机制。

Server 仍然以 Agent 上报到数据库中的状态作为节点管理、资源管理和实例管理依据。

---

## 3. 为什么采用 HTTP Pull 模式

HTTP Pull 模式适合中小客户现场环境，原因包括：

1. GPU 服务器可能位于内网、防火墙或 NAT 后面；
2. Server 不需要主动访问每台 Agent；
3. Agent 重启后可以自动恢复；
4. 网络波动时更容易重试；
5. HTTP 接口便于 curl 调试；
6. 便于后续离线部署和现场排障。

---

## 4. Agent 启动流程

Agent 启动后按以下顺序执行：

```text
读取配置
  ↓
初始化日志
  ↓
加载或生成 Agent ID
  ↓
启动本地 HTTP 服务
  ↓
暴露 /healthz 和 /metrics
  ↓
采集本机基础信息
  ↓
注册到 Server
  ↓
启动心跳循环
  ↓
启动资源采集循环
  ↓
启动任务拉取循环
  ↓
启动实例状态检查循环
```

Agent 启动失败分为两类：

### 4.1 致命失败

以下情况可以导致 Agent 退出：

1. 配置文件无法读取；
2. Server 地址为空；
3. Agent ID 无法生成或保存；
4. 日志目录无法创建；
5. 本地监听端口被占用；
6. 必要权限不足导致 Agent 无法运行。

### 4.2 非致命失败

以下情况不能导致 Agent 退出：

1. GPU 采集失败；
2. Docker 不可用；
3. 暂时无法连接 Server；
4. 某个 Collector 不可用；
5. 某个模型实例检查失败；
6. Prometheus 指标刷新失败；
7. Server 暂时拒绝心跳或资源上报。

非致命失败必须记录日志并持续重试。

---

## 5. Agent 身份设计

每个 Agent 必须有稳定身份。

Agent ID 优先来自配置文件：

```yaml
agent:
  id: "node-001"
  name: "gpu-server-001"
```

如果未配置 `agent.id`，Agent 首次启动时自动生成 UUID，并保存到本地状态文件：

```text
data/agent-id
```

Agent ID 一旦生成，不应频繁变化。

Agent Name 用于页面展示，可以修改。
Agent ID 用于系统识别，不建议修改。

---

## 6. Server 配置示例

文件：`configs/server.yaml`

```yaml
server:
  listen_addr: "0.0.0.0:8080"
  data_dir: "./data"
  log_dir: "./logs"
  database_path: "./data/lightai.db"

node:
  offline_timeout_seconds: 15

security:
  agent_token: "change-me"  # bootstrap/shared agent token

observability:
  metrics_enabled: true
  metrics_path: "/metrics"
  targets_path: "/metrics/targets"
  mode: "builtin"       # builtin / external / disabled

  builtin:
    prometheus_enabled: true
    prometheus_url: "http://127.0.0.1:9090"
    grafana_enabled: true
    grafana_url: "http://127.0.0.1:3000"
    grafana_public_url: "/grafana/"

  external:
    prometheus_url: ""
    grafana_url: ""

log:
  level: "info"
  file: "lightai-server.log"
```

---

## 7. Agent 配置示例

文件：`configs/agent.yaml`

```yaml
agent:
  id: ""
  name: "gpu-node-001"
  listen_addr: "0.0.0.0:18080"
  data_dir: "./data"
  log_dir: "./logs"

server:
  base_url: "http://127.0.0.1:8080"
  token: "change-me"

heartbeat:
  interval_seconds: 2

resource_report:
  interval_seconds: 3

task:
  pull_interval_seconds: 2

instance:
  check_interval_seconds: 5

observability:
  metrics_enabled: true
  metrics_path: "/metrics"

system:
  collector: "gopsutil"   # gopsutil / fastfetch / mock

gpu:
  profile: "production"   # production / development / test
  collectors:
    mock:
      enabled: false
    nvidia:
      enabled: true
      tool_path: "/usr/bin/nvidia-smi"
      timeout_seconds: 3
    metax:
      enabled: false
      tool_path: "/usr/bin/mx-smi"
      timeout_seconds: 3

docker:
  enabled: true
  socket: "unix:///var/run/docker.sock"

log:
  level: "info"
  file: "lightai-agent.log"
```

默认建议：

```text
heartbeat interval: 2s
resource report interval: 3s
task pull interval: 2s
instance check interval: 5s
node offline timeout: 15s
```

所有时间间隔必须可配置。

---

## 8. Server 健康检查

```http
GET /healthz
```

返回：

```json
{
  "status": "ok",
  "service": "lightai-server",
  "version": "0.1.0"
}
```

---

## 9. Agent 健康检查

```http
GET /healthz
```

返回：

```json
{
  "status": "ok",
  "service": "lightai-agent",
  "agent_id": "node-001",
  "version": "0.1.0"
}
```

---

## 10. Server Metrics

```http
GET /metrics
```

用于 Prometheus 抓取 Server 指标。

第一阶段至少预留接口。
后续逐步输出 Server 节点数量、任务数量、实例数量、API 请求耗时等指标。

---

## 11. Agent Metrics

```http
GET /metrics
```

用于 Prometheus 抓取 Agent、OS、GPU、Docker 和模型实例指标。

第一阶段至少预留接口。
资源采集完成后，应逐步输出操作系统资源、GPU 资源和实例健康指标。

---

## 12. Prometheus 动态发现接口

Server 提供：

```http
GET /metrics/targets
```

该接口用于 Prometheus 动态发现 Agent。

返回 Prometheus HTTP Service Discovery 格式：

```json
[
  {
    "targets": [
      "192.168.1.10:18080"
    ],
    "labels": {
      "job": "lightai-agent",
      "__scheme__": "http",
      "__metrics_path__": "/metrics",
      "node_id": "node-001",
      "node_name": "gpu-server-001",
      "vendor": "mixed"
    }
  },
  {
    "targets": [
      "192.168.1.11:18080"
    ],
    "labels": {
      "job": "lightai-agent",
      "__scheme__": "http",
      "__metrics_path__": "/metrics",
      "node_id": "node-002",
      "node_name": "gpu-server-002",
      "vendor": "nvidia"
    }
  }
]
```

Server 根据已注册节点生成 targets。

生成规则：

1. 只返回已注册且未删除的节点；
2. 节点必须 `metrics_enabled=true`；
3. `metrics_scheme`、`advertised_address`、`metrics_port`、`metrics_path` 必须通过 Server 校验；
4. 不以业务 `online/offline` 状态过滤 target；
5. Server 组合 target，Agent 不上报任意完整 URL；
6. labels 不放高基数字段。

完整字段和 HTTP SD 规则见 `docs/08-engineering-contracts.md`。

---

## 13. Agent 注册接口

```http
POST /api/agent/register
```

请求：

```json
{
  "agent_id": "node-001",
  "name": "gpu-server-001",
  "hostname": "gpu-server-001",
  "ip": "192.168.1.10",
  "advertised_address": "192.168.1.10",
  "agent_version": "0.1.0",
  "protocol_version": "1",
  "metrics_enabled": true,
  "metrics_scheme": "http",
  "metrics_port": 18080,
  "metrics_path": "/metrics",
  "os": "linux",
  "arch": "amd64",
  "cpu_model": "Intel Xeon",
  "cpu_cores": 32,
  "memory_total_bytes": 274877906944,
  "started_at": "2026-06-13T10:00:00Z"
}
```

响应：

```json
{
  "node_id": "node-001",
  "accepted": true,
  "server_time": "2026-06-13T10:00:00Z",
  "message": "registered"
}
```

注册逻辑：

1. 如果 `agent_id` 不存在，创建 Node；
2. 如果 `agent_id` 已存在，更新 Node 基础信息；
3. 校验并保存 advertised address、metrics scheme/port/path/enabled；
4. 校验 Agent version 和 protocol version 兼容性；
5. 使用 Server 接收时间更新 `last_seen_at`；
6. 设置节点状态为 `online`；
7. 新建 Node 时由 Server 写入 `tenant_id=default`、`owner_id=null`、`created_by=system`、`updated_by=system`；
8. Agent 注册请求不得指定或覆盖 tenant、owner、created_by、updated_by；
9. 返回注册结果。

---

## 14. Agent 心跳接口

```http
POST /api/agent/heartbeat
```

请求：

```json
{
  "agent_id": "node-001",
  "timestamp": "2026-06-13T10:00:02Z",
  "status": "online",
  "message": ""
}
```

响应：

```json
{
  "accepted": true,
  "server_time": "2026-06-13T10:00:02Z"
}
```

如果节点不存在：

```json
{
  "accepted": false,
  "need_register": true,
  "message": "agent not registered"
}
```

心跳逻辑：

1. Server 根据 `agent_id` 找到 Node；
2. 使用 Server 接收时间更新 `last_heartbeat_at`，请求中的 timestamp 只用于诊断时钟偏差；
3. 更新状态为 `online`；
4. 如果节点不存在，要求 Agent 重新注册；
5. Agent 收到 `need_register=true` 后重新执行注册流程。

---

## 15. 资源上报接口

```http
POST /api/agent/resources/report
```

资源上报包括：

1. 主机信息；
2. CPU 指标；
3. 内存指标；
4. Swap 指标；
5. 磁盘指标；
6. 网络基础信息；
7. GPU 设备列表；
8. GPU 实时指标；
9. Docker 状态；
10. Collector 诊断信息；

详细设计见：

```text
docs/03-resource-monitoring-design.md
```

---

## 16. 任务 Claim 接口

```http
POST /api/agent/tasks/claim
```

请求：

```json
{
  "agent_id": "node-001",
  "limit": 5,
  "lease_seconds": 30
}
```

响应：

```json
{
  "tasks": [
    {
      "task_id": "task-001",
      "type": "start_instance",
      "instance_id": "inst-001",
      "tenant_id": "default",
      "created_by": "user-001",
      "operation_id": "op-001",
      "generation": 3,
      "attempt": 1,
      "lease_expires_at": "2026-06-13T10:00:30Z",
      "payload": {
        "docker_run_spec": {}
      },
      "created_at": "2026-06-13T10:00:00Z"
    }
  ]
}
```

第一阶段任务类型：

```text
start_instance
stop_instance
restart_instance
refresh_instance_status
```

任务状态：

```text
pending
running
succeeded
failed
cancelled
```

任务拉取规则：

1. Agent 只能 claim 分配给自己的任务；
2. claim 必须在数据库事务中原子完成，不能先查询再更新；
3. claim 后 Server 更新 `running`、`attempt`、`lease_owner` 和 `lease_expires_at`；
4. Agent 只能续租和回报自己持有 lease 的任务；
5. lease 过期后任务只能在未超过 `max_attempts` 时重新 claim；
6. 同一实例只允许一个 active operation；
7. Agent 执行完成后必须携带 operation、generation 和 attempt 回报；
8. 第一阶段不做自动无限重试；
9. Server 创建任务前已经完成 User Session、tenant scope 和 required permission code 校验；
10. Agent 不解析 User、Role 或 Permission，只校验任务属于本节点并执行冻结规格。

---

## 17. 任务结果回报接口

```http
POST /api/agent/tasks/report
```

成功请求：

```json
{
  "agent_id": "node-001",
  "task_id": "task-001",
  "operation_id": "op-001",
  "generation": 3,
  "attempt": 1,
  "status": "succeeded",
  "started_at": "2026-06-13T10:00:01Z",
  "finished_at": "2026-06-13T10:00:05Z",
  "message": "container started",
  "error": "",
  "result": {
    "container_id": "abc123",
    "endpoint": "http://192.168.1.10:8000",
    "docker_command": "docker run ..."
  }
}
```

失败请求：

```json
{
  "agent_id": "node-001",
  "task_id": "task-001",
  "operation_id": "op-001",
  "generation": 3,
  "attempt": 1,
  "status": "failed",
  "message": "docker run failed",
  "error": "image not found",
  "result": {
    "docker_command": "docker run ..."
  }
}
```

Server 处理逻辑：

1. 校验 task lease、attempt、operation 和 generation；
2. 以 `task_id + operation_id + attempt` 幂等更新任务状态；
3. 保存执行结果；
4. 如果任务关联实例，则按 generation 规则更新实例状态；
5. 保存错误信息；
6. 保存 Docker 命令快照；
7. 保存 endpoint。

---

## 18. 实例状态回报接口

```http
POST /api/agent/instances/report
```

请求：

```json
{
  "agent_id": "node-001",
  "instances": [
    {
      "instance_id": "inst-001",
      "task_id": "task-001",
      "operation_id": "op-001",
      "generation": 3,
      "container_id": "abc123",
      "status": "running",
      "health_status": "healthy",
      "endpoint": "http://192.168.1.10:8000",
      "last_error": "",
      "checked_at": "2026-06-13T10:00:10Z"
    }
  ]
}
```

用途：

1. Agent 定期回报容器状态；
2. Server 不依赖单次任务结果判断长期状态；
3. 容器异常退出时可以及时更新；
4. Web 可以看到最新状态；
5. Prometheus 只展示指标趋势，不替代实例状态回报；
6. Server 拒绝旧 generation、旧 checked_at 或冲突 operation 的报告。

---

## 19. Server 数据模型

### 19.1 Node

```go
type Node struct {
    ID                   string
    TenantID             string
    OwnerID              *string
    CreatedBy            string
    UpdatedBy            string
    Name                 string
    Hostname             string
    IP                   string
    AdvertisedAddress    string
    AgentVersion         string
    ProtocolVersion      string
    MetricsEnabled       bool
    MetricsScheme        string
    MetricsPort          int
    MetricsPath          string
    OS                   string
    Arch                 string
    CPUModel             string
    CPUCores             int
    MemoryTotalBytes     uint64
    Status               string
    LastHeartbeatAt      time.Time
    LastResourceReportAt time.Time
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

Node 状态：

```text
online
offline
unknown
maintenance
```

第一阶段只需要：

```text
online
offline
unknown
```

### 19.2 AgentTask

```go
type AgentTask struct {
    ID          string
    TenantID    string
    CreatedBy   string
    UpdatedBy   string
    NodeID      string
    Type        string
    InstanceID  string
    OperationID string
    SpecGeneration int64
    Status      string
    Attempt     int
    MaxAttempts int
    LeaseOwner  string
    LeaseExpiresAt *time.Time
    PayloadJSON string
    ResultJSON  string
    Error       string
    CreatedAt   time.Time
    StartedAt   *time.Time
    FinishedAt  *time.Time
}
```

---

## 20. Agent 本地状态

Agent 本地保存最少状态：

```text
data/
  agent-id
  runtime/
  cache/
logs/
  lightai-agent.log
```

第一阶段 Agent 不需要本地数据库。

Agent 可以在内存中维护：

1. 已拉取任务；
2. 正在执行的任务；
3. 最近一次资源采集结果；
4. 最近一次实例检查结果；
5. 最近一次 metrics 暴露数据。

如果 Agent 重启，应依靠 Server 和 Docker inspect 恢复实例状态。

任务可靠性、结果幂等和实例 generation 的完整规则见 `docs/08-engineering-contracts.md`。

---

## 21. 超时与重试策略

### 21.1 心跳失败

Agent 心跳失败时：

1. 记录 warn 日志；
2. 不退出；
3. 下一轮继续重试；
4. 连续失败可以增加日志提示；
5. Server 根据 `last_heartbeat_at` 判断离线。

### 21.2 注册失败

Agent 注册失败时：

1. 记录 error 日志；
2. 等待后重试；
3. 不启动任务执行；
4. 可以继续本地资源采集；
5. 可以继续暴露本地 `/metrics`；
6. 注册成功后恢复正常。

### 21.3 任务执行失败

任务执行失败时：

1. Agent 捕获 stdout / stderr；
2. Agent 回报 failed；
3. Server 更新任务和实例状态；
4. Web 展示错误信息；
5. 不自动无限重试。

---

## 22. 日志要求

Server 必须记录：

1. Server 启动；
2. 配置加载；
3. 数据库初始化；
4. Agent 注册；
5. Agent 心跳异常；
6. 节点离线判断；
7. 资源上报；
8. 任务创建；
9. 任务状态变更；
10. 实例状态变更；
11. `/metrics` 暴露状态；
12. `/metrics/targets` 生成结果；
13. API 错误。

Agent 必须记录：

1. Agent 启动；
2. 配置加载；
3. 本地 HTTP 服务启动；
4. 注册成功 / 失败；
5. 心跳成功 / 失败；
6. OS 资源采集成功 / 失败；
7. GPU 资源采集成功 / 失败；
8. Docker 状态采集成功 / 失败；
9. metrics 数据刷新；
10. 任务拉取；
11. 任务执行开始；
12. Docker 命令快照；
13. Docker stdout / stderr；
14. 任务执行结果；
15. 实例状态检查结果。

日志必须同时支持控制台输出和文件输出。

---

## 23. 安全边界

第一阶段使用 bootstrap/shared agent token 认证，避免任意 Agent 上报。该 token 用于 Agent 初始接入和共享认证，不等同于用户 API Key。

Agent 配置：

```yaml
server:
  token: "change-me"
```

Agent 请求 Header：

```http
Authorization: Bearer change-me
```

Server 校验失败返回：

```http
401 Unauthorized
```

认证边界：

```text
Agent bootstrap/shared token != User Session != Future API Key
```

规则：

1. Agent 注册、心跳、资源上报、任务 claim、任务结果和实例状态回报只使用 Agent token；
2. Agent API 不读取 Cookie，不使用 User Session，不执行 RBAC permission 解析；
3. 用户登录、退出和管理 API 使用 server-side User Session；
4. 用户登录状态不影响 Agent 注册和心跳；
5. User Session 不能调用 Agent API；
6. Agent token 不能调用用户管理 API；
7. AgentTask 的 tenant_id 和 created_by 由 Server 写入，Agent 不据此判断用户权限；
8. Future API Key 后续只用于模型服务调用。

用户、租户和 RBAC 设计见 `docs/09-auth-tenant-design.md`。

---

## 24. 开发验收

Phase 0 验收：

```bash
go run ./cmd/server
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/metrics
go run ./cmd/agent
curl http://127.0.0.1:18080/healthz
curl http://127.0.0.1:18080/metrics
```

Phase 1 验收：

```bash
go run ./cmd/server
go run ./cmd/agent
curl http://127.0.0.1:8080/api/nodes
curl http://127.0.0.1:8080/metrics/targets
```

预期：

1. 能看到 Agent 节点；
2. Agent 心跳正常；
3. 停止 Agent 后节点变为 offline；
4. Server 和 Agent 日志都有清楚记录；
5. Server 和 Agent 均预留 `/metrics`；
6. Server 能返回 Prometheus targets。
