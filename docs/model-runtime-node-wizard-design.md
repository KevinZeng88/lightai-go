# LightAI Go 模型与运行配置向导设计

**文档状态**：Draft for review  
**建议放置路径**：`docs/model-runtime-node-wizard-design.md`  
**适用项目**：LightAI Go  
**日期**：2026-06-18

---

## 1. 背景与目标

当前 LightAI Go 已经完成 NVIDIA BackendRuntime / RunPlan / Docker lifecycle / health check / Docker logs / cleanup / observability 的主链路验证。下一阶段需要把模型、运行配置和实例启动流程产品化，让用户通过向导完成配置，而不是直接理解底层表、路径、Docker 参数和 RunPlan。

本设计目标：

```text
1. 新增模型时，先选择 Agent/节点，再从该节点浏览目录，选择模型文件或文件夹。
2. Agent 尽量自动扫描模型元信息，推测模型名称、格式、参数规模、量化方式和能力。
3. 一个逻辑模型可以有多个节点位置，每个位置可新增、禁用、删除、重扫和一致性核对。
4. 新增运行配置时，先选择 Backend、BackendVersion、Agent/节点和运行类型。
5. 当前运行类型以 Docker 为主，后续预留 command / external / kubernetes 等类型。
6. Agent 能列出本节点 Docker images，用户可选择已有 image，也可手工输入 image/tag/image_id。
7. 一个运行配置可以挂多个节点配置，每个节点配置可新增、禁用、删除和重新检测。
8. 启动实例时，模型位置和运行配置必须存在共同可用节点，才表示有运行可能。
9. 启动前通过 preflight 计算可运行节点、风险、缺失条件和 command preview。
10. 最终保持与既有 BackendRuntime / ModelLocation / NodeRunPlan 架构一致。
```

核心产品原则：

> 模型和运行配置都是逻辑对象；它们分别通过“节点位置”和“节点运行配置”落到具体 Agent。启动时取二者共同可用节点，再结合 GPU、端口、格式、Runtime 状态生成 RunPlan。

---

## 2. 设计边界

本设计不推翻当前已完成的主链路：

```text
Backend
BackendVersion
BackendRuntime
NodeBackendRuntime
ModelArtifact
ModelLocation
DeploymentPlan
RunPlanGroup
NodeRunPlan
Agent DockerExecutor
Docker logs
Health check
Cleanup
```

本设计重点是补齐产品化入口和管理能力：

```text
1. 模型新增向导
2. 模型节点位置管理
3. 运行配置新增向导
4. 运行配置节点配置管理
5. 启动前可运行节点计算
6. 目录浏览、模型扫描、Docker image 列表
7. 增删改查保护、审计和 i18n
```

---

## 3. 核心对象关系

### 3.1 逻辑模型与节点位置

```text
ModelArtifact
  ├── ModelLocation(node=A, path=/data/models/qwen)
  ├── ModelLocation(node=B, path=/models/qwen)
  └── ModelLocation(node=C, path=/mnt/models/qwen)
```

`ModelArtifact` 表示逻辑模型。  
`ModelLocation` 表示该模型在某个 Agent/节点上的实际文件或目录位置。

### 3.2 运行配置与节点运行配置

```text
BackendRuntime
  ├── NodeBackendRuntime(node=A, image=vllm/vllm-openai:latest, ready)
  ├── NodeBackendRuntime(node=B, image=0d307f1665d3, ready)
  └── NodeBackendRuntime(node=C, image=vllm/vllm-openai:latest, missing_image)
```

`BackendRuntime` 表示逻辑运行配置模板。  
`NodeBackendRuntime` 表示某个节点上该运行配置是否实际可用。

### 3.3 启动条件

实例启动时，系统计算：

```text
可运行节点 =
  模型的 active ModelLocation 节点集合
  ∩
  运行配置的 ready NodeBackendRuntime 节点集合
```

节点交集只是基础条件。真正可运行还要检查：

```text
1. ModelLocation 状态可用。
2. NodeBackendRuntime 状态为 ready。
3. BackendVersion 支持模型格式。
4. BackendRuntime 支持节点硬件厂商和运行类型。
5. GPU/NPU/CPU 资源足够。
6. 端口可用。
7. 模型路径可读。
8. Docker image 存在。
9. health_check 可生成。
10. command preview 可生成。
```

---

## 4. 模型对象设计

### 4.1 ModelArtifact

