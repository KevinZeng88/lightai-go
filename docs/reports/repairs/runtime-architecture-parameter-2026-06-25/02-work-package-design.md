# Work Package Design

> Status: DRAFT
> Date: 2026-06-25

---

## 1. Work Package 总览

| WP | 名称 | 覆盖 Issues | 严重级别 | 目标 | 优先级 |
|----|------|-----------|---------|------|--------|
| A | 参数编辑 UI 数据流闭环 | RAP-001, RAP-002 | P0 | 三个页面统一使用 RuntimeParameterEditor，数据正确传播 | 1 |
| B | RuntimeParameterEditor 稳定性与 OOM 修复 | RAP-003 | P0 | 消除 watch→emit 循环，页面停留无内存增长 | 2 |
| C | Catalog、打包与 clean DB 初始化 | RAP-004 | P0 | release tarball 包含 catalog，clean DB 可正常启动 | 3 |
| D | 参数 help 与用户可理解性 | RAP-005 | P1 | 每个参数旁有 ? icon，hover 显示 help | 4 |
| E | 测试体系补强 | RAP-009, RAP-010, RAP-012 | P2 | 增加 browser smoke + packaged smoke + 负向断言 | 5 |
| F | 架构遗留项与策略项处理 | RAP-006, RAP-007, RAP-008, RAP-011, RAP-013 | P1/P2/P3 | 对遗留项做决策 + 修正 closeout | 6 |

---

## 2. Work Package A：参数编辑 UI 数据流闭环

### 覆盖问题

- **RAP-001**：RunnerConfigsPage 双入口 + editParameterModel 不填充
- **RAP-002**：ModelDeploymentsPage computed 返回空 docker_json

### 相关性说明

两个问题根因相同：Phase 2 修复了 `BackendRuntimesPage` 但遗漏了 `RunnerConfigsPage` 和 `ModelDeploymentsPage`。三个页面应使用一致的参数编辑模型：
- `BackendRuntimesPage` → 已正确实现（参考实现）
- `RunnerConfigsPage` → 需删除 legacy editor + 添加数据 populate
- `ModelDeploymentsPage` → 需改 computed 为 ref + 完整数据 populate

### 目标

1. RunnerConfigsPage、ModelDeploymentsPage、BackendRuntimesPage 使用一致的 `{docker_json, args_override_json, default_env_json, parameter_values_json}` 编辑模型
2. 打开、编辑、保存、重新打开形成闭环
3. RuntimeParameterEditor 作为唯一参数编辑入口

### 执行边界

**修：**
- `RunnerConfigsPage.vue`：删除 legacy Docker editor (template lines 232-243)，showEdit() 中 populate editParameterModel
- `ModelDeploymentsPage.vue`：改 editParameterModel 从 computed 为 ref，showEdit() 从 row 填充完整数据
- 清理不再使用的 legacy ref 变量

**不修：**
- `RuntimeParameterEditor.vue` 内部逻辑（WP-B 修）
- `BackendRuntimesPage.vue`（已正确，不改）
- 后端 API
- 打包脚本

### 涉及文件

| 文件 | 修改类型 | 风险 |
|------|---------|------|
| `web/src/pages/RunnerConfigsPage.vue` | 中等修改 | 删除 legacy editor 需确认无其他引用 |
| `web/src/pages/ModelDeploymentsPage.vue` | 小修改 | computed→ref 需确认所有消费方适配 |

### 验收标准

1. RunnerConfigsPage 编辑弹窗：只有 RuntimeParameterEditor 渲染 Docker 参数
2. RunnerConfigsPage 编辑弹窗：参数编辑器显示实际 row 数据（非空白）
3. ModelDeploymentsPage 编辑弹窗：Docker 参数可编辑、可保存
4. 保存后重新打开：参数值、enabled 状态一致
5. `npm run build` PASS
6. `npm test` PASS（无回归）
7. BackendRuntimesPage 功能不受影响

### 与后续 WP 关系

- 完成后 WP-B 可在有真实数据的编辑器中验证 OOM 修复
- WP-D 依赖 WP-A 完成后接入 help UI
- WP-E 依赖 WP-A 完成后写有意义的数据流测试

---

## 3. Work Package B：RuntimeParameterEditor 稳定性与 OOM 修复

### 覆盖问题

- **RAP-003**：watch→emit→watch 循环导致 OOM

### 相关性说明

OOM 根因链涉及多个组件间交互：
- `RuntimeParameterEditor.vue` 内部 watch/emit 循环是核心
- `ModelDeploymentsPage.vue` computed 每次返回新引用是放大器（WP-A 已修）
- 父组件冗余 commandPreview 增加计算负担

### 目标

1. 参数编辑器不再出现循环 emit
2. 页面停留不持续增长内存
3. 修改参数响应稳定
4. BackendRuntimesPage、RunnerConfigsPage、ModelDeploymentsPage 全部通过验证

