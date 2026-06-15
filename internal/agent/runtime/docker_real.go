package runtime

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// compile-time interface check
var _ DockerClient = (*RealDockerClient)(nil)

// RealDockerClient wraps the Docker Engine SDK (v28.x compatible route).
//
// ## SDK version strategy (2026-06-15)
//
// We intentionally stay on the traditional github.com/docker/docker v28.x
// module (pre-split) instead of the newer github.com/moby/moby v29+:
//
//  1. Customer GPU servers may run Docker Engine 20.x / 23.x / 24.x / 25.x.
//     The v28 SDK's API surface is more widely compatible with older daemons
//     than the v29 split, which introduced breaking changes to both the Go
//     module layout and the client API signatures.
//
//  2. The v29 module split (github.com/moby/moby/client,
//     github.com/moby/moby/api) changes ContainerCreate, ContainerInspect,
//     ContainerStart, and ContainerStop signatures and moves several types
//     (PortMap, PortSet, Port) from nat to network packages.  Adapting to
//     these changes now would block Phase 2A/2B's core goal: model container
//     start/stop closed loop.
//
//  3. LightAI already has its own DockerClient interface
//     (internal/agent/runtime/docker_client.go).  DockerRuntimeDriver
//     depends ONLY on this interface, never on the real SDK types directly.
//     When the time comes to require Docker Engine 25+ or 29+ at customer
//     sites, we only need to replace RealDockerClient's adapter body —
//     DockerRuntimeDriver and all tests are unaffected.
//
// Migration path to v29:
//   - Change imports to github.com/moby/moby/client and
//     github.com/moby/moby/api/types.
//   - Adapt to the new ContainerCreate(…, ContainerCreateOptions) signature.
//   - Adapt to ContainerInspect(…, ContainerInspectOptions).
//   - Replace nat.PortMap / nat.PortSet with network.PortMap / network.PortSet.
//   - Drop the github.com/docker/go-connections/nat dependency.
type RealDockerClient struct {
	cli *client.Client
}

// NewRealDockerClient creates a client connected to the local Docker daemon
// using the traditional v28.x compatible SDK.
func NewRealDockerClient() (*RealDockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &RealDockerClient{cli: cli}, nil
}

// Close releases resources held by the underlying Docker client.
func (r *RealDockerClient) Close() error {
	return r.cli.Close()
}

func (r *RealDockerClient) ContainerCreate(ctx context.Context, opts ContainerCreateOptions) (string, error) {
	cfg := &container.Config{
		Image: opts.Image,
		Env:   opts.Env,
	}
	if len(opts.Command) > 0 {
		cfg.Cmd = strslice.StrSlice(opts.Command)
	}

	hostCfg := &container.HostConfig{
		Privileged:  opts.Privileged,
		NetworkMode: container.NetworkMode(opts.NetworkMode),
		GroupAdd:    opts.GroupAdd,
		SecurityOpt: opts.SecurityOpt,
		AutoRemove:  opts.AutoRemove,
	}

	if opts.ShmSize != "" {
		hostCfg.ShmSize = parseShmSize(opts.ShmSize)
	}
	if opts.IPCMode != "" {
		hostCfg.IpcMode = container.IpcMode(opts.IPCMode)
	}
	if opts.RestartPolicy != "" {
		hostCfg.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(opts.RestartPolicy),
		}
	}

	// Volumes / binds.
	if len(opts.Binds) > 0 {
		hostCfg.Binds = opts.Binds
	}

	// Devices.
	for _, d := range opts.Devices {
		hostCfg.Devices = append(hostCfg.Devices, container.DeviceMapping{
			PathOnHost:        d.HostPath,
			PathInContainer:   d.ContainerPath,
			CgroupPermissions: d.Permissions,
		})
	}

	// Port bindings.
	if len(opts.PortBindings) > 0 {
		hostCfg.PortBindings = make(nat.PortMap)
		cfg.ExposedPorts = make(nat.PortSet)
		for portProto, bindings := range opts.PortBindings {
			// portProto is "port/proto" e.g. "8000/tcp".
			// nat.NewPort takes (proto, port) as two args.
			proto, port := "tcp", portProto
			if idx := strings.LastIndex(portProto, "/"); idx >= 0 {
				proto = portProto[idx+1:]
				port = portProto[:idx]
			}
			np, err := nat.NewPort(proto, port)
			if err != nil {
				continue
			}
			cfg.ExposedPorts[np] = struct{}{}
			var ports []nat.PortBinding
			for _, b := range bindings {
				ports = append(ports, nat.PortBinding{HostPort: b.HostPort})
			}
			hostCfg.PortBindings[np] = ports
		}
	}

	// Ulimits.
	if len(opts.Ulimits) > 0 {
		for name, val := range opts.Ulimits {
			hard := parseUlimit(val)
			hostCfg.Ulimits = append(hostCfg.Ulimits, &container.Ulimit{
				Name: name,
				Hard: hard,
				Soft: hard,
			})
		}
	}

	// Extra hosts.
	if len(opts.ExtraHosts) > 0 {
		hostCfg.ExtraHosts = opts.ExtraHosts
	}

	// v28 signature: ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
	resp, err := r.cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, opts.ContainerName)
	if err != nil {
		return "", fmt.Errorf("docker create: %w", err)
	}
	return resp.ID, nil
}

func (r *RealDockerClient) ContainerStart(ctx context.Context, containerID string) error {
	return r.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (r *RealDockerClient) ContainerStop(ctx context.Context, containerID string, timeoutSeconds int) error {
	var timeout *int
	if timeoutSeconds > 0 {
		timeout = &timeoutSeconds
	}
	return r.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: timeout})
}

func (r *RealDockerClient) ContainerInspect(ctx context.Context, containerID string) (*InspectResult, error) {
	// v28 signature: ContainerInspect(ctx, containerID) (types.ContainerJSON, error)
	info, err := r.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("docker inspect: %w", err)
	}
	return &InspectResult{
		ID:         info.ID,
		Name:       strings.TrimPrefix(info.Name, "/"),
		State:      info.State.Status,
		ExitCode:   info.State.ExitCode,
		Error:      info.State.Error,
		StartedAt:  info.State.StartedAt,
		FinishedAt: info.State.FinishedAt,
	}, nil
}

func (r *RealDockerClient) ContainerLogs(ctx context.Context, containerID string, opts LogFetchOptions) (string, error) {
	tailStr := "all"
	if opts.Tail > 0 {
		tailStr = fmt.Sprintf("%d", opts.Tail)
	}

	reader, err := r.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tailStr,
		Timestamps: opts.Timestamps,
		Since:      opts.Since,
		Until:      opts.Until,
	})
	if err != nil {
		return "", fmt.Errorf("docker logs: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read docker logs: %w", err)
	}
	return string(data), nil
}

// parseShmSize converts a human-readable shm size string to bytes.
func parseShmSize(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	multiplier := int64(1)
	switch {
	case strings.HasSuffix(s, "gb"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "gb")
	case strings.HasSuffix(s, "g"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "mb"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "m"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "kb"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "k"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "k")
	}
	var val int64
	fmt.Sscanf(s, "%d", &val)
	return val * multiplier
}

// parseUlimit parses a ulimit value string like "-1" or "65536".
func parseUlimit(val string) int64 {
	var n int64
	fmt.Sscanf(val, "%d", &n)
	return n
}
