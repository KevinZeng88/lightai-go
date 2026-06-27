package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

// ============================================================================
// RegistryItem → tiered ConfigItem conversion tests
// ============================================================================

func TestRegistryItemToConfigItem(t *testing.T) {
	ri := RegistryItem{
		Code:         "launcher.image",
		Category:     "launcher",
		Kind:         "cli_arg",
		Type:         "string",
		Required:     true,
		Advanced:     false,
		Readonly:     false,
		Order:        10,
		SupportLevel: "verified",
		Value:        "vllm/vllm-openai:latest",
		DefaultValue: "vllm/vllm-openai:v0.6.0",
		Enabled:      true,
		Render:       map[string]any{"target": "cli", "flag": "--image"},
		Constraints:  map[string]any{"min_length": float64(1)},
		Extensions:   map[string]interface{}{"label": "Container Image", "group": "launcher"},
		Source:       map[string]string{"layer": "BackendRuntime", "ref": "rt-1", "reason": "materialize"},
	}
	ci := ri.ToConfigItem()

	// Schema assertions
	if ci.Schema.Key != "launcher.image" {
		t.Errorf("Schema.Key = %q, want %q", ci.Schema.Key, "launcher.image")
	}
	if ci.Schema.Category != "launcher" {
		t.Errorf("Schema.Category = %q, want %q", ci.Schema.Category, "launcher")
	}
	if ci.Schema.Kind != "cli_arg" {
		t.Errorf("Schema.Kind = %q, want %q", ci.Schema.Kind, "cli_arg")
	}
	if ci.Schema.Type != "string" {
		t.Errorf("Schema.Type = %q, want %q", ci.Schema.Type, "string")
	}
	if !ci.Schema.Required {
		t.Error("Schema.Required should be true")
	}
	if ci.Schema.Advanced {
		t.Error("Schema.Advanced should be false")
	}
	if ci.Schema.ReadOnly {
		t.Error("Schema.ReadOnly should be false")
	}
	if ci.Schema.DisplayOrder != 10 {
		t.Errorf("Schema.DisplayOrder = %d, want 10", ci.Schema.DisplayOrder)
	}
	if ci.Schema.SupportLevel != "verified" {
		t.Errorf("Schema.SupportLevel = %q, want %q", ci.Schema.SupportLevel, "verified")
	}
	if ci.Schema.ArgName != "--image" {
		t.Errorf("Schema.ArgName = %q, want %q", ci.Schema.ArgName, "--image")
	}
	if ci.Schema.Label != "Container Image" {
		t.Errorf("Schema.Label = %q, want %q", ci.Schema.Label, "Container Image")
	}
	if ci.Presentation.Group != "launcher" {
		t.Errorf("Presentation.Group = %q, want %q", ci.Presentation.Group, "launcher")
	}

	// Value assertions
	if ci.Value_.DefaultValue != "vllm/vllm-openai:v0.6.0" {
		t.Errorf("Value_.DefaultValue = %v, want %v", ci.Value_.DefaultValue, "vllm/vllm-openai:v0.6.0")
	}
	if ci.Value_.EffectiveValue != "vllm/vllm-openai:latest" {
		t.Errorf("Value_.EffectiveValue = %v, want %v", ci.Value_.EffectiveValue, "vllm/vllm-openai:latest")
	}

	// State assertions
	if !ci.State_.Enabled {
		t.Error("State_.Enabled should be true")
	}
	if !ci.State_.Checked {
		t.Error("State_.Checked should be true for enabled item")
	}
	if !ci.State_.Editable {
		t.Error("State_.Editable should be true for non-readonly item")
	}
	if !ci.State_.Valid {
		t.Error("State_.Valid should default to true")
	}

	// Presentation assertions
	if ci.Presentation.Priority != 10 {
		t.Errorf("Presentation.Priority = %d, want 10", ci.Presentation.Priority)
	}
}

