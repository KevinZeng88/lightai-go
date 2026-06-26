# ConfigSet Refactor — Final Closeout

Status: **CLOSED**
Date: 2026-06-26

## Commits

| Commit | Description |
|--------|-------------|
| dee0dd8 | refactor: replace catalog seeds with configset snapshots |
| 6935951 | refactor: render runplans from configsets |
| a822ac3 | refactor: migrate api ui to configsets |
| bbe0686 | test: validate configset runtime smoke |
| 404aeff | docs: close configset refactor |
| 81a236e | docs: add post-codex claude audit report |
| fc3ad97 | docs: finalize configset audit — sglang capabilities resolved |

## Resolved

- Catalog seed → ConfigSet materialization from YAML
- RunPlan resolver uses ConfigSet snapshots
- API/UI migrated to ConfigSet fields
- Legacy functions removed (seedBuiltInBackends, seedTargetBackendCatalog, repairBackendCapabilitiesV27, migrateV1-V28)
- SGLang capabilities properly materialized (config_set.items["backend.capabilities"])
- Public API does not expose config_set_json or source_metadata_json

## Open (Non-blocking)

- Table `inference_backends` vs `/api/v1/backends` naming inconsistency (pre-existing)
