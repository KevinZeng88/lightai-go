# 11 — Self-Audit, Evidence, and Acceptance Guide

Use this guide after each work package and before final closeout.

Update:

```text
docs/reports/phase-3/configedit-template-object-model-design/execution-log.md
```

## Execution log format

For each package, record:

```markdown
## Work Package <A/B/C/D> — <title>

### Scope completed

### Files changed

### Tests added or updated

### Tests run

### Failures found

### Fixes applied

### Self-audit answers

### Remaining limitations

### Commit id
```

## Required self-audit checks

### Architecture checks

- ConfigEdit is an object model, not only a field projection.
- ConfigEdit object has parent snapshot metadata.
- Child layers copy whole effective snapshots.
- Deployment ConfigEdit includes service, placement/device binding, health, mount, args, env, Docker options.
- Source/provenance does not imply readonly.
- Reset to parent/default exists at least at backend/API level.
- Runtime template and ConfigEdit component template are separate.

### Hidden injection checks

Search for hidden runtime-affecting injection in:

- runplan resolver
- semantic adapter
- Docker command preview
- deployment preview handler
- deployment create/start path
- frontend preview panels

Runtime-affecting fields must come from final ConfigEdit component/effect:

- `--gpus`
- visible-device env
- ports
- mounts
- health check
- CLI args
- Docker options
- extra env
- extra args

Allowed platform-generated readonly fields:

- container name
- instance id
- operation id
- lease id
- hardware inventory evidence
- safe resolved host path where platform-owned

### Page bypass checks

Search UI for page-specific parameter interpretation:

- vLLM
- SGLang
- llama.cpp
- NVIDIA
- CUDA_VISIBLE_DEVICES
- `--gpus`
- `model_runtime.`
- raw source map normal view
- raw Config JSON normal view
- patch target normal view

If these appear, decide whether they are developer view, test fixture, or actual user-facing bypass. Fix actual bypass.

### Externalization checks

- Backend/version parameter display/help/range/default/recommended/effects come from template/catalog metadata.
- ConfigEdit component template can be changed without rebuilding binary for supported renderer/effect types.
- Local override template wins over built-in.
- Unsafe templates fail validation.
- Unknown renderer/effect types fail validation.

### UI checks

Normal view:

- clean operator view
- no technical key
- no raw JSON
- no raw source map
- no unresolved template
- no patch target internals

Advanced view:

- advanced safe controls
- extra env/args
- advanced Docker options

Developer view:

- raw/debug data available when needed

### Cross-backend checks

For each:

- vLLM NVIDIA Docker
- SGLang NVIDIA Docker
- llama.cpp NVIDIA Docker

Verify:

- ConfigEdit object opens
- Deployment ConfigEdit copies from NBR
- RunPlan preview generates Docker command
- command tokens map to ConfigEdit component/effect
- device binding auto/manual/disabled covered
- service port covered
- model mount covered
- health check covered
- args/env/Docker options covered
- normal/advanced/developer view covered

## Evidence matrix

Final closeout must include evidence for:

| Area | Required evidence |
| --- | --- |
| ConfigEdit object API | example JSON or test snapshot |
| Parent/child snapshot | test name + result |
| Reset behavior | test name + API behavior |
| Template loading | built-in/local precedence test |
| Template validation | failure cases |
| Device binding | auto/manual/disabled tests |
| RunPlan compiler | Docker token mapping test |
| vLLM | fixture/test result |
| SGLang | fixture/test result |
| llama.cpp | fixture/test result |
| UI normal view | test/screenshot/DOM assertion |
| UI developer view | test/screenshot/DOM assertion |
| Template management MVP | route/component/tests/API evidence |
| Full verification | command results |

## Final acceptance

The run is acceptable only if:

- full verification passes
- final closeout exists
- final closeout lists commit ids and push result
- final git status is clean
- all remaining issues are documented blockers or explicit MVP limitations
- no fixable issue remains unaddressed
