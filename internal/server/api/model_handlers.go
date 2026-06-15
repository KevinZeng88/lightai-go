package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
	"lightai-go/internal/server/resolver"

	"github.com/google/uuid"
)

// ModelHandler handles all Phase 1 model runtime serving APIs.
type ModelHandler struct {
	DB *db.DB
}

// NewModelHandler creates a new ModelHandler.
func NewModelHandler(database *db.DB) *ModelHandler {
	return &ModelHandler{DB: database}
}

// ==========================================================================
// Generic helpers
// ==========================================================================

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func userID(r *http.Request) string {
	info := auth.SessionInfoFromContext(r.Context())
	if info != nil {
		return info.UserID
	}
	return "system"
}

func tenantID(r *http.Request) string {
	info := auth.SessionInfoFromContext(r.Context())
	if info != nil {
		return info.TenantID
	}
	return ""
}

func isPlatformAdmin(r *http.Request) bool {
	info := auth.SessionInfoFromContext(r.Context())
	return info != nil && info.IsPlatformAdmin
}

func strVal(m map[string]interface{}, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func intVal(m map[string]interface{}, key string, def int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func int64Val(m map[string]interface{}, key string, def int64) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case int:
			return int64(n)
		}
	}
	return def
}

func floatVal(m map[string]interface{}, key string, def float64) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return def
}

func boolVal(m map[string]interface{}, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func strSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		// Handle json.RawMessage (stored as raw JSON bytes).
		if raw, ok := v.(json.RawMessage); ok {
			var arr []string
			if err := json.Unmarshal(raw, &arr); err == nil {
				return arr
			}
			// Try as []interface{} within RawMessage.
			var iarr []interface{}
			if err := json.Unmarshal(raw, &iarr); err == nil {
				out := make([]string, len(iarr))
				for i, e := range iarr {
					out[i] = fmt.Sprint(e)
				}
				return out
			}
			return nil
		}
		// Handle []interface{} (already parsed).
		if arr, ok := v.([]interface{}); ok {
			out := make([]string, len(arr))
			for i, e := range arr {
				out[i] = fmt.Sprint(e)
			}
			return out
		}
		// Handle string (single value).
		if s, ok := v.(string); ok {
			return []string{s}
		}
	}
	return nil
}

