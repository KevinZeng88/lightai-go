# Executor Review & Suggestions

> Status: DRAFT — 执行者请在正式执行前补充
> Date: 2026-06-25
> Reviewer: Claude (executor)

---

## 1. 对 Work Package 分组的评价

### WP-A 与 WP-B 耦合度

**评价：分组合理，但存在强顺序依赖。**

WP-A (数据流) 和 WP-B (OOM) 都涉及 `RuntimeParameterEditor.vue`。如果 WP-B 先做，修复后的 syncing guard 可能在 WP-A 引入真实数据后被新的数据流模式打破。建议严格按 A→B 顺序，且 WP-B 的验收必须在 WP-A 修复后的三个页面（都有真实数据）上进行。

**建议：** 维持 A→B 顺序，WP-B 验收明确要求三个页面都有真实 row 数据。

### WP-C 独立性

**评价：可以独立执行，但建议串行。**

WP-C (打包脚本) 与其他 WP 无代码依赖，可以并行。但考虑到打包产物是 WP-B 用于 OOM 验证的载体之一，建议在 WP-A 和 WP-B 之间执行 WP-C，确保后续验证在正确的 packaged artifact 上进行。

**建议：** 执行顺序调整为 A → C → B → D → E → F。理由：WP-C 修复后可在 packaged artifact 中验证 WP-B 的 OOM 修复（更接近真实用户环境）。

### WP-F 策略项

**评价：部分问题需要在 WP-A 之前决策。**

RAP-007 (DeviceBinding dead struct) 和 RAP-008 (RuntimeRequirements) 是纯决策/文档工作，可以随时进行。但 RAP-011 (closeout 不一致) 如果先处理，可以避免在 WP-A 到 WP-E 执行期间引用声称 CLOSED 的文档。

**建议：** 将 RAP-011 (closeout 标记 REOPENED) 提前到 WP-A 之前执行（或作为 WP-A 的前置步骤），确保后续执行者不会被旧 closeout 误导。

---

## 2. 是否发现遗漏问题

### 2.1 RuntimeParameterEditor 的 hardcoded 参数列表

**发现：** `RuntimeParameterEditor.vue:162-181` 的 `scalarOptions` 和 `listOptions` 是 hardcoded 的。当 catalog YAML 新增 Docker 级参数时，这些 hardcoded 列表不会自动更新。

**影响：** 如果未来在 BackendRuntime 的 `docker_json` 中新增字段（例如 `userns_mode`），RuntimeParameterEditor 不会渲染它。

**建议：** 在 WP-D 或 WP-F 中评估是否将 Docker 参数也从 schema 驱动（类似 backend serving args），而非 hardcoded。当前不阻塞 P0 修复，但应在 issue registry 中记录。

**已纳入 issue registry 为 RAP-014。** 参见 `01-current-gap-review.md`。

### 2.2 BackendRuntimesPage clone 功能未覆盖新参数

**发现：** `BackendRuntimesPage.vue:276-281` clone 对话框使用 `parameter_values_json` 直接复制，但未验证 `docker_json` 和其他字段是否完整复制。

**影响：** 如果 clone 源是用户修改过的 BR，新参数可能丢失。

**建议：** 在 WP-A 中一并检查 clone 数据完整性。

### 2.3 三个页面的保存逻辑差异

**发现：** BackendRuntimesPage、RunnerConfigsPage、ModelDeploymentsPage 的 `doEdit()` 保存逻辑各自实现，未共享。

**影响：** 修复 WP-A 时需分别处理三个保存路径，可能遗漏。

**建议：** 在 WP-A 执行时，确保三个页面的保存路径都经过验证。长期考虑抽取公共 `useParameterEditor()` composable。

---

## 3. 问题间依赖关系

### 强依赖

- RAP-001 ↔ RAP-002：同为参数数据流问题，必须在同一 WP (WP-A) 中修复
- RAP-003 依赖 WP-A 结果：需要真实数据才能验证 OOM 修复
- RAP-005 依赖 WP-A + WP-B：需要工作正常的编辑器才能接入 help

### 弱依赖

- RAP-004 独立于 UI 修改
- RAP-006/007/008/011 独立于代码修复

### 建议调整的依赖

- RAP-011 (closeout 标记) → 提前为 WP-A 前置步骤
- RAP-004 (打包) → 移到 WP-B 之前（理由见 §1）

---

## 4. 建议调整的执行顺序

```
Step 0 (前置): RAP-011 — 标记 closeout REOPENED
  ↓
WP-A: RAP-001, RAP-002 — 参数编辑 UI 数据流闭环
  ↓
WP-C: RAP-004 — Catalog、打包与 clean DB 初始化
  ↓
WP-B: RAP-003 — RuntimeParameterEditor 稳定性与 OOM 修复
  ↓
WP-D: RAP-005 — 参数 help 与用户可理解性
  ↓
WP-E: RAP-009, RAP-010, RAP-012 — 测试体系补强
  ↓
WP-F: RAP-006, RAP-007, RAP-008, RAP-013 — 架构遗留项与策略项处理
  ↓
全量回归 + closeout 更新
```

