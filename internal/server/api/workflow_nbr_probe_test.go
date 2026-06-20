package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestWorkflowNBRProbeChain(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{
			{
				Repository: "vllm/vllm-openai",
				Tag:        "latest",
				ImageRef:   "vllm/vllm-openai:latest",
				ImageID:    "sha256:workflow-vllm-image",
				Size:       123456789,
			},
		},
		Inspect: workflowInspectSuccess("vllm/vllm-openai:latest", "sha256:workflow-vllm-image"),
	})
	nodeID := app.InsertOnlineNode(t, "workflow-nbr-node", fakeAgent)
	app.InsertGPU(t, nodeID, "nvidia")

	app.Client.LoginAsAdmin(t)

	nodesResp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes", nil, http.StatusOK)
	var nodes []map[string]interface{}
	nodesResp.Decode(t, &nodes)
	if !workflowListContainsID(nodes, nodeID) {
		t.Fatalf("node %q missing from GET /api/v1/nodes: %#v", nodeID, nodes)
	}

	imagesResp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes/"+nodeID+"/docker-images?query=vllm&limit=5", nil, http.StatusOK)
	var imagesPayload map[string]interface{}
	imagesResp.Decode(t, &imagesPayload)
	if !workflowImagesContainRef(imagesPayload, "vllm/vllm-openai:latest") {
		t.Fatalf("docker image list missing target image: %#v", imagesPayload)
	}

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "nvidia")
	enableResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          "vllm/vllm-openai:latest",
	}, http.StatusOK)
	var enabled map[string]interface{}
	enableResp.Decode(t, &enabled)
	nbrID := workflowStringField(t, enabled, "id")
	if enabled["status"] != "needs_check" {
		t.Fatalf("enable status=%#v want needs_check", enabled["status"])
	}

	probeResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/probe", map[string]interface{}{}, http.StatusOK)
	var probe map[string]interface{}
	probeResp.Decode(t, &probe)
	status := workflowStringField(t, probe, "status")
	if status == "missing_image" {
		t.Fatalf("inspect success must not produce missing_image: %#v", probe)
	}
	if status != "ready" && status != "ready_with_warnings" {
		t.Fatalf("probe status=%q want ready or ready_with_warnings body=%#v", status, probe)
	}

	getProbeResp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/probe", nil, http.StatusOK)
	var storedProbe map[string]interface{}
	getProbeResp.Decode(t, &storedProbe)
	probeResults := workflowMapField(t, storedProbe, "probe_results_json")
	level2 := workflowMapField(t, probeResults, "level2")
	if level2["image_id"] != "sha256:workflow-vllm-image" {
		t.Fatalf("stored probe level2 image_id=%#v", level2["image_id"])
	}

	listResp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes/"+nodeID+"/backend-runtimes", nil, http.StatusOK)
	var nbrs []map[string]interface{}
	listResp.Decode(t, &nbrs)
	listItem := workflowFindByID(t, nbrs, nbrID)
	listProbeResults := workflowMapField(t, listItem, "probe_results_json")
	listLevel2 := workflowMapField(t, listProbeResults, "level2")
	if listLevel2["image_id"] != "sha256:workflow-vllm-image" {
		t.Fatalf("list probe level2 image_id=%#v", listLevel2["image_id"])
	}

	compatResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/check-request", map[string]interface{}{}, http.StatusOK)
	var compat map[string]interface{}
	compatResp.Decode(t, &compat)
	if workflowStringField(t, compat, "status") == "missing_image" {
		t.Fatalf("check-request compat returned missing_image after inspect success: %#v", compat)
	}

	deleteResp := app.Client.JSON(t, http.MethodDelete, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID, nil, http.StatusOK)
	var deleted map[string]interface{}
	deleteResp.Decode(t, &deleted)
	if deleted["status"] != "deleted" {
		t.Fatalf("delete response=%#v", deleted)
	}
	afterDeleteResp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes/"+nodeID+"/backend-runtimes", nil, http.StatusOK)
	var afterDelete []map[string]interface{}
	afterDeleteResp.Decode(t, &afterDelete)
	if workflowListContainsID(afterDelete, nbrID) {
		t.Fatalf("NBR %q still visible after delete: %#v", nbrID, afterDelete)
	}
}

