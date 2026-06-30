package configedit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"lightai-go/internal/server/semanticconfig"
)

func ProjectConfigSetToEditView(input ProjectInput) (ConfigEditView, error) {
	set := NormalizeConfigSet(input.ConfigSet)
	viewLevel := normalizeViewLevel(input.ViewLevel)
	if viewLevel == "" {
		viewLevel = viewLevelFromMode(input.Mode)
	}
	sections := map[string]*EditSection{}
	for key, label := range sectionLabels {
		sections[key] = &EditSection{
			Key:       key,
			Label:     label,
			Order:     sectionOrder[key],
			Advanced:  key == "advanced_raw" || key == "advanced_parameters" || key == "expert_parameters",
			Collapsed: key == "advanced_raw" || key == "advanced_parameters" || key == "expert_parameters" || key == "environment",
		}
	}

	items := itemsMap(set)

	// Track which canonical keys have been projected to avoid duplicates.
	projectedCanonical := map[string]bool{}

	for code, raw := range items {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		if input.Mode != "advanced" && hideFromOrdinaryFlow(code, item, input.Layer) {
			continue
		}

		// Ensure each item carries its own code for taxonomy/wiget lookups.
		item["code"] = code

		// --- Canonical alias merge ---
		// If this code is an alias for a canonical key, merge into canonical.
		if canon, ok := aliasCanonicalOf[code]; ok && canon != code {
			if isLayerHidden(code, input.Layer) {
				continue
			}
			// Alias: only render if canonical hasn't been rendered yet.
			if projectedCanonical[canon] {
				continue // already projected via primary key
			}
			// Use the alias group's preferred section/widget, merge values.
			canonItem := mergeCanonicalItem(code, canon, item, items)
			projectedCanonical[canon] = true
			field := projectItem(canon, canon, nil, canonItem, input)
			if !fieldVisibleAtView(field, viewLevel) {
				continue
			}
			sections[field.Section].Fields = append(sections[field.Section].Fields, field)
			continue
		}
		if canon, ok := aliasCanonicalOf[code]; ok && canon == code {
			if isLayerHidden(code, input.Layer) {
				continue
			}
			if projectedCanonical[code] {
				continue // already projected via alias
			}
			projectedCanonical[code] = true
			canonItem := mergeCanonicalItem(code, code, item, items)
			field := projectItem(code, code, nil, canonItem, input)
			if !fieldVisibleAtView(field, viewLevel) {
				continue
			}
			sections[field.Section].Fields = append(sections[field.Section].Fields, field)
			continue
		}

		// --- Layer scope filter ---
		if isLayerHidden(code, input.Layer) {
			continue
		}

		if code == "launcher.docker_options" {
			for _, field := range projectDockerOptions(item, input) {
				if isLayerHidden(field.Key, input.Layer) {
					continue
				}
				if !fieldVisibleAtView(field, viewLevel) {
					continue
				}
				sections[field.Section].Fields = append(sections[field.Section].Fields, field)
			}
			continue
		}
		field := projectItem(code, code, nil, item, input)
		if !fieldVisibleAtView(field, viewLevel) {
			continue
		}
		sections[field.Section].Fields = append(sections[field.Section].Fields, field)
	}

	outSections := make([]EditSection, 0, len(sections))
	for _, section := range sections {
		sort.SliceStable(section.Fields, func(i, j int) bool {
			if section.Fields[i].Order == section.Fields[j].Order {
				return section.Fields[i].Label < section.Fields[j].Label
			}
			return section.Fields[i].Order < section.Fields[j].Order
		})
		if len(section.Fields) > 0 || section.Key == "advanced_raw" {
			outSections = append(outSections, *section)
		}
	}
	sort.SliceStable(outSections, func(i, j int) bool { return outSections[i].Order < outSections[j].Order })

	fields := flattenSectionFields(outSections)
	components := buildComponents(fields)
	effects := buildEffectsPreview(components)
	snapshotID := input.SnapshotID
	if snapshotID == "" {
		snapshotID = snapshotIDForConfigSet(set)
	}
	templateID := input.TemplateID
	if templateID == "" {
		templateID = templateIDFor(set, input.ObjectKind)
	}
	childInit := ChildInitContract{Strategy: "copy_effective_snapshot", CopyScope: "whole_effective_configedit_snapshot"}
	if input.ChildInit != nil {
		childInit = *input.ChildInit
	}

	return ConfigEditView{
		Layer:          input.Layer,
		ObjectID:       input.ObjectID,
		ObjectKind:     input.ObjectKind,
		TemplateID:     templateID,
		SnapshotID:     snapshotID,
		Parent:         input.Parent,
		ChildInit:      childInit,
		ViewLevel:      viewLevel,
		Readonly:       input.Readonly,
		Sections:       outSections,
		Components:     components,
		Fields:         fields,
		EffectsPreview: effects,
		Diagnostics: ConfigEditDiagnostics{
			RawConfigSet: set,
		},
		Metadata: map[string]any{
			"object_label": input.ObjectLabel,
			"mode":         input.Mode,
		},
	}, nil
}

