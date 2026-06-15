# LightAI Go 模型运行与服务管理详细设计

> 修订：2026-06-15（基于 Claude 设计审核反馈）
> 修订要点：收缩第一阶段范围、明确 RuntimeEnvironment/RunTemplate 边界、简化 RunTemplate 模板引擎、明确 API 路径和状态枚举、确认设计决策

## 0. 文档目的

LightAI Go 当前已经实现节点注册、Agent 心跳、GPU 发现、GPU 指标采集、Dashboard、节点页面、GPU 页面、基础权限租户模型、日志和打包能力。

下一阶段目标是把 GPU 真正用起来：

```text
有哪些模型？
这些模型可以用什么运行环境启动？
某个模型部署在哪台节点、哪几张 GPU 上？
模型实例当前是否运行？
模型实例的 endpoint 是什么？
某张 GPU 正被哪个模型占用？
模型启动失败的原因是什么？
模型日志在哪里看？
将来如何统一 API Key、调用审计、计费、限流和调度？
```

本设计文档的目标不是立即开发全部功能，而是先固定抽象、边界和演进路线。后续应先审核本设计，再拆分成阶段实施文档，最后逐步开发。

---

## 1. 总体定位

```text
轻量、可控、可审计
适合中小客户、离线交付、国产 GPU 现场适配
先手动绑定资源，后智能调度
先跑通模型实例，后统一 Gateway、API Key、计费和调度策略
```

参考 GPUStack 的架构思想但不照搬。当前阶段不照搬：

```text
复杂自动调度、跨节点分布式推理、自动推理引擎选择
完整模型仓库、自动模型下载与同步
Ray 集群、复杂多副本弹性伸缩、复杂硬件拓扑感知
```

核心公式：

```text
ModelArtifact + RuntimeEnvironment + RunTemplate + ResourceBinding = ModelInstance
```

---

## 2. 架构兼容性

### 2.1 角色映射

| GPUStack 思路 | LightAI Go 对应 | Phase 1 | 说明 |
|---|---|---|---|
| Server | LightAI Server | ✅ | API、状态、资源绑定、任务下发 |
| Worker | LightAI Agent + Node | ✅ | GPU 发现、模型实例启动、日志、状态上报 |
| Scheduler | Scheduler 组件 | ❌ Phase 5 | 第一阶段只做手动资源绑定与校验 |
| Controllers | 状态流转逻辑 | 部分 | 简化状态流转，不做独立 Controller |
| Inference Server | vLLM / SGLang / MindIE / Custom | ✅ | 由 RuntimeDriver 启动或纳管 |
| HTTP Proxy / Gateway | LightAI Gateway | ❌ Phase 4 | OpenAI-compatible API、API Key、审计 |
| Model Deployment | ModelDeployment | ✅ | 期望状态 |
| Model Instance | ModelInstance | ✅ | 实际运行实例 |
| Local Path Model | ModelArtifact.source_type=local_path | ✅ | 第一阶段重点 |
| Backend | RuntimeEnvironment.backend_type | ✅ | 不自动选择 |

### 2.2 第一阶段最小闭环

```text
登记模型 → 登记运行环境 → 登记启动模板
→ 创建模型部署（手动选择 node/GPU/port）
→ Dry Run 预览 → 确认创建
→ [Phase 2] Agent 启动 Docker 容器
→ [Phase 2] Server 更新 ModelInstance
→ [Phase 2] GPU 页面显示占用
→ [Phase 3] Web 完整操作路径
```

---

## 3. 核心对象设计

### 3.1 ModelArtifact

描述模型本身，不描述如何运行。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| name | string | 程序内部名称，唯一 |
| display_name | string | 展示名称 |
| source_type | enum | local_path / mounted_path / remote_repo / object_storage |
| path | string | 本地路径、挂载路径或远程路径 |
| format | enum | hf / gguf / safetensors / onnx / custom / unknown |
| task_type | enum | chat / completion / embedding / rerank / vision / audio / multimodal / custom |
| architecture | string | qwen / llama / glm / deepseek / custom |
| size_label | string | 7B / 14B / 32B / 72B 等 |
| quantization | string | fp16 / bf16 / fp8 / int8 / int4 / unknown |
| default_context_length | int | 默认上下文长度 |
| estimated_vram_bytes | int64 | 预计显存需求 |
| required_gpu_count | int | 建议 GPU 数 |
| tenant_id | uuid | 所属租户 |
| owner_id | uuid | 创建者 |
| created_at | timestamp | — |
| updated_at | timestamp | — |

