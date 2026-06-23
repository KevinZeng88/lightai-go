# Manual Regression Fix Plan

> Date: 2026-06-23
> Priority order for fixes

---

## Fix 1: Preflight Validation Regression (P0 — AUTORUN Regression)

**Problem**: Batch 4's required parameter validation reports params as missing even when provided by `default_args`.

**Fix**: In `mapParametersToArgs()`, accept the existing args list and skip required param check if the param flag is already present.

**Files**:
- `internal/server/runplan/resolver.go` — modify `mapParametersToArgs` signature and logic

**Approach**:
```go
func mapParametersToArgs(params map[string]interface{}, defs []ParameterDef, errs *[]error, existingFlags map[string]bool) []string {
    // ...
    if !ok {
        if def.Required {
            // Check if already provided by earlier layers
            cliName := def.effectiveCliName()
            if cliName != "" && existingFlags[cliName] {
                continue // already provided
            }
            if existingFlags[def.Name] {
                continue // already provided
            }
            *errs = append(*errs, fmt.Errorf("required parameter %q missing", def.Name))
        }
    }
}
```

**Test plan**:
- `go test ./internal/server/runplan/... -count=1`
- Manual: create llama.cpp deployment and start → should succeed

**Can auto-fix**: YES

---

## Fix 2: Empty Env Vars (P1 — Catalog/Config Bug)

**Problem**: `backend_versions.env_json` stores capability metadata, not Docker env vars.

**Fix**: Move capability metadata from `env_json` to `capabilities_json`, and filter out non-string env values in `buildEnv()`.

**Files**:
- `internal/server/db/db.go` — fix seed data
- `internal/server/runplan/resolver.go` — filter non-string env values

**Approach**:
1. In seed data, move `supported_formats`, `supported_tasks`, etc. from `env_json` to `capabilities_json`
2. In `buildEnv()`, skip values that are not strings (arrays, maps)

**Test plan**:
- Rebuild DB or run migration
- Check Docker command preview has no empty env vars

**Can auto-fix**: Partially — seed data fix is straightforward, but existing DB needs migration or rebuild.

---

## Fix 3: Entrypoint Position (P2 — Preview-Only Bug)

**Problem**: Equivalent Docker command puts `--entrypoint` after image.

**Fix**: Fix the preview renderer to put `--entrypoint` before the image name.

**Files**:
- Web UI component or server-side preview generation

**Can auto-fix**: YES (once the renderer location is identified)

---

## Fix 4: Parameter Editing (P3 — Feature Request)

**Problem**: Need parameter editing at model library, runtime config, and deployment levels.

**Fix**: Requires design work — not a quick fix.

**Can auto-fix**: NO — requires design discussion

---

## Fix 5: GPU Memory Limit (P3 — Feature Request)

**Problem**: No unified GPU memory limit configuration.

**Fix**: Requires design work — backend-specific parameters already exist in catalog.

**Can auto-fix**: NO — requires design discussion

---

## Recommended Execution Order

1. Fix 1 (P0) — preflight validation regression
2. Fix 2 (P1) — empty env vars
3. Fix 3 (P2) — entrypoint position
4. Fix 4/5 (P3) — design phase

---

## Push Recommendation

**暂缓 push** until Fix 1 (P0 regression) is resolved. Fix 1 is the only AUTORUN regression and blocks deployment start for SGLang and llama.cpp.
