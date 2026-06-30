# 09 — Work Package C: Runtime Effects Components + RunPlan Compiler Cleanup

## Objective

Materialize all final runtime-affecting parameters as ConfigEdit components and compile RunPlan/DockerSpec from the final materialized ConfigEdit snapshot.

Do not treat GPU binding as the only problem. Implement a component family for runtime effects.

## Required components

### 1. `runtime.device_binding`

Fields:

- enabled
- mode: auto / manual / disabled / inherited
- vendor: nvidia / metax / huawei / cpu / none
- accelerator_ids
- accelerator_count
- visible_env_key
- visible_env_value
- docker_gpu_option
- device_mounts

Rules:

- Auto selection may generate initial values, but after materialization those values become editable current-layer configuration.
- Manual mode uses selected accelerator IDs.
- Disabled mode removes GPU Docker option, visible-device env, and device mounts from DockerSpec.
- Source/provenance explains initial origin only.
- Source/provenance does not imply readonly.
- NVIDIA default effects must be template-defined:
  - Docker option: `--gpus "device=..."`
  - env: `CUDA_VISIBLE_DEVICES=...`
- MetaX/Huawei extension points must be vendor-neutral and template-driven.
- Pages must not hardcode NVIDIA/CUDA logic.

### 2. `service.port_binding`

Fields:

- listen_host
- container_port
- host_port
- protocol
- served_model_name when applicable

Effects:

- Docker port mapping
- backend CLI host/port args when defined by template
- service metadata

### 3. `runtime.model_mount`

Fields:

- host_path
- container_path
- readonly
- source model location
- editability policy

Rules:

- Platform may own safe host path resolution.
- Container path and readonly policy should be represented in ConfigEdit where allowed.
- Final `-v` token must map to the component/effect.

### 4. `runtime.health_check`

Fields:

- path
- expected_status
- startup_timeout_seconds
- interval_seconds
- timeout_seconds

Rules:

- HTTP default should be explicit, preferably 200 when path exists.
- If 0 means disabled/no-check, the template/UI must say so.
- 0 default values must not become opaque resolver errors.

### 5. `runtime.env` and `runtime.extra_env`

Use generic env component renderers.

Rules:

- Visible devices env belongs to device binding or env effect, not resolver-only injection.
- Sensitive values must be redacted in UI/source maps where appropriate.

### 6. `backend.args` and `backend.extra_args`

Use generic args component renderers.

Rules:

- Backend-specific CLI flags should come from template effects.
- New backend parameters should not require Go semantic adapter edits if the effect type already exists.

### 7. `launcher.docker_options`

Fields may include:

- ipc
- shm_size
- network mode if supported
- security options
- privileged
- device options
- ulimits
- group_add

Rules:

- High-risk options require validation and warning.
- Normal UI should show safe common options.
- Developer view can show raw advanced options.

## RunPlan compiler cleanup

Refactor RunPlan generation:

- Input is the final materialized ConfigEdit snapshot.
- Generic compiler evaluates component effects into DockerSpec.
- Source map explains effect origin; it is not source of truth.
- EquivalentCommandPreview only renders DockerSpec.
- EquivalentCommandPreview must not create semantics absent from DockerSpec/effects.
- Resolver must not hide-inject runtime-affecting behavior.
- Defensive defaults are allowed only with diagnostics and must not be the ordinary path.

## Hidden injection cleanup targets

Clean up or migrate these to ConfigEdit effects:

- `--gpus`
- visible-device env
- service port
- listen host
- served model name
- health check
- model mount
- backend CLI semantic mapping
- Docker options
- extra env
- extra args
- process start profiles if user-visible/runtime-specific

## Platform-generated readonly exceptions

These may remain platform-generated if clearly marked readonly/platform_generated:

- container name
- instance id
- lease id
- operation id
- hardware inventory evidence
- resolved safe host path when derived from selected model location and path safety policy

## Cross-backend requirements

For each of vLLM, SGLang, and llama.cpp:

- final Docker command preview is non-empty
- image maps to ConfigEdit/effect/source
- model path/mount maps to ConfigEdit/effect/source
- port maps to ConfigEdit/effect/source
- health check maps to ConfigEdit/effect/source
- backend CLI args map to ConfigEdit/effect/source
- extra args/env map to ConfigEdit/effect/source
- GPU/device binding maps to ConfigEdit/effect/source
- known fields do not display as generic unnamed configuration
- disabling device binding removes GPU injection
- manual GPU selection changes command/env

## Self-audit gate

Before moving to Work Package D, answer these in `execution-log.md`:

- Can final `--gpus` appear without `runtime.device_binding` effect? If yes, fix.
- Can final visible devices env appear without `runtime.device_binding` or env component effect? If yes, fix.
- Can final `-p`, `-v`, health check, backend CLI args, Docker options appear without ConfigEdit component/effect mapping? If yes, fix or document as a true blocker.
- Are vLLM, SGLang, and llama.cpp all covered?
- Does disabled device binding remove GPU injection?
- Does manual GPU selection change Docker command?
- Are hidden runtime-affecting `system_generated` entries gone except approved platform readonly fields?
- Does source map explain effects without being source of truth?
- Does EquivalentCommandPreview only render DockerSpec?

## Package verification

Run:

```bash
go test ./internal/server/runplan/... ./internal/server/configedit/... ./internal/server/api/...
go test ./...
cd web
npm run test:unit
```

Commit after the package is stable.
