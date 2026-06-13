# LightAI Go 文档阅读顺序

Claude / Codex 开发前必须先阅读本目录文档，并严格按文档开发。

## 1. 阅读顺序

请按以下顺序阅读：

1. `00-project-scope.md`
2. `01-architecture.md`
3. `02-server-agent-design.md`
4. `03-resource-monitoring-design.md`
5. `04-observability-design.md`
6. `10-mvp-development-plan.md`

## 2. 当前阶段重点

当前只允许实现 Phase 0、Phase 1、Phase 2。

优先级：

1. Server 可启动；
2. Agent 可启动；
3. 配置加载；
4. 日志输出；
5. Server `/healthz`；
6. Agent `/healthz`；
7. Server `/metrics`；
8. Agent `/metrics`；
9. Server `/metrics/targets`；
10. Agent 注册；
11. Agent 心跳；
12. 节点在线 / 离线；
13. SystemCollector；
14. MockGPUCollector；
15. 资源上报；
16. 节点查询 API；
17. GPU 查询 API；
18. Collector 诊断信息。

## 3. 当前禁止实现

当前禁止实现：

1. Kubernetes；
2. Ray；
3. 多集群；
4. 复杂调度；
5. 模型市场；
6. 自动下载模型；
7. API Key；
8. Token 统计；
9. 额度管理；
10. 成本核算；
11. 复杂统一网关；
12. 多租户权限体系；
13. 高可用控制面；
14. 完整模型服务代理；
15. 复杂权限系统。

## 4. GPUStack 使用原则

GPUStack 仅作为架构参考，不允许复制代码，不允许逐行翻译。

Claude 不需要自行研究 GPUStack。
LightAI Go 的开发依据是本目录下的设计文档。

如果发现 GPUStack 与本目录文档存在差异，以本目录文档为准。

## 5. 资源监控边界

LightAI Go 必须保持以下边界：

```text
Agent 负责采集
Server 负责管理
SQLite 保存当前状态
Prometheus 保存历史时序指标
Grafana 负责展示
```

Prometheus 不作为业务状态来源。
Server 不通过 Prometheus 查询结果判断节点是否在线、GPU 是否存在、实例是否运行。

## 6. 当前必须实现的接口

Phase 0 / Phase 1 / Phase 2 至少需要实现：

```text
GET  /healthz
GET  /metrics
GET  /metrics/targets

POST /api/agent/register
POST /api/agent/heartbeat
POST /api/agent/resources/report

GET  /api/nodes
GET  /api/nodes/{node_id}
GET  /api/gpus
GET  /api/gpus/{gpu_id}
```

`/metrics/targets` 只由 Server 提供。

## 7. 开发验收要求

每次开发完成后必须执行：

```bash
go fmt ./...
go test ./...
go build ./cmd/server
go build ./cmd/agent
git diff --check
```

如有失败，必须先修复后再提交。

## 8. 提交要求

每个阶段完成后单独提交。

提交信息建议：

```text
phase0: add server agent skeleton
phase1: add agent register heartbeat
phase2: add system gpu resource reporting
```

不要把多个阶段混在一个大提交里。

## 9. 第一轮开发建议

第一轮只做 Phase 0：

1. Server main；
2. Agent main；
3. 配置加载；
4. 日志；
5. 版本信息；
6. `/healthz`；
7. `/metrics`；
8. `/metrics/targets`；
9. 示例配置文件；
10. 基础测试。

第一轮不要做数据库，不要做注册心跳，不要做资源采集。

