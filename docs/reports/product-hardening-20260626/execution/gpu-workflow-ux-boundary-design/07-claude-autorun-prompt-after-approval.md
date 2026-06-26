# 07 — Claude AUTORUN Prompt After Approval

Use this only after Claude's understanding report is reviewed and accepted.

```text
批准执行 GPU workflow UX boundary repair。

请严格按照以下文档执行：

- docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/00-index.md
- docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/01-product-boundaries-and-user-mental-model.md
- docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/02-current-ux-problems-and-root-causes.md
- docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/03-target-ux-design-by-page.md
- docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/04-implementation-plan.md
- docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/05-validation-and-acceptance.md

执行原则：

1. 模型线、运行线、部署线必须保持边界清楚。
2. 用户配置运行环境，不是在编辑 ConfigSet。
3. 内部 ConfigSet key 不得出现在普通用户表单。
4. 运行模板页默认展示用户可理解的 runtime template 名称，例如 nvidia.sglang / nvidia.vllm / nvidia.llama.cpp b9700。
5. 节点运行配置向导必须支持 reset、配置名称、用户参数表单、保存/检测错误停留。
6. 模型库和节点运行配置可共享 NodeSelectorTable，但业务语义分别是模型所在节点和运行环境节点。
7. 模型部署只允许 ready / ready_with_warnings NBR，且必须校验模型位置与 NBR 节点匹配。
8. 不实现 Gateway / API Key / Usage / Billing。
9. 不做旧版本兼容 fallback。

建议按以下 commit 执行：

1. docs: audit gpu workflow ux boundaries
2. fix: reset runtime config wizard and add config naming
3. fix: simplify runtime template presentation
4. fix: add human runtime parameter form
5. fix: align model library node selector and deployment compatibility UX
6. test: add gpu workflow ux regression evidence

每个 commit 前必须运行对应 targeted tests。最终必须运行：

```bash
go test ./...
go build ./cmd/server/... ./cmd/agent/...
cd web && npm test
cd web && npm run build
cd ..
git diff --check
git status --short
```

证据写入：

```text
docs/reports/product-hardening-20260626/evidence/<TS>/gpu-workflow-ux-boundary/
```

最终报告必须包含：

- root cause
- 用户流程变化
- 修改文件
- 测试结果
- 手工 Web 验证结果
- evidence 路径
- commit ids
- push result
- final git status
- 确认未实现 Gateway/API Key/Usage/Billing
```

