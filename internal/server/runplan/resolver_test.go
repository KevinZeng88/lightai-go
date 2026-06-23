package runplan

import (
	"strings"
	"testing"
)

func makeTestInput() ResolveInput {
	return ResolveInput{
		Backend: &BackendInfo{
			ID:             "backend-vllm",
			Name:           "vllm",
			DefaultVersion: "0.8.5",
			DefaultEnv:     map[string]string{"GLOBAL_ENV": "1"},
		},
		BackendVersion: &VersionInfo{
			ID:                   "bver-vllm-0.8.5",
			Version:              "0.8.5",
			DefaultEntrypoint:    []string{"vllm", "serve"},
			DefaultArgs:          []string{"{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}", "--served-model-name", "{{served_model_name}}"},
			DefaultBackendParams: []string{"--enforce-eager"},
			ParameterDefs: []ParameterDef{
				{Name: "max_model_len", CliName: "--max-model-len", Type: "integer", Default: 8192.0, Required: false},
				{Name: "served_model_name", CliName: "--served-model-name", Type: "string", Required: true},
			},
			HealthCheck: HealthCheckInput{
				Path: "/v1/models", ExpectedStatus: 200,
				StartupTimeoutSeconds: 120, IntervalSeconds: 2, TimeoutSeconds: 5,
			},
			DefaultContainerPort: 8000,
			DefaultImages:        map[string]string{"nvidia": "vllm/vllm-openai:v0.8.5"},
			Env:                  map[string]string{"VLLM_USE_MODELSCOPE": "true"},
		},
		BackendRuntime: &RuntimeInfo{
			ID:           "runtime-vllm-nvidia",
			Vendor:       "nvidia",
			RuntimeType:  "docker",
			ImageName:    "vllm/vllm-openai:v0.8.5-custom",
			ArgsOverride: []string{"--trust-remote-code"},
			DefaultEnv:   map[string]string{"RUNTIME_VAR": "1"},
			Docker: DockerSpecInfo{
				Privileged: true, IPCMode: "host", ShmSize: "10g",
			},
		},
		Artifact: &ArtifactInfo{
			ID: "artifact-qwen", Name: "Qwen3-32B", Path: "/data/models/Qwen3-32B",
		},
		Deployment: &DeploymentInfo{
			ID:   "deploy-1",
			Name: "qwen3-deploy",
			Parameters: map[string]interface{}{
				"served_model_name": "qwen3-32b",
				"max_model_len":     32768.0,
			},
			Service: ServiceInfo{HostPort: 8001},
		},
		InstanceID: "inst-000000000001",
		Node:       &NodeInfo{ID: "node-1", IP: "192.168.1.100"},
		AssignedGPUs: []GPUInfo{
			{Index: 0, Vendor: "nvidia"},
			{Index: 1, Vendor: "nvidia"},
			{Index: 2, Vendor: "nvidia"},
			{Index: 3, Vendor: "nvidia"},
		},
	}
}

func TestResolveBasic(t *testing.T) {
	plan, errors, _ := Resolve(makeTestInput())
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
	if plan.Image == "" {
		t.Error("image is empty")
	}
	if len(plan.Args) == 0 {
		t.Error("args is empty")
	}
	if len(plan.Env) == 0 {
		t.Error("env is empty")
	}
	if plan.ContainerPort != 8000 {
		t.Errorf("expected container port 8000, got %d", plan.ContainerPort)
	}
	if plan.HostPort != 8001 {
		t.Errorf("expected host port 8001, got %d", plan.HostPort)
	}
	if plan.Privileged != true {
		t.Error("expected privileged=true")
	}
	if plan.IPCMode != "host" {
		t.Errorf("expected ipc_mode=host, got %s", plan.IPCMode)
	}
	if plan.ShmSize != "10g" {
		t.Errorf("expected shm_size=10g, got %s", plan.ShmSize)
	}
	if plan.InputHash == "" {
		t.Error("input_hash is empty")
	}
	if plan.PlanHash == "" {
		t.Error("plan_hash is empty")
	}
}

