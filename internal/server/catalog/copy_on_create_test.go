package catalog

import (
	"encoding/json"
	"testing"
)

func testBackendVersionBundle() ConfigSetBundle {
	return ConfigSetBundle{
		OwnSets: []ConfigSet{
			{
				ConfigSetKey: "BackendVersionConfigSet",
				Items: map[string]ConfigItem{
					"launcher.image": {
						Schema: ConfigItemSchema{
							Key: "launcher.image", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
							ConfigSetKey: "BackendVersionConfigSet", Category: "launcher", Kind: "launcher_option",
							Type: "string", Required: true, SupportLevel: "verified",
						},
						Value_: ConfigItemValue{DefaultValue: "vllm/vllm-openai:v0.6.0", EffectiveValue: "vllm/vllm-openai:v0.6.0"},
						State_: ConfigItemState{Enabled: true, Checked: false, Editable: true, Visible: true, Valid: true},
						Provenance_: ConfigItemProvenance{
							ValueSource: "BackendVersion", LastValueLayer: "BackendVersionConfigBundle",
						},
					},
					"model_runtime.gpu_memory_utilization": {
						Schema: ConfigItemSchema{
							Key: "model_runtime.gpu_memory_utilization", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
							ConfigSetKey: "BackendParameterConfigSet", Category: "model_runtime", Kind: "cli_arg",
							Type: "number", Required: false, SupportLevel: "documented",
						},
						Value_: ConfigItemValue{DefaultValue: 0.9, EffectiveValue: 0.9},
						State_: ConfigItemState{Enabled: false, Checked: false, Editable: true, Visible: true, Valid: true},
						Provenance_: ConfigItemProvenance{
							ValueSource: "BackendVersion", LastValueLayer: "BackendVersionConfigBundle",
						},
					},
					"service.container_port": {
						Schema: ConfigItemSchema{
							Key: "service.container_port", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
							ConfigSetKey: "BackendVersionConfigSet", Category: "launcher", Kind: "port",
							Type: "integer", Required: true, SupportLevel: "verified",
						},
						Value_: ConfigItemValue{DefaultValue: int(8000), EffectiveValue: int(8000)},
						State_: ConfigItemState{Enabled: true, Checked: false, Editable: true, Visible: true, Valid: true},
						Provenance_: ConfigItemProvenance{
							ValueSource: "BackendVersion", LastValueLayer: "BackendVersionConfigBundle",
						},
					},
				},
			},
		},
	}
}

func TestCopyOnCreateBackendVersionToBackendRuntime(t *testing.T) {
	parent := testBackendVersionBundle()
	ownSets := []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"launcher.image": {
					Schema: ConfigItemSchema{Key: "launcher.image"},
					Value_: ConfigItemValue{EffectiveValue: "vllm/vllm-openai:latest"},
					State_: ConfigItemState{Enabled: true, Checked: true, Editable: true, Visible: true, Valid: true},
				},
			},
		},
	}

	child := CreateNextLayerBundle(parent, ownSets, "BackendRuntimeConfigBundle", "rt-123")

	if len(child.InheritedBundleSnapshots) != 1 {
		t.Fatalf("expected 1 inherited snapshot, got %d", len(child.InheritedBundleSnapshots))
	}
	inherited := child.InheritedBundleSnapshots[0]

	img := inherited.Items["launcher.image"]
	if img.Snapshot_.FromLayer != "BackendRuntimeConfigBundle" {
		t.Errorf("snapshot from layer = %q, want BackendRuntimeConfigBundle", img.Snapshot_.FromLayer)
	}
	if img.Schema.Owner != "BackendVersion" {
		t.Errorf("inherited item schema.owner = %q, want BackendVersion (unchanged)", img.Schema.Owner)
	}

	snap := child.EffectiveSnapshot()
	if snap.Items["launcher.image"].Value_.EffectiveValue != "vllm/vllm-openai:latest" {
		t.Errorf("effective image = %v, want vllm/vllm-openai:latest", snap.Items["launcher.image"].Value_.EffectiveValue)
	}
	gmu := snap.Items["model_runtime.gpu_memory_utilization"]
	if gmu.Value_.EffectiveValue != 0.9 {
		t.Errorf("effective gpu_memory = %v, want 0.9 (inherited)", gmu.Value_.EffectiveValue)
	}
}

