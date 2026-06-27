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