func TestWorkflowNBRProbeMissingImageOnlyFromInspectNotFound(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images:  []fakeAgentImage{},
		Inspect: map[string]interface{}{"error": "no such image: not-exist/lightai-test:missing"},
	})
	nodeID := app.InsertOnlineNode(t, "workflow-nbr-missing-node", fakeAgent)
	app.InsertGPU(t, nodeID, "nvidia")
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "nvidia")
	nbrID := app.EnableNodeBackendRuntime(t, nodeID, runtimeID, "not-exist/lightai-test:missing")

	probeResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/probe", map[string]interface{}{}, http.StatusOK)
	var probe map[string]interface{}
	probeResp.Decode(t, &probe)
	if got := workflowStringField(t, probe, "status"); got != "missing_image" {
		t.Fatalf("status=%q want missing_image body=%#v", got, probe)
	}
}

func TestWorkflowNBRProbeInspectErrorIsNotMissingImage(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{
			{
				Repository: "vllm/vllm-openai",
				Tag:        "latest",
				ImageRef:   "vllm/vllm-openai:latest",
				ImageID:    "sha256:workflow-vllm-image",
			},
		},
		Inspect: map[string]interface{}{"error": "docker daemon timeout"},
	})
	nodeID := app.InsertOnlineNode(t, "workflow-nbr-inspect-error-node", fakeAgent)
	app.InsertGPU(t, nodeID, "nvidia")
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "nvidia")
	nbrID := app.EnableNodeBackendRuntime(t, nodeID, runtimeID, "vllm/vllm-openai:latest")

	probeResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/probe", map[string]interface{}{}, http.StatusOK)
	var probe map[string]interface{}
	probeResp.Decode(t, &probe)
	if got := workflowStringField(t, probe, "status"); got == "missing_image" {
		t.Fatalf("inspect error must not map to missing_image: %#v", probe)
	}
	if got := workflowStringField(t, probe, "status"); got != "inspect_failed" {
		t.Fatalf("status=%q want inspect_failed body=%#v", got, probe)
	}
}

func workflowInspectSuccess(imageRef, imageID string) map[string]interface{} {
	return map[string]interface{}{
		"inspect": map[string]interface{}{
			"Id":       imageID,
			"RepoTags": []string{imageRef},
			"Config": map[string]interface{}{
				"Entrypoint": []string{"python3", "-m", "vllm.entrypoints.openai.api_server"},
				"Cmd":        []string{},
				"Env":        []string{"PATH=/usr/local/bin"},
			},
			"Size": float64(123456789),
		},
	}
}

func workflowImagesContainRef(payload map[string]interface{}, imageRef string) bool {
	images, _ := payload["images"].([]interface{})
	for _, raw := range images {
		image, _ := raw.(map[string]interface{})
		if image["image_ref"] == imageRef {
			return true
		}
	}
	return false
}

func workflowListContainsID(items []map[string]interface{}, id string) bool {
	for _, item := range items {
		if item["id"] == id {
			return true
		}
	}
	return false
}

func workflowFindByID(t *testing.T, items []map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	for _, item := range items {
		if item["id"] == id {
			return item
		}
	}
	t.Fatalf("item %q not found in %#v", id, items)
	return nil
}

func workflowStringField(t *testing.T, payload map[string]interface{}, field string) string {
	t.Helper()
	value, _ := payload[field].(string)
	if strings.TrimSpace(value) == "" {
		t.Fatalf("field %q missing or empty in %#v", field, payload)
	}
	return value
}

func workflowMapField(t *testing.T, payload map[string]interface{}, field string) map[string]interface{} {
	t.Helper()
	value, _ := payload[field].(map[string]interface{})
	if value == nil {
		t.Fatalf("field %q missing or not object in %#v", field, payload)
	}
	return value
}
