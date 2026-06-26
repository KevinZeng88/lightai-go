# 09 — Claude Plan Review Prompt

Use this prompt first. It asks Claude to inspect source and produce an implementation plan without changing code.

```text
You are working in:

/home/kzeng/projects/ai-platform-study/lightai-go

Do not create a new branch.
Do not modify code yet.
Do not use docs/reports/phase-3.

Read this package:

docs/reports/product-hardening-20260626/00-index.md
docs/reports/product-hardening-20260626/01-current-code-findings.md
docs/reports/product-hardening-20260626/02-file-level-implementation-plan-template.md
docs/reports/product-hardening-20260626/03-workstream-a-naming-debt.md
docs/reports/product-hardening-20260626/04-workstream-b-model-deployment-ui.md
docs/reports/product-hardening-20260626/05-workstream-c-runtime-parameter-completeness.md
docs/reports/product-hardening-20260626/06-workstream-d-openai-gateway-audit-metering.md
docs/reports/product-hardening-20260626/07-workstream-e-stability-regression.md
docs/reports/product-hardening-20260626/08-validation-matrix.md

Then inspect current source code with rg/find/go test/npm test.

Create these files only:

docs/reports/product-hardening-20260626/execution/00-current-code-inventory.md
docs/reports/product-hardening-20260626/execution/01-file-level-implementation-plan.md
docs/reports/product-hardening-20260626/execution/02-risk-and-stop-conditions.md

The implementation plan must be concrete:

- exact files to modify;
- exact components/functions/handlers to change;
- API routes to add/change/remove;
- DB schema changes, if any;
- UI behavior changes;
- tests to add/update;
- validation commands;
- evidence paths;
- commit plan.

Do not write generic goals.
Do not say "improve UX" without saying which file/component and what behavior changes.
Do not defer fixable issues as future work.
Do not preserve legacy dirty config or old DB compatibility.

After creating the plan, stop and report:
- plan file paths;
- top 10 concrete code changes;
- DB/API impact;
- validation commands;
- risks needing approval.
```
