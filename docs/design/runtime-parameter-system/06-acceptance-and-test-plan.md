# 06 - Acceptance and Test Plan

> Status: active plan
> Date: 2026-06-25
> Governance: see 08-execution-governance-and-decisions.md

## 1. 基础检查（每个 Phase 都必须满足）

```bash
gofmt -w cmd internal
git diff --check
npm run build
npm test
go test ./internal/...
go build ./cmd/server/...
go build ./cmd/agent/...
```

如涉及脚本：

```bash
bash -n <changed-script>
```

## 2. Phase 0 验收：现状审计

**必跑测试**：
```bash
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**Evidence 要求**：
- vLLM E2E 日志 + RunPlan + equivalent command
- SGLang E2E 日志 + RunPlan
- llama.cpp E2E 日志 + RunPlan
- 当前参数链路问题清单

**验收标准**：
- 三后端 E2E 有结果
- baseline evidence 已保存
- 问题清单已记录

**Commit/Push 条件**：evidence / baseline 文档 commit + push。

**停止条件**：三后端 E2E 全部 FAIL 且无法定位原因（非外部硬件/镜像不可控原因）。

---

## 3. Phase 1 验收：参数语义正确性

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/... && go build ./cmd/server/... && go build ./cmd/agent/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**参数 Schema 验收**：

| # | 检查项 | 验证方式 |
|---|--------|---------|
| 1.1 | required 参数不能 disable | UI: checkbox locked on; API: payload 中 disabled 被拒绝或忽略 |
| 1.2 | optional 参数 enabled/value | UI: checkbox + input; API: GET 返回正确 |
| 1.3 | disabled value 保留 | API: 保存 disabled value → GET 返回 |
| 1.4 | disabled 不进入 final config | RunPlan: disabled 参数不在 args 中 |
| 1.5 | Layer 3 模板替换 | Deployment override 中 `{{container_port}}` 能正确替换 |
| 1.6 | host 不允许 Deployment override | API: Deployment override 中 host 被忽略 |
| 1.7 | container_port 默认不允许 override | API: Deployment override 中 container_port 被忽略 |
| 1.8 | required 缺失报结构化错误 | Preflight: 缺失 --host 返回 `required_parameter_missing` |

**Mini Param Trace 验收**：
- BackendVersion schema 正确
- BackendRuntime 继承正确
- NBR 继承正确
- Deployment override 优先级正确
- RunPlan 最终参数正确

**Commit/Push 条件**：代码修改 commit + push。

**停止条件**：
- required 参数仍可被 disable
- disabled value 丢失
- 三后端 E2E 退化

---

## 4. Phase 2 验收：UI 分组和唯一入口

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
```

**UI 验收**：

| # | 检查项 | 验证方式 |
|---|--------|---------|
| 2.1 | host 独立 input | UI 检查 |
| 2.2 | container_port 独立 input | UI 检查 |
| 2.3 | served_model_name 独立 input | UI 检查（如 backend 支持） |
| 2.4 | required 参数无普通 enable checkbox | UI 检查 |
| 2.5 | optional 参数有 enabled/value | UI 检查 |
| 2.6 | high-risk 参数唯一入口 | UI 检查：privileged/ipc/shm 只在一个区域 |
| 2.7 | 无重复编辑入口 | BackendRuntimesPage 无内联 scalarOptions/listOptions |
| 2.8 | Deployment override 勾选后可输入 | UI 测试 |
| 2.9 | 保存后回显 | API 测试 |
| 2.10 | 取消 enabled 不清空 value | API 测试 |
| 2.11 | command preview 与 RunPlan 一致 | 对比测试 |
| 2.12 | 无 i18n key 泄露 | npm test |

**Commit/Push 条件**：代码修改 commit + push。

**停止条件**：
- 重复入口未消除
- Deployment override 仍无法输入

---

