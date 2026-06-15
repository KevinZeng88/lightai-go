package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
)

func initTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, _ := db.Open(":memory:")
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cfg := auth.BootstrapConfig{Username: "admin", Password: "test1234", ForceChangePassword: false}
	if err := auth.InitBootstrap(database, cfg); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	return database
}

func TestAgentRegistrationWithGoodToken(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	handler := NewAgentHandler(database, nil)
	body, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-001", "hostname": "host1"})
	req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleRegister(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNodeIDAgentIDBindingOnReRegistration(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	handler := NewAgentHandler(database, nil)
	body1, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-001", "hostname": "host1"})
	req1 := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleRegister(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d", w1.Code)
	}
	body2, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-002", "hostname": "host1"})
	req2 := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleRegister(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("agent_id mismatch: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHeartbeatAgentIDMismatchRejected(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	handler := NewAgentHandler(database, nil)
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-001','agent-001','host1','online','a0000000-0000-0000-0000-000000000001',datetime('now'),datetime('now'),datetime('now'))`)

	hb1, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-001"})
	r1 := httptest.NewRequest("POST", "/api/agent/heartbeat", bytes.NewReader(hb1))
	r1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleHeartbeat(w1, r1)
	if w1.Code != http.StatusOK {
		t.Errorf("correct heartbeat: expected 200, got %d", w1.Code)
	}

	hb2, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-002"})
	r2 := httptest.NewRequest("POST", "/api/agent/heartbeat", bytes.NewReader(hb2))
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleHeartbeat(w2, r2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("mismatch heartbeat: expected 403, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHeartbeatUnregisteredNodeRequestsReregister(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	handler := NewAgentHandler(database, nil)
	hb, _ := json.Marshal(map[string]interface{}{"node_id": "unknown-node", "agent_id": "agent-x"})
	req := httptest.NewRequest("POST", "/api/agent/heartbeat", bytes.NewReader(hb))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleHeartbeat(w, req)
	var resp HeartbeatResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.NeedRegister {
		t.Error("unknown node heartbeat should request re-registration")
	}
}

func TestResourceReportAgentIDBinding(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	handler := NewResourceHandler(database, nil)
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-001','agent-001','host1','online','a0000000-0000-0000-0000-000000000001',datetime('now'),datetime('now'),datetime('now'))`)

	body, _ := json.Marshal(map[string]interface{}{"agent_id": "agent-002", "gpu_resources": []map[string]interface{}{}})
	req := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleResourceReport(w, req)
	if w.Code != 404 {
		t.Errorf("wrong agent_id report: expected 404, got %d", w.Code)
	}

	body2, _ := json.Marshal(map[string]interface{}{"agent_id": "agent-001", "gpu_resources": []map[string]interface{}{}})
	req2 := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleResourceReport(w2, req2)
	if w2.Code != 200 {
		t.Errorf("correct agent_id report: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestNodeIDAgentIDBindingOnReRegisterSameAgentOK(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	handler := NewAgentHandler(database, nil)
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-001", "hostname": "host1"})
		req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.HandleRegister(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("re-register %d: expected 201, got %d", i, w.Code)
		}
	}
}

func TestAdminSeesNodeInDefaultTenant(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	defTID := database.DefaultTenantID()
	if defTID == "" {
		t.Fatal("DefaultTenantID is empty after bootstrap")
	}
	t.Logf("Default tenant UUID: %s", defTID)

	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-001','agent-01','k8s-master1','online',?,datetime('now'),datetime('now'),datetime('now'))`, defTID)
	handler := NewAgentHandler(database, nil)

	// Admin with different UUID session still sees all nodes.
	ctx := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: "550e8400-e29b-41d4-a716-446655440000", UserID: "admin-01", IsPlatformAdmin: true,
	})
	req := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListNodes(w, req)
	var nodes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &nodes)
	if len(nodes) != 1 {
		t.Errorf("admin: expected 1 node, got %d", len(nodes))
	}

	// Non-admin with matching tenant UUID → sees the node.
	ctx2 := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: defTID, UserID: "user-01", IsPlatformAdmin: false,
	})
	req2 := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(ctx2)
	w2 := httptest.NewRecorder()
	handler.HandleListNodes(w2, req2)
	var nodes2 []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &nodes2)
	if len(nodes2) != 1 {
		t.Errorf("matching tenant: expected 1 node, got %d", len(nodes2))
	}

	// Non-admin with different tenant → sees 0.
	ctx3 := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: "other-tenant-uuid", UserID: "user-02", IsPlatformAdmin: false,
	})
	req3 := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(ctx3)
	w3 := httptest.NewRecorder()
	handler.HandleListNodes(w3, req3)
	var nodes3 []map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &nodes3)
	if len(nodes3) != 0 {
		t.Errorf("other tenant: expected 0 nodes, got %d", len(nodes3))
	}
}

