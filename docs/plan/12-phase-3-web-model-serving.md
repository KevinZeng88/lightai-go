# Phase 3：Web 模型服务

> 依赖：Phase 2（Agent Docker Runtime + Start/Stop/Logs API）
> 周期：2-3 周

## 1. 目标

在 Web 上完整操作模型部署生命周期。不通过 API 或 curl 就能完成模型管理。

## 2. 范围

- 模型库页面（列表/新增/编辑/删除）
- 运行环境页面（列表/新增/编辑，Docker 类型完整表单）
- 启动模板页面（列表/新增/编辑/预览渲染）
- 模型部署页面（创建向导 + 列表 + 详情）
- 模型实例页面（列表/详情/日志/ResolvedRunSpec）
- GPU 页面增强（占用状态/模型实例列）
- 节点页面增强（运行实例数列）
- Dashboard 增强（GPU 占用摘要）
- i18n 完整中英文

## 3. 明确不做什么

- Gateway 页面
- API Key 管理页面
- 自动调度 UI
- 多副本管理
- 计费相关页面

## 4. 页面设计

### 4.1 模型库

路由：`/models/artifacts`

字段：名称、路径、格式、任务类型、架构、量化、默认上下文、预计显存、租户

操作：新增、编辑、删除、创建部署（跳转到部署创建向导）

### 4.2 运行环境

路由：`/runtime/environments`

字段：名称、Runtime 类型、Backend、GPU 厂商、OpenAI 兼容、默认端口

Docker 类型展开：Image、Devices、Privileged、IPC、UTS、Network、Shm Size、Group Add、Security Options、Ulimits

所有可选参数有 enabled 开关，未启用时表单灰显，不进入保存 payload。

### 4.3 启动模板

路由：`/runtime/templates`

字段：名称、Runtime 类型、Backend、GPU 厂商、必填变量、可选变量

操作：新增、编辑、删除、预览渲染（输入变量值，输出 ResolvedRunSpec + 等价命令）

### 4.4 模型部署（创建向导）

路由：`/deployments/create`

步骤：选择模型 → 选择运行环境 → 选择启动模板 → 选择节点 → 选择 GPU → 设置端口 → 填写参数 → Dry Run → 确认创建

Dry Run 结果显示：
- ✅ 校验通过：显示 ResolvedRunSpec 摘要 + 等价命令预览
- ❌ 校验失败：列出具体错误和警告

### 4.5 模型部署（列表 + 操作）

路由：`/deployments`

字段：名称、模型、环境、模板、期望状态、状态、节点、GPU、端口、Endpoint、租户

操作：启动、停止、删除、查看实例、查看日志、查看 ResolvedRunSpec、Dry Run

### 4.6 模型实例

路由：`/instances`（或 `/deployments/{id}/instances`）

字段：实例 ID、部署、节点、GPU、Runtime、状态、容器 ID、端口、Endpoint、启动时间、最近错误

操作：查看日志、查看事件、查看 ResolvedRunSpec

### 4.7 GPU 页面增强

在现有 GPUs 页面基础上增加：
- 占用状态列（空闲 / 已占用）
- 模型实例/部署名称列（可点击跳转到实例页面）
- 租户列

GPU 详情抽屉增加：
- 占用信息区（当前实例、部署、租户、启动时间、Endpoint）

### 4.8 节点页面增强

在现有 Nodes 页面基础上增加：
- 运行中实例数列

节点详情抽屉增加：
- 实例列表（实例 ID、模型、状态、端口）

### 4.9 Dashboard 增强

GPU 聚合卡片增加"已占用/空闲 GPU 数"。卡片布局不变。

## 5. 自动刷新

复用 `useAutoRefresh` composable（已实现），部署列表和实例列表每 5 秒自动刷新。

## 6. i18n

以下新增页面需完整中英文：

| 中文 | 英文 |
|------|------|
| 模型库 | Model Library |
| 运行环境 | Runtime Environments |
| 启动模板 | Run Templates |
| 模型部署 | Model Deployments |
| 模型实例 | Model Instances |
| 创建部署 | Create Deployment |
| Dry Run | Dry Run |
| 等价命令预览 | Equivalent Command Preview |
| 占用状态 | Occupancy |
| 空闲 | Idle |
| 已占用 | Occupied |

## 7. 测试要求

- 创建部署向导完整走通（选择→Dry Run→确认创建）
- 启动失败原因在 Web 上可见（last_error 显示）
- GPU 页面能显示某张 GPU 被哪个模型占用
- 无异常 GPU 时显示"暂无异常 GPU"
- 空状态正确显示（无模型/无环境/无模板/无部署/无实例）
- 中英文切换正确
- 自动刷新间隔正确（5s）

## 8. 验收标准

```text
Web 上可以完整走通：
选择模型 → 选择环境 → 选择模板 → 选择节点/GPU/端口 → Dry Run → 确认创建
→ 启动 → 查看日志 → 查看状态 → 停止

GPU 页面显示某张 GPU 被哪个模型占用，点击可跳转实例
节点页面显示运行中实例数
Dashboard GPU 占用摘要正确
启动失败原因在 Web 上清晰可见（last_error）
```

## 9. 风险点

- 创建向导的 7 个步骤如果交互不流畅，用户可能放弃。建议每步都能随时返回修改
- Dry Run 错误信息需要用户友好，不能直接展示原始 Go error
- GPU 占用关系查询需要高效（Phase 2 需确认 API 查询性能）
