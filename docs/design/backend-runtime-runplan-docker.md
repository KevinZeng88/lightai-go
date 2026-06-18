# LightAI Go 模型运行设计：Backend / BackendVersion / BackendRuntime / RunPlan

> Status: CURRENT
> Last reviewed: 2026-06-18
> Scope: Current BackendRuntime / RunPlan / Docker design
> Read order: See `docs/CURRENT.md`

> 当前路径：`docs/design/backend-runtime-runplan-docker.md`
> 状态：设计沉淀稿
> 适用范围：模型资源管理、Backend Catalog、节点 Runtime、Docker 启动、GPU Lease、单机/多副本/未来分布式运行
> 更新时间：2026-06-17

---

## 1. 总体结论

LightAI Go 后续模型运行链路应从旧的 `RuntimeEnv / RuntimeTemplate` 口径切换到：

```text
Backend
  ↓
BackendVersion
  ↓
BackendRuntime
  ↓
NodeBackendRuntime
  ↓
DeploymentPlan / ServicePlan
  ↓
RunPlanGroup
  ↓
NodeRunPlan / InstanceRunPlan
  ↓
Agent Executor
```

模型资源侧对应：

```text
ModelArtifact
  ↓
ModelLocation
```

其中：

```text
Backend：推理后端类型，例如 vLLM、SGLang、llama.cpp、Ollama。
BackendVersion：后端软件版本、能力、参数 schema，原则上保持硬件无关。
BackendRuntime：某个 BackendVersion 的运行模板，硬件类别感知，例如 NVIDIA Docker、MetaX Docker、CPU Docker。
NodeBackendRuntime：某个节点上该 Runtime 是否实际可用，例如 Docker image 是否存在、driver/toolkit 是否满足。
ModelArtifact：逻辑模型定义。
ModelLocation：该模型在某个节点上的实际文件或目录位置，是 ModelArtifact 的 location 属性，数据库可单独存储。
DeploymentPlan：用户部署意图。
RunPlanGroup：一次部署解析后的编排组，支持 single、replicated、distributed 预留。
NodeRunPlan：单个节点上的最终执行计划，由 Agent 执行。
```

核心原则：

> **模型和 BackendRuntime 的定义保持可复用；模型位置和 Runtime 可用性必须节点化；RunPlan 只允许在目标节点同时找到可用 ModelLocation、ready NodeBackendRuntime 和可分配 GPU 后生成。**

---

## 2. 设计目标

### 2.1 主要目标

1. 替代旧的 `RuntimeEnv / RuntimeTemplate` 主线。
2. 内置常用 Backend Catalog，避免现场从零配置 vLLM、SGLang、llama.cpp 等。
3. 支持 NVIDIA、MetaX、CPU 等不同硬件运行形态。
4. 支持模型在不同节点有不同路径。
5. 支持 Runtime 在不同节点有不同 image 状态。
6. 支持自动扫描模型 metadata、大小、checksum、能力标签。
7. 支持手工确认不同 fingerprint 的模型位置属于同一个逻辑模型。
8. 支持单机单实例作为第一阶段主路径。
9. 预留多副本和跨机分布式推理。
10. 支持 Docker 启动、健康检查、GPU Lease、失败回收和审计日志。

### 2.2 第一阶段非目标

第一阶段不强制实现：

```text
跨机分布式推理完整编排
复杂网关和计费
镜像跨节点传输的完整产品化
K8s Runtime
所有后端的深度参数适配
```

但数据模型和接口应预留。

---

## 3. 核心对象关系

### 3.1 模型侧

```text
ModelArtifact
  1 ── N ModelLocation
```

含义：

```text
ModelArtifact：逻辑模型。
ModelLocation：该模型在某个节点上的实际位置。
```

用户界面上，`ModelLocation` 可以表现为 `ModelArtifact.locations` 属性；数据库实现上建议单独成表，便于调度、扫描、校验和审计。

---

### 3.2 后端侧

```text
Backend
  1 ── N BackendVersion
          1 ── N BackendRuntime
                    1 ── N NodeBackendRuntime
```

含义：

```text
Backend：后端类型。
BackendVersion：软件版本和能力。
BackendRuntime：运行模板，描述某类硬件/运行环境怎么跑。
NodeBackendRuntime：某节点上该 Runtime 是否实际 ready。
```

---

### 3.3 计划侧

```text
DeploymentPlan / ServicePlan
  1 ── N RunPlanGroup
            1 ── N NodeRunPlan
```

第一阶段单机时：

```text
DeploymentPlan(mode=single)
  ↓
RunPlanGroup(mode=single, desired_count=1)
  ↓
NodeRunPlan(node_id=local-node)
```

多副本时：

```text
DeploymentPlan(mode=replicated, replicas=3)
  ↓
RunPlanGroup(mode=replicated, desired_count=3)
  ↓
NodeRunPlan(replica-1, node-01)
NodeRunPlan(replica-2, node-02)
NodeRunPlan(replica-3, node-03)
```

未来跨机分布式时：

```text
DeploymentPlan(mode=distributed)
  ↓
RunPlanGroup(mode=distributed, world_size=N)
  ↓
NodeRunPlan(rank-0, role=head, node-01)
NodeRunPlan(rank-1, role=worker, node-02)
```

---

## 4. Backend / BackendVersion / BackendRuntime 分层

### 4.1 Backend

`Backend` 表示推理后端类型。

示例：

```text
llamacpp
vllm
sglang
ollama
openai-compatible-external
```

它回答：

```text
用哪类推理后端？
支持什么模型格式？
对外暴露什么协议？
默认健康检查是什么？
```

典型字段：

```text
id
name
slug
description
supported_model_formats_json
protocols_json
default_params_schema_json
default_health_check_json
status
created_at
updated_at
```

示例：

```yaml
id: backend.llamacpp
slug: llamacpp
name: llama.cpp
supported_model_formats:
  - gguf
protocols:
  - openai-compatible
default_health_check:
  type: http
  path: /v1/models
```

---

### 4.2 BackendVersion

`BackendVersion` 表示某个 Backend 的软件版本、能力和参数 schema。

原则：

> **BackendVersion 原则上保持硬件无关。**

它可以描述软件层兼容能力，但不应该绑定具体节点、具体 GPU、具体 Docker image 是否存在。

典型字段：

```text
id
backend_id
name
version
description
supported_model_formats_json
supported_protocols_json
params_schema_json
capabilities_json
compatibility_json
status
created_at
updated_at
```

示例：

```yaml
id: backend-version.vllm.openai-latest
backend: vllm
name: vLLM OpenAI Server
version: latest
supported_model_formats:
  - huggingface
  - safetensors
supported_protocols:
  - openai-compatible
params_schema:
  tensor_parallel_size:
    type: integer
    default: 1
  gpu_memory_utilization:
    type: number
    default: 0.9
  max_model_len:
    type: integer
```

#### 4.2.1 硬件专用 fork 的处理

有些版本本身可能是厂商适配版，例如：

```text
llama.cpp-metax-adapted
vllm-metax-patched
```

这种情况下，`BackendVersion` 可以表达“这是某个厂商适配版软件”，但仍然不要绑定具体节点或具体 GPU。

正确边界：

```text
BackendVersion：可以表达软件 fork / 软件兼容性。
BackendRuntime：表达该版本在某类硬件和运行环境上怎么运行。
NodeRunPlan：绑定具体节点和具体 GPU。
```

---

### 4.3 BackendRuntime

`BackendRuntime` 是关键概念，表示某个 `BackendVersion` 的具体运行模板。

它回答：

```text
这个后端版本用什么方式运行？
用 Docker 还是 process？
用哪个镜像？
命令是什么？
默认参数是什么？
容器端口是什么？
模型如何挂载？
GPU 设备如何暴露？
健康检查如何做？
```

典型字段：

```text
id
backend_id
backend_version_id
name
runtime_type
runner_type
vendor
accelerator_kind
image_ref
image_pull_policy
entrypoint_json
command_json
default_args_json
default_env_json
ports_json
mount_policy_json
device_policy_json
resource_defaults_json
health_check_json
params_mapping_json
distributed_profile_json
managed_by
status
created_at
updated_at
```

示例：

```yaml
id: runtime.llamacpp.nvidia-docker
backend: llamacpp
backend_version: llama-cpp-server
name: llama.cpp NVIDIA Docker
runner_type: docker
vendor: nvidia
image_ref: ghcr.io/ggerganov/llama.cpp:server
command:
  - llama-server
default_args:
  - --host
  - 0.0.0.0
  - --port
  - "8000"
params_mapping:
  ctx_size: --ctx-size
  n_gpu_layers: --n-gpu-layers
generated_args:
  model: "--model {{model_path_in_container}}"
ports:
  - container_port: 8000
    protocol: tcp
mount_policy:
  model:
    mount_type: directory
    container_mount_dir: /models
    readonly: true
device_policy:
  type: nvidia-container-toolkit
  visible_devices_from: gpu_device.index
health_check:
  type: http
  method: GET
  path: /v1/models
  expected_status: 200
```

#### 4.3.1 BackendRuntime 与硬件的关系

`BackendRuntime` 应硬件类别感知，但不绑定具体硬件。

可以绑定：

```text
vendor = nvidia / metax / cpu
runner_type = docker
device_policy = nvidia-container-toolkit / metax / none
requires_driver
requires_toolkit
container_mount_dir
```

不应绑定：

```text
node_id
gpu_device_id
gpu_index
具体某台机器上的 image_id
```

这些由 `NodeBackendRuntime` 和 `NodeRunPlan` 处理。

---

### 4.4 NodeBackendRuntime

`NodeBackendRuntime` 表示某个节点上某个 BackendRuntime 的实际可用状态。

它回答：

```text
这个节点上有没有这个 Docker image？
Docker 是否可用？
NVIDIA / MetaX driver 是否可用？
container toolkit 是否满足？
该 Runtime 在这个节点是否 ready？
```

典型字段：

