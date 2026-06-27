package catalog

import (
	"encoding/json"
	"testing"
)

// ============================================================================
// Helper to build a minimal BackendVersion bundle for testing
// ============================================================================

func testBackendVersionBundle() ConfigSetBundle {
	return ConfigSetBundle{
		OwnSets: []ConfigSet{
			{
				ConfigSetKey: "BackendVersionConfigSet",
				Items: map[string]ConfigItem{
					"launcher.image": {
						Code:     "launcher.image",
						Category: "launcher",
						Kind:     "launcher_option",
						Type:     "string",
						Value:    "vllm/vllm-openai:v0.6.0",
						Enabled:  true,
						Source:   map[string]string{"layer": "BackendVersion", "ref": "bv-vllm", "reason": "catalog"},
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
						Code:         "model_runtime.gpu_memory_utilization",
						Category:     "model_runtime",
						Kind:         "cli_arg",
						Type:         "number",
						Value:        0.9,
						DefaultValue: 0.9,
						Enabled:      false,
						Source:       map[string]string{"layer": "BackendVersion", "ref": "bv-vllm", "reason": "registry_default"},
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
						Code:         "service.container_port",
						Category:     "launcher",
						Kind:         "port",
						Type:         "integer",
						Value:        8000,
						DefaultValue: 8000,
						Enabled:      true,
						Required:     true,
						Source:       map[string]string{"layer": "BackendVersion", "ref": "bv-vllm", "reason": "catalog"},
						Schema: ConfigItemSchema{
							Key: "service.container_port", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
							ConfigSetKey: "BackendVersionConfigSet", Category: "launcher", Kind: "port",
							Type: "integer", Required: true, SupportLevel: "verified",
						},
						Value_: ConfigItemValue{DefaultValue: 8000, EffectiveValue: 8000},
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

// ============================================================================
// Copy-on-create: BackendVersion → BackendRuntime
// ============================================================================

func TestCopyOnCreateBackendVersionToBackendRuntime(t *testing.T) {
	parent := testBackendVersionBundle()

	ownSets := []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"launcher.image": {
					Code:    "launcher.image",
					Value:   "vllm/vllm-openai:latest",
					Enabled: true,
				},
			},
		},
	}

	child := CreateNextLayerBundle(parent, ownSets, "BackendRuntimeConfigBundle", "rt-123")

	// 1. Child must have inherited snapshot
	if len(child.InheritedBundleSnapshots) != 1 {
		t.Fatalf("expected 1 inherited snapshot, got %d", len(child.InheritedBundleSnapshots))
	}
	inherited := child.InheritedBundleSnapshots[0]

	// 2. Inherited items must have snapshot stamps
	img := inherited.Items["launcher.image"]
	if img.Snapshot_.FromLayer != "BackendRuntimeConfigBundle" {
		t.Errorf("snapshot from layer = %q, want BackendRuntimeConfigBundle", img.Snapshot_.FromLayer)
	}
	if img.Snapshot_.FromID != "rt-123" {
		t.Errorf("snapshot from id = %q, want rt-123", img.Snapshot_.FromID)
	}

	// 3. Schema owner must remain BackendVersion on inherited items
	if img.Schema.Owner != "BackendVersion" {
		t.Errorf("inherited item schema.owner = %q, want BackendVersion (unchanged)", img.Schema.Owner)
	}

	// 4. Child has own sets
	if len(child.OwnSets) != 1 {
		t.Fatalf("expected 1 own set, got %d", len(child.OwnSets))
	}

	// 5. Effective snapshot: own set overwrites inherited
	snap := child.EffectiveSnapshot()
	if snap.Items["launcher.image"].Value_.EffectiveValue != "vllm/vllm-openai:latest" {
		t.Errorf("effective image = %v, want vllm/vllm-openai:latest",
			snap.Items["launcher.image"].Value_.EffectiveValue)
	}
	// gpu_memory_utilization should still be inherited since not in own set
	gmu := snap.Items["model_runtime.gpu_memory_utilization"]
	if gmu.Value_.EffectiveValue != 0.9 {
		t.Errorf("effective gpu_memory = %v, want 0.9 (inherited)", gmu.Value_.EffectiveValue)
	}
}

// ============================================================================
// Copy-on-create: BackendRuntime → NodeBackendRuntime
// ============================================================================

func TestCopyOnCreateBackendRuntimeToNodeBackendRuntime(t *testing.T) {
	bv := testBackendVersionBundle()
	brOwn := []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"launcher.image": {
					Code:    "launcher.image",
					Value:   "vllm/vllm-openai:latest",
					Enabled: true,
					// Owner NOT set — inherited items preserve parent owner
				},
				"launcher.docker_options": {
					Code:    "launcher.docker_options",
					Type:    "object",
					Enabled: true,
					Value:   map[string]any{"shm_size": "1gb"},
					Schema: ConfigItemSchema{
						Key: "launcher.docker_options", Owner: "BackendRuntime", OwnerLayer: "BackendRuntimeConfigBundle",
					},
				},
			},
		},
	}
	br := CreateNextLayerBundle(bv, brOwn, "BackendRuntimeConfigBundle", "rt-123")

	// Now create NBR from BR
	nbrOwn := []ConfigSet{
		{
			ConfigSetKey: "NodeRuntimeEnvironmentConfigSet",
			Items: map[string]ConfigItem{
				"model_runtime.gpu_memory_utilization": {
					Code:         "model_runtime.gpu_memory_utilization",
					Value:        0.85,
					Enabled:      true,
					DefaultValue: 0.9,
					Schema: ConfigItemSchema{
						Key: "model_runtime.gpu_memory_utilization", Owner: "NodeBackendRuntime",
						OwnerLayer: "NodeBackendRuntimeConfigBundle",
					},
				},
			},
		},
	}

	nbr := CreateNextLayerBundle(br, nbrOwn, "NodeBackendRuntimeConfigBundle", "nbr-456")

	snap := nbr.EffectiveSnapshot()

	// launcher.image: value from BR, owner preserved from BV
	img := snap.Items["launcher.image"]
	if img.Value_.EffectiveValue != "vllm/vllm-openai:latest" {
		t.Errorf("NBR effective image = %v, want vllm/vllm-openai:latest", img.Value_.EffectiveValue)
	}
	if img.Schema.Owner != "BackendVersion" {
		t.Errorf("image owner after 2 copies = %q, want BackendVersion (unchanged)", img.Schema.Owner)
	}

	// gpu_memory_utilization: NBR own set overrides
	gmu := snap.Items["model_runtime.gpu_memory_utilization"]
	if gmu.Value_.EffectiveValue != 0.85 {
		t.Errorf("NBR effective gpu_memory = %v, want 0.85", gmu.Value_.EffectiveValue)
	}

	// docker_options: inherited from BR
	docker := snap.Items["launcher.docker_options"]
	if docker.Value_.EffectiveValue == nil {
		t.Error("docker_options should be inherited from BR")
	}
}

