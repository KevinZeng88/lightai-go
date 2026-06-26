# 02 - ConfigEditView 通用架构设计

## 1. 设计目标

建立一套跨 BackendVersion、BackendRuntime、NodeBackendRuntime、Deployment 复用的配置编辑抽象：

```text
ConfigSet JSON
  -> ProjectConfigSetToEditView()
  -> ConfigEditView
  -> 用户编辑
  -> ConfigEditPatch
  -> ApplyEditPatchToConfigSet()
  -> ConfigSet JSON
```

## 2. 核心原则

1. `config_set_json` 是内部 canonical storage。
2. UI 不直接理解 `launcher.xxx` / `runtime.xxx`。
3. UI 渲染 `ConfigEditView`。
4. UI 保存 `ConfigEditPatch`。
5. 后端负责 project/apply/validate/normalize。
6. 新增配置项只要具备 metadata，即可自动进入对应 section。
7. 缺 metadata 时也有稳定 fallback 规则。
8. 大多数参数默认带 enabled checkbox。
9. 必填参数 `required=true` 时默认启用，不允许取消勾选。
10. object/json 参数普通区默认拆分子字段或进入高级原始配置。
11. RunPlan 仍只读最终 NBR / Deployment snapshot，不读上游 live config。

## 3. 后端模块建议

新增：

```text
internal/server/configedit/
```

建议文件：

```text
types.go
project.go
apply.go
validate.go
taxonomy.go
docker_options.go
configset_adapter.go
```

## 4. 类型定义

### ConfigEditView

```go
type ConfigEditView struct {
    Layer       string                `json:"layer"`
    ObjectID    string                `json:"object_id"`
    ObjectKind  string                `json:"object_kind"`
    Readonly    bool                  `json:"readonly"`
    Sections    []EditSection         `json:"sections"`
    Diagnostics ConfigEditDiagnostics `json:"diagnostics,omitempty"`
    Metadata    map[string]any        `json:"metadata,omitempty"`
}
```

### EditSection

```go
type EditSection struct {
    Key         string      `json:"key"`
    Label       string      `json:"label"`
    Description string      `json:"description,omitempty"`
    Order       int         `json:"order"`
    Advanced    bool        `json:"advanced,omitempty"`
    Collapsed   bool        `json:"collapsed,omitempty"`
    Fields      []EditField `json:"fields"`
}
```

### EditField

```go
type EditField struct {
    Key          string         `json:"key"`
    InternalKey  string         `json:"internal_key"`
    ParentKey    string         `json:"parent_key,omitempty"`
    Path         []string       `json:"path,omitempty"`
    Label        string         `json:"label"`
    Help         string         `json:"help,omitempty"`
    Section      string         `json:"section"`
    Group        string         `json:"group,omitempty"`
    Order        int            `json:"order"`
    Type         string         `json:"type"`
    Widget       string         `json:"widget"`
    Value        any            `json:"value"`
    DefaultValue any            `json:"default_value,omitempty"`
    Enabled      bool           `json:"enabled"`
    HasEnable    bool           `json:"has_enable"`
    Required     bool           `json:"required"`
    Readonly     bool           `json:"readonly"`
    Advanced     bool           `json:"advanced"`
    Visibility   string         `json:"visibility"`
    Options      []EditOption   `json:"options,omitempty"`
    Constraints  map[string]any `json:"constraints,omitempty"`
    Source       map[string]any `json:"source,omitempty"`
}
```

### ConfigEditPatch

```go
type ConfigEditPatch struct {
    Layer    string           `json:"layer"`
    ObjectID string           `json:"object_id"`
    Fields   []EditFieldPatch `json:"fields"`
}
```

```go
type EditFieldPatch struct {
    Key         string   `json:"key"`
    InternalKey string   `json:"internal_key"`
    Path        []string `json:"path,omitempty"`
    Value       any      `json:"value"`
    Enabled     *bool    `json:"enabled,omitempty"`
}
```

## 5. Project 函数

```go
func ProjectConfigSetToEditView(input ProjectInput) (ConfigEditView, error)
```

Project 负责：

1. 读取 ConfigSet items。
2. 根据 `render.section` / taxonomy 归组。
3. 展开复合 object，如 `launcher.docker_options`。
4. 生成 user-facing label/help/widget。
5. 决定 `has_enable`。
6. 决定 required/readonly/advanced。
7. 排序 section 和 field。
8. 把不能普通编辑的 raw object 放入 `advanced_raw`。

## 6. Apply 函数

```go
func ApplyEditPatchToConfigSet(set map[string]any, patch ConfigEditPatch, layer, ref string) (map[string]any, error)
```

Apply 负责：

1. 根据 `internal_key` 定位 ConfigSet item。
2. 根据 `path` 回写子字段。
3. 更新 `value`。
4. 更新 `enabled`。
5. required 字段强制 enabled=true。
6. 记录 source metadata。
7. 对 object 子字段重新合并为内部 object。
8. 不允许普通 patch 修改 readonly/internal hidden 字段。
9. 不允许删除 unknown required 字段。

## 7. 为什么不直接用 JSON Schema Form

JSON Schema 很适合表达数据结构、类型、约束和 required 字段，但它不能直接表达 LightAI Go 的领域语义：ConfigSet copy-on-create 来源、enabled checkbox 与 value 的双状态、RunPlan target、Docker options 子字段合并、必填参数不可关闭、hidden/internal/advanced/raw 分层、Deployment 受保护字段、source metadata 和 audit。

所以 JSON Schema 可以作为 `EditField.constraints` 的参考或未来校验格式，但不能替代 ConfigEditView。
