package catalog

import "encoding/json"

// ============================================================================
// ConfigItem — final field-tier model (schema / value / state / provenance / snapshot / presentation)
// ============================================================================

// ConfigItemSchema holds immutable definition fields. After copy-on-create, inherited
// items retain their original Schema values; the current layer must not modify them.
type ConfigItemSchema struct {
	Key          string         `json:"key"`
	Owner        string         `json:"owner"`
	OwnerLayer   string         `json:"owner_layer"`
	ConfigSetKey string         `json:"config_set_key"`
	Category     string         `json:"category"`
	Label        string         `json:"label"`
	Description  string         `json:"description,omitempty"`
	Type         string         `json:"type"`
	Kind         string         `json:"kind"`
	Target       string         `json:"target,omitempty"`
	ArgName      string         `json:"arg_name,omitempty"`
	EnvName      string         `json:"env_name,omitempty"`
	MountTarget  string         `json:"mount_target,omitempty"`
	PortTarget   string         `json:"port_target,omitempty"`
	Constraints  map[string]any `json:"constraints,omitempty"`
	Choices      []any          `json:"choices,omitempty"`
	Required     bool           `json:"required"`
	Advanced     bool           `json:"advanced"`
	DisplayOrder int            `json:"display_order"`
	ReadOnly     bool           `json:"read_only"`
	HelpText     string         `json:"help_text,omitempty"`
	SupportLevel string         `json:"support_level"`
}

// ConfigItemValue holds value fields that the current layer may modify.
type ConfigItemValue struct {
	DefaultValue   any `json:"default_value"`
	InheritedValue any `json:"inherited_value,omitempty"`
	LocalValue     any `json:"local_value,omitempty"`
	EffectiveValue any `json:"effective_value"`
}

// ConfigItemState holds UI and resolution state that the current layer may modify.
type ConfigItemState struct {
	Enabled         bool   `json:"enabled"`
	Checked         bool   `json:"checked"`
	Editable        bool   `json:"editable"`
	Visible         bool   `json:"visible"`
	Valid           bool   `json:"valid"`
	ValidationError string `json:"validation_error,omitempty"`
}

// SourceChainEntry records one step in the value provenance chain.
type SourceChainEntry struct {
	Layer  string `json:"layer"`
	Value  any    `json:"value"`
	Reason string `json:"reason"`
}

// ConfigItemProvenance records where the current value came from and the full source chain.
type ConfigItemProvenance struct {
	ValueSource      string             `json:"value_source"`
	LastValueLayer   string             `json:"last_value_layer"`
	LastValueOwnerID string             `json:"last_value_owner_id,omitempty"`
	SourceChain      []SourceChainEntry `json:"source_chain,omitempty"`
}

// ConfigItemSnapshot records the copy-on-create source of this item.
type ConfigItemSnapshot struct {
	FromLayer string `json:"snapshot_from_layer"`
	FromID    string `json:"snapshot_from_id"`
	Version   int    `json:"snapshot_version"`
	CopiedAt  string `json:"snapshot_at"`
}

// ConfigItemPresentation holds display hints for UI rendering.
type ConfigItemPresentation struct {
	Section         string `json:"section,omitempty"`
	Group           string `json:"group,omitempty"`
	Priority        int    `json:"priority"`
	DisplayMode     string `json:"display_mode,omitempty"`
	Placeholder     string `json:"placeholder,omitempty"`
	SummaryPriority int    `json:"summary_priority"`
	HideWhenEmpty   bool   `json:"hide_when_empty"`
	DefaultExpanded bool   `json:"default_expanded"`
	Sensitive       bool   `json:"sensitive"`
}

