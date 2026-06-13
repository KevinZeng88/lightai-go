# LightAI Go 资源监控设计

## 1. 设计目标

资源监控是 LightAI Go 第一阶段最核心的能力之一。

第一阶段必须做到：

1. Agent 可以采集本机操作系统资源；
2. Agent 可以采集 CPU、内存、Swap、磁盘、网络基础信息；
3. Agent 可以发现 GPU；
4. Agent 可以采集 GPU 指标；
5. Server 可以保存节点和 GPU 最新状态；
6. Web/API 可以展示节点、OS 和 GPU 状态；
7. 资源采集失败时有诊断信息；
8. 单个 Collector 失败不能导致 Agent 崩溃；
9. 没有真实 GPU 时也能用 MockCollector 验证流程；
10. Agent 和 Server 都可以暴露 Prometheus `/metrics`；
11. Server 可以提供 `/metrics/targets` 供 Prometheus 动态发现 Agent；
12. 后续可以接入平台托管的 Prometheus + Grafana。

第一阶段资源监控主要服务于：

* 客户现场验收；
* GPU 资源可视化；
* 操作系统资源可视化；
* 模型实例创建时选择节点和 GPU；
* 后续调度能力；
* 现场排障；
* Grafana 监控看板。

---

## 2. 借鉴 GPUStack 的监控思路

LightAI Go 学习 GPUStack 的方向是：

```text
Agent 采集
Server 管理
Prometheus 存储时序指标
Grafana 展示
```

不直接复制 GPUStack 的实现。

GPUStack 的做法可以抽象为四层：

```text
采集层：
  Worker / Agent 采集本机 OS、GPU、模型运行时指标

状态层：
  Worker / Agent 把当前状态上报 Server

指标层：
  Worker / Agent 暴露 /metrics
  Server 暴露 /metrics
  Server 提供 metrics targets

看板层：
  Prometheus 抓取指标
  Grafana 展示 dashboard
```

LightAI Go 也应保持这个边界：

```text
SQLite 保存当前管理状态
Prometheus 保存历史时序指标
Grafana 展示趋势图
```

Prometheus 不作为业务状态来源。
Server 不通过 Prometheus 查询结果判断节点是否在线或实例是否运行。
节点和实例状态仍然来自 Agent 上报。

---

## 3. 监控数据边界

LightAI Go 需要区分两类数据：

```text
平台管理数据
时序监控指标
```

### 3.1 平台管理数据

平台管理数据保存到 Server / SQLite：

1. Node 最新状态；
2. Node 最近心跳；
3. Node 最近资源上报时间；
4. GPU 最新状态；
5. GPU 最新指标；
6. 实例当前状态；
7. 任务记录；
8. 运行环境配置；
9. 模型定义；
10. 最近错误；
11. Docker 命令快照；
12. Collector 诊断结果。

### 3.2 时序监控指标

时序监控指标交给 Prometheus：

1. CPU 使用率趋势；
2. 内存使用趋势；
3. Swap 使用趋势；
4. 磁盘使用趋势；
5. GPU 显存使用趋势；
6. GPU 利用率趋势；
7. GPU 温度趋势；
8. GPU 功耗趋势；
9. Agent 心跳趋势；
10. 实例运行状态趋势；
11. 实例健康状态趋势；
12. 任务成功 / 失败计数；
13. Server API 请求量和耗时。

重要原则：

```text
LightAI Go 负责管理
Prometheus 负责时序采集和存储
Grafana 负责展示和告警
```

---

## 4. Collector 总体设计

Agent 内部采用 Collector 分层。

建议 Collector 类型：

```text
SystemCollector
GPUCollector
DockerCollector
RuntimeCollector
```

第一阶段实现：

```text
SystemCollector
GPUCollector
MockGPUCollector
```

第二阶段补充：

```text
DockerCollector
RuntimeCollector
```

### 4.1 Collector 统一原则

