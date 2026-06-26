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
	Key          string         `json:"key"`
	InternalKey  string         `json:"internal_key"`
	ParentKey    string         `json:"parent_key,omitempty"`
	Path         []string       `json:"path,omitempty"`
	Label        string         `json:"label"`
	Help         string         `json:"help,omitempty"`
	Section      string         `json:"section"`
	Group        string         `json:"group,omitempty"`
	Order        int            `json:"order"`
	Type         string         `json:"type"`
	Widget       string         `json:"widget"`
	Value        any            `json:"value"`
	DefaultValue any            `json:"default_value,omitempty"`
	Enabled      bool           `json:"enabled"`
	HasEnable    bool           `json:"has_enable"`
	Required     bool           `json:"required"`
	Readonly     bool           `json:"readonly"`
	Advanced     bool           `json:"advanced"`
	Visibility   string         `json:"visibility"`
	Options      []EditOption   `json:"options,omitempty"`
	Constraints  map[string]any `json:"constraints,omitempty"`
	Source       map[string]any `json:"source,omitempty"`
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
