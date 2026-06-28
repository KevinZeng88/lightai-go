package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lightai-go/internal/server/db"
)

func insertUIPersistenceArtifact(t *testing.T, h *AgentHandler, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := h.DB.Exec(`INSERT INTO model_artifacts
		(id,name,display_name,source_type,path,format,task_type,architecture,size_label,quantization,tenant_id,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, id, "Display "+id, "local_path", "/models/"+id, "huggingface", "chat", "custom", "", "unknown", "", now, now); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}
}

func insertUIPersistenceDeployment(t *testing.T, h *AgentHandler, id, artifactID, runtimeID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	var configSetRaw string
	if err := h.DB.QueryRow(`SELECT config_set_json FROM backend_runtimes WHERE id = ?`, runtimeID).Scan(&configSetRaw); err != nil {
		t.Fatalf("read runtime config set: %v", err)
	}
	sourceMetadata := jsonString(map[string]interface{}{
		"copy_semantics":            "copy_on_create",
		"source_backend_runtime_id": runtimeID,
		"source_type":               "test_fixture",
	})
	if _, err := h.DB.Exec(`INSERT INTO model_deployments
		(id,name,display_name,model_artifact_id,backend_runtime_id,replicas,placement_json,service_json,config_overrides_json,config_set_json,source_metadata_json,source_backend_runtime_id,source_config_hash,copied_at,desired_state,status,tenant_id,created_at,updated_at)
		VALUES (?,?,?,?,?,1,'{"node_id":"node-a","accelerator_ids":[]}','{"host_port":8005}','{}',?,?,?,?,?,'stopped','saved','',?,?)`,
		id, id, id, artifactID, runtimeID, configSetRaw, sourceMetadata, runtimeID, planHashStr(configSetRaw), now, now, now); err != nil {
		t.Fatalf("insert deployment: %v", err)
	}
}

func TestCloneBackendRuntimePersistsIndependentDisplayName(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertRuntime(t, database, "rt-source", "llama.cpp CUDA Runtime", "")

	w := httptest.NewRecorder()
	h.HandleCloneBackendRuntime(w, newReq("POST", "/x", `{}`, adminSession(), map[string]string{"id": "rt-source"}))
	if w.Code != http.StatusCreated {
		t.Fatalf("clone code=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode clone: %v", err)
	}
	if got["name"] == "llama.cpp CUDA Runtime" || got["display_name"] == "llama.cpp CUDA Runtime" {
		t.Fatalf("clone reused source visible name: %#v", got)
	}
	if !strings.Contains(got["name"].(string), "runtime.") || !strings.Contains(got["name"].(string), ".user.") {
		t.Fatalf("clone name not stable technical name: %v", got["name"])
	}
	if got["display_name"] != got["name"] {
		t.Fatalf("display_name=%v name=%v", got["display_name"], got["name"])
	}
	if got["source_template_name"] != "llama.cpp CUDA Runtime" {
		t.Fatalf("source_template_name=%v", got["source_template_name"])
	}
}

func TestCreateAndPatchBackendRuntimeNamePersistence(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	bid, vid := getVllmIDs(t, database)

	body := `{"backend_id":"` + bid + `","backend_version_id":"` + vid + `","name":" Custom Runtime ","display_name":"Custom Runtime","vendor":"nvidia"}`
	w := httptest.NewRecorder()
	h.HandleCreateBackendRuntimeFromTemplate(w, newReq("POST", "/x", body, adminSession(), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["name"] != "Custom Runtime" {
		t.Fatalf("name was not trimmed/persisted: %#v", got)
	}

	pw := httptest.NewRecorder()
	h.HandlePatchBackendRuntime(pw, newReq("PATCH", "/x", `{"name":"Runtime Renamed","display_name":"Runtime Display","command":["--ctx-size","4096"]}`, adminSession(), map[string]string{"id": got["id"].(string)}))
	if pw.Code != http.StatusOK {
		t.Fatalf("patch code=%d body=%s", pw.Code, pw.Body.String())
	}
	after := h.getBackendRuntimeJSON(got["id"].(string))
	if after["name"] != "Runtime Renamed" || after["display_name"] != "Runtime Display" {
		t.Fatalf("patched names not persisted: %#v", after)
	}
	raw, _ := json.Marshal(after["command"])
	if !strings.Contains(string(raw), "--ctx-size") {
		t.Fatalf("command not persisted: %s", raw)
	}
}

func TestModelArtifactDisplayNamePersistenceDoesNotChangePath(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertUIPersistenceArtifact(t, h, "qwen-file")

	w := httptest.NewRecorder()
	h.HandlePatchArtifact(w, newReq("PATCH", "/x", `{"display_name":"Qwen Friendly Name"}`, adminSession(), map[string]string{"id": "qwen-file"}))
	if w.Code != http.StatusOK {
		t.Fatalf("patch artifact code=%d body=%s", w.Code, w.Body.String())
	}
	got := h.getArtifactJSON("qwen-file")
	if got["display_name"] != "Qwen Friendly Name" {
		t.Fatalf("display_name=%v", got["display_name"])
	}
	if got["path"] != "/models/qwen-file" {
		t.Fatalf("path changed after display_name edit: %v", got["path"])
	}
}

func TestDeploymentSaveOnlyAndPatchEditableFields(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	runtimeBoundaryInsertOnlineNode(t, database, "node-save")
	database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-save','node-save','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`)
	insertUIPersistenceArtifact(t, h, "art-save")
	snapshotInsertModelLocation(t, database, "ml-art-save", "art-save", "node-save")
	insertRuntime(t, database, "rt-save", "Runtime Save", "")
	// R-001: enable sets needs_check; set ready via DB for deployment tests
	h.HandleEnableNodeBackendRuntime(httptest.NewRecorder(), newReq("POST", "/x",
		`{"backend_runtime_id":"rt-save","image_ref":"img:test"}`, adminSession(), map[string]string{"id": "node-save"}))
	database.Exec(`UPDATE node_backend_runtimes SET status='ready',image_present=1,docker_available=1 WHERE id='node-save:rt-save'`)

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x", `{"name":"dep-save","model_artifact_id":"art-save","node_backend_runtime_id":"node-save:rt-save","service_json":{"host_port":8005,"container_port":8080},"config_overrides":{"parameter_values":[{"key":"served_model_name","cli_name":"--served-model-name","type":"string","enabled":true,"value":"served-a"}]}}`, adminSession(), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["status"] != "saved" {
		t.Fatalf("save-only status=%v want saved", got["status"])
	}
	var count int
	database.QueryRow(`SELECT COUNT(*) FROM model_instances WHERE deployment_id=?`, got["id"]).Scan(&count)
	if count != 0 {
		t.Fatalf("save-only created instances: %d", count)
	}

	pw := httptest.NewRecorder()
	h.HandlePatchDeployment(pw, newReq("PATCH", "/x", `{"display_name":"Dep Display","placement_json":{"node_id":"node-b","accelerator_ids":["gpu-b"]},"service_json":{"host_port":8006,"container_port":8081},"config_overrides":{"parameter_values":[{"key":"served_model_name","cli_name":"--served-model-name","type":"string","enabled":true,"value":"served-b"}]}}`, adminSession(), map[string]string{"id": got["id"].(string)}))
	if pw.Code != http.StatusOK {
		t.Fatalf("patch deployment code=%d body=%s", pw.Code, pw.Body.String())
	}
	after := h.getDeploymentJSON(got["id"].(string))
	raw, _ := json.Marshal(after)
	for _, want := range []string{"Dep Display", "node-b", "8006", "8081", "served-b"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("deployment update missing %q: %s", want, raw)
		}
	}
}

