# 00 — Current Code Inventory

Generated: 2026-06-26 | Git HEAD: `c13f91f` | Branch: `main`

## 1. Baseline Verification

```bash
git rev-parse --short HEAD  # c13f91f
git status --short          # ?? docs/reports/product-hardening-20260626/  (only untracked)

go test ./...               # ALL PASS (14 packages, 0 failures)
go build ./cmd/server/...   # PASS
go build ./cmd/agent/...    # PASS

cd web && npm test          # ALL PASS (37 tests, 0 failures)
cd web && npm run build     # PASS (3.29s)
git diff --check            # PASS (no whitespace errors)
```

## 2. Directory Structure (source only)

```
cmd/
  agent/main.go                                    (~900 lines, agent entrypoint)
  server/main.go                                   (server entrypoint)
internal/
  agent/
    collector/  (model_scanner, nvidia, probe, protocol, gguf_reader, metax)
    metrics/    (prometheus metrics exporter)
    register/   (agent registration + heartbeat)
    runtime/    (docker client, health check, runplan adapter)
    state/      (agent state machine)
  common/
    config/     (env/config loading)
    errors/     (typed error constructors)
    log/        (structured logging)
    token/      (bootstrap token generation)
    types/      (shared type defs)
    version/    (version info)
  runtimecontract/
    constants.go          (ServingProtocol*, Capability*, BackendTask*, etc.)
  server/
    agentclient/          (HTTP client for agent tasks)
    api/
      agent_handlers.go              (AgentHandler struct, agent registration/task handlers)
      agent_identity_handlers.go     (node identity CRUD)
      agent_proxy_handlers.go        (proxy file browse + model scan to agent)
      agent_task_result_handlers.go  (task result polling)
      audit_handlers.go              (GET /api/v1/audit-logs)
      audit_writer.go                (WriteAudit function)
      backend_handlers.go            (backend version CRUD + help)
      configset_helpers.go           (parseConfigSet, applyConfigOverrides, rejectLegacyDeploymentPayload)
      deployment_lifecycle_handlers.go (2704 lines: deployment/instance/runplan CRUD + lifecycle)
      gpu_lease_helpers.go
      gpu_lease_handlers.go
      metax_device_binding_handlers.go
      node_handlers.go
      node_model_root_handlers.go
      node_gpu_handlers.go
      observability_handlers.go
      preflight_handlers.go          (HandlePreflightDeployments, 150 lines)
      resource_handlers.go
      router.go                      (ALL route registrations)
      runtime_handlers.go            (BackendRuntime + NodeBackendRuntime CRUD)
      ...
    auth/
      bootstrap.go        (permission catalog seed)
      middleware.go        (SessionMiddleware, AgentAuthMiddleware, CSRFMiddleware, RequirePermission)
      session.go
      password.go
    authz/
      checks.go
      helpers.go
    catalog/
      types.go            (ConfigItem, ConfigSet, Registry, BackendCatalog, VersionDoc, RuntimeDoc)
      loader.go           (MaterializeBackend, MaterializeBackendVersion, MaterializeBackendRuntime, SeedCatalog)
    db/
      db.go               (ALL table DDL: tenants, users, nodes, gpu_devices, backend_*, model_*, audit_logs, etc.)
    metrics/              (server-side Prometheus metrics)
    models/               (Go structs: deployment, instance, runplan, node, etc.)
    rbac/                 (RBAC helpers)
    runplan/
      types.go            (ResolvedRunPlan, ResolveInput, ParameterDef, ParameterValue, NBRSnapshotInfo, etc.)
      resolver.go         (Resolve: image → entrypoint → args → env → docker → mounts → health → ports → GPU, 1276 lines)
      dryrun.go           (ValidateDryRun: node, model path, GPU, host port checks)
      preview.go          (EquivalentCommandPreview: docker run command string)
      lint.go             (LintRunPlan: duplicate args, env/CLI conflict, insecure flags)
      compat.go           (CheckCompatibility)
      detection.go        (ClassifyEntrypointShape, DetectProcessStart)
      profiles.go         (DefaultProcessStartProfiles for vllm/sglang/llamacpp)
      resource_controls.go (ParseResourceControls, BuildResourceControlArgs, ValidateResourceControlValue)
      template.go         (substituteVars: {{VAR}} replacement)
      log_classifier.go   (NewRuntimeLogClassifier, classify runtime log lines)
configs/
  backend-catalog/
    versions/
      vllm/vllm-v0.23.0.yaml           (17 args_schema items + 9 resource_controls)
      sglang/sglang-v0.5.13.post1.yaml  (12 args_schema items + 4 resource_controls)
      sglang/sglang-v0.5.12.post1.yaml  (similar to .13, minus --tensor-parallel-size)
      sglang/sglang-0.4.6-compatible.yaml (legacy compat, has --trust-remote-code, --dist-init-addr)
      llamacpp/llamacpp-b9700.yaml      (14 args_schema items + 8 resource_controls, gpu_memory_fraction unsupported)
      ollama/ollama-latest.yaml
    runtimes/                           (~18 runtime YAML files)
    help/
      vllm/vllm-v0.23.0.zh-CN.yaml     (7 entries)
      sglang/sglang-v0.5.13.post1.zh-CN.yaml (6 entries)
      llamacpp/llamacpp-b9700.zh-CN.yaml     (6 entries)
web/
  src/
    api/
      auditLogs.ts        (fetch audit logs)
      backends.ts         (BackendVersion, BackendRuntimeTemplate types)
      client.ts           (apiClient singleton)
      deployments.ts      (preflightDeployment, dryRunDeployment, startDeployment, stopDeployment, createDeployment)
      runtimes.ts         (BackendRuntime type)
    components/
      CopyButton.vue
      DockerImagePicker.vue
      LanguageSwitcher.vue
      MetricCard.vue
      RemoteFileBrowser.vue
      StatusTag.vue
      common/
        HealthCheckEditor.vue
        JsonViewer.vue              (189 lines: read-only JSON display + search + download)
        RuntimeParameterEditor.vue  (EXISTS but NEVER IMPORTED by any page — dead code)
    composables/
      __tests__/useAutoRefresh.test.ts
    layouts/
      ConsoleLayout.vue    (sidebar menu: Dashboard, Runtimes, Runner Configs, Models, Deployments, Instances, ...)
    locales/
      zh-CN.ts             (~420 lines, 10+ concept-specific i18n groups)
      en-US.ts             (~420 lines, matching keys)
    pages/
      ApiKeysPage.vue         — DOES NOT EXIST
      GatewayUsagePage.vue    — DOES NOT EXIST
      AuditLogsPage.vue       (77 lines: filterable audit log table)
      BackendRuntimesPage.vue (read-only list + JsonViewer drawer, NO parameter editing)
      BackendsPage.vue        (backend version list)
      DashboardPage.vue
      GpusPage.vue
      ModelArtifactsPage.vue  (model CRUD + parameter_defaults textarea, CORRECTLY placed as model hints)
      ModelDeploymentsPage.vue (148 lines: list + thin create dialog + JsonViewer drawer, NO wizard)
      ModelInstancesPage.vue  (instance list with auto-refresh, log viewer)
      NodesPage.vue
      RolesPage.vue
      RunnerConfigsPage.vue   (list + create dialog + JsonViewer drawer, NO parameter editing)
      TenantsPage.vue
      UsersPage.vue
    router/
      index.ts             (all route defs: /runtimes, /runner-configs, /models/deployments, /models/instances, etc.)
    stores/
      __tests__/auth.test.ts
  tests/
    apiClientPaths.test.mjs
    formatters.test.mjs
    i18nKeys.test.mjs
    i18nMissingKeys.test.mjs
    modelCapabilities.test.mjs
    noHardcodedCredentials.test.mjs
    runtimeBoundaryUi.test.mjs
```

