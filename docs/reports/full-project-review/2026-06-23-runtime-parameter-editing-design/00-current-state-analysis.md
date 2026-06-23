# Current State Analysis

> Date: 2026-06-23
> Purpose: Analyze existing parameter editing capabilities before designing new ones

---

## 1. Existing "Edit Runtime Template" Implementation

### Location
`web/src/pages/BackendRuntimesPage.vue` — inline dialog, not a reusable component.

### How It Works
- Edit dialog triggered by clicking "Edit" button on a BackendRuntime row
- Only for `is_editable` rows (system-managed rows show readonly alert)
- Fields: `display_name`, `image_name`, `vendor`
- High-risk options (privileged, ipc_mode, uts_mode, etc.) use inline `RuntimeOption` component (checkbox + input)
- List options (devices, env, extra_mounts, etc.) use inline `RuntimeTextarea` component (checkbox + textarea)
- Data flow: `showEdit(row)` → `loadDockerJson(row)` → hydrate from `row.docker_json` / `row.args_override_json`
- Build payload: constructs `docker_json`, `args_override_json`, `display_name`, `image_name`, `vendor`
- API: `PATCH /api/v1/backend-runtimes/{id}`

### Key Observation
The edit experience uses `{enabled, value}` pairs for Docker runtime options. This is the pattern to generalize.

---

## 2. Data Storage Summary

### BackendVersion (backend_versions table)
- `parameter_defs_json` — parameter schema (name, type, default, required, alias)
- `default_args_json` — template CLI args with `{{VAR}}`
- `env_json` — Docker env vars (currently polluted with capability metadata)
- `capabilities_json` — metadata only (supported_formats, tasks, etc.)
- `vendor_options_json` — resource_controls (gpu_memory_fraction, etc.)

### BackendRuntime (backend_runtimes table)
- `args_override_json` — CLI arg overrides
- `default_env_json` — env vars
- `docker_json` — Docker config (privileged, shm_size, devices, etc.)
- `entrypoint_override_json` — entrypoint override
- `version_snapshot_json` — frozen BackendVersion at creation
- NO structured parameter schema — all opaque JSON blobs

### NodeBackendRuntime (node_backend_runtimes table)
- `config_snapshot_json` — frozen BackendRuntime config at creation
- `image_ref` — actual Docker image on this node
- NO user-editable parameter fields

### ModelDeployment (model_deployments table)
- `parameters_json` — user parameter overrides (flat map: flag name → value)
- `env_overrides_json` — env overrides
- `config_snapshot_json` — frozen config at creation
- `source_node_backend_runtime_id` — NBR reference

---

## 3. RunPlan Resolver Input Chain

```
BackendVersion.default_args_json     (Layer 1)
BackendVersion.default_backend_params_json (Layer 2)
BackendRuntime.args_override_json    (Layer 3)
Deployment.parameters_json           (Layer 4, via mapParametersToArgs)
vendor_options.resource_controls     (Layer 4b)
→ deduplicateArgs → final args
```

**Problem**: RunPlan reads from BackendVersion AND BackendRuntime at resolution time. This violates the "NBR is source of truth" principle.

---

## 4. Web Wizard Parameter Editing

The deployment wizard (`ModelDeploymentsPage.vue`) does NOT expose parameter editing:
- `parameters_json` is always `{}`
- Parameters come entirely from `ParameterDef` defaults and `BackendVersion.default_args_json`
- User can only edit via deployment edit dialog (raw JSON textarea)

---

## 5. Key Gaps

1. **No structured parameter editing at any level** — all parameters are opaque JSON blobs
2. **No enabled/disabled state** — parameters are either present or absent
3. **No copy-on-create for parameter values** — NBR only snapshots Docker config, not parameter values
4. **RunPlan reads BackendVersion/BackendRuntime at resolution time** — violates NBR-as-source-of-truth
5. **No parameter editing in wizard** — users must edit raw JSON after creation
6. **Capability metadata mixed with env vars** — `env_json` contains non-Docker metadata
7. **No per-parameter source tracking** — can't tell if a value came from template, NBR, or user override

---

## 6. Existing Design Documents (Confirmed Principles)

From `lightai-backend-runtime-runplan-docker-design.md`:
- 5-layer snapshot chain: BV → BR → NBR → Deployment → RunPlan
- Each layer copies ALL config from parent at creation time
- Parent edits do NOT affect children
- Only explicit manual sync can pull updates

From `08-engineering-contracts.md`:
- Args: Model default as baseline; Instance override replaces wholesale
- Env: Key-identity, Runtime > Model > Instance override order
- Output must be stable — same input = byte-equivalent canonical JSON

From `06-execution-scope-reduction-and-decisions.md`:
- NBR is source of truth for runtime parameters
- Agent executes what NBR specifies
- No vendor policy engine, no privileged approval

---

## 7. Conflicts with Previous Documents

1. **RunPlan reads BackendVersion at resolution time** — contradicts "NBR is source of truth"
2. **`env_json` contains capability metadata** — contradicts "metadata cannot enter Docker env"
3. **No parameter editing at NBR level** — contradicts "NBR is source of truth for runtime parameters"
4. **Parameter merge is whole-replacement** — contradicts the need for per-parameter enable/disable

These conflicts will be resolved in the new design document.
