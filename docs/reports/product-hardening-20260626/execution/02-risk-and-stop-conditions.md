# 02 — Risk and Stop Conditions

Generated: 2026-06-26 | Based on: `00-current-code-inventory.md` + `01-file-level-implementation-plan.md`

## 1. Stop Conditions (HARD — do not proceed)

These conditions require halting the affected workstream and reporting to the user before continuing:

### S-1: Baseline test failures before any change
- **Check:** `go test ./...`, `cd web && npm test`
- **Current status:** ALL PASS (verified 2026-06-26)
- **If triggered:** Do not modify any code. Report which tests fail. Investigate whether environment change or pre-existing regression.
- **Resolution path:** Fix environment or roll back to last known-good state before proceeding.

### S-2: Gateway routing requires a node-level routing decision not covered by this package
- **Check:** During Workstream D, if `model_resolver.go` encounters a case where two deployments on *different nodes* serve the same model name and there is no node-level routing policy defined.
- **Current status:** NOT applicable — first implementation is single-node primary.
- **If triggered:** Stop Workstream D implementation. Report exact scenario. Add node-preference or load-balancing decision to scope before proceeding.

### S-3: Required route conflicts with existing middleware or route prefix
- **Check:** When adding `/v1/models` and `/v1/chat/completions` routes (outside `/api/v1` prefix), verify they do not conflict with any existing handler chain or static file serving.
- **Current status:** `/v1/*` prefix is currently unused in LightAI Go router. No known conflict.
- **If triggered:** Stop. Report exact conflict. Resolve by adjusting route registration order or prefix before proceeding.

### S-4: `RuntimeParameterEditor.vue` cannot be enhanced without breaking its existing (unused) contract
- **Check:** Before modifying `RuntimeParameterEditor.vue`, verify its current props/emits interface is NOT depended on by any component (import grep = 0 results confirmed 2026-06-26).
- **Current status:** Component is dead code — safe to modify.
- **If triggered:** (Unexpected import found.) Stop. Document every consumer. Decide whether to create a new component or refactor all consumers.

### S-5: Docker/GPU/runtime smoke is externally blocked and cannot be re-run
- **Check:** Before declaring Workstream E complete, run `scripts/e2e-real-smoke-all-three.sh` or equivalent.
- **Current status:** Last smoke evidence from 2026-06-25 shows all three backends PASS. Hardware availability required.
- **If triggered:** Do not hide. Classify each backend as `PASS`, `DOCUMENTED_BLOCKER` (external: no GPU), or `FAIL` (code/config bug). Fix code/config bugs. Document external blocks honestly in `final-regression-report.md`.

### S-6: Page rename (`RunnerConfigsPage` → `NodeRuntimeConfigsPage`) breaks import graph
- **Check:** Before renaming, grep for ALL imports of `RunnerConfigsPage` and `BackendRuntimesPage`.
- **Current status:** `RunnerConfigsPage` imported by `router/index.ts` and `tests/runtimeBoundaryUi.test.mjs`. `BackendRuntimesPage` imported by `router/index.ts` only.
- **If triggered:** (Additional consumers found.) Update all import paths before renaming. Do NOT leave broken imports.

### S-7: Clean DB rebuild deletes data that cannot be recreated
- **Check:** If the user has populated model artifacts, backends, runtimes, or deployments that cannot be re-seeded from catalog, DB rebuild is destructive.
- **Current status:** Seed data comes from `configs/backend-catalog/`. `SeedCatalog()` repopulates all backends/runtimes from YAML.
- **If triggered:** Export data before rebuild, or implement ALTER TABLE migration instead of clean rebuild. But per project rules: "Do not preserve legacy dirty config or old DB compatibility." — default is clean rebuild.

## 2. Risk Matrix

Each risk is scored: **L**ow / **M**edium / **H**igh for Likelihood × Impact = Risk Level.

