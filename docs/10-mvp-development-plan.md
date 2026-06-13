# LightAI Go MVP 开发计划

## 1. 开发原则

MVP 开发必须按阶段推进，每个阶段都要能运行、能验证、能提交。

Claude 只允许根据 docs 文档开发，不允许自行研究 GPUStack，不允许扩大范围，不允许提前实现 Token、计费、复杂网关等后续功能。

每个阶段完成后，需要执行：

```bash
go fmt ./...
go test ./...
go build ./cmd/server
go build ./cmd/agent
git diff --check
```

## 2. Phase 0：基础骨架

目标：

1. Server 可启动；
2. Agent 可启动；
3. 配置文件可加载；
4. 日志可输出；
5. 健康检查接口可访问。

任务：

1. 创建 `cmd/server/main.go`；
2. 创建 `cmd/agent/main.go`；
3. 创建配置加载模块；
4. 创建日志模块；
5. 创建版本模块；
6. Server 提供 `/healthz`；
7. Agent 提供本地启动日志；
8. 创建示例配置文件。

完成标准：

```bash
go run ./cmd/server
go run ./cmd/agent
curl http://127.0.0.1:8080/healthz
```

## 3. Phase 1：Agent 注册与心跳

目标：

1. Agent 能注册到 Server；
2. Server 能保存节点；
3. Agent 能周期性心跳；
4. Server 能判断节点在线 / 离线。

任务：

1. 定义 Node 数据结构；
2. 初始化 SQLite；
3. 实现 Node 表；
4. 实现 Agent 注册 API；
5. 实现 Agent 心跳 API；
6. 实现节点列表 API；
7. Agent 实现注册逻辑；
8. Agent 实现心跳循环；
9. Server 定期计算节点状态。

完成标准：

```text
启动 Server
启动 Agent
调用节点列表 API
可以看到 Agent 节点在线
停止 Agent 后一段时间节点变为离线
```

## 4. Phase 2：GPU 采集与资源上报

目标：

1. Agent 能发现 GPU；
2. Agent 能采集 GPU 指标；
3. Server 能保存 GPU 状态；
4. Web/API 能展示 GPU 信息。

任务：

1. 定义 GPUDevice；
2. 定义 GPUMetric；
3. 定义 GPUCollector 接口；
4. 实现 MockCollector；
5. 预留 NvidiaCollector；
6. 预留 MetaxCollector；
7. 实现资源上报 API；
8. Agent 周期性上报资源；
9. Server 保存最新 GPU 状态；
10. 提供 GPU 查询 API。

完成标准：

```text
即使没有真实 GPU，也可以通过 MockCollector 看到模拟 GPU
真实 GPU Collector 失败时 Agent 不崩溃
采集错误能在诊断信息中看到
```

## 5. Phase 3：运行环境管理

目标：

1. 可以创建 Docker 运行环境；
2. 可以编辑运行环境；
3. 可以预览 Docker 参数；
4. 未启用参数不出现在最终命令中。

任务：

1. 定义 RuntimeEnvironment；
2. 定义 DockerRunSpec；
3. 实现运行环境 CRUD；
4. 实现参数启用开关；
5. 实现 Docker 命令预览；
6. 实现配置校验。

完成标准：

```text
可以创建一个 vLLM Docker 运行环境
可以配置镜像、命令、环境变量、volume、device、shm-size
未启用参数不会出现在命令预览中
```

## 6. Phase 4：模型定义管理

目标：

1. 可以创建模型定义；
2. 可以关联默认运行环境；
3. 模型可以被实例引用。

任务：

1. 定义 Model；
2. 实现模型 CRUD；
3. 支持模型路径；
4. 支持默认端口；
5. 支持默认上下文长度；
6. 支持默认启动参数；
7. 防止删除已被实例引用的模型。

完成标准：

```text
可以创建 qwen、deepseek 等模型定义
模型定义可以选择默认运行环境
实例创建时可以选择模型
```

## 7. Phase 5：模型实例创建与任务下发

目标：

1. 可以创建模型实例；
2. Server 能生成任务；
3. Agent 能拉取任务；
4. Agent 能回报任务结果。

任务：

1. 定义 ModelInstance；
2. 定义 AgentTask；
3. 定义 TaskResult；
4. 实现实例创建 API；
5. 实现任务表；
6. 实现任务拉取 API；
7. 实现任务回报 API；
8. Agent 实现任务轮询；
9. Server 更新任务状态。

完成标准：

```text
创建实例后生成 start_instance 任务
Agent 可以拉取任务
Agent 可以回报成功或失败
任务状态可查询
```

## 8. Phase 6：Docker 启停实例

目标：

1. Agent 可以执行 docker run；
2. Agent 可以执行 docker stop；
3. Agent 可以回报容器 ID；
4. Server 可以更新实例状态。

任务：

1. 实现 Docker 命令生成；
2. 实现 docker run；
3. 实现 docker stop；
4. 实现 docker inspect；
5. 记录 Docker 命令快照；
6. 记录 stdout / stderr；
7. 回报容器 ID；
8. 更新实例状态。

完成标准：

```text
可以从平台启动一个容器
可以停止容器
启动失败时可以看到错误
实例状态能从 starting 进入 running 或 failed
```

## 9. Phase 7：实例健康检查与 endpoint

目标：

1. Agent 能检查容器状态；
2. Agent 能检查端口；
3. Server 能展示 endpoint；
4. 页面或 API 能看到实例健康状态。

任务：

1. Agent 定期 docker inspect；
2. Agent 检查端口可达；
3. Agent 上报实例状态；
4. Server 保存 endpoint；
5. Server 展示 last_error；
6. Server 展示 last_checked_at；
7. 提供实例详情 API。

完成标准：

```text
容器退出后实例状态变更
端口不可达时健康检查失败
实例详情中可看到 endpoint、状态、错误和最后检查时间
```

## 10. Phase 8：基础 Web 页面

目标：

1. 能通过 Web 查看核心资源；
2. 能完成基础实例启停操作。

页面：

1. Dashboard；
2. 节点列表；
3. 节点详情；
4. GPU 资源；
5. 运行环境；
6. 模型定义；
7. 模型实例；
8. 实例详情；
9. 任务记录。

完成标准：

```text
用户可以通过 Web 完成：
查看 GPU
创建运行环境
创建模型
创建实例
启动实例
停止实例
查看 endpoint
查看错误
```

## 11. 第二阶段入口

第一阶段稳定后，再进入：

1. API Key；
2. 统一模型访问入口；
3. OpenAI-compatible proxy；
4. Token 统计；
5. 额度；
6. 成本；
7. 简单调度。

