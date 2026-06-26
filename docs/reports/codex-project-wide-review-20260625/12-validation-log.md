# Validation Log

All commands were run from `/home/kzeng/projects/ai-platform-study/lightai-go` unless noted.

| Command | Result | Summary |
| --- | --- | --- |
| `mkdir -p docs/reports/codex-project-wide-review-20260625` | PASS | Created the requested report directory. |
| `sed -n '1,240p' docs/README.md` | PASS | Current docs entrypoint now points to `docs/CURRENT.md`; notes Phase 4 runtime chain. |
| `sed -n '1,260p' docs/PHASE-STATUS.md` | PASS | Reports runtime/model serving implemented with formal MetaX/Huawei/model consistency blockers. |
| `sed -n '1,260p' docs/RELEASE_NOTE_v0.1.9.md` | PASS | Confirms tenant UUID model and no legacy DB migration. |
| `sed -n '1,280p' docs/CURRENT.md` | PASS | Current design claims accepted NVIDIA path and snapshot boundaries. |
| `sed -n '1,320p' docs/design/backend-runtime-runplan-docker.md` | PASS | Reviewed Backend/BackendVersion/BackendRuntime/RunPlan design. |
| `sed -n '1,320p' docs/design/runtime-template-node-runtime-snapshot.md` | PASS | Reviewed NBR-only deployment and snapshot rules. |
| `sed -n '1,320p' docs/design/model-runtime-node-wizard.md` | PASS | Reviewed model/runtime wizard design and root policy. |
| `git status --short && git log --oneline -50` | PASS | Found pre-existing modified `web/package*.json` and many untracked E2E evidence directories; recent commits mostly runtime/bootstrap work. |
| `find docs -maxdepth 4 -type f | sort` | PASS | Large docs tree; many current, archived, phase, and repair reports coexist. |
| `rg --files internal cmd web/src scripts configs \| sort` | PASS | Collected code/script inventory. |
| `rg ... TODO/FIXME/legacy/...` | PASS with truncation | Output was very large; targeted follow-up searches were used. |
| `git diff --stat && git diff -- web/package.json web/package-lock.json` | PASS | Pre-existing Playwright dependency change in web package files. |
| `go test ./...` | PASS | All Go tests passed. |
| `go build ./cmd/server/...` | PASS | Server build passed. |
| `go build ./cmd/agent/...` | PASS | Agent build passed. |
| `cd web && npm test` | PASS | Frontend script tests passed. |
| `cd web && npm run build` | PASS with warning | Build passed; Vite warned main chunk over 500 kB. |
| `go test ./... -cover` | PASS | Coverage uneven: runplan 70.6%, agent runtime 66.3%, server/api 56.2%, auth 3.3%, authz 17.6%, db/rbac/metrics/main 0%. |
| `rg -n "backend_runtime_id" ...` | PASS with truncation | Confirmed current handlers reject deployment `backend_runtime_id`; active scripts and tests still contain old patterns. |
| `rg -n "parameters_json" ...` | PASS with truncation | Confirmed current code uses `parameter_values_json`, but scripts/docs/evidence still use `parameters_json`. |
| `wc -l docs/api/openapi.yaml && sed -n '1,220p' docs/api/openapi.yaml` | PASS | OpenAPI is only 221 lines and documents old runtime/deployment routes. |
| `find web/tests web/src -maxdepth 3 -type f ...` | PASS | Found frontend static/script tests and a few Vue unit tests; no browser E2E test found. |
| `find internal -name '*_test.go' | sort | wc -l` | PASS | 50 Go test files. |
| `find scripts -maxdepth 2 -type f -name 'e2e*.sh' -o -name '*smoke*.sh'` | PASS | Many E2E/smoke scripts exist; several are stale relative to current API. |

Commands not run:

- Real Docker/NVIDIA/MetaX smoke scripts were not run. Reason: this review was scoped to project-wide audit, and hardware/runtime smoke can mutate local runtime state and depends on model images, GPU, Docker, and existing server/agent processes.
- Playwright browser smoke was not run. Reason: no current Playwright workflow was identified in the active test scripts; `npm test` uses static/node tests.

Verification status: build and unit/static tests passed; runtime hardware validation was not performed.
