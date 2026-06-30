package configedit

import (
	"strings"
	"unicode"
)

var sectionOrder = map[string]int{
	"basic":               10,
	"model_serving":       20,
	"advanced_parameters": 30,
	"expert_parameters":   35,
	"backend_runtime":     40,
	"container_resources": 50,
	"devices_mounts":      60,
	"environment":         70,
	"service":             80,
	"health_check":        90,
	"advanced_raw":        90,
}

var sectionLabels = map[string]string{
	"basic":               "Basic",
	"model_serving":       "Model serving",
	"advanced_parameters": "Advanced parameters",
	"expert_parameters":   "Expert parameters",
	"backend_runtime":     "Backend runtime",
	"container_resources": "Container resources",
	"devices_mounts":      "Devices and mounts",
	"environment":         "Environment",
	"service":             "Service",
	"health_check":        "Health check",
	"advanced_raw":        "Advanced raw configuration",
}

var taxonomyLabels = map[string]string{
	"launcher.image":                           "Image",
	"launcher.command":                         "Command",
	"launcher.entrypoint":                      "Entrypoint",
	"launcher.docker_options.shm_size":         "Shared memory",
	"launcher.docker_options.privileged":       "Privileged container",
	"launcher.docker_options.ipc_mode":         "IPC mode",
	"launcher.docker_options.uts_mode":         "UTS mode",
	"launcher.docker_options.network_mode":     "Network mode",
	"launcher.docker_options.security_options": "Security options",
	"launcher.docker_options.ulimits":          "Ulimits",
	"launcher.docker_options.devices":          "Devices",
	"launcher.docker_options.group_add":        "Additional groups",
	"runtime.model_mount":                      "Model mount",
	"runtime.device_binding":                   "Device binding",
	"runtime.env":                              "Environment variables",
	"runtime.health":                           "Health check",
	"service.container_port":                   "Container port",
	"service.host_port":                        "Host port",
	"service.listen_host":                      "Container listen host",
	"deployment.served_model_name":             "Served model name",
	"backend.extra_args":                       "Extra launch arguments",
	"runtime.extra_env":                        "Extra environment variables",
	"launcher.kind":                            "Launcher type",
	"launcher.devices":                         "Device bindings",
	"launcher.ports":                           "Port mappings",
	"launcher.volumes":                         "Volume mounts",
	"model_runtime.gpu_memory_utilization":     "GPU memory utilization",
	"model_runtime.max_model_len":              "Max model length",
	"model_runtime.dtype":                      "Data type",
	"model_runtime.tensor_parallel_size":       "Tensor parallel size",
	"model_runtime.pipeline_parallel_size":     "Pipeline parallel size",
	"model_runtime.max_num_batched_tokens":     "Max batched tokens",
	"model_runtime.max_num_seqs":               "Max concurrent sequences",
	"model_runtime.kv_cache_dtype":             "KV cache data type",
	"model_runtime.cpu_offload_gb":             "CPU offload capacity",
	"model_runtime.swap_space":                 "Swap space",
	"model_runtime.enforce_eager":              "Enforce eager mode",
	"model_runtime.trust_remote_code":          "Trust remote code",
	"model_runtime.safetensors_load_strategy":  "Safetensors load strategy",
	"model_runtime.download_dir":               "Model download directory",
	"model_runtime.model":                      "Model path",
	"model_runtime.host":                       "Listen host",
	"model_runtime.port":                       "Service port",
	"backend.capabilities":                     "Backend capabilities",
	"backend.supported_config_items":           "Supported config items",
}

// capabilityLikeCodes are codes that contain capability/metadata information
// that should be shown as readonly_summary in advanced_raw, not as editable fields.
var capabilityLikeCodes = map[string]bool{
	"backend.capabilities":           true,
	"backend.supported_config_items": true,
	"backend.capability_profile":     true,
	"backend.detected_capabilities":  true,
	"capabilities":                   true,
	"capabilities_detail":            true,
}

// widgetOverrides maps internal keys to preferred widget types for structured display.
var widgetOverrides = map[string]string{
	"runtime.env":            "key_value_table",
	"runtime.device_binding": "accelerator_binding",
	"runtime.model_mount":    "mount_form",
	"runtime.health":         "health_check_form",
	"service.container_port": "port_form",
	"service.host_port":      "port_form",
}

var dockerFieldSpecs = []struct {
	Path    string
	Section string
	Type    string
	Widget  string
	Order   int
}{
	{"shm_size", "container_resources", "string", "string", 10},
	{"privileged", "container_resources", "boolean", "boolean", 20},
	{"ipc_mode", "container_resources", "string", "string", 30},
	{"uts_mode", "container_resources", "string", "string", 40},
	{"network_mode", "container_resources", "string", "string", 50},
	{"security_options", "container_resources", "array", "string_list", 60},
	{"ulimits", "container_resources", "object", "key_value_table", 70},
	{"devices", "devices_mounts", "array", "device_table", 10},
	{"group_add", "devices_mounts", "array", "string_list", 30},
}

