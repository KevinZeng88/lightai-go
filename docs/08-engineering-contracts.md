> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference or historical compatibility document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go 统一工程契约

## 1. 适用范围与优先级

本文定义 LightAI Go 第一阶段跨 Server、Agent、资源监控、Observability、运行环境、模型、实例生命周期、身份、租户和 RBAC 的统一工程契约。

当 `docs/01-architecture.md` 至 `docs/07-instance-lifecycle-design.md`、`docs/10-mvp-development-plan.md` 与本文冲突时，以本文为准。专题文档可以补充实现细节，但不得改变本文定义的单位、字段语义、职责边界、合并算法和可靠性规则。

当前开发窗口为 Phase 0、Phase 0.5、Phase 1、Phase 2。Phase 2 必须打通 System、NVIDIA、MetaX 三条真实采集链路；Mock 只用于开发和自动化测试。

---

## 2. 数据单位与空值

统一规则：

1. API、数据库和 Go 结构中的容量统一使用 bytes，字段以 `_bytes` 或 `Bytes` 结尾；
2. API、数据库和 Collector 中的百分比统一使用 `0-100`，字段以 `_percent` 或 `Percent` 结尾；
3. Prometheus 中的比例统一使用 `0-1`，指标以 `_ratio` 结尾；
4. 温度使用摄氏度，功耗使用瓦，持续时间使用秒；
5. 厂商工具输出的 MB 必须在 Collector 内转换为 bytes，换算规则为 `bytes = MB * 1024 * 1024`；
6. 厂商工具输出的百分比必须在导出 Prometheus 指标时除以 100；
7. 缺失、不可解析或厂商不支持的指标使用 `nil`、nullable 字段或 `unknown` 状态，禁止填充伪造的 0；
8. 所有采集快照和指标必须携带 `collected_at`。

---

## 3. Node Metrics 地址与动态发现

Agent 注册时只上报受控字段，不允许上报任意完整 URL：

```text
advertised_address
metrics_enabled
metrics_scheme
metrics_port
metrics_path
agent_version
protocol_version
```

约束：

1. `advertised_address` 只能是 Server 可验证的主机名或 IP，不包含 scheme、port、path、query 或 fragment；
2. `metrics_scheme` 第一阶段只允许 `http` 或 `https`；
3. `metrics_port` 必须为合法 TCP 端口；
4. `metrics_path` 必须是以 `/` 开头的规范路径，不允许 query、fragment 或路径穿越；
5. Server 校验并保存这些字段，再生成 Prometheus HTTP SD target；
6. Node 不保存 Agent 提供的任意完整 metrics URL。

`GET /metrics/targets` 返回所有满足以下条件的节点：

1. 已注册；
2. 未删除；
3. `metrics_enabled=true`；
4. metrics scheme、address、port、path 均有效。

动态发现不以节点业务状态 `online/offline` 为过滤条件。节点暂时离线时仍保留 target，由 Prometheus 自己记录 scrape 失败。Server 使用 Prometheus 内部 labels `__scheme__` 和 `__metrics_path__` 表达受控 scheme/path；业务 labels 只允许稳定、低基数字段，例如 `job`、`node_id`、`node_name`、`vendor`。

---

## 4. Observability 三模式

| 能力 | builtin | external | disabled |
| --- | --- | --- | --- |
| Server `/metrics` | 开启 | 开启 | 可关闭 |
| Agent `/metrics` | 按节点配置开启 | 按节点配置开启 | 默认关闭 |
| Server `/metrics/targets` | 开启 | 开启 | 可关闭或返回空数组 |
| 平台托管 Prometheus/Grafana | 是 | 否 | 否 |
| 外部 Prometheus/Grafana | 可选 | 由客户管理 | 不使用 |
| Web 监控入口 | 默认“打开 Grafana”链接 | 配置外部链接 | 隐藏历史趋势入口 |
| iframe | 仅同源代理或明确启用匿名 Viewer 时 | 取决于客户配置 | 不启用 |

默认使用 HTTP SD。file SD 只作为替代部署示例，不得与 HTTP SD 同时作为默认配置启用。匿名 Viewer、Grafana `3000:3000` 端口映射只适用于开发或可信内网。

