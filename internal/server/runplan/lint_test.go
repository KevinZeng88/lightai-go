package runplan

import (
	"encoding/json"
	"testing"
)

func TestLintCleanVLLM(t *testing.T) {
	in := LintInput{
		FinalArgs:           []string{"--model", "/models/qwen", "--host", "0.0.0.0", "--port", "8000"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "vllm",
	}
	result := LintRunPlan(in)
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %s; findings: %+v", result.Status, result.Findings)
	}
}

func TestLintCleanSGLang(t *testing.T) {
	in := LintInput{
		FinalArgs:           []string{"--model-path", "/models/qwen", "--host", "0.0.0.0", "--port", "30000"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "sglang",
	}
	result := LintRunPlan(in)
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %s; findings: %+v", result.Status, result.Findings)
	}
}

func TestLintCleanLlamaCpp(t *testing.T) {
	in := LintInput{
		FinalArgs:           []string{"-m", "/models/llama.gguf", "--host", "0.0.0.0", "--port", "8080", "--n-gpu-layers", "-1"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "llamacpp",
	}
	result := LintRunPlan(in)
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %s; findings: %+v", result.Status, result.Findings)
	}
}

func TestLintLlamaCppImageProvidedHostConflict(t *testing.T) {
	// Image provides LLAMA_ARG_HOST, platform provides --host → warning (not error)
	in := LintInput{
		FinalArgs: []string{"-m", "/models/llama.gguf", "--host", "0.0.0.0", "--port", "8080"},
		Env: map[string]string{
			"LLAMA_ARG_HOST": "127.0.0.1",
		},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "llamacpp",
		EnvSources: map[string]string{
			"LLAMA_ARG_HOST": "backend_default",
		},
	}
	result := LintRunPlan(in)
	if result.Status == "ok" {
		t.Fatal("expected warning or error, got ok")
	}
	found := false
	for _, f := range result.Findings {
		if f.ID == "arg.env_cli_conflict" && f.Category == LintCategoryEnvCLIConflict {
			if f.Severity != LintSeverityWarning {
				t.Errorf("image-provided env conflict should be warning, got %s", f.Severity)
			}
			found = true
		}
	}
	if !found {
		t.Error("expected env_cli_conflict finding")
	}
}

func TestLintLlamaCppUserProvidedHostConflict(t *testing.T) {
	// User sets LLAMA_ARG_HOST, platform provides --host → error
	in := LintInput{
		FinalArgs: []string{"-m", "/models/llama.gguf", "--host", "0.0.0.0", "--port", "8080"},
		Env: map[string]string{
			"LLAMA_ARG_HOST": "192.168.1.1",
		},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "llamacpp",
		EnvSources: map[string]string{
			"LLAMA_ARG_HOST": "user_env",
		},
	}
	result := LintRunPlan(in)
	if result.Status != "error" {
		t.Errorf("expected error status, got %s", result.Status)
	}
	found := false
	for _, f := range result.Findings {
		if f.ID == "arg.env_cli_conflict" && f.Category == LintCategoryEnvCLIConflict {
			if f.Severity != LintSeverityError {
				t.Errorf("user-provided env conflict should be error, got %s", f.Severity)
			}
			found = true
		}
	}
	if !found {
		t.Error("expected env_cli_conflict finding")
	}
}

func TestLintDuplicateFlag(t *testing.T) {
	// Pre-dedup args have duplicate --ctx-size
	in := LintInput{
		PreDedupArgs:        []string{"-m", "/m.gguf", "--ctx-size", "2048", "--ctx-size", "4096"},
		FinalArgs:           []string{"-m", "/m.gguf", "--ctx-size", "4096"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "llamacpp",
	}
	result := LintRunPlan(in)
	found := false
	for _, f := range result.Findings {
		if f.ID == "arg.duplicate" {
			found = true
			if f.Severity != LintSeverityError {
				t.Errorf("duplicate should be error, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected arg.duplicate finding")
	}
}

func TestLintDuplicateGpuMemoryUtilization(t *testing.T) {
	in := LintInput{
		PreDedupArgs:        []string{"--model", "/m", "--gpu-memory-utilization", "0.5", "--gpu-memory-utilization", "0.9"},
		FinalArgs:           []string{"--model", "/m", "--gpu-memory-utilization", "0.9"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "vllm",
	}
	result := LintRunPlan(in)
	found := false
	for _, f := range result.Findings {
		if f.ID == "arg.duplicate" {
			found = true
		}
	}
	if !found {
		t.Error("expected arg.duplicate finding for --gpu-memory-utilization")
	}
}

func TestLintDuplicateMemFractionStatic(t *testing.T) {
	in := LintInput{
		PreDedupArgs:        []string{"--model-path", "/m", "--mem-fraction-static", "0.5", "--mem-fraction-static", "0.9"},
		FinalArgs:           []string{"--model-path", "/m", "--mem-fraction-static", "0.9"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		BackendName:         "sglang",
	}
	result := LintRunPlan(in)
	found := false
	for _, f := range result.Findings {
		if f.ID == "arg.duplicate" {
			found = true
		}
	}
	if !found {
		t.Error("expected arg.duplicate finding for --mem-fraction-static")
	}
}

func TestLintPrivilegedWarning(t *testing.T) {
	docker := &DockerSpecInfo{Privileged: true}
	in := LintInput{
		FinalArgs:           []string{"--model", "/m"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		DockerSpec:          docker,
	}
	result := LintRunPlan(in)
	found := false
	for _, f := range result.Findings {
		if f.ID == "security.privileged_enabled" {
			found = true
			if f.Severity != LintSeverityWarning {
				t.Errorf("privileged should be warning, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected security.privileged_enabled finding")
	}
}

func TestLintIPCHostWarning(t *testing.T) {
	docker := &DockerSpecInfo{IPCMode: "host"}
	in := LintInput{
		FinalArgs:           []string{"--model", "/m"},
		Env:                 map[string]string{},
		PlatformOwnedParams: DefaultLogicalParamSpecs(),
		DockerSpec:          docker,
	}
	result := LintRunPlan(in)
	found := false
	for _, f := range result.Findings {
		if f.ID == "security.ipc_host" {
			found = true
		}
	}
	if !found {
		t.Error("expected security.ipc_host finding")
	}
}

func TestLintResultJSON(t *testing.T) {
	result := LintResult{
		Status: "warning",
		Findings: []LintFinding{
			{
				ID:       "arg.env_cli_conflict",
				Severity: LintSeverityWarning,
				Category: LintCategoryEnvCLIConflict,
				Message:  "test message",
			},
		},
	}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	var roundtrip LintResult
	if err := json.Unmarshal(b, &roundtrip); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if roundtrip.Status != "warning" {
		t.Errorf("expected warning, got %s", roundtrip.Status)
	}
	if len(roundtrip.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(roundtrip.Findings))
	}
	if roundtrip.Findings[0].ID != "arg.env_cli_conflict" {
		t.Errorf("unexpected finding ID: %s", roundtrip.Findings[0].ID)
	}
}