// ============================================================================
// Parent mutation does NOT pollute child
// ============================================================================

func TestParentMutationDoesNotPolluteChild(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	// Mutate parent after child was created
	parent.OwnSets[0].Items["launcher.image"] = ConfigItem{
		Code:  "launcher.image",
		Value: "evil-image:v1",
		Schema: ConfigItemSchema{
			Key: "launcher.image", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
		},
		Value_: ConfigItemValue{EffectiveValue: "evil-image:v1"},
	}

	// Child's effective snapshot must NOT see the mutation
	snap := child.EffectiveSnapshot()
	if snap.Items["launcher.image"].Value_.EffectiveValue == "evil-image:v1" {
		t.Fatalf("child picked up parent mutation! image = %v", snap.Items["launcher.image"].Value_.EffectiveValue)
	}
	if snap.Items["launcher.image"].Value_.EffectiveValue != "vllm/vllm-openai:v0.6.0" {
		t.Logf("child image = %v (expected original)", snap.Items["launcher.image"].Value_.EffectiveValue)
	}
}

// ============================================================================
// Child mutation does NOT pollute parent
// ============================================================================

func TestChildMutationDoesNotPolluteParent(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	// Apply local edit on child
	err := ApplyLocalEdit(&child, "BackendVersionConfigSet", "launcher.image", "new-image:v2", nil, nil, "child edit test", "user-1")
	if err != nil {
		t.Fatalf("ApplyLocalEdit: %v", err)
	}

	// Parent must NOT see the edit
	parentSnap := parent.EffectiveSnapshot()
	if parentSnap.Items["launcher.image"].Value_.EffectiveValue == "new-image:v2" {
		t.Fatal("parent picked up child local edit!")
	}
	if parentSnap.Items["launcher.image"].Value_.EffectiveValue != "vllm/vllm-openai:v0.6.0" {
		t.Errorf("parent image unexpectedly changed to %v", parentSnap.Items["launcher.image"].Value_.EffectiveValue)
	}

	// Child's effective snapshot should reflect the local edit
	childSnap := child.EffectiveSnapshot()
	if childSnap.Items["launcher.image"].Value_.EffectiveValue != "new-image:v2" {
		t.Errorf("child effective image = %v, want new-image:v2", childSnap.Items["launcher.image"].Value_.EffectiveValue)
	}
}

// ============================================================================
// Local edits with provenance tracking
// ============================================================================

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
		t.Fatal("local edits not created for config set key")
	}
	edit := edits["model_runtime.gpu_memory_utilization"]
	if edit.LocalValue != 0.82 {
		t.Errorf("local value = %v, want 0.82", edit.LocalValue)
	}
	if edit.Reason != "node tuned for 24GB GPU" {
		t.Errorf("reason = %q", edit.Reason)
	}
	if edit.EditedBy != "admin" {
		t.Errorf("edited_by = %q, want admin", edit.EditedBy)
	}
	if edit.EditedAt == "" {
		t.Error("edited_at should be set")
	}

	snap := child.EffectiveSnapshot()
	if snap.Items["model_runtime.gpu_memory_utilization"].Value_.EffectiveValue != 0.82 {
		t.Errorf("effective after edit = %v, want 0.82",
			snap.Items["model_runtime.gpu_memory_utilization"].Value_.EffectiveValue)
	}
}

