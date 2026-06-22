# LightAI Go 模型识别、能力、运行与测试抽象设计

建议上传路径：

```text
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/phase-3/web-ai-config-review/28-model-detection-runtime-plugin-design.md
```

适用任务：Web AI 模型扫描、模型属性、能力识别、运行兼容性、测试入口、RunPlan 前置校验。

---

## 1. 背景与问题

LightAI Go 当前已经具备模型文件/目录扫描、模型库、运行配置、RunPlan 预览和真实容器启动能力。近期已经修过几个关键问题：

```text
625ac16 fix(web-ai): distinguish HF directory from GGUF file in directory scan
ed2f145 fix(web-ai): resolve gguf file path in real runplans
6e9de80 fix(web-ai): clean runtime command generation
3fa3a6d feat(web-ai): persist model capabilities
```

当前已经能区分：

```text
HuggingFace directory → 目录型模型
GGUF file → 文件型模型
空目录 → 目录中没有发现模型文件
目录中多个 GGUF → 用户选择具体 GGUF 文件
```

但继续按“发现一种模型类型就补一段 if/else”的方式扩展，会带来几个问题：

1. 模型识别逻辑散落在 agent scanner、server proxy、frontend wizard、model detail、preflight、RunPlan resolver、test dialog 中。
2. HuggingFace directory 容易被错误默认成 Chat/Completion，而实际可能是 Embedding、Reranker、Vision-Language、Classifier、ASR 等。
3. “模型是什么”和“当前环境能不能跑”混在一起，导致没有运行后端的模型可能无法进入模型库，失去管理价值。
4. RunPlan 阶段才发现路径语义不对，例如 llama.cpp `-m` 指向目录而不是 `.gguf` 文件。
5. 测试入口无法根据模型能力自动选择 chat、completion、embedding、rerank 等测试方法。
6. 后续每新增一种模型格式或能力，都要修改多处业务代码，扩展成本高且容易引入回归。

因此需要建立一个清晰抽象：

```text
模型识别负责回答：这是什么模型？
运行兼容负责回答：当前有哪些 backend 能跑它？
测试方法负责回答：该怎么验证这个模型服务？
RunPlan 只负责在兼容性成立后生成启动命令。
```

---

## 2. 设计目标

本设计目标是建立一套可扩展的模型识别与运行兼容机制。

### 2.1 必须实现的目标

1. 模型扫描尽量识别常见模型类型，而不是只识别当前可运行的类型。
2. 模型具有明确的类型、格式、任务、能力、运行建议、测试方法和识别证据。
3. 当前没有运行环境的模型也可以进入模型库，但必须标记不可部署或缺少运行环境。
4. 运行前通过兼容性检查阻止错误组合。
5. 测试入口根据模型能力和 backend endpoint 自动选择测试方式。
6. 新增一种模型类型时，优先新增 detector/plugin，不应在前端、RunPlan、测试入口到处加分支。
7. GGUF 文件语义保持严格：最终可部署路径必须是具体 `.gguf` 文件。
8. HuggingFace 目录语义保持严格：如果选择 HF directory，则运行路径是目录；但还要进一步识别 task/capabilities。

### 2.2 明确不做的事项

本轮不做：

```text
1. 不进入 Phase 3 资源参数编辑器。
2. 不做多副本、跨节点调度或高可用。
3. 不新增 ONNX / TensorRT / OpenVINO / ASR / TTS / Diffusers backend。
4. 不做模型转换。
5. 不做 LoRA merge。
6. 不做图片/音频上传测试入口的完整实现。
7. 不为了旧错误数据保留兼容分支。
```

---

## 3. 核心分层

建议将模型管理拆成五层。

```text
ModelDetector
  ↓
ModelCandidate / ModelDescriptor
  ↓
ModelArtifact + ModelLocation 持久化
  ↓
CompatibilityChecker
  ↓
RunPlanResolver / TestMethodResolver
```

### 3.1 ModelDetector

负责扫描目录或文件，识别候选模型。

只回答：

```text
这个路径看起来像什么模型？
有什么证据？
置信度如何？
它通常有哪些能力？
它通常建议用哪些 backend？
```

Detector 不应该直接生成 RunPlan，也不应该直接决定最终能不能部署。

