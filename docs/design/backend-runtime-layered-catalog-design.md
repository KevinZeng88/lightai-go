# LightAI Go Backend / BackendVersion / BackendRuntime / NodeBackendRuntime 分层设计

## 1. 目标

本文定义 LightAI Go 中推理后端、后端版本、后端运行模板、节点运行配置的职责边界、字段模型、继承关系、文件 catalog 机制、Web/API 行为和内置 catalog 基线。

本设计用于修正以下混淆：

1. BackendVersion 被放入硬件/vendor 参数。
2. BackendRuntime 被误做成节点配置。
3. 运行模板页面出现节点管理能力。
4. NodeBackendRuntime 检查动作修改运行配置。
5. catalog 只存 DB，不便于脚本导入、导出和用户分享。
6. 沐曦 MacaRT-SGLang、华为 Ascend、NVIDIA CUDA 等运行参数层级不清。

最终原则：

```text
Backend                 = 推理后端软件定义
BackendVersion          = 推理后端某软件版本的能力和参数定义
BackendRuntime          = 某 BackendVersion + 某类硬件/vendor/runtime 的运行模板
NodeBackendRuntime      = BackendRuntime 绑定具体节点和具体 image 后形成的节点运行配置
```

依赖链：

```text
Backend
  -> BackendVersion
    -> BackendRuntime
      -> NodeBackendRuntime
```

继承规则：

```text
BackendVersion -> BackendRuntime：
  创建时复制，创建后独立。

BackendRuntime -> NodeBackendRuntime：
  创建时复制，创建后独立。

NodeBackendRuntime check/validate：
  只检查，不刷新 snapshot，不修改 image_ref，不修改运行配置字段。
```

存储规则：

```text
Backend / BackendVersion / BackendRuntime：
  文件 catalog 为事实源。
  DB 是 reload/sync 后的 projection。

NodeBackendRuntime：
  DB 为事实源。
  不放入 catalog 文件。
```

---

## 2. 总体分层

### 2.1 Backend：推理后端

Backend 表示推理后端软件类型，只说明“它是什么”。

示例：

```text
vLLM
SGLang
llama.cpp
```

Backend 不包含：

```text
版本号
具体 Docker image
硬件参数
节点参数
启动参数具体值
运行检查状态
```

### 2.2 BackendVersion：推理后端版本

BackendVersion 表示某个 Backend 的软件版本能力和参数定义。

它回答：

```text
这个后端版本支持哪些 API？
默认端点是什么？
官方/通用启动参数 schema 是什么？
模型路径参数叫什么？
默认 health check 怎么做？
有哪些通用 image candidates？
```

BackendVersion 必须硬件无关。

BackendVersion 不允许包含：

```text
GPU index
CUDA_VISIBLE_DEVICES
MACA_VISIBLE_DEVICE
--gpus all
/dev/mxcd
/dev/davinci0
node_id
image_present
ready / needs_check
某台服务器 host path
```

### 2.3 BackendRuntime：后端运行模板

BackendRuntime 表示某个 BackendVersion 在某类硬件/vendor/runtime distribution 下如何运行。

它回答：

```text
SGLang 0.4.6-compatible 在沐曦 MacaRT-SGLang 下如何启动？
vLLM v0.23.0 在 NVIDIA CUDA 下如何启动？
llama.cpp b9700 在 NVIDIA CUDA13 镜像下如何启动？
vLLM Ascend 在 CANN / Ascend NPU 下如何启动？
```

BackendRuntime 可以包含硬件/vendor/runtime 参数，但不绑定具体节点。

BackendRuntime 可以包含：

```text
vendor = nvidia / metax / huawei / amd / cpu
accelerator_api = cuda / mxmaca / cann / rocm / cpu
runtime_distribution = official / MacaRT-SGLang / vllm-ascend / llama.cpp-server-cuda13
docker_options
devices schema
env schema
image candidates
high-risk switches
default runtime args
```

BackendRuntime 不允许包含：

```text
node_id
image_present
ready / needs_check
last_checked_at
某节点实际 image_digest
具体 GPU index
某节点模型 host path
```

### 2.4 NodeBackendRuntime：节点运行配置

NodeBackendRuntime 表示某个 BackendRuntime 在某个节点上的实际运行配置。

