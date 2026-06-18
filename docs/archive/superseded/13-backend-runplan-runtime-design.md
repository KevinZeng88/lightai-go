> Status: ARCHIVED
> Archived on: 2026-06-18
> Do not use as current implementation guidance.
> Current entrypoint: docs/CURRENT.md

# LightAI Go 推理后端 / Runtime / RunPlan 设计

> Phase 0.1 修订版 — 后续 Phase 1-6 实施的唯一参考。
> 日期: 2026-06-16
> 版本: v2.0（根据人工审核意见修订）

---

## 1. 为什么废弃旧 runtime_env + template + instance 拼接方式

### 1.1 旧链路的根本问题

当前链路：

```
ModelArtifact → RuntimeEnvironment → RunTemplate → ModelDeployment → ModelInstance → AgentTask
```

问题清单：

1. **配置重复分散**：Docker 配置同时存在于 `runtime_environments`、`runtime_environment_docker_specs`、`run_templates`、`model_deployments` 四个表中。

2. **三向强制引用**：`model_deployments` 必须同时引用 `model_artifact_id`、`runtime_environment_id`、`run_template_id`，用户操作步骤多。

3. **概念混淆**：`RuntimeEnvironment` 既表示"推理引擎类型"（backend_type=vllm），又包含"Docker 运行参数"（privileged, ipc, shm）。GPUStack 将这两者分离为 `InferenceBackend` 和版本级运行配置，LightAI Go 需要进一步拆分。

4. **缺少版本概念**：没有版本化的镜像和命令管理。用户需要在 RuntimeEnvironment 里手动填写 Docker image，每次 vLLM 升级都要手动改。

5. **Agent 不冻结规格**：`HandleStartDeployment` 在 start 时才调用 `resolver.Resolve()`。如果 RuntimeEnvironment 在 start 后被修改，没有冻结的不可变记录作为审计依据。

6. **缺少节点级覆盖**：不同服务器的 image 名称、模型根目录、设备路径不同，旧设计无法表达这种差异。

### 1.2 明确声明不做兼容

本次重构**不兼容**旧链路：

- `runtime_environments`、`run_templates`、`model_deployments` 表、struct、API、Web 页面全部删除。
- 旧数据不迁移（当前是开发阶段，没有生产数据）。
- 旧 API 端点全部替换，不保留兼容路由。
- 旧 Web 页面全部替换。

---

## 2. 新对象模型

### 2.1 最终对象关系

```
InferenceBackend（后端家族定义）
  └── BackendVersion（后端版本定义）

BackendRuntimeTemplate（系统只读运行模板）
  └── BackendRuntime（用户可编辑的运行配置）
        └── NodeRuntimeOverride（节点级覆盖）

ModelArtifact（模型文件与元数据）

ModelDeployment（用户部署规格）
  引用 ModelArtifact + BackendRuntime

ModelInstance（调度后的运行实体）
  └── ResolvedRunPlan（冻结的最终运行计划 → 独立表）

DockerExecutor（Agent 端执行器，只消费 ResolvedRunPlan）
```

### 2.2 每个对象职责边界

| 对象 | 职责 | 不负责 |
|------|------|--------|
| **InferenceBackend** | 后端家族定义（vllm/sglang/llamacpp）、协议、参数格式 | vendor、image、Docker 参数、设备 |
| **BackendVersion** | 版本定义、默认 entrypoint/args/参数、健康检查、推荐镜像 | 实际运行 image、Docker 安全参数、设备 |
| **BackendRuntimeTemplate** | 系统只读模板，用于创建 BackendRuntime | 直接参与实例运行 |
| **BackendRuntime** | 用户可编辑运行配置：vendor + image + devices + docker flags + env + model mount | 节点差异、模型定义、部署规格 |
| **NodeRuntimeOverride** | 节点级 image/env/device/modelRoot 覆盖 | 与特定部署绑定的参数 |
| **ModelArtifact** | 模型文件路径、格式、元数据、资源估算 | 后端选择、运行配置、部署规格 |
| **ModelDeployment** | 选择 artifact + runtime、定义 placement/parameters/service | 容器配置、镜像选择、设备定义 |
| **ModelInstance** | 运行实体状态、容器 ID、endpoint | 部署规格定义、运行计划生成 |
| **ResolvedRunPlan** | 冻结最终运行计划（独立表，不可变） | 动态状态、用户编辑 |
| **DockerExecutor** | 消费 RunPlan 启动/停止 Docker 容器 | 生成 RunPlan、选择参数 |

---

## 3. InferenceBackend 设计

### 3.1 职责

`InferenceBackend` 表示后端家族定义，不绑定 vendor。

第一版只内置三个后端：

```text
vllm
sglang
llamacpp
```

暂不实现：Custom Backend、MindIE、VoxBox。

### 3.2 字段

```text
id
name                      -- "vllm", "sglang", "llamacpp"
display_name
description
protocol_json             -- OpenAI-compatible 协议定义
default_version           -- 默认版本（如 "0.8.5"），未指定版本时使用
parameter_format          -- "space" | "equal"
common_parameters_json    -- Web UI 常用参数提示（参考 GPUStack common_parameters）
default_env_json          -- 后端级默认环境变量（所有版本共享）
is_builtin
is_enabled
created_at
updated_at
```

### 3.3 不包含 vendor

`InferenceBackend` 不包含 `vendor` 字段。vendor（nvidia/metax）属于 `BackendRuntime` 层面，不属于后端家族定义。

### 3.4 示例

```yaml
apiVersion: lightai/v1
kind: InferenceBackend
metadata:
  name: vllm
spec:
  displayName: vLLM
  protocol:
    type: openai-compatible
    modelsPath: /v1/models
    chatCompletionsPath: /v1/chat/completions
    completionsPath: /v1/completions
  defaultVersion: "0.8.5"
  parameterFormat: space
  commonParameters:
    - "--tensor-parallel-size"
    - "--max-model-len"
    - "--gpu-memory-utilization"
    - "--served-model-name"
  defaultEnv:
    VLLM_USE_MODELSCOPE: "false"
  isBuiltIn: true
  isEnabled: true
```

### 3.5 BackendVersion 与 BackendRuntime 边界

**BackendVersion 负责**：
- 版本语义（"0.8.5"）
- 默认 entrypoint
- 默认 run command / args
- 默认 backend params（default_backend_params_json）
- 参数定义（parameter_defs_json）
- 默认 env（env_json）
- health check
- 推荐 image（default_images_json，按 vendor）

**BackendVersion 不负责**：
- 最终实际运行 image
- vendor-specific Docker devices
- privileged / ipc / uts / shm / ulimit / security_opt
- 节点级 image/modelRoot 差异

### 3.6 InferenceBackend 不包含的内容

- vendor
- Docker image
- entrypoint / command / args
- devices
- privileged / ipc / uts
- shm_size / ulimit / security_opt
- 模型路径
- 节点/GPU 选择

---

## 4. BackendVersion 设计

### 4.1 职责

`BackendVersion` 表示某个后端家族的特定版本定义，例如：

```text
vllm 0.8.5
vllm 0.10.0
sglang 0.4.6
sglang 0.5.0
llamacpp b4817
```

### 4.2 字段

```text
id
backend_id                  -- FK → inference_backends
version                     -- "0.8.5", "b4817"
display_name
is_default                  -- 是否为该后端的默认版本
default_entrypoint_json     -- ["vllm", "serve"]
default_args_json           -- 默认 CLI args 模板（等价 GPUStack run_command）
default_backend_params_json  -- 该版本默认追加参数
parameter_defs_json         -- 参数定义列表
health_check_json           -- 健康检查配置
default_container_port      -- 8000
default_images_json         -- 推荐镜像（按 vendor）: {"nvidia":"...", "metax":"..."}
env_json                   -- 版本级默认环境变量
```

### 4.3 default_images_json 只是推荐值

`BackendVersion.default_images_json` 是推荐镜像，不是最终运行镜像。镜像解析优先级见 §8.4。

