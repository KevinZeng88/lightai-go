# Model Runtime Contract and Backend Capability Design

## 1. 设计目标

LightAI Go 后续会持续面对多种模型形态和多种运行环境：Chat、GGUF、Embedding、Reranker、VLM、LoRA、ONNX、TensorRT、OpenVINO、Diffusers、ASR、TTS、Classification 等。

当前不能继续依赖零散字段或特殊 if 判断。需要建立统一、可验证、可被代码消费的契约：

```text
模型类型规则：ModelTypeProfile
扫描实例结果：DiscoveredMetadata
模型运行需求：RuntimeRequirements
后端能力声明：BackendCapabilityProfile
节点/运行时合并能力：ResolvedBackendCapability
真实验证证据：VerificationRecord / EvidenceIndex
```

核心匹配关系：

```text
RuntimeRequirements × ResolvedBackendCapability => CompatibilityResult
```

## 2. 分层原则

### 2.1 ModelTypeProfile 是类型规则，不是具体模型

ModelTypeProfile 定义“某一类模型如何识别”。它不应包含：

```text
1. /home/kzeng/models/... 这类本机绝对路径；
2. container id；
3. docker inspect 结果；
4. production E2E evidence 路径；
5. 某一次扫描时间；
6. 某个节点 id。
```

它应该包含：

```text
1. 需要检查哪些文件；
2. 需要读取哪些 json 配置；
3. 需要匹配哪些 config key/value；
4. 需要排除哪些模式；
5. 检测成功后默认产出什么 format/task/capability；
6. 默认 RuntimeRequirements。
```

示例：SentenceTransformers Embedding 类型规则。

```json
{
  "schema_version": 1,
  "profile_id": "embedding.sentence_transformers",
  "profile_version": 1,
  "priority": 40,
  "match": {
    "kind": "directory",
    "path_mode": "directory",
    "required_files": ["modules.json", "config_sentence_transformers.json"],
    "optional_files": ["tokenizer_config.json", "1_Pooling/config.json"],
    "file_globs": ["*.safetensors", "pytorch_model.bin"],
    "name_patterns": ["bge", "e5", "gte", "embedding", "sentence-transformers", "text2vec"],
    "exclude_name_patterns": ["reranker", "rerank", "cross-encoder"],
    "config_checks": [
      {
        "file": "modules.json",
        "json_path": "$[*].type",
        "op": "contains",
        "value": "Pooling"
      }
    ]
  },
  "extract": {
    "model_type": [{"file": "config.json", "json_path": "$.model_type"}],
    "architectures": [{"file": "config.json", "json_path": "$.architectures"}],
    "tokenizer_class": [{"file": "tokenizer_config.json", "json_path": "$.tokenizer_class"}],
    "auto_map": [{"file": "config.json", "json_path": "$.auto_map"}]
  },
  "detected_defaults": {
    "format": "sentence_transformers",
    "task": "embedding",
    "capabilities": ["embedding"],
    "default_test_mode": "embedding",
    "deployable": true,
    "requires_base_model": false,
    "recommended_backends": ["vllm", "sglang"],
    "unsupported_reason": ""
  },
  "runtime_requirements": {
    "schema_version": 1,
    "path": {
      "model_path_mode": "directory",
      "required_files": ["config.json", "modules.json"],
      "optional_files": ["tokenizer_config.json"],
      "required_file_groups": []
    },
    "modalities": {
      "input": ["text"],
      "output": ["embedding"]
    },
    "serving": {
      "protocols": ["openai_embeddings"],
      "default_test_mode": "embedding",
      "required_test_endpoints": ["embedding"]
    },
    "runtime_features": {
      "required": ["directory_model_path", "openai_embeddings"],
      "optional": [],
      "forbidden": []
    },
    "runtime_options": {
      "requires_trust_remote_code": false,
      "requires_multimodal_processor": false,
      "requires_base_model": false,
      "requires_adapter_composition": false
    },
    "runtime_arg_requirements": [],
    "environment": {
      "required_python_packages": [],
      "required_env": [],
      "required_devices": [],
      "recommended_shm_size": ""
    }
  }
}
```

### 2.2 DiscoveredMetadata 是扫描结果，不是类型规则

