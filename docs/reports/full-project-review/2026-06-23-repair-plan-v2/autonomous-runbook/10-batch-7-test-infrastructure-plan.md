# Batch 7: Test Infrastructure — Detailed Plan

---

## Goal
Add tests per batch. Finalize mock E2E.

## Current Test State

| Area | Files | Coverage |
|------|-------|----------|
| auth package | 0 test files | Zero |
| tenant isolation | 1 file, 5 tests | Nodes/GPUs only |
| RunPlan resolver | 1 file, 50+ tests | Strong |
| Docker lifecycle | 1 file | Partial |
| Frontend | 7 files (static) | No component tests |
| E2E | 20 scripts | All require real GPU |

## Per-Batch Test Additions

| Batch | Tests Added |
|-------|-------------|
| 1A | authz/checks_test.go (10), extend tenant_isolation_test.go (10+) |
| 1B | agentclient/client_test.go (13) |
| 1C | Agent endpoint auth tests |
| 2 | Docker lifecycle mock tests, race tests |
| 3 | Body limit tests, redaction tests |
| 4 | Fix resolver_test.go assertions (9.4, 9.5) |
| 6 | Frontend component tests |

## Assertion Bug Fixes

### 9.4 TestNoVarSyntax (resolver_test.go:278-288)
```go
// Current: empty if body
if strings.Contains(...) {
    // correct
} else {
    t.Errorf("expected ${MAX_MODEL_LEN} preserved, got: %v", plan.Args)
}
```

### 9.5 TestTenantAdminCannotTransferOtherTenantNode (agent_identity_test.go:257)
```go
// Current: t.Logf
if w.Code != 403 {
    t.Errorf("expected 403, got %d", w.Code)
}
```

## Mock E2E Framework

New: `scripts/e2e-mock-smoke.sh`
- Start server with mock collector
- Start agent with mock Docker
- Login, list nodes, browse files, scan models
- Create deployment (dry-run)
- Check RunPlan preview
- No real GPU required

## Race Testing

Add to CI/test scripts:
```bash
go test -race ./internal/server/...
go test -race ./internal/agent/...
go test -race ./cmd/agent/...
```

## Final Verification Matrix

| Test | Command | Requires |
|------|---------|----------|
| Auth unit tests | go test ./internal/server/auth/... | Go |
| Tenant tests | go test ./internal/server/api/... -run Tenant | Go |
| RunPlan tests | go test ./internal/server/runplan/... | Go |
| Docker tests | go test ./internal/agent/runtime/... | Go |
| Race detection | go test -race ./... | Go |
| Frontend tests | cd web && npm test | Node.js |
| Mock E2E | scripts/e2e-mock-smoke.sh | Go |
| Real E2E | scripts/e2e-real-smoke-all-three.sh | GPU+Docker |

## Commits

1. `test: add auth and tenant isolation tests`
2. `fix: assertion bugs in resolver and identity tests`
3. `test: add mock E2E framework`
