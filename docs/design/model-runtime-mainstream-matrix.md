# Model Runtime Mainstream Matrix

> Covers 14 model types × 4 backends with verified/blocked/unverified status.
> Current as of Batch 4 (Phase D closeout).

## Model Type × Backend Compatibility Matrix

| Model Type | Format | Task | vLLM | SGLang | llama.cpp | Ollama |
|-----------|--------|------|------|--------|-----------|--------|
| HF Chat | huggingface | chat | ✅ PASS | ✅ PASS | ❌ format | ❌ format |
| HF Completion | huggingface | completion | ✅ PASS | ✅ PASS | ❌ format | ❌ format |
| SentenceTransformers Embedding | sentence_transformers | embedding | ✅ PASS | ✅ PASS | ❌ format+task | ❌ format |
| CrossEncoder Reranker | huggingface | rerank | ✅ PASS | ✅ PASS | ❌ format+task | ❌ format |
| VLM (non-InternVL) | huggingface | vision_chat | ⚠️ UNVERIFIED | ⚠️ UNVERIFIED | ❌ format | ❌ format |
| VLM (InternVL2.5) | huggingface | vision_chat | ❌ BLOCKED | ❌ BLOCKED | ❌ format | ❌ format |
| GGUF Chat | gguf | chat | ❌ format | ❌ format | ✅ PASS | ❌ format |
| GGUF Completion | gguf | completion | ❌ format | ❌ format | ✅ PASS | ❌ format |
| LoRA Adapter | lora_adapter | adapter | ❌ standalone | ❌ standalone | ❌ standalone | ❌ standalone |
| Ollama Model | ollama | chat | ❌ format | ❌ format | ❌ format | ⚠️ CONFIGURED |
| ONNX | onnx | unknown | ⛔ NON_DEPLOYABLE | ⛔ | ⛔ | ⛔ |
| TensorRT Engine | tensorrt_engine | unknown | ⛔ NON_DEPLOYABLE | ⛔ | ⛔ | ⛔ |
| OpenVINO | openvino | unknown | ⛔ NON_DEPLOYABLE | ⛔ | ⛔ | ⛔ |
| Diffusers | diffusers | image_generation | ⛔ NON_DEPLOYABLE | ⛔ | ⛔ | ⛔ |
| ASR | huggingface | asr | ⛔ NON_DEPLOYABLE | ⛔ | ⛔ | ⛔ |
| TTS | huggingface | tts | ⛔ NON_DEPLOYABLE | ⛔ | ⛔ | ⛔ |
| Classification | huggingface | classification | ⛔ NON_DEPLOYABLE | ⛔ | ⛔ | ⛔ |

**Legend:**
- ✅ PASS: Production-verified (E2E test evidence)
- ❌ format/task/standalone: Blocked by format/task/path_mode mismatch
- ❌ BLOCKED: Architecture explicitly blocked (e.g., InternVL tokenizer incompatibility)
- ⚠️ UNVERIFIED: Backend declares support, not yet E2E verified in LightAI Go
- ⚠️ CONFIGURED: Structured capabilities exist, pending E2E verification
- ⛔ NON_DEPLOYABLE: Recognized by scanner, `deployable=false`, clear `unsupported_reason`

## Backend Version Capability Summary

### vLLM v0.23.0
```json
{
  "supported_formats": ["huggingface", "sentence_transformers"],
  "supported_tasks": ["chat", "completion", "embedding", "rerank", "vision_chat"],
  "model_path_modes": ["directory"],
  "serving_protocols": ["openai-compatible"],
  "test_endpoints": {
    "chat": "/v1/chat/completions", "completion": "/v1/completions",
    "embedding": "/v1/embeddings", "rerank": "/v1/rerank"
  },
  "blocked_architectures": {
    "InternVLChatModel": "vLLM runtime 无法加载 InternVL2.5 tokenizer"
  }
}
```

### SGLang v0.5.13.post1 / 0.4.6-compatible
```json
{
  "supported_formats": ["huggingface", "sentence_transformers"],
  "supported_tasks": ["chat", "completion", "embedding", "rerank", "vision_chat"],
  "model_path_modes": ["directory"],
  "serving_protocols": ["openai-compatible"],
  "test_endpoints": {
    "chat": "/v1/chat/completions", "completion": "/v1/completions",
    "embedding": "/v1/embeddings", "rerank": "/rerank"
  },
  "blocked_architectures": {
    "InternVLChatModel": "SGLang runtime 未验证 InternVL2.5 兼容性"
  }
}
```

### llama.cpp b9700
```json
{
  "supported_formats": ["gguf"],
  "supported_tasks": ["chat", "completion"],
  "model_path_modes": ["file"],
  "serving_protocols": ["openai-compatible"],
  "test_endpoints": {
    "chat": "/v1/chat/completions", "completion": "/v1/completions"
  }
}
```

### Ollama latest
```json
{
  "supported_formats": ["ollama"],
  "supported_tasks": ["chat", "completion"],
  "model_path_modes": ["ollama_managed"],
  "serving_protocols": ["ollama"],
  "test_endpoints": {
    "chat": "/api/chat", "completion": "/api/generate"
  }
}
```

## Path Mode Reference

| Path Mode | Used By | Meaning |
|-----------|---------|---------|
| `directory` | vLLM, SGLang | Model is a directory on the filesystem (HF format) |
| `file` | llama.cpp | Single GGUF file on the filesystem |
| `ollama_managed` | Ollama | Model referenced by name/tag, managed by Ollama |

## Unsupported Model Types (Recognized, Non-Deployable)

| Type | Detection | Unsupported Reason |
|------|-----------|-------------------|
| ONNX | `*.onnx` files | 平台尚未配置 ONNX Runtime 后端 |
| TensorRT | `*.engine` files | 平台尚未配置 TensorRT-LLM 后端 |
| OpenVINO | `*.xml` + `*.bin` files | 平台尚未配置 OpenVINO 后端 |
| Diffusers | `model_index.json` or `unet/` | 平台尚未配置 Diffusers/Image Generation 后端 |
| ASR | Name: whisper, funasr, paraformer, sensevoice | 平台尚未配置 ASR 后端 |
| TTS | Name: cosyvoice, chattts, gpt-sovits, etc. | 平台尚未配置 TTS 后端 |
| Classification | Architecture: *Classification | 平台尚未配置分类模型服务后端 |
| LoRA Adapter | `adapter_config.json` or `adapter_model.safetensors` | 需要选择基础模型后使用，不能独立部署 |
