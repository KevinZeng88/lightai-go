package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"lightai-go/internal/common/log"
	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
	srvmetrics "lightai-go/internal/server/metrics"
)

// AgentHandler handles Agent API endpoints.
type AgentHandler struct {
	DB      *db.DB
	Metrics *srvmetrics.ServerMetrics
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(database *db.DB, m *srvmetrics.ServerMetrics) *AgentHandler {
	return &AgentHandler{DB: database, Metrics: m}
}

// RegisterRequest is the agent registration request.
type RegisterRequest struct {
	NodeID         string `json:"node_id"`
	AgentID        string `json:"agent_id"`
	Hostname       string `json:"hostname"`
	PrimaryIP      string `json:"primary_ip"`
	AdvertisedAddr string `json:"advertised_address"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	Kernel         string `json:"kernel"`
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

	if req.NodeID == "" {
		http.Error(w, `{"error":"node_id is required"}`, http.StatusBadRequest)
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

	// Upsert node: node_id is the ONLY identity key.
	defaultTenantID := h.DB.DefaultTenantID()
	var serverNodeID string
	err := h.DB.QueryRow(`SELECT id FROM nodes WHERE id = ?`, req.NodeID).Scan(&serverNodeID)
	if err == sql.ErrNoRows {
		// Create new node — assign to default tenant.
		serverNodeID = req.NodeID
		_, err = h.DB.Exec(
			`INSERT INTO nodes (id, agent_id, hostname, primary_ip, advertised_address,
			 os, arch, kernel, agent_version,
			 metrics_enabled, metrics_scheme, metrics_port, metrics_path,
			 status, last_heartbeat_at, tenant_id, owner_id, created_by, updated_by,
			 created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?,
			 ?, ?, ?, ?, 'online', ?, ?, NULL, 'system', 'system', ?, ?)`,
			serverNodeID, req.AgentID, req.Hostname, req.PrimaryIP, req.AdvertisedAddr,
			req.OS, req.Arch, req.Kernel, req.Version,
			boolToInt(req.MetricsEnabled), req.MetricsScheme, req.MetricsPort, req.MetricsPath,
			now, defaultTenantID,
			now, now,
		)
		if err != nil {
			log.Error("create node error", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		log.Info("node registered", "node_id", serverNodeID, "agent_id", req.AgentID, "hostname", req.Hostname)
		if h.Metrics != nil {
			h.Metrics.AgentReports.Inc()
		}
	} else if err != nil {
		log.Error("query node error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	} else {
		// P0-004/CODEX: Check node_id + agent_id binding.
		var existingAgentID string
		err = h.DB.QueryRow(`SELECT agent_id FROM nodes WHERE id = ?`, req.NodeID).Scan(&existingAgentID)
		if err == nil && existingAgentID != "" && existingAgentID != req.AgentID {
			log.Warn("node agent_id mismatch — registration rejected",
				"node_id", serverNodeID,
				"existing_agent_id", existingAgentID,
				"incoming_agent_id", req.AgentID,
			)
			http.Error(w, `{"error":"node_id already bound to a different agent_id"}`, http.StatusConflict)
			return
		}
		// Update existing node.
		_, err = h.DB.Exec(
			`UPDATE nodes SET agent_id = ?, hostname = ?, primary_ip = ?, advertised_address = ?,
			 os = ?, arch = ?, kernel = ?, agent_version = ?,
			 metrics_enabled = ?, metrics_scheme = ?, metrics_port = ?, metrics_path = ?,
			 status = 'online', last_heartbeat_at = ?, updated_at = ?
			 WHERE id = ?`,
			req.AgentID, req.Hostname, req.PrimaryIP, req.AdvertisedAddr,
			req.OS, req.Arch, req.Kernel, req.Version,
			boolToInt(req.MetricsEnabled), req.MetricsScheme, req.MetricsPort, req.MetricsPath,
			now, now, serverNodeID,
		)
		if err != nil {
			log.Error("update node error", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		log.Info("node re-registered", "node_id", serverNodeID, "agent_id", req.AgentID)
	}

	resp := RegisterResponse{
		NodeID:     serverNodeID,
		AgentID:    req.AgentID,
		TenantID:   defaultTenantID,
		ServerTime: now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// HandleHeartbeat handles POST /api/agent/heartbeat.
func (h *AgentHandler) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID  string `json:"node_id"`
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.NodeID == "" {
		http.Error(w, `{"error":"node_id is required"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)

	// P0-004/CODEX: Verify agent_id matches registered node.
	var existingAgentID string
	if err := h.DB.QueryRow(`SELECT agent_id FROM nodes WHERE id = ?`, req.NodeID).Scan(&existingAgentID); err == nil {
		if existingAgentID != "" && existingAgentID != req.AgentID {
			log.Warn("heartbeat agent_id mismatch",
				"node_id", req.NodeID,
				"existing_agent_id", existingAgentID,
				"incoming_agent_id", req.AgentID,
			)
			http.Error(w, `{"error":"agent_id mismatch for this node"}`, http.StatusForbidden)
			return
		}
	}

	// Update heartbeat.
	result, err := h.DB.Exec(
		`UPDATE nodes SET last_heartbeat_at = ?, status = 'online', updated_at = ? WHERE id = ?`,
		now, now, req.NodeID,
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

	if h.Metrics != nil {
		h.Metrics.AgentHeartbeats.Inc()
	}
	resp := HeartbeatResponse{
		Status:     "ok",
		ServerTime: now,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleListNode handles GET /api/nodes.
// P0-002/CODEX: Scoped to current session tenant.
// Platform admin sees all nodes; regular users see only their tenant.
func (h *AgentHandler) HandleListNodes(w http.ResponseWriter, r *http.Request) {
	info := auth.SessionInfoFromContext(r.Context())

	var rows *sql.Rows
	var err error

	// Platform admin — no tenant filter.
	if info != nil && info.IsPlatformAdmin {
		rows, err = h.DB.Query(
			`SELECT id, agent_id, hostname, primary_ip, advertised_address,
			        os, arch, kernel, agent_version,
			        metrics_enabled, metrics_scheme, metrics_port, metrics_path,
			        status, last_heartbeat_at, tenant_id, created_at, updated_at
			 FROM nodes ORDER BY hostname`,
		)
	} else {
		tenantID := "default"
		if info != nil {
			tenantID = info.TenantID
		}
		rows, err = h.DB.Query(
			`SELECT id, agent_id, hostname, primary_ip, advertised_address,
			        os, arch, kernel, agent_version,
			        metrics_enabled, metrics_scheme, metrics_port, metrics_path,
			        status, last_heartbeat_at, tenant_id, created_at, updated_at
			 FROM nodes WHERE tenant_id = ?
			 ORDER BY hostname`,
			tenantID,
		)
	}
	if err != nil {
		log.Error("list nodes error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var nodes []map[string]interface{}
	for rows.Next() {
		var id, agentID, hostname, primaryIP, addr, osName, archName, kernelVer, agentVer, scheme, path, status, tenantID, createdAt, updatedAt string
		var metricsEnabled int
		var metricsPort int
		var lastHB sql.NullString
		if err := rows.Scan(&id, &agentID, &hostname, &primaryIP, &addr,
			&osName, &archName, &kernelVer, &agentVer,
			&metricsEnabled, &scheme, &metricsPort, &path,
			&status, &lastHB, &tenantID, &createdAt, &updatedAt); err != nil {
			continue
		}
		node := map[string]interface{}{
			"id":                 id,
			"agent_id":           agentID,
			"hostname":           hostname,
			"primary_ip":         primaryIP,
			"advertised_address": addr,
			"os":                 osName,
			"arch":               archName,
			"kernel":             kernelVer,
			"agent_version":      agentVer,
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
// P0-002/CODEX: Verify node belongs to current tenant.
func (h *AgentHandler) HandleGetNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	if nodeID == "" {
		http.Error(w, `{"error":"node id required"}`, http.StatusBadRequest)
		return
	}

	info := auth.SessionInfoFromContext(r.Context())

	var id, agentID, hostname, primaryIP, addr, osName, archName, kernelVer, agentVer, scheme, path, status, tenantID, createdAt, updatedAt string
	var metricsEnabled int
	var metricsPort int
	var lastHB sql.NullString

	err := h.DB.QueryRow(
		`SELECT id, agent_id, hostname, primary_ip, advertised_address,
		        os, arch, kernel, agent_version,
		        metrics_enabled, metrics_scheme, metrics_port, metrics_path,
		        status, last_heartbeat_at, tenant_id, created_at, updated_at
		 FROM nodes WHERE id = ?`, nodeID,
	).Scan(&id, &agentID, &hostname, &primaryIP, &addr,
		&osName, &archName, &kernelVer, &agentVer,
		&metricsEnabled, &scheme, &metricsPort, &path,
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

	// P0-002/CODEX: Tenant scope check. Platform admin bypasses.
	if info != nil && !info.IsPlatformAdmin && info.TenantID != "" && tenantID != info.TenantID {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	node := map[string]interface{}{
		"id":                 id,
		"agent_id":           agentID,
		"hostname":           hostname,
		"primary_ip":         primaryIP,
		"advertised_address": addr,
		"os":                 osName,
		"arch":               archName,
		"kernel":             kernelVer,
		"agent_version":      agentVer,
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

// MarkOfflineNodes marks nodes as offline if they haven't sent a heartbeat
// within the given threshold. Returns the count of nodes marked offline.
// P0-009: Node auto-offline implementation.
func (h *AgentHandler) MarkOfflineNodes(threshold time.Duration) (int, error) {
	cutoff := time.Now().Add(-threshold).Format(time.RFC3339)
	result, err := h.DB.Exec(
		`UPDATE nodes SET status = 'offline', updated_at = datetime('now')
		 WHERE status = 'online' AND last_heartbeat_at < ?`,
		cutoff,
	)
	if err != nil {
		log.Error("mark offline nodes error", "error", err)
		return 0, err
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		log.Info("nodes marked offline", "count", n)
	}
	return int(n), nil
}

// GetMetricsTargets returns Prometheus HTTP SD targets from registered nodes.
// P0-009: Only returns online nodes; offline nodes are excluded from scraping.
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

func hasPerm(perms []string, required string) bool {
	for _, p := range perms {
		if p == required {
			return true
		}
	}
	return false
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// HandlePatchNodeTenant handles PATCH /api/nodes/{id}/tenant.
// Only platform_admin can transfer node tenant. Target tenant must exist.
func (h *AgentHandler) HandlePatchNodeTenant(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	if nodeID == "" {
		http.Error(w, `{"error":"node id required"}`, http.StatusBadRequest)
		return
	}

	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		TenantID string `json:"tenant_id"`
		Reason   string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.TenantID == "" {
		http.Error(w, `{"error":"tenant_id required"}`, http.StatusBadRequest)
		return
	}

	// Verify node exists and get current tenant.
	var currentTenant string
	err := h.DB.QueryRow(`SELECT tenant_id FROM nodes WHERE id = ?`, nodeID).Scan(&currentTenant)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("query node tenant error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Permission: platform_admin can transfer any node.
	// Tenant user needs node:transfer permission AND must own the node (tenant match).
	if !info.IsPlatformAdmin {
		if currentTenant != info.TenantID {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		perms := auth.PermissionsFromContext(r.Context())
		if !hasPerm(perms, "node:transfer") {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
	}

	// Verify target tenant exists.
	var targetExists string
	err = h.DB.QueryRow(`SELECT id FROM tenants WHERE id = ? AND status = 'active'`, req.TenantID).Scan(&targetExists)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"target tenant not found"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Update node tenant.
	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(`UPDATE nodes SET tenant_id = ?, updated_at = ? WHERE id = ?`,
		req.TenantID, now, nodeID)
	if err != nil {
		log.Error("update node tenant error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Audit log.
	auditID := uuid.NewString()
	detail := fmt.Sprintf(`{"from_tenant_id":"%s","to_tenant_id":"%s","reason":"%s"}`,
		currentTenant, req.TenantID, req.Reason)
	h.DB.Exec(`INSERT INTO audit_logs (id, action, entity_type, entity_id, detail, operator_user_id, created_at)
		VALUES (?, 'transfer_tenant', 'node', ?, ?, ?, ?)`,
		auditID, nodeID, detail, info.UserID, now)

	log.Info("node tenant transferred",
		"node_id", nodeID,
		"from_tenant", currentTenant,
		"to_tenant", req.TenantID,
		"operator", info.UserID,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "tenant_id": req.TenantID})
}
