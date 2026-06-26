# Commit and Push Strategy

## 原则

- 默认不新建分支。
- 每个批次独立 commit。
- 每个批次通过验证后 push。
- 代码、测试、文档同步提交，避免只有代码没有 closeout。
- 如果一个批次很大，可以拆成多个 commit，但必须在 batch closeout 中列出所有 commit。
- 每批必须使用 pathspec-limited `git add <explicit files>`。
- 不允许 `git add .`。
- 不允许把 baseline unrelated files 混入批次提交。

## 推荐 commit message

| 批次 | Commit message |
| --- | --- |
| Batch 0 | `docs: add project-wide repair execution inventory` |
| Batch 1 | `runtime: harden nbr readiness and deployment payload contract` |
| Batch 2 | `runtime: align deployment preflight with runplan resolver` |
| Batch 3 | `docs: converge current api contract and e2e evidence` |
| Batch 4 | `web: repair deployment runtime workflow and nbr loading` |
| Batch 5 | `security: add agent docker policy and tenant guards` |
| Batch 6 | `runtime: harden lifecycle leases logs and recovery` |
| Batch 7 | `perf: reduce nbr fanout and frontend bundle pressure` |
| Batch 8 | `docs: clarify product scope and gateway boundaries` |
| Final | `docs: close project-wide repair execution` |

## 每批提交前

Claude 开始 Batch 0 前必须生成：

```bash
docs/reports/codex-project-wide-execution-plan-20260625/workspace-baseline.md
```

并记录：

```bash
git status --short
git diff --stat
git diff -- web/package.json web/package-lock.json
git log --oneline -30
```

每批提交前执行：

```bash
git status --short
git diff --stat
git diff --check
```

执行本批验证命令。

## 每批提交

```bash
git add <explicit files>
git commit -m "<message>"
git push
```

记录：

```bash
git rev-parse --short HEAD
git status --short
```

## 工作区污染处理

不要删除不属于本批的未知文件。  
如果发现批次开始前已有未跟踪或修改文件：

- 在 batch closeout 记录。
- 不将无关文件混入 commit。
- 如果必须纳入，说明原因。
- `.mimocode/` 默认不提交，除非用户明确要求。
- 旧 E2E evidence 目录不得自动提交，除非某批明确将其归档、标记 historical 或纳入 closeout。
- 如果某批必须修改 baseline 已修改文件，例如 `web/package.json` 或 `web/package-lock.json`，closeout 必须说明为什么它变成 in-scope，并展示 before/after diff。
- commit 前如果出现 unexplained path，必须停止该批并记录，不能强行提交。
- 失败批次不得 push partial implementation code。
- 如果只是 GitHub credentials/network 导致 push 失败，应保留本地 commit，记录 `git push` stderr，最终状态为 `BLOCKED_BY_EXTERNAL_DEPENDENCY`。

## 最终状态

最终 closeout 要求：

- 所有计划文档在指定目录。
- 所有 closeout 在指定目录。
- 所有代码/测试/脚本/OpenAPI 改动已提交并 push。
- `git status --short` 为空，或只剩明确记录的不提交项。
