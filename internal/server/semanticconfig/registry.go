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
			Key:           "model_runtime.dtype",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeString,
			DisplayTier:   TierCommon,
			Label:         "Data type",
			LegacyKeys:    []string{"backend.arg.dtype", "dtype", "--dtype"},
			DefaultSource: "Backend recommendation",
		},
		{
			Key:           "model_runtime.tensor_parallel_size",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeInteger,
			DisplayTier:   TierCommon,
			Label:         "Tensor parallel size",
			LegacyKeys:    []string{"backend.arg.tensor_parallel_size", "tensor_parallel_size", "--tensor-parallel-size"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.pipeline_parallel_size",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeInteger,
			DisplayTier:   TierAdvanced,
			Label:         "Pipeline parallel size",
			LegacyKeys:    []string{"backend.arg.pipeline_parallel_size", "pipeline_parallel_size", "--pipeline-parallel-size"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.max_num_batched_tokens",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeInteger,
			DisplayTier:   TierAdvanced,
			Label:         "Max batched tokens",
			LegacyKeys:    []string{"backend.arg.max_num_batched_tokens", "max_num_batched_tokens", "--max-num-batched-tokens"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.max_num_seqs",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeInteger,
			DisplayTier:   TierAdvanced,
			Label:         "Max concurrent sequences",
			LegacyKeys:    []string{"backend.arg.max_num_seqs", "max_num_seqs", "--max-num-seqs"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.kv_cache_dtype",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeString,
			DisplayTier:   TierAdvanced,
			Label:         "KV cache data type",
			LegacyKeys:    []string{"backend.arg.kv_cache_dtype", "kv_cache_dtype", "--kv-cache-dtype"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.cpu_offload_gb",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeNumber,
			DisplayTier:   TierAdvanced,
			Label:         "CPU offload capacity",
			LegacyKeys:    []string{"backend.arg.cpu_offload_gb", "cpu_offload_gb", "--cpu-offload-gb"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.swap_space",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeNumber,
			DisplayTier:   TierAdvanced,
			Label:         "Swap space",
			LegacyKeys:    []string{"backend.arg.swap_space", "swap_space", "--swap-space"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.enforce_eager",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeBoolean,
			DisplayTier:   TierDiagnostic,
			Label:         "Enforce eager mode",
			LegacyKeys:    []string{"backend.arg.enforce_eager", "enforce_eager", "--enforce-eager"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.trust_remote_code",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeBoolean,
			DisplayTier:   TierDiagnostic,
			Label:         "Trust remote code",
			LegacyKeys:    []string{"backend.arg.trust_remote_code", "trust_remote_code", "--trust-remote-code"},
			DefaultSource: "User security decision",
		},
		{
			Key:           "model_runtime.safetensors_load_strategy",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeString,
			DisplayTier:   TierAdvanced,
			Label:         "Safetensors load strategy",
			LegacyKeys:    []string{"backend.arg.safetensors_load_strategy", "safetensors_load_strategy", "--safetensors-load-strategy"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.download_dir",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeString,
			DisplayTier:   TierDiagnostic,
			Label:         "Model download directory",
			LegacyKeys:    []string{"backend.arg.download_dir", "download_dir", "--download-dir"},
			DefaultSource: "Backend recommendation or user override",
		},
		{
			Key:           "model_runtime.model",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeString,
			DisplayTier:   TierDiagnostic,
			Label:         "Model path",
			LegacyKeys:    []string{"backend.arg.model", "model", "--model"},
			DefaultSource: "Model location injection",
		},
		{
			Key:           "model_runtime.host",
			Owner:         OwnerRuntimeService,
			ValueType:     TypeString,
			DisplayTier:   TierDiagnostic,
			Label:         "Listen host",
			LegacyKeys:    []string{"backend.arg.host", "host", "--host"},
			DefaultSource: "BackendVersion default host or adapter default",
		},
		{
			Key:           "model_runtime.port",
			Owner:         OwnerRuntimeService,
			ValueType:     TypeInteger,
			DisplayTier:   TierDiagnostic,
			Label:         "Service port",
			LegacyKeys:    []string{"backend.arg.port", "port", "--port"},
			DefaultSource: "BackendVersion default port; runtime port defaults",
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
			Key:           "runtime.extra_env",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeString,
			DisplayTier:   TierAdvanced,
			Label:         "Extra environment variables",
			LegacyKeys:    []string{"extra_env", "env_lines"},
			DefaultSource: "User runtime overrides",
		},
		{
			Key:           "backend.extra_args",
			Owner:         OwnerModelRuntime,
			ValueType:     TypeString,
			DisplayTier:   TierAdvanced,
			Label:         "Extra launch arguments",
			LegacyKeys:    []string{"extra_args", "backend.arg.extra_args"},
			DefaultSource: "User runtime overrides",
		},
		{
			Key:           "launcher.kind",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeString,
			DisplayTier:   TierDiagnostic,
			Label:         "Launcher type",
			LegacyKeys:    []string{"runner_type", "launcher_type"},
			DefaultSource: "Runtime YAML",
		},
		{
			Key:           "launcher.devices",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeArray,
			DisplayTier:   TierAdvanced,
			Label:         "Device bindings",
			LegacyKeys:    []string{"devices"},
			DefaultSource: "Runtime YAML or vendor adapter",
		},
		{
			Key:           "launcher.ports",
			Owner:         OwnerRuntimeService,
			ValueType:     TypeArray,
			DisplayTier:   TierAdvanced,
			Label:         "Port mappings",
			LegacyKeys:    []string{"ports"},
			DefaultSource: "Runtime YAML",
		},
		{
			Key:           "launcher.volumes",
			Owner:         OwnerRuntimeEnvironment,
			ValueType:     TypeArray,
			DisplayTier:   TierAdvanced,
			Label:         "Volume mounts",
			LegacyKeys:    []string{"volumes"},
			DefaultSource: "Runtime YAML or model location injection",
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
