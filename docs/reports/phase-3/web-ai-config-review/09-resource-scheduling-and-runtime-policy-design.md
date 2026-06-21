# LightAI Go Web AI 资源限制、调度与多实例设计

建议仓库路径：

```text
docs/reports/phase-3/web-ai-config-review/09-resource-scheduling-and-runtime-policy-design.md
```

状态：

```text
Status: REVISED_FOR_PHASE_1_IMPLEMENTATION
```

## 1. 设计背景

当前 LightAI Go 已经具备模型、节点、GPU、后端、运行模板、节点运行配置、RunPlan、模型部署和模型实例等基础对象。但手工验证暴露出几个产品层问题：

1. 模型部署仍偏向“单机单实例”使用体验。
2. 资源限制入口不足，尤其缺少显存使用率、后端性能参数、容器资源限制等可理解配置。
3. GPU 选择已经从 `gpu_ids` 升级为 `accelerator_ids`，但页面和资源策略还没有完全按 vendor-neutral 方式组织。
4. 后续需要支持多台服务器、多副本、多实例、跨节点运行、运行失败重试、不同调度策略和不同 GPU vendor。
5. 第一阶段仍希望先在单机单副本场景跑通，但不能把对象和页面写死成单节点单容器。

因此本设计的目标是：

```text
当前阶段：单机单副本可运行
设计边界：多节点、多实例、资源调度可扩展
实现策略：第一阶段 scheduler/placement 可以退化为单节点候选选择和资源校验
```

## 2. 设计原则

### 2.1 Deployment 与 Instance 必须分离

```text
ModelDeployment
= 部署意图 / 服务定义 / 资源策略 / 调度策略

ModelInstance
= 调度后实际运行出来的实例
```

即使当前一个部署只有一个实例，也不能把两者合并。

未来：

```text
Deployment: qwen3-chat-service
├── Instance 1: node-a / GPU 0 / port 8001
├── Instance 2: node-b / GPU 1 / port 8002
└── Instance 3: node-c / GPU 0,1 / port 8003
```

当前第一阶段：

```text
Deployment: qwen3-chat-service
└── Instance 1: local-node / selected accelerators / selected port
```

### 2.2 Resource Policy 与 Resolved RunPlan 必须分离

用户配置的是资源策略，不应直接等同于容器最终参数。

```text
Deployment Resource Policy
= 用户希望如何使用资源

ResolvedRunPlan
= 某个实例最终如何运行
```

示例：

```text
Deployment policy:
- replicas = 1
- gpu_count = 1
- gpu_memory_utilization = 0.8
- max_model_len = 32768
- backend = vLLM

ResolvedRunPlan:
- node_id = local-node
- accelerator_ids = [...]
- args = ["--gpu-memory-utilization", "0.8", "--max-model-len", "32768"]
- docker DeviceRequest / device paths
- ports / volumes / env / health check
```

### 2.3 Vendor-neutral 输入，vendor-specific 输出

用户层和部署层使用：

```text
accelerator_ids
gpu_count
gpu_vendor
resource_policy
```

RunPlan 层才落到具体 vendor binding：

```text
NVIDIA:
- Docker DeviceRequest
- CUDA_VISIBLE_DEVICES

MetaX:
- /dev/mxcd
- /dev/dri/cardX
- /dev/dri/renderDXXX
- CUDA_VISIBLE_DEVICES

Huawei:
- 通过 vendor runtime template/catalog 定义
- 当前不要凭空硬编码未知参数

CPU:
- 无 GPU device binding
```

### 2.4 Backend-specific 性能参数不能和 Vendor binding 混淆

显存使用率通常不是 GPU driver 的统一参数，而是后端 serving 参数。

例如：

```text
vLLM:
- --gpu-memory-utilization
- --max-model-len
- --tensor-parallel-size
- --max-num-seqs
- --dtype

SGLang:
- --mem-fraction-static
- --tp-size
- --context-length
- --max-running-requests

llama.cpp:
- --ctx-size
- --n-gpu-layers
- --batch-size
- --threads
```

UI 可以统一叫“资源与性能”，但底层必须按 backend/vendor 映射。

### 2.5 第一阶段可简单，但命名和边界不能简单化

第一阶段可以只做：

```text
replicas = 1
单节点候选
手动选择节点/卡
资源校验
单实例 RunPlan
```

但不要引入如下单机化命名：

```text
single_node
gpu_id
container_only
one_instance
```

应使用：

