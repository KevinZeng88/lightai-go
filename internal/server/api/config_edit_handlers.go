package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"lightai-go/internal/server/catalog"
	"lightai-go/internal/server/configedit"
)

func (h *AgentHandler) HandleConfigEditView(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	kind := strVal(req, "object_kind", "")
	id := strVal(req, "object_id", "")
	layer := strVal(req, "layer", kind)
	if kind == "" || id == "" {
		writeError(w, http.StatusBadRequest, "object_kind and object_id required")
		return
	}
	obj, err := h.configEditObject(kind, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "object not found")
		return
	}
	configSetRaw := rawJSONString(obj["config_set_json"], "{}")
	view, err := configedit.ProjectConfigSetToEditView(configedit.ProjectInput{
		ConfigSet:   copyConfigSet(configSetRaw),
		Layer:       layer,
		ObjectKind:  kind,
		ObjectID:    id,
		ObjectLabel: strVal(obj, "display_name", strVal(obj, "name", id)),
		Readonly:    boolVal(obj, "readonly", false),
		Mode:        strVal(req, "mode", "edit"),
		ViewLevel:   strVal(req, "view_level", ""),
		TemplateID:  strVal(obj, "template_id", ""),
		SnapshotID:  strVal(obj, "snapshot_id", ""),
		Parent:      objectRefFromAny(obj["parent"]),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Generate ConfigView from ConfigSet for tiered presentation
	var configView *catalog.ConfigView
	var cs catalog.ConfigSet
	if err := json.Unmarshal([]byte(configSetRaw), &cs); err == nil {
		cv := cs.GenerateView()
		// Populate child panels from the bundle
		for i, slot := range cs.ChildSlots {
			if i < len(cv.ChildPanels) {
				cv.ChildPanels[i].Slot = slot.Slot
				cv.ChildPanels[i].Title = slot.Title
			}
		}
		configView = &cv
	}

	response := map[string]interface{}{
		"config_edit_view": view,
	}
	if configView != nil {
		response["config_view"] = configView
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AgentHandler) HandleConfigEditApply(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	kind := strVal(req, "object_kind", "")
	id := strVal(req, "object_id", "")
	layer := strVal(req, "layer", kind)
	if kind == "" || id == "" {
		writeError(w, http.StatusBadRequest, "object_kind and object_id required")
		return
	}
	obj, err := h.configEditObject(kind, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "object not found")
		return
	}
	if boolVal(obj, "readonly", false) {
		writeError(w, http.StatusForbidden, "object is readonly")
		return
	}
	patch, err := decodeConfigEditPatch(req["patch"])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if patch.Layer == "" {
		patch.Layer = layer
	}
	if patch.ObjectID == "" {
		patch.ObjectID = id
	}
	currentSet := copyConfigSet(rawJSONString(obj["config_set_json"], "{}"))
	var out map[string]any
	if resetRaw, ok := req["reset"].(map[string]interface{}); ok {
		key := strVal(resetRaw, "internal_key", strVal(resetRaw, "key", ""))
		if key == "" {
			writeError(w, http.StatusBadRequest, "reset key required")
			return
		}
		path := stringSliceFromAny(resetRaw["path"])
		switch strVal(resetRaw, "mode", "default") {
		case "parent":
			out, err = configedit.ResetFieldToParent(currentSet, key, path, layer, id)
		default:
			out, err = configedit.ResetFieldToDefault(currentSet, key, path, layer, id)
		}
	} else {
		out, err = configedit.ApplyEditPatchToConfigSet(currentSet, patch, layer, id)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.updateConfigEditObject(kind, id, configSetJSON(out)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"config_set": out})
}

func (h *AgentHandler) configEditObject(kind, id string) (map[string]interface{}, error) {
	switch kind {
	case "backend_version":
		return h.queryConfigEditObject(`SELECT id, version AS name, display_name, config_set_json, CASE WHEN readonly=1 OR managed_by='system' THEN 1 ELSE 0 END AS readonly, backend_id AS parent_id, '' AS parent_kind, COALESCE(checksum,'') AS snapshot_id, '' AS template_id FROM backend_versions WHERE id=?`, id)
	case "backend_runtime":
		return h.queryConfigEditObject(`SELECT id, name, display_name, config_set_json, CASE WHEN is_editable=0 THEN 1 ELSE 0 END AS readonly, backend_version_id AS parent_id, 'backend_version' AS parent_kind, COALESCE(checksum,'') AS snapshot_id, '' AS template_id FROM backend_runtimes WHERE id=?`, id)
	case "node_backend_runtime":
		return h.queryConfigEditObject(`SELECT id, id AS name, display_name, config_set_json, 0 AS readonly, backend_runtime_id AS parent_id, 'backend_runtime' AS parent_kind, '' AS snapshot_id, '' AS template_id FROM node_backend_runtimes WHERE id=?`, id)
	case "deployment":
		obj, err := h.queryConfigEditObject(`SELECT id, name, display_name, config_set_json, 0 AS readonly, source_node_backend_runtime_id AS parent_id, 'node_backend_runtime' AS parent_kind, COALESCE(source_config_hash,'') AS snapshot_id, '' AS template_id FROM model_deployments WHERE id=?`, id)
		if err != nil {
			return nil, err
		}
		var placementRaw, serviceRaw string
		_ = h.DB.QueryRow(`SELECT COALESCE(placement_json,'{}'), COALESCE(service_json,'{}') FROM model_deployments WHERE id=?`, id).Scan(&placementRaw, &serviceRaw)
		set := copyConfigSet(rawJSONString(obj["config_set_json"], "{}"))
		materializeDeploymentCompatConfig(set, parseObjectJSON(placementRaw), parseObjectJSON(serviceRaw), "deployment", id)
		obj["config_set_json"] = configSetJSON(set)
		return obj, nil
	default:
		return nil, fmt.Errorf("unsupported object_kind %q", kind)
	}
}

func parseObjectJSON(raw string) map[string]interface{} {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]interface{}{}
	}
	return out
}