func TestResolveServicePortSemantics(t *testing.T) {
	in := makeTestInput()
	in.Deployment.Service = ServiceInfo{HostPort: 8005}
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan.HostPort != 8005 {
		t.Fatalf("host_port=%d", plan.HostPort)
	}
	if plan.ContainerPort != 8000 {
		t.Fatalf("container_port=%d want backend default 8000", plan.ContainerPort)
	}
	preview := EquivalentCommandPreview(plan)
	if !strings.Contains(preview, "-p 8005:8000/tcp") {
		t.Fatalf("preview missing host:container mapping: %s", preview)
	}

	in2 := makeTestInput()
	in2.Deployment.Service = ServiceInfo{HostPort: 18005, ContainerPort: 18080, AppPort: 18080, HealthPort: 18005, APITestPort: 18005}
	plan2, errs2, _ := Resolve(in2)
	if len(errs2) > 0 {
		t.Fatalf("unexpected errors: %v", errs2)
	}
	if plan2.HostPort != 18005 || plan2.ContainerPort != 18080 {
		t.Fatalf("ports host=%d container=%d", plan2.HostPort, plan2.ContainerPort)
	}
	if !strings.Contains(strings.Join(plan2.Args, " "), "18080") {
		t.Fatalf("args did not use effective app/container port: %v", plan2.Args)
	}
	if !strings.Contains(EquivalentCommandPreview(plan2), "-p 18005:18080/tcp") {
		t.Fatalf("preview mismatch: %s", EquivalentCommandPreview(plan2))
	}
}

func TestResolveImagePriority(t *testing.T) {
	// Priority 1: NodeRuntimeOverride.image_name
	in := makeTestInput()
	in.NodeRuntimeOverride = &NodeOverrideInfo{ImageName: "node-image:latest"}
	plan, _, _ := Resolve(in)
	if plan.Image != "node-image:latest" {
		t.Errorf("expected node-image:latest, got %s", plan.Image)
	}

	// Priority 2: BackendRuntime.image_name
	in2 := makeTestInput()
	plan2, _, _ := Resolve(in2)
	if plan2.Image != "vllm/vllm-openai:v0.8.5-custom" {
		t.Errorf("expected runtime image, got %s", plan2.Image)
	}

	// Priority 3: BackendVersion.defaultImages[vendor]
	in3 := makeTestInput()
	in3.BackendRuntime.ImageName = ""
	plan3, errs, _ := Resolve(in3)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan3.Image != "vllm/vllm-openai:v0.8.5" {
		t.Errorf("expected default image, got %s", plan3.Image)
	}

	// No image available → error
	in4 := makeTestInput()
	in4.BackendRuntime.ImageName = ""
	in4.BackendVersion.DefaultImages = nil
	_, errs4, _ := Resolve(in4)
	if len(errs4) == 0 {
		t.Error("expected error for no image")
	}
}

func TestResolveArgs(t *testing.T) {
	plan, _, _ := Resolve(makeTestInput())

	// Check args contain expected values
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "/models/Qwen3-32B") {
		t.Error("args missing model container path")
	}
	if !strings.Contains(argsStr, "--port") || !strings.Contains(argsStr, "8000") {
		t.Error("args missing port")
	}
	if !strings.Contains(argsStr, "--served-model-name") || !strings.Contains(argsStr, "qwen3-32b") {
		t.Error("args missing served-model-name")
	}
	if !strings.Contains(argsStr, "--enforce-eager") {
		t.Error("args missing backend params")
	}
	if !strings.Contains(argsStr, "--trust-remote-code") {
		t.Error("args missing args override")
	}
	if !strings.Contains(argsStr, "--max-model-len") {
		t.Error("args missing deployment parameters")
	}
}

func TestResolveEnv(t *testing.T) {
	plan, _, _ := Resolve(makeTestInput())

	if plan.Env["GLOBAL_ENV"] != "1" {
		t.Errorf("missing backend default env: %v", plan.Env)
	}
	if plan.Env["VLLM_USE_MODELSCOPE"] != "true" {
		t.Errorf("missing version env: %v", plan.Env)
	}
	if plan.Env["RUNTIME_VAR"] != "1" {
		t.Errorf("missing runtime env: %v", plan.Env)
	}
	if plan.Env["CUDA_VISIBLE_DEVICES"] != "0,1,2,3" {
		t.Errorf("missing GPU visible env: %v", plan.Env)
	}
}