它回答：

```text
这个节点用哪个 BackendRuntime？
这个节点上实际用哪个 image_ref？
这个节点是否有该镜像？
这个节点检查结果是否 ready？
这个节点级 env/device/ports 是否如何设置？
```

NodeBackendRuntime 可以包含：

```text
node_id
backend_runtime_id
image_ref
image_id
image_digest
image_present
docker_available
driver_version
toolkit_version
status
status_reason
last_checked_at
device_check_json
config_snapshot_json
node_env_overrides_json
node_device_selection_json
port_bindings_json
```

NodeBackendRuntime 的 `config_snapshot_json` 来自 BackendRuntime 创建时复制。创建后独立。

---

## 3. 存储与同步机制

### 3.1 文件 catalog 为事实源的对象

以下对象属于定义型 catalog：

```text
Backend
BackendVersion
BackendRuntime
```

这些对象必须支持文件 catalog：

```text
system catalog：随软件发布，只读
user catalog：用户新增 / 克隆 / 修改，可导入导出
DB projection：reload/sync 后用于 Web/API 查询
```

建议目录：

```text
configs/backend-catalog/backends/
configs/backend-catalog/versions/
configs/backend-catalog/runtimes/

data/backend-catalog.d/user/backends/
data/backend-catalog.d/user/versions/
data/backend-catalog.d/user/runtimes/
```

可配置：

```text
LIGHTAI_BACKEND_CATALOG_USER_DIR
```

加载顺序：

```text
启动 / reload
  -> 读取 system Backend catalog
  -> 读取 system BackendVersion catalog
  -> 读取 system BackendRuntime catalog
  -> 读取 user catalog
  -> schema 校验
  -> 计算 revision / config_hash
  -> upsert 到 DB projection
  -> Web/API 从 DB projection 查询
```

如果文件与 DB 不一致：

```text
以文件 catalog 为准。
通过 reload/sync 修正 DB projection。
```

### 3.2 DB 为事实源的对象

以下对象属于运行态配置或状态：

```text
NodeBackendRuntime
Deployment
ModelInstance
RunPlan / NodeRunPlan
AuditLog
```

这些对象不放入 catalog 文件。

NodeBackendRuntime 可提供导入/导出能力，但不是 catalog 事实源。

---

## 4. 字段定义

### 4.1 Backend 字段

建议字段：

```yaml
id: sglang
name: SGLang
display_name: SGLang
description: High-performance serving framework for large language models and multimodal models.
homepage: https://sglang.ai
protocol_family:
  - openai-compatible
source: system
readonly: true
revision: "2026-06-18"
config_hash: "<sha256>"
official_reference:
  - source_name: SGLang Documentation
    source_type: official_docs
    checked_at: "2026-06-18"
```

可修改规则：

```text
system Backend：
  不可直接修改。
  仅可随软件升级变更。

user Backend：
  预留能力。
  一般不建议普通用户新增。
  仅管理员可新增/编辑/删除。
```

### 4.2 BackendVersion 字段

建议字段：

```yaml
id: sglang-v0.5.12.post1
backend_id: sglang
version: v0.5.12.post1
name: SGLang v0.5.12.post1
source: system
readonly: true
protocol: openai-compatible

capabilities:
  - models
  - chat_completions
  - completions
  - embeddings
  - openai_compatible

default_endpoints:
  models: /v1/models
  chat_completions: /v1/chat/completions
  completions: /v1/completions
  embeddings: /v1/embeddings

default_host: 0.0.0.0
default_port: 30000

model_mount:
  container_path: /models
  readonly: true

model_path_arg:
  name: --model-path
  value: "{{MODEL_CONTAINER_PATH}}"

entrypoint:
  - python3
  - -m
  - sglang.launch_server

args_schema:
  - name: --model-path
    required: true
    value: "{{MODEL_CONTAINER_PATH}}"
  - name: --host
    default: "0.0.0.0"
  - name: --port
    default: "30000"
  - name: --tp
    optional: true
  - name: --dp
    optional: true

env_schema: []

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200

image_candidates:
  - lmsysorg/sglang:v0.5.12.post1
  - lmsysorg/sglang:latest-runtime
  - lmsysorg/sglang:latest

official_reference:
  - source_name: SGLang official install and OpenAI API docs
    source_type: official_docs
    checked_at: "2026-06-18"

revision: "2026-06-18"
config_hash: "<sha256>"
```

