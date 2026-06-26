# Risk and Open Question Closure Review

## Risks That Current Plan Can Close After Amendments

| Risk | Closure Status Under Current Plan | Needed Before AUTORUN |
| --- | --- | --- |
| R-001 | Closeable | Hard default for `/check`; no client evidence path. |
| R-002 | Closeable | Active-script manifest plus stale-script failing scan. |
| R-003 | Closeable | Current plan is sufficient. |
| R-004 | Closeable | Default to remove misleading selector first. |
| R-005 | Closeable | Explicit migration/snapshot backup and fresh-DB test. |
| R-006 | Closeable | OpenAPI validation command and stale path scan. |
| R-007 | Closeable | Split Agent credential batch and require node-bound negative tests. |
| R-008 | Closeable | Explicit Docker policy matrix and enforcement at save/preview/start. |
| R-009 | Closeable | Fresh schema test plus handler-side table-creation cleanup rule. |
| R-010 | Closeable | Current plan is sufficient if Batch 5 is split. |
| R-011 | Closeable | Current plan is sufficient. |
| R-012 | Closeable | Current plan is sufficient. |
| R-013 | Closeable | Add one source of truth for observability claim across docs/UI/API. |
| R-014 | Closeable by scope reduction | Current plan is sufficient if UI/docs/API do not imply support. |
| R-015 | Closeable | Do not allow vague accepted threshold; require measured threshold if not splitting. |

## Open Questions

| Question | Current Plan Decision | Review Decision |
| --- | --- | --- |
| Q-001 `/check` | Remove or wrapper. | Choose one default: keep route only as server-to-Agent probe wrapper; request body evidence ignored; UI uses `/check-request`. Direct session readiness mutation removed. |
| Q-002 preflight | Full final RunPlan preflight. | Adequate. |
| Q-003 payload | `parameter_values_json` only; `parameters_json` 400. | Adequate. |
| Q-004 NBR reapply/change | Remove misleading UI or implement full flow. | Default to remove field. Full NBR change flow is not required to close R-004. |
| Q-005 Docker policy | Default deny dangerous options, admin+policy allow. | Needs exact matrix. Default deny all dangerous options for all tenants unless explicit platform policy enables each option. |
| Q-006 API contract | OpenAPI + tests. | Adequate with validation tooling. |
| Q-007 current E2E scripts | Inventory then repair/archive. | Needs active-script manifest and grep gate. |
| Q-008 MetaX | Hardware evidence or blocker. | Adequate only if final status is external blocker when hardware absent. Do not use an intentional deferral status. |

## Status Vocabulary Review

The plan allows:

```text
INTENTIONALLY_DEFERRED_WITH_OWNER_AND_ACCEPTANCE
```

That status is risky for this execution because the user explicitly wants all problems covered, not parked. Replace its use for R-001 to R-015 with:

- `CLOSED`
- `CLOSED_BY_SCOPE_REDUCTION`
- `BLOCKED_BY_EXTERNAL_DEPENDENCY`

If a design item is not implemented, close it by scope reduction only when UI/API/docs actively stop claiming support.
