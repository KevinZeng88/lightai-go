请在当前 main 分支修复 runtime 模板 Devices/Model mount 语义和模型部署回归问题，不新建分支。

先阅读：

docs/reports/phase-3/runtime-config-display-probe-fix/11-metax-device-mount-template-review-v2.md
docs/reports/phase-3/runtime-config-display-probe-fix/12-model-deployment-regression-review.md
docs/reports/phase-3/runtime-config-display-probe-fix/13-runtime-device-volume-design.md
docs/reports/phase-3/runtime-config-display-probe-fix/14-model-deployment-fix-design.md
docs/reports/phase-3/runtime-config-display-probe-fix/15-implementation-steps.md
docs/reports/phase-3/runtime-config-display-probe-fix/16-validation-and-tests.md

执行前先做 inventory：

```bash
git status --short
grep -R "optional_devices" -n internal web docs || true
grep -R "model_runtime.port\|model_runtime.host\|model_runtime.model" -n internal web || true
grep -R "unsupported runtime_type\|runtime_type" -n internal web || true
grep -R "source_metadata\|raw config\|config_json" -n web/src/pages web/src/components || true
```

目标一：Runtime Devices / Model mount / Additional volumes 语义

1. 只保留一个 Devices 配置项，语义是 Docker --device 设备透传列表。
2. 删除或隐藏 Optional devices 概念和普通页面展示。
3. NVIDIA runtime catalog：Devices 默认不启用，列表为空。
4. MetaX/沐曦 runtime catalog：Devices 默认启用，至少包含：
   - /dev/mxcd -> /dev/mxcd, permissions=rwm
   - /dev/dri -> /dev/dri, permissions=rwm
   - /dev/mem -> /dev/mem, permissions=rwm
5. MetaX/沐曦 catalog 还需要表达：
   - privileged=true
   - cap_add 包含 SYS_PTRACE
   - security_options 包含 seccomp=unconfined、apparmor=unconfined
   - network_mode=host
   - shm_size=100gb
   - ulimit memlock=-1
   - group_add 包含 video
6. Devices UI 字段使用：enabled、host_device_path、container_device_path、permissions。
7. Devices UI 不出现 readonly。readonly 只属于 Model mount / Additional volumes。
8. container_device_path 为空时默认等于 host_device_path。
9. permissions 为空时默认 rwm。
10. 设备路径存在性只用于诊断：缺失最多 warning，不得阻断 ready/preflight/deploy。
11. 只有设备配置结构无法转换为 Docker run spec 时才阻断。
12. Model mount 保持独立且默认 readonly=true。
13. Additional volumes 保持独立；/mnt:/mnt 这类目录挂载只能放 Additional volumes，不放 Model mount。
14. 不把 /bin/bash 作为生产服务默认入口。
15. 不引入 metax-docker 模式。

目标二：运行模板列表页操作入口

1. 运行模板列表页操作列提供：查看、编辑、复制为用户配置。
2. 点击行或点击查看：打开只读详情。
3. 点击编辑：直接打开编辑态。
4. 详情页可以保留编辑按钮，但编辑入口不能只存在于详情页。
5. 系统内置模板如果不允许直接编辑，编辑按钮按当前产品规则禁用或隐藏；复制为用户配置必须可用。
6. 用户配置可以直接编辑。
7. 不恢复“点击行即编辑”。

目标三：模型部署 wizard 状态 reset

1. 点击新建部署必须创建全新 draft，从第一步开始。
2. 保存成功后关闭并 reset draft。
3. 取消后关闭并 reset draft。
4. Drawer close / X / overlay close 后 reset draft。
5. 保存失败时可以保留当前 drawer 供用户修正；用户关闭或取消后，下次新建必须是干净状态。
6. reset 内容包括 currentStep、selected model、selected runtime/NBR、service config、config_overrides、preflight result、runplan preview、error/loading state。
7. 不复用上一次保存或取消时的嵌套 reactive state。

目标四：模型部署详情、编辑和 raw JSON 展示