BackendVersion 可包含通用软件 image candidates，但不应包含 hardware-runtime image。

例如：

```text
允许：
  vllm/vllm-openai:v0.23.0
  lmsysorg/sglang:v0.5.12.post1

谨慎：
  ghcr.io/ggml-org/llama.cpp:server-cuda13
  quay.io/ascend/vllm-ascend
  MacaRT-SGLang 发布包镜像
```

这些更适合放入 BackendRuntime。

可修改规则：

```text
system BackendVersion：
  只读。
  可查看。
  可 clone 为 user BackendVersion。

user BackendVersion：
  可新增。
  可编辑。
  可删除。
  修改时写 user catalog 文件，再 reload/sync DB。

BackendVersion 修改后：
  不自动影响已有 BackendRuntime。
```

### 4.3 BackendRuntime 字段

BackendRuntime 是运行模板。它由 BackendVersion + 硬件/vendor runtime profile 创建。

建议字段：

```yaml
id: sglang-metax-macart-sglang-0.4.6
name: SGLang MetaX MacaRT-SGLang 0.4.6
backend_id: sglang
backend_version_id: sglang-0.4.6-compatible

source: system
readonly: true

vendor: metax
hardware_family: xiyun
accelerator_api: mxmaca
runtime_distribution: MacaRT-SGLang
runtime_distribution_version: ""
compatibility:
  backend_version: SGLang 0.4.6-compatible
  pytorch: "2.6"

protocol: openai-compatible

image_candidates:
  - "<from Metax PDE AI release package>"
image_note: "MacaRT-SGLang image is obtained from Metax release package; do not invent public image tag."

docker_options:
  privileged: true
  network_mode: host
  uts: host
  ipc: host
  shm_size: 100gb
  group_add:
    - video
  security_opt:
    - seccomp=unconfined
    - apparmor=unconfined
  ulimits:
    memlock: -1

devices:
  required:
    - /dev/dri
    - /dev/mxcd
    - /dev/mem
  optional:
    - /dev/infiniband

volumes:
  required: []
  optional:
    - host_path: /mnt/hdd/SGLang
      container_path: /software
      readonly: false
      note: "Example from vendor docs; deployment-specific."

env_schema:
  - name: MACA_SMALL_PAGESIZE_ENABLE
    default: "1"
  - name: PYTORCH_ENABLE_PG_HIGH_PRIORITY_STREAM
    default: "1"
  - name: TRITON_ENABLE_MAC_A_OPT_MOVE_DOT_OPERANDS_OUT_LOOP
    default: "1"
  - name: TRITON_ENABLE_MAC_A_CHAIN_DOT_OPT
    default: "1"
  - name: MACA_VISIBLE_DEVICE
    node_specific: true
    optional: true
    note: "Set according to mx-smi topology on each node."
  - name: GLOO_SOCKET_IFNAME
    node_specific: true
    optional: true

entrypoint:
  - python3
  - -m
  - sglang.launch_server

args:
  - --model-path
  - "{{MODEL_CONTAINER_PATH}}"
  - --host
  - "0.0.0.0"
  - --port
  - "30000"

args_schema:
  inherit_from_backend_version: true
  extra:
    - name: --attention-backend
      optional: true
    - name: --enable-ep-moe
      optional: true
    - name: --enable-dp-attention
      optional: true
    - name: --mem-fraction-static
      optional: true
    - name: --disable-radix-cache
      optional: true
    - name: --disable-chunked-prefix-cache
      optional: true

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200

high_risk_flags:
  privileged: true
  host_network: true
  host_ipc: true
  unconfined_seccomp: true
  unconfined_apparmor: true

source_backend_version_revision: "<revision>"
source_config_hash: "<sha256>"
revision: "2026-06-18"
config_hash: "<sha256>"
```

可修改规则：

```text
system BackendRuntime：
  只读。
  可查看。
  可 clone 为 user BackendRuntime。

user BackendRuntime：
  可新增。
  可编辑。
  可删除。
  修改时写 user runtime catalog 文件，再 reload/sync DB。

BackendRuntime 修改后：
  不自动影响已有 NodeBackendRuntime。

BackendRuntime 不允许出现：
  node_id
  image_present
  ready / needs_check
  last_checked_at
  具体节点实际 image_digest
```

