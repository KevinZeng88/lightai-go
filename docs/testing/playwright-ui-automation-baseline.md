# LightAI Go Playwright UI 自动化基线记录

> 适用项目：`/home/kzeng/projects/ai-platform-study/lightai-go`  
> Web 目录：`/home/kzeng/projects/ai-platform-study/lightai-go/web`  
> 记录时间：2026-06-27  
> 目标：沉淀本轮 Playwright UI 自动化基线搭建、登录自动化、Vite Origin 修复、已验证命令和后续约束。

---

## 1. 背景

本轮工作的直接起因是运行模板、用户运行配置、参数编辑、复制保存等页面问题反复通过人工点击验证，成本高、易遗漏、易回归。

已知手工发现的问题包括：

1. 运行模板参数仍显示英文或裸内部字段，例如：
   - `Model mount`
   - `Environment variables`
   - `Kind`
   - `Ports`
   - `Volumes`
   - `Devices`
   - `Extra env`
   - `Backend extra args`
2. 部分参数默认启用状态不合理。
3. 部分字段看起来重复，例如 `devices`。
4. 复制用户配置时，创建页面输入了新名称，但保存后名称变回被复制配置的名称。
5. 复制用户配置后，修改参数值可以保存，但修改 enabled / 是否打开无法保存，保存后恢复原样。

因此，本轮先建立 UI 自动化基础设施，使后续问题可以通过 Playwright 自动复现、自动回归，并作为 Claude/Codex 或人工修复时的验收依据。

---

## 2. 本地运行环境

### 2.1 项目路径

```bash
/home/kzeng/projects/ai-platform-study/lightai-go
```

### 2.2 Web 目录

```bash
/home/kzeng/projects/ai-platform-study/lightai-go/web
```

### 2.3 端口

```text
Backend Server:  http://127.0.0.1:18080
Web / Vite:      http://127.0.0.1:15173
Agent:           http://127.0.0.1:19091
Prometheus:      http://127.0.0.1:19090
Grafana:         http://127.0.0.1:13000
```

### 2.4 Chrome / Playwright

已确认本机 Chrome：

```bash
/usr/bin/google-chrome-stable
```

已确认 Playwright 可用。曾出现 `@playwright/test` 与 `playwright` 小版本不完全一致的问题，建议保持二者版本一致。

### 2.5 Playwright 输出目录

```bash
/tmp/lightai/e2e/playwright/results
/tmp/lightai/e2e/playwright/report
```

失败时保留：

- screenshot
- video
- trace
- error context

---

## 3. 已完成工作摘要

本轮已经完成以下基础能力：

1. 建立 Playwright 配置基线。
2. 固定 Vite 端口为 `15173`，避免端口漂移。
3. 配置本机 Chrome，并禁用 GPU 参数，规避 WSL2 / Xserver / MobaXterm 下 Chrome GPU 问题。
4. 新增无登录前端 smoke：`app-load.spec.ts`。
5. 新增全栈连通 smoke：`fullstack-health.spec.ts`。
6. 新增登录 helper、global setup、登录 smoke。
7. 修复 Vite proxy 转发 Origin，解决后端 `invalid origin`。
8. 验证 clean DB 后默认管理员登录与首次改密码流程。
9. 验证登录态 storageState 可生成并复用。
10. 跑通三项基线测试：
    - `app-load.spec.ts`
    - `fullstack-health.spec.ts`
    - `login.spec.ts`

---

## 4. 关键文件

### 4.1 Playwright 配置

```text
web/playwright.config.ts
```

关键要求：

```ts
testDir: './tests/e2e'
outputDir: '/tmp/lightai/e2e/playwright/results'
workers: 1
retries: 0
fullyParallel: false
```

Chrome 项目应使用本机 Chrome：

```ts
const chromeExecutablePath = process.env.LIGHTAI_CHROME_EXECUTABLE || undefined
```

Chrome 启动参数应包含：

```ts
args: [
  '--disable-gpu',
  '--disable-gpu-compositing',
  '--disable-gpu-rasterization',
  '--disable-dev-shm-usage',
  '--no-first-run',
]
```

Web Server 应固定端口：

```ts
command: 'npm run dev -- --host 127.0.0.1 --port 15173 --strictPort'
```

### 4.2 Vite 配置

```text
web/vite.config.ts
```

本轮必须修复 Vite proxy Origin。推荐配置：

```ts
const backendTarget = 'http://127.0.0.1:18080'

const backendProxy = {
  target: backendTarget,
  changeOrigin: true,
  configure: (proxy: any) => {
    proxy.on('proxyReq', (proxyReq: any) => {
      proxyReq.removeHeader('origin')
      proxyReq.setHeader('Origin', backendTarget)
    })
  },
}
```

