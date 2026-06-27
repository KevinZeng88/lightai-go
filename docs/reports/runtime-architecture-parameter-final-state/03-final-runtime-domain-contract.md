# Final Runtime Domain Contract

## 1. 目标

本文定义 Runtime 架构最终领域对象边界。执行端应以本文作为代码修复和 E2E 验收的领域契约。

核心原则：

1. 领域对象职责单一；
2. 参数定义单一属主；
3. 层级对象 copy-on-create；
4. Deployment 只保存部署意图和覆盖；
5. ResolvedRunPlan 是最终执行权威；
6. Instance 记录运行事实。

## 2. 领域对象总览

```text
Backend
BackendVersion
BackendCapabilityProfile
RuntimeRequirements
BackendRuntime
NodeBackendRuntime
ModelArtifact
ModelLocation
ParameterDefinition
ParameterOverride
ParameterValue
ResolvedParameter
ParameterSourceMap
Deployment
ResolvedRunPlan
DeviceBinding
HealthCheck
ModelInstance
```

## 3. Backend

Backend 表示推理后端类型，例如 vLLM、SGLang、llama.cpp。

Backend 拥有：

1. 后端标识；
2. 后端名称；
3. 后端描述；
4. 后端级能力摘要；
5. 后端级参数定义中真正属于 Backend 的部分。

Backend 不拥有：

1. 节点镜像检查结果；
2. 具体 GPU vendor 设备绑定；
3. 本机模型路径；
4. 某次部署的 override；
5. 实际容器状态。

## 4. BackendVersion

BackendVersion 表示后端版本能力，例如某一版本 vLLM 支持哪些参数和 endpoint。

BackendVersion 拥有：

1. 版本标识；
2. 支持的模型格式；
3. 支持的服务协议；
4. OpenAI compatible endpoint 能力；
5. version-scoped BackendCapabilityProfile；
6. version-scoped RuntimeRequirements；
7. version-scoped ParameterDefinition。

BackendVersion 保持硬件无关。GPU vendor、设备路径、节点 Docker runtime 状态不写入 BackendVersion。

## 5. BackendCapabilityProfile

BackendCapabilityProfile 描述后端能力。它回答“这个后端/版本能做什么”。

它可以描述：

1. model formats；
2. endpoints；
3. supported parameters；
4. resource control capability；
5. health check capability；
6. streaming capability；
7. embedding/chat/rerank capability；
8. device binding abstraction support。

它不描述：

1. 某节点是否已拉取镜像；
2. 某部署选择了哪个端口；
3. 某本机模型路径；
4. 某容器运行状态。

## 6. RuntimeRequirements

RuntimeRequirements 描述运行成功所需条件。它回答“要运行起来，需要满足什么”。

它可以描述：

1. required image；
2. Docker runtime；
3. accelerator requirement；
4. device binding；
5. model path requirement；
6. required files；
7. mount rules；
8. port rules；
9. env rules；
10. health check rules；
11. required args；
12. warning / blocking error 边界。

RuntimeRequirements 应能驱动 Preflight、RunPlan、UI 提示、Agent check 和 E2E 断言。

## 7. BackendRuntime

BackendRuntime 表示运行模板。

BackendRuntime 的职责：

1. 选择 BackendVersion；
2. 定义模板级 image、command、args/env/mounts/ports 默认规则；
3. 定义模板级参数默认值或模板级 override；
4. 定义模板级健康检查默认规则；
5. 作为 NodeBackendRuntime 的上层快照来源。

BackendRuntime 创建或克隆时：

1. 拷贝 BackendVersion 当时的有效能力视图；
2. 保存模板自己拥有的数据；
3. 保存模板级 override；
4. 不修改 BackendVersion；
5. 不把 BackendVersion 参数 schema 改成 BackendRuntime owner。

## 8. NodeBackendRuntime

NodeBackendRuntime 表示某节点启用的运行环境。

NodeBackendRuntime 的职责：

1. 显式 enable 某个 BackendRuntime 到某节点；
2. copy-on-create BackendRuntime 当时的有效视图；
3. 保存节点级配置；
4. 保存节点级 override；
5. 保存 check-request evidence；
6. 保存 image inspect 结果；
7. 保存 Docker runtime / device / path / health check evidence；
8. 作为 Deployment 的唯一部署入口。

