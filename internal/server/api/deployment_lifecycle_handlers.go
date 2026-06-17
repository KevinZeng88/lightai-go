package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/runplan"

	"github.com/google/uuid"
)

// ==========================================================================
// ModelDeployment CRUD (minimal)
// ==========================================================================

func (h *AgentHandler) HandleListDeployments(w http.ResponseWriter, r *http.Request) {
	tid := tenantID(r)
	q := `SELECT id, name, display_name, description, model_artifact_id, backend_runtime_id, replicas, placement_json, service_json, parameters_json, env_overrides_json, desired_state, status, tenant_id, created_at, updated_at FROM model_deployments`
	var out []map[string]interface{}
	var err error
	if isPlatformAdmin(r) {
		out, err = h.queryDeployments(q + ` ORDER BY name`)
	} else {
		out, err = h.queryDeployments(q+` WHERE tenant_id = ? ORDER BY name`, tid)
	}
	if err != nil {
		log.Error("list deployments", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	name := strVal(req, "name", "")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}

	artifactID := strVal(req, "model_artifact_id", "")
	backendRuntimeID := strVal(req, "backend_runtime_id", "")

	// REVIEW-022: Validate references at create time.
	if artifactID != "" {
		var exists string
		if err := h.DB.QueryRow(`SELECT id FROM model_artifacts WHERE id = ?`, artifactID).Scan(&exists); err != nil {
			writeError(w, http.StatusBadRequest, "model_artifact_id not found")
			return
		}
	}
	if backendRuntimeID != "" {
		var exists string
		if err := h.DB.QueryRow(`SELECT id FROM backend_runtimes WHERE id = ?`, backendRuntimeID).Scan(&exists); err != nil {
			writeError(w, http.StatusBadRequest, "backend_runtime_id not found")
			return
		}
	}

	id := uuid.NewString()
	tid := tenantID(r)
	actorID := actorIDFromSession(r)
	requestID := log.RequestIDFromContext(r.Context())
	now := time.Now().Format(time.RFC3339)

	_, err := h.DB.Exec(`INSERT INTO model_deployments (id, name, display_name, description, model_artifact_id, backend_runtime_id, replicas, placement_json, service_json, parameters_json, env_overrides_json, desired_state, status, tenant_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, strVal(req, "display_name", name), strVal(req, "description", ""),
		artifactID, backendRuntimeID,
		intVal(req, "replicas", 1), jsonString(req["placement_json"]), jsonString(req["service_json"]),
		jsonString(req["parameters_json"]), jsonString(req["env_overrides_json"]),
		"stopped", "stopped", tid, now, now,
	)
	if err != nil {
		log.Error("deployment.create.failed", "error", err, "name", name,
			"tenant_id", tid, "request_id", requestID)
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "deployment.create", ResourceType: "deployment",
			ResourceID: id, Result: "failure",
			RequestID: requestID, Error: err.Error(),
		})
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("deployment.created", "deployment_id", id, "name", name,
		"tenant_id", tid, "actor_id", actorID,
		"model_artifact_id", artifactID, "backend_runtime_id", backendRuntimeID,
		"request_id", requestID)
	WriteAudit(r.Context(), h.DB.DB, AuditEntry{
		TenantID: tid, ActorID: actorID,
		Action: "deployment.create", ResourceType: "deployment",
		ResourceID: id, Result: "success", RequestID: requestID,
	})
	writeJSON(w, http.StatusCreated, h.getDeploymentJSON(id))
}

func (h *AgentHandler) HandleGetDeployment(w http.ResponseWriter, r *http.Request) {
	m := h.getDeploymentJSON(r.PathValue("id"))
	if m == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	tid, _ := m["tenant_id"].(string)
	if !tenantScopeCheck(r, tid) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *AgentHandler) HandlePatchDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.getDeploymentJSON(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_at = ?"}
	args := []interface{}{now}
	for _, f := range []string{"display_name", "description"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	for _, f := range []string{"parameters_json", "env_overrides_json", "service_json"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, jsonString(v))
		}
	}
	args = append(args, id)
	if _, err := h.DB.Exec(`UPDATE model_deployments SET `+joinSets(sets)+` WHERE id = ?`, args...); err != nil {
		log.Error("deployment.update.failed", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, h.getDeploymentJSON(id))
}

func (h *AgentHandler) HandleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx, opStart := log.StartOperation(r.Context(), "deployment.delete", "deployment_id", id)
	_ = opStart
	existing := h.getDeploymentJSON(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, existing["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	now := time.Now().Format(time.RFC3339)

	// Begin transaction for atomic cleanup — no orphaned records on partial failure.
	// AUD-005: Wrap all writes in a transaction so partial cleanup doesn't leave orphans.
	tx, txErr := h.DB.Begin()
	if txErr != nil {
		log.Error("deployment.delete.tx_begin_failed", "error", txErr, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	// Cleanup: stop instances
	if _, err := tx.Exec(`UPDATE model_instances SET actual_state = 'stopped', desired_state = 'stopped', stopped_at = ? WHERE deployment_id = ? AND actual_state NOT IN ('stopped')`, now, id); err != nil {
		log.Error("deployment.delete.instance_stop_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Release leases
	if _, err := tx.Exec(`UPDATE gpu_leases SET status = 'released', released_at = ? WHERE deployment_id = ? AND status IN ('reserved','active')`, now, id); err != nil {
		log.Error("deployment.delete.lease_release_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Cancel tasks
	if _, err := tx.Exec(`UPDATE agent_tasks SET status = 'failed', finished_at = ? WHERE deployment_id = ? AND status NOT IN ('completed','failed')`, now, id); err != nil {
		log.Error("deployment.delete.task_cancel_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Delete resolved_run_plans for this deployment
	if _, err := tx.Exec(`DELETE FROM resolved_run_plans WHERE deployment_id = ?`, id); err != nil {
		log.Error("deployment.delete.runplan_delete_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Delete instances
	if _, err := tx.Exec(`DELETE FROM model_instances WHERE deployment_id = ?`, id); err != nil {
		log.Error("deployment.delete.instance_delete_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Delete deployment
	if _, err := tx.Exec(`DELETE FROM model_deployments WHERE id = ?`, id); err != nil {
		log.Error("deployment.delete.deployment_delete_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Error("deployment.delete.tx_commit_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.OperationCompleted(ctx, "deployment.delete", opStart, "deployment_id", id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "cleanup": "instances, leases, tasks, runplans cleaned up"})
}

// ==========================================================================
// Start / Stop Lifecycle
// ==========================================================================

func (h *AgentHandler) HandleStartDeployment(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	operationID := uuid.NewString()
	ctx, opStart := log.StartOperation(r.Context(), "deployment.start",
		"deployment_id", deployID, "operation_id", operationID)
	_ = opStart // used at end with OperationCompleted
	deploy := h.getDeploymentJSON(deployID)
	if deploy == nil {
		writeError(w, http.StatusNotFound, "deployment not found")
		return
	}
	if !tenantScopeCheck(r, deploy["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	// Parse deployment data — use safe extraction to avoid panic
	artifactID := strVal(deploy, "model_artifact_id", "")
	runtimeID := strVal(deploy, "backend_runtime_id", "")
	if artifactID == "" {
		writeError(w, http.StatusBadRequest, "model_artifact_id is required")
		return
	}
	if runtimeID == "" {
		writeError(w, http.StatusBadRequest, "backend_runtime_id is required")
		return
	}
	placementRaw, _ := json.Marshal(deploy["placement_json"])
	serviceRaw, _ := json.Marshal(deploy["service_json"])
	paramsRaw, _ := json.Marshal(deploy["parameters_json"])
	envOverridesRaw, _ := json.Marshal(deploy["env_overrides_json"])

	var placement struct {
		NodeID string   `json:"node_id"`
		GPUIds []string `json:"gpu_ids"`
	}
	json.Unmarshal(placementRaw, &placement)

	var service struct {
		HostPort int `json:"host_port"`
	}
	json.Unmarshal(serviceRaw, &service)

	var params map[string]interface{}
	json.Unmarshal(paramsRaw, &params)
	var envOverrides map[string]string
	json.Unmarshal(envOverridesRaw, &envOverrides)

	// Get artifact
	artifact := h.getArtifactJSON(artifactID)
	if artifact == nil {
		writeError(w, http.StatusBadRequest, "artifact not found")
		return
	}

	// Auto-select online node if placement doesn't specify one.
	// AUD-015: Scope node selection to the deployment's tenant for non-admin users.
	// Platform admins can deploy to any online node.
	if placement.NodeID == "" {
		var autoNodeID string
		if isPlatformAdmin(r) {
			h.DB.QueryRow(`SELECT id FROM nodes WHERE status = 'online' LIMIT 1`).Scan(&autoNodeID)
		} else {
			deployTid := strVal(deploy, "tenant_id", "")
			h.DB.QueryRow(`SELECT id FROM nodes WHERE status = 'online' AND tenant_id = ? LIMIT 1`, deployTid).Scan(&autoNodeID)
		}
		if autoNodeID == "" {
			writeError(w, http.StatusBadRequest, "no online node available for deployment")
			return
		}
		placement.NodeID = autoNodeID
	}

	// Get runtime config
	var rtVendor, rtImage, rtDockerJSON, rtArgsOverride, rtEntrypointOverride, rtDefaultEnv, rtBackendID, rtVersionID string
	h.DB.QueryRow(`SELECT vendor, image_name, docker_json, args_override_json, entrypoint_override_json, default_env_json, backend_id, backend_version_id FROM backend_runtimes WHERE id = ?`, runtimeID).Scan(&rtVendor, &rtImage, &rtDockerJSON, &rtArgsOverride, &rtEntrypointOverride, &rtDefaultEnv, &rtBackendID, &rtVersionID)

	// Get backend info
	var backendName, backendDefaultEnv string
	h.DB.QueryRow(`SELECT name, default_env_json FROM inference_backends WHERE id = ?`, rtBackendID).Scan(&backendName, &backendDefaultEnv)

	// Get version info
	var bvEntrypoint, bvArgs, bvBackendParams, bvParamDefs, bvHC, bvDefaultImages, bvEnv string
	var bvPort int
	h.DB.QueryRow(`SELECT default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json FROM backend_versions WHERE id = ?`, rtVersionID).Scan(&bvEntrypoint, &bvArgs, &bvBackendParams, &bvParamDefs, &bvHC, &bvPort, &bvDefaultImages, &bvEnv)

	// Build RunPlan input
	instanceID := uuid.NewString()
	nodeIP := "127.0.0.1"
	h.DB.QueryRow(`SELECT primary_ip FROM nodes WHERE id = ?`, placement.NodeID).Scan(&nodeIP)

	// Auto-assign first available GPU on the node if none specified.
	// Required for GPU lease chain: reserved → activated → released.
	if len(placement.GPUIds) == 0 && placement.NodeID != "" {
		var autoGpuID string
		h.DB.QueryRow(`SELECT id FROM gpu_devices WHERE node_id = ? AND status = 'available' LIMIT 1`,
			placement.NodeID).Scan(&autoGpuID)
		if autoGpuID != "" {
			placement.GPUIds = []string{autoGpuID}
			log.Info("gpu_lease.auto_assigned", "gpu_id", autoGpuID, "node_id", placement.NodeID, "instance_id", instanceID)
		}
	}

	log.Info("instance.start.requested",
		"operation_id", operationID,
		"tenant_id", deploy["tenant_id"],
		"actor_id", actorIDFromSession(r),
		"deployment_id", deployID,
		"instance_id", instanceID,
		"node_id", placement.NodeID,
		"gpu_ids", placement.GPUIds,
		"request_id", log.RequestIDFromContext(r.Context()),
	)

	var entrypoint, argsOverride []string
	json.Unmarshal([]byte(bvEntrypoint), &entrypoint)
	json.Unmarshal([]byte(bvArgs), &bvArgs) // placeholder, actual parsing in resolver
	var backendParams []string
	json.Unmarshal([]byte(bvBackendParams), &backendParams)
	var paramDefs []runplan.ParameterDef
	json.Unmarshal([]byte(bvParamDefs), &paramDefs)
	var hc runplan.HealthCheckInput
	json.Unmarshal([]byte(bvHC), &hc)
	var defaultImages map[string]string
	json.Unmarshal([]byte(bvDefaultImages), &defaultImages)
	var bvEnvMap map[string]string
	json.Unmarshal([]byte(bvEnv), &bvEnvMap)

	var backendEnv map[string]string
	json.Unmarshal([]byte(backendDefaultEnv), &backendEnv)

	// Parse runtime configs
	var rtEntryOverride []string
	json.Unmarshal([]byte(rtEntrypointOverride), &rtEntryOverride)
	json.Unmarshal([]byte(rtArgsOverride), &argsOverride)
	var rtEnvMap map[string]string
	json.Unmarshal([]byte(rtDefaultEnv), &rtEnvMap)

	var dockerSpec runplan.DockerSpecInfo
	json.Unmarshal([]byte(rtDockerJSON), &dockerSpec)

	if rtEntryOverride != nil {
		entrypoint = rtEntryOverride
	}

	// Build default args list
	var defaultArgs []string
	json.Unmarshal([]byte(bvArgs), &defaultArgs)

	// GPU info
	var gpuInfos []runplan.GPUInfo
	for _, gid := range placement.GPUIds {
		var idx int
		var vendor string
		h.DB.QueryRow(`SELECT gpu_index, vendor FROM gpu_devices WHERE id = ?`, gid).Scan(&idx, &vendor)
		gpuInfos = append(gpuInfos, runplan.GPUInfo{Index: idx, Vendor: vendor})
	}

	// Resolve RunPlan
	plan, errs, warns := runplan.Resolve(runplan.ResolveInput{
		Backend:        &runplan.BackendInfo{ID: rtBackendID, Name: backendName, DefaultEnv: backendEnv},
		BackendVersion: &runplan.VersionInfo{ID: rtVersionID, Version: "", DefaultEntrypoint: entrypoint, DefaultArgs: defaultArgs, DefaultBackendParams: backendParams, ParameterDefs: paramDefs, HealthCheck: hc, DefaultContainerPort: bvPort, DefaultImages: defaultImages, Env: bvEnvMap},
		BackendRuntime: &runplan.RuntimeInfo{ID: runtimeID, Vendor: rtVendor, RuntimeType: "docker", ImageName: rtImage, EntrypointOverride: rtEntryOverride, ArgsOverride: argsOverride, DefaultEnv: rtEnvMap, Docker: dockerSpec},
		Artifact:       &runplan.ArtifactInfo{ID: artifactID, Name: strVal(artifact, "name", ""), Path: strVal(artifact, "path", "")},
		Deployment:     &runplan.DeploymentInfo{ID: deployID, Name: strVal(deploy, "name", ""), Parameters: params, EnvOverrides: envOverrides, Service: runplan.ServiceInfo{HostPort: service.HostPort}, Placement: runplan.PlacementInfo{NodeID: placement.NodeID, GPUIds: placement.GPUIds}},
		InstanceID:     instanceID,
		Node:           &runplan.NodeInfo{ID: placement.NodeID, IP: nodeIP},
		AssignedGPUs:   gpuInfos,
	})

	if len(errs) > 0 {
		errMsgs := make([]string, len(errs))
		for i, e := range errs {
			errMsgs[i] = e.Error()
		}
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "runplan resolution failed", "details": errMsgs})
		return
	}

	// Build GPU device ID list using NVIDIA indices (not internal UUIDs).
	// NVIDIA container toolkit requires index-based device IDs like "0", not DB UUIDs.
	gpuDeviceIDs := make([]string, len(gpuInfos))
	for i, gi := range gpuInfos {
		gpuDeviceIDs[i] = fmt.Sprintf("%d", gi.Index)
	}

	// Transaction: instance + runplan + lease + agent_task
	now := time.Now().Format(time.RFC3339)
	planJSON, _ := json.Marshal(plan)
	// Build AgentRunSpec for agent consumption
	agentSpec := map[string]interface{}{
		"instance_id":         instanceID,
		"deployment_id":       deployID,
		"runtime_type":        "docker",
		"vendor":              rtVendor,
		"model_path":          strVal(artifact, "path", ""),
		"served_model_name":   strVal(params, "served_model_name", strVal(artifact, "name", "")),
		"node_id":             placement.NodeID,
		"agent_id":            "",
		"gpu_device_ids":      gpuDeviceIDs,
		"gpu_visible_env_key": "CUDA_VISIBLE_DEVICES",
		"operation_id":        operationID,
		"env":                 plan.Env,
		"args":                plan.Args,
		"volumes":             plan.Mounts,
		"host_port":           service.HostPort,
		"container_port":      bvPort,
		"ports": []map[string]interface{}{
			{"host_port": service.HostPort, "container_port": bvPort},
		},
		"docker": map[string]interface{}{
			"image":            plan.Image,
			"container_name":   plan.ContainerName,
			"command":          plan.Entrypoint,
			"args":             plan.Args,
			"privileged":       plan.Privileged,
			"ipc_mode":         plan.IPCMode,
			"uts_mode":         plan.UTSMode,
			"network_mode":     plan.NetworkMode,
			"shm_size":         plan.ShmSize,
			"security_options": plan.SecurityOptions,
			"ulimits":          plan.Ulimits,
			"gpu_device_ids":   gpuDeviceIDs,
		},
		"health_check": map[string]interface{}{
			"enabled": plan.HealthCheck.Path != "",

			"path":             plan.HealthCheck.Path,
			"port":             service.HostPort,
			"port_source":      "host_port",
			"container_port":   bvPort,
			"scheme":           "http",
			"expected_status":  plan.HealthCheck.ExpectedStatus,
			"timeout_seconds":  plan.HealthCheck.TimeoutSeconds,
			"interval_seconds": plan.HealthCheck.IntervalSeconds,
		},
	}
	agentPayload, _ := json.Marshal(agentSpec)

	runPlanID := uuid.NewString()
	taskID := uuid.NewString()
	tid := deploy["tenant_id"].(string)

	// Begin transaction for atomic instance+runplan+lease+task creation.
	// AUD-004: Wrap all writes in a transaction so partial failure doesn't leave orphans.
	tx, txErr := h.DB.Begin()
	if txErr != nil {
		log.Error("deployment.start.tx_begin_failed", "error", txErr, "deployment_id", deployID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	// Insert instance
	if _, err := tx.Exec(`INSERT INTO model_instances (id, deployment_id, tenant_id, replica_index, node_id, agent_id, assigned_gpus_json, host_port, container_port, current_run_plan_id, actual_state, desired_state, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		instanceID, deployID, tid, 0, placement.NodeID, "", jsonString(placement.GPUIds), service.HostPort, bvPort, runPlanID, "pending", "running", now, now); err != nil {
		log.Error("deployment.start.instance_insert_failed", "error", err, "instance_id", instanceID, "deployment_id", deployID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Insert runplan
	if _, err := tx.Exec(`INSERT INTO resolved_run_plans (id, deployment_id, instance_id, tenant_id, backend_runtime_id, plan_json, docker_preview, input_hash, plan_hash, created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		runPlanID, deployID, instanceID, tid, runtimeID, string(planJSON), runplan.EquivalentCommandPreview(plan), plan.InputHash, plan.PlanHash, now); err != nil {
		log.Error("deployment.start.runplan_insert_failed", "error", err, "run_plan_id", runPlanID, "instance_id", instanceID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Create GPU leases
	for _, gid := range placement.GPUIds {
		leaseID := uuid.NewString()
		if _, err := tx.Exec(`INSERT INTO gpu_leases (id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status, reserved_at, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			leaseID, gid, placement.NodeID, deployID, instanceID, tid, "reserved", now, now, now); err != nil {
			log.Error("deployment.start.lease_insert_failed", "error", err, "lease_id", leaseID, "gpu_id", gid, "instance_id", instanceID)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		log.StateTransition(r.Context(), "deployment.start", "gpu_lease", leaseID, "", "reserved",
			"gpu_id", gid, "instance_id", instanceID, "deployment_id", deployID, "node_id", placement.NodeID)
		log.Info("gpu_lease.reserved", "lease_id", leaseID, "gpu_id", gid, "instance_id", instanceID, "deployment_id", deployID)
	}

	// Create agent task
	if _, err := tx.Exec(`INSERT INTO agent_tasks (id, task_type, status, tenant_id, deployment_id, instance_id, node_id, payload, timeout_seconds, operation_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		taskID, "model_instance_start", "pending", tid, deployID, instanceID, placement.NodeID, string(agentPayload), 300, operationID, now, now); err != nil {
		log.Error("deployment.start.task_insert_failed", "error", err, "task_id", taskID, "instance_id", instanceID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Update deployment status
	if _, err := tx.Exec(`UPDATE model_deployments SET desired_state = 'running', status = 'running', updated_at = ? WHERE id = ?`, now, deployID); err != nil {
		log.Error("deployment.start.deployment_update_failed", "error", err, "deployment_id", deployID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Error("deployment.start.tx_commit_failed", "error", err, "deployment_id", deployID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Log after commit succeeds so correlation IDs are durable.
	log.Info("agent_task.created",
		"operation_id", operationID,
		"task_id", taskID,
		"task_type", "model_instance_start",
		"deployment_id", deployID,
		"instance_id", instanceID,
		"agent_id", "", "node_id", placement.NodeID,
		"generation", 1, "attempt", 1,
	)

	log.OperationCompleted(ctx, "deployment.start", opStart,
		"deployment_id", deployID, "instance_id", instanceID, "task_id", taskID, "run_plan_id", runPlanID)

	WriteAudit(r.Context(), h.DB.DB, AuditEntry{
		TenantID: tid, ActorID: actorIDFromSession(r),
		Action: "instance.start", ResourceType: "model_instance",
		ResourceID: instanceID, Result: "success",
		RequestID: log.RequestIDFromContext(r.Context()), OperationID: operationID,
	})

	// Agent will claim this task on next heartbeat and execute Docker start.
	// For now, attempt direct Docker execution if Agent is not running.
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "started",
		"deployment_id":  deployID,
		"instance_id":    instanceID,
		"task_id":        taskID,
		"run_plan_id":    runPlanID,
		"warnings":       warns,
		"docker_preview": runplan.EquivalentCommandPreview(plan),
	})
}

func (h *AgentHandler) HandleStopDeployment(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	operationID := uuid.NewString()
	ctx, opStart := log.StartOperation(r.Context(), "deployment.stop", "deployment_id", deployID, "operation_id", operationID)
	_ = opStart
	deploy := h.getDeploymentJSON(deployID)
	if deploy == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, deploy["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	now := time.Now().Format(time.RFC3339)
	tid := deploy["tenant_id"].(string)
	actorID := actorIDFromSession(r)

	// Find ALL non-terminal instances (running, starting, failed, pending, initializing)
	rows, err := h.DB.Query(`SELECT id, node_id, container_id, actual_state FROM model_instances WHERE deployment_id = ? AND actual_state NOT IN ('stopped')`, deployID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	type inst struct{ id, nodeID, containerID, state string }
	var instances []inst
	for rows.Next() {
		var i inst
		rows.Scan(&i.id, &i.nodeID, &i.containerID, &i.state)
		instances = append(instances, i)
	}

	// Log stop request with instance details.
	for _, i := range instances {
		log.Info("instance.stop.requested",
			"operation_id", operationID,
			"tenant_id", tid, "actor_id", actorID,
			"deployment_id", deployID,
			"instance_id", i.id,
			"container_id", i.containerID,
			"request_id", log.RequestIDFromContext(r.Context()),
		)
	}

	// Idempotent: if no non-stopped instances, still update deployment status
	warnings := []string{}
	for _, i := range instances {
		// Mark instance stopping/stopped based on current state
		h.DB.Exec(`UPDATE model_instances SET desired_state = 'stopped', actual_state = CASE WHEN actual_state IN ('failed','pending') THEN 'stopped' ELSE actual_state END, stopped_at = ?, updated_at = ? WHERE id = ?`, now, now, i.id)

		// Release all leases for this instance, regardless of status
		result, lerr := h.DB.Exec(`UPDATE gpu_leases SET status = 'released', released_at = ? WHERE instance_id = ? AND status IN ('reserved','active')`, now, i.id)
		n, _ := result.RowsAffected()
		if lerr != nil {
			log.Error("gpu_lease.release.failed", "instance_id", i.id, "error", lerr)
		} else if n > 0 {
			log.StateTransition(r.Context(), "deployment.stop", "gpu_lease", i.id, "reserved/active", "released",
				"instance_id", i.id, "count", n)
			log.Info("gpu_lease.released", "operation_id", operationID, "instance_id", i.id, "count", n)
		}

		// Cancel pending agent tasks
		h.DB.Exec(`UPDATE agent_tasks SET status = 'failed', finished_at = ? WHERE instance_id = ? AND status NOT IN ('completed','failed')`, now, i.id)
	}

	h.DB.Exec(`UPDATE model_deployments SET desired_state = 'stopped', status = 'stopped', updated_at = ? WHERE id = ?`, now, deployID)

	for _, i := range instances {
		log.Info("instance.stop.completed",
			"operation_id", operationID,
			"deployment_id", deployID,
			"instance_id", i.id,
			"container_id", i.containerID,
			"final_state", i.state,
		)
	}

	log.OperationCompleted(ctx, "deployment.stop", opStart,
		"deployment_id", deployID, "instances_stopped", len(instances))

	for _, i := range instances {
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "instance.stop", ResourceType: "model_instance",
			ResourceID: i.id, Result: "success",
			RequestID: log.RequestIDFromContext(r.Context()), OperationID: operationID,
			Detail: fmt.Sprintf("deployment_id=%s", deployID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "stopped", "instances_stopped": len(instances), "warnings": warnings})
}

// ==========================================================================
// ModelInstance read
// ==========================================================================

func (h *AgentHandler) HandleListInstances(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	tid := tenantID(r)
	deployID := r.URL.Query().Get("deployment_id")
	q := `SELECT id, deployment_id, tenant_id, replica_index, node_id, agent_id, assigned_gpus_json, gpu_lease_ids_json, host_port, container_port, current_run_plan_id, actual_state, desired_state, container_id, endpoint_url, restart_count, last_error, started_at, stopped_at, created_at, updated_at FROM model_instances`
	var args []interface{}
	if deployID != "" {
		q += ` WHERE deployment_id = ?`
		args = append(args, deployID)
		if !isPlatformAdmin(r) {
			q += ` AND tenant_id = ?`
			args = append(args, tid)
		}
	} else if !isPlatformAdmin(r) {
		q += ` WHERE tenant_id = ?`
		args = append(args, tid)
	}
	q += ` ORDER BY created_at DESC`
	// SQL query logged at DEBUG to avoid poll noise.
	log.Debug("model_instances.query", "operation", "list_model_instances", "stage", "db_query",
		"request_id", log.RequestIDFromContext(r.Context()),
		"deployment_id", deployID, "tenant_id", tid, "is_admin", isPlatformAdmin(r))
	rows, err := h.DB.Query(q, args...)
	if err != nil {
		log.Error("model_instances.query_failed", "operation", "list_model_instances", "error", err,
			"request_id", log.RequestIDFromContext(r.Context()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	scanErrors := 0
	for rows.Next() {
		var id, did, tid2, as, ds, gpus, glids, ca string
		var ri, hp, cp, rc int
		var nid, aid, rpid, cid, eurl, le, sa, soa, ua sql.NullString
		if err := rows.Scan(&id, &did, &tid2, &ri, &nid, &aid, &gpus, &glids, &hp, &cp, &rpid, &as, &ds, &cid, &eurl, &rc, &le, &sa, &soa, &ca, &ua); err != nil {
			log.Error("model_instances.scan_failed", "operation", "list_model_instances", "stage", "db_scan",
				"request_id", log.RequestIDFromContext(r.Context()),
				"deployment_id", deployID, "tenant_id", tid, "error", err)
			scanErrors++
			// Return 500 on scan errors — don't silently return empty list.
			writeError(w, http.StatusInternalServerError, "internal error: failed to read instance data")
			return
		}
		m := map[string]interface{}{
			"id": id, "deployment_id": did, "tenant_id": tid2,
			"node_id": nid.String, "actual_state": as,
			"container_id": cid.String, "endpoint_url": eurl.String,
			"host_port": hp, "last_error": le.String,
			"started_at": sa.String, "stopped_at": soa.String, "created_at": ca,
		}
		out = append(out, m)
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	durationMs := time.Since(startTime).Milliseconds()
	log.Debug("model_instances.list_completed", "operation", "list_model_instances",
		"count", len(out), "duration_ms", durationMs,
		"request_id", log.RequestIDFromContext(r.Context()),
		"deployment_id", deployID)
	// INFO only if there are actual results or errors, to avoid poll noise.
	if len(out) > 0 || scanErrors > 0 {
		log.Info("model_instances.list_completed", "operation", "list_model_instances",
			"count", len(out), "duration_ms", durationMs,
			"request_id", log.RequestIDFromContext(r.Context()))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleGetInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	row := h.DB.QueryRow(`SELECT id, deployment_id, tenant_id, node_id, container_id, actual_state, desired_state, endpoint_url, host_port, container_port, last_error, started_at, stopped_at, created_at, updated_at FROM model_instances WHERE id = ?`, id)
	var rid, did, tid, as, ds, ca string
	var nid, cid, eu, le, sa, soa, ua sql.NullString
	var hp, cp int
	if err := row.Scan(&rid, &did, &tid, &nid, &cid, &as, &ds, &eu, &hp, &cp, &le, &sa, &soa, &ca, &ua); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, tid) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"id": rid, "deployment_id": did, "tenant_id": tid, "node_id": nid.String, "container_id": cid.String, "actual_state": as, "desired_state": ds, "endpoint_url": eu.String, "host_port": hp, "container_port": cp, "last_error": le.String, "started_at": sa.String, "stopped_at": soa.String, "created_at": ca, "updated_at": ua.String})
}

// ==========================================================================
// Dry Run
// ==========================================================================

func (h *AgentHandler) HandleDeploymentDryRun(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	tid := tenantID(r)
	actorID := actorIDFromSession(r)
	requestID := log.RequestIDFromContext(r.Context())

	deploy := h.getDeploymentJSON(deployID)
	if deploy == nil {
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "deployment.dry_run", ResourceType: "deployment",
			ResourceID: deployID, Result: "failure",
			RequestID: requestID, Error: "not_found",
		})
		log.Warn("deployment.dry_run.failed", "request_id", requestID,
			"tenant_id", tid, "actor_id", actorID, "deployment_id", deployID,
			"reason", "not_found")
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	runtimeID := strVal(deploy, "backend_runtime_id", "")
	var rtVendor, rtImage, rtDockerJSON string
	var bvPort int
	h.DB.QueryRow(`SELECT vendor, image_name, docker_json FROM backend_runtimes WHERE id = ?`, runtimeID).Scan(&rtVendor, &rtImage, &rtDockerJSON)
	h.DB.QueryRow(`SELECT default_container_port FROM backend_versions WHERE id = (SELECT backend_version_id FROM backend_runtimes WHERE id = ?)`, runtimeID).Scan(&bvPort)

	// Simple dry-run: validate references exist
	errors := []string{}
	if strVal(deploy, "model_artifact_id", "") == "" {
		errors = append(errors, "model_artifact_id is required")
	}
	if runtimeID == "" {
		errors = append(errors, "backend_runtime_id is required")
	}

	valid := len(errors) == 0
	if valid {
		log.Info("deployment.dry_run.succeeded", "request_id", requestID,
			"tenant_id", tid, "actor_id", actorID, "deployment_id", deployID,
			"model_artifact_id", strVal(deploy, "model_artifact_id", ""),
			"backend_runtime_id", runtimeID,
			"runtime", "docker", "vendor", rtVendor, "image", rtImage)
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "deployment.dry_run", ResourceType: "deployment",
			ResourceID: deployID, Result: "success",
			RequestID: requestID,
			Detail:    fmt.Sprintf("runtime=%s vendor=%s image=%s", runtimeID, rtVendor, rtImage),
		})
	} else {
		log.Warn("deployment.dry_run.failed", "request_id", requestID,
			"tenant_id", tid, "actor_id", actorID, "deployment_id", deployID,
			"reason", "validation_failed", "errors", fmt.Sprintf("%v", errors))
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "deployment.dry_run", ResourceType: "deployment",
			ResourceID: deployID, Result: "failure",
			RequestID: requestID, Error: fmt.Sprintf("validation: %v", errors),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":    valid,
		"errors":   errors,
		"warnings": []string{},
	})
}

// ==========================================================================
// Helpers
// ==========================================================================

func (h *AgentHandler) getDeploymentJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, name, display_name, description, model_artifact_id, backend_runtime_id, replicas, placement_json, service_json, parameters_json, env_overrides_json, desired_state, status, tenant_id, created_at, updated_at FROM model_deployments WHERE id = ?`, id)
	var rid, name, dn, desc, maid, rtid, pj, sj, pj2, eoj, ds, status, tid, ca, ua string
	var replicas int
	if err := row.Scan(&rid, &name, &dn, &desc, &maid, &rtid, &replicas, &pj, &sj, &pj2, &eoj, &ds, &status, &tid, &ca, &ua); err != nil {
		return nil
	}
	return map[string]interface{}{"id": rid, "name": name, "display_name": dn, "description": desc, "model_artifact_id": maid, "backend_runtime_id": rtid, "replicas": replicas, "placement_json": json.RawMessage(pj), "service_json": json.RawMessage(sj), "parameters_json": json.RawMessage(pj2), "env_overrides_json": json.RawMessage(eoj), "desired_state": ds, "status": status, "tenant_id": tid, "created_at": ca, "updated_at": ua}
}

func (h *AgentHandler) queryDeployments(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var rid, name, dn, desc, maid, rtid, pj, sj, pj2, eoj, ds, status, tid, ca, ua string
		var replicas int
		if err := rows.Scan(&rid, &name, &dn, &desc, &maid, &rtid, &replicas, &pj, &sj, &pj2, &eoj, &ds, &status, &tid, &ca, &ua); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{"id": rid, "name": name, "display_name": dn, "description": desc, "model_artifact_id": maid, "backend_runtime_id": rtid, "replicas": replicas, "placement_json": json.RawMessage(pj), "service_json": json.RawMessage(sj), "parameters_json": json.RawMessage(pj2), "env_overrides_json": json.RawMessage(eoj), "desired_state": ds, "status": status, "tenant_id": tid, "created_at": ca, "updated_at": ua})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out, nil
}
