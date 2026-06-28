# Runtime 配置展示与 Probe Evidence 修复文档索引

建议目录：

```text
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/phase-3/runtime-config-display-probe-fix/
```

## 文档清单

| 文件 | 用途 |
|---|---|
| `01-fix-boundary-and-acceptance.md` | 修订后的修复边界、问题定义、验收标准、同类问题检查范围 |
| `02-codex-review-prompt.md` | 给 Codex 的轻量核查 prompt；已执行，可作为审查记录保留 |
| `03-claude-execution-prompt.md` | 修订后的 Claude 执行 prompt，已吸收 Codex 代码链路核查结论 |
| `04-codex-review-acceptance.md` | Codex 核查结论采纳说明、文档修正点、执行建议 |

## 当前结论

Codex 核查结论已采纳。执行前必须使用修订后的 `03-claude-execution-prompt.md`，不要再使用早期基于旧字段链路的判断。

## 建议流程

1. 将本目录放入项目文档目录。
2. 让 Claude 先阅读 `01-fix-boundary-and-acceptance.md`、`04-codex-review-acceptance.md`、`03-claude-execution-prompt.md`。
3. Claude 按 `03-claude-execution-prompt.md` 执行修复、限定同类检查、最小测试。
4. Claude 完成后，重点验收：已发现问题是否修复、Codex 指出的真实根因是否处理、同类问题是否检查、测试是否覆盖、是否 commit/push/status clean。
