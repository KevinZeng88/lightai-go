package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/catalog"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var (
	backendCatalogSystemVersionsDir = "configs/backend-catalog/versions"
	backendCatalogUserVersionsDir   = defaultBackendCatalogUserVersionsDir()
	backendCatalogSystemRuntimesDir = "configs/backend-catalog/runtimes"
	backendCatalogUserRuntimesDir   = defaultBackendCatalogUserRuntimesDir()
	backendCatalogSystemBackendsDir = "configs/backend-catalog/backends"
	backendCatalogUserBackendsDir   = defaultBackendCatalogUserBackendsDir()
)

func defaultBackendCatalogUserRuntimesDir() string {
	if dir := strings.TrimSpace(os.Getenv("LIGHTAI_BACKEND_CATALOG_USER_DIR")); dir != "" {
		return filepath.Join(dir, "runtimes")
	}
	return "data/backend-catalog.d/user/runtimes"
}

func defaultBackendCatalogUserBackendsDir() string {
	if dir := strings.TrimSpace(os.Getenv("LIGHTAI_BACKEND_CATALOG_USER_DIR")); dir != "" {
		return filepath.Join(dir, "backends")
	}
	return "data/backend-catalog.d/user/backends"
}

func defaultBackendCatalogUserVersionsDir() string {
	if dir := strings.TrimSpace(os.Getenv("LIGHTAI_BACKEND_CATALOG_USER_DIR")); dir != "" {
		return dir
	}
	return "data/backend-catalog.d/user"
}

// ==========================================================================
// InferenceBackend
// ==========================================================================

