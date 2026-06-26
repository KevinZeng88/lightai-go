package configedit

import (
	"strings"
	"unicode"
)

var sectionOrder = map[string]int{
	"basic":               10,
	"model_serving":       20,
	"backend_runtime":     30,
	"container_resources": 40,
	"devices_mounts":      50,
	"environment":         60,
	"service":             70,
	"health_check":        80,
	"advanced_raw":        90,
}

var sectionLabels = map[string]string{
	"basic":               "Basic",
	"model_serving":       "Model serving",
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
	"launcher.docker_options.optional_devices": "Optional devices",
	"launcher.docker_options.group_add":        "Additional groups",
	"runtime.model_mount":                      "Model mount",
	"runtime.env":                              "Environment variables",
	"runtime.health":                           "Health check",
	"service.container_port":                   "Container port",
	"service.host_port":                        "Host port",
	"backend.capabilities":                     "Backend capabilities",
	"backend.supported_config_items":           "Supported config items",
}

// capabilityLikeCodes are codes that contain capability/metadata information
// that should be shown as readonly_summary in advanced_raw, not as editable fields.
var capabilityLikeCodes = map[string]bool{
	"backend.capabilities":            true,
	"backend.supported_config_items":  true,
	"backend.capability_profile":      true,
	"backend.detected_capabilities":   true,
	"capabilities":                    true,
	"capabilities_detail":             true,
}

// widgetOverrides maps internal keys to preferred widget types for structured display.
var widgetOverrides = map[string]string{
	"runtime.env":               "key_value_table",
	"runtime.model_mount":       "mount_form",
	"runtime.health":            "health_check_form",
	"service.container_port":    "port_form",
	"service.host_port":         "port_form",
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
	{"optional_devices", "devices_mounts", "array", "device_table", 20},
	{"group_add", "devices_mounts", "array", "string_list", 30},
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
	case code == "launcher.image":
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
	if label := nestedString(item, "render", "label"); label != "" {
		return label
	}
	if label := nestedString(item, "extensions", "label"); label != "" {
		return label
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
