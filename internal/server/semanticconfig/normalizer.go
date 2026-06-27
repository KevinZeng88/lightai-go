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
		// Read key from tiered schema; fall back to item key in map
		code := tieredString(item, "schema", "key")
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
	// Read docker options from tiered value.effective_value
	var values map[string]any
	if vt, ok := item["value"].(map[string]any); ok {
		if ev, ok := vt["effective_value"].(map[string]any); ok {
			values = ev
		}
	}
	if values == nil {
		return
	}
	// Read per-subfield enabled state from the parent item's state tier,
	// or from individual docker subfield ConfigItems if present.
	parentEnabled := tieredBool(item, "state", "enabled")
	for subKey, value := range values {
		legacy := "launcher.docker_options." + subKey
		canonical, ok := reg.CanonicalKey(legacy)
		if !ok {
			continue
		}
		subItem := cloneMap(item)
		// Build tiered sub-item
		ensureTieredValue(subItem, value)
		ensureTieredState(subItem, parentEnabled)
		subSchema, _ := subItem["schema"].(map[string]any)
		if subSchema == nil {
			subSchema = map[string]any{}
			subItem["schema"] = subSchema
		}
		subSchema["key"] = legacy
		addNormalizedItem(reg, snapshot, canonical, legacy, subItem)
	}
}

// tieredString reads a string field from a tiered sub-object (e.g. schema.key).
func tieredString(item map[string]any, tierKey, fieldKey string) string {
	if tier, ok := item[tierKey].(map[string]any); ok {
		return stringFromAny(tier[fieldKey])
	}
	return ""
}

// tieredBool reads a bool field from a tiered sub-object.
func tieredBool(item map[string]any, tierKey, fieldKey string) bool {
	if tier, ok := item[tierKey].(map[string]any); ok {
		return boolFromAny(tier[fieldKey], false)
	}
	return false
}

// tieredValue reads a value from a tiered value sub-object by field key.
func tieredValue(item map[string]any, fieldKey string) any {
	if vt, ok := item["value"].(map[string]any); ok {
		return vt[fieldKey]
	}
	return nil
}

// ensureTieredValue sets effective_value and default_value in the tiered value structure.
func ensureTieredValue(item map[string]any, value any) {
	vt, _ := item["value"].(map[string]any)
	if vt == nil {
		vt = map[string]any{}
		item["value"] = vt
	}
	vt["effective_value"] = value
	if vt["default_value"] == nil {
		vt["default_value"] = value
	}
}

// ensureTieredState sets enabled in the tiered state structure.
func ensureTieredState(item map[string]any, enabled bool) {
	st, _ := item["state"].(map[string]any)
	if st == nil {
		st = map[string]any{}
		item["state"] = st
	}
	st["enabled"] = enabled
}

func addNormalizedItem(reg *Registry, snapshot *Snapshot, canonical, sourceKey string, item map[string]any) {
	def, ok := reg.Get(canonical)
	if !ok {
		return
	}
	// Read from tiered value structure
	value := tieredValue(item, "effective_value")
	defaultValue := tieredValue(item, "default_value")
	if defaultValue == nil {
		defaultValue = value
	}
	// Read from tiered state structure
	enabled := tieredBool(item, "state", "enabled")
	// Read required from tiered schema structure
	if tieredBool(item, "schema", "required") {
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
