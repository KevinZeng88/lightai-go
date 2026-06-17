package api

import (
	"context"
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

// AgentTaskBrief is a lightweight task description sent in the heartbeat response.
type AgentTaskBrief struct {
	ID             string          `json:"id"`
	TaskType       string          `json:"task_type"`
	TenantID       string          `json:"tenant_id"`
	DeploymentID   string          `json:"deployment_id"`
	InstanceID     string          `json:"instance_id"`
	NodeID         string          `json:"node_id"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	AgentRunSpec   json.RawMessage `json:"agent_run_spec,omitempty"`
}

// HeartbeatResponse is the heartbeat response.
type HeartbeatResponse struct {
	Status       string           `json:"status"`
	ServerTime   string           `json:"server_time"`
	NeedRegister bool             `json:"need_register,omitempty"`
	Tasks        []AgentTaskBrief `json:"tasks,omitempty"`
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

	// Check previous state for transition logging.
	var prevStatus string
	h.DB.QueryRow(`SELECT status FROM nodes WHERE id = ?`, req.NodeID).Scan(&prevStatus)

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
		resp := HeartbeatResponse{
			Status:       "error",
			ServerTime:   now,
			NeedRegister: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// State transition: log if node was offline and is now online.
	if prevStatus == "offline" {
		log.StateTransition(r.Context(), "agent.heartbeat", "node", req.NodeID, "offline", "online")
	}

	if h.Metrics != nil {
		h.Metrics.AgentHeartbeats.Inc()
	}

	// Claim and return pending tasks for this node (atomic claim in transaction).
	tasks := claimAndReturnTasks(h.DB, req.NodeID, req.AgentID)

	resp := HeartbeatResponse{
		Status:     "ok",
		ServerTime: now,
		Tasks:      tasks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// claimAndReturnTasks atomically claims pending tasks for a node and returns them.
// REVIEW-004: Uses conditional UPDATE to prevent double-claim races between concurrent heartbeats.
func claimAndReturnTasks(database *db.DB, nodeID, agentID string) []AgentTaskBrief {
	tx, err := database.Begin()
	if err != nil {
		log.Error("claim tasks: cannot begin tx", "error", err)
		return nil
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	leaseExpiry := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)
	maxTasks := 10

	// Step 1: Sweep expired tasks and leases.
	sweepExpiredTasks(tx, nodeID, now)

	// Step 2: Atomically claim pending tasks with a conditional UPDATE.
	// Only tasks that are still 'pending' get claimed — eliminates SELECT-then-UPDATE race.
	result, err := tx.Exec(
		`UPDATE agent_tasks SET
			status = ?,
			claimed_at = ?,
			agent_id = ?,
			lease_owner = ?,
			lease_expires_at = ?,
			generation = generation + 1,
			attempt = attempt + 1,
			retry_count = retry_count + 1,
			updated_at = ?
		 WHERE id IN (
			SELECT id FROM agent_tasks
			WHERE node_id = ? AND status = ?
			ORDER BY created_at ASC LIMIT ?
		 )`,
		TaskStatusClaimed, now, agentID, agentID, leaseExpiry, now,
		nodeID, TaskStatusPending, maxTasks,
	)
	claimedCount := int64(0)
	if err == nil {
		claimedCount, _ = result.RowsAffected()
	}

	// Step 3: Fetch the tasks just claimed by this agent.
	type rawTask struct {
		id, taskType, tenantID, deploymentID, instanceID, taskNodeID, taskPayload string
		timeoutSeconds                                                            int
		generation                                                                int
		operationID                                                               string
	}
	var rawTasks []rawTask

	rows, err := tx.Query(
		`SELECT id, task_type, tenant_id, deployment_id, COALESCE(instance_id,''), node_id, timeout_seconds, payload, generation, operation_id
		 FROM agent_tasks WHERE node_id = ? AND status = ? AND agent_id = ? AND claimed_at = ?
		 ORDER BY created_at ASC LIMIT ?`,
		nodeID, TaskStatusClaimed, agentID, now, maxTasks,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rt rawTask
			if err := rows.Scan(&rt.id, &rt.taskType, &rt.tenantID, &rt.deploymentID, &rt.instanceID, &rt.taskNodeID, &rt.timeoutSeconds, &rt.taskPayload, &rt.generation, &rt.operationID); err != nil {
				continue
			}
			rawTasks = append(rawTasks, rt)
		}
	}

	if claimedCount > 0 {
		log.Info("claim: claimed tasks", "node_id", nodeID, "claimed_count", claimedCount)
	} else {
		log.Debug("claim: no tasks claimed", "node_id", nodeID)
	}

	if err := tx.Commit(); err != nil {
		log.Error("claim tasks: commit failed", "error", err)
		return nil
	}

	// Build response from claimed tasks. Include lease metadata for agent-side validation.
	var tasks []AgentTaskBrief
	for _, rt := range rawTasks {
		tasks = append(tasks, AgentTaskBrief{
			ID: rt.id, TaskType: rt.taskType, TenantID: rt.tenantID,
			DeploymentID: rt.deploymentID, InstanceID: rt.instanceID,
			NodeID: rt.taskNodeID, TimeoutSeconds: rt.timeoutSeconds,
			AgentRunSpec: json.RawMessage(rt.taskPayload),
		})
	}
	return tasks
}

// sweepExpiredTasks transitions timed-out tasks and expired leases within the
// given transaction. Called during heartbeat claim to piggyback on the
// existing 2s interval.
func sweepExpiredTasks(tx *sql.Tx, nodeID string, now string) {
	// Tasks that exceeded timeout_seconds.
	if _, err := tx.Exec(
		`UPDATE agent_tasks SET status = ?, finished_at = ?, updated_at = ?
		 WHERE node_id = ? AND status IN (?, ?, ?)
		 AND (strftime('%s', ?) - strftime('%s', created_at)) > timeout_seconds`,
		TaskStatusTimedOut, now, now, nodeID,
		TaskStatusPending, TaskStatusClaimed, TaskStatusInProgress,
		now,
	); err != nil {
		// AUD-007: Log sweep errors instead of silently discarding.
		log.Error("sweep.task_timeout.failed", "node_id", nodeID, "error", err)
	}

	// Instances for timed-out tasks → failed.
	if _, err := tx.Exec(
		`UPDATE model_instances SET actual_state = ?, updated_at = ?
		 WHERE id IN (SELECT instance_id FROM agent_tasks WHERE status = ? AND node_id = ? AND instance_id != '')`,
		InstanceStateFailed, now, TaskStatusTimedOut, nodeID,
	); err != nil {
		log.Error("sweep.instance_fail.failed", "node_id", nodeID, "error", err)
	}

	// Leases past expires_at → failed (reserved leases only).
	// Active leases are only failed by the server-side sweep when the node is offline.
	// Query before UPDATE for per-lease cleanup logging.
	leaseRows, lerr := tx.Query(
		`SELECT id, gpu_id, instance_id, deployment_id, node_id FROM gpu_leases
		 WHERE expires_at IS NOT NULL AND expires_at < ? AND status = ?`,
		now, LeaseReserved,
	)
	if lerr != nil {
		log.Error("sweep.lease_query.failed", "node_id", nodeID, "error", lerr)
	} else {
		type leaseRec struct{ id, gpuID, instanceID, deploymentID, leaseNodeID string }
		var expiredLeases []leaseRec
		for leaseRows.Next() {
			var lr leaseRec
			leaseRows.Scan(&lr.id, &lr.gpuID, &lr.instanceID, &lr.deploymentID, &lr.leaseNodeID)
			expiredLeases = append(expiredLeases, lr)
		}
		leaseRows.Close()

		if _, err := tx.Exec(
			`UPDATE gpu_leases SET status = ?, updated_at = ?
			 WHERE expires_at IS NOT NULL AND expires_at < ? AND status = ?`,
			LeaseFailed, now, now, LeaseReserved,
		); err != nil {
			log.Error("sweep.lease_fail.failed", "node_id", nodeID, "error", err)
		} else {
			for _, lr := range expiredLeases {
				log.StateTransition(context.Background(), "lease.sweep", "gpu_lease", lr.id, LeaseReserved, LeaseFailed,
					"gpu_id", lr.gpuID, "instance_id", lr.instanceID, "deployment_id", lr.deploymentID, "node_id", lr.leaseNodeID)
			}
			if len(expiredLeases) > 0 {
				log.Info("gpu_lease.sweep.expired", "count", len(expiredLeases), "node_id", nodeID)
			}
		}
	}

	// Deployments for failed instances → failed.
	if _, err := tx.Exec(
		`UPDATE model_deployments SET status = 'failed', updated_at = ?
		 WHERE id IN (SELECT deployment_id FROM model_instances WHERE actual_state = ?)`,
		now, InstanceStateFailed,
	); err != nil {
		log.Error("sweep.deployment_fail.failed", "node_id", nodeID, "error", err)
	}
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

// HandleGetNodeDockerImages proxies to the Agent's /docker-images endpoint.
func (h *AgentHandler) HandleGetNodeDockerImages(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	var addr string
	var port int
	err := h.DB.QueryRow(
		`SELECT advertised_address, metrics_port FROM nodes WHERE id = ?`, nodeID,
	).Scan(&addr, &port)
	if err != nil {
		http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
		return
	}
	if addr == "" || port == 0 {
		http.Error(w, `{"error":"node has no advertised address or metrics port"}`, http.StatusBadRequest)
		return
	}
	agentURL := fmt.Sprintf("http://%s:%d/docker-images", addr, port)
	resp, err := http.Get(agentURL)
	if err != nil {
		// AUD-008: Return 502 when agent is unreachable so caller can distinguish
		// "no images" from "agent down".
		log.Warn("failed to query agent docker images", "node_id", nodeID, "url", agentURL, "error", err)
		writeError(w, http.StatusBadGateway, "agent unreachable")
		return
	}
	defer resp.Body.Close()
	var images []interface{}
	json.NewDecoder(resp.Body).Decode(&images)
	if images == nil {
		images = []interface{}{}
	}
	writeJSON(w, http.StatusOK, images)
}

// MarkOfflineNodes marks nodes as offline if they haven't sent a heartbeat
// within the given threshold. Returns the count of nodes marked offline.
// P0-009: Node auto-offline implementation.
func (h *AgentHandler) MarkOfflineNodes(threshold time.Duration) (int, error) {
	cutoff := time.Now().Add(-threshold).Format(time.RFC3339)

	// Query nodes that will be marked offline for transition logging.
	rows, err := h.DB.Query(`SELECT id FROM nodes WHERE status = 'online' AND last_heartbeat_at < ?`, cutoff)
	if err != nil {
		log.Error("mark offline: query nodes error", "error", err)
		return 0, err
	}
	var offlineIDs []string
	for rows.Next() {
		var nid string
		rows.Scan(&nid)
		offlineIDs = append(offlineIDs, nid)
	}
	rows.Close()

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
	for _, nid := range offlineIDs {
		log.StateTransition(context.Background(), "node.health_check", "node", nid, "online", "offline",
			"reason", "heartbeat_timeout", "threshold", threshold.String())
	}
	if n > 0 {
		log.Info("nodes marked offline", "count", n, "threshold", threshold.String())
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

	// Safety: reject transfer if node has active GPU leases.
	var activeLeaseCount int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM gpu_leases WHERE node_id = ? AND status IN ('reserved','active')`, nodeID).Scan(&activeLeaseCount); err != nil {
		log.Error("transfer safety check: query active leases failed", "node_id", nodeID, "error", err)
		http.Error(w, `{"error":"internal error: cannot verify lease status"}`, http.StatusInternalServerError)
		return
	}
	if activeLeaseCount > 0 {
		http.Error(w, `{"error":"node has active GPU leases — release them before transferring"}`, http.StatusConflict)
		return
	}

	// Safety: reject transfer if node has running/starting/stopping instances.
	var activeInstanceCount int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM model_instances WHERE node_id = ? AND actual_state IN ('pending','starting','running','stopping')`, nodeID).Scan(&activeInstanceCount); err != nil {
		log.Error("transfer safety check: query active instances failed", "node_id", nodeID, "error", err)
		http.Error(w, `{"error":"internal error: cannot verify instance status"}`, http.StatusInternalServerError)
		return
	}
	if activeInstanceCount > 0 {
		http.Error(w, `{"error":"node has active model instances — stop them before transferring"}`, http.StatusConflict)
		return
	}

	// Update node tenant and audit log in a single transaction for atomicity.
	tx, err := h.DB.Begin()
	if err != nil {
		log.Error("transfer: begin tx failed", "node_id", nodeID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	if _, err := tx.Exec(`UPDATE nodes SET tenant_id = ?, updated_at = ? WHERE id = ?`,
		req.TenantID, now, nodeID); err != nil {
		log.Error("update node tenant error", "node_id", nodeID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// REVIEW-008: Transfer GPU tenant ownership in same transaction.
	if _, err := tx.Exec(`UPDATE gpu_devices SET tenant_id = ?, updated_at = ? WHERE node_id = ?`,
		req.TenantID, now, nodeID); err != nil {
		log.Error("transfer: update gpu tenant_id failed", "node_id", nodeID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Audit log — if this fails, the whole transfer rolls back.
	auditID := uuid.NewString()
	detail := fmt.Sprintf(`{"from_tenant_id":"%s","to_tenant_id":"%s","reason":"%s"}`,
		currentTenant, req.TenantID, req.Reason)
	if _, err := tx.Exec(`INSERT INTO audit_logs (id, action, entity_type, entity_id, detail, operator_user_id, created_at)
		VALUES (?, 'transfer_tenant', 'node', ?, ?, ?, ?)`,
		auditID, nodeID, detail, info.UserID, now); err != nil {
		log.Error("transfer: audit log insert failed", "node_id", nodeID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Error("transfer: commit failed", "node_id", nodeID, "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	log.Info("node tenant transferred",
		"node_id", nodeID,
		"from_tenant", currentTenant,
		"to_tenant", req.TenantID,
		"operator", info.UserID,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "tenant_id": req.TenantID})
}

// HandleTaskResult processes task completion reports from agents.
func (h *AgentHandler) HandleTaskResult(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	taskID := r.PathValue("id")

	// Read operation_id from agent's report for cross-component correlation.
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	status := strVal(req, "status", "completed")
	opID := strVal(req, "operation_id", "")
	now := time.Now().UTC().Format(time.RFC3339)

	// REVIEW-004: Validate task lease and generation before accepting result.
	// Reject stale, duplicate, or old-generation results.
	var taskStatus, taskOpID, taskLeaseOwner string
	var taskGeneration int
	err := h.DB.QueryRow(`SELECT status, COALESCE(operation_id,''), COALESCE(lease_owner,''), generation
		FROM agent_tasks WHERE id = ?`, taskID).Scan(&taskStatus, &taskOpID, &taskLeaseOwner, &taskGeneration)
	if err != nil {
		log.Warn("task_result: task not found", "task_id", taskID, "error", err)
		w.WriteHeader(http.StatusOK) // ack to prevent agent retry
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "task_not_found"})
		return
	}
	if taskStatus == "completed" || taskStatus == "failed" || taskStatus == "timed_out" {
		log.Info("task_result: duplicate/stale result ignored",
			"task_id", taskID, "task_status", taskStatus, "op_id", opID)
		w.WriteHeader(http.StatusOK)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "task_already_terminal"})
		return
	}
	// If the task was claimed by a different agent, reject.
	if taskLeaseOwner != "" {
		agentID := strVal(req, "agent_id", "")
		if agentID != "" && taskLeaseOwner != agentID {
			log.Warn("task_result: lease owner mismatch",
				"task_id", taskID, "lease_owner", taskLeaseOwner, "reporter", agentID)
			w.WriteHeader(http.StatusOK)
			writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "lease_owner_mismatch"})
			return
		}
	}

	// Resolve instance_id and deployment_id from the task.
	var taskInstanceID, taskDeploymentID, taskNodeID string
	h.DB.QueryRow(`SELECT COALESCE(instance_id,''), COALESCE(deployment_id,''), COALESCE(node_id,'') FROM agent_tasks WHERE id = ?`, taskID).Scan(&taskInstanceID, &taskDeploymentID, &taskNodeID)
	if taskInstanceID == "" {
		taskInstanceID = strVal(req, "instance_id", "")
	}

	switch status {
	case "completed", "success":
		h.DB.Exec(`UPDATE agent_tasks SET status = 'completed', finished_at = ? WHERE id = ?`, now, taskID)

		if taskInstanceID != "" {
			containerID := strVal(req, "container_id", "")
			errorMsg := strVal(req, "error_message", strVal(req, "error", ""))
			success := boolVal(req, "success", true) && errorMsg == ""
			actualState := "running"
			prevActualState := "pending"
			if !success {
				actualState = "failed"
			}
			// Get previous instance state for transition logging.
			h.DB.QueryRow(`SELECT COALESCE(actual_state,'pending') FROM model_instances WHERE id = ?`, taskInstanceID).Scan(&prevActualState)

			// Build endpoint_url from instance host_port.
			var hostPort int
			h.DB.QueryRow(`SELECT host_port FROM model_instances WHERE id = ?`, taskInstanceID).Scan(&hostPort)
			endpointURL := ""
			if hostPort > 0 {
				endpointURL = fmt.Sprintf("http://127.0.0.1:%d", hostPort)
			}

			h.DB.Exec(`UPDATE model_instances SET actual_state = ?, container_id = ?, endpoint_url = ?, started_at = ? WHERE id = ?`,
				actualState, containerID, endpointURL, now, taskInstanceID)

			// Activate leases.
			lr, lerr := h.DB.Exec(`UPDATE gpu_leases SET status = 'active', activated_at = ? WHERE instance_id = ? AND status = 'reserved'`, now, taskInstanceID)
			if lerr != nil {
				log.Error("gpu_lease.activate.failed", "instance_id", taskInstanceID, "error", lerr)
			} else if ln, _ := lr.RowsAffected(); ln > 0 {
				log.StateTransition(r.Context(), "task.result", "gpu_lease", taskInstanceID, "reserved", "active",
					"instance_id", taskInstanceID, "count", ln)
				log.Info("gpu_lease.activated", "instance_id", taskInstanceID, "count", ln)
			}

			// State transition logging.
			log.StateTransition(r.Context(), "task.result", "instance", taskInstanceID, prevActualState, actualState,
				"task_id", taskID, "deployment_id", taskDeploymentID, "node_id", taskNodeID,
				"container_id", containerID, "duration_ms", log.DurationMs(startTime))
			if opID != "" {
				log.Info("task.result.processed",
					"operation_id", opID, "task_id", taskID, "instance_id", taskInstanceID,
					"state", actualState, "container_id", containerID, "endpoint_url", endpointURL,
					"duration_ms", log.DurationMs(startTime))
			}
		}
	case "failed", "error":
		h.DB.Exec(`UPDATE agent_tasks SET status = 'failed', finished_at = ? WHERE id = ?`, now, taskID)
		errorMsg := strVal(req, "error_message", strVal(req, "error", "unknown"))
		if taskInstanceID != "" {
			var prevActualState string
			h.DB.QueryRow(`SELECT COALESCE(actual_state,'pending') FROM model_instances WHERE id = ?`, taskInstanceID).Scan(&prevActualState)
			h.DB.Exec(`UPDATE model_instances SET actual_state = 'failed', last_error = ? WHERE id = ?`, errorMsg, taskInstanceID)
			h.DB.Exec(`UPDATE gpu_leases SET status = 'failed' WHERE instance_id = ? AND status = 'reserved'`, taskInstanceID)

			log.StateTransition(r.Context(), "task.result", "instance", taskInstanceID, prevActualState, "failed",
				"task_id", taskID, "deployment_id", taskDeploymentID, "node_id", taskNodeID,
				"error", errorMsg, "duration_ms", log.DurationMs(startTime))
			if opID != "" {
				log.Error("task.result.failed",
					"operation_id", opID, "task_id", taskID, "instance_id", taskInstanceID,
					"error", errorMsg, "duration_ms", log.DurationMs(startTime))
			} else {
				log.Error("task.result.failed",
					"task_id", taskID, "instance_id", taskInstanceID,
					"error", errorMsg, "duration_ms", log.DurationMs(startTime))
			}
		} else {
			log.Error("task.result.failed", "task_id", taskID, "error", errorMsg)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
