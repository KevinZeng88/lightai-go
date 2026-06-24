# 00 - Runtime Parameter System Review Questions for MiMo

Status: discussion draft  
Target repo path: `docs/design/runtime-parameter-system/00-review-questions-for-mimo.md`  
Audience: MiMo / Claude / Codex reviewers before implementation

## 1. Purpose

This document is intentionally a review checklist, not an implementation order. MiMo should read the full design set first, then challenge assumptions, identify missing constraints, and propose an execution plan before changing code.

The current problem is not a single UI bug. It is a parameter-system modeling problem across:

- model metadata;
- backend version capability and arguments;
- vendor runtime profile;
- backend runtime template;
- node backend runtime, also called NBR;
- runner configuration;
- deployment override;
- RunPlan;
- Docker spec;
- logs and diagnostics.

The expected outcome is a clean and extensible parameter model that can support vLLM, SGLang, llama.cpp, NVIDIA, MetaX, Huawei Ascend, and future backends/vendors without cross-vendor pollution or duplicated parameter sources.

## 2. Review questions MiMo must answer before implementation

### 2.1 Current code inventory

MiMo must inspect the current implementation and answer:

1. Which files currently define BackendVersion schema, default arguments, default environment variables, health checks, and backend capabilities?
2. Which files currently define BackendRuntime and NodeBackendRuntime parameter values?
3. Which UI components currently render runtime parameters?
4. Which API handlers merge schema/default values into runtime snapshots?
5. Which resolver code converts parameter values into RunPlan args/env/devices/ports/high-risk Docker options?
6. Which code path creates Docker `Config` and `HostConfig`?
7. Which code path generates command preview / equivalent docker command?
8. Which code path classifies runtime warnings such as `LLAMA_ARG_HOST` being overwritten?
9. Which tests currently cover RuntimeParameterEditor, BackendRuntimesPage, ModelDeploymentsPage, resolver, and Docker driver behavior?
10. Which old compatibility or fallback behavior still exists and can now be deleted?

### 2.2 Modeling questions

MiMo must explicitly confirm or challenge these assumptions:

1. `command_template` is not a user editing surface. It is only a command generation template.
2. Parameter schema fields are the user editing surface.
3. Required parameters are always enabled and cannot be disabled.
4. Optional parameters use `enabled/value` semantics.
5. Disabled optional parameters retain `value`, but do not enter final RunPlan or Docker spec.
6. BackendVersion defines capability and schema. It must not contain node-specific paths, GPU IDs, host ports, or vendor devices.
7. VendorRuntimeProfile defines vendor-specific Docker/runtime defaults. It must not define backend serve arguments.
8. BackendRuntime is a template derived from BackendVersion and VendorRuntimeProfile.
9. NBR is the node-specific runtime configuration source.
10. Deployment override is the highest-priority deployment-local override and must not mutate NBR or BackendRuntime.
11. RunPlan should be able to report value source information for debugging.
12. Extra custom args/env/options must exist but cannot silently override structured core fields.

### 2.3 Backend coverage questions

MiMo must verify the target version/image behavior using both catalog files and live image help:

```bash
# vLLM

docker run --rm vllm/vllm-openai:latest vllm serve --help || true
docker run --rm vllm/vllm-openai:latest python -m vllm.entrypoints.openai.api_server --help || true

# SGLang

docker run --rm lmsysorg/sglang:latest python3 -m sglang.launch_server --help || true
docker run --rm lmsysorg/sglang:latest sglang serve --help || true

# llama.cpp

docker run --rm ghcr.io/ggml-org/llama.cpp:server-cuda13 --help || true
docker run --rm ghcr.io/ggml-org/llama.cpp:server-cuda13 llama-server --help || true
```

MiMo must report:

1. Which startup entrypoint is actually valid for each image?
2. Which core flags are accepted by each image?
3. Which common parameters changed name or behavior compared with existing catalog files?
4. Which parameters in this design are not supported by the current target image and should be omitted, hidden, or moved to extra args?
5. Which vendor-specific image variants are needed for NVIDIA, MetaX, and Huawei?

### 2.4 Vendor profile questions

MiMo must inspect current vendor-related defaults and answer:

1. Are `/dev/dri`, `/dev/mxcd`, `/dev/infiniband`, `group_add video`, `ipc=host`, `uts=host`, `privileged`, `security-opt seccomp=unconfined`, or `security-opt apparmor=unconfined` present anywhere outside MetaX-specific profile/config?
2. Are Huawei Ascend devices such as `/dev/davinci*`, `/dev/davinci_manager`, `/dev/devmm_svm`, `/dev/hisi_hdc`, `/usr/local/dcmi`, or CANN-related env/mounts present anywhere outside Huawei-specific profile/config?
3. Does NVIDIA runtime rely on Docker DeviceRequests / `--gpus` / GPU lease rather than hardcoded vendor devices?
4. Is `CUDA_VISIBLE_DEVICES` currently injected by platform default? If yes, why, and can it be disabled?
5. Are vendor profiles currently coupled to backend-specific args? If yes, how will they be separated?

### 2.5 UI/UX questions

MiMo must inspect UI behavior and answer:

1. Are all required core parameters shown as independent inputs, not raw textarea lines?
2. Are optional parameters shown with enabled/value control?
3. Does enabling a checkbox immediately make input/textarea/select editable?
4. Does disabling a parameter preserve its value?
5. Are duplicated editing surfaces removed?
6. Are high-risk Docker options available in exactly one authoritative location?
7. Is help text displayed via an external help catalog rather than hardcoded component strings?
8. Does each parameter show a `?` help affordance with meaning, default, recommendation, risk, layer, backend/vendor applicability, and source?
9. Does the UI make container port distinct from host port?
10. Does the UI make structured args distinct from custom extra args?

### 2.6 Test and evidence questions

MiMo must propose tests before code changes:

1. Which unit tests prove schema default construction works?
2. Which resolver tests prove source priority and template substitution work?
3. Which UI tests prove required/optional/enabled behavior works?
4. Which E2E script proves per-layer parameter tracing works?
5. Which E2E script proves real vLLM/SGLang/llama.cpp container behavior still works?
6. Which evidence files will be saved?
7. Which fields will be asserted in Docker inspect?
8. Which stale/legacy fields should be removed rather than migrated?

## 3. Required review output from MiMo

Before implementation, MiMo must produce a review response with:

1. Agreement/disagreement with the design principles.
2. Missing parameters or wrong assumptions found from official docs / image help.
3. Proposed file/schema changes.
4. Proposed UI changes.
5. Proposed API/resolver/Docker driver changes.
6. Proposed tests.
7. Proposed E2E evidence layout.
8. Risks and rollback-free clean-up plan.
9. Specific tasks ordered in small implementation batches.

## 4. Non-goals for this review stage

Do not implement immediately during review.

Do not add legacy compatibility for old DB values. If old DB data pollutes the current clean model, document it and rebuild the DB.

Do not create a new branch unless explicitly requested.

Do not leave fixable problems as future work.
