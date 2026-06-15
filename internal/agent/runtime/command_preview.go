package runtime

import (
	"fmt"
	"strings"
)

// EquivalentCommandPreview generates a human-readable docker run command
// from an AgentRunSpec. This is for display and debugging only — the
// Agent MUST use structured Docker API calls, never this string, when
// starting containers.
func EquivalentCommandPreview(spec *AgentRunSpec) string {
	var parts []string
	parts = append(parts, "docker", "run", "-d")

	// Name.
	if spec.Docker.ContainerName != "" {
		parts = append(parts, "--name", spec.Docker.ContainerName)
	}

	// Privileged.
	if spec.Docker.Privileged {
		parts = append(parts, "--privileged")
	}

	// IPC / UTS / network / shm.
	if spec.Docker.IPCMode != "" {
		parts = append(parts, "--ipc", spec.Docker.IPCMode)
	}
	if spec.Docker.UTSMode != "" {
		parts = append(parts, "--uts", spec.Docker.UTSMode)
	}
	if spec.Docker.NetworkMode != "" {
		parts = append(parts, "--network", spec.Docker.NetworkMode)
	}
	if spec.Docker.ShmSize != "" {
		parts = append(parts, "--shm-size", spec.Docker.ShmSize)
	}

	// Group add.
	for _, g := range spec.Docker.GroupAdd {
		parts = append(parts, "--group-add", g)
	}

	// Security options.
	for _, s := range spec.Docker.SecurityOptions {
		parts = append(parts, "--security-opt", s)
	}

	// Ulimits.
	for k, v := range spec.Docker.Ulimits {
		parts = append(parts, "--ulimit", fmt.Sprintf("%s=%s", k, v))
	}

	// Devices.
	for _, d := range spec.Devices {
		parts = append(parts, "--device", d.HostPath+":"+d.ContainerPath)
	}

	// Volumes.
	for _, v := range spec.Volumes {
		ro := ""
		if v.Readonly {
			ro = ":ro"
		}
		parts = append(parts, "-v", v.HostPath+":"+v.ContainerPath+ro)
	}

	// Ports.
	for _, p := range spec.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		parts = append(parts, "-p", fmt.Sprintf("%d:%d/%s", p.HostPort, p.ContainerPort, proto))
	}

	// Env (redact sensitive values in preview for safety).
	for k, v := range spec.Env {
		displayVal := v
		if isSensitive(k) {
			displayVal = "<redacted>"
		}
		parts = append(parts, "-e", fmt.Sprintf("%s=%s", k, displayVal))
	}

	// Image and args.
	parts = append(parts, spec.Docker.Image)
	parts = append(parts, spec.Docker.Args...)

	return strings.Join(parts, " ")
}
