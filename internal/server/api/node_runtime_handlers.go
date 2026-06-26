package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/authz"

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
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	newID := uuid.NewString()
	now := time.Now().Format(time.RFC3339)
	sourceName := strVal(original, "display_name", "")
	if sourceName == "" {
		sourceName = strVal(original, "name", "")
	}
	newName := strings.TrimSpace(strVal(req, "name", ""))
	if newName == "" {
		newName = sourceName + "-copy"
	}
	newName = h.uniqueRuntimeName(tid, newName)
	newDisplayName := strings.TrimSpace(strVal(req, "display_name", ""))
	if newDisplayName == "" {
		newDisplayName = newName
	}
	configSet := copyConfigSet(rawJSONString(original["config_set_json"], "{}"))
	if v := strVal(req, "image_ref", ""); v != "" {
		setConfigValue(configSet, "launcher.image", v, "BackendRuntime", newID, "clone_override")
	}
	if v, ok := req["docker_options"]; ok {
		setConfigValue(configSet, "launcher.docker_options", v, "BackendRuntime", newID, "clone_override")
	}
	if v, ok := req["env"]; ok {
		setConfigValue(configSet, "runtime.env", v, "BackendRuntime", newID, "clone_override")
	}
	if v, ok := req["model_mount"]; ok {
		setConfigValue(configSet, "runtime.model_mount", v, "BackendRuntime", newID, "clone_override")
	}
	if v, ok := req["health_check"]; ok {
		setConfigValue(configSet, "runtime.health", v, "BackendRuntime", newID, "clone_override")
	}
	if v, ok := req["entrypoint"]; ok {
		setConfigValue(configSet, "launcher.entrypoint", v, "BackendRuntime", newID, "clone_override")
	}
	if v, ok := req["command"]; ok {
		setConfigValue(configSet, "launcher.command", v, "BackendRuntime", newID, "clone_override")
	}
	if incoming, ok := req["config_set"].(map[string]interface{}); ok {
		configSet = incoming
	}
	vendor := strVal(req, "vendor", strVal(original, "vendor", ""))
	sourceMetadata := map[string]interface{}{
		"source_type":               "backend_runtime_clone",
		"source_backend_runtime_id": originalID,
		"source_runtime_name":       strVal(original, "name", ""),
		"source_runtime_revision":   strVal(original, "updated_at", ""),
		"copy_semantics":            "copy_on_create",
	}
	_, err := h.DB.Exec(`INSERT INTO backend_runtimes (id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, is_builtin, is_editable, tenant_id, slug, managed_by, source, catalog_version, checksum, status, config_set_json, source_metadata_json, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'docker', 0, 1, ?, ?, 'user', 'clone', 'configset-v1', ?, 'active', ?, ?, ?, ?)`,
		newID, newName, newDisplayName,
		strVal(original, "backend_id", ""), strVal(original, "backend_version_id", ""),
		sourceName, vendor, tid, slugify(newName), checksumString(configSetJSON(configSet)), configSetJSON(configSet), jsonString(sourceMetadata), now, now)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.clone", "db_write", opStart, err, "original_id", originalID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Note: Cloning a BackendRuntime does NOT auto-create NodeBackendRuntime records.
	// NodeBackendRuntime must be explicitly enabled by the user via the enable-on-node flow
	// (POST /api/v1/nodes/{id}/backend-runtimes/enable). This ensures the user confirms
	// node assignment, Docker image availability, and node-level config before deployment.

	log.OperationCompleted(ctx, "backend_runtime.clone", opStart, "id", newID, "original_id", originalID, "tenant_id", tid)
	writeJSON(w, http.StatusCreated, h.getBackendRuntimeJSON(newID))
}

func (h *AgentHandler) uniqueRuntimeName(tenantID, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "runtime-copy"
	}
	candidate := base
	for i := 2; ; i++ {
		var count int
		_ = h.DB.QueryRow(`SELECT COUNT(*) FROM backend_runtimes WHERE tenant_id = ? AND name = ?`, tenantID, candidate).Scan(&count)
		if count == 0 {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

// HandlePatchNodeBackendRuntime updates node-level fields on a NodeBackendRuntime.
func (h *AgentHandler) HandlePatchNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
	nbrID := r.PathValue("nbr_id")
	if !authz.CheckNBRTenant(r, h.DB.DB, nbrID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if _, ok := req["image_present"]; ok {
		writeError(w, http.StatusBadRequest, "image_present is server/agent verified evidence; call check-request")
		return
	}
	if _, ok := req["docker_available"]; ok {
		writeError(w, http.StatusBadRequest, "docker_available is server/agent verified evidence; call check-request")
		return
	}
	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_at = ?"}
	args := []interface{}{now}
	needsRecheck := false
	for _, f := range []string{"display_name", "image_ref", "image_id", "image_digest", "driver_version", "toolkit_version"} {
		if v, ok := req[f]; ok {
			if f == "display_name" {
				if s, ok := v.(string); ok {
					v = strings.TrimSpace(s)
					if v == "" {
						writeError(w, http.StatusBadRequest, "display_name is required")
						return
					}
				}
			}
			sets = append(sets, f+" = ?")
			args = append(args, v)
			// Editing image-ref fields invalidates ready status
			if f == "image_ref" || f == "image_id" || f == "image_digest" {
				needsRecheck = true
			}
		}
	}
	if v, ok := req["config_set"]; ok {
		sets = append(sets, "config_set_json = ?")
		args = append(args, jsonString(v))
		needsRecheck = true
	}
	if v, ok := req["config_set_json"]; ok {
		sets = append(sets, "config_set_json = ?")
		args = append(args, jsonString(v))
		needsRecheck = true
	}
	if v, ok := req["source_metadata"]; ok {
		sets = append(sets, "source_metadata_json = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["source_metadata_json"]; ok {
		sets = append(sets, "source_metadata_json = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["device_check_json"]; ok {
		sets = append(sets, "device_check_json = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["disabled"]; ok {
		if b, ok := v.(bool); ok && b {
			sets = append(sets, "status = 'disabled'")
			needsRecheck = false // explicit disable overrides needs_check
		}
	}
	if needsRecheck {
		sets = append(sets, "status = 'needs_check'")
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
	if !authz.CheckNBRTenant(r, h.DB.DB, nbrID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
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
