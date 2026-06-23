package runplan

import (
	"strings"
	"testing"
)

// TestLlamaCppNvidiaRunPlan validates that the LightAI RunPlan Resolver
// generates a structurally correct plan matching the real Docker command
// from docs/RUNBOOK-LLAMA-CPP-GGUF-NVIDIA-5090.md.
func TestLlamaCppNvidiaRunPlan(t *testing.T) {
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			ID:             "backend-llamacpp",
			Name:           "llamacpp",
			DefaultVersion: "b4817",
			DefaultEnv:     map[string]string{},
		},
		BackendVersion: &VersionInfo{
			ID:                   "bver-llamacpp-b4817",
			Version:              "b4817",
			DefaultEntrypoint:    []string{},
			DefaultArgs:          []string{"llama-server", "-m", "{{model_container_file}}", "--host", "0.0.0.0", "--port", "{{container_port}}", "-ngl", "{{assigned_gpu_count}}"},
			DefaultBackendParams: []string{},
			ParameterDefs: []ParameterDef{
				{Name: "ctx_size", CliName: "--ctx-size", Type: "integer", Default: 4096.0, Required: false},
				{Name: "n_gpu_layers", CliName: "--n-gpu-layers", Type: "integer", Default: 999.0, Required: false},
				{Name: "served_model_name", CliName: "--model", Type: "string", Required: true},
			},
			HealthCheck: HealthCheckInput{
				Path: "/health", ExpectedStatus: 200,
				StartupTimeoutSeconds: 60, IntervalSeconds: 2, TimeoutSeconds: 5,
			},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "ghcr.io/ggml-org/llama.cpp:server-cuda13"},
			Env:                  map[string]string{},
		},
		BackendRuntime: &RuntimeInfo{
			ID:           "runtime-llamacpp-nvidia",
			Vendor:       "nvidia",
			RuntimeType:  "docker",
			ImageName:    "", // leave empty to test defaultImages fallback
			ArgsOverride: []string{"--ctx-size", "4096", "--n-gpu-layers", "999"},
			DefaultEnv:   map[string]string{},
			Docker:       DockerSpecInfo{},
		},
		Artifact: &ArtifactInfo{
			ID:   "artifact-qwen35-9b-q4",
			Name: "Qwen3.5-9B-Q4_K_M.gguf",
			Path: "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf",
		},
		Deployment: &DeploymentInfo{
			ID:   "deploy-llamacpp",
			Name: "qwen35-9b-llamacpp",
			ParameterValues: []ParameterValue{
				{Key: "served_model_name", CliName: "--model", Type: "string", Enabled: true, Value: "Qwen3.5-9B-Q4_K_M.gguf"},
				{Key: "ctx_size", CliName: "--ctx-size", Type: "integer", Enabled: true, Value: 4096.0},
				{Key: "n_gpu_layers", CliName: "--n-gpu-layers", Type: "integer", Enabled: true, Value: 999.0},
			},
			Service: ServiceInfo{HostPort: 8002},
		},
		InstanceID: "inst-llamacpp-001",
		Node:       &NodeInfo{ID: "KZ-LAPTOP", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{
			{Index: 0, Vendor: "nvidia"},
		},
	})

	plan, errs, warns := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}

	// --- Structural validation against real Docker command ---

	// 1. Image: should resolve from BackendVersion.defaultImages[nvidia]
	if plan.Image != "ghcr.io/ggml-org/llama.cpp:server-cuda13" {
		t.Errorf("image: got %q, want ghcr.io/ggml-org/llama.cpp:server-cuda13", plan.Image)
	}

	// 2. Container port = 8080
	if plan.ContainerPort != 8080 {
		t.Errorf("container_port: got %d, want 8080", plan.ContainerPort)
	}

	// 3. Host port = 8002
	if plan.HostPort != 8002 {
		t.Errorf("host_port: got %d, want 8002", plan.HostPort)
	}

	// 4. Args must include key llama-server arguments
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "llama-server") {
		t.Error("args missing llama-server command")
	}
	// model container path (after mount translation) should be /models/Qwen3.5-9B-Q4_K_M.gguf
	if !strings.Contains(argsStr, "/models/Qwen3.5-9B-Q4_K_M.gguf") {
		t.Errorf("args missing model container path: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--host") || !strings.Contains(argsStr, "0.0.0.0") {
		t.Error("args missing --host 0.0.0.0")
	}
	if !strings.Contains(argsStr, "--port") || !strings.Contains(argsStr, "8080") {
		t.Error("args missing --port 8080")
	}
	if !strings.Contains(argsStr, "--ctx-size") {
		t.Error("args missing --ctx-size")
	}
	if !strings.Contains(argsStr, "--n-gpu-layers") {
		t.Error("args missing --n-gpu-layers")
	}

	// 5. GPU configuration
	if len(plan.GPUDeviceIDs) != 1 || plan.GPUDeviceIDs[0] != "0" {
		t.Errorf("GPUDeviceIDs: got %v, want [0]", plan.GPUDeviceIDs)
	}
	if plan.Env["CUDA_VISIBLE_DEVICES"] != "0" {
		t.Errorf("CUDA_VISIBLE_DEVICES: got %q, want \"0\"", plan.Env["CUDA_VISIBLE_DEVICES"])
	}

	// 6. Mount: host model path → container /models/...
	if len(plan.Mounts) == 0 {
		t.Fatal("no mounts generated")
	}
	mount := plan.Mounts[0]
	expectedContainer := "/models/Qwen3.5-9B-Q4_K_M.gguf"
	if mount.ContainerPath != expectedContainer {
		t.Errorf("mount container_path: got %q, want %q", mount.ContainerPath, expectedContainer)
	}
	expectedHost := "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf"
	if mount.HostPath != expectedHost {
		t.Errorf("mount host_path: got %q, want %q", mount.HostPath, expectedHost)
	}
	if !mount.Readonly {
		t.Error("model mount should be readonly")
	}

	// 7. Health check
	if plan.HealthCheck.Path != "/health" {
		t.Errorf("health_check path: got %q, want /health", plan.HealthCheck.Path)
	}
	if plan.HealthCheck.ExpectedStatus != 200 {
		t.Errorf("health_check expected_status: got %d, want 200", plan.HealthCheck.ExpectedStatus)
	}

	// 8. Docker preview should be non-empty
	preview := EquivalentCommandPreview(plan)
	if preview == "" {
		t.Error("docker_preview is empty")
	}
	if !strings.Contains(preview, "ghcr.io/ggml-org/llama.cpp:server-cuda13") {
		t.Error("docker_preview missing image")
	}
	if !strings.Contains(preview, "-p 8002:8080") {
		t.Error("docker_preview missing port mapping")
	}
	if !strings.Contains(preview, "CUDA_VISIBLE_DEVICES=0") {
		t.Error("docker_preview missing GPU env")
	}

	// 9. Input hash and plan hash should be non-empty
	if plan.InputHash == "" {
		t.Error("input_hash is empty")
	}
	if plan.PlanHash == "" {
		t.Error("plan_hash is empty")
	}

	// 10. No warnings for valid input (except maybe template resolution notes)
	if len(warns) > 0 {
		t.Logf("warnings (non-fatal): %v", warns)
	}

	t.Logf("docker_preview:\n  %s", preview)
	t.Logf("input_hash: %s", plan.InputHash)
	t.Logf("plan_hash: %s", plan.PlanHash)
}

