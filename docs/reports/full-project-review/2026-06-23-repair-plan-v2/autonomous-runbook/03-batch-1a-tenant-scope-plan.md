# Batch 1A: Tenant Scope — Detailed Plan

> Absorbs: `08-batch-1a-1b-execution-plan.md` §Batch 1A

---

## Goal
Add tenant scope checks to 16 endpoints. Fix rate limiter, CSRF, observability auth.

## Endpoint Matrix (16 endpoints, verified by code)

### Domain A: Node Proxy (4 endpoints) — Check `nodes.tenant_id`

| # | Handler | File:Line | Route | Fix Point |
|---|---------|-----------|-------|-----------|
| 1 | HandleProxyNodeFiles | agent_proxy_handlers.go:13 | GET /nodes/{id}/files | Before line 52 (http.Get) |
| 2 | HandleProxyNodeModelScan | agent_proxy_handlers.go:79 | POST /nodes/{id}/model-paths/scan | Before line 110 (http.Post) |
| 3 | HandleGetNodeDockerImages | agent_handlers.go:594 | GET /nodes/{id}/docker-images | Before line 612 (http.Get) |
| 4 | HandleGetNodeDockerImageInspect | agent_handlers.go:629 | GET /nodes/{id}/docker-image-inspect | Before line 650 (http.Get) |

**Fix pattern** (insert before existing DB query):
```go
var nodeTenant string
if err := h.DB.QueryRow("SELECT tenant_id FROM nodes WHERE id=?", nodeID).Scan(&nodeTenant); err != nil {
    writeError(w, http.StatusNotFound, "node not found"); return
}
if !tenantScopeCheck(r, nodeTenant) {
    http.Error(w, "not found", http.StatusNotFound); return
}
```

### Domain B: Node Model Roots (4 endpoints) — Check `node_model_roots.tenant_id`

| # | Handler | File:Line | Route | Fix Point |
|---|---------|-----------|-------|-----------|
| 5 | HandleListNodeModelRoots | model_browser_handlers.go:162 | GET /nodes/{id}/model-roots | After nodeTenant() call |
| 6 | HandleAddNodeModelRoot | model_browser_handlers.go:182 | POST /nodes/{id}/model-roots | After nodeTenant() call |
| 7 | HandlePatchNodeModelRoot | model_browser_handlers.go:232 | PATCH /nodes/{id}/model-roots/{root_id} | After resolveNodeModelRoot() |
| 8 | HandleDeleteNodeModelRoot | model_browser_handlers.go:261 | DELETE /nodes/{id}/model-roots/{root_id} | After resolveNodeModelRoot() |

### Domain C: Node Backend Runtimes (6 endpoints) — Check `node_backend_runtimes.tenant_id`

| # | Handler | File:Line | Route | Fix Point |
|---|---------|-----------|-------|-----------|
| 9 | HandleListNodeBackendRuntimes | runtime_handlers.go:248 | GET /nodes/{id}/backend-runtimes | Before DB query |
| 10 | HandleEnableNodeBackendRuntime | runtime_handlers.go:299 | POST /nodes/{id}/backend-runtimes/enable | Before upsertNodeBackendRuntime |
| 11 | HandleRequestNodeBackendRuntimeCheck | runtime_handlers.go:319 | POST /nodes/{id}/backend-runtimes/{nbr_id}/check-request | After NBR lookup |
| 12 | HandleGetNodeBackendRuntimeProbe | runtime_handlers.go:645 | GET /nodes/{id}/backend-runtimes/{nbr_id}/probe | After NBR lookup |
| 13 | HandlePatchNodeBackendRuntime | node_runtime_handlers.go:98 | PATCH /nodes/{id}/backend-runtimes/{nbr_id} | Before UPDATE |
| 14 | HandleDeleteNodeBackendRuntime | node_runtime_handlers.go:170 | DELETE /nodes/{id}/backend-runtimes/{nbr_id} | Before DELETE |

### Domain D: Model Location (2 endpoints) — Check `model_locations.tenant_id`

| # | Handler | File:Line | Route | Fix Point |
|---|---------|-----------|-------|-----------|
| 15 | HandleRescanModelLocation | artifact_handlers.go:549 | POST /model-artifacts/{id}/locations/{location_id}/rescan | Before UPDATE |
| 16 | HandleAttestModelLocation | artifact_handlers.go:559 | POST /model-artifacts/{id}/locations/{location_id}/attest | Before UPDATE |

## Authz Helpers

New file: `internal/server/authz/checks.go`

```go
package authz

func CheckNodeTenant(r *http.Request, db *sql.DB, nodeID string) bool
func CheckNBRTenant(r *http.Request, db *sql.DB, nbrID string) bool
func CheckModelRootTenant(r *http.Request, db *sql.DB, rootID string) bool
func CheckModelLocationTenant(r *http.Request, db *sql.DB, locationID string) bool
```

Each: query tenant_id → compare with session tenant → admin bypass.

## Commits

1. `feat: add authz package with tenant ownership checks`
2. `feat: add tenant scope checks to 16 endpoints`
3. `fix: rate limiter XFF, CSRF rotation, observability auth`

## Tests

- `internal/server/authz/checks_test.go` — 10 tests
- Extend `internal/server/api/tenant_isolation_test.go` — 10+ tests

## Non-Regression

| Check | Method |
|-------|--------|
| Same-tenant file browse | GET /nodes/{id}/files → 200 |
| Cross-tenant file browse | GET /nodes/{id}/files → 404 |
| Admin access all | Admin session → 200 |
| NBR list same-tenant | GET /nodes/{id}/backend-runtimes → 200 |
| NBR patch cross-tenant | PATCH /nodes/{id}/backend-runtimes/{nbr_id} → 404 |
