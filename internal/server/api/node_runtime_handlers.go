package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
		newName = h.uniqueRuntimeName(tid, sourceName+"-copy")
	}
	newDisplayName := strings.TrimSpace(strVal(req, "display_name", ""))
	if newDisplayName == "" {
		newDisplayName = newName
	}
		// Accept overrides from request body for key config fields.
		imageName := strVal(req, "image_name", strVal(original, "image_name", ""))
		vendor := strVal(req, "vendor", strVal(original, "vendor", ""))
		dockerJSON := jsonFieldRaw(req, "docker_json", original["docker_json"])
		argsOverride := jsonFieldRaw(req, "args_override_json", original["args_override_json"])
		defaultEnv := jsonFieldRaw(req, "default_env_json", original["default_env_json"])
		entryOverride := jsonFieldRaw(req, "entrypoint_override_json", original["entrypoint_override_json"])
		_, err := h.DB.Exec(`INSERT INTO backend_runtimes (id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, slug, managed_by, source, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'docker', ?, 'if_not_present', ?, ?, ?, ?, ?, ?, 0, 1, ?, ?, 'user', 'clone', 'active', ?, ?)`,
			newID, newName, newDisplayName,
			strVal(original, "backend_id", ""), strVal(original, "backend_version_id", ""),
			sourceName, vendor, imageName,
			jsonString(entryOverride), jsonString(argsOverride),
			jsonString(defaultEnv), jsonString(dockerJSON),
			jsonString(original["model_mount_json"]),
			jsonString(original["health_check_override_json"]),
			tid, strVal(original, "slug", ""), now, now)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.clone", "db_write", opStart, err, "original_id", originalID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Auto-create NodeBackendRuntime for online nodes matching the vendor,
	// so the cloned runtime appears in the deployment wizard selector immediately.
	h.autoEnableClonedRuntime(newID, vendor, imageName, tid, now)

	log.OperationCompleted(ctx, "backend_runtime.clone", opStart, "id", newID, "original_id", originalID, "tenant_id", tid)
	writeJSON(w, http.StatusCreated, h.getBackendRuntimeJSON(newID))
}
// jsonFieldRaw returns the request value if present, otherwise the fallback value.
func jsonFieldRaw(req map[string]interface{}, key string, fallback interface{}) interface{} {
	if v, ok := req[key]; ok && v != nil {
		return v
	}
	return fallback
}


// autoEnableClonedRuntime creates NodeBackendRuntime records for the cloned runtime
// on all online nodes that match the vendor, so it appears in deployment wizard immediately.
func (h *AgentHandler) autoEnableClonedRuntime(runtimeID, vendor, imageName, tenantID, now string) {
	rows, err := h.DB.Query(`SELECT id FROM nodes WHERE status = 'online' AND tenant_id = ?`, tenantID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var nodeID string
		if err := rows.Scan(&nodeID); err != nil {
			continue
		}
		// Check if vendor matches: for CPU vendor, always match.
		// For GPU vendors, check if node has matching GPU.
		vendorMatch := vendor == "cpu"
		if !vendorMatch {
			var gpuCount int
			h.DB.QueryRow(`SELECT COUNT(*) FROM gpu_devices WHERE node_id = ? AND vendor = ? AND status = 'available'`,
				nodeID, vendor).Scan(&gpuCount)
			vendorMatch = gpuCount > 0
		}
		if !vendorMatch {
			continue
		}
		// Check if NBR already exists
		nbrID := nodeID + ":" + runtimeID
		var existing string
		if h.DB.QueryRow(`SELECT id FROM node_backend_runtimes WHERE id = ?`, nbrID).Scan(&existing) == nil {
			continue // already exists
		}
		// Evaluate status
		status := "ready"
		reason := "auto-enabled from clone"
		displayName := ""
		if vendor == "huawei" || vendor == "ascend" {
			status = "template_only"
			reason = "vendor requires hardware validation"
		}
		h.DB.Exec(`INSERT INTO node_backend_runtimes
			(id, backend_runtime_id, node_id, display_name, runner_type, image_ref, image_present, docker_available,
			 driver_version, toolkit_version, device_check_json, status, status_reason, last_checked_at,
			 config_snapshot_json, source_runtime_name, source_runtime_revision, tenant_id, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			nbrID, runtimeID, nodeID, displayName, "docker", imageName, 1, 1,
			"", "", "{}",
			status, reason, now,
			"{}", "", "", tenantID, now, now)
	}
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
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
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
	if v, ok := req["image_present"]; ok {
		if b, ok := v.(bool); ok {
			sets = append(sets, "image_present = ?")
			args = append(args, boolInt(b))
			needsRecheck = true
		}
	}
	for _, f := range []string{"config_snapshot_json", "device_check_json"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, jsonString(v))
			if f == "config_snapshot_json" {
				needsRecheck = true
			}
		}
	}
	if v, ok := req["docker_available"]; ok {
		if b, ok := v.(bool); ok {
			sets = append(sets, "docker_available = ?")
			args = append(args, boolInt(b))
			needsRecheck = true
		}
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
