package collector

import (
	"os"
	"path/filepath"
	"testing"
)

// makeTempDir creates a temporary directory structure and returns the root path.
// cleanup is registered on t.Cleanup.
func makeTempDir(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	for relPath, content := range files {
		fullPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	return root
}

func hasCandidateWith(t *testing.T, candidates []ScanCandidate, task, format string) bool {
	t.Helper()
	for _, c := range candidates {
		if c.Task == task && c.Format == format {
			return true
		}
	}
	return false
}

// TestDetectHuggingFaceChat verifies HF chat directory detection through full pipeline.
func TestDetectHuggingFaceChat(t *testing.T) {
	root := makeTempDir(t, "Qwen3-0.6B-Instruct-2512", map[string]string{
		"config.json":            `{"architectures":["Qwen3ForCausalLM"],"model_type":"qwen3","torch_dtype":"bfloat16","max_position_embeddings":32768}`,
		"tokenizer_config.json":  `{}`,
		"generation_config.json": `{}`,
		"model.safetensors":      "\x00\x00\x00\x01",
	})
	result := ScanModelPath(root, ".")
	candidatesRaw, ok := result["candidates"].([]interface{})
	if !ok || len(candidatesRaw) == 0 {
		t.Fatal("expected at least 1 candidate")
	}
	cmap := candidatesRaw[0].(map[string]interface{})
	if cmap["path_type"] != "directory" {
		t.Errorf("expected path_type=directory, got %v", cmap["path_type"])
	}
	if cmap["task"] != "chat" {
		t.Errorf("expected task=chat, got %v", cmap["task"])
	}
	if cmap["deployable"] != true {
		t.Errorf("expected deployable=true, got %v", cmap["deployable"])
	}
}

// TestDetectGGUFFile verifies GGUF file detection through full pipeline.
func TestDetectGGUFFile(t *testing.T) {
	root := makeTempDir(t, "qwen35-q4", map[string]string{
		"Qwen3.5-9B-Q4_K_M.gguf": "GGUF__MINIMAL_GGUF_MAGIC_PLACEHOLDER",
	})
	result := ScanModelPath(root, "Qwen3.5-9B-Q4_K_M.gguf")
	candidatesRaw, ok := result["candidates"].([]interface{})
	if !ok || len(candidatesRaw) == 0 {
		t.Fatal("expected at least 1 GGUF candidate from scanSingleFile")
	}
	cmap := candidatesRaw[0].(map[string]interface{})
	if cmap["path_type"] != "file" {
		t.Errorf("expected path_type=file, got %v", cmap["path_type"])
	}
	if cmap["format"] != "gguf" {
		t.Errorf("expected format=gguf, got %v", cmap["format"])
	}
	if cmap["deployable"] != true {
		t.Errorf("expected deployable=true, got %v", cmap["deployable"])
	}
}

// TestDetectSentenceTransformers verifies embedding model detection.
func TestDetectSentenceTransformers(t *testing.T) {
	root := makeTempDir(t, "bge-small-zh-v1.5", map[string]string{
		"modules.json":                      `[{"type":"sentence_transformers"}]`,
		"1_Pooling/config.json":             `{}`,
		"config_sentence_transformers.json": `{}`,
		"config.json":                       `{"architectures":["BertModel"]}`,
	})
	facts := collectFileFacts(root)
	candidates := DetectSentenceTransformers(facts)
	if len(candidates) == 0 {
		t.Fatal("expected at least 1 SentenceTransformers candidate")
	}
	c := candidates[0]
	if c.PathType != "directory" {
		t.Errorf("expected path_type=directory, got %q", c.PathType)
	}
}

// TestDetectReranker verifies reranker model detection.
func TestDetectReranker(t *testing.T) {
	root := makeTempDir(t, "bge-reranker-base", map[string]string{
		"config.json": `{"architectures":["BertForSequenceClassification"]}`,
	})
	facts := collectFileFacts(root)
	candidates := DetectReranker(facts)
	if len(candidates) == 0 {
		t.Fatal("expected at least 1 reranker candidate (name contains 'reranker')")
	}
	c := candidates[0]
	if c.PathType != "directory" {
		t.Errorf("expected path_type=directory, got %q", c.PathType)
	}
}

// TestDetectVisionLanguage verifies VL model detection via name pattern.
func TestDetectVisionLanguage(t *testing.T) {
	root := makeTempDir(t, "InternVL2_5-1B", map[string]string{
		"config.json":                    `{"architectures":["InternVLChatModel"]}`,
		"configuration_internvl_chat.py": "",
		"configuration_intern_vit.py":    "",
	})
	facts := collectFileFacts(root)
	candidates := DetectVisionLanguage(facts)
	if len(candidates) == 0 {
		t.Fatal("expected at least 1 VL candidate (name contains 'internvl')")
	}
	c := candidates[0]
	if c.PathType != "directory" {
		t.Errorf("expected path_type=directory, got %q", c.PathType)
	}
}

// TestDetectLoRAAdapter verifies LoRA/adapter detection.
func TestDetectLoRAAdapter(t *testing.T) {
	root := makeTempDir(t, "lora-adapter", map[string]string{
		"adapter_config.json":       `{"base_model_name_or_path":"Qwen/Qwen3-0.6B"}`,
		"adapter_model.safetensors": "",
	})
	facts := collectFileFacts(root)
	candidates := DetectLoRAAdapter(facts)
	if len(candidates) == 0 {
		t.Fatal("expected at least 1 LoRA candidate")
	}
	c := candidates[0]
	if c.PathType != "directory" {
		t.Errorf("expected path_type=directory, got %q", c.PathType)
	}
}

// TestEmptyDirectory verifies empty directories return an error via scanDirectory.
func TestEmptyDirectory(t *testing.T) {
	root := makeTempDir(t, "empty-dir", map[string]string{})
	result := ScanModelPath(root, ".")
	errStr, _ := result["error"].(string)
	if errStr == "" || errStr != "no recognizable model files found in directory" {
		t.Errorf("expected 'no recognizable model files found in directory' error, got %q", errStr)
	}
}

// TestMixedHFAndGGUF verifies both HF and GGUF candidates appear when directory has both.
func TestMixedHFAndGGUF(t *testing.T) {
	root := makeTempDir(t, "mixed-model", map[string]string{
		"config.json":       `{"architectures":["Qwen3ForCausalLM"]}`,
		"model-q4_k_m.gguf": "GGUF_MINIMAL_DATA",
		"model-q5_k_m.gguf": "GGUF_MINIMAL_DATA",
	})
	result := ScanModelPath(root, ".")
	candidatesRaw, ok := result["candidates"].([]interface{})
	if !ok {
		t.Fatal("expected candidates in result")
	}
	if len(candidatesRaw) < 3 {
		t.Errorf("expected at least 3 candidates (1 HF + 2 GGUF), got %d", len(candidatesRaw))
	}
	candidates := make([]ScanCandidate, len(candidatesRaw))
	// We don't need full deserialization; just check the map.
	var hasHF, ggufCount int
	for _, raw := range candidatesRaw {
		cmap := raw.(map[string]interface{})
		fmt, _ := cmap["format"].(string)
		if fmt == "huggingface" {
			hasHF++
		}
		if fmt == "gguf" {
			ggufCount++
		}
	}
	if hasHF == 0 {
		t.Error("expected HF chat candidate in mixed directory")
	}
	if ggufCount != 2 {
		t.Errorf("expected 2 GGUF candidates, got %d", ggufCount)
	}
	_ = candidates
}

// TestScanDirectoryFullPipeline verifies the full scanDirectory pipeline applies defaults and auto-selection.
func TestScanDirectoryFullPipeline(t *testing.T) {
	root := makeTempDir(t, "Qwen3-0.6B-Instruct-2512", map[string]string{
		"config.json":           `{"architectures":["Qwen3ForCausalLM"],"model_type":"qwen3","max_position_embeddings":32768}`,
		"tokenizer_config.json": `{}`,
		"model.safetensors":     "\x00\x01",
	})
	result := ScanModelPath(root, ".")
	candidatesRaw, ok := result["candidates"].([]interface{})
	if !ok || len(candidatesRaw) == 0 {
		t.Fatal("expected at least 1 candidate from full pipeline")
	}
	cmap := candidatesRaw[0].(map[string]interface{})
	if cmap["kind"] != "directory" {
		t.Errorf("expected kind=directory after applyDefaults, got %v", cmap["kind"])
	}
	if cmap["task"] != "chat" {
		t.Errorf("expected task=chat after applyDefaults, got %v", cmap["task"])
	}
	if cmap["deployable"] != true {
		t.Errorf("expected deployable=true, got %v", cmap["deployable"])
	}
	caps, _ := cmap["capabilities"].([]interface{})
	if len(caps) == 0 {
		t.Errorf("expected non-empty capabilities after applyDefaults, got %v", cmap["capabilities"])
	}
}
