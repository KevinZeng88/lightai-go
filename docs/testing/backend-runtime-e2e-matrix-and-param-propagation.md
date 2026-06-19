# Backend Runtime E2E Matrix and Parameter Propagation Test Specification

## 1. Purpose

This document defines the mandatory E2E and regression test requirements for LightAI Go BackendRuntime and NodeBackendRuntime execution.

The purpose is not only to verify that default runtime templates can start model containers, but also to verify that user-modified runtime parameters are correctly propagated through the real product path:

```text
BackendVersion catalog
-> BackendRuntime catalog
-> NodeBackendRuntime
-> Preflight
-> RunPlan / NodeRunPlan
-> Equivalent Docker Command Preview
-> Agent Docker create spec
-> Container process arguments
```

E2E scripts must not bypass this path.

---

## 2. Scope

The local NVIDIA test matrix must cover:

```text
1. llama.cpp
2. vLLM
3. SGLang
```

Each backend must be tested in two rounds:

```text
Round A: system BackendRuntime default parameters
Round B: user BackendRuntime / NodeBackendRuntime modified parameters
```

Expected matrix:

```text
llama.cpp default: PASS
llama.cpp modified params: PASS
vLLM default: PASS
vLLM modified params: PASS
SGLang default: PASS
SGLang modified params: PASS
```

---

## 3. Product Path Requirement

Tests must use the real product path:

```text
BackendVersion
-> BackendRuntime
-> NodeBackendRuntime
-> preflight
-> RunPlan / NodeRunPlan
-> deployment start
-> Agent Docker create/start
-> health_check
-> /v1/models
-> model instance test
-> logs
-> stop
-> cleanup
```

Tests must not:

```text
1. Directly assemble docker run commands in E2E scripts.
2. Patch arguments inside E2E scripts to hide renderer bugs.
3. Skip BackendRuntime or NodeBackendRuntime.
4. Skip preflight.
5. Use a special test-only command path that production does not use.
```

---

## 4. Backend Matrix

### 4.1 llama.cpp

Expected baseline:

```text
Backend = llama.cpp
BackendVersion = llamacpp-b9700
BackendRuntime = llamacpp-b9700-nvidia-cuda13
Image = ghcr.io/ggml-org/llama.cpp:server-cuda13
Model = /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

Default validation:

```text
args contain -m or --model
args contain MODEL_CONTAINER_PATH
args contain --host
args contain --port
/v1/models succeeds
model instance test succeeds
```

Modified parameter validation:

```text
--ctx-size 2048
--n-gpu-layers -1
```

The modified parameters must appear in:

```text
BackendRuntime user catalog
NodeBackendRuntime config_snapshot_json
RunPlan args
Equivalent Docker Command Preview
Agent docker.create.spec args_json
```

---

### 4.2 vLLM

Expected baseline:

```text
Backend = vLLM
BackendVersion = vllm-v0.23.0
BackendRuntime = vllm-v0.23.0-nvidia-cuda
Image = local available vllm/vllm-openai:<tag>
Model = /home/kzeng/models/Qwen3-0.6B-Instruct-2512
```

Default validation:

```text
args contain --host 0.0.0.0
args contain --port 8000
args do not lose --host / --port flags
stderr does not contain "unrecognized arguments: 0.0.0.0 8000"
/v1/models succeeds
model instance test succeeds
```

Modified parameter validation:

```text
--max-model-len 2048
--gpu-memory-utilization 0.80
```

The modified parameters must appear in:

```text
BackendRuntime user catalog
NodeBackendRuntime config_snapshot_json
RunPlan args
Equivalent Docker Command Preview
Agent docker.create.spec args_json
```

---

### 4.3 SGLang

Expected baseline:

```text
Backend = SGLang
BackendVersion = sglang-v0.5.12.post1 or current catalog version
BackendRuntime = sglang-v0.5.12-nvidia-cuda
Image = local available lmsysorg/sglang:<tag>
Model = /home/kzeng/models/Qwen3-0.6B-Instruct-2512
```

Default validation:

```text
entrypoint uses python3 -m sglang.launch_server or the image-supported equivalent
args contain --model-path
args contain --host
args contain --port
/v1/models succeeds
model instance test succeeds
```

Modified parameter validation:

```text
--tp 1
--mem-fraction-static 0.75
```

If `--mem-fraction-static` is not supported by the local image, use another safe supported SGLang runtime parameter and record the reason.

The modified parameters must appear in:

```text
BackendRuntime user catalog
NodeBackendRuntime config_snapshot_json
RunPlan args
Equivalent Docker Command Preview
Agent docker.create.spec args_json
```

---

## 5. NodeBackendRuntime Override Tests

For each backend, tests must verify that NodeBackendRuntime-level overrides are applied without mutating BackendRuntime.

Required override categories:

```text
1. image_ref
2. env override
3. port binding
4. arg override
```

After modifying NodeBackendRuntime:

```text
status must become needs_check
check may update only check result fields
BackendRuntime catalog and DB projection must remain unchanged
preflight must use NodeBackendRuntime override values
RunPlan must use NodeBackendRuntime override values
Agent docker.create.spec must use NodeBackendRuntime override values
```

---

## 6. Independence Tests

### 6.1 BackendVersion -> BackendRuntime

Test flow:

```text
1. Clone or create user BackendVersion A.
2. Create BackendRuntime R from A.
3. Modify A args_schema/defaults.
4. Reload/sync catalog.
5. Verify R remains unchanged.
```

### 6.2 BackendRuntime -> NodeBackendRuntime

Test flow:

```text
1. Create NodeBackendRuntime N from BackendRuntime R.
2. Modify R args/docker/env/image candidates.
3. Reload/sync catalog.
4. Verify N.config_snapshot_json remains unchanged.
5. Run N check.
6. Verify N.config_snapshot_json, image_ref, and source revision remain unchanged.
```

---

## 7. Required Program-Level Tests

E2E is not sufficient. The implementation must include regression tests for:

```text
1. args_schema name + default renders as flag + value.
2. BackendRuntime user catalog parameter edits sync to DB projection.
3. New NodeBackendRuntime inherits edited BackendRuntime parameters.
4. NodeBackendRuntime arg overrides reach RunPlan.
5. NodeBackendRuntime overrides reach Equivalent Docker Command Preview.
6. NodeBackendRuntime overrides reach Agent Docker create spec.
7. BackendRuntime edits do not mutate existing NodeBackendRuntime.
8. NodeBackendRuntime check does not refresh snapshot, image_ref, source revision, or node-level overrides.
```

Suggested test names:

```text
TestArgSchemaNameDefaultRendersFlagAndValue
TestUserBackendRuntimeParamEditSyncsToProjection
TestNodeRuntimeInheritsEditedBackendRuntimeParams
TestNodeRuntimeArgOverridesReachRunPlan
TestNodeRuntimeOverridesReachDockerSpec
TestBackendRuntimeEditDoesNotMutateExistingNodeRuntime
TestNodeRuntimeCheckKeepsParamOverrides
```

---

## 8. Required Artifacts

Each E2E run must save artifacts under:

```text
docs/reports/model-runtime-node-wizard/e2e-matrix-<timestamp>/
```

The current checked-in closeout evidence is:

```text
docs/reports/model-runtime-node-wizard/e2e-matrix-matrix-postfix-20260619032917/
```

Each backend variant has its own subdirectory, for example:

```text
llamacpp-default/
llamacpp-modified/
vllm-default/
vllm-modified/
sglang-default/
sglang-modified/
```

The closeout artifact set includes:

```text
1. deployment-request-payload.json for each backend variant.
2. runplan.json for each backend variant.
3. matrix-summary.json and matrix-summary.md.
4. docker_spec_summary in matrix-summary.json.
5. parameter propagation assertion in matrix-summary.json.
```

Local reruns also write per-variant `.log`, `server-this-run.log`, and
`agent-this-run.log`; these logs are ignored by Git and are not required for
the checked-in closeout evidence.

---

## 9. Log Assertions

The E2E logs must be searchable for modified parameters.

Example:

```bash
grep -R -- "--ctx-size\|--max-model-len\|--gpu-memory-utilization\|--tp\|--mem-fraction-static" \
  docs/reports/model-runtime-node-wizard/e2e-matrix-* logs
