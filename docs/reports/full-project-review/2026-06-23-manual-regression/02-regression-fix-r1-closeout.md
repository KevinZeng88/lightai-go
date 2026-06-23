# Regression Fix R1 Closeout

> Date: 2026-06-23
> Status: PASS

---

## Fixes Applied

### Fix 1: P0 Preflight Validation Regression

**Root cause**: Batch 4's `mapParametersToArgs()` reported required params as missing even when provided by `default_args` (Layer 1).

**Fix**: Added `collectExistingFlags()` helper that extracts all flag names from earlier arg layers. `mapParametersToArgs()` now skips required param check if the flag is already present.

**Files**: `internal/server/runplan/resolver.go`

### Fix 2: Empty Env Vars Filter

**Root cause**: `backend_versions.env_json` stores capability metadata (arrays/maps) that deserialize to empty strings when loaded into `map[string]string`.

**Fix**: Added `addEnv()` helper in `buildEnv()` that skips empty string values.

**Files**: `internal/server/runplan/resolver.go`

### Fix 3: Entrypoint Position

**Root cause**: `EquivalentCommandPreview()` placed `--entrypoint` after image name, but Docker CLI requires it before.

**Fix**: Moved entrypoint rendering before image name.

**Files**: `internal/server/runplan/preview.go`

---

## Commits

| SHA | Message |
|-----|---------|
| 39c05e9 | fix(runplan): required param check skips earlier layers, filter empty env, fix entrypoint order |

---

## Test Results

| Command | Result |
|---------|--------|
| `go build ./cmd/server/...` | PASS |
| `go build ./cmd/agent/...` | PASS |
| `go test ./internal/server/...` | ALL PASS |
| `go test ./internal/agent/...` | ALL PASS |

---

## Remaining Items (Not Fixed)

| Item | Classification | Priority |
|------|---------------|----------|
| Parameter editing | Requirement not implemented | P3 |
| GPU memory limit | Requirement not implemented | P3 |
| Seed data cleanup (capability metadata in env_json) | Catalog/config bug | P2 |

---

## Push Recommendation

**可以 push** — P0 回归已修复，所有测试通过。

**Commit range to push**: `ee811ca..39c05e9` (12 commits)
