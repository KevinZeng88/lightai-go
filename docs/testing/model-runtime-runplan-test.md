# LightAI Go 模型运行链路 / RunPlan 测试计划

> Phase 0.1 修订版 — 根据人工审核意见更新。
> 日期: 2026-06-16

---

## 1. BackendVersion 参数定义测试

### 1.1 参数定义解析
- **输入**：BackendVersion with parameter_defs = [{name:"max_model_len", cli_name:"--max-model-len", type:"integer"}]
- **预期**：Resolver 能正确识别参数名到 CLI 名的映射
- **验证**：parameters["max_model_len"] 映射为 "--max-model-len"

### 1.2 缺少参数的默认值
- **输入**：Deployment.parameters 不包含某个参数
- **预期**：该参数不出现在 args 中（使用 BackendVersion.default_args 的原始模板值）
- **验证**：args 中不包含该参数的 CLI flag

### 1.3 参数类型验证
- **输入**：参数定义为 integer，Deployment 中值为 "not-a-number"
- **预期**：Resolver 返回 error
- **验证**：len(errors) > 0

---

## 2. BackendRuntime 从模板创建测试

### 2.1 从模板创建
- **输入**：选择 BackendRuntimeTemplate "vllm-metax-docker"，POST /api/v1/backend-runtimes/from-template
- **预期**：创建的 BackendRuntime 从模板复制 vendor、image_name、docker_json、entrypoint_override 等字段
- **验证**：runtime.vendor == "metax" && runtime.docker_json 非空

### 2.2 创建后可编辑
- **输入**：PATCH /api/v1/backend-runtimes/{id} 修改 image_name
- **预期**：image_name 更新成功
- **验证**：GET /api/v1/backend-runtimes/{id} 返回新 image_name

### 2.3 模板不直接参与运行
- **输入**：尝试用 BackendRuntimeTemplate 生成 RunPlan
- **预期**：不可行——只有 BackendRuntime 能生成 RunPlan
- **验证**：Resolver 不接受 BackendRuntimeTemplate 作为输入

---

## 3. BackendRuntime image 解析测试

### 3.1 BackendRuntime.image_name 用于 RunPlan
- **输入**：BackendRuntime.image_name = "vllm/vllm-openai:v0.8.5"，无 NodeRuntimeOverride
- **预期**：plan.Image == "vllm/vllm-openai:v0.8.5"
- **验证**：检查 plan.Image

### 3.2 无 NodeRuntimeOverride 且 BackendRuntime 无 image 时回退到 BackendVersion
- **输入**：BackendRuntime.image_name = ""，BackendVersion.defaultImages = {"nvidia":"vllm/vllm-openai:v0.8.5"}
- **预期**：plan.Image == "vllm/vllm-openai:v0.8.5"
- **验证**：检查 plan.Image

### 3.3 所有来源都无 image 时 error
- **输入**：NodeRuntimeOverride 无，BackendRuntime.image_name=""，BackendVersion.defaultImages={}
- **预期**：error "no image available"
- **验证**：len(errors) > 0

---

## 4. NodeRuntimeOverride 覆盖测试

### 4.1 NodeRuntimeOverride.image_name 最高优先级
- **输入**：NodeRuntimeOverride.image_name = "0d307f1665d3"，BackendRuntime.image_name = "vllm/vllm-openai:v0.8.5"
- **预期**：plan.Image == "0d307f1665d3"
- **验证**：检查 plan.Image

### 4.2 NodeRuntimeOverride.modelRoot 覆盖模型路径
- **输入**：Artifact.path = "/data/models/Qwen35-9B"，NodeRuntimeOverride.model_root_host_path = "/data/part2/MX-C500/model"
- **预期**：plan.Mounts[0].HostPath 基于 model_root_host_path
- **验证**：mount 路径以 model_root_host_path 为前缀

### 4.3 NodeRuntimeOverride.env 合并
- **输入**：BackendRuntime.default_env = {"A":"1"}，NodeRuntimeOverride.env = {"B":"2"}
- **预期**：plan.Env["A"] == "1" && plan.Env["B"] == "2"
- **验证**：检查 env 合并结果

