# Claude Autonomous Execution Instructions

> Status: DRAFT
> Date: 2026-06-25
> Target executor: Claude (via Claude Code)

---

## 1. 执行入口

```
读取本文件 → 读取 00-index.md → 按 §3 执行循环
```

---

## 2. 每个 Work Package 的执行流程

### Step 1: 准备

```
1. Read docs/reports/repairs/runtime-architecture-parameter-2026-06-25/00-index.md
2. Read 01-current-gap-review.md（只读当前 WP 涉及的 issue）
3. Read 02-work-package-design.md（只读当前 WP 的设计）
4. Read 03-execution-plan.md（只读当前 WP 的执行计划）
5. Read 04-acceptance-criteria.md（只读当前 WP 的验收标准）
```

### Step 2: 检查

```
1. 确认当前 WP 涉及的所有 issue 状态为 OPEN
2. 确认依赖的前置 WP 已验收通过
3. 如发现新的问题或依赖，先更新 issue registry
4. 如发现需要用户确认的取舍，先 AskUserQuestion
```

### Step 3: 修改

```
1. 读取待修改文件的当前内容
2. 执行修改（Edit / Write）
3. 对照 03-execution-plan.md 确认所有修改点已覆盖
4. 确认修改范围未超出当前 WP 边界
```

### Step 4: 验证

```
1. 运行必跑测试（见 03-execution-plan.md 当前 WP 的测试命令）
2. 如有 UI 验证要求，提醒用户手动验证
3. 如测试失败，不超过两轮修复尝试
4. 两轮后仍失败 → 停止并汇报
```

### Step 5: Evidence

```
1. 保存 git diff --stat
2. 保存测试输出
3. 保存其他 evidence（见 05-evidence-requirements.md）
4. 更新 evidence/README.md 索引
```

### Step 6: 状态更新

```
1. 更新 01-current-gap-review.md 中相关 issue 状态
2. WP 验收标准满足 → issue 状态 OPEN → FIXED
3. 不满足 → 保持 OPEN，记录未满足项
```

### Step 7: Commit

```
1. 确认满足 commit 条件（见 03-execution-plan.md）
2. 确认 git diff 与 WP 范围一致
3. 确认未发现新的阻塞性设计问题
4. Commit 使用指定格式
5. 不 push（等全部 WP 完成）
```

### Step 8: 继续

```
1. 当前 WP 验收通过 → 继续下一个 WP
2. 当前 WP 未通过 → 回到 Step 3
3. 全部 WP 完成 → 进入 §4 全量回归
```

---

## 3. 自动继续条件

Claude 可以自动继续下一个 WP，当且仅当：

1. ✅ 当前 WP 所有验收标准满足
2. ✅ 相关 issue 状态已更新为 FIXED
3. ✅ 必跑测试全部 PASS
4. ✅ Evidence 已保存
5. ✅ git diff 与 WP 范围一致（无意外修改）
6. ✅ Commit 已创建
7. ✅ 未发现新的 P0 问题
8. ✅ 未触发 §5 的停止条件

---

## 4. 全量回归流程

所有 WP 完成后：

```
1. 运行全量测试套件（见 03-execution-plan.md §8）
2. 保存 evidence 到 evidence/final-regression/
3. 填写 closeout 模板（08-closeout-template.md）
4. 更新 issue registry 最终状态
5. 创建最终 commit（如需要）
6. git push
```

---

## 5. 停止条件

以下任一条件触发时，Claude 必须停止并汇报用户：

| 停止条件 | 汇报内容 |
|---------|---------|
| 发现新的 P0 问题 | 问题描述 + 证据 + 建议处置 |
| 需要改变 WP 目标 | 原始目标 + 为什么需要改变 + 建议 |
| Clean DB packaged artifact 启动失败，两轮内未定位 | 失败日志 + 已尝试修复 + 下一步建议 |
| 浏览器 OOM 仍复现（WP-B 完成后） | Memory profiler 截图 + 可能原因 |
| 测试失败超过两轮仍未定位 | 失败输出 + 已尝试修复 |
| 需要引入新的大规模重构 | 为什么需要 + 范围评估 |
| 需要用户确认产品/架构取舍 | 取舍选项 + 建议 |