// ConfigItem is the minimum configuration atom inside a ConfigSet.
//
// Field tiers (see docs/reports/runtime-architecture-parameter-final-state/04-final-parameter-contract.md):
//
//	Schema       — definition fields; readonly after copy-on-create
//	Value_       — value fields; current layer may modify
//	State_       — UI / resolution state; current layer may modify
//	Provenance_  — source tracking; updated on local edit
//	Snapshot_    — copy-on-create origin; readonly after copy
//	Presentation — display hints; not part of RunPlan semantics
//
// Legacy flat fields (Code, Category, Kind, Type, Required, Value, DefaultValue,
// Enabled, Render, Order, etc.) are kept as convenience accessors during migration
// batches and will be removed in the final cleanup batch.
type ConfigItem struct {
	Schema       ConfigItemSchema       `json:"config_item_schema"`
	Value_       ConfigItemValue        `json:"config_item_value"`
	State_       ConfigItemState        `json:"config_item_state"`
	Provenance_  ConfigItemProvenance   `json:"config_item_provenance"`
	Snapshot_    ConfigItemSnapshot     `json:"config_item_snapshot"`
	Presentation ConfigItemPresentation `json:"config_item_presentation"`

	// === Legacy flat fields (present during migration; removed in Batch 6) ===
	Code         string                 `json:"code" yaml:"code"`
	Category     string                 `json:"category" yaml:"category"`
	Kind         string                 `json:"kind" yaml:"kind"`
	Type         string                 `json:"type" yaml:"type"`
	Required     bool                   `json:"required" yaml:"required"`
	Visibility   string                 `json:"visibility,omitempty" yaml:"visibility"`
	Readonly     bool                   `json:"readonly,omitempty" yaml:"readonly"`
	Advanced     bool                   `json:"advanced,omitempty" yaml:"advanced"`
	Value        any                    `json:"value" yaml:"value"`
	DefaultValue any                    `json:"default_value" yaml:"default_value"`
	Enabled      bool                   `json:"enabled" yaml:"enabled"`
	Render       map[string]any         `json:"render,omitempty" yaml:"render"`
	Order        int                    `json:"order" yaml:"order"`
	Constraints  map[string]any         `json:"constraints,omitempty" yaml:"constraints"`
	SupportLevel string                 `json:"support_level" yaml:"support_level"`
	Source       map[string]string      `json:"source,omitempty" yaml:"source"`
	LastModified map[string]string      `json:"last_modified,omitempty" yaml:"last_modified"`
	Extensions   map[string]interface{} `json:"extensions,omitempty" yaml:"extensions"`
}

