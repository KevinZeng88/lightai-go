package runplan

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"lightai-go/internal/common/log"
)

// ResolveInput holds all data needed for RunPlan resolution.
type ResolveInput struct {
	Backend             *BackendInfo
	BackendVersion      *VersionInfo
	BackendRuntime      *RuntimeInfo
	NodeRuntimeOverride *NodeOverrideInfo // nil if none
	Artifact            *ArtifactInfo
	Deployment          *DeploymentInfo
	InstanceID          string
	Node                *NodeInfo
	AssignedGPUs        []GPUInfo
	ProcessStartConfig  *ProcessStartConfig // nil if none (legacy behavior)
}

// BackendInfo is the minimal backend data needed for resolution.
type BackendInfo struct {
	ID             string
	Name           string
	DefaultVersion string
	DefaultEnv     map[string]string
}

// VersionInfo is the minimal version data needed for resolution.
type VersionInfo struct {
	ID                   string
	Version              string
	DefaultEntrypoint    []string
	DefaultArgs          []string
	DefaultBackendParams []string
	ParameterDefs        []ParameterDef
	HealthCheck          HealthCheckInput
	DefaultContainerPort int
	DefaultImages        map[string]string // vendor → image
	Env                  map[string]string
}

// ParameterDef defines a configurable parameter.
type ParameterDef struct {
	Name     string      `json:"name"`
	CliName  string      `json:"cli_name"`
	Alias    string      `json:"alias"`
	Type     string      `json:"type"`
	Default  interface{} `json:"default"`
	Required bool        `json:"required"`
}

// effectiveCliName returns the CLI name for this parameter, preferring
// CliName, then Alias, then Name.
func (d *ParameterDef) effectiveCliName() string {
	if d.CliName != "" {
		return d.CliName
	}
	if d.Alias != "" {
		return d.Alias
	}
	return d.Name
}

// HealthCheckInput is the health check configuration.
type HealthCheckInput struct {
	Path                  string `json:"path"`
	ExpectedStatus        int    `json:"expected_status"`
	StartupTimeoutSeconds int    `json:"startup_timeout_seconds"`
	IntervalSeconds       int    `json:"interval_seconds"`
	TimeoutSeconds        int    `json:"timeout_seconds"`
}

// RuntimeInfo is the minimal runtime data needed for resolution.
type RuntimeInfo struct {
	ID                  string
	Vendor              string
	RuntimeType         string
	ImageName           string
	EntrypointOverride  []string
	ArgsOverride        []string
	DefaultEnv          map[string]string
	Docker              DockerSpecInfo
	ModelMount          ModelMountInfo
	HealthCheckOverride *HealthCheckInput
}

// DockerSpecInfo holds Docker runtime configuration.
// All fields come from docker_json in the catalog, NBR, or user config.
// No field has an implicit code default — defaults are in the catalog seed or YAML.
type DockerSpecInfo struct {
	Privileged       bool              `json:"privileged"`
	IPCMode          string            `json:"ipc_mode"`
	UTSMode          string            `json:"uts_mode"`
	NetworkMode      string            `json:"network_mode"`
	ShmSize          string            `json:"shm_size"`
	GPUVisibleEnvKey string            `json:"gpu_visible_env_key"`
	Ulimits          map[string]string `json:"ulimits"`
	SecurityOptions  []string          `json:"security_options"`
	Devices          []DeviceMapping   `json:"devices"`
	GroupAdd         []string          `json:"group_add"`
	// GPU driver for DeviceRequest. Empty string ("") matches docker run --gpus CLI.
	// Set per vendor in catalog: NVIDIA uses "", MetaX/Huawei use raw devices (no DeviceRequest).
	GpuDriver       string     `json:"gpu_driver,omitempty"`
	GpuCapabilities [][]string `json:"gpu_capabilities,omitempty"` // e.g. [["gpu"]], [["gpu","compute"]]
}

// ModelMountInfo holds model mount configuration.
type ModelMountInfo struct {
	ContainerPath string `json:"container_path"`
	Readonly      bool   `json:"readonly"`
}

// NodeOverrideInfo is the minimal node override data.
type NodeOverrideInfo struct {
	ImageName         string
	ImagePullPolicy   string
	Env               map[string]string
	DockerOverride    *DockerSpecInfo
	ModelRootHostPath string
}

// ArtifactInfo is the minimal artifact data.
type ArtifactInfo struct {
	ID           string
	Name         string
	Path         string
	ModelRoot    string
	RelativePath string
}

