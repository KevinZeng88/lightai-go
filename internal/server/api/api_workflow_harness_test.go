package api

import (
	"net/http"
	"testing"
)

func TestWorkflowHarnessLoginAndListNodes(t *testing.T) {
	app := newWorkflowTestApp(t)

	app.Client.LoginAsAdmin(t)
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes", nil, http.StatusOK)

	var nodes []map[string]interface{}
	resp.Decode(t, &nodes)
	if nodes == nil {
		t.Fatalf("GET /api/v1/nodes returned nil list")
	}
}

func TestWorkflowHarnessCSRFProtection(t *testing.T) {
	app := newWorkflowTestApp(t)
	nodeID := app.InsertOnlineNode(t, "workflow-csrf-node", nil)
	app.Client.LoginAsAdmin(t)

	body := map[string]interface{}{"path": t.TempDir()}
	app.Client.JSONWithoutCSRF(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/model-roots", body, http.StatusForbidden)

	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/model-roots", body, http.StatusCreated)
	var root map[string]interface{}
	resp.Decode(t, &root)
	if root["id"] == "" || root["id"] == nil {
		t.Fatalf("model root create response missing id: %#v", root)
	}
}

func TestFakeAgentDockerImagesThroughRealRouter(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{
			{
				Repository: "vllm/vllm-openai",
				Tag:        "latest",
				ImageRef:   "vllm/vllm-openai:latest",
				ImageID:    "sha256:workflow-fake-image",
			},
		},
	})
	nodeID := app.InsertOnlineNode(t, "workflow-fake-agent-node", fakeAgent)
	app.Client.LoginAsAdmin(t)

	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes/"+nodeID+"/docker-images?query=vllm&limit=5", nil, http.StatusOK)

	var payload map[string]interface{}
	resp.Decode(t, &payload)
	images, ok := payload["images"].([]interface{})
	if !ok || len(images) != 1 {
		t.Fatalf("expected one fake image, got %#v", payload["images"])
	}
	first, ok := images[0].(map[string]interface{})
	if !ok {
		t.Fatalf("image payload has unexpected shape: %#v", images[0])
	}
	if first["image_ref"] != "vllm/vllm-openai:latest" {
		t.Fatalf("image_ref mismatch: %#v", first)
	}
	if fakeAgent.RequestCount("/docker-images") != 1 {
		t.Fatalf("fake agent docker-images request count=%d, want 1", fakeAgent.RequestCount("/docker-images"))
	}
}
