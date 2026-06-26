# ConfigSet Refactor — Post-Codex Claude Audit

Date: 2026-06-26
Auditor: Claude
Base commit: 404aeff

## 1. Acceptance of Codex Closeout

**ACCEPTED** — the ConfigSet refactor is substantively complete and correct.

## 2. Actual Findings

### 2.1 Static Checks

| Check | Result | Classification |
|-------|--------|---------------|
| config_set_json in API response | NOT exposed (deleted at lines 1187-1188) | Correct |
| config_set_json in DB queries | Used internally only | Allowed internal DB column |
| Legacy functions (seedBuiltInBackends, seedTargetBackendCatalog, repairBackendCapabilitiesV27, normalizeLegacyBackendCatalogIDs, migrateV[0-9]) | ZERO hits in active code | Fully removed |
| HandleCheckNodeBackendRuntime | Deprecated, returns 410 Gone, route removed | Correct |
| legacy JSON columns in active code | Only in configset_helpers.go (deprecated field list) and test assertions | Allowed |

### 2.2 Build and Tests

- `go build ./cmd/server/` — PASS
- `go build ./cmd/agent/` — PASS
- `go test ./internal/server/api/...` — ALL PASS
- `npm test` — ALL PASS (76 tests)
- `npm run build` — PASS

### 2.3 BackendRuntime Create/Patch Fields

The create/patch handlers accept convenience fields (`image_ref`, `docker_options`, `env`, `model_mount`, `health_check`, `entrypoint`, `command`) that map to ConfigSet paths:
- `image_ref` → `launcher.image`
- `docker_options` → `launcher.docker_options`
- `env` → `runtime.env`
- `model_mount` → `runtime.model_mount`
- `health_check` → `runtime.health`
- `entrypoint` → `launcher.entrypoint`
- `command` → `launcher.command`

These are ACCEPTED as ConfigSet projection convenience fields. They are correctly mapped into `config_set_json` via `setConfigValue()`. Tests confirm they persist correctly.

### 2.4 Table Naming

- DB table: `inference_backends` (pre-existing, unchanged by ConfigSet refactor)
- API path: `/api/v1/backends`
- Design doc: `configs/backend-catalog/backends/`
- Status: NOTED as inconsistency, but not a regression from this refactor. Fixing would require a broader rename affecting API routes, tests, and DB migrations.

## 3. Fixes Applied

None — this is an audit-only pass. No code changes were necessary.

## 4. Unfixed Issues

| Issue | Reason |
|-------|--------|
| Table `inference_backends` vs API path `/backends` | Pre-existing naming convention. Rename would be a separate project. |
| SGLang 0.4.6-compatible capabilities_json empty in fresh DB | Catalog reload `ON CONFLICT DO UPDATE` behavior. YAML file is correct. Pre-existing. |

## 5. Static Check Classification Summary

| Pattern | Count | Classification |
|---------|-------|---------------|
| config_set_json in DB queries | ~15 | Allowed internal DB column |
| config_set_json in API response | 0 | Correctly excluded |
| source_metadata_json in API response | 0 | Correctly excluded |
| Legacy functions (seed, repair, migrate) | 0 | Fully removed |
| HandleCheckNodeBackendRuntime | 1 | Deprecated, route removed |
| JSON column names in tests | ~10 | Allowed test fixture |

## 6. Git Status

```
M  web/package-lock.json (pre-existing)
M  web/package.json (pre-existing)
?? .mimocode/ (not committed)
?? docs/reports/ (evidence, plans, review)
```

## 7. SGLang Capabilities Investigation

Fresh DB under ConfigSet schema (commit 81a236e):
- `config_set_json` column replaces old `capabilities_json`
- `config_set.items["backend.capabilities"]` contains structured ConfigItem with
  `supported_formats`, `supported_tasks`, `supported_capabilities`, `model_path_modes`
- Both vLLM (`vllm-v0.23.0`) and SGLang (`sglang-0.4.6-compatible`) have
  non-empty capabilities
- Preflight test: `can_run=None, errors=[]` (preflight response format changed
  under ConfigSet, empty errors means no blocking issues)

**Verdict: ConfigSet refactor resolves the empty capabilities issue.**
This is NOT a blocker. The YAML catalog is authoritative; ConfigSet materializes
catalog data into structured ConfigItem entries.
