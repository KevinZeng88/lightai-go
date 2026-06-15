package api

import (
	"encoding/json"
	"net/http"

	"time"
	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"
)

// TaskHandler handles Agent task result reporting.
type TaskHandler struct {
	DB *db.DB
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(database *db.DB) *TaskHandler {
	return &TaskHandler{DB: database}
}

// TaskResult is the payload the Agent sends when reporting task completion.
type TaskResult struct {
	TaskID       string `json:"task_id"`
	NodeID       string `json:"node_id"`
	Success      bool   `json:"success"`
	InstanceID   string `json:"instance_id"`
	DeploymentID string `json:"deployment_id,omitempty"`
	ContainerID  string `json:"container_id"`
	RuntimeState string `json:"runtime_state"`
	ExitCode     int    `json:"exit_code"`
	ErrorMessage string `json:"error_message"`
	LogsSummary  string `json:"logs_summary,omitempty"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
}

// HandleTaskResult processes a task result reported by an Agent.
// POST /api/v1/agent/tasks/{id}/result
func (h *TaskHandler) HandleTaskResult(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	var result TaskResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		writeError(w, http.StatusBadRequest, "invalid task result JSON")
		return
	}
	result.TaskID = taskID

	// Fetch the task to get its type and associated IDs.
	var taskType, instanceID, deploymentID, tenantID, taskNodeID, taskStatus string
	err := h.DB.QueryRow(
		`SELECT task_type, COALESCE(instance_id,''), deployment_id, tenant_id, node_id, status FROM agent_tasks WHERE id = ?`,
		taskID,
	).Scan(&taskType, &instanceID, &deploymentID, &tenantID, &taskNodeID, &taskStatus)
	if err != nil {
		log.Error("task result: task not found", "task_id", taskID, "error", err)
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	// Security: validate that the task belongs to the reporting node.
	if result.NodeID != "" && result.NodeID != taskNodeID {
		log.Warn("task result: node mismatch", "task_id", taskID, "task_node", taskNodeID, "report_node", result.NodeID)
		writeError(w, http.StatusForbidden, "task does not belong to this node")
		return
	}

	// Idempotency: if task is already in a terminal state, return success.
	if IsTaskTerminal(taskStatus) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_reported", "task_status": taskStatus})
		return
	}

	result.InstanceID = instanceID
	result.DeploymentID = deploymentID

	switch taskType {
	case "model_instance_start":
		h.handleStartResult(taskID, result, deploymentID)
	case "model_instance_stop":
		h.handleStopResult(taskID, result, deploymentID)
	case "model_instance_logs":
		h.handleLogsResult(taskID, result)
	default:
		writeError(w, http.StatusBadRequest, "unknown task type: "+taskType)
		return
	}

	// Mark task as completed/failed (only if not already terminal).
	newStatus := TaskStatusSucceeded
	if !result.Success {
		newStatus = TaskStatusFailed
	}
	now := time.Now().UTC().Format(time.RFC3339)
	resultJSON, _ := json.Marshal(result)
	h.DB.Exec(
		`UPDATE agent_tasks SET status = ?, result = ?, finished_at = ?, updated_at = ?
		 WHERE id = ? AND status NOT IN (?, ?, ?)`,
		newStatus, string(resultJSON), now, now, taskID,
		TaskStatusSucceeded, TaskStatusFailed, TaskStatusTimedOut,
	)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TaskHandler) handleStartResult(taskID string, result TaskResult, deploymentID string) {
	if result.Success {
		// Use a transaction to atomically update instance and activate leases.
		tx, err := h.DB.Begin()
		if err != nil {
			log.Error("handleStartResult: begin tx failed", "task_id", taskID, "error", err)
			return
		}
		defer tx.Rollback()

		// Update instance to running within the transaction.
		endpointURL := ""
		if result.RuntimeState == "running" && result.ContainerID != "" {
			UpdateInstanceRunningTx(tx, result.InstanceID, result.ContainerID, endpointURL)
		}

		// Activate leases for this instance within the same transaction.
		rows, err := tx.Query(`SELECT id FROM gpu_leases WHERE instance_id = ? AND status = ?`,
			result.InstanceID, LeaseReserved)
		if err == nil {
			defer rows.Close()
			var leaseIDs []string
			for rows.Next() {
				var lid string
				rows.Scan(&lid)
				leaseIDs = append(leaseIDs, lid)
			}
			if len(leaseIDs) > 0 {
				now := nowUTC()
				for _, leaseID := range leaseIDs {
					tx.Exec(
						`UPDATE gpu_leases SET status = ?, activated_at = ?, updated_at = ? WHERE id = ? AND status = ?`,
						LeaseActive, now, now, leaseID, LeaseReserved,
					)
				}
			}
		}

		// Update deployment status within the transaction.
		tx.Exec(`UPDATE model_deployments SET status = 'running', updated_at = ? WHERE id = ?`, nowUTC(), deploymentID)

		if err := tx.Commit(); err != nil {
			log.Error("handleStartResult: commit tx failed", "task_id", taskID, "error", err)
			return
		}

		log.Info("instance started successfully",
			"task_id", taskID,
			"instance_id", result.InstanceID,
			"container_id", result.ContainerID,
		)
	} else {
		// Mark instance as failed.
		UpdateInstanceFailed(h.DB, result.InstanceID, result.ErrorMessage, result.ExitCode)

		// Fail leases.
		rows, err := h.DB.Query(`SELECT id FROM gpu_leases WHERE instance_id = ? AND status = ?`,
			result.InstanceID, LeaseReserved)
		if err == nil {
			defer rows.Close()
			var leaseIDs []string
			for rows.Next() {
				var lid string
				rows.Scan(&lid)
				leaseIDs = append(leaseIDs, lid)
			}
			if len(leaseIDs) > 0 {
				FailLeases(h.DB, leaseIDs)
			}
		}

		// Update deployment status.
		h.DB.Exec(`UPDATE model_deployments SET status = 'failed', updated_at = datetime('now') WHERE id = ?`, deploymentID)

		log.Error("instance start failed",
			"task_id", taskID,
			"instance_id", result.InstanceID,
			"error", result.ErrorMessage,
		)
	}
}

func (h *TaskHandler) handleStopResult(taskID string, result TaskResult, deploymentID string) {
	if result.Success {
		UpdateInstanceStopped(h.DB, result.InstanceID)
		ReleaseInstanceLeases(h.DB, result.InstanceID)
		h.DB.Exec(`UPDATE model_deployments SET status = 'stopped', desired_state = 'stopped', updated_at = datetime('now') WHERE id = ?`, deploymentID)
		log.Info("instance stopped successfully", "task_id", taskID, "instance_id", result.InstanceID)
	} else {
		UpdateInstanceFailed(h.DB, result.InstanceID, result.ErrorMessage, result.ExitCode)
		ReleaseInstanceLeases(h.DB, result.InstanceID)
		log.Error("instance stop failed", "task_id", taskID, "instance_id", result.InstanceID, "error", result.ErrorMessage)
	}
}

func (h *TaskHandler) handleLogsResult(taskID string, result TaskResult) {
	// Logs result is stored in the task result JSON; caller reads it.
	log.Debug("logs task completed", "task_id", taskID, "instance_id", result.InstanceID)
}

// ReleaseInstanceLeases releases all active/reserved leases for an instance.
func ReleaseInstanceLeases(database *db.DB, instanceID string) {
	rows, err := database.Query(`SELECT id FROM gpu_leases WHERE instance_id = ? AND status IN (?, ?)`,
		instanceID, LeaseReserved, LeaseActive)
	if err != nil {
		return
	}
	defer rows.Close()
	var leaseIDs []string
	for rows.Next() {
		var lid string
		rows.Scan(&lid)
		leaseIDs = append(leaseIDs, lid)
	}
	if len(leaseIDs) > 0 {
		ReleaseLeases(database, leaseIDs)
	}
}
