# Unified ConfigEdit Parameter Handling Closeout

Date: 2026-06-30

Status: PASS

Implementation commit: `ce4a77d fix(configedit): unify runtime parameter materialization`

## Root Cause

The ConfigEdit object/template implementation regressed because the projection layer treated some runtime-affecting parameters as special cases:

- `model_runtime.*` / `backend.arg.*` were hidden from `backend_runtime` views.
- `launcher.docker_options` was split through a fixed subfield list, so unmapped Docker/security/vendor keys stayed raw JSON-only.
- Runtime catalog environment-like keys embedded in `docker_options` were not materialized into `runtime.env`.
- Frontend runtime pages requested only `normal` view and showed raw Config JSON as ordinary diagnostics.
- `cap_add` was visible in catalog data but not compiled into final RunPlan / Agent Docker options.

## What Changed

Unified materialization now works as:

```text
ConfigSet item
  -> explicit/semantic metadata if present
  -> generic fallback classifier
  -> ConfigEdit field/component
  -> RunPlan / Agent Docker effect when applicable
```

Templates and semantic metadata may improve labels, renderers, sections, validation, and effects, but unmapped parameters are no longer dropped.

## Backend Coverage

The catalog regression test loads real built-in YAML and verifies structured ConfigEdit fields for:

- vLLM MetaX Docker
- vLLM NVIDIA Docker
- SGLang NVIDIA Docker
- llama.cpp NVIDIA Docker

Restored field categories include:

- Model Runtime Parameters: tensor/pipeline parallelism, max model length, memory fraction, dtype, request/concurrency controls, llama.cpp context/GPU-layer/cache controls.
- Backend Arguments: `backend.extra_args`.
- Container Options: `shm_size`, IPC, UTS, network, ulimits, GPU driver/capabilities.
- Vendor Device Mounts: MetaX `/dev/dri`, `/dev/mxcd`, `/dev/mem`.
- Security / High Risk: `privileged`, `security_options`, `cap_add`, `cap_drop`, device cgroup/PID/user namespace style fields.
- Environment: runtime env plus env-like catalog keys previously embedded in Docker options.

## Frontend Coverage

Affected pages/components:

- `BackendRuntimesPage.vue`
- `RunnerConfigsPage.vue`
- `ModelDeploymentsPage.vue`
- `BackendsPage.vue`
- `NodeRuntimeConfigWizard.vue`
- `DeploymentOverrideEditor.vue`
- `ConfigEditFieldMeta`

Runtime-related ConfigEdit callers now request `advanced` by default so structured runtime parameters are visible without raw JSON. Raw Config JSON is developer diagnostics only.

Technical keys are not shown in normal/advanced labels or tooltips; developer view may show technical keys.

## RunPlan Evidence

`shm_size` remains owned by `launcher.docker_options.shm_size` and compiles to:

```text
--shm-size <value>
```

MetaX Docker/security/env evidence:

```text
--device /dev/dri:/dev/dri
--device /dev/mxcd:/dev/mxcd
--group-add video
--security-opt seccomp=unconfined
--security-opt apparmor=unconfined
--cap-add SYS_PTRACE
--shm-size 100gb
--ulimit memlock=-1
```

RunPlan and Agent Docker DTOs now carry `cap_add` and `cap_drop`; Docker preview, source map, fake client, and real Docker HostConfig all receive them.

## Downstream Copy Evidence

The chain remains:

```text
BackendRuntime ConfigSet
  -> NodeBackendRuntime frozen config_set_json
  -> Deployment config_set_json / editable_config_patch
  -> RunPlan resolver
  -> Agent DockerSpec
```

Existing API tests verify NBR enable and Deployment create apply `editable_config_patch` to ConfigSet snapshots. New catalog tests verify built-in runtime ConfigSets expose complete fields before downstream copy.

## Tests Added Or Updated

Added:

- `internal/server/catalog/unified_configedit_materialization_test.go`

Updated:

- `internal/server/configedit/configedit_test.go`
- `internal/server/runplan/metax_huawei_test.go`
- `web/src/utils/__tests__/configEditFieldMeta.test.ts`

## Verification

Commands run:

```bash
git status --short
go test ./...
cd web
npm test
npm run test:unit
npm run build
```

Results:

- `go test ./...`: PASS
- `npm test`: PASS
- `npm run test:unit`: PASS
- `npm run build`: PASS

Build warnings:

- Vite/Rollup reported existing pure annotation and large chunk warnings. They did not fail the build.

## Problem Closure

| ID | Issue | Evidence | Status | Final Decision |
| -- | ----- | -------- | ------ | -------------- |
| UCP-001 | BackendRuntime hid model runtime parameters | Catalog ConfigEdit tests assert vLLM/SGLang/llama.cpp runtime fields exist in structured output | FIXED | Unified projection no longer hides runtime params at BackendRuntime layer |
| UCP-002 | Unmapped Docker/security options were raw JSON-only | Tests assert `cap_add`, `security_options`, `gpu_capabilities`, custom Docker keys materialize as fields | FIXED | Dynamic Docker option materialization added |
| UCP-003 | MetaX env-like keys in Docker options were not runtime env items | Tests assert `MACA_SMALL_PAGESIZE_ENABLE` and `PYTORCH_ENABLE_PG_HIGH_PRIORITY_STREAM` are in `runtime.env` | FIXED | Generic env-like catalog key normalization added |
| UCP-004 | `cap_add` did not compile into Docker effects | RunPlan MetaX test asserts `--cap-add SYS_PTRACE`; Agent/runtime tests pass | FIXED | RunPlan and Agent Docker specs carry cap fields |
| UCP-005 | Raw JSON was visible as a normal page surface | Frontend changes gate raw Config JSON behind developer view/diagnostics | FIXED | Raw JSON is developer representation only |

Unresolved problems from this round: none.

Problems existing only in chat: none.

Formal blocker document needed for this round: no.

## Push Result

Push is performed after this closeout document is committed. The final command result is reported in the assistant closeout response.

