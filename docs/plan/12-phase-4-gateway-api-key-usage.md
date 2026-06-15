# Phase 4：Gateway + API Key + Usage

> 依赖：Phase 3（Web 页面可用，模型实例可正常运行）
> 周期：3-4 周

## 1. 目标

通过统一 Gateway 调用模型，不直接暴露实例端口。支持 API Key 鉴权和调用审计。

## 2. 范围

- LightAI Gateway（OpenAI-compatible `/v1/models`、`/v1/chat/completions`）
- ModelRoute（模型名 → 实例 endpoint 映射）
- API Key 管理（CRUD + 校验 + 权限）
- ModelUsageRecord 表启用 + 写入
- 基础限流（token-based，简单实现）
- Prometheus 指标（`lightai_model_request_total` 等）

## 3. 明确不做什么

- 计费（只记录 usage，不算费用）
- 复杂负载均衡（第一阶段 round_robin 即可）
- 多租户配额
- 复杂限流策略（token bucket / sliding window）
- Remote endpoint 纳管完整实现（只做 Gateway 代理）

## 4. 数据模型

### 4.1 ModelRoute（新建表）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| model_name | string | OpenAI API 中的 model 名 |
| deployment_id | uuid | 路由到部署 |
| instance_id | uuid | 路由到实例（replicas=1 时直接指向实例） |
| tenant_id | uuid | 租户 |
| route_policy | enum | round_robin（Phase 4 只支持这个） |
| enabled | bool | 是否启用 |
| weight | int | 权重（后续 weighted 策略使用） |
| created_at | timestamp | — |
| updated_at | timestamp | — |

### 4.2 API Key（可能在现有 user/auth 体系中扩展或新建表）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| tenant_id | uuid | 所属租户 |
| name | string | Key 名称 |
| key_hash | string | API Key 哈希值 |
| prefix | string | 明文前缀（如 `sk-abc...`），用于展示 |
| status | enum | active / disabled / expired |
| rate_limit_rpm | int | 每分钟请求限制 |
| rate_limit_tpm | int | 每分钟 token 限制 |
| last_used_at | timestamp | 最近使用时间 |
| expires_at | timestamp | 过期时间 |
| created_by | uuid | 创建者 |
| created_at | timestamp | — |

### 4.3 ModelUsageRecord（新建表）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | uuid | 主键 |
| request_id | string | 请求 ID |
| tenant_id | uuid | 租户 |
| api_key_id | uuid | API Key |
| model_name | string | 模型名 |
| deployment_id | uuid | 部署 |
| instance_id | uuid | 实例 |
| user_id | uuid | 用户（如果 API Key 关联用户） |
| prompt_tokens | int | 输入 token |
| completion_tokens | int | 输出 token |
| total_tokens | int | 总 token |
| latency_ms | int | 总耗时 |
| queue_time_ms | int | 排队时间 |
| status_code | int | HTTP 状态码 |
| error_code | string | 错误码 |
| cost | decimal | 费用（Phase 4 固定为 0，后续启用） |
| created_at | timestamp | — |

## 5. Gateway 路由

Gateway 作为独立 HTTP handler，监听独立端口（如 `18081`）或挂在 Server mux 的 `/v1/*` 路径。

```text
GET  /v1/models                      → 返回可用模型列表
POST /v1/chat/completions            → 代理到目标实例
POST /v1/completions                 → 代理到目标实例
POST /v1/embeddings                  → 代理到目标实例（预留）
```

请求流程：
1. 解析 `Authorization: Bearer <api_key>`
2. 校验 API Key（存在、active、未过期、限流未超）
3. 从 request body 提取 model name
4. 查找 ModelRoute（model_name → instance endpoint）
5. 代理请求到目标实例
6. 解析响应中的 `usage`（prompt_tokens/completion_tokens/total_tokens）
7. 写入 ModelUsageRecord
8. 返回响应给客户端

## 6. API Key API

```text
GET    /api/api-keys                  # 列表（脱敏：只显示 prefix + 状态）
POST   /api/api-keys                  # 创建（返回完整 key 原文，仅此一次）
DELETE /api/api-keys/{id}             # 禁用/删除
```

## 7. 权限

新增 permission code：`apikey:read`、`apikey:write`。

## 8. 测试要求

- API Key 创建/校验/禁用/过期 round-trip
- Gateway `/v1/chat/completions` 代理到实例
- 无效 API Key 返回 401
- 限流超限返回 429
- ModelUsageRecord 正确写入
- 实例不可用时 Gateway 返回 503
- 多租户隔离（租户 A 的 Key 不能调用租户 B 的模型）

## 9. 验收标准

```bash
# 创建 API Key
curl -X POST /api/api-keys -H 'Cookie: ...' \
  -d '{"name":"my-key"}'
# → {"id":"...","key":"sk-abc123...","prefix":"sk-abc","status":"active"}

# 通过 Gateway 调用模型
curl -X POST http://localhost:18081/v1/chat/completions \
  -H 'Authorization: Bearer sk-abc123...' \
  -d '{"model":"qwen3-32b","messages":[{"role":"user","content":"hello"}]}'
# → {"choices":[...],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}

# 无效 Key
curl -X POST http://localhost:18081/v1/chat/completions \
  -H 'Authorization: Bearer invalid-key' \
  -d '{"model":"qwen3-32b","messages":[...]}'
# → 401 {"error":"invalid api key"}

# 查看 usage
curl /api/model-usage?tenant_id=... -H 'Cookie: ...'
# → [{...}]
```

## 10. 风险点

- Gateway 代理时如果实例响应非 OpenAI 格式（如 vLLM vs SGLang 的 usage 字段差异），需要做格式归一化
- 实例 endpoint 变更时（重启后端口可能变化），ModelRoute 需要同步更新
- API Key 哈希存储需要选择算法（bcrypt 或 sha256），与现有 password 哈希保持一致
- 限流实现如果基于内存，Server 重启后丢失；如果基于 DB，写入开销大
