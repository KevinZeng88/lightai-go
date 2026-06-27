package catalog

import (
	"encoding/json"
	"testing"
)

func TestConfigSetGenerateViewWithOwnSections(t *testing.T) {
	cs := ConfigSet{
		ConfigSetKey: "BackendParameterConfigSet",
		Title:        "后端参数",
		OwnSections: []ConfigSection{
			{Key: "required", Title: "必填配置", Match: map[string]any{"required": true}, DefaultExpanded: true, Priority: 10},
			{Key: "advanced", Title: "高级配置", Match: map[string]any{"advanced": true}, DefaultExpanded: false, Priority: 90},
		},
		Items: map[string]ConfigItem{
			"service.container_port": {
				Schema:       ConfigItemSchema{Key: "service.container_port", Type: "integer", Required: true, SupportLevel: "verified", ConfigSetKey: "BackendParameterConfigSet"},
				Value_:       ConfigItemValue{EffectiveValue: int(8000)},
				State_:       ConfigItemState{Enabled: true, Visible: true, Valid: true},
				Presentation: ConfigItemPresentation{Group: "common"},
			},
			"model_runtime.gpu_memory_utilization": {
				Schema:       ConfigItemSchema{Key: "model_runtime.gpu_memory_utilization", Type: "number", Advanced: true, SupportLevel: "documented", ConfigSetKey: "BackendParameterConfigSet"},
				Value_:       ConfigItemValue{EffectiveValue: 0.9},
				State_:       ConfigItemState{Enabled: false, Visible: true, Valid: true},
				Presentation: ConfigItemPresentation{Group: "tuning", Priority: 50},
			},
		},
	}

	view := cs.GenerateView()

	if len(view.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(view.Sections))
	}

	reqSec := view.Sections[0]
	if reqSec.Key != "required" {
		t.Errorf("section[0].key = %q, want required", reqSec.Key)
	}
	if len(reqSec.Fields) != 1 {
		t.Errorf("required section fields = %d, want 1", len(reqSec.Fields))
	}

	advSec := view.Sections[1]
	if advSec.DefaultExpanded {
		t.Error("advanced section should default to collapsed")
	}
}

func TestConfigSetGenerateViewDefaultGrouping(t *testing.T) {
	cs := ConfigSet{
		ConfigSetKey: "TestConfigSet",
		Items: map[string]ConfigItem{
			"req_item": {
				Schema: ConfigItemSchema{Key: "req_item", Required: true, Label: "Required Field"},
				Value_: ConfigItemValue{EffectiveValue: "val1"},
				State_: ConfigItemState{Visible: true},
			},
			"advanced_item": {
				Schema: ConfigItemSchema{Key: "advanced_item", Advanced: true, Label: "Advanced Field"},
				Value_: ConfigItemValue{EffectiveValue: "val3"},
				State_: ConfigItemState{Visible: true},
			},
		},
	}

	view := cs.GenerateView()

	if len(view.Sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(view.Sections))
	}
}

func TestFieldViewPopulatesAllTiers(t *testing.T) {
	item := ConfigItem{
		Schema: ConfigItemSchema{
			Key: "model_runtime.max_model_len", Owner: "BackendVersion", OwnerLayer: "BackendVersionConfigBundle",
			ConfigSetKey: "BackendParameterConfigSet", Type: "integer", Required: false, Advanced: true,
			ReadOnly: false, Label: "最大模型长度", HelpText: "设置模型上下文窗口大小",
		},
		Value_: ConfigItemValue{DefaultValue: 4096, InheritedValue: 8192, EffectiveValue: 8192},
		State_: ConfigItemState{Enabled: false, Checked: false, Editable: true, Visible: true, Valid: true},
		Provenance_: ConfigItemProvenance{
			ValueSource: "backend_version_default", LastValueLayer: "BackendVersionConfigBundle",
			SourceChain: []SourceChainEntry{{Layer: "BackendVersionConfigBundle", Value: 4096, Reason: "schema default"}},
		},
		Snapshot_:    ConfigItemSnapshot{FromLayer: "BackendVersionConfigBundle", FromID: "bv-vllm"},
		Presentation: ConfigItemPresentation{Section: "advanced", Group: "tuning", Priority: 50},
	}

	fv := itemToFieldView(item)

	if fv.Key != "model_runtime.max_model_len" {
		t.Errorf("Key = %q", fv.Key)
	}
	if fv.Label != "最大模型长度" {
		t.Errorf("Label = %q", fv.Label)
	}
	if fv.Type != "integer" {
		t.Errorf("Type = %q", fv.Type)
	}
	if !fv.Advanced {
		t.Error("Advanced should be true")
	}
	if fv.Required {
		t.Error("Required should be false")
	}
	if fv.Value != 8192 {
		t.Errorf("Value = %v, want 8192", fv.Value)
	}
	if fv.DefaultValue != 4096 {
		t.Errorf("DefaultValue = %v, want 4096", fv.DefaultValue)
	}
	if fv.InheritedValue != 8192 {
		t.Errorf("InheritedValue = %v, want 8192", fv.InheritedValue)
	}
	if fv.Enabled {
		t.Error("Enabled should be false for inherited item")
	}
	if fv.Checked {
		t.Error("Checked should be false for inherited item")
	}
	if fv.ValueSource != "backend_version_default" {
		t.Errorf("ValueSource = %q", fv.ValueSource)
	}
}

