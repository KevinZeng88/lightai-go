# Codex Review Fix Plan — ConfigSetBundle Final Decision

## 1. Purpose

This document records the user and ChatGPT decision after Codex produced `ACCEPT_WITH_FIXES` in `13-codex-review.md`.

Codex correctly identified seven pre-AUTORUN gaps: ConfigSet to final parameter model path; exact `parameter_source_map`; Docker subfield enabled/checked filtering; shared RunPlan builder; test matrix; fresh DB/rebuild and catalog policy; closeout open issue status format.

The decision accepts these findings but rejects a compatibility-style interim design.

## 2. Decision Summary

```text
ACCEPT Codex findings.
REJECT old-code compatibility.
REJECT ConfigSet-backed interim compatibility layer.
KEEP ConfigSet as final-domain concept.
ADOPT ConfigSetBundle + self-describing ConfigSet + ConfigItem field tiers.
```

## 3. ConfigSet is not seed-only

ConfigSet must not be downgraded to a seed-only source. ConfigSet is a first-class final-domain concept. Current old implementation forms such as `config_set_json`, `config_overrides_json`, old ConfigEdit shapes, legacy RuntimeParameterEditor mappings, or mixed schema/value/enabled structures are not automatically accepted as final design. If old structures conflict with this final model, they must be cleaned, deleted, or replaced. Fresh DB rebuild is allowed.

## 4. ConfigSetBundle

Each domain layer owns ConfigSetBundle:

```text
ConfigSetBundle = inherited_bundle_snapshots[] + own_sets[] + local_edits[] + effective_view
```

Layer creation rule:

```text
next_layer = deep_copy(parent.effective_bundle_snapshot) + current_layer.own_sets + current_layer.local_edits
```

## 5. ConfigSet

ConfigSet is self-describing, self-presenting, composable, copy-on-create compatible, and RunPlan mappable. A ConfigSet can contain child ConfigSets. The parent ConfigSet defines where and how child ConfigSets are used. The child ConfigSet is responsible for its own internal grouping, rendering metadata, validation, and RunPlan hints.

## 6. ConfigItem field tiers

Every ConfigItem is divided into schema, value, state, provenance, snapshot, presentation. schema and snapshot are readonly after copy; value and state are current-layer editable; provenance tracks source; presentation defines display hints.

## 7. Remove overridable_at as core rule

Do not maintain complex `overridable_at` rules. Default rule: if a ConfigItem exists in the current layer snapshot, the current layer can change its value/state. Special read-only cases use `schema.read_only=true` or `state.editable=false`.

## 8. Copy schema, do not redefine schema

copy-on-create may copy schema fields into the child snapshot. The copied schema fields are readonly. The copied item owner remains unchanged. The current layer cannot redefine inherited schema.

## 9. Presentation decision

External pages display Config / ConfigView / ConfigPanel. Parent ConfigSet renders its own sections, invokes child ConfigSet views according to child_slots, and child ConfigSet renders/explains itself. Default UI uses GenericConfigSetRenderer. Complex sets can use CustomRendererRegistry, but custom renderer cannot bypass ConfigItem field-tier rules.

## 10. RunPlan decision

RunPlan reads only DeploymentConfigBundle effective snapshot. Required: one shared builder, parameter_source_map, source_chain, plan_hash, preview/start consistency.

## 11. Docker decision

Docker subfields are ConfigItems or structured items inside DockerOptionsConfigSet. Old `enabled_fields` should not survive as a standalone compatibility mechanism. unchecked optional Docker item does not enter final Docker spec; state.enabled=false filters optional Docker item; system_generated Docker item may enter final spec with source; every Docker final field must be represented in parameter_source_map.

## 12. Fresh DB and non-compatibility policy

This stage permits deleting `/tmp/lightai/data/lightai.db`, recreating schema, reseeding catalog/registry into final ConfigSetBundle model, removing old API fields, removing old UI branches, and removing old resolver fallbacks. No historical compatibility is required.

## 13. Required doc/code review before Claude AUTORUN

Before Claude implementation AUTORUN, Codex should review whether the revised docs now answer all seven issues from `13-codex-review.md` and whether the new ConfigSetBundle model is specific enough to implement safely.
