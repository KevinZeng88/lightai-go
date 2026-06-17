package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"lightai-go/internal/common/log"

	"github.com/google/uuid"
)

// ==========================================================================
// ModelArtifact CRUD
// ==========================================================================

func (h *AgentHandler) HandleListArtifacts(w http.ResponseWriter, r *http.Request) {
	tid := tenantID(r)
	var err error
	var out []map[string]interface{}
	if isPlatformAdmin(r) {
		out, err = h.queryArtifacts(`SELECT id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, tenant_id, created_at, updated_at FROM model_artifacts ORDER BY name`)
	} else {
		out, err = h.queryArtifacts(`SELECT id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, tenant_id, created_at, updated_at FROM model_artifacts WHERE tenant_id = ? ORDER BY name`, tid)
	}
	if err != nil {
		log.Error("list artifacts", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleCreateArtifact(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "model_artifact.create")
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	name := strVal(req, "name", "")
	if name == "" {
		log.OpWarn("model_artifact.create", "input_validated", "error", "name required")
		writeError(w, http.StatusBadRequest, "name required")
		return
	}

	id := uuid.NewString()
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)

	_, err := h.DB.Exec(`INSERT INTO model_artifacts (id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, tenant_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), strVal(req, "source_type", "local_path"),
		strVal(req, "path", ""), strVal(req, "format", "custom"), strVal(req, "task_type", "chat"),
		strVal(req, "architecture", "custom"), strVal(req, "size_label", ""),
		strVal(req, "quantization", "unknown"), intVal(req, "default_context_length", 0),
		int64Val(req, "estimated_vram_bytes", 0), intVal(req, "required_gpu_count", 1),
		tid, now, now,
	)
	if err != nil {
		log.OperationFailed(ctx, "model_artifact.create", "db_write", opStart, err, "id", id, "name", name)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "model_artifact.create", opStart, "id", id, "name", name, "tenant_id", tid)
	writeJSON(w, http.StatusCreated, h.getArtifactJSON(id))
}

func (h *AgentHandler) HandleGetArtifact(w http.ResponseWriter, r *http.Request) {
	m := h.getArtifactJSON(r.PathValue("id"))
	if m == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	tid, _ := m["tenant_id"].(string)
	if tid != "" && !tenantScopeCheck(r, tid) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *AgentHandler) HandlePatchArtifact(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "model_artifact.update")
	id := r.PathValue("id")
	existing := h.getArtifactJSON(id)
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
	for _, f := range []string{"display_name", "path", "format", "source_type", "task_type", "architecture", "size_label", "quantization"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	for _, f := range []string{"default_context_length", "required_gpu_count"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	if v, ok := req["estimated_vram_bytes"]; ok {
		sets = append(sets, "estimated_vram_bytes = ?")
		args = append(args, v)
	}
	args = append(args, id)
	_, err := h.DB.Exec(`UPDATE model_artifacts SET `+joinSets(sets)+` WHERE id = ?`, args...)
	if err != nil {
		log.OperationFailed(ctx, "model_artifact.update", "db_write", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "model_artifact.update", opStart, "id", id)
	writeJSON(w, http.StatusOK, h.getArtifactJSON(id))
}

func (h *AgentHandler) HandleDeleteArtifact(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "model_artifact.delete")
	id := r.PathValue("id")
	existing := h.getArtifactJSON(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	_, err := h.DB.Exec(`DELETE FROM model_artifacts WHERE id = ?`, id)
	if err != nil {
		log.OperationFailed(ctx, "model_artifact.delete", "db_write", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "model_artifact.delete", opStart, "id", id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AgentHandler) getArtifactJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, tenant_id, created_at, updated_at FROM model_artifacts WHERE id = ?`, id)
	var rid, name, dn, st, path, frmt, tt, arch, sl, quant, tid, ca, ua string
	var ctxLen, gpuCount int
	var vram int64
	if err := row.Scan(&rid, &name, &dn, &st, &path, &frmt, &tt, &arch, &sl, &quant, &ctxLen, &vram, &gpuCount, &tid, &ca, &ua); err != nil {
		return nil
	}
	return map[string]interface{}{"id": rid, "name": name, "display_name": dn, "source_type": st, "path": path, "format": frmt, "task_type": tt, "architecture": arch, "size_label": sl, "quantization": quant, "default_context_length": ctxLen, "estimated_vram_bytes": vram, "required_gpu_count": gpuCount, "tenant_id": tid, "created_at": ca, "updated_at": ua, "locations": h.listModelLocations(id)}
}

func (h *AgentHandler) queryArtifacts(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var rid, name, dn, st, path, frmt, tt, arch, sl, quant, tid, ca, ua string
		var ctxLen, gpuCount int
		var vram int64
		if err := rows.Scan(&rid, &name, &dn, &st, &path, &frmt, &tt, &arch, &sl, &quant, &ctxLen, &vram, &gpuCount, &tid, &ca, &ua); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{"id": rid, "name": name, "display_name": dn, "source_type": st, "path": path, "format": frmt, "task_type": tt, "architecture": arch, "size_label": sl, "quantization": quant, "default_context_length": ctxLen, "estimated_vram_bytes": vram, "required_gpu_count": gpuCount, "tenant_id": tid, "created_at": ca, "updated_at": ua})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out, nil
}

