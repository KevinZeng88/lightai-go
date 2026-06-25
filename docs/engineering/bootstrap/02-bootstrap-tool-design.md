# lightai-bootstrap.sh 工具设计

状态：`READY_FOR_IMPLEMENTATION`

## 目标

`scripts/lightai-bootstrap.sh` 是 LightAI Go 的统一初始化工具，用于：

- reset DB 后恢复开发测试环境；
- 安装后按 profile 初始化系统；
- 为 Browser E2E / API E2E 准备稳定数据；
- 导出当前环境为可复用 profile；
- 在发行版中作为交付初始化工具。

## 文件位置

```text
scripts/lightai-bootstrap.sh
```

可选辅助库：

```text
scripts/lib/
scripts/e2e/lib/
```

## 默认短命令

最短命令：

```bash
bash scripts/lightai-bootstrap.sh
```

默认等价于：

```bash
bash scripts/lightai-bootstrap.sh \
  --profile configs/bootstrap/local-kz-laptop.yaml \
  --mode dry-run
```

默认 export：

```bash
bash scripts/lightai-bootstrap.sh --mode export
```

默认 export 输出：

```text
configs/bootstrap/local-kz-laptop.yaml
```

## 参数优先级

1. 命令行参数
2. 环境变量
3. profile 配置
4. 脚本内置默认值

## 内置默认值

| 参数 | 默认值 |
|---|---|
| profile | `configs/bootstrap/local-kz-laptop.yaml` |
| mode | `dry-run` |
| base_url | `http://localhost:18080` |
| agent_url | `http://localhost:19091` |
| runtime_dir | `/tmp/lightai` |
| output_dir | `/tmp/lightai/e2e/bootstrap` |
| initial password env | `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` |
| final/admin password env | `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` |

## 核心参数

| 参数 | 说明 |
|---|---|
| `--profile <path>` | bootstrap YAML profile |
| `--mode <mode>` | 执行模式 |
| `--base-url <url>` | server URL |
| `--agent-url <url>` | agent URL |
| `--runtime-dir <dir>` | 运行目录 |
| `--output-dir <dir>` | 输出目录 |
| `--initial-password <value>` | 初始密码，禁止写入日志 |
| `--initial-password-file <path>` | 初始密码文件 |
| `--admin-password <value>` | 目标管理员密码，禁止写入日志 |
| `--admin-password-file <path>` | 目标管理员密码文件 |
| `--allow-real-start` | 允许 full 模式真实启动容器 |
| `--output-profile <path>` | export 输出 profile 路径 |
| `--include-secrets` | export 是否包含敏感字段，默认 false |
| `--include-runtime-state` | export 是否包含 deployments 等运行态，默认 false |
| `--yes` | 高风险导出确认 |

## 支持 mode

所有 mode 第一版都应实现：

- `auth-only`
- `catalog-only`
- `models-only`
- `runtimes-only`
- `dry-run`
- `full`
- `export`

## Mode 语义

### auth-only

执行：

1. server 可访问检查；
2. agent 可访问检查；
3. runtime-dir 检查；
4. 读取 final/admin password；
5. 用 final/admin password 尝试登录；
6. 如失败，读取 initial password；
7. 用 initial password 登录；
8. 如返回 `must_change_password=true`，调用改密接口；
9. 用 final/admin password 重新登录；
10. 输出 `auth.json`。

### CSRF 与 Session 处理

Bootstrap 脚本通过 API 操作服务端，会话管理必须正确处理 CSRF 保护。流程如下：

1. **登录后获取 CSRF token**：`POST /api/v1/auth/login` 返回 JSON 中包含 `csrf_token` 字段。
2. **保存 session cookie**：从登录响应的 `Set-Cookie` header 中提取 session cookie，写入 cookie jar 文件。
3. **后续请求携带凭据**：所有 write API 请求必须同时携带：
   - session cookie（从 cookie jar 读取）
   - CSRF token（从登录响应提取，放入 `X-CSRF-Token` header）
