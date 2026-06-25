# ConfigSet 重构实施计划

## 总体策略

这是一次断裂式重构，不兼容旧结构。

允许：

- 重建 DB；
- 删除旧 API 字段；
- 删除旧 migration/repair/fallback；
- 归档过时文档；
- 更新 UI；
- 更新 E2E；
- 重写 OpenAPI。

禁止：

- 保留旧 DB 兼容迁移；
- 保留 `db.go` backend catalog mirror；
- 保留 `capabilities_json` / `parameter_schema_json` 等旧字段为权威；
- 用 drift test 替代去硬编码；
- 用 manual DB update 作为修复；
- 用 preflight PASS / task claimed / image present 冒充 runtime smoke PASS。

## Phase 0：冻结当前 final closeout 口径

目标：

- 暂停 `FINAL_CLOSEOUT_ACCEPTED`。
- 记录当前代码状态、commit、git status、open tasks。
- 明确这是 ConfigSet 重构，不是 SGLang 单点修复。

输出：

```text
docs/reports/configset-refactor-20260625/00-baseline.md
```

## Phase 1：写入设计文档与归档计划

动作：

1. 写入 `docs/design/catalog-configset-and-runtime-snapshot.md`。
2. 搜索所有旧字段和旧口径：
   - `capabilities_json`
   - `parameter_schema_json`
   - `parameter_values_json`
   - `parameters_json`
   - `env_json`
   - `ports_json`
   - `volumes_json`
   - `devices_json`
   - `health_check_json`
   - `db.go seed`
   - `/check`
   - `image/model present smoke`
3. 生成归档清单。
4. 归档过时文档到 `docs/archive/<date>-pre-configset-catalog-model/`。

验收：

- 设计文档存在；
- 过时文档清单存在；
- 归档文档顶部有 superseded 说明。

## Phase 2：新增 Config Registry / Catalog Loader

新增目录：

```text
configs/config-registry/
configs/backend-catalog/
internal/server/catalog/
```

实现职责：

```text
LoadRegistry()
LoadBackendCatalog()
ValidateRegistry()
ValidateCatalog()
MaterializeBackend()
MaterializeBackendVersion()
MaterializeBackendRuntime()
SeedCatalog()
```

要求：

- registry 定义 ConfigItem；
- backend catalog 引用和覆盖 ConfigItem；
- Go 代码只做通用加载、校验、materialize、seed；
- 不写 vLLM / SGLang / llama.cpp 业务 literal。

验收：

- loader 单元测试；
- registry lint；
- catalog lint；
- no array-format capabilities；
- no unknown config item code；
- no duplicate ConfigItem code。

## Phase 3：删除 db.go Backend Catalog 硬编码

动作：

1. 删除 `seedBuiltInBackends()` 中 vLLM/SGLang/llama.cpp catalog literal。
2. 删除或替换 `seedTargetBackendCatalog()`。
3. 删除 `repairBackendCapabilitiesV27()` 和类似 repair。
4. `db.go` 只调用 `catalog.SeedCatalog()`。
5. 删除 seed-only deprecated versions，或迁入 YAML 并标记 deprecated/disabled。

验收：

- `rg` 证明 `db.go` 不含 vLLM/SGLang/llama.cpp catalog literal；
- fresh DB 从 YAML + registry 生成；
- 不存在 seed-only deprecated versions。

## Phase 4：DB Schema 重建

不兼容旧 DB。

目标字段：

```text
backends.config_set_json
backend_versions.config_set_json
backend_runtimes.config_set_json
node_backend_runtimes.config_set_json
model_deployments.config_set_json
source_metadata_json
```

删除旧权威字段：

```text
capabilities_json
parameter_schema_json
parameter_values_json
env_json
ports_json
volumes_json
devices_json
health_check_json
resource_controls_json
parameters_json
default_args_json
parameter_defs_json
default_backend_params_json
default_images_json
image_candidates_json
docker_options_json
model_mount_json
```

不允许 additive compatibility migration，不允许通过 ADD COLUMN 给旧 schema 增加 config_set_json 后保留旧列过渡，不允许旧字段暂留以维持旧 API 读路径。

如果删除旧字段导致 API、UI、测试或运行链路断裂，必须同步修改 API、UI、测试和运行链路，使当前提交点直接使用 clean schema。不得把“后续阶段再清理旧字段/旧 API/fallback”作为已提交状态。

