# LightAI Go 认证与租户设计

## 1. 设计目标

LightAI Go 第一阶段正式实现基础认证、Tenant 和 RBAC 底座，主要用于：

1. 支持本地用户登录和当前用户上下文；
2. 支持全局 User 与 TenantMembership；
3. 支持一个 Membership 绑定多个 Role；
4. 支持系统内置角色和租户自定义角色；
5. 支持统一按 Permission code 进行 API 授权；
6. 支持资源 tenant、owner 和审计归属；
7. 支持后续 API Key、Token 统计、额度和成本核算；
8. 避免 RuntimeEnvironment、Model、ModelInstance、AgentTask 等核心对象后续大改表结构；
9. 保持 Agent token、User Session 和 Future API Key 的凭证边界。

第一阶段实现的是可执行的 RBAC 基础模型，不是完整企业级多租户系统。

---

## 2. 第一阶段范围

第一阶段实现：

1. default tenant；
2. 全局本地 User；
3. TenantMembership；
4. TenantMembershipRole；
5. 全局只读 built-in Role；
6. tenant custom Role；
7. 系统只读 Permission catalog；
8. RolePermission；
9. bootstrap platform admin；
10. server-side Session；
11. 登录、退出、修改密码和当前用户查询；
12. current tenant 上下文；
13. 实时 roles / permissions 解析；
14. platform admin 与 tenant admin 边界；
15. tenant scope 和 permission code 授权；
16. 资源 tenant_id / owner_id / created_by / updated_by；
17. 基础审计日志；
18. CSRF、Origin、Argon2id 和登录限流；
19. Agent token 与用户认证隔离。

---

## 3. 第一阶段暂不实现

第一阶段不实现：

1. API Key；
2. Token usage；
3. 额度；
4. 计费；
5. 用户自定义 Permission code；
6. 资源级 ACL；
7. 字段级权限；
8. 多级组织继承；
9. 项目 / Workspace；
10. SSO；
11. LDAP；
12. OAuth；
13. 企业微信登录；
14. 租户级 GPU 配额；
15. 跨租户资源共享；
16. 租户隔离调度；
17. 租户账单；
18. 租户隔离网络策略；
19. 用户邀请流程；
20. tenant switch API / UI。

custom Role 只能组合系统 Permission catalog 中已有的 permission code，不等于允许用户定义新的权限点。

---

## 4. Tenant 数据模型

