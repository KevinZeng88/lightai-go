# NBR Process Start Config — Acceptance Criteria

> Status: ACCEPTANCE_CRITERIA_DRAFT
> Implementation: not approved yet
> Depends on: `nbr-process-start-config-implementation-plan.md`, `nbr-process-start-config-design-draft.md`

## 1. Global Acceptance Criteria

These apply across all phases. Every phase gate must re-verify them.

```
G-001  No DB migration added in v1. All new data in existing TEXT columns.
G-002  Existing NBR without process_start_config uses legacy entrypoint behavior.
G-003  Existing Deployment snapshot unchanged after code update.
G-004  process_start_detection never silently overwrites process_start_config.
G-005  process_start_config is frozen into new Deployment snapshot on create.
G-006  image_default produces nil Entrypoint → Docker preserves image ENTRYPOINT.
G-007  custom produces explicit Docker Config.Entrypoint.
G-008  Final Cmd = command_prefix + buildArgs() result.
G-009  command_prefix does not enter Layer 4 dedup or applyServiceArgs.
G-010  NVIDIA DeviceRequest + GPUDeviceIDs unchanged after Layer 3 change.
G-011  MetaX /dev/dri, /dev/mxcd, /dev/infiniband unchanged after Layer 3 change.
G-012  group_add, security_options, privileged, ipc, shm, ulimits unchanged.
G-013  vLLM target behavior: image_default, empty command_prefix, model args in Cmd.
G-014  SGLang target behavior: image_default, python launcher prefix, model args in Cmd.
G-015  llama.cpp target behavior: image_default, empty command_prefix, model args in Cmd.
G-016  Manual override works: user can set custom entrypoint_mode + entrypoint.
G-017  Invalid process_start_config produces clear error, not silent fallback.
G-018  No i18n key leaks introduced in new log/error messages (if any text is added).
G-019  Logs and evidence include operation_id where applicable.
G-020  `go test ./...` passes in every phase.
G-021  `go vet ./...` passes in every phase.
G-022  `go build ./...` passes in every phase.
```

## 2. Phase 1 Acceptance Criteria: Static Profiles + Detection

```
P1-001  ProcessStartProfile and ProcessStartDetection Go types are defined.
P1-002  Default profiles exist for vllm, sglang, llamacpp backend families.
P1-003  Profiles are matched by backend_family — no image repo name hardcoding.
P1-004  vLLM detection output:
          backend_family: vllm
          selected profile: vllm.image_default
          entrypoint_mode: image_default
          command_prefix: []
          confidence: high (when image entrypoint known)
P1-005  SGLang detection output:
          backend_family: sglang
          selected profile: sglang.python_module_launcher (when wrapper entrypoint)
          entrypoint_mode: image_default
          command_prefix: ["python3","-m","sglang.launch_server"]
          confidence: high (when image entrypoint is wrapper-like)
P1-006  llama.cpp detection output:
          backend_family: llamacpp
          selected profile: llamacpp.image_default
          entrypoint_mode: image_default
          command_prefix: []
          confidence: high (when image entrypoint known)
P1-007  Detection includes candidate list with score, confidence, reasons, warnings.
P1-008  Detection is written to NBR.probe_results_json.process_start_detection.
P1-009  Detection uses existing level2 image inspect entrypoint/cmd data.
P1-010  Missing image inspect → detection status: "image_not_inspected", low confidence.
P1-011  Unknown entrypoint shape → detection status: "candidate_found", medium confidence, warnings present.
P1-012  image_ref is recorded as evidence, not used as primary match key.
P1-013  No RunPlan behavior change in this phase.
P1-014  No Agent Docker Create change in this phase.
P1-015  No container startup change in this phase.
P1-016  Existing NBR check flow still works (detection is additive).
P1-017  Unit tests pass: profiles, classification, scoring, detection output.
P1-018  `go test ./internal/server/runplan/... -count=1` passes.
P1-019  `go test ./internal/server/api/... -count=1` passes.
```

## 3. Phase 2 Acceptance Criteria: Config Acceptance + Snapshot Flow

