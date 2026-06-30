package runplan

import (
	"encoding/json"
	"testing"
)

func TestSourceMapBuilderRecordsAllTargets(t *testing.T) {
	b := NewSourceMapBuilder()

	b.AddArg("gpu_memory_utilization", "--gpu-memory-utilization", 0.82, "deployment_local_edit",
		"BackendParameterConfigSet", "DeploymentConfigBundle",
		[]SourceChainEntry{
			{Layer: "BackendVersionConfigBundle", Value: 0.9, Reason: "schema default"},
			{Layer: "NodeBackendRuntimeConfigBundle", Value: 0.85, Reason: "node local edit"},
			{Layer: "DeploymentConfigBundle", Value: 0.82, Reason: "deployment local edit"},
		})

	b.AddEnv("CUDA_VISIBLE_DEVICES", "0,1", "system_generated",
		"", "SystemGenerated",
		[]SourceChainEntry{
			{Layer: "SystemGenerated", Value: "0,1", Reason: "gpu assignment"},
		})

	b.AddDockerOption("docker.shm_size", "2gb", "backend_runtime",
		"RuntimeDockerConfigSet", "BackendRuntimeConfigBundle",
		[]SourceChainEntry{
			{Layer: "BackendRuntimeConfigBundle", Value: "2gb", Reason: "runtime template default"},
		})

	b.AddMount("model_mount", "/models/Qwen2.5-7B", "model_location",
		"ModelLocationConfigSet", "ModelLocationConfigBundle",
		nil)

	b.AddPort("service.container_port", int(8000), "backend_version",
		"BackendVersionConfigSet", "BackendVersionConfigBundle",
		[]SourceChainEntry{
			{Layer: "BackendVersionConfigBundle", Value: 8000, Reason: "catalog default"},
		})

	b.AddHealthCheck("health_check.path", "/v1/models", "backend_runtime",
		"RuntimeHealthCheckConfigSet", "BackendRuntimeConfigBundle",
		nil)

	sm := b.Build()

	// Args
	if len(sm.Args) != 1 {
		t.Errorf("Args count = %d, want 1", len(sm.Args))
	} else {
		a := sm.Args[0]
		if a.EffectiveSource != "deployment_local_edit" {
			t.Errorf("arg effective_source = %q", a.EffectiveSource)
		}
		if len(a.SourceChain) != 3 {
			t.Errorf("arg source chain length = %d, want 3", len(a.SourceChain))
		}
	}

	// Env
	if len(sm.Env) != 1 {
		t.Errorf("Env count = %d, want 1", len(sm.Env))
	}

	// Docker options
	if len(sm.DockerOptions) != 1 {
		t.Errorf("DockerOptions count = %d, want 1", len(sm.DockerOptions))
	}

	// Mounts
	if len(sm.Mounts) != 1 {
		t.Errorf("Mounts count = %d, want 1", len(sm.Mounts))
	}

	// Ports
	if len(sm.Ports) != 1 {
		t.Errorf("Ports count = %d, want 1", len(sm.Ports))
	}

	// Health check
	if len(sm.HealthCheck) != 1 {
		t.Errorf("HealthCheck count = %d, want 1", len(sm.HealthCheck))
	}
}

