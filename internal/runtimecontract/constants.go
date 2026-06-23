// Package runtimecontract defines the canonical enum vocabulary for LightAI Go's
// model runtime contract system (formats, tasks, capabilities, path modes, etc.).
//
// This package is a NEUTRAL dependency — it imports nothing outside the standard
// library. Both internal/server/ and internal/agent/ can import it without
// creating reverse dependencies.
//
// Canonical source chain:
//
//	runtimecontract constants  →  compile-time truth (this package)
//	DB backend_versions.capabilities_json  →  runtime truth (preflight reads this)
//	configs/backend-catalog/  →  human-readable catalog (must match DB)
//	GET /api/enums/model-capabilities  →  frontend source (reads from these constants)
package runtimecontract

// ── Format constants ──

const (
	FormatHuggingFace          = "huggingface"
	FormatSentenceTransformers = "sentence_transformers"
	FormatGGUF                 = "gguf"
	FormatLoRAAdapter          = "lora_adapter"
	FormatDiffusers            = "diffusers"
	FormatONNX                 = "onnx"
	FormatTensorRT             = "tensorrt_engine"
	FormatOpenVINO             = "openvino"
	FormatOllama               = "ollama"
)

// ── Task constants ──

const (
	TaskChat            = "chat"
	TaskCompletion      = "completion"
	TaskEmbedding       = "embedding"
	TaskRerank          = "rerank"
	TaskVisionChat      = "vision_chat"
	TaskAdapter         = "adapter"
	TaskUnknown         = "unknown"
	TaskImageGeneration = "image_generation"
	TaskASR             = "asr"
	TaskTTS             = "tts"
	TaskClassification  = "classification"
)

// ── Capability constants ──

const (
	CapabilityChat             = "chat"
	CapabilityCompletion       = "completion"
	CapabilityEmbedding        = "embedding"
	CapabilityRerank           = "rerank"
	CapabilityVision           = "vision"
	CapabilityImageGeneration  = "image_generation"
	CapabilityASR              = "asr"
	CapabilityTTS              = "tts"
	CapabilityClassification   = "classification"
	CapabilityToolCalling      = "tool_calling"
	CapabilityStructuredOutput = "structured_output"
)

// ── PathMode constants ──

const (
	PathModeDirectory     = "directory"
	PathModeFile          = "file"
	PathModeOllamaManaged = "ollama_managed"
)

// ── CapabilitySource constants ──

const (
	CapabilitySourceScan         = "scan"
	CapabilitySourceInferred     = "inferred"
	CapabilitySourceUserOverride = "user_override"
	CapabilitySourceBackendProbe = "backend_probe"
)

// ── TestMode constants ──

const (
	TestModeAuto       = "auto"
	TestModeChat       = "chat"
	TestModeCompletion = "completion"
	TestModeEmbedding  = "embedding"
	TestModeRerank     = "rerank"
)

// ── ServingProtocol constants ──

const (
	ServingProtocolOpenAICompatible = "openai-compatible"
	ServingProtocolOllama           = "ollama"
)

// ── Compatibility status codes (from CheckCompatibility) ──

const (
	CompatCodeOK                       = "ok"
	CompatCodeFormatMismatch           = "format_mismatch"
	CompatCodeTaskMismatch             = "task_mismatch"
	CompatCodePathModeMismatch         = "path_mode_mismatch"
	CompatCodeArchitectureBlocked      = "architecture_blocked"
	CompatCodeNotDeployable            = "not_deployable"
	CompatCodeBackendCapabilityMissing = "backend_capability_missing"
)