---

## 6. Commit 格式

```
Step 0: docs(runtime-params): mark runtime parameter closeout docs as REOPENED
WP-A:  fix(runtime-params): unify parameter edit data flow across config pages
WP-C:  fix(package): include backend-catalog in release artifact
WP-B:  fix(runtime-params): eliminate watch-emit cycle in RuntimeParameterEditor
WP-D:  feat(runtime-params): add parameter help tooltips to RuntimeParameterEditor
WP-E:  test(runtime-params): add ui and packaged smoke coverage
WP-F:  docs(runtime-params): reconcile architecture gap items and closeout status
```

每个 commit 以 `Co-Authored-By: Claude <noreply@anthropic.com>` 结尾。

---

## 7. 执行检查清单

### 前置步骤（WP-A 之前）

- [ ] Read `00-index.md`, `01-current-gap-review.md`, `02-work-package-design.md`
- [ ] Step 0: 标记 RAP-011 closeout REOPENED（两个文档）
- [ ] 验证假设：grep legacy ref 引用范围
- [ ] Commit

### WP-A

- [ ] Read WP-A 执行计划 (`03-execution-plan.md §2`)
- [ ] Read WP-A 验收标准 (`04-acceptance-criteria.md §1 RAP-001/002, §2 WP-A`)
- [ ] 修改 RunnerConfigsPage.vue
- [ ] 修改 ModelDeploymentsPage.vue
- [ ] `npm run build && npm test`
- [ ] 保存 evidence
- [ ] 更新 issue status
- [ ] Commit

### WP-C

- [ ] Read WP-C 执行计划 (`03-execution-plan.md §3`)
- [ ] 检查 `configs/server.release.yaml` catalog 路径
- [ ] 修改 `scripts/package-release.sh`
- [ ] 打包 + 验证 tarball 内容
- [ ] Clean DB 启动验证
- [ ] 保存 evidence
- [ ] 更新 issue status
- [ ] Commit

### WP-B

- [ ] Read WP-B 执行计划 (`03-execution-plan.md §4`)
- [ ] 修改 RuntimeParameterEditor.vue（syncing guard）
- [ ] 检查并移除父组件冗余 commandPreview
- [ ] `npm run build && npm test`
- [ ] 手动内存验证（三个页面 × 2 分钟）
- [ ] 保存 evidence
- [ ] 更新 issue status
- [ ] Commit

### WP-D

- [ ] Read WP-D 执行计划 (`03-execution-plan.md §5`)
- [ ] 确认 help 数据加载方式（AskUserQuestion 如需）
- [ ] 修改 RuntimeParameterEditor.vue（help popover）
- [ ] `npm run build && npm test`
- [ ] 手动验证三后端 help popover
- [ ] 保存 evidence
- [ ] Commit

### WP-E

- [ ] Read WP-E 执行计划 (`03-execution-plan.md §6`)
- [ ] 增强 runtimeBoundaryUi.test.mjs
- [ ] 创建 e2e-packaged-smoke.sh
- [ ] 创建 e2e-ui-browser-smoke.sh
- [ ] 运行新测试
- [ ] 保存 evidence
- [ ] Commit

### WP-F

- [ ] Read WP-F 执行计划 (`03-execution-plan.md §7`)
- [ ] 处理 RAP-006/007/008 决策
- [ ] 更新 closeout 文档
- [ ] 保存 evidence
- [ ] Commit

### 全量回归

- [ ] 运行全部 E2E
- [ ] 运行全部测试
- [ ] 填写 closeout
- [ ] 最终 commit + push

---

## 8. 输出格式

每个 WP 完成后输出：

```
## WP-X Complete

**Status:** PASS / FAIL
**Issues:** RAP-XXX: FIXED, RAP-YYY: FIXED
**Modified files:** file1, file2
**Tests:**
- npm test: PASS (N tests)
- go test: PASS
**Evidence:** evidence/wp-x-*/ 
**Commit:** <commit-hash> <commit-message>
**Continue:** YES / NO (reason)
```
