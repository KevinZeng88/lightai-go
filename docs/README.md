# LightAI Go 文档阅读顺序

Claude / Codex 开发前必须先阅读本目录文档，并严格按文档开发。

## 1. 阅读顺序

请按以下顺序阅读：

1. `00-project-scope.md`
2. `01-architecture.md`
3. `02-server-agent-design.md`
4. `03-resource-monitoring-design.md`
5. `04-observability-design.md`
6. `05-runtime-environment-design.md`
7. `06-model-design.md`
8. `07-instance-lifecycle-design.md`
9. `08-engineering-contracts.md`
10. `09-auth-tenant-design.md`
11. `10-mvp-development-plan.md`

跨文档出现冲突时，以 `08-engineering-contracts.md` 的统一工程契约为准。

## 2. 当前阶段重点

当前开发窗口只允许实现 Phase 0、Phase 0.5、Phase 1、Phase 2。这是当前窗口范围，不代表第一阶段只有这些 Phase。

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
10. default tenant；
11. bootstrap platform admin；
12. 本地 User、Membership、TenantMembershipRole；
13. built-in Role、tenant custom Role 和 RolePermission；
14. 系统只读 Permission catalog；
15. 基础登录、退出和当前用户查询；
16. Session、CSRF 和实时 permission code 校验；
17. 核心资源 tenant / owner / audit 字段；
18. Agent 注册；
19. Agent 心跳；
20. 节点在线 / 离线；
21. SystemCollector；
22. NvidiaCollector；
23. MetaxCollector；
24. CollectorRegistry；
25. 资源上报；
26. 节点查询 API；
27. GPU 查询 API；
28. Collector 诊断信息。

MockGPUCollector 仅用于 `development` / `test` profile，默认关闭，不得替代 Phase 2 的 NVIDIA 和 MetaX 真实环境验收。

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
12. 用户自定义 Permission code、资源级 ACL、字段级权限和多级组织继承；
13. 高可用控制面；
14. 完整模型服务代理；
15. SSO / LDAP / OAuth；
16. 租户级 GPU 配额、账单和隔离调度。

第一阶段正式实现 Tenant/User/Membership、TenantMembershipRole、built-in/custom Role、系统只读 Permission catalog、RolePermission、Session 和资源归属字段，不实现上述资源级权限与组织能力。

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

## 6. 身份与凭证边界

```text
Agent bootstrap/shared token != User Session != Future API Key
```

Agent token 只用于 Agent 注册、心跳、资源上报、任务 claim 和状态回报。
User Session 只用于 Web/API 管理操作。
Future API Key 后续只用于模型服务调用。
三类凭证不能混用。

## 7. 当前必须实现的接口

Phase 0 / Phase 0.5 / Phase 1 / Phase 2 至少需要实现：

```text
GET  /healthz
GET  /metrics
GET  /metrics/targets

POST /api/v1/auth/login
POST /api/v1/auth/logout
POST /api/v1/auth/change-password
GET  /api/v1/auth/me

GET  /api/v1/users
POST /api/v1/users
GET  /api/v1/tenants
POST /api/v1/tenants
GET  /api/v1/tenant-memberships
POST /api/v1/tenant-memberships
GET  /api/v1/roles
POST /api/v1/roles
GET  /api/v1/permissions

POST /api/v1/agent/register
POST /api/v1/agent/heartbeat
POST /api/v1/agent/resources/report

GET  /api/v1/nodes
GET  /api/v1/nodes/{node_id}
GET  /api/v1/gpus
GET  /api/v1/gpus/{gpu_id}
```

`/metrics/targets` 只由 Server 提供。

## 8. 开发验收要求

每次开发完成后必须执行：

```bash
go fmt ./...
go test ./...
go build ./cmd/server
go build ./cmd/agent
git diff --check
```

如有失败，必须先修复后再提交。

## 9. 提交要求

每个阶段完成后单独提交。

提交信息建议：

```text
phase0: add server agent skeleton
phase0.5: add auth tenant rbac foundation
phase1: add agent register heartbeat
phase2: add system gpu resource reporting
```

不要把多个阶段混在一个大提交里。

## 10. 第一轮开发建议

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
