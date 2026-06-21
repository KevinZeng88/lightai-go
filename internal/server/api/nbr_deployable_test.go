package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestIsNBRDeployable(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"ready", true},
		{"ready_with_warnings", true},
		{"missing_image", false},
		{"needs_check", false},
		{"runtime_image_mismatch", false},
		{"inspect_failed", false},
		{"agent_unreachable", false},
		{"docker_error", false},
		{"unsupported_device", false},
		{"disabled", false},
		{"evidence_missing", false},
		{"node_offline", false},
		{"failed", false},
		{"unknown", false},
		{"error", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			got := isNBRDeployable(tc.status)
			if got != tc.expected {
				t.Errorf("isNBRDeployable(%q) = %v, want %v", tc.status, got, tc.expected)
			}
		})
	}
}

func TestNBRDisabledReason(t *testing.T) {
	// Deployable statuses should return empty disabled_reason
	for _, s := range []string{"ready", "ready_with_warnings"} {
		if r := nbrDisabledReason(s, ""); r != "" {
			t.Errorf("nbrDisabledReason(%q) = %q, want empty for deployable status", s, r)
		}
	}
	// Non-deployable statuses should return a non-empty reason
	for _, s := range []string{"missing_image", "needs_check", "runtime_image_mismatch", "inspect_failed", "agent_unreachable", "docker_error", "unsupported_device", "disabled", "evidence_missing"} {
		if r := nbrDisabledReason(s, ""); r == "" {
			t.Errorf("nbrDisabledReason(%q) is empty, want non-empty for non-deployable status", s)
		}
	}
	// Fallback should include the status
	if r := nbrDisabledReason("custom_status", ""); r == "" {
		t.Error("nbrDisabledReason for unknown status should not be empty")
	}
}

func TestExtractProbeWarnings(t *testing.T) {
	// Empty probe results — no warnings
	if w := extractProbeWarnings("{}", "ready"); w != nil {
		t.Errorf("extractProbeWarnings({}) = %v, want nil", w)
	}

	// Skipped level 4 — no warnings
	skippedProbe := map[string]interface{}{
		"level4": map[string]interface{}{
			"version_probed": false,
			"probe_skipped":  true,
			"skip_reason":    "version probe not yet implemented",
		},
	}
	skippedJSON, _ := json.Marshal(skippedProbe)
	if w := extractProbeWarnings(string(skippedJSON), "ready"); w != nil {
		t.Errorf("extractProbeWarnings with skipped probe = %v, want nil", w)
	}

	// Failed level 4 (not skipped) — should have warning
	failedProbe := map[string]interface{}{
		"level4": map[string]interface{}{
			"version_probed": false,
			"probe_skipped":  false,
		},
	}
	failedJSON, _ := json.Marshal(failedProbe)
	if w := extractProbeWarnings(string(failedJSON), "ready_with_warnings"); w == nil {
		t.Error("extractProbeWarnings with failed probe = nil, want warning")
	}
}

func TestNBRListResponseIncludesDeployable(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{
			{
				Repository: "vllm/vllm-openai",
				Tag:        "latest",
				ImageRef:   "vllm/vllm-openai:latest",
				ImageID:    "sha256:test-deployable-image",
				Size:       123456789,
			},
		},
		Inspect: workflowInspectSuccess("vllm/vllm-openai:latest", "sha256:test-deployable-image"),
	})
	nodeID := app.InsertOnlineNode(t, "deployable-list-node", fakeAgent)
	app.InsertGPU(t, nodeID, "nvidia")
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "nvidia")
	// Enable NBR
	enableResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          "vllm/vllm-openai:latest",
	}, http.StatusOK)
	var enabled map[string]interface{}
	enableResp.Decode(t, &enabled)
	nbrID := workflowStringField(t, enabled, "id")

	// Run check-request to set ready_with_warnings
	app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/check-request", map[string]interface{}{}, http.StatusOK)

	// List NBRs — verify deployable is present
	listResp := app.Client.JSON(t, http.MethodGet, "/api/v1/nodes/"+nodeID+"/backend-runtimes", nil, http.StatusOK)
	var nbrs []map[string]interface{}
	listResp.Decode(t, &nbrs)

	found := false
	for _, nbr := range nbrs {
		if nbr["id"] == nbrID {
			found = true
			// Verify deployable field exists
			deployable, ok := nbr["deployable"].(bool)
			if !ok {
				t.Errorf("NBR list item missing deployable field or not bool: %#v", nbr)
			} else if !deployable {
				t.Errorf("NBR deployable=false for ready_with_warnings status: %#v", nbr)
			}
			// Verify warnings field exists
			if _, ok := nbr["warnings"]; !ok {
				t.Logf("NBR list item missing warnings field (may be nil): %#v", nbr)
			}
			// Verify disabled_reason is empty for deployable NBR
			if dr, ok := nbr["disabled_reason"].(string); ok && dr != "" {
				t.Errorf("NBR disabled_reason should be empty for deployable status, got %q", dr)
			}
			break
		}
	}
	if !found {
		t.Fatalf("NBR %q not found in list response: %#v", nbrID, nbrs)
	}
}

