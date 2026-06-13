# LightAI Go 模型实例生命周期设计

## 1. 设计目标

模型实例是 LightAI Go 第一阶段最关键的业务对象。

模型定义描述“运行什么模型”。
运行环境描述“怎么运行”。
模型实例描述“在哪台节点、使用哪些 GPU、以哪些参数启动一个实际服务”。

第一阶段目标：

1. 可以创建模型实例；
2. 可以指定模型；
3. 可以指定运行环境；
4. 可以指定节点；
5. 可以指定 GPU；
6. 可以指定端口；
7. 可以启动实例；
8. 可以停止实例；
9. 可以重启实例；
10. 可以查看实例状态；
11. 可以查看 endpoint；
12. 可以查看 Docker 命令快照；
13. 可以查看最近错误；
14. 可以通过 Agent 定期回报实例状态；
15. 容器异常退出后，Server 能反映状态变化。

---

## 2. 实例与模型、运行环境、节点的关系

```text
ModelInstance
  ↓ references
Model
RuntimeEnvironment
Node
GPUDevice
```

一个实例必须关联：

1. 一个 Model；
2. 一个 RuntimeEnvironment；
3. 一个 Node；
4. 零个或多个 GPUDevice。

GPU 可以为空，表示 CPU-only 或不绑定 GPU。
第一阶段主要面向 GPU 模型，Web 可以默认要求选择 GPU。

---

## 3. ModelInstance 数据结构

```go
type ModelInstance struct {
    ID          string
    Name        string
    Description string

    ModelID     string
    RuntimeID   string
    NodeID      string

    GPUDeviceIDs []string

    ContainerName string
    ContainerID   string

    HostPort      int
    ContainerPort int
    Endpoint      string

    Status        string
    HealthStatus  string
    LastError     string

    DockerRunSpecJSON string
    DockerCommand     string

    RuntimeMetricsEnabled bool
    RuntimeMetricsURL     string

    CreatedAt     time.Time
    UpdatedAt     time.Time
    StartedAt     *time.Time
    StoppedAt     *time.Time
    LastCheckedAt *time.Time
}
```

---

## 4. 实例状态

实例状态建议：

```text
created
pending
starting
running
stopping
stopped
failed
unknown
```

含义：

| 状态       | 含义                       |
| -------- | ------------------------ |
| created  | 已创建，但未启动                 |
| pending  | 已提交启动 / 停止任务，等待 Agent 拉取 |
| starting | Agent 正在启动容器             |
| running  | 容器运行且基础健康检查通过            |
| stopping | Agent 正在停止容器             |
| stopped  | 已停止                      |
| failed   | 启动、停止或运行检查失败             |
| unknown  | Agent 离线或状态暂时不可确认        |

第一阶段不做复杂状态机，但状态转换必须清晰。

---

## 5. 健康状态

健康状态建议：

```text
healthy
unhealthy
unknown
```

含义：

1. healthy：容器运行，端口检查通过；
2. unhealthy：容器异常退出或端口不可达；
3. unknown：Agent 未上报或节点离线。

实例状态和健康状态分开：

```text
status = running
health_status = unhealthy
```

表示容器可能在运行，但服务端口不可用。

---

## 6. 状态转换

### 6.1 创建实例

```text
无实例
  ↓
created
```

### 6.2 启动实例成功

```text
created / stopped / failed
  ↓ 用户点击启动
pending
  ↓ Agent 拉取任务
starting
  ↓ docker run 成功
running
```

### 6.3 启动实例失败

```text
created / stopped / failed
  ↓ 用户点击启动
pending
  ↓ Agent 拉取任务
starting
  ↓ docker run 失败
failed
```

### 6.4 停止实例成功

```text
running / failed / unknown
  ↓ 用户点击停止
pending
  ↓ Agent 拉取任务
stopping
  ↓ docker stop 成功
stopped
```

### 6.5 容器异常退出

```text
running
  ↓ Agent 定期检查
failed / stopped
```

具体进入 failed 还是 stopped：

1. 用户主动停止，进入 stopped；
2. 非用户主动停止，进入 failed；
3. Agent 无法判断原因时，进入 unknown 或 failed。

### 6.6 Agent 离线

```text
running
  ↓ Agent 心跳超时
unknown
```

Agent 恢复后：

```text
unknown
  ↓ Agent inspect 容器运行
running

unknown
  ↓ Agent inspect 容器不存在
failed / stopped
```

---

## 7. 实例创建流程

```text
用户填写实例信息
  ↓
Server 校验模型、运行环境、节点、GPU
  ↓
Server 创建 ModelInstance
  ↓
状态为 created
```

