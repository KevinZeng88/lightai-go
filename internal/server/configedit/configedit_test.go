package configedit

import "testing"

func testConfigSet() map[string]any {
	return map[string]any{
		"schema_version": 1,
		"items": map[string]any{
			"launcher.image": map[string]any{
				"code": "launcher.image", "category": "launcher", "kind": "image", "type": "string",
				"value": "vllm:test", "enabled": true, "required": true,
			},
			"launcher.docker_options": map[string]any{
				"code": "launcher.docker_options", "category": "launcher", "kind": "docker_options", "type": "object",
				"enabled": true,
				"value": map[string]any{
					"shm_size":   "16gb",
					"privileged": false,
					"devices":    []any{"/dev/nvidia0"},
					"group_add":  []any{"video"},
				},
			},
			"runtime.env": map[string]any{
				"code": "runtime.env", "category": "runtime", "kind": "env", "type": "object",
				"enabled": true,
				"value":   map[string]any{"HF_HOME": "/cache/hf"},
			},
			"backend.arg.fake_new_param": map[string]any{
				"code": "backend.arg.fake_new_param", "category": "model_runtime", "kind": "cli_arg", "type": "string",
				"value": "abc", "enabled": false,
				"render": map[string]any{"flag": "--fake-new-param"},
			},
			"internal.checksum": map[string]any{
				"code": "internal.checksum", "category": "internal", "kind": "metadata", "type": "string",
				"value": "sha", "enabled": true,
			},
		},
	}
}

func TestProjectConfigSetToEditViewHidesInternalKeysAndSplitsDockerOptions(t *testing.T) {
	view, err := ProjectConfigSetToEditView(ProjectInput{
		ConfigSet:   testConfigSet(),
		Layer:       "backend_runtime",
		ObjectKind:  "backend_runtime",
		ObjectID:    "rt-test",
		ObjectLabel: "Runtime Test",
	})
	if err != nil {
		t.Fatalf("project view: %v", err)
	}

	fields := flattenFields(view)
	if len(fields) == 0 {
		t.Fatal("expected fields")
	}
	for _, field := range fields {
		if !field.Advanced && (field.Label == "launcher.image" || field.Label == "launcher.docker_options" || field.Label == "runtime.env") {
			t.Fatalf("ordinary label exposes internal key: %#v", field)
		}
	}
	requireField(t, fields, "launcher.docker_options.shm_size", "launcher.docker_options", []string{"shm_size"}, "container_resources")
	requireField(t, fields, "launcher.docker_options.privileged", "launcher.docker_options", []string{"privileged"}, "container_resources")
	requireField(t, fields, "launcher.docker_options.devices", "launcher.docker_options", []string{"devices"}, "devices_mounts")
	requireField(t, fields, "launcher.docker_options.group_add", "launcher.docker_options", []string{"group_add"}, "devices_mounts")
	requireField(t, fields, "backend.arg.fake_new_param", "backend.arg.fake_new_param", nil, "model_serving")

	requiredImage := requireField(t, fields, "launcher.image", "launcher.image", nil, "basic")
	if requiredImage.HasEnable || !requiredImage.Enabled || !requiredImage.Required {
		t.Fatalf("required image should be enabled without user-toggle: %#v", requiredImage)
	}
	optionalParam := requireField(t, fields, "backend.arg.fake_new_param", "backend.arg.fake_new_param", nil, "model_serving")
	if !optionalParam.HasEnable || optionalParam.Enabled {
		t.Fatalf("optional param should expose disabled enable checkbox: %#v", optionalParam)
	}
	internal := requireField(t, fields, "internal.checksum", "internal.checksum", nil, "advanced_raw")
	if !internal.Advanced {
		t.Fatalf("internal field should be advanced/raw only: %#v", internal)
	}
}

func TestApplyEditPatchToConfigSetMergesDockerOptionsAndForcesRequiredEnabled(t *testing.T) {
	out, err := ApplyEditPatchToConfigSet(testConfigSet(), ConfigEditPatch{
		Layer:    "backend_runtime",
		ObjectID: "rt-test",
		Fields: []EditFieldPatch{
			{Key: "launcher.docker_options.shm_size", InternalKey: "launcher.docker_options", Path: []string{"shm_size"}, Value: "24gb", Enabled: boolPtr(true)},
			{Key: "launcher.docker_options.privileged", InternalKey: "launcher.docker_options", Path: []string{"privileged"}, Value: true, Enabled: boolPtr(true)},
			{Key: "backend.arg.fake_new_param", InternalKey: "backend.arg.fake_new_param", Value: "xyz", Enabled: boolPtr(true)},
			{Key: "launcher.image", InternalKey: "launcher.image", Value: "vllm:changed", Enabled: boolPtr(false)},
		},
	}, "BackendRuntime", "rt-test")
	if err != nil {
		t.Fatalf("apply patch: %v", err)
	}
	items := out["items"].(map[string]any)
	docker := items["launcher.docker_options"].(map[string]any)["value"].(map[string]any)
	if docker["shm_size"] != "24gb" || docker["privileged"] != true {
		t.Fatalf("docker options not merged: %#v", docker)
	}
	image := items["launcher.image"].(map[string]any)
	if image["enabled"] != true {
		t.Fatalf("required image enabled should be forced true: %#v", image)
	}
	param := items["backend.arg.fake_new_param"].(map[string]any)
	if param["value"] != "xyz" || param["enabled"] != true {
		t.Fatalf("param patch not applied: %#v", param)
	}
}

func TestValidateEditPatchRejectsDeploymentProtectedFields(t *testing.T) {
	err := ValidateEditPatch(testConfigSet(), ConfigEditPatch{
		Layer:    "deployment",
		ObjectID: "dep-test",
		Fields: []EditFieldPatch{
			{Key: "launcher.image", InternalKey: "launcher.image", Value: "should-not-change"},
		},
	})
	if err == nil {
		t.Fatal("expected deployment protected field validation error")
	}
}

func flattenFields(view ConfigEditView) []EditField {
	var out []EditField
	for _, section := range view.Sections {
		out = append(out, section.Fields...)
	}
	return out
}

func requireField(t *testing.T, fields []EditField, key, internal string, path []string, section string) EditField {
	t.Helper()
	for _, field := range fields {
		if field.Key != key {
			continue
		}
		if field.InternalKey != internal || field.Section != section {
			t.Fatalf("field %s mismatch: %#v", key, field)
		}
		if len(path) != len(field.Path) {
			t.Fatalf("field %s path mismatch: %#v", key, field.Path)
		}
		for i := range path {
			if path[i] != field.Path[i] {
				t.Fatalf("field %s path mismatch: %#v", key, field.Path)
			}
		}
		return field
	}
	t.Fatalf("missing field %s", key)
	return EditField{}
}

func boolPtr(v bool) *bool { return &v }
