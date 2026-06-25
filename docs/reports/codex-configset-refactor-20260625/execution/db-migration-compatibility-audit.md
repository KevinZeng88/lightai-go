# DB Migration Compatibility Audit

Scope: Checkpoint B/C clean-schema correction.

Decision:

Fresh DB clean schema is the only accepted DB baseline. Active DB initialization must not replay the historical V1->V28 compatibility migration chain. `schema_version` may remain only as a clean-schema baseline marker and must not imply support for upgrading historical DB versions.

## migrateV1-migrateV28 Inventory

| Version | Function / Purpose | Classification | Old Fields / Structures Introduced | Old Repair / Backfill / Normalize Behavior | Removal Plan |
| --- | --- | --- | --- | --- | --- |
| V1 | `migrateV1`: initial tenants, users, memberships, roles, permissions, sessions, nodes, audit logs | collapse into clean schema | auth/RBAC/node/audit base tables | none | Keep current table definitions that still exist, collapse into `createConfigSetSchema`, delete `migrateV1`. |
| V2 | `migrateV2`: add node detail fields | legacy compatibility ADD COLUMN | `nodes.primary_ip`, `os`, `arch`, `kernel`, `agent_version` | additive ALTER TABLE chain | Collapse current node columns into clean schema, delete `migrateV2`. |
| V3 | `migrateV3`: model artifacts, runtime environments, runtime docker specs, run templates, deployments, instances, leases | mixed obsolete + collapse | `model_artifacts.capabilities_json`, old `runtime_environments`, `runtime_environment_docker_specs`, `run_templates`, old deployment fields | old runtime-template schema | Keep current artifact/deployment/instance/lease concepts only as clean ConfigSet tables; delete old runtime environment/run template tables and `migrateV3`. |
| V4 | `migrateV4`: add GPU lease fields | legacy compatibility ADD COLUMN | lease timestamps/status additions | additive ALTER TABLE chain | Collapse current lease columns into clean schema, delete `migrateV4`. |
| V5 | `migrateV5`: create agent_tasks | collapse into clean schema | initial task table | none | Collapse current task table into clean schema, delete `migrateV5`. |
| V6 | `migrateV6`: task lifecycle columns and instance tenant | legacy compatibility ADD COLUMN | `claimed_at`, `started_at`, `finished_at`, `agent_id`, `retry_count`, `model_instances.tenant_id` | additive ALTER TABLE chain | Collapse current columns into clean schema, delete `migrateV6`. |
| V7 | `migrateV7`: tenant type and resource pool tables | obsolete / collapse | `tenants.type`, resource pool tables | additive + obsolete resource pool support | Keep `tenants.type` if still current; delete obsolete resource pool tables unless current code proves active use; delete `migrateV7`. |
| V8 | `migrateV8`: bootstrap/RBAC adjustments | collapse into clean schema or delete | current auth/RBAC seed support | idempotent compatibility setup | Keep current auth/RBAC schema/seed separately if active; delete historical migration function. |
| V9 | `migrateV9`: runtime template extra args and docker spec volumes | legacy compatibility ADD COLUMN / obsolete | `run_templates.extra_args`, `runtime_environment_docker_specs.volumes`, `extra_args` | additive ALTER TABLE chain | Replace with ConfigSet `backend.extra_args` and docker option items; delete `migrateV9`. |
| V10 | `migrateV10`: backend catalog tables and runtime tables | replace with ConfigSet / catalog loader | `default_args_json`, `default_backend_params_json`, `parameter_defs_json`, `default_images_json`, `env_json`, `model_mount_json`, old runtime fields | old DB-backed catalog seed path | Collapse current non-config metadata into clean schema; move configuration to `config_set_json`; delete `migrateV10`. |
| V11 | `migrateV11`: task lease/generation/attempt fields | legacy compatibility ADD COLUMN | `lease_owner`, `lease_expires_at`, `operation_id`, `generation`, `attempt`, `max_attempts` | additive ALTER TABLE chain | Collapse current task columns into clean schema, delete `migrateV11`. |
| V12 | `migrateV12`: audit tenant and resource monitoring snapshots | mixed collapse + ADD COLUMN | `audit_logs.tenant_id`, `gpu_devices`, node system/filesystem/network snapshots | additive ALTER TABLE for audit/gpu reported_at | Collapse current observability/resource tables into clean schema, delete `migrateV12`. |
| V13 | `migrateV13`: catalog metadata columns, model locations, NBR, run_plan_groups, seed built-ins | replace with ConfigSet / delete | catalog `slug/managed_by/source/checksum/status`, NBR old snapshot fields, model locations, `seedBuiltInBackends` | old catalog seed literals | Keep current non-config metadata and model locations/NBR concepts in clean schema; replace config with ConfigSet; delete seed literals and `migrateV13`. |
| V14 | `migrateV14`: node metrics endpoint fields | legacy compatibility ADD COLUMN | `advertised_address`, `metrics_enabled`, `metrics_scheme`, `metrics_port`, `metrics_path` | additive ALTER TABLE chain | Collapse current node metrics fields into clean schema, delete `migrateV14`. |
| V15 | `migrateV15`: node model roots | collapse into clean schema | `node_model_roots` | none | Collapse current table into clean schema, delete `migrateV15`. |
| V16 | `migrateV16`: NBR config snapshot | replace with ConfigSet | `config_snapshot_json`, `source_runtime_name`, `source_runtime_revision` | snapshot compatibility | Replace with `node_backend_runtimes.config_set_json` and `source_metadata_json`; delete `migrateV16`. |
| V17 | `migrateV17`: backend version capabilities and runtime source snapshots | replace with ConfigSet | `capabilities_json`, `docker_options_json`, `model_mount_json`, `vendor_options_json`, `source_backend_id`, `source_backend_version_id`, `version_snapshot_json` | old runtime snapshot sync | Replace all config authority with ConfigSet/source metadata; delete `migrateV17`. |
| V18 | `migrateV18`: software catalog fields | mixed collapse + replace with ConfigSet | `image_candidates_json`, `default_endpoints_json`, `default_args_schema_json`, `default_env_schema_json`, `default_health_check_json`, `official_reference_json` | derives image candidates from old default images | Keep current metadata such as readonly/protocol/revision as columns; move config data into ConfigSet; delete `migrateV18`. |
| V19 | `migrateV19`: catalog load metadata | collapse into clean schema | `config_hash`, `loaded_from`, `loaded_at` | additive ALTER TABLE chain | Keep current non-config loader metadata where still useful; collapse into clean schema, delete `migrateV19`. |
| V20 | `migrateV20`: runtime hardware/vendor options | replace with ConfigSet / collapse metadata | `compatibility_json`, `image_candidates_json`, `devices_json`, `volumes_json`, `env_schema_json`, `args_schema_json`, `ports_json`, `high_risk_flags_json` | additive ALTER TABLE chain | Keep hardware/runtime distribution metadata as columns; move runtime options to ConfigSet; delete `migrateV20`. |
| V21 | `migrateV21`: NBR display name | legacy compatibility ADD COLUMN | `node_backend_runtimes.display_name` | additive ALTER TABLE chain | Collapse current display column into clean schema, delete `migrateV21`. |
| V22 | `migrateV22`: deployment config snapshot | replace with ConfigSet | `model_deployments.config_snapshot_json` | additive snapshot column | Replace with deployment `config_set_json` and `config_overrides_json`; delete `migrateV22`. |
| V23 | `migrateV23`: deployment source metadata | collapse into source_metadata | `source_backend_runtime_id`, `source_node_backend_runtime_id`, `source_template_name`, `source_template_version`, `source_config_hash`, `copied_at` | additive metadata chain | Keep required source metadata either as explicit columns or `source_metadata_json`; delete historical migration. |
| V24 | `migrateV24`: NBR probe results | collapse into clean schema | `probe_results_json` | additive ALTER TABLE chain | Collapse current probe results into clean schema, delete `migrateV24`. |
| V25 | `migrateV25`: model artifact capabilities | replace/rename cleanly | `model_artifacts.capabilities_json`, `capability_sources_json`, `default_test_mode` | additive capability columns | Replace old `capabilities_json` authority with `capability_set_json`; keep `default_test_mode` if current; delete `migrateV25`. |
| V26 | `migrateV26`: llama.cpp placeholder repair | delete | old `default_args_json` and snapshot text replacements | old-data backfill/repair | Delete; current YAML/ConfigSet must carry correct placeholders with tests. |
| V27 | `migrateV27` and `repairBackendCapabilitiesV27`: force backend capabilities | replace with catalog loader | `backend_versions.capabilities_json` updates | unconditional seed repair after migration | Delete; backend capabilities come from YAML catalog materialized into ConfigSet. |
| V28 | `migrateV28`: structured parameter schema columns | replace with ConfigSet | `parameter_schema_json`, `parameter_values_json`, `disabled_parameters_json`, `parameter_defaults_json` | additive ALTER TABLE chain | Replace with ConfigSet items and deployment `config_overrides_json`; delete `migrateV28`. |

