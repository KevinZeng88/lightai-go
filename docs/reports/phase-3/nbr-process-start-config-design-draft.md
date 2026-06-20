# LightAI Go NBR Process Start Discovery & Config Design Draft

> Status: **DESIGN_DRAFT_REVISED**  
> Purpose: design discussion and implementation planning input only  
> Implementation: **not approved yet**  
> Date: 2026-06-21 (revised 2026-06-21)  
> Scope: Model serving container startup, especially Layer 3: Docker `ENTRYPOINT` / `CMD` / command prefix discovery and configuration.  
> Review: `docs/reports/phase-3/nbr-process-start-config-claude-review.md`  
> Audit: `docs/reports/phase-3/docker-launch-parameter-chain-audit.md`

---

## 0. Claude Code-Aware Review Summary

A code-aware review was completed against the current codebase. Key verified conclusions:

1. **Four-layer model is consistent with current code.** The audit (`docker-launch-parameter-chain-audit.md`) confirmed Layers 1, 2, and 4 have existing mechanisms. Layer 3 is currently implicit in `BackendVersion.default_entrypoint_json` / `BackendRuntime.entrypoint_override_json`.

2. **NBR as Layer 3 authority is code-verified.** `config_snapshot_json` is a flat `map[string]interface{}` in a `TEXT` column — adding `process_start_config` as a new top-level key requires no DB migration. `probe_results_json` can similarly accept `process_start_detection` as a new top-level key.

3. **`nil` Entrypoint → Docker-preserve chain works end-to-end.** Verified: `resolver.go:240` → `types.go:8` (omitempty) → agentSpec JSON → agent unmarshal → `docker_real.go:82` (`len(nil) > 0` is false) → Docker preserves image ENTRYPOINT.

4. **`command_prefix` must go into `Config.Cmd`, not `Config.Entrypoint`.** This matches the external `docker run IMAGE python3 -m sglang.launch_server <args>` pattern. The final Cmd is `command_prefix + buildArgs()`.

5. **GPU DeviceRequest and raw device passthrough are independent of entrypoint.** Changing entrypoint to nil for "image_default" mode has zero effect on Layer 2 HostConfig fields (NVIDIA DeviceRequest, MetaX `/dev/*` devices, etc.).

6. **`process_start_profiles` on BackendRuntime requires new DB column or code-based storage.** The `backend_runtimes` table has no generic extensible JSON column. v1 recommendation: Go constants or YAML catalog file, avoiding migration.

7. **`mergeNBRConfigSnapshot` and `applyDeploymentConfigSnapshot` need explicit key additions.** Both use hardcoded key lists or explicit field extraction — no generic pass-through. Adding `process_start_config` requires explicit code changes in both functions.

These conclusions are incorporated throughout this revised draft.

LightAI Go starts model serving containers by combining image selection, Docker/HostConfig settings, process start semantics, and model service parameters. Most of the mechanism already exists, but the **process start layer** is currently implicit and spread across `BackendVersion.default_entrypoint_json`, `BackendRuntime.entrypoint_override_json`, Docker image metadata, RunPlan resolution, and Agent Docker create behavior.

The proposed direction is:

1. Keep the existing four-layer startup model.
2. Treat **NodeBackendRuntime (NBR)** as the node-level authority for the selected image and its actual runtime configuration.
3. Introduce a clear Layer 3 concept: **Process Start Config**.
4. Do not make users fill `ENTRYPOINT` / `CMD` from scratch. BackendRuntime/catalog should provide **candidate process start profiles** and detection methods.
5. During NBR creation/check, LightAI should inspect the selected image, evaluate candidate profiles, generate a `process_start_detection` result with evidence/confidence/warnings, and allow the user to accept, trial-run, customize, or manually override it.
6. The accepted result becomes `NBR.process_start_config` and is later frozen into Deployment snapshot and resolved by RunPlan.
7. Model service parameters such as `--model`, `--model-path`, `--host`, and `--port` remain in the existing Layer 4 parameter mechanism. They must not be duplicated into Process Start Config.
8. Docker/HostConfig/hardware exposure such as NVIDIA `DeviceRequest`, MetaX `/dev/*` devices, `group_add`, `privileged`, `security_opt`, `ipc`, `shm`, and `ulimits` remain in Layer 2. Layer 3 must not replace or simplify these settings.

The target model is:

```text
BackendRuntime / Catalog:
  backend_family + candidate process_start_profiles + detection rules

Agent / NBR check:
  image list + image inspect + probe evidence

NBR:
  selected image_ref
  Docker/HostConfig/hardware config
  process_start_detection (system suggestion)
  process_start_config (user accepted/customized actual config)

Deployment:
  freezes NBR process_start_config

RunPlan:
  Docker Config.Entrypoint
  Docker Config.Cmd = command_prefix + existing Layer 4 model service args

Agent Docker Create:
  executes exactly what RunPlan resolves
```

---

## 2. Context and Motivation

Recent investigation showed that external direct `docker run` baselines for vLLM and SGLang can succeed on the local WSL2/NVIDIA Docker runtime, while LightAI-managed real smoke has failed in some runs. The prior audit concluded that the root cause is likely in the parameter chain and/or Docker API create semantics rather than a simple host-level Docker/NVIDIA runtime failure.

Existing audit and design draft documents have already identified:

