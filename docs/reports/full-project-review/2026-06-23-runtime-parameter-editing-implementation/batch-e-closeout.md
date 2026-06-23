# Batch E Closeout: Deployment Override / Disabled Tombstone

> Date: 2026-06-24
> Status: PASS

---

## Summary

Implemented deployment parameter overrides and disabled tombstones.

## E1: API / Storage

### Changes
- Added `parameter_values_json` and `disabled_parameters_json` to deployment INSERT/SELECT/PATCH
- API returns both fields in deployment responses

### Commit
```
1322df9 feat(deployments): add parameter overrides and disabled tombstones
```

## E2: RunPlan Merge

### Changes
- Layer 3: deployment parameter_values override NBR values
- Layer 4: deployment parameters_json (legacy support)
- Disabled tombstones remove args and env from final output
- Disabled != absent != empty value
- NBR still source of truth; deployment overrides have highest priority

### Commit
```
1322df9 feat(deployments): add parameter overrides and disabled tombstones
```

## E3: Web Deployment Parameter Editor

### Changes
- Import RuntimeParameterEditor into ModelDeploymentsPage
- Add parameter_values_json and disabled_parameters_json to edit form
- Load structured parameters on showEdit
- Include in save payload
- Add i18n key structuredParameters

### Commit
```
8f95109 feat(web): add deployment parameter override editor
```

## Files Changed

| File | Change |
|------|--------|
| `internal/server/api/deployment_lifecycle_handlers.go` | Add parameter fields to INSERT/SELECT/PATCH |
| `internal/server/runplan/resolver.go` | Add deployment overrides and disabled tombstones to buildArgs/buildEnv |
| `web/src/pages/ModelDeploymentsPage.vue` | Add RuntimeParameterEditor integration |
| `web/src/locales/en-US.ts` | Add structuredParameters key |
| `web/src/locales/zh-CN.ts` | Add structuredParameters key |

## RunPlan Merge Rules

1. NBR parameter values (Layer 2)
2. Deployment parameter_values overrides (Layer 3) — highest priority
3. Deployment parameters_json (Layer 4, legacy)
4. Disabled tombstones remove args/env from final output
5. absent = keep upstream value
6. disabled = remove from output
7. empty enabled value = skip (not output)

## `/tmp/lightai` Status

**NOT updated.** Running server/agent has NOT been rebuilt.

## Test Results

| Command | Result |
|---------|--------|
| `go build ./internal/server/...` | PASS |
| `go test ./internal/server/api/...` | PASS |
| `go test ./internal/server/runplan/...` | PASS |
| `cd web && npm run build` | PASS |
| `cd web && npm test` | PASS |

## Git Status

```
 M VERSION
?? .mimocode/plans/1782215119986-calm-planet.md
?? .mimocode/skills/
```