func TestRegistryItemDefaultValueIsEffectiveWhenNotEnabled(t *testing.T) {
	ri := RegistryItem{
		Code:         "model_runtime.gpu_memory_utilization",
		Value:        0.85,
		DefaultValue: 0.9,
		Enabled:      false,
	}
	ci := ri.ToConfigItem()

	if ci.Value_.EffectiveValue != 0.85 {
		t.Errorf("EffectiveValue = %v, want inherited 0.85", ci.Value_.EffectiveValue)
	}
	if ci.State_.Checked {
		t.Error("Checked should be false when enabled is false")
	}
	if ci.Value_.InheritedValue != 0.85 {
		t.Errorf("InheritedValue = %v, want 0.85", ci.Value_.InheritedValue)
	}
	if ci.Value_.LocalValue != nil {
		t.Errorf("LocalValue = %v, want nil (not enabled = not local edit)", ci.Value_.LocalValue)
	}
}

func TestRegistryItemOptionalNotChecked(t *testing.T) {
	ri := RegistryItem{Code: "model_runtime.dtype", Required: false, Enabled: false}
	ci := ri.ToConfigItem()

	if ci.State_.Enabled {
		t.Error("optional item should not be enabled by default")
	}
	if ci.State_.Checked {
		t.Error("optional item should not be checked by default")
	}
}

func TestRegistryItemRequiredDoesNotImplyChecked(t *testing.T) {
	ri := RegistryItem{Code: "service.container_port", Required: true, Enabled: false, Value: 8000}
	ci := ri.ToConfigItem()

	if ci.State_.Checked {
		t.Error("required item should not be checked unless current-layer explicitly enables it")
	}
	if ci.Value_.EffectiveValue != 8000 {
		t.Errorf("EffectiveValue = %v, want 8000", ci.Value_.EffectiveValue)
	}
}

func TestRegistryItemReadOnlyMapsToNotEditable(t *testing.T) {
	ri := RegistryItem{Code: "runtime.health", Readonly: true}
	ci := ri.ToConfigItem()

	if ci.State_.Editable {
		t.Error("readonly item should have Editable=false")
	}
}

func TestRegistryItemHiddenMapsToNotVisible(t *testing.T) {
	ri := RegistryItem{Code: "backend.extra_args", Visibility: "hidden"}
	ci := ri.ToConfigItem()

	if ci.State_.Visible {
		t.Error("hidden item should have Visible=false")
	}
}

func TestConfigItemJSONRoundTripWithTiers(t *testing.T) {
	ci := ConfigItem{
		Schema: ConfigItemSchema{
			Key: "model_runtime.max_model_len", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
			ConfigSetKey: "BackendParameterConfigSet", Category: "model_runtime", Kind: "cli_arg",
			Type: "integer", Required: false, SupportLevel: "documented",
		},
		Value_: ConfigItemValue{DefaultValue: 4096, EffectiveValue: 8192},
		State_: ConfigItemState{Enabled: true, Checked: true, Editable: true, Visible: true, Valid: true},
		Provenance_: ConfigItemProvenance{
			ValueSource: "backend_version_default", LastValueLayer: "BackendVersionConfigBundle",
		},
	}

	b, err := json.Marshal(ci)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ConfigItem
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Schema.Key != "model_runtime.max_model_len" {
		t.Errorf("Schema.Key after round-trip = %q", decoded.Schema.Key)
	}
	if decoded.Schema.Owner != "BackendVersion" {
		t.Errorf("Schema.Owner after round-trip = %q", decoded.Schema.Owner)
	}
	if decoded.Value_.EffectiveValue != float64(8192) {
		t.Errorf("Value_.EffectiveValue after round-trip = %v", decoded.Value_.EffectiveValue)
	}
	if !decoded.State_.Enabled {
		t.Error("State_.Enabled should be true after round-trip")
	}
	if decoded.Provenance_.ValueSource != "backend_version_default" {
		t.Errorf("Provenance_.ValueSource after round-trip = %q", decoded.Provenance_.ValueSource)
	}
}

// ============================================================================
// ConfigSetBundle tests
// ============================================================================