`ModelArtifact` 是逻辑模型，不应绑定单个节点路径。

建议字段：

```text
id
tenant_id
name
display_name
description
format
family
architecture
parameter_size
quantization
capabilities_json
canonical_fingerprint
identity_policy
metadata_json
status
created_by
created_at
updated_at
deleted_at
```

说明：

```text
1. name/display_name 可由扫描结果推测，用户可修改。
2. format 包括 huggingface、safetensors、gguf、ollama 等。
3. canonical_fingerprint 用于跨节点一致性核对。
4. status 可为 active、disabled、no_locations、deleted。
```

### 4.2 ModelLocation

`ModelLocation` 表示模型在某个 Agent/节点上的实际位置。

建议字段：

```text
id
tenant_id
model_artifact_id
node_id
path_type                 file / directory
model_root
relative_path
absolute_path
size_bytes
checksum
manifest_digest
metadata_json
discovered_name
discovered_format
match_status              exact_match / probable_match / mismatch / manual_attested / unknown
verification_status       verified / warning / manually_accepted / failed / missing / changed / unverified
manual_override
override_reason
override_by
override_at
last_scanned_at
last_error
status                    active / disabled / deleted
created_at
updated_at
```

路径规则：

```text
model_root + relative_path = absolute_path
```

示例：HuggingFace 目录模型

```text
model_root: /home/kzeng/models
relative_path: Qwen3-0.6B-Instruct-2512
absolute_path: /home/kzeng/models/Qwen3-0.6B-Instruct-2512
path_type: directory
```

示例：GGUF 文件模型

```text
model_root: /home/kzeng/models/Qwen3.5-9B-Q4
relative_path: Qwen3.5-9B-Q4_K_M.gguf
absolute_path: /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
path_type: file
```

---

## 5. 运行配置对象设计

### 5.1 Backend

`Backend` 表示推理后端类型：

```text
vLLM
SGLang
llama.cpp
Ollama
OpenAI-compatible external
```

Backend 不代表版本，也不代表节点实际可用状态。

### 5.2 BackendVersion

`BackendVersion` 表示后端软件版本、协议、能力和参数 schema。

示例：

```text
vllm-openai-latest
vllm-openai-0.9
sglang-openai-latest
llama-cpp-server
llama-cpp-server-metax
ollama-latest
```

职责：

```text
1. 定义支持的模型格式。
2. 定义 OpenAI-compatible 等协议。
3. 定义标准参数 schema。
4. 原则上硬件无关，除非确实是厂商适配版。
```

### 5.3 BackendRuntime

`BackendRuntime` 表示某个 BackendVersion 在某类运行环境下的逻辑运行配置模板。

示例：

```text
runtime.vllm.nvidia-docker
runtime.vllm.metax-docker
runtime.vllm.huawei-docker
runtime.llamacpp.nvidia-docker
runtime.llamacpp.cpu-docker
```

建议字段：

```text
id
tenant_id
backend_id
backend_version_id
name
runner_type              docker / command / external
runtime_type             docker / command
vendor                   nvidia / metax / huawei / cpu
accelerator_kind         gpu / npu / cpu
image_ref
command_json
default_args_json
params_mapping_json
generated_args_json
ports_json
mount_policy_json
device_policy_json
docker_options_json
custom_json
health_check_json
verification_json
managed_by               system / user
is_editable
status
created_at
updated_at
deleted_at
```

规则：

```text
1. system-managed BackendRuntime 默认只读。
2. 用户要修改系统模板时，应复制为 user-managed Runtime。
3. BackendRuntime 是逻辑模板，不等于某个节点已经 ready。
4. 某节点是否可用，由 NodeBackendRuntime 表示。
```

### 5.4 NodeBackendRuntime

`NodeBackendRuntime` 表示某个 BackendRuntime 在某个 Agent/节点上的实际可用状态和节点级覆盖。

建议字段：

```text
id
tenant_id
node_id
backend_runtime_id
image_ref
image_id
image_digest
image_present
runner_available
docker_available
driver_version
toolkit_version
device_check_json
override_json
enabled_blocks_json
status
status_reason
last_checked_at
last_error
created_at
updated_at
```

状态建议：

```text
ready
missing_image
driver_mismatch
toolkit_missing
adapter_missing
template_only
unsupported_device
invalid
unknown
disabled
deleted
```

说明：

