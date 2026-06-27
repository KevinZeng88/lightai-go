# RuntimeRequirements and BackendCapabilityProfile Design

## 1. 设计目标

RuntimeRequirements 与 BackendCapabilityProfile 是 Runtime 架构中的两个核心契约。

它们需要支撑：

1. 内置 catalog；
2. BackendRuntime 创建；
3. NodeBackendRuntime enable；
4. check-request；
5. Preflight；
6. RunPlan；
7. UI 参数渲染；
8. API-first E2E；
9. 后续新增后端。

## 2. 职责区分

### 2.1 BackendCapabilityProfile

描述后端“能做什么”。

示例：

1. 支持 HuggingFace 模型；
2. 支持 GGUF 模型；
3. 支持 OpenAI compatible API；
4. 支持 `/v1/models`；
5. 支持 `/v1/chat/completions`；
6. 支持 GPU memory fraction 参数；
7. 支持 context length 参数；
8. 支持 CUDA_VISIBLE_DEVICES；
9. 支持 Docker health check；
10. 支持 tensor parallel。

### 2.2 RuntimeRequirements

描述后端“要运行起来需要什么”。

示例：

1. 需要 Docker；
2. 需要 image 可 inspect；
3. 需要模型路径存在；
4. 需要模型格式匹配；
5. 需要端口可用；
6. 需要 GPU 设备可用；
7. 需要 device binding 可构造；
8. 需要 mount 可读；
9. 需要 health check endpoint；
10. 需要参数合法。

## 3. BackendCapabilityProfile 推荐结构

```json
{
  "schema_version": "backend-capability-profile/v1",
  "backend": "vllm",
  "protocols": {
    "openai_compatible": {
      "enabled": true,
      "models_path": "/v1/models",
      "chat_completions_path": "/v1/chat/completions",
      "embeddings_path": "/v1/embeddings"
    }
  },
  "model_formats": ["huggingface"],
  "model_tasks": ["generation", "embedding"],
  "parameter_groups": ["server", "model", "resource_controls", "advanced"],
  "resource_controls": {
    "gpu_memory_utilization": {
      "supported": true,
      "arg_name": "--gpu-memory-utilization",
      "type": "number",
      "min": 0.1,
      "max": 1.0
    },
    "max_model_len": {
      "supported": true,
      "arg_name": "--max-model-len",
      "type": "integer"
    }
  },
  "device_binding_modes": ["env_cuda_visible_devices", "docker_gpus"],
  "health_check": {
    "default_path": "/v1/models",
    "success_status": [200]
  }
}
```

## 4. RuntimeRequirements 推荐结构

```json
{
  "schema_version": "runtime-requirements/v1",
  "container": {
    "image_required": true,
    "image_inspect_required": true,
    "docker_required": true
  },
  "model": {
    "path_required": true,
    "path_must_exist": true,
    "supported_formats": ["huggingface"],
    "required_files_any": ["config.json", "*.gguf"]
  },
  "network": {
    "container_port_required": true,
    "host_port_available_required": true
  },
  "accelerator": {
    "required": false,
    "supported_vendors": ["nvidia", "metax", "huawei"],
    "device_binding_required": true
  },
  "mounts": {
    "model_mount_required": true,
    "read_only_supported": true
  },
  "health_check": {
    "required": true,
    "path": "/v1/models",
    "timeout_seconds": 120
  },
  "warnings": {
    "version_probe_failed": "ready_with_warnings"
  }
}
```

## 5. Preflight 映射

Preflight 必须使用二者：

```text
BackendCapabilityProfile
        ↓
判断后端能力是否支持当前模型/参数/endpoint

RuntimeRequirements
        ↓
判断当前节点、镜像、路径、设备、端口是否满足运行条件
```

Preflight 输出：

```json
{
  "status": "ok",
  "errors": [],
  "warnings": [],
  "evidence": {
    "image": {},
    "model_path": {},
    "ports": {},
    "devices": {},
    "parameters": {}
  }
}
```