并应用到：

```ts
proxy: {
  '/api': backendProxy,
  '/metrics': backendProxy,
  '/healthz': backendProxy,
}
```

### 4.3 NPM Scripts

```text
web/package.json
```

应包含：

```json
{
  "test:e2e": "playwright test",
  "test:e2e:noauth": "LIGHTAI_SKIP_AUTH_SETUP=1 playwright test",
  "test:e2e:headed": "playwright test --headed",
  "test:e2e:ui": "playwright test --ui",
  "test:e2e:debug": "playwright test --debug",
  "test:e2e:report": "playwright show-report /tmp/lightai/e2e/playwright/report"
}
```

### 4.4 登录自动化文件

```text
web/tests/e2e/helpers/auth.ts
web/tests/e2e/global.setup.ts
web/tests/e2e/auth/login.spec.ts
web/tests/e2e/auth/login-debug.spec.ts
web/tests/e2e/.auth/admin.json
```

其中：

```text
web/tests/e2e/.auth/admin.json
```

是本地生成的登录态文件，不允许提交。

`.gitignore` 应包含：

```gitignore
tests/e2e/.auth/
```

---

## 5. 已验证测试

### 5.1 前端无后端登录 smoke

命令：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/app-load.spec.ts
```

结果：

```text
1 passed
```

目标：

- Vite 可启动。
- 页面可打开。
- `body` 与 `#app` 可见。
- 页面不是空白页。
- 不要求 Backend 登录。

### 5.2 全栈连通 smoke

命令：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/smoke/fullstack-health.spec.ts
```

结果：

```text
1 passed
```

目标：

- Backend `18080` 可达。
- 未登录访问 `/api/v1/auth/me` 返回 `401 unauthorized` 是正常状态。
- 不应出现 `500 / 502 / 503 / 504`。

### 5.3 登录 smoke

命令：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web
npm run test:e2e -- --project=chrome-local tests/e2e/auth/login.spec.ts
```

结果：

```text
1 passed
```

目标：

- global setup 能完成管理员登录。
- 如果遇到首次登录强制改密码，能够完成改密并重新登录。
- 生成并复用：

```text
web/tests/e2e/.auth/admin.json
```

- 浏览器上下文中执行：

```ts
fetch('/api/v1/auth/me', { credentials: 'include' })
```

返回 `200`。

---

## 6. 环境变量约定

当前 shell 已设置环境变量，后续命令不要重复写入。

需要存在：

```bash
LIGHTAI_CHROME_EXECUTABLE
LIGHTAI_WEB_URL
LIGHTAI_E2E_ADMIN_USERNAME
LIGHTAI_E2E_ADMIN_PASSWORD
LIGHTAI_E2E_ADMIN_NEW_PASSWORD
```

可以用以下命令确认，但不要在脚本中打印明文密码：

```bash
echo "$LIGHTAI_CHROME_EXECUTABLE"
echo "$LIGHTAI_WEB_URL"
echo "$LIGHTAI_E2E_ADMIN_USERNAME"
echo "$LIGHTAI_E2E_ADMIN_PASSWORD" | sed 's/./*/g'
echo "$LIGHTAI_E2E_ADMIN_NEW_PASSWORD" | sed 's/./*/g'
```

---

## 7. Clean DB 注意事项

本轮曾遇到旧 DB schema 残留导致 Server 启动失败：

```text
seed backend runtime runtime.llamacpp.cpu-docker: table backend_runtimes has no column named visibility
```

处理原则：

- 本项目当前阶段不做旧 DB 兼容。
- 本地开发允许删除 `data/lightai.db` 重建。
- 如果 clean DB 仍失败，说明 migration / seed / DAO 字段不一致，必须修最终 schema。

清理方式：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
./scripts/stop-all.sh || true
mkdir -p /tmp/lightai/backups/db
cp -a data/lightai.db "/tmp/lightai/backups/db/lightai.$(date +%Y%m%d-%H%M%S).db" 2>/dev/null || true
rm -f data/lightai.db data/lightai.db-wal data/lightai.db-shm
./scripts/start-all.sh
```

Server 启动后，未登录访问：

```bash
curl -i http://127.0.0.1:18080/api/v1/auth/me
```

返回 `401 unauthorized` 属于正常状态。

---

## 8. 当前基线结论

当前 UI 自动化地基已具备：

```text
Playwright + Chrome 可用
Vite 15173 可用
Backend 18080 可用
Clean DB 启动可用
Vite proxy Origin 修复可用
默认管理员登录可用
首次改密码流程已验证
storageState 可生成并复用
```

后续业务 UI 自动化可以在此基础上继续扩展。