创建实例时不自动启动，除非用户明确选择“创建后立即启动”。

第一阶段建议：

```text
创建实例
启动实例
```

两个动作分开，便于排障。

---

## 8. 实例启动流程

```text
用户点击启动
  ↓
Server 校验实例状态
  ↓
Server 创建 start_instance task
  ↓
实例状态改为 pending
  ↓
Agent 拉取任务
  ↓
Agent 将任务状态改为 running
  ↓
实例状态改为 starting
  ↓
Agent 生成 DockerRunSpec
  ↓
Agent 执行 docker run
  ↓
Agent inspect 容器
  ↓
Agent 回报任务结果
  ↓
Server 更新实例状态
```

启动成功后 Server 保存：

1. container_id；
2. container_name；
3. endpoint；
4. docker_run_spec_json；
5. docker_command；
6. started_at；
7. last_error 清空。

启动失败后 Server 保存：

1. status = failed；
2. last_error；
3. docker_command；
4. task result；
5. stderr 摘要。

---

## 9. 实例停止流程

```text
用户点击停止
  ↓
Server 校验实例状态
  ↓
Server 创建 stop_instance task
  ↓
实例状态改为 pending 或 stopping
  ↓
Agent 拉取任务
  ↓
Agent 执行 docker stop
  ↓
Agent 回报任务结果
  ↓
Server 更新实例状态 stopped
```

停止成功后：

1. status = stopped；
2. stopped_at 更新；
3. health_status = unknown；
4. endpoint 可保留但标记不可用；
5. container_id 可保留用于历史排障。

---

## 10. 实例重启流程

重启可以拆成：

```text
stop_instance
start_instance
```

第一阶段建议 Server 创建一个 `restart_instance` 任务，由 Agent 内部执行：

```text
docker stop
docker rm 可选
docker run
```

是否删除旧容器由配置决定：

```text
remove_before_restart = true / false
```

第一阶段建议默认：

```text
remove_before_restart = true
```

避免同名容器冲突。

---

## 11. Agent 任务类型

实例生命周期相关任务：

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

任务中必须包含：

1. task_id；
2. instance_id；
3. node_id；
4. task_type；
5. payload；
6. created_at。

---

## 12. StartInstanceTask Payload

```json
{
  "instance_id": "inst-001",
  "model_id": "model-001",
  "runtime_id": "runtime-001",
  "node_id": "node-001",
  "gpu_device_ids": ["gpu-001", "gpu-002"],
  "host_port": 8001,
  "container_port": 8000
}
```

Agent 拉取任务后，需要根据 Server 返回的信息或本地缓存生成 DockerRunSpec。

第一阶段建议任务 payload 中包含足够信息，避免 Agent 再频繁查 Server。

---

## 13. TaskResult

```go
type TaskResult struct {
    TaskID     string
    AgentID    string
    Status     string
    Message    string
    Error      string
    Result     map[string]any
    StartedAt  time.Time
    FinishedAt time.Time
}
```

启动成功 result：

```json
{
  "container_id": "abc123",
  "container_name": "lightai-inst-001",
  "endpoint": "http://192.168.1.10:8001",
  "docker_command": "docker run ...",
  "runtime_metrics_url": "http://192.168.1.10:8001/metrics"
}
```

启动失败 result：

```json
{
  "docker_command": "docker run ...",
  "stderr": "image not found"
}
```

stderr 需要截断，避免过长。

---

## 14. 实例状态回报

Agent 定期回报实例状态：

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
      "container_id": "abc123",
      "status": "running",
      "health_status": "healthy",
      "endpoint": "http://192.168.1.10:8001",
      "runtime_metrics_url": "http://192.168.1.10:8001/metrics",
      "last_error": "",
      "checked_at": "2026-06-13T10:00:10Z"
    }
  ]
}
```

Server 收到后：

1. 更新实例状态；
2. 更新 health_status；
3. 更新 last_checked_at；
4. 更新 endpoint；
5. 更新 runtime_metrics_url；
6. 保存 last_error。

---

## 15. endpoint 生成规则

endpoint 由 Agent 或 Server 生成。

建议第一阶段由 Agent 回报，因为 Agent 更了解本机 IP 和端口映射。

格式：

```text
http://<node_ip>:<host_port>
```

如果节点有多个 IP，优先使用：

1. Agent 配置中的 advertised_ip；
2. Agent 采集的 primary_ip；
3. Server 看到的 remote_addr。

Agent 配置可预留：

```yaml
agent:
  advertised_ip: ""
