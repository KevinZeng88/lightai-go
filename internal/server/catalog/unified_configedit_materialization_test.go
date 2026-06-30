package catalog_test

import (
	"encoding/json"
	"testing"

	"lightai-go/internal/server/catalog"
	"lightai-go/internal/server/configedit"
)

func TestBuiltinRuntimeConfigEditMaterializesRuntimeParameters(t *testing.T) {
	registry, err := catalog.LoadRegistry("")
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	cat, err := catalog.LoadBackendCatalog("")
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	backends := map[string]catalog.BackendDoc{}
	for _, backend := range cat.Backends {
		backends[backend.ID] = backend
	}
	versionSets := map[string]catalog.ConfigSet{}
	for _, version := range cat.Versions {
		versionSets[version.ID] = catalog.MaterializeBackendVersion(registry, backends[version.BackendID], version)
	}
	runtimes := map[string]catalog.RuntimeDoc{}
	for _, runtime := range cat.Runtimes {
		runtimes[runtime.ID] = runtime
	}

	cases := []struct {
		name         string
		runtimeID    string
		wantFields   []string
		wantSections map[string]string
		wantEnvKeys  []string
		wantDocker   []string
	}{
		{
			name:      "vLLM MetaX",
			runtimeID: "runtime.vllm.metax-docker",
			wantFields: []string{
				"model_runtime.tensor_parallel_size",
				"model_runtime.pipeline_parallel_size",
				"model_runtime.max_model_len",
				"model_runtime.gpu_memory_utilization",
				"model_runtime.max_num_seqs",
				"model_runtime.max_num_batched_tokens",
				"model_runtime.kv_cache_dtype",
				"model_runtime.dtype",
				"model_runtime.swap_space",
				"model_runtime.cpu_offload_gb",
				"model_runtime.enforce_eager",
				"model_runtime.safetensors_load_strategy",
				"model_runtime.trust_remote_code",
				"model_runtime.download_dir",
				"backend.extra_args",
			},
			wantSections: map[string]string{
				"docker.shm_size":                          "container_resources",
				"docker.ipc_mode":                          "container_resources",
				"docker.network_mode":                      "container_resources",
				"docker.group_add":                         "devices_mounts",
				"docker.devices":                           "devices_mounts",
				"docker.privileged":                        "security_high_risk",
				"launcher.docker_options.security_options": "security_high_risk",
				"launcher.docker_options.cap_add":          "security_high_risk",
			},
			wantEnvKeys: []string{"MACA_SMALL_PAGESIZE_ENABLE", "PYTORCH_ENABLE_PG_HIGH_PRIORITY_STREAM"},
			wantDocker:  []string{"shm_size", "ipc_mode", "network_mode", "group_add", "devices", "privileged", "security_options", "cap_add", "ulimits"},
		},
		{
			name:      "vLLM NVIDIA",
			runtimeID: "runtime.vllm.nvidia-docker",
			wantFields: []string{
				"model_runtime.tensor_parallel_size",
				"model_runtime.max_model_len",
				"model_runtime.gpu_memory_utilization",
				"model_runtime.dtype",
				"backend.extra_args",
			},
			wantSections: map[string]string{
				"docker.shm_size": "container_resources",
				"docker.ipc_mode": "container_resources",
			},
			wantDocker: []string{"shm_size", "ipc_mode", "gpu_driver", "gpu_capabilities"},
		},
		{
			name:      "SGLang NVIDIA",
			runtimeID: "runtime.sglang.nvidia-docker",
			wantFields: []string{
				"model_runtime.gpu_memory_utilization",
				"model_runtime.max_model_len",
				"model_runtime.tp",
				"model_runtime.dp",
				"model_runtime.max_running_requests",
				"model_runtime.disable_cuda_graph",
				"backend.extra_args",
			},
			wantSections: map[string]string{
				"docker.shm_size": "container_resources",
				"docker.ipc_mode": "container_resources",
			},
			wantDocker: []string{"shm_size", "ipc_mode", "gpu_driver", "gpu_capabilities"},
		},
		{
			name:      "llama.cpp NVIDIA",
			runtimeID: "runtime.llamacpp.nvidia-docker",
			wantFields: []string{
				"model_runtime.max_model_len",
				"model_runtime.n_gpu_layers",
				"model_runtime.threads",
				"model_runtime.batch_size",
				"model_runtime.ubatch_size",
				"model_runtime.cache_type_k",
				"model_runtime.cache_type_v",
				"backend.extra_args",
			},
			wantSections: map[string]string{
				"docker.shm_size": "container_resources",
				"docker.ipc_mode": "container_resources",
			},
			wantDocker: []string{"shm_size", "ipc_mode", "gpu_driver", "gpu_capabilities"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runtime := runtimes[tc.runtimeID]
			if runtime.ID == "" {
				t.Fatalf("missing runtime %s", tc.runtimeID)
			}
			set := catalog.MaterializeBackendRuntime(registry, versionSets[runtime.BackendVersionID], runtime)
			view, err := configedit.ProjectConfigSetToEditView(configedit.ProjectInput{
				ConfigSet:  configSetMap(t, set),
				Layer:      "backend_runtime",
				ObjectKind: "backend_runtime",
				ObjectID:   runtime.ID,
				ViewLevel:  "advanced",
				Readonly:   true,
			})
			if err != nil {
				t.Fatalf("project ConfigEdit: %v", err)
			}
			fields := fieldsByKey(view.Fields)
			for _, key := range tc.wantFields {
				field := fields[key]
				if field.Key == "" {
					t.Fatalf("%s missing structured field %s", tc.runtimeID, key)
				}
				if field.Widget == "raw_json" {
					t.Fatalf("%s field %s is raw JSON-only: %#v", tc.runtimeID, key, field)
				}
				if field.Label == key || field.Label == field.InternalKey {
					t.Fatalf("%s field %s has technical label only: %#v", tc.runtimeID, key, field)
				}
			}
			for key, section := range tc.wantSections {
				field := fields[key]
				if field.Key == "" {
					t.Fatalf("%s missing structured field %s", tc.runtimeID, key)
				}
				if field.Section != section {
					t.Fatalf("%s field %s section=%s want %s", tc.runtimeID, key, field.Section, section)
				}
				if field.Widget == "raw_json" {
					t.Fatalf("%s field %s is raw JSON-only: %#v", tc.runtimeID, key, field)
				}
			}
			assertRuntimeEnvKeys(t, set, tc.wantEnvKeys)
			assertDockerKeys(t, set, tc.wantDocker)
		})
	}
}

