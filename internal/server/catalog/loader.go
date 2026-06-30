package catalog

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func LoadRegistry(dir string) (*Registry, error) {
	if dir == "" {
		var err error
		dir, err = findRepoPath("configs/config-registry")
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "items.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read config registry: %w", err)
	}
	var registry Registry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parse config registry: %w", err)
	}
	if err := ValidateRegistry(&registry); err != nil {
		return nil, err
	}
	return &registry, nil
}

func LoadBackendCatalog(root string) (*BackendCatalog, error) {
	if root == "" {
		var err error
		root, err = findRepoPath("configs/backend-catalog")
		if err != nil {
			return nil, err
		}
	}
	catalog := &BackendCatalog{Root: root}
	if err := loadDocs(filepath.Join(root, "backends", "*.yaml"), &catalog.Backends); err != nil {
		return nil, err
	}
	if err := loadDocs(filepath.Join(root, "versions", "*", "*.yaml"), &catalog.Versions); err != nil {
		return nil, err
	}
	if err := loadDocs(filepath.Join(root, "runtimes", "*", "*.yaml"), &catalog.Runtimes); err != nil {
		return nil, err
	}
	if err := ValidateCatalog(catalog); err != nil {
		return nil, err
	}
	return catalog, nil
}

func ValidateRegistry(registry *Registry) error {
	if registry == nil {
		return fmt.Errorf("config registry is nil")
	}
	allowedCategory := map[string]bool{"launcher": true, "runtime_env": true, "model_runtime": true}
	allowedKind := map[string]bool{"cli_arg": true, "cli_args": true, "env": true, "env_lines": true, "port": true, "volume": true, "device": true, "health_check": true, "launcher_option": true}
	allowedType := map[string]bool{"string": true, "integer": true, "number": true, "boolean": true, "array": true, "object": true, "lines": true}
	allowedSupport := map[string]bool{"verified": true, "documented": true, "experimental": true}
	registry.byCode = map[string]RegistryItem{}
	for _, item := range registry.Items {
		if item.Code == "" {
			return fmt.Errorf("config registry item has empty code")
		}
		if registry.byCode[item.Code].Code != "" {
			return fmt.Errorf("duplicate config item code %q", item.Code)
		}
		if !allowedCategory[item.Category] {
			return fmt.Errorf("config item %s has invalid category %q", item.Code, item.Category)
		}
		if !allowedKind[item.Kind] {
			return fmt.Errorf("config item %s has invalid kind %q", item.Code, item.Kind)
		}
		if !allowedType[item.Type] {
			return fmt.Errorf("config item %s has invalid type %q", item.Code, item.Type)
		}
		if item.SupportLevel == "" {
			item.SupportLevel = "documented"
		}
		if !allowedSupport[item.SupportLevel] {
			return fmt.Errorf("config item %s has invalid support_level %q", item.Code, item.SupportLevel)
		}
		registry.byCode[item.Code] = item
	}
	return nil
}

func ValidateCatalog(catalog *BackendCatalog) error {
	if catalog == nil {
		return fmt.Errorf("backend catalog is nil")
	}
	backendIDs := map[string]bool{}
	for _, backend := range catalog.Backends {
		if backend.ID == "" {
			return fmt.Errorf("backend catalog has backend with empty id")
		}
		if backendIDs[backend.ID] {
			return fmt.Errorf("duplicate backend id %q", backend.ID)
		}
		backendIDs[backend.ID] = true
	}
	versionIDs := map[string]bool{}
	for _, version := range catalog.Versions {
		if version.ID == "" || version.BackendID == "" {
			return fmt.Errorf("backend version has empty id/backend_id")
		}
		if !backendIDs[version.BackendID] {
			return fmt.Errorf("backend version %s references unknown backend %s", version.ID, version.BackendID)
		}
		if versionIDs[version.ID] {
			return fmt.Errorf("duplicate backend version id %q", version.ID)
		}
		versionIDs[version.ID] = true
	}
	runtimeIDs := map[string]bool{}
	visibleRuntimeKeys := map[string]string{}
	for _, runtime := range catalog.Runtimes {
		if runtime.ID == "" || runtime.BackendID == "" || runtime.BackendVersionID == "" {
			return fmt.Errorf("backend runtime has empty id/backend_id/backend_version_id")
		}
		if !backendIDs[runtime.BackendID] {
			return fmt.Errorf("backend runtime %s references unknown backend %s", runtime.ID, runtime.BackendID)
		}
		if !versionIDs[runtime.BackendVersionID] {
			return fmt.Errorf("backend runtime %s references unknown backend version %s", runtime.ID, runtime.BackendVersionID)
		}
		if runtimeIDs[runtime.ID] {
			return fmt.Errorf("duplicate backend runtime id %q", runtime.ID)
		}
		runtimeIDs[runtime.ID] = true
		if runtimeVisibility(runtime) == "visible" {
			status := runtimeStatus(runtime)
			if status == "active" || status == "experimental" {
				key := runtime.Vendor + "|" + runtime.BackendID + "|" + runtime.BackendVersionID
				if existing := visibleRuntimeKeys[key]; existing != "" {
					return fmt.Errorf("duplicate visible backend runtime for %s: %s and %s", key, existing, runtime.ID)
				}
				visibleRuntimeKeys[key] = runtime.ID
			}
		}
	}
	return nil
}

