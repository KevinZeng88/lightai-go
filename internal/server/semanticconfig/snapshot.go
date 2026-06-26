package semanticconfig

import (
	"time"
)

type SnapshotBuilder struct {
	reg *Registry
}

type BuildInput struct {
	SourceKind string
	SourceID   string
	TargetKind string
	TargetID   string
	ConfigSet  map[string]any
	Snapshot   Snapshot
	Values     map[string]any
}

type DeploymentBuildInput struct {
	SourceKind string
	SourceID   string
	TargetID   string
	Snapshot   Snapshot
	Service    ServiceInput
	ModelFacts ModelFacts
	Values     map[string]any
}

type ServiceInput struct {
	HostPort        int
	ContainerPort   int
	ServedModelName string
}

type ModelFacts struct {
	ContextLength int
}

type PatchField struct {
	Key     string
	Value   any
	Enabled *bool
}

func NewSnapshotBuilder(reg *Registry) *SnapshotBuilder {
	if reg == nil {
		reg = DefaultRegistry()
	}
	return &SnapshotBuilder{reg: reg}
}

func (b *SnapshotBuilder) BuildBackendRuntimeSnapshot(input BuildInput) (Snapshot, error) {
	base, err := b.baseSnapshot(input)
	if err != nil {
		return Snapshot{}, err
	}
	return b.copySnapshot(base, input.SourceKind, input.SourceID, input.TargetKind, input.TargetID, input.Values)
}

func (b *SnapshotBuilder) BuildNodeBackendRuntimeSnapshot(input BuildInput) (Snapshot, error) {
	base, err := b.baseSnapshot(input)
	if err != nil {
		return Snapshot{}, err
	}
	return b.copySnapshot(base, input.SourceKind, input.SourceID, input.TargetKind, input.TargetID, input.Values)
}

func (b *SnapshotBuilder) BuildDeploymentSnapshot(input DeploymentBuildInput) (Snapshot, error) {
	values := map[string]any{}
	for key, value := range input.Values {
		values[key] = value
	}
	if input.Service.HostPort != 0 {
		values["deployment.host_port"] = input.Service.HostPort
	}
	if input.Service.ContainerPort != 0 {
		values["service.container_port"] = input.Service.ContainerPort
	}
	if input.Service.ServedModelName != "" {
		values["deployment.served_model_name"] = input.Service.ServedModelName
	}
	if input.ModelFacts.ContextLength != 0 {
		values["model_runtime.context_length"] = input.ModelFacts.ContextLength
		if _, ok := values["model_runtime.max_model_len"]; !ok {
			values["model_runtime.max_model_len"] = input.ModelFacts.ContextLength
		}
	}
	return b.copySnapshot(input.Snapshot, input.SourceKind, input.SourceID, "Deployment", input.TargetID, values)
}

func ApplyPatch(reg *Registry, snapshot Snapshot, fields []PatchField) (Snapshot, error) {
	if reg == nil {
		reg = DefaultRegistry()
	}
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.Key)
	}
	if err := ValidateSnapshotPatch(reg, snapshot, fields); err != nil {
		return Snapshot{}, err
	}
	out := cloneSnapshot(snapshot)
	for _, field := range fields {
		def, _ := reg.Get(field.Key)
		item := out.Items[field.Key]
		if item.Key == "" {
			item = itemFromDefinition(def, field.Value, field.Value, true)
		}
		item.Value = field.Value
		if field.Enabled != nil {
			item.Enabled = *field.Enabled
		}
		item.Dirty = true
		item.Source = map[string]any{
			"reason":     "semantic_patch",
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		}
		out.Items[field.Key] = item
	}
	return out, nil
}

func DerivedServiceJSON(snapshot Snapshot) map[string]any {
	out := map[string]any{}
	if item, ok := snapshot.Items["deployment.host_port"]; ok {
		out["host_port"] = item.Value
	}
	if item, ok := snapshot.Items["service.container_port"]; ok {
		out["container_port"] = item.Value
	}
	if item, ok := snapshot.Items["deployment.served_model_name"]; ok {
		out["served_model_name"] = item.Value
	}
	return out
}

func (b *SnapshotBuilder) baseSnapshot(input BuildInput) (Snapshot, error) {
	if len(input.Snapshot.Items) > 0 {
		return input.Snapshot, nil
	}
	return NormalizeConfigSet(b.reg, input.ConfigSet)
}

func (b *SnapshotBuilder) copySnapshot(source Snapshot, sourceKind, sourceID, targetKind, targetID string, values map[string]any) (Snapshot, error) {
	out := cloneSnapshot(source)
	out.Context = cloneStringMap(out.Context)
	if out.Context == nil {
		out.Context = map[string]string{}
	}
	out.Context["object_kind"] = targetKind
	out.Context["object_id"] = targetID
	now := time.Now().UTC().Format(time.RFC3339)
	for key, item := range out.Items {
		item.CopiedFrom = sourceKind + ":" + sourceID
		item.SourceSnapshot = sourceID
		item.CopiedAt = now
		item.Dirty = false
		out.Items[key] = item
	}
	if err := ValidatePatchKeys(b.reg, keysFromMap(values)); err != nil {
		return Snapshot{}, err
	}
	for key, value := range values {
		def, _ := b.reg.Get(key)
		item := out.Items[key]
		if item.Key == "" {
			item = itemFromDefinition(def, value, value, true)
		}
		item.Value = value
		item.DefaultValue = firstNonNil(item.DefaultValue, value)
		item.Enabled = true
		item.CopiedFrom = sourceKind + ":" + sourceID
		item.SourceSnapshot = sourceID
		item.CopiedAt = now
		item.Dirty = false
		out.Items[key] = item
	}
	return out, nil
}

func itemFromDefinition(def Definition, value, defaultValue any, enabled bool) SnapshotItem {
	return SnapshotItem{
		Key:          def.Key,
		Owner:        def.Owner,
		Type:         def.ValueType,
		DisplayTier:  def.DisplayTier,
		Label:        def.Label,
		Value:        value,
		DefaultValue: defaultValue,
		Enabled:      enabled,
	}
}

func keysFromMap(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func cloneSnapshot(in Snapshot) Snapshot {
	out := Snapshot{
		SchemaVersion: in.SchemaVersion,
		Context:       cloneStringMap(in.Context),
		Items:         make(map[string]SnapshotItem, len(in.Items)),
		Warnings:      append([]Warning(nil), in.Warnings...),
	}
	for key, item := range in.Items {
		copied := item
		copied.Warnings = append([]Warning(nil), item.Warnings...)
		copied.Source = cloneAnyMap(item.Source)
		out.Items[key] = copied
	}
	if out.Items == nil {
		out.Items = map[string]SnapshotItem{}
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