1. Collector 只负责本机采集；
2. Collector 不直接写数据库；
3. Collector 输出统一结构；
4. Collector 失败必须返回诊断信息；
5. 一个 Collector 失败不能影响其他 Collector；
6. Agent 汇总采集结果后统一上报 Server；
7. Agent 同时把最新采集结果刷新到 metrics exporter。

---

## 5. SystemCollector 设计

SystemCollector 负责采集操作系统和主机资源。

### 5.1 推荐实现方式

Go 实现建议优先使用：

```text
github.com/shirou/gopsutil/v4
```

原因：

1. Go 原生集成简单；
2. 不强依赖外部命令；
3. 可采集 CPU、内存、磁盘、网络、主机信息；
4. 适合 Linux / WSL2 / Windows / macOS；
5. 比解析命令行输出更稳定。

也可以预留：

```text
FastfetchCollector
MockSystemCollector
```

### 5.2 SystemCollector 接口

```go
type SystemCollector interface {
    Name() string
    Collect(ctx context.Context) (SystemSnapshot, CollectorDiagnosis)
}
```

### 5.3 SystemSnapshot

```go
type SystemSnapshot struct {
    Hostname             string
    PrimaryIP            string
    OS                   string
    Platform             string
    PlatformVersion      string
    KernelVersion        string
    Arch                 string
    UptimeSeconds        uint64
    BootTime             time.Time

    CPUModel             string
    CPUCores             int
    CPUUsagePercent      float64

    MemoryTotalBytes     uint64
    MemoryUsedBytes      uint64
    MemoryAvailableBytes uint64
    MemoryUsagePercent   float64

    SwapTotalBytes       uint64
    SwapUsedBytes        uint64
    SwapUsagePercent     float64

    Filesystems          []FilesystemSnapshot
    NetworkInterfaces    []NetworkInterfaceSnapshot

    CollectedAt          time.Time
}
```

### 5.4 FilesystemSnapshot

```go
type FilesystemSnapshot struct {
    MountPoint            string
    Device                string
    FSType                string
    TotalBytes            uint64
    UsedBytes             uint64
    AvailableBytes        uint64
    UsagePercent          float64
}
```

### 5.5 NetworkInterfaceSnapshot

```go
type NetworkInterfaceSnapshot struct {
    Name          string
    Addresses     []string
    IsUp          bool
    BytesSent     uint64
    BytesRecv     uint64
    PacketsSent   uint64
    PacketsRecv   uint64
}
```

### 5.6 第一阶段采集范围

第一阶段至少采集：

```text
hostname
primary_ip
os
platform
platform_version
kernel_version
arch
uptime_seconds
cpu_model
cpu_cores
cpu_usage_percent
memory_total_bytes
memory_used_bytes
memory_available_bytes
memory_usage_percent
swap_total_bytes
swap_used_bytes
swap_usage_percent
filesystem usage
network interface addresses
```

---

## 6. GPUCollector 设计

GPUCollector 负责采集 GPU 设备和 GPU 指标。

### 6.1 GPUCollector 接口

```go
type GPUCollector interface {
    Name() string
    Vendor() string
    Discover(ctx context.Context) ([]GPUDeviceInfo, error)
    Metrics(ctx context.Context) ([]GPUMetricInfo, error)
    Diagnose(ctx context.Context) CollectorDiagnosis
}
```

### 6.2 GPUDeviceInfo

```go
type GPUDeviceInfo struct {
    Vendor         string
    Index          int
    Name           string
    UUID           string
    PCIBusID       string
    DriverVersion  string
    RuntimeVersion string
    MemoryTotalMB  int64
    Status         string
}
```

### 6.3 GPUMetricInfo

```go
type GPUMetricInfo struct {
    Vendor          string
    Index           int
    UUID            string
    PCIBusID         string
    MemoryUsedMB     int64
    MemoryFreeMB     int64
    UtilizationGPU   float64
    UtilizationMem   float64
    TemperatureC     float64
    PowerW           float64
    Health           string
    CollectedAt      time.Time
}
```

### 6.4 CollectorDiagnosis