```text
id
backend_runtime_id
node_id
runner_type
image_ref
image_id
image_digest
image_present
docker_available
driver_version
toolkit_version
device_check_json
status
last_checked_at
created_at
updated_at
```

状态建议：

```text
ready
missing_image
driver_mismatch
toolkit_missing
unsupported_device
invalid
unknown
```

示例：

```yaml
NodeBackendRuntime:
  backend_runtime_id: runtime.vllm.nvidia-docker
  node_id: node-01
  image_ref: vllm/vllm-openai:latest
  image_id: sha256:abc
  image_digest: sha256:def
  image_present: true
  docker_available: true
  driver_version: "610.43.02"
  status: ready
```

#### 4.4.1 为什么不让 BackendRuntime 直接节点绑定

如果直接让 `BackendRuntime` 节点绑定，会出现：

```text
runtime-vllm-node-01
runtime-vllm-node-02
runtime-vllm-node-03
```

同一个模板被复制多份，升级、对比、审计都会复杂。

推荐：

```text
BackendRuntime：vLLM NVIDIA Docker 模板。
NodeBackendRuntime：node-01 上该模板 ready，node-02 上该模板 missing_image。
```

---

## 5. ModelArtifact / ModelLocation

### 5.1 ModelArtifact

`ModelArtifact` 表示逻辑模型。

典型字段：

```text
id
tenant_id
name
format
family
parameter_size
quantization
capabilities_json
canonical_fingerprint_json
identity_policy
metadata_json
created_at
updated_at
```

示例：

```yaml
ModelArtifact:
  id: model.qwen3.5-9b-q4-k-m
  name: Qwen3.5-9B-Q4_K_M
  format: gguf
  family: qwen
  parameter_size: 9B
  quantization: Q4_K_M
  capabilities:
    - chat
    - completion
```

---

### 5.2 ModelLocation

`ModelLocation` 表示该模型在某个节点上的实际位置。

从用户理解上，它是 `ModelArtifact` 的一个 location 属性；从实现上建议单独存储。

典型字段：

```text
id
model_artifact_id
node_id
path_type
model_root
relative_path
absolute_path
size_bytes
checksum
manifest_digest
discovered_metadata_json
match_status
verification_status
manual_override
override_reason
override_by
override_at
last_scanned_at
last_error
created_at
updated_at
```

其中：

```text
path_type = file / directory
model_root = 节点上的模型根目录
relative_path = 模型相对路径
absolute_path = model_root + relative_path
```

示例一，GGUF 文件：

```yaml
ModelLocation:
  model_artifact_id: model.qwen3.5-9b-q4-k-m
  node_id: node-01
  path_type: file
  model_root: /home/kzeng/models
  relative_path: Qwen3.5-9B-Q4_K_M.gguf
  absolute_path: /home/kzeng/models/Qwen3.5-9B-Q4_K_M.gguf
  size_bytes: 5616076800
  checksum: sha256:...
  verification_status: verified
  match_status: exact
```

示例二，第二个节点路径不同：

```yaml
ModelLocation:
  model_artifact_id: model.qwen3.5-9b-q4-k-m
  node_id: node-02
  path_type: file
  model_root: /data/models
  relative_path: qwen.gguf
  absolute_path: /data/models/qwen.gguf
  verification_status: manually_accepted
  match_status: manual_attested
```

---

### 5.3 添加模型流程

第一次配置模型时，应从节点和路径开始：

```text
1. 先添加或选择 Agent 节点。
2. 选择模型目录或模型文件。
3. Agent 在该节点扫描模型。
4. 自动识别格式、大小、checksum、manifest、metadata、能力标签。
5. 用户可以修改模型名称、能力标签、说明等。
6. Server 创建 ModelArtifact。
7. Server 创建第一条 ModelLocation。
```

GGUF 文件可扫描：

```text
format
size_bytes
checksum
gguf metadata
architecture
n_params
n_ctx_train
quantization
tokenizer info
capabilities guess
```

HuggingFace 目录可扫描：

```text
config.json
tokenizer.json
tokenizer_config.json
generation_config.json
*.safetensors
*.bin
model.safetensors.index.json
```

目录模型建议生成：

```text
manifest_digest = sha256(file path + size + checksum list)
```

---

### 5.4 添加第二个节点模型位置

流程：

```text
1. 选择已有 ModelArtifact。
2. 选择第二个节点。
3. 选择该节点上的模型路径。
4. Agent 扫描 fingerprint。
5. Server 与已有模型 fingerprint 对比。
6. 精确一致则直接加入。
7. 高概率一致则提示 warning 后加入。
8. 不一致时允许管理员手工确认为同一模型。
```

系统不应只支持严格 checksum 一致，因为现场可能出现：

```text
文件名不同
目录中多了 README/cache 文件
mtime 不同
大模型全量 checksum 太慢
不同来源同步但用户确认是同一模型
国产适配版本 metadata 有差异
```

因此应支持：

```text
exact_match
probable_match
manual_attested
mismatch
```

对应状态：

```text
verified
warning
manually_accepted
failed
```

---

### 5.5 手工确认为同一模型

必须允许管理员强行把某个 Location 设为同一模型，但要有审计和风险标记。

字段建议：

```text
manual_override: true
override_reason
override_by
override_at
match_status: manual_attested
verification_status: manually_accepted
```

RunPlan 使用时：

```text
exact / verified：默认允许。
probable / warning：允许，但记录 warning。
manual_attested / manually_accepted：允许，但 RunPlan 记录 warning。
mismatch 未确认：不允许。
missing / changed：不允许，除非重新确认。
```

审计事件：

```text
model_location.added
model_location.verified
model_location.mismatch_detected
model_location.manual_attested
model_location.changed
model_location.missing
```

---

## 6. 挂载目录设计

### 6.1 核心规则

不同节点宿主机路径可以不同，但容器内路径应尽量统一。

例如：

```text
node-01 host path: /home/kzeng/models/Qwen.gguf
node-02 host path: /data/models/Qwen.gguf
```

RunPlan on node-01：

```text
-v /home/kzeng/models:/models:ro
--model /models/Qwen.gguf
```

RunPlan on node-02：

```text
-v /data/models:/models:ro
--model /models/Qwen.gguf
```

如果文件名也不同：

```text
node-01: /home/kzeng/models/Qwen3.5-9B-Q4_K_M.gguf
node-02: /data/models/qwen.gguf
```

则：

```text
node-01:
  -v /home/kzeng/models:/models:ro
  --model /models/Qwen3.5-9B-Q4_K_M.gguf

node-02:
  -v /data/models:/models:ro
  --model /models/qwen.gguf
```

### 6.2 分工

```text
ModelLocation：
  记录宿主机模型目录和相对路径。

BackendRuntime：
  记录容器内挂载策略。

NodeRunPlan：
  记录本次启动最终 Docker mount 和 model_path_in_container。
```

BackendRuntime 不应写死：

```text
/home/kzeng/models
```

它只应该写：

```text
container_mount_dir: /models
readonly: true
mount_type: directory
model_arg: --model
```

NodeRunPlan 生成：

```text
host_mount_dir = ModelLocation.model_root
container_mount_dir = BackendRuntime.mount_policy.model.container_mount_dir
model_path_in_container = container_mount_dir + "/" + ModelLocation.relative_path
```

---

## 7. 内置 Backend Catalog

### 7.1 设计原则

产品应内置常用 Backend、BackendVersion、BackendRuntime 配置，避免现场从零配置推理后端。

内置应覆盖：

```text
Backend:
  - vllm
  - sglang
  - llamacpp
  - ollama

BackendVersion:
  - vLLM 主流版本
  - SGLang 主流版本
  - llama.cpp 主流 server 版本
  - Ollama 主流版本

BackendRuntime:
  - NVIDIA Docker Runtime
  - MetaX Docker Runtime
  - CPU Docker Runtime
```

现场用户主要配置：

```text
节点
模型目录 / 模型文件
Docker image 是否可用
少量运行参数
```

不应要求用户从零配置：

```text
后端类型
后端版本
Docker 命令
默认参数
设备暴露策略
健康检查规则
```

---

### 7.2 推荐目录结构

```text
configs/backend-catalog/
  catalog.yaml

  backends/
    llamacpp.yaml
    vllm.yaml
    sglang.yaml
    ollama.yaml

  versions/
    llamacpp/
      llama-cpp-server.yaml
    vllm/
      vllm-openai-latest.yaml
      vllm-openai-0.9.yaml
    sglang/
      sglang-openai-latest.yaml
    ollama/
      ollama-latest.yaml

  runtimes/
    llamacpp/
      nvidia-docker.yaml
      metax-docker.yaml
      cpu-docker.yaml
    vllm/
      nvidia-docker.yaml
      metax-docker.yaml
    sglang/
      nvidia-docker.yaml
      metax-docker.yaml
    ollama/
      nvidia-docker.yaml
      cpu-docker.yaml
```

现场覆盖：

```text
data/backend-catalog.d/user/
  <backend>/
    user-version.yaml
```

### 7.3 Seed 策略

采用：

```text
内置 system Backend Catalog 文件
        ↓
启动 / reload 读取 system + user catalog 文件
        ↓
upsert 到 DB projection
        ↓
按稳定 slug / id 合并
```

要求：

```text
内置 catalog 通过 Go embed 打包进二进制。
外部目录可新增或覆盖 catalog 项。
seed 幂等，重复启动不重复插入。
用户修改过的配置不被升级覆盖。
每个预置项有稳定 ID 或 slug。
每个预置项记录 catalog_version、checksum、managed_by。
```

建议字段：

```text
managed_by: system / user
source: embedded / file / api
catalog_version
checksum
```

被实例引用的 Runtime 不应被破坏性覆盖。需要升级时，应新增 Runtime 版本或新 revision。

---

## 8. Runtime 配置与节点启用

### 8.1 第一次启用 Runtime

流程：

