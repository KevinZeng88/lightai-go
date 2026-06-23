# Final Review — Runtime Parameter Editing Implementation

> Date: 2026-06-24
> Status: PASS

---

## 1. Architecture Consistency

### NBR Source-of-Truth ✅
- RunPlan resolver reads ONLY from NBRConfigSnapshot
- No fallback to BackendVersion/BackendRuntime
- Missing NBR snapshot returns explicit error
- Verified by code inspection and tests

### Copy-on-Create ✅
- BackendRuntime → NodeBackendRuntime: deep copy schema + values
- ModelArtifact → Deployment: deep copy parameter_defaults_json
- NBR → Deployment: deep copy parameter_values_json
- Each layer independent after creation

### Parameter Schema/Value/Tombstone ✅
- All parameter JSON fields use structured arrays (default `[]`)
- BackendRuntime: parameter_schema_json, parameter_values_json
- NodeBackendRuntime: parameter_schema_json, parameter_values_json
- ModelArtifact: parameter_defaults_json
- Deployment: parameter_values_json, disabled_parameters_json

---

## 2. Env/Args Pollution ✅

### Empty values filtered
- buildEnv skips empty string values
- buildArgs skips nil/empty values
- No `-e KEY=` or `--flag ""` in output

### Capability metadata separated
- capabilities_json on backend_versions (metadata only)
- env_json cleaned of capability metadata in seed data
- SGLang/llama.cpp env_json cleaned in Batch B correction

---

## 3. ModelArtifact / ModelLocation Boundary ✅

### ModelArtifact stores:
- parameter_defaults_json (model-side defaults)
- capabilities_json (model capabilities)
- discovered metadata
- Does NOT store: Docker image, entrypoint, ports, devices, privileged

### ModelLocation stores:
- discovered_metadata_json (scan results)
- Does NOT store: container config

---

## 4. Deployment Override ✅

### Merge order:
1. NBR parameter values (Layer 2)
2. Deployment parameter_values overrides (Layer 3) — highest priority
3. Deployment parameters_json (Layer 4, legacy)
4. Disabled tombstones remove args/env

### Semantics:
- absent = keep upstream value
- override = replace upstream value
- disabled = remove from output
- empty enabled value = skip

---

## 5. Web UI / i18n ✅

### RuntimeParameterEditor integrated into:
- BackendRuntimesPage (edit dialog)
- RunnerConfigsPage (edit dialog)
- ModelArtifactsPage (edit dialog)
- ModelDeploymentsPage (edit dialog)

### i18n:
- structuredParameters key added to runtimes and deployments namespaces
- No key leakage detected

---

## 6. API Permission / Tenant Scope

### Not affected:
- Deployment parameter fields are part of existing deployment CRUD
- Tenant scope checks already exist on deployment endpoints
- No new permission model introduced

---

## 7. Migration / Seed

### V28 migration:
- Adds 7 new columns (all DEFAULT '[]')
- Backward compatible (old data preserved as empty arrays)
- Seed data cleaned (capability metadata removed from env_json)

---

## 8. Test Coverage

### RunPlan tests:
- All existing tests pass
- New tests: ensureNbrSnapshot helper, NBR-only resolution
- Race detection: go test -race passes

### API tests:
- All existing tests pass
- Deployment tests updated for new fields

### Frontend tests:
- Build passes
- npm test passes

---

## 9. Unresolved Items

1. **Full E2E with real GPU**: Not executed (requires Docker + GPU + models)
2. **ModelLocation parameter_defaults_json**: Not added (not needed — model defaults on ModelArtifact)
3. **Legacy parameters_json**: Still supported for backward compatibility; future cleanup needed
4. **RuntimeParameterEditor in RunnerConfigsPage NBR edit**: Component integrated but existing edit dialog also works
