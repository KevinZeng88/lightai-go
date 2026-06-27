# RuntimeRequirements and BackendCapabilityProfile Design

## 1. 目标

本文定义 RuntimeRequirements 与 BackendCapabilityProfile 的最终设计。二者需要与参数 owner、copy-on-create、Preflight、RunPlan、UI/API 形成闭环。

## 2. 职责区分

### 2.1 BackendCapabilityProfile

BackendCapabilityProfile 描述能力：

```text
这个后端/版本能支持什么。
```

包括：

1. 支持的模型格式；
2. 支持的服务协议；
3. OpenAI compatible endpoint；
4. 支持的参数能力；
5. 支持的资源控制；
6. 支持的健康检查方式；
7. 支持的设备绑定抽象；
8. 支持的 warning 场景。

### 2.2 RuntimeRequirements

RuntimeRequirements 描述要求：

```text
要让这个后端/版本在当前节点和当前部署下运行，需要满足什么。
```

包括：

1. image；
2. Docker runtime；
3. GPU / accelerator；
4. device binding；
5. model path；
6. required files；
7. mounts；
8. ports；
9. env；
10. health check；
11. required args；
12. resource controls；
13. blocking error 与 warning。

## 3. 与参数 owner 的关系

1. Backend / BackendVersion 可以拥有后端能力相关 ParameterDefinition。
2. BackendCapabilityProfile 可以声明支持哪些 parameter key 和 target。
3. RuntimeRequirements 可以声明某些参数在运行时是否 required。
4. BackendRuntime 可以给这些参数提供模板默认值或模板 override。
5. NodeBackendRuntime 可以给这些参数提供节点 override 或 runtime evidence。
6. Deployment 可以提供部署 override。
7. ResolvedRunPlan 合成最终值。

BackendCapabilityProfile 和 RuntimeRequirements 不复制下层 override，不保存本机模型路径，不保存节点运行事实。

## 4. 数据结构建议

### 4.1 CapabilityProfile 示例

```json
{
  "backend": "vllm",
  "version": "latest",
  "protocols": ["openai_compatible"],
  "endpoints": ["/v1/models", "/v1/chat/completions"],
  "model_formats": ["huggingface_transformers"],
  "parameter_capabilities": [
    "host",
    "port",
    "gpu_memory_utilization",
    "max_model_len",
    "dtype",
    "quantization",
    "tensor_parallel_size",
    "served_model_name"
  ],
  "resource_controls": ["gpu_memory_utilization", "max_model_len"],
  "health_checks": ["http_get"],
  "device_binding_modes": ["nvidia_visible_devices", "vendor_neutral_accelerator_ids"]
}
```

### 4.2 RuntimeRequirements 示例

```json
{
  "image": {"required": true, "source": "backend_runtime"},
  "model_path": {"required": true, "source": "model_location"},
  "ports": [{"container_port": 8000, "protocol": "tcp", "required": true}],
  "health_check": {"type": "http_get", "path": "/v1/models", "port_ref": "service_port"},
  "accelerator": {"required": true, "vendor_neutral": true},
  "blocking_errors": ["missing_image", "missing_model_path", "invalid_parameters"],
  "warnings": ["version_probe_unavailable"]
}
```

## 5. vLLM 要求

CapabilityProfile：

1. HuggingFace Transformers model format；
2. OpenAI compatible API；
3. `/v1/models`；
4. `/v1/chat/completions`；
5. resource controls：`--gpu-memory-utilization`、`--max-model-len`；
6. dtype / quantization / tensor parallel；
7. health check：HTTP GET `/v1/models` 或 `/health`，以实际镜像能力为准。

RuntimeRequirements：

1. model path required；
2. image required；
3. service port required；
4. accelerator required for GPU profile；
5. model path mount required；
6. args 渲染由 RunPlan 完成。

## 6. SGLang 要求

CapabilityProfile：

1. HuggingFace model path；
2. OpenAI compatible API；
3. `--mem-fraction-static`；
4. `--context-length`；
5. dtype / tensor parallel；
6. health check HTTP endpoint。

RuntimeRequirements：

1. model path required；
2. service port required；
3. accelerator requirement by selected runtime profile；
4. image inspect evidence。

## 7. llama.cpp 要求

CapabilityProfile：

1. GGUF model format；
2. OpenAI compatible server mode；
3. `--ctx-size`；
4. `--n-gpu-layers` / `-ngl`；
5. batch / ubatch；
6. HTTP health check。

RuntimeRequirements：

1. GGUF file path required；
2. file mount required；
3. service port required；
4. accelerator optional depending profile；
5. image inspect evidence。

## 8. Accelerator 抽象

BackendCapabilityProfile 可以声明支持设备绑定抽象，但不写具体硬件事实。

支持方向：

1. NVIDIA：`NVIDIA_VISIBLE_DEVICES` / Docker GPU device request / CUDA_VISIBLE_DEVICES；
2. MetaX：vendor-neutral AcceleratorIds，设备文件绑定由 NodeBackendRuntime / DeviceBinding 解析；
3. Huawei：保留抽象，按 NodeBackendRuntime / Agent 能力扩展。

Backend / BackendVersion 不保存 GPU vendor 设备文件。

## 9. Preflight 映射

Preflight 输入：

1. BackendCapabilityProfile；
2. RuntimeRequirements；
3. BackendRuntime snapshot；
4. NodeBackendRuntime evidence；
5. ModelArtifact / ModelLocation；
6. Deployment override。

Preflight 输出：

1. errors；
2. warnings；
3. evidence；
4. normalized requirements result；
5. whether deployable。

## 10. RunPlan 映射

RunPlan resolver 使用 CapabilityProfile 和 RuntimeRequirements：

1. 选择参数渲染规则；
2. 验证 required 参数；
3. 生成 args/env/mounts/ports/devices；
4. 生成 health check；
5. 生成 parameter_source_map；
6. 输出 warnings/errors。

## 11. UI 映射

UI 使用二者：

1. 展示后端能力；
2. 展示运行要求；
3. 标识 blocking error；
4. 标识 warning；
5. 分类展示参数；
6. 标识 required/default/inherited/override；
7. 展示 check-request evidence；
8. 展示 RunPlan preview。

## 12. 验收

必须证明：

1. RuntimeRequirements 可驱动 Preflight；
2. BackendCapabilityProfile 可驱动参数渲染和 UI 提示；
3. 二者不保存本机模型路径；
4. 二者不保存部署实例状态；
5. vLLM/SGLang/llama.cpp 至少有可执行参数映射；
6. NVIDIA/MetaX 抽象边界清晰。
