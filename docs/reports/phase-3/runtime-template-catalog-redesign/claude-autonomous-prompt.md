AUTORUN

仓库：

`/home/kzeng/projects/ai-platform-study/lightai-go`

请在当前分支继续执行，不新建分支。

先阅读并严格执行：

`docs/reports/phase-3/runtime-template-catalog-redesign/runtime-template-ux-repair-plan-for-claude.md`

按文档中的 1-13 步顺序修复运行模板 UX 和 ConfigEditView 展示问题。重点是：

1. 复制为用户配置的命名、i18n、复制后显示和选中。
2. 用户配置的编辑、删除、重命名操作。
3. 系统模板高级诊断从 raw JSON 改为只读结构化参数 + 来源摘要 + 原始 JSON 折叠。
4. ConfigField/ConfigEditView 消灭 `[object Object]`，结构化 env/model_mount/health/devices/ports/capabilities。
5. 运行模板页、节点运行配置向导、部署 override、BackendVersion 等页面统一展示规则。
6. 补齐中英文 i18n。
7. 补测试并跑全量验证。

必须运行：

```bash
go build ./cmd/server/...
go build ./cmd/agent/...
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm run build
cd web && npm test
```

更新：

`docs/reports/phase-3/runtime-template-catalog-redesign/final-closeout.md`

新增章节：

`Post-closeout Runtime Template UX and ConfigEditView Display Repair`

完成后：

```bash
git status --short
git add .
git commit -m "web: polish runtime template config editing ux"
git push
```

最终输出：

- PASS/FAIL
- commit id
- push result
- test summary
- closeout path
- remaining blocked items, if any
- git status
