# LightAI Go Prometheus / Grafana 集成设计

## 1. 设计目标

LightAI Go 需要内置 Prometheus + Grafana 监控能力，并且能在 LightAI Go Web 页面中看到监控图表。

这里的“内置”不是指把 Prometheus 和 Grafana 编译进 Go 二进制，而是指：

1. LightAI Go 提供 Prometheus / Grafana 的部署配置；
2. LightAI Go 可以通过 Docker Compose 一键启动 Prometheus / Grafana；
3. LightAI Go Server 提供 Prometheus 动态发现接口；
4. LightAI Go 提供 Grafana datasource 和 dashboard provisioning；
5. LightAI Go Web 页面可以内嵌 Grafana dashboard 或 panel；
6. 用户无需手工配置复杂监控系统即可看到资源监控图表。

第一阶段以 Docker Compose 托管 Prometheus / Grafana 为主。
后续可以支持外部已有 Prometheus / Grafana。

---

## 2. 学习 GPUStack 的监控架构

LightAI Go 学习 GPUStack 的整体监控边界：

```text
Agent / Worker 负责采集
Server 负责管理和动态发现
Prometheus 负责抓取和存储时序指标
Grafana 负责展示 Dashboard
Web 页面负责内嵌 Grafana
```

LightAI Go 不复制 GPUStack 代码，但学习它的设计思路：

1. 内置 Prometheus / Grafana；
2. 允许外接已有 Prometheus / Grafana；
3. Agent 暴露 `/metrics`；
4. Server 暴露 `/metrics`；
5. Server 提供动态 targets；
6. Grafana 默认提供资源监控 Dashboard；
7. 监控系统不替代业务状态同步。

---

## 3. 架构边界

LightAI Go 的业务管理链路：

```text
Agent → Server → SQLite → Web
```

Prometheus / Grafana 的监控展示链路：

```text
Prometheus → Server /metrics
Prometheus → Server /metrics/targets
Prometheus → Agent /metrics
Grafana → Prometheus
LightAI Web → Grafana iframe / reverse proxy
```

两条链路必须分开。

LightAI Go 管理：

1. 节点；
2. GPU；
3. 运行环境；
4. 模型；
5. 实例；
6. 任务；
7. 当前状态；
8. 错误信息；
9. Docker 命令快照。

Prometheus 管理：

1. 时序指标；
2. CPU 使用趋势；
3. 内存使用趋势；
4. 磁盘使用趋势；
5. GPU 使用趋势；
6. 节点在线趋势；
7. 实例健康趋势；
8. 任务统计趋势。

Grafana 管理：

1. 监控图表；
2. Dashboard；
3. 告警；
4. 可视化展示。

---

## 4. 不建议的方式

不建议把 Prometheus 和 Grafana 真正编译进 LightAI Go 二进制。

原因：

1. Prometheus 和 Grafana 本身是独立系统；
2. Grafana 有自己的前端、数据库、插件和配置体系；
3. 编译进 Go 二进制会导致升级困难；
4. 部署、备份、插件、权限都会变复杂；
5. 出问题时现场排障更困难。

推荐方式：

```text
LightAI Go 管理和托管 Prometheus / Grafana
Prometheus / Grafana 独立运行
LightAI Web 内嵌 Grafana 页面
```

---

## 5. Observability 模式

配置支持三种模式：

```yaml
observability:
  mode: "builtin"   # builtin / external / disabled
```

### 5.1 builtin

由 LightAI Go 提供内置部署配置，使用 Docker Compose 启动 Prometheus / Grafana。

适合：

1. 中小客户；
2. 无现成监控系统；
3. 快速交付；
4. 单套平台管理数台 GPU 服务器。

### 5.2 external

客户已有 Prometheus / Grafana，LightAI Go 只暴露 `/metrics` 和 `/metrics/targets`。

适合：

1. 大型客户；
2. 已有统一监控平台；
3. 运维体系成熟；
4. 需要接入现有告警系统。

### 5.3 disabled

不启用 Prometheus / Grafana，仅保留 LightAI Go 自身状态展示。

适合：

1. 极简部署；
2. 临时测试；
3. 安全隔离环境；
4. 客户暂不需要时序监控。

---

## 6. 部署目录

第一阶段采用 Docker Compose。

目录建议：

```text
deploy/
  observability/
    docker-compose.yml
    prometheus/
      prometheus.yml
      targets/
        agents.yml
    grafana/
      provisioning/
        datasources/
          prometheus.yml
        dashboards/
          lightai.yml
      dashboards/
        lightai-gpu-overview.json
        lightai-node-overview.json
        lightai-instance-overview.json
```

说明：

1. `prometheus.yml` 是 Prometheus 主配置；
2. `targets/agents.yml` 是静态兜底 targets 文件；
3. 正式场景优先使用 Server `/metrics/targets` 动态发现；
4. Grafana datasource 自动指向 Prometheus；
5. Grafana dashboards 自动预置；
6. Web 页面通过 iframe 或 reverse proxy 内嵌 Grafana。