## 3. Current State by Workstream

### Workstream A — Naming Debt: Current Naming Map

| Internal entity | Route path | Vue component | zh-CN label | en-US label | i18n key | Problem |
|---|---|---|---|---|---|---|
| BackendRuntime | `/runtimes` | `BackendRuntimesPage.vue` | 运行模板 | Runtime Templates | `runtimes.title` | Component name ≠ label (page says "Templates" not "BackendRuntimes") |
| NodeBackendRuntime | `/runner-configs` | `RunnerConfigsPage.vue` | 运行配置 | Runtime Configs | `runnerConfigs.title` | Route name "runner-configs" is vestigial; component not named for entity |
| ModelDeployment | `/models/deployments` | `ModelDeploymentsPage.vue` | 模型部署 | Model Deployments | `deployments.title` | OK |
| ModelInstance | `/models/instances` | `ModelInstancesPage.vue` | 模型实例 | Model Instances | `instances.title` | OK |
| ResolvedRunPlan | (nested in detail) | (inline) | (raw "RunPlan") | (raw "RunPlan") | `help.runPlanTitle` | Internal term exposed to users |
| ConfigSet | (nested in drawer) | (JsonViewer title) | (raw "ConfigSet") | (raw "ConfigSet") | N/A (hardcoded) | Internal term shown as drawer title on 4 pages + 1 component |

