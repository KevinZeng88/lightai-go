package models

// ModelInstance represents a single runtime entity — one container on one node.
type ModelInstance struct {
	ID               string `json:"id"`
	DeploymentID     string `json:"deployment_id"`
	TenantID         string `json:"tenant_id"`
	ReplicaIndex     int    `json:"replica_index"`
	NodeID           string `json:"node_id"`
	AgentID          string `json:"agent_id"`
	AssignedGPUsJSON string `json:"assigned_gpus_json"`
	GPULeaseIDsJSON  string `json:"gpu_lease_ids_json"`
	HostPort         int    `json:"host_port"`
	ContainerPort    int    `json:"container_port"`
	CurrentRunPlanID string `json:"current_run_plan_id"`
	ActualState      string `json:"actual_state"`
	DesiredState     string `json:"desired_state"`
	ContainerID      string `json:"container_id"`
	EndpointURL      string `json:"endpoint_url"`
	RestartCount     int    `json:"restart_count"`
	LastError        string `json:"last_error"`
	StartedAt        string `json:"started_at"`
	StoppedAt        string `json:"stopped_at"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// GpuLease represents a GPU reservation for a model instance.
type GpuLease struct {
	ID           string `json:"id"`
	GpuID        string `json:"gpu_id"`
	NodeID       string `json:"node_id"`
	DeploymentID string `json:"deployment_id"`
	InstanceID   string `json:"instance_id"`
	TenantID     string `json:"tenant_id"`
	Status       string `json:"status"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	ReservedAt   string `json:"reserved_at"`
	ActivatedAt  string `json:"activated_at,omitempty"`
	ReleasedAt   string `json:"released_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// AgentTask represents a task dispatched to an Agent for execution.
type AgentTask struct {
	ID             string `json:"id"`
	TaskType       string `json:"task_type"`
	Status         string `json:"status"`
	TenantID       string `json:"tenant_id"`
	DeploymentID   string `json:"deployment_id"`
	InstanceID     string `json:"instance_id,omitempty"`
	NodeID         string `json:"node_id"`
	AgentID        string `json:"agent_id,omitempty"`
	RequestedBy    string `json:"requested_by"`
	Payload        string `json:"payload"`
	Result         string `json:"result,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	RetryCount     int    `json:"retry_count"`
	ClaimedAt      string `json:"claimed_at,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
