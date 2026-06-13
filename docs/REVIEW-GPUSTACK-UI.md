# GPUStack UI 信息架构审查

> 审查日期：2026-06-13
> 审查范围：`~/projects/ai-platform-study/gpustack-ui-reference`
> 审查目的：UI 设计参考，禁止复制代码，本轮不实现 Web

## 1. GPUStack UI 借鉴点

### 1.1 信息架构

GPUStack UI 使用 **UmiJS v4 + Ant Design ProComponents** 构建，采用侧边栏布局。

**页面层级**：

| 分组 | 页面 |
|------|------|
| Dashboard | 概览（集群/Worker/GPU/部署/副本计数、系统负载、用量、活跃部署） |
| Playground | Chat、Embedding、Rerank、Image、Audio |
| Model Service | Deployments、Catalog、My Models、Routes、Providers、Benchmarks、Backends、Model Files |
| GPU Service | GPU Instances、Templates、Storage、Storage Types、SSH Keys |
| Resources | Clusters、Workers、GPUs、Cloud Credentials |
| Usage & Billing | Usage Overview、Billing |
| Access Control | Organizations、Users、API Keys |

### 1.2 导航模式

- 左侧固定侧边栏（220px），可折叠
- 分组可折叠（点击分组标题）
- 图标使用 filled/outline 变体区分 active/inactive
- 右上角：帮助菜单 + 用户菜单（API Keys、Settings、Logout）
- 无顶部导航栏、无面包屑

### 1.3 Dashboard 布局

四段式垂直布局：
1. **统计卡片行**：5 个数字卡片（Clusters、Workers、GPUs、Deployments、Replicas）
2. **系统负载**：左 2/3 折线图（CPU/RAM/GPU/VRAM 趋势），右 1/3 仪表盘（当前利用率）
3. **用量**：日期范围选择器 + 按模型甜甜圈图 + 按用户条形图
4. **活跃部署表格**：名称、VRAM/RAM、副本数、Token 用量

### 1.4 节点/GPU 页面

- Resources 页面使用卡片式 Tab（Workers / GPUs / Model Files）
- FilterBar 组件：集群筛选 + 名称搜索 + 操作按钮
- 表格列：主机名、IP、状态、标签、CPU、RAM、GPU/VRAM、磁盘、集群
- 行操作：编辑标签、删除、详情、SSH Key、维护模式、Metrics/Grafana
- GPU 表格：Index、Worker、GPU Name、Vendor、VRAM（Total/Used/Allocated）、Utilization、Temperature

### 1.5 状态展示

- `StatusTag` 组件（非 Ant Design Tag）：
  - `success`（绿）、`transitioning`（黄/spin）、`warning`（橙）、`error`（红）、`inactive`（灰）
- 空状态：`NoResult` 组件（图标 + 标题 + 副标题 + CTA 按钮）
- 加载态：Ant Design `Spin`（size: middle）
- 错误态：`ErrorMessageContent` 组件（可复制错误文本）

### 1.6 操作模式

- 批量操作：`DropdownButtons`（选中行后显示，含计数徽章）
- 单行操作：`DropdownActions`（每行下拉菜单）
- 确认弹窗：`DeleteModal`（标准化删除确认）
- 创建表单：`FormDrawer`（右侧抽屉，600px 宽，不可点击遮罩关闭）

### 1.7 数据展示

- 表格：自定义 `SealTable`（可展开行、行内编辑）
- 详情：独立页面或 Modal
- 日志：`VirtualLogList`（虚拟滚动、分页加载、下载）
- 事件：Modal 内 Tab 切换（实例事件 / 卷事件）

---

## 2. LightAI Go Web 第一阶段页面建议

LightAI Go Phase 8 实现 Web。当前窗口（Phase 0-2B）不实现 Web，但基于 GPUStack UI 审查提出以下设计建议：

### 2.1 必需页面（最小可用集）

