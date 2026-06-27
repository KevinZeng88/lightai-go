package catalog

import "testing"

func TestLoadRegistryAndBackendCatalog(t *testing.T) {
	registry, err := LoadRegistry("")
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if _, ok := registry.Item("backend.extra_args"); !ok {
		t.Fatalf("registry missing backend.extra_args")
	}
	catalog, err := LoadBackendCatalog("")
	if err != nil {
		t.Fatalf("LoadBackendCatalog: %v", err)
	}
	if len(catalog.Backends) < 3 {
		t.Fatalf("expected built-in backend catalog entries, got %d", len(catalog.Backends))
	}
	if len(catalog.Versions) < 3 {
		t.Fatalf("expected backend versions, got %d", len(catalog.Versions))
	}
	if len(catalog.Runtimes) < 3 {
		t.Fatalf("expected backend runtimes, got %d", len(catalog.Runtimes))
	}
}

func TestMaterializeConfigSetsPreservesRuntimeRequirements(t *testing.T) {
	registry, err := LoadRegistry("")
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	cat, err := LoadBackendCatalog("")
	if err != nil {
		t.Fatalf("LoadBackendCatalog: %v", err)
	}
	backendByID := map[string]BackendDoc{}
	for _, backend := range cat.Backends {
		backendByID[backend.ID] = backend
	}
	versionSetByID := map[string]ConfigSet{}
	for _, version := range cat.Versions {
		versionSetByID[version.ID] = MaterializeBackendVersion(registry, backendByID[version.BackendID], version)
	}
	assertRuntime := func(id string, wantDevice string) {
		t.Helper()
		for _, runtime := range cat.Runtimes {
			if runtime.ID != id {
				continue
			}
			set := MaterializeBackendRuntime(registry, versionSetByID[runtime.BackendVersionID], runtime)
			if len(set.Items) == 0 {
				t.Fatalf("%s materialized empty config set", id)
			}
			item := set.Items["launcher.docker_options"]
			if item.Code == "" || item.Value == nil {
				t.Fatalf("%s missing launcher.docker_options", id)
			}
			if wantDevice != "" && !containsString(mustJSON(item.Value), wantDevice) {
				t.Fatalf("%s docker options did not preserve %s: %s", id, wantDevice, mustJSON(item.Value))
			}
			return
		}
		t.Fatalf("runtime %s not found", id)
	}
	assertRuntime("runtime.vllm.nvidia-docker", "gpu_capabilities")
	assertRuntime("runtime.sglang.metax-docker", "/dev/mxcd")
	assertRuntime("runtime.llamacpp.nvidia-docker", "gpu_capabilities")
}

func TestMaterializeConfigSetsUseCanonicalSemanticKeys(t *testing.T) {
	registry, err := LoadRegistry("")
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	cat, err := LoadBackendCatalog("")
	if err != nil {
		t.Fatalf("LoadBackendCatalog: %v", err)
	}
	backendByID := map[string]BackendDoc{}
	for _, backend := range cat.Backends {
		backendByID[backend.ID] = backend
	}
	legacyKeys := []string{
		"backend.common.host",
		"backend.common.port",
		"launcher.listen_host",
		"launcher.container_port",
		"backend.arg.max_model_len",
		"backend.arg.gpu_memory_utilization",
		"backend.common.served_model_name",
	}
	for _, version := range cat.Versions {
		set := MaterializeBackendVersion(registry, backendByID[version.BackendID], version)
		for _, legacy := range legacyKeys {
			if _, ok := set.Items[legacy]; ok {
				t.Fatalf("version %s materialized legacy key %s", version.ID, legacy)
			}
		}
		for _, canonical := range []string{"service.listen_host", "service.container_port"} {
			if _, ok := set.Items[canonical]; !ok {
				t.Fatalf("version %s missing canonical key %s", version.ID, canonical)
			}
		}
	}
}

func TestMaterializeBackendVersionDoesNotAutoEnableOptionalDefaults(t *testing.T) {
	registry := &Registry{}
	backend := BackendDoc{ID: "backend.test", Slug: "test", Name: "Test"}
	version := VersionDoc{
		ID:        "version.test",
		BackendID: "backend.test",
		DefaultArgsSchema: []map[string]any{
			{"name": "--optional-with-default", "type": "string", "default": "prefilled"},
			{"name": "--required-with-default", "type": "string", "default": "required", "required": true},
		},
	}

	set := MaterializeBackendVersion(registry, backend, version)
	optional := set.Items["model_runtime.optional_with_default"]
	if optional.Code == "" {
		t.Fatalf("optional arg was not materialized: %#v", set.Items)
	}
	if optional.Enabled {
		t.Fatalf("optional default arg must not be enabled by default: %#v", optional)
	}
	if optional.Value != "prefilled" || optional.DefaultValue != "prefilled" {
		t.Fatalf("optional default value should still prefill value/default: %#v", optional)
	}

	required := set.Items["model_runtime.required_with_default"]
	if !required.Required || !required.Enabled {
		t.Fatalf("required arg should remain required and enabled: %#v", required)
	}
}

func containsString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
