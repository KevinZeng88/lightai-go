package api

import (
	"encoding/json"
	"net/http"
	"time"

	"lightai-go/internal/common/log"

	"github.com/google/uuid"
)

// HandleCloneBackendRuntime creates a user-managed copy of a system BackendRuntime.
func (h *AgentHandler) HandleCloneBackendRuntime(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "backend_runtime.clone")
	originalID := r.PathValue("id")
	original := h.getBackendRuntimeJSON(originalID)
	if original == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	tid := tenantID(r)
	newID := uuid.NewString()
	now := time.Now().Format(time.RFC3339)
	newName := strVal(original, "name", "") + "-clone-" + newID[:8]
	_, err := h.DB.Exec(`INSERT INTO backend_runtimes (id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, slug, managed_by, source, status, created_at, updated_at)
		SELECT ?, ?, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, 0, 1, ?, slug, 'user', 'clone', status, ?, ?
		FROM backend_runtimes WHERE id = ?`,
		newID, newName, tid, now, now, originalID)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.clone", "db_write", opStart, err, "original_id", originalID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "backend_runtime.clone", opStart, "id", newID, "original_id", originalID, "tenant_id", tid)
	writeJSON(w, http.StatusCreated, h.getBackendRuntimeJSON(newID))
}

// HandlePatchNodeBackendRuntime updates node-level fields on a NodeBackendRuntime.
func (h *AgentHandler) HandlePatchNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
	nbrID := r.PathValue("nbr_id")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_at = ?"}
	args := []interface{}{now}
	for _, f := range []string{"image_ref", "image_id", "image_digest", "driver_version", "toolkit_version"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	if v, ok := req["image_present"]; ok {
		if b, ok := v.(bool); ok {
			sets = append(sets, "image_present = ?")
			args = append(args, boolInt(b))
		}
	}
	if v, ok := req["disabled"]; ok {
		if b, ok := v.(bool); ok && b {
			sets = append(sets, "status = 'disabled'")
		}
	}
	args = append(args, nbrID)
	if _, err := h.DB.Exec(`UPDATE node_backend_runtimes SET `+joinSets(sets)+` WHERE id = ?`, args...); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// HandleDeleteNodeBackendRuntime removes a node backend runtime.
// Blocks if active instances reference it.
func (h *AgentHandler) HandleDeleteNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
	nbrID := r.PathValue("nbr_id")
	var nbrNodeID, nbrRuntimeID string
	if err := h.DB.QueryRow(`SELECT node_id, backend_runtime_id FROM node_backend_runtimes WHERE id = ?`, nbrID).Scan(&nbrNodeID, &nbrRuntimeID); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var instanceCount int
	h.DB.QueryRow(`SELECT COUNT(*) FROM resolved_run_plans rp
		JOIN model_instances mi ON mi.id = rp.instance_id
		WHERE rp.backend_runtime_id = ? AND rp.node_backend_runtime_id = ?
		AND mi.actual_state IN ('pending','starting','running')`,
		nbrRuntimeID, nbrID).Scan(&instanceCount)
	if instanceCount > 0 {
		writeError(w, http.StatusConflict, "node runtime is used by active instances")
		return
	}
	h.DB.Exec(`DELETE FROM node_backend_runtimes WHERE id = ?`, nbrID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
