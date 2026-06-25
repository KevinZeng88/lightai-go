# Runtime Architecture & Parameter — Repair Closeout

> Status: TEMPLATE — 执行完成后填写
> Date: [执行完成日期]
> Branch: [执行分支]
> Trigger: `docs/reports/repairs/runtime-architecture-parameter-2026-06-25/`

---

## 1. 修复摘要

[简要描述本轮修复了什么，解决了什么问题]

---

## 2. Issue Registry 最终状态

| Issue ID | Severity | Problem | Status | Resolution | Verified By |
|----------|----------|---------|--------|------------|-------------|
| RAP-001 | P0 | RunnerConfigsPage 双编辑入口 | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-002 | P0 | ModelDeploymentsPage 空 docker_json | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-003 | P0 | watch/emit 循环 OOM | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-004 | P0 | 打包脚本缺失 catalog | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-005 | P1 | help 文档 UI 未接入 | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-006 | P1 | extra_args 冲突仅 warning | [FIXED/DEFERRED] | [描述] | [evidence 文件] |
| RAP-007 | P1 | DeviceBinding dead struct | [FIXED/DEFERRED] | [描述] | [evidence 文件] |
| RAP-008 | P2 | RuntimeRequirements 未落地 | [DEFERRED] | [描述] | [evidence 文件] |
| RAP-009 | P2 | 零浏览器测试 | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-010 | P2 | 零 packaged smoke | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-011 | P2 | closeout 文档不一致 | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-012 | P2 | npm test 静态检查 | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-013 | P3 | evidence 目录缺索引 | [FIXED/...] | [描述] | [evidence 文件] |
| RAP-014 | P2 | Docker 参数 hardcoded | [FIXED/DEFERRED] | [描述] | [evidence 文件] |

---

## 3. Work Package 完成表

| WP | 名称 | 状态 | Issues | Commit |
|----|------|------|--------|--------|
| WP-A | 参数编辑 UI 数据流闭环 | [PASS/FAIL] | RAP-001, RAP-002 | [hash] |
| WP-B | RuntimeParameterEditor 稳定性与 OOM | [PASS/FAIL] | RAP-003 | [hash] |
| WP-C | Catalog、打包与 clean DB | [PASS/FAIL] | RAP-004 | [hash] |
| WP-D | 参数 help 与用户可理解性 | [PASS/FAIL] | RAP-005 | [hash] |
| WP-E | 测试体系补强 | [PASS/FAIL] | RAP-009, RAP-010, RAP-012 | [hash] |
| WP-F | 架构遗留项与策略项 | [PASS/FAIL] | RAP-006, RAP-007, RAP-008, RAP-011, RAP-013 | [hash] |

---

## 4. 问题关闭表

| 状态 | 数量 | Issues |
|------|------|--------|
| FIXED | [N] | [list] |
| DEFERRED_WITH_REASON | [N] | [list] |
| BLOCKED | [N] | [list] |
| OPEN (遗留) | [N] | [list] |

---

## 5. Deferred Items 表

| Issue ID | 原因 | 风险 | 触发条件 | 建议处理批次 |
|----------|------|------|---------|-------------|
| RAP-006 | [如保持 warning] | [重复参数可能影响容器行为] | [用户报告因重复参数导致的部署问题] | [后续 minor release] |
| RAP-008 | [无 MetaX/Huawei 硬件验证环境] | [vendor capability 匹配不完整] | [引入自动调度或 GPU capability matching] | [Phase 4+] |

---

## 6. 测试结果表

| 测试 | 结果 | 备注 |
|------|------|------|
| `cd web && npm test` | [PASS/FAIL] | [N tests] |
| `cd web && npm run build` | [PASS/FAIL] | |
| `go test ./internal/...` | [PASS/FAIL] | |
| `bash scripts/e2e-real-smoke-all-three.sh` | [PASS/FAIL] | |
| `bash scripts/e2e-model-runtime-param-trace.sh` | [PASS/FAIL] | |
| `bash scripts/e2e-packaged-smoke.sh` | [PASS/FAIL] | |
| `bash scripts/e2e-matrix-verifier.sh` | [PASS/FAIL] | |
| `bash scripts/e2e-dryrun-parameter-matrix-enhanced.sh` | [PASS/FAIL] | |