1. 模型部署列表/详情必须有编辑入口。
2. 打开已有部署默认显示结构化详情，不以 Raw config JSON / Source metadata JSON 作为主要内容。
3. Raw config JSON、Source metadata JSON、Resolved RunPlan JSON 只能作为诊断区，默认收起。
4. 编辑部署使用结构化 ConfigEditView 或等价结构化表单，不要求用户编辑 raw JSON。
5. RunPlan 预览入口保留。

目标五：修复 runtime_type resolve error

1. 定位 `[resolve_error] unsupported runtime_type: (only docker is supported)` 的触发点。
2. 追踪 runtime_type：BackendRuntime catalog -> NodeBackendRuntime snapshot -> Deployment create snapshot -> Preflight/RunPlan -> Start。
3. Docker 部署必须从所选 NBR/runtime snapshot 解析出 runtime_type=docker。
4. deployment config_overrides 不能把 runtime_type 覆盖为空。
5. wizard draft 不能携带空 runtime_type 到新建请求。
6. preflight、RunPlan preview、start 使用一致的 resolver/source。
7. 不在本轮支持非 Docker runtime_type。

目标六：端口字段和部署参数分层

1. 用户可见 canonical 容器端口是 service.container_port。
2. model_runtime.port 不作为普通 required 部署覆盖字段展示。
3. 如果后端 CLI 需要 --port，从 service.container_port 派生。
4. network_mode=host 时，宿主机端口显示为“不适用 / host network 使用容器端口”，不要空白。
5. bridge/default network 时，宿主机端口显示 configured / auto / unconfigured 状态，不要静默空白。
6. 以下字段不进入普通部署覆盖表单：
   - model_runtime.model
   - model_runtime.host
   - model_runtime.port
   - model_runtime.download_dir
7. 常用稳定参数可继续显示，例如 gpu_memory_utilization、max_model_len、tensor_parallel_size、served_model_name。
8. cpu_offload_gb、kv_cache_dtype、max_num_batched_tokens、max_num_seqs、swap_space、safetensors_load_strategy 等放入 Advanced/Expert 默认收起区，或通过 Custom args / Extra args 表达。
9. 这些高级/专家参数不得默认 required。
10. Custom args / Extra args 必须进入 RunPlan preview。

测试要求：

请补充或更新测试，至少覆盖：

1. ConfigEditView 中 Devices 使用设备语义字段，不出现 readonly。
2. Optional devices 不再作为普通用户字段出现。
3. NVIDIA 模板 Devices 默认不启用或为空。
4. MetaX/沐曦模板 Devices 默认启用并包含 /dev/mxcd、/dev/dri、/dev/mem。
5. device path 缺失只产生 warning，不阻断 preflight/deploy。
6. Model mount 仍默认 readonly。
7. Additional volumes 与 Model mount 分离。
8. 运行模板列表页存在查看/编辑/复制为用户配置操作入口。
9. 点击查看进入只读详情；点击编辑进入编辑态。
10. 新建模型部署每次从干净第一步开始。
11. 保存成功、取消、drawer close 都 reset 部署 draft。
12. 已有部署详情默认结构化展示，raw JSON 默认收起。
13. 部署详情有编辑入口。
14. Docker 部署 RunPlan/preflight 解析 runtime_type=docker。
15. config_overrides 不能覆盖 runtime_type 为空。
16. service.container_port 是 canonical 字段。
17. model_runtime.port 不显示为 required empty。
18. host network 下宿主机端口显示不适用，不为空白。
19. 高级/专家参数不默认 required。

限制：

1. 不重做 runtime/config 架构。
2. 不引入 Optional devices。
3. 不引入 metax-docker 模式。
4. 不把 Model mount 改为默认读写。
5. 不要求真实沐曦硬件测试。
6. 不接入 Playwright。
7. 不做无关 UI 大重构。
8. 不处理 VERSION。

完成后运行：

```bash
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm test
cd web && npm run build
```

提交并推送，输出：

1. 根因
2. 修改文件
3. Runtime Devices / Model mount / Additional volumes 最终语义说明
4. MetaX/沐曦模板最终配置摘要
5. 运行模板列表操作入口说明
6. 模型部署 wizard/detail 行为说明
7. runtime_type 修复证据
8. 端口和参数分层修复说明
9. 测试结果
10. commit id
11. push 结果
12. git status --short
