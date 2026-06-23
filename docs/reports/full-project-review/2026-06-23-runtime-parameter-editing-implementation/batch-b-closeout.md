# Batch B Closeout: Schema / Seed / Catalog Cleanup

> Date: 2026-06-24
> Status: PASS (revised after checkpoint audit)

---

## Summary

Added structured parameter schema columns via V28 migration. Updated seed data with backend-specific memory/resource parameters. Cleaned capability metadata from env_json.

## Files Changed

| File | Change |
|------|--------|
| `internal/server/db/db.go` | V28 migration, seed data updates |
| `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml` | Added memory/resource params |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml` | Added memory/resource params |
| `configs/backend-catalog/versions/sglang/sglang-v0.5.12.post1.yaml` | Added memory/resource params |
| `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml` | Added batch-size, ubatch-size |

## Catalog Source-of-Truth

**Conclusion**: `configs/backend-catalog/` YAML files are NOT loaded at runtime. The server only uses `internal/server/db/db.go` seed data. YAML files are reference documentation only.

**Implication**: Batch B only needs to fix `db.go` seed data. YAML files are updated for consistency but do not affect runtime behavior.

## Schema Changes (V28 Migration)

```sql
ALTER TABLE backend_runtimes ADD COLUMN parameter_schema_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE backend_runtimes ADD COLUMN parameter_values_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE node_backend_runtimes ADD COLUMN parameter_schema_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE node_backend_runtimes ADD COLUMN parameter_values_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_deployments ADD COLUMN parameter_values_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_deployments ADD COLUMN disabled_parameters_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_artifacts ADD COLUMN parameter_defaults_json TEXT NOT NULL DEFAULT '[]';
```

All 7 new columns use structured array default `[]`.

## Seed Data Cleanup

| Backend | env_json Before | env_json After | Notes |
|---------|----------------|----------------|-------|
| vLLM v0.23.0 | `{}` | `{}` | Already clean |
| SGLang v0.5.13.post1 | capability metadata | `{}` | Cleaned in checkpoint audit |
| SGLang v0.5.12.post1 | capability metadata | `{}` | Cleaned in checkpoint audit |
| llama.cpp b9700 | `{}` | `{}` | Already clean |

Capability metadata stays in `capabilities_json` only.

## New Parameters Added

| Backend | New Parameters |
|---------|---------------|
| vLLM | `--max-num-seqs`, `--max-num-batched-tokens`, `--gpu-memory-utilization`, `--enforce-eager`, `--trust-remote-code` |
| SGLang | `--max-running-requests`, `--served-model-name`, `--mem-fraction-static`, `--context-length`, `--disable-cuda-graph` |
| llama.cpp | `--batch-size`, `--ubatch-size` |

## `/tmp/lightai` Running Instance

**NOT updated.** The running server/agent at `/tmp/lightai` has NOT been rebuilt or restarted. V28 migration has NOT been applied to `/tmp/lightai/data/lightai.db`.

To verify V28 migration:
1. Build new server/agent binaries
2. Backup `/tmp/lightai/data/lightai.db`
3. Restart server to trigger migration
4. Only then check DB schema

**This checkpoint does NOT modify `/tmp/lightai/data` or `/tmp/lightai` programs.**

## API Changes

None — columns added but not yet used by handlers.

## Test Results

| Command | Result |
|---------|--------|
| `go build ./cmd/server/...` | PASS |
| `go test ./internal/server/runplan/...` | PASS |
| `git diff --check` | PASS |

## DB Rebuild

- Existing databases will get new columns via V28 migration on next server start
- Old data preserved (empty arrays)
- **SGLang/llama.cpp env_json cleanup**: Will take effect on next seed run (server restart with rebuilt binary)
- Full rebuild recommended for clean state but not required

## Commit SHA

```
17594db feat(runtime): add structured parameter schema snapshots
```

## Git Status

```
 M VERSION
?? .mimocode/plans/1782215119986-calm-planet.md
?? .mimocode/skills/
```