- Layers 1, 2, and 4 largely exist.
- Layer 3 is currently implicit.
- vLLM and llama.cpp images often use useful built-in image `ENTRYPOINT` values.
- SGLang images may use an NVIDIA wrapper entrypoint such as `/opt/nvidia/nvidia_entrypoint.sh`; overriding it can bypass wrapper behavior.
- NBR already represents node-level runtime/image configuration and snapshots selected runtime configuration.
- Existing NBR/Deployment snapshot behavior means catalog/seed changes do not automatically change existing NBRs or deployments.

The current design task is therefore not to rebuild the entire startup subsystem, but to make the missing Layer 3 explicit, discoverable, and safely configurable.

---

## 3. Four-Layer Startup Model

### 3.1 Layer Overview

| Layer | Name | Responsibility | Current Status |
|---|---|---|---|
| Layer 1 | Image | Select final Docker image | Existing: `BackendRuntime.image_name`, `NBR.image_ref`, Deployment snapshot, RunPlan image |
| Layer 2 | Docker / HostConfig / Hardware | Ports, volumes, env, devices, GPU exposure, ipc, shm, security, privileged, ulimits | Existing, but must preserve vendor differences |
| Layer 3 | Process Start | Docker `ENTRYPOINT` / `CMD`, image default vs custom entrypoint, command prefix, shell mode | Currently implicit; design target |
| Layer 4 | Model Service Params | Backend-specific model-serving parameters such as model path, host, port, tp, max length | Existing: `default_args_json`, `parameter_defs_json`, `args_override_json`, deployment params |

### 3.2 Layer Boundaries

Layer 3 must not absorb the responsibilities of the other layers.

Layer 3 **does not** decide:

- Which image to use.
- Which GPU/device to expose.
- Whether to mount `/dev/dri`, `/dev/mxcd`, `/dev/infiniband`, or NVIDIA `DeviceRequest`.
- What host port or container port is selected.
- What model path is mounted.
- What `--model`, `--model-path`, `--host`, `--port`, `--tp`, or `--max-model-len` values are.

Layer 3 **does** decide:

- Whether Docker image `ENTRYPOINT` should be preserved.
- Whether Docker `Config.Entrypoint` should be set explicitly.
- Whether a command prefix such as `python3 -m sglang.launch_server` should be prepended to `Config.Cmd`.
- Whether shell mode is needed for vendor wrapper images.
- How to document evidence, confidence, and warnings for the process start selection.

---

## 4. Terminology

### 4.1 Backend Family

A normalized backend family name, for example:

```text
vllm
sglang
llamacpp
custom
```

Profiles and detection must primarily use `backend_family`, not official image repository names. Official image names can be evidence, but must not be required for detection because vendor/private/custom images must remain supported.

### 4.2 Process Start Profile

A candidate startup skeleton for a backend family. It does not include concrete model path or service port values. It describes how to combine image entrypoint behavior with a command prefix.

Example:

```json
{
  "id": "sglang.python_module_launcher",
  "backend_family": "sglang",
  "entrypoint_mode": "image_default",
  "command_prefix": ["python3", "-m", "sglang.launch_server"],
  "priority": 100,
  "description": "Run SGLang through python module launcher as Docker CMD, preserving image ENTRYPOINT."
}
```

### 4.3 Process Start Detection

A system-generated suggestion based on backend family, candidate profiles, image inspect evidence, probe evidence, and optional trial-run results.

It should be safe to overwrite when a new probe/check is run, because it is not the authoritative runtime configuration.

### 4.4 Process Start Config

The accepted actual configuration saved on NBR. It may come from accepting a detection, selecting an alternate profile, custom editing, or manual override.

This is the authoritative Layer 3 configuration for that NBR and image.

---

## 5. Core Design Principles

1. **NBR is the authoritative landing point for Layer 3.**  
   BackendRuntime/BackendVersion can recommend, but the actual selected image and runtime behavior belongs to the node-level runtime configuration.

2. **BackendRuntime/catalog should provide candidate profiles, not one mandatory default config.**  
   A backend family may have multiple valid startup styles.

3. **Detection should find at least one workable startup style, not prove a unique correct answer.**  
   If multiple candidates are possible, rank them by confidence and risk.

4. **Do not match profiles by official image repository name.**  
   Match primarily by backend family and image Entrypoint/Cmd characteristics. Use `image_ref` only as evidence or a confidence signal.

5. **Detection and configuration are separate.**  
   The system can generate `process_start_detection`, but it should not silently overwrite `process_start_config`.

6. **Trial-run probing must be explicit.**  
   Starting a Docker container, even temporarily, must require a user-triggered action or clearly scoped workflow.

7. **Layer 3 should produce Docker API semantics, not shell strings.**  
   Default output should be token arrays: `Config.Entrypoint` and `Config.Cmd`. Shell mode should be explicit and high-risk.

8. **Layer 4 model service args remain separate.**  
   `process_start_config` only contributes `entrypoint` and `command_prefix`; model args come from the existing argument-building pipeline.

9. **Layer 2 hardware exposure remains separate.**  
   `--gpus all` is not a universal solution. NVIDIA, MetaX, Huawei, and vendor devices must retain their own HostConfig/device paths.

10. **Backward compatibility must be explicit.**  
   Missing `process_start_config` should preserve current behavior unless and until a migration/upgrade workflow is approved.

