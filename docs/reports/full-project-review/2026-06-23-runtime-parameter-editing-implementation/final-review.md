# Final Review — Runtime Parameter Editing Implementation (Corrected)

> Date: 2026-06-24
> Status: PASS

---

## 1. Architecture Consistency

### NBR Source-of-Truth ✅
- RunPlan resolver reads ONLY from NBRConfigSnapshot
- No fallback to BackendVersion/BackendRuntime
- Missing NBR snapshot returns explicit error
- Required parameters validated against NBR schema
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

### Empty values rejected
- Enabled parameter with empty value returns validation error
- No silent skip for enabled empty values
- Disabled parameters excluded from output
- No `-e KEY=` or `--flag ""` in output

### Capability metadata separated
- capabilities_json on backend_versions (metadata only)
- env_json cleaned of capability metadata in seed data

---

## 3. Legacy parameters_json

### Current status:
- `parameters_json` column still exists in DB (schema cleanup item)
- RunPlan resolver does NOT read from `parameters_json`
- Deployment creation still accepts `parameters_json` for backward API compatibility
- But `parameter_values_json` is the new structured parameter source
- `parameters_json` is NOT a supported runtime path — only `parameter_values_json` is used

---

## 4. Empty Enabled Value Semantics

### Current behavior:
- Enabled parameter with empty value → validation error
- Disabled parameter → excluded from output
- Absent parameter → keep upstream value
- This is the correct semantic: empty enabled value is an error, not a skip

---

## 5. ModelArtifact / ModelLocation Boundary ✅

### ModelArtifact stores:
- parameter_defaults_json (model-side defaults)
- capabilities_json (model capabilities)
- discovered metadata
- Does NOT store: Docker image, entrypoint, ports, devices, privileged

---

## 6. Deployment Override ✅

### Merge order:
1. NBR parameter values (Layer 2)
2. Deployment parameter_values overrides (Layer 3) — highest priority
3. Disabled tombstones remove args/env

### Semantics:
- absent = keep upstream value
- override = replace upstream value
- disabled = remove from output
- empty enabled value = validation error

---

## 7. Web UI / i18n ✅

### RuntimeParameterEditor integrated into:
- BackendRuntimesPage (edit dialog)
- RunnerConfigsPage (edit dialog)
- ModelArtifactsPage (edit dialog)
- ModelDeploymentsPage (edit dialog)

---

## 8. Test Coverage

### RunPlan tests:
- All existing tests pass
- New tests: ensureNbrSnapshot helper, NBR-only resolution
- Race detection: go test -race passes

### API tests:
- All existing tests pass
- Deployment tests updated for parameter_values_json

### Frontend tests:
- Build passes
- npm test passes

---

## 9. Unresolved Items

1. **Full E2E with real GPU**: Server start timed out in isolated test environment; requires manual verification
2. **Legacy parameters_json**: Column exists in DB but not used by resolver; schema cleanup item
3. **ModelLocation parameter_defaults_json**: Not added (not needed — model defaults on ModelArtifact)