// DeploymentInfo is the minimal deployment data.
type DeploymentInfo struct {
	ID           string
	Name         string
	Parameters   map[string]interface{}
	EnvOverrides map[string]string
	Placement    PlacementInfo
	Service      ServiceInfo
}

// PlacementInfo holds deployment placement configuration.
type PlacementInfo struct {
	NodeID         string   `json:"node_id"`
	AcceleratorIds []string `json:"accelerator_ids"`
}

// ServiceInfo holds deployment service configuration.
type ServiceInfo struct {
	HostPort      int `json:"host_port"`
	ContainerPort int `json:"container_port,omitempty"`
	AppPort       int `json:"app_port,omitempty"`
	HealthPort    int `json:"health_port,omitempty"`
	APITestPort   int `json:"api_test_port,omitempty"`
}

// NodeInfo holds node data.
type NodeInfo struct {
	ID string
	IP string
}

// GPUInfo holds GPU device data.
type GPUInfo struct {
	Index  int
	Vendor string
}

// Resolve generates a ResolvedRunPlan from the given inputs.
func Resolve(in ResolveInput) (*ResolvedRunPlan, []error, []string) {
	startTime := time.Now()
	log.Info("runplan resolve: begin", "backend", in.Backend.Name, "vendor", in.BackendRuntime.Vendor)
	var errors []error
	var warnings []string

	// 1. Validate runtime_type == "docker".
	if in.BackendRuntime.RuntimeType != "docker" {
		errors = append(errors, fmt.Errorf("unsupported runtime_type: %s (only docker is supported)", in.BackendRuntime.RuntimeType))
		return nil, errors, warnings
	}

	// 2. Build variable map for template substitution.
	vars := buildVarMap(in)

	// 3. Resolve image: NodeOverride > BackendRuntime > BackendVersion.defaultImages[vendor] > error.
	image, imgWarns := resolveImage(in)
	warnings = append(warnings, imgWarns...)
	if image == "" {
		errors = append(errors, fmt.Errorf("no image available: configure NodeRuntimeOverride.image_name, BackendRuntime.image_name, or BackendVersion.default_images[%s]", in.BackendRuntime.Vendor))
		return nil, errors, warnings
	}

	// 4. Resolve entrypoint.
	// Phase 3: process_start_config takes priority over legacy mechanism.
	// Missing process_start_config → legacy behavior (BR override > BV default).
	entrypoint := in.BackendVersion.DefaultEntrypoint
	if len(in.BackendRuntime.EntrypointOverride) > 0 {
		entrypoint = in.BackendRuntime.EntrypointOverride
	}
	if in.ProcessStartConfig != nil {
		switch in.ProcessStartConfig.EntrypointMode {
		case "image_default":
			entrypoint = nil // Docker preserves image ENTRYPOINT
		case "custom":
			if len(in.ProcessStartConfig.Entrypoint) > 0 {
				entrypoint = in.ProcessStartConfig.Entrypoint
			}
		}
		// Unknown modes fall through to whatever was resolved above (legacy).
	}

	// 5. Build final args.
	args, argErrs := buildArgs(in, vars)
	errors = append(errors, argErrs...)

	// Prepend command_prefix to Cmd (Layer 3).
	// command_prefix is added AFTER buildArgs() returns — it does NOT enter
	// Layer 4 dedup or applyServiceArgs.
	if in.ProcessStartConfig != nil && len(in.ProcessStartConfig.CommandPrefix) > 0 {
		args = append(in.ProcessStartConfig.CommandPrefix, args...)
	}

	// 6. Build final env.
	env, envWarns := buildEnv(in, vars)
	warnings = append(warnings, envWarns...)

	// 7. Merge docker spec.
	docker := mergeDockerSpec(in)

	// 8. Build model mounts.
	mounts, mountErr := buildMounts(in)
	if mountErr != nil {
		errors = append(errors, mountErr)
		return nil, errors, warnings
	}

	// 9. Build health check.
	hc := buildHealthCheck(in)

	// 10. Build ports.
	containerPort := effectiveContainerPort(in)
	hostPort := in.Deployment.Service.HostPort

	// 11. GPU visible env.
	gpuVisibleKey := docker.GPUVisibleEnvKey
	if gpuVisibleKey == "" {
		gpuVisibleKey = defaultVisibleEnvKey(in.BackendRuntime.Vendor)
	}
	gpuIDs := make([]string, 0)
	for _, g := range in.AssignedGPUs {
		gpuIDs = append(gpuIDs, fmt.Sprintf("%d", g.Index))
	}
	if len(gpuIDs) > 0 {
		env[gpuVisibleKey] = strings.Join(gpuIDs, ",")
	}

	// 12. Build result.
	plan := &ResolvedRunPlan{
		Image:         image,
		ContainerName: fmt.Sprintf("lightai-%s", in.InstanceID[:minInt(12, len(in.InstanceID))]),
		Entrypoint:    entrypoint,
		Args:          args,
		Env:           env,

		Privileged:  docker.Privileged,
		IPCMode:     docker.IPCMode,
		UTSMode:     docker.UTSMode,
		NetworkMode: docker.NetworkMode,
		ShmSize:     docker.ShmSize,
		Ulimits:     docker.Ulimits,

		Devices:  docker.Devices,
		Mounts:   mounts,
		GroupAdd: docker.GroupAdd,

		HostPort:      hostPort,
		ContainerPort: containerPort,

		GPUDeviceIDs:     gpuIDs,
		GPUVisibleEnvKey: gpuVisibleKey,
		GpuDriver:        docker.GpuDriver,
		GpuCapabilities:  docker.GpuCapabilities,

		SecurityOptions: docker.SecurityOptions,
		ExtraArgs:       []string{},

		HealthCheck: hc,

		BackendName:     in.Backend.Name,
		BackendVersion:  in.BackendVersion.Version,
		ModelName:       in.Artifact.Name,
		ModelPath:       in.Artifact.Path,
		ServedModelName: vars["SERVED_MODEL_NAME"],
		DeploymentID:    in.Deployment.ID,
		InstanceID:      in.InstanceID,
	}

	// Generate docker preview and compute hashes.
	plan.InputHash = computeInputHash(in)
	plan.PlanHash = computePlanHash(plan)

	log.Info("runplan.docker_spec.resolved",
		"image", plan.Image,
		"entrypoint", plan.Entrypoint,
		"args_json", plan.Args,
		"args_count", len(plan.Args),
		"host_port", plan.HostPort,
		"container_port", plan.ContainerPort,
		"mounts_count", len(plan.Mounts),
		"env_keys", mapKeys(plan.Env),
		"health_check_path", plan.HealthCheck.Path,
		"source_backend_version_id", in.BackendVersion.ID,
		"errors", len(errors), "warnings", len(warnings),
		"duration_ms", time.Since(startTime).Milliseconds())
	return plan, errors, warnings
}