### 4.4 NodeRuntimeOverride.docker_override_json.devices 覆盖设备
- **输入**：BackendRuntime.docker_json.devices = [{"/dev/mxcd":"/dev/mxcd"}]，NodeRuntimeOverride.docker_override_json.devices = [{"/dev/dri":"/dev/dri"}]
- **预期**：plan.Devices 包含覆盖后的设备列表（统一使用 docker_override_json.devices，不再有独立的 device_override_json）
- **验证**：检查 plan.Devices

---

## 5. Image 优先级完整测试

| 优先级 | 来源 | 测试场景 |
|--------|------|----------|
| 1 (highest) | NodeRuntimeOverride.image_name | 有 NodeOverride → 使用它 |
| 2 | BackendRuntime.image_name | 无 NodeOverride → 使用 BackendRuntime |
| 3 | BackendVersion.defaultImages[vendor] | 前两者都无 → 使用推荐镜像 |
| 4 | error | 都没有 → error |

---

## 6. 模板替换测试（仅 `{{var}}`）

### 6.1 `{{model_container_path}}` 替换
- **输入**：model.path = "/data/models/Qwen35-9B"，有 mount /data/models:/models
- **预期**：args 中包含 /models/Qwen35-9B（容器内路径）
- **验证**：contains(plan.Args, "/models/Qwen35-9B")

### 6.2 `{{model_host_path}}` 替换
- **输入**：model.path = "/data/models/Qwen35-9B"
- **预期**：如果 args 模板中有 {{model_host_path}}，替换为 /data/models/Qwen35-9B
- **验证**：检查替换结果

### 6.3 `{{container_port}}` 替换
- **输入**：BackendVersion.default_container_port = 8000
- **预期**：args 中包含 --port 8000
- **验证**：contains(plan.Args, "--port", "8000")

### 6.4 `{{served_model_name}}` 替换
- **输入**：Deployment.parameters.served_model_name = "qwen35-9b"
- **预期**：args 中包含 --served-model-name qwen35-9b
- **验证**：contains(plan.Args, "--served-model-name", "qwen35-9b")

### 6.5 `{{assigned_gpu_indexes}}` 替换
- **输入**：assigned GPU indexes = [0, 1, 2, 3]
- **预期**：plan.Env["CUDA_VISIBLE_DEVICES"] == "0,1,2,3"
- **验证**：检查 env value

### 6.6 `{{node_ip}}` 替换
- **输入**：Node IP = "192.168.1.100"
- **预期**：如果 args 模板中有 {{node_ip}}，替换为 IP
- **验证**：检查替换结果

---

## 7. 未知变量返回 error 测试

### 7.1 未知变量
- **输入**：args 模板包含 `{{undefined_variable}}`
- **预期**：Resolver 返回 error（不是 warning，不保留原样）
- **验证**：len(errors) > 0 && strings.Contains(errors[0], "undefined variable")

### 7.2 `${VAR}` 语法不支持
- **输入**：args 模板包含 `${MAX_MODEL_LEN}`
- **预期**：Resolver 不识别此语法，将 `${MAX_MODEL_LEN}` 视为普通文本或报错
- **验证**：确认 `${VAR}` 语法不工作

### 7.3 所有已知变量都正确处理
- **输入**：使用所有 15 个支持的变量
- **预期**：所有变量都被正确替换
- **验证**：逐个检查替换结果

---

## 8. 单节点多 GPU RunPlan 测试

### 8.1 4 GPU 配置
- **输入**：node-01，GPU 0,1,2,3，tensor_parallel_size=4
- **预期**：
  - plan.Env["CUDA_VISIBLE_DEVICES"] == "0,1,2,3"
  - args 包含 --tensor-parallel-size 4
- **验证**：检查 plan

### 8.2 GPU 索引存在性
- **输入**：GPU 索引 [0, 1, 5]，但 node 只有 4 GPUs
- **预期**：DryRun 报错
- **验证**：DryRunResult.Errors 包含 GPU index out of range