### 3.2 ModelCandidate / ModelDescriptor

表示扫描出来的候选模型。它是模型识别的标准输出。

### 3.3 ModelArtifact / ModelLocation

用户选择 candidate 后，系统把 candidate 的语义持久化到模型库。

原则：

```text
ModelArtifact 表达“模型是什么”。
ModelLocation 表达“这个模型在某节点上的哪个可部署路径”。
```

### 3.4 CompatibilityChecker

负责判断：

```text
某个模型 + 某个 BackendVersion / BackendRuntime 是否兼容？
如果不兼容，为什么？
如果兼容，支持哪些运行方法和测试方法？
```

### 3.5 RunPlanResolver / TestMethodResolver

只在兼容性成立后工作。

RunPlanResolver 负责生成启动命令。

TestMethodResolver 负责选择默认测试方法和请求 payload。

---

## 4. ModelCandidate 抽象

扫描 API 和内部 scanner 应输出统一结构，不要只返回 path/format。

建议结构：

```json
{
  "kind": "directory | file | adapter | bundle",
  "format": "huggingface | sentence_transformers | gguf | lora_adapter | onnx | tensorrt_engine | openvino | diffusers | whisper | funasr | unknown",
  "task": "chat | completion | embedding | rerank | vision_chat | adapter | classification | token_classification | asr | tts | image_generation | unknown",
  "capabilities": ["chat", "completion", "embedding", "rerank", "vision"],
  "default_test_mode": "chat | completion | embedding | rerank | vision | auto",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["vllm", "sglang", "llamacpp"],
  "path": "/path/to/model",
  "display_name": "model-name",
  "confidence": "high | medium | low",
  "evidence": [
    "config.json",
    "modules.json",
    "filename contains reranker"
  ],
  "unsupported_reason": ""
}
```

字段说明：

| 字段 | 含义 |
|---|---|
| `kind` | 存储形态，决定路径语义。目录、文件、adapter、bundle。 |
| `format` | 模型文件/目录格式，例如 HF、GGUF、SentenceTransformers、ONNX。 |
| `task` | 模型任务类型，例如 chat、embedding、rerank。 |
| `capabilities` | 模型能力，供测试入口、兼容性、UI 展示使用。 |
| `default_test_mode` | 默认测试方式。 |
| `deployable` | 模型本身是否可独立部署。注意这不是“当前环境能否运行”。 |
| `requires_base_model` | LoRA/Adapter 等是否需要基础模型。 |
| `recommended_backends` | 通常推荐的 backend 类型。 |
| `path` | 最终候选路径。HF 为目录，GGUF 为具体文件。 |
| `confidence` | 识别置信度。 |
| `evidence` | 识别证据，用于审计、UI 展示和排错。 |
| `unsupported_reason` | 当前不支持或不可部署的原因。 |

关键约束：

```text
format=gguf 时，path 必须是具体 .gguf 文件。
format=huggingface 时，path 可以是模型目录。
kind=adapter 时，不能作为独立基础模型部署。
```

---

## 5. Detector Registry 抽象

不应把识别逻辑写成到处散落的 if/else。建议实现 detector registry。

Go 侧可参考：

```go
type FileFacts struct {
    RootPath      string
    IsFile        bool
    IsDirectory   bool
    FileName      string
    DirEntries    []string
    ConfigJSON    map[string]any
    FileGlobs     map[string][]string
    EvidenceFiles []string
}

type ModelDetector interface {
    ID() string
    Detect(facts FileFacts) []ModelCandidate
}
```

如果项目当前风格不适合接口，也可以用函数表：

```go
var detectors = []DetectorFunc{
    DetectLoRAAdapter,
    DetectSentenceTransformers,
    DetectReranker,
    DetectVisionLanguage,
    DetectHuggingFaceChat,
    DetectGGUF,
    DetectONNX,
    DetectTensorRTEngine,
    DetectOpenVINO,
    DetectDiffusers,
    DetectASR,
    DetectTTS,
    DetectClassification,
}
```

要求：

