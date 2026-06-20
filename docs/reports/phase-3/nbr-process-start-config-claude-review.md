# Claude Code-Aware Review: NBR Process Start Config Design Draft

> Status: REVIEW
> Implementation: none
> Date: 2026-06-21
> Reviewed document: `docs/reports/phase-3/nbr-process-start-config-design-draft.md`
> Reference: `docs/reports/phase-3/docker-launch-parameter-chain-audit.md`

## 1. Overall Assessment

**方向认可 (Direction Approved).** The design draft's core architecture — four-layer model, NBR as Layer 3 authority, detection/config separation, profile-based candidate matching by backend_family, and "find at least one workable startup" rather than "prove a unique correct answer" — is well-motivated and consistent with the current codebase.

The draft correctly identifies the primary code gap: Layer 3 entrypoint behavior is currently implicit in `BackendVersion.default_entrypoint_json` / `BackendRuntime.entrypoint_override_json` with no explicit semantics for "preserve image ENTRYPOINT" vs "override" vs "clear."

**Recommendation: Move to Phase 1 implementation (static detection model) after resolving the open questions listed in §17 below.**

---

## 2. Must-Fix Design Points

### 2.1 BackendRuntime `process_start_profiles` Storage Requires a New Column or Compromise

**Issue**: The design draft proposes `process_start_profiles` on BackendRuntime, but the `backend_runtimes` table has no generic extensible JSON column (`docker_json` has a typed Go struct, `version_snapshot_json` is semantically for backend-version snapshots).

**Code evidence** (`internal/server/models/runtime.go:5-28`):
```go
type BackendRuntime struct {
    DockerJSON              string `json:"docker_json"`              // typed struct (DockerSpecInfo)
    EntrypointOverrideJSON  string `json:"entrypoint_override_json"` // string array
    VersionSnapshotJSON     string `json:"version_snapshot_json"`    // opaque blob
    // ... no generic "extra_json" or "catalog_json" field
}
```

**Options ranked**:

| Option | Migration? | Cleanliness | Recommendation |
|--------|-----------|-------------|---------------|
| A. New `process_start_profiles_json TEXT` column on `backend_runtimes` | Yes (V25) | Best — dedicated purpose | **Recommended for final design** but breaks the "no migration" constraint this round |
| B. Store profiles in Go constant / YAML catalog file, not in DB | No | Clean — profiles are configuration, not state | **Recommended for v1 implementation** — avoids migration entirely |
| C. Store inside `docker_json` | No | Poor — mixes Layer 2 and Layer 3 | Rejected per design principles |
| D. Store inside existing `version_snapshot_json` | No | Poor — semantically wrong column | Rejected |

**Recommendation**: For v1, define profiles as Go constants (e.g., `defaultProcessStartProfiles` in a new file `internal/server/runplan/profiles.go`) or YAML catalog files. This avoids DB migration, keeps Layer 3 separate, and allows profiles to be versioned in code. If runtime DB persistence is needed later, add a dedicated column in a future migration.

### 2.2 `mergeNBRConfigSnapshot` Needs Explicit Key Addition

**Issue**: The draft says `process_start_config` should flow from NBR → Deployment snapshot, but the merge function uses a hardcoded key list.

**Code evidence** (`deployment_lifecycle_handlers.go:104-108`):
```go
for _, key := range []string{
    "vendor", "image_name", "image_pull_policy",
    "entrypoint_override_json", "args_override_json", "default_env_json",
    "docker_json", "model_mount_json", "health_check_override_json",
} { ... }
```

`"process_start_config"` **must** be added to this slice for the config to flow into deployments. This is a one-line change but must not be forgotten.

### 2.3 `applyDeploymentConfigSnapshot` Needs Explicit Field Extraction

**Issue**: The snapshot application function uses individual `snap["key"]` lookups with type assertions. A new key needs explicit handling.

**Code evidence** (`deployment_lifecycle_handlers.go:922-986`): Every known key is extracted explicitly — there is no generic pass-through. A new field on `preflightResult` (e.g., `pf.processStartConfigJSON string`) and explicit extraction logic are required.

---

## 3. Design Points Acceptable But Requiring Confirmation

### 3.1 `entrypoint_mode` Values: `image_default` | `custom` (Two, Not Three)

