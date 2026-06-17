package runtime

// ConvertRunplanToAgentSpec converts a server-generated ResolvedRunPlan (from runplan package)
// into the Agent's AgentRunSpec for Docker container execution.
//
// The AgentRunSpec struct is kept JSON-compatible with the server's ResolvedRunPlan
// so the Agent can deserialize it directly from task payloads.
func ConvertRunplanToAgentSpec(plan PlanInput) AgentRunSpec {
	spec := AgentRunSpec{
		InstanceID:       plan.InstanceID,
		DeploymentID:     plan.DeploymentID,
		RuntimeType:      "docker",
		ModelPath:        plan.ModelPath,
		ServedModelName:  plan.ServedModelName,
		NodeID:           plan.NodeID,
		AgentID:          plan.AgentID,
		GPUDeviceIDs:     plan.GPUDeviceIDs,
		GPUVisibleEnvKey: plan.GPUVisibleEnvKey,
		Env:              plan.Env,
		Args:             plan.Args,
		HostPort:         plan.HostPort,
		ContainerPort:    plan.ContainerPort,
		Docker: DockerSpec{
			Image:           plan.Image,
			ContainerName:   plan.ContainerName,
			Command:         plan.Entrypoint,
			Args:            plan.Args,
			Privileged:      plan.Privileged,
			IPCMode:         plan.IPCMode,
			UTSMode:         plan.UTSMode,
			NetworkMode:     plan.NetworkMode,
			ShmSize:         plan.ShmSize,
			SecurityOptions: plan.SecurityOptions,
			Ulimits:         plan.Ulimits,
			GPUDeviceIDs:    plan.GPUDeviceIDs,
		},
	}

	// Map volumes from resolved mounts.
	for _, m := range plan.Mounts {
		spec.Volumes = append(spec.Volumes, VolumeSpec{
			HostPath:      m.HostPath,
			ContainerPath: m.ContainerPath,
			Readonly:      m.Readonly,
		})
	}

	// Map devices.
	for _, d := range plan.Devices {
		spec.Devices = append(spec.Devices, DeviceSpec{
			HostPath:      d.HostPath,
			ContainerPath: d.ContainerPath,
			Permissions:   d.Permissions,
		})
	}

	// Map ports.
	if spec.HostPort > 0 {
		spec.Ports = append(spec.Ports, PortSpec{
			HostPort:      spec.HostPort,
			ContainerPort: spec.ContainerPort,
			Protocol:      "tcp",
		})
	}

	return spec
}

// PlanInput is a minimal interface for plan data that avoids importing
// the server-side runplan package into the agent.
type PlanInput struct {
	InstanceID       string
	DeploymentID     string
	NodeID           string
	AgentID          string
	ModelPath        string
	ServedModelName  string
	Image            string
	ContainerName    string
	Entrypoint       []string
	Args             []string
	Env              map[string]string
	Privileged       bool
	IPCMode          string
	UTSMode          string
	NetworkMode      string
	ShmSize          string
	Ulimits          map[string]string
	Devices          []PlanDevice
	Mounts           []PlanMount
	HostPort         int
	ContainerPort    int
	GPUDeviceIDs     []string
	GPUVisibleEnvKey string
	SecurityOptions  []string
}

// PlanDevice mirrors runplan.DeviceMapping.
type PlanDevice struct {
	HostPath      string
	ContainerPath string
	Permissions   string
}

// PlanMount mirrors runplan.MountMapping.
type PlanMount struct {
	HostPath      string
	ContainerPath string
	Readonly      bool
}
