package api

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestWorkflowDeploymentPreflightRunPlan(t *testing.T) {
	app := newWorkflowTestApp(t)
	fixture := newWorkflowDeploymentFixture(t, app, "runplan")

	preflight := workflowDeploymentPreflight(t, app, fixture)
	if preflight["can_run"] != true {
		t.Fatalf("preflight can_run=%#v response=%#v", preflight["can_run"], preflight)
	}
	workflowAssertPreflightCandidate(t, preflight, fixture.NodeID)

	deployment := workflowCreateDeployment(t, app, fixture, "runplan")
	deploymentID := workflowStringField(t, deployment, "id")
	workflowAssertDeploymentListDetailConsistent(t, app, deployment)
	workflowAssertDeploymentSnapshot(t, deployment, fixture)

	dryRun := workflowDeploymentDryRun(t, app, deploymentID)
	workflowAssertDryRunRunPlanFields(t, dryRun, fixture, "workflow/frozen-runplan:latest")

	workflowDeleteDeployment(t, app, deploymentID)
	workflowAssertDeploymentDeleted(t, app, deploymentID)
}

func TestWorkflowDeploymentRunPlanPreservesNBRSnapshot(t *testing.T) {
	app := newWorkflowTestApp(t)
	fixture := newWorkflowDeploymentFixture(t, app, "freeze")

	deployment := workflowCreateDeployment(t, app, fixture, "freeze")
	deploymentID := workflowStringField(t, deployment, "id")
	beforeDryRun := workflowDeploymentDryRun(t, app, deploymentID)
	workflowAssertDryRunRunPlanFields(t, beforeDryRun, fixture, "workflow/frozen-freeze:latest")

	app.Client.JSON(t, http.MethodPatch, "/api/v1/nodes/"+fixture.NodeID+"/backend-runtimes/"+fixture.NBRID, map[string]interface{}{
		"image_ref": "workflow/live-mutation:latest",
	}, http.StatusOK)
	app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+fixture.NodeID+"/backend-runtimes/check", map[string]interface{}{
		"backend_runtime_id": fixture.RuntimeID,
		"image_ref":          "workflow/live-check-mutation:latest",
		"image_present":      true,
		"docker_available":   true,
	}, http.StatusOK)

	afterDryRun := workflowDeploymentDryRun(t, app, deploymentID)
	workflowAssertDryRunRunPlanFields(t, afterDryRun, fixture, "workflow/frozen-freeze:latest")
	if afterDryRun["resolved_image"] != beforeDryRun["resolved_image"] {
		t.Fatalf("dry-run image changed after live NBR mutation: before=%#v after=%#v", beforeDryRun, afterDryRun)
	}

	afterDetail := workflowGetDeployment(t, app, deploymentID, http.StatusOK)
	workflowAssertDeploymentSnapshot(t, afterDetail, fixture)

	workflowDeleteDeployment(t, app, deploymentID)
	workflowAssertDeploymentDeleted(t, app, deploymentID)
}

func TestWorkflowDeploymentCleanup(t *testing.T) {
	app := newWorkflowTestApp(t)
	fixture := newWorkflowDeploymentFixture(t, app, "cleanup")

	deployment := workflowCreateDeployment(t, app, fixture, "cleanup")
	deploymentID := workflowStringField(t, deployment, "id")
	workflowGetDeployment(t, app, deploymentID, http.StatusOK)
	workflowAssertDeploymentListDetailConsistent(t, app, deployment)

	workflowDeleteDeployment(t, app, deploymentID)
	workflowAssertDeploymentDeleted(t, app, deploymentID)
}

type workflowDeploymentFixture struct {
	NodeID      string
	GPUID       string
	RuntimeID   string
	NBRID       string
	ArtifactID  string
	LocationID  string
	ModelPath   string
	ModelRoot   string
	Service     map[string]interface{}
	Parameters  map[string]interface{}
	EnvOverride map[string]interface{}
}

