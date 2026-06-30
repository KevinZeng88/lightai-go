# Acceptance Checklist — Unified ConfigEdit Parameter Handling

## Principle

- [ ] All parameters use ConfigEdit Item/Component.
- [ ] No real runtime-affecting parameter exists only in raw JSON.
- [ ] Raw JSON is developer representation only.
- [ ] View level changes presentation, not the underlying parameter mechanism.
- [ ] Templates enhance structured rendering but do not hide unmapped parameters.
- [ ] Page code does not hardcode backend/vendor parameter meaning.

## Generic materialization

- [ ] Explicit template mapping works.
- [ ] Registry metadata mapping works.
- [ ] ConfigSet metadata mapping works.
- [ ] Generic fallback classifier works.
- [ ] Every non-hidden ConfigSet item becomes a ConfigEdit field/component.
- [ ] Unknown non-internal keys become advanced ConfigEdit fields, not raw-only data.
- [ ] Hidden/developer-only fields require explicit policy.

## Runtime page

- [ ] BackendRuntime page does not shrink to only image/mount/env/port/health.
- [ ] Built-in readonly templates show complete fields readonly.
- [ ] Clone-to-edit or downstream-edit guidance appears where appropriate.
- [ ] Raw Config JSON is developer-only.

## Required sections

- [ ] Model Runtime Parameters
- [ ] Backend Parameters / Backend Arguments
- [ ] Runtime Launch
- [ ] Container Options
- [ ] Device Binding
- [ ] Vendor Device Mounts
- [ ] Volume Mounts
- [ ] Environment
- [ ] Model Mount
- [ ] Health Check
- [ ] Service
- [ ] Security / High Risk

## Specific fields

- [ ] `model_runtime.tensor_parallel_size`
- [ ] `model_runtime.pipeline_parallel_size`
- [ ] `model_runtime.max_model_len`
- [ ] `model_runtime.gpu_memory_utilization`
- [ ] `model_runtime.dtype`
- [ ] `model_runtime.trust_remote_code`
- [ ] `backend.extra_args`
- [ ] `launcher.docker_options.shm_size`
- [ ] `launcher.docker_options.ipc_mode`
- [ ] `launcher.docker_options.network_mode`
- [ ] `launcher.docker_options.group_add`
- [ ] `launcher.docker_options.ulimits`
- [ ] `launcher.docker_options.security_options`
- [ ] `launcher.docker_options.privileged`
- [ ] MetaX `/dev/mxcd`
- [ ] MetaX `/dev/dri`
- [ ] MetaX `/dev/mem`
- [ ] MetaX runtime env

## Cross runtime

- [ ] vLLM NVIDIA
- [ ] vLLM MetaX
- [ ] vLLM Huawei if present
- [ ] SGLang NVIDIA
- [ ] SGLang MetaX/Huawei if present
- [ ] llama.cpp NVIDIA
- [ ] llama.cpp CPU if present

## Downstream chain

- [ ] BackendRuntime fields copy to NodeBackendRuntime.
- [ ] NodeBackendRuntime fields copy to Deployment.
- [ ] Deployment can override copied fields.
- [ ] RunPlan compiles copied/overridden fields.
- [ ] `shm_size` compiles to Docker option.
- [ ] Vendor devices compile to Docker device/mount options where applicable.
- [ ] Security options compile or are explicitly blocked by policy with clear reason.

## UI

- [ ] Normal view has operator-friendly labels.
- [ ] Advanced/security views show advanced and high-risk parameters.
- [ ] Developer view shows technical keys/raw JSON/source map.
- [ ] No normal/advanced label shows `技术键: ...`.
- [ ] Warnings appear for high-risk parameters.
- [ ] Readonly reason appears when a field is readonly.

## Tests and verification

- [ ] Go tests cover partial template fallback.
- [ ] Go tests cover all non-hidden ConfigSet item materialization.
- [ ] Go tests cover vLLM MetaX visibility.
- [ ] Go tests cover `shm_size`.
- [ ] Web tests cover BackendRuntime field visibility.
- [ ] Web tests cover raw JSON developer-only behavior.
- [ ] `go test ./...` PASS.
- [ ] `npm test` PASS.
- [ ] `npm run test:unit` PASS.
- [ ] `npm run build` PASS.
