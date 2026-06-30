package catalog

import "encoding/json"

// ============================================================================
// ConfigItem — final field-tier model (schema / value / state / provenance / snapshot / presentation)
// ============================================================================

// ConfigItemSchema holds immutable definition fields. After copy-on-create, inherited
// items retain their original Schema values; the current layer must not modify them.
type ConfigItemSchema struct {
	Key                string         `json:"key"`
	Owner              string         `json:"owner"`
	OwnerLayer         string         `json:"owner_layer"`
	ConfigSetKey       string         `json:"config_set_key"`
	Category           string         `json:"category"`
	Label              string         `json:"label"`
	LabelI18nKey       string         `json:"label_i18n_key,omitempty"`
	Description        string         `json:"description,omitempty"`
	DescriptionI18nKey string         `json:"description_i18n_key,omitempty"`
	Type               string         `json:"type"`
	Kind               string         `json:"kind"`
	Target             string         `json:"target,omitempty"`
	ArgName            string         `json:"arg_name,omitempty"`
	EnvName            string         `json:"env_name,omitempty"`
	MountTarget        string         `json:"mount_target,omitempty"`
	PortTarget         string         `json:"port_target,omitempty"`
	Constraints        map[string]any `json:"constraints,omitempty"`
	Choices            []any          `json:"choices,omitempty"`
	Required           bool           `json:"required"`
	Advanced           bool           `json:"advanced"`
	DisplayOrder       int            `json:"display_order"`
	ReadOnly           bool           `json:"read_only"`
	HelpText           string         `json:"help_text,omitempty"`
	HelpI18nKey        string         `json:"help_i18n_key,omitempty"`
	TooltipI18nKey     string         `json:"tooltip_i18n_key,omitempty"`
	SupportLevel       string         `json:"support_level"`
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
type ConfigItem struct {
	Schema       ConfigItemSchema       `json:"schema"`
	Value_       ConfigItemValue        `json:"value"`
	State_       ConfigItemState        `json:"state"`
	Provenance_  ConfigItemProvenance   `json:"provenance"`
	Snapshot_    ConfigItemSnapshot     `json:"snapshot"`
	Presentation ConfigItemPresentation `json:"presentation"`
}

// RegistryItem is the YAML shape from configs/config-registry/items.yaml.
// It is converted to a ConfigItem with tiered fields during MaterializeBase.
type RegistryItem struct {
	Code               string                 `yaml:"code"`
	Category           string                 `yaml:"category"`
	Kind               string                 `yaml:"kind"`
	Type               string                 `yaml:"type"`
	Label              string                 `yaml:"label"`
	LabelI18nKey       string                 `yaml:"label_i18n_key"`
	Description        string                 `yaml:"description"`
	DescriptionI18nKey string                 `yaml:"description_i18n_key"`
	Help               string                 `yaml:"help"`
	HelpI18nKey        string                 `yaml:"help_i18n_key"`
	TooltipI18nKey     string                 `yaml:"tooltip_i18n_key"`
	Required           bool                   `yaml:"required"`
	Visibility         string                 `yaml:"visibility"`
	Readonly           bool                   `yaml:"readonly"`
	Advanced           bool                   `yaml:"advanced"`
	Value              any                    `yaml:"value"`
	DefaultValue       any                    `yaml:"default_value"`
	Enabled            bool                   `yaml:"enabled"`
	Render             map[string]any         `yaml:"render"`
	Order              int                    `yaml:"order"`
	Constraints        map[string]any         `yaml:"constraints"`
	SupportLevel       string                 `yaml:"support_level"`
	Source             map[string]string      `yaml:"source"`
	LastModified       map[string]string      `yaml:"last_modified"`
	Extensions         map[string]interface{} `yaml:"extensions"`
}

// ToConfigItem converts a YAML registry item into a tiered ConfigItem.
func (ri RegistryItem) ToConfigItem() ConfigItem {
	ci := ConfigItem{
		Schema: ConfigItemSchema{
			Key:                ri.Code,
			Category:           ri.Category,
			Kind:               ri.Kind,
			Type:               ri.Type,
			Label:              ri.Label,
			LabelI18nKey:       ri.LabelI18nKey,
			Description:        ri.Description,
			DescriptionI18nKey: ri.DescriptionI18nKey,
			HelpText:           ri.Help,
			HelpI18nKey:        ri.HelpI18nKey,
			TooltipI18nKey:     ri.TooltipI18nKey,
			Required:           ri.Required,
			Advanced:           ri.Advanced,
			ReadOnly:           ri.Readonly,
			DisplayOrder:       ri.Order,
			SupportLevel:       ri.SupportLevel,
			Constraints:        ri.Constraints,
			ConfigSetKey:       "",
			Owner:              "",
			OwnerLayer:         "",
		},
		Value_: ConfigItemValue{
			DefaultValue: ri.DefaultValue,
		},
		State_: ConfigItemState{
			Enabled:  ri.Enabled,
			Checked:  ri.Enabled,
			Editable: !ri.Readonly,
			Visible:  ri.Visibility != "hidden",
			Valid:    true,
		},
	}

	if ri.Render != nil {
		ci.Schema.Target = strVal(ri.Render["target"])
		ci.Schema.ArgName = strVal(ri.Render["flag"])
		ci.Schema.EnvName = strVal(ri.Render["env_name"])
	}
	if ri.Extensions != nil {
		if l, ok := ri.Extensions["label"].(string); ok {
			ci.Schema.Label = l
		}
		if g, ok := ri.Extensions["group"].(string); ok {
			ci.Presentation.Group = g
		}
	}

	// Value: apply flat value into tiered
	if ri.Value != nil {
		if ri.Enabled {
			ci.Value_.LocalValue = ri.Value
			ci.Value_.EffectiveValue = ri.Value
		} else {
			ci.Value_.InheritedValue = ri.Value
			ci.Value_.EffectiveValue = ri.Value
		}
	} else {
		ci.Value_.EffectiveValue = ri.DefaultValue
	}

	if ci.Presentation.Priority == 0 {
		ci.Presentation.Priority = ri.Order
	}

	return ci
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
	View              string `json:"view"`         // summary, summary_then_edit, edit, preview
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
			merged.Items[k] = v
		}
	}

	// Layer 2: apply own sets (overwrite inherited by key)
	for _, set := range b.OwnSets {
		for k, v := range set.Items {
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
	SchemaVersion int            `yaml:"schema_version"`
	Items         []RegistryItem `yaml:"items"`
	byCode        map[string]RegistryItem
}

func (r *Registry) Item(code string) (RegistryItem, bool) {
	if r == nil {
		return RegistryItem{}, false
	}
	item, ok := r.byCode[code]
	return item, ok
}

func (r *Registry) MaterializeBase(layer, ref string) map[string]ConfigItem {
	items := make(map[string]ConfigItem, len(r.Items))
	for _, item := range r.Items {
		ci := item.ToConfigItem()
		ci.Provenance_ = ConfigItemProvenance{
			ValueSource:      layer,
			LastValueLayer:   layer,
			LastValueOwnerID: ref,
		}
		items[item.Code] = ci
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