func MaterializeBackend(registry *Registry, backend BackendDoc) ConfigSet {
	items := registry.MaterializeBase("Backend", backend.ID)
	setItemTiered(items, "runtime.health", backend.DefaultHealthCheck, backend.DefaultHealthCheck, true, "Backend", backend.ID)
	addDynamicTiered(items, "backend.capabilities", "model_runtime", "object", map[string]any{
		"supported_formats": backend.SupportedModelFormats,
		"protocols":         backend.Protocols,
	}, "Backend", backend.ID, 5)
	return ConfigSet{
		SchemaVersion: 1,
		ConfigSetKey:  "BackendConfigSet",
		Context: map[string]string{
			"backend":       backend.Slug,
			"backend_id":    backend.ID,
			"launcher_kind": "docker",
		},
		Items:          items,
		SourceMetadata: sourceMetadata(backend.SourcePath, backend.SourceHash, "", "backend"),
	}
}

func MaterializeBackendVersion(registry *Registry, backend BackendDoc, version VersionDoc) ConfigSet {
	items := registry.MaterializeBase("BackendVersion", version.ID)
	args := version.DefaultArgs
	if len(args) == 0 {
		args = version.DefaultCommand
	}
	setItemTiered(items, "launcher.entrypoint", version.DefaultEntrypoint, version.DefaultEntrypoint, len(version.DefaultEntrypoint) > 0, "BackendVersion", version.ID)
	setItemTiered(items, "launcher.command", args, args, len(args) > 0, "BackendVersion", version.ID)
	setItemTiered(items, "service.listen_host", nonEmpty(version.DefaultHost, "0.0.0.0"), nonEmpty(version.DefaultHost, "0.0.0.0"), true, "BackendVersion", version.ID)
	setItemTiered(items, "service.container_port", nonZero(version.DefaultPort, 8000), nonZero(version.DefaultPort, 8000), true, "BackendVersion", version.ID)
	setItemTiered(items, "runtime.model_mount", version.DefaultModelMount, version.DefaultModelMount, len(version.DefaultModelMount) > 0, "BackendVersion", version.ID)
	setItemTiered(items, "runtime.health", version.HealthCheck, version.HealthCheck, len(version.HealthCheck) > 0, "BackendVersion", version.ID)
	addDynamicTiered(items, "backend.capabilities", "model_runtime", "object", normalizedCapabilities(backend, version), "BackendVersion", version.ID, 5)
	addDynamicTiered(items, "backend.supported_config_items", "model_runtime", "array", configCodesFromArgs(version.DefaultArgsSchema), "BackendVersion", version.ID, 6)
	addArgConfigItemsTiered(items, version.DefaultArgsSchema, "BackendVersion", version.ID)
	return ConfigSet{
		SchemaVersion: 1,
		ConfigSetKey:  "BackendVersionConfigSet",
		Context: map[string]string{
			"backend":         backend.Slug,
			"backend_id":      version.BackendID,
			"backend_version": version.ID,
			"launcher_kind":   "docker",
		},
		Items:          items,
		SourceMetadata: sourceMetadata(version.SourcePath, version.SourceHash, "backend:"+version.BackendID, "backend_version"),
	}
}

