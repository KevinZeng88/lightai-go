package runplan

import (
	"database/sql"
	"fmt"
)

// DBQuerier is a minimal DB interface for the validator.
type DBQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// DryRunInput holds all data needed for dry-run validation.
type DryRunInput struct {
	NodeID          string
	GPUIds          []string
	HostPort        int
	RuntimeVendor   string
	ModelArtifactID string
	ModelPath       string
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
		} else if err == nil && nodeStatus != "online" {
			result.Errors = append(result.Errors, fmt.Sprintf("specified node is not online (status=%s)", nodeStatus))
		}
	}

	// 2. Model path non-empty.
	if in.ModelArtifactID != "" && in.ModelPath == "" {
		result.Errors = append(result.Errors, "model path is empty for the specified artifact")
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
		if in.RuntimeVendor != "" && in.RuntimeVendor != "custom" && gpuVendor != "" && gpuVendor != in.RuntimeVendor {
			result.Errors = append(result.Errors, fmt.Sprintf("GPU %s vendor=%s does not match runtime vendor=%s", gpuID, gpuVendor, in.RuntimeVendor))
		}
		if in.RuntimeVendor == "custom" {
			result.Warnings = append(result.Warnings, "Runtime vendor=custom, GPU vendor strict matching skipped. Please verify compatibility.")
		}
	}

	// 4. Host port conflict.
	if in.HostPort > 0 {
		var existingPort int
		if err := db.QueryRow(`SELECT host_port FROM model_instances WHERE host_port = ? AND actual_state IN ('pending','initializing','starting','running') LIMIT 1`, in.HostPort).Scan(&existingPort); err == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("host_port %d is already in use by another model instance", in.HostPort))
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}
