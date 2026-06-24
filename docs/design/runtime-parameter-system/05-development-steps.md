# 05 - Development Steps

> Status: active plan
> Date: 2026-06-25
> Governance: see 08-execution-governance-and-decisions.md

## Phase 0: 现状审计和基线 evidence

**目标**：跑当前三后端 E2E，保存基线 evidence，记录当前参数链路问题。

**可修改范围**：无代码修改，只跑测试和保存 evidence。

**不可修改范围**：所有代码文件。

**实施步骤**：
1. 启动 Server + Agent
2. 跑 vLLM E2E，保存 RunPlan / equivalent command / Docker inspect
3. 跑 SGLang E2E，保存同上
4. 跑 llama.cpp E2E，保存同上
5. 记录当前参数链路问题清单

**必跑测试**：
```bash
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**Evidence 路径**：`docs/reports/phase-0/evidence/`

**验收标准**：
- vLLM E2E 有结果（PASS 或 FAIL + 原因）
- SGLang E2E 有结果
- llama.cpp E2E 有结果
- baseline evidence 已保存
- 当前参数链路问题已记录

**Commit/Push 条件**：evidence / baseline 文档 commit + push。如 evidence 过大不适合入库，在报告中写明不入库原因。

**停止条件**：三后端 E2E 全部 FAIL 且无法定位原因（非外部硬件/镜像不可控原因）。

**是否允许自动进入下一 Phase**：是（evidence only，无代码风险）。

---

## Phase 1: 参数语义正确性最小闭环

**目标**：required 参数 locked-on，optional 参数 enabled/value，Layer 3 模板替换，核心 schema 正确，mini param trace 证明优先级。

**可修改范围**：
- `configs/backend-catalog/versions/*.yaml`
- `internal/server/api/runtime_handlers.go`
- `internal/server/runplan/resolver.go`
- `web/src/components/common/RuntimeParameterEditor.vue`
- `web/tests/runtimeBoundaryUi.test.mjs`

**不可修改范围**：
- DB schema
- API 端点定义
- 其它页面组件

**实施步骤**：
1. 扩展 `default_args_schema` 添加 `type`, `group`, `label`, `placeholder` 字段（最小 UI 元数据，不添加 `help`、`help_ref`、不实现 help loader 和 `?` 说明，全部放 Phase 7）
2. required 参数在 UI 中 locked（el-switch disabled 或 el-checkbox disabled checked）
3. 修复 resolver Layer 3 deployment override 执行 `substituteVars`
4. 确认 host / container_port / model_path / served_model_name 核心 schema 正确
5. 确认 raw startup args 不承载 required core args
6. 跑 mini param trace：BackendVersion → BackendRuntime → NBR → Deployment → RunPlan
7. 验证 disabled value 保留但不进入 final config
8. 验证 required 缺失时 preflight 报结构化错误
9. 验证 host 不允许 Deployment override
10. 验证 container_port 默认不允许 Deployment override

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/... && go build ./cmd/server/... && go build ./cmd/agent/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**Evidence 路径**：`docs/reports/phase-1/evidence/`

**验收标准**：
- required 参数 UI 上 locked 或无普通 checkbox
- required 参数不能被 API/payload disabled
- optional 参数 disabled 后 value 保留
- disabled 参数不进入 RunPlan
- Deployment override 中模板变量能替换
- host 不允许 Deployment override
- container_port 默认不允许 Deployment override
- mini trace 证明 Deployment override > NBR > BackendRuntime > BackendVersion default
- vLLM modified params RunPlan 正确
- 三后端基础 E2E 不退化

**Commit/Push 条件**：代码修改 commit + push，evidence commit。

**停止条件**：
- required 参数仍可被 disable
- disabled value 丢失
- Deployment override 模板替换失败
- 三后端 E2E 任一退化

**是否允许自动进入下一 Phase**：是（验收通过后）。

---

## Phase 2: UI 分组、唯一编辑入口、基础可用性

**目标**：参数按 group 分组，high-risk 参数唯一编辑入口，入口统一，Deployment override 勾选后可输入，command preview 统一。

**可修改范围**：
- `web/src/components/common/RuntimeParameterEditor.vue`
- `web/src/pages/BackendRuntimesPage.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- i18n 文件

**不可修改范围**：
- Go 代码
- catalog YAML
- DB schema

**实施步骤**：
1. 参数按 `group` 字段分组显示
2. high-risk 参数（privileged/ipc/shm/devices/security/group_add）确保唯一编辑入口
3. BackendRuntimesPage 移除重复 scalarOptions/listOptions/customArgs/customEnv 编辑入口
4. Deployment override 勾选后 input/textarea/select 立即可编辑
5. 统一 command preview 来源
6. raw startup args 降级为 extra_args

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
```

**Evidence 路径**：`docs/reports/phase-2/evidence/`

**验收标准**：
- host 独立 input
- container_port 独立 input
- served_model_name 独立 input（如 backend 支持）
- required 参数不显示普通 enable checkbox
- optional 参数显示 enabled/value
- privileged/ipc/shm/devices/security/group_add 只在一个权威位置编辑
- BackendRuntimesPage 无重复编辑入口
- Deployment override 勾选后 input/textarea/select 立即可编辑
- 保存后 GET 回显
- 取消 enabled 不清空 value
- command preview 与 RunPlan 一致
- UI 无 i18n key 泄露

**Commit/Push 条件**：代码修改 commit + push。

**停止条件**：
- 重复编辑入口未消除
- Deployment override 勾选后仍无法输入
- enabled/value 保存回显失败

**是否允许自动进入下一 Phase**：是。

---

## Phase 3: 冲突检测和 preflight 强化

**目标**：extra_args 与 structured args 冲突检测，required 缺失检测，disabled 参数进入 final config 检测，vendor/backend 不匹配检测。

**可修改范围**：
- `internal/server/runplan/resolver.go`
- `internal/server/runplan/compat.go`
- `internal/server/api/deployment_lifecycle_handlers.go`
- `web/src/components/common/RuntimeParameterEditor.vue`（可选：UI 层面警告）

**不可修改范围**：
- catalog YAML
- DB schema

**实施步骤**：
1. resolver `deduplicateArgs` 检测重复 host/port/model/served-model-name，记录 warning
2. preflight 检测 required 参数缺失，返回结构化错误
3. preflight 检测 disabled 参数进入 final config
4. preflight 检测 vendor/backend 不匹配
5. UI 展示冲突 warning（可选）

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
go build ./cmd/server/... && go build ./cmd/agent/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
```

**Evidence 路径**：`docs/reports/phase-3/evidence/`

**验收标准**：
- extra_args 中重复 --host 有 warning 或 preflight error
- extra_args 中重复 --port 有 warning 或 preflight error
- extra_args 中重复 --model / --model-path / -m 有 warning 或 preflight error
- required 参数缺失返回结构化错误
- disabled env/device 不进入 final config
- vendor 不匹配时 preflight 报错
- 不允许 generic unknown error
- UI 展示字段、来源层、建议修复方式

**Commit/Push 条件**：代码修改 commit + push。

**停止条件**：
- 冲突检测导致正常参数被误报
- preflight 结构化错误信息不完整

**是否允许自动进入下一 Phase**：是。

---

## Phase 4: vendor 隔离和厂商模板

**目标**：NVIDIA / MetaX / Huawei vendor-specific runtime 参数隔离，未验证项标记。

**可修改范围**：
- `configs/backend-catalog/runtimes/*/`
- `internal/server/db/db.go`（seed data）
- `internal/server/runplan/resolver.go`（vendor filtering, optional）

**不可修改范围**：
- DB schema
- API 端点

**实施步骤**：
1. 确认 NVIDIA runtime YAML 不含 MetaX/Huawei devices
2. 确认 MetaX runtime YAML 只含 MetaX devices
3. 确认 Huawei runtime YAML 只含 Huawei devices
4. common schema 只定义字段，不带 vendor 默认值
5. MetaX/Huawei 未验证模板标记 verification.status

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
```