func (h *AgentHandler) HandleListBackends(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, name, display_name, description, slug, managed_by, source, catalog_version, checksum, status, config_set_json, source_metadata_json, created_at, updated_at FROM inference_backends ORDER BY name`)
	if err != nil {
		log.Error("list backends failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, name, dn, desc, slug, managedBy, source, catalogVersion, checksum, status, configSetRaw, sourceMetaRaw, ca, ua string
		if err := rows.Scan(&id, &name, &dn, &desc, &slug, &managedBy, &source, &catalogVersion, &checksum, &status, &configSetRaw, &sourceMetaRaw, &ca, &ua); err != nil {
			continue
		}
		configSet := parseConfigSet(configSetRaw)
		out = append(out, map[string]interface{}{
			"id": id, "name": name, "display_name": dn, "description": desc,
			"slug": slug, "managed_by": managedBy, "source": source, "catalog_version": catalogVersion,
			"checksum": checksum, "status": status,
			"is_builtin": managedBy == "system", "is_enabled": status == "active",
			"capabilities": configObject(configSet, "backend.capabilities"),
			"config_set":   configSet, "config_set_json": json.RawMessage(configSetRaw),
			"source_metadata": configSourceMetadata(sourceMetaRaw), "source_metadata_json": json.RawMessage(sourceMetaRaw),
			"created_at": ca, "updated_at": ua,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleGetBackend(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row := h.DB.QueryRow(`SELECT id, name, display_name, description, slug, managed_by, source, catalog_version, checksum, status, config_set_json, source_metadata_json, created_at, updated_at FROM inference_backends WHERE id = ?`, id)
	var bid, name, dn, desc, slug, managedBy, source, catalogVersion, checksum, status, configSetRaw, sourceMetaRaw, ca, ua string
	if err := row.Scan(&bid, &name, &dn, &desc, &slug, &managedBy, &source, &catalogVersion, &checksum, &status, &configSetRaw, &sourceMetaRaw, &ca, &ua); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	configSet := parseConfigSet(configSetRaw)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": bid, "name": name, "display_name": dn, "description": desc,
		"slug": slug, "managed_by": managedBy, "source": source, "catalog_version": catalogVersion,
		"checksum": checksum, "status": status,
		"is_builtin": managedBy == "system", "is_enabled": status == "active",
		"capabilities": configObject(configSet, "backend.capabilities"),
		"config_set":   configSet, "config_set_json": json.RawMessage(configSetRaw),
		"source_metadata": configSourceMetadata(sourceMetaRaw), "source_metadata_json": json.RawMessage(sourceMetaRaw),
		"created_at": ca, "updated_at": ua,
	})
}

func (h *AgentHandler) HandleListBackendVersions(w http.ResponseWriter, r *http.Request) {
	backendID := r.PathValue("id")
	rows, err := h.DB.Query(backendVersionSelectSQL()+` WHERE backend_id = ? AND status != 'deprecated' ORDER BY version`, backendID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		item, err := scanBackendVersionMap(rows)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleListAllBackendVersions(w http.ResponseWriter, r *http.Request) {
	backendID := r.URL.Query().Get("backend_id")
	query := backendVersionSelectSQL()
	var args []interface{}
	if backendID != "" {
		query += ` WHERE backend_id = ? AND status != 'deprecated'`
		args = append(args, backendID)
	} else {
		query += ` WHERE status != 'deprecated'`
	}
	query += ` ORDER BY backend_id, version`
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		item, err := scanBackendVersionMap(rows)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleCreateBackendVersion(w http.ResponseWriter, r *http.Request) {
	backendID := r.PathValue("id")
	if backendID == "" {
		writeError(w, http.StatusBadRequest, "backend_id is required")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	version := strings.TrimSpace(strVal(req, "version", ""))
	if version == "" {
		writeError(w, http.StatusBadRequest, "version is required")
		return
	}
	id := strVal(req, "id", "")
	if id == "" {
		id = "backend-version.user." + uuid.NewString()
	}
	if err := h.upsertBackendVersionFromRequest(id, backendID, req, true); err != nil {
		writeError(w, backendVersionUpsertStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, h.getBackendVersionJSON(id))
}

func (h *AgentHandler) HandlePatchBackendVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("version_id")
	if id == "" {
		id = r.PathValue("id")
	}
	existing := h.getBackendVersionJSON(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if managedBy, _ := existing["managed_by"].(string); managedBy == "system" {
		writeError(w, http.StatusConflict, "system backend version is read-only; clone before editing")
		return
	}
	if readonly, _ := existing["readonly"].(bool); readonly {
		writeError(w, http.StatusConflict, "system backend version is read-only; clone before editing")
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	backendID, _ := existing["backend_id"].(string)
	if err := h.upsertBackendVersionFromRequest(id, backendID, req, false); err != nil {
		writeError(w, backendVersionUpsertStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.getBackendVersionJSON(id))
}

func (h *AgentHandler) HandleDeleteBackendVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("version_id")
	existing := h.getBackendVersionJSON(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if managedBy, _ := existing["managed_by"].(string); managedBy == "system" {
		writeError(w, http.StatusConflict, "system backend version is read-only")
		return
	}
	var used int
	h.DB.QueryRow(`SELECT COUNT(*) FROM backend_runtimes WHERE backend_version_id=?`, id).Scan(&used)
	if used > 0 {
		writeError(w, http.StatusConflict, "backend version is used by runtime templates")
		return
	}
	if loadedFrom := strings.TrimSpace(strVal(existing, "loaded_from", "")); loadedFrom != "" {
		if absUser, err := filepath.Abs(backendCatalogUserVersionsDir); err == nil {
			if absLoaded, err := filepath.Abs(loadedFrom); err == nil && strings.HasPrefix(absLoaded, absUser+string(os.PathSeparator)) {
				if err := os.Remove(absLoaded); err != nil && !os.IsNotExist(err) {
					writeError(w, http.StatusInternalServerError, "delete user catalog failed")
					return
				}
			}
		}
	}
	h.DB.Exec(`DELETE FROM backend_versions WHERE id=?`, id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AgentHandler) HandleCloneBackendVersion(w http.ResponseWriter, r *http.Request) {
	sourceID := r.PathValue("version_id")
	source := h.getBackendVersionJSON(sourceID)
	if source == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	id := "backend-version.user." + uuid.NewString()
	short := strings.Split(id, ".")
	suffix := short[len(short)-1]
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	version := strVal(source, "version", "") + "-user-" + suffix
	req := map[string]interface{}{
		"version":      version,
		"display_name": strVal(source, "display_name", version) + " (user)",
		"slug":         slugify(version),
		"description":  strVal(source, "description", ""),
		"config_set":   source["config_set"],
	}
	if err := h.upsertBackendVersionFromRequest(id, strVal(source, "backend_id", ""), req, true); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, h.getBackendVersionJSON(id))
}

func (h *AgentHandler) HandleReloadBackendCatalog(w http.ResponseWriter, r *http.Request) {
	ctx, opStart := log.StartOperation(r.Context(), "backend_catalog.reload")
	sum, err := h.reloadAllCatalogs()
	if err != nil {
		log.OperationFailed(ctx, "backend_catalog.reload", "reload", opStart, err)
		writeError(w, http.StatusInternalServerError, "reload failed")
		return
	}
	log.OperationCompleted(ctx, "backend_catalog.reload", opStart,
		"backends", sum["backends"], "versions", sum["versions"], "runtimes", sum["runtimes"])
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "reloaded", "backends": sum["backends"], "versions": sum["versions"], "runtimes": sum["runtimes"]})
}

func (h *AgentHandler) ReloadBackendCatalogProjection() (int, error) {
	sum, err := h.reloadAllCatalogs()
	if err != nil {
		return 0, err
	}
	return sum["versions"], nil // keep legacy return type compat
}

// reloadAllCatalogs reloads Backend, BackendVersion, and BackendRuntime
// from system and user catalog files into DB projection tables.
func (h *AgentHandler) reloadAllCatalogs() (map[string]int, error) {
	if err := catalog.SeedCatalog(h.DB.DB, "", ""); err != nil {
		return map[string]int{}, err
	}
	var result = map[string]int{}
	var backends, versions, runtimes int
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM inference_backends`).Scan(&backends)
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM backend_versions`).Scan(&versions)
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM backend_runtimes`).Scan(&runtimes)
	result["backends"] = backends
	result["versions"] = versions
	result["runtimes"] = runtimes
	return result, nil
}

