# Claude Review Instructions: Runtime Metadata Contract

## 任务性质

本轮只做设计审查和实施计划，不要修改代码。

请先阅读本目录下所有文档：

```text
00-readme.md
01-model-runtime-contract-design.md
02-mainstream-model-runtime-matrix.md
03-hardcode-audit-guidance.md
04-staged-implementation-and-acceptance.md
```

然后审查当前 LightAI Go 代码，给出审查意见和实施计划。

## 仓库

```text
/home/kzeng/projects/ai-platform-study/lightai-go
```

## 背景状态

当前模型检测与运行兼容主线已经完成：

```text
CLOSED_WITH_VLM_BLOCKER
```

最终 closeout：

```text
docs/reports/phase-3/web-ai-config-review/37-model-detection-runtime-final-closeout.md
```

当前不要再继续扩功能。先讨论 metadata contract 设计是否正确。

## 本轮目标

请输出一份 review report，至少包含：

```text
1. 你对这组文档设计的理解；
2. 你认为设计是否合理；
3. 哪些地方过度设计；
4. 哪些地方仍不够清晰；
5. 当前代码中哪些模块已经符合；
6. 当前代码中哪些模块与设计冲突；
7. 当前 hardcode audit 初步结果；
8. RuntimeRequirements × BackendCapabilityProfile 的落地风险；
9. ResolvedBackendCapability 在当前模型中的实现难点；
10. 分阶段开发建议；
11. 本批建议先改哪些，暂缓哪些；
12. 需要用户确认的问题。
```

## 严格限制

本轮不要：

```text
1. 不改代码；
2. 不新增文档到仓库，除非用户后续批准；
3. 不提交 commit；
4. 不新增 schema/migration；
5. 不新增 backend/runtime；
6. 不重跑 production E2E；
7. 不删除 VLM-RUNTIME-001 blocker；
8. 不把 review 变成实现。
```

## 需要重点审查的代码点

请检查：

```text
1. model_locations.discovered_metadata_json 读写；
2. model_artifacts.capabilities_json 读写；
3. model_artifacts.capability_sources_json 读写；
4. model_artifacts.default_test_mode 读写；
5. backend_versions.capabilities_json 读写；
6. BackendRuntime / NodeBackendRuntime 运行模板；
7. scanner candidate -> artifact/location 保存链路；
8. CompatibilityChecker；
9. preflight；
10. RunPlan resolver；
11. TestDispatcher / model test handlers；
12. frontend detail/edit/test dialog；
13. seed/catalog。
```

## 建议搜索命令

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

grep -R "supported_formats\|supported_tasks\|supported_capabilities\|test_endpoints\|default_test_mode\|chat/completions\|/v1/embeddings\|/v1/rerank\|--trust-remote-code\|--model-path\|--model\|-m\|gguf\|embedding\|rerank\|vision_chat\|capabilities_json\|discovered_metadata_json" -n internal cmd web configs docs | head -300

grep -R "vllm\|sglang\|llama.cpp\|llamacpp\|backend ==\|task ==\|format ==\|capability" -n internal cmd web configs | head -300
```

## Review Report 格式

请按以下结构输出：

```text
# Runtime Metadata Contract Review Report

## 1. Executive Summary

## 2. Design Understanding

## 3. Agreement / Disagreement

## 4. Current Code Alignment

## 5. Current Code Conflicts

## 6. Hardcode Audit Findings

| Area | File | Pattern | Risk | Suggested Decision |
|---|---|---|---|---|

Suggested Decision 只能是：
- refactor_now
- validated_as_constant
- accepted_catalog_seed
- formal_blocker

## 7. RuntimeRequirements Review

## 8. BackendCapabilityProfile Review

## 9. ResolvedBackendCapability Feasibility

## 10. TestDispatcher / RunPlan Refactor Risk

## 11. Suggested Implementation Plan

## 12. Suggested Acceptance Criteria

## 13. Questions for User Approval
```

## 重点判断标准

设计正确的方向：

```text
ModelTypeProfile 是类型规则；
DiscoveredMetadata 是扫描结果；
RuntimeRequirements 是模型侧需求；
BackendCapabilityProfile 是后端侧能力；
ResolvedBackendCapability 是最终可用能力；
VerificationRecord 是真实验证证据；
CompatibilityChecker 做可计算匹配；
RunPlan/TestDispatcher 消费契约，而不是自己猜。
```

设计错误的方向：

```text
1. 在 ModelTypeProfile 里放 /home/kzeng/models/...；
2. 在 RuntimeRequirements 里放 --trust-remote-code；
3. 在 BackendCapabilityProfile 里放 container id 或 production evidence path；
4. 在 TestDispatcher 里硬猜 endpoint；
5. 在 RunPlan 里到处散落 backend-specific arg；
6. preflight 靠 task/backend 字符串 if 判断；
7. frontend 自己维护一套 task/capability 语义。
```

## 最终输出要求

只输出 review report。不要开发。
