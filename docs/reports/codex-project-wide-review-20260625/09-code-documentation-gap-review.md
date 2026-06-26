# Code and Documentation Gap Review

## Current docs situation

`docs/README.md` and `docs/CURRENT.md` accurately describe many current Phase 4 concepts: model roots, NBR, RunPlan preview, Docker start, logs, stop, cleanup, and snapshot boundaries.

However, documentation is split across current docs, historical docs, phase reports, archived reports, and live code. Future agents can easily pick stale evidence.

## Major gaps

| Gap | Evidence | Recommendation |
| --- | --- | --- |
| OpenAPI is stale. | `docs/api/openapi.yaml` still documents `/runtime-environments`, `/run-templates`, `/model-deployments`; no current NBR/deployments/runplan/model-root API. | Regenerate or rewrite OpenAPI from current router and tests. |
| Scripts contradict current design. | Current design says no deployment `backend_runtime_id`; scripts still send it. | Repair or archive stale scripts. |
| Closeout reports include old payloads. | Recent evidence directories still store `parameters_json` and older `backend_runtime_id` payloads. | Mark historical evidence by date/contract version. |
| Docs say no old compatibility; code has compatibility/fallback paths. | `runtime_handlers.go`, `db.go`, `cmd/server/main.go`. | Either remove paths or explicitly document why each remains. |
| Open issues are spread. | Current formal issues live in backend-runtime-runplan, model-runtime-node-wizard, documentation-governance; new issues here are not part of those. | Use this review's `10-risk-register.md` as the current risk register for this audit. |

## Documentation acceptance standard

For the next closeout, require:

- Current API contract doc includes exact request/response fields.
- Every E2E script has a contract version note.
- Archived evidence cannot be cited as current unless re-run.
- Any remaining compatibility path is either removed or listed with owner and removal criteria.
