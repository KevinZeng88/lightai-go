package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FirstStart_GeneratesNodeID(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir, "agent-test-01", "test-host")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s.NodeID() == "" {
		t.Error("expected generated node_id, got empty")
	}
	if !hasPrefix(s.NodeID(), "node-") {
		t.Errorf("expected node_id to start with 'node-', got %q", s.NodeID())
	}

	// Verify file was written with 0600.
	path := filepath.Join(dir, identityFileName)
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("identity file not created: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %#o", fi.Mode().Perm())
	}
}

func TestLoad_ReusesExistingNodeID(t *testing.T) {
	dir := t.TempDir()
	s1, err := Load(dir, "agent-01", "host-a")
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}
	nodeID := s1.NodeID()

	// Reload — must get the same node_id.
	s2, err := Load(dir, "agent-01", "host-a")
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}
	if s2.NodeID() != nodeID {
		t.Errorf("expected %q, got %q on reload", nodeID, s2.NodeID())
	}
}

func TestLoad_CorruptState_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, identityFileName), []byte("not json"), 0600)

	_, err := Load(dir, "agent-01", "host-a")
	if err == nil {
		t.Fatal("expected error on corrupt identity, got nil")
	}
}

func TestLoad_EmptyNodeID_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, identityFileName), []byte(`{"node_id":""}`), 0600)

	_, err := Load(dir, "agent-01", "host-a")
	if err == nil {
		t.Fatal("expected error on empty node_id, got nil")
	}
}

func TestSetNodeID_Persists(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir, "agent-01", "host-a")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if err := s.SetNodeID("node-explicit-123"); err != nil {
		t.Fatalf("SetNodeID failed: %v", err)
	}

	s2, err := Load(dir, "agent-01", "host-b")
	if err != nil {
		t.Fatalf("re-Load failed: %v", err)
	}
	if s2.NodeID() != "node-explicit-123" {
		t.Errorf("expected node-explicit-123, got %q", s2.NodeID())
	}
}

func TestCheckMismatch_NoCached(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir, "agent-01", "host-a")
	nodeID := s.NodeID()
	// After Load, node_id is always set. Same node_id should match.
	if s.CheckMismatch(nodeID) {
		t.Error("expected no mismatch when server returns same node_id")
	}
}

func TestCheckMismatch_Match(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir, "agent-01", "host-a")
	nodeID := s.NodeID()

	if s.CheckMismatch(nodeID) {
		t.Error("expected no mismatch when IDs match")
	}
}

func TestCheckMismatch_Mismatch(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir, "agent-01", "host-a")
	s.SetNodeID("node-old")

	if !s.CheckMismatch("node-new") {
		t.Error("expected mismatch detected")
	}
}

func TestCheckMismatch_EmptyServer(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir, "agent-01", "host-a")

	if s.CheckMismatch("") {
		t.Error("expected no mismatch when server node_id is empty")
	}
}

func TestIdentity_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	s, err := Load(dir, "agent-01", "host-a")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify current perms.
	fi, err := os.Stat(s.Path())
	if err != nil {
		t.Fatalf("stat identity file: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %#o", fi.Mode().Perm())
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
