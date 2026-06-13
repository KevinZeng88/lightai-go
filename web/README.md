# LightAI Go Web Console

LightAI Go 第一版 Web 管理控制台。

## 技术栈

- Vue 3 + TypeScript
- Vite
- Vue Router
- Pinia
- Element Plus
- vue-i18n

## 快速开始

### 安装依赖

```bash
cd web
npm install
```

### 启动开发服务器

```bash
npm run dev
```

开发服务器运行在 http://127.0.0.1:15173。

API 请求自动代理到 http://127.0.0.1:18080。

### 构建生产版本

```bash
npm run build
```

构建产物输出到 `web/dist/`。

### 构建嵌入式 Server

```bash
cd web && npm run build
cd ..
go build -tags web -o bin/lightai-server ./cmd/server
./bin/lightai-server --config configs/server.dev.yaml
```

访问 http://127.0.0.1:18080 即可使用 Web 控制台。

### 类型检查

```bash
npm run lint
```

## API 代理说明

开发模式下，Vite 代理以下路径到 Go Server：

- `/api` → http://127.0.0.1:18080
- `/metrics` → http://127.0.0.1:18080
- `/healthz` → http://127.0.0.1:18080

生产模式下（embedded），所有请求同源，无需代理。

## 国际化 (i18n)

默认语言：简体中文 (zh-CN)。

支持语言：
- 简体中文 (zh-CN)
- English (en-US)

右上角可切换语言，选择后保存在 localStorage，刷新页面后保持。

### 新增语言

1. 在 `src/locales/` 下创建新语言文件，如 `ja-JP.ts`
2. 在 `src/locales/index.ts` 中注册
3. 所有组件自动支持新语言

## 目录结构

```
web/
├── src/
│   ├── api/          # API 客户端
│   │   ├── client.ts  # 基础 HTTP 客户端
│   │   ├── auth.ts    # 认证 API
│   │   ├── nodes.ts   # 节点 API
│   │   ├── gpus.ts    # GPU API
│   │   └── metrics.ts # Metrics Targets API
│   ├── components/    # 通用组件
│   │   ├── StatusTag.vue
│   │   ├── CopyButton.vue
│   │   ├── MetricCard.vue
│   │   └── LanguageSwitcher.vue
│   ├── layouts/       # 布局
│   │   └── ConsoleLayout.vue
│   ├── locales/       # 国际化
│   │   ├── index.ts
│   │   ├── zh-CN.ts
│   │   └── en-US.ts
│   ├── pages/         # 页面
│   │   ├── LoginPage.vue
│   │   ├── ChangePasswordPage.vue
│   │   ├── DashboardPage.vue
│   │   ├── NodesPage.vue
│   │   ├── GpusPage.vue
│   │   ├── ObservabilityTargetsPage.vue
│   │   └── PlaceholderPage.vue
│   ├── router/        # 路由
│   ├── stores/        # 状态管理
│   ├── utils/         # 工具函数
│   ├── App.vue
│   └── main.ts
├── index.html
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## 常见问题

### 开发模式下登录失败

确保 Go Server 在 http://127.0.0.1:18080 运行。

### 生产模式下页面空白

确保先执行 `npm run build`，再使用 `-tags web` 构建。

### 中文乱码

所有日志和 debug bundle 使用英文。Web Console 默认中文，无需服务器 locale 支持。
