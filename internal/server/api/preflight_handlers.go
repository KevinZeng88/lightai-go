package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HandlePreflightDeployments computes candidate nodes for a deployment
// without requiring an existing deployment ID. It finds the intersection of
// nodes that have both a valid ModelLocation and a ready NodeBackendRuntime.
func (h *AgentHandler) HandlePreflightDeployments(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelArtifactID       string   `json:"model_artifact_id"`
		BackendRuntimeID      string   `json:"backend_runtime_id"`
		NodeBackendRuntimeID  string   `json:"node_backend_runtime_id"`
		NodeID                string   `json:"node_id"`
		GPUIds                []string `json:"gpu_ids"`
		HostPort              int      `json:"host_port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ModelArtifactID == "" {
		writeError(w, http.StatusBadRequest, "model_artifact_id is required")
		return
	}

	// Resolve backend_runtime_id and node_id from node_backend_runtime_id if provided.
	if req.NodeBackendRuntimeID != "" {
		var nbrBackendRuntimeID, nbrNodeID, nbrStatus string
		if err := h.DB.QueryRow(
			`SELECT backend_runtime_id, node_id, status FROM node_backend_runtimes WHERE id = ?`,
			req.NodeBackendRuntimeID,
		).Scan(&nbrBackendRuntimeID, &nbrNodeID, &nbrStatus); err != nil {
			writeError(w, http.StatusBadRequest, "node_backend_runtime_id not found")
			return
		}
		if nbrStatus != "ready" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"can_run": false, "candidate_nodes": []interface{}{},
				"errors":  []string{fmt.Sprintf("node backend runtime is not ready (status=%s)", nbrStatus)},
				"warnings": []string{},
			})
			return
		}
		// If backend_runtime_id is also provided, validate consistency.
		if req.BackendRuntimeID != "" && req.BackendRuntimeID != nbrBackendRuntimeID {
			writeError(w, http.StatusBadRequest, "backend_runtime_id does not match node_backend_runtime_id")
			return
		}
		req.BackendRuntimeID = nbrBackendRuntimeID
		if req.NodeID == "" {
			req.NodeID = nbrNodeID
		} else if req.NodeID != nbrNodeID {
			writeError(w, http.StatusBadRequest, "node_id does not match node_backend_runtime_id node")
			return
		}
	}

	if req.BackendRuntimeID == "" {
		writeError(w, http.StatusBadRequest, "backend_runtime_id or node_backend_runtime_id is required")
		return
	}

	tid := tenantID(r)
	errors := []string{}
	warnings := []string{}
	var candidateNodes []map[string]interface{}

	// Find nodes that have both a ModelLocation and a ready NodeBackendRuntime
	rows, err := h.DB.Query(`SELECT DISTINCT ml.node_id FROM model_locations ml
		JOIN node_backend_runtimes nbr ON nbr.node_id = ml.node_id
		WHERE ml.model_artifact_id = ?
		AND ml.verification_status IN ('verified','warning','manually_accepted')
		AND nbr.backend_runtime_id = ? AND nbr.status = 'ready'`,
		req.ModelArtifactID, req.BackendRuntimeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()

	var nodeIDs []string
	for rows.Next() {
		var nid string
		rows.Scan(&nid)
		// Tenant scope for non-admin users
		if !isPlatformAdmin(r) {
			var nodeTid string
			h.DB.QueryRow("SELECT tenant_id FROM nodes WHERE id = ?", nid).Scan(&nodeTid)
			if nodeTid != tid && nodeTid != "" {
				continue
			}
		}
		nodeIDs = append(nodeIDs, nid)
	}

	if len(nodeIDs) == 0 {
		errors = append(errors, "no node has both a valid model location and a ready runtime")
	} else {
		for _, nid := range nodeIDs {
			info := map[string]interface{}{"node_id": nid, "status": "ready"}
			var gpuCount int
			h.DB.QueryRow("SELECT COUNT(*) FROM gpu_devices WHERE node_id = ? AND status = 'available'", nid).Scan(&gpuCount)
			if gpuCount == 0 {
				info["warnings"] = []string{"no available GPU"}
			}
			candidateNodes = append(candidateNodes, info)
		}
	}

	canRun := len(errors) == 0
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"can_run":         canRun,
		"candidate_nodes": candidateNodes,
		"errors":          errors,
		"warnings":        warnings,
	})
}