func stringMap(m map[string]interface{}, key string) map[string]string {
	if v, ok := m[key]; ok {
		if sm, ok := v.(map[string]interface{}); ok {
			out := make(map[string]string)
			for k, val := range sm {
				out[k] = fmt.Sprint(val)
			}
			return out
		}
	}
	return nil
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func audit(database *db.DB, action, entityType, entityID, detail, operatorUserID string) {
	// Redact sensitive values in audit detail before writing.
	redacted := redactDetailString(detail)
	id := uuid.NewString()
	database.Exec(
		`INSERT INTO audit_logs (id, action, entity_type, entity_id, detail, operator_user_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		id, action, entityType, entityID, redacted, operatorUserID,
	)
}

// redactDetailString replaces sensitive key=value patterns in JSON-like detail strings.
func redactDetailString(s string) string {
	result := s
	for _, sk := range sensitiveKeys() {
		// Match "key":"value" or "key"="value" patterns
		upper := strings.ToUpper(sk)
		lower := strings.ToLower(sk)
		result = strings.ReplaceAll(result, upper, "<redacted>")
		result = strings.ReplaceAll(result, lower, "<redacted>")
	}
	return result
}

func sensitiveKeys() []string {
	return []string{
		"KEY", "TOKEN", "PASSWORD", "PASSWD", "PWD",
		"SECRET", "AUTH", "CREDENTIAL", "ACCESS",
		"API_KEY", "APIKEY", "ACCESS_KEY", "SECRET_KEY",
		"AUTHORIZATION", "BEARER",
		"HF_TOKEN", "DASHSCOPE_API_KEY", "OPENAI_API_KEY",
		"AK", "SK", "PRIVATE",
	}
}

func isSensitive(key string) bool {
	upper := strings.ToUpper(key)
	for _, sk := range sensitiveKeys() {
		if strings.Contains(upper, sk) {
			return true
		}
	}
	return false
}

func redactEnvMap(env map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range env {
		if isSensitive(k) {
			out[k] = "<redacted>"
		} else {
			out[k] = v
		}
	}
	return out
}

func redactStringMap(env map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range env {
		if isSensitive(k) {
			out[k] = "<redacted>"
		} else {
			out[k] = v
		}
	}
	return out
}

func tenantScopeCheck(r *http.Request, resourceTenantID string) bool {
	if isPlatformAdmin(r) {
		return true
	}
	return resourceTenantID == tenantID(r)
}

// ==========================================================================
// ModelArtifact CRUD
// ==========================================================================

func (h *ModelHandler) HandleListModelArtifacts(w http.ResponseWriter, r *http.Request) {
	var rows *sql.Rows
	var err error
	if isPlatformAdmin(r) {
		rows, err = h.DB.Query(modelArtifactCols + ` FROM model_artifacts ORDER BY name`)
	} else {
		rows, err = h.DB.Query(modelArtifactCols+` FROM model_artifacts WHERE tenant_id = ? ORDER BY name`, tenantID(r))
	}
	if err != nil {
		log.Error("list model artifacts", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	writeJSON(w, http.StatusOK, scanModelArtifacts(rows))
}

func (h *ModelHandler) HandleCreateModelArtifact(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	id := uuid.NewString()
	uid := userID(r)
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)

	name := strVal(req, "name", "")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	_, err := h.DB.Exec(
		`INSERT INTO model_artifacts (id, name, display_name, source_type, path, format, task_type,
		 architecture, size_label, quantization, default_context_length, estimated_vram_bytes,
		 required_gpu_count, tenant_id, owner_id, created_by, updated_by, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), strVal(req, "source_type", "local_path"),
		strVal(req, "path", ""), strVal(req, "format", "custom"), strVal(req, "task_type", "chat"),
		strVal(req, "architecture", "custom"), strVal(req, "size_label", ""),
		strVal(req, "quantization", "unknown"), intVal(req, "default_context_length", 0),
		int64Val(req, "estimated_vram_bytes", 0), intVal(req, "required_gpu_count", 1),
		tid, uid, uid, uid, now, now,
	)
	if err != nil {
		log.Error("create model artifact", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	audit(h.DB, "created", "model_artifact", id, `{"name":"`+name+`"}`, uid)
	writeJSON(w, http.StatusCreated, h.getModelArtifact(id))
}

func (h *ModelHandler) HandleGetModelArtifact(w http.ResponseWriter, r *http.Request) {
	m := h.getModelArtifact(r.PathValue("id"))
	if m == nil || !tenantScopeCheck(r, m["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *ModelHandler) HandlePatchModelArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getModelArtifact(id)
	if existing == nil || !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	uid := userID(r)
	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_by = ?", "updated_at = ?"}
	args := []interface{}{uid, now}
	for _, f := range []string{"display_name", "path", "format", "quantization", "architecture", "size_label", "source_type", "task_type"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	if v, ok := req["estimated_vram_bytes"]; ok {
		sets = append(sets, "estimated_vram_bytes = ?")
		args = append(args, v)
	}
	if v, ok := req["required_gpu_count"]; ok {
		sets = append(sets, "required_gpu_count = ?")
		args = append(args, v)
	}
	if v, ok := req["default_context_length"]; ok {
		sets = append(sets, "default_context_length = ?")
		args = append(args, v)
	}
	args = append(args, id)
	h.DB.Exec(`UPDATE model_artifacts SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	audit(h.DB, "updated", "model_artifact", id, `{"patch":true}`, uid)
	writeJSON(w, http.StatusOK, h.getModelArtifact(id))
}

func (h *ModelHandler) HandleDeleteModelArtifact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getModelArtifact(id)
	if existing == nil || !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	uid := userID(r)
	h.DB.Exec(`DELETE FROM model_artifacts WHERE id = ?`, id)
	audit(h.DB, "deleted", "model_artifact", id, `{}`, uid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

const modelArtifactCols = `SELECT id, name, display_name, source_type, path, format, task_type,
 architecture, size_label, quantization, default_context_length, estimated_vram_bytes,
 required_gpu_count, tenant_id, owner_id, created_by, updated_by, created_at, updated_at`

func (h *ModelHandler) getModelArtifact(id string) map[string]interface{} {
	return scanModelArtifactRow(h.DB.QueryRow(modelArtifactCols+` FROM model_artifacts WHERE id = ?`, id))
}

func scanModelArtifacts(rows *sql.Rows) []map[string]interface{} {
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		if m := scanModelArtifact(rows); m != nil {
			out = append(out, m)
		}
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out
}

func scanModelArtifact(scanner interface{ Scan(...interface{}) error }) map[string]interface{} {
	var id, name, dn, st, path, frmt, tt, arch, sl, quant, tid, cb, ub, ca, ua string
	var oid sql.NullString
	var ctxLen, gpuCount int
	var vram int64
	if err := scanner.Scan(&id, &name, &dn, &st, &path, &frmt, &tt, &arch, &sl, &quant,
		&ctxLen, &vram, &gpuCount, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid {
		oidStr = oid.String
	}
	return map[string]interface{}{
		"id": id, "name": name, "display_name": dn, "source_type": st, "path": path,
		"format": frmt, "task_type": tt, "architecture": arch, "size_label": sl,
		"quantization": quant, "default_context_length": ctxLen, "estimated_vram_bytes": vram,
		"required_gpu_count": gpuCount, "tenant_id": tid, "owner_id": oidStr,
		"created_by": cb, "updated_by": ub, "created_at": ca, "updated_at": ua,
	}
}

func scanModelArtifactRow(row *sql.Row) map[string]interface{} {
	var id, name, dn, st, path, frmt, tt, arch, sl, quant, tid, cb, ub, ca, ua string
	var oid sql.NullString
	var ctxLen, gpuCount int
	var vram int64
	if err := row.Scan(&id, &name, &dn, &st, &path, &frmt, &tt, &arch, &sl, &quant,
		&ctxLen, &vram, &gpuCount, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid {
		oidStr = oid.String
	}
	return map[string]interface{}{
		"id": id, "name": name, "display_name": dn, "source_type": st, "path": path,
		"format": frmt, "task_type": tt, "architecture": arch, "size_label": sl,
		"quantization": quant, "default_context_length": ctxLen, "estimated_vram_bytes": vram,
		"required_gpu_count": gpuCount, "tenant_id": tid, "owner_id": oidStr,
		"created_by": cb, "updated_by": ub, "created_at": ca, "updated_at": ua,
	}
}

// ==========================================================================
// RuntimeEnvironment CRUD
// ==========================================================================

func (h *ModelHandler) HandleListRuntimeEnvironments(w http.ResponseWriter, r *http.Request) {
	var rows *sql.Rows
	var err error
	if isPlatformAdmin(r) {
		rows, err = h.DB.Query(reCols + ` FROM runtime_environments ORDER BY name`)
	} else {
		rows, err = h.DB.Query(reCols+` FROM runtime_environments WHERE tenant_id IS NULL OR tenant_id = ? ORDER BY name`, tenantID(r))
	}
	if err != nil {
		log.Error("list runtime environments", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	writeJSON(w, http.StatusOK, scanRuntimeEnvironments(rows, h.DB))
}

func (h *ModelHandler) HandleCreateRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	// operator cannot create global or high-risk runtime envs
	perms := auth.PermissionsFromContext(r.Context())
	isOperator := !isPlatformAdmin(r) && !hasPerm(perms, "membership:write") && hasPerm(perms, "runtime:write")
	_ = isOperator // reserved for future high-risk checks

	id := uuid.NewString()
	uid := userID(r)
	tid := tenantID(r)
	// global if platform_admin and no tenant_id specified, or if explicitly requested
	global := isPlatformAdmin(r) && (strVal(req, "tenant_id", "") == "" || strVal(req, "tenant_id", "") == "*")
	var reTenantID string
	if global {
		reTenantID = ""
	} else {
		reTenantID = tid
	}
	now := time.Now().Format(time.RFC3339)

	name := strVal(req, "name", "")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	runtimeType := strVal(req, "runtime_type", "docker")
	_, err := h.DB.Exec(
		`INSERT INTO runtime_environments (id, name, display_name, runtime_type, backend_type, vendor,
		 openai_compatible, default_port, health_check_path, description,
		 tenant_id, owner_id, created_by, updated_by, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), runtimeType,
		strVal(req, "backend_type", "custom"), strVal(req, "vendor", "custom"),
		boolToInt(boolVal(req, "openai_compatible", false)),
		intVal(req, "default_port", 8000), strVal(req, "health_check_path", "/health"),
		strVal(req, "description", ""), reTenantID, uid, uid, uid, now, now,
	)
	if err != nil {
		log.Error("create runtime environment", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create docker spec if runtime_type=docker and docker data present.
	if runtimeType == "docker" {
		docker, ok := req["docker"]
		if ok {
			h.createOrUpdateDockerSpec(id, docker, uid, now)
		}
	}

	audit(h.DB, "created", "runtime_environment", id, `{"name":"`+name+`"}`, uid)
	writeJSON(w, http.StatusCreated, h.getRuntimeEnvironment(id))
}

func (h *ModelHandler) HandleGetRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	re := h.getRuntimeEnvironment(r.PathValue("id"))
	if re == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	reTenantID, _ := re["tenant_id"].(string)
	if reTenantID != "" && !tenantScopeCheck(r, reTenantID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	// global (tenant_id="") is visible to all.
	writeJSON(w, http.StatusOK, re)
}

func (h *ModelHandler) HandlePatchRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getRuntimeEnvironment(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	// tenant scope: global envs editable only by platform_admin; tenant envs editable by same tenant
	reTenantID, _ := existing["tenant_id"].(string)
	if reTenantID == "" && !isPlatformAdmin(r) {
		writeError(w, http.StatusForbidden, "only platform_admin can modify global runtime environments")
		return
	}
	if reTenantID != "" && !tenantScopeCheck(r, reTenantID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	uid := userID(r)
	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_by = ?", "updated_at = ?"}
	args := []interface{}{uid, now}
	for _, f := range []string{"display_name", "description", "health_check_path"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	if v, ok := req["default_port"]; ok {
		sets = append(sets, "default_port = ?")
		args = append(args, v)
	}
	args = append(args, id)
	h.DB.Exec(`UPDATE runtime_environments SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)

	// Update docker spec if provided.
	if docker, ok := req["docker"]; ok {
		h.createOrUpdateDockerSpec(id, docker, uid, now)
	}

	audit(h.DB, "updated", "runtime_environment", id, `{}`, uid)
	writeJSON(w, http.StatusOK, h.getRuntimeEnvironment(id))
}

func (h *ModelHandler) HandleDeleteRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getRuntimeEnvironment(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	reTenantID, _ := existing["tenant_id"].(string)
	if reTenantID == "" && !isPlatformAdmin(r) {
		writeError(w, http.StatusForbidden, "only platform_admin can delete global runtime environments")
		return
	}
	if reTenantID != "" && !tenantScopeCheck(r, reTenantID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	uid := userID(r)
	h.DB.Exec(`DELETE FROM runtime_environment_docker_specs WHERE runtime_environment_id = ?`, id)
	h.DB.Exec(`DELETE FROM runtime_environments WHERE id = ?`, id)
	audit(h.DB, "deleted", "runtime_environment", id, `{}`, uid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ModelHandler) createOrUpdateDockerSpec(reID string, dockerData interface{}, uid, now string) {
	dm, ok := dockerData.(map[string]interface{})
	if !ok {
		return
	}
	// Upsert.
	var existingID string
	h.DB.QueryRow(`SELECT id FROM runtime_environment_docker_specs WHERE runtime_environment_id = ?`, reID).Scan(&existingID)
	if existingID != "" {
		h.DB.Exec(`DELETE FROM runtime_environment_docker_specs WHERE id = ?`, existingID)
	}
	specID := uuid.NewString()
	h.DB.Exec(
		`INSERT INTO runtime_environment_docker_specs (id, runtime_environment_id, image, image_pull_policy,
		 devices, privileged, ipc_mode, uts_mode, network_mode, shm_size, group_add,
		 security_options, ulimits, restart_policy, gpu_visible_env_key, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		specID, reID,
		strVal(dm, "image", ""),
		strVal(dm, "image_pull_policy", "never"),
		jsonString(dm["devices"]),
		jsonString(dm["privileged"]),
		jsonString(dm["ipc_mode"]),
		jsonString(dm["uts_mode"]),
		jsonString(dm["network_mode"]),
		jsonString(dm["shm_size"]),
		jsonString(dm["group_add"]),
		jsonString(dm["security_options"]),
		jsonString(dm["ulimits"]),
		jsonString(dm["restart_policy"]),
		strVal(dm, "gpu_visible_env_key", "CUDA_VISIBLE_DEVICES"),
		now, now,
	)
}

const reCols = `SELECT id, name, display_name, runtime_type, backend_type, vendor,
 openai_compatible, default_port, health_check_path, description,
 tenant_id, owner_id, created_by, updated_by, created_at, updated_at`

func (h *ModelHandler) getRuntimeEnvironment(id string) map[string]interface{} {
	re := scanReRow(h.DB.QueryRow(reCols+` FROM runtime_environments WHERE id = ?`, id))
	if re == nil {
		return nil
	}
	// Attach docker spec if exists.
	var specID, image, pullPolicy, devices, priv, ipc, uts, net, shm, ga, sec, ul, restart, gpuEnv, ca, ua string
	err := h.DB.QueryRow(`SELECT id, image, image_pull_policy, devices, privileged, ipc_mode, uts_mode,
		network_mode, shm_size, group_add, security_options, ulimits, restart_policy,
		gpu_visible_env_key, created_at, updated_at
		FROM runtime_environment_docker_specs WHERE runtime_environment_id = ?`, id).
		Scan(&specID, &image, &pullPolicy, &devices, &priv, &ipc, &uts, &net, &shm, &ga, &sec, &ul, &restart, &gpuEnv, &ca, &ua)
	if err == nil {
		re["docker"] = map[string]interface{}{
			"id": specID, "image": redactImageIfSensitive(image), "image_pull_policy": pullPolicy,
			"devices": json.RawMessage(devices), "privileged": json.RawMessage(priv),
			"ipc_mode": json.RawMessage(ipc), "uts_mode": json.RawMessage(uts),
			"network_mode": json.RawMessage(net), "shm_size": json.RawMessage(shm),
			"group_add": json.RawMessage(ga), "security_options": json.RawMessage(sec),
			"ulimits": json.RawMessage(ul), "restart_policy": json.RawMessage(restart),
			"gpu_visible_env_key": gpuEnv, "created_at": ca, "updated_at": ua,
		}
	}
	return re
}

func redactImageIfSensitive(image string) string {
	if isSensitive(image) {
		return "<redacted>"
	}
	return image
}

func scanRuntimeEnvironments(rows *sql.Rows, database *db.DB) []map[string]interface{} {
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		re := scanRe(rows)
		if re != nil {
			id := re["id"].(string)
			// Attach docker spec.
			var specID, image, pullPolicy, devices, priv, ipc, uts, net, shm, ga, sec, ul, restart, gpuEnv, ca, ua string
			err := database.QueryRow(`SELECT id, image, image_pull_policy, devices, privileged, ipc_mode, uts_mode,
				network_mode, shm_size, group_add, security_options, ulimits, restart_policy,
				gpu_visible_env_key, created_at, updated_at
				FROM runtime_environment_docker_specs WHERE runtime_environment_id = ?`, id).
				Scan(&specID, &image, &pullPolicy, &devices, &priv, &ipc, &uts, &net, &shm, &ga, &sec, &ul, &restart, &gpuEnv, &ca, &ua)
			if err == nil {
				re["docker"] = map[string]interface{}{
					"id": specID, "image": image, "image_pull_policy": pullPolicy,
					"devices": json.RawMessage(devices), "privileged": json.RawMessage(priv),
					"ipc_mode": json.RawMessage(ipc), "uts_mode": json.RawMessage(uts),
					"network_mode": json.RawMessage(net), "shm_size": json.RawMessage(shm),
					"group_add": json.RawMessage(ga), "security_options": json.RawMessage(sec),
					"ulimits": json.RawMessage(ul), "restart_policy": json.RawMessage(restart),
					"gpu_visible_env_key": gpuEnv, "created_at": ca, "updated_at": ua,
				}
			}
			out = append(out, re)
		}
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out
}

func scanRe(scanner interface{ Scan(...interface{}) error }) map[string]interface{} {
	var id, name, dn, rt, bt, vendor, desc, hcp, tid, cb, ub, ca, ua string
	var oid sql.NullString
	var oai int
	var dp int
	if err := scanner.Scan(&id, &name, &dn, &rt, &bt, &vendor, &oai, &dp, &hcp, &desc, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid {
		oidStr = oid.String
	}
	return map[string]interface{}{
		"id": id, "name": name, "display_name": dn, "runtime_type": rt, "backend_type": bt,
		"vendor": vendor, "openai_compatible": oai == 1, "default_port": dp,
		"health_check_path": hcp, "description": desc, "tenant_id": tid, "owner_id": oidStr,
		"created_by": cb, "updated_by": ub, "created_at": ca, "updated_at": ua,
	}
}

func scanReRow(row *sql.Row) map[string]interface{} {
	var id, name, dn, rt, bt, vendor, desc, hcp, tid, cb, ub, ca, ua string
	var oid sql.NullString
	var oai int
	var dp int
	if err := row.Scan(&id, &name, &dn, &rt, &bt, &vendor, &oai, &dp, &hcp, &desc, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid {
		oidStr = oid.String
	}
	return map[string]interface{}{
		"id": id, "name": name, "display_name": dn, "runtime_type": rt, "backend_type": bt,
		"vendor": vendor, "openai_compatible": oai == 1, "default_port": dp,
		"health_check_path": hcp, "description": desc, "tenant_id": tid, "owner_id": oidStr,
		"created_by": cb, "updated_by": ub, "created_at": ca, "updated_at": ua,
	}
}

// ==========================================================================
// RunTemplate CRUD + render-preview
// ==========================================================================

func (h *ModelHandler) HandleListRunTemplates(w http.ResponseWriter, r *http.Request) {
	var rows *sql.Rows
	var err error
	if isPlatformAdmin(r) {
		rows, err = h.DB.Query(rtCols + ` FROM run_templates ORDER BY name`)
	} else {
		rows, err = h.DB.Query(rtCols+` FROM run_templates WHERE tenant_id IS NULL OR tenant_id = ? ORDER BY name`, tenantID(r))
	}
	if err != nil {
		log.Error("list run templates", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, scanRunTemplates(rows))
}

func (h *ModelHandler) HandleCreateRunTemplate(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	id := uuid.NewString()
	uid := userID(r)
	tid := tenantID(r)
	global := isPlatformAdmin(r)
	var rtTenantID string
	if global {
		rtTenantID = ""
	} else {
		rtTenantID = tid
	}
	now := time.Now().Format(time.RFC3339)
	name := strVal(req, "name", "")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	reqVars := strSlice(req, "required_variables")
	optVars := strSlice(req, "optional_variables")
	argsTmpl := strSlice(req, "args_template")
	if reqVars == nil {
		reqVars = []string{}
	}
	if optVars == nil {
		optVars = []string{}
	}
	if argsTmpl == nil {
		argsTmpl = []string{}
	}

	_, err := h.DB.Exec(
		`INSERT INTO run_templates (id, name, display_name, runtime_type, vendor, backend_type,
		 required_variables, optional_variables, env_mappings, args_template,
		 volume_mappings, port_mappings, backend_flags, description,
		 tenant_id, owner_id, created_by, updated_by, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), strVal(req, "runtime_type", "docker"),
		strVal(req, "vendor", "custom"), strVal(req, "backend_type", "custom"),
		jsonString(reqVars), jsonString(optVars),
		jsonString(req["env_mappings"]), jsonString(argsTmpl),
		jsonString(req["volume_mappings"]), jsonString(req["port_mappings"]),
		jsonString(req["backend_flags"]), strVal(req, "description", ""),
		rtTenantID, uid, uid, uid, now, now,
	)
	if err != nil {
		log.Error("create run template", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	audit(h.DB, "created", "run_template", id, `{"name":"`+name+`"}`, uid)
	writeJSON(w, http.StatusCreated, h.getRunTemplate(id))
}

func (h *ModelHandler) HandleGetRunTemplate(w http.ResponseWriter, r *http.Request) {
	rt := h.getRunTemplate(r.PathValue("id"))
	if rt == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rtTenantID, _ := rt["tenant_id"].(string)
	if rtTenantID != "" && !tenantScopeCheck(r, rtTenantID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *ModelHandler) HandlePatchRunTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getRunTemplate(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rtTenantID, _ := existing["tenant_id"].(string)
	if rtTenantID == "" && !isPlatformAdmin(r) {
		writeError(w, http.StatusForbidden, "only platform_admin can modify global run templates")
		return
	}
	if rtTenantID != "" && !tenantScopeCheck(r, rtTenantID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	uid := userID(r)
	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_by = ?", "updated_at = ?"}
	args := []interface{}{uid, now}
	for _, f := range []string{"display_name", "description"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	if v, ok := req["args_template"]; ok {
		sets = append(sets, "args_template = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["env_mappings"]; ok {
		sets = append(sets, "env_mappings = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["required_variables"]; ok {
		sets = append(sets, "required_variables = ?")
		args = append(args, jsonString(v))
	}
	args = append(args, id)
	h.DB.Exec(`UPDATE run_templates SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	audit(h.DB, "updated", "run_template", id, `{}`, uid)
	writeJSON(w, http.StatusOK, h.getRunTemplate(id))
}

func (h *ModelHandler) HandleDeleteRunTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getRunTemplate(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rtTenantID, _ := existing["tenant_id"].(string)
	if rtTenantID == "" && !isPlatformAdmin(r) {
		writeError(w, http.StatusForbidden, "only platform_admin can delete global run templates")
		return
	}
	if rtTenantID != "" && !tenantScopeCheck(r, rtTenantID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	uid := userID(r)
	h.DB.Exec(`DELETE FROM run_templates WHERE id = ?`, id)
	audit(h.DB, "deleted", "run_template", id, `{}`, uid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// HandleRenderPreview handles POST /api/run-templates/{id}/render-preview.
func (h *ModelHandler) HandleRenderPreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rt := h.getRunTemplate(id)
	if rt == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Build a minimal resolve input from request + template.
	artifact := &resolver.ModelArtifactInput{Path: strVal(req, "model_path", "/data/models/example")}
	env := &resolver.EnvironmentInput{
		RuntimeType: strVal(rt, "runtime_type", "docker"),
		BackendType: strVal(rt, "backend_type", "custom"),
		Vendor:      strVal(rt, "vendor", "custom"),
		DefaultPort: 8000,
	}
	deploy := &resolver.DeploymentInput{
		ServedModelName:      strVal(req, "served_model_name", "model"),
		NodeID:               strVal(req, "node_id", "node-1"),
		GPUIds:               strSlice(req, "gpu_ids"),
		HostPort:             intVal(req, "host_port", 8001),
		MaxModelLen:          intVal(req, "max_model_len", 4096),
		GPUMemoryUtilization: floatVal(req, "gpu_memory_utilization", 0.9),
	}
	agentID := strVal(req, "agent_id", "agent-01")

	// Parse template args.
	argsTmpl := strSlice(rt, "args_template")
	if argsTmpl == nil {
		argsTmpl = []string{}
	}
	reqVars := strSlice(rt, "required_variables")
	if reqVars == nil {
		reqVars = []string{}
	}

	spec, errs, warns := resolver.Resolve(resolver.ResolveInput{
		Artifact:  artifact,
		Env:       env,
		Deployment: deploy,
		ArgsTemplate: argsTmpl,
		RequiredVars: reqVars,
		AgentID:   agentID,
		InstanceID: "preview-instance",
	})

	cmdPreview := resolver.EquivalentCommandPreview(spec)

	// Redact sensitive in spec.
	specMap := specToMap(spec)
	redactResolvedSpec(specMap)

	resp := map[string]interface{}{
		"valid":                      len(errs) == 0,
		"resolved_run_spec":          specMap,
		"equivalent_command_preview": redactCommandPreview(cmdPreview),
	}
	if len(errs) > 0 {
		errMsgs := make([]string, len(errs))
		for i, e := range errs {
			errMsgs[i] = e.Error()
		}
		resp["errors"] = errMsgs
	} else {
		resp["errors"] = []string{}
	}
	if len(warns) > 0 {
		resp["warnings"] = warns
	} else {
		resp["warnings"] = []string{}
	}

	uid := userID(r)
	audit(h.DB, "render_preview", "run_template", id, `{}`, uid)
	writeJSON(w, http.StatusOK, resp)
}

func redactResolvedSpec(spec map[string]interface{}) {
	if env, ok := spec["env"].(map[string]interface{}); ok {
		spec["env"] = redactEnvMap(env)
	}
}

func redactCommandPreview(cmd string) string {
	for _, sk := range sensitiveKeys() {
		lower := strings.ToLower(sk)
		cmd = strings.ReplaceAll(cmd, lower, "<redacted>")
		upper := strings.ToUpper(sk)
		cmd = strings.ReplaceAll(cmd, upper, "<redacted>")
	}
	return cmd
}

func specToMap(spec *resolver.ResolvedRunSpec) map[string]interface{} {
	env := make(map[string]interface{})
	for k, v := range spec.Env {
		env[k] = v
	}
	volumes := make([]interface{}, len(spec.Volumes))
	for i, v := range spec.Volumes {
		volumes[i] = map[string]interface{}{"host_path": v.HostPath, "container_path": v.ContainerPath, "readonly": v.Readonly}
	}
	ports := make([]interface{}, len(spec.Ports))
	for i, p := range spec.Ports {
		ports[i] = map[string]interface{}{"host_port": p.HostPort, "container_port": p.ContainerPort, "protocol": p.Protocol}
	}
	return map[string]interface{}{
		"instance_id": spec.InstanceID, "deployment_id": spec.DeploymentID,
		"runtime_type": spec.RuntimeType, "backend_type": spec.BackendType, "vendor": spec.Vendor,
		"model_path": spec.ModelPath, "served_model_name": spec.ServedModelName,
		"node_id": spec.NodeID, "agent_id": spec.AgentID, "gpu_device_ids": spec.GPUDeviceIDs,
		"gpu_visible_env_key": spec.GPUVisibleEnvKey, "env": env, "args": spec.Args,
		"host_port": spec.HostPort, "container_port": spec.ContainerPort,
		"volumes": volumes, "ports": ports,
		"docker": map[string]interface{}{
			"image": spec.Docker.Image, "container_name": spec.Docker.ContainerName,
			"args": spec.Docker.Args, "privileged": spec.Docker.Privileged,
			"ipc_mode": spec.Docker.IPCMode, "shm_size": spec.Docker.ShmSize,
			"gpu_device_ids": spec.Docker.GPUDeviceIDs,
		},
	}
}

const rtCols = `SELECT id, name, display_name, runtime_type, vendor, backend_type,
 required_variables, optional_variables, env_mappings, args_template,
 volume_mappings, port_mappings, backend_flags, description,
 tenant_id, owner_id, created_by, updated_by, created_at, updated_at`

func (h *ModelHandler) getRunTemplate(id string) map[string]interface{} {
	return scanRtRow(h.DB.QueryRow(rtCols+` FROM run_templates WHERE id = ?`, id))
}

func scanRunTemplates(rows *sql.Rows) []map[string]interface{} {
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		if m := scanRt(rows); m != nil {
			out = append(out, m)
		}
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out
}

func scanRt(scanner interface{ Scan(...interface{}) error }) map[string]interface{} {
	var id, name, dn, rt, vendor, bt, reqV, optV, envM, argsT, volM, portM, backF, desc, tid, cb, ub, ca, ua string
	var oid sql.NullString
	if err := scanner.Scan(&id, &name, &dn, &rt, &vendor, &bt, &reqV, &optV, &envM, &argsT, &volM, &portM, &backF, &desc, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid {
		oidStr = oid.String
	}
	return map[string]interface{}{
		"id": id, "name": name, "display_name": dn, "runtime_type": rt, "vendor": vendor, "backend_type": bt,
		"required_variables": json.RawMessage(reqV), "optional_variables": json.RawMessage(optV),
		"env_mappings": json.RawMessage(envM), "args_template": json.RawMessage(argsT),
		"volume_mappings": json.RawMessage(volM), "port_mappings": json.RawMessage(portM),
		"backend_flags": json.RawMessage(backF), "description": desc,
		"tenant_id": tid, "owner_id": oidStr, "created_by": cb, "updated_by": ub,
		"created_at": ca, "updated_at": ua,
	}
}

func scanRtRow(row *sql.Row) map[string]interface{} {
	var id, name, dn, rt, vendor, bt, reqV, optV, envM, argsT, volM, portM, backF, desc, tid, cb, ub, ca, ua string
	var oid sql.NullString
	if err := row.Scan(&id, &name, &dn, &rt, &vendor, &bt, &reqV, &optV, &envM, &argsT, &volM, &portM, &backF, &desc, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid {
		oidStr = oid.String
	}
	return map[string]interface{}{
		"id": id, "name": name, "display_name": dn, "runtime_type": rt, "vendor": vendor, "backend_type": bt,
		"required_variables": json.RawMessage(reqV), "optional_variables": json.RawMessage(optV),
		"env_mappings": json.RawMessage(envM), "args_template": json.RawMessage(argsT),
		"volume_mappings": json.RawMessage(volM), "port_mappings": json.RawMessage(portM),
		"backend_flags": json.RawMessage(backF), "description": desc,
		"tenant_id": tid, "owner_id": oidStr, "created_by": cb, "updated_by": ub,
		"created_at": ca, "updated_at": ua,
	}
}

// ==========================================================================
// ModelDeployment CRUD + dry-run
// ==========================================================================

func (h *ModelHandler) HandleListModelDeployments(w http.ResponseWriter, r *http.Request) {
	var rows *sql.Rows
	var err error
	if isPlatformAdmin(r) {
		rows, err = h.DB.Query(mdCols + ` FROM model_deployments ORDER BY name`)
	} else {
		rows, err = h.DB.Query(mdCols+` FROM model_deployments WHERE tenant_id = ? ORDER BY name`, tenantID(r))
	}
	if err != nil {
		log.Error("list model deployments", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, scanModelDeployments(rows))
}

func (h *ModelHandler) HandleCreateModelDeployment(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	id := uuid.NewString()
	uid := userID(r)
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)
	name := strVal(req, "name", "")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	scheduleMode := strVal(req, "schedule_mode", "manual")
	if scheduleMode != "manual" {
		writeError(w, http.StatusBadRequest, "schedule_mode must be manual in Phase 1")
		return
	}
	replicas := intVal(req, "replicas", 1)
	if replicas != 1 {
		writeError(w, http.StatusBadRequest, "replicas must be 1 in Phase 1")
		return
	}
	_, err := h.DB.Exec(
		`INSERT INTO model_deployments (id, name, display_name, model_artifact_id, runtime_environment_id,
		 run_template_id, replicas, desired_state, status, node_id, gpu_ids, host_port,
		 served_model_name, max_model_len, tensor_parallel_size, gpu_memory_utilization, dtype,
		 gpu_visible_env_key, env_overrides, arg_overrides, extra_args,
		 schedule_mode, placement_strategy, expose_mode, service_path,
		 tenant_id, owner_id, created_by, updated_by, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name),
		strVal(req, "model_artifact_id", ""), strVal(req, "runtime_environment_id", ""),
		strVal(req, "run_template_id", ""), replicas, "stopped", "stopped",
		strVal(req, "node_id", ""), jsonString(req["gpu_ids"]), intVal(req, "host_port", 0),
		strVal(req, "served_model_name", ""), intVal(req, "max_model_len", 0),
		intVal(req, "tensor_parallel_size", 1), floatVal(req, "gpu_memory_utilization", 0.9),
		strVal(req, "dtype", "auto"), strVal(req, "gpu_visible_env_key", ""),
		jsonString(req["env_overrides"]), jsonString(req["arg_overrides"]),
		jsonString(req["extra_args"]), scheduleMode, strVal(req, "placement_strategy", "manual"),
		strVal(req, "expose_mode", "direct"), strVal(req, "service_path", ""),
		tid, uid, uid, uid, now, now,
	)
	if err != nil {
		log.Error("create model deployment", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	audit(h.DB, "created", "model_deployment", id, `{"name":"`+name+`"}`, uid)
	writeJSON(w, http.StatusCreated, h.getModelDeployment(id))
}

func (h *ModelHandler) HandleGetModelDeployment(w http.ResponseWriter, r *http.Request) {
	md := h.getModelDeployment(r.PathValue("id"))
	if md == nil || !tenantScopeCheck(r, md["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, md)
}

func (h *ModelHandler) HandlePatchModelDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getModelDeployment(id)
	if existing == nil || !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	uid := userID(r)
	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_by = ?", "updated_at = ?"}
	args := []interface{}{uid, now}
	for _, f := range []string{"display_name", "served_model_name", "dtype", "node_id", "gpu_visible_env_key"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	if v, ok := req["host_port"]; ok {
		sets = append(sets, "host_port = ?")
		args = append(args, v)
	}
	if v, ok := req["max_model_len"]; ok {
		sets = append(sets, "max_model_len = ?")
		args = append(args, v)
	}
	if v, ok := req["gpu_ids"]; ok {
		sets = append(sets, "gpu_ids = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["env_overrides"]; ok {
		sets = append(sets, "env_overrides = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["arg_overrides"]; ok {
		sets = append(sets, "arg_overrides = ?")
		args = append(args, jsonString(v))
	}
	args = append(args, id)
	h.DB.Exec(`UPDATE model_deployments SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	audit(h.DB, "updated", "model_deployment", id, `{}`, uid)
	writeJSON(w, http.StatusOK, h.getModelDeployment(id))
}

func (h *ModelHandler) HandleDeleteModelDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getModelDeployment(id)
	if existing == nil || !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	uid := userID(r)
	h.DB.Exec(`DELETE FROM model_deployments WHERE id = ?`, id)
	audit(h.DB, "deleted", "model_deployment", id, `{}`, uid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// HandleDryRun handles POST /api/model-deployments/{id}/dry-run.
func (h *ModelHandler) HandleDryRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	md := h.getModelDeployment(id)
	if md == nil || !tenantScopeCheck(r, md["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	uid := userID(r)
	nodeID := strVal(req, "node_id", strVal(md, "node_id", ""))
	gpuIDs := strSlice(req, "gpu_ids")
	if gpuIDs == nil {
		gpuIDs = strSlice(md, "gpu_ids")
	}
	hostPort := intVal(req, "host_port", 0)
	if hostPort == 0 {
		hostPort = intVal(md, "host_port", 0)
	}
	modelArtifactID := strVal(md, "model_artifact_id", "")
	vendor := ""
	envID := strVal(md, "runtime_environment_id", "")
	if envID != "" {
		h.DB.QueryRow(`SELECT vendor FROM runtime_environments WHERE id = ?`, envID).Scan(&vendor)
	}
	// Fetch model path for validation.
	var modelPath string
	if modelArtifactID != "" {
		h.DB.QueryRow(`SELECT path FROM model_artifacts WHERE id = ?`, modelArtifactID).Scan(&modelPath)
	}
	templateID := strVal(md, "run_template_id", "")
	var reqVarsStr string
	if templateID != "" {
		h.DB.QueryRow(`SELECT required_variables FROM run_templates WHERE id = ?`, templateID).Scan(&reqVarsStr)
	}
	var requiredVars []string
	json.Unmarshal([]byte(reqVarsStr), &requiredVars)

	// Delegate validation to the resolver package.
	dryRunInput := resolver.DryRunInput{
		NodeID:              nodeID,
		GPUIds:              gpuIDs,
		HostPort:            hostPort,
		RuntimeVendor:       vendor,
		ModelArtifactID:     modelArtifactID,
		ModelPath:           modelPath,
		TemplateRequiredVars: requiredVars,
	}
	result := resolver.ValidateDryRun(h.DB.DB, dryRunInput)

	// VRAM warning.
	modelVRAM := int64(0)
	if modelArtifactID != "" {
		h.DB.QueryRow(`SELECT estimated_vram_bytes FROM model_artifacts WHERE id = ?`, modelArtifactID).Scan(&modelVRAM)
	}
	for _, gpuID := range gpuIDs {
		var freeMem int64
		if err := h.DB.QueryRow(`SELECT memory_free_bytes FROM gpu_devices WHERE id = ?`, gpuID).Scan(&freeMem); err == nil {
			if modelVRAM > 0 && modelVRAM > freeMem {
				result.Warnings = append(result.Warnings, fmt.Sprintf("GPU %s free memory %d bytes < estimated %d bytes", gpuID, freeMem, modelVRAM))
			}
		}
	}

	// Generate ResolvedRunSpec if no errors.
	var resolvedSpec map[string]interface{}
	var cmdPreview string
	if result.Valid {
			resolveIn := buildResolveInputForDeployment(h.DB, resolveDeploymentInput{
				ArtifactID:  modelArtifactID,
				EnvID:       envID,
				TemplateID:  templateID,
				DeployID:    id,
				NodeID:      nodeID,
				GPUIds:      gpuIDs,
				HostPort:    hostPort,
				ModelPath:   modelPath,
				Vendor:      vendor,
				RuntimeType: "docker",
				BackendType: "custom",
				DefaultPort: 8000,
				ServedModelName:      strVal(md, "served_model_name", ""),
				MaxModelLen:          intVal(md, "max_model_len", 0),
				GPUMemoryUtilization: floatVal(md, "gpu_memory_utilization", 0.9),
			})
			resolveIn.AgentID = "dry-run-agent"
			resolveIn.InstanceID = "dry-run-instance"
			spec, _, _ := resolver.Resolve(resolveIn)
		resolvedSpec = specToMap(spec)
		redactResolvedSpec(resolvedSpec)
		cmdPreview = redactCommandPreview(resolver.EquivalentCommandPreview(spec))
	}

	resp := map[string]interface{}{
		"valid":   result.Valid,
		"errors":  result.Errors,
		"warnings": result.Warnings,
	}
	if resolvedSpec != nil {
		resp["resolved_run_spec"] = resolvedSpec
	}
	if cmdPreview != "" {
		resp["equivalent_command_preview"] = cmdPreview
	}
	if resp["errors"] == nil {
		resp["errors"] = []string{}
	}
	if resp["warnings"] == nil {
		resp["warnings"] = []string{}
	}

	audit(h.DB, "dry_run", "model_deployment", id, `{"valid":`+fmt.Sprintf("%v", result.Valid)+`}`, uid)
	writeJSON(w, http.StatusOK, resp)
}

const mdCols = `SELECT id, name, display_name, model_artifact_id, runtime_environment_id,
 run_template_id, replicas, desired_state, status, node_id, gpu_ids, host_port,
 served_model_name, max_model_len, tensor_parallel_size, gpu_memory_utilization, dtype,
 gpu_visible_env_key, env_overrides, arg_overrides, extra_args,
 schedule_mode, placement_strategy, expose_mode, service_path,
 tenant_id, owner_id, created_by, updated_by, created_at, updated_at`

func (h *ModelHandler) getModelDeployment(id string) map[string]interface{} {
	return scanMdRow(h.DB.QueryRow(mdCols+` FROM model_deployments WHERE id = ?`, id))
}

func scanModelDeployments(rows *sql.Rows) []map[string]interface{} {
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		if m := scanMd(rows); m != nil {
			out = append(out, m)
		}
	}
	if out == nil { out = []map[string]interface{}{} }
	return out
}

func scanMd(scanner interface{ Scan(...interface{}) error }) map[string]interface{} {
	var id, name, dn, maid, reid, rtid, ds, st, nid, gids, smn, dtype, gvek, eo, ao, ea, sm, ps, em, sp, tid, cb, ub, ca, ua string
	var oid sql.NullString
	var replicas, hp, ml, tps int
	var gmu float64
	if err := scanner.Scan(&id, &name, &dn, &maid, &reid, &rtid, &replicas, &ds, &st, &nid, &gids, &hp, &smn, &ml, &tps, &gmu, &dtype, &gvek, &eo, &ao, &ea, &sm, &ps, &em, &sp, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid { oidStr = oid.String }
	return map[string]interface{}{
		"id":id,"name":name,"display_name":dn,"model_artifact_id":maid,"runtime_environment_id":reid,"run_template_id":rtid,
		"replicas":replicas,"desired_state":ds,"status":st,"node_id":nid,"gpu_ids":json.RawMessage(gids),"host_port":hp,
		"served_model_name":smn,"max_model_len":ml,"tensor_parallel_size":tps,"gpu_memory_utilization":gmu,"dtype":dtype,
		"gpu_visible_env_key":gvek,"env_overrides":json.RawMessage(eo),"arg_overrides":json.RawMessage(ao),"extra_args":json.RawMessage(ea),
		"schedule_mode":sm,"placement_strategy":ps,"expose_mode":em,"service_path":sp,
		"tenant_id":tid,"owner_id":oidStr,"created_by":cb,"updated_by":ub,"created_at":ca,"updated_at":ua,
	}
}

func scanMdRow(row *sql.Row) map[string]interface{} {
	var id, name, dn, maid, reid, rtid, ds, st, nid, gids, smn, dtype, gvek, eo, ao, ea, sm, ps, em, sp, tid, cb, ub, ca, ua string
	var oid sql.NullString
	var replicas, hp, ml, tps int
	var gmu float64
	if err := row.Scan(&id, &name, &dn, &maid, &reid, &rtid, &replicas, &ds, &st, &nid, &gids, &hp, &smn, &ml, &tps, &gmu, &dtype, &gvek, &eo, &ao, &ea, &sm, &ps, &em, &sp, &tid, &oid, &cb, &ub, &ca, &ua); err != nil {
		return nil
	}
	oidStr := ""
	if oid.Valid { oidStr = oid.String }
	return map[string]interface{}{
		"id":id,"name":name,"display_name":dn,"model_artifact_id":maid,"runtime_environment_id":reid,"run_template_id":rtid,
		"replicas":replicas,"desired_state":ds,"status":st,"node_id":nid,"gpu_ids":json.RawMessage(gids),"host_port":hp,
		"served_model_name":smn,"max_model_len":ml,"tensor_parallel_size":tps,"gpu_memory_utilization":gmu,"dtype":dtype,
		"gpu_visible_env_key":gvek,"env_overrides":json.RawMessage(eo),"arg_overrides":json.RawMessage(ao),"extra_args":json.RawMessage(ea),
		"schedule_mode":sm,"placement_strategy":ps,"expose_mode":em,"service_path":sp,
		"tenant_id":tid,"owner_id":oidStr,"created_by":cb,"updated_by":ub,"created_at":ca,"updated_at":ua,
	}
}

// ==========================================================================
// ModelInstance read-only
// ==========================================================================

func (h *ModelHandler) HandleListModelInstances(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.URL.Query().Get("deployment_id")
	var rows *sql.Rows
	var err error
	cols := `SELECT id, deployment_id, replica_index, node_id, agent_id, runtime_type, gpu_ids, gpu_lease_ids,
	 desired_state, actual_state, container_id, process_id, remote_url, endpoint_url, host_port, container_port,
	 restart_count, last_error, last_exit_code, resolved_run_spec,
	 started_at, stopped_at, last_heartbeat_at, created_at, updated_at`
	if isPlatformAdmin(r) {
		if deploymentID != "" {
			rows, err = h.DB.Query(cols+` FROM model_instances WHERE deployment_id = ? ORDER BY created_at DESC`, deploymentID)
		} else {
			rows, err = h.DB.Query(cols+` FROM model_instances ORDER BY created_at DESC`)
		}
	} else {
		if deploymentID != "" {
			rows, err = h.DB.Query(cols+` FROM model_instances WHERE deployment_id = ? ORDER BY created_at DESC`, deploymentID)
		} else {
			rows, err = h.DB.Query(cols+` FROM model_instances ORDER BY created_at DESC`)
		}
	}
	if err != nil {
		log.Error("list model instances", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, scanModelInstances(rows))
}

func (h *ModelHandler) HandleGetModelInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	mi := scanMiRow(h.DB.QueryRow(`SELECT id, deployment_id, replica_index, node_id, agent_id, runtime_type, gpu_ids, gpu_lease_ids,
	 desired_state, actual_state, container_id, process_id, remote_url, endpoint_url, host_port, container_port,
	 restart_count, last_error, last_exit_code, resolved_run_spec,
	 started_at, stopped_at, last_heartbeat_at, created_at, updated_at
	 FROM model_instances WHERE id = ?`, id))
	if mi == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, mi)
}

func scanModelInstances(rows *sql.Rows) []map[string]interface{} {
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		if m := scanMi(rows); m != nil { out = append(out, m) }
	}
	if out == nil { out = []map[string]interface{}{} }
	return out
}

func scanMi(scanner interface{ Scan(...interface{}) error }) map[string]interface{} {
	var id, did, nid, aid, rt, gids, lids, ds, as, cid, rurl, eurl, le, rrs, ca, ua string
	var sa, soa, lha sql.NullString
	var ri, pid, hp, cp, rc, lec int
	if err := scanner.Scan(&id, &did, &ri, &nid, &aid, &rt, &gids, &lids, &ds, &as, &cid, &pid, &rurl, &eurl, &hp, &cp, &rc, &le, &lec, &rrs, &sa, &soa, &lha, &ca, &ua); err != nil {
		return nil
	}
	saStr := ""; if sa.Valid { saStr = sa.String }
	soaStr := ""; if soa.Valid { soaStr = soa.String }
	lhaStr := ""; if lha.Valid { lhaStr = lha.String }
	return map[string]interface{}{
		"id":id,"deployment_id":did,"replica_index":ri,"node_id":nid,"agent_id":aid,"runtime_type":rt,
		"gpu_ids":json.RawMessage(gids),"gpu_lease_ids":json.RawMessage(lids),
		"desired_state":ds,"actual_state":as,"container_id":cid,"process_id":pid,"remote_url":rurl,"endpoint_url":eurl,
		"host_port":hp,"container_port":cp,"restart_count":rc,"last_error":le,"last_exit_code":lec,
		"resolved_run_spec":json.RawMessage(rrs),"started_at":saStr,"stopped_at":soaStr,"last_heartbeat_at":lhaStr,
		"created_at":ca,"updated_at":ua,
	}
}

func scanMiRow(row *sql.Row) map[string]interface{} {
	var id, did, nid, aid, rt, gids, lids, ds, as, cid, rurl, eurl, le, rrs, ca, ua string
	var sa, soa, lha sql.NullString
	var ri, pid, hp, cp, rc, lec int
	if err := row.Scan(&id, &did, &ri, &nid, &aid, &rt, &gids, &lids, &ds, &as, &cid, &pid, &rurl, &eurl, &hp, &cp, &rc, &le, &lec, &rrs, &sa, &soa, &lha, &ca, &ua); err != nil {
		return nil
	}
	saStr := ""; if sa.Valid { saStr = sa.String }
	soaStr := ""; if soa.Valid { soaStr = soa.String }
	lhaStr := ""; if lha.Valid { lhaStr = lha.String }
	return map[string]interface{}{
		"id":id,"deployment_id":did,"replica_index":ri,"node_id":nid,"agent_id":aid,"runtime_type":rt,
		"gpu_ids":json.RawMessage(gids),"gpu_lease_ids":json.RawMessage(lids),
		"desired_state":ds,"actual_state":as,"container_id":cid,"process_id":pid,"remote_url":rurl,"endpoint_url":eurl,
		"host_port":hp,"container_port":cp,"restart_count":rc,"last_error":le,"last_exit_code":lec,
		"resolved_run_spec":json.RawMessage(rrs),"started_at":saStr,"stopped_at":soaStr,"last_heartbeat_at":lhaStr,
		"created_at":ca,"updated_at":ua,
	}
}

// ==========================================================================
// GpuLease read-only
// ==========================================================================

func (h *ModelHandler) HandleListGpuLeases(w http.ResponseWriter, r *http.Request) {
	var rows *sql.Rows
	var err error
	if isPlatformAdmin(r) {
		rows, err = h.DB.Query(`SELECT id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status, expires_at, reserved_at, activated_at, released_at, created_at, updated_at FROM gpu_leases ORDER BY created_at DESC`)
	} else {
		rows, err = h.DB.Query(`SELECT id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status, expires_at, created_at, updated_at FROM gpu_leases WHERE tenant_id = ? ORDER BY created_at DESC`, tenantID(r))
	}
	if err != nil {
		log.Error("list gpu leases", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, scanGpuLeases(rows))
}

func (h *ModelHandler) HandleGetGpuLease(w http.ResponseWriter, r *http.Request) {
	gl := scanGlRow(h.DB.QueryRow(`SELECT id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status, expires_at, reserved_at, activated_at, released_at, created_at, updated_at FROM gpu_leases WHERE id = ?`, r.PathValue("id")))
	if gl == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !isPlatformAdmin(r) && gl["tenant_id"] != tenantID(r) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, gl)
}

func scanGpuLeases(rows *sql.Rows) []map[string]interface{} {
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		if m := scanGl(rows); m != nil { out = append(out, m) }
	}
	if out == nil { out = []map[string]interface{}{} }
	return out
}

func scanGl(scanner interface{ Scan(...interface{}) error }) map[string]interface{} {
	var id, gid, nid, did, iid, tid, status, reserved_at, ca, ua string
	var exp, activated_at, released_at sql.NullString
	if err := scanner.Scan(&id, &gid, &nid, &did, &iid, &tid, &status, &exp, &reserved_at, &activated_at, &released_at, &ca, &ua); err != nil {
		return nil
	}
	expStr := ""; if exp.Valid { expStr = exp.String }
	actStr := ""; if activated_at.Valid { actStr = activated_at.String }
	relStr := ""; if released_at.Valid { relStr = released_at.String }
	return map[string]interface{}{
		"id":id,"gpu_id":gid,"node_id":nid,"deployment_id":did,"instance_id":iid,
		"tenant_id":tid,"status":status,"expires_at":expStr,
		"reserved_at":reserved_at,"activated_at":actStr,"released_at":relStr,
		"created_at":ca,"updated_at":ua,
	}
}

func scanGlRow(row *sql.Row) map[string]interface{} {
	var id, gid, nid, did, iid, tid, status, reserved_at, ca, ua string
	var exp, activated_at, released_at sql.NullString
	if err := row.Scan(&id, &gid, &nid, &did, &iid, &tid, &status, &exp, &reserved_at, &activated_at, &released_at, &ca, &ua); err != nil {
		return nil
	}
	expStr := ""; if exp.Valid { expStr = exp.String }
	actStr := ""; if activated_at.Valid { actStr = activated_at.String }
	relStr := ""; if released_at.Valid { relStr = released_at.String }
	return map[string]interface{}{
		"id":id,"gpu_id":gid,"node_id":nid,"deployment_id":did,"instance_id":iid,
		"tenant_id":tid,"status":status,"expires_at":expStr,
		"reserved_at":reserved_at,"activated_at":actStr,"released_at":relStr,
		"created_at":ca,"updated_at":ua,
	}
}
