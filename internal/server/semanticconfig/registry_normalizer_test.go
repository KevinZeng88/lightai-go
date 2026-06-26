package semanticconfig

import (
	"strings"
	"testing"
)

func TestDefaultRegistryContainsCanonicalOwners(t *testing.T) {
	reg := DefaultRegistry()

	maxLen, ok := reg.Get("model_runtime.max_model_len")
	if !ok {
		t.Fatal("missing model_runtime.max_model_len")
	}
	if maxLen.Owner != OwnerModelRuntime {
		t.Fatalf("max_model_len owner = %q, want %q", maxLen.Owner, OwnerModelRuntime)
	}
	if maxLen.DisplayTier != TierDeploymentCommonAdvanced {
		t.Fatalf("max_model_len display tier = %q", maxLen.DisplayTier)
	}

	servedName, ok := reg.Get("deployment.served_model_name")
	if !ok {
		t.Fatal("missing deployment.served_model_name")
	}
	if servedName.Owner != OwnerDeploymentService {
		t.Fatalf("served_model_name owner = %q, want %q", servedName.Owner, OwnerDeploymentService)
	}

	if _, ok := reg.Get("backend.arg.max_model_len"); ok {
		t.Fatal("legacy backend.arg.max_model_len must not be registered as canonical")
	}
}

func TestNormalizeConfigSetRewritesLegacyKeysAndReportsConflicts(t *testing.T) {
	reg := DefaultRegistry()
	set := map[string]any{
		"schema_version": 1,
		"items": map[string]any{
			"backend.common.port": map[string]any{
				"code":  "backend.common.port",
				"type":  "integer",
				"value": 8000,
			},
			"service.container_port": map[string]any{
				"code":  "service.container_port",
				"type":  "integer",
				"value": 9000,
			},
			"backend.arg.max_model_len": map[string]any{
				"code":  "backend.arg.max_model_len",
				"type":  "integer",
				"value": 4096,
				"render": map[string]any{
					"flag": "--max-model-len",
				},
			},
			"backend.common.served_model_name": map[string]any{
				"code":  "backend.common.served_model_name",
				"type":  "string",
				"value": "llama",
			},
			"launcher.docker_options": map[string]any{
				"code": "launcher.docker_options",
				"type": "object",
				"value": map[string]any{
					"shm_size":  "16g",
					"group_add": []any{"video"},
				},
			},
		},
	}

	out, err := NormalizeConfigSet(reg, set)
	if err != nil {
		t.Fatalf("normalize config set: %v", err)
	}
	items := out.Items
	for _, legacy := range []string{
		"backend.common.port",
		"backend.arg.max_model_len",
		"backend.common.served_model_name",
		"launcher.docker_options",
	} {
		if _, ok := items[legacy]; ok {
			t.Fatalf("legacy key %q remained in normalized items", legacy)
		}
	}
	if got := items["service.container_port"].Value; got != 9000 {
		t.Fatalf("canonical service.container_port value = %#v, want canonical precedence 9000", got)
	}
	if got := items["model_runtime.max_model_len"].Value; got != 4096 {
		t.Fatalf("model_runtime.max_model_len value = %#v, want 4096", got)
	}
	if got := items["deployment.served_model_name"].Owner; got != OwnerDeploymentService {
		t.Fatalf("deployment.served_model_name owner = %q", got)
	}
	if got := items["docker.shm_size"].Value; got != "16g" {
		t.Fatalf("docker.shm_size value = %#v, want 16g", got)
	}
	if len(out.Warnings) == 0 {
		t.Fatal("expected conflict warning")
	}
	var foundConflict bool
	for _, warning := range out.Warnings {
		if warning.Code == WarningConflict && warning.SemanticKey == "service.container_port" {
			foundConflict = true
		}
	}
	if !foundConflict {
		t.Fatalf("missing service.container_port conflict warning: %#v", out.Warnings)
	}
}

func TestValidatePatchRejectsLegacyKeysAndUnknownCanonicalKeys(t *testing.T) {
	reg := DefaultRegistry()
	err := ValidatePatchKeys(reg, []string{"backend.arg.max_model_len"})
	if err == nil || !strings.Contains(err.Error(), "direct legacy key patch") {
		t.Fatalf("expected direct legacy key patch error, got %v", err)
	}

	err = ValidatePatchKeys(reg, []string{"model_runtime.max_model_len", "runtime.env"})
	if err != nil {
		t.Fatalf("canonical patch keys should validate: %v", err)
	}

	err = ValidatePatchKeys(reg, []string{"model_runtime.unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown canonical key") {
		t.Fatalf("expected unknown canonical key error, got %v", err)
	}
}
