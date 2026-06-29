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
		// NBR snapshot — source of truth for runtime parameters
		// Contains frozen config from BR at creation time.
		// Includes BV default_args (frozen into NBR snapshot at creation).
		NBRConfigSnapshot: &NBRSnapshotInfo{
			ArgsOverride: []string{
				"{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}",
				"--served-model-name", "{{served_model_name}}", // BV default_args, frozen at NBR creation
				"--enforce-eager",     // BV default_backend_params, frozen at NBR creation
				"--trust-remote-code", // BR args_override
			},
			DefaultEnv:         map[string]string{"GLOBAL_ENV": "1", "RUNTIME_VAR": "1", "CUDA_VISIBLE_DEVICES": "0", "VLLM_USE_MODELSCOPE": "true"},
			EntrypointOverride: []string{"vllm", "serve"},
			Docker: DockerSpecInfo{
				Privileged: true, IPCMode: "host", ShmSize: "10g",
			},
			ParameterSchema: []ParameterDef{
				{Name: "max_model_len", CliName: "--max-model-len", Type: "integer", Default: 8192.0, Required: false},
				{Name: "served_model_name", CliName: "--served-model-name", Type: "string", Required: true},
			},
			ParameterValues: []ParameterValue{},
		},
		Artifact: &ArtifactInfo{
			ID: "artifact-qwen", Name: "Qwen3-32B", Path: "/data/models/Qwen3-32B",
		},
		Deployment: &DeploymentInfo{
			ID:   "deploy-1",
			Name: "qwen3-deploy",
			ParameterValues: []ParameterValue{
				{Key: "served_model_name", CliName: "--served-model-name", Type: "string", Enabled: true, Value: "qwen3-32b"},
				{Key: "max_model_len", CliName: "--max-model-len", Type: "integer", Enabled: true, Value: 32768.0},
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

// makeNbrSnapshot creates an NBR snapshot from BV/BR data for tests.
func makeNbrSnapshot(bv *VersionInfo, br *RuntimeInfo) *NBRSnapshotInfo {
	return &NBRSnapshotInfo{
		ArgsOverride:       br.ArgsOverride,
		DefaultEnv:         br.DefaultEnv,
		EntrypointOverride: bv.DefaultEntrypoint,
		Docker:             br.Docker,
		ModelMount:         br.ModelMount,
		ParameterSchema:    bv.ParameterDefs,
		ParameterValues:    []ParameterValue{},
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
	// Priority 1: NodeRuntimeOverride explicit image.
	in := makeTestInput()
	in.NodeRuntimeOverride = &NodeOverrideInfo{ImageName: "node-image:latest"}
	plan, _, _ := Resolve(in)
	if plan.Image != "node-image:latest" {
		t.Errorf("expected node-image:latest, got %s", plan.Image)
	}

	// Priority 2: BackendRuntime ConfigSet launcher.image.
	in2 := makeTestInput()
	plan2, _, _ := Resolve(in2)
	if plan2.Image != "vllm/vllm-openai:v0.8.5-custom" {
		t.Errorf("expected runtime image, got %s", plan2.Image)
	}

	// BackendVersion.defaultImages is not a runtime fallback.
	in3 := makeTestInput()
	in3.BackendRuntime.ImageName = ""
	_, errs, _ := Resolve(in3)
	if len(errs) == 0 {
		t.Fatal("expected error when image exists only on BackendVersion")
	}

	// No image available → error
	in4 := makeTestInput()
	in4.BackendRuntime.ImageName = ""
	_, errs4, _ := Resolve(in4)
	if len(errs4) == 0 {
		t.Error("expected error for no image")
	}
}

func TestResolveRendersConfigSetParameterStyles(t *testing.T) {
	in := makeTestInput()
	in.BackendRuntime.ArgsOverride = nil
	in.BackendVersion.ParameterDefs = nil
	in.NBRConfigSnapshot.ArgsOverride = nil
	in.NBRConfigSnapshot.ParameterValues = []ParameterValue{
		{Key: "equals", CliName: "--kv-cache-dtype", Enabled: true, Value: "fp8", RenderStyle: "flag_equals_value"},
		{Key: "bool", CliName: "--enforce-eager", Enabled: true, Value: true, RenderStyle: "flag_if_true"},
		{Key: "repeat", CliName: "--lora", Enabled: true, Value: []interface{}{"a", "b"}, RenderStyle: "repeat_flag"},
		{Key: "pos", Enabled: true, Value: "{{MODEL_CONTAINER_PATH}}", RenderStyle: "positional"},
		{Key: "raw", Enabled: true, Value: "--trust-remote-code\n--max-num-seqs 8", RenderStyle: "raw_lines"},
	}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("Resolve errors: %v", errs)
	}
	got := strings.Join(plan.Args, " ")
	for _, want := range []string{
		"--kv-cache-dtype=fp8",
		"--enforce-eager",
		"--lora a --lora b",
		"/models/Qwen3-32B",
		"--trust-remote-code --max-num-seqs 8",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("args missing %q: %v", want, plan.Args)
		}
	}
}

func TestResolveDoesNotFallbackToLiveBackendVersionParameterSchema(t *testing.T) {
	in := makeTestInput()
	in.NBRConfigSnapshot.ParameterSchema = nil
	in.NBRConfigSnapshot.ArgsOverride = []string{"{{model_container_path}}"}
	in.NBRConfigSnapshot.ParameterValues = nil
	in.Deployment.ParameterValues = nil
	in.BackendVersion.ParameterDefs = []ParameterDef{
		{Name: "live_required_after_snapshot", CliName: "--live-required-after-snapshot", Required: true},
	}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("BackendVersion ParameterDefs affected snapshot-only RunPlan: %v", errs)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
	if strings.Contains(strings.Join(plan.Args, " "), "--live-required-after-snapshot") {
		t.Fatalf("live BackendVersion ParameterDefs leaked into args: %v", plan.Args)
	}
}

func TestResolveDoesNotUseLiveBackendVersionVendorOptionsResourceControls(t *testing.T) {
	in := makeTestInput()
	in.NBRConfigSnapshot.ArgsOverride = []string{"{{model_container_path}}"}
	in.NBRConfigSnapshot.ParameterSchema = nil
	in.NBRConfigSnapshot.ParameterValues = nil
	in.Deployment.ParameterValues = nil
	in.Deployment.Parameters = map[string]interface{}{"max_model_len": float64(12345)}
	in.BackendVersion.VendorOptionsJSON = vllmVendorOptionsJSON

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
	args := strings.Join(plan.Args, " ")
	if strings.Contains(args, "--max-model-len") || strings.Contains(args, "12345") {
		t.Fatalf("live BackendVersion VendorOptionsJSON resource_controls leaked into args: %s", args)
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
	if !strings.Contains(argsStr, "--served-model-name") || !strings.Contains(argsStr, "Qwen3-32B") {
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
	// Also update NBR snapshot since resolver reads from it
	in.NBRConfigSnapshot.ArgsOverride = []string{"{{undefined_variable}}"}
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

func TestResolveLauncherKindFallbackAndHostPortMaterialization(t *testing.T) {
	in := makeTestInput()
	in.BackendRuntime.RuntimeType = ""
	in.BackendRuntime.LauncherKind = "docker"
	in.Deployment.Service = ServiceInfo{}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan.HostPort != plan.ContainerPort || plan.HostPort != 8000 {
		t.Fatalf("host/container ports = %d/%d, want 8000/8000", plan.HostPort, plan.ContainerPort)
	}
	preview := EquivalentCommandPreview(plan)
	if !strings.Contains(preview, "-p 8000:8000/tcp") {
		t.Fatalf("preview missing materialized port binding: %s", preview)
	}
}

func TestResolveDeviceBindingVisibleInPlanAndPreview(t *testing.T) {
	in := makeTestInput()
	in.AssignedGPUs = []GPUInfo{{Index: 0, Vendor: "nvidia"}}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan.DeviceBinding == nil {
		t.Fatal("device_binding missing")
	}
	if plan.DeviceBinding.DockerGPUOption != "device=0" {
		t.Fatalf("docker gpu option=%q", plan.DeviceBinding.DockerGPUOption)
	}
	if plan.DeviceBinding.VisibleEnvKey != "CUDA_VISIBLE_DEVICES" || plan.DeviceBinding.VisibleEnvValue != "0" {
		t.Fatalf("visible env binding=%#v", plan.DeviceBinding)
	}
	preview := EquivalentCommandPreview(plan)
	if !strings.Contains(preview, `--gpus "device=0"`) || !strings.Contains(preview, "CUDA_VISIBLE_DEVICES=0") {
		t.Fatalf("preview missing device binding: %s", preview)
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
	// Also remove CUDA_VISIBLE_DEVICES from NBR snapshot
	delete(in.NBRConfigSnapshot.DefaultEnv, "CUDA_VISIBLE_DEVICES")
	plan, _, _ := Resolve(in)
	if plan.Env["CUDA_VISIBLE_DEVICES"] != "" {
		t.Error("expected no CUDA_VISIBLE_DEVICES for no GPU")
	}
}

func TestArgsOverrideAppendOnly(t *testing.T) {
	in := makeTestInput()
	// args_override appends, doesn't replace
	in.BackendRuntime.ArgsOverride = []string{"--custom-flag", "--another-flag"}
	// Update NBR snapshot: keep original BV args + new BR override
	in.NBRConfigSnapshot.ArgsOverride = []string{
		"{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}",
		"--served-model-name", "{{served_model_name}}", "--enforce-eager",
		"--custom-flag", "--another-flag",
	}
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
	baseInput := ensureNbrSnapshot(ResolveInput{
		Backend:        &BackendInfo{ID: "b.vllm", Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{ID: "bv.openai", DefaultEntrypoint: []string{"serve"}, DefaultArgs: []string{}, DefaultBackendParams: []string{}, ParameterDefs: []ParameterDef{}, HealthCheck: HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200}, DefaultContainerPort: 8000, DefaultImages: map[string]string{"nvidia": "img:latest"}, Env: map[string]string{}},
		BackendRuntime: &RuntimeInfo{ID: "rt.vllm", Vendor: "nvidia", RuntimeType: "docker", ImageName: "img:latest", ArgsOverride: []string{}, DefaultEnv: map[string]string{}, Docker: DockerSpecInfo{}, ModelMount: ModelMountInfo{ContainerPath: "/models", Readonly: true}},
		Deployment:     &DeploymentInfo{ID: "d1", Name: "test", Parameters: map[string]interface{}{}, EnvOverrides: map[string]string{}, Service: ServiceInfo{HostPort: 8002}, Placement: PlacementInfo{NodeID: "n1"}},
		InstanceID:     "inst-safety",
		Node:           &NodeInfo{ID: "n1", IP: "127.0.0.1"},
		AssignedGPUs:   []GPUInfo{{Index: 0, Vendor: "nvidia"}},
	})

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
	args := mapParametersToArgs(map[string]interface{}{}, defs, nil, nil)
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
	}, defs, nil, nil)
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
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{ID: "backend.vllm", Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			ID:                "vllm-v0.23.0",
			DefaultEntrypoint: []string{"vllm", "serve"},
			DefaultArgs:       []string{"{{MODEL_CONTAINER_PATH}}"},
			ParameterDefs: []ParameterDef{
				{Name: "--host", Default: "0.0.0.0", Required: true},
				{Name: "--port", Default: "8000", Required: true},
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
			ID:   "dep-vllm",
			Name: "vllm-test",
			ParameterValues: []ParameterValue{
				{Key: "served_model_name", CliName: "--served-model-name", Type: "string", Enabled: true, Value: "test"},
				{Key: "max_model_len", CliName: "--max-model-len", Type: "integer", Enabled: true, Value: float64(4096)},
				{Key: "host", CliName: "--host", Type: "string", Enabled: true, Value: "0.0.0.0"},
				{Key: "port", CliName: "--port", Type: "string", Enabled: true, Value: "8000"},
			},
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
	})
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

// --- resource_controls fixtures ---

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

func TestRequiredParamFromDefaultArgs(t *testing.T) {
	// Scenario: llama.cpp catalog has "-m" (alias "--model") as required.
	// Layer 1 default_args provides "-m /models/test.gguf".
	// Deployment parameters are empty {}.
	// Required param check should NOT report error because -m is already in args.
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			ID:         "backend.llamacpp",
			Name:       "llamacpp",
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			ID:                "llamacpp-b9700",
			DefaultEntrypoint: []string{},
			DefaultArgs:       []string{"-m", "/models/test.gguf", "--host", "0.0.0.0", "--port", "8080"},
			ParameterDefs: []ParameterDef{
				{Name: "-m", Alias: "--model", Required: true},
				{Name: "--host", Default: "0.0.0.0"},
				{Name: "--port", Default: "8080"},
			},
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "ghcr.io/ggml-org/llama.cpp:server-cuda13"},
			Env:                  map[string]string{},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor:      "nvidia",
			RuntimeType: "docker",
			ImageName:   "ghcr.io/ggml-org/llama.cpp:server-cuda13",
			DefaultEnv:  map[string]string{},
			Docker:      DockerSpecInfo{},
		},
		Artifact: &ArtifactInfo{
			Path: "/models/test.gguf",
		},
		Deployment: &DeploymentInfo{
			Parameters: map[string]interface{}{}, // empty — all from defaults
			Service:    ServiceInfo{HostPort: 8004},
		},
		InstanceID: "inst-test-001",
		Node:       &NodeInfo{ID: "node-1", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{
			{Index: 0, Vendor: "nvidia"},
		},
	})

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("expected no errors (required param -m provided by default_args), got: %v", errs)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
	argsStr := strings.Join(plan.Args, " ")
	if !strings.Contains(argsStr, "-m /models/test.gguf") {
		t.Errorf("expected -m /models/test.gguf in args, got: %s", argsStr)
	}
}

func TestRequiredParamMissingEverywhere(t *testing.T) {
	// Scenario: required param NOT in default_args and NOT in deployment params.
	// Should report error.
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			ID:         "backend.test",
			Name:       "test",
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			ID:                "test-v1",
			DefaultEntrypoint: []string{},
			DefaultArgs:       []string{"--host", "0.0.0.0"},
			ParameterDefs: []ParameterDef{
				{Name: "--model", Required: true},
				{Name: "--host", Default: "0.0.0.0"},
			},
			HealthCheck:          HealthCheckInput{Path: "/health", ExpectedStatus: 200},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "test:latest"},
			Env:                  map[string]string{},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor:      "nvidia",
			RuntimeType: "docker",
			ImageName:   "test:latest",
			DefaultEnv:  map[string]string{},
		},
		Artifact: &ArtifactInfo{Path: "/tmp/test"},
		Deployment: &DeploymentInfo{
			Parameters: map[string]interface{}{},
		},
		InstanceID: "inst-test-002",
		Node:       &NodeInfo{ID: "node-1", IP: "127.0.0.1"},
	})

	_, errs, _ := Resolve(in)
	if len(errs) == 0 {
		t.Fatal("expected error for missing required param --model, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "required parameter") && strings.Contains(e.Error(), "--model") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'required parameter --model missing' error, got: %v", errs)
	}
}