func normalizeViewLevel(view string) string {
	switch strings.ToLower(strings.TrimSpace(view)) {
	case "normal", "advanced", "security", "high-risk", "high_risk", "developer":
		if strings.ToLower(strings.TrimSpace(view)) == "high-risk" || strings.ToLower(strings.TrimSpace(view)) == "high_risk" {
			return "security"
		}
		return strings.ToLower(strings.TrimSpace(view))
	default:
		return ""
	}
}

func viewLevelFromMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "developer", "debug":
		return "developer"
	case "advanced":
		return "advanced"
	case "security", "high-risk", "high_risk":
		return "security"
	case "normal":
		return "normal"
	case "":
		return "developer"
	default:
		return "developer"
	}
}

func fieldVisibleAtView(field EditField, view string) bool {
	switch view {
	case "developer":
		return true
	case "advanced", "security":
		return field.Visibility != "internal" && field.Visibility != "hidden"
	default:
		return !field.Advanced && !field.Diagnostic && field.Visibility != "internal" && field.Visibility != "hidden"
	}
}

func flattenSectionFields(sections []EditSection) []EditField {
	var fields []EditField
	for _, section := range sections {
		fields = append(fields, section.Fields...)
	}
	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i].Section == fields[j].Section {
			if fields[i].Order == fields[j].Order {
				return fields[i].Key < fields[j].Key
			}
			return fields[i].Order < fields[j].Order
		}
		return sectionOrder[fields[i].Section] < sectionOrder[fields[j].Section]
	})
	return fields
}

func buildComponents(fields []EditField) []EditComponent {
	byKey := map[string]*EditComponent{}
	for _, field := range fields {
		key := componentKeyForField(field)
		c := byKey[key]
		if c == nil {
			c = &EditComponent{
				Key:      key,
				Type:     componentTypeForField(field),
				Renderer: componentRendererForField(field),
				Label:    componentLabelForField(field),
				Section:  field.Section,
				View:     field.View,
				Order:    field.Order,
				Enabled:  field.Enabled,
				Readonly: field.Readonly,
				Source:   field.Source,
				Reset:    field.Reset,
			}
			byKey[key] = c
		}
		c.Fields = append(c.Fields, field.Key)
		c.Effects = append(c.Effects, field.Effects...)
		if !field.Readonly {
			c.Readonly = false
		}
		if field.Enabled {
			c.Enabled = true
		}
	}
	out := make([]EditComponent, 0, len(byKey))
	for _, c := range byKey {
		out = append(out, *c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Section == out[j].Section {
			if out[i].Order == out[j].Order {
				return out[i].Key < out[j].Key
			}
			return out[i].Order < out[j].Order
		}
		return sectionOrder[out[i].Section] < sectionOrder[out[j].Section]
	})
	return out
}

func buildEffectsPreview(components []EditComponent) []EditEffectPreview {
	var out []EditEffectPreview
	for _, c := range components {
		out = append(out, c.Effects...)
	}
	return out
}

func snapshotIDForConfigSet(set map[string]any) string {
	b, _ := json.Marshal(set)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}

func templateIDFor(set map[string]any, objectKind string) string {
	ctx, _ := set["context"].(map[string]any)
	backend := strings.TrimSpace(stringValue(ctx["backend"]))
	vendor := strings.TrimSpace(stringValue(ctx["vendor"]))
	if vendor == "" {
		vendor = strings.TrimSpace(stringValue(ctx["accelerator_vendor"]))
	}
	if backend != "" && vendor != "" {
		return backend + "-" + vendor + "-docker-configedit-v1"
	}
	if backend != "" {
		return backend + "-docker-configedit-v1"
	}
	if objectKind != "" {
		return objectKind + "-configedit-v1"
	}
	return "generic-configedit-v1"
}