func TestParameterSourceMapJSON(t *testing.T) {
	b := NewSourceMapBuilder()

	b.AddArg("max_model_len", "--max-model-len", int(8192), "deployment_override",
		"BackendParameterConfigSet", "DeploymentConfigBundle",
		[]SourceChainEntry{
			{Layer: "BackendVersionConfigBundle", Value: 4096, Reason: "schema default"},
			{Layer: "DeploymentConfigBundle", Value: 8192, Reason: "deployment override"},
		})

	b.AddDockerOption("docker.shm_size", "1gb", "backend_runtime",
		"RuntimeDockerConfigSet", "BackendRuntimeConfigBundle",
		[]SourceChainEntry{
			{Layer: "BackendRuntimeConfigBundle", Value: "1gb", Reason: "runtime template"},
		})

	b.AddDockerOption("docker.ipc_mode", "host", "backend_runtime",
		"RuntimeDockerConfigSet", "BackendRuntimeConfigBundle",
		[]SourceChainEntry{
			{Layer: "BackendRuntimeConfigBundle", Value: "host", Reason: "runtime template"},
		})

	b.AddSystemGenerated("gpu_device_ids", "0,1", "system_generated",
		"", "SystemGenerated",
		[]SourceChainEntry{
			{Layer: "SystemGenerated", Value: "0,1", Reason: "gpu scheduler assignment"},
		})

	sm := b.Build()

	data, err := json.Marshal(sm)
	if err != nil {
		t.Fatalf("marshal source map: %v", err)
	}

	var decoded ParameterSourceMap
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal source map: %v", err)
	}

	if len(decoded.Args) != 1 {
		t.Errorf("decoded args = %d", len(decoded.Args))
	}
	if decoded.Args[0].Key != "max_model_len" {
		t.Errorf("decoded arg key = %q", decoded.Args[0].Key)
	}
	if len(decoded.Args[0].SourceChain) != 2 {
		t.Errorf("decoded arg source chain = %d", len(decoded.Args[0].SourceChain))
	}
	if len(decoded.DockerOptions) != 2 {
		t.Errorf("decoded docker options = %d", len(decoded.DockerOptions))
	}
	if len(decoded.SystemGenerated) != 1 {
		t.Errorf("decoded system generated = %d", len(decoded.SystemGenerated))
	}
}

func TestResolvedRunPlanNowIncludesSourceMap(t *testing.T) {
	sm := &ParameterSourceMap{
		Args: []ParameterSourceEntry{
			{Key: "gpu_memory_utilization", Target: "args", Arg: "--gpu-memory-utilization", Value: 0.9, EffectiveSource: "backend_version"},
		},
	}

	plan := ResolvedRunPlan{
		Image:              "vllm/vllm-openai:latest",
		Args:               []string{"--gpu-memory-utilization", "0.9"},
		ParameterSourceMap: sm,
		PlanHash:           "abc123",
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan with source map: %v", err)
	}

	var decoded ResolvedRunPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ParameterSourceMap == nil {
		t.Fatal("parameter_source_map was nil after round-trip")
	}
	if len(decoded.ParameterSourceMap.Args) != 1 {
		t.Errorf("source map args = %d", len(decoded.ParameterSourceMap.Args))
	}
}

func TestDockerOptionUncheckedNotInSourceMap(t *testing.T) {
	// Simulate: optional Docker item with enabled=false should not appear in
	// the docker_options source map entries (resolver filters them before building).
	// This test verifies the builder shape is correct — the filtering logic
	// lives in the resolver.

	b := NewSourceMapBuilder()

	// Only checked/enabled items are added
	b.AddDockerOption("docker.shm_size", "1gb", "backend_runtime",
		"RuntimeDockerConfigSet", "BackendRuntimeConfigBundle",
		[]SourceChainEntry{
			{Layer: "BackendRuntimeConfigBundle", Value: "1gb", Reason: "runtime template"},
		})

	// "docker.extra_hosts" is NOT added because it's unchecked (simulated)

	sm := b.Build()

	// extra_hosts should not appear
	for _, e := range sm.DockerOptions {
		if e.Key == "docker.extra_hosts" {
			t.Error("unchecked docker option should not be in source map")
		}
	}

	if len(sm.DockerOptions) != 1 {
		t.Errorf("docker options count = %d, want 1 (only checked)", len(sm.DockerOptions))
	}
}

