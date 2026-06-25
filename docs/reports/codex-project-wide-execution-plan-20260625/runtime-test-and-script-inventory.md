# Runtime Test & Script Inventory

## Categories

| Category | Existing Asset | Status | Purpose | Reuse Plan | Fix Required |
|----------|---------------|--------|---------|------------|--------------|
| Env Prep | scripts/start-all.sh | CURRENT | Start server+agent+observability | Direct reuse | None |
| Env Prep | scripts/start-server.sh | CURRENT | Start server | Direct reuse | None |
| Env Prep | scripts/start-agent.sh | CURRENT | Start agent | Direct reuse | None |
| Env Prep | scripts/stop-all.sh | CURRENT | Stop all services | Direct reuse | None |
| Docker | scripts/start-all.sh (implicit) | CURRENT | Docker GPU check | Reuse | None |
| API E2E | scripts/e2e-model-runtime-param-trace.sh | NEEDS-REPAIR | Param trace vLLM/SGLang/llama.cpp | Fix payload to current contract | Update to node_backend_runtime_id |
| API E2E | scripts/e2e-matrix-verifier.sh | NEEDS-REPAIR | Cross-backend matrix | Fix payload | Update to current contract |
| API E2E | scripts/e2e-dryrun-parameter-matrix-enhanced.sh | NEEDS-REPAIR | Dry-run parameter matrix | Fix payload | Update to current contract |
| API E2E | scripts/e2e-model-runtime-wizard-nvidia-api.sh | NEEDS-REPAIR | Full vLLM wizard via API | Fix payload | Update to current contract |
| Real Smoke | scripts/e2e-real-smoke-all-three.sh | CURRENT | vLLM+SGLang+llama.cpp real | Direct reuse | None |
| Real Smoke | scripts/e2e-model-runtime-wizard-nvidia-vllm.sh | CURRENT | vLLM real test | Reuse | None |
| Bootstrap | scripts/lightai-bootstrap.sh | CURRENT | Auth+Catalog+Models+Runtimes+DryRun+Full+Export | Direct reuse | None |
| Bootstrap | scripts/lib/bootstrap-export.py | CURRENT | API-first profile export | Direct reuse | None |
| Package | scripts/package-release.sh | CURRENT | Release artifact build | Direct reuse | None |
| Package | scripts/package-release-docker.sh | CURRENT | Docker release build | Reuse | None |
| Test | web/tests/*.test.mjs | CURRENT | Static web tests (7 files) | Direct reuse | None |
| Test | internal/server/api/*_test.go | CURRENT | Go API tests (17 files) | Direct reuse | None |
| Test | internal/server/runplan/*_test.go | CURRENT | RunPlan tests (9 files) | Direct reuse | None |
| Test | internal/agent/*_test.go | CURRENT | Agent tests (8 files) | Direct reuse | None |
| Evidence | docs/reports/model-runtime-node-wizard/e2e-*/ | HISTORICAL | Prior E2E evidence | Reference only | Do NOT use for current validation |
| Evidence | docs/reports/phase-3/ | HISTORICAL | Phase 3 closeout docs | Reference only | Superseded |
| Browser | None | MISSING | Browser/Playwright smoke | NEED CREATE | P1 gap |
| OpenAPI | docs/api/openapi.yaml | STALE | API documentation | NEED UPDATE | R-006 |

## New Smoke Script (if needed)
If existing scripts are fixed and verified, a consolidated smoke entry can be created at:
```bash
scripts/e2e-current-runtime-smoke.sh
```
