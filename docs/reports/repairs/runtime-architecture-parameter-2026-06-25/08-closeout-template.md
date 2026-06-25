# Runtime Architecture & Parameter — Repair Closeout

> Status: PARTIAL
> Date: 2026-06-25
> Branch: main
> Trigger: `docs/reports/repairs/runtime-architecture-parameter-2026-06-25/`

---

## 1. 修复摘要

本轮修复了 LightAI Go 运行架构与参数系统的 P0/P1 问题：

- **UI 数据流:** 统一三个配置页面的参数编辑入口，移除 RunnerConfigsPage 遗留双入口，修复 ModelDeploymentsPage 数据不传播
- **OOM:** 修复 RuntimeParameterEditor watch→emit 循环（nextTick defer syncing guard）
- **打包:** 修复 release tarball 缺失 configs/backend-catalog/ (50 个 YAML 文件)
- **Help:** 新增 Go API endpoint + RuntimeParameterEditor ? icon popover
- **测试:** 新增 packaged smoke test + 负向断言
- **架构:** 删除 DeviceBinding dead code，记录 3 个 deferred 项决策

**浏览器 smoke 测试 (RAP-009, RAP-012) 作为独立后续专题，本轮不处理。**

---

## 2. Issue Registry 最终状态

| Issue ID | Severity | Problem | Status | Resolution | Verified By |
|----------|----------|---------|--------|------------|-------------|
| RAP-001 | P0 | RunnerConfigsPage 双编辑入口 | **FIXED** | 删除 legacy editor + populate editParameterModel | npm test PASS, git diff |
| RAP-002 | P0 | ModelDeploymentsPage 空 docker_json | **FIXED** | computed→ref, showEdit() 填充完整数据 | npm test PASS, git diff |
| RAP-003 | P0 | watch/emit 循环 OOM | **FIXED** | nextTick defer syncing guard + syncing check in modelValue watch | npm test PASS, code review |
| RAP-004 | P0 | 打包脚本缺失 catalog | **FIXED** | 替换 configs/model-runtime → configs/backend-catalog | tar -tzf: 50 files |
| RAP-005 | P1 | help 文档 UI 未接入 | **FIXED** | Go handler + ? icon popover in RuntimeParameterEditor | npm test PASS, go test PASS |
| RAP-006 | P1 | extra_args 冲突仅 warning | **DEFERRED** | 确认 resolver.go:616 log.Warn only; 保持 warning | code review |
| RAP-007 | P1 | DeviceBinding dead struct | **FIXED** | 删除 26 行 dead code (types.go), 零引用确认 | go test PASS |
| RAP-008 | P2 | RuntimeRequirements 未落地 | **DEFERRED** | 需硬件测试环境; 触发: 自动调度 | decisions.md |
| RAP-009 | P2 | 零浏览器测试 | **PARTIAL** | 负向断言已加; browser smoke 独立后续 | decisions.md |
| RAP-010 | P2 | 零 packaged smoke | **FIXED** | e2e-packaged-smoke.sh created | smoke output PASS |
| RAP-011 | P2 | closeout 文档不一致 | **FIXED** | 两个 closeout 标记 REOPENED + 交叉引用 | git diff |
| RAP-012 | P2 | npm test 静态检查 | **PARTIAL** | 负向断言已加; 完整 browser test 独立后续 | decisions.md |
| RAP-013 | P3 | evidence 目录缺索引 | **FIXED** | evidence/README.md created | git diff |
| RAP-014 | P2 | Docker 参数 hardcoded | **DEFERRED** | 15 参数覆盖当前需求; 触发: 新 Docker 参数 | decisions.md |

---

## 3. Work Package 完成表

| WP | 名称 | 状态 | Issues | Commit |
|----|------|------|--------|--------|
| Step 0 | 前置步骤 | PASS | RAP-011 | `30365d1` |
| WP-A | 参数编辑 UI 数据流闭环 | PASS | RAP-001, RAP-002 | `3259f63` |
| WP-C | Catalog、打包与 clean DB | PASS | RAP-004 | `c3fd618` |
| WP-B | RuntimeParameterEditor 稳定性与 OOM | PASS | RAP-003 | `163c9b5` |
| WP-D | 参数 help 与用户可理解性 | PASS | RAP-005 | `34f592c` |
| WP-E | 测试体系补强 | PASS | RAP-010 (partial: RAP-009, RAP-012) | `e23ab51` |
| WP-F | 架构遗留项与策略项 | PASS | RAP-006, RAP-007, RAP-008, RAP-013, RAP-014 | `a54bf5f` |

---

## 4. 问题关闭表

