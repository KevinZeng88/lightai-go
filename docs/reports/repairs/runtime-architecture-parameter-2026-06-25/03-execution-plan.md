# Execution Plan

> Status: DRAFT
> Date: 2026-06-25
> Prerequisite: `02-work-package-design.md`

---

## 1. 总体执行策略

- 按 Step 0 → WP-A → WP-C → WP-B → WP-D → WP-E → WP-F 顺序执行
- 每个 WP 完成后独立验证，满足验收标准后再进入下一个
- 每个 WP 一个 git commit
- 如发现新的 P0 问题，停止并汇报
- 如连续两轮无法定位问题，停止并汇报

---

## 0. Step 0：标记 closeout REOPENED（前置步骤）

### 目标

在执行任何代码修复前，先修正 closeout 文档状态，避免后续执行者被旧 closeout 误导。

### 涉及文件

| 文件 | 修改要点 |
|------|---------|
| `docs/reports/phase-3/runtime-parameter-system-final-closeout.md` | 顶部添加 REOPENED banner + 交叉引用本 repair 目录 |
| `docs/reports/phase-3/runtime-parameter-layering-final-closeout.md` | 顶部添加 REOPENED banner + 交叉引用本 repair 目录 |

### 修改要点

在两个 closeout 文档顶部（`#` 标题之后，正文之前）插入：

```markdown
> ⚠️ **REOPENED** (2026-06-25)
> 本 closeout 声称 CLOSED 但真实 UI 仍存在以下问题：
> - 运行配置页面重复编辑入口 (RAP-001, RAP-002)
> - 浏览器 OOM (RAP-003)
> - help 文档未接入 UI (RAP-005)
> 详见：`docs/reports/repairs/runtime-architecture-parameter-2026-06-25/`
```

### 风险点

无代码风险。纯文档修改。

### 必跑测试

无需测试。

### Evidence 路径

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-f-architecture-items/
```

### Commit 条件

- 文档已更新

### Commit 格式

```
docs(runtime-params): mark runtime parameter closeout docs as REOPENED

Co-Authored-By: Claude <noreply@anthropic.com>
```

### 继续执行条件

Step 0 完成后，继续 WP-A。

---

## 2. Work Package A：参数编辑 UI 数据流闭环

### 目标

修复 RunnerConfigsPage 和 ModelDeploymentsPage 的参数编辑器数据流。以 BackendRuntimesPage 为参考实现。

### 涉及文件

| 文件 | 修改要点 |
|------|---------|
| `web/src/pages/RunnerConfigsPage.vue` | 删除 legacy Docker editor (template lines 232-243)；删除对应 ref 变量；showEdit() 中添加 editParameterModel.value populate（参考 BackendRuntimesPage.showEdit） |
| `web/src/pages/ModelDeploymentsPage.vue` | 将 editParameterModel 从 computed 改为 ref；showEdit() 中从 row 填充 docker_json + parameter_values_json；修正 setter 支持完整字段 |

### 修改要点

**RunnerConfigsPage.vue：**
```typescript
// showEdit() 中添加（参考 BackendRuntimesPage.vue:241-246）：
editParameterModel.value = {
  docker_json: row.config_snapshot_json?.docker_json || {},
  args_override_json: Array.isArray(row.config_snapshot_json?.args_override_json)
    ? row.config_snapshot_json.args_override_json : [],
  default_env_json: typeof row.config_snapshot_json?.default_env_json === 'object'
    ? row.config_snapshot_json.default_env_json : {},
  parameter_values_json: Array.isArray(row.config_snapshot_json?.parameter_values_json)
    ? row.config_snapshot_json.parameter_values_json : [],
}
```

**ModelDeploymentsPage.vue：**
```typescript
// 改 computed 为 ref：
const editParameterModel = ref({
  docker_json: {} as Record<string, any>,
  args_override_json: [] as string[],
  default_env_json: {} as Record<string, string>,
  parameter_values_json: [] as any[],
})

