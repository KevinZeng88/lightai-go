# 验收标准

状态：`READY_FOR_IMPLEMENTATION`

## 总体验收

1. `bash scripts/lightai-bootstrap.sh` 可以按默认 profile 和默认 mode 运行。
2. 所有 mode 第一版均可运行：
   - `auth-only`
   - `catalog-only`
   - `models-only`
   - `runtimes-only`
   - `dry-run`
   - `full`
   - `export`
3. 输出默认进入 `/tmp/lightai/e2e/bootstrap/`。
4. 日志和 JSON 状态文件不包含明文密码、token、cookie、CSRF。
5. 脚本重复运行不会产生重复脏数据。
6. 发行版包含 bootstrap 脚本、profile 和文档。

## 密码契约验收

1. server / scripts / docs 中统一使用：
   - `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD`
   - `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD`
2. `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` 作为初始密码。
3. `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` 作为最终管理员密码。
4. legacy fallback 如存在，必须输出 WARN。
5. clean DB 初始化时，实际生效的 initial password 会写入 runtime initial credentials file。
6. 环境变量提供 initial password 时，也会写入 credentials file。
7. DB/admin 已存在时，不会因环境变量存在覆盖 credentials file。
8. 支持从 runtime-dir 下 existing credentials file 读取 initial password。
9. credentials file 权限为 `0600`。
10. reset DB 后重新初始化会写入本次实际生效密码。

## 密码测试

必须通过：

```bash
go build ./cmd/server/
go test ./internal/...
```

Go 测试覆盖：

1. `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` 初始化 admin，并写入 credentials file。
2. legacy `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` fallback 初始化 admin，并写入 credentials file，同时 WARN。
3. existing credentials file 在没有 env 时被复用。
4. auto-generate 只在没有 env、没有 legacy、没有 credentials file 时发生，并写入 credentials file。
5. DB/admin 已存在时，不因 env 存在覆盖 credentials file。
6. credentials file 权限为 `0600`。
7. reset DB 后重新初始化会写入本次实际生效密码。

## auth-only 验收

命令：

```bash
export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='<initial-password>'
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='<final-password>'

bash scripts/lightai-bootstrap.sh --mode auth-only
```

验收：

1. server 可访问检查有结果。
2. agent 可访问检查有结果。
3. runtime-dir 检查有结果。
4. final password 可登录时直接 PASS。
5. final password 不可登录时，能读取 initial password。
6. `must_change_password=true` 时能完成改密。
7. 改密后能用 final password 重新登录。
8. 输出 `auth.json`。
9. `auth.json` 不含明文密码、token、cookie、CSRF。

## catalog-only 验收

命令：

```bash
bash scripts/lightai-bootstrap.sh --mode catalog-only
```

验收：

1. 完成 auth 前置步骤。
2. catalog 能找到 vLLM。
3. catalog 能找到 SGLang。
4. catalog 能找到 llama.cpp。
5. 输出 `catalog.json`。
6. 找不到时给出明确 FAIL 和 API response。

## models-only 验收

命令：

```bash
bash scripts/lightai-bootstrap.sh --mode models-only
```

验收：

1. 完成 catalog 前置步骤。
2. 检查 `/home/kzeng/models/Qwen3-0.6B-Instruct-2512` 存在。
3. 检查 `/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf` 存在。
4. 注册或确认 Qwen3 HF 模型。
5. 注册或确认 Qwen3.5 GGUF 模型。
6. 注册或确认 model locations。
7. 输出 `models.json` 和 `model-locations.json`。
8. 重复执行不重复创建脏数据。

## runtimes-only 验收

命令：

```bash
bash scripts/lightai-bootstrap.sh --mode runtimes-only
```

验收：