The draft correctly limits to two modes for v1. The `nil` vs `[]string{}` distinction is handled by the Docker API `len(opts.Entrypoint) > 0` check — both nil and empty produce the same result (preserve image ENTRYPOINT). No "clear" mode is needed.

**Verified in code**: `docker_real.go:82`: `if len(opts.Entrypoint) > 0 { cfg.Entrypoint = ... }` — nil and `[]` are equivalent.

### 3.2 `command_prefix` in CMD, NOT in ENTRYPOINT

The draft correctly places `command_prefix` in `Config.Cmd` (prepended to model args), not in `Config.Entrypoint`. This matches the external `docker run IMAGE python3 -m sglang.launch_server <args>` pattern.

### 3.3 `process_start_detection` in `probe_results_json` — Verified Possible

**Code evidence** (`runtime_handlers.go:342-347`): `probeResults` is a `map[string]interface{}` with existing level1-4 keys. Adding `"process_start_detection"` as a new top-level key is a one-line addition with no schema change. The probe results are written to `probe_results_json TEXT` column — any valid JSON is accepted.

### 3.4 `process_start_config` in `config_snapshot_json` — Verified Possible

**Code evidence** (`runtime_handlers.go:798-818`): `buildRuntimeConfigSnapshot()` constructs a flat `map[string]interface{}`. Adding `"process_start_config": rt["process_start_config_json"]` is a one-line addition with no schema change. The column is `TEXT NOT NULL DEFAULT '{}'` — any valid JSON is accepted.

**However**, the config must also be captured by `buildDeploymentRuntimeSnapshot` (`deployment_lifecycle_handlers.go:59`) and flowed through the merge/apply chain (see §2.2, §2.3 above).

### 3.5 `buildRuntimeConfigSnapshot` Source Question

The snapshot captures fields from `rt map[string]interface{}` which is the BackendRuntime row. If `process_start_config` is first defined as a Go constant / catalog file profile, it won't exist in the BackendRuntime DB row. The snapshot builder would need a different source (the profile selected during detection → config acceptance). This is a data flow design question for implementation, not a blocker.

---

## 4. High-Risk or Objectionable Design Points

### 4.1 Profile Matching by `image_ref` Patterns — Addressed

The draft explicitly rejects hardcoded image repo name matching (e.g., `lmsysorg/sglang:*`). Section 7.3 gives a clear "bad rule" vs "better rule" example. This is correct and essential for vendor/private/custom image support.

### 4.2 `shell_mode` Deferral — Appropriate

Shell mode introduces string escaping risks and complicates Docker API semantics. Deferring it is correct. The field can remain in the struct as `false` by default with no implementation.

### 4.3 Trial-Run Probe Safety — Needs Explicit Guardrails

The draft's Level 4 trial-run probe (NBR-level lightweight + Start Wizard full) is well-scoped. However, the implementation must ensure:

1. **Cleanup guarantee**: Temporary containers must be removed even on panic/signal. Docker `--rm` flag or explicit `ContainerRemove` with `Force: true` is required.
2. **Timeout**: Trial containers must have a hard timeout (not indefinite).
3. **Resource isolation**: Trial containers should not hold GPU leases or affect production containers.
4. **User trigger only**: No automatic trial-run without explicit user action.

These guardrails should be added to the design document before implementation.

### 4.4 NBR PATCH Limitation — All-or-Nothing Snapshot Replacement

**Code evidence** (`node_runtime_handlers.go:135-143`): PATCH replaces the entire `config_snapshot_json` column. Individual fields within the snapshot cannot be patched atomically. If a user wants to update only `process_start_config` within the snapshot, they must either:
1. Send the entire snapshot (client-side read-merge-write), or
2. The server must implement a read-merge-write endpoint.

This is a UX concern for Web UI but acceptable for API-first implementation.

---

## 5. Inconsistencies With Current Code

### 5.1 `ResolvedRunPlan.Entrypoint` nil vs empty — No Inconsistency Found

**Verified**: nil entrypoint propagates correctly through the entire chain:
- `resolver.go:240` → `Entrypoint: entrypoint` (nil OK)
- `types.go:8` → `json:"entrypoint,omitempty"` (nil omitted from JSON)
- `deployment_lifecycle_handlers.go:1121` → `"command": pf.plan.Entrypoint` (nil → JSON null)
- Agent unmarshal → nil `DockerSpec.Command`
- `docker.go:420` → `Entrypoint: spec.Docker.Command` (nil → nil)
- `docker_real.go:82` → `len(nil) > 0` is false → `cfg.Entrypoint` unset → Docker preserves image ENTRYPOINT