```text
1. 同一个 BackendRuntime 可以挂多个 NodeBackendRuntime。
2. 不同节点可以使用不同 image_ref/image_id。
3. 节点级覆盖不能破坏系统模板，只影响该节点。
4. NodeBackendRuntime status=ready 才能参与启动。
```

---

## 6. Web 页面与菜单设计

建议 Web 菜单保持简单：

```text
模型
运行配置
实例
```

用户看到的是业务对象；底层对象在详情页中分区展示。

### 6.1 模型页面

模型列表展示：

```text
模型名称
格式
参数规模
量化
能力
节点位置数量
可用位置数量
状态
最近扫描时间
操作
```

模型详情展示：

```text
基本信息
扫描元信息
节点位置列表
一致性核对结果
历史运行记录
审计记录
```

### 6.2 运行配置页面

运行配置列表展示：

```text
名称
Backend
BackendVersion
运行类型
Vendor
默认 Image
节点配置数量
Ready 节点数量
状态
操作
```

运行配置详情展示：

```text
基本信息
默认参数
Docker 参数
节点配置列表
Command Preview 示例
审计记录
```

### 6.3 实例页面

实例新增向导展示：

```text
选择模型
选择运行配置
可运行节点
GPU/端口
参数确认
Command Preview
启动
```

实例详情展示：

```text
状态
RunPlan
Command Preview
Docker logs
健康检查
阶段耗时
GPU lease
容器状态
```

---

## 7. 新增模型向导

### 7.1 入口

```text
模型 → 新增模型
```

### 7.2 流程

```text
Step 1：选择 Agent/节点
Step 2：浏览模型目录
Step 3：选择文件或文件夹
Step 4：Agent 扫描模型元信息
Step 5：系统推测模型名称、格式、参数规模、量化方式
Step 6：用户确认或修改模型名称
Step 7：保存 ModelArtifact + 第一个 ModelLocation
```

### 7.3 目录浏览能力

Agent 需要提供受控目录浏览能力。

建议 API：

```text
GET /api/v1/nodes/{node_id}/files?root=...&path=...&type=model
POST /api/v1/nodes/{node_id}/model-paths/scan
```

目录浏览要求：

```text
1. 只允许浏览 Agent 配置中的 allowed_model_roots。
2. 默认只列当前目录，不递归。
3. 支持分页。
4. 支持文件类型过滤。
5. 支持显示文件大小、修改时间、是否目录。
6. 不允许浏览 /etc、/root、/home/其他用户等未授权目录。
7. 错误要明确：路径不存在、无权限、超出 allowed roots、扫描超时。
```

Agent 配置建议：

```yaml
model_browser:
  enabled: true
  allowed_roots:
    - /home/kzeng/models
    - /data/models
    - /data/part2/MX-C500/model
  max_entries: 1000
  max_scan_depth: 2
  follow_symlinks: false
```

### 7.4 模型扫描

Agent 扫描目录或文件时，尽量识别：

```text
format
model_name
architecture
family
parameter_size
quantization
tokenizer
config.json
generation_config.json
safetensors files
gguf metadata
size_bytes
manifest_digest / checksum
capabilities guess
```

扫描规则：

```text
1. 目录模型优先读取 config.json、tokenizer_config.json、generation_config.json。
2. safetensors 模型读取 index 和文件列表。
3. GGUF 模型读取 GGUF metadata。
4. checksum 可以异步或按需深度扫描，避免大文件阻塞。
5. 扫描结果应返回 confidence。
```

---

## 8. 为模型添加节点位置

### 8.1 入口

```text
模型详情 → 节点位置 → 添加节点位置
```

### 8.2 流程

```text
Step 1：选择已有模型
Step 2：选择 Agent/节点
Step 3：浏览目录
Step 4：选择文件或文件夹
Step 5：Agent 扫描
Step 6：系统与已有模型做一致性核对
Step 7：显示 exact/probable/mismatch/manual_attested
Step 8：保存 ModelLocation
```

### 8.3 一致性核对

一致性核对应分层，不要只靠 checksum。

快速核对：

```text
format
architecture
family
parameter_size
quantization
config.json 关键字段
tokenizer 元信息
gguf metadata
文件大小
safetensors index
```

深度核对：

```text
manifest_digest
checksum
tokenizer hash
全部权重文件 checksum
```

核对结果：

```text
exact_match
probable_match
mismatch
manual_attested
unknown
```

策略：

