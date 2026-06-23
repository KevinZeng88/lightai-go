# Runtime Metadata Contract Review Docs

## 目的

这组文档用于让 Claude / Codex 先理解并审查 LightAI Go 的模型类型、运行需求、后端能力、兼容性匹配和现有硬编码问题。当前阶段只要求讨论、审查、提出计划，不直接开发。

## 背景

已完成的模型检测与运行兼容主线状态：

```text
CLOSED_WITH_VLM_BLOCKER
```

已生产级验证能力：

```text
1. HF Chat + vLLM
2. HF Chat + SGLang
3. GGUF Chat + llama.cpp
4. Embedding + vLLM
5. Reranker + vLLM
6. 错误组合 preflight 阻断
7. unsupported assets 可识别、可入库、不可部署
```

仍未生产级验证：

```text
1. VLM / InternVL2_5-1B runtime
2. ONNX serving
3. TensorRT / TensorRT-LLM serving
4. OpenVINO serving
5. Diffusers / Image Generation serving
6. ASR serving
7. TTS serving
8. Classification serving
9. LoRA standalone deployment
```

## 文档清单

```text
00-readme.md
01-model-runtime-contract-design.md
02-mainstream-model-runtime-matrix.md
03-hardcode-audit-guidance.md
04-staged-implementation-and-acceptance.md
05-claude-review-instructions.md
```

## 推荐放置目录

建议放到仓库内：

```text
docs/reports/phase-3/web-ai-config-review/runtime-metadata-contract-review/
```

后续如果设计被接受，可以再将稳定设计沉淀到：

```text
docs/design/model-runtime-contract-and-backend-capability-profile.md
docs/design/model-runtime-mainstream-matrix.md
```

## 当前阶段要求

Claude 只需要：

```text
1. 阅读这组文档；
2. 审查当前代码与文档设计是否冲突；
3. 找出遗漏、风险、过度设计或不合理边界；
4. 给出讨论问题；
5. 给出分阶段实施计划；
6. 暂不修改代码。
```