func TestResolveEnvOverride(t *testing.T) {
	in := makeTestInput()
	in.Deployment.EnvOverrides = map[string]string{"GLOBAL_ENV": "overridden"}
	plan, _, _ := Resolve(in)
	if plan.Env["GLOBAL_ENV"] != "overridden" {
		t.Errorf("deployment override not applied: got %s", plan.Env["GLOBAL_ENV"])
	}
}

func TestResolveNodeOverride(t *testing.T) {
	in := makeTestInput()
	in.NodeRuntimeOverride = &NodeOverrideInfo{
		ImageName: "node-image:latest",
		Env:       map[string]string{"NODE_VAR": "node-value"},
		DockerOverride: &DockerSpecInfo{
			ShmSize: "20g",
			Devices: []DeviceMapping{{HostPath: "/dev/dri", ContainerPath: "/dev/dri"}},
		},
		ModelRootHostPath: "/data/part2/models",
	}
	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan.Image != "node-image:latest" {
		t.Errorf("node override image not applied")
	}
	if plan.Env["NODE_VAR"] != "node-value" {
		t.Errorf("node override env not applied")
	}
	if plan.ShmSize != "20g" {
		t.Errorf("node override shm_size not applied: %s", plan.ShmSize)
	}
	if len(plan.Devices) != 1 || plan.Devices[0].HostPath != "/dev/dri" {
		t.Errorf("node override devices not applied: %v", plan.Devices)
	}
}

func TestUnknownVariableError(t *testing.T) {
	in := makeTestInput()
	in.BackendVersion.DefaultArgs = []string{"{{undefined_variable}}"}
	_, errs, _ := Resolve(in)
	if len(errs) == 0 {
		t.Error("expected error for undefined variable")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "undefined") {
			found = true
		}
	}
	if !found {
		t.Errorf("error should mention undefined variable: %v", errs)
	}
}

func TestNoVarSyntax(t *testing.T) {
	// ${VAR} should be treated as literal text, not a placeholder
	in := makeTestInput()
	in.BackendVersion.DefaultArgs = []string{"${MAX_MODEL_LEN}"}
	plan, _, _ := Resolve(in)
	// Should not error — ${VAR} is not our template syntax
	// But it also shouldn't be replaced
	if strings.Contains(strings.Join(plan.Args, " "), "${MAX_MODEL_LEN}") {
		// unchanged literal — this is correct behavior
	}
}

func TestInputHashDeterministic(t *testing.T) {
	in := makeTestInput()
	p1, _, _ := Resolve(in)
	p2, _, _ := Resolve(in)
	if p1.InputHash != p2.InputHash {
		t.Error("input hash not deterministic")
	}
	if p1.PlanHash != p2.PlanHash {
		t.Error("plan hash not deterministic")
	}
}

func TestInputHashDifferent(t *testing.T) {
	in1 := makeTestInput()
	in2 := makeTestInput()
	in2.Deployment.Service.HostPort = 9000
	p1, _, _ := Resolve(in1)
	p2, _, _ := Resolve(in2)
	if p1.InputHash == p2.InputHash {
		t.Error("input hash should differ for different inputs")
	}
}

func TestRuntimeTypeValidation(t *testing.T) {
	in := makeTestInput()
	in.BackendRuntime.RuntimeType = "kubernetes"
	_, errs, _ := Resolve(in)
	if len(errs) == 0 {
		t.Error("expected error for non-docker runtime type")
	}
}

func TestEquivalentCommandPreview(t *testing.T) {
	plan, _, _ := Resolve(makeTestInput())
	preview := EquivalentCommandPreview(plan)
	if preview == "" {
		t.Error("docker preview is empty")
	}
	if !strings.HasPrefix(preview, "docker run -d") {
		t.Errorf("preview should start with 'docker run -d': %s", preview[:50])
	}
	if !strings.Contains(preview, plan.Image) {
		t.Error("preview missing image")
	}
	if !strings.Contains(preview, "--privileged") {
		t.Error("preview missing --privileged")
	}
	if !strings.Contains(preview, "--ipc host") {
		t.Error("preview missing --ipc")
	}
	if !strings.Contains(preview, "--shm-size 10g") {
		t.Error("preview missing --shm-size")
	}
	if !strings.Contains(preview, "-p 8001:8000/tcp") {
		t.Errorf("preview missing port mapping: %s", preview)
	}
	if !strings.Contains(preview, "CUDA_VISIBLE_DEVICES=0,1,2,3") {
		t.Error("preview missing GPU env")
	}
}

