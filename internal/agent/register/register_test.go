package register

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lightai-go/internal/agent/state"
)

func TestDo_Success(t *testing.T) {
	// Mock server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RegisterResponse{
			NodeID:  "node-12345",
			AgentID: "agent-01",
		})
	}))
	defer server.Close()

	st, _ := state.Load(t.TempDir(), "agent-01")
	client := server.Client()

	nodeID, err := Do(client, Config{
		ServerURL:      server.URL,
		AgentToken:     "test-token",
		AgentID:        "agent-01",
		Hostname:       "test-host",
		AdvertisedAddr: "test-host",
		MetricsEnabled: true,
		MetricsScheme:  "http",
		MetricsPort:    19091,
		MetricsPath:    "/metrics",
		Version:        "0.1.0",
	}, st)

	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if nodeID != "node-12345" {
		t.Errorf("expected node-12345, got %q", nodeID)
	}
	if st.CachedNodeID() != "node-12345" {
		t.Errorf("expected cached node-12345, got %q", st.CachedNodeID())
	}
}

func TestDo_NodeIDEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RegisterResponse{
			NodeID:  "", // Empty!
			AgentID: "agent-01",
		})
	}))
	defer server.Close()

	st, _ := state.Load(t.TempDir(), "agent-01")
	client := server.Client()

	_, err := Do(client, Config{
		ServerURL:  server.URL,
		AgentToken: "test-token",
		AgentID:    "agent-01",
		Hostname:   "test",
	}, st)

	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

func TestDo_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json {{{"))
	}))
	defer server.Close()

	st, _ := state.Load(t.TempDir(), "agent-01")
	client := server.Client()

	_, err := Do(client, Config{
		ServerURL:  server.URL,
		AgentToken: "test-token",
		AgentID:    "agent-01",
		Hostname:   "test",
	}, st)

	if err == nil {
		t.Error("expected parse error for malformed response")
	}
}

func TestDo_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	st, _ := state.Load(t.TempDir(), "agent-01")
	client := server.Client()

	_, err := Do(client, Config{
		ServerURL:  server.URL,
		AgentToken: "wrong-token",
		AgentID:    "agent-01",
		Hostname:   "test",
	}, st)

	if err == nil {
		t.Error("expected error for non-2xx response")
	}
}

func TestDo_NodeIDReused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RegisterResponse{
			NodeID:  "node-existing",
			AgentID: "agent-01",
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	st, _ := state.Load(dir, "agent-01")
	st.SetNodeID("node-existing") // Pre-cache same ID.

	client := server.Client()
	nodeID, err := Do(client, Config{
		ServerURL:  server.URL,
		AgentToken: "test-token",
		AgentID:    "agent-01",
		Hostname:   "test",
	}, st)

	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if nodeID != "node-existing" {
		t.Errorf("expected node-existing, got %q", nodeID)
	}
	// Should still be the same.
	if st.CachedNodeID() != "node-existing" {
		t.Errorf("expected cached node-existing, got %q", st.CachedNodeID())
	}
}

func TestDo_Mismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RegisterResponse{
			NodeID:  "node-new-server",
			AgentID: "agent-01",
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	st, _ := state.Load(dir, "agent-01")
	st.SetNodeID("node-old-cached") // Different cached value.

	client := server.Client()
	nodeID, err := Do(client, Config{
		ServerURL:  server.URL,
		AgentToken: "test-token",
		AgentID:    "agent-01",
		Hostname:   "test",
	}, st)

	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if nodeID != "node-new-server" {
		t.Errorf("expected server ID node-new-server, got %q", nodeID)
	}
	// Should have updated.
	if st.CachedNodeID() != "node-new-server" {
		t.Errorf("expected cached update to node-new-server, got %q", st.CachedNodeID())
	}
}

func TestDo_ServerNotReachable(t *testing.T) {
	st, _ := state.Load(t.TempDir(), "agent-01")
	client := &http.Client{}

	_, err := Do(client, Config{
		ServerURL:  "http://127.0.0.1:19999", // Non-existent port.
		AgentToken: "test-token",
		AgentID:    "agent-01",
		Hostname:   "test",
	}, st)

	if err == nil {
		t.Error("expected connection error")
	}
}
