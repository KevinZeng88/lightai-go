package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
		AcceleratorIds       []string `json:"accelerator_ids"`
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
	// R-003: Use same deployability check as dry-run/start. Accepts ready + ready_with_warnings.
	if !isNBRDeployable(nbrStatus) {
		reason := nbrDisabledReason(nbrStatus, "")
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"can_run":           false,
			"candidate_nodes":   []interface{}{},
			"errors":            []map[string]interface{}{{"code": "nbr_not_deployable", "message": fmt.Sprintf("NBR status=%s: %s", nbrStatus, reason), "field": "node_backend_runtime_id", "severity": "error"}},
			"warnings":          []map[string]interface{}{},
			"resolved_run_plan": nil,
		})
		return
	}
	// Collect NBR warnings for ready_with_warnings case.
	var preflightWarnings []map[string]interface{}
	if nbrStatus == "ready_with_warnings" {
		preflightWarnings = append(preflightWarnings, map[string]interface{}{
			"code": "nbr_ready_with_warnings", "message": "NBR is ready with warnings — deployment may succeed but has non-blocking issues",
			"field": "node_backend_runtime_id", "severity": "warning",
		})
	}
	if req.NodeID != "" && req.NodeID != nbrNodeID {
		writeError(w, http.StatusBadRequest, "node_id does not match node_backend_runtime_id node")
		return
	}
	req.NodeID = nbrNodeID

	tid := tenantID(r)
	errors := []map[string]interface{}{}
	warnings := preflightWarnings

	// R-012: Reject replicas > 1 until supported.
	if _, ok := rawBody["replicas"]; ok {
		n := intVal(rawBody, "replicas", 1)
		if n > 1 {
			errors = append(errors, errEntry("replicas_unsupported", "multi-replica deployments are not yet supported", "replicas", "error"))
		}
	}

	// Verify the NBR's node has a deployable ModelLocation for the given artifact.
	if loc, _, reason := h.findDeployableModelLocation(req.ModelArtifactID, req.NodeID); loc == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"can_run":           false,
			"candidate_nodes":   []interface{}{},
			"errors":            []map[string]interface{}{{"code": "model_location_missing", "message": reason, "field": "model_artifact_id", "severity": "error"}},
			"warnings":          warnings,
			"resolved_run_plan": nil,
		})
		return
	}

	// Tenant scope check.
	if !isPlatformAdmin(r) {
		var nodeTid string
		h.DB.QueryRow("SELECT tenant_id FROM nodes WHERE id = ?", req.NodeID).Scan(&nodeTid)
		if nodeTid != tid && nodeTid != "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"can_run":           false,
				"candidate_nodes":   []interface{}{},
				"errors":            []map[string]interface{}{{"code": "tenant_mismatch", "message": "tenant does not have access to the NBR's node", "field": "node_id", "severity": "error"}},
				"warnings":          warnings,
				"resolved_run_plan": nil,
			})
			return
		}
	}

	info := map[string]interface{}{"node_id": req.NodeID, "status": nbrStatus}
	var gpuCount int
	h.DB.QueryRow("SELECT COUNT(*) FROM gpu_devices WHERE node_id = ? AND status = 'available'", req.NodeID).Scan(&gpuCount)
	if gpuCount == 0 {
		warnings = append(warnings, map[string]interface{}{"code": "no_available_gpu", "message": "no available GPU found on node", "field": "node_id", "severity": "warning"})
	}
	candidateNodes := []map[string]interface{}{info}

	canRun := len(errors) == 0
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"can_run":           canRun,
		"candidate_nodes":   candidateNodes,
		"errors":            errors,
		"warnings":          warnings,
		"resolved_run_plan": nil,
	})
}

// errEntry creates a structured error entry for preflight responses.
func errEntry(code, message, field, severity string) map[string]interface{} {
	blocking := severity == "error"
	return map[string]interface{}{
		"code":     code,
		"message":  message,
		"field":    field,
		"key":      field,
		"path":     fieldPath(field),
		"reason":   message,
		"source":   "preflight",
		"severity": severity,
		"blocking": blocking,
	}
}

func fieldPath(field string) []string {
	if field == "" {
		return nil
	}
	return strings.Split(field, ".")
}
