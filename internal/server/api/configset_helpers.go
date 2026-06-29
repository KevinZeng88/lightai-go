package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"lightai-go/internal/server/runplan"
)

func parseConfigSet(raw string) map[string]interface{} {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]interface{}{"schema_version": 1, "items": map[string]interface{}{}}
	}
	if _, ok := out["items"]; !ok {
		out["items"] = map[string]interface{}{}
	}
	return out
}

func configSetItems(set map[string]interface{}) map[string]interface{} {
	items, _ := set["items"].(map[string]interface{})
	if items == nil {
		return map[string]interface{}{}
	}
	return items
}

func configItem(set map[string]interface{}, code string) map[string]interface{} {
	item, _ := configSetItems(set)[code].(map[string]interface{})
	return item
}

// configValue returns the effective value from the tiered ConfigItemValue structure.
// No flat fallback — data must be in tiered shape.
func configValue(set map[string]interface{}, code string, def interface{}) interface{} {
	item := configItem(set, code)
	if item == nil {
		return def
	}
	if v, ok := item["value"].(map[string]interface{}); ok {
		if ev, ok := v["effective_value"]; ok && ev != nil {
			return ev
		}
		if dv, ok := v["default_value"]; ok && dv != nil {
			return dv
		}
	}
	return def
}

func configString(set map[string]interface{}, code, def string) string {
	v := configValue(set, code, def)
	if s, ok := v.(string); ok {
		if strings.TrimSpace(s) == "" {
			return def
		}
		return s
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func configStringSlice(set map[string]interface{}, code string) []string {
	raw := configValue(set, code, []interface{}{})
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		var out []string
		if err := json.Unmarshal([]byte(v), &out); err == nil {
			return out
		}
		return []string{v}
	default:
		return nil
	}
}

func configObject(set map[string]interface{}, code string) map[string]interface{} {
	raw := configValue(set, code, map[string]interface{}{})
	switch v := raw.(type) {
	case map[string]interface{}:
		return v
	case map[string]string:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = val
		}
		return out
	case string:
		var out map[string]interface{}
		if err := json.Unmarshal([]byte(v), &out); err == nil && out != nil {
			return out
		}
	}
	return map[string]interface{}{}
}

func configStringMap(set map[string]interface{}, code string) map[string]string {
	raw := configObject(set, code)
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		if v == nil {
			continue
		}
		out[k] = strings.TrimSpace(fmt.Sprint(v))
	}
	return out
}

func configLauncherKind(set map[string]interface{}, fallback string) string {
	if k := configString(set, "launcher.kind", ""); strings.TrimSpace(k) != "" {
		return k
	}
	if ctx, _ := set["context"].(map[string]interface{}); ctx != nil {
		if k := strings.TrimSpace(fmt.Sprint(ctx["launcher_kind"])); k != "" && k != "<nil>" {
			return k
		}
	}
	if ctx, _ := set["context"].(map[string]string); ctx != nil {
		if k := strings.TrimSpace(ctx["launcher_kind"]); k != "" {
			return k
		}
	}
	return fallback
}

func configDockerSpec(set map[string]interface{}) runplan.DockerSpecInfo {
	raw := configObject(set, "launcher.docker_options")
	var spec runplan.DockerSpecInfo
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &spec)
	return spec
}

func configModelMount(set map[string]interface{}) runplan.ModelMountInfo {
	raw := configObject(set, "runtime.model_mount")
	var mount runplan.ModelMountInfo
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &mount)
	return mount
}

func configHealthCheckPtr(set map[string]interface{}) *runplan.HealthCheckInput {
	raw := configObject(set, "runtime.health")
	if len(raw) == 0 {
		return nil
	}
	var hc runplan.HealthCheckInput
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &hc)
	if hc.Path == "" && hc.ExpectedStatus == 0 && hc.StartupTimeoutSeconds == 0 && hc.IntervalSeconds == 0 && hc.TimeoutSeconds == 0 {
		return nil
	}
	return &hc
}

func configArray(set map[string]interface{}, code string) []interface{} {
	raw := configValue(set, code, []interface{}{})
	switch v := raw.(type) {
	case []interface{}:
		return v
	case []string:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	case string:
		var out []interface{}
		if err := json.Unmarshal([]byte(v), &out); err == nil {
			return out
		}
	}
	return []interface{}{}
}