---

## 6. Existing Mechanism Summary

This section summarizes the current intended field mapping. Claude should verify exact file/line references on the development server before implementation.

### 6.1 Image Layer

Likely existing sources:

- `BackendVersion.default_images_json`
- `BackendRuntime.image_name`
- `NodeBackendRuntime.image_ref`
- `NBR.config_snapshot_json`
- `Deployment.config_snapshot_json.nbr_image_ref`
- `ResolvedRunPlan.Image`
- Agent task payload image

Expected priority:

```text
NBR image_ref / node runtime override
  > BackendRuntime.image_name
  > BackendVersion.default_images_json[vendor]
```

### 6.2 Docker / HostConfig / Hardware Layer

Likely existing fields:

- `docker_json`
- `model_mount_json`
- `default_env_json`
- deployment service JSON
- placement GPU IDs
- Agent Docker create options

Examples of Layer 2 content:

```text
ports / port bindings
volumes / binds / mounts
env
model_mount_json
NVIDIA DeviceRequest
raw devices
ipc_mode
uts_mode
shm_size
privileged
security_options
group_add
ulimits
vendor_visible_devices / CUDA_VISIBLE_DEVICES
```

### 6.3 Model Service Params Layer

Likely existing sources:

- `BackendVersion.default_args_json`
- `BackendVersion.default_backend_params_json`
- `BackendVersion.parameter_defs_json`
- `BackendRuntime.args_override_json`
- `Deployment.parameters_json`
- `buildArgs()` and post-processing such as deduplication and service port application

Examples:

```text
vLLM:      --model, --host, --port, --served-model-name, --max-model-len
SGLang:   --model-path, --host, --port, --tp, --mem-fraction-static
llama.cpp:-m, --host, --port, --ctx-size, -ngl / --n-gpu-layers
```

These stay out of Layer 3.

### 6.4 Current Process Start Layer

Current behavior appears to be implicit:

```text
BackendVersion.default_entrypoint_json
  overridden by BackendRuntime.entrypoint_override_json if non-empty
  passed to RunPlan.Entrypoint
  passed to Agent Docker create as Config.Entrypoint if len > 0
```

Observed implications:

- Non-empty entrypoint means Docker image entrypoint is overridden.
- Empty entrypoint means the Agent does not set `Config.Entrypoint`, so Docker preserves the image entrypoint.
- There is no explicit distinction between “unset”, “preserve image default”, and “clear entrypoint”.
- There is no independent command prefix concept.

---

## 7. BackendRuntime / Catalog Process Start Profiles

### 7.1 Why Profiles Instead of a Single Default

A backend family can be started in more than one way. For example:

- vLLM may use image default entrypoint `vllm serve` with CMD args.
- vLLM may also be explicitly started with custom entrypoint `vllm serve`.
- SGLang may be launched via `python3 -m sglang.launch_server` as CMD behind an image wrapper.
- SGLang may be launched with a direct custom entrypoint in images that do not have a wrapper.
- Vendor images may use a bash/script wrapper.
- llama.cpp server images often have `/app/llama-server` as image entrypoint.

Therefore BackendRuntime/catalog should provide a list of candidates and detection hints.

**The goal is not to prove the unique correct startup method.** The goal is to find at least one workable startup method for the selected `backend_family` + image, rank candidates by confidence and risk, and allow user acceptance, trial-run validation, customization, or manual override.

目标不是证明唯一正确的启动方式，而是找到至少一种可用的启动方式，按置信度和风险排序，允许用户接受建议、试启动验证、自定义或手工覆盖。

### 7.2 Profile Storage — v1 Approach

Conceptually, `process_start_profiles` belong to the BackendRuntime / catalog layer.

**v1 physical storage recommendation**: Go constants or YAML catalog files. NOT a new `backend_runtimes` DB column.

Rationale:
- The `backend_runtimes` table (`internal/server/models/runtime.go:5-28`) has no generic extensible JSON field. Existing JSON columns (`docker_json`, `entrypoint_override_json`, `version_snapshot_json`) all have specific, typed purposes.
- Adding `process_start_profiles_json TEXT` would require a new DB migration (e.g., V25).
- v1 avoids migration by defining profiles in code, where they can be versioned alongside the resolver logic.
- Future: if user-customizable runtime profiles are needed, a dedicated column or file-based catalog extension can be added.

Profiles map to `backend_family` (vllm, sglang, llamacpp, ollama, custom), not to image repository names. This is a hard requirement — vendor images, private registries, and custom images must all be supported.

### 7.2 Proposed Candidate Profile Structure

This is a conceptual structure for review, not a committed schema:

```json
{
  "process_start_profiles": [
    {
      "id": "sglang.python_module_launcher",
      "backend_family": "sglang",
      "entrypoint_mode": "image_default",
      "entrypoint": [],
      "command_prefix": ["python3", "-m", "sglang.launch_server"],
      "shell_mode": false,
      "priority": 100,
      "detection_hints": {
        "entrypoint_kinds": ["wrapper", "empty", "unknown"],
        "avoid_if_entrypoint_already_starts_backend": true
      },
      "warnings": []
    },
    {
      "id": "sglang.custom_entrypoint",
      "backend_family": "sglang",
      "entrypoint_mode": "custom",
      "entrypoint": ["python3", "-m", "sglang.launch_server"],
      "command_prefix": [],
      "shell_mode": false,
      "priority": 40,
      "warnings": ["May bypass image ENTRYPOINT wrapper."]
    }
  ]
}
```

