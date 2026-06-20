# NBR Process Start Config — Implementation Plan

> Status: PLAN_DRAFT
> Implementation: not approved yet
> Code changes: none in this step
> Depends on: `nbr-process-start-config-design-draft.md`, `nbr-process-start-config-claude-review.md`, `docker-launch-parameter-chain-audit.md`

## 1. Goals

This plan provides a phased implementation path for Layer 3 Process Start Config in LightAI Go. The plan is designed to:

- Deliver one self-contained, testable phase at a time.
- Prioritize API-first validation over Web UI changes.
- Prioritize static detection (no container execution) over trial-run probes.
- Guarantee backward compatibility — existing NBRs and Deployments are never broken.
- Keep v1 minimal: no DB migration, no new API endpoints, no parameter schema redesign.
- Enable vLLM, SGLang, and llama.cpp to correctly express Docker ENTRYPOINT / CMD through `process_start_config`.

## 2. Non-Goals

This plan explicitly does NOT include:

```
- DB migration or new columns
- Parameter schema redesign
- command_template system
- gpu_mode implementation
- --gpus all as universal GPU design
- vLLM default_args_json bare path fix (Layer 4 — separate issue)
- ParameterDef.Value field gap fix (Layer 4 — separate issue)
- Web UI overhaul
- Trial-run probe in v1 (deferred to Phase 5)
- shell_mode=true execution in v1
- Browser E2E testing
- Long-duration real smoke in early phases
```

## 3. Design Constraints (Mandatory)

All phases must comply with these constraints derived from the design draft and code-aware review.

### 3.1 Four-Layer Model

```
Layer 1: Image — resolveImage() priority chain, unchanged
Layer 2: Docker / HostConfig / Hardware — docker_json, unchanged
Layer 3: Process Start / ENTRYPOINT / CMD — THIS PLAN
Layer 4: Model Service Params — buildArgs(), unchanged
```

### 3.2 NBR Is the Layer 3 Authority

```
BackendRuntime / catalog → candidate profiles (v1: Go constants)
NBR.probe_results_json → process_start_detection
NBR.config_snapshot_json → process_start_config (authoritative)
Deployment.config_snapshot_json → frozen copy
RunPlan → resolves from frozen config
```

### 3.3 Profiles v1: Go Constants, Not DB Columns

```
process_start_profiles in Go constants or YAML catalog files.
NOT in backend_runtimes table.
NOT in docker_json (Layer 2).
NOT in version_snapshot_json.
No DB migration in v1.
```

### 3.4 Profile Matching: backend_family, Not Image Repo Name

```
Primary match key: backend_family (vllm/sglang/llamacpp)
Secondary match: image Entrypoint/Cmd shape classification
Auxiliary: labels, env, exposed ports, probe evidence
image_ref: evidence/confidence signal only — NOT a match condition
```

### 3.5 Detection / Config / Profile Separation

```
process_start_profiles:  candidate list (Go constants / catalog)
process_start_detection: system suggestion (probe_results_json, regeneratable)
process_start_config:    user-confirmed authoritative config (config_snapshot_json)
Detection never silently overwrites config.
```

### 3.6 process_start_config v1 Field Scope

```json
{
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
}
```

v1 supports: `entrypoint_mode = image_default | custom`
v1 does NOT support: `clear`, `runtime_default` as explicit mode, `shell_mode=true` execution
Missing `process_start_config` → legacy behavior (BackendRuntime.entrypoint_override > BackendVersion.default_entrypoint)

### 3.7 command_prefix in CMD, Not ENTRYPOINT

```
modelArgs = buildArgs(...)  // Layer 4, existing mechanism
finalCmd  = process_start_config.command_prefix + modelArgs
command_prefix does NOT enter dedup, does NOT enter applyServiceArgs.
```

### 3.8 ENTRYPOINT Semantics

```
image_default:  RunPlan.Entrypoint = nil  → Docker preserves image ENTRYPOINT
custom:         RunPlan.Entrypoint = config.entrypoint  → Docker sets ENTRYPOINT
missing config: legacy behavior (entrypoint_override > default_entrypoint)
```

### 3.9 Layer 4 Issues Separate

```
vLLM default_args_json bare path → NOT fixed here
ParameterDef.Value gap → NOT fixed here
--model / --model-path / -m parameter logic → NOT changed here
```

### 3.10 Layer 2 Regression Guard

