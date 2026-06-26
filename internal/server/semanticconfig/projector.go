package semanticconfig

import "sort"

type ProjectOptions struct {
	ObjectKind string
	Layer      string
}

type ProjectedView struct {
	ObjectKind string             `json:"object_kind"`
	Layer      string             `json:"layer"`
	Sections   []ProjectedSection `json:"sections"`
}

type ProjectedSection struct {
	Key    string           `json:"key"`
	Label  string           `json:"label"`
	Order  int              `json:"order"`
	Fields []ProjectedField `json:"fields"`
}

type ProjectedField struct {
	Key             string      `json:"key"`
	SemanticKey     string      `json:"semantic_key"`
	Owner           Owner       `json:"owner"`
	Tier            DisplayTier `json:"tier"`
	Label           string      `json:"label"`
	Type            ValueType   `json:"type"`
	Value           any         `json:"value"`
	DefaultValue    any         `json:"default_value,omitempty"`
	Enabled         bool        `json:"enabled"`
	Required        bool        `json:"required"`
	Readonly        bool        `json:"readonly"`
	CopiedFrom      string      `json:"copied_from,omitempty"`
	SourceSnapshot  string      `json:"source_snapshot,omitempty"`
	Dirty           bool        `json:"dirty"`
	Warnings        []Warning   `json:"warnings,omitempty"`
	Diagnostic      bool        `json:"diagnostic,omitempty"`
	OriginalValue   any         `json:"original_value,omitempty"`
	OriginalEnabled bool        `json:"original_enabled"`
}

func ProjectSnapshot(reg *Registry, snapshot Snapshot, opts ProjectOptions) ProjectedView {
	if reg == nil {
		reg = DefaultRegistry()
	}
	warnings := EvaluateWarnings(reg, snapshot)
	sections := map[string]*ProjectedSection{}
	for key, item := range snapshot.Items {
		def, ok := reg.Get(key)
		if !ok {
			continue
		}
		if item.Key == "" {
			item.Key = key
		}
		item.Owner = firstOwner(item.Owner, def.Owner)
		item.Type = firstType(item.Type, def.ValueType)
		item.DisplayTier = firstTier(item.DisplayTier, def.DisplayTier)
		label := item.Label
		if label == "" {
			label = def.Label
		}
		fieldWarnings := append([]Warning(nil), item.Warnings...)
		fieldWarnings = append(fieldWarnings, warnings[key]...)
		sectionKey := sectionKeyForTier(item.DisplayTier)
		section := sections[sectionKey]
		if section == nil {
			section = &ProjectedSection{Key: sectionKey, Label: sectionLabel(sectionKey), Order: sectionOrder(sectionKey)}
			sections[sectionKey] = section
		}
		section.Fields = append(section.Fields, ProjectedField{
			Key:             key,
			SemanticKey:     key,
			Owner:           item.Owner,
			Tier:            item.DisplayTier,
			Label:           label,
			Type:            item.Type,
			Value:           item.Value,
			DefaultValue:    item.DefaultValue,
			Enabled:         item.Enabled,
			CopiedFrom:      item.CopiedFrom,
			SourceSnapshot:  item.SourceSnapshot,
			Dirty:           item.Dirty,
			Warnings:        fieldWarnings,
			Diagnostic:      item.Diagnostic || item.DisplayTier == TierDiagnostic,
			OriginalValue:   item.Value,
			OriginalEnabled: item.Enabled,
		})
	}
	out := ProjectedView{ObjectKind: opts.ObjectKind, Layer: opts.Layer}
	for _, section := range sections {
		sort.Slice(section.Fields, func(i, j int) bool { return section.Fields[i].Key < section.Fields[j].Key })
		out.Sections = append(out.Sections, *section)
	}
	sort.Slice(out.Sections, func(i, j int) bool { return out.Sections[i].Order < out.Sections[j].Order })
	return out
}

func sectionKeyForTier(tier DisplayTier) string {
	switch tier {
	case TierRequired:
		return "required"
	case TierCommon, TierDeploymentCommonAdvanced:
		return "common"
	case TierRecommended:
		return "recommended"
	case TierDiagnostic:
		return "diagnostic"
	default:
		return "advanced"
	}
}

func sectionLabel(key string) string {
	switch key {
	case "required":
		return "Required"
	case "common":
		return "Common"
	case "recommended":
		return "Recommended"
	case "diagnostic":
		return "Diagnostic"
	default:
		return "Advanced"
	}
}

func sectionOrder(key string) int {
	switch key {
	case "required":
		return 10
	case "common":
		return 20
	case "recommended":
		return 30
	case "advanced":
		return 40
	default:
		return 90
	}
}

func firstOwner(value, fallback Owner) Owner {
	if value != "" {
		return value
	}
	return fallback
}

func firstType(value, fallback ValueType) ValueType {
	if value != "" {
		return value
	}
	return fallback
}

func firstTier(value, fallback DisplayTier) DisplayTier {
	if value != "" {
		return value
	}
	return fallback
}