### 7.3 Matching Rules Must Not Depend on Official Image Names

Bad rule:

```json
{
  "match_image_ref": ["lmsysorg/sglang:*"]
}
```

Better rule:

```json
{
  "backend_family": "sglang",
  "entrypoint_kinds": ["wrapper", "empty", "unknown"],
  "command_prefix": ["python3", "-m", "sglang.launch_server"]
}
```

`image_ref` can still be recorded as evidence:

```json
{
  "evidence": {
    "image_ref": "registry.local/vendor/custom-sglang:202606",
    "image_entrypoint": ["/opt/vendor/entrypoint.sh"]
  }
}
```

But it must not be the primary condition for profile selection.

---

## 8. Detection Flow

### 8.1 Profile Matching Principles

Process start profiles must match primarily by `backend_family` and image `Entrypoint`/`Cmd` characteristics, NOT by official image repository name (`image_ref`).

启动 Profile 必须主要基于 `backend_family` 和 image `Entrypoint`/`Cmd` 特征匹配，不能绑定官方镜像仓库名。`image_ref` 只能作为证据或置信度参考，因为厂商镜像、私有镜像、定制镜像必须可识别。

**Hard requirement**: No matching rule like `"match_image_ref": ["lmsysorg/sglang:*"]` or `"match_image_ref": ["vllm/vllm-openai:*"]`. Matching is by `backend_family` + entrypoint shape classification.

### 8.2 Inputs

Detection should use:

```text
backend_family
BackendRuntime process_start_profiles[]
image inspect Config.Entrypoint
image inspect Config.Cmd
image labels/env/exposed ports
NBR probe_results_json
optional static script analysis (deferred)
optional trial-run probe result (deferred)
user override history
```

### 8.2 Output: process_start_detection

Example:

```json
{
  "process_start_detection": {
    "status": "candidate_found",
    "selected_profile_id": "sglang.python_module_launcher",
    "entrypoint_mode": "image_default",
    "entrypoint": [],
    "command_prefix": ["python3", "-m", "sglang.launch_server"],
    "shell_mode": false,
    "confidence": "high",
    "source": "backend_profile+image_inspect",
    "candidate_profiles": [
      {
        "id": "sglang.python_module_launcher",
        "score": 92,
        "confidence": "high",
        "reasons": [
          "backend_family=sglang",
          "image entrypoint looks like wrapper",
          "profile preserves image ENTRYPOINT and passes launcher as CMD"
        ],
        "warnings": []
      },
      {
        "id": "sglang.custom_entrypoint",
        "score": 45,
        "confidence": "low",
        "reasons": [
          "backend_family=sglang",
          "direct custom entrypoint is a known fallback"
        ],
        "warnings": [
          "May bypass image ENTRYPOINT wrapper."
        ]
      }
    ],
    "evidence": {
      "backend_family": "sglang",
      "image_ref": "registry.local/vendor/custom-sglang:202606",
      "image_entrypoint": ["/opt/vendor/entrypoint.sh"],
      "image_cmd": null,
      "matched_signals": [
        "backend_family",
        "entrypoint_wrapper_shape"
      ]
    },
    "warnings": []
  }
}
```

### 8.3 Output: process_start_config

User acceptance or manual edit creates the authoritative NBR config:

```json
{
  "process_start_config": {
    "profile_id": "sglang.python_module_launcher",
    "entrypoint_mode": "image_default",
    "entrypoint": [],
    "command_prefix": ["python3", "-m", "sglang.launch_server"],
    "shell_mode": false,
    "source": "user_accepted_detection",
    "confidence": "high",
    "warnings": []
  }
}
```

---

## 9. Detection Levels

### 9.1 Level 1: Static Image Inspect

Default and low-risk.

Actions:

```text
docker image inspect <image>
read Config.Entrypoint
read Config.Cmd
read labels/env/exposed ports
```

No container execution.

### 9.2 Level 2: Profile Matching and Scoring

Default and low-risk.

Actions:

```text
filter profiles by backend_family
classify image Entrypoint/Cmd shape
score profiles
produce process_start_detection
```

Entrypoint shape examples:

```text
empty
backend_server_binary
nvidia_wrapper
vendor_wrapper
shell_script
unknown_binary
```

### 9.3 Level 3: Static Script Probe

Optional, explicit or advanced.

Possible approach:

```text
docker create <image>
docker cp <container>:/path/to/entrypoint.sh ...
docker rm <container>
analyze script text
```

This does not start the container process, does not load a model, and does not require GPU. It can inspect whether a wrapper script calls `exec "$@"`, swallows CMD, or starts the service itself.

This should be best-effort and non-blocking.

### 9.4 Level 4: Trial-Run Probe

Explicit user action only.

There are two variants:

#### NBR-Level Lightweight Trial

Goal: verify launcher existence or `--help`, not model startup.

Examples:

```bash
docker run --rm <image> python3 -m sglang.launch_server --help
docker run --rm <image> vllm serve --help
docker run --rm <image> /app/llama-server --help
```

This starts a container but does not load a model. It can still be slow/risky for some images, so it needs user confirmation.

#### Start Wizard / Deployment Trial

