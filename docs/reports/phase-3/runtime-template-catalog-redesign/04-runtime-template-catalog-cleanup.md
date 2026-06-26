# 04 - Runtime Template Catalog 清理设计

## 1. 目标

解决当前 UI 中出现大量重复、占位和 `runtime.xxx` 配置的问题。

最终普通用户在“节点运行配置 / 选择运行模板”里只应该看到少量可用模板，例如：

```text
NVIDIA / vLLM / compatible
NVIDIA / SGLang / compatible
NVIDIA / llama.cpp / compatible
CPU / llama.cpp / compatible
MetaX / vLLM / compatible
Huawei / vLLM / compatible
```

隐藏参考模板可以保留在 catalog，但不能进入普通选择器。

---

## 2. Catalog 层级

建议继续使用：

```text
configs/backend-catalog/backends
configs/backend-catalog/versions
configs/backend-catalog/runtimes
```

但将 Runtime 的语义明确为：

```text
BackendRuntimeTemplate / BackendRuntime Catalog Projection
```

即：数据库 `backend_runtimes` 中的 system rows 是 runtime template catalog projection。

---

## 3. BackendVersion 策略

BackendVersion 不再为每个小版本铺很多模板。保留：

```text
backend-version.vllm.compat
backend-version.sglang.compat
backend-version.llamacpp.compat
backend-version.ollama.compat
```

必要时可增加精确版本：

```text
backend-version.vllm.v0.23
backend-version.sglang.v0.5
backend-version.llamacpp.b9700
```

BackendVersion 的重点是：

- 参数 schema
- protocol
- capabilities
- default host/port
- health check
- backend common defaults

---

## 4. RuntimeTemplate 策略

RuntimeTemplate 以：

```text
vendor + backend + compatible
```

组织。

建议 visible 模板：

```text
nvidia.vllm.compat
nvidia.sglang.compat
nvidia.llamacpp.compat
cpu.llamacpp.compat
metax.vllm.compat
huawei.vllm.compat
```

可作为 hidden reference 的国产加速器模板：

```text
metax.sglang.compat
huawei.sglang.compat
huawei.llamacpp.compat
hygon.vllm.compat
enflame.vllm.compat
cambricon.vllm.compat
mthreads.vllm.compat
iluvatar.vllm.compat
biren.vllm.compat
thead.vllm.compat
```

这些 hidden 模板可以进入 catalog 文件和系统审查文档，但普通选择器不展示。

---

## 5. visibility/status 设计

建议在 `backend_runtimes` 增加字段：

```sql
visibility TEXT NOT NULL DEFAULT 'visible'
support_level TEXT NOT NULL DEFAULT 'documented'
```

或至少在 `source_metadata_json` 中保存：

```json
{
  "visibility": "hidden",
  "support_level": "experimental",
  "reference_only": true
}
```

推荐用字段，不建议仅靠 JSON。

### status

```text
active        可用模板
experimental 可见但需 check 后可部署
disabled     不可用
deprecated   旧模板，不展示
```

### visibility

```text
visible       普通 UI 可见
hidden        普通 UI 不展示，高级/诊断可见
internal      不展示，仅系统使用
```

普通“选择运行模板”过滤：

```text
visibility = visible
status in active / experimental
```

---

## 6. 禁止出现在普通 UI 的内容

普通选择器中不得出现：

```text
runtime.xxx
template-only
<from Metax release package>
0d307f1665d3
重复 nvidia.vllm
重复 nvidia.sglang
重复 nvidia.llamacpp
CPU 模板使用 CUDA-only 镜像
hidden reference 模板
disabled 模板
```

---

## 7. Seed 幂等与唯一性

当前 seed 使用 `ON CONFLICT(id)`，无法阻止逻辑重复。

建议增加 catalog validation：

### visible runtime 逻辑唯一

```text
visibility=visible 且 status in active/experimental 时：
unique(vendor, backend_id, backend_version_id)
```

如果同一个 vendor/backend/version 需要多个模板，则必须：

```text
visibility=hidden
```

或者明确加 role：

```text
profile: default / debug / high-throughput
```

并且普通 UI 只展示 `profile=default`。

### 数据库约束

SQLite partial unique 可以考虑：

```sql
CREATE UNIQUE INDEX idx_backend_runtimes_visible_default
ON backend_runtimes(vendor, backend_id, backend_version_id)
WHERE tenant_id = '' AND visibility = 'visible' AND status IN ('active','experimental');
```

如果 SQLite 版本或字段调整不方便，至少在 `ValidateCatalog()` 里强校验。

---

## 8. UI 显示规则

运行模板显示名应来自字段：

```text
display_name || name
```

不得使用：

```ts
$t(`runtime.${name}`)
```

可以对 vendor/backend 枚举做 i18n，但不要对业务对象 ID 做动态 i18n。

推荐显示：

```text
NVIDIA vLLM compatible
MetaX vLLM compatible
Huawei vLLM compatible
CPU llama.cpp compatible
```

表格列：

| 列 | 来源 |
|---|---|
| 名称 | display_name |
| 厂商 | vendor |
| 后端 | backend.display_name |
| 版本策略 | backend_version.display_name / version |
| 镜像 | launcher.image |
| 状态 | status |
| 可见性 | visibility |
| 支持级别 | support_level |
| 来源 | managed_by |

---

## 9. 国内 GPU 模板原则

### 9.1 MetaX / 沐曦

沐曦 vLLM 可作为 visible experimental：

```text
metax.vllm.compat
visibility: visible
status: experimental
support_level: experimental
```

必须通过 NBR check-request 验证本节点：

- 镜像是否存在
- Docker 是否可用
- 设备文件是否存在
- 环境变量是否正确
- 版本 probe 是否有警告

不允许再出现 `<from Metax release package>` 作为 image。

### 9.2 Huawei

Huawei vLLM 可作为 visible experimental，但需明确依赖 Ascend 节点检查。

SGLang / llama.cpp CANN 可先 hidden experimental，等具体镜像和参数稳定后再 visible。

### 9.3 其他国产加速器

海光、燧原、寒武纪、摩尔线程、天数智芯、壁仞、阿里平头哥等可以先放 hidden reference：

```text
visibility: hidden
status: experimental 或 disabled
support_level: reference
```

用途：

- 保留设计位置
- 给后续适配参考
- 不污染普通 UI
- 不影响用户选择

---

## 10. 验收

1. `/backend-runtimes` 返回普通可见模板无重复。
2. NodeRuntimeConfigWizard 选择器不显示 hidden/disabled/reference。
3. BackendRuntimesPage 可切换“显示隐藏/参考模板”用于管理员审查。
4. `runtime.xxx` 不再出现在页面。
5. 所有模板都有明确 image 或隐藏/禁用。
6. 运行模板 seed 多次执行后数量不重复增长。
7. 删除脏数据：`template-only`、`<from Metax release package>`、本地 image id。
8. API-first 测试验证 visible filter。