```text
1. 选择节点。
2. 系统检测 Docker、GPU vendor、driver、container toolkit。
3. 展示推荐 Runtime。
4. 用户选择 Backend，例如 llama.cpp / vLLM / SGLang。
5. 系统推荐 BackendRuntime。
6. 用户选择默认 image、手工输入 image，或从本机已有 image 选择。
7. Agent 执行 docker image inspect。
8. 校验 image、driver、toolkit、设备策略。
9. 创建或更新 NodeBackendRuntime。
```

如果使用系统预置模板：

```text
BackendRuntime：系统内置。
NodeBackendRuntime：当前节点启用并 ready。
```

如果手工输入 image：

```text
复制系统 BackendRuntime 为 user-managed Runtime
或创建新的 user-managed BackendRuntime
然后创建 NodeBackendRuntime
```

避免污染系统内置 catalog。

---

### 8.2 添加第二个节点 Runtime

第二个节点可以：

```text
直接 pull 同一个 image
手工输入 image
从节点已有 image 选择
从第一个节点复制 image
导入离线 runtime bundle
```

实现优先级建议：

```text
第一阶段：手工输入 image + 本节点 inspect。
第二阶段：从节点已有 image 中选择。
第三阶段：支持节点间 docker save/load 复制。
第四阶段：支持内网 registry / 离线 runtime bundle。
```

节点间复制流程可预留：

```text
1. node-01 docker save image
2. 压缩
3. 通过 Server 中转或 Agent 直连传输
4. node-02 docker load
5. docker image inspect
6. 校验 digest
7. 创建 NodeBackendRuntime
```

注意：

```text
大镜像传输慢
需要断点续传
需要 checksum 校验
需要权限控制
需要避免 Server 磁盘被打满
```

---

## 9. 三层计划模型

### 9.1 内部三层

推荐固定为：

```text
DeploymentPlan / ServicePlan
        ↓
RunPlanGroup
        ↓
NodeRunPlan / InstanceRunPlan
```

### 9.2 DeploymentPlan / ServicePlan

描述用户意图。

它回答：

```text
我要部署哪个模型？
用哪个后端？
要几个副本？
每个副本需要多少 GPU？
是否暴露服务入口？
调度策略是什么？
```

示例：

```yaml
DeploymentPlan:
  model_artifact_id: model.qwen3.5-9b-q4-k-m
  backend_runtime_id: runtime.llamacpp.nvidia-docker
  mode: single
  replicas: 1
  resource_request:
    gpu_count: 1
  placement_policy:
    node_id: node-01
  service_policy:
    expose_openai_api: true
```

这一层不包含具体 Docker mount、GPU index、容器名。

---

### 9.3 RunPlanGroup

描述一次部署解析后的编排结果。

它回答：

```text
这次部署拆成几个执行单元？
它们是什么关系？
整体状态是什么？
```

字段建议：

```text
id
deployment_plan_id
mode: single / replicated / distributed
desired_count
ready_count
status
group_config_json
created_at
updated_at
```

---

### 9.4 NodeRunPlan / InstanceRunPlan

Agent 真正执行的节点级计划。

原则：

> **一个 NodeRunPlan = 一个节点上的一个 executor task。**

字段建议：

```text
id
group_id
deployment_plan_id
instance_id
node_id
agent_id
role
rank
world_size
model_artifact_id
model_location_id
backend_id
backend_version_id
backend_runtime_id
node_backend_runtime_id
gpu_lease_ids_json
run_plan_json
command_preview
status
created_at
updated_at
```

`run_plan_json` 中应包含：

```text
model_path_in_container
docker image
container_name
mounts
ports
env
args
devices
gpu mapping
health check
warnings
```

---

### 9.5 单机、多副本、分布式的统一表达

单机：

```text
DeploymentPlan(mode=single)
  ↓
RunPlanGroup(mode=single, desired_count=1)
  ↓
NodeRunPlan(node-01)
```

多副本：

```text
DeploymentPlan(mode=replicated, replicas=3)
  ↓
RunPlanGroup(mode=replicated, desired_count=3)
  ↓
NodeRunPlan(replica-1, node-01)
NodeRunPlan(replica-2, node-02)
NodeRunPlan(replica-3, node-03)
```

未来分布式：

```text
DeploymentPlan(mode=distributed)
  ↓
RunPlanGroup(mode=distributed, world_size=2)
  ↓
NodeRunPlan(rank-0, role=head, node-01)
NodeRunPlan(rank-1, role=worker, node-02)
```

第一阶段只需要实现：

```text
single
replicated 可预留或简单实现
```

`distributed` 先预留字段，不急着完整实现。

---

## 10. RunPlan 生成校验

### 10.1 普通单机 / 多副本校验

每个 NodeRunPlan 生成前，必须校验目标节点：

```text
1. ModelArtifact exists。
2. 目标节点存在对应 ModelLocation。
3. ModelLocation 状态允许运行。
4. BackendRuntime exists。
5. 目标节点存在对应 NodeBackendRuntime。
6. NodeBackendRuntime status = ready。
7. BackendRuntime 支持模型格式。
8. NodeBackendRuntime vendor 与目标节点 GPU vendor 匹配。
9. GPU 资源可分配。
10. GPU Lease 创建成功。
11. 模型路径存在且可读。
12. 端口可用。
13. Docker image present。
14. BackendRuntime 健康检查配置有效。
```

如果失败，应停在 resolving failed，不应下发 Agent。

---

### 10.2 分布式预留校验

未来跨机分布式推理需要整体校验：

```text
所有目标节点在线。
所有节点都有 ModelLocation。
所有节点都有 ready NodeBackendRuntime。
BackendRuntime supports_distributed = true。
网络互通。
通信端口可用。
driver/runtime 版本兼容。
GPU 拓扑满足要求。
```

BackendRuntime 应声明：

```text
supports_multi_replica
supports_distributed
distributed_modes
required_ports
required_network
```

普通 llama.cpp：

```text
supports_multi_replica: true
supports_distributed: false
```

某些 vLLM / SGLang 分布式 runtime：

```text
supports_multi_replica: true
supports_distributed: true
distributed_modes:
  - ray
  - torchrun
```

---

## 11. Docker NodeRunPlan 生成

### 11.1 输入

```text
ModelArtifact
ModelLocation on target node
Backend
BackendVersion
BackendRuntime
NodeBackendRuntime on target node
runtime_params
GPUDevice
GPULease
service policy
```

### 11.2 输出

```yaml
NodeRunPlan:
  node_id: node-01
  model_location:
    model_root: /home/kzeng/models
    relative_path: Qwen3.5-9B-Q4_K_M.gguf
  node_backend_runtime:
    image_ref: ghcr.io/ggerganov/llama.cpp:server
    image_digest: sha256:...
  docker:
    mounts:
      - host_path: /home/kzeng/models
        container_path: /models
        readonly: true
    image: ghcr.io/ggerganov/llama.cpp:server
    args:
      - llama-server
      - --host
      - 0.0.0.0
      - --port
      - "8000"
      - --model
      - /models/Qwen3.5-9B-Q4_K_M.gguf
      - --ctx-size
      - "4096"
      - --n-gpu-layers
      - "999"
    gpus:
      - internal_gpu_id: ...
        runtime_visible_device: "0"
  health_check:
    method: GET
    url: http://127.0.0.1:8002/v1/models
```

等价命令预览：

```bash
docker run -d \
  --name lightai-inst-xxx \
  --gpus '"device=0"' \
  -v /home/kzeng/models:/models:ro \
  -p 8002:8000 \
  ghcr.io/ggerganov/llama.cpp:server \
  llama-server \
  --host 0.0.0.0 \
  --port 8000 \
  --model /models/Qwen3.5-9B-Q4_K_M.gguf \
  --ctx-size 4096 \
  --n-gpu-layers 999
```

---

## 12. GPU Lease 与设备映射

GPU 必须走 lease。

生命周期：

```text
reserved → activated → released
```

流程：

```text
1. Server 选择 GPU。
2. 创建 gpu_lease，状态 reserved。
3. NodeRunPlan 记录 lease 和 GPU 设备。
4. Agent 启动容器。
5. 健康检查通过后 lease 变为 activated。
6. 停止、失败或删除时 lease 变为 released。
```

NVIDIA 场景必须避免把平台内部 GPU ID 直接传给 Docker。

错误：

```text
--gpus "device=59ebe637-c0af-4a7a-9279-bccd170d55ee"
```

正确：

```text
gpu_devices.id
  ↓
gpu_devices.index
  ↓
--gpus '"device=0"'
```

MetaX 等国产 GPU 通过 vendor adapter 或 device_policy 转换，不硬编码 NVIDIA。

---

## 13. 健康检查

健康检查来自 BackendRuntime。

OpenAI-compatible 后端优先：

```text
GET /v1/models
```

健康判定：

```text
HTTP 200
响应体可解析
models 或 data 非空
```

注意：

```text
Docker container started ≠ 模型服务可用
端口 opened ≠ 模型加载完成
/v1/models 通过后才进入 running
```

已知 llama.cpp 测试：

```bash
curl http://127.0.0.1:8002/v1/models
```

---

## 14. 状态机

### 14.1 实例状态

```text
pending
resolving
reserved
claimed
starting
health_checking
running
stopping
stopped
failed
unknown
```

成功路径：

```text
pending → resolving → reserved → claimed → starting → health_checking → running
```

失败路径：

```text
resolving → failed
reserved → failed → release lease
starting → failed → release lease
health_checking → failed → release lease
```

要求：

```text
failed / starting / pending / unknown 状态必须可 stop、delete、cleanup。
不能出现实例卡死后无法清理。
失败必须释放 lease。
失败必须记录错误详情。
```

---

### 14.2 ModelLocation 状态

```text
verified
warning
manually_accepted
missing
changed
unverified
unknown
failed
```

---

### 14.3 NodeBackendRuntime 状态

```text
ready
missing_image
driver_mismatch
toolkit_missing
unsupported_device
invalid
unknown
```

---

## 15. 审计与日志

关键审计事件：