func TestCreateDeploymentAcceptsReadyWithWarnings(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{
			{
				Repository: "vllm/vllm-openai",
				Tag:        "latest",
				ImageRef:   "vllm/vllm-openai:latest",
				ImageID:    "sha256:test-create-rww-image",
				Size:       123456789,
			},
		},
		Inspect: workflowInspectSuccess("vllm/vllm-openai:latest", "sha256:test-create-rww-image"),
	})
	nodeID := app.InsertOnlineNode(t, "create-rww-node", fakeAgent)
	app.InsertGPU(t, nodeID, "nvidia")
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "nvidia")
	// Enable NBR
	enableResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          "vllm/vllm-openai:latest",
	}, http.StatusOK)
	var enabled map[string]interface{}
	enableResp.Decode(t, &enabled)
	nbrID := workflowStringField(t, enabled, "id")

	// Run check-request — should produce ready or ready_with_warnings
	checkResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/check-request", map[string]interface{}{}, http.StatusOK)
	var check map[string]interface{}
	checkResp.Decode(t, &check)
	status := workflowStringField(t, check, "status")
	if status != "ready" && status != "ready_with_warnings" {
		t.Fatalf("check-request status=%q want ready or ready_with_warnings", status)
	}
	deployable := check["deployable"]
	if deployable != true {
		t.Fatalf("check-request deployable=%v want true for status=%s", deployable, status)
	}

	// Create artifact
	artifactResp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts", map[string]interface{}{
		"name":         "test-create-rww-artifact",
		"display_name": "Test Create RWW",
		"path":         "/tmp/test-model",
		"format":       "huggingface",
		"task_type":    "chat",
	}, http.StatusCreated)
	var artifact map[string]interface{}
	artifactResp.Decode(t, &artifact)
	artifactID := workflowStringField(t, artifact, "id")

	// Add model root and location for the node
	app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/model-roots", map[string]interface{}{
		"path": "/tmp",
	}, http.StatusCreated)
	app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts/"+artifactID+"/locations", map[string]interface{}{
		"node_id":             nodeID,
		"absolute_path":       "/tmp/test-model",
		"path_type":           "directory",
		"verification_status": "verified",
		"match_status":        "exact_match",
	}, http.StatusCreated)

	// Create deployment with ready_with_warnings NBR — should succeed
	createResp := app.Client.JSON(t, http.MethodPost, "/api/v1/deployments", map[string]interface{}{
		"name":                      "test-create-rww-deploy",
		"model_artifact_id":         artifactID,
		"node_backend_runtime_id":   nbrID,
		"placement_json":            map[string]interface{}{"node_id": nodeID, "gpu_ids": []interface{}{}},
		"service_json":              map[string]interface{}{"host_port": 8999},
	}, http.StatusCreated)
	var deploy map[string]interface{}
	createResp.Decode(t, &deploy)
	if deploy["id"] == nil || deploy["id"] == "" {
		t.Fatalf("deployment create failed with deployable NBR: %#v", deploy)
	}
	t.Logf("Created deployment %v with NBR status=%s", deploy["id"], status)
}

func TestCreateDeploymentRejectsBlockedNBR(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images:  []fakeAgentImage{},
		Inspect: map[string]interface{}{"error": "no such image: not-exist/blocked:latest"},
	})
	nodeID := app.InsertOnlineNode(t, "blocked-nbr-node", fakeAgent)
	app.InsertGPU(t, nodeID, "nvidia")
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "nvidia")
	// Enable NBR
	enableResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          "not-exist/blocked:latest",
	}, http.StatusOK)
	var enabled map[string]interface{}
	enableResp.Decode(t, &enabled)
	nbrID := workflowStringField(t, enabled, "id")

	// Run check-request — should produce missing_image
	checkResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/check-request", map[string]interface{}{}, http.StatusOK)
	var check map[string]interface{}
	checkResp.Decode(t, &check)
	status := workflowStringField(t, check, "status")
	if status != "missing_image" {
		t.Fatalf("check-request status=%q want missing_image", status)
	}
	deployable := check["deployable"]
	if deployable != false {
		t.Fatalf("check-request deployable=%v want false for missing_image", deployable)
	}
	disabledReason := check["disabled_reason"]
	if disabledReason == nil || disabledReason == "" {
		t.Errorf("check-request disabled_reason should not be empty for blocked NBR")
	}

	// Create artifact
	artifactResp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts", map[string]interface{}{
		"name":         "test-blocked-artifact",
		"display_name": "Test Blocked",
		"path":         "/tmp/test-blocked-model",
		"format":       "huggingface",
		"task_type":    "chat",
	}, http.StatusCreated)
	var artifact map[string]interface{}
	artifactResp.Decode(t, &artifact)
	artifactID := workflowStringField(t, artifact, "id")

	// Add model root and location
	app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/model-roots", map[string]interface{}{
		"path": "/tmp",
	}, http.StatusCreated)
	app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts/"+artifactID+"/locations", map[string]interface{}{
		"node_id":             nodeID,
		"absolute_path":       "/tmp/test-blocked-model",
		"path_type":           "directory",
		"verification_status": "verified",
		"match_status":        "exact_match",
	}, http.StatusCreated)

	// Create deployment with blocked NBR — should be rejected
	createResp := app.Client.JSON(t, http.MethodPost, "/api/v1/deployments", map[string]interface{}{
		"name":                      "test-blocked-deploy",
		"model_artifact_id":         artifactID,
		"node_backend_runtime_id":   nbrID,
		"placement_json":            map[string]interface{}{"node_id": nodeID, "gpu_ids": []interface{}{}},
		"service_json":              map[string]interface{}{"host_port": 8998},
	}, http.StatusBadRequest)
	var errResp map[string]interface{}
	createResp.Decode(t, &errResp)
	t.Logf("Blocked NBR correctly rejected: %v", errResp["error"])
}
