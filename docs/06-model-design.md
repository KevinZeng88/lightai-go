# LightAI Go 模型定义设计

## 1. 设计目标

模型定义用于描述一个可部署的模型，包括模型名称、模型路径、模型类型、默认运行环境、默认端口、默认启动参数和描述信息。

模型定义不直接等于运行中的服务。
模型定义只是模板，真正运行的是模型实例。

第一阶段目标：

1. 可以创建模型定义；
2. 可以编辑模型定义；
3. 可以删除未被实例引用的模型；
4. 可以选择默认运行环境；
5. 可以配置模型路径；
6. 可以配置默认端口；
7. 可以配置默认上下文长度；
8. 可以配置默认启动参数；
9. 模型可以被实例引用；
10. 删除模型时保护已有实例。

---

## 2. 模型、运行环境、实例关系

三者关系：

```text
RuntimeEnvironment：怎么运行
Model：运行什么模型
ModelInstance：在哪台机器、用哪些 GPU、以什么参数运行
```

示例：

```text
RuntimeEnvironment:
  vLLM Docker Runtime

Model:
  Qwen3-32B
  container path: /models/Qwen3-32B
  host path: /data/models/Qwen3-32B
  default port: 8000

ModelInstance:
  qwen3-32b-node01-gpu0
  node: node-001
  gpu: 0
  host port: 8001
```

模型定义不保存容器 ID。
容器 ID 属于模型实例。

---

## 3. Model 数据结构

