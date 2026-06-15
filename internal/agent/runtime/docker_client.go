package runtime

import (
	"context"
)

// DockerClient is a minimal Docker API subset used by DockerRuntimeDriver.
// It exists so callers can inject a fake for testing without needing a
// real Docker daemon.
//
// The interface uses purpose-built parameter structs rather than Docker SDK
// types so that both real and fake implementations are straightforward.
type DockerClient interface {
	// ContainerCreate creates a container but does not start it.
	ContainerCreate(ctx context.Context, opts ContainerCreateOptions) (string, error)

	// ContainerStart starts a created container.
	ContainerStart(ctx context.Context, containerID string) error

	// ContainerStop stops a running container. The timeout is in seconds;
	// 0 means use the daemon default.
	ContainerStop(ctx context.Context, containerID string, timeoutSeconds int) error

	// ContainerInspect returns low-level information about a container.
	ContainerInspect(ctx context.Context, containerID string) (*InspectResult, error)

	// ContainerLogs returns log output from a container.
	ContainerLogs(ctx context.Context, containerID string, opts LogFetchOptions) (string, error)
}

// ContainerCreateOptions holds parameters for creating a container.
type ContainerCreateOptions struct {
	Image         string
	ContainerName string
	Command       []string // entrypoint + cmd
	Env           []string // "KEY=VALUE" format
	Binds         []string // "host:container[:ro]" format
	Devices       []DeviceMapping
	PortBindings  map[string][]PortBinding // containerPort/proto → host bindings
	Privileged    bool
	IPCMode       string
	ShmSize       string
	NetworkMode   string
	GroupAdd      []string
	SecurityOpt   []string
	Ulimits       map[string]string
	RestartPolicy string
	ExtraHosts    []string
	AutoRemove    bool
}

// DeviceMapping is a device to pass through to the container.
type DeviceMapping struct {
	HostPath      string
	ContainerPath string
	Permissions   string
}

// PortBinding maps a host port to a container port.
type PortBinding struct {
	HostPort string
}

// InspectResult summarises container state.
type InspectResult struct {
	ID      string
	Name    string
	State   string // running, exited, created, etc.
	ExitCode int
	Error   string
	StartedAt string
	FinishedAt string
}

// LogFetchOptions controls log retrieval.
type LogFetchOptions struct {
	Tail       int    // number of lines from end (0 = all)
	Timestamps bool   // include timestamps
	Since      string // RFC3339 or relative (e.g. "10m")
	Until      string // RFC3339 or relative
}
