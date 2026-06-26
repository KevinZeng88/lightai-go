# Current Code Audit

Date: 2026-06-26

Commands run:

```bash
git status --short
git rev-parse --abbrev-ref HEAD
git log --oneline -5
find configs/backend-catalog -type f | sort
rg -n "HumanRuntimeParameterForm|getHumanFieldsForBackend|runtime\.|template-only|from Metax release package|0d307f1665d3|mergeNBRConfigSnapshot|resolveImage|VendorOptionsJSON|BackendVersion.defaultImages|config_set_json" web/src configs internal/server internal/server/runplan
```

Findings closed in this change:

| ID | Finding | Status | Closure |
| -- | ------- | ------ | ------- |
| AUDIT-001 | BackendVersion create/patch accepted runtime-only fields such as `image_ref`, `command`, `entrypoint`, and `model_mount`. | FIXED | API now returns 400 and points the caller to BackendRuntime. |
| AUDIT-002 | Deployment create could fall back from missing NBR snapshot to BackendRuntime snapshot. | FIXED | Deployment create now rejects empty NBR ConfigSet snapshots. |
| AUDIT-003 | RunPlan image resolution could use BackendVersion default image fallback. | FIXED | RunPlan now uses NodeRuntimeOverride or BackendRuntime/NBR snapshot image only. |
| AUDIT-004 | Editable runtime UI used `HumanRuntimeParameterForm` and hardcoded parameter lists. | FIXED | BackendRuntime edit and NBR wizard use `RuntimeParameterEditor` driven by `config_set.items`. |
| AUDIT-005 | BackendVersion had API support but no UI management entry. | FIXED | `BackendsPage.vue` now includes version list, clone, create, edit, delete, and add-parameter UI. |
| AUDIT-006 | RuntimeParameterEditor did not fully honor schema metadata. | FIXED | It now reads render/extensions labels/help/groups, top-level constraints, order, select/multi-select options, and hides hidden/internal items. |
| AUDIT-007 | Runtime template ordinary selector included hidden/reference/placeholder templates. | FIXED | Runtime catalog has visibility/support metadata and default list filtering. Visible uniqueness is validated. |
| AUDIT-008 | `configSetParameterValues()` had an unreachable env branch. | FIXED | Env items with `render.env_name` are extracted for env target; map-valued `runtime.env` is not converted into CLI args. |
