# Model Runtime Node Wizard Current State Audit

**Date:** 2026-06-18
**Branch:** `main` → `phase-4-model-runtime-wizards`
**Design Doc:** `docs/model-runtime-node-wizard-design.md`

## 1. Current Model Management — API & Pages

### 1.1 ModelArtifact CRUD

| Capability | Status | Handler | File |
|-----------|--------|---------|------|
| List artifacts | ✅ | `HandleListArtifacts` | `artifact_handlers.go:18` |
| Create artifact | ✅ | `HandleCreateArtifact` | `artifact_handlers.go:35` |
| Get artifact | ✅ | `HandleGetArtifact` | `artifact_handlers.go:70` |
| Patch artifact | ✅ | `HandlePatchArtifact` | `artifact_handlers.go:84` |
| Delete artifact | ✅ | `HandleDeleteArtifact` | `artifact_handlers.go:133` |
| Discover artifact | ✅ | `HandleDiscoverArtifact` | `artifact_handlers.go:216` |

### 1.2 ModelLocation CRUD

| Capability | Status | Handler | File |
|-----------|--------|---------|------|
| List locations | ✅ | `listModelLocations` (embedded in getArtifactJSON) | `artifact_handlers.go:340` |
| Create location | ✅ | `HandleCreateModelLocation` | `artifact_handlers.go:279` |
| Rescan location | ✅ | `HandleRescanModelLocation` | `artifact_handlers.go:328` |
| Attest location | ✅ | `HandleAttestModelLocation` | `artifact_handlers.go:338` |
| **Patch location** | ❌ | **MISSING** |
| **Delete location** | ❌ | **MISSING** (only via cascade in `HandleDeleteArtifact`) |

### 1.3 ModelArtifact Web Page

- `web/src/pages/ModelArtifactsPage.vue` — Simple CRUD table with create/edit dialog
- Uses Element Plus (`el-table`, `el-dialog`, `el-form`)
- **No wizard flow** (no directory browser, no model scan, no node selection)
- **No ModelLocation management tab**
- Route: `/models/artifacts`

## 2. Current Runtime Configuration — API & Pages

### 2.1 BackendRuntime CRUD

| Capability | Status | Handler | File |
|-----------|--------|---------|------|
| List runtimes | ✅ | `HandleListBackendRuntimes` | `runtime_handlers.go:19` |
| Create from template | ✅ | `HandleCreateBackendRuntimeFromTemplate` | `runtime_handlers.go:37` |
| Get runtime | ✅ | `HandleGetBackendRuntime` | `runtime_handlers.go:93` |
| Patch runtime | ✅ | `HandlePatchBackendRuntime` | `runtime_handlers.go:107` |
| Delete runtime | ✅ (blocks built-in) | `HandleDeleteBackendRuntime` | `runtime_handlers.go:153` |
| **Clone runtime** | ❌ | **MISSING** |
| **Update status/disable** | ❌ | **MISSING** |

### 2.2 NodeBackendRuntime CRUD

| Capability | Status | Handler | File |
|-----------|--------|---------|------|
| List node runtimes | ✅ | `HandleListNodeBackendRuntimes` | `runtime_handlers.go:180` |
| Enable (upsert) | ✅ | `HandleEnableNodeBackendRuntime` | `runtime_handlers.go:215` |
| Check (upsert) | ✅ | `HandleCheckNodeBackendRuntime` | `runtime_handlers.go:219` |
| **Patch node runtime** | ❌ | **MISSING** |
| **Delete node runtime** | ❌ | **MISSING** |

### 2.3 Runtime Web Page

- `web/src/pages/BackendRuntimesPage.vue` — Runtime table with create-from-template and edit dialogs
- Edit dialog is rich: scalar options, list options, Custom Args/Env/Docker Options, command preview
- **No wizard flow** (no Docker image browser, no node selection in creation)
- **No NodeBackendRuntime management tab**
- Route: `/runtimes`

## 3. Current Agent Capabilities

### 3.1 File Browsing

| Capability | Status | Evidence |
|-----------|--------|----------|
| Agent directory listing | ❌ | No endpoint in agent — searched `cmd/agent/`, `internal/agent/` |
| Agent file metadata scan | ❌ | No config.json/safetensors parsing |
| Agent model discovery | ❌ | `HandleDiscoverArtifact` is server-side only, no agent proxy |

### 3.2 Docker Image Listing

| Capability | Status | Evidence |
|-----------|--------|----------|
| Agent docker image list | ✅ | `cmd/agent/main.go:295-326` — `GET /docker-images` on agent metrics port |
| Server proxy for images | ✅ | `HandleGetNodeDockerImages` in `agent_handlers.go:593` |
| Route | ✅ | `GET /api/v1/nodes/{id}/docker-images` in `router.go:90` |
| **Returned fields** | ⚠️ MINIMAL | Only `{image, size}` — no image_id, digest, created_at, labels |
| **Search/pagination** | ❌ | No query/filter/pagination support |

### 3.3 Model Scanning

| Capability | Status |
|-----------|--------|
| HuggingFace config.json reading | ❌ |
| Safetensors metadata | ❌ |
| GGUF metadata | ❌ |
| File size estimation | ❌ |
| Model format detection | ❌ |