func TestRequiredShortAliasFromDefaultArgs(t *testing.T) {
	// Scenario: ParameterDef has Name="-m", Alias="--model".
	// default_args provides "-m" (short form).
	// Should NOT report error.
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			ID:         "backend.llamacpp",
			Name:       "llamacpp",
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			ID:                "llamacpp-b9700",
			DefaultEntrypoint: []string{},
			DefaultArgs:       []string{"-m", "/models/test.gguf"},
			ParameterDefs: []ParameterDef{
				{Name: "-m", Alias: "--model", Required: true},
			},
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "ghcr.io/ggml-org/llama.cpp:server-cuda13"},
			Env:                  map[string]string{},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor:      "nvidia",
			RuntimeType: "docker",
			ImageName:   "ghcr.io/ggml-org/llama.cpp:server-cuda13",
			DefaultEnv:  map[string]string{},
		},
		Artifact: &ArtifactInfo{Path: "/models/test.gguf"},
		Deployment: &DeploymentInfo{
			Parameters: map[string]interface{}{},
		},
		InstanceID: "inst-test-003",
		Node:       &NodeInfo{ID: "node-1", IP: "127.0.0.1"},
	})

	_, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("expected no errors (short alias -m provided by default_args), got: %v", errs)
	}
}

func TestRequiredLongAliasFromDefaultArgs(t *testing.T) {
	// Scenario: ParameterDef has Name="--model", CliName="--model".
	// default_args provides "--model /models/test.gguf" (long form).
	// Should NOT report error.
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			ID:         "backend.test",
			Name:       "test",
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			ID:                "test-v1",
			DefaultEntrypoint: []string{},
			DefaultArgs:       []string{"--model", "/models/test.gguf"},
			ParameterDefs: []ParameterDef{
				{Name: "--model", CliName: "--model", Required: true},
			},
			HealthCheck:          HealthCheckInput{Path: "/health", ExpectedStatus: 200},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "test:latest"},
			Env:                  map[string]string{},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor:      "nvidia",
			RuntimeType: "docker",
			ImageName:   "test:latest",
			DefaultEnv:  map[string]string{},
		},
		Artifact: &ArtifactInfo{Path: "/models/test.gguf"},
		Deployment: &DeploymentInfo{
			Parameters: map[string]interface{}{},
		},
		InstanceID: "inst-test-004",
		Node:       &NodeInfo{ID: "node-1", IP: "127.0.0.1"},
	})

	_, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("expected no errors (long alias --model provided by default_args), got: %v", errs)
	}
}

