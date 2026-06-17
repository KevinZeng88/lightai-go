package models

// InferenceBackend represents an inference backend family (vllm, sglang, llamacpp).
// It does NOT contain vendor, image, or Docker config — those belong to BackendRuntime.
type InferenceBackend struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	DisplayName          string `json:"display_name"`
	Description          string `json:"description"`
	ProtocolJSON         string `json:"protocol_json"`
	DefaultVersion       string `json:"default_version"`
	ParameterFormat      string `json:"parameter_format"`
	CommonParametersJSON string `json:"common_parameters_json"`
	DefaultEnvJSON       string `json:"default_env_json"`
	IsBuiltin            bool   `json:"is_builtin"`
	IsEnabled            bool   `json:"is_enabled"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

// BackendVersion represents a specific version of an inference backend.
// It contains version-level defaults: entrypoint, args template, parameters, health check,
// and recommended images (per vendor). It does NOT contain actual runtime Docker config.
type BackendVersion struct {
	ID                       string `json:"id"`
	BackendID                string `json:"backend_id"`
	Version                  string `json:"version"`
	DisplayName              string `json:"display_name"`
	IsDefault                bool   `json:"is_default"`
	DefaultEntrypointJSON    string `json:"default_entrypoint_json"`
	DefaultArgsJSON          string `json:"default_args_json"`
	DefaultBackendParamsJSON string `json:"default_backend_params_json"`
	ParameterDefsJSON        string `json:"parameter_defs_json"`
	HealthCheckJSON          string `json:"health_check_json"`
	DefaultContainerPort     int    `json:"default_container_port"`
	DefaultImagesJSON        string `json:"default_images_json"`
	EnvJSON                  string `json:"env_json"`
	IsDeprecated             bool   `json:"is_deprecated"`
	CreatedAt                string `json:"created_at"`
	UpdatedAt                string `json:"updated_at"`
}

// BackendRuntimeTemplate is a system readonly template for creating BackendRuntimes.
// It is NOT stored in the database — it comes from config files only.
type BackendRuntimeTemplate struct {
	Name                    string `json:"name" yaml:"name"`
	DisplayName             string `json:"display_name" yaml:"display_name"`
	BackendName             string `json:"backend_name" yaml:"backend_name"`
	BackendVersion          string `json:"backend_version" yaml:"backend_version"`
	Vendor                  string `json:"vendor" yaml:"vendor"`
	RuntimeType             string `json:"runtime_type" yaml:"runtime_type"`
	ImageName               string `json:"image_name" yaml:"image_name"`
	ImagePullPolicy         string `json:"image_pull_policy" yaml:"image_pull_policy"`
	EntrypointJSON          string `json:"entrypoint_json" yaml:"entrypoint"`
	ArgsOverrideJSON        string `json:"args_override_json" yaml:"args_override"`
	DefaultEnvJSON          string `json:"default_env_json" yaml:"default_env"`
	DockerJSON              string `json:"docker_json" yaml:"docker"`
	ModelMountJSON          string `json:"model_mount_json" yaml:"model_mount"`
	HealthCheckOverrideJSON string `json:"health_check_override_json" yaml:"health_check_override"`
}
