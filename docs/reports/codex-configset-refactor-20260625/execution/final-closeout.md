# ConfigSet Refactor Final Closeout

## Final Status

`CONFIGSET_REFACTOR_COMPLETE`

## Design Document

`docs/design/catalog-configset-and-runtime-snapshot.md`

## Execution Reports

- `phase-01-design-and-inventory.md`
- `phase-05-api-ui-docs-configset.md`
- `phase-06-final-validation-runtime-smoke.md`
- `validation-log.md`
- `open-issues.md`
- `commit-log.md`
- `db-migration-compatibility-audit.md`

## Removed Old Structures

Old authority fields removed from active ConfigSet runtime/catalog/API/script scopes:

- `capabilities_json`
- `parameter_schema_json`
- `parameter_values_json`
- `env_json`
- `ports_json`
- `volumes_json`
- `devices_json`
- `health_check_json`
- `resource_controls_json`
- `parameters_json`
- `default_args_json`
- `parameter_defs_json`
- `default_backend_params_json`
- `default_images_json`
- `image_candidates_json`
- `docker_options_json`
- `model_mount_json`

Old DB/API compatibility paths removed from active DB/API/RunPlan initialization:

- V1 to V28 active migration replay path;
- V29 additive compatibility migration approach;
- old authority-field ADD COLUMN/backfill/repair path;
- hardcoded `seedBuiltInBackends`;
- hardcoded `seedTargetBackendCatalog`;
- `repairBackendCapabilitiesV27`;
- `normalizeLegacyBackendCatalogIDs`;
- deployment create/preflight/update payload fallback for bare `backend_runtime_id`;
- client-trusted readiness evidence in NBR enable/check payloads.

## Registry And Catalog

- Config registry is loaded from `configs/config-registry/items.yaml`.
- Backend catalog is loaded from `configs/backend-catalog`.
- Runtime/version config is materialized into `config_set_json` and `source_metadata_json`.
- Backend catalog capability detail uses `capabilities_detail`, not old DB column naming.

## Copy-On-Create And Renderer

- BackendVersion to BackendRuntime to NodeBackendRuntime to Deployment uses ConfigSet copy-on-create snapshots.
- Deployment `config_overrides` are applied into the copied deployment ConfigSet.
- RunPlan and AgentRunSpec are rendered from resolved ConfigSet data.
- Repeat flags and runtime-specific argument styles are preserved.
- NBR probe-derived process start detection is applied before Docker spec rendering.

## Runtime Smoke

Final run:

```text
configset-f-20260626061623
```

Evidence:

```text
docs/reports/model-runtime-node-wizard/e2e-matrix-configset-f-20260626061623
```

Result:

- vLLM: PASS
- SGLang: PASS
- llama.cpp: PASS

Each backend completed platform-chain health, inference, logs, stop, and cleanup.

## Validation Summary

All required validation commands passed:

- `go test ./...`
- `go build ./cmd/server/...`
- `go build ./cmd/agent/...`
- `cd web && npm test`
- `cd web && npm run build`
- OpenAPI current-path check
- active old-field/static stale gates
- fresh DB schema probe
- real runtime smoke for vLLM/SGLang/llama.cpp
- `git diff --check`

## Open Issues

No undocumented problems remain.

`CS-A-003` remains `DOCUMENTED_BLOCKER` for unrelated pre-existing workspace files. It is outside the ConfigSet refactor commit scope and is controlled by explicit pathspec staging.