ModelArtifact 与 ModelDeployment 允许 1:N（一个模型可以有多个部署，搭配不同环境/模板）。

---

### 3.2 RuntimeEnvironment

描述运行环境。**只保存相对不变的运行基础设施**。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| name | string | 内部名称 |
| display_name | string | 展示名称 |
| runtime_type | enum | docker / process / remote / systemd / k8s / custom |
| backend_type | enum | vllm / sglang / llama_cpp / mindie / ollama / custom |
| vendor | enum | nvidia / metax / ascend / cambricon / hygon / cpu / custom / unknown |
| openai_compatible | bool | 是否 OpenAI API 兼容 |
| default_port | int | 默认服务端口 |
| health_check_path | string | 健康检查路径 |
| description | string | 描述 |
| tenant_id | uuid/null | 租户级运行环境；空表示全局 |
| owner_id | uuid | 创建者 |
| created_at | timestamp | — |
| updated_at | timestamp | — |

runtime_type 第一阶段实际只使用 `docker`。`process`、`remote`、`systemd`、`k8s` 保留 enum 值，Phase 2+ 逐步支持。

---

### 3.3 RuntimeEnvironmentDockerSpec

当 `runtime_type = docker` 时使用。1:1 关联 RuntimeEnvironment。

只保存**运行基础设施**（镜像、设备、安全配置）：

| 字段 | 类型 | 说明 |
|---|---|---|
| image | string | Docker image repo:tag 或 image id |
| image_pull_policy | enum | never / if_not_present / always（Phase 1 默认 never） |
| devices | array + enabled | 设备映射，如 /dev/dri |
| privileged | bool + enabled | privileged 模式 |
| ipc_mode | string + enabled | ipc 模式，如 host |
| uts_mode | string + enabled | uts 模式，如 host |
| network_mode | string + enabled | network 模式 |
| shm_size | string + enabled | shm size，如 100gb |
| group_add | array + enabled | 附加组，如 video |
| security_options | array + enabled | security-opt |
| ulimits | map + enabled | ulimits，如 memlock:-1 |
| restart_policy | string + enabled | restart policy |
| gpu_visible_env_key | string | GPU 可见设备环境变量名默认值（如 CUDA_VISIBLE_DEVICES），可由 RunTemplate 或 Deployment 覆盖 |

**所有可选参数必须有 `enabled` 开关。未启用的参数不得渲染进 ResolvedRunSpec。**

---

### 3.4 RuntimeEnvironmentProcessSpec

Phase 1 只保留数据结构定义，不做执行。

| 字段 | 类型 | 说明 |
|---|---|---|
| binary_path | string | 可执行文件路径 |
| working_dir | string | 工作目录 |
| default_args | array | 默认参数 |

其余字段（stdout_log_path、stderr_log_path、pid_file、stop_signal、stop_timeout_seconds、run_as_user）注释预留，Phase 2+ 实现。

---

### 3.5 RuntimeEnvironmentRemoteSpec

Phase 1 只保留数据结构定义，不做执行。

| 字段 | 类型 | 说明 |
|---|---|---|
| base_url | string | 已有模型服务地址 |
| auth_type | enum | none / bearer / basic / custom |
| health_check_path | string | 健康检查路径 |

其余字段（auth_config、openai_base_path、tls_verify）注释预留，Phase 4 实现。

---

### 3.6 RunTemplate

**保留独立 CRUD**。描述启动方式（参数、环境变量映射、挂载规则、端口映射）。

Phase 1 **不实现通用字符串模板引擎**。只保存结构化启动模板：

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| name | string | 内部名称 |
| display_name | string | 展示名称 |
| runtime_type | enum | docker / process / remote |
| vendor | enum | nvidia / metax / ascend / custom |
| backend_type | enum | vllm / sglang / mindie / custom |
| required_variables | array | 必填变量，如 INSTANCE_ID、MODEL_PATH、SERVED_MODEL_NAME、HOST_PORT、GPU_IDS |
| optional_variables | array | 可选变量，如 MAX_MODEL_LEN、GPU_MEMORY_UTILIZATION、TENSOR_PARALLEL_SIZE、DTYPE、EXTRA_ARGS |
| env_mappings | array + enabled | 环境变量映射，如 `{"key":"CUDA_VISIBLE_DEVICES","value_from":"GPU_IDS"}` |
| args_template | array | 结构化 args 数组，如 `["--model","${MODEL_PATH}","--served-model-name","${SERVED_MODEL_NAME}"]`。`${VAR}` 在渲染时由 Resolver 替换为实际值 |
| volume_mappings | array + enabled | 卷挂载映射，如 `{"host_path":"${MODEL_PATH}","container_path":"${MODEL_PATH}","readonly":true}` |
| port_mappings | array + enabled | 端口映射，如 `{"host_port":"${HOST_PORT}","container_port":8000,"protocol":"tcp"}` |
| backend_flags | map + enabled | backend 特定参数 |
| description | string | 描述 |
| tenant_id | uuid/null | 租户级或全局 |
| owner_id | uuid | 创建者 |
| created_at | timestamp | — |
| updated_at | timestamp | — |

