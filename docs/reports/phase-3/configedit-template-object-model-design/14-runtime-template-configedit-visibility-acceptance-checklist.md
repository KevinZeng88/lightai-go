# Acceptance Checklist — Runtime Template ConfigEdit Visibility Regression

## Must pass

- [ ] BackendRuntime / 运行模板 page shows complete effective ConfigEdit fields, not only image/mount/env/port/health.
- [ ] Built-in readonly templates show fields as readonly instead of hiding them.
- [ ] Partial ConfigEdit component templates do not cause unmapped ConfigSet items to disappear.
- [ ] Unmapped but valid ConfigSet items appear in fallback structured sections.
- [ ] Normal/advanced views do not show technical keys as labels.
- [ ] Raw Config JSON/source map/patch target internals are developer view only.

## Field coverage

- [ ] `model_runtime.*` fields show under Model Runtime Parameters.
- [ ] `backend.extra_args` shows under Backend/Advanced Parameters.
- [ ] `runtime.env` / extra env shows under Environment.
- [ ] `runtime.model_mount` shows under Model Mount.
- [ ] `runtime.health` shows under Health Check.
- [ ] `service.*` shows under Service.
- [ ] Docker/container options show under Container Options.
- [ ] High-risk Docker/security options show under Security / High Risk with warning.
- [ ] Vendor device mounts show under Vendor Device Mounts or Device & Volume Mounts.

## Specific fields

- [ ] `shm_size` visible and editable/copyable as Runtime Container Option.
- [ ] `ipc_mode` visible.
- [ ] `network_mode` visible when configured.
- [ ] `group_add` visible when configured.
- [ ] `ulimits` visible when configured.
- [ ] `security_options` visible when configured.
- [ ] `privileged` visible with warning when configured.
- [ ] MetaX `/dev/mxcd` visible when configured.
- [ ] MetaX `/dev/dri` visible when configured.
- [ ] MetaX `/dev/mem` visible with warning when configured.
- [ ] MetaX runtime env visible when configured.

## Cross runtime coverage

- [ ] vLLM NVIDIA
- [ ] vLLM MetaX
- [ ] vLLM Huawei if runtime exists
- [ ] SGLang NVIDIA
- [ ] SGLang MetaX/Huawei if runtime exists
- [ ] llama.cpp NVIDIA
- [ ] llama.cpp CPU if runtime exists

## Downstream chain

- [ ] Fields visible at BackendRuntime copy to NodeBackendRuntime.
- [ ] Fields visible at NodeBackendRuntime copy to Deployment.
- [ ] Deployment can override copied fields.
- [ ] RunPlan compiles copied/overridden fields into DockerSpec.
- [ ] `--shm-size` compiles from ConfigEdit field/effect.
- [ ] MetaX devices/security/env compile from ConfigEdit field/effect or documented policy.

## Tests

- [ ] Go tests added for projection visibility.
- [ ] Go tests added for partial template fallback.
- [ ] Go tests added for MetaX runtime fields.
- [ ] Go tests added for `shm_size` effect.
- [ ] Web tests added for BackendRuntime page field visibility.
- [ ] Web tests added for raw JSON developer-only behavior.
- [ ] Full verification commands pass.
