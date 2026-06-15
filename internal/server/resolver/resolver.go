// Package resolver generates ResolvedRunSpec and validates deployments.
package resolver

import (
	"database/sql"
	"fmt"
	"strings"
)

// ResolvedRunSpec is the frozen run specification sent to the Agent.
type ResolvedRunSpec struct {
	InstanceID       string            `json:"instance_id"`
	DeploymentID     string            `json:"deployment_id"`
	RuntimeType      string            `json:"runtime_type"`
	BackendType      string            `json:"backend_type"`
	Vendor           string            `json:"vendor"`
	ModelPath        string            `json:"model_path"`
	ServedModelName  string            `json:"served_model_name"`
	NodeID           string            `json:"node_id"`
	AgentID          string            `json:"agent_id"`
	GPUDeviceIDs     []string          `json:"gpu_device_ids"`
	GPUVisibleEnvKey string            `json:"gpu_visible_env_key,omitempty"`
	Env              map[string]string `json:"env"`
	Args             []string          `json:"args"`
	HostPort         int               `json:"host_port"`
	ContainerPort    int               `json:"container_port"`
	Volumes          []DockerVolume    `json:"volumes,omitempty"`
	Devices          []DockerDevice    `json:"devices,omitempty"`
	Ports            []DockerPort      `json:"ports,omitempty"`
	Docker           DockerSpec        `json:"docker,omitempty"`
}

// DockerSpec mirrors DockerRunSpec from contracts but simplified for Phase 1.
type DockerSpec struct {
	Image           string            `json:"image"`
	ContainerName   string            `json:"container_name"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args"`
	Privileged      bool              `json:"privileged,omitempty"`
	IPCMode         string            `json:"ipc_mode,omitempty"`
	UTSMode         string            `json:"uts_mode,omitempty"`
	NetworkMode     string            `json:"network_mode,omitempty"`
	ShmSize         string            `json:"shm_size,omitempty"`
	GroupAdd        []string          `json:"group_add,omitempty"`
	SecurityOptions []string          `json:"security_options,omitempty"`
	Ulimits         map[string]string `json:"ulimits,omitempty"`
	RestartPolicy   string            `json:"restart_policy,omitempty"`
	GPUDeviceIDs    []string          `json:"gpu_device_ids,omitempty"`
}

// DockerVolume is a volume mount.
type DockerVolume struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Readonly      bool   `json:"readonly,omitempty"`
}

// DockerDevice is a device mapping.
type DockerDevice struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Permissions   string `json:"permissions,omitempty"`
}

// DockerPort is a port mapping.
type DockerPort struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
}

// ModelArtifactInput is a minimal model input for resolution.
type ModelArtifactInput struct {
	Path string
}

// EnvironmentInput is a minimal runtime environment input for resolution.
type EnvironmentInput struct {
	RuntimeType string
	BackendType string
	Vendor      string
	DefaultPort int
}

// DeploymentInput is a minimal deployment input for resolution.
type DeploymentInput struct {
	ServedModelName      string
	NodeID               string
	GPUIds               []string
	HostPort             int
	MaxModelLen          int
	GPUMemoryUtilization float64
	GPUVisibleEnvKey     string
}

// ResolveInput holds all inputs needed for resolution.
type ResolveInput struct {
	InstanceID     string
	DeploymentID   string
	AgentID        string
	Artifact       *ModelArtifactInput
	Env            *EnvironmentInput
	Deployment     *DeploymentInput
	ArgsTemplate   []string
	RequiredVars   []string
	Image          string
	VolumeMappings []VolumeMapping
	EnvMappings    []EnvMapping
	Devices        []DockerDevice
	EnvOverrides   map[string]string
	ArtifactName   string
	DeploymentName string
}

// VolumeMapping is a volume mount template (may contain ${VAR} placeholders).
type VolumeMapping struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Readonly      bool   `json:"readonly,omitempty"`
}

// EnvMapping is an env variable template.
type EnvMapping struct {
	Key       string `json:"key"`
	ValueFrom string `json:"value_from"`
}

