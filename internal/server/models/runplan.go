package models

// ResolvedRunPlan is the frozen, immutable run plan generated at instance start time.
// Stored in the resolved_run_plans table. Each start/restart creates a new row.
type ResolvedRunPlan struct {
	ID                    string `json:"id"`
	DeploymentID          string `json:"deployment_id"`
	InstanceID            string `json:"instance_id"`
	TenantID              string `json:"tenant_id"`
	BackendRuntimeID      string `json:"backend_runtime_id"`
	NodeRuntimeOverrideID string `json:"node_runtime_override_id"`
	PlanJSON              string `json:"plan_json"`
	DockerPreview         string `json:"docker_preview"`
	InputHash             string `json:"input_hash"`
	PlanHash              string `json:"plan_hash"`
	CreatedBy             string `json:"created_by"`
	CreatedAt             string `json:"created_at"`
}