### 执行边界

**修：**
- `RuntimeParameterEditor.vue`：修复 syncing guard + 优化 modelValue watch
- `BackendRuntimesPage.vue`：移除父组件冗余 commandPreview（如存在）
- `RunnerConfigsPage.vue`：移除父组件冗余 commandPreview（如存在）

**不修：**
- RuntimeParameterEditor 参数控件结构
- hardcoded scalarOptions/listOptions

### 涉及文件

| 文件 | 修改类型 | 风险 |
|------|---------|------|
| `web/src/components/common/RuntimeParameterEditor.vue` | 核心修改 | syncing 逻辑改动需仔细验证 |
| `web/src/pages/BackendRuntimesPage.vue` | 小修改 | 移除冗余 preview |
| `web/src/pages/RunnerConfigsPage.vue` | 小修改 | 移除冗余 preview |

### 修复方案建议

**方案 A（推荐）：** 将 modelValue → local state 同步改为单向
- `props.modelValue` → `loadFromModel()` 只在 parent 显式设置新数据时触发（通过 watch `props.modelValue` 的特定 key 而非全对象）
- `buildOutput()` emit 后不触发回写
- `syncing` guard 改为在 `nextTick` 后清除

**方案 B（备选）：** 完全移除 modelValue watcher
- 参数编辑器完全自主管理内部 state
- 只在 save 时 emit 最终值
- parent 打开编辑弹窗时通过 expose 方法注入初始数据

### 验收标准

1. BackendRuntimesPage 编辑弹窗停留 2 分钟 → 浏览器内存稳定
2. RunnerConfigsPage 编辑弹窗停留 2 分钟 → 浏览器内存稳定
3. ModelDeploymentsPage 编辑弹窗停留 2 分钟 → 浏览器内存稳定
4. 修改参数值 → 实时预览更新 → 无卡顿
5. Chrome DevTools Memory profiler：无 continuous allocation
6. `npm run build` PASS
7. `npm test` PASS

### 与后续 WP 关系

- 依赖 WP-A 完成后提供真实数据编辑器环境
- WP-D 的 help popover 显示不应触发 OOM

---

## 4. Work Package C：Catalog、打包与 clean DB 初始化

### 覆盖问题

- **RAP-004**：package-release.sh 未包含 configs/backend-catalog/

### 相关性说明

打包脚本缺失 catalog 直接影响 packaged artifact 可用性，也可能影响 clean DB 场景下的 catalog 初始化。

### 目标

1. release tarball 包含 configs/backend-catalog/
2. clean DB 启动后能加载 vLLM、SGLang、llama.cpp
3. API 和 UI 都能看到 catalog 初始化结果
4. 打包验证成为固定 smoke

### 执行边界

**修：**
- `scripts/package-release.sh`：添加 `configs/backend-catalog/` 拷贝

**不修：**
- catalog YAML 内容
- catalog 加载逻辑（已正确）
- DB migration

### 涉及文件

| 文件 | 修改类型 | 风险 |
|------|---------|------|
| `scripts/package-release.sh` | 小修改 | 确认 `configs/model-runtime/` 是否应删除拷贝行 |

### 验收标准

1. `tar -tzf dist/lightai-go-*.tar.gz | grep backend-catalog` 显示 catalog YAML 文件
2. Clean DB 后启动 packaged artifact → `curl /api/v1/inference-backends` 返回 backend 列表
3. BackendRuntimesPage 显示 runtime 列表
4. `npm test` PASS（无回归）

### 与后续 WP 关系

- 完成后 WP-E 的 packaged smoke 可基于正确的 tarball 进行
- 完成后 WP-B 的 OOM 验证可在 packaged artifact 上进行

---

## 5. Work Package D：参数 help 与用户可理解性

### 覆盖问题

- **RAP-005**：help YAML 存在但 UI 无接入

### 相关性说明

help 文档在 Phase 7 创建了 YAML 文件但未接入 RuntimeParameterEditor UI。

### 目标

1. 每个参数旁显示 help 入口（? icon）
2. vLLM、SGLang、llama.cpp help 可用
3. help 内容展示 summary、default、recommendation、risk

### 执行边界

**修：**
- `RuntimeParameterEditor.vue`：每个参数行添加 ? icon + el-popover
- 可能需要加载 help 数据的 API 或内联方式

**不修：**
- help YAML 内容（已完备）
- en-US help（本轮不补英文）

### 涉及文件

| 文件 | 修改类型 | 风险 |
|------|---------|------|
| `web/src/components/common/RuntimeParameterEditor.vue` | 中等修改 | 需接入 help 数据 |
| Help 数据加载方式 | 待定 | 可通过 API 或编译时内联 |

### 验收标准