// mapKeys returns the keys of a map as a sorted slice for safe logging.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func resolveImage(in ResolveInput) (string, []string) {
	var warnings []string

	// 1. NodeRuntimeOverride.image_name
	if in.NodeRuntimeOverride != nil && in.NodeRuntimeOverride.ImageName != "" {
		return in.NodeRuntimeOverride.ImageName, warnings
	}

	// 2. BackendRuntime.image_name
	if in.BackendRuntime.ImageName != "" {
		return in.BackendRuntime.ImageName, warnings
	}

	// 3. BackendVersion.defaultImages[vendor]
	if img, ok := in.BackendVersion.DefaultImages[in.BackendRuntime.Vendor]; ok && img != "" {
		return img, warnings
	}

	// 4. No image available.
	return "", warnings
}

func buildArgs(in ResolveInput, vars map[string]string) ([]string, []error) {
	var errors []error
	var args []string

	// Layer 1: BackendVersion.default_args_json
	for _, arg := range in.BackendVersion.DefaultArgs {
		resolved, err := substituteVars(arg, vars)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		args = append(args, resolved)
	}

	// Layer 2: BackendVersion.default_backend_params_json
	for _, param := range in.BackendVersion.DefaultBackendParams {
		resolved, err := substituteVars(param, vars)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		args = append(args, resolved)
	}

	// Layer 3: BackendRuntime.args_override_json (append only)
	for _, arg := range in.BackendRuntime.ArgsOverride {
		resolved, err := substituteVars(arg, vars)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		args = append(args, resolved)
	}

	// Layer 4: Deployment.parameters_json mapped to CLI args
	paramArgs := mapParametersToArgs(in.Deployment.Parameters, in.BackendVersion.ParameterDefs)
	args = append(args, paramArgs...)

	// Deduplicate: remove duplicate consecutive flag-value pairs
	args = deduplicateArgs(args)

	// Apply service-level config to args.  Service fields (app_port, host, etc.)
	// always take priority over ParameterDef defaults from any layer.
	args = applyServiceArgs(args, in.Deployment.Service)

	return args, errors
}