```go
type Tenant struct {
    ID        string
    Name      string
    Status    string // active / disabled
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

第一阶段默认创建：

```text
tenant_id = default
tenant_name = Default Tenant
```

规则：

1. 单客户部署可以只有 default tenant；
2. 后续可以按部门、业务线或项目创建多个 Tenant；
3. `status=disabled` 的 Tenant 不能建立新的用户 Session，也不能执行资源操作；
4. Tenant 使用禁用优先，第一阶段不硬删除；
5. 第一阶段不做租户配额和租户账单；
6. default tenant 是普通 Tenant 记录，不能通过特殊实现阻止未来增加其他 Tenant。

创建 Tenant 时应同时指定一个 active 全局 User 作为首个 tenant admin。Server 在同一事务中创建 Tenant、TenantMembership 和 built-in admin Role 绑定，避免产生没有管理员的 active Tenant。

---

## 5. User 数据模型

```go
type User struct {
    ID                 string
    Username           string
    DisplayName        string
    PasswordHash       string
    Status             string // active / disabled
    IsPlatformAdmin    bool
    MustChangePassword bool
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

规则：

1. User 是全局账号，不从属于某一个 Tenant；
2. username 全局唯一；
3. password 只保存 Argon2id hash；
4. 不保存或记录明文密码；
5. `status=disabled` 的 User 不能登录；
6. disabled User 的现有 Session 在下一次请求时必须失效；
7. User 使用禁用优先，第一阶段不硬删除；
8. 第一阶段只支持本地用户；
9. `is_platform_admin` 用于跨租户 User / Tenant 管理，不等同于任何 tenant Role；
10. 只有 platform admin 可以创建、禁用、重置全局 User 或修改 `is_platform_admin`。

系统必须至少保留一个 active platform admin。任何会导致 active platform admin 数量变为 0 的禁用或撤权操作必须拒绝。

---

## 6. TenantMembership 数据模型

```go
type TenantMembership struct {
    ID        string
    TenantID  string
    UserID    string
    Status    string // active / disabled
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

唯一约束：

```text
tenant_id + user_id
```

规则：

1. TenantMembership 不保存角色字段；
2. 一个 User 可以属于一个或多个 Tenant；
3. User 在同一 Tenant 只有一个 Membership；
4. Membership 使用禁用优先，第一阶段不硬删除；
5. disabled Membership 只禁止该 User 进入对应 Tenant；
6. active Membership 至少绑定一个 active Role；
7. tenant admin 只能管理 `session.current_tenant_id` 下的 Membership；
8. tenant admin 只能把已存在且 active 的全局 User 加入当前 Tenant；
9. tenant admin 不能创建、禁用或删除全局 User；
10. 第一阶段不实现 invitation 流程。

---

## 7. Role、Permission 与绑定模型

### 7.1 Role

```go
type Role struct {
    ID          string
    TenantID    *string
    Name        string
    DisplayName string
    Description string
    BuiltIn     bool
    Status      string // active / disabled
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

唯一约束：

```text
tenant_id + name
```

built-in Role：

```text
admin
operator
viewer
```

规则：

1. built-in Role 是全局记录，`tenant_id=null`；
2. built-in Role 由系统初始化和版本迁移维护；
3. built-in Role、名称和 RolePermission 对用户只读；
4. built-in Role 不可禁用、修改或删除；
5. built-in admin 表示 tenant admin，不等于 platform admin；
6. custom Role 必须设置 tenant_id；
7. custom Role 只在对应 Tenant 内可见、可管理和可分配；
8. tenant admin 可以管理当前 Tenant 的 custom Role；
9. custom Role 未被 TenantMembershipRole 引用时可以删除；
10. custom Role 已被分配时必须先解绑，不能直接删除；
11. custom Role 只能绑定系统 Permission catalog 中已有的 tenant-scope Permission；
12. custom Role 不能获得 platform-scope Permission；
13. custom Role 使用禁用优先；disabled Role 不产生 Permission。

### 7.2 Permission

```go
type Permission struct {
    ID          string
    Code        string
    Scope       string // tenant / platform
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Permission catalog 是系统内置、全局只读数据：

1. permission code 由代码版本和数据库迁移维护；
2. 用户不能创建、修改或删除 Permission；
3. 所有用户侧受保护 API 必须声明 required permission code；
4. custom Role 只能选择 catalog 中已有的 tenant-scope Permission；
5. platform-scope Permission 只由 `is_platform_admin` 产生，不能绑定给 tenant Role。

第一阶段至少定义：

```text
dashboard:read
node:read
gpu:read
monitoring:read
log:read

runtime:read
runtime:write
model:read
model:write
instance:read
instance:write
instance:operate
task:read

membership:read
membership:write
role:read
role:write
tenant:settings:write

platform:user:manage
platform:tenant:manage
platform:settings:write
```

### 7.3 RolePermission

```go
type RolePermission struct {
    ID           string
    RoleID       string
    PermissionID string
    CreatedAt    time.Time
}
```

唯一约束：

```text
role_id + permission_id
```

规则：

1. built-in RolePermission 由系统初始化和版本迁移维护；
2. tenant admin 不能修改 built-in RolePermission；
3. tenant admin 可以管理当前 Tenant custom Role 的 RolePermission；
4. 只能绑定 Permission catalog 中存在的 Permission；
5. custom Role 不能绑定 platform-scope Permission。

### 7.4 TenantMembershipRole

```go
type TenantMembershipRole struct {
    ID           string
    MembershipID string
    RoleID       string
    CreatedAt    time.Time
}
```

唯一约束：

```text
membership_id + role_id
```

规则：

1. 一个 Membership 可以绑定多个 Role；
2. 可以绑定 `tenant_id=null` 的 built-in Role；
3. 可以绑定 `role.tenant_id=membership.tenant_id` 的 custom Role；
4. 不允许绑定其他 Tenant 的 custom Role；
5. disabled Role 不产生 Permission；
6. active Membership 至少保留一个 active Role；
7. 移除最后一个 active Role 时拒绝；如需移除访问应先禁用 Membership。

---

## 8. Built-in Role 默认权限

built-in viewer：

```text
dashboard:read
node:read
gpu:read
monitoring:read
log:read
runtime:read
model:read
instance:read
task:read
```

built-in operator 包含 viewer 权限，并增加：

```text
runtime:write
model:write
instance:write
instance:operate
```

built-in admin 包含 operator 权限，并增加：

```text
membership:read
membership:write
role:read
role:write
tenant:settings:write
```

built-in admin 只管理当前 Tenant，不包含：

```text
platform:user:manage
platform:tenant:manage
platform:settings:write
```

---

## 9. Platform Admin 与 Tenant Admin

### 9.1 Platform Admin

`User.IsPlatformAdmin=true` 表示 platform admin。

platform admin 可以：

1. 查询全部 Tenant；
2. 创建、修改和禁用 Tenant；
3. 创建、修改、禁用和重置全局 User；
4. 授予或撤销 `is_platform_admin`；
5. 为新 Tenant 初始化首个 tenant admin；
6. 管理平台级系统配置。

授权解析时，active platform admin 获得以下 platform-scope Permission：

```text
platform:user:manage
platform:tenant:manage
platform:settings:write
```

platform API 仍统一声明 required permission code，同时必须确认 `is_platform_admin=true`。

### 9.2 Tenant Admin

Tenant admin 是当前 Membership 通过 RolePermission 获得 tenant 管理 permission 的用户。通常来自 built-in admin，也可以来自 custom Role。

tenant admin：

1. 只管理当前 Tenant 的 Membership；
2. 只管理当前 Tenant 的 custom Role；
3. 可以分配 built-in Role 和当前 Tenant custom Role；
4. 只能操作当前 Tenant 的业务资源；
5. 不能查看或操作其他 Tenant；
6. 不能创建、禁用或重置全局 User；
7. 不能授予 `is_platform_admin`；
8. built-in admin Role 不自动获得 platform admin 能力。

---

## 10. Bootstrap Admin

Server 首次启动必须幂等初始化：

1. Permission catalog；
2. built-in admin/operator/viewer Role；
3. built-in RolePermission；
4. default tenant；
5. bootstrap User；
6. bootstrap User 的 default TenantMembership；
7. Membership 与 built-in admin Role 的 TenantMembershipRole；
8. `is_platform_admin=true`；
9. `must_change_password`。

配置示例：

```yaml
auth:
  enabled: true
  session:
    idle_timeout_hours: 12
    refresh_window_hours: 6
    cookie_name: "lightai_session"
    secure_cookie: false
  bootstrap_admin:
    username: "admin"
    password: ""
    password_env: "LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD"
    force_change_password: true
```

规则：

1. 初始密码优先来自 `password_env`，其次来自明确配置；
2. 两者都为空时生成随机高强度初始密码；
3. 自动生成密码只在首次创建 bootstrap User 时输出一次；
4. 后续启动不得再次输出；
5. 生产环境建议强制修改初始密码；
6. bootstrap 流程不能重复创建 Tenant、User、Membership、Role 或绑定记录；
7. 强制改密用户只允许访问 `/api/auth/me`、`/api/auth/change-password` 和 `/api/auth/logout`。

---

## 11. Session 与认证接口

第一阶段采用：

```text
server-side session + HTTP-only cookie
```

### 11.1 Session

```go
type Session struct {
    ID              string
    UserID          string
    CurrentTenantID string
    CSRFSecretHash  string
    CreatedAt       time.Time
    LastSeenAt      time.Time
    ExpiresAt       time.Time
    RevokedAt       *time.Time
}
```

Session 只保存 user_id、current_tenant_id 和过期/撤销等最小上下文，不保存 roles 或 permissions 快照。

规则：

1. Session ID 使用高熵随机值；
2. 数据库存储 Session ID hash；
3. Cookie 使用 `HttpOnly`、`SameSite=Lax`，HTTPS 部署启用 `Secure`；
4. Session 使用 12 小时滑动过期；
5. logout 后立即撤销 Session；
6. 每次请求实时解析 User、Tenant、Membership、Roles 和 Permissions；
7. User、Tenant、Membership 或 Role 禁用后，下一次请求立即生效。

### 11.2 登录

```http
POST /api/auth/login
```

请求：

```json
{
  "username": "admin",
  "password": "******",
  "tenant_id": "default"
}
```

规则：

1. tenant_id 可选；
2. User 只有一个 active Membership 时自动选择；
3. 指定 tenant_id 时必须校验 active Membership；
4. 多个 active Membership 且未指定 tenant_id 时返回 `409 tenant_selection_required`；
5. User、Tenant 或 Membership disabled 时拒绝；
6. 登录失败统一返回无效凭据；
7. 按 username 和请求来源执行基础限流；
8. login 尚无 Session CSRF token，必须严格校验 Origin 且仅接受 JSON。

### 11.3 当前用户

```http
GET /api/auth/me
```

返回示例：

```json
{
  "user": {
    "id": "user-001",
    "username": "admin",
    "display_name": "Administrator",
    "is_platform_admin": true,
    "must_change_password": false
  },
  "tenant": {
    "id": "default",
    "name": "Default Tenant"
  },
  "roles": [
    {
      "id": "role-admin",
      "name": "admin",
      "built_in": true
    }
  ],
  "permissions": [
    "runtime:read",
    "runtime:write",
    "membership:write",
    "platform:user:manage"
  ]
}
```

roles 和 permissions 是本次请求实时解析结果。

### 11.4 退出

```http
POST /api/auth/logout
```

Server 撤销当前 Session 并清除 Cookie。

### 11.5 修改密码

```http
POST /api/auth/change-password
```

规则：

1. 需要有效 Session 和 CSRF token；
2. 请求包含当前密码和新密码；
3. bootstrap 首次改密后清除 `must_change_password`；
4. 修改密码后撤销该 User 的其他 Session；
5. 当前 Session 可以保留，但必须刷新 CSRF secret；
6. 不记录明文密码。

---

## 12. 实时授权解析

每个用户请求按顺序解析：

1. Session；
2. active User；
3. active Tenant；
4. active TenantMembership；
5. TenantMembershipRole；
6. active built-in / current Tenant custom Role；
7. RolePermission；
8. Permission catalog；
9. `is_platform_admin` 产生的 platform-scope Permission。

request context 至少包含：

```text
current_user_id
current_tenant_id
is_platform_admin
current_role_ids
current_permission_codes
```

API 授权规则：

1. API 必须声明 required permission code；
2. required permission 必须存在于 `current_permission_codes`；
3. platform API 还必须确认 `is_platform_admin=true`；
4. tenant resource 必须属于 `current_tenant_id`；
5. API 不比较 `admin/operator/viewer` 名称；
6. 一个 Membership 的多个 Role 权限取并集；
7. Role、RolePermission 或 Membership 变化在下一次请求立即生效；
8. 后续可以增加短 TTL cache，但不能改变实时撤权语义。

---

## 13. CSRF、密码和登录安全

### 13.1 CSRF

Cookie Session 的用户侧写请求必须：

1. 携带 `X-CSRF-Token`；
2. 通过 Session CSRF secret 校验；
3. 通过 Origin 校验；
4. login 成功时提供前端可读取的 CSRF token；
5. logout 和所有 POST/PUT/PATCH/DELETE 管理 API 都执行 CSRF 校验；
6. login 没有既有 CSRF token，必须执行严格 Origin 校验和登录限流；
7. Agent API 不使用 Cookie，不参与 CSRF 校验。

### 13.2 Password

1. 使用 Argon2id；
2. hash 包含 salt 和参数；
3. 参数可配置并提供安全默认值；
4. 不记录密码、hash 或完整登录请求；
5. 登录失败返回统一错误；
6. 按 username 和来源限制连续失败；
7. 密码重置撤销该 User 的现有 Session；
8. User disabled 时现有 Session 下一次请求立即拒绝。

---

## 14. Agent Token 与用户认证边界

必须保持：

```text
Agent bootstrap/shared token != User Session != Future API Key
```

| 凭证 | 用途 | 身份上下文 |
| --- | --- | --- |
| Agent bootstrap/shared token | Agent 注册、心跳、资源上报、任务 claim 和状态回报 | agent_id / node_id |
| User Session | Web/API 管理操作 | user_id / tenant_id / roles / permissions |
| Future API Key | 后续模型服务调用 | tenant_id / user_id / api_key_id |

要求：

1. Agent 请求不使用 User Session；
2. 用户请求不使用 Agent token；
3. Future API Key 后续只用于模型服务网关；
4. Agent token 不产生 user_id；
5. Agent 创建或更新系统发现资源时 actor 为 `system`；
6. Server 创建 AgentTask 前已完成 Session、tenant scope 和 permission code 校验；
7. Agent 不解析 User、Role 或 Permission，只校验任务属于本节点并执行任务契约。

---

## 15. 核心资源归属字段

### 15.1 用户业务资源

RuntimeEnvironment、Model 和 ModelInstance：

```go
TenantID  string
OwnerID   string
CreatedBy string
UpdatedBy string
```

用户创建时默认：

```text
tenant_id = session.current_tenant_id
owner_id = session.current_user_id
created_by = session.current_user_id
updated_by = session.current_user_id
```

### 15.2 AgentTask

```go
TenantID  string
CreatedBy string
UpdatedBy string
```

AgentTask 不需要业务 OwnerID。tenant_id 来自关联实例或已授权操作，created_by 为触发用户或 `system`。

### 15.3 Node

```go
TenantID  string
OwnerID   *string
CreatedBy string
UpdatedBy string
```

第一阶段 Agent 注册新 Node：

```text
tenant_id = default
owner_id = null
created_by = system
updated_by = system
```

### 15.4 GPUDevice

```go
TenantID  string
OwnerID   *string
CreatedBy string
UpdatedBy string
```

GPUDevice 跟随 Node：

```text
tenant_id = node.tenant_id
owner_id = null
created_by = system
updated_by = system
```

### 15.5 Owner 与审计语义

1. owner_id 是可转移的当前业务所有者；
2. owner_id 不等于 created_by；
3. owner_id 不作为 ACL，不直接授予权限；
4. 授权仍由 tenant scope 和 permission code 决定；
5. owner 只能转移给同 Tenant 的 active User Membership；
6. owner 转移不修改 created_by；
7. created_by 表示最初创建 actor，创建后不修改；
8. updated_by 表示最近修改 actor；
9. `system` 是保留 actor；
10. 客户端不能覆盖 tenant、owner 或审计字段；
11. 跨 Tenant 资源引用和操作必须拒绝。

第一阶段可以不实现 owner 转移 API，但数据模型和语义必须固定。

---

## 16. Permission 与 API 要求

### 16.1 公开或部署边界控制

```http
GET /healthz
GET /metrics
```

### 16.2 认证接口

```http
POST /api/auth/login
POST /api/auth/logout
POST /api/auth/change-password
GET  /api/auth/me
```

### 16.3 资源 API

资源读 API 要求对应 `*:read` permission，并按 current Tenant 过滤。

资源写 API 要求对应 `*:write` 或 `instance:operate` permission。

### 16.4 Platform Admin API

```http
GET  /api/users
POST /api/users
GET  /api/users/{id}
PUT  /api/users/{id}
POST /api/users/{id}/disable
POST /api/users/{id}/reset-password
POST /api/users/{id}/set-platform-admin

GET  /api/tenants
POST /api/tenants
GET  /api/tenants/{id}
PUT  /api/tenants/{id}
POST /api/tenants/{id}/disable
```

这些 API 要求对应 platform permission 且 `is_platform_admin=true`。

### 16.5 Tenant Admin API

```http
GET  /api/tenant-memberships
POST /api/tenant-memberships
GET  /api/tenant-memberships/{id}
PUT  /api/tenant-memberships/{id}
POST /api/tenant-memberships/{id}/disable
POST /api/tenant-memberships/{id}/roles
DELETE /api/tenant-memberships/{id}/roles/{role_id}

GET  /api/roles
POST /api/roles
GET  /api/roles/{id}
PUT  /api/roles/{id}
DELETE /api/roles/{id}
PUT  /api/roles/{id}/permissions

GET  /api/permissions
```

规则：

1. 所有操作限定 `session.current_tenant_id`；
2. `GET /api/roles` 返回全局 built-in Role 和当前 Tenant custom Role；
3. Permission catalog 只读；
4. tenant admin 只能把已有 active User 加入当前 Tenant；
5. 创建 Membership 时至少绑定一个合法 Role；
6. 客户端提交的 tenant、owner 或审计字段必须忽略或拒绝。

### 16.6 Agent API

```http
POST /api/agent/register
POST /api/agent/heartbeat
POST /api/agent/resources/report
POST /api/agent/tasks/claim
POST /api/agent/tasks/report
POST /api/agent/instances/report
```

这些 API 只使用 bootstrap/shared agent token，不使用 Cookie Session 或 RBAC。

---

## 17. 审计字段与操作日志

核心表建议包含：

```go
CreatedBy string
UpdatedBy string
CreatedAt time.Time
UpdatedAt time.Time
```

关键操作日志记录：

1. user_id；
2. tenant_id；
3. is_platform_admin；
4. role_ids；
5. required_permission；
6. operation；
7. resource_type；
8. resource_id；
9. result；
10. error；
11. timestamp。

系统操作没有 user_id 时，actor 使用 `system`，并记录 agent_id 或 node_id。

第一阶段可以先写 Server 结构化日志，不要求建立 audit_log 表。后续可以扩展独立 audit_log 表。

日志不得记录：

1. 明文密码；
2. PasswordHash；
3. Session ID；
4. CSRF secret；
5. Agent token；
6. Future API Key 明文。

---

## 18. 后续 API Key / Token / 计费扩展

第一阶段只预留归属字段，不实现以下能力。

### 18.1 APIKey

```go
type APIKey struct {
    ID         string
    TenantID   string
    UserID     string
    Name       string
    KeyHash    string
    Status     string
    CreatedAt  time.Time
    LastUsedAt *time.Time
}
```

### 18.2 UsageRecord

```go
type UsageRecord struct {
    ID               string
    TenantID         string
    UserID           string
    APIKeyID         string
    ModelID          string
    ModelInstanceID  string
    PromptTokens     int64
    CompletionTokens int64
    TotalTokens      int64
    StartedAt        time.Time
    FinishedAt       time.Time
}
```

后续再考虑 tenant quota、model quota、token quota、cost policy、billing period、tenant bill 和 project/workspace。

第一阶段的 TenantID、OwnerID、CreatedBy 和 API 身份上下文将作为后续模型访问、Token usage 和成本聚合基础。

---

## 19. 测试与验收建议

第一阶段至少覆盖：

1. Permission catalog 初始化幂等；
2. built-in Role / RolePermission 初始化幂等；
3. default Tenant 和 bootstrap platform admin 初始化幂等；
4. 自动生成初始密码只输出一次；
5. Argon2id 验证和统一登录错误；
6. 登录限流、Origin 和 CSRF 校验；
7. 12 小时滑动 Session；
8. User、Tenant、Membership disabled 立即拒绝；
9. 单 Membership 自动选择 Tenant；
10. 多 Membership 返回 tenant_selection_required；
11. 一个 Membership 多 Role 权限取并集；
12. built-in Role 只读且不可删除；
13. custom Role 只能管理当前 Tenant；
14. custom Role 不能绑定 platform Permission；
15. 被分配 custom Role 删除失败，解绑后可删除；
16. active Membership 最后一个 Role 不能移除；
17. RolePermission 变更在下一次请求生效；
18. API 统一按 required permission code 授权；
19. built-in admin 不获得 platform admin 权限；
20. tenant admin 不能创建全局 User；
21. platform admin 可以管理全局 User 和 Tenant；
22. 最后一个 active platform admin 不能被撤销；
23. 所有资源查询按 current Tenant 过滤；
24. 创建资源时归属字段由 Server 写入；
25. owner_id 不作为 ACL；
26. Node / GPU owner_id=null 且 created_by=system；
27. Agent token 不能调用用户管理 API；
28. User Session 不能调用 Agent API；
29. AgentTask 创建前由 Server 完成 permission 校验；
30. Agent 不解析用户 RBAC。

---

## 13. Tenant 数据模型（更新）

Tenant 表结构：

```sql
CREATE TABLE tenants (
    id TEXT PRIMARY KEY,       -- UUID, e.g., a0000000-0000-0000-0000-000000000001
    slug TEXT NOT NULL,        -- 短标识，如 'default', 'tenant-a'
    name TEXT NOT NULL,        -- 显示名称，如 'Default Tenant'
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT,
    updated_at TEXT
);
```

default tenant：
- id = `a0000000-0000-0000-0000-000000000001`（确定性 UUID）
- slug = `'default'`
- name = `'Default Tenant'`

规则：
- tenant_id 字段必须是 UUID，严禁写入 'default' 字符串。
- 查询 default tenant 使用 `slug = 'default'`。
- Agent 注册时写入 `nodes.tenant_id = default tenant UUID`。

---

## 14. Node 租户归属（更新）

Agent 首次注册新 Node：
- Server 写入 `nodes.tenant_id = default tenant UUID`。
- Agent 不允许指定租户。

Agent 重新注册：
- 不覆盖已有 `nodes.tenant_id`。
- 只更新运行状态字段（hostname, metrics, status 等）。

节点租户转移：
- `PATCH /api/nodes/{id}/tenant`。
- platform_admin 可转移任意节点。
- 租户内拥有 `node:transfer` 权限的用户可转出本租户节点。
- 转移写入 `audit_logs` 表。

---

## 15. Permissions（更新）

最小权限点：
- `node:read` — 查看节点（viewer, operator, admin）
- `node:transfer` — 转移节点租户（admin）

built-in admin 拥有 `node:transfer`。operator/viewer 默认不拥有。

---

## 16. 后续扩展：API Key / Token / 计费预留

以下为文档预留，本轮不实现：

1. **API Key 认证**：未来支持创建 API Key，绑定 Tenant 和 Role，用于程序化访问。
2. **Token 用量统计**：未来 UsageRecord 记录每次请求的 token 消费。
3. **跨租户资源共享**：GPUDevice 可从 Node 继承 tenant_id，但未来支持独立分配。
4. **计费与额度**：未来支持租户级 GPU 配额、优先级、限流、计费策略。
5. **Resource Owner vs Consumer**：
   - Tenant 可以是资源拥有者（拥有 GPUDevice、ModelInstance、Endpoint）。
   - Tenant 也可以是资源使用者（通过 API Key 访问被授权的 Endpoint）。
   - UsageRecord 可区分 resource_owner_tenant、consumer_tenant、billing_tenant。
6. **Endpoint 共享**：未来支持跨租户共享推理 Endpoint。

本轮不实现：
- API Key
- Token usage 统计
- 额度/计费/账单
- 跨租户资源共享
- 租户级 GPU 配额
- GPUDevice 独立分配 UI
- 限流策略