```json
{
  "nvidia": "vllm/vllm-openai:v0.8.5",
  "metax": "registry.local/vllm-openai:v0.8.5-metax"
}
```

### 4.4 示例

```yaml
apiVersion: lightai/v1
kind: BackendVersion
metadata:
  backend: vllm
  version: "0.8.5"
spec:
  isDefault: true

  defaultEntrypoint:
    - vllm
    - serve

  defaultArgs:
    - "{{model_container_path}}"
    - "--host"
    - "0.0.0.0"
    - "--port"
    - "{{container_port}}"
    - "--served-model-name"
    - "{{served_model_name}}"
    - "--max-model-len"
    - "{{max_model_len}}"
    - "--gpu-memory-utilization"
    - "{{gpu_memory_utilization}}"

  defaultBackendParams:
    - "--enforce-eager"

  env:
    VLLM_USE_MODELSCOPE: "true"

  parameters:
    - name: max_model_len
      cli_name: "--max-model-len"
      type: integer
      default: 8192
      required: false
    - name: gpu_memory_utilization
      cli_name: "--gpu-memory-utilization"
      type: number
      default: 0.9
      required: false
    - name: served_model_name
      cli_name: "--served-model-name"
      type: string
      required: true
    - name: tensor_parallel_size
      cli_name: "--tensor-parallel-size"
      type: integer
      default: 1
      required: false

  healthCheck:
    path: /v1/models
    expectedStatus: 200
    startupTimeoutSeconds: 120
    intervalSeconds: 2
    timeoutSeconds: 5

  defaultContainerPort: 8000

  defaultImages:
    nvidia: "vllm/vllm-openai:v0.8.5"
```

### 4.5 不包含的内容

- 具体服务器实际 image（属于 BackendRuntime 和 NodeRuntimeOverride）
- MetaX / NVIDIA 运行差异（属于 BackendRuntime）
- devices / privileged / ipc / shm_size / ulimit / security_opt（属于 BackendRuntime）
- 模型路径 / 节点/GPU 选择（属于 ModelDeployment）

---

## 5. BackendRuntimeTemplate 设计

### 5.1 职责

`BackendRuntimeTemplate` 是系统只读运行模板，用于创建 `BackendRuntime`。

它不落入模型实例运行链路，不被 `ModelDeployment` 直接引用，不直接生成 RunPlan。

```text
BackendRuntimeTemplate
    ↓ clone/create
BackendRuntime
    ↓ referenced by
ModelDeployment
    ↓ resolve
ResolvedRunPlan
```

### 5.2 第一版内置模板

```text
vllm-nvidia-docker
vllm-metax-docker
sglang-nvidia-docker
sglang-metax-docker
llamacpp-nvidia-docker
```

### 5.3 字段（不落库，仅来自配置文件）

**明确：第一版 BackendRuntimeTemplate 不创建数据库表，不落库。** API 只读返回配置文件内容。
POST /api/v1/backend-runtimes/from-template 时，才 clone 成 backend_runtimes 表记录。

```text
name              -- "vllm-nvidia-docker"
display_name
backend_name      -- "vllm"
backend_version   -- "0.8.5"
vendor            -- "nvidia"
runtime_type      -- "docker"
image_name
image_pull_policy
entrypoint_json
args_override_json
default_env_json
docker_json       -- devices, privileged, ipc, shm, ulimits, etc.
model_mount_json
health_check_override_json
```

### 5.4 配置目录

```
configs/model-runtime/
  backends/
    vllm.yaml
    sglang.yaml
    llamacpp.yaml

  backend-versions/
    vllm/
      0.8.5.yaml
      0.10.0.yaml
    sglang/
      0.4.6.yaml
      0.5.0.yaml
    llamacpp/
      b4817.yaml

  backend-runtime-templates/
    vllm-nvidia-docker.yaml
    vllm-metax-docker.yaml
    sglang-nvidia-docker.yaml
    sglang-metax-docker.yaml
    llamacpp-nvidia-docker.yaml
```

不再使用旧的 `configs/templates/backends/*.json` 结构（那个结构把 Backend、BackendVersion、RuntimeTemplate 混在一个文件里）。

---

## 6. BackendRuntime 设计

### 6.1 职责

`BackendRuntime` 是用户现场可编辑的运行配置。

它表示：

```text
某个 BackendVersion
  + 某个 GPU vendor
  + Docker runtime
  + 具体 image
  + 设备挂载
  + 安全参数
  + 默认环境变量
  + 模型挂载策略
```

### 6.2 示例 Runtime 实例

```text
vllm 0.8.5 + nvidia + docker
vllm 0.8.5 + metax + docker
sglang 0.4.6 + nvidia + docker
sglang 0.4.6 + metax + docker
llamacpp b4817 + nvidia + docker
```

### 6.3 字段

```text
id
name
display_name
backend_id                  -- FK → inference_backends
backend_version_id            -- FK → backend_versions
source_template_name          -- 从哪个模板创建的
vendor                        -- "nvidia" | "metax" | "cpu"
runtime_type                  -- 当前只允许 "docker"
image_name                    -- 默认运行镜像
image_pull_policy             -- "always" | "if_not_present" | "never"
entrypoint_override_json      -- 覆盖 BackendVersion 的 entrypoint
args_override_json            -- 追加 BackendVersion.default_args_json（第一版只做 append，不支持 replace）
default_env_json              -- 默认环境变量
docker_json                   -- devices, privileged, ipc, shm, ulimits, security_opts
model_mount_json              -- 模型挂载策略
health_check_override_json    -- 覆盖 BackendVersion 的健康检查
is_builtin
is_editable
tenant_id
created_at
updated_at
```

### 6.4 BackendRuntime.image_name 的角色

`BackendRuntime.image_name` 是这个运行配置的默认实际镜像。

它不同于 `BackendVersion.defaultImages`：

| 字段 | 含义 | 优先级 |
|------|------|--------|
| `BackendVersion.defaultImages[vendor]` | 某个版本对某个 vendor 的推荐镜像 | 最低，仅推荐 |
| `BackendRuntime.image_name` | 用户现场保存的默认运行镜像 | 中等 |
| `NodeRuntimeOverride.image_name` | 某节点对该 Runtime 的实际镜像覆盖 | 最高 |

---

## 7. NodeRuntimeOverride 设计

### 7.1 为什么需要

同一个 BackendRuntime 在不同服务器上可能需要不同的 image、模型根目录、设备路径或 env。

例如：

```text
node-01 本地镜像是 0d307f1665d3（docker load 导入的）
node-02 镜像是 registry.local/metax/vllm:0.8.5
node-03 模型目录是 /mnt/models
node-04 设备路径有差异
```

如果只把 image 放在 BackendRuntime，全局只能有一个值。

### 7.2 字段

```text
id
node_id                       -- FK → nodes
backend_runtime_id            -- FK → backend_runtimes
image_name                    -- 覆盖 BackendRuntime.image_name
image_pull_policy
env_json                      -- 追加或覆盖 BackendRuntime.default_env
docker_override_json          -- 覆盖 BackendRuntime.docker_json 中的部分字段（含 devices）
model_root_host_path          -- 覆盖模型根目录（该节点上的实际路径）
is_enabled
created_at
updated_at
```

### 7.3 示例

```yaml
apiVersion: lightai/v1
kind: NodeRuntimeOverride
metadata:
  name: node-k8s-master1-vllm-metax
spec:
  nodeId: node-k8s-master1
  backendRuntimeId: vllm-metax-docker

  image:
    name: "0d307f1665d3"
    pullPolicy: never

  modelRoot:
    hostPath: /data/part2/MX-C500/model

  docker:
    devices:
      - hostPath: /dev/dri
        containerPath: /dev/dri
      - hostPath: /dev/mxcd
        containerPath: /dev/mxcd
      - hostPath: /dev/infiniband
        containerPath: /dev/infiniband
```

