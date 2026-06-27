package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

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
	out, err := configedit.ApplyEditPatchToConfigSet(copyConfigSet(rawJSONString(obj["config_set_json"], "{}")), patch, layer, id)
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
		return h.queryConfigEditObject(`SELECT id, version AS name, display_name, config_set_json, CASE WHEN readonly=1 OR managed_by='system' THEN 1 ELSE 0 END AS readonly FROM backend_versions WHERE id=?`, id)
	case "backend_runtime":
		return h.queryConfigEditObject(`SELECT id, name, display_name, config_set_json, CASE WHEN is_editable=0 THEN 1 ELSE 0 END AS readonly FROM backend_runtimes WHERE id=?`, id)
	case "node_backend_runtime":
		return h.queryConfigEditObject(`SELECT id, id AS name, display_name, config_set_json, 0 AS readonly FROM node_backend_runtimes WHERE id=?`, id)
	case "deployment":
		return h.queryConfigEditObject(`SELECT id, name, display_name, config_set_json, 0 AS readonly FROM model_deployments WHERE id=?`, id)
	default:
		return nil, fmt.Errorf("unsupported object_kind %q", kind)
	}
}

func (h *AgentHandler) queryConfigEditObject(query, id string) (map[string]interface{}, error) {
	var objID, name, displayName, configSetRaw string
	var readonly int
	if err := h.DB.QueryRow(query, id).Scan(&objID, &name, &displayName, &configSetRaw, &readonly); err != nil {
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
