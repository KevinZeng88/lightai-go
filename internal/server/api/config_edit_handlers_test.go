package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lightai-go/internal/server/db"
)

func TestConfigEditViewAPIProjectsRuntimeWithoutInternalOrdinaryLabels(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	insertRuntime(t, db, "rt-config-edit-view", "Runtime Config Edit View", "")

	w := httptest.NewRecorder()
	h.HandleConfigEditView(w, newReq("POST", "/x", `{"object_kind":"backend_runtime","object_id":"rt-config-edit-view","layer":"backend_runtime"}`, adminSession(), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode view: %v", err)
	}
	view, _ := response["config_edit_view"].(map[string]any)
	if view == nil {
		view = response // backward compat: old format had sections at top level
	}
	raw, _ := json.Marshal(view["sections"])
	if strings.Contains(string(raw), `"label":"launcher.docker_options"`) || strings.Contains(string(raw), `"label":"runtime.env"`) {
		t.Fatalf("ordinary labels expose internal keys: %s", raw)
	}
	if !strings.Contains(string(raw), `"key":"docker.shm_size"`) || !strings.Contains(string(raw), `"key":"docker.privileged"`) {
		t.Fatalf("docker options were not projected as structured fields: %s", raw)
	}
}

func TestNodeBackendRuntimeEnableAppliesEditableConfigPatch(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-config-edit")
	insertRuntime(t, db, "rt-config-edit-nbr", "Runtime Config Edit NBR", "")

	w := httptest.NewRecorder()
	body := jsonString(map[string]any{
		"backend_runtime_id": "rt-config-edit-nbr",
		"editable_config_patch": map[string]any{
			"layer":     "node_backend_runtime",
			"object_id": "node-config-edit:rt-config-edit-nbr",
			"fields": []map[string]any{
				{"key": "launcher.docker_options.shm_size", "internal_key": "launcher.docker_options", "path": []string{"shm_size"}, "value": "24gb", "enabled": true},
			},
		},
	})
	h.HandleEnableNodeBackendRuntime(w, newReq("POST", "/x", body, adminSession(), map[string]string{"id": "node-config-edit"}))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var raw string
	if err := db.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE id='node-config-edit:rt-config-edit-nbr'`).Scan(&raw); err != nil {
		t.Fatalf("read NBR config: %v", err)
	}
	set := parseConfigSet(raw)
	docker := configObject(set, "launcher.docker_options")
	if docker["shm_size"] != "24gb" {
		t.Fatalf("editable_config_patch not applied to NBR snapshot: %s", raw)
	}
}

func TestDeploymentCreateAppliesEditableConfigPatchToSnapshot(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	runtimeBoundaryInsertOnlineNode(t, db, "node-config-edit-dep")
	insertRuntime(t, db, "rt-config-edit-dep", "Runtime Config Edit Deployment", "")
	var rtSetRaw string
	if err := db.QueryRow(`SELECT config_set_json FROM backend_runtimes WHERE id='rt-config-edit-dep'`).Scan(&rtSetRaw); err != nil {
		t.Fatalf("read runtime config set: %v", err)
	}
	rtSet := copyConfigSet(rtSetRaw)
	items := configSetItems(rtSet)
		items["model_runtime.max_model_len"] = map[string]interface{}{
			"schema": map[string]interface{}{"key": "model_runtime.max_model_len", "category": "model_runtime", "kind": "cli_arg", "type": "integer"},
			"state":  map[string]interface{}{"enabled": false, "checked": false},
			"value":  map[string]interface{}{"default_value": float64(2048), "effective_value": float64(2048)},
		}
	if _, err := db.Exec(`UPDATE backend_runtimes SET config_set_json=? WHERE id='rt-config-edit-dep'`, configSetJSON(rtSet)); err != nil {
		t.Fatalf("update runtime config set: %v", err)
	}
	insertNodeBackendRuntime(t, db, "node-config-edit-dep:rt-config-edit-dep", "rt-config-edit-dep", "node-config-edit-dep", "img:dep", "ready", "", 1, 1, "")
	insertDeploymentArtifactLocation(t, db, "art-config-edit-dep", "node-config-edit-dep")

	w := httptest.NewRecorder()
	body := jsonString(map[string]any{
		"name":                    "dep-config-edit",
		"model_artifact_id":       "art-config-edit-dep",
		"node_backend_runtime_id": "node-config-edit-dep:rt-config-edit-dep",
		"editable_config_patch": map[string]any{
			"layer":     "deployment",
			"object_id": "new",
			"fields": []map[string]any{
				{"key": "model_runtime.max_model_len", "internal_key": "model_runtime.max_model_len", "value": 4096, "enabled": true},
			},
		},
	})
	h.HandleCreateDeployment(w, newReq("POST", "/x", body, adminSession(), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode deployment: %v", err)
	}
	set, _ := resp["config_set"].(map[string]any)
		item, _ := configSetItems(set)["model_runtime.max_model_len"].(map[string]interface{})
		if item == nil || !configItemEnabled(item) {
		t.Fatalf("deployment editable_config_patch enabled not set: %s", configSetJSON(set))
		}
		if vt, ok := item["value"].(map[string]interface{}); !ok || vt["effective_value"] != float64(4096) {
		t.Fatalf("deployment editable_config_patch value not in snapshot: %s", configSetJSON(set))
		}
	}

func insertDeploymentArtifactLocation(t *testing.T, db *db.DB, artifactID, nodeID string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO model_artifacts (id,name,display_name,source_type,tenant_id,created_at,updated_at)
		VALUES (?,?,?,?,?,datetime('now'),datetime('now'))`, artifactID, artifactID, artifactID, "manual", ""); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO model_locations (id,model_artifact_id,node_id,model_root,relative_path,absolute_path,path_type,verification_status,match_status,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,datetime('now'),datetime('now'))`, artifactID+"-loc", artifactID, nodeID, "/models", artifactID, "/models/"+artifactID, "directory", "verified", "exact_match"); err != nil {
		t.Fatalf("insert location: %v", err)
	}
}
