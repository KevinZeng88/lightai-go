package api

import (
	"encoding/json"
	"net/http"
	"fmt"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/resolver"

	"github.com/google/uuid"
)

// HandleStartDeployment starts a model deployment.
// POST /api/v1/model-deployments/{id}/start
func (h *ModelHandler) HandleStartDeployment(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.PathValue("id")
	if deploymentID == "" {
		writeError(w, http.StatusBadRequest, "deployment id is required")
		return
	}

	dep := h.getModelDeployment(deploymentID)
	if dep == nil {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}

	// Tenant scope check.
	if !tenantScopeCheck(r, strVal(dep, "tenant_id", "")) {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}

	// Already running?
	if strVal(dep, "desired_state", "") == "running" && strVal(dep, "status", "") == "running" {
		// Return the existing running instance.
		var existingInstanceID string
		h.DB.QueryRow(`SELECT id FROM model_instances WHERE deployment_id = ? AND actual_state = ? ORDER BY created_at DESC LIMIT 1`,
			deploymentID, InstanceStateRunning).Scan(&existingInstanceID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "already_running",
			"instance_id": existingInstanceID,
		})
		return
	}

	// Check for existing in-flight start task.
	var existingTaskID string
	err := h.DB.QueryRow(
		`SELECT id FROM agent_tasks WHERE deployment_id = ? AND task_type = 'model_instance_start' AND status IN (?, ?, ?) LIMIT 1`,
		deploymentID, TaskStatusPending, TaskStatusClaimed, TaskStatusInProgress,
	).Scan(&existingTaskID)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"error": "start already in progress",
			"existing_task_id": existingTaskID,
		})
		return
	}

	currentUserID := userID(r)
	currentTenantID := tenantID(r)

	// Collect deployment details for resolution.
	nodeID := strVal(dep, "node_id", "")
	deploymentNodeID := nodeID
	_ = deploymentNodeID

	modelArtifactID := strVal(dep, "model_artifact_id", "")
	runtimeEnvID := strVal(dep, "runtime_environment_id", "")
	runTemplateID := strVal(dep, "run_template_id", "")

	// Fetch cross-references.
	var modelPath, runtimeVendor, runtimeType, backendType string
	var defaultPort int
	var templateRequiredVars []string
	var templateArgsTemplate []string

	var artifactName string
	h.DB.QueryRow(`SELECT path, name FROM model_artifacts WHERE id = ?`, modelArtifactID).Scan(&modelPath, &artifactName)
	var depName string
	h.DB.QueryRow(`SELECT name FROM model_deployments WHERE id = ?`, deploymentID).Scan(&depName)
	h.DB.QueryRow(`SELECT vendor, runtime_type, backend_type, default_port FROM runtime_environments WHERE id = ?`,
		runtimeEnvID).Scan(&runtimeVendor, &runtimeType, &backendType, &defaultPort)

	var reqVarsJSON, argsTplJSON string
	h.DB.QueryRow(`SELECT required_variables, args_template FROM run_templates WHERE id = ?`,
		runTemplateID).Scan(&reqVarsJSON, &argsTplJSON)
	json.Unmarshal([]byte(reqVarsJSON), &templateRequiredVars)
	json.Unmarshal([]byte(argsTplJSON), &templateArgsTemplate)

	// GPU IDs from deployment (LightAI internal UUIDs for leases).
	var gpuIDsJSON string
	h.DB.QueryRow(`SELECT gpu_ids FROM model_deployments WHERE id = ?`, deploymentID).Scan(&gpuIDsJSON)
	var gpuIDs []string
	json.Unmarshal([]byte(gpuIDsJSON), &gpuIDs)

	// Resolve GPU indices for Docker DeviceRequests and CUDA_VISIBLE_DEVICES.
	// LightAI internal UUIDs are for DB/lease only; Docker needs NVIDIA index (0,1,2...).
	var gpuDeviceIDs []string
	for _, gpuID := range gpuIDs {
		var indexNum int
		if err := h.DB.QueryRow(`SELECT index_num FROM gpu_devices WHERE id = ?`, gpuID).Scan(&indexNum); err == nil {
			gpuDeviceIDs = append(gpuDeviceIDs, fmt.Sprintf("%d", indexNum))
		}
	}
	// Fallback: if no indices found (e.g. test with mock GPU), use original IDs.
	if len(gpuDeviceIDs) == 0 {
		gpuDeviceIDs = gpuIDs
	}

	hostPort := intVal(dep, "host_port", 0)
	servedModelName := strVal(dep, "served_model_name", "")
	maxModelLen := intVal(dep, "max_model_len", 0)
	gpuMemUtil := floatVal(dep, "gpu_memory_utilization", 0.9)

	// Validate via dry-run validator.
	dryRun := resolver.ValidateDryRun(h.DB.DB, resolver.DryRunInput{
		NodeID:               nodeID,
		GPUIds:               gpuIDs,
		HostPort:             hostPort,
		RuntimeVendor:        runtimeVendor,
		ModelArtifactID:      modelArtifactID,
		ModelPath:            modelPath,
		TemplateRequiredVars: templateRequiredVars,
	})
	if !dryRun.Valid {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "validation failed",
			"errors":  dryRun.Errors,
			"warnings": dryRun.Warnings,
		})
		return
	}

	// Resolve to generate run spec using the shared input builder.
	resolveIn := buildResolveInputForDeployment(h.DB, resolveDeploymentInput{
		ArtifactID:  modelArtifactID,
		EnvID:       runtimeEnvID,
		TemplateID:  runTemplateID,
		DeployID:    deploymentID,
		NodeID:      nodeID,
		GPUIds:      gpuIDs,
		HostPort:    hostPort,
		ModelPath:   modelPath,
		Vendor:      runtimeVendor,
		RuntimeType: runtimeType,
		BackendType: backendType,
		DefaultPort: defaultPort,
		ServedModelName:      servedModelName,
		MaxModelLen:          maxModelLen,
		GPUMemoryUtilization: gpuMemUtil,
	})
	resolveIn.InstanceID = uuid.NewString()
	resolveIn.DeploymentID = deploymentID
	spec, resolveErrors, warnings := resolver.Resolve(resolveIn)
	if len(resolveErrors) > 0 {
		errMsgs := make([]string, len(resolveErrors))
		for i, e := range resolveErrors {
			errMsgs[i] = e.Error()
		}
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "resolution failed", "errors": errMsgs,
		})
		return
	}

	// Convert ResolvedRunSpec to AgentRunSpec JSON for task payload.
	spec.InstanceID = "" // will be filled after instance creation
	specJSON, _ := json.Marshal(spec)

	// Fetch agent_id for the node.
	var agentID string
	h.DB.QueryRow(`SELECT agent_id FROM nodes WHERE id = ?`, nodeID).Scan(&agentID)

	// Create instance and leases in transaction.
	now := time.Now().UTC().Format(time.RFC3339)
	instanceID := uuid.NewString()
	spec.InstanceID = instanceID
	spec.AgentID = agentID
	specJSON, _ = json.Marshal(spec)

	gpuIDsJSONBytes, _ := json.Marshal(gpuIDs)

	leaseIDs, err := CreateLeases(h.DB, gpuIDs, nodeID, deploymentID, instanceID, currentTenantID)
	if err != nil {
		log.Error("start deployment: cannot create leases", "deployment_id", deploymentID, "error", err)
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	leaseIDsJSONBytes, _ := json.Marshal(leaseIDs)

	_, err = h.DB.Exec(
		`INSERT INTO model_instances
		 (id, deployment_id, replica_index, node_id, agent_id, runtime_type,
		  gpu_ids, gpu_lease_ids, desired_state, actual_state,
		  host_port, resolved_run_spec, tenant_id, created_at, updated_at)
		 VALUES (?, ?, 0, ?, ?, ?, ?, ?, 'running', 'pending', ?, ?, ?, ?, ?)`,
		instanceID, deploymentID, nodeID, agentID, runtimeType,
		string(gpuIDsJSONBytes), string(leaseIDsJSONBytes),
		hostPort, string(specJSON), currentTenantID, now, now,
	)
	if err != nil {
		// Rollback leases.
		FailLeases(h.DB, leaseIDs)
		log.Error("start deployment: cannot create instance", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create instance")
		return
	}

	// Update deployment status.
	h.DB.Exec(`UPDATE model_deployments SET status = 'pending', desired_state = 'running', updated_at = ? WHERE id = ?`, now, deploymentID)

	// Create agent task.
	taskID := uuid.NewString()
	taskPayload, _ := json.Marshal(spec)
	h.DB.Exec(
		`INSERT INTO agent_tasks (id, task_type, status, tenant_id, deployment_id, instance_id, node_id, requested_by, payload, timeout_seconds, created_at, updated_at)
		 VALUES (?, 'model_instance_start', 'pending', ?, ?, ?, ?, ?, ?, 300, ?, ?)`,
		taskID, currentTenantID, deploymentID, instanceID, nodeID, currentUserID, string(taskPayload), now, now,
	)

	// Audit.
	audit(h.DB, "start", "model_deployment", deploymentID, `{"instance_id":"`+instanceID+`"}`, currentUserID)

	cmdPreview := resolver.EquivalentCommandPreview(spec)
	redactedPreview := redactCommandPreview(cmdPreview)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"instance_id":              instanceID,
		"task_id":                  taskID,
		"status":                   "pending",
		"equivalent_command_preview": redactedPreview,
		"warnings":                 warnings,
	})
}

