# 当前代码关键发现

> 依据 GitHub 仓库 `KevinZeng88/lightai-go` 当前 main 分支及近期 commits 抽样核查。
> 本文用于说明为什么必须做 ConfigSet / ConfigItem 重构，而不是继续补旧 seed 或 drift test。

## 1. db.go 仍是 backend catalog 的硬编码来源

`internal/server/db/db.go` 中 `Migrate()` 在迁移过程中直接调用 `seedTargetBackendCatalog()`，随后还调用 `repairBackendCapabilitiesV27()`。这说明 catalog seed 与 capability repair 已经进入 DB migration 主流程。

关键风险：

- catalog 数据和 schema migration 混在一起；
- 每次迁移/启动都可能重写或修复 catalog；
- 后续新 backend/version/runtime 很容易继续写进 Go 代码；
- V27 repair 这类“修补式逻辑”会长期存在，掩盖数据模型问题。

## 2. seedBuiltInBackends() 保留旧一代手写 Backend / BackendVersion

`seedBuiltInBackends()` 中仍然手写 vLLM / SGLang / llama.cpp 的：

- backend id/name/display；
- protocol；
- common parameters；
- default env；
- backend_versions；
- default_entrypoint_json；
- default_args_json；
- parameter_defs_json；
- health_check_json；
- default_container_port；
- default_images_json。

其中还存在 `bver-*` 旧版本，例如：

- `bver-vllm-0.8.5`
- `bver-vllm-0.10.0`
- `bver-sglang-0.4.6`
- `bver-sglang-0.5.0`
- `bver-llamacpp-b4817`

这些不应继续作为 fresh DB seed 来源。项目不需要历史兼容，旧版本应删除，或迁入 YAML catalog 并标记 deprecated/disabled。

## 3. seedTargetBackendCatalog() 是另一套手写 catalog

`seedTargetBackendCatalog()` 中又维护了一套新的 backend/version catalog，包括：

- `backend.llamacpp`
- `backend.vllm`
- `backend.sglang`
- `backend.ollama`
- 版本、entrypoint、args、params、paramDefs、health check、images、capabilities、mount、references 等。

这形成了至少三套口径：

1. 旧 `seedBuiltInBackends()`；
2. 新 `seedTargetBackendCatalog()`；
3. `configs/backend-catalog/` YAML。

这正是 SGLang capability 缺失、array-format vs structured-format、manual DB update 等问题的根源。

## 4. YAML catalog 当前反而声明 DB 是 canonical

例如 `configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml` 中有注释：

```yaml
# Canonical capabilities are in DB backend_versions.capabilities_json (V27 repair).
# This YAML is a human-readable mirror; must match DB seed.
```

这与目标设计相反。目标应为：

```text
YAML / registry 是唯一权威来源；
DB 是 materialized snapshot；
Go migration 不维护业务 catalog literal。
```

因此当前 YAML 需要改写，不再把 DB / V27 repair 当 canonical。

## 5. YAML 版本文件仍混杂旧字段

以 vLLM version YAML 为例，文件中同时存在：

```yaml
capabilities:
  - models
  - chat_completions
  - completions
  - embeddings
  - openai_compatible

capabilities_json:
  supported_formats: [huggingface, safetensors]
  ...
default_args_schema:
  - name: --model
  ...
```

这说明 YAML 本身也需要规范化到 ConfigSet / ConfigItem，不应保留多套字段（`capabilities` + `capabilities_json` + `default_args_schema`）作为并行权威。

## 6. API / Runtime handler 仍以旧字段为主

`internal/server/api/runtime_handlers.go` 中仍然查询和更新大量旧字段，例如：

- `entrypoint_override_json`
- `args_override_json`
- `default_env_json`
- `docker_json`
- `model_mount_json`
- `health_check_override_json`
- `parameter_schema_json`
- `parameter_values_json`
- `version_snapshot_json`
- `config_snapshot_json`

这些字段分散表达 launcher/runtime/model 参数，无法统一记录：

- 参数分类；
- 渲染方式；
- 渲染顺序；
- 默认值；
- 当前值；
- 来源层级；
- 最后修改层级；
- extra 参数重复排除。

目标设计应统一为：

```text
config_set_json
config_overrides_json
source_metadata_json
```

## 7. 近期 drift test 证明问题仍存在

近期 commit `8c0d31a` 添加了 `TestCatalogSeedDrift`，其 commit message 明确写道：

```text
Catalog authority: YAML files are authoritative; seed is bootstrap mirror
```

该 test 还把 deprecated versions 设计为 “seed-only”。这不是去硬编码，而是接受双源和 seed-only legacy。目标重构不应保留这种模式。

## 8. 结论

当前问题不是 SGLang 一条 capability 配错，而是 catalog/config/snapshot 模型没有统一：

- Go 代码里有硬编码 catalog；
- YAML 不是唯一权威；
- DB migration 修补 catalog；
- API/DB 以旧散字段为主；
- runtime smoke / preflight / RunPlan 可能仍由旧字段支撑；
- 文档可能继续描述旧口径。

因此必须做 ConfigSet / ConfigItem 重构，并且不兼容旧结构。