func addArgConfigItemsTiered(items map[string]ConfigItem, args []map[string]any, layer, ref string) {
	for idx, arg := range args {
		name := strings.TrimSpace(fmt.Sprint(arg["name"]))
		if name == "" || name == "{{MODEL_CONTAINER_PATH}}" {
			continue
		}
		code := configCodeFromArgName(name)
		if code == "" {
			continue
		}
		typ := strings.TrimSpace(fmt.Sprint(arg["type"]))
		if typ == "" || typ == "<nil>" {
			typ = "string"
		}
		defaultValue := arg["default"]
		if defaultValue == nil {
			defaultValue = arg["value"]
		}
		required := boolFromAny(arg["required"])
		enabled := required
		item := ConfigItem{
			Schema: ConfigItemSchema{
				Key: code, Category: "model_runtime", Kind: "cli_arg", Type: normalizeConfigType(typ),
				Required: required, SupportLevel: "documented", DisplayOrder: 300 + idx,
				Target:             "cli",
				ArgName:            name,
				LabelI18nKey:       stringFromMap(arg, "label_i18n_key"),
				DescriptionI18nKey: stringFromMap(arg, "description_i18n_key"),
				HelpI18nKey:        stringFromMap(arg, "help_i18n_key"),
				TooltipI18nKey:     stringFromMap(arg, "tooltip_i18n_key"),
			},
			Value_: ConfigItemValue{
				DefaultValue: defaultValue, EffectiveValue: defaultValue,
			},
			State_: ConfigItemState{Enabled: enabled, Checked: enabled, Editable: true, Visible: true, Valid: true},
			Provenance_: ConfigItemProvenance{
				ValueSource: layer, LastValueLayer: layer, LastValueOwnerID: ref,
			},
			Presentation: ConfigItemPresentation{Priority: 300 + idx},
		}
		if label := strings.TrimSpace(fmt.Sprint(arg["label"])); label != "" && label != "<nil>" {
			item.Schema.Label = label
			item.Presentation.Group = strings.TrimSpace(fmt.Sprint(arg["group"]))
		}
		if help := firstArgString(arg, "help", "description", "tooltip"); help != "" {
			item.Schema.HelpText = help
		}
		if description := firstArgString(arg, "description", "help"); description != "" {
			item.Schema.Description = description
		}
		if options := argOptions(arg); len(options) > 0 {
			item.Schema.Choices = options
			item.Schema.Constraints = map[string]any{"options": options}
		}
		items[code] = item
	}
}

func MaterializeBackendRuntime(registry *Registry, versionSet ConfigSet, runtime RuntimeDoc) ConfigSet {
	items := cloneItems(versionSet.Items)
	image := runtime.ImageRef
	if image == "" && len(runtime.ImageCandidates) > 0 {
		image = runtime.ImageCandidates[0]
	}
	setItemTiered(items, "launcher.kind", nonEmpty(runtime.RunnerType, "docker"), nonEmpty(runtime.RunnerType, "docker"), true, "BackendRuntime", runtime.ID)
	setItemTiered(items, "launcher.image", image, image, image != "", "BackendRuntime", runtime.ID)
	setItemTiered(items, "launcher.entrypoint", runtime.Entrypoint, runtime.Entrypoint, len(runtime.Entrypoint) > 0, "BackendRuntime", runtime.ID)
	setItemTiered(items, "launcher.command", runtime.Args, runtime.Args, len(runtime.Args) > 0, "BackendRuntime", runtime.ID)
	dockerOptions := normalizeDockerOptions(runtime)
	runtimeEnv := normalizeRuntimeEnv(runtime.Env, runtime.DockerOptions)
	setItemTiered(items, "launcher.docker_options", dockerOptions, dockerOptions, true, "BackendRuntime", runtime.ID)
	setItemTiered(items, "runtime.env", runtimeEnv, runtimeEnv, len(runtimeEnv) > 0, "BackendRuntime", runtime.ID)
	setItemTiered(items, "runtime.model_mount", runtime.ModelMount, runtime.ModelMount, len(runtime.ModelMount) > 0, "BackendRuntime", runtime.ID)
	setItemTiered(items, "runtime.health", runtime.HealthCheck, runtime.HealthCheck, len(runtime.HealthCheck) > 0, "BackendRuntime", runtime.ID)
	if len(runtime.Ports) > 0 {
		setItemTiered(items, "launcher.ports", runtime.Ports, runtime.Ports, true, "BackendRuntime", runtime.ID)
	}
	return ConfigSet{
		SchemaVersion: 1,
		ConfigSetKey:  "BackendRuntimeConfigSet",
		Context: map[string]string{
			"backend_id":      runtime.BackendID,
			"backend_version": runtime.BackendVersionID,
			"backend_runtime": runtime.ID,
			"launcher_kind":   nonEmpty(runtime.RunnerType, "docker"),
			"vendor":          runtime.Vendor,
		},
		Items:          items,
		SourceMetadata: sourceMetadata(runtime.SourcePath, runtime.SourceHash, "backend_version:"+runtime.BackendVersionID, "backend_runtime"),
	}
}