func TestPartialProjectionDoesNotDropUnmappedConfigSetItems(t *testing.T) {
	set := map[string]any{
		"schema_version": 1,
		"items": map[string]any{
			"launcher.image": map[string]any{
				"schema": map[string]any{"key": "launcher.image", "category": "launcher", "kind": "image", "type": "string", "required": true},
				"state":  map[string]any{"enabled": true, "visible": true, "editable": true},
				"value":  map[string]any{"effective_value": "example:latest"},
			},
			"model_runtime.unmapped_new_flag": map[string]any{
				"schema": map[string]any{"key": "model_runtime.unmapped_new_flag", "category": "model_runtime", "kind": "cli_arg", "type": "string"},
				"state":  map[string]any{"enabled": true, "visible": true, "editable": true},
				"value":  map[string]any{"effective_value": "kept"},
			},
			"launcher.docker_options": map[string]any{
				"schema": map[string]any{"key": "launcher.docker_options", "category": "launcher", "kind": "launcher_option", "type": "object"},
				"state":  map[string]any{"enabled": true, "visible": true, "editable": true},
				"value": map[string]any{"effective_value": map[string]any{
					"custom_runtime": "nvidia",
					"cap_add":        []any{"SYS_PTRACE"},
				}},
			},
		},
	}
	view, err := configedit.ProjectConfigSetToEditView(configedit.ProjectInput{
		ConfigSet:  set,
		Layer:      "backend_runtime",
		ObjectKind: "backend_runtime",
		ObjectID:   "rt-unmapped",
		ViewLevel:  "advanced",
	})
	if err != nil {
		t.Fatalf("project ConfigEdit: %v", err)
	}
	fields := fieldsByKey(view.Fields)
	for _, key := range []string{"model_runtime.unmapped_new_flag", "launcher.docker_options.custom_runtime", "launcher.docker_options.cap_add"} {
		field := fields[key]
		if field.Key == "" {
			t.Fatalf("unmapped item %s was dropped", key)
		}
		if field.Widget == "raw_json" {
			t.Fatalf("unmapped item %s became raw JSON-only: %#v", key, field)
		}
	}
	if fields["launcher.docker_options.cap_add"].Section != "security_high_risk" {
		t.Fatalf("cap_add section=%s want security_high_risk", fields["launcher.docker_options.cap_add"].Section)
	}
}

func configSetMap(t *testing.T, set catalog.ConfigSet) map[string]any {
	t.Helper()
	raw, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal config set: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal config set: %v", err)
	}
	return out
}

func fieldsByKey(fields []configedit.EditField) map[string]configedit.EditField {
	out := make(map[string]configedit.EditField, len(fields))
	for _, field := range fields {
		out[field.Key] = field
	}
	return out
}

func assertRuntimeEnvKeys(t *testing.T, set catalog.ConfigSet, keys []string) {
	t.Helper()
	if len(keys) == 0 {
		return
	}
	env := set.Items["runtime.env"].Value_.EffectiveValue
	envMap, _ := env.(map[string]string)
	if envMap == nil {
		t.Fatalf("runtime.env is not a string map: %#v", env)
	}
	for _, key := range keys {
		if envMap[key] == "" {
			t.Fatalf("runtime.env missing %s in %#v", key, envMap)
		}
	}
}

func assertDockerKeys(t *testing.T, set catalog.ConfigSet, keys []string) {
	t.Helper()
	docker, _ := set.Items["launcher.docker_options"].Value_.EffectiveValue.(map[string]any)
	if docker == nil {
		t.Fatalf("launcher.docker_options is not an object: %#v", set.Items["launcher.docker_options"].Value_.EffectiveValue)
	}
	for _, key := range keys {
		if _, ok := docker[key]; !ok {
			t.Fatalf("launcher.docker_options missing %s in %#v", key, docker)
		}
	}
}
