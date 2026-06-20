package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	startTime := time.Now()
	opID := spec.OperationID
	if opID == "" {
		opID = spec.InstanceID
	} // fallback
	ctx = log.WithOperationID(ctx, opID)

	if spec.RuntimeType != "docker" {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedRuntimeType, spec.RuntimeType)
	}

	opts := d.buildCreateOptions(spec)

	// DEBUG: log full Docker spec (image, name, env keys only, ports, volumes, devices).
	log.DebugContext(ctx, "docker.spec.dump",
		"operation_id", opID,
		"instance_id", spec.InstanceID,
		"image", opts.Image,
		"container_name", opts.ContainerName,
		"env_keys", log.RedactEnvKeys(spec.Env),
		"ports", fmt.Sprintf("%v", spec.Ports),
		"volumes_count", len(spec.Volumes),
		"devices_count", len(spec.Devices),
		"privileged", opts.Privileged,
		"network_mode", opts.NetworkMode,
		"shm_size", opts.ShmSize,
		"ipc_mode", opts.IPCMode,
		"gpu_device_ids", spec.GPUDeviceIDs,
	)

	// --- Docker create ---
	createStart := time.Now()
	log.Info("docker.create.spec",
		"operation_id", opID,
		"instance_id", spec.InstanceID,
		"deployment_id", spec.DeploymentID,
		"image", opts.Image,
		"container_name", opts.ContainerName,
		"command_json", opts.Command,
		"host_port", spec.HostPort,
		"container_port", spec.ContainerPort,
		"binds_count", len(opts.Binds),
		"devices_count", len(opts.Devices),
		"env_keys", log.RedactEnvKeys(spec.Env),
	)

	containerID, err := d.client.ContainerCreate(ctx, opts)
	createDuration := time.Since(createStart).Milliseconds()
	if err != nil {
		log.Error("docker.create.failed",
			"operation_id", opID,
			"instance_id", spec.InstanceID,
			"container_name", opts.ContainerName,
			"image", opts.Image,
			"duration_ms", createDuration,
			"error", err,
		)
		return nil, fmt.Errorf("docker create: %w", err)
	}

	log.Info("docker.create.completed",
		"operation_id", opID,
		"container_id", containerID,
		"container_name", opts.ContainerName,
		"deployment_id", spec.DeploymentID,
		"duration_ms", createDuration,
	)
	if createDuration > log.SummaryConfig.SlowDockerThresholdMs {
		log.SlowOperation(ctx, "docker.create", "container_create", createDuration, log.SummaryConfig.SlowDockerThresholdMs,
			"operation_id", opID, "container_id", containerID, "image", opts.Image)
	}

	// --- Docker start ---
	startStart := time.Now()
	log.Info("docker.start.started",
		"operation_id", opID,
		"container_id", containerID,
		"container_name", opts.ContainerName,
		"deployment_id", spec.DeploymentID,
	)

	if err := d.client.ContainerStart(ctx, containerID); err != nil {
		startDuration := time.Since(startStart).Milliseconds()
		log.Error("docker.start.failed",
			"operation_id", opID,
			"container_id", containerID,
			"container_name", opts.ContainerName,
			"duration_ms", startDuration,
			"error", err,
		)
		inst := d.diagnoseContainerFailure(ctx, spec, containerID, opts.ContainerName, "container_exited")
		return inst, fmt.Errorf("docker start: %w", err)
	}

	startDuration := time.Since(startStart).Milliseconds()
	log.Info("docker.start.completed",
		"operation_id", opID,
		"container_id", containerID,
		"deployment_id", spec.DeploymentID,
		"duration_ms", startDuration,
	)
	if startDuration > log.SummaryConfig.SlowDockerThresholdMs {
		log.SlowOperation(ctx, "docker.start", "container_start", startDuration, log.SummaryConfig.SlowDockerThresholdMs,
			"operation_id", opID, "container_id", containerID)
	}

	// Minimal post-start verification: inspect container state to detect exited(1).
	verifyStart := time.Now()
	info, verr := d.client.ContainerInspect(ctx, containerID)
	if verr != nil {
		log.WarnContext(ctx, "docker.post_start.inspect_failed",
			"container_id", containerID,
			"error", verr,
			"duration_ms", time.Since(verifyStart).Milliseconds())
	} else if info.State != "running" {
		log.ErrorContext(ctx, "docker.post_start.container_not_running",
			"container_id", containerID,
			"container_name", opts.ContainerName,
			"container_state", info.State,
			"exit_code", info.ExitCode,
			"container_error", info.Error,
			"inspect_duration_ms", time.Since(verifyStart).Milliseconds())
		inst := d.diagnoseContainerFailure(ctx, spec, containerID, opts.ContainerName, "container_exited")
		return inst, fmt.Errorf("container %s is %s (exit_code=%d): %s", shortContainerID(containerID), info.State, info.ExitCode, info.Error)
	}
	log.DebugContext(ctx, "docker.post_start.verified_running",
		"container_id", containerID,
		"container_state", info.State,
		"inspect_duration_ms", time.Since(verifyStart).Milliseconds())

	// Endpoint health check (if configured).
	if spec.HealthCheck != nil && spec.HealthCheck.Enabled {
		resolvedCfg := resolveHealthCheckConfig(spec.HealthCheck, spec.HostPort)
		// Pass container inspect function for re-inspect on health check failure.
		inspectFn := func(ctx context.Context) (string, int, error) {
			info, err := d.client.ContainerInspect(ctx, containerID)
			if err != nil {
				return "", 0, err
			}
			return info.State, info.ExitCode, nil
		}
		if err := CheckEndpointReady(ctx, resolvedCfg, spec.InstanceID, containerID, opts.ContainerName, inspectFn); err != nil {
			reasonCode := "health_check_failed"
			if strings.Contains(strings.ToLower(err.Error()), "timeout") {
				reasonCode = "health_timeout"
			} else if strings.Contains(strings.ToLower(err.Error()), "container") && strings.Contains(strings.ToLower(err.Error()), "exit_code") {
				reasonCode = "container_exited"
			}
			log.ErrorContext(ctx, "health_check.failed",
				"container_id", containerID,
				"instance_id", spec.InstanceID,
				"error", err,
				"duration_ms", time.Since(startTime).Milliseconds(),
			)
			inst := d.diagnoseContainerFailure(ctx, spec, containerID, opts.ContainerName, reasonCode)
			return inst, fmt.Errorf("health check failed: %w", err)
		}
	} else {
		log.InfoContext(ctx, "health_check.skipped",
			"reason", "no_health_config",
			"instance_id", spec.InstanceID,
			"container_id", containerID,
		)
	}

	endpointURL := ""
	if spec.HostPort != 0 {
		endpointURL = fmt.Sprintf("http://localhost:%d", spec.HostPort)
	}

	totalDuration := time.Since(startTime).Milliseconds()
	log.Info("docker.start.operation_completed",
		"operation_id", opID,
		"instance_id", spec.InstanceID,
		"container_id", containerID,
		"endpoint", endpointURL,
		"total_duration_ms", totalDuration,
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
	startTime := time.Now()
	containerName := containerNameFromInstance(instanceID)

	log.Info("docker.stop.started",
		"instance_id", instanceID,
		"container_name", containerName,
	)

	// Find container ID by name using inspect.
	// REVIEW-006: Treat missing container as already stopped — idempotent stop.
	info, err := d.client.ContainerInspect(ctx, containerName)
	if err != nil {
		log.Info("docker.stop.container_missing_treat_as_stopped",
			"instance_id", instanceID,
			"container_name", containerName,
			"duration_ms", time.Since(startTime).Milliseconds(),
			"error", err,
		)
		return nil // Already stopped/removed — success.
	}

	stopStart := time.Now()
	if err := d.client.ContainerStop(ctx, info.ID, 30); err != nil {
		log.Error("docker.stop.failed",
			"container_id", info.ID,
			"instance_id", instanceID,
			"duration_ms", time.Since(stopStart).Milliseconds(),
			"error", err,
		)
		return fmt.Errorf("docker stop: %w", err)
	}

	stopDuration := time.Since(stopStart).Milliseconds()
	log.Info("docker.stop.completed",
		"container_id", info.ID,
		"instance_id", instanceID,
		"stop_duration_ms", stopDuration,
		"total_duration_ms", time.Since(startTime).Milliseconds(),
	)
	if stopDuration > log.SummaryConfig.SlowDockerThresholdMs {
		log.SlowOperation(ctx, "docker.stop", "container_stop", stopDuration, log.SummaryConfig.SlowDockerThresholdMs,
			"container_id", info.ID, "instance_id", instanceID)
	}

	return nil
}

