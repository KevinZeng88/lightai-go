package catalog

// ============================================================================
// ConfigView / ConfigPanel — external presentation layer
// ============================================================================

// ConfigView is the external-facing view of a ConfigSet. UI pages consume
// ConfigView/ConfigPanel, not raw internal ConfigSet structures.
//
// Each ConfigSet can generate its own ConfigView. Parent ConfigSets delegate
// child rendering to child ConfigViews via child_slots.
type ConfigView struct {
	ConfigSetKey string        `json:"config_set_key"`
	Title        string        `json:"title"`
	Subtitle     string        `json:"subtitle,omitempty"`
	Summary      string        `json:"summary,omitempty"`
	Sections     []ViewSection `json:"sections"`
	ChildPanels  []ConfigPanel `json:"child_panels,omitempty"`
	LocalEdits   []LocalEditSummary `json:"local_edits_summary,omitempty"`
	Preview      *ConfigSet    `json:"effective_preview,omitempty"`
}

// ViewSection is one grouping of ConfigItems within a ConfigView.
type ViewSection struct {
	Key             string       `json:"key"`
	Title           string       `json:"title"`
	Description     string       `json:"description,omitempty"`
	DefaultExpanded bool         `json:"default_expanded"`
	Priority        int          `json:"priority"`
	Fields          []FieldView  `json:"fields"`
}

// FieldView is the external-facing representation of a single ConfigItem.
type FieldView struct {
	Key          string            `json:"key"`
	ConfigSetKey string            `json:"config_set_key"`
	Label        string            `json:"label"`
	Help         string            `json:"help,omitempty"`
	Section      string            `json:"section"`
	Group        string            `json:"group,omitempty"`
	Priority     int               `json:"priority"`

	// Schema-derived
	Type       string         `json:"type"`
	Required   bool           `json:"required"`
	Advanced   bool           `json:"advanced"`
	ReadOnly   bool           `json:"readonly"`
	Choices    []any          `json:"choices,omitempty"`
	Constraints map[string]any `json:"constraints,omitempty"`

	// Value
	Value        any `json:"value"`
	DefaultValue any `json:"default_value,omitempty"`
	InheritedValue any `json:"inherited_value,omitempty"`

	// State
	Enabled  bool `json:"enabled"`
	Checked  bool `json:"checked"`
	Editable bool `json:"editable"`
	Visible  bool `json:"visible"`
	Valid    bool   `json:"valid"`
	ValidationError string `json:"validation_error,omitempty"`

	// Provenance
	ValueSource    string             `json:"value_source"`
	LastValueLayer string             `json:"last_value_layer"`
	SourceChain    []SourceChainEntry `json:"source_chain,omitempty"`

	// Snapshot
	CopiedFrom string `json:"copied_from,omitempty"`

	// Presentation
	DisplayMode     string `json:"display_mode,omitempty"`
	Placeholder     string `json:"placeholder,omitempty"`
	Sensitive       bool   `json:"sensitive"`
	HideWhenEmpty   bool   `json:"hide_when_empty"`
}

// LocalEditSummary is a brief summary of one local edit for display.
type LocalEditSummary struct {
	ConfigSetKey string `json:"config_set_key"`
	ItemKey      string `json:"item_key"`
	Label        string `json:"label"`
	Value        any    `json:"value"`
	Reason       string `json:"reason"`
	EditedAt     string `json:"edited_at"`
}

// ConfigPanel wraps a child ConfigSet's view for embedding in a parent view.
type ConfigPanel struct {
	Slot            string     `json:"slot"`
	ChildConfigSetKey string   `json:"child_config_set_key"`
	Title           string     `json:"title"`
	View            string     `json:"view"`       // summary, summary_then_edit, edit, preview
	DisplayMode     string     `json:"display_mode"` // panel, card, inline
	DefaultExpanded bool       `json:"default_expanded"`
	Order           int        `json:"order"`
	ConfigView      ConfigView `json:"config_view"`
}

