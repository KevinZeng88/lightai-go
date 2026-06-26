package semanticconfig

import "testing"

func TestSnapshotBuilderCopiesLineageAcrossRuntimeNodeAndDeployment(t *testing.T) {
	reg := DefaultRegistry()
	builder := NewSnapshotBuilder(reg)
	versionSet := map[string]any{
		"schema_version": 1,
		"items": map[string]any{
			"backend.common.port": map[string]any{"code": "backend.common.port", "type": "integer", "value": 8000},
			"backend.arg.gpu_memory_utilization": map[string]any{
				"code":  "backend.arg.gpu_memory_utilization",
				"type":  "number",
				"value": 0.9,
			},
		},
	}

	runtime, err := builder.BuildBackendRuntimeSnapshot(BuildInput{
		SourceKind: "BackendVersion",
		SourceID:   "version.vllm",
		TargetKind: "BackendRuntime",
		TargetID:   "runtime.vllm",
		ConfigSet:  versionSet,
		Values: map[string]any{
			"runtime.image_ref": "vllm:test",
		},
	})
	if err != nil {
		t.Fatalf("build runtime snapshot: %v", err)
	}
	node, err := builder.BuildNodeBackendRuntimeSnapshot(BuildInput{
		SourceKind: "BackendRuntime",
		SourceID:   "runtime.vllm",
		TargetKind: "NodeBackendRuntime",
		TargetID:   "nbr.vllm",
		Snapshot:   runtime,
		Values: map[string]any{
			"runtime.image_ref": "vllm:node",
		},
	})
	if err != nil {
		t.Fatalf("build node snapshot: %v", err)
	}
	deployment, err := builder.BuildDeploymentSnapshot(DeploymentBuildInput{
		SourceKind: "NodeBackendRuntime",
		SourceID:   "nbr.vllm",
		TargetID:   "dep.vllm",
		Snapshot:   node,
		Service: ServiceInput{
			HostPort:        18000,
			ContainerPort:   8000,
			ServedModelName: "llama",
		},
		ModelFacts: ModelFacts{
			ContextLength: 4096,
		},
		Values: map[string]any{
			"model_runtime.max_model_len": 2048,
		},
	})
	if err != nil {
		t.Fatalf("build deployment snapshot: %v", err)
	}

	if got := deployment.Items["deployment.host_port"].Value; got != 18000 {
		t.Fatalf("deployment.host_port = %#v, want 18000", got)
	}
	if got := deployment.Items["service.container_port"].Value; got != 8000 {
		t.Fatalf("service.container_port = %#v, want 8000", got)
	}
	if got := deployment.Items["deployment.served_model_name"].Value; got != "llama" {
		t.Fatalf("deployment.served_model_name = %#v, want llama", got)
	}
	if got := deployment.Items["model_runtime.context_length"].Value; got != 4096 {
		t.Fatalf("model_runtime.context_length = %#v, want 4096", got)
	}
	if got := deployment.Items["model_runtime.max_model_len"].Value; got != 2048 {
		t.Fatalf("model_runtime.max_model_len = %#v, want 2048", got)
	}
	if item := deployment.Items["model_runtime.max_model_len"]; item.CopiedFrom == "" || item.SourceSnapshot == "" || item.CopiedAt == "" || item.Dirty {
		t.Fatalf("deployment max_model_len lineage not initialized: %#v", item)
	}

	changed, err := ApplyPatch(reg, deployment, []PatchField{{Key: "model_runtime.max_model_len", Value: 1024}})
	if err != nil {
		t.Fatalf("patch deployment snapshot: %v", err)
	}
	if got := changed.Items["model_runtime.max_model_len"].Value; got != 1024 {
		t.Fatalf("patched deployment max_model_len = %#v, want 1024", got)
	}
	if !changed.Items["model_runtime.max_model_len"].Dirty {
		t.Fatalf("patched item should be dirty: %#v", changed.Items["model_runtime.max_model_len"])
	}
	if got := deployment.Items["model_runtime.max_model_len"].Value; got != 2048 {
		t.Fatalf("original deployment snapshot mutated: %#v", got)
	}
	if got := node.Items["runtime.image_ref"].Value; got != "vllm:node" {
		t.Fatalf("node snapshot mutated unexpectedly: %#v", got)
	}
	if got := runtime.Items["runtime.image_ref"].Value; got != "vllm:test" {
		t.Fatalf("runtime snapshot mutated unexpectedly: %#v", got)
	}
}

func TestDerivedServiceJSONComesFromSemanticSnapshot(t *testing.T) {
	service := DerivedServiceJSON(Snapshot{
		Items: map[string]SnapshotItem{
			"deployment.host_port":         {Key: "deployment.host_port", Value: 18080},
			"service.container_port":       {Key: "service.container_port", Value: 8000},
			"deployment.served_model_name": {Key: "deployment.served_model_name", Value: "served"},
		},
	})
	if service["host_port"] != 18080 || service["container_port"] != 8000 || service["served_model_name"] != "served" {
		t.Fatalf("unexpected derived service json: %#v", service)
	}
}