func hideFromOrdinaryFlow(key string, item map[string]any, layer string) bool {
	if layer != "node_backend_runtime" && layer != "deployment" {
		return false
	}
	visibility := itemVisibility(item)
	if visibility == "internal" || visibility == "hidden" || visibility == "deprecated" {
		return true
	}
	if tieredBoolField(item, "state", "deprecated") {
		return true
	}
	category := itemCategory(item)
	if category == "internal" || category == "debug" || strings.HasPrefix(key, "internal.") || strings.HasPrefix(key, "source_metadata.") || strings.HasPrefix(key, "resolver.") {
		return true
	}
	return false
}

// mergeCanonicalItem creates a merged item for a canonical key by combining
// values from the primary key and all its aliases.
func mergeCanonicalItem(code, canon string, item map[string]any, items map[string]any) map[string]any {
	out := cloneMap(item)
	if out == nil {
		out = map[string]any{}
	}
	out["code"] = canon

	// Resolve value: primary key value takes precedence, then first alias with value.
	currentValue := itemEffectiveValue(item)
	if isEmptyValue(currentValue) {
		for _, g := range canonicalAliases {
			if g.Canonical != canon {
				continue
			}
			for _, alias := range g.Aliases {
				aliasItem, _ := items[alias].(map[string]any)
				if aliasItem == nil {
					continue
				}
				av := aliasItem["value"]
				if !isEmptyValue(av) {
					currentValue = av
					break
				}
			}
		}
	}
	out["value"] = currentValue

	// Resolve label per canonical group.
	for _, g := range canonicalAliases {
		if g.Canonical == canon {
			if g.Label != "" {
				if out["render"] == nil {
					out["render"] = map[string]any{}
				}
				nestedMap(out, "render")["label"] = g.Label
			}
			if g.Section != "" {
				if out["render"] == nil {
					out["render"] = map[string]any{}
				}
				nestedMap(out, "render")["section"] = g.Section
			}
			if g.Widget != "" {
				if out["render"] == nil {
					out["render"] = map[string]any{}
				}
				nestedMap(out, "render")["widget"] = g.Widget
			}
			break
		}
	}

	return out
}

func projectDockerOptions(item map[string]any, input ProjectInput) []EditField {
	value := valueMap(item)
	enabledFields := nestedMap(item, "enabled_fields")
	var fields []EditField
	projected := map[string]bool{}
	for _, spec := range dockerFieldSpecs {
		code := "launcher.docker_options." + spec.Path
		projected[spec.Path] = true
		dockerItem := cloneMap(item)
		dockerItem["type"] = spec.Type
		subVal := value[spec.Path]
		// Replace entire value tier so parent default_value does not leak
		// into child fields that are absent from the object (e.g. uts_mode,
		// network_mode show the full docker_options parent object).
		dockerItem["value"] = map[string]any{
			"effective_value": subVal,
			"local_value":     subVal,
			"default_value":   subVal,
		}
		dockerItem["required"] = false
		// Docker sub-fields keep value and enabled state independently. The
		// parent object stores values under value and per-subfield toggles under
		// enabled_fields to avoid inferring activation from a prefilled value.
		dockerItem["enabled"] = boolValue(enabledFields[spec.Path])
		// Also update state tier for tiered-only reading
		if st, ok := dockerItem["state"].(map[string]any); ok {
			st["enabled"] = boolValue(enabledFields[spec.Path])
		}
		field := projectItem(code, "launcher.docker_options", []string{spec.Path}, dockerItem, input)
		field.Section = spec.Section
		field.Widget = spec.Widget
		field.Order = spec.Order
		field.Label = fieldLabel(code, dockerItem)
		applyDockerFieldPolicy(&field, spec.Path, subVal)
		fields = append(fields, field)
	}
	var extraPaths []string
	for path := range value {
		if projected[path] || strings.TrimSpace(path) == "" {
			continue
		}
		extraPaths = append(extraPaths, path)
	}
	sort.Strings(extraPaths)
	for _, path := range extraPaths {
		code := "launcher.docker_options." + path
		subVal := value[path]
		dockerItem := cloneMap(item)
		dockerItem["type"] = inferredConfigType(subVal)
		dockerItem["value"] = map[string]any{
			"effective_value": subVal,
			"local_value":     subVal,
			"default_value":   subVal,
		}
		dockerItem["required"] = false
		if _, ok := enabledFields[path]; ok {
			dockerItem["enabled"] = boolValue(enabledFields[path])
		} else {
			dockerItem["enabled"] = !isEmptyValue(subVal)
		}
		if st, ok := dockerItem["state"].(map[string]any); ok {
			st["enabled"] = boolValue(dockerItem["enabled"])
		}
		field := projectItem(code, "launcher.docker_options", []string{path}, dockerItem, input)
		applyDockerFieldPolicy(&field, path, subVal)
		fields = append(fields, field)
	}
	return fields
}