每个 checkpoint 的 commit/push 都必须保持方向干净：不得新增或保留 legacy compatibility path。若某个 checkpoint 无法单独形成 clean state，应扩大当前 checkpoint 范围，连续完成后续必要改造，再提交。

验收：

- fresh DB schema clean；
- 所有 catalog/runtime/deployment config 只以 ConfigSet 为权威；
- no legacy JSON authority fields。

## Phase 5：Copy-on-create 链路改造

实现：

```text
BackendVersion = copy(Backend ConfigSet) + version overrides
BackendRuntime = copy(BackendVersion ConfigSet) + runtime overrides
NBR = copy(BackendRuntime ConfigSet) + node overrides
Deployment = copy(NBR ConfigSet) + deployment overrides
RunPlan = render(Deployment ConfigSet)
```

测试：

1. Backend → BackendVersion copy；
2. BackendVersion → BackendRuntime copy；
3. BackendRuntime → NBR copy；
4. NBR → Deployment copy；
5. 修改 parent 后 child 不变；
6. RunPlan 不回读 parent live data。

## Phase 6：Renderer 改造

实现：

- CLI renderer；
- Docker launcher renderer；
- env renderer；
- port renderer；
- volume renderer；
- device renderer；
- health renderer；
- extra_args parser；
- extra_env parser；
- duplicate structured flag/env/option exclusion。

禁止 backend-name hardcode：

```text
if backend == "vllm" append "--gpu-memory-utilization"
if backend == "sglang" append "--mem-fraction-static"
if backend == "llamacpp" append "--ctx-size"
```

参数差异必须来自 ConfigItem.render。

## Phase 7：API 改造

删除旧字段 API：

```text
capabilities_json
parameter_schema_json
parameter_values_json
parameters_json
env_json
ports_json
volumes_json
devices_json
health_check_json
resource_controls_json
```

新接口统一：

```json
{
  "config_set": {},
  "config_overrides": {},
  "source_metadata": {}
}
```

涉及：

- Backend list/get；
- BackendVersion list/get；
- BackendRuntime CRUD/clone；
- NBR enable/check-request；
- Deployment create/edit/preflight/dry-run/start；
- RunPlan preview。

## Phase 8：UI 改造

RuntimeParameterEditor 改成 ConfigSet editor：

- 按 category 分组：
  - launcher
  - runtime_env
  - model_runtime
- 显示 current/default/source/last_modified/support_level/warnings；
- extra_args textarea，每行一个；
- extra_env textarea，每行一个；
- duplicate structured flag/env 报错；
- RunPlan preview 从 ConfigSet renderer 结果展示。

## Phase 9：E2E / OpenAPI / 文档收敛

动作：

1. 更新 OpenAPI。
2. 更新 active E2E scripts。
3. 归档旧 payload scripts。
4. active stale gate 禁止：
   - `parameters_json`
   - `capabilities_json` authority
   - `parameter_schema_json` authority
   - `/check`
   - image/model presence only smoke。

## Phase 10：Fresh DB + 三 Runtime Smoke

Fresh DB：

```bash
export LIGHTAI_HOME=/tmp/lightai-configset-fresh
rm -rf "$LIGHTAI_HOME"
```

三 runtime 必须走平台链路：

```text
YAML catalog
→ ConfigSet materialization
→ BackendRuntime
→ NBR enable/check-request
→ Deployment preflight/dry-run/start
→ ResolvedRunPlan
→ AgentRunSpec
→ Docker container
→ health/models
→ inference
→ logs
→ stop/cleanup
```

三条：

```text
vLLM PASS
SGLang PASS
llama.cpp PASS
```

不能用：

```text
preflight PASS only
previously demonstrated
task claimed only
image/model present only
direct docker run bypass
```

## Phase 11：Final closeout

输出：

```text
docs/reports/configset-refactor-20260625/final-closeout.md
```

必须包含：

- commit list；
- old fields removed；
- db.go hardcode removal evidence；
- catalog authority evidence；
- copy-on-create tests；
- renderer tests；
- fresh DB verification；
- runtime smoke evidence；
- OpenAPI/E2E stale gate；
- final git status。
