# Runtime Metadata Hardcode Audit Guidance

## 1. 目的

本文件用于审查 LightAI Go 现有代码中是否存在写死的 formats/tasks/capabilities/endpoints/args/path modes/backend-specific logic。

目标不是机械消灭所有常量，而是区分：

```text
1. 合理常量：enum、validation、test fixture、seed/catalog；
2. 不合理硬编码：preflight/test/RunPlan/UI 中绕过 metadata contract 的特殊判断。
```

## 2. 必须审查的对象

```text
1. model_locations.discovered_metadata_json 所有读写点；
2. model_artifacts.capabilities_json 所有读写点；
3. model_artifacts.capability_sources_json 所有读写点；
4. model_artifacts.default_test_mode 所有读写点；
5. backend_versions.capabilities_json 所有读写点；
6. BackendRuntime / NodeBackendRuntime 中 args/env/ports/health/test 相关读写点；
7. scanner candidate -> artifact/location 保存链路；
8. preflight / CompatibilityChecker 消费字段；
9. RunPlan resolver 中 backend/model/task 判断；
10. TestDispatcher / model test handlers 中 endpoint、payload、test mode 判断；
11. frontend detail/edit/test dialog 中 task/format/capability/endpoints/labels 判断；
12. seed/catalog 中自由 JSON 字段。
```

## 3. 建议搜索命令

```bash
grep -R "supported_formats\|supported_tasks\|supported_capabilities\|test_endpoints\|default_test_mode\|chat/completions\|/v1/embeddings\|/v1/rerank\|--trust-remote-code\|--model-path\|--model\|-m\|gguf\|embedding\|rerank\|vision_chat\|capabilities_json\|discovered_metadata_json" -n internal cmd web configs docs | head -300
```

也可以补充：

```bash
grep -R "vllm\|sglang\|llama.cpp\|llamacpp\|backend ==\|task ==\|format ==\|capability" -n internal cmd web configs | head -300
```

## 4. 分类标准

### 4.1 允许保留的常量

```text
1. enum constants；
2. validation error codes；
3. test fixtures；
4. seed/catalog 默认声明；
5. documented compatibility blockers；
6. i18n key names；
7. stable API path constants if they are declared in BackendCapabilityProfile and code only reads them。
```

### 4.2 需要改造的硬编码

```text
1. preflight 中直接判断 backend == "vllm" && task == "embedding"；
2. CompatibilityChecker 直接读取自由 map 或字符串猜测能力；
3. test handler 不看 BackendCapabilityProfile.test_endpoints，直接调用 /v1/embeddings 或 /v1/rerank；
4. RunPlan 直接因为 task/format 拼 --model、--model-path、-m，而不是通过 abstract arg mapping；
5. scanner 输出未定义 JSON 字段；
6. UI 根据字符串自由判断格式/任务/能力，而不走 enum/i18n；
7. backend capability JSON 中混入 evidence path/container id/local model path；
8. RuntimeRequirements 中混入 backend-specific CLI 参数。
```

## 5. Audit 输出格式

请形成表格：

| Area | File | Pattern | Current Behavior | Risk | Decision | Action |
|---|---|---|---|---|---|---|
| Compatibility | internal/... | task == embedding | bypasses BackendCapabilityProfile | endpoint mismatch | refactored_now | read test_endpoints |
| RunPlan | internal/... | -m | llama.cpp specific arg | backend-specific spread | formal_blocker | arg mapping refactor later |

Decision 只能使用：

```text
refactored_now
validated_as_constant
accepted_catalog_seed
formal_blocker
```

不允许使用：

```text
TODO
later
future
unknown
```

## 6. Formal blocker 要求

如果某个硬编码本批不改，必须写成 formal blocker：

```text
ID:
Title:
Area:
Current behavior:
Why not fixed in this batch:
Risk:
Required future fix:
Validation required:
Owner/status:
```

示例：

```text
ID: RUNTIME-CONTRACT-BLOCKER-001
Title: RunPlan arg mapping still partially backend-specific
Area: internal/server/runplan
Current behavior: llama.cpp -m and vLLM --model are still selected in resolver branches.
Why not fixed in this batch: Full BackendRuntime arg mapping requires broader RunPlan template refactor.
Risk: New backend requires code change instead of catalog-only arg mapping.
Required future fix: Move all backend-specific CLI args to BackendCapabilityProfile/BackendRuntime arg_support.arg_mappings.
Validation required: Add backend fixture proving model_path abstract_arg maps to --model, --model-path, -m without task branches.
Status: formal_blocker
```

## 7. Closeout 要求

最终 closeout 必须说明：

```text
1. 找到哪些 hardcode；
2. 改造了哪些；
3. 哪些保留为合法常量；
4. 哪些属于 seed/catalog；
5. 哪些无法本批改造并形成 formal blocker；
6. 是否还有未分类的 TODO/future；
7. hardcode audit 文档路径。
```