### 4.4 NodeBackendRuntime 字段

NodeBackendRuntime 是节点运行配置，DB 为事实源。

建议字段：

```yaml
id: <uuid>
node_id: <node uuid>
backend_runtime_id: sglang-metax-macart-sglang-0.4.6

image_ref: "<actual image on this node>"
image_id: ""
image_digest: ""
image_present: false

config_snapshot_json:
  copied_from_backend_runtime: true
  snapshot: "<frozen BackendRuntime config at creation time>"

node_env_overrides_json:
  MACA_VISIBLE_DEVICE: "7,5,1,3,4,0,6,2"
  GLOO_SOCKET_IFNAME: "eth0"

node_device_selection_json:
  visible_devices:
    - "0"
    - "1"

port_bindings_json:
  - container_port: 30000
    host_port: 30000
    protocol: tcp

docker_available: true
driver_version: ""
toolkit_version: ""
device_check_json: {}

status: needs_check
status_reason: "created; check required"
last_checked_at: null

source_backend_runtime_id: sglang-metax-macart-sglang-0.4.6
source_backend_runtime_revision: "<revision>"
source_config_hash: "<sha256>"
created_at: ""
updated_at: ""
```

可修改规则：

```text
允许修改：
  image_ref
  node_env_overrides_json
  node_device_selection_json
  port_bindings_json
  node-level volumes / devices if UI allows
  status only through check/validate or explicit admin action

修改 image_ref / env / device / port / snapshot 相关字段：
  status 必须变为 needs_check。

check/validate 允许更新：
  image_present
  docker_available
  driver_version
  toolkit_version
  device_check_json
  status
  status_reason
  last_checked_at
  updated_at

check/validate 禁止更新：
  image_ref
  config_snapshot_json
  source_backend_runtime_revision
  source_config_hash
  node_id
  backend_runtime_id
```

---

## 5. 官方内置 Backend 基线

### 5.1 vLLM Backend

```yaml
id: vllm
name: vLLM
description: High-throughput and memory-efficient LLM inference and serving engine.
protocol_family:
  - openai-compatible
source: system
readonly: true
```

### 5.2 SGLang Backend

```yaml
id: sglang
name: SGLang
description: High-performance serving framework for large language models and multimodal models.
protocol_family:
  - openai-compatible
source: system
readonly: true
```

### 5.3 llama.cpp Backend

```yaml
id: llamacpp
name: llama.cpp
description: Lightweight C/C++ LLM inference engine with llama-server HTTP API and GGUF model support.
protocol_family:
  - openai-compatible-subset
source: system
readonly: true
```

---

## 6. 官方内置 BackendVersion 基线

### 6.1 vLLM v0.23.0

```yaml
id: vllm-v0.23.0
backend_id: vllm
version: v0.23.0
source: system
readonly: true
protocol: openai-compatible

image_candidates:
  - vllm/vllm-openai:v0.23.0
  - vllm/vllm-openai:v0.23.0-cu129-ubuntu2404
  - vllm/vllm-openai:latest

default_host: 0.0.0.0
default_port: 8000

default_endpoints:
  models: /v1/models
  chat_completions: /v1/chat/completions
  completions: /v1/completions
  embeddings: /v1/embeddings

capabilities:
  - models
  - chat_completions
  - completions
  - embeddings
  - openai_compatible

model_mount:
  container_path: /models
  readonly: true

entrypoint:
  - vllm
  - serve

args_schema:
  - name: --model
    required: true
    value: "{{MODEL_CONTAINER_PATH}}"
  - name: --host
    default: "0.0.0.0"
  - name: --port
    default: "8000"
  - name: --served-model-name
    optional: true
  - name: --tensor-parallel-size
    optional: true
  - name: --max-model-len
    optional: true
  - name: --gpu-memory-utilization
    optional: true
  - name: --dtype
    optional: true
  - name: --trust-remote-code
    optional: true

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

### 6.2 SGLang v0.5.12.post1

```yaml
id: sglang-v0.5.12.post1
backend_id: sglang
version: v0.5.12.post1
source: system
readonly: true
protocol: openai-compatible

