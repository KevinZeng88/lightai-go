package api

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestWorkflowBackendRuntimeCRUDChain(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	fixture := workflowBackendRuntimeFixture(t, app)
	systemBefore := workflowGetBackendRuntime(t, app, workflowStringField(t, fixture.SystemRuntime, "id"), http.StatusOK)

	patchSystemResp := app.Client.JSON(t, http.MethodPatch, "/api/v1/backend-runtimes/"+workflowStringField(t, fixture.SystemRuntime, "id"), map[string]interface{}{
		"display_name": "must-not-mutate-system-runtime",
	}, http.StatusConflict)
	if !strings.Contains(string(patchSystemResp.Body), "read-only") {
		t.Fatalf("system runtime PATCH should be rejected as read-only: %s", string(patchSystemResp.Body))
	}

	userRuntime, markDeleted := workflowCloneBackendRuntime(t, app, fixture.SystemRuntime, "crud")

	userID := workflowStringField(t, userRuntime, "id")
	if !workflowBoolField(t, userRuntime, "is_editable") {
		t.Fatalf("cloned runtime is not editable: %#v", userRuntime)
	}
	if workflowBoolField(t, userRuntime, "is_builtin") {
		t.Fatalf("cloned runtime must not be builtin: %#v", userRuntime)
	}

	detail := workflowGetBackendRuntime(t, app, userID, http.StatusOK)
	if workflowStringField(t, detail, "id") != userID {
		t.Fatalf("detail id mismatch: %#v", detail)
	}
	workflowAssertSameScalarFields(t, userRuntime, detail, "backend_id", "backend_version_id", "vendor", "runtime_type")

	patch := workflowBackendRuntimePatchPayload("crud")
	patchResp := app.Client.JSON(t, http.MethodPatch, "/api/v1/backend-runtimes/"+userID, patch, http.StatusOK)
	var patched map[string]interface{}
	patchResp.Decode(t, &patched)
	workflowAssertBackendRuntimePatchApplied(t, patched, patch)
	workflowAssertBackendRuntimeJSONTypes(t, patched)
	workflowAssertSameScalarFields(t, detail, patched, "backend_id", "backend_version_id", "vendor", "runtime_type")

	afterPatchDetail := workflowGetBackendRuntime(t, app, userID, http.StatusOK)
	workflowAssertBackendRuntimePatchApplied(t, afterPatchDetail, patch)
	workflowAssertBackendRuntimeJSONTypes(t, afterPatchDetail)
	workflowAssertBackendRuntimeListDetailConsistent(t, app, afterPatchDetail)

	systemAfter := workflowGetBackendRuntime(t, app, workflowStringField(t, fixture.SystemRuntime, "id"), http.StatusOK)
	workflowAssertSystemRuntimeUnchanged(t, systemBefore, systemAfter)

	workflowDeleteBackendRuntime(t, app, userID)
	markDeleted()
	workflowGetBackendRuntime(t, app, userID, http.StatusNotFound)
	workflowAssertBackendRuntimeNotListed(t, app, userID)
}

func TestWorkflowBackendRuntimePatchPreservesFields(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	fixture := workflowBackendRuntimeFixture(t, app)
	userRuntime, markDeleted := workflowCloneBackendRuntime(t, app, fixture.SystemRuntime, "preserve")

	userID := workflowStringField(t, userRuntime, "id")
	initialPatch := workflowBackendRuntimePatchPayload("preserve")
	initialResp := app.Client.JSON(t, http.MethodPatch, "/api/v1/backend-runtimes/"+userID, initialPatch, http.StatusOK)
	var initial map[string]interface{}
	initialResp.Decode(t, &initial)
	workflowAssertBackendRuntimePatchApplied(t, initial, initialPatch)

	preservedFields := workflowPickFields(initial,
		"backend_id",
		"backend_version_id",
		"vendor",
		"runtime_type",
		"env",
		"docker_options",
		"model_mount",
		"health_check",
		"command",
		"entrypoint",
		"config_set",
	)

	secondPatch := map[string]interface{}{
		"display_name": "Workflow Preserve Runtime Renamed",
		"image_ref":    "lightai/workflow-preserve-renamed:latest",
	}
	secondResp := app.Client.JSON(t, http.MethodPatch, "/api/v1/backend-runtimes/"+userID, secondPatch, http.StatusOK)
	var second map[string]interface{}
	secondResp.Decode(t, &second)
	if second["display_name"] != secondPatch["display_name"] || second["image_ref"] != secondPatch["image_ref"] {
		t.Fatalf("second patch not applied: patch=%#v response=%#v", secondPatch, second)
	}
	workflowAssertFieldsPreserved(t, preservedFields, second, "display_name", "image_ref", "config_set")
	workflowAssertBackendRuntimeJSONTypes(t, second)
	workflowAssertBackendRuntimeListDetailConsistent(t, app, second)

	workflowDeleteBackendRuntime(t, app, userID)
	markDeleted()
}

