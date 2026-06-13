# LightAI Go 开发阶段状态

> 最后更新：2026-06-13 17:30
> 当前阶段：准备开始 Phase 0

## 阶段总览

| Phase | 名称 | 状态 | Commit | 备注 |
|-------|------|------|--------|------|
| 0 | 基础骨架 | ⏳ pending | - | - |
| 0.5 | 认证/RBAC | ⏳ pending | - | - |
| 1 | Agent 注册心跳 | ⏳ pending | - | - |
| 2A | System/Registry/Mock | ⏳ pending | - | - |
| 2B | NVIDIA Collector | ⏳ pending | - | - |
| 2C | MetaX Collector | 🚫 本轮不做 | - | - |
| 3 | 运行环境 | 🚫 本轮不做 | - | - |
| 4 | 模型定义 | 🚫 本轮不做 | - | - |
| 5 | 实例任务 | 🚫 本轮不做 | - | - |
| 6 | Docker 启停 | 🚫 本轮不做 | - | - |
| 7 | 实例健康检查 | 🚫 本轮不做 | - | - |
| 8 | Web | 🚫 本轮不做 | - | - |
| 9 | Prometheus/Grafana | 🚫 本轮不做 | - | - |

## 文档审查状态

| 文档 | 状态 | 问题数 |
|------|------|--------|
| 00-project-scope.md | ✅ 已审查 | 0 |
| 01-architecture.md | ✅ 已审查 | 0 |
| 02-server-agent-design.md | ✅ 已审查 | 0 |
| 03-resource-monitoring-design.md | ✅ 已审查 | 0 |
| 04-observability-design.md | ✅ 已审查 | 0 |
| 05-runtime-environment-design.md | ✅ 已审查 | 0 |
| 06-model-design.md | ✅ 已审查 | 0 |
| 07-instance-lifecycle-design.md | ✅ 已审查 | 0 |
| 08-engineering-contracts.md | ✅ 已审查 | 0 |
| 09-auth-tenant-design.md | ✅ 已审查 | 0 |
| 10-mvp-development-plan.md | ✅ 已审查 | 0 |

## 新增文档

| 文档 | 状态 |
|------|------|
| REVIEW-GPUSTACK-AUDIT.md | ✅ 已创建 |
| REVIEW-GPUSTACK-UI.md | ✅ 已创建 |
| RUNBOOK-LOCAL-VERIFY.md | ✅ 已创建 |
| PHASE-STATUS.md | ✅ 已创建（本文档） |

## 环境信息

| 项目 | 值 |
|------|-----|
| Go 版本 | 待检测 |
| OS | Linux (WSL2) |
| NVIDIA GPU | 待检测 |
| nvidia-smi | 待检测 |

---

## Phase 0：基础骨架

### 状态：⏳ pending

### 目标
Server / Agent 基础骨架、配置、日志、版本、healthz、metrics、metrics targets

### 测试命令
```bash
go build ./cmd/server && go build ./cmd/agent
go test ./...
curl http://localhost:8080/healthz
curl http://localhost:8080/metrics
```

### 测试结果
待执行

### Commit
待提交

---

## Phase 0.5：认证、租户与 RBAC

### 状态：⏳ pending

### 目标
SQLite、Tenant、User、Membership、Role、Permission、Session、Argon2id、login/logout、CSRF、middleware

### 测试命令
```bash
go test ./...
curl -X POST http://localhost:8080/api/auth/login -d '{"username":"admin","password":"admin123"}'
curl -b cookies.txt http://localhost:8080/api/auth/me
```

### 测试结果
待执行

### Commit
待提交

---

## Phase 1：Agent 注册与心跳

### 状态：⏳ pending

### 目标
Agent register、heartbeat、Node 表、在线/离线、Agent token、/api/nodes、/metrics/targets

### 测试命令
```bash
go test ./...
curl http://localhost:8080/api/nodes
curl http://localhost:8080/metrics/targets
```

### 测试结果
待执行

### Commit
待提交

---

## Phase 2A：System / Registry / Mock

### 状态：⏳ pending

### 目标
SystemCollector、gopsutil、CollectorRegistry、MockGPUCollector、资源上报、字节/百分比规则

### 测试命令
```bash
go test ./...
curl http://localhost:9090/metrics | grep lightai_system
curl http://localhost:8080/api/gpus
```

### 测试结果
待执行

### Commit
待提交

---

## Phase 2B：NVIDIA Collector

### 状态：⏳ pending

### 目标
NvidiaCollector、nvidia-smi 解析、真实 NVIDIA 验收

### NVIDIA 环境检测
```bash
which nvidia-smi || true
nvidia-smi || true
```
待执行

### 测试命令
```bash
go test ./...
curl http://localhost:9090/metrics | grep lightai_gpu
curl http://localhost:8080/api/gpus | jq '.[] | select(.vendor=="nvidia")'
```

### 测试结果
待执行

### 真实 NVIDIA 验收
- [ ] 待检测

### Commit
待提交

---

## 已知问题

无

## 下一步

开始 Phase 0 实现
