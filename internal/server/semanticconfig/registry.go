package semanticconfig

type Registry struct {
	defs      map[string]Definition
	legacyMap map[string]string
}

func DefaultRegistry() *Registry {
	defs := []Definition{
		{
			Key:           "runtime.image_ref",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeString,
			DisplayTier:   TierRequired,
			Label:         "Image",
			LegacyKeys:    []string{"launcher.image", "image_ref"},
			DefaultSource: "Runtime YAML image candidates; node image picker at NBR create",
		},
		{
			Key:           "runtime.command",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeArray,
			DisplayTier:   TierDiagnostic,
			Label:         "Command",
			LegacyKeys:    []string{"launcher.command", "default_args"},
			DefaultSource: "Backend/runtime YAML command/profile",
		},
		{
			Key:           "runtime.entrypoint",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeArray,
			DisplayTier:   TierDiagnostic,
			Label:         "Entrypoint",
			LegacyKeys:    []string{"launcher.entrypoint", "default_entrypoint"},
			DefaultSource: "Backend/runtime YAML entrypoint",
		},
		{
			Key:           "runtime.env",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeObject,
			DisplayTier:   TierCommon,
			Label:         "Environment variables",
			LegacyKeys:    []string{"env", "env_overrides"},
			DefaultSource: "Runtime YAML env/env_schema",
		},
		{
			Key:           "service.listen_host",
			Owner:         OwnerRuntimeService,
			ValueType:     TypeString,
			DisplayTier:   TierCommon,
			Label:         "Container listen host",
			LegacyKeys:    []string{"backend.common.host", "launcher.listen_host"},
			DefaultSource: "BackendVersion default host or adapter default",
			ResolverMappings: map[string]string{
				"vllm":     "--host",
				"sglang":   "--host",
				"llamacpp": "--host",
			},
		},
		{
			Key:           "service.container_port",
			Owner:         OwnerRuntimeService,
			ValueType:     TypeInteger,
			DisplayTier:   TierCommon,
			Label:         "Container listen port",
			LegacyKeys:    []string{"backend.common.port", "launcher.container_port", "service_json.container_port"},
			DefaultSource: "BackendVersion default port; runtime port defaults",
			ResolverMappings: map[string]string{
				"vllm":     "--port",
				"sglang":   "--port",
				"llamacpp": "--port",
			},
		},
		{
			Key:           "deployment.host_port",
			Owner:         OwnerDeploymentExposure,
			ValueType:     TypeInteger,
			DisplayTier:   TierCommon,
			Label:         "Host port",
			LegacyKeys:    []string{"host_port", "service.host_port", "service_json.host_port"},
			DefaultSource: "User input or allocator recommendation",
		},
		{
			Key:           "deployment.served_model_name",
			Owner:         OwnerDeploymentService,
			ValueType:     TypeString,
			DisplayTier:   TierCommon,
			Label:         "Served model name",
			LegacyKeys:    []string{"backend.common.served_model_name", "backend.arg.served_model_name", "served_model_name"},
			DefaultSource: "Artifact name",
			ResolverMappings: map[string]string{
				"vllm":   "--served-model-name",
				"sglang": "--served-model-name",
			},
		},
		{
			Key:           "model_runtime.context_length",
			Owner:         OwnerModelArtifact,
			ValueType:     TypeInteger,
			DisplayTier:   TierRecommended,
			Label:         "Model context length",
			LegacyKeys:    []string{"context_length", "backend.arg.context_length", "default_context_length"},
			DefaultSource: "Scanner/HF/GGUF metadata",
		},
		{
			Key:           "model_runtime.max_model_len",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeInteger,
			DisplayTier:   TierDeploymentCommonAdvanced,
			Label:         "Max model length",
			LegacyKeys:    []string{"backend.arg.max_model_len", "max_model_len", "--max-model-len"},
			DefaultSource: "model_runtime.context_length / ModelArtifact facts / backend recommendation",
			WarningRules:  []string{"above model context length", "estimated VRAM risk"},
			ResolverMappings: map[string]string{
				"vllm":     "--max-model-len",
				"sglang":   "--context-length",
				"llamacpp": "--ctx-size",
			},
		},
		{
			Key:           "model_runtime.gpu_memory_utilization",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeNumber,
			DisplayTier:   TierAdvanced,
			Label:         "GPU memory utilization",
			LegacyKeys:    []string{"backend.arg.gpu_memory_utilization", "gpu_memory_utilization"},
			DefaultSource: "Backend recommendation",
		},
		{
			Key:           "runtime.health",
			Owner:         OwnerRuntimeService,
			ValueType:     TypeObject,
			DisplayTier:   TierRecommended,
			Label:         "Health check",
			LegacyKeys:    []string{"health_check", "default_health_check"},
			DefaultSource: "Backend/runtime YAML health",
		},
		{
			Key:           "runtime.model_mount",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeObject,
			DisplayTier:   TierCommon,
			Label:         "Model mount",
			LegacyKeys:    []string{"model_mount", "default_model_mount"},
			DefaultSource: "Runtime/backend YAML model mount",
		},
		{
			Key:           "docker.shm_size",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeString,
			DisplayTier:   TierCommon,
			Label:         "Shared memory",
			LegacyKeys:    []string{"launcher.docker_options.shm_size", "docker_options.shm_size"},
			DefaultSource: "Runtime YAML Docker options",
		},
		{
			Key:           "docker.ipc_mode",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeString,
			DisplayTier:   TierAdvanced,
			Label:         "IPC mode",
			LegacyKeys:    []string{"launcher.docker_options.ipc_mode", "docker_options.ipc_mode"},
			DefaultSource: "Runtime YAML Docker options",
		},
		{
			Key:           "docker.privileged",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeBoolean,
			DisplayTier:   TierAdvanced,
			Label:         "Privileged container",
			LegacyKeys:    []string{"launcher.docker_options.privileged", "docker_options.privileged"},
			DefaultSource: "Runtime YAML Docker options",
		},
		{
			Key:           "docker.network_mode",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeString,
			DisplayTier:   TierAdvanced,
			Label:         "Network mode",
			LegacyKeys:    []string{"launcher.docker_options.network_mode", "docker_options.network_mode"},
			DefaultSource: "Runtime YAML Docker options",
		},
		{
			Key:           "docker.devices",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeArray,
			DisplayTier:   TierAdvanced,
			Label:         "Docker devices",
			LegacyKeys:    []string{"launcher.docker_options.devices", "docker_options.devices", "devices"},
			DefaultSource: "Runtime YAML Docker options",
		},
		{
			Key:           "docker.optional_devices",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeArray,
			DisplayTier:   TierAdvanced,
			Label:         "Optional Docker devices",
			LegacyKeys:    []string{"launcher.docker_options.optional_devices", "docker_options.optional_devices", "optional_devices"},
			DefaultSource: "Runtime YAML Docker options",
		},
		{
			Key:           "docker.group_add",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeArray,
			DisplayTier:   TierAdvanced,
			Label:         "Additional groups",
			LegacyKeys:    []string{"launcher.docker_options.group_add", "docker_options.group_add", "group_add"},
			DefaultSource: "Runtime YAML Docker options",
		},
	}
	reg := &Registry{defs: map[string]Definition{}, legacyMap: map[string]string{}}
	for _, def := range defs {
		reg.defs[def.Key] = def
		for _, legacy := range def.LegacyKeys {
			reg.legacyMap[legacy] = def.Key
		}
	}
	return reg
}

func (r *Registry) Get(key string) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	def, ok := r.defs[key]
	return def, ok
}

func (r *Registry) CanonicalKey(key string) (string, bool) {
	if r == nil {
		return "", false
	}
	if _, ok := r.defs[key]; ok {
		return key, true
	}
	canonical, ok := r.legacyMap[key]
	return canonical, ok
}

func (r *Registry) IsLegacyKey(key string) bool {
	if r == nil {
		return false
	}
	_, ok := r.legacyMap[key]
	return ok
}