func TestSourceMapCoversAllRequiredTargets(t *testing.T) {
	b := NewSourceMapBuilder()

	// Simulate a full resolution
	b.AddArg("gpu_memory_utilization", "--gpu-memory-utilization", 0.9, "backend_version", "", "", nil)
	b.AddArg("max_model_len", "--max-model-len", int(4096), "backend_version", "", "", nil)
	b.AddEnv("CUDA_VISIBLE_DEVICES", "0,1", "system_generated", "", "", nil)
	b.AddEnv("LIGHTAI_INSTANCE_ID", "inst-123", "system_generated", "", "", nil)
	b.AddMount("model_mount", "/models/Qwen", "model_location", "", "", nil)
	b.AddPort("container_port", int(8000), "backend_version", "", "", nil)
	b.AddDevice("nvidia_gpu", "/dev/nvidia0", "system_generated", "", "", nil)
	b.AddDockerOption("docker.shm_size", "1gb", "backend_runtime", "", "", nil)
	b.AddDockerOption("docker.ipc_mode", "host", "backend_runtime", "", "", nil)
	b.AddHealthCheck("health_check.path", "/v1/models", "backend_runtime", "", "", nil)
	b.AddSystemGenerated("gpu_assignment", "0,1", "system_generated", "", "", nil)

	sm := b.Build()

	// All targets must have at least one entry
	if len(sm.Args) < 2 {
		t.Errorf("args = %d, want >= 2", len(sm.Args))
	}
	if len(sm.Env) < 2 {
		t.Errorf("env = %d, want >= 2", len(sm.Env))
	}
	if len(sm.Mounts) < 1 {
		t.Errorf("mounts = %d, want >= 1", len(sm.Mounts))
	}
	if len(sm.Ports) < 1 {
		t.Errorf("ports = %d, want >= 1", len(sm.Ports))
	}
	if len(sm.Devices) < 1 {
		t.Errorf("devices = %d, want >= 1", len(sm.Devices))
	}
	if len(sm.DockerOptions) < 2 {
		t.Errorf("docker_options = %d, want >= 2", len(sm.DockerOptions))
	}
	if len(sm.HealthCheck) < 1 {
		t.Errorf("health_check = %d, want >= 1", len(sm.HealthCheck))
	}
	if len(sm.SystemGenerated) < 1 {
		t.Errorf("system_generated = %d, want >= 1", len(sm.SystemGenerated))
	}
}

