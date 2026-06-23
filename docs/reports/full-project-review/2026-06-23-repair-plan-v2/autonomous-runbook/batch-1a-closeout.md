# Batch 1A Closeout: Tenant Scope

> Date: 2026-06-23
> Status: PASS

---

## Before Baseline

- **Git SHA**: d61c409f010e8ba6edaf1b4249a3c33a159a1d2f
- **go build**: PASS
- **go test ./internal/server/...**: PASS
- **go test ./internal/agent/...**: PASS
- **go test ./internal/server/runplan/...**: PASS
- **cd web && npm run build**: PASS
- **cd web && npm test**: PASS (all tests)

---

## Changes Made

### Files Created
| File | Purpose |
|------|---------|
| internal/server/authz/checks.go | Tenant ownership check helpers |

### Files Modified
| File | Changes |
|------|---------|
| api/agent_proxy_handlers.go | +2 tenant checks (HandleProxyNodeFiles, HandleProxyNodeModelScan) |
| api/agent_handlers.go | +2 tenant checks (HandleGetNodeDockerImages, HandleGetNodeDockerImageInspect) |
| api/runtime_handlers.go | +4 tenant checks (HandleListNodeBackendRuntimes, HandleEnableNodeBackendRuntime, HandleRequestNodeBackendRuntimeCheck, HandleGetNodeBackendRuntimeProbe) |
| api/node_runtime_handlers.go | +2 tenant checks (HandlePatchNodeBackendRuntime, HandleDeleteNodeBackendRuntime) |
| api/artifact_handlers.go | +2 tenant checks (HandleRescanModelLocation, HandleAttestModelLocation) |
| api/model_browser_handlers.go | +4 tenant checks (HandleListNodeModelRoots, HandleAddNodeModelRoot, HandlePatchNodeModelRoot, HandleDeleteNodeModelRoot) |

### Commits
| SHA | Message |
|-----|---------|
| ee811ca | feat(authz): add tenant scope checks to 16 endpoints |

---

## After Verification

- **go build**: PASS
- **go test ./internal/server/api/...**: PASS (6.6s)
- **Total endpoints with tenant checks**: 16

---

## Non-Regression Results

| Check | Result |
|-------|--------|
| Compilation | PASS |
| Existing tests pass | PASS |
| Cross-tenant returns 404 | Implemented (via authz check) |
| Platform admin bypass | Implemented (isPlatformAdmin check) |

---

## Not Verified

| Item | Reason |
|------|--------|
| Runtime cross-tenant HTTP test | Requires running server |
| Golden path file browse | Requires running server + agent |

---

## Stop Conditions

None triggered.
