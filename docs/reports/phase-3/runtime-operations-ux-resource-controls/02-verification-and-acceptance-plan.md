# Runtime Operations UX & Resource Controls — Verification and Acceptance Plan

> Date: 2026-06-23  
> Status: Verification plan for implementation batch.

## 1. Purpose

This document defines how to verify the Runtime Operations UX & Resource Controls implementation. It is intended to prevent regressions and ensure the batch solves classes of problems rather than only the specific observed examples.

## 2. Verification matrix

| Area | Required verification | Default required |
|---|---|---|
| RunPlan lint | Go unit tests and dry-run command checks | Yes |
| Resource controls | Go unit tests and command preview checks | Yes |
| Shared GPU admission | Go unit tests | Yes |
| Runtime log classifier | Fixture tests | Yes |
| Instance auto-refresh | Frontend tests | Yes |
| JsonViewer | Frontend component tests | Yes |
| Config layout | Frontend smoke/component tests | Yes |
| Real Docker/GPU | Gated smoke only | No |
| Browser login E2E | Deferred unless separately approved | No |

## 3. RunPlan lint acceptance

### 3.1 Required tests

Test names can differ, but coverage must include:

- `TestRunPlanLintLlamaCppHostEnvCLIConflict`
- `TestRunPlanLintLlamaCppPortEnvCLIConflict`
- `TestRunPlanLintDuplicateCtxSize`
- `TestRunPlanLintDuplicateVLLMGpuMemoryUtilization`
- `TestRunPlanLintDuplicateSGLangMemFractionStatic`
- `TestRunPlanLintHighRiskContainerFlags`
- `TestRunPlanLintCleanVLLM`
- `TestRunPlanLintCleanSGLang`
- `TestRunPlanLintCleanLlamaCpp`

### 3.2 Acceptance criteria

- Conflict findings have stable IDs.
- Severity is deterministic.
- Findings include source information.
- Lint result is present in RunPlan preview/preflight response.
- Lint errors block start unless an approved override mechanism exists.

## 4. Resource controls acceptance

### 4.1 vLLM

Required:

- UI/API can express memory fraction.
- RunPlan maps memory fraction to `--gpu-memory-utilization`.
- Duplicate memory fraction arg is rejected by lint.
- Docker command preview shows the final value.

Tests:

- memory fraction 0.5 produces `--gpu-memory-utilization 0.5`
- invalid fraction below min rejected
- invalid fraction above max rejected
- duplicate flag rejected

### 4.2 SGLang

Required:

- UI/API can express memory fraction.
- RunPlan maps memory fraction to `--mem-fraction-static`.
- attention backend can be set in advanced config, default `auto` means no unnecessary arg unless the implementation chooses explicit value.
- Docker command preview shows the final value.

Tests:

- memory fraction 0.5 produces `--mem-fraction-static 0.5`
- duplicate flag rejected
- attention backend arg is generated only when expected

### 4.3 llama.cpp

Required:

- UI must not show fake memory fraction.
- Exposed controls must be backend-real controls:
  - GPU layers
  - ctx size
  - batch size
  - ubatch size
  - cache type k/v
  - split mode
  - main GPU
  - tensor split
- Host/port env and CLI conflict must be prevented or linted.

Tests:

- setting memory fraction for llama.cpp is rejected or ignored with explicit warning
- gpu layers maps to `--n-gpu-layers` or current canonical flag
- ctx size maps to `--ctx-size`
- `LLAMA_ARG_HOST` + `--host` is rejected

## 5. Shared GPU admission acceptance

### Required tests

- vLLM 0.5 + vLLM 0.5 on same GPU → ok
- vLLM 0.7 + vLLM 0.7 on same GPU → blocked
- vLLM 0.4 + SGLang 0.5 on same GPU → ok
- llama.cpp shared GPU with unknown budget → warning
- existing exclusive instance blocks new placement
- requested exclusive placement blocks when active instance exists
- oversubscribe override, if implemented, produces audit requirement

### Required behavior

- Docker GPU device binding alone is never described as memory isolation.
- Unknown budget is visible to user.
- Override cannot be silent.
- Admission result is visible in RunPlan preview.

## 6. Runtime log classifier acceptance

### Required fixtures

```text
sglang-torchao-syntax-warning.log
sglang-attention-backend-default.log
llamacpp-env-host-overwritten.log
cuda-oom.log
```

### Required classifications

| Fixture | Expected severity |
|---|---|
| SGLang torchao syntax warning | noise or advisory |
| SGLang attention backend default | advisory |
| llama.cpp host env overwritten | warning |
| CUDA OOM | error |
| fatal startup traceback | error or fatal |

### Required behavior

- `noise` and `advisory` do not mark instance failed.
- `warning` is visible in diagnostics.
- `error`/`fatal` are visible in diagnostics and can support failure analysis.
- Classification output includes rule ID, severity, message, suggestion, raw line, and occurrence count.

## 7. Instance auto-refresh acceptance

### Required tests

- transitional state triggers fast polling;
- stable state triggers slow polling;
- document hidden slows or pauses polling;
- route leave/unmount stops polling;
- API error produces stale-data warning and backoff;
- manual refresh triggers immediate fetch.

### Required UI behavior

- no manual browser refresh required to see status changes;
- page shows last refreshed time;
- page has manual refresh control;
- no console error;
- no i18n key leak.

## 8. JsonViewer acceptance

### Required tests

- long JSON renders inside constrained height;
- copy returns full content;
- fullscreen opens and shows content;
- download, if implemented, emits full JSON;
- malformed JSON falls back to raw display;
- long strings do not break layout.

### Required UI behavior

- diagnostic JSON is readable in page;
- full content can be copied;
- no horizontal page overflow;
- same component used across diagnostic locations.

## 9. Configuration layout acceptance

### Required tests

- ConfigEditorLayout renders top summary;
- common fields visible without opening raw JSON;
- advanced section collapses/expands;
- RunPlan preview panel renders command/lint/admission;
- generated diagnostic JSON is read-only by default;
- expert mode requires explicit enablement if raw editing remains.

### Required UI behavior

- runtime config page follows the standard layout;
- deployment/test related page uses standard preview/diagnostic components where applicable;
- health check config is structured;
- health result JSON and advanced diagnosis JSON are not shown as ordinary editable fields.

## 10. Full regression commands

Implementation closeout must report the exact commands used. Minimum expected commands:

```bash
go test ./...
go build ./...
gofmt -l internal/
node web/tests/modelCapabilities.test.mjs
cd web && npm run build
git diff --check
git status --short
```

If frontend tests are added under a project script, include:

```bash
cd web && npm test
# or
cd web && npm run test
# or exact project-specific test command
```

## 11. Evidence requirements

Closeout must include:

- test command output summary;
- new fixture paths;
- screenshots or trace references only if UI tests produce them;
- RunPlan lint sample output;
- resource admission sample output;
- classified log event sample output;
- before/after note for each known issue.

## 12. Failure handling

If a verification step fails:

1. Do not continue to commit.
2. Fix the issue or document it as an approved blocker.
3. Re-run the relevant phase test.
4. Re-run final regression.
5. Update closeout with accurate status.

## 13. Completion criteria

The batch is complete only when:

- all known issues are fixed or approved as documented blockers;
- all required tests pass;
- closeout is written;
- commit is created;
- push succeeds;
- final `git status --short` is empty.
