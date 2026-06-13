# LightAI Go MVP 开发计划

## 1. 开发原则

MVP 开发必须按阶段推进，每个阶段都要能运行、能验证、能提交。

Claude 只允许根据 docs 文档开发，不允许自行研究 GPUStack，不允许扩大范围，不允许提前实现 Token、计费、复杂网关等后续功能。

每个阶段完成后，需要执行：

```bash
go fmt ./...
go test ./...
go build ./cmd/server
go build ./cmd/agent
git diff --check
```

如有失败，必须先修复后再提交。

---

## 2. Phase 0：基础骨架

目标：

1. Server 可启动；
2. Agent 可启动；
3. 配置文件可加载；
4. 日志可输出；
5. Server 健康检查接口可访问；
6. Agent 健康检查接口可访问；
7. Server `/metrics` 可访问；
8. Agent `/metrics` 可访问；
9. Server `/metrics/targets` 可访问；
10. 创建 Prometheus / Grafana 部署目录占位。

任务：

1. 创建 `cmd/server/main.go`；
2. 创建 `cmd/agent/main.go`；
3. 创建配置加载模块；
4. 创建日志模块；
5. 创建版本模块；
6. Server 提供 `/healthz`；
7. Server 提供 `/metrics`；
8. Server 提供 `/metrics/targets`；
9. Agent 启动本地 HTTP 服务；
10. Agent 提供 `/healthz`；
11. Agent 提供 `/metrics`；
12. 创建示例配置文件 `configs/server.yaml`；
13. 创建示例配置文件 `configs/agent.yaml`；
14. 创建 `deploy/observability/` 目录占位；
15. 创建 Prometheus / Grafana 配置占位文件。

完成标准：

```bash
go run ./cmd/server
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/metrics
curl http://127.0.0.1:8080/metrics/targets

go run ./cmd/agent
curl http://127.0.0.1:18080/healthz
curl http://127.0.0.1:18080/metrics
```

验收要求：

1. Server 日志能看到启动信息；
2. Agent 日志能看到启动信息；
3. `/healthz` 返回 JSON；
4. `/metrics` 返回 Prometheus 文本格式或基础占位指标；
5. `/metrics/targets` 返回合法 JSON 数组；
6. `go test ./...` 通过；
7. `go build ./cmd/server` 通过；
8. `go build ./cmd/agent` 通过。

---

## 3. Phase 1：Agent 注册与心跳

目标：

1. Agent 能注册到 Server；
2. Server 能保存节点；
3. Agent 能周期性心跳；
4. Server 能判断节点在线 / 离线；
5. Agent 注册时能上报 `agent_metrics_url`；
6. Server `/metrics/targets` 能返回 Agent target。

任务：

1. 定义 Node 数据结构；
2. 初始化 SQLite；
3. 实现 Node 表；
4. 实现 Agent 注册 API；
5. 实现 Agent 心跳 API；
6. 实现节点列表 API；
7. Agent 实现注册逻辑；
8. Agent 实现心跳循环；
9. Server 定期计算节点状态；
10. Server 保存 `agent_metrics_url`；
11. Server 根据在线节点生成 `/metrics/targets`；
12. 增加注册和心跳日志。

完成标准：

```text
启动 Server
启动 Agent
调用节点列表 API
可以看到 Agent 节点在线
调用 /metrics/targets
可以看到 Agent metrics 地址
停止 Agent 后一段时间节点变为 offline
```

验收命令：

```bash
go run ./cmd/server
go run ./cmd/agent

curl http://127.0.0.1:8080/api/nodes
curl http://127.0.0.1:8080/metrics/targets
```

预期：

1. `/api/nodes` 可以看到节点；
2. 节点状态为 online；
3. 节点包含 `agent_metrics_url`；
4. `/metrics/targets` 返回 Agent target；
5. 停止 Agent 后节点变为 offline；
6. Agent 重启后可以重新注册或恢复心跳。

---

## 4. Phase 2：OS / GPU 资源采集与上报

目标：

