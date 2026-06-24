# 07 - Open Questions and Risks

> Status: active
> Date: 2026-06-25

## A. 已决策项

以下问题已有明确决策，不再需要讨论：

| # | 决策 | 来源 |
|---|------|------|
| A1 | 当前不新建 VendorRuntimeProfile 表 | 08-execution-governance §3.8 |
| A2 | source map 不写 DB | 08-execution-governance §3.7 |
| A3 | help 文档最后独立阶段做（Phase 7），不阻塞 Phase 1-6 | 08-execution-governance §3.8 |
| A4 | host 不允许 Deployment override | 08-execution-governance §3.4 |
| A5 | container_port 默认不允许 Deployment override | 08-execution-governance §3.4 |
| A6 | 高风险 Docker 参数不允许 Deployment override | 08-execution-governance §3.4 |
| A7 | extra_args 冲突检测渐进式推进，最终不得绕过 structured core args | 08-execution-governance §3.6 |
| A8 | 当前不改 DB schema | 08-execution-governance §3.8 |
| A9 | 可删除 lightai.db 重建验证环境，不做旧数据兼容 | 08-execution-governance §2 |
| A10 | MetaX/Huawei 无硬件验证时必须标记 template_only / requires_hardware_validation | 08-execution-governance §3.8 |
| A11 | 每个 Phase 通过后可以自动 commit/push 并进入下一 Phase，除非触发停止条件 | 08-execution-governance §6 |
| A12 | parameter_schema_json / parameter_values_json 继续使用，不改结构 | 04-mimo-review §3 |
| A13 | command_template 不是用户编辑入口，schema fields 才是 | 08-execution-governance §3.1 |

## B. 仍开放问题

以下问题需要在对应 Phase 开始前或执行中讨论确认：

### B1. served_model_name 的 slug 细节

**问题**：默认值的精确规则是什么？

**当前建议**：
1. 优先 deployment display name 的 slug
2. 其次 model display name / artifact name 的 slug
3. 再其次 model directory name 的 slug
4. 可加 `lightai-` 前缀

**影响 Phase**：Phase 1

**建议**：在 Phase 1 实现时，先用 artifact name 的 slug 作为默认值，后续可调整。

### B2. 参数分组具体命名

**问题**：group 字段的具体值是什么？

**当前建议**：
- `startup`：host, port, model_path, served_model_name
- `performance`：gpu_memory_utilization, max_model_len, max_num_seqs, etc.
- `capacity`：context_length, swap_space, cpu_offload, etc.
- `parallelism`：tensor_parallel_size, pipeline_parallel_size, etc.
- `docker`：privileged, ipc_mode, shm_size, devices, etc.
- `vendor`：vendor-specific devices/env
- `advanced`：extra_args, extra_env, extra_docker_options

**影响 Phase**：Phase 2

**建议**：Phase 1 不需要分组，Phase 2 实现时确定分组。

### B3. 每个 backend 的常用参数清单最终版

**问题**：vLLM 100+ 参数中哪些是"常用"的？

**当前建议**：以 `default_args_schema` 中已定义的参数为准，不扩展到 100+ 参数。长尾参数通过 `extra_args` 支持。

**影响 Phase**：Phase 6

**建议**：Phase 6 时根据实际使用反馈决定是否扩展。

### B4. Huawei/SGLang 具体镜像和参数来源

**问题**：Huawei SGLang 是否有公开可用镜像？

**当前建议**：暂不添加 SGLang MetaX/Huawei catalog entry，等待官方支持。

**影响 Phase**：Phase 6

**建议**：Phase 6 时标记 template_only。

### B5. 多实例部署命名

**问题**：同一模型多实例部署时，served_model_name 是否需要区分？

**当前建议**：served_model_name 默认基于 deployment name，天然区分。

**影响 Phase**：Phase 1

**建议**：Phase 1 实现时验证。

### B6. 参数版本升级策略

**问题**：BackendVersion 升级时，已有 NBR 的参数是否仍然有效？

**当前建议**：NBR 的 config_snapshot_json 是冻结的，不受 BackendVersion 升级影响。用户需要手动重新创建 NBR 来获取新版本的参数。

**影响 Phase**：Phase 6

**建议**：Phase 6 文档中说明。

### B7. 并发修改是否引入乐观锁

**问题**：多用户同时修改 NBR 参数时如何处理？

**当前建议**：当前不处理，last-write-wins。后续需要时引入乐观锁。

**影响 Phase**：当前无

**建议**：不在本次范围内。

### B8. container_port 哪些 backend/schema 允许显式 allow_override

**问题**：container_port 默认不允许 Deployment override，但某些场景可能需要。

**当前建议**：如果 schema 中参数标记 `allow_override=true`，则允许 Deployment override。当前所有 required 参数都不允许 override。

**影响 Phase**：Phase 1

**建议**：Phase 1 实现时，required 参数统一不允许 override。

### B9. 哪些性能参数默认启用

**问题**：gpu_memory_utilization 等性能参数是否默认启用？

**当前建议**：optional 参数默认 disabled，提供推荐值但不自动启用。用户需要显式启用。

**影响 Phase**：Phase 1

**建议**：Phase 1 实现时确认。

## C. 风险

| # | 风险 | 影响 | 缓解措施 |
|---|------|------|---------|
| C1 | required 参数 locked 实现复杂 | Phase 1 延迟 | 使用 el-switch disabled 或 el-checkbox disabled checked |
| C2 | Layer 3 模板替换可能引入副作用 | Phase 1 回归 | 充分测试现有 E2E |
| C3 | 参数分组可能不适合所有后端 | Phase 2 返工 | 使用后端特定的 group 值 |
| C4 | MetaX/Huawei 无法真实验证 | Phase 4 不完整 | 标记 template_only，不阻塞 |
| C5 | E2E 脚本维护成本 | Phase 5 返工 | 复用现有 helper |
| C6 | help 文档维护成本 | Phase 7 延迟 | Phase 7 独立做，不阻塞 |
