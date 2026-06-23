package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lightai-go/internal/runtimecontract"
)

// ScanCandidate represents one model candidate found during scanning.
type ScanCandidate struct {
	Path     string `json:"path"`
	PathType string `json:"path_type"` // "file" or "directory"

	// ── Phase A+B1: enriched model type fields ──
	Kind                string   `json:"kind"`                 // "directory" | "file" | "adapter" | "bundle"
	Format              string   `json:"format"`               // "huggingface" | "sentence_transformers" | "gguf" | "lora_adapter" | ...
	Task                string   `json:"task"`                 // "chat" | "completion" | "embedding" | "rerank" | "vision_chat" | "adapter" | "unknown"
	Capabilities        []string `json:"capabilities"`         // ["chat","completion"] | ["embedding"] | ...
	DefaultTestMode     string   `json:"default_test_mode"`    // "chat" | "completion" | "embedding" | "rerank" | "auto"
	Deployable          bool     `json:"deployable"`           // can it run as standalone?
	RequiresBaseModel   bool     `json:"requires_base_model"`  // adapter/lora case
	RecommendedBackends []string `json:"recommended_backends"` // ["vllm","sglang"] | ["llamacpp"] | []
	Confidence          string   `json:"confidence"`           // "high" | "medium" | "low"
	Evidence            []string `json:"evidence"`             // e.g. ["config.json","tokenizer_config.json"]
	UnsupportedReason   string   `json:"unsupported_reason"`   // only when deployable=false

	// ── Existing fields ──
	DetectedMetadata map[string]interface{} `json:"detected_metadata"`
	Warnings         []string               `json:"warnings"`
	AutoSelected     bool                   `json:"auto_selected"`
	SelectionReason  string                 `json:"selection_reason"`
	SizeBytes        int64                  `json:"size_bytes"`
	SizeLabel        string                 `json:"size_label"`
}

// ScanResult is the structured result of a model path scan.
type ScanResult struct {
	ScanRoot   string          `json:"scan_root"`
	Candidates []ScanCandidate `json:"candidates"`
	Warnings   []string        `json:"warnings"`
	Error      string          `json:"error,omitempty"`
}

// ── Plugin abstraction ──

// FileFacts collects low-cost filesystem facts once per scanned directory.
// Detectors read from FileFacts instead of accessing the filesystem directly.
type FileFacts struct {
	AbsPath     string
	IsDirectory bool
	DirEntries  []string // base names of entries in this directory

	// Parsed JSON files (nil if not present or parse failed)
	ConfigJSON             map[string]interface{}
	GenerationConfigJSON   map[string]interface{}
	TokenizerConfigJSON    map[string]interface{}
	SentenceBertConfigJSON map[string]interface{}
	AdapterConfigJSON      map[string]interface{}
	ModelIndexJSON         map[string]interface{}

	// File glob results
	GGUFFiles       []string // *.gguf
	ONNXFiles       []string // *.onnx
	EngineFiles     []string // *.engine
	SafetensorFiles []string // *.safetensors
	XMLFiles        []string // *.xml
	BinFiles        []string // *.bin (for OpenVINO)
	PTModelFiles    []string // pytorch_model.bin

	// Presence checks
	HasConfigJSON               bool
	HasTokenizerConfigJSON      bool
	HasGenerationConfigJSON     bool
	HasSentenceBertConfigJSON   bool
	HasModulesJSON              bool
	HasOnePooling               bool // 1_Pooling/config.json exists
	HasAdapterConfigJSON        bool
	HasAdapterModelSafetensors  bool
	HasPreprocessorConfigJSON   bool
	HasImageProcessorConfigJSON bool
	HasModelIndexJSON           bool

	EvidenceFiles []string // files found during fact collection
}

// ModelTypeDefaults holds default values applied to candidates produced by a detector.
type ModelTypeDefaults struct {
	Kind                string
	Format              string
	Task                string
	Capabilities        []string
	DefaultTestMode     string
	Deployable          bool
	RequiresBaseModel   bool
	RecommendedBackends []string
	Confidence          string
	UnsupportedReason   string
}

// DetectorFunc receives pre-collected FileFacts and returns zero or more ScanCandidate.
type DetectorFunc func(facts FileFacts) []ScanCandidate

// ModelTypePlugin bundles a detector with its defaults.
type ModelTypePlugin struct {
	ID       string
	Detect   DetectorFunc
	Defaults ModelTypeDefaults
}

// ── Plugin registry (priority-ordered) ──

