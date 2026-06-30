package configedit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ComponentTemplate struct {
	TemplateID string                 `json:"template_id" yaml:"template_id"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Version    int                    `json:"version" yaml:"version"`
	Source     string                 `json:"source" yaml:"-"`
	Path       string                 `json:"path" yaml:"-"`
	AppliesTo  TemplateAppliesTo      `json:"applies_to" yaml:"applies_to"`
	Metadata   map[string]any         `json:"metadata" yaml:"metadata"`
	Views      TemplateViews          `json:"views" yaml:"views"`
	Layers     map[string]LayerPolicy `json:"layers" yaml:"layers"`
	Sections   []TemplateSection      `json:"sections" yaml:"sections"`
	Fields     []TemplateField        `json:"fields,omitempty" yaml:"fields,omitempty"`
	Components []TemplateComponent    `json:"components" yaml:"components"`
}

type TemplateAppliesTo struct {
	Backend         string   `json:"backend" yaml:"backend"`
	BackendVersions []string `json:"backend_versions" yaml:"backend_versions"`
	RuntimeKind     string   `json:"runtime_kind" yaml:"runtime_kind"`
	Vendors         []string `json:"vendors" yaml:"vendors"`
}

type TemplateViews struct {
	DefaultView    string   `json:"default_view" yaml:"default_view"`
	SupportedViews []string `json:"supported_views" yaml:"supported_views"`
}

type LayerPolicy struct {
	Editable       bool   `json:"editable" yaml:"editable"`
	CopyFromParent string `json:"copy_from_parent,omitempty" yaml:"copy_from_parent"`
}

type TemplateSection struct {
	Key       string `json:"key" yaml:"key"`
	Label     string `json:"label" yaml:"label"`
	Order     int    `json:"order" yaml:"order"`
	View      string `json:"view" yaml:"view"`
	Collapsed bool   `json:"collapsed,omitempty" yaml:"collapsed"`
}

type TemplateField struct {
	Key          string           `json:"key" yaml:"key"`
	InternalKey  string           `json:"internal_key,omitempty" yaml:"internal_key,omitempty"`
	ComponentKey string           `json:"component_key,omitempty" yaml:"component_key,omitempty"`
	Label        string           `json:"label" yaml:"label"`
	LabelI18nKey string           `json:"label_i18n_key,omitempty" yaml:"label_i18n_key,omitempty"`
	HelpI18nKey  string           `json:"help_i18n_key,omitempty" yaml:"help_i18n_key,omitempty"`
	Section      string           `json:"section" yaml:"section"`
	Tier         string           `json:"tier,omitempty" yaml:"tier,omitempty"`
	View         string           `json:"view" yaml:"view"`
	Risk         string           `json:"risk,omitempty" yaml:"risk,omitempty"`
	Order        int              `json:"order" yaml:"order"`
	Type         string           `json:"type" yaml:"type"`
	Widget       string           `json:"widget" yaml:"widget"`
	Enabled      bool             `json:"enabled" yaml:"enabled"`
	Source       map[string]any   `json:"source,omitempty" yaml:"source,omitempty"`
	Path         []string         `json:"path,omitempty" yaml:"path,omitempty"`
	Effects      []TemplateEffect `json:"effects,omitempty" yaml:"effects,omitempty"`
}

type TemplateComponent struct {
	Key             string              `json:"key" yaml:"key"`
	Component       string              `json:"component" yaml:"component"`
	Renderer        string              `json:"renderer" yaml:"renderer"`
	Label           string              `json:"label" yaml:"label"`
	Description     string              `json:"description,omitempty" yaml:"description"`
	Section         string              `json:"section" yaml:"section"`
	View            string              `json:"view" yaml:"view"`
	Order           int                 `json:"order" yaml:"order"`
	Value           map[string]any      `json:"value,omitempty" yaml:"value"`
	Editability     map[string]bool     `json:"editability,omitempty" yaml:"editability"`
	Copy            map[string]any      `json:"copy,omitempty" yaml:"copy"`
	Reset           map[string]any      `json:"reset,omitempty" yaml:"reset"`
	Validation      []map[string]any    `json:"validation,omitempty" yaml:"validation"`
	Help            map[string]any      `json:"help,omitempty" yaml:"help"`
	Effects         []TemplateEffect    `json:"effects,omitempty" yaml:"effects"`
	ComponentFields []TemplateComponent `json:"fields,omitempty" yaml:"fields"`
}

type TemplateEffect struct {
	Type       string         `json:"type" yaml:"type"`
	Target     string         `json:"target,omitempty" yaml:"target"`
	Flag       string         `json:"flag,omitempty" yaml:"flag"`
	ValueFrom  string         `json:"value_from,omitempty" yaml:"value_from"`
	KeyFrom    string         `json:"key_from,omitempty" yaml:"key_from"`
	When       string         `json:"when,omitempty" yaml:"when"`
	OmitEmpty  bool           `json:"omit_if_empty,omitempty" yaml:"omit_if_empty"`
	Properties map[string]any `json:"properties,omitempty" yaml:",inline"`
}

type TemplateValidationIssue struct {
	Path     string `json:"path"`
	Reason   string `json:"reason"`
	Severity string `json:"severity"`
}

type TemplateStore struct {
	Templates []ComponentTemplate       `json:"templates"`
	Issues    []TemplateValidationIssue `json:"issues,omitempty"`
}

func LoadComponentTemplates(builtinRoot, localRoot string) (*TemplateStore, error) {
	merged := map[string]ComponentTemplate{}
	var issues []TemplateValidationIssue
	for _, root := range []struct {
		path   string
		source string
	}{
		{builtinRoot, "built_in"},
		{localRoot, "local"},
	} {
		if strings.TrimSpace(root.path) == "" {
			continue
		}
		templates, loadIssues, err := loadTemplateRoot(root.path, root.source)
		issues = append(issues, loadIssues...)
		if err != nil && root.source == "built_in" {
			return nil, err
		}
		for _, tmpl := range templates {
			merged[tmpl.TemplateID] = tmpl
		}
	}
	out := make([]ComponentTemplate, 0, len(merged))
	for _, tmpl := range merged {
		out = append(out, tmpl)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TemplateID < out[j].TemplateID })
	return &TemplateStore{Templates: out, Issues: issues}, nil
}

func loadTemplateRoot(root, source string) ([]ComponentTemplate, []TemplateValidationIssue, error) {
	var matches []string
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	var out []ComponentTemplate
	var issues []TemplateValidationIssue
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, TemplateValidationIssue{Path: path, Reason: err.Error(), Severity: "error"})
			continue
		}
		var tmpl ComponentTemplate
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			issues = append(issues, TemplateValidationIssue{Path: path, Reason: err.Error(), Severity: "error"})
			continue
		}
		tmpl.Source = source
		tmpl.Path = path
		if errs := ValidateComponentTemplate(tmpl); len(errs) > 0 {
			issues = append(issues, errs...)
			continue
		}
		out = append(out, tmpl)
	}
	return out, issues, nil
}

func ValidateComponentTemplate(t ComponentTemplate) []TemplateValidationIssue {
	var issues []TemplateValidationIssue
	add := func(path, reason string) {
		issues = append(issues, TemplateValidationIssue{Path: path, Reason: reason, Severity: "error"})
	}
	if t.TemplateID == "" {
		add("template_id", "template_id is required")
	}
	if t.Kind != "config_edit_template" {
		add("kind", "kind must be config_edit_template")
	}
	if t.Version <= 0 {
		add("version", "version must be positive")
	}
	if t.AppliesTo.Backend == "" {
		add("applies_to.backend", "backend is required")
	}
	allowedViews := map[string]bool{"normal": true, "advanced": true, "developer": true}
	allowedLayers := map[string]bool{"backend_runtime": true, "node_backend_runtime": true, "deployment": true, "deployment_override": true}
	allowedRenderers := map[string]bool{"string": true, "number": true, "boolean": true, "select": true, "string_list": true, "key_value_table": true, "raw_json": true, "accelerator_binding": true, "port_binding": true, "health_check_form": true, "mount_form": true, "args_editor": true, "docker_options": true}
	allowedEffects := map[string]bool{"cli_arg": true, "env": true, "docker": true, "mount": true, "port": true, "health_check": true, "device_binding": true}
	sectionKeys := map[string]bool{}
	for i, section := range t.Sections {
		if section.Key == "" {
			add(fmt.Sprintf("sections[%d].key", i), "section key is required")
		}
		if section.View != "" && !allowedViews[section.View] {
			add(fmt.Sprintf("sections[%d].view", i), "invalid view level")
		}
		if sectionKeys[section.Key] {
			add(fmt.Sprintf("sections[%d].key", i), "duplicate section key")
		}
		sectionKeys[section.Key] = true
	}
	componentKeys := map[string]bool{}
	for i, c := range t.Components {
		path := fmt.Sprintf("components[%d]", i)
		if c.Key == "" {
			add(path+".key", "component key is required")
		}
		if componentKeys[c.Key] {
			add(path+".key", "duplicate component key")
		}
		componentKeys[c.Key] = true
		if c.Renderer == "" {
			add(path+".renderer", "renderer is required")
		} else if !allowedRenderers[c.Renderer] {
			add(path+".renderer", "unknown renderer")
		}
		if c.View == "" || !allowedViews[c.View] {
			add(path+".view", "invalid view level")
		}
		if c.Section != "" && !sectionKeys[c.Section] {
			add(path+".section", "unknown section")
		}
		for layer := range c.Editability {
			if !allowedLayers[layer] {
				add(path+".editability."+layer, "invalid editability layer")
			}
		}
		for j, effect := range c.Effects {
			effectPath := fmt.Sprintf("%s.effects[%d]", path, j)
			if !allowedEffects[effect.Type] {
				add(effectPath+".type", "unknown effect type")
			}
			if effect.Type != "" && effect.Target == "" && effect.Flag == "" && effect.ValueFrom == "" {
				add(effectPath, "effect target, flag, or value_from is required")
			}
			if strings.Contains(effect.ValueFrom, "exec(") || strings.Contains(effect.When, "exec(") {
				add(effectPath, "unsafe expression function")
			}
		}
	}
	return issues
}

func FindTemplateFor(store *TemplateStore, backend, vendor string) *ComponentTemplate {
	if store == nil {
		return nil
	}
	for i := range store.Templates {
		t := &store.Templates[i]
		if t.AppliesTo.Backend != backend {
			continue
		}
		if len(t.AppliesTo.Vendors) == 0 || stringInSlice(vendor, t.AppliesTo.Vendors) {
			return t
		}
	}
	return nil
}

func stringInSlice(value string, items []string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