### 7.4 第一版实现建议

- 表结构和 Resolver 支持必须实现
- Web 页面可以简单处理：BackendRuntime 详情中显示"节点覆盖"区域
- 如果没有覆盖，使用 BackendRuntime 默认配置

---

## 8. 配置归属表

### 8.1 完整归属矩阵

| 配置项 | Backend | Version | RuntimeTemplate | BackendRuntime | NodeOverride | Artifact | Deployment | RunPlan |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| 后端名称/类型 | ✓ | | | | | | | |
| 协议定义 | ✓ | | | | | | | |
| 参数格式 | ✓ | | | | | | | |
| 版本号 | | ✓ | | | | | | |
| 默认 entrypoint | | ✓ | | | | | | ✓(frozen) |
| 默认 args 模板 | | ✓ | | | | | | ✓(resolved) |
| 参数定义 | | ✓ | | | | | | |
| 健康检查 | | ✓ | | | | | | ✓(frozen) |
| 默认端口 | | ✓ | | | | | | |
| 推荐镜像 | | ✓ | | | | | | |
| vendor | | | ✓ | ✓ | | | | |
| runtime_type | | | ✓ | ✓ | | | | ✓(frozen) |
| 实际运行镜像 | | | ✓ | ✓ | ✓ | | | ✓(frozen) |
| image_pull_policy | | | ✓ | ✓ | ✓ | | | ✓(frozen) |
| entrypoint override | | | ✓ | ✓ | | | | ✓(frozen) |
| args override | | | ✓ | ✓ | | | | ✓(frozen) |
| devices (via docker_json) | | | ✓ | ✓ | ✓ | | | ✓(frozen) |
| privileged | | | ✓ | ✓ | | | | ✓(frozen) |
| ipc_mode | | | ✓ | ✓ | | | | ✓(frozen) |
| shm_size | | | ✓ | ✓ | | | | ✓(frozen) |
| ulimits | | | ✓ | ✓ | | | | ✓(frozen) |
| security_options | | | ✓ | ✓ | | | | ✓(frozen) |
| default env | | | ✓ | ✓ | | | | ✓(merged) |
| env override | | | | | ✓ | | ✓ | ✓(merged) |
| model mount 策略 | | | ✓ | ✓ | | | | ✓(frozen) |
| model_root_host_path | | | | | ✓ | | | ✓(frozen) |
| 模型路径 | | | | | | ✓ | | ✓(frozen) |
| 模型格式/架构 | | | | | | ✓ | | |
| VRAM 估算 | | | | | | ✓ | | |
| 后端选择 | | | | | | | ✓ | |
| Runtime 选择 | | | | | | | ✓ | |
| placement（节点/GPU） | | | | | | | ✓ | ✓(frozen) |
| service（端口/暴露） | | | | | | | ✓ | ✓(frozen) |
| parameters | | | | | | | ✓ | ✓(frozen) |
| env_overrides | | | | | | | ✓ | ✓(merged) |
| replicas | | | | | | | ✓ | |
| 实例状态 | | | | | | | | 属于 Instance |

### 8.2 image 解析优先级

Resolver 解析 image 的优先级：

```text
1. NodeRuntimeOverride.image_name
2. BackendRuntime.image_name
3. BackendVersion.defaultImages[vendor]
4. error: no image available
```

第一版不开放 `ModelDeployment.image_override`。

---

## 9. ModelArtifact 设计

### 9.1 职责

描述一个可部署的模型文件及其元数据。与当前设计基本一致，但**不引用**任何 Backend 或 Runtime。

### 9.2 字段

```text
id
name                          -- 唯一标识
display_name
description
source                        -- "local_path" | "huggingface" | "model_scope"
path                          -- 宿主路径或 HF repo ID
format                        -- "safetensors", "gguf", "pytorch", "custom"
task_type                     -- "chat", "embedding", "reranker", "image", "audio"
architecture                  -- "LlamaForCausalLM", "Qwen2ForCausalLM", etc.
size_label                    -- "7B", "13B", "70B"
quantization                  -- "none", "q4_k_m", "fp16", "int8"
estimated_vram_bytes
required_gpu_count
default_context_length
tenant_id
owner_id
// 审计字段
```

### 9.3 不包含

- `backend_id` — 后端选择是 ModelDeployment 的职责
- `runtime_environment_id` — 旧概念，已删除

---

## 10. ModelDeployment 设计

### 10.1 职责

用户对一次模型部署的规格定义——选择哪个模型、哪个 BackendRuntime、放到哪个节点、用哪些 GPU。

### 10.2 引用 BackendRuntime（不是 backend_id + version_id）

```text
backend_runtime_id    -- FK → backend_runtimes
```

不使用：

```text
backend_id                  -- FK → inference_backends
backend_version_id    -- 删除
```

原因：Deployment 选择的是某个可运行配置，BackendRuntime 已经知道自己属于哪个 BackendVersion。

### 10.3 字段

```text
id
name
display_name
description
model_artifact_id             -- FK → model_artifacts
backend_runtime_id            -- FK → backend_runtimes
replicas                      -- Phase 1 固定为 1
placement_json                -- placement 配置
service_json                  -- service 配置
parameters_json               -- 后端参数值
env_overrides_json            -- 覆盖 BackendRuntime 的环境变量
desired_state                 -- "stopped" | "running"
status                        -- "stopped" | "running" | "error"
tenant_id
owner_id
created_by
updated_by
created_at
updated_at
```

### 10.4 Placement 设计

使用 JSON 字段而非平铺字段：

```yaml
placement:
  mode: manual              # "manual" | "auto"（Phase 1: manual only）
  strategy: single_node     # "single_node" | "multi_replica" | "distributed"
  nodeId: node-01
  gpuIds: [0, 1, 2, 3]
  gpuCountPerReplica: 4
  allowMultiNodeSingleReplica: false  # Phase 1: false
```

第一版只实现：

```text
mode = manual
strategy = single_node
allowMultiNodeSingleReplica = false
replicas = 1
```

### 10.5 Parameters 设计

`parameters_json` 保存后端参数值：

```json
{
  "served_model_name": "qwen35-9b",
  "max_model_len": 8192,
  "gpu_memory_utilization": 0.9,
  "tensor_parallel_size": 2
}
```

参数名来自 `BackendVersion.parameter_defs_json`。

### 10.6 Service 设计

```yaml
service:
  hostPort: 8001
  exposeMode: direct         # "direct" | "proxy"
  path: ""                   # 仅 proxy mode
```

---

## 11. ModelInstance 设计

### 11.1 职责

表示调度后的单个运行实体。第一版：一个 ModelInstance = 一个节点上的一个容器。

### 11.2 字段

```text
id
deployment_id                 -- FK → model_deployments
replica_index
node_id
agent_id
assigned_gpus_json            -- 分配的 GPU 列表
gpu_lease_ids_json
host_port
container_port
current_run_plan_id           -- FK → resolved_run_plans
actual_state                  -- "pending" | "initializing" | "starting" | "running" | "stopped" | "error"
desired_state                 -- "running" | "stopped"
container_id
endpoint_url
restart_count
last_error
started_at
stopped_at
```

### 11.3 不要把完整 RunPlan 只塞在 Instance JSON 字段里

旧设计中的 `resolved_run_spec TEXT` 字段删除或降级为缓存。

**正式设计使用独立表 `resolved_run_plans`**。ModelInstance 只保存：

```text
current_run_plan_id
```

---

## 12. ResolvedRunPlan 设计（独立表）

### 12.1 职责

最终不可变的运行计划。RunPlan Resolver 在部署启动时生成，Agent 只能消费不能修改。

### 12.2 新增表：resolved_run_plans

```text
id
deployment_id                 -- FK → model_deployments
instance_id                   -- FK → model_instances
backend_runtime_id            -- 生成时使用的 BackendRuntime（用于审计）
node_runtime_override_id      -- 生成时使用的 NodeRuntimeOverride（可为 NULL）
plan_json                     -- 完整的 ResolvedRunPlan JSON
docker_preview                -- docker run 命令预览
input_hash                    -- SHA256(all inputs)
plan_hash                     -- SHA256(plan_json)
created_by
created_at
```

