package api

import (
	"encoding/json"
	"net/http"
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
	return map[string]interface{}{"id": rid, "name": name, "display_name": dn, "source_type": st, "path": path, "format": frmt, "task_type": tt, "architecture": arch, "size_label": sl, "quantization": quant, "default_context_length": ctxLen, "estimated_vram_bytes": vram, "required_gpu_count": gpuCount, "tenant_id": tid, "created_at": ca, "updated_at": ua}
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