```text
1. exact_match：可直接保存，verification_status=verified。
2. probable_match：可保存，但 verification_status=warning。
3. mismatch：默认不允许保存为同一模型。
4. manual_attested：需要用户强制确认，记录 reason/operator/time/audit。
5. unknown：允许保存为 unverified，但启动时提示风险。
```

---

## 9. 模型与模型位置增删改规则

### 9.1 ModelArtifact 删除

删除模型前必须检查：

```text
1. 是否有 active deployment 引用。
2. 是否有 running/pending/starting instance 引用。
3. 是否有 RunPlan 或 audit 需要保留引用。
```

策略：

```text
1. 默认 soft delete / disabled。
2. 无引用时允许 hard delete。
3. 有历史运行记录时保留只读记录。
4. 删除需要 audit log。
```

### 9.2 ModelLocation 删除

删除某个节点位置前检查：

```text
1. 是否有 running/pending/starting instance 正在使用该 location。
2. 是否有 deployment 默认绑定该 location。
3. 是否是该模型唯一 location。
```

策略：

```text
1. 正在使用时禁止删除。
2. 唯一 location 删除后 ModelArtifact 仍可保留，但状态变为 no_locations。
3. 删除后启动向导不再把该节点作为可运行节点。
4. 删除需要 audit log。
```

### 9.3 ModelLocation 修改

允许修改：

```text
display name
description
status active/disabled
override reason
metadata 备注
```

不建议直接修改：

```text
node_id
absolute_path
model_root
relative_path
path_type
```

如果路径错了，建议新增 location 后禁用旧 location，保留审计链路。

---

## 10. 新增运行配置向导

### 10.1 入口

```text
运行配置 → 新增运行配置
```

### 10.2 流程

```text
Step 1：选择 Backend
Step 2：选择 BackendVersion
Step 3：选择 Agent/节点
Step 4：选择运行类型，目前仅 Docker，将来支持 command/external
Step 5：Agent 列出该节点 Docker images
Step 6：选择 image 或手工输入 image_ref
Step 7：系统根据 Backend/Version/节点硬件推荐 BackendRuntime 模板
Step 8：检测 Docker / GPU runtime / image / driver
Step 9：加载默认参数
Step 10：用户确认，保存 BackendRuntime 或 RuntimeProfile + NodeBackendRuntime
```

### 10.3 运行类型

当前只开放：

```text
docker
```

未来预留：

```text
command
systemd
external
kubernetes
```

底层 `runner_type` 不应写死为 Docker。

### 10.4 Docker image 列表

Agent 应提供 Docker image 列表。

建议 API：

```text
GET /api/v1/nodes/{node_id}/docker/images
```

返回字段：

```text
repository
tag
image_id
digest
created_at
size
labels
repo_tags
repo_digests
```

用户可选择：

```text
1. 已有 image tag，例如 vllm/vllm-openai:latest。
2. image ID，例如 0d307f1665d3。
3. 手工输入新 image_ref。
```

保存时区分：

```text
image_ref：用户选择或输入的值
image_id：Agent 检测到的实际 ID
image_digest：如果可获取
image_present：true/false
```

### 10.5 运行配置保存策略

如果用户基于系统模板创建：

```text
BackendRuntime managed_by=system 不直接修改
复制为 managed_by=user 的自定义运行配置
```

如果只是把系统 Runtime 启用到某个节点：

```text
创建或更新 NodeBackendRuntime
```

建议 UX：

```text
新增运行配置：
  创建用户自定义 BackendRuntime + 第一个 NodeBackendRuntime

启用系统运行配置：
  不改 BackendRuntime，只创建 NodeBackendRuntime
```

实现时应按当前代码已有对象适配，不重复造 RuntimeProfile 概念，除非项目已存在同义对象。

---

## 11. 为运行配置添加节点

### 11.1 入口

```text
运行配置详情 → 节点配置 → 添加节点
```

### 11.2 流程

```text
Step 1：选择 Agent/节点
Step 2：列出该节点 Docker images
Step 3：选择或输入 image_ref
Step 4：检测 Docker / GPU / driver / toolkit / vendor adapter
Step 5：显示检测结果
Step 6：保存 NodeBackendRuntime
```

### 11.3 节点级覆盖

NodeBackendRuntime 可覆盖：

```text
image_ref
enabled docker option blocks
custom args
custom env
custom docker options
device policy overrides
```

但不能修改 system BackendRuntime 本身。

### 11.4 删除节点配置

