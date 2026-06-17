package models

// BackendRuntime is a user-editable runtime configuration.
// It represents a specific BackendVersion + vendor + Docker runtime configuration.
type BackendRuntime struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	DisplayName             string `json:"display_name"`
	BackendID               string `json:"backend_id"`
	BackendVersionID        string `json:"backend_version_id"`
	SourceTemplateName      string `json:"source_template_name"`
	Vendor                  string `json:"vendor"`
	RuntimeType             string `json:"runtime_type"`
	ImageName               string `json:"image_name"`
	ImagePullPolicy         string `json:"image_pull_policy"`
	EntrypointOverrideJSON  string `json:"entrypoint_override_json"`
	ArgsOverrideJSON        string `json:"args_override_json"`
	DefaultEnvJSON          string `json:"default_env_json"`
	DockerJSON              string `json:"docker_json"`
	ModelMountJSON          string `json:"model_mount_json"`
	HealthCheckOverrideJSON string `json:"health_check_override_json"`
	IsBuiltin               bool   `json:"is_builtin"`
	IsEditable              bool   `json:"is_editable"`
	TenantID                string `json:"tenant_id"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
}

// NodeRuntimeOverride represents per-node overrides for a BackendRuntime.
// Used when different servers need different images, model roots, devices, or env.
type NodeRuntimeOverride struct {
	ID                 string `json:"id"`
	NodeID             string `json:"node_id"`
	BackendRuntimeID   string `json:"backend_runtime_id"`
	TenantID           string `json:"tenant_id"`
	ImageName          string `json:"image_name"`
	ImagePullPolicy    string `json:"image_pull_policy"`
	EnvJSON            string `json:"env_json"`
	DockerOverrideJSON string `json:"docker_override_json"`
	ModelRootHostPath  string `json:"model_root_host_path"`
	IsEnabled          bool   `json:"is_enabled"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}
