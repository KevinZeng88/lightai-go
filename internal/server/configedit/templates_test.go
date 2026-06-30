package configedit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadComponentTemplatesLocalOverridePrecedence(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "builtin")
	local := filepath.Join(root, "local")
	if err := os.MkdirAll(builtin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate := func(dir, display string) {
		t.Helper()
		body := `template_id: test-vllm-configedit-v1
kind: config_edit_template
version: 1
applies_to:
  backend: vllm
  runtime_kind: docker
  vendors: [nvidia]
metadata:
  display_name: ` + display + `
views:
  default_view: normal
  supported_views: [normal, advanced, developer]
sections:
  - key: resources
    label: Resources
    order: 10
    view: normal
components:
  - key: runtime.device_binding
    component: accelerator_binding
    renderer: accelerator_binding
    label: Device Binding
    section: resources
    view: normal
    order: 10
    effects:
      - type: device_binding
        target: docker.gpus
        value_from: docker_gpu_option
`
		if err := os.WriteFile(filepath.Join(dir, "template.yaml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeTemplate(builtin, "Built In")
	writeTemplate(local, "Local")
	store, err := LoadComponentTemplates(builtin, local)
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	if len(store.Templates) != 1 {
		t.Fatalf("templates=%d issues=%#v", len(store.Templates), store.Issues)
	}
	if store.Templates[0].Source != "local" || store.Templates[0].Metadata["display_name"] != "Local" {
		t.Fatalf("local override did not win: %#v", store.Templates[0])
	}
}

func TestValidateComponentTemplateRejectsUnsafeUnknowns(t *testing.T) {
	issues := ValidateComponentTemplate(ComponentTemplate{
		TemplateID: "bad",
		Kind:       "config_edit_template",
		Version:    1,
		AppliesTo:  TemplateAppliesTo{Backend: "vllm"},
		Sections:   []TemplateSection{{Key: "x", View: "normal"}},
		Components: []TemplateComponent{{
			Key:      "x",
			Renderer: "shell",
			View:     "normal",
			Section:  "x",
			Effects:  []TemplateEffect{{Type: "cli_arg", ValueFrom: "exec('rm')"}},
		}},
	})
	if len(issues) == 0 {
		t.Fatal("expected validation issues")
	}
}