```text
backend_catalog.seeded
backend_runtime.created
node_backend_runtime.enabled
node_backend_runtime.checked
model_artifact.created
model_location.added
model_location.verified
model_location.manual_attested
deployment_plan.created
run_plan_group.created
node_run_plan.created
gpu_lease.reserved
agent.task.claimed
docker.container.created
docker.container.started
model_instance.health_check.started
model_instance.health_check.passed
gpu_lease.activated
model_instance.running
model_instance.failed
gpu_lease.released
docker.container.stopped
docker.container.removed
```

失败记录至少包括：

```text
instance_id
deployment_plan_id
run_plan_group_id
node_run_plan_id
tenant_id
node_id
backend_id
backend_version_id
backend_runtime_id
node_backend_runtime_id
model_artifact_id
model_location_id
container_name
container_id
exit_code
stderr_tail
health_check_error
lease_ids
```

---

## 16. 安全要求

```text
1. 模型路径必须校验，防止越权挂载。
2. 默认只读挂载模型目录。
3. BackendRuntime 镜像来源应可控。
4. privileged 默认关闭。
5. devices、cap_add、ipc_mode、network_mode 等高危配置需要权限控制。
6. 环境变量中的密钥必须脱敏。
7. 租户只能使用有权访问的模型、后端和资源。
8. RunPlan 生成前必须完成 RBAC 校验。
9. Agent 只接受已授权任务。
10. 端口映射必须检测冲突。
11. Docker volume 不允许挂载任意系统目录。
12. 删除实例时必须清理容器、任务和租约。
13. 手工确认模型一致必须写审计日志。
14. 节点间复制 image 必须校验 checksum/digest。
```

---

## 17. 数据模型建议

### 17.1 model_artifacts