```go
type CollectorDiagnosis struct {
    Name        string
    Type        string
    Vendor      string
    Available   bool
    ToolPath    string
    Error       string
    CheckedAt   time.Time
}
```

---

## 7. GPU 设备数据模型

Server 侧 GPUDevice 表示一张 GPU 卡的静态或半静态信息。

```go
type GPUDevice struct {
    ID              string
    NodeID          string
    Vendor          string
    Index           int
    Name            string
    UUID            string
    PCIBusID        string
    DriverVersion   string
    RuntimeVersion  string
    MemoryTotalMB   int64
    Status          string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

GPU ID 生成规则：

优先：

```text
node_id + vendor + uuid
```

其次：

```text
node_id + vendor + pci_bus_id
```

最后：

```text
node_id + vendor + index
```

---

## 8. GPU 指标数据模型

Server 侧 GPUMetric 表示 GPU 最新指标。

```go
type GPUMetric struct {
    GPUDeviceID      string
    NodeID           string
    MemoryUsedMB     int64
    MemoryFreeMB     int64
    UtilizationGPU   float64
    UtilizationMem   float64
    TemperatureC     float64
    PowerW           float64
    Health           string
    CollectedAt      time.Time
}
```

第一阶段 Server 只保存最新指标。
长期历史趋势交给 Prometheus。

---

## 9. Collector Registry

Agent 应有 CollectorRegistry。

职责：

1. 注册多个 Collector；
2. 按配置决定启用哪些 Collector；
3. 执行 SystemCollector；
4. 执行 GPUCollector；
5. 聚合诊断结果；
6. 某个 Collector 失败时继续执行其他 Collector；
7. 将最新采集结果同步给 Agent metrics exporter；
8. 将采集结果上报 Server。

伪流程：

```text
load collectors
  ↓
collect system snapshot
  ↓
for each enabled gpu collector:
    diagnose
    if available:
        discover
        metrics
    else:
        record diagnosis
  ↓
merge results
  ↓
update metrics exporter
  ↓
report to server
```

---

## 10. MockCollector

MockCollector 用于开发和无 GPU 环境测试。

MockCollector 应返回固定模拟数据，例如：

```text
1 张 NVIDIA 模拟 GPU
显存 24576 MB
已用 4096 MB
利用率 12%
温度 45℃
功耗 80W
状态 healthy
```

MockCollector 价值：

1. WSL2 开发环境可验证完整链路；
2. 不依赖真实 GPU；
3. 可用于自动化测试；
4. 可用于前端页面调试；
5. 可用于 Prometheus / Grafana Dashboard 开发。

MockCollector 必须可配置关闭。

---

## 11. NvidiaCollector 预留

后续可以通过 `nvidia-smi` 采集。

建议命令：

```bash
nvidia-smi --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw --format=csv,noheader,nounits
```

注意事项：

1. 命令不存在时返回诊断错误；
2. 命令超时时返回诊断错误；
3. 单行解析失败不能导致全部失败；
4. 字段为空时使用 unknown；
5. power.draw 可能为空或 N/A。

---

## 12. MetaxCollector 预留

沐曦 GPU 后续通过厂商工具采集，例如 `mx-smi` 或实际现场工具。

配置示例：

```yaml
gpu:
  collectors:
    metax:
      enabled: true
      tool_path: "/usr/bin/mx-smi"
      timeout_seconds: 3
