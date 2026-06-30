# Claude 执行指令：全面审查并修复配置渲染与向导按钮易用性

请在 LightAI Go 仓库当前分支直接执行审查、修复、测试、提交、推送。默认项目路径：`/home/kzeng/projects/ai-platform-study/lightai-go`。不要新建分支。

## A. 背景

用户在 Web 页面“节点运行配置”中配置运行配置，进入“镜像与参数”步骤后，结构化参数区域大量字段只显示“配置项”。这些字段实际有稳定 key，例如 `model_runtime.gpu_memory_utilization`、`model_runtime.max_model_len`、`launcher.image`、`runtime.health` 等。用户疑问：此前已经讨论过参数渲染应由公共模块统一处理，为什么这里还会出问题。

同一页面中，“上一步 / 下一步”位于长参数表单底部，“取消”位于 Dialog footer。滚动长表单时操作不方便，操作区也分散。用户希望按钮固定在相应页面头部或统一可见位置，并从通用易用性角度处理。

## B. 核心目标

1. 确认结构化参数渲染是否由公共组件/公共 normalizer 统一处理。
2. 找出“配置项”退化显示的真实根因。
3. 在公共层修复 label/tooltip/field metadata 解析，覆盖所有使用公共配置编辑器的页面。
4. 统一长表单向导操作区，让主要按钮在滚动时持续可见。
5. 补齐单测、集成测试和必要的 i18n leak/label audit。
6. 生成审查与 closeout 文档，记录根因、修复范围、测试结果、commit、push、剩余风险。可修复的问题必须修复完成。

## C. 结构化参数渲染审查要求

请先做只读审查，输出当前代码链路结论，然后实施修复。

重点回答：

1. 当前公共配置编辑器组件是谁？调用路径有哪些？
2. `RunnerConfigsPage` 的“镜像与参数”步骤是否使用该公共组件？
3. 字段 key 如何从后端 schema 到前端 DOM？
4. 字段 label 当前从哪里来？优先级是什么？
5. 为什么已经有 `data-internal-key`，可见 label 仍然显示“配置项”？
6. 哪些页面共用同一渲染链路？哪些页面仍有私有渲染逻辑？
7. 后端 seed / API DTO / 前端 i18n / 前端 normalizer 是否存在命名不一致？

修复时建立统一字段解析契约。建议字段显示优先级：

1. `label_i18n_key` 命中 i18n。
2. `label` / `title` 中的本地化结构命中当前 locale。
3. `label` / `title` 字符串。
4. 已知 internal key 的公共 label 字典。
5. 开发环境显示带 key 的诊断 fallback，例如 `未命名配置项: model_runtime.gpu_memory_utilization`；生产环境也要避免静默显示大量“配置项”。

tooltip/help 显示优先级同理：

1. `tooltip_i18n_key` / `help_i18n_key`。
2. `description_i18n_key`。
3. `help` / `description` 本地化结构。
4. `help` / `description` 字符串。
5. 已知 internal key 的公共 tooltip 字典。

请补齐至少以下 key 的中文和英文 label，并给关键字段补 tooltip：

- `launcher.image`
- `model_runtime.gpu_memory_utilization`
- `model_runtime.max_model_len`
- `model_runtime.dtype`
- `model_runtime.tensor_parallel_size`
- `model_runtime.max_num_batched_tokens`
- `model_runtime.pipeline_parallel_size`
- `model_runtime.cpu_offload_gb`
- `model_runtime.download_dir`
- `model_runtime.enforce_eager`
- `model_runtime.host`
- `model_runtime.kv_cache_dtype`
- `model_runtime.max_num_seqs`
- `model_runtime.model`
- `model_runtime.port`
- `model_runtime.safetensors_load_strategy`
- `model_runtime.swap_space`
- `model_runtime.trust_remote_code`
- `launcher.docker_options`
- `runtime.model_mount`
- `runtime.env`
- `service.listen_host`
- `service.container_port`
- `backend.extra_args`
- `launcher.command`
- `launcher.devices`
- `launcher.entrypoint`
- `runtime.extra_env`
- `launcher.kind`
- `launcher.ports`
- `deployment.served_model_name`
- `launcher.volumes`
- `runtime.health`

字段 label 建议采用面向用户的短中文，例如：

- `GPU 显存利用率`
- `最大上下文长度`
- `数据类型`
- `张量并行数`
- `最大批处理 Token 数`
- `流水线并行数`
- `CPU Offload 容量`
- `模型下载目录`
- `强制 eager 模式`
- `监听地址`
- `KV Cache 数据类型`
- `最大并发序列数`
- `模型路径`
- `服务端口`
- `权重加载策略`
- `Swap 空间`
- `信任远程代码`
- `Docker 选项`
- `模型挂载`
- `环境变量`
- `服务监听地址`
- `容器端口`
- `附加启动参数`
- `启动命令`
- `设备绑定`
- `入口命令`
- `附加环境变量`
- `启动方式`
- `端口映射`
- `服务模型名`
- `卷挂载`
- `健康检查`

请根据实际术语和现有 i18n 风格微调，但保持短、清晰、稳定。

## D. 公共组件改造要求

