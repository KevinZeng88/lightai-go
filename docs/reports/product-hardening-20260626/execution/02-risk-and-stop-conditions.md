# 02 — Risk and Stop Conditions

Revised: 2026-06-26 | Scope: 模型运行管理闭环 (excludes OpenAI Gateway/API Key/Usage Metering)

## 1. Stop Conditions (HARD — do not proceed)

These conditions require halting the affected workstream and reporting to the user before continuing:

### S-1: Baseline test failures before any change
- **Check:** `go test ./...`, `cd web && npm test`
- **Current status:** ALL PASS (verified 2026-06-26)
- **If triggered:** Do not modify any code. Report which tests fail. Investigate environment or pre-existing regression.
- **Resolution path:** Fix environment or roll back to last known-good state.

### S-2: `RuntimeParameterEditor.vue` cannot be enhanced without breaking existing consumers
- **Check:** Before modifying, verify import count = 0 (confirmed 2026-06-26).
- **Current status:** Component is dead code — safe to modify.
- **If triggered:** (Unexpected import found.) Stop. Document consumers. Decide: enhance in-place vs. create new component.

### S-3: Docker/GPU/runtime smoke is externally blocked
- **Check:** Before declaring Workstream E complete, run `scripts/e2e-real-smoke-all-three.sh` or equivalent.
- **Current status:** Last smoke evidence from 2026-06-25 shows all three backends PASS.
- **If triggered:** Classify each backend as `PASS`, `DOCUMENTED_BLOCKER` (external: no GPU), or `FAIL` (code/config bug). Fix code/config bugs. Document external blocks honestly.

### S-4: Preview and start resolver paths diverge
- **Check:** During Workstream C, verify `HandleDeploymentPreview` and `HandleStartDeployment` share the same `preflightDeployment()` (or extracted `performDeploymentPreflight()`) function.
- **Current status:** Not yet implemented — verify during implementation.
- **If triggered:** (Different resolver path found.) Stop Workstream C. Unify resolver before proceeding. Add test `preview_and_start_use_same_resolver_path`.

### S-5: `POST /api/v1/deployments/preview` conflicts with existing route
- **Check:** Verify no existing handler at `/api/v1/deployments/preview` (confirmed: only `/deployments/preflight` and `/deployments/{id}/dry-run` exist).
- **Current status:** Path is unused.
- **If triggered:** (Route conflict.) Rename endpoint or adjust handler registration order.

### S-6: Frontend build breaks due to new component imports or TypeScript errors
- **Check:** Run `cd web && npm run build` after each commit.
- **Current status:** Build passes at baseline (3.29s).
- **If triggered:** Fix build before next commit. Do NOT accumulate build failures across commits.

### S-7: Existing instance lifecycle tests fail after Workstream D changes
- **Check:** Run `go test ./internal/server/api/... -run 'Instance|Lifecycle|Start|Stop'` after D changes.
- **Current status:** All tests pass at baseline.
- **If triggered:** Roll back D changes. Fix incrementally. Re-verify.

### S-8: Naming changes cause i18n missing-key errors
- **Check:** `web/tests/i18nMissingKeys.test.mjs` after each i18n change.
- **Current status:** All 220+ keys matched at baseline.
- **If triggered:** Add missing keys before proceeding. Do NOT delete old keys without updating all template references.

## 2. Risk Matrix

| ID | Risk | Likelihood | Impact | Risk Level | Mitigation |
|---|---|---|---|---|---|
| R1 | i18n value changes cause missing-key test failures | M | L | LOW | `i18nMissingKeys.test.mjs` catches. Keys are NOT renamed — only values change. Run `npm test` after each locale edit. |
| R2 | `RuntimeParameterEditor` enhancement introduces new props requiring all consumers to update | L | M | LOW | Editor is currently dead code (0 imports). New consumers are all planned (BackendRuntimesPage, RunnerConfigsPage, DeploymentOverrideEditor). |
| R3 | RunPlan resolver and preview endpoint diverge | M | H | **MEDIUM** | Share `performDeploymentPreflight()` across preview, preflight, and start handlers. Test: `preview_and_start_use_same_resolver_path`. |
| R4 | Frontend build fails due to new component imports or TypeScript errors | M | M | **MEDIUM** | Run `cd web && npm run build` after each commit. `vue-tsc --noEmit` in build pipeline catches type errors. |
| R5 | Existing Go tests break due to handler signature changes or refactored preflight | L | M | LOW | Refactored `preflight_handlers.go` preserves existing behavior. All new handlers are additive. `go test ./...` after each Go change. |
| R6 | SGLang capability blocker resurfaces during runtime smoke | M | M | **MEDIUM** | Previous evidence noted capability blocker; current catalog has `capabilities_detail` added. Run SGLang smoke first. If blocked: fix if code/config, document if external. |
| R7 | `config_overrides` API format mismatch between frontend emit and backend expect | M | H | **MEDIUM** | Backend `applyConfigOverrides()` accepts `parameter_values`, `disabled_parameters`, `env`. Frontend must emit matching format. Test: `config_overrides_roundtrip` (create → read → values match). |
| R8 | `POST /api/v1/deployments/preview` response size too large | L | L | LOW | RunPlan JSON already returned by dry-run. Same structure. Acceptable. |
| R9 | Legacy-contract scripts cause confusion about which E2E to run | L | L | LOW | Add `README.md` in `scripts/archive/legacy-contract/` with deprecation notice. |
| R10 | Parameter editor backend/vendor filter hides too many params | L | M | LOW | Filter logic uses `backend` and `vendor` fields from catalog YAML. Test with each backend to verify correct set. Add test: `vllm_params_exclude_sglang_only_params`. |
| R11 | Instance lifecycle hardening introduces regression in start/stop flow | L | H | MEDIUM | D changes are verification + gap-filling, not redesign. Run full lifecycle test suite before committing. |
| R12 | Naming cleanup accidentally removes i18n keys still referenced in templates | L | M | LOW | Keys are NOT removed — only values change. If keys are renamed, `i18nMissingKeys.test.mjs` catches. |