```sql
CREATE TABLE model_artifacts (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  format TEXT NOT NULL,
  family TEXT,
  parameter_size TEXT,
  quantization TEXT,
  capabilities_json TEXT NOT NULL DEFAULT '[]',
  canonical_fingerprint_json TEXT NOT NULL DEFAULT '{}',
  identity_policy TEXT NOT NULL DEFAULT 'strict_digest',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

### 17.2 model_locations

```sql
CREATE TABLE model_locations (
  id TEXT PRIMARY KEY,
  model_artifact_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  path_type TEXT NOT NULL,
  model_root TEXT NOT NULL,
  relative_path TEXT NOT NULL,
  absolute_path TEXT NOT NULL,
  size_bytes INTEGER,
  checksum TEXT,
  manifest_digest TEXT,
  discovered_metadata_json TEXT NOT NULL DEFAULT '{}',
  match_status TEXT NOT NULL DEFAULT 'unverified',
  verification_status TEXT NOT NULL DEFAULT 'unverified',
  manual_override INTEGER NOT NULL DEFAULT 0,
  override_reason TEXT,
  override_by TEXT,
  override_at TEXT,
  last_scanned_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (model_artifact_id) REFERENCES model_artifacts(id)
);
```

### 17.3 backends

```sql
CREATE TABLE backends (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT,
  supported_model_formats_json TEXT NOT NULL DEFAULT '[]',
  protocols_json TEXT NOT NULL DEFAULT '[]',
  default_params_schema_json TEXT NOT NULL DEFAULT '{}',
  default_health_check_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

### 17.4 backend_versions

```sql
CREATE TABLE backend_versions (
  id TEXT PRIMARY KEY,
  backend_id TEXT NOT NULL,
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  description TEXT,
  supported_model_formats_json TEXT NOT NULL DEFAULT '[]',
  supported_protocols_json TEXT NOT NULL DEFAULT '[]',
  params_schema_json TEXT NOT NULL DEFAULT '{}',
  capabilities_json TEXT NOT NULL DEFAULT '{}',
  compatibility_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (backend_id) REFERENCES backends(id)
);
```

### 17.5 backend_runtimes

```sql
CREATE TABLE backend_runtimes (
  id TEXT PRIMARY KEY,
  backend_id TEXT NOT NULL,
  backend_version_id TEXT NOT NULL,
  name TEXT NOT NULL,
  runtime_type TEXT NOT NULL,
  runner_type TEXT NOT NULL,
  vendor TEXT,
  accelerator_kind TEXT,
  image_ref TEXT,
  image_pull_policy TEXT NOT NULL DEFAULT 'if_not_present',
  entrypoint_json TEXT NOT NULL DEFAULT '[]',
  command_json TEXT NOT NULL DEFAULT '[]',
  default_args_json TEXT NOT NULL DEFAULT '[]',
  default_env_json TEXT NOT NULL DEFAULT '{}',
  ports_json TEXT NOT NULL DEFAULT '[]',
  mount_policy_json TEXT NOT NULL DEFAULT '{}',
  device_policy_json TEXT NOT NULL DEFAULT '{}',
  resource_defaults_json TEXT NOT NULL DEFAULT '{}',
  health_check_json TEXT NOT NULL DEFAULT '{}',
  params_mapping_json TEXT NOT NULL DEFAULT '{}',
  distributed_profile_json TEXT NOT NULL DEFAULT '{}',
  managed_by TEXT NOT NULL DEFAULT 'system',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (backend_id) REFERENCES backends(id),
  FOREIGN KEY (backend_version_id) REFERENCES backend_versions(id)
);
```

### 17.6 node_backend_runtimes

```sql
CREATE TABLE node_backend_runtimes (
  id TEXT PRIMARY KEY,
  backend_runtime_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  runner_type TEXT NOT NULL,
  image_ref TEXT,
  image_id TEXT,
  image_digest TEXT,
  image_present INTEGER NOT NULL DEFAULT 0,
  docker_available INTEGER NOT NULL DEFAULT 0,
  driver_version TEXT,
  toolkit_version TEXT,
  device_check_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'unknown',
  last_checked_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (backend_runtime_id) REFERENCES backend_runtimes(id)
);
```

### 17.7 deployment_plans

```sql
CREATE TABLE deployment_plans (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  model_artifact_id TEXT NOT NULL,
  backend_id TEXT,
  backend_version_id TEXT,
  backend_runtime_id TEXT,
  mode TEXT NOT NULL DEFAULT 'single',
  replicas INTEGER NOT NULL DEFAULT 1,
  runtime_params_json TEXT NOT NULL DEFAULT '{}',
  resource_request_json TEXT NOT NULL DEFAULT '{}',
  placement_policy_json TEXT NOT NULL DEFAULT '{}',
  service_policy_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

### 17.8 run_plan_groups

```sql
CREATE TABLE run_plan_groups (
  id TEXT PRIMARY KEY,
  deployment_plan_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  desired_count INTEGER NOT NULL DEFAULT 1,
  ready_count INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'pending',
  group_config_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (deployment_plan_id) REFERENCES deployment_plans(id)
);
```

### 17.9 node_run_plans

```sql
CREATE TABLE node_run_plans (
  id TEXT PRIMARY KEY,
  run_plan_group_id TEXT NOT NULL,
  deployment_plan_id TEXT NOT NULL,
  instance_id TEXT,
  node_id TEXT NOT NULL,
  role TEXT,
  rank INTEGER,
  world_size INTEGER,
  model_artifact_id TEXT NOT NULL,
  model_location_id TEXT NOT NULL,
  backend_id TEXT NOT NULL,
  backend_version_id TEXT NOT NULL,
  backend_runtime_id TEXT NOT NULL,
  node_backend_runtime_id TEXT NOT NULL,
  gpu_lease_ids_json TEXT NOT NULL DEFAULT '[]',
  run_plan_json TEXT NOT NULL,
  command_preview TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (run_plan_group_id) REFERENCES run_plan_groups(id),
  FOREIGN KEY (deployment_plan_id) REFERENCES deployment_plans(id)
);
```

---

## 18. API 建议

### 18.1 模型

```text
GET    /api/v1/model-artifacts
POST   /api/v1/model-artifacts
GET    /api/v1/model-artifacts/{id}
PATCH  /api/v1/model-artifacts/{id}
DELETE /api/v1/model-artifacts/{id}

POST   /api/v1/model-artifacts/discover
POST   /api/v1/model-artifacts/{id}/locations
POST   /api/v1/model-artifacts/{id}/locations/{location_id}/rescan
POST   /api/v1/model-artifacts/{id}/locations/{location_id}/attest
DELETE /api/v1/model-artifacts/{id}/locations/{location_id}
```

### 18.2 Backend Catalog

```text
GET    /api/v1/backends
GET    /api/v1/backend-versions
GET    /api/v1/backend-runtimes
POST   /api/v1/backend-runtimes
PATCH  /api/v1/backend-runtimes/{id}
```

### 18.3 节点 Runtime

```text
GET    /api/v1/nodes/{node_id}/backend-runtimes
POST   /api/v1/nodes/{node_id}/backend-runtimes/enable
POST   /api/v1/nodes/{node_id}/backend-runtimes/check
POST   /api/v1/nodes/{node_id}/backend-runtimes/pull-image
POST   /api/v1/backend-runtimes/copy-image
```

### 18.4 部署与计划

```text
POST   /api/v1/deployments
GET    /api/v1/deployments
GET    /api/v1/deployments/{id}
POST   /api/v1/deployments/{id}/start
POST   /api/v1/deployments/{id}/stop
DELETE /api/v1/deployments/{id}

GET    /api/v1/deployments/{id}/run-plan-groups
GET    /api/v1/node-run-plans/{id}
GET    /api/v1/node-run-plans/{id}/command-preview
GET    /api/v1/node-run-plans/{id}/events
```

---

## 19. Web 产品体验

### 19.1 第一阶段隐藏复杂度

三层计划模型是内部架构，不应暴露给初期用户。

第一阶段用户看到：

```text
添加模型
  ↓
选择运行后端
  ↓
选择 GPU / 自动选择
  ↓
填写少量参数
  ↓
启动
```

默认值：

```text
mode = single
replicas = 1
node = 当前节点
runtime = 系统推荐
gpu = 自动选择
port = 自动分配
```

### 19.2 主菜单建议

第一阶段主菜单建议：

```text
节点
模型
实例
```

高级设置中再放：

```text
Backend Catalog
BackendVersion
BackendRuntime
NodeBackendRuntime
RunPlan
GPU Lease
```

### 19.3 添加模型界面

```text
1. 选择节点，默认当前 Agent。
2. 选择模型文件或目录。
3. 系统扫描。
4. 自动填充名称、格式、大小、checksum、能力标签。
5. 用户确认或修改。
```

展示：

```text
模型名称
模型格式
路径
大小
能力标签
校验状态
所在节点列表
```

### 19.4 启用 Runtime 界面

```text
1. 选择节点。
2. 系统检测 Docker/GPU。
3. 推荐 llama.cpp / vLLM / SGLang。
4. 用户选择默认镜像、手工输入镜像或本机已有镜像。
5. 检查通过后启用。
```

### 19.5 启动实例界面

极简字段：

```text
模型
后端
节点
GPU：自动 / 手动
端口：自动 / 手动
参数：默认 / 高级
```

高级展开：

```text
BackendVersion
BackendRuntime
NodeBackendRuntime
宿主机端口
ctx-size
n-gpu-layers
tensor-parallel-size
gpu-memory-utilization
环境变量
命令预览
```

---

## 20. 第一阶段实施建议

### Phase A：Catalog 与节点 Runtime

```text
1. 建 backends / backend_versions / backend_runtimes。
2. 内置 Backend Catalog。
3. seed 幂等。
4. 增加 node_backend_runtimes。
5. 支持 Docker image inspect。
6. 支持启用本节点 Runtime。
```

### Phase B：模型扫描与位置管理

```text
1. 建 model_artifacts / model_locations。
2. Agent 支持扫描文件和目录。
3. 自动解析 GGUF / HuggingFace metadata。
4. 支持 checksum / manifest_digest。
5. 支持手工确认一致。
6. Web 上把 locations 作为模型属性展示。
```

### Phase C：单机 RunPlan

```text
1. 建 deployment_plans / run_plan_groups / node_run_plans。
2. 单机模式自动生成三层计划。
3. 校验 ModelLocation + NodeBackendRuntime。
4. 创建 GPU Lease。
5. 生成 Docker NodeRunPlan。
6. 生成命令预览。
```

### Phase D：Agent DockerExecutor

```text
1. Agent claim NodeRunPlan。
2. Docker create/start。
3. 健康检查 /v1/models。
4. 上报 running / failed。
5. 停止、删除、cleanup。
6. Lease activated/released。
```

### Phase E：多副本预留或简单实现

```text
1. DeploymentPlan replicas。
2. RunPlanGroup desired_count。
3. 多个 NodeRunPlan。
4. 部分失败时 Deployment degraded。
5. Gateway/endpoint 后续接入。
```

---

## 21. 验收标准

### 21.1 Catalog

```text
1. 干净安装后自动内置 vLLM、SGLang、llama.cpp、Ollama。
2. 自动内置 NVIDIA / MetaX / CPU BackendRuntime。
3. seed 幂等，不重复插入。
4. 用户自定义 Runtime 不被升级覆盖。
```

### 21.2 Runtime

```text
1. 节点可检测 Docker。
2. 节点可检测 GPU vendor、driver、toolkit。
3. 可选择或输入 image。
4. docker image inspect 成功后 NodeBackendRuntime ready。
5. image tag 相同但 digest 不同时能记录差异。
```

### 21.3 模型

```text
1. 添加模型必须先选节点和文件/目录。
2. Agent 能扫描 GGUF 文件。
3. Agent 能扫描 HuggingFace 目录。
4. 能生成 checksum / manifest_digest。
5. 能自动填充模型大小、格式、metadata、能力标签。
6. 能添加第二节点位置。
7. checksum 不一致时允许管理员手工确认。
8. 手工确认必须写审计。
```

### 21.4 RunPlan

```text
1. 目标节点没有 ModelLocation 时不能生成 NodeRunPlan。
2. 目标节点没有 ready NodeBackendRuntime 时不能生成 NodeRunPlan。
3. BackendRuntime 不支持模型格式时不能生成 NodeRunPlan。
4. GPU 不足时不能生成 NodeRunPlan。
5. NodeRunPlan 中 host mount 来自 ModelLocation。
6. NodeRunPlan 中 container mount 来自 BackendRuntime。
7. NodeRunPlan 中 model_path_in_container 正确。
8. GPU 传参使用厂商可识别 ID，不使用平台内部 GPU UUID。
9. EquivalentCommandPreview 与 NodeRunPlan 一致。
```

### 21.5 执行

```text
1. Agent 能领取 NodeRunPlan。
2. Docker 容器能创建和启动。
3. /v1/models 健康检查通过后进入 running。
4. lease reserved → activated。
5. 停止后 lease released。
6. 容器启动失败进入 failed。
7. 健康检查失败进入 failed。
8. failed / pending / starting / unknown 均可 cleanup。
```

---

---

## 22. 产品目标：选择即可运行、启停、监控与日志

本设计的产品目标不是让用户手工拼 Docker 命令，而是让常见场景做到：

```text
选择 Backend
选择 BackendVersion
选择 BackendRuntime
选择 ModelArtifact / ModelLocation
选择节点 / GPU，默认可自动
点击启动
```

系统应自动完成：

```text
1. 校验目标节点是否有模型位置。
2. 校验目标节点是否有可用 Runtime。
3. 校验 Docker image、driver、toolkit、GPU/NPU 设备。
4. 生成 DeploymentPlan。
5. 生成 RunPlanGroup。
6. 生成 NodeRunPlan。
7. 预留 GPU Lease。
8. 生成 docker run 预览。
9. Agent 创建并启动容器。
10. 执行健康检查。
11. 进入 running 或 failed。
```

第一阶段用户理想体验：

```text
1. 在“节点”页启用 Runtime。
2. 在“模型”页添加模型位置。
3. 在“实例”页选择模型和后端。
4. 大部分参数使用内置模板默认值。
5. 点击启动即可运行 Docker。
6. 页面可启停实例。
7. 页面可查看实例状态。
8. 页面可查看健康检查结果。
9. 页面可查看 Docker 日志。
10. 页面可查看最终 docker run 命令预览。
```

也就是说，Backend Catalog、BackendVersion、BackendRuntime 的内置模板必须足够完整，尤其是 NVIDIA、MetaX/沐曦、Huawei/昇腾这类硬件的 Docker 参数，不能要求现场每次重新输入。

---

## 23. 实例运行后的操作能力

每个实例或 NodeRunPlan 详情页应提供以下能力。

### 23.1 启停控制

```text
Start
Stop
Restart
Delete
Cleanup
```

要求：

```text
1. Stop 应停止容器，并释放 GPU Lease。
2. Restart 应复用或重新生成 NodeRunPlan，按策略处理端口和 GPU。
3. Delete 应清理实例、容器、任务和 lease。
4. Cleanup 必须能处理 failed / pending / starting / unknown 状态。
```

### 23.2 状态检查

页面应展示：

```text
DeploymentPlan status
RunPlanGroup status
NodeRunPlan status
Container status
Health check status
GPU Lease status
Runtime status
ModelLocation status
```

容器状态至少包括：

```text
container_id
container_name
image
created_at
started_at
exit_code
restart_count
ports
mounts
devices
```

### 23.3 Docker 日志

实例详情页必须可以查看 Docker 日志。

建议 API：

```text
GET /api/v1/node-run-plans/{id}/logs?tail=200
GET /api/v1/node-run-plans/{id}/logs?since=...
```

能力要求：

```text
1. 支持 tail 最近 N 行。
2. 支持 stdout/stderr 合并展示。
3. 支持刷新。
4. 支持复制日志。
5. 日志中敏感 env 应脱敏。
6. failed 时自动展示最后 N 行日志。
```

### 23.4 Docker inspect / 运行详情

高级模式可以展示：

```text
docker inspect 摘要
最终 env
最终 args
最终 mounts
最终 devices
最终 docker_options
最终 health_check
GPU/NPU 映射
command preview
```

### 23.5 监控

实例详情页应展示：

```text
CPU
Memory
GPU/NPU 利用率
GPU/NPU 显存
容器运行时长
健康检查延迟
请求指标，后续接入网关后展示
```

第一阶段可以先展示节点/GPU 指标与实例状态关联；后续再接入容器级 cgroup 指标和网关请求指标。

---

## 24. Runtime 参数配置页面设计

### 24.1 总原则

BackendRuntime 和 NodeBackendRuntime 的页面配置目标是：

```text
默认模板足够好，用户几乎不用改；
需要现场调试时，所有关键 Docker 参数可见、可勾选、可修改、可预览。
```

不要把所有参数隐藏在 JSON 中，也不要要求用户直接编辑完整 YAML。

推荐形态：

```text
高风险单值项：独立开关 + 输入框
同类列表项：启用框 + 多行输入框
Custom 参数：单独区域
最终结果：docker run 命令预览
```

### 24.2 高风险单值项

这些参数应独立展示，不放在多行输入框中：

```text
privileged
ipc_mode
uts_mode
network_mode
pid_mode
shm_size
```

示例：

```text
☑ privileged = true       高风险
☑ ipc_mode = host         高风险
☑ uts_mode = host         高风险
☐ network_mode = host     高风险
☑ shm_size = 100gb
```

要求：

```text
1. 每项有 enabled。
2. enabled=false 时不进入 NodeRunPlan。
3. 高风险值必须醒目标记。
4. 保存时 Server 做 schema 校验。
```

### 24.3 同类列表项

以下字段使用“启用框 + 多行输入框”：

```text
devices
optional_devices
group_add
security_opt
cap_add
device_cgroup_rules
extra_hosts
ulimits
env
extra_mounts
custom_args
custom_env
custom_docker_options
```

示例，devices：

```text
☑ Devices

/dev/dri
/dev/mxcd
/dev/infiniband
```

示例，security_opt：

```text
☑ Security Opt

seccomp=unconfined
apparmor=unconfined
```

示例，env：

```text
☑ Environment Variables

LIGHTAI_VENDOR=metax
CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}
```

示例，extra_mounts：

```text
☑ Extra Mounts

