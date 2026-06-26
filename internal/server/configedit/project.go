package configedit

import (
	"sort"
	"strings"
)

func ProjectConfigSetToEditView(input ProjectInput) (ConfigEditView, error) {
	set := NormalizeConfigSet(input.ConfigSet)
	sections := map[string]*EditSection{}
	for key, label := range sectionLabels {
		sections[key] = &EditSection{
			Key:       key,
			Label:     label,
			Order:     sectionOrder[key],
			Advanced:  key == "advanced_raw",
			Collapsed: key == "advanced_raw",
		}
	}

	items := itemsMap(set)
	for code, raw := range items {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		if code == "launcher.docker_options" {
			for _, field := range projectDockerOptions(item, input) {
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

func projectDockerOptions(item map[string]any, input ProjectInput) []EditField {
	value := valueMap(item)
	var fields []EditField
	for _, spec := range dockerFieldSpecs {
		code := "launcher.docker_options." + spec.Path
		dockerItem := cloneMap(item)
		dockerItem["type"] = spec.Type
		dockerItem["value"] = value[spec.Path]
		dockerItem["required"] = false
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
	required := boolValue(item["required"])
	enabled := true
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
	advanced := boolValue(item["advanced"]) || isCapability || isInternalMeta || visibility == "internal" || visibility == "hidden"
	section := sectionFor(key, item)
	if advanced {
		section = "advanced_raw"
	}
	readonly := input.Readonly || boolValue(item["readonly"]) || isCapability || visibility == "readonly" || visibility == "internal" || (input.Layer == "deployment" && deploymentProtectedFields[internalKey])

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

	return EditField{
		Key:          key,
		InternalKey:  internalKey,
		ParentKey:    parentKey(internalKey, path),
		Path:         path,
		Label:        fieldLabel(key, item),
		Help:         firstString(nestedString(item, "render", "help"), nestedString(item, "extensions", "help")),
		Section:      section,
		Group:        firstString(nestedString(item, "render", "group"), nestedString(item, "extensions", "group")),
		Order:        intValue(item["order"]),
		Type:         firstString(stringValue(item["type"]), "string"),
		Widget:       widget,
		Value:        value,
		DefaultValue: item["default_value"],
		Enabled:      enabled,
		HasEnable:    !required && !readonly,
		Required:     required,
		Readonly:     readonly,
		Advanced:     advanced || section == "advanced_raw",
		Visibility:   visibility,
		Options:      optionsFor(item),
		Constraints:  nestedMap(item, "constraints"),
		Source:       nestedMap(item, "source"),
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
