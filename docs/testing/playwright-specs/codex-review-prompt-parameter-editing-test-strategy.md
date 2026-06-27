# Codex Review Prompt: Parameter Editing Test Strategy

Use this prompt to ask Codex for a review-only pass before implementation.

```text
You are reviewing the LightAI Go parameter editing test strategy before any further implementation.

Do not modify code.
Do not implement Playwright specs.
Do not commit.
Do not create a branch.
Do not touch unrelated files.

Please read these documents:

- docs/reports/ui-automation-audit/lightai-code-review-and-gap-analysis.md
- docs/testing/playwright-specs/parameter-editor-test-architecture.md
- docs/testing/playwright-specs/parameter-editor-contract-spec.md
- docs/testing/playwright-specs/parameter-editor-surfaces-matrix.md
- docs/testing/playwright-specs/parameter-editor-codex-execution-plan.md
- docs/testing/playwright-specs/runtime-config-parameter-enabled-persistence.md
- docs/testing/playwright-specs/runtime-config-clone-name-persistence.md
- docs/testing/playwright-specs/runtime-template-parameter-display.md
- docs/testing/playwright-specs/parameter-editing-test-strategy-decision.md
- docs/testing/playwright-specs/parameter-editing-first-phase-review-plan.md

Then inspect the current code, especially:

- web/src/components/config/
- web/src/utils/configEditView.ts
- web/src/pages/BackendRuntimesPage.vue
- web/src/pages/RunnerConfigsPage.vue
- web/src/pages/ModelDeploymentsPage.vue
- web/tests/e2e/
- internal/server/configedit/
- internal/server/semanticconfig/
- internal/server/catalog/
- internal/server/runplan/
- internal/server runtime/deployment handlers and repositories

Review objective:

Confirm whether the proposed strategy is correct:

1. Core enabled/value/default/clone rules should be tested mainly by Go/API and Vitest.
2. Playwright should only provide thin representative surface tests.
3. We should not write one full Playwright suite for every page.
4. Shared ConfigEdit components should provide stable selectors for all surfaces.
5. Existing runtime-config specific specs should be treated as examples/acceptance references, not duplicated per page.

Please answer:

1. Which pages currently use ConfigEditView / ConfigField?
2. Which pages use configEditView.ts or related helpers?
3. Which pages still have custom parameter editing logic?
4. Is RuntimeParameterEditor still active in normal flows, or legacy/diagnostic only?
5. Where are parameter_schema_json and parameter_values_json projected, applied, saved, and reloaded?
6. Does enabled=true round-trip today?
7. Does enabled=false round-trip today?
8. Does disabling preserve value today?
9. Does default value automatically enable optional fields anywhere?
10. Does missing enabled default to true anywhere?
11. Does clone/snapshot isolation hold for runtime configs and deployment snapshots?
12. Does RunnerConfigsPage reload the current edit view after save?
13. Which components need data-testid for future Playwright stability?
14. Which existing docs conflict with the new strategy?
15. What amendments should be made before implementation?

Expected output format:

## Summary

## Confirmed Reuse Map

## Active Editing Components

## Backend Semantics Findings

## Frontend Save/Reload Findings

## Testability Gaps

## Documentation Conflicts

## Recommended Amendments

## Proposed First Implementation Batch

## Commands Run

## git status --short

Important constraints:

- Review only.
- No code changes.
- No Playwright implementation in this pass.
- No branch creation.
- No unrelated file changes.
```
