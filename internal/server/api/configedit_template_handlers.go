package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lightai-go/internal/server/configedit"

	"gopkg.in/yaml.v3"
)

func (h *AgentHandler) HandleListConfigEditTemplates(w http.ResponseWriter, r *http.Request) {
	store, err := loadConfigEditTemplateStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, store)
}

func (h *AgentHandler) HandleGetConfigEditTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := loadConfigEditTemplateStore()
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
