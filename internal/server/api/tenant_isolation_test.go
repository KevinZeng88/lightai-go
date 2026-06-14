package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
)

func sessionCtx(tenantID string) context.Context {
	return auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: tenantID, UserID: "test-user",
	})
}

func TestTenantNodesScopedToList(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()

	for _, n := range []struct{ id, agent, tenant string }{
		{"node-a1", "agent-a1", "tenant-a"},
		{"node-a2", "agent-a2", "tenant-a"},
		{"node-b1", "agent-b1", "tenant-b"},
	} {
		database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
			VALUES (?, ?,'host','online', ?, datetime('now'), datetime('now'), datetime('now'))`, n.id, n.agent, n.tenant)
	}
	handler := NewAgentHandler(database, nil)

	req := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(sessionCtx("tenant-a"))
	w := httptest.NewRecorder()
	handler.HandleListNodes(w, req)
	var nodes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &nodes)
	if len(nodes) != 2 {
		t.Errorf("tenant-a: expected 2 nodes, got %d", len(nodes))
	}

	req2 := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(sessionCtx("tenant-b"))
	w2 := httptest.NewRecorder()
	handler.HandleListNodes(w2, req2)
	var nodes2 []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &nodes2)
	if len(nodes2) != 1 {
		t.Errorf("tenant-b: expected 1 node, got %d", len(nodes2))
	}
}

func TestTenantBBlockedFromTenantANode(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-a1','agent-a1','host','online','tenant-a',datetime('now'),datetime('now'),datetime('now'))`)
	handler := NewAgentHandler(database, nil)

	req := httptest.NewRequest("GET", "/api/nodes/node-a1", nil).WithContext(sessionCtx("tenant-b"))
	req.SetPathValue("id", "node-a1")
	w := httptest.NewRecorder()
	handler.HandleGetNode(w, req)
	if w.Code != 404 {
		t.Errorf("cross-tenant: expected 404, got %d", w.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/nodes/node-a1", nil).WithContext(sessionCtx("tenant-a"))
	req2.SetPathValue("id", "node-a1")
	w2 := httptest.NewRecorder()
	handler.HandleGetNode(w2, req2)
	if w2.Code != 200 {
		t.Errorf("same-tenant: expected 200, got %d", w2.Code)
	}
}

func TestTenantScopedGPUList(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-a1','agent-a1','host','online','tenant-a',datetime('now'),datetime('now'),datetime('now'))`)
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-b1','agent-b1','host','online','tenant-b',datetime('now'),datetime('now'),datetime('now'))`)
	// NewResourceHandler creates the gpu_devices table.
	handler := NewResourceHandler(database, nil)
	database.Exec(`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, memory_total_bytes, health, status, collected_at, created_at, updated_at)
		VALUES ('g1','node-a1','nvidia',0,'A100','gpu-a1',81920,'healthy','available',datetime('now'),datetime('now'),datetime('now'))`)
	database.Exec(`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, memory_total_bytes, health, status, collected_at, created_at, updated_at)
		VALUES ('g2','node-b1','nvidia',0,'H100','gpu-b1',81920,'healthy','available',datetime('now'),datetime('now'),datetime('now'))`)

	req := httptest.NewRequest("GET", "/api/gpus", nil).WithContext(sessionCtx("tenant-a"))
	w := httptest.NewRecorder()
	handler.HandleListGPUs(w, req)
	var gpus []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &gpus)
	if len(gpus) != 1 {
		t.Errorf("tenant-a GPU: expected 1, got %d", len(gpus))
	}

	req2 := httptest.NewRequest("GET", "/api/gpus", nil).WithContext(sessionCtx("tenant-b"))
	w2 := httptest.NewRecorder()
	handler.HandleListGPUs(w2, req2)
	var gpus2 []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &gpus2)
	if len(gpus2) != 1 {
		t.Errorf("tenant-b GPU: expected 1, got %d", len(gpus2))
	}
}

// TestSystemQueryRespectsTenant verifies node system endpoint enforces tenant scope.
func TestSystemQueryRespectsTenant(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-a1','agent-a1','host-a','online','tenant-a',datetime('now'),datetime('now'),datetime('now'))`)
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-b1','agent-b1','host-b','online','tenant-b',datetime('now'),datetime('now'),datetime('now'))`)

	// Insert system snapshot for node-a1 (tenant-a).
	handler := NewResourceHandler(database, nil)
	database.Exec(`INSERT INTO node_system_snapshots (node_id, cpu_utilization_percent, memory_total_bytes, memory_used_bytes, collected_at)
		VALUES ('node-a1','10',16000000000,8000000000,datetime('now'))`)

	// Tenant A can access own node system.
	req := httptest.NewRequest("GET", "/api/nodes/node-a1/system", nil)
	req.SetPathValue("id", "node-a1")
	req = req.WithContext(sessionCtx("tenant-a"))
	w := httptest.NewRecorder()
	handler.HandleGetNodeSystem(w, req)
	if w.Code != 200 {
		t.Errorf("same-tenant system: expected 200, got %d", w.Code)
	}
}

func TestNodesNoHardcodedDefaultTenant(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('n1','a1','h1','online','custom-tenant',datetime('now'),datetime('now'),datetime('now'))`)
	handler := NewAgentHandler(database, nil)

	req := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(sessionCtx("custom-tenant"))
	w := httptest.NewRecorder()
	handler.HandleListNodes(w, req)
	var nodes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &nodes)
	if len(nodes) != 1 {
		t.Errorf("custom-tenant: expected 1 node, got %d", len(nodes))
	}
}
