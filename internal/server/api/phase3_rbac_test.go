package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func newReq(method, path, body string, sess *auth.SessionInfo, pv map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if sess != nil {
		req = req.WithContext(auth.NewContextWithSessionInfo(req.Context(), sess))
	}
	for k, v := range pv {
		req.SetPathValue(k, v)
	}
	return req
}

func getVllmIDs(t *testing.T, db *db.DB) (backendID, versionID string) {
	t.Helper()
	if err := db.QueryRow("SELECT id FROM inference_backends WHERE name='vllm'").Scan(&backendID); err != nil {
		t.Fatalf("vllm backend: %v", err)
	}
	if err := db.QueryRow("SELECT id FROM backend_versions WHERE backend_id=(SELECT id FROM inference_backends WHERE name='vllm') AND is_default=1").Scan(&versionID); err != nil {
		t.Fatalf("vllm default version: %v", err)
	}
	return
}

func insertRuntime(t *testing.T, db *db.DB, id, name, tenantID string) {
	t.Helper()
	bid, vid := getVllmIDs(t, db)
	var versionConfigSetRaw string
	if err := db.QueryRow(`SELECT config_set_json FROM backend_versions WHERE id = ?`, vid).Scan(&versionConfigSetRaw); err != nil {
		t.Fatalf("read version config set: %v", err)
	}
	configSet := copyConfigSet(versionConfigSetRaw)
	setConfigValue(configSet, "launcher.kind", "docker", "BackendRuntime", id, "test_fixture")
	setConfigValue(configSet, "launcher.image", "img:test", "BackendRuntime", id, "test_fixture")
	setConfigValue(configSet, "launcher.docker_options", map[string]interface{}{"privileged": true}, "BackendRuntime", id, "test_fixture")
	setConfigValue(configSet, "runtime.env", map[string]interface{}{"HF_TOKEN": "secret123", "PUBLIC_VAR": "visible", "API_KEY": "abc"}, "BackendRuntime", id, "test_fixture")
	setConfigValue(configSet, "runtime.model_mount", map[string]interface{}{"container_path": "/models", "readonly": true}, "BackendRuntime", id, "test_fixture")
	sourceMeta := jsonString(map[string]interface{}{"source_type": "test_fixture", "source_backend_version_id": vid})
	_, err := db.Exec(`INSERT INTO backend_runtimes
		(id,name,display_name,backend_id,backend_version_id,source_template_name,vendor,runtime_type,is_builtin,is_editable,tenant_id,slug,managed_by,source,catalog_version,checksum,status,config_set_json,source_metadata_json,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, name, bid, vid, "", "nvidia", "docker", 0, 1, tenantID, slugify(name), "user", "test-fixture", "test", id+"-checksum", "active", configSetJSON(configSet), sourceMeta, "2024-01-01", "2024-01-01")
	if err != nil {
		t.Fatalf("insert runtime: %v", err)
	}
}

func insertNodeBackendRuntime(t *testing.T, db *db.DB, id, runtimeID, nodeID, imageRef, status, reason string, imagePresent, dockerAvailable int, tenantID string) {
	t.Helper()
	var configSetRaw, sourceMetaRaw string
	if err := db.QueryRow(`SELECT config_set_json, source_metadata_json FROM backend_runtimes WHERE id = ?`, runtimeID).Scan(&configSetRaw, &sourceMetaRaw); err != nil {
		t.Fatalf("read runtime config set: %v", err)
	}
	set := copyConfigSet(configSetRaw)
	if imageRef != "" {
		setConfigValue(set, "launcher.image", imageRef, "NodeBackendRuntime", id, "test_fixture")
	}
	meta := configSourceMetadata(sourceMetaRaw)
	meta["source_type"] = "node_backend_runtime_fixture"
	meta["source_backend_runtime_id"] = runtimeID
	_, err := db.Exec(`INSERT INTO node_backend_runtimes
		(id,backend_runtime_id,node_id,runner_type,image_ref,image_present,docker_available,config_set_json,source_metadata_json,status,status_reason,tenant_id,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,datetime('now'),datetime('now'))`,
		id, runtimeID, nodeID, "docker", imageRef, imagePresent, dockerAvailable, configSetJSON(set), jsonString(meta), status, reason, tenantID)
	if err != nil {
		t.Fatalf("insert nbr: %v", err)
	}
}

func adminSession() *auth.SessionInfo {
	return &auth.SessionInfo{UserID: "admin", IsPlatformAdmin: true}
}
func tenantSession(tid string) *auth.SessionInfo {
	return &auth.SessionInfo{UserID: "user-" + tid, TenantID: tid}
}

func TestBackendListReadOnly(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	w := httptest.NewRecorder()
	h.HandleListBackends(w, newReq("GET", "/x", "", adminSession(), nil))
	if w.Code != 200 {
		t.Fatalf("code=%d", w.Code)
	}
	var list []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 4 {
		t.Errorf("got %d backends, want 4", len(list))
	}
}

func TestBackendVersionList(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	bid, _ := getVllmIDs(t, db)
	w := httptest.NewRecorder()
	h.HandleListBackendVersions(w, newReq("GET", "/x/versions", "", adminSession(), map[string]string{"id": bid}))
	if w.Code != 200 {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var list []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) < 1 {
		t.Errorf("got %d versions, want at least 1", len(list))
	}
	foundTarget := false
	for _, item := range list {
		if item["id"] == "vllm-v0.23.0" {
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Errorf("official vLLM BackendVersion missing from list: %v", list)
	}
}

func TestBackendRuntimeCRUD(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	insertRuntime(t, db, "t1", "test-rt", "")

	w := httptest.NewRecorder()
	h.HandlePatchBackendRuntime(w, newReq("PATCH", "/x", `{"image_ref":"new:v2"}`, adminSession(), map[string]string{"id": "t1"}))
	if w.Code != 200 {
		t.Fatalf("PATCH code=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	h.HandleGetBackendRuntime(w2, newReq("GET", "/x", "", adminSession(), map[string]string{"id": "t1"}))
	var rt map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &rt)
	if rt["image_ref"] != "new:v2" {
		t.Errorf("image_ref: %v", rt["image_ref"])
	}
	if rt["vendor"] != "nvidia" {
		t.Errorf("vendor cleared: %v", rt["vendor"])
	}

	w3 := httptest.NewRecorder()
	h.HandleDeleteBackendRuntime(w3, newReq("DELETE", "/x", "", adminSession(), map[string]string{"id": "t1"}))
	if w3.Code != 200 {
		t.Errorf("DELETE code=%d", w3.Code)
	}
}

func TestBackendRuntimeTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	insertRuntime(t, db, "rta", "RT-A", "tenant-a")

	w := httptest.NewRecorder()
	h.HandleGetBackendRuntime(w, newReq("GET", "/x", "", tenantSession("tenant-b"), map[string]string{"id": "rta"}))
	if w.Code != 404 {
		t.Errorf("cross-tenant should 404, got %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	h.HandleGetBackendRuntime(w2, newReq("GET", "/x", "", tenantSession("tenant-a"), map[string]string{"id": "rta"}))
	if w2.Code != 200 {
		t.Errorf("own tenant should 200, got %d", w2.Code)
	}

	w3 := httptest.NewRecorder()
	h.HandleGetBackendRuntime(w3, newReq("GET", "/x", "", adminSession(), map[string]string{"id": "rta"}))
	if w3.Code != 200 {
		t.Errorf("admin should 200, got %d", w3.Code)
	}
}

func TestDefaultEnvJSONRedaction(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	insertRuntime(t, db, "rr", "RedactTest", "")

	w := httptest.NewRecorder()
	h.HandleGetBackendRuntime(w, newReq("GET", "/x", "", adminSession(), map[string]string{"id": "rr"}))
	if w.Code != 200 {
		t.Fatalf("GET code=%d", w.Code)
	}
	var rt map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rt)

	envRaw := rt["env"]
	envBytes, _ := json.Marshal(envRaw)
	envStr := string(envBytes)

	if !strings.Contains(envStr, "redacted") {
		t.Errorf("sensitive not redacted: %s", envStr)
	}
	if !strings.Contains(envStr, "visible") {
		t.Errorf("PUBLIC_VAR not visible: %s", envStr)
	}
	if strings.Contains(envStr, "secret123") {
		t.Errorf("HF_TOKEN leaked: %s", envStr)
	}
	if strings.Contains(envStr, "abc") {
		t.Errorf("API_KEY leaked: %s", envStr)
	}
}

func TestCreateFromTemplate(t *testing.T) {
	// Override osReadFile to resolve config path from project root.
	origReadFile := osReadFile
	osReadFile = func(name string) ([]byte, error) {
		return os.ReadFile("../../../" + name)
	}
	defer func() { osReadFile = origReadFile }()

	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	bid, vid := getVllmIDs(t, db)

	w := httptest.NewRecorder()
	h.HandleCreateBackendRuntimeFromTemplate(w, newReq("POST", "/x",
		`{"template_name":"vllm-nvidia-docker","name":"my-vllm","display_name":"My vLLM","vendor":"nvidia","backend_id":"`+bid+`","backend_version_id":"`+vid+`"}`,
		adminSession(), nil))
	if w.Code != 201 {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var rt map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rt)
	if rt["name"] != "my-vllm" {
		t.Errorf("name: %v", rt["name"])
	}
	if rt["vendor"] != "nvidia" {
		t.Errorf("vendor: %v", rt["vendor"])
	}
	if rt["source_template_name"] != "vllm-nvidia-docker" {
		t.Errorf("source_template: %v", rt["source_template_name"])
	}
}