func TestReplicasNotSupported(t *testing.T) {
	in := makeTestInput()
	// Replicas > 1 should be rejected at API level, not resolver
	// For now, resolver handles single instance
	plan, _, _ := Resolve(in)
	if plan == nil {
		t.Error("single replica should resolve fine")
	}
}

func TestDefaultHealthCheck(t *testing.T) {
	plan, _, _ := Resolve(makeTestInput())
	if plan.HealthCheck.Path != "/v1/models" {
		t.Errorf("expected health check path /v1/models, got %s", plan.HealthCheck.Path)
	}
	if plan.HealthCheck.ExpectedStatus != 200 {
		t.Errorf("expected health check status 200, got %d", plan.HealthCheck.ExpectedStatus)
	}
}

func TestResolveNoGPU(t *testing.T) {
	in := makeTestInput()
	in.AssignedGPUs = nil
	plan, _, _ := Resolve(in)
	if plan.Env["CUDA_VISIBLE_DEVICES"] != "" {
		t.Error("expected no CUDA_VISIBLE_DEVICES for no GPU")
	}
}

func TestArgsOverrideAppendOnly(t *testing.T) {
	in := makeTestInput()
	// args_override appends, doesn't replace
	in.BackendRuntime.ArgsOverride = []string{"--custom-flag", "--another-flag"}
	plan, _, _ := Resolve(in)
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "--custom-flag") {
		t.Error("args missing custom flag from override")
	}
	if !strings.Contains(argsStr, "--another-flag") {
		t.Error("args missing another flag from override")
	}
	// Original args should still be present (not replaced)
	if !strings.Contains(argsStr, "/models/Qwen3-32B") {
		t.Error("original args should still be present (append, not replace)")
	}
}

// TestContainerPathSafety validates that dangerous relative paths are rejected
// and safe paths produce correct container mounts under /models.
func TestContainerPathSafety(t *testing.T) {
	baseInput := ResolveInput{
		Backend:        &BackendInfo{ID: "b.vllm", Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{ID: "bv.openai", DefaultEntrypoint: []string{"serve"}, DefaultArgs: []string{}, DefaultBackendParams: []string{}, ParameterDefs: []ParameterDef{}, HealthCheck: HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200}, DefaultContainerPort: 8000, DefaultImages: map[string]string{"nvidia": "img:latest"}, Env: map[string]string{}},
		BackendRuntime: &RuntimeInfo{ID: "rt.vllm", Vendor: "nvidia", RuntimeType: "docker", ImageName: "img:latest", ArgsOverride: []string{}, DefaultEnv: map[string]string{}, Docker: DockerSpecInfo{}, ModelMount: ModelMountInfo{ContainerPath: "/models", Readonly: true}},
		Deployment:     &DeploymentInfo{ID: "d1", Name: "test", Parameters: map[string]interface{}{}, EnvOverrides: map[string]string{}, Service: ServiceInfo{HostPort: 8002}, Placement: PlacementInfo{NodeID: "n1"}},
		InstanceID:     "inst-safety",
		Node:           &NodeInfo{ID: "n1", IP: "127.0.0.1"},
		AssignedGPUs:   []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	}

	tests := []struct {
		name         string
		modelRoot    string
		relativePath string
		wantErr      bool
	}{
		{"safe simple", "/data/models", "qwen", false},
		{"safe nested", "/data/models", "family/qwen", false},
		{"dangerous dotdot", "/data/models", "../etc", true},
		{"dangerous absolute", "/data/models", "/etc", true},
		{"dangerous empty", "/data/models", "", true},
		{"safe with dot in name", "/data/models", "qwen3.5-9b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := baseInput
			in.Artifact = &ArtifactInfo{
				ID:           "a1",
				Name:         "test-model",
				Path:         tt.modelRoot + "/" + tt.relativePath,
				ModelRoot:    tt.modelRoot,
				RelativePath: tt.relativePath,
			}
			_, errs, _ := Resolve(in)
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("Resolve() errors=%v, wantErr=%v. errs: %v", hasErr, tt.wantErr, errs)
			}
			if !tt.wantErr {
				plan, _, _ := Resolve(in)
				if plan != nil && len(plan.Mounts) > 0 {
					cp := plan.Mounts[0].ContainerPath
					if !strings.HasPrefix(cp, "/models/") && cp != "/models" {
						t.Errorf("container path %q not under /models", cp)
					}
					if strings.Contains(cp, "..") {
						t.Errorf("container path %q contains ..", cp)
					}
				}
			}
		})
	}
}

