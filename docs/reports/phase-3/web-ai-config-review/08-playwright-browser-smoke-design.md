# 08 — Playwright Browser Smoke Test Design

> Status: DESIGN_READY_FOR_IMPLEMENTATION
> Scope: Playwright-based browser automation smoke tests for Web AI pages
> Date: 2026-06-21
>
> 本文件仅为 Playwright 浏览器 smoke tests 设计文档。
> 本轮没有新增 Playwright 测试代码。
> 本轮没有修改数据库结构。
> 本轮没有新增 migration。

## 1. Background and Goals

### Why Browser Automation

Previous Web AI development rounds revealed recurring issues:

- **i18n key leaks** — raw `status.running`, `deployments.title` appearing in UI
- **Display artifacts** — `undefined`, `null`, `[object Object]` in rendered content
- **Test mode mismatch** — frontend recommended mode vs backend endpoint (e.g. Completion vs Chat Completion)
- **NBR regression** — structured parameter editor potentially regressing to raw JSON primary entry
- **Stopped filter** — default visibility logic needing live page verification
- **Manual testing cost** — full UI click-through is time-consuming and error-prone

### Goals

- **NOT** a replacement for API E2E or real container smoke tests
- **IS** a stable, fast, repeatable smoke layer for Web pages
- Catch display errors that unit tests cannot see (DOM content, i18n leaks)
- Verify page-level interactions (navigation, modal dialogs, form defaults, button availability)
- Run without real server, Docker, GPU, or model containers

## 2. Test Layering

```text
┌─────────────────────────────────────────────┐
│ Layer 1: Playwright UI Smoke + Mocked API    │  ← THIS ROUND (next phase)
│  - No real server/agent/Docker/model         │
│  - page.route() mock all API responses       │
│  - Verify: rendering, navigation, defaults,  │
│    payload shape, stopped filter, i18n leaks │
├─────────────────────────────────────────────┤
│ Layer 2: UI + Real Server API                │  ← Future
│  - Start real server, no model containers    │
│  - Verify: page ↔ real API DTO compatibility │
├─────────────────────────────────────────────┤
│ Layer 3: Real Runtime E2E                    │  ← Future (API/script)
│  - server + agent + Docker + model instance  │
│  - /v1/chat/completions real inference       │
│  - Can be API/script, not necessarily browser│
└─────────────────────────────────────────────┘
```

**Priority**: Next phase implements Layer 1 only.

## 3. Chrome Strategy

```text
System Chrome binary: /usr/bin/google-chrome
Version:              Google Chrome 149.0.7827.114
Playwright version:   1.61.0 (installed as @playwright/test in web/)

Playwright config MUST use:
  channel: "chrome"
  headless: true

MUST NOT:
  - Download bundled Chromium
  - Execute playwright install chromium
```

If sandbox issues arise in WSL/Linux:

```ts
// web/playwright.config.ts
export default defineConfig({
  use: {
    channel: 'chrome',
    headless: true,
    launchOptions: {
      args: ['--no-sandbox'],
    },
  },
})
```

`channel: "chrome"` is mandatory — do not fall back to bundled Chromium.

## 4. Suggested File Structure

After this design round, implementation should add:

```text
web/playwright.config.ts          ← Playwright configuration
web/e2e/web-ai-smoke.spec.ts      ← Web AI smoke scenarios
```

`package.json` additions:

```json
{
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:headed": "playwright test --headed",
    "test:e2e:ui": "playwright test --ui"
  }
}
```

## 5. Mock API Strategy

Layer 1 tests use `page.route()` to mock ALL API responses. No real server.

### Principles

- **Minimal but representative** — mock data covers Qwen3, NBR, deployment, running/failed/stopped instances
- **No real login** — mock `POST /api/v1/auth/login` to return session + CSRF token
- **No real server** — all `/api/v1/*` routes are mocked
- **No real model** — no `/v1/models`, `/v1/chat/completions` calls in Layer 1

### Mock Data Requirements