## 3. External Dependencies and Blockers

| Dependency | Needed for | Status | If blocked |
|---|---|---|---|
| NVIDIA GPU + Docker | Runtime smoke (Workstream E) | Available per last evidence (2026-06-25) | Classify as `DOCUMENTED_BLOCKER: external_hardware` |
| SGLang Docker image | SGLang runtime smoke | Available per last evidence | Same |
| vLLM Docker image | vLLM runtime smoke | Available per last evidence | Same |
| llama.cpp Docker image + GGUF model | llama.cpp runtime smoke | Available per last evidence | Same |
| Playwright (installed, unconfigured) | Browser smoke (Workstream E) | Binary present, no config | Use manual browser verification; not a blocker |
| MetaX hardware | MetaX collector validation | NOT in scope | Already `DOCUMENTED_BLOCKER` in RC1 |

## 4. Fallback Positions

| Workstream | Minimum viable | Degraded but acceptable |
|---|---|---|
| B — Runtime Parameters | Editor integrated in BackendRuntimes + RunnerConfigs pages | Deployment override editor deferred to after C |
| C — Deployment UI | Preview endpoint + wizard created | Wizard simplified to 3 steps (model → NBR → preview) instead of 6 |
| D — Start/Stop/Logs | Instance state display fixes | Auto-refresh optimization deferred |
| E — E2E Regression | Go + frontend tests pass + build + diff | Runtime smoke classified as externally blocked if GPU unavailable |
| F — Naming Cleanup | i18n values fixed + hardcoded labels fixed | `docs/engineering/naming-dictionary.md` deferred |
| G — Gateway Notes | Future notes document created | N/A — document only |

## 5. Rollback Plan

Each commit is self-contained. If a commit introduces problems:

1. `git revert <commit>` — each commit independently revertable
2. Re-run `go test ./...` and `cd web && npm test` to verify rollback
3. Report reverted scope and reason

Workstreams B, C, F modify overlapping frontend files (`ModelDeploymentsPage.vue`, `RunnerConfigsPage.vue`, `BackendRuntimesPage.vue`). Mitigation: complete each workstream fully (tests + build pass) before committing. Do not commit partial work.

## 6. Pre-Commit Gate Checklist

Before each commit:

- [ ] `go test ./...` — zero failures
- [ ] `go build ./cmd/server/...` — pass
- [ ] `go build ./cmd/agent/...` — pass
- [ ] `cd web && npm test` — zero failures
- [ ] `cd web && npm run build` — pass (warnings OK)
- [ ] `git diff --check` — clean
- [ ] No `TODO`, `FIXME`, `later` comments added (Problem Closure Policy §7.2)
- [ ] All discovered problems are FIXED or in formal open-issues document
- [ ] No raw `ConfigSet` in user-facing Vue template text (Workstream F)
- [ ] No raw `NBR` in i18n display values (Workstream F)
- [ ] No raw UUIDs as primary table column values (Workstream F)
- [ ] `node_backend_runtime_id` used in deployment create payload, not `backend_runtime_id` (Workstream C)

## 7. Revised DB Impact

**DB change: NONE.**

No new tables. No schema changes. No migrations. No clean DB rebuild required.

Gateway tables (`api_keys`, `gateway_usage_records`) and related permission seeds are deferred to future workstream — see `future-openai-gateway-notes.md`.

## 8. Revised API Impact

**1 new route, 3 enhanced routes, 0 removed:**

| Method | Path | Change |
|---|---|---|
| POST | `/api/v1/deployments/preview` | ADD |
| PATCH | `/api/v1/backend-runtimes/{id}` | ENHANCE |
| PATCH | `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | ENHANCE |
| PATCH | `/api/v1/deployments/{id}` | ENHANCE |

**Routes NOT added:** `/v1/models`, `/v1/chat/completions`, `/api/v1/api-keys`, `/api/v1/gateway/usage`.

## 9. Approval Checkpoint

After plan documents are revised and reviewed:

- **If user explicitly instructed AUTORUN:** Proceed to implementation.
- **Otherwise:** STOP here. Wait for user approval.

### Plan file paths:

| File | Path |
|---|---|
| Current code inventory | `docs/reports/product-hardening-20260626/execution/00-current-code-inventory.md` |
| Implementation plan (revised) | `docs/reports/product-hardening-20260626/execution/01-file-level-implementation-plan.md` |
| Risk and stop conditions (revised) | `docs/reports/product-hardening-20260626/execution/02-risk-and-stop-conditions.md` |
| Gateway future notes | `docs/reports/product-hardening-20260626/execution/future-openai-gateway-notes.md` |

### Revised implementation order:

```
A (inventory, done) → B (NBR / runtime parameters) → C (deployment UI + preview) → D (start/stop/status/logs) → E (E2E regression) → F (naming cleanup, safe only) → G (gateway future notes, document only)
```

### Validation commands (minimum before commit):
```bash
go test ./...
go build ./cmd/server/... && go build ./cmd/agent/...
cd web && npm test && npm run build
git diff --check
git status --short
```