func SeedCatalog(db *sql.DB, registryDir, catalogRoot string) error {
	registry, err := LoadRegistry(registryDir)
	if err != nil {
		return err
	}
	catalog, err := LoadBackendCatalog(catalogRoot)
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	backendByID := map[string]BackendDoc{}
	for _, backend := range catalog.Backends {
		backendByID[backend.ID] = backend
		configSet, err := MaterializeBackend(registry, backend).JSON()
		if err != nil {
			return err
		}
		sourceMeta := mustJSON(sourceMetadata(backend.SourcePath, backend.SourceHash, "", "backend"))
		if _, err := db.Exec(`INSERT INTO inference_backends
			(id, name, display_name, description, slug, managed_by, source, catalog_version, checksum, status, config_set_json, source_metadata_json, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(id) DO UPDATE SET
				name=excluded.name, display_name=excluded.display_name, description=excluded.description,
				slug=excluded.slug, managed_by=excluded.managed_by, source=excluded.source,
				catalog_version=excluded.catalog_version, checksum=excluded.checksum, status=excluded.status,
				config_set_json=excluded.config_set_json, source_metadata_json=excluded.source_metadata_json,
				updated_at=excluded.updated_at`,
			backend.ID, backend.Slug, backend.Name, backend.Name+" inference backend",
			backend.Slug, "system", "config-registry", "configset-v1", backend.SourceHash, "active", configSet, sourceMeta, now, now); err != nil {
			return fmt.Errorf("seed backend %s: %w", backend.ID, err)
		}
	}

	versionSets := map[string]ConfigSet{}
	for _, version := range catalog.Versions {
		backend := backendByID[version.BackendID]
		set := MaterializeBackendVersion(registry, backend, version)
		versionSets[version.ID] = set
		configSet, err := set.JSON()
		if err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO backend_versions
			(id, backend_id, version, display_name, is_default, is_deprecated, slug, managed_by, source, catalog_version, checksum, status, description, readonly, protocol, revision, config_set_json, source_metadata_json, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(id) DO UPDATE SET
				backend_id=excluded.backend_id, version=excluded.version, display_name=excluded.display_name,
				is_default=excluded.is_default, is_deprecated=0, slug=excluded.slug, managed_by=excluded.managed_by,
				source=excluded.source, catalog_version=excluded.catalog_version, checksum=excluded.checksum,
				status=excluded.status, description=excluded.description, readonly=excluded.readonly, protocol=excluded.protocol, revision=excluded.revision,
				config_set_json=excluded.config_set_json, source_metadata_json=excluded.source_metadata_json,
				updated_at=excluded.updated_at`,
			version.ID, version.BackendID, version.Version, displayName(version.ID, version.Version), defaultVersionFlag(version.ID), 0,
			nonEmpty(version.Slug, version.ID), "system", "config-registry", "configset-v1", version.SourceHash, "active", displayName(version.ID, version.Version)+" system software version", boolInt(version.Readonly), version.Protocol, version.Version, configSet, mustJSON(set.SourceMetadata), now, now); err != nil {
			return fmt.Errorf("seed backend version %s: %w", version.ID, err)
		}
	}

	for _, runtime := range catalog.Runtimes {
		versionSet, ok := versionSets[runtime.BackendVersionID]
		if !ok {
			return fmt.Errorf("runtime %s references version without materialized config_set %s", runtime.ID, runtime.BackendVersionID)
		}
		set := MaterializeBackendRuntime(registry, versionSet, runtime)
		configSet, err := set.JSON()
		if err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO backend_runtimes
			(id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, is_builtin, is_editable, tenant_id, slug, managed_by, source, catalog_version, checksum, status, visibility, support_level, verification_json, hardware_family, accelerator_api, runtime_distribution, runtime_distribution_version, config_hash, loaded_from, loaded_at, config_set_json, source_metadata_json, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(id) DO UPDATE SET
				name=excluded.name, display_name=excluded.display_name, backend_id=excluded.backend_id,
				backend_version_id=excluded.backend_version_id, source_template_name=excluded.source_template_name,
				vendor=excluded.vendor, runtime_type=excluded.runtime_type, is_builtin=excluded.is_builtin,
				is_editable=excluded.is_editable, tenant_id=excluded.tenant_id, slug=excluded.slug,
				managed_by=excluded.managed_by, source=excluded.source, catalog_version=excluded.catalog_version,
				checksum=excluded.checksum, status=excluded.status, visibility=excluded.visibility,
				support_level=excluded.support_level, verification_json=excluded.verification_json,
				hardware_family=excluded.hardware_family, accelerator_api=excluded.accelerator_api,
				runtime_distribution=excluded.runtime_distribution, runtime_distribution_version=excluded.runtime_distribution_version,
				config_hash=excluded.config_hash, loaded_from=excluded.loaded_from, loaded_at=excluded.loaded_at,
				config_set_json=excluded.config_set_json, source_metadata_json=excluded.source_metadata_json,
				updated_at=excluded.updated_at`,
			runtime.ID, nonEmpty(runtime.Name, runtime.Slug), displayRuntimeName(runtime), runtime.BackendID, runtime.BackendVersionID, runtime.Slug, runtime.Vendor, nonEmpty(runtime.RunnerType, "docker"), 1, 0, "",
			runtime.Slug, "system", "config-registry", "configset-v1", runtime.SourceHash, runtimeStatus(runtime), runtimeVisibility(runtime), runtimeSupportLevel(runtime), mustJSON(runtime.Verification),
			runtime.HardwareFamily, runtime.AcceleratorAPI, runtime.RuntimeDistribution, runtime.RuntimeDistributionVersion, runtime.SourceHash, runtime.SourcePath, now, configSet, mustJSON(set.SourceMetadata), now, now); err != nil {
			return fmt.Errorf("seed backend runtime %s: %w", runtime.ID, err)
		}
	}
	return nil
}

