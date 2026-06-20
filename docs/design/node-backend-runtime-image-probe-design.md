# NodeBackendRuntime Image Probe 与节点级运行校验设计

## 1. 背景

当前 LightAI Go 在 Web「新增节点运行配置」向导中出现过一个关键问题：

用户从目标节点 Agent 返回的 Docker image 列表中选择了：

```text
vllm/vllm-openai:latest
```

但最终校验仍报：

```text
镜像缺失
docker image vllm/vllm-openai:latest is not present on node ...
```

这说明当前 NBR 向导的镜像列表链路和最终校验链路之间存在不一致。
修复该 blocker 后，还需要进一步完善 NBR 的镜像探测、运行参数展示、节点级校验、Start Wizard / RunPlan 使用方式。

本设计用于定义：

1. NodeBackendRuntime 中镜像探测信息应该放在哪里；
2. 多节点 NBR 下镜像元信息如何按节点保存；
3. NBR 向导、NBR 详情页、列表页、Start Wizard 中如何展示和使用这些信息；
4. 如何设计测试，避免再依赖人工操作页面才发现类似问题。

## 2. 核心结论

### 2.1 镜像探测信息必须是节点级

不能假定同一个 NBR 下多个节点上的镜像元信息完全一致。

例如同一个 image ref：

```text
vllm/vllm-openai:latest
```

在不同节点上可能对应不同内容：

```text
node-A:
  image_id: sha256:aaa
  framework_version: 0.8.x
  entrypoint: python -m vllm.entrypoints.openai.api_server

node-B:
  image_id: sha256:bbb
  framework_version: 0.9.x
  entrypoint: /bin/bash /start.sh
```

尤其是 `latest` tag、厂商重打包镜像、离线导入镜像、不同节点更新时间不一致时，这种差异非常常见。

因此：

```text
Probe Snapshot 必须至少按 node_id + image_ref 维度保存。
```

如果一个逻辑 NBR 覆盖多个节点，则每个节点都必须有独立 Probe Snapshot。

### 2.2 Probe Snapshot 是证据，不是配置真相

NBR 中应区分两类数据。

第一类是用户配置，表示“用户希望这个节点怎么运行”：

```text
image_ref
ports
volumes
devices
env
extra_args
privileged
ipc
shm_size
ulimits
group_add
security_opt
health_check
```

第二类是 Probe Snapshot，表示“平台在这个节点上实际查到了什么”：

```text
image_id
repo_tags
repo_digests
entrypoint
cmd
env
exposed_ports
labels
framework_version
backend_match_status
warnings
errors
checked_at
```

Probe Snapshot 不能静默覆盖用户配置。
如果要使用探测结果填充端口、命令、env 或 health check，必须由用户明确点击确认。

### 2.3 最终校验应以 ImageInspect 为准

`/docker-images` 列表适合用于 UI 选择，但不应作为最终 `missing_image` 的唯一依据。

最终存在性校验应优先使用目标节点 Agent 的 Docker ImageInspect：

```text
ImageInspect success       -> image exists
Docker not found           -> missing_image
Agent unreachable          -> agent_unreachable
Docker API error           -> docker_error
Inspect failed non-notfound -> inspect_failed
```

不能把以下状态映射成 `missing_image`：

```text
evidence_missing
agent_unreachable
docker_error
inspect_failed
probe_failed
version_unknown
backend ambiguous
script_probe_failed
```

只有 Docker 明确确认目标节点不存在该 image，才允许返回 `missing_image`。

## 3. 概念模型

建议明确区分以下对象。

### 3.1 Backend / BackendVersion

描述推理后端及版本能力，例如：

```text
vLLM
SGLang
llama.cpp
Ollama
```

Backend / BackendVersion 应偏系统 catalog，描述框架能力、默认参数、版本探测规则、匹配规则等。
它不应绑定具体节点，也不应绑定具体 GPU 厂商。

### 3.2 BackendRuntime

描述一个逻辑运行配置模板，例如：

```text
vLLM + Docker + NVIDIA 默认参数
vLLM + Docker + MetaX 默认参数
SGLang + Docker + NVIDIA 默认参数
llama.cpp + Docker + CUDA 默认参数
```