// configItemEnabled returns the enabled state from the tiered state structure.
func configItemEnabled(item map[string]interface{}) bool {
	if item == nil {
		return false
	}
	if state, ok := item["state"].(map[string]interface{}); ok {
		if en, ok := state["enabled"].(bool); ok {
			return en
		}
	}
	return false
}

// configItemSchemaField returns a string field from the schema tier.
func configItemSchemaField(item map[string]interface{}, field string) string {
	if item == nil {
		return ""
	}
	if schema, ok := item["schema"].(map[string]interface{}); ok {
		return strings.TrimSpace(fmt.Sprint(schema[field]))
	}
	return ""
}

func configSetParameterDefs(set map[string]interface{}) []runplan.ParameterDef {
	items := configSetItems(set)
	out := make([]runplan.ParameterDef, 0, len(items))
	for code, raw := range items {
		item, _ := raw.(map[string]interface{})
		if item == nil {
			continue
		}
		kind := configItemSchemaField(item, "kind")
		if kind != "cli_arg" {
			continue
		}
		var render map[string]interface{}
		if r, ok := item["render"].(map[string]interface{}); ok {
			render = r
		}
		flag := strings.TrimSpace(fmt.Sprint(render["flag"]))
		if flag == "" {
			flag = configItemSchemaField(item, "arg_name")
		}
		if flag == "" {
			if s, ok := item["cli_name"]; ok {
				flag = strings.TrimSpace(fmt.Sprint(s))
			}
		}
		def := runplan.ParameterDef{
			Name:    configItemSchemaField(item, "key"),
			CliName: flag,
			Type:    configItemSchemaField(item, "type"),
			Default: defaultValueFromItem(item),
		}
		if def.Name == "" {
			def.Name = code
		}
		out = append(out, def)
	}
	return out
}

func defaultValueFromItem(item map[string]interface{}) interface{} {
	if v, ok := item["value"].(map[string]interface{}); ok {
		return v["default_value"]
	}
	return nil
}

func configSetParameterValues(set map[string]interface{}) []runplan.ParameterValue {
	items := configSetItems(set)
	out := make([]runplan.ParameterValue, 0, len(items))
	for code, raw := range items {
		item, _ := raw.(map[string]interface{})
		if item == nil {
			continue
		}
		kind := configItemSchemaField(item, "kind")
		if kind != "cli_arg" && kind != "cli_args" && kind != "env" {
			continue
		}

		var render map[string]interface{}
		if r, ok := item["render"].(map[string]interface{}); ok {
			render = r
		}
		target := configItemSchemaField(item, "target")
		if target == "" {
			target = strings.TrimSpace(fmt.Sprint(render["target"]))
		}
		flag := configItemSchemaField(item, "arg_name")
		if flag == "" {
			flag = strings.TrimSpace(fmt.Sprint(render["flag"]))
		}
		envName := configItemSchemaField(item, "env_name")
		if envName == "" {
			envName = strings.TrimSpace(fmt.Sprint(render["env_name"]))
		}
		style := strings.TrimSpace(fmt.Sprint(render["style"]))
		if kind == "env" && envName == "" {
			continue
		}
		enabled := configItemEnabled(item)
		value := configValue(set, code, defaultValueFromItem(item))
		if kind == "cli_args" && strings.TrimSpace(fmt.Sprint(value)) == "" {
			continue
		}
		pv := runplan.ParameterValue{
			Key:         code,
			Type:        configItemSchemaField(item, "type"),
			Target:      target,
			CliName:     flag,
			EnvName:     envName,
			RenderStyle: style,
			Enabled:     enabled,
			Value:       value,
			Default:     defaultValueFromItem(item),
			Source:      "config_set",
		}
		if kind == "cli_args" && pv.RenderStyle == "" {
			pv.RenderStyle = "raw_lines"
		}
		if kind == "env" && pv.Target == "" {
			pv.Target = "env"
		}
		if (kind == "cli_arg" || kind == "cli_args") && pv.Target == "" {
			pv.Target = "cli"
		}
		out = append(out, pv)
	}
	return out
}