### 5.2 `EquivalentCommandPreview` Nil Safety — No Inconsistency Found

**Verified** (`preview.go:68`): `append(parts, plan.Entrypoint...)` with nil slice is safe in Go — variadic expansion of nil produces zero arguments.

### 5.3 GPU DeviceRequest Independent of Entrypoint — No Coupling

**Verified** (`docker.go:457`): `if spec.Vendor == "nvidia" && len(spec.GPUDeviceIDs) > 0` — purely vendor-based, no dependency on entrypoint.

---

## 6. Recommended Data Landing Points

| Data | Location | DB Migration? | Key Addition Needed? |
|------|----------|---------------|---------------------|
| `process_start_profiles` (v1) | Go constant or YAML catalog file | **No** | No — profiles are code, not state |
| `process_start_profiles` (future) | `backend_runtimes.process_start_profiles_json` | **Yes (V25)** | New column |
| `process_start_detection` | `NBR.probe_results_json.process_start_detection` | **No** | One-line key addition in probe handler |
| `process_start_config` | `NBR.config_snapshot_json.process_start_config` | **No** | Key in `buildRuntimeConfigSnapshot` + `mergeNBRConfigSnapshot` key list + `applyDeploymentConfigSnapshot` handler |

---

## 7. Recommended Minimal Field Structure

### process_start_config (stored in NBR config_snapshot_json)

```json
{
  "process_start_config": {
    "entrypoint_mode": "image_default",
    "entrypoint": [],
    "command_prefix": [],
    "shell_mode": false,
    "profile_id": "vllm.image_default",
    "source": "user_accepted_detection",
    "confidence": "high",
    "warnings": []
  }
}
```

### process_start_detection (stored in NBR probe_results_json)

```json
{
  "process_start_detection": {
    "status": "candidate_found",
    "selected_profile_id": "vllm.image_default",
    "entrypoint_mode": "image_default",
    "command_prefix": [],
    "confidence": "high",
    "candidates": [...],
    "evidence": {...},
    "warnings": []
  }
}
```

### process_start_profiles (v1: Go constant)

```go
// internal/server/runplan/profiles.go
var DefaultProcessStartProfiles = map[string][]ProcessStartProfile{
    "vllm": {
        {ID: "vllm.image_default", EntrypointMode: "image_default", CommandPrefix: nil, Priority: 100},
    },
    "sglang": {
        {ID: "sglang.python_module_launcher", EntrypointMode: "image_default", CommandPrefix: []string{"python3", "-m", "sglang.launch_server"}, Priority: 100},
        {ID: "sglang.custom_entrypoint", EntrypointMode: "custom", Entrypoint: []string{"python3", "-m", "sglang.launch_server"}, Priority: 40},
    },
    "llamacpp": {
        {ID: "llamacpp.image_default", EntrypointMode: "image_default", CommandPrefix: nil, Priority: 100},
    },
}
```

This avoids DB migration, keeps Layer 3 separate, and allows profiles to be versioned in code. Profiles map to `backend_family` (vllm/sglang/llamacpp), not image repo names.

---

## 8. Detection / Config / Profile Relationship

```
┌─────────────────────────────────────────────────────────┐
│ process_start_profiles (BackendRuntime / catalog)       │
│   Static candidates for each backend_family            │
│   Defined in Go code or YAML catalog                   │
│   NOT in DB (v1); DB column optional for future        │
├─────────────────────────────────────────────────────────┤
│                         │                               │
│                         ▼                               │
│ process_start_detection (NBR.probe_results_json)       │
│   System-generated suggestion                          │
│   Inputs: profiles + image inspect + backend_family    │
│   Can be regenerated on re-probe                       │
│   Does NOT auto-overwrite process_start_config         │
├─────────────────────────────────────────────────────────┤
│                         │                               │
│                    [User Accepts]                       │
│                         │                               │
│                         ▼                               │
│ process_start_config (NBR.config_snapshot_json)        │
│   Authoritative Layer 3 configuration                  │
│   Frozen into Deployment snapshot on create            │
│   Source tracks provenance (detection/user_override)   │
├─────────────────────────────────────────────────────────┤
│                         │                               │
│                         ▼ (at start/dry-run)            │
│ RunPlan resolves:                                      │
│   if process_start_config missing → legacy behavior    │
│   if entrypoint_mode=image_default → Entrypoint=nil    │
│   if entrypoint_mode=custom → Entrypoint=config value  │
│   Cmd = command_prefix + buildArgs()                   │
├─────────────────────────────────────────────────────────┤
│                         │                               │
│                         ▼                               │
│ Agent Docker Create:                                   │
│   cfg.Entrypoint = resolved Entrypoint (or nil)        │
│   cfg.Cmd = resolved Cmd                               │
│   Docker preserves image ENTRYPOINT when nil           │
└─────────────────────────────────────────────────────────┘
```

