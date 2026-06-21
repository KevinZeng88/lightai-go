# 02 - 页面配置设计（不修改数据结构版）

## 1. 总原则

本轮只能修改：

- 页面布局
- 导航结构
- 字段展示名称
- 字段分组
- 表单组织
- i18n 文案
- 列表字段
- 详情页摘要
- 高级区域显示/隐藏
- 基于现有字段的编辑入口
- 基于现有 metadata 的推断和展示

本轮不能修改：

- 数据库 schema
- migration
- 后端核心数据结构
- 持久化字段语义
- API 契约中的必填字段
- Backend / Runtime / Deployment / Instance 的对象关系

如果某页面需要的字段当前不存在：

1. 不新增 schema。
2. 不伪造已持久化能力。
3. 在 review 文档中标注“需要后续数据模型支持”。
4. 前端可以显示“暂不可配置”或“仅自动推断”。

## 2. 模型库页面

### 2.1 列表字段

建议展示：

```text
模型名称
格式
架构
参数规模
量化
上下文长度
支持能力
位置数量
最近扫描时间
操作
```

如当前 API 没有某字段，则不强行补后端 schema。优先使用已有 metadata 字段；没有则显示 `—`。

### 2.2 模型详情

分区：

```text
基础信息
- 名称
- 格式
- 架构
- 参数规模
- 量化
- 上下文长度
- 文件大小

模型位置
- 节点
- 路径
- 是否存在
- 最近扫描时间

能力
- 自动发现能力
- 来源
- 置信度
- 人工修正状态，如已有字段支持

测试建议
- 推荐测试类型
- 推荐 endpoint
- 最近测试结果，如已有
```

### 2.3 能力配置

如果现有后端已有 `capabilities` 字段和更新 API：

- 页面提供 checkbox 编辑。
- 保存到现有字段。
- 不新增字段。

如果没有可写字段：

- 页面只展示自动推断能力。
- 显示提示：`当前版本仅展示自动推断能力，人工持久化配置需后续数据模型支持。`
- 在 review 文档记录后续需求。

### 2.4 能力推断 UI

即使不落库，也可以基于现有 metadata/name 展示推断结果：

```text
Qwen3-0.6B-Instruct-2512
推断能力：对话 Chat
原因：模型名称包含 Instruct，疑似指令/对话模型
置信度：中
```

若 tokenizer_config.chat_template 可读取：

```text
推断能力：对话 Chat
原因：tokenizer_config 包含 chat_template
置信度：高
```

## 3. 运行配置页面

对象：NodeBackendRuntime。

### 3.1 列表字段

```text
配置名称
节点
后端
后端版本
镜像
设备类型
主要端口
状态/可用性
来源模板
操作
```

### 3.2 编辑页分区

#### 基础信息

```text
配置名称
节点
后端
后端版本
来源模板
```

后端/模板一般只读或高级编辑，不作为普通修改重点。

#### 镜像与命令

```text
镜像 image
entrypoint
command
args / extra args
working dir
```

#### 环境变量

key/value 表格：

```text
名称
值
是否敏感
来源
操作
```

如无来源字段，则先不显示来源。

#### 卷映射

```text
宿主机路径
容器路径
只读
用途
操作
```

#### 端口

```text
宿主机端口
容器端口
协议
用途
```

#### 设备与权限

```text
devices
group_add
privileged
ipc
security_opt
shm_size
ulimits
```

高危项要有明显提示：

```text
privileged、host ipc、security_opt 会提高容器权限，仅管理员应修改。
```

#### 健康检查

```text
health path
interval
timeout
retries
startup grace period
```

#### 预览

```text
RunPlan dry-run
等价 docker 命令
参数摘要
```

#### 高级诊断

```text
配置快照 JSON
RunPlan JSON
```

只读显示，默认折叠。不要作为主编辑入口。

### 3.3 保存规则

- 只使用已有 API 和字段保存。
- 如果某字段当前 UI 可展示但后端无法保存，不提供编辑或标明只读。
- 保存后应可通过 dry-run / preflight 验证生效。

## 4. 模型部署页面

对象：ModelDeployment。

### 4.1 列表字段

```text
部署名称
模型
推理后端
后端版本
运行配置
镜像
节点
GPU/加速卡
状态
实例数
Endpoint
最近错误
创建时间
操作
```

如果后端 API 没有 join 字段：

- 前端可以基于已有详情调用或现有列表字段拼出部分信息。
- 不为展示方便新增 schema。
- 如需要后端优化，记录为后续 API summary improvement。

### 4.2 新建部署步骤

#### 第一步：选择模型

展示：