BackendRuntime 可以带 vendor/runtime 适配信息，但仍然不是某个具体节点上的真实运行证据。

### 3.3 NodeBackendRuntime

描述某个节点或某组节点上的运行配置。

如果当前系统允许一个 NBR 覆盖多个节点，则建议在概念上拆分：

```text
逻辑 NBR
  - backend_runtime_id
  - backend_id
  - backend_version_id
  - 用户级默认配置

NBR Node Binding
  - nbr_id
  - node_id
  - image_ref
  - ports
  - volumes
  - devices
  - env
  - extra_args
  - security options
```

如果现有数据库中 NBR 已经是节点级对象，则可以继续使用当前 NBR 作为 node binding。
但无论现有模型如何，Probe Snapshot 都必须落到具体 node 维度。

### 3.4 NodeBackendRuntimeProbeSnapshot

描述某个节点上某个 image ref 在某次检查时的实际证据。

推荐设计独立表，而不是只在 NBR 上保存一份全局 JSON。

建议表名：

```text
node_backend_runtime_probe_snapshots
```

建议字段：

```text
id
node_backend_runtime_id
node_id
agent_id
image_ref
image_id
repo_tags_json
repo_digests_json
os
architecture
size_bytes
image_created_at
working_dir
user
entrypoint_json
cmd_json
env_json
exposed_ports_json
volumes_json
labels_json
healthcheck_json
inspect_json
startup_script_path
startup_script_preview
detected_backend_command
detected_args_json
backend_match_status
backend_match_method
backend_match_detail
framework_version
version_probe_status
version_probe_output
warnings_json
errors_json
final_status
checked_at
operation_id
created_at
updated_at
```

建议索引：

```text
(node_backend_runtime_id, node_id)
(node_id, image_ref)
(node_id, image_ref, image_id)
(node_backend_runtime_id, final_status)
```

如果短期不加表，也至少应在 NBR JSON 中按节点分组：

```json
{
  "by_node": {
    "node-a": {
      "image_ref": "vllm/vllm-openai:latest",
      "image_id": "sha256:aaa",
      "checked_at": "..."
    },
    "node-b": {
      "image_ref": "vllm/vllm-openai:latest",
      "image_id": "sha256:bbb",
      "checked_at": "..."
    }
  }
}
```

但长期建议使用独立表，便于查询、重检、聚合状态、过期判断和 Start Wizard 使用。

## 4. Probe 流程设计

### 4.1 Level 1：节点与 Agent 解析

输入：

```text
node_id
image_ref
backend_id
backend_version_id
node_backend_runtime_id
```

处理：

```text
resolve node -> agent
```

结果：

```text
success             -> 继续
node not found      -> node_not_found
agent unresolved    -> agent_unresolved
agent unreachable   -> agent_unreachable
```

日志必须包含：

```text
operation_id
node_id
agent_id
image_ref
backend_id
backend_version_id
```

### 4.2 Level 2：Docker ImageInspect

最终存在性校验应以 Docker ImageInspect 为准。

流程：

```text
agent Docker ImageInspect(image_ref)
```

结果：

```text
inspect success       -> image exists，进入 metadata capture
docker image notfound -> missing_image
docker api error      -> docker_error
inspect failed        -> inspect_failed
agent unreachable     -> agent_unreachable
```

要求：

1. `/docker-images` 列表只能作为 UI 选择来源，不能作为最终 missing_image 的唯一依据；
2. ImageInspect 成功即表示 image exists；
3. ImageInspect not found 才能返回 missing_image；
4. 其他失败不能映射成 missing_image。

### 4.3 Level 3：ImageInspect 元信息采集

成功 inspect 后，采集并保存：

```text
image_id
repo_tags
repo_digests
os
architecture
size
created
working_dir
user
entrypoint
cmd
env
exposed_ports
volumes
labels
healthcheck
stop_signal
shell
```

这些信息用于：

