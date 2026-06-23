# Batch C Closeout: BackendRuntime / NodeBackendRuntime Parameter Editing

> Date: 2026-06-24
> Status: PASS (corrected)

---

## Summary

Corrected Batch C to remove BV/BR fallback from RunPlan resolver and integrate RuntimeParameterEditor into BR/NBR pages.

## Corrections Made

### C2 Correction: Remove BV/BR Fallback

**Original issue**: RunPlan resolver had legacy path that fell back to BackendVersion/BackendRuntime when NBR snapshot was missing.

**Fix**: Removed legacy path. Resolver now returns explicit error if NBRConfigSnapshot is missing: "node backend runtime parameter snapshot is missing; recreate node backend runtime or rebuild database"

**Commits**:
```
b8a8756 fix(runplan): remove BV/BR fallback from parameter resolution
```

### C3 Correction: Integrate RuntimeParameterEditor

**Original issue**: RuntimeParameterEditor component created but not integrated into pages.

**Fix**: Integrated into BackendRuntimesPage and RunnerConfigsPage edit dialogs.

**Commits**:
```
ccb604f fix(web): wire runtime parameter editor into BR and NBR pages
```

## Files Changed

| File | Change |
|------|--------|
| `internal/server/runplan/resolver.go` | Remove BV/BR fallback, NBR-only path |
| `internal/server/runplan/resolver_test.go` | Update tests for NBR-only path |
| `internal/server/runplan/test_helpers_test.go` | New: ensureNbrSnapshot helper |
| `internal/server/runplan/llamacpp_nvidia_test.go` | Add NBR snapshot to tests |
| `internal/server/runplan/vllm_sglang_nvidia_test.go` | Add NBR snapshot to tests |
| `internal/server/runplan/metax_huawei_test.go` | Add NBR snapshot to tests |
| `web/src/pages/BackendRuntimesPage.vue` | Integrate RuntimeParameterEditor |
| `web/src/pages/RunnerConfigsPage.vue` | Integrate RuntimeParameterEditor |
| `web/src/api/runtimes.ts` | Add parameter fields to BackendRuntime |
| `web/src/locales/en-US.ts` | Add structuredParameters key |
| `web/src/locales/zh-CN.ts` | Add structuredParameters key |

## RunPlan Changes

- **No fallback**: Resolver returns error if NBR snapshot missing
- **NBR is sole source**: All runtime params read from NBR snapshot
- **BV/BR only for creation**: BV/BR used when creating NBR, not at resolution time

## Web Changes

- RuntimeParameterEditor integrated into BackendRuntimesPage edit dialog
- RuntimeParameterEditor integrated into RunnerConfigsPage edit dialog
- parameter_values_json and parameter_schema_json included in API payloads

## `/tmp/lightai` Status

**NOT updated.** Running server/agent has NOT been rebuilt.

## Test Results

| Command | Result |
|---------|--------|
| `go build ./internal/server/...` | PASS |
| `go test ./internal/server/runplan/...` | PASS |
| `cd web && npm run build` | PASS |
| `cd web && npm test` | PASS |

## Commits

```
ccb604f fix(web): wire runtime parameter editor into BR and NBR pages
b8a8756 fix(runplan): remove BV/BR fallback from parameter resolution
```

## Git Status

```
 M VERSION
?? .mimocode/plans/1782215119986-calm-planet.md
?? .mimocode/skills/
```