/usr/local/dcmi:/usr/local/dcmi:rw
/usr/local/bin/npu-smi:/usr/local/bin/npu-smi:ro
```

规则：

```text
1. 勾选该类别时，多行内容进入 NodeRunPlan。
2. 未勾选时，整块忽略。
3. 空行忽略。
4. 重复行去重。
5. 格式错误时保存失败并提示。
6. command preview 只包含 enabled=true 的配置。
```

### 24.4 Custom 参数区

必须提供 Custom 参数区，解决不同镜像、不同版本、不同现场的临时差异。

```text
Custom Args
Custom Env
Custom Docker Options
```

#### Custom Args

用于追加到后端启动命令：

```text
☑ Custom Args

--trust-remote-code
--max-num-seqs 64
--max-num-batched-tokens 4096
--dtype float16
```

第一阶段可以按“一行一个参数片段”处理，原样追加到 args 尾部；后续可升级为结构化表格。

#### Custom Env

```text
☑ Custom Env

VLLM_USE_MODELSCOPE=true
MACA_VISIBLE_DEVICES={{vendor_visible_devices}}
```

#### Custom Docker Options

专家模式展示，用于无法归类的 Docker 参数：

```text
☑ Custom Docker Options

--add-host host.docker.internal:host-gateway
--log-driver json-file
--log-opt max-size=100m
```

要求：

```text
1. custom_docker_options 默认隐藏在专家模式。
2. 保存时尽量解析和校验。
3. 无法安全解析的参数应提示风险。
4. 最终必须体现在 command preview 中。
```

### 24.5 Runtime 修改策略

系统内置 Runtime 默认只读。

```text
managed_by=system：只读
managed_by=user：可编辑
```

用户需要修改系统模板时：

```text
复制为自定义 Runtime
```

NodeBackendRuntime 允许节点级覆盖：

```text
image_ref
enabled blocks
custom args
custom env
custom docker options
```

---

## 25. BackendRuntime 配置结构：enabled block

BackendRuntime / NodeBackendRuntime 中 Docker 参数建议采用 enabled block 结构。

示例：

```yaml
docker_options:
  privileged:
    enabled: true
    value: true

  ipc_mode:
    enabled: true
    value: host

  uts_mode:
    enabled: true
    value: host

  network_mode:
    enabled: false
    value: host

  shm_size:
    enabled: true
    value: 100gb

  devices:
    enabled: true
    value: |
      /dev/dri
      /dev/mxcd
      /dev/infiniband

  group_add:
    enabled: true
    value: |
      video

  security_opt:
    enabled: true
    value: |
      seccomp=unconfined
      apparmor=unconfined

  ulimits:
    enabled: true
    value: |
      memlock=-1

  env:
    enabled: true
    value: |
      LIGHTAI_VENDOR=metax
      CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}

  extra_mounts:
    enabled: false
    value: ""

custom:
  args:
    enabled: true
    value: |
      --trust-remote-code
      --max-num-seqs 64

  env:
    enabled: false
    value: ""

  docker_options:
    enabled: false
    value: ""
```

Resolver 处理规则：

```text
1. 读取 BackendRuntime 默认 enabled blocks。
2. 应用 NodeBackendRuntime 覆盖。
3. 应用 Deployment runtime_params。
4. 应用 Custom 参数。
5. 过滤 enabled=false 的 block。
6. 解析多行 block。
7. 替换模板变量，例如 {{vendor_visible_devices}}。
8. 去重。
9. 检测冲突。
10. 生成 NodeRunPlan。
11. 生成 command preview。
```

---

## 26. MetaX / 沐曦 Runtime 模板设计

### 26.1 本地实测启动命令

已有本地实测命令：

```bash
docker run -itd \
  --device=/dev/dri \
  --device=/dev/mxcd \
  --device=/dev/infiniband \
  --group-add video \
  --name vllm111 \
  --uts=host \
  --ipc=host \
  --privileged=true \
  --security-opt seccomp=unconfined \
  --security-opt apparmor=unconfined \
  --shm-size '100gb' \
  --ulimit memlock=-1 \
  -p 8001:8000 \
  -v /data/part2/MX-C500/model:/models \
  -e CUDA_VISIBLE_DEVICES=6,7 \
  0d307f1665d3
```

该命令应作为 MetaX vLLM Runtime 模板的重要依据。

### 26.2 参数抽象原则

上述参数不应硬编码在 DockerExecutor 中，而应进入：

```text
BackendRuntime.docker_options
NodeBackendRuntime override
NodeRunPlan.run_plan_json
```

其中：

```text
/data/part2/MX-C500/model:/models
```

不应作为固定 Runtime 参数，而应由：

```text
ModelLocation.model_root
+
BackendRuntime.mount_policy.model.container_mount_dir
```

生成。

```text
CUDA_VISIBLE_DEVICES=6,7
```

不应写死，而应由 GPU Lease 解析出的 vendor visible devices 生成：

```text
CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}
```

### 26.3 vLLM MetaX Docker Runtime 示例

```yaml
id: runtime.vllm.metax-docker
backend: vllm
backend_version: vllm-openai-latest
name: vLLM MetaX Docker
runner_type: docker
runtime_type: docker
vendor: metax
accelerator_kind: gpu
image_ref: lightai/vllm-metax:latest
image_pull_policy: if_not_present
command:
  - python
  - -m
  - vllm.entrypoints.openai.api_server
default_args:
  - --host
  - 0.0.0.0
  - --port
  - "8000"
params_mapping:
  tensor_parallel_size: --tensor-parallel-size
  gpu_memory_utilization: --gpu-memory-utilization
  max_model_len: --max-model-len
  max_num_seqs: --max-num-seqs
  max_num_batched_tokens: --max-num-batched-tokens
  dtype: --dtype
  trust_remote_code: --trust-remote-code
generated_args:
  model: "--model {{model_path_in_container}}"
ports:
  - container_port: 8000
    protocol: tcp
mount_policy:
  model:
    mount_type: directory
    container_mount_dir: /models
    readonly: true
device_policy:
  type: metax
  visible_devices_from: vendor_adapter
  device_id_source: gpu_device.index
  device_env:
    name: CUDA_VISIBLE_DEVICES
    value_from: vendor_visible_devices
docker_options:
  privileged:
    enabled: true
    value: true
  uts_mode:
    enabled: true
    value: host
  ipc_mode:
    enabled: true
    value: host
  shm_size:
    enabled: true
    value: 100gb
  devices:
    enabled: true
    value: |
      /dev/dri
      /dev/mxcd
      /dev/infiniband
  optional_devices:
    enabled: false
    value: |
      /dev/mem
  group_add:
    enabled: true
    value: |
      video
  security_opt:
    enabled: true
    value: |
      seccomp=unconfined
      apparmor=unconfined
  ulimits:
    enabled: true
    value: |
      memlock=-1
  env:
    enabled: true
    value: |
      LIGHTAI_VENDOR=metax
      CUDA_VISIBLE_DEVICES={{vendor_visible_devices}}
custom:
  args:
    enabled: false
    value: ""
  env:
    enabled: false
    value: ""
  docker_options:
    enabled: false
    value: ""
health_check:
  type: http
  method: GET
  path: /v1/models
  expected_status: 200
distributed_profile:
  supports_multi_replica: true
  supports_distributed: true
  distributed_modes:
    - torchrun
    - native
verification:
  status: pending_verification
  notes: "Template derived from local user-tested MetaX docker run and public MetaX/MACA vLLM references; validate image-specific arguments on target node."
managed_by: system
catalog_version: "2026.06"
```

### 26.4 MetaX 高风险项

页面应提示高风险：

```text
privileged=true
ipc=host
uts=host
seccomp=unconfined
apparmor=unconfined
/dev/mem
```

`/dev/mem` 默认不启用，只作为 optional_devices。

### 26.5 MetaX NodeBackendRuntime

如果现场使用本机已有 image，例如：

```text
0d307f1665d3
```

不要写死到系统 catalog。节点页应允许选择本机已有 image，并在 NodeBackendRuntime 记录：

```text
image_ref: 0d307f1665d3
image_id
image_digest
status: ready
```

---

## 27. Huawei / 昇腾 Runtime 模板设计

### 27.1 设计原则

Huawei/Ascend 不能按 NVIDIA `--gpus` 处理。应通过 direct device mount 或 Ascend runtime/vendor adapter。

Huawei Runtime 应能表达：

```text
/dev/davinci{{index}}
/dev/davinci_manager
/dev/devmm_svm
/dev/hisi_hdc
/usr/local/dcmi
/usr/local/bin/npu-smi
/usr/local/Ascend/driver/lib64
/usr/local/Ascend/driver/version.info
/etc/ascend_install.info
```

### 27.2 vLLM Huawei Docker Runtime 示例

```yaml
id: runtime.vllm.huawei-docker
backend: vllm
backend_version: vllm-openai-latest
name: vLLM Huawei Ascend Docker
runner_type: docker
runtime_type: docker
vendor: huawei
accelerator_kind: npu
image_ref: quay.io/ascend/vllm-ascend:v0.9.0rc2-openeuler
image_pull_policy: if_not_present
command:
  - python
  - -m
  - vllm.entrypoints.openai.api_server
default_args:
  - --host
  - 0.0.0.0
  - --port
  - "8000"
params_mapping:
  tensor_parallel_size: --tensor-parallel-size
  gpu_memory_utilization: --gpu-memory-utilization
  max_model_len: --max-model-len
  dtype: --dtype
generated_args:
  model: "--model {{model_path_in_container}}"
ports:
  - container_port: 8000
    protocol: tcp
mount_policy:
  model:
    mount_type: directory
    container_mount_dir: /models
    readonly: true
device_policy:
  type: huawei-ascend
  visible_devices_from: vendor_adapter
  device_id_source: npu_device.index
  device_env:
    name: ASCEND_VISIBLE_DEVICES
    value_from: vendor_visible_devices
  alternate_device_env:
    - ASCEND_RT_VISIBLE_DEVICES
