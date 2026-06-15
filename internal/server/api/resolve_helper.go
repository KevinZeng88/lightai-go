package api

import (
	"encoding/json"
	"fmt"

	"lightai-go/internal/server/db"
	"lightai-go/internal/common/log"
	"lightai-go/internal/server/resolver"
)

// resolveDeploymentInput holds all data extracted from a deployment and its
// associated artifact, runtime environment, and run template.  It is used
// by buildResolveInputForDeployment to construct a resolver.ResolveInput.
type resolveDeploymentInput struct {
	ArtifactID  string
	EnvID       string
	TemplateID  string
	DeployID    string
	NodeID      string
	GPUIds      []string // LightAI internal UUIDs
	HostPort    int
	ModelPath   string
	Vendor      string
	RuntimeType string
	BackendType string
	DefaultPort int

	ServedModelName      string
	MaxModelLen          int
	GPUMemoryUtilization float64
}

// buildResolveInputForDeployment fetches all cross-reference data from the
// database and constructs a resolver.ResolveInput with consistent image,
// volume mappings, env mappings, GPU device indices, and variable data.
//
// This is the single source of truth used by dry-run, start, and
// render-preview endpoints.
func buildResolveInputForDeployment(database *db.DB, in resolveDeploymentInput) resolver.ResolveInput {
	// Resolve GPU indices for DeviceRequests (NVIDIA uses 0-based index).
	var gpuDeviceIDs []string
	for _, gpuID := range in.GPUIds {
		var idx int
		var gpuVendor string
		err := database.QueryRow(`SELECT index_num, vendor FROM gpu_devices WHERE id = ?`, gpuID).Scan(&idx, &gpuVendor)
		if err == nil {
			gpuDeviceIDs = append(gpuDeviceIDs, fmt.Sprintf("%d", idx))
		} else {
			log.Warn("GPU index lookup failed", "gpu_id", gpuID, "error", err)
		}
	}
	if len(gpuDeviceIDs) == 0 && len(in.GPUIds) > 0 {
		log.Warn("GPU index resolution empty, using raw IDs as fallback", "input_ids", in.GPUIds)
		gpuDeviceIDs = in.GPUIds
	}

	// Fetch artifact name.
	var artifactName string
	if in.ArtifactID != "" {
		database.QueryRow(`SELECT name FROM model_artifacts WHERE id = ?`, in.ArtifactID).Scan(&artifactName)
	}

	// Fetch docker image.
	var dockerImage string
	if in.EnvID != "" {
		database.QueryRow(`SELECT image FROM runtime_environment_docker_specs WHERE runtime_environment_id = ?`, in.EnvID).Scan(&dockerImage)
	}

	// Fetch volume and env mappings from run template.
	var volMappingsJSON, envMappingsJSON string
	if in.TemplateID != "" {
		database.QueryRow(`SELECT volume_mappings, env_mappings FROM run_templates WHERE id = ?`, in.TemplateID).Scan(&volMappingsJSON, &envMappingsJSON)
	}

	volMappings := parseVolumeMappings(volMappingsJSON)
	envMappings := parseEnvMappings(envMappingsJSON)

	// Fetch env overrides from deployment.
	var envOverridesJSON string
	if in.DeployID != "" {
		database.QueryRow(`SELECT env_overrides FROM model_deployments WHERE id = ?`, in.DeployID).Scan(&envOverridesJSON)
	}
	var envOverrides map[string]string
	json.Unmarshal([]byte(envOverridesJSON), &envOverrides)

	// Fetch args_template and required_variables from template.
	var argsTemplate []string
	var reqVarsStr string
	if in.TemplateID != "" {
		var atStr string
		database.QueryRow(`SELECT args_template, required_variables FROM run_templates WHERE id = ?`, in.TemplateID).Scan(&atStr, &reqVarsStr)
		json.Unmarshal([]byte(atStr), &argsTemplate)
	}
	var requiredVars []string
	json.Unmarshal([]byte(reqVarsStr), &requiredVars)

	// Fetch deployment name.
	var depName string
	if in.DeployID != "" {
		database.QueryRow(`SELECT name FROM model_deployments WHERE id = ?`, in.DeployID).Scan(&depName)
	}

	return resolver.ResolveInput{
		Artifact:       &resolver.ModelArtifactInput{Path: in.ModelPath},
		Env:            &resolver.EnvironmentInput{RuntimeType: in.RuntimeType, BackendType: in.BackendType, Vendor: in.Vendor, DefaultPort: in.DefaultPort},
		Deployment:     &resolver.DeploymentInput{ServedModelName: in.ServedModelName, NodeID: in.NodeID, GPUIds: gpuDeviceIDs, HostPort: in.HostPort, MaxModelLen: in.MaxModelLen, GPUMemoryUtilization: in.GPUMemoryUtilization},
		ArgsTemplate:   argsTemplate,
		RequiredVars:   requiredVars,
		Image:          dockerImage,
		VolumeMappings: volMappings,
		EnvMappings:    envMappings,
		EnvOverrides:   envOverrides,
		ArtifactName:   artifactName,
		DeploymentName: depName,
	}
}

func parseVolumeMappings(jsonStr string) []resolver.VolumeMapping {
	if jsonStr == "" || jsonStr == "{}" {
		return nil
	}
	var raw struct {
		Enabled bool                     `json:"enabled"`
		Value   []resolver.VolumeMapping `json:"value"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err == nil && raw.Enabled {
		return raw.Value
	}
	var fallback []resolver.VolumeMapping
	json.Unmarshal([]byte(jsonStr), &fallback)
	return fallback
}

func parseEnvMappings(jsonStr string) []resolver.EnvMapping {
	if jsonStr == "" || jsonStr == "{}" {
		return nil
	}
	var raw struct {
		Enabled bool                  `json:"enabled"`
		Value   []resolver.EnvMapping `json:"value"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err == nil && raw.Enabled {
		return raw.Value
	}
	var fallback []resolver.EnvMapping
	json.Unmarshal([]byte(jsonStr), &fallback)
	return fallback
}