---

## 5. 是否建议合并或拆分 Work Package

### 建议合并

无需合并。当前 6 个 WP 粒度合理，每个可独立验证。

### 建议拆分

WP-F 可拆为两个子步骤：
- **WP-Fa**：文档/策略项 (RAP-006, RAP-008, RAP-011, RAP-013) — 纯文档工作
- **WP-Fb**：代码清理项 (RAP-007 DeviceBinding) — 需要 go test 验证

但拆分的收益不大（WP-F 总体工作量小），建议保持合并。

---

## 6. 更低风险的修复方案

### WP-A: RunnerConfigsPage

**方案 A（推荐）：** 删除 legacy editor + populate editParameterModel
- 风险：legacy ref 变量在其他地方被引用
- 缓解：修复前 grep 所有 legacy ref 引用，确认只在 showEdit/doEdit 中使用

**方案 B（保守）：** 保留 legacy editor 但隐藏 + 同步数据到 RuntimeParameterEditor
- 风险：遗留代码累积技术债
- 建议：不推荐。方案 A 更清晰。

### WP-B: OOM 修复

**方案 A（推荐）：** `nextTick` 后重置 syncing guard + hashKeys 替代 JSON.stringify
- 风险：hashKeys 遗漏关键字段导致参数不同步
- 缓解：hashKeys 覆盖 docker_json key set + parameter_values_json length + customArgs length

**方案 B（保守）：** `flush: 'sync'` 使 syncing guard 在同步时序中生效
- 风险：sync flush 可能影响性能
- 建议：优先尝试方案 A

---

## 7. 需要先验证再修复的假设

| 假设 | 验证方式 | 优先级 |
|------|---------|--------|
| RunnerConfigsPage legacy ref 只在 showEdit/doEdit 中使用 | `grep -n 'editPrivileged\|editIpcMode\|editShmSize\|editDevicesText\|editGroupAddText\|editSecurityOptText\|editUlimitsText' web/src/pages/RunnerConfigsPage.vue` | WP-A 开始前 |
| ModelDeploymentsPage computed 改为 ref 后 setter 逻辑不受影响 | `grep -n 'editParameterModel' web/src/pages/ModelDeploymentsPage.vue` | WP-A 开始前 |
| server.release.yaml 中 catalog 路径指向 `configs/backend-catalog/` | Read `configs/server.release.yaml` | WP-C 开始前 |
| RuntimeParameterEditor 修改不影响 parent 的 commandPreview | 手动测试：修改参数 → 确认 preview 更新 | WP-B 完成后 |
| clean DB 启动 packaged artifact 时 server 正确加载 catalog | `docker run ... && curl` | WP-C 验收 |

---

## 8. 需要用户确认的设计取舍

| 取舍 | 选项 | 建议 | 确认窗口 |
|------|------|------|---------|
| Help 数据加载方式 | A: 新 API endpoint / B: 内联到 schema | 建议 B（简单），但需确认不破坏 schema 结构 | WP-D 开始前 |
| extra_args 冲突检测策略 | A: 升级为 preflight 阻断 / B: 保持 warning | 建议 B（避免破坏性），记录为 deferred | WP-F 执行中 |
| DeviceBinding 处置 | A: 删除 dead code / B: 保留待实现 | 建议 A（YAGNI） | WP-F 执行中 |
| 是否补 en-US help | A: 本轮补 / B: 延后 | 建议 B（zh-CN 已够用） | WP-D 开始前 |

---

## 9. 风险与缓解

| 风险 | 严重度 | 缓解措施 |
|------|--------|---------|
| WP-A 删除 legacy editor 破坏 RunnerConfigsPage 保存功能 | 中 | 修复前理解 doEdit 逻辑；修复后完整测试保存→重开 |
| WP-B syncing guard 修改导致参数编辑失效 | 中 | 双方案备选；每个页面验证参数修改 |
| WP-C 打包脚本修改后 server 找不到 catalog | 低 | 检查 server.release.yaml catalog 路径 |
| WP-D help 数据加载失败阻塞编辑器渲染 | 低 | help 加载失败时静默降级（只不显示 ?） |
| WP-E browser smoke 依赖安装复杂 | 中 | 优先 curl 级别验证；browser smoke 用简单脚本 |

---

## 10. Recommendation

**PROCEED_WITH_ADJUSTMENTS**

建议调整：
1. 将 RAP-011 (closeout REOPENED) 作为前置步骤，在 WP-A 之前完成
2. 将 WP-C 移到 WP-B 之前（A → C → B → D → E → F）
3. WP-D 开始前确认 help 数据加载方式
4. 新增 RAP-014 (hardcoded Docker 参数列表) 到 issue registry