---

## 5. Collector Profile 与结果语义

Agent 使用显式 `gpu.profile` 隔离运行环境：

```text
production  启用真实 Collector，禁止 Mock
development 允许显式启用 Mock
test        允许测试夹具和 Mock
```

默认 profile 为 `production`，Mock 默认关闭。生产 profile 中即使误配 `mock.enabled=true` 也必须拒绝启动 Mock Collector，并产生配置诊断。

每个 Collector 的一次执行只有三类结果：

1. 成功且有设备：用本次设备和指标更新当前状态；
2. 成功且设备列表为空：这是有效事实，清空该 Collector 本次负责的当前设备集合或按状态策略标记为不存在，不保留伪造设备；
3. 失败：保存诊断，但不使用空结果覆盖上一次成功状态。

Server 仅接受不早于当前记录的 `collected_at`。旧时间戳、乱序重试或重复上报不得覆盖较新的设备、指标和诊断状态。

---

## 6. DockerRunSpec 职责与结构

Server 是 DockerRunSpec 的唯一生成方。Server 在创建 start/restart operation 时：

1. 读取 RuntimeEnvironment、Model、ModelInstance、Node 和 GPU 选择；
2. 使用统一生成器完成校验与合并；
3. 生成结构化 DockerRunSpec；
4. 写入 ownership labels；
5. 冻结规格并随任务下发。

Preview API 和任务创建必须复用同一个生成器。Agent 不重新合并业务对象，不自行补参数；Agent 只校验冻结规格是否可执行，然后通过 Docker API 或参数数组执行。

DockerRunSpec 至少包含：

```go
type DockerRunSpec struct {
    Image       string
    Name        string
    Entrypoint  []string
    Command     []string
    Args        []string
    WorkingDir  string
    Env         []EnvVar
    Ports       []PortMapping
    Volumes     []VolumeMount
    Devices     []DeviceMapping
    GpuPolicy   string
    GpuDeviceIDs []string
    ExtraArgs   []string
    Labels      map[string]string
}
```

Runtime 决定镜像、entrypoint 和 command。Model 提供默认 `Args`。Instance 启用 args override 时整体替换 Model args，不做参数名级合并。所有字段必须保持结构化，禁止将整条命令或 shell 片段作为字符串执行。

---

## 7. 参数合并算法

输入优先级为 RuntimeEnvironment、Model、ModelInstance，后者只在本文明确允许覆盖时生效。

1. `Entrypoint`、`Command`：只来自 RuntimeEnvironment；未启用则为空；
2. `Args`：Model 默认 args 为基线；Instance args override 启用时整体替换，空数组表示明确清空；
3. `Env`：以 key 为身份，按 Runtime、Model、Instance 顺序覆盖；保留首次出现顺序，新 key 追加到末尾；
4. `Volumes`：以 `container_path` 为身份，按 Runtime、Model 自动挂载、Instance 顺序覆盖；同一层重复 key 直接报错；
5. `Devices`：以非空 `container_path` 为身份，否则使用 `host_path`，按 Runtime、Instance 顺序覆盖；同一层重复 key 直接报错；
6. `Ports`：以 `container_port/protocol` 为身份，按 Runtime、Model 默认端口、Instance 顺序覆盖；最终 host port 冲突时报错；
7. `ExtraArgs`：Runtime 提供基线；Instance 显式启用 extra args override 时整体替换，禁止 shell 运算符和未拆分的 shell 片段；
8. GPU：Runtime 只保存 `none/all/selected/vendor_specific` 策略；具体 GPU ID 来自 Instance，只有 `selected` 策略使用；
9. 输出必须稳定，同一输入生成字节等价的规范 JSON 和等价 Preview。

---

## 8. Model 路径语义

Model 使用两个不同字段：

```text
model_container_path  必填，模型在容器内被命令参数引用的路径
model_host_path       可选，模型在目标节点宿主机上的路径
```

唯一语义：

