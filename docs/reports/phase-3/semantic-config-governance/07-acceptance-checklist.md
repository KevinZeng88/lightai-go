# 07. Acceptance Checklist

## 1. 架构验收

- [ ] 存在统一 Semantic Config Registry。
- [ ] 存在 canonical semantic key 表。
- [ ] 每个参数有唯一 owner。
- [ ] Backend CLI flag 不作为用户配置 key。
- [ ] legacy duplicate keys 已 normalize 或删除。
- [ ] 所有页面复用统一 ConfigProjector。
- [ ] 所有保存复用统一 ConfigValidator。
- [ ] RunPlan 复用统一 resolver / adapter mapping。
- [ ] 不存在页面私有字段过滤规则作为主逻辑。

## 2. 重复建模清理

- [ ] `backend.common.host` 不作为长期 ConfigSet key。
- [ ] `launcher.listen_host` 不作为长期 ConfigSet key。
- [ ] `service.listen_host` 是唯一容器监听地址 key。
- [ ] `backend.common.port` 不作为长期 ConfigSet key。
- [ ] `launcher.container_port` 不作为长期 ConfigSet key。
- [ ] `service.container_port` 是唯一容器监听端口 key。
- [ ] `deployment.host_port` 只属于部署外部访问。
- [ ] `backend.arg.max_model_len` 不作为用户配置 key。
- [ ] `model_runtime.max_model_len` 是模型运行参数 key。

## 3. Copy-on-create 快照

- [ ] BackendRuntime 从 catalog/BackendVersion 复制运行模板快照。
- [ ] NBR 从 BackendRuntime 复制节点运行配置快照。
- [ ] Deployment 从 ModelArtifact/NBR/BackendVersion mapping 复制部署快照。
- [ ] 下游快照包含 source_snapshot。
- [ ] 下游修改后 dirty=true。
- [ ] 上游修改不影响已创建下游。
- [ ] RunPlan 使用当前对象快照，不 live 读取上游。

## 4. Warning / Validation

- [ ] 类型错误阻断保存。
- [ ] 必填缺失阻断保存。
- [ ] 格式非法阻断保存。
- [ ] 旧 alias key 直接 patch 被拒绝。
- [ ] 超建议值生成 warning，不阻断保存。
- [ ] 可能显存不足生成 warning，不阻断保存。
- [ ] 修改高级参数生成 warning 或 dirty 标记。
- [ ] UI 参数名前显示 `!` 或 warning tag。

## 5. UI 验收：运行模板

- [ ] 字段显示中文（English）。
- [ ] 普通区只显示必须/常用/建议字段。
- [ ] 高级参数默认折叠。
- [ ] 诊断参数只读并折叠。
- [ ] 不显示 Backend common host。
- [ ] 不显示 Listen Host 作为第二个重复字段。
- [ ] 只显示一个容器监听地址。
- [ ] 只显示一个容器监听端口。
- [ ] 空 devices/group_add/security_options 默认不启用。
- [ ] 保存不报 unknown config field。

## 6. UI 验收：节点运行配置

- [ ] 镜像可从节点已有镜像列表选择。
- [ ] 镜像可手动输入。
- [ ] 不显示模型运行参数。
- [ ] 显示节点本地运行环境参数。
- [ ] 高级 Docker/设备参数折叠。
- [ ] Check 使用最终 image_ref。

## 7. UI 验收：部署

- [ ] Deployment 显示模型运行参数副本。
- [ ] 模型运行参数默认在高级区。
- [ ] 可以修改 max_model_len。
- [ ] 超建议范围只 warning。
- [ ] host_port 只在服务暴露区。
- [ ] 不显示基础 runtime image 普通修改入口。
- [ ] Preview RunPlan 反映页面最终配置。

## 8. RunPlan 验收

- [ ] vLLM 使用 service.listen_host 生成 --host。
- [ ] vLLM 使用 service.container_port 生成 --port。
- [ ] vLLM 使用 model_runtime.max_model_len 生成 --max-model-len。
- [ ] SGLang 映射正确。
- [ ] llama.cpp 映射正确。
- [ ] Docker shm/ipc/devices/env 从 semantic keys 生成。
- [ ] Health check 默认端口引用 service.container_port。
- [ ] RunPlan preview 显示 warnings。

## 9. 测试验收

- [ ] go build ./cmd/server/... PASS。
- [ ] go build ./cmd/agent/... PASS。
- [ ] go test ./internal/server/... PASS。
- [ ] go test ./internal/agent/... PASS。
- [ ] cd web && npm run build PASS。
- [ ] cd web && npm test PASS。
- [ ] API-first E2E PASS。
- [ ] 手工页面验证 PASS。

## 10. Closeout 验收

- [ ] closeout 文档更新。
- [ ] 记录 commit id。
- [ ] 记录 push result。
- [ ] 记录测试结果。
- [ ] 记录 remaining blockers。
- [ ] git status clean。
- [ ] 无未记录 future/follow-up。