// === Tiered-field helpers (replace old setItem / addDynamic) ===

func setItemTiered(items map[string]ConfigItem, code string, value, defaultValue any, enabled bool, layer, ref string) {
	item := items[code]
	if item.Schema.Key == "" {
		item.Schema.Key = code
		item.Schema.SupportLevel = "documented"
	}
	if value != nil {
		item.Value_.LocalValue = value
		if enabled {
			item.Value_.EffectiveValue = value
		}
	} else if defaultValue != nil {
		item.Value_.EffectiveValue = defaultValue
	}
	item.Value_.DefaultValue = defaultValue
	item.State_.Enabled = enabled
	item.State_.Checked = enabled
	if item.State_.Editable == false && !item.Schema.ReadOnly {
		item.State_.Editable = true
	}
	if item.State_.Visible == false {
		item.State_.Visible = true
	}
	item.State_.Valid = true
	item.Provenance_.LastValueLayer = layer
	item.Provenance_.LastValueOwnerID = ref
	item.Provenance_.ValueSource = layer
	items[code] = item
}

func addDynamicTiered(items map[string]ConfigItem, code, category, typ string, value any, layer, ref string, order int) {
	items[code] = ConfigItem{
		Schema: ConfigItemSchema{
			Key: code, Category: category, Kind: "launcher_option", Type: typ,
			DisplayOrder: order, SupportLevel: "documented",
		},
		Value_: ConfigItemValue{DefaultValue: value, EffectiveValue: value},
		State_: ConfigItemState{Enabled: true, Checked: false, Editable: true, Visible: true, Valid: true},
		Provenance_: ConfigItemProvenance{
			ValueSource: layer, LastValueLayer: layer, LastValueOwnerID: ref,
		},
		Presentation: ConfigItemPresentation{Priority: order},
	}
}

