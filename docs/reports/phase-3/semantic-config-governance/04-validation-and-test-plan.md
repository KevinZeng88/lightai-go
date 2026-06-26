# 04. Validation and Test Plan

## 1. 必跑命令

每个实施批次至少运行相关测试；最终必须运行：

```bash
go build ./cmd/server/...
go build ./cmd/agent/...
go test ./internal/server/...
go test ./internal/agent/...
cd web && npm run build
cd web && npm test
```

## 2. 后端单元测试

### 2.1 Semantic Registry

测试点：

1. canonical key 注册成功。
2. legacy key normalize：
   - `backend.common.host` -> `service.listen_host`
   - `launcher.listen_host` -> `service.listen_host`
   - `backend.common.port` -> `service.container_port`
   - `launcher.container_port` -> `service.container_port`
3. normalize 后无旧 key。
4. 同时存在多个值且冲突时产生 warning。
5. Backend CLI flag 不作为 semantic key。
6. unknown legacy key 进入 diagnostic 或明确错误。

### 2.2 Config Snapshot

测试点：

1. BackendRuntime 创建时复制 runtime 环境参数。
2. NBR 创建时复制 BackendRuntime 快照。
3. Deployment 创建时复制模型参数、服务暴露参数、NBR 运行参数。
4. 上游修改不影响已创建下游。
5. source_snapshot 记录来源。
6. dirty 标记正确。

### 2.3 Validation / Warning

硬错误测试：

- required missing。
- int 字段传字符串。
- port 非数字。
- enum 不在选项内。
- direct patch deprecated legacy key。

warning 测试：

- `model_runtime.max_model_len` 超过模型建议长度。
- `gpu_memory_utilization` 超过建议范围。
- 修改 advanced Docker 参数。
- health path 为空但服务需要健康检查。
- 估算显存可能不足。

要求：

- warning 不阻断保存。
- error 阻断保存。
- warnings 返回到 ConfigEditView。

### 2.4 Projector

测试点：

1. BackendRuntime 普通区仅显示 required/common/recommended。
2. advanced 字段进入高级折叠。
3. diagnostic 字段只读。
4. NBR 不显示 model_runtime 参数，除非其快照明确复制了该参数。
5. Deployment 显示模型运行参数副本。
6. Field label 输出支持中文（英文）。
7. ConfigEditPatch 只提交 dirty 或 changed 字段，避免无意义保存全部字段。

### 2.5 Resolver / RunPlan

测试点：

1. vLLM：
   - `service.listen_host` -> `--host`
   - `service.container_port` -> `--port`
   - `model_runtime.max_model_len` -> `--max-model-len`
2. SGLang：
   - context length 映射正确。
3. llama.cpp：
   - ctx size 映射正确。
4. Docker：
   - `docker.shm_size` -> `--shm-size`
   - devices/group_add/env 正确。
5. Health check：
   - port 默认引用 `service.container_port`。
   - path 来自 runtime health default 或 backend default。
6. RunPlan 不读取旧 alias key。

## 3. 前端测试

### 3.1 ConfigEditView

测试点：

1. required/common/recommended/advanced/diagnostic 分区。
2. advanced 默认折叠。
3. diagnostic 只读。
4. warning 显示 `!` 或 warning tag。
5. 中文界面显示 `中文（English）`。
6. changed-only patch 或 dirty patch 正确。
7. 空 advanced 字段默认 unchecked。
8. inherited default 不等于 checked。

### 3.2 运行模板页

测试点：

1. 不出现 `backend.common.host` / `launcher.listen_host`。
2. 只出现一个 `容器监听地址（Listen Host）`。
3. 只出现一个 `容器监听端口（Container Port）`。
4. 普通区字段明显减少。
5. devices/group_add/security_options 默认折叠、默认未启用。
6. 保存不报 `unknown config field`。

### 3.3 添加节点运行配置

测试点：

1. 能从节点镜像列表选择。
2. 能手工输入 image。
3. 不显示模型运行参数。
4. 能显示节点本地 Docker/env/health 参数。
5. 保存后 check-request 使用最终 image_ref。

### 3.4 部署页

测试点：

1. 模型运行参数在 Deployment 可见，但默认放高级。
2. 显示继承来源和 warning。
3. host_port 只在部署服务暴露区。
4. 不显示基础 runtime image 修改。
5. Preview RunPlan 使用 Deployment snapshot 值。

## 4. API-first E2E

至少新增脚本或测试覆盖：

1. 创建/复制运行模板。
2. 创建 NBR。
3. 修改 NBR 镜像。
4. 创建 Deployment。
5. 修改 Deployment 模型参数。
6. Preview RunPlan。
7. 验证 RunPlan 命令参数。
8. 验证 warnings 返回。

建议命名：

```text
scripts/e2e/e2e-semantic-config-snapshot.sh
scripts/e2e/e2e-semantic-config-runplan.sh
```

## 5. 手工验收

必须手工验证：

1. 运行模板用户配置详情：字段中文（英文）。
2. 普通区只显示必须/常用/建议字段。
3. 高级字段折叠，默认不启用。
4. 保存不再出现 unknown config field。
5. NBR 阶段不出现模型参数。
6. Deployment 阶段可以改模型参数，但默认不突出。
7. 修改 risky 参数出现 `!` warning，不阻断保存。
8. RunPlan preview 与页面配置一致。
9. 健康检查有默认值时显示，无默认值时不显示空表单。
10. Raw JSON 只在诊断区。

## 6. Closeout 要求

每轮 closeout 必须包含：

- commit id。
- push result。
- test summary。
- changed files。
- semantic key changes。
- removed legacy keys。
- remaining blockers。
- git status。

不允许出现：

```text
future follow-up
later cleanup
left for next phase
```

除非属于明确外部硬件/镜像验证 blocker，并已记录为 DOCUMENTED_BLOCKER。
