# Phase 01 - Design and Inventory

## 1. Phase Objective

Checkpoint A: design document + inventory + old-structure deletion list.

This phase does not modify business code, API code, UI code, schema, scripts, or runtime behavior. It establishes the ConfigSet design baseline and records the deletion inventory for Checkpoints B-E.

## 2. Actual Changed Files

| File | Change |
| --- | --- |
| `docs/design/catalog-configset-and-runtime-snapshot.md` | Confirmed synchronized with `02-configset-configitem-design.md`; no content diff. |
| `docs/reports/codex-configset-refactor-20260625/manifest.json` | Added `06-progress-reporting-and-commit-policy.md` to `files`. |
| `docs/reports/codex-configset-refactor-20260625/execution/execution-status.md` | Created Checkpoint A execution status. |
| `docs/reports/codex-configset-refactor-20260625/execution/validation-log.md` | Created command log and inventory summary. |
| `docs/reports/codex-configset-refactor-20260625/execution/open-issues.md` | Created open issue tracker for Checkpoint A and next phases. |
| `docs/reports/codex-configset-refactor-20260625/execution/commit-log.md` | Created commit log with pending Checkpoint A entry. |
| `docs/reports/codex-configset-refactor-20260625/execution/phase-01-design-and-inventory.md` | Created this phase report. |

## 3. Deleted Old Structures / APIs / Fields

None deleted in Checkpoint A. This checkpoint records the deletion list only.

Old structures to delete/replace in Checkpoints B-E:

- `capabilities_json`
- `parameter_schema_json`
- `parameter_values_json`
- `env_json`
- `ports_json`
- `volumes_json`
- `devices_json`
- `health_check_json`
- `resource_controls_json`
- `parameters_json`
- `default_args_json`
- `parameter_defs_json`
- `default_backend_params_json`
- `default_images_json`
- `image_candidates_json`
- `docker_options_json`
- `model_mount_json`
- `seedBuiltInBackends`
- `seedTargetBackendCatalog`
- `repairBackendCapabilitiesV27`
- old `/check` route references
- runtime evidence patterns: `preflight PASS only`, `task claimed only`, `image/model present only`

## 4. New Structures / APIs / Files Added

New execution files:

- `execution-status.md`
- `validation-log.md`
- `open-issues.md`
- `commit-log.md`
- `phase-01-design-and-inventory.md`

No new runtime structures or APIs were added in Checkpoint A.

## 5. Tests / Validation Executed

| Command | Result | Summary |
| --- | --- | --- |
| `diff -u docs/reports/codex-configset-refactor-20260625/02-configset-configitem-design.md docs/design/catalog-configset-and-runtime-snapshot.md` | PASS | No diff; design doc is synchronized. |
| `git status --short` | PASS | Captured current dirty worktree. |
| `git log --oneline -20` | PASS | Captured recent history. |
| `rg -l <old-structure-patterns> ... \| wc -l` | PASS | 328 files contain old structure patterns. |
| `rg -l <backend-name-patterns> ... \| wc -l` | PASS | 589 files contain backend-name terms. |
| `rg -n "seedBuiltInBackends|seedTargetBackendCatalog|repairBackendCapabilitiesV27" internal/server/db internal --glob '*.go'` | PASS | Found seed/repair calls and definitions in `internal/server/db/db.go`. |
| `git diff --stat` | PASS | Captured tracked diff stat. |
| `git diff --check` | PASS | No whitespace errors in tracked diffs. |

No build/test suite was required for this documentation/inventory-only checkpoint.

## 6. Evidence Files or Logs

- `validation-log.md`
- `open-issues.md`
- `commit-log.md`

## 7. Remaining Issues

| Issue | Blocker? | Notes |
| --- | --- | --- |
| Old structures remain in code/docs/scripts/configs. | No | Expected. They are deletion targets for Checkpoints B-E. |
| Backend-name hardcode candidates require classification. | No | Some occurrences are valid catalog/docs/tests; forbidden business logic must be removed by the relevant checkpoint. |
| Worktree has unrelated pre-existing modified/untracked files. | No | Explicit path staging is required. |

## 8. Next Phase

Checkpoint B: config registry / backend catalog loader + db.go seed hardcode removal.

Do not start Checkpoint B until Checkpoint A is committed and pushed.

## 9. git status --short raw output

```text
 M web/package-lock.json
 M web/package.json
?? .mimocode/plans/1782215119986-calm-planet.md
?? .mimocode/skills/
?? docs/design/catalog-configset-and-runtime-snapshot.md
?? docs/reports/codex-configset-refactor-20260625/
?? docs/reports/codex-project-wide-execution-plan-20260625-final-review/
?? docs/reports/codex-project-wide-execution-plan-20260625-review/
?? docs/reports/codex-project-wide-execution-plan-20260625/
?? docs/reports/codex-project-wide-review-20260625/
?? docs/reports/model-runtime-node-wizard/e2e-*/...
```