image_candidates:
  - lmsysorg/sglang:v0.5.12.post1
  - lmsysorg/sglang:latest-runtime
  - lmsysorg/sglang:latest

default_host: 0.0.0.0
default_port: 30000

default_endpoints:
  models: /v1/models
  chat_completions: /v1/chat/completions
  completions: /v1/completions
  embeddings: /v1/embeddings

capabilities:
  - models
  - chat_completions
  - completions
  - embeddings
  - openai_compatible

model_mount:
  container_path: /models
  readonly: true

entrypoint:
  - python3
  - -m
  - sglang.launch_server

args_schema:
  - name: --model-path
    required: true
    value: "{{MODEL_CONTAINER_PATH}}"
  - name: --host
    default: "0.0.0.0"
  - name: --port
    default: "30000"
  - name: --tp
    optional: true
  - name: --tensor-parallel-size
    optional: true
  - name: --dp
    optional: true
  - name: --enable-metrics
    optional: true
  - name: --log-level
    optional: true
  - name: --trust-remote-code
    optional: true

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

### 6.3 SGLang 0.4.6-compatible

用于 MacaRT-SGLang 等 vendor runtime。

```yaml
id: sglang-0.4.6-compatible
backend_id: sglang
version: 0.4.6-compatible
source: system
readonly: true
protocol: openai-compatible

default_host: 0.0.0.0
default_port: 30000

default_endpoints:
  models: /v1/models
  chat_completions: /v1/chat/completions
  completions: /v1/completions

capabilities:
  - models
  - chat_completions
  - completions
  - openai_compatible

model_mount:
  container_path: /models
  readonly: true

entrypoint:
  - python3
  - -m
  - sglang.launch_server

args_schema:
  - name: --model-path
    required: true
    value: "{{MODEL_CONTAINER_PATH}}"
  - name: --host
    default: "0.0.0.0"
  - name: --port
    default: "30000"
  - name: --tp
    optional: true
  - name: --dp
    optional: true
  - name: --dist-init-addr
    optional: true
  - name: --nnodes
    optional: true
  - name: --node-rank
    optional: true
  - name: --trust-remote-code
    optional: true
  - name: --attention-backend
    optional: true
  - name: --enable-dp-attention
    optional: true
  - name: --enable-ep-moe
    optional: true

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

### 6.4 llama.cpp b9700

```yaml
id: llamacpp-b9700
backend_id: llamacpp
version: b9700
source: system
readonly: true
protocol: openai-compatible-subset

default_host: 0.0.0.0
default_port: 8080

default_endpoints:
  models: /v1/models
  chat_completions: /v1/chat/completions
  completions: /v1/completions
  embeddings: /v1/embeddings

capabilities:
  - gguf
  - models
  - chat_completions
  - completions
  - embeddings
  - openai_compatible
  - web_ui

model_mount:
  container_path: /models
  readonly: true

entrypoint:
  - llama-server

args_schema:
  - name: -m
    alias: --model
    required: true
    value: "{{MODEL_CONTAINER_PATH}}"
  - name: --host
    default: "0.0.0.0"
  - name: --port
    default: "8080"
  - name: --ctx-size
    alias: -c
    optional: true
  - name: --n-gpu-layers
    alias: -ngl
    optional: true
  - name: --threads
    alias: -t
    optional: true
  - name: --threads-batch
    alias: -tb
    optional: true

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

---

## 7. 系统内置 BackendRuntime 基线

### 7.1 vLLM + NVIDIA CUDA

```yaml
id: vllm-v0.23.0-nvidia-cuda
name: vLLM v0.23.0 NVIDIA CUDA
backend_id: vllm
backend_version_id: vllm-v0.23.0
source: system
readonly: true

vendor: nvidia
accelerator_api: cuda
runtime_distribution: official-vllm-openai
hardware_family: nvidia-gpu

image_candidates:
  - vllm/vllm-openai:v0.23.0
  - vllm/vllm-openai:v0.23.0-cu129-ubuntu2404
  - vllm/vllm-openai:latest

docker_options:
  gpus: all
  ipc: host
  shm_size: ""
  runtime: nvidia

env_schema:
  - name: HF_TOKEN
    optional: true
    secret: true
  - name: VLLM_ENABLE_CUDA_COMPATIBILITY
    optional: true
  - name: NVIDIA_VISIBLE_DEVICES
    node_specific: true
    optional: true

args:
  - --model
  - "{{MODEL_CONTAINER_PATH}}"

ports:
  - container_port: 8000
    protocol: tcp

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

### 7.2 SGLang + NVIDIA CUDA

```yaml
id: sglang-v0.5.12-nvidia-cuda
name: SGLang v0.5.12 NVIDIA CUDA
backend_id: sglang
backend_version_id: sglang-v0.5.12.post1
source: system
readonly: true