1. 展示镜像默认启动方式；
2. 辅助判断端口、entrypoint、cmd；
3. 诊断为什么 RunPlan 启动失败；
4. 辅助 Start Wizard / Preflight；
5. 检测同一 image_ref 在不同节点上的 image_id 差异。

### 4.4 Level 4：Backend 类型匹配

Backend 类型匹配必须避免把 GPU vendor 当成 backend。

错误逻辑示例：

```text
vendor = nvidia -> backend = vllm
```

这是错误的。
NVIDIA、MetaX、Huawei、Hygon 等是硬件/运行环境维度；vLLM、SGLang、llama.cpp、Ollama 是推理后端维度。两者不能互相推导。

匹配规则应优先来自 Backend catalog / BackendVersion 配置，而不是硬编码在 handler 里。

建议匹配状态：

```text
confirmed_match
probable_match
declared_match_unverified
ambiguous
confirmed_mismatch
```

解释：

```text
confirmed_match:
  强证据证明镜像匹配当前 backend，例如 image name、label、entrypoint、version probe 明确命中。

probable_match:
  有弱证据证明可能匹配。

declared_match_unverified:
  用户选择了该 backend，但镜像元信息无法证明。厂商自建镜像常见。

ambiguous:
  无法判断。

confirmed_mismatch:
  明确证据证明镜像是另一个 backend。
```

要求：

1. 镜像名不包含 vllm/sglang/llama.cpp，不能直接判 mismatch；
2. labels 不规范，不能直接判 mismatch；
3. entrypoint 是 bash/sh，不能直接判 mismatch；
4. version probe 失败，不能直接判 mismatch；
5. 只有明确识别为另一个 backend，才允许 confirmed_mismatch。

### 4.5 Level 5：静态 Script Probe

如果 entrypoint/cmd 中包含明显脚本路径，可以尝试静态读取脚本内容。

候选路径示例：

```text
/start.sh
/entrypoint.sh
/docker-entrypoint.sh
/usr/local/bin/*.sh
/opt/*/start*.sh
/opt/*/entrypoint*.sh
```

建议 Agent 使用：

```text
docker create <image>
docker cp <container>:/path/to/script -
docker rm <container>
```

要求：

1. 不执行容器；
2. 不挂载宿主目录；
3. 不传 GPU；
4. 不传 privileged；
5. 内容截断，例如 16KB 或 32KB；
6. 读取失败只产生 warning；
7. 不能阻断保存；
8. 不能映射成 missing_image。

可从脚本中 best-effort 提取：

```text
python -m vllm...
vllm serve...
python -m sglang...
sglang.launch_server...
llama-server...
ollama serve...
--host
--port
--model
--served-model-name
--tensor-parallel-size
--gpu-memory-utilization
--max-model-len
--ctx-size
--n-gpu-layers
```

注意：脚本解析结果只是诊断信息，不是强校验依据。

### 4.6 Level 6：Version Probe

Version Probe 是主动执行镜像，必须谨慎，默认 best-effort、非阻断。

Probe 命令应来自 Backend catalog / BackendVersion 配置，不要在 Go 代码里写死大量规则。

执行要求：

```text
--pull=never
--network=none
--rm
no GPU
no mounts
no privileged
--cap-drop=ALL
--security-opt no-new-privileges
timeout 5-10s
stdout/stderr 截断
```

失败结果：

```text
version_unknown
probe_failed
probe_timeout
```

这些结果只能作为 warning，不能阻断保存，也不能映射成 missing_image。

## 5. 状态模型

### 5.1 节点级 Probe 状态

建议节点级状态包括：

```text
ready
ready_with_warnings
missing_image
agent_unreachable
agent_unresolved
docker_error
inspect_failed
runtime_image_mismatch
version_unknown
probe_failed
evidence_missing
needs_recheck
stale
```

### 5.2 阻断错误

以下状态默认阻断该节点作为可用 NBR 候选：

```text
missing_image
agent_unreachable
agent_unresolved
docker_error
inspect_failed
runtime_image_mismatch，且 strict=true
```

### 5.3 非阻断 warning

以下状态默认不阻断保存：

