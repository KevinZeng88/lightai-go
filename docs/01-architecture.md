# LightAI Go 总体架构设计

## 1. 总体架构

LightAI Go 第一阶段采用 Server / Agent 架构。

Server 是控制面，负责管理节点、操作系统资源、GPU 资源、运行环境、模型、模型实例、任务、Web/API 和监控组件配置。

Agent 是执行面，运行在每台 GPU 服务器上，负责注册、心跳、操作系统资源采集、GPU 资源采集、资源上报、任务拉取、Docker 启停、实例状态检查、日志回报和 Prometheus `/metrics` 指标暴露。

整体链路：

```text
Web / API
   ↓
Server
   ↓
任务下发
   ↓
Agent
   ↓
OS / GPU / Docker / Model Instance
```

监控链路：

```text
Prometheus
   ↓
Server /metrics
Server /metrics/targets
Agent /metrics
   ↓
Grafana
   ↓
LightAI Web 内嵌看板
```

第一阶段必须保持两个边界：

```text
业务管理链路：Agent → Server → SQLite → Web
监控展示链路：Server/Agent /metrics → Prometheus → Grafana → Web
```

Prometheus / Grafana 只用于监控趋势、指标展示和告警，不作为 Server 的业务状态来源。

---

## 2. Server 职责

Server 第一阶段负责：

1. 提供 Web/API 服务；
2. 保存节点信息；
3. 保存操作系统资源最新状态；
4. 保存 GPU 设备和最新指标；
5. 保存 Collector 诊断信息；
6. 判断 Agent 在线 / 离线；
7. 管理运行环境；
8. 管理模型定义；
9. 管理模型实例；
10. 生成实例启动、停止、重启任务；
11. 接收 Agent 任务执行结果；
12. 接收 Agent 实例状态回报；
13. 展示实例状态、错误、endpoint；
14. 提供健康检查接口 `/healthz`；
15. 提供 Server `/metrics`；
16. 提供 Prometheus 动态发现接口 `/metrics/targets`；
17. 提供 Prometheus / Grafana 配置和内嵌入口；
18. 持久化管理数据到 SQLite。

Server 不直接操作 GPU，不直接执行 Docker 命令，不直接启动模型。

Server 不能通过 Prometheus 查询结果判断节点是否在线或实例是否运行。节点状态、GPU 状态、实例状态必须来自 Agent 上报和数据库记录。

---

## 3. Agent 职责

Agent 第一阶段负责：

1. 读取本地配置；
2. 加载或生成稳定 Agent ID；
3. 启动本地 HTTP 服务；
4. 提供 Agent `/healthz`；
5. 提供 Agent `/metrics`；
6. 注册到 Server；
7. 周期性发送心跳；
8. 周期性采集操作系统资源；
9. 周期性采集 GPU 状态；
10. 周期性上报资源；
11. 周期性拉取任务；
12. 执行 Docker 启动、停止、重启；
13. 检查容器状态；
14. 检查实例端口；
15. 回报任务结果；
16. 回报实例状态；
17. 输出本地日志。

Agent 不能自行决定全局架构，不保存复杂业务状态，只负责本机执行。

Agent 采集失败不能导致整体退出。GPU 不存在、Docker 不可用、某个 Collector 不可用，都应作为诊断信息上报或记录。

---

## 4. 模块划分

项目目录采用以下分层：

```text
cmd/
  server/
  agent/

internal/
  common/
    types/
    config/
    log/
    errors/
    version/

  server/
    api/
    db/
    node/
    gpu/
    runtime/
    model/
    instance/
    task/
    health/

  agent/
    register/
    heartbeat/
    gpu/
    docker/
    task/
    instance/
    health/
```

后续可以增加：

```text
internal/
  server/
    observability/
    diagnostics/

  agent/
    system/
    metrics/
    collectors/
```

第一阶段不为了目录完美而过度拆分，但必须保证职责清楚。

---

## 5. 核心数据对象

第一阶段核心对象包括：

1. Node：一台 GPU 服务器；
2. SystemSnapshot：节点操作系统资源快照；
3. FilesystemSnapshot：文件系统资源快照；
4. NetworkInterfaceSnapshot：网络接口基础信息；
5. GPUDevice：一张 GPU 卡；
6. GPUMetric：GPU 最新指标；
7. CollectorDiagnosis：采集器诊断信息；
8. RuntimeEnvironment：运行环境模板；
9. Model：模型定义；
10. ModelInstance：模型实例；
11. AgentTask：Server 下发给 Agent 的任务；
12. TaskResult：Agent 回报的任务结果；
13. InstanceStatus：实例状态；
14. DockerRunSpec：最终 Docker 启动参数快照。

---

## 6. 核心状态流

### 6.1 Agent 接入流程

```text
Agent 启动
  ↓
读取配置
  ↓
加载或生成 Agent ID
  ↓
启动本地 /healthz 和 /metrics
  ↓
POST /api/agent/register
  ↓
Server 创建或更新 Node
  ↓
Agent 周期性 heartbeat
  ↓
Server 更新 last_heartbeat_at
  ↓
Server 根据超时时间判断在线 / 离线
```

### 6.2 操作系统资源监控流程

```text
Agent 定时采集 OS 资源
  ↓
SystemCollector 采集 CPU / 内存 / Swap / 磁盘 / 网络
  ↓
刷新 Agent /metrics
  ↓
POST /api/agent/resources/report
  ↓
Server 保存最新状态
  ↓
Web/API 展示节点资源
  ↓
Prometheus 抓取 Agent /metrics
  ↓
Grafana 展示趋势图
```

