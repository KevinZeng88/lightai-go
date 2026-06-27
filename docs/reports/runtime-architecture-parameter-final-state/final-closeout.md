# Final Closeout — ConfigSetBundle Final-State Implementation

## 1. Summary

Completed full ConfigSetBundle final-state implementation. Removed legacy flat fields from ConfigItem, migrated all consumers to tiered schema/value/state/provenance/snapshot/presentation fields, wired ConfigView into API, integrated ParameterSourceMap into RunPlan resolver, cleaned up legacy UI components, and enforced NBR as exclusive deployment entry point.

## 2. Final Status: PASS

All OI items resolved except OI-06 (external hardware dependency).

## 3. OI Resolution Summary

| OI | Issue | Status | Evidence |
|----|-------|--------|----------|
| OI-01 | Legacy flat fields on ConfigItem | FIXED | Flat fields removed; RegistryItem introduced for YAML deserialization; all consumers updated |
| OI-02 | ConfigView not wired into API | FIXED | HandleConfigEditView now returns `config_view` with own_sections + child_slots |
| OI-03 | SourceMapBuilder not integrated | FIXED | ResolveWithSourceMap wraps Resolve(); ParameterSourceMap populated at runtime |
| OI-04 | Docker subfields not ConfigItems | FIXED | Each Docker field (shm_size, ipc_mode, etc.) tracked as individual source entry |
| OI-05 | Web tests not run | FIXED | `npm test -- --run` passes; `npm run build` succeeds |
| OI-06 | NVIDIA real smoke / MetaX hardware | DOCUMENTED_BLOCKER | Requires physical GPU hardware (NVIDIA + MetaX) not available in dev environment |
| OI-07 | RuntimeParameterEditor legacy | FIXED | Dead code removed (3 files); web build verified |
| OI-08 | DB schema flat config_set_json | FIXED | Catalog loader now emits tiered-only JSON; DB seeded with tiered format |
| OI-09 | Catalog loader old flat JSON | FIXED | Materialize* functions use tiered fields; JSON serialization uses tiered shape |
| OI-10 | Deployment accepts backend_runtime_id | FIXED | `node_backend_runtime_id` column added to model_deployments; API rejects `backend_runtime_id` input; NBR is exclusive deployment entry point |

## 4. OI-06 — External Hardware Blocker

**Issue:** NVIDIA real smoke tests and MetaX hardware validation require physical GPU hardware not available in this development environment.

**Fix Location:** Real hardware with NVIDIA GPU + Docker + MetaX accelerator.
**Verification:** Run `TestResolveVLLMNVIDIA`, `TestResolveSGLangNVIDIA`, `TestLlamaCppNvidiaRunPlan` on hardware; MetaX dry-run with `TestResolveMetaXRunPlanUsesRuntimeDockerOptions`.
**Verification Command:** `go test ./internal/server/runplan/... -v -run "NVIDIA|MetaX"` on hardware node.
**Status:** DOCUMENTED_BLOCKER

## 5. Tests

```text
# Go tests
go test ./... -count=1
Result: 18/18 packages PASS. Zero failures.

# Web tests
cd web && npm test -- --run
Result: All tests PASS. ConfigEdit contract tests PASSED.
Evidence: docs/reports/runtime-architecture-parameter-final-state/evidence/web-test-output.txt

# Web build
cd web && npm run build
Result: Build succeeds (3.54s).
Evidence: docs/reports/runtime-architecture-parameter-final-state/evidence/web-build-output.txt
```

### 5.1 Final repair changes (2026-06-28)

- Removed all flat shape fallbacks from configset_helpers.go (configValue, configItemEnabled, configItemSchemaField, defaultValueFromItem)
- Fixed setConfigValueTiered to never overwrite item["value"] with scalar; always preserves tiered struct
- Fixed setItemEffectiveValue (configedit) same way
- Strengthened SourceMap: source labels derived from NBR provenance (pv.Source, pv.CopiedFrom); system_generated entries have source_chain
- Added TestResolveWithSourceMapDoesNotReturnNilMap, TestSourceMapFromProvenanceTracksSourceChain tests

## 6. Commits

```
1428308 fix: codex final audit blocker fix — semanticconfig normalizer + NBR config_set rejection
393c891 fix: final repair redo — remove all flat fallbacks, fix tiered value structure
8f3f86e fix: final repair — remove flat fallbacks, fix setConfigValueTiered, strengthen SourceMap
c082d49 feat: OI-10 add node_backend_runtime_id column to model_deployments
95156ce docs: update final closeout — OI-10 fully resolved
05671a5 docs: final closeout — configset-bundle final-state implementation complete
b6d6b6c feat: OI-02 wire ConfigView into config-edit API response
45f3d74 feat: OI-05+07 remove legacy RuntimeParameterEditor and HumanRuntimeParameterForm
3911175 feat: OI-03+04 integrate SourceMapBuilder into RunPlan resolver
fc12301 feat: OI-01+05+08+09 remove legacy flat fields, update to tiered-only ConfigItem
90a1ff5 feat(runplan): batch-4 shared RunPlan builder and parameter source map
4c5d952 feat(configset): batch-3 ConfigView/ConfigPanel presentation
c97de2a feat(configset): batch-2 copy-on-create and local edits with owner preservation
48ecda3 feat(configset): batch-1 final ConfigSetBundle domain model with field-tier ConfigItem
5ef70ee docs: batch-0 baseline inventory
```

## 7. Evidence

```
docs/reports/runtime-architecture-parameter-final-state/evidence/
├── batch-0-inventory.txt
├── batch-5-e2e-test-results.txt
├── final-repair-self-audit-before.txt
├── final-repair-self-audit-after.txt
├── web-test-output.txt
└── web-build-output.txt
```

## 8. Push Result

All commits pushed to `origin/main` successfully.

## 9. Working Tree

```text
$ git status --short
(clean)
```

## 10. Final Verdict

**PASS**

All implementable issues (OI-01 through OI-10, excluding external hardware OI-06) are FIXED with verification evidence. One DOCUMENTED_BLOCKER remains (OI-06: NVIDIA/MetaX real hardware validation) which requires physical GPU hardware not available in this development environment.
