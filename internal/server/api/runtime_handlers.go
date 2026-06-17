package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"lightai-go/internal/common/log"

	"github.com/google/uuid"
)

// ==========================================================================
// BackendRuntime CRUD
// ==========================================================================

func (h *AgentHandler) HandleListBackendRuntimes(w http.ResponseWriter, r *http.Request) {
	tid := tenantID(r)
	q := `SELECT id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, created_at, updated_at FROM backend_runtimes`
	var err error
	var out []map[string]interface{}
	if isPlatformAdmin(r) {
		out, err = h.queryBackendRuntimes(q + ` ORDER BY name`)
	} else {
		out, err = h.queryBackendRuntimes(q+` WHERE tenant_id = ? OR tenant_id = '' ORDER BY name`, tid)
	}
	if err != nil {
		log.Error("list backend runtimes", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleCreateBackendRuntimeFromTemplate(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "backend_runtime.create")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	templateName := strVal(req, "template_name", "")
	if templateName == "" {
		log.OpWarn("backend_runtime.create", "input_validated", "error", "template_name required")
		writeError(w, http.StatusBadRequest, "template_name is required")
		return
	}
	// Read template from config file.
	path := "configs/model-runtime/backend-runtime-templates/" + templateName + ".yaml"
	_, err := osReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "template not found: "+templateName)
		return
	}

	id := uuid.NewString()
	_ = userID(r) // reserved for audit
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)
	name := strVal(req, "name", templateName+"-"+id[:8])

	// Get backend and version IDs from the seeded data.
	var backendID, versionID string
	h.DB.QueryRow(`SELECT id FROM inference_backends WHERE name = ?`, strVal(req, "backend_name", "")).Scan(&backendID)
	h.DB.QueryRow(`SELECT id FROM backend_versions WHERE backend_id = ? AND version = ?`, backendID, strVal(req, "backend_version", "")).Scan(&versionID)

	_, err = h.DB.Exec(
		`INSERT INTO backend_runtimes (id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), backendID, versionID, templateName,
		strVal(req, "vendor", "custom"), "docker",
		strVal(req, "image_name", ""), strVal(req, "image_pull_policy", "if_not_present"),
		"[]", "[]", "{}", "{}", "{}", "{}", 0, 1, tid, now, now,
	)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.create", "db_write", opStart, err,
			"id", id, "name", name, "template", templateName)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "backend_runtime.create", opStart,
		"id", id, "name", name, "template", templateName, "tenant_id", tid)
	writeJSON(w, http.StatusCreated, h.getBackendRuntimeJSON(id))
}

func (h *AgentHandler) HandleGetBackendRuntime(w http.ResponseWriter, r *http.Request) {
	m := h.getBackendRuntimeJSON(r.PathValue("id"))
	if m == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	tid, _ := m["tenant_id"].(string)
	if tid != "" && !tenantScopeCheck(r, tid) && !isPlatformAdmin(r) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *AgentHandler) HandlePatchBackendRuntime(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "backend_runtime.update")
	id := r.PathValue("id")
	existing := h.getBackendRuntimeJSON(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	tid, _ := existing["tenant_id"].(string)
	if tid != "" && !tenantScopeCheck(r, tid) && !isPlatformAdmin(r) {
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
	for _, f := range []string{"display_name", "image_name", "image_pull_policy", "vendor"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	args = append(args, id)
	_, err := h.DB.Exec(`UPDATE backend_runtimes SET `+joinSets(sets)+` WHERE id = ?`, args...)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.update", "db_write", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "backend_runtime.update", opStart, "id", id)
	writeJSON(w, http.StatusOK, h.getBackendRuntimeJSON(id))
}

func (h *AgentHandler) HandleDeleteBackendRuntime(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "backend_runtime.delete")
	id := r.PathValue("id")
	existing := h.getBackendRuntimeJSON(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	tid, _ := existing["tenant_id"].(string)
	if tid != "" && !tenantScopeCheck(r, tid) && !isPlatformAdmin(r) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	_, err := h.DB.Exec(`DELETE FROM backend_runtimes WHERE id = ?`, id)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.delete", "db_write", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "backend_runtime.delete", opStart, "id", id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AgentHandler) getBackendRuntimeJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, created_at, updated_at FROM backend_runtimes WHERE id = ?`, id)
	var rid, name, dn, bid, bvid, stn, vendor, rt, img, ipp, eoj, aoj, defEnv, dj, mmj, hcoj, tid, ca, ua string
	var isB, isE int
	if err := row.Scan(&rid, &name, &dn, &bid, &bvid, &stn, &vendor, &rt, &img, &ipp, &eoj, &aoj, &defEnv, &dj, &mmj, &hcoj, &isB, &isE, &tid, &ca, &ua); err != nil {
		return nil
	}
	return map[string]interface{}{
		"id": rid, "name": name, "display_name": dn, "backend_id": bid, "backend_version_id": bvid,
		"source_template_name": stn, "vendor": vendor, "runtime_type": rt,
		"image_name": img, "image_pull_policy": ipp,
		"entrypoint_override_json": json.RawMessage(eoj), "args_override_json": json.RawMessage(aoj),
		"default_env_json": redactRawJSON(defEnv), "docker_json": json.RawMessage(dj),
		"model_mount_json": json.RawMessage(mmj), "health_check_override_json": json.RawMessage(hcoj),
		"is_builtin": isB == 1, "is_editable": isE == 1, "tenant_id": tid,
		"created_at": ca, "updated_at": ua,
	}
}

func (h *AgentHandler) queryBackendRuntimes(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var rid, name, dn, bid, bvid, stn, vendor, rt, img, ipp, eoj, aoj, defEnv, dj, mmj, hcoj, tid, ca, ua string
		var isB, isE int
		if err := rows.Scan(&rid, &name, &dn, &bid, &bvid, &stn, &vendor, &rt, &img, &ipp, &eoj, &aoj, &defEnv, &dj, &mmj, &hcoj, &isB, &isE, &tid, &ca, &ua); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": rid, "name": name, "display_name": dn, "backend_id": bid, "backend_version_id": bvid,
			"source_template_name": stn, "vendor": vendor, "runtime_type": rt,
			"image_name": img, "image_pull_policy": ipp,
			"entrypoint_override_json": json.RawMessage(eoj), "args_override_json": json.RawMessage(aoj),
			"default_env_json": redactRawJSON(defEnv), "docker_json": json.RawMessage(dj),
			"model_mount_json": json.RawMessage(mmj), "health_check_override_json": json.RawMessage(hcoj),
			"is_builtin": isB == 1, "is_editable": isE == 1, "tenant_id": tid,
			"created_at": ca, "updated_at": ua,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out, nil
}

func joinSets(sets []string) string {
	s := ""
	for i, set := range sets {
		if i > 0 {
			s += ", "
		}
		s += set
	}
	return s
}

// osReadFile is a wrapper for testing.
var osReadFile = os.ReadFile //nolint
