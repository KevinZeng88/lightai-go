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
	_, err := db.Exec("INSERT INTO backend_runtimes (id,name,display_name,backend_id,backend_version_id,source_template_name,vendor,runtime_type,image_name,image_pull_policy,entrypoint_override_json,args_override_json,default_env_json,docker_json,model_mount_json,health_check_override_json,is_builtin,is_editable,tenant_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		id, name, name, bid, vid, "", "nvidia", "docker", "img:test", "never",
		"[]", "[]", "{\"HF_TOKEN\":\"secret123\",\"PUBLIC_VAR\":\"visible\",\"API_KEY\":\"abc\"}", "{\"privileged\":true}", "{}", "{}", 0, 1, tenantID, "2024-01-01", "2024-01-01")
	if err != nil {
		t.Fatalf("insert runtime: %v", err)
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
	h.HandlePatchBackendRuntime(w, newReq("PATCH", "/x", `{"image_name":"new:v2"}`, adminSession(), map[string]string{"id": "t1"}))
	if w.Code != 200 {
		t.Fatalf("PATCH code=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	h.HandleGetBackendRuntime(w2, newReq("GET", "/x", "", adminSession(), map[string]string{"id": "t1"}))
	var rt map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &rt)
	if rt["image_name"] != "new:v2" {
		t.Errorf("image_name: %v", rt["image_name"])
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

	envRaw := rt["default_env_json"]
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

	w := httptest.NewRecorder()
	h.HandleCreateBackendRuntimeFromTemplate(w, newReq("POST", "/x",
		`{"template_name":"vllm-nvidia-docker","name":"my-vllm","display_name":"My vLLM","vendor":"nvidia","backend_name":"vllm","backend_version":"0.8.5"}`,
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
