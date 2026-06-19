package models

// ModelDeployment represents a user's deployment specification.
// It binds a ModelArtifact to a BackendRuntime, with placement and parameter config.
type ModelDeployment struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	Description      string `json:"description"`
	ModelArtifactID  string `json:"model_artifact_id"`
	BackendRuntimeID string `json:"backend_runtime_id"`
	Replicas         int    `json:"replicas"`
	PlacementJSON    string `json:"placement_json"`
	ServiceJSON      string `json:"service_json"`
	ParametersJSON   string `json:"parameters_json"`
	EnvOverridesJSON     string `json:"env_overrides_json"`
	ConfigSnapshotJSON          string `json:"config_snapshot_json"`
	SourceBackendRuntimeID      string `json:"source_backend_runtime_id"`
	SourceNodeBackendRuntimeID  string `json:"source_node_backend_runtime_id"`
	SourceTemplateName          string `json:"source_template_name"`
	SourceTemplateVersion       string `json:"source_template_version"`
	SourceConfigHash            string `json:"source_config_hash"`
	CopiedAt                    string `json:"copied_at"`
	DesiredState                string `json:"desired_state"`
	Status           string `json:"status"`
	TenantID         string `json:"tenant_id"`
	OwnerID          string `json:"owner_id"`
	CreatedBy        string `json:"created_by"`
	UpdatedBy        string `json:"updated_by"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}