func (h *AgentHandler) upsertBackendVersionFromRequest(id, backendID string, req map[string]interface{}, creating bool) error {
	if backendID == "" {
		return fmt.Errorf("backend_id is required")
	}
	if field := backendVersionRuntimeOnlyField(req); field != "" {
		return fmt.Errorf("%s belongs to BackendRuntime, not BackendVersion", field)
	}
	var exists int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM inference_backends WHERE id=?`, backendID).Scan(&exists); err != nil || exists == 0 {
		return fmt.Errorf("backend not found")
	}
	now := time.Now().Format(time.RFC3339)
	current := h.getBackendVersionJSON(id)
	version := strings.TrimSpace(strVal(req, "version", ""))
	if version == "" && current != nil {
		version = strVal(current, "version", "")
	}
	if version == "" {
		return fmt.Errorf("version is required")
	}
	displayName := strings.TrimSpace(strVal(req, "display_name", ""))
	if displayName == "" && current != nil {
		displayName = strVal(current, "display_name", "")
	}
	if displayName == "" {
		displayName = version
	}
	isDefault := 0
	if current != nil {
		if b, ok := current["is_default"].(bool); ok && b {
			isDefault = 1
		}
	}
	if _, ok := req["is_default"]; ok {
		isDefault = boolInt(boolVal(req, "is_default", false))
	}
	isDeprecated := 0
	if current != nil {
		if b, ok := current["is_deprecated"].(bool); ok && b {
			isDeprecated = 1
		}
	}
	if _, ok := req["is_deprecated"]; ok {
		isDeprecated = boolInt(boolVal(req, "is_deprecated", false))
	}
	slug := strings.TrimSpace(strVal(req, "slug", ""))
	if slug == "" && current != nil {
		slug = strVal(current, "slug", "")
	}
	if slug == "" {
		slug = slugify(version)
	}
	description := strings.TrimSpace(strVal(req, "description", ""))
	if description == "" && current != nil {
		description = strVal(current, "description", "")
	}
	protocol := strings.TrimSpace(strVal(req, "protocol", ""))
	if protocol == "" && current != nil {
		protocol = strVal(current, "protocol", "")
	}
	revision := strings.TrimSpace(strVal(req, "revision", ""))
	if revision == "" {
		revision = checksumString(id + version + now)
	}
	configSet := mapFromAny(req["config_set"])
	callerProvided := len(configSet) > 0
	if !callerProvided {
		configSet = mapFromAny(req["config_set_json"])
		callerProvided = len(configSet) > 0
	}
	// Validate caller-provided ConfigSet: must be strict tiered shape
	if callerProvided {
		if err := validateTieredConfigSet(configSet); err != nil {
			return err
		}
	}
		if len(configSet) == 0 && current != nil {
			configSet = mapFromAny(current["config_set"])
		}
		if len(configSet) == 0 {
			var backendConfigRaw string
			if err := h.DB.QueryRow(`SELECT config_set_json FROM inference_backends WHERE id=?`, backendID).Scan(&backendConfigRaw); err != nil {
				return fmt.Errorf("read backend config set: %w", err)
			}
			configSet = copyConfigSet(backendConfigRaw)
		}
	if health, ok := req["health_check"]; ok {
		setConfigValue(configSet, "runtime.health", health, "BackendVersion", id, "api_request")
	}
	sourceMeta := jsonString(map[string]interface{}{"source_type": "api", "source_ref": id, "copied_at": now, "materialized_at": now, "copy_semantics": "copy_on_create", "copy_boundary": "detached_after_create"})
	checksum := checksumString(id + version + configSetJSON(configSet))
	if creating || current == nil {
		_, err := h.DB.Exec(`INSERT INTO backend_versions
			(id, backend_id, version, display_name, is_default, is_deprecated, slug, managed_by, source, catalog_version, checksum, status, description, readonly, protocol, revision, config_set_json, source_metadata_json, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, backendID, version, displayName, isDefault, isDeprecated, slug, "user", "api", "configset-user", checksum, "active", description, 0, protocol, revision, configSetJSON(configSet), sourceMeta, now, now)
		return err
	}
	_, err := h.DB.Exec(`UPDATE backend_versions SET
		version=?, display_name=?, is_default=?, is_deprecated=?, slug=?, managed_by='user', source='api', catalog_version='configset-user', checksum=?, status='active',
		description=?, readonly=0, protocol=?, revision=?, config_set_json=?, source_metadata_json=?, updated_at=?
		WHERE id=?`,
		version, displayName, isDefault, isDeprecated, slug, checksum, description, protocol, revision, configSetJSON(configSet), sourceMeta, now, id)
	return err
}