DiscoveredMetadata 记录某个 ModelLocation 按某个 ModelTypeProfile 扫描后的结果。实际绝对路径应保存在 `model_locations.path`，不要在 `discovered_metadata_json` 中重复保存。

它可以包含：

```text
1. profile_id / profile_version；
2. 是否匹配；
3. 置信度；
4. 匹配到的相对文件；
5. 缺失的可选文件；
6. 命中的配置检查；
7. 提取出的 config facts；
8. 最终 detected 结果；
9. resolved RuntimeRequirements。
```

示例：

```json
{
  "schema_version": 1,
  "profile_id": "embedding.sentence_transformers",
  "profile_version": 1,
  "matched": true,
  "confidence": "high",
  "scan_result": {
    "kind": "directory",
    "path_mode": "directory",
    "matched_files": ["modules.json", "config_sentence_transformers.json", "1_Pooling/config.json"],
    "missing_optional_files": ["sentence_bert_config.json"],
    "matched_name_patterns": ["bge"],
    "matched_config_checks": [
      {
        "file": "modules.json",
        "json_path": "$[*].type",
        "op": "contains",
        "value": "Pooling"
      }
    ]
  },
  "extracted_facts": {
    "model_type": "bert",
    "architectures": ["BertModel"],
    "tokenizer_class": null,
    "auto_map": {},
    "requires_remote_code": false,
    "quantization": null
  },
  "detected": {
    "format": "sentence_transformers",
    "task": "embedding",
    "capabilities": ["embedding"],
    "default_test_mode": "embedding",
    "deployable": true,
    "requires_base_model": false,
    "recommended_backends": ["vllm", "sglang"],
    "architecture": "BertModel",
    "model_family": "bge",
    "unsupported_reason": ""
  },
  "runtime_requirements": {}
}
```

### 2.3 RuntimeRequirements 是模型侧可执行契约

RuntimeRequirements 表达“模型运行需要什么”。它不是说明性文本，必须能被 scanner、CompatibilityChecker、RunPlan resolver、TestDispatcher 消费。

标准结构：

```json
{
  "schema_version": 1,
  "path": {
    "model_path_mode": "directory",
    "required_files": ["config.json"],
    "optional_files": ["tokenizer_config.json"],
    "required_file_groups": [
      {
        "any_of": ["model.safetensors", "pytorch_model.bin", "*.safetensors"],
        "all_of": [],
        "reason": "At least one model weight file is required."
      }
    ]
  },
  "modalities": {
    "input": ["text"],
    "output": ["text"]
  },
  "serving": {
    "protocols": ["openai_chat_completions"],
    "default_test_mode": "chat",
    "required_test_endpoints": ["chat"]
  },
  "runtime_features": {
    "required": ["directory_model_path", "openai_chat_completions"],
    "optional": [],
    "forbidden": []
  },
  "runtime_options": {
    "requires_trust_remote_code": false,
    "requires_multimodal_processor": false,
    "requires_base_model": false,
    "requires_adapter_composition": false
  },
  "runtime_arg_requirements": [
    {
      "feature": "trust_remote_code",
      "required_when": "requires_trust_remote_code == true",
      "abstract_arg": "trust_remote_code",
      "reason": "Remote-code model requires backend-specific trust-remote-code flag."
    }
  ],
  "environment": {
    "required_python_packages": [],
    "required_env": [],
    "required_devices": [],
    "recommended_shm_size": ""
  }
}
```

#### 字段语义

`path` 用于 scanner 与 RunPlan path 判断。

```text
model_path_mode: file | directory | bundle | adapter
required_files: 模型根目录内相对路径
optional_files: 模型根目录内相对路径
required_file_groups: any_of / all_of 文件组要求
```

`modalities` 用于 UI、测试 payload 与未来输入校验。

```text
input: text | image | audio | video
output: text | embedding | score | image | audio | classification
```

`serving` 用于 TestDispatcher。

```text
protocols: openai_chat_completions | openai_completions | openai_embeddings | openai_multimodal_chat | rerank | diffusers_pipeline | asr_transcription | tts_synthesis | custom
default_test_mode: auto | chat | completion | embedding | rerank
required_test_endpoints: chat | completion | embedding | rerank
```

`runtime_features` 用于 CompatibilityChecker。

