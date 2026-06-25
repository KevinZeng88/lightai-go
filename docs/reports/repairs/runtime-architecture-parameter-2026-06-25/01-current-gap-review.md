# Runtime Architecture & Parameter — Current Gap Review (Final)

> Status: FINAL
> Date: 2026-06-25
> Source: `docs/reports/phase-3/runtime-architecture-and-parameter-current-gap-review.md`

---

## 1. 总结判断

**总体结论：架构核心链已落地，但 UI 层存在严重重复渲染和数据未传播问题；参数系统 API 层完备，但 help 文档未接入 UI；closeout 文档声称完成但真实 UI 仍有 P0 问题；测试体系没有浏览器测试，无法发现前端 bug。**

核心问题排序：
1. **P0**：RunnerConfigsPage 同时渲染两套 Docker 编辑入口，且 RuntimeParameterEditor 未收到数据
2. **P0**：ModelDeploymentsPage 的 editParameterModel computed 永远返回空 docker_json
3. **P0**：RuntimeParameterEditor 存在 watch→emit→watch 循环，结合 parent 每次创建新对象引用，导致 OOM
4. **P0**：打包脚本未包含 `configs/backend-catalog/`，拷贝空目录 `configs/model-runtime/`

---

## 2. 当前真实问题

### 2.1 运行配置页面停留后浏览器 OOM

**确认存在。** 根因链：

1. **deep watch→emit→watch 循环**：`RuntimeParameterEditor.vue:389` deep watch 监听 scalarOptions/listOptions → `buildOutput()` → `emit('update:modelValue')` → parent 更新 ref → `props.modelValue` watch 触发 → `loadFromModel()` → 修改 reactive state → 再次触发 deep watch
2. **ModelDeploymentsPage computed 每次返回新对象引用**：`ModelDeploymentsPage.vue:346-348` `get: () => ({docker_json:{}, ...})` 每次渲染创建新对象
3. **syncing guard 在 Vue 3 异步 flush 下无效**：`syncing` flag 在 `finally` 清除后，下一个 microtask 的 modelValue watch 拿到 `syncing=false`
4. **JSON.stringify 大对象每周期执行**：`RuntimeParameterEditor.vue:396` 对完整 modelValue 对象序列化

### 2.2 运行配置编辑页重复输入入口

**确认存在。** 具体：

| Legacy 入口 (RunnerConfigsPage.vue) | RuntimeParameterEditor 入口 |
|---|---|
| 特权模式 (line 238) | privileged (scalarOptions) |
| IPC 模式 (line 241) | ipc_mode (scalarOptions) |
| 共享内存大小 (line 242) | shm_size (scalarOptions) |
| Devices (line 234) | devices (listOptions) |
| Group Add (line 235) | group_add (listOptions) |
| Security Opt (line 236) | security_opt (listOptions) |
| Ulimits (line 243) | ulimits (listOptions) |

RunnerConfigsPage `showEdit()` (lines 541-565) 只 populate legacy 字段，不设置 `editParameterModel.value`。

### 2.3 新增运行参数未显示

**确认存在。** 根因：RunnerConfigsPage 的 `showEdit()` 不 populate `editParameterModel`，ModelDeploymentsPage 的 computed getter 永远返回 `docker_json:{}`。BackendRuntimesPage 实现正确，参数应显示。

### 2.4 打包脚本 catalog 缺失

`package-release.sh:238-239` 拷贝 `configs/model-runtime/*` → 此目录为空。实际 catalog 在 `configs/backend-catalog/`，未被拷贝。

---

## 3. 架构设计 vs 当前代码差距表 (Line A)

| # | 设计要求 | 状态 | 差距 |
|---|---------|------|------|
| 1 | BackendVersion 硬件无关 | ✅ FULFILLED | — |
| 2 | BackendRuntime 承载 runtime/image/docker/vendor | ✅ FULFILLED | — |
| 3 | NBR 唯一可部署 | ✅ FULFILLED | — |
| 4 | Deployment API 拒绝 backend_runtime_id | ✅ FULFILLED | — |
| 5 | enable/check 必要流程 | ✅ FULFILLED | — |
| 6 | check-request Server 代理 Agent 验证 | ✅ FULFILLED | — |
| 7 | preflight 只接受 node_backend_runtime_id | ✅ FULFILLED | — |
| 8 | Deployment/NBR copy-on-create | ✅ FULFILLED | — |
| 9 | RunPlan NBR + Deployment override | ✅ FULFILLED | — |
| 10 | RuntimeRequirements/BackendCapabilityProfile | ❌ NOT FULFILLED | Go 类型不存在 |
| 11 | DeviceBinding/AcceleratorIds 接入 | ⚠️ PARTIAL | DeviceBinding dead struct |
| 12 | vendor 参数隔离 | ✅ FULFILLED | — |
| 13 | 旧 runtime 字段/fallback 残留 | ✅ FULFILLED | — |
| 14 | 打包产物使用最新 catalog | ❌ NOT FULFILLED | 打包脚本 bug |

