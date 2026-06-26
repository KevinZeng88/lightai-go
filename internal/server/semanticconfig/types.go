package semanticconfig

type Owner string

const (
	OwnerRuntimeEnvironment Owner = "runtime_environment"
	OwnerRuntimeService     Owner = "runtime_service"
	OwnerDeploymentExposure Owner = "deployment_exposure"
	OwnerDeploymentService  Owner = "deployment_service"
	OwnerModelRuntime       Owner = "model_runtime"
	OwnerModelArtifact      Owner = "model_artifact"
	OwnerSchedulerResource  Owner = "scheduler_resource"
	OwnerBackendCapability  Owner = "backend_capability"
)

type DisplayTier string

const (
	TierRequired                 DisplayTier = "required"
	TierCommon                   DisplayTier = "common"
	TierRecommended              DisplayTier = "recommended"
	TierAdvanced                 DisplayTier = "advanced"
	TierDiagnostic               DisplayTier = "diagnostic"
	TierDeploymentCommonAdvanced DisplayTier = "Deployment common/advanced"
)

type ValueType string

const (
	TypeString  ValueType = "string"
	TypeInteger ValueType = "integer"
	TypeNumber  ValueType = "number"
	TypeBoolean ValueType = "boolean"
	TypeArray   ValueType = "array"
	TypeObject  ValueType = "object"
)

type Definition struct {
	Key              string
	Owner            Owner
	ValueType        ValueType
	DisplayTier      DisplayTier
	Label            string
	LegacyKeys       []string
	DefaultSource    string
	WarningRules     []string
	HardValidation   []string
	ResolverMappings map[string]string
}

type WarningCode string

const (
	WarningConflict         WarningCode = "conflict"
	WarningLegacyNormalized WarningCode = "legacy_normalized"
)

type Warning struct {
	Code        WarningCode `json:"code"`
	SemanticKey string      `json:"semantic_key"`
	LegacyKey   string      `json:"legacy_key,omitempty"`
	Message     string      `json:"message"`
}

type SnapshotItem struct {
	Key            string         `json:"key"`
	Owner          Owner          `json:"owner"`
	Type           ValueType      `json:"type"`
	DisplayTier    DisplayTier    `json:"display_tier"`
	Label          string         `json:"label,omitempty"`
	Value          any            `json:"value,omitempty"`
	DefaultValue   any            `json:"default_value,omitempty"`
	Enabled        bool           `json:"enabled"`
	Source         map[string]any `json:"source,omitempty"`
	CopiedFrom     string         `json:"copied_from,omitempty"`
	SourceSnapshot string         `json:"source_snapshot,omitempty"`
	CopiedAt       string         `json:"copied_at,omitempty"`
	Dirty          bool           `json:"dirty"`
	Warnings       []Warning      `json:"warnings,omitempty"`
	Diagnostic     bool           `json:"diagnostic,omitempty"`
}

type Snapshot struct {
	SchemaVersion int                     `json:"schema_version"`
	Context       map[string]string       `json:"context,omitempty"`
	Items         map[string]SnapshotItem `json:"items"`
	Warnings      []Warning               `json:"warnings,omitempty"`
}
