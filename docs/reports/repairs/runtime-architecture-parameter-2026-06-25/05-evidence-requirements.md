# Evidence Requirements

> Status: DRAFT
> Date: 2026-06-25

---

## 1. Evidence 目录结构

```
docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/
├── README.md                              # 证据索引与可信度说明
├── wp-a-ui-data-flow/                     # WP-A 证据
│   ├── git-diff-stat.txt                  # git diff --stat
│   ├── git-status.txt                     # git status --short
│   ├── npm-build.txt                      # npm run build 输出
│   ├── npm-test.txt                       # npm test 输出
│   ├── runner-configs-edit-before.png     # 修复前：RunnerConfigsPage 编辑弹窗
│   ├── runner-configs-edit-after.png      # 修复后：RunnerConfigsPage 编辑弹窗（单入口）
│   ├── runner-configs-save-reopen.png     # 保存后重新打开
│   ├── model-deployments-edit.png         # ModelDeploymentsPage 编辑弹窗（Docker 参数可编辑）
│   └── backend-runtimes-edit.png          # BackendRuntimesPage（确认无回归）
├── wp-b-editor-stability/                 # WP-B 证据
│   ├── git-diff-stat.txt
│   ├── npm-build.txt
│   ├── npm-test.txt
│   ├── memory-backend-runtimes-2min.png   # Chrome Memory profiler: BackendRuntimesPage 停留 2 分钟
│   ├── memory-runner-configs-2min.png     # Chrome Memory profiler: RunnerConfigsPage 停留 2 分钟
│   └── memory-model-deployments-2min.png  # Chrome Memory profiler: ModelDeploymentsPage 停留 2 分钟
├── wp-c-package-catalog/                  # WP-C 证据
│   ├── git-diff-stat.txt
│   ├── tar-tzf-catalog.txt                # tar -tzf | grep backend-catalog 输出
│   ├── curl-backends-clean-db.json        # clean DB 启动后 curl /api/v1/inference-backends
│   └── ui-backend-runtimes-list.png       # UI BackendRuntimesPage 列表
├── wp-d-help-ui/                          # WP-D 证据
│   ├── git-diff-stat.txt
│   ├── npm-build.txt
│   ├── npm-test.txt
│   ├── help-vllm.png                      # vLLM 参数 help popover 截图
│   ├── help-sglang.png                    # SGLang 参数 help popover
│   └── help-llamacpp.png                  # llama.cpp 参数 help popover
├── wp-e-tests/                            # WP-E 证据
│   ├── git-diff-stat.txt
│   ├── npm-test-output.txt                # npm test 完整输出（含新负向断言）
│   ├── packaged-smoke-output.txt          # e2e-packaged-smoke.sh 输出
│   └── browser-smoke-output.txt           # e2e-ui-browser-smoke.sh 输出
├── wp-f-architecture-items/               # WP-F 证据
│   ├── git-diff-stat.txt
│   ├── go-test-output.txt                 # go test ./internal/... 输出
│   ├── decisions.md                       # 架构决策记录
│   └── closeout-updates.md                # closeout 文档更新记录
├── final-regression/                      # 全量回归证据
│   ├── e2e-real-smoke-all-three.txt
│   ├── e2e-param-trace.txt
│   ├── e2e-packaged-smoke.txt
│   ├── e2e-matrix-verifier.txt
│   ├── e2e-dryrun-matrix.txt
│   ├── npm-test.txt
│   ├── npm-build.txt
│   └── go-test.txt
└── closeout/                              # 最终 closeout 证据
    ├── issue-registry-final.md            # issue registry 最终状态
    ├── wp-completion-table.md             # work package 完成表
    ├── test-results.md                    # 测试结果汇总
    ├── packaged-verification.md           # packaged artifact 验证
    └── final-status.md                    # 最终状态声明
```

---

## 2. 每个 WP 必须收集的证据

| WP | 必须证据 |
|----|---------|
| WP-A | git diff --stat, git status --short, npm build 输出, npm test 输出, UI 截图（修复前/后 RunnerConfigsPage, ModelDeploymentsPage, BackendRuntimesPage） |
| WP-B | git diff --stat, npm build 输出, npm test 输出, Chrome DevTools Memory profiler 截图（三个页面 × 2 分钟停留） |
| WP-C | git diff --stat, tar -tzf 输出（grep backend-catalog）, curl 输出（clean DB 启动）, UI 截图 |
| WP-D | git diff --stat, npm build 输出, npm test 输出, UI 截图（三个后端 help popover） |
| WP-E | git diff --stat, npm test 完整输出, packaged smoke 输出, browser smoke 输出 |
| WP-F | git diff --stat, go test 输出, 决策记录文档, closeout 更新记录 |
| 全量回归 | 所有 E2E 脚本输出, npm test, go test |
| Closeout | issue registry 最终状态, WP 完成表, 测试结果汇总, 最终状态声明 |

---

## 3. Evidence 文件命名规范

- 文本输出：`<description>.txt`
- JSON 输出：`<description>.json`
- UI 截图：`<page>-<action>.png`
- 浏览器内存截图：`memory-<page>-<duration>.png`
- Curl 输出：`curl-<endpoint>.json`
- 测试输出：`<test-suite>-output.txt`
- Git 信息：`git-diff-stat.txt`, `git-status.txt`

---

## 4. Evidence 可信度说明

每个 evidence 文件在 README.md 索引中标注：

- **A 级**：自动化输出（测试、CI、脚本输出）— 可直接复现
- **B 级**：手动截图/录屏 — 有截图时间戳但不可自动复现
- **C 级**：观察描述 — 文字描述无截图

优先使用 A 级 evidence。B/C 级需在索引中说明原因。

---

## 5. Evidence README 模板

```markdown
# Evidence Index

## WP-A: 参数编辑 UI 数据流闭环

| 文件 | 级别 | 说明 | 时间 |
|------|------|------|------|
| git-diff-stat.txt | A | git diff --stat | 2026-06-25 |
| npm-build.txt | A | npm run build | 2026-06-25 |
| npm-test.txt | A | npm test | 2026-06-25 |
| runner-configs-edit-before.png | B | 修复前截图 | 2026-06-25 |
| runner-configs-edit-after.png | B | 修复后截图 | 2026-06-25 |
| ... | | | |
```