// ============================================================================
// ConfigView generation from ConfigSet
// ============================================================================

// GenerateView produces a ConfigView from a ConfigSet using its own_sections
// and child_slots definitions. If a parent bundle is provided, child ConfigSets
// are rendered via their own GenerateView.
func (cs ConfigSet) GenerateView() ConfigView {
	view := ConfigView{
		ConfigSetKey: cs.ConfigSetKey,
		Title:        cs.Title,
		Sections:     make([]ViewSection, 0),
		ChildPanels:  make([]ConfigPanel, 0),
	}

	if view.Title == "" {
		view.Title = cs.ConfigSetKey
	}

	// Build sections from own_sections
	for _, sec := range cs.OwnSections {
		fields := cs.collectFieldsForSection(sec)
		if len(fields) == 0 {
			continue
		}
		view.Sections = append(view.Sections, ViewSection{
			Key:             sec.Key,
			Title:           sec.Title,
			DefaultExpanded: sec.DefaultExpanded,
			Priority:        sec.Priority,
			Fields:          fields,
		})
	}

	// If no own_sections defined, generate default grouping
	if len(cs.OwnSections) == 0 {
		view.Sections = cs.defaultSectionGrouping()
	}

	// Build child panels from child_slots
	for _, slot := range cs.ChildSlots {
		panel := ConfigPanel{
			Slot:              slot.Slot,
			ChildConfigSetKey: slot.ChildConfigSetKey,
			Title:             slot.Title,
			View:              slot.View,
			DisplayMode:       slot.DisplayMode,
			DefaultExpanded:   slot.DefaultExpanded,
			Order:             slot.Order,
		}
		// Child ConfigSet view is populated by the caller who has access to the child
		view.ChildPanels = append(view.ChildPanels, panel)
	}

	return view
}

// collectFieldsForSection returns FieldViews matching the section's filter criteria.
func (cs ConfigSet) collectFieldsForSection(sec ConfigSection) []FieldView {
	fields := make([]FieldView, 0)
	for _, item := range cs.Items {
		if !item.State_.Visible {
			continue
		}
		if !matchesSection(item, sec) {
			continue
		}
		fields = append(fields, itemToFieldView(item))
	}
	return fields
}

// defaultSectionGrouping produces standard required/common/advanced sections.
func (cs ConfigSet) defaultSectionGrouping() []ViewSection {
	requiredFields := make([]FieldView, 0)
	commonFields := make([]FieldView, 0)
	advancedFields := make([]FieldView, 0)

	for _, item := range cs.Items {
		if !item.State_.Visible {
			continue
		}
		fv := itemToFieldView(item)
		if item.Schema.Required {
			requiredFields = append(requiredFields, fv)
		} else if item.Schema.Advanced {
			advancedFields = append(advancedFields, fv)
		} else {
			commonFields = append(commonFields, fv)
		}
	}

	sections := make([]ViewSection, 0, 3)
	if len(requiredFields) > 0 {
		sections = append(sections, ViewSection{
			Key: "required", Title: "必填配置", DefaultExpanded: true, Priority: 10,
			Fields: requiredFields,
		})
	}
	if len(commonFields) > 0 {
		sections = append(sections, ViewSection{
			Key: "common", Title: "常用配置", DefaultExpanded: true, Priority: 20,
			Fields: commonFields,
		})
	}
	if len(advancedFields) > 0 {
		sections = append(sections, ViewSection{
			Key: "advanced", Title: "高级配置", DefaultExpanded: false, Priority: 90,
			Fields: advancedFields,
		})
	}
	return sections
}