// logContainerFailure attempts to retrieve container state and logs on failure.
// It does not block the caller — failures in this diagnostic helper are silent.
// operation_id is read from ctx.
func (d *DockerRuntimeDriver) logContainerFailure(ctx context.Context, containerID, containerName string) {
	_ = d.diagnoseContainerFailure(ctx, AgentRunSpec{}, containerID, containerName, "")
}

func (d *DockerRuntimeDriver) diagnoseContainerFailure(ctx context.Context, spec AgentRunSpec, containerID, containerName, reasonCode string) *RuntimeInstance {
	inst := &RuntimeInstance{
		InstanceID:        spec.InstanceID,
		ContainerID:       containerID,
		ContainerName:     containerName,
		HostPort:          spec.HostPort,
		FailureReasonCode: reasonCode,
		ExitCode:          -1,
	}
	// Best-effort inspect.
	info, err := d.client.ContainerInspect(ctx, containerName)
	if err != nil {
		log.WarnContext(ctx, "docker.diagnose.inspect_failed",
			"container_id", containerID,
			"container_name", containerName,
			"error", err,
		)
		return inst
	}
	inst.ContainerID = info.ID
	inst.ContainerState = info.State
	inst.ExitCode = info.ExitCode
	inst.ContainerError = info.Error
	if inst.FailureReasonCode == "" && info.State != "" && info.State != "running" {
		inst.FailureReasonCode = "container_exited"
	}

	log.ErrorContext(ctx, "docker.container.exited",
		"container_id", info.ID,
		"container_name", containerName,
		"state", info.State,
		"exit_code", info.ExitCode,
		"error_message", info.Error,
		"started_at", info.StartedAt,
		"finished_at", info.FinishedAt,
	)

	// Try to tail logs (with a short timeout to avoid blocking).
	logCtx, logCancel := context.WithTimeout(ctx, 5*time.Second)
	defer logCancel()
	stdout, stderr, err := d.client.ContainerLogs(logCtx, containerID, LogFetchOptions{Tail: 50})
	if err != nil {
		log.WarnContext(ctx, "docker.diagnose.logs_failed",
			"container_id", containerID,
			"error", err,
		)
		return inst
	}
	if stderr != "" {
		inst.StderrTailPreview = singleLineTailStr(stderr, 2048)
		log.ErrorContext(ctx, "docker.container.stderr",
			"container_id", containerID,
			"stderr_tail_preview", inst.StderrTailPreview,
		)
	}
	if stdout != "" {
		inst.StdoutTailPreview = singleLineTailStr(stdout, 2048)
		log.InfoContext(ctx, "docker.container.stdout_tail",
			"container_id", containerID,
			"stdout_tail_preview", inst.StdoutTailPreview,
		)
	}
	return inst
}