## 5. Phase 3 验收：冲突检测

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
go build ./cmd/server/... && go build ./cmd/agent/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
```

**冲突检测验收**：

| # | 检查项 | 验证方式 |
|---|--------|---------|
| 3.1 | extra_args 重复 --host | resolver warning |
| 3.2 | extra_args 重复 --port | resolver warning |
| 3.3 | extra_args 重复 --model | resolver warning |
| 3.4 | required 缺失 | preflight 结构化错误 |
| 3.5 | disabled 进入 final config | RunPlan 验证 |
| 3.6 | vendor 不匹配 | preflight 报错 |
| 3.7 | 无 generic unknown error | preflight response 检查 |

**Commit/Push 条件**：代码修改 commit + push。

**停止条件**：
- 正常参数被误报
- 结构化错误不完整

---

## 6. Phase 4 验收：vendor 隔离

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
```

**Vendor 验收**：

| # | 检查项 | 验证方式 |
|---|--------|---------|
| 4.1 | NVIDIA 无 /dev/dri | RunPlan devices=[] |
| 4.2 | MetaX 仅 MetaX profile | catalog 检查 |
| 4.3 | Huawei 仅 Huawei profile | catalog 检查 |
| 4.4 | 无 vendor 串台 | RunPlan + Docker inspect |
| 4.5 | 未验证模板有 status | catalog 检查 |

**Commit/Push 条件**：catalog 修改 commit + push。

**停止条件**：
- vendor 串台
- NVIDIA E2E 退化

---

## 7. Phase 5 验收：完整参数溯源 E2E

**必跑测试**：
```bash
bash scripts/e2e-model-runtime-param-trace.sh
RUN_REAL_CONTAINER=1 TRACE_BACKEND=vllm bash scripts/e2e-model-runtime-param-trace.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**Evidence 要求**：
```text
00-backend-version.json
01-model-created.json
02-model-modified.json
03-backend-runtime-created-or-cloned.json
04-backend-runtime-modified.json
05-nbr-created.json
06-nbr-modified.json
07-deployment-created.json
08-deployment-override-modified.json
09-preflight.json
10-runplan.json
11-equivalent-command.txt
12-docker-inspect.json
13-final-assertions.txt
14-param-source-table.md
```

**Full Trace 验收**：
- 每层 snapshot 存在
- 每层 GET round-trip
- 继承/覆盖正确
- Deployment override 最高优先级
- disabled value 保留但不进入 final config
- RunPlan 与 Docker inspect 一致
- vLLM modified params 真实运行 PASS
- SGLang / llama.cpp E2E PASS

**Commit/Push 条件**：脚本 + evidence commit + push。

**停止条件**：
- E2E 无法完成全流程
- RunPlan 与 Docker inspect 不一致

---

## 8. Phase 6 验收：矩阵扩展

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**矩阵验收**：
- 三后端常用参数覆盖表完成
- 三厂商 vendor 参数覆盖表完成
- 每个组合标记 verified / requires_hardware_validation / template_only
- 可验证组合 E2E PASS

**Commit/Push 条件**：catalog + 文档 commit + push。

**停止条件**：
- 可验证组合 E2E FAIL
- catalog 与 image help 不一致

---

## 9. Phase 7 验收：外置 help

**必跑测试**：
```bash
npm run build && npm test
```

**Help 验收**：
- help 文档按 backend/version/vendor 分目录
- 每个常用参数有说明
- 每个高风险参数有风险提示
- UI ? 可以展示说明
- help 缺失不影响运行
- 无 i18n key 泄露

**Commit/Push 条件**：help 文档 + UI 修改 commit + push。

**停止条件**：
- help 与 schema 不同步

---

## 10. Autonomous Phase Gate

每个 Phase 完成后：

1. 运行基础检查
2. 运行 Phase 特定测试
3. 保存 evidence
4. 检查验收标准
5. 如全部 PASS → commit + push → 自动进入下一 Phase
6. 如任一 FAIL → 停在当前 Phase → 修复 → 重新验收

## 11. 无人工干预推进规则

- Phase 0 → Phase 1：自动（evidence only）
- Phase 1 → Phase 2：自动（验收通过）
- Phase 2 → Phase 3：自动
- Phase 3 → Phase 4：自动
- Phase 4 → Phase 5：自动
- Phase 5 → Phase 6：自动
- Phase 6 → Phase 7：自动
- Phase 7 → 完成：自动

停止条件触发时暂停，等待人工确认后继续。
