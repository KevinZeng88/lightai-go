package configedit

import "time"

func ApplyEditPatchToConfigSet(set map[string]any, patch ConfigEditPatch, layer, ref string) (map[string]any, error) {
	if patch.Layer == "" {
		patch.Layer = layer
	}
	if err := ValidateEditPatch(set, patch); err != nil {
		return nil, err
	}
	out := NormalizeConfigSet(set)
	items := itemsMap(out)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, field := range patch.Fields {
		internal := field.InternalKey
		if internal == "" {
			internal = field.Key
		}
		item := itemMap(items, internal)
		if len(field.Path) > 0 {
			target := valueMap(item)
			setPathValue(target, field.Path, field.Value)
			if field.Enabled != nil {
				setPathEnabled(item, field.Path, *field.Enabled)
			}
		} else {
			setItemEffectiveValue(item, field.Value)
		}
		required := itemRequired(item)
		if required {
			setItemStateEnabled(item, true)
		} else if field.Enabled != nil && len(field.Path) == 0 {
			setItemStateEnabled(item, *field.Enabled)
		}
		item["source"] = map[string]any{
			"layer":      layer,
			"ref":        ref,
			"reason":     "config_edit_patch",
			"updated_at": now,
		}
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item == nil || !itemRequired(item) {
			continue
		}
		setItemStateEnabled(item, true)
	}
	return out, nil
}

func ResetFieldToDefault(set map[string]any, key string, path []string, layer, ref string) (map[string]any, error) {
	out := NormalizeConfigSet(set)
	items := itemsMap(out)
	item := itemMap(items, key)
	defaultValue := itemDefaultValue(item)
	if len(path) > 0 {
		target := valueMap(item)
		setPathValue(target, path, defaultValue)
	} else {
		setItemEffectiveValue(item, defaultValue)
	}
	setResetSource(item, layer, ref, "reset_to_default")
	return out, nil
}

func ResetFieldToParent(set map[string]any, key string, path []string, layer, ref string) (map[string]any, error) {
	out := NormalizeConfigSet(set)
	items := itemsMap(out)
	item := itemMap(items, key)
	parentValue := itemInheritedValue(item)
	if parentValue == nil {
		parentValue = itemDefaultValue(item)
	}
	if len(path) > 0 {
		target := valueMap(item)
		setPathValue(target, path, parentValue)
	} else {
		setItemEffectiveValue(item, parentValue)
	}
	setResetSource(item, layer, ref, "reset_to_parent")
	return out, nil
}

func setResetSource(item map[string]any, layer, ref, reason string) {
	item["source"] = map[string]any{
		"layer":      layer,
		"ref":        ref,
		"reason":     reason,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

func setPathEnabled(item map[string]any, path []string, enabled bool) {
	if len(path) == 0 {
		return
	}
	fields, _ := item["enabled_fields"].(map[string]any)
	if fields == nil {
		fields = map[string]any{}
		item["enabled_fields"] = fields
	}
	fields[pathKey(path)] = enabled
}

func pathKey(path []string) string {
	if len(path) == 0 {
		return ""
	}
	key := path[0]
	for _, part := range path[1:] {
		key += "." + part
	}
	return key
}

func setPathValue(root map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}
	current := root
	for _, part := range path[:len(path)-1] {
		next, _ := current[part].(map[string]any)
		if next == nil {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[path[len(path)-1]] = value
}