docker_options:
  privileged:
    enabled: false
    value: false
  ipc_mode:
    enabled: true
    value: host
  shm_size:
    enabled: true
    value: 16g
  devices:
    enabled: true
    value: |
      /dev/davinci_manager
      /dev/devmm_svm
      /dev/hisi_hdc
      /dev/davinci{{index}}
  env:
    enabled: true
    value: |
      LIGHTAI_VENDOR=huawei
      ASCEND_VISIBLE_DEVICES={{vendor_visible_devices}}
  extra_mounts:
    enabled: true
    value: |
      /usr/local/dcmi:/usr/local/dcmi:rw
      /usr/local/bin/npu-smi:/usr/local/bin/npu-smi:ro
      /usr/local/Ascend/driver/lib64:/usr/local/Ascend/driver/lib64:ro
      /usr/local/Ascend/driver/version.info:/usr/local/Ascend/driver/version.info:ro
      /etc/ascend_install.info:/etc/ascend_install.info:ro
custom:
  args:
    enabled: false
    value: ""
  env:
    enabled: false
    value: ""
  docker_options:
    enabled: false
    value: ""
health_check:
  type: http
  method: GET
  path: /v1/models
  expected_status: 200
distributed_profile:
  supports_multi_replica: true
  supports_distributed: true
  distributed_modes:
    - torchrun
    - ray
verification:
  status: template_only
  notes: "Huawei Ascend runtime template based on public vLLM-Ascend/CANN examples; requires huawei vendor adapter and target-node validation."
managed_by: system
catalog_version: "2026.06"
```

### 27.3 Huawei 状态标记

Huawei 模板默认不能标记为已验证。

```text
verification.status = template_only
```

如果当前节点没有 huawei vendor adapter，NodeBackendRuntime 应显示：

```text
adapter_missing
unsupported_device
template_only
```

不能显示 `ready`。

---

## 28. Vendor 扩展方式

新增厂商的目标体验：

```text
如果只是 Docker 参数不同：
  新增 runtime yaml 即可。

如果涉及设备发现、设备 ID 映射、指标采集：
  新增 runtime yaml + vendor adapter。
```

建议目录：

```text
configs/backend-catalog/runtimes/vllm/huawei-docker.yaml
configs/backend-catalog/runtimes/sglang/huawei-docker.yaml
configs/backend-catalog/runtimes/llamacpp/huawei-docker.yaml
```

vendor adapter 预留：

```text
internal/runtime/vendors/nvidia.go
internal/runtime/vendors/metax.go
internal/runtime/vendors/huawei.go
```

新增厂商步骤：

```text
1. 增加 BackendRuntime yaml。
2. 配置 vendor、accelerator_kind、device_policy。
3. 配置 docker_options enabled blocks。
4. 配置 health_check。
5. 配置 params_mapping。
6. Server seed 后 API 展示。
7. Web 展示 Runtime。
8. 节点页执行 runtime check。
9. 如果没有 vendor adapter，状态显示 adapter_missing/template_only。
10. 如需设备发现/映射/监控，实现 vendor adapter。
11. 创建模型实例，生成 NodeRunPlan。
12. 通过 command preview 检查最终 docker run。
```

---

## 29. NodeRunPlan Command Preview 要求

Command preview 必须能体现最终固化配置。

对于 MetaX 示例，应能生成接近：

```bash
docker run -d \
  --name lightai-inst-xxx \
  --device=/dev/dri \
  --device=/dev/mxcd \
  --device=/dev/infiniband \
  --group-add video \
  --uts=host \
  --ipc=host \
  --privileged=true \
  --security-opt seccomp=unconfined \
  --security-opt apparmor=unconfined \
  --shm-size 100gb \
  --ulimit memlock=-1 \
  -p 8001:8000 \
  -v /data/part2/MX-C500/model:/models:ro \
  -e CUDA_VISIBLE_DEVICES=6,7 \
  <image_ref_or_image_id> \
  python -m vllm.entrypoints.openai.api_server \
  --host 0.0.0.0 \
  --port 8000 \
  --model /models/<relative_path>
```

要求：

```text
1. 只包含 enabled=true 的配置。
2. 模型挂载来自 ModelLocation。
3. 容器内路径来自 BackendRuntime mount_policy。
4. 设备可见 ID 来自 GPU Lease + vendor adapter。
5. 高风险参数在页面上提示。
6. command preview 与 Agent 实际执行一致。
```

---

## 30. 设计验收补充

新增验收标准：

```text
1. 用户选择 Backend、BackendVersion、BackendRuntime、Model 后，默认参数基本可直接启动 Docker。
2. Web 支持启动、停止、重启、删除、cleanup。
3. Web 支持查看 Docker 日志。
4. Web 支持查看容器状态和健康检查状态。
5. Web 支持查看 command preview。
6. BackendRuntime 页面支持高风险单值项独立开关。
7. BackendRuntime 页面支持列表类参数的 enabled + 多行输入。
8. BackendRuntime 页面支持 Custom Args / Custom Env / Custom Docker Options。
9. 未勾选的参数不进入 NodeRunPlan。
10. MetaX Runtime 模板包含本地实测 Docker 参数。
11. Huawei Runtime 模板包含 Ascend 常见 device 和 driver/tool mounts。
12. Huawei 未验证时显示 template_only / adapter_missing，不得显示 ready。
13. NodeRunPlan 详情页展示最终固化的 image、args、env、mounts、devices、docker_options、health_check。
14. failed 状态自动展示最后 N 行 Docker 日志。

---

## 31. 本机 NVIDIA 标准测试场景

开发服务器已经具备 NVIDIA GPU、Docker、模型和镜像，应作为第一阶段必须跑通的标准验收环境。

### 31.1 测试环境

| 项目 | 值 |
|---|---|
| Host | KZ-LAPTOP |
| GPU | NVIDIA GeForce RTX 5090 Laptop GPU, 24,463 MiB |
| Docker | 29.5.3, nvidia runtime available |
| NVIDIA-SMI | 610.43.02, CUDA UMD 13.3 |

要求：

```text
1. Agent 能识别 KZ-LAPTOP 节点。
2. Agent 能上报 NVIDIA GPU。
3. Agent 能识别 Docker 可用。
4. Agent 能确认 NVIDIA runtime/toolkit 可用。
5. Server 能创建或更新 NodeBackendRuntime。
6. Web 节点页能展示 GPU、Docker、Runtime 状态。
```

---

### 31.2 测试模型和镜像

| Backend | Image | Model | Host Port | Container Port |
|---|---|---|---:|---:|
| vLLM | `vllm/vllm-openai:latest` | `/home/kzeng/models/Qwen3-0.6B-Instruct-2512` | 8004 | 8000 |
| SGLang | `lmsysorg/sglang:latest` | `/home/kzeng/models/Qwen3-0.6B-Instruct-2512` | 30000 | 30000 |
| llama.cpp | `ghcr.io/ggml-org/llama.cpp:server-cuda13` | `/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf` | 8002 | 8080 |

这些模型和镜像应作为本地 E2E 的默认验证输入。
如果镜像未提前 pull，Runtime 检测应显示 `missing_image`，并允许通过 Web/API 触发 pull 或选择本机已有 image。

---

### 31.3 推荐通过 API 配置并验证

本机 NVIDIA 验收应优先通过 API 自动配置，避免只依赖 Web 手工操作。

推荐验证链路：

```text
1. 登录获取 session / token。
2. 查询节点，确认 KZ-LAPTOP online。
3. 查询 GPU，确认 NVIDIA GPU online。
4. 查询 Backend Catalog，确认 vLLM / SGLang / llama.cpp 存在。
5. 查询 BackendVersion，确认对应版本存在。
6. 查询 BackendRuntime，确认 NVIDIA Docker Runtime 存在。
7. 对目标节点执行 NodeBackendRuntime check。
8. 添加 ModelArtifact + ModelLocation。
9. 创建 DeploymentPlan。
10. 启动 Deployment。
11. 查询 RunPlanGroup。
12. 查询 NodeRunPlan。
13. 查询 command preview。
14. 等待实例 running。
15. 访问健康检查 endpoint。
16. 查看 Docker 日志。
17. 停止实例。
18. 确认容器停止、GPU Lease released。
```

---

### 31.4 API 验收示例：vLLM

#### 31.4.1 Runtime 检查

目标：

```text
Backend: vllm
BackendVersion: vllm-openai-latest
BackendRuntime: runtime.vllm.nvidia-docker
Image: vllm/vllm-openai:latest
Node: KZ-LAPTOP
```

API 应能完成：

```text
1. 启用或检查 runtime.vllm.nvidia-docker。
2. 确认 Docker image 存在或可拉取。
3. 确认 NVIDIA runtime 可用。
4. 创建 NodeBackendRuntime ready。
```

#### 31.4.2 模型添加

模型：

```text
path_type: directory
model_root: /home/kzeng/models
relative_path: Qwen3-0.6B-Instruct-2512
absolute_path: /home/kzeng/models/Qwen3-0.6B-Instruct-2512
format: huggingface / safetensors
```

Agent 应扫描：

```text
config.json
tokenizer.json / tokenizer_config.json
generation_config.json
*.safetensors
model.safetensors.index.json，如存在
```

并生成：

```text
ModelArtifact
ModelLocation(node=KZ-LAPTOP, verified/warning/manual_attested)
```

#### 31.4.3 Deployment 创建

DeploymentPlan 建议：

```yaml
mode: single
replicas: 1
backend: vllm
backend_version: vllm-openai-latest
backend_runtime: runtime.vllm.nvidia-docker
model: Qwen3-0.6B-Instruct-2512
node: KZ-LAPTOP
gpu_count: 1
host_port: 8004
container_port: 8000
runtime_params:
  tensor_parallel_size:
    enabled: true
    value: 1
  gpu_memory_utilization:
    enabled: true
    value: 0.9
```

预期 NodeRunPlan：

```text
image: vllm/vllm-openai:latest
mount: /home/kzeng/models:/models:ro
model_path_in_container: /models/Qwen3-0.6B-Instruct-2512
port: 8004:8000
GPU: --gpus device=<index> 或 NVIDIA_VISIBLE_DEVICES=<index>
health_check: GET http://127.0.0.1:8004/v1/models
```