1. 每个 detector 只负责一种或一组相近类型。
2. detector 输出 ModelCandidate，而不是直接操作 UI 或 RunPlan。
3. detector 必须提供 evidence 和 confidence。
4. detector 不直接决定当前 backend 是否可运行。
5. 新增模型类型时优先新增 detector，不修改多个页面和 resolver。

---

## 6. FileFacts：扫描事实层

为了避免每个 detector 重复访问文件系统，应先构建 FileFacts。

FileFacts 负责收集低成本事实：

```text
1. 当前路径是文件还是目录。
2. 文件名、扩展名、父目录名。
3. 当前目录直接子项。
4. 少量关键 JSON 文件解析结果：
   - config.json
   - generation_config.json
   - tokenizer_config.json
   - sentence_bert_config.json
   - modules.json
   - adapter_config.json
   - model_index.json
   - preprocessor_config.json
   - image_processor_config.json
5. 关键 glob：
   - *.gguf
   - *.onnx
   - *.engine
   - *.safetensors
   - pytorch_model.bin
   - *.xml + *.bin
```

扫描规则：

```text
默认只扫描当前目录和必要的一层子目录。
不要深度递归全盘扫描。
不要加载大权重文件。
不要为了识别读取完整模型文件。
checksum 可以作为后续增强，不作为本轮必要项。
```

---

## 7. 扫描识别优先级

目录扫描应按以下顺序：

```text
1. 先判断当前目录本身是否是目录型模型：
   - LoRA / Adapter
   - SentenceTransformers / Embedding
   - Reranker / CrossEncoder
   - Vision-Language / Multimodal
   - HuggingFace Chat / Completion
   - Diffusers / Image Generation
   - OpenVINO / TensorRT directory
   - ASR / TTS / Classification 等可识别目录

2. 如果当前目录不是目录型模型，再扫描目录中的文件型模型：
   - *.gguf
   - *.onnx
   - *.engine

3. 如果同一目录既像 HF directory，又包含 GGUF 文件：
   - 不静默选择。
   - 展示多个 candidate。
   - 可默认预选 HF directory，但用户必须能切换 GGUF 文件。

4. 如果没有发现任何支持的模型文件或目录格式：
   - 报错：目录中没有发现模型文件。
```

注意：目录是扫描入口，不一定是最终部署路径。

```text
HF / SentenceTransformers / Diffusers → 最终路径通常是目录。
GGUF / ONNX / TensorRT engine → 最终路径通常是具体文件。
LoRA Adapter → 可识别，但不能独立部署。
```

---

## 8. 常见模型类型识别规则

### 8.1 Chat / Completion HF directory

识别特征：

```text
config.json
tokenizer.json / tokenizer_config.json
generation_config.json
*.safetensors / pytorch_model.bin
architectures / model_type / name indicates causal LM, instruct, chat
```

输出：

```json
{
  "kind": "directory",
  "format": "huggingface",
  "task": "chat",
  "capabilities": ["chat", "completion"],
  "default_test_mode": "chat",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["vllm", "sglang"],
  "confidence": "medium"
}
```

### 8.2 Embedding model

Embedding 是一等模型类型，必须识别。

识别特征：

```text
modules.json
sentence_bert_config.json
1_Pooling/config.json
config_sentence_transformers.json
name contains embedding / embeddings / bge / bge-m3 / e5 / gte / text2vec / m3e / jina-embeddings / sentence-transformers
```

输出：

```json
{
  "kind": "directory",
  "format": "sentence_transformers",
  "task": "embedding",
  "capabilities": ["embedding"],
  "default_test_mode": "embedding",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["vllm", "sglang"],
  "confidence": "high"
}
```

如果底层结构是标准 HF directory，但名称或 config 明确为 embedding，也可以：

```json
{
  "kind": "directory",
  "format": "huggingface",
  "task": "embedding",
  "capabilities": ["embedding"],
  "default_test_mode": "embedding",
  "deployable": true,
  "recommended_backends": ["vllm", "sglang"]
}
```

UI 必须展示为 Embedding，不能显示为普通 Chat 模型。

### 8.3 Reranker / CrossEncoder

Reranker 是一等模型类型，必须识别。

识别特征：

```text
name contains reranker / rerank / bge-reranker / cross-encoder / ms-marco / ranker / jina-reranker
config indicates sequence classification / num_labels=1 / score/regression
sentence-transformers cross-encoder structure
```

