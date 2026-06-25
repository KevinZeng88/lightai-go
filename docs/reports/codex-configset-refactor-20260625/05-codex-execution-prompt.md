# Codex 执行 Prompt：LightAI Go ConfigSet / ConfigItem 重构

你现在接手 LightAI Go 的配置模型重构。请不要继续沿着旧结构补洞，不要只修 SGLang capability、drift test、db.go seed literal。

## 项目

```bash
cd /home/kzeng/projects/ai-platform-study/lightai-go
```

默认在当前分支/main 工作，不新建分支，除非用户另行要求。

## 目标

按 `ConfigSet / ConfigItem` 模型重构 Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / Deployment / RunPlan 链路。

这次不需要兼容旧 DB、旧 API、旧 payload、旧 UI、旧文档。允许重建 DB。旧结构、旧字段、旧 seed、旧兼容逻辑、过时文档必须删除或归档。

## 必须先读取/写入的文档

请把以下设计写入仓库：

```text
docs/design/catalog-configset-and-runtime-snapshot.md
```

设计内容以本交接包的 `02-configset-configitem-design.md` 为准。

然后读取：

```text
01-current-code-findings.md
02-configset-configitem-design.md
03-implementation-plan.md
04-validation-and-acceptance.md
```

## 当前已知问题

1. `internal/server/db/db.go` 仍含 vLLM / SGLang / llama.cpp backend catalog seed literal。
2. `seedBuiltInBackends()` 和 `seedTargetBackendCatalog()` 形成多套 catalog。
3. V27 repair / manual capability repair 说明 catalog materialization 不可靠。
4. YAML 文件目前仍有 “DB capabilities_json 是 canonical，YAML 是 mirror” 的反向口径。
5. API/DB 仍有旧字段，如 `capabilities_json`、`parameter_schema_json`、`parameter_values_json`、`env_json`、`docker_json`、`model_mount_json`、`health_check_json` 等。
6. Runtime smoke 曾错误接受 preflight PASS / task claimed / previously demonstrated。
7. 近期 drift test 只是证明双源存在，不能替代去硬编码。

## 设计原则

### ConfigItem

每个配置项至少包含：

```text
code
category = launcher / runtime_env / model_runtime
kind
type
value
default_value
enabled
render
order
constraints
support_level
source
last_modified
```

### ConfigSet

每一层保存完整 materialized ConfigSet：

```text
context
items
source_metadata
```

### 三类参数

```text
launcher      启动载体参数，支持 docker，预留 process/systemd/k8s
runtime_env   运行环境参数
model_runtime 模型/后端进程参数
```

### extra

```text
backend.extra_args       每行一个 CLI 参数
runtime.extra_env        每行一个 KEY=VALUE
launcher.extra_options   每行一个启动载体 option
```

extra 不允许重复结构化 flag/env/option。unknown extra warning，允许继续。

### 校验

偏宽松，尽力跑通。只有明确无法运行或重复冲突才 error；其他 warning。

## 必须删除/替换的旧字段

不作为 DB/API 权威继续保存：

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

如短期内部需要投影，只能由 ConfigSet 派生，不得继续作为 create/update 权威。

## 必须去除的硬编码

`internal/server/db/db.go` 不得继续包含 vLLM/SGLang/llama.cpp 的：

```text
backend/version/runtime seed literal
capabilities literal
model format literal
CLI args literal
image literal
health check literal
runtime defaults literal
deprecated seed-only bver-* version
```

`db.go` 只能调用通用 catalog loader / validator / materializer / seeder。

不接受：

```text
YAML authoritative + db.go bootstrap mirror + drift test
```

这仍然是双源硬编码。

## 实现步骤

### Phase 0：baseline

输出：

```text
docs/reports/configset-refactor-20260625/00-baseline.md
```

包含：

```bash
git status --short
git log --oneline -20
rg 旧字段和 backend-name hardcode 的结果摘要
```

### Phase 1：设计文档与文档归档

