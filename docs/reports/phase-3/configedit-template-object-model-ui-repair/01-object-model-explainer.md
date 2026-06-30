# ConfigEdit Template Object Model — Explainer for Codex

## 1. Why this object model exists

LightAI Go has several places where users edit runtime-related configuration:

- model library / model location related settings
- backend runtime templates
- node backend runtimes
- model deployments
- run plan previews
- Docker launcher options
- health checks
- volume mounts
- environment variables
- backend-specific command arguments such as vLLM, SGLang, and llama.cpp flags

Earlier bugs showed that parameters could disappear, fall back to raw JSON-only, or be shown only when a page had a hard-coded allowlist. The ConfigEdit template object model is intended to remove that class of bug.

The intended architecture is:

```text
Catalog / BackendVersion / BackendRuntime / NBR / Deployment data
        ↓
ConfigEdit template registry and materialization
        ↓
ConfigEditTemplate objects with fields, sections, levels, labels, source metadata
        ↓
Reusable ConfigEdit / RuntimeParameterEditor UI projection
        ↓
Runtime template, NBR, deployment and diagnostics pages
```

The model must support both known catalog fields and fallback-materialized fields. A field missing from a page-specific whitelist must still be representable as a structured ConfigEdit field, not silently hidden and not forced into raw JSON-only display.

## 2. Important terms

### ConfigEdit Template

A normalized template object describing editable parameters. It should include identity, backend/scope metadata, source metadata, sections, fields, labels, help text, display level, ordering, risk, and default behavior.

Suggested conceptual shape:

```text
ConfigEditTemplate {
  id/key
  display_name / label_i18n_key
  backend_key or backend_version_key when applicable
  scope: model | backend_runtime | node_backend_runtime | deployment | launcher | health | mount | env | mixed
  source: catalog | backend_version | backend_runtime | node_backend_runtime | deployment | fallback_materialized
  sections: ConfigEditSection[]
}
```

This is a conceptual contract. Codex should inspect the existing code and adapt to the actual names/types already used in the repository.

### ConfigEdit Section

A logical group of fields. Examples:

```text
model
runtime
resource
service
health
mount
env
docker
security
raw
```

Sections are used for user navigation, grouping, and stable ordering.

### ConfigEdit Field

A normalized editable parameter. Examples:

```text
backend.arg.max_model_len
backend.arg.gpu_memory_utilization
backend.arg.mem_fraction_static
backend.arg.ctx_size
launcher.docker_options.privileged
launcher.docker_options.security_options
launcher.docker_options.cap_add
runtime.env.CUDA_VISIBLE_DEVICES
health.path
mounts.model_path
```

Suggested conceptual shape:

```text
ConfigEditField {
  key/path
  label_i18n_key
  help_i18n_key
  value_type: string | number | boolean | enum | string_array | object | json
  input_type: text | number | checkbox | select | list | json
  section
  tier: normal | advanced | developer
  risk_level: normal | warning | high
  display_order
  default_value
  source
  enabled_default
  validation
}
```

### Parameter value state

The value state should preserve whether a field is enabled separately from its value. This is required because a disabled parameter may still have a saved value or default value that should be visible but inactive.

Suggested conceptual shape:

```text
ParameterValue {
  enabled: boolean
  value: any
  source: inherited | template_default | runtime_override | nbr_override | deployment_override
}
```

## 3. View levels: Normal / Advanced / Developer

The UI currently shows `Normal`, `Advanced`, and `Developer`. These are display levels, not runtime modes and not separate configuration profiles.

Product labels should be:

```text
Normal    -> 常用
Advanced  -> 高级
Developer -> 专家
```

Expected filtering behavior:

```text
常用: show normal fields
高级: show normal + advanced fields
专家: show normal + advanced + developer fields
```

Developer/expert fields include low-level, raw, source-map, fallback, Docker security, raw args, raw env, and diagnostic fields.

## 4. ConfigEdit Templates page role

The `ConfigEdit Templates` page is a registry/observability surface for the object model. It should help answer:

- Which parameter templates exist?
- Which backend or scope does each template apply to?
- Which source produced the template?
- Which fields belong to each template?
- Which tier, section, risk level, default value, and source does each field have?
- Are fallback-materialized fields visible as structured fields?
- Are vLLM, SGLang, llama.cpp, Docker launcher, health check, mount, env, and resource-control templates registered?

The page can be developer/admin-oriented, but it must not be a blank English page in the main product. If exposed, it must have working data and localized UI.

## 5. Required invariants

These invariants should hold after the repair:

1. A parameter available to runtime editing should also be explainable through the ConfigEdit template model.
2. Unknown or newly introduced fields should be fallback-materialized into structured fields when possible.
3. Raw JSON is allowed only as expert/diagnostic fallback, not as the default user experience for ordinary fields.
4. Sorting and grouping must be driven by template metadata or shared ConfigEdit utilities, not by page-specific hard-coded lists.
5. Enabled state and value must be preserved separately.
6. View level selection must consistently filter fields across all ConfigEdit consumers.
7. i18n labels must come from stable keys or deterministic fallback rules. Raw English UI labels must not leak into zh-CN product pages.
8. The ConfigEdit Templates page must display the real template registry or a truthful, actionable error/empty state.