## 6. RunPlan 映射

RunPlan 使用：

1. BackendCapabilityProfile 决定可用参数、endpoint、health check；
2. RuntimeRequirements 决定必要检查、mount、device、port；
3. ParameterSchema 决定 args/env/mounts/ports 绑定；
4. ModelLocation 决定模型路径；
5. Deployment 决定覆盖项；
6. NodeBackendRuntime 决定节点级运行环境。

## 7. vLLM 示例

### Capability

```json
{
  "backend": "vllm",
  "model_formats": ["huggingface"],
  "protocols": {
    "openai_compatible": {
      "models_path": "/v1/models",
      "chat_completions_path": "/v1/chat/completions"
    }
  },
  "resource_controls": {
    "gpu_memory_utilization": true,
    "max_model_len": true
  },
  "device_binding_modes": ["env_cuda_visible_devices", "docker_gpus"]
}
```

### Requirements

```json
{
  "container": {
    "image_required": true
  },
  "model": {
    "path_required": true,
    "supported_formats": ["huggingface"]
  },
  "network": {
    "container_port": 8000
  },
  "health_check": {
    "path": "/v1/models"
  }
}
```

## 8. SGLang 示例

### Capability

```json
{
  "backend": "sglang",
  "model_formats": ["huggingface"],
  "protocols": {
    "openai_compatible": {
      "models_path": "/v1/models",
      "chat_completions_path": "/v1/chat/completions"
    }
  },
  "resource_controls": {
    "mem_fraction_static": true,
    "context_length": true
  },
  "device_binding_modes": ["env_cuda_visible_devices", "docker_gpus"]
}
```

### Requirements

```json
{
  "container": {
    "image_required": true
  },
  "model": {
    "path_required": true,
    "supported_formats": ["huggingface"]
  },
  "network": {
    "container_port": 30000
  },
  "health_check": {
    "path": "/v1/models"
  }
}
```

## 9. llama.cpp 示例

### Capability

```json
{
  "backend": "llamacpp",
  "model_formats": ["gguf"],
  "protocols": {
    "openai_compatible": {
      "models_path": "/v1/models",
      "chat_completions_path": "/v1/chat/completions"
    }
  },
  "resource_controls": {
    "n_gpu_layers": true,
    "ctx_size": true
  },
  "device_binding_modes": ["env_cuda_visible_devices", "docker_gpus"]
}
```

### Requirements

```json
{
  "container": {
    "image_required": true
  },
  "model": {
    "path_required": true,
    "supported_formats": ["gguf"],
    "required_files_any": ["*.gguf"]
  },
  "network": {
    "container_port": 8000
  },
  "health_check": {
    "path": "/v1/models"
  }
}
```

## 10. Accelerator 抽象

### NVIDIA

推荐 RunPlan 表达：

```json
{
  "vendor": "nvidia",
  "env": {
    "CUDA_VISIBLE_DEVICES": "0"
  },
  "docker": {
    "gpus": "device=0"
  }
}
```

### MetaX

推荐 RunPlan 表达：

```json
{
  "vendor": "metax",
  "env": {
    "CUDA_VISIBLE_DEVICES": "0"
  },
  "devices": [
    "/dev/mxcd",
    "/dev/dri"
  ],
  "mode": "native_docker"
}
```

### Huawei

推荐作为扩展预留：

```json
{
  "vendor": "huawei",
  "env": {},
  "devices": [],
  "mode": "vendor_runtime"
}
```

## 11. 禁止事项

1. Backend / BackendVersion 写入固定 GPU vendor；
2. CapabilityProfile 写入某个节点的 check 结果；
3. RuntimeRequirements 写入某个本机模型路径；
4. Preflight 依赖前端声明的 image_present；
5. RunPlan 使用与 Preflight 不同的要求定义；
6. UI 用另一套参数 schema；
7. E2E 只判断 API 成功但不检查实际 Docker spec。
