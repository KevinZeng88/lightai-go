# 10 — Work Package D: UI Full-Chain Integration + Template Management MVP + Final Audit

## Objective

Expose the new ConfigEdit object/template model through the UI and add the ConfigEdit Template Management MVP. Remove page-level parameter interpretation.

## UI integration scope

Update these paths to consume ConfigEdit objects/components as the primary contract:

- BackendRuntime / runtime template editing
- NodeBackendRuntime / node runtime config
- Deployment create/edit
- Deployment override
- RunPlan preview
- Model deployment detail
- ConfigEditView / ConfigField

Pages should orchestrate flow. Pages should not understand backend parameter semantics.

## View levels

Implement or enforce:

- normal
- advanced
- developer

### Normal view must hide

- technical keys
- raw source map
- raw Config JSON
- unresolved template command
- patch target internals
- system_generated internals
- raw DockerSpec internals
- raw dry-run detail

### Advanced view may show

- advanced runtime parameters
- extra env
- extra args
- advanced Docker options
- advanced health/mount options

### Developer view may show

- raw Config JSON
- source map
- technical keys
- unresolved templates
- dry-run detail
- DockerSpec internals
- patch targets

## Required user-editable components

The UI must expose these via ConfigEdit components:

### Device Binding

- enabled
- mode
- vendor
- accelerator IDs
- visible env key/value
- Docker GPU option
- device mounts where applicable
- preview of Docker/env effects

### Service Port

- listen host
- container port
- host port
- protocol
- served model name where applicable

### Health Check

- path
- expected status
- startup timeout
- interval
- timeout

### Model Mount

- host path if editable or readonly with reason
- container path
- readonly flag
- source model location

### Runtime Env / Extra Env

- key/value editor
- sensitive field behavior

### Backend Args / Extra Args

- structured args where defined
- generic extra args editor in advanced view

### Docker Options

- ipc
- shm size
- safe common options
- advanced/high-risk options with warnings and policy validation

## Help and tooltip content

For technical runtime parameters, English is acceptable.

Display useful help:

- default
- recommended
- valid range
- examples
- effect
- applicability
- source
- edit scope
- reset behavior

Avoid showing technical keys in normal view.

## RunPlan preview UI

RunPlan preview should show:

- can run
- warnings vs blocking errors
- Docker command
- summarized source/effect explanation
- device binding summary
- service/mount/health summary

Normal view should not show raw source map.

Developer view can show raw source map and DockerSpec.

## ConfigEdit Template Management MVP

Add a page separate from runtime template pages.

Minimum capabilities:

1. List templates

- built-in
- local override
- draft/user editable if implemented
- backend/vendor applicability
- status
- source

2. View template

- metadata
- sections
- components
- fields
- effects
- validation status

3. Clone built-in

- clone built-in template to local/user draft
- cloned template becomes editable

4. Edit template

MVP can be simple:

- structured editor for common metadata/sections/components/fields/effects where feasible
- raw YAML/JSON advanced editor
- avoid complex drag-and-drop in MVP

5. Validate/lint

- run server-side validation
- show errors with path/reason/severity
- reject unsafe expressions/effects

6. Preview

- preview ConfigEdit object for selected backend/version/vendor/layer
- preview RunPlan/Docker command for selected model/node fixture or selected live objects

7. Save draft

- save local/user editable template
- export/import template file

Optional if time permits:

- publish/disable/rollback
- diff against built-in
- version history

If optional items are not implemented, record them as explicit MVP limitations, not untracked future work.

## Security and RBAC

Template editing is high risk.

Minimum:

- admin-only access
- fail-closed validation for unknown effect types
- reject or policy-gate privileged Docker options
- reject or policy-gate host path mounts
- reject or policy-gate device mounts
- reject unsafe expressions
- audit template save/validate actions if audit framework exists

## Page-level hardcoding cleanup

Search and remove/avoid page-level logic for:

- vLLM parameter names
- SGLang parameter names
- llama.cpp parameter names
- NVIDIA
- CUDA_VISIBLE_DEVICES
- `--gpus`
- Docker option label dictionaries
- CLI flag help dictionaries
- raw source map normal-view tables

When a field needs a label/help/effect, it should come from ConfigEdit component template metadata.

## Self-audit gate

Before final closeout, answer these in `execution-log.md`:

- Do any pages still hardcode vLLM/SGLang/llama.cpp/NVIDIA/CUDA parameter knowledge? If yes, fix.
- Does normal UI still show technical keys/raw JSON/source map? If yes, fix.
- Can the user edit Device Binding from deployment create/edit?
- Can the user edit service port, health check, model mount, env, args, and Docker options through ConfigEdit components?
- Can operators validate and preview a ConfigEdit template without rebuilding binaries?
- Are vLLM, SGLang, and llama.cpp validated end-to-end?
- Do all tests pass?
- Are remaining limitations recorded in closeout?

## Final verification

Run the full suite:

```bash
git status --short
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Fix failures and rerun.

Commit and push after this package and final verification.