vendor: nvidia
accelerator_api: cuda
runtime_distribution: official-sglang
hardware_family: nvidia-gpu

image_candidates:
  - lmsysorg/sglang:v0.5.12.post1
  - lmsysorg/sglang:latest-runtime
  - lmsysorg/sglang:latest

docker_options:
  gpus: all
  ipc: host
  shm_size: 32g

env_schema:
  - name: HF_TOKEN
    optional: true
    secret: true
  - name: NVIDIA_VISIBLE_DEVICES
    node_specific: true
    optional: true

entrypoint:
  - python3
  - -m
  - sglang.launch_server

args:
  - --model-path
  - "{{MODEL_CONTAINER_PATH}}"
  - --host
  - "0.0.0.0"
  - --port
  - "30000"

ports:
  - container_port: 30000
    protocol: tcp

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

### 7.3 llama.cpp + NVIDIA CUDA13

```yaml
id: llamacpp-b9700-nvidia-cuda13
name: llama.cpp b9700 NVIDIA CUDA13
backend_id: llamacpp
backend_version_id: llamacpp-b9700
source: system
readonly: true

vendor: nvidia
accelerator_api: cuda
runtime_distribution: llama.cpp-server-cuda13
hardware_family: nvidia-gpu

image_candidates:
  - ghcr.io/ggml-org/llama.cpp:server-cuda13
  - ghcr.io/ggml-org/llama.cpp:server-cuda
  - ghcr.io/ggml-org/llama.cpp:server

docker_options:
  gpus: all

env_schema:
  - name: NVIDIA_VISIBLE_DEVICES
    node_specific: true
    optional: true

entrypoint:
  - llama-server

args:
  - -m
  - "{{MODEL_CONTAINER_PATH}}"
  - --host
  - "0.0.0.0"
  - --port
  - "8080"

args_defaults:
  - name: --n-gpu-layers
    value: "-1"
    optional: true

ports:
  - container_port: 8080
    protocol: tcp

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

### 7.4 SGLang + MetaX MacaRT-SGLang

```yaml
id: sglang-0.4.6-metax-macart
name: SGLang 0.4.6 MetaX MacaRT-SGLang
backend_id: sglang
backend_version_id: sglang-0.4.6-compatible
source: system
readonly: true

vendor: metax
hardware_family: xiyun
accelerator_api: mxmaca
runtime_distribution: MacaRT-SGLang

compatibility:
  sglang: "0.4.6"
  pytorch: "2.6"

image_candidates:
  - "<from Metax release package>"
image_note: "Do not invent public image tag. Image is obtained from Metax release package."

docker_options:
  privileged: true
  network_mode: host
  uts: host
  ipc: host
  shm_size: 100gb
  group_add:
    - video
  security_opt:
    - seccomp=unconfined
    - apparmor=unconfined
  ulimits:
    memlock: -1

devices:
  required:
    - /dev/dri
    - /dev/mxcd
    - /dev/mem
  optional:
    - /dev/infiniband

env_schema:
  - name: MACA_SMALL_PAGESIZE_ENABLE
    default: "1"
  - name: PYTORCH_ENABLE_PG_HIGH_PRIORITY_STREAM
    default: "1"
  - name: TRITON_ENABLE_MAC_A_OPT_MOVE_DOT_OPERANDS_OUT_LOOP
    default: "1"
  - name: TRITON_ENABLE_MAC_A_CHAIN_DOT_OPT
    default: "1"
  - name: MACA_VISIBLE_DEVICE
    node_specific: true
    optional: true
  - name: GLOO_SOCKET_IFNAME
    node_specific: true
    optional: true

entrypoint:
  - python3
  - -m
  - sglang.launch_server

