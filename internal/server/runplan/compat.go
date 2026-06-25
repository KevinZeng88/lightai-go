package runplan

import "encoding/json"

// ModelDescriptor captures model-level facts for compatibility checking.
type ModelDescriptor struct {
	Format       string
	Task         string
	Deployable   bool
	PathType     string
	Architecture string // model architecture (e.g., "InternVLChatModel")
}

// BackendDescriptor captures backend-level capability facts.
type BackendDescriptor struct {
	BackendName           string
	SupportedFormats      []string
	SupportedTasks        []string
	SupportedCapabilities []string
	ModelPathModes        []string
	ServingProtocols      []string // e.g., "openai-compatible", "ollama"
	TestEndpoints         map[string]interface{}
	BlockedArchitectures  map[string]string // arch → reason for blocking
}

// CompatResult is the output of a compatibility check.
type CompatResult struct {
	Compatible bool
	Code       string // short machine-readable code
	Reason     string // human-readable Chinese message
}

// CheckCompatibility validates that a model can run on a given backend.
func CheckCompatibility(model ModelDescriptor, backend BackendDescriptor) CompatResult {
	// 1. Backend capability must be declared.
	if len(backend.SupportedFormats) == 0 {
		return CompatResult{false, "backend_capability_missing",
			"后端能力未声明，无法确认该模型是否可运行。"}
	}

	// 2. Deployable check.
	if !model.Deployable {
		return CompatResult{false, "not_deployable",
			"该模型不能独立部署。"}
	}

	// 3. Format check.
	formatOK := false
	for _, f := range backend.SupportedFormats {
		if f == model.Format {
			formatOK = true
			break
		}
	}
	if !formatOK {
		return CompatResult{false, "format_mismatch",
			formatMismatchMsg(model.Format, backend.BackendName)}
	}

	// 4. Path type / model_path_mode check.
	pathOK := false
	for _, pm := range backend.ModelPathModes {
		if pm == model.PathType {
			pathOK = true
			break
		}
	}
	if !pathOK {
		return CompatResult{false, "path_mode_mismatch",
			pathModeMismatchMsg(model.PathType, backend.BackendName, backend.ModelPathModes)}
	}

	// 5. Architecture-level blocking (VLM triage: E2E evidence shows InternVL fails).
	if reason, blocked := backend.BlockedArchitectures[model.Architecture]; blocked {
		return CompatResult{false, "architecture_blocked", reason}
	}

	// 6. Task check.
	taskOK := false
	for _, t := range backend.SupportedTasks {
		if t == model.Task {
			taskOK = true
			break
		}
	}
	if !taskOK {
		return CompatResult{false, "task_mismatch",
			taskMismatchMsg(model.Task, backend.BackendName)}
	}

	return CompatResult{true, "ok", ""}
}

// ParseBackendCapabilities unmarshals ConfigSet backend capability data into BackendDescriptor.
func ParseBackendCapabilities(capabilitiesJSON string) (BackendDescriptor, error) {
	var caps struct {
		SupportedFormats      []string               `json:"supported_formats"`
		SupportedTasks        []string               `json:"supported_tasks"`
		SupportedCapabilities []string               `json:"supported_capabilities"`
		ModelPathModes        []string               `json:"model_path_modes"`
		ServingProtocols      []string               `json:"serving_protocols"`
		TestEndpoints         map[string]interface{} `json:"test_endpoints"`
		BlockedArchitectures  map[string]string      `json:"blocked_architectures"`
	}
	if err := json.Unmarshal([]byte(capabilitiesJSON), &caps); err != nil {
		return BackendDescriptor{}, err
	}
	if caps.BlockedArchitectures == nil {
		caps.BlockedArchitectures = map[string]string{}
	}
	return BackendDescriptor{
		SupportedFormats:      caps.SupportedFormats,
		SupportedTasks:        caps.SupportedTasks,
		SupportedCapabilities: caps.SupportedCapabilities,
		ModelPathModes:        caps.ModelPathModes,
		ServingProtocols:      caps.ServingProtocols,
		TestEndpoints:         caps.TestEndpoints,
		BlockedArchitectures:  caps.BlockedArchitectures,
	}, nil
}

func formatMismatchMsg(format, backend string) string {
	switch {
	case format == "gguf" && (backend == "vllm" || backend == "sglang"):
		return "模型为 GGUF 文件，vLLM/SGLang 不支持。请使用 llama.cpp。"
	case format == "huggingface" && backend == "llamacpp":
		return "模型为 HuggingFace 目录，llama.cpp 不支持。请使用 vLLM 或 SGLang。"
	case format == "sentence_transformers" && backend == "llamacpp":
		return "模型为 SentenceTransformers/Embedding 格式，llama.cpp 不支持。请使用 vLLM 或 SGLang。"
	case format == "lora_adapter":
		return "这是 LoRA/Adapter，需要选择基础模型后使用，不能作为独立模型直接部署。"
	default:
		return "模型格式 " + format + " 与后端 " + backend + " 不兼容。"
	}
}

func pathModeMismatchMsg(pathType, backend string, allowed []string) string {
	allowStr := ""
	for i, m := range allowed {
		if i > 0 {
			allowStr += ", "
		}
		allowStr += m
	}
	return "模型路径类型为 " + pathType + "，但后端 " + backend + " 需要 " + allowStr + " 路径。"
}

func taskMismatchMsg(task, backend string) string {
	return "模型任务为 " + task + "，但当前后端 " + backend + " 未声明 " + task + " 支持。"
}