// showEdit() 中填充：
editParameterModel.value = {
  docker_json: row.config_snapshot_json?.docker_json || {},
  args_override_json: Array.isArray(row.config_snapshot_json?.args_override_json)
    ? row.config_snapshot_json.args_override_json : [],
  default_env_json: typeof row.config_snapshot_json?.default_env_json === 'object'
    ? row.config_snapshot_json.default_env_json : {},
  parameter_values_json: Array.isArray(row.parameter_values_json)
    ? row.parameter_values_json : [],
}
```

### 风险点

1. 删除 legacy editor 后，RunnerConfigsPage 的 `doEdit()` 保存逻辑需确认仍能正确序列化参数 → 参考 `BackendRuntimesPage.doEdit()` 的保存方式
2. ModelDeploymentsPage 的 computed→ref 变更影响所有 `editParameterModel` 消费方 → grep 确认所有引用
3. 其他组件/页面可能引用 legacy ref 变量（如 `editPrivileged`, `editIpcMode`）→ 删除前 grep 确认

### 必跑测试

```bash
cd web && npm run build
cd web && npm test
# 手动 UI 验证：
# 1. RunnerConfigsPage → 编辑 → 确认单入口 → 修改参数 → 保存 → 重新打开 → 数据一致
# 2. ModelDeploymentsPage → 编辑 → Docker 参数可编辑 → 保存 → 重新打开 → 数据一致
# 3. BackendRuntimesPage → 编辑 → 功能不受影响
```

### Evidence 路径

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-a-ui-data-flow/
```

### Commit 条件

- `npm run build` PASS
- `npm test` PASS（无回归）
- UI 验证：三个页面编辑弹窗只有 RuntimeParameterEditor 渲染 Docker 参数
- UI 验证：保存后重新打开数据一致
- 无新增 TypeScript 编译错误
- git diff 与 WP-A 范围一致

### Commit 格式

```
fix(runtime-params): unify parameter edit data flow across config pages

- Remove legacy Docker editor from RunnerConfigsPage
- Populate editParameterModel in RunnerConfigsPage.showEdit()
- Change ModelDeploymentsPage editParameterModel from computed to ref
- Populate full editor model from row data in ModelDeploymentsPage.showEdit()

Co-Authored-By: Claude <noreply@anthropic.com>
```

### 继续执行条件

WP-A 所有验收标准满足后，继续 WP-C。

### 建议人工确认

**否。** 此为明确 bug fix，不需要设计取舍。

---

## 4. Work Package B：RuntimeParameterEditor 稳定性与 OOM 修复

### 目标

消除 watch→emit→watch 循环，确保页面停留无内存持续增长。

### 涉及文件

| 文件 | 修改要点 |
|------|---------|
| `web/src/components/common/RuntimeParameterEditor.vue` | 修复 syncing guard + 优化 modelValue watch |
| `web/src/pages/BackendRuntimesPage.vue` | 移除父组件冗余 commandPreview（如 RuntimeParameterEditor 内部已有） |
| `web/src/pages/RunnerConfigsPage.vue` | 移除父组件冗余 commandPreview（如存在） |

### 修改要点

**RuntimeParameterEditor.vue：**

方案 A（推荐）— 将 modelValue watch 改为单向同步：

```typescript
// 1. 将 syncing guard 改为在 nextTick 后清除
let syncing = false
let syncPending = false

// 2. modelValue watch 改为浅比较 + 异步 guard
watch(() => props.modelValue, (newVal) => {
  if (syncing) return
  // 使用序列号或 hash 而非 JSON.stringify 大对象
  const key = hashKeys(newVal)  // 只比较关键字段
  if (key !== lastModelKey) {
    lastModelKey = key
    syncing = true
    loadFromModel()
    nextTick(() => { syncing = false })
  }
})

// 3. buildOutput 内 emit 前设置 syncing
function buildOutput() {
  // ... build output ...
  syncing = true
  emit('update:modelValue', output)
  nextTick(() => { syncing = false })
}
```

方案 B（备选）— 完全移除 modelValue watcher：

```typescript
// parent 通过 expose 方法注入初始数据
function setModelValue(val: any) {
  syncing = true
  loadFromModel(val)
  nextTick(() => { syncing = false })
}
defineExpose({ setModelValue })
```

**BackendRuntimesPage.vue：**
- 如果 `<pre class="preview">{{ commandPreview }}</pre>` 存在且 RuntimeParameterEditor 内部也有 commandPreview → 移除父组件版本

### 风险点

1. syncing guard 修改可能影响参数实时预览 → 确保修改参数后 command preview 仍更新
2. 方案 A 的 hashKeys 实现需要覆盖所有影响 buildOutput 的字段
3. 方案 B 需要所有调用方改为 expose 方式传数据（BackendRuntimesPage, RunnerConfigsPage, ModelDeploymentsPage）

