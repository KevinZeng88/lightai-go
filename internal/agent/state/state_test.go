package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FreshState(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir, "agent-test-01")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s.CachedNodeID() != "" {
		t.Errorf("expected empty cached node_id, got %q", s.CachedNodeID())
	}
	if s.AgentID != "agent-test-01" {
		t.Errorf("expected agent-test-01, got %q", s.AgentID)
	}
}

func TestSetNodeID_Persists(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir, "agent-01")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if err := s.SetNodeID("node-abc-123"); err != nil {
		t.Fatalf("SetNodeID failed: %v", err)
	}

	// Reload and verify.
	s2, err := Load(dir, "agent-01")
	if err != nil {
		t.Fatalf("re-Load failed: %v", err)
	}
	if s2.CachedNodeID() != "node-abc-123" {
		t.Errorf("expected node-abc-123, got %q", s2.CachedNodeID())
	}
}

func TestCheckMismatch_NoCached(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir, "agent-01")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s.CheckMismatch("server-node-1") {
		t.Error("expected no mismatch when no cached node_id")
	}
}

func TestCheckMismatch_Match(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir, "agent-01")
	s.SetNodeID("node-xyz")

	if s.CheckMismatch("node-xyz") {
		t.Error("expected no mismatch when IDs match")
	}
}

func TestCheckMismatch_Mismatch(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir, "agent-01")
	s.SetNodeID("node-old")

	if !s.CheckMismatch("node-new") {
		t.Error("expected mismatch detected")
	}
}

func TestCheckMismatch_EmptyServer(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir, "agent-01")
	s.SetNodeID("node-old")

	if s.CheckMismatch("") {
		t.Error("expected no mismatch when server node_id is empty")
	}
}

func TestLoad_CorruptState(t *testing.T) {
	dir := t.TempDir()
	// Write corrupt JSON.
	os.WriteFile(filepath.Join(dir, stateFileName), []byte("not json"), 0600)

	s, err := Load(dir, "agent-01")
	if err != nil {
		t.Fatalf("Load should not error on corrupt state: %v", err)
	}
	if s.CachedNodeID() != "" {
		t.Errorf("expected empty node_id after corrupt state, got %q", s.CachedNodeID())
	}
}
