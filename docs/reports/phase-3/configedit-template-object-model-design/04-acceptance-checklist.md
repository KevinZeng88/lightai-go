# Acceptance Checklist for Future Implementation

Use this checklist after Codex produces an implementation plan and before accepting final code changes.

## 1. Architecture

- [ ] ConfigEdit is modeled as an object, not only a flat field list.
- [ ] ConfigEdit object contains parent snapshot information.
- [ ] ConfigEdit object can describe child initialization/copy behavior.
- [ ] Child layers copy whole effective snapshots at creation time.
- [ ] Child snapshots are independently editable after copy.
- [ ] Parent changes do not silently overwrite child snapshots.
- [ ] Fields record source/provenance and override state.
- [ ] Fields support reset to parent/default where applicable.
- [ ] Pages only render ConfigEdit and submit patches; pages do not interpret backend/vendor/runtime params.

## 2. Template-driven components

- [ ] ConfigEdit component templates can be externalized.
- [ ] Template defines sections/components/renderers.
- [ ] Template defines label/help/default/recommended/range/validation.
- [ ] Template defines editability by layer.
- [ ] Template defines copy behavior to child ConfigEdit.
- [ ] Template defines normal/advanced/developer view grouping.
- [ ] Template defines effects on CLI/env/Docker/mount/port/health check.
- [ ] Runtime/backend template and ConfigEdit component template are clearly separated.
- [ ] New backend parameters can be added by template/catalog update without binary changes.

## 3. Device binding

- [ ] Device binding is a ConfigEdit component, not hidden RunPlan injection.
- [ ] Device binding supports enabled/disabled.
- [ ] Device binding supports auto/manual/inherited/disabled modes or a clearly documented subset.
- [ ] Device binding exposes vendor-neutral fields.
- [ ] NVIDIA `--gpus` and `CUDA_VISIBLE_DEVICES` are editable via ConfigEdit fields.
- [ ] Final Docker effects are compiled from current ConfigEdit snapshot.
- [ ] Disabling device binding removes GPU Docker option and visible devices env.
- [ ] Manual GPU selection changes final Docker option and env.
- [ ] MetaX/Huawei/CPU paths are not blocked by NVIDIA hardcoding in pages.

## 4. Final Docker command coverage

Every final runtime-affecting parameter must map to a ConfigEdit field/component/effect:

- [ ] image
- [ ] entrypoint/command/args
- [ ] extra args
- [ ] env/extra env
- [ ] model mount
- [ ] extra volumes
- [ ] ports
- [ ] host/container port
- [ ] IPC mode
- [ ] shm size
- [ ] device binding
- [ ] health check
- [ ] served model name
- [ ] model path/container path
- [ ] vLLM runtime params
- [ ] SGLang runtime params
- [ ] llama.cpp runtime params

## 5. UI/UX

- [ ] Normal UI does not show technical keys such as `model_runtime.pipeline_parallel_size`.
- [ ] Technical keys are available only in developer/debug mode.
- [ ] Raw command templates are not shown in normal UI.
- [ ] Config JSON / dry-run detail / raw source maps are developer/debug only.
- [ ] Normal UI does not show misleading empty raw template arrays that conflict with final effective values.
- [ ] Fields show useful help: default, recommended, range, effect, applicability.
- [ ] English technical help is acceptable.
- [ ] Advanced fields are grouped/collapsed appropriately.

## 6. RunPlan compiler

- [ ] RunPlan compiles from final materialized ConfigEdit snapshot.
- [ ] RunPlan does not silently inject runtime-affecting parameters outside ConfigEdit/effects.
- [ ] Source map explains final values but is not the source of truth.
- [ ] Diagnostics distinguish warning and blocking error.
- [ ] `can_run` reflects final resolved plan plus blocking issues only.
- [ ] Docker command preview is generated from final DockerSpec.

## 7. Cross-backend and cross-page coverage

- [ ] vLLM fixture covered.
- [ ] SGLang fixture covered.
- [ ] llama.cpp fixture covered.
- [ ] BackendRuntime page covered.
- [ ] NodeBackendRuntime page covered.
- [ ] ModelDeployment wizard covered.
- [ ] ModelDeployment edit page covered.
- [ ] Deployment override covered.
- [ ] RunPlan preview covered.

## 8. Template management page

- [ ] Built-in templates are listed and read-only.
- [ ] Built-in templates can be cloned.
- [ ] User-defined templates can be edited.
- [ ] Structured template editor exists or has a concrete implementation plan.
- [ ] Raw YAML/JSON editor is advanced mode only.
- [ ] Template validation exists.
- [ ] ConfigEdit preview exists.
- [ ] RunPlan/Docker command preview exists.
- [ ] Publish/disable/rollback/import/export plan exists.

## 9. Test gates

Future implementation must pass:

```bash
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Also require targeted tests for:

- ConfigEdit parent/child copy snapshot.
- Device binding editable propagation.
- Disable/manual GPU binding behavior.
- Final Docker parameter → ConfigEdit field/effect coverage.
- Normal/advanced/developer visibility.
- External template validation.
- vLLM/SGLang/llama.cpp fixtures.
