# Frontend Review

## Strengths

- Vue 3 + Element Plus console covers nodes, GPUs, artifacts, runtimes, runner configs, deployments, instances, observability, RBAC, audit logs.
- RuntimeParameterEditor centralizes many parameter and Docker option controls.
- NBR deployability is read from API `deployable` rather than re-derived from status in deployment selection.
- i18n key tests and runtime boundary UI tests catch several regressions.

## Findings

| Finding | Evidence | Impact | Recommendation |
| --- | --- | --- | --- |
| Deployment edit runtime selector is misleading. | `ModelDeploymentsPage.vue` edit form field `backend_runtime_id`; `doEdit()` omits it. | User can think runtime changed while backend ignores it. | Remove field or implement explicit NBR change flow. |
| Deployment create dialog remains ID-heavy. | Create dialog asks artifact ID/node ID while wizard is product path. | Manual ID entry contradicts wizard goal. | Hide simple create dialog or make it advanced/admin only. |
| Per-node NBR fan-out. | `loadAllNBRs()` loops nodes and calls `/nodes/{id}/backend-runtimes`. | Slower with more nodes. | Add aggregate NBR endpoint. |
| Static tests dominate. | `web/tests/*.mjs` inspect source strings and i18n; no browser smoke was found. | UX regressions can pass tests. | Add Playwright smoke for core wizard. |
| Main bundle is large. | `npm run build` warning. | Initial load cost grows. | Code split heavy routes/vendors. |

## UI-specific next steps

- Normalize labels around BackendRuntime template vs NodeBackendRuntime config.
- Remove or gate outdated dialogs that bypass wizard assumptions.
- Add visible warnings for deployable-with-warnings NBRs.
- Add explicit display for snapshot source and whether deployment is stale vs template.