func TestWorkflowBackendRuntimeDeleteCleanup(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	fixture := workflowBackendRuntimeFixture(t, app)
	userRuntime, markDeleted := workflowCloneBackendRuntime(t, app, fixture.SystemRuntime, "delete")

	userID := workflowStringField(t, userRuntime, "id")
	workflowGetBackendRuntime(t, app, userID, http.StatusOK)
	workflowAssertBackendRuntimeListDetailConsistent(t, app, userRuntime)

	workflowDeleteBackendRuntime(t, app, userID)
	markDeleted()

	workflowGetBackendRuntime(t, app, userID, http.StatusNotFound)
	workflowAssertBackendRuntimeNotListed(t, app, userID)
}

type workflowBackendRuntimeSelection struct {
	Backend       map[string]interface{}
	Version       map[string]interface{}
	SystemRuntime map[string]interface{}
}

func workflowBackendRuntimeFixture(t *testing.T, app *workflowTestApp) workflowBackendRuntimeSelection {
	t.Helper()

	backendsResp := app.Client.JSON(t, http.MethodGet, "/api/v1/backends", nil, http.StatusOK)
	var backends []map[string]interface{}
	backendsResp.Decode(t, &backends)
	if len(backends) == 0 {
		t.Fatalf("GET /api/v1/backends returned no backends")
	}

	runtimesResp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes", nil, http.StatusOK)
	var runtimes []map[string]interface{}
	runtimesResp.Decode(t, &runtimes)
	if len(runtimes) == 0 {
		t.Fatalf("GET /api/v1/backend-runtimes returned no runtimes")
	}

	systemRuntime := workflowFindSystemBackendRuntime(t, runtimes)
	backendID := workflowStringField(t, systemRuntime, "backend_id")
	backend := workflowFindByID(t, backends, backendID)

	versionsResp := app.Client.JSON(t, http.MethodGet, "/api/v1/backends/"+backendID+"/versions", nil, http.StatusOK)
	var versions []map[string]interface{}
	versionsResp.Decode(t, &versions)
	if len(versions) == 0 {
		t.Fatalf("GET /api/v1/backends/%s/versions returned no versions", backendID)
	}

	allVersionsResp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-versions?backend_id="+backendID, nil, http.StatusOK)
	var allVersions []map[string]interface{}
	allVersionsResp.Decode(t, &allVersions)
	if len(allVersions) == 0 {
		t.Fatalf("GET /api/v1/backend-versions?backend_id=%s returned no versions", backendID)
	}

	versionID := workflowStringField(t, systemRuntime, "backend_version_id")
	version := workflowFindByID(t, versions, versionID)
	if !workflowListContainsID(allVersions, versionID) {
		t.Fatalf("backend version %q missing from all-version list: %#v", versionID, allVersions)
	}

	return workflowBackendRuntimeSelection{
		Backend:       backend,
		Version:       version,
		SystemRuntime: systemRuntime,
	}
}

func workflowFindSystemBackendRuntime(t *testing.T, runtimes []map[string]interface{}) map[string]interface{} {
	t.Helper()
	for _, runtime := range runtimes {
		if workflowBoolField(t, runtime, "is_builtin") && !workflowBoolField(t, runtime, "is_editable") && runtime["backend_id"] != "" && runtime["backend_version_id"] != "" {
			return runtime
		}
	}
	t.Fatalf("system backend runtime/template not found in %#v", runtimes)
	return nil
}