| 页面 | 优先级 | 说明 |
|------|--------|------|
| Login | P0 | 登录页（用户名+密码） |
| Dashboard | P0 | 概览：节点数、GPU 数、在线/离线 |
| Nodes | P0 | 节点列表：主机名、IP、状态、CPU、RAM、GPU 数、最后心跳 |
| Node Detail | P1 | 节点详情：系统信息、GPU 列表、诊断 |
| GPUs | P1 | GPU 列表：名称、厂商、显存、利用率、温度 |
| Users | P1 | 用户管理（平台管理员） |
| Tenants | P1 | 租户管理（平台管理员） |
| Roles | P1 | 角色管理（租户管理员） |

### 2.2 不需要的页面（第一阶段）

- Playground（Chat/Embedding 等）
- Model Service（Deployments/Catalog/Routes）
- GPU Service（Instances/Templates/Storage）
- Usage & Billing
- API Keys
- Cloud Credentials

---

## 3. 哪些布局适合 LightAI Go

### 3.1 推荐采用

1. **侧边栏布局**：简单、清晰、适合管理后台
2. **统计卡片行**：4-5 个数字卡片概览
3. **FilterBar + Table 模式**：筛选 + 搜索 + 表格 + 分页
4. **StatusTag 状态标签**：统一的颜色映射（绿/黄/橙/红/灰）
5. **NoResult 空状态**：图标 + 说明 + CTA
6. **FormDrawer 侧边抽屉**：创建/编辑表单
7. **DeleteModal 确认弹窗**：标准化删除确认
8. **CopyButton**：一键复制（端点、Token）
9. **AutoTooltip**：文本溢出自动 Tooltip
10. **分页 + 页面大小切换**：标准表格交互

### 3.2 推荐简化

1. **不需要分组折叠菜单**（第一阶段页面少）
2. **不需要多级筛选**（单条件筛选即可）
3. **不需要行内编辑**（用抽屉编辑）
4. **不需要虚拟滚动日志**（Phase 8 可能不需要日志页面）

---

## 4. 哪些交互过重，暂不采用

1. **Playground 交互**（Chat 界面、流式输出）：过重
2. **多步骤部署向导**（Source → Backend → Config → Schedule → Env）：过重
3. **Grafana 嵌入**（iframe / 外部链接）：Phase 9 再考虑
4. **实时轮询/Watch 模式**：第一版用手动刷新
5. **插件系统**（route extensions、access extensions）：过重
6. **多租户切换**（用户菜单切换租户）：第一阶段可以只显示当前租户
7. **批量操作**（多选删除/启动/停止）：第一版可以只做单行操作
8. **内嵌 Swagger/Redoc**：不需要，用外部工具

---

## 5. 后续 Phase 8 Web 页面建议

当 LightAI Go 发展到 Phase 8 时，建议的页面结构：

```
/login                          # 登录
/dashboard                      # 概览
/nodes                          # 节点列表
/nodes/:id                      # 节点详情
/gpus                           # GPU 列表
/gpus/:id                       # GPU 详情
/models                         # 模型列表（Phase 4+）
/models/:id                     # 模型详情
/instances                      # 实例列表（Phase 5+）
/instances/:id                  # 实例详情（含日志）
/runtimes                       # 运行环境（Phase 3+）
/runtimes/:id                   # 运行环境详情
/users                          # 用户管理（平台管理员）
/tenants                        # 租户管理（平台管理员）
/memberships                    # 成员管理（租户管理员）
/roles                          # 角色管理（租户管理员）
/settings                       # 个人设置
```

技术选型建议：
- React + TypeScript
- 轻量 UI 库（如 shadcn/ui 或 Ant Design 裁剪版）
- 不引入 UmiJS（过重）
- 不引入 ProComponents（过重）

---

## 6. 不得复制 GPUStack UI 代码的说明

**禁止行为**：
- 禁止复制 GPUStack UI 的 React 组件代码
- 禁止复制 GPUStack UI 的 CSS/Less 样式
- 禁止复制 GPUStack UI 的路由配置
- 禁止复制 GPUStack UI 的国际化文件
- 禁止复制 GPUStack UI 的自定义 Hook
- 禁止复制 GPUStack UI 的图标资源
- 禁止复制 GPUStack UI 的配置文件（proxy、plugins 等）

**允许行为**：
- 学习信息架构组织方式
- 学习状态展示的颜色语义
- 学习表格/表单/弹窗的交互模式
- 学习 Dashboard 的信息密度平衡
- 参考 API 路径设计（非 UI 代码）
