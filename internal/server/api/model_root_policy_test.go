package api

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateNodeModelRootRejectsDeniedAndTraversal(t *testing.T) {
	tmp := t.TempDir()
	etcLink := filepath.Join(tmp, "etc-link")
	if err := os.Symlink("/etc", etcLink); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	for _, path := range []string{"/", "/etc", "/etc/lightai", "/root", "/proc", "/sys", "/dev", "/run", "/boot", "/tmp/../etc", etcLink} {
		if _, err := normalizeAllowedModelRoot(path); err == nil {
			t.Fatalf("expected %s to be rejected", path)
		}
	}
}

func TestNodeModelRootCRUDUsesPersistentRows(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	tenantID := database.DefaultTenantID()
	_, err := database.Exec(`INSERT INTO nodes (id, agent_id, hostname, primary_ip, status, tenant_id, created_at, updated_at)
		VALUES ('node-root-1','agent-root-1','node-root-1','127.0.0.1','online',?,'2024-01-01','2024-01-01')`, tenantID)
	if err != nil {
		t.Fatalf("insert node: %v", err)
	}

	w0 := httptest.NewRecorder()
	h.HandleListNodeModelRoots(w0, newReq("GET", "/api/v1/nodes/node-root-1/model-roots", "", adminSession(), map[string]string{"id": "node-root-1"}))
	if w0.Code != 200 {
		t.Fatalf("list code=%d body=%s", w0.Code, w0.Body.String())
	}
	var initial []map[string]interface{}
	if err := json.Unmarshal(w0.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode initial: %v", err)
	}
	if len(initial) != 0 {
		t.Fatalf("default roots should be empty, got %v", initial)
	}

	wBad := httptest.NewRecorder()
	h.HandleAddNodeModelRoot(wBad, newReq("POST", "/api/v1/nodes/node-root-1/model-roots", `{"path":"/"}`, adminSession(), map[string]string{"id": "node-root-1"}))
	if wBad.Code != 400 || !strings.Contains(wBad.Body.String(), "not allowed") {
		t.Fatalf("denied root code=%d body=%s", wBad.Code, wBad.Body.String())
	}

	allowed := filepath.Join(t.TempDir(), "models")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatalf("mkdir allowed: %v", err)
	}
	wAdd := httptest.NewRecorder()
	h.HandleAddNodeModelRoot(wAdd, newReq("POST", "/api/v1/nodes/node-root-1/model-roots", `{"path":"`+allowed+`","description":"test models"}`, adminSession(), map[string]string{"id": "node-root-1"}))
	if wAdd.Code != 201 {
		t.Fatalf("add code=%d body=%s", wAdd.Code, wAdd.Body.String())
	}
	var root map[string]interface{}
	if err := json.Unmarshal(wAdd.Body.Bytes(), &root); err != nil {
		t.Fatalf("decode add: %v", err)
	}
	rootID, _ := root["id"].(string)
	if rootID == "" || root["path"] != allowed || root["status"] != "enabled" {
		t.Fatalf("unexpected root: %#v", root)
	}

	wList := httptest.NewRecorder()
	h.HandleListNodeModelRoots(wList, newReq("GET", "/api/v1/nodes/node-root-1/model-roots", "", adminSession(), map[string]string{"id": "node-root-1"}))
	var list []map[string]interface{}
	_ = json.Unmarshal(wList.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["id"] != rootID {
		t.Fatalf("persistent root not listed: %#v", list)
	}
}
