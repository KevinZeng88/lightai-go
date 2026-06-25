package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTenantA_CannotAccessTenantB_Node(t *testing.T) {
	db := setupTestDB(t)
	h := NewAgentHandler(db, nil)
	db.Exec(`INSERT INTO tenants (id,slug,name,status,created_at,updated_at) VALUES ('tA','tenant-a','TA','active',datetime('now'),datetime('now'))`)
	db.Exec(`INSERT INTO tenants (id,slug,name,status,created_at,updated_at) VALUES ('tB','tenant-b','TB','active',datetime('now'),datetime('now'))`)
	db.Exec(`INSERT INTO nodes (id,agent_id,hostname,primary_ip,status,tenant_id,last_heartbeat_at,created_at,updated_at) VALUES ('nA','a1','ha','10.0.0.1','online','tA',datetime('now'),datetime('now'),datetime('now'))`)
	db.Exec(`INSERT INTO nodes (id,agent_id,hostname,primary_ip,status,tenant_id,last_heartbeat_at,created_at,updated_at) VALUES ('nB','a2','hb','10.0.0.2','online','tB',datetime('now'),datetime('now'),datetime('now'))`)
	w := httptest.NewRecorder()
	h.HandleListNodes(w, newReq("GET", "/x", "", tenantSession("tA"), nil))
	if w.Code != http.StatusOK { t.Fatalf("list code=%d", w.Code) }
	var nodes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &nodes)
	for _, n := range nodes {
		if n["id"] == "nB" { t.Fatal("tenant A saw tenant B node") }
	}
}
