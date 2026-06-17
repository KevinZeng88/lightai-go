// Package runtime defines the RuntimeDriver interface and types for
// managing model serving instances on the Agent side.
//
// The driver consumes only structured spec objects (AgentRunSpec) — it
// must never access the database or re-derive business objects.
package runtime

import (
	"context"
	"errors"
)

// ErrUnsupportedRuntimeType is returned when Start receives a spec whose
// RuntimeType is not supported by this driver.
var ErrUnsupportedRuntimeType = errors.New("unsupported runtime type")

// RuntimeDriver is the interface for managing model serving instances.
// Each implementation handles a specific runtime type (docker, process,
// remote, etc.).
type RuntimeDriver interface {
	// Start creates and starts a new instance from the given spec.
	// Returns the instance descriptor on success.
	Start(ctx context.Context, spec AgentRunSpec) (*RuntimeInstance, error)

	// Stop stops the instance identified by instanceID.
	Stop(ctx context.Context, instanceID string) error

	// Inspect returns the current status of an instance.
	Inspect(ctx context.Context, instanceID string) (*RuntimeInstanceStatus, error)

	// Logs returns recent log output for an instance.
	Logs(ctx context.Context, instanceID string, opts LogOptions) (*RuntimeLogs, error)
}

// ==========================================================================
// AgentRunSpec — agent-side mirror of server ResolvedRunSpec
// ==========================================================================

// AgentRunSpec is the frozen run specification consumed by the Agent.
// It is JSON-compatible with the server's ResolvedRunSpec so that the
// Agent can deserialize it directly without importing server packages.
type AgentRunSpec struct {
	OperationID      string             `json:"operation_id,omitempty"`
	InstanceID       string             `json:"instance_id"`
	DeploymentID     string             `json:"deployment_id"`
	RuntimeType      string             `json:"runtime_type"`
	BackendType      string             `json:"backend_type"`
	Vendor           string             `json:"vendor"`
	ModelPath        string             `json:"model_path"`
	ServedModelName  string             `json:"served_model_name"`
	NodeID           string             `json:"node_id"`
	AgentID          string             `json:"agent_id"`
	GPUDeviceIDs     []string           `json:"gpu_device_ids"`
	GPUVisibleEnvKey string             `json:"gpu_visible_env_key,omitempty"`
	Env              map[string]string  `json:"env"`
	Args             []string           `json:"args"`
	HostPort         int                `json:"host_port"`
	ContainerPort    int                `json:"container_port"`
	Volumes          []VolumeSpec       `json:"volumes,omitempty"`
	Devices          []DeviceSpec       `json:"devices,omitempty"`
	Ports            []PortSpec         `json:"ports,omitempty"`
	Docker           DockerSpec         `json:"docker,omitempty"`
	HealthCheck      *HealthCheckConfig `json:"health_check,omitempty"`
}

// HealthCheckConfig is the health check configuration for endpoint readiness.
type HealthCheckConfig struct {
	Enabled         bool   `json:"enabled"`
	Path            string `json:"path"`
	Port            int    `json:"port"`
	Scheme          string `json:"scheme"`
	ExpectedStatus  int    `json:"expected_status"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	IntervalSeconds int    `json:"interval_seconds"`
}

// DockerSpec holds Docker-specific runtime configuration.
type DockerSpec struct {
	Image           string            `json:"image"`
	ContainerName   string            `json:"container_name"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args"`
	Privileged      bool              `json:"privileged,omitempty"`
	IPCMode         string            `json:"ipc_mode,omitempty"`
	UTSMode         string            `json:"uts_mode,omitempty"`
	NetworkMode     string            `json:"network_mode,omitempty"`
	ShmSize         string            `json:"shm_size,omitempty"`
	GroupAdd        []string          `json:"group_add,omitempty"`
	SecurityOptions []string          `json:"security_options,omitempty"`
	Ulimits         map[string]string `json:"ulimits,omitempty"`
	RestartPolicy   string            `json:"restart_policy,omitempty"`
	GPUDeviceIDs    []string          `json:"gpu_device_ids,omitempty"`
}

// VolumeSpec describes a volume mount.
type VolumeSpec struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Readonly      bool   `json:"readonly,omitempty"`
}

// DeviceSpec describes a device mapping.
type DeviceSpec struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Permissions   string `json:"permissions,omitempty"`
}

// PortSpec describes a port mapping.
type PortSpec struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
}

// ==========================================================================
// Result types
// ==========================================================================

// RuntimeInstance is returned by Start on success.
type RuntimeInstance struct {
	InstanceID    string `json:"instance_id"`
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	EndpointURL   string `json:"endpoint_url,omitempty"`
	HostPort      int    `json:"host_port"`
}

// RuntimeInstanceStatus is returned by Inspect.
type RuntimeInstanceStatus struct {
	InstanceID  string `json:"instance_id"`
	ContainerID string `json:"container_id"`
	State       string `json:"state"`
	ExitCode    int    `json:"exit_code"`
	Error       string `json:"error,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
}

// RuntimeLogs holds log output from a container.
type RuntimeLogs struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// LogOptions controls log retrieval behaviour.
type LogOptions struct {
	Tail       int    // number of lines from end (0 = all)
	Timestamps bool   // include timestamps
	Since      string // RFC3339 or relative (e.g. "10m")
	Until      string // RFC3339 or relative
}