// TestLlamaCppGGUFFileInDirectory verifies that when the model_locations.relative_path
// is a directory but the artifact path points to a .gguf file, the -m flag uses the
// specific .gguf file path while the mount uses the directory. This is the production
// scenario where the old scan proxy stored directory-level paths (WEB-AI-RC-001).
func TestLlamaCppGGUFFileInDirectory(t *testing.T) {
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			ID:             "backend-llamacpp",
			Name:           "llamacpp",
			DefaultVersion: "b9700",
			DefaultEnv:     map[string]string{},
		},
		BackendVersion: &VersionInfo{
			ID:                "llamacpp-b9700",
			Version:           "b9700",
			DefaultEntrypoint: []string{},
			DefaultArgs:       []string{"-m", "{{model_container_file}}", "--host", "0.0.0.0", "--port", "{{container_port}}"},
			ParameterDefs: []ParameterDef{
				{Name: "ctx_size", CliName: "--ctx-size", Type: "integer", Default: 4096.0, Required: false},
				{Name: "n_gpu_layers", CliName: "--n-gpu-layers", Type: "integer", Default: 999.0, Required: false},
			},
			HealthCheck: HealthCheckInput{
				Path: "/health", ExpectedStatus: 200,
				StartupTimeoutSeconds: 60, IntervalSeconds: 2, TimeoutSeconds: 5,
			},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "ghcr.io/ggml-org/llama.cpp:server-cuda13"},
			Env:                  map[string]string{},
		},
		BackendRuntime: &RuntimeInfo{
			ID:          "runtime-llamacpp-nvidia",
			Vendor:      "nvidia",
			RuntimeType: "docker",
			DefaultEnv:  map[string]string{},
			Docker:      DockerSpecInfo{},
			ModelMount:  ModelMountInfo{ContainerPath: "/models", Readonly: true},
		},
		// Production scenario: model_locations.relative_path = directory name,
		// but artifact path = specific .gguf file.
		Artifact: &ArtifactInfo{
			ID:           "artifact-qwen35-9b-q4",
			Name:         "Qwen3.5-9B-Q4_K_M",
			Path:         "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf",
			ModelRoot:    "/home/kzeng/models",
			RelativePath: "Qwen3.5-9B-Q4", // directory name from old scan proxy
		},
		Deployment: &DeploymentInfo{
			ID:      "deploy-llamacpp-gguf",
			Name:    "qwen35-9b-llamacpp-gguf",
			Service: ServiceInfo{HostPort: 8004},
		},
		InstanceID: "inst-llamacpp-gguf-001",
		Node:       &NodeInfo{ID: "KZ-LAPTOP", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{
			{Index: 0, Vendor: "nvidia"},
		},
	})

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}

	argsStr := strings.Join(plan.Args, " ")

	// CRITICAL: -m must point to the .gguf FILE, not the directory.
	if !strings.Contains(argsStr, "/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf") {
		t.Errorf("args must contain file path in -m, got: %s", argsStr)
	}
	if strings.HasSuffix(strings.TrimSpace(argsStr), "/models/Qwen3.5-9B-Q4") {
		t.Error("args must not end with directory path for -m")
	}

	// Mount should use the directory (for multi-file access).
	if len(plan.Mounts) == 0 {
		t.Fatal("no mounts generated")
	}
	mount := plan.Mounts[0]
	expectedContainer := "/models/Qwen3.5-9B-Q4"
	if mount.ContainerPath != expectedContainer {
		t.Errorf("mount container_path: got %q, want %q (directory mount preserves multi-file access)", mount.ContainerPath, expectedContainer)
	}

	t.Logf("docker_preview:\n  %s", EquivalentCommandPreview(plan))
	t.Logf("args: %s", argsStr)
}

