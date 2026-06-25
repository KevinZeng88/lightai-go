# Phase 04 - Renderer, RunPlan, AgentRunSpec, DockerSpec

## Scope

Checkpoint D implemented the renderer boundary from ConfigSet-derived runtime data into `ResolvedRunPlan`, `AgentRunSpec`, and Docker command preview/start inputs.

This phase keeps the clean-state invariant:

- no legacy DB/API payload fallback
- no old authority fields as runtime source
- no direct Docker start spec assembled independently from RunPlan
- no active V1->V28 migration replay

## Changed Files

| File | Purpose |
| --- | --- |
| `internal/server/api/configset_helpers.go` | Maps ConfigSet `cli_arg` and `cli_args` items into structured RunPlan parameter values, including render style metadata. |
| `internal/server/runplan/resolver.go` | Renders `flag_space_value`, `flag_equals_value`, `flag_if_true`, `repeat_flag`, `positional`, and `raw_lines` styles from ConfigSet parameter values. Preserves repeatable flags during argument deduplication. |
| `internal/server/runplan/resolver_test.go` | Covers ConfigSet render styles and repeatable flag preservation. |
| `internal/agent/runtime/runplan_adapter.go` | Extends RunPlan-to-AgentRunSpec conversion with operation ID, backend metadata, GPU device request metadata, and health check. |
| `internal/agent/runtime/runplan_adapter_test.go` | Covers the expanded AgentRunSpec conversion contract. |
| `internal/server/api/deployment_lifecycle_handlers.go` | Uses `agentruntime.ConvertRunplanToAgentSpec` when creating start tasks, so agent input is derived from ResolvedRunPlan. |

## Deleted Old Structures

No additional old DB columns or legacy APIs were deleted in this phase. Checkpoints B/C already removed active old authority fields from DB/API/RunPlan scope. Checkpoint D prevented a new divergence by replacing hand-built start task runtime specs with the shared RunPlan adapter.

## New Structures

| Structure | Purpose |
| --- | --- |
| `ParameterValue.RenderStyle` | Carries ConfigSet renderer metadata into RunPlan resolution. |
| ConfigSet parameter renderer | Produces CLI args from structured ConfigSet values instead of old JSON parameter fields. |
| Repeatable flag metadata | Allows parameters such as LoRA adapters to intentionally render repeated flags without being collapsed by generic deduplication. |
| Expanded `PlanInput` / `AgentRunSpec` mapping | Keeps deployment start task payloads aligned with ResolvedRunPlan and DockerSpec preview data. |

## Validation

| Command | Result | Summary |
| --- | --- | --- |
| `go test ./internal/server/runplan ./internal/agent/runtime ./internal/server/api -count=1` | PASS | Targeted renderer, AgentRunSpec, and API lifecycle tests passed. |
| `go test ./...` | PASS | All Go packages passed. |
| `go build ./cmd/server/...` | PASS | Server binary builds. |
| `go build ./cmd/agent/...` | PASS | Agent binary builds. |
| `rg -n "config_snapshot_json|parameter_schema_json|parameter_values_json|image_name|docker_json|default_env_json|capabilities_json|capability_sources_json|parameter_defaults_json|default_args_json|parameter_defs_json|default_backend_params_json|default_images_json|image_candidates_json|docker_options_json|model_mount_json|seedBuiltInBackends|seedTargetBackendCatalog|repairBackendCapabilitiesV27|normalizeLegacyBackendCatalogIDs|migrateV[0-9]+" internal/server/api internal/server/runplan internal/server/db` | PASS | No exact old authority fields, old catalog seed/repair functions, or `migrateVx` chain were introduced. |
| `git diff --check` | PASS | No whitespace errors. |

## Problem Closure

| ID | Issue | Status | Evidence |
| --- | --- | --- | --- |
| CS-D-001 | `repeat_flag` output was collapsed by generic argument deduplication. | FIXED | `go test ./internal/server/runplan ./internal/agent/runtime ./internal/server/api -count=1`; `go test ./...`. |

All Checkpoint D problems found during implementation are either fixed or tracked in `open-issues.md`.

## Workspace Notes

The unrelated pre-existing files `web/package.json`, `web/package-lock.json`, `.mimocode/`, project-wide execution-plan reports, project-wide review reports, and historical runtime evidence directories remain outside this checkpoint. They must not be staged for the Checkpoint D commit.

## Next Checkpoint

Checkpoint E: API/UI refactor + stale documentation archive.