Goal: verify the full startup path.

Inputs:

```text
NBR.process_start_config
Layer 2 Docker/HostConfig/hardware config
ModelLocation
Layer 4 model service params
health check
```

Behavior:

```text
create temporary container
start with resolved Entrypoint/Cmd
wait for health check
request /v1/models or configured health endpoint
collect logs/evidence
cleanup by default
```

This is the strongest proof that at least one startup method works.

---

## 10. Manual Override and Custom Mode

Users must be able to override the detected startup method.

Manual controls:

```text
entrypoint_mode: image_default | custom
entrypoint: string[]
command_prefix: string[]
shell_mode: boolean
warnings / notes
```

Potential UI actions:

```text
[Auto Detect]
[Apply Suggestion]
[Edit Manually]
[Test This Startup]
[Reset to Catalog Profiles]
```

Manual override should set:

```json
{
  "source": "user_override",
  "confidence": "user_confirmed"
}
```

If shell mode is enabled, the UI/API should flag high risk because shell strings are less safe and harder to preview accurately.

---

## 11. NBR Storage and Snapshot Design

### 11.1 Authoritative Landing Point: `config_snapshot_json.process_start_config` (Top-Level Key)

`process_start_config` MUST be a top-level key in `config_snapshot_json`, NOT nested inside `docker_json`.

Rationale:
- `docker_json` belongs to Layer 2 (Docker/HostConfig/Hardware exposure) — it carries `ipc_mode`, `shm_size`, `privileged`, `devices`, `security_options`, `group_add`, `ulimits`, `gpu_visible_env_key`.
- `process_start_config` belongs to Layer 3 (Process ENTRYPOINT/CMD).
- Mixing them violates layer separation and creates ambiguity about which `docker_json` fields are Layer 2 vs Layer 3.
- The snapshot is a flat `map[string]interface{}` with a `TEXT` column — adding a new top-level key requires no schema migration (verified in code review: `runtime_handlers.go:798`).

Recommended shape:

```json
{
  "config_snapshot_json": {
    "image_name": "...",
    "docker_json": { "...": "Layer 2 Docker/HostConfig/hardware settings" },
    "model_mount_json": { "...": "Layer 2 Model mount settings" },
    "entrypoint_override_json": "...",
    "args_override_json": "...",
    "process_start_config": {
      "entrypoint_mode": "image_default",
      "entrypoint": [],
      "command_prefix": [],
      "shell_mode": false,
      "profile_id": "sglang.python_module_launcher",
      "source": "user_accepted_detection",
      "confidence": "high",
      "warnings": []
    }
  },
  "probe_results_json": {
    "process_start_detection": { "...": "system suggestion" }
  }
}
```

**IMPORTANT**: Implementation must explicitly add `"process_start_config"` to:
1. `buildRuntimeConfigSnapshot()` (`runtime_handlers.go:798`) — capture from BR config
2. `buildDeploymentRuntimeSnapshot()` (`deployment_lifecycle_handlers.go:59`) — capture for deployment freeze
3. `mergeNBRConfigSnapshot()` hardcoded key list (`deployment_lifecycle_handlers.go:104-108`) — propagate NBR → Deployment
4. `applyDeploymentConfigSnapshot()` (`deployment_lifecycle_handlers.go:922`) — extract at start/dry-run time

**Rejected alternative**: Nesting `process_start_config` inside `docker_json`. This was the initial proposal in an earlier design round. It is rejected because it mixes Layer 2 (Docker daemon config) with Layer 3 (process entrypoint). The two layers have different lifecycle, ownership, and modification patterns.

### 11.2 Detection Storage

`process_start_detection` goes into `NBR.probe_results_json.process_start_detection`.

Rationale:
- Detection is probe output — it can be regenerated on re-probe without affecting runtime configuration.
- `probe_results_json` is a `map[string]interface{}` in a `TEXT` column — adding a new key requires no schema migration (verified in code review: `runtime_handlers.go:342-347`).
- Detection does NOT automatically overwrite `process_start_config`. The user must explicitly accept/apple the suggestion.
- Detection CAN be overwritten when a new probe/check runs, because it is not authoritative.

```text
process_start_detection  →  NBR.probe_results_json  (system-generated, regeneratable)
process_start_config     →  NBR.config_snapshot_json  (user-confirmed, authoritative, frozen into deployments)
```

---

## 12. RunPlan Semantics

### 12.1 Missing Config

If `process_start_config` is missing:

```text
Use legacy behavior exactly as today.
```

This preserves existing NBRs and existing deployments.

### 12.2 image_default

```text
Docker Config.Entrypoint = nil / unset
Docker Config.Cmd = command_prefix + Layer 4 buildArgs()
```

This preserves the image's own entrypoint.

### 12.3 custom

```text
Docker Config.Entrypoint = process_start_config.entrypoint
Docker Config.Cmd = command_prefix + Layer 4 buildArgs()
```

Use this only when users or catalog profiles explicitly want to override image entrypoint.

### 12.4 `command_prefix` Placement in CMD

The `command_prefix` is prepended to `Config.Cmd`, NOT placed in `Config.Entrypoint`. This matches the external `docker run IMAGE python3 -m sglang.launch_server <args>` pattern.

```text
modelArgs  = buildArgs(...)  // existing Layer 4 mechanism (4-layer merge + dedup + applyServiceArgs)
finalCmd   = process_start_config.command_prefix + modelArgs
```