1. Agent 能采集操作系统资源；
2. Agent 能采集 CPU、内存、Swap、磁盘、网络基础信息；
3. Agent 能发现 GPU；
4. Agent 能采集 GPU 指标；
5. Agent 能把资源状态上报 Server；
6. Server 能保存节点、OS 和 GPU 最新状态；
7. Server 能保存 Collector 诊断信息；
8. Server 能提供资源查询 API；
9. Agent `/metrics` 能暴露 OS 和 GPU 指标；
10. Server `/metrics/targets` 能返回 Agent targets。

任务：

1. 定义 SystemSnapshot；
2. 定义 FilesystemSnapshot；
3. 定义 NetworkInterfaceSnapshot；
4. 定义 CollectorDiagnosis；
5. 定义 GPUDevice；
6. 定义 GPUMetric；
7. 定义 SystemCollector 接口；
8. 实现 GopsutilSystemCollector；
9. 定义 GPUCollector 接口；
10. 实现 MockGPUCollector；
11. 预留 NvidiaCollector；
12. 预留 MetaxCollector；
13. 实现 CollectorRegistry；
14. 实现资源上报 API；
15. Agent 周期性上报资源；
16. Server 保存最新 OS 状态；
17. Server 保存最新 GPU 状态；
18. Server 保存 Collector 诊断信息；
19. 提供节点详情 API；
20. 提供 GPU 查询 API；
21. 更新 Agent `/metrics`；
22. 更新 Server `/metrics/targets`；
23. 增加采集失败不崩溃处理；
24. 增加 Mock 数据配置开关。

完成标准：

无真实 GPU 时：

```text
Agent 可以启动
SystemCollector 能采集 OS 资源
MockGPUCollector 能上报模拟 GPU
Agent /metrics 可看到 OS / GPU 指标
Server 能看到节点 CPU / 内存 / 磁盘 / GPU
/metrics/targets 能返回 Agent metrics 地址
```

采集失败时：

```text
NvidiaCollector 工具不存在不导致 Agent 崩溃
Docker 不可用不导致 Agent 崩溃
诊断信息可以通过 API 查看
```

验收命令：

```bash
go run ./cmd/server
go run ./cmd/agent

curl http://127.0.0.1:8080/api/nodes
curl http://127.0.0.1:8080/api/nodes/<node_id>
curl http://127.0.0.1:8080/api/gpus
curl http://127.0.0.1:18080/metrics
curl http://127.0.0.1:8080/metrics/targets
```

---

## 5. Phase 3：运行环境管理

目标：

1. 可以创建 Docker 运行环境；
2. 可以编辑运行环境；
3. 可以预览 Docker 参数；
4. 未启用参数不出现在最终命令中。

任务：

1. 定义 RuntimeEnvironment；
2. 定义 DockerRunSpec；
3. 实现运行环境 CRUD；
4. 实现参数启用开关；
5. 实现 Docker 命令预览；
6. 实现配置校验；
7. 防止删除已被实例引用的运行环境。

完成标准：

```text
可以创建一个 vLLM Docker 运行环境
可以配置镜像、命令、环境变量、volume、device、shm-size
未启用参数不会出现在命令预览中
```

---

## 6. Phase 4：模型定义管理

目标：

1. 可以创建模型定义；
2. 可以关联默认运行环境；
3. 模型可以被实例引用。

任务：

1. 定义 Model；
2. 实现模型 CRUD；
3. 支持模型路径；
4. 支持默认端口；
5. 支持默认上下文长度；
6. 支持默认启动参数；
7. 支持 runtime metrics endpoint 预留字段；
8. 防止删除已被实例引用的模型。

完成标准：

```text
可以创建 qwen、deepseek 等模型定义
模型定义可以选择默认运行环境
实例创建时可以选择模型
```

---

## 7. Phase 5：模型实例创建与任务下发

目标：

1. 可以创建模型实例；
2. Server 能生成任务；
3. Agent 能拉取任务；
4. Agent 能回报任务结果。

任务：