// AlignTiers populates the tiered fields from legacy flat fields.
// Call this after materializing a ConfigItem from catalog YAML or registry defaults.
//
// IMPORTANT: tiered fields that already have non-zero values are NOT overwritten.
// This prevents EffectiveSnapshot/merge operations from reverting local edits.
func (ci *ConfigItem) AlignTiers() {
	if ci == nil {
		return
	}
	// Schema — only populate from flat fields when tiered field is zero-value
	if ci.Schema.Key == "" {
		ci.Schema.Key = ci.Code
	}
	if ci.Schema.Category == "" {
		ci.Schema.Category = ci.Category
	}
	if ci.Schema.Kind == "" {
		ci.Schema.Kind = ci.Kind
	}
	if ci.Schema.Type == "" {
		ci.Schema.Type = ci.Type
	}
	if !ci.Schema.Required && ci.Required {
		ci.Schema.Required = ci.Required
	}
	if !ci.Schema.Advanced && ci.Advanced {
		ci.Schema.Advanced = ci.Advanced
	}
	if !ci.Schema.ReadOnly && ci.Readonly {
		ci.Schema.ReadOnly = ci.Readonly
	}
	if ci.Schema.DisplayOrder == 0 {
		ci.Schema.DisplayOrder = ci.Order
	}
	if ci.Schema.SupportLevel == "" {
		ci.Schema.SupportLevel = ci.SupportLevel
	}
	if ci.Schema.Constraints == nil {
		ci.Schema.Constraints = ci.Constraints
	}
	if ci.Render != nil {
		if ci.Schema.Target == "" {
			ci.Schema.Target = strVal(ci.Render["target"])
		}
		if ci.Schema.ArgName == "" {
			ci.Schema.ArgName = strVal(ci.Render["flag"])
		}
		if ci.Schema.EnvName == "" {
			ci.Schema.EnvName = strVal(ci.Render["env_name"])
		}
	}
	if ci.Extensions != nil {
		if ci.Schema.Label == "" {
			if l, ok := ci.Extensions["label"].(string); ok {
				ci.Schema.Label = l
			}
		}
		if ci.Presentation.Group == "" {
			if g, ok := ci.Extensions["group"].(string); ok {
				ci.Presentation.Group = g
			}
		}
	}

	// Value — only set from flat fields if tiered EffectiveValue is nil
	if ci.Value_.EffectiveValue == nil {
		ci.Value_.DefaultValue = ci.DefaultValue
		if ci.Value != nil {
			if ci.Enabled {
				ci.Value_.LocalValue = ci.Value
			} else {
				ci.Value_.InheritedValue = ci.Value
			}
			ci.Value_.EffectiveValue = ci.Value
		} else {
			ci.Value_.EffectiveValue = ci.DefaultValue
		}
	}

	// State — only set from flat fields if not explicitly set
	if !ci.State_.Enabled && ci.Enabled {
		ci.State_.Enabled = ci.Enabled
	}
	if !ci.State_.Checked && ci.Enabled && ci.Value_.LocalValue != nil {
		ci.State_.Checked = ci.Enabled
	}
	if !ci.State_.Editable && !ci.Readonly {
		ci.State_.Editable = !ci.Readonly
	}
	if ci.State_.Visible == false && ci.Visibility == "" {
		ci.State_.Visible = ci.Visibility != "hidden"
	}
	if !ci.State_.Valid {
		ci.State_.Valid = true
	}

	// Provenance — only populate if ValueSource is empty
	if ci.Provenance_.ValueSource == "" {
		if ci.Source != nil {
			ci.Provenance_.ValueSource = ci.Source["layer"]
			ci.Provenance_.LastValueLayer = ci.Source["layer"]
			ci.Provenance_.LastValueOwnerID = ci.Source["ref"]
		}
		if ci.LastModified != nil {
			ci.Provenance_.LastValueLayer = ci.LastModified["layer"]
			ci.Provenance_.LastValueOwnerID = ci.LastModified["ref"]
		}
	}

	// Presentation
	if ci.Presentation.Priority == 0 {
		ci.Presentation.Priority = ci.Order
	}
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// ConfigSet is a self-describing, self-presenting, composable configuration unit.
//
// A ConfigSet:
//   - Owns its ConfigItems (schema owner)
//   - May contain child ConfigSets (nested composition)
//   - Defines own_sections for item grouping (required / common / advanced / local_edits)
//   - Defines child_slots for child placement and display mode
//   - Can generate summary_view, edit_view, preview_view, effective_view
//   - Participates in RunPlan resolution through item target fields
//
// Child ConfigSets are stored as items under "child_sets" JSON key within the items map.
type ConfigSet struct {
	SchemaVersion  int                    `json:"schema_version"`
	ConfigSetKey   string                 `json:"config_set_key"`
	Title          string                 `json:"title,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Context        map[string]string      `json:"context"`
	Items          map[string]ConfigItem  `json:"items"`
	ChildSets      map[string]ConfigSet   `json:"child_sets,omitempty"`
	OwnSections    []ConfigSection        `json:"own_sections,omitempty"`
	ChildSlots     []ConfigChildSlot      `json:"child_slots,omitempty"`
	SourceMetadata map[string]interface{} `json:"source_metadata"`
}

// ConfigSection defines how a ConfigSet groups its own items for display.
type ConfigSection struct {
	Key             string         `json:"key"`
	Title           string         `json:"title"`
	Match           map[string]any `json:"match,omitempty"`
	DefaultExpanded bool           `json:"default_expanded"`
	Priority        int            `json:"priority"`
}

// ConfigChildSlot defines where and how a child ConfigSet appears in the parent view.
type ConfigChildSlot struct {
	Slot              string `json:"slot"`
	ChildConfigSetKey string `json:"child_config_set_key"`
	Title             string `json:"title"`
	View              string `json:"view"`       // summary, summary_then_edit, edit, preview
	DisplayMode       string `json:"display_mode"` // panel, card, inline
	DefaultExpanded   bool   `json:"default_expanded"`
	Order             int    `json:"order"`
}

// ============================================================================
// ConfigSetBundle — per-layer composition of inherited snapshots, own sets, local edits, and effective view
// ============================================================================

// ConfigSetBundle is owned by every domain layer (BackendVersion, BackendRuntime,
// NodeBackendRuntime, Deployment). It captures the full parameter snapshot chain.
//
//	ConfigSetBundle = inherited_bundle_snapshots[] + own_sets[] + local_edits[] + effective_view
//
// On copy-on-create:
//
//	next_layer_bundle = deep_copy(parent.effective_bundle_snapshot)
//	                  + next_layer_own_sets
//	                  + next_layer_local_edits
//
// The effective_view is materialized on read and is not independently persisted.
type ConfigSetBundle struct {
	// InheritedBundleSnapshots are deep copies of parent effective bundles at creation time.
	// Read-only; used for provenance display and copy-on-create ancestry.
	InheritedBundleSnapshots []ConfigSet `json:"inherited_bundle_snapshots"`

	// OwnSets are ConfigSets defined by this layer.
	OwnSets []ConfigSet `json:"own_sets"`

	// LocalEdits are value/state overrides applied by this layer on inherited items.
	// Keyed by ConfigSetKey then ItemKey. Only modified items appear here.
	LocalEdits map[string]map[string]ConfigItemLocalEdit `json:"local_edits"`

	// EffectiveView is materialized at read time and is not independently serialized.
	EffectiveView *ConfigSet `json:"effective_view,omitempty"`
}

// ConfigItemLocalEdit records a single value/state override applied at the current layer.
type ConfigItemLocalEdit struct {
	ConfigSetKey string `json:"config_set_key"`
	ItemKey      string `json:"item_key"`
	LocalValue   any    `json:"local_value,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
	Checked      *bool  `json:"checked,omitempty"`
	Reason       string `json:"reason"`
	EditedAt     string `json:"edited_at"`
	EditedBy     string `json:"edited_by,omitempty"`
}

// EffectiveSnapshot returns a deep-merged ConfigSet representing the union of
// inherited snapshots, own sets, and local edits at this layer.
// This is what the next layer copies on create.
func (b *ConfigSetBundle) EffectiveSnapshot() ConfigSet {
	merged := ConfigSet{
		SchemaVersion: 1,
		Items:         make(map[string]ConfigItem),
	}

	// Layer 1: apply inherited snapshots
	for _, snap := range b.InheritedBundleSnapshots {
		for k, v := range snap.Items {
			v.AlignTiers()
			merged.Items[k] = v
		}
	}

	// Layer 2: apply own sets (overwrite inherited by key)
	for _, set := range b.OwnSets {
		for k, v := range set.Items {
			v.AlignTiers()
			merged.Items[k] = v
		}
	}

	// Layer 3: apply local edits
	for _, edits := range b.LocalEdits {
		for itemKey, edit := range edits {
			if existing, ok := merged.Items[itemKey]; ok {
				if edit.LocalValue != nil {
					existing.Value_.LocalValue = edit.LocalValue
					existing.Value_.EffectiveValue = edit.LocalValue
				}
				if edit.Enabled != nil {
					existing.State_.Enabled = *edit.Enabled
				}
				if edit.Checked != nil {
					existing.State_.Checked = *edit.Checked
				}
				existing.Provenance_.ValueSource = "local_edit"
				existing.Provenance_.LastValueLayer = "current"
				merged.Items[itemKey] = existing
			}
		}
	}

	return merged
}

// DeepCopySnapshot returns a deep copy of the effective snapshot suitable for
// passing to the next layer on copy-on-create.
func (b *ConfigSetBundle) DeepCopySnapshot(layer, id string) ConfigSet {
	snap := b.EffectiveSnapshot()
	// Stamp snapshot provenance on every item
	now := "" // filled by caller or DB trigger
	items := make(map[string]ConfigItem, len(snap.Items))
	for k, v := range snap.Items {
		v.Snapshot_ = ConfigItemSnapshot{
			FromLayer: layer,
			FromID:    id,
			Version:   1,
			CopiedAt:  now,
		}
		// Inherited items retain their schema owner; schema/snapshot fields are readonly
		// at the child layer.
		items[k] = v
	}
	snap.Items = items
	return snap
}

type Registry struct {
	SchemaVersion int          `yaml:"schema_version"`
	Items         []ConfigItem `yaml:"items"`
	byCode        map[string]ConfigItem
}

func (r *Registry) Item(code string) (ConfigItem, bool) {
	if r == nil {
		return ConfigItem{}, false
	}
	item, ok := r.byCode[code]
	return item, ok
}

func (r *Registry) MaterializeBase(layer, ref string) map[string]ConfigItem {
	items := make(map[string]ConfigItem, len(r.Items))
	for _, item := range r.Items {
		copied := item
		copied.Source = map[string]string{
			"layer":  layer,
			"ref":    ref,
			"reason": "registry_default",
		}
		copied.AlignTiers()
		items[item.Code] = copied
	}
	return items
}

func (cs ConfigSet) JSON() (string, error) {
	b, err := json.Marshal(cs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type BackendCatalog struct {
	Root     string
	Backends []BackendDoc
	Versions []VersionDoc
	Runtimes []RuntimeDoc
}

type BackendDoc struct {
	ID                    string         `yaml:"id"`
	Slug                  string         `yaml:"slug"`
	Name                  string         `yaml:"name"`
	ManagedBy             string         `yaml:"managed_by"`
	SupportedModelFormats []string       `yaml:"supported_model_formats"`
	Protocols             []string       `yaml:"protocols"`
	DefaultHealthCheck    map[string]any `yaml:"default_health_check"`
	SourcePath            string         `yaml:"-"`
	SourceHash            string         `yaml:"-"`
}

type VersionDoc struct {
	ID                 string           `yaml:"id"`
	BackendID          string           `yaml:"backend_id"`
	Slug               string           `yaml:"slug"`
	Version            string           `yaml:"version"`
	ManagedBy          string           `yaml:"managed_by"`
	Source             string           `yaml:"source"`
	Readonly           bool             `yaml:"readonly"`
	Protocol           string           `yaml:"protocol"`
	ImageCandidates    []string         `yaml:"image_candidates"`
	DefaultPort        int              `yaml:"default_port"`
	DefaultHost        string           `yaml:"default_host"`
	DefaultModelMount  map[string]any   `yaml:"default_model_mount"`
	DefaultEndpoints   map[string]any   `yaml:"default_endpoints"`
	Capabilities       []string         `yaml:"capabilities"`
	CapabilitiesDetail any              `yaml:"capabilities_detail"`
	DefaultEntrypoint  []string         `yaml:"default_entrypoint"`
	DefaultCommand     []string         `yaml:"default_command"`
	DefaultArgs        []string         `yaml:"default_args"`
	DefaultArgsSchema  []map[string]any `yaml:"default_args_schema"`
	HealthCheck        map[string]any   `yaml:"health_check"`
	VendorOptions      map[string]any   `yaml:"vendor_options"`
	OfficialReference  []any            `yaml:"official_reference_note"`
	SourcePath         string           `yaml:"-"`
	SourceHash         string           `yaml:"-"`
}

type RuntimeDoc struct {
	ID                         string            `yaml:"id"`
	Name                       string            `yaml:"name"`
	DisplayName                string            `yaml:"display_name"`
	BackendID                  string            `yaml:"backend_id"`
	BackendVersionID           string            `yaml:"backend_version_id"`
	Slug                       string            `yaml:"slug"`
	ManagedBy                  string            `yaml:"managed_by"`
	Source                     string            `yaml:"source"`
	Readonly                   bool              `yaml:"readonly"`
	Visibility                 string            `yaml:"visibility"`
	SupportLevel               string            `yaml:"support_level"`
	Status                     string            `yaml:"status"`
	Vendor                     string            `yaml:"vendor"`
	HardwareFamily             string            `yaml:"hardware_family"`
	AcceleratorAPI             string            `yaml:"accelerator_api"`
	RuntimeDistribution        string            `yaml:"runtime_distribution"`
	RuntimeDistributionVersion string            `yaml:"runtime_distribution_version"`
	Compatibility              map[string]any    `yaml:"compatibility"`
	ImageRef                   string            `yaml:"image_ref"`
	ImageCandidates            []string          `yaml:"image_candidates"`
	ImageNote                  string            `yaml:"image_note"`
	RunnerType                 string            `yaml:"runner_type"`
	ModelMount                 map[string]any    `yaml:"model_mount"`
	DockerOptions              map[string]any    `yaml:"docker_options"`
	Devices                    map[string]any    `yaml:"devices"`
	Volumes                    map[string]any    `yaml:"volumes"`
	Env                        map[string]string `yaml:"env"`
	EnvSchema                  []map[string]any  `yaml:"env_schema"`
	Entrypoint                 []string          `yaml:"entrypoint"`
	Args                       []string          `yaml:"args"`
	ArgsDefaults               []map[string]any  `yaml:"args_defaults"`
	Ports                      []map[string]any  `yaml:"ports"`
	HealthCheck                map[string]any    `yaml:"health_check"`
	HighRiskFlags              map[string]any    `yaml:"high_risk_flags"`
	Verification               map[string]any    `yaml:"verification"`
	SourcePath                 string            `yaml:"-"`
	SourceHash                 string            `yaml:"-"`
}