**Key invariant**: detection never silently overwrites config. Source tracking distinguishes auto-detected from user-overridden.

---

## 9. Auto-Detection / Trial-Run Recommendations

### 9.1 Level 1-2 (Static Image Inspect + Profile Matching) — Implement First

These are low-risk, no-container-execution operations. They use existing `docker image inspect` data already stored in `probe_results_json.level2`. Implementation can be purely server-side: read `entrypoint`, `cmd`, `backend_family` → score profiles → produce `process_start_detection`.

### 9.2 Level 3 (Static Script Probe) — Defer

`docker create` + `docker cp` + `docker rm` requires Agent-side orchestration that doesn't exist today. Defer until Level 1-2 detection proves insufficient for real images.

### 9.3 Level 4 (Trial-Run Probe) — Defer With Guardrails

Trial-run is the strongest evidence but introduces:
- Resource consumption (GPU, ports, disk)
- Cleanup risk (orphaned containers)
- Time cost (model loading can take minutes)

Implement only after static detection is complete, with explicit user trigger, timeout, and cleanup guarantees.

---

## 10. RunPlan / Agent Docker Create Semantics

### 10.1 Verified Correct

The `nil` → omit → Docker-preserve chain works end-to-end (verified in §5.1). No code changes needed in the Agent Docker Create path beyond the resolver producing the correct `Entrypoint` value.

### 10.2 Recommended RunPlan Output

| Field | Source | When `image_default` | When `custom` |
|-------|--------|---------------------|---------------|
| `Image` | `resolveImage()` | Unchanged | Unchanged |
| `Entrypoint` | `process_start_config` | `nil` (omitted from JSON) | `entrypoint` value |
| `Args`/`Cmd` | `command_prefix + buildArgs()` | Prepended + model args | Prepended + model args |
| `Env` | 5-layer merge | Unchanged | Unchanged |
| All Layer 2 fields | `docker_json` | Unchanged | Unchanged |

### 10.3 `command_prefix` Placement in RunPlan Args

The current `ResolvedRunPlan.Args` is consumed by the Agent as `Config.Cmd`. Prepending `command_prefix` means the args array becomes `command_prefix + buildArgs()`.

