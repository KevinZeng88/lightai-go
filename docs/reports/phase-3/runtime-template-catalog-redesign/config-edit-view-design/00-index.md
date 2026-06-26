# LightAI Go ConfigEditView 自适应配置编辑设计索引

日期：2026-06-26  
仓库：`KevinZeng88/lightai-go`  
主题：内部 ConfigSet 存储与外部用户编辑模型分离

## 背景

当前 LightAI Go 已经完成 Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / Deployment 的 `config_set_json` 快照化改造，并逐步接入 schema-driven 参数编辑。但手工验证发现：

- 页面仍直接展示 `launcher.xxx`、`runtime.xxx` 等内部 key。
- `launcher.docker_options` 等 object 参数仍以大段 JSON 输入框形式出现。
- BackendVersion、BackendRuntime、NBR、Deployment 多处页面各自处理 ConfigSet，缺少统一投影、编辑、校验和回写逻辑。
- 新增配置项虽然能动态出现，但分组、顺序、label、enabled/required、普通/高级区域等还没有形成统一产品规则。
- 大多数参数应有“启用”勾选框；少数必填参数应默认启用且不可关闭。

因此需要在现有 ConfigSet 基础上新增一层通用抽象：

```text
ConfigSet JSON（内部规范存储）
  -> ConfigEditView（用户可理解的编辑视图）
  -> ConfigEditPatch（用户修改）
  -> ApplyEditPatchToConfigSet（回写内部 ConfigSet）
```

## 文档清单

1. `01-source-review-current-state.md`：当前源码现状、问题边界、为什么不能继续在每个页面硬编码。
2. `02-config-edit-view-architecture.md`：后端 ConfigEditView / ConfigEditPatch / Project / Apply / Validate 的通用架构设计。
3. `03-section-field-taxonomy.md`：配置分组、顺序、字段归属、enabled/required 规则、Docker/模型/运行环境边界。
4. `04-api-frontend-design.md`：API、前端统一组件、页面替换路径、与现有 RuntimeParameterEditor 的迁移关系。
5. `05-implementation-plan-and-acceptance.md`：分批开发步骤、测试要求、验收标准。
6. `06-codex-development-prompt.md`：可直接给 Codex 执行的开发指令。

## 总体判断

推荐采用“自研 ConfigEditView 转换层 + 轻量表单渲染组件”的方式，而不是直接引入一个通用 JSON 表单库替换当前前端。

原因：

- 现有系统的核心复杂度在 copy-on-create、RunPlan、enabled/required、Docker options 子字段拆分、内部 key 与外部字段转换，而不是简单生成输入框。
- Vue/Element Plus 已经是现有前端技术栈；直接引入第三方表单生成器容易导致 UI 样式、校验、i18n 和权限控制分裂。
- JSON Schema / FormKit / JSON Forms 可作为设计参考或局部验证工具，但 LightAI Go 需要自己的领域投影层。
