// Package runplan provides the RunPlan Resolver and related types.
package runplan

// ResolvedRunPlan is the frozen run specification sent to the Agent.
type ResolvedRunPlan struct {
	Image         string            `json:"image"`
	ContainerName string            `json:"container_name"`
	Entrypoint    []string          `json:"entrypoint,omitempty"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`

	Privileged  bool              `json:"privileged,omitempty"`
	IPCMode     string            `json:"ipc_mode,omitempty"`
	UTSMode     string            `json:"uts_mode,omitempty"`
	NetworkMode string            `json:"network_mode,omitempty"`
	ShmSize     string            `json:"shm_size,omitempty"`
	Ulimits     map[string]string `json:"ulimits,omitempty"`

	Devices  []DeviceMapping `json:"devices,omitempty"`
	Mounts   []MountMapping  `json:"mounts,omitempty"`
	GroupAdd []string        `json:"group_add,omitempty"`

	HostPort      int `json:"host_port"`
	ContainerPort int `json:"container_port"`

	GPUDeviceIDs     []string   `json:"gpu_device_ids,omitempty"`
	GPUVisibleEnvKey string     `json:"gpu_visible_env_key,omitempty"`
	GpuDriver        string     `json:"gpu_driver,omitempty"`       // DeviceRequest driver, e.g. "" for docker run --gpus CLI
	GpuCapabilities  [][]string `json:"gpu_capabilities,omitempty"` // e.g. [["gpu"]]

	SecurityOptions []string `json:"security_options,omitempty"`
	ExtraArgs       []string `json:"extra_args,omitempty"`

	HealthCheck HealthCheck `json:"health_check,omitempty"`

	// Hash identifiers
	InputHash string `json:"input_hash"`
	PlanHash  string `json:"plan_hash"`

	// Audit references
	BackendName     string `json:"backend_name"`
	BackendVersion  string `json:"backend_version"`
	ModelName       string `json:"model_name"`
	ModelPath       string `json:"model_path"`
	ServedModelName string `json:"served_model_name"`
	DeploymentID    string `json:"deployment_id"`
	InstanceID      string `json:"instance_id"`
}

// DeviceMapping maps a host device to a container device.
type DeviceMapping struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Permissions   string `json:"permissions,omitempty"`
}

// MountMapping maps a host path to a container path.
type MountMapping struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Readonly      bool   `json:"readonly,omitempty"`
}

// HealthCheck defines the container health check configuration.
type HealthCheck struct {
	Path                  string `json:"path"`
	ExpectedStatus        int    `json:"expected_status"`
	StartupTimeoutSeconds int    `json:"startup_timeout_seconds"`
	IntervalSeconds       int    `json:"interval_seconds"`
	TimeoutSeconds        int    `json:"timeout_seconds"`
}

// DeviceBinding was removed (2026-06-25) as dead code.
// GPU/vendor binding is handled directly in agent/runtime/docker.go buildCreateOptions()
// via spec.Vendor string check, without needing this intermediate abstraction.
// See: docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-f-architecture-items/decisions.md