```
P2-001  Accepted detection can be applied to NBR.process_start_config.
P2-002  Manual custom process_start_config can be saved.
P2-003  process_start_config located at config_snapshot_json.process_start_config (top-level).
P2-004  process_start_config NOT inside docker_json (Layer 2 separation).
P2-005  buildRuntimeConfigSnapshot captures process_start_config.
P2-006  buildDeploymentRuntimeSnapshot captures process_start_config.
P2-007  mergeNBRConfigSnapshot propagates process_start_config to deployment snapshot.
P2-008  applyDeploymentConfigSnapshot extracts process_start_config at start/dry-run.
P2-009  New deployment freezes process_start_config in config_snapshot_json.
P2-010  Frozen config survives deployment read/detail API round-trip.
P2-011  NBR update (new process_start_config) does not mutate existing deployment snapshots.
P2-012  Existing NBR without process_start_config continues legacy behavior.
P2-013  Existing Deployment without process_start_config continues legacy behavior.
P2-014  Invalid process_start_config (unknown entrypoint_mode) → error or clear warning.
P2-015  Valid process_start_config → stored, accessible via NBR read API.
P2-016  No RunPlan behavior change in this phase (data flow only).
P2-017  No Agent Docker Create change in this phase.
P2-018  Unit tests pass: snapshot, merge, apply, freeze, backward compat.
P2-019  `go test ./internal/server/api/... -count=1` passes.
```

## 4. Phase 3 Acceptance Criteria: RunPlan Execution

```
P3-001  image_default → RunPlan.Entrypoint = nil (omitted from JSON).
P3-002  custom → RunPlan.Entrypoint = process_start_config.entrypoint.
P3-003  missing process_start_config → legacy behavior (entrypoint_override > default_entrypoint).
P3-004  command_prefix prepended to Cmd (ResolvedRunPlan.Args).
P3-005  command_prefix NOT affected by deduplicateArgs or applyServiceArgs.
P3-006  buildArgs boundary verified: prefix added AFTER buildArgs internal dedup/applyServiceArgs.
P3-007  command_preview for image_default: no --entrypoint flag, annotation present.
P3-008  command_preview for custom: --entrypoint flag shown.
P3-009  command_preview for legacy: unchanged from current behavior.
P3-010  Agent Docker Create: nil entrypoint → cfg.Entrypoint unset → Docker preserves image ENTRYPOINT.
P3-011  Agent Docker Create: explicit entrypoint → cfg.Entrypoint set.
P3-012  Agent task payload correctly serializes nil/omitted entrypoint.
P3-013  NVIDIA GPU DeviceRequest unchanged after entrypoint change.
P3-014  MetaX device passthrough unchanged after entrypoint change.
P3-015  All HostConfig fields (ipc, shm, privileged, security_opt, ulimits, group_add) unchanged.
P3-016  Unit tests pass: all entrypoint modes, command_prefix, legacy fallback, preview.
P3-017  `go test ./internal/server/runplan/... -count=1` passes.
P3-018  `go test ./internal/agent/runtime/... -count=1` passes.
```

## 5. Phase 4 Acceptance Criteria: API Workflow / E2E / Real Smoke

```
P4-001  API workflow: NBR enable → check → read detection → accept config → verify snapshot.
P4-002  API workflow: create deployment → verify frozen process_start_config.
P4-003  API workflow: preflight/dry-run → verify runplan entrypoint/cmd.
P4-004  API workflow: start → verify /v1/models returns 200 (at least one backend).
P4-005  vLLM RunPlan: Entrypoint nil, Cmd = model args from Layer 4.
P4-006  SGLang RunPlan: Entrypoint nil, Cmd = ["python3","-m","sglang.launch_server"] + model args.
P4-007  llama.cpp RunPlan: Entrypoint nil, Cmd = model args from Layer 4.
P4-008  Real smoke: at least llama.cpp verified (known working).
P4-009  Real smoke: vLLM verified OR documented blocker with exact image/pull/error details.
P4-010  Real smoke: SGLang verified OR documented blocker with exact image/pull/error details.
P4-011  Backward compatibility smoke: existing deployment starts unchanged.
P4-012  NVIDIA HostConfig regression: DeviceRequest + GPU IDs present in real smoke evidence.
P4-013  MetaX HostConfig regression: devices/group_add/security_options present in resolver test.
P4-014  Container cleanup: no orphan lightai-* containers after smoke test.
P4-015  Logs captured: runplan JSON, command_preview, container logs, /v1/models response.
P4-016  Evidence saved to docs/reports/phase-3/ with timestamp.
```

