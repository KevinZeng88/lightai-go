> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference or historical compatibility document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

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
curl http://127.0.0.1:18080/healthz
curl http://127.0.0.1:18080/metrics
curl http://127.0.0.1:18080/metrics/targets

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

## 3. Phase 0.5：基础认证、租户与 RBAC

目标：

1. 初始化 default tenant；
2. 初始化 bootstrap platform admin；
3. 支持全局本地 User；
4. 支持 TenantMembership；
5. 支持 TenantMembershipRole 多 Role 绑定；
6. 支持全局只读 built-in admin/operator/viewer；
7. 支持 tenant custom Role；
8. 支持系统只读 Permission catalog 和 RolePermission；
9. 支持登录、退出、修改密码和当前用户查询；
10. 支持 server-side Session、CSRF、Origin 和登录限流；
11. 每次请求实时解析 Membership、Roles 和 Permissions；
12. API 统一按 required permission code 授权；
13. 区分 platform admin、tenant admin、Agent token 和 User Session；
14. 核心资源预留 tenant、owner 和 audit 字段。

任务：

1. 定义 Tenant；
2. 定义 User 和 `is_platform_admin`；
3. 定义 TenantMembership；
4. 定义 TenantMembershipRole；
5. 定义 Role；
6. 定义 Permission；
7. 定义 RolePermission；
8. 定义 Session；
9. 初始化系统只读 Permission catalog；
10. 初始化全局 built-in admin/operator/viewer Role 和 RolePermission；
11. 幂等初始化 default tenant；
12. 幂等初始化 bootstrap platform admin、Membership 和 built-in admin Role 绑定；
13. 使用 Argon2id 保存和验证本地密码；
14. 实现 `POST /api/auth/login`；
15. 实现多 Membership tenant 选择响应；
16. 实现 `POST /api/auth/logout`；
17. 实现 `POST /api/auth/change-password` 和 bootstrap 强制首次改密；
18. 实现 `GET /api/auth/me`；
19. 实现 12 小时滑动 Session 和安全 Cookie；
20. 实现 CSRF token 和 Origin 校验；
21. 实现按 username 和来源的登录限流；
22. 实现每请求实时 Role / Permission 解析；
23. 实现 required permission 中间件；
24. 实现 platform admin 的全局 User / Tenant 管理；
25. 实现 tenant admin 的 Membership 和 custom Role 管理；
26. 实现 custom Role 的 RolePermission 管理；
27. 实现 built-in Role、Permission catalog 只读保护；
28. 实现 Tenant、User、Membership 禁用优先；
29. 实现 custom Role 未分配可删除、已分配需先解绑；
30. RuntimeEnvironment、Model、ModelInstance、AgentTask、Node、GPUDevice 加入 tenant、owner 和 audit 字段；
31. 所有用户侧查询使用 `session.current_tenant_id` 过滤；
32. 所有用户侧创建操作由 Server 写入归属字段；
33. Node / GPU 写入 `owner_id=null`、`created_by=system`；
34. 明确 Agent 接口继续只使用 bootstrap/shared agent token；
35. 增加结构化认证、授权和资源操作审计日志。

完成标准：

