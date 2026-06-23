package runplan

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseResourceControlsVLLM(t *testing.T) {
	vendorJSON := `{
		"resource_controls": {
			"gpu_memory_fraction": {
				"supported": true,
				"arg": "--gpu-memory-utilization",
				"type": "float",
				"min": 0.1,
				"max": 0.95,
				"default": 0.9,
				"semantics": "per_instance_backend_allocation_budget"
			},
			"max_model_len": {"arg": "--max-model-len", "type": "int"}
		}
	}`
	rcm := ParseResourceControls(vendorJSON)
	if rcm == nil {
		t.Fatal("expected non-nil resource controls")
	}
	if len(rcm) != 2 {
		t.Fatalf("expected 2 controls, got %d", len(rcm))
	}
	if !rcm.IsSupported("gpu_memory_fraction") {
		t.Error("gpu_memory_fraction should be supported")
	}
	if rcm.ResourceControlArg("gpu_memory_fraction") != "--gpu-memory-utilization" {
		t.Errorf("unexpected arg: %s", rcm.ResourceControlArg("gpu_memory_fraction"))
	}
}

func TestParseResourceControlsLlamaCpp(t *testing.T) {
	vendorJSON := `{
		"resource_controls": {
			"gpu_memory_fraction": {
				"supported": false,
				"reason": "llama.cpp does not expose a vLLM-style GPU memory fraction."
			},
			"gpu_layers": {"arg": "--n-gpu-layers", "type": "string_or_int"},
			"ctx_size": {"arg": "--ctx-size", "type": "int"}
		}
	}`
	rcm := ParseResourceControls(vendorJSON)
	if rcm == nil {
		t.Fatal("expected non-nil resource controls")
	}
	if rcm.IsSupported("gpu_memory_fraction") {
		t.Error("gpu_memory_fraction should NOT be supported for llama.cpp")
	}
	// gpu_layers has no Supported field → defaults to supported
	if !rcm.IsSupported("gpu_layers") {
		t.Error("gpu_layers should be supported (no supported=false)")
	}
	if !rcm.IsSupported("ctx_size") {
		t.Error("ctx_size should be supported (no supported=false)")
	}
}

func TestValidateResourceControlMinMax(t *testing.T) {
	vendorJSON := `{
		"resource_controls": {
			"gpu_memory_fraction": {
				"arg": "--gpu-memory-utilization",
				"type": "float",
				"min": 0.1,
				"max": 0.95
			}
		}
	}`
	rcm := ParseResourceControls(vendorJSON)

	// Valid value
	if msg := rcm.ValidateResourceControlValue("gpu_memory_fraction", 0.5); msg != "" {
		t.Errorf("expected valid, got: %s", msg)
	}

	// Below min
	if msg := rcm.ValidateResourceControlValue("gpu_memory_fraction", 0.05); msg == "" {
		t.Error("expected error for value below min")
	}

	// Above max
	if msg := rcm.ValidateResourceControlValue("gpu_memory_fraction", 0.99); msg == "" {
		t.Error("expected error for value above max")
	}
}

func TestValidateResourceControlEnum(t *testing.T) {
	vendorJSON := `{
		"resource_controls": {
			"attention_backend": {
				"arg": "--attention-backend",
				"type": "enum",
				"values": ["auto", "flashinfer", "triton"]
			}
		}
	}`
	rcm := ParseResourceControls(vendorJSON)

	if msg := rcm.ValidateResourceControlValue("attention_backend", "flashinfer"); msg != "" {
		t.Errorf("expected valid, got: %s", msg)
	}
	if msg := rcm.ValidateResourceControlValue("attention_backend", "invalid"); msg == "" {
		t.Error("expected error for invalid enum value")
	}
}

func TestValidateResourceControlUnsupported(t *testing.T) {
	vendorJSON := `{
		"resource_controls": {
			"gpu_memory_fraction": {
				"supported": false,
				"reason": "not supported"
			}
		}
	}`
	rcm := ParseResourceControls(vendorJSON)
	if msg := rcm.ValidateResourceControlValue("gpu_memory_fraction", 0.5); msg == "" {
		t.Error("expected error for unsupported control")
	}
}

func TestParseResourceControlsEmpty(t *testing.T) {
	if rcm := ParseResourceControls(""); rcm != nil {
		t.Error("expected nil for empty string")
	}
	if rcm := ParseResourceControls("{}"); rcm != nil {
		t.Error("expected nil for empty object")
	}
	if rcm := ParseResourceControls(`{"other": "data"}`); rcm != nil {
		t.Error("expected nil when no resource_controls key")
	}
}

func TestBuildResourceControlArgs(t *testing.T) {
	vendorJSON := `{
		"resource_controls": {
			"gpu_memory_fraction": {
				"arg": "--gpu-memory-utilization",
				"type": "float",
				"min": 0.1,
				"max": 0.95
			},
			"max_model_len": {"arg": "--max-model-len", "type": "int"}
		}
	}`
	rcm := ParseResourceControls(vendorJSON)

	params := map[string]interface{}{
		"gpu_memory_fraction": 0.7,
		"max_model_len":       4096.0,
	}
	args := BuildResourceControlArgs(params, rcm)
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	// Check that both args were generated (order may vary due to map iteration)
	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "--gpu-memory-utilization") {
		t.Error("expected --gpu-memory-utilization in args")
	}
	if !strings.Contains(argStr, "--max-model-len") {
		t.Error("expected --max-model-len in args")
	}
}

func TestResourceControlJSON(t *testing.T) {
	vendorJSON := `{
		"resource_controls": {
			"gpu_memory_fraction": {
				"arg": "--gpu-memory-utilization",
				"type": "float",
				"min": 0.1,
				"max": 0.95,
				"default": 0.9
			}
		}
	}`
	rcm := ParseResourceControls(vendorJSON)
	b, err := json.Marshal(rcm)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var roundtrip ResourceControlsMap
	if err := json.Unmarshal(b, &roundtrip); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if roundtrip.ResourceControlArg("gpu_memory_fraction") != "--gpu-memory-utilization" {
		t.Error("roundtrip lost arg")
	}
}