1. 若已有 `ConfigEditView` / `ConfigField` / `RuntimeParameterEditor`，优先复用并增强公共 normalizer。
2. 若多个组件各自处理 schema，抽出单一 `normalizeConfigSchema()` 或等价公共函数。
3. 所有调用方传入 schema 后先走 normalizer，再交给字段组件渲染。
4. 字段组件只消费标准化后的 `FieldSpec`，包含：
   - `key`
   - `section`
   - `type`
   - `label`
   - `tooltip/help`
   - `required`
   - `enabled`
   - `value`
   - `default`
   - `options`
   - `readonly/disabled`
5. enabled checkbox 与 input 显示分离：未启用字段仍展示输入框，仅值是否参与最终 RunPlan 由 enabled 控制。
6. 数字、字符串、布尔、枚举、数组、KV、mounts、ports、health 等复杂字段都经公共字段组件渲染。
7. 保持 backend runtime、node backend runtime、deployment override 的 copy-on-create / override 语义，修复 UI 显示时不得破坏 RunPlan 解析。

## E. 向导按钮与布局改造要求

请把长表单向导操作区提炼为公共组件或公共布局约定，例如 `WizardActionBar` / `WizardShell` / `StickyWizardActions`，实际命名按项目风格确定。

统一行为：

1. 在 Dialog/Drawer/Page 向导中，主要操作按钮滚动时持续可见。
2. 优先放在向导标题/步骤条附近的 sticky 顶部操作区；如项目已有 sticky header 规范，沿用该规范。
3. `取消 / 上一步 / 下一步 / 保存 / 保存并检测` 放在同一操作区。
4. 主按钮靠右，危险或退出类操作靠左或次要位置。
5. 第一步隐藏或禁用“上一步”。
6. 最后一步主按钮文案切换为保存/保存并检测。
7. disabled 状态附带明确校验提示，避免用户只看到灰色按钮。
8. 小屏适配：按钮换行后仍可操作，不遮挡输入内容。
9. 键盘 tab 顺序合理，按钮有可访问名称。
10. 保留必要的 Dialog 关闭按钮，但主流程动作集中到公共操作区。

需要检查并统一的页面至少包括：

- 节点运行配置向导。
- 模型部署向导。
- 运行模板/Backend Runtime 编辑页中长参数编辑区域。
- 其他使用步骤条或长配置表单的页面。

## F. 测试要求

请补齐或更新测试，至少覆盖：

1. 公共 schema normalizer：
   - i18n key 命中。
   - label/title fallback。
   - 已知 internal key fallback。
   - unknown key 诊断 fallback。
   - tooltip/help fallback。
   - enabled=false 时 input 仍展示。
2. ConfigField/ConfigEditView 组件测试：
   - 渲染 vLLM 样例 schema，不出现大量“配置项”。
   - 渲染 `GPU 显存利用率`、`最大上下文长度`、`数据类型`、`张量并行数` 等关键 label。
   - tooltip icon 与 help 文案存在。
   - checkbox 与输入框同时存在。
3. RunnerConfigsPage 集成测试：
   - 打开新增/编辑节点运行配置向导，进入“镜像与参数”。
   - 验证结构化参数字段 label 正确。
   - 验证 `上一步 / 下一步 / 取消` 在统一 sticky action bar 中。
4. ModelDeploymentsPage 或同类部署向导测试：
   - 验证复用同一 action bar。
   - 验证同一字段 label 解析行为。
5. i18n/audit：
   - 中文 locale 下关键字段无 raw English label 泄露，技术缩写如 GPU、Token、Docker、KV Cache 可保留。
   - 无 `chatgpt` 类无关检查；只检查 LightAI Go 项目自身 i18n。
   - `配置项` 只允许作为极少量 unknown fallback，不允许出现在已知字段快照中。

建议执行命令，以实际 package scripts 为准：

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go

git status --short

# 后端 schema/seed 如有改动
go test ./...

# 前端
cd web
npm test
npm run test:unit
npm run build

# 重点测试可按项目实际命令追加
npm test -- ConfigEditView
npm test -- RunnerConfigsPage
npm test -- ModelDeploymentsPage
```

## G. 验收标准

1. `/runner-configs` 的“镜像与参数”步骤中，已知字段不再显示“配置项”。
2. vLLM/SGLang/llama.cpp 的关键运行参数显示短中文 label 和 tooltip。
3. 公共配置编辑器被证实为统一入口；若发现分叉渲染，完成合并或给出已修复证据。
4. 模型页、运行模板页、节点运行配置页、部署页的参数渲染使用统一 normalizer。
5. 长表单向导滚动时，主要操作按钮持续可见。
6. `取消 / 上一步 / 下一步 / 保存 / 保存并检测` 操作区一致。
7. 单测、集成测试、build 全部通过。
8. 生成 closeout 文档，包含根因、修复文件、测试命令、测试结果、截图或 DOM evidence、commit id、push 结果、git status。
9. `git status --short` 最终只允许为空，或仅包含用户已明确说明的既有未跟踪/未处理项，并在 closeout 中说明。
10. 提交并推送到当前分支。

## H. 输出要求

请最终输出：

1. 根因结论。
2. 公共渲染链路说明。
3. 修改文件清单。
4. 新增/修改测试清单。
5. 验证命令和结果。
6. closeout 文档路径。
7. commit id、push 结果、git status。
8. 若仍有无法在当前环境验证的事项，说明原因、影响面、已经留下的自动化防线。
