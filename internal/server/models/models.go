// Package models defines all database models for LightAI Go.
package models

import "time"

// Tenant represents a tenant boundary for resources and members.
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // active / disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents a global local user account.
type User struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	DisplayName        string    `json:"display_name"`
	PasswordHash       string    `json:"-"`
	Status             string    `json:"status"` // active / disabled
	IsPlatformAdmin    bool      `json:"is_platform_admin"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TenantMembership represents a user's membership in a tenant.
type TenantMembership struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"` // active / disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Role represents a built-in or custom role.
type Role struct {
	ID          string    `json:"id"`
	TenantID    *string   `json:"tenant_id,omitempty"` // nil for built-in
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	BuiltIn     bool      `json:"built_in"`
	Status      string    `json:"status"` // active / disabled
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a system-readonly permission code.
type Permission struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Scope       string    `json:"scope"` // tenant / platform
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RolePermission binds a role to a permission.
type RolePermission struct {
	ID           string    `json:"id"`
	RoleID       string    `json:"role_id"`
	PermissionID string    `json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// TenantMembershipRole binds a membership to a role.
type TenantMembershipRole struct {
	ID           string    `json:"id"`
	MembershipID string    `json:"membership_id"`
	RoleID       string    `json:"role_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents a server-side user session.
type Session struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	CurrentTenantID string     `json:"current_tenant_id"`
	CSRFSecretHash  string     `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked returns true if the session has been revoked.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsValid returns true if the session is neither expired nor revoked.
func (s *Session) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked()
}

// --- Phase 1 Model Runtime Serving Objects ---

// ModelArtifact represents a registered AI model.
type ModelArtifact struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	DisplayName          string `json:"display_name"`
	SourceType           string `json:"source_type"`
	Path                 string `json:"path"`
	Format               string `json:"format"`
	TaskType             string `json:"task_type"`
	Architecture         string `json:"architecture"`
	SizeLabel            string `json:"size_label"`
	Quantization         string `json:"quantization"`
	DefaultContextLength int    `json:"default_context_length"`
	EstimatedVRAMBytes   int64  `json:"estimated_vram_bytes"`
	RequiredGPUCount     int    `json:"required_gpu_count"`
	TenantID             string `json:"tenant_id"`
	OwnerID              string `json:"owner_id,omitempty"`
	CreatedBy            string `json:"created_by"`
	UpdatedBy            string `json:"updated_by"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

// RuntimeEnvironment represents a runtime environment definition.
type RuntimeEnvironment struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	RuntimeType      string `json:"runtime_type"`
	BackendType      string `json:"backend_type"`
	Vendor           string `json:"vendor"`
	OpenAICompatible bool   `json:"openai_compatible"`
	DefaultPort      int    `json:"default_port"`
	HealthCheckPath  string `json:"health_check_path"`
	Description      string `json:"description"`
	TenantID         string `json:"tenant_id,omitempty"`
	OwnerID          string `json:"owner_id,omitempty"`
	CreatedBy        string `json:"created_by"`
	UpdatedBy        string `json:"updated_by"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// EnabledValue is a generic optional value with an enabled switch.
type EnabledValue[T any] struct {
	Enabled bool `json:"enabled"`
	Value   T    `json:"value,omitempty"`
}

// DockerDevice represents a device mapping.
type DockerDevice struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Permissions   string `json:"permissions,omitempty"`
}

// DockerVolume represents a volume mapping.
type DockerVolume struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Readonly      bool   `json:"readonly,omitempty"`
}

// RuntimeEnvironmentDockerSpec holds docker-specific infrastructure config.
type RuntimeEnvironmentDockerSpec struct {
	ID                    string                    `json:"id"`
	RuntimeEnvironmentID  string                    `json:"runtime_environment_id"`
	Image                 string                    `json:"image"`
	ImagePullPolicy       string                    `json:"image_pull_policy"`
	Devices               EnabledValue[[]DockerDevice] `json:"devices"`
	Privileged            EnabledValue[bool]        `json:"privileged"`
	IPCMode               EnabledValue[string]      `json:"ipc_mode"`
	UTSMode               EnabledValue[string]      `json:"uts_mode"`
	NetworkMode           EnabledValue[string]      `json:"network_mode"`
	ShmSize               EnabledValue[string]      `json:"shm_size"`
	GroupAdd              EnabledValue[[]string]    `json:"group_add"`
	SecurityOptions       EnabledValue[[]string]    `json:"security_options"`
	Ulimits               EnabledValue[map[string]string] `json:"ulimits"`
	RestartPolicy         EnabledValue[string]      `json:"restart_policy"`
	GPUVisibleEnvKey      string                    `json:"gpu_visible_env_key"`
	CreatedAt             string                    `json:"created_at"`
	UpdatedAt             string                    `json:"updated_at"`
}

// EnvMapping maps a variable to an environment key.
type EnvMapping struct {
	Key       string `json:"key"`
	ValueFrom string `json:"value_from"`
}

// VolumeMapping maps a variable to a volume mount.
type VolumeMapping struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Readonly      bool   `json:"readonly,omitempty"`
}

// PortMapping maps a variable to a port binding.
type PortMapping struct {
	HostPort      string `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
}

// RunTemplate represents a reusable launch template.
type RunTemplate struct {
	ID                string                       `json:"id"`
	Name              string                       `json:"name"`
	DisplayName       string                       `json:"display_name"`
	RuntimeType       string                       `json:"runtime_type"`
	Vendor            string                       `json:"vendor"`
	BackendType       string                       `json:"backend_type"`
	RequiredVariables []string                     `json:"required_variables"`
	OptionalVariables []string                     `json:"optional_variables"`
	EnvMappings       EnabledValue[[]EnvMapping]   `json:"env_mappings"`
	ArgsTemplate      []string                     `json:"args_template"`
	VolumeMappings    EnabledValue[[]VolumeMapping] `json:"volume_mappings"`
	PortMappings      EnabledValue[[]PortMapping]  `json:"port_mappings"`
	BackendFlags      EnabledValue[map[string]string] `json:"backend_flags"`
	Description       string                       `json:"description"`
	TenantID          string                       `json:"tenant_id,omitempty"`
	OwnerID           string                       `json:"owner_id,omitempty"`
	CreatedBy         string                       `json:"created_by"`
	UpdatedBy         string                       `json:"updated_by"`
	CreatedAt         string                       `json:"created_at"`
	UpdatedAt         string                       `json:"updated_at"`
}

// ModelDeployment represents a desired model deployment.
type ModelDeployment struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	DisplayName          string   `json:"display_name"`
	ModelArtifactID      string   `json:"model_artifact_id"`
	RuntimeEnvironmentID string   `json:"runtime_environment_id"`
	RunTemplateID        string   `json:"run_template_id"`
	Replicas             int      `json:"replicas"`
	DesiredState         string   `json:"desired_state"`
	Status               string   `json:"status"`
	NodeID               string   `json:"node_id"`
	GPUIds               []string `json:"gpu_ids"`
	HostPort             int      `json:"host_port"`
	ServedModelName      string   `json:"served_model_name"`
	MaxModelLen          int      `json:"max_model_len"`
	TensorParallelSize   int      `json:"tensor_parallel_size"`
	GPUMemoryUtilization float64  `json:"gpu_memory_utilization"`
	Dtype                string   `json:"dtype"`
	GPUVisibleEnvKey     string   `json:"gpu_visible_env_key"`
	EnvOverrides         map[string]string `json:"env_overrides"`
	ArgOverrides         map[string]string `json:"arg_overrides"`
	ExtraArgs            []string `json:"extra_args"`
	ScheduleMode         string   `json:"schedule_mode"`
	PlacementStrategy    string   `json:"placement_strategy"`
	ExposeMode           string   `json:"expose_mode"`
	ServicePath          string   `json:"service_path"`
	TenantID             string   `json:"tenant_id"`
	OwnerID              string   `json:"owner_id,omitempty"`
	CreatedBy            string   `json:"created_by"`
	UpdatedBy            string   `json:"updated_by"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
}

// ModelInstance represents an actual running model instance.
type ModelInstance struct {
	ID              string   `json:"id"`
	DeploymentID    string   `json:"deployment_id"`
	ReplicaIndex    int      `json:"replica_index"`
	NodeID          string   `json:"node_id"`
	AgentID         string   `json:"agent_id"`
	RuntimeType     string   `json:"runtime_type"`
	GPUIds          []string `json:"gpu_ids"`
	GPULeaseIDs     []string `json:"gpu_lease_ids"`
	DesiredState    string   `json:"desired_state"`
	ActualState     string   `json:"actual_state"`
	ContainerID     string   `json:"container_id"`
	ProcessID       int      `json:"process_id"`
	RemoteURL       string   `json:"remote_url"`
	EndpointURL     string   `json:"endpoint_url"`
	HostPort        int      `json:"host_port"`
	ContainerPort   int      `json:"container_port"`
	RestartCount    int      `json:"restart_count"`
	LastError       string   `json:"last_error"`
	LastExitCode    int      `json:"last_exit_code"`
	ResolvedRunSpec string   `json:"resolved_run_spec"`
	StartedAt       string   `json:"started_at,omitempty"`
	StoppedAt       string   `json:"stopped_at,omitempty"`
	LastHeartbeatAt string   `json:"last_heartbeat_at,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// GpuLease represents a GPU resource lock.
type GpuLease struct {
	ID           string `json:"id"`
	GpuID        string `json:"gpu_id"`
	NodeID       string `json:"node_id"`
	DeploymentID string `json:"deployment_id"`
	InstanceID   string `json:"instance_id"`
	TenantID     string `json:"tenant_id"`
	Status       string `json:"status"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Node represents a GPU server (Agent node).
type Node struct {
	ID              string     `json:"id"`
	AgentID         string     `json:"agent_id"`
	Hostname        string     `json:"hostname"`
	AdvertisedAddr  string     `json:"advertised_address"`
	MetricsEnabled  bool       `json:"metrics_enabled"`
	MetricsScheme   string     `json:"metrics_scheme"`
	MetricsPort     int        `json:"metrics_port"`
	MetricsPath     string     `json:"metrics_path"`
	Status          string     `json:"status"` // online / offline
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	TenantID        string     `json:"tenant_id"`
	OwnerID         *string    `json:"owner_id,omitempty"`
	CreatedBy       string     `json:"created_by"`
	UpdatedBy       string     `json:"updated_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
