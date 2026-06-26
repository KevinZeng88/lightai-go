# Batch Order and Dependency Review

## Order Assessment

The high-level order is mostly correct:

1. Baseline inventory.
2. Contract/readiness hardening.
3. RunPlan/preflight convergence.
4. E2E/OpenAPI/documentation convergence.
5. UI repair.
6. Security/tenant hardening.
7. Reliability.
8. Performance.
9. Product scope boundaries.

This sequence avoids most dependency inversions. Contract and RunPlan work precede E2E/OpenAPI and UI, which is the right order.

## Required Splits

| Current Batch | Issue | Recommended Split |
| --- | --- | --- |
| Batch 5 | Combines Agent credentials, endpoint exposure, Docker policy, schema cleanup, and RBAC tests. | Split into Batch 5A Agent credentials, Batch 5B Docker policy, Batch 5C tenant/schema/RBAC. Each gets its own commit and closeout. |
| Batch 4 | Includes removing misleading UI field, optional full NBR change flow, aggregate endpoint, parameter editor audit, Playwright. | Default sub-order: remove misleading field first, add aggregate endpoint second, browser smoke third. Full NBR change flow should be a separate explicit sub-batch if implemented. |
| Batch 3 | Mixes script repair, OpenAPI rewrite, evidence marking, new E2E, NVIDIA smoke. | Run as inventory-driven substeps: active script manifest, contract docs/OpenAPI, API dry-run E2E, then hardware smoke script. |

## Dependency Notes

| Dependency | Review |
| --- | --- |
| Batch 1 before Batch 3 | Correct. Stale scripts cannot be fixed safely until `/check` and payload rules are final. |
| Batch 2 before Batch 4 | Correct. UI preflight behavior should follow final API semantics. |
| Batch 4 aggregate endpoint before Batch 7 fan-out closure | Correct, but Batch 7 should not re-open API shape. |
| Batch 5 before Batch 6 real operational hardening | Mostly correct. Lease/task tests can proceed before Agent token changes, but real smoke should happen after security policy is stable. |
| Batch 8 after Batch 2 | Multi-replica rejection belongs in Batch 2 for API consistency and Batch 8 for docs/UI scope. This is handled. |

## Parallelizable Work

- OpenAPI route inventory and stale script inventory can happen in Batch 0/3 without waiting for UI changes.
- Documentation product-scope cleanup for R-014 can be drafted independently, but final wording should wait for Batch 8 decisions.
- Performance audit docs can start before code splitting, but endpoint changes should wait until aggregate NBR shape is final.

## Smoke-Gated Closure

The following must not be marked fully closed without runtime or explicit external-blocker evidence:

| Item | Gate |
| --- | --- |
| R-002 E2E trust | At least one current API dry-run E2E passes. NVIDIA smoke may SKIP only with environment evidence. |
| R-007 Agent token | Node-bound negative tests pass; real agent registration path is covered or explicitly blocked by missing runtime environment. |
| R-008 Docker policy | Save/preview/start negative tests pass; real smoke proves allowed safe path still starts. |
| Q-008 MetaX | Real hardware evidence or `BLOCKED_BY_EXTERNAL_DEPENDENCY`. |

## Dependency Verdict

Batch order is directionally sound, but Batch 5 and parts of Batch 4/3 should be split before AUTORUN to keep commits reviewable and failures diagnosable.