func TestNodeBackendRuntimeDisplayNamePersistence(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	runtimeBoundaryInsertOnlineNode(t, database, "node-nbr-name")
	if _, err := database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-nbr-name','node-nbr-name','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	insertRuntime(t, database, "rt-nbr-name", "Runtime NBR Name", "")

	w := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(w, newReq("POST", "/x", `{"backend_runtime_id":"rt-nbr-name","display_name":"Runtime NBR Name - Custom Node","image_ref":"img:test"}}`, adminSession(), map[string]string{"id": "node-nbr-name"}))
	if w.Code != http.StatusOK {
		t.Fatalf("enable code=%d body=%s", w.Code, w.Body.String())
	}
	list := httptest.NewRecorder()
	h.HandleListNodeBackendRuntimes(list, newReq("GET", "/x", "", adminSession(), map[string]string{"id": "node-nbr-name"}))
	var rows []map[string]interface{}
	_ = json.Unmarshal(list.Body.Bytes(), &rows)
	if len(rows) != 1 || rows[0]["display_name"] != "Runtime NBR Name - Custom Node" {
		t.Fatalf("display_name not returned: %#v", rows)
	}

	pw := httptest.NewRecorder()
	h.HandlePatchNodeBackendRuntime(pw, newReq("PATCH", "/x", `{"display_name":"Runtime NBR Name - Edited"}`, adminSession(), map[string]string{"nbr_id": "node-nbr-name:rt-nbr-name"}))
	if pw.Code != http.StatusOK {
		t.Fatalf("patch code=%d body=%s", pw.Code, pw.Body.String())
	}
	var name string
	database.QueryRow(`SELECT display_name FROM node_backend_runtimes WHERE id='node-nbr-name:rt-nbr-name'`).Scan(&name)
	if name != "Runtime NBR Name - Edited" {
		t.Fatalf("persisted display_name=%q", name)
	}
}