4. **复用已有模式**：`scripts/e2e/lib/` 中已有 cookie jar 和 CSRF 处理模式，bootstrap 应复用而非重新实现。

以下 write API 需要 session + CSRF（当前已确认的路由）：

| 操作 | API | 所需凭据 |
|------|-----|---------|
| 改密 | `POST /api/v1/auth/change-password` | session + CSRF |
| 创建 model artifact | `POST /api/v1/model-artifacts` | session + CSRF |
| discover model artifact | `POST /api/v1/model-artifacts/discover` | session + CSRF |
| 创建 model location | `POST /api/v1/model-artifacts/{id}/locations` | session + CSRF |
| 创建 BackendRuntime | `POST /api/v1/backend-runtimes` | session + CSRF |
| clone BackendRuntime | `POST /api/v1/backend-runtimes/{id}/clone` | session + CSRF |
| enable NBR | `POST /api/v1/nodes/{id}/backend-runtimes/enable` | session + CSRF |
| check-request NBR | `POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request` | session + CSRF |
| deployment preflight | `POST /api/v1/deployments/preflight` | session + CSRF |
| deployment dry-run | `POST /api/v1/deployments/{id}/dry-run` | session + CSRF |
| 创建 deployment | `POST /api/v1/deployments` | session + CSRF |
| 启动 deployment | `POST /api/v1/deployments/{id}/start` | session + CSRF |
| 测试模型 | `POST /api/v1/model-instances/{id}/test` | session + CSRF |

**安全规则**：所有输出文件（`auth.json`、`effective-config.json`、`bootstrap-state.json`、`*.log`）**不得**保存 CSRF token、session cookie 或任何 token 明文。

### catalog-only

执行 `auth-only` 前置步骤，然后检查 catalog：

- vLLM
- SGLang
- llama.cpp

输出 `catalog.json`。

### models-only

执行 `catalog-only` 前置步骤，然后：

1. 检查 profile 中模型路径存在；
2. 注册或确认 model artifacts；
3. 注册或确认 model locations；
4. 输出 `models.json` 和 `model-locations.json`。

### runtimes-only

执行 `models-only` 前置步骤，然后：

1. 确认 node 存在且可用；
2. 创建或确认 BackendRuntime；
3. 创建或确认 NodeBackendRuntime；
4. 输出 `backend-runtimes.json` 和 `node-backend-runtimes.json`。

### dry-run

执行 `runtimes-only` 前置步骤，然后：

1. enable 每个 NodeBackendRuntime；
2. check-request 每个 NodeBackendRuntime；
3. 对每个 runtime 执行 preflight / runplan dry-run；
4. 输出 `preflight-results.json` 和 `bootstrap-state.json`。

### full

执行 `dry-run` 前置步骤，然后：

1. 创建或确认 deployment；
2. 启动真实容器；
3. 检查 instance 状态；
4. 读取 logs；
5. 检查 health / models endpoint；
6. 停止容器或按 profile 配置保留。

full 模式真实启动必须同时满足（双重确认）：

```text
profile.bootstrap.allow_real_container_start=true
--allow-real-start
```

**Chat completion 默认不执行。** 只有同时满足以下条件才执行 chat completion：
- `profile.bootstrap.allow_chat_completion=true`
- `--allow-chat-completion` 命令行 flag

第一版实现 chat completion 为 opt-in。后续可根据需要调整默认行为。

### export

登录后读取当前环境并生成 bootstrap profile。只读，不修改系统状态。

详见 `04-export-mode-design.md`。

## Runtime-dir 检查

启动时检查：

- `<runtime-dir>/data/lightai.db`
- `<runtime-dir>/logs/`
- `<runtime-dir>/logs/server.log`
- `<runtime-dir>/logs/agent.log`
- runtime initial credentials file
- server 是否连接到该 runtime-dir 下的 DB，如现有 API 或日志可验证

如 DB 文件不存在：

- 输出 WARN；
- 继续走 API 初始化；
- 在 `runtime-dir-check.json` 中记录状态。

