# LightAI Go 全流程 UI 自动化回归计划

> 基于已完成的 Playwright / 登录 / Vite Origin 基线，规划后续业务 UI 自动化覆盖范围、测试顺序、验收标准和问题修复原则。

---

## 1. 总目标

建立 LightAI Go 全流程 UI 自动化回归能力，覆盖从模型、运行模板、节点运行配置、添加运行环境、模型部署、实例生命周期到 OpenAI 兼容入口的主要页面流程。

目标不是只验证页面能打开，而是验证：

1. 页面展示正确。
2. 用户操作真实可用。
3. 保存 payload 正确。
4. 刷新后状态保持。
5. API 返回与 UI 展示一致。
6. 源配置与副本配置互不污染。
7. 失败时有明确错误信息和可追溯 evidence。

---

## 2. 总体测试原则

后续 UI 自动化必须遵守：

1. 使用真实浏览器行为，不绕过页面。
2. 需要登录的测试统一复用 `tests/e2e/.auth/admin.json`。
3. 所有写操作必须走浏览器上下文和 Vite proxy。
4. 保存类操作必须做 UI + API 双重校验。
5. 刷新页面后必须再次校验状态。
6. 失败时保留 screenshot / trace / video / API response。
7. 优先使用稳定 selector，必要时补充 `data-testid`。
8. 不使用截图比对作为主要断言。
9. 不依赖人工观察。
10. 有副作用测试不并发执行。
11. 不为旧数据写兼容 fallback。
12. 不把问题只记录为 future，如果当前可定位、可修复、可验证，应直接修复并补测试。

---

## 3. 推荐目录结构

```text
web/tests/e2e/
  smoke/
    app-load.spec.ts
    fullstack-health.spec.ts

  auth/
    login.spec.ts
    login-debug.spec.ts

  helpers/
    auth.ts
    api.ts
    selectors.ts
    runtime-config-api.ts
    test-data.ts

  page-objects/
    LoginPage.ts
    ConsoleLayout.ts
    RuntimeTemplatesPage.ts
    NodeBackendRuntimesPage.ts
    ModelArtifactsPage.ts
    DeploymentsPage.ts
    InstancesPage.ts

  runtime-configs/
    runtime-template-parameter-display.spec.ts
    runtime-config-clone-name-persistence.spec.ts
    runtime-config-parameter-value-persistence.spec.ts
    runtime-config-parameter-enabled-persistence.spec.ts
    runtime-config-no-duplicate-fields.spec.ts

  node-backend-runtimes/
    nbr-create-enable-check.spec.ts
    nbr-runplan-preview.spec.ts
    nbr-device-binding.spec.ts

  models/
    model-create-edit-location.spec.ts
    model-capability-display.spec.ts

  deployments/
    deployment-create-preflight.spec.ts
    deployment-parameter-override.spec.ts
    deployment-runplan-preview.spec.ts

  instances/
    instance-start-status-logs-stop.spec.ts
    instance-logs-refresh.spec.ts

  openai-compatible/
    openai-models-chat.spec.ts
```

---

## 4. Batch 1：运行模板 / 用户运行配置

这是当前优先级最高的一批，因为手工验证已发现多个具体问题。

### 4.1 参数展示语言

测试文件：

```text
web/tests/e2e/runtime-configs/runtime-template-parameter-display.spec.ts
```

验证点：

- 中文界面下不显示裸英文参数名。
- 不显示内部 key。
- 参数展示使用中文 label 或 i18n label。
- 以下字段不应作为普通用户参数裸露展示：
  - `Model mount`
  - `Environment variables`
  - `Kind`
  - `Ports`
  - `Volumes`
  - `Devices`
  - `Extra env`
  - `Backend extra args`

验收：

```text
页面不存在上述裸英文 label。
页面存在可理解的中文标签。
```

### 4.2 内部字段分层

测试文件：

```text
web/tests/e2e/runtime-configs/runtime-config-no-duplicate-fields.spec.ts
```

验证点：

- Docker 结构字段与普通运行参数分离。
- 端口、卷、设备、环境变量应属于结构化配置区，不应混入普通参数编辑器。
- `devices` / device binding 不应重复展示。

验收：

```text
同一语义字段不重复出现。
结构字段不混入 RuntimeParameterEditor 普通参数列表。
```

### 4.3 复制配置名称持久化

测试文件：

```text
web/tests/e2e/runtime-configs/runtime-config-clone-name-persistence.spec.ts
```

测试流程：

1. 登录。
2. 进入运行模板或节点运行配置页面。
3. 选择一个内置模板或已有配置。
4. 点击复制。
5. 输入新的名称 / 显示名。
6. 保存。
7. 回到列表。
8. 重新进入详情页。
9. 调 API 查询。
10. 验证名称保持新值。
11. 验证源配置名称未被修改。

验收：

```text
创建页输入的新名称在保存后、刷新后、API 中均保持一致。
源配置不被污染。
```

### 4.4 参数 value 持久化

测试文件：

```text
web/tests/e2e/runtime-configs/runtime-config-parameter-value-persistence.spec.ts
```

测试流程：

1. 复制一个配置。
2. 修改一个参数 value。
3. 保存。
4. 刷新页面。
5. 重新进入配置。
6. 调 API 查询。
7. 验证 UI 与 API 中 value 一致。

验收：

```text
参数 value 不因刷新或重新进入页面丢失。
```

### 4.5 参数 enabled 持久化

测试文件：