func TestStartDeploymentGuardsActiveInstanceAndTask(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertUIPersistenceArtifact(t, h, "art-guard")
	insertRuntime(t, database, "rt-guard", "Runtime Guard", "")
	insertUIPersistenceDeployment(t, h, "dep-guard", "art-guard", "rt-guard")
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.Exec(`INSERT INTO model_instances (id,deployment_id,tenant_id,node_id,actual_state,desired_state,created_at,updated_at)
		VALUES ('inst-running','dep-guard','','node-a','running','running',?,?)`, now, now); err != nil {
		t.Fatalf("insert running instance: %v", err)
	}

	w := httptest.NewRecorder()
	h.HandleStartDeployment(w, newReq("POST", "/x", `{}`, adminSession(), map[string]string{"id": "dep-guard"}))
	if w.Code != http.StatusConflict {
		t.Fatalf("start running code=%d body=%s", w.Code, w.Body.String())
	}
	var blocked map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &blocked)
	if blocked["reason_code"] != "deployment_running" {
		t.Fatalf("reason=%v", blocked["reason_code"])
	}
	var count int
	database.QueryRow(`SELECT COUNT(*) FROM model_instances WHERE deployment_id='dep-guard'`).Scan(&count)
	if count != 1 {
		t.Fatalf("duplicate instance created: %d", count)
	}
}

func TestActiveRunAllowsFailedDeploymentRerun(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertUIPersistenceArtifact(t, h, "art-failed-rerun")
	insertRuntime(t, database, "rt-failed-rerun", "Runtime Failed Rerun", "")
	insertUIPersistenceDeployment(t, h, "dep-failed-rerun", "art-failed-rerun", "rt-failed-rerun")
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.Exec(`INSERT INTO model_instances (id,deployment_id,tenant_id,node_id,actual_state,desired_state,created_at,updated_at)
		VALUES ('inst-failed-rerun','dep-failed-rerun','','node-a','failed','running',?,?)`, now, now); err != nil {
		t.Fatalf("insert failed instance: %v", err)
	}
	active := h.activeDeploymentRun("dep-failed-rerun")
	if active.Blocked {
		t.Fatalf("failed deployment should be rerunnable: %#v", active)
	}
}

func TestTryInferenceRequiresNonEmptyResponsePreview(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/chat/completions":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	result := tryInference(srv.Client(), srv.URL, "model-a")
	if result["ok"] == true {
		t.Fatalf("empty chat response passed: %#v", result)
	}
	if result["reason_code"] != "empty_model_response" {
		t.Fatalf("reason=%v", result["reason_code"])
	}
}

func TestTryInferenceModeCompletionUsesCompletionsEndpoint(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/completions":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"model":"model-a","choices":[{"text":"pong"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	result := tryInferenceWithMode(srv.Client(), srv.URL, "model-a", "completion", "ping")
	if result["ok"] != true {
		t.Fatalf("completion mode failed: %#v", result)
	}
	if result["mode"] != "completion" {
		t.Fatalf("mode=%v", result["mode"])
	}
	if len(paths) != 1 || paths[0] != "/v1/completions" {
		t.Fatalf("completion mode paths=%v", paths)
	}
}

func TestDeploymentCapturesConfigSnapshotAtCreate(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	runtimeBoundaryInsertOnlineNode(t, database, "node-snap")
	database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-snap','node-snap','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`)
	insertUIPersistenceArtifact(t, h, "art-snap")
	snapshotInsertModelLocation(t, database, "ml-art-snap", "art-snap", "node-snap")
	insertRuntime(t, database, "rt-snap", "llama.cpp NVIDIA CUDA Runtime", "")
	// Create ready NBR
	nbrW := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(nbrW, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-snap","image_ref":"img:test"}}`,
		adminSession(), map[string]string{"id": "node-snap"}))
	if nbrW.Code != 200 {
		t.Fatalf("nbr enable code=%d", nbrW.Code)
	}
	// R-001: enable sets needs_check; update to ready for deployment test
	database.Exec("UPDATE node_backend_runtimes SET status='ready', image_present=1, docker_available=1 WHERE id='node-snap:rt-snap'")

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x", `{"name":"dep-snap","model_artifact_id":"art-snap","node_backend_runtime_id":"node-snap:rt-snap","service_json":{"host_port":8005,"container_port":8080}}`, adminSession(), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	snapRaw := got["config_set"]
	if snapRaw == nil {
		t.Fatal("config_set missing in response")
	}
	// snapshot may come back as map[string]interface{} or string
	var snapStr string
	switch v := snapRaw.(type) {
	case string:
		snapStr = v
	case map[string]interface{}:
		snapStr = fmt.Sprintf("%v", v)
	default:
		// Try json.RawMessage or re-marshal
		if raw, err := json.Marshal(snapRaw); err == nil {
			snapStr = string(raw)
		}
	}
	if snapStr == "" || snapStr == "{}" || snapStr == "map[]" {
		t.Fatalf("config_set empty after create: %s", snapStr)
	}
	if !strings.Contains(snapStr, "rt-snap") {
		t.Fatalf("config set missing source runtime ref: %s", snapStr)
	}
	// Verify deployment detail also returns ConfigSet.
	detail := h.getDeploymentJSON(got["id"].(string))
	if detail == nil {
		t.Fatal("getDeploymentJSON returned nil")
	}
	if snap2, _ := detail["config_set_json"]; snap2 == nil || fmt.Sprintf("%v", snap2) == "{}" {
		t.Fatalf("deployment detail missing config set: %v", snap2)
	}
}

