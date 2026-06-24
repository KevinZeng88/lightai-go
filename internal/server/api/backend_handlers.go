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
	doc, err := h.backendVersionDocFromRequest(id, backendID, nil, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.writeUserBackendVersionCatalogDoc(doc); err != nil {
		writeError(w, http.StatusInternalServerError, "write user catalog failed")
		return
	}
	if _, err := h.reloadBackendVersionCatalogs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
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
	doc, err := h.backendVersionDocFromRequest(id, backendID, existing, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.writeUserBackendVersionCatalogDoc(doc); err != nil {
		writeError(w, http.StatusInternalServerError, "write user catalog failed")
		return
	}
	if _, err := h.reloadBackendVersionCatalogs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
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
		"version":                   version,
		"display_name":              strVal(source, "display_name", version) + " (user)",
		"slug":                      slugify(version),
		"description":               strVal(source, "description", ""),
		"default_entrypoint_json":   jsonToAny(rawJSONString(source["default_entrypoint_json"], "[]")),
		"default_args_json":         jsonToAny(rawJSONString(source["default_args_json"], "[]")),
		"default_args_schema_json":  jsonToAny(rawJSONString(source["default_args_schema_json"], rawJSONString(source["parameter_defs_json"], "[]"))),
		"default_health_check_json": jsonToAny(rawJSONString(source["default_health_check_json"], rawJSONString(source["health_check_json"], "{}"))),
		"default_env_schema_json":   jsonToAny(rawJSONString(source["default_env_schema_json"], "[]")),
		"default_container_port":    intVal(source, "default_container_port", 8000),
		"default_images_json":       jsonToAny(rawJSONString(source["default_images_json"], "{}")),
		"image_candidates_json":     jsonToAny(rawJSONString(source["image_candidates_json"], "[]")),
		"env_json":                  jsonToAny(rawJSONString(source["env_json"], "{}")),
		"capabilities_json":         jsonToAny(rawJSONString(source["capabilities_json"], "[]")),
		"docker_options_json":       jsonToAny(rawJSONString(source["docker_options_json"], "{}")),
		"model_mount_json":          jsonToAny(rawJSONString(source["model_mount_json"], "{}")),
		"vendor_options_json":       jsonToAny(rawJSONString(source["vendor_options_json"], "{}")),
		"default_host":              strVal(source, "default_host", "0.0.0.0"),
		"default_endpoints_json":    jsonToAny(rawJSONString(source["default_endpoints_json"], "{}")),
		"protocol":                  strVal(source, "protocol", ""),
		"official_reference_json":   jsonToAny(rawJSONString(source["official_reference_json"], "[]")),
	}
	doc, err := h.backendVersionDocFromRequest(id, strVal(source, "backend_id", ""), nil, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.writeUserBackendVersionCatalogDoc(doc); err != nil {
		writeError(w, http.StatusInternalServerError, "write user catalog failed")
		return
	}
	if _, err := h.reloadBackendVersionCatalogs(); err != nil {
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

// reloadBackendCatalogs loads Backend definitions from system and user catalog files.
func (h *AgentHandler) reloadBackendCatalogs() (int, error) {
	count := 0
	if n, err := h.reloadBackendCatalogDir(backendCatalogSystemBackendsDir, "system"); err != nil {
		return count, err
	} else {
		count += n
	}
	if n, err := h.reloadBackendCatalogDir(backendCatalogUserBackendsDir, "user"); err != nil {
		return count, err
	} else {
		count += n
	}
	return count, nil
}

func (h *AgentHandler) reloadBackendCatalogDir(root, source string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() {
			return err
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc backendCatalogDoc
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if err := h.upsertBackendProjection(doc, source, path, data); err != nil {
			return err
		}
		count++
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return count, err
}

func (h *AgentHandler) upsertBackendProjection(doc backendCatalogDoc, source, loadedFrom string, data []byte) error {
	if doc.ID == "" || doc.Name == "" {
		return fmt.Errorf("invalid backend catalog file %s", loadedFrom)
	}
	if source == "" && doc.Source != "" {
		source = doc.Source
	}
	if source == "" && doc.ManagedBy != "" {
		source = doc.ManagedBy
	}
	if source == "" {
		source = "system"
	}
	readonly := source == "system" || doc.Readonly
	slug := doc.Slug
	if slug == "" {
		slug = slugify(doc.Name)
	}
	protocolJSON := jsonString(doc.ProtocolFamily)
	if protocolJSON == "null" || protocolJSON == "" {
		protocolJSON = "[]"
	}
	now := time.Now().Format(time.RFC3339)
	configHash := checksumString(string(data))
	name := doc.Name
	if name == "" {
		name = doc.ID
	}
	displayName := doc.DisplayName
	if displayName == "" {
		displayName = doc.Name
		if displayName == "" {
			displayName = name
		}
	}
	_, err := h.DB.Exec(`INSERT INTO inference_backends
		(id, name, display_name, description, protocol_json, default_version, parameter_format, is_builtin, is_enabled, created_at, updated_at)
		VALUES (?,?,?,?,?,'','',1,1,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			display_name=excluded.display_name,
			description=excluded.description,
			protocol_json=excluded.protocol_json,
			updated_at=excluded.updated_at`,
		doc.ID, name, displayName, doc.Description, protocolJSON, now, now)
	if err != nil {
		return fmt.Errorf("upsert backend %s: %w", doc.ID, err)
	}
	_ = configHash
	_ = readonly
	_ = slug
	return nil
}

// reloadBackendRuntimeCatalogs loads BackendRuntime definitions from system and user catalog files.
func (h *AgentHandler) reloadBackendRuntimeCatalogs() (int, error) {
	count := 0
	if n, err := h.reloadBackendRuntimeCatalogDir(backendCatalogSystemRuntimesDir, "system"); err != nil {
		return count, err
	} else {
		count += n
	}
	if n, err := h.reloadBackendRuntimeCatalogDir(backendCatalogUserRuntimesDir, "user"); err != nil {
		return count, err
	} else {
		count += n
	}
	return count, nil
}

func (h *AgentHandler) reloadBackendRuntimeCatalogDir(root, source string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() {
			return err
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc backendRuntimeCatalogDoc
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if err := h.upsertBackendRuntimeProjection(doc, source, path, data); err != nil {
			return err
		}
		count++
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return count, err
}

func (h *AgentHandler) upsertBackendRuntimeProjection(doc backendRuntimeCatalogDoc, source, loadedFrom string, data []byte) error {
	if doc.ID == "" || doc.BackendID == "" || doc.BackendVersionID == "" {
		return fmt.Errorf("invalid backend runtime catalog file %s", loadedFrom)
	}
	if ok, err := h.backendExists(doc.BackendID); err != nil || !ok {
		return fmt.Errorf("backend %q not found for %s", doc.BackendID, loadedFrom)
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = strings.TrimSpace(doc.Source)
	}
	if source == "" {
		source = strings.TrimSpace(doc.ManagedBy)
	}
	if source == "" {
		source = "user"
	}
	readonly := source == "system"
	if source != "system" && doc.Readonly {
		readonly = true
	}
	isBuiltin := 0
	isEditable := 1
	if readonly {
		isBuiltin = 1
		isEditable = 0
	}
	slug := doc.Slug
	if slug == "" {
		slug = slugify(doc.Name)
	}
	runnerType := doc.RunnerType
	if runnerType == "" {
		runnerType = "docker"
	}
	// Pick first image candidate as default image_name
	imageName := firstStringFromAny(doc.ImageCandidates)
	if imageName == "" && doc.ImageRef != "" {
		imageName = doc.ImageRef
	}
	vendor := doc.Vendor
	name := doc.Name
	if name == "" {
		name = doc.ID
	}
	displayName := doc.DisplayName
	if displayName == "" {
		displayName = doc.Name
		if displayName == "" {
			displayName = name
		}
	}
	now := time.Now().Format(time.RFC3339)
	configHash := checksumString(string(data))

	// Read existing record to preserve runtime config fields that the
	// YAML does not supply. This prevents old-format or partial YAML files
	// from silently overwriting seeded runtime data with empty values.
	var existing struct {
		imageName, ipp, entrypoint, args, env, docker, mount, healthCheck string
	}
	row := h.DB.QueryRow(`SELECT image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json FROM backend_runtimes WHERE id=?`, doc.ID)
	_ = row.Scan(&existing.imageName, &existing.ipp, &existing.entrypoint, &existing.args, &existing.env, &existing.docker, &existing.mount, &existing.healthCheck)

	// Merge: use YAML values when provided, otherwise keep existing DB values.
	// This protects against catalog reloads that would otherwise clear seeded data.
	effectiveImage := imageName
	if effectiveImage == "" && existing.imageName != "" {
		effectiveImage = existing.imageName
	}
	effEntrypoint := jsonString(doc.Entrypoint)
	if isEmptyJSON(effEntrypoint) && !isEmptyJSON(existing.entrypoint) && existing.entrypoint != "null" {
		effEntrypoint = existing.entrypoint
	}
	effArgs := jsonString(doc.Args)
	if isEmptyJSON(effArgs) && !isEmptyJSON(existing.args) && existing.args != "null" {
		effArgs = existing.args
	}
	effEnv := jsonString(doc.EnvSchema)
	if isEmptyJSON(effEnv) && !isEmptyJSON(existing.env) && existing.env != "null" {
		effEnv = existing.env
	}
	effDocker := jsonString(doc.DockerOptions)
	if isEmptyJSON(effDocker) && !isEmptyJSON(existing.docker) && existing.docker != "null" {
		effDocker = existing.docker
	}
	effMount := jsonString(doc.ModelMount)
	if isEmptyJSON(effMount) {
		effMount = jsonString(map[string]interface{}{})
	}
	if isEmptyJSON(effMount) && !isEmptyJSON(existing.mount) && existing.mount != "null" {
		effMount = existing.mount
	}
	effHealthCheck := jsonString(doc.HealthCheck)
	if isEmptyJSON(effHealthCheck) && !isEmptyJSON(existing.healthCheck) && existing.healthCheck != "null" {
		effHealthCheck = existing.healthCheck
	}

	_, err := h.DB.Exec(`INSERT INTO backend_runtimes
		(id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, image_name, image_pull_policy, entrypoint_override_json, args_override_json, default_env_json, docker_json, model_mount_json, health_check_override_json, is_builtin, is_editable, tenant_id, slug, managed_by, source, catalog_version, checksum, status, verification_json, hardware_family, accelerator_api, runtime_distribution, runtime_distribution_version, compatibility_json, image_candidates_json, image_note, devices_json, volumes_json, env_schema_json, args_schema_json, ports_json, high_risk_flags_json, config_hash, loaded_from, loaded_at, created_at, updated_at)
		VALUES (?,?,?,?,?,'',?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			display_name=excluded.display_name,
			backend_id=excluded.backend_id,
			backend_version_id=excluded.backend_version_id,
			vendor=excluded.vendor,
			runtime_type=excluded.runtime_type,
			image_name=excluded.image_name,
			image_pull_policy=excluded.image_pull_policy,
			entrypoint_override_json=excluded.entrypoint_override_json,
			args_override_json=excluded.args_override_json,
			default_env_json=excluded.default_env_json,
			docker_json=excluded.docker_json,
			model_mount_json=excluded.model_mount_json,
			health_check_override_json=excluded.health_check_override_json,
			is_builtin=excluded.is_builtin,
			is_editable=excluded.is_editable,
			slug=excluded.slug,
			managed_by=excluded.managed_by,
			source=excluded.source,
			catalog_version=excluded.catalog_version,
			checksum=excluded.checksum,
			status=excluded.status,
			verification_json=excluded.verification_json,
			hardware_family=excluded.hardware_family,
			accelerator_api=excluded.accelerator_api,
			runtime_distribution=excluded.runtime_distribution,
			runtime_distribution_version=excluded.runtime_distribution_version,
			compatibility_json=excluded.compatibility_json,
			image_candidates_json=excluded.image_candidates_json,
			image_note=excluded.image_note,
			devices_json=excluded.devices_json,
			volumes_json=excluded.volumes_json,
			env_schema_json=excluded.env_schema_json,
			args_schema_json=excluded.args_schema_json,
			ports_json=excluded.ports_json,
			high_risk_flags_json=excluded.high_risk_flags_json,
			config_hash=excluded.config_hash,
			loaded_from=excluded.loaded_from,
			loaded_at=excluded.loaded_at,
			updated_at=excluded.updated_at`,
		doc.ID, name, displayName, doc.BackendID, doc.BackendVersionID, vendor, runnerType, effectiveImage, "if_not_present",
		effEntrypoint, effArgs, effEnv, effDocker, effMount, effHealthCheck,
		isBuiltin, isEditable, "", slug, source, source, "v1", checksumString(doc.ID+doc.BackendID+doc.BackendVersionID), "active", jsonString(doc.Verification),
		doc.HardwareFamily, doc.AcceleratorAPI, doc.RuntimeDistribution, doc.RuntimeDistributionVersion,
		jsonString(doc.Compatibility), jsonString(doc.ImageCandidates), doc.ImageNote,
		jsonString(doc.Devices), jsonString(doc.Volumes), jsonString(doc.EnvSchema),
		jsonString(doc.ArgsSchema), jsonString(doc.Ports), jsonString(doc.HighRiskFlags),
		configHash, loadedFrom, now, now, now)
	if err != nil {
		return fmt.Errorf("upsert runtime %s: %w", doc.ID, err)
	}
	return nil
}

type backendCatalogDoc struct {
	ID             string      `yaml:"id"`
	Name           string      `yaml:"name"`
	DisplayName    string      `yaml:"display_name,omitempty"`
	Description    string      `yaml:"description,omitempty"`
	ProtocolFamily interface{} `yaml:"protocol_family,omitempty"`
	Protocols      interface{} `yaml:"protocols,omitempty"`
	ManagedBy      string      `yaml:"managed_by,omitempty"`
	Source         string      `yaml:"source,omitempty"`
	Readonly       bool        `yaml:"readonly"`
	Slug           string      `yaml:"slug,omitempty"`
	Revision       string      `yaml:"revision,omitempty"`
	ConfigHash     string      `yaml:"config_hash,omitempty"`
}

type backendRuntimeCatalogDoc struct {
	ID                           string      `yaml:"id"`
	Name                         string      `yaml:"name"`
	DisplayName                  string      `yaml:"display_name,omitempty"`
	BackendID                    string      `yaml:"backend_id"`
	BackendVersionID             string      `yaml:"backend_version_id"`
	Source                       string      `yaml:"source,omitempty"`
	ManagedBy                    string      `yaml:"managed_by,omitempty"`
	Readonly                     bool        `yaml:"readonly"`
	Slug                         string      `yaml:"slug,omitempty"`
	Vendor                       string      `yaml:"vendor,omitempty"`
	HardwareFamily               string      `yaml:"hardware_family,omitempty"`
	AcceleratorAPI               string      `yaml:"accelerator_api,omitempty"`
	RuntimeDistribution          string      `yaml:"runtime_distribution,omitempty"`
	RuntimeDistributionVersion   string      `yaml:"runtime_distribution_version,omitempty"`
	Compatibility                interface{} `yaml:"compatibility,omitempty"`
	ImageCandidates              interface{} `yaml:"image_candidates,omitempty"`
	ImageNote                    string      `yaml:"image_note,omitempty"`
	RunnerType                   string      `yaml:"runner_type,omitempty"`
	DockerOptions                interface{} `yaml:"docker_options,omitempty"`
	Devices                      interface{} `yaml:"devices,omitempty"`
	Volumes                      interface{} `yaml:"volumes,omitempty"`
	EnvSchema                    interface{} `yaml:"env_schema,omitempty"`
	Entrypoint                   interface{} `yaml:"entrypoint,omitempty"`
	Args                         interface{} `yaml:"args,omitempty"`
	ArgsSchema                   interface{} `yaml:"args_schema,omitempty"`
	ArgsDefaults                 interface{} `yaml:"args_defaults,omitempty"`
	Ports                        interface{} `yaml:"ports,omitempty"`
	HealthCheck                  interface{} `yaml:"health_check,omitempty"`
	HighRiskFlags                interface{} `yaml:"high_risk_flags,omitempty"`
	ModelMount                   interface{} `yaml:"model_mount,omitempty"`
	ImageRef                     string      `yaml:"image_ref,omitempty"`
	Verification                 interface{} `yaml:"verification,omitempty"`
	SourceBackendVersionRevision string      `yaml:"source_backend_version_revision,omitempty"`
	Revision                     string      `yaml:"revision,omitempty"`
	ConfigHash                   string      `yaml:"config_hash,omitempty"`
}

// reloadAllCatalogs reloads Backend, BackendVersion, and BackendRuntime
// from system and user catalog files into DB projection tables.
func (h *AgentHandler) reloadAllCatalogs() (map[string]int, error) {
	result := make(map[string]int)
	backends, err := h.reloadBackendCatalogs()
	if err != nil {
		return result, err
	}
	result["backends"] = backends
	versions, err := h.reloadBackendVersionCatalogs()
	if err != nil {
		return result, err
	}
	result["versions"] = versions
	runtimes, err := h.reloadBackendRuntimeCatalogs()
	if err != nil {
		return result, err
	}
	result["runtimes"] = runtimes
	return result, nil
}

func (h *AgentHandler) upsertBackendVersionFromRequest(id, backendID string, req map[string]interface{}, creating bool) error {
	if backendID == "" {
		return fmt.Errorf("backend_id is required")
	}
	var exists int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM inference_backends WHERE id=?`, backendID).Scan(&exists); err != nil || exists == 0 {
		return fmt.Errorf("backend not found")
	}
	now := time.Now().Format(time.RFC3339)
	current := h.getBackendVersionJSON(id)
	get := func(key, def string) string {
		if v, ok := req[key]; ok {
			if strings.HasSuffix(key, "_json") || key == "capabilities_json" || key == "docker_options_json" || key == "model_mount_json" || key == "vendor_options_json" {
				return jsonString(v)
			}
			if s, ok := v.(string); ok {
				return s
			}
		}
		if current != nil {
			if v, ok := current[key]; ok {
				if strings.HasSuffix(key, "_json") || key == "capabilities_json" || key == "docker_options_json" || key == "model_mount_json" || key == "vendor_options_json" {
					return rawJSONString(v, def)
				}
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
		return def
	}
	version := get("version", "")
	if version == "" {
		return fmt.Errorf("version is required")
	}
	displayName := get("display_name", version)
	defaultPort := intVal(req, "default_container_port", 8000)
	if current != nil {
		defaultPort = intVal(current, "default_container_port", defaultPort)
	}
	if v, ok := req["default_container_port"]; ok {
		switch n := v.(type) {
		case float64:
			defaultPort = int(n)
		case int:
			defaultPort = n
		}
	}
	isDefault := boolInt(boolVal(req, "is_default", false))
	if current != nil {
		if b, ok := current["is_default"].(bool); ok && b {
			isDefault = 1
		}
	}
	isDeprecated := boolInt(boolVal(req, "is_deprecated", false))
	if current != nil {
		if b, ok := current["is_deprecated"].(bool); ok && b {
			isDeprecated = 1
		}
	}
	slug := get("slug", slugify(version))
	checksum := checksumString(id + version + get("default_args_json", "[]") + get("default_images_json", "{}") + now)
	if creating || current == nil {
		_, err := h.DB.Exec(`INSERT INTO backend_versions
			(id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated, slug, managed_by, source, catalog_version, checksum, status, description, capabilities_json, docker_options_json, model_mount_json, vendor_options_json, readonly, protocol, image_candidates_json, default_host, default_endpoints_json, default_args_schema_json, default_env_schema_json, default_health_check_json, official_reference_json, revision, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, backendID, version, displayName, isDefault, get("default_entrypoint_json", "[]"), get("default_args_json", "[]"), get("default_backend_params_json", "[]"),
			get("parameter_defs_json", "[]"), get("health_check_json", "{}"), defaultPort, get("default_images_json", "{}"), get("env_json", "{}"), isDeprecated,
			slug, "user", "user-catalog", "user", checksum, "active", get("description", ""), get("capabilities_json", "{}"), get("docker_options_json", "{}"),
			get("model_mount_json", "{}"), get("vendor_options_json", "{}"), 0, get("protocol", ""), get("image_candidates_json", "[]"), get("default_host", "0.0.0.0"),
			get("default_endpoints_json", "{}"), get("default_args_schema_json", get("parameter_defs_json", "[]")), get("default_env_schema_json", "[]"),
			get("default_health_check_json", get("health_check_json", "{}")), get("official_reference_json", "[]"), get("revision", checksum), now, now)
		return err
	}
	_, err := h.DB.Exec(`UPDATE backend_versions SET
		version=?, display_name=?, is_default=?, default_entrypoint_json=?, default_args_json=?, default_backend_params_json=?,
		parameter_defs_json=?, health_check_json=?, default_container_port=?, default_images_json=?, env_json=?, is_deprecated=?,
		slug=?, managed_by='user', source='user-catalog', catalog_version='user', checksum=?, status='active',
		description=?, capabilities_json=?, docker_options_json=?, model_mount_json=?, vendor_options_json=?,
		readonly=0, protocol=?, image_candidates_json=?, default_host=?, default_endpoints_json=?,
		default_args_schema_json=?, default_env_schema_json=?, default_health_check_json=?, official_reference_json=?, revision=?, updated_at=?
		WHERE id=?`,
		version, displayName, isDefault, get("default_entrypoint_json", "[]"), get("default_args_json", "[]"), get("default_backend_params_json", "[]"),
		get("parameter_defs_json", "[]"), get("health_check_json", "{}"), defaultPort, get("default_images_json", "{}"), get("env_json", "{}"), isDeprecated,
		slug, checksum, get("description", ""), get("capabilities_json", "{}"), get("docker_options_json", "{}"), get("model_mount_json", "{}"),
		get("vendor_options_json", "{}"), get("protocol", ""), get("image_candidates_json", "[]"), get("default_host", "0.0.0.0"),
		get("default_endpoints_json", "{}"), get("default_args_schema_json", get("parameter_defs_json", "[]")), get("default_env_schema_json", "[]"),
		get("default_health_check_json", get("health_check_json", "{}")), get("official_reference_json", "[]"), get("revision", checksum), now, id)
	return err
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
	return `SELECT id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated, slug, managed_by, source, catalog_version, checksum, status, description, capabilities_json, docker_options_json, model_mount_json, vendor_options_json, readonly, protocol, image_candidates_json, default_host, default_endpoints_json, default_args_schema_json, default_env_schema_json, default_health_check_json, official_reference_json, revision, config_hash, loaded_from, loaded_at, created_at, updated_at FROM backend_versions`
}

func scanBackendVersionMap(scanner backendVersionScanner) (map[string]interface{}, error) {
	var id, bid, ver, dn, dep, dargs, dbp, pdefs, hc, dimg, env, slug, managedBy, source, catalogVersion, checksum, status, desc, caps, dockerOpts, mount, vendorOpts, protocol, imageCandidates, defaultHost, endpoints, argsSchema, envSchema, defaultHC, refs, revision, configHash, loadedFrom, loadedAt, ca, ua string
	var isDef, isDep, readonly int
	var dcp int
	if err := scanner.Scan(&id, &bid, &ver, &dn, &isDef, &dep, &dargs, &dbp, &pdefs, &hc, &dcp, &dimg, &env, &isDep, &slug, &managedBy, &source, &catalogVersion, &checksum, &status, &desc, &caps, &dockerOpts, &mount, &vendorOpts, &readonly, &protocol, &imageCandidates, &defaultHost, &endpoints, &argsSchema, &envSchema, &defaultHC, &refs, &revision, &configHash, &loadedFrom, &loadedAt, &ca, &ua); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": id, "backend_id": bid, "version": ver, "display_name": dn,
		"is_default": isDef == 1, "default_entrypoint_json": json.RawMessage(dep),
		"default_args_json": json.RawMessage(dargs), "default_backend_params_json": json.RawMessage(dbp),
		"parameter_defs_json": json.RawMessage(pdefs), "health_check_json": json.RawMessage(hc),
		"default_container_port": dcp, "default_images_json": json.RawMessage(dimg),
		"env_json": redactRawJSON(env), "is_deprecated": isDep == 1,
		"slug": slug, "managed_by": managedBy, "source": source, "catalog_version": catalogVersion,
		"checksum": checksum, "status": status, "description": desc,
		"capabilities_json": json.RawMessage(caps), "docker_options_json": json.RawMessage(dockerOpts),
		"model_mount_json": json.RawMessage(mount), "vendor_options_json": json.RawMessage(vendorOpts),
		"readonly": readonly == 1, "protocol": protocol, "image_candidates_json": json.RawMessage(imageCandidates),
		"default_host": defaultHost, "default_endpoints_json": json.RawMessage(endpoints),
		"default_args_schema_json": json.RawMessage(argsSchema), "default_env_schema_json": json.RawMessage(envSchema),
		"default_health_check_json": json.RawMessage(defaultHC), "official_reference_json": json.RawMessage(refs), "revision": revision,
		"config_hash": configHash, "loaded_from": loadedFrom, "loaded_at": loadedAt,
		"created_at": ca, "updated_at": ua,
	}, nil
}

func (h *AgentHandler) backendVersionDocFromRequest(id, backendID string, existing map[string]interface{}, req map[string]interface{}) (backendVersionCatalogDoc, error) {
	if backendID == "" {
		return backendVersionCatalogDoc{}, fmt.Errorf("backend_id is required")
	}
	if ok, err := h.backendExists(backendID); err != nil || !ok {
		return backendVersionCatalogDoc{}, fmt.Errorf("backend not found")
	}
	get := func(key string, def interface{}) interface{} {
		if v, ok := req[key]; ok {
			return v
		}
		if existing != nil {
			if v, ok := existing[key]; ok {
				return v
			}
		}
		return def
	}
	version := strings.TrimSpace(fmt.Sprint(get("version", "")))
	if version == "" {
		return backendVersionCatalogDoc{}, fmt.Errorf("version is required")
	}
	displayName := strings.TrimSpace(fmt.Sprint(get("display_name", version)))
	if displayName == "" {
		displayName = version
	}
	imageCandidates := get("image_candidates_json", nil)
	if imageCandidates == nil {
		imageCandidates = get("image_candidates", []interface{}{})
	}
	defaultImages := get("default_images_json", nil)
	if defaultImages == nil {
		defaultImages = get("default_images", map[string]interface{}{})
	}
	modelMount := get("model_mount_json", nil)
	if modelMount == nil {
		modelMount = get("default_model_mount", map[string]interface{}{})
	}
	healthCheck := get("default_health_check_json", nil)
	if healthCheck == nil {
		healthCheck = get("health_check_json", nil)
	}
	if healthCheck == nil {
		healthCheck = get("health_check", map[string]interface{}{})
	}
	defaultArgsSchema := get("default_args_schema_json", nil)
	if defaultArgsSchema == nil {
		defaultArgsSchema = get("parameter_defs_json", nil)
	}
	if defaultArgsSchema == nil {
		defaultArgsSchema = get("default_args_schema", []interface{}{})
	}
	defaultCommand := get("default_command", nil)
	defaultArgs := get("default_args_json", nil)
	if defaultArgs == nil {
		defaultArgs = get("default_args", nil)
	}
	if defaultCommand == nil {
		defaultCommand = defaultArgs
	}
	defaultPort := intFromAny(get("default_port", 0), 0)
	if defaultPort == 0 {
		defaultPort = intFromAny(get("default_container_port", 8000), 8000)
	}
	return backendVersionCatalogDoc{
		ID:                    id,
		BackendID:             backendID,
		Slug:                  strings.TrimSpace(fmt.Sprint(get("slug", slugify(version)))),
		Version:               version,
		DisplayName:           displayName,
		Source:                "user",
		ManagedBy:             "user",
		Readonly:              false,
		Protocol:              strings.TrimSpace(fmt.Sprint(get("protocol", ""))),
		ImageCandidates:       jsonToAny(rawJSONString(imageCandidates, "[]")),
		DefaultPort:           defaultPort,
		DefaultHost:           strings.TrimSpace(fmt.Sprint(get("default_host", "0.0.0.0"))),
		DefaultModelMount:     jsonToAny(rawJSONString(modelMount, "{}")),
		DefaultEndpoints:      jsonToAny(rawJSONString(get("default_endpoints_json", get("default_endpoints", map[string]interface{}{})), "{}")),
		Capabilities:          jsonToAny(rawJSONString(get("capabilities_json", get("capabilities", []interface{}{})), "[]")),
		DefaultEntrypoint:     jsonToAny(rawJSONString(get("default_entrypoint_json", get("entrypoint", get("default_entrypoint", []interface{}{}))), "[]")),
		Entrypoint:            jsonToAny(rawJSONString(get("entrypoint", get("default_entrypoint_json", []interface{}{})), "[]")),
		DefaultCommand:        jsonToAny(rawJSONString(defaultCommand, "[]")),
		DefaultArgs:           jsonToAny(rawJSONString(defaultArgs, "[]")),
		DefaultArgsSchema:     jsonToAny(rawJSONString(defaultArgsSchema, "[]")),
		DefaultEnvSchema:      jsonToAny(rawJSONString(get("default_env_schema_json", get("default_env_schema", []interface{}{})), "[]")),
		HealthCheck:           jsonToAny(rawJSONString(healthCheck, "{}")),
		OfficialReferenceNote: jsonToAny(rawJSONString(get("official_reference_json", get("official_reference_note", []interface{}{})), "[]")),
		Description:           strings.TrimSpace(fmt.Sprint(get("description", ""))),
		DefaultImages:         jsonToAny(rawJSONString(defaultImages, "{}")),
		Env:                   jsonToAny(rawJSONString(get("env_json", get("env", map[string]interface{}{})), "{}")),
		DockerOptions:         jsonToAny(rawJSONString(get("docker_options_json", get("docker_options", map[string]interface{}{})), "{}")),
		VendorOptions:         jsonToAny(rawJSONString(get("vendor_options_json", get("vendor_options", map[string]interface{}{})), "{}")),
		DefaultBackendParams:  jsonToAny(rawJSONString(get("default_backend_params_json", get("default_backend_params", []interface{}{})), "[]")),
		Revision:              strings.TrimSpace(fmt.Sprint(get("revision", ""))),
	}, nil
}

func (h *AgentHandler) backendExists(backendID string) (bool, error) {
	var exists int
	err := h.DB.QueryRow(`SELECT COUNT(*) FROM inference_backends WHERE id=?`, backendID).Scan(&exists)
	return exists > 0, err
}

func (h *AgentHandler) writeUserBackendVersionCatalogDoc(doc backendVersionCatalogDoc) error {
	backendSlug, err := h.backendCatalogSlug(doc.BackendID)
	if err != nil {
		return err
	}
	if doc.Slug == "" {
		doc.Slug = slugify(doc.Version)
	}
	doc.Source = "user"
	doc.ManagedBy = "user"
	doc.Readonly = false
	dir := filepath.Join(backendCatalogUserVersionsDir, backendSlug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, doc.Slug+".yaml"), data, 0644)
}

func (h *AgentHandler) backendCatalogSlug(backendID string) (string, error) {
	var name string
	if err := h.DB.QueryRow(`SELECT name FROM inference_backends WHERE id=?`, backendID).Scan(&name); err != nil {
		return "", err
	}
	return slugify(name), nil
}

func (h *AgentHandler) reloadBackendVersionCatalogs() (int, error) {
	count := 0
	if n, err := h.reloadBackendVersionCatalogDir(backendCatalogSystemVersionsDir, "system"); err != nil {
		return count, err
	} else {
		count += n
	}
	if n, err := h.reloadBackendVersionCatalogDir(backendCatalogUserVersionsDir, "user"); err != nil {
		return count, err
	} else {
		count += n
	}
	return count, nil
}

func (h *AgentHandler) reloadBackendVersionCatalogDir(root, source string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() {
			return err
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc backendVersionCatalogDoc
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if err := h.upsertBackendVersionProjection(doc, source, path, data); err != nil {
			return err
		}
		count++
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return count, err
}

func (h *AgentHandler) upsertBackendVersionProjection(doc backendVersionCatalogDoc, source, loadedFrom string, data []byte) error {
	if doc.ID == "" || doc.Version == "" {
		return fmt.Errorf("invalid backend version catalog file %s", loadedFrom)
	}
	backendID := doc.BackendID
	if backendID == "" && doc.Backend != "" {
		if err := h.DB.QueryRow(`SELECT id FROM inference_backends WHERE name=? OR display_name=?`, doc.Backend, doc.Backend).Scan(&backendID); err != nil {
			return fmt.Errorf("backend %q not found for %s", doc.Backend, loadedFrom)
		}
	}
	if ok, err := h.backendExists(backendID); err != nil || !ok {
		return fmt.Errorf("backend %q not found for %s", backendID, loadedFrom)
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = strings.TrimSpace(doc.Source)
	}
	if source == "" {
		source = strings.TrimSpace(doc.ManagedBy)
	}
	if source == "" {
		source = "user"
	}
	readonly := source == "system"
	if source != "system" && doc.Readonly {
		readonly = true
	}
	defaultPort := doc.DefaultPort
	if defaultPort == 0 {
		defaultPort = doc.DefaultContainerPort
	}
	if defaultPort == 0 {
		defaultPort = 8000
	}
	imageCandidates := doc.ImageCandidates
	if imageCandidates == nil {
		imageCandidates = []interface{}{}
	}
	defaultImages := doc.DefaultImages
	if isEmptyJSON(defaultImages) {
		if first := firstStringFromAny(imageCandidates); first != "" {
			defaultImages = map[string]interface{}{"default": first}
		} else {
			defaultImages = map[string]interface{}{}
		}
	}
	defaultArgs := doc.DefaultCommand
	if defaultArgs == nil {
		defaultArgs = doc.DefaultArgs
	}
	defaultEntrypoint := doc.DefaultEntrypoint
	if defaultEntrypoint == nil {
		defaultEntrypoint = doc.Entrypoint
	}
	modelMount := doc.DefaultModelMount
	if modelMount == nil {
		modelMount = doc.ModelMount
	}
	argsSchema := doc.DefaultArgsSchema
	if argsSchema == nil {
		argsSchema = doc.ParameterDefs
	}
	healthCheck := doc.HealthCheck
	refs := doc.OfficialReferenceNote
	now := time.Now().Format(time.RFC3339)
	configHash := checksumString(string(data))
	revision := doc.Revision
	if revision == "" {
		revision = configHash
	}
	slug := doc.Slug
	if slug == "" {
		slug = slugify(doc.Version)
	}
	managedBy := source
	catalogVersion := source
	status := "active"
	isDeprecated := 0
	if err := h.clearConflictingBackendVersionProjection(doc.ID, backendID, doc.Version, loadedFrom); err != nil {
		return err
	}
	_, err := h.DB.Exec(`INSERT INTO backend_versions
		(id, backend_id, version, display_name, is_default, default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json, is_deprecated, slug, managed_by, source, catalog_version, checksum, status, description, capabilities_json, docker_options_json, model_mount_json, vendor_options_json, readonly, protocol, image_candidates_json, default_host, default_endpoints_json, default_args_schema_json, default_env_schema_json, default_health_check_json, official_reference_json, revision, config_hash, loaded_from, loaded_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			backend_id=excluded.backend_id,
			version=excluded.version,
			display_name=excluded.display_name,
			is_default=excluded.is_default,
			default_entrypoint_json=excluded.default_entrypoint_json,
			default_args_json=excluded.default_args_json,
			default_backend_params_json=excluded.default_backend_params_json,
			parameter_defs_json=excluded.parameter_defs_json,
			health_check_json=excluded.health_check_json,
			default_container_port=excluded.default_container_port,
			default_images_json=excluded.default_images_json,
			env_json=excluded.env_json,
			is_deprecated=excluded.is_deprecated,
			slug=excluded.slug,
			managed_by=excluded.managed_by,
			source=excluded.source,
			catalog_version=excluded.catalog_version,
			checksum=excluded.checksum,
			status=excluded.status,
			description=excluded.description,
			capabilities_json=excluded.capabilities_json,
			docker_options_json=excluded.docker_options_json,
			model_mount_json=excluded.model_mount_json,
			vendor_options_json=excluded.vendor_options_json,
			readonly=excluded.readonly,
			protocol=excluded.protocol,
			image_candidates_json=excluded.image_candidates_json,
			default_host=excluded.default_host,
			default_endpoints_json=excluded.default_endpoints_json,
			default_args_schema_json=excluded.default_args_schema_json,
			default_env_schema_json=excluded.default_env_schema_json,
			default_health_check_json=excluded.default_health_check_json,
			official_reference_json=excluded.official_reference_json,
			revision=excluded.revision,
			config_hash=excluded.config_hash,
			loaded_from=excluded.loaded_from,
			loaded_at=excluded.loaded_at,
			updated_at=excluded.updated_at`,
		doc.ID, backendID, doc.Version, valueOrDefault(doc.DisplayName, doc.Version), 0,
		jsonString(defaultEntrypoint), jsonString(defaultArgs), jsonString(nilToDefault(doc.DefaultBackendParams, []interface{}{})),
		jsonString(nilToDefault(argsSchema, []interface{}{})), jsonString(nilToDefault(healthCheck, map[string]interface{}{})),
		defaultPort, jsonString(nilToDefault(defaultImages, map[string]interface{}{})), jsonString(nilToDefault(doc.Env, map[string]interface{}{})), isDeprecated,
		slug, managedBy, source, catalogVersion, configHash, status, doc.Description,
		jsonString(nilToDefault(firstNonNil(doc.CapabilitiesJSON, doc.Capabilities), []interface{}{})), jsonString(nilToDefault(doc.DockerOptions, map[string]interface{}{})),
		jsonString(nilToDefault(modelMount, map[string]interface{}{})), jsonString(nilToDefault(doc.VendorOptions, map[string]interface{}{})),
		boolInt(readonly), doc.Protocol, jsonString(nilToDefault(imageCandidates, []interface{}{})), valueOrDefault(doc.DefaultHost, "0.0.0.0"),
		jsonString(nilToDefault(doc.DefaultEndpoints, map[string]interface{}{})), jsonString(nilToDefault(argsSchema, []interface{}{})),
		jsonString(nilToDefault(doc.DefaultEnvSchema, []interface{}{})), jsonString(nilToDefault(healthCheck, map[string]interface{}{})),
		jsonString(nilToDefault(refs, []interface{}{})), revision, configHash, loadedFrom, now, now, now)
	return err
}

func (h *AgentHandler) clearConflictingBackendVersionProjection(id, backendID, version, loadedFrom string) error {
	var existingID string
	err := h.DB.QueryRow(`SELECT id FROM backend_versions WHERE backend_id=? AND version=? AND id<>?`, backendID, version, id).Scan(&existingID)
	if err != nil {
		return nil
	}
	var refs int
	_ = h.DB.QueryRow(`SELECT COUNT(*) FROM backend_runtimes WHERE backend_version_id=?`, existingID).Scan(&refs)
	if refs > 0 {
		return fmt.Errorf("backend version catalog %s conflicts with existing referenced projection %s", loadedFrom, existingID)
	}
	_, err = h.DB.Exec(`DELETE FROM backend_versions WHERE id=?`, existingID)
	return err
}

type backendVersionCatalogDoc struct {
	ID                    string      `yaml:"id"`
	BackendID             string      `yaml:"backend_id,omitempty"`
	Backend               string      `yaml:"backend,omitempty"`
	Slug                  string      `yaml:"slug,omitempty"`
	Version               string      `yaml:"version"`
	DisplayName           string      `yaml:"display_name,omitempty"`
	ManagedBy             string      `yaml:"managed_by,omitempty"`
	Source                string      `yaml:"source,omitempty"`
	Readonly              bool        `yaml:"readonly"`
	Protocol              string      `yaml:"protocol,omitempty"`
	ImageCandidates       interface{} `yaml:"image_candidates,omitempty"`
	DefaultPort           int         `yaml:"default_port,omitempty"`
	DefaultContainerPort  int         `yaml:"default_container_port,omitempty"`
	DefaultHost           string      `yaml:"default_host,omitempty"`
	DefaultModelMount     interface{} `yaml:"default_model_mount,omitempty"`
	DefaultEndpoints      interface{} `yaml:"default_endpoints,omitempty"`
	Capabilities          interface{} `yaml:"capabilities,omitempty"`
	CapabilitiesJSON      interface{} `yaml:"capabilities_json,omitempty"`
	Entrypoint            interface{} `yaml:"entrypoint,omitempty"`
	DefaultEntrypoint     interface{} `yaml:"default_entrypoint,omitempty"`
	DefaultCommand        interface{} `yaml:"default_command,omitempty"`
	DefaultArgs           interface{} `yaml:"default_args,omitempty"`
	DefaultArgsSchema     interface{} `yaml:"default_args_schema,omitempty"`
	DefaultEnvSchema      interface{} `yaml:"default_env_schema,omitempty"`
	HealthCheck           interface{} `yaml:"health_check,omitempty"`
	OfficialReferenceNote interface{} `yaml:"official_reference_note,omitempty"`
	Revision              string      `yaml:"revision,omitempty"`
	Description           string      `yaml:"description,omitempty"`
	DefaultBackendParams  interface{} `yaml:"default_backend_params,omitempty"`
	ParameterDefs         interface{} `yaml:"parameter_defs,omitempty"`
	DefaultImages         interface{} `yaml:"default_images,omitempty"`
	Env                   interface{} `yaml:"env,omitempty"`
	DockerOptions         interface{} `yaml:"docker_options,omitempty"`
	ModelMount            interface{} `yaml:"model_mount,omitempty"`
	VendorOptions         interface{} `yaml:"vendor_options,omitempty"`
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