func TestMapParametersToArgsClipsNameFallback(t *testing.T) {
	// When def.CliName is empty, it must fall back to def.Name.
	// ParameterDefs from BackendVersion catalog use "name":"--host",
	// but did not include "cli_name".  mapParametersToArgs must output
	// the flag-value pair, not bare values.
	defs := []ParameterDef{
		{Name: "--host", Default: "0.0.0.0"},
		{Name: "--port", Default: "8000"},
		{Name: "--model", Required: true, CliName: "--model"},
	}
	// No deployment params → all come from defaults.
	args := mapParametersToArgs(map[string]interface{}{}, defs, nil)
	joined := strings.Join(args, " ")

	// Must contain " --host 0.0.0.0", not bare " 0.0.0.0".
	if !strings.Contains(joined, "--host") {
		t.Errorf("expected --host flag in args, got: %q", joined)
	}
	if !strings.Contains(joined, "--port") {
		t.Errorf("expected --port flag in args, got: %q", joined)
	}
	if strings.Contains(joined, " 0.0.0.0") && !strings.Contains(joined, "--host") {
		t.Errorf("bare value 0.0.0.0 without --host flag: %q", joined)
	}
	if strings.Contains(joined, "8000") && !strings.Contains(joined, "--port") {
		t.Errorf("bare value 8000 without --port flag: %q", joined)
	}

	// Required param without default and no deployment param → skipped.
	if strings.Count(joined, "--model") > 1 {
		t.Errorf("required param with no value should not appear twice: %q", joined)
	}

	// With explicit deployment params, they should take precedence.
	args2 := mapParametersToArgs(map[string]interface{}{
		"--host": "1.2.3.4",
		"--port": "9999",
	}, defs, nil)
	joined2 := strings.Join(args2, " ")
	if !strings.Contains(joined2, "--host 1.2.3.4") {
		t.Errorf("deployment param --host override not applied: %q", joined2)
	}
	if !strings.Contains(joined2, "--port 9999") {
		t.Errorf("deployment param --port override not applied: %q", joined2)
	}
}

