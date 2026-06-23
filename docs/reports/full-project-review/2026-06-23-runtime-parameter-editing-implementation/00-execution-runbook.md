# Runtime Parameter Editing — Execution Runbook

> Date: 2026-06-24
> Status: Active

---

## 1. Confirmed Decisions

1. Parameter JSON uses structured array, default `[]`
2. BackendRuntime saves `parameter_schema_json` + `parameter_values_json`
3. NodeBackendRuntime saves `parameter_schema_json` + `parameter_values_json`
4. NBR is final RunPlan backend-side source of truth
5. RunPlan does NOT dynamically query BackendVersion/BackendRuntime
6. Deployment disabled override uses structured tombstone array
7. Disabled ≠ absent ≠ empty value
8. Deployment override replaces upstream value (not merge)
9. RunPlan resolver big bang migration, no fallback
10. Batch B can rebuild DB, no old data compatibility
11. Old `parameters_json` replaced by structured `parameter_values_json`
12. Memory params must be backend-specific, no unified `gpu_memory_limit`
13. llama.cpp does not fake memory percentage parameter

---

## 2. Batch List

### Batch A — Documentation & Contract Solidification
- **Goal**: Update engineering contracts and design docs with runtime parameter editing contract
- **Scope**: `docs/08-engineering-contracts.md`, `docs/lightai-backend-runtime-runplan-docker-design.md`
- **Forbidden**: No code changes
- **Tests**: `git diff --check`
- **Commit**: `docs: solidify runtime parameter editing contract`
- **Closeout**: `batch-a-closeout.md`

### Batch B — Schema / Seed / Catalog Cleanup
- **Goal**: Add structured parameter schema columns, clean capability metadata from env, add backend-specific parameter schemas
- **Scope**: `internal/server/db/db.go`, `configs/backend-catalog/`, `internal/server/runplan/`
- **Forbidden**: No backward compatibility, no old DB migration shim
- **Tests**: `go test ./internal/server/runplan/... -count=1`, `go test ./internal/server/... -count=1`
- **Commit**: `feat(runtime): add structured parameter schema snapshots`
- **Closeout**: `batch-b-closeout.md`

### Batch C — BackendRuntime / NodeBackendRuntime Parameter Editing
- **Goal**: Enable structured parameter editing at BR and NBR levels
- **Scope**: Web components, API handlers, RunPlan resolver migration
- **Forbidden**: No fallback to old resolver path
- **Tests**: Unit + integration
- **Commit**: `feat(runtime): enable BR/NBR parameter editing`
- **Closeout**: `batch-c-closeout.md`

### Batch D — ModelArtifact / ModelLocation Parameter Editing
- **Goal**: Enable model-level default parameter editing
- **Scope**: Model artifacts page, API handlers
- **Forbidden**: ModelArtifact/ModelLocation cannot override NBR container config
- **Tests**: Unit + integration
- **Commit**: `feat(model): enable model parameter editing`
- **Closeout**: `batch-d-closeout.md`

### Batch E — Deployment Override / Disabled Tombstone
- **Goal**: Enable deployment-level parameter override with disabled state
- **Scope**: Deployment page, RunPlan merge logic
- **Forbidden**: No backward compatibility with old parameters_json
- **Tests**: Unit + integration
- **Commit**: `feat(deployment): enable parameter override with disabled tombstone`
- **Closeout**: `batch-e-closeout.md`

### Batch F — E2E Validation and Closeout
- **Goal**: End-to-end validation, final closeout
- **Scope**: All modified code
- **Tests**: Full test suite, manual validation
- **Commit**: `docs: runtime parameter editing implementation closeout`
- **Closeout**: `batch-f-closeout.md`

---

## 3. Stop Conditions

Must stop and ask user when:

1. Need to change NBR-as-source-of-truth principle
2. Need to preserve old DB/API/resolver fallback
3. Need complex migration for old data compatibility
4. Design docs conflict with code reality, cannot proceed
5. Need to delete large amounts of non-task code
6. Need real long-running GPU E2E
7. Need to push
8. Test failure cannot be resolved within current batch
9. git status shows non-task file modifications

---

## 4. Closeout Template

Each batch closeout must contain:

```
# Batch {X} Closeout: {Name}

> Date: YYYY-MM-DD
> Status: PASS / FAIL / STOPPED

## Summary
- What was done
- Why

## Files Changed
| File | Change |
|------|--------|

## Schema Changes
(if any)

## API Changes
(if any)

## Test Results
| Command | Result |
|---------|--------|

## Unresolved Issues
(if any)

## Git Status
`git status --short`

## Commit SHA
{sha}
```
