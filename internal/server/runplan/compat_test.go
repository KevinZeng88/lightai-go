package runplan

import "testing"

func assertCompatFail(t *testing.T, name string, result CompatResult) {
	t.Helper()
	if result.Compatible {
		t.Errorf("%s: expected FAIL but got PASS (code=%s reason=%s)", name, result.Code, result.Reason)
	} else {
		t.Logf("%s: PASS (blocked: %s — %s)", name, result.Code, result.Reason)
	}
}

func assertCompatPass(t *testing.T, name string, result CompatResult) {
	t.Helper()
	if !result.Compatible {
		t.Errorf("%s: expected PASS but got FAIL (code=%s reason=%s)", name, result.Code, result.Reason)
	} else {
		t.Logf("%s: PASS", name)
	}
}

func TestCompatVLLMWithGGUFFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "gguf", Task: "chat", Deployable: true, PathType: "file"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatFail(t, "vLLM+GGUF", result)
}

func TestCompatSGLangWithGGUFFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "gguf", Task: "chat", Deployable: true, PathType: "file"},
		BackendDescriptor{BackendName: "sglang", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatFail(t, "SGLang+GGUF", result)
}

func TestCompatLlamaCppWithHFFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "chat", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "llamacpp", SupportedFormats: []string{"gguf"}, SupportedTasks: []string{"chat"}, ModelPathModes: []string{"file"}},
	)
	assertCompatFail(t, "llama.cpp+HF", result)
}

func TestCompatLlamaCppWithEmbeddingFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "sentence_transformers", Task: "embedding", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "llamacpp", SupportedFormats: []string{"gguf"}, SupportedTasks: []string{"chat", "completion"}, ModelPathModes: []string{"file"}},
	)
	assertCompatFail(t, "llama.cpp+Embedding", result)
}

func TestCompatLlamaCppWithRerankerFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "rerank", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "llamacpp", SupportedFormats: []string{"gguf"}, SupportedTasks: []string{"chat", "completion"}, ModelPathModes: []string{"file"}},
	)
	assertCompatFail(t, "llama.cpp+Reranker", result)
}

func TestCompatLoRAStandaloneFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "lora_adapter", Task: "adapter", Deployable: false, PathType: "directory"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatFail(t, "LoRA+standalone", result)
}

func TestCompatDeployableFalseFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "onnx", Task: "unknown", Deployable: false, PathType: "file"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatFail(t, "deployable=false", result)
}

func TestCompatMissingBackendCapabilitiesFails(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "chat", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: nil, SupportedTasks: nil, ModelPathModes: nil},
	)
	assertCompatFail(t, "missing backend caps", result)
}

func TestCompatVLLMWithHFPasses(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "chat", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat", "completion", "embedding"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatPass(t, "vLLM+HF", result)
}

func TestCompatLlamaCppWithGGUFPasses(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "gguf", Task: "chat", Deployable: true, PathType: "file"},
		BackendDescriptor{BackendName: "llamacpp", SupportedFormats: []string{"gguf"}, SupportedTasks: []string{"chat", "completion"}, ModelPathModes: []string{"file"}},
	)
	assertCompatPass(t, "llama.cpp+GGUF", result)
}

func TestCompatEmbeddingWithVLLMPasses(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "sentence_transformers", Task: "embedding", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface", "sentence_transformers"}, SupportedTasks: []string{"chat", "embedding", "rerank"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatPass(t, "Embedding+vLLM", result)
}

func TestCompatRerankerWithVLLMPasses(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "rerank", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat", "rerank"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatPass(t, "Reranker+vLLM", result)
}

func TestCompatVisionWithVLLMPasses(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "vision_chat", Deployable: true, PathType: "directory"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat", "vision_chat"}, ModelPathModes: []string{"directory"}},
	)
	assertCompatPass(t, "Vision+vLLM", result)
}