func applyDockerFieldPolicy(field *EditField, path string, value any) {
	field.Section = dockerSectionFor(path)
	field.Widget = dockerWidgetFor(path, value)
	field.Type = inferredConfigType(value)
	field.Label = fieldLabel("launcher.docker_options."+path, map[string]any{})
	field.Order = dockerOrderFor(path)
	if field.Section == "security_high_risk" {
		field.Advanced = true
		field.View = "security"
		field.Diagnostic = false
		if len(field.Warnings) == 0 {
			field.Warnings = []any{map[string]any{
				"level":   "warning",
				"message": "High-risk Docker option. Review host isolation and device exposure before enabling.",
			}}
		}
	}
}

func dockerSectionFor(path string) string {
	code := "launcher.docker_options." + path
	if isHighRiskDockerOptionCode(code) {
		return "security_high_risk"
	}
	switch path {
	case "devices", "group_add":
		return "devices_mounts"
	default:
		return "container_resources"
	}
}

func dockerWidgetFor(path string, value any) string {
	switch path {
	case "devices":
		return "device_table"
	case "ulimits":
		return "key_value_table"
	case "privileged":
		return "boolean"
	}
	switch value.(type) {
	case []any, []string:
		return "string_list"
	case map[string]any, map[string]string:
		return "key_value_table"
	case bool:
		return "boolean"
	case int, int64, float64, float32:
		return "number"
	default:
		return "string"
	}
}

func dockerOrderFor(path string) int {
	for _, spec := range dockerFieldSpecs {
		if spec.Path == path {
			return spec.Order
		}
	}
	switch dockerSectionFor(path) {
	case "security_high_risk":
		return 200 + stablePathOrder(path)
	case "devices_mounts":
		return 200 + stablePathOrder(path)
	default:
		return 200 + stablePathOrder(path)
	}
}

func stablePathOrder(path string) int {
	total := 0
	for _, r := range path {
		total += int(r)
	}
	return total % 100
}

func inferredConfigType(value any) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int64:
		return "integer"
	case float64, float32:
		return "number"
	case []any, []string:
		return "array"
	case map[string]any, map[string]string:
		return "object"
	default:
		return "string"
	}
}