输出：

```json
{
  "kind": "directory",
  "format": "huggingface",
  "task": "rerank",
  "capabilities": ["rerank"],
  "default_test_mode": "rerank",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["vllm", "sglang"],
  "confidence": "medium"
}
```

### 8.4 Vision-Language / Multimodal LLM

识别特征：

```text
name contains qwen-vl / qwen2-vl / qwen2.5-vl / internvl / llava / minicpm-v / glm-4v
config model_type or architectures indicate vision-language
image_processor_config.json
preprocessor_config.json
vision_config
```

输出：

```json
{
  "kind": "directory",
  "format": "huggingface",
  "task": "vision_chat",
  "capabilities": ["chat", "vision"],
  "default_test_mode": "chat",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["vllm", "sglang"],
  "confidence": "medium"
}
```

本轮只要求识别、展示和兼容性，不要求实现图片测试入口。

### 8.5 GGUF file

识别特征：

```text
*.gguf
```

输出：

```json
{
  "kind": "file",
  "format": "gguf",
  "task": "chat",
  "capabilities": ["chat", "completion"],
  "default_test_mode": "chat",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["llamacpp"],
  "confidence": "high"
}
```

强约束：

```text
GGUF 最终 path 必须是具体 .gguf 文件。
llama.cpp RunPlan 的 -m 必须指向容器内具体 .gguf 文件。
不能生成 -m /models/<dir>。
```

### 8.6 LoRA / Adapter

识别特征：

```text
adapter_config.json
adapter_model.safetensors
adapter_model.bin
```

输出：

```json
{
  "kind": "adapter",
  "format": "lora_adapter",
  "task": "adapter",
  "capabilities": [],
  "default_test_mode": "auto",
  "deployable": false,
  "requires_base_model": true,
  "recommended_backends": [],
  "confidence": "high",
  "unsupported_reason": "这是 LoRA/Adapter，需要选择基础模型后使用，不能作为独立模型直接部署。"
}
```

### 8.7 ONNX

识别特征：

```text
*.onnx
```

输出：

```json
{
  "kind": "file",
  "format": "onnx",
  "task": "unknown",
  "capabilities": [],
  "default_test_mode": "auto",
  "deployable": false,
  "recommended_backends": [],
  "confidence": "high",
  "unsupported_reason": "当前平台尚未配置 ONNX Runtime 后端。"
}
```

### 8.8 TensorRT / TensorRT-LLM Engine

识别特征：

```text
*.engine
rank*.engine
```

输出：

```json
{
  "kind": "file",
  "format": "tensorrt_engine",
  "task": "unknown",
  "capabilities": [],
  "default_test_mode": "auto",
  "deployable": false,
  "recommended_backends": [],
  "confidence": "high",
  "unsupported_reason": "当前平台尚未配置 TensorRT-LLM 后端。"
}
```

### 8.9 OpenVINO

识别特征：

```text
*.xml + *.bin
openvino_model.xml
openvino_model.bin
```

输出：

```json
{
  "kind": "directory",
  "format": "openvino",
  "task": "unknown",
  "capabilities": [],
  "default_test_mode": "auto",
  "deployable": false,
  "recommended_backends": [],
  "confidence": "medium",
  "unsupported_reason": "当前平台尚未配置 OpenVINO 后端。"
}
```

### 8.10 Diffusers / Image Generation

识别特征：

```text
model_index.json
unet/
vae/
scheduler/
text_encoder/
```

输出：

```json
{
  "kind": "directory",
  "format": "diffusers",
  "task": "image_generation",
  "capabilities": ["text_to_image"],
  "default_test_mode": "auto",
  "deployable": false,
  "recommended_backends": [],
  "confidence": "high",
  "unsupported_reason": "当前平台尚未配置 Diffusers/Image Generation 后端。"
}
```

### 8.11 ASR / Speech-to-Text

识别特征：

```text
name contains whisper / funasr / paraformer / sensevoice
preprocessor_config.json
feature_extractor_config.json
```

输出：