```text
version_unknown
probe_failed
script_probe_failed
backend ambiguous
declared_match_unverified
entrypoint is shell wrapper
exposed ports missing
labels incomplete
image_id differs across nodes
probe stale
```

### 5.4 多节点聚合状态

如果一个逻辑 NBR 覆盖多个节点，列表页只能显示聚合摘要，不能替代节点级详情。

建议聚合规则：

```text
all ready                         -> ready
some ready, some warning          -> ready_with_warnings
some ready, some error            -> partially_available
all error                         -> unavailable
not checked or stale              -> needs_recheck
same image_ref but image_id drift -> ready_with_warnings
```

## 6. Web 展示设计

### 6.1 NBR 向导最后一步

「新增节点运行配置」最后一步建议命名为：

```text
校验与运行预览
```

如果一次添加多个节点，应展示节点级结果表：

```text
节点      镜像                         Image ID      状态                  警告
node-A   vllm/vllm-openai:latest       sha256:aaa    ready                 0
node-B   vllm/vllm-openai:latest       sha256:bbb    ready_with_warnings   2
node-C   vllm/vllm-openai:latest       -             missing_image         -
```

每个节点可展开查看：

```text
A. 镜像存在与元信息
B. 镜像默认启动参数
C. 探测与解析结果
D. 当前节点最终运行参数
```

### 6.2 镜像存在与元信息

展示：

```text
image ref
image id
repo tags
repo digests
os / architecture
size
created
checked_at
agent_id
operation_id
```

### 6.3 镜像默认启动参数

展示：

```text
entrypoint
cmd
working dir
user
env
exposed ports
volumes
healthcheck
labels
```

如果 entrypoint 是 bash/sh/python 等通用解释器，应显示友好说明：

```text
入口类型：Shell wrapper
说明：镜像默认通过 shell 启动，真实服务参数可能在 Cmd 或启动脚本中。
```

### 6.4 探测与解析结果

展示：

```text
backend_match_status
backend_match_detail
framework_version
version_probe_status
startup_script_path
startup_script_preview
detected_backend_command
detected_args
warnings
errors
```

厂商自建镜像如果无法确认 backend，应显示：

```text
状态：可用但有警告
原因：该镜像可能是厂商自定义封装，未从镜像名、labels、entrypoint 或版本探测中确认 backend。
```

不能显示为：

```text
镜像缺失
runtime mismatch
```

除非有明确证据。

### 6.5 当前节点最终运行参数

展示该节点 NBR 最终会使用的运行参数：

```text
selected node
resolved agent
backend
backend version
vendor/runtime
docker image
ports
volumes
devices
env
extra args
privileged
ipc
shm_size
ulimits
group_add
security_opt
health check
generated docker command preview
```

这里展示的是 NBR 基础运行参数。
完整模型部署命令仍应在 Start Wizard / Deployment RunPlan 中展示，因为它还需要叠加 ModelLocation、ModelArtifact 和 Deployment 参数。

### 6.6 NBR 列表页

建议显示：

```text
NBR 名称 / Backend
节点数
ready 节点数
warning 节点数
error 节点数
image_ref
last_checked
聚合状态
操作：重新校验、查看详情、编辑、克隆、删除
```

### 6.7 NBR 详情页

建议包含：

```text
概览
节点列表
镜像与探测
节点运行参数
运行预览 / RunPlan 模板
```

查看镜像信息时必须先明确当前节点：

```text
当前查看节点：node-A
```

不能展示一份全局镜像信息误导用户。

## 7. Start Wizard / Preflight 使用方式

Start Wizard 不能只判断“这个 NBR 是否可用”，而要判断：

```text
这个 NBR 在候选节点上是否可用。
```

候选项应基于：

```text
model_location.node_id
node_backend_runtime.node_id
probe_snapshot.node_id
```

只有同一个节点上同时满足：

```text
模型位置存在
运行配置可用
镜像 probe 可用
GPU/资源可用
```

才是有效候选。

如果 probe 过期或 image tag 变化，应提示重新校验。

## 8. latest tag 与 image_id drift

对于 `latest` 或其他可变 tag，必须保存：

```text
image_ref
image_id_at_probe
checked_at
```