允许从运行配置中删除某个节点配置，即删除或禁用 NodeBackendRuntime。

删除前检查：

```text
1. 是否有 running/pending/starting instance 使用该 NodeBackendRuntime。
2. 是否有 deployment 默认绑定该 node runtime。
3. 是否有 active RunPlan 引用。
```

策略：

```text
1. 正在使用时禁止删除。
2. 可 disabled，禁用后启动向导不再选择该节点。
3. 删除/禁用需要 audit log。
```

---

## 12. 运行配置删除规则

### 12.1 BackendRuntime 删除

删除前检查：

```text
1. 是否 system-managed。
2. 是否被 NodeBackendRuntime 引用。
3. 是否被 Deployment 引用。
4. 是否有历史 RunPlan 引用。
```

策略：

```text
1. system-managed 不允许删除，只能隐藏或禁用。
2. user-managed 若无引用，可删除。
3. 有引用时 soft delete / disabled。
4. 删除前提示影响的节点配置和部署。
```

### 12.2 NodeBackendRuntime 删除

原则：

```text
只删除节点可用性，不删除逻辑运行配置。
```

规则见第 11.4 节。

---

## 13. 新增实例启动向导

### 13.1 入口

```text
实例 → 新增实例
```

### 13.2 流程

```text
Step 1：选择模型
Step 2：选择运行配置
Step 3：系统计算共同可运行节点
Step 4：选择节点，或自动选择
Step 5：GPU 自动/手动
Step 6：端口自动/手动
Step 7：参数确认
Step 8：展示 command preview
Step 9：启动
```

### 13.3 可运行节点计算

基础交集：

```text
ModelArtifact.active ModelLocation.node_id
∩
BackendRuntime.active NodeBackendRuntime.node_id
```

真正可运行条件：

```text
1. ModelLocation.status active。
2. ModelLocation.verification_status 允许运行。
3. NodeBackendRuntime.status ready。
4. BackendVersion 支持模型 format。
5. BackendRuntime 支持节点 vendor/accelerator。
6. GPU 资源足够。
7. 端口可用。
8. 模型路径可读。
9. image present。
10. health_check 配置有效。
```

UI 展示建议：

```text
可运行节点：
  KZ-LAPTOP    ✅ 模型存在，Runtime ready，GPU 可用
  node-02      ⚠️ 模型存在，但 image missing
  node-03      ❌ Runtime ready，但模型位置不存在
```

没有可运行节点时，提示：

```text
该模型和运行配置没有共同可用节点。

可解决：
1. 为模型添加该节点位置。
2. 为运行配置添加该节点配置。
3. 修复节点 Runtime 状态。
4. 检查模型格式是否被 BackendVersion 支持。
```

---

## 14. API 建议

### 14.1 文件浏览与模型扫描

```text
GET  /api/v1/nodes/{node_id}/files
POST /api/v1/nodes/{node_id}/model-paths/scan
```

参数示例：

```text
root
path
limit
cursor
file_types
```

### 14.2 模型与位置

```text
GET    /api/v1/model-artifacts
POST   /api/v1/model-artifacts
GET    /api/v1/model-artifacts/{id}
PATCH  /api/v1/model-artifacts/{id}
DELETE /api/v1/model-artifacts/{id}

GET    /api/v1/model-artifacts/{id}/locations
POST   /api/v1/model-artifacts/{id}/locations
PATCH  /api/v1/model-artifacts/{id}/locations/{location_id}
DELETE /api/v1/model-artifacts/{id}/locations/{location_id}
POST   /api/v1/model-artifacts/{id}/locations/{location_id}/rescan
POST   /api/v1/model-artifacts/{id}/locations/{location_id}/attest
```

### 14.3 Docker images

```text
GET /api/v1/nodes/{node_id}/docker/images
```

### 14.4 运行配置与节点配置

```text
GET    /api/v1/backend-runtimes
POST   /api/v1/backend-runtimes
GET    /api/v1/backend-runtimes/{id}
PATCH  /api/v1/backend-runtimes/{id}
DELETE /api/v1/backend-runtimes/{id}
POST   /api/v1/backend-runtimes/{id}/clone

GET    /api/v1/backend-runtimes/{id}/nodes
POST   /api/v1/backend-runtimes/{id}/nodes
PATCH  /api/v1/backend-runtimes/{id}/nodes/{node_runtime_id}
DELETE /api/v1/backend-runtimes/{id}/nodes/{node_runtime_id}

GET    /api/v1/nodes/{node_id}/backend-runtimes
POST   /api/v1/nodes/{node_id}/backend-runtimes/check
POST   /api/v1/nodes/{node_id}/backend-runtimes/enable
```