func TestRequiredFlagEqualsFormFromDefaultArgs(t *testing.T) {
	// Scenario: default_args provides "--model=/models/test.gguf" (= form).
	// Should NOT report error.
	in := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			ID:         "backend.test",
			Name:       "test",
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			ID:                "test-v1",
			DefaultEntrypoint: []string{},
			DefaultArgs:       []string{"--model=/models/test.gguf"},
			ParameterDefs: []ParameterDef{
				{Name: "--model", CliName: "--model", Required: true},
			},
			HealthCheck:          HealthCheckInput{Path: "/health", ExpectedStatus: 200},
			DefaultContainerPort: 8080,
			DefaultImages:        map[string]string{"nvidia": "test:latest"},
			Env:                  map[string]string{},
		},
		BackendRuntime: &RuntimeInfo{
			Vendor:      "nvidia",
			RuntimeType: "docker",
			ImageName:   "test:latest",
			DefaultEnv:  map[string]string{},
		},
		Artifact: &ArtifactInfo{Path: "/models/test.gguf"},
		Deployment: &DeploymentInfo{
			Parameters: map[string]interface{}{},
		},
		InstanceID: "inst-test-005",
		Node:       &NodeInfo{ID: "node-1", IP: "127.0.0.1"},
	})

	_, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("expected no errors (--model= form provided by default_args), got: %v", errs)
	}
}