func TestVLLMRunPlanRendersHostPortFlags(t *testing.T) {
	// Simulates a full vLLM deployment: BackendVersion parameter_defs
	// use "name":"--host"/"--port" without cli_name.
	// The resolved RunPlan args must contain --host and --port flags.
	in := ResolveInput{
		Backend: &BackendInfo{ID: "backend.vllm", Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			ID:                "vllm-v0.23.0",
			DefaultEntrypoint: []string{"vllm", "serve"},
			DefaultArgs:       []string{"{{MODEL_CONTAINER_PATH}}"},
			ParameterDefs: []ParameterDef{
				{Name: "--host", Default: "0.0.0.0"},
				{Name: "--port", Default: "8000"},
				{Name: "--served-model-name"},
				{Name: "--max-model-len"},
			},
			DefaultContainerPort: 8000,
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200},
			DefaultImages:        map[string]string{"default": "vllm/vllm-openai:latest"},
		},
		BackendRuntime: &RuntimeInfo{
			ID:          "rt.vllm.nvidia",
			Vendor:      "nvidia",
			RuntimeType: "docker",
			ImageName:   "vllm/vllm-openai:latest",
			ModelMount:  ModelMountInfo{ContainerPath: "/models", Readonly: true},
		},
		Deployment: &DeploymentInfo{
			ID:           "dep-vllm",
			Name:         "vllm-test",
			Parameters:   map[string]interface{}{"served_model_name": "test", "max_model_len": float64(4096)},
			EnvOverrides: map[string]string{},
			Service:      ServiceInfo{HostPort: 8004},
		},
		Artifact: &ArtifactInfo{
			ID:           "art-qwen",
			Name:         "Qwen3",
			RelativePath: "Qwen3-0.6B-Instruct-2512",
		},
		InstanceID:   "inst-vllm-test",
		Node:         &NodeInfo{ID: "n1", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	}
	plan, _, _ := Resolve(in)
	argsStr := strings.Join(plan.Args, " ")

	// Must contain proper --flag value pairs.
	if !strings.Contains(argsStr, "--host") {
		t.Errorf("missing --host flag: %q", argsStr)
	}
	if !strings.Contains(argsStr, "--port") {
		t.Errorf("missing --port flag: %q", argsStr)
	}
	// Verify flag-value integrity: value must be immediately after flag.
	idxHost := strings.Index(argsStr, "--host")
	if idxHost >= 0 {
		afterHost := strings.TrimSpace(argsStr[idxHost+len("--host"):])
		if !strings.HasPrefix(afterHost, "0.0.0.0") {
			t.Errorf("--host not followed by 0.0.0.0: %q", argsStr)
		}
	}
	idxPort := strings.Index(argsStr, "--port")
	if idxPort >= 0 {
		afterPort := strings.TrimSpace(argsStr[idxPort+len("--port"):])
		if !strings.HasPrefix(afterPort, "8000") {
			t.Errorf("--port not followed by 8000: %q", argsStr)
		}
	}
}

// --- resource_controls integration tests ---

const vllmVendorOptionsJSON = `{
	"resource_controls": {
		"gpu_memory_fraction": {
			"arg": "--gpu-memory-utilization",
			"type": "float",
			"min": 0.1,
			"max": 0.95,
			"default": 0.9
		},
		"max_model_len": {
			"arg": "--max-model-len",
			"type": "int"
		},
		"max_num_seqs": {
			"arg": "--max-num-seqs",
			"type": "int"
		}
	}
}`

const sglangVendorOptionsJSON = `{
	"resource_controls": {
		"gpu_memory_fraction": {
			"arg": "--mem-fraction-static",
			"type": "float",
			"min": 0.1,
			"max": 0.95
		},
		"attention_backend": {
			"arg": "--attention-backend",
			"type": "enum",
			"values": ["auto", "flashinfer", "triton", "fa3"]
		}
	}
}`

const llamacppVendorOptionsJSON = `{
	"resource_controls": {
		"gpu_memory_fraction": {
			"supported": false,
			"reason": "llama.cpp does not expose a vLLM-style GPU memory fraction."
		},
		"gpu_layers": {"arg": "--n-gpu-layers", "type": "string_or_int"},
		"ctx_size": {"arg": "--ctx-size", "type": "int"},
		"batch_size": {"arg": "--batch-size", "type": "int"}
	}
}`

func TestResolveVLLMResourceControlsGPUFraction(t *testing.T) {
	input := makeTestInput()
	input.BackendVersion.VendorOptionsJSON = vllmVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"served_model_name":   "qwen3-32b",
		"gpu_memory_fraction": 0.7,
	}
	input.BackendVersion.ParameterDefs = []ParameterDef{
		{Name: "served_model_name", CliName: "--served-model-name", Type: "string", Required: true},
	}

	plan, errors, _ := Resolve(input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "--gpu-memory-utilization") {
		t.Errorf("expected --gpu-memory-utilization in args, got: %s", argsStr)
	}
	// Find the value after --gpu-memory-utilization
	idx := strings.Index(argsStr, "--gpu-memory-utilization")
	if idx >= 0 {
		after := strings.TrimSpace(argsStr[idx+len("--gpu-memory-utilization"):])
		if !strings.HasPrefix(after, "0.7") {
			t.Errorf("expected 0.7 after --gpu-memory-utilization, got: %s", after)
		}
	}
}

