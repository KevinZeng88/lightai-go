package runplan

// ============================================================================
// Parameter source map builder — shared by preview, preflight, dry-run, start
// ============================================================================

// SourceMapBuilder accumulates ParameterSourceEntry values during RunPlan resolution.
// It is consumed by Resolve() to produce the final ParameterSourceMap.
type SourceMapBuilder struct {
	image            []ParameterSourceEntry
	args             []ParameterSourceEntry
	env              []ParameterSourceEntry
	mounts           []ParameterSourceEntry
	ports            []ParameterSourceEntry
	devices          []ParameterSourceEntry
	dockerOptions    []ParameterSourceEntry
	healthCheck      []ParameterSourceEntry
	resourceControls []ParameterSourceEntry
	systemGenerated  []ParameterSourceEntry
}

// NewSourceMapBuilder creates an empty source map builder.
func NewSourceMapBuilder() *SourceMapBuilder {
	return &SourceMapBuilder{}
}

// AddImage records the resolved container image source.
func (b *SourceMapBuilder) AddImage(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.image = append(b.image, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "image",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddArg records an arg parameter with its source chain.
func (b *SourceMapBuilder) AddArg(key, flag string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.args = append(b.args, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "args",
		Arg:             flag,
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddEnv records an env parameter with its source chain.
func (b *SourceMapBuilder) AddEnv(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.env = append(b.env, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "env",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddMount records a mount with its source chain.
func (b *SourceMapBuilder) AddMount(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.mounts = append(b.mounts, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "mounts",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddPort records a port mapping with its source chain.
func (b *SourceMapBuilder) AddPort(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.ports = append(b.ports, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "ports",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddDevice records a device mapping with its source chain.
func (b *SourceMapBuilder) AddDevice(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.devices = append(b.devices, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "devices",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddDockerOption records a Docker option with its source chain.
func (b *SourceMapBuilder) AddDockerOption(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.dockerOptions = append(b.dockerOptions, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "docker_options",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddHealthCheck records a health check field with its source chain.
func (b *SourceMapBuilder) AddHealthCheck(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.healthCheck = append(b.healthCheck, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "health_check",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// AddSystemGenerated records a system-generated field with its source chain.
func (b *SourceMapBuilder) AddSystemGenerated(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.systemGenerated = append(b.systemGenerated, enrichSourceEntry(ParameterSourceEntry{
		Key:             key,
		Target:          "system_generated",
		Value:           value,
		FinalValue:      value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	}))
}

// Build returns the assembled ParameterSourceMap.
func (b *SourceMapBuilder) Build() *ParameterSourceMap {
	return &ParameterSourceMap{
		Image:            b.image,
		Args:             b.args,
		Env:              b.env,
		Mounts:           b.mounts,
		Ports:            b.ports,
		Devices:          b.devices,
		DockerOptions:    b.dockerOptions,
		HealthCheck:      b.healthCheck,
		ResourceControls: b.resourceControls,
		SystemGenerated:  b.systemGenerated,
	}
}

func enrichSourceEntry(e ParameterSourceEntry) ParameterSourceEntry {
	if len(e.Path) == 0 {
		e.Path = sourceEntryPath(e)
	}
	if e.SourceLayer == "" {
		e.SourceLayer = sourceLayerFor(e.EffectiveSource, e.LastValueLayer)
	}
	if e.SourceKind == "" {
		e.SourceKind = sourceKindFor(e.EffectiveSource)
	}
	if e.PatchTarget == "" {
		e.PatchTarget = sourcePatchTarget(e)
	}
	if e.DockerEffect == "" {
		e.DockerEffect = sourceDockerEffect(e)
	}
	if e.Severity == "" {
		e.Severity = "info"
	}
	if e.Reason == "" {
		e.Reason = sourceReason(e)
	}
	if !e.UserEditable && e.ReadonlyReason == "" && (e.EffectiveSource == "system_generated" || e.EffectiveSource == "derived" || e.EffectiveSource == "model_location") {
		e.ReadonlyReason = "derived from deployment selection or node inventory"
	}
	if e.EffectiveSource == "deployment_override" || e.EffectiveSource == "deployment_service" || e.EffectiveSource == "node_backend_runtime" || e.EffectiveSource == "backend_runtime" || e.EffectiveSource == "configedit_effect" {
		e.UserEditable = true
	}
	if e.EffectiveSource == "system_generated" || e.EffectiveSource == "derived" || e.EffectiveSource == "model_location" {
		e.Derived = true
	}
	return e
}

func sourceEntryPath(e ParameterSourceEntry) []string {
	if e.Key == "" {
		return []string{e.Target}
	}
	return []string{e.Target, e.Key}
}

func sourceLayerFor(source, fallback string) string {
	if fallback != "" {
		return fallback
	}
	switch source {
	case "backend_version":
		return "BackendVersion"
	case "backend_runtime":
		return "BackendRuntime"
	case "node_backend_runtime":
		return "NodeBackendRuntime"
	case "deployment_override", "deployment_service":
		return "ModelDeployment"
	case "model_location":
		return "ModelLocation"
	case "system_generated", "derived", "node_inventory":
		return "RunPlanResolver"
	case "configedit_effect":
		return "DeploymentConfigEdit"
	default:
		return source
	}
}

func sourceKindFor(source string) string {
	switch source {
	case "deployment_override":
		return "user_override"
	case "deployment_service":
		return "deployment_selection"
	case "backend_runtime", "node_backend_runtime":
		return "inherited"
	case "backend_version":
		return "system_default"
	case "model_location":
		return "deployment_selection"
	case "system_generated", "derived", "node_inventory":
		return source
	case "configedit_effect":
		return "configedit_component_effect"
	default:
		return source
	}
}

func sourcePatchTarget(e ParameterSourceEntry) string {
	switch e.EffectiveSource {
	case "deployment_override":
		return "deployment.config_overrides"
	case "deployment_service":
		return "deployment.service_json"
	case "node_backend_runtime":
		return "node_backend_runtime.config_snapshot_json"
	case "backend_runtime":
		return "backend_runtime.config_set_json"
	case "model_location":
		return "model_location"
	case "system_generated", "derived", "node_inventory":
		return "deployment.placement_json"
	case "configedit_effect":
		return "runtime.device_binding"
	default:
		if e.ConfigSetKey != "" {
			return e.ConfigSetKey
		}
		return ""
	}
}

func sourceDockerEffect(e ParameterSourceEntry) string {
	switch e.Target {
	case "image":
		return "docker image"
	case "args":
		if e.Arg != "" {
			return e.Arg
		}
		return "container command"
	case "env":
		return "-e " + e.Key
	case "mounts":
		return "-v"
	case "ports":
		return "-p"
	case "devices":
		return "--device"
	case "docker_options":
		switch e.Key {
		case "docker.shm_size":
			return "--shm-size"
		case "docker.ipc_mode":
			return "--ipc"
		case "docker.network_mode":
			return "--network"
		case "docker.privileged":
			return "--privileged"
		case "docker.gpus", "runtime.device_binding":
			return "--gpus"
		default:
			return "docker option"
		}
	case "health_check":
		return "container health check"
	case "system_generated":
		if e.Key == "docker.gpus" || e.Key == "gpu_device_ids" {
			return "--gpus"
		}
		if e.Key == "gpu_visible_env" || e.Key == "gpu_visible_env_key" {
			return "-e"
		}
		return "system generated"
	default:
		return ""
	}
}

func sourceReason(e ParameterSourceEntry) string {
	if len(e.SourceChain) > 0 && e.SourceChain[len(e.SourceChain)-1].Reason != "" {
		return e.SourceChain[len(e.SourceChain)-1].Reason
	}
	switch e.EffectiveSource {
	case "deployment_override":
		return "deployment override"
	case "deployment_service":
		return "deployment service setting"
	case "node_backend_runtime":
		return "node runtime configuration snapshot"
	case "backend_runtime":
		return "runtime template snapshot"
	case "backend_version":
		return "backend version default"
	case "model_location":
		return "resolved model location"
	case "system_generated", "derived":
		return "derived by runplan resolver"
	case "configedit_effect":
		return "compiled from ConfigEdit component effect"
	default:
		return e.EffectiveSource
	}
}