1. 完成 models 前置步骤。
2. 确认 node `KZ-LAPTOP` 存在且可用。
3. 配置 vLLM BackendRuntime 和 NodeBackendRuntime。
4. 配置 SGLang BackendRuntime 和 NodeBackendRuntime。
5. 配置 llama.cpp BackendRuntime 和 NodeBackendRuntime。
6. 保留参数 enabled + value。
7. 输出 `backend-runtimes.json` 和 `node-backend-runtimes.json`。
8. 重复执行可复用或更新已有资源。

## dry-run 验收

命令：

```bash
bash scripts/lightai-bootstrap.sh --mode dry-run
```

验收：

1. 完成 runtimes 前置步骤。
2. 对每个 NBR 执行 enable。
3. 对每个 NBR 执行 check-request。
4. 对 vLLM 执行 preflight / runplan dry-run。
5. 对 SGLang 执行 preflight / runplan dry-run。
6. 对 llama.cpp 执行 preflight / runplan dry-run。
7. 输出 `preflight-results.json`。
8. 输出 `bootstrap-state.json`。
9. `bootstrap-state.json` 包含 tenant_id、node_id、model ids、runtime ids、NBR ids、preflight results。
10. 不启动真实容器。

## full 验收

命令：

```bash
bash scripts/lightai-bootstrap.sh --mode full --allow-real-start
```

前置：

```yaml
bootstrap:
  allow_real_container_start: true
```

验收：

1. 同时要求 profile 允许和命令行允许。
2. 完成 dry-run 前置步骤。
3. 创建或确认 deployment。
4. 启动真实容器。
5. 检查 instance 状态。
6. 读取 logs。
7. 检查 health / models endpoint。
8. Chat completion 默认不执行。只有同时满足 `profile.bootstrap.allow_chat_completion=true` 和 `--allow-chat-completion` 时才执行。第一版实现 chat completion 为 opt-in。
9. 按 profile 配置停止或保留容器。
10. 输出 `full-results.json`。

## export 验收

命令：

```bash
bash scripts/lightai-bootstrap.sh --mode export
```

验收：

1. 输出 `configs/bootstrap/local-kz-laptop.yaml`。
2. 已存在文件时先备份。
3. profile 不包含密码、token、cookie、CSRF。
4. profile 包含 vLLM / SGLang / llama.cpp runtime。
5. profile 包含当前测试模型路径。
6. profile 保留参数 enabled + value。
7. 输出字段顺序稳定。
8. 输出 `export-summary.json`。
9. 输出 `export-resource-map.json`。
10. 输出 `export-warnings.json`。
11. 用 export profile 再跑 `dry-run` 能恢复相同测试环境。

## 输出文件验收

默认目录：

```text
/tmp/lightai/e2e/bootstrap/
```

必须根据 mode 输出对应文件：

- `bootstrap.log`
- `effective-config.json`
- `bootstrap-state.json`
- `auth.json`
- `runtime-dir-check.json`
- `catalog.json`
- `nodes.json`
- `models.json`
- `model-locations.json`
- `backend-runtimes.json`
- `node-backend-runtimes.json`
- `preflight-results.json`
- `full-results.json`
- `export-summary.json`
- `export-resource-map.json`
- `export-warnings.json`
- `errors.json`

## 发行版验收

运行项目现有打包命令，例如：

```bash
./scripts/package-release.sh
```

或：

```bash
./scripts/package-release-docker.sh
```

检查 artifact：

```bash
tar -tzf <release>.tar.gz | grep -E 'scripts/lightai-bootstrap.sh|configs/bootstrap|docs/engineering/bootstrap/lightai-bootstrap.md'
```

必须包含：

- `scripts/lightai-bootstrap.sh`
- `configs/bootstrap/bootstrap-profile.example.yaml`
- `configs/bootstrap/local-kz-laptop.yaml`
- `docs/engineering/bootstrap/lightai-bootstrap.md`

## 通用测试

```bash
go build ./cmd/server/
go build ./cmd/agent/
go test ./internal/...
cd web && npm test
cd web && npm run build
```

如果某个 batch 只改 server/auth，可只跑 Go 测试；最终 closeout 必须跑全量。