**User-facing naming debt found:**

1. **"ConfigSet"** — hardcoded string `title="ConfigSet"` in `BackendsPage.vue:16`, `BackendRuntimesPage.vue:34`, `RunnerConfigsPage.vue:53`, `ModelDeploymentsPage.vue:55`, and `RuntimeParameterEditor.vue:46`. Also in i18n: `deployments.existingOverrides` = `"部署级 ConfigSet 覆盖"`, `deployments.overrideHint` = `"...materialized into the deployment ConfigSet."`

2. **"RunPlan"** — untranslated in i18n: `help.runPlanTitle`, `deployments.viewRunPlan`, `deployments.previewRunPlan`. Better: `deployments.finalRunPlan` = `"最终运行计划"`.

3. **"NBR" acronym** — exposed in i18n: `deployments.nbrTemplateGroup` = `"NBR 静态模板预览"`, `deployments.runPlanSourceNote` = `"参数按来源分组：NBR 模板 → ..."`.

4. **Raw UUIDs in table columns** — `RunnerConfigsPage.vue:14` shows `prop="backend_runtime_id"` as display value. `ModelDeploymentsPage.vue:14` shows `prop="source_node_backend_runtime_id"` as display value.

5. **`/runner-configs` route** — URL is vestigial "runner config" from old design. Entity is NodeBackendRuntime.

### Workstream B — Deployment UI: Current State

**Existing endpoints:**
| Method | Path | Handler | File:Line |
|---|---|---|---|
| GET | `/api/v1/deployments` | HandleListDeployments | deployment_lifecycle_handlers.go:28 |
| POST | `/api/v1/deployments` | HandleCreateDeployment | deployment_lifecycle_handlers.go:79 |
| GET | `/api/v1/deployments/{id}` | HandleGetDeployment | deployment_lifecycle_handlers.go:213 |
| PATCH | `/api/v1/deployments/{id}` | HandlePatchDeployment | deployment_lifecycle_handlers.go:227 |
| DELETE | `/api/v1/deployments/{id}` | HandleDeleteDeployment | deployment_lifecycle_handlers.go:365 |
| POST | `/api/v1/deployments/{id}/dry-run` | HandleDeploymentDryRun | deployment_lifecycle_handlers.go:1766 |
| POST | `/api/v1/deployments/preflight` | HandlePreflightDeployments | preflight_handlers.go:14 |
| POST | `/api/v1/deployments/{id}/start` | HandleStartDeployment | deployment_lifecycle_handlers.go:1061 |
| POST | `/api/v1/deployments/{id}/stop` | HandleStopDeployment | deployment_lifecycle_handlers.go:1333 |
| GET | `/api/v1/deployments/{id}/run-plan-groups` | HandleListRunPlanGroups | deployment_lifecycle_handlers.go:1520 |
| POST | `/api/v1/deployments/{id}/template-sync/preview` | HandleDeploymentTemplateSyncPreview | deployment_lifecycle_handlers.go:1857 |
| POST | `/api/v1/deployments/{id}/template-sync/apply` | HandleDeploymentTemplateSyncApply | deployment_lifecycle_handlers.go:1909 |

**What is MISSING:**
- `POST /api/v1/deployments/preview` — standalone unsaved preview endpoint DOES NOT EXIST
- No deployment wizard component exists (`DeploymentWizard.vue`, `ModelSelector.vue`, etc.)
- Current `ModelDeploymentsPage.vue` has only a thin create dialog (5 fields: name, model_artifact_id, node_backend_runtime_id, host_port, served_model_name)
- No RunPlan preview before save, no Docker command preview, no lint/preflight display in UI
- `RuntimeParameterEditor.vue` exists but is NEVER imported — dead code