1. **Auth** — `POST /api/v1/auth/login` → `{csrf_token, user}` + Set-Cookie
2. **Model artifacts** — at least one HF model (Qwen3-0.6B-Instruct-2512) with task_type, format, metadata
3. **Backends** — vLLM, SGLang, llama.cpp entries
4. **Backend versions** — per backend, with capabilities_json
5. **Backend runtimes** — NVIDIA and MetaX templates, with docker_json, model_mount_json
6. **Node backend runtimes** — ready_with_warnings status, structured config_snapshot_json with image/args/env/volumes/ports/devices/privileged/ipc/shm_size/ulimits/health_check
7. **Deployments** — at least one deployment with full placement_json, service_json, backend info
8. **Instances** — three: running(Qwen3), failed(other), stopped(old)
9. **GPUs** — at least one NVIDIA GPU with index_num=0

### Route Setup Pattern

```ts
// Example pattern — NOT implementation, just design reference
await page.route('**/api/v1/model-artifacts*', (route) => {
  route.fulfill({ status: 200, body: JSON.stringify(mockArtifacts) })
})
```

## 6. Smoke Scenarios (6 required)

### Scenario 1: Navigation Structure

**What to verify**:

- Sidebar shows "模型运行" section
- Under 模型运行: 模型库, 运行配置, 模型部署, 模型实例, 测试与诊断 (if page exists)
- Sidebar shows "配置" section
- Under 配置: 推理后端, 运行模板
- Clicking each entry navigates to the correct route

**Assertions**:

- No `undefined`, `null`, `[object Object]` in sidebar text
- All navigation labels are fully translated (no `nav.xxx` keys)

### Scenario 2: Model Capability Display

**Mock**: Qwen3-0.6B-Instruct-2512 model artifact with hf metadata

**Verify**:

- Model library shows the model
- Capability section infers or displays "Chat / 对话"
- Recommended test mode shows "Chat Completion"
- No `undefined`, `null`, `[object Object]`, raw JSON in capability display

**Anti-checks**:

- Must NOT contain: capabilities raw JSON, `backend_name`, `gpu_ids`

### Scenario 3: Instance Test Default = Chat

**Mock**: Running instance for Qwen3-0.6B-Instruct-2512

**Verify**:

- Instance detail → test dialog default type is "Chat Completion"
- Click "执行测试" sends `POST /api/v1/model-instances/<id>/test`
- Request body contains `"mode":"chat"`
- Success response → page shows success (not raw JSON error)
- Failure response → page shows reason code with i18n text

**Payload assertion** (intercept request):

```ts
const testReq = page.waitForRequest(req =>
  req.url().includes('/model-instances/') && req.url().endsWith('/test')
)
// Verify JSON body has mode: "chat"
```

### Scenario 4: NBR Structured Runtime Parameters

**Mock**: NBR with config_snapshot_json containing image, args, env, volumes, ports, devices, privileged, ipc, shm_size, ulimits, health_check

**Verify**:

- Page shows structured sections (NOT raw JSON as primary):
  - 镜像与命令
  - 环境变量
  - 卷映射
  - 端口
  - 设备与权限
  - 健康检查
  - 高级诊断 (collapsed)
- Value fields are populated from mock data
- "高级诊断 JSON" is collapsed by default

**Anti-checks**:

- Must NOT show raw `config_snapshot_json` as main body
- Must NOT show i18n key leaks in section labels

### Scenario 5: Deployment Page Display

**Mock**: Deployment with full context (model, backend, version, runtime, node, GPU, endpoint, status)

**Verify**:

- List row shows: name, status tag, model, backend, version, runtime, image, node, endpoint
- No `undefined`, `null`, `[object Object]`, `backend_name`, `status.running` in any cell
- Status tags use translated text, not raw internal status strings
- "加速卡" / "Accelerators" label (NOT "GPU IDs")

### Scenario 6: Stopped Instance Filter

**Mock**: 3 instances: running(Qwen3), failed(error-model), stopped(old-deploy)

**Verify**:

- Default view: shows running and failed, does NOT show stopped
- Toggle "显示已停止实例" / "Show stopped instances" → stopped instance appears
- Toggle off → stopped instance hidden again
- `failed` instance always visible (never filtered out)

## 7. Page Routes and Selector Strategy

### Routes

Current routes (`web/src/router/index.ts`):

