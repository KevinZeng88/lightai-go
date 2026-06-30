package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lightai-go/internal/server/catalog"
	"lightai-go/internal/server/configedit"

	"gopkg.in/yaml.v3"
)

func (h *AgentHandler) HandleListConfigEditTemplates(w http.ResponseWriter, r *http.Request) {
	store, err := loadConfigEditTemplateRegistry()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, store)
}

func (h *AgentHandler) HandleGetConfigEditTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := loadConfigEditTemplateRegistry()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, tmpl := range store.Templates {
		if tmpl.TemplateID == id {
			writeJSON(w, http.StatusOK, tmpl)
			return
		}
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (h *AgentHandler) HandleValidateConfigEditTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl configedit.ComponentTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	issues := configedit.ValidateComponentTemplate(tmpl)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":  len(issues) == 0,
		"issues": issues,
	})
}

func (h *AgentHandler) HandleCloneConfigEditTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := loadConfigEditTemplateStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var src *configedit.ComponentTemplate
	for i := range store.Templates {
		if store.Templates[i].TemplateID == id {
			src = &store.Templates[i]
			break
		}
	}
	if src == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	next := *src
	next.TemplateID = strings.TrimSuffix(src.TemplateID, "-configedit-v1") + "-local-configedit-v1"
	if next.Metadata == nil {
		next.Metadata = map[string]any{}
	}
	next.Metadata["source"] = "local"
	next.Source = "local"
	localRoot := configEditTemplateLocalRoot()
	if err := os.MkdirAll(localRoot, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	path := filepath.Join(localRoot, safeTemplateFileName(next.TemplateID)+".yaml")
	data, _ := yaml.Marshal(next)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	next.Path = path
	writeJSON(w, http.StatusCreated, next)
}

func loadConfigEditTemplateStore() (*configedit.TemplateStore, error) {
	return configedit.LoadComponentTemplates(configEditTemplateBuiltinRoot(), configEditTemplateLocalRoot())
}

func loadConfigEditTemplateRegistry() (*configedit.TemplateStore, error) {
	store, err := loadConfigEditTemplateStore()
	if err != nil {
		return nil, err
	}
	generated, issues := loadMaterializedConfigEditTemplates()
	store.Issues = append(store.Issues, issues...)
	byID := make(map[string]configedit.ComponentTemplate, len(store.Templates)+len(generated))
	for _, tmpl := range store.Templates {
		byID[tmpl.TemplateID] = tmpl
	}
	for _, tmpl := range generated {
		if _, exists := byID[tmpl.TemplateID]; !exists {
			byID[tmpl.TemplateID] = tmpl
		}
	}
	store.Templates = store.Templates[:0]
	for _, tmpl := range byID {
		store.Templates = append(store.Templates, tmpl)
	}
	sort.SliceStable(store.Templates, func(i, j int) bool {
		if store.Templates[i].Source == store.Templates[j].Source {
			return store.Templates[i].TemplateID < store.Templates[j].TemplateID
		}
		return sourceRank(store.Templates[i].Source) < sourceRank(store.Templates[j].Source)
	})
	return store, nil
}

func sourceRank(source string) int {
	switch source {
	case "local":
		return 0
	case "built_in":
		return 1
	case "catalog_materialized":
		return 2
	default:
		return 9
	}
}

func loadMaterializedConfigEditTemplates() ([]configedit.ComponentTemplate, []configedit.TemplateValidationIssue) {
	registry, err := catalog.LoadRegistry("")
	if err != nil {
		return nil, []configedit.TemplateValidationIssue{{Path: "configs/config-registry", Reason: err.Error(), Severity: "error"}}
	}
	backendCatalog, err := catalog.LoadBackendCatalog("")
	if err != nil {
		return nil, []configedit.TemplateValidationIssue{{Path: "configs/backend-catalog", Reason: err.Error(), Severity: "error"}}
	}

	backendByID := map[string]catalog.BackendDoc{}
	for _, backend := range backendCatalog.Backends {
		backendByID[backend.ID] = backend
	}
	versionSets := map[string]catalog.ConfigSet{}
	for _, version := range backendCatalog.Versions {
		backend := backendByID[version.BackendID]
		versionSets[version.ID] = catalog.MaterializeBackendVersion(registry, backend, version)
	}

	out := make([]configedit.ComponentTemplate, 0, len(backendCatalog.Runtimes))
	for _, runtime := range backendCatalog.Runtimes {
		versionSet, ok := versionSets[runtime.BackendVersionID]
		if !ok {
			continue
		}
		set := catalog.MaterializeBackendRuntime(registry, versionSet, runtime)
		view, err := configedit.ProjectConfigSetToEditView(configedit.ProjectInput{
			ConfigSet:   configSetToMap(set),
			Layer:       "backend_runtime",
			ObjectKind:  "backend_runtime",
			ObjectID:    runtime.ID,
			ObjectLabel: runtime.DisplayName,
			ViewLevel:   "developer",
			Readonly:    true,
		})
		if err != nil {
			return out, []configedit.TemplateValidationIssue{{Path: runtime.SourcePath, Reason: err.Error(), Severity: "error"}}
		}
		out = append(out, materializedTemplateFromView(runtime, set, view))
	}
	return out, nil
}

func configSetToMap(set catalog.ConfigSet) map[string]any {
	data, _ := json.Marshal(set)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return out
}

