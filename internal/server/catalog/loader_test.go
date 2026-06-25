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

func containsString(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
