package semanticconfig

import "testing"

func TestProjectSnapshotIncludesSemanticMetadataAndWarnings(t *testing.T) {
	reg := DefaultRegistry()
	snapshot := Snapshot{
		Items: map[string]SnapshotItem{
			"model_runtime.context_length": {
				Key:   "model_runtime.context_length",
				Owner: OwnerModelArtifact,
				Type:  TypeInteger,
				Value: 2048,
			},
			"model_runtime.max_model_len": {
				Key:          "model_runtime.max_model_len",
				Owner:        OwnerModelRuntime,
				Type:         TypeInteger,
				DisplayTier:  TierDeploymentCommonAdvanced,
				Value:        4096,
				DefaultValue: 2048,
				CopiedFrom:   "NodeBackendRuntime:nbr.vllm",
				Dirty:        true,
			},
		},
	}

	view := ProjectSnapshot(reg, snapshot, ProjectOptions{ObjectKind: "deployment", Layer: "deployment"})
	field := requireSemanticField(t, view, "model_runtime.max_model_len")
	if field.Owner != OwnerModelRuntime {
		t.Fatalf("owner = %q, want %q", field.Owner, OwnerModelRuntime)
	}
	if field.Tier != TierDeploymentCommonAdvanced {
		t.Fatalf("tier = %q, want %q", field.Tier, TierDeploymentCommonAdvanced)
	}
	if !field.Dirty {
		t.Fatal("dirty metadata not projected")
	}
	if field.CopiedFrom == "" {
		t.Fatal("copied_from metadata not projected")
	}
	if len(field.Warnings) == 0 {
		t.Fatal("expected max_model_len warning")
	}
}

func TestValidateSnapshotPatchHardErrorsOnly(t *testing.T) {
	reg := DefaultRegistry()
	snapshot := Snapshot{
		Items: map[string]SnapshotItem{
			"service.container_port": {
				Key:     "service.container_port",
				Owner:   OwnerRuntimeService,
				Type:    TypeInteger,
				Enabled: true,
			},
		},
	}
	if err := ValidateSnapshotPatch(reg, snapshot, []PatchField{{Key: "service.container_port", Value: "not-a-port"}}); err == nil {
		t.Fatal("expected type/port parse hard error")
	}
	if err := ValidateSnapshotPatch(reg, snapshot, []PatchField{{Key: "backend.common.port", Value: 8000}}); err == nil {
		t.Fatal("expected direct legacy patch hard error")
	}
	if err := ValidateSnapshotPatch(reg, snapshot, []PatchField{{Key: "service.container_port", Value: 8000}}); err != nil {
		t.Fatalf("valid patch should pass: %v", err)
	}
}

func requireSemanticField(t *testing.T, view ProjectedView, key string) ProjectedField {
	t.Helper()
	for _, section := range view.Sections {
		for _, field := range section.Fields {
			if field.Key == key {
				return field
			}
		}
	}
	t.Fatalf("missing field %s in %#v", key, view)
	return ProjectedField{}
}
