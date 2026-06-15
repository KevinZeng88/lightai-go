package runtime

import (
	"context"
	"fmt"
	"strings"

	"lightai-go/internal/common/log"
)

// compile-time interface check
var _ RuntimeDriver = (*DockerRuntimeDriver)(nil)

// DockerRuntimeDriver implements RuntimeDriver for Docker containers.
//
// It consumes only AgentRunSpec — it never accesses the database or
// re-derives business objects. Container lifecycle is delegated to a
// DockerClient interface so that tests can inject a fake.
type DockerRuntimeDriver struct {
	client DockerClient
}

// NewDockerRuntimeDriver creates a new DockerRuntimeDriver backed by the
// given DockerClient.
func NewDockerRuntimeDriver(client DockerClient) *DockerRuntimeDriver {
	return &DockerRuntimeDriver{client: client}
}

// Start creates and starts a Docker container from the given spec.
//
// Only RuntimeType == "docker" is supported. The method:
//  1. Validates the spec
//  2. Builds a ContainerCreateOptions from the AgentRunSpec
//  3. Creates the container
//  4. Starts the container
//  5. Returns a RuntimeInstance descriptor
func (d *DockerRuntimeDriver) Start(ctx context.Context, spec AgentRunSpec) (*RuntimeInstance, error) {
	if spec.RuntimeType != "docker" {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedRuntimeType, spec.RuntimeType)
	}

	opts := d.buildCreateOptions(spec)

	log.Info("docker runtime: creating container",
		"image", opts.Image,
		"name", opts.ContainerName,
		"env", redactEnvForLog(spec.Env),
	)

	containerID, err := d.client.ContainerCreate(ctx, opts)
	if err != nil {
		log.Error("docker runtime: container create failed",
			"name", opts.ContainerName,
			"error", err,
		)
		return nil, fmt.Errorf("docker create: %w", err)
	}

	log.Info("docker runtime: starting container",
		"id", containerID,
		"name", opts.ContainerName,
	)

	if err := d.client.ContainerStart(ctx, containerID); err != nil {
		log.Error("docker runtime: container start failed",
			"id", containerID,
			"name", opts.ContainerName,
			"error", err,
		)
		return nil, fmt.Errorf("docker start: %w", err)
	}

	endpointURL := ""
	if spec.HostPort != 0 {
		endpointURL = fmt.Sprintf("http://localhost:%d", spec.HostPort)
	}

	log.Info("docker runtime: container started",
		"id", containerID,
		"instance_id", spec.InstanceID,
		"endpoint", endpointURL,
	)

	return &RuntimeInstance{
		InstanceID:    spec.InstanceID,
		ContainerID:   containerID,
		ContainerName: opts.ContainerName,
		EndpointURL:   endpointURL,
		HostPort:      spec.HostPort,
	}, nil
}

// Stop stops the container associated with the given instance ID.
//
// The container is looked up by name: lightai-{instanceID}.
func (d *DockerRuntimeDriver) Stop(ctx context.Context, instanceID string) error {
	containerName := containerNameFromInstance(instanceID)

	log.Info("docker runtime: stopping container",
		"instance_id", instanceID,
		"container", containerName,
	)

	// Find container ID by name using inspect.
	info, err := d.client.ContainerInspect(ctx, containerName)
	if err != nil {
		log.Error("docker runtime: container not found for stop",
			"instance_id", instanceID,
			"container", containerName,
			"error", err,
		)
		return fmt.Errorf("stop: container not found: %w", err)
	}

	if err := d.client.ContainerStop(ctx, info.ID, 30); err != nil {
		log.Error("docker runtime: container stop failed",
			"id", info.ID,
			"instance_id", instanceID,
			"error", err,
		)
		return fmt.Errorf("docker stop: %w", err)
	}

	log.Info("docker runtime: container stopped",
		"id", info.ID,
		"instance_id", instanceID,
	)
	return nil
}

// Inspect returns the current status of the instance's container.
func (d *DockerRuntimeDriver) Inspect(ctx context.Context, instanceID string) (*RuntimeInstanceStatus, error) {
	containerName := containerNameFromInstance(instanceID)

	info, err := d.client.ContainerInspect(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("inspect: %w", err)
	}

	state := mapContainerState(info.State)

	return &RuntimeInstanceStatus{
		InstanceID:  instanceID,
		ContainerID: info.ID,
		State:       state,
		ExitCode:    info.ExitCode,
		Error:       info.Error,
		StartedAt:   info.StartedAt,
		FinishedAt:  info.FinishedAt,
	}, nil
}