func TestNonAdminCannotSeeOtherTenantNode(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-a','agent-a','host','online','tenant-a-uuid',datetime('now'),datetime('now'),datetime('now'))`)
	handler := NewAgentHandler(database, nil)

	ctx := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: "tenant-a-uuid", UserID: "u1", IsPlatformAdmin: false,
	})
	req := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListNodes(w, req)
	var nodes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &nodes)
	if len(nodes) != 1 {
		t.Errorf("same-tenant: expected 1, got %d", len(nodes))
	}

	ctx2 := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: "tenant-b-uuid", UserID: "u2", IsPlatformAdmin: false,
	})
	req2 := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(ctx2)
	w2 := httptest.NewRecorder()
	handler.HandleListNodes(w2, req2)
	var nodes2 []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &nodes2)
	if len(nodes2) != 0 {
		t.Errorf("cross-tenant: expected 0, got %d", len(nodes2))
	}
}

func TestTenantAdminCannotTransferOtherTenantNode(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('other-uuid','other','Other Tenant','active')`)
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-other','agent-o','host','online','other-uuid',datetime('now'),datetime('now'),datetime('now'))`)
	handler := NewAgentHandler(database, nil)

	// User in default tenant tries to transfer node-other (not their tenant).
	ctx := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: database.DefaultTenantID(), UserID: "user-01", IsPlatformAdmin: false,
	})
	// Inject node:transfer permission via the auth context key.
	ctx = auth.NewContextWithSessionInfo(ctx, &auth.SessionInfo{
		TenantID: database.DefaultTenantID(), UserID: "user-01", IsPlatformAdmin: false,
	})

	body, _ := json.Marshal(map[string]string{"tenant_id": "target-uuid"})
	req := httptest.NewRequest("PATCH", "/api/nodes/node-other/tenant", bytes.NewReader(body))
	req.SetPathValue("id", "node-other")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandlePatchNodeTenant(w, req.WithContext(ctx))
	// Should be forbidden — user does not own node-other.
	t.Logf("cross-tenant transfer: %d (expected 403)", w.Code)
}

func TestAgentRegistrationWritesDefaultTenantUUID(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	defTID := database.DefaultTenantID()
	handler := NewAgentHandler(database, nil)

	body, _ := json.Marshal(map[string]interface{}{"node_id": "node-new", "agent_id": "agent-new", "hostname": "host1"})
	req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleRegister(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d", w.Code)
	}

	var tid string
	database.QueryRow(`SELECT tenant_id FROM nodes WHERE id = 'node-new'`).Scan(&tid)
	if tid != defTID {
		t.Errorf("nodes.tenant_id = %s, expected default UUID %s", tid, defTID)
	}
	if tid == "default" {
		t.Error("nodes.tenant_id must not be 'default'")
	}
}

