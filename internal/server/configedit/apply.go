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
		} else {
			item["value"] = field.Value
		}
		required := boolValue(item["required"])
		if required {
			item["enabled"] = true
		} else if field.Enabled != nil {
			item["enabled"] = *field.Enabled
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
		if item == nil || !boolValue(item["required"]) {
			continue
		}
		item["enabled"] = true
	}
	return out, nil
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