可以按当前项目已有 API 命名适配，但语义必须覆盖。

### 14.5 启动前可运行性检查

建议新增：

```text
POST /api/v1/deployments/preflight
```

输入：

```json
{
  "model_artifact_id": "...",
  "backend_runtime_id": "...",
  "node_id": "",
  "gpu_policy": "auto",
  "host_port": 0
}
```

输出：

```json
{
  "can_run": true,
  "candidate_nodes": [
    {
      "node_id": "...",
      "status": "ready",
      "model_location_id": "...",
      "node_backend_runtime_id": "...",
      "warnings": []
    }
  ],
  "errors": [],
  "warnings": []
}
```

---

## 15. Agent 能力要求

### 15.1 目录浏览

Agent 实现受控目录浏览：

```text
list directory
stat file
validate allowed root
pagination
no recursive default
```

### 15.2 模型扫描

Agent 实现：

```text
scan model path
read config files
read GGUF metadata，如可行
compute fast fingerprint
optionally compute checksum
```

### 15.3 Docker image 列表

Agent 实现：

```text
docker images list
docker image inspect
return tag/id/digest/size/created
```

### 15.4 Runtime check

Agent 实现或协助 Server 判断：

```text
docker available
image present
GPU/NPU vendor available
driver/toolkit available
vendor adapter available
```

---

## 16. 权限、安全与审计

### 16.1 RBAC

建议权限：

```text
model_artifact:read
model_artifact:write
model_location:read
model_location:write
backend_runtime:read
backend_runtime:write
node_backend_runtime:read
node_backend_runtime:write
node_file:read
node_docker_image:read
model_deployment:read
model_deployment:write
model_deployment:start
model_deployment:stop
```

### 16.2 目录浏览安全

必须：

```text
1. 限制 allowed_model_roots。
2. 禁止路径穿越，例如 ../../。
3. 不跟随 symlink，除非明确配置。
4. 限制返回条数。
5. 限制扫描深度。
6. 对访问失败返回明确错误。
7. 审计目录浏览与模型扫描。
```

### 16.3 删除审计

以下操作必须写 audit log：

```text
删除模型
禁用模型
删除 ModelLocation
手工 attestation
删除/禁用 BackendRuntime
删除/禁用 NodeBackendRuntime
启动/停止/删除实例
```

---

## 17. Web i18n 要求

历史上项目多次发生 i18n key 泄露。新增页面必须严格遵守：

```text
1. 所有新增文案必须使用 i18n。
2. zh-CN 和 en-US key 必须同步。
3. leaf 必须是 string，不能是 object。
4. 页面不得显示 modelWizard.xxx、runtimeWizard.xxx 等 key。
5. 表单 label、placeholder、tooltip、按钮、错误提示、状态枚举都要有翻译。
6. i18nMissingKeys.test.mjs 必须覆盖新增 key。
```

建议新增 key namespace：

```text
modelWizard.*
runtimeWizard.*
startWizard.*
modelLocations.*
nodeRuntime.*
fileBrowser.*
dockerImages.*
preflight.*
```

---

## 18. 日志与可观测性

新增向导和 API 必须输出阶段日志：

```text
model.wizard.scan.started
model.wizard.scan.completed
model.location.compare.started
model.location.compare.completed
runtime.wizard.image_list.started
runtime.wizard.image_list.completed
runtime.node.check.started
runtime.node.check.completed
deployment.preflight.started
deployment.preflight.completed
```

日志字段建议：

```text
operation_id
node_id
agent_id
model_artifact_id
model_location_id
backend_id
backend_version_id
backend_runtime_id
node_backend_runtime_id
stage
status
duration_ms
error
```

敏感信息不要输出。

---

## 19. 测试要求

### 19.1 单元测试

覆盖：

```text
ModelLocation path normalization
allowed roots validation
model metadata scan parser
model consistency compare
BackendRuntime clone
NodeBackendRuntime add/delete/disable
preflight node intersection
delete protection
i18n missing key
```

### 19.2 API 测试

覆盖：

```text
list files
scan model path
create ModelArtifact + ModelLocation
add second ModelLocation
attest mismatch
list docker images
enable runtime on node
delete/disable node runtime
preflight candidate nodes
```

