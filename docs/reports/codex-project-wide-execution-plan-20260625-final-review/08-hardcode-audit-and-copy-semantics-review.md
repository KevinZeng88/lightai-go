# Hardcode Audit & Copy Semantics Review (Evidence)

## Hardcode Classification (grep: vllm|sglang|llamacpp|llama.cpp)

### A. Catalog/config data (ALLOWED)
- configs/backend-catalog/backends/*.yaml — Backend family definitions
- configs/backend-catalog/versions/*.yaml — Version definitions with parameters, entrypoint, health
- configs/backend-catalog/runtimes/*.yaml — Runtime templates with image, docker, vendor
**Status**: PASS — catalog data is configuration, not code

### B. Test fixture (ALLOWED)
- internal/server/api/nbr_deployable_test.go — vllm image references in test setup
- internal/server/api/*_test.go — backend-specific test fixtures
**Status**: PASS — test fixtures are acceptable

### C. Docs/example (ALLOWED)
- docs/RUNBOOK-LOCAL-VERIFY.md — vLLM/SGLang/llama.cpp documentation
- docs/engineering/bootstrap/ — bootstrap tool documentation
**Status**: PASS — documentation is acceptable

### D. Backend adapter abstraction (ALLOWED)
- internal/server/runplan/profiles.go — BackendFamily-based process start profiles
- internal/server/runplan/compat.go — Format-backend compatibility validation
- internal/server/runplan/detection.go — Entrypoint detection by binary name patterns
**Status**: PASS — backend-family adapter pattern, data-driven

### E. Business logic hardcoding (NONE FOUND)
No backend-specific if/else in deployment lifecycle, preflight, or handler code
that hardcodes image/port/volume/device/env values.
**Status**: PASS

### F. Seed hardcoding (EXISTS — needs drift protection)
- internal/server/db/db.go:1230-1330 — seedBuiltInBackends() legacy entries
- internal/server/db/db.go:1380-1400 — seedTargetBackendCatalog() current entries
**Status**: ACCEPTED_WITH_DRIFT_TEST — drift test added (8c0d31a)

### G. Runtime smoke bypass (NONE FOUND)
No scripts directly execute `docker run` to bypass platform configuration.
**Status**: PASS

### H. Legacy compatibility fallback (EXISTS — deprecated versions)
- db.go seed includes deprecated versions (bver-vllm-0.8.5, bver-sglang-0.4.6, etc.)
- These are marked is_deprecated=1 and excluded from active use
**Status**: ACCEPTED — historical only, no active use

## Catalog Authority

**Decision**: YAML catalog (`configs/backend-catalog/`) is AUTHORITATIVE.
db.go seed (`seedTargetBackendCatalog`, `seedBuiltInBackends`) is a BOOTSTRAP MIRROR.

Drift protection: `TestCatalogSeedDrift` and `TestCapabilitiesNotArrayFormat` verify consistency (8c0d31a).

## Copy-on-Create Verification

| Chain | Mechanism | Test |
|-------|-----------|------|
| Backend→BackendVersion | YAML catalog → DB via upsertBackendVersionProjection | TestCatalogSeedDrift |
| BackendVersion→BackendRuntime | HandleCreateBackendRuntimeFromTemplate copies default_images, default_env_json, parameter_defs | Existing Go tests |
| BackendRuntime→NBR | buildRuntimeConfigSnapshot freezes config_snapshot_json at enable time | TestNodeBackendRuntimeCheckDoesNotRefreshSnapshot |
| NBR→Deployment | buildDeploymentRuntimeSnapshot freezes BR+NBR config at create time | TestWorkflowDeploymentRunPlanPreservesNBRSnapshot |