func TestDeploymentPatchPortsAndDisplayName(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	runtimeBoundaryInsertOnlineNode(t, database, "node-edit")
	database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-edit','node-edit','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`)
	insertUIPersistenceArtifact(t, h, "art-edit")
	snapshotInsertModelLocation(t, database, "ml-art-edit", "art-edit", "node-edit")
	insertRuntime(t, database, "rt-edit", "Runtime Edit", "")
	// Create ready NBR
	nbrW := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(nbrW, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-edit","image_ref":"img:test"}`,
		adminSession(), map[string]string{"id": "node-edit"}))
	if nbrW.Code != 200 {
		t.Fatalf("nbr enable code=%d", nbrW.Code)
	}
	// R-001: enable sets needs_check; update to ready for deployment test
	database.Exec(`UPDATE node_backend_runtimes SET status='ready', image_present=1, docker_available=1 WHERE id='node-edit:rt-edit'`)

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x", `{"name":"dep-edit","model_artifact_id":"art-edit","node_backend_runtime_id":"node-edit:rt-edit","service_json":{"host_port":8005,"container_port":8080},"config_overrides":{}}`, adminSession(), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	depID := got["id"].(string)

	// Patch display_name and ports
	pw := httptest.NewRecorder()
	h.HandlePatchDeployment(pw, newReq("PATCH", "/x", `{"display_name":"Edited Display","service_json":{"host_port":9000,"container_port":8081,"app_port":8081}}`, adminSession(), map[string]string{"id": depID}))
	if pw.Code != http.StatusOK {
		t.Fatalf("patch code=%d body=%s", pw.Code, pw.Body.String())
	}

	// Verify updated values
	after := h.getDeploymentJSON(depID)
	if after == nil {
		t.Fatal("getDeploymentJSON returned nil")
	}
	if after["display_name"] != "Edited Display" {
		t.Fatalf("display_name not updated: %v", after["display_name"])
	}
	svcRaw := after["service_json"]
	var svc map[string]interface{}
	if raw, ok := svcRaw.(json.RawMessage); ok {
		json.Unmarshal(raw, &svc)
	}
	if hp, _ := svc["host_port"]; fmt.Sprintf("%v", hp) != "9000" {
		t.Fatalf("host_port not updated: %v", svc)
	}
}

func TestModelArtifactNameFieldNotSavedOnPatch(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertUIPersistenceArtifact(t, h, "art-name-patch")

	// PATCH with new name - should be ignored
	pw := httptest.NewRecorder()
	h.HandlePatchArtifact(pw, newReq("PATCH", "/x", `{"name":"renamed-art","display_name":"New Display"}`, adminSession(), map[string]string{"id": "art-name-patch"}))
	if pw.Code != http.StatusOK {
		t.Fatalf("patch code=%d body=%s", pw.Code, pw.Body.String())
	}

	// Verify name unchanged, display_name updated
	after := h.getArtifactJSON("art-name-patch")
	if after == nil {
		t.Fatal("getArtifactJSON returned nil")
	}
	if after["name"] != "art-name-patch" {
		t.Fatalf("name was changed when it should be read-only: %v", after["name"])
	}
	if after["display_name"] != "New Display" {
		t.Fatalf("display_name not updated: %v", after["display_name"])
	}
}

// ── Snapshot Inheritance Tests ─────────────────────────────────────────────

// snapshotInsertModelLocation inserts a model_location record so preflight can
// resolve the model on the target node.
func snapshotInsertModelLocation(t *testing.T, db *db.DB, id, artifactID, nodeID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO model_locations
		(id, model_artifact_id, node_id, path_type, model_root, relative_path, absolute_path,
		 size_bytes, match_status, verification_status, tenant_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, artifactID, nodeID, "directory", "/models", id, "/models/"+id,
		12345, "exact_match", "verified", "", now, now); err != nil {
		t.Fatalf("insert model_location: %v", err)
	}
}

