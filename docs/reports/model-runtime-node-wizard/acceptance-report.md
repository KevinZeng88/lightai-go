# Model Runtime Node Wizard Acceptance Report

**Phase:** 4 — Model Runtime Wizards
**Branch:** `phase-4-model-runtime-wizards`
**Commits:**
- `84d86ec` — backend foundation + i18n
- `83b7ed8` — web UI (model wizard, start wizard, RemoteFileBrowser, DockerImagePicker) + E2E script
- `8bdc7b8` — runtime wizard web + acceptance draft
- `7b205b0` — file browser root picker + error i18n
- `a392269` — dynamic root management (DB + API + Web)
- `50a25a5` — RunnerConfigsPage + 模板/配置 概念拆分

**Final Conclusion: ACCEPTED_WITH_SCOPE_NOTE**
(Core wizard flows complete; model consistency deep comparison deferred to future phase.)

---

## 1. 概念模型

```
推理后端 (Backend)         已有 /backends 页面
    ↓
运行模板 (BackendRuntime)   /runtimes 页面 — 系统只读 / 用户可编辑
    ↓
运行配置 (新)               /runner-configs 页面 — 模板 + 节点 + 运行方式 + 镜像
    ↓
实例启动                    已有 /models/deployments 页面
```

---

## 2. Server API

| Endpoint | Method | File |
|----------|--------|------|
| `/api/v1/nodes/{id}/docker-images` | GET | `agent_handlers.go` |
| `/api/v1/nodes/{id}/files` | GET | `agent_proxy_handlers.go` |
| `/api/v1/nodes/{id}/model-paths/scan` | POST | `agent_proxy_handlers.go` |
| `/api/v1/nodes/{id}/model-browser/roots` | GET/POST/DELETE | `model_browser_handlers.go` |
| `/api/v1/model-artifacts/{id}/locations/{lid}` | PATCH/DELETE | `model_location_handlers.go` |
| `/api/v1/backend-runtimes/{id}/clone` | POST | `node_runtime_handlers.go` |
| `/api/v1/nodes/{id}/backend-runtimes/{nbr_id}` | PATCH/DELETE | `node_runtime_handlers.go` |
| `/api/v1/deployments/preflight` | POST | `preflight_handlers.go` |

---

## 3. Agent Capabilities

| Capability | Endpoint | Details |
|-----------|----------|---------|
| Docker images | `GET /docker-images` | repository, tag, image_id, digest, created_at, size, search, pagination |
| File browser | `GET /files` | controlled listing, allowed_roots + dynamic extra_roots, path traversal prevention |
| Model scanner | `POST /model-paths/scan` | HuggingFace config.json, safetensors, GGUF detection |

---

## 4. Web Pages

| Page | Route | Description |
|------|-------|-------------|
| BackendsPage | `/backends` | 推理后端列表（已有，未改） |
| **BackendRuntimesPage** | `/runtimes` | 运行模板列表 + CRUD + 克隆 + 详情（含节点配置管理） |
| **RunnerConfigsPage** | `/runner-configs` | **新建** 运行配置列表 + 5 步 wizard |
| **ModelArtifactsPage** | `/models/artifacts` | 模型列表 + 新建 wizard + 位置管理 |
| **ModelDeploymentsPage** | `/models/deployments` | 部署列表 + 启动 wizard（含 preflight） |

---

## 5. Web Components

| Component | Used In | Description |
|-----------|---------|-------------|
| RemoteFileBrowser | ModelArtifactsPage, BackendRuntimesPage | 目录浏览 + 动态根目录管理 |
| DockerImagePicker | RunnerConfigsPage, BackendRuntimesPage | Docker 镜像选择 + 搜索 |

---

## 6. Wizard Flows

### 模型新增 (ModelArtifactsPage)
```
选择节点 → 浏览目录 → 扫描 → 命名 → 保存
```

### 运行配置新增 (RunnerConfigsPage)
```
选择模板 → 运行方式(docker) → 节点 → 镜像 → 命名+检测+保存
```

### 实例启动 (ModelDeploymentsPage)
```
选择模型 → 选择运行配置 → preflight → 节点+端口 → 启动
```

---

## 7. 文件浏览动态根目录

| 来源 | 存储 | 管理方式 |
|------|------|---------|
| 静态 roots | `agent.yaml` `allowed_roots` | 编辑配置文件，重启 Agent |
| 动态 roots | DB `nodes.model_browser_extra_roots` | Web "+" 按钮，即时生效 |

安全：禁止添加 `/etc`, `/root`, `/proc`, `/sys`, `/dev`, `/run`, `/boot`

---

## 8. i18n

- 521 leaf keys in both zh-CN and en-US
- 448 key references in Vue/TS — all resolve to strings
- Namespaces: modelWizard, modelLocations, runtimeWizard, runnerConfigs, nodeRuntime, fileBrowser, dockerImages, preflight, startWizard
- `runtimes.*` renamed to "运行模板"/"Runtime Templates"

---

## 9. DB Migration

| Version | Change |
|---------|--------|
| V14 | `nodes.model_browser_extra_roots TEXT NOT NULL DEFAULT '[]'` |

---

## 10. E2E Script

- `scripts/e2e-model-runtime-wizard-nvidia-api.sh`
- 覆盖：login → files → scan → artifact+location → docker images → enable runtime → clone → preflight → deploy → /v1/models → logs → stop → cleanup

---

## 11. Verification

```bash
go build ./...     ✅
go test ./...      ✅
go vet ./...       ✅
npm run build      ✅
npm test           ✅ 521 keys, 448 refs, no i18n leaks
bash -n scripts    ✅
git diff --check   ✅
```

---

## 12. Remaining

| Item | Priority | Notes |
|------|----------|-------|
| E2E full run on NVIDIA hardware | P1 | Script ready; needs services + GPU + model |
| Deep model consistency comparison | P2 | Basic scanner exists |
| Runner type: non-Docker (local/remote) | P2 | Wizard dropdown has only docker; UI ready to expand |
