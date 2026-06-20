package runplan

import (
	"fmt"
	"strings"
)

// EquivalentCommandPreview generates a human-readable docker run command string.
// This is for display and debugging only — the Agent MUST use the ResolvedRunPlan struct, not this string.
func EquivalentCommandPreview(plan *ResolvedRunPlan) string {
	var parts []string
	parts = append(parts, "docker", "run", "-d")

	if plan.ContainerName != "" {
		parts = append(parts, "--name", plan.ContainerName)
	}
	if plan.Privileged {
		parts = append(parts, "--privileged")
	}
	if plan.IPCMode != "" {
		parts = append(parts, "--ipc", plan.IPCMode)
	}
	if plan.UTSMode != "" {
		parts = append(parts, "--uts", plan.UTSMode)
	}
	if plan.NetworkMode != "" {
		parts = append(parts, "--network", plan.NetworkMode)
	}
	if plan.ShmSize != "" {
		parts = append(parts, "--shm-size", plan.ShmSize)
	}
	for _, g := range plan.GroupAdd {
		parts = append(parts, "--group-add", g)
	}
	for _, s := range plan.SecurityOptions {
		parts = append(parts, "--security-opt", s)
	}
	for k, v := range plan.Ulimits {
		parts = append(parts, "--ulimit", fmt.Sprintf("%s=%s", k, v))
	}
	if len(plan.GPUDeviceIDs) > 0 {
		parts = append(parts, "--gpus", fmt.Sprintf("\"device=%s\"", strings.Join(plan.GPUDeviceIDs, ",")))
	}
	for _, d := range plan.Devices {
		if d.Permissions != "" {
			parts = append(parts, "--device", fmt.Sprintf("%s:%s:%s", d.HostPath, d.ContainerPath, d.Permissions))
		} else {
			parts = append(parts, "--device", fmt.Sprintf("%s:%s", d.HostPath, d.ContainerPath))
		}
	}
	for _, v := range plan.Mounts {
		ro := ""
		if v.Readonly {
			ro = ":ro"
		}
		parts = append(parts, "-v", fmt.Sprintf("%s:%s%s", v.HostPath, v.ContainerPath, ro))
	}
	for k, v := range plan.Env {
		parts = append(parts, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	if plan.HostPort > 0 && plan.NetworkMode != "host" {
		parts = append(parts, "-p", fmt.Sprintf("%d:%d/tcp", plan.HostPort, plan.ContainerPort))
	}
	for _, a := range plan.ExtraArgs {
		parts = append(parts, a)
	}
	// Image + optional explicit entrypoint + args.
	// When Entrypoint is nil (image_default mode), Docker preserves the
	// image's built-in ENTRYPOINT — only Cmd (args) is shown.
	parts = append(parts, plan.Image)
	if len(plan.Entrypoint) > 0 {
		parts = append(parts, "--entrypoint", strings.Join(plan.Entrypoint, " "))
	}
	parts = append(parts, plan.Args...)

	return strings.Join(parts, " ")
}