```text
replicas
placement
candidate
node_id
accelerator_ids
resource_policy
runtime_overrides
resolved_runplan
```

## 3. 核心对象边界

### 3.1 ModelArtifact

表示模型资产。

职责：

```text
模型名称
模型格式
模型架构
参数规模
量化信息
上下文长度
模型能力
默认测试方式
扫描 metadata
```

不负责：

```text
节点路径
运行参数
容器参数
调度结果
```

### 3.2 ModelLocation

表示模型在某个节点上的实际路径。

字段语义：

```text
model_artifact_id
node_id
host_path
exists / verified
checksum / consistency status
last_scanned_at
```

多节点场景下，一个模型可以有多个 location。

### 3.3 ModelCapability

表示模型可用能力。必须支持人工修正并持久化。

能力建议：

```text
chat
completion
embedding
rerank
vision
tool_calling
structured_output
```

能力来源：

```text
scan
inferred
user_override
backend_probe
```

默认测试方式：

```text
auto
chat
completion
embedding
rerank
```

### 3.4 Backend / BackendVersion

表示后端能力定义。

例如：

```text
vLLM
SGLang
llama.cpp
```

职责：

```text
支持模型格式
支持 API 类型
支持性能参数 schema
支持 vendor
支持运行模式
```

### 3.5 BackendRuntime

表示系统运行模板。

职责：

```text
默认镜像
默认命令
默认参数
默认端口
默认 health check
默认 vendor binding 模板
```

普通用户通常不直接编辑，放在“配置/运行模板”。

### 3.6 NodeBackendRuntime

表示某个节点上的实际运行配置。

职责：

```text
某节点使用哪个 image
某节点的 env/volumes/ports/devices
某节点的 vendor-specific 参数
某节点的默认资源与性能参数
```

### 3.7 ModelDeployment

表示部署意图 / 服务定义 / 资源策略。

职责：

```text
部署名称
模型
目标后端
副本数
资源策略
调度策略
运行参数覆盖
服务别名 / served model name，后续
```

不应该直接等同于一个容器。

### 3.8 ModelInstance

表示实际运行实例。

职责：

```text
属于哪个 deployment
运行在哪个 node
使用哪些 accelerator_ids
使用哪个 model_location
使用哪个 node_backend_runtime
对应哪个 resolved_runplan
当前状态
容器/进程信息
endpoint
```

### 3.9 Placement / Candidate / NodeRunPlan

表示调度候选和调度结果。

职责：

```text
候选节点
候选模型位置
候选 NBR
候选 accelerator_ids
资源校验结果
预估/实际端口
资源策略解析结果
```

### 3.10 ResolvedRunPlan

表示最终运行计划。

职责：

```text
image
command
args
env
volumes
ports
device_binding
health_check
equivalent docker command
```

## 4. 调度设计

### 4.1 调度输入

第一阶段建议输入：

```json
{
  "model_id": "...",
  "backend_id": "...",
  "node_backend_runtime_id": "...",
  "replicas": 1,
  "placement_policy": {
    "mode": "manual",
    "node_ids": ["..."],
    "accelerator_ids": ["..."]
  },
  "resource_policy": {
    "gpu_count": 1,
    "gpu_memory_utilization": 0.8,
    "cpu_limit": null,
    "memory_limit_bytes": null,
    "shm_size": "8gb"
  },
  "backend_params": {
    "max_model_len": 32768,
    "tensor_parallel_size": 1,
    "dtype": "auto"
  },
  "runtime_overrides": {
    "extra_args": [],
    "extra_env": {},
    "extra_volumes": []
  }
}
```

第一阶段不一定一次性实现全部字段，但 UI/设计必须按该方向组织。

### 4.2 候选筛选

候选节点必须满足：

```text
节点在线
节点有目标模型 ModelLocation
节点有匹配 NodeBackendRuntime
NBR backend 支持目标模型格式
NBR vendor 支持节点 GPU vendor
请求的 accelerator_ids 存在
请求的 accelerator_ids 未被 active lease 占用
后端性能参数合法
容器资源限制合法
端口可用
```

### 4.3 第一阶段退化调度

当前可以实现为：

```text
1. 用户手动选择节点/NBR/accelerator_ids
2. scheduler 校验这些选择是否合法
3. 若合法，返回唯一 candidate
4. resolver 生成 ResolvedRunPlan
```

这不是“没有调度”，而是：

```text
单候选调度 + 资源校验
```

### 4.4 未来调度策略