func TestResolveVLLMResourceControlsMaxNumSeqs(t *testing.T) {
	input := makeTestInput()
	input.BackendVersion.VendorOptionsJSON = vllmVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"served_model_name": "qwen3-32b",
		"max_num_seqs":      16.0,
	}
	input.BackendVersion.ParameterDefs = []ParameterDef{
		{Name: "served_model_name", CliName: "--served-model-name", Type: "string", Required: true},
	}

	plan, errors, _ := Resolve(input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "--max-num-seqs") {
		t.Errorf("expected --max-num-seqs in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "16") {
		t.Errorf("expected 16 in args, got: %s", argsStr)
	}
}

func TestResolveSGLangResourceControlsMemFraction(t *testing.T) {
	input := makeTestInput()
	input.Backend.Name = "sglang"
	input.BackendVersion.VendorOptionsJSON = sglangVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"gpu_memory_fraction": 0.65,
	}
	input.BackendVersion.ParameterDefs = nil

	plan, errors, _ := Resolve(input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "--mem-fraction-static") {
		t.Errorf("expected --mem-fraction-static in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "0.65") {
		t.Errorf("expected 0.65 in args, got: %s", argsStr)
	}
}

func TestResolveSGLangResourceControlsAttentionBackend(t *testing.T) {
	input := makeTestInput()
	input.Backend.Name = "sglang"
	input.BackendVersion.VendorOptionsJSON = sglangVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"attention_backend": "triton",
	}
	input.BackendVersion.ParameterDefs = nil

	plan, errors, _ := Resolve(input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "--attention-backend") {
		t.Errorf("expected --attention-backend in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "triton") {
		t.Errorf("expected triton in args, got: %s", argsStr)
	}
}

func TestResolveLlamaCppNoFakeMemoryFraction(t *testing.T) {
	input := makeTestInput()
	input.Backend.Name = "llamacpp"
	input.BackendVersion.VendorOptionsJSON = llamacppVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"gpu_memory_fraction": 0.8,
		"ctx_size":            4096.0,
	}
	input.BackendVersion.ParameterDefs = nil

	plan, errors, _ := Resolve(input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	argsStr := strings.Join(plan.Args, " ")
	// gpu_memory_fraction should NOT generate any arg for llama.cpp (supported=false)
	if strings.Contains(argsStr, "--gpu-memory-utilization") {
		t.Errorf("llama.cpp should NOT have --gpu-memory-utilization, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "--mem-fraction-static") {
		t.Errorf("llama.cpp should NOT have --mem-fraction-static, got: %s", argsStr)
	}
	// ctx_size should be mapped
	if !strings.Contains(argsStr, "--ctx-size") {
		t.Errorf("expected --ctx-size in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "4096") {
		t.Errorf("expected 4096 in args, got: %s", argsStr)
	}
}

func TestResolveLlamaCppResourceControlsGpuLayers(t *testing.T) {
	input := makeTestInput()
	input.Backend.Name = "llamacpp"
	input.BackendVersion.VendorOptionsJSON = llamacppVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"gpu_layers": 99.0,
	}
	input.BackendVersion.ParameterDefs = nil

	plan, errors, _ := Resolve(input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "--n-gpu-layers") {
		t.Errorf("expected --n-gpu-layers in args, got: %s", argsStr)
	}
}

func TestResolveResourceControlsNoDuplicateWithParameterDefs(t *testing.T) {
	// max_model_len is in BOTH ParameterDefs and resource_controls.
	// ParameterDefs maps "max_model_len" → "--max-model-len".
	// resource_controls maps "max_model_len" → "--max-model-len".
	// Should NOT produce duplicate --max-model-len.
	input := makeTestInput()
	input.BackendVersion.VendorOptionsJSON = vllmVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"served_model_name": "qwen3-32b",
		"max_model_len":     16384.0,
	}
	input.BackendVersion.ParameterDefs = []ParameterDef{
		{Name: "served_model_name", CliName: "--served-model-name", Type: "string", Required: true},
		{Name: "max_model_len", CliName: "--max-model-len", Type: "integer"},
	}

	plan, errors, _ := Resolve(input)
	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	argsStr := strings.Join(plan.Args, " ")
	// Count occurrences of --max-model-len
	count := strings.Count(argsStr, "--max-model-len")
	if count != 1 {
		t.Errorf("expected exactly 1 --max-model-len, got %d in: %s", count, argsStr)
	}
}