如果后续 recheck 发现同一个 image_ref 在同一节点上的 image_id 变化，应提示：

```text
镜像 tag 已变化，建议重新确认运行参数。
```

如果同一个 image_ref 在不同节点上的 image_id 不一致，应提示：

```text
同一 image tag 在不同节点上对应不同 image id，请确认是否符合预期。
```

这不一定是错误，但必须可见。

## 9. 实施步骤建议

### Phase 0：当前 blocker 收口

目标：

修复 NBR 向导从镜像列表选择 image 后最终校验误报 `missing_image` 的问题。

验收：

```text
从 /docker-images 列表选择 vllm/vllm-openai:latest
最终校验/保存不再报 missing_image
ImageInspect 成功即认为 image exists
evidence_missing/docker_error/inspect_failed 不再映射成 missing_image
```

### Phase 1：状态模型与数据模型设计

目标：

明确 Probe Snapshot 的节点级存储方式。

输出：

```text
状态枚举
错误映射规则
node-scoped probe snapshot schema
migration 方案
API response schema
```

验收：

```text
同一个 NBR 下多个节点可以保存独立 probe snapshot
同一 image_ref 不同 image_id 可以表达
聚合状态不覆盖节点级状态
```

### Phase 2：Agent Docker Inspect 能力

目标：

Agent 提供稳定的 image inspect API。

建议 API：

```text
GET /docker-image-inspect?ref=<image_ref>
```

返回：

```text
exists
not_found
inspect_metadata
docker_error
```

验收：

```text
vllm/vllm-openai:latest inspect success
不存在镜像返回 not_found
Docker API 错误不返回 missing_image
```

### Phase 3：Agent Script Probe 能力

目标：

支持静态读取 entrypoint/cmd 指向的脚本。

建议 API：

```text
POST /docker-image-script-probe
```

要求：

```text
docker create + docker cp + docker rm
不执行容器
内容截断
失败 warning
```

验收：

```text
/start.sh 可读取并截断展示
脚本不存在返回 script_probe_failed warning
失败不阻断 NBR 保存
```

### Phase 4：Server NBR Probe API

目标：

Server 负责编排 node resolve、agent inspect、backend match、script probe、version probe，并保存节点级 snapshot。

建议 API：

```text
POST /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/probe
POST /api/v1/backend-runtimes/{nbr_id}/probe-all-nodes
GET  /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/probe
```

如果现有路由不同，可按现有风格调整，但必须真实 router 测试 path 参数。

验收：

```text
单节点 probe 可保存 snapshot
多节点 probe 每个节点独立保存 snapshot
router path node_id/nbr_id 正确
错误状态分层正确
```

### Phase 5：Backend Match Catalog 化

目标：

Backend match 规则从 catalog / BackendVersion 配置读取，避免硬编码。

要求：

```text
不能 vendor=nvidia -> backend=vllm
厂商自建镜像识别不出 backend 时返回 declared_match_unverified
明确冲突才 confirmed_mismatch
```

验收：

```text
vLLM 官方镜像 confirmed_match
SGLang 官方镜像 confirmed_match
llama.cpp 官方镜像 confirmed_match
MetaX 自建未知镜像 declared_match_unverified
NVIDIA CUDA 基础镜像不能被识别成 vLLM
```

### Phase 6：Web NBR 向导展示

目标：

最后一步展示节点级“校验与运行预览”。

验收：

```text
多节点结果表可见
每个节点可展开查看 ImageInspect
entrypoint/cmd/env/exposed_ports 可见
warnings/errors 分层展示
unknown 不显示为 missing_image
```

### Phase 7：NBR 列表与详情页

目标：

NBR 保存后可以查看节点级 Probe Snapshot。

验收：

```text
列表显示聚合状态
详情页必须选择节点查看镜像信息
image_id drift 可见
last_checked 可见
可手动 recheck
```

### Phase 8：Start Wizard / Preflight 集成

目标：

Start Wizard 使用候选节点对应的 Probe Snapshot。

验收：