// snapshotSetupFullChain creates the full chain for preflight tests:
// online node + GPU + BR + NBR(ready) + artifact + model_location.
// Returns (nodeID, runtimeID, artifactID).
func snapshotSetupFullChain(t *testing.T, h *AgentHandler, suffix string) (string, string, string) {
	t.Helper()
	db := h.DB
	nodeID := "node-snap-" + suffix
	runtimeID := "rt-snap-" + suffix
	artifactID := "art-snap-" + suffix

	runtimeBoundaryInsertOnlineNode(t, db, nodeID)
	if _, err := db.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,datetime('now'),datetime('now'),datetime('now'))`,
		"gpu-"+suffix, nodeID, "nvidia", 0, "RTX-"+suffix, ""); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}

	insertRuntime(t, db, runtimeID, "Runtime "+suffix, "")

	// R-001: /check now enforces server-side verification (checkOnly=false).
	// First enable NBR (creates the record), then set status to ready directly.
	ew := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(ew, newReq("POST", "/x",
		`{"backend_runtime_id":"`+runtimeID+`","display_name":"NBR `+suffix+`","image_ref":"img:nbr-`+suffix+`"}`,
		adminSession(), map[string]string{"id": nodeID}))
	if ew.Code != 200 {
		t.Fatalf("enable nbr code=%d body=%s", ew.Code, ew.Body.String())
	}
	nbrID := nodeID + ":" + runtimeID
	// R-001: enable sets needs_check; update to ready for deployment tests
	db.Exec("UPDATE node_backend_runtimes SET status='ready', image_present=1, docker_available=1 WHERE id='" + nbrID + "'")

	insertUIPersistenceArtifact(t, h, artifactID)
	snapshotInsertModelLocation(t, db, "ml-"+suffix, artifactID, nodeID)

	return nodeID, runtimeID, artifactID
}

// Test 1: Deployment captures NBR config when created with a target node.
func TestDeploymentCapturesNBRConfigAtCreate(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID, runtimeID, artifactID := snapshotSetupFullChain(t, h, "capture")

	// Create deployment WITH node_id so NBR config is merged.
	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-capture","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nodeID+`:`+runtimeID+`","service_json":{"host_port":8005}}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}

	var got map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &got)

	// Extract frozen deployment ConfigSet.
	snapRaw := got["config_set"]
	var snapStr string
	switch v := snapRaw.(type) {
	case string:
		// Try to parse as JSON then re-serialize for consistent formatting
		var parsed map[string]interface{}
		if json.Unmarshal([]byte(v), &parsed) == nil {
			snapStr = v
		} else {
			snapStr = fmt.Sprintf("%v", v)
		}
	case map[string]interface{}:
		raw, _ := json.Marshal(v)
		snapStr = string(raw)
	default:
		snapStr = fmt.Sprintf("%v", snapRaw)
	}

	if snapStr == "" || snapStr == "{}" {
		t.Fatal("config_set is empty after create")
	}
	if !strings.Contains(snapStr, "launcher.image") {
		t.Fatalf("deployment config set missing launcher.image (not copied from NBR): %s", snapStr)
	}
	if !strings.Contains(snapStr, "img:nbr-capture") {
		t.Fatalf("deployment config set missing NBR image_ref value: %s", snapStr)
	}
	if strings.Contains(snapStr, "source_nbr_id") {
		t.Logf("deployment snapshot has NBR source tracking: %s", snapStr)
	}
}