```go
type Model struct {
    ID          string
    TenantID    string
    OwnerID     string
    CreatedBy   string
    UpdatedBy   string
    Name        string
    DisplayName string
    Description string

    ModelType          string
    ModelContainerPath string
    ModelHostPath      string

    DefaultRuntimeID string

    DefaultPort       int
    DefaultContextLen int

    DefaultCommandArgsEnabled bool
    DefaultCommandArgs        []string

    DefaultEnvEnabled bool
    DefaultEnv        []EnvVar

    RuntimeMetricsEnabled bool
    RuntimeMetricsPath    string

    Enabled     bool

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## 4. 字段说明

### 4.1 Name

平台内部模型名称。

要求：

1. 必填；
2. 唯一；
3. 建议使用小写、数字、横线；
4. 不建议使用空格；
5. 后续可以作为 API model name。

示例：

```text
qwen3-32b
deepseek-r1-32b
glm-4-9b
```

### 4.2 DisplayName

展示名称。

示例：

```text
Qwen3 32B
DeepSeek R1 32B
GLM-4 9B
```

### 4.3 ModelType

模型类型。

建议枚举：

```text
llm
embedding
reranker
multimodal
audio
custom
```

第一阶段重点支持：

```text
llm
embedding
custom
```

### 4.4 ModelContainerPath 与 ModelHostPath

模型路径拆分为：

```text
model_container_path  必填，容器内路径
model_host_path       可选，目标节点宿主机路径
```

示例：

```text
model_container_path: /models/Qwen3-32B
model_host_path: /data/models/Qwen3-32B
```

规则：

1. `model_container_path` 只表示容器内路径；
2. `model_host_path` 只表示宿主机路径，不能直接用于容器命令；
3. `model_host_path` 非空时，Server 可生成到 `model_container_path` 的明确只读 volume；
4. 默认仍推荐 Runtime 或 Instance 显式配置 volume；
5. 第一阶段不做自动下载；
6. 第一阶段不校验所有 Agent 上路径是否存在；
7. 实例启动时由 Agent 校验冻结规格中的宿主机挂载路径。

### 4.5 DefaultRuntimeID

默认运行环境 ID。

一个模型可以有默认运行环境，但创建实例时可以覆盖。

规则：

1. 可以为空；
2. 如果填写，必须引用存在且启用的 RuntimeEnvironment；
3. 删除运行环境时，如果被模型引用，应阻止删除。

### 4.6 DefaultPort

默认容器端口。

常见：

```text
8000
8080
11434
```

创建实例时可覆盖。

### 4.7 DefaultContextLen

默认上下文长度。

示例：

```text
32768
65536
131072
```

第一阶段仅作为元数据或默认参数生成来源，不参与容量计算。如果默认 args 已包含等价上下文参数，Server 不得重复注入。

### 4.8 DefaultCommandArgs

模型默认启动参数。

示例：

```text
--model /models/Qwen3-32B
--served-model-name qwen3-32b
--max-model-len 32768
```

只有 `DefaultCommandArgsEnabled=true` 时才参与命令生成。

### 4.9 DefaultEnv

模型默认环境变量。

示例：

```text
VLLM_USE_MODELSCOPE=true
TRUST_REMOTE_CODE=true
```

只有 `DefaultEnvEnabled=true` 时才参与命令生成。

### 4.10 Runtime Metrics

为后续 RuntimeCollector 预留。

字段：

```text
RuntimeMetricsEnabled
RuntimeMetricsPath
```

示例：

```text
/metrics
```

第一阶段不采集模型 runtime metrics，但模型和实例对象需要预留字段。

---

## 5. 模型状态

模型状态可以简单处理为：

```text
enabled
disabled
```

字段：

```go
Enabled bool
```

含义：

1. enabled：可以创建新实例；
2. disabled：不允许创建新实例，但不影响已运行实例；
3. disabled 不等于删除；
4. 删除需要满足无实例引用。

---

## 6. 模型 CRUD API

权限和租户规则：

1. 所有查询默认按 `session.current_tenant_id` 过滤；
2. 查询要求 `model:read`；
3. 创建、修改、禁用和删除要求 `model:write`；
4. 创建时 Server 从 Session 写入 tenant、owner 和审计字段，不接受客户端覆盖；
5. 更新时 Server 写入 `updated_by=session.current_user_id`；
6. 不允许查询、引用、修改或删除其他 tenant 的模型；
7. 默认 RuntimeEnvironment 必须与 Model 属于同一 tenant；
8. API 不比较 built-in 或 custom Role 名称，只检查实时解析的 permission code。

创建时固定写入：

```text
tenant_id = session.current_tenant_id
owner_id = session.current_user_id
created_by = session.current_user_id
updated_by = session.current_user_id
```

### 6.1 创建模型

```http
POST /api/models
```

### 6.2 查询模型列表

```http
GET /api/models
```

支持参数：

```text
enabled
model_type
keyword
```

### 6.3 查询模型详情

```http
GET /api/models/{id}
```

### 6.4 更新模型

```http
PUT /api/models/{id}
```

### 6.5 删除模型

```http
DELETE /api/models/{id}
```

删除规则：

1. 没有实例引用，可以删除；
2. 已有实例引用，不允许删除；
3. 可以先 disabled，再迁移或删除实例后删除；
4. 目标模型必须属于 `session.current_tenant_id`；
5. owner_id 是可转移业务所有者，不等于 created_by，也不直接决定删除权限；删除仍以 tenant scope 和 `model:write` 为准。

---

## 7. 创建模型请求示例

```json
{
  "name": "qwen3-32b",
  "display_name": "Qwen3 32B",
  "description": "Qwen3 32B local model",
  "model_type": "llm",
  "model_container_path": "/models/Qwen3-32B",
  "model_host_path": "/data/models/Qwen3-32B",
  "default_runtime_id": "runtime-vllm",
  "default_port": 8000,
  "default_context_len": 32768,
  "default_command_args_enabled": true,
  "default_command_args": [
    "--model",
    "/models/Qwen3-32B",
    "--served-model-name",
    "qwen3-32b",
    "--max-model-len",
    "32768"
  ],
  "default_env_enabled": false,
  "default_env": [],
  "runtime_metrics_enabled": true,
  "runtime_metrics_path": "/metrics",
  "enabled": true
}
```

---

## 8. 模型与 Docker 命令关系

模型定义提供：

```text
模型容器路径和可选宿主机路径
默认端口
默认启动参数
默认环境变量
runtime metrics path
```

运行环境提供：

```text
镜像
entrypoint
基础命令
端口映射模板
volume
device
security options
shm-size
ulimit
extra args
```

实例提供：

```text
节点
GPU
host port
实例名称
参数覆盖
环境变量覆盖
```

最终由 Server 生成并冻结 DockerRunSpec，Agent 只执行。

---

## 9. 模型参数覆盖规则

创建模型实例时允许覆盖模型默认参数。

规则：

1. 覆盖只影响该实例；
2. 不修改模型定义；
3. 启动命令快照必须记录最终参数；
4. Web 页面应显示哪些字段来自默认值，哪些字段被实例覆盖；
5. Model 提供默认 args，Instance 启用 override 时整体替换，不做参数名级合并；
6. env、volume、port 等字段按 `docs/08-engineering-contracts.md` 的确定算法合并。

---

## 10. 模型路径校验

第一阶段不在 Server 侧强制检查模型路径是否存在。

原因：

1. Server 可能不在 GPU 节点上；
2. 不同节点挂载路径可能不同；
3. 模型可能通过 volume 映射到容器内；
4. 实际可用性要由 Agent 启动时检查。

Agent 启动实例时应检查：

1. 宿主机挂载路径是否存在；
2. 冻结规格中的容器路径是否为绝对路径；
3. Docker 启动失败时记录 stderr。

---

## 11. Web 页面要求

模型页面应支持：

1. 模型列表；
2. 创建模型；
3. 编辑模型；
4. 启用 / 禁用模型；
5. 删除模型；
6. 查看关联实例数量；
7. 查看默认运行环境；
8. 查看默认启动参数。

模型详情页应展示：

1. 基础信息；
2. 模型容器路径和可选宿主机路径；
3. 默认运行环境；
4. 默认端口；
5. 默认上下文长度；
6. 默认启动参数；
7. 默认环境变量；
8. runtime metrics 配置；
9. 已创建实例列表。

---

## 12. 与 API Key / Token 的关系

第一阶段不实现 API Key、Token 统计和成本核算。

但模型名称后续可能用于：

1. OpenAI-compatible API 中的 model name；
2. API Key allowed models；
3. Token usage aggregation；
4. 成本统计；
5. 模型级限额。

因此模型 `name` 一旦被实例或用量引用，后续不建议随意修改。

`Model.TenantID` 是后续 API Key model access、Token usage 和成本聚合的归属依据。第一阶段只写入归属字段，不实现这些功能。

---

## 13. 日志与审计

模型变更需要记录：

1. 创建；
2. 修改；
3. 启用；
4. 禁用；
5. 删除；
6. 删除失败原因；
7. 被实例引用情况。

结构化日志同时记录 user_id、tenant_id、role_ids、required_permission、model_id 和 result。

第一阶段可以先写 Server 日志。
后续再做操作审计表。

---

## 14. 测试要求

至少包含：

1. 模型创建测试；
2. 模型名称唯一性测试；
3. 默认运行环境引用测试；
4. 删除引用保护测试；
5. disabled 模型不能创建新实例测试；
6. args 整体替换测试；
7. runtime metrics 字段保存测试；
8. model host/container path 语义测试；
9. DefaultContextLen 不重复注入测试；
10. tenant scope 查询过滤测试；
11. 跨 tenant Runtime/Model 引用拒绝测试；
12. 创建时归属字段由 Session 写入测试；
13. 缺少 `model:write` 时写操作拒绝、具有该 permission 时允许测试；
14. custom Role 的 model permission 生效测试。

---

## 15. MVP 完成标准

模型模块完成后，应达到：

1. 可以创建模型定义；
2. 可以编辑模型定义；
3. 可以禁用模型；
4. 可以删除未被引用的模型；
5. 已被实例引用的模型不能删除；
6. 模型可以选择默认运行环境；
7. 模型可以被实例引用；
8. 模型默认参数可以参与实例启动命令生成。
