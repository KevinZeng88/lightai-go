package api

import (
	"net/http"
	"reflect"
	"testing"
)

// TestModelCapabilityDefaultValues verifies that newly created artifacts get
// correct default values for capability persistence columns.
func TestModelCapabilityDefaultValues(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts", map[string]interface{}{
		"name":         "cap-default-test",
		"display_name": "Cap Default Test",
		"source_type":  "local_path",
		"path":         "/models/cap-default-test/model.gguf",
		"format":       "gguf",
	}, http.StatusCreated)
	var artifact map[string]interface{}
	resp.Decode(t, &artifact)

	// Verify default values.
	caps, ok := artifact["capabilities"].([]interface{})
	if !ok {
		t.Fatalf("capabilities should be an array, got %T: %v", artifact["capabilities"], artifact["capabilities"])
	}
	if len(caps) != 0 {
		t.Fatalf("expected empty capabilities, got %v", caps)
	}

	sources, ok := artifact["capability_sources"].(map[string]interface{})
	if !ok {
		t.Fatalf("capability_sources should be a map, got %T: %v", artifact["capability_sources"], artifact["capability_sources"])
	}
	if len(sources) != 0 {
		t.Fatalf("expected empty capability_sources, got %v", sources)
	}

	dtm, ok := artifact["default_test_mode"].(string)
	if !ok || dtm != "auto" {
		t.Fatalf("expected default_test_mode='auto', got %v", artifact["default_test_mode"])
	}
}

// TestPatchModelArtifactCapabilities verifies that capabilities can be updated
// via PATCH and the updated values are returned in GET list and detail.
func TestPatchModelArtifactCapabilities(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	// Create an artifact.
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts", map[string]interface{}{
		"name":         "cap-patch-test",
		"display_name": "Cap Patch Test",
		"source_type":  "local_path",
		"path":         "/models/cap-patch-test/model.gguf",
		"format":       "gguf",
	}, http.StatusCreated)
	var artifact map[string]interface{}
	resp.Decode(t, &artifact)
	artifactID := artifact["id"].(string)

	// Patch capabilities and default_test_mode.
	patchResp := app.Client.JSON(t, "PATCH", "/api/v1/model-artifacts/"+artifactID, map[string]interface{}{
		"capabilities":      []interface{}{"chat", "completion"},
		"default_test_mode": "chat",
	}, http.StatusOK)
	var patched map[string]interface{}
	patchResp.Decode(t, &patched)

	// Verify capabilities in response.
	caps, ok := patched["capabilities"].([]interface{})
	if !ok {
		t.Fatalf("patched capabilities should be array, got %T", patched["capabilities"])
	}
	if len(caps) != 2 {
		t.Fatalf("expected 2 capabilities, got %d: %v", len(caps), caps)
	}
	found := map[string]bool{}
	for _, c := range caps {
		found[c.(string)] = true
	}
	if !found["chat"] || !found["completion"] {
		t.Fatalf("expected [chat, completion], got %v", caps)
	}

	// Sources should be marked as user_override.
	sources, ok := patched["capability_sources"].(map[string]interface{})
	if !ok {
		t.Fatalf("capability_sources should be map, got %T", patched["capability_sources"])
	}
	if sources["chat"] != "user_override" {
		t.Fatalf("expected chat source=user_override, got %v", sources["chat"])
	}

	// default_test_mode should be chat.
	if patched["default_test_mode"] != "chat" {
		t.Fatalf("expected default_test_mode=chat, got %v", patched["default_test_mode"])
	}

	// Verify GET detail returns the same.
	detail := app.Client.JSON(t, http.MethodGet, "/api/v1/model-artifacts/"+artifactID, nil, http.StatusOK)
	var detailArtifact map[string]interface{}
	detail.Decode(t, &detailArtifact)
	detailCaps := detailArtifact["capabilities"].([]interface{})
	if len(detailCaps) != 2 {
		t.Fatalf("detail capabilities mismatch: %v", detailCaps)
	}

	// Verify GET list returns the same.
	listResp := app.Client.JSON(t, http.MethodGet, "/api/v1/model-artifacts", nil, http.StatusOK)
	var artifacts []map[string]interface{}
	listResp.Decode(t, &artifacts)
	foundInList := false
	for _, a := range artifacts {
		if a["id"] == artifactID {
			foundInList = true
			listCaps := a["capabilities"].([]interface{})
			if len(listCaps) != 2 {
				t.Fatalf("list capabilities mismatch: %v", listCaps)
			}
			if !reflect.DeepEqual(listCaps, detailCaps) {
				t.Fatalf("list/detail capabilities mismatch: list=%v detail=%v", listCaps, detailCaps)
			}
			break
		}
	}
	if !foundInList {
		t.Fatalf("artifact not found in list response")
	}
}

// TestModelArtifactDefaultTestModeValidation verifies that invalid
// default_test_mode values are rejected.
func TestModelArtifactDefaultTestModeValidation(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	// Create an artifact.
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts", map[string]interface{}{
		"name":         "cap-dtm-validate-test",
		"display_name": "DTM Validate Test",
		"source_type":  "local_path",
		"path":         "/models/dtm-validate/model.gguf",
		"format":       "gguf",
	}, http.StatusCreated)
	var artifact map[string]interface{}
	resp.Decode(t, &artifact)
	artifactID := artifact["id"].(string)

	// Try invalid default_test_mode.
	app.Client.JSON(t, "PATCH", "/api/v1/model-artifacts/"+artifactID, map[string]interface{}{
		"default_test_mode": "invalid_mode",
	}, http.StatusBadRequest)

	// Verify it still has the default.
	detail := app.Client.JSON(t, http.MethodGet, "/api/v1/model-artifacts/"+artifactID, nil, http.StatusOK)
	var detailArtifact map[string]interface{}
	detail.Decode(t, &detailArtifact)
	if detailArtifact["default_test_mode"] != "auto" {
		t.Fatalf("expected default_test_mode=auto after failed patch, got %v", detailArtifact["default_test_mode"])
	}
}

// TestModelArtifactInvalidCapabilityRejected verifies that invalid capability
// values are rejected.
func TestModelArtifactInvalidCapabilityRejected(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	// Create an artifact.
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/model-artifacts", map[string]interface{}{
		"name":         "cap-invalid-test",
		"display_name": "Cap Invalid Test",
		"source_type":  "local_path",
		"path":         "/models/cap-invalid/model.gguf",
		"format":       "gguf",
	}, http.StatusCreated)
	var artifact map[string]interface{}
	resp.Decode(t, &artifact)
	artifactID := artifact["id"].(string)

	// Try invalid capability.
	app.Client.JSON(t, "PATCH", "/api/v1/model-artifacts/"+artifactID, map[string]interface{}{
		"capabilities": []interface{}{"invalid_capability"},
	}, http.StatusBadRequest)

	// Verify capabilities are still empty.
	detail := app.Client.JSON(t, http.MethodGet, "/api/v1/model-artifacts/"+artifactID, nil, http.StatusOK)
	var detailArtifact map[string]interface{}
	detail.Decode(t, &detailArtifact)
	caps := detailArtifact["capabilities"].([]interface{})
	if len(caps) != 0 {
		t.Fatalf("expected empty capabilities after failed patch, got %v", caps)
	}
}