// Resolve generates a ResolvedRunSpec from the given inputs.
func Resolve(in ResolveInput) (*ResolvedRunSpec, []error, []string) {
	var errors []error
	var warnings []string

	spec := &ResolvedRunSpec{
		InstanceID:      in.InstanceID,
		DeploymentID:    in.DeploymentID,
		RuntimeType:     in.Env.RuntimeType,
		BackendType:     in.Env.BackendType,
		Vendor:          in.Env.Vendor,
		ModelPath:       in.Artifact.Path,
		ServedModelName: in.Deployment.ServedModelName,
		NodeID:          in.Deployment.NodeID,
		AgentID:         in.AgentID,
		GPUDeviceIDs:    in.Deployment.GPUIds,
		HostPort:        in.Deployment.HostPort,
		ContainerPort:   in.Env.DefaultPort,
		Env:             make(map[string]string),
		Args:            make([]string, 0),
	}

	// Variable substitution.
	vars := buildVarMap(in)
	missing := checkRequiredVars(in.RequiredVars, vars)

	// Image: use deployment override or docker spec image.
	spec.Docker = DockerSpec{
		Image:         in.Image,
		ContainerName: fmt.Sprintf("lightai-%s", in.InstanceID[:min(12, len(in.InstanceID))]),
	}
	if spec.Docker.Image == "" {
		spec.Docker.Image = "lightai-runtime"
	}

	// Ports — skip if network_mode=host (port mapping has no effect).
	if in.Deployment.HostPort != 0 {
		if spec.Docker.NetworkMode == "host" {
			warnings = append(warnings, "port mapping skipped: network_mode=host makes port mapping ineffective")
		} else {
			spec.Ports = append(spec.Ports, DockerPort{
				HostPort: in.Deployment.HostPort, ContainerPort: in.Env.DefaultPort, Protocol: "tcp",
			})
		}
	}

	// Volumes from template mappings, with variable substitution.
	for _, vm := range in.VolumeMappings {
		spec.Volumes = append(spec.Volumes, DockerVolume{
			HostPath:      substituteVars(vm.HostPath, vars),
			ContainerPath: substituteVars(vm.ContainerPath, vars),
			Readonly:      vm.Readonly,
		})
	}

	// Env from template mappings.
	for _, em := range in.EnvMappings {
		val := substituteVars(em.ValueFrom, vars)
		spec.Env[em.Key] = val
	}

	// Env overrides from deployment.
	for k, v := range in.EnvOverrides {
		spec.Env[k] = v
	}

	// Devices from docker spec.
	spec.Devices = in.Devices
	if len(missing) > 0 {
		errors = append(errors, fmt.Errorf("missing required variables: %s", strings.Join(missing, ", ")))
	}

	// Args from template.
	for _, arg := range in.ArgsTemplate {
		spec.Args = append(spec.Args, substituteVars(arg, vars))
	}

	// GPU visible env.
	if spec.GPUVisibleEnvKey == "" {
		spec.GPUVisibleEnvKey = "CUDA_VISIBLE_DEVICES"
	}
	if in.Deployment.GPUVisibleEnvKey != "" {
		spec.GPUVisibleEnvKey = in.Deployment.GPUVisibleEnvKey
	}
	if len(spec.GPUDeviceIDs) > 0 {
		spec.Env[spec.GPUVisibleEnvKey] = strings.Join(spec.GPUDeviceIDs, ",")
	}

	spec.Docker.Args = spec.Args
	spec.Docker.GPUDeviceIDs = spec.GPUDeviceIDs

	return spec, errors, warnings
}

// ==========================================================================
// Dry Run Validator — extracted from handler for testability
// ==========================================================================

// DBQuerier is a minimal DB interface for the validator.
type DBQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// DryRunInput holds all data needed for dry-run validation.
type DryRunInput struct {
	NodeID              string
	GPUIds              []string
	HostPort            int
	RuntimeVendor       string
	ModelArtifactID     string
	ModelPath           string
	TemplateRequiredVars []string
}