**Current create payload (ModelDeploymentsPage.vue:109-122):**
```typescript
createDeployment({
  name: form.name,
  model_artifact_id: form.model_artifact_id,
  node_backend_runtime_id: form.node_backend_runtime_id,
  service_json: { host_port: form.host_port },
  config_overrides: { parameter_values: [{ key: 'backend.common.served_model_name', value: form.served_model_name, enabled: true }] }
})
```

**What the backend already supports but frontend doesn't use:**
- `placement_json` (accelerator_ids, gpu_policy)
- Full `config_overrides.parameter_values` array for all backend parameters
- `config_overrides.disabled_parameters`
- `config_overrides.env`
- `display_name`
- Preflight validation result display
- Dry-run result display (backend endpoint exists, frontend calls it but dumps raw JSON in drawer)

### Workstream C — Runtime Parameters: Current State

**Backend catalog completeness (vLLM v0.23.0 as reference):**

| Parameter | In YAML schema | In resource_controls | Surfaced in UI | Status |
|---|---|---|---|---|
| `--model` | YES | — | NO (auto) | LOCKED |
| `--host` | YES | — | NO (auto) | LOCKED |
| `--port` | YES | — | NO (via service) | LOCKED |
| `--served-model-name` | YES | — | YES (deployment create form) | EDITABLE |
| `--tensor-parallel-size` | YES | — | NO | MISSING FROM UI |
| `--pipeline-parallel-size` | YES | YES | NO | MISSING FROM UI |
| `--max-model-len` | YES | YES | NO | MISSING FROM UI |
| `--gpu-memory-utilization` | YES | YES | NO | MISSING FROM UI |
| `--max-num-seqs` | YES | YES | NO | MISSING FROM UI |
| `--max-num-batched-tokens` | YES | YES | NO | MISSING FROM UI |
| `--kv-cache-dtype` | YES | YES | NO | MISSING FROM UI |
| `--dtype` | YES | YES | NO | MISSING FROM UI |
| `--swap-space` | YES | YES | NO | MISSING FROM UI |
| `--cpu-offload-gb` | YES | YES | NO | MISSING FROM UI |
| `--enforce-eager` | YES | — | NO | MISSING FROM UI |
| `--safetensors-load-strategy` | YES | YES | NO | MISSING FROM UI |
| `--trust-remote-code` | YES | — | NO | MISSING FROM UI |
| `--download-dir` | YES | — | NO | MISSING FROM UI |

**Same applies to SGLang and llama.cpp — only `served_model_name` is UI-editable.**

**RuntimeParameterEditor.vue** (exists at `web/src/components/common/RuntimeParameterEditor.vue`):
- Props: `modelValue`, `readonly`, `backendSchema`, `vendor`, `helpBackend`, `helpVersion`
- Emits: `update:modelValue`
- Features: grouped sections (launcher, runtime_env, model_runtime), enable/disable per param, text/textarea/switch inputs, ConfigSet preview
- Import count in codebase: **ZERO** — never imported by any page or component

**Backend resolution pipeline (fully implemented in Go):**
1. catalog YAML → `VersionDoc.DefaultArgsSchema` / `VersionDoc.VendorOptions.resource_controls`
2. `addArgConfigItems()` → `ConfigItem` objects in ConfigSet
3. `SeedCatalog()` → persisted to SQLite
4. `configSetParameterValues()` → extracts `ParameterValue[]` from ConfigSet
5. `preflightDeployment()` → `runplan.Resolve()` → `buildArgs()` layers:
   - Layer 1: NBR `args_override`
   - Layer 2: NBR parameter values
   - Layer 3: Deployment parameter overrides
   - Layer 4b: `resource_controls`
6. Dedup, disabled tombstones, service args override
7. `EquivalentCommandPreview()` → human-readable `docker run` string

**ModelArtifactsPage parameter_defaults: CORRECTLY PLACED** — labeled "Model Serving Parameter Defaults", stored as model hints (not runtime parameters), used to pre-fill deployment form values.

### Workstream D — Gateway/Audit/Metering: Current State

**What EXISTS:**

