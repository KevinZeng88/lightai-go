package runplan

import (
	"strings"
	"testing"
)

func TestRunPlanSourceVisibilityMatrixVLLMSGLangLlamaCpp(t *testing.T) {
	tests := []struct {
		name         string
		backend      string
		image        string
		entrypoint   []string
		args         []string
		container    int
		healthPath   string
		modelName    string
		relativePath string
	}{
		{
			name: "vllm", backend: "vllm", image: "vllm/vllm-openai:latest",
			entrypoint: []string{}, args: []string{"--model", "{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}"},
			container: 8000, healthPath: "/v1/models", modelName: "Qwen3-0.6B-Instruct-2512", relativePath: "Qwen3-0.6B-Instruct-2512",
		},
		{
			name: "sglang", backend: "sglang", image: "lmsysorg/sglang:latest",
			entrypoint: []string{}, args: []string{"python3", "-m", "sglang.launch_server", "--model-path", "{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}"},
			container: 30000, healthPath: "/health", modelName: "Qwen3-0.6B-Instruct-2512", relativePath: "Qwen3-0.6B-Instruct-2512",
		},
		{
			name: "llamacpp", backend: "llamacpp", image: "ghcr.io/ggml-org/llama.cpp:server-cuda13",
			entrypoint: []string{"llama-server"}, args: []string{"-m", "{{model_container_file}}", "--host", "0.0.0.0", "--port", "{{container_port}}"},
			container: 8080, healthPath: "/health", modelName: "qwen.gguf", relativePath: "Qwen3/qwen.gguf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := ensureNbrSnapshot(ResolveInput{
				Backend: &BackendInfo{ID: "backend." + tt.backend, Name: tt.backend, DefaultEnv: map[string]string{}},
				BackendVersion: &VersionInfo{
					ID: "version." + tt.backend, Version: "test", DefaultEntrypoint: tt.entrypoint,
					DefaultArgs: tt.args,
					ParameterDefs: []ParameterDef{
						{Name: "--host", CliName: "--host", Default: "0.0.0.0"},
						{Name: "--port", CliName: "--port", Default: tt.container},
					},
					HealthCheck:          HealthCheckInput{Path: tt.healthPath},
					DefaultContainerPort: tt.container,
					DefaultImages:        map[string]string{"nvidia": tt.image},
				},
				BackendRuntime: &RuntimeInfo{
					ID: "runtime." + tt.backend + ".nvidia-docker", Vendor: "nvidia", RuntimeType: "docker", ImageName: tt.image,
					Docker:     DockerSpecInfo{IPCMode: "host", ShmSize: "8gb"},
					ModelMount: ModelMountInfo{ContainerPath: "/models", Readonly: true},
				},
				Artifact: &ArtifactInfo{
					ID: "artifact-" + tt.backend, Name: tt.modelName,
					Path: "/home/kzeng/models/" + tt.relativePath, ModelRoot: "/home/kzeng/models", RelativePath: tt.relativePath,
				},
				Deployment: &DeploymentInfo{
					ID: "deployment-" + tt.backend, Name: "deployment-" + tt.backend,
					Service:   ServiceInfo{HostPort: tt.container, ContainerPort: tt.container, AppPort: tt.container, ListenHost: "0.0.0.0"},
					Placement: PlacementInfo{NodeID: "node-1", AcceleratorSelectionMode: "auto", AllowAutoSelect: true},
				},
				InstanceID: "instance-" + tt.backend,
				Node:       &NodeInfo{ID: "node-1", IP: "127.0.0.1"},
				AssignedGPUs: []GPUInfo{
					{Index: 0, Vendor: "nvidia"},
				},
			})
			plan, errs, _ := ResolveWithSourceMap(in)
			if len(errs) > 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			preview := EquivalentCommandPreview(plan)
			for _, want := range []string{tt.image, "--ipc host", "--shm-size 8gb", `--gpus "device=0"`, "CUDA_VISIBLE_DEVICES=0", "-p"} {
				if !strings.Contains(preview, want) {
					t.Fatalf("preview missing %q:\n%s", want, preview)
				}
			}
			if plan.ParameterSourceMap == nil {
				t.Fatal("source map missing")
			}
			assertSelfContainedEntries(t, "image", plan.ParameterSourceMap.Image)
			assertSelfContainedEntries(t, "args", plan.ParameterSourceMap.Args)
			assertSelfContainedEntries(t, "env", plan.ParameterSourceMap.Env)
			assertSelfContainedEntries(t, "mounts", plan.ParameterSourceMap.Mounts)
			assertSelfContainedEntries(t, "ports", plan.ParameterSourceMap.Ports)
			assertSelfContainedEntries(t, "docker_options", plan.ParameterSourceMap.DockerOptions)
			assertSelfContainedEntries(t, "health_check", plan.ParameterSourceMap.HealthCheck)
			assertSelfContainedEntries(t, "system_generated", plan.ParameterSourceMap.SystemGenerated)
		})
	}
}
