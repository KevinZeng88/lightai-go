# AUTORUN Closeout Report
Status: **B0+B1 CLOSED, B2-B8 NOT_STARTED**
Date: 2026-06-25
Commits: fd75d29 (B0), 3d5b501 (B1)

## Executed Batches

| Batch | Status | Commit | Summary |
|-------|--------|--------|---------|
| B0 | CLOSED | fd75d29 | Workspace baseline + 6 inventory docs |
| B1 | CLOSED | 3d5b501 | R-001: Delete /check, enforce server-proxied NBR readiness |
| B2-B8 | NOT_STARTED | — | — |

## Risk Register

| ID | Severity | Status |
|----|----------|--------|
| R-001 | P0 | CLOSED — /check route deleted; /check-request is sole readiness path |
| R-002 | P1 | NOT_STARTED (B3) |
| R-003 | P1 | NOT_STARTED (B2) |
| R-004 | P1 | NOT_STARTED (B4) |
| R-005 | P1 | NOT_STARTED |
| R-006 | P1 | NOT_STARTED (B3) |
| R-007 | P1 | NOT_STARTED (B5A) |
| R-008 | P1 | NOT_STARTED (B5B) |
| R-009 | P2 | NOT_STARTED (B5C) |
| R-010 | P2 | NOT_STARTED |
| R-011 | P2 | NOT_STARTED (B4) |
| R-012 | P2 | NOT_STARTED (B8) |
| R-013 | P2 | NOT_STARTED (B6) |
| R-014 | P2 | NOT_STARTED (B8) |
| R-015 | P3 | NOT_STARTED (B7) |

## /check Route: DELETED

- Route `POST /api/v1/nodes/{id}/backend-runtimes/check` removed from router
- Handler returns 410 Gone (deprecated, use /enable or /check-request)
- Session callers cannot access /check
- E2E scripts updated to use /enable + /check-request

## Verification

- go test ./internal/server/api/... — ALL PASS
- go test ./internal/server/runplan/... — PASS
- go test ./... — PASS
- go build ./cmd/server/ — PASS
- go build ./cmd/agent/ — PASS
- npm test — PASS
- npm run build — PASS

## Files Changed (B1)

M internal/server/api/router.go
M internal/server/api/runtime_handlers.go
M internal/server/api/runtime_boundary_test.go
M internal/server/api/ui_persistence_runplan_test.go
M internal/server/api/workflow_deployment_runplan_test.go
M scripts/e2e-*.sh (5 files)

## Git Status

Clean (no uncommitted code changes; untracked items are baseline evidence/docs)
