# BackendRuntime vs NodeBackendRuntime: Template and Node-Runtime Snapshot Design

> Status: CURRENT
> Last reviewed: 2026-06-18
> Scope: Formal definition of BackendRuntime (运行模板) and NodeBackendRuntime (节点运行配置)
> Read order: After `docs/CURRENT.md` and `docs/design/backend-runtime-runplan-docker.md`

## 1. Core Definitions

### 1.1 BackendRuntime = 运行模板

A BackendRuntime is a **template-layer object** that describes how a class of backend should run. It is not bound to any specific node.

A BackendRuntime defines:

| Field | Purpose |
|-------|---------|
| `backend_id` | Which inference backend (vllm, llamacpp, ollama, sglang) |
| `backend_version_id` | Which backend version |
| `name` / `display_name` | Human-readable runtime template name |
| `vendor` | GPU/CPU vendor (nvidia, metax, cpu, huawei) |
| `image_name` | Default Docker image |
| `image_pull_policy` | Image pull policy |
| `entrypoint_override_json` | Container entrypoint override |
| `args_override_json` | Default args |
| `default_env_json` | Default environment variables |
| `docker_json` | Docker options (privileged, ipc, shm, devices, etc.) |
| `model_mount_json` | Model mount configuration |
| `health_check_override_json` | Health check endpoint configuration |
| `is_builtin` | System-managed (0/1) |
| `is_editable` | Whether user can modify |
| `tenant_id` | Tenant ownership |

**Rules:**
- BackendRuntime does not bind to any node.
- BackendRuntime is not a guarantee that any node is ready.
- The "运行模板" page must only show BackendRuntime records (template layer).
- System templates (`is_builtin=1, is_editable=0`) are read-only.
- User templates (`is_editable=1`) can be created by cloning a system template or from scratch.

### 1.2 NodeBackendRuntime = 节点运行配置

A NodeBackendRuntime is a **node-level configuration record** that represents a specific BackendRuntime enabled on a specific node.

Current schema (`node_backend_runtimes` table):

| Field | Purpose |
|-------|---------|
| `id` | Synthetic ID: `{node_id}:{backend_runtime_id}` |
| `backend_runtime_id` | Reference to source BackendRuntime |
| `node_id` | Target node |
| `runner_type` | Runner type (docker, command, etc.) |
| `image_ref` | Node-level image reference override |
| `image_id` | Resolved image ID |
| `image_digest` | Image digest |
| `image_present` | Whether image exists on node |
| `docker_available` | Whether Docker is available on node |
| `driver_version` | GPU driver version |
| `toolkit_version` | GPU toolkit version |
| `device_check_json` | Device check results |
| `status` | `ready`, `missing_image`, `unsupported_device`, `template_only`, `unknown` |
| `status_reason` | Human-readable reason for status |
| `last_checked_at` | Last check timestamp |
| `tenant_id` | Tenant ownership |

**Rules:**
- NodeBackendRuntime references a BackendRuntime via `backend_runtime_id` as the source template.
- At enable/check time, the system captures a **frozen config snapshot** (`config_snapshot_json`) of the BackendRuntime's args, env, docker, mounts, health_check, entrypoint, image, and vendor.
- RunPlan resolution reads the NBR snapshot as the primary execution config source.
- BackendRuntime template edits do NOT affect existing NodeBackendRuntime RunPlans.
- NodeBackendRuntime status is determined by `evaluateNodeBackendRuntime()` which checks GPU vendor match, Docker availability, and image presence.
- NodeBackendRuntime image fields can be overridden per node; a node-level `image_ref` is stored separately and takes precedence over the snapshot image.

## 2. Relationship Rules

### 2.1 Creation

```
User selects BackendRuntime → creates/enables NodeBackendRuntime on a specific node.
At enable/check time, the system captures a config_snapshot_json from the current BackendRuntime:
  - source_runtime_id, source_runtime_name, source_runtime_revision
  - backend_id, backend_version_id
  - vendor, runtime_type, image_name, image_pull_policy
  - args_override_json, entrypoint_override_json
  - default_env_json, docker_json, model_mount_json, health_check_override_json
Node-level overrides (image_ref, image_present) are also stored.
```

**Do NOT create a new BackendRuntime when enabling on a node.** The `HandleEnableNodeBackendRuntime` endpoint (POST `/nodes/{id}/backend-runtimes/enable`) upserts a NodeBackendRuntime record with the given `backend_runtime_id` AND captures the snapshot.

Creating a user-managed BackendRuntime (template clone) is a separate action:
- BackendRuntimesPage "Clone" button → `POST /backend-runtimes/{id}/clone`
- This creates a new BackendRuntime with `is_editable=1, is_builtin=0`
- This is the "另存为模板 / 保存为用户模板" action

