# Acceptance Criteria

> Status: DRAFT
> Date: 2026-06-25

---

## 1. 按 Issue 验收

### RAP-001: RunnerConfigsPage 双编辑入口与 editParameterModel 未填充

| 维度 | 标准 |
|------|------|
| 代码行为 | RunnerConfigsPage 编辑弹窗 template 中只保留 RuntimeParameterEditor，无 legacy Docker editor (el-switch/editPrivileged, el-input/editIpcMode, el-input/editShmSize, textarea/editDevicesText/editGroupAddText/editSecurityOptText/editUlimitsText) |
| 代码行为 | showEdit() 中包含 `editParameterModel.value = { docker_json: ..., args_override_json: ..., default_env_json: ..., parameter_values_json: ... }` |
| 代码行为 | 删除的 legacy ref 在文件中无残留引用 |
| UI 行为 | 编辑弹窗打开 → Docker 参数（privileged, ipc_mode, shm_size, devices, group_add, security_opt, ulimits）仅在 RuntimeParameterEditor 的 high-risk/list 区域出现一次 |
| UI 行为 | 参数值显示实际 row 数据（非空白/默认值） |
| 保存/重新打开 | 修改参数值 → 保存 → 关闭弹窗 → 重新打开 → 参数值与保存时一致 |
| npm test | PASS，无新增 failure |
| npm run build | PASS，无 TypeScript 错误 |

### RAP-002: ModelDeploymentsPage editParameterModel 返回空 docker_json

| 维度 | 标准 |
|------|------|
| 代码行为 | editParameterModel 从 computed 改为 ref |
| 代码行为 | showEdit() 中从 row 填充 docker_json + parameter_values_json |
| UI 行为 | 编辑弹窗 → Docker 参数（high-risk options）显示实际 row 数据 |
| UI 行为 | 可修改 Docker 参数值并保存 |
| 保存/重新打开 | 修改 → 保存 → 重新打开 → 数据一致 |
| npm test | PASS |

### RAP-003: RuntimeParameterEditor watch/emit 循环导致 OOM

| 维度 | 标准 |
|------|------|
| 代码行为 | syncing guard 在 Vue 3 异步 flush 下有效（用 nextTick 或序列号替代简单 flag） |
| 代码行为 | modelValue watch 不触发 loadFromModel 无限循环 |
| UI 行为 | BackendRuntimesPage 编辑弹窗停留 2 分钟 → Chrome Memory: JS heap size 稳定（不持续增长） |
| UI 行为 | RunnerConfigsPage 编辑弹窗停留 2 分钟 → 同上 |
| UI 行为 | ModelDeploymentsPage 编辑弹窗停留 2 分钟 → 同上 |
| UI 行为 | 修改任意参数 10 次 → 响应无卡顿 |
| npm test | PASS |

### RAP-004: package-release.sh 未包含 configs/backend-catalog/

| 维度 | 标准 |
|------|------|
| 代码行为 | package-release.sh 包含 `cp -r configs/backend-catalog "$BUILD_DIR/configs/"` |
| 打包产物 | `tar -tzf dist/lightai-go-*.tar.gz \| grep -c 'backend-catalog/'` ≥ 20 |
| Clean DB | `rm -f data/lightai.db` 后启动 packaged container → `curl /api/v1/inference-backends` 返回非空数组 |
| API | `curl /api/v1/inference-backends` 包含 vllm, sglang, llamacpp |
| UI | BackendRuntimesPage 显示 runtime 列表 |

### RAP-005: help YAML 存在但 UI 未接入

| 维度 | 标准 |
|------|------|
| UI 行为 | BackendRuntimesPage 编辑弹窗 → 每个参数旁有 ? icon |
| UI 行为 | hover ? icon → popover 显示 help（title + summary + recommendation + risk） |
| UI 行为 | vLLM/SGLang/llama.cpp 三后端的 help 均可查看 |
| npm test | PASS |

### RAP-006: extra_args 冲突检测仅 warning

| 维度 | 标准 |
|------|------|
| 决策 | 有明确文档记录：是否升级为 preflight 阻断，或保持 warning |
| 实现 | 如升级：preflight response 包含 structured error |
| 实现 | 如保持：代码注释说明选择理由 |

### RAP-007: DeviceBinding dead struct

| 维度 | 标准 |
|------|------|
| 决策 | 有明确文档记录：删除或保留 |
| 实现 | 如删除：grep 确认无残留引用 |
| go test | PASS |

### RAP-008: RuntimeRequirements/BackendCapabilityProfile 未落地

