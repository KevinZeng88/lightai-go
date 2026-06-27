package configedit

import (
	"fmt"
	"strings"
)

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
		if layer != "node_backend_runtime" && isDirectLegacyPatchKey(field.Key) {
			return fmt.Errorf("direct legacy key patch %q is not allowed", field.Key)
		}
		internal := field.InternalKey
		if internal == "" {
			internal = field.Key
		}
		// --- Layer scope enforcement ---
		// Reject fields hidden at this layer.
		if isLayerHidden(field.Key, layer) {
			return fmt.Errorf("field %q is hidden at layer %q", field.Key, layer)
		}
		if internal != field.Key && isLayerHidden(internal, layer) {
			return fmt.Errorf("field %q is hidden at layer %q", internal, layer)
		}
		// Reject fields that are readonly at this layer.
		if isLayerReadonly(field.Key, layer) {
			return fmt.Errorf("field %q is readonly at layer %q", field.Key, layer)
		}
		if internal != field.Key && isLayerReadonly(internal, layer) {
			return fmt.Errorf("field %q is readonly at layer %q", internal, layer)
		}
		// --- Deployment protected fields ---
		if layer == "deployment" && deploymentProtectedFields[internal] {
			return fmt.Errorf("field %q is protected at deployment layer", internal)
		}
		item, _ := items[internal].(map[string]any)
		if item == nil {
			return fmt.Errorf("unknown config field %q", internal)
		}
		visibility := itemVisibility(item)
		if itemReadonly(item) || visibility == "internal" || visibility == "hidden" {
			return fmt.Errorf("field %q is readonly", internal)
		}
	}
	return nil
}

func isDirectLegacyPatchKey(key string) bool {
	return strings.HasPrefix(key, "backend.arg.") ||
		strings.HasPrefix(key, "backend.common.") ||
		key == "launcher.listen_host" ||
		key == "launcher.container_port"
}
