> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# Model Runtime Node Wizard Implementation Plan

**Status: COMPLETED** (Phase 4 closed at `50a25a5`)

## Phase 1: Agent Capabilities ✅

| Item | Endpoint | File |
|------|----------|------|
| File browser | `GET /files` | `cmd/agent/main.go` |
| Model scanner | `POST /model-paths/scan` | `internal/agent/collector/model_scanner.go` |
| Enhanced Docker images | `GET /docker-images` | `cmd/agent/main.go` |
| Dynamic extra_roots | query param | `cmd/agent/main.go` |

## Phase 2: Server API ✅

| Item | Endpoint | File |
|------|----------|------|
| ModelLocation PATCH/DELETE | `/api/v1/model-artifacts/{id}/locations/{lid}` | `model_location_handlers.go` |
| NodeBackendRuntime PATCH/DELETE | `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | `node_runtime_handlers.go` |
| BackendRuntime clone | `/api/v1/backend-runtimes/{id}/clone` | `node_runtime_handlers.go` |
| File proxy | `/api/v1/nodes/{id}/files` | `agent_proxy_handlers.go` |
| Model scan proxy | `/api/v1/nodes/{id}/model-paths/scan` | `agent_proxy_handlers.go` |
| Dynamic roots CRUD | `/api/v1/nodes/{id}/model-browser/roots` | `model_browser_handlers.go` |
| Standalone preflight | `/api/v1/deployments/preflight` | `preflight_handlers.go` |
| DB migration V14 | `model_browser_extra_roots` | `db.go` |

## Phase 3: Web Wizards ✅

| Page/Component | Route | File |
|---------------|-------|------|
| RemoteFileBrowser | — | `components/RemoteFileBrowser.vue` |
| DockerImagePicker | — | `components/DockerImagePicker.vue` |
| BackendRuntimesPage (运行模板) | `/runtimes` | `pages/BackendRuntimesPage.vue` |
| RunnerConfigsPage (运行配置) | `/runner-configs` | `pages/RunnerConfigsPage.vue` |
| ModelArtifactsPage (模型) | `/models/artifacts` | `pages/ModelArtifactsPage.vue` |
| ModelDeploymentsPage (实例) | `/models/deployments` | `pages/ModelDeploymentsPage.vue` |

## Concept Mapping

```
Backend (推理后端)     →  BackendRuntime (运行模板)  →  NodeBackendRuntime (节点配置)
/backends                  /runtimes                    (内嵌在详情抽屉)
                           /runner-configs (组合模板+节点+镜像=运行配置)
```
