# Active E2E Scripts — Current Contract

Generated: 2026-06-25
Contract version: v1 (node_backend_runtime_id, /check-request, parameter_values_json)

## Active Scripts (Current Contract)

| Script | Purpose | Requires HW |
|--------|---------|-------------|
| scripts/e2e-real-smoke-all-three.sh | Full vLLM/SGLang/llama.cpp smoke | NVIDIA GPU + Docker |
| scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh | llama.cpp real smoke | NVIDIA GPU + Docker |
| scripts/e2e-model-runtime-wizard-nvidia-vllm.sh | vLLM real smoke | NVIDIA GPU + Docker |
| scripts/e2e-backend-runtime-nvidia-api.sh | vLLM API lifecycle | NVIDIA GPU |
| scripts/e2e-deployment-visibility-selected.sh | Deployment CRUD API | None (API-only) |
| scripts/e2e-model-runtime-param-trace.sh | Parameter trace dry-run | None (API-only) |
| scripts/e2e-model-runtime-wizard-nvidia-sglang.sh | SGLang wizard | NVIDIA GPU + Docker |
| scripts/e2e-instance-stop-real-llamacpp.sh | Instance stop test | NVIDIA GPU |
| scripts/e2e-model-runtime-failed-instance-logs.sh | Failed instance logs | NVIDIA GPU |
| scripts/e2e-clone-template-parameter-persistence.sh | Clone template test | None (API-only) |
| scripts/e2e-runtime-config-web-check-flow.sh | Web check flow | None (API proxy) |
| scripts/e2e-packaged-smoke.sh | Package smoke | None |
| scripts/lightai-bootstrap.sh | Bootstrap tool | None (API-only) |

## Stale Scripts (Archived)

These scripts use legacy contracts (backend_runtime_id, parameters_json) and are archived at:
scripts/archive/legacy-contract/

- scripts/e2e-matrix-verifier.sh
- scripts/e2e-dryrun-parameter-matrix-enhanced.sh
- scripts/e2e-model-runtime-wizard-nvidia-api.sh
- scripts/e2e-model-runtime-api.sh
- scripts/e2e-model-runtime-local.sh
- scripts/e2e-runplan-parameter-source-audit.sh