### 必跑测试

```bash
cd web && npm run build && npm test
# 手动内存验证（Chrome DevTools）：
# 1. 打开 BackendRuntimesPage 编辑弹窗 → Performance Monitor → 观察 JS heap
# 2. 修改参数 10 次 → 确认 heap 不持续增长
# 3. 停留 2 分钟 → 确认 heap 稳定
# 4. RunnerConfigsPage、ModelDeploymentsPage 重复以上
```

### Evidence 路径

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-b-editor-stability/
```

### Commit 条件

- Chrome Memory profiler 确认无循环分配
- 三个页面编辑弹窗停留 2 分钟无 OOM
- 参数修改响应稳定
- `npm test` PASS

### Commit 格式

```
fix(runtime-params): eliminate watch-emit cycle in RuntimeParameterEditor

- Fix syncing guard timing for Vue 3 async watcher flush
- Optimize modelValue watch to avoid deep comparison
- Remove redundant parent-side commandPreview when editor has its own

Co-Authored-By: Claude <noreply@anthropic.com>
```

### 继续执行条件

WP-B 所有验收标准满足后，继续 WP-D。

### 建议人工确认

**否。** 技术修复，不需要设计取舍。

---

## 3. Work Package C：Catalog、打包与 clean DB 初始化

### 目标

修复打包脚本，确保 release tarball 包含 backend-catalog，clean DB 可正常初始化。

### 涉及文件

| 文件 | 修改要点 |
|------|---------|
| `scripts/package-release.sh` | 添加 `cp -r configs/backend-catalog "$BUILD_DIR/configs/"`；评估是否删除 `configs/model-runtime/` 拷贝行 |

### 修改要点

```bash
# 替换 line 237-239 的 model-runtime 拷贝：
# 旧：
mkdir -p "$BUILD_DIR/configs/model-runtime"
cp -r configs/model-runtime/* "$BUILD_DIR/configs/model-runtime/" 2>/dev/null || true

# 新：
mkdir -p "$BUILD_DIR/configs/backend-catalog"
cp -r configs/backend-catalog/* "$BUILD_DIR/configs/backend-catalog/"
```

### 风险点

1. server 启动时的 catalog 加载路径是否与 release 目录结构匹配 → 检查 `server.release.yaml` 中的 catalog 路径配置
2. `configs/model-runtime/` 拷贝行是否可以安全删除 → grep 确认 server 代码不引用此路径

### 必跑测试

```bash
# 1. 打包
./scripts/package-release-docker.sh

# 2. 验证 tarball 内容
tar -tzf dist/lightai-go-*.tar.gz | grep -E 'backend-catalog/(backends|runtimes|versions|catalog.yaml)'

# 3. Clean DB 启动验证
rm -f data/lightai.db
docker run --rm -d --name lightai-test -p 18080:18080 lightai-go:latest
sleep 5
curl -s http://localhost:18080/api/v1/inference-backends | jq '.data | length'
docker stop lightai-test

# 4. 现有测试无回归
cd web && npm test
go test ./internal/...
```

### Evidence 路径

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-c-package-catalog/
```

### Commit 条件

- `tar -tzf` 确认包含 catalog YAML
- Clean DB 启动后 API 返回 backend 列表
- 现有测试 PASS

### Commit 格式

```
fix(package): include backend-catalog in release artifact

- Replace empty configs/model-runtime copy with configs/backend-catalog
- Update package-release.sh to include catalog YAML files

Co-Authored-By: Claude <noreply@anthropic.com>
```

### 继续执行条件

WP-C 所有验收标准满足后，继续 WP-B。

### 建议人工确认

**否。** 明确 bug fix。

---

## 5. Work Package D：参数 help 与用户可理解性

### 目标

在 RuntimeParameterEditor 中为每个参数添加 help 入口。

### 涉及文件

| 文件 | 修改要点 |
|------|---------|
| `web/src/components/common/RuntimeParameterEditor.vue` | 为每个参数行添加 ? icon + el-popover |
| `web/src/api/` | 可能需要添加 help 数据加载 API 客户端 |

### 修改要点

**数据来源决策（二选一）：**

选项 1 — 通过 API 加载（推荐，与 catalog 解耦）：
- 新增 `GET /api/v1/backend-help/:backend_version_id?lang=zh-CN`
- 前端在打开编辑弹窗时加载 help 数据

选项 2 — 内联到 schema（简单但 YAML 需改）：
- 将 help 字段合并到 `default_args_schema_json` 中
- 编辑器中直接从 schema 读取 help

**UI 实现：**
```html
<el-popover placement="right" :width="300" trigger="hover">
  <template #reference>
    <el-icon class="help-icon"><QuestionFilled /></el-icon>
  </template>
  <div class="param-help">
    <p><strong>{{ help.title }}</strong></p>
    <p>{{ help.summary }}</p>
    <el-divider />
    <p>{{ $t('params.defaultValue') }}: {{ help.official_default }}</p>
    <p>{{ $t('params.recommendation') }}: {{ help.lightai_recommendation }}</p>
    <el-alert v-if="help.risk" :title="help.risk" type="warning" />
  </div>
</el-popover>
```

### 风险点

1. 选项 1 需要新增后端 API endpoint + 从 YAML 加载 help
2. 选项 2 需要修改 catalog YAML 结构和 schema 传播链
3. help 内容较长可能影响编辑器布局

### 必跑测试

```bash
cd web && npm run build && npm test
# 手动：每个后端打开编辑弹窗 → 验证 help popover
```

### Evidence 路径

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-d-help-ui/
```

### Commit 条件

- vLLM/SGLang/llama.cpp 三后端 help popover 可用
- 现有测试无回归

### Commit 格式

```
feat(runtime-params): add parameter help tooltips to RuntimeParameterEditor

- Add ? icon with popover for each parameter
- Load help data from backend API / inline schema

Co-Authored-By: Claude <noreply@anthropic.com>
```

### 继续执行条件

WP-D 所有验收标准满足后，继续 WP-E。

### 建议人工确认

**是。** 需要确认：选项 1（新 API endpoint）vs 选项 2（内联 schema）。

---

## 6. Work Package E：测试体系补强

### 目标

补最小有效测试，让后续同类问题能被自动化发现。

### 涉及文件

| 文件 | 修改要点 |
|------|---------|
| `web/tests/runtimeBoundaryUi.test.mjs` | 增加负向断言：RunnerConfigsPage 不应含 legacy Docker editor 控件字符串 |
| `scripts/e2e-packaged-smoke.sh` (新建) | 从 release tarball 启动 → API 验证 → 健康检查 |
| `scripts/e2e-ui-browser-smoke.sh` (新建) | Headless browser 打开关键页面，验证元素存在/不存在 |

### 修改要点

**runtimeBoundaryUi.test.mjs 负向断言：**
```javascript
// 新增：确认 RunnerConfigsPage 不含重复 Docker 编辑入口
const runnerConfigsSource = fs.readFileSync('src/pages/RunnerConfigsPage.vue', 'utf-8')
// 确认 RuntimeParameterEditor 存在
assert(runnerConfigsSource.includes('RuntimeParameterEditor'), 'RunnerConfigsPage should use RuntimeParameterEditor')
// 确认 getEditParameterModel 或 populate 逻辑存在
assert(runnerConfigsSource.includes('editParameterModel.value'), 'RunnerConfigsPage should populate editParameterModel')
// 负向：不应同时包含 legacy 和 RuntimeParameterEditor 的双入口模式
// （具体断言取决于修复后代码结构）
```

**e2e-packaged-smoke.sh：**
```bash
#!/bin/bash
# 1. Build release
./scripts/package-release-docker.sh
# 2. Start container
# 3. Wait for health
# 4. curl /api/v1/inference-backends → assert count > 0
# 5. curl /api/v1/nodes → assert count > 0
# 6. Stop and clean
```

**e2e-ui-browser-smoke.sh：**
```bash
#!/bin/bash
# 使用 headless Chromium / Playwright
# 1. 打开 BackendRuntimesPage
# 2. 截图
# 3. 确认关键元素存在
# 4. 确认无重复 Docker 编辑入口
```

### 风险点

1. Playwright/headless browser 需要安装依赖 → 可能增加 CI 复杂度
2. 负向断言可能因实现细节变化而脆弱 → 用 loose 匹配
3. Packaged smoke 增加 CI 时间

### 必跑测试

```bash
cd web && npm test
bash scripts/e2e-packaged-smoke.sh
bash scripts/e2e-ui-browser-smoke.sh
```

### Evidence 路径

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-e-tests/
```

### Commit 条件

- 新测试 PASS
- 现有测试无回归
- packaged smoke 能发现 catalog 缺失
- browser smoke 能发现重复 UI 入口

### 继续执行条件

WP-E 所有验收标准满足后，继续 WP-F。

---

## 7. Work Package F：架构遗留项与策略项处理

### 目标

对架构遗留项做出明确处置决策，修正 closeout 文档状态。

### 涉及文件

| 文件 | 修改要点 |
|------|---------|
| `docs/reports/phase-3/runtime-parameter-system-final-closeout.md` | 添加 REOPENED marker + 交叉引用本 repair |
| `docs/reports/phase-3/runtime-parameter-layering-final-closeout.md` | 添加 REOPENED marker + 交叉引用 |
| `docs/reports/phase-3/open-issues-closeout.md` | 添加 RAP-001 至 RAP-014 的当前状态 |
| `internal/server/runplan/types.go` | 如决策删除：移除 DeviceBinding dead struct |
| `docs/reports/README.md` | 更新 evidence 索引 |
| `web/src/components/common/RuntimeParameterEditor.vue` | 如决策迁移：将 Docker 参数列表改为 schema/catalog 驱动 |

### 修改要点

**RAP-006 (extra_args 冲突检测)：**
- 评估：保持 WARNING 级别 + 添加 preflight 结构化 warning（非阻断 error）
- 理由：阻断 arg 冲突可能阻止合法的参数覆盖
- 如决策保持 warning → DEFERRED_WITH_REASON

**RAP-007 (DeviceBinding dead struct)：**
- 评估：删除 dead code（当前无消费者）
- 如决策保留 → 添加注释说明预期用途

**RAP-008 (RuntimeRequirements)：**
- 评估：DEFERRED_WITH_REASON
- 理由：当前 preflight 用 ad-hoc 检查覆盖了关键场景；完整实现需要硬件测试环境
- 触发条件：引入自动调度或 GPU capability matching 时

**RAP-011 (closeout 不一致)：**
- 执行：在两个 closeout 文档顶部添加 REOPENED banner
- 交叉引用：`docs/reports/repairs/runtime-architecture-parameter-2026-06-25/`

**RAP-013 (evidence 索引)：**
- 执行：更新 `docs/reports/README.md` 或 `evidence/README.md`

**RAP-014 (hardcoded Docker 参数)：**
- 评估：将 scalarOptions/listOptions 迁移到 schema/catalog 驱动的成本和风险
- 选项 A：迁移 → 新增 Docker 参数 schema 定义，RuntimeParameterEditor 动态读取
- 选项 B：保持 → 记录为技术债，明确后续扩展路径和触发条件
- 建议：本轮先评估；如改造成本低（<2h）则执行迁移；否则记录决策

### 风险点

1. 删除 DeviceBinding 需确认 100% 无引用（包括测试、序列化）
2. closeout 修改可能与其他文档交叉引用冲突
3. RAP-014 迁移到 schema 驱动可能影响现有参数编辑行为

### 必跑测试

```bash
go test ./internal/...
cd web && npm test
```

### Evidence 路径

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/wp-f-architecture-items/
```

### Commit 条件

- 所有 RAP-006/007/008 有明确处置决策
- RAP-011 closeout 已更新
- go test PASS（如删除了 Go 代码）

### Commit 格式

```
docs(runtime-params): reconcile architecture gap items and closeout status

- Mark closeout docs REOPENED with cross-reference to repair plan
- Address DeviceBinding dead code / extra_args policy / RuntimeRequirements deferral

Co-Authored-By: Claude <noreply@anthropic.com>
```

---

## 8. 全量回归

WP-A 至 WP-F 全部完成后，执行全量回归：

```bash
# 完整测试套件
bash scripts/e2e-real-smoke-all-three.sh
bash scripts/e2e-model-runtime-param-trace.sh
bash scripts/e2e-packaged-smoke.sh
bash scripts/e2e-matrix-verifier.sh
bash scripts/e2e-dryrun-parameter-matrix-enhanced.sh
cd web && npm test && npm run build
go test ./internal/...
```

Evidence 路径：`evidence/final-regression/`

---

## 9. Closeout

全量回归通过后：
1. 更新 `08-closeout-template.md` 为 FINAL
2. 填写实际验证结果
3. 汇总 evidence 索引
4. 更新 issue registry 最终状态
5. commit + push

Evidence 路径：`evidence/closeout/`