## Seed / Repair / Normalize Inventory

| Function | Classification | Removal Plan |
| --- | --- | --- |
| `seedBuiltInBackends` | old catalog seed literal | Delete. Replace with YAML backend catalog + Config Registry loader. |
| `seedTargetBackendCatalog` | old catalog seed/repair literal | Delete. Replace with YAML backend catalog + Config Registry loader. |
| `repairBackendCapabilitiesV27` | catalog seed repair | Delete. Capabilities are ConfigSet materialized from YAML, not DB repair. |
| `normalizeLegacyBackendCatalogIDs` | old ID normalize compatibility | Delete. Fresh DB uses current IDs only; no legacy ID compatibility. |

## Final Evidence Commands And Results

Commands required by validation:

```bash
rg -n "func \\(db \\*DB\\) migrateV[0-9]+|migrateV[0-9]+\\(" internal/server/db
rg -n "ALTER TABLE .* ADD COLUMN|Backfill|backfill|repair|normalizeLegacy|compat|legacy" internal/server/db internal/server
rg -n "seedBuiltInBackends|seedTargetBackendCatalog|repairBackendCapabilitiesV27|normalizeLegacyBackendCatalogIDs" internal/server/db internal/server
```

Current result at audit creation:

- `migrateV1` through `migrateV28` exist in `HEAD:internal/server/db/db.go` and are classified above.
- The worktree clean-schema implementation is required to make the active `internal/server/db` checks return no active historical compatibility migration chain, no old catalog seed/repair functions, and no old authority field ADD COLUMN migrations before Checkpoint B/C implementation commit.

## Acceptance

Checkpoint B/C cannot be committed unless:

- fresh DB is initialized by a single clean schema path;
- V1->V28 active migration functions are deleted from active DB initialization;
- old authority-field ADD COLUMN migrations are gone;
- old-data backfill/repair/normalizeLegacy paths are gone;
- old catalog seed literals and seed repair functions are gone;
- ConfigSet/catalog loader is the only backend catalog configuration authority.