// overridePortArg replaces the value of the last --port flag with the given port.
// If no --port flag exists, appends it with the given port.
// applyServiceArgs overrides args with values from the deployment service config.
// Service-level settings (app_port, host) always beat ParameterDef defaults.
func applyServiceArgs(args []string, svc ServiceInfo) []string {
	if svc.AppPort > 0 {
		args = setLastFlagValue(args, "--port", fmt.Sprintf("%d", svc.AppPort))
	}
	// --host from service is the bind address (typically 0.0.0.0).
	// Only override if explicitly set.
	if svc.HostPort > 0 {
		// host_port is the external port; app_host is the listen address.
		// Currently no separate app_host field, so --host remains from args.
	}
	return args
}

// setLastFlagValue replaces the value of the last occurrence of flag, or appends.
func setLastFlagValue(args []string, flag, value string) []string {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == flag && i+1 < len(args) {
			args[i+1] = value
			return args
		}
	}
	return append(args, flag, value)
}

// Deprecated: use applyServiceArgs instead.
func overridePortArg(args []string, port int) []string {
	if port <= 0 {
		return args
	}
	portStr := fmt.Sprintf("%d", port)
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == "--port" && i+1 < len(args) {
			args[i+1] = portStr
			return args
		}
	}
	return append(args, "--port", portStr)
}

// deduplicateArgs removes duplicate --flag value pairs, keeping the LAST occurrence
// (highest priority — user parameters from Layer 4 override defaults from Layer 1).
// Processes in reverse so later values naturally win.
func deduplicateArgs(args []string) []string {
	seen := make(map[string]bool)
	// First pass: identify all flag keys (in order, so later occurrences win).
	type flagPair struct{ idx, valIdx int }
	lastSeen := make(map[string]flagPair)
	i := 0
	for i < len(args) {
		if strings.HasPrefix(args[i], "-") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			lastSeen[args[i]] = flagPair{idx: i, valIdx: i + 1}
			i += 2
		} else {
			i++
		}
	}
	// Second pass: keep only the last occurrence of each flag.
	var result []string
	i = 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			if fp, ok := lastSeen[arg]; ok && fp.idx == i {
				// This is the last occurrence — keep it.
				if !seen[arg] {
					seen[arg] = true
					result = append(result, arg, args[i+1])
				}
			}
			i += 2
		} else {
			result = append(result, arg)
			i++
		}
	}
	return result
}

func mapParametersToArgs(params map[string]interface{}, defs []ParameterDef) []string {
	var args []string
	for _, def := range defs {
		// Look up value by multiple name forms:
		//   1. ParameterDef.Name (CLI format e.g. "--served-model-name")
		//   2. Normalized Name (snake_case e.g. "served_model_name")
		//   3. ParameterDef.CliName / Alias (e.g. "-ngl" for "--n-gpu-layers")
		//   4. Normalized CliName/Alias (e.g. "ngl")
		val, ok := params[def.Name]
		if !ok {
			normalized := strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(def.Name, "-"), "-"), "-", "_")
			if normalized != def.Name {
				val, ok = params[normalized]
			}
		}
		effCli := def.effectiveCliName()
		if !ok && effCli != "" && effCli != def.Name {
			val, ok = params[effCli]
			if !ok {
				normalizedCli := strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(effCli, "-"), "-"), "-", "_")
				if normalizedCli != effCli {
					val, ok = params[normalizedCli]
				}
			}
		}
		if !ok {
			if def.Default != nil {
				val = def.Default
			} else if def.Required {
				// Required parameter missing — skip for now, resolver will report
				continue
			} else {
				continue
			}
		}
		cliName := def.effectiveCliName()
		if cliName == "" {
			cliName = def.Name
		}
		args = append(args, cliName)
		args = append(args, fmt.Sprintf("%v", val))
	}
	return args
}

