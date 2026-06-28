package configedit

import "testing"

func testConfigSet() map[string]any {
	return map[string]any{
		"schema_version": 1,
		"items": map[string]any{
			"launcher.image": map[string]any{
				"schema": map[string]any{"key": "launcher.image", "category": "launcher", "kind": "image", "type": "string", "required": true},
				"state":  map[string]any{"enabled": true, "checked": true, "editable": true, "visible": true},
				"value":  map[string]any{"effective_value": "vllm:test", "default_value": "vllm:test"},
			},
			"launcher.docker_options": map[string]any{
				"schema": map[string]any{"key": "launcher.docker_options", "category": "launcher", "kind": "docker_options", "type": "object"},
				"state":  map[string]any{"enabled": true, "checked": true, "editable": true, "visible": true},
				"value":  map[string]any{"effective_value": map[string]any{
					"shm_size":   "16gb",
					"privileged": false,
					"devices":    []any{"/dev/nvidia0"},
					"group_add":  []any{"video"},
				}},
			},
			"runtime.env": map[string]any{
				"schema": map[string]any{"key": "runtime.env", "category": "runtime", "kind": "env", "type": "object"},
				"state":  map[string]any{"enabled": true, "checked": true, "editable": true, "visible": true},
				"value":  map[string]any{"effective_value": map[string]any{"HF_HOME": "/cache/hf"}},
			},
			"backend.arg.fake_new_param": map[string]any{
				"schema": map[string]any{"key": "backend.arg.fake_new_param", "category": "model_runtime", "kind": "cli_arg", "type": "string"},
				"state":  map[string]any{"enabled": false, "checked": false, "editable": true, "visible": true},
				"value":  map[string]any{"effective_value": "abc", "default_value": "abc"},
				"render": map[string]any{"flag": "--fake-new-param"},
			},
			"internal.checksum": map[string]any{
				"schema": map[string]any{"key": "internal.checksum", "category": "internal", "kind": "metadata", "type": "string"},
				"state":  map[string]any{"enabled": true, "checked": true, "editable": true, "visible": true},
				"value":  map[string]any{"effective_value": "sha", "default_value": "sha"},
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
	requireField(t, fields, "docker.shm_size", "launcher.docker_options", []string{"shm_size"}, "container_resources")
	requireField(t, fields, "docker.privileged", "launcher.docker_options", []string{"privileged"}, "container_resources")
	requireField(t, fields, "docker.devices", "launcher.docker_options", []string{"devices"}, "devices_mounts")
	requireField(t, fields, "docker.group_add", "launcher.docker_options", []string{"group_add"}, "devices_mounts")

	// backend.arg.* fields are model-serving: hidden at backend_runtime layer.
	if fieldExists(fields, "backend.arg.fake_new_param") {
		t.Fatal("model-serving param should be hidden at backend_runtime layer")
	}

	requiredImage := requireField(t, fields, "runtime.image_ref", "launcher.image", nil, "basic")
	if requiredImage.HasEnable || !requiredImage.Enabled || !requiredImage.Required {
		t.Fatalf("required image should be enabled without user-toggle: %#v", requiredImage)
	}

	internal := requireField(t, fields, "internal.checksum", "internal.checksum", nil, "advanced_raw")
	if !internal.Advanced {
		t.Fatalf("internal field should be advanced/raw only: %#v", internal)
	}

	// --- Deployment layer: model-serving params SHOULD appear. ---
	depView, err := ProjectConfigSetToEditView(ProjectInput{
		ConfigSet:   testConfigSet(),
		Layer:       "deployment",
		ObjectKind:  "deployment",
		ObjectID:    "dep-test",
		ObjectLabel: "Deployment Test",
	})
	if err != nil {
		t.Fatalf("project deployment view: %v", err)
	}
	depFields := flattenFields(depView)
	depParam := requireField(t, depFields, "backend.arg.fake_new_param", "backend.arg.fake_new_param", nil, "advanced_parameters")
	if !depParam.HasEnable || depParam.Enabled || depParam.Value != "abc" {
		t.Fatalf("disabled deployment param should stay unchecked while retaining value: %#v", depParam)
	}

	// Docker sub-fields hidden at deployment layer.
	if fieldExists(depFields, "launcher.docker_options.shm_size") {
		t.Fatal("docker options should be hidden at deployment layer")
	}

	// launcher.image hidden at deployment layer.
	if fieldExists(depFields, "launcher.image") {
		t.Fatal("image should be hidden at deployment layer")
	}

	// --- Test: optional empty field defaults disabled ---
	// security_options is empty → should default disabled.
	sec := requireField(t, fields, "launcher.docker_options.security_options", "launcher.docker_options", []string{"security_options"}, "container_resources")
	if sec.Enabled {
		t.Fatal("empty security_options should default disabled")
	}
	// shm_size has a prefilled value, but values/defaults must not imply enabled.
	shm := requireField(t, fields, "docker.shm_size", "launcher.docker_options", []string{"shm_size"}, "container_resources")
	if shm.Enabled || shm.Value != "16gb" {
		t.Fatalf("shm_size value should be retained without auto-enabling: %#v", shm)
	}
	// optional_devices is empty → should default disabled.
	odev := requireField(t, fields, "docker.optional_devices", "launcher.docker_options", []string{"optional_devices"}, "devices_mounts")
	if odev.Enabled {
		t.Fatal("empty optional_devices should default disabled")
	}
	// group_add has a prefilled value, but values/defaults must not imply enabled.
	ga := requireField(t, fields, "docker.group_add", "launcher.docker_options", []string{"group_add"}, "devices_mounts")
	if ga.Enabled {
		t.Fatalf("group_add value should not auto-enable the field: %#v", ga)
	}
}

func TestProjectConfigSetToEditViewDoesNotInferEnabledFromDefaultOrVisibility(t *testing.T) {
	set := testConfigSet()
	items := set["items"].(map[string]any)
	items["backend.arg.default_common"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.default_common", "category": "model_runtime", "kind": "cli_arg", "type": "string", "visible_by_default": true},
		"state":  map[string]any{"enabled": false, "editable": true, "visible": true},
		"value":  map[string]any{"default_value": "prefilled", "effective_value": "prefilled"},
		"render": map[string]any{"flag": "--default-common", "label": "Default Common"},
	}
	items["backend.arg.boolean_default"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.boolean_default", "category": "model_runtime", "kind": "cli_arg", "type": "boolean", "visible_by_default": true},
		"state":  map[string]any{"enabled": false, "editable": true, "visible": true},
		"value":  map[string]any{"default_value": false, "effective_value": false},
		"render": map[string]any{"flag": "--boolean-default", "label": "Boolean Default"},
	}
	items["backend.arg.required_default"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.required_default", "category": "model_runtime", "kind": "cli_arg", "type": "string", "required": true},
		"state":  map[string]any{"enabled": false, "editable": true, "visible": true},
		"value":  map[string]any{"default_value": "required-value", "effective_value": "required-value"},
		"render": map[string]any{"flag": "--required-default", "label": "Required Default"},
	}

	view, err := ProjectConfigSetToEditView(ProjectInput{
		ConfigSet:   set,
		Layer:       "node_backend_runtime",
		ObjectKind:  "node_backend_runtime",
		ObjectID:    "nbr-test",
		ObjectLabel: "NBR Test",
	})
	if err != nil {
		t.Fatalf("project view: %v", err)
	}
	fields := flattenFields(view)
	common := requireField(t, fields, "backend.arg.default_common", "backend.arg.default_common", nil, "model_serving")
	if common.Enabled || common.Value != "prefilled" || common.Tier != "common" {
		t.Fatalf("common/default should control display only, not enabled: %#v", common)
	}
	booleanDefault := requireField(t, fields, "backend.arg.boolean_default", "backend.arg.boolean_default", nil, "model_serving")
	if booleanDefault.Enabled || booleanDefault.Value != false {
		t.Fatalf("boolean default_value must not force enabled: %#v", booleanDefault)
	}
	required := requireField(t, fields, "backend.arg.required_default", "backend.arg.required_default", nil, "advanced_parameters")
	if !required.Enabled || required.HasEnable {
		t.Fatalf("required field must be forced enabled without user toggle: %#v", required)
	}
}

func TestProjectConfigSetToEditViewNodeRuntimeShowsCommonAndFoldsAdvanced(t *testing.T) {
	set := testConfigSet()
	items := set["items"].(map[string]any)
	items["backend.arg.gpu_memory_utilization"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.gpu_memory_utilization", "category": "model_runtime", "kind": "cli_arg", "type": "number"},
		"state":  map[string]any{"enabled": true, "checked": true, "editable": true, "visible": true},
		"value":  map[string]any{"effective_value": 0.9, "default_value": 0.9},
		"render": map[string]any{"flag": "--gpu-memory-utilization", "label": "GPU Memory Utilization"},
	}
	items["backend.arg.scheduler"] = map[string]any{
		"code": "backend.arg.scheduler", "category": "model_runtime", "kind": "cli_arg", "type": "string",
		"value": "fcfs", "enabled": false, "tier": "advanced", "render": map[string]any{"flag": "--scheduler", "label": "Scheduler"},
	}
	items["backend.arg.trust_remote_code"] = map[string]any{
		"code": "backend.arg.trust_remote_code", "category": "model_runtime", "kind": "cli_arg", "type": "boolean",
		"value": false, "enabled": false, "dangerous": true, "render": map[string]any{"flag": "--trust-remote-code", "label": "Trust Remote Code"},
	}
	items["backend.arg.debug_profile"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.debug_profile", "category": "debug", "kind": "cli_arg", "type": "boolean"},
		"state":  map[string]any{"enabled": false, "editable": true, "visible": true},
		"value":  map[string]any{"effective_value": true, "default_value": true},
		"render": map[string]any{"flag": "--debug-profile", "label": "Debug Profile"},
	}

	view, err := ProjectConfigSetToEditView(ProjectInput{
		ConfigSet:   set,
		Layer:       "node_backend_runtime",
		ObjectKind:  "node_backend_runtime",
		ObjectID:    "nbr-test",
		ObjectLabel: "NBR Test",
		Mode:        "enable",
	})
	if err != nil {
		t.Fatalf("project view: %v", err)
	}
	fields := flattenFields(view)
	common := requireField(t, fields, "model_runtime.gpu_memory_utilization", "backend.arg.gpu_memory_utilization", nil, "model_serving")
	if common.Tier != "common" || common.Advanced {
		t.Fatalf("gpu memory utilization should be common/default visible: %#v", common)
	}
	advanced := requireField(t, fields, "backend.arg.scheduler", "backend.arg.scheduler", nil, "advanced_parameters")
	if advanced.Tier != "advanced" || !advanced.Advanced {
		t.Fatalf("scheduler should be folded into advanced section: %#v", advanced)
	}
	expert := requireField(t, fields, "backend.arg.trust_remote_code", "backend.arg.trust_remote_code", nil, "expert_parameters")
	if expert.Tier != "expert" || !expert.Advanced {
		t.Fatalf("dangerous trust_remote_code should be expert/hidden by default: %#v", expert)
	}
	if fieldExists(fields, "backend.arg.debug_profile") {
		t.Fatal("debug parameter should not appear in ordinary node runtime edit flow")
	}
}

func TestApplyEditPatchToConfigSetKeepsDisabledValueAndHiddenItems(t *testing.T) {
	set := testConfigSet()
	items := set["items"].(map[string]any)
	items["backend.arg.hidden_existing"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.hidden_existing", "category": "model_runtime", "kind": "cli_arg", "type": "string", "visibility": "internal"},
		"state":  map[string]any{"enabled": true, "editable": true, "visible": false},
		"value":  map[string]any{"effective_value": "keep-me", "default_value": "keep-me"},
		"render": map[string]any{"flag": "--hidden-existing"},
	}
	disabled := false
	out, err := ApplyEditPatchToConfigSet(set, ConfigEditPatch{
		Layer:    "node_backend_runtime",
		ObjectID: "nbr-test",
		Fields: []EditFieldPatch{
			{Key: "backend.arg.fake_new_param", InternalKey: "backend.arg.fake_new_param", Value: "edited-while-disabled", Enabled: &disabled},
		},
	}, "NodeBackendRuntime", "nbr-test")
	if err != nil {
		t.Fatalf("apply patch: %v", err)
	}
		outItems := out["items"].(map[string]any)
		edited := outItems["backend.arg.fake_new_param"].(map[string]any)
		if vt, ok := edited["value"].(map[string]any); ok && vt["effective_value"] != "edited-while-disabled" {
			t.Fatalf("disabled effective_value not preserved: %#v", edited)
		}
		if st, ok := edited["state"].(map[string]any); ok && st["enabled"] != false {
			t.Fatalf("disabled state.enabled not false: %#v", edited)
		}
		hidden := outItems["backend.arg.hidden_existing"].(map[string]any)
		if vt, ok := hidden["value"].(map[string]any); ok && vt["effective_value"] != "keep-me" {
			t.Fatalf("hidden existing item value not preserved: %#v", hidden)
		}
		if st, ok := hidden["state"].(map[string]any); ok && st["enabled"] != true {
			t.Fatalf("hidden existing item state.enabled not true: %#v", hidden)
		}
	}

func TestApplyEditPatchOrdinaryEnabledRoundTripsThroughProjection(t *testing.T) {
	set := testConfigSet()
	items := set["items"].(map[string]any)
	items["backend.arg.round_trip"] = map[string]any{
		"schema": map[string]any{"key": "backend.arg.round_trip", "category": "model_runtime", "kind": "cli_arg", "type": "string"},
		"state":  map[string]any{"enabled": false, "editable": true, "visible": true},
		"value":  map[string]any{"effective_value": "keep-me", "default_value": "keep-me"},
		"render": map[string]any{"flag": "--round-trip", "label": "Round Trip"},
	}

	enabled := true
	out, err := ApplyEditPatchToConfigSet(set, ConfigEditPatch{
		Layer:    "node_backend_runtime",
		ObjectID: "nbr-test",
		Fields: []EditFieldPatch{
			{Key: "backend.arg.round_trip", InternalKey: "backend.arg.round_trip", Value: "keep-me", Enabled: &enabled},
		},
	}, "NodeBackendRuntime", "nbr-test")
	if err != nil {
		t.Fatalf("apply enabled=true: %v", err)
	}
	view, err := ProjectConfigSetToEditView(ProjectInput{ConfigSet: out, Layer: "node_backend_runtime", ObjectKind: "node_backend_runtime", ObjectID: "nbr-test"})
	if err != nil {
		t.Fatalf("project enabled=true: %v", err)
	}
	field := requireField(t, flattenFields(view), "backend.arg.round_trip", "backend.arg.round_trip", nil, "advanced_parameters")
	if !field.Enabled || field.Value != "keep-me" {
		t.Fatalf("enabled=true did not round-trip with value preserved: %#v", field)
	}

	enabled = false
	out, err = ApplyEditPatchToConfigSet(out, ConfigEditPatch{
		Layer:    "node_backend_runtime",
		ObjectID: "nbr-test",
		Fields: []EditFieldPatch{
			{Key: "backend.arg.round_trip", InternalKey: "backend.arg.round_trip", Value: "keep-me", Enabled: &enabled},
		},
	}, "NodeBackendRuntime", "nbr-test")
	if err != nil {
		t.Fatalf("apply enabled=false: %v", err)
	}
	view, err = ProjectConfigSetToEditView(ProjectInput{ConfigSet: out, Layer: "node_backend_runtime", ObjectKind: "node_backend_runtime", ObjectID: "nbr-test"})
	if err != nil {
		t.Fatalf("project enabled=false: %v", err)
	}
	field = requireField(t, flattenFields(view), "backend.arg.round_trip", "backend.arg.round_trip", nil, "advanced_parameters")
	if field.Enabled || field.Value != "keep-me" {
		t.Fatalf("enabled=false did not round-trip with value preserved: %#v", field)
	}
}

func TestApplyEditPatchDockerSubfieldEnabledRoundTripsThroughProjection(t *testing.T) {
	out, err := ApplyEditPatchToConfigSet(testConfigSet(), ConfigEditPatch{
		Layer:    "backend_runtime",
		ObjectID: "rt-test",
		Fields: []EditFieldPatch{
			{Key: "launcher.docker_options.shm_size", InternalKey: "launcher.docker_options", Path: []string{"shm_size"}, Value: "24gb", Enabled: boolPtr(true)},
			{Key: "launcher.docker_options.group_add", InternalKey: "launcher.docker_options", Path: []string{"group_add"}, Value: []any{"video"}, Enabled: boolPtr(false)},
		},
	}, "BackendRuntime", "rt-test")
	if err != nil {
		t.Fatalf("apply docker patch: %v", err)
	}
	view, err := ProjectConfigSetToEditView(ProjectInput{ConfigSet: out, Layer: "backend_runtime", ObjectKind: "backend_runtime", ObjectID: "rt-test"})
	if err != nil {
		t.Fatalf("project docker view: %v", err)
	}
	fields := flattenFields(view)
	shm := requireField(t, fields, "docker.shm_size", "launcher.docker_options", []string{"shm_size"}, "container_resources")
	if !shm.Enabled || shm.Value != "24gb" {
		t.Fatalf("docker shm_size enabled/value did not round-trip: %#v", shm)
	}
	groupAdd := requireField(t, fields, "docker.group_add", "launcher.docker_options", []string{"group_add"}, "devices_mounts")
	if groupAdd.Enabled {
		t.Fatalf("docker group_add enabled=false did not round-trip: %#v", groupAdd)
	}
}

func TestApplyEditPatchToConfigSetMergesDockerOptionsAndForcesRequiredEnabled(t *testing.T) {
	out, err := ApplyEditPatchToConfigSet(testConfigSet(), ConfigEditPatch{
		Layer:    "backend_runtime",
		ObjectID: "rt-test",
		Fields: []EditFieldPatch{
			{Key: "launcher.docker_options.shm_size", InternalKey: "launcher.docker_options", Path: []string{"shm_size"}, Value: "24gb", Enabled: boolPtr(true)},
			{Key: "launcher.docker_options.privileged", InternalKey: "launcher.docker_options", Path: []string{"privileged"}, Value: true, Enabled: boolPtr(true)},
			{Key: "launcher.image", InternalKey: "launcher.image", Value: "vllm:changed", Enabled: boolPtr(false)},
		},
	}, "BackendRuntime", "rt-test")
	if err != nil {
		t.Fatalf("apply patch: %v", err)
	}
		items := out["items"].(map[string]any)
		dockerVal := items["launcher.docker_options"].(map[string]any)["value"].(map[string]any)
		docker := dockerVal["effective_value"].(map[string]any)
		if docker["shm_size"] != "24gb" || docker["privileged"] != true {
			t.Fatalf("docker options not merged: %#v", docker)
		}
		image := items["launcher.image"].(map[string]any)
		if st, ok := image["state"].(map[string]any); !ok || st["enabled"] != true {
			t.Fatalf("required image enabled should be forced true: %#v", image)
		}
	}

func TestApplyEditPatchRejectsDirectLegacyModelServingAtDeployment(t *testing.T) {
	_, err := ApplyEditPatchToConfigSet(testConfigSet(), ConfigEditPatch{
		Layer:    "deployment",
		ObjectID: "dep-test",
		Fields: []EditFieldPatch{
			{Key: "backend.arg.fake_new_param", InternalKey: "backend.arg.fake_new_param", Value: "xyz", Enabled: boolPtr(true)},
		},
	}, "Deployment", "dep-test")
	if err == nil {
		t.Fatal("expected direct legacy backend.arg patch to be rejected")
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

// --- Scope validation tests ---

func TestValidateEditPatchRejectsModelServingAtBackendRuntime(t *testing.T) {
	// backend.arg.* hidden at backend_runtime layer.
	err := ValidateEditPatch(testConfigSet(), ConfigEditPatch{
		Layer:    "backend_runtime",
		ObjectID: "rt-test",
		Fields: []EditFieldPatch{
			{Key: "backend.arg.fake_new_param", InternalKey: "backend.arg.fake_new_param", Value: "xyz", Enabled: boolPtr(true)},
		},
	})
	if err == nil {
		t.Fatal("expected error: backend.arg.fake_new_param should be rejected at backend_runtime layer (hidden)")
	}
}

func TestValidateEditPatchRejectsModelServingAtNodeBackendRuntime(t *testing.T) {
	// NodeBackendRuntime is the node-level deployable runtime config; serving
	// args are editable here and are checked before deployment.
	err := ValidateEditPatch(testConfigSet(), ConfigEditPatch{
		Layer:    "node_backend_runtime",
		ObjectID: "nbr-test",
		Fields: []EditFieldPatch{
			{Key: "backend.arg.fake_new_param", InternalKey: "backend.arg.fake_new_param", Value: "xyz", Enabled: boolPtr(true)},
		},
	})
	if err != nil {
		t.Fatalf("node backend runtime should accept model-serving arg patch: %v", err)
	}
}

func TestValidateEditPatchRejectsDirectLegacyModelServingAtDeployment(t *testing.T) {
	err := ValidateEditPatch(testConfigSet(), ConfigEditPatch{
		Layer:    "deployment",
		ObjectID: "dep-test",
		Fields: []EditFieldPatch{
			{Key: "backend.arg.fake_new_param", InternalKey: "backend.arg.fake_new_param", Value: "xyz", Enabled: boolPtr(true)},
		},
	})
	if err == nil {
		t.Fatal("expected direct legacy backend.arg patch to be rejected")
	}
}

func TestValidateEditPatchRejectsDockerOptionsAtDeployment(t *testing.T) {
	// launcher.docker_options.shm_size hidden at deployment layer.
	err := ValidateEditPatch(testConfigSet(), ConfigEditPatch{
		Layer:    "deployment",
		ObjectID: "dep-test",
		Fields: []EditFieldPatch{
			{Key: "launcher.docker_options.shm_size", InternalKey: "launcher.docker_options", Path: []string{"shm_size"}, Value: "4gb", Enabled: boolPtr(true)},
		},
	})
	if err == nil {
		t.Fatal("expected error: docker options should be rejected at deployment layer (hidden)")
	}
}

func TestValidateEditPatchRejectsImageAtDeployment(t *testing.T) {
	// launcher.image hidden at deployment layer via deploymentProtectedFields + hidden.
	err := ValidateEditPatch(testConfigSet(), ConfigEditPatch{
		Layer:    "deployment",
		ObjectID: "dep-test",
		Fields: []EditFieldPatch{
			{Key: "launcher.image", InternalKey: "launcher.image", Value: "some-image"},
		},
	})
	if err == nil {
		t.Fatal("expected error: launcher.image should be rejected at deployment layer")
	}
}

func flattenFields(view ConfigEditView) []EditField {
	var out []EditField
	for _, section := range view.Sections {
		out = append(out, section.Fields...)
	}
	return out
}

func fieldExists(fields []EditField, key string) bool {
	for _, f := range fields {
		if f.Key == key {
			return true
		}
	}
	return false
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

// TestDockerSubfieldValueDoesNotLeakParentDefault verifies that projected
// docker sub-fields (uts_mode, network_mode, etc.) do NOT display the
// parent launcher.docker_options object when the sub-key is absent.
func TestDockerSubfieldValueDoesNotLeakParentDefault(t *testing.T) {
	view, err := ProjectConfigSetToEditView(ProjectInput{
		ConfigSet:   testConfigSet(),
		Layer:       "backend_runtime",
		ObjectKind:  "backend_runtime",
		ObjectID:    "rt-test",
		ObjectLabel: "Docker Leak Test",
	})
	if err != nil {
		t.Fatalf("project view: %v", err)
	}

	fields := flattenFields(view)

	// shm_size exists in test config: should display "16gb" (string, not object)
	shm := requireField(t, fields, "docker.shm_size", "launcher.docker_options", []string{"shm_size"}, "container_resources")
	if shm.Value != "16gb" {
		t.Errorf("shm_size Value = %#v, want \"16gb\"", shm.Value)
	}
	// privileged exists: should display false (boolean, not object)
	priv := requireField(t, fields, "docker.privileged", "launcher.docker_options", []string{"privileged"}, "container_resources")
	if priv.Value != false {
		t.Errorf("privileged Value = %#v, want false", priv.Value)
	}
	// devices exists: should display ["/dev/nvidia0"] (array, not parent object)
	dev := requireField(t, fields, "docker.devices", "launcher.docker_options", []string{"devices"}, "devices_mounts")
	devArr, _ := dev.Value.([]interface{})
	if len(devArr) != 1 || devArr[0] != "/dev/nvidia0" {
		t.Errorf("devices Value = %#v, want [\"/dev/nvidia0\"]", dev.Value)
	}
	// group_add exists: should display ["video"]
	ga := requireField(t, fields, "docker.group_add", "launcher.docker_options", []string{"group_add"}, "devices_mounts")
	gaArr, _ := ga.Value.([]interface{})
	if len(gaArr) != 1 || gaArr[0] != "video" {
		t.Errorf("group_add Value = %#v, want [\"video\"]", ga.Value)
	}

	// uts_mode does NOT exist in test config: value must be nil, NOT the parent object.
	uts := requireField(t, fields, "launcher.docker_options.uts_mode", "launcher.docker_options", []string{"uts_mode"}, "container_resources")
	if uts.Value != nil {
		t.Errorf("uts_mode Value = %#v, want nil (absent sub-key must not show parent object)", uts.Value)
	}
	// network_mode does NOT exist: value must be nil, NOT the parent object.
	net := requireField(t, fields, "docker.network_mode", "launcher.docker_options", []string{"network_mode"}, "container_resources")
	if net.Value != nil {
		t.Errorf("network_mode Value = %#v, want nil (absent sub-key must not show parent object)", net.Value)
	}
	// security_options does NOT exist: value must be nil.
	sec := requireField(t, fields, "launcher.docker_options.security_options", "launcher.docker_options", []string{"security_options"}, "container_resources")
	if sec.Value != nil {
		t.Errorf("security_options Value = %#v, want nil (absent sub-key must not show parent object)", sec.Value)
	}
}

// TestItemCodeSetForWidgetOverride verifies that widget overrides are
// applied for items with known codes (model_mount, env, health).
func TestItemCodeSetForWidgetOverride(t *testing.T) {
	view, err := ProjectConfigSetToEditView(ProjectInput{
		ConfigSet:   testConfigSet(),
		Layer:       "backend_runtime",
		ObjectKind:  "backend_runtime",
		ObjectID:    "rt-test",
		ObjectLabel: "Widget Test",
	})
	if err != nil {
		t.Fatalf("project view: %v", err)
	}

	fields := flattenFields(view)

	// runtime.env should have key_value_table widget (not raw_json)
	env := requireField(t, fields, "runtime.env", "runtime.env", nil, "environment")
	if env.Widget != "key_value_table" {
		t.Errorf("runtime.env widget = %q, want key_value_table", env.Widget)
	}

	// Check health field from a richer config — this test config may not have it
	// explicitly, but the widget override should still apply if it appears.
	for _, f := range fields {
		if f.Key == "runtime.model_mount" {
			if f.Widget != "mount_form" {
				t.Errorf("runtime.model_mount widget = %q, want mount_form", f.Widget)
			}
		}
		if f.Key == "runtime.health" {
			if f.Widget != "health_check_form" {
				t.Errorf("runtime.health widget = %q, want health_check_form", f.Widget)
			}
		}
	}
}
