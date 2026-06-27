package configedit

import (
	"encoding/json"
)

func NormalizeConfigSet(set map[string]any) map[string]any {
	out := cloneMap(set)
	if out == nil {
		out = map[string]any{}
	}
	if _, ok := out["items"].(map[string]any); !ok {
		out["items"] = map[string]any{}
	}
	return out
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

func itemsMap(set map[string]any) map[string]any {
	items, _ := set["items"].(map[string]any)
	if items == nil {
		items = map[string]any{}
		set["items"] = items
	}
	return items
}

func itemMap(items map[string]any, key string) map[string]any {
	item, _ := items[key].(map[string]any)
	if item == nil {
		item = map[string]any{"code": key}
		items[key] = item
	}
	return item
}

func valueMap(item map[string]any) map[string]any {
	// Tiered: item["value"] may already be the ConfigItemValue wrapper.
	// Navigate to effective_value for the actual value map.
	if vt, ok := item["value"].(map[string]any); ok {
		// Check if this is a tiered wrapper (has effective_value key)
		if ev, ok := vt["effective_value"]; ok {
			if evMap, ok := ev.(map[string]any); ok {
				return evMap
			}
			// effective_value is not a map — return nil so caller can't mutate
			return nil
		}
		// Flat shape — return as-is
		return vt
	}
	// No value — create tiered structure
	vt := map[string]any{}
	item["value"] = map[string]any{"effective_value": vt}
	return vt
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func intValue(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}

func nestedString(item map[string]any, parent, key string) string {
	m, _ := item[parent].(map[string]any)
	if m == nil {
		return ""
	}
	return stringValue(m[key])
}

func nestedMap(item map[string]any, key string) map[string]any {
	m, _ := item[key].(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func hasValue(item map[string]any, key string) bool {
	_, ok := item[key]
	return ok
}

// itemEffectiveValue returns the effective value from tiered or flat shape.
func itemEffectiveValue(item map[string]any) any {
	if item == nil {
		return nil
	}
	if v, ok := item["value"].(map[string]any); ok {
		if ev, ok := v["effective_value"]; ok && ev != nil {
			return ev
		}
		if dv, ok := v["default_value"]; ok {
			return dv
		}
	}
	return item["value"]
}

// itemEnabled returns enabled from state tier only.
func itemEnabled(item map[string]any) bool {
	if item == nil {
		return false
	}
	if state, ok := item["state"].(map[string]any); ok {
		if en, ok := state["enabled"].(bool); ok {
			return en
		}
	}
	return false
}

// itemRequired returns required from schema tier only.
func itemRequired(item map[string]any) bool {
	if item == nil {
		return false
	}
	if schema, ok := item["schema"].(map[string]any); ok {
		if r, ok := schema["required"].(bool); ok {
			return r
		}
	}
	return false
}

// itemReadonly returns readonly from schema tier only.
func itemReadonly(item map[string]any) bool {
	if item == nil {
		return false
	}
	if schema, ok := item["schema"].(map[string]any); ok {
		if r, ok := schema["read_only"].(bool); ok {
			return r
		}
	}
	return false
}

// itemVisibility returns visibility from schema tier only.
func itemVisibility(item map[string]any) string {
	if item == nil {
		return ""
	}
	if schema, ok := item["schema"].(map[string]any); ok {
		if v, ok := schema["visibility"].(string); ok {
			return v
		}
	}
	return ""
}

// itemCategory returns category from schema tier only.
func itemCategory(item map[string]any) string {
	if item == nil {
		return ""
	}
	if schema, ok := item["schema"].(map[string]any); ok {
		if c, ok := schema["category"].(string); ok {
			return c
		}
	}
	return ""
}

// itemLabel returns label from schema tier only.
func itemLabel(item map[string]any) string {
	if item == nil {
		return ""
	}
	if schema, ok := item["schema"].(map[string]any); ok {
		if l, ok := schema["label"].(string); ok {
			return l
		}
	}
	return ""
}

// tieredStringField reads a string field from a tiered sub-object, with flat fallback.
func tieredStringField(item map[string]any, tierKey, fieldKey string) string {
	if item == nil {
		return ""
	}
	if tier, ok := item[tierKey].(map[string]any); ok {
		return stringValue(tier[fieldKey])
	}
	return stringValue(item[fieldKey])
}

// tieredBoolField reads a bool field from a tiered sub-object, with flat fallback.
func tieredBoolField(item map[string]any, tierKey, fieldKey string) bool {
	if item == nil {
		return false
	}
	if tier, ok := item[tierKey].(map[string]any); ok {
		return boolValue(tier[fieldKey])
	}
	return boolValue(item[fieldKey])
}