func newWorkflowDeploymentFixture(t *testing.T, app *workflowTestApp, suffix string) workflowDeploymentFixture {
	t.Helper()

	scan := workflowModelScanPayload("deployment-"+suffix, "/models/deployment-"+suffix+"/Qwen3-0.6B-Instruct")
	modelFixture := newWorkflowModelWizardFixture(t, app, "deployment-"+suffix, scan)
	app.Client.LoginAsAdmin(t)
	modelFixture.ensureRoot(t, app)

	nodeID := modelFixture.NodeID
	gpuID := app.InsertGPU(t, nodeID, "nvidia")

	systemRuntime := workflowFindSystemRuntimeByVendor(t, app, "nvidia")
	runtime, markRuntimeDeleted := workflowCloneBackendRuntime(t, app, systemRuntime, "deployment-"+suffix)
	runtimeID := workflowStringField(t, runtime, "id")
	patch := workflowDeploymentRuntimePatchPayload(suffix)
	patchResp := app.Client.JSON(t, http.MethodPatch, "/api/v1/backend-runtimes/"+runtimeID, patch, http.StatusOK)
	var patchedRuntime map[string]interface{}
	patchResp.Decode(t, &patchedRuntime)
	workflowAssertBackendRuntimePatchApplied(t, patchedRuntime, patch)

	scanned := workflowScanModelPath(t, app, nodeID, modelFixture.RootID, "Qwen3-0.6B-Instruct")
	artifact, markArtifactDeleted := workflowCreateModelArtifactFromScan(t, app, "deployment-"+suffix, scanned)
	artifactID := workflowStringField(t, artifact, "id")
	location := workflowCreateModelLocationFromScan(t, app, artifactID, nodeID, scanned, "deployment-"+suffix)
	locationID := workflowStringField(t, location, "id")

	nbrImage := "workflow/frozen-" + suffix + ":latest"
	checkResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/check", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"display_name":       "Workflow deployment NBR " + suffix,
		"image_ref":          nbrImage,
		"image_present":      true,
		"docker_available":   true,
		"driver_version":     "555.55",
		"toolkit_version":    "12.5",
	}, http.StatusOK)
	var nbr map[string]interface{}
	checkResp.Decode(t, &nbr)
	if nbr["status"] != "ready" {
		t.Fatalf("NBR status=%#v want ready response=%#v", nbr["status"], nbr)
	}
	nbrID := workflowStringField(t, nbr, "id")

	t.Cleanup(func() {
		_, _ = app.DB.Exec(`DELETE FROM model_deployments WHERE source_node_backend_runtime_id = ?`, nbrID)
		workflowDeleteNodeBackendRuntimeIfPresent(t, app, nodeID, nbrID)
		workflowDeleteModelArtifact(t, app, artifactID)
		markArtifactDeleted()
		workflowDeleteBackendRuntime(t, app, runtimeID)
		markRuntimeDeleted()
	})

	service := map[string]interface{}{
		"host_port":      float64(18080),
		"container_port": float64(8000),
		"app_port":       float64(8000),
		"health_port":    float64(18080),
	}
	parameters := map[string]interface{}{
		"served_model_name": "workflow-" + suffix,
		"max_model_len":     float64(2048),
	}
	envOverrides := map[string]interface{}{
		"LIGHTAI_DEPLOYMENT_WORKFLOW": suffix,
	}

	return workflowDeploymentFixture{
		NodeID:      nodeID,
		GPUID:       gpuID,
		RuntimeID:   runtimeID,
		NBRID:       nbrID,
		ArtifactID:  artifactID,
		LocationID:  locationID,
		ModelPath:   workflowStringField(t, scanned, "absolute_path"),
		ModelRoot:   workflowStringField(t, scanned, "model_root"),
		Service:     service,
		Parameters:  parameters,
		EnvOverride: envOverrides,
	}
}

func workflowDeploymentRuntimePatchPayload(suffix string) map[string]interface{} {
	return map[string]interface{}{
		"name":         "workflow-deployment-runtime-" + suffix,
		"display_name": "Workflow Deployment Runtime " + suffix,
		"image_name":   "workflow/runtime-template-" + suffix + ":latest",
		"default_env_json": map[string]interface{}{
			"LIGHTAI_RUNTIME_WORKFLOW": suffix,
			"VLLM_LOGGING_LEVEL":       "INFO",
		},
		"docker_json": map[string]interface{}{
			"devices": []interface{}{
				map[string]interface{}{"host_path": "/dev/nvidia0", "container_path": "/dev/nvidia0", "permissions": "rwm"},
			},
			"privileged":       true,
			"ipc_mode":         "host",
			"shm_size":         "16g",
			"security_options": []interface{}{"label=disable"},
			"ulimits": map[string]interface{}{
				"memlock": "-1",
			},
		},
		"args_override_json": []interface{}{"--host", "0.0.0.0", "--port", "8000"},
		"entrypoint_override_json": []interface{}{
			"python3", "-m", "vllm.entrypoints.openai.api_server",
		},
		"model_mount_json": map[string]interface{}{
			"container_path": "/models",
			"readonly":       true,
		},
		"health_check_override_json": map[string]interface{}{
			"path":                    "/v1/models",
			"expected_status":         float64(200),
			"startup_timeout_seconds": float64(120),
			"interval_seconds":        float64(5),
			"timeout_seconds":         float64(2),
		},
	}
}