func cloneItems(in map[string]ConfigItem) map[string]ConfigItem {
	out := make(map[string]ConfigItem, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// === Utility functions (unchanged) ===

func loadDocs[T any](pattern string, out *[]T) error {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		var doc T
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", file, err)
		}
		setSource(&doc, file, data)
		*out = append(*out, doc)
	}
	return nil
}

func setSource(doc any, file string, data []byte) {
	sum := sha256.Sum256(data)
	hash := "sha256:" + hex.EncodeToString(sum[:])
	switch v := doc.(type) {
	case *BackendDoc:
		v.SourcePath = file
		v.SourceHash = hash
	case *VersionDoc:
		v.SourcePath = file
		v.SourceHash = hash
	case *RuntimeDoc:
		v.SourcePath = file
		v.SourceHash = hash
	}
}

func findRepoPath(rel string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(wd, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", fmt.Errorf("could not locate %s from current directory", rel)
}

func sourceMetadata(path, hash, parentRef, kind string) map[string]interface{} {
	return map[string]interface{}{
		"source_ref":      path,
		"source_hash":     hash,
		"parent_ref":      parentRef,
		"copied_at":       "catalog-import",
		"materialized_at": "catalog-import",
		"kind":            kind,
		"copy_semantics":  "copy_on_create",
		"copy_boundary":   "detached_after_create",
	}
}

func normalizeDockerOptions(runtime RuntimeDoc) map[string]any {
	out := map[string]any{}
	for k, v := range runtime.DockerOptions {
		key := k
		if key == "security_opt" {
			key = "security_options"
		}
		if key == "uts" {
			key = "uts_mode"
		}
		if isEnvLikeCatalogKey(key) {
			continue
		}
		out[key] = normalizeDockerValue(key, v)
	}
	if _, ok := out["devices"]; !ok && len(runtime.Devices) > 0 {
		out["devices"] = runtime.Devices
	}
	if _, ok := out["volumes"]; !ok && len(runtime.Volumes) > 0 {
		out["volumes"] = runtime.Volumes
	}
	return out
}

func normalizeRuntimeEnv(runtimeEnv map[string]string, dockerOptions map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range runtimeEnv {
		if strings.TrimSpace(k) != "" {
			out[k] = v
		}
	}
	for k, v := range dockerOptions {
		if !isEnvLikeCatalogKey(k) {
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprint(v))
	}
	return out
}

func isEnvLikeCatalogKey(key string) bool {
	if strings.TrimSpace(key) == "" {
		return false
	}
	hasUpper := false
	for _, r := range key {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return hasUpper && strings.Contains(key, "_")
}

func normalizeDockerValue(key string, value any) any {
	if key != "devices" {
		return value
	}
	switch v := value.(type) {
	case []any:
		var devices []map[string]string
		for _, item := range v {
			switch d := item.(type) {
			case string:
				devices = append(devices, map[string]string{"host_path": d, "container_path": d})
			case map[string]any:
				dev := map[string]string{}
				for k, raw := range d {
					dev[k] = fmt.Sprint(raw)
				}
				devices = append(devices, dev)
			default:
				devices = append(devices, map[string]string{"host_path": fmt.Sprint(d), "container_path": fmt.Sprint(d)})
			}
		}
		return devices
	default:
		return value
	}
}

func configCodesFromArgs(args []map[string]any) []string {
	codes := make([]string, 0, len(args)+1)
	for _, arg := range args {
		name := strings.TrimSpace(fmt.Sprint(arg["name"]))
		if name == "" {
			continue
		}
		if code := configCodeFromArgName(name); code != "" {
			codes = append(codes, code)
		}
	}
	codes = append(codes, "backend.extra_args")
	return codes
}

func configCodeFromArgName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || strings.HasPrefix(trimmed, "{{") {
		return ""
	}
	normalized := strings.TrimLeft(strings.ReplaceAll(trimmed, "-", "_"), "_")
	switch normalized {
	case "max_model_len", "context_length", "ctx_size":
		return "model_runtime.max_model_len"
	case "gpu_memory_utilization", "mem_fraction_static":
		return "model_runtime.gpu_memory_utilization"
	case "served_model_name":
		return "deployment.served_model_name"
	default:
		return "model_runtime." + normalized
	}
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(m[key]))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

func firstArgString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := stringFromMap(m, key); s != "" {
			return s
		}
	}
	return ""
}