func TestConfigViewJSONRoundTrip(t *testing.T) {
	view := ConfigView{
		ConfigSetKey: "DeploymentConfigSet",
		Title:        "部署配置",
		Sections: []ViewSection{
			{Key: "required", Title: "必填", DefaultExpanded: true, Priority: 10, Fields: []FieldView{
				{Key: "launcher.image", Label: "镜像", Value: "vllm/vllm-openai:latest", Required: true},
			}},
		},
	}

	b, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal ConfigView: %v", err)
	}

	var decoded ConfigView
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal ConfigView: %v", err)
	}

	if decoded.ConfigSetKey != "DeploymentConfigSet" {
		t.Errorf("ConfigSetKey = %q", decoded.ConfigSetKey)
	}
}

func TestGenerateBundleViewIncludesLocalEdits(t *testing.T) {
	parent := testBackendVersionBundle()
	child := CreateNextLayerBundle(parent, nil, "BackendRuntimeConfigBundle", "rt-123")

	err := ApplyLocalEdit(&child, "BackendParameterConfigSet", "model_runtime.gpu_memory_utilization",
		0.82, boolPtr(true), boolPtr(true), "节点24GB GPU调优", "admin")
	if err != nil {
		t.Fatalf("ApplyLocalEdit: %v", err)
	}

	view := child.GenerateBundleView()

	if len(view.LocalEdits) != 1 {
		t.Fatalf("expected 1 local edit summary, got %d", len(view.LocalEdits))
	}
	edit := view.LocalEdits[0]
	if edit.ItemKey != "model_runtime.gpu_memory_utilization" {
		t.Errorf("edit item key = %q", edit.ItemKey)
	}
	if edit.Value != 0.82 {
		t.Errorf("edit value = %v, want 0.82", edit.Value)
	}
}

func TestFieldViewRequiredNotChecked(t *testing.T) {
	item := ConfigItem{
		Schema: ConfigItemSchema{Key: "service.container_port", Required: true},
		Value_: ConfigItemValue{EffectiveValue: int(8000)},
		State_: ConfigItemState{Enabled: true, Checked: false, Editable: true, Visible: true},
	}
	fv := itemToFieldView(item)

	if !fv.Required {
		t.Error("required should be true")
	}
	if fv.Checked {
		t.Error("required item without local edit should NOT be checked")
	}
}

func TestFieldViewCheckedForLocalEdit(t *testing.T) {
	item := ConfigItem{
		Schema: ConfigItemSchema{Key: "model_runtime.gpu_memory_utilization", Required: false},
		Value_: ConfigItemValue{LocalValue: 0.82, EffectiveValue: 0.82},
		State_: ConfigItemState{Enabled: true, Checked: true, Editable: true, Visible: true},
	}
	fv := itemToFieldView(item)

	if !fv.Enabled {
		t.Error("enabled should be true")
	}
	if !fv.Checked {
		t.Error("locally edited item should be checked")
	}
}

func TestAdvancedSectionDefaultCollapsed(t *testing.T) {
	cs := ConfigSet{
		ConfigSetKey: "Test",
		OwnSections:  []ConfigSection{{Key: "advanced", Title: "高级", DefaultExpanded: false, Priority: 90}},
		Items: map[string]ConfigItem{
			"adv_1": {
				Schema: ConfigItemSchema{Key: "adv_1", Advanced: true},
				Value_: ConfigItemValue{EffectiveValue: "x"},
				State_: ConfigItemState{Visible: true},
			},
		},
	}

	view := cs.GenerateView()
	if len(view.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(view.Sections))
	}
	if view.Sections[0].DefaultExpanded {
		t.Error("advanced section should default to collapsed")
	}
}

func TestCustomRendererRegistry(t *testing.T) {
	testRenderer := &testCustomRenderer{key: "test_renderer"}
	RegisterCustomRenderer("DockerOptionsConfigSet", testRenderer)

	if _, ok := CustomRendererRegistry["DockerOptionsConfigSet"]; !ok {
		t.Error("custom renderer not registered")
	}
}

type testCustomRenderer struct{ key string }

func (r *testCustomRenderer) RenderSection(cs ConfigSet) ViewSection {
	return ViewSection{Key: "docker", Title: "Docker配置", Fields: []FieldView{{Key: "shm_size", Label: "共享内存", Value: "1gb"}}}
}
