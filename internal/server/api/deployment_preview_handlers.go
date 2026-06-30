package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"lightai-go/internal/server/runplan"
)

// HandleDeploymentPreview accepts a deployment create payload and returns a full
// resolution preview including RunPlan, lint results, Docker command preview,
// and preflight checks — without creating any database records.
// POST /api/v1/deployments/preview
func (h *AgentHandler) HandleDeploymentPreview(w http.ResponseWriter, r *http.Request) {
	var rawBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Reject legacy field.
	if _, ok := rawBody["backend_runtime_id"]; ok {
		writeError(w, http.StatusBadRequest,
			"BackendRuntime is a template and cannot be used for deployment. Use node_backend_runtime_id.")
		return
	}

	modelArtifactID := strVal(rawBody, "model_artifact_id", "")
	nbrID := strVal(rawBody, "node_backend_runtime_id", "")

	errs := []map[string]interface{}{}
	warns := []map[string]interface{}{}

	if modelArtifactID == "" {
		errs = append(errs, errEntry("missing_field", "model_artifact_id is required", "model_artifact_id", "error"))
	}
	if nbrID == "" {
		errs = append(errs, errEntry("missing_field", "node_backend_runtime_id is required", "node_backend_runtime_id", "error"))
	}
	if len(errs) > 0 {
		writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
		return
	}

	// Validate NBR.
	var nbrBackendRuntimeID, nbrNodeID, nbrStatus, nbrConfigSetRaw string
	if err := h.DB.QueryRow(
		`SELECT backend_runtime_id, node_id, status, COALESCE(config_set_json,'{}')
		 FROM node_backend_runtimes WHERE id = ?`, nbrID,
	).Scan(&nbrBackendRuntimeID, &nbrNodeID, &nbrStatus, &nbrConfigSetRaw); err != nil {
		writeError(w, http.StatusBadRequest, "node_backend_runtime_id not found")
		return
	}
	if !isNBRDeployable(nbrStatus) {
		reason := nbrDisabledReason(nbrStatus, "")
		errs = append(errs, errEntry("nbr_not_deployable",
			fmt.Sprintf("NBR status=%s: %s", nbrStatus, reason), "node_backend_runtime_id", "error"))
		writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
		return
	}
	if nbrStatus == "ready_with_warnings" {
		warns = append(warns, errEntry("nbr_ready_with_warnings",
			"NBR is ready with warnings — deployment may succeed but has non-blocking issues",
			"node_backend_runtime_id", "warning"))
	}

	// Validate artifact.
	artifact := h.getArtifactJSON(modelArtifactID)
	if artifact == nil {
		errs = append(errs, errEntry("model_not_found", "model artifact not found", "model_artifact_id", "error"))
		writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
		return
	}

	// Check model location.
	loc, _, reason := h.findDeployableModelLocation(modelArtifactID, nbrNodeID)
	if loc == nil {
		errs = append(errs, errEntry("model_location_missing",
			reason,
			"model_artifact_id", "error"))
		writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
		return
	}

	// Tenant scope.
	tid := tenantID(r)
	if !isPlatformAdmin(r) {
		var nodeTid string
		h.DB.QueryRow("SELECT tenant_id FROM nodes WHERE id = ?", nbrNodeID).Scan(&nodeTid)
		if nodeTid != tid && nodeTid != "" {
			errs = append(errs, errEntry("tenant_mismatch", "tenant does not have access", "node_id", "error"))
			writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
			return
		}
	}

	// GPU check (warning only).
	var gpuCount int
	h.DB.QueryRow("SELECT COUNT(*) FROM gpu_devices WHERE node_id = ? AND status = 'available'", nbrNodeID).Scan(&gpuCount)
	if gpuCount == 0 {
		warns = append(warns, errEntry("no_available_gpu", "no available GPU found on node", "node_id", "warning"))
	}

	// Host port conflict check.
	hostPort := 0
	if svc, ok := rawBody["service_json"].(map[string]interface{}); ok {
		switch v := svc["host_port"].(type) {
		case float64:
			hostPort = int(v)
		case int:
			hostPort = v
		}
	}
	if hostPort > 0 {
		var conflictCount int
		h.DB.QueryRow(`SELECT COUNT(*) FROM model_instances WHERE host_port = ? AND actual_state IN ('pending','starting','running')`, hostPort).Scan(&conflictCount)
		if conflictCount > 0 {
			errs = append(errs, errEntry("host_port_conflict",
				fmt.Sprintf("host port %d is already in use", hostPort), "service_json.host_port", "error"))
			writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
			return
		}
	}

	// Build merged ConfigSet from NBR snapshot + deployment overrides.
	nbrConfigSet := copyConfigSet(nbrConfigSetRaw)
	if overrides, ok := rawBody["config_overrides"].(map[string]interface{}); ok {
		applyConfigOverrides(nbrConfigSet, overrides, "deployment", "preview")
	}
	var patchErr error
	nbrConfigSet, patchErr = applyEditableConfigPatchIfPresent(nbrConfigSet, rawBody, "deployment", "preview")
	if patchErr != nil {
		errs = append(errs, errEntry("config_edit_patch_invalid", patchErr.Error(), "editable_config_patch", "error"))
		writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
		return
	}
	if placementRaw, ok := rawBody["placement_json"].(map[string]interface{}); ok {
		serviceRaw, _ := rawBody["service_json"].(map[string]interface{})
		materializeDeploymentCompatConfig(nbrConfigSet, placementRaw, serviceRaw, "deployment", "preview")
	} else if serviceRaw, ok := rawBody["service_json"].(map[string]interface{}); ok {
		materializeDeploymentCompatConfig(nbrConfigSet, map[string]interface{}{}, serviceRaw, "deployment", "preview")
	}

	// Get BackendRuntime for resolution.
	rtRow := h.getBackendRuntimeJSON(nbrBackendRuntimeID)
	if rtRow == nil {
		errs = append(errs, errEntry("runtime_not_found", "backend runtime not found", "node_backend_runtime_id", "error"))
		writeJSON(w, http.StatusOK, previewResponse(false, nil, "", runplan.LintResult{Status: "ok"}, errs, warns))
		return
	}

	backendID := strVal(rtRow, "backend_id", "")
	backendName := strings.TrimPrefix(backendID, "backend.")

	// Build resolver input.
	paramDefs := configSetParameterDefs(nbrConfigSet)
	paramVals := configSetParameterValues(nbrConfigSet)

	nbrSnapshot := &runplan.NBRSnapshotInfo{
		ArgsOverride:        configStringSlice(nbrConfigSet, "launcher.command"),
		DefaultEnv:          configStringMap(nbrConfigSet, "runtime.env"),
		EntrypointOverride:  configStringSlice(nbrConfigSet, "launcher.entrypoint"),
		Docker:              configDockerSpec(nbrConfigSet),
		ModelMount:          configModelMount(nbrConfigSet),
		HealthCheckOverride: configHealthCheckPtr(nbrConfigSet),
		DeviceBinding:       configDeviceBinding(nbrConfigSet),
		ServicePortBinding:  configServicePortBinding(nbrConfigSet),
		ParameterSchema:     paramDefs,
		ParameterValues:     paramVals,
	}

	// Placement — build GPUInfo list from accelerator_ids. Match start/dry-run:
	// if the user did not explicitly pick GPUs, preview the first available GPU.
	gpuIDs := []runplan.GPUInfo{}
	acceleratorIDs := []string{}
	placement := runplan.PlacementInfo{NodeID: nbrNodeID, AllowAutoSelect: true}
	if placementRaw, ok := rawBody["placement_json"].(map[string]interface{}); ok {
		if aids, ok := placementRaw["accelerator_ids"].([]interface{}); ok {
			for _, a := range aids {
				if s, ok := a.(string); ok {
					acceleratorIDs = append(acceleratorIDs, s)
				}
			}
		}
		placement.AcceleratorSelectionMode = strings.TrimSpace(strVal(placementRaw, "accelerator_selection_mode", ""))
		placement.AcceleratorCount = intFromAny(placementRaw["accelerator_count"], 0)
		placement.AllowAutoSelect = boolFromAny(placementRaw["allow_auto_select"], true)
		if _, ok := placementRaw["device_binding_enabled"]; ok {
			enabled := boolFromAny(placementRaw["device_binding_enabled"], true)
			placement.DeviceBindingEnabled = &enabled
		}
	}
	if placement.DeviceBindingEnabled != nil && !*placement.DeviceBindingEnabled {
		placement.AcceleratorSelectionMode = "disabled"
		acceleratorIDs = nil
	}
	if placement.AcceleratorSelectionMode == "disabled" {
		acceleratorIDs = nil
	}
	if len(acceleratorIDs) == 0 && nbrNodeID != "" && placement.AcceleratorSelectionMode != "disabled" {
		var autoGpuID string
		_ = h.DB.QueryRow(`SELECT id FROM gpu_devices WHERE node_id = ? AND status = 'available' LIMIT 1`, nbrNodeID).Scan(&autoGpuID)
		if autoGpuID != "" {
			acceleratorIDs = append(acceleratorIDs, autoGpuID)
			if placement.AcceleratorSelectionMode == "" {
				placement.AcceleratorSelectionMode = "auto"
			}
		}
	}
	if len(acceleratorIDs) > 0 && placement.AcceleratorSelectionMode == "" {
		placement.AcceleratorSelectionMode = "manual"
	}
	placement.AcceleratorIds = acceleratorIDs
	if placement.AcceleratorCount == 0 {
		placement.AcceleratorCount = len(acceleratorIDs)
	}
	for _, gid := range acceleratorIDs {
		var idx int
		var vendor string
		if err := h.DB.QueryRow(`SELECT index_num, vendor FROM gpu_devices WHERE id = ?`, gid).Scan(&idx, &vendor); err == nil {
			gpuIDs = append(gpuIDs, runplan.GPUInfo{Index: idx, Vendor: vendor})
		}
	}

	input := runplan.ResolveInput{
		Backend: &runplan.BackendInfo{
			ID:   backendID,
			Name: backendName,
		},
		BackendVersion: &runplan.VersionInfo{
			ID: strVal(rtRow, "backend_version_id", ""),
		},
		BackendRuntime: &runplan.RuntimeInfo{
			ID:                  nbrBackendRuntimeID,
			Vendor:              strVal(rtRow, "vendor", ""),
			RuntimeType:         strVal(rtRow, "runtime_type", ""),
			LauncherKind:        configLauncherKind(nbrConfigSet, strVal(rtRow, "runtime_type", "")),
			ImageName:           configString(nbrConfigSet, "launcher.image", ""),
			EntrypointOverride:  configStringSlice(nbrConfigSet, "launcher.entrypoint"),
			ArgsOverride:        configStringSlice(nbrConfigSet, "launcher.command"),
			DefaultEnv:          configStringMap(nbrConfigSet, "runtime.env"),
			Docker:              configDockerSpec(nbrConfigSet),
			ModelMount:          configModelMount(nbrConfigSet),
			HealthCheckOverride: configHealthCheckPtr(nbrConfigSet),
		},
		Artifact: &runplan.ArtifactInfo{
			ID:           modelArtifactID,
			Name:         firstNonEmpty(strVal(artifact, "display_name", ""), strVal(artifact, "name", "")),
			Path:         firstNonEmpty(strVal(loc, "absolute_path", ""), strVal(artifact, "path", "")),
			ModelRoot:    strVal(loc, "model_root", ""),
			RelativePath: strVal(loc, "relative_path", ""),
		},
		Deployment: &runplan.DeploymentInfo{
			ID:              "preview",
			Name:            strVal(rawBody, "name", "preview"),
			ParameterValues: paramVals,
			Placement:       placement,
		},
		InstanceID:        "preview",
		Node:              &runplan.NodeInfo{ID: nbrNodeID},
		AssignedGPUs:      gpuIDs,
		NBRConfigSnapshot: nbrSnapshot,
	}
	serviceJSON, _ := rawBody["service_json"].(map[string]interface{})
	input = runplan.ApplySemanticSnapshot(input, semanticDeploymentSnapshot(nbrConfigSet, serviceJSON), backendName)

	plan, resolveErrs, resolveWarns := runplan.ResolveWithSourceMap(input)
	for _, e := range resolveErrs {
		blocking := plan == nil
		entry := errEntry("resolve_error", e.Error(), "runplan", "warning")
		entry["source"] = "runplan_resolver"
		entry["blocking"] = blocking
		if blocking {
			entry["severity"] = "error"
			errs = append(errs, entry)
		} else {
			entry["severity"] = "warning"
			warns = append(warns, entry)
		}
	}
	for _, w := range resolveWarns {
		warns = append(warns, errEntry("resolve_warning", w, "runplan", "warning"))
	}

	// Lint.
	var finalArgs []string
	var lintDockerSpec *runplan.DockerSpecInfo
	envMap := map[string]string{}
	if plan != nil {
		finalArgs = plan.Args
		envMap = plan.Env
		lintDockerSpec = &runplan.DockerSpecInfo{
			Privileged:      plan.Privileged,
			IPCMode:         plan.IPCMode,
			SecurityOptions: plan.SecurityOptions,
		}
	}
	lintInput := runplan.LintInput{
		FinalArgs:           finalArgs,
		Env:                 envMap,
		PlatformOwnedParams: runplan.DefaultLogicalParamSpecs(),
		BackendName:         backendName,
		BackendArgsSchema:   defsToFlags(paramDefs),
		Vendor:              strVal(rtRow, "vendor", ""),
		DockerSpec:          lintDockerSpec,
	}
	lintResult := runplan.LintRunPlan(lintInput)

	// Docker command preview.
	dockerPreview := ""
	if plan != nil {
		dockerPreview = runplan.EquivalentCommandPreview(plan)
	}

	errs = dedupeIssueEntries(errs)
	warns = dedupeIssueEntries(warns)
	canRun := plan != nil && !hasBlockingIssue(errs)
	writeJSON(w, http.StatusOK, previewResponse(canRun, plan, dockerPreview, lintResult, errs, warns))
}

