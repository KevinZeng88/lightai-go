# 01 - Web AI 流程与导航设计

## 1. 总体意见

当前页面如果直接按 Backend、BackendRuntime、NodeBackendRuntime、ModelDeployment、ModelInstance 等内部对象展开，客户理解成本较高。

建议把 Web AI 主流程调整为任务式导航：

```text
模型运行
├── 模型库
├── 运行配置
├── 模型部署
├── 模型实例
├── 测试与诊断
└── 配置
    ├── 推理后端
    ├── 运行模板
    └── 高级参数/系统模板
```

其中：

- 普通用户主要使用：模型库、运行配置、模型部署、模型实例、测试与诊断。
- 管理员/实施人员使用：配置 → 推理后端、运行模板。
- Backend / BackendVersion / BackendRuntime 默认隐藏到配置区，不作为主流程高频入口。

## 2. 推荐主流程

### Step 1：添加模型

用户目标：告诉平台“有哪些模型、模型在哪里、模型大概能做什么”。

页面入口：`模型库 → 添加模型`

操作步骤：

1. 选择节点。
2. 浏览/输入模型路径。
3. 扫描模型 metadata。
4. 展示自动发现信息：
   - 模型名称
   - 格式
   - 架构
   - 参数规模
   - 量化
   - 上下文长度
   - 文件大小
   - 模型位置
   - 推断能力
5. 用户确认模型名称和能力。
6. 保存模型。

注意：

- 本轮不修改数据结构。
- 若已有 capabilities 字段，展示并允许编辑。
- 若没有可写 capabilities 字段，只展示推断结果，并在文档记录后续需要持久化能力配置。

### Step 2：确认模型能力

用户目标：确认测试和部署应该按什么方式处理模型。

能力标签建议：

```text
对话 Chat
文本补全 Completion
向量 Embedding
重排 Rerank
视觉 Vision
工具调用 Tool Calling
结构化输出 Structured Output
```

能力来源建议展示：

```text
自动发现：tokenizer_config.chat_template
自动推断：模型名称包含 Instruct
后端探测：/v1/models 返回
人工修正：管理员设置
```

置信度：

```text
高 / 中 / 低
```

本轮不要求新增复杂能力模型。优先使用已有 metadata/capabilities/tags；如果没有字段，前端可以展示“推断能力”，但不得伪造已持久化配置。

### Step 3：配置节点运行参数

页面入口：`运行配置`

对象：NodeBackendRuntime。

用户目标：配置某台节点上如何运行某类后端。

普通用户不应直接面对 BackendRuntime JSON 快照。页面应改为结构化参数：

- 镜像
- 命令
- args
- env
- volumes
- ports
- devices
- privileged
- ipc
- shm_size
- ulimits
- health check
- equivalent docker command 预览

Backend / BackendRuntime 模板选择可以保留，但放在高级区域：

```text
来源模板：vLLM NVIDIA Docker 模板
[查看模板详情]
[从模板重新生成]
```

### Step 4：创建模型部署

页面入口：`模型部署 → 新建部署`

用户目标：选择模型、运行配置、资源，并启动。

建议步骤式表单：

1. 选择模型
   - 展示模型能力
   - 展示模型位置
2. 选择运行配置
   - 后端
   - 后端版本
   - NBR
   - 镜像
   - 节点
   - 后端能力匹配检查
3. 选择资源
   - accelerator_ids
   - GPU 数量
   - 节点
4. 部署级覆盖
   - 额外卷
   - 额外 env
   - 额外 args
   - 端口覆盖
   - endpoint alias，如已有字段支持
5. RunPlan 预览
   - image
   - command
   - env
   - volumes
   - ports
   - devices
   - health check
   - equivalent docker command
6. 启动

本轮不新增 Deployment schema。若已有字段支持 overrides，就展示；否则在 UI 中只展示可用项，并记录缺口。

### Step 5：启动实例

启动后进入模型实例。

实例详情应该产品化展示：

- 基础信息
- 运行信息
- 资源信息
- 测试入口
- 日志
- 诊断信息

不要直接展示 raw JSON。RunPlan JSON 放到高级诊断只读区。

### Step 6：测试模型

测试入口建议出现在：

- 模型实例详情
- 模型部署详情
- 测试与诊断页面

测试步骤：

1. 检查实例状态。
2. 调用 `/v1/models` 确认模型 ID。
3. 根据模型能力选择默认测试：
   - Chat → `/v1/chat/completions`
   - Completion → `/v1/completions`
   - Embedding → `/v1/embeddings`
4. 用户可切换测试类型。
5. 展示 endpoint、request 摘要、response、latency、错误原因。

对于 `Qwen3-0.6B-Instruct-2512`，默认应优先 Chat Completion，而不是 Completion。

### Step 7：停止与清理

用户主动停止成功后：

- 停止容器。
- 释放资源。
- 模型实例主列表默认不显示 stopped 实例。
- 历史保留在 audit/log/operation。
- 可提供“显示已停止/失败实例”的筛选项。

## 3. 导航重组建议

### 主导航

```text
概览
节点
GPU
模型运行
  - 模型库
  - 运行配置
  - 模型部署
  - 模型实例
  - 测试与诊断
监控
审计日志
配置
  - 推理后端
  - 运行模板
  - 系统设置
```

### 为什么后端和运行模板应放到配置区

Backend / BackendVersion / BackendRuntime 是平台能力定义和模板，不是日常运行任务。客户日常关心的是：

- 我有哪些模型？
- 我要在哪台机器跑？
- 用哪个镜像/参数？
- 跑起来了吗？
- 能不能访问？
- 失败怎么查？

因此推理后端和运行模板应该作为“配置/高级配置”存在。默认模板只读，必要时 clone 成节点运行配置。

## 4. 页面角色建议

### 普通用户/租户用户

可见：

- 模型库
- 模型部署
- 模型实例
- 测试与诊断

有限可见：

- 运行配置，只能使用管理员提供的配置，或编辑允许字段。

不可见或只读：

- 推理后端
- 系统运行模板
- 高危 Docker 参数

### 平台管理员/实施人员

可见并可编辑：

- 运行配置
- 推理后端
- 运行模板
- 节点能力
- 高级诊断
- 参数来源和 RunPlan JSON

## 5. 本轮边界

本轮只做展现和入口调整：

- 不改数据库结构。
- 不新增迁移。
- 不改变核心 API 语义。
- 不把隐藏到配置区的页面删除，只调整导航位置和展示优先级。
- 如果当前字段/API 不支持某配置持久化，则记录为后续，不强行实现。