func projectItem(key, internalKey string, path []string, item map[string]any, input ProjectInput) EditField {
	registry := semanticconfig.DefaultRegistry()
	semanticKey := key
	def, hasDef := registry.Get(key)
	if !hasDef {
		if canonical, ok := registry.CanonicalKey(key); ok {
			semanticKey = canonical
			def, hasDef = registry.Get(canonical)
		}
	}
	displayKey := key
	if hasDef && semanticKey != "" {
		displayKey = semanticKey
	}
	required := itemRequired(item)

	// Enabled: read from tiered state only.
	enabled := itemEnabled(item)
	if required {
		enabled = true
	}

	// Determine if this is a capability/internal field that should be forced to readonly summary.
	isCapability := capabilityLikeCodes[key] || strings.Contains(key, "capabilities") || strings.Contains(key, "supported_config")
	isInternalMeta := strings.HasPrefix(key, "internal.") || strings.HasPrefix(key, "source_metadata.") || strings.HasPrefix(key, "resolver.")

	visibility := itemVisibility(item)
	deprecated := tieredBoolField(item, "state", "deprecated") || visibility == "deprecated"
	debugLike := itemCategory(item) == "debug" || strings.Contains(key, "debug") || strings.Contains(key, "profile")
	tier := displayTierFor(displayKey, item, hasDef, string(def.DisplayTier))
	advanced := tieredBoolField(item, "schema", "advanced") || tier == "advanced" || tier == "expert" || isCapability || isInternalMeta || visibility == "internal" || visibility == "hidden" || deprecated || debugLike
	section := sectionFor(displayKey, item)
	if input.Mode != "advanced" && (deprecated || debugLike || visibility == "internal" || visibility == "hidden") {
		section = "advanced_raw"
	}
	if isModelServingCode(key) || strings.HasPrefix(displayKey, "model_runtime.") {
		switch tier {
		case "common":
			section = "model_serving"
		case "expert":
			section = "expert_parameters"
		default:
			section = "advanced_parameters"
		}
	}
	if advanced {
		if section != "advanced_parameters" && section != "expert_parameters" && section != "security_high_risk" {
			section = "advanced_raw"
		}
	}

	// Readonly from input, item config, layer scope, or capability status.
	readonly := input.Readonly || itemReadonly(item) || isCapability || visibility == "readonly" || visibility == "internal"
	if isLayerReadonly(key, input.Layer) {
		readonly = true
	}
	// Deployment protected fields (image/command/entrypoint/model_mount) handled separately.
	if input.Layer == "deployment" && deploymentProtectedFields[internalKey] {
		readonly = true
	}

	// Determine widget, forcing readonly_summary for capability fields.
	widget := widgetFor(item)
	if isCapability && section == "advanced_raw" {
		widget = "readonly_summary"
	}

	// Handle {{container_port}} template variable: show as readonly hint, not editable.
	value := valueOrDefault(item)
	if s, ok := value.(string); ok && strings.Contains(s, "{{") {
		readonly = true
	}

	warnings, _ := item["warnings"].([]any)
	technicalKey := firstString(displayKey, key, internalKey)
	field := EditField{
		Key:                displayKey,
		InternalKey:        internalKey,
		SemanticKey:        semanticKey,
		ParentKey:          parentKey(internalKey, path),
		Path:               path,
		Label:              fieldLabel(displayKey, item),
		LabelI18nKey:       firstString(nestedString(item, "schema", "label_i18n_key"), nestedString(item, "render", "label_i18n_key"), nestedString(item, "extensions", "label_i18n_key"), "configEdit.labels."+displayKey),
		TitleI18nKey:       firstString(nestedString(item, "schema", "title_i18n_key"), nestedString(item, "render", "title_i18n_key"), nestedString(item, "extensions", "title_i18n_key")),
		DescriptionI18nKey: firstString(nestedString(item, "schema", "description_i18n_key"), nestedString(item, "render", "description_i18n_key"), nestedString(item, "extensions", "description_i18n_key"), "configEdit.descriptions."+displayKey),
		HelpI18nKey:        firstString(nestedString(item, "schema", "help_i18n_key"), nestedString(item, "render", "help_i18n_key"), nestedString(item, "extensions", "help_i18n_key"), "configEdit.descriptions."+displayKey),
		TooltipI18nKey:     firstString(nestedString(item, "schema", "tooltip_i18n_key"), nestedString(item, "render", "tooltip_i18n_key"), nestedString(item, "extensions", "tooltip_i18n_key")),
		Title:              firstString(nestedString(item, "schema", "title"), nestedString(item, "render", "title"), nestedString(item, "extensions", "title")),
		Description:        firstString(nestedString(item, "schema", "description"), nestedString(item, "render", "description"), nestedString(item, "extensions", "description")),
		Help:               firstString(nestedString(item, "schema", "help_text"), nestedString(item, "schema", "help"), nestedString(item, "render", "help"), nestedString(item, "extensions", "help")),
		CliFlag:            configEditCliFlag(displayKey, item),
		EnvKey:             configEditEnvKey(displayKey, item),
		TechnicalKey:       technicalKey,
		Section:            section,
		Group:              firstString(nestedString(item, "render", "group"), nestedString(item, "extensions", "group")),
		Order:              firstInt(tieredAnyField(item, "schema", "display_order"), item["order"]),
		Type:               firstString(tieredStringField(item, "schema", "type"), stringValue(item["type"]), "string"),
		Widget:             widget,
		Value:              value,
		DefaultValue:       itemDefaultValue(item),
		Enabled:            enabled,
		HasEnable:          !required && !readonly,
		Required:           required,
		Readonly:           readonly,
		Advanced:           advanced || section == "advanced_raw",
		Visibility:         visibility,
		Options:            optionsFor(item),
		Constraints:        constraintsFor(item),
		ValidationRules:    validationRulesFor(item),
		Placeholder:        firstString(nestedString(item, "schema", "placeholder"), nestedString(item, "presentation", "placeholder"), nestedString(item, "render", "placeholder"), nestedString(item, "extensions", "placeholder")),
		Sensitive:          tieredBoolField(item, "presentation", "sensitive"),
		Disabled:           readonly,
		Source:             sourceFor(item),
		ValueSource:        nestedString(item, "provenance", "value_source"),
		LastValueLayer:     nestedString(item, "provenance", "last_value_layer"),
		InheritedValue:     itemInheritedValue(item),
		CopyBehavior:       firstString(nestedString(item, "schema", "copy_behavior"), nestedString(item, "render", "copy_behavior"), "copy_on_create"),
		OverrideBehavior:   firstString(nestedString(item, "schema", "override_behavior"), nestedString(item, "render", "override_behavior"), "patch_local_value"),
		DisableBehavior:    firstString(nestedString(item, "schema", "disable_behavior"), nestedString(item, "render", "disable_behavior"), "retain_value_when_disabled"),
		PatchTarget:        firstString(nestedString(item, "schema", "patch_target"), internalKey),
		CopiedFrom:         firstString(nestedString(item, "snapshot", "snapshot_from_id"), stringValue(item["copied_from"]), stringValue(item["copiedFrom"])),
		Dirty:              boolValue(item["dirty"]),
		Warnings:           warnings,
		Diagnostic:         advanced || visibility == "internal" || visibility == "hidden",
		OriginalValue:      value,
		OriginalEnabled:    enabled,
		ComponentKey:       componentKeyForRaw(displayKey, internalKey),
		View:               viewForField(advanced, visibility),
		Reset: ResetBehavior{
			AllowResetToParent:  true,
			AllowResetToDefault: true,
		},
	}
	field.Effects = effectsForField(field)
	if hasDef {
		field.Owner = string(def.Owner)
		field.Tier = tier
		field.Label = firstString(field.Label, def.Label)
	} else {
		field.Tier = tier
	}
	return field
}

