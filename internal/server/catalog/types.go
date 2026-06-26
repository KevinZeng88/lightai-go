package catalog

import "encoding/json"

type ConfigItem struct {
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

type ConfigSet struct {
	SchemaVersion  int                    `json:"schema_version"`
	Context        map[string]string      `json:"context"`
	Items          map[string]ConfigItem  `json:"items"`
	SourceMetadata map[string]interface{} `json:"source_metadata"`
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