func backendVersionRuntimeOnlyField(req map[string]interface{}) string {
	for _, field := range []string{"image_ref", "command", "entrypoint", "model_mount", "docker_options", "devices", "volumes", "env"} {
		if _, ok := req[field]; ok {
			return field
		}
	}
	return ""
}

func backendVersionUpsertStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := err.Error()
		if strings.Contains(msg, "belongs to BackendRuntime") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "not found") ||
			strings.Contains(msg, "tiered shape") ||
			strings.Contains(msg, "flat \"") {
			return http.StatusBadRequest
		}
	return http.StatusInternalServerError
}

func (h *AgentHandler) getBackendVersionJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(backendVersionSelectSQL()+` WHERE id=?`, id)
	item, err := scanBackendVersionMap(row)
	if err != nil {
		return nil
	}
	return item
}

type backendVersionScanner interface {
	Scan(dest ...interface{}) error
}

func backendVersionSelectSQL() string {
	return `SELECT id, backend_id, version, display_name, is_default, is_deprecated, slug, managed_by, source, catalog_version, checksum, status, description, readonly, protocol, revision, config_set_json, source_metadata_json, created_at, updated_at FROM backend_versions`
}

func scanBackendVersionMap(scanner backendVersionScanner) (map[string]interface{}, error) {
	var id, bid, ver, dn, slug, managedBy, source, catalogVersion, checksum, status, desc, protocol, revision, configSetRaw, sourceMetaRaw, ca, ua string
	var isDef, isDep, readonly int
	if err := scanner.Scan(&id, &bid, &ver, &dn, &isDef, &isDep, &slug, &managedBy, &source, &catalogVersion, &checksum, &status, &desc, &readonly, &protocol, &revision, &configSetRaw, &sourceMetaRaw, &ca, &ua); err != nil {
		return nil, err
	}
	configSet := parseConfigSet(configSetRaw)
	return map[string]interface{}{
		"id": id, "backend_id": bid, "version": ver, "display_name": dn,
		"is_default": isDef == 1, "is_deprecated": isDep == 1,
		"slug": slug, "managed_by": managedBy, "source": source, "catalog_version": catalogVersion,
		"checksum": checksum, "status": status, "description": desc,
		"readonly": readonly == 1, "protocol": protocol, "revision": revision,
		"entrypoint":             configStringSlice(configSet, "launcher.entrypoint"),
		"command":                configStringSlice(configSet, "launcher.command"),
		"model_mount":            configObject(configSet, "runtime.model_mount"),
		"health_check":           configObject(configSet, "runtime.health"),
		"capabilities":           configObject(configSet, "backend.capabilities"),
		"supported_config_items": configArray(configSet, "backend.supported_config_items"),
		"config_set":             configSet, "config_set_json": json.RawMessage(configSetRaw),
		"source_metadata": configSourceMetadata(sourceMetaRaw), "source_metadata_json": json.RawMessage(sourceMetaRaw),
		"created_at": ca, "updated_at": ua,
	}, nil
}