```json
{
  "kind": "directory",
  "format": "huggingface",
  "task": "asr",
  "capabilities": ["speech_to_text"],
  "default_test_mode": "auto",
  "deployable": false,
  "recommended_backends": [],
  "confidence": "medium",
  "unsupported_reason": "当前平台尚未配置 ASR 后端。"
}
```

### 8.12 TTS

识别特征：

```text
name contains cosyvoice / chattts / gpt-sovits / fish-speech
```

输出：

```json
{
  "kind": "directory",
  "format": "huggingface",
  "task": "tts",
  "capabilities": ["text_to_speech"],
  "default_test_mode": "auto",
  "deployable": false,
  "recommended_backends": [],
  "confidence": "low",
  "unsupported_reason": "当前平台尚未配置 TTS 后端。"
}
```

### 8.13 Classification / Token Classification

识别特征：

```text
config architectures indicate SequenceClassification / TokenClassification
id2label / label2id
```

输出：

```json
{
  "kind": "directory",
  "format": "huggingface",
  "task": "classification",
  "capabilities": ["classify"],
  "default_test_mode": "auto",
  "deployable": false,
  "recommended_backends": [],
  "confidence": "medium",
  "unsupported_reason": "当前平台尚未配置分类模型服务后端。"
}
```

---

## 9. 模型类型插件规范

长期目标是新增模型类型时，只新增一个 detector/plugin 定义。

建议每类模型用统一 spec 描述：

```json
{
  "type_id": "embedding.sentence_transformers",
  "labels": {
    "zh-CN": "Embedding / SentenceTransformers",
    "en-US": "Embedding / SentenceTransformers"
  },
  "kind": "directory",
  "formats": ["sentence_transformers", "huggingface"],
  "tasks": ["embedding"],
  "capabilities": ["embedding"],
  "default_test_mode": "embedding",
  "recommended_backends": ["vllm", "sglang"],
  "deployable": true,
  "requires_base_model": false,
  "detector": {
    "required_any": ["modules.json", "sentence_bert_config.json", "config_sentence_transformers.json"],
    "name_contains_any": ["embedding", "embeddings", "bge", "e5", "gte", "text2vec", "m3e", "jina-embeddings"]
  }
}
```

当前阶段可以先在 Go 中实现 registry；后续可迁移到文件化 catalog，例如：

```text
configs/model-type-catalog/system/*.yaml
configs/model-type-catalog.d/*.yaml
```

原则：

```text
新增一种模型类型 = 新增 detector/spec + compatibility metadata + i18n labels + tests。
不应该改多个业务页面和 RunPlan resolver。
```

---

## 10. Backend 能力抽象

BackendVersion / BackendRuntime 需要表达或派生其支持能力。

建议 backend capability spec：

```json
{
  "backend": "vllm",
  "supported_formats": ["huggingface", "sentence_transformers"],
  "supported_tasks": ["chat", "completion", "embedding", "rerank", "vision_chat"],
  "supported_capabilities": ["chat", "completion", "embedding", "rerank", "vision"],
  "model_path_mode": "directory",
  "test_endpoints": {
    "chat": "/v1/chat/completions",
    "completion": "/v1/completions",
    "embedding": "/v1/embeddings",
    "rerank": ["/v1/rerank", "/rerank", "/v2/rerank", "/score", "/v1/score"]
  }
}
```

示例：

### vLLM

```text
formats: huggingface, sentence_transformers
tasks: chat, completion, embedding, rerank, vision_chat
capabilities: chat, completion, embedding, rerank, vision
model_path_mode: directory
```

### SGLang

```text
formats: huggingface, sentence_transformers
tasks: chat, completion, embedding, rerank, vision_chat
capabilities: chat, completion, embedding, rerank, vision
model_path_mode: directory
```

### llama.cpp

```text
formats: gguf
tasks: chat, completion
capabilities: chat, completion
model_path_mode: file
```

备注：llama.cpp 的 embedding 能力可以后续单独启用，但不要默认把所有 GGUF 都当 embedding。

---

## 11. CompatibilityChecker 规则

CompatibilityChecker 输入：

```text
ModelDescriptor / ModelArtifact
ModelLocation
BackendVersion / BackendRuntime / NodeBackendRuntime
```

输出：

