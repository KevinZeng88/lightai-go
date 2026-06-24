# Runtime Parameter System — Final Closeout

> Status: CLOSED
> Date: 2026-06-25
> Branch: main

## 1. Phase 0-7 Commit List

| Phase | Commit | Summary |
|-------|--------|---------|
| Docs | `c2a2f2b` | docs: add runtime parameter system execution plan |
| 0 | `dcd2c2c` | runtime-param: phase 0 - baseline evidence and E2E results |
| Code | `4cb03a7` | runtime-param: fix parameter layering, schema propagation, and UI unification |
| 1 | `099e627` | runtime-param: phase 1 - required locked, Layer 3 template substitution, host/port protection |
| 2 | `f803fdd` | runtime-param: phase 2 - add group/label to schema, group-based rendering |
| 3 | `bceeae1` | runtime-param: phase 3 - add conflict warning to deduplicateArgs |
| 4 | `5fba5ce` | runtime-param: phase 4 - vendor isolation verification confirmed |
| 5 | `c9cdc90` | runtime-param: phase 5 - param trace E2E script and evidence |
| 6 | `076bc9f` | runtime-param: phase 6 - vendor runtime verification status coverage |
| 7 | `b296bf4` | runtime-param: phase 7 - external help documentation for vLLM/SGLang/llama.cpp |
| Fix | `73abf06` | docs: runtime parameter system final closeout |
| Fix | `eabdbef` | runtime-param: SGLang entrypoint correction to sglang serve |
| Fix | `5bc614a` | runtime-param: fix equivalent command preview for multi-element entrypoint |

## 2. Per-Phase Summary

### Phase 0: 现状审计和基线 evidence
- **目标**: 跑三后端 E2E，保存基线 evidence
- **完成项**: vLLM/SGLang/llama.cpp E2E 全部 PASS，baseline evidence 保存
- **Evidence**: `docs/reports/phase-0/evidence/`

### Phase 1: 参数语义正确性最小闭环
- **目标**: required locked, optional enabled/value, Layer 3 模板替换
- **完成项**:
  - required 参数 checkbox locked（disabled when required）
  - required 参数强制 enabled
  - Layer 3 deployment override 执行 substituteVars
  - host/container_port 不允许 Deployment override
  - test helper 创建默认参数值
- **Evidence**: `docs/reports/phase-1/evidence/phase1-report.md`

### Phase 2: UI 分组、唯一编辑入口
- **目标**: 参数按 group 分组，唯一编辑入口
- **完成项**:
  - BackendVersion schema 添加 group/label 字段
  - RuntimeParameterEditor 按 group 分组渲染
  - BackendRuntimesPage 移除重复内联参数编辑
  - Deployment override 传递 backendSchema
- **Evidence**: `docs/reports/phase-2/evidence/phase2-report.md`

### Phase 3: 冲突检测和 preflight 强化
- **目标**: extra_args 冲突检测
- **完成项**: deduplicateArgs 添加冲突警告日志
- **Evidence**: `docs/reports/phase-3/evidence/phase3-report.md`

### Phase 4: vendor 隔离确认
- **目标**: NVIDIA/MetaX/Huawei 参数隔离
- **完成项**: 确认 NVIDIA 无 MetaX devices，MetaX devices 仅在 MetaX profile 下
- **Evidence**: `docs/reports/phase-4/evidence/phase4-report.md`

### Phase 5: 完整参数溯源 E2E
- **目标**: 创建参数溯源 E2E 脚本
- **完成项**:
  - 创建 `scripts/e2e-model-runtime-param-trace.sh`
  - 三后端 param trace 全部 PASS
  - 验证 --host/--port 在 RunPlan 中，无 /dev/dri
- **Evidence**: `docs/reports/phase-5/evidence/param-trace/{vllm,sglang,llamacpp}/`

### Phase 6: 三后端/三厂商矩阵扩展
- **目标**: 扩展参数覆盖和 vendor verification status
- **完成项**:
  - vLLM 18 params, SGLang 13 params, llama.cpp 14 params
  - 19 vendor runtimes 全部有 verification status
  - NVIDIA: verified, MetaX: requires_hardware_validation, Huawei: template_only
- **Evidence**: `docs/reports/phase-6/evidence/phase6-report.md`

### Phase 7: 外置 help 文档
- **目标**: 建立外置 help 文档
- **完成项**:
  - vLLM help: 8 core parameters
  - SGLang help: 6 core parameters
  - llama.cpp help: 6 core parameters
