package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
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

	// Enrich with node_count and ready_count from node_backend_runtimes
	countRows, cerr := h.DB.Query(`SELECT backend_runtime_id, COUNT(*) AS node_count, SUM(CASE WHEN status = 'ready' THEN 1 ELSE 0 END) AS ready_count FROM node_backend_runtimes GROUP BY backend_runtime_id`)
	if cerr == nil {
		defer countRows.Close()
		countMap := make(map[string]map[string]int)
		for countRows.Next() {
			var rid string
			var nc, rc int
			if e := countRows.Scan(&rid, &nc, &rc); e == nil {
				countMap[rid] = map[string]int{"node_count": nc, "ready_count": rc}
			}
		}
		for _, entry := range out {
			id := strVal(entry, "id", "")
			if stats, ok := countMap[id]; ok {
				entry["node_count"] = stats["node_count"]
				entry["ready_count"] = stats["ready_count"]
			} else {
				entry["node_count"] = 0
				entry["ready_count"] = 0
			}
		}
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
	// BRR-RV-004: Try new catalog path first, fall back to old path for
	// backward compatibility. Template names like "vllm-nvidia-docker" map
	// to "configs/backend-catalog/runtimes/vllm/nvidia-docker.yaml" in the
	// new layout, or "configs/model-runtime/backend-runtime-templates/vllm-nvidia-docker.yaml"
	// in the old flat layout.
	path := resolveTemplatePath(templateName)
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
	if editable, _ := existing["is_editable"].(bool); !editable {
		writeError(w, http.StatusConflict, "system-managed runtime is read-only; create a user-managed copy before editing")
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
	for _, f := range []string{"name", "display_name", "image_name", "image_pull_policy", "vendor", "default_env_json", "docker_json", "model_mount_json", "health_check_override_json", "args_override_json", "entrypoint_override_json"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			if strings.HasSuffix(f, "_json") || f == "docker_json" {
				args = append(args, jsonString(v))
			} else {
				args = append(args, v)
			}
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
	if editable, _ := existing["is_editable"].(bool); !editable {
		writeError(w, http.StatusConflict, "system-managed runtime is read-only")
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

func (h *AgentHandler) HandleListNodeBackendRuntimes(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	rows, err := h.DB.Query(`SELECT nbr.id, nbr.backend_runtime_id, nbr.node_id, nbr.runner_type, nbr.image_ref, nbr.image_id, nbr.image_digest, nbr.image_present, nbr.docker_available, nbr.driver_version, nbr.toolkit_version, nbr.device_check_json, nbr.status, nbr.status_reason, nbr.last_checked_at, nbr.tenant_id, nbr.created_at, nbr.updated_at, br.name, br.display_name, br.vendor
		FROM node_backend_runtimes nbr
		JOIN backend_runtimes br ON br.id = nbr.backend_runtime_id
		WHERE nbr.node_id = ?
		ORDER BY br.name`, nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, runtimeID, nid, runner, imageRef, imageID, digest, driver, toolkit, checkJSON, status, reason, checked, tid, ca, ua, rtName, rtDisplay, vendor string
		var imagePresent, dockerAvailable int
		if err := rows.Scan(&id, &runtimeID, &nid, &runner, &imageRef, &imageID, &digest, &imagePresent, &dockerAvailable, &driver, &toolkit, &checkJSON, &status, &reason, &checked, &tid, &ca, &ua, &rtName, &rtDisplay, &vendor); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": id, "backend_runtime_id": runtimeID, "node_id": nid, "runner_type": runner,
			"image_ref": imageRef, "image_id": imageID, "image_digest": digest,
			"image_present": imagePresent == 1, "docker_available": dockerAvailable == 1,
			"driver_version": driver, "toolkit_version": toolkit, "device_check_json": json.RawMessage(checkJSON),
			"status": status, "status_reason": reason, "last_checked_at": checked, "tenant_id": tid,
			"created_at": ca, "updated_at": ua,
			"backend_runtime": map[string]interface{}{"name": rtName, "display_name": rtDisplay, "vendor": vendor},
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleEnableNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
	h.upsertNodeBackendRuntime(w, r, false)
}

func (h *AgentHandler) HandleCheckNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
	h.upsertNodeBackendRuntime(w, r, true)
}

func (h *AgentHandler) upsertNodeBackendRuntime(w http.ResponseWriter, r *http.Request, checkOnly bool) {
	nodeID := r.PathValue("id")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	runtimeID := strVal(req, "backend_runtime_id", "")
	if runtimeID == "" {
		writeError(w, http.StatusBadRequest, "backend_runtime_id is required")
		return
	}
	rt := h.getBackendRuntimeJSON(runtimeID)
	if rt == nil {
		writeError(w, http.StatusNotFound, "backend runtime not found")
		return
	}
	vendor := strVal(rt, "vendor", "")
	imageRef := strVal(req, "image_ref", strVal(rt, "image_name", ""))
	imagePresent := boolVal(req, "image_present", false)
	dockerAvailable := boolVal(req, "docker_available", false)
	status, reason := h.evaluateNodeBackendRuntime(nodeID, vendor, imageRef, imagePresent, dockerAvailable)
	if checkOnly && status == "unknown" {
		reason = "check request did not provide docker/image evidence"
	}
	id := nodeID + ":" + runtimeID
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`INSERT INTO node_backend_runtimes
		(id, backend_runtime_id, node_id, runner_type, image_ref, image_present, docker_available, driver_version, toolkit_version, device_check_json, status, status_reason, last_checked_at, tenant_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(node_id, backend_runtime_id) DO UPDATE SET
			image_ref=excluded.image_ref,
			image_present=excluded.image_present,
			docker_available=excluded.docker_available,
			driver_version=excluded.driver_version,
			toolkit_version=excluded.toolkit_version,
			device_check_json=excluded.device_check_json,
			status=excluded.status,
			status_reason=excluded.status_reason,
			last_checked_at=excluded.last_checked_at,
			updated_at=excluded.updated_at`,
		id, runtimeID, nodeID, "docker", imageRef, boolInt(imagePresent), boolInt(dockerAvailable),
		strVal(req, "driver_version", ""), strVal(req, "toolkit_version", ""), jsonString(map[string]interface{}{"vendor": vendor}),
		status, reason, now, tid, now, now)
	if err != nil {
		log.Error("node backend runtime upsert failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": id, "backend_runtime_id": runtimeID, "node_id": nodeID,
		"image_ref": imageRef, "image_present": imagePresent, "docker_available": dockerAvailable,
		"status": status, "status_reason": reason, "last_checked_at": now,
	})
}

func (h *AgentHandler) evaluateNodeBackendRuntime(nodeID, vendor, imageRef string, imagePresent, dockerAvailable bool) (string, string) {
	if vendor == "huawei" || vendor == "ascend" {
		return "template_only", "Huawei/Ascend runtime is a template only until an adapter and hardware validation are available"
	}
	if vendor != "cpu" {
		var count int
		h.DB.QueryRow(`SELECT COUNT(*) FROM gpu_devices WHERE node_id = ? AND lower(vendor) = lower(?)`, nodeID, vendor).Scan(&count)
		if count == 0 {
			return "unsupported_device", "node has no matching GPU vendor"
		}
	}
	if !dockerAvailable {
		return "unknown", "docker availability has not been verified"
	}
	if imageRef != "" && !imagePresent {
		return "missing_image", "docker image is not present on node"
	}
	return "ready", "runtime verified for node"
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
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

// resolveTemplatePath returns the config file path for a template name.
// It tries the new catalog layout first (configs/backend-catalog/runtimes/{backend}/{vendor}-docker.yaml),
// then falls back to the old flat layout (configs/model-runtime/backend-runtime-templates/{name}.yaml).
func resolveTemplatePath(templateName string) string {
	// New path: extract backend from "vllm-nvidia-docker" -> "vllm"/"nvidia-docker.yaml"
	// Template names follow the pattern {backend}-{vendor}-docker
	if idx := strings.Index(templateName, "-"); idx > 0 {
		backend := templateName[:idx]
		rest := templateName[idx+1:]
		newPath := "configs/backend-catalog/runtimes/" + backend + "/" + rest + ".yaml"
		if _, err := osReadFile(newPath); err == nil {
			return newPath
		}
	}
	// Old path: flat layout for backward compatibility
	return "configs/model-runtime/backend-runtime-templates/" + templateName + ".yaml"
}

// osReadFile is a wrapper for testing.
var osReadFile = os.ReadFile //nolint