### 2.2 Independence After Creation

```
NodeBackendRuntime persists a frozen config_snapshot_json at enable/check time.
After creation, template modifications do NOT affect the NBR's RunPlan output.
The RunPlan resolver reads the NBR snapshot as the primary config source.
BackendRuntime is still referenced for metadata (name, source tracking) but its
current config values do not override the snapshot.
```

**Current implementation (v16 migration):**
- `node_backend_runtimes.config_snapshot_json` stores the frozen config.
- `preflightDeployment` reads the snapshot and uses it to override the RuntimeInfo before calling `runplan.Resolve`.
- Image resolution: NBR `image_ref` (node-level override) > snapshot `image_name` > BackendVersion.defaultImages.
- If snapshot is empty (legacy data), the live BackendRuntime config is used as fallback.

### 2.3 Editing NodeBackendRuntime

```
Editing a NodeBackendRuntime's node-level fields (image_ref, image_present,
config_snapshot_json) invalidates the ready status → status becomes "needs_check".
Re-check is required before deployment.
```

**Implementation:** `HandlePatchNodeBackendRuntime` sets `status = 'needs_check'` when image-related fields or snapshot config are modified. The preflight resolver excludes NBRs with `status != 'ready'`.

Fields that trigger status invalidation:
- `image_ref`
- `image_id`
- `image_digest`
- `image_present`
- Any `config_snapshot_json` edit

Fields that do NOT trigger invalidation (informational only):
- `driver_version`
- `toolkit_version`
- `disabled` (sets to 'disabled' directly)

### 2.4 Template Change / Re-apply

Not implemented in current phase. Documented as P2:

```
1. "Re-apply template": re-read the current source BackendRuntime and update
   the NodeBackendRuntime's reference/overrides. Must be explicit user action.
2. "Change template": select a different BackendRuntime as source.
   Must show diff and require explicit confirmation.
   If backend_id/backend_version_id differs, default to disallow.
   Status must become "needs_check" after change.
3. If the NBR has running instances, must warn — changes only affect
   subsequent starts.
```

## 3. Model Mount Per-Node Resolution

### 3.1 ModelLocation → NodeRunPlan Mount

```
ModelArtifact (logical model)
    └── ModelLocation (per-node physical location)
            ├── node_id: which node
            ├── model_root: root directory on that node (e.g., /data/models)
            ├── relative_path: model subdirectory (e.g., qwen3.5-9b)
            └── absolute_path: full path on disk

Deployment with artifact + node → RunPlan resolver:
    1. Query ModelLocation WHERE model_artifact_id=? AND node_id=?
    2. host_path = model_root + "/" + relative_path
    3. container_path = /models/<relative_path>  (standardized)
    4. Mount: host_path:container_path:ro
    5. MODEL_CONTAINER_PATH var = container_path
```

### 3.2 Multi-Node Example

```
ModelArtifact: Qwen3.5-9B

Node A (model at /data/models/qwen3.5-9b):
    host mount = /data/models/qwen3.5-9b:/models/qwen3.5-9b:ro
    args use: --model /models/qwen3.5-9b

Node B (model at /mnt/nvme/model-store/qwen3.5-9b):
    host mount = /mnt/nvme/model-store/qwen3.5-9b:/models/qwen3.5-9b:ro
    args use: --model /models/qwen3.5-9b

→ Different host paths, same container path. Args remain identical.
```

### 3.3 NodeBackendRuntime Does Not Bind ModelLocation

```
NodeBackendRuntime = runtime config snapshot (how to run)
ModelLocation = model file location (where the model is)

These are independent:
- Same NodeBackendRuntime can be used for different models.
- Each Deployment selects artifact + runtime independently.
- Model mount is resolved at RunPlan time based on (deployment.artifact, node).
- NBR snapshot does not store model host paths.
```

## 4. UI Display Rules

### 3.1 "运行模板" Page (BackendRuntimesPage.vue)

Shows BackendRuntime records only:

| Column | Source |
|--------|--------|
| Name | `backend_runtimes.name` |
| Backend | `backend_runtimes.backend_id` |
| Version | `backend_runtimes.backend_version_id` |
| Managed By | System / User (`is_editable`) |
| Node Count | Aggregated from `node_backend_runtimes` COUNT |
| Ready Count | Aggregated from `node_backend_runtimes WHERE status='ready'` |
| Actions | View, Edit, Clone, Delete (user only) |

### 3.2 "运行配置" Page (RunnerConfigsPage.vue)

Shows NodeBackendRuntime records (node-level configs):

| Column | Source |
|--------|--------|
| Template Name | JOIN `backend_runtimes.name` |
| Node | `node_id` (with label) |
| Runner Type | `runner_type` |
| Status | `status` (translated via i18n) |
| Image Ref | `image_ref` |

