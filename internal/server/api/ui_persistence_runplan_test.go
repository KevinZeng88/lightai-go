package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func insertUIPersistenceArtifact(t *testing.T, h *AgentHandler, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := h.DB.Exec(`INSERT INTO model_artifacts
		(id,name,display_name,source_type,path,format,task_type,architecture,size_label,quantization,tenant_id,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, id, "Display "+id, "local_path", "/models/"+id+".gguf", "gguf", "chat", "custom", "", "unknown", "", now, now); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}
}

func insertUIPersistenceDeployment(t *testing.T, h *AgentHandler, id, artifactID, runtimeID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := h.DB.Exec(`INSERT INTO model_deployments
		(id,name,display_name,model_artifact_id,backend_runtime_id,replicas,placement_json,service_json,parameters_json,env_overrides_json,desired_state,status,tenant_id,created_at,updated_at)
		VALUES (?,?,?,?,?,1,'{"node_id":"node-a","gpu_ids":[]}','{"host_port":8005}','{}','{}','stopped','saved','',?,?)`,
		id, id, id, artifactID, runtimeID, now, now); err != nil {
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
	if !strings.Contains(got["name"].(string), "-copy") {
		t.Fatalf("clone name does not show copy suffix: %v", got["name"])
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
	h.HandlePatchBackendRuntime(pw, newReq("PATCH", "/x", `{"name":"Runtime Renamed","display_name":"Runtime Display","args_override_json":["--ctx-size","4096"]}`, adminSession(), map[string]string{"id": got["id"].(string)}))
	if pw.Code != http.StatusOK {
		t.Fatalf("patch code=%d body=%s", pw.Code, pw.Body.String())
	}
	after := h.getBackendRuntimeJSON(got["id"].(string))
	if after["name"] != "Runtime Renamed" || after["display_name"] != "Runtime Display" {
		t.Fatalf("patched names not persisted: %#v", after)
	}
	raw, _ := json.Marshal(after["args_override_json"])
	if !strings.Contains(string(raw), "--ctx-size") {
		t.Fatalf("args_override_json not persisted: %s", raw)
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
	if got["path"] != "/models/qwen-file.gguf" {
		t.Fatalf("path changed after display_name edit: %v", got["path"])
	}
}

func TestDeploymentSaveOnlyAndPatchEditableFields(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertUIPersistenceArtifact(t, h, "art-save")
	insertRuntime(t, database, "rt-save", "Runtime Save", "")

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x", `{"name":"dep-save","model_artifact_id":"art-save","backend_runtime_id":"rt-save","placement_json":{"node_id":"node-a","gpu_ids":[]},"service_json":{"host_port":8005,"container_port":8080},"parameters_json":{"served_model_name":"served-a"}}`, adminSession(), nil))
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
	h.HandlePatchDeployment(pw, newReq("PATCH", "/x", `{"display_name":"Dep Display","placement_json":{"node_id":"node-b","gpu_ids":["gpu-b"]},"service_json":{"host_port":8006,"container_port":8081},"parameters_json":{"served_model_name":"served-b"}}`, adminSession(), map[string]string{"id": got["id"].(string)}))
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
	h.HandleEnableNodeBackendRuntime(w, newReq("POST", "/x", `{"backend_runtime_id":"rt-nbr-name","display_name":"Runtime NBR Name - Custom Node","image_ref":"img:test","image_present":true,"docker_available":true}`, adminSession(), map[string]string{"id": "node-nbr-name"}))
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


func TestDeploymentCapturesConfigSnapshotAtCreate(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertUIPersistenceArtifact(t, h, "art-snap")
	insertRuntime(t, database, "rt-snap", "llama.cpp NVIDIA CUDA Runtime", "")

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x", `{"name":"dep-snap","model_artifact_id":"art-snap","backend_runtime_id":"rt-snap","placement_json":{"node_id":"node-a","gpu_ids":[]},"service_json":{"host_port":8005,"container_port":8080}}`, adminSession(), nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("create deployment code=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	snapRaw := got["config_snapshot_json"]
	if snapRaw == nil {
		t.Fatal("config_snapshot_json missing in response")
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
		t.Fatalf("config_snapshot_json empty after create: %s", snapStr)
	}
	if !strings.Contains(snapStr, "rt-snap") {
		t.Fatalf("snapshot missing source_runtime_id: %s", snapStr)
	}
	// Verify deployment detail also returns snapshot
	detail := h.getDeploymentJSON(got["id"].(string))
	if detail == nil {
		t.Fatal("getDeploymentJSON returned nil")
	}
	if snap2, _ := detail["config_snapshot_json"]; snap2 == nil || fmt.Sprintf("%v", snap2) == "{}" {
		t.Fatalf("deployment detail missing snapshot: %v", snap2)
	}
}

func TestDeploymentPatchPortsAndDisplayName(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	insertUIPersistenceArtifact(t, h, "art-edit")
	insertRuntime(t, database, "rt-edit", "Runtime Edit", "")

	w := httptest.NewRecorder()
	h.HandleCreateDeployment(w, newReq("POST", "/x", `{"name":"dep-edit","model_artifact_id":"art-edit","backend_runtime_id":"rt-edit","placement_json":{"node_id":"node-a","gpu_ids":[]},"service_json":{"host_port":8005,"container_port":8080},"parameters_json":{}}`, adminSession(), nil))
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
