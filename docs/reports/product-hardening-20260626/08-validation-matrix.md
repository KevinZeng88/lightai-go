# 08 — Validation Matrix

## Workstream A — Naming

| Check | Command/evidence | Pass criteria |
| --- | --- | --- |
| naming inventory | `rg ... > naming-rg.txt` | all user-facing terms classified |
| dictionary | `docs/engineering/naming-dictionary.md` | all concepts defined |
| i18n | `cd web && npm test` | no missing/stale keys |
| build | `cd web && npm run build` | pass |

## Workstream B — Deployment UI

| Check | Command/evidence | Pass criteria |
| --- | --- | --- |
| preview API | API test | unsaved preview returns RunPlan/lint/preflight |
| UI payload | frontend test | uses `node_backend_runtime_id` |
| non-ready NBR | API + UI test | blocked |
| ready_with_warnings | API + UI test | allowed with warning |
| RunPlan preview | browser smoke | Docker command visible before start |
| start | E2E | cannot start with blocking errors |

## Workstream C — Runtime parameters

| Check | Command/evidence | Pass criteria |
| --- | --- | --- |
| vLLM params | editor + RunPlan test | enabled values appear in args |
| SGLang params | editor + RunPlan test | enabled values appear in args |
| llama.cpp params | editor + RunPlan test | no fake memory fraction |
| disabled optional | Go + frontend test | value retained, not applied |
| required locked | frontend test | cannot disable |
| model cleanup | frontend test | model page has no backend args |

## Workstream D — Gateway/audit/metering

| Check | Command/evidence | Pass criteria |
| --- | --- | --- |
| API key creation | API test | key shown once, hash stored |
| invalid key | API test | rejected |
| cross tenant | API test | rejected |
| `/v1/models` | API E2E | returns allowed models |
| `/v1/chat/completions` | API E2E | proxied |
| usage record | DB/API test | success/failure recorded |
| audit log | DB/API test | success/failure recorded |
| redaction | test/log review | no full key in output |

## Workstream E — Regression

| Check | Command/evidence | Pass criteria |
| --- | --- | --- |
| Go tests | `go test ./...` | pass |
| server build | `go build ./cmd/server/...` | pass |
| agent build | `go build ./cmd/agent/...` | pass |
| frontend tests | `cd web && npm test` | pass |
| frontend build | `cd web && npm run build` | pass |
| diff hygiene | `git diff --check` | pass |
| API E2E | evidence directory | pass or honest external block |
| browser smoke | evidence directory | pass |
| runtime smoke | matrix | vLLM/SGLang/llama.cpp pass or honest classified block |
| git | `git status --short` | clean |

## Final closeout pass criteria

Final closeout is accepted only if:

- all completed workstreams have evidence;
- all mandatory tests pass;
- runtime smoke is current;
- OpenAPI/docs updated;
- commits pushed;
- final status is clean.
