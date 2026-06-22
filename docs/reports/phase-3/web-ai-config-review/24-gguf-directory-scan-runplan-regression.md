# 24 — GGUF Directory Scan RunPlan Regression

> Status: INVESTIGATING
> Scope: Directory scan loses .gguf file path; RunPlan uses directory for -m
> Date: 2026-06-23
> Baseline: commit `ed2f145`

## WEB-AI-RC-006: Directory Scan -m Still Points to Directory While Direct File Works

### User Discovery

| Method | RunPlan -m | Status |
|--------|-----------|--------|
| Direct file selection: `/home/.../Qwen3.5-9B-Q4_K_M.gguf` | `-m /models/.../Qwen3.5-9B-Q4_K_M.gguf` | ✅ Correct |
| Directory scan: `/home/.../Qwen3.5-9B-Q4` directory | `-m /models/Qwen3.5-9B-Q4` | ❌ Wrong directory |

### Root Cause

In `doWizardSave()` (ModelArtifactsPage.vue), the model location was created using `scanResult.relative_path` (the scan directory) instead of deriving the relative_path from the candidate's specific file path. The scan proxy's top-level `relative_path` always points to the directory that was scanned, not the discovered file within.

```javascript
// BEFORE (wrong):
relative_path: scanResult.value?.relative_path  // "Qwen3.5-9B-Q4" (directory)
path_type: c.path_type || 'directory'

// AFTER (correct):
// Derive relative_path from candidate path relative to scan root
// "/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf"
// → model_root = "/home/kzeng/models"
// → relative_path = "Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf"
// → path_type = "file"
```

### Fix

1. Frontend wizard: derive `relative_path` from candidate file path relative to scan root
2. Backend: validate GGUF locations have file paths (not directories)
3. Add "no models found" handling for empty scan results
4. Add multi-file selection for directories with multiple GGUF files

### File Discovery Rules

- 0 files: error "目录中没有发现模型文件", block creation
- 1 file: auto-select, save as file location
- Multiple files: show candidate list, require user selection
