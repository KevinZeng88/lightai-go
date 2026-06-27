package configedit

// setItemEffectiveValue writes to both the tiered value structure and flat compat.
func setItemEffectiveValue(item map[string]any, value any) {
	if item == nil {
		return
	}
	// Tiered write: item["value"]["effective_value"]
	valueTier, _ := item["value"].(map[string]any)
	if valueTier == nil {
		valueTier = map[string]any{}
		item["value"] = valueTier
	}
	valueTier["effective_value"] = value
	// Flat compat: for non-map values, also set item["value"] directly
	// so that old code reading item["value"] gets the scalar.
	if _, isMap := value.(map[string]any); !isMap {
		item["value"] = value
	}
}

// setItemStateEnabled writes to the tiered state structure AND flat compat.
func setItemStateEnabled(item map[string]any, enabled bool) {
	if item == nil {
		return
	}
	state, _ := item["state"].(map[string]any)
	if state == nil {
		state = map[string]any{}
		item["state"] = state
	}
	state["enabled"] = enabled
	// Flat compat
	item["enabled"] = enabled
}

// getItemEffectiveValue reads from tiered value, with flat compat fallback.
func getItemEffectiveValue(item map[string]any) (any, bool) {
	if item == nil {
		return nil, false
	}
	// Tiered: item["value"]["effective_value"]
	if v, ok := item["value"].(map[string]any); ok {
		if ev, ok := v["effective_value"]; ok {
			return ev, true
		}
		if dv, ok := v["default_value"]; ok {
			return dv, true
		}
	}
	// Flat compat: item["value"] (only if it's not a map — a map means tiered)
	if v, ok := item["value"]; ok {
		if _, isMap := v.(map[string]any); !isMap {
			return v, true
		}
	}
	return item["value"], item["value"] != nil
}

// getItemStateEnabled reads from tiered state, with flat compat fallback.
func getItemStateEnabled(item map[string]any) bool {
	if item == nil {
		return false
	}
	if state, ok := item["state"].(map[string]any); ok {
		if en, ok := state["enabled"].(bool); ok {
			return en
		}
	}
	if en, ok := item["enabled"].(bool); ok {
		return en
	}
	return false
}
