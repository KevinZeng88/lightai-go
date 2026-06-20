# NBR Image Probe — Integration Validation Plan

> Status: DRAFT
> Date: 2026-06-20
> Scope: Verification of currently implemented NBR Image Probe chain (Phase 0–4)
> NOT: New feature design, E2E framework, version/script probe, catalog match

## 1. Validation Goals

This plan verifies the currently implemented NBR Image Probe chain (commit `7149bcb` and prior). The goal is to detect these failure modes before manual Web testing:

| # | Failure Mode | Detection Layer |
|---|-------------|-----------------|
| F1 | Image in `/docker-images` list but probe/check reports `missing_image` | L1, L3 |
| F2 | ImageInspect succeeds but status maps to `missing_image` | L1 |
| F3 | ImageInspect error (not "not found") maps to `missing_image` | L1 |
| F4 | POST /probe diverges from old check-request behavior | L1 |
| F5 | GET /probe cannot read freshly stored `probe_results_json` | L1, L3 |
| F6 | `refresh()` drops `probe_results_json` from list items | L2 |
| F7 | Detail drawer shows no probe information | L2, L4 |
| F8 | i18n key leaks (raw key displayed instead of translation) | L2, L4 |
| F9 | `ready_with_warnings` / `declared_match_unverified` / `inspect_failed` display incorrect tag type | L2, L4 |
| F10 | Route PathValue `id`/`nbr_id` mismatch | L1 |

## 2. Validation Layers

### Layer 1: Backend Unit + Route Tests

**Command**: `go test ./internal/server/api/... -count=1 -v` (from repo root)

**Coverage**:

| Test | Verifies |
|------|----------|
| `TestProbeEndpointPathValuesCorrect` | POST/GET /probe route `{id}` → `PathValue("id")`, `{nbr_id}` → `PathValue("nbr_id")` |
| `TestProbeEndpointRejectsMissingPathValues` | Missing path params → 400 |
| `TestCheckRequestBackwardCompatible` | Old check-request still works, all response fields present |
| `TestGetProbeReturnsEmptyWhenNeverProbed` | GET /probe on unprobed NBR → 200 with `{}` |
| `TestGetProbeReturnsSnapshotAfterProbe` | POST /probe → GET /probe → snapshot round-trip |
| `TestPostProbeMissingImageOnlyFromInspectNotFound` | ImageInspect success → NOT `missing_image` (regression) |
| `TestCheckRequestImageExistsSuccess` | Image in list → `image_present=true`, status not `missing_image` |
| `TestCheckRequestImageMissing` | ImageInspect "no such image" → `missing_image` |
| `TestCheckRequestAgentUnreachable` | Agent down → `agent_unreachable`, NOT `missing_image` |
| `TestCheckRequestListMissesInspectFound` | List misses, Inspect succeeds → NOT `missing_image` |
| `TestCheckRequestInspectNotFound` | Inspect "no such image" → `missing_image` |
| `TestCheckRequestInspectErrorNotNotFound` | Inspect error (not "not found") → `inspect_failed` |
| `TestCheckRequestEvidenceMissing` | No agent → `agent_unreachable`, NOT `missing_image` |
| `TestCheckRequestStatusNotMissingImage` | Agent 500 → NOT `missing_image` |
| `TestCheckRequestAllBackendImageFormats` | vllm/sglang/llamacpp formats all match |
| `TestCheckRequestProbeResultsStored` | probe_results_json persisted to DB |
| `TestCheckRequestEndpointPathValuesCorrect` | PathValue regression |
| `TestCheckRequestEndpointRejectsMissingPathValues` | Missing path params → 400 |

**Problems covered**: F1, F2, F3, F4, F5, F10  
**Problems NOT covered**: F6, F7, F8, F9 (frontend-only)

### Layer 2: Frontend Static + Component-Level Verification

**Command**: `npm run build && npm test`

**Coverage**:

| Check | Verifies |
|-------|----------|
| `npm run build` | No TypeScript/Vue compilation errors |
| i18n key consistency | zh-CN ↔ en-US key count match (782) |
| i18n key resolution | All 630+ `$t()` references resolve to strings, no `[object Object]` |
| No hardcoded credentials | Security audit |
| Runner config page exposes create/edit/check actions | Component presence |
| Runner config wizard displays selected image | Wizard state management |
| Runtime wizard exposes name inputs | Form controls present |
| status.ts mapping | `getStatusType()` maps all 9+ NBR statuses |

**Explicitly checks via regex patterns in test scripts**:
- No "刚刚" / "just now" in output (time formatting correctness)
- zh-CN cross-year formats as YYYY-MM-DD

**Problems covered**: F8 (i18n resolution — key leaks detected by static reference check).  
**Problems NOT fully covered at L2**: F6 — `npm build` + `npm test` cover compilation and i18n static references; whether the detail drawer actually renders probe panels and `probe_results_json` data correctly still requires L4 Web smoke for final confirmation. F7, F9 also require L4 visual verification.

### Layer 3: Real API Smoke

**Prerequisites**:
- Agent binary built and running on `127.0.0.1:19091`
- `vllm/vllm-openai:latest` present on the local Docker daemon
- Server binary built and running on `127.0.0.1:18080`
- Valid session cookie + CSRF token for server API calls

**Smoke commands** (manual or script-driven):

```bash
# A. Agent endpoints (no auth required)
AGENT="http://127.0.0.1:19091"

# A1: Docker image list includes target image
curl -s "$AGENT/docker-images?search=vllm&limit=5" | python3 -c "
import sys,json; d=json.load(sys.stdin)
found = any('vllm/vllm-openai' in i.get('image_ref','') for i in d.get('images',[]))
print('PASS: vllm in list' if found else 'FAIL: vllm not in list')
"

# A2: Docker image inspect succeeds
curl -s "$AGENT/docker-image-inspect?ref=vllm/vllm-openai:latest" | python3 -c "
import sys,json; d=json.load(sys.stdin)
ok = 'inspect' in d and 'Id' in d.get('inspect',{})
print('PASS: inspect success' if ok else f'FAIL: inspect error={d.get(\"error\",\"?\")}')
"

# A3: Non-existent image inspect returns error (not success)
curl -s "$AGENT/docker-image-inspect?ref=not-exist/lightai-test:missing" | python3 -c "
import sys,json; d=json.load(sys.stdin)
not_ok = 'inspect' not in d or not d['inspect']
print('PASS: missing image returns error' if not_ok else 'FAIL: missing image inspect succeeded')
"

# B. Server endpoints (require auth — see note below)
# These require a valid session. The test suite covers these via httptest.
# For real smoke, the auth must be solved first (see smoke script discussion).
```

**Server API calls** (for reference, require auth):
```bash
# POST /probe
curl -s -X POST "$SERVER/api/v1/nodes/$NODE_ID/backend-runtimes/$NBR_ID/probe" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" -b cookies.txt \
  -d '{}'

# GET /probe
curl -s "$SERVER/api/v1/nodes/$NODE_ID/backend-runtimes/$NBR_ID/probe" \
  -H "X-CSRF-Token: $CSRF" -b cookies.txt
```

**Problems covered**: F1 (real agent), F2 (real agent), F3 (real agent), F5 (real DB)  
**Problems NOT covered**: F6, F7, F8, F9 (require browser)

### Layer 4: Minimal Web Smoke

**Method**: Manual, semi-automated. Open browser, perform these steps:

| Step | Action | Expected |
|------|--------|----------|
| 1 | Open Runner Configs page | Page loads, list shows existing NBRs |
| 2 | Click "新增节点运行配置" | Wizard opens at step 0 |
| 3 | Step 1: Select a template (vLLM) | Template selected, auto-advance |
| 4 | Step 2: Select a node | Node selected, auto-advance |
| 5 | Step 3: Click refresh in DockerImagePicker, select `vllm/vllm-openai:latest` | Image selected, shown below picker |
| 6 | Step 4: Click "检测" (Check) | Alert shows: status is `ready` or `ready_with_warnings` (green/orange). NOT "镜像缺失" |
| 7 | Check probe collapsible panels | Image Metadata shows Image ID, RepoTags, Entrypoint, Size. Backend Match shows match detail. Version Probe shows "未探测" |
| 8 | Check no i18n key leaks | All labels are Chinese text (e.g., "镜像 ID", "入口点", "仓库标签"), not raw keys like "nodeRuntimeProbe.imageId" |
| 9 | Click "创建" (Create) | Success message. Wizard closes. |
| 10 | Click "详情" (Detail) on the new row | Drawer opens |
| 11 | Verify drawer shows probe data | Same collapsible panels as wizard step 4 show probe data |
| 12 | Verify diagnostic notices | If entrypoint is not bash/sh, no shell wrapper notice. If backend_match_status is "declared_match_unverified", vendor image notice appears |
| 13 | Verify Run Parameters section | Shows image_name, vendor from config_snapshot_json |
| 14 | Verify status tag | Row in list shows correct tag color: green for ready, orange for ready_with_warnings |
| 15 | Test non-existent image | Enter `not-exist/lightai-test:missing` in manual input, click Check → should show "镜像缺失" (red) |
| 16 | Click "重新检测" on an existing NBR row | Success message, list refreshes |

**Problems covered**: F6, F7, F8, F9 (all frontend)  
**Problems NOT covered**: F1–F5, F10 (backend, covered by L1+L3)

## 3. Smoke Script Recommendation

### Design

File: `scripts/e2e-nbr-image-probe-smoke.sh`

```
Purpose: Verify Agent Docker endpoints + Server probe API without starting a model container.
Scope:   Agent /docker-images, /docker-image-inspect, Server POST/GET /probe.
Does NOT: Start containers, load models, require GPU, modify DB records.
```

**Flow**:

```
1. Check prerequisites (docker, curl, jq, python3)
2. Verify Agent is reachable (GET /healthz)
3. Verify Agent /docker-images lists target image_ref
4. Verify Agent /docker-image-inspect succeeds for target image_ref
5. Verify Agent /docker-image-inspect returns error for non-existent image_ref
6. [Requires auth] Verify Server POST /probe succeeds
7. [Requires auth] Verify Server GET /probe returns snapshot
8. [Requires auth] Verify non-existent image → missing_image only
9. Output PASS/FAIL summary
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENT_URL` | `http://127.0.0.1:19091` | Agent metrics HTTP address |
| `SERVER_URL` | `http://127.0.0.1:18080` | Server API address |
| `TARGET_IMAGE` | `vllm/vllm-openai:latest` | Image to verify exists |
| `MISSING_IMAGE` | `not-exist/lightai-test:missing` | Image that must not exist |
| `NODE_ID` | (required) | Node ID for probe API calls |
| `NBR_ID` | (required) | NBR ID for probe API calls |
| `SESSION_COOKIE` | (optional) | Server auth session cookie file |
| `CSRF_TOKEN` | (optional) | Server CSRF token |

### Auth Dependency

Steps 6–8 require valid server authentication (session cookie + CSRF token). Options:

1. **If `SESSION_COOKIE` and `CSRF_TOKEN` are provided**: Script runs all 8 steps.
2. **If not provided**: Script runs only steps 1–5 (Agent-only) and skips 6–8 with a note: "Server steps skipped: set SESSION_COOKIE and CSRF_TOKEN to enable."

### Implementation Cost

Low (~80 lines of bash). Steps 1–5 are straightforward `curl | python3 -c` one-liners. Steps 6–8 require login or pre-obtained tokens.

### Recommendation

**Defer implementation** until auth handling is resolved. The current test suite (Layer 1) covers all server probe logic. Layer 3 manual commands cover Agent endpoints. The smoke script adds convenience but is not blocking.