**约束**：
- `env_mappings`、`volume_mappings`、`port_mappings`、`backend_flags` 必须有 `enabled` 开关
- `${VAR}` 替换在 Resolver 中实现，不在 RunTemplate 存储时做
- RunTemplate 不保存 Shell 片段，只保存结构化参数

---

### 3.7 ModelDeployment

描述期望状态。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| name | string | 内部名称 |
| display_name | string | 展示名称 |
| model_artifact_id | uuid | 模型 |
| runtime_environment_id | uuid | 运行环境 |
| run_template_id | uuid | 启动模板 |
| replicas | int | 副本数（Phase 1 固定 1） |
| desired_state | enum | running / stopped / deleted |
| status | enum | pending / starting / running / stopped / failed / unknown |
| node_id | uuid | 手动选择的节点 |
| gpu_ids | array | 手动选择的 GPU ID |
| host_port | int | 手动端口 |
| served_model_name | string | 对外服务模型名 |
| max_model_len | int | 最大上下文 |
| tensor_parallel_size | int | TP 数 |
| gpu_memory_utilization | float | GPU 显存使用比例 |
| dtype | string | 数据类型 |
| gpu_visible_env_key | string | 覆盖 GPU 可见设备环境变量名（默认来自 RuntimeEnvironment） |
| env_overrides | json | 环境变量覆盖 |
| arg_overrides | json | 参数覆盖 |
| extra_args | array | 额外参数 |
| schedule_mode | enum | manual / auto（Phase 1 只允许 manual） |
| placement_strategy | enum | manual / binpack / spread（Phase 1 只允许 manual，后续预留） |
| expose_mode | enum | none / direct / gateway / external |
| service_path | string | Gateway 阶段预留 |
| tenant_id | uuid | 所属租户 |
| owner_id | uuid | 创建者 |
| created_at | timestamp | — |
| updated_at | timestamp | — |

Phase 1 限制：`schedule_mode=manual`、`replicas=1`、`placement_strategy=manual`。

---

### 3.8 ModelInstance

描述实际运行实例。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| deployment_id | uuid | 所属部署 |
| replica_index | int | 副本序号 |
| node_id | uuid | 节点 |
| agent_id | string | Agent |
| runtime_type | enum | docker / process / remote |
| gpu_ids | array | 绑定 GPU ID |
| gpu_lease_ids | array | GpuLease ID |
| desired_state | enum | running / stopped / deleted |
| actual_state | enum | pending / starting / loading / running / unhealthy / stopping / stopped / failed / unknown |
| container_id | string | Docker 容器 ID |
| process_id | int | 进程 PID |
| remote_url | string | Remote endpoint |
| endpoint_url | string | 实例访问地址 |
| host_port | int | 主机端口 |
| container_port | int | 容器端口 |
| restart_count | int | 重启次数 |
| last_error | text | 最近错误 |
| last_exit_code | int | 最近退出码 |
| resolved_run_spec | json | 最终运行规格（审计用） |
| started_at | timestamp | 启动时间 |
| stopped_at | timestamp | 停止时间 |
| last_heartbeat_at | timestamp | 最近状态上报 |
| created_at | timestamp | — |
| updated_at | timestamp | — |

**状态枚举说明**：
- `pending`：已创建，等待分配/下发（Phase 1 手动绑定时与 assigned 合并）
- `starting`：Agent 已接收任务，正在执行 docker run
- `loading`：容器已启动，模型加载到显存中
- `running`：健康检查通过
- `unhealthy`：健康检查失败
- `stopping`：正在执行 docker stop
- `stopped`：已停止
- `failed`：启动或运行失败
- `unknown`：Agent 离线，无法确认状态

---

### 3.9 GpuLease