// setConfigValueTiered writes to the tiered ConfigItemValue structure.
// item["value"] must remain {default_value, inherited_value, local_value, effective_value}.
// This function updates local_value and effective_value. It NEVER overwrites
// item["value"] with a scalar.
func setConfigValueTiered(item map[string]interface{}, value interface{}) {
	if item == nil {
		return
	}
	valueTier, _ := item["value"].(map[string]interface{})
	if valueTier == nil {
		valueTier = map[string]interface{}{}
		item["value"] = valueTier
	}
	valueTier["local_value"] = value
	valueTier["effective_value"] = value
}

// setConfigEnabledTiered writes enabled into the tiered shape: item["state"]["enabled"]
func setConfigEnabledTiered(item map[string]interface{}, enabled bool) {
	if item == nil {
		return
	}
	state, _ := item["state"].(map[string]interface{})
	if state == nil {
		state = map[string]interface{}{}
		item["state"] = state
	}
	state["enabled"] = enabled
}

func setConfigValue(set map[string]interface{}, code string, value interface{}, layer, ref, reason string) {
	items := configSetItems(set)
	item, _ := items[code].(map[string]interface{})
	if item == nil {
		item = map[string]interface{}{}
		item["schema"] = map[string]interface{}{"key": code}
		item["value"] = map[string]interface{}{}
		item["state"] = map[string]interface{}{}
	}
	setConfigValueTiered(item, value)
	setConfigEnabledTiered(item, true)
	// Write provenance to tiered provenance structure only
	if prov, _ := item["provenance"].(map[string]interface{}); prov != nil {
		prov["value_source"] = layer
		prov["last_value_layer"] = layer
		prov["last_value_owner_id"] = ref
	} else {
		item["provenance"] = map[string]interface{}{
			"value_source":        layer,
			"last_value_layer":    layer,
			"last_value_owner_id": ref,
		}
	}
	items[code] = item
	set["items"] = items
}

func setConfigEnabled(set map[string]interface{}, code string, enabled bool, layer, ref, reason string) {
	items := configSetItems(set)
	item, _ := items[code].(map[string]interface{})
	if item == nil {
		item = map[string]interface{}{}
	}
	setConfigEnabledTiered(item, enabled)
	items[code] = item
	set["items"] = items
}

func applyConfigOverrides(set map[string]interface{}, overrides map[string]interface{}, layer, ref string) {
	if len(overrides) == 0 {
		return
	}
	for key, value := range overrides {
		switch key {
		case "parameter_values":
			values, _ := value.([]interface{})
			for _, raw := range values {
				item, _ := raw.(map[string]interface{})
				if item == nil {
					continue
				}
				code := strings.TrimSpace(fmt.Sprint(item["key"]))
				if code == "" {
					code = strings.TrimSpace(fmt.Sprint(item["code"]))
				}
				if code == "" {
					code = strings.TrimSpace(fmt.Sprint(item["name"]))
				}
				if code == "" {
					code = strings.TrimSpace(fmt.Sprint(item["cli_name"]))
				}
				if code == "" {
					continue
				}
				if v, ok := item["value"]; ok {
					setConfigValue(set, code, v, layer, ref, "config_override")
				}
				if enabled, ok := item["enabled"].(bool); ok {
					setConfigEnabled(set, code, enabled, layer, ref, "config_override")
				}
			}
		case "disabled_parameters":
			values, _ := value.([]interface{})
			for _, raw := range values {
				item, _ := raw.(map[string]interface{})
				if item == nil {
					continue
				}
				code := strings.TrimSpace(fmt.Sprint(item["key"]))
				if code == "" {
					code = strings.TrimSpace(fmt.Sprint(item["code"]))
				}
				if code != "" {
					setConfigEnabled(set, code, false, layer, ref, "config_override_disabled")
				}
			}
		case "env":
			current := configObject(set, "runtime.env")
			for envKey, envVal := range mapFromAny(value) {
				current[envKey] = envVal
			}
			setConfigValue(set, "runtime.env", current, layer, ref, "config_override")
		default:
			if entry, ok := value.(map[string]interface{}); ok {
				if v, exists := entry["value"]; exists {
					setConfigValue(set, key, v, layer, ref, "config_override")
					if enabled, ok := entry["enabled"].(bool); ok {
						setConfigEnabled(set, key, enabled, layer, ref, "config_override")
					}
					continue
				}
			}
			setConfigValue(set, key, value, layer, ref, "config_override")
		}
	}
}