### 12.3 不可变规则

```text
RunPlan 生成后不可修改。
每次启动或重启都生成新的 RunPlan。
ModelInstance.current_run_plan_id 指向当前使用的 RunPlan。
历史 RunPlan 保留用于审计和排障。
```

### 12.4 RunPlan 冻结内容

```text
backend name
backend version
backend runtime
node runtime override（如果存在）
model artifact
deployment parameters
assigned node
assigned GPU
image（最终解析值）
entrypoint（最终解析值）
args（所有模板变量已替换）
env（所有来源已合并）
docker devices
docker mounts
ports
privileged
ipc
uts
network mode
shm size
ulimits
security options
health check
docker preview
input_hash
plan_hash
```

### 12.5 Start 事务顺序（解决双向引用）

`model_instances` 和 `resolved_run_plans` 存在双向引用（instance 引用 current_run_plan_id，run_plan 引用 instance_id）。Start deployment 时必须使用以下顺序，单事务内执行：

```text
1. INSERT model_instance（current_run_plan_id = NULL）
2. 调用 RunPlan Resolver（带入 instance_id）
3. INSERT resolved_run_plans（instance_id = 新创建的 instance.id）
4. UPDATE model_instance SET current_run_plan_id = run_plan.id
5. INSERT gpu_leases
6. INSERT agent_task
7. COMMIT
```

如果任一步失败，整个事务回滚。

### 12.6 每次 restart 创建新 RunPlan

```text
每次 start 或 restart 都创建新的 resolved_run_plans 记录。
不能覆盖旧 RunPlan。
旧 RunPlan 保留用于审计和排障。
ModelInstance.current_run_plan_id 更新为最新 RunPlan。
```

### 12.7 数据库约束说明

`resolved_run_plans.instance_id` 允许 NULL，因为插入时 instance 的 `current_run_plan_id` 尚未更新（Step 3 时 instance 已存在但 run_plan_id 在 Step 4 才回填）。但业务逻辑保证最终一定非 NULL。

---

## 13. 模板变量语法

### 13.1 只支持 `{{var}}`

统一使用 `{{var}}` 语法。**不支持 `${VAR}` 语法**。

支持的变量列表：

| 变量 | 来源 | 示例值 |
|------|------|--------|
| `{{model_container_path}}` | 模型容器内路径（经 mount 转换） | `/models/Qwen3-32B` |
| `{{model_host_path}}` | 模型宿主机路径 | `/data/models/Qwen3-32B` |
| `{{model_parent_host_path}}` | 模型宿主机父目录 | `/data/models` |
| `{{container_port}}` | BackendVersion.default_container_port | `8000` |
| `{{host_port}}` | Deployment.service.hostPort | `8001` |
| `{{served_model_name}}` | Deployment.parameters.served_model_name | `qwen35-9b` |
| `{{max_model_len}}` | Deployment.parameters.max_model_len | `8192` |
| `{{gpu_memory_utilization}}` | Deployment.parameters.gpu_memory_utilization | `0.9` |
| `{{tensor_parallel_size}}` | Deployment.parameters.tensor_parallel_size | `2` |
| `{{assigned_gpu_indexes}}` | 调度分配的 GPU 索引列表 | `0,1,2,3` |
| `{{assigned_gpu_count}}` | 调度分配的 GPU 数量 | `4` |
| `{{deployment_name}}` | Deployment.name | `qwen35-9b-prod` |
| `{{instance_id}}` | 系统生成 | `abc123` |
| `{{node_id}}` | 调度分配的节点 | `node-01` |
| `{{node_ip}}` | 节点 IP | `192.168.1.100` |

### 13.2 未知变量处理

未知变量**必须返回 error**。不保留原样，不只 warning。

### 13.3 删除的语法

- `${VAR}` — 不支持
- `{{worker_ip}}` → 改为 `{{node_ip}}`
- `{{gpu_ids}}` → 改为 `{{assigned_gpu_indexes}}`
- `{{port}}` → 改为 `{{container_port}}`

---

## 14. 当前只支持 Docker

### 14.1 明确声明

```text
本阶段严格参考 GPUStack 的 container-first 模型运行方式。
vLLM / SGLang / llama.cpp 的命令行只表示容器内部 entrypoint / command / args。
LightAI Go 当前不在宿主机直接拉起长期运行的裸进程。
不实现 ProcessExecutor。
不实现 KubernetesExecutor。
```

### 14.2 BackendRuntime.runtime_type

保留字段 `runtime_type`，但当前只允许：

```text
runtime_type = docker
```

---

## 15. 多服务器 / 多 GPU 设计

### 15.1 场景 A：单节点多 GPU（第一版主路径）

```text
node-01
GPU 0,1,2,3
一个 ModelInstance
一个 Docker 容器
tensor_parallel_size = 4
CUDA_VISIBLE_DEVICES=0,1,2,3
```

**第一版必须实现**：

- manual node selection
- manual GPU selection
- single_node strategy
- multi-GPU（单节点内）
- replicas = 1

### 15.2 场景 B：多节点多副本（预留）

```text
replicas = 2

inst-001:
  node-01
  gpu 0,1,2,3

inst-002:
  node-02
  gpu 0,1,2,3
```

不是单模型跨节点并行，而是多个独立副本。

**文档预留**：`replicas`、`gpuCountPerReplica`、`placement strategy: spread / binpack`

**当前不实现**：自动调度和多副本启动。

### 15.3 场景 C：多节点单模型分布式并行（预留）

```text
一个超大模型
node-01 8 GPU
node-02 8 GPU
共同组成一个分布式推理实例
```

需要的复杂能力：
- Ray / torchrun / SGLang distributed
- leader / worker role
- master_addr / master_port
- 多节点通信端口
- 启动顺序
- 整体健康检查
- 失败回滚

**明确写入**：

```text
distributed.enabled = false
allowMultiNodeSingleReplica = false
```

当前不实现。

---

## 16. 数据库表设计

### 16.1 inference_backends

