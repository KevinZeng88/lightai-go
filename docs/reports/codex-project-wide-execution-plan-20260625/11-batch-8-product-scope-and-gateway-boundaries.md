# Batch 8 — Product Scope and Gateway Boundaries

## 目标

收敛当前产品成熟度声明，避免 UI/API/文档暗示尚未实现的能力已经可用；同时为下一阶段功能形成设计入口。

覆盖：

- R-012
- R-013
- R-014
- Q-008
- product maturity claims

## 任务

### 8.1 多副本/分布式能力边界

当前状态：DB/UI 可能存在 replicas/distributed 字段，但 start path 是 single instance。

处理选项：

- 短期：API 拒绝 `replicas > 1`，UI 隐藏或禁用多副本字段。
- 中期：设计 scheduler/multi-runplan/multi-instance 后再启用。

本批默认：短期拒绝，避免误导。

验收：

- create/update/start/preflight/dry-run 对 replicas > 1 一致 400 或 disabled。
- 文档说明当前只支持 single instance。
- UI 不展示可配置多副本，或显示“not supported”。

### 8.2 Prometheus/Grafana 管理边界

当前状态：Go server 未完整 supervisor Prom/Grafana。

处理：

- docs/CURRENT、observability docs、UI 文案同步。
- 如果有 `/observability/status`，必须反映真实 external script mode。
- 不宣称 server-managed Prom/Grafana，除非实现 supervisor。

### 8.3 OpenAI-compatible gateway/API key/usage/billing

当前状态：主要是 instance runtime probe，不是完整 platform gateway。

处理：

- 文档明确当前不支持完整 gateway/API key/usage billing。
- UI 如有入口，标记 experimental 或隐藏。
- 新增设计文档：

```text
docs/design/openai-gateway-api-key-usage-billing.md
```

设计至少包括：

- API key model。
- tenant/project/consumer。
- routing to deployment/instance。
- usage metering。
- audit log。
- rate limit。
- quota。
- billing summary。
- security boundary。
- tests/E2E gate。

### 8.4 MetaX readiness bar

如果无真实 MetaX 硬件：

- 保持 documented blocker。
- 不宣称 MetaX production ready。
- 保留 mock/template tests。
- 明确需要真实 hardware evidence 才能 close。

如果有 MetaX 硬件：

- 新增 real smoke。
- 记录 device binding、env、image、model、logs、stop/cleanup。

## 验证命令

```bash
go test ./...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm test
cd web && npm run build
```

如果涉及 UI 文案：

```bash
cd web && npm test
```

## 验收

- R-012 CLOSED：多副本不再误导。
- R-013 CLOSED：observability claim 与实现一致。
- R-014 CLOSED：OpenAI gateway/usage 不再被误宣称，设计文档存在。
- Q-008 有明确 evidence bar。
