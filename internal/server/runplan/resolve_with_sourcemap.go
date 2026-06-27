package runplan

// ResolveWithSourceMap calls Resolve and then populates the ParameterSourceMap
// on the returned plan. This is the canonical entry point for preview, preflight,
// dry-run, and start — all paths must call this to ensure the source map is present.
func ResolveWithSourceMap(in ResolveInput) (*ResolvedRunPlan, []error, []string) {
	plan, errs, warns := Resolve(in)
	if plan == nil {
		return plan, errs, warns
	}

	// Build parameter source map from resolve inputs and resolved plan.
	sm := buildSourceMap(in, plan)
	plan.ParameterSourceMap = sm

	return plan, errs, warns
}

// buildSourceMap populates a ParameterSourceMap from the resolve inputs and resolved plan.
func buildSourceMap(in ResolveInput, plan *ResolvedRunPlan) *ParameterSourceMap {
	sm := NewSourceMapBuilder()

	// Args: track each parameter value's source from NBR and deployment
	nbrParams := in.NBRConfigSnapshot
	if nbrParams != nil {
		for _, pv := range nbrParams.ParameterValues {
			if !pv.Enabled {
				continue
			}
			chain := []SourceChainEntry{{
				Layer: "NodeBackendRuntimeConfigBundle", Value: pv.Value, Reason: "nbr parameter value",
			}}
			sm.AddArg(pv.Key, pv.CliName, pv.Value, pv.Source, "", pv.Source, chain)
		}
	}
	for _, pv := range in.Deployment.ParameterValues {
		if !pv.Enabled {
			continue
		}
		chain := []SourceChainEntry{{
			Layer: "DeploymentConfigBundle", Value: pv.Value, Reason: "deployment override",
		}}
		sm.AddArg(pv.Key, pv.CliName, pv.Value, "deployment_override", "", "DeploymentConfigBundle", chain)
	}

	// Env: track each resolved env variable
	for k, v := range plan.Env {
		sm.AddEnv(k, v, "resolved", "", "ResolvedRunPlan", nil)
	}

	// Docker options: track each docker subfield
	if plan.ShmSize != "" {
		sm.AddDockerOption("docker.shm_size", plan.ShmSize, "backend_runtime", "", "", nil)
	}
	if plan.IPCMode != "" {
		sm.AddDockerOption("docker.ipc_mode", plan.IPCMode, "backend_runtime", "", "", nil)
	}
	if plan.NetworkMode != "" {
		sm.AddDockerOption("docker.network_mode", plan.NetworkMode, "backend_runtime", "", "", nil)
	}
	if plan.Privileged {
		sm.AddDockerOption("docker.privileged", true, "backend_runtime", "", "", nil)
	}
	for _, d := range plan.Devices {
		sm.AddDevice(d.HostPath, d, "system_generated", "", "", nil)
	}
	for _, g := range plan.GroupAdd {
		sm.AddDockerOption("docker.group_add", g, "backend_runtime", "", "", nil)
	}

	// Mounts
	for _, m := range plan.Mounts {
		sm.AddMount(m.HostPath, m, "model_location", "", "", nil)
	}

	// Health check
	if plan.HealthCheck.Path != "" {
		sm.AddHealthCheck("health_check.path", plan.HealthCheck.Path, "backend_runtime", "", "", nil)
		sm.AddHealthCheck("health_check.expected_status", plan.HealthCheck.ExpectedStatus, "backend_runtime", "", "", nil)
	}

	// Ports
	sm.AddPort("container_port", plan.ContainerPort, "backend_version", "", "", nil)
	sm.AddPort("host_port", plan.HostPort, "deployment_service", "", "", nil)

	// System generated
	if len(plan.GPUDeviceIDs) > 0 {
		sm.AddSystemGenerated("gpu_device_ids", plan.GPUDeviceIDs, "system_generated", "", "", nil)
	}
	if plan.GPUVisibleEnvKey != "" {
		sm.AddSystemGenerated("gpu_visible_env_key", plan.GPUVisibleEnvKey, "system_generated", "", "", nil)
	}
	if plan.GpuDriver != "" {
		sm.AddSystemGenerated("gpu_driver", plan.GpuDriver, "system_generated", "", "", nil)
	}

	return sm.Build()
}