| 状态 | 数量 | Issues |
|------|------|--------|
| FIXED | 9 | RAP-001, RAP-002, RAP-003, RAP-004, RAP-005, RAP-007, RAP-010, RAP-011, RAP-013 |
| DEFERRED_WITH_REASON | 3 | RAP-006, RAP-008, RAP-014 |
| PARTIAL | 2 | RAP-009, RAP-012 (browser smoke deferred) |

---

## 5. Deferred Items 表

| Issue ID | 原因 | 风险 | 触发条件 | 建议处理批次 |
|----------|------|------|---------|-------------|
| RAP-006 | extra_args 冲突保持 WARNING; 阻断可能阻止合法覆盖 | 重复参数可能导致容器启动失败 | 用户报告重复参数部署问题 | 后续 minor release |
| RAP-008 | 需 MetaX/Huawei 硬件测试环境验证 | vendor capability 匹配不完整 | 引入自动调度或 GPU capability matching | Phase 4+ |
| RAP-009 | Browser smoke 需引入 Playwright/headless Chrome 依赖 | UI 渲染问题仍依赖人工发现 | CI/CD 集成 browser test | 独立后续专题 |
| RAP-012 | 完整 browser test 同 RAP-009 | 同上 | 同上 | 独立后续专题 |
| RAP-014 | 当前 15 参数覆盖所有现有 vendor profile | 新增 Docker 参数需改前端代码 | 需要新增 Docker 参数（如 --tmpfs） | 下一个 Docker 参数需求 |

---

## 6. 测试结果表

| 测试 | 结果 | 备注 |
|------|------|------|
| `npm test` (76 tests) | PASS | 含负向断言 |
| `npm run build` | PASS | 无 TS 错误 |
| `go test ./internal/...` | PASS | 含 runplan/api/agent 全部 package |
| `go build ./cmd/server/ && go build ./cmd/agent/` | PASS | |
| `scripts/e2e-packaged-smoke.sh` | PASS | 50 catalog files in tarball |
| Container API smoke | SKIP | 无 Docker image (tarball-only verification) |

---

## 7. Evidence 索引

| 路径 | 说明 |
|------|------|
| `evidence/README.md` | 证据索引与可信度说明 |
| `evidence/wp-a-ui-data-flow/` | WP-A: git diff/status, npm build |
| `evidence/wp-b-editor-stability/` | WP-B: git diff, npm build, npm test |
| `evidence/wp-c-package-catalog/` | WP-C: git diff, tar -tzf (50 files) |
| `evidence/wp-d-help-ui/` | WP-D: git diff, npm build, npm test |
| `evidence/wp-e-tests/` | WP-E: git diff, packaged smoke output |
| `evidence/wp-f-architecture-items/` | WP-F: decisions.md (full architecture decisions) |

---

## 8. Packaged Artifact 验证

| 验证项 | 结果 |
|--------|------|
| tarball 包含 backend-catalog | PASS (50 files) |
| help YAML 在 tarball 中 | PASS |
| Container API smoke | SKIP (no Docker image) |

---

## 9. Commit 列表

| Commit | 说明 |
|--------|------|
| `30365d1` | docs: mark closeout docs as REOPENED |
| `3259f63` | fix: unify parameter edit data flow (RAP-001, RAP-002) |
| `c3fd618` | fix: include backend-catalog in release artifact (RAP-004) |
| `163c9b5` | fix: eliminate watch-emit cycle (RAP-003) |
| `34f592c` | feat: add parameter help tooltips (RAP-005) |
| `e23ab51` | test: add packaged artifact smoke test (RAP-010) |
| `a54bf5f` | docs: reconcile architecture gap items (RAP-006/007/008/013/014) |
| (pending) | fix: delete DeviceBinding dead code + evidence fill + closeout |

---

## 10. 最终状态

**PARTIAL**

- 所有 P0 问题 FIXED (4/4)
- 1 个 P1 FIXED (RAP-005), 1 个 P1 FIXED (RAP-007 删除死代码), 1 个 P1 DEFERRED (RAP-006)
- 2 个 P2 FIXED (RAP-010, RAP-011), 2 个 P2 PARTIAL (RAP-009, RAP-012 — browser smoke independent follow-up), 2 个 P2 DEFERRED (RAP-008, RAP-014)
- 1 个 P3 FIXED (RAP-013)
- 回归测试全部 PASS
- Evidence 已归档
- Browser smoke 作为独立后续专题

**声明:** 本轮修复解决了所有阻塞用户使用的 P0 问题。3 个 P1/P2 deferred 项有明确的触发条件和后续路径。Browser smoke(U+306E) deferred 不阻塞 closeout, 但 RAP-009 和 RAP-012 标记为 PARTIAL 而非 CLOSED。