// HandleStopDeployment stops a running model deployment.
// POST /api/v1/model-deployments/{id}/stop
func (h *ModelHandler) HandleStopDeployment(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.PathValue("id")
	if deploymentID == "" {
		writeError(w, http.StatusBadRequest, "deployment id is required")
		return
	}

	dep := h.getModelDeployment(deploymentID)
	if dep == nil {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}

	if !tenantScopeCheck(r, strVal(dep, "tenant_id", "")) {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}

	status := strVal(dep, "status", "")
	if status == "stopped" || status == "failed" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "already_stopped",
			"deployment_status": status,
		})
		return
	}

	// Check for existing in-flight stop task.
	var existingStopTaskID string
	err := h.DB.QueryRow(
		`SELECT id FROM agent_tasks WHERE deployment_id = ? AND task_type = 'model_instance_stop' AND status IN (?, ?, ?) LIMIT 1`,
		deploymentID, TaskStatusPending, TaskStatusClaimed, TaskStatusInProgress,
	).Scan(&existingStopTaskID)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"error": "stop already in progress",
			"existing_task_id": existingStopTaskID,
		})
		return
	}

	currentUserID := userID(r)
	currentTenantID := tenantID(r)
	nodeID := strVal(dep, "node_id", "")

	// Find active instance.
	var instanceID, containerID string
	err = h.DB.QueryRow(
		`SELECT id, container_id FROM model_instances WHERE deployment_id = ? AND actual_state IN ('pending','starting','running') ORDER BY created_at DESC LIMIT 1`,
		deploymentID,
	).Scan(&instanceID, &containerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no active instance found for this deployment")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Mark instance as stopping.
	UpdateInstanceStopping(h.DB, instanceID)

	// Update deployment.
	h.DB.Exec(`UPDATE model_deployments SET desired_state = 'stopped', status = 'stopping', updated_at = ? WHERE id = ?`, now, deploymentID)

	// Create stop task.
	taskID := uuid.NewString()
	taskPayload, _ := json.Marshal(map[string]interface{}{
		"instance_id":  instanceID,
		"container_id": containerID,
		"node_id":      nodeID,
	})
	h.DB.Exec(
		`INSERT INTO agent_tasks (id, task_type, status, tenant_id, deployment_id, instance_id, node_id, requested_by, payload, timeout_seconds, created_at, updated_at)
		 VALUES (?, 'model_instance_stop', 'pending', ?, ?, ?, ?, ?, ?, 60, ?, ?)`,
		taskID, currentTenantID, deploymentID, instanceID, nodeID, currentUserID, string(taskPayload), now, now,
	)

	audit(h.DB, "stop", "model_deployment", deploymentID, `{"instance_id":"`+instanceID+`"}`, currentUserID)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"instance_id": instanceID,
		"task_id":     taskID,
		"status":      "stopping",
	})
}

