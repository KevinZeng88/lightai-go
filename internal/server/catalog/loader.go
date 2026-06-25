package catalog

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
	registry.byCode = map[string]ConfigItem{}
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
	}
	return nil
}

func MaterializeBackend(registry *Registry, backend BackendDoc) ConfigSet {
	items := registry.MaterializeBase("Backend", backend.ID)
	setItem(items, "runtime.health", backend.DefaultHealthCheck, backend.DefaultHealthCheck, true, "Backend", backend.ID)
	addDynamic(items, "backend.capabilities", "model_runtime", "object", map[string]any{
		"supported_formats": backend.SupportedModelFormats,
		"protocols":         backend.Protocols,
	}, "Backend", backend.ID, 5)
	return ConfigSet{
		SchemaVersion: 1,
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
	setItem(items, "launcher.entrypoint", version.DefaultEntrypoint, version.DefaultEntrypoint, len(version.DefaultEntrypoint) > 0, "BackendVersion", version.ID)
	setItem(items, "launcher.command", args, args, len(args) > 0, "BackendVersion", version.ID)
	setItem(items, "backend.common.host", nonEmpty(version.DefaultHost, "0.0.0.0"), nonEmpty(version.DefaultHost, "0.0.0.0"), true, "BackendVersion", version.ID)
	setItem(items, "backend.common.port", nonZero(version.DefaultPort, 8000), nonZero(version.DefaultPort, 8000), true, "BackendVersion", version.ID)
	setItem(items, "runtime.model_mount", version.DefaultModelMount, version.DefaultModelMount, len(version.DefaultModelMount) > 0, "BackendVersion", version.ID)
	setItem(items, "runtime.health", version.HealthCheck, version.HealthCheck, len(version.HealthCheck) > 0, "BackendVersion", version.ID)
	addDynamic(items, "backend.capabilities", "model_runtime", "object", normalizedCapabilities(backend, version), "BackendVersion", version.ID, 5)
	addDynamic(items, "backend.supported_config_items", "model_runtime", "array", configCodesFromArgs(version.DefaultArgsSchema), "BackendVersion", version.ID, 6)
	addArgConfigItems(items, version.DefaultArgsSchema, "BackendVersion", version.ID)
	return ConfigSet{
		SchemaVersion: 1,
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

func addArgConfigItems(items map[string]ConfigItem, args []map[string]any, layer, ref string) {
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
		enabled := boolFromAny(arg["required"]) || defaultValue != nil
		item := ConfigItem{
			Code:         code,
			Category:     "model_runtime",
			Kind:         "cli_arg",
			Type:         normalizeConfigType(typ),
			Value:        defaultValue,
			DefaultValue: defaultValue,
			Enabled:      enabled,
			Render: map[string]any{
				"target": "cli",
				"flag":   name,
				"style":  renderStyleForArgType(typ),
			},
			Order:        300 + idx,
			SupportLevel: "documented",
			Source:       map[string]string{"layer": layer, "ref": ref, "reason": "default_args_schema"},
		}
		if label := strings.TrimSpace(fmt.Sprint(arg["label"])); label != "" && label != "<nil>" {
			item.Extensions = map[string]interface{}{"label": label, "group": strings.TrimSpace(fmt.Sprint(arg["group"]))}
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
	setItem(items, "launcher.kind", nonEmpty(runtime.RunnerType, "docker"), nonEmpty(runtime.RunnerType, "docker"), true, "BackendRuntime", runtime.ID)
	setItem(items, "launcher.image", image, image, image != "", "BackendRuntime", runtime.ID)
	setItem(items, "launcher.entrypoint", runtime.Entrypoint, runtime.Entrypoint, len(runtime.Entrypoint) > 0, "BackendRuntime", runtime.ID)
	setItem(items, "launcher.command", runtime.Args, runtime.Args, len(runtime.Args) > 0, "BackendRuntime", runtime.ID)
	setItem(items, "launcher.docker_options", normalizeDockerOptions(runtime), normalizeDockerOptions(runtime), true, "BackendRuntime", runtime.ID)
	setItem(items, "runtime.env", runtime.Env, runtime.Env, len(runtime.Env) > 0, "BackendRuntime", runtime.ID)
	setItem(items, "runtime.model_mount", runtime.ModelMount, runtime.ModelMount, len(runtime.ModelMount) > 0, "BackendRuntime", runtime.ID)
	setItem(items, "runtime.health", runtime.HealthCheck, runtime.HealthCheck, len(runtime.HealthCheck) > 0, "BackendRuntime", runtime.ID)
	if len(runtime.Ports) > 0 {
		setItem(items, "launcher.ports", runtime.Ports, runtime.Ports, true, "BackendRuntime", runtime.ID)
	}
	return ConfigSet{
		SchemaVersion: 1,
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
			(id, name, display_name, backend_id, backend_version_id, source_template_name, vendor, runtime_type, is_builtin, is_editable, tenant_id, slug, managed_by, source, catalog_version, checksum, status, verification_json, hardware_family, accelerator_api, runtime_distribution, runtime_distribution_version, config_hash, loaded_from, loaded_at, config_set_json, source_metadata_json, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(id) DO UPDATE SET
				name=excluded.name, display_name=excluded.display_name, backend_id=excluded.backend_id,
				backend_version_id=excluded.backend_version_id, source_template_name=excluded.source_template_name,
				vendor=excluded.vendor, runtime_type=excluded.runtime_type, is_builtin=excluded.is_builtin,
				is_editable=excluded.is_editable, tenant_id=excluded.tenant_id, slug=excluded.slug,
				managed_by=excluded.managed_by, source=excluded.source, catalog_version=excluded.catalog_version,
				checksum=excluded.checksum, status=excluded.status, verification_json=excluded.verification_json,
				hardware_family=excluded.hardware_family, accelerator_api=excluded.accelerator_api,
				runtime_distribution=excluded.runtime_distribution, runtime_distribution_version=excluded.runtime_distribution_version,
				config_hash=excluded.config_hash, loaded_from=excluded.loaded_from, loaded_at=excluded.loaded_at,
				config_set_json=excluded.config_set_json, source_metadata_json=excluded.source_metadata_json,
				updated_at=excluded.updated_at`,
			runtime.ID, nonEmpty(runtime.Name, runtime.Slug), displayRuntimeName(runtime), runtime.BackendID, runtime.BackendVersionID, runtime.Slug, runtime.Vendor, nonEmpty(runtime.RunnerType, "docker"), 1, 0, "",
			runtime.Slug, "system", "config-registry", "configset-v1", runtime.SourceHash, runtimeStatus(runtime), mustJSON(runtime.Verification),
			runtime.HardwareFamily, runtime.AcceleratorAPI, runtime.RuntimeDistribution, runtime.RuntimeDistributionVersion, runtime.SourceHash, runtime.SourcePath, now, configSet, mustJSON(set.SourceMetadata), now, now); err != nil {
			return fmt.Errorf("seed backend runtime %s: %w", runtime.ID, err)
		}
	}
	return nil
}

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

func cloneItems(in map[string]ConfigItem) map[string]ConfigItem {
	out := make(map[string]ConfigItem, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func setItem(items map[string]ConfigItem, code string, value, defaultValue any, enabled bool, layer, ref string) {
	item := items[code]
	if item.Code == "" {
		item = ConfigItem{Code: code, SupportLevel: "documented"}
	}
	item.Value = value
	item.DefaultValue = defaultValue
	item.Enabled = enabled
	item.LastModified = map[string]string{"layer": layer, "ref": ref, "operation": "materialize"}
	items[code] = item
}

func addDynamic(items map[string]ConfigItem, code, category, typ string, value any, layer, ref string, order int) {
	items[code] = ConfigItem{
		Code:         code,
		Category:     category,
		Kind:         "launcher_option",
		Type:         typ,
		Value:        value,
		DefaultValue: value,
		Enabled:      true,
		Order:        order,
		SupportLevel: "documented",
		Source:       map[string]string{"layer": layer, "ref": ref, "reason": "catalog_materialized"},
	}
}

func sourceMetadata(path, hash, parentRef, kind string) map[string]interface{} {
	return map[string]interface{}{
		"source_ref":      path,
		"source_hash":     hash,
		"parent_ref":      parentRef,
		"materialized_at": "catalog-import",
		"kind":            kind,
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
	return "backend.arg." + strings.TrimLeft(strings.ReplaceAll(trimmed, "-", "_"), "_")
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

func renderStyleForArgType(typ string) string {
	if normalizeConfigType(typ) == "boolean" {
		return "flag_if_true"
	}
	return "flag_space_value"
}

func boolFromAny(v any) bool {
	b, _ := v.(bool)
	return b
}

func normalizedCapabilities(backend BackendDoc, version VersionDoc) any {
	switch caps := version.CapabilitiesJSON.(type) {
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

func firstNonEmptySlice(a, b []string) []string {
	if len(a) > 0 {
		return a
	}
	return b
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
	if status, _ := runtime.Verification["status"].(string); status != "" {
		return status
	}
	return "active"
}