var modelTypePlugins = []ModelTypePlugin{
	{ID: "lora_adapter", Detect: DetectLoRAAdapter, Defaults: loRADefaults},
	{ID: "sentence_transformers", Detect: DetectSentenceTransformers, Defaults: sentenceTransformersDefaults},
	{ID: "reranker", Detect: DetectReranker, Defaults: rerankerDefaults},
	{ID: "vision_language", Detect: DetectVisionLanguage, Defaults: visionLanguageDefaults},
	{ID: "diffusers", Detect: DetectDiffusers, Defaults: diffusersDefaults},
	{ID: "asr", Detect: DetectASR, Defaults: asrDefaults},
	{ID: "tts", Detect: DetectTTS, Defaults: ttsDefaults},
	{ID: "classification", Detect: DetectClassification, Defaults: classificationDefaults},
	{ID: "hf_chat", Detect: DetectHuggingFaceChat, Defaults: hfChatDefaults},
	{ID: "openvino", Detect: DetectOpenVINO, Defaults: openvinoDefaults},
	{ID: "tensorrt", Detect: DetectTensorRT, Defaults: tensorrtDefaults},
	{ID: "onnx", Detect: DetectONNX, Defaults: onnxDefaults},
	{ID: "gguf", Detect: DetectGGUF, Defaults: ggufDefaults},
}

// Plugin defaults
var (
	loRADefaults = ModelTypeDefaults{
		Kind: "adapter", Format: runtimecontract.FormatLoRAAdapter, Task: runtimecontract.TaskAdapter,
		Capabilities: []string{}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: true,
		RecommendedBackends: []string{}, Confidence: "high",
		UnsupportedReason: "这是 LoRA/Adapter，需要选择基础模型后使用，不能作为独立模型直接部署。",
	}
	sentenceTransformersDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatSentenceTransformers, Task: runtimecontract.TaskEmbedding,
		Capabilities: []string{runtimecontract.CapabilityEmbedding}, DefaultTestMode: runtimecontract.TestModeEmbedding,
		Deployable: true, RequiresBaseModel: false,
		RecommendedBackends: []string{"vllm", "sglang"}, Confidence: "high",
	}
	rerankerDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatHuggingFace, Task: runtimecontract.TaskRerank,
		Capabilities: []string{runtimecontract.CapabilityRerank}, DefaultTestMode: runtimecontract.TestModeRerank,
		Deployable: true, RequiresBaseModel: false,
		RecommendedBackends: []string{"vllm", "sglang"}, Confidence: "medium",
	}
	visionLanguageDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatHuggingFace, Task: runtimecontract.TaskVisionChat,
		Capabilities: []string{runtimecontract.CapabilityChat, runtimecontract.CapabilityVision}, DefaultTestMode: runtimecontract.TestModeChat,
		Deployable: true, RequiresBaseModel: false,
		RecommendedBackends: []string{"vllm", "sglang"}, Confidence: "high",
	}
	hfChatDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatHuggingFace, Task: runtimecontract.TaskChat,
		Capabilities: []string{runtimecontract.CapabilityChat, runtimecontract.CapabilityCompletion}, DefaultTestMode: runtimecontract.TestModeChat,
		Deployable: true, RequiresBaseModel: false,
		RecommendedBackends: []string{"vllm", "sglang"}, Confidence: "medium",
	}
	diffusersDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatDiffusers, Task: runtimecontract.TaskImageGeneration,
		Capabilities: []string{runtimecontract.CapabilityImageGeneration}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: false,
		RecommendedBackends: []string{}, Confidence: "high",
		UnsupportedReason: "当前平台尚未配置 Diffusers/Image Generation 后端。",
	}
	asrDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatHuggingFace, Task: runtimecontract.TaskASR,
		Capabilities: []string{runtimecontract.CapabilityASR}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: false,
		RecommendedBackends: []string{}, Confidence: "medium",
		UnsupportedReason: "当前平台尚未配置 ASR 后端。",
	}
	ttsDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatHuggingFace, Task: runtimecontract.TaskTTS,
		Capabilities: []string{runtimecontract.CapabilityTTS}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: false,
		RecommendedBackends: []string{}, Confidence: "medium",
		UnsupportedReason: "当前平台尚未配置 TTS 后端。",
	}
	classificationDefaults = ModelTypeDefaults{
		Kind: "directory", Format: runtimecontract.FormatHuggingFace, Task: runtimecontract.TaskClassification,
		Capabilities: []string{runtimecontract.CapabilityClassification}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: false,
		RecommendedBackends: []string{}, Confidence: "medium",
		UnsupportedReason: "当前平台尚未配置分类模型服务后端。",
	}
	openvinoDefaults = ModelTypeDefaults{
		Kind: "bundle", Format: runtimecontract.FormatOpenVINO, Task: runtimecontract.TaskUnknown,
		Capabilities: []string{}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: false,
		RecommendedBackends: []string{}, Confidence: "high",
		UnsupportedReason: "当前平台尚未配置 OpenVINO 后端。",
	}
	tensorrtDefaults = ModelTypeDefaults{
		Kind: "bundle", Format: runtimecontract.FormatTensorRT, Task: runtimecontract.TaskUnknown,
		Capabilities: []string{}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: false,
		RecommendedBackends: []string{}, Confidence: "high",
		UnsupportedReason: "当前平台尚未配置 TensorRT-LLM 后端。",
	}
	onnxDefaults = ModelTypeDefaults{
		Kind: "file", Format: runtimecontract.FormatONNX, Task: runtimecontract.TaskUnknown,
		Capabilities: []string{}, DefaultTestMode: runtimecontract.TestModeAuto,
		Deployable: false, RequiresBaseModel: false,
		RecommendedBackends: []string{}, Confidence: "high",
		UnsupportedReason: "当前平台尚未配置 ONNX Runtime 后端。",
	}

	ggufDefaults = ModelTypeDefaults{
		Kind: "file", Format: runtimecontract.FormatGGUF, Task: runtimecontract.TaskChat,
		Capabilities: []string{runtimecontract.CapabilityChat, runtimecontract.CapabilityCompletion}, DefaultTestMode: runtimecontract.TestModeChat,
		Deployable: true, RequiresBaseModel: false,
		RecommendedBackends: []string{"llamacpp"}, Confidence: "high",
	}
)