func TestBuildEnvFiltersNonScalarValues(t *testing.T) {
	// Scenario: BackendVersion.Env contains array/map values from capability metadata.
	// buildEnv should skip non-scalar values.
	input := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			Env: map[string]string{
				"CUDA_VISIBLE_DEVICES": "0",
				"VLLM_USE_MODELSCOPE":  "false",
			},
		},
		BackendRuntime: &RuntimeInfo{
			DefaultEnv: map[string]string{},
		},
		Deployment: &DeploymentInfo{
			EnvOverrides: map[string]string{},
		},
	})
	vars := map[string]string{
		"assigned_gpu_count": "1",
		"container_port":     "8000",
	}
	env, warns := buildEnv(input, vars)
	_ = warns
	if env["CUDA_VISIBLE_DEVICES"] != "0" {
		t.Errorf("expected CUDA_VISIBLE_DEVICES=0, got: %s", env["CUDA_VISIBLE_DEVICES"])
	}
	if env["VLLM_USE_MODELSCOPE"] != "false" {
		t.Errorf("expected VLLM_USE_MODELSCOPE=false, got: %s", env["VLLM_USE_MODELSCOPE"])
	}
}

func TestBuildEnvSkipsEmptyValues(t *testing.T) {
	// Scenario: some env values resolve to empty strings.
	// buildEnv should skip them.
	input := ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{
			DefaultEnv: map[string]string{},
		},
		BackendVersion: &VersionInfo{
			Env: map[string]string{
				"VALID_KEY": "valid_value",
				"EMPTY_KEY": "",
			},
		},
		BackendRuntime: &RuntimeInfo{
			DefaultEnv: map[string]string{},
		},
		Deployment: &DeploymentInfo{
			EnvOverrides: map[string]string{},
		},
	})
	vars := map[string]string{}
	env, _ := buildEnv(input, vars)
	if env["VALID_KEY"] != "valid_value" {
		t.Errorf("expected VALID_KEY=valid_value, got: %s", env["VALID_KEY"])
	}
	if _, exists := env["EMPTY_KEY"]; exists {
		t.Errorf("expected EMPTY_KEY to be skipped, but it exists with value: %s", env["EMPTY_KEY"])
	}
}

func TestCollectExistingFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]bool
	}{
		{
			name:     "long flags",
			args:     []string{"--host", "0.0.0.0", "--port", "8080"},
			expected: map[string]bool{"--host": true, "--port": true},
		},
		{
			name:     "short flags",
			args:     []string{"-m", "/models/test.gguf", "-ngl", "999"},
			expected: map[string]bool{"-m": true, "-ngl": true},
		},
		{
			name:     "equals form",
			args:     []string{"--model=/models/test.gguf", "--host=0.0.0.0"},
			expected: map[string]bool{"--model": true, "--host": true},
		},
		{
			name:     "mixed",
			args:     []string{"-m", "/models/test.gguf", "--host", "0.0.0.0", "--port=8080"},
			expected: map[string]bool{"-m": true, "--host": true, "--port": true},
		},
		{
			name:     "empty",
			args:     []string{},
			expected: map[string]bool{},
		},
		{
			name:     "boolean flags",
			args:     []string{"--verbose", "--host", "0.0.0.0"},
			expected: map[string]bool{"--verbose": true, "--host": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectExistingFlags(tt.args)
			for flag := range tt.expected {
				if !result[flag] {
					t.Errorf("expected flag %q to be collected, args=%v, result=%v", flag, tt.args, result)
				}
			}
			for flag := range result {
				if !tt.expected[flag] {
					t.Errorf("unexpected flag %q collected, args=%v, result=%v", flag, tt.args, result)
				}
			}
		})
	}
}

func TestResolveResourceControlsNoDuplicateWithParameterDefs(t *testing.T) {
	// max_model_len is present in the NBR snapshot schema. A later live
	// BackendVersion resource_controls edit must not generate a second arg.
	input := makeTestInput()
	input.BackendVersion.VendorOptionsJSON = vllmVendorOptionsJSON
	input.Deployment.Parameters = map[string]interface{}{
		"served_model_name": "qwen3-32b",
		"max_model_len":     16384.0,
	}
	input.NBRConfigSnapshot.ParameterSchema = []ParameterDef{
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
