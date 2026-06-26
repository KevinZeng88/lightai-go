# LightAI Go Runtime Template / BackendVersion / Copy-on-Create 落地设计与代码审查索引

生成日期：2026-06-26

## 背景

本组文档基于对 GitHub 仓库 `KevinZeng88/lightai-go` 关键路径的代码阅读，围绕以下目标形成：

1. 让 `Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / Deployment` 的边界清晰。
2. 支持“某个 BackendVersion 新增参数后，界面自动多出输入框”。
3. 严格实现逐层 `copy-on-create`，禁止运行时动态继承和查询时跨层 merge。
4. 清理 Runtime Template catalog，避免 `runtime.xxx`、重复模板、占位镜像、隐藏参考模板污染普通 UI。
5. 顺带对当前代码做一次结构性 review，输出可交给 Claude 执行的开发计划、验收标准和问题清单。

## 已阅读的关键代码路径

> 本文档不是全量静态扫描结果，而是基于 GitHub 连接器读取关键文件后的架构审查。Claude 在本机开发时必须按 `05-implementation-plan-and-acceptance.md` 中的命令补做全仓 grep、测试和 E2E 验证。

主要阅读文件：

- `README.md`
- `go.mod`
- `web/package.json`
- `web/src/router/index.ts`
- `web/src/pages/BackendsPage.vue`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/components/common/RuntimeParameterEditor.vue`
- `web/src/components/runtime/HumanRuntimeParameterForm.vue`
- `web/src/utils/runtimeParameterViewModel.ts`
- `web/src/components/deployments/NodeRuntimeConfigWizard.vue`
- `web/src/components/deployments/DeploymentWizard.vue`
- `web/src/components/deployments/DeploymentOverrideEditor.vue`
- `cmd/server/main.go`
- `internal/server/db/db.go`
- `internal/server/api/router.go`
- `internal/server/api/backend_handlers.go`
- `internal/server/api/runtime_handlers.go`
- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/server/api/configset_helpers.go`
- `internal/server/catalog/loader.go`
- `internal/server/catalog/types.go`
- `internal/server/runplan/resolver.go`

## 文档清单

1. `01-current-code-map-and-review.md`  
   当前代码结构、已有能力、主要问题、风险分级。

2. `02-schema-driven-parameter-ui-design.md`  
   参数 schema 驱动 UI 的落地设计，重点解决“新增参数，界面自动多出输入框”。

3. `03-copy-on-create-data-model-and-api-design.md`  
   严格逐层复制模型：Backend → BackendVersion → BackendRuntime → NodeBackendRuntime → Deployment。

4. `04-runtime-template-catalog-cleanup.md`  
   Runtime Template catalog 清理、国产 GPU 模板管理、visible/hidden/experimental 策略。

5. `05-implementation-plan-and-acceptance.md`  
   Claude 执行步骤、验收标准、测试命令、API-first E2E 用例。

6. `06-claude-development-prompt.md`  
   可直接复制给 Claude 的开发指令。

## 总体结论

当前代码已经具备一部分关键基础：

- 数据库各层已经都有 `config_set_json`。
- BackendVersion API 已经存在 create/list/patch/clone/delete。
- BackendRuntime 创建已经从 BackendVersion 复制 ConfigSet。
- NodeBackendRuntime 和 Deployment 已经在保存配置快照。
- `RuntimeParameterEditor.vue` 已经能按 `config_set.items` 动态渲染参数。
- RunPlan 的参数解析已经基本倾向使用 NBR snapshot。

但也存在几个会阻断最终设计的问题：

- BackendVersion 没有真正可用的前端管理入口。
- 简化参数表单 `HumanRuntimeParameterForm` 仍然是前端硬编码字段。
- catalog 中参数 label/group/order/constraints/required 与前端渲染字段不完全对齐。
- Deployment 创建仍然存在 NBR 缺失时 fallback 到 BackendRuntime snapshot 的逻辑。
- RunPlan image 解析仍然有 BackendRuntime/BackendVersion fallback。
- BackendVersion API 允许写入 image/entrypoint/command/model_mount，边界不干净。
- Runtime catalog 缺少 visibility/filter 约束，逻辑重复模板可能继续出现。