// Logs returns log output from the instance's container.
// The instanceID parameter can be an instance UUID (from which the container
// name is derived) or a direct container ID/name.
func (d *DockerRuntimeDriver) Logs(ctx context.Context, instanceID string, opts LogOptions) (*RuntimeLogs, error) {
	// Try the given ID directly first (Docker supports container IDs and names).
	// Fall back to the derived container name from instance ID.
	target := instanceID
	if _, err := d.client.ContainerInspect(ctx, target); err != nil {
		// Try derived name.
		target = containerNameFromInstance(instanceID)
	}

	stdout, stderr, err := d.client.ContainerLogs(ctx, target, LogFetchOptions{
		Tail:       opts.Tail,
		Timestamps: opts.Timestamps,
		Since:      opts.Since,
		Until:      opts.Until,
	})
	if err != nil {
		return nil, fmt.Errorf("logs: %w", err)
	}

	// For the real Docker client, the multiplexed stream has been decoded
	// into separate stdout and stderr strings. The fake client also returns
	// separated output (stdout only for now).
	return &RuntimeLogs{
		Stdout: stdout,
		Stderr: stderr,
	}, nil
}

// buildCreateOptions converts an AgentRunSpec into ContainerCreateOptions.
//
// Only enabled/valid fields are included; disabled fields are excluded.
// Sensitive env values are present in the container config (needed for
// the actual process) but are redacted in log output.
func (d *DockerRuntimeDriver) buildCreateOptions(spec AgentRunSpec) ContainerCreateOptions {
	opts := ContainerCreateOptions{
		Image:         spec.Docker.Image,
		ContainerName: containerNameFromInstance(spec.InstanceID),
		Env:           mapToEnvList(spec.Env),
		Command:       spec.Docker.Args,
	}

	// Docker options.
	if spec.Docker.Privileged {
		opts.Privileged = true
	}
	if spec.Docker.IPCMode != "" {
		opts.IPCMode = spec.Docker.IPCMode
	}
	if spec.Docker.UTSMode != "" {
		opts.UTSMode = spec.Docker.UTSMode
	}
	if spec.Docker.ShmSize != "" {
		opts.ShmSize = spec.Docker.ShmSize
	}
	if spec.Docker.NetworkMode != "" {
		opts.NetworkMode = spec.Docker.NetworkMode
	}
	if len(spec.Docker.GroupAdd) > 0 {
		opts.GroupAdd = spec.Docker.GroupAdd
	}
	if len(spec.Docker.SecurityOptions) > 0 {
		opts.SecurityOpt = spec.Docker.SecurityOptions
	}
	if len(spec.Docker.Ulimits) > 0 {
		opts.Ulimits = spec.Docker.Ulimits
	}
	if spec.Docker.RestartPolicy != "" {
		opts.RestartPolicy = spec.Docker.RestartPolicy
	}

	// GPU DeviceRequests — structured GPU access via Docker API.
	// NVIDIA GPUs use DeviceRequest with driver="nvidia".
	// MetaX and other vendors use raw device passthrough (/dev/dri, etc.)
	// via opts.Devices, not DeviceRequest.
	if spec.Vendor == "nvidia" && len(spec.GPUDeviceIDs) > 0 {
		dr := DeviceRequest{
			Driver:       "nvidia",
			Capabilities: [][]string{{"gpu"}},
			DeviceIDs:    spec.GPUDeviceIDs,
		}
		opts.DeviceRequests = append(opts.DeviceRequests, dr)
	}

	// Volumes.
	for _, v := range spec.Volumes {
		bind := v.HostPath + ":" + v.ContainerPath
		if v.Readonly {
			bind += ":ro"
		}
		opts.Binds = append(opts.Binds, bind)
	}

	// Devices.
	for _, d := range spec.Devices {
		perms := d.Permissions
		if perms == "" {
			perms = "rwm"
		}
		opts.Devices = append(opts.Devices, DeviceMapping{
			HostPath:      d.HostPath,
			ContainerPath: d.ContainerPath,
			Permissions:   perms,
		})
	}

	// Ports.
	if len(spec.Ports) > 0 {
		opts.PortBindings = make(map[string][]PortBinding)
		for _, p := range spec.Ports {
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			key := fmt.Sprintf("%d/%s", p.ContainerPort, proto)
			opts.PortBindings[key] = append(opts.PortBindings[key], PortBinding{
				HostPort: fmt.Sprintf("%d", p.HostPort),
			})
		}
	}

	return opts
}

// mapToEnvList converts a map[string]string to a "KEY=VALUE" slice.
func mapToEnvList(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

// containerNameFromInstance returns the deterministic container name for an
// instance ID: lightai-{first 12 chars}.
func containerNameFromInstance(instanceID string) string {
	if len(instanceID) > 12 {
		instanceID = instanceID[:12]
	}
	return "lightai-" + instanceID
}

// mapContainerState converts a Docker container state string to a LightAI
// instance state.
func mapContainerState(dockerState string) string {
	switch strings.ToLower(dockerState) {
	case "created":
		return "pending"
	case "running":
		return "running"
	case "paused":
		return "unhealthy"
	case "restarting":
		return "starting"
	case "removing":
		return "stopping"
	case "exited":
		return "stopped"
	case "dead":
		return "failed"
	default:
		return "unknown"
	}
}
