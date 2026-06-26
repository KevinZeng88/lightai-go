# 07 — Workstream E: Project-Wide Stability Regression

## Goal

Create a current golden regression baseline after the product-hardening changes.

## Step E1 — Baseline before changes

Run before implementation:

```bash
git rev-parse --short HEAD
git status --short
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
git diff --check
```

Store output:

```text
docs/reports/product-hardening-20260626/evidence/<timestamp>/baseline/
```

## Step E2 — Script inventory

Run:

```bash
find scripts -maxdepth 4 -type f | sort
find docs/reports -maxdepth 5 -type f | sort
rg -n "HISTORICAL|e2e|smoke|runtime smoke|final-runtime-smoke|vLLM|SGLang|llama.cpp|BLOCKED|PASS|FAIL" scripts docs
```

Create:

```text
docs/reports/product-hardening-20260626/execution/test-and-evidence-inventory.md
```

Classify scripts:

| Script | Current | Purpose | Keep/repair/archive | Reason |
| --- | --- | --- | --- | --- |

## Step E3 — API-first E2E

Required chain:

1. login;
2. CSRF;
3. list nodes;
4. list models;
5. list runtime templates;
6. list node runtime configs;
7. check/probe NBR;
8. create deployment preview;
9. create deployment;
10. dry-run;
11. start;
12. poll instance;
13. fetch logs;
14. test model endpoint;
15. stop;
16. verify stopped/removed according to current contract;
17. verify audit logs;
18. if gateway implemented, test `/v1/models` and `/v1/chat/completions`.

Output:

```text
docs/reports/product-hardening-20260626/evidence/<timestamp>/api-e2e/
```

## Step E4 — Browser smoke

Use Playwright if available.

Required scenario:

1. login;
2. open model page;
3. verify model facts do not show runtime args;
4. open runtime template page;
5. verify naming and parameter editor;
6. open node runtime config page;
7. verify check/probe/status;
8. open deployment page;
9. create deployment through wizard;
10. preview RunPlan;
11. save/start;
12. open instances;
13. see auto-refresh;
14. open logs;
15. stop deployment;
16. verify final state.

Output:

```text
docs/reports/product-hardening-20260626/evidence/<timestamp>/browser-smoke/
```

## Step E5 — Runtime smoke

Where local assets exist, run:

- vLLM with HF model;
- SGLang with HF model;
- llama.cpp with GGUF model.

Required for each backend:

- NBR check/probe;
- deployment preview;
- start;
- `/v1/models`;
- chat/completion test according to model capability;
- logs;
- stop;
- Docker ps final state;
- RunPlan JSON;
- equivalent Docker command;
- inspect summary.

If a backend is blocked:

- classify as external dependency, catalog/config bug, code bug, or environment missing;
- include exact error;
- fix if code/config bug;
- document if external.

Special check:

- Re-test SGLang capability support because previous evidence reported a capability blocker, while current catalog should be verified.

## Step E6 — Regression tests

Minimum final:

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
git diff --check
```

Additional targeted:

```bash
go test ./internal/server/api/... -run 'Deployment|Preview|Preflight|Gateway|APIKey|Usage|Audit'
go test ./internal/server/runplan/...
```

## Step E7 — Final evidence report

Create:

```text
docs/reports/product-hardening-20260626/execution/final-regression-report.md
```

Required sections:

- commit range;
- test commands;
- API E2E result;
- browser smoke result;
- runtime smoke matrix;
- known skips/blocks;
- fixed regressions;
- unresolved externally blocked items;
- final git status.

## Acceptance

- All mandatory tests pass.
- Current evidence exists and is not confused with historical evidence.
- vLLM/SGLang/llama.cpp are honestly classified.
- Gateway is covered if implemented.
- Final closeout is complete.