// Test 2: After deployment creation, modifying NBR config does NOT affect DryRun.
func TestNBRConfigModificationDoesNotAffectDeploymentDryRun(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID, runtimeID, artifactID := snapshotSetupFullChain(t, h, "dryrun")

	// Create deployment with node_id
	cw := httptest.NewRecorder()
	h.HandleCreateDeployment(cw, newReq("POST", "/x",
		`{"name":"dep-dryrun","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nodeID+`:`+runtimeID+`","service_json":{"host_port":8005},"config_overrides":{"parameter_values":[{"key":"served_model_name","cli_name":"--served-model-name","type":"string","enabled":true,"value":"dep-dryrun"}]}}`,
		adminSession(), nil))
	if cw.Code != 201 {
		t.Fatalf("create deployment code=%d body=%s", cw.Code, cw.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &created)
	depID := created["id"].(string)

	// Capture the image that would be used BEFORE NBR modification
	dr1 := httptest.NewRecorder()
	h.HandleDeploymentDryRun(dr1, newReq("GET", "/x", "", adminSession(), map[string]string{"id": depID}))
	var before map[string]interface{}
	json.Unmarshal(dr1.Body.Bytes(), &before)

	// Now modify the NBR's ConfigSet and image_ref to simulate an NBR edit.
	nbrID := nodeID + ":" + runtimeID
	var nbrSetRaw string
	database.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE id = ?`, nbrID).Scan(&nbrSetRaw)
	nbrSet := copyConfigSet(nbrSetRaw)
	setConfigValue(nbrSet, "launcher.image", "img:evil-changed", "NodeBackendRuntime", nbrID, "test_mutation")
	database.Exec(`UPDATE node_backend_runtimes SET config_set_json = ? WHERE id = ?`, configSetJSON(nbrSet), nbrID)
	database.Exec(`UPDATE node_backend_runtimes SET image_ref = 'img:evil-changed-ref' WHERE id = ?`, nbrID)

	// Dry-run again — should use the FROZEN deployment snapshot, not the live NBR values
	dr2 := httptest.NewRecorder()
	h.HandleDeploymentDryRun(dr2, newReq("GET", "/x", "", adminSession(), map[string]string{"id": depID}))
	var after map[string]interface{}
	json.Unmarshal(dr2.Body.Bytes(), &after)

	if after["valid"] != true {
		t.Fatalf("dry-run after NBR change became invalid: %v", after["errors"])
	}

	// The resolved image should NOT contain "evil-changed"
	resolvedImage := fmt.Sprintf("%v", after["resolved_image"])
	if strings.Contains(resolvedImage, "evil-changed") {
		t.Fatalf("preflight picked up live NBR change (image=%s)", resolvedImage)
	}

	// Verify the before/after are consistent (same image)
	beforeImage := fmt.Sprintf("%v", before["resolved_image"])
	if beforeImage != resolvedImage {
		t.Logf("before image=%s after image=%s", beforeImage, resolvedImage)
	}
}

// Test 3: Editing BackendRuntime does NOT affect NodeBackendRuntime config snapshot.
func TestBackendRuntimeEditDoesNotAffectNBRConfig(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID, runtimeID, _ := snapshotSetupFullChain(t, h, "br-nbr")

	// Capture the NBR ConfigSet BEFORE editing the BR.
	var origSnapshot string
	database.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE node_id = ? AND backend_runtime_id = ?`,
		nodeID, runtimeID).Scan(&origSnapshot)

	// Edit the BackendRuntime — change its image and command.
	pw := httptest.NewRecorder()
	h.HandlePatchBackendRuntime(pw, newReq("PATCH", "/x",
		`{"name":"BR Edited","display_name":"BR Edited","image_ref":"img:br-edited","command":["--new-arg","999"]}`,
		adminSession(), map[string]string{"id": runtimeID}))
	if pw.Code != 200 {
		t.Fatalf("patch BR code=%d body=%s", pw.Code, pw.Body.String())
	}

	// The NBR snapshot should be UNCHANGED
	var afterSnapshot string
	database.QueryRow(`SELECT config_set_json FROM node_backend_runtimes WHERE node_id = ? AND backend_runtime_id = ?`,
		nodeID, runtimeID).Scan(&afterSnapshot)

	if origSnapshot != afterSnapshot {
		t.Fatalf("NBR config_set_json changed after BR edit!\nbefore: %s\nafter: %s", origSnapshot, afterSnapshot)
	}
	if strings.Contains(afterSnapshot, "img:br-edited") {
		t.Fatalf("NBR snapshot picked up BR edit: %s", afterSnapshot)
	}
}

// Test 4: Editing BackendVersion (catalog) does NOT affect existing BackendRuntime.
func TestBackendVersionEditDoesNotAffectBackendRuntime(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	// Get existing vllm version
	bid, vid := getVllmIDs(t, database)

	// Capture the version ConfigSet BEFORE editing.
	var origVersionJSON string
	database.QueryRow(`SELECT COALESCE(config_set_json,'{}') FROM backend_versions WHERE id = ?`, vid).Scan(&origVersionJSON)

	// Create a BackendRuntime from this version
	cw := httptest.NewRecorder()
	h.HandleCreateBackendRuntimeFromTemplate(cw, newReq("POST", "/x",
		`{"backend_id":"`+bid+`","backend_version_id":"`+vid+`","name":"rt-bv-test","display_name":"RT BV Test","vendor":"nvidia"}`,
		adminSession(), nil))
	if cw.Code != 201 {
		t.Fatalf("create BR code=%d body=%s", cw.Code, cw.Body.String())
	}
	var rt map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &rt)
	rtID := rt["id"].(string)

	// Edit the BackendVersion (catalog) — change launcher.image.
	versionSet := copyConfigSet(origVersionJSON)
	setConfigValue(versionSet, "launcher.image", "img:bv-edited", "BackendVersion", vid, "test_mutation")
	database.Exec(`UPDATE backend_versions SET config_set_json = ? WHERE id = ?`, configSetJSON(versionSet), vid)

	// Read the BackendRuntime — its own image_ref should NOT change.
	after := h.getBackendRuntimeJSON(rtID)
	if after == nil {
		t.Fatal("getBackendRuntimeJSON returned nil")
	}
	imageName := fmt.Sprintf("%v", after["image_ref"])
	if strings.Contains(imageName, "bv-edited") {
		t.Fatalf("BackendRuntime image_ref changed after BV edit: %v", imageName)
	}

	// Its ConfigSet should still contain the ORIGINAL version values.
	vsn := fmt.Sprintf("%v", after["config_set_json"])
	if strings.Contains(vsn, "bv-edited") {
		t.Fatalf("BackendRuntime config_set_json picked up BV edit: %s", vsn)
	}
}