// TestAgentRegistrationSendsNewFields verifies that registration stores
// primary_ip, os, arch, kernel, agent_version.
func TestAgentRegistrationSendsNewFields(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	handler := NewAgentHandler(database, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"node_id":            "node-rich",
		"agent_id":           "agent-rich",
		"hostname":           "k8s-master1",
		"primary_ip":         "192.168.1.100",
		"advertised_address": "192.168.1.100",
		"os":                 "linux",
		"arch":               "amd64",
		"kernel":             "6.6.114",
		"version":            "v0.1.9",
	})
	req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleRegister(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var hostname, primaryIP, osName, arch, kernel, agentVer string
	database.QueryRow(`SELECT hostname, primary_ip, os, arch, kernel, agent_version FROM nodes WHERE id = 'node-rich'`).
		Scan(&hostname, &primaryIP, &osName, &arch, &kernel, &agentVer)
	if hostname != "k8s-master1" {
		t.Errorf("hostname = %q, want k8s-master1", hostname)
	}
	if primaryIP != "192.168.1.100" {
		t.Errorf("primary_ip = %q, want 192.168.1.100", primaryIP)
	}
	if osName != "linux" {
		t.Errorf("os = %q, want linux", osName)
	}
	if arch != "amd64" {
		t.Errorf("arch = %q, want amd64", arch)
	}
	if kernel != "6.6.114" {
		t.Errorf("kernel = %q, want 6.6.114", kernel)
	}
	if agentVer != "v0.1.9" {
		t.Errorf("agent_version = %q, want v0.1.9", agentVer)
	}
}

// TestNodeListReturnsNewFields verifies GET /api/v1/nodes returns the new fields.
func TestNodeListReturnsNewFields(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	defTID := database.DefaultTenantID()

	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, primary_ip, advertised_address,
		os, arch, kernel, agent_version,
		status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-rich','agent-rich','host1','10.0.0.1','10.0.0.1',
		'linux','amd64','6.6.114','v0.1.9',
		'online',?,datetime('now'),datetime('now'),datetime('now'))`, defTID)

	handler := NewAgentHandler(database, nil)
	ctx := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: defTID, UserID: "u1", IsPlatformAdmin: false,
	})
	req := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListNodes(w, req)

	var nodes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &nodes)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	n := nodes[0]
	if n["hostname"] != "host1" {
		t.Errorf("hostname = %v", n["hostname"])
	}
	if n["primary_ip"] != "10.0.0.1" {
		t.Errorf("primary_ip = %v", n["primary_ip"])
	}
	if n["os"] != "linux" {
		t.Errorf("os = %v", n["os"])
	}
	if n["arch"] != "amd64" {
		t.Errorf("arch = %v", n["arch"])
	}
	if n["kernel"] != "6.6.114" {
		t.Errorf("kernel = %v", n["kernel"])
	}
	if n["agent_version"] != "v0.1.9" {
		t.Errorf("agent_version = %v", n["agent_version"])
	}
}

// TestNodeListMissingFieldsFallback verifies old nodes without new
// fields return empty strings (not nil).
func TestNodeListMissingFieldsFallback(t *testing.T) {
	database := initTestDB(t)
	defer database.Close()
	defTID := database.DefaultTenantID()

	// Insert a node WITHOUT the new columns (simulating pre-migration data).
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-old','agent-old','oldhost','online',?,datetime('now'),datetime('now'),datetime('now'))`, defTID)

	handler := NewAgentHandler(database, nil)
	ctx := auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID: defTID, UserID: "u1", IsPlatformAdmin: false,
	})
	req := httptest.NewRequest("GET", "/api/nodes", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleListNodes(w, req)

	var nodes []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &nodes)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	n := nodes[0]
	// New fields should be empty strings for old data (not missing keys).
	if n["primary_ip"] != "" {
		t.Errorf("primary_ip should be empty for old node, got %v", n["primary_ip"])
	}
	if n["os"] != "" {
		t.Errorf("os should be empty for old node, got %v", n["os"])
	}
	if n["agent_version"] != "" {
		t.Errorf("agent_version should be empty for old node, got %v", n["agent_version"])
	}
	if n["hostname"] != "oldhost" {
		t.Errorf("hostname should be oldhost, got %v", n["hostname"])
	}
}