后续扩展：

```text
least_allocated
most_available_memory
spread_across_nodes
pack_by_node
prefer_same_model_location
prefer_same_vendor
anti_affinity
priority
quota-aware scheduling
retry_on_failure
failover_to_next_candidate
```

这些策略第一阶段不实现，但对象和页面不能阻断未来扩展。

## 5. 资源策略设计

### 5.1 通用资源策略

建议逻辑结构：

```json
{
  "replicas": 1,
  "gpu_count": 1,
  "accelerator_ids": ["..."],
  "cpu_limit": 4,
  "memory_limit_bytes": 17179869184,
  "shm_size": "8gb",
  "ulimits": {
    "memlock": "-1"
  }
}
```

### 5.2 后端性能参数

建议按 backend 拆分。

#### vLLM

```json
{
  "gpu_memory_utilization": 0.8,
  "max_model_len": 32768,
  "tensor_parallel_size": 1,
  "max_num_seqs": 256,
  "dtype": "auto"
}
```

映射：

```text
--gpu-memory-utilization 0.8
--max-model-len 32768
--tensor-parallel-size 1
--max-num-seqs 256
--dtype auto
```

#### SGLang

```json
{
  "mem_fraction_static": 0.8,
  "tp_size": 1,
  "context_length": 32768,
  "max_running_requests": 128
}
```

映射：

```text
--mem-fraction-static 0.8
--tp-size 1
--context-length 32768
--max-running-requests 128
```

#### llama.cpp

```json
{
  "ctx_size": 8192,
  "n_gpu_layers": -1,
  "batch_size": 512,
  "threads": 8
}
```

映射：

```text
--ctx-size 8192
--n-gpu-layers -1
--batch-size 512
--threads 8
```

### 5.3 Vendor binding

#### NVIDIA

```text
DeviceBinding.mode = nvidia_device_request
CUDA_VISIBLE_DEVICES = selected visible IDs
Docker DeviceRequest = selected accelerator IDs or mapped indices
```

#### MetaX

```text
DeviceBinding.mode = metax_device_paths
/dev/mxcd
/dev/dri/cardX
/dev/dri/renderDXXX
CUDA_VISIBLE_DEVICES as framework-level visibility control
```

注意：

```text
不得使用 MACA_VISIBLE_DEVICE
不得使用 METAX_VISIBLE_DEVICES
```

#### Huawei

当前只保留设计位置：

```text
通过 backend/runtime catalog 或 vendor template 定义
不要凭空写死未知参数
```

#### CPU

```text
DeviceBinding.mode = cpu_none
无 GPU env
无 GPU device
```

### 5.4 Resource Parameter → Existing Field Mapping (Phase 1/2)

Phase 1 通过现有字段承载资源参数，不新增 first-class column。

| 参数 | 存储字段 | 阶段 |
|------|---------|------|
| gpu_memory_utilization (vLLM) | parameters_json | Phase 1 |
| max_model_len (vLLM) | parameters_json | Phase 1 |
| tensor_parallel_size (vLLM) | parameters_json | Phase 1 |
| max_num_seqs (vLLM) | parameters_json | Phase 1 |
| dtype (vLLM) | parameters_json | Phase 1 |
| mem_fraction_static (SGLang) | parameters_json | Phase 1 |
| tp_size (SGLang) | parameters_json | Phase 1 |
| context_length (SGLang) | parameters_json | Phase 1 |
| max_running_requests (SGLang) | parameters_json | Phase 1 |
| ctx_size (llama.cpp) | parameters_json | Phase 1 |
| n_gpu_layers (llama.cpp) | parameters_json | Phase 1 |
| batch_size (llama.cpp) | parameters_json | Phase 1 |
| threads (llama.cpp) | parameters_json | Phase 1 |
| shm_size | docker_json / config_snapshot_json | Phase 1 |
| ulimits | docker_json / config_snapshot_json | Phase 1 |
| ipc / privileged / security_opt | docker_json / config_snapshot_json | Phase 1 |
| cpu_limit | P2 — 当前无字段 | Phase 2+ |
| memory_limit_bytes | P2 — 当前无字段 | Phase 2+ |
| resource_policy first-class column | 不在 Phase 1 新增 | Phase 4+ |
| placement_policy first-class column | 不在 Phase 1 新增，使用现有 placement_json | Phase 4+ |

**parameter_defs_json 权威来源**：