```text
required: backend 必须声明支持
optional: backend 支持则可用，不支持不阻断
forbidden: backend 如声明该 feature，则阻断
```

`runtime_arg_requirements` 用于 RunPlan resolver。这里只能出现 abstract arg，不能出现 backend-specific CLI 参数。

正确：

```json
{"abstract_arg": "trust_remote_code"}
```

错误：

```json
{"arg": "--trust-remote-code"}
```

具体 CLI 参数应该由 BackendCapabilityProfile / BackendRuntime 的 arg mapping 翻译。

### 2.4 BackendCapabilityProfile 是后端侧可执行契约

BackendCapabilityProfile 表达“某个 backend version/runtime 能提供什么”。它必须和 RuntimeRequirements 可计算匹配。

标准结构：

```json
{
  "schema_version": 1,
  "backend": "vllm",
  "backend_version": "v0.20.1",
  "runtime_kind": "docker",
  "supported_formats": ["huggingface", "sentence_transformers"],
  "supported_tasks": ["chat", "completion", "embedding", "rerank"],
  "supported_capabilities": ["chat", "completion", "embedding", "rerank"],
  "model_path_modes": ["directory"],
  "modalities": {
    "input": ["text"],
    "output": ["text", "embedding", "score"]
  },
  "supported_serving_protocols": [
    "openai_chat_completions",
    "openai_completions",
    "openai_embeddings",
    "rerank"
  ],
  "supported_runtime_features": [
    "directory_model_path",
    "openai_chat_completions",
    "openai_completions",
    "openai_embeddings",
    "rerank"
  ],
  "test_endpoints": {
    "chat": "/v1/chat/completions",
    "completion": "/v1/completions",
    "embedding": "/v1/embeddings",
    "rerank": "/v1/rerank"
  },
  "arg_support": {
    "supported_abstract_args": [
      "model_path",
      "served_model_name",
      "trust_remote_code",
      "max_model_len",
      "tensor_parallel_size"
    ],
    "arg_mappings": {
      "model_path": {
        "cli_arg": "--model",
        "value_source": "model_location.path",
        "required": true
      },
      "trust_remote_code": {
        "cli_arg": "--trust-remote-code",
        "mode": "flag",
        "required": false
      }
    }
  },
  "supported_architectures": {
    "chat": ["Qwen3ForCausalLM"],
    "embedding": ["BertModel", "XLMRobertaModel"],
    "rerank": ["XLMRobertaForSequenceClassification"]
  },
  "blocked_architectures": {
    "InternVLChatModel": "当前 vLLM runtime 未通过 InternVL2.5 生产验证。"
  },
  "blocked_model_families": {},
  "blocked_tasks": {},
  "notes": []
}
```

BackendCapabilityProfile 不应包含：

```text
1. production E2E evidence path；
2. container id；
3. docker inspect 实例结果；
4. 某个本地模型路径；
5. GPU vendor/hardware 固定信息。
```

GPU vendor/hardware 相关内容应放在 BackendRuntime / NodeBackendRuntime / Node runtime 配置层。

### 2.5 RuntimeRequirements × BackendCapabilityProfile 匹配规则

CompatibilityChecker 至少应检查：

```text
1. deployable=false 优先阻断；
2. model format ∈ backend.supported_formats；
3. model task ∈ backend.supported_tasks；
4. model capabilities ⊆ backend.supported_capabilities；
5. req.path.model_path_mode ∈ backend.model_path_modes；
6. req.serving.protocols 与 backend.supported_serving_protocols 有交集；
7. req.serving.required_test_endpoints ⊆ keys(backend.test_endpoints)；
8. req.runtime_features.required ⊆ backend.supported_runtime_features；
9. req.runtime_features.forbidden ∩ backend.supported_runtime_features == empty；
10. req.runtime_arg_requirements.abstract_arg ⊆ backend.arg_support.supported_abstract_args；
11. architecture ∉ backend.blocked_architectures；
12. family ∉ backend.blocked_model_families；
13. task ∉ backend.blocked_tasks；
14. if backend.supported_architectures[task] 非空，architecture 必须命中 allowlist；
15. requires_base_model=true 不能独立部署，除非 composition 已明确满足；
16. requires_adapter_composition=true 要求 backend 支持 adapter_composition。
```

CompatibilityResult 应是结构化结果，不只是 bool：