func matchesSection(item ConfigItem, sec ConfigSection) bool {
	if sec.Match == nil {
		return true
	}
	for k, v := range sec.Match {
		switch k {
		case "required":
			if b, ok := v.(bool); ok && item.Schema.Required != b {
				return false
			}
		case "advanced":
			if b, ok := v.(bool); ok && item.Schema.Advanced != b {
				return false
			}
		case "group":
			if s, ok := v.(string); ok && item.Presentation.Group != s {
				return false
			}
		case "category":
			if s, ok := v.(string); ok && item.Schema.Category != s {
				return false
			}
		}
	}
	return true
}

func itemToFieldView(item ConfigItem) FieldView {
	return FieldView{
		Key:            item.Schema.Key,
		ConfigSetKey:   item.Schema.ConfigSetKey,
		Label:          item.Schema.Label,
		Help:           item.Schema.HelpText,
		Section:        item.Presentation.Section,
		Group:          item.Presentation.Group,
		Priority:       item.Presentation.Priority,
		Type:           item.Schema.Type,
		Required:       item.Schema.Required,
		Advanced:       item.Schema.Advanced,
		ReadOnly:       item.Schema.ReadOnly,
		Choices:        item.Schema.Choices,
		Constraints:    item.Schema.Constraints,
		Value:          item.Value_.EffectiveValue,
		DefaultValue:   item.Value_.DefaultValue,
		InheritedValue: item.Value_.InheritedValue,
		Enabled:        item.State_.Enabled,
		Checked:        item.State_.Checked,
		Editable:       item.State_.Editable,
		Visible:        item.State_.Visible,
		Valid:          item.State_.Valid,
		ValidationError: item.State_.ValidationError,
		ValueSource:    item.Provenance_.ValueSource,
		LastValueLayer: item.Provenance_.LastValueLayer,
		SourceChain:    item.Provenance_.SourceChain,
		CopiedFrom:     item.Snapshot_.FromLayer,
		DisplayMode:    item.Presentation.DisplayMode,
		Placeholder:    item.Presentation.Placeholder,
		Sensitive:      item.Presentation.Sensitive,
		HideWhenEmpty:  item.Presentation.HideWhenEmpty,
	}
}

// ============================================================================
// Bundle-level view generation
// ============================================================================

// GenerateBundleView produces a ConfigView that merges the effective snapshot
// with own sets and local edits. Child ConfigSets referenced by child_slots
// are resolved from OwnSets or InheritedBundleSnapshots.
func (b *ConfigSetBundle) GenerateBundleView() ConfigView {
	effSnap := b.EffectiveSnapshot()

	// Use the last own set's config_set_key if available, otherwise inherited
	view := effSnap.GenerateView()

	// Add local edits summary
	for _, setEdits := range b.LocalEdits {
		for itemKey, edit := range setEdits {
			label := itemKey
			if item := b.findItem(itemKey); item != nil {
				if item.Schema.Label != "" {
					label = item.Schema.Label
				}
			}
			view.LocalEdits = append(view.LocalEdits, LocalEditSummary{
				ConfigSetKey: edit.ConfigSetKey,
				ItemKey:      itemKey,
				Label:        label,
				Value:        edit.LocalValue,
				Reason:       edit.Reason,
				EditedAt:     edit.EditedAt,
			})
		}
	}

	return view
}

// ============================================================================
// Custom renderer registry
// ============================================================================

// CustomRenderer produces a ViewSection for a complex ConfigSet that cannot
// be fully represented by the generic field-based renderer.
type CustomRenderer interface {
	// RenderSection returns a ViewSection for the given ConfigSet.
	// Must consume ConfigView schema and obey ConfigItem field-tier rules.
	RenderSection(cs ConfigSet) ViewSection
}

// CustomRendererRegistry maps config_set_key to custom renderers.
var CustomRendererRegistry = map[string]CustomRenderer{}

// RegisterCustomRenderer adds a custom renderer for a specific ConfigSet key.
func RegisterCustomRenderer(key string, renderer CustomRenderer) {
	CustomRendererRegistry[key] = renderer
}
