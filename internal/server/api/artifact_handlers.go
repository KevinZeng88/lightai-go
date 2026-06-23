package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/runtimecontract"
	"lightai-go/internal/server/authz"

	"github.com/google/uuid"
)

// Allowed capability values — canonical source is runtimecontract.IsValidCapability.
var allowedCapabilities = map[string]bool{
	runtimecontract.CapabilityChat:             true,
	runtimecontract.CapabilityCompletion:       true,
	runtimecontract.CapabilityEmbedding:        true,
	runtimecontract.CapabilityRerank:           true,
	runtimecontract.CapabilityVision:           true,
	runtimecontract.CapabilityToolCalling:      true,
	runtimecontract.CapabilityStructuredOutput: true,
}

// Allowed capability source values — canonical source is runtimecontract.IsValidCapabilitySource.
var allowedCapabilitySources = map[string]bool{
	runtimecontract.CapabilitySourceScan:         true,
	runtimecontract.CapabilitySourceInferred:     true,
	runtimecontract.CapabilitySourceUserOverride: true,
	runtimecontract.CapabilitySourceBackendProbe: true,
}

// Allowed task type values — canonical source is runtimecontract.IsValidTask.
var allowedTaskTypes = map[string]bool{
	runtimecontract.TaskChat:       true,
	runtimecontract.TaskCompletion: true,
	runtimecontract.TaskEmbedding:  true,
	runtimecontract.TaskRerank:     true,
	runtimecontract.TaskVisionChat: true,
	runtimecontract.TaskAdapter:    true,
	runtimecontract.TaskUnknown:    true,
}

// Allowed default_test_mode values — canonical source is runtimecontract.IsValidTestMode.
var allowedTestModes = map[string]bool{
	runtimecontract.TestModeAuto:       true,
	runtimecontract.TestModeChat:       true,
	runtimecontract.TestModeCompletion: true,
	runtimecontract.TestModeEmbedding:  true,
	runtimecontract.TestModeRerank:     true,
}

// validateCapabilitiesJSON checks that all values in the given JSON array are
// valid capability strings. Returns the array and nil on success, or nil and
// an error message on validation failure.
func validateCapabilitiesJSON(raw interface{}) ([]interface{}, error) {
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("capabilities_json must be a JSON array")
	}
	for _, v := range arr {
		s, ok := v.(string)
		if !ok || !allowedCapabilities[s] {
			return nil, fmt.Errorf("invalid capability: %v", v)
		}
	}
	return arr, nil
}

// validateDefaultTestMode checks that the given value is a valid test mode.
func validateDefaultTestMode(v string) error {
	if v == "" {
		return nil // empty is fine, will default
	}
	if !allowedTestModes[v] {
		return fmt.Errorf("invalid default_test_mode: %s (allowed: auto, chat, completion, embedding, rerank)", v)
	}
	return nil
}

// normalizeCapabilitySources ensures all capability sources are valid and
// user-provided capabilities are marked as user_override.
func normalizeCapabilitySources(capabilities []interface{}, sources map[string]interface{}) map[string]interface{} {
	if sources == nil {
		sources = make(map[string]interface{})
	}
	for _, c := range capabilities {
		key, ok := c.(string)
		if !ok {
			continue
		}
		existing, hasExisting := sources[key]
		if !hasExisting || existing == "inferred" || existing == "scan" || existing == "" {
			sources[key] = "user_override"
		}
	}
	return sources
}

// ==========================================================================
// ModelArtifact CRUD
// ==========================================================================