NodeBackendRuntime 不由 Deployment 自动创建。

## 9. ModelArtifact

ModelArtifact 表示模型制品。它拥有模型自身信息：

1. 模型名称；
2. 模型家族；
3. 模型格式；
4. 模型能力；
5. 上下文能力；
6. 量化信息；
7. discovered metadata；
8. 模型级参数定义中真正属于模型的部分。

ModelArtifact 不拥有：

1. Docker 镜像；
2. Docker args；
3. GPU runtime；
4. 节点路径检查结果；
5. 部署端口。

## 10. ModelLocation

ModelLocation 表示模型在某个节点或路径上的实例位置。

它拥有：

1. path；
2. node_id；
3. file type；
4. scan evidence；
5. size / checksum / detected format；
6. availability evidence。

ModelLocation 是保存 `/home/kzeng/models/...` 这类路径的位置。通用 catalog 和模型 metadata 不保存本机实例路径。

## 11. ParameterDefinition

ParameterDefinition 是参数 schema。它只能有一个 owner。

必须包含或等价表达：

1. owner_type；
2. owner_id 或 owner_key；
3. key；
4. label；
5. category；
6. scope；
7. target；
8. type；
9. required；
10. default_value；
11. default_enabled；
12. editable_at；
13. visibility；
14. advanced；
15. order；
16. constraints；
17. choices；
18. depends_on / show_when；
19. arg_name / env_name / mount_target / port_target；
20. help_text。

同一个 owner + key 是唯一参数定义。

## 12. ParameterOverride

ParameterOverride 表示某层级对已有参数的覆盖。

它必须引用：

1. definition_owner_type；
2. definition_owner_key 或 definition_id；
3. parameter_key；
4. override_owner_type；
5. override_owner_id；
6. enabled；
7. value；
8. source；
9. reason / user_visible_note。

ParameterOverride 不包含完整 schema。它不能重新定义 label、category、target、type、arg_name 等 schema 字段。

## 13. copy-on-create 层级链

层级链：

```text
BackendVersion / ModelArtifact
        ↓ copy-on-create
BackendRuntime
        ↓ copy-on-create
NodeBackendRuntime
        ↓ copy-on-create
Deployment
        ↓ resolve
ResolvedRunPlan
        ↓ execute
ModelInstance
```

规则：

1. 创建下一层时，拷贝上一层当时的有效视图；
2. 下一层只叠加自己拥有的数据或 override；
3. 上一层后续修改不影响已创建的下一层；
4. 下一层修改不影响上一层；
5. 克隆对象时保留 owner、key、value、enabled、source，不扩大 checked 范围；
6. copy-on-create 不能把所有上层参数 schema 复制成当前层 schema。

## 14. Deployment

Deployment 表示一次部署意图。

Deployment 拥有：

1. model artifact/location 选择；
2. node_backend_runtime_id；
3. deployment-level override；
4. resource override；
5. ports / mounts / health check override；
6. snapshot of selected NBR effective view；
7. desired state。

Deployment 不拥有：

1. BackendRuntime schema 定义；
2. NodeBackendRuntime schema 定义；
3. 自动创建 NBR 的逻辑；
4. 实际容器运行结果。

## 15. ResolvedRunPlan

ResolvedRunPlan 是最终执行权威。它拥有最终合成结果：

1. image；
2. command；
3. args；
4. env；
5. mounts；
6. ports；
7. devices；
8. health check；
9. resource controls；
10. labels；
11. parameter_source_map；
12. warnings；
13. errors。

ResolvedRunPlan 的输出必须同时供 preview 和 Agent Docker create 使用。

## 16. ModelInstance

ModelInstance 表示运行事实。

它拥有：

1. instance id；
2. deployment id；
3. container id；
4. actual Docker spec summary；
5. lifecycle status；
6. health status；
7. logs pointer；
8. error reason；
9. operation_id。

Instance 不编辑运行参数。参数修改应通过 Deployment 新建或重新部署流程完成。
