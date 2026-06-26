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

func configValue(set map[string]interface{}, code string, def interface{}) interface{} {
	item := configItem(set, code)
	if item == nil {
		return def
	}
	if v, ok := item["value"]; ok && v != nil {
		return v
	}
	if v, ok := item["default_value"]; ok && v != nil {
		return v
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

func configSetParameterDefs(set map[string]interface{}) []runplan.ParameterDef {
	items := configSetItems(set)
	out := make([]runplan.ParameterDef, 0, len(items))
	for code, raw := range items {
		item, _ := raw.(map[string]interface{})
		if item == nil {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(item["kind"])) != "cli_arg" {
			continue
		}
		render, _ := item["render"].(map[string]interface{})
		flag := strings.TrimSpace(fmt.Sprint(render["flag"]))
		if flag == "" {
			flag = strings.TrimSpace(fmt.Sprint(item["cli_name"]))
		}
		def := runplan.ParameterDef{
			Name:    strings.TrimSpace(fmt.Sprint(item["name"])),
			CliName: flag,
			Type:    strings.TrimSpace(fmt.Sprint(item["type"])),
			Default: item["default_value"],
		}
		if def.Name == "" {
			def.Name = code
		}
		out = append(out, def)
	}
	return out
}

func configSetParameterValues(set map[string]interface{}) []runplan.ParameterValue {
	items := configSetItems(set)
	out := make([]runplan.ParameterValue, 0, len(items))
	for code, raw := range items {
		item, _ := raw.(map[string]interface{})
		if item == nil {
			continue
		}
		kind := strings.TrimSpace(fmt.Sprint(item["kind"]))
		if kind != "cli_arg" && kind != "cli_args" && kind != "env" {
			continue
		}
		render, _ := item["render"].(map[string]interface{})
		target := strings.TrimSpace(fmt.Sprint(render["target"]))
		flag := strings.TrimSpace(fmt.Sprint(render["flag"]))
		envName := strings.TrimSpace(fmt.Sprint(render["env_name"]))
		style := strings.TrimSpace(fmt.Sprint(render["style"]))
		if kind == "env" && envName == "" {
			continue
		}
		enabled, _ := item["enabled"].(bool)
		value := configValue(set, code, item["default_value"])
		if kind == "cli_args" && strings.TrimSpace(fmt.Sprint(value)) == "" {
			continue
		}
		pv := runplan.ParameterValue{
			Key:         code,
			Type:        strings.TrimSpace(fmt.Sprint(item["type"])),
			Target:      target,
			CliName:     flag,
			EnvName:     envName,
			RenderStyle: style,
			Enabled:     enabled,
			Value:       value,
			Default:     item["default_value"],
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

func setConfigValue(set map[string]interface{}, code string, value interface{}, layer, ref, reason string) {
	items := configSetItems(set)
	item, _ := items[code].(map[string]interface{})
	if item == nil {
		item = map[string]interface{}{"code": code}
	}
	item["value"] = value
	item["enabled"] = true
	item["source"] = map[string]string{"layer": layer, "ref": ref, "reason": reason}
	items[code] = item
	set["items"] = items
}

func setConfigEnabled(set map[string]interface{}, code string, enabled bool, layer, ref, reason string) {
	items := configSetItems(set)
	item, _ := items[code].(map[string]interface{})
	if item == nil {
		item = map[string]interface{}{"code": code}
	}
	item["enabled"] = enabled
	item["source"] = map[string]string{"layer": layer, "ref": ref, "reason": reason}
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