1. Permission catalog 和 built-in Role 初始化幂等；
2. Server 首次启动创建 default tenant 和 bootstrap platform admin；
3. 自动生成的初始密码只输出一次；
4. 可以登录、查询当前用户和退出；
5. bootstrap 用户首次改密前只能访问 me、change-password 和 logout；
6. `/api/auth/me` 返回 current tenant、roles 和 permissions；
7. Session 12 小时滑动过期，logout 后立即失效；
8. CSRF、Origin 和登录限流生效；
9. 一个 Membership 可以绑定多个 Role，permission 取并集；
10. built-in Role 全局只读、tenant_id 为空且不可删除；
11. custom Role 必须绑定 Tenant，只能当前 Tenant 管理；
12. custom Role 只能绑定系统 Permission，不允许用户创建 permission code；
13. 被分配 custom Role 不能删除，解绑后可以删除；
14. active Membership 最后一个 active Role 不能移除；
15. User、Tenant、Membership 或 Role 禁用后下一次请求立即生效；
16. API 不比较 Role 名称，统一按 required permission code 判断；
17. built-in admin 只具备 tenant admin 权限，不等于 platform admin；
18. tenant admin 只能管理当前 Tenant Membership、custom Role 和资源；
19. tenant admin 只能把已有 User 加入当前 Tenant，不能创建全局 User；
20. platform admin 可以管理全局 User 和 Tenant；
21. 所有新建用户业务资源都有 tenant_id、owner_id、created_by、updated_by；
22. owner_id 不等于 created_by，也不作为 ACL；
23. Node / GPU 的 owner_id=null、created_by=system；
24. Agent 注册、心跳、资源上报和任务 claim 不受 User Session 影响；
25. Agent token 不能调用用户管理 API，User Session 不能调用 Agent API。

Phase 0.5 明确不实现：

1. API Key；
2. Token 统计；
3. 额度；
4. 成本和计费；
5. SSO；
6. LDAP；
7. OAuth；
8. 用户自定义 Permission code；
9. 资源级 ACL 和字段级权限；
10. 多级组织权限继承；
11. tenant switch API/UI；
12. 用户邀请；
13. 租户配额；
14. 租户账单；
15. 租户隔离调度。

---

## 4. Phase 1：Agent 注册与心跳

目标：

1. Agent 能注册到 Server；
2. Server 能保存节点；
3. Agent 能周期性心跳；
4. Server 能判断节点在线 / 离线；
5. Agent 注册时能上报 advertised address 和受控 metrics 字段；
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
10. Server 保存 `advertised_address`、metrics scheme/port/path/enabled 和版本字段；
11. Server 根据已注册、未删除、metrics enabled 且地址有效的节点生成 `/metrics/targets`；
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

curl http://127.0.0.1:18080/api/nodes
curl http://127.0.0.1:18080/metrics/targets
```

预期：

1. `/api/nodes` 可以看到节点；
2. 节点状态为 online；
3. 节点包含受控 metrics 地址字段；
4. `/metrics/targets` 返回 Agent target；
5. 停止 Agent 后节点变为 offline；
6. Agent 重启后可以重新注册或恢复心跳。

---

## 5. Phase 2：OS / NVIDIA / MetaX 资源采集与上报

目标：

1. Agent 能采集操作系统资源；
2. Agent 能采集 CPU、内存、Swap、磁盘、网络基础信息；
3. Agent 能通过 NvidiaCollector 发现并采集 NVIDIA GPU；
4. Agent 能通过 MetaxCollector 发现并采集 MetaX GPU；
5. Agent 能把资源状态上报 Server；
6. Server 能保存节点、OS 和 GPU 最新状态；
7. Server 能保存 Collector 诊断信息；
8. Server 能提供资源查询 API；
9. Agent `/metrics` 能暴露 OS 和 GPU 指标；
10. Server `/metrics/targets` 能返回 Agent targets。

Phase 2A：System / Registry / Mock

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
10. 实现 CollectorRegistry；
11. 实现 `gpu.profile=production/development/test`；
12. 实现仅用于 development/test 且默认关闭的 MockGPUCollector；
13. 实现资源上报 API；
14. Agent 周期性上报资源；
15. Server 保存最新 OS、GPU 和诊断状态；
16. 提供节点详情和 GPU 查询 API；
17. 更新 Agent `/metrics`；
18. 更新 Server `/metrics/targets`；
19. 区分成功、失败和成功空列表；
20. Collector 失败保留旧状态；
21. Server 拒绝旧 `collected_at` 覆盖新状态。

Phase 2A 验收：

```text
Agent 可以启动
SystemCollector 能采集 OS 资源
development/test profile 可显式启用 MockGPUCollector
production profile 禁止 Mock
Agent /metrics 可看到 OS / GPU 指标
Server 能看到节点 CPU / 内存 / 磁盘 / GPU
/metrics/targets 能返回 Agent metrics 地址
Collector 成功空列表会更新为空设备事实
Collector 失败或旧报告不会覆盖较新成功状态
```

Phase 2B：NVIDIA

任务：

1. 按脱敏真实样例实现 NvidiaCollector；
2. 处理命令不存在、超时、非零退出和部分字段缺失；
3. 将 NVIDIA MB 转为 bytes；
4. API/DB 保存 `0-100` percent；
5. Prometheus 导出 `0-1` ratio；
6. 增加样例解析和单位转换测试；
7. 在真实 NVIDIA 环境验收。

Phase 2B 验收：

1. Agent `/metrics` 可看到 NVIDIA 指标；
2. Server GPU API 返回 bytes 和 percent；
3. diagnostics 显示 NvidiaCollector 可用；
4. 工具失败不导致 Agent 崩溃。

Phase 2C：MetaX

任务：

1. 在测试环境确认 MetaX 工具、版本、命令和机器可读格式；
2. 按 `docs/vendor-samples/README.md` 保存脱敏样例；
3. 基于样例实现 MetaxCollector；
4. 将结果映射到统一 bytes/percent/nil 数据结构；
5. 处理命令不存在、超时、非零退出和部分字段缺失；
6. 增加样例解析测试；
7. 在真实 MetaX 环境验收。

Phase 2C 验收：

1. Agent `/metrics` 可看到 MetaX 指标；
2. Server GPU API 返回统一字段；
3. diagnostics 显示 MetaxCollector 可用；
4. 不支持的指标为 unknown/nil，不伪造。

Phase 2 完成标准：Phase 2A、2B、2C 全部通过。Mock 不能替代任一真实设备验收。

验收命令：

```bash
go run ./cmd/server
go run ./cmd/agent

