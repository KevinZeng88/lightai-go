# Inventory — Active Scripts

## Current Contract Scripts (use  /  / )

- **UNCLASSIFIED**: scripts/.gitkeep
- **ACTIVE-CURRENT**: scripts/apply-patch.sh
- **UNCLASSIFIED**: scripts/check-glibc-compat.sh
- **ACTIVE-CURRENT**: scripts/collect-debug-bundle.sh
- **ACTIVE-CURRENT**: scripts/collect-logs.sh
- **ACTIVE-CURRENT**: scripts/diagnose-model-runtime-spec.sh
- **ACTIVE-CURRENT**: scripts/e2e-backend-runtime-nvidia-api.sh
- **ACTIVE-CURRENT**: scripts/e2e-clone-template-parameter-persistence.sh
- **ACTIVE-CURRENT**: scripts/e2e-deployment-visibility-selected.sh
- **ACTIVE-NEEDS-REPAIR**: scripts/e2e-dryrun-parameter-matrix-enhanced.sh (may use legacy payload)
- **UNCLASSIFIED**: scripts/e2e-inference-parser-llamacpp.sh
- **ACTIVE-CURRENT**: scripts/e2e-instance-stop-real-llamacpp.sh
- **ACTIVE-NEEDS-REPAIR**: scripts/e2e-matrix-verifier.sh (may use legacy payload)
- **ACTIVE-NEEDS-REPAIR**: scripts/e2e-model-runtime-api.sh (may use legacy payload)
- **UNCLASSIFIED**: scripts/e2e-model-runtime-failed-instance-logs.sh
- **ACTIVE-NEEDS-REPAIR**: scripts/e2e-model-runtime-local.sh (may use legacy payload)
- **ACTIVE-CURRENT**: scripts/e2e-model-runtime-param-trace.sh
- **ACTIVE-CURRENT**: scripts/e2e-model-runtime-wizard-nvidia-api.sh
- **ACTIVE-CURRENT**: scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
- **ACTIVE-CURRENT**: scripts/e2e-model-runtime-wizard-nvidia-matrix.sh
- **ACTIVE-CURRENT**: scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
- **ACTIVE-CURRENT**: scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
- **ACTIVE-CURRENT**: scripts/e2e-packaged-smoke.sh
- **ACTIVE-CURRENT**: scripts/e2e-real-smoke-all-three.sh
- **ACTIVE-NEEDS-REPAIR**: scripts/e2e-runplan-parameter-source-audit.sh (may use legacy payload)
- **ACTIVE-CURRENT**: scripts/e2e-runtime-config-copy-first-save-selection.sh
- **ACTIVE-CURRENT**: scripts/e2e-runtime-config-web-check-flow.sh
- **ACTIVE-CURRENT**: scripts/e2e-ui-persistence-runplan-selected.sh
- **UNCLASSIFIED**: scripts/e2e/lib/api-client.sh
- **UNCLASSIFIED**: scripts/e2e/lib/assert.sh
- **UNCLASSIFIED**: scripts/e2e/lib/cleanup.sh
- **UNCLASSIFIED**: scripts/e2e/lib/docker.sh
- **UNCLASSIFIED**: scripts/e2e/lib/e2e-assert-selftest.sh
- **UNCLASSIFIED**: scripts/e2e/lib/e2e-assert.sh
- **UNCLASSIFIED**: scripts/e2e/lib/env.sh
- **UNCLASSIFIED**: scripts/e2e/lib/model-runtime-common.sh
- **UNCLASSIFIED**: scripts/e2e/lib/report.sh
- **UNCLASSIFIED**: scripts/e2e/lib/resources.sh
- **UNCLASSIFIED**: scripts/lib/__pycache__/bootstrap-export.cpython-312.pyc
- **ACTIVE-CURRENT**: scripts/lib/bootstrap-export.py
- **ACTIVE-CURRENT**: scripts/lightai-bootstrap.sh
- **UNCLASSIFIED**: scripts/observability-down.sh
- **ACTIVE-CURRENT**: scripts/observability-status.sh
- **UNCLASSIFIED**: scripts/observability-up.sh
- **UNCLASSIFIED**: scripts/package-patch.sh
- **ACTIVE-CURRENT**: scripts/package-release-docker.sh
- **ACTIVE-CURRENT**: scripts/package-release.sh
- **UNCLASSIFIED**: scripts/prepare-observability-binaries.sh
- **ACTIVE-CURRENT**: scripts/reset-agent-identity.sh
- **ACTIVE-CURRENT**: scripts/reset-grafana-password.sh
- **ACTIVE-CURRENT**: scripts/reset-password.sh
- **ACTIVE-CURRENT**: scripts/smoke-model-backends.sh
- **ACTIVE-CURRENT**: scripts/start-agent.sh
- **ACTIVE-CURRENT**: scripts/start-all.sh
- **ACTIVE-CURRENT**: scripts/start-observability.sh
- **ACTIVE-CURRENT**: scripts/start-server.sh
- **ACTIVE-CURRENT**: scripts/status.sh
- **ACTIVE-CURRENT**: scripts/stop-agent.sh
- **ACTIVE-CURRENT**: scripts/stop-all.sh
- **ACTIVE-CURRENT**: scripts/stop-observability.sh
- **ACTIVE-CURRENT**: scripts/stop-server.sh
- **ACTIVE-CURRENT**: scripts/verify-local.sh
scripts/e2e-ui-persistence-runplan-selected.sh:104:nbr_json="$(api_ok POST "/api/v1/nodes/$node_id/backend-runtimes/enable" "{\"backend_runtime_id\":\"$runtime_id\",\"display_name\":\"UI Node Runtime $run_id\",\"image_ref\":\"$runtime_image\",\"image_present\":true,\"docker_available\":true}")"
scripts/e2e-ui-persistence-runplan-selected.sh:107:api_ok POST "/api/v1/nodes/$node_id/backend-runtimes/check" "{\"backend_runtime_id\":\"$runtime_id\",\"image_ref\":\"$runtime_image\",\"image_present\":true,\"docker_available\":true}" > /dev/null
scripts/e2e-ui-persistence-runplan-selected.sh:109:deployment_payload="{\"name\":\"ui-persist-deploy-$run_id\",\"display_name\":\"UI Persist Deploy $run_id\",\"model_artifact_id\":\"$artifact_id\",\"node_backend_runtime_id\":\"$node_id:$runtime_id\",\"placement_json\":{\"node_id\":\"$node_id\",\"accelerator_ids\":[]},\"service_json\":{\"host_port\":8005,\"container_port\":8080,\"app_port\":8080,\"health_port\":8005,\"api_test_port\":8005},\"parameters_json\":{\"served_model_name\":\"ui-persist-$run_id\"},\"env_overrides_json\":{}}"
scripts/e2e/lib/model-runtime-common.sh:169:    "{\"backend_runtime_id\":\"$BACKEND_RUNTIME_ID\",\"image_ref\":\"$IMAGE_REF\",\"image_present\":$ip,\"docker_available\":true}")"
scripts/e2e/lib/model-runtime-common.sh:178:  local r; r="$(api_ok POST "/api/v1/nodes/$NODE_ID/backend-runtimes/$nbr_id/check-request" '{}')"
scripts/e2e/lib/model-runtime-common.sh:200:  local payload; payload="{\"name\":\"$name\",\"model_artifact_id\":\"$ARTIFACT_ID\",\"node_backend_runtime_id\":\"$NODE_ID:$BACKEND_RUNTIME_ID\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[\"$GPU_ID\"]},\"service_json\":{\"host_port\":$HOST_PORT}"
scripts/e2e/lib/model-runtime-common.sh:202:    payload="$payload,\"parameters_json\":{$DEPLOY_PARAMS}"
scripts/e2e/lib/model-runtime-common.sh:218:    "{\"model_artifact_id\":\"$ARTIFACT_ID\",\"node_backend_runtime_id\":\"$NODE_ID:$BACKEND_RUNTIME_ID\",\"host_port\":$HOST_PORT}")"
scripts/e2e-runtime-config-copy-first-save-selection.sh:123:    if n.get('backend_runtime_id') == '$clone_id':
scripts/e2e-runtime-config-copy-first-save-selection.sh:131:  local enable_resp; enable_resp=$(api_post "nodes/$NODE_ID/backend-runtimes/enable" "{\"backend_runtime_id\":\"$clone_id\"}")
scripts/e2e-runtime-config-copy-first-save-selection.sh:139:  local check_resp; check_resp=$(api_post "nodes/$NODE_ID/backend-runtimes/check" "{\"backend_runtime_id\":\"$clone_id\",\"image_present\":true,\"docker_available\":true}")
scripts/e2e-runtime-config-copy-first-save-selection.sh:148:    local dep_payload="{\"name\":\"$PREFIX-${label}-dep\",\"model_artifact_id\":\"$art_id\",\"node_backend_runtime_id\":\"$NODE_ID:$clone_id\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[]},\"service_json\":{\"host_port\":8501,\"container_port\":8000,\"app_port\":8000},\"parameters_json\":{}}"
scripts/e2e-real-smoke-all-three.sh:102:  api_post "nodes/$NODE_ID/backend-runtimes/enable" "{\"backend_runtime_id\":\"$rt\",\"image_ref\":\"vllm/vllm-openai:latest\",\"image_present\":true,\"docker_available\":true}" >/dev/null || true
scripts/e2e-real-smoke-all-three.sh:103:  api_post "nodes/$NODE_ID/backend-runtimes/$NODE_ID:$rt/check-request" '{}' >/dev/null || true
scripts/e2e-real-smoke-all-three.sh:104:  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-vllm\",\"display_name\":\"vLLM Smoke\",\"model_artifact_id\":\"$HF_ART\",\"node_backend_runtime_id\":\"$NODE_ID:$rt\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[]},\"service_json\":{\"host_port\":8191,\"container_port\":8022,\"app_port\":8022},\"parameters_json\":{\"served_model_name\":\"e2e-vllm-smoke\",\"gpu_memory_utilization\":0.85,\"max_model_len\":4096,\"tensor_parallel_size\":1}}")
scripts/e2e-real-smoke-all-three.sh:154:  api_post "nodes/$NODE_ID/backend-runtimes/enable" "{\"backend_runtime_id\":\"$rt\",\"image_ref\":\"lmsysorg/sglang:latest\",\"image_present\":true,\"docker_available\":true}" >/dev/null || true
scripts/e2e-real-smoke-all-three.sh:155:  api_post "nodes/$NODE_ID/backend-runtimes/$NODE_ID:$rt/check-request" '{}' >/dev/null || true
scripts/e2e-real-smoke-all-three.sh:156:  local dep_resp; dep_resp=$(api_post "deployments" "{\"name\":\"$PREFIX-sglang\",\"display_name\":\"SGLang Smoke\",\"model_artifact_id\":\"$HF_ART\",\"node_backend_runtime_id\":\"$NODE_ID:$rt\",\"placement_json\":{\"node_id\":\"$NODE_ID\",\"accelerator_ids\":[]},\"service_json\":{\"host_port\":8194,\"container_port\":31000,\"app_port\":31000},\"parameters_json\":{\"served_model_name\":\"e2e-sglang-smoke\",\"tp\":1}}")
scripts/e2e-real-smoke-all-three.sh:202:  api_post "nodes/$NODE_ID/backend-runtimes/enable" "{\"backend_runtime_id\":\"$rt\",\"image_ref\":\"ghcr.io/ggml-org/llama.cpp:server-cuda13\",\"image_present\":true,\"docker_available\":true}" >/dev/null || true
scripts/e2e-real-smoke-all-three.sh:203:  api_post "nodes/$NODE_ID/backend-runtimes/$NODE_ID:$rt/check-request" '{}' >/dev/null || true