---

## 9. 多副本文档预留测试

### 9.1 replicas = 1（当前实现）
- **输入**：replicas = 1
- **预期**：正常创建 1 个 instance
- **验证**：len(instances) == 1

### 9.2 replicas > 1（应拒绝）
- **输入**：replicas = 2
- **预期**：返回 unsupported 错误
- **验证**：error 包含 "replicas > 1 not supported"

---

## 10. 多节点分布式明确不支持测试

### 10.1 allowMultiNodeSingleReplica=true 应拒绝
- **输入**：placement.allowMultiNodeSingleReplica = true
- **预期**：返回 unsupported 错误
- **验证**：error 包含 "not supported"

### 10.2 多节点 placement 应拒绝
- **输入**：placement.strategy = "distributed"
- **预期**：返回 unsupported 错误
- **验证**：error 包含 "not supported"

---

## 11. Deployment 引用 backend_runtime_id 测试

### 11.1 正确引用
- **输入**：Deployment.backend_runtime_id 指向存在的 BackendRuntime
- **预期**：Resolver 正常解析
- **验证**：plan 非空

### 11.2 引用不存在的 Runtime
- **输入**：Deployment.backend_runtime_id = "nonexistent"
- **预期**：error
- **验证**：len(errors) > 0

### 11.3 不使用 backend_id + backend_version_id
- **输入**：尝试用旧字段创建 Deployment
- **预期**：API 不接受这些字段
- **验证**：API 返回 400

---

## 12. ResolvedRunPlan 独立落库测试

### 12.1 Start 时创建 RunPlan
- **输入**：POST /api/v1/model-deployments/{id}/start
- **预期**：resolved_run_plans 表中新增一行
- **验证**：SELECT * FROM resolved_run_plans WHERE deployment_id = ?

### 12.2 RunPlan 不可变
- **输入**：已存在的 RunPlan
- **预期**：PATCH API 返回 405 或 RunPlan 内容不变
- **验证**：plan_hash 不变

### 12.3 每次重启生成新 RunPlan
- **输入**：Stop → Start → Stop → Start
- **预期**：有 2 个 RunPlan 记录（start 1 和 start 2）
- **验证**：SELECT count(*) FROM resolved_run_plans WHERE deployment_id = ? → 2

### 12.4 Instance 指向当前 RunPlan
- **输入**：GET /api/v1/model-instances/{id}
- **预期**：返回 current_run_plan_id
- **验证**：instance.current_run_plan_id 对应最新的 RunPlan

---

## 13. Docker 启动测试

### 13.1 基本 Docker 启动
- **输入**：完整的 ResolvedRunPlan
- **环境**：需要 Docker
- **预期**：容器启动成功
- **验证**：docker ps 显示容器运行

### 13.2 环境变量注入
- **输入**：plan.Env = {"CUDA_VISIBLE_DEVICES": "0", "TEST_VAR": "test"}
- **预期**：容器内可看到环境变量
- **验证**：docker exec {name} env

---

## 14. Health Check 测试

### 14.1 BackendVersion 默认 Health Check
- **输入**：BackendVersion.health_check = {path:"/v1/models", expectedStatus:200}
- **预期**：启动后 checker 使用 /v1/models
- **验证**：curl http://localhost:{host_port}/v1/models → 200

### 14.2 BackendRuntime 覆盖 Health Check
- **输入**：BackendRuntime.health_check_override = {path:"/health", expectedStatus:200}
- **预期**：checker 使用 /health
- **验证**：curl http://localhost:{host_port}/health → 200

---

## 15. Web 操作验收测试

### 15.1 推理后端页面
- **操作**：访问 `/backends`
- **验证**：显示 vLLM, SGLang, llama.cpp 三个后端（只读）
- **验证**：点击后端可查看版本列表和参数定义

### 15.2 运行模板页面
- **操作**：访问 `/runtime-templates`
- **验证**：显示 5 个模板（只读）
- **操作**：点击"从模板创建 Runtime"
- **验证**：跳转到 Runtime 创建页，字段预填