// HandleGetInstanceLogs returns logs for a model instance.
// GET /api/v1/model-instances/{id}/logs
//
// Dedup strategy:
//  1. Succeeded task with logs → 200 + logs content
//  2. Pending/claimed/in_progress task → 202 + existing task_id (no new task)
//  3. Failed/timed_out + no refresh → error
//  4. No task or refresh=true → create new task → 202
func (h *ModelHandler) HandleGetInstanceLogs(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("id")
	if instanceID == "" {
		writeError(w, http.StatusBadRequest, "instance id is required")
		return
	}

	// Use scanMiRow (same as HandleGetModelInstance) for reliable instance lookup.
	miRow := h.DB.QueryRow(`SELECT id, deployment_id, replica_index, node_id, agent_id, runtime_type, gpu_ids, gpu_lease_ids,
	 desired_state, actual_state, container_id, process_id, remote_url, endpoint_url, host_port, container_port,
	 restart_count, last_error, last_exit_code, resolved_run_spec,
	 started_at, stopped_at, last_heartbeat_at, created_at, updated_at
	 FROM model_instances WHERE id = ?`, instanceID)
	inst := scanMiRow(miRow)
	if inst == nil {
		log.Warn("logs: instance not found", "instance_id", instanceID)
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	instTenant := strVal(inst, "tenant_id", "")
	instNode := strVal(inst, "node_id", "")
	instContainer := strVal(inst, "container_id", "")
	log.Info("logs: instance fetched", "instance_id", instanceID, "tenant", instTenant, "node", instNode, "container", instContainer[:20])

	if !tenantScopeCheck(r, instTenant) {
		log.Warn("logs: tenant scope check failed", "instance_id", instanceID, "inst_tenant", instTenant)
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	forceRefresh := r.URL.Query().Get("refresh") == "true"

	// Case 1: Check for an existing succeeded logs task with content.
	if !forceRefresh {
		var logsContent string
		var logsTaskID string
		err := h.DB.QueryRow(
			`SELECT id, result FROM agent_tasks WHERE instance_id = ? AND task_type = 'model_instance_logs' AND status = ? ORDER BY created_at DESC LIMIT 1`,
			instanceID, TaskStatusSucceeded,
		).Scan(&logsTaskID, &logsContent)
		if err == nil {
			// Extract logs from result JSON.
			var taskResult TaskResult
			if e := json.Unmarshal([]byte(logsContent), &taskResult); e == nil && taskResult.LogsSummary != "" {
				logText := redactDetailString(taskResult.LogsSummary)
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"instance_id":    instanceID,
					"logs":           logText,
					"source_task_id": logsTaskID,
				})
				return
			}
			// Result exists but logs field is empty — fall through to check pending tasks.
		}

		// Case 2: Check for in-flight task (pending/claimed/in_progress).
		var inflightTaskID, inflightStatus string
		err = h.DB.QueryRow(
			`SELECT id, status FROM agent_tasks WHERE instance_id = ? AND task_type = 'model_instance_logs' AND status IN (?,?,?) ORDER BY created_at DESC LIMIT 1`,
			instanceID, TaskStatusPending, TaskStatusClaimed, TaskStatusInProgress,
		).Scan(&inflightTaskID, &inflightStatus)
		if err == nil {
			writeJSON(w, http.StatusAccepted, map[string]interface{}{
				"task_id": inflightTaskID,
				"status":  inflightStatus,
				"message": "logs task in progress",
			})
			return
		}

		// Case 3: Latest task failed/timed_out — return error unless force refresh.
		var failedTaskID, failedStatus string
		err = h.DB.QueryRow(
			`SELECT id, status FROM agent_tasks WHERE instance_id = ? AND task_type = 'model_instance_logs' ORDER BY created_at DESC LIMIT 1`,
			instanceID,
		).Scan(&failedTaskID, &failedStatus)
		if err == nil && (failedStatus == TaskStatusFailed || failedStatus == TaskStatusTimedOut) {
			if !forceRefresh {
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"instance_id": instanceID,
					"status":      failedStatus,
					"task_id":     failedTaskID,
					"message":     "previous logs task " + failedStatus + ". Use ?refresh=true to retry.",
				})
				return
			}
		}
	}

	// Case 4: Create a new logs task.
	containerID := strVal(inst, "container_id", "")
	if containerID == "" {
		log.Warn("logs: no container_id", "instance_id", instanceID, "inst_keys", func() []string {
			keys := make([]string, 0, len(inst))
			for k := range inst { keys = append(keys, k) }
			return keys
		}())
		writeError(w, http.StatusNotFound, "instance has no container yet")
		return
	}

	nodeID := strVal(inst, "node_id", "")
	deploymentID := strVal(inst, "deployment_id", "")
	currentUserID := userID(r)
	tenantIDStr := tenantID(r)

	now := time.Now().UTC().Format(time.RFC3339)
	taskID := uuid.NewString()
	containerName := "lightai-" + instanceID
	if len(instanceID) > 12 {
		containerName = "lightai-" + instanceID[:12]
	}
	taskPayload, _ := json.Marshal(map[string]interface{}{
		"instance_id":    instanceID,
		"container_id":   containerID,
		"container_name": containerName,
		"node_id":        nodeID,
		"deployment_id":  deploymentID,
	})

	h.DB.Exec(
		`INSERT INTO agent_tasks (id, task_type, status, tenant_id, deployment_id, instance_id, node_id, requested_by, payload, timeout_seconds, created_at, updated_at)
		 VALUES (?, 'model_instance_logs', 'pending', ?, ?, ?, ?, ?, ?, 30, ?, ?)`,
		taskID, tenantIDStr, deploymentID, instanceID, nodeID, currentUserID, string(taskPayload), now, now,
	)

	audit(h.DB, "logs", "model_instance", instanceID, `{}`, currentUserID)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"task_id": taskID,
		"status":  "pending",
		"message": "logs task dispatched, check back shortly",
	})
}

