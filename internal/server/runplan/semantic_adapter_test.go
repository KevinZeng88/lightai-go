package runplan

import (
	"strings"
	"testing"

	"lightai-go/internal/server/semanticconfig"
)

func TestSemanticAdapterVLLMMapsCanonicalKeysToRunPlan(t *testing.T) {
	in := semanticAdapterBaseInput("vllm")
	snapshot := semanticAdapterSnapshot()
	in = ApplySemanticSnapshot(in, snapshot, "vllm")

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("resolve errors: %v", errs)
	}
	args := strings.Join(plan.Args, " ")
	for _, want := range []string{"--host 0.0.0.0", "--port 8000", "--max-model-len 4096", "--served-model-name llama"} {
		if !strings.Contains(args, want) {
			t.Fatalf("vllm args missing %q: %s", want, args)
		}
	}
}

func TestSemanticAdapterSGLangAndLlamaCppUseBackendSpecificContextFlags(t *testing.T) {
	for _, tt := range []struct {
		backend string
		flag    string
	}{
		{backend: "sglang", flag: "--context-length"},
		{backend: "llamacpp", flag: "--ctx-size"},
	} {
		t.Run(tt.backend, func(t *testing.T) {
			in := semanticAdapterBaseInput(tt.backend)
			in = ApplySemanticSnapshot(in, semanticAdapterSnapshot(), tt.backend)
			plan, errs, _ := Resolve(in)
			if len(errs) > 0 {
				t.Fatalf("resolve errors: %v", errs)
			}
			args := strings.Join(plan.Args, " ")
			if !strings.Contains(args, tt.flag+" 4096") {
				t.Fatalf("%s args missing %s mapping: %s", tt.backend, tt.flag, args)
			}
		})
	}
}

func semanticAdapterBaseInput(backend string) ResolveInput {
	args := []string{"{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}"}
	if backend == "llamacpp" {
		args = []string{"-m", "{{model_container_file}}", "--host", "0.0.0.0", "--port", "{{container_port}}"}
	}
	return ensureNbrSnapshot(ResolveInput{
		Backend: &BackendInfo{Name: backend},
		BackendVersion: &VersionInfo{
			Version:              "test",
			DefaultEntrypoint:    []string{},
			DefaultArgs:          args,
			DefaultContainerPort: 8000,
			HealthCheck:          HealthCheckInput{Path: "/health", ExpectedStatus: 200},
		},
		BackendRuntime: &RuntimeInfo{
			ID:          "runtime." + backend,
			Vendor:      "nvidia",
			RuntimeType: "docker",
			ImageName:   backend + ":test",
			ModelMount:  ModelMountInfo{ContainerPath: "/models", Readonly: true},
		},
		Artifact:   &ArtifactInfo{Name: "artifact", Path: "/data/models/llama.gguf"},
		Deployment: &DeploymentInfo{ID: "dep", Name: "dep"},
		InstanceID: "inst-semantic-adapter",
		Node:       &NodeInfo{ID: "node", IP: "127.0.0.1"},
	})
}

func semanticAdapterSnapshot() semanticconfig.Snapshot {
	return semanticconfig.Snapshot{
		Items: map[string]semanticconfig.SnapshotItem{
			"deployment.host_port": {
				Key:   "deployment.host_port",
				Value: 18080,
			},
			"service.container_port": {
				Key:   "service.container_port",
				Value: 8000,
			},
			"deployment.served_model_name": {
				Key:   "deployment.served_model_name",
				Value: "llama",
			},
			"model_runtime.max_model_len": {
				Key:   "model_runtime.max_model_len",
				Value: 4096,
			},
		},
	}
}