### 15.3 运行配置页面
- **操作**：CRUD BackendRuntime
- **验证**：创建、编辑、删除功能正常

### 15.4 节点覆盖页面
- **操作**：为 node-01 的某个 Runtime 配置覆盖 image
- **验证**：覆盖配置成功

### 15.5 模型部署页面
- **操作**：创建部署 → 选择 artifact + runtime → 填写参数
- **操作**：Preview RunPlan → 查看 docker 预览
- **操作**：Start → 查看实例状态变化

### 15.6 i18n
- **操作**：切换 en-US ↔ zh-CN
- **验证**：新页面文本正确切换

---

## 16. 删除的测试

以下测试在 Phase 0.1 后不再需要：

- ~~`${VAR}` Shell 风格替换测试~~ — 不支持 `${VAR}`
- ~~混合模板语法测试~~ — 只支持 `{{var}}`
- ~~BackendVersion 直接带完整 Docker 运行配置的测试~~ — 拆分为 BackendRuntime
- ~~Backend vendor 匹配测试~~ — 改为 BackendRuntime.vendor 与 GPU vendor 匹配
- ~~Custom Backend 测试~~ — 第一版不实现
- ~~MindIE / VoxBox 测试~~ — 第一版不实现
- ~~多节点分布式测试~~ — 第一版不实现

---

## 17. Web 保存 Roundtrip 测试（新增 Phase 0.2）

### 17.1 BackendRuntime 创建后 GET 校验
- **操作**：POST /api/v1/backend-runtimes/from-template
- **验证**：返回 201，GET /api/v1/backend-runtimes/{id} 返回相同数据
- **验证**：刷新页面后数据仍存在

### 17.2 BackendRuntime PATCH 后 GET 校验
- **操作**：PATCH /api/v1/backend-runtimes/{id} 修改 image_name
- **验证**：GET 返回更新后的 image_name，未修改字段不变

### 17.3 NodeRuntimeOverride 保存 roundtrip
- **操作**：POST /api/v1/node-runtime-overrides，然后 GET
- **验证**：数据一致。PATCH 修改 image_name 后 GET 确认

### 17.4 ModelArtifact 保存 roundtrip
- **操作**：POST /api/v1/model-artifacts，GET，PATCH，GET
- **验证**：每个步骤数据一致

### 17.5 ModelDeployment 保存 roundtrip
- **操作**：POST /api/v1/model-deployments，GET
- **验证**：backend_runtime_id、placement_json、parameters_json 都正确持久化

### 17.6 Web 本地数组禁止
- **验证**：前端代码中保存成功后使用响应对象更新数据，不存在只往本地数组 push 一行的情况

### 17.7 保存失败显示后端错误
- **操作**：提交无效数据（如缺少 name）
- **验证**：前端显示后端返回的错误消息

---

## 18. Start 事务顺序测试（新增 Phase 0.2）

### 18.1 正常事务流程
- **操作**：POST /api/v1/model-deployments/{id}/start
- **验证**：
  1. model_instances 新行 current_run_plan_id 非空
  2. resolved_run_plans 新行 instance_id = 新 instance.id
  3. gpu_leases 新行存在
  4. agent_tasks 新行存在

### 18.2 事务回滚：Resolve 失败
- **操作**：Start 时 BackendRuntime.image_name 为空且 defaultImages 为空
- **验证**：model_instances 无残留行，resolved_run_plans 无残留行

### 18.3 每次 restart 创建新 RunPlan
- **操作**：Start → 记录 run_plan_id_1 → Stop → Start → 记录 run_plan_id_2
- **验证**：run_plan_id_1 != run_plan_id_2，两个 RunPlan 都在 resolved_run_plans 表中

### 18.4 旧 RunPlan 不被覆盖
- **操作**：三次 Start/Stop 循环
- **验证**：SELECT count(*) FROM resolved_run_plans WHERE deployment_id = ? → 3

---

## 19. args_override_json Append 测试（新增 Phase 0.2）

