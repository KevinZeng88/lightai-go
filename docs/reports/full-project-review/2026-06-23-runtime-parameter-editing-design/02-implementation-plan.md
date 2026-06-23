# Implementation Plan

> Date: 2026-06-23
> Purpose: Batch implementation plan for runtime parameter editing

---

## Batch A: Documentation Alignment & Parameter Model Solidification

### Goal
Align old documents, define parameter type taxonomy, confirm NBR snapshot as source of truth.

### Deliverables
- Update `docs/08-engineering-contracts.md` with parameter model
- Update `docs/lightai-backend-runtime-runplan-docker-design.md` with NBR-only RunPlan input
- Define parameter record structure in code comments

### Files
- `docs/08-engineering-contracts.md`
- `docs/lightai-backend-runtime-runplan-docker-design.md`

### No Schema Changes
This batch is documentation only.

---

## Batch B: Catalog / Seed / Schema Cleanup

### Goal
Separate capability metadata from Docker env. Define parameter schema structure.

### Deliverables
- Move capability metadata from `env_json` to `capabilities_json` in seed data
- Add `parameter_values_json` column to `backend_runtimes` and `node_backend_runtimes`
- Add `parameter_schema_json` column to `node_backend_runtimes` (NBR must save schema snapshot)
- Add `parameter_values_json` column to `model_deployments`
- Add `disabled_parameters_json` column to `model_deployments` (explicit disabled overrides)
- Update seed data to populate structured parameter values

### Files
- `internal/server/db/db.go` — seed data, migrations
- `configs/backend-catalog/versions/*/` — YAML cleanup

### Schema Changes
```sql
ALTER TABLE backend_runtimes ADD COLUMN parameter_schema_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE backend_runtimes ADD COLUMN parameter_values_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE node_backend_runtimes ADD COLUMN parameter_schema_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE node_backend_runtimes ADD COLUMN parameter_values_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_deployments ADD COLUMN parameter_values_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_deployments ADD COLUMN disabled_parameters_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE model_artifacts ADD COLUMN parameter_defaults_json TEXT NOT NULL DEFAULT '[]';
```

### DB Rebuild
**Yes** — existing DB has polluted `env_json` with capability metadata. Rebuild recommended.

---

## Batch C: BackendRuntime / NodeBackendRuntime Parameter Editing

### Goal
Enable structured parameter editing at BR and NBR levels with enabled/disabled state.

### Deliverables
- Extract `RuntimeParameterEditor` component from `BackendRuntimesPage.vue`
- Add parameter editing to `RunnerConfigsPage.vue` (NBR)
- Implement copy-on-create: BR → NBR deep copy of parameter values
- Update RunPlan resolver to read from NBR `parameter_values_json`

### Files
- `web/src/components/RuntimeParameterEditor.vue` — new reusable component
- `web/src/pages/BackendRuntimesPage.vue` — refactor to use new component
- `web/src/pages/RunnerConfigsPage.vue` — add parameter editing
- `internal/server/api/runtime_handlers.go` — handle parameter_values_json
- `internal/server/api/node_runtime_handlers.go` — handle parameter_values_json
- `internal/server/runplan/resolver.go` — read from NBR snapshot

### Tests
- Unit: parameter value serialization/deserialization
- Unit: copy-on-create produces correct snapshot
- Unit: enabled/disabled state filtering
- Integration: BR edit → NBR create → RunPlan preview

---

## Batch D: ModelArtifact / ModelLocation Parameter Editing

### Goal
Enable model-level default parameter editing.

### Deliverables
- Add parameter editing to `ModelArtifactsPage.vue`
- Model parameters stored as `parameter_defaults_json` on model_artifacts
- Deployment creation deep-copies model defaults

### Files
- `web/src/pages/ModelArtifactsPage.vue` — add parameter editor
- `internal/server/api/artifact_handlers.go` — handle parameter_defaults_json
- `internal/server/models/artifact.go` — add field

### Schema Changes
```sql
ALTER TABLE model_artifacts ADD COLUMN parameter_defaults_json TEXT NOT NULL DEFAULT '[]';
```

---

## Batch E: Deployment Parameter Override

### Goal
Enable deployment-level parameter override with disabled state.

### Deliverables
- Add `RuntimeParameterEditor` to deployment edit dialog
- Show source/default/override for each parameter
- Deployment disabled parameters excluded from RunPlan
- Final RunPlan preview shows merged result

### Files
- `web/src/pages/ModelDeploymentsPage.vue` — add parameter editor to edit dialog
- `internal/server/api/deployment_lifecycle_handlers.go` — merge logic
- `internal/server/runplan/resolver.go` — respect disabled state

---

## Batch F: Backend-Specific Memory/Resource Parameters

### Goal
Expose backend-specific memory/resource parameters in UI.

### Deliverables
- Define parameter schema for vLLM, SGLang, llama.cpp memory/resource params
- Add to catalog YAML and seed data
- UI groups under "显存 / 上下文 / 并发 / 批处理"
- RunPlan maps to correct CLI args

### Files
- `configs/backend-catalog/versions/vllm/vllm-v0.23.0.yaml` — add parameter defs
- `configs/backend-catalog/versions/sglang/sglang-v0.5.13.post1.yaml` — add parameter defs
- `configs/backend-catalog/versions/llamacpp/llamacpp-b9700.yaml` — add parameter defs
- `internal/server/db/db.go` — update seed data

### Tests
- Unit: vLLM parameter mapping
- Unit: SGLang parameter mapping
- Unit: llama.cpp parameter mapping
- Integration: create deployment with memory params → RunPlan preview correct

---

## Execution Order

```
Batch A (docs) → Batch B (schema) → Batch C (BR/NBR editing)
                                        ↓
                                   Batch D (model editing)
                                        ↓
                                   Batch E (deployment override)
                                        ↓
                                   Batch F (memory params)
```

Batches A and B are prerequisites. C/D/E/F can be done sequentially.

---

## Commit Strategy

| Batch | Commits |
|-------|---------|
| A | 1: docs update |
| B | 2: migration + seed cleanup |
| C | 3: component extraction, NBR editing, resolver update |
| D | 2: model parameter editing |
| E | 2: deployment parameter override |
| F | 3: catalog updates, UI grouping, tests |