## 4. Current Deployment / Preflight

### 4.1 Deployment Lifecycle

| Capability | Status | Handler/Route |
|-----------|--------|---------------|
| Create deployment | ✅ | `POST /api/v1/deployments` |
| Start deployment | ✅ | `POST /api/v1/deployments/{id}/start` |
| Stop deployment | ✅ | `POST /api/v1/deployments/{id}/stop` |
| Delete deployment | ✅ | `DELETE /api/v1/deployments/{id}` |
| Dry-run | ✅ | `POST /api/v1/deployments/{id}/dry-run` — calls `preflightDeployment()` |
| **Standalone preflight** | ❌ | No `/api/v1/deployments/preflight` endpoint — only deployment-specific dry-run |

### 4.2 Instance Pages

- `web/src/pages/ModelDeploymentsPage.vue` — List of deployments with start/stop/delete, create dialog
- `web/src/pages/ModelInstancesPage.vue` — List of instances with Docker logs drawer
- **No instance start wizard** (no step-by-step flow, no preflight check)

## 5. Web UI Component Inventory

| Component | Library | Available? |
|-----------|---------|------------|
| Table | Element Plus `el-table` | ✅ |
| Tree | Element Plus `el-tree` | ✅ (importable) |
| Pagination | Element Plus `el-pagination` | ✅ |
| Dialog | Element Plus `el-dialog` | ✅ |
| Drawer | Element Plus `el-drawer` | ✅ (used in ModelInstancesPage) |
| Breadcrumb | Element Plus `el-breadcrumb` | ✅ (importable) |
| Form | Element Plus `el-form` | ✅ |
| Select | Element Plus `el-select` | ✅ |
| Input | Element Plus `el-input` | ✅ |
| Tag | Element Plus `el-tag` | ✅ |
| Steps | Element Plus `el-steps` | ✅ (importable — good for wizards) |
| Descriptions | Element Plus `el-descriptions` | ✅ (used in BackendRuntimesPage) |
| Autocomplete | Element Plus `el-autocomplete` | ✅ (importable) |

## 6. i18n Current State

- 407 leaf keys in both zh-CN and en-US
- Test: `web/tests/i18nMissingKeys.test.mjs` checks all `$t()`/`t()` references resolve to strings
- New namespaces needed: `modelWizard.*`, `runtimeWizard.*`, `startWizard.*`, `fileBrowser.*`, `dockerImages.*`, `preflight.*`, `modelLocations.*`, `nodeRuntime.*`

## 7. DB Schema

| Table | Created In | Key Fields |
|-------|-----------|------------|
| `model_artifacts` | migrateV3 | id, name, path, format, task_type, status, tenant_id |
| `model_locations` | migrateV13 | id, model_artifact_id, node_id, path_type, model_root, relative_path, absolute_path, match_status, verification_status, status, tenant_id |
| `backend_runtimes` | migrateV10 | id, backend_id, backend_version_id, vendor, image_name, docker_json, is_editable, status, tenant_id |
| `node_backend_runtimes` | migrateV13 | id, backend_runtime_id, node_id, image_ref, image_present, docker_available, status, status_reason |

**All foreign keys: NO CASCADE. Application must handle deletion ordering.**

## 8. Consistent with Design Doc

- ModelArtifact / ModelLocation separation exists
- BackendRuntime / NodeBackendRuntime separation exists
- preflightDeployment already validates all required checks
- Element Plus is the UI library — all needed components available
- Docker image API exists on agent (needs enhancement)

## 9. Missing Items (Gap Summary)

| ID | Gap | Priority |
|----|-----|----------|
| GAP-001 | Agent file browsing endpoint | P0 |
| GAP-002 | Agent model scanning capability | P0 |
| GAP-003 | Enhanced Docker image listing (fields, search, pagination) | P0 |
| GAP-004 | ModelLocation PATCH / DELETE handlers | P0 |
| GAP-005 | NodeBackendRuntime PATCH / DELETE handlers | P0 |
| GAP-006 | BackendRuntime clone handler | P1 |
| GAP-007 | Standalone preflight endpoint | P1 |
| GAP-008 | Model creation wizard (Web) | P0 |
| GAP-009 | Runtime creation wizard (Web) | P0 |
| GAP-010 | Instance start wizard (Web) | P1 |
| GAP-011 | File browser Web component | P0 |
| GAP-012 | Docker image browser Web component | P0 |
| GAP-013 | ModelLocation management Web UI | P0 |
| GAP-014 | NodeBackendRuntime management Web UI | P0 |
| GAP-015 | Model consistency comparison | P1 |
| GAP-016 | i18n for all new pages/components | P0 |
| GAP-017 | Delete protection (running instances) | P1 |
| GAP-018 | Audit logging for new operations | P1 |

## 10. Risk Items

| ID | Risk |
|----|------|
| RISK-001 | Agent file browsing needs path traversal protection — security-critical |
| RISK-002 | Agent model scan is CPU/IO intensive — needs timeout |
| RISK-003 | Docker image list via `docker images` CLI is slow — should parse Docker API or cache |
| RISK-004 | BackendRuntime clone must properly clear node-specific fields |
| RISK-005 | NodeBackendRuntime delete must check no running instances reference it |
