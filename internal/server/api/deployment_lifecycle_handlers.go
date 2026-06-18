package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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
	if _, err := tx.Exec(`DELETE FROM agent_tasks WHERE deployment_id = ?`, id); err != nil {
		log.Error("deployment.delete.task_delete_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Delete resolved_run_plans for this deployment
	if _, err := tx.Exec(`DELETE FROM resolved_run_plans WHERE deployment_id = ?`, id); err != nil {
		log.Error("deployment.delete.runplan_delete_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if _, err := tx.Exec(`DELETE FROM run_plan_groups WHERE deployment_plan_id = ?`, id); err != nil {
		log.Error("deployment.delete.run_plan_group_delete_failed", "error", err, "deployment_id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if _, err := tx.Exec(`DELETE FROM gpu_leases WHERE deployment_id = ?`, id); err != nil {
		log.Error("deployment.delete.lease_delete_failed", "error", err, "deployment_id", id)
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

// PreflightError is a structured preflight validation error with a stable
// code for frontend i18n mapping and a human-readable message for logs.
type PreflightError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// preflightResult holds the output of the shared pre-start validation and
// resolution logic used by both dry-run and real start.
type preflightResult struct {
	deploy     map[string]interface{}
	artifactID string
	artifact   map[string]interface{}
	runtimeID  string
	nbrSnapshot string // config_snapshot_json from NodeBackendRuntime
	nbrImageRef string // image_ref from NodeBackendRuntime
	placement  struct {
		NodeID string
		GPUIds []string
	}
	service struct {
		HostPort int `json:"host_port"`
	}
	params            map[string]interface{}
	envOverrides      map[string]string
	rtVendor          string
	rtImage           string
	rtDockerJSON      string
	rtArgsOverride    string
	rtEntryOverride   string
	rtDefaultEnv      string
	rtBackendID       string
	rtVersionID       string
	rtModelMount      string
	backendName       string
	backendDefaultEnv string
	bvEntrypoint      string
	bvArgs            string
	bvBackendParams   string
	bvParamDefs       string
	bvHC              string
	bvPort            int
	bvDefaultImages   string
	bvEnv             string
	nodeIP            string
	gpuInfos          []runplan.GPUInfo
	nodeRuntimeID     string
	locationID        string
	modelRoot         string
	relativePath      string
	absolutePath      string
	plan              *runplan.ResolvedRunPlan
	errs              []PreflightError
	warns             []string
	commandPreview    string
}

// addErr appends a structured PreflightError to the result.
func (pf *preflightResult) addErr(code, message string, ctx map[string]interface{}) {
	pf.errs = append(pf.errs, PreflightError{Code: code, Message: message, Context: ctx})
}

// preflightDeployment performs all pre-start validation and resolution steps
// shared between dry-run and real start. It does NOT create any database records.
// BRR-RV-001: Extracted from HandleStartDeployment so dry-run can use real resolver.
func (h *AgentHandler) preflightDeployment(deployID string, r *http.Request) *preflightResult {
	pf := &preflightResult{}

	deploy := h.getDeploymentJSON(deployID)
	if deploy == nil {
		pf.addErr("unknown", "deployment not found", nil)
		return pf
	}
	if !tenantScopeCheck(r, deploy["tenant_id"].(string)) {
		pf.addErr("unknown", "deployment not found", nil)
		return pf
	}
	pf.deploy = deploy

	pf.artifactID = strVal(deploy, "model_artifact_id", "")
	pf.runtimeID = strVal(deploy, "backend_runtime_id", "")
	if pf.artifactID == "" {
		pf.addErr("unknown", "model_artifact_id is required", map[string]interface{}{"artifact_id": pf.artifactID})
		return pf
	}
	if pf.runtimeID == "" {
		pf.addErr("unknown", "backend_runtime_id is required", map[string]interface{}{"runtime_id": pf.runtimeID})
		return pf
	}

	// Parse placement/service JSON.
	placementRaw, _ := json.Marshal(deploy["placement_json"])
	serviceRaw, _ := json.Marshal(deploy["service_json"])
	paramsRaw, _ := json.Marshal(deploy["parameters_json"])
	envOverridesRaw, _ := json.Marshal(deploy["env_overrides_json"])
	json.Unmarshal(placementRaw, &pf.placement)
	json.Unmarshal(serviceRaw, &pf.service)
	json.Unmarshal(paramsRaw, &pf.params)
	json.Unmarshal(envOverridesRaw, &pf.envOverrides)

	// Validate artifact exists.
	artifact := h.getArtifactJSON(pf.artifactID)
	if artifact == nil {
		pf.addErr("model_location_missing", "model artifact not found", map[string]interface{}{"artifact_id": pf.artifactID})
		return pf
	}
	pf.artifact = artifact

	// Auto-select online node if placement doesn't specify one.
	if pf.placement.NodeID == "" {
		var autoNodeID string
		if isPlatformAdmin(r) {
			h.DB.QueryRow(`SELECT id FROM nodes WHERE status = 'online' LIMIT 1`).Scan(&autoNodeID)
		} else {
			deployTid := strVal(deploy, "tenant_id", "")
			h.DB.QueryRow(`SELECT id FROM nodes WHERE status = 'online' AND tenant_id = ? LIMIT 1`, deployTid).Scan(&autoNodeID)
		}
		if autoNodeID == "" {
			pf.addErr("node_offline", "no online node available for deployment", nil)
			return pf
		}
		pf.placement.NodeID = autoNodeID
	}

	// Fetch runtime chain: backend_runtime → inference_backend → backend_version.
	h.DB.QueryRow(`SELECT vendor, image_name, docker_json, args_override_json, entrypoint_override_json, default_env_json, backend_id, backend_version_id, model_mount_json FROM backend_runtimes WHERE id = ?`, pf.runtimeID).Scan(&pf.rtVendor, &pf.rtImage, &pf.rtDockerJSON, &pf.rtArgsOverride, &pf.rtEntryOverride, &pf.rtDefaultEnv, &pf.rtBackendID, &pf.rtVersionID, &pf.rtModelMount)
	h.DB.QueryRow(`SELECT name, default_env_json FROM inference_backends WHERE id = ?`, pf.rtBackendID).Scan(&pf.backendName, &pf.backendDefaultEnv)
	h.DB.QueryRow(`SELECT default_entrypoint_json, default_args_json, default_backend_params_json, parameter_defs_json, health_check_json, default_container_port, default_images_json, env_json FROM backend_versions WHERE id = ?`, pf.rtVersionID).Scan(&pf.bvEntrypoint, &pf.bvArgs, &pf.bvBackendParams, &pf.bvParamDefs, &pf.bvHC, &pf.bvPort, &pf.bvDefaultImages, &pf.bvEnv)

	// Fetch node IP.
	pf.nodeIP = "127.0.0.1"
	h.DB.QueryRow(`SELECT primary_ip FROM nodes WHERE id = ?`, pf.placement.NodeID).Scan(&pf.nodeIP)

	// Auto-assign first available GPU on the node if none specified.
	if len(pf.placement.GPUIds) == 0 && pf.placement.NodeID != "" {
		var autoGpuID string
		h.DB.QueryRow(`SELECT id FROM gpu_devices WHERE node_id = ? AND status = 'available' LIMIT 1`,
			pf.placement.NodeID).Scan(&autoGpuID)
		if autoGpuID != "" {
			pf.placement.GPUIds = []string{autoGpuID}
		}
	}

	// Validate NodeBackendRuntime readiness and read snapshot + image_ref.
	var nodeRuntimeStatus string
	h.DB.QueryRow(`SELECT id, status, COALESCE(config_snapshot_json,'{}'), COALESCE(image_ref,'') FROM node_backend_runtimes WHERE node_id = ? AND backend_runtime_id = ?`, pf.placement.NodeID, pf.runtimeID).Scan(&pf.nodeRuntimeID, &nodeRuntimeStatus, &pf.nbrSnapshot, &pf.nbrImageRef)
	if pf.nodeRuntimeID == "" || nodeRuntimeStatus != "ready" {
		pf.addErr("node_backend_runtime_not_ready", fmt.Sprintf("node backend runtime is not ready (status=%s)", nodeRuntimeStatus), map[string]interface{}{"node_id": pf.placement.NodeID, "runtime_id": pf.runtimeID, "node_runtime_id": pf.nodeRuntimeID, "nbr_status": nodeRuntimeStatus})
		return pf
	}

	// Validate ModelLocation.
	var verificationStatus, matchStatus string
	h.DB.QueryRow(`SELECT id, model_root, relative_path, absolute_path, verification_status, match_status
		FROM model_locations
		WHERE model_artifact_id = ? AND node_id = ? AND verification_status IN ('verified','warning','manually_accepted') AND match_status IN ('exact_match','probable_match','manual_attested')
		ORDER BY updated_at DESC LIMIT 1`, pf.artifactID, pf.placement.NodeID).Scan(&pf.locationID, &pf.modelRoot, &pf.relativePath, &pf.absolutePath, &verificationStatus, &matchStatus)
	if pf.locationID == "" {
		pf.addErr("model_location_missing", fmt.Sprintf("model location is not available on target node %s for artifact %s", pf.placement.NodeID, pf.artifactID), map[string]interface{}{"node_id": pf.placement.NodeID, "artifact_id": pf.artifactID})
		return pf
	}
	_ = verificationStatus
	_ = matchStatus

	// Fetch GPU info.
	for _, gid := range pf.placement.GPUIds {
		var idx int
		var vendor string
		h.DB.QueryRow(`SELECT gpu_index, vendor FROM gpu_devices WHERE id = ?`, gid).Scan(&idx, &vendor)
		pf.gpuInfos = append(pf.gpuInfos, runplan.GPUInfo{Index: idx, Vendor: vendor})
	}

	// Parse overlay JSONs for resolver input.
	var entrypoint, argsOverride []string
	json.Unmarshal([]byte(pf.bvEntrypoint), &entrypoint)
	var backendParams []string
	json.Unmarshal([]byte(pf.bvBackendParams), &backendParams)
	var paramDefs []runplan.ParameterDef
	json.Unmarshal([]byte(pf.bvParamDefs), &paramDefs)
	var hc runplan.HealthCheckInput
	json.Unmarshal([]byte(pf.bvHC), &hc)
	var defaultImages map[string]string
	json.Unmarshal([]byte(pf.bvDefaultImages), &defaultImages)
	var bvEnvMap map[string]string
	json.Unmarshal([]byte(pf.bvEnv), &bvEnvMap)
	var backendEnv map[string]string
	json.Unmarshal([]byte(pf.backendDefaultEnv), &backendEnv)
	var rtEntryOverride []string
	json.Unmarshal([]byte(pf.rtEntryOverride), &rtEntryOverride)
	json.Unmarshal([]byte(pf.rtArgsOverride), &argsOverride)
	var rtEnvMap map[string]string
	json.Unmarshal([]byte(pf.rtDefaultEnv), &rtEnvMap)
	var dockerSpec runplan.DockerSpecInfo
	json.Unmarshal([]byte(pf.rtDockerJSON), &dockerSpec)
	var modelMount runplan.ModelMountInfo
	json.Unmarshal([]byte(pf.rtModelMount), &modelMount)
	if rtEntryOverride != nil {
		entrypoint = rtEntryOverride
	}
	var defaultArgs []string
	json.Unmarshal([]byte(pf.bvArgs), &defaultArgs)

	// If NodeBackendRuntime has a config snapshot, use it for runtime configuration
	// instead of the live BackendRuntime template. This decouples node runs from
	// future template edits.
	if pf.nbrSnapshot != "" && pf.nbrSnapshot != "{}" {
		var snap map[string]interface{}
		if json.Unmarshal([]byte(pf.nbrSnapshot), &snap) == nil {
			if v, ok := snap["args_override_json"]; ok {
				var snapArgs []string
				if raw, _ := json.Marshal(v); json.Unmarshal(raw, &snapArgs) == nil {
					argsOverride = snapArgs
				}
			}
			if v, ok := snap["entrypoint_override_json"]; ok {
				var snapEntry []string
				if raw, _ := json.Marshal(v); json.Unmarshal(raw, &snapEntry) == nil && len(snapEntry) > 0 {
					rtEntryOverride = snapEntry
					entrypoint = snapEntry
				}
			}
			if v, ok := snap["default_env_json"]; ok {
				var snapEnv map[string]string
				if raw, _ := json.Marshal(v); json.Unmarshal(raw, &snapEnv) == nil {
					rtEnvMap = snapEnv
				}
			}
			if v, ok := snap["docker_json"]; ok {
				var snapDocker runplan.DockerSpecInfo
				if raw, _ := json.Marshal(v); json.Unmarshal(raw, &snapDocker) == nil {
					dockerSpec = snapDocker
				}
			}
			if v, ok := snap["model_mount_json"]; ok {
				var snapMount runplan.ModelMountInfo
				if raw, _ := json.Marshal(v); json.Unmarshal(raw, &snapMount) == nil {
					modelMount = snapMount
				}
			}
			if v, ok := snap["image_name"]; ok {
				if s, ok := v.(string); ok && s != "" {
					pf.rtImage = s
				}
			}
			if v, ok := snap["vendor"]; ok {
				if s, ok := v.(string); ok && s != "" {
					pf.rtVendor = s
				}
			}
		}
	}

	// Build NodeRuntimeOverride from NBR image_ref (node-level image override).
	var nbrOverride *runplan.NodeOverrideInfo
	if pf.nbrImageRef != "" {
		nbrOverride = &runplan.NodeOverrideInfo{
			ImageName: pf.nbrImageRef,
		}
	}

	instanceID := uuid.NewString()

	// Call the real RunPlan resolver with snapshot-based RuntimeInfo.
	plan, resolveErrs, resolveWarns := runplan.Resolve(runplan.ResolveInput{
		Backend:            &runplan.BackendInfo{ID: pf.rtBackendID, Name: pf.backendName, DefaultEnv: backendEnv},
		BackendVersion:     &runplan.VersionInfo{ID: pf.rtVersionID, Version: "", DefaultEntrypoint: entrypoint, DefaultArgs: defaultArgs, DefaultBackendParams: backendParams, ParameterDefs: paramDefs, HealthCheck: hc, DefaultContainerPort: pf.bvPort, DefaultImages: defaultImages, Env: bvEnvMap},
		BackendRuntime:     &runplan.RuntimeInfo{ID: pf.runtimeID, Vendor: pf.rtVendor, RuntimeType: "docker", ImageName: pf.rtImage, EntrypointOverride: rtEntryOverride, ArgsOverride: argsOverride, DefaultEnv: rtEnvMap, Docker: dockerSpec, ModelMount: modelMount},
		NodeRuntimeOverride: nbrOverride,
		Artifact:           &runplan.ArtifactInfo{ID: pf.artifactID, Name: strVal(artifact, "name", ""), Path: pf.absolutePath, ModelRoot: pf.modelRoot, RelativePath: pf.relativePath},
		Deployment:         &runplan.DeploymentInfo{ID: deployID, Name: strVal(deploy, "name", ""), Parameters: pf.params, EnvOverrides: pf.envOverrides, Service: runplan.ServiceInfo{HostPort: pf.service.HostPort}, Placement: runplan.PlacementInfo{NodeID: pf.placement.NodeID, GPUIds: pf.placement.GPUIds}},
		InstanceID:         instanceID,
		Node:               &runplan.NodeInfo{ID: pf.placement.NodeID, IP: pf.nodeIP},
		AssignedGPUs:       pf.gpuInfos,
	})
	for _, e := range resolveErrs {
		pf.addErr("unknown", e.Error(), nil)
	}
	for _, w := range resolveWarns {
		pf.warns = append(pf.warns, w)
	}
	if plan != nil {
		pf.plan = plan
		pf.commandPreview = runplan.EquivalentCommandPreview(plan)
	} else if len(pf.errs) == 0 {
		// Resolver returned nil plan without explicit errors — add a catch-all.
		pf.addErr("unknown", "runplan resolution returned no plan", nil)
	}

	return pf
}

func (h *AgentHandler) HandleStartDeployment(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	operationID := uuid.NewString()
	ctx, opStart := log.StartOperation(r.Context(), "deployment.start",
		"deployment_id", deployID, "operation_id", operationID)
	_ = opStart // used at end with OperationCompleted

	// BRR-RV-001: Shared pre-flight validation via preflightDeployment.
	pfStageStart := time.Now()
	pf := h.preflightDeployment(deployID, r)
	if len(pf.errs) > 0 {
		// Collect structured errors for JSON response.
		var errSummary []string
		for _, e := range pf.errs {
			errSummary = append(errSummary, e.Message)
		}
		log.StageFailed(ctx, "deployment.start", "preflight", pfStageStart, fmt.Errorf("%v", pf.errs),
			"deployment_id", deployID, "errors", pf.errs)
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "preflight validation failed", "errors": pf.errs, "details": errSummary})
		return
	}
	log.StageCompleted(ctx, "deployment.start", "preflight", pfStageStart,
		"deployment_id", deployID, "node_id", pf.placement.NodeID, "duration_ms", log.DurationMs(pfStageStart))

	instanceID := uuid.NewString()
	log.Info("instance.start.requested",
		"operation_id", operationID,
		"tenant_id", pf.deploy["tenant_id"],
		"actor_id", actorIDFromSession(r),
		"deployment_id", deployID,
		"instance_id", instanceID,
		"node_id", pf.placement.NodeID,
		"gpu_ids", pf.placement.GPUIds,
		"request_id", log.RequestIDFromContext(r.Context()),
	)

	// Build GPU device ID list using NVIDIA indices (not internal UUIDs).
	gpuDeviceIDs := make([]string, len(pf.gpuInfos))
	for i, gi := range pf.gpuInfos {
		gpuDeviceIDs[i] = fmt.Sprintf("%d", gi.Index)
	}

	healthTimeoutSeconds := planHealthTimeout2(pf.bvHC)

	// Transaction: instance + runplan + lease + agent_task
	now := time.Now().Format(time.RFC3339)
	planJSON, _ := json.Marshal(pf.plan)
	agentSpec := map[string]interface{}{
		"instance_id":         instanceID,
		"deployment_id":       deployID,
		"runtime_type":        "docker",
		"vendor":              pf.rtVendor,
		"model_path":          pf.absolutePath,
		"served_model_name":   strVal(pf.params, "served_model_name", strVal(pf.artifact, "name", "")),
		"node_id":             pf.placement.NodeID,
		"agent_id":            "",
		"gpu_device_ids":      gpuDeviceIDs,
		"gpu_visible_env_key": pf.plan.GPUVisibleEnvKey,
		"operation_id":        operationID,
		"env":                 pf.plan.Env,
		"args":                pf.plan.Args,
		"volumes":             pf.plan.Mounts,
		"devices":             pf.plan.Devices,
		"host_port":           pf.service.HostPort,
		"container_port":      pf.bvPort,
		"ports": []map[string]interface{}{
			{"host_port": pf.service.HostPort, "container_port": pf.bvPort},
		},
		"docker": map[string]interface{}{
			"image":            pf.plan.Image,
			"container_name":   pf.plan.ContainerName,
			"command":          pf.plan.Entrypoint,
			"args":             pf.plan.Args,
			"privileged":       pf.plan.Privileged,
			"ipc_mode":         pf.plan.IPCMode,
			"uts_mode":         pf.plan.UTSMode,
			"network_mode":     pf.plan.NetworkMode,
			"shm_size":         pf.plan.ShmSize,
			"security_options": pf.plan.SecurityOptions,
			"ulimits":          pf.plan.Ulimits,
			"group_add":        pf.plan.GroupAdd,
			"gpu_device_ids":   gpuDeviceIDs,
		},
		"health_check": map[string]interface{}{
			"enabled":          pf.plan.HealthCheck.Path != "",
			"path":             pf.plan.HealthCheck.Path,
			"port":             pf.service.HostPort,
			"port_source":      "host_port",
			"container_port":   pf.bvPort,
			"scheme":           "http",
			"expected_status":  pf.plan.HealthCheck.ExpectedStatus,
			"timeout_seconds":  healthTimeoutSeconds,
			"interval_seconds": pf.plan.HealthCheck.IntervalSeconds,
		},
	}
	agentPayload, _ := json.Marshal(agentSpec)

	// BRR-E2E-001: Log host/container port mapping so health check URL can be traced.
	log.Info("deployment.start.agent_spec.ports",
		"deployment_id", deployID,
		"instance_id", instanceID,
		"host_port", pf.service.HostPort,
		"container_port", pf.bvPort,
		"health_check_path", pf.plan.HealthCheck.Path,
		"health_check_port_source", "host_port",
	)

	runPlanID := uuid.NewString()
	taskID := uuid.NewString()
	tid := pf.deploy["tenant_id"].(string)

	tx, txErr := h.DB.Begin()
	if txErr != nil {
		log.Error("deployment.start.tx_begin_failed", "error", txErr, "deployment_id", deployID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO model_instances (id, deployment_id, tenant_id, replica_index, node_id, agent_id, assigned_gpus_json, host_port, container_port, current_run_plan_id, actual_state, desired_state, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		instanceID, deployID, tid, 0, pf.placement.NodeID, "", jsonString(pf.placement.GPUIds), pf.service.HostPort, pf.bvPort, runPlanID, "pending", "running", now, now); err != nil {
		log.Error("deployment.start.instance_insert_failed", "error", err, "instance_id", instanceID, "deployment_id", deployID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	groupID := uuid.NewString()
	if _, err := tx.Exec(`INSERT INTO run_plan_groups (id, deployment_plan_id, mode, desired_count, ready_count, status, tenant_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		groupID, deployID, "single", 1, 0, "pending", tid, now, now); err != nil {
		log.Error("deployment.start.runplan_group_insert_failed", "error", err, "group_id", groupID, "deployment_id", deployID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if _, err := tx.Exec(`INSERT INTO resolved_run_plans (id, deployment_id, instance_id, tenant_id, backend_runtime_id, node_backend_runtime_id, plan_json, docker_preview, input_hash, plan_hash, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		runPlanID, deployID, instanceID, tid, pf.runtimeID, pf.nodeRuntimeID, string(planJSON), pf.commandPreview, pf.plan.InputHash, pf.plan.PlanHash, now); err != nil {
		log.Error("deployment.start.runplan_insert_failed", "error", err, "run_plan_id", runPlanID, "instance_id", instanceID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	for _, gid := range pf.placement.GPUIds {
		leaseID := uuid.NewString()
		if _, err := tx.Exec(`INSERT INTO gpu_leases (id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status, reserved_at, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			leaseID, gid, pf.placement.NodeID, deployID, instanceID, tid, "reserved", now, now, now); err != nil {
			log.Error("deployment.start.lease_insert_failed", "error", err, "lease_id", leaseID, "gpu_id", gid, "instance_id", instanceID)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		log.StateTransition(r.Context(), "deployment.start", "gpu_lease", leaseID, "", "reserved",
			"gpu_id", gid, "instance_id", instanceID, "deployment_id", deployID, "node_id", pf.placement.NodeID)
	}

	if _, err := tx.Exec(`INSERT INTO agent_tasks (id, task_type, status, tenant_id, deployment_id, instance_id, node_id, payload, timeout_seconds, operation_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		taskID, "model_instance_start", "pending", tid, deployID, instanceID, pf.placement.NodeID, string(agentPayload), 300, operationID, now, now); err != nil {
		log.Error("deployment.start.task_insert_failed", "error", err, "task_id", taskID, "instance_id", instanceID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

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

	log.Info("agent_task.created",
		"operation_id", operationID,
		"task_id", taskID,
		"task_type", "model_instance_start",
		"deployment_id", deployID,
		"instance_id", instanceID,
		"agent_id", "", "node_id", pf.placement.NodeID,
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "started",
		"deployment_id":  deployID,
		"instance_id":    instanceID,
		"task_id":        taskID,
		"run_plan_id":    runPlanID,
		"warnings":       pf.warns,
		"docker_preview": pf.commandPreview,
	})
}

func planHealthTimeout(hc runplan.HealthCheckInput, raw string) int {
	if hc.StartupTimeoutSeconds > 0 {
		return hc.StartupTimeoutSeconds
	}
	var m map[string]interface{}
	if json.Unmarshal([]byte(raw), &m) == nil {
		for _, key := range []string{"startup_timeout_seconds", "startupTimeoutSeconds", "timeout_seconds", "timeoutSeconds"} {
			if v, ok := m[key]; ok {
				switch n := v.(type) {
				case float64:
					if n > 0 {
						return int(n)
					}
				case int:
					if n > 0 {
						return n
					}
				}
			}
		}
	}
	if hc.TimeoutSeconds > 0 {
		return hc.TimeoutSeconds
	}
	return 30
}

// planHealthTimeout2 parses a health check JSON string and returns the timeout
// in seconds. Used when only the raw JSON is available (e.g., from preflight).
func planHealthTimeout2(raw string) int {
	var m map[string]interface{}
	if json.Unmarshal([]byte(raw), &m) == nil {
		for _, key := range []string{"startup_timeout_seconds", "startupTimeoutSeconds", "timeout_seconds", "timeoutSeconds"} {
			if v, ok := m[key]; ok {
				switch n := v.(type) {
				case float64:
					if n > 0 {
						return int(n)
					}
				case int:
					if n > 0 {
						return n
					}
				}
			}
		}
	}
	return 30
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

	// Find ALL non-terminal instances (running, starting, failed, pending, initializing).
	rows, err := h.DB.Query(`SELECT mi.id, mi.node_id, mi.container_id, mi.actual_state, COALESCE(n.status,'unknown')
		FROM model_instances mi
		LEFT JOIN nodes n ON n.id = mi.node_id
		WHERE mi.deployment_id = ? AND mi.actual_state NOT IN ('stopped')`, deployID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	type inst struct{ id, nodeID, containerID, state, nodeStatus string }
	var instances []inst
	for rows.Next() {
		var i inst
		rows.Scan(&i.id, &i.nodeID, &i.containerID, &i.state, &i.nodeStatus)
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
		h.DB.Exec(`UPDATE model_instances SET desired_state = 'stopped', actual_state = CASE WHEN actual_state = 'running' THEN 'stopping' ELSE actual_state END, updated_at = ? WHERE id = ?`, now, i.id)

		if i.nodeStatus != "online" {
			warnings = append(warnings, fmt.Sprintf("node %s is %s; stop task not dispatched for instance %s", i.nodeID, i.nodeStatus, i.id))
			continue
		}

		taskID := uuid.NewString()
		payloadMap := map[string]interface{}{
			"instance_id":    i.id,
			"container_id":   i.containerID,
			"container_name": fmt.Sprintf("lightai-%s", shortContainerSuffix(i.id)),
		}
		payloadJSON, _ := json.Marshal(payloadMap)
		if _, err := h.DB.Exec(`INSERT INTO agent_tasks (id, task_type, status, tenant_id, deployment_id, instance_id, node_id, payload, timeout_seconds, operation_id, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			taskID, "model_instance_stop", "pending", tid, deployID, i.id, i.nodeID, string(payloadJSON), 90, operationID, now, now); err != nil {
			log.Error("deployment.stop.task_insert_failed", "instance_id", i.id, "error", err)
			warnings = append(warnings, fmt.Sprintf("failed to dispatch stop task for instance %s", i.id))
			continue
		}
		status, result, waitErr := h.waitForAgentTaskResult(r.Context(), taskID, 90*time.Second)
		if waitErr != nil {
			warnings = append(warnings, fmt.Sprintf("stop task timed out for instance %s: %v", i.id, waitErr))
			continue
		}
		if status != "completed" {
			errMsg := strVal(result, "error_message", strVal(result, "error", "docker stop task failed"))
			warnings = append(warnings, fmt.Sprintf("stop task failed for instance %s: %s", i.id, errMsg))
			continue
		}
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
			"node_id": nid.String, "current_run_plan_id": rpid.String, "actual_state": as,
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

func (h *AgentHandler) HandleListRunPlanGroups(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	rows, err := h.DB.Query(`SELECT id, deployment_plan_id, mode, desired_count, ready_count, status, group_config_json, tenant_id, created_at, updated_at FROM run_plan_groups WHERE deployment_plan_id = ? ORDER BY created_at DESC`, deployID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, did, mode, status, cfg, tid, ca, ua string
		var desired, ready int
		if err := rows.Scan(&id, &did, &mode, &desired, &ready, &status, &cfg, &tid, &ca, &ua); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": id, "deployment_plan_id": did, "mode": mode,
			"desired_count": desired, "ready_count": ready, "status": status,
			"group_config_json": json.RawMessage(cfg), "tenant_id": tid,
			"created_at": ca, "updated_at": ua,
		})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgentHandler) HandleGetNodeRunPlan(w http.ResponseWriter, r *http.Request) {
	m := h.getNodeRunPlanJSON(r.PathValue("id"))
	if m == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *AgentHandler) HandleGetNodeRunPlanPreview(w http.ResponseWriter, r *http.Request) {
	m := h.getNodeRunPlanJSON(r.PathValue("id"))
	if m == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":              m["id"],
		"command_preview": m["command_preview"],
	})
}

func (h *AgentHandler) HandleGetNodeRunPlanLogs(w http.ResponseWriter, r *http.Request) {
	runPlanID := r.PathValue("id")
	tail := 200
	if rawTail := strings.TrimSpace(r.URL.Query().Get("tail")); rawTail != "" {
		parsed, err := strconv.Atoi(rawTail)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "tail must be a positive integer")
			return
		}
		tail = parsed
	}
	if tail > 5000 {
		tail = 5000
	}
	since := strings.TrimSpace(r.URL.Query().Get("since"))

	var deploymentID, instanceID, tenantID, nodeID, agentID, containerID, nodeStatus string
	err := h.DB.QueryRow(`SELECT r.deployment_id, r.instance_id, r.tenant_id,
			COALESCE(mi.node_id,''), COALESCE(mi.agent_id,''), COALESCE(mi.container_id,''),
			COALESCE(n.status,''), COALESCE(n.agent_id,'')
		FROM resolved_run_plans r
		JOIN model_instances mi ON mi.id = r.instance_id
		JOIN nodes n ON n.id = mi.node_id
		WHERE r.id = ?`, runPlanID).Scan(&deploymentID, &instanceID, &tenantID, &nodeID, &agentID, &containerID, &nodeStatus, &agentID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Error("node_run_plan.logs.lookup_failed", "run_plan_id", runPlanID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !tenantScopeCheck(r, tenantID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if nodeStatus != "online" {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("node %s is offline; docker logs cannot be fetched", nodeID))
		return
	}

	taskID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	payloadMap := map[string]interface{}{
		"instance_id":    instanceID,
		"container_id":   containerID,
		"container_name": fmt.Sprintf("lightai-%s", shortContainerSuffix(instanceID)),
		"tail":           tail,
	}
	if since != "" {
		payloadMap["since"] = since
	}
	payloadJSON, _ := json.Marshal(payloadMap)

	if _, err := h.DB.Exec(`INSERT INTO agent_tasks (id, task_type, status, tenant_id, deployment_id, instance_id, node_id, payload, timeout_seconds, operation_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		taskID, "model_instance_logs", "pending", tenantID, deploymentID, instanceID, nodeID, string(payloadJSON), 30, taskID, now, now); err != nil {
		log.Error("node_run_plan.logs.task_insert_failed", "run_plan_id", runPlanID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	status, result, waitErr := h.waitForAgentTaskResult(r.Context(), taskID, 30*time.Second)
	if waitErr != nil {
		writeJSON(w, http.StatusGatewayTimeout, map[string]interface{}{
			"error":   waitErr.Error(),
			"id":      runPlanID,
			"task_id": taskID,
			"status":  "timeout",
		})
		return
	}
	if status != "completed" {
		errorMsg := strVal(result, "error_message", strVal(result, "error", "docker logs task failed"))
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"error":        errorMsg,
			"id":           runPlanID,
			"task_id":      taskID,
			"status":       status,
			"container_id": containerID,
		})
		return
	}

	stdout := redactDockerLogText(strVal(result, "stdout", ""))
	stderr := redactDockerLogText(strVal(result, "stderr", ""))
	logsText := redactDockerLogText(strVal(result, "logs", strVal(result, "logs_summary", stdout+stderr)))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":            runPlanID,
		"task_id":       taskID,
		"deployment_id": deploymentID,
		"instance_id":   instanceID,
		"node_id":       nodeID,
		"container_id":  strVal(result, "container_id", containerID),
		"tail":          tail,
		"since":         since,
		"status":        "ok",
		"runtime_state": strVal(result, "runtime_state", "ok"),
		"stdout":        stdout,
		"stderr":        stderr,
		"logs":          logsText,
	})
}

func (h *AgentHandler) waitForAgentTaskResult(ctx interface{ Done() <-chan struct{} }, taskID string, timeout time.Duration) (string, map[string]interface{}, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", nil, fmt.Errorf("request cancelled while waiting for agent task")
		case <-deadline.C:
			return "", nil, fmt.Errorf("timed out waiting for agent task")
		case <-ticker.C:
			var status string
			var raw sql.NullString
			err := h.DB.QueryRow(`SELECT status, result FROM agent_tasks WHERE id = ?`, taskID).Scan(&status, &raw)
			if err != nil {
				return "", nil, fmt.Errorf("agent logs task disappeared")
			}
			if status == "completed" || status == "failed" || status == "timed_out" {
				out := map[string]interface{}{}
				if raw.Valid && strings.TrimSpace(raw.String) != "" {
					_ = json.Unmarshal([]byte(raw.String), &out)
				}
				return status, out, nil
			}
		}
	}
}

func shortContainerSuffix(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

var sensitiveEnvLogPattern = regexp.MustCompile(`(?i)([A-Z0-9_]*(TOKEN|SECRET|PASSWORD|PASSWD|API_KEY|SESSION|CSRF)[A-Z0-9_]*=)[^\s]+`)

func redactDockerLogText(s string) string {
	if s == "" {
		return ""
	}
	return sensitiveEnvLogPattern.ReplaceAllString(s, `${1}<redacted>`)
}

func (h *AgentHandler) getNodeRunPlanJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(`SELECT id, deployment_id, instance_id, tenant_id, backend_runtime_id, COALESCE(node_backend_runtime_id,''), plan_json, docker_preview, input_hash, plan_hash, created_at FROM resolved_run_plans WHERE id = ?`, id)
	var rid, did, iid, tid, brid, nbrid, plan, preview, inputHash, planHash, ca string
	if err := row.Scan(&rid, &did, &iid, &tid, &brid, &nbrid, &plan, &preview, &inputHash, &planHash, &ca); err != nil {
		return nil
	}
	return map[string]interface{}{
		"id": rid, "deployment_plan_id": did, "instance_id": iid, "tenant_id": tid,
		"backend_runtime_id": brid, "node_backend_runtime_id": nbrid,
		"run_plan_json": json.RawMessage(plan), "command_preview": preview,
		"input_hash": inputHash, "plan_hash": planHash, "created_at": ca,
	}
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

	// BRR-RV-001: Use the real preflight resolver so dry-run reports the same
	// validation results as a real start would, including RunPlan resolution,
	// NodeBackendRuntime readiness, ModelLocation availability, and GPU checks.
	pf := h.preflightDeployment(deployID, r)

	valid := len(pf.errs) == 0
	result := map[string]interface{}{
		"valid":    valid,
		"errors":   pf.errs, "error_details": func() []string { var s []string; for _, e := range pf.errs { s = append(s, e.Message) }; return s }(),
		"warnings": pf.warns,
	}
	if pf.plan != nil {
		result["command_preview"] = pf.commandPreview
		result["selected_node"] = pf.placement.NodeID
		result["selected_runtime"] = pf.runtimeID
		result["selected_model_location"] = pf.locationID
		if pf.plan.Image != "" {
			result["resolved_image"] = pf.plan.Image
		}
	}

	if valid {
		log.Info("deployment.dry_run.succeeded", "request_id", requestID,
			"tenant_id", tid, "actor_id", actorID, "deployment_id", deployID,
			"model_artifact_id", pf.artifactID,
			"backend_runtime_id", pf.runtimeID,
			"runtime", "docker", "vendor", pf.rtVendor,
			"image", pf.plan.Image, "node_id", pf.placement.NodeID)
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "deployment.dry_run", ResourceType: "deployment",
			ResourceID: deployID, Result: "success",
			RequestID: requestID,
			Detail:    fmt.Sprintf("runtime=%s vendor=%s image=%s node=%s", pf.runtimeID, pf.rtVendor, pf.plan.Image, pf.placement.NodeID),
		})
	} else {
		log.Warn("deployment.dry_run.failed", "request_id", requestID,
			"tenant_id", tid, "actor_id", actorID, "deployment_id", deployID,
			"reason", "validation_failed", "errors", fmt.Sprintf("%v", pf.errs))
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "deployment.dry_run", ResourceType: "deployment",
			ResourceID: deployID, Result: "failure",
			RequestID: requestID, Error: fmt.Sprintf("validation: %v", pf.errs),
		})
	}

	writeJSON(w, http.StatusOK, result)
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
