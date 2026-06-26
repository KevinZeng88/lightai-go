# 04 - API 与前端组件设计

## 1. API 设计

新增：

```http
POST /api/v1/config-edit/view
POST /api/v1/config-edit/apply
```

也可以在现有对象详情中追加 `editable_config_view`，但建议先使用独立接口，降低对现有 API 的侵入。

### view request

```json
{
  "object_kind": "backend_runtime",
  "object_id": "runtime.vllm.nvidia-docker",
  "layer": "backend_runtime",
  "mode": "edit"
}
```

支持 object_kind：

```text
backend_version
backend_runtime
node_backend_runtime
deployment
```

### apply request

```json
{
  "object_kind": "backend_runtime",
  "object_id": "xxx",
  "layer": "backend_runtime",
  "patch": {
    "fields": [
      {
        "key": "launcher.docker_options.shm_size",
        "internal_key": "launcher.docker_options",
        "path": ["shm_size"],
        "value": "100gb",
        "enabled": true
      }
    ]
  }
}
```

对于 create/clone/enable/deploy，也可以接受：

```json
{
  "editable_config_patch": {...}
}
```

后端统一 apply 到 ConfigSet 后再保存。

## 2. 页面替换路径

### BackendVersion

详情加载 ConfigEditView；新增参数仍可保留，但新增时必须生成完整 ConfigItem metadata；保存走 ConfigEditPatch/apply；系统版本 readonly。

### BackendRuntime

用户配置编辑区使用 ConfigEditView；系统模板只读，只能 clone；高级诊断只读展示 raw ConfigSet；clone 必须弹窗输入 display_name/name。

### NodeRuntimeConfigWizard

选择器使用产品化 RuntimeTemplateDisplay；Step 2 从 selected runtime project 出 ConfigEditView；用户修改生成 ConfigEditPatch；enable API 接收 editable_config_patch。

### DeploymentOverrideEditor

用 ConfigEditView mode=`deployment_override`；默认只显示允许部署层覆盖的字段；host/port/protected flags readonly 或 hidden。

## 3. 前端组件

新增：

```text
web/src/components/config/ConfigEditView.vue
web/src/components/config/ConfigSection.vue
web/src/components/config/ConfigField.vue
web/src/components/config/fields/StringField.vue
web/src/components/config/fields/NumberField.vue
web/src/components/config/fields/BooleanField.vue
web/src/components/config/fields/SelectField.vue
web/src/components/config/fields/KeyValueListField.vue
web/src/components/config/fields/DeviceListField.vue
web/src/components/config/fields/StringListField.vue
web/src/components/config/fields/RawJsonField.vue
web/src/utils/configEditView.ts
```

## 4. ConfigEditView.vue 行为

- 渲染 `sections`。
- section 按 order 排序。
- advanced/raw 默认折叠。
- field 根据 widget 渲染。
- 大多数 field 左侧有 enabled checkbox。
- required field checkbox 隐藏或 disabled。
- disabled field 仍显示值。
- readonly field 不可修改。
- 修改后 emit `update:patch`。
- 不直接 emit config_set。

## 5. RuntimeParameterEditor 的定位

短期保留 RuntimeParameterEditor 作为兼容 fallback 和 raw diagnostic。普通编辑入口替换为 ConfigEditView。长期可降级为 RawConfigSetEditor 或删除。

## 6. 复制运行模板

BackendRuntime clone API 增强：

```json
{
  "display_name": "vLLM / MetaX - 招行配置",
  "name": "vllm-metax-cmb"
}
```

后端规则：

- display_name 必填或有合理默认。
- name 可选。
- name 冲突时自动唯一化或返回 409；建议自动唯一化并返回 final name。
- `managed_by=user`
- `is_editable=true`
- `source_metadata.source_backend_runtime_id` 保留来源。

前端：clone 弹窗，用户输入 display_name，raw technical name 放高级选项，列表主标题显示 display_name。
