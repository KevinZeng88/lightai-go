# 02 — File-Level Implementation Plan Template

Claude must create a filled version before coding:

```text
docs/reports/product-hardening-20260626/execution/01-file-level-implementation-plan.md
```

Use this exact structure.

## 1. Baseline

```bash
git rev-parse --short HEAD
git status --short
go test ./...
cd web && npm test
```

Record output summary and failures.

## 2. Files to inspect

List exact files and why.

Example:

| File | Reason | Expected action |
| --- | --- | --- |
| `web/src/pages/ModelDeploymentsPage.vue` | deployment create UX | replace thin dialog with wizard/sections |
| `web/src/pages/RunnerConfigsPage.vue` | NBR UI | rename labels, add parameter editor, probe/preflight status |
| `internal/server/api/router.go` | route contract | add gateway routes, verify deployment preview route |
| `internal/server/db/db.go` | schema | add API keys/usage if needed |
| `docs/api/openapi.yaml` | API contract | update gateway/deployment preview schemas |

## 3. Planned code changes

For each workstream:

### Workstream A

- Files:
- Functions/components:
- Data/API contract changes:
- Tests:
- Validation:

### Workstream B

- Files:
- Functions/components:
- Data/API contract changes:
- Tests:
- Validation:

### Workstream C

- Files:
- Functions/components:
- Data/API contract changes:
- Tests:
- Validation:

### Workstream D

- Files:
- Functions/components:
- Data/API contract changes:
- Tests:
- Validation:

### Workstream E

- Files:
- Functions/components:
- Data/API contract changes:
- Tests:
- Validation:

## 4. DB/schema impact

State one:

- no DB change;
- clean fresh DB schema change only;
- data-preserving migration required and why.

If DB changes are breaking, document:

```bash
rm -f /tmp/lightai/data/lightai.db
```

or current equivalent rebuild command.

## 5. API contract impact

List every added/changed/removed route:

| Method | Path | Change | Tests |
| --- | --- | --- | --- |

## 6. UI contract impact

List every changed page/component:

| Page/component | Change | Tests |
| --- | --- | --- |

## 7. Stop conditions

Stop and report if:

- a required route does not exist and adding it affects major architecture;
- gateway requires a routing decision not covered by this package;
- current tests fail before any change due unrelated breakage;
- Docker/GPU runtime smoke is externally blocked.

Do not stop for ordinary fixable compile/test failures.

## 8. Approval checkpoint

After this plan is created, wait for approval unless the user explicitly instructed AUTORUN.