func TestConfigSetBundleEffectiveSnapshotMergesInheritedOwnAndLocalEdits(t *testing.T) {
	bundle := ConfigSetBundle{
		InheritedBundleSnapshots: []ConfigSet{
			{
				ConfigSetKey: "BackendVersionConfigSet",
				Items: map[string]ConfigItem{
					"launcher.image": {
						Schema:  ConfigItemSchema{Key: "launcher.image", Owner: "BackendVersion"},
						Value_:  ConfigItemValue{DefaultValue: "vllm:v0.6.0", EffectiveValue: "vllm:v0.6.0"},
						State_:  ConfigItemState{Enabled: true},
					},
					"model_runtime.gpu_memory_utilization": {
						Schema:  ConfigItemSchema{Key: "model_runtime.gpu_memory_utilization", Owner: "BackendVersion"},
						Value_:  ConfigItemValue{DefaultValue: 0.9, EffectiveValue: 0.9},
						State_:  ConfigItemState{Enabled: false},
					},
				},
			},
		},
		OwnSets: []ConfigSet{
			{
				ConfigSetKey: "BackendRuntimeConfigSet",
				Items: map[string]ConfigItem{
					"launcher.image": {
						Schema: ConfigItemSchema{Key: "launcher.image", Owner: "BackendRuntime"},
						Value_: ConfigItemValue{DefaultValue: "vllm:v0.6.0", LocalValue: "vllm/vllm-openai:latest", EffectiveValue: "vllm/vllm-openai:latest"},
						State_: ConfigItemState{Enabled: true, Checked: true},
					},
				},
			},
		},
		LocalEdits: map[string]map[string]ConfigItemLocalEdit{
			"BackendParameterConfigSet": {
				"model_runtime.gpu_memory_utilization": {
					ConfigSetKey: "BackendParameterConfigSet", ItemKey: "model_runtime.gpu_memory_utilization",
					LocalValue: 0.85, Reason: "node tuned for GPU memory",
				},
			},
		},
	}

	snap := bundle.EffectiveSnapshot()

	if img, ok := snap.Items["launcher.image"]; ok {
		if img.Value_.EffectiveValue != "vllm/vllm-openai:latest" {
			t.Errorf("image effective = %v, want vllm/vllm-openai:latest", img.Value_.EffectiveValue)
		}
	} else {
		t.Error("launcher.image missing from effective snapshot")
	}

	if gmu, ok := snap.Items["model_runtime.gpu_memory_utilization"]; ok {
		if gmu.Value_.EffectiveValue != 0.85 {
			t.Errorf("gpu_memory effective = %v, want 0.85", gmu.Value_.EffectiveValue)
		}
	} else {
		t.Error("gpu_memory_utilization missing from effective snapshot")
	}
}

func TestConfigSetBundleDeepCopySnapshotStampsProvenance(t *testing.T) {
	bundle := ConfigSetBundle{
		InheritedBundleSnapshots: []ConfigSet{
			{
				ConfigSetKey: "BackendVersionConfigSet",
				Items: map[string]ConfigItem{
					"launcher.image": {
						Schema: ConfigItemSchema{Key: "launcher.image", Owner: "BackendVersion"},
						Value_: ConfigItemValue{DefaultValue: "vllm:v0.6.0", EffectiveValue: "vllm:v0.6.0"},
					},
				},
			},
		},
	}

	snap := bundle.DeepCopySnapshot("BackendRuntimeConfigBundle", "rt-123")

	img := snap.Items["launcher.image"]
	if img.Snapshot_.FromLayer != "BackendRuntimeConfigBundle" {
		t.Errorf("Snapshot.FromLayer = %q, want BackendRuntimeConfigBundle", img.Snapshot_.FromLayer)
	}
	if img.Snapshot_.FromID != "rt-123" {
		t.Errorf("Snapshot.FromID = %q, want rt-123", img.Snapshot_.FromID)
	}
	if img.Schema.Owner != "BackendVersion" {
		t.Errorf("Schema.Owner after deep copy = %q, want BackendVersion (unchanged)", img.Schema.Owner)
	}
}

