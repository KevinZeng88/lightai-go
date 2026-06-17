package models

// ModelArtifact represents a registered AI model file with metadata.
type ModelArtifact struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	DisplayName          string `json:"display_name"`
	Source               string `json:"source"`
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
	OwnerID              string `json:"owner_id"`
	CreatedBy            string `json:"created_by"`
	UpdatedBy            string `json:"updated_by"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}