1. BackendRuntimesPage 编辑弹窗：每个参数旁有 ? icon
2. Hover/click ? → 显示 help popover（含 summary, recommendation, risk）
3. vLLM/SGLang/llama.cpp 三后端 help 均可显示
4. `npm test` PASS

### 与后续 WP 关系

- 依赖 WP-A 完成后编辑器正常渲染
- 依赖 WP-B 完成后 help popover 不触发 OOM
- WP-E 可添加 help UI 测试

---

## 6. Work Package E：测试体系补强

### 覆盖问题

- **RAP-009**：零浏览器测试
- **RAP-010**：零 packaged artifact smoke
- **RAP-012**：npm test 偏静态源码检查

### 相关性说明

三个问题属同一类：测试体系缺口。一并处理效率更高。

### 目标

1. 增加最小真实 UI smoke（browser-based）
2. 增加 packaged artifact smoke
3. 增加现有静态测试的负向断言（检测重复入口）
4. 增加参数保存/重新打开验证
5. 让后续同类问题能被测试发现

### 执行边界

**修：**
- `web/tests/runtimeBoundaryUi.test.mjs`：增加负向断言
- 新建 `scripts/e2e-packaged-smoke.sh`
- 新建 `scripts/e2e-ui-browser-smoke.sh`（Playwright 或 headless browser）

**不修：**
- 不引入完整 Playwright test suite（只加 smoke）
- 不重构现有测试框架

### 涉及文件

| 文件 | 修改类型 | 风险 |
|------|---------|------|
| `web/tests/runtimeBoundaryUi.test.mjs` | 增强 | 负向断言可能因实现细节变化而脆弱 |
| `scripts/e2e-packaged-smoke.sh` (新建) | 小 | 脚本依赖 packaged artifact 构建 |
| `scripts/e2e-ui-browser-smoke.sh` (新建) | 中 | 需安装 headless browser |

### 验收标准

1. `npm test` 包含负向断言（RunnerConfigsPage 不应含 legacy Docker editor）
2. `e2e-packaged-smoke.sh` 从 tarball 启动 + API 验证成功
3. `e2e-ui-browser-smoke.sh` 能打开页面并检查元素
4. 新增测试在干净环境 PASS

---

## 7. Work Package F：架构遗留项与策略项处理

### 覆盖问题

- **RAP-006**：extra_args 冲突检测策略
- **RAP-007**：DeviceBinding dead struct
- **RAP-008**：RuntimeRequirements / BackendCapabilityProfile 未落地
- **RAP-011**：closeout 文档状态不一致
- **RAP-013**：evidence 目录缺少统一索引
- **RAP-014**：RuntimeParameterEditor Docker 参数列表 hardcoded

### 相关性说明

这些问题属于架构设计 vs 实现的差距，以及文档/策略问题。不是紧急的代码 bug，但需要明确的处置决策。

### 目标

1. 对每个架构遗留项给出处理建议
2. 能直接修复的进入执行计划
3. 需要设计讨论的写清风险和建议方案
4. closeout 文档状态与真实验证一致

### 执行边界

**修：**
- `RAP-011`：标记 closeout REOPENED + 添加交叉引用
- `RAP-007`：删除 DeviceBinding dead code 或实现
- `RAP-013`：更新 evidence README

**不修（只做决策）：**
- `RAP-006`：评估是否升级 extra_args 为 preflight 阻断 → 本轮实现 structured warning，不做 blocking error
- `RAP-008`：评估 RuntimeRequirements 实现时机 → 形成正式架构决策
- `RAP-014`：评估 hardcoded Docker 参数改造为 schema/catalog 驱动的成本和风险

### 涉及文件

| 文件 | 修改类型 | 风险 |
|------|---------|------|
| `docs/reports/phase-3/*.md` | 文档更新 | 无代码风险 |
| `internal/server/runplan/types.go` | 可能删除 dead code | 确认无引用 |
| `docs/reports/README.md` | 更新 evidence 索引 | 无风险 |

---

## 8. Work Package 依赖关系

```
Step 0 (前置): RAP-011 closeout REOPENED
  ↓
WP-A ──→ WP-C ──→ WP-B ──→ WP-D ──→ WP-E
  │                                      │
  └──→ WP-F (独立，可随时进行) ←─────────┘
```

- 执行顺序：Step 0 → WP-A → WP-C → WP-B → WP-D → WP-E → WP-F
- WP-A 先于 WP-B：需要先打通数据流，才能在有真实数据的编辑器中验证 OOM 修复
- WP-C 在 WP-B 之前：确保 OOM 验证在正确的 packaged artifact 上进行
- WP-D 依赖 WP-A + WP-B：需要工作正常且稳定的编辑器才能接入 help
- WP-E 依赖 WP-A/B/C：需要正常工作的代码和打包产物才能写有意义的测试
- WP-F 独立于其他 WP，可随时进行