---

## 4. 参数设计 vs 当前代码差距表 (Line B)

| # | 设计要求 | 状态 | 差距 |
|---|---------|------|------|
| 1 | required locked-on | ✅ FULFILLED | — |
| 2 | optional enabled/value | ✅ FULFILLED | — |
| 3 | disabled value 保留 | ✅ FULFILLED | — |
| 4 | copy/clone 保留 enabled/value | ✅ FULFILLED | — |
| 5 | Deployment override 最高优先级 | ✅ FULFILLED | — |
| 6 | Layer 3 template substitution | ✅ FULFILLED | — |
| 7 | host 禁止 Deployment override | ✅ FULFILLED | — |
| 8 | container_port 默认禁止 override | ✅ FULFILLED | — |
| 9 | extra_args 冲突检测 | ⚠️ PARTIAL | 仅 warning，非 preflight 阻断 |
| 10 | vendor-specific 过滤 | ✅ FULFILLED | — |
| 11 | SGLang sglang serve | ✅ FULFILLED | — |
| 12 | equivalent command 可复制 | ✅ FULFILLED | — |
| 13 | help 文档 ? tooltip | ❌ NOT FULFILLED | YAML 存在 UI 无接入 |
| 14 | 新参数在 UI 中显示 | ⚠️ PARTIAL | 链正确但 UI 数据不 populate |
| 15 | schema 传播链 | ✅ FULFILLED | — |

---

## 5. 测试可信度

| 测试类别 | 发现 UI 问题能力 | 发现打包问题能力 | 结论 |
|---------|----------------|----------------|------|
| npm test (7 static files) | ❌ 不能 | ❌ 不能 | 只检查源码字符串 |
| Go unit tests | ❌ 不能 | ❌ 不能 | 只测后端逻辑 |
| E2E (21 curl scripts) | ❌ 不能 | ❌ 不能 | 绕过前端，不从 tarball 启动 |
| 人工浏览器验证 | ✅ 能 | ✅ 能 | 唯一能发现问题的途径 |

---

## 6. Issue Registry

### P0 — 阻塞真实用户使用（立即修复）

