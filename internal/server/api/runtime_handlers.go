package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/authz"
	"lightai-go/internal/server/runplan"

	"github.com/google/uuid"
)

// ==========================================================================
// BackendRuntime CRUD
// ==========================================================================

func (h *AgentHandler) HandleListBackendRuntimes(w http.ResponseWriter, r *http.Request) {
	tid := tenantID(r)
	q := `SELECT id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, is_builtin, is_editable, tenant_id, status, visibility, support_level, config_set_json, source_metadata_json, created_at, updated_at FROM backend_runtimes`
	ordinarySelectorFilter := `(managed_by != 'system' OR (visibility = 'visible' AND status IN ('active','experimental')))`
	var err error
	var out []map[string]interface{}
	if isPlatformAdmin(r) && strings.EqualFold(r.URL.Query().Get("include_hidden"), "true") {
		out, err = h.queryBackendRuntimes(q + ` ORDER BY name`)
	} else if isPlatformAdmin(r) {
		out, err = h.queryBackendRuntimes(q + ` WHERE ` + ordinarySelectorFilter + ` ORDER BY name`)
	} else {
		out, err = h.queryBackendRuntimes(q+` WHERE (tenant_id = ? OR tenant_id = '') AND `+ordinarySelectorFilter+` ORDER BY name`, tid)
	}
	if err != nil {
		log.Error("list backend runtimes", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Enrich with node_count and ready_count from node_backend_runtimes
	countRows, cerr := h.DB.Query(`SELECT backend_runtime_id, COUNT(*) AS node_count, SUM(CASE WHEN status = 'ready' THEN 1 ELSE 0 END) AS ready_count, SUM(CASE WHEN status IN ('ready','ready_with_warnings') THEN 1 ELSE 0 END) AS deployable_count FROM node_backend_runtimes GROUP BY backend_runtime_id`)
	if cerr == nil {
		defer countRows.Close()
		countMap := make(map[string]map[string]int)
		for countRows.Next() {
			var rid string
			var nc, rc, dc int
			if e := countRows.Scan(&rid, &nc, &rc, &dc); e == nil {
				countMap[rid] = map[string]int{"node_count": nc, "ready_count": rc, "deployable_count": dc}
			}
		}
		for _, entry := range out {
			id := strVal(entry, "id", "")
			if stats, ok := countMap[id]; ok {
				entry["node_count"] = stats["node_count"]
				entry["ready_count"] = stats["ready_count"]
				entry["deployable_count"] = stats["deployable_count"]
			} else {
				entry["node_count"] = 0
				entry["ready_count"] = 0
				entry["deployable_count"] = 0
			}
		}
	}
	writeJSON(w, http.StatusOK, publicBackendRuntimeList(out))
}

func (h *AgentHandler) HandleCreateBackendRuntimeFromTemplate(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "backend_runtime.create")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	templateName := strVal(req, "template_name", "")

	id := uuid.NewString()
	_ = userID(r) // reserved for audit
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)
	name := strings.TrimSpace(strVal(req, "name", templateName+"-"+id[:8]))
	if name == "-"+id[:8] {
		name = "runtime-" + id[:8]
	}
	name = h.uniqueRuntimeName(tid, name)
	displayName := strings.TrimSpace(strVal(req, "display_name", name))
	if displayName == "" {
		displayName = name
	}

	backendID := strVal(req, "backend_id", "")
	if backendID == "" {
		h.DB.QueryRow(`SELECT id FROM inference_backends WHERE name = ?`, strVal(req, "backend_name", "")).Scan(&backendID)
	}
	versionID := strVal(req, "backend_version_id", "")
	if versionID == "" {
		h.DB.QueryRow(`SELECT id FROM backend_versions WHERE backend_id = ? AND version = ?`, backendID, strVal(req, "backend_version", "")).Scan(&versionID)
	}
	version := h.getBackendVersionJSON(versionID)
	if backendID == "" || version == nil {
		writeError(w, http.StatusBadRequest, "backend_id and backend_version_id are required")
		return
	}
	vendor := strVal(req, "vendor", "custom")
	versionSet := mapFromAny(version["config_set"])
	configSet := copyConfigSet(rawJSONString(version["config_set_json"], "{}"))
	if len(configSetItems(configSet)) == 0 && len(versionSet) > 0 {
		configSet = versionSet
	}
	if imageRef := strVal(req, "image_ref", ""); imageRef != "" {
		setConfigValue(configSet, "launcher.image", imageRef, "BackendRuntime", id, "user_create")
	}
	if v, ok := req["docker_options"]; ok {
		setConfigValue(configSet, "launcher.docker_options", v, "BackendRuntime", id, "user_create")
	}
	if v, ok := req["env"]; ok {
		setConfigValue(configSet, "runtime.env", v, "BackendRuntime", id, "user_create")
	}
	if v, ok := req["model_mount"]; ok {
		setConfigValue(configSet, "runtime.model_mount", v, "BackendRuntime", id, "user_create")
	}
	if v, ok := req["health_check"]; ok {
		setConfigValue(configSet, "runtime.health", v, "BackendRuntime", id, "user_create")
	}
	if v, ok := req["entrypoint"]; ok {
		setConfigValue(configSet, "launcher.entrypoint", v, "BackendRuntime", id, "user_create")
	}
	if v, ok := req["command"]; ok {
		setConfigValue(configSet, "launcher.command", v, "BackendRuntime", id, "user_create")
	}
	sourceRevision := strVal(version, "checksum", "")
	if sourceRevision == "" {
		sourceRevision = strVal(version, "updated_at", "")
	}
	sourceMetadata := map[string]interface{}{
		"source_type":                "backend_runtime",
		"source_backend_id":          backendID,
		"source_backend_version_id":  versionID,
		"source_version_revision":    sourceRevision,
		"source_backend_runtime_id":  "",
		"source_template_name":       templateName,
		"copy_semantics":             "copy_on_create",
		"source_config_set_checksum": checksumString(rawJSONString(version["config_set_json"], "{}")),
	}

	_, err := h.DB.Exec(
		`INSERT INTO backend_runtimes (id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, is_builtin, is_editable, tenant_id, slug, managed_by, source, catalog_version, checksum, status, config_set_json, source_metadata_json, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, displayName, backendID, versionID, templateName,
		vendor, "docker", 0, 1, tid, slugify(name), "user", "user-config", "configset-v1", checksumString(configSetJSON(configSet)), "active", configSetJSON(configSet), jsonString(sourceMetadata), now, now,
	)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.create", "db_write", opStart, err,
			"id", id, "name", name, "template", templateName)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "backend_runtime.create", opStart,
		"id", id, "name", name, "template", templateName, "tenant_id", tid)
	writeJSON(w, http.StatusCreated, publicBackendRuntimeJSON(h.getBackendRuntimeJSON(id)))
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
	writeJSON(w, http.StatusOK, publicBackendRuntimeJSON(m))
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
	configSet := copyConfigSet(rawJSONString(existing["config_set_json"], "{}"))
	for _, f := range []string{"name", "display_name", "vendor"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			if s, ok := v.(string); ok {
				v = strings.TrimSpace(s)
				if (f == "name" || f == "display_name") && v == "" {
					writeError(w, http.StatusBadRequest, f+" is required")
					return
				}
			}
			args = append(args, v)
		}
	}
	if v, ok := req["image_ref"]; ok {
		setConfigValue(configSet, "launcher.image", v, "BackendRuntime", id, "user_patch")
	}
	if v, ok := req["docker_options"]; ok {
		setConfigValue(configSet, "launcher.docker_options", v, "BackendRuntime", id, "user_patch")
	}
	if v, ok := req["env"]; ok {
		setConfigValue(configSet, "runtime.env", v, "BackendRuntime", id, "user_patch")
	}
	if v, ok := req["model_mount"]; ok {
		setConfigValue(configSet, "runtime.model_mount", v, "BackendRuntime", id, "user_patch")
	}
	if v, ok := req["health_check"]; ok {
		setConfigValue(configSet, "runtime.health", v, "BackendRuntime", id, "user_patch")
	}
	if v, ok := req["entrypoint"]; ok {
		setConfigValue(configSet, "launcher.entrypoint", v, "BackendRuntime", id, "user_patch")
	}
	if v, ok := req["command"]; ok {
		setConfigValue(configSet, "launcher.command", v, "BackendRuntime", id, "user_patch")
	}
	if _, ok := req["config_set"]; ok {
		writeError(w, http.StatusBadRequest, "config_set is not accepted; use editable_config_patch to modify individual parameters")
		return
	}
	if _, ok := req["config_set_json"]; ok {
		writeError(w, http.StatusBadRequest, "config_set_json is not accepted; use editable_config_patch to modify individual parameters")
		return
	}
	sets = append(sets, "config_set_json = ?", "checksum = ?")
	args = append(args, configSetJSON(configSet), checksumString(configSetJSON(configSet)))
	args = append(args, id)
	_, err := h.DB.Exec(`UPDATE backend_runtimes SET `+joinSets(sets)+` WHERE id = ?`, args...)
	if err != nil {
		log.OperationFailed(ctx, "backend_runtime.update", "db_write", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "backend_runtime.update", opStart, "id", id)
	writeJSON(w, http.StatusOK, publicBackendRuntimeJSON(h.getBackendRuntimeJSON(id)))
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
	if !authz.CheckNodeTenant(r, h.DB.DB, nodeID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rows, err := h.DB.Query(`SELECT nbr.id, nbr.backend_runtime_id, nbr.node_id, COALESCE(nbr.display_name,''), COALESCE(nbr.runner_type,''), COALESCE(nbr.image_ref,''), COALESCE(nbr.image_id,''), COALESCE(nbr.image_digest,''), nbr.image_present, nbr.docker_available, COALESCE(nbr.driver_version,''), COALESCE(nbr.toolkit_version,''), COALESCE(nbr.device_check_json,'{}'), COALESCE(nbr.status,''), COALESCE(nbr.status_reason,''), COALESCE(nbr.last_checked_at,''), COALESCE(nbr.tenant_id,''), COALESCE(nbr.created_at,''), COALESCE(nbr.updated_at,''), COALESCE(nbr.config_set_json,'{}'), COALESCE(nbr.source_metadata_json,'{}'), COALESCE(nbr.probe_results_json,'{}'), br.name, COALESCE(br.display_name,''), COALESCE(br.vendor,'')
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
		var id, runtimeID, nid, displayName, runner, imageRef, imageID, digest, driver, toolkit, checkJSON, status, reason, checked, tid, ca, ua, configSetJSONRaw, sourceMetaJSON, probeResultsJSON, rtName, rtDisplay, vendor string
		var imagePresent, dockerAvailable int
		if err := rows.Scan(&id, &runtimeID, &nid, &displayName, &runner, &imageRef, &imageID, &digest, &imagePresent, &dockerAvailable, &driver, &toolkit, &checkJSON, &status, &reason, &checked, &tid, &ca, &ua, &configSetJSONRaw, &sourceMetaJSON, &probeResultsJSON, &rtName, &rtDisplay, &vendor); err != nil {
			log.Error("node backend runtime list scan failed", "error", err, "node_id", nodeID)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if displayName == "" {
			displayName = rtDisplay
			if displayName == "" {
				displayName = rtName
			}
			if nid != "" {
				displayName += " - " + nid
			}
		}
		deployable := isNBRDeployable(status)
		warnings := extractProbeWarnings(probeResultsJSON, status)
		out = append(out, map[string]interface{}{
			"id": id, "backend_runtime_id": runtimeID, "node_id": nid, "name": displayName, "display_name": displayName, "runner_type": runner,
			"image_ref": imageRef, "image_id": imageID, "image_digest": digest,
			"image_present": imagePresent == 1, "docker_available": dockerAvailable == 1,
			"driver_version": driver, "toolkit_version": toolkit, "device_check_json": json.RawMessage(checkJSON),
			"status": status, "status_reason": reason, "last_checked_at": checked, "tenant_id": tid,
			"created_at": ca, "updated_at": ua,
			"deployable":         deployable,
			"warnings":           warnings,
			"disabled_reason":    nbrDisabledReason(status, reason),
			"config_set":         parseConfigSet(configSetJSONRaw),
			"source_metadata":    configSourceMetadata(sourceMetaJSON),
			"probe_results_json": json.RawMessage(probeResultsJSON),
			"backend_runtime":    map[string]interface{}{"name": rtName, "display_name": rtDisplay, "vendor": vendor},
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleListAllNodeBackendRuntimes returns all NBRs across all tenant-accessible nodes.
// R-011: Single aggregate call replaces per-node fan-out from frontend.
func (h *AgentHandler) HandleListAllNodeBackendRuntimes(w http.ResponseWriter, r *http.Request) {
	tid := tenantID(r)
	isAdmin := isPlatformAdmin(r)
	query := `SELECT nbr.id, nbr.backend_runtime_id, nbr.node_id, COALESCE(nbr.display_name,''), COALESCE(nbr.runner_type,''), COALESCE(nbr.image_ref,''), COALESCE(nbr.image_id,''), COALESCE(nbr.image_digest,''), nbr.image_present, nbr.docker_available, COALESCE(nbr.driver_version,''), COALESCE(nbr.toolkit_version,''), COALESCE(nbr.device_check_json,'{}'), COALESCE(nbr.status,''), COALESCE(nbr.status_reason,''), COALESCE(nbr.last_checked_at,''), COALESCE(nbr.tenant_id,''), COALESCE(nbr.created_at,''), COALESCE(nbr.updated_at,''), COALESCE(nbr.config_set_json,'{}'), COALESCE(nbr.source_metadata_json,'{}'), COALESCE(nbr.probe_results_json,'{}'), br.name, COALESCE(br.display_name,''), COALESCE(br.vendor,'')
		FROM node_backend_runtimes nbr
		JOIN backend_runtimes br ON br.id = nbr.backend_runtime_id
		JOIN nodes n ON n.id = nbr.node_id`
	var args []interface{}
	if !isAdmin && tid != "" {
		query += ` WHERE (n.tenant_id = ? OR n.tenant_id = '') AND (nbr.tenant_id = ? OR nbr.tenant_id = '')`
		args = append(args, tid, tid)
	}
	query += ` ORDER BY n.hostname, br.name`
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, runtimeID, nid, displayName, runner, imageRef, imageID, digest, driver, toolkit, checkJSON, status, reason, checked, tid2, ca, ua, configSetJSONRaw, sourceMetaJSON, probeResultsJSON, rtName, rtDisplay, vendor string
		var imagePresent, dockerAvailable int
		if err := rows.Scan(&id, &runtimeID, &nid, &displayName, &runner, &imageRef, &imageID, &digest, &imagePresent, &dockerAvailable, &driver, &toolkit, &checkJSON, &status, &reason, &checked, &tid2, &ca, &ua, &configSetJSONRaw, &sourceMetaJSON, &probeResultsJSON, &rtName, &rtDisplay, &vendor); err != nil {
			log.Error("all node backend runtime list scan failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if displayName == "" {
			displayName = rtDisplay
			if displayName == "" {
				displayName = rtName
			}
		}
		deployable := isNBRDeployable(status)
		warnings := extractProbeWarnings(probeResultsJSON, status)
		out = append(out, map[string]interface{}{
			"id": id, "backend_runtime_id": runtimeID, "node_id": nid, "name": displayName, "display_name": displayName,
			"runner_type": runner, "image_ref": imageRef, "image_present": imagePresent == 1, "docker_available": dockerAvailable == 1,
			"status": status, "status_reason": reason, "last_checked_at": checked,
			"deployable": deployable, "warnings": warnings,
			"disabled_reason":    nbrDisabledReason(status, reason),
			"config_set":         parseConfigSet(configSetJSONRaw),
			"source_metadata":    configSourceMetadata(sourceMetaJSON),
			"probe_results_json": json.RawMessage(probeResultsJSON),
			"backend_runtime":    map[string]interface{}{"name": rtName, "display_name": rtDisplay, "vendor": vendor},
			"tenant_id":          tid2, "created_at": ca, "updated_at": ua,
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

// HandleCheckNodeBackendRuntime is DEPRECATED (R-001, 2026-06-25).
// The /check route has been deleted from the router. Session callers must use
// /enable for NBR creation and /check-request for server-proxied agent verification.
// This function is retained only as an internal handler reference; no route maps to it.
func (h *AgentHandler) HandleCheckNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"deprecated — use /enable or /check-request"}`, http.StatusGone)
}

// HandleRequestNodeBackendRuntimeCheck is the UI-facing check endpoint.
// It implements a multi-level Image Capability Probe:
//
//	Level 1: Docker image list (evidence only, NOT authoritative).
//	Level 2: Docker ImageInspect (AUTHORITATIVE existence check).
//	Level 3: Backend type matching (best-effort, lenient).
//	Level 4: Version probe (deferred to future design).
//
// It does NOT accept client-provided image_present/docker_available.
// Instead, the server queries the agent for Docker image status and
// evaluates readiness with server-verified evidence.
// POST /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/check-request
func (h *AgentHandler) HandleRequestNodeBackendRuntimeCheck(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	nbrID := r.PathValue("nbr_id")
	if !authz.CheckNBRTenant(r, h.DB.DB, nbrID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if nodeID == "" || nbrID == "" {
		writeError(w, http.StatusBadRequest, "node_id and nbr_id are required")
		return
	}

	// Look up NBR.
	var nbrBackendRuntimeID, nbrImageRef, nbrStatus string
	if err := h.DB.QueryRow(
		`SELECT backend_runtime_id, COALESCE(image_ref,''), status FROM node_backend_runtimes WHERE id = ? AND node_id = ?`,
		nbrID, nodeID,
	).Scan(&nbrBackendRuntimeID, &nbrImageRef, &nbrStatus); err != nil {
		writeError(w, http.StatusNotFound, "node backend runtime not found")
		return
	}

	// Resolve image_ref: use NBR override, else fall back to BackendRuntime default.
	rt := h.getBackendRuntimeJSON(nbrBackendRuntimeID)
	if rt == nil {
		writeError(w, http.StatusNotFound, "backend runtime not found")
		return
	}
	if nbrImageRef == "" {
		nbrImageRef = strVal(rt, "image_ref", "")
	}
	vendor := strVal(rt, "vendor", "")

	// Initialize probe results (4 levels).
	probeStart := time.Now()
	probeResults := map[string]interface{}{
		"level1": map[string]interface{}{},
		"level2": map[string]interface{}{},
		"level3": map[string]interface{}{},
		"level4": map[string]interface{}{},
	}
	dockerAvailable := false
	imagePresent := false
	inspectSuccess := false
	inspectNotFound := false
	var imageID, imageDigest, dockerErr string

	nodeAddr, nodePort := h.getNodeAddress(nodeID)

	// ---- Level 1: Docker image list (evidence only, NOT authoritative) ----
	// The /docker-images list is used for UI selection and supporting evidence.
	// It does NOT determine missing_image. Only ImageInspect (Level 2) is authoritative.
	if nodeAddr == "" || nodePort == 0 {
		dockerErr = "node has no advertised address or metrics port"
		probeResults["level1"] = map[string]interface{}{
			"image_present": false,
			"source":        "docker_images_list",
			"error":         dockerErr,
		}
	} else if h.AgentClient == nil {
		dockerErr = "agent client not configured"
		probeResults["level1"] = map[string]interface{}{
			"image_present": false,
			"source":        "docker_images_list",
			"error":         dockerErr,
		}
	} else {
		params := url.Values{"limit": {"1000"}}
		body, _, err := h.AgentClient.GetJSON(r.Context(), nodeAddr, nodePort, "/docker-images", params)
		if err != nil {
			dockerErr = fmt.Sprintf("agent unreachable: %v", err)
			log.Warn("nbr.check_request.agent_unreachable", "node_id", nodeID, "addr", nodeAddr, "port", nodePort, "error", err)
			probeResults["level1"] = map[string]interface{}{
				"image_present": false,
				"source":        "docker_images_list",
				"error":         dockerErr,
			}
		} else {
			dockerAvailable = true
			if nbrImageRef != "" {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err == nil {
					imagesRaw, _ := result["images"].([]interface{})
					for _, imgRaw := range imagesRaw {
						img, _ := imgRaw.(map[string]interface{})
						if img == nil {
							continue
						}
						imgRef := strVal(img, "image_ref", "")
						if imgRef == "" {
							repo := strVal(img, "repository", "")
							tag := strVal(img, "tag", "")
							if repo != "" && tag != "" && tag != "<none>" {
								imgRef = repo + ":" + tag
							}
						}
						if imgRef == "" {
							continue
						}
						if nbrImageRef == imgRef || strings.Contains(imgRef, nbrImageRef) || strings.Contains(nbrImageRef, imgRef) {
							imagePresent = true
							imageID = strVal(img, "image_id", "")
							imageDigest = strVal(img, "digest", "")
							probeResults["level1"] = map[string]interface{}{
								"image_present": true,
								"source":        "docker_images_list",
								"image_ref":     imgRef,
								"image_id":      imageID,
								"digest":        imageDigest,
								"created_at":    strVal(img, "created_at", ""),
								"size":          strVal(img, "size", ""),
							}
							break
						}
					}
					if !imagePresent {
						probeResults["level1"] = map[string]interface{}{
							"image_present": false,
							"source":        "docker_images_list",
							"image_ref":     nbrImageRef,
							"note":          "image not in docker images list; authoritative check via ImageInspect",
						}
					}
				}
			}
		}
	}

	// ---- Level 2: Docker ImageInspect (AUTHORITATIVE existence check) ----
	// ImageInspect is the single source of truth for image existence.
	// Only ImageInspect returning a clear "not found" error produces missing_image.
	if nodeAddr != "" && nodePort > 0 && nbrImageRef != "" && h.AgentClient != nil {
		inspectParams := url.Values{"ref": {nbrImageRef}}
		inspectBody, _, inspectErr := h.AgentClient.GetJSON(r.Context(), nodeAddr, nodePort, "/docker-image-inspect", inspectParams)
		if inspectErr != nil {
			log.Warn("nbr.check_request.inspect_failed", "node_id", nodeID, "addr", nodeAddr, "port", nodePort, "error", inspectErr)
			probeResults["level2"] = map[string]interface{}{
				"inspect_success": false,
				"error":           fmt.Sprintf("inspect request failed: %v", inspectErr),
			}
		} else {
			var inspectResult map[string]interface{}
			if err := json.Unmarshal(inspectBody, &inspectResult); err == nil {
				if inspectData, ok := inspectResult["inspect"].(map[string]interface{}); ok {
					inspectSuccess = true
					// Authoritative: if ImageInspect succeeds, image exists — even if
					// Level 1 (docker images list) didn't find it.
					imagePresent = true
					entrypoint := toStringSlice(inspectData["Entrypoint"])
					cmd := toStringSlice(inspectData["Cmd"])
					repoTags := toStringSlice(inspectData["RepoTags"])
					repoDigests := toStringSlice(inspectData["RepoDigests"])
					var config map[string]interface{}
					if c, _ := inspectData["Config"].(map[string]interface{}); c != nil {
						config = c
					} else if c, _ := inspectData["ContainerConfig"].(map[string]interface{}); c != nil {
						config = c
					}
					var labels map[string]interface{}
					var exposedPorts map[string]interface{}
					var env []string
					if config != nil {
						if l, _ := config["Labels"].(map[string]interface{}); l != nil {
							labels = l
						}
						if ep, _ := config["ExposedPorts"].(map[string]interface{}); ep != nil {
							exposedPorts = ep
						}
						env = toStringSlice(config["Env"])
					}
					probeResults["level2"] = map[string]interface{}{
						"inspect_success": true,
						"image_id":        strVal(inspectData, "Id", ""),
						"repotags":        repoTags,
						"repodigests":     repoDigests,
						"architecture":    strVal(inspectData, "Architecture", ""),
						"os":              strVal(inspectData, "Os", ""),
						"size_bytes":      inspectData["Size"],
						"created":         strVal(inspectData, "Created", ""),
						"entrypoint":      entrypoint,
						"cmd":             cmd,
						"env":             env,
						"exposed_ports":   exposedPorts,
						"labels":          labels,
					}
				} else if inspectErr, _ := inspectResult["error"].(string); inspectErr != "" {
					// Check if the error indicates the image was not found.
					// Docker CLI returns "no such image" or "not found" patterns.
					if strings.Contains(strings.ToLower(inspectErr), "no such image") ||
						strings.Contains(strings.ToLower(inspectErr), "not found") ||
						strings.Contains(strings.ToLower(inspectErr), "does not exist") {
						inspectNotFound = true
					}
					probeResults["level2"] = map[string]interface{}{
						"inspect_success":   false,
						"inspect_not_found": inspectNotFound,
						"error":             inspectErr,
					}
				}
			}
		}
	} else {
		probeResults["level2"] = map[string]interface{}{
			"inspect_success": false,
			"error":           "agent not reachable or no image_ref for inspect",
		}
	}

	// ---- Level 3: Backend type matching (best-effort, lenient) ----
	backendID := strVal(rt, "backend_id", "")
	if inspectSuccess {
		l2, _ := probeResults["level2"].(map[string]interface{})
		repoTags, _ := l2["repotags"].([]string)
		labels, _ := l2["labels"].(map[string]interface{})
		probeResults["level3"] = matchBackendType(backendID, vendor, repoTags, labels)
	} else {
		probeResults["level3"] = map[string]interface{}{
			"backend_match_status": "not_checked",
			"confirmed_match":      false,
			"blocking":             false,
			"warning":              true,
			"match_detail":         "inspect data not available for backend type matching",
		}
	}

	// ---- Level 4: Version probe (DEFERRED) ----
	// The /version-probe agent endpoint is not yet enabled. It requires security
	// review (--pull=never, --network=none, --cap-drop=ALL, etc.) before deployment.
	// probe_skipped=true + skip_reason means this is NOT a warning — it's an
	// intentionally deferred feature. Only real probe failures (timeout, error,
	// version mismatch) produce warnings.
	probeResults["level4"] = map[string]interface{}{
		"version_probed": false,
		"probe_skipped":  true,
		"skip_reason":    "version probe not yet implemented; deferred to future design",
	}

	// ---- Process Start Detection (Layer 3) ----
	// Generate process_start_detection from backend_family + image inspect evidence.
	// This is a read-only system suggestion; it does NOT write process_start_config.
	backendFamily := runplan.DeriveBackendFamily(backendID)
	if inspectSuccess {
		l2, _ := probeResults["level2"].(map[string]interface{})
		imageEntrypoint, _ := l2["entrypoint"].([]string)
		imageCmd, _ := l2["cmd"].([]string)
		detection := runplan.DetectProcessStart(backendFamily, nbrImageRef, imageEntrypoint, imageCmd)
		probeResults["process_start_detection"] = detection
	} else {
		probeResults["process_start_detection"] = &runplan.ProcessStartDetection{
			Status:     "image_not_inspected",
			Confidence: "low",
			Source:     "backend_profile+image_inspect",
			Evidence: &runplan.DetectionEvidence{
				BackendFamily: backendFamily,
				ImageRef:      nbrImageRef,
			},
			Warnings: []string{"image inspect data not available; cannot detect process start config"},
		}
	}

	// ---- Evaluate final status from probe results ----
	status, reason := evaluateProbeStatus(nodeID, vendor, nbrImageRef, nodeAddr, nodePort,
		imagePresent, dockerAvailable, dockerErr, inspectSuccess, inspectNotFound, probeResults)

	// ---- Update NBR with check results ----
	now := time.Now().Format(time.RFC3339)
	probeJSON := jsonString(probeResults)
	if _, err := h.DB.Exec(`UPDATE node_backend_runtimes SET
		image_present=?, docker_available=?,
		driver_version=?, toolkit_version=?, device_check_json=?,
		status=?, status_reason=?, last_checked_at=?,
		probe_results_json=?,
		updated_at=?
		WHERE id=?`,
		boolInt(imagePresent), boolInt(dockerAvailable),
		strVal(rt, "driver_version", ""), strVal(rt, "toolkit_version", ""), jsonString(map[string]interface{}{"vendor": vendor}),
		status, reason, now,
		probeJSON,
		now, nbrID); err != nil {
		log.Error("nbr check_request update failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	probeDuration := time.Since(probeStart).Milliseconds()
	log.Info("nbr.check_request.probe",
		"node_id", nodeID, "agent_addr", nodeAddr, "agent_port", nodePort, "nbr_id", nbrID,
		"image_ref", nbrImageRef, "vendor", vendor, "backend_id", backendID,
		"status", status, "reason", reason,
		"l1_list_found", imagePresent, "l1_docker_available", dockerAvailable,
		"l2_inspect_success", inspectSuccess, "l2_inspect_not_found", inspectNotFound,
		"duration_ms", probeDuration)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":                 nbrID,
		"backend_runtime_id": nbrBackendRuntimeID,
		"node_id":            nodeID,
		"image_ref":          nbrImageRef,
		"checked_image_ref":  nbrImageRef,
		"image_present":      imagePresent,
		"docker_available":   dockerAvailable,
		"status":             status,
		"status_reason":      reason,
		"deployable":         isNBRDeployable(status),
		"warnings":           extractProbeWarnings(jsonString(probeResults), status),
		"disabled_reason":    nbrDisabledReason(status, reason),
		"last_checked_at":    now,
		"probe_results":      probeResults,
	})
}

// HandleProbeNodeBackendRuntime triggers a recheck of the NBR on the target node.
// Delegates to the existing check-request logic. Same response schema.
// POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe
func (h *AgentHandler) HandleProbeNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
	h.HandleRequestNodeBackendRuntimeCheck(w, r)
}

// HandleGetNodeBackendRuntimeProbe returns the latest probe snapshot for an NBR.
// This is a read-only endpoint — it does NOT trigger any Docker checks.
// Returns 200 with probe_results_json (may be empty {}) on success.
// GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe
func (h *AgentHandler) HandleGetNodeBackendRuntimeProbe(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	nbrID := r.PathValue("nbr_id")
	if !authz.CheckNBRTenant(r, h.DB.DB, nbrID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if nodeID == "" || nbrID == "" {
		writeError(w, http.StatusBadRequest, "node_id and nbr_id are required")
		return
	}

	var probeJSON string
	var status string
	err := h.DB.QueryRow(
		`SELECT COALESCE(probe_results_json,'{}'), status FROM node_backend_runtimes WHERE id = ? AND node_id = ?`,
		nbrID, nodeID,
	).Scan(&probeJSON, &status)
	if err != nil {
		writeError(w, http.StatusNotFound, "node backend runtime not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":                 nbrID,
		"node_id":            nodeID,
		"status":             status,
		"probe_results_json": json.RawMessage(probeJSON),
	})
}

// getNodeAddress returns the advertised address and metrics port for a node.
func (h *AgentHandler) getNodeAddress(nodeID string) (string, int) {
	var addr string
	var port int
	h.DB.QueryRow(
		`SELECT advertised_address, metrics_port FROM nodes WHERE id = ?`, nodeID,
	).Scan(&addr, &port)
	return addr, port
}

// toStringSlice converts an interface{} that may be []interface{} to []string.
func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, len(arr))
	for i, item := range arr {
		if s, ok := item.(string); ok {
			out[i] = s
		}
	}
	return out
}

func (h *AgentHandler) upsertNodeBackendRuntime(w http.ResponseWriter, r *http.Request, checkOnly bool) {
	nodeID := r.PathValue("id")
	if !authz.CheckNodeTenant(r, h.DB.DB, nodeID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
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
	if !checkOnly {
		if _, ok := req["image_present"]; ok {
			writeError(w, http.StatusBadRequest, "image_present is server/agent verified evidence; call check-request after enable")
			return
		}
		if _, ok := req["docker_available"]; ok {
			writeError(w, http.StatusBadRequest, "docker_available is server/agent verified evidence; call check-request after enable")
			return
		}
	}
	rt := h.getBackendRuntimeJSON(runtimeID)
	if rt == nil {
		writeError(w, http.StatusNotFound, "backend runtime not found")
		return
	}
	vendor := strVal(rt, "vendor", "")
	imageRef := strVal(req, "image_ref", strVal(rt, "image_ref", ""))
	displayName := strings.TrimSpace(strVal(req, "display_name", ""))
	if displayName == "" {
		displayName = strVal(rt, "display_name", "")
		if displayName == "" {
			displayName = strVal(rt, "name", runtimeID)
		}
		if nodeID != "" {
			displayName += " - " + nodeID
		}
	}
	imagePresent := boolVal(req, "image_present", false)
	dockerAvailable := boolVal(req, "docker_available", false)
	// Security: client-provided image_present/docker_available are only trusted
	// when the caller is the agent (checkOnly=true). The UI enable path
	// (checkOnly=false) must NOT set status=ready or status=unknown based on
	// unverified client claims. Server can only verify node online + vendor/GPU.
	status, reason := h.evaluateNodeBackendRuntime(nodeID, vendor, imageRef, imagePresent, dockerAvailable)
	if !checkOnly {
		// UI-initiated enable: require agent verification for Docker/image.
		// Server-verified failures (offline, unsupported_device, template_only)
		// are preserved. ready/unknown/missing_image are replaced with needs_check.
		switch status {
		case "failed", "unsupported_device", "template_only":
			// Server-verified blocking conditions — keep as-is
		default:
			status = "needs_check"
			reason = "awaiting agent verification of Docker and image availability"
		}
	} else if status == "unknown" {
		// Agent evidence check called without docker/image evidence.
		// UI must use /check-request endpoint instead.
		writeError(w, http.StatusBadRequest,
			"agent check evidence required (image_present, docker_available); UI must call /check-request endpoint instead")
		return
	}

	id := nodeID + ":" + runtimeID
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)

	// Check whether a NodeBackendRuntime already exists for this (node, runtime) pair.
	var existingID string
	row := h.DB.QueryRow(`SELECT id FROM node_backend_runtimes WHERE node_id=? AND backend_runtime_id=?`, nodeID, runtimeID)
	_ = row.Scan(&existingID)
	exists := existingID != ""

	if !exists {
		// First time: create a new NodeBackendRuntime.
		// Capture a frozen ConfigSet from the BackendRuntime template and apply
		// node/runtime explicit choices into the ConfigSet authority.
		configSetJSONRaw, buildErr := h.buildRuntimeConfigSnapshot(rt, runtimeID, imageRef, req)
		if buildErr != nil {
			writeError(w, http.StatusBadRequest, buildErr.Error())
			return
		}
		sourceMetadata := map[string]interface{}{
			"source_type":               "node_backend_runtime",
			"source_backend_runtime_id": runtimeID,
			"source_runtime_name":       strVal(rt, "name", ""),
			"source_runtime_revision":   strVal(rt, "updated_at", ""),
			"copy_semantics":            "copy_on_create",
			"copy_boundary":             "detached_after_create",
		}
		_, err := h.DB.Exec(`INSERT INTO node_backend_runtimes
			(id, backend_runtime_id, node_id, display_name, runner_type, image_ref, image_present, docker_available, driver_version, toolkit_version, device_check_json, status, status_reason, last_checked_at, config_set_json, source_metadata_json, tenant_id, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, runtimeID, nodeID, displayName, "docker", imageRef, boolInt(imagePresent), boolInt(dockerAvailable),
			strVal(req, "driver_version", ""), strVal(req, "toolkit_version", ""), jsonString(map[string]interface{}{"vendor": vendor}),
			status, reason, now, configSetJSONRaw, jsonString(sourceMetadata), tid, now, now)
		if err != nil {
			log.Error("node backend runtime insert failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	} else {
		// Existing record: update only check result fields.
		// Check/validate must NOT mutate runtime configuration fields.
		// ConfigSet and source metadata are frozen at creation time and remain
		// independent. image_ref is read from the request solely for status
		// evaluation; it is NOT persisted back.
		_, err := h.DB.Exec(`UPDATE node_backend_runtimes SET
			image_present=?, docker_available=?,
			driver_version=?, toolkit_version=?, device_check_json=?,
			status=?, status_reason=?, last_checked_at=?,
			updated_at=?
			WHERE id=?`,
			boolInt(imagePresent), boolInt(dockerAvailable),
			strVal(req, "driver_version", ""), strVal(req, "toolkit_version", ""), jsonString(map[string]interface{}{"vendor": vendor}),
			status, reason, now,
			now, existingID)
		if err != nil {
			log.Error("node backend runtime update failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": id, "backend_runtime_id": runtimeID, "node_id": nodeID, "name": displayName, "display_name": displayName,
		"image_ref": imageRef, "image_present": imagePresent, "docker_available": dockerAvailable,
		"status": status, "status_reason": reason, "last_checked_at": now,
		"deployable": isNBRDeployable(status), "disabled_reason": nbrDisabledReason(status, reason),
	})
}

// buildRuntimeConfigSnapshot captures a frozen ConfigSet from a BackendRuntime.
// This is called only at NodeBackendRuntime creation time (not on check/validate).
func (h *AgentHandler) buildRuntimeConfigSnapshot(rt map[string]interface{}, runtimeID, imageRef string, req map[string]interface{}) (string, error) {
	set := copyConfigSet(rawJSONString(rt["config_set_json"], "{}"))
	// Reject caller-provided raw config_set / config_set_json — only tiered snapshot from
	// the BackendRuntime is accepted. Use editable_config_patch for modifications.
	if _, ok := req["config_set"]; ok {
		return "", fmt.Errorf("config_set is not accepted; use editable_config_patch to modify individual parameters")
	}
	if _, ok := req["config_set_json"]; ok {
		return "", fmt.Errorf("config_set_json is not accepted; use editable_config_patch to modify individual parameters")
	}
	if imageRef != "" {
		setConfigValue(set, "launcher.image", imageRef, "NodeBackendRuntime", runtimeID, "explicit_node_runtime_image")
	}
	if overrides := mapFromAny(req["config_overrides"]); len(overrides) > 0 {
		applyConfigOverrides(set, overrides, "NodeBackendRuntime", runtimeID)
	}
	patched, err := applyEditableConfigPatchIfPresent(set, req, "node_backend_runtime", runtimeID)
	if err != nil {
		return "", err
	}
	return configSetJSON(patched), nil
}

func (h *AgentHandler) evaluateNodeBackendRuntime(nodeID, vendor, imageRef string, imagePresent, dockerAvailable bool) (string, string) {
	var nodeStatus string
	if err := h.DB.QueryRow(`SELECT status FROM nodes WHERE id=?`, nodeID).Scan(&nodeStatus); err != nil || nodeStatus != "online" {
		return "failed", "node is offline"
	}
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

// evaluateProbeStatus determines the final NBR status from probe results.
// ImageInspect (Level 2) is the AUTHORITATIVE source for image existence.
// missing_image is ONLY returned when ImageInspect explicitly says "not found".
// docker-images list (Level 1) is evidence only and never produces missing_image.
func evaluateProbeStatus(nodeID, vendor, imageRef, agentAddr string, agentPort int,
	imagePresent, dockerAvailable bool,
	dockerErr string,
	inspectSuccess, inspectNotFound bool,
	probeResults map[string]interface{}) (string, string) {

	if agentAddr == "" || agentPort == 0 {
		return "agent_unreachable", dockerErr
	}
	if !dockerAvailable && dockerErr != "" {
		if strings.Contains(dockerErr, "agent unreachable") {
			return "agent_unreachable", dockerErr
		}
		return "docker_error", dockerErr
	}
	if imageRef == "" {
		return "evidence_missing", "no image_ref configured for this node backend runtime"
	}

	// Authoritative check: ImageInspect result.
	if inspectNotFound {
		return "missing_image", fmt.Sprintf("docker image %s is not present on node %s (ImageInspect: not found)", imageRef, nodeID)
	}
	if !inspectSuccess {
		l2, _ := probeResults["level2"].(map[string]interface{})
		l2Err := ""
		if l2 != nil {
			l2Err = strVal(l2, "error", "")
		}
		if l2Err == "" {
			l2Err = "docker image inspect failed"
		}
		return "inspect_failed", fmt.Sprintf("docker inspect failed for image %s: %s", imageRef, l2Err)
	}

	// ImageInspect succeeded — image exists. Check for warnings.
	var warnings []string

	l3, _ := probeResults["level3"].(map[string]interface{})
	if l3 != nil {
		matchStatus, _ := l3["backend_match_status"].(string)
		if matchStatus == "confirmed_mismatch" {
			return "runtime_image_mismatch", fmt.Sprintf("image %s does not match expected backend", imageRef)
		}
		if matchStatus == "declared_match_unverified" || matchStatus == "ambiguous" {
			warnings = append(warnings, fmt.Sprintf("backend match: %s", matchStatus))
		}
	}

	l4, _ := probeResults["level4"].(map[string]interface{})
	if l4 != nil && !boolVal(l4, "version_probed", false) {
		// If probe was intentionally skipped (deferred/not-implemented), do NOT
		// treat it as a warning. Only real probe failures produce warnings.
		if !boolVal(l4, "probe_skipped", false) {
			warnings = append(warnings, "version not probed")
		}
	}

	if len(warnings) > 0 {
		return "ready_with_warnings", fmt.Sprintf("image %s verified (inspect ok); warnings: %s", imageRef, strings.Join(warnings, "; "))
	}
	return "ready", fmt.Sprintf("runtime verified for node (image=%s)", imageRef)
}

// matchBackendType checks whether the Docker image's RepoTags and labels are
// consistent with the expected Backend. Returns a map with structured fields:
//
//	backend_match_status: confirmed_match | probable_match | declared_match_unverified | ambiguous | not_checked
//	confirmed_match: true/false — only true when pattern/label match is strong
//	blocking: true/false — only true for confirmed_mismatch
//	warning: true/false
//
// IMPORTANT: vendor is NOT used to derive backend (vendor=nvidia != vllm).
// Vendor-built images that don't match known patterns are treated leniently.
func matchBackendType(backendID, vendor string, repoTags []string, labels map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"backend_match_status": "not_checked",
		"confirmed_match":      false,
		"blocking":             false,
		"warning":              false,
		"match_method":         "",
		"match_detail":         "",
		"backend_id":           backendID,
	}

	allTags := strings.Join(repoTags, " ")
	allTagsLower := strings.ToLower(allTags)

	labelStr := ""
	if labels != nil {
		parts := make([]string, 0, len(labels))
		for k, v := range labels {
			if vs, ok := v.(string); ok {
				parts = append(parts, k+"="+vs)
			}
		}
		labelStr = strings.ToLower(strings.Join(parts, " "))
	}

	baseID := backendID
	if strings.HasPrefix(baseID, "backend.") {
		baseID = baseID[len("backend."):]
	}
	// patterns maps backend family names to common image name variants.
	// This is for user input normalization only — NOT a backend capability source.
	// Canonical capability data is in BackendVersion ConfigSet.
	patterns := map[string][]string{
		"vllm":     {"vllm", "vllm-openai"},
		"sglang":   {"sglang", "lmsysorg/sglang"},
		"llamacpp": {"llama.cpp", "llama-cpp", "llamacpp", "ghcr.io/ggml-org/llama.cpp"},
		"ollama":   {"ollama"},
	}

	expectedPatterns, ok := patterns[baseID]
	if !ok {
		result["backend_match_status"] = "declared_match_unverified"
		result["confirmed_match"] = false
		result["warning"] = true
		result["match_method"] = "unknown_backend"
		result["match_detail"] = fmt.Sprintf("backend_id '%s' has no matching patterns; declared match not verified", baseID)
		return result
	}

	for _, p := range expectedPatterns {
		if strings.Contains(allTagsLower, p) {
			result["backend_match_status"] = "confirmed_match"
			result["confirmed_match"] = true
			result["match_method"] = "repo_pattern"
			result["match_detail"] = fmt.Sprintf("repo tags match pattern '%s' for backend '%s'", p, backendID)
			return result
		}
		if strings.Contains(labelStr, p) {
			result["backend_match_status"] = "confirmed_match"
			result["confirmed_match"] = true
			result["match_method"] = "label_match"
			result["match_detail"] = fmt.Sprintf("labels match pattern '%s' for backend '%s'", p, backendID)
			return result
		}
	}

	// No pattern matched. Not a mismatch — just unverified.
	result["backend_match_status"] = "declared_match_unverified"
	result["confirmed_match"] = false
	result["blocking"] = false
	result["warning"] = true
	result["match_method"] = "no_pattern_match"
	result["match_detail"] = fmt.Sprintf("no pattern matched for backend '%s' (expected: %v) in repo tags %v; declared match not verified", backendID, expectedPatterns, repoTags)
	return result
}

// getVersionProbeConfig extracts the version_probe configuration from a
// BackendRuntime template's version_snapshot_json. Returns nil if not configured.
// NOTE: Version probe execution is DEFERRED. This function exists to support
// future catalog-driven probe configuration.
func getVersionProbeConfig(rt map[string]interface{}) map[string]interface{} {
	return nil
}

func (h *AgentHandler) getBackendRuntimeJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, is_builtin, is_editable, tenant_id, status, visibility, support_level, config_set_json, source_metadata_json, created_at, updated_at FROM backend_runtimes WHERE id = ?`, id)
	var rid, name, dn, bid, bvid, stn, vendor, rt, tid, status, visibility, supportLevel, configSetRaw, sourceMetaRaw, ca, ua string
	var isB, isE int
	if err := row.Scan(&rid, &name, &dn, &bid, &bvid, &stn, &vendor, &rt, &isB, &isE, &tid, &status, &visibility, &supportLevel, &configSetRaw, &sourceMetaRaw, &ca, &ua); err != nil {
		return nil
	}
	configSet := parseConfigSet(configSetRaw)
	sourceMeta := configSourceMetadata(sourceMetaRaw)
	return map[string]interface{}{
		"id": rid, "name": name, "display_name": dn, "backend_id": bid, "backend_version_id": bvid,
		"source_template_name": stn, "vendor": vendor, "runtime_type": rt,
		"is_builtin": isB == 1, "is_editable": isE == 1, "tenant_id": tid,
		"status": status, "visibility": visibility, "support_level": supportLevel,
		"image_ref":            configString(configSet, "launcher.image", ""),
		"entrypoint":           configStringSlice(configSet, "launcher.entrypoint"),
		"command":              configStringSlice(configSet, "launcher.command"),
		"env":                  redactEnvMap(configObject(configSet, "runtime.env")),
		"docker_options":       configObject(configSet, "launcher.docker_options"),
		"model_mount":          configObject(configSet, "runtime.model_mount"),
		"health_check":         configObject(configSet, "runtime.health"),
		"config_set":           configSet,
		"config_set_json":      json.RawMessage(configSetRaw),
		"source_metadata":      sourceMeta,
		"source_metadata_json": json.RawMessage(sourceMetaRaw),
		"created_at":           ca, "updated_at": ua,
	}
}

func publicBackendRuntimeList(in []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(in))
	for _, item := range in {
		out = append(out, publicBackendRuntimeJSON(item))
	}
	return out
}

func publicBackendRuntimeJSON(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	delete(out, "config_set_json")
	delete(out, "source_metadata_json")
	return out
}

func (h *AgentHandler) queryBackendRuntimes(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var rid, name, dn, bid, bvid, stn, vendor, rt, tid, status, visibility, supportLevel, configSetRaw, sourceMetaRaw, ca, ua string
		var isB, isE int
		if err := rows.Scan(&rid, &name, &dn, &bid, &bvid, &stn, &vendor, &rt, &isB, &isE, &tid, &status, &visibility, &supportLevel, &configSetRaw, &sourceMetaRaw, &ca, &ua); err != nil {
			continue
		}
		configSet := parseConfigSet(configSetRaw)
		out = append(out, map[string]interface{}{
			"id": rid, "name": name, "display_name": dn, "backend_id": bid, "backend_version_id": bvid,
			"source_template_name": stn, "vendor": vendor, "runtime_type": rt,
			"is_builtin": isB == 1, "is_editable": isE == 1, "tenant_id": tid,
			"status": status, "visibility": visibility, "support_level": supportLevel,
			"image_ref":            configString(configSet, "launcher.image", ""),
			"entrypoint":           configStringSlice(configSet, "launcher.entrypoint"),
			"command":              configStringSlice(configSet, "launcher.command"),
			"env":                  redactEnvMap(configObject(configSet, "runtime.env")),
			"docker_options":       configObject(configSet, "launcher.docker_options"),
			"model_mount":          configObject(configSet, "runtime.model_mount"),
			"health_check":         configObject(configSet, "runtime.health"),
			"config_set":           configSet,
			"config_set_json":      json.RawMessage(configSetRaw),
			"source_metadata":      configSourceMetadata(sourceMetaRaw),
			"source_metadata_json": json.RawMessage(sourceMetaRaw),
			"created_at":           ca, "updated_at": ua,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out, nil
}

func jsonToStringMap(raw string) map[string]string {
	var src map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &src); err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = strings.TrimSpace(strings.Trim(fmt.Sprint(v), `"`))
	}
	return out
}

func jsonToStringSlice(raw string) []string {
	var arr []interface{}
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
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

// resolveTemplatePath returns the config file path for a template name
// using the backend-catalog layout: configs/backend-catalog/runtimes/{backend}/{vendor}-docker.yaml.
// Template names follow the pattern {backend}-{vendor}-docker.
// The old configs/model-runtime/ flat layout has been removed (Phase 3 Batch 4).
func resolveTemplatePath(templateName string) string {
	if idx := strings.Index(templateName, "-"); idx > 0 {
		backend := templateName[:idx]
		rest := templateName[idx+1:]
		newPath := "configs/backend-catalog/runtimes/" + backend + "/" + rest + ".yaml"
		if _, err := osReadFile(newPath); err == nil {
			return newPath
		}
	}
	// Fallback to the name under the first path segment as backend.
	// If templateName has no "-", try it as a direct file under the runtimes dir.
	return ""
}

// isNBRDeployable returns true when the NBR status allows deployment.
// ready and ready_with_warnings are deployable; all other statuses are not.
// This is the single source of truth for NBR deployability — all callers
// (create deployment, preflight, frontend) must use this helper.
func isNBRDeployable(status string) bool {
	return status == "ready" || status == "ready_with_warnings"
}

// extractProbeWarnings extracts user-visible warnings from probe_results_json.
// Returns nil when there are no warnings. Skipped/deferred probes are NOT warnings.
func extractProbeWarnings(probeResultsJSON string, status string) []string {
	if probeResultsJSON == "" || probeResultsJSON == "{}" {
		return nil
	}
	var pr map[string]interface{}
	if json.Unmarshal([]byte(probeResultsJSON), &pr) != nil {
		return nil
	}
	var warnings []string
	// Level 3: backend match warnings
	if l3, ok := pr["level3"].(map[string]interface{}); ok {
		if ws, _ := l3["warning"].(bool); ws {
			if detail := strVal(l3, "match_detail", ""); detail != "" {
				warnings = append(warnings, "backend_match: "+detail)
			}
		}
	}
	// Level 4: version probe warnings (only when not skipped)
	if l4, ok := pr["level4"].(map[string]interface{}); ok {
		if !boolVal(l4, "version_probed", false) && !boolVal(l4, "probe_skipped", false) {
			warnings = append(warnings, "version not probed")
		}
	}
	// Process start detection warnings
	if psd, ok := pr["process_start_detection"].(map[string]interface{}); ok {
		if psdWarnings, ok := psd["warnings"].([]interface{}); ok {
			for _, w := range psdWarnings {
				if s, ok := w.(string); ok && s != "" {
					warnings = append(warnings, s)
				}
			}
		}
	}
	if len(warnings) == 0 {
		return nil
	}
	return warnings
}

// nbrDisabledReason returns a human-readable reason when an NBR is not deployable.
// Returns empty string when the NBR is deployable.
func nbrDisabledReason(status, reason string) string {
	if isNBRDeployable(status) {
		return ""
	}
	switch status {
	case "missing_image":
		return "Docker image is not present on the node; pull the image or check the image reference"
	case "needs_check":
		return "Node runtime config has not been checked; run agent check first"
	case "inspect_failed":
		return "Docker image inspect failed; verify Docker daemon and image availability"
	case "runtime_image_mismatch":
		return "Docker image does not match the declared backend; verify the image reference"
	case "agent_unreachable":
		return "Agent is unreachable; verify node connectivity and agent status"
	case "docker_error":
		return "Docker daemon error; verify Docker is running on the node"
	case "unsupported_device":
		return "Node has no matching GPU vendor for this runtime"
	case "disabled":
		return "Node runtime config is disabled"
	case "evidence_missing":
		return "No image reference configured; set an image_ref before checking"
	case "node_offline", "failed":
		return "Node is offline or in failed state"
	default:
		if reason != "" {
			return reason
		}
		return "Node runtime is not deployable (status=" + status + ")"
	}
}

// osReadFile is a wrapper for testing.
var osReadFile = os.ReadFile //nolint