```

---

## 16. runtime metrics URL

如果模型服务暴露 `/metrics`，实例可记录：

```text
runtime_metrics_url
```

生成规则：

```text
endpoint + runtime_metrics_path
```

例如：

```text
http://192.168.1.10:8001/metrics
```

第一阶段只记录，不一定采集。

后续 RuntimeCollector 可以抓取这个地址，并转换为统一指标。

---

## 17. Docker 容器命名规则

建议：

```text
lightai-{instance_id}
```

或：

```text
lightai-{safe_instance_name}-{short_id}
```

规则：

1. 名称必须稳定；
2. 避免特殊字符；
3. 避免超长；
4. 重启时可以复用；
5. 冲突时应记录明确错误。

---

## 18. 实例 CRUD API

### 18.1 创建实例

```http
POST /api/model-instances
```

### 18.2 查询实例列表

```http
GET /api/model-instances
```

支持参数：

```text
model_id
node_id
status
health_status
```

### 18.3 查询实例详情

```http
GET /api/model-instances/{id}
```

### 18.4 更新实例

```http
PUT /api/model-instances/{id}
```

只允许在未运行状态下修改关键字段：

1. model_id；
2. runtime_id；
3. node_id；
4. gpu_device_ids；
5. port；
6. command args；
7. env。

running 状态下不允许修改关键启动参数。

### 18.5 删除实例

```http
DELETE /api/model-instances/{id}
```

删除规则：

1. running 状态不允许直接删除；
2. 必须先停止；
3. stopped / failed / created 可以删除；
4. 删除实例不一定删除历史任务记录。

---

## 19. 实例操作 API

### 19.1 启动实例

```http
POST /api/model-instances/{id}/start
```

### 19.2 停止实例

```http
POST /api/model-instances/{id}/stop
```

### 19.3 重启实例

```http
POST /api/model-instances/{id}/restart
```

### 19.4 刷新实例状态

```http
POST /api/model-instances/{id}/refresh
```

---

## 20. Web 页面要求

实例列表展示：

1. 实例名称；
2. 模型；
3. 运行环境；
4. 节点；
5. GPU；
6. 状态；
7. 健康状态；
8. endpoint；
9. 最近错误；
10. 操作按钮。

实例详情展示：

1. 基础信息；
2. 模型信息；
3. 运行环境；
4. 节点和 GPU；
5. Docker 命令快照；
6. endpoint；
7. runtime metrics URL；
8. 最近任务；
9. 最近错误；
10. 状态检查时间。

操作按钮：

```text
启动
停止
重启
刷新状态
复制 endpoint
查看 Docker 命令
```

---

## 21. 安全和防误操作

第一阶段至少做：

1. running 实例不能直接删除；
2. running 实例不能修改关键启动参数；
3. 启动前检查节点在线；
4. 启动前检查运行环境启用；
5. 启动前检查模型启用；
6. 启动前检查 GPU 属于目标节点；
7. 启动前检查端口不为空；
8. 停止前检查实例是否已运行；
9. 重启前记录旧状态。

---

## 22. 日志要求

Server 记录：

1. 实例创建；
2. 实例启动请求；
3. 实例停止请求；
4. 实例重启请求；
5. 任务创建；
6. 任务回报；
7. 实例状态变化；
8. 错误信息。

Agent 记录：

1. 拉取任务；
2. 开始执行实例任务；
3. Docker 命令；
4. Docker stdout；
5. Docker stderr；
6. container_id；
7. endpoint；
8. inspect 结果；
9. 端口检查结果；
10. 回报结果。

---

## 23. 测试要求

至少包含：

1. 实例创建测试；
2. 实例状态转换测试；
3. 启动任务创建测试；
4. 停止任务创建测试；
5. 重启任务创建测试；
6. running 实例禁止删除测试；
7. running 实例禁止修改关键字段测试；
8. 节点离线禁止启动测试；
9. GPU 不属于节点时禁止启动测试；
10. 任务回报更新实例状态测试；
11. 实例状态回报测试。

---

## 24. MVP 完成标准

模型实例生命周期完成后，应达到：

1. 可以创建实例；
2. 可以启动实例；
3. 可以停止实例；
4. 可以重启实例；
5. 可以查看实例状态；
6. 可以查看健康状态；
7. 可以查看 endpoint；
8. 可以查看 Docker 命令快照；
9. 可以查看最近错误；
10. Agent 能回报实例状态；
11. 容器异常退出后状态能变化；
12. 节点离线时实例状态能进入 unknown。

