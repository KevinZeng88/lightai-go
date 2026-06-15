// Package register handles agent registration with the LightAI Server.
package register

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"lightai-go/internal/agent/state"
	"lightai-go/internal/common/log"
)

// RegisterResponse matches the server's agent register response.
type RegisterResponse struct {
	NodeID     string `json:"node_id"`
	AgentID    string `json:"agent_id"`
	TenantID   string `json:"tenant_id"`
	ServerTime string `json:"server_time"`
}

// HeartbeatResponse matches the server's heartbeat response.
type HeartbeatResponse struct {
	Status       string `json:"status"`
	ServerTime   string `json:"server_time"`
	NeedRegister bool   `json:"need_register,omitempty"`
}

// Config holds registration configuration.
type Config struct {
	ServerURL      string
	AgentToken     string
	AgentID        string
	Hostname       string
	PrimaryIP      string
	AdvertisedAddr string
	OS             string
	Arch           string
	Kernel         string
	MetricsEnabled bool
	MetricsScheme  string
	MetricsPort    int
	MetricsPath    string
	Version        string
	RequestTimeout time.Duration
}

// Do performs agent registration with the server.
// It returns the server-assigned node_id, which is also persisted to local state.
func Do(client *http.Client, cfg Config, st *state.State) (nodeID string, err error) {
	log.Debug("register start",
		"agent_id", cfg.AgentID,
		"server_url", cfg.ServerURL,
		"cached_node_id", st.CachedNodeID(),
	)

	reqBody := map[string]interface{}{
		"node_id":            st.CachedNodeID(),
		"agent_id":           cfg.AgentID,
		"hostname":           cfg.Hostname,
		"primary_ip":         cfg.PrimaryIP,
		"advertised_address": cfg.AdvertisedAddr,
		"os":                 cfg.OS,
		"arch":               cfg.Arch,
		"kernel":             cfg.Kernel,
		"metrics_enabled":    cfg.MetricsEnabled,
		"metrics_scheme":     cfg.MetricsScheme,
		"metrics_port":       cfg.MetricsPort,
		"metrics_path":       cfg.MetricsPath,
		"version":            cfg.Version,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal register request: %w", err)
	}

	req, err := http.NewRequest("POST", cfg.ServerURL+"/api/v1/agent/register", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AgentToken)

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		log.Error("register failed",
			"agent_id", cfg.AgentID,
			"error", err,
			"latency_ms", latency.Milliseconds(),
		)
		return "", fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read body for logging (limited size).
	bodyBytes, _ = io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		log.Error("register failed",
			"agent_id", cfg.AgentID,
			"status", resp.StatusCode,
			"latency_ms", latency.Milliseconds(),
			"response_body", truncate(string(bodyBytes), 200),
		)
		return "", fmt.Errorf("register returned status %d: %s", resp.StatusCode, truncate(string(bodyBytes), 200))
	}

	// Parse response.
	var regResp RegisterResponse
	if err := json.Unmarshal(bodyBytes, &regResp); err != nil {
		log.Error("register response parse failed",
			"agent_id", cfg.AgentID,
			"error", err,
			"response_body", truncate(string(bodyBytes), 200),
		)
		return "", fmt.Errorf("parse register response: %w", err)
	}

	// Validate node_id.
	if regResp.NodeID == "" {
		log.Error("node_id missing in register response",
			"agent_id", cfg.AgentID,
			"response_body", truncate(string(bodyBytes), 200),
		)
		return "", fmt.Errorf("node_id missing in register response")
	}

	serverNodeID := regResp.NodeID
	cachedNodeID := st.CachedNodeID()

	log.Info("register success",
		"agent_id", cfg.AgentID,
		"server_returned_node_id", serverNodeID,
		"cached_node_id", cachedNodeID,
		"latency_ms", latency.Milliseconds(),
	)

	// Check for mismatch.
	if cachedNodeID != "" && cachedNodeID != serverNodeID {
		log.Warn("node_id_mismatch",
			"agent_id", cfg.AgentID,
			"cached_node_id", cachedNodeID,
			"server_returned_node_id", serverNodeID,
		)
	}

	// Persist node_id.
	if cachedNodeID == "" || cachedNodeID != serverNodeID {
		if err := st.SetNodeID(serverNodeID); err != nil {
			log.Error("failed to persist node_id",
				"agent_id", cfg.AgentID,
				"node_id", serverNodeID,
				"error", err,
			)
			// Don't fail registration for persistence error.
		} else {
			if cachedNodeID == "" {
				log.Info("node_id persisted",
					"agent_id", cfg.AgentID,
					"node_id", serverNodeID,
				)
			} else {
				log.Info("node_id updated from mismatch",
					"agent_id", cfg.AgentID,
					"old_node_id", cachedNodeID,
					"new_node_id", serverNodeID,
				)
			}
		}
	} else {
		log.Info("node_id reused",
			"agent_id", cfg.AgentID,
			"node_id", serverNodeID,
		)
	}

	return serverNodeID, nil
}

// SendHeartbeat sends a heartbeat to the server.
func SendHeartbeat(client *http.Client, serverURL, agentToken, agentID, nodeID string) (*HeartbeatResponse, error) {
	reqBody := map[string]string{"node_id": nodeID, "agent_id": agentID}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", serverURL+"/api/v1/agent/heartbeat", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+agentToken)

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		log.Warn("heartbeat failed",
			"agent_id", agentID,
			"error", err,
			"latency_ms", latency.Milliseconds(),
		)
		return nil, fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ = io.ReadAll(io.LimitReader(resp.Body, 1024))

	var hbResp HeartbeatResponse
	if err := json.Unmarshal(bodyBytes, &hbResp); err != nil {
		return nil, fmt.Errorf("parse heartbeat response: %w", err)
	}

	log.Debug("heartbeat success",
		"agent_id", agentID,
		"node_id", agentID, // placeholder, actual node_id is tracked in state
		"latency_ms", latency.Milliseconds(),
		"status", hbResp.Status,
	)

	return &hbResp, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
