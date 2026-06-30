package configedit

type ProjectInput struct {
	ConfigSet   map[string]any
	Layer       string
	ObjectKind  string
	ObjectID    string
	ObjectLabel string
	Readonly    bool
	Mode        string
}

type ConfigEditView struct {
	Layer       string                `json:"layer"`
	ObjectID    string                `json:"object_id"`
	ObjectKind  string                `json:"object_kind"`
	Readonly    bool                  `json:"readonly"`
	Sections    []EditSection         `json:"sections"`
	Diagnostics ConfigEditDiagnostics `json:"diagnostics,omitempty"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
}

type ConfigEditDiagnostics struct {
	RawConfigSet map[string]any `json:"raw_config_set,omitempty"`
}

type EditSection struct {
	Key         string      `json:"key"`
	Label       string      `json:"label"`
	Description string      `json:"description,omitempty"`
	Order       int         `json:"order"`
	Advanced    bool        `json:"advanced,omitempty"`
	Collapsed   bool        `json:"collapsed,omitempty"`
	Fields      []EditField `json:"fields"`
}

type EditField struct {
	Key                string         `json:"key"`
	InternalKey        string         `json:"internal_key"`
	SemanticKey        string         `json:"semantic_key,omitempty"`
	Owner              string         `json:"owner,omitempty"`
	Tier               string         `json:"tier,omitempty"`
	ParentKey          string         `json:"parent_key,omitempty"`
	Path               []string       `json:"path,omitempty"`
	Label              string         `json:"label"`
	LabelI18nKey       string         `json:"label_i18n_key,omitempty"`
	TitleI18nKey       string         `json:"title_i18n_key,omitempty"`
	DescriptionI18nKey string         `json:"description_i18n_key,omitempty"`
	HelpI18nKey        string         `json:"help_i18n_key,omitempty"`
	TooltipI18nKey     string         `json:"tooltip_i18n_key,omitempty"`
	Title              string         `json:"title,omitempty"`
	Description        string         `json:"description,omitempty"`
	Help               string         `json:"help,omitempty"`
	CliFlag            string         `json:"cli_flag,omitempty"`
	EnvKey             string         `json:"env_key,omitempty"`
	TechnicalKey       string         `json:"technical_key,omitempty"`
	Section            string         `json:"section"`
	Group              string         `json:"group,omitempty"`
	Order              int            `json:"order"`
	Type               string         `json:"type"`
	Widget             string         `json:"widget"`
	Value              any            `json:"value"`
	DefaultValue       any            `json:"default_value,omitempty"`
	Enabled            bool           `json:"enabled"`
	HasEnable          bool           `json:"has_enable"`
	Required           bool           `json:"required"`
	Readonly           bool           `json:"readonly"`
	Advanced           bool           `json:"advanced"`
	Visibility         string         `json:"visibility"`
	Options            []EditOption   `json:"options,omitempty"`
	Constraints        map[string]any `json:"constraints,omitempty"`
	ValidationRules    map[string]any `json:"validation_rules,omitempty"`
	Placeholder        string         `json:"placeholder,omitempty"`
	Sensitive          bool           `json:"sensitive,omitempty"`
	Disabled           bool           `json:"disabled,omitempty"`
	Source             map[string]any `json:"source,omitempty"`
	ValueSource        string         `json:"value_source,omitempty"`
	LastValueLayer     string         `json:"last_value_layer,omitempty"`
	InheritedValue     any            `json:"inherited_value,omitempty"`
	CopyBehavior       string         `json:"copy_behavior,omitempty"`
	OverrideBehavior   string         `json:"override_behavior,omitempty"`
	DisableBehavior    string         `json:"disable_behavior,omitempty"`
	PatchTarget        string         `json:"patch_target,omitempty"`
	CopiedFrom         string         `json:"copied_from,omitempty"`
	Dirty              bool           `json:"dirty,omitempty"`
	Warnings           []any          `json:"warnings,omitempty"`
	Diagnostic         bool           `json:"diagnostic,omitempty"`
	OriginalValue      any            `json:"original_value,omitempty"`
	OriginalEnabled    bool           `json:"original_enabled"`
}

type EditOption struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

type ConfigEditPatch struct {
	Layer    string           `json:"layer"`
	ObjectID string           `json:"object_id"`
	Fields   []EditFieldPatch `json:"fields"`
}

type EditFieldPatch struct {
	Key         string   `json:"key"`
	InternalKey string   `json:"internal_key"`
	Path        []string `json:"path,omitempty"`
	Value       any      `json:"value"`
	Enabled     *bool    `json:"enabled,omitempty"`
}
