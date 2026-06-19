package api

import (
	"encoding/json"
	"fmt"
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
	if templateName != "" {
		// Read template from config file for backward compatibility with the
		// older clone-from-template flow.
		path := resolveTemplatePath(templateName)
		_, err := osReadFile(path)
		if err != nil {
			writeError(w, http.StatusNotFound, "template not found: "+templateName)
			return
		}
	}

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
	defaultImages := jsonToStringMap(rawJSONString(version["default_images_json"], "{}"))
	imageCandidates := jsonToStringSlice(rawJSONString(version["image_candidates_json"], "[]"))
	imageName := strVal(req, "image_name", "")
	if imageName == "" {
		imageName = defaultImages[vendor]
	}
	if imageName == "" {
		imageName = defaultImages["default"]
	}
	if imageName == "" && len(imageCandidates) > 0 {
		imageName = imageCandidates[0]
	}
	versionSnapshot := backendVersionSnapshot(version)
	sourceRevision := strVal(version, "checksum", "")
	if sourceRevision == "" {
		sourceRevision = strVal(version, "updated_at", "")
	}
	dockerJSON := jsonField(req, "docker_json", rawJSONString(version["docker_options_json"], "{}"))
	modelMountJSON := jsonField(req, "model_mount_json", rawJSONString(version["model_mount_json"], "{}"))
	defaultEnvJSON := jsonField(req, "default_env_json", rawJSONString(version["env_json"], "{}"))

	_, err := h.DB.Exec(
		`INSERT INTO backend_runtimes (id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, source_backend_id, source_backend_version_id, source_version_revision, version_snapshot_json, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, displayName, backendID, versionID, templateName,
		vendor, "docker",
		imageName, strVal(req, "image_pull_policy", "if_not_present"),
		jsonField(req, "entrypoint_override_json", "[]"), jsonField(req, "args_override_json", "[]"), defaultEnvJSON, dockerJSON, modelMountJSON, jsonField(req, "health_check_override_json", "{}"),
		0, 1, tid, backendID, versionID, sourceRevision, jsonString(versionSnapshot), now, now,
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
	rows, err := h.DB.Query(`SELECT nbr.id, nbr.backend_runtime_id, nbr.node_id, COALESCE(nbr.display_name,''), nbr.runner_type, nbr.image_ref, nbr.image_id, nbr.image_digest, nbr.image_present, nbr.docker_available, nbr.driver_version, nbr.toolkit_version, nbr.device_check_json, nbr.status, nbr.status_reason, nbr.last_checked_at, nbr.tenant_id, nbr.created_at, nbr.updated_at, COALESCE(nbr.config_snapshot_json,'{}'), br.name, br.display_name, br.vendor
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
		var id, runtimeID, nid, displayName, runner, imageRef, imageID, digest, driver, toolkit, checkJSON, status, reason, checked, tid, ca, ua, snapshotJSON, rtName, rtDisplay, vendor string
		var imagePresent, dockerAvailable int
		if err := rows.Scan(&id, &runtimeID, &nid, &displayName, &runner, &imageRef, &imageID, &digest, &imagePresent, &dockerAvailable, &driver, &toolkit, &checkJSON, &status, &reason, &checked, &tid, &ca, &ua, &snapshotJSON, &rtName, &rtDisplay, &vendor); err != nil {
			continue
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
		out = append(out, map[string]interface{}{
			"id": id, "backend_runtime_id": runtimeID, "node_id": nid, "name": displayName, "display_name": displayName, "runner_type": runner,
			"image_ref": imageRef, "image_id": imageID, "image_digest": digest,
			"image_present": imagePresent == 1, "docker_available": dockerAvailable == 1,
			"driver_version": driver, "toolkit_version": toolkit, "device_check_json": json.RawMessage(checkJSON),
			"status": status, "status_reason": reason, "last_checked_at": checked, "tenant_id": tid,
			"created_at": ca, "updated_at": ua,
			"config_snapshot_json": json.RawMessage(snapshotJSON),
			"backend_runtime":      map[string]interface{}{"name": rtName, "display_name": rtDisplay, "vendor": vendor},
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

// HandleRequestNodeBackendRuntimeCheck is the UI-facing check endpoint.
// It does NOT accept client-provided image_present/docker_available.
// Instead, the server queries the agent for Docker image status and
// evaluates readiness with server-verified evidence.
// POST /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/check-request
func (h *AgentHandler) HandleRequestNodeBackendRuntimeCheck(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	nbrID := r.PathValue("nbr_id")
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
		nbrImageRef = strVal(rt, "image_name", "")
	}
	vendor := strVal(rt, "vendor", "")

	// Query Docker images on the node (server-to-agent proxy).
	dockerAvailable := false
	imagePresent := false
	var dockerErr string

	nodeAddr, nodePort := h.getNodeAddress(nodeID)
	if nodeAddr != "" && nodePort > 0 {
		agentURL := fmt.Sprintf("http://%s:%d/docker-images?limit=1000", nodeAddr, nodePort)
		resp, err := http.Get(agentURL)
		if err != nil {
			dockerErr = fmt.Sprintf("agent unreachable: %v", err)
			log.Warn("nbr.check_request.agent_unreachable", "node_id", nodeID, "url", agentURL, "error", err)
		} else {
			defer resp.Body.Close()
			dockerAvailable = true
			if nbrImageRef != "" {
				var result struct {
					Images []struct {
						RepoTags []string `json:"repotags"`
						ID       string   `json:"id"`
					} `json:"images"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					// Try legacy flat array format.
					resp.Body.Close()
					resp2, err2 := http.Get(agentURL)
					if err2 == nil {
						defer resp2.Body.Close()
						var images []map[string]interface{}
						if json.NewDecoder(resp2.Body).Decode(&images) == nil {
							for _, img := range images {
								tags := toStringSlice(img["repotags"])
								for _, t := range tags {
									if nbrImageRef == t || strings.Contains(t, nbrImageRef) {
										imagePresent = true
										break
									}
								}
								if imagePresent {
									break
								}
							}
						}
					}
				} else {
					for _, img := range result.Images {
						for _, t := range img.RepoTags {
							if nbrImageRef == t || strings.Contains(t, nbrImageRef) {
								imagePresent = true
								break
							}
						}
						if imagePresent {
							break
						}
					}
				}
			}
		}
	} else {
		dockerErr = "node has no advertised address or metrics port"
	}

	// Evaluate with server-verified evidence.
	status, reason := h.evaluateNodeBackendRuntime(nodeID, vendor, nbrImageRef, imagePresent, dockerAvailable)
	if !dockerAvailable && dockerErr != "" {
		reason = dockerErr
	}
	if status == "missing_image" && nbrImageRef != "" {
		reason = fmt.Sprintf("docker image %s is not present on node %s", nbrImageRef, nodeID)
	}

	// Update NBR with check results.
	now := time.Now().Format(time.RFC3339)
	if _, err := h.DB.Exec(`UPDATE node_backend_runtimes SET
		image_present=?, docker_available=?,
		status=?, status_reason=?, last_checked_at=?,
		updated_at=?
		WHERE id=?`,
		boolInt(imagePresent), boolInt(dockerAvailable),
		status, reason, now,
		now, nbrID); err != nil {
		log.Error("nbr check_request update failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("nbr.check_request.completed",
		"node_id", nodeID, "nbr_id", nbrID,
		"status", status, "image_present", imagePresent, "docker_available", dockerAvailable)

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
		"last_checked_at":    now,
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
	var existingID, existingSnapshot string
	row := h.DB.QueryRow(`SELECT id, COALESCE(config_snapshot_json,'{}') FROM node_backend_runtimes WHERE node_id=? AND backend_runtime_id=?`, nodeID, runtimeID)
	_ = row.Scan(&existingID, &existingSnapshot)
	exists := existingID != ""
	hasSnapshot := exists && existingSnapshot != "{}" && existingSnapshot != ""

	if !exists {
		// First time: create a new NodeBackendRuntime.
		// Capture a frozen config snapshot from the BackendRuntime template.
		// This freezes the runtime configuration so future template edits do not
		// silently change the behavior of existing NodeBackendRuntime records.
		snapshotJSON := h.buildRuntimeConfigSnapshot(rt, runtimeID)
		_, err := h.DB.Exec(`INSERT INTO node_backend_runtimes
			(id, backend_runtime_id, node_id, display_name, runner_type, image_ref, image_present, docker_available, driver_version, toolkit_version, device_check_json, status, status_reason, last_checked_at, config_snapshot_json, source_runtime_name, source_runtime_revision, tenant_id, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, runtimeID, nodeID, displayName, "docker", imageRef, boolInt(imagePresent), boolInt(dockerAvailable),
			strVal(req, "driver_version", ""), strVal(req, "toolkit_version", ""), jsonString(map[string]interface{}{"vendor": vendor}),
			status, reason, now, snapshotJSON, strVal(rt, "name", ""), strVal(rt, "updated_at", ""), tid, now, now)
		if err != nil {
			log.Error("node backend runtime insert failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	} else if checkOnly || hasSnapshot {
		// Existing record: update only check result fields.
		// Check/validate must NOT mutate runtime configuration fields
		// (image_ref, config_snapshot_json, source_runtime_name,
		//  source_runtime_revision). The snapshot is frozen at creation
		// time and remains independent. image_ref is read from the
		// request solely for status evaluation; it is NOT persisted back.
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
	} else {
		// Legacy case: existing record with no snapshot — rebuild from template.
		// This path only triggers for records created before the snapshot feature.
		snapshotJSON := h.buildRuntimeConfigSnapshot(rt, runtimeID)
		_, err := h.DB.Exec(`UPDATE node_backend_runtimes SET
			image_ref=?, image_present=?, docker_available=?,
			driver_version=?, toolkit_version=?, device_check_json=?,
			status=?, status_reason=?, last_checked_at=?,
			config_snapshot_json=?, source_runtime_name=?, source_runtime_revision=?,
			updated_at=?
			WHERE id=?`,
			imageRef, boolInt(imagePresent), boolInt(dockerAvailable),
			strVal(req, "driver_version", ""), strVal(req, "toolkit_version", ""), jsonString(map[string]interface{}{"vendor": vendor}),
			status, reason, now,
			snapshotJSON, strVal(rt, "name", ""), strVal(rt, "updated_at", ""),
			now, existingID)
		if err != nil {
			log.Error("node backend runtime legacy update failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": id, "backend_runtime_id": runtimeID, "node_id": nodeID, "name": displayName, "display_name": displayName,
		"image_ref": imageRef, "image_present": imagePresent, "docker_available": dockerAvailable,
		"status": status, "status_reason": reason, "last_checked_at": now,
	})
}

// buildRuntimeConfigSnapshot captures a frozen config snapshot from a BackendRuntime.
// This is called only at NodeBackendRuntime creation time (not on check/validate).
func (h *AgentHandler) buildRuntimeConfigSnapshot(rt map[string]interface{}, runtimeID string) string {
	snapshot := map[string]interface{}{
		"source_runtime_id":          runtimeID,
		"source_runtime_name":        strVal(rt, "name", ""),
		"source_runtime_revision":    strVal(rt, "updated_at", ""),
		"backend_id":                 strVal(rt, "backend_id", ""),
		"backend_version_id":         strVal(rt, "backend_version_id", ""),
		"vendor":                     strVal(rt, "vendor", ""),
		"runtime_type":               strVal(rt, "runtime_type", "docker"),
		"image_name":                 strVal(rt, "image_name", ""),
		"image_pull_policy":          strVal(rt, "image_pull_policy", "if_not_present"),
		"entrypoint_override_json":   rt["entrypoint_override_json"],
		"args_override_json":         rt["args_override_json"],
		"default_env_json":           rt["default_env_json"],
		"docker_json":                rt["docker_json"],
		"model_mount_json":           rt["model_mount_json"],
		"health_check_override_json": rt["health_check_override_json"],
		"version_snapshot_json":      rt["version_snapshot_json"],
	}
	return jsonString(snapshot)
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

func (h *AgentHandler) getBackendRuntimeJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, source_backend_id, source_backend_version_id, source_version_revision, version_snapshot_json, created_at, updated_at FROM backend_runtimes WHERE id = ?`, id)
	var rid, name, dn, bid, bvid, stn, vendor, rt, img, ipp, eoj, aoj, defEnv, dj, mmj, hcoj, tid, sourceBID, sourceBVID, sourceRevision, versionSnapshot, ca, ua string
	var isB, isE int
	if err := row.Scan(&rid, &name, &dn, &bid, &bvid, &stn, &vendor, &rt, &img, &ipp, &eoj, &aoj, &defEnv, &dj, &mmj, &hcoj, &isB, &isE, &tid, &sourceBID, &sourceBVID, &sourceRevision, &versionSnapshot, &ca, &ua); err != nil {
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
		"source_backend_id": sourceBID, "source_backend_version_id": sourceBVID,
		"source_version_revision": sourceRevision, "version_snapshot_json": json.RawMessage(versionSnapshot),
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

func backendVersionSnapshot(version map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":                          strVal(version, "id", ""),
		"backend_id":                  strVal(version, "backend_id", ""),
		"version":                     strVal(version, "version", ""),
		"display_name":                strVal(version, "display_name", ""),
		"default_entrypoint_json":     version["default_entrypoint_json"],
		"default_args_json":           version["default_args_json"],
		"default_backend_params_json": version["default_backend_params_json"],
		"parameter_defs_json":         version["parameter_defs_json"],
		"health_check_json":           version["health_check_json"],
		"default_container_port":      intVal(version, "default_container_port", 8000),
		"default_images_json":         version["default_images_json"],
		"image_candidates_json":       version["image_candidates_json"],
		"protocol":                    strVal(version, "protocol", ""),
		"default_host":                strVal(version, "default_host", "0.0.0.0"),
		"default_endpoints_json":      version["default_endpoints_json"],
		"default_args_schema_json":    version["default_args_schema_json"],
		"default_env_schema_json":     version["default_env_schema_json"],
		"default_health_check_json":   version["default_health_check_json"],
		"official_reference_json":     version["official_reference_json"],
		"revision":                    strVal(version, "revision", ""),
		"env_json":                    version["env_json"],
		"capabilities_json":           version["capabilities_json"],
		"docker_options_json":         version["docker_options_json"],
		"model_mount_json":            version["model_mount_json"],
		"vendor_options_json":         version["vendor_options_json"],
		"source_revision":             strVal(version, "checksum", strVal(version, "updated_at", "")),
	}
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