// previewResponse builds the standard preview response.
func previewResponse(canRun bool, plan *runplan.ResolvedRunPlan, dockerPreview string,
	lintResult runplan.LintResult, errors, warnings []map[string]interface{}) map[string]interface{} {

	if errors == nil {
		errors = []map[string]interface{}{}
	}
	if warnings == nil {
		warnings = []map[string]interface{}{}
	}

	resp := map[string]interface{}{
		"can_run":        canRun,
		"docker_preview": dockerPreview,
		"lint":           lintResult,
		"resource_admission": map[string]interface{}{
			"status":   "ok",
			"findings": []interface{}{},
		},
		"preflight": map[string]interface{}{
			"status":   lintResult.Status,
			"errors":   errors,
			"warnings": warnings,
		},
	}
	if plan != nil {
		resp["run_plan"] = plan
	}
	return resp
}

func hasBlockingIssue(items []map[string]interface{}) bool {
	for _, item := range items {
		if b, ok := item["blocking"].(bool); ok {
			if b {
				return true
			}
			continue
		}
		if item["severity"] == "error" {
			return true
		}
	}
	return false
}

func dedupeIssueEntries(items []map[string]interface{}) []map[string]interface{} {
	seen := map[string]bool{}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%v|%v|%v|%v|%v", item["code"], item["key"], item["path"], item["reason"], item["source"])
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

// hostPortFromJSON extracts an integer from a JSON map.
func hostPortFromJSON(dst *int, m map[string]interface{}, key string) {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			*dst = int(val)
		case int:
			*dst = val
		}
	}
}

// defsToFlags extracts CLI flag names from parameter definitions.
func defsToFlags(defs []runplan.ParameterDef) []string {
	flags := make([]string, 0, len(defs))
	for _, d := range defs {
		if d.CliName != "" {
			flags = append(flags, d.CliName)
		}
	}
	return flags
}