GPU 占用锁。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| gpu_id | uuid | GPU |
| node_id | uuid | 节点 |
| deployment_id | uuid | 部署 |
| instance_id | uuid | 实例 |
| tenant_id | uuid | 租户 |
| status | enum | reserved / active / released / expired / failed |
| expires_at | timestamp | 预留超时时间 |
| created_at | timestamp | — |
| updated_at | timestamp | — |

GpuLease 不存储 `reserved_memory_bytes` 和 `reserved_gpu_count`（Phase 1 冗余字段，从 gpu_devices JOIN 获取）。

生命周期：
```text
创建实例前：reserved → Agent 启动成功：active
→ 启动失败：failed → released
→ 实例停止：released
→ 预留超时：expired
```

---

### 3.10 预留对象

以下对象 Phase 1 只定义 struct，不建表，不实现 API：

- **ModelRoute**（Phase 4 Gateway 阶段建表）
- **ModelUsageRecord**（Phase 4 Gateway 阶段建表）
- **ModelEvent**（Phase 1 复用 `audit_logs` 表，逻辑概念保留）

---

## 4. RuntimeEnvironment / RunTemplate / ModelDeployment 职责边界

```
RuntimeEnvironment  = 基础设施（image、device、privileged、shm_size、security、gpu_visible_env_key）
RunTemplate         = 启动方式（args、env mappings、volume mappings、port mappings、backend flags）
ModelDeployment     = 本次部署参数（选模型、选环境、选模板、选节点/GPU/端口、覆盖参数）
```

合并顺序：RuntimeDefaults → TemplateOverrides → DeploymentOverrides。

---

## 5. Dry Run 设计

Dry Run 接收绑定参数，输出校验结果 + ResolvedRunSpec + 等价命令预览。

不创建 GpuLease、不创建 ModelInstance、不下发 Agent 任务、不启动容器。

---

## 6. ResolvedRunSpec

Server 是唯一生成方。复用现有 `DockerRunSpec` 结构（`08-engineering-contracts.md` §6），外层增加元信息：

```go
type ResolvedRunSpec struct {
    InstanceID   string        `json:"instance_id"`
    DeploymentID string        `json:"deployment_id"`
    RuntimeType  string        `json:"runtime_type"`
    BackendType  string        `json:"backend_type"`
    Vendor       string        `json:"vendor"`
    ModelPath    string        `json:"model_path"`
    ServedModelName string    `json:"served_model_name"`
    NodeID       string        `json:"node_id"`
    AgentID      string        `json:"agent_id"`
    GPUDeviceIDs []string      `json:"gpu_device_ids"`
    Env          []EnvVar      `json:"env"`
    Args         []string      `json:"args"`
    Docker       DockerRunSpec `json:"docker"`   // 复用现有
    Process      *ProcessSpec  `json:"process"`  // Phase 2+
    Remote       *RemoteSpec   `json:"remote"`   // Phase 4+
}
```

Agent 不拼接 Shell 字符串。等价命令预览仅用于展示和排错。

---

## 7. RuntimeDriver（Phase 2）

```go
type RuntimeDriver interface {
    Validate(ctx context.Context, spec ResolvedRunSpec) error
    Start(ctx context.Context, spec ResolvedRunSpec) (*StartResult, error)
    Stop(ctx context.Context, instance RuntimeInstance) error
    Restart(ctx context.Context, instance RuntimeInstance) error
    Status(ctx context.Context, instance RuntimeInstance) (*RuntimeStatus, error)
    Logs(ctx context.Context, instance RuntimeInstance, opts LogOptions) (*LogResult, error)
}
```

Phase 2 只实现 `DockerRuntimeDriver`。`ProcessRuntimeDriver` 和 `RemoteRuntimeDriver` 提供 stub。

---

## 8. Agent Task 设计（Phase 2）

新增任务类型：StartModelInstance、StopModelInstance、RestartModelInstance、GetModelInstanceLogs、InspectModelInstance。

---

## 9. 状态流转

### 启动成功

```text
用户 start → Server 校验 → 创建 ModelInstance(pending) → 创建 GpuLease(reserved)
→ 生成 ResolvedRunSpec → 下发 StartModelInstance
→ Agent Validate+Start → 返回 container_id/endpoint_url
→ Server: actual_state=running, lease=active, deployment=running
```

### 启动失败

```text
Agent 失败 → 返回 last_error
→ Server: actual_state=failed, lease=released, deployment=failed
→ 不遗留 active GpuLease
```

### Agent 离线

