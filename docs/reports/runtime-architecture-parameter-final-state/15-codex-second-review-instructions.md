# Codex Second Review Instructions

## 1. Task

Run a second documentation review after the ConfigSetBundle revision. This is review-only. Do not modify functional code.

## 2. Working directory

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
```

## 3. Review target

```text
docs/reports/runtime-architecture-parameter-final-state/
```

## 4. Required reading

Read `00-index.md`, `13-codex-review.md`, `14-codex-review-fix-plan.md`, `03-final-runtime-domain-contract.md`, `04-final-parameter-contract.md`, `04a-parameter-ownership-and-layered-presentation-contract.md`, `05a-configset-bundle-composition-and-presentation-contract.md`, `06-runplan-and-preflight-contract.md`, `07-ui-and-api-contract.md`, `08-api-first-e2e-and-automation-requirements.md`, `09-implementation-plan.md`, `10-claude-execution-prompt.md`, `11-final-closeout-template.md`, `manifest.json`.

## 5. Review questions

Answer whether the revised docs close all seven issues from `13-codex-review.md`; whether ConfigSet is now clearly defined as final-domain concept; whether ConfigSetBundle is sufficiently defined; whether ConfigItem field tiers are implementable; whether copy-on-create with copied schema snapshot and readonly schema is clear; whether removal of `overridable_at` is safe and replaced by schema.read_only/state.editable; whether self-describing/self-presenting ConfigSet presentation is clear; whether child_slots and GenericConfigSetRenderer are sufficiently specified; whether RunPlan shared builder, parameter_source_map, Docker subfield rules, and no-compatibility policy are clear; whether Claude execution can proceed batch-by-batch after user approval.

## 6. Output

Generate:

```text
docs/reports/runtime-architecture-parameter-final-state/16-codex-second-review.md
```

Allowed updates: `00-index.md` and `manifest.json` only to register `16-codex-second-review.md`.

## 7. Review document structure

```markdown
# Codex Second Review — ConfigSetBundle Final-State Docs

## 1. Review Scope
## 2. Overall Verdict
ACCEPT / ACCEPT_WITH_FIXES / REJECT
## 3. Executive Summary
## 4. Closure of First Review Issues
## 5. ConfigSetBundle Model Review
## 6. ConfigItem Field-Tier Review
## 7. Presentation Contract Review
## 8. RunPlan / Source Map Review
## 9. No-Compatibility Policy Review
## 10. Remaining Required Fixes
## 11. Claude Execution Recommendation
## 12. Final Status
```

## 8. Commit and push

```bash
git status --short
git diff -- docs/reports/runtime-architecture-parameter-final-state
git add docs/reports/runtime-architecture-parameter-final-state
git commit -m "docs: add second codex review for configset bundle plan"
git push
git status --short
```

## 9. Final terminal output

```text
CODEX_SECOND_REVIEW_COMPLETED

1. Review verdict:
2. Review document:
3. Files changed:
4. Commit id:
5. Push result:
6. git status --short:
7. Most important findings:
8. Recommendation:
```

Do not start functional implementation.
