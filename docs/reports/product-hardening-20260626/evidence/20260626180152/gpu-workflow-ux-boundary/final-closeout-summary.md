# GPU Workflow UX Boundary Repair — Final Closeout Summary

Timestamp: 2026-06-26 18:39 UTC | Evidence dir: `20260626180152/gpu-workflow-ux-boundary/`

## Commit List (12 commits)

```
3cd3be0  docs: add gpu workflow ux boundary design package (8 files, 1619 lines)
6869c1e  docs: confirm gpu workflow ux boundary understanding
e95f54b  docs: audit gpu workflow ux boundaries
9ce7ced  fix: reset runtime config wizard and add config naming (NodeSelectorTable, wizard reset, config name)
9383031  fix: simplify runtime template presentation (runtimeDisplay.ts, advanced diagnostics)
a535774  fix: add human runtime parameter form (HumanRuntimeParameterForm, viewModel adapter)
717cc04  fix: align model library node selector and deployment compatibility ux
3812bac  test: add gpu workflow ux regression evidence
ee00c25  fix: close gpu workflow ux boundary blockers (5 blockers: config name, compatibility, shm_size, system editor, tests)
20b7bcf  fix: derive deployment model locations from artifacts (no /model-locations API needed)
2c27bb4  fix: enforce deployment create model location compatibility
e9e71df  fix: add model location fixtures to deployment create tests
```

## Tests Run

| Suite | Result |
|---|---|
| `go test ./...` | ALL PASS (18 packages) |
| `go build ./cmd/server/...` | PASS |
| `go build ./cmd/agent/...` | PASS |
| `cd web && npm test` | ALL PASS (37 tests, 991 i18n keys) |
| `cd web && npm run build` | PASS (3.42s) |
| `git diff --check` | PASS |

## New Go Tests

- `TestCreateDeploymentRejectsModelLocationMissing` — 400 on missing model location ✅
- `TestCreateDeploymentAcceptsWithModelLocation` — 201 when location present ✅

## Accepted Blockers (All Fixed)

1. Default config name not saved → `form.display_name || defaultConfigName.value`
2. DeploymentWizard no node compatibility → `checkNodeCompatibility()`
3. shm_size mapped to wrong key → flat `shm_size`, merged into `launcher.docker_options`
4. RuntimeParameterEditor on system templates → moved to Advanced Diagnostics
5. `/model-locations` not found → derived from artifact `.locations`
6. `HandleCreateDeployment` no model_location check → added before INSERT

## Remaining Issues: NONE

## Final Git Status: CLEAN

## Confirmed: No Gateway/API Key/Usage/Billing code in any commit