// TestLlamaCppRunPlanNoGPU verifies CPU-only mode.
func TestLlamaCppRunPlanNoGPU(t *testing.T) {
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			Name:       "llamacpp",
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			Version:              "b4817",
			DefaultEntrypoint:    []string{},
			DefaultArgs:          []string{"llama-server", "-m", "{{model_container_path}}"},
			DefaultBackendParams: []string{},
			ParameterDefs:        []ParameterDef{},
			HealthCheck:          HealthCheckInput{Path: "/health", ExpectedStatus: 200},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "ghcr.io/ggml-org/llama.cpp:server-cuda13"},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor:      "cpu",
			RuntimeType: "docker",
			ImageName:   "ghcr.io/ggml-org/llama.cpp:server-cuda13",
		},
		Artifact: &ArtifactInfo{
			Path: "/tmp/test.gguf",
		},
		Deployment: &DeploymentInfo{
			Parameters: map[string]interface{}{},
		},
		InstanceID:   "inst-test",
		Node:         &NodeInfo{ID: "node-1", IP: "127.0.0.1"},
		AssignedGPUs: nil, // no GPUs
	})

	plan, _, _ := Resolve(in)
	if plan == nil {
		t.Fatal("plan is nil")
	}
	if plan.Env["CUDA_VISIBLE_DEVICES"] != "" {
		t.Error("should not set CUDA_VISIBLE_DEVICES with no GPUs")
	}
}