**Evidence 路径**：`docs/reports/phase-4/evidence/`

**验收标准**：
- NVIDIA RunPlan / Docker inspect 不含 /dev/dri、/dev/mxcd、/dev/infiniband
- MetaX vendor template 的 MetaX devices 仅 vendor-scoped
- Huawei vendor template 的 Ascend devices/mounts/env 仅 vendor-scoped
- MetaX/Huawei 未验证模板有 verification.status
- vendor 不匹配时 UI 不显示、RunPlan 不使用、Docker Spec 不包含
- vLLM NVIDIA E2E 不退化

**Commit/Push 条件**：catalog 修改 commit + push。

**停止条件**：
- vendor 串台验证失败
- NVIDIA E2E 退化

**是否允许自动进入下一 Phase**：是。

---

## Phase 5: 完整参数溯源 E2E

**目标**：创建参数溯源 E2E 脚本，验证每层参数的创建、修改、继承、覆盖，验证 final RunPlan / Docker inspect。

**可修改范围**：
- 新增 `scripts/e2e-model-runtime-param-trace.sh`
- 新增 evidence 目录

**不可修改范围**：
- 现有代码逻辑

**实施步骤**：
1. 创建 E2E 脚本
2. vLLM 做 full trace：创建模型 → 修改模型 → clone BackendRuntime → 修改 BR → 创建 NBR → 修改 NBR → 创建 Deployment → 修改 override → preflight → RunPlan → Docker inspect
3. SGLang / llama.cpp 做轻量 trace
4. 保存每层 evidence