func workflowFindSystemRuntimeByVendor(t *testing.T, app *workflowTestApp, vendor string) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes", nil, http.StatusOK)
	var runtimes []map[string]interface{}
	resp.Decode(t, &runtimes)
	for _, runtime := range runtimes {
		if workflowBoolField(t, runtime, "is_builtin") && !workflowBoolField(t, runtime, "is_editable") && runtime["vendor"] == vendor {
			return runtime
		}
	}
	t.Fatalf("system runtime for vendor %q not found: %#v", vendor, runtimes)
	return nil
}

func workflowDeploymentPreflight(t *testing.T, app *workflowTestApp, fixture workflowDeploymentFixture) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/deployments/preflight", map[string]interface{}{
		"model_artifact_id":       fixture.ArtifactID,
		"node_backend_runtime_id": fixture.NBRID,
		"node_id":                 fixture.NodeID,
		"accelerator_ids":         []interface{}{fixture.GPUID},
		"host_port":               float64(18080),
	}, http.StatusOK)
	var preflight map[string]interface{}
	resp.Decode(t, &preflight)
	return preflight
}

func workflowCreateDeployment(t *testing.T, app *workflowTestApp, fixture workflowDeploymentFixture, suffix string) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/deployments", map[string]interface{}{
		"name":                    "workflow-deployment-" + suffix,
		"display_name":            "Workflow Deployment " + suffix,
		"description":             "API workflow deployment fixture",
		"model_artifact_id":       fixture.ArtifactID,
		"node_backend_runtime_id": fixture.NBRID,
		"replicas":                float64(1),
		"placement_json": map[string]interface{}{
			"node_id":         fixture.NodeID,
			"accelerator_ids": []interface{}{fixture.GPUID},
		},
		"service_json":       fixture.Service,
		"parameters_json":    fixture.Parameters,
		"env_overrides_json": fixture.EnvOverride,
	}, http.StatusCreated)
	var deployment map[string]interface{}
	resp.Decode(t, &deployment)
	deploymentID := workflowStringField(t, deployment, "id")
	t.Cleanup(func() {
		_, _ = app.DB.Exec(`DELETE FROM model_deployments WHERE id = ?`, deploymentID)
	})
	return deployment
}

func workflowGetDeployment(t *testing.T, app *workflowTestApp, id string, wantStatus int) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/deployments/"+id, nil, wantStatus)
	if wantStatus != http.StatusOK {
		return nil
	}
	var deployment map[string]interface{}
	resp.Decode(t, &deployment)
	return deployment
}

func workflowDeploymentDryRun(t *testing.T, app *workflowTestApp, deploymentID string) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/deployments/"+deploymentID+"/dry-run", map[string]interface{}{}, http.StatusOK)
	var dryRun map[string]interface{}
	resp.Decode(t, &dryRun)
	if dryRun["valid"] != true {
		t.Fatalf("dry-run valid=%#v response=%#v", dryRun["valid"], dryRun)
	}
	return dryRun
}

func workflowDeleteDeployment(t *testing.T, app *workflowTestApp, id string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodDelete, "/api/v1/deployments/"+id, nil, http.StatusOK)
	var payload map[string]interface{}
	resp.Decode(t, &payload)
	if payload["status"] != "deleted" {
		t.Fatalf("deployment delete response=%#v", payload)
	}
}

func workflowAssertPreflightCandidate(t *testing.T, preflight map[string]interface{}, nodeID string) {
	t.Helper()
	candidates, ok := preflight["candidate_nodes"].([]interface{})
	if !ok || len(candidates) == 0 {
		t.Fatalf("preflight candidate_nodes missing: %#v", preflight)
	}
	first, ok := candidates[0].(map[string]interface{})
	if !ok || first["node_id"] != nodeID || first["status"] != "ready" {
		t.Fatalf("preflight candidate mismatch: %#v", preflight)
	}
	errors, ok := preflight["errors"].([]interface{})
	if !ok || len(errors) != 0 {
		t.Fatalf("preflight errors=%#v response=%#v", preflight["errors"], preflight)
	}
}

func workflowAssertDryRunRunPlanFields(t *testing.T, dryRun map[string]interface{}, fixture workflowDeploymentFixture, wantImage string) {
	t.Helper()
	if dryRun["resolved_image"] != wantImage {
		t.Fatalf("resolved_image=%#v want %#v dryRun=%#v", dryRun["resolved_image"], wantImage, dryRun)
	}
	if dryRun["selected_node"] != fixture.NodeID {
		t.Fatalf("selected_node=%#v want %#v dryRun=%#v", dryRun["selected_node"], fixture.NodeID, dryRun)
	}
	if dryRun["selected_runtime"] != fixture.RuntimeID {
		t.Fatalf("selected_runtime=%#v want %#v dryRun=%#v", dryRun["selected_runtime"], fixture.RuntimeID, dryRun)
	}
	if dryRun["selected_model_location"] != fixture.LocationID {
		t.Fatalf("selected_model_location=%#v want %#v dryRun=%#v", dryRun["selected_model_location"], fixture.LocationID, dryRun)
	}
	preview := workflowStringField(t, dryRun, "command_preview")
	for _, want := range []string{
		"docker run",
		wantImage,
		fixture.ModelPath,
		"--name",
		"--device /dev/nvidia0:/dev/nvidia0:rwm",
		"--ipc host",
		"--shm-size 16g",
		"--security-opt label=disable",
		"--ulimit memlock=-1",
		"-e LIGHTAI_RUNTIME_WORKFLOW=",
		"-e LIGHTAI_DEPLOYMENT_WORKFLOW=",
		"-v " + fixture.ModelPath + ":/models/Qwen3-0.6B-Instruct:ro",
		"-p 18080:8000/tcp",
		"--host 0.0.0.0",
		"--port 8000",
	} {
		if !strings.Contains(preview, want) {
			t.Fatalf("command_preview missing %q:\n%s", want, preview)
		}
	}
}