// DryRunResult holds validation output.
type DryRunResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// ValidateDryRun performs all Phase 1 dry-run checks against the database.
func ValidateDryRun(db DBQuerier, in DryRunInput) DryRunResult {
	result := DryRunResult{Errors: []string{}, Warnings: []string{}}

	// 1. Node exists and is online.
	if in.NodeID == "" {
		result.Errors = append(result.Errors, "node_id is required")
	} else {
		var nodeStatus string
		if err := db.QueryRow(`SELECT status FROM nodes WHERE id = ?`, in.NodeID).Scan(&nodeStatus); err == sql.ErrNoRows {
			result.Errors = append(result.Errors, "specified node does not exist")
		} else if nodeStatus != "online" {
			result.Errors = append(result.Errors, "specified node is not online")
		}
	}

	// 2. Model path non-empty.
	if in.ModelArtifactID != "" && in.ModelPath == "" {
		result.Errors = append(result.Errors, "model path is empty")
	}

	// 3. Validate GPUs.
	for _, gpuID := range in.GPUIds {
		var gpuHealth, gpuStatus, gpuVendor string
		if err := db.QueryRow(`SELECT health, status, vendor FROM gpu_devices WHERE id = ?`, gpuID).Scan(&gpuHealth, &gpuStatus, &gpuVendor); err == sql.ErrNoRows {
			result.Errors = append(result.Errors, fmt.Sprintf("GPU %s does not exist", gpuID))
			continue
		}
		if gpuHealth != "healthy" {
			result.Errors = append(result.Errors, fmt.Sprintf("GPU %s health=%s (required: healthy)", gpuID, gpuHealth))
		}
		if gpuStatus == "unavailable" {
			result.Errors = append(result.Errors, fmt.Sprintf("GPU %s is unavailable", gpuID))
		}
		// Lease conflict check.
		var leaseID string
		if err := db.QueryRow(`SELECT id FROM gpu_leases WHERE gpu_id = ? AND status IN ('reserved','active')`, gpuID).Scan(&leaseID); err == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("GPU %s is already reserved/active (lease %s)", gpuID, leaseID))
		}
		// Vendor matching.
		if in.RuntimeVendor != "" && in.RuntimeVendor != "custom" && gpuVendor != in.RuntimeVendor {
			result.Errors = append(result.Errors, fmt.Sprintf("GPU %s vendor=%s does not match runtime vendor=%s", gpuID, gpuVendor, in.RuntimeVendor))
		}
		if in.RuntimeVendor == "custom" {
			result.Warnings = append(result.Warnings, "Runtime vendor=custom, GPU vendor strict matching skipped. Please verify compatibility.")
		}
	}

	// 4. Host port conflict.
	if in.HostPort > 0 {
		var existingPort int
		if err := db.QueryRow(`SELECT host_port FROM model_instances WHERE host_port = ? AND actual_state IN ('pending','starting','loading','running') LIMIT 1`, in.HostPort).Scan(&existingPort); err == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("host_port %d is already in use by another model instance", in.HostPort))
		}
	}

	// 5. VRAM warning.
	for _, gpuID := range in.GPUIds {
		var freeMem int64
		if err := db.QueryRow(`SELECT memory_free_bytes FROM gpu_devices WHERE id = ?`, gpuID).Scan(&freeMem); err == nil {
			// estimatedVRAMBytes is checked at call site.
		}
		_ = freeMem
	}

	result.Valid = len(result.Errors) == 0
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func buildVarMap(in ResolveInput) map[string]string {
	// Host path extraction.
	hostPath := in.Artifact.Path
	hostDir := hostPath
	modelFile := hostPath
	if idx := strings.LastIndex(hostPath, "/"); idx >= 0 {
		hostDir = hostPath[:idx]
		modelFile = hostPath[idx+1:]
	}

	// Compute container model path by applying volume mappings.
	// If a volume maps a host directory prefix to a container directory,
	// translate the model path accordingly.
	containerModelPath := hostPath
	containerModelDir := hostDir
	for _, vm := range in.VolumeMappings {
		if strings.HasPrefix(hostPath, vm.HostPath) {
			rel := strings.TrimPrefix(hostPath, vm.HostPath)
			rel = strings.TrimPrefix(rel, "/")
			containerModelPath = vm.ContainerPath + "/" + rel
			// Also compute container dir
			if strings.HasPrefix(hostDir, vm.HostPath) {
				relDir := strings.TrimPrefix(hostDir, vm.HostPath)
				relDir = strings.TrimPrefix(relDir, "/")
				containerModelDir = vm.ContainerPath
				if relDir != "" {
					containerModelDir = vm.ContainerPath + "/" + relDir
				}
			}
			break
		}
	}

	vars := make(map[string]string)
	vars["INSTANCE_ID"] = in.InstanceID
	vars["DEPLOYMENT_ID"] = in.DeploymentID
	// MODEL_PATH = container path (for args inside container).
	vars["MODEL_PATH"] = containerModelPath
	vars["HOST_MODEL_PATH"] = hostPath
	vars["MODEL_FILE"] = modelFile
	vars["MODEL_DIR"] = containerModelDir
	vars["MODEL_PATH_DIR"] = containerModelDir
	vars["HOST_MODEL_DIR"] = hostDir
	vars["ARTIFACT_PATH"] = hostPath
	vars["ARTIFACT_NAME"] = in.ArtifactName
	vars["DEPLOYMENT_NAME"] = in.DeploymentName
	vars["SERVED_MODEL_NAME"] = in.Deployment.ServedModelName
	vars["HOST_PORT"] = fmt.Sprintf("%d", in.Deployment.HostPort)
	vars["CONTAINER_PORT"] = fmt.Sprintf("%d", in.Env.DefaultPort)
	vars["GPU_IDS"] = strings.Join(in.Deployment.GPUIds, ",")
	vars["NODE_ID"] = in.Deployment.NodeID
	if in.Deployment.MaxModelLen > 0 {
		vars["MAX_MODEL_LEN"] = fmt.Sprintf("%d", in.Deployment.MaxModelLen)
	}
	vars["GPU_MEMORY_UTILIZATION"] = fmt.Sprintf("%.2f", in.Deployment.GPUMemoryUtilization)
	return vars
}

