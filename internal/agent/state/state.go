// Package state manages agent-local persistent identity.
// node_id is generated on first start, persisted to runtime/agent-identity.json,
// and reused on subsequent starts.  The identity file is the single source of truth.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"lightai-go/internal/common/log"
)

// Identity holds the agent's persistent node identity.
type Identity struct {
	NodeID    string `json:"node_id"`
	AgentID   string `json:"agent_id"`
	Hostname  string `json:"hostname"`
	CreatedAt string `json:"created_at"`
}

// State wraps the identity with a mutex and load/save semantics.
type State struct {
	mu       sync.RWMutex
	identity Identity
	path     string // full path to identity file
	dir      string // dir of identity file (for cleanup)
}

const identityFileName = "agent-identity.json"

// Load loads agent identity from dir/agent-identity.json.
// If the file does not exist, a new node_id is generated and persisted.
// If the file is corrupt, Load returns an error (do not silently regenerate).
// dir is typically "runtime".
func Load(dir, agentID, hostname string) (*State, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create identity dir %s: %w", dir, err)
	}

	path := filepath.Join(dir, identityFileName)
	s := &State{
		dir:      dir,
		path:     path,
		identity: Identity{AgentID: agentID, Hostname: hostname},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// First start — generate a new node_id.
			s.identity.NodeID = "node-" + uuid.NewString()
			s.identity.CreatedAt = time.Now().Format(time.RFC3339)
			if err := s.saveLocked(); err != nil {
				return nil, fmt.Errorf("write initial identity: %w", err)
			}
			log.Info("new node identity generated",
				"node_id", s.identity.NodeID,
				"agent_id", agentID,
				"identity_file", path,
			)
			return s, nil
		}
		return nil, fmt.Errorf("read identity file %s: %w", path, err)
	}

	// Parse existing identity.
	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return nil, fmt.Errorf("identity file %s is corrupt: %w — remove it and restart, or run scripts/reset-agent-identity.sh", path, err)
	}

	// Validate.
	if id.NodeID == "" {
		return nil, fmt.Errorf("identity file %s has empty node_id — remove it and restart, or run scripts/reset-agent-identity.sh", path)
	}

	// Merge loaded identity (agent_id / hostname may have changed).
	s.identity = id
	s.identity.AgentID = agentID
	s.identity.Hostname = hostname
	if s.identity.CreatedAt == "" {
		s.identity.CreatedAt = time.Now().Format(time.RFC3339)
	}

	log.Info("loaded persistent node identity",
		"node_id", s.identity.NodeID,
		"agent_id", agentID,
		"created_at", s.identity.CreatedAt,
	)

	return s, nil
}

// NodeID returns the persistent node_id.
func (s *State) NodeID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.identity.NodeID
}

// CachedNodeID is an alias for NodeID (backward compatibility with register package).
func (s *State) CachedNodeID() string { return s.NodeID() }

// SetNodeID updates the node_id and persists.  Used when the server returns a
// different node_id (should not happen in normal operation after initial gen).
func (s *State) SetNodeID(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identity.NodeID = nodeID
	return s.saveLocked()
}

// CheckMismatch compares a server-returned node_id with the local identity.
func (s *State) CheckMismatch(serverNodeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cached := s.identity.NodeID
	if cached == "" || serverNodeID == "" {
		return false
	}
	return cached != serverNodeID
}

// Identity returns a copy of the full identity (for registration payload).
func (s *State) Identity() Identity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.identity
}

// Path returns the identity file path.
func (s *State) Path() string { return s.path }

// saveLocked writes the identity to disk.  Caller must hold mu.
func (s *State) saveLocked() error {
	data, err := json.MarshalIndent(s.identity, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal identity: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("write identity file: %w", err)
	}
	return nil
}
