package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HandlePreflightDeployments computes candidate nodes for a deployment
// without requiring an existing deployment ID. It validates the given
// NodeBackendRuntime and confirms the target node has a valid ModelLocation.
// backend_runtime_id is NOT accepted — BackendRuntime is a template, not a
// deployable object. Use node_backend_runtime_id.
func (h *AgentHandler) HandlePreflightDeployments(w http.ResponseWriter, r *http.Request) {
	// Use a raw decode first to detect and reject backend_runtime_id.
	var rawBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if _, ok := rawBody["backend_runtime_id"]; ok {
		writeError(w, http.StatusBadRequest,
			"BackendRuntime is a template and cannot be used for deployment. Use node_backend_runtime_id.")
		return
	}

	var req struct {
		ModelArtifactID      string   `json:"model_artifact_id"`
		NodeBackendRuntimeID string   `json:"node_backend_runtime_id"`
		NodeID               string   `json:"node_id"`
		GPUIds               []string `json:"gpu_ids"`
		HostPort             int      `json:"host_port"`
	}
	// Re-marshal to typed struct for clean parsing.
	bodyBytes, _ := json.Marshal(rawBody)
	json.Unmarshal(bodyBytes, &req)

	if req.ModelArtifactID == "" {
		writeError(w, http.StatusBadRequest, "model_artifact_id is required")
		return
	}
	if req.NodeBackendRuntimeID == "" {
		writeError(w, http.StatusBadRequest, "node_backend_runtime_id is required")
		return
	}

	// Resolve NBR.
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
	if req.NodeID != "" && req.NodeID != nbrNodeID {
		writeError(w, http.StatusBadRequest, "node_id does not match node_backend_runtime_id node")
		return
	}
	req.NodeID = nbrNodeID

	tid := tenantID(r)
	errors := []string{}
	warnings := []string{}
	var candidateNodes []map[string]interface{}

	// Verify the NBR's node has a valid ModelLocation for the given artifact.
	var mlID string
	h.DB.QueryRow(`SELECT id FROM model_locations
		WHERE model_artifact_id = ? AND node_id = ?
		AND verification_status IN ('verified','warning','manually_accepted')
		ORDER BY updated_at DESC LIMIT 1`,
		req.ModelArtifactID, req.NodeID).Scan(&mlID)
	if mlID == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"can_run":         false,
			"candidate_nodes": []interface{}{},
			"errors":          []string{"no model location found on the NBR's node for this model artifact"},
			"warnings":        warnings,
		})
		return
	}

	// Tenant scope check.
	if !isPlatformAdmin(r) {
		var nodeTid string
		h.DB.QueryRow("SELECT tenant_id FROM nodes WHERE id = ?", req.NodeID).Scan(&nodeTid)
		if nodeTid != tid && nodeTid != "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"can_run":         false,
				"candidate_nodes": []interface{}{},
				"errors":          []string{"tenant does not have access to the NBR's node"},
				"warnings":        warnings,
			})
			return
		}
	}

	info := map[string]interface{}{"node_id": req.NodeID, "status": "ready"}
	var gpuCount int
	h.DB.QueryRow("SELECT COUNT(*) FROM gpu_devices WHERE node_id = ? AND status = 'available'", req.NodeID).Scan(&gpuCount)
	if gpuCount == 0 {
		info["warnings"] = []string{"no available GPU"}
	}
	candidateNodes = append(candidateNodes, info)

	canRun := len(errors) == 0
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"can_run":         canRun,
		"candidate_nodes": candidateNodes,
		"errors":          errors,
		"warnings":        warnings,
	})
}