func componentKeyForField(field EditField) string {
	if field.ComponentKey != "" {
		return field.ComponentKey
	}
	return componentKeyForRaw(field.Key, field.InternalKey)
}

func componentKeyForRaw(key, internalKey string) string {
	candidate := key
	if internalKey != "" {
		candidate = internalKey
	}
	switch {
	case candidate == "runtime.device_binding" || strings.HasPrefix(candidate, "runtime.device_binding."):
		return "runtime.device_binding"
	case candidate == "service.port_binding" || strings.HasPrefix(candidate, "service.port_binding."):
		return "service.port_binding"
	case candidate == "runtime.model_mount" || strings.HasPrefix(candidate, "runtime.model_mount."):
		return "runtime.model_mount"
	case candidate == "runtime.health" || strings.HasPrefix(candidate, "runtime.health."):
		return "runtime.health_check"
	case candidate == "runtime.env" || strings.HasPrefix(candidate, "runtime.env."):
		return "runtime.env"
	case candidate == "runtime.extra_env" || strings.HasPrefix(candidate, "runtime.extra_env."):
		return "runtime.extra_env"
	case candidate == "backend.extra_args" || strings.HasPrefix(candidate, "backend.extra_args."):
		return "backend.extra_args"
	case candidate == "launcher.docker_options" || strings.HasPrefix(candidate, "launcher.docker_options."):
		return "launcher.docker_options"
	case strings.HasPrefix(candidate, "backend.arg.") || strings.HasPrefix(candidate, "model_runtime."):
		return "backend.args"
	case candidate == "service.container_port" || candidate == "service.listen_host" || candidate == "deployment.host_port" || candidate == "deployment.served_model_name":
		return "service.port_binding"
	default:
		return candidate
	}
}

func componentTypeForField(field EditField) string {
	switch componentKeyForField(field) {
	case "runtime.device_binding":
		return "accelerator_binding"
	case "service.port_binding":
		return "port_binding"
	case "runtime.model_mount":
		return "model_mount"
	case "runtime.health_check":
		return "health_check"
	case "runtime.env", "runtime.extra_env":
		return "env"
	case "backend.args", "backend.extra_args":
		return "args"
	case "launcher.docker_options":
		return "docker_options"
	default:
		return "field"
	}
}

func componentRendererForField(field EditField) string {
	switch componentTypeForField(field) {
	case "accelerator_binding":
		return "accelerator_binding"
	case "port_binding":
		return "port_binding"
	case "model_mount":
		return "mount_form"
	case "health_check":
		return "health_check_form"
	case "env":
		return "key_value_table"
	case "args":
		return "args_editor"
	case "docker_options":
		return "docker_options"
	default:
		return field.Widget
	}
}

func componentLabelForField(field EditField) string {
	switch componentKeyForField(field) {
	case "runtime.device_binding":
		return "Device binding"
	case "service.port_binding":
		return "Service port"
	case "runtime.model_mount":
		return "Model mount"
	case "runtime.health_check":
		return "Health check"
	case "runtime.env":
		return "Runtime environment"
	case "runtime.extra_env":
		return "Extra environment"
	case "backend.args":
		return "Backend arguments"
	case "backend.extra_args":
		return "Extra arguments"
	case "launcher.docker_options":
		return "Docker options"
	default:
		return field.Label
	}
}