// ── FileFacts construction ──

func collectFileFacts(absPath string) FileFacts {
	facts := FileFacts{AbsPath: absPath}
	info, err := os.Stat(absPath)
	if err != nil {
		return facts
	}
	facts.IsDirectory = info.IsDir()
	if !facts.IsDirectory {
		return facts
	}

	entries, _ := os.ReadDir(absPath)
	for _, e := range entries {
		facts.DirEntries = append(facts.DirEntries, e.Name())
	}

	// Parse key JSON files
	facts.ConfigJSON, _ = readJSONFile(filepath.Join(absPath, "config.json"))
	facts.GenerationConfigJSON, _ = readJSONFile(filepath.Join(absPath, "generation_config.json"))
	facts.TokenizerConfigJSON, _ = readJSONFile(filepath.Join(absPath, "tokenizer_config.json"))
	facts.SentenceBertConfigJSON, _ = readJSONFile(filepath.Join(absPath, "sentence_bert_config.json"))
	facts.AdapterConfigJSON, _ = readJSONFile(filepath.Join(absPath, "adapter_config.json"))
	facts.ModelIndexJSON, _ = readJSONFile(filepath.Join(absPath, "model_index.json"))

	facts.HasConfigJSON = facts.ConfigJSON != nil
	facts.HasTokenizerConfigJSON = fileExists(filepath.Join(absPath, "tokenizer_config.json"))
	facts.HasGenerationConfigJSON = fileExists(filepath.Join(absPath, "generation_config.json"))
	facts.HasSentenceBertConfigJSON = facts.SentenceBertConfigJSON != nil || fileExists(filepath.Join(absPath, "config_sentence_transformers.json"))
	facts.HasModulesJSON = fileExists(filepath.Join(absPath, "modules.json"))
	facts.HasOnePooling = fileExists(filepath.Join(absPath, "1_Pooling/config.json"))
	facts.HasAdapterConfigJSON = facts.AdapterConfigJSON != nil
	facts.HasAdapterModelSafetensors = fileExists(filepath.Join(absPath, "adapter_model.safetensors"))
	facts.HasPreprocessorConfigJSON = fileExists(filepath.Join(absPath, "preprocessor_config.json"))
	facts.HasImageProcessorConfigJSON = fileExists(filepath.Join(absPath, "image_processor_config.json"))
	facts.HasModelIndexJSON = facts.ModelIndexJSON != nil

	// Glob files
	facts.GGUFFiles, _ = filepath.Glob(filepath.Join(absPath, "*.gguf"))
	facts.ONNXFiles, _ = filepath.Glob(filepath.Join(absPath, "*.onnx"))
	facts.EngineFiles, _ = filepath.Glob(filepath.Join(absPath, "*.engine"))
	facts.SafetensorFiles, _ = filepath.Glob(filepath.Join(absPath, "*.safetensors"))
	facts.XMLFiles, _ = filepath.Glob(filepath.Join(absPath, "*.xml"))
	facts.BinFiles, _ = filepath.Glob(filepath.Join(absPath, "*.bin"))
	facts.PTModelFiles, _ = filepath.Glob(filepath.Join(absPath, "pytorch_model.bin"))

	// Collect evidence files
	for _, f := range entries {
		n := f.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		switch n {
		case "config.json", "tokenizer_config.json", "tokenizer.json", "generation_config.json",
			"modules.json", "sentence_bert_config.json", "config_sentence_transformers.json",
			"adapter_config.json", "adapter_model.safetensors", "preprocessor_config.json",
			"image_processor_config.json", "model_index.json":
			facts.EvidenceFiles = append(facts.EvidenceFiles, n)
		case "1_Pooling":
			facts.EvidenceFiles = append(facts.EvidenceFiles, "1_Pooling/config.json")
		}
	}
	// Add glob-based evidence
	if len(facts.GGUFFiles) > 0 {
		facts.EvidenceFiles = append(facts.EvidenceFiles, "*.gguf")
	}
	if len(facts.ONNXFiles) > 0 {
		facts.EvidenceFiles = append(facts.EvidenceFiles, "*.onnx")
	}
	if len(facts.EngineFiles) > 0 {
		facts.EvidenceFiles = append(facts.EvidenceFiles, "*.engine")
	}

	return facts
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func readJSONFile(p string) (map[string]interface{}, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// nameMatches checks whether the model name (from path or any related name field)
// contains any of the given keywords (case-insensitive).
func nameMatches(path string, keywords ...string) bool {
	lower := strings.ToLower(path)
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// ── Detectors ──

// DetectHuggingFaceChat detects standard HF chat/completion directories.
func DetectHuggingFaceChat(facts FileFacts) []ScanCandidate {
	if !facts.HasConfigJSON {
		return nil
	}
	c := ScanCandidate{
		Path: facts.AbsPath, PathType: "directory",
		DetectedMetadata: make(map[string]interface{}),
	}
	c.Evidence = append(c.Evidence, "config.json")
	if facts.HasTokenizerConfigJSON {
		c.Evidence = append(c.Evidence, "tokenizer_config.json")
	}
	if facts.HasGenerationConfigJSON {
		c.Evidence = append(c.Evidence, "generation_config.json")
	}
	if len(facts.SafetensorFiles) > 0 {
		c.Evidence = append(c.Evidence, "*.safetensors")
	}
	extractHFMetadata(facts.AbsPath, facts.ConfigJSON, &c)
	return []ScanCandidate{c}
}

// DetectGGUF detects GGUF file candidates.
func DetectGGUF(facts FileFacts) []ScanCandidate {
	var candidates []ScanCandidate
	for _, ggufPath := range facts.GGUFFiles {
		c := ScanCandidate{
			Path: ggufPath, PathType: "file",
			DetectedMetadata: make(map[string]interface{}),
			Evidence:         []string{"*.gguf"},
		}
		if fi, err := os.Stat(ggufPath); err == nil {
			c.SizeBytes = fi.Size()
			c.SizeLabel = FormatBytes(fi.Size())
			c.DetectedMetadata["file_size_bytes"] = float64(fi.Size())
		}
		c.DetectedMetadata["format"] = "gguf"
		extractGGUFMetadata(ggufPath, &c)
		candidates = append(candidates, c)
	}
	return candidates
}

// DetectSentenceTransformers detects SentenceTransformers / Embedding models.
func DetectSentenceTransformers(facts FileFacts) []ScanCandidate {
	hasSTStructure := facts.HasModulesJSON && facts.HasOnePooling
	hasSTConfig := facts.HasSentenceBertConfigJSON || facts.SentenceBertConfigJSON != nil
	hasSTName := nameMatches(facts.AbsPath,
		"bge", "e5", "gte", "embedding", "embeddings",
		"sentence-transformers", "text2vec", "m3e", "jina-embeddings",
		"stella", "multilingual-e5", "bce-embedding",
	)
	// Exclude reranker names that happen to match embedding keywords ("bge")
	hasRerankerName := nameMatches(facts.AbsPath,
		"reranker", "rerank", "cross-encoder", "cross_encoder", "ranker",
	)
	if hasRerankerName {
		hasSTName = false
	}

	if !hasSTStructure && !hasSTConfig && !hasSTName {
		return nil
	}

	c := ScanCandidate{
		Path: facts.AbsPath, PathType: "directory",
		DetectedMetadata: make(map[string]interface{}),
	}
	if facts.HasModulesJSON {
		c.Evidence = append(c.Evidence, "modules.json")
	}
	if facts.HasOnePooling {
		c.Evidence = append(c.Evidence, "1_Pooling/config.json")
	}
	if facts.HasSentenceBertConfigJSON {
		c.Evidence = append(c.Evidence, "config_sentence_transformers.json")
	}
	if facts.SentenceBertConfigJSON != nil {
		c.Evidence = append(c.Evidence, "sentence_bert_config.json")
	}

	// If there's a config.json, extract HF metadata for size/architecture info
	if facts.ConfigJSON != nil {
		extractHFMetadata(facts.AbsPath, facts.ConfigJSON, &c)
	}
	return []ScanCandidate{c}
}

// DetectReranker detects Reranker / CrossEncoder models.
func DetectReranker(facts FileFacts) []ScanCandidate {
	hasRerankerName := nameMatches(facts.AbsPath,
		"reranker", "rerank", "cross-encoder", "cross_encoder",
		"ranker", "bge-reranker", "jina-reranker", "ms-marco",
		"bce-reranker",
	)

	if !hasRerankerName {
		return nil
	}
	// Require at least some HF directory evidence to avoid false positives
	if !facts.HasConfigJSON && len(facts.SafetensorFiles) == 0 && len(facts.PTModelFiles) == 0 {
		return nil
	}

	c := ScanCandidate{
		Path: facts.AbsPath, PathType: "directory",
		DetectedMetadata: make(map[string]interface{}),
	}
	c.Evidence = append(c.Evidence, "name contains reranker/cross-encoder")
	if facts.HasConfigJSON {
		c.Evidence = append(c.Evidence, "config.json")
	}
	if len(facts.SafetensorFiles) > 0 {
		c.Evidence = append(c.Evidence, "*.safetensors")
	}
	if facts.ConfigJSON != nil {
		extractHFMetadata(facts.AbsPath, facts.ConfigJSON, &c)
	}
	return []ScanCandidate{c}
}

// DetectVisionLanguage detects Vision-Language / Multimodal models.
func DetectVisionLanguage(facts FileFacts) []ScanCandidate {
	hasVLName := nameMatches(facts.AbsPath,
		"internvl", "qwen-vl", "qwen2-vl", "qwen2.5-vl",
		"llava", "minicpm-v", "glm-4v", "cogvlm", "phi-3-vision",
		"phi-3.5-vision", "paligemma", "fuyu", "idefics",
	)
	hasVLIndicator := fileExists(filepath.Join(facts.AbsPath, "configuration_internvl_chat.py")) ||
		fileExists(filepath.Join(facts.AbsPath, "configuration_intern_vit.py")) ||
		facts.HasImageProcessorConfigJSON ||
		facts.HasPreprocessorConfigJSON

	if !hasVLName && !hasVLIndicator {
		return nil
	}
	if !facts.HasConfigJSON {
		return nil
	}

	c := ScanCandidate{
		Path: facts.AbsPath, PathType: "directory",
		DetectedMetadata: make(map[string]interface{}),
	}
	c.Evidence = append(c.Evidence, "name contains vision-language model pattern")
	if fileExists(filepath.Join(facts.AbsPath, "configuration_internvl_chat.py")) {
		c.Evidence = append(c.Evidence, "configuration_internvl_chat.py")
	}
	if fileExists(filepath.Join(facts.AbsPath, "configuration_intern_vit.py")) {
		c.Evidence = append(c.Evidence, "configuration_intern_vit.py")
	}
	if facts.HasImageProcessorConfigJSON {
		c.Evidence = append(c.Evidence, "image_processor_config.json")
	}
	if facts.HasPreprocessorConfigJSON {
		c.Evidence = append(c.Evidence, "preprocessor_config.json")
	}
	if facts.ConfigJSON != nil {
		extractHFMetadata(facts.AbsPath, facts.ConfigJSON, &c)
	}
	return []ScanCandidate{c}
}

// DetectLoRAAdapter detects LoRA/Adapter models.
func DetectLoRAAdapter(facts FileFacts) []ScanCandidate {
	if !facts.HasAdapterConfigJSON && !facts.HasAdapterModelSafetensors {
		return nil
	}
	c := ScanCandidate{
		Path: facts.AbsPath, PathType: "directory",
		DetectedMetadata: make(map[string]interface{}),
	}
	if facts.HasAdapterConfigJSON {
		c.Evidence = append(c.Evidence, "adapter_config.json")
	}
	if facts.HasAdapterModelSafetensors {
		c.Evidence = append(c.Evidence, "adapter_model.safetensors")
	}
	// Extract adapter-specific metadata if available
	if facts.AdapterConfigJSON != nil {
		if baseModel, ok := facts.AdapterConfigJSON["base_model_name_or_path"]; ok {
			c.DetectedMetadata["base_model_name_or_path"] = baseModel
		}
	}
	return []ScanCandidate{c}
}

// DetectDiffusers detects Diffusers / Image Generation models.
func DetectDiffusers(facts FileFacts) []ScanCandidate {
	if !facts.HasModelIndexJSON && !fileExists(filepath.Join(facts.AbsPath, "unet")) {
		return nil
	}
	c := ScanCandidate{Path: facts.AbsPath, PathType: "directory", DetectedMetadata: make(map[string]interface{})}
	if facts.HasModelIndexJSON {
		c.Evidence = append(c.Evidence, "model_index.json")
	}
	if fileExists(filepath.Join(facts.AbsPath, "unet")) {
		c.Evidence = append(c.Evidence, "unet/")
	}
	return []ScanCandidate{c}
}

// DetectASR detects Automatic Speech Recognition models.
func DetectASR(facts FileFacts) []ScanCandidate {
	if !nameMatches(facts.AbsPath, "whisper", "funasr", "paraformer", "sensevoice") {
		return nil
	}
	if !facts.HasConfigJSON {
		return nil
	}
	c := ScanCandidate{Path: facts.AbsPath, PathType: "directory", DetectedMetadata: make(map[string]interface{})}
	c.Evidence = append(c.Evidence, "name contains asr model pattern", "config.json")
	return []ScanCandidate{c}
}

// DetectTTS detects Text-to-Speech models.
func DetectTTS(facts FileFacts) []ScanCandidate {
	if !nameMatches(facts.AbsPath, "cosyvoice", "chattts", "gpt-sovits", "fish-speech", "bark") {
		return nil
	}
	if !facts.HasConfigJSON {
		return nil
	}
	c := ScanCandidate{Path: facts.AbsPath, PathType: "directory", DetectedMetadata: make(map[string]interface{})}
	c.Evidence = append(c.Evidence, "name contains tts model pattern", "config.json")
	return []ScanCandidate{c}
}

// DetectClassification detects classification / token classification models.
func DetectClassification(facts FileFacts) []ScanCandidate {
	if facts.ConfigJSON == nil {
		return nil
	}
	architectures, _ := facts.ConfigJSON["architectures"].([]interface{})
	hasClassArch := false
	for _, a := range architectures {
		if s, ok := a.(string); ok {
			if strings.Contains(s, "SequenceClassification") || strings.Contains(s, "TokenClassification") || strings.Contains(s, "ImageClassification") || strings.Contains(s, "AudioClassification") {
				hasClassArch = true
				break
			}
		}
	}
	if !hasClassArch {
		return nil
	}
	c := ScanCandidate{Path: facts.AbsPath, PathType: "directory", DetectedMetadata: make(map[string]interface{})}
	c.Evidence = append(c.Evidence, "config.json: classification architecture")
	return []ScanCandidate{c}
}

// DetectOpenVINO detects OpenVINO model bundles.
func DetectOpenVINO(facts FileFacts) []ScanCandidate {
	if len(facts.XMLFiles) == 0 || len(facts.BinFiles) == 0 {
		return nil
	}
	c := ScanCandidate{Path: facts.AbsPath, PathType: "directory", DetectedMetadata: make(map[string]interface{})}
	c.Evidence = append(c.Evidence, "*.xml", "*.bin")
	return []ScanCandidate{c}
}

// DetectTensorRT detects TensorRT/TensorRT-LLM engine files.
func DetectTensorRT(facts FileFacts) []ScanCandidate {
	if len(facts.EngineFiles) == 0 {
		return nil
	}
	c := ScanCandidate{Path: facts.AbsPath, PathType: "directory", DetectedMetadata: make(map[string]interface{})}
	c.Evidence = append(c.Evidence, "*.engine")
	return []ScanCandidate{c}
}

// DetectONNX detects ONNX model files.
func DetectONNX(facts FileFacts) []ScanCandidate {
	if len(facts.ONNXFiles) == 0 {
		return nil
	}
	var candidates []ScanCandidate
	for _, onnxPath := range facts.ONNXFiles {
		c := ScanCandidate{Path: onnxPath, PathType: "file", DetectedMetadata: make(map[string]interface{})}
		c.Evidence = append(c.Evidence, "*.onnx")
		candidates = append(candidates, c)
	}
	return candidates
}

// ── applyDefaults fills unset fields from plugin defaults ──

func applyDefaults(c *ScanCandidate, d ModelTypeDefaults) {
	if c.Kind == "" {
		c.Kind = d.Kind
	}
	if c.Format == "" {
		c.Format = d.Format
	}
	if c.Task == "" {
		c.Task = d.Task
	}
	if len(c.Capabilities) == 0 {
		c.Capabilities = d.Capabilities
	}
	if c.DefaultTestMode == "" {
		c.DefaultTestMode = d.DefaultTestMode
	}
	// Deployable/RequiresBaseModel: plugin defaults always win for these flags.
	c.Deployable = d.Deployable
	c.RequiresBaseModel = d.RequiresBaseModel
	if len(c.RecommendedBackends) == 0 {
		c.RecommendedBackends = d.RecommendedBackends
	}
	if c.Confidence == "" {
		c.Confidence = d.Confidence
	}
	if c.UnsupportedReason == "" {
		c.UnsupportedReason = d.UnsupportedReason
	}
}

// ── Public API ──

// ScanModelPath scans a directory or file for model candidates.
func ScanModelPath(root, relPath string) map[string]interface{} {
	absPath := filepath.Join(root, relPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("cannot access path: %v", err),
		}
	}

	if info.IsDir() {
		return scanDirectory(absPath)
	}

	return scanSingleFile(absPath, root, relPath, info)
}

// scanDirectory uses the plugin registry to detect model types.
func scanDirectory(absPath string) map[string]interface{} {
	facts := collectFileFacts(absPath)
	result := &ScanResult{ScanRoot: absPath}

	var candidates []ScanCandidate
	for _, plugin := range modelTypePlugins {
		detected := plugin.Detect(facts)
		for i := range detected {
			applyDefaults(&detected[i], plugin.Defaults)
			if len(detected[i].Evidence) == 0 {
				detected[i].Evidence = facts.EvidenceFiles
			}
		}
		candidates = append(candidates, detected...)
	}

	// De-duplication: suppress HF Chat when a more specific plugin matched.
	// Higher-priority plugins (LoRA, ST, Reranker, VLM) are more specific;
	// HF Chat is the catch-all that fires whenever config.json exists.
	// When a more specific plugin found a match for the same directory,
	// remove the generic HF Chat candidate to avoid confusion.
	hasSpecificMatch := false
	for _, c := range candidates {
		if c.Format == "sentence_transformers" || c.Format == "lora_adapter" ||
			c.Format == "diffusers" || c.Format == "openvino" ||
			c.Format == "tensorrt_engine" || c.Format == "onnx" ||
			c.Task == "rerank" || c.Task == "vision_chat" ||
			c.Task == "image_generation" || c.Task == "asr" ||
			c.Task == "tts" || c.Task == "classification" {
			hasSpecificMatch = true
			break
		}
	}
	if hasSpecificMatch {
		filtered := candidates[:0]
		for _, c := range candidates {
			if c.Format == "huggingface" && c.Task == "chat" {
				continue
			}
			filtered = append(filtered, c)
		}
		candidates = filtered
	}
	result.Candidates = candidates

	// ── Auto-selection logic ──
	totalCandidates := len(result.Candidates)
	if totalCandidates == 1 {
		result.Candidates[0].AutoSelected = true
		result.Candidates[0].SelectionReason = fmt.Sprintf("single %s candidate in directory", result.Candidates[0].Format)
	} else if totalCandidates > 1 {
		allGGUF := true
		for _, c := range result.Candidates {
			if c.Format != "gguf" {
				allGGUF = false
				break
			}
		}
		if allGGUF {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("found %d GGUF files, user must select one", totalCandidates))
		} else {
			result.Warnings = append(result.Warnings,
				"mixed model types found, user must select one")
		}
	}

	if totalCandidates == 0 {
		result.Error = "no recognizable model files found in directory"
	}

	return toMap(result)
}

// scanSingleFile scans a single file for model metadata.
func scanSingleFile(absPath, root, relPath string, info os.FileInfo) map[string]interface{} {
	result := &ScanResult{ScanRoot: absPath}

	c := ScanCandidate{
		Path: absPath, PathType: "file",
		DetectedMetadata: make(map[string]interface{}),
		AutoSelected:     true,
		SelectionReason:  "single file selected",
	}

	c.SizeBytes = info.Size()
	c.SizeLabel = FormatBytes(info.Size())
	c.DetectedMetadata["file_size_bytes"] = float64(info.Size())

	if strings.HasSuffix(strings.ToLower(relPath), ".gguf") {
		// Apply GGUF plugin defaults
		applyDefaults(&c, ggufDefaults)
		c.DetectedMetadata["format"] = "gguf"
		extractGGUFMetadata(absPath, &c)
		c.Evidence = append(c.Evidence, "*.gguf")
	} else {
		c.Format = "custom"
		c.Task = "unknown"
		c.DetectedMetadata["format"] = "custom"
		c.Evidence = append(c.Evidence, fmt.Sprintf("file extension: %s", filepath.Ext(relPath)))
	}

	result.Candidates = append(result.Candidates, c)
	return toMap(result)
}

// ── Metadata extraction (unchanged from original) ──

func extractGGUFMetadata(path string, candidate *ScanCandidate) {
	meta, err := readGGUFMeta(path)
	if err != nil {
		candidate.Warnings = append(candidate.Warnings,
			fmt.Sprintf("GGUF metadata read failed: %v", err))
		if candidate.DetectedMetadata["quantization"] == nil {
			candidate.DetectedMetadata["quantization"] = guessQuantFromFilename(path)
		}
		return
	}

	candidate.DetectedMetadata["architecture"] = meta.Architecture
	if meta.ContextLength > 0 {
		candidate.DetectedMetadata["context_length"] = float64(meta.ContextLength)
	}
	if meta.EmbeddingLength > 0 {
		candidate.DetectedMetadata["embedding_length"] = float64(meta.EmbeddingLength)
	}
	if meta.BlockCount > 0 {
		candidate.DetectedMetadata["block_count"] = float64(meta.BlockCount)
	}
	if meta.VocabSize > 0 {
		candidate.DetectedMetadata["vocab_size"] = float64(meta.VocabSize)
	}
	if meta.HeadCount > 0 {
		candidate.DetectedMetadata["head_count"] = float64(meta.HeadCount)
	}
	if meta.HeadCountKV > 0 {
		candidate.DetectedMetadata["head_count_kv"] = float64(meta.HeadCountKV)
	}
	candidate.DetectedMetadata["file_size_bytes"] = float64(meta.FileSizeBytes)

	q := meta.Quantization
	if q == "" || q == "unknown" {
		q = guessQuantFromFilename(path)
	}
	if q != "" && q != "unknown" {
		candidate.DetectedMetadata["quantization"] = q
	}

	candidate.Warnings = append(candidate.Warnings, meta.Warnings...)
}

func extractHFMetadata(absPath string, config map[string]interface{}, candidate *ScanCandidate) {
	candidate.DetectedMetadata["format"] = "huggingface"

	if arch, ok := config["architectures"]; ok {
		candidate.DetectedMetadata["architectures"] = arch
		if archList, ok := arch.([]interface{}); ok && len(archList) > 0 {
			candidate.DetectedMetadata["architecture"] = archList[0]
		}
	}
	if mt, ok := config["model_type"]; ok {
		candidate.DetectedMetadata["model_type"] = mt
	}
	if dt, ok := config["torch_dtype"]; ok {
		candidate.DetectedMetadata["torch_dtype"] = dt
	}
	if mpe, ok := config["max_position_embeddings"]; ok {
		candidate.DetectedMetadata["max_position_embeddings"] = mpe
		switch v := mpe.(type) {
		case float64:
			candidate.DetectedMetadata["context_length"] = v
		}
	}
	if hs, ok := config["hidden_size"]; ok {
		candidate.DetectedMetadata["hidden_size"] = hs
	}
	if nhl, ok := config["num_hidden_layers"]; ok {
		candidate.DetectedMetadata["num_hidden_layers"] = nhl
	}
	if nah, ok := config["num_attention_heads"]; ok {
		candidate.DetectedMetadata["num_attention_heads"] = nah
	}
	if nkvh, ok := config["num_key_value_heads"]; ok {
		candidate.DetectedMetadata["num_key_value_heads"] = nkvh
	}
	if vs, ok := config["vocab_size"]; ok {
		candidate.DetectedMetadata["vocab_size"] = vs
	}
	if rs, ok := config["rope_scaling"]; ok {
		candidate.DetectedMetadata["rope_scaling"] = rs
	}
	if qc, ok := config["quantization_config"]; ok {
		candidate.DetectedMetadata["quantization_config"] = qc
	}

	safePattern := filepath.Join(absPath, "*.safetensors")
	if matches, _ := filepath.Glob(safePattern); len(matches) > 0 {
		var totalSize int64
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil {
				totalSize += fi.Size()
			}
		}
		candidate.DetectedMetadata["safetensors_count"] = float64(len(matches))
		candidate.SizeBytes = totalSize
		candidate.SizeLabel = FormatBytes(totalSize)
		candidate.DetectedMetadata["file_size_bytes"] = float64(totalSize)
	} else {
		var totalSize int64
		filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})
		candidate.SizeBytes = totalSize
		candidate.SizeLabel = FormatBytes(totalSize)
		candidate.DetectedMetadata["file_size_bytes"] = float64(totalSize)
	}

	if _, err := os.Stat(filepath.Join(absPath, "tokenizer_config.json")); err == nil {
		candidate.DetectedMetadata["has_tokenizer"] = true
	}

	if data, err := os.ReadFile(filepath.Join(absPath, "generation_config.json")); err == nil {
		var genConfig map[string]interface{}
		if json.Unmarshal(data, &genConfig) == nil {
			if ml, ok := genConfig["max_length"]; ok {
				candidate.DetectedMetadata["generation_max_length"] = ml
			}
		}
	}

	if candidate.DetectedMetadata["max_position_embeddings"] == nil {
		candidate.Warnings = append(candidate.Warnings, "max_position_embeddings not found in config.json")
	}
	if candidate.DetectedMetadata["architectures"] == nil {
		candidate.Warnings = append(candidate.Warnings, "architectures not found in config.json")
	}
}