---

## 7. Docker Compose 示例

文件：`deploy/observability/docker-compose.yml`

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: lightai-prometheus
    restart: unless-stopped
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
      - "--storage.tsdb.path=/prometheus"
      - "--storage.tsdb.retention.time=15d"
      - "--web.enable-lifecycle"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./prometheus/targets:/etc/prometheus/targets:ro
      - prometheus-data:/prometheus
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    container_name: lightai-grafana
    restart: unless-stopped
    environment:
      - GF_SECURITY_ALLOW_EMBEDDING=true
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer
      - GF_SERVER_ROOT_URL=%(protocol)s://%(domain)s:%(http_port)s/grafana/
      - GF_SERVER_SERVE_FROM_SUB_PATH=true
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - ./grafana/dashboards:/var/lib/grafana/dashboards:ro
    ports:
      - "3000:3000"

volumes:
  prometheus-data:
  grafana-data:
```

注意：

1. 生产环境是否启用匿名访问，需要根据客户安全要求决定；
2. 内嵌 Grafana 需要允许 iframe；
3. 如果未来做统一登录，应替换为反向代理鉴权或 auth proxy；
4. Prometheus 和 Grafana 端口是否暴露给外部，应由部署策略决定。

---

## 8. Prometheus 配置示例

文件：`deploy/observability/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 5s
  evaluation_interval: 5s

scrape_configs:
  - job_name: "lightai-server"
    metrics_path: "/metrics"
    static_configs:
      - targets:
          - "host.docker.internal:8080"

  - job_name: "lightai-agents"
    metrics_path: "/metrics"
    http_sd_configs:
      - url: "http://host.docker.internal:8080/metrics/targets"
        refresh_interval: 10s
```

如果 `host.docker.internal` 在 Linux 环境不可用，可以改为 Server 实际 IP，例如：

```yaml
      - targets:
          - "192.168.1.100:8080"
```

也可以增加静态兜底配置：

```yaml
  - job_name: "lightai-agents-static"
    metrics_path: "/metrics"
    file_sd_configs:
      - files:
          - "/etc/prometheus/targets/agents.yml"
        refresh_interval: 10s
```

---

## 9. Agent target 文件示例

文件：`deploy/observability/prometheus/targets/agents.yml`

```yaml
- targets:
    - "192.168.1.10:18080"
    - "192.168.1.11:18080"
  labels:
    job: "lightai-agent"
```

第一阶段可以先手工维护 `agents.yml`。
后续由 Server 根据已注册节点通过 `/metrics/targets` 动态生成。

---

## 10. Grafana Datasource Provisioning

文件：`deploy/observability/grafana/provisioning/datasources/prometheus.yml`

```yaml
apiVersion: 1

datasources:
  - name: LightAI Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

---

## 11. Grafana Dashboard Provisioning

文件：`deploy/observability/grafana/provisioning/dashboards/lightai.yml`

```yaml
apiVersion: 1

providers:
  - name: "LightAI Dashboards"
    orgId: 1
    folder: "LightAI"
    type: file
    disableDeletion: false
    editable: true
    options:
      path: /var/lib/grafana/dashboards
```

Dashboard 文件后续放在：

```text
deploy/observability/grafana/dashboards/
```

建议内置：

```text
lightai-gpu-overview.json
lightai-node-overview.json
lightai-instance-overview.json
```

---

## 12. LightAI Web 内嵌方式

LightAI Web 中新增“监控看板”页面。

页面建议：

```text
/monitoring
/monitoring/gpu
/monitoring/nodes
/monitoring/instances
```

第一阶段可以用 iframe 内嵌 Grafana dashboard：

```html
<iframe
  src="/grafana/d/lightai-gpu-overview/gpu-overview?orgId=1&kiosk"
  width="100%"
  height="900"
  frameborder="0">
</iframe>
```

为了让 `/grafana/` 生效，Server 或前端网关需要反向代理到 Grafana：

```text
LightAI Web /grafana/* → Grafana http://127.0.0.1:3000/*
```

如果不做反向代理，也可以直接打开 Grafana 地址：

```text
http://server-ip:3000
```

但产品体验不如内嵌。

---

## 13. Server 反向代理设计

Server 后续可以提供反向代理：

```text
/grafana/* → http://127.0.0.1:3000/*
```

第一阶段建议只内嵌 Grafana，不内嵌 Prometheus。

原因：

1. 普通用户主要看 Grafana；
2. Prometheus 更偏工程诊断；
3. 暴露 Prometheus 查询界面有安全风险；
4. Grafana 已经可以满足图表展示。

管理页面可以提供：

```text
Grafana 状态
Prometheus 状态
打开 Grafana
重启监控组件
查看 scrape targets
```

---

