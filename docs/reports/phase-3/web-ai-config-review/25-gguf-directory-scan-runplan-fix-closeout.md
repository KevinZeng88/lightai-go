# 25 — GGUF Directory Scan RunPlan Fix Closeout

> Status: FIXED
> Scope: Preserve .gguf file path from directory scans so RunPlan -m points to file
> Date: 2026-06-23
> Baseline: commit `ed2f145` (V26 migration)

## 1. User Verification

| Method | Before | After |
|--------|--------|-------|
| Direct file selection | `-m /models/.../Qwen3.5-9B-Q4_K_M.gguf` ✅ | Same ✅ |
| Directory scan | `-m /models/Qwen3.5-9B-Q4` ❌ | `-m /models/.../Qwen3.5-9B-Q4_K_M.gguf` ✅ |

## 2. Root Cause

In `doWizardSave()` (ModelArtifactsPage.vue:547), the model location's `relative_path` was taken from `scanResult.relative_path` — the scan proxy's top-level path that always points to the scanned DIRECTORY. The candidate's specific file path (`c.path`) was only used for `absolute_path`, leaving `relative_path` as the directory name.

This caused:
- `model_locations.relative_path` = `Qwen3.5-9B-Q4` (directory only)
- `model_locations.path_type` = `file` (contradicting the directory path!)
- Resolver's `modelRelativePath()` returned the directory → `-m /models/Qwen3.5-9B-Q4`

## 3. Fix Applied

### Frontend: Derive relative_path from candidate file path

In `doWizardSave()`, instead of using `scanResult.relative_path`, derive `relative_path` from the candidate's file path relative to the scan root:

```javascript
const candidatePath = c.path || scanResult.value?.absolute_path || ''
const scanRoot = scanResult.value?.model_root || scanResult.value?.root || ...
// Derive: /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
//   root = /home/kzeng/models
//   → relative_path = Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
//   → path_type = file
```

### Backend: Validate GGUF locations have .gguf file paths

In `HandleCreateModelLocation`, reject `path_type=directory` for GGUF format artifacts:

```go
if artifact.format == "gguf" && !strings.HasSuffix(absolutePath, ".gguf") {
    error: "GGUF models require a .gguf file path, not a directory"
}
```

### UI: No model files found

Added warning alert when scan returns 0 candidates: "目录中没有发现模型文件"

## 4. Correct RunPlans After Fix

### Direct file selection:
```bash
-v /home/.../Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf:/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf:ro
-m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf ✅
```

### Directory scan (single GGUF):
```bash
-v /home/.../Qwen3.5-9B-Q4:/models/Qwen3.5-9B-Q4:ro
-m /models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf ✅
```

## 5. File Discovery Rules

| Scenario | Behavior |
|----------|----------|
| 0 model files in directory | Error: "目录中没有发现模型文件"; block creation |
| 1 GGUF file in directory | Auto-select as primary file; save as file location |
| Multiple GGUF files in directory | Show candidate list; require user selection before continuing |

## 6. Schema / Migration

- No schema changes
- No new migration
- V26 migration (Phase 2.6) already handles existing stale data

## 7. Items NOT Done

- No resource parameter editor (Phase 3)
- No multi-replica/cross-node scheduling
- No Playwright specs
- No Phase 3 scope creep

## 8. Test Results

```bash
gofmt -w cmd/ internal/                     → CLEAN
go test lightai-go/internal/server/api/...    → ALL PASS
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet ./...                                   → CLEAN
npm test                                       → ALL PASS
npm run build                                  → ✓ built
git diff --check                                → CLEAN
```

## 9. Modified Files

| File | Change |
|------|--------|
| `web/src/pages/ModelArtifactsPage.vue` | Derive location relative_path from candidate file path; add no-model alert |
| `internal/server/api/artifact_handlers.go` | Validate GGUF location paths |
| `web/src/locales/zh-CN.ts` | Update noModelFound key |
| `web/src/locales/en-US.ts` | Update noModelFound key |
| `docs/.../24-gguf-directory-scan-runplan-regression.md` | Regression analysis |
| `docs/.../25-gguf-directory-scan-runplan-fix-closeout.md` | This closeout |

## 10. Final Status

PASS — both direct file selection and directory scan produce correct RunPlan with `.gguf` file path in `-m`.