func buildEnv(in ResolveInput, vars map[string]string) (map[string]string, []string) {
	env := make(map[string]string)
	var warnings []string

	// Layer 1: InferenceBackend.default_env_json
	for k, v := range in.Backend.DefaultEnv {
		resolved, err := substituteVars(v, vars)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("env %s: %v", k, err))
			continue
		}
		env[k] = resolved
	}

	// Layer 2: BackendVersion.env_json
	for k, v := range in.BackendVersion.Env {
		resolved, err := substituteVars(v, vars)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("env %s: %v", k, err))
			continue
		}
		env[k] = resolved
	}

	// Layer 3: BackendRuntime.default_env_json
	for k, v := range in.BackendRuntime.DefaultEnv {
		resolved, err := substituteVars(v, vars)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("env %s: %v", k, err))
			continue
		}
		env[k] = resolved
	}

	// Layer 4: NodeRuntimeOverride.env_json
	if in.NodeRuntimeOverride != nil {
		for k, v := range in.NodeRuntimeOverride.Env {
			resolved, err := substituteVars(v, vars)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("env %s: %v", k, err))
				continue
			}
			env[k] = resolved
		}
	}

	// Layer 5: ModelDeployment.env_overrides_json
	for k, v := range in.Deployment.EnvOverrides {
		env[k] = v
	}

	return env, warnings
}

func mergeDockerSpec(in ResolveInput) DockerSpecInfo {
	docker := in.BackendRuntime.Docker

	// Apply NodeRuntimeOverride.docker_override_json
	if in.NodeRuntimeOverride != nil && in.NodeRuntimeOverride.DockerOverride != nil {
		override := in.NodeRuntimeOverride.DockerOverride
		if override.Privileged {
			docker.Privileged = true
		}
		if override.IPCMode != "" {
			docker.IPCMode = override.IPCMode
		}
		if override.UTSMode != "" {
			docker.UTSMode = override.UTSMode
		}
		if override.NetworkMode != "" {
			docker.NetworkMode = override.NetworkMode
		}
		if override.ShmSize != "" {
			docker.ShmSize = override.ShmSize
		}
		if override.GPUVisibleEnvKey != "" {
			docker.GPUVisibleEnvKey = override.GPUVisibleEnvKey
		}
		if len(override.Devices) > 0 {
			docker.Devices = override.Devices
		}
		if len(override.SecurityOptions) > 0 {
			docker.SecurityOptions = override.SecurityOptions
		}
		for k, v := range override.Ulimits {
			if docker.Ulimits == nil {
				docker.Ulimits = make(map[string]string)
			}
			docker.Ulimits[k] = v
		}
		if len(override.GroupAdd) > 0 {
			docker.GroupAdd = override.GroupAdd
		}
	}

	return docker
}

func buildMounts(in ResolveInput) ([]MountMapping, error) {
	var mounts []MountMapping

	// Model mount: construct per-node host path from model_root + relative_path.
	// Different nodes can have different roots; the container path is standardized
	// and must match MODEL_CONTAINER_PATH used in template variable substitution.
	hostRoot := modelHostRoot(in.Artifact)
	if in.NodeRuntimeOverride != nil && in.NodeRuntimeOverride.ModelRootHostPath != "" {
		hostRoot = in.NodeRuntimeOverride.ModelRootHostPath
	}
	relPath := modelRelativePath(in.Artifact)

	// Validate relative path: must not be empty, must not escape.
	// Check both the trimmed value AND the original RelativePath for absolute prefix.
	if relPath == "" || relPath == "." {
		return nil, fmt.Errorf("model relative_path is empty")
	}
	if strings.Contains(relPath, "..") {
		return nil, fmt.Errorf("model relative_path is invalid: %q (must not contain ..)", relPath)
	}
	if in.Artifact != nil && strings.HasPrefix(in.Artifact.RelativePath, "/") {
		return nil, fmt.Errorf("model relative_path must not be absolute: %q", in.Artifact.RelativePath)
	}

	hostPath := strings.TrimRight(hostRoot, "/") + "/" + strings.TrimLeft(relPath, "/")

	containerMountDir := in.BackendRuntime.ModelMount.ContainerPath
	if containerMountDir == "" {
		containerMountDir = "/models"
	}
	// Container path matches MODEL_CONTAINER_PATH: /models/<relative-path>
	containerPath := strings.TrimRight(containerMountDir, "/") + "/" + strings.TrimLeft(relPath, "/")

	// Safety: cleaned container path must stay under the mount directory.
	cleaned := cleanPath(containerPath)
	if !strings.HasPrefix(cleaned, cleanPath(containerMountDir)+"/") && cleaned != cleanPath(containerMountDir) {
		return nil, fmt.Errorf("container model path escapes mount dir: %q (cleaned: %q)", containerPath, cleaned)
	}

	readonly := in.BackendRuntime.ModelMount.Readonly
	if in.BackendRuntime.ModelMount.ContainerPath == "" {
		readonly = true
	}

	mounts = append(mounts, MountMapping{
		HostPath:      hostPath,
		ContainerPath: cleaned,
		Readonly:      readonly,
	})

	return mounts, nil
}