```sql
CREATE TABLE inference_backends (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    protocol_json TEXT NOT NULL DEFAULT '{}',
    default_version TEXT NOT NULL DEFAULT '',
    parameter_format TEXT NOT NULL DEFAULT 'space',
    common_parameters_json TEXT NOT NULL DEFAULT '[]',
    default_env_json TEXT NOT NULL DEFAULT '{}',
    is_builtin INTEGER NOT NULL DEFAULT 0,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### 16.2 backend_versions

```sql
CREATE TABLE backend_versions (
    id TEXT PRIMARY KEY,
    backend_id TEXT NOT NULL REFERENCES inference_backends(id),
    version TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    is_default INTEGER NOT NULL DEFAULT 0,
    default_entrypoint_json TEXT NOT NULL DEFAULT '[]',
    default_args_json TEXT NOT NULL DEFAULT '[]',
    default_backend_params_json TEXT NOT NULL DEFAULT '[]',
    parameter_defs_json TEXT NOT NULL DEFAULT '[]',
    health_check_json TEXT NOT NULL DEFAULT '{}',
    default_container_port INTEGER NOT NULL DEFAULT 8000,
    default_images_json TEXT NOT NULL DEFAULT '{}',
    env_json TEXT NOT NULL DEFAULT '{}',
    is_deprecated INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(backend_id, version)
);
```

### 16.3 backend_runtimes

```sql
CREATE TABLE backend_runtimes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    backend_id TEXT NOT NULL REFERENCES inference_backends(id),
    backend_version_id TEXT NOT NULL REFERENCES backend_versions(id),
    source_template_name TEXT NOT NULL DEFAULT '',
    vendor TEXT NOT NULL DEFAULT 'custom',
    runtime_type TEXT NOT NULL DEFAULT 'docker',
    image_name TEXT NOT NULL DEFAULT '',
    image_pull_policy TEXT NOT NULL DEFAULT 'if_not_present',
    entrypoint_override_json TEXT NOT NULL DEFAULT '[]',
    args_override_json TEXT NOT NULL DEFAULT '[]',
    default_env_json TEXT NOT NULL DEFAULT '{}',
    docker_json TEXT NOT NULL DEFAULT '{}',
    model_mount_json TEXT NOT NULL DEFAULT '{}',
    health_check_override_json TEXT NOT NULL DEFAULT '{}',
    is_builtin INTEGER NOT NULL DEFAULT 0,
    is_editable INTEGER NOT NULL DEFAULT 1,
    tenant_id TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(tenant_id, name)
);
```

### 16.4 node_runtime_overrides

```sql
CREATE TABLE node_runtime_overrides (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    tenant_id TEXT NOT NULL DEFAULT '',
    backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
    image_name TEXT NOT NULL DEFAULT '',
    image_pull_policy TEXT NOT NULL DEFAULT '',
    env_json TEXT NOT NULL DEFAULT '{}',
    docker_override_json TEXT NOT NULL DEFAULT '{}',
    model_root_host_path TEXT NOT NULL DEFAULT '',
    is_enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(node_id, backend_runtime_id)
);
```

### 16.5 model_deployments（新结构）

```sql
CREATE TABLE model_deployments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    model_artifact_id TEXT NOT NULL REFERENCES model_artifacts(id),
    backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
    replicas INTEGER NOT NULL DEFAULT 1,
    placement_json TEXT NOT NULL DEFAULT '{}',
    service_json TEXT NOT NULL DEFAULT '{}',
    parameters_json TEXT NOT NULL DEFAULT '{}',
    env_overrides_json TEXT NOT NULL DEFAULT '{}',
    desired_state TEXT NOT NULL DEFAULT 'stopped',
    status TEXT NOT NULL DEFAULT 'stopped',
    tenant_id TEXT NOT NULL,
    owner_id TEXT,
    created_by TEXT NOT NULL DEFAULT 'system',
    updated_by TEXT NOT NULL DEFAULT 'system',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### 16.6 model_instances

