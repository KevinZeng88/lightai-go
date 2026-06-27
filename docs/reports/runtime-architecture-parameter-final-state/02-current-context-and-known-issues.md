# Current Context and Known Issues

## 1. 当前上下文

LightAI Go 当前 Runtime 架构已引入 Backend、BackendVersion、BackendRuntime、NodeBackendRuntime、ModelArtifact、ModelLocation、Deployment、RunPlan、NodeRunPlan、DeviceBinding 等概念。近期修复已经覆盖若干运行链路问题，但 Runtime 架构与参数体系仍需要最终收敛。

本专题不以局部 UI 修复为目标，而是要求把领域边界、参数属主、参数定义、运行要求、能力定义、预检、RunPlan、UI/API、E2E 统一起来。

## 2. 已知问题总览

需要重点核查和修复的问题组：

1. 模型 metadata 与本机模型路径混淆；
2. RuntimeRequirements 定义和使用链路不清；
3. BackendCapabilityProfile 定义和 RuntimeRequirements 边界不清；
4. 参数 owner、schema、value、override、source 边界不清；
5. 参数展示层级不清；
6. enabled / checked / default / required 语义混淆；
7. copy-on-create 快照链不完整或不可验证；
8. UI 参数编辑器存在历史问题；
9. RunPlan preview 与实际 Docker spec 存在分裂风险；
10. Preflight / check-request evidence 不完整；
11. API-first E2E 证据不足。

## 3. discovered_metadata_json 边界问题

必须检查：

1. `discovered_metadata_json` 是否写入具体本机模型路径；
2. catalog / seed / tests / docs 中是否存在 `/home/kzeng/models/...` 被当成通用定义；
3. 模型类别信息是否与模型实例路径混在一起；
4. 模型扫描结果是否被错误沉淀为通用模板；
5. RunPlan / Preflight / UI 是否依赖错误 metadata。

最终要求：

1. 模型类别信息归模型 metadata；
2. 模型实例路径归 ModelLocation；
3. 运行能力归 BackendCapabilityProfile；
4. 运行要求归 RuntimeRequirements；
5. 节点运行状态归 NodeBackendRuntime；
6. 部署覆盖归 Deployment；
7. 最终执行归 ResolvedRunPlan。

## 4. RuntimeRequirements 已知风险

需要检查 RuntimeRequirements 是否只是说明性字段。它必须能被翻译为：

1. 前端提示；
2. API 校验；
3. Preflight 检查；
4. Agent 环境检查；
5. RunPlan 构造；
6. E2E 断言。

必须覆盖：

1. container image；
2. Docker runtime；
3. GPU / accelerator；
4. device binding；
5. mounts；
6. ports；
7. env；
8. health check；
9. model format；
10. model path；
11. required files；
12. backend-specific args；
13. OpenAI compatible endpoint；
14. resource controls；
15. warning 与 blocking error。

## 5. BackendCapabilityProfile 已知风险

BackendCapabilityProfile 应描述后端能力，不描述节点状态或部署状态。必须检查是否混入：

1. 某个节点的 Docker image 检测结果；
2. 某个本机模型路径；
3. 某个部署的临时参数；
4. GPU vendor 的具体设备文件；
5. NodeBackendRuntime 的 check evidence。

BackendCapabilityProfile 应描述：

1. 支持的模型格式；
2. 支持的服务协议；
3. OpenAI compatible endpoint；
4. 参数能力；
5. 资源控制能力；
6. 健康检查能力；
7. 设备绑定能力抽象；
8. warning 场景。

## 6. 参数 owner / schema / override 问题

必须审查并修复以下风险：

1. 参数可能被多个层级重复定义；
2. 同名参数在 BackendRuntime、NodeBackendRuntime、Deployment 各自复制 schema；
3. UI 为了展示复制一份 schema；
4. Deployment 覆盖参数时重新定义 schema；
5. owner + key 没有作为 override 引用；
6. 参数 source 无法追踪；
7. RunPlan 无 parameter_source_map；
8. clone 可能扩大 checked 范围；
9. copy-on-create 快照链不清；
10. 上层变更可能污染已有下层对象；
11. 下层变更可能反向污染上层对象。

最终原则：

1. 一个参数只有一个 owner；
2. 一个参数只有一个 schema 定义位置；
3. 其他层级只保存 override value；
4. 每一层创建时 copy-on-create 上一层当时的有效视图；
5. 每一层只叠加自己这一层的 owner 数据或 override；
6. 只有 ResolvedRunPlan 合成全部参数。

## 7. 参数展示和 checked 语义问题

必须审查并修复以下风险：

1. 每个页面展示全部参数；
2. 参数没有分类；
3. 所有参数默认 checked；
4. 有 default value 的参数全部 checked；
5. required 参数被显示成用户 checked；
6. optional 参数默认进入 args；
7. advanced 参数默认展开；
8. disabled input 不显示值；
9. Model 页面展示 Docker 参数；
10. BackendRuntime / NodeBackendRuntime / Deployment 参数边界混乱；
11. RunPlan preview 看不到参数来源；
12. Deployment 页面覆盖参数时复制 schema。

最终要求：

1. 参数按 category 分组；
2. 不同页面只展示当前层级拥有或允许覆盖的内容；
3. default value 不等于 enabled；
4. required 不等于用户 checked；
5. optional 默认不 checked；
6. advanced 默认折叠；
7. disabled input 仍显示当前值；
8. RunPlan preview 展示最终值和来源。

## 8. UI/API 已知问题

必须检查：

1. RunnerConfigsPage 双入口；
2. legacy Docker editor 与 RuntimeParameterEditor 并存；
3. RuntimeParameterEditor 数据未 populate；
4. watch → emit 循环导致 OOM；
5. 只显示勾选框、不显示 disabled input；
6. 保存后 schema / value / enabled 丢失；
7. clone 后参数丢失或 checked 扩大；
8. Deployment 页面无法覆盖运行参数；
9. Deployment 页面端口、卷、健康检查配置不足；
10. Model 页面展示 Docker 参数；
11. Instance 状态刷新不准确；
12. Logs 自动刷新不足；
13. container id 与 instance id 混用；
14. ready_with_warnings 可部署口径不一致。

## 9. RunPlan / Preflight 已知问题

必须检查：

1. RunPlan preview 与实际 Docker create spec 是否一致；
2. env 是否混入 capabilities_json；
3. args 是否重复；
4. unchecked optional 参数是否进入 args；
5. resource controls 是否正确进入 args；
6. health check 与端口映射是否一致；
7. Docker image inspect 是否仍信任 client image_present；
8. check-request 是否由 Server 代理 Agent 获取 evidence；
9. errors/warnings 是否可断言；
10. API 和 UI 错误口径是否一致。

## 10. API-first E2E 缺口

最终验收需要覆盖：

1. fresh DB；
2. server / agent start；
3. login / CSRF；
4. BackendRuntime；
5. NodeBackendRuntime enable；
6. check-request；
7. model scan；
8. ModelArtifact / ModelLocation；
9. Deployment；
10. Preflight；
11. RunPlan preview；
12. start；
13. health check；
14. logs；
15. stop；
16. final state；
17. vLLM；
18. SGLang；
19. llama.cpp；
20. NVIDIA real smoke；
21. MetaX dry-run / structure check；
22. 参数 owner/source/checked/default/override 行为断言。