| Issue ID | Severity | Area | Problem | Evidence | Affected Files | Root Cause | Related Issues | Suggested WP | Proposed Fix | Acceptance Criteria | Status |
|----------|----------|------|---------|----------|---------------|-------------|---------------|-------------|-------------|---------------------|--------|
| RAP-001 | P0 | UI | RunnerConfigsPage 同时渲染 legacy Docker editor 和 RuntimeParameterEditor，后者从未收到数据 | `RunnerConfigsPage.vue:232-249` 两套入口；`showEdit():541-565` 不设置 editParameterModel | `web/src/pages/RunnerConfigsPage.vue` | Phase 2 修复了 BackendRuntimesPage 但遗漏了 RunnerConfigsPage | RAP-002, RAP-003 | WP-A | 删除 legacy editor (lines 232-243)；showEdit() 中 populate editParameterModel | 编辑弹窗只有一套参数入口；数据正确显示 | OPEN |
| RAP-002 | P0 | UI | ModelDeploymentsPage editParameterModel computed 永远返回空 docker_json | `ModelDeploymentsPage.vue:346-348` get: () => ({docker_json:{}, ...}) | `web/src/pages/ModelDeploymentsPage.vue` | Phase 2 用 computed 替代了 ref，但 getter 未从 row 读取 docker_json | RAP-001, RAP-003 | WP-A | 改 computed 为 ref；showEdit() 从 row 填充完整数据 | Docker 参数可编辑、可保存、可重新打开验证 | OPEN |
| RAP-003 | P0 | UI | RuntimeParameterEditor watch→emit→watch 循环导致 OOM | `RuntimeParameterEditor.vue:389` deep watch → buildOutput → emit → modelValue watch:395 → loadFromModel → 回到 deep watch | `web/src/components/common/RuntimeParameterEditor.vue` | syncing guard 在 Vue 3 异步 flush 下无效；ModelDeploymentsPage computed 每次创建新引用加剧循环 | RAP-001, RAP-002 | WP-B | 修复 syncing guard + 优化 modelValue watch 比较策略 | 页面停留 2 分钟无持续内存增长 | OPEN |
| RAP-004 | P0 | Packaging | package-release.sh 未包含 configs/backend-catalog/ | `package-release.sh:238-239` 拷贝 configs/model-runtime/*（空目录）而非 configs/backend-catalog/ | `scripts/package-release.sh` | configs/model-runtime/ 目录已废弃但脚本未更新 | RAP-009 | WP-C | 添加 `cp -r configs/backend-catalog "$BUILD_DIR/configs/"` | tarball 包含 catalog YAML；clean DB 启动后 API 返回 backend 列表 | OPEN |

### P1 — 功能不完整（尽快修复）

| Issue ID | Severity | Area | Problem | Evidence | Affected Files | Root Cause | Related Issues | Suggested WP | Proposed Fix | Acceptance Criteria | Status |
|----------|----------|------|---------|----------|---------------|-------------|---------------|-------------|-------------|---------------------|--------|
| RAP-005 | P1 | UI | help YAML 文件存在（3 后端 × zh-CN）但 UI 无 ? icon/popover 接入 | `RuntimeParameterEditor.vue` grep help/tooltip/popover 零结果；help YAML 在 `configs/backend-catalog/help/` | `web/src/components/common/RuntimeParameterEditor.vue` | Phase 7 创建了 help YAML 但未实现 UI 接入 | — | WP-D | 添加 ? icon + el-popover + help 数据加载 | 每个参数旁有 ? icon；hover 显示 help 内容 | OPEN |
| RAP-006 | P1 | Backend | extra_args 冲突检测仅 WARNING log，非 preflight 阻断 | `resolver.go:616` log.Warn 而非返回 structured error | `internal/server/runplan/resolver.go` | 设计要求 preflight 阻断但实现降级为 warning | — | WP-F | 评估是否需要升级为 preflight error；或保持 warning + 文档说明 | 决策记录 + 实现（如需要） | OPEN |
| RAP-007 | P1 | Backend | DeviceBinding struct 定义但从未 populate 或 consume | `types.go:78-102` 完整定义；grep 确认无代码创建/读取 | `internal/server/runplan/types.go` | 设计时定义了抽象但实现时未使用 | RAP-008 | WP-F | 评估后删除 dead code 或实现实际逻辑 | 决策记录 + 实现（如需要） | OPEN |

### P2 — 测试体系缺口 / 技术债（本轮规划，后续执行）

| Issue ID | Severity | Area | Problem | Evidence | Affected Files | Root Cause | Related Issues | Suggested WP | Proposed Fix | Acceptance Criteria | Status |
|----------|----------|------|---------|----------|---------------|-------------|---------------|-------------|-------------|---------------------|--------|
| RAP-008 | P2 | Architecture | RuntimeRequirements / BackendCapabilityProfile Go 类型不存在于源码中 | grep 全 internal/ 零结果；设计文档 03-core-abstractions-v2.md 有完整定义 | 无对应实现 | 设计时定义了抽象但未排入实现计划 | RAP-007 | WP-F | 评估后在 preflight 中逐步引入或标记为 deferred | 决策记录 | OPEN |
| RAP-009 | P2 | Testing | 零浏览器测试 — 所有测试都是 static source analysis 或 curl API | `web/tests/` 7 文件全 static；`scripts/e2e-*` 21 脚本全 curl | `web/tests/`, `scripts/e2e-*` | 项目从未建立浏览器测试基础设施 | RAP-010, RAP-011, RAP-012 | WP-E | 添加最小 Playwright/headless browser smoke | browser smoke 能发现重复 UI 入口和 OOM | OPEN |
| RAP-010 | P2 | Testing | 零 packaged artifact smoke — E2E 都从源码编译，从不验证 release tarball | 所有 E2E 脚本 go build 或连接已运行 server | `scripts/e2e-*` | 测试流程未包含打包验证步骤 | RAP-004, RAP-009 | WP-E | 添加 e2e-packaged-smoke.sh | packaged smoke 能发现 catalog 缺失 | OPEN |
| RAP-011 | P2 | Docs | closeout 文档状态与真实代码/UI 不一致 | `runtime-parameter-system-final-closeout.md` 声称 CLOSED 无待修问题；`runtime-parameter-layering-final-closeout.md` OOM fix 声称完成但 guard 无效 | `docs/reports/phase-3/` | 测试无法发现 UI 问题 → closeout 基于不完整验证 | — | WP-F | 标记 REOPENED + 交叉引用本 repair | closeout 状态与真实验证一致 | OPEN |
| RAP-012 | P2 | Testing | npm test 偏静态源码检查，无法覆盖真实 UI 行为 | `runtimeBoundaryUi.test.mjs` 只做字符串 includes() 检查 | `web/tests/runtimeBoundaryUi.test.mjs` | 测试设计时选择了低成本静态检查而非运行时测试 | RAP-009 | WP-E | 增强现有测试 + 添加负向断言 + 补 browser smoke | 静态测试能检测重复入口；browser smoke 覆盖运行时 | OPEN |
| RAP-014 | P2 | UI / Schema | RuntimeParameterEditor Docker 参数列表 hardcoded — scalarOptions/listOptions 是固定数组，新增 Docker 级参数（如 userns_mode）不会自动出现在编辑器中 | `RuntimeParameterEditor.vue:162-181` scalarOptions 和 listOptions 硬编码定义 | `web/src/components/common/RuntimeParameterEditor.vue` | 设计初将 Docker 参数与 backend serving args 分离，但 Docker 参数未走 schema/catalog 驱动 | RAP-001, RAP-005 | WP-F | 本轮评估改造成本；如低则迁移到 schema/catalog 驱动；如高则记录决策 + 后续路径 + 触发条件 | 有明确处理结论；后续扩展路径已记录 | OPEN |

### P3 — 低优先级（可延后）

| Issue ID | Severity | Area | Problem | Evidence | Affected Files | Root Cause | Related Issues | Suggested WP | Proposed Fix | Acceptance Criteria | Status |
|----------|----------|------|---------|----------|---------------|-------------|---------------|-------------|-------------|---------------------|--------|
| RAP-013 | P3 | Docs | evidence 目录较多但缺少统一索引和可信度说明 | `docs/reports/model-runtime-node-wizard/e2e-*` 大量目录无 README | `docs/reports/` | E2E 自动输出未整理 | — | WP-F | 更新 evidence 目录 README 或清理 | evidence 目录有索引 | OPEN |

---

## 7. Closeout 文档可信度评估

| Closeout 文档 | 声称状态 | 实际状态 | 虚假项 | 建议 |
|---|---|---|---|---|
| `runtime-parameter-system-final-closeout.md` | CLOSED, 无 P0/P1/P2 | **REOPENED** | OOM 声称修复但 UI 仍复现；help 声称完成但 UI 无接入 | 标记 REOPENED |
| `runtime-parameter-layering-final-closeout.md` | CLOSED | **REOPENED** | OOM fix 描述 code-level fix 但 guard 逻辑无效；Phase 2 声称去重但遗漏 RunnerConfigsPage | 标记 REOPENED |
| `batch-b-frontend-runtime-ux-closeout.md` | CLOSED | **UNCERTAIN** | 需确认是否覆盖 RunnerConfigsPage | 审核后决定 |

---

## 8. 可以继续作为设计依据的文档

| 文档 | 可信度 | 原因 |
|------|--------|------|
| `docs/design/runtime-parameter-system/01-parameter-layering-design.md` | ✅ 可信 | 参数分层设计正确，代码已按此实现 |
| `docs/design/runtime-parameter-system/02-backend-vendor-parameter-catalog.md` | ✅ 可信 | catalog 结构设计正确 |
| `docs/reports/full-project-review/2026-06-23-repair-plan-v2/03-core-abstractions-v2.md` | ✅ 可信 | 架构抽象定义正确 |
| `docs/design/runtime-parameter-system/08-execution-governance-and-decisions.md` | ✅ 可信 | 治理决策仍然有效 |
| `docs/reports/phase-3/runtime-parameter-system-final-closeout.md` | ❌ 不可信 | 与代码/UI 不一致 |
| `docs/reports/phase-3/runtime-parameter-layering-final-closeout.md` | ❌ 不可信 | OOM fix 未实际解决 |
