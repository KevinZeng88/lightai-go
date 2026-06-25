# Architecture Decisions — WP-F

> Date: 2026-06-25

## RAP-006: extra_args 冲突检测策略

**Decision:** KEEP AS WARNING (DEFERRED)
**Verified:** `resolver.go:616` uses `log.Warn("runplan.deduplicate_args_conflict", ...)` — confirmed warning-only, no structured preflight error.
**Reason:** 阻断 arg 冲突可能阻止合法的参数覆盖（用户在 deployment 层有意覆盖 NBR 参数）。冲突信息可通过 server log 查看。
**Risk:** 重复参数可能导致 Docker 容器启动失败或使用非预期值。
**Trigger:** 用户报告因重复参数导致的部署问题，且反馈需要 preflight 阻断。
**Target:** 后续 minor release 中评估升级为 preflight structured warning。

## RAP-007: DeviceBinding dead struct

**Decision:** FIXED — DELETED (2026-06-25)
**Action:** Removed `DeviceBinding` struct (was ~26 lines) and its field from `DockerSpec` in `internal/server/runplan/types.go`. Confirmed zero usage across entire codebase via grep. GPU/vendor binding is handled directly in `agent/runtime/docker.go:481-498` via `spec.Vendor` string check.
**Verification:** `go test ./internal/server/runplan/...` PASS.

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
**Impact Assessment:**
- `scalarOptions` (6 params): privileged, ipc_mode, uts_mode, network_mode, pid_mode, shm_size
- `listOptions` (9 params): devices, optional_devices, group_add, security_options, cap_add, device_cgroup_rules, extra_hosts, ulimits, extra_mounts
- **Coverage:** Current 15 hardcoded params cover all Docker options used by existing NVIDIA/MetaX/Huawei catalog profiles
- **Gap:** Adding a new Docker param (e.g. `userns_mode`, `tmpfs`) requires:
  1. Adding entry to `scalarOptions` or `listOptions` array in RuntimeParameterEditor.vue
  2. Adding corresponding field in `buildOutput()` and `loadFromModel()`
  3. Updating `commandPreview` computed to render it
  4. Updating runtimeBoundaryUi.test.mjs to assert its presence
- **Migration path to schema-driven:**
  1. Define Docker param schema in BackendVersion `default_args_schema_json` (new group: "docker")
  2. Extend `BackendParamDef` interface with Docker-specific fields (e.g. `is_list`, `placeholder`)
  3. Modify RuntimeParameterEditor to merge Docker schema entries with current hardcoded entries
  4. Deprecate `scalarOptions`/`listOptions` reactive arrays
  5. Update commandPreview to iterate schema instead of hardcoded keys
**Risk:** Current hardcoded list missing a needed param blocks that feature until code change.
**Trigger:** Next new Docker param needed (e.g., `--tmpfs`, `--userns`) → implement migration.
**Estimated cost:** ~3-4h for full migration.