// TestCompatInternVLWithVLLMBlocked verifies architecture-level blocking.
func TestCompatInternVLWithVLLMBlocked(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "vision_chat", Deployable: true, PathType: "directory", Architecture: "InternVLChatModel"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat", "vision_chat"}, ModelPathModes: []string{"directory"}, BlockedArchitectures: map[string]string{"InternVLChatModel": "vLLM cannot load InternVL tokenizer"}},
	)
	assertCompatFail(t, "InternVL+vLLM blocked", result)
	if result.Code != "architecture_blocked" {
		t.Errorf("expected code=architecture_blocked, got %s", result.Code)
	}
}

// TestCompatHFWithVLLMNoBlock verifies that HF chat models pass even with
// blocked architectures declared (architecture check only matches exact arch).
func TestCompatHFWithVLLMNoBlock(t *testing.T) {
	result := CheckCompatibility(
		ModelDescriptor{Format: "huggingface", Task: "chat", Deployable: true, PathType: "directory", Architecture: "Qwen3ForCausalLM"},
		BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat"}, ModelPathModes: []string{"directory"}, BlockedArchitectures: map[string]string{"InternVLChatModel": "vLLM cannot load InternVL tokenizer"}},
	)
	assertCompatPass(t, "HF+vLLM no block", result)
}

// TestCompatDeployableFalseFailsOnUnsupportedTypes verifies all B2 types are blocked.
func TestCompatDeployableFalseFailsOnUnsupportedTypes(t *testing.T) {
	vllmCaps := BackendDescriptor{BackendName: "vllm", SupportedFormats: []string{"huggingface"}, SupportedTasks: []string{"chat"}, ModelPathModes: []string{"directory"}}
	tests := []struct{ format, task, pathType string }{
		{"onnx", "unknown", "file"},
		{"tensorrt_engine", "unknown", "directory"},
		{"openvino", "unknown", "directory"},
		{"diffusers", "image_generation", "directory"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := CheckCompatibility(ModelDescriptor{Format: tt.format, Task: tt.task, Deployable: false, PathType: tt.pathType}, vllmCaps)
			if result.Compatible {
				t.Errorf("%s: expected FAIL (deployable=false), got PASS", tt.format)
			}
		})
	}
}

func TestParseBackendCapabilities(t *testing.T) {
	capsJSON := `{"supported_formats":["huggingface"],"supported_tasks":["chat","embedding"],"supported_capabilities":["chat","embedding"],"model_path_modes":["directory"],"test_endpoints":{"chat":"/v1/chat/completions","embedding":"/v1/embeddings"}}`
	bd, err := ParseBackendCapabilities(capsJSON)
	if err != nil {
		t.Fatalf("failed to parse capabilities: %v", err)
	}
	if len(bd.SupportedFormats) != 1 || bd.SupportedFormats[0] != "huggingface" {
		t.Errorf("wrong supported_formats: %v", bd.SupportedFormats)
	}
	if len(bd.SupportedTasks) != 2 {
		t.Errorf("expected 2 tasks, got %v", bd.SupportedTasks)
	}
	if len(bd.BlockedArchitectures) != 0 {
		t.Errorf("expected 0 blocked architectures from JSON without blocked_architectures key, got %v", bd.BlockedArchitectures)
	}
	// Test with blocked_architectures
	capsWithBlock := `{"supported_formats":["huggingface"],"supported_tasks":["chat","vision_chat"],"supported_capabilities":["chat","vision"],"model_path_modes":["directory"],"blocked_architectures":{"InternVLChatModel":"tokenizer not supported"}}`
	bd2, err2 := ParseBackendCapabilities(capsWithBlock)
	if err2 != nil {
		t.Fatalf("failed to parse with blocks: %v", err2)
	}
	if bd2.BlockedArchitectures["InternVLChatModel"] != "tokenizer not supported" {
		t.Errorf("wrong blocked arch: %v", bd2.BlockedArchitectures)
	}
	if ep, ok := bd.TestEndpoints["chat"].(string); !ok || ep != "/v1/chat/completions" {
		t.Errorf("wrong chat endpoint: %v", bd.TestEndpoints["chat"])
	}
}
