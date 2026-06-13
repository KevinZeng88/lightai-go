// Package state manages agent-local state persistence (node_id cache).
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"lightai-go/internal/common/log"
)

// State holds the agent's persistent local state.
type State struct {
	mu      sync.RWMutex
	dataDir string
	NodeID  string `json:"node_id"`
	AgentID string `json:"agent_id"`
}

const stateFileName = "agent-state.json"

// Load loads agent state from disk, or creates a fresh one.
func Load(dataDir, agentID string) (*State, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", dataDir, err)
	}

	s := &State{dataDir: dataDir, AgentID: agentID}
	path := filepath.Join(dataDir, stateFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("no cached agent state, starting fresh",
				"agent_id", agentID,
			)
			return s, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		log.Warn("corrupt agent state file, starting fresh",
			"error", err,
			"agent_id", agentID,
		)
		// Reset.
		s.NodeID = ""
		return s, nil
	}

	log.Info("cached node_id loaded",
		"cached_node_id", s.NodeID,
		"agent_id", agentID,
	)
	return s, nil
}

// CachedNodeID returns the currently cached node_id (empty if none).
func (s *State) CachedNodeID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.NodeID
}

// SetNodeID updates the node_id in memory and persists to disk.
func (s *State) SetNodeID(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.NodeID = nodeID
	return s.saveLocked()
}

// CheckMismatch compares a server-returned node_id with the cached value.
// Returns true if there is a mismatch (both non-empty and different).
func (s *State) CheckMismatch(serverNodeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cached := s.NodeID
	if cached == "" || serverNodeID == "" {
		return false
	}
	return cached != serverNodeID
}

func (s *State) saveLocked() error {
	path := filepath.Join(s.dataDir, stateFileName)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	log.Info("node_id persisted",
		"node_id", s.NodeID,
		"path", path,
	)
	return nil
}