## 4. Acceptance Criteria (Current Phase)

| Check | Method | Status |
|-------|--------|--------|
| `go test ./...` PASS | Automated | ✅ Covered (17 probe-related tests) |
| `go build ./...` PASS | Automated | ✅ CI-ready |
| `go vet ./...` PASS | Automated | ✅ CI-ready |
| `npm run build` PASS | Automated | ✅ CI-ready |
| `npm test` PASS | Automated | ✅ CI-ready (782 keys, no leaks) |
| Agent `/docker-image-inspect` succeeds for real image | Manual L3 | ⬜ Requires running agent |
| POST /probe succeeds and writes probe_results_json | Manual L3 | ⬜ Requires auth + running server |
| GET /probe reads snapshot | Manual L3 | ⬜ Requires auth + running server |
| `missing_image` only for truly absent image | L1 + L3 | ✅ L1 covers, L3 confirms real env |
| Detail drawer shows probe info | Manual L4 | ⬜ Requires browser |
| `git status --short` shows only committed or explicitly tracked changes | Automated | ✅ On each commit; any untracked or unexpected changes must be reviewed before commit |

## 5. What Tests Currently Cover

### Problems Detectable Without Manual Testing

| Problem | Detection |
|---------|-----------|
| Field name mismatch (root cause of Phase 0) | L1: `TestCheckRequestImageExistsSuccess` |
| ImageInspect success → `missing_image` | L1: `TestPostProbeMissingImageOnlyFromInspectNotFound` |
| List miss + Inspect success → `missing_image` | L1: `TestCheckRequestListMissesInspectFound` |
| Inspect error (not "not found") → `missing_image` | L1: `TestCheckRequestInspectErrorNotNotFound` |
| Agent unreachable → `missing_image` | L1: `TestCheckRequestAgentUnreachable`, `TestCheckRequestEvidenceMissing` |
| Agent 500 → `missing_image` | L1: `TestCheckRequestStatusNotMissingImage` |
| PathValue parameter mismatch | L1: `TestProbeEndpointPathValuesCorrect`, `TestCheckRequestEndpointPathValuesCorrect` |
| check-request backward compat broken | L1: `TestCheckRequestBackwardCompatible` |
| GET /probe returns nothing | L1: `TestGetProbeReturnsSnapshotAfterProbe` |
| probe_results_json not persisted | L1: `TestCheckRequestProbeResultsStored` |
| i18n key leaks | L2: i18n resolution check |
| i18n key count mismatch | L2: zh-CN ↔ en-US consistency check |
| TypeScript/Vue compile errors | L2: `npm run build` |
| `refresh()` drops probe_results_json | L2: TypeScript compilation verifies field access is valid (build passes); L4 Web smoke confirms data renders correctly in drawer |

### Problems Still Requiring Manual Web Smoke

| Problem | Why |
|---------|-----|
| Detail drawer probe panels actually render | No browser automation framework |
| Diagnostic notices appear correctly | Shell wrapper / vendor image / blocking error conditions vary by image |
| Run Parameters section displays correct data | Depends on actual `config_snapshot_json` content |
| Status tag color renders correctly in list | Visual check; `getStatusType()` logic is covered but DOM output is not |
| Non-existent image manual input → missing_image in wizard | Requires manual interaction with wizard UI |

## 6. Boundaries

This validation plan is explicitly NOT:

- A complete E2E testing framework
- A new feature development plan
- A Start Wizard integration design
- A script/version probe implementation plan
- A catalog match implementation plan
- An independent probe table design

## 7. Recommended Next Steps

1. **Run Layer 1 + 2** on every commit (already covered by `go test` + `npm test`)
2. **Run Layer 3** (Agent endpoints) before any release — takes 30 seconds
3. **Run Layer 4** (Web smoke) before any release — takes 5 minutes manually
4. **Implement smoke script** (Layer 5) when auth handling is simplified or token injection is available
5. **No new backend features** until Phases 5+ are explicitly approved
