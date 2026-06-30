package configedit

import "testing"

func TestProjectConfigSetToEditViewReturnsObjectContract(t *testing.T) {
	view, err := ProjectConfigSetToEditView(ProjectInput{
		ConfigSet:  testConfigSet(),
		Layer:      "deployment",
		ObjectKind: "deployment",
		ObjectID:   "dep-1",
		TemplateID: "vllm-nvidia-docker-configedit-v1",
		SnapshotID: "snapshot-1",
		Parent:     &ObjectRef{ObjectKind: "node_backend_runtime", ObjectID: "nbr-1", SnapshotID: "snapshot-parent"},
		ViewLevel:  "developer",
	})
	if err != nil {
		t.Fatalf("project object: %v", err)
	}
	if view.ObjectKind != "deployment" || view.ObjectID != "dep-1" {
		t.Fatalf("identity missing: %#v", view)
	}
	if view.TemplateID == "" || view.SnapshotID == "" {
		t.Fatalf("template/snapshot missing: %#v", view)
	}
	if view.Parent == nil || view.Parent.ObjectKind != "node_backend_runtime" {
		t.Fatalf("parent missing: %#v", view.Parent)
	}
	if view.ChildInit.Strategy != "copy_effective_snapshot" {
		t.Fatalf("child init contract missing: %#v", view.ChildInit)
	}
	if len(view.Fields) == 0 || len(view.Components) == 0 {
		t.Fatalf("fields/components missing: fields=%d components=%d", len(view.Fields), len(view.Components))
	}
	if len(view.EffectsPreview) == 0 {
		t.Fatalf("effects preview missing")
	}
}

func TestResetFieldToDefaultAndParent(t *testing.T) {
	set := testConfigSet()
	items := set["items"].(map[string]any)
	items["backend.arg.resettable"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.resettable", "category": "model_runtime", "kind": "cli_arg", "type": "string"},
		"state":  map[string]any{"enabled": true, "editable": true, "visible": true},
		"value":  map[string]any{"effective_value": "local", "default_value": "default", "inherited_value": "parent"},
	}
	out, err := ResetFieldToDefault(set, "backend.arg.resettable", nil, "deployment", "dep-1")
	if err != nil {
		t.Fatalf("reset default: %v", err)
	}
	item := out["items"].(map[string]any)["backend.arg.resettable"].(map[string]any)
	if itemEffectiveValue(item) != "default" {
		t.Fatalf("reset default value=%#v", item)
	}
	out, err = ResetFieldToParent(out, "backend.arg.resettable", nil, "deployment", "dep-1")
	if err != nil {
		t.Fatalf("reset parent: %v", err)
	}
	item = out["items"].(map[string]any)["backend.arg.resettable"].(map[string]any)
	if itemEffectiveValue(item) != "parent" {
		t.Fatalf("reset parent value=%#v", item)
	}
}