**Implementation constraints**:
- `command_prefix` must NOT enter the Layer 4 `buildArgs()` pipeline — it is not a model parameter and should not be deduplicated or affected by `applyServiceArgs`.
- `command_prefix` is prepended AFTER `buildArgs()` returns, BEFORE the result is stored in `ResolvedRunPlan.Args`.
- If current `buildArgs()` internally performs dedup and `applyServiceArgs`, the implementation must verify the exact function boundary before prepending `command_prefix`.

### 12.5 clear

Do not implement initially.

Reason:

```text
Docker API nil vs empty Entrypoint semantics need careful handling and tests.
```

If needed later, `clear` must have explicit Agent Docker create support and tests proving Docker receives an empty entrypoint override.

### 12.5 shell_mode

Do not implement initially unless a specific vendor image requires it.

If implemented later:

```text
shell_mode=true should be explicit high risk
command preview must show exact shell string
arguments must be escaped carefully
```

---

## 13. Backend-Specific Target Behaviors

### 13.1 vLLM

Target external-equivalent behavior:

```bash
docker run ... vllm/vllm-openai:latest \
  --model /models/qwen \
  --host 0.0.0.0 \
  --port 8000
```

Process start:

```json
{
  "entrypoint_mode": "image_default",
  "entrypoint": [],
  "command_prefix": []
}
```

RunPlan:

```text
Config.Entrypoint = nil
Config.Cmd = Layer 4 vLLM args
```

Notes:

- If the image entrypoint is `vllm serve`, preserving image entrypoint is equivalent to Docker CLI baseline.
- vLLM bare model path vs `--model /path` is a Layer 4 issue and must be discussed separately.
- `ParameterDef.Value` gap is also a Layer 4 issue and must not be mixed into Layer 3 implementation.

### 13.2 SGLang

Target external-equivalent behavior:

```bash
docker run ... <sglang image> \
  python3 -m sglang.launch_server \
  --model-path /models/qwen \
  --host 0.0.0.0 \
  --port 30000
```

Process start:

```json
{
  "entrypoint_mode": "image_default",
  "entrypoint": [],
  "command_prefix": ["python3", "-m", "sglang.launch_server"]
}
```

RunPlan:

```text
Config.Entrypoint = nil
Config.Cmd = ["python3", "-m", "sglang.launch_server"] + Layer 4 SGLang args
```

Important:

- Do not match by `lmsysorg/sglang`. Vendor/private images should work.
- Match by backend family `sglang` and image entrypoint/cmd shape.
- If image entrypoint is wrapper-like, preserving it is safer.
- If image entrypoint already launches SGLang, detection should avoid duplicate command_prefix and warn.

### 13.3 llama.cpp

Target external-equivalent behavior:

```bash
docker run ... ghcr.io/ggml-org/llama.cpp:server-cuda13 \
  -m /models/model.gguf \
  --host 0.0.0.0 \
  --port 8080
```

Process start:

```json
{
  "entrypoint_mode": "image_default",
  "entrypoint": [],
  "command_prefix": []
}
```

RunPlan:

```text
Config.Entrypoint = nil
Config.Cmd = Layer 4 llama.cpp args
```

This formalizes the behavior that already works today.

---

## 14. UI / API Behavior Boundaries

**Principle**: API-first first, Web UI later. The API behavior for detection, config acceptance, and trial-run must be designed and tested via shell E2E scripts before any Web UI work begins.

Supported API-level actions (conceptual, not implemented):

- **Auto Detect**: Trigger static image inspect + profile matching → produce `process_start_detection`
- **Apply Suggestion**: Copy `process_start_detection` fields into `process_start_config` on NBR
- **Edit Manually**: PATCH NBR `config_snapshot_json.process_start_config` with custom values
- **Test This Startup** (deferred): Trial-run probe with explicit user trigger
- **Reset to Catalog Profiles** (deferred): Re-run detection from profiles

### 14.1 NBR Wizard / Runtime Wizard (Conceptual)

When user selects a backend runtime and image from Agent:

1. Show selected backend family.
2. Show selected image.
3. Run static image inspect.
4. Evaluate candidate profiles.
5. Show recommended process start detection.
6. Let user apply, edit, or trial-run.

Suggested UI section:

```text
Process Start Detection

Image ENTRYPOINT:
  /opt/nvidia/nvidia_entrypoint.sh

Image CMD:
  empty

Recommended startup:
  Keep image ENTRYPOINT
  CMD prefix: python3 -m sglang.launch_server
  Model args: generated from existing model service parameter settings

Confidence:
  High

Reason:
  Backend family is SGLang and image entrypoint looks like a wrapper.

Actions:
  [Apply Suggestion] [Edit Manually] [Test This Startup]
```

### 14.2 Start Wizard / Deployment Wizard

When NBR + ModelLocation + service parameters are all selected:

1. Show full RunPlan.
2. Show Docker API equivalent:
   - image
   - Entrypoint
   - Cmd
   - env
   - ports
   - mounts
   - hardware devices
3. Provide optional real trial-run button.
4. Save evidence/logs.

---

## 15. Command Preview Requirements

The command preview must reflect Docker API semantics.

### 15.1 image_default

If `Config.Entrypoint` is unset:

```text
Do not show --entrypoint.
Optionally annotate: # preserves image ENTRYPOINT
```

Example:

```bash
docker run ... <image> python3 -m sglang.launch_server --model-path /models/qwen --host 0.0.0.0 --port 30000
```

### 15.2 custom

If `Config.Entrypoint` is explicitly set:

```bash
docker run ... --entrypoint "python3 -m sglang.launch_server" <image> --model-path /models/qwen ...
```

But note: real Docker CLI `--entrypoint` accepts one executable, and arguments are subtle. The preview must avoid misleading users. It may be better to show Docker API fields explicitly:

```json
{
  "Config.Entrypoint": ["python3", "-m", "sglang.launch_server"],
  "Config.Cmd": ["--model-path", "/models/qwen", "--host", "0.0.0.0", "--port", "30000"]
}
```

### 15.3 Recommended Preview Format

Show both:

1. Human-readable equivalent command.
2. Exact Docker API config fields.

This prevents ambiguity.

---

## 16. Existing NBR and Deployment Compatibility

### 16.1 Existing NBRs

Existing NBRs without `process_start_config` should keep legacy behavior.

They can optionally run Auto Detect and Apply Suggestion to create a new config.

### 16.2 Existing Deployments

Existing deployments should continue to use frozen snapshots.

If a NBR is updated later, existing deployments should not silently change. Users should create/update deployments intentionally.

### 16.3 Catalog Updates

Adding profiles to BackendRuntime/catalog affects future NBR creation/check workflows, not existing NBRs unless users explicitly apply recommendations.

---

## 17. Storage Implementation (Resolved)

The landing points have been finalized in §11:

- `process_start_config` → `config_snapshot_json.process_start_config` (top-level key)
- `process_start_detection` → `probe_results_json.process_start_detection` (top-level key)
- `process_start_profiles` (v1) → Go constants or YAML catalog files (no DB column)

Both `config_snapshot_json` and `probe_results_json` are `TEXT` columns containing flat `map[string]interface{}` — adding new top-level keys requires no DB migration. The `mergeNBRConfigSnapshot` hardcoded key list and `applyDeploymentConfigSnapshot` explicit field extraction must be updated (see §11.1 implementation notes).

---

## 18. Out of Scope for Initial Implementation

Do not include initially:

```text
Full parameter_schema redesign
New command_template system
GPU mode all/specific redesign
Replacing vendor-specific devices with --gpus all
Shell mode implementation (field reserved, default false)
Entrypoint clear mode
Long-running automatic real smoke during NBR creation
Web frontend overhaul
Automatic silent overwrite of NBR config
Network-based online documentation lookup at runtime
```

**Explicitly Layer 4 — Separate Issues**:

```text
vLLM default_args_json bare positional model path → Layer 4
ParameterDef.Value field gap → Layer 4 Go struct + seed data issue
--host / --port / --model-path migration into Layer 3 → NOT happening
```

These Layer 4 issues are documented in the audit (`docker-launch-parameter-chain-audit.md`) and should be tracked as separate design/implementation items. They are NOT part of this Layer 3 design.

---

## 19. Implementation Phases (Draft)

These are draft phases for later approval. Each phase is self-contained: Phase 1 does not change container startup behavior; Phase 2 does not change RunPlan resolution; Phase 3 is the first phase that changes actual Docker API calls.

### Phase 1: Static Profiles + Detection (READ-ONLY)

**Goal**: Generate `process_start_detection` without changing any container startup behavior.

```text
- Define ProcessStartConfig and ProcessStartProfile Go types
- Define default profiles as Go constants (runplan/profiles.go)
  or YAML catalog files
  vLLM:    image_default + empty command_prefix
  SGLang:  image_default + ["python3","-m","sglang.launch_server"]
  llama.cpp: image_default + empty command_prefix
- Implement static detection:
  filter profiles by backend_family
  classify image Entrypoint/Cmd shape
  score candidates
  produce process_start_detection
- Store detection in NBR.probe_results_json.process_start_detection
  (new top-level key in probe results)

Does NOT:
- Write process_start_config
- Change RunPlan resolution
- Change Agent Docker Create
- Change any container startup behavior
```

### Phase 2: Config Acceptance + Snapshot Flow

**Goal**: Allow users to accept detection into NBR config, freeze it into Deployment snapshot.

```text
- Add "process_start_config" key to:
  buildRuntimeConfigSnapshot() [runtime_handlers.go:798]
  buildDeploymentRuntimeSnapshot() [deployment_lifecycle_handlers.go:59]
  mergeNBRConfigSnapshot() key list [deployment_lifecycle_handlers.go:104]
  applyDeploymentConfigSnapshot() [deployment_lifecycle_handlers.go:922]
- API: accept detection → write process_start_config to NBR snapshot
- API: manual edit of process_start_config via NBR PATCH
- Deployment creation freezes process_start_config into deployment snapshot
- Existing NBRs/Deployments: missing key → legacy behavior (unchanged)

Does NOT:
- Change RunPlan resolution
- Change Agent Docker Create
```

### Phase 3: RunPlan Execution

**Goal**: Resolver reads frozen process_start_config and produces correct Docker API spec.