```
NVIDIA DeviceRequest + GPUDeviceIDs unchanged
CUDA_VISIBLE_DEVICES unchanged
MetaX /dev/dri, /dev/mxcd, /dev/infiniband unchanged
group_add, security_options, privileged, ipc, shm, ulimits, env unchanged
No --gpus all universalization
```

## 4. Phase Breakdown

### Phase 1: Static Profiles + Detection (READ-ONLY)

**Goal**: Generate `process_start_detection` without changing any container startup behavior.

**Boundary**: Detection only. No config writing. No RunPlan change. No Docker API change.

**Tasks**:

1. **Define Go types** in `internal/server/runplan/`:
   - `ProcessStartProfile` struct (id, backend_family, entrypoint_mode, entrypoint, command_prefix, priority, detection_hints, warnings)
   - `ProcessStartDetection` struct (status, selected_profile_id, entrypoint_mode, command_prefix, confidence, source, candidate_profiles, evidence, warnings)

2. **Define default profiles** as Go constants:
   - `internal/server/runplan/profiles.go`
   - vLLM: `{id:"vllm.image_default", backend_family:"vllm", entrypoint_mode:"image_default", command_prefix:[], priority:100}`
   - SGLang: `{id:"sglang.python_module_launcher", backend_family:"sglang", entrypoint_mode:"image_default", command_prefix:["python3","-m","sglang.launch_server"], priority:100}` + `{id:"sglang.custom_entrypoint", backend_family:"sglang", entrypoint_mode:"custom", entrypoint:["python3","-m","sglang.launch_server"], priority:40}`
   - llama.cpp: `{id:"llamacpp.image_default", backend_family:"llamacpp", entrypoint_mode:"image_default", command_prefix:[], priority:100}`

3. **Derive backend_family** from BackendRuntime → Backend → `inference_backends.name`:
   - Verify: `backend_runtimes.backend_id` → `inference_backends.id` → `inference_backends.name`

4. **Implement image Entrypoint/Cmd classification**:
   - `classifyEntrypointShape(entrypoint []string) string`
   - Shapes: `"empty"`, `"server_binary"`, `"wrapper_script"`, `"python_launcher"`, `"unknown"`
   - vLLM `["vllm","serve"]` → server_binary
   - SGLang `["/opt/nvidia/nvidia_entrypoint.sh"]` → wrapper_script
   - llama.cpp `["/app/llama-server"]` → server_binary

5. **Implement profile scoring**:
   - Score = backend_family match (base) + entrypoint shape compatibility (bonus) + avoid-overlap penalty
   - Select highest-scoring profile with confidence assessment

6. **Write detection to probe_results_json**:
   - Add `"process_start_detection"` key in `probeResults` map (`runtime_handlers.go:342-347`)
   - Populate during NBR check (`HandleRequestNodeBackendRuntimeCheck`)
   - Read image inspect data from existing level2 probe results

7. **Unit tests** (`internal/server/runplan/profiles_test.go`):
   - `TestProfilesForVLLM`: correct profile for vllm backend_family
   - `TestProfilesForSGLang`: two candidates, python_module_launcher has higher priority
   - `TestProfilesForLLamaCPP`: correct profile for llamacpp
   - `TestClassifyEntrypoint`: all shape classifications
   - `TestDetectionWithWrapperEntrypoint`: SGLang image → wrapper_script → python_module_launcher selected
   - `TestDetectionWithServerBinaryEntrypoint`: vLLM → server_binary → image_default selected
   - `TestDetectionUnknownEntrypoint`: unknown → low confidence + warnings

**Files involved**:

| File | Change | Why | Risk |
|------|--------|-----|------|
| `internal/server/runplan/profiles.go` | NEW — profile constants and types | Define profiles in code | Low |
| `internal/server/runplan/detection.go` | NEW — classification + scoring logic | Entrypoint shape classification | Low |
| `internal/server/api/runtime_handlers.go` | ADD key in probeResults map | Store detection in probe | Low |
| `internal/server/runplan/profiles_test.go` | NEW — unit tests | Profile correctness | Low |
| `internal/server/runplan/detection_test.go` | NEW — detection tests | Detection correctness | Low |

**Phase 1 does NOT change**: RunPlan, Agent Docker Create, container startup, NBR config_snapshot_json, Deployment snapshot.

---

### Phase 2: Config Acceptance + Snapshot Flow

**Goal**: Users can accept detection into NBR config. Config flows through snapshot chain into Deployment freeze.

**Boundary**: Data flow only. Still no RunPlan execution change.

**Tasks**:

1. **Add `process_start_config` to snapshot builders**:
   - `buildRuntimeConfigSnapshot()` (`runtime_handlers.go:798`): add `"process_start_config": rt["process_start_config_json"]`
   - `buildDeploymentRuntimeSnapshot()` (`deployment_lifecycle_handlers.go:59`): same capture

2. **Add `process_start_config` to merge key list**:
   - `mergeNBRConfigSnapshot()` (`deployment_lifecycle_handlers.go:104`): add `"process_start_config"` to the hardcoded key slice

3. **Add `process_start_config` to snapshot application**:
   - `applyDeploymentConfigSnapshot()` (`deployment_lifecycle_handlers.go:922`): extract `"process_start_config"` from snapshot, unmarshal into preflight struct field

4. **NBR config acceptance API**:
   - Read-merge-write pattern: read existing `config_snapshot_json`, merge `process_start_config`, PATCH entire snapshot
   - Or: dedicated handler that reads detection from `probe_results_json`, applies it to `config_snapshot_json` server-side
   - Accept detection → copy `process_start_detection` selected profile fields into `process_start_config` with `source: "user_accepted_detection"`

5. **Manual config edit**:
   - Allow PATCH of `config_snapshot_json` with custom `process_start_config`
   - Set `source: "user_override"`, `confidence: "user_confirmed"`

6. **Deployment snapshot freeze**:
   - Verify: `buildDeploymentRuntimeSnapshot` + `mergeNBRConfigSnapshot` → `process_start_config` present in deployment `config_snapshot_json`

7. **Unit tests**:
   - `TestSnapshotCapturesProcessStartConfig`: snapshot includes the key
   - `TestMergePropagatesProcessStartConfig`: merge carries the key
   - `TestDeploymentFreezesProcessStartConfig`: deployment snapshot contains config
   - `TestExistingNBRWithoutConfigUsesLegacy`: missing key → no effect
   - `TestInvalidConfigRejected`: invalid entrypoint_mode → clear error

**Files involved**:

| File | Change | Why | Risk |
|------|--------|-----|------|
| `internal/server/api/runtime_handlers.go` | Add key to `buildRuntimeConfigSnapshot` | Capture config in NBR snapshot | Low |
| `internal/server/api/deployment_lifecycle_handlers.go` | Add to merge key list + apply extraction + preflight field | Flow config into deployment snapshot | Medium |
| `internal/server/api/node_runtime_handlers.go` | Possibly add accept-detection helper | Apply detection → config | Low |

**Phase 2 does NOT change**: RunPlan resolution, Agent Docker Create, container startup.

---

### Phase 3: RunPlan Execution

**Goal**: Resolver reads frozen `process_start_config` and produces correct Docker API Entrypoint/Cmd.

**Boundary**: First phase that changes container startup behavior.

**Tasks**:

1. **Add `ProcessStartConfig` field to resolver input**:
   - `preflightResult` struct (`deployment_lifecycle_handlers.go`): add `processStartConfig *ProcessStartConfig`
   - Or pass through `ResolveInput.NodeRuntimeOverride` if pattern fits

2. **Implement entrypoint resolution with process_start_config**:
   - In `Resolve()` (`resolver.go:192`), after existing entrypoint resolution:
   ```
   if processStartConfig != nil {
       switch processStartConfig.EntrypointMode {
       case "image_default":
           entrypoint = nil
       case "custom":
           entrypoint = processStartConfig.Entrypoint
       }
   }
   // else: legacy behavior (existing entrypoint_override > default_entrypoint)
   ```

3. **Implement command_prefix prepend**:
   - After `buildArgs()` returns. `buildArgs()` internally completes Layer 4 parameter handling, dedup, and `applyServiceArgs` before returning. `command_prefix` is a Layer 3 concept and must be prepended AFTER `buildArgs()` returns — it does NOT enter Layer 4 dedup or `applyServiceArgs`.
   - Correct semantic:
     ```
     modelArgs = buildArgs(...)   // Layer 4: includes dedup + applyServiceArgs
     finalCmd  = process_start_config.command_prefix + modelArgs
     ```
   - Code insertion point (after `buildArgs` call at `resolver.go` line 199):
     ```
     if processStartConfig != nil && len(processStartConfig.CommandPrefix) > 0 {
         args = append(processStartConfig.CommandPrefix, args...)
     }
     ```

4. **Verify buildArgs boundary**:
   - Ensure `command_prefix` is added AFTER these steps

5. **Update command preview** (`preview.go`):
   - `image_default`: no `--entrypoint` flag, annotate `# preserves image ENTRYPOINT`
   - `custom`: show `--entrypoint` flag with joined entrypoint tokens