### 19.1 基本 Append
- **输入**：BackendVersion.default_args = ["--a", "1"]，BackendRuntime.args_override = ["--b", "2"]
- **预期**：最终 args = ["--a", "1", "--b", "2"]
- **验证**：args 按顺序拼接

### 19.2 不支持 Replace
- **输入**：args_override 尝试用 args_mode=replace
- **预期**：无效果或被拒绝
- **验证**：args_override 始终被 append

### 19.3 Deployment.parameters 追加
- **输入**：default_args = ["--a", "1"]，args_override = ["--b", "2"]，Deployment.parameters = {"c": "3"}
- **预期**：最终 args = ["--a", "1", "--b", "2", "--c", "3"]
- **验证**：全部按顺序 append

---

## 20. E2E API Roundtrip 测试（新增 Phase 0.2）

详见 `test/e2e/model-runtime-api-roundtrip.sh`：

1. 创建 BackendRuntime → GET 校验 → PATCH → GET 校验
2. 创建 NodeRuntimeOverride → GET 校验
3. 创建 ModelArtifact → GET 校验 → PATCH → GET 校验
4. 创建 ModelDeployment → GET 校验 → PATCH → GET 校验
5. Start → 验证 instance + run_plan + leases + agent_task
6. Stop → 验证旧 run_plan 保留
7. Start → 验证新 run_plan 创建
8. 所有 GET 验证刷新后数据存在

---

## 21. Backend / BackendVersion 新增字段测试（Phase 0.3）

### 21.1 Backend.default_version 测试
- **输入**：Backend.default_version = "0.8.5"
- **预期**：未指定版本时自动使用 0.8.5
- **验证**：BackendVersion resolution 返回版本 "0.8.5"

### 21.2 Backend.default_env_json 合并测试
- **输入**：Backend.default_env = {"GLOBAL": "1"}，BackendVersion.env = {"VERSION": "2"}，BackendRuntime.default_env = {"RUNTIME": "3"}
- **预期**：final_env = {"GLOBAL": "1", "VERSION": "2", "RUNTIME": "3"}
- **验证**：env 按优先级合并

### 21.3 Backend.common_parameters_json 测试
- **输入**：Backend.common_parameters = ["--tensor-parallel-size", "--max-model-len"]
- **预期**：Web UI 可据此显示常用参数提示
- **验证**：API 返回 common_parameters 字段

### 21.4 BackendVersion.is_default 测试
- **输入**：BackendVersion.is_default = true
- **预期**：未指定版本时优先匹配 is_default=true 的版本
- **验证**：Resolver 选择 is_default=true 的版本

### 21.5 BackendVersion.env_json 合并测试
- **输入**：Backend.default_env = {"A":"1"}，BackendVersion.env = {"A":"2"}
- **预期**：final_env["A"] = "2"（Version 覆盖 Backend）
- **验证**：env 合并优先级正确

### 21.6 BackendVersion.default_backend_params_json 进入 args 测试
- **输入**：BackendVersion.default_backend_params = ["--enforce-eager"]
- **预期**：final_args 包含 "--enforce-eager"
- **验证**：contains(plan.Args, "--enforce-eager")

### 21.7 parameter_defs_json 字段测试
- **输入**：parameter_defs 含 name, cli_name, type, default, required
- **验证**：
  - name: "max_model_len" 映射到 cli_name: "--max-model-len"
  - type: integer 在输入非数字时报错
  - default: 8192 在未提供值时生效
  - required: true 在未提供值时要求必填

---

## 22. 文档一致性检查（Phase 0.4）

### 22.1 default_images_json 归属检查
运行以下命令：
```
grep -n "default_images_json" docs/design/13-backend-runplan-runtime-design.md
```
人工确认 `default_images_json` 只出现于：
- BackendVersion 字段说明（§4.2）
- BackendVersion 边界说明（§3.5）
- `default_images_json 只是推荐值` 文档节（§4.3）
- `backend_versions` SQL（§16.2）
- 配置归属表 Version 列（§8.1）

不允许出现于：
- BackendRuntime 字段列表
- NodeRuntimeOverride 字段列表
- ModelDeployment 字段列表
- ModelInstance 字段列表
- ResolvedRunPlan 表字段列表

