# Phase 2F Continuation Steps

## Completed (commit 07f4ed7)

- [x] Step 0: Baseline verification
- [x] Step 1: Plan document + auth/RBAC audit
- [x] Step 2: Schema V7 (tenant type, ResourcePool tables)
- [x] Step 3A: Audit log API (`GET /api/v1/audit-logs`)
- [x] Step 9: Fix model_instances tenant isolation

## Remaining

| Step | Description | Status |
|------|-------------|--------|
| 3B | Audit log Web page | pending |
| 4 | Tenant management Web page | pending |
| 5 | User management Web page | pending |
| 6 | Role management Web page | pending |
| 7 | Active tenant switching | pending |
| 8 | Node/GPU transfer hardening | pending |
| 10 | RBAC tests | pending |
| 11 | Docs | pending |
| 12 | Final regression | pending |

## Verification per step

Each step verifies: `go test ./...`, `go build ./cmd/server`, `go build ./cmd/agent`.
Web steps also: `cd web && npm run build`.