func workflowCloneBackendRuntime(t *testing.T, app *workflowTestApp, source map[string]interface{}, suffix string) (map[string]interface{}, func()) {
	t.Helper()

	sourceID := workflowStringField(t, source, "id")
	name := fmt.Sprintf("workflow-%s-runtime-%s", suffix, strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-")))
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/backend-runtimes/"+sourceID+"/clone", map[string]interface{}{
		"name":         name,
		"display_name": "Workflow " + suffix + " Runtime",
	}, http.StatusCreated)
	var cloned map[string]interface{}
	resp.Decode(t, &cloned)

	id := workflowStringField(t, cloned, "id")
	deleted := false
	markDeleted := func() {
		if deleted {
			return
		}
		deleted = true
	}
	t.Cleanup(func() {
		if deleted {
			return
		}
		app.Client.JSON(t, http.MethodDelete, "/api/v1/backend-runtimes/"+id, nil, http.StatusOK)
		deleted = true
	})
	return cloned, markDeleted
}

func workflowBackendRuntimePatchPayload(suffix string) map[string]interface{} {
	return map[string]interface{}{
		"name":         "workflow-" + suffix + "-runtime-user",
		"display_name": "Workflow " + suffix + " Runtime User",
		"image_ref":    "lightai/workflow-" + suffix + ":latest",
		"env": map[string]interface{}{
			"LIGHTAI_WORKFLOW_MODE": suffix,
			"VLLM_LOGGING_LEVEL":    "INFO",
		},
		"docker_options": map[string]interface{}{
			"ports": []interface{}{
				map[string]interface{}{"container_port": float64(8000), "host_port": float64(18080), "protocol": "tcp"},
			},
			"volumes": []interface{}{
				map[string]interface{}{"host_path": "/tmp/lightai-workflow-models", "container_path": "/models", "read_only": true},
			},
			"devices": []interface{}{"/dev/nvidia0"},
			"extra_args": []interface{}{
				"--ulimit", "memlock=-1",
			},
			"privileged":   true,
			"ipc_mode":     "host",
			"shm_size":     "16g",
			"security_opt": []interface{}{"label=disable"},
		},
		"command": []interface{}{"--host", "0.0.0.0", "--port", "8000"},
		"entrypoint": []interface{}{
			"python3", "-m", "vllm.entrypoints.openai.api_server",
		},
		"model_mount": map[string]interface{}{
			"container_path": "/models",
			"read_only":      true,
		},
		"health_check": map[string]interface{}{
			"path":             "/v1/models",
			"interval_seconds": float64(5),
			"timeout_seconds":  float64(2),
		},
	}
}

func workflowAssertBackendRuntimePatchApplied(t *testing.T, runtime map[string]interface{}, patch map[string]interface{}) {
	t.Helper()
	for _, field := range []string{"name", "display_name", "image_ref"} {
		if runtime[field] != patch[field] {
			t.Fatalf("%s=%#v want %#v in %#v", field, runtime[field], patch[field], runtime)
		}
	}
	for _, field := range []string{"env", "docker_options", "model_mount", "health_check", "command", "entrypoint"} {
		if !reflect.DeepEqual(runtime[field], patch[field]) {
			t.Fatalf("%s changed or not preserved:\n got=%#v\nwant=%#v\nruntime=%#v", field, runtime[field], patch[field], runtime)
		}
	}
}

func workflowAssertBackendRuntimeJSONTypes(t *testing.T, runtime map[string]interface{}) {
	t.Helper()
	for _, field := range []string{"env", "docker_options", "model_mount", "health_check", "config_set", "source_metadata"} {
		workflowMapField(t, runtime, field)
	}
	for _, field := range []string{"command", "entrypoint"} {
		value, ok := runtime[field].([]interface{})
		if !ok {
			t.Fatalf("field %q missing or not array in %#v", field, runtime)
		}
		if value == nil {
			t.Fatalf("field %q is nil array in %#v", field, runtime)
		}
	}
	dockerJSON := workflowMapField(t, runtime, "docker_options")
	for _, field := range []string{"ports", "volumes", "devices", "extra_args", "security_opt"} {
		if _, ok := dockerJSON[field].([]interface{}); !ok {
			t.Fatalf("docker_options.%s missing or not array in %#v", field, dockerJSON)
		}
	}
	if _, ok := dockerJSON["privileged"].(bool); !ok {
		t.Fatalf("docker_options.privileged missing or not bool in %#v", dockerJSON)
	}
	if _, ok := dockerJSON["ipc_mode"].(string); !ok {
		t.Fatalf("docker_options.ipc_mode missing or not string in %#v", dockerJSON)
	}
	if _, ok := dockerJSON["shm_size"].(string); !ok {
		t.Fatalf("docker_options.shm_size missing or not string in %#v", dockerJSON)
	}
}

