# Performance and Scalability Review

## Current expected scale

The architecture is suitable for small deployments and a few GPU servers. It is not yet designed as a high-cardinality scheduler or large fleet control plane.

## Findings

| Area | Finding | Evidence | Impact |
| --- | --- | --- | --- |
| SQLite | WAL is enabled, but there is no broad pagination/index audit. | `db.Open()` uses WAL; many list handlers order all rows. | Fine for small deployments; list/query pressure grows with nodes, logs, artifacts, instances. |
| N+1 frontend calls | Deployment and runtime pages fetch NBRs per node. | `ModelDeploymentsPage.vue` `loadAllNBRs()`, `BackendRuntimesPage.vue` `loadNodeRuntimes()`. | More nodes means page load fan-out. |
| Docker image checks | `/check-request` calls `/docker-images` and `/docker-image-inspect`. | `runtime_handlers.go`. | Image list/inspect can be slow with many images or unreachable agents. Cache/probe history may be needed. |
| Logs | Docker logs endpoint supports tail, but large log handling is bounded only by requested tail and Docker stream decoder max payload. | `docker_real.go` `maxStreamPayload=100MB`. | Need API tail defaults and hard limits for UI. |
| Web bundle | Build warns `index` JS chunk is 1,313 kB minified / 423 kB gzip. | `npm run build` output. | Acceptable now; code splitting needed before UI grows. |
| Multi-replica | DB fields exist, but flow inserts one instance and one run plan group in `mode='single'`. | `HandleStartDeployment`. | Replicas/distributed are not implemented. |

## Recommendations

- Add pagination and indexes for list-heavy APIs before larger scale testing.
- Add aggregate NBR endpoint to avoid per-node frontend fan-out.
- Add probe result caching and background refresh for Docker image checks.
- Add explicit log API max tail/bytes.
- Split the frontend main chunk with manual chunks or route-level dynamic imports where not already effective.