```text
- Resolver reads process_start_config from frozen deployment snapshot
  (via applyDeploymentConfigSnapshot → preflight struct)
- image_default → Entrypoint = nil (Docker preserves image ENTRYPOINT)
- custom → Entrypoint = process_start_config.entrypoint
- finalCmd = process_start_config.command_prefix + buildArgs()
- missing process_start_config → legacy behavior unchanged
- Add resolver tests for vLLM/SGLang/llama.cpp with process_start_config
- command_preview shows:
  image_default: no --entrypoint, annotate "# preserves image ENTRYPOINT"
  custom: explicit --entrypoint flag

Does NOT:
- Change Agent Docker Create logic (nil entrypoint already handled correctly)
```

### Phase 4: API Workflow / E2E / Real Smoke

**Goal**: End-to-end validation with real Docker containers.

```text
- Update API-first shell E2E scripts:
  assert detection generation
  assert config acceptance flow
  assert RunPlan Entrypoint/Cmd
  assert command preview accuracy
- Real smoke for vLLM/SGLang/llama.cpp:
  capture image inspect Entrypoint/Cmd
  capture process_start_config
  capture RunPlan Entrypoint/Cmd
  capture Agent Docker create Entrypoint/Cmd
  verify /v1/models health check
- Backward compatibility smoke:
  existing deployment starts unchanged after code update
- NVIDIA/MetaX HostConfig regression guard:
  MetaX device passthrough unchanged after Layer 3 change

Does NOT:
- Add trial-run probe
- Change Web UI
```

### Phase 5: Optional Trial-Run Probe (DEFERRED)

**Goal**: User-triggered container startup validation.

```text
- NBR-level lightweight probe (deferred):
  docker run --rm <image> <command_prefix> --help
- Start Wizard full trial (deferred):
  temporary container with model mount + health check
- Guardrails:
  explicit user action only
  hard timeout
  guaranteed cleanup (--rm + ContainerRemove Force)
  no GPU lease holding
  evidence/logs collection
```

---

## 20. Acceptance Criteria (Draft)

Final acceptance criteria should be derived after Claude/code review. Candidate criteria:

1. Existing deployments without `process_start_config` are unchanged.
2. NBR can store a process start detection result and an accepted process start config.
3. Profiles are matched by backend family and image entrypoint/cmd characteristics, not official image repository names.
4. SGLang official/private/vendor images with wrapper entrypoint can be configured as:
   - preserve image entrypoint
   - `Cmd = python3 -m sglang.launch_server + model args`
5. vLLM can be configured as:
   - preserve image entrypoint
   - `Cmd = model args`
6. llama.cpp remains no-regression:
   - preserve image entrypoint
   - `Cmd = model args`
7. Docker/HostConfig/hardware settings remain unaffected.
8. MetaX/vendor device settings are not replaced or simplified into NVIDIA-only GPU logic.
9. Command preview shows exact Docker API Entrypoint/Cmd fields.
10. API tests cover detection, config application, RunPlan resolution, and Agent spec conversion.
11. Real smoke for vLLM/SGLang/llama.cpp records:
    - image
    - image inspect Entrypoint/Cmd
    - process_start_config
    - RunPlan Entrypoint/Cmd
    - Agent Docker create Entrypoint/Cmd
    - container logs
    - `/v1/models` response
12. No new half-implemented field is left in the codebase.

---

## 21. Open Questions for Claude Review

1. What is the cleanest storage location for BackendRuntime process start profiles in the current schema?
2. Can NBR `config_snapshot_json` top-level keys be modified and frozen cleanly without DB migration?
3. Should `process_start_detection` live inside `probe_results_json`, and does the current API expose it clearly?
4. Does existing NBR PATCH support applying detection to config, or is a small action endpoint needed?
5. How should the UI represent multiple candidate profiles and confidence/warnings?
6. Can the current command preview safely express Docker API `Entrypoint` with multiple tokens, or should exact Docker API fields be displayed separately?
7. Which existing E2E scripts should be updated first to avoid broad churn?
8. Is there a local/vendor image set available to test wrapper entrypoint behavior beyond official images?
9. Should image script static probe be implemented before or after trial-run probe?
10. Are there any security concerns with exposing command_prefix editing to tenant operators?

---

## 22. Review Instructions for Claude

When reviewing this document, please do **not** implement immediately.

Claude should produce:

1. A code-aware review of the proposed design.
2. A mapping of proposed concepts to existing files/functions/tables.
3. A recommended minimal storage approach.
4. A list of required code changes if implemented.
5. A test strategy.
6. E2E update recommendations.
7. Risks and rejected alternatives.
8. A proposed step-by-step implementation plan for human approval.
9. Acceptance criteria for each step.

Claude should explicitly call out if any assumption in this document conflicts with current code.

---

## 23. Summary

The key design change is not just to add another static startup field. The target is a controlled discovery and configuration workflow:

```text
BackendRuntime/catalog provides backend-family-level startup candidates.
Agent inspects the selected image.
LightAI generates process_start_detection with candidates, evidence, confidence, and warnings.
User accepts, customizes, manually overrides, or trial-runs.
NBR stores process_start_config as node-level truth.
Deployment freezes it.
RunPlan resolves exact Docker API Entrypoint/Cmd.
Agent executes it exactly.
```

This preserves existing Layer 1, Layer 2, and Layer 4 mechanisms, while making the missing Layer 3 explicit, inspectable, testable, and suitable for official, vendor, private, and custom images.
