# 08 - Execution Governance and Decisions

> Status: active governance document
> Date: 2026-06-25
> Authority: supersedes conflicting guidance in 00-07 for execution purposes

## 1. 总目标

在 LightAI Go 中建立干净、可追溯、可验证的 runtime 参数体系，覆盖 vLLM / SGLang / llama.cpp 三后端和 NVIDIA / MetaX / Huawei 三厂商，确保参数从 schema 定义到最终 Docker 容器启动的全链路正确性。

## 2. 当前边界

- 项目：lightai-go
- 分支：当前分支，不新建分支
- DB：可删除 lightai.db 重建，不做旧数据兼容
- 旧配置：不做兼容 fallback
- 无法验证的厂商：标记 template_only / requires_hardware_validation

## 3. 固定原则

以下原则在整个执行过程中不可违反：

### 3.1 命令模板原则

- `command_template` / `default_command` 是命令生成模板，不是用户编辑入口。
- `schema fields` 才是用户编辑参数入口。
- 用户不应在 raw textarea 中维护 required core args。

### 3.2 参数启用原则

- `required` 参数 locked-on，不能 disable。
- `optional` 参数使用 `enabled/value`。
- `enabled=false` 时 value 保留，但不进入 final config。
- `copy/clone` 必须保留 enabled/value。

### 3.3 参数优先级原则

- `Deployment override > NBR > BackendRuntime > BackendVersion default`。
- 修改下一层不得反向污染上一层。

### 3.4 Deployment override 限制

- `host` 不允许 Deployment override。
- `container_port` 默认不允许 Deployment override，除非 schema 明确 `allow_override=true`，且 resolver 同步 RunPlan、Docker port mapping、health URL。
- `host_port` 由 Deployment `service_json` 控制。
- 高风险 Docker 参数不允许 Deployment override。

### 3.5 Vendor 隔离原则

- NVIDIA 默认不得出现 `/dev/dri`、`/dev/mxcd`、`/dev/infiniband`。
- MetaX/Huawei 参数只能在对应 vendor 配置下出现。
- vendor-specific devices/env/security options 不得串台。

### 3.6 冲突检测原则

- `extra_args` 不能重复 `host`/`port`/`model`/`model-path`/`served-model-name`。
- 冲突检测渐进式推进，最终不得绕过 structured core args。

### 3.7 安全原则

- sensitive env / API key / token / password 必须脱敏。
- image default warning 和平台配置错误要区分。
- `source map` 当前不写 DB。

### 3.8 当前不做项

- 当前不新增独立 `VendorRuntimeProfile` 表。
- 当前不改 DB schema。
- help 文档最后独立阶段做（Phase 7），不阻塞 Phase 1-6。
- 不做旧 DB / 旧配置兼容。

## 4. 分阶段执行规则

- 每个 Phase 独立可提交。
- Phase 之间不交叉修改。
- 每个 Phase 通过验收后可自动进入下一 Phase，除非触发停止条件。
- 每个 Phase 必须有 evidence。

## 5. 每阶段提交规则

- 每个 Phase 完成后 commit，commit message 格式：`runtime-param: phase N - <summary>`
- 每个 Phase 完成后 push（如 remote 可达）。
- 不允许跨 Phase 合并 commit。
- 如 Phase 内发现问题需要回退，只回退当前 Phase 的修改。

## 6. 自动推进规则

- Phase N 验收通过 → 自动进入 Phase N+1。
- 如 Phase N 验收失败 → 停在当前 Phase，修复后重新验收。
- 如触发停止条件 → 暂停所有自动推进，等待人工确认。
- 每个 Phase 开始前必须确认前一个 Phase 的 evidence 完整。

## 7. 停止条件

以下情况触发停止（真实阻塞）：

- 当前 Phase 的可验证目标发生回归，且无法定位或无法修复。
- 当前 Phase 要求必须 PASS 的 E2E 失败，且不是外部硬件/镜像/环境不可控原因。
- RunPlan 与 Docker inspect 不一致。
- vendor 隔离验证失败。
- `npm run build` / `npm test` / `go test` / `go build` 持续失败。
- 出现敏感信息泄露风险。
- 需要 DB schema 破坏性变更（需人工确认）。
- 需要引入独立 VendorRuntimeProfile 实体（需人工确认）。
- 发现设计文档存在根本矛盾。

不作为停止条件（可记录为后续 Phase 输入）：

- Phase 0 中 E2E 有结果但 FAIL 原因已记录。
- MetaX/Huawei 无真实硬件导致无法验证。
- `template_only` / `requires_hardware_validation` 的厂商模板无法真实运行。
- 厂商镜像/参数资料不足，已明确标记未验证。
- evidence-only 阶段发现问题但已沉淀为后续 Phase 输入。

## 8. 验收门槛

每个 Phase 的验收标准见 `05-development-steps.md` 和 `06-acceptance-and-test-plan.md`。

基础门槛（每个 Phase 都必须满足）：

```bash
gofmt -w cmd internal
git diff --check
npm run build
npm test
go test ./internal/...
go build ./cmd/server/...
go build ./cmd/agent/...
```

如涉及脚本：

```bash
bash -n <changed-script>
```

## 9. Evidence 要求

- 每个 Phase 的 evidence 保存到 `docs/reports/phase-N/evidence/`。
- 文件命名：`{序号}-{描述}.json/txt/md`。
- 最终报告引用 evidence 使用相对路径。
- E2E 日志保存完整，不截断。

## 10. Commit / Push 规则

- 文档修改和代码修改分开 commit。
- 每个 Phase 的 commit message 包含 Phase 编号和摘要。
- push 前必须确认 build/test PASS。
- 不允许 force push。
- 不允许修改已 push 的 commit。
