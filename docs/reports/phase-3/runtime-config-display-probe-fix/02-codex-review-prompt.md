# Codex 轻量核查 Prompt

```text
请对 LightAI Go 当前 runtime 配置展示问题做代码链路核查，只输出修复建议和验收点，不修改代码，不提交。

请先阅读：
docs/reports/phase-3/runtime-config-display-probe-fix/01-fix-boundary-and-acceptance.md

当前实际测试发现三个问题：
1. 运行模板详情页参数不显示。
2. 复制运行模板为用户配置后，用户主标题混用了技术 name；当前还显示 v0.23.0，但 catalog 没有按具体 vLLM/SGLang/llama.cpp 软件版本形成差异化模板，用户侧应显示适配版本 * 或列表不显示版本。
3. 节点运行配置详情默认展示完整 probe JSON，其中包含 Docker image inspect .Config.Env，例如完整 NVIDIA_REQUIRE_CUDA、PATH、LD_LIBRARY_PATH 等；这些只能作为 raw evidence，不能作为用户运行配置，也不能默认展示。

请核查：
1. BackendRuntime detail/list API 是否返回 parameter_schema_json / parameter_values_json / resource_controls_json 等字段。
2. 前端运行模板详情页是否只使用列表行数据，是否重新 fetch detail。
3. RuntimeParameterEditor 是否在详情页正确接收 schema/value。
4. clone runtime config 的 display_name/name/version_requirement 处理逻辑。
5. 运行模板、用户配置、节点运行配置列表是否混用 display_name 和 name。
6. catalog/seed 中 BackendVersion 或 version 字段当前是否被用户界面误展示为具体版本。
7. Docker image inspect .Config.Env 是否进入 env_json、Deployment snapshot、ResolvedRunPlan.env、RuntimeParameterEditor。
8. 节点运行配置详情页是否默认直接展示 raw probe_evidence_json。
9. 是否有 deferred to future design / not yet implemented 这类开发口径进入用户页面。
10. 需要补哪些最小测试。

输出：
1. 每个问题的代码根因判断。
2. 涉及文件清单。
3. 推荐修复步骤。
4. 明确边界，避免大范围重构。
5. 验收标准。
6. 建议测试命令。
```
