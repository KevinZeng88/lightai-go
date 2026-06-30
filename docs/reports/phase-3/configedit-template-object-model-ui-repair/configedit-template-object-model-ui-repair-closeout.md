# ConfigEdit Template Object Model UI Repair Closeout

Date: 2026-07-01

## Root Cause

The ConfigEdit Templates page was bound only to explicit component template YAML files. It did not list ConfigEdit templates materialized from the catalog/runtime object model, so the page could be empty or incomplete even though runtime parameters existed as structured ConfigEdit fields.

The UI also had hardcoded English copy for template inspection and display levels, and ConfigEdit field ordering was page-neutral section ordering rather than the requested product grouping. Enabled placement used live field state, which could make rows jump while editing if implemented per toggle.

## Implementation Summary

- Added materialized ConfigEdit template registry data to `/api/v1/config-edit/templates` and `/api/v1/config-edit/templates/{id}`.
- Extended the template DTO with top-level `fields` carrying identity, tier, section, view, source, type/widget, order, risk, enabled state, and effects.
- Generated materialized templates from `catalog.MaterializeBackendRuntime` plus `configedit.ProjectConfigSetToEditView`.
- Localized ConfigEdit Templates page and side navigation.
- Replaced `Normal / Advanced / Developer` hardcoded options with shared `常用 / 高级 / 专家` display-level options and help text.
- Added shared ConfigEdit grouping and ordering: enabled, common, advanced, expert; expert/security/raw fields stay expert even when enabled.
- Implemented load-time enabled placement via `original_enabled`, so editing toggles do not reorder fields until save/reload.
- Updated tests for API materialized templates, i18n/hardcoded string checks, grouping, hierarchical display semantics, raw/expert handling, and enabled placement stability.

## Changed Files

- `internal/server/api/configedit_template_handlers.go`
- `internal/server/api/configedit_template_handlers_test.go`
- `internal/server/configedit/templates.go`
- `web/src/components/config/ConfigSection.vue`
- `web/src/components/config/__tests__/ConfigEditView.render.test.ts`
- `web/src/layouts/ConsoleLayout.vue`
- `web/src/locales/en-US.ts`
- `web/src/locales/zh-CN.ts`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/pages/ConfigEditTemplatesPage.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- `web/src/utils/configEditDisplay.ts`
- `web/src/utils/configEditView.ts`
- `web/src/utils/__tests__/configEditView.test.ts`
- `web/tests/runtimeBoundaryUi.test.mjs`
- `docs/reports/phase-3/configedit-template-object-model-ui-repair/*.md`

## Verification Commands and Results

- `go test ./internal/server/api -run TestConfigEditTemplatesListIncludesMaterializedRegistryData -v`: PASS
- `go test ./...`: PASS
- `cd web && node tests/runtimeBoundaryUi.test.mjs`: PASS
- `cd web && npm run test:unit`: PASS, 14 files / 80 tests
- `cd web && npm test`: PASS
- `cd web && npm run build`: PASS

Build warning observed: Vite reported existing chunk-size and Rollup annotation warnings from dependencies. The build completed successfully; no functional failure was produced.

## Evidence Paths

- API materialized template test: `internal/server/api/configedit_template_handlers_test.go`
- Shared grouping/ordering test: `web/src/utils/__tests__/configEditView.test.ts`
- ConfigEdit render test: `web/src/components/config/__tests__/ConfigEditView.render.test.ts`
- Static UI/i18n guard: `web/tests/runtimeBoundaryUi.test.mjs`
- Phase docs:
  - `phase-0-baseline-findings.md`
  - `phase-1-object-model-contract.md`
  - `phase-2-template-list-data-source.md`
  - `phase-3-i18n-and-copy.md`
  - `phase-4-view-level-semantics.md`
  - `phase-5-grouping-and-ordering.md`
  - `phase-6-enabled-placement.md`

## API Evidence

The API test decodes `HandleListConfigEditTemplates` response and verifies:

- `catalog_materialized` templates are present.
- Materialized backends include `vllm`, `sglang`, and `llamacpp`.
- Materialized fields include `model_runtime.*` backend arguments.
- `docker.privileged` is structured under `security_high_risk` with `tier=expert` and `risk=high`.

Manual equivalent:

```bash
curl -s http://127.0.0.1:18080/api/v1/config-edit/templates
```

## Problem Closure

- Unresolved problems remain: no.
- Problems existing only in chat: no.
- Formal open-issues document needed: no.
- Final problem states:
  - ConfigEdit Templates empty/incomplete data source: FIXED.
  - zh-CN hardcoded English strings in targeted ConfigEdit surfaces: FIXED.
  - Ambiguous display-level labels: FIXED.
  - Non-shared parameter grouping/sorting: FIXED.
  - Enabled toggle live reorder risk: FIXED.
  - Raw/diagnostic fields entering enabled group: FIXED.

Final status: PASS.

## Commit and Push

- Commit id: recorded in the final execution report after commit.
- Push result: recorded in the final execution report after push.
- Final `git status --short`: recorded in the final execution report after push.

