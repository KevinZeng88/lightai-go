# LightAI Go 本地验证手册

> 最后更新：2026-06-13
> 适用版本：Phase 0 - Phase 2B

## 目录

1. [前置条件](#前置条件)
2. [启动 Server](#启动-server)
3. [启动 Agent](#启动-agent)
4. [查看日志](#查看日志)
5. [Health Check](#health-check)
6. [Metrics](#metrics)
7. [登录与认证](#登录与认证)
8. [CSRF Token](#csrf-token)
9. [用户信息](#用户信息)
10. [节点查询](#节点查询)
11. [GPU 查询](#gpu-查询)
12. [Agent Metrics 检查](#agent-metrics-检查)
13. [Server Metrics Targets](#server-metrics-targets)
14. [各阶段验收标准](#各阶段验收标准)
15. [常见失败原因和排查](#常见失败原因和排查)

---

## 前置条件

```bash
# 确保 Go 环境可用
go version  # >= 1.21

# 进入项目目录
cd ~/projects/ai-platform-study/lightai-go

# 编译
go build ./cmd/server
go build ./cmd/agent
```

---

## 启动 Server

```bash
# 使用开发配置启动
./server -config configs/server.dev.yaml

# 或使用环境变量
LIGHTAI_SERVER_PORT=18080 ./server
```

Server 默认监听 `http://127.0.0.1:18080`。

启动成功后应看到日志：
```
INFO  server started  addr=127.0.0.1:18080
```

---

## 启动 Agent

```bash
# 使用开发配置启动
./agent -config configs/agent.dev.yaml

# 关键参数
./agent \
  -server-url http://127.0.0.1:18080 \
  -agent-token <bootstrap-token> \
  -agent-id agent-01
```

Agent 启动后会依次：注册 → 心跳 → 资源采集。

---

## 查看日志

```bash
# Server 日志输出到 stdout
./server -config configs/server.dev.yaml 2>&1 | tee server.log

# Agent 日志输出到 stdout
./agent -config configs/agent.dev.yaml 2>&1 | tee agent.log

# 日志级别通过配置控制（debug/info/warn/error）
```

---

## Health Check

```bash
# Server healthz
curl -s http://127.0.0.1:18080/healthz
# 预期：{"status":"ok"}

# Agent healthz（Agent 也暴露 healthz）
curl -s http://127.0.0.1:19091/healthz
# 预期：{"status":"ok"}
```

---

## Metrics

```bash
# Server metrics
curl -s http://127.0.0.1:18080/metrics | head -20
# 预期：包含 lightai_ 前缀的 Prometheus 指标

# Agent metrics
curl -s http://127.0.0.1:19091/metrics | head -20
# 预期：包含 lightai_ 前缀的 Prometheus 指标
# Phase 2B 完成后期望包含 nvidia_ 相关指标
```

---

## 登录与认证

```bash
# 登录（获取 Session Cookie）
curl -v -c cookies.txt -X POST http://127.0.0.1:18080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"admin123"}'

# 预期：返回 200，Set-Cookie 包含 lightai_session
# 首次登录可能要求修改密码（must_change_password=true）
```

---

## CSRF Token

```bash
# 获取 CSRF token
curl -s -b cookies.txt http://127.0.0.1:18080/api/auth/csrf-token
# 预期：{"csrf_token":"..."}

# 在状态变更请求中使用
curl -X POST http://127.0.0.1:18080/api/users \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"testuser","password":"test123","display_name":"Test User"}'
```

---

## 用户信息

```bash
# 获取当前用户信息
curl -s -b cookies.txt http://127.0.0.1:18080/api/auth/me
# 预期：{"user_id":"...","username":"admin","is_platform_admin":true,...}
```

---

## 节点查询

```bash
# 列出所有节点
curl -s -b cookies.txt http://127.0.0.1:18080/api/nodes
# 预期：[{"id":"...","hostname":"...","status":"online",...}]

# 查询单个节点
curl -s -b cookies.txt http://127.0.0.1:18080/api/nodes/<node_id>
```

---

## GPU 查询

```bash
# 列出所有 GPU
curl -s -b cookies.txt http://127.0.0.1:18080/api/gpus
# 预期：[{"id":"...","vendor":"nvidia","name":"...","memory_total_bytes":...}]

# 按节点过滤
curl -s -b cookies.txt "http://127.0.0.1:18080/api/gpus?node_id=<node_id>"

# 按厂商过滤
curl -s -b cookies.txt "http://127.0.0.1:18080/api/gpus?vendor=nvidia"
```

---

## Agent Metrics 检查

```bash
# Agent 指标（Phase 2A+）
curl -s http://127.0.0.1:19091/metrics | grep -E "lightai_(system|gpu)"

# 预期 Phase 2A：
# lightai_system_cpu_utilization_ratio
# lightai_system_memory_total_bytes
# lightai_system_memory_used_bytes

# 预期 Phase 2B（NVIDIA 可用时）：
# lightai_gpu_memory_total_bytes
# lightai_gpu_memory_used_bytes
# lightai_gpu_utilization_ratio
# lightai_gpu_temperature_celsius
# lightai_gpu_power_watts
```

---

## Server Metrics Targets

```bash
# Server metrics targets（Prometheus HTTP SD 格式）
curl -s http://127.0.0.1:18080/metrics/targets
# 预期：[{"targets":["<agent_host>:<metrics_port>"],"labels":{"agent_id":"...","hostname":"..."}}]

# 注意：只包含已注册、未删除、metrics_enabled=true 的节点
# 不按 online/offline 状态过滤
```

---

## 各阶段验收标准

### Phase 0 验收

```bash
go build ./cmd/server && go build ./cmd/agent
./server -config configs/server.dev.yaml &
SERVER_PID=$!
sleep 2
curl -s http://127.0.0.1:18080/healthz | grep '"ok"'
curl -s http://127.0.0.1:18080/metrics | grep 'lightai_'
kill $SERVER_PID
```

通过条件：healthz 返回 ok，metrics 包含 lightai_ 前缀指标。

### Phase 0.5 验收

```bash
# 启动 Server
./server -config configs/server.dev.yaml &
SERVER_PID=$!
sleep 2

# 登录
curl -c cookies.txt -X POST http://127.0.0.1:18080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"admin123"}'

# 获取 CSRF token
CSRF=$(curl -s -b cookies.txt http://127.0.0.1:18080/api/auth/csrf-token | jq -r '.csrf_token')

# 获取当前用户
curl -s -b cookies.txt http://127.0.0.1:18080/api/auth/me | jq '.username'

# 登出
curl -X POST http://127.0.0.1:18080/api/auth/logout \
  -b cookies.txt \
  -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080"

kill $SERVER_PID
```

通过条件：登录成功，me 返回 admin，登出成功。

### Phase 1 验收

```bash
# 启动 Server 和 Agent
./server -config configs/server.dev.yaml &
SERVER_PID=$!
sleep 2
./agent -config configs/agent.dev.yaml &
AGENT_PID=$!
sleep 5

# 登录并查询节点
curl -c cookies.txt -X POST http://127.0.0.1:18080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"admin123"}'
curl -s -b cookies.txt http://127.0.0.1:18080/api/nodes | jq '.[0].status'

# 检查 metrics targets
curl -s http://127.0.0.1:18080/metrics/targets | jq '.'

kill $AGENT_PID $SERVER_PID
```

通过条件：节点注册成功，状态为 online，metrics/targets 包含 Agent。

### Phase 2A 验收

```bash
# 启动 Server 和 Agent（development profile）
./server -config configs/server.dev.yaml &
SERVER_PID=$!
sleep 2
./agent -config configs/agent.dev.yaml &
AGENT_PID=$!
sleep 10

# 检查 Agent 系统指标
curl -s http://127.0.0.1:19091/metrics | grep -E "lightai_system"

# 检查节点详情（含系统信息）
curl -c cookies.txt -X POST http://127.0.0.1:18080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"admin123"}'
curl -s -b cookies.txt http://127.0.0.1:18080/api/nodes | jq '.[0]'

# development profile 下 Mock GPU 可用
curl -s -b cookies.txt http://127.0.0.1:18080/api/gpus | jq '.'

kill $AGENT_PID $SERVER_PID
```

通过条件：系统指标正常，节点详情包含 OS/CPU/内存，Mock GPU 出现在 development profile。

### Phase 2B 验收

```bash
# 检查 NVIDIA 环境
which nvidia-smi
nvidia-smi --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw --format=csv,noheader,nounits

# 启动 Server 和 Agent（production profile，启用 NvidiaCollector）
./server -config configs/server.prod.yaml &
SERVER_PID=$!
sleep 2
./agent -config configs/agent.prod.yaml &
AGENT_PID=$!
sleep 10

# 检查 Agent NVIDIA 指标
curl -s http://127.0.0.1:19091/metrics | grep -E "lightai_gpu.*nvidia"

# 检查 Server GPU API 返回 NVIDIA GPU
curl -c cookies.txt -X POST http://127.0.0.1:18080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"admin123"}'
curl -s -b cookies.txt http://127.0.0.1:18080/api/gpus | jq '.[] | select(.vendor=="nvidia")'

# 检查诊断
curl -s -b cookies.txt http://127.0.0.1:18080/api/nodes | jq '.[0].diagnostics'

kill $AGENT_PID $SERVER_PID
```

通过条件：nvidia-smi 可用时，Agent metrics 包含 NVIDIA 指标，GPU API 返回真实 NVIDIA GPU。

如果 nvidia-smi 不可用：
- 单元测试（NVIDIA 样例 CSV 解析）必须通过
- `docs/PHASE-STATUS.md` 标记真实 NVIDIA 验收 blocked

---

## 常见失败原因和排查

### Server 启动失败

```bash
# 检查端口占用
ss -tlnp | grep 18080

# 检查配置文件
cat configs/server.dev.yaml

# 检查 SQLite 数据库权限
ls -la data/
```

### Agent 注册失败

```bash
# 检查 Server 是否可达
curl http://127.0.0.1:18080/healthz

# 检查 Agent token 是否正确
grep agent_token configs/agent.dev.yaml

# 查看 Agent 日志中的错误信息
./agent -config configs/agent.dev.yaml 2>&1 | grep -i error
```

### NVIDIA Collector 失败

```bash
# 检查 nvidia-smi 是否可用
which nvidia-smi
nvidia-smi

# 检查 NVIDIA 驱动
lsmod | grep nvidia

# 检查 /dev/nvidia* 设备
ls -la /dev/nvidia*

# 手动执行查询命令
nvidia-smi --query-gpu=index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw --format=csv,noheader,nounits
```

### 认证失败

```bash
# 检查 bootstrap 管理员密码
# 默认：admin / admin123（首次登录需改密码）

# 检查 Cookie
curl -v -c cookies.txt http://127.0.0.1:18080/api/auth/me

# 检查 CSRF token 是否过期
curl -s -b cookies.txt http://127.0.0.1:18080/api/auth/csrf-token
```

### 数据库问题

```bash
# 重置数据库（开发环境）
rm -f data/lightai.db
# 重启 Server 将自动重新初始化

# 检查数据库内容
sqlite3 data/lightai.db ".tables"
sqlite3 data/lightai.db "SELECT * FROM tenants;"
sqlite3 data/lightai.db "SELECT * FROM users;"
```