func workflowAssertBackendRuntimeListDetailConsistent(t *testing.T, app *workflowTestApp, detail map[string]interface{}) {
	t.Helper()

	id := workflowStringField(t, detail, "id")
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes", nil, http.StatusOK)
	var runtimes []map[string]interface{}
	resp.Decode(t, &runtimes)
	listItem := workflowFindByID(t, runtimes, id)
	for _, field := range []string{
		"id",
		"name",
		"display_name",
		"backend_id",
		"backend_version_id",
		"vendor",
		"runtime_type",
		"image_ref",
		"env",
		"docker_options",
		"model_mount",
		"health_check",
		"command",
		"entrypoint",
		"is_builtin",
		"is_editable",
		"tenant_id",
	} {
		if !reflect.DeepEqual(listItem[field], detail[field]) {
			t.Fatalf("list/detail mismatch field %q:\n list=%#v\n detail=%#v", field, listItem[field], detail[field])
		}
	}
}

func workflowAssertSystemRuntimeUnchanged(t *testing.T, before, after map[string]interface{}) {
	t.Helper()
	for _, field := range []string{
		"id",
		"name",
		"display_name",
		"backend_id",
		"backend_version_id",
		"vendor",
		"runtime_type",
		"image_ref",
		"env",
		"docker_options",
		"model_mount",
		"health_check",
		"command",
		"entrypoint",
		"is_builtin",
		"is_editable",
		"tenant_id",
	} {
		if !reflect.DeepEqual(before[field], after[field]) {
			t.Fatalf("system runtime mutated field %q:\n before=%#v\n after=%#v", field, before[field], after[field])
		}
	}
}

func workflowAssertSameScalarFields(t *testing.T, before, after map[string]interface{}, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if before[field] != after[field] {
			t.Fatalf("field %q changed: before=%#v after=%#v", field, before[field], after[field])
		}
	}
}

func workflowPickFields(runtime map[string]interface{}, fields ...string) map[string]interface{} {
	out := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		out[field] = runtime[field]
	}
	return out
}

func workflowAssertFieldsPreserved(t *testing.T, want, got map[string]interface{}, changedFields ...string) {
	t.Helper()
	changed := make(map[string]bool, len(changedFields))
	for _, field := range changedFields {
		changed[field] = true
	}
	for field, wantValue := range want {
		if changed[field] {
			continue
		}
		if !reflect.DeepEqual(got[field], wantValue) {
			t.Fatalf("field %q was not preserved:\n got=%#v\nwant=%#v\ngot runtime=%#v", field, got[field], wantValue, got)
		}
	}
}

func workflowGetBackendRuntime(t *testing.T, app *workflowTestApp, id string, wantStatus int) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes/"+id, nil, wantStatus)
	if wantStatus != http.StatusOK {
		return nil
	}
	var runtime map[string]interface{}
	resp.Decode(t, &runtime)
	return runtime
}

func workflowDeleteBackendRuntime(t *testing.T, app *workflowTestApp, id string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodDelete, "/api/v1/backend-runtimes/"+id, nil, http.StatusOK)
	var deleted map[string]interface{}
	resp.Decode(t, &deleted)
	if deleted["status"] != "deleted" {
		t.Fatalf("delete response=%#v", deleted)
	}
}

func workflowAssertBackendRuntimeNotListed(t *testing.T, app *workflowTestApp, id string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes", nil, http.StatusOK)
	var runtimes []map[string]interface{}
	resp.Decode(t, &runtimes)
	if workflowListContainsID(runtimes, id) {
		t.Fatalf("backend runtime %q still visible after delete: %#v", id, runtimes)
	}
}

func workflowBoolField(t *testing.T, payload map[string]interface{}, field string) bool {
	t.Helper()
	value, ok := payload[field].(bool)
	if !ok {
		t.Fatalf("field %q missing or not bool in %#v", field, payload)
	}
	return value
}
