# 03 — Implementation Guardrails

Created: 2026-06-26 | Status: **MANDATORY — read before any code change**

These guardrails are hard constraints for the current product hardening round. They override any conflicting wording in `01-file-level-implementation-plan.md`. Every workstream closeout must explicitly confirm each guardrail.

---

## Guardrail 1: BackendRuntime clone route must follow current code contract

**Constraint:** Before implementing BackendRuntime clone (Workstream B — BackendRuntimesPage "Clone from system template" button), verify the actual route in `internal/server/api/router.go`.

**Verified routes (2026-06-26):**

| Entity | Actual Route | Handler | Router Line |
|---|---|---|---|
| BackendVersion clone | `POST /api/v1/backend-versions/{version_id}/clone` | `HandleCloneBackendVersion` | 138 |
| **BackendRuntime clone** | **`POST /api/v1/backend-runtimes/{id}/clone`** | **`HandleCloneBackendRuntime`** | **178** |

**DO NOT:**
- Use `/api/v1/backend-versions/{id}/clone` for BackendRuntime clone (that is the Backend**Version** clone route).
- Invent a clone route without checking `router.go` first.
- Call `HandleCloneBackendVersion` when the intent is to clone a BackendRuntime.

**DO:**
- Use `POST /api/v1/backend-runtimes/{id}/clone` for BackendRuntime clone.
- If the plan text conflicts with the actual route, the actual route in `router.go` takes precedence.
- Document any plan-vs-code discrepancy in the workstream closeout.

---

## Guardrail 2: RunPlan remains a product concept

**Constraint:** Do not delete or hide the RunPlan concept from the UI.

**User-facing labels:**

| Context | zh-CN | en-US |
|---|---|---|
| Preview/detail panel title | 运行计划 | Run Plan |
| Docker preview section | Docker 预览 | Docker Preview |
| Diagnostic/debug context | 运行计划详情 | Run Plan Details |
| Source trace | 参数来源 | Parameter Source |

**DO NOT:**
- Remove the RunPlan preview panel from the deployment wizard.
- Replace "运行计划 / Run Plan" with a generic label like "Configuration JSON".
- Hide the RunPlan JSON viewer from diagnostic contexts.

**DO:**
- Keep "运行计划 / Run Plan" as the user-facing label for `ResolvedRunPlan`.
- Keep Docker command preview visible before deployment start.
- Keep source trace visible in preview/diagnostic contexts.
- Use `JsonViewer` component (already exists) for RunPlan JSON display.

**Allowed technical-only labels:**
- `ConfigSet` — NOT a user-facing label; use "技术配置 / Technical Config" in drawer titles.
- `NBR` — NOT a user-facing acronym; use "节点运行配置 / Node Runtime Config" or BackendRuntime display name in text.

---

## Guardrail 3: ModelArtifact must not carry runtime serving parameters

**Constraint:** ModelArtifact's `parameter_defaults` field must not be repurposed or renamed to carry backend runtime serving parameters. The model layer is for model facts and hints only.

**Allowed in ModelArtifact UI + DB:**
- Display name, format, task type, capabilities, architecture, quantization
- Recommended context length (model fact, not Docker arg)
- Tokenizer/chat template metadata
- Model-level notes and descriptions
- Safe hints that do NOT enter RunPlan resolution

**FORBIDDEN in ModelArtifact UI + DB:**
- `--max-model-len`
- `--served-model-name`
- `--gpu-memory-utilization`
- Docker args (`--tensor-parallel-size`, `--pipeline-parallel-size`, etc.)
- Docker env vars
- Devices / volumes / mounts
- Host ports / container ports
- Backend runtime security options (`--trust-remote-code`, `--enforce-eager`)
- Any field that enters `runplan.Resolve()`

**Required test evidence:**
- Go test: `model_page_parameter_defaults_not_used_as_runtime_args` — verify `parameter_defaults` from ModelArtifact are NOT passed to `runplan.Resolve()`
- Go test: `deployment_does_not_mutate_nbr` — verify deployment overrides write to deployment record, not to NBR
- Frontend test: ModelArtifactsPage placeholder text does NOT reference `--max-model-len`, `--gpu-memory-utilization`, `--served-model-name`