// cleanPath removes redundant separators and resolves . components without
// following symlinks. Unlike path.Clean, it rejects .. traversal.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	parts := strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
	var out []string
	for _, part := range parts {
		switch part {
		case ".":
			continue
		case "..":
			// Reject — caller should have already validated no ..
			return ""
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return "/"
	}
	return "/" + strings.Join(out, "/")
}

func modelHostRoot(artifact *ArtifactInfo) string {
	if artifact == nil {
		return ""
	}
	if artifact.ModelRoot != "" {
		return artifact.ModelRoot
	}
	modelHostPath := artifact.Path
	if idx := strings.LastIndex(modelHostPath, "/"); idx >= 0 {
		return modelHostPath[:idx]
	}
	return modelHostPath
}

func modelRelativePath(artifact *ArtifactInfo) string {
	if artifact == nil {
		return ""
	}
	if artifact.RelativePath != "" {
		return strings.TrimPrefix(artifact.RelativePath, "/")
	}
	return filepathBase(artifact.Path)
}

func filepathBase(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func buildHealthCheck(in ResolveInput) HealthCheck {
	// BackendRuntime.health_check_override > BackendVersion.health_check
	if in.BackendRuntime.HealthCheckOverride != nil {
		return HealthCheck{
			Path:                  in.BackendRuntime.HealthCheckOverride.Path,
			ExpectedStatus:        in.BackendRuntime.HealthCheckOverride.ExpectedStatus,
			StartupTimeoutSeconds: in.BackendRuntime.HealthCheckOverride.StartupTimeoutSeconds,
			IntervalSeconds:       in.BackendRuntime.HealthCheckOverride.IntervalSeconds,
			TimeoutSeconds:        in.BackendRuntime.HealthCheckOverride.TimeoutSeconds,
		}
	}

	return HealthCheck{
		Path:                  in.BackendVersion.HealthCheck.Path,
		ExpectedStatus:        in.BackendVersion.HealthCheck.ExpectedStatus,
		StartupTimeoutSeconds: in.BackendVersion.HealthCheck.StartupTimeoutSeconds,
		IntervalSeconds:       in.BackendVersion.HealthCheck.IntervalSeconds,
		TimeoutSeconds:        in.BackendVersion.HealthCheck.TimeoutSeconds,
	}
}

func buildVarMap(in ResolveInput) map[string]string {
	vars := make(map[string]string)

	// Model path in container (after mount translation).
	// relative_path is validated in buildMounts before reaching here; extra defense
	// skips path-dependent vars if the path is invalid.
	modelBase := modelRelativePath(in.Artifact)
	containerMount := in.BackendRuntime.ModelMount.ContainerPath
	if containerMount == "" {
		containerMount = "/models"
	}
	modelContainerPath := strings.TrimRight(containerMount, "/") + "/" + strings.TrimLeft(modelBase, "/")
	// Defense: if relative path is empty or escape-like, fall back to a safe default.
	if modelBase == "" || modelBase == "." || strings.Contains(modelBase, "..") || strings.HasPrefix(modelBase, "/") {
		modelContainerPath = containerMount // safe fallback: just the mount dir
	}

	// Compute per-node host model path: model_root + "/" + relative_path.
	// Different nodes can have different host paths for the same model.
	modelHostPath := in.Artifact.Path
	if in.Artifact.ModelRoot != "" && in.Artifact.RelativePath != "" {
		modelHostPath = strings.TrimRight(in.Artifact.ModelRoot, "/") + "/" + strings.TrimLeft(in.Artifact.RelativePath, "/")
	}

	vars["MODEL_CONTAINER_PATH"] = modelContainerPath
	vars["model_container_path"] = vars["MODEL_CONTAINER_PATH"]

	// MODEL_CONTAINER_FILE: for GGUF/file-type models, includes the specific
	// .gguf filename even when the location path is a directory.
	// llama.cpp's -m requires the exact .gguf file path (WEB-AI-RC-001).
	modelContainerFile := modelContainerPath
	if !strings.HasSuffix(modelBase, ".gguf") {
		artifactBase := filepathBase(in.Artifact.Path)
		if strings.HasSuffix(artifactBase, ".gguf") {
			modelContainerFile = strings.TrimRight(containerMount, "/") + "/" + strings.TrimLeft(modelBase, "/") + "/" + artifactBase
		}
	}
	vars["MODEL_CONTAINER_FILE"] = modelContainerFile
	vars["model_container_file"] = modelContainerFile

	vars["MODEL_HOST_PATH"] = modelHostPath
	vars["model_host_path"] = vars["MODEL_HOST_PATH"]
	vars["model_parent_host_path"] = modelHostRoot(in.Artifact)
	vars["MODEL_PARENT_HOST_PATH"] = vars["model_parent_host_path"]

	containerPort := effectiveContainerPort(in)
	appPort := effectiveAppPort(in, containerPort)
	port := fmt.Sprintf("%d", containerPort)
	vars["CONTAINER_PORT"] = port
	vars["container_port"] = port
	vars["APP_PORT"] = fmt.Sprintf("%d", appPort)
	vars["app_port"] = vars["APP_PORT"]

	if in.Deployment.Service.HostPort > 0 {
		vars["HOST_PORT"] = fmt.Sprintf("%d", in.Deployment.Service.HostPort)
		vars["host_port"] = vars["HOST_PORT"]
	}

	vars["SERVED_MODEL_NAME"] = ""
	vars["served_model_name"] = ""
	// Helper: get parameter value from deployment or fall back to definition default
	getParam := func(name string) interface{} {
		if v, ok := in.Deployment.Parameters[name]; ok {
			return v
		}
		// Also try the CLI-format name (e.g. "--served-model-name" for "served_model_name").
		cliName := "--" + strings.ReplaceAll(name, "_", "-")
		for _, d := range in.BackendVersion.ParameterDefs {
			n := d.Name
			// Strip leading -- and convert - to _ for comparison.
			normalized := strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(n, "-"), "-"), "-", "_")
			if n == name || n == cliName || normalized == name {
				if d.Default != nil {
					return d.Default
				}
			}
		}
		return nil
	}

	if v := getParam("served_model_name"); v != nil {
		s := fmt.Sprintf("%v", v)
		if strings.TrimSpace(s) != "" {
			vars["SERVED_MODEL_NAME"] = s
			vars["served_model_name"] = s
		}
	}
	// Derive served model name from artifact name when not explicitly set.
	// Priority: deployment param > param def default > artifact name > sanitized path basename.
	if vars["SERVED_MODEL_NAME"] == "" && in.Artifact.Name != "" {
		sn := strings.TrimSpace(in.Artifact.Name)
		if sn == "" && in.Artifact.Path != "" {
			// Fallback: sanitize path basename.
			base := in.Artifact.Path
			if idx := strings.LastIndex(base, "/"); idx >= 0 {
				base = base[idx+1:]
			}
			sn = strings.TrimSpace(base)
		}
		if sn != "" {
			vars["SERVED_MODEL_NAME"] = sn
			vars["served_model_name"] = sn
		}
	}
	if v := getParam("max_model_len"); v != nil {
		s := fmt.Sprintf("%v", v)
		vars["MAX_MODEL_LEN"] = s
		vars["max_model_len"] = s
	}
	if v := getParam("gpu_memory_utilization"); v != nil {
		s := fmt.Sprintf("%v", v)
		vars["GPU_MEMORY_UTILIZATION"] = s
		vars["gpu_memory_utilization"] = s
	}
	if v := getParam("tensor_parallel_size"); v != nil {
		s := fmt.Sprintf("%v", v)
		vars["TENSOR_PARALLEL_SIZE"] = s
		vars["tensor_parallel_size"] = s
	}

	gpuIDs := make([]string, 0)
	for _, g := range in.AssignedGPUs {
		gpuIDs = append(gpuIDs, fmt.Sprintf("%d", g.Index))
	}
	vars["ASSIGNED_GPU_INDEXES"] = strings.Join(gpuIDs, ",")
	vars["assigned_gpu_indexes"] = vars["ASSIGNED_GPU_INDEXES"]
	vars["VENDOR_VISIBLE_DEVICES"] = vars["ASSIGNED_GPU_INDEXES"]
	vars["vendor_visible_devices"] = vars["ASSIGNED_GPU_INDEXES"]
	vars["ASSIGNED_GPU_COUNT"] = fmt.Sprintf("%d", len(gpuIDs))
	vars["assigned_gpu_count"] = vars["ASSIGNED_GPU_COUNT"]

	vars["DEPLOYMENT_NAME"] = in.Deployment.Name
	vars["deployment_name"] = vars["DEPLOYMENT_NAME"]
	vars["INSTANCE_ID"] = in.InstanceID
	vars["instance_id"] = vars["INSTANCE_ID"]
	vars["NODE_ID"] = in.Node.ID
	vars["node_id"] = vars["NODE_ID"]
	vars["NODE_IP"] = in.Node.IP
	vars["node_ip"] = vars["NODE_IP"]

	return vars
}