// getInstance fetches a model instance by ID with tenant_id.

func (h *ModelHandler) getInstance(id string) map[string]interface{} {
	var id2, deploymentID, nodeID, agentID, runtimeType, gpuIDsJSON, leaseIDsJSON string
	var actualState, containerID, endpointURL, lastError, startedAt, stoppedAt, createdAt, updatedAt string
	var hostPort, containerPort, lastExitCode int

	var tenantIDFromDB string
	err := h.DB.QueryRow(
		`SELECT id, deployment_id, node_id, agent_id, runtime_type, gpu_ids, gpu_lease_ids,
		        actual_state, container_id, endpoint_url, host_port, container_port,
		        last_error, last_exit_code, started_at, stopped_at, created_at, updated_at,
		        COALESCE(tenant_id,'')
		 FROM model_instances WHERE id = ?`, id,
	).Scan(&id2, &deploymentID, &nodeID, &agentID, &runtimeType,
		&gpuIDsJSON, &leaseIDsJSON, &actualState, &containerID, &endpointURL,
		&hostPort, &containerPort, &lastError, &lastExitCode,
		&startedAt, &stoppedAt, &createdAt, &updatedAt, &tenantIDFromDB,
	)
	if err != nil {
		log.Warn("getInstance scan failed", "instance_id", id, "error", err)
		return nil
	}

	var gpuIDs, leaseIDs []string
	json.Unmarshal([]byte(gpuIDsJSON), &gpuIDs)
	json.Unmarshal([]byte(leaseIDsJSON), &leaseIDs)

	return map[string]interface{}{
		"id": id2, "deployment_id": deploymentID, "node_id": nodeID,
		"agent_id": agentID, "runtime_type": runtimeType,
		"gpu_ids": gpuIDs, "gpu_lease_ids": leaseIDs,
		"actual_state": actualState, "container_id": containerID,
		"endpoint_url": endpointURL, "host_port": hostPort,
		"container_port": containerPort, "last_error": lastError,
		"last_exit_code": lastExitCode, "started_at": startedAt,
		"stopped_at": stoppedAt, "created_at": createdAt, "updated_at": updatedAt,
		"tenant_id": tenantIDFromDB,
	}
}