- `BackendVersion.parameter_defs_json` 是后端参数定义的权威来源。
- 前端资源与性能编辑器应优先读取 parameter_defs_json 生成适用参数。
- 不要在前端硬编码所有 backend 参数列表。
- 缺失字段再使用 catalog/runtime 默认值。

## 6. 失败重试设计

### 6.1 第一阶段

第一阶段可以不实现自动重试，但需要记录失败原因：

```text
failed stage
error message
operation_id
runplan
container logs
```

### 6.2 后续扩展

后续支持：

```text
restart_policy
max_retries
retry_backoff_seconds
retry_on_exit_code
failover_to_next_candidate
manual_retry
```

重试必须基于 deployment/instance 边界，不应直接重启旧容器状态。

## 7. UI 设计

### 7.1 模型编辑页

分区：

```text
可编辑信息：
- 模型显示名称
- 描述
- 标签
- 模型能力
- 默认测试方式

扫描事实，只读：
- 文件大小
- checksum
- format
- architecture
- quantization
- parameter count
- context length
- model path/location
```

能力编辑：

```text
[ ] Chat
[ ] Completion
[ ] Embedding
[ ] Rerank
[ ] Vision
[ ] Tool Calling
[ ] Structured Output
```

保存后必须持久化。

### 7.2 NBR 运行配置页

新增“资源与性能”分区：

```text
资源与性能
- GPU vendor
- 后端
- 显存使用率
- 最大上下文长度
- Tensor Parallel / TP Size
- dtype
- 最大并发/序列数
- CPU limit
- memory limit
- shm-size
- ulimits
```

不同 backend 只显示适用字段。

### 7.3 模型部署页

步骤式：

```text
1. 选择模型
2. 选择后端/运行配置
3. 选择资源与调度
4. 资源策略/运行参数覆盖
5. Placement candidate / RunPlan 预览
6. 启动
```

必须显示：

```text
副本数：1（当前版本固定）
目标节点
模型位置
NBR
accelerator_ids
GPU vendor
资源与性能参数
RunPlan 预览
```

### 7.4 模型实例页

必须展示：

```text
所属 Deployment
所在 Node
accelerator_ids
GPU vendor
runtime image
endpoint
RunPlan
日志
测试
诊断
```

### 7.5 去掉“诊断与测试”独立菜单

当前阶段不保留重复菜单。

测试与诊断入口放在：

```text
模型实例详情
模型部署详情
```

未来独立诊断台另行设计。

## 8. 实现规划

### Phase 0：文档确认

产出：

```text
09-resource-scheduling-and-runtime-policy-design.md
10-manual-verification-issues-and-fix-plan.md
11-product-capability-implementation-plan.md
```

不改代码。

### Phase 1：现有问题 P0 修复

```text
去掉“诊断与测试”独立菜单
部署页模型名显示修复
Qwen3 Chat 404 诊断增强
```

### Phase 2：模型能力持久化

```text
设计/实现 capability 持久化
模型编辑页
测试入口使用持久化能力
```

### Phase 3：资源与性能参数入口

```text
NBR 资源与性能分区
vLLM gpu_memory_utilization
SGLang/llama.cpp 参数映射
RunPlan/equivalent docker command 验证
```

### Phase 4：Deployment resource policy / placement 表达

```text
部署页显示副本数=1
placement candidate
资源策略
RunPlan per-candidate preview
```

### Phase 5：测试与回归

```text
Go tests
frontend tests
build
server smoke
Qwen3 endpoint diagnosis
```

### Phase 6：Playwright

产品主线稳定后再实施。

## 9. 验收标准

### 文档验收

```text
现有问题全部落文档
资源限制与调度设计完整
多节点/多实例/调度扩展边界清楚
第一阶段范围清楚
实施步骤和验收标准清楚
```

### 功能验收

```text
模型能力可编辑并持久化
部署页显示模型名而不是 UUID
NBR 页面可配置显存使用率等资源参数
RunPlan 展示资源参数
去掉重复“诊断与测试”菜单
Qwen3 404 有明确诊断信息
```

### 技术验收

```bash
gofmt -w cmd/ internal/
go test ./internal/server/api/...
go test ./internal/server/runplan/...
go vet ./...
npm --prefix web test
npm --prefix web run build
git diff --check
git status --short
```

## 10. 不做事项

第一阶段不做：

```text
完整多副本调度
跨节点自动调度
quota
优先级调度
亲和/反亲和
自动 failover
完整 Playwright UI E2E
API Gateway / API Key
```

但设计必须预留这些方向。