// ── Utilities ──

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func strsToInterfaces(strs []string) []interface{} {
	if strs == nil {
		return []interface{}{}
	}
	out := make([]interface{}, len(strs))
	for i, s := range strs {
		out[i] = s
	}
	return out
}

func toMap(sr *ScanResult) map[string]interface{} {
	if sr.Error != "" {
		return map[string]interface{}{
			"scan_root":  sr.ScanRoot,
			"error":      sr.Error,
			"candidates": []interface{}{},
			"warnings":   sr.Warnings,
		}
	}
	candidates := make([]interface{}, len(sr.Candidates))
	for i, c := range sr.Candidates {
		candidates[i] = candidateToMap(c)
	}
	return map[string]interface{}{
		"scan_root":  sr.ScanRoot,
		"candidates": candidates,
		"warnings":   sr.Warnings,
	}
}

func candidateToMap(c ScanCandidate) map[string]interface{} {
	return map[string]interface{}{
		"path":                 c.Path,
		"path_type":            c.PathType,
		"kind":                 c.Kind,
		"format":               c.Format,
		"task":                 c.Task,
		"capabilities":         strsToInterfaces(c.Capabilities),
		"default_test_mode":    c.DefaultTestMode,
		"deployable":           c.Deployable,
		"requires_base_model":  c.RequiresBaseModel,
		"recommended_backends": c.RecommendedBackends,
		"confidence":           c.Confidence,
		"evidence":             strsToInterfaces(c.Evidence),
		"unsupported_reason":   c.UnsupportedReason,
		"detected_metadata":    c.DetectedMetadata,
		"warnings":             c.Warnings,
		"auto_selected":        c.AutoSelected,
		"selection_reason":     c.SelectionReason,
		"size_bytes":           c.SizeBytes,
		"size_label":           c.SizeLabel,
	}
}
