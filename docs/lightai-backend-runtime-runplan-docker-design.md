# Backend Runtime / RunPlan / Docker Semantics

This document records the layered snapshot inheritance model and RunPlan consistency rules.

## Snapshot Inheritance Model (Chain-Copy)

Each layer copies ALL config from its parent at creation time, then is fully decoupled. Editing a parent does NOT affect children. Only explicit manual sync can pull updates.

```
BackendVersion  ──create──▶ BackendRuntime (captures version_snapshot_json)
                                     │
                                     │  create
                                     ▼
                            NodeBackendRuntime (captures config_snapshot_json)
                                     │
                                     │  create (with node_id)
                                     ▼
                              Deployment (captures config_snapshot_json, merged with NBR overrides)
                                     │
                                     │  start
                                     ▼
                                RunPlan (immutable, stored as plan_json)
                                     │
                                     │  agent execute
                                     ▼
                            ModelInstance / Docker Create Spec
```

### Layer 1: BackendVersion → BackendRuntime

- BackendRuntime captures `version_snapshot_json` (frozen BackendVersion config) at creation time.
- BackendRuntime stores its own independent config fields (image, args, env, docker, mounts).
- Editing BackendVersion or reloading catalog does NOT affect existing BackendRuntime records.
- Editing BackendRuntime does NOT modify the source BackendVersion.

### Layer 2: BackendRuntime → NodeBackendRuntime

- NodeBackendRuntime captures `config_snapshot_json` (frozen BackendRuntime config) at first enable.
- NodeBackendRuntime stores node-specific overrides (image_ref, display_name, model_root).
- Editing BackendRuntime does NOT affect existing NodeBackendRuntime records.
- Editing NodeBackendRuntime does NOT modify the source BackendRuntime.

### Layer 3: NodeBackendRuntime → Deployment

- Deployment captures `config_snapshot_json` at creation time from the BackendRuntime.
- If placement specifies a target node AND a NodeBackendRuntime exists, the NBR's config_snapshot_json and image_ref are merged into the deployment snapshot.
- Deployment stores its own service config (ports, parameters, env overrides, placement).
- preflight/DryRun/Start use ONLY the deployment's config_snapshot_json. There is no live re-read of NBR config.
- Editing NodeBackendRuntime after Deployment creation does NOT affect Deployment DryRun/Start behavior.
- Editing Deployment does NOT modify the source NodeBackendRuntime or BackendRuntime.

### Layer 4: Deployment → RunPlan

- RunPlan is generated at instance start time from the deployment's complete config snapshot.
- RunPlan is stored immutably in `resolved_run_plans.plan_json`. It is never re-derived.
- Editing Deployment after RunPlan generation does NOT affect historical RunPlans.

### Layer 5: RunPlan → Docker Spec / ModelInstance

- Docker create spec is derived from the stored RunPlan (plan_json), never from live data.
- The Agent reads the frozen plan, converts to AgentRunSpec, and builds Docker ContainerCreateOptions.
- ModelInstance references the RunPlan. Deleting/editing Deployment or Runtime does not affect running instances.

## Manual Sync (Explicit Only)

- The template sync feature (preview + apply) is the only mechanism to pull updates from a parent template.
- There is NO automatic sync. Parent edits never silently propagate to children.
- Sync produces a diff preview. User must explicitly confirm before changes are applied.
- Sync does not affect running instances or historical RunPlans.

## Object Responsibilities

- ModelArtifact stores model identity and user-facing metadata. `display_name` is for UI selection. `name` is the platform artifact name. `path` is the original source path.
- ModelLocation stores the node-specific model path: `model_root`, `relative_path`, and `absolute_path`.
- Backend and BackendVersion are system/catalog capability layers and remain read-mostly.
- BackendRuntime is a user-manageable runtime template. System templates must be cloned before editing.
- NodeBackendRuntime is a node-level runtime config with its own `display_name`, image evidence, readiness, and frozen runtime snapshot.
- Deployment stores the service-level config: model artifact, runtime config, placement, service ports, deployment parameters, env overrides, and frozen config snapshot.
- RunPlan is immutable once generated. Later Deployment or Runtime edits affect only future RunPlans.
- ModelInstance stores one concrete run: task, container, health, logs, status, and last error.

## Field Semantics

- `display_name`: user-visible label for model/runtime/config selection.
- `artifact_name` / `name`: stable internal artifact or config name.
- `source_path`: node/source path used to discover the model.
- `mount_path`: path mounted inside the container.
- `served_model_name`: optional OpenAI-compatible model id.
- `backend_model_arg`: backend-specific model path/name argument, usually the container mount path.

## Port Semantics

- `host_port`: host access port, for example `8005`.
- `container_port`: Docker exposed container port, for example `8080`.
- `app_port`: backend process listening port. Current templates use `container_port`; advanced edits should keep `app_port` and `container_port` aligned.
- `health_port`: host-side health probe port. Defaults to `host_port`.
- `api_test_port`: host-side model test port. Defaults to `host_port`.

RunPlan JSON, Docker command preview, and Agent Docker create spec must all use the same effective `host_port` and `container_port`.

## Config Resolution at Start Time

At start/dry-run time (preflightDeployment), the deployment's `config_snapshot_json` (with merged NBR overrides captured at creation time) is the SOLE source of runtime configuration. The resolution order within the deployment snapshot:

1. Deployment explicit parameters and env overrides.
2. NBR frozen config (merged into deployment snapshot at creation).
3. BackendRuntime frozen config (captured in deployment snapshot).
4. BackendVersion frozen snapshot (captured in BackendRuntime, carried into deployment snapshot).

There is NO live re-read of BackendRuntime, BackendVersion, or NodeBackendRuntime config during preflight/start/dry-run. Node-specific runtime discoveries (GPU IDs, node IP) are read live at start time but are not config parameters.

## Run Idempotency

The server enforces single-active-instance per deployment. `pending`, `starting`, `provisioning`, `running`, `healthy`, and `stopping` block duplicate start requests with HTTP 409. `failed`, `stopped`, and `saved` can start a new RunPlan and ModelInstance while preserving history.

## Runtime Parameter Editing Contract

### NBR is the Source of Truth for Runtime Parameters

**Hard rule**: RunPlan resolution reads ONLY from:
1. NodeBackendRuntime snapshot (`parameter_schema_json` + `parameter_values_json`)
2. ModelArtifact/ModelLocation snapshot (`parameter_defaults_json`)
3. Deployment overrides (`parameter_values_json` + `disabled_parameters_json`)

RunPlan does NOT dynamically query BackendVersion or BackendRuntime at resolution time.

### Structured Parameter Records

All parameter JSON fields use structured arrays (default `[]`):
- `parameter_schema_json` — parameter definitions (name, type, default, required, cli_name)
- `parameter_values_json` — parameter values with enabled/disabled state
- `disabled_parameters_json` — explicit tombstones for disabled parameters

### Copy-on-Create Chain

```
BackendVersion (catalog/schema source)
    ↓ deep copy at creation
BackendRuntime (parameter_schema_json + parameter_values_json)
    ↓ deep copy at creation
NodeBackendRuntime (parameter_schema_json + parameter_values_json)
    ↓ deep copy at creation
Deployment (parameter_values_json + disabled_parameters_json)
```

Each layer is independent after creation. Parent edits do NOT propagate to children.

### Disabled Override Semantics

- Deployment can explicitly disable a parameter from upstream
- Disabled ≠ absent ≠ empty value
- Disabled parameters are saved as tombstone records in `disabled_parameters_json`
- Disabled parameters do NOT enter final RunPlan args/env