func argOptions(m map[string]any) []any {
	for _, key := range []string{"options", "values", "choices"} {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case []any:
			return v
		case []string:
			out := make([]any, 0, len(v))
			for _, item := range v {
				out = append(out, item)
			}
			return out
		}
	}
	return nil
}

func normalizeConfigType(typ string) string {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "int":
		return "integer"
	case "float", "double":
		return "number"
	case "bool":
		return "boolean"
	case "integer", "number", "boolean", "array", "object", "lines":
		return strings.ToLower(strings.TrimSpace(typ))
	default:
		return "string"
	}
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func normalizedCapabilities(backend BackendDoc, version VersionDoc) any {
	switch caps := version.CapabilitiesDetail.(type) {
	case map[string]any:
		if len(caps) > 0 {
			return caps
		}
	case string:
		if strings.TrimSpace(caps) != "" {
			var decoded any
			if err := json.Unmarshal([]byte(caps), &decoded); err == nil {
				return decoded
			}
		}
	}
	tasks := []string{}
	for _, cap := range version.Capabilities {
		switch cap {
		case "chat_completions":
			tasks = appendIfMissing(tasks, "chat")
		case "completions":
			tasks = appendIfMissing(tasks, "completion")
		case "embeddings":
			tasks = appendIfMissing(tasks, "embedding")
		}
	}
	return map[string]any{
		"supported_formats":       backend.SupportedModelFormats,
		"supported_tasks":         tasks,
		"supported_capabilities":  version.Capabilities,
		"serving_protocols":       []string{version.Protocol},
		"test_endpoints":          version.DefaultEndpoints,
		"derived_from_config_set": true,
	}
}

func appendIfMissing(in []string, value string) []string {
	for _, existing := range in {
		if existing == value {
			return in
		}
	}
	return append(in, value)
}

func mustJSON(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func nonEmpty(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

func nonZero(v, fallback int) int {
	if v != 0 {
		return v
	}
	return fallback
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func defaultVersionFlag(id string) int {
	switch id {
	case "vllm-v0.23.0", "sglang-v0.5.13.post1", "llamacpp-b9700", "backend-version.ollama.latest":
		return 1
	default:
		return 0
	}
}

func displayName(id, version string) string {
	if version == "" {
		return id
	}
	return id + " (" + version + ")"
}

func displayRuntimeName(runtime RuntimeDoc) string {
	if runtime.DisplayName != "" {
		return runtime.DisplayName
	}
	if runtime.Name != "" {
		return runtime.Name
	}
	return runtime.ID
}

func runtimeStatus(runtime RuntimeDoc) string {
	if runtime.Status != "" {
		return runtime.Status
	}
	if runtimeVisibility(runtime) != "visible" {
		return "disabled"
	}
	if status, _ := runtime.Verification["status"].(string); status != "" {
		if status == "template_only" {
			return "experimental"
		}
		if status == "requires_hardware_validation" {
			return "experimental"
		}
		if status == "verified" {
			return "active"
		}
		return status
	}
	return "active"
}

func runtimeVisibility(runtime RuntimeDoc) string {
	if runtime.Visibility != "" {
		return runtime.Visibility
	}
	if visibleRuntimeIDs[runtime.ID] {
		return "visible"
	}
	return "hidden"
}

func runtimeSupportLevel(runtime RuntimeDoc) string {
	if runtime.SupportLevel != "" {
		return runtime.SupportLevel
	}
	if runtimeVisibility(runtime) != "visible" {
		return "reference"
	}
	if runtime.Vendor == "metax" || runtime.Vendor == "huawei" {
		return "experimental"
	}
	if status, _ := runtime.Verification["status"].(string); status == "verified" {
		return "verified"
	}
	return "documented"
}

var visibleRuntimeIDs = map[string]bool{
	"runtime.vllm.nvidia-docker":     true,
	"runtime.sglang.nvidia-docker":   true,
	"runtime.llamacpp.nvidia-docker": true,
	"runtime.llamacpp.cpu-docker":    true,
	"runtime.vllm.metax-docker":      true,
	"runtime.vllm.huawei-docker":     true,
}
