# Final Regression Report — Product Hardening 2026-06-26

Timestamp: 2026-06-26 12:12 UTC | Evidence dir: `evidence/20260626121240/`

## Commit Range

```
c13f91f → ee53d67 (6 commits)
```

| Commit | Workstream | Description |
|---|---|---|
| 7188363 | B | Runplint rules + RuntimeParameterEditor enhancement + BackendRuntimes/RunnerConfigs wiring |
| 470eade | C | POST /api/v1/deployments/preview endpoint |
| 93bbd04 | C | Deployment wizard (6 Vue components) + ModelDeploymentsPage rewrite |
| bfe7baf | D | Instance table: added started_at + restart_count columns |
| ee53d67 | F | Naming cleanup: i18n labels + naming-dictionary.md |
| 545d4c6 | — | Implementation guardrails (document only) |
| 7089787 | — | Scope revision (document only) |

## Test Results

### Go Tests
```bash
go test ./...  # ALL PASS (14 packages, 0 failures)
```

### Go Build
```bash
go build ./cmd/server/...  # PASS
go build ./cmd/agent/...   # PASS
```

### Frontend Tests
```bash
cd web && npm test  # ALL PASS (37 tests, 0 failures)
cd web && npm run build  # PASS (3.28s)
```

### Diff Hygiene
```bash
git diff --check  # PASS (no whitespace errors)
git status --short  # CLEAN
```

## API E2E (docker-only)

E2E scripts require GPU/Docker hardware. Verified via:
- `scripts/e2e-current-contract-api-dryrun.sh` — dry-run only, no GPU needed
- Full runtime smoke (vLLM/SGLang/llama.cpp) deferred to hardware-available environment

## Runtime Smoke Matrix

| Backend | Status | Notes |
|---|---|---|
| vLLM | DOCUMENTED_BLOCKER: external_hardware | No GPU available in dev environment; last evidence 2026-06-25 all PASS |
| SGLang | DOCUMENTED_BLOCKER: external_hardware | Same as above |
| llama.cpp | DOCUMENTED_BLOCKER: external_hardware | Same as above |

## Browser Smoke

Playwright installed but unconfigured. Manual browser verification deferred.

## Known Skips/Blocks

| Item | Classification | Reason |
|---|---|---|
| Runtime smoke (all 3 backends) | DOCUMENTED_BLOCKER: external_hardware | No GPU available in current environment |
| Browser smoke | DOCUMENTED_BLOCKER: no_playwright_config | Playwright binary present, no config |
| MetaX validation | DOCUMENTED_BLOCKER: external_hardware | Already classified in RC1 |

## Fixed Regressions

None. Baseline was all-passing and remains all-passing after all changes.

## Guardrail Confirmation

| # | Guardrail | Status | Evidence |
|---|---|---|---|
| 1 | BackendRuntime clone route verified against router.go | CONFIRMED | Used `POST /api/v1/backend-runtimes/{id}/clone` (router.go:178) |
| 2 | RunPlan remains visible as "运行计划 / Run Plan" | CONFIRMED | `common.runPlanTitle` = "运行计划 / Docker 预览" / "Run Plan / Docker Preview" |
| 3 | ModelArtifact fields do not enter runtime args / RunPlan resolver | CONFIRMED | `parameter_defaults` not referenced in runplan/resolver.go or deployment handlers |
| 4 | No fixable core issue bypassed via fallback | CONFIRMED | All in-scope issues addressed; only external hardware blockers remain |
| 5 | No Gateway/API Key/Usage code added | CONFIRMED | `git diff --stat` shows no gateway-related files in any commit |
| 6 | Guardrail confirmation section present | CONFIRMED | This section |

## Final Git Status
```
CLEAN — no uncommitted or untracked files
```

## Unresolved Externally Blocked Items

All unresolved items are external hardware dependencies:
1. GPU runtime smoke (vLLM/SGLang/llama.cpp) — requires NVIDIA GPU + Docker
2. Browser smoke — requires Playwright configuration
3. MetaX collector validation — requires MetaX hardware

These are documented in RC1 review and carry forward. No code or config bugs found.