1. 写入 `docs/design/catalog-configset-and-runtime-snapshot.md`。
2. 归档过时文档到 `docs/archive/<date>-pre-configset-catalog-model/`。
3. 归档文档顶部加：
   ```text
   Archived. Superseded by docs/design/catalog-configset-and-runtime-snapshot.md.
   ```

### Phase 2：Config registry / catalog loader

新增：

```text
configs/config-registry/
internal/server/catalog/
```

实现：

```text
LoadRegistry
LoadBackendCatalog
ValidateRegistry
ValidateCatalog
MaterializeBackend
MaterializeBackendVersion
MaterializeBackendRuntime
SeedCatalog
```

### Phase 3：DB schema 重建

不做历史兼容迁移。

目标字段：

```text
config_set_json
source_metadata_json
```

用于：

```text
backends
backend_versions
backend_runtimes
node_backend_runtimes
model_deployments
```

清理旧字段和旧 repair/fallback。

Do not add a V29-style compatibility migration that only appends config_set_json/source_metadata_json while preserving old authority columns. Rebuild/replace the schema for the fresh-DB baseline instead.

Do not keep legacy columns to avoid API breakage. Fix API/UI/tests to the ConfigSet contract in the same clean-state commit range.

Do not commit a checkpoint that introduces dual authority, legacy fallback, or temporary compatibility paths.

### Phase 4：copy-on-create

实现：

```text
BackendVersion = copy(Backend ConfigSet) + version overrides
BackendRuntime = copy(BackendVersion ConfigSet) + runtime overrides
NBR = copy(BackendRuntime ConfigSet) + node overrides
Deployment = copy(NBR ConfigSet) + deployment overrides
RunPlan = render(Deployment ConfigSet)
```

确保 parent 修改不 live mutation child。

### Phase 5：renderer

实现通用 renderer：

```text
cli
env
docker.image
docker.port
docker.volume
docker.device
docker.option
health
raw_lines
```

禁止 backend-name 拼接参数。

### Phase 6：API/UI

API 使用：

```json
{
  "config_set": {},
  "config_overrides": {},
  "source_metadata": {}
}
```

UI 按 category 展示 ConfigItem，并支持 extra_args / extra_env。

### Phase 7：测试、fresh DB、runtime smoke

必须完成：

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

Fresh DB 验证和三 runtime platform-chain smoke。

## Runtime smoke 必须通过平台链路

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

不接受：

```text
preflight PASS only
previously demonstrated
task claimed only
image/model present only
direct docker run bypass
```

## 提交规则

禁止：

```bash
git add .
```

使用：

```bash
git status --short
git diff --stat
git diff --check
git add <explicit files only>
git commit -m "refactor: replace backend catalog seeds with configset snapshots"
git push
```

## 阶段性输出

这次可以阶段性 commit/push，但不要每个小问题都停下来问用户。普通测试失败、脚本失败、runtime smoke 暴露真实 bug，均自行修复后重跑。

只有以下情况允许停止：

```text
Docker/GPU/image/model 外部资源不可用且有命令级证据
Git push 凭据/网络失败
需要删除未纳入 baseline 的用户数据
需要引入明显超出本设计的大型架构
```

## 最终输出

必须包含：

```text
1. 设计文档路径
2. 归档文档路径
3. 删除的旧字段 / 旧 API / 旧兼容逻辑
4. db.go 是否仍含 vLLM/SGLang/llama.cpp catalog literal
5. registry/catalog loader 实现路径
6. ConfigSet copy-on-create 测试结果
7. renderer 测试结果
8. extra_args / extra_env 重复排除测试结果
9. fresh DB verification 结果
10. vLLM / SGLang / llama.cpp platform-chain smoke 结果
11. 全量测试结果
12. commit id
13. push result
14. git status --short 原文
15. open task 是否为 0
```

在全部完成前，不得输出 `Complete` 或 `FINAL_CLOSEOUT_ACCEPTED`。
