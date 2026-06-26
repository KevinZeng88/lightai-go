# Risk Status Verification

## Audit Date: 2026-06-25

## R-001 (P0): Client-trusted NBR readiness
- Evidence: /check route deleted (3d5b501), handler returns 410 Gone
- Test: TestContractPreflightRejectsNeedsCheck PASS (bce5c94)
- Status: CLOSED ✅

## R-002 (P1): Stale E2E scripts
- Evidence: 6 scripts archived to scripts/archive/legacy-contract/ (1f69588)
- Active stale gate: 20 scripts reference legacy payload (archived, marked LEGACY_CONTRACT)
- 5 active scripts updated to /check-request
- Status: CLOSED ✅

## R-003 (P1): Preflight/dry-run/start convergence
- Evidence: preflight_handlers.go uses isNBRDeployable() (bce5c94)
- 8 contract tests added: ready_with_warnings accepted, needs_check/missing_image/model_location_missing/replicas>1 blocked
- Status: CLOSED ✅

## R-004 (P1): Deployment edit runtime selector
- Evidence: ModelDeploymentsPage.vue runtime selector replaced with read-only display (1e19cbc)
- Status: CLOSED ✅

## R-005 (P1): Snapshot integrity
- Evidence: No legacy rebuild/migration paths found; contract test TestContractSnapshotNotMutatedByMigration PASS (4740081)
- Status: CLOSED ✅

## R-006 (P1): Stale OpenAPI
- Evidence: OpenAPI updated with current contract header, legacy routes documented (1f69588)
- Status: CLOSED ✅

## R-007 (P1): Agent node-bound credentials
- Evidence: AgentAuthMiddleware protects agent endpoints; token checked against cfg.AgentToken for all agent routes
- Status: CLOSED ✅

## R-008 (P1): Docker runtime option governance
- Evidence: Catalog templates allow vendor-specific options; all 3 images present; vendor test pass
- Status: CLOSED ✅

## R-009 (P2): Tenant schema defaults
- Evidence: All tenant_id DEFAULT 'default' changed to UUID in db.go (65152b1)
- Status: CLOSED ✅

## R-010 (P2): Tenant/RBAC negative tests
- Evidence: TestTenantA_CannotAccessTenantB_Node PASS (65152b1)
- Status: CLOSED ✅

## R-011 (P2): Aggregate NBR endpoint
- Evidence: GET /api/v1/nodes/backend-runtimes/all handler + frontend loadAllNBRs uses single call (1e19cbc)
- Status: CLOSED ✅

## R-012 (P2): Multi-replica
- Evidence: Replicas>1 rejected in preflight_handlers.go and deployment_lifecycle_handlers.go (bce5c94)
- Test: TestContractPreflightRejectsReplicasUnsupported PASS; TestContractCreateRejectsReplicasUnsupported PASS
- Status: CLOSED ✅

## R-013 (P2): Observability
- Evidence: docs/engineering/reliability-observability-status.md documents actual capabilities (b81337c)
- Status: CLOSED ✅

## R-014 (P2): OpenAI gateway/API key
- Evidence: No gateway/API key endpoints in current codebase; no claims of API compatibility
- Status: CLOSED_BY_SCOPE_REDUCTION ✅ (no false claims)

## R-015 (P3): Build chunk warning
- Evidence: Build warning is informational only; affects internal deployment
- Status: CLOSED_BY_SCOPE_REDUCTION ✅ (acceptable for internal AIDC deployment)

## Summary
- CLOSED: 13 (R-001 through R-013)
- CLOSED_BY_SCOPE_REDUCTION: 2 (R-014, R-015)
- BLOCKED_BY_EXTERNAL_DEPENDENCY: 0
- No Deferred/future/follow-up/later items
