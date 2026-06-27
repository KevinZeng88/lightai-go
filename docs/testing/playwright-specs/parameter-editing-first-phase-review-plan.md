# Parameter Editing First Phase Review Plan

## 1. Objective

Create a stable review checkpoint before any further implementation.

This phase asks Codex to review the current code and test strategy. It should not write production code or Playwright specs yet.

## 2. Inputs

Codex must read:

```text
docs/reports/ui-automation-audit/lightai-code-review-and-gap-analysis.md
docs/testing/playwright-specs/parameter-editor-test-architecture.md
docs/testing/playwright-specs/parameter-editor-contract-spec.md
docs/testing/playwright-specs/parameter-editor-surfaces-matrix.md
docs/testing/playwright-specs/parameter-editor-codex-execution-plan.md
docs/testing/playwright-specs/runtime-config-parameter-enabled-persistence.md
docs/testing/playwright-specs/runtime-config-clone-name-persistence.md
docs/testing/playwright-specs/runtime-template-parameter-display.md
docs/testing/playwright-specs/parameter-editing-test-strategy-decision.md
```

Codex must inspect the code areas:

```text
web/src/components/config/
web/src/utils/configEditView.ts
web/src/pages/BackendRuntimesPage.vue
web/src/pages/RunnerConfigsPage.vue
web/src/pages/ModelDeploymentsPage.vue
web/src/pages/
web/tests/e2e/
internal/server/configedit/
internal/server/semanticconfig/
internal/server/catalog/
internal/server/runplan/
internal/server/*runtime*
internal/server/*deployment*
```

## 3. Review Questions

Codex should answer these questions.

### 3.1 Reuse boundary

```text
1. Which pages use ConfigEditView?
2. Which pages use ConfigField?
3. Which pages use configEditView.ts helpers?
4. Which pages still use custom parameter editing logic?
5. Is RuntimeParameterEditor still active in normal runtime/deployment flows, or only diagnostic/dev-only?
```

### 3.2 Backend semantics

```text
1. Where does configedit project schema + values into UI view?
2. Where does configedit apply a patch back to stored values?
3. Does enabled=true round-trip?
4. Does enabled=false round-trip?
5. Does disabled preserve value?
6. Does default value automatically enable optional fields?
7. Does missing enabled default to true anywhere?
8. Do clone/snapshot boundaries preserve source isolation?
```

### 3.3 Frontend semantics

```text
1. Does ConfigField emit both enabled and value?
2. Does ConfigEditView preserve value when disabled?
3. Does frontend patch construction include enabled=false explicitly?
4. Does page save logic reload fresh edit view after save?
5. Does RunnerConfigsPage behave differently from BackendRuntimesPage after save?
```

### 3.4 Testability

```text
1. Which shared components need data-testid?
2. Can one selector contract serve all ConfigEdit pages?
3. Which current Playwright tests are thin enough to keep?
4. Which planned Playwright tests should move to backend/unit tests?
```

### 3.5 Documentation consistency

```text
1. Does parameter-editing-test-strategy-decision.md contradict existing docs?
2. Which older docs should be marked as examples rather than source-of-truth?
3. Should parameter-editor-codex-execution-plan.md be amended before implementation?
```

## 4. Expected Codex Output

Codex should output a review report with this structure:

```text
1. Summary
2. Confirmed reuse map
3. Active editing components
4. Backend enabled/value round-trip findings
5. Frontend save/reload findings
6. Testability gaps
7. Documentation conflicts
8. Recommended amendments before implementation
9. Proposed first implementation batch
10. Commands run
11. git status --short
```

## 5. Allowed Actions in This Phase

Allowed:

```text
- Read files
- Search code
- Run existing tests if useful
- Produce a review report
- Suggest doc changes
```

Disallowed:

```text
- Implement Playwright specs
- Modify production code
- Modify tests
- Commit code
- Create a new branch
- Touch unrelated files
```

If Codex finds an urgent bug, it should document it with file path, behavior, and recommended fix. It should not fix it in this review phase.

## 6. Review Acceptance Criteria

The review is accepted when it clearly answers:

```text
- Whether ConfigEdit is the correct shared abstraction.
- Whether each surface reuses or bypasses ConfigEdit.
- Which layer should test each rule.
- Which docs need updates before execution.
- What the first implementation batch should include.
```

After acceptance, implementation can proceed with a focused prompt based on the reviewed plan.