### 3.3 Node Detail

Shows NodeBackendRuntime records for that node (via `GET /nodes/{id}/backend-runtimes`).

### 3.4 Template Detail Drawer

Shows BackendRuntime detail + NodeBackendRuntime records that reference this template (via `GET /nodes/{id}/backend-runtimes` for each node, filtered).

### 3.5 Deployment Wizard

```
Step 1: Select model
Step 2: Select backend
Step 3: Select backend version
Step 4: Select BackendRuntime (template) — filtered by backend_version_id
Step 5: Preflight → Server resolver checks:
  - ModelLocation exists on candidate nodes
  - NodeBackendRuntime with status='ready' exists on candidate nodes
  - backend_version_id matches
  - Node is online
Step 6: Start
```

## 4. Status Lifecycle

```
              enable/check
  unknown ─────────────────→ ready / missing_image / unsupported_device / template_only
     ↑                            │
     │                            │ edit image_ref / image_present
     │                            ↓
     └──────── re-check ──── needs_check
```

### 5.1 Status Values

| Status | Meaning | Display (zh-CN) |
|--------|---------|-----------------|
| `unknown` | Not yet checked | 未知 |
| `ready` | All checks passed | 就绪 |
| `needs_check` | Config edited, re-check needed | 需重新检测 |
| `missing_image` | Docker image not present | 镜像缺失 |
| `unsupported_device` | No matching GPU vendor | 设备不支持 |
| `template_only` | Template-only (Huawei) | 仅模板 |
| `disabled` | Explicitly disabled | 已禁用 |

## 5. Backend API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/backend-runtimes` | List BackendRuntime (templates) |
| POST | `/api/v1/backend-runtimes/from-template` | Create BackendRuntime from template |
| POST | `/api/v1/backend-runtimes/{id}/clone` | Clone as user template |
| PATCH | `/api/v1/backend-runtimes/{id}` | Edit BackendRuntime |
| DELETE | `/api/v1/backend-runtimes/{id}` | Delete BackendRuntime |
| GET | `/api/v1/nodes/{id}/backend-runtimes` | List NodeBackendRuntime for node |
| POST | `/api/v1/nodes/{id}/backend-runtimes/enable` | Create/update NodeBackendRuntime |
| POST | `/api/v1/nodes/{id}/backend-runtimes/check` | Check NodeBackendRuntime readiness |
| PATCH | `/api/v1/nodes/{nid}/backend-runtimes/{nbr_id}` | Edit NodeBackendRuntime |
| DELETE | `/api/v1/nodes/{nid}/backend-runtimes/{nbr_id}` | Delete NodeBackendRuntime |

## 6. Current Implementation Status

### Implemented (Phase 4, v16):
- BackendRuntime CRUD + template-based creation
- NodeBackendRuntime enable/check with **config snapshot capture** (`config_snapshot_json`)
- RunPlan resolver reads NBR snapshot as the primary execution config
- BackendRuntime template edits do NOT affect existing NBR RunPlans
- `preflightDeployment` overrides RuntimeInfo from NBR snapshot before calling `runplan.Resolve`
- Per-node model mount using `model_root + "/" + relative_path` → container path `/models/<slug>`
- Node count / ready count aggregation on BackendRuntime list
- RunnerConfigsPage shows NodeBackendRuntime records
- Deployment wizard filters by backend_version_id + preflight
- Status invalidation on NodeBackendRuntime edit (`needs_check`)
- Expanded `needs_check` triggers: image_ref, image_id, image_digest, image_present, config_snapshot_json edits

### P2 / future enhancements:
- Template re-apply / template change with diff UI
- Template revision visualization
- Non-Docker runner types
- GPU lease picker in deployment wizard
- Port auto-suggestion

## 7. Migration Notes

Historical data: Previous code (before commit 271e8b3) cloned BackendRuntime records when enabling on a node. These clone records remain in `backend_runtimes` as user-managed templates. They are distinguishable:
- `is_editable = 1`
- `source_template_name` points to the original system template
- `managed_by = 'user'`
- Name may be identical to the system template name

No automatic cleanup is performed. Users should manually review and delete unwanted user templates. Future UI may add a "deduplicate" or "cleanup" action.

## 8. Related Documents

- `docs/design/backend-runtime-runplan-docker.md` — Backend / BackendVersion / RunPlan design
- `docs/design/model-runtime-node-wizard.md` — Model wizard and deployment wizard design
- `docs/reports/model-runtime-node-wizard/open-issues-closeout.md` — Formal closeout
- `docs/reports/backend-runtime-runplan/open-issues-closeout.md` — BackendRuntime blockers
