# LightAI Go：节点运行配置渲染与向导操作审查目标

## 1. 用户反馈与现场证据

用户在“节点运行配置”页面配置运行配置，进入“镜像与参数”步骤后发现：

1. “结构化参数”区域大量字段只显示“配置项”，无法看出每个字段的真实含义。
2. “上一步 / 下一步 / 取消”等操作按钮位于长内容区域底部或 Dialog footer，滚动长表单时不方便操作。
3. 用户希望确认配置渲染是否已经是通用公共程序。如果是公共程序，应从公共契约和公共组件层修复，避免每个页面逐一补丁。
4. 用户希望向导按钮也从公共易用性角度统一处理。

附件 `LightAI Go.mhtml` 现场观察：

- Snapshot URL：`/runner-configs`。
- 页面处于“节点运行配置”向导第 3 步“镜像与参数”。
- 页面加载了 `RunnerConfigsPage-*.css` 与 `ConfigEditView-*.css`。
- DOM 中出现公共配置编辑器样式：`.config-edit-view`、`.config-field`、`.field-label`。
- 字段 DOM 中存在稳定内部 key，例如：
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
- 这些字段的可见 label 全部退化为“配置项”。
- 当前 CSS 显示 `.wizard-footer` 只是普通 flex 容器；`上一步 / 下一步` 在长内容后方，`取消` 位于 `el-dialog__footer`，操作区分散。

## 2. 初步判断

该问题高度可能出在公共配置渲染链路的 label/tooltip 元数据契约，而非单个页面的纯样式问题。

已看到字段有稳定 key，说明前端已经能识别字段身份；可见 label 退化为“配置项”，说明以下链路至少一处存在缺口：

1. 后端/seed 内置 schema 未给字段提供稳定 `label_i18n_key` / `tooltip_i18n_key` / `label` / `title`。
2. 前端 schema normalizer 没有把 `key`、`i18n_key`、`title`、`description` 等字段统一映射成公共 `FieldSpec`。
3. 公共字段组件的 fallback 过于宽泛，导致已知字段也显示“配置项”。
4. i18n 字典存在缺口，`t()` lookup 失败后进入“配置项”兜底。
5. RunnerConfigsPage 在传 schema 时丢失了字段 metadata，仅保留了 `data-internal-key`。

修复方向：在公共 schema 契约、schema normalizer、公共 ConfigEditView/ConfigField、i18n 字典、调用方测试上形成闭环。

## 3. 修复目标

### 3.1 结构化参数渲染

统一实现以下行为：

1. 任意使用公共配置编辑器的页面，都通过同一个 schema normalizer 和字段组件渲染参数。
2. 已知 key 必须显示业务化中文/英文 label，禁止退化为“配置项”。
3. 字段 tooltip/help 通过同一套 i18n/metadata 机制显示。
4. 未启用字段仍显示输入框，checkbox 只控制 enabled 状态。
5. API schema、后端 seed、前端 i18n、公共组件的字段命名保持一致。
6. 模型页、运行模板页、节点运行配置页、部署页使用同一套字段解析逻辑。
7. 新增字段时只需补 schema metadata/i18n 字典，公共组件自动正确渲染。

### 3.2 向导操作按钮

统一实现以下行为：

1. 长表单滚动时，主要操作按钮持续可见。
2. `取消 / 上一步 / 下一步 / 保存 / 保存并检测` 位于统一操作区。
3. 操作区在 Dialog/Drawer/Page 向导中复用，不在每个页面重复写布局。
4. `下一步` disabled 时给出清晰原因或校验提示。
5. 第一步隐藏或禁用“上一步”；最后一步主按钮文案切换为保存/保存并检测。
6. 支持键盘可达和基础 accessibility。
7. 小屏下操作区不遮挡内容，按钮换行或压缩合理。

## 4. 禁止的修复方式

1. 禁止只在 `RunnerConfigsPage` 写死中文 label。
2. 禁止用字段名直接替代 label，例如直接显示 `model_runtime.gpu_memory_utilization`。
3. 禁止保留“配置项”作为已知字段的正常显示结果。
4. 禁止每个页面各自维护一套参数 label 表。
5. 禁止只移动当前页面按钮，遗漏部署向导、运行模板编辑等同类长表单。
6. 禁止把问题记录为 future/follow-up 后结束；可定位、可修复、可验证的问题必须完成修复和验证。

## 5. 建议重点检查文件

以实际仓库为准，优先检查以下方向：

- `web/src/components/**/ConfigEditView.vue`
- `web/src/components/**/ConfigField.vue`
- `web/src/components/**/RuntimeParameterEditor.vue`
- `web/src/components/**/JsonViewer.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/pages/ModelArtifactsPage.vue`
- `web/src/i18n/**`
- `internal/**/backend_runtime*`
- `internal/**/parameter_schema*`
- `internal/**/seed*`
- `docs/**runtime*parameter*`

如果实际文件名不同，按功能查找：配置编辑器、字段组件、schema normalizer、NBR/BackendRuntime seed、运行配置向导、部署向导。