1. `model_container_path` 不能被解释为宿主机路径；
2. `model_host_path` 不能直接出现在容器命令参数中；
3. 当 `model_host_path` 非空时，Server 可生成一个明确的只读 volume：`model_host_path:model_container_path:ro`；
4. 推荐优先由 Runtime 或 Instance 显式配置 volume，自动挂载只用于简单场景；
5. 自动挂载与显式 volume 的 `container_path` 冲突时，Instance 显式配置优先，其他同层冲突报错；
6. `DefaultContextLen` 只作为元数据或生成默认 args 的输入；如果 args 已包含等价上下文参数，不得重复注入。

---

## 9. AgentTask Claim、Lease 与幂等

任务拉取必须是数据库原子 claim，不是先查询再更新。AgentTask 至少包含：

```text
task_id
node_id
instance_id
task_type
operation_id
spec_generation
status
attempt
max_attempts
lease_owner
lease_expires_at
payload
created_at
started_at
finished_at
```

规则：

1. claim 时原子地把 `pending` 或 lease 已过期的可重试任务更新为 `running`；
2. `attempt` 每次成功 claim 后加一；
3. Agent 只能续租和回报自己持有 lease 的任务；
4. 结果回报以 `task_id + operation_id + attempt` 幂等；
5. 重复成功回报返回已接受结果，不重复改变实例；
6. 旧 attempt、lease owner 不匹配或已被新 operation 取代的结果拒绝；
7. 达到 `max_attempts` 后任务失败，不自动无限重试；
8. 同一实例任何时刻只允许一个 active operation。

---

## 10. Operation、Generation 与状态报告

ModelInstance 至少保存：

```text
active_operation_id
active_operation_type
spec_generation
last_observed_generation
last_checked_at
```

每次 start、stop、restart 创建新的 `operation_id`。会改变期望运行规格的操作递增 `spec_generation`。Agent 的任务结果和周期状态报告必须携带：

```text
task_id
operation_id
generation
checked_at
```

Server 接收规则：

1. generation 小于实例当前 `spec_generation`，拒绝；
2. generation 小于 `last_observed_generation`，拒绝；
3. `checked_at` 早于当前 `last_checked_at`，拒绝；
4. operation 与当前 active operation 冲突，拒绝；
5. 接受终态结果后以条件更新清除对应 active operation；
6. 相同 operation 的重复终态报告幂等接受。

start：目标容器已按相同 generation 运行时视为成功。stop：容器不存在或已停止时视为成功。restart：固定执行 stop、remove、start，不提供保留旧容器的可选策略。

---

## 11. Docker Ownership 与重启 Reconciliation

所有受管容器必须包含：

```text
lightai.managed=true
lightai.instance_id=<instance_id>
lightai.node_id=<node_id>
lightai.operation_id=<operation_id>
lightai.spec_generation=<generation>
```

Agent 启动和重新连接 Server 后扫描这些 labels，并按以下规则回报：

1. 容器运行：inspect 并回报 `running` 及 health；
2. 容器已退出：最新操作为 stop 时回报 `stopped`，否则回报 `failed`；
3. 容器缺失且实例原状态为 `created/stopped`：回报 `stopped`；
4. 容器缺失且期望运行，或存在 start/restart operation：回报 `failed`；
5. Docker 不可用、labels 冲突或无法可靠判断：回报 `unknown`；
6. Agent 不接管没有 `lightai.managed=true` 的容器；
7. generation 与 operation 仍按第 10 节规则校验。

---

## 12. 厂商样例与脱敏

NVIDIA 和 MetaX Collector 的解析器必须基于测试环境采集的真实样例开发。采样流程和建议路径见 `docs/vendor-samples/README.md`。

要求：

1. 不提交未脱敏的真实输出；
2. 删除主机名、IP、序列号、UUID、资产编号、用户名和客户目录等信息；
3. 保留字段结构、分隔符、空值、错误样例和版本信息；
4. 优先使用厂商提供的机器可读格式；
5. MetaX 工具名称、参数和输出格式以测试环境样例为准，文档不得猜测；
6. 缺失指标使用 unknown/nil，禁止根据其他字段推算或伪造。

---

## 13. 身份、租户与资源归属契约

