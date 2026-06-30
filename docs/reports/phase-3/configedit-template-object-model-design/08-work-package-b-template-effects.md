# 08 — Work Package B: External ConfigEdit Component Template + Effect Engine

## Objective

Create the external ConfigEdit component template system and generic effect engine.

This package defines how ConfigEdit components are described outside Go/Vue code and converted into generic effects.

## Key distinction

Runtime template describes how a backend runs.

ConfigEdit component template describes how configuration objects are edited, inherited, validated, rendered, and compiled.

Keep these separate.

## Template roots

Add template loading from:

```text
configs/configedit-templates/builtin/
configs/configedit-templates/local/
```

Precedence:

```text
local override > built-in
```

The local override path should be reloadable or at least loadable without rebuilding the binary.

## Template contract

Support at least:

- `template_id`
- `kind: config_edit_template`
- template version
- backend applicability
- backend version applicability
- runtime kind applicability
- vendor applicability
- layer applicability
- sections
- components
- fields
- renderer types
- labels
- help text
- default values
- recommended values
- ranges
- options
- validation
- editability by layer
- copy-to-child behavior
- reset behavior
- view level
- effects

Supported view levels:

- normal
- advanced
- developer

## Required generic effect types

Support effects for:

- CLI args
- environment variables
- Docker options
- mounts
- ports
- health check
- device binding

Effects must be generic data, not backend-specific Go code.

Example direction:

```yaml
components:
  - key: model_runtime.gpu_memory_utilization
    type: field
    renderer: number
    view: normal
    label: GPU Memory Utilization
    help: Fraction of GPU memory available to the runtime.
    default: 0.9
    recommended: "0.85 - 0.95"
    min: 0.1
    max: 1.0
    step: 0.01
    editable_by_layer:
      backend_runtime: true
      node_backend_runtime: true
      deployment: true
    copy_to_child: true
    effects:
      - type: cli_arg
        flag: "--gpu-memory-utilization"
        value_from: self.value
        omit_if_empty: true
```

## Constrained expression language

If expressions are needed, implement a constrained expression evaluator.

Allowed:

- access to current component values
- access to sibling values by explicit reference
- simple string interpolation
- list join
- conditional omit
- numeric/string/boolean operations needed for templates

Forbidden:

- filesystem access
- network access
- shell/process execution
- arbitrary code execution
- reflection into server internals
- unrestricted function calls

Fail closed for unknown functions.

## Built-in templates

Add initial built-in ConfigEdit component templates for:

- vLLM NVIDIA Docker
- SGLang NVIDIA Docker
- llama.cpp NVIDIA Docker

These templates should cover common fields:

### Shared

- image
- model mount
- service port
- health check
- device binding
- extra env
- extra args
- Docker options

### vLLM examples

- max model length
- GPU memory utilization
- dtype
- tensor parallel size
- pipeline parallel size
- max num seqs
- max num batched tokens
- trust remote code
- served model name

### SGLang examples

- context length
- memory fraction static
- tensor parallel size
- max running requests
- host
- port
- model path

### llama.cpp examples

- ctx size
- GPU layers
- threads
- batch/cache/split-related fields where present in current catalog
- GGUF file model path behavior

Labels/help may be English.

## Validation

Template validation must reject:

- duplicate keys
- missing renderer
- unknown renderer
- missing effect target
- unknown effect type
- invalid editability layer
- invalid view level
- unsafe Docker options
- unsafe mounts
- unsafe expression functions
- unresolved required references
- incompatible backend/vendor applicability
- invalid min/max/default/recommended data

High-risk Docker options must be policy-gated or rejected by default.

## Self-audit gate

Before moving to Work Package C, answer these in `execution-log.md`:

- Are runtime templates and ConfigEdit component templates separate?
- Can templates be loaded from built-in and local override roots?
- Does local override take precedence over built-in?
- Can vLLM/SGLang/llama.cpp component templates be validated?
- Can a supported parameter be added by template/catalog update without Go/Vue code changes, assuming the renderer/effect type exists?
- Does validation fail closed for unsafe or incomplete templates?
- Are backend-specific parameter meanings being added to Go/Vue? If yes, fix before moving on.

## Package verification

Run:

```bash
go test ./internal/server/configedit/... ./internal/server/catalog/... ./internal/server/runplan/...
cd web
npm run test:unit
```

Commit after the package is stable.