// ---------------------------------------------------------------------------
// Canonical alias groups — duplicate fields merged into one canonical field.
// ---------------------------------------------------------------------------

type aliasGroup struct {
	Canonical string   // primary key shown in UI
	Label     string   // display label for the canonical field
	Aliases   []string // other keys folded into canonical
	Section   string   // preferred section for canonical field
	Widget    string   // preferred widget
}

var canonicalAliases = []aliasGroup{
	{
		Canonical: "service.listen_host",
		Label:     "Container listen host",
		Aliases:   []string{"backend.common.host", "launcher.listen_host"},
		Section:   "service",
		Widget:    "string",
	},
	{
		Canonical: "service.container_port",
		Label:     "Container listen port",
		Aliases:   []string{"backend.common.port", "launcher.container_port"},
		Section:   "service",
		Widget:    "port_form",
	},
}

// aliasCanonicalOf maps alias keys → canonical key for quick lookup.
var aliasCanonicalOf = buildAliasMap()

func buildAliasMap() map[string]string {
	m := map[string]string{}
	for _, g := range canonicalAliases {
		for _, a := range g.Aliases {
			m[a] = g.Canonical
		}
		m[g.Canonical] = g.Canonical // self-map
	}
	return m
}

// ---------------------------------------------------------------------------
// Layer scope — which codes are hidden or readonly for specific layers.
// ---------------------------------------------------------------------------

// modelServingCodes are backend serving parameters that belong at Deployment
// layer, not at BackendRuntime or NodeBackendRuntime.
var modelServingCodes = map[string]bool{
	"backend.arg.max_model_len":          true,
	"backend.arg.max_num_seqs":           true,
	"backend.arg.context_length":         true,
	"backend.arg.gpu_memory_utilization": true,
	"backend.arg.served_model_name":      true,
	"backend.common.served_model_name":   true,
	"backend.arg.max_num_batched_tokens": true,
	"backend.arg.tensor_parallel_size":   true,
	"backend.arg.pipeline_parallel_size": true,
	"backend.arg.enforce_eager":          true,
	"backend.arg.trust_remote_code":      true,
	"backend.arg.dtype":                  true,
	"backend.arg.seed":                   true,
	"backend.arg.temperature":            true,
	"backend.arg.top_p":                  true,
	"backend.arg.top_k":                  true,
	"backend.arg.max_tokens":             true,
	"backend.arg.repetition_penalty":     true,
}

var commonRuntimeArgs = map[string]bool{
	"backend.arg.gpu_memory_utilization":   true,
	"model_runtime.gpu_memory_utilization": true,
	"backend.arg.max_model_len":            true,
	"model_runtime.max_model_len":          true,
	"backend.arg.dtype":                    true,
	"model_runtime.dtype":                  true,
	"backend.arg.tensor_parallel_size":     true,
	"model_runtime.tensor_parallel_size":   true,
	"backend.common.port":                  true,
	"service.container_port":               true,
	"backend.arg.served_model_name":        true,
	"backend.common.served_model_name":     true,
	"deployment.served_model_name":         true,

	// model_runtime.port is removed from common runtime args because it
	// competes with service.container_port as the canonical port field.  The
	// two-port confusion produces a required + readonly + empty field in the
	// runtime template / NBR wizard that confuses users.

	"backend.arg.mem_fraction_static":   true,
	"model_runtime.mem_fraction_static": true,
	"backend.arg.context_length":        true,
	"model_runtime.context_length":      true,
	"backend.arg.tp_size":               true,
	"model_runtime.tp_size":             true,
	"backend.arg.tp":                    true,

	"backend.arg.n_gpu_layers":   true,
	"backend.arg.ngl":            true,
	"model_runtime.n_gpu_layers": true,
	"model_runtime.ngl":          true,
	"backend.arg.ctx_size":       true,
	"model_runtime.ctx_size":     true,
	"backend.arg.threads":        true,
	"model_runtime.threads":      true,
	"backend.arg.batch_size":     true,
	"model_runtime.batch_size":   true,
}

var expertRuntimeArgs = map[string]bool{
	"backend.arg.trust_remote_code":   true,
	"backend.arg.enforce_eager":       true,
	"model_runtime.trust_remote_code": true,
	"model_runtime.enforce_eager":     true,
	// Advanced/Expert parameters NOT shown as ordinary required fields:
	"backend.arg.cpu_offload_gb":              true,
	"model_runtime.cpu_offload_gb":            true,
	"backend.arg.kv_cache_dtype":              true,
	"model_runtime.kv_cache_dtype":            true,
	"backend.arg.max_num_batched_tokens":      true,
	"backend.arg.max_num_seqs":                true,
	"model_runtime.max_num_seqs":              true,
	"backend.arg.swap_space":                  true,
	"model_runtime.swap_space":                true,
	"backend.arg.safetensors_load_strategy":   true,
	"model_runtime.safetensors_load_strategy": true,
	// Internal/system parameters NOT shown in ordinary deployment form:
	"model_runtime.model":        true,
	"model_runtime.host":         true,
	"model_runtime.port":         true,
	"model_runtime.download_dir": true,
}