func viewForField(advanced bool, visibility string) string {
	if visibility == "internal" || visibility == "hidden" {
		return "developer"
	}
	if advanced {
		return "advanced"
	}
	return "normal"
}

func effectsForField(field EditField) []EditEffectPreview {
	componentKey := componentKeyForField(field)
	base := EditEffectPreview{
		ComponentKey: componentKey,
		FieldKey:     field.Key,
		Source:       firstString(field.ValueSource, "configedit_snapshot"),
		PatchTarget:  field.PatchTarget,
	}
	switch componentKey {
	case "runtime.device_binding":
		return []EditEffectPreview{
			{ComponentKey: componentKey, FieldKey: field.Key, Type: "docker", Target: "docker.gpus", Value: field.Value, PatchTarget: field.PatchTarget, DockerEffect: "--gpus"},
			{ComponentKey: componentKey, FieldKey: field.Key, Type: "env", Target: "env", Key: "visible_devices", Value: field.Value, PatchTarget: field.PatchTarget, DockerEffect: "-e"},
		}
	case "service.port_binding":
		base.Type = "port"
		base.Target = "ports"
		base.Value = field.Value
		base.DockerEffect = "-p / backend CLI port"
		return []EditEffectPreview{base}
	case "runtime.model_mount":
		base.Type = "mount"
		base.Target = "mounts"
		base.Value = field.Value
		base.DockerEffect = "-v"
		return []EditEffectPreview{base}
	case "runtime.health_check":
		base.Type = "health_check"
		base.Target = "health_check"
		base.Value = field.Value
		return []EditEffectPreview{base}
	case "runtime.env", "runtime.extra_env":
		base.Type = "env"
		base.Target = "env"
		base.Value = field.Value
		base.DockerEffect = "-e"
		return []EditEffectPreview{base}
	case "backend.args", "backend.extra_args":
		base.Type = "cli_arg"
		base.Target = "args"
		base.Key = field.CliFlag
		base.Value = field.Value
		base.DockerEffect = field.CliFlag
		return []EditEffectPreview{base}
	case "launcher.docker_options":
		base.Type = "docker"
		base.Target = "docker_options"
		base.Value = field.Value
		base.DockerEffect = sourceEffectForDockerField(field.Key)
		return []EditEffectPreview{base}
	}
	return nil
}

func sourceEffectForDockerField(key string) string {
	switch {
	case strings.Contains(key, "shm_size"):
		return "--shm-size"
	case strings.Contains(key, "ipc_mode"):
		return "--ipc"
	case strings.Contains(key, "network_mode"):
		return "--network"
	case strings.Contains(key, "privileged"):
		return "--privileged"
	case strings.Contains(key, "devices"):
		return "--device"
	case strings.Contains(key, "group_add"):
		return "--group-add"
	case strings.Contains(key, "cap_add"):
		return "--cap-add"
	case strings.Contains(key, "cap_drop"):
		return "--cap-drop"
	default:
		return "docker option"
	}
}

func displayTierFor(key string, item map[string]any, hasDef bool, registryTier string) string {
	if tier := stringValue(item["tier"]); tier != "" {
		return normalizeTier(tier)
	}
	if tier := nestedString(item, "render", "tier"); tier != "" {
		return normalizeTier(tier)
	}
	if tier := nestedString(item, "extensions", "tier"); tier != "" {
		return normalizeTier(tier)
	}
	if boolValue(item["dangerous"]) || expertRuntimeArgs[key] {
		return "expert"
	}
	if commonRuntimeArgs[key] || tieredBoolField(item, "schema", "visible_by_default") {
		return "common"
	}
	if hasDef {
		return normalizeTier(registryTier)
	}
	if isModelServingCode(key) || strings.HasPrefix(key, "model_runtime.") {
		return "advanced"
	}
	return "advanced"
}

func normalizeTier(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "common", "required", "recommended", "deployment_common_advanced":
		return "common"
	case "expert", "dangerous", "diagnostic":
		return "expert"
	case "advanced":
		return "advanced"
	default:
		return "advanced"
	}
}

func parentKey(internalKey string, path []string) string {
	if len(path) == 0 {
		return ""
	}
	return internalKey
}

