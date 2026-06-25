# Checkpoint E: API/UI ConfigSet Contract

## Scope

Checkpoint E migrated public API, OpenAPI, Web pages/tests, and active scripts to the ConfigSet/current deployment contract. The checkpoint did not introduce legacy API fallback, old payload compatibility, or DB dual authority.

## Changed Areas

| Area | Result |
| --- | --- |
| Public deployment API | Responses expose `config_set`, `config_overrides`, `source_metadata`, and source NBR fields. DB column names such as `config_set_json` are not public response fields. |
| Deployment create/update | Create uses `node_backend_runtime_id` plus optional `config_overrides`. Old deployment payload keys are rejected with a clear ConfigSet contract error. |
| NBR enable/check | Enable and patch reject client-provided readiness evidence such as `image_present` and `docker_available`; readiness is established by check-request/probe. |
| OpenAPI | `docs/api/openapi.yaml` reflects current `/deployments`, `/deployments/preflight`, NBR enable, and NBR check-request routes. Old `/model-deployments`, `/runtime-environments`, and `/run-templates` paths are absent. |
| Web | Runtime catalog, BackendRuntime, RunnerConfig, Deployment, and RuntimeParameterEditor views use ConfigSet/current contract fields. Deployment edit no longer shows a non-effective runtime selector. |
| Scripts | Active smoke/E2E helpers use `node_backend_runtime_id` and `config_overrides`; stale legacy-contract scripts are archived or removed from active paths. |

## Deleted Or Archived Active Legacy Structures

| Structure | Action |
| --- | --- |
| Legacy deployment payload fields in active scripts | Replaced with `config_overrides` or archived under `scripts/archive/legacy-contract`. |
| Client-trusted readiness evidence in active scripts | Removed from enable/check bodies. |
| Old deployment routes in active scripts/OpenAPI | Replaced with current `/deployments` routes. |
| Deployment edit runtime selector | Removed from Web deployment page; source runtime/NBR information is displayed read-only. |

## Validation

| Command | Result | Summary |
| --- | --- | --- |
| `go test ./internal/server/catalog ./internal/server/api ./internal/server/runplan ./internal/agent/runtime -count=1` | PASS | Targeted backend contract tests pass. |
| `go test ./...` | PASS | All Go packages pass. |
| `go build ./cmd/server/...` | PASS | Server builds. |
| `go build ./cmd/agent/...` | PASS | Agent builds. |
| `cd web && npm test` | PASS | Web static/unit tests pass, including ConfigSet UI boundary assertions. |
| `cd web && npm run build` | PASS | Web production build passes. |
| OpenAPI YAML parse/path check | PASS | Current deployment paths exist; old routes are absent. |
| Active stale gate | PASS | No old authority fields, old deployment routes, or client-trusted readiness terms remain in active Web/OpenAPI/script paths. |
| `git diff --check` | PASS | No whitespace errors. |

## Issue Closure

All Checkpoint E issues found during validation are recorded in `open-issues.md` with `FIXED` status. No undocumented Checkpoint E blockers remain.

## Next Checkpoint

Checkpoint F: full validation, fresh DB, and platform-chain runtime smoke for vLLM, SGLang, and llama.cpp.