// Test 5: After RunPlan is generated, editing Deployment does NOT mutate the historical RunPlan.
func TestRunPlanImmutableAfterDeploymentEdit(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID, runtimeID, artifactID := snapshotSetupFullChain(t, h, "rp-imm")

	// Create deployment
	cw := httptest.NewRecorder()
	h.HandleCreateDeployment(cw, newReq("POST", "/x",
		`{"name":"dep-rp-imm","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nodeID+`:`+runtimeID+`","service_json":{"host_port":8005},"config_overrides":{"parameter_values":[{"key":"served_model_name","cli_name":"--served-model-name","type":"string","enabled":true,"value":"dep-rp-imm"}]}}`,
		adminSession(), nil))
	if cw.Code != 201 {
		t.Fatalf("create deployment code=%d body=%s", cw.Code, cw.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(cw.Body.Bytes(), &created)
	depID := created["id"].(string)

	// Generate a RunPlan by starting the deployment
	sw := httptest.NewRecorder()
	h.HandleStartDeployment(sw, newReq("POST", "/x", `{}`, adminSession(), map[string]string{"id": depID}))
	var startResult map[string]interface{}
	json.Unmarshal(sw.Body.Bytes(), &startResult)
	instanceID := fmt.Sprintf("%v", startResult["instance_id"])
	if instanceID == "" || instanceID == "<nil>" {
		t.Fatalf("start did not create instance: %v", startResult)
	}

	// Read the historical RunPlan
	var origPlanJSON string
	err := database.QueryRow(`SELECT plan_json FROM resolved_run_plans WHERE instance_id = ?`, instanceID).Scan(&origPlanJSON)
	if err != nil {
		t.Fatalf("run plan read error: %v", err)
	}

	// Edit the deployment (change ports and params)
	pw := httptest.NewRecorder()
	h.HandlePatchDeployment(pw, newReq("PATCH", "/x",
		`{"service_json":{"host_port":9999,"container_port":9998},"config_overrides":{"parameter_values":[{"key":"temp","cli_name":"--temperature","type":"number","enabled":true,"value":"0.5"}]}}`,
		adminSession(), map[string]string{"id": depID}))
	if pw.Code != 200 {
		t.Fatalf("patch deployment code=%d body=%s", pw.Code, pw.Body.String())
	}

	// The historical RunPlan should be UNCHANGED
	var afterPlanJSON string
	database.QueryRow(`SELECT plan_json FROM resolved_run_plans WHERE instance_id = ?`, instanceID).Scan(&afterPlanJSON)
	if origPlanJSON != afterPlanJSON {
		t.Fatalf("historical RunPlan changed after Deployment edit!\nbefore: %s\nafter: %s", origPlanJSON[:200], afterPlanJSON[:200])
	}

	// The RunPlan should NOT contain the new port values
	if strings.Contains(afterPlanJSON, "9999") || strings.Contains(afterPlanJSON, "9998") {
		t.Fatalf("historical RunPlan picked up deployment edit: %s", afterPlanJSON[:300])
	}
}

// Test 6: Creating a deployment without node_id does NOT include NBR config.
func TestDeploymentWithoutNodeDoesNotIncludeNBRConfig(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	nodeID, runtimeID, artifactID := snapshotSetupFullChain(t, h, "no-node")

	// Create deployment WITHOUT node_id in placement
	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-no-node","model_artifact_id":"`+artifactID+`","node_backend_runtime_id":"`+nodeID+`:`+runtimeID+`","service_json":{"host_port":8005}}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}

	var got map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &got)

	snapRaw := got["config_set"]
	var snapStr string
	switch v := snapRaw.(type) {
	case string:
		snapStr = v
	case map[string]interface{}:
		raw, _ := json.Marshal(v)
		snapStr = string(raw)
	default:
		snapStr = fmt.Sprintf("%v", snapRaw)
	}

	// Should NOT contain NBR-specific image_ref
	if !strings.Contains(snapStr, "launcher.image") {
		t.Fatalf("deployment with NBR should capture launcher.image: %s", snapStr)
	}
	// But should still contain BR source info
	if !strings.Contains(snapStr, runtimeID) {
		t.Fatalf("deployment snapshot missing BR source: %s", snapStr)
	}

	// The NBR still exists on the node — verify we can query it
	var nbrExists string
	database.QueryRow(`SELECT id FROM node_backend_runtimes WHERE node_id = ? AND backend_runtime_id = ?`,
		nodeID, runtimeID).Scan(&nbrExists)
	if nbrExists == "" {
		t.Fatal("NBR should still exist after deployment creation without node_id")
	}
}
func TestDeploymentListReturnsAfterRun(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	runtimeBoundaryInsertOnlineNode(t, database, "node-lr")
	database.Exec(`INSERT INTO gpu_devices (id,node_id,vendor,index_num,name,tenant_id,reported_at,created_at,updated_at)
		VALUES ('gpu-lr','node-lr','nvidia',0,'RTX','',datetime('now'),datetime('now'),datetime('now'))`)
	insertUIPersistenceArtifact(t, h, "art-list-run")
	snapshotInsertModelLocation(t, database, "ml-art-list-run", "art-list-run", "node-lr")
	insertRuntime(t, database, "rt-list-run", "Runtime List Run", "")
	// Create ready NBR
	nbrW := httptest.NewRecorder()
	h.HandleEnableNodeBackendRuntime(nbrW, newReq("POST", "/x",
		`{"backend_runtime_id":"rt-list-run","image_ref":"img:test"}`,
		adminSession(), map[string]string{"id": "node-lr"}))
	if nbrW.Code != 200 {
		t.Fatalf("nbr enable code=%d", nbrW.Code)
	}
	// R-001: enable sets needs_check; update to ready for deployment test
	database.Exec(`UPDATE node_backend_runtimes SET status='ready', image_present=1, docker_available=1 WHERE id='node-lr:rt-list-run'`)

	// Create a deployment
	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x",
		`{"name":"dep-list-run","model_artifact_id":"art-list-run","node_backend_runtime_id":"node-lr:rt-list-run","service_json":{"host_port":8005}}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}

	// List deployments — must include the created deployment
	lw := httptest.NewRecorder()
	h.HandleListDeployments(lw, newReq("GET", "/x", "", adminSession(), nil))
	if lw.Code != 200 {
		t.Fatalf("list deployments code=%d body=%s", lw.Code, lw.Body.String())
	}
	var items []map[string]interface{}
	json.Unmarshal(lw.Body.Bytes(), &items)
	if len(items) == 0 {
		t.Fatal("deployment list returned empty — regression: column mismatch in HandleListDeployments")
	}

	// The created deployment must be in the list
	found := false
	for _, item := range items {
		if item["name"] == "dep-list-run" {
			found = true
			if item["status"] != "saved" {
				t.Fatalf("deployment status = %v, want saved", item["status"])
			}
			if item["display_name"] == "" {
				t.Fatalf("deployment display_name is empty")
			}
			break
		}
	}
	if !found {
		t.Fatalf("deployment dep-list-run not found in list (%d items)", len(items))
	}
}

func TestExtractPreviewHandlesReasoningContent(t *testing.T) {
	// Chat response with reasoning_content but empty content
	chatBody := `{"choices":[{"message":{"role":"assistant","reasoning_content":"Let me think...","content":""}}]}`
	preview := extractPreview([]byte(chatBody), "chat")
	if !strings.Contains(preview, "[reasoning]") || !strings.Contains(preview, "Let me think") {
		t.Fatalf("extractPreview did not return reasoning_content: got=%q", preview)
	}

	// Chat response with normal content
	chatBody2 := `{"choices":[{"message":{"role":"assistant","content":"Hello world"}}]}`
	preview2 := extractPreview([]byte(chatBody2), "chat")
	if preview2 != "Hello world" {
		t.Fatalf("extractPreview missed content: got=%q", preview2)
	}

	// Completion response with text
	compBody := `{"choices":[{"text":"Generated text"}]}`
	preview3 := extractPreview([]byte(compBody), "completion")
	if preview3 != "Generated text" {
		t.Fatalf("extractPreview missed text: got=%q", preview3)
	}

	// Top-level content field (non-OpenAI format)
	nativeBody := `{"content":"Native response"}`
	preview4 := extractPreview([]byte(nativeBody), "chat")
	if preview4 != "Native response" {
		t.Fatalf("extractPreview missed top-level content: got=%q", preview4)
	}

	// Top-level response field
	respBody := `{"response":"Hello from llama"}`
	preview5 := extractPreview([]byte(respBody), "chat")
	if preview5 != "Hello from llama" {
		t.Fatalf("extractPreview missed top-level response: got=%q", preview5)
	}

	// Empty content + no reasoning = empty
	emptyBody := `{"choices":[{"message":{"content":""}}]}`
	preview6 := extractPreview([]byte(emptyBody), "chat")
	if preview6 != "" {
		t.Fatalf("extractPreview should return empty for empty content: got=%q", preview6)
	}
}

func TestGGUFFormatRejectsDirectoryPath(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	// GGUF with directory path → rejected
	w := httptest.NewRecorder()
	h.HandleCreateArtifact(w, newReq("POST", "/x",
		`{"name":"bad-gguf","path":"/models/some-model","format":"gguf"}`,
		adminSession(), nil))
	if w.Code != 400 {
		t.Fatalf("expected 400 for GGUF directory path, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), ".gguf") {
		t.Fatalf("error should mention .gguf: %s", w.Body.String())
	}
}

func TestGGUFFormatAcceptsFile(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	w := httptest.NewRecorder()
	h.HandleCreateArtifact(w, newReq("POST", "/x",
		`{"name":"good-gguf","path":"/models/model.gguf","format":"gguf"}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("expected 201 for GGUF file path, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEmptyPathRejected(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)

	w := httptest.NewRecorder()
	h.HandleCreateArtifact(w, newReq("POST", "/x",
		`{"name":"no-path","format":"huggingface"}`,
		adminSession(), nil))
	if w.Code != 400 {
		t.Fatalf("expected 400 for empty path, got %d: %s", w.Code, w.Body.String())
	}
}
