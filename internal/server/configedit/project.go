package configedit

import (
	"sort"
	"strings"

	"lightai-go/internal/server/semanticconfig"
)

func ProjectConfigSetToEditView(input ProjectInput) (ConfigEditView, error) {
	set := NormalizeConfigSet(input.ConfigSet)
	sections := map[string]*EditSection{}
	for key, label := range sectionLabels {
		sections[key] = &EditSection{
			Key:       key,
			Label:     label,
			Order:     sectionOrder[key],
			Advanced:  key == "advanced_raw" || key == "advanced_parameters" || key == "expert_parameters",
			Collapsed: key == "advanced_raw" || key == "advanced_parameters" || key == "expert_parameters",
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
				sections[field.Section].Fields = append(sections[field.Section].Fields, field)
			}
			continue
		}
		field := projectItem(code, code, nil, item, input)
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

	return ConfigEditView{
		Layer:      input.Layer,
		ObjectID:   input.ObjectID,
		ObjectKind: input.ObjectKind,
		Readonly:   input.Readonly,
		Sections:   outSections,
		Diagnostics: ConfigEditDiagnostics{
			RawConfigSet: set,
		},
		Metadata: map[string]any{
			"object_label": input.ObjectLabel,
			"mode":         input.Mode,
		},
	}, nil
}

func hideFromOrdinaryFlow(key string, item map[string]any, layer string) bool {
	if layer != "node_backend_runtime" && layer != "deployment" {
		return false
	}
	visibility := stringValue(item["visibility"])
	if visibility == "internal" || visibility == "hidden" || visibility == "deprecated" {
		return true
	}
	if boolValue(item["deprecated"]) {
		return true
	}
	category := stringValue(item["category"])
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
	currentValue := item["value"]
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
	var fields []EditField
	for _, spec := range dockerFieldSpecs {
		code := "launcher.docker_options." + spec.Path
		dockerItem := cloneMap(item)
		dockerItem["type"] = spec.Type
		dockerItem["value"] = value[spec.Path]
		dockerItem["required"] = false
		// Docker sub-fields keep prefilled values separate from their enable toggle.
		// The parent object only stores values, so sub-field projection defaults to
		// unchecked unless a future schema explicitly carries a per-field enabled bit.
		dockerItem["enabled"] = false
		field := projectItem(code, "launcher.docker_options", []string{spec.Path}, dockerItem, input)
		field.Section = spec.Section
		field.Widget = spec.Widget
		field.Order = spec.Order
		field.Label = fieldLabel(code, dockerItem)
		fields = append(fields, field)
	}
	return fields
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
	required := boolValue(item["required"])

	// Enabled is an explicit saved toggle. Defaults/values/tier/visibility only
	// prefill or organize UI fields; they must not opt parameters into RunPlan.
	enabled := false
	if hasValue(item, "enabled") {
		enabled = boolValue(item["enabled"])
	}
	if required {
		enabled = true
	}

	// Determine if this is a capability/internal field that should be forced to readonly summary.
	isCapability := capabilityLikeCodes[key] || strings.Contains(key, "capabilities") || strings.Contains(key, "supported_config")
	isInternalMeta := strings.HasPrefix(key, "internal.") || strings.HasPrefix(key, "source_metadata.") || strings.HasPrefix(key, "resolver.")

	visibility := stringValue(item["visibility"])
	deprecated := boolValue(item["deprecated"]) || visibility == "deprecated"
	debugLike := stringValue(item["category"]) == "debug" || strings.Contains(key, "debug") || strings.Contains(key, "profile")
	tier := displayTierFor(displayKey, item, hasDef, string(def.DisplayTier))
	advanced := boolValue(item["advanced"]) || tier == "advanced" || tier == "expert" || isCapability || isInternalMeta || visibility == "internal" || visibility == "hidden" || deprecated || debugLike
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
		if section != "advanced_parameters" && section != "expert_parameters" {
			section = "advanced_raw"
		}
	}

	// Readonly from input, item config, layer scope, or capability status.
	readonly := input.Readonly || boolValue(item["readonly"]) || isCapability || visibility == "readonly" || visibility == "internal"
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
	field := EditField{
		Key:             displayKey,
		InternalKey:     internalKey,
		SemanticKey:     semanticKey,
		ParentKey:       parentKey(internalKey, path),
		Path:            path,
		Label:           fieldLabel(displayKey, item),
		Help:            firstString(nestedString(item, "render", "help"), nestedString(item, "extensions", "help")),
		Section:         section,
		Group:           firstString(nestedString(item, "render", "group"), nestedString(item, "extensions", "group")),
		Order:           intValue(item["order"]),
		Type:            firstString(stringValue(item["type"]), "string"),
		Widget:          widget,
		Value:           value,
		DefaultValue:    item["default_value"],
		Enabled:         enabled,
		HasEnable:       !required && !readonly,
		Required:        required,
		Readonly:        readonly,
		Advanced:        advanced || section == "advanced_raw",
		Visibility:      visibility,
		Options:         optionsFor(item),
		Constraints:     nestedMap(item, "constraints"),
		Source:          nestedMap(item, "source"),
		CopiedFrom:      firstString(stringValue(item["copied_from"]), stringValue(item["copiedFrom"])),
		Dirty:           boolValue(item["dirty"]),
		Warnings:        warnings,
		Diagnostic:      advanced || visibility == "internal" || visibility == "hidden",
		OriginalValue:   value,
		OriginalEnabled: enabled,
	}
	if hasDef {
		field.Owner = string(def.Owner)
		field.Tier = tier
		field.Label = firstString(def.Label, field.Label)
	} else {
		field.Tier = tier
	}
	return field
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
	if commonRuntimeArgs[key] || boolValue(item["visible_by_default"]) {
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
	if v, ok := item["value"]; ok {
		return v
	}
	return item["default_value"]
}

func optionsFor(item map[string]any) []EditOption {
	raw, _ := nestedMap(item, "render")["options"].([]any)
	if len(raw) == 0 {
		raw, _ = nestedMap(item, "constraints")["options"].([]any)
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
