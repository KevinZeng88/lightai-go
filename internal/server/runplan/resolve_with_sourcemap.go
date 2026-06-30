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

	sm.AddImage("launcher.image", plan.Image, "node_backend_runtime", "launcher.image", "NodeBackendRuntime", []SourceChainEntry{{
		Layer: "NodeBackendRuntime", Value: plan.Image, Reason: "resolved runtime image",
	}})

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
			source := pv.Source
			if source == "" {
				source = "node_backend_runtime"
			}
			sm.AddArg(pv.Key, pv.CliName, pv.Value, source, pv.Key, "NodeBackendRuntime", chain)
		}
	}
	for _, pv := range in.Deployment.ParameterValues {
		if !pv.Enabled {
			continue
		}
		chain := []SourceChainEntry{{
			Layer: "DeploymentConfigBundle", Value: pv.Value, Reason: "deployment override",
		}}
		sm.AddArg(pv.Key, pv.CliName, pv.Value, "deployment_override", pv.Key, "Deployment", chain)
	}
	if len(plan.Args) > 0 {
		sm.AddArg("launcher.command", "", plan.Args, "node_backend_runtime", "launcher.command", "NodeBackendRuntime", []SourceChainEntry{{
			Layer: "NodeBackendRuntime", Value: plan.Args, Reason: "final container command arguments",
		}})
	}

	// Env: track each resolved env variable
	for k, v := range plan.Env {
		source := "node_backend_runtime"
		layer := "NodeBackendRuntime"
		if plan.DeviceBinding != nil && k == plan.DeviceBinding.VisibleEnvKey && v == plan.DeviceBinding.VisibleEnvValue {
			source = "configedit_effect"
			layer = "DeploymentConfigEdit"
		}
		sm.AddEnv(k, v, source, "runtime.env", layer, []SourceChainEntry{{Layer: layer, Value: v, Reason: "resolved environment variable"}})
	}

	// Docker options: track each docker subfield
	if plan.ShmSize != "" {
		sm.AddDockerOption("docker.shm_size", plan.ShmSize, "node_backend_runtime", "launcher.docker_options.shm_size", "NodeBackendRuntime", nil)
	}
	if plan.IPCMode != "" {
		sm.AddDockerOption("docker.ipc_mode", plan.IPCMode, "node_backend_runtime", "launcher.docker_options.ipc_mode", "NodeBackendRuntime", nil)
	}
	if plan.NetworkMode != "" {
		sm.AddDockerOption("docker.network_mode", plan.NetworkMode, "node_backend_runtime", "launcher.docker_options.network_mode", "NodeBackendRuntime", nil)
	}
	if plan.Privileged {
		sm.AddDockerOption("docker.privileged", true, "node_backend_runtime", "launcher.docker_options.privileged", "NodeBackendRuntime", nil)
	}
	for _, d := range plan.Devices {
		sm.AddDevice(d.HostPath, d, "node_backend_runtime", "launcher.devices", "NodeBackendRuntime", nil)
	}
	for _, g := range plan.GroupAdd {
		sm.AddDockerOption("docker.group_add", g, "node_backend_runtime", "launcher.docker_options.group_add", "NodeBackendRuntime", nil)
	}

	// Mounts
	for _, m := range plan.Mounts {
		sm.AddMount("runtime.model_mount", m, "model_location", "runtime.model_mount", "ModelLocation", []SourceChainEntry{{Layer: "ModelLocation", Value: m, Reason: "model host path mounted into container"}})
	}

	// Health check
	if plan.HealthCheck.Path != "" {
		sm.AddHealthCheck("health_check.path", plan.HealthCheck.Path, "node_backend_runtime", "runtime.health.path", "NodeBackendRuntime", nil)
		sm.AddHealthCheck("health_check.expected_status", plan.HealthCheck.ExpectedStatus, "node_backend_runtime", "runtime.health.expected_status", "NodeBackendRuntime", nil)
		sm.AddHealthCheck("health_check.startup_timeout_seconds", plan.HealthCheck.StartupTimeoutSeconds, "node_backend_runtime", "runtime.health.startup_timeout_seconds", "NodeBackendRuntime", nil)
		sm.AddHealthCheck("health_check.interval_seconds", plan.HealthCheck.IntervalSeconds, "node_backend_runtime", "runtime.health.interval_seconds", "NodeBackendRuntime", nil)
		sm.AddHealthCheck("health_check.timeout_seconds", plan.HealthCheck.TimeoutSeconds, "node_backend_runtime", "runtime.health.timeout_seconds", "NodeBackendRuntime", nil)
	}

	// Ports
	sm.AddPort("service.container_port", plan.ContainerPort, "deployment_service", "service.container_port", "Deployment", nil)
	sm.AddPort("deployment.host_port", plan.HostPort, "deployment_service", "deployment.host_port", "Deployment", nil)

	// System generated
	if plan.DeviceBinding != nil {
		sm.AddDockerOption("runtime.device_binding", plan.DeviceBinding, "configedit_effect", "runtime.device_binding", "DeploymentConfigEdit", []SourceChainEntry{{Layer: "DeploymentConfigEdit", Value: plan.DeviceBinding, Reason: "device binding component effect"}})
		if len(plan.GPUDeviceIDs) > 0 {
			sm.AddDevice("runtime.device_binding.accelerator_ids", plan.GPUDeviceIDs, "configedit_effect", "runtime.device_binding", "DeploymentConfigEdit", nil)
		}
		if plan.DeviceBinding.DockerGPUOption != "" {
			sm.AddDockerOption("docker.gpus", plan.DeviceBinding.DockerGPUOption, "configedit_effect", "runtime.device_binding", "DeploymentConfigEdit", nil)
		}
		if plan.GPUVisibleEnvKey != "" && plan.DeviceBinding.VisibleEnvValue != "" {
			sm.AddEnv(plan.GPUVisibleEnvKey, plan.DeviceBinding.VisibleEnvValue, "configedit_effect", "runtime.device_binding", "DeploymentConfigEdit", nil)
		}
		if plan.GpuDriver != "" {
			sm.AddDockerOption("gpu_driver", plan.GpuDriver, "configedit_effect", "runtime.device_binding", "DeploymentConfigEdit", nil)
		}
	}

	return sm.Build()
}