**If model hints are retained on ModelArtifact:**
- Field must be explicitly labeled "模型提示 / Model Hints — not runtime args"
- Must NOT be wired to RunPlan resolution
- Must NOT be the default value source for deployment parameter editor

---

## Guardrail 4: Fallback/degraded acceptable cannot bypass fixable core issues

**Constraint:** The "Fallback Positions" table in `02-risk-and-stop-conditions.md` describes degraded-but-acceptable outcomes for *externally blocked* issues. It does NOT authorize skipping fixable issues within the current model runtime management scope.

**Fixable-in-scope issues that MUST be addressed:**
- Runtime parameter editing at BackendRuntime, NBR, and Deployment layers
- Deployment preview endpoint returning RunPlan + lint + preflight
- Deployment wizard UI with model/NBR/service/override/preview
- Preflight gate blocking start on validation errors
- Instance lifecycle: start, stop, status transitions, log fetch
- Lint rules: duplicate args, env/CLI conflict, platform arg override, etc.
- ModelArtifactsPage cleanup (remove runtime args from model layer)
- All Go + frontend tests passing at each commit

**Allowed to classify as DOCUMENTED_BLOCKER:**
- GPU hardware unavailable for runtime smoke (external dependency)
- SGLang capability blocked by upstream SGLang version (external dependency, document exact error)
- Playwright unconfigured for browser smoke (use manual verification instead)
- MetaX hardware unavailable (already DOCUMENTED_BLOCKER in RC1)

**Decision rule:** If a problem is in the current scope, is locatable in code, and is fixable without cascading architectural changes, it MUST be fixed. Only defer if:
1. The fix requires a DB migration that the project policy forbids;
2. The fix requires a new external dependency not yet approved;
3. The fix requires a cross-cutting architectural refactor explicitly out of scope;
4. The issue is caused by an external system (GPU driver, Docker engine, upstream backend image).

---

## Guardrail 5: OpenAI Gateway remains document-only in this round

**Constraint:** No OpenAI Gateway, API Key, Usage Metering, or Billing code is implemented in this round.

**FORBIDDEN in this round:**
- `api_keys` table creation in `db/db.go`
- `gateway_usage_records` table creation in `db/db.go`
- `GET /v1/models` or `POST /v1/chat/completions` route registration
- API key CRUD routes (`/api/v1/api-keys`)
- Gateway usage query routes (`/api/v1/gateway/usage`)
- `GatewayAuthMiddleware` implementation
- `gateway/model_resolver.go` implementation
- Gateway proxy handler implementation
- `ApiKeysPage.vue` or `GatewayUsagePage.vue` creation
- API key or usage API client files
- `api_key:*` or `gateway_usage:*` permission seeds
- Any billing or cost calculation logic

**Allowed:**
- `future-openai-gateway-notes.md` exists as design reference (already created)
- Referencing `future-openai-gateway-notes.md` in closeout documents

**If gateway-related code is accidentally introduced:** Revert before commit. Check `git diff --stat` for any gateway-related files.

---

## Guardrail 6: Workstream closeout must explicitly confirm guardrails

**Constraint:** Every workstream closeout document (or the final `final-regression-report.md`) must include a "Guardrail Confirmation" section with this exact checklist:

```markdown
## Guardrail Confirmation

| # | Guardrail | Status | Evidence |
|---|---|---|---|
| 1 | BackendRuntime clone route verified against router.go | CONFIRMED / N/A | [route path used, router.go line] |
| 2 | RunPlan remains visible as "运行计划 / Run Plan" | CONFIRMED | [UI component, i18n key, or screenshot] |
| 3 | ModelArtifact fields do not enter runtime args / RunPlan resolver | CONFIRMED | [test name + result] |
| 4 | No fixable core issue bypassed via fallback | CONFIRMED | [list of issues fixed vs. documented_blocker] |
| 5 | No Gateway/API Key/Usage code added | CONFIRMED | [git diff --stat shows no gateway files] |
| 6 | Guardrail confirmation section present | CONFIRMED | [this section] |
```

If a guardrail is N/A for a specific workstream, state `N/A` with a one-line reason. If a guardrail is violated, do NOT close the workstream — fix the violation first.
