package runplan

import (
	"encoding/json"
	"fmt"
)

// ResourceControlDef describes one resource control knob for a backend.
type ResourceControlDef struct {
	Supported  *bool       `json:"supported,omitempty"` // nil = supported; false = explicitly unsupported
	Arg        string      `json:"arg,omitempty"`
	Type       string      `json:"type,omitempty"` // int, float, string, enum, string_or_int, bool
	Min        *float64    `json:"min,omitempty"`
	Max        *float64    `json:"max,omitempty"`
	Default    interface{} `json:"default,omitempty"`
	Values     []string    `json:"values,omitempty"`      // for enum type
	ValuesHint []string    `json:"values_hint,omitempty"` // suggested values
	Semantics  string      `json:"semantics,omitempty"`
	Reason     string      `json:"reason,omitempty"` // when supported=false
}

// ResourceControlsMap is the full resource_controls definition for a backend version.
// Key is the logical control name (e.g. "gpu_memory_fraction", "ctx_size").
type ResourceControlsMap map[string]ResourceControlDef

// ParseResourceControls parses vendor_options_json and extracts resource_controls.
// Returns nil if vendor_options_json is empty or has no resource_controls key.
func ParseResourceControls(vendorOptionsJSON string) ResourceControlsMap {
	if vendorOptionsJSON == "" || vendorOptionsJSON == "{}" {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(vendorOptionsJSON), &raw); err != nil {
		return nil
	}
	rcRaw, ok := raw["resource_controls"]
	if !ok {
		return nil
	}
	var rcmap ResourceControlsMap
	if err := json.Unmarshal(rcRaw, &rcmap); err != nil {
		return nil
	}
	if len(rcmap) == 0 {
		return nil
	}
	return rcmap
}

// ResourceControlArg returns the CLI arg for a given control name, or empty string if not found/unsupported.
func (rcm ResourceControlsMap) ResourceControlArg(controlName string) string {
	if rcm == nil {
		return ""
	}
	def, ok := rcm[controlName]
	if !ok {
		return ""
	}
	if def.Supported != nil && !*def.Supported {
		return ""
	}
	return def.Arg
}

// IsSupported returns true if the given control is supported.
func (rcm ResourceControlsMap) IsSupported(controlName string) bool {
	if rcm == nil {
		return false
	}
	def, ok := rcm[controlName]
	if !ok {
		return false
	}
	if def.Supported != nil && !*def.Supported {
		return false
	}
	return true
}

// ValidateResourceControlValue validates a value against the control definition.
// Returns empty string if valid, or an error message.
func (rcm ResourceControlsMap) ValidateResourceControlValue(controlName string, value interface{}) string {
	if rcm == nil {
		return ""
	}
	def, ok := rcm[controlName]
	if !ok {
		return ""
	}
	if def.Supported != nil && !*def.Supported {
		return fmt.Sprintf("resource control %q is not supported for this backend: %s", controlName, def.Reason)
	}

	// Type-specific validation
	switch def.Type {
	case "float", "int":
		var fval float64
		switch v := value.(type) {
		case float64:
			fval = v
		case int:
			fval = float64(v)
		case int64:
			fval = float64(v)
		case json.Number:
			fval, _ = v.Float64()
		default:
			return fmt.Sprintf("resource control %q expects numeric value, got %T", controlName, value)
		}
		if def.Min != nil && fval < *def.Min {
			return fmt.Sprintf("resource control %q value %v below minimum %v", controlName, fval, *def.Min)
		}
		if def.Max != nil && fval > *def.Max {
			return fmt.Sprintf("resource control %q value %v above maximum %v", controlName, fval, *def.Max)
		}
	case "enum":
		sval := fmt.Sprintf("%v", value)
		if len(def.Values) > 0 {
			found := false
			for _, v := range def.Values {
				if v == sval {
					found = true
					break
				}
			}
			if !found {
				return fmt.Sprintf("resource control %q value %q not in allowed values %v", controlName, sval, def.Values)
			}
		}
	}
	return ""
}

// BuildResourceControlArgs builds CLI args from deployment parameters and resource_controls definition.
// Only includes args for parameters that exist in the resource_controls definition and are present in params.
func BuildResourceControlArgs(params map[string]interface{}, rcm ResourceControlsMap) []string {
	if rcm == nil || len(params) == 0 {
		return nil
	}
	var args []string
	for name, def := range rcm {
		if def.Supported != nil && !*def.Supported {
			continue
		}
		if def.Arg == "" {
			continue
		}
		// Look up value by control name or by CLI arg name
		val, ok := params[name]
		if !ok {
			// Try without leading dashes
			normalized := def.Arg
			if len(normalized) > 2 && normalized[:2] == "--" {
				normalized = normalized[2:]
			}
			normalized = replaceDash(normalized, "_")
			val, ok = params[normalized]
		}
		if !ok {
			// Try the raw CLI arg name
			val, ok = params[def.Arg]
		}
		if !ok {
			continue
		}
		// Validate
		if msg := rcm.ValidateResourceControlValue(name, val); msg != "" {
			continue // skip invalid values; lint will catch them
		}
		args = append(args, def.Arg, fmt.Sprintf("%v", val))
	}
	return args
}

func replaceDash(s string, replacement string) string {
	result := make([]byte, len(s))
	for i, c := range []byte(s) {
		if c == '-' {
			result[i] = replacement[0]
		} else {
			result[i] = c
		}
	}
	return string(result)
}
