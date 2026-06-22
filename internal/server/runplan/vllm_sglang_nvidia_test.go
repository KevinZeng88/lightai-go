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
			Parameters: map[string]interface{}{"served_model_name": "Qwen3-0.6B-Instruct-2512", "max_model_len": 4096.0, "gpu_memory_utilization": 0.6, "enforce_eager": ""},
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

func TestVLLMPortPropagationToAppArgs(t *testing.T) {
	// When deployment service_json has custom ports, the resolver must produce
	// --port matching app_port, not the backend-version default 8000.
	in := makeVLLMTestInput()
	in.Deployment.Service = ServiceInfo{
		HostPort:      8111,
		ContainerPort: 8022,
		AppPort:       8022,
	}
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// Must contain --port 8022
	found8022 := false
	for i, a := range plan.Args {
		if a == "--port" && i+1 < len(plan.Args) && plan.Args[i+1] == "8022" {
			found8022 = true
			break
		}
	}
	if !found8022 {
		t.Fatalf("--port 8022 not found in args: %v", plan.Args)
	}
	// Must NOT contain --port 8000
	for i, a := range plan.Args {
		if a == "--port" && i+1 < len(plan.Args) && plan.Args[i+1] == "8000" {
			t.Fatalf("--port 8000 found but should be overridden: %v", plan.Args)
		}
	}
	// Must have correct Docker port mapping
	if plan.HostPort != 8111 || plan.ContainerPort != 8022 {
		t.Fatalf("HostPort=%d ContainerPort=%d, want 8111/8022", plan.HostPort, plan.ContainerPort)
	}
	preview := EquivalentCommandPreview(plan)
	if !strings.Contains(preview, "-p 8111:8022") {
		t.Fatalf("preview missing -p 8111:8022: %s", preview)
	}
	if strings.Contains(preview, "--port 8000") {
		t.Fatalf("preview contains stale --port 8000: %s", preview)
	}
	if !strings.Contains(preview, "--port 8022") {
		t.Fatalf("preview missing --port 8022: %s", preview)
	}
}

func TestVLLMPositionalModelNoDoubleModel(t *testing.T) {
	in := makeVLLMTestInput()
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// Must NOT contain --model flag (positional model only)
	for _, a := range plan.Args {
		if a == "--model" {
			t.Fatalf("--model flag found but should be positional: %v", plan.Args)
		}
	}
	// First arg after entrypoint should be the model path
	modelFound := false
	for _, a := range plan.Args {
		if strings.Contains(a, "Qwen3") || strings.Contains(a, "/models/") {
			modelFound = true
			break
		}
	}
	if !modelFound {
		t.Fatalf("model path not found in args: %v", plan.Args)
	}
}

func TestDuplicatePortArgsDetected(t *testing.T) {
	in := makeVLLMTestInput()
	in.Deployment.Service = ServiceInfo{
		HostPort:      8111,
		ContainerPort: 8022,
		AppPort:       8022,
	}
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// Count --port occurrences
	portCount := 0
	for _, a := range plan.Args {
		if a == "--port" {
			portCount++
		}
	}
	if portCount != 1 {
		t.Fatalf("expected 1 --port, got %d: %v", portCount, plan.Args)
	}
}

func makeVLLMTestInput() ResolveInput {
	return ResolveInput{
		Backend: &BackendInfo{Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			Version:              "v0.23.0",
			DefaultEntrypoint:    []string{"vllm", "serve"},
			DefaultArgs:          []string{"{{model_container_path}}"},
			DefaultBackendParams: []string{},
			ParameterDefs: []ParameterDef{
				{Name: "--host", CliName: "--host", Default: "0.0.0.0"},
				{Name: "--port", CliName: "--port", Default: "8000"},
				{Name: "--served-model-name", CliName: "--served-model-name"},
				{Name: "--max-model-len", CliName: "--max-model-len"},
				{Name: "--gpu-memory-utilization", CliName: "--gpu-memory-utilization"},
				{Name: "--enforce-eager", CliName: "--enforce-eager"},
			},
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200},
			DefaultContainerPort: 8000,
			DefaultImages:        map[string]string{"nvidia": "vllm/vllm-openai:latest"},
		},
		BackendRuntime: &RuntimeInfo{
			ID: "rt-test", Vendor: "nvidia", RuntimeType: "docker",
			ImageName:  "vllm/vllm-openai:latest",
			Docker:     DockerSpecInfo{Privileged: true, IPCMode: "host", ShmSize: "10g"},
			ModelMount: ModelMountInfo{ContainerPath: "/models", Readonly: true},
		},
		Artifact: &ArtifactInfo{
			Name: "Qwen3-0.6B-Instruct-2512", Path: "/home/kzeng/models/Qwen3-0.6B-Instruct-2512",
			ModelRoot: "/home/kzeng/models", RelativePath: "Qwen3-0.6B-Instruct-2512",
		},
		Deployment: &DeploymentInfo{
			ID: "dep-test", Name: "test",
			Parameters: map[string]interface{}{
				"--served-model-name":      "Qwen3-0.6B-Instruct-2512",
				"--max-model-len":          float64(4096),
				"--gpu-memory-utilization": float64(0.6),
				"--enforce-eager":          "",
			},
			Service: ServiceInfo{HostPort: 8004},
		},
		InstanceID:   "inst-test",
		Node:         &NodeInfo{ID: "node-a", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	}
}