func materializeDeploymentCompatConfig(set map[string]interface{}, placement, service map[string]interface{}, layer, ref string) {
	if len(service) > 0 {
		portBinding := map[string]interface{}{
			"listen_host":       firstNonEmpty(strVal(service, "listen_host", ""), "0.0.0.0"),
			"host_port":         intFromAny(service["host_port"], 0),
			"container_port":    firstPositive(intFromAny(service["container_port"], 0), intFromAny(service["app_port"], 0)),
			"protocol":          firstNonEmpty(strVal(service, "protocol", ""), "tcp"),
			"served_model_name": strVal(service, "served_model_name", ""),
		}
		setConfigValue(set, "service.port_binding", portBinding, layer, ref, "materialized_from_service_json")
		if portBinding["host_port"] != 0 {
			setConfigValue(set, "deployment.host_port", portBinding["host_port"], layer, ref, "materialized_from_service_json")
		}
		if portBinding["container_port"] != 0 {
			setConfigValue(set, "service.container_port", portBinding["container_port"], layer, ref, "materialized_from_service_json")
		}
		if portBinding["served_model_name"] != "" {
			setConfigValue(set, "deployment.served_model_name", portBinding["served_model_name"], layer, ref, "materialized_from_service_json")
		}
	}
	if len(placement) > 0 {
		enabled := true
		if _, ok := placement["device_binding_enabled"]; ok {
			enabled = boolFromAny(placement["device_binding_enabled"], true)
		}
		mode := strings.TrimSpace(strVal(placement, "accelerator_selection_mode", ""))
		if !enabled {
			mode = "disabled"
		}
		ids := stringSliceFromAny(placement["accelerator_ids"])
		if mode == "" {
			if len(ids) > 0 {
				mode = "manual"
			} else {
				mode = "auto"
			}
		}
		deviceBinding := map[string]interface{}{
			"enabled":           enabled,
			"mode":              mode,
			"vendor":            strVal(placement, "vendor", ""),
			"accelerator_ids":   ids,
			"accelerator_count": firstPositive(intFromAny(placement["accelerator_count"], 0), len(ids)),
			"visible_env_key":   strVal(placement, "visible_env_key", ""),
			"visible_env_value": strVal(placement, "visible_env_value", ""),
			"docker_gpu_option": strVal(placement, "docker_gpu_option", ""),
			"device_mounts":     stringSliceFromAny(placement["device_mounts"]),
		}
		setConfigValue(set, "runtime.device_binding", deviceBinding, layer, ref, "materialized_from_placement_json")
	}
}

