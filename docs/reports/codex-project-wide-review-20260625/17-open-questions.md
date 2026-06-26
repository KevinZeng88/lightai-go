# Open Questions

These are not closure blockers for this audit, but require evidence or product decisions before a release claim.

| ID | Question | Evidence Needed | Owner Decision |
| --- | --- | --- | --- |
| Q-001 | Should `/nodes/{id}/backend-runtimes/check` remain public session API? | Security decision and replacement contract. | Prefer remove or make Agent-only. |
| Q-002 | Should `/deployments/preflight` become full RunPlan preflight? | API compatibility decision and frontend impact. | Prefer full resolver or rename to candidate-check. |
| Q-003 | What is the supported parameter payload after `parameters_json` removal? | Contract doc and scripts updated to `parameter_values_json`. | Decide whether to reject old fields. |
| Q-004 | Are deployment template sync and NBR reapply/change in current phase? | Product flow and tests for running instance behavior. | If not, hide UI affordances. |
| Q-005 | What is the minimum accepted Docker security policy? | Tenant/admin policy rules for privileged, devices, host networking, mounts. | Define before multi-tenant use. |
| Q-006 | What is the accepted API contract source? | Updated OpenAPI or generated route contract. | Do not rely on stale OpenAPI. |
| Q-007 | Which E2E scripts are current? | Script inventory with pass/fail/hardware skip results. | Archive stale scripts. |
| Q-008 | What is the real MetaX readiness bar? | Real hardware run logs and adapter validation. | Keep documented blocker until complete. |
