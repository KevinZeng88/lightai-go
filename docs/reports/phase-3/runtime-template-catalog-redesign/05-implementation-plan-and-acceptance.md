# 05 - 实施步骤与验收标准

## 1. 执行原则

1. 不做历史兼容迁移。
2. 允许重建开发 DB。
3. 不新建分支，除非用户明确要求。
4. 每个可定位、可修复、可验证的问题应直接修复、测试、提交、push。
5. 不使用 UI 手工验证作为唯一证据，优先 API-first 自动化。
6. 严格执行 copy-on-create。
7. 禁止 runtime dynamic inheritance。
8. 上游配置只作为创建来源，不作为运行依赖。

---

## 2. 开发批次

## Batch 0：本机全仓审计

Claude 先执行：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

git status --short
git rev-parse --abbrev-ref HEAD
git log --oneline -5

find configs/backend-catalog -type f | sort
grep -R "HumanRuntimeParameterForm" -n web/src || true
grep -R "getHumanFieldsForBackend" -n web/src || true
grep -R "runtime\\." -n web/src configs internal | head -200 || true
grep -R "template-only\\|from Metax release package\\|0d307f1665d3" -n configs internal web || true
grep -R "mergeNBRConfigSnapshot\\|resolveImage\\|VendorOptionsJSON\\|BackendVersion.defaultImages" -n internal/server || true
grep -R "config_set_json" -n internal/server | head -200
```

输出审计结果到：

```text
docs/reports/phase-3/runtime-template-redesign/current-code-audit.md
```

---

## Batch 1：ConfigItem schema 与 editor 对齐

修改：

- `internal/server/catalog/types.go`
- `internal/server/catalog/loader.go`
- `web/src/components/common/RuntimeParameterEditor.vue`
- 相关测试

目标：

1. `ConfigItem` 增加/支持：
   - required
   - visibility
   - readonly
   - advanced
   - render.label/help/group/options/placeholder/unit
   - constraints
2. `addArgConfigItems()` 把 label/group 同时写入 `render`，或前端读取 `extensions`。
3. RuntimeParameterEditor：
   - 支持 label/group/order/constraints/required。
   - 支持 select/multi_select。
   - 支持 top-level constraints。
   - 支持 hidden/internal 过滤。
   - 保留 enabled + value。
4. 新增 frontend test。

验收：

```bash
cd web
npm run build
npm test
```

---

## Batch 2：替换 HumanRuntimeParameterForm

修改：

- `web/src/components/runtime/HumanRuntimeParameterForm.vue`
- `web/src/utils/runtimeParameterViewModel.ts`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/components/deployments/NodeRuntimeConfigWizard.vue`

方案：

- 直接删除或停用 `HumanRuntimeParameterForm`。
- 用增强后的 `RuntimeParameterEditor` 或新 `SchemaDrivenParameterForm` 替代。
- 确保新增 schema 参数自动显示在：
  - BackendRuntime 编辑页
  - NBR 创建 wizard
  - NBR 详情页
  - Deployment override

验收：

1. 不再依赖 `HUMAN_FIELDS` 才能显示友好参数。
2. 新增 fake 参数后所有相关 UI 自动出现。
3. unmapped 参数不丢失。
4. hidden internal 项不在普通表单出现。

---

## Batch 3：BackendVersion UI

修改：

- `web/src/pages/BackendsPage.vue`
- 或新增：
  - `web/src/pages/BackendVersionsPage.vue`
  - `web/src/components/backend/BackendVersionEditor.vue`
  - `web/src/components/backend/ConfigItemEditor.vue`
- `web/src/router/index.ts`
- i18n 文件和测试

功能：

1. 后端列表展示。
2. 点击 Backend 后展示版本列表。
3. 支持 clone 系统版本。
4. 支持新增用户版本。
5. 支持编辑用户版本 ConfigSet。
6. 支持新增参数。
7. 支持删除未被 runtime 使用的用户版本。
8. 系统版本只读。

BackendVersion 参数新增表单至少支持：

- code
- label
- help
- category
- group
- kind
- type
- cli flag / env name
- render style
- default value
- enabled
- required
- order
- constraints min/max/step
- support level

验收：

```bash
# API
POST /api/v1/backend-versions/{id}/clone
PATCH /api/v1/backend-versions/{new_id}

# UI
BackendVersion 页面能新增 fake 参数并自动显示输入框
```

---

## Batch 4：BackendVersion 边界清理

修改：

- `internal/server/api/backend_handlers.go`
- `internal/server/catalog/loader.go`
- catalog YAML

目标：

1. BackendVersion 禁止写 vendor image 和 Docker/device 字段。
2. `image_ref`、docker options、devices、vendor env 只能放 Runtime。
3. `MaterializeBackendVersion()` 以 Backend materialized ConfigSet 为来源，再追加版本参数。
4. 系统版本只读，clone 后可编辑。
5. BackendVersion source_metadata 记录 copy-on-create。

验收：

