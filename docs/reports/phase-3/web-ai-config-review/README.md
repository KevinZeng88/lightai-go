# LightAI Go Web AI 页面与流程重构审议文档

本目录用于给 Claude/Codex 审阅和后续分阶段实施使用。

本轮核心约束：

- 只修改展现方式、导航组织、页面层级、表单组织、已有字段呈现、已有 API 的使用方式。
- 不做数据库 schema 修改。
- 不新增持久化数据结构。
- 不改 Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / ModelDeployment / ModelInstance 的核心语义。
- 如果发现某能力需要数据结构支持，但当前没有字段/API，必须记录为后续事项，不得在本轮擅自迁移。
- 本轮目标是把现有能力整理成客户可理解、可操作、可测试、可诊断的 Web AI 流程。

建议阅读顺序：

1. `00-current-issues-and-product-goals.md`
2. `01-web-ai-flow-and-navigation-design.md`
3. `02-page-configuration-design-no-schema-change.md`
4. `03-staged-implementation-and-acceptance.md`
5. `04-claude-review-and-implementation-instructions.md`