curl http://127.0.0.1:18080/api/nodes
curl http://127.0.0.1:18080/api/nodes/<node_id>
curl http://127.0.0.1:18080/api/gpus
curl http://127.0.0.1:18080/metrics
curl http://127.0.0.1:18080/metrics/targets
```

---

## 6. Phase 3：运行环境管理

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
7. 防止删除已被实例引用的运行环境；
8. Preview 和任务创建复用 Server DockerRunSpec 生成器；
9. 禁止 shell 字符串和 shell 片段式 ExtraArgs；
10. 查询按 current tenant 过滤；
11. 创建时由 Session 写入 tenant、owner 和 audit 字段；
12. 增加 `runtime:read` / `runtime:write` permission 校验和跨 tenant 拒绝。

完成标准：

```text
可以创建一个 vLLM Docker 运行环境
可以配置镜像、命令、环境变量、volume、device、shm-size
未启用参数不会出现在命令预览中
```

---

## 7. Phase 4：模型定义管理

目标：

1. 可以创建模型定义；
2. 可以关联默认运行环境；
3. 模型可以被实例引用。

任务：

1. 定义 Model；
2. 实现模型 CRUD；
3. 支持 `model_container_path` 和可选 `model_host_path`；
4. 支持默认端口；
5. 支持默认上下文长度；
6. 支持默认启动参数；
7. 支持 runtime metrics endpoint 预留字段；
8. 防止删除已被实例引用的模型；
9. 查询按 current tenant 过滤；
10. 创建时由 Session 写入 tenant、owner 和 audit 字段；
11. 增加 `model:read` / `model:write` permission 校验和跨 tenant 引用拒绝。

完成标准：

```text
可以创建 qwen、deepseek 等模型定义
模型定义可以选择默认运行环境
实例创建时可以选择模型
```

---

## 8. Phase 5：模型实例创建与任务下发

目标：

1. 可以创建模型实例；
2. Server 能生成任务；
3. Agent 能拉取任务；
4. Agent 能回报任务结果；
5. Server 能生成并冻结 DockerRunSpec；
6. 任务具备 claim、lease、attempt 和幂等语义；
7. 同一实例只允许一个 active operation。

任务：

1. 定义 ModelInstance；
2. 定义 AgentTask；
3. 定义 TaskResult；
4. 实现实例创建 API；
5. 实现任务表；
6. 实现原子任务 claim API；
7. 实现任务回报 API；
8. Agent 实现任务轮询；
9. Server 更新任务状态；
10. 实例对象预留 runtime metrics endpoint 字段；
11. AgentTask 增加 operation、generation、attempt、lease 字段；
12. ModelInstance 增加 active operation 和 generation 字段；
13. StartInstanceTask 携带冻结 DockerRunSpec；
14. 实现重复结果和旧 attempt 拒绝规则；
15. 实例查询和操作按 current tenant 过滤；
16. 创建时由 Session 写入 tenant、owner 和 audit 字段；
17. AgentTask 写入 tenant_id、created_by 和 updated_by；
18. Server 创建任务前完成 `instance:write` / `instance:operate` permission 校验；
19. Agent 不执行用户角色判断。

完成标准：

```text
创建实例后生成 start_instance 任务
Agent 可以拉取任务
Agent 可以回报成功或失败
任务状态可查询
lease 过期任务可在 max_attempts 内重新 claim
重复回报不重复改变实例
```

---

## 9. Phase 6：Docker 启停实例

目标：

1. Agent 可以执行 docker run；
2. Agent 可以执行 docker stop；
3. Agent 可以回报容器 ID；
4. Server 可以更新实例状态。

任务：

1. Server 实现并复用 DockerRunSpec 生成；
2. 实现 docker run；
3. 实现 docker stop；
4. 实现 docker inspect；
5. 记录 Docker 命令快照；
6. 记录 stdout / stderr；
7. 回报容器 ID；
8. 更新实例状态；
9. 启动失败时记录 last_error；
10. Agent 只校验和执行冻结规格；
11. 写入 Docker ownership labels；
12. restart 固定执行 stop、remove、start；
13. start/stop/restart 操作幂等；
14. ownership labels 写入 tenant_id 和 created_by。

完成标准：

```text
可以从平台启动一个容器
可以停止容器
启动失败时可以看到错误
实例状态能从 starting 进入 running 或 failed
```

---

## 10. Phase 7：实例健康检查与 endpoint

目标：

1. Agent 能检查容器状态；
2. Agent 能检查端口；
3. Server 能展示 endpoint；
4. 页面或 API 能看到实例健康状态；
5. 后续可以把实例健康状态暴露到 Prometheus；
6. 旧 generation、旧时间和冲突 operation 不覆盖当前状态；
7. Agent 重启后能基于 ownership labels reconciliation。

任务：

1. Agent 定期 docker inspect；
2. Agent 检查端口可达；
3. Agent 上报实例状态；
4. Server 保存 endpoint；
5. Server 展示 last_error；
6. Server 展示 last_checked_at；
7. 提供实例详情 API；
8. Agent `/metrics` 预留实例健康指标；
9. 状态报告携带 task、operation、generation、checked_at；
10. Server 实现状态报告接收规则；
11. Agent 启动扫描受管容器并回报 running/stopped/failed/unknown；
12. Docker 不可用时回报 unknown。

完成标准：

```text
容器退出后实例状态变更
端口不可达时健康检查失败
实例详情中可看到 endpoint、状态、错误和最后检查时间
```

---

## 11. Phase 8：基础 Web 页面

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
11. 监控看板；
12. 登录；
13. 用户和 Tenant 管理；
14. Membership 管理。

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
登录、退出和首次强制改密
platform admin 可以管理全局 User 和 Tenant
tenant admin 可以管理当前 Tenant 的 Membership 和 custom Role
built-in/custom Role 的页面操作与实时 permission code 一致
```

---

## 12. Phase 9：Prometheus / Grafana 内置部署

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

## 13. 第二阶段入口

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