| ID | Risk | Likelihood | Impact | Risk Level | Mitigation |
|---|---|---|---|---|---|
| R1 | Page rename breaks route-based navigation in browser bookmarks | H | L | LOW | Old `/runner-configs` path → add redirect in `router/index.ts` (`redirect: '/node-runtime-configs'`). Old route kept as alias. |
| R2 | Naming changes in i18n keys cause missing translation errors in tests | M | L | LOW | `i18nKeys.test.mjs` and `i18nMissingKeys.test.mjs` catch this. Run `npm test` after each i18n batch. |
| R3 | `RuntimeParameterEditor` enhancement introduces new props that require updates to all consumers | L | M | LOW | Only consumer will be the new wizard components (after Workstream B) and the three pages (after Workstream C). All new integrations are planned. |
| R4 | RunPlan resolver and preview endpoint diverge (preview shows different result than actual start) | M | H | **MEDIUM** | Share `preflightDeployment()` function between preview and start handlers. Test: `preview_and_start_use_same_resolver_path` explicitly verifies. |
| R5 | Gateway proxy introduces latency/timeout issues for chat completions | M | M | **MEDIUM** | Set configurable proxy timeout (default 120s for chat, 30s for models list). Implement context cancellation propagation. Test with slow backend responses. |
| R6 | API key hash/comparison uses wrong algorithm causing auth failures | L | H | MEDIUM | Use bcrypt for key hashing (same as password hashing in existing codebase). Test: `hashed_key_matches`, `hashed_key_rejects_wrong_key`. |
| R7 | Gateway usage recording overhead slows down chat completions | L | M | LOW | Usage record INSERT is async (fire-and-forget after response sent). Audit write is sync but lightweight. |
| R8 | Model resolver returns ambiguous match for common model names | M | M | **MEDIUM** | Implement clear resolution order: (1) deployment `served_model_name` exact match, (2) deployment `name` exact match, (3) model_artifact `name`/`display_name`. If multiple matches at same tier, return structured error with candidate list. Test: `ambiguous_multiple_matches_rejected`. |
| R9 | Frontend build fails due to new component imports or TypeScript errors | M | M | **MEDIUM** | Run `cd web && npm run build` after each workstream commit. Do not proceed to next workstream until build passes. `vue-tsc --noEmit` catches type errors. |
| R10 | Existing Go tests break due to handler signature changes or route conflicts | L | M | LOW | All new handlers are additive. Refactored `preflight_handlers.go` extraction preserves existing behavior. Run `go test ./...` after each Go change. |
| R11 | SGLang capability blocker resurfaces during runtime smoke | M | M | **MEDIUM** | Previous evidence reported capability blocker for SGLang; current catalog has `capabilities_detail` added. Run SGLang smoke first in runtime validation to catch early. If blocked, classify as external dependency or catalog bug — fix if code/config, document if external. |
| R12 | i18n keys removed by mistake cause Vue template rendering errors | L | M | LOW | `i18nMissingKeys.test.mjs` catches missing keys in templates. Run after every i18n change. |
| R13 | `config_overrides` API format mismatch between frontend send and backend expect | M | H | **MEDIUM** | Document exact format in implementation plan. Backend `applyConfigOverrides()` in `configset_helpers.go:243` accepts `parameter_values`, `disabled_parameters`, `env`. Frontend `RuntimeParameterEditor` must emit matching format. Test: `config_overrides_roundtrip` — create → read back → values match. |
| R14 | `POST /api/v1/deployments/preview` response size too large (full RunPlan JSON) | L | L | LOW | RunPlan is already returned by dry-run endpoint. Preview returns same structure. Acceptable. |
| R15 | 15 legacy-contract scripts in archive cause confusion about which scripts to run | L | L | LOW | Add `README.md` in `scripts/archive/legacy-contract/` with explicit deprecation notice. |

## 3. External Dependencies and Blockers

| Dependency | Needed for | Status | If blocked |
|---|---|---|---|
| NVIDIA GPU + Docker | Runtime smoke (Workstream E) | Available per last evidence (2026-06-25) | Classify as `DOCUMENTED_BLOCKER: external_hardware` |
| SGLang Docker image | SGLang runtime smoke | Available per last evidence | Same as above |
| vLLM Docker image | vLLM runtime smoke | Available per last evidence | Same as above |
| llama.cpp Docker image + GGUF model | llama.cpp runtime smoke | Available per last evidence | Same as above |
| Playwright (installed, unconfigured) | Browser smoke (Workstream E) | Binary present, no config | Use manual browser verification or skip browser smoke — not a blocker for code changes |
| MetaX hardware | MetaX collector validation | NOT in scope for this hardening | Already classified as `DOCUMENTED_BLOCKER: external_hardware` in RC1 review |

## 4. Fallback Positions

If a workstream cannot be completed:

| Workstream | Minimum viable | Degraded but acceptable |
|---|---|---|
| A — Naming | Dictionary created + page titles fixed | Route renames deferred (old routes kept as aliases) |
| B — Deployment UI | Preview endpoint + wizard created | Wizard simplified to 3 steps instead of 6 |
| C — Runtime Parameters | Editor integrated in all 3 pages | llama.cpp editor deferred (fewest params) |
| D — Gateway | `/v1/models` + `/v1/chat/completions` + API keys + usage | Embeddings/completions endpoints deferred; usage query API deferred |
| E — Regression | Go + frontend tests pass + build + diff | Runtime smoke classified as externally blocked if GPU unavailable |

## 5. Rollback Plan

