package api

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	agentruntime "lightai-go/internal/agent/runtime"
	"lightai-go/internal/common/log"
	"lightai-go/internal/server/runplan"

	"github.com/google/uuid"
)

// ==========================================================================
// ModelDeployment CRUD (minimal)
// ==========================================================================

func (h *AgentHandler) HandleListDeployments(w http.ResponseWriter, r *http.Request) {
	tid := tenantID(r)
	q := deploymentSelectSQL()
	var out []map[string]interface{}
	var err error
	if isPlatformAdmin(r) {
		out, err = h.queryDeployments(q + ` ORDER BY md.name`)
	} else {
		out, err = h.queryDeployments(q+` WHERE md.tenant_id = ? ORDER BY md.name`, tid)
	}
	if err != nil {
		log.Error("list deployments", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, publicDeploymentList(out))
}

func deploymentConfigSnapshotFromNBR(nbrSnapshot, imageRef string) string {
	if nbrSnapshot == "" || nbrSnapshot == "{}" {
		return ""
	}
	set := copyConfigSet(nbrSnapshot)
	if imageRef != "" {
		setConfigValue(set, "launcher.image", imageRef, "NodeBackendRuntime", "", "checked_image_ref")
	}
	return configSetJSON(set)
}

func (h *AgentHandler) buildDeploymentRuntimeSnapshot(runtimeID string) string {
	rt := h.getBackendRuntimeJSON(runtimeID)
	if rt == nil {
		return "{}"
	}
	return rawJSONString(rt["config_set_json"], "{}")
}

func (h *AgentHandler) HandleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	name := strVal(req, "name", "")

	if legacyKey := rejectLegacyDeploymentPayload(req); legacyKey != "" {
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("%s is not accepted by the ConfigSet deployment contract; use node_backend_runtime_id and config_overrides", legacyKey))
		return
	}

	// R-012: Reject replicas > 1 until multi-instance support is implemented.
	if intVal(req, "replicas", 1) > 1 {
		writeError(w, http.StatusBadRequest, "multi-replica deployments are not yet supported; use replicas=1")
		return
	}

	artifactID := strVal(req, "model_artifact_id", "")
	nodeBackendRuntimeID := strVal(req, "node_backend_runtime_id", "")
	if nodeBackendRuntimeID == "" {
		writeError(w, http.StatusBadRequest, "node_backend_runtime_id is required")
		return
	}

	// Resolve node_backend_runtime_id: must exist and be ready.
	var nbrBackendRuntimeID, nbrNodeID, nbrStatus, nbrConfigSetRaw, nbrSourceMetaRaw, nbrImageRef string
	if err := h.DB.QueryRow(
		`SELECT backend_runtime_id, node_id, status, config_set_json, source_metadata_json, image_ref FROM node_backend_runtimes WHERE id = ?`,
		nodeBackendRuntimeID,
	).Scan(&nbrBackendRuntimeID, &nbrNodeID, &nbrStatus, &nbrConfigSetRaw, &nbrSourceMetaRaw, &nbrImageRef); err != nil {
		writeError(w, http.StatusBadRequest, "node_backend_runtime_id not found")
		return
	}
	if !isNBRDeployable(nbrStatus) {
		reason := nbrDisabledReason(nbrStatus, "")
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("node backend runtime is not deployable (status=%s): %s", nbrStatus, reason))
		return
	}

	// Derive backend_runtime_id and node_id from the NBR.
	backendRuntimeID := nbrBackendRuntimeID

	// Set placement node_id from NBR if not explicitly provided.
	if placementRaw, ok := req["placement_json"]; ok && placementRaw != nil {
		if pm, ok := placementRaw.(map[string]interface{}); ok {
			if existingNode, ok := pm["node_id"].(string); ok && existingNode != "" && existingNode != nbrNodeID {
				writeError(w, http.StatusBadRequest, "placement node_id does not match node_backend_runtime_id node")
				return
			}
			pm["node_id"] = nbrNodeID
		}
	} else {
		req["placement_json"] = map[string]interface{}{"node_id": nbrNodeID}
	}

	// REVIEW-022: Validate references at create time.
	if artifactID != "" {
		var exists string
		if err := h.DB.QueryRow(`SELECT id FROM model_artifacts WHERE id = ?`, artifactID).Scan(&exists); err != nil {
			writeError(w, http.StatusBadRequest, "model_artifact_id not found")
			return
		}
	}
	if name == "" || name == "deployment" {
		name = h.defaultDeploymentName(artifactID, nodeBackendRuntimeID)
	}
	displayName := strVal(req, "display_name", "")
	if displayName == "" {
		displayName = name
	}

	// Validate model location matches NBR node (same check as preview/preflight/start).
	if artifactID != "" && nbrNodeID != "" {
		if loc, _, reason := h.findDeployableModelLocation(artifactID, nbrNodeID); loc == nil {
			writeError(w, http.StatusBadRequest, reason)
			return
		}
	}

	id := uuid.NewString()
	tid := tenantID(r)
	actorID := actorIDFromSession(r)
	requestID := log.RequestIDFromContext(r.Context())
	now := time.Now().Format(time.RFC3339)

	configSetRaw := deploymentConfigSnapshotFromNBR(nbrConfigSetRaw, nbrImageRef)
	if configSetRaw == "" {
		writeError(w, http.StatusBadRequest, "node backend runtime config snapshot is missing; recreate node backend runtime")
		return
	}
	configOverrides := map[string]interface{}{}
	if overrides, ok := req["config_overrides"]; ok {
		configOverrides = mapFromAny(overrides)
	}
	deploymentConfigSet := copyConfigSet(configSetRaw)
	applyConfigOverrides(deploymentConfigSet, configOverrides, "Deployment", id)
	var patchErr error
	deploymentConfigSet, patchErr = applyEditableConfigPatchIfPresent(deploymentConfigSet, req, "deployment", id)
	if patchErr != nil {
		writeError(w, http.StatusBadRequest, patchErr.Error())
		return
	}
	placementCompat, _ := req["placement_json"].(map[string]interface{})
	serviceCompat, _ := req["service_json"].(map[string]interface{})
	materializeDeploymentCompatConfig(deploymentConfigSet, placementCompat, serviceCompat, "deployment", id)
	configSetRaw = configSetJSON(deploymentConfigSet)
	sourceMetadata := map[string]interface{}{
		"copy_semantics":                  "copy_on_create",
		"source_backend_runtime_id":       backendRuntimeID,
		"source_node_backend_runtime_id":  nodeBackendRuntimeID,
		"source_node_runtime_metadata":    configSourceMetadata(nbrSourceMetaRaw),
		"source_config_hash":              planHashStr(configSetRaw),
		"source_type":                     "node_backend_runtime",
		"source_runtime_config_authority": "config_set",
	}

	_, err := h.DB.Exec(`INSERT INTO model_deployments (id, name, display_name, description, model_artifact_id, backend_runtime_id, node_backend_runtime_id, replicas, placement_json, service_json, config_overrides_json, source_backend_runtime_id, source_node_backend_runtime_id, source_config_hash, copied_at, config_set_json, source_metadata_json, desired_state, status, tenant_id, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, name, displayName, strVal(req, "description", ""),
		artifactID, backendRuntimeID, nodeBackendRuntimeID,
		intVal(req, "replicas", 1), jsonString(req["placement_json"]), jsonString(req["service_json"]),
		jsonString(configOverrides),
		backendRuntimeID,
		nodeBackendRuntimeID,
		planHashStr(configSetRaw),
		now,
		configSetRaw,
		jsonString(sourceMetadata),
		"stopped", "saved", tid, now, now,
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
	writeJSON(w, http.StatusCreated, publicDeploymentJSON(h.getDeploymentJSON(id)))
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
	writeJSON(w, http.StatusOK, publicDeploymentJSON(m))
}

func (h *AgentHandler) defaultDeploymentName(artifactID, nodeBackendRuntimeID string) string {
	var artifactName string
	_ = h.DB.QueryRow(`SELECT COALESCE(NULLIF(display_name,''), name, id) FROM model_artifacts WHERE id = ?`, artifactID).Scan(&artifactName)
	if strings.TrimSpace(artifactName) == "" {
		artifactName = "model"
	}
	var runtimeName string
	_ = h.DB.QueryRow(`SELECT COALESCE(NULLIF(display_name,''), id) FROM node_backend_runtimes WHERE id = ?`, nodeBackendRuntimeID).Scan(&runtimeName)
	if strings.TrimSpace(runtimeName) == "" {
		runtimeName = "runtime"
	}
	base := slugify(artifactName + "-" + runtimeName)
	if base == "" {
		base = "deployment"
	}
	suffix := time.Now().Format("20060102150405")
	name := base + "-" + suffix
	if len(name) > 120 {
		name = name[:120]
	}
	return name
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
	if legacyKey := rejectLegacyDeploymentPayload(req); legacyKey != "" {
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("%s is not accepted by the ConfigSet deployment contract; use config_overrides", legacyKey))
		return
	}

	now := time.Now().Format(time.RFC3339)
	sets := []string{"updated_at = ?"}
	args := []interface{}{now}
	for _, f := range []string{"name", "display_name", "description", "model_artifact_id"} {
		if v, ok := req[f]; ok {
			if s, ok := v.(string); ok {
				v = strings.TrimSpace(s)
				if f == "name" && v == "" {
					writeError(w, http.StatusBadRequest, "name is required")
					return
				}
			}
			sets = append(sets, f+" = ?")
			args = append(args, v)
		}
	}
	for _, f := range []string{"placement_json", "service_json"} {
		if v, ok := req[f]; ok {
			sets = append(sets, f+" = ?")
			args = append(args, jsonString(v))
		}
	}
	if v, ok := req["config_overrides"]; ok {
		configSetRaw := rawJSONString(existing["config_set"], rawJSONString(existing["config_set_json"], "{}"))
		configSet := copyConfigSet(configSetRaw)
		overrides := mapFromAny(v)
		applyConfigOverrides(configSet, overrides, "Deployment", id)
		sets = append(sets, "config_overrides_json = ?")
		args = append(args, jsonString(overrides))
		sets = append(sets, "config_set_json = ?")
		args = append(args, configSetJSON(configSet))
	}
	if v, ok := req["replicas"]; ok {
		sets = append(sets, "replicas = ?")
		args = append(args, intVal(map[string]interface{}{"replicas": v}, "replicas", 1))
	}
	args = append(args, id)
	if _, err := h.DB.Exec(`UPDATE model_deployments SET `+joinSets(sets)+` WHERE id = ?`, args...); err != nil {
		log.Error("deployment.update.failed", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, publicDeploymentJSON(h.getDeploymentJSON(id)))
}

func publicDeploymentList(in []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(in))
	for _, item := range in {
		out = append(out, publicDeploymentJSON(item))
	}
	return out
}

func publicDeploymentJSON(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	for _, k := range []string{
		"backend_runtime_id",
		"config_overrides_json",
		"config_set_json",
		"source_metadata_json",
	} {
		delete(out, k)
	}
	return out
}

type activeDeploymentRunResponse struct {
	Blocked      bool   `json:"blocked"`
	ReasonCode   string `json:"reason_code"`
	Message      string `json:"message"`
	DeploymentID string `json:"deployment_id"`
	InstanceID   string `json:"instance_id,omitempty"`
	TaskID       string `json:"task_id,omitempty"`
	State        string `json:"state,omitempty"`
}

func (h *AgentHandler) activeDeploymentRun(deployID string) activeDeploymentRunResponse {
	var instanceID, state string
	err := h.DB.QueryRow(`SELECT id, actual_state FROM model_instances
		WHERE deployment_id = ? AND actual_state IN ('pending','starting','provisioning','running','healthy','stopping')
		ORDER BY created_at DESC LIMIT 1`, deployID).Scan(&instanceID, &state)
	if err == nil {
		switch state {
		case "pending", "starting", "provisioning":
			return activeDeploymentRunResponse{Blocked: true, ReasonCode: "deployment_starting", Message: "deployment is already starting", DeploymentID: deployID, InstanceID: instanceID, State: state}
		case "running", "healthy":
			return activeDeploymentRunResponse{Blocked: true, ReasonCode: "deployment_running", Message: "deployment is already running", DeploymentID: deployID, InstanceID: instanceID, State: state}
		case "stopping":
			return activeDeploymentRunResponse{Blocked: true, ReasonCode: "deployment_stopping", Message: "deployment is stopping", DeploymentID: deployID, InstanceID: instanceID, State: state}
		default:
			return activeDeploymentRunResponse{Blocked: true, ReasonCode: "deployment_active", Message: "deployment already has an active instance", DeploymentID: deployID, InstanceID: instanceID, State: state}
		}
	}
	var taskID, taskStatus string
	err = h.DB.QueryRow(`SELECT id, status FROM agent_tasks
		WHERE deployment_id = ? AND task_type IN ('model_instance_start','model_instance_stop') AND status IN ('pending','in_progress')
		ORDER BY created_at DESC LIMIT 1`, deployID).Scan(&taskID, &taskStatus)
	if err == nil {
		return activeDeploymentRunResponse{Blocked: true, ReasonCode: "deployment_task_active", Message: "deployment already has an active task", DeploymentID: deployID, TaskID: taskID, State: taskStatus}
	}
	return activeDeploymentRunResponse{DeploymentID: deployID}
}

func firstPositive(vals ...int) int {
	for _, v := range vals {
		if v > 0 {
			return v
		}
	}
	return 0
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
	deploy               map[string]interface{}
	artifactID           string
	artifact             map[string]interface{}
	runtimeID            string
	nbrConfigSet         string // config_set_json from NodeBackendRuntime
	deployConfigSnapshot string // config_set_json from ModelDeployment
	placement            struct {
		NodeID         string   `json:"node_id"`
		AcceleratorIds []string `json:"accelerator_ids"`
	}
	service struct {
		HostPort      int `json:"host_port"`
		ContainerPort int `json:"container_port"`
		AppPort       int `json:"app_port"`
		HealthPort    int `json:"health_port"`
		APITestPort   int `json:"api_test_port"`
	}
	params             map[string]interface{}
	envOverrides       map[string]string
	parameterValues    []runplan.ParameterValue
	disabledParameters []runplan.ParameterValue
	rtVendor           string
	rtImage            string
	rtDockerJSON       string
	rtArgsOverride     string
	rtEntryOverride    string
	rtDefaultEnv       string
	rtBackendID        string
	rtVersionID        string
	rtModelMount       string
	rtHC               string
	rtVersionSnapshot  string
	processStartConfig *runplan.ProcessStartConfig // from ConfigSet process profile.
	backendName        string
	backendDefaultEnv  string
	bvEntrypoint       string
	bvArgs             string
	bvBackendParams    string
	bvParamDefs        string
	bvHC               string
	bvPort             int
	bvDefaultImages    string
	bvEnv              string
	bvVendorOptions    string
	nodeIP             string
	gpuInfos           []runplan.GPUInfo
	nodeRuntimeID      string
	locationID         string
	modelRoot          string
	relativePath       string
	absolutePath       string
	plan               *runplan.ResolvedRunPlan
	lintResult         *runplan.LintResult
	errs               []PreflightError
	warns              []string
	commandPreview     string
}

// addErr appends a structured PreflightError to the result.
func (pf *preflightResult) addErr(code, message string, ctx map[string]interface{}) {
	pf.errs = append(pf.errs, PreflightError{Code: code, Message: message, Context: ctx})
}

// addWarn appends a warning string to the result.
func (pf *preflightResult) addWarn(message string) {
	pf.warns = append(pf.warns, message)
}

// validateContextLength checks the user-requested context parameter against
// the model artifact's default_context_length. It adds errors or warnings
// to the preflight result as appropriate.
func (pf *preflightResult) validateContextLength() {
	// Read model's default_context_length from artifact.
	modelCtx := intVal(pf.artifact, "default_context_length", 0)

	// Determine which context parameter to check based on backend.
	var userCtx int
	var paramName string
	backendName := pf.backendName

	switch {
	case strings.Contains(backendName, "vllm"):
		paramName = "max_model_len"
		if v, ok := pf.params[paramName]; ok {
			userCtx = intFromInterface(v)
		}
		if userCtx == 0 {
			if v, ok := pf.params["--max-model-len"]; ok {
				userCtx = intFromInterface(v)
				paramName = "--max-model-len"
			}
		}
	case strings.Contains(backendName, "sglang"):
		paramName = "context_length"
		if v, ok := pf.params[paramName]; ok {
			userCtx = intFromInterface(v)
		}
		if userCtx == 0 {
			if v, ok := pf.params["--context-length"]; ok {
				userCtx = intFromInterface(v)
				paramName = "--context-length"
			}
		}
	case strings.Contains(backendName, "llamacpp"):
		paramName = "ctx_size"
		if v, ok := pf.params[paramName]; ok {
			userCtx = intFromInterface(v)
		}
		if userCtx == 0 {
			if v, ok := pf.params["n_gpu_layers"]; ok {
				_ = intFromInterface(v) // not ctx, skip
			}
		}
		if userCtx == 0 {
			if v, ok := pf.params["--ctx-size"]; ok {
				userCtx = intFromInterface(v)
				paramName = "--ctx-size"
			}
		}
	}

	// No user context parameter set — nothing to validate.
	if userCtx == 0 {
		return
	}

	// Build context info for DryRun visibility.
	ctxInfo := map[string]interface{}{
		"user_context":  userCtx,
		"model_context": modelCtx,
		"param_name":    paramName,
	}

	// Model context length unknown — warn, don't block.
	if modelCtx == 0 {
		pf.addWarn(fmt.Sprintf(
			"unknown_model_context_length: cannot validate user %s=%d against model; model default_context_length is not set",
			paramName, userCtx))
		ctxInfo["status"] = "warning"
		ctxInfo["code"] = "unknown_model_context_length"
		pf.commandPreview = fmt.Sprintf("%s\n# context_validation: %s", pf.commandPreview, toJSON(ctxInfo))
		return
	}

	// User context within model limits — pass silently.
	if userCtx <= modelCtx {
		ctxInfo["status"] = "pass"
		ctxInfo["code"] = "context_length_ok"
		return
	}

	// User context exceeds model context — check for rope_scaling.
	hasRopeScaling := false
	if rs, ok := pf.artifact["rope_scaling"]; ok && rs != nil {
		hasRopeScaling = true
	}
	// Also check from metadata_json on the location (if available via artifact metadata)
	// For now, check if the artifact has a metadata field indicating rope scaling.
	_ = hasRopeScaling

	ctxInfo["status"] = "error"
	ctxInfo["code"] = "context_length_exceeded"
	message := fmt.Sprintf(
		"user %s=%d exceeds model default_context_length=%d",
		paramName, userCtx, modelCtx)

	// If rope_scaling is present, warn instead of error.
	if hasRopeScaling {
		ctxInfo["status"] = "warning"
		ctxInfo["code"] = "context_length_exceeded_with_rope"
		pf.addWarn(message)
		pf.addWarn("model has rope_scaling; extended context may be supported")
	} else {
		pf.addErr("context_length_exceeded", message, map[string]interface{}{
			"user_context":  userCtx,
			"model_context": modelCtx,
			"param_name":    paramName,
			"artifact_id":   pf.artifactID,
		})
	}

	pf.commandPreview = fmt.Sprintf("%s\n# context_validation: %s", pf.commandPreview, toJSON(ctxInfo))
}

// intFromInterface converts an interface{} to int, handling float64 (JSON numbers).
func intFromInterface(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

// toJSON marshals a value to a compact JSON string.
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
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
	if v, ok := deploy["config_set_json"]; ok {
		switch raw := v.(type) {
		case json.RawMessage:
			pf.deployConfigSnapshot = string(raw)
		case string:
			pf.deployConfigSnapshot = raw
		default:
			if b, err := json.Marshal(v); err == nil {
				pf.deployConfigSnapshot = string(b)
			}
		}
	}
	if pf.deployConfigSnapshot == "" {
		pf.deployConfigSnapshot = "{}"
	}
	deployConfigSet := mapFromAny(deploy["config_set_json"])

	pf.artifactID = strVal(deploy, "model_artifact_id", "")
	pf.nodeRuntimeID = strVal(deploy, "source_node_backend_runtime_id", "")
	pf.runtimeID = strVal(deploy, "backend_runtime_id", "") // internal template reference, not deployment selector
	if pf.artifactID == "" {
		pf.addErr("unknown", "model_artifact_id is required", map[string]interface{}{"artifact_id": pf.artifactID})
		return pf
	}
	if pf.nodeRuntimeID == "" {
		pf.addErr("node_backend_runtime_not_ready",
			"deployment has no node_backend_runtime_id reference; recreate with a valid NBR",
			map[string]interface{}{"deployment_id": deployID})
		return pf
	}

	// Parse placement/service JSON from DB fields.
	json.Unmarshal(rawJSONBytes(deploy["placement_json"]), &pf.placement)
	json.Unmarshal(rawJSONBytes(deploy["service_json"]), &pf.service)
	runtimeParameterValues := configSetParameterValues(deployConfigSet)

	// Inject service ports into parameters so mapParametersToArgs uses the
	// user's app_port instead of the ParameterDef hardcoded default (e.g. --port 8000).
	// Without this, the resolver silently ignores the deployment port config.
	if pf.params == nil {
		pf.params = make(map[string]interface{})
	}
	if pf.service.AppPort > 0 {
		pf.params["--port"] = float64(pf.service.AppPort)
		pf.params["port"] = float64(pf.service.AppPort)
	}
	if pf.service.HostPort > 0 {
		pf.params["--host-port"] = float64(pf.service.HostPort)
	}

	// Validate artifact exists.
	artifact := h.getArtifactJSON(pf.artifactID)
	if artifact == nil {
		pf.addErr("model_location_missing", "model artifact not found", map[string]interface{}{"artifact_id": pf.artifactID})
		return pf
	}
	pf.artifact = artifact

	// Node comes from the NodeBackendRuntime — resolved below.

	h.DB.QueryRow(`SELECT br.backend_id, br.backend_version_id, br.vendor, br.runtime_type, ib.name
		FROM backend_runtimes br
		JOIN inference_backends ib ON ib.id = br.backend_id
		WHERE br.id = ?`, pf.runtimeID).Scan(&pf.rtBackendID, &pf.rtVersionID, &pf.rtVendor, &pf.rtVersionSnapshot, &pf.backendName)
	pf.rtImage = configString(deployConfigSet, "launcher.image", "")
	pf.rtDockerJSON = jsonString(configObject(deployConfigSet, "launcher.docker_options"))
	pf.rtArgsOverride = jsonString(configStringSlice(deployConfigSet, "launcher.command"))
	pf.rtEntryOverride = jsonString(configStringSlice(deployConfigSet, "launcher.entrypoint"))
	pf.rtDefaultEnv = jsonString(configStringMap(deployConfigSet, "runtime.env"))
	pf.rtModelMount = jsonString(configObject(deployConfigSet, "runtime.model_mount"))
	pf.rtHC = jsonString(configObject(deployConfigSet, "runtime.health"))
	pf.bvEntrypoint = pf.rtEntryOverride
	pf.bvArgs = pf.rtArgsOverride
	pf.bvBackendParams = "[]"
	pf.bvParamDefs = jsonString(configSetParameterDefs(deployConfigSet))
	pf.bvHC = pf.rtHC
	pf.bvPort = intFromAny(configValue(deployConfigSet, "backend.common.port", 8000), 8000)
	pf.bvDefaultImages = jsonString(map[string]string{pf.rtVendor: pf.rtImage})
	pf.bvEnv = pf.rtDefaultEnv
	pf.bvVendorOptions = "{}"

	// ── Context Length Validation ──
	// Compare user-requested context parameter against model's default_context_length.
	pf.validateContextLength()

	// Fetch node IP.
	pf.nodeIP = "127.0.0.1"
	h.DB.QueryRow(`SELECT primary_ip FROM nodes WHERE id = ?`, pf.placement.NodeID).Scan(&pf.nodeIP)

	// Auto-assign first available GPU on the node if none specified.
	if len(pf.placement.AcceleratorIds) == 0 && pf.placement.NodeID != "" {
		var autoGpuID string
		h.DB.QueryRow(`SELECT id FROM gpu_devices WHERE node_id = ? AND status = 'available' LIMIT 1`,
			pf.placement.NodeID).Scan(&autoGpuID)
		if autoGpuID != "" {
			pf.placement.AcceleratorIds = []string{autoGpuID}
		}
	}

	// Validate NodeBackendRuntime readiness. Runtime configuration comes from
	// the deployment's frozen ConfigSet, not from the live NBR row.
	var nodeRuntimeStatus, nbrProbeResults string
	h.DB.QueryRow(`SELECT status, backend_runtime_id, node_id, COALESCE(config_set_json,'{}'), COALESCE(probe_results_json,'{}') FROM node_backend_runtimes WHERE id = ?`, pf.nodeRuntimeID).Scan(&nodeRuntimeStatus, &pf.runtimeID, &pf.placement.NodeID, &pf.nbrConfigSet, &nbrProbeResults)
	if !isNBRDeployable(nodeRuntimeStatus) {
		reason := nbrDisabledReason(nodeRuntimeStatus, "")
		if nodeRuntimeStatus == "" {
			reason = "node_backend_runtime_id not found; recreate deployment with a valid NBR"
		}
		pf.addErr("node_backend_runtime_not_ready", reason, map[string]interface{}{"node_runtime_id": pf.nodeRuntimeID, "nbr_status": nodeRuntimeStatus, "node_id": pf.placement.NodeID, "runtime_id": pf.runtimeID})
		return pf
	}
	pf.processStartConfig = processStartConfigFromProbe(nbrProbeResults)

	// Validate ModelLocation.
	location, _, reason := h.findDeployableModelLocation(pf.artifactID, pf.placement.NodeID)
	if location == nil {
		pf.addErr("model_location_missing", reason, map[string]interface{}{"node_id": pf.placement.NodeID, "artifact_id": pf.artifactID})
		return pf
	}
	pf.locationID = strVal(location, "id", "")
	pf.modelRoot = strVal(location, "model_root", "")
	pf.relativePath = strVal(location, "relative_path", "")
	pf.absolutePath = strVal(location, "absolute_path", "")

	// Fetch GPU info.
	for _, gid := range pf.placement.AcceleratorIds {
		var idx int
		var vendor string
		if err := h.DB.QueryRow(`SELECT index_num, vendor FROM gpu_devices WHERE id = ?`, gid).Scan(&idx, &vendor); err != nil {
			log.Warn("preflight.gpu_lookup_failed", "gpu_id", gid, "error", err)
			continue
		}
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
	if hc.ExpectedStatus == 0 {
		ss := successStatusFromRaw(pf.bvHC)
		if len(ss) > 0 {
			hc.ExpectedStatus = ss[0]
		}
	}
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
	var rtHC runplan.HealthCheckInput
	if pf.rtHC != "" && pf.rtHC != "{}" {
		json.Unmarshal([]byte(pf.rtHC), &rtHC)
	}
	if len(rtEntryOverride) > 0 {
		entrypoint = rtEntryOverride
	}
	var defaultArgs []string
	json.Unmarshal([]byte(pf.bvArgs), &defaultArgs)

	instanceID := uuid.NewString()

	// ── Phase D: Compatibility check before RunPlan resolution ──
	modelFormat := strVal(artifact, "format", "custom")
	modelTask := strVal(artifact, "task_type", "chat")
	modelPathType := "directory" // fallback default
	modelDeployable := true
	if loc := h.getModelLocationJSON(pf.locationID); loc != nil {
		// path_type is persisted by the scanner in model_locations.path_type.
		// Use the stored value instead of inferring from format.
		if pt, ok := loc["path_type"].(string); ok && pt != "" {
			modelPathType = pt
		} else {
			log.Warn("deployment_lifecycle: model_locations.path_type is empty for location, falling back to format inference",
				"location_id", pf.locationID,
				"format", modelFormat)
			if modelFormat == "gguf" {
				modelPathType = "file"
			}
		}
		if metaRaw, ok := loc["discovered_metadata_json"]; ok {
			var metaMap map[string]interface{}
			switch v := metaRaw.(type) {
			case map[string]interface{}:
				metaMap = v
			case json.RawMessage:
				json.Unmarshal(v, &metaMap)
			}
			if metaMap != nil {
				if dp, ok := metaMap["deployable"].(bool); ok {
					modelDeployable = dp
				}
			}
		}
	}
	backendCapRaw := jsonString(configObject(deployConfigSet, "backend.capabilities"))
	backendCaps, capsErr := runplan.ParseBackendCapabilities(backendCapRaw)
	if capsErr != nil || len(backendCaps.SupportedFormats) == 0 {
		backendCaps = runplan.BackendDescriptor{BackendName: pf.backendName}
	} else {
		backendCaps.BackendName = pf.backendName
	}
	compatResult := runplan.CheckCompatibility(
		runplan.ModelDescriptor{Format: modelFormat, Task: modelTask, Deployable: modelDeployable, PathType: modelPathType, Architecture: strVal(artifact, "architecture", "")},
		backendCaps,
	)
	if !compatResult.Compatible {
		pf.addErr(compatResult.Code, compatResult.Reason, map[string]interface{}{
			"artifact_id": pf.artifactID, "backend": pf.backendName,
			"model_format": modelFormat, "model_task": modelTask,
		})
		return pf
	}

	// Build NBR snapshot for resolver
	nbrSnapshot := &runplan.NBRSnapshotInfo{
		ArgsOverride:        argsOverride,
		DefaultEnv:          rtEnvMap,
		EntrypointOverride:  rtEntryOverride,
		Docker:              dockerSpec,
		ModelMount:          modelMount,
		HealthCheckOverride: rtHCOverridePtr(rtHC),
		DeviceBinding:       configDeviceBinding(deployConfigSet),
		ServicePortBinding:  configServicePortBinding(deployConfigSet),
		ParameterSchema:     paramDefs,
		ParameterValues:     runtimeParameterValues,
	}

	// Call the real RunPlan resolver with snapshot-based RuntimeInfo.
	resolveInput := runplan.ResolveInput{
		Backend:             &runplan.BackendInfo{ID: pf.rtBackendID, Name: pf.backendName, DefaultEnv: backendEnv},
		BackendVersion:      &runplan.VersionInfo{ID: pf.rtVersionID, Version: "", DefaultEntrypoint: entrypoint, DefaultArgs: defaultArgs, DefaultBackendParams: backendParams, ParameterDefs: paramDefs, HealthCheck: hc, DefaultContainerPort: pf.bvPort, DefaultImages: defaultImages, Env: bvEnvMap, VendorOptionsJSON: pf.bvVendorOptions},
		BackendRuntime:      &runplan.RuntimeInfo{ID: pf.runtimeID, Vendor: pf.rtVendor, RuntimeType: pf.rtVersionSnapshot, LauncherKind: configLauncherKind(deployConfigSet, pf.rtVersionSnapshot), ImageName: pf.rtImage, EntrypointOverride: rtEntryOverride, ArgsOverride: argsOverride, DefaultEnv: rtEnvMap, Docker: dockerSpec, ModelMount: modelMount, HealthCheckOverride: rtHCOverridePtr(rtHC)},
		NodeRuntimeOverride: nil,
		Artifact:            &runplan.ArtifactInfo{ID: pf.artifactID, Name: strVal(artifact, "name", ""), Path: pf.absolutePath, ModelRoot: pf.modelRoot, RelativePath: pf.relativePath},
		Deployment: &runplan.DeploymentInfo{ID: deployID, Name: strVal(deploy, "name", ""), Parameters: pf.params, EnvOverrides: pf.envOverrides, ParameterValues: pf.parameterValues, DisabledParameters: pf.disabledParameters, Service: runplan.ServiceInfo{
			HostPort:      pf.service.HostPort,
			ContainerPort: pf.service.ContainerPort,
			AppPort:       pf.service.AppPort,
			HealthPort:    pf.service.HealthPort,
			APITestPort:   pf.service.APITestPort,
		}, Placement: runplan.PlacementInfo{NodeID: pf.placement.NodeID, AcceleratorIds: pf.placement.AcceleratorIds}},
		InstanceID:         instanceID,
		Node:               &runplan.NodeInfo{ID: pf.placement.NodeID, IP: pf.nodeIP},
		AssignedGPUs:       pf.gpuInfos,
		ProcessStartConfig: pf.processStartConfig,
		NBRConfigSnapshot:  nbrSnapshot,
	}
	resolveInput = runplan.ApplySemanticSnapshot(resolveInput, semanticDeploymentSnapshot(deployConfigSet, map[string]interface{}{
		"host_port":      pf.service.HostPort,
		"container_port": pf.service.ContainerPort,
	}), pf.backendName)
	plan, resolveErrs, resolveWarns := runplan.ResolveWithSourceMap(resolveInput)
	for _, e := range resolveErrs {
		pf.addErr("unknown", e.Error(), nil)
	}
	for _, w := range resolveWarns {
		pf.warns = append(pf.warns, w)
	}
	if plan != nil {
		pf.plan = plan
		pf.commandPreview = runplan.EquivalentCommandPreview(plan)

		// Run lint on the resolved plan.
		envSources := make(map[string]string)
		for k := range plan.Env {
			envSources[k] = "platform" // simplified; actual source tracking requires layer metadata
		}
		dockerForLint := planRunplanDockerSpec(plan)
		lintResult := runplan.LintRunPlan(runplan.LintInput{
			FinalArgs:           plan.Args,
			Env:                 plan.Env,
			PlatformOwnedParams: runplan.DefaultLogicalParamSpecs(),
			BackendName:         pf.backendName,
			DockerSpec:          &dockerForLint,
			EnvSources:          envSources,
		})
		pf.lintResult = &lintResult
	} else if len(pf.errs) == 0 {
		// Resolver returned nil plan without explicit errors — add a catch-all.
		pf.addErr("unknown", "runplan resolution returned no plan", nil)
	}

	return pf
}

func processStartConfigFromProbe(raw string) *runplan.ProcessStartConfig {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "{}" {
		return nil
	}
	var probe map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil
	}
	psd, _ := probe["process_start_detection"].(map[string]interface{})
	if psd == nil || strings.TrimSpace(fmt.Sprint(psd["status"])) != "candidate_found" {
		return nil
	}
	entrypointMode := strings.TrimSpace(fmt.Sprint(psd["entrypoint_mode"]))
	commandPrefix := toStringSlice(psd["command_prefix"])
	entrypoint := toStringSlice(psd["entrypoint"])
	if entrypointMode == "" && len(commandPrefix) == 0 && len(entrypoint) == 0 {
		return nil
	}
	return &runplan.ProcessStartConfig{
		EntrypointMode: entrypointMode,
		Entrypoint:     entrypoint,
		CommandPrefix:  commandPrefix,
		ShellMode:      boolVal(psd, "shell_mode", false),
		ProfileID:      strings.TrimSpace(fmt.Sprint(psd["selected_profile_id"])),
		Source:         "probe_results",
		Confidence:     strings.TrimSpace(fmt.Sprint(psd["confidence"])),
		Warnings:       toStringSlice(psd["warnings"]),
	}
}

// planRunplanDockerSpec extracts a DockerSpecInfo from a ResolvedRunPlan for lint.
func planRunplanDockerSpec(plan *runplan.ResolvedRunPlan) runplan.DockerSpecInfo {
	return runplan.DockerSpecInfo{
		Privileged:      plan.Privileged,
		IPCMode:         plan.IPCMode,
		ShmSize:         plan.ShmSize,
		SecurityOptions: plan.SecurityOptions,
		CapAdd:          plan.CapAdd,
		CapDrop:         plan.CapDrop,
	}
}

func agentPlanDevices(devices []runplan.DeviceMapping) []agentruntime.PlanDevice {
	out := make([]agentruntime.PlanDevice, 0, len(devices))
	for _, d := range devices {
		out = append(out, agentruntime.PlanDevice{
			HostPath:      d.HostPath,
			ContainerPath: d.ContainerPath,
			Permissions:   d.Permissions,
		})
	}
	return out
}

func agentPlanMounts(mounts []runplan.MountMapping) []agentruntime.PlanMount {
	out := make([]agentruntime.PlanMount, 0, len(mounts))
	for _, m := range mounts {
		out = append(out, agentruntime.PlanMount{
			HostPath:      m.HostPath,
			ContainerPath: m.ContainerPath,
			Readonly:      m.Readonly,
		})
	}
	return out
}

func rtHCOverridePtr(hc runplan.HealthCheckInput) *runplan.HealthCheckInput {
	if hc.Path == "" {
		return nil
	}
	return &hc
}

func (h *AgentHandler) HandleStartDeployment(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	operationID := uuid.NewString()
	ctx, opStart := log.StartOperation(r.Context(), "deployment.start",
		"deployment_id", deployID, "operation_id", operationID)
	_ = opStart // used at end with OperationCompleted

	if active := h.activeDeploymentRun(deployID); active.Blocked {
		writeJSON(w, http.StatusConflict, active)
		return
	}

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
		"accelerator_ids", pf.placement.AcceleratorIds,
		"request_id", log.RequestIDFromContext(r.Context()),
	)

	// Build GPU device ID list using NVIDIA indices (not internal UUIDs).
	gpuDeviceIDs := make([]string, len(pf.gpuInfos))
	for i, gi := range pf.gpuInfos {
		gpuDeviceIDs[i] = fmt.Sprintf("%d", gi.Index)
	}

	healthTimeoutSeconds := planHealthTimeout2(pf.bvHC)
	log.Info("preflight.health_timeout",
		"deployment_id", deployID,
		"backend_version_id", pf.rtVersionID,
		"health_timeout_seconds", healthTimeoutSeconds,
	)

	// Transaction: instance + runplan + lease + agent_task
	now := time.Now().Format(time.RFC3339)
	planJSON, _ := json.Marshal(pf.plan)
	agentSpec := agentruntime.ConvertRunplanToAgentSpec(agentruntime.PlanInput{
		OperationID:      operationID,
		InstanceID:       instanceID,
		DeploymentID:     deployID,
		NodeID:           pf.placement.NodeID,
		AgentID:          "",
		BackendName:      pf.backendName,
		Vendor:           pf.rtVendor,
		ModelPath:        pf.absolutePath,
		ServedModelName:  strVal(pf.params, "served_model_name", strVal(pf.artifact, "name", "")),
		Image:            pf.plan.Image,
		ContainerName:    pf.plan.ContainerName,
		Entrypoint:       pf.plan.Entrypoint,
		Args:             pf.plan.Args,
		Env:              pf.plan.Env,
		Privileged:       pf.plan.Privileged,
		IPCMode:          pf.plan.IPCMode,
		UTSMode:          pf.plan.UTSMode,
		NetworkMode:      pf.plan.NetworkMode,
		ShmSize:          pf.plan.ShmSize,
		Ulimits:          pf.plan.Ulimits,
		CapAdd:           pf.plan.CapAdd,
		CapDrop:          pf.plan.CapDrop,
		Devices:          agentPlanDevices(pf.plan.Devices),
		Mounts:           agentPlanMounts(pf.plan.Mounts),
		HostPort:         pf.plan.HostPort,
		ContainerPort:    pf.plan.ContainerPort,
		GPUDeviceIDs:     gpuDeviceIDs,
		GPUVisibleEnvKey: pf.plan.GPUVisibleEnvKey,
		GPUDriver:        pf.plan.GpuDriver,
		GPUCapabilities:  pf.plan.GpuCapabilities,
		SecurityOptions:  pf.plan.SecurityOptions,
		GroupAdd:         pf.plan.GroupAdd,
		HealthCheck: &agentruntime.PlanHealthCheck{
			Enabled:         pf.plan.HealthCheck.Path != "",
			Path:            pf.plan.HealthCheck.Path,
			Port:            firstPositive(pf.service.HealthPort, pf.plan.HostPort),
			Scheme:          "http",
			ExpectedStatus:  pf.plan.HealthCheck.ExpectedStatus,
			TimeoutSeconds:  healthTimeoutSeconds,
			IntervalSeconds: pf.plan.HealthCheck.IntervalSeconds,
		},
	})
	agentPayload, _ := json.Marshal(agentSpec)

	// BRR-E2E-001: Log host/container port mapping so health check URL can be traced.
	log.Info("deployment.start.agent_spec.ports",
		"deployment_id", deployID,
		"instance_id", instanceID,
		"host_port", pf.plan.HostPort,
		"container_port", pf.plan.ContainerPort,
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
		instanceID, deployID, tid, 0, pf.placement.NodeID, "", jsonString(pf.placement.AcceleratorIds), pf.plan.HostPort, pf.plan.ContainerPort, runPlanID, "pending", "running", now, now); err != nil {
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

	for _, gid := range pf.placement.AcceleratorIds {
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
		Action: "instance.start.requested", ResourceType: "model_instance",
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

// successStatusFromRaw extracts success_status array from raw health check JSON.
func successStatusFromRaw(raw string) []int {
	var m map[string]interface{}
	if json.Unmarshal([]byte(raw), &m) != nil {
		return nil
	}
	ss, _ := m["success_status"].([]interface{})
	var out []int
	for _, v := range ss {
		if n, ok := v.(float64); ok {
			out = append(out, int(n))
		}
	}
	return out
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

	// Classify known log patterns.
	classifier := runplan.NewRuntimeLogClassifier()
	classifiedEvents := classifier.ClassifyLogText(logsText + "\n" + stderr)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":                    runPlanID,
		"task_id":               taskID,
		"deployment_id":         deploymentID,
		"instance_id":           instanceID,
		"node_id":               nodeID,
		"container_id":          strVal(result, "container_id", containerID),
		"tail":                  tail,
		"since":                 since,
		"status":                "ok",
		"runtime_state":         strVal(result, "runtime_state", "ok"),
		"stdout":                stdout,
		"stderr":                stderr,
		"logs":                  logsText,
		"classified_log_events": classifiedEvents,
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
	row := h.DB.QueryRow(`SELECT id, deployment_id, tenant_id, node_id, container_id, actual_state, desired_state, endpoint_url, host_port, container_port, COALESCE(current_run_plan_id,''), last_error, started_at, stopped_at, created_at, updated_at FROM model_instances WHERE id = ?`, id)
	var rid, did, tid, as, ds, ca string
	var nid, cid, eu, rpid, le, sa, soa, ua sql.NullString
	var hp, cp int
	if err := row.Scan(&rid, &did, &tid, &nid, &cid, &as, &ds, &eu, &hp, &cp, &rpid, &le, &sa, &soa, &ca, &ua); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, tid) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"id": rid, "deployment_id": did, "tenant_id": tid, "node_id": nid.String, "container_id": cid.String, "actual_state": as, "desired_state": ds, "endpoint_url": eu.String, "host_port": hp, "container_port": cp, "current_run_plan_id": rpid.String, "last_error": le.String, "started_at": sa.String, "stopped_at": soa.String, "created_at": ca, "updated_at": ua.String})
}

// ==========================================================================
// Dry Run
// ==========================================================================

// planHashStr computes a simple hash of a string for config comparison.
func planHashStr(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

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
		"valid":  valid,
		"errors": pf.errs, "error_details": func() []string {
			var s []string
			for _, e := range pf.errs {
				s = append(s, e.Message)
			}
			return s
		}(),
		"warnings": pf.warns,
	}
	if pf.plan != nil {
		result["run_plan"] = pf.plan
		result["command_preview"] = pf.commandPreview
		result["selected_node"] = pf.placement.NodeID
		result["selected_runtime"] = pf.runtimeID
		result["selected_model_location"] = pf.locationID
		if pf.plan.Image != "" {
			result["resolved_image"] = pf.plan.Image
		}
	}
	if pf.lintResult != nil {
		result["lint"] = pf.lintResult
		// Merge lint errors/warnings into top-level for backward compatibility.
		for _, f := range pf.lintResult.Findings {
			switch f.Severity {
			case runplan.LintSeverityError:
				pf.warns = append(pf.warns, fmt.Sprintf("[lint] %s: %s", f.ID, f.Message))
			case runplan.LintSeverityWarning, runplan.LintSeverityAdvisory:
				pf.warns = append(pf.warns, fmt.Sprintf("[lint] %s: %s", f.ID, f.Message))
			}
		}
		result["warnings"] = pf.warns
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
// Template Sync: Preview & Apply
// ==========================================================================

// TemplateSyncDiff represents a single field difference between deployment and template.
type TemplateSyncDiff struct {
	Field         string      `json:"field"`
	DeployValue   interface{} `json:"deploy_value"`
	TemplateValue interface{} `json:"template_value"`
	AppliedValue  interface{} `json:"applied_value"`
	UserModified  bool        `json:"user_modified"`
	Conflict      bool        `json:"conflict"`
}

// HandleDeploymentTemplateSyncPreview compares the deployment current config
// with the source runtime template current config and returns a diff.
func (h *AgentHandler) HandleDeploymentTemplateSyncPreview(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	deploy := h.getDeploymentJSON(deployID)
	if deploy == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, deploy["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	sourceRuntimeID := strVal(deploy, "source_backend_runtime_id", "")
	if sourceRuntimeID == "" {
		sourceRuntimeID = strVal(deploy, "backend_runtime_id", "")
	}

	// Check if source template still exists
	sourceRT := h.getBackendRuntimeJSON(sourceRuntimeID)
	sourceExists := sourceRT != nil

	// Build current template ConfigSet.
	currentTemplateConfigSet := "{}"
	if sourceExists {
		currentTemplateConfigSet = h.buildDeploymentRuntimeSnapshot(sourceRuntimeID)
	}

	diffs := computeConfigSetDiffs(rawJSONString(deploy["config_set_json"], "{}"), currentTemplateConfigSet)

	currentTemplateHash := ""
	if currentTemplateConfigSet != "{}" {
		currentTemplateHash = planHashStr(currentTemplateConfigSet)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id":           deployID,
		"source_runtime_id":       sourceRuntimeID,
		"source_exists":           sourceExists,
		"source_template_name":    strVal(deploy, "source_template_name", ""),
		"source_template_version": strVal(deploy, "source_template_version", ""),
		"copied_at":               strVal(deploy, "copied_at", ""),
		"original_config_hash":    strVal(deploy, "source_config_hash", ""),
		"current_template_hash":   currentTemplateHash,
		"template_changed":        strVal(deploy, "source_config_hash", "") != currentTemplateHash,
		"diffs":                   diffs,
		"changed_fields":          changedFieldsFromDiffs(diffs),
		"conflicted_fields":       conflictedFieldsFromDiffs(diffs),
	})
}

// HandleDeploymentTemplateSyncApply applies template changes to the deployment.
// Supports preserve_overrides and reset_to_template strategies.
func (h *AgentHandler) HandleDeploymentTemplateSyncApply(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("id")
	deploy := h.getDeploymentJSON(deployID)
	if deploy == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !tenantScopeCheck(r, deploy["tenant_id"].(string)) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	strategy := strVal(req, "strategy", "preserve_overrides")
	sourceRuntimeID := strVal(deploy, "source_backend_runtime_id", "")
	if sourceRuntimeID == "" {
		sourceRuntimeID = strVal(deploy, "backend_runtime_id", "")
	}

	sourceRT := h.getBackendRuntimeJSON(sourceRuntimeID)
	if sourceRT == nil {
		writeError(w, http.StatusBadRequest, "source runtime template not found")
		return
	}

	newConfigSet := h.buildDeploymentRuntimeSnapshot(sourceRuntimeID)
	newHash := planHashStr(newConfigSet)
	diffs := computeConfigSetDiffs(rawJSONString(deploy["config_set_json"], "{}"), newConfigSet)

	changedFields := []string{}
	conflictedFields := []string{}

	for _, d := range diffs {
		if d.Conflict {
			conflictedFields = append(conflictedFields, d.Field)
			continue
		}
		if strategy == "preserve_overrides" && d.UserModified {
			continue
		}
		changedFields = append(changedFields, d.Field)
	}

	now := time.Now().Format(time.RFC3339)
	h.DB.Exec("UPDATE model_deployments SET config_set_json = ?, source_config_hash = ?, updated_at = ? WHERE id = ?",
		newConfigSet, newHash, now, deployID)

	tid := deploy["tenant_id"].(string)
	actorID := actorIDFromSession(r)
	WriteAudit(r.Context(), h.DB.DB, AuditEntry{
		TenantID: tid, ActorID: actorID,
		Action: "deployment.template_sync", ResourceType: "deployment",
		ResourceID: deployID, Result: "success",
		RequestID: log.RequestIDFromContext(r.Context()),
		Detail:    fmt.Sprintf("strategy=%s changed=%v conflicted=%v", strategy, changedFields, conflictedFields),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":            "synced",
		"deployment_id":     deployID,
		"strategy":          strategy,
		"changed_fields":    changedFields,
		"conflicted_fields": conflictedFields,
		"new_config_hash":   newHash,
		"updated_at":        now,
		"diffs":             diffs,
	})
}

func computeConfigSetDiffs(oldConfigSet, newConfigSet string) []TemplateSyncDiff {
	var diffs []TemplateSyncDiff
	oldItems := configSetItems(parseConfigSet(oldConfigSet))
	newItems := configSetItems(parseConfigSet(newConfigSet))
	seen := map[string]bool{}
	for field, oldItem := range oldItems {
		seen[field] = true
		newItem := newItems[field]
		if fmt.Sprintf("%v", oldItem) != fmt.Sprintf("%v", newItem) {
			diffs = append(diffs, TemplateSyncDiff{
				Field:         field,
				DeployValue:   oldItem,
				TemplateValue: newItem,
				AppliedValue:  newItem,
				UserModified:  false,
				Conflict:      false,
			})
		}
	}
	for field, newItem := range newItems {
		if seen[field] {
			continue
		}
		oldItem := oldItems[field]
		diffs = append(diffs, TemplateSyncDiff{
			Field:         field,
			DeployValue:   oldItem,
			TemplateValue: newItem,
			AppliedValue:  newItem,
			UserModified:  false,
			Conflict:      false,
		})
	}
	return diffs
}

func changedFieldsFromDiffs(diffs []TemplateSyncDiff) []string {
	var out []string
	for _, d := range diffs {
		if !d.Conflict {
			out = append(out, d.Field)
		}
	}
	return out
}

func conflictedFieldsFromDiffs(diffs []TemplateSyncDiff) []string {
	var out []string
	for _, d := range diffs {
		if d.Conflict {
			out = append(out, d.Field)
		}
	}
	return out
}

func parseJSONMap(raw string) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

// ==========================================================================
// Helpers
// ==========================================================================

func (h *AgentHandler) getDeploymentJSON(id string) map[string]interface{} {
	row := h.DB.QueryRow(deploymentSelectSQL()+` WHERE md.id = ?`, id)
	var rid, name, dn, desc, maid, modelName, modelDisplay, rtid, pj, sj, coj, configSetRaw, sourceMetaRaw, sbrid, snbrid, stn, stv, sch, copiedAt, ds, status, tid, ca, ua, nbrDisplay string
	var replicas int
	if err := row.Scan(&rid, &name, &dn, &desc, &maid, &modelName, &modelDisplay, &rtid, &replicas, &pj, &sj, &coj, &configSetRaw, &sourceMetaRaw, &sbrid, &snbrid, &stn, &stv, &sch, &copiedAt, &ds, &status, &tid, &ca, &ua, &nbrDisplay); err != nil {
		return nil
	}
	configSet := parseConfigSet(configSetRaw)
	return map[string]interface{}{"id": rid, "name": name, "display_name": dn, "description": desc, "model_artifact_id": maid, "model_name": modelName, "model_display_name": modelDisplay, "backend_runtime_id": rtid, "replicas": replicas, "placement_json": json.RawMessage(pj), "service_json": json.RawMessage(sj), "config_overrides_json": json.RawMessage(coj), "config_overrides": mapFromAny(coj), "config_set": configSet, "config_set_json": json.RawMessage(configSetRaw), "source_metadata": configSourceMetadata(sourceMetaRaw), "source_metadata_json": json.RawMessage(sourceMetaRaw), "source_backend_runtime_id": sbrid, "source_node_backend_runtime_id": snbrid, "source_node_backend_runtime_display_name": nbrDisplay, "source_template_name": stn, "source_template_version": stv, "source_config_hash": sch, "copied_at": copiedAt, "desired_state": ds, "status": status, "tenant_id": tid, "created_at": ca, "updated_at": ua}
}

func (h *AgentHandler) queryDeployments(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var rid, name, dn, desc, maid, modelName, modelDisplay, rtid, pj, sj, coj, configSetRaw, sourceMetaRaw, sbrid, snbrid, stn, stv, sch, copiedAt, ds, status, tid, ca, ua, nbrDisplay string
		var replicas int
		if err := rows.Scan(&rid, &name, &dn, &desc, &maid, &modelName, &modelDisplay, &rtid, &replicas, &pj, &sj, &coj, &configSetRaw, &sourceMetaRaw, &sbrid, &snbrid, &stn, &stv, &sch, &copiedAt, &ds, &status, &tid, &ca, &ua, &nbrDisplay); err != nil {
			continue
		}
		configSet := parseConfigSet(configSetRaw)
		out = append(out, map[string]interface{}{"id": rid, "name": name, "display_name": dn, "description": desc, "model_artifact_id": maid, "model_name": modelName, "model_display_name": modelDisplay, "backend_runtime_id": rtid, "replicas": replicas, "placement_json": json.RawMessage(pj), "service_json": json.RawMessage(sj), "config_overrides_json": json.RawMessage(coj), "config_overrides": mapFromAny(coj), "config_set": configSet, "config_set_json": json.RawMessage(configSetRaw), "source_metadata": configSourceMetadata(sourceMetaRaw), "source_metadata_json": json.RawMessage(sourceMetaRaw), "source_backend_runtime_id": sbrid, "source_node_backend_runtime_id": snbrid, "source_node_backend_runtime_display_name": nbrDisplay, "source_template_name": stn, "source_template_version": stv, "source_config_hash": sch, "copied_at": copiedAt, "desired_state": ds, "status": status, "tenant_id": tid, "created_at": ca, "updated_at": ua})
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	return out, nil
}

func deploymentSelectSQL() string {
	return `SELECT md.id, md.name, md.display_name, md.description, md.model_artifact_id, COALESCE(ma.name,''), COALESCE(ma.display_name,''), md.backend_runtime_id, md.replicas, md.placement_json, md.service_json, md.config_overrides_json, md.config_set_json, md.source_metadata_json, COALESCE(md.source_backend_runtime_id,''), COALESCE(md.source_node_backend_runtime_id,''), COALESCE(md.source_template_name,''), COALESCE(md.source_template_version,''), COALESCE(md.source_config_hash,''), COALESCE(md.copied_at,''), md.desired_state, md.status, md.tenant_id, md.created_at, md.updated_at, COALESCE(nbr.display_name,'') AS source_node_backend_runtime_display_name FROM model_deployments md LEFT JOIN model_artifacts ma ON ma.id = md.model_artifact_id LEFT JOIN node_backend_runtimes nbr ON nbr.id = md.source_node_backend_runtime_id`
}

// ==========================================================================
// Model Instance Smoke Test
// ==========================================================================

// HandleModelInstanceTest executes a smoke-test inference request against a
// running instance. It resolves the runtime model id by querying /v1/models,
// then attempts /v1/chat/completions with fallback to /v1/completions.
func (h *AgentHandler) HandleModelInstanceTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tid := tenantID(r)
	actorID := actorIDFromSession(r)
	requestID := log.RequestIDFromContext(r.Context())
	checkedAt := time.Now().Format(time.RFC3339)

	// Read instance.
	var instID, deployID, instTid, instState string
	var hostPort int
	var endpointURL sql.NullString
	row := h.DB.QueryRow(`SELECT id, deployment_id, tenant_id, actual_state, endpoint_url, host_port FROM model_instances WHERE id = ?`, id)
	if err := row.Scan(&instID, &deployID, &instTid, &instState, &endpointURL, &hostPort); err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	if !tenantScopeCheck(r, instTid) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if instState != "running" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "reason_code": "instance_not_running",
			"message": fmt.Sprintf("instance is %s, must be running to test", instState),
		})
		return
	}

	// Read artifact info for model id resolution.
	var artifactName, artifactPath string
	h.DB.QueryRow(`SELECT COALESCE(ma.name,''), COALESCE(ma.path,'') FROM model_deployments md JOIN model_artifacts ma ON ma.id = md.model_artifact_id WHERE md.id = ?`, deployID).Scan(&artifactName, &artifactPath)
	if artifactName == "" {
		artifactName = "unknown"
	}

	// Resolve endpoint.
	endpoint := endpointURL.String
	if endpoint == "" && hostPort > 0 {
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", hostPort)
	}
	if endpoint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "reason_code": "no_endpoint",
			"message": "instance has no endpoint URL or host port",
		})
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var testReq struct {
		Mode   string `json:"mode"`
		Prompt string `json:"prompt"`
	}
	_ = json.NewDecoder(r.Body).Decode(&testReq)
	if testReq.Mode == "" {
		testReq.Mode = "auto"
	}
	if testReq.Prompt == "" {
		testReq.Prompt = "Reply with exactly one word: pong"
	}

	// --- Phase 1: Resolve model id from /v1/models ---
	// 1. Try RunPlan / NodeRunPlan resolved model name.
	var runplanJSON string
	var runplanModel string
	h.DB.QueryRow(`SELECT COALESCE(rp.plan_json,'{}') FROM resolved_run_plans rp JOIN model_instances mi ON mi.current_run_plan_id = rp.id WHERE mi.deployment_id = ? ORDER BY rp.created_at DESC LIMIT 1`, deployID).Scan(&runplanJSON)
	if strings.TrimSpace(runplanJSON) != "" {
		var plan struct {
			ModelName       string `json:"model_name"`
			ServedModelName string `json:"served_model_name"`
		}
		if json.Unmarshal([]byte(runplanJSON), &plan) == nil {
			runplanModel = plan.ServedModelName
			if runplanModel == "" {
				runplanModel = plan.ModelName
			}
		}
	}

	modelName, resolutionMethod, availableModels := resolveModelID(client, endpoint, artifactName, artifactPath, runplanModel)

	if modelName == "" {
		resp := map[string]interface{}{
			"ok": false, "reason_code": "model_id_not_resolved",
			"message":                 "could not resolve model id from /v1/models or artifact",
			"model_resolution_method": resolutionMethod,
			"requested_model":         runplanModel,
			"available_models":        availableModels,
			"checked_at":              checkedAt,
		}
		if runplanModel != "" && len(availableModels) > 0 {
			resp["hint"] = "The requested model name does not match any model served by the runtime. Add --served-model-name to the vLLM/SGLang launch command, or use an available model id."
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// If the resolved model differs from the requested runplan model, include
	// diagnostics so the caller can see the mismatch.
	_ = runplanModel != "" && modelName != runplanModel

	// --- Phase 2: Attempt chat/completions, fallback to completions ---
	WriteAudit(r.Context(), h.DB.DB, AuditEntry{
		TenantID: tid, ActorID: actorID,
		Action: "model_instance.test.started", ResourceType: "instance",
		ResourceID: id, Result: "success",
		RequestID: requestID,
		Detail:    fmt.Sprintf("endpoint=%s model=%s method=%s", endpoint, modelName, resolutionMethod),
	})

	result := tryInferenceWithMode(client, endpoint, modelName, testReq.Mode, testReq.Prompt)
	result["model_resolution_method"] = resolutionMethod
	result["checked_at"] = checkedAt
	if runplanModel != "" {
		result["requested_model"] = runplanModel
	}
	if len(availableModels) > 0 {
		result["available_models"] = availableModels
	}

	// Phase 1: collect diagnostic probes on failure for richer error context.
	if result["ok"] != true {
		diag := collectTestDiagnostics(client, endpoint, deployID, h)
		for k, v := range diag {
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}
	}

	if result["ok"] == true {
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "model_instance.test.succeeded", ResourceType: "instance",
			ResourceID: id, Result: "success",
			RequestID: requestID,
			Detail:    fmt.Sprintf("endpoint=%s model=%s mode=%s latency_ms=%v", result["endpoint"], result["model"], result["mode"], result["latency_ms"]),
		})
	} else {
		WriteAudit(r.Context(), h.DB.DB, AuditEntry{
			TenantID: tid, ActorID: actorID,
			Action: "model_instance.test.failed", ResourceType: "instance",
			ResourceID: id, Result: "failure",
			RequestID: requestID,
			Detail:    fmt.Sprintf("reason=%s endpoint=%s latency_ms=%v", result["reason_code"], result["endpoint"], result["latency_ms"]),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// collectTestDiagnostics gathers endpoint and runtime diagnostic information
// when an instance test fails. Phase 1: /v1/models probe, /health probe,
// backend name, and runtime image.
func collectTestDiagnostics(client *http.Client, endpoint, deployID string, h *AgentHandler) map[string]interface{} {
	diag := map[string]interface{}{}

	// Probe /v1/models
	modelsURL := endpoint + "/v1/models"
	if !strings.HasSuffix(endpoint, "/") {
		modelsURL = endpoint + "/v1/models"
	}
	modelsResp, modelsErr := client.Get(modelsURL)
	if modelsErr != nil {
		diag["models_probe"] = map[string]interface{}{
			"ok": false, "error": fmt.Sprintf("failed to reach /v1/models: %v", modelsErr),
		}
	} else {
		defer modelsResp.Body.Close()
		bodyBytes, _ := io.ReadAll(io.LimitReader(modelsResp.Body, 4096))
		diag["models_probe"] = map[string]interface{}{
			"ok":          modelsResp.StatusCode >= 200 && modelsResp.StatusCode < 300,
			"status_code": modelsResp.StatusCode,
			"body":        string(bodyBytes)[:500],
		}
	}

	// Probe /health
	healthURL := endpoint + "/health"
	if !strings.HasSuffix(endpoint, "/") {
		healthURL = endpoint + "/health"
	}
	healthResp, healthErr := client.Get(healthURL)
	if healthErr != nil {
		diag["health_probe"] = map[string]interface{}{
			"ok": false, "error": fmt.Sprintf("failed to reach /health: %v", healthErr),
		}
	} else {
		defer healthResp.Body.Close()
		bodyBytes, _ := io.ReadAll(io.LimitReader(healthResp.Body, 1024))
		diag["health_probe"] = map[string]interface{}{
			"ok":          healthResp.StatusCode >= 200 && healthResp.StatusCode < 300,
			"status_code": healthResp.StatusCode,
			"body":        string(bodyBytes)[:200],
		}
	}

	// Collect runtime context: backend name, runtime image
	if deployID != "" {
		var backendName, runtimeConfigSetRaw string
		h.DB.QueryRow(
			`SELECT COALESCE(ib.name,''), COALESCE(br.config_set_json,'{}')
				 FROM model_deployments md
				 JOIN backend_runtimes br ON br.id = md.backend_runtime_id
				 JOIN inference_backends ib ON ib.id = br.backend_id
				 WHERE md.id = ?`, deployID,
		).Scan(&backendName, &runtimeConfigSetRaw)
		runtimeImage := configString(parseConfigSet(runtimeConfigSetRaw), "launcher.image", "")
		if backendName != "" {
			diag["backend"] = backendName
		}
		if runtimeImage != "" {
			diag["runtime_image"] = runtimeImage
		}
	}

	// Suggest diagnostic actions
	var suggestions []string
	if mp, ok := diag["models_probe"].(map[string]interface{}); ok {
		if ok, _ := mp["ok"].(bool); !ok {
			suggestions = append(suggestions, "/v1/models endpoint unreachable — verify container is running an OpenAI-compatible server and port mapping is correct")
		}
	}
	if hp, ok := diag["health_probe"].(map[string]interface{}); ok {
		if ok, _ := hp["ok"].(bool); !ok {
			suggestions = append(suggestions, "/health endpoint unreachable — container may not be ready or endpoint port is incorrect")
		}
	}
	if len(suggestions) > 0 {
		diag["suggestions"] = suggestions
	}

	return diag
}

// resolveModelID resolves the runtime model id by querying /v1/models and
// matching against the artifact name/path basename.
// Returns: (modelName, resolutionMethod, availableModels).
// Always probes /v1/models; runplanModel is only used if it appears in the
// runtime's actual /v1/models response (verified runplan priority).
func resolveModelID(client *http.Client, endpoint, artifactName, artifactPath, runplanModel string) (string, string, []string) {
	// 1. Always query /v1/models from the runtime.
	modelsURL := strings.TrimRight(endpoint, "/") + "/v1/models"
	httpReq, _ := http.NewRequest("GET", modelsURL, nil)
	resp, err := client.Do(httpReq)
	var modelIDs []string
	if err != nil {
		// /v1/models unreachable — fall back to artifact name.
		if runplanModel != "" {
			return runplanModel, "runplan_unverified", modelIDs
		}
		return artifactName, "artifact_name_fallback", modelIDs
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if runplanModel != "" {
			return runplanModel, "runplan_unverified", modelIDs
		}
		return artifactName, "artifact_name_fallback", modelIDs
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	var modelsResp map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &modelsResp); err != nil {
		if runplanModel != "" {
			return runplanModel, "runplan_unverified", modelIDs
		}
		return artifactName, "artifact_name_fallback", modelIDs
	}

	// Extract model ids.
	if data, ok := modelsResp["data"].([]interface{}); ok {
		for _, d := range data {
			if m, ok := d.(map[string]interface{}); ok {
				if mid, ok := m["id"].(string); ok && mid != "" {
					modelIDs = append(modelIDs, mid)
				}
			}
		}
	}
	if len(modelIDs) == 0 {
		if runplanModel != "" {
			return runplanModel, "runplan_unverified", modelIDs
		}
		return artifactName, "artifact_name_fallback", modelIDs
	}

	// 2. RunPlan model takes priority IF it exists in available models.
	if runplanModel != "" {
		for _, mid := range modelIDs {
			if mid == runplanModel {
				return runplanModel, "runplan", modelIDs
			}
		}
		// runplanModel not found in available models — fall through to matching.
	}

	// 3. Single model — use it directly.
	if len(modelIDs) == 1 {
		return modelIDs[0], "single_model_fallback", modelIDs
	}

	// 4. Multiple models — try to match.
	// Build candidates from artifact name, filename, path basename.
	candidates := []string{artifactName}
	if artifactPath != "" {
		base := artifactPath
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		if base != "" && base != artifactName {
			candidates = append(candidates, base)
		}
	}

	for _, c := range candidates {
		for _, mid := range modelIDs {
			if mid == c {
				return mid, "models_exact_match", modelIDs
			}
		}
	}
	for _, c := range candidates {
		cl := strings.ToLower(c)
		for _, mid := range modelIDs {
			if strings.Contains(strings.ToLower(mid), cl) || strings.Contains(cl, strings.ToLower(mid)) {
				return mid, "models_alias_match", modelIDs
			}
		}
	}

	// 5. Cannot match — fail with available models for diagnostics.
	return "", "model_id_not_resolved", modelIDs
}

// tryInference attempts chat/completions first, then falls back to completions
// if the endpoint returns 404/405 (unsupported). Does not fallback for real
// inference errors (OOM, auth failure, model load fail).
func tryInference(client *http.Client, endpoint, modelName string) map[string]interface{} {
	return tryInferenceWithMode(client, endpoint, modelName, "auto", "Reply with exactly one word: pong")
}

func tryInferenceWithMode(client *http.Client, endpoint, modelName, mode, prompt string) map[string]interface{} {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if prompt == "" {
		prompt = "Reply with exactly one word: pong"
	}
	if mode == "completion" {
		return tryCompletionInference(client, endpoint, modelName, prompt)
	}
	if mode == "chat" {
		return tryChatInference(client, endpoint, modelName, prompt, false)
	}
	if mode == "embedding" {
		return tryEmbeddingInference(client, endpoint, modelName)
	}
	if mode == "rerank" {
		return tryRerankInference(client, endpoint, modelName)
	}
	// Try chat/completions.
	chatResult := tryChatInference(client, endpoint, modelName, prompt, true)
	if chatResult["ok"] == true {
		return chatResult
	}
	if chatResult["reason_code"] != "chat_endpoint_unsupported" {
		return chatResult
	}
	return tryCompletionInference(client, endpoint, modelName, prompt)
}

func tryChatInference(client *http.Client, endpoint, modelName, prompt string, unsupportedAsFallback bool) map[string]interface{} {
	chatURL := strings.TrimRight(endpoint, "/") + "/v1/chat/completions"
	chatBody, _ := json.Marshal(map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": "Reply with exactly one word: pong"},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 8, "temperature": 0, "stream": false,
	})

	startTime := time.Now()
	httpReq, _ := http.NewRequest("POST", chatURL, bytes.NewReader(chatBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := client.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		return map[string]interface{}{"ok": false, "reason_code": "network_error", "message": fmt.Sprintf("failed to reach instance: %v", err), "endpoint": chatURL, "latency_ms": latencyMs}
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	resp.Body.Close()

	// Determine if this is a "method not supported" type error (404, 405).
	isEndpointErr := resp.StatusCode == 404 || resp.StatusCode == 405

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		preview := extractPreview(bodyBytes, "chat")
		if strings.TrimSpace(preview) == "" {
			return map[string]interface{}{
				"ok": false, "mode": "chat", "reason_code": "empty_model_response",
				"message":  "request succeeded but model response was empty",
				"endpoint": chatURL, "model": modelName,
				"latency_ms": latencyMs, "response_preview": preview,
				"raw_response": string(bodyBytes),
			}
		}
		resolvedModel := modelName
		var respData map[string]interface{}
		json.Unmarshal(bodyBytes, &respData)
		if m, ok := respData["model"].(string); ok && m != "" {
			resolvedModel = m
		}
		return map[string]interface{}{
			"ok": true, "mode": "chat",
			"endpoint": chatURL, "model": resolvedModel,
			"latency_ms": latencyMs, "response_preview": preview,
			"raw_response": string(bodyBytes),
		}
	}

	// If chat endpoint not supported, try completions.
	if isEndpointErr {
		if unsupportedAsFallback {
			return map[string]interface{}{"ok": false, "mode": "chat", "reason_code": "chat_endpoint_unsupported", "message": fmt.Sprintf("chat completions returned HTTP %d", resp.StatusCode), "endpoint": chatURL, "http_status": resp.StatusCode, "latency_ms": latencyMs}
		}
		return map[string]interface{}{"ok": false, "mode": "chat", "reason_code": "chat_endpoint_failed", "message": fmt.Sprintf("chat completions returned HTTP %d", resp.StatusCode), "endpoint": chatURL, "http_status": resp.StatusCode, "latency_ms": latencyMs, "raw_response": string(bodyBytes)}
	}

	// Real error (not endpoint-unsupported) — do not fallback.
	var respData map[string]interface{}
	json.Unmarshal(bodyBytes, &respData)
	errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if em, ok := respData["error"].(map[string]interface{}); ok {
		if m, ok := em["message"].(string); ok {
			errMsg = m
		}
	}
	return map[string]interface{}{"ok": false, "mode": "chat", "reason_code": "chat_endpoint_failed", "message": errMsg, "endpoint": chatURL, "http_status": resp.StatusCode, "latency_ms": latencyMs, "raw_response": string(bodyBytes)}
}

// tryEmbeddingInference sends an embedding test request to /v1/embeddings.
func tryEmbeddingInference(client *http.Client, endpoint, modelName string) map[string]interface{} {
	embURL := strings.TrimRight(endpoint, "/") + "/v1/embeddings"
	embBody, _ := json.Marshal(map[string]interface{}{
		"model": modelName,
		"input": "hello world",
	})
	startTime := time.Now()
	httpReq, _ := http.NewRequest("POST", embURL, bytes.NewReader(embBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := client.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()
	if err != nil {
		return map[string]interface{}{"ok": false, "mode": "embedding", "reason_code": "network_error", "message": err.Error(), "endpoint": embURL, "latency_ms": latencyMs}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var preview string
		if len(body) > 300 {
			preview = string(body[:300])
		} else {
			preview = string(body)
		}
		return map[string]interface{}{"ok": true, "mode": "embedding", "endpoint": embURL, "model": modelName, "latency_ms": latencyMs, "response_preview": preview, "raw_response": string(body), "checked_at": time.Now().Format(time.RFC3339)}
	}
	return map[string]interface{}{"ok": false, "mode": "embedding", "reason_code": "embedding_endpoint_failed", "http_status": resp.StatusCode, "endpoint": embURL, "model": modelName, "latency_ms": latencyMs, "error_body": string(body), "checked_at": time.Now().Format(time.RFC3339)}
}

// tryRerankInference sends a rerank test request to the declared rerank endpoint.
func tryRerankInference(client *http.Client, endpoint, modelName string) map[string]interface{} {
	rankURL := strings.TrimRight(endpoint, "/") + "/v1/rerank"
	rankBody, _ := json.Marshal(map[string]interface{}{
		"model": modelName,
		"query": "what is GPU",
		"documents": []string{
			"GPU is a processor for parallel computation.",
			"A database stores structured data.",
		},
	})
	startTime := time.Now()
	httpReq, _ := http.NewRequest("POST", rankURL, bytes.NewReader(rankBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := client.Do(httpReq)
	latencyMs := time.Since(startTime).Milliseconds()
	if err != nil {
		return map[string]interface{}{"ok": false, "mode": "rerank", "reason_code": "network_error", "message": err.Error(), "endpoint": rankURL, "latency_ms": latencyMs}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var preview string
		if len(body) > 300 {
			preview = string(body[:300])
		} else {
			preview = string(body)
		}
		return map[string]interface{}{"ok": true, "mode": "rerank", "endpoint": rankURL, "model": modelName, "latency_ms": latencyMs, "response_preview": preview, "raw_response": string(body), "checked_at": time.Now().Format(time.RFC3339)}
	}
	return map[string]interface{}{"ok": false, "mode": "rerank", "reason_code": "rerank_endpoint_failed", "http_status": resp.StatusCode, "endpoint": rankURL, "model": modelName, "latency_ms": latencyMs, "error_body": string(body), "checked_at": time.Now().Format(time.RFC3339)}
}

func tryCompletionInference(client *http.Client, endpoint, modelName, prompt string) map[string]interface{} {
	compURL := strings.TrimRight(endpoint, "/") + "/v1/completions"
	compBody, _ := json.Marshal(map[string]interface{}{
		"model":  modelName,
		"prompt": prompt, "max_tokens": 8, "temperature": 0, "stream": false,
	})
	compStart := time.Now()
	compReq, _ := http.NewRequest("POST", compURL, bytes.NewReader(compBody))
	compReq.Header.Set("Content-Type", "application/json")
	compReq.Header.Set("Accept", "application/json")
	compResp, compErr := client.Do(compReq)
	compLatency := time.Since(compStart).Milliseconds()

	if compErr != nil {
		return map[string]interface{}{"ok": false, "mode": "completion", "reason_code": "completion_endpoint_failed", "message": fmt.Sprintf("completions unreachable: %v", compErr), "endpoint": compURL, "latency_ms": compLatency}
	}
	compBytes, _ := io.ReadAll(io.LimitReader(compResp.Body, 8192))
	compResp.Body.Close()

	if compResp.StatusCode >= 200 && compResp.StatusCode < 300 {
		preview := extractPreview(compBytes, "completion")
		if strings.TrimSpace(preview) == "" {
			return map[string]interface{}{
				"ok": false, "mode": "completion", "reason_code": "empty_model_response",
				"message":  "request succeeded but model response was empty",
				"endpoint": compURL, "model": modelName,
				"latency_ms": compLatency, "response_preview": preview,
				"raw_response": string(compBytes),
			}
		}
		resolvedModel := modelName
		var compData map[string]interface{}
		json.Unmarshal(compBytes, &compData)
		if m, ok := compData["model"].(string); ok && m != "" {
			resolvedModel = m
		}
		return map[string]interface{}{
			"ok": true, "mode": "completion",
			"endpoint": compURL, "model": resolvedModel,
			"latency_ms": compLatency, "response_preview": preview,
			"raw_response": string(compBytes),
		}
	}
	return map[string]interface{}{"ok": false, "mode": "completion", "reason_code": "completion_endpoint_failed", "message": fmt.Sprintf("completions returned HTTP %d", compResp.StatusCode), "endpoint": compURL, "http_status": compResp.StatusCode, "latency_ms": compLatency, "raw_response": string(compBytes)}
}

// extractPreview extracts a short response preview from chat or completion JSON.
// Handles: choices[0].message.content, choices[0].message.reasoning_content,
// choices[0].text, choices[0].delta.content (stream fallback),
// top-level content/response/generated_text.
func extractPreview(bodyBytes []byte, mode string) string {
	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return ""
	}
	// Try choices array first (OpenAI-compatible format).
	choices, _ := data["choices"].([]interface{})
	if len(choices) > 0 {
		choice, _ := choices[0].(map[string]interface{})
		if mode == "chat" {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				// Prefer content, but use reasoning_content if content is empty.
				if c, ok := msg["content"]; ok {
					s := fmt.Sprintf("%v", c)
					if strings.TrimSpace(s) != "" {
						return s
					}
				}
				if rc, ok := msg["reasoning_content"]; ok {
					s := fmt.Sprintf("%v", rc)
					if strings.TrimSpace(s) != "" {
						return "[reasoning] " + s
					}
				}
			}
			// Stream delta fallback.
			if delta, ok := choice["delta"].(map[string]interface{}); ok {
				if c, ok := delta["content"]; ok {
					s := fmt.Sprintf("%v", c)
					if strings.TrimSpace(s) != "" {
						return s
					}
				}
			}
		}
		// text field (completions or plain).
		if t, ok := choice["text"]; ok {
			s := fmt.Sprintf("%v", t)
			if strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	// Top-level fields (non-OpenAI formats, e.g. llama.cpp native).
	for _, key := range []string{"content", "response", "generated_text"} {
		if v, ok := data[key]; ok {
			s := fmt.Sprintf("%v", v)
			if strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}