func (h *AgentHandler) HandleDiscoverArtifact(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if strVal(req, "node_id", "") == "" || strVal(req, "path", "") == "" {
		writeError(w, http.StatusBadRequest, "node_id and path are required")
		return
	}
	name := strVal(req, "name", filepath.Base(strVal(req, "path", "")))
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	id := uuid.NewString()
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)
	absolutePath := strVal(req, "path", "")
	_, err := h.DB.Exec(`INSERT INTO model_artifacts (id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, tenant_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), strVal(req, "source_type", "local_path"),
		absolutePath, strVal(req, "format", "custom"), strVal(req, "task_type", "chat"),
		strVal(req, "architecture", "custom"), strVal(req, "size_label", ""),
		strVal(req, "quantization", "unknown"), intVal(req, "default_context_length", 0),
		int64Val(req, "estimated_vram_bytes", 0), intVal(req, "required_gpu_count", 1),
		tid, now, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	locReq := map[string]interface{}{
		"node_id":       strVal(req, "node_id", ""),
		"absolute_path": absolutePath,
		"path_type":     strVal(req, "path_type", "directory"),
	}
	locationID := uuid.NewString()
	_, _ = h.DB.Exec(`INSERT INTO model_locations (id, model_artifact_id, node_id, path_type, model_root, relative_path, absolute_path, verification_status, match_status, tenant_id, last_scanned_at, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		locationID, id, locReq["node_id"], locReq["path_type"], filepath.Dir(absolutePath), filepath.Base(absolutePath), absolutePath, "verified", "exact_match", tid, now, now, now)
	writeJSON(w, http.StatusCreated, h.getArtifactJSON(id))
}