### 22.2 device_override_json 已删除检查
新设计正文中不应出现 `device_override_json`。设备覆盖统一使用 `docker_override_json.devices`。

### 22.3 cliName 已替换检查
新设计正文中不应出现 `cliName`，统一使用 `cli_name`。

### 22.4 env 合并规则完整检查
执行手册中 env 合并规则必须包含 6 层：Backend → BackendVersion → BackendRuntime → NodeRuntimeOverride → Deployment → GPU visible。

---

## 23. 权限与租户隔离测试（Phase 0.5）

### 23.1 未登录 → 401
- **操作**：无 Cookie/Token 访问任一新 API
- **预期**：返回 401
- **验证**：`curl -s -o /dev/null -w "%{http_code}" http://localhost:18080/api/v1/backend-runtimes` → 401

### 23.2 viewer 写操作 → 403
- **操作**：viewer 用户调用 POST/PATCH/DELETE
- **预期**：返回 403
- **验证**：POST backend-runtimes、PATCH backend-runtimes/{id}、DELETE backend-runtimes/{id} 均返回 403

### 23.3 viewer 调 start/stop → 403
- **操作**：viewer 用户调用 POST /start、POST /stop
- **预期**：返回 403

### 23.4 operator 创建本 tenant BackendRuntime → 201
- **操作**：operator 在本 tenant 下 POST /backend-runtimes/from-template
- **预期**：201，返回新 BackendRuntime，tenant_id = operator 所在 tenant

### 23.5 operator PATCH 本 tenant BackendRuntime → 200
- **操作**：operator PATCH 自己 tenant 的 BackendRuntime
- **预期**：200

### 23.6 operator 跨租户 PATCH → 403
- **操作**：operator 尝试 PATCH 其他 tenant 的 BackendRuntime
- **预期**：403

### 23.7 operator 使用其他 tenant ModelArtifact 创建 Deployment → 403
- **操作**：创建 Deployment 时 model_artifact_id 指向其他 tenant artifact
- **预期**：403

### 23.8 operator 使用其他 tenant BackendRuntime 创建 Deployment → 403
- **操作**：创建 Deployment 时 backend_runtime_id 指向其他 tenant runtime
- **预期**：403

### 23.9 operator 使用无权 node/GPU 创建 Deployment → 403
- **操作**：创建 Deployment 时 node_id 指向其他 tenant 节点
- **预期**：403

### 23.10 operator start 本 tenant Deployment → 200
- **操作**：operator start 本 tenant 的 Deployment
- **预期**：200，创建 instance/run_plan/lease/agent_task

### 23.11 operator start 其他 tenant Deployment → 403
- **操作**：operator start 其他 tenant 的 Deployment
- **预期**：403

### 23.12 viewer 查看其他 tenant RunPlan → 403
- **操作**：viewer GET /api/v1/run-plans/{id}（其他 tenant 的 RunPlan）
- **预期**：403

### 23.13 platform_admin 跨租户操作允许
- **操作**：platform_admin 查看/操作任意 tenant 的 Runtime/Deployment/RunPlan
- **预期**：200

### 23.14 RunPlan 敏感 env 脱敏
- **操作**：Deployment env_overrides 含 `HF_TOKEN=secret123`，Start 后 GET /api/v1/run-plans/{id}
- **预期**：`HF_TOKEN` 的值为 `****` 而非 `secret123`

### 23.15 docker_preview 敏感 env 脱敏
- **操作**：GET /api/v1/run-plans/{id} 返回 docker_preview
- **预期**：`-e HF_TOKEN=****` 而非 `-e HF_TOKEN=secret123`

### 23.16 Web 无权限按钮隐藏
- **操作**：viewer 登录 Web
- **验证**：创建/编辑/删除/Start/Stop 按钮隐藏或 disabled

### 23.17 Web 直接构造请求仍被 403
- **操作**：viewer 通过浏览器 devtools 直接 fetch POST backend-runtimes
- **预期**：后端返回 403