6. **Unit tests**:
   - `TestRunPlanImageDefaultNilEntrypoint`: Entrypoint is nil, Args include command_prefix + model args
   - `TestRunPlanCustomSetsEntrypoint`: Entrypoint is config.entrypoint
   - `TestRunPlanMissingConfigLegacyBehavior`: no process_start_config → current behavior
   - `TestRunPlanCommandPrefixPrepended`: command_prefix appears before model args
   - `TestRunPlanCommandPrefixNotInDedup`: command_prefix not affected by flag dedup
   - `TestCommandPreviewImageDefault`: preview omits --entrypoint
   - `TestCommandPreviewCustom`: preview shows --entrypoint flag

**Files involved**:

| File | Change | Why | Risk |
|------|--------|-----|------|
| `internal/server/runplan/resolver.go` | Entrypoint resolution + command_prefix prepend | Core behavior change | Medium |
| `internal/server/runplan/preview.go` | Command preview update | Accurate preview | Low |
| `internal/server/api/deployment_lifecycle_handlers.go` | Pass process_start_config into resolver | Data flow | Medium |

**Phase 3 changes**: RunPlan.Entrypoint can become nil for `image_default` mode. Agent Docker Create path handles nil correctly (verified in code review: `docker_real.go:82` — `len(nil) > 0` is false → `cfg.Entrypoint` unset → Docker preserves image ENTRYPOINT). No Agent code change needed.

---

### Phase 4: API Workflow / E2E / Real Smoke

**Goal**: End-to-end validation with real Docker containers.

**Boundary**: Validation only. No new features.

**Tasks**:

1. **Update API-first shell E2E scripts**:
   - NBR detection workflow: enable runtime → check → read `probe_results_json.process_start_detection`
   - Config acceptance workflow: accept detection → read `config_snapshot_json.process_start_config`
   - Deployment snapshot workflow: create deployment → verify frozen config
   - RunPlan assertions: verify Entrypoint nil for image_default, verify Cmd = command_prefix + args

2. **Real smoke — llama.cpp** (lowest risk, known working):
   - Enable NBR → check → verify detection
   - Accept detection → create deployment
   - Start → verify `/v1/models` returns 200
   - Capture: image inspect Entrypoint/Cmd, process_start_config, RunPlan Entrypoint/Cmd, container logs
   - Stop, cleanup

3. **Real smoke — vLLM** (if image available):
   - Same flow as llama.cpp
   - If `vllm/vllm-openai:v0.23.0` not available locally, use `:latest` and document tagged version gap
   - If image pull fails, block DOCUMENTED_BLOCKER with exact error

4. **Real smoke — SGLang** (if image available):
   - Same flow as llama.cpp
   - If `lmsysorg/sglang:v0.5.13.post1-cu129-runtime` not available locally, pull or document blocker
   - Verify wrapper entrypoint is preserved

5. **Backward compatibility smoke**:
   - Existing deployment (without process_start_config) starts unchanged after code update

6. **Layer 2 regression guard**:
   - MetaX runtime: all devices, group_add, security_options, privileged, ulimits present in docker_json
   - NVIDIA runtime: DeviceRequest present with correct GPU IDs

**Files involved**:

| File | Change | Why | Risk |
|------|--------|-----|------|
| `scripts/e2e/lib/model-runtime-common.sh` | Add detection/config assertions | E2E coverage | Low |
| `scripts/e2e-model-runtime-wizard-nvidia-api.sh` | Update for process_start_config | E2E coverage | Low |
| `docs/reports/phase-3/` | Real smoke evidence | Evidence capture | Low |

---

### Phase 5: Optional Trial-Run Probe (DEFERRED)

**Goal**: User-triggered container startup validation.

**Boundary**: Explicit user action only. Deferred — not a prerequisite for Phases 1-4.

**Tasks** (deferred):

1. **NBR-level lightweight probe**: `docker run --rm <image> <command_prefix> --help` — verifies launcher exists
2. **Start Wizard full trial**: Temporary container with model mount + health check + cleanup
3. **Guardrails**: Hard timeout, guaranteed cleanup, no GPU lease, no port collision, explicit user trigger only

**Files**: Deferred. Will be specified in a separate plan if approved.

---

## 5. File-Level Change Summary

