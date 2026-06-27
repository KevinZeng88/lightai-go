# LightAI Go Playwright 登录与 Origin 问题处理手册

> 适用范围：LightAI Go Web UI 自动化、Vite Dev Server、Backend Origin / CSRF 校验、Playwright 登录态复用。

---

## 1. 问题背景

Playwright 登录自动化初期失败，页面显示：

```text
用户名或密码错误
```

但登录 debug 记录显示真实后端错误是：

```json
{"error":"invalid origin"}
```

这说明失败原因不是账号密码，而是后端 Origin 校验拒绝了来自 Vite Dev Server 的请求。

---

## 2. Origin 问题定位过程

### 2.1 失败表现

Playwright 页面访问：

```text
http://127.0.0.1:15173/login
```

前端通过 Vite proxy 请求：

```text
POST /api/v1/auth/login
```

后端返回：

```text
403 invalid origin
```

### 2.2 Origin 探测结果

曾验证以下 Origin：

```text
Origin: http://127.0.0.1:15173    => 403 invalid origin
Origin: http://localhost:15173     => 403 invalid origin
Origin: http://127.0.0.1:5173      => 403 invalid origin
Origin: http://localhost:5173       => 403 invalid origin
Origin: http://127.0.0.1:18080     => 200
Origin: http://localhost:18080      => 403 invalid origin
No Origin                          => 403 invalid origin
```

结论：

```text
Backend 当前 Origin 校验严格接受 http://127.0.0.1:18080。
```

Vite Dev Server 运行在 `15173`，浏览器真实 Origin 是：

```text
http://127.0.0.1:15173
```

如果 Vite proxy 不重写 Origin，Backend 会拒绝。

---

## 3. 正确修复方式

本轮采用 Vite proxy 重写 Origin，而不是关闭后端安全校验。

### 3.1 不推荐方式

不要为了测试直接关闭：

- Origin 校验
- CSRF 校验
- Session 校验

原因：

- 后续 UI 自动化需要覆盖真实保存类操作。
- 关闭安全校验会降低测试可信度。
- 保存运行配置、部署、启动实例等写操作都依赖真实 CSRF / Origin 流程。

### 3.2 推荐方式

在 `web/vite.config.ts` 中为 `/api`、`/metrics`、`/healthz` proxy 设置后端 Origin：

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

应用：

```ts
proxy: {
  '/api': backendProxy,
  '/metrics': backendProxy,
  '/healthz': backendProxy,
}
```

### 3.3 验证标准

运行登录 debug：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go/web
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/auth/login-debug.spec.ts --reporter=line
```

期望看到：

```text
POST /api/v1/auth/login 200
```

不应再出现：

```text
403 invalid origin
```

---

## 4. 登录自动化设计

### 4.1 文件结构

```text
web/tests/e2e/helpers/auth.ts
web/tests/e2e/global.setup.ts
web/tests/e2e/auth/login.spec.ts
web/tests/e2e/auth/login-debug.spec.ts
web/tests/e2e/.auth/admin.json
```

### 4.2 storageState

登录态保存到：

```text
web/tests/e2e/.auth/admin.json
```

该文件是本地运行产物，不允许提交。

`.gitignore` 应包含：

```gitignore
tests/e2e/.auth/
```

### 4.3 global setup 职责

`global.setup.ts` 必须完成：

1. 启动浏览器。
2. 打开 Web 首页。
3. 使用默认管理员账号登录。
4. 如遇首次登录改密码页，提交新密码。
5. 如果改密码后系统自动 logout，使用新密码重新登录。
6. 验证 `/api/v1/auth/me` 返回 `200`。
7. 保存 storageState。

---

## 5. 首次登录改密码流程

Clean DB 后，管理员登录响应可能包含：

```json
{
  "must_change_password": true
}
```

实际验证中，首次登录后进入：

```text
/change-password
```

页面提示：

```text
这是您首次登录，为了账户安全，请修改默认密码。
```

改密码接口：

```text
POST /api/v1/auth/change-password
```

成功返回：

```json
{"status":"ok"}
```

注意：改密码成功后系统可能自动调用：

```text
POST /api/v1/auth/logout
```

因此，global setup 必须重新使用新密码登录，不能假设改密码后仍处于登录态。

---

## 6. 认证判断方式

### 6.1 正确方式

在 UI 自动化中，认证判断必须走浏览器上下文：

```ts
export async function isAuthenticated(page: Page): Promise<boolean> {
  const status = await page.evaluate(async () => {
    const response = await fetch('/api/v1/auth/me', {
      credentials: 'include',
    })

    return response.status
  })

  return status === 200
}
```

### 6.2 不推荐方式

不要在 UI 测试中直接请求 Backend：

```ts
page.request.get('http://127.0.0.1:18080/api/v1/auth/me')
```

原因：

- 不经过 Vite proxy。
- 不等价于浏览器真实行为。
- 不一定共享页面 cookie。
- 容易误判登录状态。

---

## 7. 登录 smoke 断言注意事项

不要使用过宽断言：

```ts
page.getByText(/登录|登陆|sign in|log in/i)
```

原因：已登录页面存在：

```text
退出登录
```

该文本也包含“登录”，会导致误判。

推荐断言：

```ts
await expect(page.getByText(/Administrator @ Default Tenant/)).toBeVisible()
await expect(page.getByText(/^登录$/)).toHaveCount(0)
```

---

## 8. 登录 debug 测试用途

`login-debug.spec.ts` 用于诊断登录链路，不作为长期业务断言主入口。

它应输出：

- 登录后 URL
- 登录后页面文本
- localStorage / sessionStorage
- `/api/v1/auth/me` 浏览器 fetch 结果
- API response 摘要
- 截图：
  - `/tmp/lightai/e2e/playwright/login-after-login.png`
  - `/tmp/lightai/e2e/playwright/login-after-change-password.png`

使用命令：

```bash
npm run test:e2e:noauth -- --project=chrome-local tests/e2e/auth/login-debug.spec.ts --reporter=line
```

---

## 9. 常见故障与处理

### 9.1 `Executable doesn't exist ... ffmpeg-linux`

原因：启用了 Playwright video，但未安装 Playwright ffmpeg。

修复：

```bash
npx playwright install ffmpeg
```

或临时关闭：

```ts
video: 'off'
```

### 9.2 Playwright 看似挂住

现象：Vite 输出：

```text
Local: http://127.0.0.1:15173/
```

但 Playwright 等待 `5173`。

原因：`baseURL` 与实际 Vite 端口不一致。

修复：

```ts
const baseURL = process.env.LIGHTAI_WEB_URL ?? 'http://127.0.0.1:15173'
```

并固定：

```ts
command: 'npm run dev -- --host 127.0.0.1 --port 15173 --strictPort'
```

### 9.3 `/api/v1/auth/login` 返回 `invalid origin`

原因：Vite proxy 未重写 Origin。

修复：见本文第 3 节。

### 9.4 登录 smoke 失败，提示页面仍有“登录”

原因：已登录页存在“退出登录”。

修复：使用 `^登录$` 或更稳定的页面元素判断。

### 9.5 Clean DB 后登录状态变化

原因：首次登录强制改密码。

处理：global setup 必须支持改密码和重新登录。

---

## 10. 当前结论

本轮已确认：

```text
Vite proxy Origin 修复后，登录接口返回 200。
Clean DB 首次改密码流程可自动处理。
改密码后重新登录可进入首页。
浏览器上下文 /api/v1/auth/me 返回 200。
storageState 可生成并用于登录 smoke。
```
