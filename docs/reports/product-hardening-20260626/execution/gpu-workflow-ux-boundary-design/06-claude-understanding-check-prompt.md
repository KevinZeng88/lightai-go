# 06 — Claude Understanding Check Prompt

Use this prompt first. Claude must read the design package and explain its understanding before writing code.

```text
请先不要改功能代码。

请阅读以下文档包：

/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/00-index.md
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/01-product-boundaries-and-user-mental-model.md
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/02-current-ux-problems-and-root-causes.md
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/03-target-ux-design-by-page.md
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/04-implementation-plan.md
/home/kzeng/projects/ai-platform-study/lightai-go/docs/reports/product-hardening-20260626/execution/gpu-workflow-ux-boundary-design/05-validation-and-acceptance.md

阅读后请只输出理解确认报告，不要修改代码。

报告必须包含：

1. 你如何理解三条线：
   - 模型线 ModelArtifact / ModelLocation
   - 运行线 Backend / BackendVersion / BackendRuntime / NodeBackendRuntime
   - 部署线 ModelDeployment / ResolvedRunPlan / ModelInstance

2. 你如何理解“模型线和运行线都与节点相关，但不能合并成一条流程”。

3. 你认为当前 UI 最大的 10 个问题是什么。

4. 你准备如何改造以下页面：
   - 运行模板
   - 节点运行配置
   - 模型库
   - 模型部署
   - 模型实例

5. 你准备如何隐藏内部 ConfigSet key，包括：
   - launcher.command
   - launcher.args
   - launcher.*
   - runtime_env.*
   - {{MODEL_CONTAINER_PATH}}
   - {{MODEL_CONTAINER_DIR}}

6. 你准备如何把这些内部 key 映射成用户可理解的参数，例如：
   - 共享内存 shm_size
   - GPU 内存比例
   - 上下文长度
   - GPU 可见设备
   - 健康检查超时
   - vLLM / SGLang / llama.cpp 常用参数

7. 你准备如何修复向导状态 reset、配置名称、保存/检测错误处理。

8. 你准备如何统一模型库和节点运行配置里的节点选择体验，同时保持两条业务线独立。

9. 你认为需要修改哪些文件，按 workstream 列出。

10. 你准备新增或修改哪些测试。

11. 你预计的 commit 划分。

12. 你确认本轮不实现：
    - OpenAI Gateway
    - API Key
    - Usage Metering
    - Billing
    - 历史兼容 migration

最后输出：

UNDERSTANDING_READY_FOR_REVIEW

等待审核确认后再执行。
```

