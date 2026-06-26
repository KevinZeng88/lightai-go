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
	value, _ := item["value"].(map[string]any)
	if value == nil {
		value = map[string]any{}
		item["value"] = value
	}
	return value
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