func (h *AgentHandler) HandleListArtifacts(w http.ResponseWriter, r *http.Request) {
	tid := tenantID(r)
	var err error
	var out []map[string]interface{}
	if isPlatformAdmin(r) {
		out, err = h.queryArtifacts(`SELECT id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, capabilities_json, capability_sources_json, default_test_mode, tenant_id, created_at, updated_at FROM model_artifacts ORDER BY name`)
	} else {
		out, err = h.queryArtifacts(`SELECT id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, capabilities_json, capability_sources_json, default_test_mode, tenant_id, created_at, updated_at FROM model_artifacts WHERE tenant_id = ? ORDER BY name`, tid)
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
	path := strVal(req, "path", "")
	format := strVal(req, "format", "custom")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")
		return
	}
	// GGUF is a single-file format; a directory path is semantically wrong and
	// causes downstream failures (mount, -m argument, etc.).
	if format == "gguf" && !strings.Contains(path, ".gguf") {
		writeError(w, http.StatusBadRequest, "GGUF format requires a .gguf file path, not a directory")
		return
	}

	taskType := strVal(req, "task_type", "chat")
	if !allowedTaskTypes[taskType] {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid task_type: %s", taskType))
		return
	}

	id := uuid.NewString()
	tid := tenantID(r)
	now := time.Now().Format(time.RFC3339)

	_, err := h.DB.Exec(`INSERT INTO model_artifacts (id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, capabilities_json, capability_sources_json, default_test_mode, tenant_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), strVal(req, "source_type", "local_path"),
		path, format, taskType,
		strVal(req, "architecture", "custom"), strVal(req, "size_label", ""),
		strVal(req, "quantization", "unknown"), intVal(req, "default_context_length", 0),
		int64Val(req, "estimated_vram_bytes", 0), intVal(req, "required_gpu_count", 1),
		strVal(req, "capabilities_json", "[]"), strVal(req, "capability_sources_json", "{}"), strVal(req, "default_test_mode", "auto"),
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

	// Validate GGUF path when format or path is being changed.
	newFormat := strVal(req, "format", strVal(existing, "format", "custom"))
	newPath := strVal(req, "path", strVal(existing, "path", ""))
	if newFormat == "gguf" && !strings.Contains(newPath, ".gguf") {
		writeError(w, http.StatusBadRequest, "GGUF format requires a .gguf file path, not a directory")
		return
	}

	sets := []string{"updated_at = ?"}
	args := []interface{}{now}
	for _, f := range []string{"display_name", "path", "format", "source_type", "task_type", "architecture", "size_label", "quantization"} {
		if f == "task_type" {
			if tv, ok := req[f]; ok {
				if ts, ok2 := tv.(string); ok2 && !allowedTaskTypes[ts] {
					writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid task_type: %s", ts))
					return
				}
			}
		}
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
	// Handle capabilities_json with validation.
	if v, ok := req["capabilities"]; ok {
		caps, err := validateCapabilitiesJSON(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sets = append(sets, "capabilities_json = ?")
		args = append(args, jsonString(caps))
		// Normalize sources: user-provided caps get user_override source.
		var rawSources interface{}
		if sv, ok2 := req["capability_sources"]; ok2 {
			rawSources = sv
		} else if existingSources, ok3 := existing["capability_sources"]; ok3 {
			rawSources = existingSources
		}
		if rawSources != nil {
			if sm, ok4 := rawSources.(map[string]interface{}); ok4 {
				normalizedSources := normalizeCapabilitySources(caps, sm)
				sets = append(sets, "capability_sources_json = ?")
				args = append(args, jsonString(normalizedSources))
			}
		}
	} else if v, ok := req["capability_sources"]; ok {
		// Allow updating sources independently.
		sets = append(sets, "capability_sources_json = ?")
		args = append(args, jsonString(v))
	}
	if v, ok := req["default_test_mode"]; ok {
		tm, _ := v.(string)
		if err := validateDefaultTestMode(tm); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sets = append(sets, "default_test_mode = ?")
		args = append(args, tm)
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

	tx, txErr := h.DB.Begin()
	if txErr != nil {
		log.OperationFailed(ctx, "model_artifact.delete", "tx_begin", opStart, txErr, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	var deploymentCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM model_deployments WHERE model_artifact_id = ?`, id).Scan(&deploymentCount); err != nil {
		log.OperationFailed(ctx, "model_artifact.delete", "deployment_check", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if deploymentCount > 0 {
		writeError(w, http.StatusConflict, "model artifact is still referenced by deployments")
		return
	}
	if _, err := tx.Exec(`DELETE FROM model_locations WHERE model_artifact_id = ?`, id); err != nil {
		log.OperationFailed(ctx, "model_artifact.delete", "location_delete", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if _, err := tx.Exec(`DELETE FROM model_artifacts WHERE id = ?`, id); err != nil {
		log.OperationFailed(ctx, "model_artifact.delete", "db_write", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := tx.Commit(); err != nil {
		log.OperationFailed(ctx, "model_artifact.delete", "tx_commit", opStart, err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	log.OperationCompleted(ctx, "model_artifact.delete", opStart, "id", id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AgentHandler) getArtifactJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, capabilities_json, capability_sources_json, default_test_mode, tenant_id, created_at, updated_at FROM model_artifacts WHERE id = ?`, id)
	var rid, name, dn, st, path, frmt, tt, arch, sl, quant, capsJSON, sourcesJSON, testMode, tid, ca, ua string
	var ctxLen, gpuCount int
	var vram int64
	if err := row.Scan(&rid, &name, &dn, &st, &path, &frmt, &tt, &arch, &sl, &quant, &ctxLen, &vram, &gpuCount, &capsJSON, &sourcesJSON, &testMode, &tid, &ca, &ua); err != nil {
		return nil
	}
	var caps interface{}
	if err := json.Unmarshal([]byte(capsJSON), &caps); err != nil {
		caps = []interface{}{}
	}
	var sources interface{}
	if err := json.Unmarshal([]byte(sourcesJSON), &sources); err != nil {
		sources = map[string]interface{}{}
	}
	return map[string]interface{}{"id": rid, "name": name, "display_name": dn, "source_type": st, "path": path, "format": frmt, "task_type": tt, "architecture": arch, "size_label": sl, "quantization": quant, "default_context_length": ctxLen, "estimated_vram_bytes": vram, "required_gpu_count": gpuCount, "capabilities": caps, "capability_sources": sources, "default_test_mode": testMode, "tenant_id": tid, "created_at": ca, "updated_at": ua, "locations": h.listModelLocations(id)}
}

func (h *AgentHandler) queryArtifacts(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var rid, name, dn, st, path, frmt, tt, arch, sl, quant, capsJSON, sourcesJSON, testMode, tid, ca, ua string
		var ctxLen, gpuCount int
		var vram int64
		if err := rows.Scan(&rid, &name, &dn, &st, &path, &frmt, &tt, &arch, &sl, &quant, &ctxLen, &vram, &gpuCount, &capsJSON, &sourcesJSON, &testMode, &tid, &ca, &ua); err != nil {
			continue
		}
		var caps interface{}
		if err := json.Unmarshal([]byte(capsJSON), &caps); err != nil {
			caps = []interface{}{}
		}
		var sources interface{}
		if err := json.Unmarshal([]byte(sourcesJSON), &sources); err != nil {
			sources = map[string]interface{}{}
		}
		out = append(out, map[string]interface{}{"id": rid, "name": name, "display_name": dn, "source_type": st, "path": path, "format": frmt, "task_type": tt, "architecture": arch, "size_label": sl, "quantization": quant, "default_context_length": ctxLen, "estimated_vram_bytes": vram, "required_gpu_count": gpuCount, "capabilities": caps, "capability_sources": sources, "default_test_mode": testMode, "tenant_id": tid, "created_at": ca, "updated_at": ua})
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

	// BRR-RV-005: Wrap artifact + location inserts in a transaction so a
	// partial failure does not leave an orphan model_artifact row.
	tx, txErr := h.DB.Begin()
	if txErr != nil {
		log.Error("discover_artifact.tx_begin_failed", "error", txErr)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	_, err := tx.Exec(`INSERT INTO model_artifacts (id, name, display_name, source_type, path, format, task_type, architecture, size_label, quantization, default_context_length, estimated_vram_bytes, required_gpu_count, capabilities_json, capability_sources_json, default_test_mode, tenant_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), strVal(req, "source_type", "local_path"),
		absolutePath, strVal(req, "format", "custom"), strVal(req, "task_type", "chat"),
		strVal(req, "architecture", "custom"), strVal(req, "size_label", ""),
		strVal(req, "quantization", "unknown"), intVal(req, "default_context_length", 0),
		int64Val(req, "estimated_vram_bytes", 0), intVal(req, "required_gpu_count", 1),
		strVal(req, "capabilities_json", "[]"), strVal(req, "capability_sources_json", "{}"), strVal(req, "default_test_mode", "auto"),
		tid, now, now)
	if err != nil {
		log.Error("discover_artifact.insert_artifact_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	locReq := map[string]interface{}{
		"node_id":       strVal(req, "node_id", ""),
		"absolute_path": absolutePath,
		"path_type":     strVal(req, "path_type", "directory"),
	}
	locationID := uuid.NewString()
	if _, err := tx.Exec(`INSERT INTO model_locations (id, model_artifact_id, node_id, path_type, model_root, relative_path, absolute_path, verification_status, match_status, tenant_id, last_scanned_at, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		locationID, id, locReq["node_id"], locReq["path_type"], filepath.Dir(absolutePath), filepath.Base(absolutePath), absolutePath, "verified", "exact_match", tid, now, now, now); err != nil {
		log.Error("discover_artifact.insert_location_failed", "artifact_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Error("discover_artifact.tx_commit_failed", "artifact_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
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
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "node_id is required")
		return
	}
	pathType := strVal(req, "path_type", "directory")
	modelRoot, relativePath, absolutePath, err := h.resolveModelLocationRequestPath(nodeID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// GGUF models require a .gguf file path, not a directory (WEB-AI-RC-006).
	if strVal(artifact, "format", "") == "gguf" && !strings.HasSuffix(absolutePath, ".gguf") {
		writeError(w, http.StatusBadRequest, "GGUF models require a .gguf file path, not a directory. Please select the specific .gguf file.")
		return
	}
	id := uuid.NewString()
	tid := artifact["tenant_id"].(string)
	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(`INSERT INTO model_locations
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

func (h *AgentHandler) resolveModelLocationRequestPath(nodeID string, req map[string]interface{}) (string, string, string, error) {
	rootID := strVal(req, "root_id", "")
	rootPath := strVal(req, "model_root", strVal(req, "root", ""))
	relativePath := strVal(req, "relative_path", "")
	if rootID == "" && rootPath == "" {
		absolutePath := filepath.Clean(strVal(req, "absolute_path", strVal(req, "path", "")))
		if absolutePath == "." || absolutePath == "" {
			return "", "", "", fmt.Errorf("root_id or model_root and relative_path are required")
		}
		roots, err := h.listNodeModelRoots(nodeID, false)
		if err != nil {
			return "", "", "", fmt.Errorf("root not allowed")
		}
		for _, root := range roots {
			if pathWithinRoot(absolutePath, root.Path) {
				rel, err := filepath.Rel(root.Path, absolutePath)
				if err != nil {
					return "", "", "", fmt.Errorf("path traversal blocked")
				}
				relativePath = rel
				rootPath = root.Path
				rootID = root.ID
				break
			}
		}
	}
	root, err := h.resolveNodeModelRoot(nodeID, rootID, rootPath)
	if err != nil {
		return "", "", "", fmt.Errorf("root not allowed")
	}
	rel, err := safeRelativePath(relativePath)
	if err != nil {
		return "", "", "", err
	}
	abs := filepath.Clean(filepath.Join(root.Path, rel))
	if !pathWithinRoot(abs, root.Path) {
		return "", "", "", fmt.Errorf("path traversal blocked")
	}
	return root.Path, rel, abs, nil
}

func (h *AgentHandler) HandleRescanModelLocation(w http.ResponseWriter, r *http.Request) {
	locationID := r.PathValue("location_id")
	if !authz.CheckModelLocationTenant(r, h.DB.DB, locationID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	now := time.Now().Format(time.RFC3339)
	if _, err := h.DB.Exec(`UPDATE model_locations SET last_scanned_at = ?, last_error = '', updated_at = ? WHERE id = ?`, now, now, locationID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, h.getModelLocationJSON(locationID))
}

func (h *AgentHandler) HandleAttestModelLocation(w http.ResponseWriter, r *http.Request) {
	locationID := r.PathValue("location_id")
	if !authz.CheckModelLocationTenant(r, h.DB.DB, locationID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
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

// HandleGetModelCapabilityEnums returns the canonical lists of valid format, task,
// capability, and test mode values. Used by the frontend to populate option lists
// without hardcoding enum values client-side.
//
// Authentication required (same as model_artifact:read).
func (h *AgentHandler) HandleGetModelCapabilityEnums(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"formats":      runtimecontract.AllFormats(),
		"tasks":        runtimecontract.AllTasks(),
		"capabilities": runtimecontract.AllCapabilities(),
		"test_modes":   runtimecontract.AllTestModes(),
	})
}
