# 00 - 当前问题与产品目标

## 1. 当前问题

本轮反馈集中在 Web AI / 模型运行管理流程，不是单点 bug。

### 1.1 模型实例详情可读性差

实例启动后，实例详情中存在大量英文、内部字段名、raw JSON、状态值直出等问题。客户无法直接理解实例状态、后端、镜像、端口、设备绑定、错误原因。

### 1.2 模型测试方式不准确

`Qwen3-0.6B-Instruct-2512` 属于 Instruct / Chat 类模型，但当前测试入口报告：

```text
测试失败
Completion 接口请求失败。
```

这说明测试入口可能默认只调用 `/v1/completions`，没有根据模型能力、模型名称、metadata、后端 endpoint 能力选择 `/v1/chat/completions`。

### 1.3 模型能力没有产品化展示与配置

模型导入/扫描时已有一部分自动发现能力，但页面没有清楚展示：

- 支持对话 Chat？
- 支持文本补全 Completion？
- 支持 Embedding？
- 支持 Rerank？
- 支持 Vision？
- 能力来源是什么？
- 是否可人工修正？

本轮不改数据结构；如果已有 capabilities / metadata / tags / parameters 等字段，优先基于现有字段展示和编辑。若当前没有可持久化字段/API，不得新增 schema，先展示自动推断结果，并记录后续数据模型需求。

### 1.4 NBR 页面把配置快照 JSON 作为主要入口

用户实际要修改的是节点运行配置中的运行参数，而不是直接编辑“配置快照 JSON”。

客户需要看到和修改：

- 镜像
- 启动命令
- args / extra args
- env
- volumes
- ports
- devices
- group_add
- privileged
- ipc
- shm_size
- ulimits
- health check

配置快照 JSON 只能作为高级诊断入口，不应作为主编辑入口。

### 1.5 部署时缺少 deployment-level overrides

NodeBackendRuntime 会提供默认模型卷/运行参数，但部署某个模型时还需要临时额外配置：

- 额外卷映射
- 额外环境变量
- 额外启动参数
- 端口覆盖
- served model name / endpoint alias
- 测试 profile

本轮不新增数据结构。若已有 Deployment payload/RunPlan 支持相关字段，则通过 UI 暴露；若没有，不得擅自改 schema，先作为后续事项记录。

### 1.6 停止实例后模型实例列表语义不清

模型部署运行后出现在模型实例。如果用户主动停止成功，实例主列表应默认不再显示 stopped 实例，避免客户误以为还有运行实例。

但 audit/log/operation 历史必须保留，用于诊断和审计。

### 1.7 模型部署页面信息不足

部署页面需要展示更多关键信息，例如：

- 模型
- 推理后端
- 后端版本
- 节点运行配置
- 镜像
- 节点
- GPU / accelerator
- endpoint
- 状态
- 最近错误

### 1.8 推理后端、运行模板过于暴露

Backend / BackendVersion / BackendRuntime 对普通用户而言属于系统能力和模板，通常不需要频繁修改。它们不应占据主流程，而应放到更隐藏的“配置/系统设置/高级配置”区域。

## 2. 产品目标

本轮目标不是增加新数据模型，而是调整 Web AI 的展现和操作方式，让用户按自然流程完成模型运行：

```text
添加模型
→ 确认模型能力
→ 配置节点运行参数
→ 创建部署
→ 预览 RunPlan
→ 启动实例
→ 测试模型
→ 查看日志与诊断
→ 停止/清理实例
```

核心目标：

1. 页面组织更接近用户任务，而不是数据库对象。
2. 主流程聚焦模型运行，不让用户先理解所有底层对象。
3. Backend / Runtime 模板放到配置/高级区域。
4. NBR 作为“节点运行参数”结构化编辑入口。
5. 模型能力自动发现，但允许基于现有字段进行人工配置。
6. 测试入口根据模型能力和后端能力自动选择 Chat / Completion / Embedding 等方式。
7. 所有详情页中文化、产品化，不直出内部英文和 raw JSON。
8. RunPlan 作为最终预览和诊断材料，默认摘要展示，高级区域只读展开。
9. 不修改数据结构，不新增迁移。
