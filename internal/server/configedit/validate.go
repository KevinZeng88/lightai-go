package configedit

import "fmt"

var deploymentProtectedFields = map[string]bool{
	"launcher.image":      true,
	"launcher.command":    true,
	"launcher.entrypoint": true,
	"runtime.model_mount": true,
}

func ValidateEditPatch(set map[string]any, patch ConfigEditPatch) error {
	normalized := NormalizeConfigSet(set)
	items := itemsMap(normalized)
	layer := patch.Layer
	for _, field := range patch.Fields {
		internal := field.InternalKey
		if internal == "" {
			internal = field.Key
		}
		if layer == "deployment" && deploymentProtectedFields[internal] {
			return fmt.Errorf("field %q is protected at deployment layer", internal)
		}
		item, _ := items[internal].(map[string]any)
		if item == nil {
			return fmt.Errorf("unknown config field %q", internal)
		}
		visibility := stringValue(item["visibility"])
		if boolValue(item["readonly"]) || visibility == "internal" || visibility == "hidden" {
			return fmt.Errorf("field %q is readonly", internal)
		}
	}
	return nil
}
