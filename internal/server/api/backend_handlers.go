package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lightai-go/internal/common/log"
)

// ==========================================================================
// InferenceBackend (read-only)
// ==========================================================================

func (h *AgentHandler) HandleListBackends(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, name, display_name, description, protocol_json, default_version, parameter_format, common_parameters_json, default_env_json, is_builtin, is_enabled, created_at, updated_at FROM inference_backends ORDER BY name`)
	if err != nil {
		log.Error("list backends failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, name, dn, desc, proto, defVer, pfmt, commonP, defEnv, ca, ua string
		var isB, isE int
		if err := rows.Scan(&id, &name, &dn, &desc, &proto, &defVer, &pfmt, &commonP, &defEnv, &isB, &isE, &ca, &ua); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": id, "name": name, "display_name": dn, "description": desc,
			"protocol_json": json.RawMessage(proto), "default_version": defVer,
			"parameter_format": pfmt, "common_parameters_json": json.RawMessage(commonP),
			"default_env_json": redactRawJSON(defEnv), "is_builtin": isB == 1,
			"is_enabled": isE == 1, "created_at": ca, "updated_at": ua,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleGetBackend(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row := h.DB.QueryRow(`SELECT id, name, display_name, description, protocol_json, default_version, parameter_format, common_parameters_json, default_env_json, is_builtin, is_enabled, created_at, updated_at FROM inference_backends WHERE id = ?`, id)
	var bid, name, dn, desc, proto, defVer, pfmt, commonP, defEnv, ca, ua string
	var isB, isE int
	if err := row.Scan(&bid, &name, &dn, &desc, &proto, &defVer, &pfmt, &commonP, &defEnv, &isB, &isE, &ca, &ua); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": bid, "name": name, "display_name": dn, "description": desc,
		"protocol_json": json.RawMessage(proto), "default_version": defVer,
		"parameter_format": pfmt, "common_parameters_json": json.RawMessage(commonP),
		"default_env_json": redactRawJSON(defEnv), "is_builtin": isB == 1,
		"is_enabled": isE == 1, "created_at": ca, "updated_at": ua,
	})
}

func (h *AgentHandler) HandleListBackendVersions(w http.ResponseWriter, r *http.Request) {
	backendID := r.PathValue("id")
	rows, err := h.DB.Query(`SELECT id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated, created_at, updated_at FROM backend_versions WHERE backend_id = ? ORDER BY version`, backendID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, bid, ver, dn, dep, dargs, dbp, pdefs, hc, dimg, env, ca, ua string
		var isDef, isDep int
		var dcp int
		if err := rows.Scan(&id, &bid, &ver, &dn, &isDef, &dep, &dargs, &dbp, &pdefs, &hc, &dcp, &dimg, &env, &isDep, &ca, &ua); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": id, "backend_id": bid, "version": ver, "display_name": dn,
			"is_default": isDef == 1, "default_entrypoint_json": json.RawMessage(dep),
			"default_args_json": json.RawMessage(dargs), "default_backend_params_json": json.RawMessage(dbp),
			"parameter_defs_json": json.RawMessage(pdefs), "health_check_json": json.RawMessage(hc),
			"default_container_port": dcp, "default_images_json": json.RawMessage(dimg),
			"env_json": redactRawJSON(env), "is_deprecated": isDep == 1,
			"created_at": ca, "updated_at": ua,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

// ==========================================================================
// BackendRuntimeTemplate (read-only from config files)
// ==========================================================================

func HandleListRuntimeTemplates(w http.ResponseWriter, r *http.Request) {
	presetsDir := "configs/model-runtime/backend-runtime-templates"
	entries, err := os.ReadDir(presetsDir)
	if err != nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	var templates []interface{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(presetsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		templates = append(templates, map[string]interface{}{
			"name":    name,
			"source":  path,
			"content": string(data),
		})
	}
	if templates == nil {
		templates = []interface{}{}
	}
	writeJSON(w, http.StatusOK, templates)
}

func HandleGetRuntimeTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := filepath.Join("configs/model-runtime/backend-runtime-templates", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":    name,
		"source":  path,
		"content": string(data),
	})
}

// redactRawJSON parses a JSON string and redacts sensitive values.
func redactRawJSON(raw string) json.RawMessage {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return json.RawMessage(raw)
	}
	redacted := redactEnvMap(m)
	b, _ := json.Marshal(redacted)
	return json.RawMessage(b)
}