func shortContainerID(containerID string) string {
	if len(containerID) <= 12 {
		return containerID
	}
	return containerID[:12]
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
		Entrypoint:    spec.Docker.Command,
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
	// Driver and Capabilities come from docker_json (catalog/NBR), not hardcoded.
	// Default (empty driver, [["gpu"]] caps) matches docker run --gpus CLI.
	// DeviceIDs are GPU indices (not UUIDs) from the resolver.
	// nil/empty DeviceIDs means all GPUs (maps to Count=-1 in Docker SDK).
	// MetaX and other vendors use raw device passthrough (/dev/dri, etc.)
	// via opts.Devices, not DeviceRequest.
	if spec.Vendor == "nvidia" {
		driver := spec.Docker.GpuDriver
		caps := spec.Docker.GpuCapabilities
		if len(caps) == 0 {
			caps = [][]string{{"gpu"}}
		}
		dr := DeviceRequest{
			Driver:       driver,
			Capabilities: caps,
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

// singleLineTail escapes newlines, carriage returns, and tabs in a log tail
// so it fits in a single structured log line.
func singleLineTail(s string, maxBytes int) (escaped string, truncated bool, byteCount int) {
	byteCount = len(s)
	if len(s) > maxBytes {
		s = s[:maxBytes]
		truncated = true
	}
	r := strings.NewReplacer("\n", "\\n", "\r", "\\r", "\t", "\\t")
	escaped = r.Replace(s)
	return
}

// singleLineTailStr is a convenience wrapper returning only the escaped string.
func singleLineTailStr(s string, maxBytes int) string {
	escaped, _, _ := singleLineTail(s, maxBytes)
	return escaped
}
