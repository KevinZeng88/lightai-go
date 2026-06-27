package configedit

// setItemEffectiveValue writes to the tiered ConfigItemValue structure.
// item["value"] must remain {default_value, inherited_value, local_value, effective_value}.
// Updates local_value and effective_value. NEVER overwrites item["value"] with a scalar.
func setItemEffectiveValue(item map[string]any, value any) {
	if item == nil {
		return
	}
	valueTier, _ := item["value"].(map[string]any)
	if valueTier == nil {
		valueTier = map[string]any{}
		item["value"] = valueTier
	}
	valueTier["local_value"] = value
	valueTier["effective_value"] = value
}

// setItemStateEnabled writes to the tiered state structure only.
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
	state["checked"] = enabled
}

// getItemEffectiveValue reads from tiered value structure only.
func getItemEffectiveValue(item map[string]any) (any, bool) {
	if item == nil {
		return nil, false
	}
	if v, ok := item["value"].(map[string]any); ok {
		if ev, ok := v["effective_value"]; ok {
			return ev, true
		}
		if dv, ok := v["default_value"]; ok {
			return dv, true
		}
	}
	return nil, false
}

// getItemStateEnabled reads from tiered state structure only.
func getItemStateEnabled(item map[string]any) bool {
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
