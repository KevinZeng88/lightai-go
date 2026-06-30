# ConfigEdit 防回归工程设计包 — Index

## 目的

本设计包用于在 LightAI Go 中系统性补强 ConfigEdit 共享组件与参数模板对象模型相关的测试，避免近期反复出现的参数编辑、显示层级、enabled 分组、raw JSON、i18n、页面行为不一致等问题。

重点不是新增零散测试，而是建立一组可长期维护的 **ConfigEdit 行为契约测试**。ConfigEdit 是运行模板、节点运行配置、部署配置等多个功能的公共参数编辑底座，因此测试必须优先覆盖共享组件和共享工具，而不是只覆盖单个页面。

## 文档顺序

1. `01-current-state-review.md`  
   基于当前仓库文件的测试现状审查，说明已经有什么、缺什么、哪些测试规则当前是错误的。

2. `02-configedit-behavior-contract.md`  
   定义 ConfigEdit 组件必须遵守的产品/架构不变量，尤其是 enabled、view level、risk、section、patch、i18n 的契约。

3. `03-regression-test-suite-design.md`  
   设计测试分层、测试文件、测试用例和每类测试要防止的回归风险。

4. `04-executable-implementation-plan.md`  
   给 Codex 分阶段执行的具体步骤，先修正规则，再补测试，再跑验证。

5. `05-acceptance-and-ci-gates.md`  
   验收标准、CI 门禁、测试命令、禁止通过方式。

6. `06-codex-autonomous-execution-prompt.md`  
   可直接给 Codex 的自包含执行 prompt。

## 核心结论

当前最重要的规则修正是：

```text
enabled=true 的参数全部进入“已启用参数”，包括 high-risk / expert / raw / diagnostic。
enabled=false 的参数再按 常用 / 高级 / 专家 分组。
```

高风险参数不能因为属于 expert/security/raw 而留在后面的“专家参数”。已经启用的危险配置必须前置显示，让用户一进页面就看到它正在生效。风险控制应通过字段自身的风险标识、专家标识、说明文案和视觉告警完成，而不是通过隐藏在后面的专家分组完成。

## 目标目录建议

建议把本设计包放到：

```bash
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/phase-3/configedit-regression-engineering/
```

## 执行策略

- 先提交文档，确认后再让 Codex 开发。
- 不做历史兼容。
- 不新建分支，除非用户明确要求。
- 不靠页面级硬编码绕过共享组件问题。
- 不把 ConfigEdit 的核心行为散落到各页面里。
- 所有关键规则必须由测试锁死。