func checkRequiredVars(required []string, vars map[string]string) []string {
	var missing []string
	for _, r := range required {
		if _, ok := vars[r]; !ok {
			missing = append(missing, r)
		}
	}
	return missing
}

func substituteVars(template string, vars map[string]string) string {
	result := template
	for k, v := range vars {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
	}
	return result
}

// EquivalentCommandPreview generates a human-readable docker run command.
// This is for display and debugging only — Agent MUST use ResolvedRunSpec, not this string.
func EquivalentCommandPreview(spec *ResolvedRunSpec) string {
	var parts []string
	parts = append(parts, "docker", "run", "-d")

	if spec.Docker.ContainerName != "" {
		parts = append(parts, "--name", spec.Docker.ContainerName)
	}
	if spec.Docker.Privileged {
		parts = append(parts, "--privileged")
	}
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
	for _, g := range spec.Docker.GroupAdd {
		parts = append(parts, "--group-add", g)
	}
	for _, s := range spec.Docker.SecurityOptions {
		parts = append(parts, "--security-opt", s)
	}
	for k, v := range spec.Docker.Ulimits {
		parts = append(parts, "--ulimit", fmt.Sprintf("%s=%s", k, v))
	}
	// GPU device requests (NVIDIA --gpus).
	if len(spec.Docker.GPUDeviceIDs) > 0 {
		parts = append(parts, "--gpus", fmt.Sprintf("\"device=%s\"", strings.Join(spec.Docker.GPUDeviceIDs, ",")))
	}
	for _, d := range spec.Devices {
		parts = append(parts, "--device", d.HostPath+":"+d.ContainerPath)
	}
	for _, v := range spec.Volumes {
		ro := ""
		if v.Readonly {
			ro = ":ro"
		}
		parts = append(parts, "-v", v.HostPath+":"+v.ContainerPath+ro)
	}
	for _, p := range spec.Ports {
		parts = append(parts, "-p", fmt.Sprintf("%d:%d/%s", p.HostPort, p.ContainerPort, p.Protocol))
	}
	for k, v := range spec.Env {
		parts = append(parts, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	parts = append(parts, spec.Docker.Image)
	parts = append(parts, spec.Docker.Args...)

	return strings.Join(parts, " ")
}