func TestConfigSetBundleEffectiveSnapshotDoesNotMutateOriginal(t *testing.T) {
	bundle := ConfigSetBundle{
		InheritedBundleSnapshots: []ConfigSet{
			{
				ConfigSetKey: "BackendVersionConfigSet",
				Items: map[string]ConfigItem{
					"model_runtime.gpu_memory_utilization": {
						Schema: ConfigItemSchema{Key: "model_runtime.gpu_memory_utilization", Owner: "BackendVersion"},
						Value_: ConfigItemValue{DefaultValue: 0.9, EffectiveValue: 0.9},
					},
				},
			},
		},
		LocalEdits: map[string]map[string]ConfigItemLocalEdit{
			"BackendParameterConfigSet": {
				"model_runtime.gpu_memory_utilization": {LocalValue: 0.82},
			},
		},
	}

	snap := bundle.EffectiveSnapshot()
	if snap.Items["model_runtime.gpu_memory_utilization"].Value_.EffectiveValue != 0.82 {
		t.Error("effective snapshot should reflect local edit")
	}

	bundle2 := ConfigSetBundle{InheritedBundleSnapshots: bundle.InheritedBundleSnapshots}
	snap2 := bundle2.EffectiveSnapshot()
	if snap2.Items["model_runtime.gpu_memory_utilization"].Value_.EffectiveValue != 0.9 {
		t.Errorf("second snapshot should return original value 0.9, got %v",
			snap2.Items["model_runtime.gpu_memory_utilization"].Value_.EffectiveValue)
	}
}

func TestConfigSetOwnSectionsDefineGrouping(t *testing.T) {
	cs := ConfigSet{
		ConfigSetKey: "BackendParameterConfigSet",
		OwnSections: []ConfigSection{
			{Key: "required", Title: "必填配置", Match: map[string]any{"required": true}, DefaultExpanded: true, Priority: 10},
			{Key: "common", Title: "常用配置", Match: map[string]any{"group": "common"}, DefaultExpanded: true, Priority: 20},
			{Key: "advanced", Title: "高级配置", Match: map[string]any{"advanced": true}, DefaultExpanded: false, Priority: 90},
		},
	}

	b, err := json.Marshal(cs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ConfigSet
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.OwnSections) != 3 {
		t.Fatalf("OwnSections count = %d, want 3", len(decoded.OwnSections))
	}
	if decoded.OwnSections[2].Key != "advanced" {
		t.Errorf("third section key = %q, want advanced", decoded.OwnSections[2].Key)
	}
}

func TestConfigSetChildSlotsDefineChildPlacement(t *testing.T) {
	cs := ConfigSet{
		ConfigSetKey: "DeploymentConfigSet",
		ChildSlots: []ConfigChildSlot{
			{Slot: "runtime", ChildConfigSetKey: "node_backend_runtime", Title: "继承的节点运行配置", View: "summary_then_edit", DisplayMode: "panel", DefaultExpanded: true, Order: 30},
			{Slot: "ports", ChildConfigSetKey: "deployment_ports", Title: "端口映射", View: "edit", DisplayMode: "inline", DefaultExpanded: true, Order: 40},
		},
	}

	b, err := json.Marshal(cs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ConfigSet
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.ChildSlots) != 2 {
		t.Fatalf("ChildSlots count = %d, want 2", len(decoded.ChildSlots))
	}
}

func TestSourceChainEntriesCaptureProvenance(t *testing.T) {
	chain := []SourceChainEntry{
		{Layer: "BackendVersionConfigBundle", Value: 0.9, Reason: "schema default"},
		{Layer: "NodeBackendRuntimeConfigBundle", Value: 0.8, Reason: "node local edit"},
		{Layer: "DeploymentConfigBundle", Value: 0.82, Reason: "deployment local edit"},
	}

	b, err := json.Marshal(chain)
	if err != nil {
		t.Fatalf("marshal source chain: %v", err)
	}

	if !strings.Contains(string(b), "schema default") {
		t.Error("source chain JSON should contain reason strings")
	}
	if !strings.Contains(string(b), "deployment local edit") {
		t.Error("source chain JSON should contain deployment edit reason")
	}
}