健康检查：

```bash
curl http://127.0.0.1:8004/v1/models
```

预期：

```text
HTTP 200
返回 OpenAI-compatible models 列表
```

---

### 31.5 API 验收示例：SGLang

#### 31.5.1 Runtime 检查

目标：

```text
Backend: sglang
BackendVersion: sglang-openai-latest
BackendRuntime: runtime.sglang.nvidia-docker
Image: lmsysorg/sglang:latest
Node: KZ-LAPTOP
```

#### 31.5.2 模型添加

复用模型：

```text
/home/kzeng/models/Qwen3-0.6B-Instruct-2512
```

如果该模型已经作为 ModelArtifact 存在，则只需要复用已有 ModelArtifact 和 ModelLocation。

#### 31.5.3 Deployment 创建

DeploymentPlan 建议：

```yaml
mode: single
replicas: 1
backend: sglang
backend_version: sglang-openai-latest
backend_runtime: runtime.sglang.nvidia-docker
model: Qwen3-0.6B-Instruct-2512
node: KZ-LAPTOP
gpu_count: 1
host_port: 30000
container_port: 30000
runtime_params:
  tp_size:
    enabled: true
    value: 1
```

预期 NodeRunPlan：

```text
image: lmsysorg/sglang:latest
mount: /home/kzeng/models:/models:ro
model_path_in_container: /models/Qwen3-0.6B-Instruct-2512
port: 30000:30000
health_check: GET http://127.0.0.1:30000/v1/models
```

健康检查：

```bash
curl http://127.0.0.1:30000/v1/models
```

---

### 31.6 API 验收示例：llama.cpp

#### 31.6.1 Runtime 检查

目标：

```text
Backend: llamacpp
BackendVersion: llama-cpp-server
BackendRuntime: runtime.llamacpp.nvidia-docker
Image: ghcr.io/ggml-org/llama.cpp:server-cuda13
Node: KZ-LAPTOP
```

注意：llama.cpp container port 为 `8080`，host port 为 `8002`。

#### 31.6.2 模型添加

模型：

```text
path_type: file
model_root: /home/kzeng/models/Qwen3.5-9B-Q4
relative_path: Qwen3.5-9B-Q4_K_M.gguf
absolute_path: /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
format: gguf
```

Agent 应扫描 GGUF metadata，并至少返回：

```text
size_bytes
checksum
format=gguf
architecture / family，如可识别
quantization，如可识别
capabilities guess
```

#### 31.6.3 Deployment 创建

DeploymentPlan 建议：

```yaml
mode: single
replicas: 1
backend: llamacpp
backend_version: llama-cpp-server
backend_runtime: runtime.llamacpp.nvidia-docker
model: Qwen3.5-9B-Q4_K_M.gguf
node: KZ-LAPTOP
gpu_count: 1
host_port: 8002
container_port: 8080
runtime_params:
  ctx_size:
    enabled: true
    value: 4096
  n_gpu_layers:
    enabled: true
    value: 999
```

预期 NodeRunPlan：

```text
image: ghcr.io/ggml-org/llama.cpp:server-cuda13
mount: /home/kzeng/models/Qwen3.5-9B-Q4:/models:ro
model_path_in_container: /models/Qwen3.5-9B-Q4_K_M.gguf
port: 8002:8080
args:
  --host 0.0.0.0
  --port 8080
  --model /models/Qwen3.5-9B-Q4_K_M.gguf
  --ctx-size 4096
  --n-gpu-layers 999
health_check: GET http://127.0.0.1:8002/v1/models
```

健康检查：

```bash
curl http://127.0.0.1:8002/v1/models
```

预期：

```text
HTTP 200
返回模型列表
```

---

### 31.7 API 自动化脚本建议

建议新增脚本：

```text
scripts/e2e-backend-runtime-nvidia-api.sh
```

脚本目标：

```text
1. 登录。
2. 查询节点和 GPU。
3. 检查 Backend Catalog。
4. 检查/启用 vLLM Runtime。
5. 添加 vLLM 模型位置。
6. 创建 vLLM Deployment。
7. 等待 running。
8. curl /v1/models。
9. 查询日志。
10. 停止并 cleanup。
11. 重复 SGLang。
12. 重复 llama.cpp。
```

脚本要求：

```text
1. 所有创建的资源使用固定前缀，例如 e2e-nvidia-*。
2. cleanup 只清理自己创建的容器、实例、lease。
3. 不 kill 其他用户进程。
4. 端口占用时给出清晰错误。
5. 镜像或模型不存在时 skip，并输出 skip 原因。
6. 失败时打印 NodeRunPlan command preview 和 Docker logs tail。
```

建议环境变量：

```bash
LIGHTAI_BASE_URL=http://127.0.0.1:18080
LIGHTAI_USERNAME=admin
LIGHTAI_PASSWORD=...
LIGHTAI_NODE_NAME=KZ-LAPTOP

VLLM_IMAGE=vllm/vllm-openai:latest
VLLM_MODEL=/home/kzeng/models/Qwen3-0.6B-Instruct-2512
VLLM_HOST_PORT=8004
VLLM_CONTAINER_PORT=8000

SGLANG_IMAGE=lmsysorg/sglang:latest
SGLANG_MODEL=/home/kzeng/models/Qwen3-0.6B-Instruct-2512
SGLANG_HOST_PORT=30000
SGLANG_CONTAINER_PORT=30000

LLAMACPP_IMAGE=ghcr.io/ggml-org/llama.cpp:server-cuda13
LLAMACPP_MODEL=/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
LLAMACPP_HOST_PORT=8002
LLAMACPP_CONTAINER_PORT=8080
```

---

### 31.8 本机 NVIDIA 验收标准

本机 NVIDIA 环境下，至少需要完成：

```text
1. API 能查询 KZ-LAPTOP 节点 online。
2. API 能查询 NVIDIA GPU online。
3. API 能查询 Docker available。
4. API 能查询 NVIDIA runtime/toolkit available。
5. Backend Catalog 中存在 vLLM、SGLang、llama.cpp。
6. BackendVersion 中存在 vllm-openai-latest、sglang-openai-latest、llama-cpp-server。
7. BackendRuntime 中存在 NVIDIA Docker Runtime。
8. NodeBackendRuntime check 能显示 image present 或 missing_image。
9. ModelArtifact + ModelLocation 能通过 API 创建。
10. DeploymentPlan 能通过 API 创建。
11. RunPlanGroup 和 NodeRunPlan 能生成。
12. Command preview 正确显示 image、mount、port、GPU、args。
13. Agent 能启动 Docker。
14. 健康检查 `/v1/models` 通过。
15. Web 能显示 running。
16. Web 能查看 Docker logs。
17. Stop 后容器停止。
18. GPU Lease released。
19. Cleanup 不影响非 E2E 容器。
```

---

### 31.9 失败场景验收

必须覆盖：

```text
1. 模型路径不存在：RunPlan 生成失败，不下发 Agent。
2. Runtime image 不存在：NodeBackendRuntime missing_image，不能启动，或按策略 pull。
3. 端口被占用：RunPlan 生成失败或 Agent 启动失败并进入 failed。
4. GPU 不足：GPU Lease reserved 失败。
5. 容器启动后立即退出：实例 failed，展示 exit_code 和 Docker logs tail。
6. 健康检查失败：实例 failed，释放 lease。
7. 用户停止实例：容器停止，lease released。
```


## 32. 最终固定口径

后续文档、代码注释、开发 prompt 和审查报告统一使用以下口径：

```text
Backend 定义推理后端类型。
BackendVersion 定义后端软件版本、能力和参数 schema，原则上硬件无关。
BackendRuntime 定义该版本在某类硬件和运行环境上的运行模板，例如 NVIDIA Docker、MetaX Docker、CPU Docker。
NodeBackendRuntime 定义某个节点上该 Runtime 的实际可用状态，例如 image、driver、toolkit 是否 ready。

ModelArtifact 定义逻辑模型。
ModelLocation 是 ModelArtifact 的节点位置属性，记录该模型在某个节点的路径、校验和状态；实现上可单独存储。

DeploymentPlan 描述用户部署意图。
RunPlanGroup 组织一次部署解析后的一个或多个节点计划。
NodeRunPlan 是 Agent 执行的节点级计划，包含模型路径、Runtime 安装、GPU Lease、Docker mount、args、env、ports、health check。

第一阶段产品界面按“启动一个模型实例”简化；
内部保留三层计划模型，支持未来多副本和分布式扩展。
```

一句话总结：

> **LightAI Go 不再让用户从零拼 Docker。系统内置 Backend Catalog，节点启用 Runtime，模型按节点扫描位置；用户选择模型和后端后，Server 校验目标节点上的 ModelLocation、NodeBackendRuntime 和 GPU Lease，生成 NodeRunPlan，由 Agent 执行 Docker，并用健康检查和审计保证状态一致。**

---
## Observability Closeout (2026-06-19)

### Docker logs in non-running states
Logs are accessible for any instance with a `current_run_plan_id`:
- Web: `ModelInstancesPage.vue` checks `row.current_run_plan_id` (not `actual_state=running`)
- API: `GET /api/v1/node-run-plans/{run_plan_id}/logs`
- run_plan_id obtained from `GET /api/v1/model-instances/{id}`

### Failure diagnostics
On container failure, the agent reports:
- container_id, exit_code, failure_reason_code
- stderr_tail_preview, stdout_tail_preview (single-line escaped)
- Server stores structured JSON in model_instances.last_error

### Log noise reduction
- /metrics and /metrics/targets: DEBUG via highFrequencyPrefixes prefix match
- High-frequency GET list polling (instances, deployments, nodes, gpus): DEBUG via isHighFrequencyGET

### Audit
- instance.start.requested: task created, request accepted
- instance.state.updated running: container start succeeded
- instance.state.updated failed: health check failed or container exited

Status: CLOSED — no remaining observability gaps.