- **Evidence**: `configs/backend-catalog/help/`

## 3. 最终测试结果

| 测试 | 结果 |
|------|------|
| npm run build | PASS |
| npm test | PASS (132 tests) |
| go test ./internal/... | PASS (all packages) |
| go build ./cmd/server/... | PASS |
| go build ./cmd/agent/... | PASS |
| vLLM E2E | PASS |
| SGLang E2E | PASS |
| llama.cpp E2E | PASS |
| vLLM param trace | PASS |
| SGLang param trace | PASS |
| llama.cpp param trace | PASS |

## 4. 参数体系最终状态

| 规则 | 状态 |
|------|------|
| required locked（不可 disable） | ✅ implemented |
| optional enabled/value | ✅ implemented |
| disabled value 保留但不进入 RunPlan | ✅ implemented |
| Layer 3 template substitution | ✅ implemented |
| Deployment override 优先级 | ✅ Deployment > NBR > BR > BV |
| host/container_port 不允许 Deployment override | ✅ implemented |
| extra_args 冲突检测 | ✅ deduplicateArgs warning |
| vendor 隔离 | ✅ NVIDIA 无 MetaX devices |
| help 文档 | ✅ vLLM/SGLang/llama.cpp zh-CN |
| 参数分组 | ✅ startup/performance/parallelism/security/observability/advanced |
| NBR 参数传播 | ✅ BR → NBR with fallback to BV schema |
| 快照 env 原始值 | ✅ buildRuntimeConfigSnapshot 不脱敏 |

## 5. 未真实验证的厂商/硬件项

| 项目 | 状态 | 原因 |
|------|------|------|
| MetaX vLLM | requires_hardware_validation | 无 MetaX GPU 硬件 |
| MetaX SGLang | requires_hardware_validation | 无 MetaX GPU 硬件 |
| MetaX llama.cpp | requires_hardware_validation | 无 MetaX GPU 硬件 |
| MetaX MacaRT | requires_hardware_validation | 无 MetaX GPU 硬件 |
| Huawei vLLM | template_only | 无 Huawei Ascend 硬件 |
| Huawei SGLang | template_only | 无 Huawei Ascend 硬件 |
| Huawei llama.cpp | template_only | 无 Huawei Ascend 硬件 |
| Ascend CANN | template_only | 无 Huawei Ascend 硬件 |

以上均为外部硬件限制，不是代码 bug。

## 6. 待修问题确认

无 P0/P1/P2 待修问题。

## 7. Docker SDK Spec 与 Equivalent Command 关系

| 后端 | Docker SDK | Equivalent Command | 进程 argv 等价 |
|------|-----------|-------------------|---------------|
| vLLM | `Entrypoint=["vllm","serve"]`, `Cmd=["--model",...]` | `--entrypoint vllm image serve --model ...` | ✓ |
| SGLang | `Entrypoint=["sglang","serve"]`, `Cmd=["--model-path",...]` | `--entrypoint sglang image serve --model-path ...` | ✓ |
| llama.cpp | `Entrypoint=[]`, `Cmd=["-m",...]` | `image -m ...` | ✓ |

Docker SDK `Entrypoint` 数组直接映射为进程 argv 前缀，`Cmd` 数组映射为后续参数。Equivalent command 使用 `--entrypoint` 设置第一个元素，其余元素放在 image 之后。两者生成的进程 argv 等价，可复制执行。

## 8. 最终 git log

```
b296bf4 runtime-param: phase 7 - external help documentation for vLLM/SGLang/llama.cpp
076bc9f runtime-param: phase 6 - vendor runtime verification status coverage
c9cdc90 runtime-param: phase 5 - param trace E2E script and evidence
5fba5ce runtime-param: phase 4 - vendor isolation verification confirmed
bceeae1 runtime-param: phase 3 - add conflict warning to deduplicateArgs
f803fdd runtime-param: phase 2 - add group/label to schema, group-based rendering
099e627 runtime-param: phase 1 - required locked, Layer 3 template substitution, host/port protection
4cb03a7 runtime-param: fix parameter layering, schema propagation, and UI unification
dcd2c2c runtime-param: phase 0 - baseline evidence and E2E results
c2a2f2b docs: add runtime parameter system execution plan
```

## 8. 最终 git status

```
(no uncommitted changes)
```