详细设计见 `docs/09-auth-tenant-design.md`。跨模块必须遵守：

1. 第一阶段正式实现 Tenant、User、TenantMembership、TenantMembershipRole、Role、Permission、RolePermission 和 Session；
2. 普通用户请求不能通过 query、body 或 header 指定数据范围，tenant context 只来自 `session.current_tenant_id`；
3. TenantMembership 不保存角色字段；一个 Membership 通过 TenantMembershipRole 绑定一个或多个 Role；
4. built-in Role 全局只读且 `tenant_id=null`，包括 admin/operator/viewer；
5. custom Role 必须绑定 tenant_id，只能在当前 Tenant 管理和分配；
6. Permission catalog 系统只读，由系统初始化和版本迁移维护，用户不能定义 permission code；
7. API 统一按 required permission code 授权，不按角色名称判断；
8. Session 只保存 user、current tenant 和过期/撤销等最小上下文；每次请求实时解析 Membership、Roles 和 Permissions；
9. `User.IsPlatformAdmin` 用于跨租户 User / Tenant 管理；built-in admin 只表示 tenant admin，不等于 platform admin；
10. Tenant、User 和 Membership 禁用优先，第一阶段不硬删除；
11. built-in Role 不可修改或删除；custom Role 未被分配时可删除，被分配时必须先解绑；
12. Agent API 只使用 bootstrap/shared agent token，不使用 User Session，不产生 user_id；
13. Future API Key 只用于后续模型服务调用，不用于 Agent 或用户管理登录；
14. 用户创建的 RuntimeEnvironment、Model、ModelInstance 必须写入：

```text
tenant_id = session.current_tenant_id
owner_id = session.current_user_id
created_by = session.current_user_id
updated_by = session.current_user_id
```

15. 客户端提交的 tenant、owner 和审计字段不能覆盖 Server 计算结果；
16. owner_id 是可转移业务所有者，不等于 created_by，不作为 ACL；授权仍由 tenant scope 和 permission code 决定；
17. Node 是系统发现资源，默认 `tenant_id=default`、`owner_id=null`、`created_by/updated_by=system`；
18. GPUDevice 跟随 Node，`owner_id=null`、`created_by/updated_by=system`；
19. AgentTask 的 tenant_id 来自关联实例或已授权操作，created_by 为触发用户或 system；
20. Server 创建 AgentTask 前完成 Session、tenant scope 和 permission code 授权；Agent 只校验任务属于本节点，并继续执行 lease、attempt、operation 和 generation 规则；
21. 受管容器 ownership labels 在第 11 节基础上增加：

```text
lightai.tenant_id=<tenant_id>
lightai.created_by=<user_id-or-system>
```

22. 日志应尽量记录 tenant_id、user_id、role_ids、required_permission、operation、resource 和 result，但不得记录密码、Session ID、CSRF secret、Agent token 或 Future API Key 明文；
23. User Session 使用 12 小时滑动过期，并保持 CSRF token、Origin 校验、Argon2id 和登录限流；
24. default tenant 是普通 Tenant 记录，不允许通过特殊实现阻止未来增加其他 Tenant；
25. 身份契约不得改变 metrics、Collector、DockerRunSpec、task lease、generation 和 reconciliation 规则。

---

## 14. 验证与变更纪律

跨契约变更必须同步检查：

1. `01/02/03/10` 对 Phase 2 范围一致；
2. `02/03/04` 对 metrics target 字段和发现条件一致；
3. `05/06/07` 对 DockerRunSpec、路径和参数合并一致；
4. `07/10` 对 task lease、operation、generation 和 reconciliation 一致；
5. API/DB bytes、percent 与 Prometheus ratio 不混用；
6. Mock 不进入 production profile；
7. NVIDIA 和 MetaX 均有真实环境验收证据；
8. `00/01/02/05/06/07/09/10` 对 tenant、owner、audit 和凭证边界一致；
9. 身份契约不得改变 metrics、Collector、DockerRunSpec、task lease、generation 和 reconciliation 规则。

修改实现前应先更新本文或确认本文无需变化；实现和测试不得引入第二套隐含契约。