```text
Agent 离线 → ModelInstance actual_state=unknown
→ GpuLease 暂不释放（避免误判导致双占用）
→ Web 显示"Agent 离线，实例状态未知"
```

Phase 2+ 增加离线超时自动释放策略。

---

## 10. API 设计

沿用现有 `/api/` 前缀（无 v1），与现有 `/api/v1/nodes`、`/api/v1/gpus` 风格一致。

```text
# ModelArtifact
GET    /api/v1/model-artifacts
POST   /api/v1/model-artifacts
GET    /api/v1/model-artifacts/{id}
PATCH  /api/v1/model-artifacts/{id}
DELETE /api/v1/model-artifacts/{id}

# RuntimeEnvironment
GET    /api/v1/runtime-environments
POST   /api/v1/runtime-environments
GET    /api/v1/runtime-environments/{id}
PATCH  /api/v1/runtime-environments/{id}
DELETE /api/v1/runtime-environments/{id}

# RunTemplate
GET    /api/v1/run-templates
POST   /api/v1/run-templates
GET    /api/v1/run-templates/{id}
PATCH  /api/v1/run-templates/{id}
DELETE /api/v1/run-templates/{id}

# ModelDeployment
GET    /api/v1/model-deployments
POST   /api/v1/model-deployments
GET    /api/v1/model-deployments/{id}
PATCH  /api/v1/model-deployments/{id}
DELETE /api/v1/model-deployments/{id}
POST   /api/v1/model-deployments/{id}/dry-run
POST   /api/v1/model-deployments/{id}/start       # Phase 2
POST   /api/v1/model-deployments/{id}/stop        # Phase 2

# ModelInstance
GET    /api/v1/model-instances
GET    /api/v1/model-instances/{id}
GET    /api/v1/model-instances/{id}/logs           # Phase 2

# GpuLease（只读，系统管理）
GET    /api/v1/gpu-leases
GET    /api/v1/gpu-leases/{id}
```

---

## 11. 权限

新增 permission codes：

```text
model:read
model:write
runtime:read
runtime:write
deployment:read
deployment:write
```

复用现有 viewer/operator/admin/platform_admin 角色体系。

---

## 12. 敏感字段脱敏

env key 包含 KEY/TOKEN/PASSWORD/SECRET/AUTH/CREDENTIAL/ACCESS/PRIVATE 时默认脱敏。脱敏显示为 `<redacted>`。

---

## 13. 指标（Phase 2+）

```text
lightai_model_instance_status
lightai_model_instance_start_total
lightai_model_instance_start_failed_total
lightai_gpu_lease_active
```

---

## 14. 设计决策确认

1. ✅ RunTemplate 保留独立 CRUD，Phase 1 简化（结构化，无通用模板引擎）
2. ✅ ModelArtifact 与 ModelDeployment 允许 1:N
3. ✅ GPU 可见变量由 RuntimeEnvironment 提供默认值（gpu_visible_env_key），RunTemplate 引用，Deployment 可覆盖
4. ✅ docker stop 后默认不 rm；delete instance/deployment 时才 rm -f
5. ✅ Agent 离线时不立即释放 GpuLease，先标记实例为 unknown，避免误释放导致双占用
6. ✅ Phase 1 复用 audit_logs，不新增 ModelEvent 表；ModelEvent 作为逻辑概念保留
7. ✅ Phase 1 不建 ModelRoute 表，不建 ModelUsageRecord 表
8. ✅ API 路径不加 /v1 前缀，沿用现有风格

---

## 15. 分阶段路线图

| Phase | 名称 | 核心交付 |
|---|---|---|
| 1 | 数据模型与 Dry Run | CRUD API + Dry Run + ResolvedRunSpec + 权限 |
| 2 | Agent Docker Runtime | DockerRuntimeDriver + Start/Stop/Logs + GpuLease 流转 |
| 3 | Web 模型服务 | 模型库/环境/模板/部署/实例页面 + GPU 占用展示 |
| 4 | Gateway + API Key | OpenAI-compatible API + API Key + Usage 记录 |
| 5 | 基础自动调度 | manual→auto + vendor/显存过滤 + best-fit |

详见 `docs/plan/12-model-runtime-serving-implementation-plan.md` 及各 Phase 文档。

---

## 16. 结论

第一阶段应实现：模型可登记、运行环境可配置、启动模板可复用、资源可手动绑定、启动前可 Dry Run。同时通过 RuntimeEnvironment / RunTemplate / ResolvedRunSpec / RuntimeDriver / GpuLease 等抽象为后续调度和 Gateway 留出空间。