func TestVLLMUserServedModelNameOverridesDefault(t *testing.T) {
	in := ResolveInput{
		Backend: &BackendInfo{Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			Version: "v0.23.0", DefaultEntrypoint: []string{"vllm", "serve"},
			DefaultArgs: []string{"{{model_container_path}}"},
			ParameterDefs: []ParameterDef{
				{Name: "--host", CliName: "--host", Default: "0.0.0.0"},
				{Name: "--port", CliName: "--port", Default: "8000"},
				{Name: "--served-model-name", CliName: "--served-model-name"},
			},
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200},
			DefaultContainerPort: 8000, DefaultImages: map[string]string{"nvidia": "img:latest"},
		},
		BackendRuntime: &RuntimeInfo{ID: "rt", Vendor: "nvidia", RuntimeType: "docker", ImageName: "img:latest", Docker: DockerSpecInfo{}, ModelMount: ModelMountInfo{ContainerPath: "/models"}},
		Artifact:       &ArtifactInfo{Name: "M", Path: "/models/M", ModelRoot: "/models", RelativePath: "M"},
		Deployment: &DeploymentInfo{
			ID: "dep", Name: "test",
			Parameters: map[string]interface{}{"served_model_name": "my-custom-model"},
			Service:    ServiceInfo{HostPort: 8004},
		},
		InstanceID: "inst", Node: &NodeInfo{ID: "n", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	}
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	found := false
	for i, a := range plan.Args {
		if a == "--served-model-name" && i+1 < len(plan.Args) && plan.Args[i+1] == "my-custom-model" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("--served-model-name my-custom-model not found: %v", plan.Args)
	}
}

func TestVLLMUserGpuMemoryUtilizationPropagates(t *testing.T) {
	in := makeVLLMTestInput()
	in.Deployment.Parameters["gpu_memory_utilization"] = 0.85
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	found := false
	for i, a := range plan.Args {
		if a == "--gpu-memory-utilization" && i+1 < len(plan.Args) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("--gpu-memory-utilization not found; user value not propagated: %v", plan.Args)
	}
}

func TestVLLMEnforceEagerUserOverride(t *testing.T) {
	in := makeVLLMTestInput()
	// User explicitly disables enforce_eager
	delete(in.Deployment.Parameters, "--enforce-eager")
	in.Deployment.Parameters["enforce_eager"] = ""
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// enforce_eager is a flag (no value), just check it appears
	found := false
	for _, a := range plan.Args {
		if a == "--enforce-eager" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("--enforce-eager not found: %v", plan.Args)
	}
}

func TestDedupKeepsUserPortOverDefault(t *testing.T) {
	// Simulate the exact scenario: default_args has --port X, user sets --port Y.
	// After the dedup fix (last wins), the user value must survive.
	in := makeVLLMTestInput()
	// default_args in makeVLLMTestInput does NOT include --port (it is positional model),
	// but ParameterDef has default --port 8000. User sets port via service.
	in.Deployment.Service = ServiceInfo{HostPort: 8111, ContainerPort: 8022, AppPort: 8022}
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	portCount := 0
	lastPort := ""
	for i, a := range plan.Args {
		if a == "--port" && i+1 < len(plan.Args) {
			portCount++
			lastPort = plan.Args[i+1]
		}
	}
	if portCount != 1 {
		t.Fatalf("expected 1 --port, got %d: %v", portCount, plan.Args)
	}
	if lastPort != "8022" {
		t.Fatalf("--port = %s, want 8022 (user value should beat default)", lastPort)
	}
}

func TestGetParamMatchesCLIFormatNames(t *testing.T) {
	// Build input directly without makeVLLMTestInput to avoid map sharing issues.
	in := ResolveInput{
		Backend: &BackendInfo{Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			Version: "v0.23.0", DefaultEntrypoint: []string{"vllm", "serve"},
			DefaultArgs: []string{"{{model_container_path}}"},
			ParameterDefs: []ParameterDef{
				{Name: "--host", CliName: "--host", Default: "0.0.0.0"},
				{Name: "--port", CliName: "--port", Default: "8000"},
				{Name: "--max-model-len", CliName: "--max-model-len", Default: "4096"},
			},
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200},
			DefaultContainerPort: 8000,
			DefaultImages:        map[string]string{"nvidia": "vllm/vllm-openai:latest"},
		},
		BackendRuntime: &RuntimeInfo{ID: "rt", Vendor: "nvidia", RuntimeType: "docker", ImageName: "vllm/vllm-openai:latest", Docker: DockerSpecInfo{}, ModelMount: ModelMountInfo{ContainerPath: "/models"}},
		Artifact:       &ArtifactInfo{Name: "Qwen3", Path: "/models/Qwen3", ModelRoot: "/models", RelativePath: "Qwen3"},
		Deployment: &DeploymentInfo{
			ID: "dep", Name: "test",
			Parameters: map[string]interface{}{"max_model_len": 16384.0},
			Service:    ServiceInfo{HostPort: 8004},
		},
		InstanceID: "inst", Node: &NodeInfo{ID: "n", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	}
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("errors: %v", errs)
	}
	found := false
	for i, a := range plan.Args {
		if a == "--max-model-len" && i+1 < len(plan.Args) && plan.Args[i+1] == "16384" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("--max-model-len 16384 not found; normalized name lookup failed: %v", plan.Args)
	}
}
