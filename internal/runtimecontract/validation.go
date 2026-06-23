package runtimecontract

// IsValidFormat returns true if s is a known model format identifier.
func IsValidFormat(s string) bool {
	switch s {
	case FormatHuggingFace, FormatSentenceTransformers, FormatGGUF, FormatLoRAAdapter,
		FormatDiffusers, FormatONNX, FormatTensorRT, FormatOpenVINO, FormatOllama:
		return true
	default:
		return false
	}
}

// IsValidTask returns true if s is a known task identifier.
func IsValidTask(s string) bool {
	switch s {
	case TaskChat, TaskCompletion, TaskEmbedding, TaskRerank, TaskVisionChat,
		TaskAdapter, TaskUnknown, TaskImageGeneration, TaskASR, TaskTTS, TaskClassification:
		return true
	default:
		return false
	}
}

// IsValidCapability returns true if s is a known capability identifier.
func IsValidCapability(s string) bool {
	switch s {
	case CapabilityChat, CapabilityCompletion, CapabilityEmbedding, CapabilityRerank,
		CapabilityVision, CapabilityImageGeneration, CapabilityASR, CapabilityTTS,
		CapabilityClassification, CapabilityToolCalling, CapabilityStructuredOutput:
		return true
	default:
		return false
	}
}

// IsValidPathMode returns true if s is a known path mode identifier.
func IsValidPathMode(s string) bool {
	switch s {
	case PathModeDirectory, PathModeFile, PathModeOllamaManaged:
		return true
	default:
		return false
	}
}

// IsValidCapabilitySource returns true if s is a known capability source identifier.
func IsValidCapabilitySource(s string) bool {
	switch s {
	case CapabilitySourceScan, CapabilitySourceInferred,
		CapabilitySourceUserOverride, CapabilitySourceBackendProbe:
		return true
	default:
		return false
	}
}

// IsValidTestMode returns true if s is a known test mode identifier.
func IsValidTestMode(s string) bool {
	switch s {
	case TestModeAuto, TestModeChat, TestModeCompletion, TestModeEmbedding, TestModeRerank:
		return true
	default:
		return false
	}
}

// IsValidServingProtocol returns true if s is a known serving protocol identifier.
func IsValidServingProtocol(s string) bool {
	switch s {
	case ServingProtocolOpenAICompatible, ServingProtocolOllama:
		return true
	default:
		return false
	}
}

// ── Canonical list accessors for API use ──

// AllFormats returns the canonical list of known model formats.
func AllFormats() []string {
	return []string{
		FormatGGUF, FormatHuggingFace, FormatSentenceTransformers,
		FormatLoRAAdapter, FormatDiffusers, FormatONNX,
		FormatTensorRT, FormatOpenVINO, FormatOllama,
	}
}

// AllTasks returns the canonical list of known tasks.
func AllTasks() []string {
	return []string{
		TaskChat, TaskCompletion, TaskEmbedding, TaskRerank,
		TaskVisionChat, TaskAdapter, TaskUnknown,
		TaskImageGeneration, TaskASR, TaskTTS, TaskClassification,
	}
}

// AllCapabilities returns the canonical list of known capabilities.
func AllCapabilities() []string {
	return []string{
		CapabilityChat, CapabilityCompletion, CapabilityEmbedding,
		CapabilityRerank, CapabilityVision, CapabilityImageGeneration,
		CapabilityASR, CapabilityTTS, CapabilityClassification,
		CapabilityToolCalling, CapabilityStructuredOutput,
	}
}

// AllPathModes returns the canonical list of known path modes.
func AllPathModes() []string {
	return []string{PathModeDirectory, PathModeFile, PathModeOllamaManaged}
}

// AllCapabilitySources returns the canonical list of known capability sources.
func AllCapabilitySources() []string {
	return []string{
		CapabilitySourceScan, CapabilitySourceInferred,
		CapabilitySourceUserOverride, CapabilitySourceBackendProbe,
	}
}

// AllTestModes returns the canonical list of known test modes.
func AllTestModes() []string {
	return []string{
		TestModeAuto, TestModeChat, TestModeCompletion,
		TestModeEmbedding, TestModeRerank,
	}
}

// AllServingProtocols returns the canonical list of known serving protocols.
func AllServingProtocols() []string {
	return []string{ServingProtocolOpenAICompatible, ServingProtocolOllama}
}