```sql
CREATE TABLE model_instances (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
    tenant_id TEXT NOT NULL DEFAULT '',
    replica_index INTEGER NOT NULL DEFAULT 0,
    node_id TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    assigned_gpus_json TEXT NOT NULL DEFAULT '[]',
    gpu_lease_ids_json TEXT NOT NULL DEFAULT '[]',
    host_port INTEGER NOT NULL DEFAULT 0,
    container_port INTEGER NOT NULL DEFAULT 0,
    current_run_plan_id TEXT REFERENCES resolved_run_plans(id),
    actual_state TEXT NOT NULL DEFAULT 'pending',
    desired_state TEXT NOT NULL DEFAULT 'running',
    container_id TEXT NOT NULL DEFAULT '',
    endpoint_url TEXT NOT NULL DEFAULT '',
    restart_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    started_at TEXT,
    stopped_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### 16.7 resolved_run_plans

```sql
CREATE TABLE resolved_run_plans (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
    tenant_id TEXT NOT NULL DEFAULT '',
    instance_id TEXT REFERENCES model_instances(id),
    backend_runtime_id TEXT NOT NULL REFERENCES backend_runtimes(id),
    node_runtime_override_id TEXT REFERENCES node_runtime_overrides(id),
    plan_json TEXT NOT NULL DEFAULT '{}',
    docker_preview TEXT NOT NULL DEFAULT '',
    input_hash TEXT NOT NULL DEFAULT '',
    plan_hash TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT 'system',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

---

## 17. API 设计

### 17.1 InferenceBackend API（第一版只读）

```
GET  /api/v1/inference-backends
GET  /api/v1/inference-backends/{id}
```

第一版不提供创建、修改、删除 Backend 的 Web/API 主流程。Backend 和 BackendVersion 通过配置文件导入。

### 17.2 BackendVersion API（第一版只读）

```
GET  /api/v1/inference-backends/{backend_id}/versions
GET  /api/v1/backend-versions/{id}
```

第一版只读。如果保留写接口，仅平台管理员/配置导入使用。

### 17.3 BackendRuntimeTemplate API（来自配置文件，不落库）

```
GET  /api/v1/backend-runtime-templates
GET  /api/v1/backend-runtime-templates/{name}
```

### 17.4 BackendRuntime API

```
GET    /api/v1/backend-runtimes
POST   /api/v1/backend-runtimes/from-template
GET    /api/v1/backend-runtimes/{id}
PATCH  /api/v1/backend-runtimes/{id}
DELETE /api/v1/backend-runtimes/{id}
```

### 17.5 NodeRuntimeOverride API

```
GET    /api/v1/node-runtime-overrides          ?node_id=&backend_runtime_id=
POST   /api/v1/node-runtime-overrides
GET    /api/v1/node-runtime-overrides/{id}
PATCH  /api/v1/node-runtime-overrides/{id}
DELETE /api/v1/node-runtime-overrides/{id}
```

### 17.6 ModelArtifact API（保持现有）

```
GET    /api/v1/model-artifacts
POST   /api/v1/model-artifacts
GET    /api/v1/model-artifacts/{id}
PATCH  /api/v1/model-artifacts/{id}
DELETE /api/v1/model-artifacts/{id}
```

### 17.7 ModelDeployment API

```
GET    /api/v1/model-deployments
POST   /api/v1/model-deployments
GET    /api/v1/model-deployments/{id}
PATCH  /api/v1/model-deployments/{id}
DELETE /api/v1/model-deployments/{id}
POST   /api/v1/model-deployments/{id}/start
POST   /api/v1/model-deployments/{id}/stop
```

Deployment 创建时使用：

```json
{
  "model_artifact_id": "...",
  "backend_runtime_id": "...",
  "placement_json": {...},
  "service_json": {...},
  "parameters_json": {...},
  "env_overrides_json": {...}
}
```

### 17.8 ModelInstance API

```
GET  /api/v1/model-instances            ?deployment_id=
GET  /api/v1/model-instances/{id}
GET  /api/v1/model-instances/{id}/logs
```

### 17.9 RunPlan API（新增）

```
POST /api/v1/run-plans/preview
GET  /api/v1/run-plans/{id}
GET  /api/v1/model-instances/{id}/run-plans
```

`POST /api/v1/run-plans/preview`：

- 输入 deployment 草稿
- 解析 BackendRuntime → BackendVersion → NodeRuntimeOverride
- 生成 ResolvedRunPlan preview
- 生成 docker_preview
- 返回 validation errors/warnings
- 不启动容器
- 第一版默认不落库（除非 `persist=true`）

---

## 18. RunPlan Resolver 设计

### 18.1 输入

```go
type ResolveInput struct {
    Backend             *InferenceBackend
    BackendVersion      *BackendVersion
    BackendRuntime      *BackendRuntime
    NodeRuntimeOverride *NodeRuntimeOverride  // 可为 nil
    Artifact            *ModelArtifact
    Deployment          *ModelDeployment
    InstanceID          string
    Node                *Node
    AssignedGPUs        []GPUDevice
}
```

### 18.2 解析流程

```text
1. 校验 BackendRuntime.runtime_type == "docker"。

2. 通过 BackendRuntime.backend_version_id 找到 BackendVersion。

3. 解析 image（优先级从高到低）：
   NodeRuntimeOverride.image_name
   > BackendRuntime.image_name
   > BackendVersion.defaultImages[BackendRuntime.vendor]
   > error: no image available

4. 解析 entrypoint：
   BackendRuntime.entrypoint_override_json
   > BackendVersion.default_entrypoint_json

5. 解析 args（final_args）：
   BackendVersion.default_args_json
   + BackendVersion.default_backend_params_json
   + BackendRuntime.args_override_json（第一版只 append）
   + Deployment.parameters_json 映射生成的参数
   只使用 {{var}} 替换模板。未知变量直接 error。

6. 合并 env（final_env，后者覆盖前者）：
   InferenceBackend.default_env_json
   + BackendVersion.env_json
   + BackendRuntime.default_env_json
   + NodeRuntimeOverride.env_json
   + ModelDeployment.env_overrides_json
   + GPU visible env（如 CUDA_VISIBLE_DEVICES）

7. 合并 docker spec：
   BackendRuntime.docker_json
   + NodeRuntimeOverride.docker_override_json

8. 生成模型挂载路径：
   使用 BackendRuntime.model_mount_json
   如果有 NodeRuntimeOverride.model_root_host_path，覆盖 host 端路径

9. 生成 ports：
   Deployment.service_json.hostPort : BackendVersion.default_container_port

10. 生成 health check：
    BackendRuntime.health_check_override_json
    > BackendVersion.health_check_json

11. 生成 docker_preview 字符串

12. 计算 input_hash = SHA256(所有输入)
13. 计算 plan_hash = SHA256(plan_json)
14. 返回 ResolvedRunPlan
```

### 18.3 GPU 可见变量

```go
if len(assignedGPUs) > 0 {
    switch vendor {
    case "nvidia":
        plan.Env["CUDA_VISIBLE_DEVICES"] = joinGPUIndexes(assignedGPUs)
    case "metax":
        plan.Env["CUDA_VISIBLE_DEVICES"] = joinGPUIndexes(assignedGPUs)
    }
}
```

---

## 19. Docker Executor 设计

### 19.1 接口

```go
type DockerExecutor interface {
    Start(plan ResolvedRunPlan) (*RuntimeInstance, error)
    Stop(instanceID string) error
    Inspect(instanceID string) (*RuntimeInstance, error)
    Logs(instanceID string, opts LogOptions) (*RuntimeLogs, error)
}
```

### 19.2 Agent 执行流程

```text
Agent 接收 AgentTask (task_type=model_instance_start, payload=ResolvedRunPlan)

1. 验证 ResolvedRunPlan.plan_hash 未被篡改
2. 检查镜像存在：docker image inspect {plan.Image}
   如果不存在且 image_pull_policy != "never"：docker pull {plan.Image}
3. 检查端口冲突
4. 检查 GPU 可用性
5. 检查挂载路径存在
6. 构建 docker create 参数（从 ResolvedRunPlan 直接映射）
7. docker create → docker start
8. 回报结果给 Server
```

### 19.3 Docker 参数映射

| ResolvedRunPlan 字段 | Docker 参数 |
|----------------------|-------------|
| Image | `docker create {image}` |
| ContainerName | `--name {name}` |
| Entrypoint | entrypoint override |
| Args | CMD override |
| Env | `-e KEY=VALUE` |
| HostPort:ContainerPort | `-p {host}:{container}` |
| Mounts | `-v {host}:{container}[:ro]` |
| Devices | `--device {host}:{container}` |
| Privileged | `--privileged` |
| IPCMode | `--ipc {mode}` |
| NetworkMode | `--network {mode}` |
| ShmSize | `--shm-size {size}` |
| Ulimits | `--ulimit {k}={v}` |
| SecurityOptions | `--security-opt {opt}` |

---

## 20. 状态机设计

### 20.1 ModelDeployment 状态

```
stopped → [start] → running → [stop] → stopped
                        ↓
                      error → [restart] → running
```

### 20.2 ModelInstance 状态

```
pending → initializing → starting → running → stopped
   ↓                                    ↓
 error ←────────────────────────── error
```

简化版（Phase 1，与 GPUStack 对比去掉 ANALYZING、DOWNLOADING、UNREACHABLE）。

---

## 21. Web 页面设计

### 21.1 新页面

| 页面 | 路由 | 操作 |
|------|------|------|
| 推理后端列表 | `/backends` | 只读查看 Backend + 版本列表 |
| 运行模板列表 | `/runtime-templates` | 只读查看模板，"从模板创建 Runtime" |
| 运行配置列表 | `/runtimes` | CRUD BackendRuntime |
| 节点覆盖列表 | `/node-overrides` | CRUD NodeRuntimeOverride |
| 模型工件列表 | `/models/artifacts` | CRUD ModelArtifact |
| 模型部署列表 | `/models/deployments` | CRUD + Dry Run + Start/Stop |
| 模型实例列表 | `/models/instances` | 查看 + 详情 + RunPlan + 日志 |

### 21.2 删除的旧页面

- `/runtime/environments` → 替换为 `/runtimes` + `/runtime-templates`
- `/runtime/templates` → 替换为 BackendRuntime + BackendVersion 概念
- 旧的 `/models/artifacts`、`/models/deployments`、`/models/instances` → 重写

### 21.3 部署页面交互流程

```
创建部署:
  1. 选择 ModelArtifact（下拉搜索）
  2. 选择 BackendRuntime（下拉，显示 vendor + backend + version）
  3. 自动显示 Runtime 信息（vendor, image, backend version）
  4. 选择 Node（只显示在线节点）
  5. 选择 GPU（该节点可用 GPU）
  6. 填写 HostPort, ServedModelName, MaxModelLen 等参数
  7. [Preview RunPlan] → 调用 POST /api/v1/run-plans/preview
  8. [Create Deployment]
  9. [Start] → 创建 Instance → 创建 RunPlan → 分配 AgentTask
```

---
### 21.4 Web 保存 roundtrip 验收要求

BackendRuntime、NodeRuntimeOverride、ModelArtifact、ModelDeployment 的创建/编辑/PATCH 操作必须满足：

1. **保存后 GET 校验**：保存成功后必须使用服务端返回对象或重新 GET 结果更新页面数据，禁止只在前端本地数组里拼接一行。
2. **刷新不丢失**：刷新页面后数据必须仍存在（已持久化到数据库）。
3. **保存失败显示错误**：后端校验失败时，前端必须展示后端返回的错误信息，不能静默失败。
4. **PATCH 精确更新**：PATCH 只更新提交的字段，未提交字段保持不变。

测试脚本要求：
- \`web/tests/apiSaveRoundtrip.test.mjs\` — Web 前端保存 roundtrip 自动化测试。
- \`test/e2e/model-runtime-api-roundtrip.sh\` — E2E API 保存 roundtrip 测试。

Phase 5 验收：不能只 \`npm run build\`，必须通过保存 roundtrip 测试。
Phase 6 验收：必须验证 Web 保存后刷新不丢失。


## 22. 旧代码和旧配置清理原则

### 22.1 旧代码文件（全部删除）

**Server（9 files）：**
- `internal/server/api/model_handlers.go`
- `internal/server/api/deployment_lifecycle.go`
- `internal/server/api/instance_state.go`
- `internal/server/api/lease.go`
- `internal/server/api/task_handlers.go`
- `internal/server/api/task_constants.go`
- `internal/server/api/sweep.go`
- `internal/server/api/resolve_helper.go`
- `internal/server/resolver/resolver.go`

**Web（10 files）：**
- 旧的 5 个页面（RuntimeEnvironments, RunTemplates, ModelArtifacts, ModelDeployments, ModelInstances）
- 旧的 5 个 API client（runtimeEnvironments, runTemplates, modelArtifacts, modelDeployments, modelInstances）

**配置：**
- `configs/templates/runtime/`（整个目录）
- `configs/templates/run/`（整个目录）
- `configs/templates/docker-images.json`
- `docs/templates-config.md`

### 22.2 保留的代码

- Agent Docker runtime（`internal/agent/runtime/`）— 全部保留
- Auth/RBAC — 全部保留
- Agent 注册/心跳 — node 管理部分保留
- 资源管理（GPU/Nodes）— 保留
- 指标/监控 — 保留
- Web 基础设施 — 保留
- DB 框架 — 保留

---

## 23. 权限与租户隔离设计

### 23.1 总体原则

所有新 API 必须接入现有 Auth / Session / CSRF / RBAC 机制。以下为硬约束：

1. **后端强制校验**：Web 菜单隐藏不是权限控制，后端必须强制校验权限。
2. **租户过滤**：所有租户级对象必须按 `tenant_id` 过滤查询。
3. **platform_admin 兜底**：platform_admin 可以跨租户访问所有对象。
4. **普通用户隔离**：tenant admin / operator / viewer 只能访问本租户对象。
5. **系统全局对象**：InferenceBackend / BackendVersion / BackendRuntimeTemplate 第一版为系统只读对象，全局可见，但普通用户不可修改。
6. **敏感脱敏**：RunPlan / env / docker_preview 可能包含宿主机路径、image、设备、环境变量，必须脱敏。

### 23.2 权限点设计

新增以下权限点。若现有 RBAC 已有等价权限则复用，否则新增：

| 权限 code | 说明 |
|-----------|------|
| `backend:read` | 读取推理后端 / 版本 / 运行时模板 |
| `backend:write` | 修改推理后端 / 版本（第一版不开放 Web/API） |
| `backend_runtime:read` | 读取 BackendRuntime / BackendRuntimeTemplate |
| `backend_runtime:write` | 创建/修改/删除 BackendRuntime |
| `node_runtime_override:read` | 读取 NodeRuntimeOverride |
| `node_runtime_override:write` | 创建/修改/删除 NodeRuntimeOverride |
| `model_artifact:read` | 读取 ModelArtifact |
| `model_artifact:write` | 创建/修改/删除 ModelArtifact |
| `model_deployment:read` | 读取 ModelDeployment |
| `model_deployment:write` | 创建/修改/删除 ModelDeployment |
| `model_deployment:start` | 启动 ModelDeployment |
| `model_deployment:stop` | 停止 ModelDeployment |
| `model_instance:read` | 读取 ModelInstance |
| `model_instance:logs` | 查看 ModelInstance 日志 |
| `run_plan:read` | 读取 ResolvedRunPlan |
| `run_plan:preview` | 预览 RunPlan（dry-run） |
| `gpu_lease:read` | 读取 GPU lease |
| `gpu_lease:write` | 创建/释放 GPU lease（由 start/stop 触发） |
| `agent_task:read` | 读取 AgentTask |
| `agent_task:write` | 创建 AgentTask（由 start/stop 触发） |

### 23.3 角色权限映射

| 角色 | 权限 |
|------|------|
| **viewer** | backend:read, backend_runtime:read, node_runtime_override:read, model_artifact:read, model_deployment:read, model_instance:read, run_plan:read, run_plan:preview |
| **operator** | viewer 全部 + backend_runtime:write, node_runtime_override:write, model_artifact:write, model_deployment:write, model_deployment:start, model_deployment:stop, model_instance:logs, gpu_lease:read, agent_task:read |
| **admin** | operator 全部 + 删除权限 + gpu_lease:write + agent_task:write |
| **platform_admin** | 全部权限。未来 Backend / BackendVersion 写接口仅限 platform_admin |

### 23.4 对象级租户规则

#### InferenceBackend / BackendVersion / BackendRuntimeTemplate

- 不含 `tenant_id`（系统全局对象）
- GET：登录用户可读
- POST/PATCH/DELETE：第一版不开放。如代码中保留写接口，仅限 platform_admin

#### BackendRuntime

BackendRuntime 是租户级对象。

| 操作 | 规则 |
|------|------|
| GET list | platform_admin 看全部；普通用户看本 tenant + builtin/global |
| GET detail | 只能看本 tenant 或 builtin/global |
| POST /from-template | 需要 `backend_runtime:write`；`tenant_id` = 当前用户 tenant_id |
| PATCH | 需要 `backend_runtime:write`；只能修改本 tenant；builtin/global 不可修改 |
| DELETE | 需要 `backend_runtime:write`；只能删除本 tenant；被 Deployment 引用时拒绝 |

#### NodeRuntimeOverride

- GET：需要 `node_runtime_override:read`。只能看当前 tenant 可访问节点上的 override。
- POST/PATCH/DELETE：需要 `node_runtime_override:write`。仅限本 tenant 有权使用的 node。platform_admin 跨租户兜底。

#### ModelArtifact

| 操作 | 规则 |
|------|------|
| GET | 只能看本 tenant artifact |
| POST | `tenant_id`/`owner_id` 由 Session 写入，不信任客户端 |
| PATCH/DELETE | 只能操作本 tenant artifact；被 Deployment 引用时拒绝删除 |

#### ModelDeployment

| 操作 | 规则 |
|------|------|
| GET | 只能看本 tenant deployment |
| POST | `model_artifact_id` 必须属本 tenant；`backend_runtime_id` 必须属本 tenant 或 builtin/global；`placement_json.nodeId` 必须是本 tenant 可访问节点 |
| PATCH | 只能修改本 tenant；running 状态下禁止修改 artifact/runtime/placement |
| DELETE | 只能删除本 tenant；running 状态下拒绝，要求先 stop |

#### Start / Stop

- `POST /start`：需要 `model_deployment:start`。deployment/artifact/runtime/node/GPU 必须属本 tenant。instance/run_plan/lease/agent_task 关联同一 tenant。
- `POST /stop`：需要 `model_deployment:stop`。deployment 必须属本 tenant。

#### ModelInstance

- GET：需要 `model_instance:read`。只能查看本 tenant deployment 产生的 instance。
- logs：需要 `model_instance:logs`。只能查看本 tenant instance 日志。

#### ResolvedRunPlan

- GET：需要 `run_plan:read`。只能查看本 tenant deployment/instance 的 RunPlan。
- preview：需要 `run_plan:preview`。必须校验 artifact/runtime/node/GPU 都属于当前 tenant。

ResolvedRunPlan.plan_json 和 docker_preview 可能包含宿主机路径、设备路径、image、env，**不得跨租户泄露**。

---

## 24. 敏感字段脱敏

### 24.1 需要脱敏的字段

| 位置 | 字段 |
|------|------|
| `InferenceBackend.default_env_json` | 含敏感 key 的 env |
| `BackendVersion.env_json` | 含敏感 key 的 env |
| `BackendRuntime.default_env_json` | 含敏感 key 的 env |
| `NodeRuntimeOverride.env_json` | 含敏感 key 的 env |
| `ModelDeployment.env_overrides_json` | 含敏感 key 的 env |
| `ResolvedRunPlan.plan_json.env` | 含敏感 key 的 env |
| `docker_preview` 字符串 | `-e KEY=VALUE` 中的 VALUE |

### 24.2 敏感 key 规则

key 包含以下关键词（不区分大小写）时，value 默认显示为 `****`：

```
token, key, secret, password, passwd, pwd,
ak, sk, credential, access_key, secret_key,
api_key, apikey, authorization, bearer,
hf_token, dashscope_api_key, openai_api_key
```

### 24.3 脱敏层级

```
API 返回 → Server 层脱敏 → 返回 `****`
Web 展示 → 默认脱敏显示
docker_preview → 字符串层面脱敏 `-e` 参数值
复制 docker_preview → 使用脱敏版本
```

第一版：所有用户（含 platform_admin）API 返回脱敏值，Web 展示脱敏值。未来如需"显示明文"，需实现 `secret:read` 权限 + 二次确认。

### 24.4 Web 展示要求

- RunPlan 详情页和 docker_preview 默认脱敏
- 复制 docker_preview 时也使用脱敏版本
- 如需"显示明文"，二次确认 + 权限检查

---

## 25. API 权限矩阵

| 端点 | Method | 权限 |
|------|--------|------|
| `/api/v1/inference-backends` | GET | `backend:read` |
| `/api/v1/inference-backends/{id}` | GET | `backend:read` |
| `/api/v1/inference-backends/{id}/versions` | GET | `backend:read` |
| `/api/v1/backend-versions/{id}` | GET | `backend:read` |
| `/api/v1/backend-runtime-templates` | GET | `backend_runtime:read` |
| `/api/v1/backend-runtime-templates/{name}` | GET | `backend_runtime:read` |
| `/api/v1/backend-runtimes` | GET | `backend_runtime:read` |
| `/api/v1/backend-runtimes/from-template` | POST | `backend_runtime:write` |
| `/api/v1/backend-runtimes/{id}` | GET | `backend_runtime:read` |
| `/api/v1/backend-runtimes/{id}` | PATCH | `backend_runtime:write` |
| `/api/v1/backend-runtimes/{id}` | DELETE | `backend_runtime:write` |
| `/api/v1/node-runtime-overrides` | GET | `node_runtime_override:read` |
| `/api/v1/node-runtime-overrides` | POST | `node_runtime_override:write` |
| `/api/v1/node-runtime-overrides/{id}` | GET | `node_runtime_override:read` |
| `/api/v1/node-runtime-overrides/{id}` | PATCH | `node_runtime_override:write` |
| `/api/v1/node-runtime-overrides/{id}` | DELETE | `node_runtime_override:write` |
| `/api/v1/model-artifacts` | GET | `model_artifact:read` |
| `/api/v1/model-artifacts` | POST | `model_artifact:write` |
| `/api/v1/model-artifacts/{id}` | GET | `model_artifact:read` |
| `/api/v1/model-artifacts/{id}` | PATCH | `model_artifact:write` |
| `/api/v1/model-artifacts/{id}` | DELETE | `model_artifact:write` |
| `/api/v1/model-deployments` | GET | `model_deployment:read` |
| `/api/v1/model-deployments` | POST | `model_deployment:write` |
| `/api/v1/model-deployments/{id}` | GET | `model_deployment:read` |
| `/api/v1/model-deployments/{id}` | PATCH | `model_deployment:write` |
| `/api/v1/model-deployments/{id}` | DELETE | `model_deployment:write` |
| `/api/v1/model-deployments/{id}/start` | POST | `model_deployment:start` |
| `/api/v1/model-deployments/{id}/stop` | POST | `model_deployment:stop` |
| `/api/v1/model-instances` | GET | `model_instance:read` |
| `/api/v1/model-instances/{id}` | GET | `model_instance:read` |
| `/api/v1/model-instances/{id}/logs` | GET | `model_instance:logs` |
| `/api/v1/run-plans/preview` | POST | `run_plan:preview` |
| `/api/v1/run-plans/{id}` | GET | `run_plan:read` |
| `/api/v1/model-instances/{id}/run-plans` | GET | `run_plan:read` |

---

## 26. 数据库 tenant_id 补充建议

### 26.1 当前状态

| 表 | tenant_id 现状 | 建议 |
|----|---------------|------|
| `inference_backends` | 无 | 保持无（全局对象） |
| `backend_versions` | 无 | 保持无（全局对象） |
| `backend_runtimes` | 有 | 保持 |
| `node_runtime_overrides` | 通过 node_id 追溯 | **建议新增 tenant_id**，减少 join |
| `model_artifacts` | 有 | 保持 |
| `model_deployments` | 有 | 保持 |
| `model_instances` | 无 | **建议新增 tenant_id**，减少权限过滤的复杂 join |
| `resolved_run_plans` | 无 | **建议新增 tenant_id**，方便审计和权限过滤 |
| `gpu_leases` | 无 | **建议新增 tenant_id**，方便权限过滤 |
| `agent_tasks` | 无 | **建议新增 tenant_id**，方便权限过滤 |

### 26.2 推荐方案

第一版直接给以下表增加 `tenant_id NOT NULL DEFAULT ''`：

```sql
-- model_instances
ALTER TABLE model_instances ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';

-- resolved_run_plans  
ALTER TABLE resolved_run_plans ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';

-- gpu_leases
ALTER TABLE gpu_leases ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';

-- agent_tasks
ALTER TABLE agent_tasks ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';

-- node_runtime_overrides
ALTER TABLE node_runtime_overrides ADD COLUMN tenant_id TEXT NOT NULL DEFAULT '';
```

创建时从 deployment 或 node 关联写入 tenant_id。查询时直接 `WHERE tenant_id = ?`，避免跨表 JOIN。

### 26.3 更新后的表 SQL

`model_instances`、`resolved_run_plans`、`gpu_leases`、`agent_tasks`、`node_runtime_overrides` 的创建 SQL 需包含 `tenant_id` 字段。详见 §16 各表定义。

---

## 27. 附：MetaX vLLM Docker 示例

### Backend
```yaml
apiVersion: lightai/v1
kind: InferenceBackend
metadata:
  name: vllm
spec:
  displayName: vLLM
  protocol: { type: openai-compatible, modelsPath: /v1/models, chatCompletionsPath: /v1/chat/completions }
  parameterFormat: space
  isBuiltIn: true
```

### BackendVersion
```yaml
apiVersion: lightai/v1
kind: BackendVersion
metadata:
  backend: vllm
  version: "0.8.5"
spec:
  defaultEntrypoint: [vllm, serve]
  defaultArgs:
    - "{{model_container_path}}"
    - "--host"
    - "0.0.0.0"
    - "--port"
    - "{{container_port}}"
    - "--served-model-name"
    - "{{served_model_name}}"
  defaultContainerPort: 8000
  defaultImages:
    nvidia: "vllm/vllm-openai:v0.8.5"
    metax: "registry.local/vllm-openai:v0.8.5-metax"
  healthCheck:
    path: /v1/models
    expectedStatus: 200
    startupTimeoutSeconds: 120
```

### BackendRuntime
```yaml
apiVersion: lightai/v1
kind: BackendRuntime
metadata:
  name: vllm-metax-docker
spec:
  backendId: vllm
  backendVersionId: vllm-0.8.5
  vendor: metax
  runtimeType: docker
  imageName: "registry.local/vllm-openai:v0.8.5-metax"
  docker:
    privileged: true
    ipcMode: host
    shmSize: "10g"
    devices:
      - hostPath: /dev/mxcd
        containerPath: /dev/mxcd
      - hostPath: /dev/dri
        containerPath: /dev/dri
  defaultEnv:
    VLLM_USE_MODELSCOPE: "true"
```

### ResolvedRunPlan（生成结果）
```json
{
  "image": "0d307f1665d3",
  "container_name": "lightai-qwen35-9b-abc123",
  "entrypoint": ["vllm", "serve"],
  "args": [
    "/models/Qwen35-9B",
    "--host", "0.0.0.0",
    "--port", "8000",
    "--served-model-name", "qwen35-9b",
    "--tensor-parallel-size", "4",
    "--max-model-len", "8192",
    "--gpu-memory-utilization", "0.9"
  ],
  "env": {
    "CUDA_VISIBLE_DEVICES": "0,1,2,3",
    "VLLM_USE_MODELSCOPE": "true"
  },
  "privileged": true,
  "ipc_mode": "host",
  "shm_size": "10g",
  "devices": [
    {"host_path": "/dev/mxcd", "container_path": "/dev/mxcd"},
    {"host_path": "/dev/dri", "container_path": "/dev/dri"}
  ],
  "mounts": [
    {"host_path": "/data/part2/MX-C500/model/Qwen35-9B", "container_path": "/models/Qwen35-9B", "readonly": true}
  ],
  "host_port": 8001,
  "container_port": 8000,
  "gpu_device_ids": ["0", "1", "2", "3"],
  "health_check": {
    "path": "/v1/models",
    "expected_status": 200,
    "startup_timeout_seconds": 120,
    "interval_seconds": 2,
    "timeout_seconds": 5
  },
  "input_hash": "sha256:abc123...",
  "plan_hash": "sha256:def456..."
}
```

> 注意：此例中 image 来自 NodeRuntimeOverride 的覆盖（`0d307f1665d3`），model_root_host_path 也来自 NodeRuntimeOverride（`/data/part2/MX-C500/model`），devices 来自 BackendRuntime.docker_json。
