# Phase 0 - Baseline

Date: 2026-07-01

## Commands Run

```bash
git status --short
git log --oneline -8
sed -n '1,320p' web/src/utils/configEditView.ts
sed -n '1,220p' web/src/utils/__tests__/configEditView.test.ts
sed -n '1,340p' web/src/components/config/__tests__/ConfigEditView.render.test.ts
sed -n '1,260p' web/tests/configEditContract.test.mjs
sed -n '1,260p' web/tests/runtimeBoundaryUi.test.mjs
sed -n '1,160p' web/package.json
sed -n '1,120p' web/vitest.config.ts
```

## Current Git State

`docs/reports/phase-3/configedit-regression-engineering/` is present as untracked prompt/design input. No tracked source changes existed before this implementation pass.

Latest commits:

- `3e1fc4a fix(configedit): refresh deployment edit view on level change`
- `ae74086 fix(configedit): repair template registry ui and parameter grouping`
- `d210763 docs(configedit): close unified parameter handling fix`

## Current Display Group Logic

`web/src/utils/configEditView.ts` currently checks `isExpertField(field)` before `enabledAtLoad`, which sends enabled high-risk/security/raw/diagnostic fields to `expert`. This conflicts with the product contract that all load-time enabled fields must be visible in `enabled_parameters`.

## Current Utility Test Gap

`web/src/utils/__tests__/configEditView.test.ts` currently contains the incorrect assertion:

`keeps enabled high-risk and diagnostic fields in the expert group`

The suite has basic group ordering, field sorting, and edit-session stability coverage, but it does not cover the full enabled/disabled matrix, patch behavior, or required/readonly edge cases in Vitest.

## Current ConfigEditView Render Coverage

`web/src/components/config/__tests__/ConfigEditView.render.test.ts` already covers:

- display groups render
- structured Docker fields do not show parent object JSON
- mount, health, env widgets remain structured
- enabled group expands
- expert group collapses
- readonly/editable mode basics
- self-contained field metadata rendering

Missing coverage:

- enabled high-risk/security/raw/diagnostic fields must render in `enabled_parameters`
- disabled high-risk/security/raw/diagnostic fields must render in `expert_parameters_group`
- zh-CN display group labels with real zh-CN i18n messages
- enabled group expanded and expert group collapsed for the complete risk matrix

## Current Static Contract Coverage

`web/tests/configEditContract.test.mjs` covers basic `buildConfigEditPatch` behavior and stable selectors for ConfigEditView/ConfigSection/ConfigField. It does not check displayGroup priority order, consumer page level reload behavior, or ConfigEdit-specific active-page boundaries.

`web/tests/runtimeBoundaryUi.test.mjs` contains broad ConfigEdit checks, but it is overloaded with runtime, deployment, wizard, probe, i18n, and display checks. ConfigEdit needs a dedicated boundary test.

## Test Changes Needed

- Fix `displayGroupForField` priority.
- Replace the wrong enabled high-risk expert assertion with enabled-first matrix tests.
- Add ConfigEditView render matrix tests.
- Add ConfigField enabled/value state tests.
- Add `web/tests/configEditRegressionBoundary.test.mjs` to `npm test`.
- Add risk metadata/badge visibility to `ConfigField`.

