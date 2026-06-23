# Batch Execution Plan — Runtime Operations UX & Resource Controls

> Date: 2026-06-23
> Status: Execution plan. See 04-claude-review-report.md for review findings.

## 1. Why Batch-based instead of Phase-based

The original plan (05-revised-execution-plan.md) defines Phases 0–7 by risk boundary. That is correct for review and risk analysis, but executing phase-by-phase would cause repeated modifications to the same files:

- `resolver.go` touched by Phase 1 (lint hook) and Phase 2a (resource_controls injection)
- `deployment_lifecycle_handlers.go` touched by Phase 1 (lint in dry-run), Phase 2a (resource_controls in dry-run), and Phase 3 (log classifier in test result)
- `RunnerConfigsPage.vue` touched by Phase 5 (JsonViewer) and Phase 6a (HealthCheckEditor)
- `ModelInstancesPage.vue` touched by Phase 4 (auto-refresh)

Batch execution groups by **modification surface** to minimize churn:

| Batch | Scope | Modification surface |
|-------|-------|---------------------|
| A | Backend Runtime Diagnostics & Resource Controls | Go backend: runplan package, API handlers, catalog YAML |
| B | Frontend Runtime UX & Diagnostics | Vue frontend: composables, components, pages |
| C | Final Regression & Closeout | Tests, build, docs, git |

Phase risk boundaries are preserved. Deferred items remain deferred.

## 2. Batch A — Backend Runtime Diagnostics & Resource Controls

**Aggregates**: Phase 1 (RunPlan lint) + Phase 2a (resource_controls) + Phase 3 (log classifier) + dry-run/model-test diagnostic response + Go tests/fixtures/closeout.

### 2.1 RunPlan lint

Two-stage lint:

1. **Pre-normalization lint**: runs on raw Layer 1–4 args before deduplicateArgs. Records user overrides of platform-owned parameters and duplicate flags.
2. **Deduplicate**: existing `deduplicateArgs()` keeps last occurrence.
3. **Final lint**: runs on the final resolved args + env. Detects env/CLI conflicts, high-risk Docker flags, and remaining anomalies.

LLAMA_ARG_HOST handling:
- Image-provided env (detected by checking if env var exists in the image but not in platform/user layers) → `warning`, does NOT block
- User-provided env conflicting with platform CLI → `error`, blocks in dry-run
- Log classifier separately classifies the runtime warning

Lint result is embedded in dry-run response, NOT a new API route.

### 2.2 resource_controls

- Definition stored in catalog YAML `vendor_options.resource_controls`
- User-tunable parameters (e.g. memory_fraction) belong in deployment `parameters_json`, not in backend_versions
- `vendor_options_json` stores "what resource controls this backend supports" metadata
- llama.cpp does NOT have `gpu_memory_fraction`
- No shared GPU admission (Phase 2b is DOCUMENTED_BLOCKER)
- No schema change

Seed behavior: catalog YAML `vendor_options` → `vendor_options_json` in DB. ON CONFLICT overwrites. Users adjust parameters via deployment, not via backend_versions.

### 2.3 Runtime log classifier

Go-built-in rules, no DB storage. Fixture tests with real log samples.

### 2.4 Dry-run response enrichment

Add `lint` field to existing dry-run response alongside `valid`, `errors`, `warnings`, `command_preview`.

### 2.5 Files

**New files**:
- `internal/server/runplan/lint.go`
- `internal/server/runplan/lint_test.go`
- `internal/server/runplan/resource_controls.go`
- `internal/server/runplan/resource_controls_test.go`
- `internal/server/runplan/log_classifier.go`
- `internal/server/runplan/log_classifier_test.go`
- `internal/server/runplan/testdata/runtime-logs/sglang-torchao-syntax-warning.log`
- `internal/server/runplan/testdata/runtime-logs/sglang-attention-backend-default.log`
- `internal/server/runplan/testdata/runtime-logs/llamacpp-env-host-overwritten.log`
- `internal/server/runplan/testdata/runtime-logs/cuda-oom.log`
- `docs/reports/phase-3/runtime-operations-ux-resource-controls/06-batch-execution-plan.md`
- `docs/reports/phase-3/runtime-operations-ux-resource-controls/batch-a-backend-runtime-diagnostics-closeout.md`

**Modified files**:
- `internal/server/runplan/resolver.go` (lint hook in buildArgs)
- `internal/server/api/deployment_lifecycle_handlers.go` (lint in dry-run response)
- `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml` (vendor_options.resource_controls)
- `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml` (vendor_options.resource_controls)
- `configs/backend-catalog/versions/sglang/sglang-v0.5.12.post1.yaml` (vendor_options.resource_controls)
- `configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml` (vendor_options.resource_controls)
- `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml` (vendor_options.resource_controls)

**NOT modified**:
- DB schema / gpu_leases / shared GPU admission
- Frontend pages / components
- JsonViewer / HealthCheckEditor / ConfigEditorLayout
- status-summary API
- vitest / package.json
- VERSION

### 2.6 Verification

```bash
go test ./internal/server/runplan/... -v
go test ./internal/server/api/... -v
go test ./...
go build ./...
gofmt -l internal/
git diff --check
```

## 3. Batch B — Frontend Runtime UX & Diagnostics (NOT executed now)

**Aggregates**: Phase 4 (auto-refresh) + Phase 5 (JsonViewer) + Phase 6a (HealthCheckEditor).

Requires Batch A complete (lint/resource_controls/log_classifier available via API).

## 4. Batch C — Final Regression & Closeout (NOT executed now)

**Aggregates**: Full regression + closeout + deferred/blocker documentation.

Requires Batch A and B complete.

## 5. Deferred items (unchanged)

| Item | Status | Trigger |
|------|--------|---------|
| Phase 2b: Shared GPU admission / budget-based GPU lease | DOCUMENTED_BLOCKER | User requests shared GPU |
| Phase 6b: Complete ConfigEditorLayout | deferred | Separate project |
| Status-summary API | conditional | Polling performance insufficient |
| vitest introduction | deferred | Vue composable tests require it |
| llama.cpp VRAM estimator | deferred | User requests VRAM estimation |
