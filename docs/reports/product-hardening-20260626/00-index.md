# LightAI Go Product Hardening 2026-06-26 — Executable Plan Package

Status: Claude review and development package  
Target repo: `/home/kzeng/projects/ai-platform-study/lightai-go`  
Output directory in repo: `docs/reports/product-hardening-20260626`

## What this package is

This is not a vision document. It is an executable repair and hardening package.

Claude must use it to:

1. inspect current source code;
2. produce a file-level implementation plan;
3. wait for approval if requested;
4. implement changes workstream by workstream;
5. validate each change with tests and evidence;
6. commit, push, and close out.

## Workstreams

| Workstream | Main deliverable | Main code areas |
| --- | --- | --- |
| A. Naming debt | consistent concept vocabulary and UI labels | `web/src/router`, `web/src/layouts`, `web/src/pages`, `web/src/locales`, docs, tests |
| B. Model deployment UI | guided deployment wizard with preview/preflight | `ModelDeploymentsPage.vue`, deployment API client, deployment handlers, RunPlan/preflight |
| C. Runtime parameter completeness | shared parameter editor across BackendRuntime/NBR/Deployment | backend catalog YAML, ConfigSet, parameter editor, resolver, RunPlan tests |
| D. OpenAI gateway/audit/metering | minimal tenant-scoped OpenAI-compatible gateway | API routes, auth/API keys, DB schema, audit logs, usage records, proxy client |
| E. Stability regression | current golden path and evidence | Go tests, frontend tests, scripts, Playwright/API smoke, runtime smoke |

## Non-negotiable project rules

- Do not create a new branch unless explicitly instructed.
- Do not preserve legacy config/template/API paths unless the current code contract requires them.
- Do not add compatibility fallback for old dirty DB data.
- If schema changes are breaking, document clean DB rebuild.
- Do not trust client-provided Docker/image evidence.
- Do not hide discovered problems as future work if they are fixable now.
- Every behavior change must have test/evidence.
- Every workstream must produce closeout evidence.
- Final `git status --short` must be clean or explicitly justified.

## Required Claude first action

Before code changes, Claude must create:

```text
docs/reports/product-hardening-20260626/execution/00-current-code-inventory.md
docs/reports/product-hardening-20260626/execution/01-file-level-implementation-plan.md
docs/reports/product-hardening-20260626/execution/02-risk-and-stop-conditions.md
```

These files must include exact source paths, exact functions/components to modify, exact tests to add/update, and expected outputs.

## Required validation commands

Minimum final validation:

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
git diff --check
git status --short
```

Additional runtime smoke is required where local Docker/GPU/model assets are available.

## Final output

Final closeout must include:

- commits;
- push result;
- validation command outputs;
- runtime smoke evidence;
- unresolved issues, only if externally blocked and documented;
- clean `git status --short`;
- paths to all evidence files.