| 维度 | 标准 |
|------|------|
| 决策 | 有明确文档记录：DEFERRED_WITH_REASON（原因、风险、触发条件） |

### RAP-009: 零浏览器测试

| 维度 | 标准 |
|------|------|
| 代码行为 | 存在至少一个 browser-based smoke 脚本（Playwright/headless Chrome） |
| 测试 | browser smoke 能打开 BackendRuntimesPage 并检查关键元素 |

### RAP-010: 零 packaged artifact smoke

| 维度 | 标准 |
|------|------|
| 代码行为 | 存在 `scripts/e2e-packaged-smoke.sh` |
| 测试 | 脚本从 tarball 启动 → API 验证 → 成功 |

### RAP-011: closeout 文档状态不一致

| 维度 | 标准 |
|------|------|
| 文档 | `runtime-parameter-system-final-closeout.md` 顶部有 REOPENED banner |
| 文档 | `runtime-parameter-layering-final-closeout.md` 顶部有 REOPENED banner |
| 文档 | 交叉引用指向 `docs/reports/repairs/runtime-architecture-parameter-2026-06-25/` |

### RAP-012: npm test 偏静态源码检查

| 维度 | 标准 |
|------|------|
| 测试 | 现有 test 文件增加至少 1 个负向断言（检测不应出现的 pattern） |
| 测试 | npm test PASS |

### RAP-013: evidence 目录缺少索引

| 维度 | 标准 |
|------|------|
| 文档 | evidence 目录有 README 或索引 |

### RAP-014: Docker 参数列表 hardcoded

| 维度 | 标准 |
|------|------|
| 决策 | 已确认当前 hardcoded 范围（scalarOptions 6项 + listOptions 9项）和影响 |
| 决策 | 已形成处理结论：本轮迁移到 schema/catalog 驱动，或记录为 deferred 并明确触发条件 |
| 文档 | 如 deferred：closeout 记录原因、风险、触发条件 |
| 文档 | 后续新增 Docker 参数时有明确扩展路径 |
| 文档 | 如迁移：RuntimeParameterEditor 从 schema 动态渲染 Docker 参数 |

---

## 2. 按 Work Package 验收

### WP-A: 参数编辑 UI 数据流闭环

1. RunnerConfigsPage 编辑弹窗只有一套参数编辑入口
2. ModelDeploymentsPage 编辑弹窗 Docker 参数可编辑
3. 三个页面编辑模型格式一致 (`{docker_json, args_override_json, default_env_json, parameter_values_json}`)
4. 保存→重开数据一致
5. BackendRuntimesPage 功能无回归

### WP-B: RuntimeParameterEditor 稳定性与 OOM 修复

1. 三个页面编辑弹窗停留 2 分钟 → Chrome JS heap 稳定
2. watch→emit 循环不再发生（可在代码中加 counter 验证）
3. 参数修改后 command preview 仍正常更新
4. 三个页面都通过内存验证

### WP-C: Catalog、打包与 clean DB 初始化

1. `tar -tzf` 确认 catalog 文件在 tarball 中
2. Clean DB startup → API 返回 backend 列表
3. BackendRuntimesPage 可正常显示

### WP-D: 参数 help 与用户可理解性

1. 三后端所有参数有 ? icon
2. popover 展示 help 内容
3. help 内容准确（与 YAML 一致）

### WP-E: 测试体系补强

1. 负向断言能发现重复入口
2. packaged smoke 能发现 catalog 缺失
3. browser smoke 能打开关键页面

### WP-F: 架构遗留项与策略项处理

1. 每个遗留项有明确决策记录
2. closeout 文档状态更新
3. dead code（如决策删除）已清理

---

## 3. 全体验收（WP-A 至 WP-F 完成后）

| 测试 | 预期 |
|------|------|
| `bash scripts/e2e-real-smoke-all-three.sh` | vLLM/SGLang/llama.cpp 全部 PASS |
| `bash scripts/e2e-model-runtime-param-trace.sh` | 三后端 param trace PASS |
| `bash scripts/e2e-packaged-smoke.sh` | 从 tarball 启动 + API 验证 PASS |
| `bash scripts/e2e-matrix-verifier.sh` | 矩阵验证 PASS |
| `cd web && npm test` | 132+ tests PASS |
| `cd web && npm run build` | PASS |
| `go test ./internal/...` | 所有 packages PASS |
| Clean DB packaged startup | API 返回 backend 列表 |
| UI 重复入口 | 确认无 |
| UI OOM | 确认无 |
| Help popover | 可用 |