args:
  - --model-path
  - "{{MODEL_CONTAINER_PATH}}"
  - --host
  - "0.0.0.0"
  - --port
  - "30000"

ports:
  - container_port: 30000
    protocol: tcp

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

### 7.5 vLLM + Huawei Ascend CANN

```yaml
id: vllm-ascend-cann
name: vLLM Ascend CANN
backend_id: vllm
backend_version_id: vllm-v0.23.0
source: system
readonly: true

vendor: huawei
hardware_family: ascend
accelerator_api: cann
runtime_distribution: vllm-ascend

image_candidates:
  - quay.io/ascend/vllm-ascend:v0.21.0rc1
  - quay.io/ascend/vllm-ascend:v0.21.0rc1-openeuler
  - quay.io/ascend/vllm-ascend:v0.21.0rc1-a3
  - quay.io/ascend/vllm-ascend:v0.21.0rc1-a3-openeuler
  - quay.io/ascend/vllm-ascend:v0.21.0rc1-310p

docker_options:
  network_mode: host
  shm_size: 1g

devices:
  node_specific:
    - /dev/davinci0
  required:
    - /dev/davinci_manager
    - /dev/devmm_svm
    - /dev/hisi_hdc

volumes:
  recommended:
    - host_path: /usr/local/dcmi
      container_path: /usr/local/dcmi
      readonly: true
    - host_path: /usr/local/bin/npu-smi
      container_path: /usr/local/bin/npu-smi
      readonly: true
    - host_path: /usr/local/Ascend/driver/lib64
      container_path: /usr/local/Ascend/driver/lib64
      readonly: true
    - host_path: /usr/local/Ascend/driver/version.info
      container_path: /usr/local/Ascend/driver/version.info
      readonly: true
    - host_path: /etc/ascend_install.info
      container_path: /etc/ascend_install.info
      readonly: true

env_schema:
  - name: VLLM_USE_MODELSCOPE
    optional: true
  - name: PYTORCH_NPU_ALLOC_CONF
    optional: true

entrypoint:
  - vllm
  - serve

args:
  - "{{MODEL_CONTAINER_PATH}}"
  - --host
  - "0.0.0.0"
  - --port
  - "8000"

ports:
  - container_port: 8000
    protocol: tcp

health_check:
  type: http
  path: /v1/models
  success_status:
    - 200
```

---

## 8. Web 页面设计

### 8.1 推理后端页面

对象：Backend。

功能：

```text
列表
详情
系统只读
管理员可新增 user Backend（预留）
```

字段：

```text
名称
描述
协议族
来源 system/user
只读状态
```

不显示：

```text
版本配置
硬件参数
节点信息
```

### 8.2 后端版本页面

对象：BackendVersion。

功能：

```text
列表
详情
新增 user BackendVersion
编辑 user BackendVersion
删除 user BackendVersion
clone system BackendVersion 为 user BackendVersion
reload/sync catalog
```

不显示：

```text
硬件设备
节点镜像检查结果
节点状态
```

### 8.3 运行模板页面

对象：BackendRuntime。

功能：

```text
列表
详情
新增 user BackendRuntime
编辑 user BackendRuntime
删除 user BackendRuntime
clone system BackendRuntime
```

创建流程：

```text
选择 Backend
选择 BackendVersion
选择 vendor / hardware / runtime_distribution
自动继承 BackendVersion 软件参数
自动加载 BackendRuntime 或 runtime profile 硬件参数
用户调整模板
保存为 BackendRuntime catalog 文件
reload/sync 到 DB projection
```

显示字段：

```text
名称
Backend
BackendVersion
vendor
hardware_family
accelerator_api
runtime_distribution
image candidates
docker options
devices schema
env schema
args
被多少 NodeBackendRuntime 引用
ready_count 聚合统计
```

注意：

```text
运行模板页面不能添加节点。
运行模板页面不能修改 NodeBackendRuntime。
如果显示节点，只能是只读引用统计。
```

### 8.4 运行配置页面

对象：NodeBackendRuntime。

功能：

```text
新增
详情
编辑
检查/校验
删除
```

创建流程：

```text
选择 BackendRuntime
选择节点
选择/输入 image_ref
填写节点级 env/device/port overrides
保存为 NodeBackendRuntime
status = needs_check
点击检查
```