| Route | Page |
|-------|------|
| `/models/artifacts` | ModelArtifactsPage |
| `/models/deployments` | ModelDeploymentsPage |
| `/models/instances` | ModelInstancesPage |
| `/runner-configs` | RunnerConfigsPage |
| `/backends` | BackendsPage |
| `/runtimes` | BackendRuntimesPage |
| `/models/test-diagnostics` | TestDiagnosticsPage (if present) |

Implementation MUST read actual `router/index.ts` and `ConsoleLayout.vue` for exact paths before writing test locators.

### Selector Strategy

1. **Preferred**: `page.getByRole()`, `page.getByText()`, `page.getByLabel()`
2. **Fallback**: `page.locator('[data-testid="..."]')` — only added when no stable semantic selector exists
3. **`data-testid` rules**:
   - Only for test stability, never changes UI behavior
   - Namespace: `testid` prefix for avoid collision
   - Examples: `data-testid="nbr-section-devices"`, `data-testid="instance-filter-stopped"`
4. **Avoid**:
   - CSS deep selectors (`.el-table__body tr:nth-child(3) .cell`)
   - Fixed `sleep()` / `waitForTimeout()`
   - Use `expect(...).toBeVisible()`, `waitFor()` with locator

## 8. i18n Leakage Assertions

Every smoke scenario MUST include a page-level assertion:

```ts
// Generic i18n leak check — adapt to actual page content extraction
const pageText = await page.content()
const leaks = [
  'undefined', 'null', '[object Object]',
  'status.', 'nav.', 'backend_name', 'gpu_ids', 'GPU IDs',
]
for (const leak of leaks) {
  expect(pageText).not.toContain(leak)
}
```

Note: `GPU IDs` was corrected to `Accelerators` / `加速卡` in a previous round. Playwright smoke must cover this regression point.

## 9. What NOT To Do (Layer 1)

The first Playwright smoke implementation MUST NOT:

1. Start real Docker model containers
2. Verify real Qwen3 Chat Completion response content
3. Verify NVIDIA/MetaX real hardware
4. Add new database schema
5. Add new deployment override data structures
6. Implement full user permission matrix
7. Introduce complex Playwright project matrix (multi-browser)
8. Do screenshot baseline comparison
9. Connect to CI (if project has no CI yet)

## 10. Implementation Plan (Next Phase)

When implementation begins:

```text
Phase A: Create web/playwright.config.ts
  - channel: "chrome", headless: true
  - Test directory: ./e2e
  - No bundled Chromium

Phase B: Add test:e2e scripts to web/package.json
  - "test:e2e": "playwright test"
  - "test:e2e:headed": "playwright test --headed"

Phase C: Create web/e2e/web-ai-smoke.spec.ts
  - Use page.route() for all API mocking
  - Implement 6 smoke scenarios

Phase D: Add data-testid attributes where needed
  - Only where semantic selectors are insufficient
  - Do not change UI appearance

Phase E: Run and fix
  - npm --prefix web run test:e2e
  - Fix failures, missing i18n keys, leaked raw text
  - Iterate until all 6 scenarios pass

Phase F: Update closeout document
  - Record results, evidence, remaining gaps
  - Commit and push
```

## 11. Acceptance Commands (Future)

After implementation, these commands must all pass:

```bash
npm --prefix web test               # existing unit/i18n tests
npm --prefix web run build          # production build
npm --prefix web run test:e2e       # Playwright Layer 1 smoke
go test ./internal/server/api/...   # backend API tests
go test ./internal/server/runplan/... # RunPlan resolver tests
go vet ./...                         # Go static analysis
git diff --check                     # whitespace check
git status --short                   # must be clean
```

## 12. File Inventory

| Phase | File | Purpose |
|-------|------|---------|
| This round | `08-playwright-browser-smoke-design.md` | Design document (this file) |
| Next phase | `web/playwright.config.ts` | Playwright config (channel:chrome) |
| Next phase | `web/e2e/web-ai-smoke.spec.ts` | 6 browser smoke scenarios |
| Already present | `web/package.json` | Already has @playwright/test |
| Already present | `web/package-lock.json` | Already has dependency lock |