```json
{
  "status": "unsupported_runtime_feature",
  "reason": "backend vllm does not support runtime feature image_processor",
  "field": "runtime_features.required",
  "missing": ["image_processor"]
}
```

### 2.6 ResolvedBackendCapability

当前系统已有 BackendVersion、BackendRuntime、NodeBackendRuntime 概念。需要明确边界：

```text
BackendVersion:
  backend/version 通用能力；不包含 GPU vendor/hardware。

BackendRuntime:
  docker image、默认 args/env/ports/health、arg mapping、runtime kind。

NodeBackendRuntime:
  某节点上的实际 runtime 配置；可以包含 image override、设备、vendor-specific flags、可用状态。

ResolvedBackendCapability:
  BackendVersion capability + BackendRuntime template + NodeBackendRuntime overlay 后的最终能力，供 preflight / RunPlan / TestDispatcher 使用。
```

如果当前代码没有完整 overlay 实现，先在文档和 Go type 里预留，不要写假实现。

### 2.7 VerificationRecord / EvidenceIndex

真实验证证据不属于 ModelTypeProfile，也不属于 BackendCapabilityProfile。

VerificationRecord 可以引用 evidence 文件：

```json
{
  "schema_version": 1,
  "verification_id": "batch4.e2e.embedding.vllm",
  "model_ref": {
    "model_family": "bge",
    "architecture": "BertModel",
    "task": "embedding"
  },
  "backend_ref": {
    "backend": "vllm",
    "backend_version": "v0.20.1"
  },
  "status": "pass",
  "evidence_files": [
    "docs/reports/phase-3/web-ai-config-review/evidence/batch4-full-flow-e2e/e2e-4-embedding-response.json"
  ]
}
```

当前可以只存在 closeout/evidence 文档中，暂不入 DB。

## 3. 统一枚举

### FormatID

```text
huggingface
sentence_transformers
gguf
lora_adapter
onnx
tensorrt_engine
openvino
diffusers
unknown
```

### TaskID

```text
chat
completion
embedding
rerank
vision_chat
adapter
image_generation
asr
tts
classification
unknown
```

### CapabilityID

```text
chat
completion
embedding
rerank
vision
image_generation
asr
tts
classification
adapter
```

### PathMode

```text
file
directory
bundle
adapter
```

### Modality

```text
text
image
audio
video
embedding
score
classification
```

### ServingProtocol

```text
openai_chat_completions
openai_completions
openai_embeddings
openai_multimodal_chat
rerank
diffusers_pipeline
asr_transcription
tts_synthesis
custom
```

### RuntimeFeature

```text
directory_model_path
file_model_path
bundle_model_path
adapter_model_path
openai_chat_completions
openai_completions
openai_embeddings
openai_multimodal_chat
rerank
vision
image_processor
multimodal_processor
audio_processor
adapter_composition
trust_remote_code
streaming
batching
tensor_parallel
gpu_required
cpu_supported
```

### AbstractArg

```text
model_path
model_dir
served_model_name
trust_remote_code
max_model_len
context_length
tensor_parallel_size
gpu_memory_utilization
dtype
quantization
chat_template
embedding_task
rerank_task
mm_projector
host
port
api_key
```

### CompatibilityStatus

```text
pass
blocked
non_deployable
unsupported_format
unsupported_task
unsupported_capability
unsupported_path_mode
unsupported_serving_protocol
unsupported_runtime_feature
forbidden_runtime_feature
missing_test_endpoint
missing_abstract_arg_support
unsupported_architecture
backend_capability_blocked
requires_base_model
requires_adapter_composition
external_dependency_blocked
unverified
```

## 4. 设计底线

```text
1. 类型定义不放本地路径；
2. 扫描结果只放相对 evidence 和提取事实；
3. 模型需求不放 backend-specific CLI；
4. 后端能力可以放 CLI 映射，但不能放生产证据；
5. 真实证据单独放 VerificationRecord / closeout；
6. CompatibilityChecker 做可计算匹配；
7. RunPlan 只消费匹配后的抽象参数映射；
8. TestDispatcher 不猜 endpoint，只看 backend profile；
9. 所有 enum 统一定义并校验；
10. hardcode 必须审查、迁移或 formal blocker。
```