---

## 7. Evidence 索引

| 文件 | 说明 |
|------|------|
| `evidence/wp-a-ui-data-flow/` | [WP-A evidence 列表] |
| `evidence/wp-b-editor-stability/` | [WP-B evidence 列表] |
| `evidence/wp-c-package-catalog/` | [WP-C evidence 列表] |
| `evidence/wp-d-help-ui/` | [WP-D evidence 列表] |
| `evidence/wp-e-tests/` | [WP-E evidence 列表] |
| `evidence/wp-f-architecture-items/` | [WP-F evidence 列表] |
| `evidence/final-regression/` | [全量回归 evidence 列表] |

---

## 8. Packaged Artifact 验证

| 验证项 | 结果 | Evidence |
|--------|------|----------|
| tarball 包含 backend-catalog | [PASS/FAIL] | `tar -tzf` 输出 |
| Clean DB 启动后 API 返回 backend 列表 | [PASS/FAIL] | `curl` 输出 |
| BackendRuntimesPage 显示 runtime 列表 | [PASS/FAIL] | UI 截图 |
| 参数编辑无重复入口 | [PASS/FAIL] | UI 截图 |
| 页面停留 2 分钟无 OOM | [PASS/FAIL] | Memory profiler 截图 |
| Help popover 可用 | [PASS/FAIL] | UI 截图 |

---

## 9. Clean DB 验证

| 验证项 | 结果 | Evidence |
|--------|------|----------|
| `rm -f data/lightai.db` 后启动 | [PASS/FAIL] | |
| vLLM backend 可加载 | [PASS/FAIL] | |
| SGLang backend 可加载 | [PASS/FAIL] | |
| llama.cpp backend 可加载 | [PASS/FAIL] | |

---

## 10. UI 验证

| 验证项 | 结果 | Evidence |
|--------|------|----------|
| RunnerConfigsPage 单入口 | [PASS/FAIL] | Screenshot |
| RunnerConfigsPage 参数数据显示 | [PASS/FAIL] | Screenshot |
| ModelDeploymentsPage Docker 参数可编辑 | [PASS/FAIL] | Screenshot |
| BackendRuntimesPage 无回归 | [PASS/FAIL] | Screenshot |
| 保存后重新打开数据一致 | [PASS/FAIL] | Screenshot |

---

## 11. 浏览器内存验证

| 验证项 | 结果 | Evidence |
|--------|------|----------|
| BackendRuntimesPage 停留 2 分钟 | [PASS/FAIL] | Memory profiler |
| RunnerConfigsPage 停留 2 分钟 | [PASS/FAIL] | Memory profiler |
| ModelDeploymentsPage 停留 2 分钟 | [PASS/FAIL] | Memory profiler |

---

## 12. Commit 列表

| Commit | 说明 |
|--------|------|
| [hash] | [message] |
| [hash] | [message] |

---

## 13. Push 结果

```
[git push 输出]
```

---

## 14. Git Status

```
[git status --short]
```

---

## 15. 最终状态

**状态选项：**

- `CLOSED` — 所有 P0/P1/P2 问题 FIXED 或 DEFERRED_WITH_REASON；全量回归 PASS；evidence 完备
- `PARTIAL` — P0 全部 FIXED，但部分 P1/P2 DEFERRED；回归 PASS
- `REOPENED` — 发现问题未修复或新问题出现
- `BLOCKED` — 有外部阻塞因素

**本轮状态：** [CLOSED / PARTIAL / REOPENED / BLOCKED]

**声明：** [总结本轮修复成果和剩余风险]