Each commit is self-contained within one workstream. If a workstream commit introduces problems:

1. `git revert <commit>` — each commit is designed to be independently revertable.
2. Re-run `go test ./...` and `cd web && npm test` to verify rollback state.
3. Report reverted scope and reason.

Workstreams A, B, C modify overlapping frontend files. If one workstream needs rollback after the next is committed, the revert may have conflicts. Mitigation: complete each workstream fully (tests pass + build pass + validation) before committing. Do not commit partial work.

## 6. Pre-Commit Gate Checklist

Before each commit, verify:

- [ ] `go test ./...` passes with zero failures
- [ ] `go build ./cmd/server/...` passes
- [ ] `go build ./cmd/agent/...` passes
- [ ] `cd web && npm test` passes with zero failures
- [ ] `cd web && npm run build` passes (warnings OK, errors NOT OK)
- [ ] `git diff --check` is clean
- [ ] No `TODO`, `FIXME`, `later` comments added (per Problem Closure Policy §7.2)
- [ ] All discovered problems are either FIXED or in formal open-issues document
- [ ] No raw `ConfigSet`, `RunPlan`, `NBR` in user-facing i18n strings (per Workstream A)
- [ ] No raw UUIDs displayed as primary labels in table columns (per Workstream A)
- [ ] No `backend_runtime_id` accepted as deployment selector (per Workstream B)

## 7. Approval Checkpoint

After all three plan files are created and reviewed, the next action depends on user instruction:

- **If user explicitly instructed AUTORUN:** Proceed to implementation without waiting.
- **Otherwise:** STOP here. Wait for user approval before modifying any code.

### Approval request summary:

| Plan file | Path |
|---|---|
| Current code inventory | `docs/reports/product-hardening-20260626/execution/00-current-code-inventory.md` |
| Implementation plan | `docs/reports/product-hardening-20260626/execution/01-file-level-implementation-plan.md` |
| Risk and stop conditions | `docs/reports/product-hardening-20260626/execution/02-risk-and-stop-conditions.md` |

### Top 10 concrete code changes:

1. **Rename `RunnerConfigsPage.vue` → `NodeRuntimeConfigsPage.vue`**, update route `/runner-configs` → `/node-runtime-configs`, update all i18n keys from `runnerConfigs` → `nodeRuntimeConfigs` (Workstream A)
2. **Replace thin create dialog** in `ModelDeploymentsPage.vue` with 6-section `DeploymentWizard.vue` component (Workstream B)
3. **Add `POST /api/v1/deployments/preview`** endpoint in `deployment_preview_handlers.go` — shared resolver path with start, returns RunPlan/lint/preflight before save (Workstream B)
4. **Wire `RuntimeParameterEditor.vue`** into `BackendRuntimesPage.vue`, `NodeRuntimeConfigsPage.vue`, and `ModelDeploymentsPage.vue` — enable editing of all 30+ catalog parameters (Workstream C)
5. **Enhance `RuntimeParameterEditor.vue`** with props: `layer`, `baseValues`, `showSource`, `showAdvanced`; emits: `validate`; behavior: required locked, optional toggle, source diff, backend/vendor filter (Workstream C)
6. **Add `api_keys` + `gateway_usage_records` tables** to `db/db.go` — clean DB schema only, no migration (Workstream D)
7. **Add `GET /v1/models` + `POST /v1/chat/completions`** external routes with `GatewayAuthMiddleware` in `gateway_handlers.go` — Bearer API key auth, model routing, backend proxy, usage recording (Workstream D)
8. **Create `ApiKeysPage.vue` + `GatewayUsagePage.vue`** — API key CRUD, usage table with summary stats (Workstream D)
9. **Add 7 lint rules** to `runplan/lint.go`: duplicate args, env/CLI conflict, platform arg override, unsupported param, vendor incompatibility, disabled field applied, missing required (Workstream C)
10. **Strip "ConfigSet"/"RunPlan"/"NBR" from all user-facing UI labels and i18n strings** — replace with "Technical Configuration"/"运行计划"/runtime template name (Workstream A)

### DB impact:
- ADD `api_keys` table (6 columns + 2 indexes)
- ADD `gateway_usage_records` table (17 columns + 3 indexes)
- ADD 3 permission seeds in `auth/bootstrap.go`
- Clean DB rebuild required: `rm -f /tmp/lightai/data/lightai.db`
- No existing table changes, no data migration

### API impact:
- 8 new routes (1 deployment preview + 2 gateway external + 5 API key/usage management)
- 3 enhanced routes (accept expanded parameter values/overrides)
- 0 routes removed
- 0 breaking changes to existing route contracts

### Validation commands (minimum before commit):
```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
git diff --check
git status --short
```
