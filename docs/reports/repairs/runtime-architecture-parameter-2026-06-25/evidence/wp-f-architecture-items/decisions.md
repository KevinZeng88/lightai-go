# Architecture Decisions — WP-F

> Date: 2026-06-25

## RAP-006: extra_args 冲突检测策略

**Decision:** KEEP AS WARNING (DEFERRED)
**Reason:** 阻断 arg 冲突可能阻止合法的参数覆盖（用户在 deployment 层有意覆盖 NBR 参数）。当前 `deduplicateArgs()` WARNING log 已足够。
**Risk:** 重复参数可能导致 Docker 容器启动失败或使用非预期值。
**Trigger:** 用户报告因重复参数导致的部署问题，且反馈需要 preflight 阻断。
**Target:** 后续 minor release 中评估升级为 preflight structured warning。

## RAP-007: DeviceBinding dead struct

**Decision:** DELETE (YAGNI)
**Reason:** `types.go:78-102` 定义了完整的 `DeviceBinding` struct，但 zero usage throughout codebase。GPU/vendor binding 当前通过 `docker.go:481-498` 的 `spec.Vendor` string check 实现，不需要中间抽象。
**Risk:** 如果未来需要跨 vendor 统一 binding 接口，可能需要重新引入。
**Trigger:** 引入自动调度或 GPU capability matching 时重新评估。

## RAP-008: RuntimeRequirements / BackendCapabilityProfile

**Decision:** DEFERRED_WITH_REASON
**Reason:** 设计文档 `03-core-abstractions-v2.md` 定义了完整的 `RuntimeRequirements` 和 `BackendCapabilityProfile`，但当前 preflight 使用 ad-hoc 检查（`DockerSpecInfo` + `BackendDescriptor`）覆盖了关键场景。完整实现需要：
- 定义 Go struct（需要 MetaX/Huawei 硬件测试环境验证）
- 接入 preflight validation
- 与 catalog schema 对齐
**Risk:** vendor capability 匹配不完全，未来添加新 vendor 时需要手动扩展检查。
**Trigger:** 引入自动调度或 GPU capability matching 时实现。

## RAP-013: Evidence 目录索引

**Decision:** UPDATED
**Action:** 在 `docs/reports/repairs/runtime-architecture-parameter-2026-06-25/evidence/` 下已建立统一证据目录结构，按 WP 子目录组织。后续 E2E 输出也应归档到此结构。

## RAP-014: Hardcoded Docker 参数列表

**Decision:** DEFERRED_WITH_REASON
**Reason:** `RuntimeParameterEditor.vue:162-181` 的 `scalarOptions` (6 项) 和 `listOptions` (9 项) 是 hardcoded。迁移到 schema/catalog 驱动需要：
- 在 BackendRuntime / BackendVersion schema 中定义 Docker 参数组
- 修改 `RuntimeParameterEditor` 从 schema 动态渲染
- 验证所有现有页面和保存逻辑
**Risk:** 新增 Docker 参数（如 `userns_mode`）需要修改前端代码。
**Trigger:** 需要新增 Docker 参数时，应优先考虑迁移到 schema 驱动。
**Estimated cost:** ~3-4h (medium complexity)