func workflowAssertDeploymentSnapshot(t *testing.T, deployment map[string]interface{}, fixture workflowDeploymentFixture) {
	t.Helper()
	snapshot := workflowMapField(t, deployment, "config_snapshot_json")
	if snapshot["nbr_image_ref"] == nil || snapshot["nbr_image_ref"] != "workflow/frozen-"+strings.TrimPrefix(workflowStringField(t, deployment, "name"), "workflow-deployment-")+":latest" {
		t.Fatalf("deployment snapshot missing frozen NBR image_ref: %#v", snapshot)
	}
	if deployment["source_node_backend_runtime_id"] != fixture.NBRID {
		t.Fatalf("deployment source_node_backend_runtime_id=%#v want %#v deployment=%#v", deployment["source_node_backend_runtime_id"], fixture.NBRID, deployment)
	}
	docker := workflowMapField(t, snapshot, "docker_json")
	if _, ok := docker["devices"].([]interface{}); !ok {
		t.Fatalf("deployment snapshot docker devices missing: %#v", docker)
	}
	if docker["ipc_mode"] != "host" || docker["shm_size"] != "16g" {
		t.Fatalf("deployment snapshot docker fields missing: %#v", docker)
	}
	env := workflowMapField(t, snapshot, "default_env_json")
	if env["LIGHTAI_RUNTIME_WORKFLOW"] == nil {
		t.Fatalf("deployment snapshot env missing workflow key: %#v", env)
	}
	workflowMapField(t, snapshot, "health_check_override_json")
	workflowMapField(t, snapshot, "model_mount_json")
	if _, ok := snapshot["args_override_json"].([]interface{}); !ok {
		t.Fatalf("deployment snapshot args_override_json missing or wrong type: %#v", snapshot)
	}
}

func workflowAssertDeploymentListDetailConsistent(t *testing.T, app *workflowTestApp, deployment map[string]interface{}) {
	t.Helper()
	detail := workflowGetDeployment(t, app, workflowStringField(t, deployment, "id"), http.StatusOK)
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/deployments", nil, http.StatusOK)
	var deployments []map[string]interface{}
	resp.Decode(t, &deployments)
	listItem := workflowFindByID(t, deployments, workflowStringField(t, deployment, "id"))
	for _, field := range []string{
		"id",
		"name",
		"display_name",
		"description",
		"model_artifact_id",
		"backend_runtime_id",
		"replicas",
		"placement_json",
		"service_json",
		"parameters_json",
		"env_overrides_json",
		"config_snapshot_json",
		"source_node_backend_runtime_id",
		"desired_state",
		"status",
		"tenant_id",
	} {
		if !reflect.DeepEqual(listItem[field], detail[field]) {
			t.Fatalf("deployment list/detail mismatch field %q: list=%#v detail=%#v", field, listItem[field], detail[field])
		}
	}
}

func workflowAssertDeploymentDeleted(t *testing.T, app *workflowTestApp, id string) {
	t.Helper()
	workflowGetDeployment(t, app, id, http.StatusNotFound)
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/deployments", nil, http.StatusOK)
	var deployments []map[string]interface{}
	resp.Decode(t, &deployments)
	if workflowListContainsID(deployments, id) {
		t.Fatalf("deployment %q still visible after delete: %#v", id, deployments)
	}
}

func workflowDeleteNodeBackendRuntimeIfPresent(t *testing.T, app *workflowTestApp, nodeID, nbrID string) {
	t.Helper()
	if nodeID == "" || nbrID == "" {
		return
	}
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes/"+nodeID+"/backend-runtimes", nil, http.StatusOK)
	var nbrs []map[string]interface{}
	resp.Decode(t, &nbrs)
	if !workflowListContainsID(nbrs, nbrID) {
		return
	}
	app.Client.JSON(t, http.MethodDelete, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID, nil, http.StatusOK)
}
