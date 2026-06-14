package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lightai-go/internal/server/db"
)

// TestAgentRegistrationWithGoodToken verifies registration with custom token.
func TestAgentRegistrationWithGoodToken(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()

	handler := NewAgentHandler(database, nil)
	body, _ := json.Marshal(map[string]interface{}{
		"node_id":  "node-001",
		"agent_id": "agent-001",
		"hostname": "host1",
	})
	req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleRegister(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// TestNodeIDAgentIDBindingOnReRegistration verifies agent_id mismatch is rejected.
func TestNodeIDAgentIDBindingOnReRegistration(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	handler := NewAgentHandler(database, nil)

	// First registration: node-001 with agent-001.
	body1, _ := json.Marshal(map[string]interface{}{
		"node_id": "node-001", "agent_id": "agent-001", "hostname": "host1",
	})
	req1 := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleRegister(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d", w1.Code)
	}

	// Second registration: same node-001, different agent-002 → 409 Conflict.
	body2, _ := json.Marshal(map[string]interface{}{
		"node_id": "node-001", "agent_id": "agent-002", "hostname": "host1",
	})
	req2 := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleRegister(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("agent_id mismatch: expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestHeartbeatAgentIDMismatchRejected verifies heartbeat with wrong agent_id.
func TestHeartbeatAgentIDMismatchRejected(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	handler := NewAgentHandler(database, nil)

	// Register.
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-001','agent-001','host1','online','default',datetime('now'),datetime('now'),datetime('now'))`)

	// Heartbeat with correct agent_id → 200.
	hb1, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-001"})
	req1 := httptest.NewRequest("POST", "/api/agent/heartbeat", bytes.NewReader(hb1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.HandleHeartbeat(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("correct heartbeat: expected 200, got %d", w1.Code)
	}

	// Heartbeat with wrong agent_id → 403.
	hb2, _ := json.Marshal(map[string]interface{}{"node_id": "node-001", "agent_id": "agent-002"})
	req2 := httptest.NewRequest("POST", "/api/agent/heartbeat", bytes.NewReader(hb2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleHeartbeat(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("mismatch heartbeat: expected 403, got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestHeartbeatUnregisteredNodeRequestsReregister verifies unknown node needs re-registration.
func TestHeartbeatUnregisteredNodeRequestsReregister(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	handler := NewAgentHandler(database, nil)

	hb, _ := json.Marshal(map[string]interface{}{"node_id": "unknown-node", "agent_id": "agent-x"})
	req := httptest.NewRequest("POST", "/api/agent/heartbeat", bytes.NewReader(hb))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleHeartbeat(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("unknown node heartbeat: expected 200 with need_register, got %d", w.Code)
	}
	var resp HeartbeatResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.NeedRegister {
		t.Error("unknown node heartbeat should request re-registration")
	}
}

// TestNodeIDAgentIDBindingOnReRegisterSameAgentOK verifies same agent_id re-registration OK.
// TestResourceReportAgentIDBinding verifies resource report checks agent_id.
func TestResourceReportAgentIDBinding(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	handler := NewResourceHandler(database, nil)

	// Register node with agent-001.
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-001','agent-001','host1','online','default',datetime('now'),datetime('now'),datetime('now'))`)

	// Resource report with wrong agent_id.
	body, _ := json.Marshal(map[string]interface{}{
		"agent_id": "agent-002",
		"gpu_resources": []map[string]interface{}{},
	})
	req := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleResourceReport(w, req)
	// Should return 404 because agent lookup fails.
	if w.Code != 404 {
		t.Errorf("wrong agent_id report: expected 404, got %d", w.Code)
	}

	// Resource report with correct agent_id.
	body2, _ := json.Marshal(map[string]interface{}{
		"agent_id": "agent-001",
		"gpu_resources": []map[string]interface{}{},
	})
	req2 := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleResourceReport(w2, req2)
	// Not found = 200 because the node exists and report succeeded (empty GPUs).
	if w2.Code != 200 {
		t.Errorf("correct agent_id report: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestNodeIDAgentIDBindingOnReRegisterSameAgentOK(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	handler := NewAgentHandler(database, nil)

	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]interface{}{
			"node_id": "node-001", "agent_id": "agent-001", "hostname": "host1",
		})
		req := httptest.NewRequest("POST", "/api/agent/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.HandleRegister(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("re-register %d with same agent: expected 201, got %d: %s", i, w.Code, w.Body.String())
		}
	}
}