```

设计要求：

1. 不将沐曦采集逻辑写死在业务层；
2. 所有厂商差异放在 Collector 内部；
3. 采集结果统一转成 GPUDeviceInfo / GPUMetricInfo；
4. 采集失败时上报 CollectorDiagnosis；
5. 现场工具路径必须可配置。

---

## 13. 其他 GPU 厂商扩展

后续预留：

```text
AscendCollector
CambriconCollector
HygonDCUCollector
```

统一原则：

1. 每个厂商一个 Collector；
2. 不同厂商命令行输出在 Collector 内部解析；
3. 对 Server 暴露统一数据结构；
4. 对 Web 暴露统一字段；
5. 不在 Server 里写厂商判断逻辑。

---

## 14. DockerCollector 预留

DockerCollector 第二阶段实现。

采集内容：

1. Docker daemon 是否可用；
2. Docker 版本；
3. 容器数量；
4. LightAI 管理的实例容器状态；
5. 容器 CPU / 内存使用；
6. 容器 restart count；
7. 容器端口映射；
8. 容器健康检查状态。

第一阶段只预留字段，不强制实现。

---

## 15. RuntimeCollector 预留

RuntimeCollector 用于后续采集 vLLM、SGLang、Ollama、Xinference 等模型服务运行时指标。

采集内容：

1. 正在运行请求数；
2. 等待请求数；
3. KV cache 使用率；
4. TTFT；
5. TPOT；
6. 请求延迟；
7. 请求成功 / 失败；
8. prompt tokens；
9. generation tokens。

第一阶段暂不实现，但模型实例对象应预留 runtime metrics endpoint 字段。

---

## 16. 资源上报 API

接口：

```http
POST /api/agent/resources/report
```

请求：

```json
{
  "agent_id": "node-001",
  "reported_at": "2026-06-13T10:00:00Z",
  "host": {
    "hostname": "gpu-server-001",
    "primary_ip": "192.168.1.10",
    "os": "linux",
    "platform": "ubuntu",
    "platform_version": "24.04",
    "kernel_version": "6.8.0",
    "arch": "amd64",
    "uptime_seconds": 3600,
    "cpu_model": "Intel Xeon",
    "cpu_cores": 32,
    "cpu_usage_percent": 18.5,
    "memory_total_bytes": 274877906944,
    "memory_used_bytes": 68719476736,
    "memory_available_bytes": 206158430208,
    "memory_usage_percent": 25.0,
    "swap_total_bytes": 8589934592,
    "swap_used_bytes": 0,
    "swap_usage_percent": 0,
    "agent_metrics_url": "http://192.168.1.10:18080/metrics"
  },
  "filesystems": [
    {
      "mount_point": "/",
      "device": "/dev/sda1",
      "fs_type": "ext4",
      "total_bytes": 107374182400,
      "used_bytes": 53687091200,
      "available_bytes": 53687091200,
      "usage_percent": 50.0
    }
  ],
  "gpu_devices": [
    {
      "vendor": "nvidia",
      "index": 0,
      "name": "NVIDIA L20",
      "uuid": "GPU-xxxx",
      "pci_bus_id": "0000:18:00.0",
      "driver_version": "550.54",
      "runtime_version": "12.4",
      "memory_total_mb": 49152,
      "status": "available"
    }
  ],
  "gpu_metrics": [
    {
      "vendor": "nvidia",
      "index": 0,
      "uuid": "GPU-xxxx",
      "pci_bus_id": "0000:18:00.0",
      "memory_used_mb": 4096,
      "memory_free_mb": 45056,
      "utilization_gpu": 12.5,
      "utilization_mem": 8.0,
      "temperature_c": 45,
      "power_w": 80,
      "health": "healthy",
      "collected_at": "2026-06-13T10:00:00Z"
    }
  ],
  "diagnostics": [
    {
      "name": "gopsutil",
      "type": "system",
      "vendor": "",
      "available": true,
      "tool_path": "",
      "error": "",
      "checked_at": "2026-06-13T10:00:00Z"
    },
    {
      "name": "mock",
      "type": "gpu",
      "vendor": "mock",
      "available": true,
      "tool_path": "",
      "error": "",
      "checked_at": "2026-06-13T10:00:00Z"
    }
  ]
}
```

响应：

```json
{
  "accepted": true,
  "message": "resources updated"
}
```

---

## 17. Server 处理逻辑

Server 收到资源上报后：

1. 校验 Agent 是否存在；
2. 更新 Node 主机信息；
3. 更新 OS / CPU / Memory / Swap 最新状态；
4. 更新文件系统最新状态；
5. 更新 `agent_metrics_url`；
6. 更新 `last_resource_report_at`；
7. Upsert GPUDevice；
8. 更新 GPU 最新指标；
9. 保存 Collector 诊断信息；
10. 更新 Server 侧 Prometheus 指标缓存；
11. 不因为部分数据异常导致整次上报失败；
12. 返回 accepted。

---

## 18. 查询 API

### 18.1 查询节点列表

```http
GET /api/nodes
```

响应：

```json
{
  "items": [
    {
      "id": "node-001",
      "name": "gpu-server-001",
      "hostname": "gpu-server-001",
      "ip": "192.168.1.10",
      "status": "online",
      "cpu_usage_percent": 18.5,
      "memory_usage_percent": 25.0,
      "gpu_count": 8,
      "agent_metrics_url": "http://192.168.1.10:18080/metrics",
      "last_heartbeat_at": "2026-06-13T10:00:00Z",
      "last_resource_report_at": "2026-06-13T10:00:00Z"
    }
  ]
}
```

### 18.2 查询节点详情

```http
GET /api/nodes/{node_id}
```

包括：

1. 节点基础信息；
2. OS 信息；
3. CPU 信息；
4. 内存信息；
5. Swap 信息；
6. 文件系统信息；
7. GPU 列表；
8. Collector 诊断；
9. 最近心跳；
10. 最近资源上报时间；
11. Agent metrics URL。

### 18.3 查询 GPU 列表

```http
GET /api/gpus
```

支持参数：

```text
node_id
vendor
status
```

### 18.4 查询 GPU 详情

```http
GET /api/gpus/{gpu_id}
```

返回：

1. GPU 设备信息；
2. 最新指标；
3. 所属节点；
4. 最近采集时间。

---

## 19. Prometheus 指标设计

### 19.1 Agent OS 指标

```text
lightai_agent_up
lightai_agent_info
lightai_node_uptime_seconds
lightai_node_cpu_cores
lightai_node_cpu_usage_ratio
lightai_node_memory_total_bytes
lightai_node_memory_used_bytes
lightai_node_memory_available_bytes
lightai_node_memory_usage_ratio
lightai_node_swap_total_bytes
lightai_node_swap_used_bytes
lightai_node_swap_usage_ratio
lightai_node_filesystem_total_bytes
lightai_node_filesystem_used_bytes
lightai_node_filesystem_available_bytes
lightai_node_filesystem_usage_ratio
```

### 19.2 Agent GPU 指标

```text
lightai_gpu_info
lightai_gpu_memory_total_bytes
lightai_gpu_memory_used_bytes
lightai_gpu_memory_free_bytes
lightai_gpu_utilization_ratio
lightai_gpu_memory_utilization_ratio
lightai_gpu_temperature_celsius
lightai_gpu_power_watts
lightai_gpu_health_status
```

### 19.3 Agent 采集状态指标

```text
lightai_agent_heartbeat_success_total
lightai_agent_heartbeat_failed_total
lightai_agent_resource_collect_success_total
lightai_agent_resource_collect_failed_total
lightai_agent_collector_available
lightai_agent_collector_collect_duration_seconds
```

### 19.4 Server 指标

```text
lightai_server_up
lightai_server_nodes_total
lightai_server_nodes_online
lightai_server_gpus_total
lightai_server_instances_total
lightai_server_instances_running
lightai_server_tasks_pending
lightai_server_tasks_running
lightai_server_tasks_failed_total
lightai_server_api_requests_total
lightai_server_api_request_duration_seconds
```

---

## 20. Label 规范

推荐 label：

```text
node_id
node_name
vendor
gpu_index
gpu_uuid
mount_point
device
collector
model_name
instance_id
task_type
status
```

禁止放入 label：

```text
完整错误信息
Docker 命令
完整 endpoint URL
长 stdout / stderr
动态 request_id
高频变化 timestamp
用户输入内容
```

高变化信息进入日志或数据库，不进入 Prometheus label。

---

## 21. `/metrics/targets` 动态发现

Server 提供：

```http
GET /metrics/targets
```

Prometheus 通过该接口动态发现 Agent。

返回格式：

```json
[
  {
    "targets": ["192.168.1.10:18080"],
    "labels": {
      "job": "lightai-agent",
      "node_id": "node-001",
      "node_name": "gpu-server-001"
    }
  }
]
```

这个机制可以避免每新增一台 Agent 都手工修改 Prometheus 配置。

---

## 22. 状态与健康判断

GPU 状态：

```text
available
unavailable
unknown
```

GPU 健康：

```text
healthy
warning
critical
unknown
```

节点状态：

```text
online
offline
unknown
maintenance
```

判断建议：

1. Agent 心跳正常，节点 online；
2. 超过 offline timeout 未收到心跳，节点 offline；
3. Collector 正常采集到 GPU，GPU available；
4. 上一次存在、本次未上报，可以先标记 unknown；
5. 连续多次未出现，可以标记 unavailable；
6. 温度、功耗、错误状态等异常时标记 warning 或 critical；
7. 第一阶段不做复杂阈值，只保留字段和基础判断。

---

## 23. 采集失败处理

采集失败必须区分：

```text
Collector 不可用
命令不存在
命令超时
命令返回错误
输出解析失败
没有发现 GPU
Docker 不可用
系统指标采集失败
```

没有发现 GPU 不等于 Agent 异常。
没有发现 GPU 应显示为诊断信息。

示例：

```text
nvidia collector unavailable: nvidia-smi not found
metax collector unavailable: mx-smi not configured
mock collector available
docker collector unavailable: docker socket not found
```

Agent 不应因为 GPU 不存在或 Docker 不可用而退出。

---

## 24. WSL2 开发环境策略

当前开发环境是 WSL2 Ubuntu 24.04，可能没有真实 GPU 或 GPU 工具不完整。

因此第一阶段必须支持：

1. MockGPUCollector；
2. MockSystemCollector 或 gopsutil；
3. 无 GPU 启动 Agent；
4. 无 Docker 启动 Agent；
5. Collector 诊断可见；
6. 资源上报链路可验证；
7. `/metrics` 可访问；
8. Grafana dashboard 可以基于 Mock 指标调试。

---

## 25. 日志要求

Agent 资源采集日志至少包括：

```text
collector name
collector type
collector vendor
collector available
discover gpu count
metrics gpu count
error message
elapsed time
```

Server 资源接收日志至少包括：

```text
agent_id
node_id
cpu_usage
memory_usage
filesystem_count
gpu_device_count
gpu_metric_count
diagnostic_count
reported_at
处理结果
```

日志中不要打印过长原始命令输出，必要时截断。

---

## 26. 测试要求

第一阶段至少包括：

1. SystemCollector 单元测试；
2. MockGPUCollector 单元测试；
3. CollectorRegistry 测试；
4. GPU ID 生成测试；
5. 资源上报 API 测试；
6. Node / GPU upsert 测试；
7. 采集失败不崩溃测试；
8. `/metrics` endpoint 测试；
9. `/metrics/targets` 测试。

验收场景：

```text
无 GPU 环境：
- Agent 启动
- SystemCollector 正常采集 OS 信息
- MockGPUCollector 上报 1 张 GPU
- Agent /metrics 可访问
- Server 保存 Node / GPU
- API 查询可见
- Prometheus targets 可见

Collector 失败：
- NvidiaCollector 工具不存在
- Agent 不崩溃
- Server 收到 diagnostics
- API 可看到错误信息
```

---

## 27. MVP 完成标准

资源监控模块完成后，应达到：

1. 启动 Server；
2. 启动 Agent；
3. Agent 注册成功；
4. Agent 心跳正常；
5. Agent 上报 OS 资源；
6. Agent 上报 GPU 资源；
7. Agent `/metrics` 可访问；
8. Server `/metrics` 可访问；
9. Server `/metrics/targets` 可访问；
10. Server 能看到节点；
11. Server 能看到 CPU / 内存 / 磁盘；
12. Server 能看到 GPU；
13. 无真实 GPU 时 Mock 数据可见；
14. Collector 失败有诊断；
15. Agent 不因采集失败退出。

