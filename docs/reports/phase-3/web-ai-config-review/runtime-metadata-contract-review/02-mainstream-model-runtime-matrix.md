# Mainstream Model and Runtime Matrix

## 1. 目的

本文用于让 Claude / Codex 在开发前理解：不同模型类型和运行环境需要通过统一契约表达，而不是在 detector、preflight、RunPlan、TestDispatcher 中写特殊逻辑。

本文不是要求当前全部实现 runtime 支持。它定义覆盖边界、当前状态和未来扩展点。

## 2. 当前 LightAI Go 已验证能力

```text
HF Chat + vLLM: PASS
HF Chat + SGLang: PASS
GGUF Chat + llama.cpp: PASS
Embedding + vLLM: PASS
Reranker + vLLM: PASS
Wrong combinations blocked: PASS
Unsupported assets recognized: PASS_NON_DEPLOYABLE
VLM / InternVL2_5-1B: BACKEND_CAPABILITY_BLOCKED
```

## 3. 模型类型矩阵

| Model Type | Format | Task | Capabilities | Path Mode | Modalities | Serving Protocol | Runtime Features | Current Status |
|---|---|---|---|---|---|---|---|---|
| HF CausalLM Chat | huggingface | chat/completion | chat, completion | directory | text -> text | openai_chat_completions, openai_completions | directory_model_path, openai_chat_completions | production verified with vLLM/SGLang |
| GGUF Chat | gguf | chat/completion | chat, completion | file | text -> text | openai_chat_completions, openai_completions | file_model_path, openai_chat_completions | production verified with llama.cpp |
| SentenceTransformers Embedding | sentence_transformers | embedding | embedding | directory | text -> embedding | openai_embeddings | directory_model_path, openai_embeddings | production verified with vLLM |
| CrossEncoder Reranker | huggingface | rerank | rerank | directory | text pair -> score | rerank | directory_model_path, rerank | production verified with vLLM |
| VLM / Vision-Language | huggingface | vision_chat | chat, vision | directory | text+image -> text | openai_multimodal_chat | directory_model_path, openai_chat_completions, vision, image_processor, multimodal_processor | recognized; InternVLChatModel blocked |
| LoRA / PEFT Adapter | lora_adapter | adapter | adapter | adapter | depends on base | composition-dependent | adapter_model_path, adapter_composition | recognized non-standalone |
| ONNX | onnx | unknown/classification/etc | depends | bundle/file | depends | custom/onnx | bundle_model_path | recognized non-deployable |
| TensorRT Engine | tensorrt_engine | unknown/chat/etc | depends | bundle | depends | custom/tensorrt | bundle_model_path, gpu_required | recognized non-deployable |
| OpenVINO | openvino | unknown/classification/etc | depends | bundle | depends | custom/openvino | bundle_model_path, cpu_supported | recognized non-deployable |
| Diffusers | diffusers | image_generation | image_generation | directory | text/image -> image | diffusers_pipeline | directory_model_path | recognized non-deployable |
| ASR | huggingface | asr | asr | directory | audio -> text | asr_transcription | directory_model_path, audio_processor | recognized non-deployable |
| TTS | huggingface | tts | tts | directory | text -> audio | tts_synthesis | directory_model_path, audio_processor | recognized non-deployable |
| Classification | huggingface | classification | classification | directory | text/image/audio -> classification | custom/openai-compatible optional | directory_model_path | recognized non-deployable |
| Long-context models | huggingface | chat/completion | chat, completion | directory | text -> text | openai_chat_completions | max_model_len/context_length/tensor_parallel | not separate type; runtime requirement modifier |
| Quantized AWQ/GPTQ/FP8 | huggingface | chat/etc | depends | directory | depends | depends | quantization/dtype support | modifier; requires backend-specific support |
| MoE | huggingface | chat/completion | chat, completion | directory | text -> text | openai_chat_completions | tensor_parallel/batching/gpu_required | modifier; requires capacity/runtime validation |

## 4. 每类模型的契约重点

### 4.1 HF CausalLM Chat

Profile 重点：

```text
required_files: config.json
required_file_groups: one of model.safetensors / *.safetensors / pytorch_model.bin
extract: model_type, architectures, tokenizer_class, auto_map
```

RuntimeRequirements：

```text
path.model_path_mode: directory
serving.protocols: openai_chat_completions, openai_completions
serving.default_test_mode: chat
runtime_features.required: directory_model_path, openai_chat_completions
runtime_arg_requirements: model_path, optional trust_remote_code if auto_map requires it
```

### 4.2 GGUF

Profile 重点：

```text
required_file_groups: *.gguf
path_mode: file
```

RuntimeRequirements：

```text
path.model_path_mode: file
runtime_features.required: file_model_path, openai_chat_completions
runtime_arg_requirements.abstract_arg: model_path
```

RunPlan 要求：

```text
llama.cpp 的 -m 必须指向具体 .gguf 文件，不是目录。
```

### 4.3 Embedding

Profile 重点：

```text
SentenceTransformers: modules.json, config_sentence_transformers.json, Pooling module
HF embedding: architecture/model_type/name hint may also classify
exclude reranker/cross-encoder patterns
```

RuntimeRequirements：