func rejectLegacyDeploymentPayload(req map[string]interface{}) string {
	for _, key := range []string{
		"backend_runtime_id",
		"parameters_json",
		"parameter_values_json",
		"disabled_parameters_json",
		"env_overrides_json",
		"config_snapshot_json",
		"config_overrides_json",
		"source_metadata_json",
		"config_set_json",
	} {
		if _, ok := req[key]; ok {
			return key
		}
	}
	return ""
}

func copyConfigSet(raw string) map[string]interface{} {
	var out map[string]interface{}
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		out = map[string]interface{}{"schema_version": 1, "items": map[string]interface{}{}}
	}
	return out
}

func configSetJSON(set map[string]interface{}) string {
	b, _ := json.Marshal(set)
	return string(b)
}

func configSourceMetadata(raw string) map[string]interface{} {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

// validateTieredConfigSet checks that a caller-provided ConfigSet map has strict tiered
// shape. Returns nil if valid, or an error describing the first violation found.
// Requirements for every item:
//   - must have "schema" (map) with at least "key"
//   - must have "value" (map)
//   - must have "state" (map)
//   - must NOT have top-level "code", "value" scalar, "enabled", "default_value", "required"
//   - must NOT have "enabled_fields"
func validateTieredConfigSet(set map[string]interface{}) error {
	if set == nil || len(set) == 0 {
		return nil
	}
	items, _ := set["items"].(map[string]interface{})
	for code, raw := range items {
		item, _ := raw.(map[string]interface{})
		if item == nil {
			return fmt.Errorf("config_set item %q is not a valid object", code)
		}
		// Reject old flat top-level fields
		if _, ok := item["code"]; ok {
			return fmt.Errorf("config_set item %q has flat \"code\" field; items must use {\"schema\":{\"key\":...}} (tiered shape)", code)
		}
		if _, ok := item["enabled"]; ok {
			return fmt.Errorf("config_set item %q has flat \"enabled\" field; items must use {\"state\":{\"enabled\":...}} (tiered shape)", code)
		}
		if _, ok := item["default_value"]; ok {
			return fmt.Errorf("config_set item %q has flat \"default_value\" field; items must use {\"value\":{\"default_value\":...}} (tiered shape)", code)
		}
		if _, ok := item["required"]; ok {
			return fmt.Errorf("config_set item %q has flat \"required\" field; items must use {\"schema\":{\"required\":...}} (tiered shape)", code)
		}
		if _, ok := item["enabled_fields"]; ok {
			return fmt.Errorf("config_set item %q has legacy \"enabled_fields\" field; not accepted in final tiered shape", code)
		}
		// Reject scalar "value" — must be a map (the tiered wrapper)
		if v, ok := item["value"]; ok {
			if _, isMap := v.(map[string]interface{}); !isMap {
				return fmt.Errorf("config_set item %q has scalar \"value\" field; items must use {\"value\":{\"effective_value\":...}} (tiered shape)", code)
			}
		}
		// Require schema, value, state tiers
		if _, ok := item["schema"].(map[string]interface{}); !ok {
			return fmt.Errorf("config_set item %q is missing \"schema\" tier (tiered shape required)", code)
		}
		if _, ok := item["value"].(map[string]interface{}); !ok {
			return fmt.Errorf("config_set item %q is missing \"value\" tier (tiered shape required)", code)
		}
		if _, ok := item["state"].(map[string]interface{}); !ok {
			return fmt.Errorf("config_set item %q is missing \"state\" tier (tiered shape required)", code)
		}
	}
	return nil
}

func mapFromAny(v interface{}) map[string]interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return t
	case json.RawMessage:
		var out map[string]interface{}
		if err := json.Unmarshal(t, &out); err == nil {
			return out
		}
	case string:
		var out map[string]interface{}
		if err := json.Unmarshal([]byte(t), &out); err == nil {
			return out
		}
	}
	return map[string]interface{}{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
