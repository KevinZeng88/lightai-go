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
- NodeBackendRuntime references a BackendRuntime via `backend_runtime_id`.
- It stores **node-level overrides** (image_ref, image_present, docker_available).
- It does NOT duplicate the full template; it references + overrides.
- NodeBackendRuntime status is determined by `evaluateNodeBackendRuntime()` which checks GPU vendor match, Docker availability, and image presence.

## 2. Relationship Rules

### 2.1 Creation

```
User selects BackendRuntime → creates NodeBackendRuntime on a specific node.
System copies the template reference (backend_runtime_id) to the node record.
Node-level fields (image_ref, image_present) are set during creation.
```

**Do NOT create a new BackendRuntime when enabling on a node.** The `HandleEnableNodeBackendRuntime` endpoint (POST `/nodes/{id}/backend-runtimes/enable`) already implements this correctly: it upserts a NodeBackendRuntime record with the given `backend_runtime_id`.

Creating a user-managed BackendRuntime (template clone) is a separate action:
- BackendRuntimesPage "Clone" button → `POST /backend-runtimes/{id}/clone`
- This creates a new BackendRuntime with `is_editable=1, is_builtin=0`
- This is the "另存为模板 / 保存为用户模板" action

### 2.2 Independence After Creation

```
NodeBackendRuntime records the source template reference.
After creation, template modifications do NOT automatically affect node configs.
Runtime behavior is determined by the preflight/RunPlan resolver, which reads
the BackendRuntime (template) + NodeBackendRuntime (node overrides) and merges them.
```

**Current implementation:** The `backend_runtime_id` reference means that template-level changes (args, env, docker options) WILL affect the next deployment start because the RunPlan resolver reads the template at resolution time. This is the "reference" pattern — intentional for the current phase.

**Future full-snapshot upgrade:** A future migration may add `config_snapshot_json` and/or `override_json` fields to NodeBackendRuntime to fully decouple the node config from template changes. This is documented as a P2 enhancement and NOT required for current phase.

### 2.3 Editing NodeBackendRuntime

```
Editing a NodeBackendRuntime's node-level fields (image_ref, image_present)
invalidates the ready status → status becomes "needs_check".
Re-check is required before deployment.
```

**Implementation:** `HandlePatchNodeBackendRuntime` sets `status = 'needs_check'` when image-related fields are modified. The preflight resolver excludes NBRs with `status != 'ready'`.

Fields that trigger status invalidation:
- `image_ref`
- `image_id`
- `image_digest`
- `image_present`

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

## 3. UI Display Rules

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

### 4.1 Status Values

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

### Implemented (Phase 4, verified by NVIDIA E2E):
- BackendRuntime CRUD + template-based creation
- NodeBackendRuntime enable/check/status evaluation
- Node count / ready count aggregation on BackendRuntime list
- RunnerConfigsPage shows NodeBackendRuntime records
- Deployment wizard filters by backend_version_id + preflight
- Status invalidation on NodeBackendRuntime edit (`needs_check`)

### Not yet implemented (P2 / future):
- Full config snapshot in NodeBackendRuntime
- Template re-apply / template change with diff UI
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
