package runplan

import (
	"strings"
	"testing"
)

func TestResolveVLLMNVIDIA(t *testing.T) {
	in := ResolveInput{
		Backend: &BackendInfo{Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			Version:              "0.8.5",
			DefaultEntrypoint:    []string{"vllm", "serve"},
			DefaultArgs:          []string{"{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}", "--served-model-name", "{{served_model_name}}", "--max-model-len", "{{max_model_len}}", "--gpu-memory-utilization", "{{gpu_memory_utilization}}"},
			DefaultBackendParams: []string{"--enforce-eager"},
			ParameterDefs: []ParameterDef{
				{Name: "max_model_len", CliName: "--max-model-len", Type: "integer", Default: 4096.0, Required: false},
				{Name: "gpu_memory_utilization", CliName: "--gpu-memory-utilization", Type: "number", Default: 0.6, Required: false},
				{Name: "served_model_name", CliName: "--served-model-name", Type: "string", Required: true},
			},
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200, StartupTimeoutSeconds: 120, IntervalSeconds: 2, TimeoutSeconds: 5},
			DefaultContainerPort: 8000,
			DefaultImages:        map[string]string{"nvidia": "vllm/vllm-openai:latest"},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor: "nvidia", RuntimeType: "docker", ImageName: "",
			ArgsOverride: []string{},
			DefaultEnv:   map[string]string{},
			Docker:       DockerSpecInfo{Privileged: true, IPCMode: "host", ShmSize: "10g"},
		},
		Artifact: &ArtifactInfo{
			Name: "Qwen3-0.6B-Instruct-2512", Path: "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
		},
		Deployment: &DeploymentInfo{
			ID: "deploy-vllm", Name: "vllm-test",
			Parameters: map[string]interface{}{"served_model_name": "Qwen3-0.6B-Instruct-2512", "max_model_len": 4096.0, "gpu_memory_utilization": 0.6},
			Service:    ServiceInfo{HostPort: 8004},
		},
		InstanceID:   "inst-vllm-001",
		Node:         &NodeInfo{ID: "KZ-LAPTOP", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}

	if plan.Image != "vllm/vllm-openai:latest" {
		t.Errorf("image: got %q, want vllm/vllm-openai:latest", plan.Image)
	}
	if plan.ContainerPort != 8000 {
		t.Errorf("container_port: %d", plan.ContainerPort)
	}
	if plan.HostPort != 8004 {
		t.Errorf("host_port: %d", plan.HostPort)
	}
	if plan.Privileged != true {
		t.Error("privileged should be true")
	}
	if plan.IPCMode != "host" {
		t.Errorf("ipc_mode: %s", plan.IPCMode)
	}

	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "vllm serve") && !strings.Contains(argsStr, "vllm") {
		// entrypoint may be in Entrypoint field not Args
	}
	if !strings.Contains(argsStr, "/models/Qwen3-0.6B-Instruct-2512") {
		t.Errorf("missing model path: %s", argsStr)
	}
	if !strings.Contains(argsStr, "--served-model-name") {
		t.Error("missing --served-model-name")
	}
	if !strings.Contains(argsStr, "--max-model-len") {
		t.Error("missing --max-model-len")
	}
	if !strings.Contains(argsStr, "--gpu-memory-utilization") {
		t.Error("missing --gpu-memory-utilization")
	}
	if !strings.Contains(argsStr, "--enforce-eager") {
		t.Error("missing backend params")
	}

	preview := EquivalentCommandPreview(plan)
	if preview == "" {
		t.Error("preview empty")
	}
	if !strings.Contains(preview, "vllm/vllm-openai:latest") {
		t.Error("preview missing image")
	}
	if !strings.Contains(preview, "-p 8004:8000") {
		t.Error("preview missing port")
	}
	if !strings.Contains(preview, "CUDA_VISIBLE_DEVICES=0") {
		t.Error("preview missing GPU")
	}

	t.Logf("vLLM preview:\n  %s", preview)
}

func TestResolveSGLangNVIDIA(t *testing.T) {
	in := ResolveInput{
		Backend: &BackendInfo{Name: "sglang", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			Version:              "0.4.6",
			DefaultEntrypoint:    []string{"python3", "-m", "sglang.launch_server"},
			DefaultArgs:          []string{"--model-path", "{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}"},
			DefaultBackendParams: []string{},
			ParameterDefs:        []ParameterDef{{Name: "served_model_name", CliName: "--served-model-name", Type: "string", Required: false}},
			HealthCheck:          HealthCheckInput{Path: "/health", ExpectedStatus: 200, StartupTimeoutSeconds: 120},
			DefaultContainerPort: 30000,
			DefaultImages:        map[string]string{"nvidia": "lmsysorg/sglang:latest"},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor: "nvidia", RuntimeType: "docker", ImageName: "",
			ArgsOverride: []string{},
			DefaultEnv:   map[string]string{},
			Docker:       DockerSpecInfo{Privileged: true, IPCMode: "host", ShmSize: "32g"},
		},
		Artifact: &ArtifactInfo{
			Name: "Qwen3-0.6B-Instruct-2512", Path: "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
		},
		Deployment: &DeploymentInfo{
			ID: "deploy-sglang", Name: "sglang-test",
			Parameters: map[string]interface{}{},
			Service:    ServiceInfo{HostPort: 30000},
		},
		InstanceID:   "inst-sglang-001",
		Node:         &NodeInfo{ID: "KZ-LAPTOP", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}

	if plan.Image != "lmsysorg/sglang:latest" {
		t.Errorf("image: got %q", plan.Image)
	}
	if plan.ContainerPort != 30000 {
		t.Errorf("container_port: %d", plan.ContainerPort)
	}
	if plan.HostPort != 30000 {
		t.Errorf("host_port: %d", plan.HostPort)
	}
	if plan.ShmSize != "32g" {
		t.Errorf("shm_size: %s", plan.ShmSize)
	}
	if plan.IPCMode != "host" {
		t.Errorf("ipc_mode: %s", plan.IPCMode)
	}

	preview := EquivalentCommandPreview(plan)
	if preview == "" {
		t.Error("preview empty")
	}
	if !strings.Contains(preview, "lmsysorg/sglang:latest") {
		t.Error("preview missing image")
	}
	if !strings.Contains(preview, "--shm-size 32g") {
		t.Error("preview missing shm-size")
	}

	t.Logf("SGLang preview:\n  %s", preview)
}