```json
{
  "compatible": true,
  "severity": "ok | warning | error",
  "reason": "",
  "supported_test_modes": ["chat", "embedding"],
  "preferred_test_mode": "embedding",
  "run_method": "vllm-directory"
}
```

不兼容时：

```json
{
  "compatible": false,
  "severity": "error",
  "reason": "当前模型为 GGUF 文件，所选运行后端 vLLM 不支持。请使用 llama.cpp。"
}
```

必须阻止的组合：

```text
vLLM/SGLang + GGUF file → fail
llama.cpp + HuggingFace directory → fail
LoRA/Adapter standalone deploy → fail
ONNX without ONNX Runtime backend → fail
TensorRT engine without TensorRT backend → fail
OpenVINO without OpenVINO backend → fail
Diffusers without image generation backend → fail
ASR/TTS without corresponding backend → fail
```

---

## 12. Run Method 抽象

模型运行方法由模型类型和 backend 能力共同决定。

建议抽象：

```json
{
  "run_method_id": "vllm.hf-directory.chat",
  "backend": "vllm",
  "accepted_formats": ["huggingface", "sentence_transformers"],
  "accepted_tasks": ["chat", "completion", "embedding", "rerank", "vision_chat"],
  "path_mode": "directory",
  "model_arg_template": "{{model_container_dir}}",
  "test_modes": ["chat", "completion", "embedding", "rerank"]
}
```

### vLLM / HF directory

```text
path_mode = directory
model argument = /models/<hf-dir>
chat endpoint = /v1/chat/completions
embedding endpoint = /v1/embeddings
rerank endpoint = backend-declared rerank endpoint
```

### SGLang / HF directory

```text
path_mode = directory
model argument = /models/<hf-dir>
chat endpoint = /v1/chat/completions
embedding endpoint = /v1/embeddings
rerank endpoint = backend-declared rerank endpoint
```

### llama.cpp / GGUF file

```text
path_mode = file
model argument = /models/<dir>/<file>.gguf
chat/completion endpoint = current llama.cpp server endpoints
```

关键规则：

```text
RunPlanResolver 不能自己猜模型类型。
RunPlanResolver 只消费 compatibility 已确认的 run_method。
```

---

## 13. Test Method 抽象

测试方法由模型能力和 backend endpoint 共同决定。

建议定义：

```json
{
  "test_mode": "embedding",
  "required_capability": "embedding",
  "endpoint_candidates": ["/v1/embeddings"],
  "payload_template": {
    "input": "hello world"
  },
  "success_criteria": "HTTP 2xx and valid embedding vector/list"
}
```

### Chat 测试

```json
{
  "messages": [
    {"role": "system", "content": "You are a test endpoint. Reply with exactly one word: pong"},
    {"role": "user", "content": "ping"}
  ],
  "temperature": 0,
  "max_tokens": 8
}
```

成功标准：HTTP 2xx 且返回 OpenAI-compatible chat response。不要强制输出必须等于 `pong`。

### Completion 测试

```json
{
  "prompt": "Reply with exactly one word: pong",
  "temperature": 0,
  "max_tokens": 8
}
```

### Embedding 测试

```json
{
  "input": "hello world"
}
```

成功标准：HTTP 2xx 且返回 embedding vector/list。

### Rerank 测试

```json
{
  "query": "what is gpu",
  "documents": [
    "gpu is a graphics processing unit",
    "apple is a fruit"
  ]
}
```

endpoint 候选：

```text
/v1/rerank
/rerank
/v2/rerank
/score
/v1/score
```

如果 backend 未声明 rerank endpoint，显示：

```text
该模型识别为 Reranker，但当前运行后端未声明 Rerank 测试端点。
```

---

## 14. 持久化设计

优先不新增 schema，使用现有字段承载。

已知可用字段：

```text
capabilities_json
capability_sources_json
default_test_mode
metadata_json
```

建议 metadata_json 结构：

```json
{
  "kind": "directory",
  "format": "sentence_transformers",
  "task": "embedding",
  "deployable": true,
  "requires_base_model": false,
  "recommended_backends": ["vllm", "sglang"],
  "confidence": "high",
  "evidence": ["modules.json", "1_Pooling/config.json"],
  "unsupported_reason": "",
  "detector_id": "sentence_transformers_embedding",
  "scan_root": "/home/kzeng/models/bge-m3"
}
```