| Component | File | Status |
|---|---|---|
| audit_logs table | `db/db.go:530-547` | EXISTS |
| WriteAudit function | `audit_writer.go:1-105` | EXISTS |
| HandleListAuditLogs | `audit_handlers.go:1-114` | EXISTS at `GET /api/v1/audit-logs` |
| AuditLogsPage.vue | `web/src/pages/AuditLogsPage.vue` | EXISTS |
| Route `/system/audit-logs` | `router/index.ts:90` | EXISTS |
| HandleModelInstanceTest | `deployment_lifecycle_handlers.go:2095` | EXISTS (diagnostic only, not a proxy) |

**What DOES NOT EXIST (all of Workstream D):**

- `api_keys` table — NOT IN DB SCHEMA
- `gateway_usage_records` table — NOT IN DB SCHEMA
- API key CRUD handlers — NOT IMPLEMENTED
- API key UI pages — `ApiKeysPage.vue` DOES NOT EXIST, `GatewayUsagePage.vue` DOES NOT EXIST
- API key auth middleware (for external consumers) — NOT IMPLEMENTED (only AgentAuthMiddleware exists for agent nodes)
- `GET /v1/models` and `POST /v1/chat/completions` routes — NOT REGISTERED
- Model routing/resolution logic for proxying — NOT IMPLEMENTED
- Proxy handler to forward to backend instances — NOT IMPLEMENTED
- Usage recording (token counts, latency, success/failure) — NOT IMPLEMENTED
- Gateway usage query API — NOT IMPLEMENTED
- `api_key:*` permissions — NOT IN BOOTSTRAP CATALOG

**Existing audit infrastructure** (can be reused for gateway auditing):
- `audit_logs` table with columns: id, tenant_id, actor_id, action, resource_type, resource_id, result, detail, metadata_json, created_at
- `WriteAudit()` writes audit records — gateway handlers should call this
- `HandleListAuditLogs` with filtering — gateway audit records will appear here by default

### Workstream E — Regression: Current State

**Existing test architecture:**

| Layer | Count | Framework | All PASS? |
|---|---|---|---|
| Go unit tests | 63 files, 142+ cases | Go testing | YES |
| Frontend MJS tests | 7 files | Node.js assert | YES |
| Frontend Vitest tests | 3 files | Vitest | YES |
| Bash E2E (current contract) | 6 files | Bash + curl | YES (when hardware available) |
| Bash E2E (legacy contract) | 15 files in `scripts/archive/legacy-contract/` | Bash + curl | NEEDS MIGRATION |
| Playwright browser tests | 0 files (installed but unconfigured) | Playwright | NONE EXIST |

**Runtime smoke evidence (most recent: 2026-06-25):**
- vLLM: health PASS, /v1/models PASS, inference PASS, stop PASS
- SGLang: started + stopped, inference PASS
- llama.cpp: started + stopped, inference PASS
- All containers stopped, no residual instances

**Scripts needing attention (Workstream E Step E2):**
- `scripts/smoke-model-backends.sh` — bypasses product API (direct Docker), needs repair or archival
- 15 scripts in `scripts/archive/legacy-contract/` — use deprecated `backend_runtime_id` / `parameters_json` fields, need migration or explicit archival label

## 4. DB Schema — All Existing Tables

```
tenants, users, tenant_memberships, roles, permissions, role_permissions,
tenant_membership_roles, sessions,
nodes, gpu_devices, node_system_snapshots, node_filesystem_snapshots,
node_network_snapshots,
inference_backends, backend_versions, backend_runtimes, node_backend_runtimes,
model_artifacts, model_locations, node_model_roots,
model_deployments, model_instances, resolved_run_plans, run_plan_groups,
gpu_leases, agent_tasks,
audit_logs
```

**Tables to ADD (Workstream D):** `api_keys`, `gateway_usage_records`

**No other DB changes needed.** No legacy data migration required (clean DB policy).

## 5. API Route Map — All Registered Routes

Routes are registered in `internal/server/api/router.go`. Full listing (grouped by prefix):