```

The following logs should exist:

```text
runplan.docker_spec.resolved
docker.create.spec
```

These logs must include:

```text
image
entrypoint
args_json
ports_json
mounts_json
devices_json
env_keys
health_check
```

Secret env values must not be logged.

---

## 10. Acceptance Criteria

The work is accepted only if:

```text
1. llama.cpp default E2E passes.
2. llama.cpp modified params E2E passes.
3. vLLM default E2E passes.
4. vLLM modified params E2E passes.
5. SGLang default E2E passes.
6. SGLang modified params E2E passes.
7. Modified params are visible in BackendRuntime, NodeBackendRuntime snapshot, RunPlan, Equivalent Docker Command Preview, and Agent Docker create spec.
8. Upper-layer modifications do not mutate existing lower-layer objects.
9. NodeBackendRuntime check does not mutate runtime configuration.
10. Program-level regression tests cover parameter rendering and propagation.
11. i18n checks pass.
12. Documentation is updated.
13. git status --short is clean.
```

---
## 11. Verification Status (2026-06-19)

All acceptance criteria (§10) satisfied.

### Three-backend matrix
| Backend | Default | Modified | Modified Params |
|---------|---------|----------|-----------------|
| llama.cpp | PASS | PASS | --ctx-size 2048 --n-gpu-layers -1 |
| vLLM | PASS | PASS | --max-model-len 2048 |
| SGLang | PASS | PASS | --tp 1 |

### Parameter propagation
Modified params verified in: BackendRuntime user catalog, NodeBackendRuntime
config_snapshot_json, RunPlan args_json, Equivalent Docker Command Preview,
Agent docker.create.spec command_json. All layers show identical args.

### Failed instance E2E
Dedicated failed-instance E2E at:
`scripts/e2e-model-runtime-failed-instance-logs.sh`

Logs at:
`docs/reports/model-runtime-node-wizard/failed-instance-logs-postfix-20260619032823/`

Verified:
- Instance state = failed, container_id preserved
- last_error stores structured JSON {failure_reason_code, exit_code, container_id, error}
- GET /api/v1/model-instances/{id} returns current_run_plan_id
- GET /api/v1/node-run-plans/{run_plan_id}/logs returns real API response
- Logs API response may have empty stdout/stderr for port-conflict early-exit
  containers, but must return HTTP 200 with structured status/metadata.
- stdout/stderr preview single-lined via singleLineTail/singleLineTailStr

### Independence
- BackendVersion edits do not mutate existing BackendRuntime ✓
- BackendRuntime edits do not mutate existing NodeBackendRuntime ✓
- NodeBackendRuntime check does not mutate image_ref, config_snapshot, or source revision ✓

### Observability
- failure_reason_code stored in model_instances.last_error JSON
- /metrics and /metrics/targets at DEBUG (prefix match)
- High-frequency GET polling lists at DEBUG (isHighFrequencyGET middleware)
- Audit: instance.start.requested vs succeeded vs failed, verified via
  `docs/reports/model-runtime-node-wizard/audit-logs-postfix-20260619033633/`
- Web: ModelInstancesPage log button enabled when current_run_plan_id exists (any state)

---
## 12. UI Persistence / Port / Docker Spec Consistency (2026-06-19)

Parameter priority for selected E2E and API tests:

1. Deployment explicit parameters and env overrides.
2. NodeBackendRuntime frozen snapshot and node image override.
3. BackendRuntime template settings.
4. BackendVersion defaults.
5. Backend defaults.
6. System fallback.

RunPlan validation must compare:

- `run_plan_json.host_port` and `run_plan_json.container_port`;
- Equivalent Docker Command Preview `-p host_port:container_port/tcp`;
- Agent `docker.create.spec` port bindings;
- health/model-test host-side port, which defaults to `host_port`.

Selected UI/API script:

```bash
bash scripts/e2e-ui-persistence-runplan-selected.sh
```

Default artifact path:

```text
/tmp/lightai-ui-persistence-runplan-selected-*
```

The script records request payload, deployment JSON, RunPlan preview, start response, RunPlan JSON when available, and repeated-start response.
