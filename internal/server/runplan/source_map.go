package runplan

// ============================================================================
// Parameter source map builder — shared by preview, preflight, dry-run, start
// ============================================================================

// SourceMapBuilder accumulates ParameterSourceEntry values during RunPlan resolution.
// It is consumed by Resolve() to produce the final ParameterSourceMap.
type SourceMapBuilder struct {
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

// AddArg records an arg parameter with its source chain.
func (b *SourceMapBuilder) AddArg(key, flag string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.args = append(b.args, ParameterSourceEntry{
		Key:             key,
		Target:          "args",
		Arg:             flag,
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// AddEnv records an env parameter with its source chain.
func (b *SourceMapBuilder) AddEnv(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.env = append(b.env, ParameterSourceEntry{
		Key:             key,
		Target:          "env",
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// AddMount records a mount with its source chain.
func (b *SourceMapBuilder) AddMount(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.mounts = append(b.mounts, ParameterSourceEntry{
		Key:             key,
		Target:          "mounts",
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// AddPort records a port mapping with its source chain.
func (b *SourceMapBuilder) AddPort(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.ports = append(b.ports, ParameterSourceEntry{
		Key:             key,
		Target:          "ports",
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// AddDevice records a device mapping with its source chain.
func (b *SourceMapBuilder) AddDevice(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.devices = append(b.devices, ParameterSourceEntry{
		Key:             key,
		Target:          "devices",
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// AddDockerOption records a Docker option with its source chain.
func (b *SourceMapBuilder) AddDockerOption(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.dockerOptions = append(b.dockerOptions, ParameterSourceEntry{
		Key:             key,
		Target:          "docker_options",
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// AddHealthCheck records a health check field with its source chain.
func (b *SourceMapBuilder) AddHealthCheck(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.healthCheck = append(b.healthCheck, ParameterSourceEntry{
		Key:             key,
		Target:          "health_check",
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// AddSystemGenerated records a system-generated field with its source chain.
func (b *SourceMapBuilder) AddSystemGenerated(key string, value any, effectiveSource, configSetKey, lastValueLayer string, chain []SourceChainEntry) {
	b.systemGenerated = append(b.systemGenerated, ParameterSourceEntry{
		Key:             key,
		Target:          "system_generated",
		Value:           value,
		EffectiveSource: effectiveSource,
		ConfigSetKey:    configSetKey,
		LastValueLayer:  lastValueLayer,
		SourceChain:     chain,
	})
}

// Build returns the assembled ParameterSourceMap.
func (b *SourceMapBuilder) Build() *ParameterSourceMap {
	return &ParameterSourceMap{
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