## 输出目录

默认：

```text
/tmp/lightai/e2e/bootstrap/
```

可通过 `--output-dir` 或 profile 覆盖。

## 输出文件

| 文件 | 内容 |
|---|---|
| `bootstrap.log` | 全流程日志，脱敏 |
| `effective-config.json` | 实际生效配置，脱敏 |
| `bootstrap-state.json` | 后续 E2E 使用的资源 ID 与结果 |
| `auth.json` | 登录和改密状态，脱敏 |
| `runtime-dir-check.json` | runtime-dir 检查结果 |
| `catalog.json` | catalog 检查结果 |
| `nodes.json` | node 识别结果 |
| `models.json` | model artifact 结果 |
| `model-locations.json` | model location 结果 |
| `backend-runtimes.json` | backend runtime 结果 |
| `node-backend-runtimes.json` | NBR 结果 |
| `preflight-results.json` | preflight / runplan dry-run 结果 |
| `full-results.json` | full 模式结果 |
| `export-summary.json` | export 资源统计 |
| `export-resource-map.json` | export 资源映射 |
| `export-warnings.json` | export 警告 |
| `errors.json` | 错误列表 |

## bootstrap-state.json schema

至少包含：

```json
{
  "base_url": "http://localhost:18080",
  "agent_url": "http://localhost:19091",
  "runtime_dir": "/tmp/lightai",
  "output_dir": "/tmp/lightai/e2e/bootstrap",
  "db_path": "/tmp/lightai/data/lightai.db",
  "tenant_id": "...",
  "node_id": "...",
  "model_artifact_ids": {},
  "model_location_ids": {},
  "backend_ids": {},
  "backend_version_ids": {},
  "backend_runtime_ids": {},
  "node_backend_runtime_ids": {},
  "deployment_ids": {},
  "preflight_results": {},
  "full_results": {},
  "generated_at": "..."
}
```

## effective-config.json schema

至少包含：

```json
{
  "profile_path": "configs/bootstrap/local-kz-laptop.yaml",
  "mode": "dry-run",
  "base_url": "http://localhost:18080",
  "agent_url": "http://localhost:19091",
  "runtime_dir": "/tmp/lightai",
  "output_dir": "/tmp/lightai/e2e/bootstrap",
  "auth_username": "admin",
  "initial_password_source": "env:LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD",
  "final_password_source": "env:LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD",
  "allow_real_start": false,
  "include_runtime_state": false,
  "include_secrets": false
}
```

不得包含明文密码、token、cookie、CSRF。

## auth.json schema

至少包含：

```json
{
  "login_status": "PASS",
  "username": "admin",
  "auth_method": "final_password",
  "token_present": true,
  "csrf_present": true,
  "password_changed": false,
  "initial_password_source": "env:LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD",
  "final_password_source": "env:LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD",
  "runtime_initial_credentials_file": "/tmp/lightai/runtime/initial-credentials.txt",
  "timestamp": "..."
}
```

不得包含明文密码、token、cookie、CSRF。

## 幂等要求

重复运行时：

- tenant 存在则复用；
- node 存在则复用；
- model artifact 存在则复用或更新；
- model location 存在则复用或更新；
- BackendRuntime 存在则复用或更新；
- NodeBackendRuntime 存在则复用或更新；
- 已 enable 则跳过或确认；
- check-request 可重新执行；
- deployment 存在则复用或更新。

每一步输出状态：

- `CHECK`
- `CREATE`
- `UPDATE`
- `SKIP`
- `PASS`
- `WARN`
- `FAIL`

## API 优先原则

bootstrap 应优先使用 API 完成初始化，避免直接写 DB。runtime-dir / DB 读取用于诊断、补充、验证，不作为主写入路径。

## 失败处理

失败时：

1. 当前步骤输出 `FAIL`；
2. API response 保存到对应 JSON 或 `errors.json`；
3. 明确错误原因；
4. 不吞错；
5. 不把部分成功伪装成 PASS；
6. 不输出敏感信息。