func defaultVisibleEnvKey(vendor string) string {
	switch strings.ToLower(vendor) {
	case "huawei", "ascend":
		return "ASCEND_VISIBLE_DEVICES"
	case "metax":
		return "CUDA_VISIBLE_DEVICES"
	default:
		return "CUDA_VISIBLE_DEVICES"
	}
}

func computeInputHash(in ResolveInput) string {
	data, _ := json.Marshal(map[string]interface{}{
		"backend":         in.Backend.Name,
		"version":         in.BackendVersion.Version,
		"runtime":         in.BackendRuntime.ID,
		"artifact":        in.Artifact.Path,
		"deployment":      in.Deployment.ID,
		"host_port":       in.Deployment.Service.HostPort,
		"container_port":  in.Deployment.Service.ContainerPort,
		"app_port":        in.Deployment.Service.AppPort,
		"parameters":      in.Deployment.Parameters,
		"env_overrides":   in.Deployment.EnvOverrides,
		"accelerator_ids": in.Deployment.Placement.AcceleratorIds,
		"node_id":         in.Deployment.Placement.NodeID,
	})
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h[:8])
}

func effectiveContainerPort(in ResolveInput) int {
	if in.Deployment != nil && in.Deployment.Service.ContainerPort > 0 {
		return in.Deployment.Service.ContainerPort
	}
	if in.Deployment != nil && in.Deployment.Service.AppPort > 0 {
		return in.Deployment.Service.AppPort
	}
	if in.BackendVersion != nil && in.BackendVersion.DefaultContainerPort > 0 {
		return in.BackendVersion.DefaultContainerPort
	}
	return 8000
}