```
/api/v1/auth/login, /logout, /change-password, /me, /csrf-token
/api/v1/session/switch-tenant
/api/v1/users, /users/{id}
/api/v1/tenants, /tenants/{id}
/api/v1/tenant-memberships, /tenant-memberships/{id}
/api/v1/roles, /roles/{id}
/api/v1/permissions, /permissions/{id}
/api/v1/nodes, /nodes/{id}, /nodes/{id}/files, /nodes/{id}/model-paths/scan,
  /nodes/{id}/backend-runtimes/enable, /nodes/{id}/backend-runtimes/{nbr_id},
  /nodes/backend-runtimes/all, /nodes/{id}/backend-runtimes/{nbr_id}/check-request
/api/v1/gpus
/api/v1/backends, /backends/{id}
/api/v1/backend-versions, /backend-versions/{id}, /backend-versions/{id}/clone
/api/v1/backend-runtimes, /backend-runtimes/{id}
/api/v1/backend-runtime-templates
/api/v1/model-artifacts, /model-artifacts/{id}
/api/v1/model-locations, /model-locations/{id}
/api/v1/node-model-roots
/api/v1/deployments, /deployments/{id}, /deployments/{id}/dry-run,
  /deployments/{id}/template-sync/preview, /deployments/{id}/template-sync/apply,
  /deployments/preflight, /deployments/{id}/start, /deployments/{id}/stop,
  /deployments/{id}/run-plan-groups
/api/v1/model-instances, /model-instances/{id}, /model-instances/{id}/test
/api/v1/node-run-plans/{id}, /node-run-plans/{id}/command-preview,
  /node-run-plans/{id}/logs
/api/v1/gpu-leases
/api/v1/agent/register, /agent/heartbeat, /agent/tasks/{id}/result,
  /agent/config
/api/v1/observability/status
/api/v1/audit-logs
```

**Routes to ADD (Workstream D):**
```
POST   /api/v1/api-keys
GET    /api/v1/api-keys
POST   /api/v1/api-keys/{id}/disable
DELETE /api/v1/api-keys/{id}
GET    /api/v1/gateway/usage
GET    /v1/models                    (external, outside /api/v1)
POST   /v1/chat/completions         (external, outside /api/v1)
```

**Routes to ADD (Workstream B):**
```
POST   /api/v1/deployments/preview
```

## 6. UI Page/Component Map

| Page | Lines | Has parameter editing? | Has preview/wizard? | Notes |
|---|---|---|---|---|
| BackendRuntimesPage | ~60 | NO (read-only JsonViewer) | NO | Shows runtime templates |
| RunnerConfigsPage | ~120 | NO (read-only JsonViewer) | NO | Shows node runtime configs |
| ModelDeploymentsPage | 148 | Only served_model_name | NO (thin dialog) | MAIN TARGET for wizard |
| ModelInstancesPage | ~340 | N/A | NO | Instance list + logs + auto-refresh |
| ModelArtifactsPage | ~560 | parameter_defaults textarea (correct) | N/A | Model hints, NOT runtime params |
| BackendsPage | ~60 | NO (read-only JsonViewer) | NO | Backend versions |
| AuditLogsPage | 77 | N/A | N/A | Audit log browsing |
| ApiKeysPage | — | — | — | DOES NOT EXIST |
| GatewayUsagePage | — | — | — | DOES NOT EXIST |

## 7. Component Inventory

| Component | Path | Purpose | Used by |
|---|---|---|---|
| RuntimeParameterEditor | `componts/common/RuntimeParameterEditor.vue` | Structured parameter editing with sections | **UNUSED — dead code** |
| JsonViewer | `components/common/JsonViewer.vue` | Read-only JSON display + search | BackendRuntimesPage, RunnerConfigsPage, ModelDeploymentsPage, BackendsPage |
| HealthCheckEditor | `components/common/HealthCheckEditor.vue` | Health check config editing | (check usage) |
| DockerImagePicker | `components/DockerImagePicker.vue` | Docker image selection | RunnerConfigsPage |
| StatusTag | `components/StatusTag.vue` | Status badge | Multiple pages |
| RemoteFileBrowser | `components/RemoteFileBrowser.vue` | Remote file system browse | Nodes |
| CopyButton | `components/CopyButton.vue` | Copy to clipboard | Multiple pages |
| MetricCard | `components/MetricCard.vue` | Dashboard metric cards | Dashboard |
| LanguageSwitcher | `components/LanguageSwitcher.vue` | zh-CN/en-US toggle | ConsoleLayout |