ModelLocation 规则：

```text
HF directory / SentenceTransformers / Reranker / VLM:
- location path = 模型目录
- kind = directory

GGUF:
- location path = 具体 .gguf 文件
- kind = file

LoRA Adapter:
- location path = adapter 目录或文件
- kind = adapter / directory
- deployable = false
```

如果未来 schema 需要规范化，可新增字段：

```text
model_artifacts.task
model_artifacts.deployable
model_artifacts.recommended_backends_json
model_locations.kind
model_locations.scan_metadata_json
```

但本轮默认不做 schema 变更。

---

## 15. UI 展示规则

### 15.1 扫描结果列表

每个 candidate 必须显示：

```text
名称
路径
存储形态
模型格式
任务类型
能力
默认测试方式
可部署性
推荐后端
识别置信度
识别证据
不支持原因，如有
```

示例：

```text
Qwen3-0.6B-Instruct-2512
HuggingFace / Chat
能力：Chat, Completion
默认测试方式：Chat
推荐后端：vLLM, SGLang
证据：config.json, tokenizer_config.json, generation_config.json

bge-m3
SentenceTransformers / Embedding
能力：Embedding
默认测试方式：Embedding
推荐后端：vLLM, SGLang
证据：modules.json, 1_Pooling/config.json

bge-reranker-v2-m3
HuggingFace / Reranker
能力：Rerank
默认测试方式：Rerank
推荐后端：vLLM, SGLang
证据：name contains reranker, config num_labels=1

Qwen3.5-9B-Q4_K_M.gguf
GGUF 文件 / Chat
能力：Chat, Completion
默认测试方式：Chat
推荐后端：llama.cpp

adapter_model.safetensors
LoRA Adapter
不可独立部署：需要基础模型

model.onnx
ONNX
当前不支持：未配置 ONNX Runtime 后端
```

### 15.2 模型详情页

必须展示：

```text
存储形态
模型格式
任务类型
能力
默认测试方式
可部署性
推荐后端
能力来源
识别证据
不支持原因
```

### 15.3 模型编辑页

至少支持修改：

```text
任务类型
能力
默认测试方式
```

用户可以修正误判：

```text
Unknown → Embedding
Chat → Reranker
default_test_mode → embedding/rerank/chat
```

不得显示：

```text
undefined
null
[object Object]
format.xxx
task.xxx
capability.xxx
```

---

## 16. Preflight / RunPlan 规则

Preflight 必须先调用 CompatibilityChecker。

如果不兼容：

```text
不生成 RunPlan。
不允许进入部署。
返回明确错误。
```

如果兼容：

```text
生成 RunPlan。
RunPlan 中必须展示使用的模型类型、path_mode、run_method。
```

关键断言：

```text
llama.cpp + GGUF:
- -m 必须指向具体 .gguf 文件。

vLLM/SGLang + HF directory:
- model path 必须指向目录。

Embedding/Reranker:
- 只能选择声明支持 embedding/rerank 的 backend。

LoRA/Adapter:
- 不能独立部署。
```

---

## 17. 实施步骤建议

### Step 1：写入设计文档

将本文档保存到：

```text
docs/reports/phase-3/web-ai-config-review/28-model-detection-runtime-plugin-design.md
```

### Step 2：梳理现有代码

检查：

```bash
grep -R "model-paths/scan\|ModelLocation\|capabilities_json\|default_test_mode\|format\|gguf\|huggingface\|RunPlan\|preflight" -n internal cmd web/src | cat
```

明确：

```text
1. agent scanner 当前输出什么。
2. server scan proxy 是否丢字段。
3. frontend wizard 如何展示 candidate。
4. create model/location 时保存哪些字段。
5. model detail/edit 如何展示 capabilities。
6. preflight/RunPlan 如何选择模型路径。
7. test dialog 如何选择测试方式。
```

### Step 3：实现 ModelCandidate / Detector Registry

优先在后端/agent scanner 内统一模型识别输出。

### Step 4：实现 detectors

至少实现：

```text
HF Chat/Completion
SentenceTransformers/Embedding
Reranker/CrossEncoder
Vision-Language
GGUF
LoRA/Adapter
ONNX
TensorRT Engine
OpenVINO
Diffusers
ASR
TTS
Classification
```