func TestLocalEditOnReadOnlyItemIsRejected(t *testing.T) {
	parent := testBackendVersionBundle()
	// Mark an item as schema.read_only
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

// ============================================================================
// Clone does NOT expand checked/enabled scope
// ============================================================================

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
		t.Errorf("clone effective gpu_memory = %v, want 0.85 (inherited from parent)", gmu.Value_.EffectiveValue)
	}
}

// ============================================================================
// Owner unchanged through copy chain
// ============================================================================

func TestOwnerUnchangedThroughCopyChain(t *testing.T) {
	bv := testBackendVersionBundle()
	br := CreateNextLayerBundle(bv, nil, "BackendRuntimeConfigBundle", "rt-123")
	nbr := CreateNextLayerBundle(br, nil, "NodeBackendRuntimeConfigBundle", "nbr-456")
	dep := CreateNextLayerBundle(nbr, nil, "DeploymentConfigBundle", "dep-789")

	for _, bundle := range []ConfigSetBundle{bv, br, nbr, dep} {
		snap := bundle.EffectiveSnapshot()
		img := snap.Items["launcher.image"]
		if img.Schema.Owner != "BackendVersion" {
			t.Errorf("image owner = %q, want BackendVersion (unchanged through copy chain)", img.Schema.Owner)
		}
	}

	depSnap := dep.EffectiveSnapshot()
	if depSnap.Items["launcher.image"].Value_.EffectiveValue != "vllm/vllm-openai:v0.6.0" {
		t.Errorf("deployment image = %v, want original vllm/vllm-openai:v0.6.0",
			depSnap.Items["launcher.image"].Value_.EffectiveValue)
	}
}

// ============================================================================
// Schema immutability validation
// ============================================================================

func TestValidateSchemaImmutabilityDetectsOwnerChange(t *testing.T) {
	parent := testBackendVersionBundle()

	// Create child normally first to get inherited snapshot
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	// Directly set a corrupted own set that illegally changes owner on inherited item.
	// Bypass both CreateNextLayerBundle and AddOwnSet (which now prevent owner changes).
	child.OwnSets = []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"launcher.image": {
					Code:  "launcher.image",
					Value: "my-image:latest",
					Schema: ConfigItemSchema{
						Key: "launcher.image", Owner: "BackendRuntime",
						OwnerLayer: "BackendRuntimeConfigBundle",
					},
				},
			},
		},
	}

	violations := ValidateSchemaImmutability(child)
	if len(violations) == 0 {
		t.Error("expected violations for owner change, got none")
	}
	found := false
	for _, v := range violations {
		for i := 0; i <= len(v)-5; i++ {
			if v[i:i+5] == "owner" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected owner violation in: %v", violations)
	}
}

func TestValidateSchemaImmutabilityAllowsNewOwnItems(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, []ConfigSet{
		{
			ConfigSetKey: "BackendRuntimeConfigSet",
			Items: map[string]ConfigItem{
				"runtime.health": {
					Code:  "runtime.health",
					Value: map[string]any{"type": "http_get", "path": "/health"},
					Schema: ConfigItemSchema{
						Key: "runtime.health", Owner: "BackendRuntime", OwnerLayer: "BackendRuntimeConfigBundle",
					},
				},
			},
		},
	}, "BackendRuntimeConfigBundle", "rt-123")

	violations := ValidateSchemaImmutability(child)
	if len(violations) != 0 {
		t.Errorf("expected no violations for new own item, got: %v", violations)
	}
}

// ============================================================================
// Copy-on-create preserves NBR snapshot isolation (regression from codex review)
// ============================================================================

func TestNBRIsolationAfterBVMutation(t *testing.T) {
	bv := testBackendVersionBundle()
	br := CreateNextLayerBundle(bv, nil, "BackendRuntimeConfigBundle", "rt-123")
	nbr := CreateNextLayerBundle(br, nil, "NodeBackendRuntimeConfigBundle", "nbr-456")

	nbrJSON, err := json.Marshal(nbr)
	if err != nil {
		t.Fatalf("marshal nbr: %v", err)
	}

	// Mutate BV after NBR was created
	bv.OwnSets[0].Items["launcher.image"] = ConfigItem{
		Code:  "launcher.image",
		Value: "changed-after-nbr:v9",
		Schema: ConfigItemSchema{
			Key: "launcher.image", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
		},
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
	if nbrSnap.Items["launcher.image"].Value_.EffectiveValue != "vllm/vllm-openai:v0.6.0" {
		t.Errorf("NBR image = %v, want original", nbrSnap.Items["launcher.image"].Value_.EffectiveValue)
	}
}

// ============================================================================
// Helpers
// ============================================================================

func boolPtr(b bool) *bool {
	return &b
}