func (h *AgentHandler) queryConfigEditObject(query, id string) (map[string]interface{}, error) {
	var objID, name, displayName, configSetRaw string
	var readonly int
	var parentID, parentKind, snapshotID, templateID string
	if err := h.DB.QueryRow(query, id).Scan(&objID, &name, &displayName, &configSetRaw, &readonly, &parentID, &parentKind, &snapshotID, &templateID); err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return map[string]interface{}{
		"id":              objID,
		"name":            name,
		"display_name":    displayName,
		"config_set_json": configSetRaw,
		"readonly":        readonly == 1,
		"snapshot_id":     snapshotID,
		"template_id":     templateID,
		"parent": map[string]interface{}{
			"object_kind": parentKind,
			"object_id":   parentID,
			"snapshot_id": "",
		},
	}, nil
}

func (h *AgentHandler) updateConfigEditObject(kind, id, configSetRaw string) error {
	switch kind {
	case "backend_version":
		_, err := h.DB.Exec(`UPDATE backend_versions SET config_set_json=?, updated_at=datetime('now') WHERE id=?`, configSetRaw, id)
		return err
	case "backend_runtime":
		_, err := h.DB.Exec(`UPDATE backend_runtimes SET config_set_json=?, checksum=?, updated_at=datetime('now') WHERE id=?`, configSetRaw, checksumString(configSetRaw), id)
		return err
	case "node_backend_runtime":
		_, err := h.DB.Exec(`UPDATE node_backend_runtimes SET config_set_json=?, status='needs_check', status_reason='configuration changed', updated_at=datetime('now') WHERE id=?`, configSetRaw, id)
		return err
	case "deployment":
		_, err := h.DB.Exec(`UPDATE model_deployments SET config_set_json=?, updated_at=datetime('now') WHERE id=?`, configSetRaw, id)
		return err
	default:
		return fmt.Errorf("unsupported object_kind %q", kind)
	}
}

func decodeConfigEditPatch(raw interface{}) (configedit.ConfigEditPatch, error) {
	var patch configedit.ConfigEditPatch
	if raw == nil {
		return patch, fmt.Errorf("patch required")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return patch, err
	}
	if err := json.Unmarshal(b, &patch); err != nil {
		return patch, err
	}
	return patch, nil
}

func objectRefFromAny(raw interface{}) *configedit.ObjectRef {
	m, _ := raw.(map[string]interface{})
	if m == nil {
		return nil
	}
	kind := strVal(m, "object_kind", "")
	id := strVal(m, "object_id", "")
	if kind == "" || id == "" {
		return nil
	}
	return &configedit.ObjectRef{ObjectKind: kind, ObjectID: id, SnapshotID: strVal(m, "snapshot_id", "")}
}

func stringSliceFromAny(raw interface{}) []string {
	items, _ := raw.([]interface{})
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := fmt.Sprint(item); s != "" && s != "<nil>" {
			out = append(out, s)
		}
	}
	return out
}

func applyEditableConfigPatchIfPresent(set map[string]interface{}, req map[string]interface{}, layer, ref string) (map[string]interface{}, error) {
	raw, ok := req["editable_config_patch"]
	if !ok || raw == nil {
		return set, nil
	}
	patch, err := decodeConfigEditPatch(raw)
	if err != nil {
		return nil, err
	}
	if patch.Layer == "" {
		patch.Layer = layer
	}
	if patch.ObjectID == "" {
		patch.ObjectID = ref
	}
	return configedit.ApplyEditPatchToConfigSet(set, patch, layer, ref)
}