func widgetFor(item map[string]any) string {
	if w := nestedString(item, "render", "widget"); w != "" {
		return w
	}
	if style := nestedString(item, "render", "style"); style == "raw_lines" {
		return "textarea"
	}
	// Check widget overrides based on code.
	if code := stringValue(item["code"]); code != "" {
		if w, ok := widgetOverrides[code]; ok {
			return w
		}
	}
	switch stringValue(item["type"]) {
	case "boolean", "bool":
		return "boolean"
	case "integer", "int", "number", "float":
		return "number"
	case "select", "enum":
		return "select"
	case "multi_select":
		return "multi_select"
	case "object":
		return "raw_json"
	case "array", "list":
		return "string_list"
	default:
		return "string"
	}
}

func valueOrDefault(item map[string]any) any {
	v := itemEffectiveValue(item)
	if v != nil {
		return v
	}
	if schema, ok := item["schema"].(map[string]any); ok {
		if dv, ok := schema["default_value"]; ok {
			return dv
		}
	}
	// Flat compat
	if dv := item["default_value"]; dv != nil {
		return dv
	}
	return nil
}

func optionsFor(item map[string]any) []EditOption {
	raw, _ := nestedMap(item, "render")["options"].([]any)
	if len(raw) == 0 {
		raw, _ = nestedMap(item, "constraints")["options"].([]any)
	}
	if len(raw) == 0 {
		if constraints, ok := nestedMap(item, "schema")["constraints"].(map[string]any); ok {
			raw, _ = constraints["options"].([]any)
		}
	}
	if len(raw) == 0 {
		raw, _ = nestedMap(item, "schema")["choices"].([]any)
	}
	var out []EditOption
	for _, v := range raw {
		switch opt := v.(type) {
		case map[string]any:
			value := opt["value"]
			label := stringValue(opt["label"])
			if label == "" {
				label = stringValue(value)
			}
			out = append(out, EditOption{Label: label, Value: value})
		case string:
			out = append(out, EditOption{Label: opt, Value: opt})
		}
	}
	return out
}

func firstString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func configEditCliFlag(key string, item map[string]any) string {
	if flag := firstString(
		nestedString(item, "schema", "arg_name"),
		nestedString(item, "schema", "cli_flag"),
		nestedString(item, "render", "arg_name"),
		nestedString(item, "render", "cli_flag"),
		nestedString(item, "render", "flag"),
		nestedString(item, "extensions", "arg_name"),
		nestedString(item, "extensions", "cli_flag"),
		nestedString(item, "extensions", "flag"),
	); flag != "" {
		return flag
	}
	switch key {
	case "service.container_port":
		return "container_port"
	case "service.listen_host":
		return "--host"
	case "deployment.host_port":
		return "host_port"
	case "deployment.served_model_name":
		return "--served-model-name"
	case "model_runtime.max_model_len":
		return "--max-model-len / --context-length / --ctx-size"
	case "model_runtime.gpu_memory_utilization":
		return "--gpu-memory-utilization / --mem-fraction-static"
	case "model_runtime.dtype":
		return "--dtype"
	case "model_runtime.tensor_parallel_size":
		return "--tensor-parallel-size"
	case "model_runtime.pipeline_parallel_size":
		return "--pipeline-parallel-size"
	case "model_runtime.max_num_batched_tokens":
		return "--max-num-batched-tokens"
	case "model_runtime.max_num_seqs":
		return "--max-num-seqs"
	case "model_runtime.kv_cache_dtype":
		return "--kv-cache-dtype"
	case "model_runtime.cpu_offload_gb":
		return "--cpu-offload-gb"
	case "model_runtime.swap_space":
		return "--swap-space"
	case "model_runtime.enforce_eager":
		return "--enforce-eager"
	case "model_runtime.trust_remote_code":
		return "--trust-remote-code"
	case "model_runtime.safetensors_load_strategy":
		return "--safetensors-load-strategy"
	case "model_runtime.download_dir":
		return "--download-dir"
	case "model_runtime.model":
		return "--model"
	case "model_runtime.host":
		return "--host"
	case "model_runtime.port":
		return "--port"
	case "model_runtime.mem_fraction_static":
		return "--mem-fraction-static"
	case "model_runtime.context_length":
		return "--context-length"
	case "model_runtime.gpu_layers":
		return "-ngl"
	case "model_runtime.ctx_size":
		return "--ctx-size"
	}
	return ""
}

func configEditEnvKey(key string, item map[string]any) string {
	if env := firstString(
		nestedString(item, "schema", "env_key"),
		nestedString(item, "render", "env_key"),
		nestedString(item, "extensions", "env_key"),
	); env != "" {
		return env
	}
	if strings.HasPrefix(key, "runtime.env.") {
		return strings.TrimPrefix(key, "runtime.env.")
	}
	return ""
}