func TestResolveWithSourceMapDoesNotReturnNilMap(t *testing.T) {
	in := ResolveInput{
		Backend:        &BackendInfo{Name: "test", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{Version: "1.0", DefaultImages: map[string]string{"nvidia": "test:latest"}, HealthCheck: HealthCheckInput{}},
		BackendRuntime: &RuntimeInfo{Vendor: "nvidia", RuntimeType: "docker", ImageName: "test:latest", Docker: DockerSpecInfo{}},
		Deployment:     &DeploymentInfo{ID: "d1", Service: ServiceInfo{}},
		Artifact:       &ArtifactInfo{},
		InstanceID:     "i1",
		Node:           &NodeInfo{ID: "n1"},
		AssignedGPUs:   []GPUInfo{{Index: 0, Vendor: "nvidia"}},
		NBRConfigSnapshot: &NBRSnapshotInfo{
			ParameterValues: []ParameterValue{
				{Key: "max_model_len", CliName: "--max-model-len", Value: float64(4096), Enabled: true, Source: "node_backend_runtime", CopiedFrom: "bv-vllm"},
			},
		},
	}
	plan, errs, _ := ResolveWithSourceMap(in)
	if len(errs) > 0 {
		t.Logf("resolve errors (expected with minimal input): %v", errs)
	}
	if plan == nil {
		return
	}
	if plan.ParameterSourceMap == nil {
		t.Error("ParameterSourceMap is nil when plan is non-nil")
	}
}

func TestResolveWithSourceMapEmitsSelfContainedSourceEntries(t *testing.T) {
	in := makeTestInput()
	in.Deployment.Placement = PlacementInfo{NodeID: "node-1", AcceleratorSelectionMode: "auto", AllowAutoSelect: true}

	plan, errs, _ := ResolveWithSourceMap(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if plan.ParameterSourceMap == nil {
		t.Fatal("parameter_source_map missing")
	}
	sm := plan.ParameterSourceMap
	if len(sm.Image) == 0 || sm.Image[0].PatchTarget == "" || sm.Image[0].DockerEffect == "" {
		t.Fatalf("image source entry not self-contained: %#v", sm.Image)
	}
	assertSelfContainedEntries(t, "args", sm.Args)
	assertSelfContainedEntries(t, "env", sm.Env)
	assertSelfContainedEntries(t, "mounts", sm.Mounts)
	assertSelfContainedEntries(t, "ports", sm.Ports)
	assertSelfContainedEntries(t, "docker_options", sm.DockerOptions)
	assertSelfContainedEntries(t, "health_check", sm.HealthCheck)
	if len(sm.SystemGenerated) > 0 {
		assertSelfContainedEntries(t, "system_generated", sm.SystemGenerated)
	}
	foundGPU := false
	for _, e := range sm.DockerOptions {
		if e.Key == "docker.gpus" {
			foundGPU = e.PatchTarget == "runtime.device_binding" && e.DockerEffect == "--gpus" && e.EffectiveSource == "configedit_effect"
		}
	}
	if !foundGPU {
		t.Fatalf("GPU docker effect source entry missing or incomplete: %#v", sm.DockerOptions)
	}
}

func assertSelfContainedEntries(t *testing.T, name string, entries []ParameterSourceEntry) {
	t.Helper()
	if len(entries) == 0 {
		t.Fatalf("%s source entries missing", name)
	}
	for _, e := range entries {
		if e.Key == "" || e.Target == "" || len(e.Path) == 0 || e.EffectiveSource == "" || e.SourceLayer == "" || e.SourceKind == "" || e.Reason == "" {
			t.Fatalf("%s entry is not self-contained: %#v", name, e)
		}
	}
}

func TestSourceMapFromProvenanceTracksSourceChain(t *testing.T) {
	sm := NewSourceMapBuilder()
	chain := []SourceChainEntry{
		{Layer: "BackendVersionConfigBundle", Value: float64(4096), Reason: "schema default"},
		{Layer: "NodeBackendRuntimeConfigBundle", Value: float64(8192), Reason: "nbr override (copied_from=bv-vllm)"},
	}
	sm.AddArg("max_model_len", "--max-model-len", float64(8192), "node_backend_runtime", "BackendParameterConfigSet", "NodeBackendRuntimeConfigBundle", chain)
	gpuChain := []SourceChainEntry{
		{Layer: "SystemGenerated", Value: "0,1", Reason: "gpu scheduler assignment"},
	}
	sm.AddSystemGenerated("gpu_device_ids", "0,1", "system_generated", "", "SystemGenerated", gpuChain)
	built := sm.Build()
	if len(built.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(built.Args))
	}
	a := built.Args[0]
	if a.EffectiveSource != "node_backend_runtime" {
		t.Errorf("effective_source = %q, want node_backend_runtime", a.EffectiveSource)
	}
	if len(a.SourceChain) != 2 {
		t.Errorf("source_chain length = %d, want 2", len(a.SourceChain))
	}
	if a.SourceChain[0].Layer != "BackendVersionConfigBundle" {
		t.Errorf("chain[0].Layer = %q", a.SourceChain[0].Layer)
	}
	if len(built.SystemGenerated) != 1 {
		t.Fatalf("expected 1 system_generated, got %d", len(built.SystemGenerated))
	}
	sg := built.SystemGenerated[0]
	if sg.EffectiveSource != "system_generated" {
		t.Errorf("system_generated effective_source = %q, want system_generated", sg.EffectiveSource)
	}
	if len(sg.SourceChain) == 0 {
		t.Error("system_generated source_chain is empty")
	}
}