func (h *AgentHandler) HandleCreateModelLocation(w http.ResponseWriter, r *http.Request) {
	artifactID := r.PathValue("id")
	artifact := h.getArtifactJSON(artifactID)
	if artifact == nil {
		writeError(w, http.StatusNotFound, "model artifact not found")
		return
	}
	if !tenantScopeCheck(r, artifact["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	nodeID := strVal(req, "node_id", "")
	absolutePath := strVal(req, "absolute_path", strVal(req, "path", ""))
	if nodeID == "" || absolutePath == "" {
		writeError(w, http.StatusBadRequest, "node_id and absolute_path are required")
		return
	}
	pathType := strVal(req, "path_type", "directory")
	modelRoot := strVal(req, "model_root", "")
	relativePath := strVal(req, "relative_path", "")
	if modelRoot == "" {
		modelRoot = filepath.Dir(absolutePath)
	}
	if relativePath == "" {
		relativePath = filepath.Base(absolutePath)
	}
	id := uuid.NewString()
	tid := artifact["tenant_id"].(string)
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`INSERT INTO model_locations
		(id, model_artifact_id, node_id, path_type, model_root, relative_path, absolute_path, size_bytes, checksum, manifest_digest, discovered_metadata_json, match_status, verification_status, manual_override, tenant_id, last_scanned_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, artifactID, nodeID, pathType, modelRoot, relativePath, absolutePath,
		int64Val(req, "size_bytes", 0), strVal(req, "checksum", ""), strVal(req, "manifest_digest", ""),
		jsonString(req["discovered_metadata_json"]), strVal(req, "match_status", "exact_match"),
		strVal(req, "verification_status", "verified"), boolInt(boolVal(req, "manual_override", false)),
		tid, now, now, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, h.getModelLocationJSON(id))
}

func (h *AgentHandler) HandleRescanModelLocation(w http.ResponseWriter, r *http.Request) {
	locationID := r.PathValue("location_id")
	now := time.Now().Format(time.RFC3339)
	if _, err := h.DB.Exec(`UPDATE model_locations SET last_scanned_at = ?, last_error = '', updated_at = ? WHERE id = ?`, now, now, locationID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, h.getModelLocationJSON(locationID))
}

func (h *AgentHandler) HandleAttestModelLocation(w http.ResponseWriter, r *http.Request) {
	locationID := r.PathValue("location_id")
	var req map[string]interface{}
	_ = json.NewDecoder(r.Body).Decode(&req)
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`UPDATE model_locations SET manual_override = 1, override_reason = ?, override_by = ?, override_at = ?, match_status = 'manual_attested', verification_status = 'manually_accepted', updated_at = ? WHERE id = ?`,
		strVal(req, "override_reason", ""), userID(r), now, now, locationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, h.getModelLocationJSON(locationID))
}

func (h *AgentHandler) listModelLocations(artifactID string) []map[string]interface{} {
	rows, err := h.DB.Query(`SELECT id, model_artifact_id, node_id, path_type, model_root, relative_path, absolute_path, size_bytes, checksum, manifest_digest, discovered_metadata_json, match_status, verification_status, manual_override, override_reason, override_by, COALESCE(override_at,''), COALESCE(last_scanned_at,''), last_error, tenant_id, created_at, updated_at FROM model_locations WHERE model_artifact_id = ? ORDER BY node_id, absolute_path`, artifactID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		if m := scanModelLocation(rows); m != nil {
			out = append(out, m)
		}
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out
}

func (h *AgentHandler) getModelLocationJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, model_artifact_id, node_id, path_type, model_root, relative_path, absolute_path, size_bytes, checksum, manifest_digest, discovered_metadata_json, match_status, verification_status, manual_override, override_reason, override_by, COALESCE(override_at,''), COALESCE(last_scanned_at,''), last_error, tenant_id, created_at, updated_at FROM model_locations WHERE id = ?`, id)
	return scanModelLocation(row)
}

type modelLocationScanner interface {
	Scan(dest ...interface{}) error
}

func scanModelLocation(row modelLocationScanner) map[string]interface{} {
	var id, aid, nid, pt, root, rel, abs, checksum, manifest, meta, match, verification, reason, by, overrideAt, scannedAt, lastErr, tid, ca, ua string
	var size int64
	var manual int
	if err := row.Scan(&id, &aid, &nid, &pt, &root, &rel, &abs, &size, &checksum, &manifest, &meta, &match, &verification, &manual, &reason, &by, &overrideAt, &scannedAt, &lastErr, &tid, &ca, &ua); err != nil {
		return nil
	}
	return map[string]interface{}{
		"id": id, "model_artifact_id": aid, "node_id": nid, "path_type": pt,
		"model_root": root, "relative_path": rel, "absolute_path": abs,
		"size_bytes": size, "checksum": checksum, "manifest_digest": manifest,
		"discovered_metadata_json": json.RawMessage(meta),
		"match_status":             match, "verification_status": verification,
		"manual_override": manual == 1, "override_reason": reason, "override_by": by,
		"override_at": overrideAt, "last_scanned_at": scannedAt, "last_error": lastErr,
		"tenant_id": tid, "created_at": ca, "updated_at": ua,
	}
}