func TestCopyOnCreateBackendRuntimeToNodeBackendRuntime(t *testing.T) {
	bv := testBackendVersionBundle()
	brOwn := []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"launcher.image": {
					Schema: ConfigItemSchema{Key: "launcher.image"},
					Value_: ConfigItemValue{EffectiveValue: "vllm/vllm-openai:latest"},
					State_: ConfigItemState{Enabled: true, Editable: true, Visible: true, Valid: true},
				},
				"launcher.docker_options": {
					Schema: ConfigItemSchema{Key: "launcher.docker_options", Owner: "BackendRuntime", OwnerLayer: "BackendRuntimeConfigBundle", ConfigSetKey: "BackendRuntimeConfigSet"},
					Value_: ConfigItemValue{EffectiveValue: map[string]any{"shm_size": "1gb"}},
					State_: ConfigItemState{Enabled: true, Editable: true, Visible: true, Valid: true},
				},
			},
		},
	}
	br := CreateNextLayerBundle(bv, brOwn, "BackendRuntimeConfigBundle", "rt-123")

	nbrOwn := []ConfigSet{
		{
			ConfigSetKey: "NodeRuntimeEnvironmentConfigSet",
			Items: map[string]ConfigItem{
				"model_runtime.gpu_memory_utilization": {
					Schema: ConfigItemSchema{Key: "model_runtime.gpu_memory_utilization", Owner: "NodeBackendRuntime", OwnerLayer: "NodeBackendRuntimeConfigBundle"},
					Value_: ConfigItemValue{DefaultValue: 0.9, EffectiveValue: 0.85},
					State_: ConfigItemState{Enabled: true, Editable: true, Visible: true, Valid: true},
				},
			},
		},
	}

	nbr := CreateNextLayerBundle(br, nbrOwn, "NodeBackendRuntimeConfigBundle", "nbr-456")
	snap := nbr.EffectiveSnapshot()

	img := snap.Items["launcher.image"]
	if img.Value_.EffectiveValue != "vllm/vllm-openai:latest" {
		t.Errorf("NBR effective image = %v, want vllm/vllm-openai:latest", img.Value_.EffectiveValue)
	}
	if img.Schema.Owner != "BackendVersion" {
		t.Errorf("image owner after 2 copies = %q, want BackendVersion (unchanged)", img.Schema.Owner)
	}

	gmu := snap.Items["model_runtime.gpu_memory_utilization"]
	if gmu.Value_.EffectiveValue != 0.85 {
		t.Errorf("NBR effective gpu_memory = %v, want 0.85", gmu.Value_.EffectiveValue)
	}

	docker := snap.Items["launcher.docker_options"]
	if docker.Value_.EffectiveValue == nil {
		t.Error("docker_options should be inherited from BR")
	}
}

