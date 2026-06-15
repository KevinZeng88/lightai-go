package api

import (
	"fmt"
	"time"

	"lightai-go/internal/server/db"

	"github.com/google/uuid"
)

// InstanceState values.
const (
	InstanceStatePending   = "pending"
	InstanceStateStarting  = "starting"
	InstanceStateRunning   = "running"
	InstanceStateStopping  = "stopping"
	InstanceStateStopped   = "stopped"
	InstanceStateFailed    = "failed"
	InstanceStateUnknown   = "unknown"
)

// CreateInstance inserts a new model_instance row with actual_state=pending.
func CreateInstance(database *db.DB, deploymentID, nodeID, agentID, runtimeType, tenantID, resolvedSpecJSON string, hostPort int, gpuIDsJSON, leaseIDsJSON string) (string, error) {
	instanceID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := database.Exec(
		`INSERT INTO model_instances
		 (id, deployment_id, replica_index, node_id, agent_id, runtime_type,
		  gpu_ids, gpu_lease_ids, desired_state, actual_state,
		  host_port, resolved_run_spec, started_at, created_at, updated_at)
		 VALUES (?, ?, 0, ?, ?, ?, ?, ?, 'running', ?, ?, ?, ?, ?, ?)`,
		instanceID, deploymentID, nodeID, agentID, runtimeType,
		gpuIDsJSON, leaseIDsJSON, InstanceStatePending,
		hostPort, resolvedSpecJSON, now, now, now,
	)
	if err != nil {
		return "", fmt.Errorf("create instance: %w", err)
	}
	return instanceID, nil
}

// UpdateInstanceRunning updates an instance to running state with container info.
func UpdateInstanceRunning(database *db.DB, instanceID, containerID, endpointURL string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := database.Exec(
		`UPDATE model_instances SET actual_state = ?, container_id = ?, endpoint_url = ?, started_at = ?, updated_at = ? WHERE id = ?`,
		InstanceStateRunning, containerID, endpointURL, now, now, instanceID,
	)
	return err
}

// UpdateInstanceStopping marks an instance as stopping.
func UpdateInstanceStopping(database *db.DB, instanceID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := database.Exec(
		`UPDATE model_instances SET actual_state = ?, updated_at = ? WHERE id = ?`,
		InstanceStateStopping, now, instanceID,
	)
	return err
}

// UpdateInstanceStopped marks an instance as stopped.
func UpdateInstanceStopped(database *db.DB, instanceID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := database.Exec(
		`UPDATE model_instances SET actual_state = ?, stopped_at = ?, updated_at = ? WHERE id = ?`,
		InstanceStateStopped, now, now, instanceID,
	)
	return err
}

// UpdateInstanceFailed marks an instance as failed with an error message.
func UpdateInstanceFailed(database *db.DB, instanceID, lastError string, exitCode int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := database.Exec(
		`UPDATE model_instances SET actual_state = ?, last_error = ?, last_exit_code = ?, updated_at = ? WHERE id = ?`,
		InstanceStateFailed, lastError, exitCode, now, instanceID,
	)
	return err
}