func intFromAny(v interface{}, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
	case string:
		var n int
		if _, err := fmt.Sscanf(t, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func firstNonNil(vals ...interface{}) interface{} {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func nilToDefault(v interface{}, def interface{}) interface{} {
	if v == nil {
		return def
	}
	return v
}

func valueOrDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func isEmptyJSON(v interface{}) bool {
	if v == nil {
		return true
	}
	raw := strings.TrimSpace(rawJSONString(v, ""))
	return raw == "" || raw == "{}" || raw == "[]" || raw == "null"
}

func firstStringFromAny(v interface{}) string {
	switch t := v.(type) {
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	case []interface{}:
		if len(t) > 0 {
			return strings.TrimSpace(fmt.Sprint(t[0]))
		}
	case json.RawMessage:
		var arr []string
		if err := json.Unmarshal(t, &arr); err == nil && len(arr) > 0 {
			return arr[0]
		}
	case string:
		if strings.HasPrefix(strings.TrimSpace(t), "[") {
			var arr []string
			if err := json.Unmarshal([]byte(t), &arr); err == nil && len(arr) > 0 {
				return arr[0]
			}
		}
	}
	return ""
}

func checksumString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("sha256:%x", sum[:8])
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-", ":", "-")
	s = replacer.Replace(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "version"
	}
	return out
}

func rawJSONString(v interface{}, def string) string {
	switch t := v.(type) {
	case nil:
		return def
	case json.RawMessage:
		if len(t) == 0 {
			return def
		}
		return string(t)
	case string:
		if t == "" {
			return def
		}
		return t
	default:
		return jsonString(t)
	}
}

func jsonToAny(raw string) interface{} {
	var out interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return raw
	}
	return out
}

func boolValFromRequest(r *http.Request, key string) bool {
	return r.URL.Query().Get(key) == "1" || r.URL.Query().Get(key) == "true"
}

// ==========================================================================
// BackendRuntimeTemplate (read-only from config files)
// ==========================================================================

func HandleListRuntimeTemplates(w http.ResponseWriter, r *http.Request) {
	// Read recursively from the new catalog layout.
	presetsDir := "configs/backend-catalog/runtimes"
	var templates []interface{}
	filepath.WalkDir(presetsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(presetsDir, path)
		templates = append(templates, map[string]interface{}{
			"name":    strings.TrimSuffix(rel, ".yaml"),
			"source":  path,
			"content": string(data),
		})
		return nil
	})
	if templates == nil {
		templates = []interface{}{}
	}
	writeJSON(w, http.StatusOK, templates)
}

func HandleGetRuntimeTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	path := resolveTemplatePath(name)
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

// HandleGetBackendHelp returns parameter help documentation for a backend version.
// Query params: backend (required), version (required), lang (optional, default zh-CN)
// Reads from configs/backend-catalog/help/{backend}/{version}.{lang}.yaml
func HandleGetBackendHelp(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	backend := q.Get("backend")
	version := q.Get("version")
	lang := q.Get("lang")
	if lang == "" {
		lang = "zh-CN"
	}
	if backend == "" || version == "" {
		writeError(w, http.StatusBadRequest, "backend and version query parameters are required")
		return
	}
	helpPath := fmt.Sprintf("configs/backend-catalog/help/%s/%s.%s.yaml", backend, version, lang)
	data, err := os.ReadFile(helpPath)
	if err != nil {
		// Return empty array if help file does not exist (graceful fallback)
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	var entries []map[string]interface{}
	if err := yaml.Unmarshal(data, &entries); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse help file: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