## 14. ObservabilityManager 设计

Server 后续可以增加 ObservabilityManager。

职责：

1. 检查 Prometheus 是否可访问；
2. 检查 Grafana 是否可访问；
3. 提供 `/metrics/targets`；
4. 生成 Prometheus Agent targets 文件；
5. 生成 Grafana dashboard provisioning 文件；
6. 提供监控状态 API；
7. 提供前端监控页面配置；
8. 提供 Grafana iframe URL。

第一阶段可以先不实现完整 ObservabilityManager，只保留部署配置和接口规划。

---

## 15. 监控状态 API

后续 API：

```http
GET /api/observability/status
```

返回：

```json
{
  "mode": "builtin",
  "prometheus": {
    "enabled": true,
    "url": "http://127.0.0.1:9090",
    "healthy": true
  },
  "grafana": {
    "enabled": true,
    "url": "http://127.0.0.1:3000",
    "public_url": "/grafana/",
    "healthy": true
  },
  "dashboards": [
    {
      "name": "GPU Overview",
      "path": "/grafana/d/lightai-gpu-overview/gpu-overview?orgId=1&kiosk"
    }
  ]
}
```

---

## 16. Agent 指标命名

Agent OS 指标：

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

Agent GPU 指标：

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

Agent 采集状态指标：

```text
lightai_agent_heartbeat_success_total
lightai_agent_heartbeat_failed_total
lightai_agent_resource_collect_success_total
lightai_agent_resource_collect_failed_total
lightai_agent_collector_available
lightai_agent_collector_collect_duration_seconds
```

---

## 17. Server 指标命名

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

## 18. Label 规范

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

## 19. Grafana Dashboard 建议

第一批 Dashboard：

### 19.1 GPU Overview

展示：

1. GPU 总数；
2. 在线节点数；
3. GPU 显存使用率；
4. GPU 利用率；
5. GPU 温度；
6. GPU 功耗；
7. 按节点分组的 GPU 列表；
8. 异常 GPU 数量。

### 19.2 Node Overview

展示：

1. 节点在线状态；
2. Agent up；
3. CPU 使用率；
4. 内存使用率；
5. 文件系统使用率；
6. 心跳成功 / 失败；
7. 资源采集成功 / 失败；
8. 节点 GPU 数量；
9. 节点最近上报时间。

### 19.3 Instance Overview

展示：

1. 实例总数；
2. running 实例；
3. failed 实例；
4. 实例健康状态；
5. 按模型分组的实例数量；
6. 按节点分组的实例数量。

第一阶段可以先只做 Node Overview 和 GPU Overview。

---

## 20. 安全设计

第一阶段为了快速验证，可以启用 Grafana anonymous Viewer。

但生产环境需要注意：

1. Grafana 只能给 Viewer 权限；
2. 不允许匿名编辑 Dashboard；
3. 不要开放 Prometheus 管理接口到公网；
4. Grafana 是否开放给外部网络由部署决定；
5. 内嵌 iframe 应优先走 LightAI Server 反向代理；
6. 后续做统一登录后，替换匿名访问。

客户现场如对安全要求较高，可以使用：

```text
LightAI 登录
  ↓
Server 反向代理鉴权
  ↓
Grafana auth proxy
```

第一阶段暂不实现复杂 SSO。

---

## 21. 第一阶段实现策略

第一阶段分三步：

### Step 1：预留 metrics endpoint

实现：

```text
Server /metrics
Agent /metrics
Server /metrics/targets
```

可以先只输出基础指标。

### Step 2：提供 Docker Compose

提供：

```text
deploy/observability/docker-compose.yml
deploy/observability/prometheus/prometheus.yml
deploy/observability/grafana/provisioning/
```

做到可以手工启动 Prometheus + Grafana。

### Step 3：Web 内嵌 Grafana

Web 增加“监控看板”页面。

初期可以直接 iframe 到 Grafana。

---

## 22. 验收标准

完成后应达到：

1. Server `/metrics` 可访问；
2. Agent `/metrics` 可访问；
3. Server `/metrics/targets` 可访问；
4. Prometheus 能抓取 Server 指标；
5. Prometheus 能通过 `/metrics/targets` 发现 Agent；
6. Prometheus 能抓取 Agent 指标；
7. Grafana 能连接 Prometheus；
8. Grafana 有 LightAI Dashboard；
9. LightAI Web 页面能看到 Grafana 图表；
10. 没有真实 GPU 时，MockCollector 指标也能展示。

---

## 23. 后续扩展

后续可以扩展：

1. Server 自动生成 Prometheus targets；
2. Server 管理 Prometheus / Grafana 生命周期；
3. Grafana dashboard 自动导入；
4. Grafana iframe 统一鉴权；
5. 告警规则；
6. 企业微信 / 邮件告警；
7. GPU 异常告警；
8. 实例异常告警；
9. Token / 成本监控看板。