// isModelServingCode checks if a code is a model-serving parameter (should only
// appear at Deployment layer).
func isModelServingCode(code string) bool {
	if modelServingCodes[code] {
		return true
	}
	if strings.HasPrefix(code, "model_runtime.") {
		return true
	}
	// Also match pattern: backend.arg.max_*, backend.arg.context_*, etc.
	if strings.HasPrefix(code, "backend.arg.") {
		return true // all backend args are model serving
	}
	return false
}

// layerHiddenCodes defines codes hidden per layer.
var layerHiddenCodes = map[string]map[string]bool{
	"backend_runtime": {
		"backend.capabilities":           true,
		"backend.supported_config_items": true,
	},
	"node_backend_runtime": {
		"backend.capabilities":           true,
		"backend.supported_config_items": true,
	},
	"deployment": {
		"launcher.image":                 true,
		"backend.capabilities":           true,
		"backend.supported_config_items": true,
		"launcher.docker_options":        true, // docker sub-fields handled individually
	},
}

// layerReadonlyCodes defines codes forced readonly per layer.
var layerReadonlyCodes = map[string]map[string]bool{
	"deployment": {
		"launcher.command":    true,
		"launcher.entrypoint": true,
		"runtime.model_mount": true,
	},
}

// isLayerHidden checks if a code should be hidden for the given layer.
func isLayerHidden(code string, layer string) bool {
	if hidden, ok := layerHiddenCodes[layer]; ok && hidden[code] {
		return true
	}
	// Model serving codes hidden from non-deployment layers.
	if layer != "deployment" && layer != "node_backend_runtime" && isModelServingCode(code) {
		return true
	}
	// Docker sub-fields hidden from deployment (they're inherited from NBR).
	if layer == "deployment" && strings.HasPrefix(code, "launcher.docker_options.") {
		return true
	}
	return false
}

// isLayerReadonly checks if a code should be readonly for the given layer.
func isLayerReadonly(code string, layer string) bool {
	if ro, ok := layerReadonlyCodes[layer]; ok && ro[code] {
		return true
	}
	return false
}

// emptyValue returns true if the field value is empty/nil/zero.
func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	case bool:
		return false // boolean false is a valid value
	case float64:
		return val == 0
	case int:
		return val == 0
	default:
		return false
	}
}

func sectionFor(code string, item map[string]any) string {
	if sec := nestedString(item, "render", "section"); sec != "" {
		return sec
	}
	if sec := nestedString(item, "extensions", "section"); sec != "" {
		return sec
	}
	// Capability-like codes always go to advanced_raw.
	if capabilityLikeCodes[code] || strings.Contains(code, "capabilities") || strings.Contains(code, "supported_config") {
		return "advanced_raw"
	}
	switch {
	case code == "launcher.image" || code == "runtime.image_ref":
		return "basic"
	case code == "launcher.command" || code == "launcher.entrypoint":
		return "backend_runtime"
	case code == "launcher.docker_options":
		return "advanced_raw"
	case strings.HasPrefix(code, "backend.arg.") || strings.HasPrefix(code, "model_runtime."):
		return "model_serving"
	case code == "runtime.env":
		return "environment"
	case code == "runtime.model_mount":
		return "devices_mounts"
	case code == "runtime.health":
		return "health_check"
	case strings.HasPrefix(code, "service.") || strings.HasPrefix(code, "deployment.service"):
		return "service"
	case strings.HasPrefix(code, "source_metadata.") || strings.HasPrefix(code, "internal.") || strings.HasPrefix(code, "resolver."):
		return "advanced_raw"
	}
	if cat := stringValue(item["category"]); cat != "" {
		switch cat {
		case "model_runtime":
			return "model_serving"
		case "env":
			return "environment"
		case "advanced", "internal":
			return "advanced_raw"
		case "capabilities", "metadata":
			return "advanced_raw"
		}
	}
	return "advanced_raw"
}

func fieldLabel(code string, item map[string]any) string {
	if label := nestedString(item, "schema", "label"); label != "" {
		return label
	}
	if label := nestedString(item, "render", "label"); label != "" {
		return label
	}
	if label := nestedString(item, "extensions", "label"); label != "" {
		return label
	}
	// Check canonical alias label.
	if canon, ok := aliasCanonicalOf[code]; ok && canon != code {
		for _, g := range canonicalAliases {
			if g.Canonical == canon {
				return g.Label
			}
		}
	}
	if label, ok := taxonomyLabels[code]; ok {
		return label
	}
	if strings.HasPrefix(code, "backend.arg.") {
		return humanize(strings.TrimPrefix(code, "backend.arg."))
	}
	return humanize(code)
}

func humanize(v string) string {
	v = strings.TrimPrefix(v, "launcher.")
	v = strings.TrimPrefix(v, "runtime.")
	v = strings.TrimPrefix(v, "backend.arg.")
	v = strings.ReplaceAll(v, "_", " ")
	v = strings.ReplaceAll(v, ".", " ")
	v = strings.TrimSpace(v)
	if v == "" {
		return "Configuration"
	}
	r := []rune(v)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
