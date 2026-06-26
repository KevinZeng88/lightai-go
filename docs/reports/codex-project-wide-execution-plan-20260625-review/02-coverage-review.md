# Coverage Review

## Risk and Question Coverage

| Original ID | Severity / Type | Source | Covered By Plan | Adequate? | Comment |
| ----------- | --------------- | ------ | --------------- | --------- | ------- |
| R-001 | P0 | `10-risk-register.md` | Batch 1 | Partial | Covered, but Batch 1 allows either wrapper or removal. AUTORUN needs one default path and must forbid request-body readiness evidence. |
| R-002 | P1 | `10-risk-register.md` | Batch 3 | Yes | Stale scripts, active scripts, and current E2E are covered. Needs explicit CI/active-list command. |
| R-003 | P1 | `10-risk-register.md` | Batch 2 | Yes | Final RunPlan preflight and `ready_with_warnings` consistency are clearly specified. |
| R-004 | P1 | `10-risk-register.md` | Batch 4 | Partial | Covered, but allows either remove field or implement full NBR change flow. AUTORUN should default to removal first. |
| R-005 | P1 | `10-risk-register.md` | Batch 1 + Batch 2 | Partial | Covered, but DB migration/snapshot repair policy needs backup and fresh-DB verification commands. |
| R-006 | P1 | `10-risk-register.md` | Batch 3 | Partial | OpenAPI update is covered; sample validation and route-to-OpenAPI diff are not concrete enough. |
| R-007 | P1 | `10-risk-register.md` | Batch 5 | Partial | Covered, but "first stage" wording can close without full node-bound proof. Needs exact minimum implementation. |
| R-008 | P1 | `10-risk-register.md` | Batch 5 | Partial | Covered, but policy default, route enforcement points, and negative tests need sharper acceptance. |
| R-009 | P2 | `10-risk-register.md` | Batch 5 | Partial | Covered, but plan should require fresh DB schema test plus no handler-side `CREATE TABLE IF NOT EXISTS`. |
| R-010 | P2 | `10-risk-register.md` | Batch 5 | Yes | Negative matrix and coverage targets are covered. |
| R-011 | P2 | `10-risk-register.md` | Batch 4 + Batch 7 | Yes | Aggregate NBR endpoint and fan-out removal are covered. |
| R-012 | P2 | `10-risk-register.md` | Batch 2 + Batch 8 | Yes | Plan defaults to rejecting/hiding replicas > 1 until supported. |
| R-013 | P2 | `10-risk-register.md` | Batch 6 + Batch 8 | Partial | Covered, but final close criteria should require UI/docs/API all use one observability support statement. |
| R-014 | P2 | `10-risk-register.md` | Batch 8 | Yes | Product scope reduction and design doc are covered. |
| R-015 | P3 | `10-risk-register.md` | Batch 7 | Partial | Covered, but "accepted threshold" could leave the warning unresolved without a hard reason. |
| Q-001 | Decision | `17-open-questions.md` | Batch 1 | Partial | Default decision exists but still permits two implementation paths. Must choose one for AUTORUN. |
| Q-002 | Decision | `17-open-questions.md` | Batch 2 | Yes | Default final resolver preflight is clear. |
| Q-003 | Decision | `17-open-questions.md` | Batch 1 + Batch 3 | Yes | `parameter_values_json` only and `parameters_json` 400 are clear. |
| Q-004 | Decision | `17-open-questions.md` | Batch 4 | Partial | AUTORUN should default to remove the misleading edit field; full NBR change flow is a separate batch. |
| Q-005 | Decision | `17-open-questions.md` | Batch 5 | Partial | Default deny is stated, but exact allowlist and role/policy matrix need to be inserted. |
| Q-006 | Decision | `17-open-questions.md` | Batch 3 | Yes | OpenAPI plus contract tests are specified. |
| Q-007 | Decision | `17-open-questions.md` | Batch 0 + Batch 3 | Partial | Inventory and archive are covered; plan needs an explicit active-script manifest and command that fails on stale active scripts. |
| Q-008 | Decision | `17-open-questions.md` | Batch 8 + Smoke Plan | Partial | Correctly keeps MetaX as hardware-gated, but final status should be `BLOCKED_BY_EXTERNAL_DEPENDENCY` unless real hardware evidence exists. |

## Executive Summary Top Risks

| Top Risk | Covered By Plan | Adequate? | Comment |
| --- | --- | --- | --- |
| Client-trusted NBR readiness | Batch 1 | Partial | Needs one hard default and a false-ready regression test. |
| Stale E2E scripts | Batch 0 + Batch 3 | Yes | Needs active-script manifest to avoid ambiguous archival. |
| Preflight not final RunPlan | Batch 2 | Yes | Strong plan. |
| Misleading deployment edit runtime selector | Batch 4 | Partial | Default should be removal, not optional full workflow. |
| Snapshot mutation/legacy branch | Batch 1 + Batch 2 | Partial | Needs backup/fresh-DB migration boundary. |
| Stale OpenAPI | Batch 3 | Partial | Needs validation against samples/routes. |
| Shared Agent bearer token | Batch 5 | Partial | Needs split batch and minimum node-bound token acceptance. |
| Docker dangerous options | Batch 5 | Partial | Needs explicit policy matrix and enforcement points. |
| Weak coverage | Batch 5 + global validation | Yes | Sufficient if negative tests are added. |
| Dirty workspace baseline | Batch 0 + commit strategy | Partial | Current status is dirty; plan needs hard commit inclusion/exclusion rules. |

## Original Next-Development Batches

| Original Batch | Covered By Plan | Adequate? | Comment |
| --- | --- | --- | --- |
| Batch 1 contract/readiness hardening | Plan Batch 1 + Batch 2 | Partial | Coverage good; default decisions need hardening. |
| Batch 2 E2E/documentation convergence | Plan Batch 3 | Partial | Add OpenAPI sample validation and active-script manifest. |
| Batch 3 UI workflow repair | Plan Batch 4 | Partial | Default should remove misleading runtime selector first. |
| Batch 4 security/tenant hardening | Plan Batch 5 | Partial | Too large for one autonomous batch. |
| Batch 5 scale/reliability | Plan Batch 6 + Batch 7 | Yes | Reasonable order after contract/security work. |
