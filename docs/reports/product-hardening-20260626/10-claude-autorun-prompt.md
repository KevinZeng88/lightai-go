# 10 — Claude AUTORUN Prompt

Use after approving Claude's plan.

```text
You are working in:

/home/kzeng/projects/ai-platform-study/lightai-go

Execute the approved product-hardening plan under:

docs/reports/product-hardening-20260626/

Do not create a new branch.
Do not use docs/reports/phase-3.
Work on the current branch.
Commit and push after validated batches.

Execution order:

1. Workstream A — naming debt
2. Workstream B — model deployment UI
3. Workstream C — runtime parameter completeness
4. Workstream D — OpenAI-compatible gateway/audit/metering
5. Workstream E — full regression

For each workstream:

- inspect current code before editing;
- make the smallest clean implementation that satisfies the contract;
- remove replaced legacy/stale logic;
- update tests;
- update docs/OpenAPI if contract changes;
- run targeted validation;
- write closeout under:
  docs/reports/product-hardening-20260626/execution/workstream-<letter>-closeout.md
- commit with a clear message;
- push.

Required final validation:

go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
git diff --check
git status --short

Required final closeout:

docs/reports/product-hardening-20260626/execution/final-closeout.md

Final closeout must include:

- final commit list;
- push result;
- validation outputs;
- API E2E evidence;
- browser smoke evidence;
- runtime smoke evidence for vLLM/SGLang/llama.cpp or honest classified blocks;
- DB rebuild note if schema changed;
- OpenAPI/docs update summary;
- unresolved issues only if externally blocked;
- clean git status.

Do not stop for ordinary test failures; fix them.
Stop only if a decision is required that changes product scope or safety boundary.
```