```text
模型名称
模型位置
格式
能力标签
推荐测试方式
```

#### 第二步：选择运行配置

展示：

```text
后端
后端版本
NBR 名称
镜像
节点
支持 API 类型
```

#### 第三步：资源选择

展示：

```text
节点
accelerator_ids
GPU 数量
设备绑定摘要
```

#### 第四步：部署级覆盖

本轮只展示现有字段支持的覆盖项。建议顺序：

```text
额外卷
额外 env
额外 args
端口覆盖
served model name / endpoint alias
```

若当前数据结构/API 不支持某项：

- 不新增字段。
- 在页面显示为后续能力，或先不展示。
- 写入 review 文档。

#### 第五步：RunPlan 预览

展示：

```text
镜像
命令
环境变量
卷
端口
设备绑定
健康检查
等价 docker 命令
```

高级展开：

```text
ResolvedRunPlan JSON
```

### 4.3 匹配检查

部署前应给出摘要：

```text
模型能力：Chat
后端能力：Chat Completions
匹配结果：通过
```

如果当前无法精确判断：

```text
匹配结果：无法确认，启动后将通过 /v1/models 和测试接口验证。
```

## 5. 模型实例页面

### 5.1 列表字段

```text
实例名称
模型
部署
后端
节点
状态
Endpoint
运行时长
最近错误
操作
```

默认不显示用户主动停止成功的 stopped 实例。

建议筛选：

```text
[ ] 显示已停止实例
[ ] 显示失败实例
```

如果当前后端 API 无法区分用户停止和异常退出：

- 默认过滤 stopped。
- failed/exited 保留。
- 记录后续需要更细事件语义。

### 5.2 详情页分区

#### 基础信息

```text
实例名称
状态
模型
部署
后端
后端版本
节点
创建时间
启动时间
```

#### 运行信息

```text
镜像
容器名
Endpoint
端口
健康检查
```

#### 资源信息

```text
accelerator_ids
设备绑定摘要
CPU/内存，如已有
```

#### 测试

```text
推荐测试类型
测试类型选择
Prompt / Messages
执行测试
结果
```

#### 日志和诊断

```text
最近错误
Docker logs
operation_id
等价 docker 命令
RunPlan JSON，高级折叠
```

### 5.3 i18n

所有状态和字段必须中文化：

```text
running → 运行中
starting → 启动中
failed → 失败
stopped → 已停止
missing_image → 镜像缺失
```

不得泄露：

```text
status.running
backend_name
device_binding
[object Object]
raw JSON 直接展示
```

## 6. 测试与诊断页面

### 6.1 测试入口

建议支持：

```text
Auto
Chat Completion
Text Completion
Embedding
Rerank
```

本轮至少修复 Chat / Completion。

### 6.2 自动选择规则

不改数据结构前，使用现有字段和前端推断：

```text
若已有 capabilities 包含 chat → 默认 Chat Completion
若模型名包含 Instruct/Chat 且无明确能力 → 默认 Chat Completion，标记为推断
若 capabilities 只有 completion → 默认 Completion
若未知 → 默认 Auto，先 /v1/models，再允许用户选择
```

### 6.3 错误提示

错误必须具体：

```text
Chat Completion 请求失败：后端返回 404。
Completion 请求失败：该模型可能更适合 Chat Completion，请切换测试类型。
模型未加载完成：/v1/models 未返回目标模型。
实例未运行：当前状态为 failed。
```

## 7. 配置区

### 7.1 推理后端

Backend / BackendVersion 放入：

```text
配置 → 推理后端
```

默认只读，展示：

```text
后端名称
版本
支持模型格式
支持 API 类型
设备支持
```

### 7.2 运行模板

BackendRuntime 放入：

```text
配置 → 运行模板
```

默认只读，展示模板摘要：

```text
镜像
端口
默认 env
默认 volumes
默认 devices
默认 health check
```

提供：

```text
复制为节点运行配置
查看模板 JSON，高级只读
```

### 7.3 为什么隐藏

这些对象是系统 catalog 和模板，不是客户日常主任务。隐藏到配置区可以降低使用门槛，同时保留实施人员调整能力。

## 8. 本轮不做事项

明确不做：

1. 不新增 ModelCapability 表。
2. 不新增 capabilities schema。
3. 不新增 Deployment overrides schema。
4. 不新增 API Key / Gateway 数据模型。
5. 不新增 BackendRuntime schema。
6. 不改运行核心逻辑，除非现有 UI 调用明显错误。
7. 不做真实 MetaX/NVIDIA 之外的新硬件能力。
8. 不承诺模型能力自动发现 100% 准确。
