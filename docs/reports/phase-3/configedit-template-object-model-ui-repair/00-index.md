# ConfigEdit Template Object Model Repair Pack — Index

## Purpose

This pack is for a Codex execution pass in the LightAI Go repository. It explains the intended ConfigEdit template object model, records the current symptoms, and turns the requested fixes into phased implementation and validation steps.

The key point: **ConfigEdit Templates is part of the ConfigEdit template object model. It must not be treated as a disposable debug page or a simple i18n-only issue.** It should expose and validate the parameter template registry used by runtime templates, node backend runtimes, deployments, and other ConfigEdit-based editors.

## Suggested target directory in the repository

Copy these files to:

```bash
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/phase-3/configedit-template-object-model-ui-repair/
```

## Document order

1. `01-object-model-explainer.md` — explains the ConfigEdit template object model and expected invariants.
2. `02-current-issues-and-scope.md` — records the observed issues and repair scope.
3. `03-executable-repair-plan.md` — phased implementation plan for Codex.
4. `04-acceptance-and-tests.md` — required validation, tests, and evidence.
5. `05-codex-autonomous-execution-prompt.md` — final self-contained prompt for Codex.

## Execution policy

- Work on the current branch unless the user explicitly says otherwise.
- Do not add legacy compatibility branches for old snapshots or old data.
- Do not hide a failing feature to pass acceptance. Fix the object model or its presentation path.
- Do not downgrade structured parameters into raw JSON-only display except in the expert/diagnostic view.
- Do not implement sorting in one page only. Sorting must be reusable across ConfigEdit consumers.
- Finish with a commit, push, clean git status, and a closeout note containing root cause, changed files, tests, and evidence.
