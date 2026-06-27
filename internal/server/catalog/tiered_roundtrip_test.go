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

func TestTieredConfigRoundTripWithOverwrites(t *testing.T) {
	// Simulate: catalog materializes tiered JSON → cloned via setConfigValue → deployment reads capabilities
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

	// Step 1: Serialize (catalog seed writes to DB)
	csJSON, _ := json.Marshal(cs)

	// Step 2: Deserialize as map (API reads from DB)
	var set map[string]interface{}
	json.Unmarshal(csJSON, &set)

	// Step 3: Simulate setConfigValue for launcher.image (what clone/overwrite does)
	items, _ := set["items"].(map[string]interface{})
	item, _ := items["launcher.image"].(map[string]interface{})
	// This is what setConfigValue does — writes flat "value" OVER the tiered structure!
	item["value"] = "overwritten-image:latest"
	item["enabled"] = true
	items["launcher.image"] = item
	set["items"] = items

	// Step 4: Serialize back (API writes to DB)
	roundTripped, _ := json.Marshal(set)

	// Step 5: Parse again (another API read)
	var deploySet map[string]interface{}
	json.Unmarshal(roundTripped, &deploySet)

	// Step 6: Simulate configObject(deployConfigSet, "backend.capabilities")
	deployItems, _ := deploySet["items"].(map[string]interface{})
	capItem, _ := deployItems["backend.capabilities"].(map[string]interface{})

	if capItem == nil {
		t.Fatal("backend.capabilities not found after round-trip")
	}

	// New tiered access
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

	// Also verify launcher.image survived (as flat compat)
	imgItem, _ := deployItems["launcher.image"].(map[string]interface{})
	imgVal := imgItem["value"]
	if imgVal == "overwritten-image:latest" {
		t.Logf("launcher.image value (flat): %v", imgVal)
	} else {
		t.Logf("launcher.image value (tiered?): %v", imgVal)
	}
}

func TestSetConfigValueDestroysTieredShape(t *testing.T) {
	// This test demonstrates the bug: setConfigValue's flat compat write
	// overwrites the tiered "value" object with a scalar.

	// Build a tiered item
	item := map[string]interface{}{
		"schema": map[string]interface{}{"key": "launcher.image"},
		"value": map[string]interface{}{
			"default_value":   "vllm:v0.6.0",
			"effective_value": "vllm:v0.6.0",
		},
		"state": map[string]interface{}{"enabled": true},
	}

	// Simulate setConfigValue's flat compat path
	item["value"] = "overwritten-image:latest" // THIS DESTROYS THE TIERED STRUCTURE

	// Now try to read effective_value
	if v, ok := item["value"].(map[string]interface{}); ok {
		t.Logf("tiered value still accessible: %v", v["effective_value"])
	} else {
		t.Logf("WARNING: value is now %T = %v — tiered structure DESTROYED", item["value"], item["value"])
	}

	// The fix: setConfigValue should ONLY write to item["value"]["effective_value"],
	// NOT to item["value"] directly.
}