### 6.3 GPU 监控流程

```text
Agent 定时采集 GPU
  ↓
执行 GPU Collector
  ↓
生成 GPUDevice / GPUMetric / CollectorDiagnosis
  ↓
刷新 Agent /metrics
  ↓
上报 Server
  ↓
Server 保存最新状态
  ↓
Web/API 展示 GPU 列表和指标
  ↓
Prometheus 抓取 Agent /metrics
  ↓
Grafana 展示 GPU 趋势
```

### 6.4 Prometheus 动态发现流程

```text
Agent 注册时上报 agent_metrics_url
  ↓
Server 保存 Node.agent_metrics_url
  ↓
Prometheus 请求 Server /metrics/targets
  ↓
Server 返回 online Agent target 列表
  ↓
Prometheus 抓取 Agent /metrics
```

### 6.5 模型实例启动流程

```text
用户创建实例
  ↓
Server 保存 ModelInstance
  ↓
Server 创建 StartInstanceTask
  ↓
Agent 拉取任务
  ↓
Agent 生成 Docker 命令
  ↓
Agent 执行 docker run
  ↓
Agent 检查容器状态
  ↓
Agent 回报结果
  ↓
Server 更新实例状态和 endpoint
```

### 6.6 模型实例停止流程

```text
用户停止实例
  ↓
Server 创建 StopInstanceTask
  ↓
Agent 拉取任务
  ↓
Agent 执行 docker stop
  ↓
Agent 回报结果
  ↓
Server 更新实例状态
```

---

## 7. 第一阶段通信方式

第一阶段采用 HTTP Pull 模式。

Agent 主动访问 Server：

```text
POST /api/agent/register
POST /api/agent/heartbeat
POST /api/agent/resources/report
GET  /api/agent/tasks/pull
POST /api/agent/tasks/report
POST /api/agent/instances/report
```

Server 和 Agent 指标接口：

```text
GET /healthz
GET /metrics
GET /metrics/targets   # Server only
```

不使用消息队列，不使用 gRPC，不使用 WebSocket 作为第一阶段强依赖。

---

## 8. 资源监控设计原则

资源监控采用四层设计：

```text
采集层 Collector：
  SystemCollector
  GPUCollector
  DockerCollector
  RuntimeCollector

状态层 State Report：
  Agent 上报 Server
  Server 保存 SQLite
  Web 展示当前状态

指标层 Metrics Exporter：
  Agent /metrics
  Server /metrics
  Prometheus scrape

看板层 Grafana：
  内置 Grafana
  Web iframe / reverse proxy
  Dashboard provisioning
```

第一阶段实现：

1. SystemCollector；
2. MockGPUCollector；
3. GPUCollector 接口；
4. NvidiaCollector 预留；
5. MetaxCollector 预留；
6. Agent `/metrics`；
7. Server `/metrics`；
8. Server `/metrics/targets`。

DockerCollector 和 RuntimeCollector 作为第二阶段或后续阶段实现。

---

## 9. 存储设计

第一阶段使用 SQLite。

原因：

1. 部署简单；
2. 适合中小客户；
3. 便于单机 Server 运行；
4. 便于备份和排障；
5. 后续可以迁移到 PostgreSQL。

SQLite 保存平台管理数据：

1. 节点最新状态；
2. OS 最新状态；
3. GPU 最新状态；
4. Collector 诊断；
5. 运行环境；
6. 模型定义；
7. 模型实例；
8. 任务记录；
9. Docker 命令快照；
10. 最近错误。

SQLite 不保存长期时序曲线。CPU、内存、磁盘、GPU 利用率等历史趋势由 Prometheus 保存。

---

## 10. Prometheus / Grafana 集成原则

LightAI Go 支持三种 Observability 模式：

```text
builtin
external
disabled
```

builtin 模式：

1. LightAI Go 提供 Prometheus / Grafana Docker Compose；
2. LightAI Go 提供 Prometheus 配置；
3. LightAI Go 提供 Grafana datasource provisioning；
4. LightAI Go 提供 Grafana dashboard provisioning；
5. LightAI Web 内嵌 Grafana 看板。

external 模式：

1. 客户已有 Prometheus / Grafana；
2. LightAI Go 只暴露 `/metrics` 和 `/metrics/targets`；
3. 客户自己的 Prometheus 抓取 LightAI 指标。

disabled 模式：

1. 不启用 Prometheus / Grafana；
2. LightAI Go 仍然展示当前资源状态；
3. 不展示历史趋势图。

---

## 11. Web 设计原则

第一阶段 Web 页面优先做清楚，不追求复杂效果。

页面包括：

1. Dashboard；
2. 节点列表；
3. 节点详情；
4. GPU 资源页面；
5. 运行环境页面；
6. 模型定义页面；
7. 模型实例页面；
8. 实例详情页面；
9. 任务记录页面；
10. 系统诊断页面；
11. 监控看板页面。

监控看板页面可以内嵌 Grafana：

```text
/monitoring
/monitoring/nodes
/monitoring/gpu
/monitoring/instances
```

第一阶段可以先通过 iframe 方式内嵌 Grafana。后续再做统一鉴权和反向代理增强。

---

## 12. 后续扩展方向

第二阶段以后再扩展：

1. API Key；
2. 统一 OpenAI-compatible API；
3. Token 统计；
4. 成本核算；
5. 简单调度；
6. 多实例路由；
7. DockerCollector；
8. RuntimeCollector；
9. 更多 GPU 厂商；
10. PostgreSQL；
11. 权限体系；
12. Grafana 统一鉴权；
13. 告警规则；
14. 企业微信 / 邮件告警。