```text
serving.protocols: openai_embeddings
serving.default_test_mode: embedding
serving.required_test_endpoints: embedding
runtime_features.required: directory_model_path, openai_embeddings
```

### 4.4 Reranker

Profile 重点：

```text
name_patterns: reranker, rerank, cross-encoder
architectures: SequenceClassification-style architectures
```

RuntimeRequirements：

```text
serving.protocols: rerank
serving.default_test_mode: rerank
serving.required_test_endpoints: rerank
runtime_features.required: directory_model_path, rerank
```

### 4.5 VLM

Profile 重点：

```text
architecture / auto_map / image_processor / preprocessor / vision config
must preserve chat + vision capabilities
```

RuntimeRequirements：

```text
modalities.input: text, image
serving.protocols: openai_multimodal_chat
serving.required_test_endpoints: chat
runtime_features.required: directory_model_path, openai_chat_completions, vision, image_processor, multimodal_processor
runtime_options.requires_multimodal_processor: true
runtime_arg_requirements: trust_remote_code when required
```

Compatibility 注意：

```text
不能只因为 backend 声明 vision_chat 就放行；必须看 architecture/model_family allowlist/blocklist 或 verified status。
```

### 4.6 LoRA / Adapter

Profile 重点：

```text
adapter_config.json
peft_type
base_model_name_or_path
```

RuntimeRequirements：

```text
path.model_path_mode: adapter
runtime_options.requires_base_model: true
runtime_options.requires_adapter_composition: true
runtime_features.required: adapter_model_path, adapter_composition
```

Compatibility 注意：

```text
standalone deploy 必须阻断，除非请求中包含 base model composition。
```

### 4.7 Unsupported recognized types

ONNX、TensorRT、OpenVINO、Diffusers、ASR、TTS、Classification 当前重点不是运行，而是：

```text
1. 能识别；
2. deployable=false；
3. unsupported_reason 清晰；
4. preflight 阻断；
5. 不伪装成 chat/embedding/rerank。
```

## 5. 运行环境矩阵

| Runtime | Typical Formats | Typical Tasks | Path Modes | Serving Protocols | Abstract Args | Current LightAI Go Status |
|---|---|---|---|---|---|---|
| vLLM | huggingface, sentence_transformers | chat, completion, embedding, rerank, some VLM | directory | OpenAI chat/completions/embeddings, rerank | model_path, trust_remote_code, max_model_len, tensor_parallel_size, dtype, gpu_memory_utilization | production verified for chat/embedding/rerank; InternVL blocked |
| SGLang | huggingface | chat, completion, some VLM | directory | OpenAI-compatible chat/completions | model_path, trust_remote_code, context_length, tensor_parallel_size | production verified for chat; VLM not verified |
| llama.cpp | gguf | chat, completion, some multimodal GGUF | file, sometimes file+projector | OpenAI-compatible chat/completions | model_path, mm_projector, context_length, host, port | production verified for GGUF chat |
| LMDeploy | huggingface, some VLM | chat, VLM | directory | OpenAI-compatible | model_path, trust_remote_code, tensor_parallel_size | not configured |
| HF TGI | huggingface | text generation | directory | TGI/OpenAI-compatible variants | model_path, max_input_length, max_total_tokens | not configured |
| Transformers local server | huggingface | many | directory | custom/OpenAI-compatible depending implementation | model_path, trust_remote_code | not configured |
| TensorRT-LLM | tensorrt_engine | chat/completion depending engine | bundle | custom/OpenAI-compatible server possible | engine_dir, tokenizer_dir, tensor_parallel_size | recognized non-deployable |
| OpenVINO Model Server | openvino | classification/embedding/etc | bundle | REST/gRPC custom | model_dir, device | recognized non-deployable |
| ONNX Runtime | onnx | classification/embedding/etc | file/bundle | custom | model_path | recognized non-deployable |
| Triton Inference Server | onnx/tensorrt/openvino/custom | many | repository/bundle | HTTP/gRPC | model_repository | not configured |
| Diffusers server | diffusers | image_generation | directory | custom/diffusers pipeline | model_path, dtype, device | recognized non-deployable |
| ASR server | huggingface/custom | asr | directory | asr_transcription | model_path, language, sample_rate | recognized non-deployable |
| TTS server | huggingface/custom | tts | directory | tts_synthesis | model_path, speaker, sample_rate | recognized non-deployable |

## 6. 当前不应声明的能力

```text
1. 不声明所有 VLM 都能跑；
2. 不声明 InternVL2_5-1B runtime 已通过；
3. 不声明 ONNX/TensorRT/OpenVINO/Diffusers/ASR/TTS/Classification serving 已支持；
4. 不声明 LoRA 可以 standalone deploy；
5. 不声明 BackendVersion 的 GPU vendor/hardware 能力。
```

## 7. 设计约束

```text
1. BackendVersion 描述 backend/version 通用能力；
2. BackendRuntime 描述运行模板、image、args/env/ports/health/arg mapping；
3. NodeBackendRuntime 描述节点实例化能力、设备、vendor-specific flags；
4. ResolvedBackendCapability 是实际用于 preflight/RunPlan/TestDispatcher 的合并结果；
5. 未验证能力不要写成 supported，应写 not_configured / recognized_non_deployable / blocked / unverified。
```
