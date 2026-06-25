# Runtime Architecture & Parameter Repair — Index

> Status: READY_FOR_REVIEW
> Date: 2026-06-25
> Branch: main
> Trigger: User-requested gap review of current code vs design documents vs real UI behavior

---

## 1. 修复背景

用户执行 `./scripts/package-release-docker.sh` 打包运行后，真实 UI 复现以下问题：

1. 运行配置页面停留一段时间后浏览器 Out of Memory
2. 运行配置编辑页仍有重复输入入口（legacy Docker editor + RuntimeParameterEditor 同时渲染）
3. 新增的运行参数在页面中没有明显体现
4. High-risk Docker 参数存在两套编辑入口

同时，对架构设计文档与当前代码进行逐项核对，发现：

- 架构核心链（Backend → BackendVersion → BackendRuntime → NBR → Deployment → RunPlan）已落地
- 参数系统 API 层完备，但 UI 层存在严重数据未传播和重复渲染问题
- 打包脚本未包含 `configs/backend-catalog/` 目录
- 帮助文档 YAML 存在但 UI 无接入
- 测试体系缺少浏览器测试，无法发现前端渲染问题
- 多个 closeout 文档声称完成但真实问题仍存在

**本轮目标：** 汇总所有已发现问题，按相关性分组设计 work package，制定可独立验证的执行计划，准备 closeout 模板和 Claude 自动执行规则。

**本轮边界：** 只沉淀文档、分组方案、执行建议和执行入口，不进入代码修复。

---

## 2. 当前问题摘要

| 严重级别 | 数量 | 关键问题 |
|---------|------|---------|
| P0 | 4 | 双编辑入口、数据不传播、OOM 循环、打包脚本缺失 catalog |
| P1 | 3 | help UI 未接入、extra_args 仅 warning、DeviceBinding dead struct |
| P2 | 5 | 架构抽象未落地、缺浏览器测试、缺 packaged smoke、closeout 不一致、npm test 静态检查 |
| P3 | 1 | evidence 目录缺少统一索引 |

**总计：13 个已识别问题，已全部进入 issue registry。**

---

## 3. 文档清单

| 文档 | 说明 | 读者 |
|------|------|------|
| `00-index.md` (本文) | 修复背景、问题摘要、文档清单、推荐阅读顺序、执行顺序、后续入口 | 所有人 |
| `01-current-gap-review.md` | 最终版差距 review，含完整 issue registry（13 个问题） | 执行者、审核者 |
| `02-work-package-design.md` | 按相关性分组的 6 个 work package 设计 | 执行者 |
| `03-execution-plan.md` | 以 work package 为主线的执行计划 | 执行者 |
| `04-acceptance-criteria.md` | 按 issue 和 work package 两维度的验收标准 | 执行者、审核者 |
| `05-evidence-requirements.md` | 证据目录结构和收集要求 | 执行者 |
| `06-executor-review-and-suggestions.md` | Claude 执行者建议窗口 | 执行者、审核者 |
| `07-claude-autonomous-execution-instructions.md` | Claude 自动执行规则 | Claude |
| `08-closeout-template.md` | 最终 closeout 模板 | Claude、审核者 |

---

## 4. 推荐阅读顺序

### 快速了解（10 分钟）

1. `00-index.md` — 本文
2. `01-current-gap-review.md` — 只读 §1-§2（总结 + 真实问题）和 §12（必须修复清单）
3. `02-work-package-design.md` — 只读 §1（work package 总览表）

### 准备执行（30 分钟）

1. `01-current-gap-review.md` — 完整阅读
2. `02-work-package-design.md` — 完整阅读
3. `03-execution-plan.md` — 完整阅读
4. `04-acceptance-criteria.md` — 完整阅读
5. `06-executor-review-and-suggestions.md` — 阅读并补充建议

### Claude 自动执行前（必读）

1. `07-claude-autonomous-execution-instructions.md` — 必读全文
2. `03-execution-plan.md` — 当前 work package 的执行计划
3. `04-acceptance-criteria.md` — 当前 work package 的验收标准
4. `05-evidence-requirements.md` — 当前 work package 的 evidence 要求

---

## 5. 推荐执行顺序

```
Work Package A: 参数编辑 UI 数据流闭环 (P0)
  ↓
Work Package B: RuntimeParameterEditor 稳定性与 OOM 修复 (P0)
  ↓
Work Package C: Catalog、打包与 clean DB 初始化 (P0)
  ↓
Work Package D: 参数 help 与用户可理解性 (P1)
  ↓
Work Package E: 测试体系补强 (P2)
  ↓
Work Package F: 架构遗留项与策略项处理 (P1/P2)
  ↓
全量回归 + evidence 汇总 + closeout 更新
```

**顺序理由：**
- WP-A 先于 WP-B：需要先打通数据流，才能在有真实数据的编辑器中验证 OOM 修复
- WP-C 可与 WP-A/WP-B 并行（打包脚本独立），但建议串行以减少变量
- WP-D 依赖 WP-A/WP-B（UI 需要正常工作的编辑器才能接入 help）
- WP-E 依赖 WP-A/B/C（需要正常工作的代码和打包产物才能写有意义的测试）
- WP-F 为文档/策略工作，可随时进行，放在重体力工作后

---

## 6. 当前状态

**READY_FOR_REVIEW**

所有文档已生成，等待执行者审核。审核通过后状态变更为 `READY_FOR_EXECUTION`。

---

## 7. 后续执行入口文件

Claude 自动执行时，从这里开始：

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/07-claude-autonomous-execution-instructions.md
```

或由人工执行者从这里开始：

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/03-execution-plan.md
```
