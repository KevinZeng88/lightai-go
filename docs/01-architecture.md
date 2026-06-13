# LightAI Go 总体架构设计

## 1. 总体架构

LightAI Go 第一阶段采用 Server / Agent 架构。

Server 是控制面，负责管理节点、GPU、运行环境、模型、模型实例、任务和 Web/API。

Agent 是执行面，运行在每台 GPU 服务器上，负责注册、心跳、GPU 采集、任务拉取、Docker 启停、实例状态检查和日志回报。

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
Docker / GPU / Model Instance
```

## 2. Server 职责

Server 第一阶段负责：

1. 提供 Web/API 服务；
2. 保存节点信息；
3. 保存 GPU 设备和指标；
4. 判断 Agent 在线 / 离线；
5. 管理运行环境；
6. 管理模型定义；
7. 管理模型实例；
8. 生成实例启动、停止、重启任务；
9. 接收 Agent 任务执行结果；
10. 展示实例状态、错误、endpoint；
11. 提供健康检查接口；
12. 持久化数据到 SQLite。

Server 不直接操作 GPU，不直接执行 Docker 命令，不直接启动模型。

## 3. Agent 职责

Agent 第一阶段负责：

1. 读取本地配置；
2. 注册到 Server；
3. 周期性发送心跳；
4. 周期性采集主机和 GPU 状态；
5. 周期性拉取任务；
6. 执行 Docker 启动、停止、重启；
7. 检查容器状态；
8. 检查实例端口；
9. 回报任务结果；
10. 回报实例状态；
11. 输出本地日志。

Agent 不能自行决定全局架构，不保存复杂业务状态，只负责本机执行。

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

## 5. 核心数据对象

第一阶段核心对象包括：

1. Node：一台 GPU 服务器；
2. GPUDevice：一张 GPU 卡；
3. GPUMetric：GPU 指标；
4. RuntimeEnvironment：运行环境模板；
5. Model：模型定义；
6. ModelInstance：模型实例；
7. AgentTask：Server 下发给 Agent 的任务；
8. TaskResult：Agent 回报的任务结果；
9. InstanceStatus：实例状态；
10. DockerRunSpec：最终 Docker 启动参数快照。

## 6. 核心状态流

### 6.1 Agent 接入流程

```text
Agent 启动
  ↓
读取配置
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

### 6.2 GPU 监控流程

```text
Agent 定时采集 GPU
  ↓
执行 GPU Collector
  ↓
生成 GPUDevice / GPUMetric
  ↓
上报 Server
  ↓
Server 保存最新状态
  ↓
Web 展示 GPU 列表和指标
```

### 6.3 模型实例启动流程

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

### 6.4 模型实例停止流程

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

不使用消息队列，不使用 gRPC，不使用 WebSocket 作为第一阶段强依赖。

## 8. 存储设计

第一阶段使用 SQLite。

原因：

1. 部署简单；
2. 适合中小客户；
3. 便于单机 Server 运行；
4. 便于备份和排障；
5. 后续可以迁移到 PostgreSQL。

## 9. Web 设计原则

第一阶段 Web 页面优先做清楚，不追求复杂效果。

页面包括：

1. 节点列表；
2. 节点详情；
3. GPU 资源页面；
4. 运行环境页面；
5. 模型定义页面；
6. 模型实例页面；
7. 实例详情页面；
8. 任务记录页面；
9. 系统诊断页面。

## 10. 后续扩展方向

第二阶段以后再扩展：

1. API Key；
2. 统一 OpenAI-compatible API；
3. Token 统计；
4. 成本核算；
5. 简单调度；
6. 多实例路由；
7. Prometheus 指标；
8. 更多 GPU 厂商；
9. PostgreSQL；
10. 权限体系。