**Implementation note**: The resolver currently builds args at line 199 (`buildArgs(in, vars)`). The `command_prefix` prepend should happen after `buildArgs()` returns, before dedup (which handles flag-value pairs — command_prefix items are not flags and won't be affected):

```go
args, argErrs := buildArgs(in, vars)
if processStartConfig != nil && len(processStartConfig.CommandPrefix) > 0 {
    args = append(processStartConfig.CommandPrefix, args...)
}
```

### 10.4 No `clear` Mode — Confirmed

The Docker API distinction between nil (unset ENTRYPOINT) and `[]` (explicit empty override) is not currently needed by any known image. The `len(opts.Entrypoint) > 0` check treats both identically. Adding `clear` later would require changing this check to distinguish nil from `[]`, adding complexity without proven need.

---

## 11. vLLM / SGLang / llama.cpp Target Behavior Review

### 11.1 vLLM: Preserve Image ENTRYPOINT — CONFIRMED

- `vllm/vllm-openai:latest` image inspect: `Entrypoint: ["vllm","serve"]`, `Cmd: null`
- Target: `entrypoint_mode: "image_default"`, `command_prefix: []`
- RunPlan: `Entrypoint=nil`, `Cmd=buildArgs()` → Docker runs `vllm serve <model_args>`
- Matches external baseline: `docker run ... vllm/vllm-openai:latest --model /path --host 0.0.0.0 --port 8000`
- **Note**: `vllm/vllm-openai:v0.23.0` is NOT locally available — its ENTRYPOINT is unverified. The catalog default may differ from `:latest`.

### 11.2 SGLang: Preserve Image ENTRYPOINT + command_prefix — CONFIRMED

- `lmsysorg/sglang:latest` image inspect: `Entrypoint: ["/opt/nvidia/nvidia_entrypoint.sh"]`, `Cmd: null`
- Target: `entrypoint_mode: "image_default"`, `command_prefix: ["python3","-m","sglang.launch_server"]`
- RunPlan: `Entrypoint=nil`, `Cmd=["python3","-m","sglang.launch_server"] + buildArgs()`
- Docker runs: `/opt/nvidia/nvidia_entrypoint.sh python3 -m sglang.launch_server <model_args>`
- Matches external baseline: `docker run ... lmsysorg/sglang:latest python3 -m sglang.launch_server <args>`
- **Risk**: `lmsysorg/sglang:v0.5.13.post1-cu129-runtime` (catalog default) is NOT locally available — its ENTRYPOINT may differ from `:latest`.
- **Double-execution risk**: If the image ENTRYPOINT already starts the SGLang server (not a wrapper), prepending `command_prefix` would cause the server to start twice. Profile scoring should detect overlapping entrypoint/command_prefix patterns.

### 11.3 llama.cpp: Preserve Image ENTRYPOINT — CONFIRMED

- `ghcr.io/ggml-org/llama.cpp:server-cuda13` image inspect: `Entrypoint: ["/app/llama-server"]`, `Cmd: null`
- Target: `entrypoint_mode: "image_default"`, `command_prefix: []`
- RunPlan: `Entrypoint=nil`, `Cmd=buildArgs()` → Docker runs `/app/llama-server <model_args>`
- Matches current successful behavior exactly. **No regression risk.**

---

## 12. NVIDIA / MetaX / 国产卡 Impact Review

### 12.1 Layer 3 Is Vendor-Independent

The `process_start_config` (entrypoint_mode, command_prefix) is identical for a vLLM NVIDIA container and a vLLM MetaX container. The vendor differences are exclusively in Layer 2 `docker_json` (devices, group_add, security_options, privileged, ulimits, GPU visibility key).

### 12.2 Verified: Entrypoint Change Does Not Affect GPU DeviceRequest

**Code evidence** (`docker.go:457`): NVIDIA `DeviceRequest` is controlled by `spec.Vendor == "nvidia"` and GPU IDs — no dependency on entrypoint.

### 12.3 Verified: Entrypoint Change Does Not Affect Raw Device Passthrough

**Code evidence** (`docker.go:476-486`): Device mappings from `spec.Devices` are applied independently of entrypoint.

### 12.4 Risk: None

Setting `Entrypoint=nil` for "image_default" has zero effect on HostConfig fields. The MetaX runtime's full device passthrough chain (`/dev/dri`, `/dev/mxcd`, `/dev/infiniband`, `privileged:true`, `group_add:["video"]`, `security_options`, `ulimits`) remains unchanged.

---

## 13. Backward Compatibility

### 13.1 Missing `process_start_config` → Legacy Behavior

The resolver's existing entrypoint logic (`BackendRuntime.entrypoint_override > BackendVersion.default_entrypoint`) is unchanged when `process_start_config` is absent. All existing NBRs and Deployments continue exactly as today.

### 13.2 New NBRs With Config → New Behavior

NBRs created after the feature is deployed can have `process_start_config` in their snapshot. The resolver applies it when present.

### 13.3 Snapshot Freeze Guarantee

Existing Deployments won't change because their `config_snapshot_json` is frozen at create time. Even if the NBR is later updated with `process_start_config`, existing deployments continue using their frozen snapshot (which lacks the key → legacy behavior).

---

## 14. Existing NBR / Deployment Impact

| Scenario | Behavior |
|----------|----------|
| Existing NBR, no process_start_config, existing Deployment | Unchanged — Legacy entrypoint logic |
| Existing NBR, user applies detection → config added | NBR updated; existing Deployments unchanged (frozen snapshot); new Deployments use new config |
| New NBR from catalog (with profiles) | Detection runs; user accepts → config stored; Deployments freeze it |
| NBR re-checked (new probe) | Detection regenerated in probe_results_json; config unchanged unless user accepts |

---

## 15. Recommended Implementation Phases

### Phase 0: Confirm Code Paths (1 day)
- Verify exact field/handler locations (this review has done most of this)
- Confirm `backend_family` mapping from BackendRuntime to Backend

### Phase 1: Static Profiles + Detection (minimal, no container execution)
1. Add `ProcessStartConfig` and related Go types to `runplan/` package
2. Define default profiles as Go constants (`runplan/profiles.go`)
3. Add `process_start_detection` to NBR probe results (Level 5)
4. Implement profile scoring based on backend_family + image inspect entrypoint/cmd shape
5. Store detection in `probe_results_json.process_start_detection`

### Phase 2: Config Acceptance (NBR update path)
1. Add `process_start_config` to `buildRuntimeConfigSnapshot`
2. Add key to `mergeNBRConfigSnapshot` hardcoded list
3. Add field extraction in `applyDeploymentConfigSnapshot`
4. Allow PATCH to set `process_start_config` in NBR snapshot
5. Config frozen into Deployment snapshot on create

### Phase 3: RunPlan + Agent Docker Create (the actual change)
1. Resolver reads `process_start_config` from frozen snapshot
2. `image_default` → `Entrypoint = nil`
3. `custom` → `Entrypoint = config.entrypoint`
4. `Cmd = command_prefix + buildArgs()`
5. Missing config → legacy behavior
6. Add resolver tests for all three backends

### Phase 4: Preview + E2E Update
1. Command preview shows exact Docker API fields
2. `image_default`: no `--entrypoint`, annotate "# preserves image ENTRYPOINT"
3. `custom`: show `--entrypoint` flag
4. Update API-first E2E scripts to assert detection and config application
5. Real smoke for vLLM/SGLang/llama.cpp with new config

### Phase 5: Optional Trial-Run Probe (deferred)
1. NBR-level lightweight `--help` probe
2. Start Wizard full temporary container validation
3. Cleanup guarantees

---

## 16. Test & E2E Recommendations Per Phase

### Phase 1 Tests
- `TestProcessStartDetection_VLLM`: Static detection produces `image_default` + empty prefix
- `TestProcessStartDetection_SGLang`: Detection with wrapper entrypoint → `image_default` + python launcher
- `TestProcessStartDetection_LLamaCPP`: Detection with server binary entrypoint → `image_default` + empty prefix
- `TestProcessStartDetection_MissingImage`: Missing image inspect → low confidence + warnings

### Phase 2 Tests
- `TestNBRConfigSnapshot_IncludesProcessStartConfig`: Snapshot captures the key
- `TestMergeNBRConfigSnapshot_IncludesProcessStartConfig`: Merge propagates the key
- `TestDeploymentSnapshot_FreezesProcessStartConfig`: Deployment captures frozen config
- `TestExistingNBR_NoProcessStartConfig_Unchanged`: Missing key → legacy behavior

### Phase 3 Tests
- `TestRunPlan_ImageDefault_NilEntrypoint`: `entrypoint_mode=image_default` → Entrypoint=nil
- `TestRunPlan_Custom_SetsEntrypoint`: `entrypoint_mode=custom` → Entrypoint=config value
- `TestRunPlan_CommandPrefix`: command_prefix prepended to args
- `TestRunPlan_MissingConfig_LegacyBehavior`: No config → current behavior unchanged

### Phase 4 Tests
- `TestCommandPreview_ImageDefault_NoEntrypointFlag`: Preview omits `--entrypoint`
- `TestCommandPreview_Custom_ShowsEntrypointFlag`: Preview shows `--entrypoint`
- `TestAgentDockerCreate_ImageDefault_PreservesImageEntrypoint`: nil → Docker preserves
- Existing API workflow tests: `TestWorkflowDeployment`, `TestWorkflowLifecycle` — must continue passing

### E2E Tests (Phase 4+)
- `vllm-nvidia-process-start`: Full wizard flow with detection → config → RunPlan → health check
- `sglang-nvidia-process-start`: Full wizard flow with wrapper entrypoint handling
- `llamacpp-nvidia-process-start`: No regression from current behavior
- `existing-deployment-backward-compat`: Old deployment starts unchanged after code update
- `metax-vllm-process-start`: MetaX HostConfig unchanged after Layer 3 change (regression guard)

---

## 17. Pre-Implementation Questions Requiring Answers

1. **SGLang `v0.5.13.post1-cu129-runtime` ENTRYPOINT**: Must `docker pull` and inspect before finalizing SGLang profile. The catalog default image is not locally available.

2. **vLLM `v0.23.0` ENTRYPOINT**: Same — not available locally. Catalog default may differ from `:latest`.

3. **`backend_family` mapping**: How is `backend_family` derived from BackendRuntime? The BackendRuntime has `backend_id` (FK to `inference_backends`) and `vendor`. The `inference_backends` table has `name` (e.g., "vllm", "sglang", "llamacpp", "ollama"). Is `inference_backends.name` the canonical `backend_family`? **Need to confirm.**

4. **Image ENTRYPOINT wrapper detection**: How does static detection classify an image entrypoint as "wrapper" vs "backend_server_binary" vs "unknown_binary"? The classification logic needs a concrete algorithm.

5. **`command_prefix` overlap detection**: If image ENTRYPOINT is `["python3"]` and `command_prefix` is `["python3", "-m", "sglang.launch_server"]`, Docker would run `python3 python3 -m sglang.launch_server`. Should detection detect and warn about this? What's the overlap check?

6. **NBR PATCH UX for individual snapshot fields**: Client-side read-merge-write for `config_snapshot_json` is fragile. Should a dedicated endpoint be added for applying detection → config?

---

## 18. Out of Scope (Confirmed)

| Item | Reason |
|------|--------|
| `parameter_schema` redesign | Layer 4; existing mechanism works |
| `command_template` system | Not needed; `buildArgs()` + `command_prefix` sufficient |
| `gpu_mode` (all vs specific) | Layer 2; separate design needed |
| `--gpus all` as universal solution | Rejected; vendor-specific devices must remain |
| `shell_mode` implementation | Deferred until real image requires it |
| `entrypoint_mode: "clear"` | Not proven needed |
| DB migration for new columns | Avoided in v1 (profiles in code, config in existing TEXT fields) |
| New REST API endpoints | Not needed; existing PATCH + probe endpoints sufficient |
| Web UI overhaul | Deferred; API-first validation first |
| Silent auto-overwrite of NBR config | Explicitly rejected by design principle #5 |
| Full real smoke during NBR creation | Too heavy; trial-run is explicit user action |

---

## 19. Specific Document Modification Suggestions

1. **§7.2 (Profiles)**: Add a concrete Go struct definition alongside the JSON example. This grounds the conceptual design in implementable code.

2. **§8 (Detection Flow)**: Add the concrete entrypoint classification algorithm. The draft says "classify image Entrypoint/Cmd shape" but doesn't define the shapes. Suggest:

   ```
   "empty"            → len(entrypoint)==0
   "server_binary"    → single binary path like ["/app/llama-server"] or ["vllm","serve"]
   "wrapper_script"   → shell script path like ["/opt/nvidia/nvidia_entrypoint.sh"]
   "python_launcher"  → starts with ["python3"] or ["python"]
   "unknown"          → fallback
   ```

3. **§6.2 (Layer 2)**: Add explicit note that `docker_json` is a typed Go struct (`DockerSpecInfo` in `resolver.go:94-105`), not a free-form JSON blob. This is a common point of confusion.

4. **§11 (NBR Storage)**: Option A (top-level key) should be the **only** recommendation. The alternative (nesting in `docker_json`) was rejected in the previous design round. Remove or clearly mark it as rejected.

5. **§14 (UI)**: Add note that UI is deferred. The API behavior should be designed and tested first via shell E2E scripts.

---

## 20. Final Recommendation

**Continue to implementation — Phase 1 (Static Detection Model) can begin.**

The design direction is solid. The four-layer model maps cleanly to existing code. The `process_start_config` placement as a top-level key in `config_snapshot_json` requires no DB migration. The nil entrypoint → Docker-preserve chain is verified correct end-to-end.

The single highest-value next step is **Phase 1**: define profiles as Go constants, implement static detection based on backend_family + image inspect, store detection in `probe_results_json`, and add resolver tests. This validates the detection model without changing any container startup behavior.

**Blockers before Phase 3** (the actual RunPlan change):
- Answers to questions in §17 (especially items 1-4)
- SGLang `v0.5.13.post1-cu129-runtime` image inspect
- vLLM `v0.23.0` image inspect
- `backend_family` derivation confirmed in code
