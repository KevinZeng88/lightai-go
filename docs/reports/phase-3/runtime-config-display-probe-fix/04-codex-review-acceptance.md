# Codex 核查结论采纳与文档修正说明

## 1. 采纳结论

Codex 的核查结论建议采纳。原文档的边界方向基本正确，但需要按代码事实修正三个重点：

1. P0-1 的主链路不是旧字段 `parameter_schema_json` / `parameter_values_json` / `resource_controls_json`，而是当前的 `config_set_json -> /api/v1/config-edit/view -> config_edit_view.sections -> ConfigEditView`。
2. P0-2 除了前端显示问题，还涉及 catalog runtime YAML 缺少用户可见 `display_name`、clone 前端默认值绕过 display adapter、clone 后端技术 `name` 生成规则，以及用户配置版本展示规则不一致。
3. P0-3 当前没有证据表明 Docker image inspect `.Config.Env` 已污染 `env_json` 或 `ResolvedRunPlan.env`；修复重点应是 raw probe evidence 默认直出、`level4` 开发口径、以及补测试防止后续污染。

## 2. 文档修正方向

### 2.1 P0-1：运行模板详情页参数不显示

原文档中关于旧字段和 SQL SELECT 的判断降级为历史兼容检查。当前首要修复点改为：

- 后端 `/api/v1/config-edit/view` 返回 envelope：`{ config_edit_view, config_view }`。
- 前端 `getConfigEditView()` 当前直接返回整个响应对象，并声明为 `ConfigEditView`。
- `ConfigEditView.vue` 读取 `localView.sections`，实际拿到外层对象时没有 `sections`，导致参数区为空。

首要修复：

- `web/src/api/configEdit.ts` 对 response 做 unwrap：优先返回 `resp.config_edit_view ?? resp`。
- 补前端测试验证 `getConfigEditView()` 能解包 envelope。
- 验证所有使用 `getConfigEditView()` 的页面恢复显示 sections。

### 2.2 P0-2：display_name / name / version 展示错误

修复需要覆盖四点：

1. runtime catalog YAML 补用户可见 `display_name`，至少覆盖：
   - `configs/backend-catalog/runtimes/vllm/nvidia-docker.yaml`
   - `configs/backend-catalog/runtimes/sglang/nvidia-docker.yaml`
   - `configs/backend-catalog/runtimes/llamacpp/nvidia-docker.yaml`
2. clone 弹窗默认值使用 display adapter 后的用户可见名，不直接使用 `row.display_name || row.name`。
3. clone 后端在未显式提供 `name` 时生成稳定技术名，不从用户显示名派生。
4. version 展示规则统一：当前运行模板主 UI 不显示具体 backend version；内置模板和用户配置均显示 `*` 或在列表隐藏版本列。具体 `backend_version_id` 仅作为技术信息保留。

### 2.3 P0-3：probe evidence raw JSON 默认直出

修复重点：

- `RunnerConfigsPage.vue` 默认显示 probe summary。
- raw `probe_results_json` 放在折叠区，默认收起。
- 默认视图不出现完整 `NVIDIA_REQUIRE_CUDA`、`PATH`、`LD_LIBRARY_PATH`。
- 后端 `level4` 去掉 `version probe not yet implemented; deferred to future design` 这类开发口径，改为结构化产品口径。
- 保留 `.Config.Env` 作为 raw evidence，但补测试证明它不进入 configured env 和 RunPlan env。

## 3. Claude 执行前需要采用的更新

Claude 执行时应优先阅读：

1. `01-fix-boundary-and-acceptance.md`
2. `04-codex-review-acceptance.md`
3. `03-claude-execution-prompt.md`

其中 `03-claude-execution-prompt.md` 已按 Codex 结论修正，Claude 应以修正版为准。

## 4. 是否进入执行

建议进入 Claude 执行。

执行限制保持不变：

- 不新建分支。
- 不做大范围架构重构。
- 先修已发现问题，再做限定同类检查，再补最小必要测试。
- 修复后执行测试、提交、推送，并输出 commit、push、`git status --short`。