func materializedTemplateFromView(runtime catalog.RuntimeDoc, set catalog.ConfigSet, view configedit.ConfigEditView) configedit.ComponentTemplate {
	backend := strings.TrimPrefix(runtime.BackendID, "backend.")
	displayName := strings.TrimSpace(runtime.DisplayName)
	if displayName == "" {
		displayName = runtime.Name
	}
	sections := make([]configedit.TemplateSection, 0, len(view.Sections))
	for _, section := range view.Sections {
		sections = append(sections, configedit.TemplateSection{
			Key:       section.Key,
			Label:     section.Label,
			Order:     section.Order,
			View:      sectionView(section),
			Collapsed: section.Collapsed,
		})
	}
	fields := make([]configedit.TemplateField, 0, len(view.Fields))
	for _, field := range view.Fields {
		fields = append(fields, configedit.TemplateField{
			Key:          field.Key,
			InternalKey:  field.InternalKey,
			ComponentKey: field.ComponentKey,
			Label:        field.Label,
			LabelI18nKey: field.LabelI18nKey,
			HelpI18nKey:  field.HelpI18nKey,
			Section:      field.Section,
			Tier:         fieldTier(field),
			View:         normalizeTemplateView(field.View),
			Risk:         fieldRisk(field),
			Order:        field.Order,
			Type:         field.Type,
			Widget:       field.Widget,
			Enabled:      field.Enabled,
			Source:       field.Source,
			Path:         field.Path,
			Effects:      templateEffects(field.Effects),
		})
	}
	components := make([]configedit.TemplateComponent, 0, len(view.Components))
	for _, component := range view.Components {
		components = append(components, configedit.TemplateComponent{
			Key:       component.Key,
			Component: component.Type,
			Renderer:  component.Renderer,
			Label:     component.Label,
			Section:   component.Section,
			View:      normalizeTemplateView(component.View),
			Order:     component.Order,
			Effects:   templateEffects(component.Effects),
		})
	}
	return configedit.ComponentTemplate{
		TemplateID: safeTemplateFileName(runtime.ID) + "-materialized-configedit-v1",
		Kind:       "config_edit_template",
		Version:    1,
		Source:     "catalog_materialized",
		Path:       runtime.SourcePath,
		AppliesTo: configedit.TemplateAppliesTo{
			Backend:         backend,
			BackendVersions: []string{runtime.BackendVersionID},
			RuntimeKind:     nonEmptyRuntime(runtime.RunnerType, "docker"),
			Vendors:         []string{runtime.Vendor},
		},
		Metadata: map[string]any{
			"display_name":       displayName,
			"source":             "catalog_materialized",
			"scope":              "backend_runtime",
			"runtime_id":         runtime.ID,
			"backend_id":         runtime.BackendID,
			"backend_version_id": runtime.BackendVersionID,
			"vendor":             runtime.Vendor,
			"field_count":        len(fields),
			"component_count":    len(components),
			"source_metadata":    set.SourceMetadata,
		},
		Views: configedit.TemplateViews{
			DefaultView:    "advanced",
			SupportedViews: []string{"normal", "advanced", "developer"},
		},
		Layers: map[string]configedit.LayerPolicy{
			"backend_runtime":      {Editable: false},
			"node_backend_runtime": {Editable: true, CopyFromParent: "whole_effective_configedit_snapshot"},
			"deployment":           {Editable: true, CopyFromParent: "whole_effective_configedit_snapshot"},
			"deployment_override":  {Editable: true, CopyFromParent: "whole_effective_configedit_snapshot"},
		},
		Sections:   sections,
		Fields:     fields,
		Components: components,
	}
}

func sectionView(section configedit.EditSection) string {
	if section.Key == "advanced_raw" || section.Key == "expert_parameters" || section.Key == "security_high_risk" {
		return "developer"
	}
	if section.Advanced {
		return "advanced"
	}
	return "normal"
}

func fieldTier(field configedit.EditField) string {
	if field.View == "developer" || field.Visibility == "internal" || field.Visibility == "hidden" || field.Diagnostic {
		return "expert"
	}
	if field.View == "security" || field.Section == "security_high_risk" {
		return "expert"
	}
	if field.Advanced || field.View == "advanced" {
		return "advanced"
	}
	return "normal"
}

func fieldRisk(field configedit.EditField) string {
	if field.View == "security" || field.Section == "security_high_risk" {
		return "high"
	}
	return ""
}

func normalizeTemplateView(view string) string {
	if view == "security" {
		return "developer"
	}
	if view == "" {
		return "normal"
	}
	return view
}

func templateEffects(effects []configedit.EditEffectPreview) []configedit.TemplateEffect {
	out := make([]configedit.TemplateEffect, 0, len(effects))
	for _, effect := range effects {
		out = append(out, configedit.TemplateEffect{
			Type:   effect.Type,
			Target: effect.Target,
			Flag:   effect.Key,
			Properties: map[string]any{
				"component_key": effect.ComponentKey,
				"field_key":     effect.FieldKey,
				"source":        effect.Source,
				"patch_target":  effect.PatchTarget,
				"docker_effect": effect.DockerEffect,
			},
		})
	}
	return out
}

func nonEmptyRuntime(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func configEditTemplateBuiltinRoot() string {
	if v := strings.TrimSpace(os.Getenv("LIGHTAI_CONFIGEDIT_TEMPLATE_BUILTIN_DIR")); v != "" {
		return v
	}
	return repoRelativePath("configs/configedit-templates/builtin")
}

func configEditTemplateLocalRoot() string {
	if v := strings.TrimSpace(os.Getenv("LIGHTAI_CONFIGEDIT_TEMPLATE_LOCAL_DIR")); v != "" {
		return v
	}
	return repoRelativePath("configs/configedit-templates/local")
}

func repoRelativePath(path string) string {
	wd, err := os.Getwd()
	if err == nil {
		for dir := wd; dir != "." && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, path)
			if _, statErr := os.Stat(filepath.Dir(candidate)); statErr == nil {
				return candidate
			}
		}
	}
	return path
}

func safeTemplateFileName(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "configedit-template"
	}
	return b.String()
}
