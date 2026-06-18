> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference or historical compatibility document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go 项目范围说明

## 1. 项目定位

LightAI Go 是一个参考 GPUStack 架构思想、面向中小型客户的轻量 AI 资源与模型服务管理平台。

平台主要用于管理数台 GPU 服务器，提供 GPU 资源监控、模型定义管理、运行环境管理、模型实例启停、实例状态查看、服务入口展示和现场排障能力。

GPUStack 仅作为架构参考，不复制代码，不逐行翻译，不追求第一阶段完整复刻。

## 2. 第一阶段核心目标

第一阶段优先完成身份、资源与实例管理闭环：

1. Server / Agent 基础架构；
2. Agent 注册与心跳；
3. 节点在线 / 离线状态；
4. GPU 资源发现；
5. GPU 指标采集；
6. GPU 监控页面；
7. 运行环境管理；
8. 模型定义管理；
9. 模型实例创建；
10. 模型实例启动、停止、重启；
11. 实例状态、日志、错误信息可见；
12. 模型服务 endpoint 可查看、可复制、可检查；
13. 初始化 default tenant 和 bootstrap platform admin；
14. 支持本地用户和 TenantMembership；
15. 支持 built-in admin/operator/viewer、租户 custom Role 和 Permission catalog；
16. 支持基础登录、退出和当前用户查询；
17. 支持 TenantMembershipRole、RolePermission 和统一 permission code 授权；
18. 核心资源从第一阶段写入 tenant、owner 和审计字段。

第一阶段的目标是让客户现场能够清楚看到 GPU 资源，能够启动模型，能够知道模型跑在哪台机器、用了哪些 GPU、服务地址是什么、失败原因是什么。

## 3. 第一阶段暂不实现

第一阶段不实现以下能力：

1. Kubernetes；
2. Ray；
3. 多集群；
4. 复杂自动调度；
5. 模型市场；
6. 自动下载模型；
7. API Key 管理；
8. Token 统计；
9. 额度管理；
10. 成本核算；
11. 复杂统一网关；
12. 用户自定义 Permission code、资源级 ACL、字段级权限和多级组织继承；
13. 高可用控制面。

这些能力放到后续阶段，不允许提前污染第一阶段架构。

第一阶段正式加入 Tenant、User、Membership、Role、Permission、RolePermission、TenantMembershipRole、Session、自定义角色管理和资源归属底座，但不实现 API Key、Token、额度、计费、SSO、用户自定义 Permission code、资源级 ACL、租户配额或账单。详细边界见 `docs/09-auth-tenant-design.md`。

## 4. 优先级原则

第一优先级：

* 节点接入；
* 资源监控；
* GPU 状态；
* 运行环境；
* 模型定义；
* 实例启停；
* 实例状态；
* 排障日志；
* 基础认证、租户、RBAC 和审计。

第二优先级：

* 简单服务入口；
* endpoint 展示；
* 健康检查；
* 启动命令快照；
* Docker 日志查看。

第三优先级：

* API Key；
* 请求代理；
* Token 统计；
* 额度；
* 成本。

## 5. 设计原则

1. 轻量优先，不引入重型依赖；
2. Server / Agent 架构优先；
3. 先手工选择节点和 GPU，再考虑自动调度；
4. 先 Docker 启动模型，再考虑更多运行方式；
5. 所有高级参数都必须显式启用，未启用不出现在命令中；
6. Agent 失败不能影响 Server；
7. GPU 采集失败不能导致 Agent 崩溃；
8. 所有关键操作必须记录日志；
9. 现场问题必须能通过页面、API、日志定位；
10. 不过早抽象，但模块边界要清楚。

## 6. 第一阶段成功标准

第一阶段完成后，应达到以下效果：

1. Server 可以启动；
2. Agent 可以启动；
3. 多台 Agent 可以注册到 Server；
4. Web 可以看到节点列表；
5. Web 可以看到每台节点 GPU 状态；
6. GPU 显存、利用率、温度、功耗等信息可见；
7. 可以配置 Docker 运行环境；
8. 可以创建模型定义；
9. 可以创建模型实例；
10. 可以选择节点和 GPU 启动实例；
11. 可以停止和重启实例；
12. 可以看到实例 endpoint；
13. 可以看到实例最近错误和启动命令；
14. 容器异常退出后页面能体现状态变化；
15. 现场人员可以根据日志排查问题；
16. default tenant 和 bootstrap platform admin 可初始化；
17. 用户登录后只能访问当前 tenant 范围内的资源；
18. built-in Role、custom Role、platform admin 和 tenant admin 边界清晰；
19. API 统一按 permission code 授权；
20. Agent token 与 User Session 完全隔离。
