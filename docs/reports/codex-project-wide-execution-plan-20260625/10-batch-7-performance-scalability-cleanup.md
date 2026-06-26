# Batch 7 — Performance and Scalability Cleanup

## 目标

处理 P2/P3 性能与扩展性问题，让小规模多节点/多模型场景不被明显 N+1、无分页、日志大响应、前端大 chunk 阻塞。

覆盖：

- R-011 后端/前端完整关闭
- R-015
- performance/scalability findings

## 任务

### 7.1 API pagination/index audit

审查 list-heavy APIs：

- nodes
- GPUs
- artifacts
- locations
- backend runtimes
- NBRs
- deployments
- instances
- run plans
- audit logs
- operation logs
- metrics targets

要求：

- 大列表有 limit/offset 或 cursor。
- 常用过滤字段有索引。
- tenant_id 过滤字段有索引。
- 排序字段有合理索引。
- 文档说明默认 limit 和 max limit。

输出：

```text
docs/performance/api-pagination-index-audit.md
```

### 7.2 Aggregate NBR endpoint 完整使用

如果 Batch 4 已新增 endpoint，本批完成：

- 所有前端 per-node fan-out 替换。
- 测试聚合 endpoint tenant filter。
- 文档写入 API contract/OpenAPI。

### 7.3 Docker image check caching

`/check-request` 对 Docker image list/inspect 可能慢。

实现建议：

- Agent 侧短 TTL cache。
- Server 侧 probe history。
- UI 显示上次 check 时间。
- 手动 force refresh 参数。
- 不允许 cache 直接绕过 missing image 的安全判断，除非 evidence 未过期并带来源。

### 7.4 Logs performance

补充：

- 前端默认 tail。
- 自动刷新间隔。
- backoff。
- 大日志下载/查看策略。
- 服务端 hard bytes cap。

### 7.5 Frontend code splitting

处理 Vite main chunk warning：

- route-level lazy loading。
- vendor manual chunks。
- charts/editor/json viewer 分块。
- 构建阈值记录。

验收目标：

- `npm run build` 不再出现 main chunk 超 500 kB 警告；或项目明确设置新阈值并说明原因。
- 不引入运行时错误。

## 验证命令

```bash
go test ./internal/server/api
go test ./...
cd web && npm test
cd web && npm run build
```

## 验收

- R-011 CLOSED。
- R-015 CLOSED 或有明确 accepted threshold。
- API pagination/index audit 完成。
- NBR fan-out 消除。
- logs 大响应受控。