func effectiveAppPort(in ResolveInput, containerPort int) int {
	if in.Deployment != nil && in.Deployment.Service.AppPort > 0 {
		return in.Deployment.Service.AppPort
	}
	return containerPort
}

func computePlanHash(plan *ResolvedRunPlan) string {
	data, _ := json.Marshal(plan)
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h[:8])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildDeviceBinding creates a vendor-specific DeviceBinding from resolver inputs.
func buildDeviceBinding(in ResolveInput, gpuIDs []string, gpuVisibleKey string, docker DockerSpecInfo) *DeviceBinding {
	binding := &DeviceBinding{
		Vendor:           in.BackendRuntime.Vendor,
		GPUVisibleEnvKey: gpuVisibleKey,
		Privileged:       docker.Privileged,
		IPCMode:          docker.IPCMode,
		ShmSize:          docker.ShmSize,
		Devices:          docker.Devices,
		GroupAdd:         docker.GroupAdd,
		SecurityOpt:      docker.SecurityOptions,
		Ulimits:          docker.Ulimits,
	}

	// Collect platform accelerator indices from assigned GPUs
	for _, g := range in.AssignedGPUs {
		binding.AcceleratorIds = append(binding.AcceleratorIds, fmt.Sprintf("%d", g.Index))
	}

	switch strings.ToLower(in.BackendRuntime.Vendor) {
	case "nvidia":
		binding.Mode = "nvidia_device_request"
		binding.VisibleDeviceIDs = gpuIDs
		binding.GPUDriver = docker.GpuDriver
		binding.GPUCapabilities = docker.GpuCapabilities
	case "metax":
		// Privileged with device paths = native device_paths mode
		binding.Mode = "metax_device_paths"
		for _, d := range docker.Devices {
			binding.DevicePaths = append(binding.DevicePaths, d.HostPath)
		}
	case "cpu":
		binding.Mode = "cpu_none"
	case "huawei", "ascend":
		// template_only: no real device binding until hardware validation + smoke evidence.
		// Do not fabricate device bindings or suggest the runtime can actually start.
		binding.Mode = "template_only"
	default:
		binding.Mode = "nvidia_device_request"
		binding.VisibleDeviceIDs = gpuIDs
		binding.GPUDriver = docker.GpuDriver
		binding.GPUCapabilities = docker.GpuCapabilities
	}

	return binding
}
