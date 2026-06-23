# Claude Review Prompt — Runtime Operations UX & Resource Controls

Use this prompt when asking Claude/Codex to inspect the design before implementation.

```text
先不要修改代码。

请基于当前 main 分支，检查以下文件中的 Runtime Operations UX & Resource Controls 设计是否可实施、是否遗漏关键风险、是否与现有代码结构冲突：

1. docs/design/runtime-operations-ux-resource-controls.md
2. docs/reports/phase-3/runtime-operations-ux-resource-controls/00-known-issues-and-evidence.md
3. docs/reports/phase-3/runtime-operations-ux-resource-controls/01-implementation-plan.md
4. docs/reports/phase-3/runtime-operations-ux-resource-controls/02-verification-and-acceptance-plan.md

你的任务是审查设计并输出 review report，不是实现。

背景问题：
1. SGLang Docker 日志出现 torchao SyntaxWarning 和 attention backend default advisory；
2. 模型实例页面状态不会自动更新；
3. 模型测试结果页高级诊断 JSON 在当前位置看不全；
4. 运行配置页面的健康检查 JSON / 高级诊断 JSON 边界不清；
5. 运行配置、模型部署等配置页面应学习“复制为用户配置”的布局；
6. llama.cpp Docker 日志出现 LLAMA_ARG_HOST 被 --host 覆盖；
7. 当前没有清晰显存限制/资源控制建模，也需要明确多个 Docker 是否可共享一张 GPU；
8. 目标是解决此类问题和测试发现机制，不是逐条修补。

请重点检查：

1. 现有代码中是否已经有 RunPlan lint、resource admission、log classifier、JsonViewer、polling composable、ConfigEditorLayout 等类似能力；
2. 设计建议的文件路径是否符合当前项目结构；
3. 是否需要 schema change；
4. BackendVersion capabilities/resource_controls JSON 是否足够承载本批资源控制；
5. vLLM/SGLang/llama.cpp 参数映射是否与当前 catalog/seed/RunPlan resolver 兼容；
6. llama.cpp LLAMA_ARG_HOST/--host 冲突应该由生成器避免还是由 lint 阻断；
7. 模型实例自动刷新是否可以复用现有列表 API，还是需要新增 status-summary API；
8. JsonViewer 是否已有可复用组件；
9. 运行配置页面和模型部署页面是否适合先局部接入 ConfigEditorLayout；
10. 旧配置/旧模板是否有仍在使用的引用，是否可以删除；
11. 测试计划是否足够发现这类问题；
12. 哪些内容应该先做，哪些必须 deferred。

输出要求：

- 总体判断；
- 不合理或过度设计的部分；
- 遗漏的问题；
- 是否需要 schema change；
- 是否需要删除旧逻辑/旧配置；
- 建议的最终实施阶段；
- 每阶段涉及文件；
- 每阶段测试命令；
- 风险与回滚点；
- 需要用户确认的问题；
- 明确写一句：本轮只做设计审查，不修改代码。

禁止：
- 不要修改代码；
- 不要创建 commit；
- 不要新建分支；
- 不要实现完整 arg abstraction 大重构，除非 review 证明没有它无法落地；
- 不要把真实 GPU 重 E2E 作为默认必跑；
- 不要保留旧配置/旧模板/旧兼容脏逻辑。
```
