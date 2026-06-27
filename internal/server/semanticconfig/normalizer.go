package semanticconfig

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func NormalizeConfigSet(reg *Registry, set map[string]any) (Snapshot, error) {
	if reg == nil {
		reg = DefaultRegistry()
	}
	normalized := Snapshot{
		SchemaVersion: intFromAny(set["schema_version"], 1),
		Context:       stringMap(set["context"]),
		Items:         map[string]SnapshotItem{},
	}
	rawItems, _ := set["items"].(map[string]any)
	for rawKey, raw := range rawItems {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		code := stringFromAny(item["code"])
		if code == "" {
			code = rawKey
		}
		if code == "launcher.docker_options" {
			normalizeDockerOptions(reg, &normalized, item)
			continue
		}
		canonical, ok := reg.CanonicalKey(code)
		if !ok {
			if strings.HasPrefix(code, "backend.arg.") {
				normalized.Warnings = append(normalized.Warnings, Warning{
					Code:      WarningLegacyNormalized,
					LegacyKey: code,
					Message:   fmt.Sprintf("legacy backend arg %q has no semantic mapping", code),
				})
			}
			continue
		}
		addNormalizedItem(reg, &normalized, canonical, code, item)
	}
	return normalized, nil
}

func ValidatePatchKeys(reg *Registry, keys []string) error {
	if reg == nil {
		reg = DefaultRegistry()
	}
	for _, key := range keys {
		if reg.IsLegacyKey(key) || strings.HasPrefix(key, "backend.arg.") || strings.HasPrefix(key, "backend.common.") || strings.HasPrefix(key, "launcher.listen_") || strings.HasPrefix(key, "launcher.container_") {
			return fmt.Errorf("direct legacy key patch %q is not allowed; use canonical semantic key", key)
		}
		if _, ok := reg.Get(key); !ok {
			return fmt.Errorf("unknown canonical key %q", key)
		}
	}
	return nil
}

func normalizeDockerOptions(reg *Registry, snapshot *Snapshot, item map[string]any) {
	values, _ := item["value"].(map[string]any)
	enabledFields, _ := item["enabled_fields"].(map[string]any)
	for subKey, value := range values {
		legacy := "launcher.docker_options." + subKey
		canonical, ok := reg.CanonicalKey(legacy)
		if !ok {
			continue
		}
		subItem := cloneMap(item)
		subItem["code"] = legacy
		subItem["value"] = value
		subItem["default_value"] = value
		subItem["enabled"] = boolFromAny(enabledFields[subKey], false)
		addNormalizedItem(reg, snapshot, canonical, legacy, subItem)
	}
}

func addNormalizedItem(reg *Registry, snapshot *Snapshot, canonical, sourceKey string, item map[string]any) {
	def, ok := reg.Get(canonical)
	if !ok {
		return
	}
	value := item["value"]
	defaultValue := item["default_value"]
	if defaultValue == nil {
		defaultValue = value
	}
	enabled := boolFromAny(item["enabled"], false)
	if boolFromAny(item["required"], false) {
		enabled = true
	}
	next := SnapshotItem{
		Key:          canonical,
		Owner:        def.Owner,
		Type:         def.ValueType,
		DisplayTier:  def.DisplayTier,
		Label:        def.Label,
		Value:        value,
		DefaultValue: defaultValue,
		Enabled:      enabled,
		Source:       anyMap(item["source"]),
	}
	if existing, ok := snapshot.Items[canonical]; ok {
		if !reflect.DeepEqual(existing.Value, next.Value) {
			warning := Warning{
				Code:        WarningConflict,
				SemanticKey: canonical,
				LegacyKey:   sourceKey,
				Message:     fmt.Sprintf("conflicting values for %s from %s", canonical, sourceKey),
			}
			if sourceKey == canonical {
				next.Warnings = append(next.Warnings, existing.Warnings...)
				next.Warnings = append(next.Warnings, warning)
				snapshot.Warnings = append(snapshot.Warnings, warning)
				snapshot.Items[canonical] = next
				return
			}
			existing.Warnings = append(existing.Warnings, warning)
			snapshot.Warnings = append(snapshot.Warnings, warning)
			snapshot.Items[canonical] = existing
			return
		}
		return
	}
	if sourceKey != canonical {
		warning := Warning{
			Code:        WarningLegacyNormalized,
			SemanticKey: canonical,
			LegacyKey:   sourceKey,
			Message:     fmt.Sprintf("normalized legacy key %s to %s", sourceKey, canonical),
		}
		next.Warnings = append(next.Warnings, warning)
		snapshot.Warnings = append(snapshot.Warnings, warning)
	}
	snapshot.Items[canonical] = next
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	b, _ := json.Marshal(in)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func intFromAny(v any, fallback int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return fallback
	}
}

func boolFromAny(v any, fallback bool) bool {
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func stringMap(v any) map[string]string {
	out := map[string]string{}
	switch m := v.(type) {
	case map[string]string:
		for k, val := range m {
			out[k] = val
		}
	case map[string]any:
		for k, val := range m {
			if s, ok := val.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func anyMap(v any) map[string]any {
	out, _ := v.(map[string]any)
	if out == nil {
		return nil
	}
	return out
}