## 6. Phase 5 Acceptance Criteria: Optional Trial-Run Probe (DEFERRED)

```
P5-001  Trial-run requires explicit user trigger (button/API action).
P5-002  Trial-run has hard timeout (configurable, default reasonable).
P5-003  Trial-run container is guaranteed cleaned up (--rm or ContainerRemove Force).
P5-004  No orphan containers after trial-run (docker ps verification).
P5-005  Trial-run does not mutate production NBR or Deployment config.
P5-006  Trial-run does not hold GPU lease.
P5-007  Trial-run logs and evidence captured and saved.
P5-008  Trial-run failure is actionable (clear error, logs, exit code).
P5-009  Port collision with running instances is prevented or clearly warned.
```

## 7. Negative Tests / Regression Tests

```
N-001  Official image repo name changed, backend_family same → detection still works.
N-002  Custom/private/vendor image (unknown registry) → gets candidate profiles by backend_family.
N-003  Image entrypoint already includes launcher → detection warns about command_prefix overlap.
N-004  Image entrypoint is empty → detection suggests image_default, low confidence.
N-005  process_start_config with invalid entrypoint_mode → error, not silent fallback.
N-006  process_start_config with custom mode but empty entrypoint → error or clear warning.
N-007  command_prefix is not duplicated in final Cmd (no double-launcher).
N-008  Existing deployment with frozen snapshot unaffected by NBR update.
N-009  MetaX devices present in docker_json after Layer 3 change.
N-010  NVIDIA DeviceRequest present after Layer 3 change.
N-011  Manual override (source: user_override) not overwritten by re-probe.
N-012  Re-probe only updates process_start_detection, leaves process_start_config intact.
N-013  Probe failure does not break NBR read/list/detail API.
N-014  Missing backend_family in profiles → detection status "no_profiles", actionable error.
N-015  Very long command_prefix (>10 tokens) → handled without truncation or error.
```

## 8. Evidence Requirements Per Phase

Each phase must produce and document:

```
Phase 1:
  - git status --short (clean, no unrelated changes)
  - go test ./internal/server/runplan/... -count=1 -v (all pass)
  - go test ./internal/server/api/... -count=1 -v (all pass)
  - Sample process_start_detection JSON from test output

Phase 2:
  - git status --short
  - go test ./internal/server/api/... -count=1 -v
  - Sample NBR config_snapshot_json with process_start_config
  - Sample Deployment config_snapshot_json with frozen process_start_config

Phase 3:
  - git status --short
  - go test ./internal/server/runplan/... -count=1 -v
  - go test ./internal/agent/runtime/... -count=1 -v
  - RunPlan JSON for vLLM/SGLang/llama.cpp (image_default and custom modes)
  - command_preview output for all three backends
  - Agent task payload sample with nil entrypoint

Phase 4:
  - git status --short
  - Shell E2E log for detection + config + deploy workflow
  - Real smoke log: image inspect Entrypoint/Cmd
  - Real smoke log: process_start_config
  - Real smoke log: RunPlan Entrypoint/Cmd
  - Real smoke log: /v1/models response
  - Real smoke log: container cleanup verification (docker ps -a)
  - Evidence saved to docs/reports/phase-3/real-smoke-<timestamp>/

Phase 5 (if approved):
  - Trial-run log with timeout
  - docker ps -a showing no orphan containers
  - Trial-run failure case: clear error + actionable message
```

## 9. Final Approval Gate

```
Implementation may start only after:
1. This acceptance criteria document is reviewed.
2. The implementation plan is reviewed.
3. All open questions in the implementation plan §6 are answered.
4. The user explicitly approves entry into Phase 1 implementation.
```

## 10. Final State

```
ACCEPTANCE_CRITERIA_DRAFT_READY_FOR_REVIEW
```