- API 传 `image_ref` 到 BackendVersion 返回 400，提示应放 BackendRuntime。
- BackendVersion 创建时复制 Backend 当前 config_set。
- 修改 Backend 后既有 BackendVersion 不变。

---

## Batch 5：Runtime catalog 清理

修改：

- `internal/server/db/db.go`
- `internal/server/catalog/types.go`
- `internal/server/catalog/loader.go`
- `configs/backend-catalog/runtimes/**`
- `web/src/api/runtimes.ts`
- `NodeRuntimeConfigWizard.vue`
- `BackendRuntimesPage.vue`

目标：

1. 增加 visibility/support_level 或 source_metadata 等价字段。
2. visible selector 只展示：
   - visibility=visible
   - status in active/experimental
3. hidden/reference 模板不进入普通选择器。
4. ValidateCatalog 阻止 visible 逻辑重复模板。
5. 清理脏模板。
6. 保留国内 GPU hidden reference 模板。

验收：

```bash
go test ./internal/server/catalog/... ./internal/server/api/...
```

API 验证：

```bash
curl /api/v1/backend-runtimes
```

确认普通列表不包含：

```text
template-only
<from Metax release package>
0d307f1665d3
重复 nvidia.vllm
runtime.xxx
```

---

## Batch 6：严格 copy-on-create

修改：

- `internal/server/api/backend_handlers.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/server/runplan/resolver.go`
- tests

目标：

1. BackendVersion 创建复制 Backend。
2. BackendRuntime 创建复制 BackendVersion。
3. NBR enable 复制 BackendRuntime。
4. Deployment create 复制 NBR。
5. 删除 Deployment fallback 到 BackendRuntime。
6. RunPlan 不 fallback 到 BackendRuntime/BackendVersion 当前值。
7. source_metadata 记录来源 hash/copy time/detached boundary。

验收 API-first：

- 修改上游不影响已创建下游。
- Deployment dry-run 不读取上游当前值。
- 缺少 NBR snapshot 时明确失败。

---

## Batch 7：全仓 review 问题修复

至少处理以下问题：

1. `configSetParameterValues()` env kind 不可达。
2. reset admin password interactive 明文输入。
3. NBR aggregate endpoint N+1 查询优化或记录为小规模可接受并补测试。
4. Deployment detail 增加参数表。
5. RuntimeTemplate API 返回 parsed object。
6. router 格式整理。
7. README / docs 更新。

---

## 3. API-first E2E 用例

新增脚本建议：

```text
scripts/e2e-backend-version-schema-driven-parameters.sh
scripts/e2e-copy-on-create-boundary.sh
scripts/e2e-runtime-template-catalog-clean.sh
```

### 3.1 schema-driven 参数用例

流程：

1. 登录获取 session/CSRF。
2. 查询 backend.vllm。
3. clone `backend-version.vllm.compat`。
4. PATCH 新版本，新增：

```json
backend.arg.fake_new_param
```

5. GET 版本，确认 config_set.items 包含该参数。
6. 从该版本创建 BackendRuntime。
7. GET BackendRuntime，确认已复制 fake 参数。
8. enable NBR。
9. 创建 Deployment override 启用 fake 参数。
10. dry-run，确认 docker preview 包含：

```text
--fake-new-param 123
```

11. disabled 后 dry-run 不包含该参数。

### 3.2 copy-on-create 边界用例

1. 创建 Version V1。
2. 创建 Runtime R1。
3. 修改 V1 新增参数 P2。
4. 确认 R1 不包含 P2。
5. 从 V1 创建 R2，确认 R2 包含 P2。
6. enable NBR N1 from R2。
7. 修改 R2 参数 P2 值。
8. 确认 N1 不变。
9. 创建 Deployment D1 from N1。
10. 修改 N1 参数。
11. dry-run D1，确认 D1 不变。

### 3.3 catalog clean 用例

1. 重建 DB。
2. 启动 server，catalog seed。
3. 查 backend_runtimes。
4. 按 vendor/backend/version 统计 visible 模板。
5. 确认无重复。
6. 确认 hidden reference 不在普通 selector。
7. 重启 server 或 reload catalog。
8. 再查数量，不增长。

---

## 4. 常规测试命令

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...

cd web
npm run build
npm test
```

如已有 E2E：

```bash
ls scripts/e2e*.sh
bash scripts/e2e-real-smoke-all-three.sh
```

Claude 需根据本机现有脚本选择最相关脚本执行，不能只跑 build。

---

## 5. 验收输出要求

Claude 完成后必须输出：

1. 修改文件列表。
2. Backend/BackendVersion/Runtime/NBR/Deployment copy-on-create 说明。
3. 最终 visible runtime templates。
4. hidden/reference runtime templates。
5. 删除或隐藏的脏模板。
6. 新增参数自动渲染测试结果。
7. 上游修改不影响下游的 API-first 证据。
8. RunPlan snapshot-only 证据。
9. Web build/test 结果。
10. Go build/test 结果。
11. E2E 结果。
12. closeout 文档路径。
13. commit id。
14. push 结果。
15. `git status --short`。
