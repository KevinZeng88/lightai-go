package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"

	"github.com/google/uuid"
)

// AgentHandler handles Agent API endpoints.
type AgentHandler struct {
	DB *db.DB
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(database *db.DB) *AgentHandler {
	return &AgentHandler{DB: database}
}

// RegisterRequest is the agent registration request.
type RegisterRequest struct {
	AgentID        string `json:"agent_id"`
	Hostname       string `json:"hostname"`
	AdvertisedAddr string `json:"advertised_address"`
	MetricsEnabled bool   `json:"metrics_enabled"`
	MetricsScheme  string `json:"metrics_scheme"`
	MetricsPort    int    `json:"metrics_port"`
	MetricsPath    string `json:"metrics_path"`
	Version        string `json:"version"`
}

// RegisterResponse is the agent registration response.
type RegisterResponse struct {
	NodeID     string `json:"node_id"`
	AgentID    string `json:"agent_id"`
	TenantID   string `json:"tenant_id"`
	AgentToken string `json:"agent_token"`
	ServerTime string `json:"server_time"`
}

// HeartbeatResponse is the heartbeat response.
type HeartbeatResponse struct {
	Status       string `json:"status"`
	ServerTime   string `json:"server_time"`
	NeedRegister bool   `json:"need_register,omitempty"`
}

// HandleRegister handles POST /api/agent/register.
func (h *AgentHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		http.Error(w, `{"error":"agent_id required"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)

	// Default metrics settings.
	if req.MetricsScheme == "" {
		req.MetricsScheme = "http"
	}
	if req.MetricsPath == "" {
		req.MetricsPath = "/metrics"
	}
	if req.MetricsPort == 0 {
		req.MetricsPort = 9090
	}

	// Upsert node: check if agent_id already exists.
	var nodeID string
	err := h.DB.QueryRow(`SELECT id FROM nodes WHERE agent_id = ?`, req.AgentID).Scan(&nodeID)
	if err == sql.ErrNoRows {
		// Create new node.
		nodeID = uuid.NewString()
		_, err = h.DB.Exec(
			`INSERT INTO nodes (id, agent_id, hostname, advertised_address,
			 metrics_enabled, metrics_scheme, metrics_port, metrics_path,
			 status, last_heartbeat_at, tenant_id, owner_id, created_by, updated_by,
			 created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'online', ?, 'default', NULL, 'system', 'system', ?, ?)`,
			nodeID, req.AgentID, req.Hostname, req.AdvertisedAddr,
			boolToInt(req.MetricsEnabled), req.MetricsScheme, req.MetricsPort, req.MetricsPath,
			now, now, now,
		)
		if err != nil {
			log.Error("create node error", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		log.Info("node registered", "node_id", nodeID, "agent_id", req.AgentID, "hostname", req.Hostname)
	} else if err != nil {
		log.Error("query node error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	} else {
		// Update existing node.
		_, err = h.DB.Exec(
			`UPDATE nodes SET hostname = ?, advertised_address = ?,
			 metrics_enabled = ?, metrics_scheme = ?, metrics_port = ?, metrics_path = ?,
			 status = 'online', last_heartbeat_at = ?, updated_at = ?
			 WHERE id = ?`,
			req.Hostname, req.AdvertisedAddr,
			boolToInt(req.MetricsEnabled), req.MetricsScheme, req.MetricsPort, req.MetricsPath,
			now, now, nodeID,
		)
		if err != nil {
			log.Error("update node error", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		log.Info("node re-registered", "node_id", nodeID, "agent_id", req.AgentID)
	}

	resp := RegisterResponse{
		NodeID:     nodeID,
		AgentID:    req.AgentID,
		TenantID:   "default",
		ServerTime: now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// HandleHeartbeat handles POST /api/agent/heartbeat.
func (h *AgentHandler) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		http.Error(w, `{"error":"agent_id required"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)

	// Update heartbeat.
	result, err := h.DB.Exec(
		`UPDATE nodes SET last_heartbeat_at = ?, status = 'online', updated_at = ? WHERE agent_id = ?`,
		now, now, req.AgentID,
	)
	if err != nil {
		log.Error("heartbeat update error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		// Node not registered.
		resp := HeartbeatResponse{
			Status:       "error",
			ServerTime:   now,
			NeedRegister: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := HeartbeatResponse{
		Status:     "ok",
		ServerTime: now,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleListNode handles GET /api/nodes.
func (h *AgentHandler) HandleListNodes(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(
		`SELECT id, agent_id, hostname, advertised_address, metrics_enabled,
		        metrics_scheme, metrics_port, metrics_path,
		        status, last_heartbeat_at, tenant_id, created_at, updated_at
		 FROM nodes WHERE tenant_id = 'default'
		 ORDER BY hostname`,
	)
	if err != nil {
		log.Error("list nodes error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var nodes []map[string]interface{}
	for rows.Next() {
		var id, agentID, hostname, addr, scheme, path, status, tenantID, createdAt, updatedAt string
		var metricsEnabled int
		var metricsPort int
		var lastHB sql.NullString
		if err := rows.Scan(&id, &agentID, &hostname, &addr, &metricsEnabled,
			&scheme, &metricsPort, &path,
			&status, &lastHB, &tenantID, &createdAt, &updatedAt); err != nil {
			continue
		}
		node := map[string]interface{}{
			"id":                 id,
			"agent_id":           agentID,
			"hostname":           hostname,
			"advertised_address": addr,
			"metrics_enabled":    metricsEnabled == 1,
			"metrics_scheme":     scheme,
			"metrics_port":       metricsPort,
			"metrics_path":       path,
			"status":             status,
			"tenant_id":          tenantID,
			"created_at":         createdAt,
			"updated_at":         updatedAt,
		}
		if lastHB.Valid {
			node["last_heartbeat_at"] = lastHB.String
		}
		nodes = append(nodes, node)
	}
	if nodes == nil {
		nodes = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

// HandleGetNode handles GET /api/nodes/{id}.
func (h *AgentHandler) HandleGetNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	if nodeID == "" {
		http.Error(w, `{"error":"node id required"}`, http.StatusBadRequest)
		return
	}

	var id, agentID, hostname, addr, scheme, path, status, tenantID, createdAt, updatedAt string
	var metricsEnabled int
	var metricsPort int
	var lastHB sql.NullString

	err := h.DB.QueryRow(
		`SELECT id, agent_id, hostname, advertised_address, metrics_enabled,
		        metrics_scheme, metrics_port, metrics_path,
		        status, last_heartbeat_at, tenant_id, created_at, updated_at
		 FROM nodes WHERE id = ?`, nodeID,
	).Scan(&id, &agentID, &hostname, &addr, &metricsEnabled,
		&scheme, &metricsPort, &path,
		&status, &lastHB, &tenantID, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("get node error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	node := map[string]interface{}{
		"id":                 id,
		"agent_id":           agentID,
		"hostname":           hostname,
		"advertised_address": addr,
		"metrics_enabled":    metricsEnabled == 1,
		"metrics_scheme":     scheme,
		"metrics_port":       metricsPort,
		"metrics_path":       path,
		"status":             status,
		"tenant_id":          tenantID,
		"created_at":         createdAt,
		"updated_at":         updatedAt,
	}
	if lastHB.Valid {
		node["last_heartbeat_at"] = lastHB.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

// GetMetricsTargets returns Prometheus HTTP SD targets from registered nodes.
func (h *AgentHandler) GetMetricsTargets() []map[string]interface{} {
	rows, err := h.DB.Query(
		`SELECT agent_id, hostname, advertised_address, metrics_scheme, metrics_port, metrics_path
		 FROM nodes
		 WHERE metrics_enabled = 1 AND advertised_address != '' AND metrics_port > 0`,
	)
	if err != nil {
		log.Error("query metrics targets error", "error", err)
		return nil
	}
	defer rows.Close()

	type target struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}

	var targets []map[string]interface{}
	for rows.Next() {
		var agentID, hostname, addr, scheme, path string
		var port int
		if err := rows.Scan(&agentID, &hostname, &addr, &scheme, &port, &path); err != nil {
			continue
		}
		targets = append(targets, map[string]interface{}{
			"targets": []string{addr + ":" + strconv.Itoa(port)},
			"labels": map[string]string{
				"agent_id":         agentID,
				"hostname":         hostname,
				"__metrics_path__": path,
				"__scheme__":       scheme,
			},
		})
	}
	if targets == nil {
		targets = []map[string]interface{}{}
	}

	return targets
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
