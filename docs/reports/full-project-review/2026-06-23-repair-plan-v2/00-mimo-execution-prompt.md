# MIMO Execution Prompt — Repair Plan V2 + Future Platform Architecture Constraints

> Saved: 2026-06-23
> Source: User prompt for Repair Plan V2 documentation phase

---

请基于现有 full-project review、evidence triage、issue family analysis、architecture repair plan，以及下面这份指令，再做一次"Repair Plan V2 + Future Platform Architecture Constraints"文档化工作。

本阶段目标不是修代码，而是重新思考、修正、补充、沉淀文档。

请严格遵守：

* 不修改功能代码
* 不做安全修复
* 不做重构实现
* 不进入任何 repair batch
* 不提交修复代码
* 只生成或更新文档
* 所有新文档尽量放在同一个目录
* 文档完成后输出文档路径，供后续审阅

---

# 0. 先沉淀本 prompt

请先创建目录：

`docs/reports/full-project-review/2026-06-23-repair-plan-v2/`

然后把本 prompt 原文保存为：

`docs/reports/full-project-review/2026-06-23-repair-plan-v2/00-mimo-execution-prompt.md`

保存 prompt 后，再按该 prompt 逐步执行分析和文档生成。

最终所有新增文档优先放在：

`docs/reports/full-project-review/2026-06-23-repair-plan-v2/`

---

# 1. 输入文档

请阅读并复核以下已有文档：

* `docs/reports/full-project-review/2026-06-23-full-project-review.md`
* `docs/reports/full-project-review/2026-06-23-evidence-triage.md`
* `docs/reports/full-project-review/2026-06-23-issue-family-analysis.md`
* `docs/reports/full-project-review/2026-06-23-architecture-repair-plan.md`

请不要机械照搬已有结论，要复核其中矛盾、遗漏和执行风险。

---

# 2. 本阶段输出文档建议

请在目录：

`docs/reports/full-project-review/2026-06-23-repair-plan-v2/`

生成以下文档。文档数量不限，但建议至少包括：

1. `00-mimo-execution-prompt.md` — 保存本 prompt 原文。
2. `01-triage-corrections.md` — 修正已有 triage / summary / priority 的矛盾。
3. `02-future-architecture-constraints.md` — 补充未来多服务器、多副本、自动调度、统一 API、审计计费等架构约束。
4. `03-core-abstractions-v2.md` — 重新定义关键抽象。
5. `04-repair-batch-plan-v2.md` — 重新拆分修复批次。
6. `05-open-decisions-and-risks.md` — 列出必须由 reviewer 决策的问题、风险和不应写死的点。

如果你认为合并成更少文档更清晰，也可以合并，但必须保证内容完整，并在最终输出中列出实际生成的文档路径。

---

# 3. 先复核并修正文档内部矛盾

## 3.1 P0/P1 统计矛盾
## 3.2 5.3 Agent token 结论矛盾
## 3.3 8.2 Grafana 默认凭据问题口径

---

# 4. 重新审视现有问题族 A～I

---

# 5. 新增问题族 J：Cluster / Replica / Scheduling / Reconciliation

---

# 6. 新增问题族 K：Unified Model API / Gateway / Audit / Usage / Billing

---

# 7. 修订关键核心抽象

---

# 8. 修订 Batch Plan

---

# 9. 需要避免写死的点

---

# 10. Open Decisions

---

# 11. 最终输出要求

详见用户原始 prompt 完整文本（此处为摘要索引）。
