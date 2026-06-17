package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// HandlePatchModelLocation updates a model location's status or metadata.
func (h *AgentHandler) HandlePatchModelLocation(w http.ResponseWriter, r *http.Request) {
	locationID := r.PathValue("location_id")
	existing := h.getModelLocationJSON(locationID)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_at = ?"}
	args := []interface{}{now}
	for _, f := range []string{"verification_status", "match_status", "model_root", "relative_path", "absolute_path"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	if v, ok := req["disabled"]; ok {
		if b, ok := v.(bool); ok && b {
			sets = append(sets, "status = 'disabled'")
		}
	}
	args = append(args, locationID)
	if _, err := h.DB.Exec(`UPDATE model_locations SET `+joinSets(sets)+` WHERE id = ?`, args...); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, h.getModelLocationJSON(locationID))
}

// HandleDeleteModelLocation removes a model location, blocking if it is
// referenced by an active (pending/starting/running) instance.
func (h *AgentHandler) HandleDeleteModelLocation(w http.ResponseWriter, r *http.Request) {
	locationID := r.PathValue("location_id")
	existing := h.getModelLocationJSON(locationID)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	artifactID := existing["model_artifact_id"].(string)
	nodeID := existing["node_id"].(string)
	var instanceCount int
	h.DB.QueryRow(`SELECT COUNT(*) FROM model_instances mi
		JOIN model_deployments md ON md.id = mi.deployment_id
		WHERE md.model_artifact_id = ? AND mi.node_id = ? AND mi.actual_state IN ('pending','starting','running')`,
		artifactID, nodeID).Scan(&instanceCount)
	if instanceCount > 0 {
		writeError(w, http.StatusConflict, "model location is used by active instances")
		return
	}
	if _, err := h.DB.Exec(`DELETE FROM model_locations WHERE id = ?`, locationID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
