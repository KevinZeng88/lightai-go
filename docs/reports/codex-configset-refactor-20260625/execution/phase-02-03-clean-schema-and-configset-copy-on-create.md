# Phase 02-03: Clean Schema And ConfigSet Copy-On-Create

## Scope

Combined Checkpoint B and Checkpoint C were completed in one clean-state implementation range because the user explicitly rejected any committed transition state with old authority columns, additive compatibility migrations, or dual ConfigSet/legacy field authority.

## Result

PASS in worktree before commit.

## Changed Areas

| Area | Result |
| --- | --- |
| Config registry | Added `configs/config-registry/items.yaml` as the ConfigItem registry. |
| Catalog loader | Added `internal/server/catalog` with registry/catalog loading, validation, materialization, and clean DB seeding. |
| DB schema | Replaced V1->V28 replay initialization with a fresh ConfigSet schema baseline. `schema_version=100` is a clean baseline marker only. |
| Backend catalog | `inference_backends`, `backend_versions`, and `backend_runtimes` use `config_set_json` and `source_metadata_json` as configuration authority. |
| Node runtime | `node_backend_runtimes` uses ConfigSet copy-on-create and source metadata. Enable writes explicit image selection into `launcher.image`. |
| Deployment | `model_deployments` freezes `config_set_json` and stores user edits in `config_overrides_json`. Dry-run/start use the frozen deployment ConfigSet, not live NBR image columns. |
| Model artifacts | Capability data moved to `capability_set_json`. |
| RunPlan input | Runtime resolution consumes ConfigSet-derived image, command, entrypoint, env, Docker options, model mount, health, and parameter values. |
| Tests | API fixtures and contract tests were updated to current ConfigSet payloads and clean schema. |

## Deleted Old Structures

- Active `migrateV1` through `migrateV28` replay path.
- Active old authority-field `ALTER TABLE ADD COLUMN` migration chain.
- `seedBuiltInBackends`, `seedTargetBackendCatalog`, `repairBackendCapabilitiesV27`, and `normalizeLegacyBackendCatalogIDs` active seed/repair/normalize paths.
- Old backend catalog projection SQL that wrote old BackendVersion/BackendRuntime columns.
- Old deployment snapshot helpers that read `config_snapshot_json` and old runtime field names.
- Old `backendVersionSnapshot` helper.
- Old API/RunPlan/DB exact authority field references in the active implementation scope.

## New Structures

- `configs/config-registry/items.yaml`
- `internal/server/catalog/types.go`
- `internal/server/catalog/loader.go`
- `internal/server/catalog/loader_test.go`
- Fresh DB clean schema in `internal/server/db/db.go`
- ConfigSet helper functions in `internal/server/api/configset_helpers.go`

## Validation

| Command | Result | Summary |
| --- | --- | --- |
| `go test ./internal/server/api -count=1` | PASS | API package passes with ConfigSet schema and copy-on-create tests. |
| `go test ./internal/server/api ./internal/server/runplan -count=1` | PASS | API and RunPlan packages pass. |
| `go test ./internal/server/catalog ./internal/server/db -count=1` | PASS | Catalog loader tests pass; DB package builds. |
| `go build ./cmd/server/... && go build ./cmd/agent/...` | PASS | Server and Agent build. |
| `go test ./...` | PASS | All Go packages pass. |
| Exact old authority field static gate over `internal/server/api internal/server/runplan internal/server/db` | PASS | No exact old field, old seed/repair, or `migrateVx` hits remain in active B/C scope. |
| `git diff --check` | PASS | No whitespace errors. |

## Remaining Work For Later Checkpoints

The active B/C implementation scope is clean. Checkpoint D must continue renderer / AgentRunSpec / DockerSpec convergence. Checkpoint E must clean stale UI/docs/scripts/OpenAPI references and archive stale evidence as planned.
