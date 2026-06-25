# ConfigSet 重构验证矩阵与验收标准

## 1. 静态硬编码检查

运行：

```bash
rg -n "vllm|sglang|llama|llama.cpp|llamacpp|vllm-openai|lmsysorg/sglang|ggml-org|supported_formats|capabilities_json|parameter_schema_json|parameter_values_json|default_args_json|parameter_defs_json|default_images_json|image_candidates_json|docker_options_json|model_mount_json" internal cmd web scripts configs docs
```

分类：

| 分类 | 是否允许 |
|---|---|
| configs/config-registry | 允许 |
| configs/backend-catalog | 允许 |
| tests fixture | 允许 |
| docs/design 或 docs/archive | 允许 |
| backend adapter abstraction | 谨慎允许 |
| db.go catalog literal | 禁止 |
| preflight/runplan/docker spec backend-name business logic | 禁止 |
| active API response authority old fields | 禁止 |
| active runtime smoke bypass | 禁止 |
| legacy fallback | 禁止 |

## 2. Registry / Catalog 测试

必须覆盖：

```text
ConfigItem code 唯一
category 合法
kind 合法
type 合法
render.target 合法
render.style 合法
order 合法
support_level 合法
registry default_value 符合 type
catalog 引用的 ConfigItem code 存在
BackendVersion supported_config_items 存在
BackendRuntime 使用的 config_items 存在
active BackendVersion config_set 非空
active BackendRuntime config_set 非空
无 array-format capabilities
无 seed-only deprecated backend version
```

## 3. Copy-on-create 测试

### Backend → BackendVersion

断言：

```text
BackendVersion 创建时 copy Backend ConfigSet
BackendVersion apply version overrides
BackendVersion 保存完整 materialized ConfigSet
修改 Backend 后，已有 BackendVersion 不变
```

### BackendVersion → BackendRuntime

断言：

```text
BackendRuntime 创建时 copy BackendVersion ConfigSet
BackendRuntime apply launcher/runtime overrides
BackendRuntime 保存完整 materialized ConfigSet
修改 BackendVersion 后，已有 BackendRuntime 不变
```

### BackendRuntime → NBR

断言：

```text
NBR enable/create 时 copy BackendRuntime ConfigSet
NBR apply node overrides
NBR 保存完整 materialized ConfigSet
修改 BackendRuntime 后，已有 NBR 不变
check-request 使用 NBR snapshot
```

### NBR → Deployment / RunPlan

断言：

```text
Deployment 创建时 copy NBR ConfigSet
Deployment apply deployment overrides
RunPlan 只读 Deployment ConfigSet
修改 NBR 后，已有 Deployment/RunPlan 不变
```

## 4. Renderer 测试

### CLI renderer

覆盖：

```text
flag_space_value
flag_equals_value
flag_if_true
repeat_flag
positional
raw_lines
order sorting
disabled item skipped
```

### Docker renderer

覆盖：

```text
launcher.image
launcher.ports
launcher.volumes
launcher.devices
launcher.privileged
launcher.shm_size
runtime.env
runtime.visible_devices
runtime.model_mount
model_runtime cli args
```

### Backend examples

覆盖：

```text
vLLM renders --gpu-memory-utilization / --max-model-len / --trust-remote-code
SGLang renders --mem-fraction-static / --context-length / --trust-remote-code
llama.cpp renders -m / --ctx-size / --n-gpu-layers
```

这些必须从 ConfigItem.render 来，不允许 backend-name hardcode。

## 5. extra 验证

### backend.extra_args

测试：

```text
--flag
--flag value
--flag=value
unknown flag warning allowed
duplicate structured flag error
raw_lines order after structured args
```

### runtime.extra_env

测试：

```text
KEY=VALUE
unknown env warning allowed
duplicate structured env error
```

### launcher.extra_options

测试：

```text
unknown launcher option warning allowed
duplicate structured option error
```

## 6. Fresh DB 验证

使用独立目录：

```bash
export LIGHTAI_HOME=/tmp/lightai-configset-fresh-db
rm -rf "$LIGHTAI_HOME"
mkdir -p "$LIGHTAI_HOME"
```

验证：

```text
fresh DB 由 YAML + registry 生成
db.go 不含 backend catalog literal
backends.config_set_json 非空
backend_versions.config_set_json 非空
backend_runtimes.config_set_json 非空
active vLLM/SGLang/llama.cpp version/runtime config_set 完整
无 seed-only deprecated versions
无 manual DB update
```

SQLite 检查示例按最终 schema 调整：

```bash
sqlite3 "$LIGHTAI_HOME/data/lightai.db" ".schema backends"
sqlite3 "$LIGHTAI_HOME/data/lightai.db" ".schema backend_versions"
sqlite3 "$LIGHTAI_HOME/data/lightai.db" ".schema backend_runtimes"
sqlite3 "$LIGHTAI_HOME/data/lightai.db" ".schema node_backend_runtimes"
sqlite3 "$LIGHTAI_HOME/data/lightai.db" "select id, length(config_set_json), length(source_metadata_json) from backend_versions;"
```

## 7. API / UI 验证

禁止 active API 出现旧权威字段：

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

允许 response 中出现从 ConfigSet 派生的 preview 字段，但不能作为 create/update 权威 payload。

UI 验证：

```text
参数按 launcher/runtime_env/model_runtime 分类展示
常用结构化参数可编辑
未启用也显示输入框
extra_args 每行一个
extra_env 每行一个
重复结构化 flag/env 报错
保存后 last_modified 记录正确
RunPlan preview 来自 ConfigSet renderer
```

## 8. Runtime Smoke 验证

每条 runtime evidence 必须包含：

```text
environment
catalog materialization
BackendRuntime config_set
NBR config_set
Deployment config_set
ResolvedRunPlan
AgentRunSpec
DockerSpec
container id
docker inspect
health or /v1/models response
inference response
logs tail
stop response
cleanup log
final docker ps
```

三条必须本轮真实 PASS：

```text
vLLM
SGLang
llama.cpp
```

不接受：

```text
preflight PASS only
task claimed only
previously demonstrated
image/model present only
direct docker run bypass
```

## 9. 全量验证命令

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

## 10. Final closeout 验收

最终 closeout 必须包含：

```text
设计文档路径
归档文档路径
删除旧字段/旧 API/旧兼容逻辑清单
db.go hardcode 删除证据
registry/catalog loader 实现路径
ConfigSet copy-on-create 测试结果
renderer 测试结果
extra_args / extra_env 重复排除测试结果
fresh DB verification
vLLM / SGLang / llama.cpp platform-chain smoke
OpenAPI current contract
active E2E stale gate
全量测试结果
commit id
push result
git status --short 原文
open task = 0
```