检查内容：

```text
节点在线
Docker 可用
image_ref 是否存在
GPU/vendor 设备是否可见
必要设备文件是否存在
status 更新为 ready/failed/needs_check
```

检查禁止：

```text
禁止修改 image_ref
禁止修改 config_snapshot_json
禁止刷新 source_backend_runtime_revision
禁止从 BackendRuntime 重新复制 snapshot
```

---

## 9. API 设计建议

### 9.1 Catalog reload

```text
POST /api/v1/backend-catalog/reload
```

行为：

```text
读取 Backend / BackendVersion / BackendRuntime system + user catalog 文件
校验 schema
计算 hash
刷新 DB projection
不影响 NodeBackendRuntime
不影响 Deployment
不影响 Instance
```

### 9.2 BackendVersion

```text
GET    /api/v1/backends
GET    /api/v1/backends/{id}/versions
POST   /api/v1/backends/{id}/versions
PATCH  /api/v1/backend-versions/{id}
DELETE /api/v1/backend-versions/{id}
POST   /api/v1/backend-versions/{id}/clone
```

规则：

```text
PATCH system BackendVersion -> 403
DELETE system BackendVersion -> 403
clone system -> 写 user catalog 文件
POST/PATCH user -> 写 user catalog 文件，再 reload/sync
```

### 9.3 BackendRuntime

```text
GET    /api/v1/backend-runtimes
POST   /api/v1/backend-runtimes
GET    /api/v1/backend-runtimes/{id}
PATCH  /api/v1/backend-runtimes/{id}
DELETE /api/v1/backend-runtimes/{id}
POST   /api/v1/backend-runtimes/{id}/clone
```

规则：

```text
BackendRuntime 是 catalog 文件主导对象。
POST/PATCH user BackendRuntime -> 写 user runtime catalog 文件，再 reload/sync。
PATCH system BackendRuntime -> 403。
clone system BackendRuntime -> 写 user runtime catalog 文件。
```

### 9.4 NodeBackendRuntime

```text
GET    /api/v1/node-backend-runtimes
POST   /api/v1/node-backend-runtimes
GET    /api/v1/node-backend-runtimes/{id}
PATCH  /api/v1/node-backend-runtimes/{id}
DELETE /api/v1/node-backend-runtimes/{id}
POST   /api/v1/node-backend-runtimes/{id}/check
```

规则：

```text
NodeBackendRuntime 是 DB 主导对象。
POST 创建时复制 BackendRuntime snapshot。
PATCH 修改运行字段后 status=needs_check。
check 只更新检查结果字段。
```

---

## 10. 关键验收标准

### 10.1 层级边界

必须满足：

```text
Backend 不含版本/硬件/节点参数。
BackendVersion 不含硬件/节点参数。
BackendRuntime 可含硬件/vendor/runtime 参数，但不含节点状态。
NodeBackendRuntime 绑定节点、image_ref、检查状态。
```

### 10.2 文件与 DB

必须满足：

```text
Backend / BackendVersion / BackendRuntime 文件为主。
Web/API 修改 user catalog 时，先写文件，再 reload/sync DB。
DB projection 不是唯一事实源。
NodeBackendRuntime DB 为主。
```

### 10.3 继承独立性

必须满足：

```text
BackendVersion 修改后，已有 BackendRuntime 不变。
BackendRuntime 修改后，已有 NodeBackendRuntime 不变。
NodeBackendRuntime check 不修改运行配置。
```

### 10.4 Web 边界

必须满足：

```text
运行模板页没有节点管理操作。
运行配置页管理 NodeBackendRuntime。
BackendRuntime 页面可配置硬件/vendor runtime 参数。
BackendVersion 页面只配置软件版本参数。
```

---

## 11. 建议文档落地路径

建议保存本文为：

```text
docs/design/backend-runtime-layered-catalog-design.md
```

并同步更新：

```text
docs/CURRENT.md
docs/README.md
docs/design/README.md
docs/design/runtime-template-node-runtime-snapshot.md
docs/design/model-runtime-node-wizard.md
docs/design/backend-runtime-runplan-docker.md
docs/backend-catalog-vendor-extension.md
docs/reports/model-runtime-node-wizard/open-issues-closeout.md
```