```text
ready 节点可选
ready_with_warnings 节点可选但提示
missing_image 节点不可选
probe stale 提示重检
同一 image_ref 不同节点 image_id 不一致时提示 warning
```

### Phase 9：文档、测试、回归

目标：

将设计、操作说明、测试策略沉淀到文档和自动化测试。

验收：

```text
设计文档更新
API 文档更新
E2E 文档更新
测试全部通过
git status clean
```

## 10. 验收建议

### 10.1 基础验收

必须通过：

```text
1. 从目标节点 Docker image 列表选择 image 后，最终校验不误报 missing_image。
2. Docker ImageInspect 成功时，NBR probe 状态为 ready 或 ready_with_warnings。
3. Docker not found 时，才返回 missing_image。
4. Agent 不可达时，返回 agent_unreachable。
5. Docker API 错误时，返回 docker_error。
6. Inspect 非 notfound 失败时，返回 inspect_failed。
7. Version probe 失败时，返回 ready_with_warnings / version_unknown。
8. Script probe 失败时，返回 warning。
```

### 10.2 多节点验收

必须通过：

```text
1. 一个逻辑 NBR 覆盖多个节点时，每个节点有独立 probe snapshot。
2. node-A 和 node-B 使用同一个 image_ref 但 image_id 不同，页面显示 warning。
3. node-A ready，node-B missing_image 时，聚合状态为 partially_available。
4. Start Wizard 只把 ready 节点作为默认可选候选。
5. 查看镜像详情时必须明确节点。
```

### 10.3 厂商镜像验收

必须通过：

```text
1. 镜像名不包含 vllm/sglang/llama.cpp。
2. entrypoint 是 bash/sh。
3. labels 不规范或为空。
4. version probe 未定义或失败。
5. 用户选择 backend=vllm。
6. ImageInspect 成功。
7. 最终状态应为 ready_with_warnings / declared_match_unverified。
8. 不能返回 missing_image。
9. 不能返回 runtime_image_mismatch，除非有明确冲突证据。
```

### 10.4 安全验收

必须通过：

```text
1. Script probe 不执行容器。
2. Version probe 不挂载宿主目录。
3. Version probe 不传 GPU。
4. Version probe 不使用 privileged。
5. Version probe 使用 --pull=never。
6. Version probe 使用 --network=none。
7. Version probe 有超时。
8. stdout/stderr 截断。
```

## 11. 测试建议：不能依赖人工操作才发现问题

这类问题必须通过自动化测试提前发现。
不能等用户在 Web 页面手动点击到最后一步才暴露。

### 11.1 状态映射矩阵测试

建立完整状态矩阵：

```text
input condition                    expected status
---------------------------------------------------
inspect success                    ready
inspect success + warning          ready_with_warnings
docker not found                   missing_image
agent unreachable                  agent_unreachable
docker api error                   docker_error
inspect non-notfound error         inspect_failed
evidence missing                   evidence_missing
version probe failed               ready_with_warnings
script probe failed                ready_with_warnings
backend ambiguous                  ready_with_warnings
confirmed backend mismatch strict  runtime_image_mismatch
```

明确断言：

```text
只有 docker not found 可以返回 missing_image。
```

### 11.2 真实 HTTP Router 测试

必须覆盖真实路由，不只直接调用 handler。

测试：

```text
POST /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/probe
GET  /api/v1/nodes/{node_id}/backend-runtimes/{nbr_id}/probe
```

断言：

```text
PathValue("node_id") 正确
PathValue("nbr_id") 正确
node_id/nbr_id 不混淆
```

避免再次出现 route 参数名和 handler 读取名不一致的问题。

### 11.3 List-to-Probe 一致性测试

模拟 Agent：

```text
/docker-images 返回 vllm/vllm-openai:latest
/docker-image-inspect 对同一个 ref 返回 success
```

执行完整链路：

```text
前端/测试选择 list 中 image
提交最终 probe
期望 ready
```

断言：

```text
列表能看到的 image，最终不能误报 missing_image。
```

### 11.4 多节点同名镜像差异测试

模拟：

```text
node-A image_ref=vllm/vllm-openai:latest image_id=sha256:aaa
node-B image_ref=vllm/vllm-openai:latest image_id=sha256:bbb
```