```text
web/tests/e2e/runtime-configs/runtime-config-parameter-enabled-persistence.spec.ts
```

测试流程：

1. 复制一个配置。
2. 修改一个参数 enabled 状态。
3. 保存。
4. 刷新页面。
5. 重新进入配置。
6. 调 API 查询。
7. 验证 UI 与 API 中 enabled 一致。
8. 验证 value 与 enabled 是独立保存的。

验收：

```text
enabled 不因保存、刷新、schema default 或 value 存在而被重置。
```

---

## 5. Batch 2：添加运行环境 / 节点运行配置

测试目录：

```text
web/tests/e2e/node-backend-runtimes/
```

建议测试：

```text
nbr-create-enable-check.spec.ts
nbr-runplan-preview.spec.ts
nbr-device-binding.spec.ts
```

验证点：

1. 可以选择节点。
2. 可以选择 backend runtime / image。
3. 可以填写端口、卷、设备、环境变量、健康检查。
4. 可以执行 check。
5. `ready` 或 `ready_with_warnings` 状态可用于部署。
6. check 失败时错误信息明确，不显示“未知”。
7. NVIDIA / MetaX 等设备绑定抽象不混乱。
8. RunPlan 预览与 API 返回一致。

---

## 6. Batch 3：模型库 / 模型位置

测试目录：

```text
web/tests/e2e/models/
```

建议测试：

```text
model-create-edit-location.spec.ts
model-capability-display.spec.ts
```

验证点：

1. 可以新增模型。
2. 可以编辑模型显示名、路径、格式、任务类型等基础信息。
3. 可以添加 / 修改模型位置。
4. 模型页不展示 Docker 参数。
5. 模型能力展示来自模型事实 / metadata / 用户配置，不和运行参数混淆。
6. 保存后 UI 与 API 一致。

---

## 7. Batch 4：模型部署 / Preflight / RunPlan

测试目录：

```text
web/tests/e2e/deployments/
```

建议测试：

```text
deployment-create-preflight.spec.ts
deployment-parameter-override.spec.ts
deployment-runplan-preview.spec.ts
```

验证点：

1. 可以选择模型。
2. 可以选择模型位置。
3. 可以选择节点运行配置。
4. 可以设置部署级参数覆盖。
5. 最终 RunPlan 预览与 API 返回一致。
6. preflight 错误通过 errors 数组展示。
7. `ready_with_warnings` 可部署。
8. 部署 API 不接受 legacy `backend_runtime_id`。
9. 上游模板修改不会 live 覆盖 deployment snapshot。

---

## 8. Batch 5：实例生命周期 / 日志

测试目录：

```text
web/tests/e2e/instances/
```

建议测试：

```text
instance-start-status-logs-stop.spec.ts
instance-logs-refresh.spec.ts
```

验证点：

1. 可以启动实例。
2. 状态从等待 / 启动中进入运行或明确失败。
3. 状态由真实健康检查驱动。
4. 日志可以刷新。
5. 失败实例显示具体失败原因和容器日志。
6. 停止实例后，实例列表符合产品预期。
7. 容器 ID / instance ID 不混用。

---

## 9. Batch 6：OpenAI 兼容入口 / 审计计费

测试目录：

```text
web/tests/e2e/openai-compatible/
```

建议测试：

```text
openai-models-chat.spec.ts
```

验证点：

1. `/v1/models` 可访问。
2. chat/completions 或当前支持接口可访问。
3. 失败时页面/API 给出明确错误。
4. 后续补充 API Key、审计、用量统计、计费验证。

---

## 10. 每个 UI 测试的标准结构

每个测试必须包含：

```text
前置条件
UI 操作步骤
保存前状态
保存 payload 或 API response
保存后 API 状态
刷新后 UI 状态
源对象是否被污染
失败 evidence 路径
```

保存类测试必须覆盖：

```text
保存前 UI
保存请求
保存后 API
刷新后 UI
重新进入页面后 UI
```

---

## 11. 数据隔离与命名规则

测试创建的数据必须带唯一前缀，例如：

```text
e2e-runtime-config-<timestamp>
e2e-nbr-<timestamp>
e2e-deployment-<timestamp>
```

测试完成后应尽量清理。无法清理时必须确保：

1. 不影响下一次测试。
2. 可通过前缀识别。
3. 后续可批量删除。

---

## 12. 失败证据要求

Playwright 已配置失败时保留：

```text
/tmp/lightai/e2e/playwright/results
/tmp/lightai/e2e/playwright/report
```

失败时应检查：

```bash
npm run test:e2e:report
```

必要时查看 trace：

```bash
npx playwright show-trace /tmp/lightai/e2e/playwright/results/<case>/trace.zip
```

---

## 13. 当前下一步建议

基线已完成后，下一步先做：

```text
Batch 1：运行模板 / 用户运行配置参数 UI 回归
```

不要直接跳到部署或实例生命周期。原因：

1. 当前已知问题集中在参数展示、复制保存、enabled/value 持久化。
2. 参数体系是后续部署、RunPlan、实例启动的上游基础。
3. 如果参数 UI 不稳定，后续全流程测试会产生大量伪失败。

建议先完成并通过：

```text
runtime-template-parameter-display.spec.ts
runtime-config-clone-name-persistence.spec.ts
runtime-config-parameter-value-persistence.spec.ts
runtime-config-parameter-enabled-persistence.spec.ts
runtime-config-no-duplicate-fields.spec.ts
```

之后再进入添加运行环境和部署流程。
