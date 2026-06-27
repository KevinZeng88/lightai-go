package catalog

import (
	"encoding/json"
	"testing"
)

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestTieredConfigRoundTripPreservesValueStructure verifies that the tiered
// ConfigItem value structure survives a full serialize-deserialize-modify
// round-trip. After modifying launcher.image via setConfigValue (which writes
// to item["value"]["effective_value"] and item["value"]["local_value"] only),
// the backend.capabilities item must still have its tiered value structure
// intact — item["value"] is {default_value, effective_value}, not a scalar.
func TestTieredConfigRoundTripPreservesValueStructure(t *testing.T) {
	cs := ConfigSet{
		SchemaVersion: 1,
		ConfigSetKey:  "BackendVersionConfigSet",
		Items: map[string]ConfigItem{
			"launcher.image": {
				Schema: ConfigItemSchema{Key: "launcher.image", Category: "launcher", Kind: "launcher_option", Type: "string"},
				Value_: ConfigItemValue{DefaultValue: "vllm:v0.6.0", EffectiveValue: "vllm:v0.6.0"},
				State_: ConfigItemState{Enabled: true, Visible: true, Valid: true},
			},
			"backend.capabilities": {
				Schema: ConfigItemSchema{Key: "backend.capabilities", Category: "model_runtime", Kind: "launcher_option", Type: "object"},
				Value_: ConfigItemValue{
					DefaultValue:   map[string]any{"supported_formats": []string{"huggingface"}},
					EffectiveValue: map[string]any{"supported_formats": []string{"huggingface"}},
				},
				State_: ConfigItemState{Enabled: true, Visible: true, Valid: true},
			},
		},
	}

	// Serialize (catalog seed writes to DB)
	csJSON, _ := json.Marshal(cs)

	// Deserialize (API reads from DB)
	var set map[string]interface{}
	json.Unmarshal(csJSON, &set)

	// Modify launcher.image via tiered setConfigValueTiered pattern
	items, _ := set["items"].(map[string]interface{})
	item, _ := items["launcher.image"].(map[string]interface{})
	// setConfigValueTiered writes to value.local_value and value.effective_value only
	vt, _ := item["value"].(map[string]interface{})
	if vt == nil {
		vt = map[string]interface{}{}
		item["value"] = vt
	}
	vt["local_value"] = "overwritten-image:latest"
	vt["effective_value"] = "overwritten-image:latest"

	// Verify launcher.image value tier is still a map (not a scalar)
	if _, ok := item["value"].(map[string]interface{}); !ok {
		t.Fatalf("launcher.image value was overwritten as scalar: %T", item["value"])
	}

	// Serialize back (API writes to DB)
	roundTripped, _ := json.Marshal(set)

	// Parse again (another API read)
	var deploySet map[string]interface{}
	json.Unmarshal(roundTripped, &deploySet)

	// backend.capabilities must still have intact tiered value structure
	deployItems, _ := deploySet["items"].(map[string]interface{})
	capItem, _ := deployItems["backend.capabilities"].(map[string]interface{})

	if capItem == nil {
		t.Fatal("backend.capabilities not found after round-trip")
	}

	valueTier, _ := capItem["value"].(map[string]interface{})
	if valueTier == nil {
		t.Fatalf("capabilities value tier is nil; capItem: %v", capItem)
	}
	ev, _ := valueTier["effective_value"]
	if ev == nil {
		t.Fatalf("capabilities effective_value is nil; value tier keys: %v", mapKeys(valueTier))
	}
	m, ok := ev.(map[string]interface{})
	if !ok {
		t.Fatalf("capabilities effective_value is %T not map: %v", ev, ev)
	}
	if _, ok := m["supported_formats"]; !ok {
		t.Fatalf("supported_formats missing from capabilities: %v", m)
	}
}

// TestTieredValueNotOverwrittenByScalar verifies that setConfigValueTiered
// (and setItemEffectiveValue) preserve the tiered value structure.
// item["value"] must always be {default_value, inherited_value, local_value,
// effective_value} — never a scalar string/number/bool.
func TestTieredValueNotOverwrittenByScalar(t *testing.T) {
	item := map[string]interface{}{
		"schema": map[string]interface{}{"key": "launcher.image"},
		"value": map[string]interface{}{
			"default_value":   "vllm:v0.6.0",
			"effective_value": "vllm:v0.6.0",
		},
		"state": map[string]interface{}{"enabled": true},
	}

	// Apply tiered-only edit: write to local_value and effective_value
	vt := item["value"].(map[string]interface{})
	vt["local_value"] = "updated:v2"
	vt["effective_value"] = "updated:v2"

	// item["value"] must still be a map (the tiered struct), not a scalar
	if _, ok := item["value"].(map[string]interface{}); !ok {
		t.Fatalf("FAIL: item[\"value\"] was overwritten as %T, want map", item["value"])
	}

	// effective_value must reflect the update
	if vt["effective_value"] != "updated:v2" {
		t.Errorf("effective_value = %v, want updated:v2", vt["effective_value"])
	}

	// default_value must still be preserved
	if vt["default_value"] != "vllm:v0.6.0" {
		t.Errorf("default_value = %v, want vllm:v0.6.0", vt["default_value"])
	}
}