断言：

```text
两个节点各自保存 snapshot
聚合状态 ready_with_warnings
页面/API 返回 image_id drift warning
Start Wizard 使用节点级 snapshot
```

### 11.5 厂商自建镜像测试

模拟：

```text
image_ref=registry.local/metax/runtime:latest
entrypoint=["/bin/bash"]
cmd=["/opt/maca/start.sh"]
labels={}
backend_id=vllm
version_probe fails
```

期望：

```text
ready_with_warnings
backend_match_status=declared_match_unverified
not missing_image
not runtime_image_mismatch
```

### 11.6 错误不得误映射测试

为每种错误单独测试：

```text
agent unreachable
docker timeout
inspect permission denied
version probe timeout
script probe failed
evidence missing
```

断言：

```text
均不得返回 missing_image。
```

### 11.7 API Contract / Golden JSON 测试

为 NBR probe response 建立 golden JSON。

覆盖：

```text
ready
ready_with_warnings
missing_image
agent_unreachable
multi-node partially_available
vendor image declared_match_unverified
image_id drift
```

确保前端字段变更能被测试发现。

### 11.8 前端组件测试

NBR 向导最后一步需要组件测试。

覆盖：

```text
ready 节点显示绿色/可保存
ready_with_warnings 节点显示 warning 但可保存
missing_image 节点阻断
version_unknown 不显示为镜像缺失
entrypoint bash 显示 shell wrapper 说明
多节点结果按节点展示
点击节点展开显示该节点 image metadata
```

### 11.9 Web E2E 测试

增加最小 E2E：

```text
启动 fake agent / test server
打开 NBR 向导
选择 node
从 image list 选择 image
进入校验与运行预览
确认 ready
保存 NBR
进入详情页
确认节点级 Probe Snapshot 可见
```

E2E 不必每次真实启动模型，但必须覆盖 list -> select -> probe -> save -> detail 的完整 UI 链路。

### 11.10 RunPlan 预览测试

断言：

```text
RunPlan 使用 NBR 用户配置
ImageInspect 默认值不会静默覆盖用户配置
探测到 exposed port 只产生提示，不自动改端口
```

## 12. 给 Claude 的执行要求

执行该设计前，应先完成当前 NBR 向导 image missing blocker 修复并提交。

随后 Claude 应：

```text
1. 审阅本文档设计。
2. 对比当前代码结构，指出差异和风险。
3. 将最终设计写入 docs/design/node-backend-runtime-image-probe-design.md。
4. 输出分阶段开发计划。
5. 本轮不要直接实现。
6. 等用户审核计划后，再按阶段实现。
```

实现时要求：

```text
1. 不新建分支，除非用户明确要求。
2. 不做大范围无关重构。
3. 不把 Probe Snapshot 当成配置真相。
4. 不静默用 ImageInspect 结果覆盖 NBR 用户配置。
5. 不把厂商镜像无法识别 backend 判成 mismatch。
6. 不把 version probe / script probe 失败判成 missing_image。
7. 不留下未记录的 future/follow-up。
8. 每个阶段必须有测试和验收。
9. 最终 git status --short 必须 clean。
```

## 13. 最重要的回归原则

这次问题暴露的根因不是单个 if 判断，而是测试缺少“完整用户链路”和“状态映射矩阵”。

后续所有类似能力都必须至少覆盖：

```text
UI 选择来源
最终提交 payload
Server handler 真实 router
Agent 返回
状态映射
Web 展示
保存后详情
Start Wizard 使用
```

不能只测 handler，也不能只测 API 单点。
必须有能发现以下问题的测试：

```text
列表能看到但最终校验失败
node_id / agent_id 混淆
path 参数名不一致
evidence missing 被误报 missing_image
厂商镜像被误判 runtime mismatch
同一 image_ref 多节点 image_id 不一致却被当成一致
Probe Snapshot 被当成全局信息
ImageInspect 默认值静默覆盖用户配置
```

这类问题必须通过测试在开发阶段发现，而不是等用户手动点页面发现。

