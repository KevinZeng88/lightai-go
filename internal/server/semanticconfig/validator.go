package semanticconfig

import "fmt"

func ValidateSnapshotPatch(reg *Registry, snapshot Snapshot, fields []PatchField) error {
	if reg == nil {
		reg = DefaultRegistry()
	}
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.Key)
	}
	if err := ValidatePatchKeys(reg, keys); err != nil {
		return err
	}
	for _, field := range fields {
		def, _ := reg.Get(field.Key)
		if err := validateValueType(def, field.Value); err != nil {
			return err
		}
	}
	return nil
}

func validateValueType(def Definition, value any) error {
	switch def.ValueType {
	case TypeInteger:
		if intFromAny(value, -1) < 0 {
			return fmt.Errorf("field %s expects integer", def.Key)
		}
	case TypeNumber:
		switch value.(type) {
		case int, int64, float64, float32:
		default:
			return fmt.Errorf("field %s expects number", def.Key)
		}
	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %s expects boolean", def.Key)
		}
	case TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field %s expects string", def.Key)
		}
	case TypeArray:
		switch value.(type) {
		case []any, []string:
		default:
			return fmt.Errorf("field %s expects array", def.Key)
		}
	case TypeObject:
		switch value.(type) {
		case map[string]any, map[string]string:
		default:
			return fmt.Errorf("field %s expects object", def.Key)
		}
	}
	if (def.Key == "service.container_port" || def.Key == "deployment.host_port") && !validPort(value) {
		return fmt.Errorf("field %s expects port 1-65535", def.Key)
	}
	return nil
}

func validPort(value any) bool {
	port := intFromAny(value, 0)
	return port >= 1 && port <= 65535
}