**必跑测试**：
```bash
bash scripts/e2e-model-runtime-param-trace.sh
RUN_REAL_CONTAINER=1 TRACE_BACKEND=vllm bash scripts/e2e-model-runtime-param-trace.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**Evidence 路径**：`docs/reports/phase-5/evidence/param-trace/`

**必须保存的文件**：
```text
00-backend-version.json
01-model-created.json
02-model-modified.json
03-backend-runtime-created-or-cloned.json
04-backend-runtime-modified.json
05-nbr-created.json
06-nbr-modified.json
07-deployment-created.json
08-deployment-override-modified.json
09-preflight.json
10-runplan.json
11-equivalent-command.txt
12-docker-inspect.json
13-final-assertions.txt
14-param-source-table.md
```

**验收标准**：
- 每层 snapshot 存在
- 每层修改可 GET 回显
- 下一层继承符合规则
- Deployment override 最高优先级
- disabled value 保留但不进入 final config
- required 参数无法 disable
- final RunPlan args/env/devices/ports/high-risk options 与预期一致
- equivalent command 与 RunPlan 一致
- Docker inspect 与 RunPlan 一致
- NVIDIA 无 vendor 污染
- vLLM modified params 真实运行 PASS
- SGLang / llama.cpp 默认 E2E PASS

**Commit/Push 条件**：脚本 + evidence commit + push。

**停止条件**：
- E2E 脚本无法完成全流程
- RunPlan 与 Docker inspect 不一致
- vendor 污染验证失败

**是否允许自动进入下一 Phase**：是。

---

## Phase 6: 三后端/三厂商矩阵扩展

**目标**：扩展常用参数覆盖，扩展 vendor runtime 模板，可验证的真实 E2E，不可验证的 dry-run/catalog/evidence 标记。

**可修改范围**：
- `configs/backend-catalog/versions/*/`
- `configs/backend-catalog/runtimes/*/`
- 相关文档

**不可修改范围**：
- Go 核心逻辑
- DB schema

**实施步骤**：
1. vLLM 常用参数覆盖表
2. SGLang 常用参数覆盖表
3. llama.cpp 常用参数覆盖表
4. NVIDIA vendor 参数覆盖表
5. MetaX vendor 参数覆盖表
6. Huawei vendor 参数覆盖表
7. 每个 backend/vendor 组合标记 verified / requires_hardware_validation / template_only
8. 可验证组合 E2E

**必跑测试**：
```bash
npm run build && npm test
go test ./internal/...
bash scripts/e2e-model-runtime-wizard-nvidia-vllm.sh
bash scripts/e2e-model-runtime-wizard-nvidia-sglang.sh
bash scripts/e2e-model-runtime-wizard-nvidia-llamacpp.sh
```

**Evidence 路径**：`docs/reports/phase-6/evidence/`

**验收标准**：
- vLLM 常用参数覆盖表完成
- SGLang 常用参数覆盖表完成
- llama.cpp 常用参数覆盖表完成
- NVIDIA vendor 参数覆盖表完成
- MetaX vendor 参数覆盖表完成
- Huawei vendor 参数覆盖表完成
- 每个 backend/vendor 组合标记 verified / requires_hardware_validation / template_only
- 可验证组合 E2E PASS
- 不可验证组合 dry-run/catalog 检查 PASS 或记录原因

**Commit/Push 条件**：catalog + 文档 commit + push。

**停止条件**：
- 可验证组合 E2E FAIL
- catalog 参数与 image help 不一致

**是否允许自动进入下一 Phase**：是。

---

## Phase 7: 外置 help 文档和 ? 参数说明

**目标**：建立外置 help 文档，UI ? 弹窗，help 内容和 schema 解耦。

**可修改范围**：
- 新增 `configs/backend-catalog/help/*/`
- `web/src/components/common/RuntimeParameterEditor.vue`
- i18n 文件

**不可修改范围**：
- Go 核心逻辑
- DB schema
- catalog YAML 参数定义

**实施步骤**：
1. 建立 help 文档目录结构
2. 编写 backend 参数 help（vLLM / SGLang / llama.cpp）
3. 编写 vendor 参数 help（NVIDIA / MetaX / Huawei）
4. 编写 Docker runtime 参数 help
5. 编写 best-practice recommendation
6. 编写 risk warning
7. 实现 UI ? 弹窗
8. zh-CN / en-US 扩展

**必跑测试**：
```bash
npm run build && npm test
```

**Evidence 路径**：`docs/reports/phase-7/evidence/`

**验收标准**：
- help 文档按 backend/version/vendor 分目录
- 每个常用参数有说明
- 每个高风险参数有风险提示
- 每个 vendor-specific 参数有适用厂商提示
- UI ? 可以展示说明
- help 缺失时不影响参数编辑和运行
- help 文档可独立更新
- 无 i18n key 泄露
- npm build/test PASS

**Commit/Push 条件**：help 文档 + UI 修改 commit + push。

**停止条件**：
- help 文档与 schema 不同步
- UI ? 弹窗功能异常

**是否允许自动进入下一 Phase**：是（最后一个 Phase）。

---

## 开发顺序

```
Phase 0 → Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6 → Phase 7
```

每个 Phase 独立可提交，不依赖后续 Phase。Phase 1-6 不涉及 help 文档。Phase 7 独立做 help 文档。