### 19.3 E2E

在 NVIDIA 环境至少覆盖：

```text
1. 浏览 /home/kzeng/models。
2. 选择 Qwen3-0.6B-Instruct-2512 目录。
3. 扫描并创建模型。
4. 选择 vLLM backend/version。
5. 列出 Docker images。
6. 选择 vllm/vllm-openai:latest。
7. 创建运行配置并启用节点。
8. 启动实例。
9. /v1/models PASS。
10. Docker logs PASS。
11. stop/cleanup PASS。
```

---

## 20. 兼容当前实现

当前已经有：

```text
Backend Catalog
BackendVersion
BackendRuntime
NodeBackendRuntime
ModelArtifact
ModelLocation
DeploymentPlan
RunPlanGroup
NodeRunPlan
Docker lifecycle
Docker logs
E2E
```

本阶段不要推翻现有实现，而是在现有基础上产品化：

```text
1. 增加向导 API。
2. 增强 Web 流程。
3. 补 Agent 文件浏览、模型扫描、Docker image 列表能力。
4. 补增删改查和删除保护。
5. 补 preflight 可运行节点计算。
6. 补 i18n、日志和测试。
```

---

## 21. 分阶段实施建议

### Phase 1：后端能力和 API

```text
Agent 文件浏览
Agent 模型扫描
Agent Docker image 列表
ModelLocation PATCH/DELETE
NodeBackendRuntime PATCH/DELETE
BackendRuntime clone
Deployment preflight
删除保护
审计日志
```

### Phase 2：Web 向导

```text
新增模型向导
添加模型节点位置向导
新增运行配置向导
添加运行配置节点向导
新增实例启动向导
可运行节点计算展示
```

### Phase 3：测试与收口

```text
API tests
Web i18n tests
NVIDIA E2E
日志可观测性检查
文档和验收报告
```

---

## 22. 验收标准

### 22.1 模型向导验收

```text
1. 用户能选择 Agent。
2. 用户能在 allowed_model_roots 内浏览目录。
3. 用户能选择文件或文件夹。
4. Agent 能扫描模型元信息。
5. 系统能自动填入模型名称，用户可修改。
6. 能创建 ModelArtifact + 第一个 ModelLocation。
7. 能为已有模型添加第二个节点位置。
8. 添加第二位置时能做一致性核对。
9. 能禁用/删除 ModelLocation，并有删除保护。
```

### 22.2 运行配置向导验收

```text
1. 用户能选择 Backend。
2. 用户能选择 BackendVersion。
3. 用户能选择 Agent。
4. 用户能选择 runner_type=docker。
5. 系统能列出 Agent 上的 Docker images。
6. 用户能选择已有 image 或手工输入 image_ref。
7. 系统能创建或启用 NodeBackendRuntime。
8. 能为运行配置添加第二个节点配置。
9. 能禁用/删除 NodeBackendRuntime，并有删除保护。
```

### 22.3 启动向导验收

```text
1. 选择模型和运行配置后，系统能计算共同可运行节点。
2. 节点不可运行时能说明原因。
3. 可运行节点能进入 GPU/端口选择。
4. 能生成 command preview。
5. 能启动 Docker。
6. /v1/models PASS。
7. Docker logs 可查看。
8. stop/cleanup 无残留。
```

### 22.4 安全验收

```text
1. 不能浏览 allowed_model_roots 外目录。
2. 不能路径穿越。
3. 不能删除正在运行实例使用的模型位置或节点运行配置。
4. system-managed BackendRuntime 不能直接修改或删除。
5. i18n 无 key 泄露。
6. 日志无敏感信息泄露。
```

---

## 23. 关键设计结论

```text
1. 模型不是单个文件路径，而是逻辑模型 + 多个节点位置。
2. 运行配置不是单个 Docker image，而是逻辑运行配置 + 多个节点运行配置。
3. Agent 是模型位置和运行配置落地的节点，不是模型或运行配置的唯一归属。
4. 启动实例时必须计算模型位置与节点运行配置的共同可用节点。
5. 节点交集只是基础条件，还要检查 Runtime ready、模型格式、GPU、端口、路径、image 和 health_check。
6. 所有新增、删除、禁用、手工确认和启动停止都要有审计和日志。
7. Web 向导是产品化入口，底层仍保持现有 BackendRuntime / ModelLocation / NodeRunPlan 架构。
```