其中 unsupported 类型可以只识别和展示，不允许运行。

### Step 5：持久化 candidate 语义

用户选择 candidate 后，保存：

```text
format
task
capabilities
default_test_mode
metadata.kind
metadata.deployable
metadata.requires_base_model
metadata.recommended_backends
metadata.evidence
metadata.unsupported_reason
```

### Step 6：实现 CompatibilityChecker

在 preflight / RunPlan 前校验模型与 backend 是否兼容。

### Step 7：更新 UI

扫描结果、模型详情、模型编辑、部署选择页面都要展示模型类型/能力/兼容性。

### Step 8：更新测试入口

根据 default_test_mode 和 backend endpoint 选择测试方法。

### Step 9：测试和 closeout

执行完整测试、写 closeout、commit、push。

---

## 18. 测试要求

必须新增/更新测试覆盖：

```text
1. Chat HF directory scan
2. Embedding directory scan
3. Reranker directory scan
4. Vision-language directory scan
5. GGUF file scan
6. Directory scan with one GGUF
7. Directory scan with multiple GGUF and selected file
8. Empty directory
9. LoRA adapter recognition
10. ONNX recognition unsupported
11. TensorRT engine recognition unsupported
12. OpenVINO recognition unsupported
13. Diffusers recognition unsupported
14. ASR recognition unsupported
15. TTS recognition unsupported
16. Classification recognition unsupported
17. Backend compatibility:
    - vLLM + GGUF fails
    - llama.cpp + HF fails
    - LoRA standalone deploy fails
    - ONNX without backend fails
18. Test mode:
    - embedding defaults to embedding
    - reranker defaults to rerank or clear unsupported endpoint message
19. UI i18n:
    - no undefined/null/[object Object]/format.xxx/task.xxx/capability.xxx leakage
```

---

## 19. 验证命令

必须运行：

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

如果修改 agent scanner，也运行对应 agent/server 测试。

---

## 20. Claude 执行要求

Claude 阅读本文档后，应先输出一个简短 review：

```text
1. 是否认可本文档抽象。
2. 当前代码与设计的主要差异。
3. 是否需要 schema 变更；默认不需要，如认为需要必须说明理由。
4. 分阶段实现计划。
5. 每阶段验证命令。
```

如果没有阻塞问题，直接开始实现，不要停留在计划阶段。

实现过程中如果发现问题：

```text
能修复、能验证的问题必须本轮修复。
除非属于外部依赖、无硬件环境、或高风险大重构，否则不要写 future。
```

---

## 21. Closeout 要求

完成后新增：

```text
docs/reports/phase-3/web-ai-config-review/29-model-detection-runtime-plugin-closeout.md
```

Closeout 必须包含：

```text
1. 本轮目标
2. ModelCandidate 抽象实现情况
3. Detector Registry 实现情况
4. 已识别模型类型清单
5. 各类型可部署/不可部署策略
6. Backend compatibility 实现情况
7. RunPlan 前置校验结果
8. 测试入口实现情况
9. 模型详情页/编辑页展示结果
10. 是否修改 schema：必须说明
11. 是否新增 migration：必须说明
12. 是否进入 Phase 3：必须否
13. 测试命令和结果
14. 修改文件清单
15. commit id
16. push 结果
17. final git status
```

---

## 22. Commit 要求

完成后：

```bash
git add .
git commit -m "feat(web-ai): abstract model detection and runtime compatibility"
git push
git status --short
```

最终报告必须包含：

```text
1. 设计文档路径
2. closeout 文档路径
3. 是否完成 ModelCandidate 抽象
4. 是否完成 Detector Registry 或等价插件化机制
5. 已支持识别哪些模型类型
6. 哪些模型类型可部署，哪些只是识别但当前不可运行
7. backend compatibility 是否生效
8. 模型属性/能力/default_test_mode 是否展示并持久化
9. 测试入口是否支持 embedding/rerank 默认测试
10. 是否未进入 Phase 3
11. 是否修改 schema/migration
12. 测试结果
13. commit id
14. push 结果
15. git status 是否 clean
```
