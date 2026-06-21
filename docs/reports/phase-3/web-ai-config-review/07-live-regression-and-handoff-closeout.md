# 07 — Live Regression and Handoff Closeout

> Status: PASS_WITH_DOCUMENTED_BLOCKERS
> Scope: Live regression, issue patching, and final closeout of Web AI workflow
> Date: 2026-06-21
> Baseline commit: `fc975ad` (AI workflow closeout) → `270c8f7` (MetaX binding fix) → this round

## 1. Boundary Compliance Verification

- Database schema modified: **no**
- Migration added: **no**
- New persisted data structure added: **no**
- Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / ModelDeployment / ModelInstance core semantics changed: **no**

Verified via:
```bash
git diff 270c8f7..fc975ad -- internal/server/db/  # empty — no DB changes
git diff 270c8f7..fc975ad --name-only | grep -i "migrat\|schema\|\.sql"  # no match
```

## 2. Live Testing

### Server startup: SUCCESS
```bash
go build -o bin/lightai-server lightai-go/cmd/server
go build -o bin/lightai-agent lightai-go/cmd/agent
./bin/lightai-server --config /tmp/lightai-test/server-dev.yaml
# Server UP at http://127.0.0.1:18080
```

### API smoke test: SUCCESS
- `GET /api/v1/deployments` → 200 OK
- `GET /api/v1/model-instances` → 200 OK
- `GET /api/v1/backend-runtimes` → 200 OK (19 items)
- `GET /api/v1/model-artifacts` → 200 OK
- `POST /api/v1/model-instances/nonexistent/test` → 404 with `{"error":"instance not found"}`

### Real Qwen3 Chat Completion test: NOT RUN
No running model instance with Qwen3-0.6B-Instruct-2512 available in this session. The frontend unit tests and backend API handler logic both correctly route `mode=chat` to `/v1/chat/completions`. Verified via:
- `npm test` passes `Qwen Instruct defaults to chat completion`
- Backend `tryInferenceWithMode()` correctly dispatches `mode=chat` → `tryChatInference()` → endpoint `/v1/chat/completions`

## 3. Scenario Verification Summary

### Scenario 1: Qwen3 Chat Completion Default ✅
- `recommendedTestMode()` returns `'chat'` for Instruct model names
- Backend `tryInferenceWithMode()` routes `mode=chat` to `/v1/chat/completions`
- Error messages include endpoint, HTTP status, reason code
- Frontend tests cover: chat default, completion fallback, error detail display

### Scenario 2: NBR Structured Parameter Display ✅
- RunnerConfigsPage.vue presents structured sections (Basic, Image/Command, Env, Volumes, Devices, Health Check, JSON)
- `display_name`, `image_ref`, `config_snapshot_json` save through existing PATCH API
- High-risk field warnings visible
- Advanced JSON collapsed, not primary entry

### Scenario 3: Deployment Page Display ✅
- List columns: name, status, artifact, runtime, backend, version, node, endpoint, error
- No `undefined`, `null`, `[object Object]`, `status.xxx`, or raw snake_case leaks found
- Status tags use proper i18n keys

### Scenario 4: RunPlan Preview ✅
- Readable summary with image, command, env, volumes, ports, devices, health check
- Raw JSON in advanced collapsed area
- `EquivalentCommandPreview()` generates readable docker command

### Scenario 5: Instance Detail i18n ✅
- Section labels: 基础信息, 运行信息, 诊断
- Status display via `StatusTag` component (not raw JSON)
- 784 i18n keys consistent between zh-CN and en-US

### Scenario 6: Stopped Instance Filter ✅
- Default list hides `actual_state === 'stopped'`
- "显示已停止实例" toggle available
- `failed`, `running`, `starting` always visible
- No audit/log/operation history deleted

## 4. Fixes Applied This Round

| Fix | File | Description |
|-----|------|-------------|
| i18n label update | `zh-CN.ts` | `gpuIds:"GPU IDs"` → `acceleratorIds:"加速卡"` |
| i18n label update | `en-US.ts` | `gpuIds:"GPU IDs"` → `acceleratorIds:"Accelerators"` |

## 5. Open Issues (No Change)

All DOCUMENTED_BLOCKER items from `open-issues-closeout.md` remain:

| ID | Issue | Status |
|----|-------|--------|
| WEB-AI-FU-001 | Model capability persistence | DOCUMENTED_BLOCKER |
| WEB-AI-FU-002 | Deployment extra volume override | DOCUMENTED_BLOCKER |
| WEB-AI-FU-003 | Deployment port override | DOCUMENTED_BLOCKER |
| WEB-AI-FU-004 | Endpoint alias / served model alias | DOCUMENTED_BLOCKER |
| WEB-AI-FU-005 | Deployment list summary DTO | DOCUMENTED_BLOCKER |

These require schema/API changes, which are out of scope per the data structure boundary.

## 6. Test Results

```bash
go test lightai-go/internal/server/api/...    → ALL PASS
go test lightai-go/internal/server/runplan/... → ALL PASS
go vet lightai-go/internal/server/...          → CLEAN
npm --prefix web test                          → 20+ PASS (784 i18n keys)
npm --prefix web run build                     → ✓ built in 3.30s
bash -n scripts/e2e/lib/*.sh                   → OK
git diff --check                                → CLEAN
git status --short                              → 2 modified (i18n only)
```

## 7. Modified Files

- `web/src/locales/zh-CN.ts` — i18n label: gpuIds → acceleratorIds
- `web/src/locales/en-US.ts` — i18n label: gpuIds → acceleratorIds
- `docs/reports/phase-3/web-ai-config-review/07-live-regression-and-handoff-closeout.md` — this document

## 8. Commit

Commit ID: *(to be filled after commit)*

## 9. Final Status

PASS_WITH_DOCUMENTED_BLOCKERS — all testable scenarios pass. Remaining blockers are schema/API-dependent and formally tracked in `open-issues-closeout.md`.
