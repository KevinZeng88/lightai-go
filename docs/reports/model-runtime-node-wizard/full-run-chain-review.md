# Model Runtime Node Wizard Full Run Chain Review

> Status: CURRENT_REPORT
> Last reviewed: 2026-06-18
> Scope: Page-to-Docker run chain review
> Read order: See `docs/CURRENT.md`

Date: 2026-06-18

Conclusion: ACCEPTED_WITH_GAPS

The browser-to-Docker path is usable for formal NVIDIA local validation after this round: a user can add a node model root, browse and scan a model, create ModelArtifact/ModelLocation, enable NodeBackendRuntime, preflight, start Docker, inspect logs, stop, and clean up. Remaining product-depth gaps are tracked as formal entries in `docs/reports/model-runtime-node-wizard/open-issues-closeout.md`, not left only in this review.

| Step | Required capability | API ready | Agent ready | Web ready | Wizard ready | Status | Evidence file/function/page | Action required | Priority |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Node preparation | Agent online | Yes | Yes | Yes | N/A | READY | `internal/server/api/agent_handlers.go`, `web/src/pages/NodesPage.vue` | None | P2 |
| Node preparation | GPU recognized | Yes | Yes | Yes | N/A | READY | `internal/server/api/resource_handlers.go`, `internal/agent/collector`, `web/src/pages/GpusPage.vue` | None | P2 |
| Node preparation | Docker available | Yes | Yes | Partial | Partial | PARTIAL | `cmd/agent/main.go` `GET /docker-images`, `web/src/components/DockerImagePicker.vue` | Show Docker readiness on Node detail, not only runtime flows. | P2 |
| Node preparation | Docker images browseable | Yes | Yes | Yes | Yes | READY | `internal/server/api/agent_handlers.go` `HandleGetNodeDockerImages`, `DockerImagePicker.vue` | None | P2 |
| Node preparation | Node model roots configurable | Yes | Yes | Yes | Yes | READY | `internal/server/api/model_browser_handlers.go`, `RemoteFileBrowser.vue` | None | P0 |
| Node preparation | Runtime readiness check | Yes | Yes | Yes | Yes | READY | `internal/server/api/node_runtime_handlers.go`, `BackendRuntimesPage.vue`, `RunnerConfigsPage.vue` | None | P1 |
| Model configuration | Select Agent/node | Yes | N/A | Yes | Yes | READY | `ModelArtifactsPage.vue`, `useNodeLabels.ts` | None | P1 |
| Model configuration | Add/select model root | Yes | Yes | Yes | Yes | READY | `/api/v1/nodes/{id}/model-roots`, `RemoteFileBrowser.vue` | None | P0 |
| Model configuration | Browse model directory | Yes | Yes | Yes | Yes | READY | `HandleProxyNodeFiles`, Agent `GET /files`, `RemoteFileBrowser.vue` | None | P0 |
| Model configuration | Scan model metadata | Yes | Yes | Yes | Yes | READY | `HandleProxyNodeModelScan`, `collector.ScanModelPath`, `ModelArtifactsPage.vue` | None | P0 |
| Model configuration | Create ModelArtifact | Yes | N/A | Yes | Yes | READY | `HandleCreateArtifact`, `ModelArtifactsPage.vue` | None | P1 |
| Model configuration | Create ModelLocation with root semantics | Yes | N/A | Yes | Yes | READY | `HandleCreateModelLocation`, `resolveModelLocationRequestPath`, `ModelArtifactsPage.vue` | None | P0 |
| Model configuration | Add other node locations | Yes | Yes | Yes | Yes | READY | `HandleCreateModelLocation`, `RemoteFileBrowser.vue` add-location dialog | None | P1 |
| Model configuration | Delete/disable locations | Yes | N/A | Partial | Partial | PARTIAL | `HandleDeleteModelLocation`, `HandlePatchModelLocation`, `ModelArtifactsPage.vue` | Add explicit disable button in location table; delete exists. | P2 |
| Runtime configuration | Select Backend | Yes | N/A | Yes | Yes | READY | `BackendsPage.vue`, `ModelDeploymentsPage.vue` | None | P1 |
| Runtime configuration | Select BackendVersion | Yes | N/A | Partial | Yes | PARTIAL | `HandleListBackendVersions`, `ModelDeploymentsPage.vue` | Improve BackendVersion capability display. | P2 |
| Runtime configuration | Select/create BackendRuntime | Yes | N/A | Yes | Yes | READY | `BackendRuntimesPage.vue`, `RunnerConfigsPage.vue` | None | P1 |
| Runtime configuration | Select node and Docker image | Yes | Yes | Yes | Yes | READY | `DockerImagePicker.vue`, `RunnerConfigsPage.vue`, `BackendRuntimesPage.vue` | None | P1 |
| Runtime configuration | Enable NodeBackendRuntime | Yes | Yes | Yes | Yes | READY | `HandleEnableNodeBackendRuntime`, `BackendRuntimesPage.vue` | None | P1 |
| Runtime configuration | Recheck readiness | Yes | Yes | Yes | Yes | READY | `HandleCheckNodeBackendRuntime`, `BackendRuntimesPage.vue` | None | P1 |
| Runtime configuration | Delete/disable NodeBackendRuntime | Yes | N/A | Yes | Partial | PARTIAL | `HandleDeleteNodeBackendRuntime`, `BackendRuntimesPage.vue` | Add disable action separate from delete in Web. | P2 |
| Deployment start | Select model | Yes | N/A | Yes | Yes | READY | `ModelDeploymentsPage.vue` | None | P0 |
| Deployment start | Select Backend | Yes | N/A | Yes | Yes | READY | `ModelDeploymentsPage.vue` | None | P1 |
| Deployment start | Select BackendVersion | Yes | N/A | Yes | Yes | READY | `ModelDeploymentsPage.vue` | None | P1 |
| Deployment start | Select Runtime filtered by version | Yes | N/A | Yes | Yes | READY | `ModelDeploymentsPage.vue` `filteredRuntimes` | None | P1 |
| Deployment start | Compute ModelLocation and NodeBackendRuntime intersection | Yes | N/A | Yes | Yes | READY | `preflight_handlers.go`, `deployment_lifecycle_handlers.go` | None | P0 |
| Deployment start | GPU auto/manual | Partial | Yes | Partial | Partial | PARTIAL | `preflightDeployment`, `ModelDeploymentsPage.vue` | Add explicit GPU auto/manual controls in wizard. | P2 |
| Deployment start | Port auto/manual | Partial | N/A | Partial | Partial | PARTIAL | `ModelDeploymentsPage.vue` host port input | Add auto-port suggestion and conflict UI. | P2 |
| Deployment start | Preflight checks | Yes | Yes | Yes | Yes | READY | `HandlePreflightDeployments`, E2E script | None | P0 |
| Deployment start | Server RunPlan command preview | Yes | N/A | Yes | Yes | READY | `HandleDeploymentDryRun`, `HandleStartDeployment`, `ModelDeploymentsPage.vue` | Preview is Server-generated; pre-start preview pane is tracked in formal closeout. | P1 |
| Deployment start | Start Docker | Yes | Yes | Yes | Yes | READY | `HandleStartDeployment`, Agent `model_instance_start`, E2E `/v1/models PASS` | None | P0 |
| Runtime management | Instance status | Yes | N/A | Yes | N/A | READY | `HandleListInstances`, `ModelInstancesPage.vue` | None | P1 |
| Runtime management | Docker logs | Yes | Yes | Yes | N/A | READY | `HandleGetNodeRunPlanLogs`, Agent `model_instance_logs`, `ModelInstancesPage.vue` | None | P1 |
| Runtime management | Health check | Yes | Yes | Partial | N/A | PARTIAL | Agent Docker runtime health check, `ModelInstancesPage.vue` status | Add health-check detail panel. | P2 |
| Runtime management | Command preview | Yes | N/A | Partial | N/A | PARTIAL | `HandleGetNodeRunPlanPreview`, start result dialog | Link preview from instance/deployment detail. | P2 |
| Runtime management | Stop/delete/cleanup | Yes | Yes | Yes | N/A | READY | `HandleStopDeployment`, `HandleDeleteDeployment`, E2E cleanup | None | P0 |
| Runtime management | GPU lease release | Yes | N/A | Partial | N/A | READY | `agent_handlers.go`, `deployment_lifecycle_handlers.go`, E2E cleanup | Web lease display is tracked in formal closeout. | P2 |

Final decision: ACCEPTED_WITH_GAPS. P0 items are fixed and verified. P1 wizard blockers are fixed for the formal single-node NVIDIA Docker flow. P2 product improvements are listed in `open-issues-closeout.md` with `DOCUMENTED_BLOCKER` status.
