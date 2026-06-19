# Runtime Config Web Flow Test Gap Closeout

> Date: 2026-06-20
> Status: FIXED
> Scope: Why user-visible Web bugs weren't caught by existing tests, and fixes

## 1. Test Gap Table

| Problem found by user | Why existing tests missed it | New regression coverage | Evidence |
|---|---|---|---|
| `node_id and nbr_id are required` on check click | Go tests called handler directly with `newReq(..., map[string]string{"id": nodeID, ...})` — never tested real HTTP router with path params. `PathValue("node_id")` vs route `{id}` mismatch only detectable via router dispatch or targeted handler test with wrong PathValue name. | `TestCheckRequestEndpointPathValuesCorrect` + `TestCheckRequestEndpointRejectsMissingPathValues` in `runtime_boundary_test.go` | 8bdb7e6 |
| `镜像缺失` displayed but no real Docker check happened | Previous E2E used `image_present=true, docker_available=true` in enable/check payloads (client-trusted evidence). The check-request endpoint does real agent proxy but no test verified it with actual Docker images. No positive/negative Docker image case existed. | `scripts/e2e-runtime-config-web-check-flow.sh` with real Docker image positive (vllm/vllm-openai:latest, lmsysorg/sglang:latest, ghcr.io/ggml-org/llama.cpp:server-cuda13) and negative (lightai/nonexistent-image:e2e-missing) cases | This commit |
| UI shows "未知" for known error states | `translateStatus()` uses i18n key `status.{status}` but no `status.ready`, `status.missing_image`, `status.needs_check`, `status.unknown`, `status.unsupported_device`, `status.failed`, `status.template_only` keys existed. `STATUS_REASON_MAP` only had 4 entries, missing new check-request reason strings. | Added full i18n status matrix. Extended STATUS_REASON_MAP. | This commit |
| check-request reason says generic "node has no advertised address" | Server handler returns raw technical reason strings with no context (which image, which node). user needs actionable message. | check-request response now includes `checked_image_ref`; `missing_image` reason includes image name; new STATUS_REASON_MAP entries for agent-unreachable case | This commit |

## 2. Root Cause Analysis

### 2.1 Why `node_id and nbr_id are required` wasn't caught

Existing Go tests (`runtime_boundary_test.go`, `ui_persistence_runplan_test.go`) call handlers via `httptest.NewRecorder` and `newReq()`, which passes path params as a `map[string]string`. Tests constructed the correct params: `map[string]string{"id": nodeID, "nbr_id": nbrID}`. But the handler was reading `r.PathValue("node_id")` while the route parameter is `{id}`. The test accidentally worked around the bug because `newReq` passes path params by name — but the handler was reading a DIFFERENT path value name.

**Key fact:** `newReq` passes path params as `map[string]string{"id": nodeID}`. The handler reads `r.PathValue("id")` which matches. But the handler code in the first version of `HandleRequestNodeBackendRuntimeCheck` used `r.PathValue("node_id")` — this name doesn't match the route param `{id}` and never would have worked with a real router. The existing tests didn't hit this because no test called `HandleRequestNodeBackendRuntimeCheck` with the original code.

### 2.2 Why "镜像缺失" wasn't caught

Previous tests and E2E scripts:
- Used `image_present=true, docker_available=true` in enable/check payloads (client-trusted mock evidence)
- Never called the check-request endpoint with a real agent
- Never verified server/image matching against actual Docker image lists
- Never had a positive case (real image → ready) or negative case (nonexistent → missing_image)

The `e2e-model-runtime-wizard-nvidia-*.sh` scripts call the agent evidence `check` endpoint directly with `image_present=true`, bypassing the check-request flow entirely. This is correct for agent-simulated E2E but missed the UI path.

### 2.3 Why "未知" was shown

`translateStatus()` in `web/src/utils/status.ts` maps status via `t("status.${status}")`. The i18n files had no `status.ready`, `status.missing_image`, etc. entries. The fallback is returning the raw status string. When status is `"unknown"`, no i18n entry matches, so it returns `"unknown"` which Vue-i18n may or may not translate (it doesn't match `status_unknown` in deployments context). The RunnerConfigsPage wizard check alert showed `"unknown"` as the title.

## 3. Fixes Applied

### 3.1 Handler PathValue fix (8bdb7e6)
- `r.PathValue("node_id")` → `r.PathValue("id")` in `HandleRequestNodeBackendRuntimeCheck`

### 3.2 i18n status matrix (this commit)
- Added `status` namespace i18n keys for all NBR states: `ready`, `needs_check`, `missing_image`, `unknown`, `unsupported_device`, `failed`, `template_only`, `disabled`, `error`

### 3.3 STATUS_REASON_MAP extension (this commit)
- Added entries for: agent unreachable, Docker unavailable, image not found on node, node has no address

### 3.4 Real Docker E2E script (this commit)
- `scripts/e2e-runtime-config-web-check-flow.sh`: positive cases with real Docker images, negative case with nonexistent image, row check, wizard check, preflight, DryRun

### 3.5 Enhanced check-request response (this commit)
- Response now includes `checked_image_ref` for client display
- Missing image reason includes the image name

## 4. New Acceptance Criteria

Going forward, Runtime Config / Check / Deployment Wizard issues require:
1. **Level 1:** Handler tests (unit) — for state machine and validation
2. **Level 2:** Real API E2E — with real server, real node, real agent proxy, real Docker images
3. **Level 3:** Web-equivalent flow — verified API payloads match frontend button actions

This closeout confirms all three levels are covered for the check-request flow.