| File | Phase | Change Type | Risk |
|------|-------|-------------|------|
| `internal/server/runplan/profiles.go` | 1 | NEW — Go constants | Low |
| `internal/server/runplan/detection.go` | 1 | NEW — classification + scoring | Low |
| `internal/server/api/runtime_handlers.go` | 1, 2 | ADD key in probe results + snapshot builder | Low |
| `internal/server/api/deployment_lifecycle_handlers.go` | 2, 3 | ADD merge key + apply extraction + preflight field + resolver input | Medium |
| `internal/server/api/node_runtime_handlers.go` | 2 | Possibly ADD accept-detection logic | Low |
| `internal/server/runplan/resolver.go` | 3 | MODIFY entrypoint resolution + command_prefix prepend | Medium |
| `internal/server/runplan/preview.go` | 3 | MODIFY command preview | Low |
| `scripts/e2e/lib/model-runtime-common.sh` | 4 | MODIFY E2E assertions | Low |
| `scripts/e2e-model-runtime-wizard-nvidia-api.sh` | 4 | MODIFY E2E workflow | Low |
| `internal/agent/runtime/docker.go` | — | NO CHANGE (nil entrypoint already handled) | None |
| `internal/agent/runtime/docker_real.go` | — | NO CHANGE | None |

---

## 6. Open Questions Before Implementation

1. **`backend_family` derivation**: Is `inference_backends.name` (vllm/sglang/llamacpp) the canonical `backend_family`? Or is there another field? Must verify before Phase 1.

2. **vLLM `v0.23.0` ENTRYPOINT**: Catalog default image not locally available. Must `docker pull` or document as blocker before finalizing vLLM profile.

3. **SGLang `v0.5.13.post1-cu129-runtime` ENTRYPOINT**: Same — not locally available. Must inspect before finalizing SGLang profile.

4. **Image entrypoint classification algorithm**: What is the exact rule for "wrapper_script" vs "server_binary" vs "unknown"? Needs concrete algorithm before Phase 1 implementation.

5. **`command_prefix` overlap detection**: If image ENTRYPOINT is `["python3"]` and `command_prefix` is `["python3", "-m", "sglang.launch_server"]`, should detection warn about `python3 python3` double-execution?

6. **NBR PATCH pattern**: Use existing all-or-nothing `config_snapshot_json` replacement, or add a server-side read-merge-write for atomic detection acceptance?

7. **Profiles storage**: Go constants in `runplan/profiles.go` or YAML catalog files in `configs/`? Constants are simpler for v1, YAML allows future user customization.

8. **Detection regeneration trigger**: Should `process_start_detection` be regenerated on every NBR check, only when image_ref changes, or only on explicit request?

9. **Config immutability after probe**: When `process_start_config` already exists, should re-probe update only detection and leave config untouched? (Design says yes — detection never overwrites config.)

10. **Real smoke images/models**: Which exact image tags, model files, and health-check timeouts should Phase 4 use?

---

## 7. Rollback Plan

If Phase 3 introduces issues after deployment:

```
1. Disable process_start_config reading in resolver:
   - Guard with feature flag or empty config check
   - All deployments fall back to legacy entrypoint behavior
2. Missing/invalid process_start_config → legacy behavior (already the default)
3. Existing deployments unaffected (snapshot freeze guarantees isolation)
4. Detection evidence preserved in probe_results_json (can re-evaluate later)
5. Restore old E2E expected outputs
6. Clean up any trial-run temporary containers
```

Rollback surface is minimal because:
- Each phase is additive (Phase 1 adds detection without changing startup, Phase 2 adds data flow, Phase 3 is the first to change behavior).
- Legacy behavior is the default when `process_start_config` is absent.
- Existing deployments are protected by snapshot freeze.

---

## 8. Commit Strategy (Future)

Recommended per-phase commits (NOT executed in this round):

```
Phase 1:
  feat(runplan): add process start profile constants
  feat(runplan): add entrypoint shape classification
  feat(api): write process_start_detection to NBR probe results
  test(runplan): add detection unit tests

Phase 2:
  feat(api): add process_start_config to NBR snapshot chain
  feat(api): add detection acceptance to NBR config
  test(api): add snapshot/config acceptance tests

Phase 3:
  feat(runplan): resolve entrypoint from process_start_config
  feat(runplan): prepend command_prefix to Cmd
  fix(preview): show entrypoint correctly in command preview
  test(runplan): add process_start_config resolution tests

Phase 4:
  test(e2e): add process_start_config E2E assertions
  docs(e2e): record real smoke evidence
```

Each commit must be: readable, testable, revertible, and free of unrelated changes.

---

## 9. Final State

```
PLAN_DRAFT_READY_FOR_REVIEW
```

Implementation may start only after this plan and the acceptance criteria are explicitly reviewed and approved.
