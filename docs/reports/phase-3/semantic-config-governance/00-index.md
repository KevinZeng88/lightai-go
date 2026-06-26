# LightAI Go Semantic Config Governance 文档包

目的：把 LightAI Go 中分散在 BackendVersion、BackendRuntime、NodeBackendRuntime、Deployment、ModelArtifact、RunPlan、Web 编辑页面里的参数配置统一成一套基础能力，避免同一语义重复建模、各页面各写一套判断、保存/预览/运行链路不一致。

本包不是让 Codex 直接盲改的单点修复 prompt，而是给 Codex 先做设计复审、差距审计、分批实施和验收使用的基础文档。

## 文档列表

1. `01-semantic-config-design.md`
   - 核心设计：唯一语义参数、唯一 owner、copy-on-create 快照、warning 优先、resolver 映射。
   - 解释为什么不能继续使用 `backend.common.host` / `launcher.listen_host` 这类重复字段。

2. `02-cross-entrypoint-audit-scope.md`
   - 需要统一审查的所有参数编辑入口。
   - 明确不能在每个页面单独写判断，必须复用统一程序。

3. `03-implementation-plan.md`
   - 分阶段实施计划。
   - 包括 registry、snapshot、projector、renderer、validator、resolver、migration/DB rebuild、UI 接入。

4. `04-validation-and-test-plan.md`
   - 后端、前端、E2E、RunPlan、UI 手工验证标准。
   - 覆盖参数归属、复制快照、warning、保存、运行预览。

5. `05-codex-review-and-planning-prompt.md`
   - 给 Codex 的第一阶段 prompt：先审查、生成执行计划，不直接改代码。

6. `06-codex-implementation-prompt.md`
   - 给 Codex 的实施 prompt：在计划确认后分批执行。

7. `07-acceptance-checklist.md`
   - 最终验收清单。
   - 包含用户手工验证关注点。

## 使用建议

建议先把本目录复制到仓库：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
mkdir -p docs/reports/phase-3/semantic-config-governance
cp /mnt/data/lightai-semantic-config-governance/*.md docs/reports/phase-3/semantic-config-governance/
```

然后让 Codex 先执行 `05-codex-review-and-planning-prompt.md`，产出审查报告和修复计划。计划确认后，再执行 `06-codex-implementation-prompt.md`。

## 基本原则

- 同一个业务语义只能有一个 canonical semantic key。
- 参数的语义 owner 只有一个。
- 下游对象创建时复制参数快照，复制后下游可改，和上游解除运行时联动。
- 不使用 override 作为主要模型；使用 snapshot / copied_from / dirty / warnings 表达。
- 除硬错误外，参数限制以 warning 展示，不随意阻断保存。
- 所有参数编辑入口必须复用同一套 Semantic Config 程序，不允许页面各自写规则。
- Backend CLI flag / Docker 参数名不直接作为用户配置 key；由 resolver/adapter 从 semantic key 映射生成。