1. 定义 ModelInstance；
2. 定义 AgentTask；
3. 定义 TaskResult；
4. 实现实例创建 API；
5. 实现任务表；
6. 实现任务拉取 API；
7. 实现任务回报 API；
8. Agent 实现任务轮询；
9. Server 更新任务状态；
10. 实例对象预留 runtime metrics endpoint 字段。

完成标准：

```text
创建实例后生成 start_instance 任务
Agent 可以拉取任务
Agent 可以回报成功或失败
任务状态可查询
```

---

## 8. Phase 6：Docker 启停实例

目标：

1. Agent 可以执行 docker run；
2. Agent 可以执行 docker stop；
3. Agent 可以回报容器 ID；
4. Server 可以更新实例状态。

任务：

1. 实现 Docker 命令生成；
2. 实现 docker run；
3. 实现 docker stop；
4. 实现 docker inspect；
5. 记录 Docker 命令快照；
6. 记录 stdout / stderr；
7. 回报容器 ID；
8. 更新实例状态；
9. 启动失败时记录 last_error。

完成标准：

```text
可以从平台启动一个容器
可以停止容器
启动失败时可以看到错误
实例状态能从 starting 进入 running 或 failed
```

---

## 9. Phase 7：实例健康检查与 endpoint

目标：

1. Agent 能检查容器状态；
2. Agent 能检查端口；
3. Server 能展示 endpoint；
4. 页面或 API 能看到实例健康状态；
5. 后续可以把实例健康状态暴露到 Prometheus。

任务：

1. Agent 定期 docker inspect；
2. Agent 检查端口可达；
3. Agent 上报实例状态；
4. Server 保存 endpoint；
5. Server 展示 last_error；
6. Server 展示 last_checked_at；
7. 提供实例详情 API；
8. Agent `/metrics` 预留实例健康指标。

完成标准：

```text
容器退出后实例状态变更
端口不可达时健康检查失败
实例详情中可看到 endpoint、状态、错误和最后检查时间
```

---

## 10. Phase 8：基础 Web 页面

目标：

1. 能通过 Web 查看核心资源；
2. 能完成基础实例启停操作；
3. 能看到基础监控入口。

页面：

1. Dashboard；
2. 节点列表；
3. 节点详情；
4. GPU 资源；
5. 运行环境；
6. 模型定义；
7. 模型实例；
8. 实例详情；
9. 任务记录；
10. 系统诊断；
11. 监控看板。

完成标准：

```text
用户可以通过 Web 完成：
查看节点
查看 CPU / 内存 / 磁盘
查看 GPU
查看 Collector 诊断
创建运行环境
创建模型
创建实例
启动实例
停止实例
查看 endpoint
查看错误
打开监控看板
```

---

## 11. Phase 9：Prometheus / Grafana 内置部署

目标：

1. 提供内置 Prometheus / Grafana Docker Compose；
2. Prometheus 能抓取 Server；
3. Prometheus 能通过 `/metrics/targets` 动态发现 Agent；
4. Grafana 能连接 Prometheus；
5. Grafana 能展示 Node / GPU 基础 Dashboard；
6. LightAI Web 能内嵌 Grafana 页面。

任务：

1. 创建 `deploy/observability/docker-compose.yml`；
2. 创建 Prometheus 配置；
3. 配置 Server scrape；
4. 配置 Agent http_sd scrape；
5. 创建 Grafana datasource provisioning；
6. 创建 Grafana dashboard provisioning；
7. 创建基础 Node Overview Dashboard；
8. 创建基础 GPU Overview Dashboard；
9. Web 增加监控看板入口；
10. 预留 `/api/observability/status`。

完成标准：

```text
docker compose 可以启动 Prometheus 和 Grafana
Prometheus targets 能看到 Server 和 Agent
Grafana 能看到 LightAI dashboard
LightAI Web 可以打开或内嵌 Grafana
Mock GPU 指标能出现在 Grafana 中
```

---

## 12. 第二阶段入口

第一阶段稳定后，再进入：

1. API Key；
2. 统一模型访问入口；
3. OpenAI-compatible proxy；
4. Token 统计；
5. 额度；
6. 成本；
7. 简单调度；
8. RuntimeCollector；
9. 模型服务请求指标；
10. 告警规则。