func TestParentMutationDoesNotPolluteChild(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	parent.OwnSets[0].Items["launcher.image"] = ConfigItem{
		Schema: ConfigItemSchema{Key: "launcher.image", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle"},
		Value_: ConfigItemValue{EffectiveValue: "evil-image:v1"},
	}

	snap := child.EffectiveSnapshot()
	if snap.Items["launcher.image"].Value_.EffectiveValue == "evil-image:v1" {
		t.Fatalf("child picked up parent mutation! image = %v", snap.Items["launcher.image"].Value_.EffectiveValue)
	}
}

func TestChildMutationDoesNotPolluteParent(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	err := ApplyLocalEdit(&child, "BackendVersionConfigSet", "launcher.image", "new-image:v2", nil, nil, "child edit test", "user-1")
	if err != nil {
		t.Fatalf("ApplyLocalEdit: %v", err)
	}

	parentSnap := parent.EffectiveSnapshot()
	if parentSnap.Items["launcher.image"].Value_.EffectiveValue == "new-image:v2" {
		t.Fatal("parent picked up child local edit!")
	}

	childSnap := child.EffectiveSnapshot()
	if childSnap.Items["launcher.image"].Value_.EffectiveValue != "new-image:v2" {
		t.Errorf("child effective image = %v, want new-image:v2", childSnap.Items["launcher.image"].Value_.EffectiveValue)
	}
}

func TestLocalEditRecordsProvenance(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	err := ApplyLocalEdit(&child, "BackendParameterConfigSet", "model_runtime.gpu_memory_utilization",
		0.82, boolPtr(true), boolPtr(true), "node tuned for 24GB GPU", "admin")
	if err != nil {
		t.Fatalf("ApplyLocalEdit: %v", err)
	}

	edits := child.LocalEdits["BackendParameterConfigSet"]
	if edits == nil {
		t.Fatal("local edits not created")
	}
	edit := edits["model_runtime.gpu_memory_utilization"]
	if edit.LocalValue != 0.82 {
		t.Errorf("local value = %v, want 0.82", edit.LocalValue)
	}

	snap := child.EffectiveSnapshot()
	if snap.Items["model_runtime.gpu_memory_utilization"].Value_.EffectiveValue != 0.82 {
		t.Errorf("effective after edit = %v, want 0.82", snap.Items["model_runtime.gpu_memory_utilization"].Value_.EffectiveValue)
	}
}

func TestLocalEditOnReadOnlyItemIsRejected(t *testing.T) {
	parent := testBackendVersionBundle()
	for k, item := range parent.OwnSets[0].Items {
		item.Schema.ReadOnly = true
		item.State_.Editable = false
		parent.OwnSets[0].Items[k] = item
	}
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	err := ApplyLocalEdit(&child, "", "launcher.image", "new:v2", nil, nil, "try edit readonly", "u1")
	if err == nil {
		t.Error("expected error editing readonly item, got nil")
	}
}

func TestCloneDoesNotExpandCheckedEnabled(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	err := ApplyLocalEdit(&child, "BackendParameterConfigSet", "model_runtime.gpu_memory_utilization",
		0.85, boolPtr(true), boolPtr(true), "enable for node", "admin")
	if err != nil {
		t.Fatalf("ApplyLocalEdit: %v", err)
	}

	clone := CreateNextLayerBundle(child, nil, "NodeBackendRuntimeConfigBundle", "nbr-clone")
	cloneSnap := clone.EffectiveSnapshot()
	gmu := cloneSnap.Items["model_runtime.gpu_memory_utilization"]

	if gmu.Value_.EffectiveValue != 0.85 {
		t.Errorf("clone effective gpu_memory = %v, want 0.85", gmu.Value_.EffectiveValue)
	}
}

func TestOwnerUnchangedThroughCopyChain(t *testing.T) {
	bv := testBackendVersionBundle()
	br := CreateNextLayerBundle(bv, nil, "BackendRuntimeConfigBundle", "rt-123")
	nbr := CreateNextLayerBundle(br, nil, "NodeBackendRuntimeConfigBundle", "nbr-456")
	dep := CreateNextLayerBundle(nbr, nil, "DeploymentConfigBundle", "dep-789")

	for _, bundle := range []ConfigSetBundle{bv, br, nbr, dep} {
		snap := bundle.EffectiveSnapshot()
		img := snap.Items["launcher.image"]
		if img.Schema.Owner != "BackendVersion" {
			t.Errorf("image owner = %q, want BackendVersion (unchanged)", img.Schema.Owner)
		}
	}
}

func TestValidateSchemaImmutabilityDetectsOwnerChange(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	child.OwnSets = []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"launcher.image": {
					Schema: ConfigItemSchema{Key: "launcher.image", Owner: "BackendRuntime", OwnerLayer: "BackendRuntimeConfigBundle"},
					Value_: ConfigItemValue{EffectiveValue: "my-image:latest"},
				},
			},
		},
	}

	violations := ValidateSchemaImmutability(child)
	if len(violations) == 0 {
		t.Error("expected violations for owner change, got none")
	}
}

func TestValidateSchemaImmutabilityAllowsNewOwnItems(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"runtime.health": {
					Schema: ConfigItemSchema{Key: "runtime.health", Owner: "BackendRuntime", OwnerLayer: "BackendRuntimeConfigBundle"},
					Value_: ConfigItemValue{EffectiveValue: map[string]any{"type": "http_get", "path": "/health"}},
				},
			},
		},
	}, "BackendRuntimeConfigBundle", "rt-123")

	violations := ValidateSchemaImmutability(child)
	if len(violations) != 0 {
		t.Errorf("expected no violations for new own item, got: %v", violations)
	}
}

func TestNBRIsolationAfterBVMutation(t *testing.T) {
	bv := testBackendVersionBundle()
	br := CreateNextLayerBundle(bv, nil, "BackendRuntimeConfigBundle", "rt-123")
	nbr := CreateNextLayerBundle(br, nil, "NodeBackendRuntimeConfigBundle", "nbr-456")

	nbrJSON, err := json.Marshal(nbr)
	if err != nil {
		t.Fatalf("marshal nbr: %v", err)
	}

	bv.OwnSets[0].Items["launcher.image"] = ConfigItem{
		Schema: ConfigItemSchema{Key: "launcher.image", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle"},
		Value_: ConfigItemValue{EffectiveValue: "changed-after-nbr:v9"},
	}

	var nbrRestored ConfigSetBundle
	if err := json.Unmarshal(nbrJSON, &nbrRestored); err != nil {
		t.Fatalf("unmarshal nbr: %v", err)
	}

	nbrSnap := nbrRestored.EffectiveSnapshot()
	if nbrSnap.Items["launcher.image"].Value_.EffectiveValue == "changed-after-nbr:v9" {
		t.Fatal("NBR picked up BV mutation after copy-on-create!")
	}
}

func boolPtr(b bool) *bool { return &b }
