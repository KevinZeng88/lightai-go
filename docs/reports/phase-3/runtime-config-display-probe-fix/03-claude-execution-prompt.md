# Claude 执行 Prompt

```text
请在当前分支修复 LightAI Go runtime 配置展示问题。不要新建分支。按固定范围执行，修复后测试、提交并推送。

请先阅读：
docs/reports/phase-3/runtime-config-display-probe-fix/01-fix-boundary-and-acceptance.md
docs/reports/phase-3/runtime-config-display-probe-fix/04-codex-review-acceptance.md

目标顺序：
1. 先修复已发现问题。
2. 再做限定范围的同类问题检查。
3. 最后补最小必要测试。

问题 1：运行模板详情页参数不显示
代码事实：
- 当前主链路是 config_set_json -> /api/v1/config-edit/view -> config_edit_view.sections -> ConfigEditView。
- 后端 /api/v1/config-edit/view 返回 { config_edit_view, config_view }。
- 前端 getConfigEditView() 当前需要兼容 envelope，返回 config_edit_view，而不是外层对象。

要求：
- 修复 web/src/api/configEdit.ts，使 getConfigEditView() 返回 resp.config_edit_view ?? resp。
- 确认 ConfigEditView.vue 接收到的对象包含 sections。
- 验证所有调用 getConfigEditView() 的页面，包括运行模板详情、节点运行配置详情、节点配置向导，不再因 envelope 未解包导致参数为空。
- 不要把旧字段 parameter_schema_json / parameter_values_json / resource_controls_json 作为主修复方向；旧字段只做残留风险检查。

验收：
- vLLM / SGLang / llama.cpp 运行模板详情页参数不为空。
- 复制后的用户配置详情页参数不为空。
- 刷新后仍显示。
- 无 OOM / watch emit 循环。

问题 2：复制用户配置后的 display_name / name / version 展示错误
代码事实：
- 核心 runtime YAML 缺少用户可见 display_name，loader 回退到 runtime ID。
- clone 弹窗默认值直接用 row.display_name || row.name 拼“用户配置”，绕过 display adapter。
- clone 后端未显式 name 时用 sourceName + "-copy" 生成技术 name，可能继续带技术名。
- extractVersion() 只对内置模板返回 *；用户配置会从 backend_version_id 提取 v0.23.0 等具体版本。

要求：
- 给 vLLM / SGLang / llama.cpp NVIDIA Docker runtime catalog 补用户可见 display_name。
- 列表主列改为“显示名称”。
- 用户主显示字段使用 display_name 或 display adapter 后的 displayName。
- 技术 name 只放详情或辅助字段。
- clone 弹窗默认 display_name 使用 display adapter 后的用户可见名：源 displayName + " - 用户配置"。
- clone 后端在未显式提供 name 时生成稳定唯一技术名，例如 runtime.<backend>.<vendor>.user.<shortid>，不要从用户显示名派生。
- 当前运行模板没有按具体软件版本形成差异化能力定义，运行模板 / 用户配置主 UI 不显示具体 backend version；统一显示适配版本 * 或列表隐藏版本列。
- 具体 backend_version_id 只作为技术信息或诊断信息保留。

验收：
- 复制后列表显示类似 vLLM NVIDIA Docker - 用户配置。
- 列名为“显示名称”。
- 技术 name 不作为用户主标题。
- 详情页可以看到技术 name。
- 列表不显示 v0.23.0、sglang-v0.5.13.post1、llamacpp-b9700 等具体 backend version。
- 源模板和复制配置的适配版本一致，均为 * 或列表隐藏版本字段。
- 不出现 runtime.vllm.nvidia-docker 作为用户标题。
- 无 i18n key 泄露。

问题 3：节点运行配置探测 JSON 默认展示异常
代码事实：
- 当前没有证据表明 Docker image inspect .Config.Env 已进入 env_json 或 ResolvedRunPlan.env。
- .Config.Env 保存到 probe_results_json.level2.env，属于 raw evidence。
- 真实问题是 RunnerConfigs 详情默认直接展示 raw probe_results_json，用户会看到完整 NVIDIA_REQUIRE_CUDA、PATH、LD_LIBRARY_PATH 等 image metadata。
- level4 仍有开发口径：version probe not yet implemented; deferred to future design。

要求：
- 保持 image config env、configured env、resolved runplan env 三类 env 分离。
- Docker image inspect .Config.Env 只能作为 raw evidence。
- 补测试确认它不得进入 ConfigSet runtime env、Deployment snapshot env、ResolvedRunPlan.env、RuntimeParameterEditor / ConfigEditView 可编辑参数。
- RunnerConfigs 详情默认显示 probe summary，不直接显示完整 raw JSON。
- 摘要包括镜像状态、镜像引用、CUDA 版本、是否存在 NVIDIA CUDA 约束、后端匹配状态、匹配依据、启动方式、启动 profile、置信度、兼容性检查状态、是否阻断部署。
- Raw JSON 放入“诊断原文 / Raw probe evidence”折叠区，默认收起。
- 默认页面不得显示完整 NVIDIA_REQUIRE_CUDA、PATH、LD_LIBRARY_PATH。
- 后端 level4 去掉 deferred to future design / not yet implemented，改为结构化产品口径，例如 compatibility_check_status=not_run、version_probe_status=not_available、blocking=false、message=当前仅完成镜像存在性与后端匹配检查。

验收：
- 节点运行配置详情默认显示摘要。
- Raw JSON 默认收起。
- 默认页面不显示完整 NVIDIA_REQUIRE_CUDA。
- env_json / ConfigSet runtime env 不包含 Docker inspect 原始 Config.Env。
- ResolvedRunPlan.env 不包含 Docker inspect 原始 Config.Env。
- vLLM 最终启动 env 只包含明确需要注入的变量。
- 页面不出现 deferred to future design / not yet implemented。

限定同类检查：
1. 参数字段是否在运行模板、用户配置、节点运行配置、节点配置向导、部署 RunPlan 预览中丢失。
2. display_name / name 是否在运行模板、用户配置、节点运行配置、模型部署、模型实例中混用。部署列表/详情如仍显示 source_node_backend_runtime_id，若已有 display_name 可用则做最小 UI 修复；若需要较大 DTO / 查询改造，在报告中明确记录链路和建议。
3. raw evidence 是否在节点运行配置、镜像检测、preflight、部署诊断、模型实例诊断中默认直出。

最小必要测试：
- Frontend test: getConfigEditView() unwraps config_edit_view。
- Frontend test: runtime template detail renders ConfigEditView sections。
- Frontend test: copied runtime config displays display_name and not runtime.vllm.nvidia-docker。
- Frontend test: user runtime config list/detail does not show v0.23.0; show * or hide version。
- Frontend test: RunnerConfigs detail shows probe summary and raw JSON is collapsed by default。
- Frontend test: default RunnerConfigs detail does not show NVIDIA_REQUIRE_CUDA / PATH / LD_LIBRARY_PATH。
- Go/API test: clone runtime config uses user-visible display_name and stable technical name。
- Go/API test: Docker image inspect Config.Env stays in raw evidence only and does not enter ConfigSet runtime env。
- Go/API test: ResolvedRunPlan.env does not contain NVIDIA_REQUIRE_CUDA / PATH / LD_LIBRARY_PATH unless explicitly configured by user.

测试命令：
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm test
cd web && npm run build

完成后输出：
1. 每个问题的根因。
2. 修改文件清单。
3. 同类问题检查结果。
4. 新增或修改测试。
5. 测试命令和结果。
6. git commit id。
7. push 结果。
8. git status --short。
```
