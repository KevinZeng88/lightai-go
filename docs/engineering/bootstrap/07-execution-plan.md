# 执行计划

状态：`READY_FOR_IMPLEMENTATION`

## 执行原则

1. 先修密码契约，再写 bootstrap 工具。
2. 先 API-first 初始化，再扩展 full 真实容器链路。
3. 每个 batch 有独立验收和 commit 条件。
4. 输出文件不含敏感信息。
5. 不把未完成能力伪装成 PASS。
6. 发现 API 缺口时，先记录并给出最小必要补充，不做无边界重构。
7. 默认在当前分支/main 执行，不新建分支。

## Batch 0：文档审核修订

目标：

- 审核 `docs/engineering/bootstrap/` 下设计文档。
- 修正与现有代码不一致的路径、接口、密码文件格式。

涉及文件：

- `docs/engineering/bootstrap/*.md`

必做：

- grep 当前代码确认 auth/bootstrap 逻辑；
- 确认 initial credentials file 路径和格式；
- 确认登录/改密 API；
- 确认 package-release 脚本名称。

验收：

- 文档与代码真实路径/API 对齐；
- 输出需要用户确认的问题，如有。

Commit 条件：

- 文档一致性完成。

## Batch 1：密码契约与 credentials file server 侧修正

**状态：实质完成**（2026-06-25，commits `d3c6e98`、`4aefea1`）

已实现内容：

- server 侧 5-step 密码解析优先级（cfg.Password → INITIAL_PASSWORD → ADMIN_PASSWORD WARN → credentials file → auto-generate）
- `readPasswordFromCredentialsFile()` 函数
- `BootstrapConfig.InitialPasswordEnv` 字段
- credentials file 复用、不覆盖、0600 权限
- legacy `ADMIN_PASSWORD` fallback with WARN
- `cmd/server/main.go`、`scripts/start-server.sh`、`scripts/e2e/lib/env.sh` 已更新
- `docs/engineering/bootstrap/lightai-bootstrap.md` 已同步

待完成（本 batch scope 外）：

- password-env-audit.txt refresh（已完成，`4aefea1`）
- Go 测试覆盖 8 cases（已完成，`bootstrap_test.go`）

涉及文件（已修改）：

- `cmd/server/main.go`
- `internal/server/auth/bootstrap.go`
- `internal/server/auth/bootstrap_test.go`（新增）
- `scripts/start-server.sh`
- `scripts/e2e/lib/env.sh`
- `docs/engineering/bootstrap/lightai-bootstrap.md`
- `docs/engineering/bootstrap/password-env-audit.txt`

测试：

```bash
go build ./cmd/server/
go test ./internal/...
```

验收：

- `06-acceptance-criteria.md` 中密码部分通过。

Commit message 建议：

```text
fix(auth): unify bootstrap password contract and credentials file
```

## Batch 2：bootstrap 脚本框架与配置解析

目标：

- 新增 `scripts/lightai-bootstrap.sh`。
- 实现短命令默认值、profile 读取、参数合并、输出目录、脱敏日志。

涉及文件：

- `scripts/lightai-bootstrap.sh`
- `configs/bootstrap/bootstrap-profile.example.yaml`
- `configs/bootstrap/local-kz-laptop.yaml`
- `docs/engineering/bootstrap/lightai-bootstrap.md`

修改要点：

- 参数优先级；
- 默认 profile；
- 默认 mode；
- runtime-dir / output-dir；
- effective-config.json；
- bootstrap.log；
- errors.json；
- 敏感信息脱敏。

测试：

```bash
bash -n scripts/lightai-bootstrap.sh
bash scripts/lightai-bootstrap.sh --help
bash scripts/lightai-bootstrap.sh --mode auth-only
```

Commit message：

```text
feat(bootstrap): add bootstrap cli and profile defaults
```

## Batch 3：auth-only

目标：

- 实现 server/agent 检查、runtime-dir 检查、登录、初始密码读取、首次改密、重新登录。

涉及文件：

- `scripts/lightai-bootstrap.sh`
- auth helper，如有
- docs

修改要点：

- final password 先登录；
- initial password fallback；
- runtime credentials file 自动发现；
- must_change_password；
- change password API；
- auth.json。

测试：

```bash
export LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD='<initial-password>'
export LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD='<final-password>'
bash scripts/lightai-bootstrap.sh --mode auth-only
```

验收：

- auth-only PASS；
- `auth.json` 脱敏。

Commit message：

```text
feat(bootstrap): implement auth-only initialization
```

## Batch 4：catalog-only / models-only

目标：

- catalog 检查；
- 模型路径检查；
- model artifact / location 注册或确认。

涉及文件：

- `scripts/lightai-bootstrap.sh`
- docs/profile

修改要点：

- API-first；
- vLLM / SGLang / llama.cpp catalog；
- Qwen3 HF 模型；
- Qwen3.5 GGUF 模型；
- 幂等查找与更新。

测试：

```bash
bash scripts/lightai-bootstrap.sh --mode catalog-only
bash scripts/lightai-bootstrap.sh --mode models-only
```

Commit message：

```text
feat(bootstrap): initialize catalog and model locations
```

## Batch 5：runtimes-only

目标：

- 创建或确认 BackendRuntime / NodeBackendRuntime。

涉及文件：

- `scripts/lightai-bootstrap.sh`
- configs/bootstrap
- docs

修改要点：

- node 查找；
- vLLM runtime；
- SGLang runtime；
- llama.cpp runtime；
- 参数 enabled + value；
- 幂等更新。

测试：

```bash
bash scripts/lightai-bootstrap.sh --mode runtimes-only
```

Commit message：

```text
feat(bootstrap): initialize backend and node runtimes
```

## Batch 6：dry-run

目标：

- enable/check/preflight/runplan dry-run。

涉及文件：

- `scripts/lightai-bootstrap.sh`
- docs

修改要点：

- NBR enable；
- check-request；
- preflight；
- runplan dry-run；
- bootstrap-state.json。

测试：

```bash
bash scripts/lightai-bootstrap.sh --mode dry-run
```

Commit message：

```text
feat(bootstrap): add dry-run preflight and runplan validation
```

## Batch 7：full

目标：

- 显式允许时执行真实 deployment/start/logs/health/stop。

涉及文件：

- `scripts/lightai-bootstrap.sh`
- configs/bootstrap
- docs

修改要点：

- 双重允许：profile + `--allow-real-start`；
- 创建 deployment；
- 启动实例；
- logs；
- health / models endpoint；
- stop 或保留。

测试：

```bash
bash scripts/lightai-bootstrap.sh --mode full --allow-real-start
```

Commit message：

```text
feat(bootstrap): add guarded full deployment validation
```

## Batch 8：export

目标：

- 从当前环境导出 profile。

涉及文件：

- `scripts/lightai-bootstrap.sh`
- configs/bootstrap
- docs

修改要点：

- API 读取资源；
- key 生成；
- YAML 稳定输出；
- 脱敏；
- backup；
- export summary/resource map/warnings。

测试：

```bash
bash scripts/lightai-bootstrap.sh --mode export
bash scripts/lightai-bootstrap.sh --mode dry-run
```

Commit message：

```text
feat(bootstrap): export current environment profile
```

## Batch 9：发行版集成

目标：

- release artifact 包含 bootstrap 工具、profile、文档。

涉及文件：

- `scripts/package-release.sh` — 添加以下目录和文件的拷贝：
  ```bash
  # Copy bootstrap tool
  cp scripts/lightai-bootstrap.sh "$BUILD_DIR/scripts/"
  # Copy bootstrap profiles
  mkdir -p "$BUILD_DIR/configs/bootstrap"
  cp -r configs/bootstrap/* "$BUILD_DIR/configs/bootstrap/"
  # Copy bootstrap documentation
  mkdir -p "$BUILD_DIR/docs/engineering/bootstrap"
  cp docs/engineering/bootstrap/lightai-bootstrap.md "$BUILD_DIR/docs/engineering/bootstrap/"
  ```
- `scripts/package-release-docker.sh` — 确认 Docker 构建也包含上述文件

release artifact 必须包含的 4 个路径：

```text
scripts/lightai-bootstrap.sh
configs/bootstrap/bootstrap-profile.example.yaml
configs/bootstrap/local-kz-laptop.yaml
docs/engineering/bootstrap/lightai-bootstrap.md
```

测试：

```bash
./scripts/package-release.sh
# 或 ./scripts/package-release-docker.sh
```

验证：

```bash
tar -tzf <release>.tar.gz | grep -E 'scripts/lightai-bootstrap.sh|configs/bootstrap/|docs/engineering/bootstrap/lightai-bootstrap.md'
# 输出必须包含上述 4 个路径
```

Commit message：

```text
chore(package): include bootstrap tool and profiles
```

## Batch 10：最终回归与 closeout

目标：

- 全量测试；
- evidence；
- closeout；
- push。

测试：

```bash
go build ./cmd/server/
go build ./cmd/agent/
go test ./internal/...
cd web && npm test
cd web && npm run build
bash scripts/lightai-bootstrap.sh --mode auth-only
bash scripts/lightai-bootstrap.sh --mode catalog-only
bash scripts/lightai-bootstrap.sh --mode models-only
bash scripts/lightai-bootstrap.sh --mode runtimes-only
bash scripts/lightai-bootstrap.sh --mode dry-run
bash scripts/lightai-bootstrap.sh --mode export
```

如环境允许：

```bash
bash scripts/lightai-bootstrap.sh --mode full --allow-real-start
```

Commit message：

```text
docs(bootstrap): close bootstrap tool evidence and status
```

## 停止条件

遇到以下情况停止并汇报：

1. 现有 API 不支持必要初始化动作，需要新增后端接口；
2. 登录/改密 API 与预期不一致；
3. credentials file 规则会破坏现有安全设计；
4. full 模式需要大规模重构 deployment lifecycle；
5. export 无法通过 API 获取关键资源且需要直接 DB 强依赖；
6. 测试失败超过两轮仍未定位；
7. 发现新的 P0 安全问题。

## 最终输出要求

1. 新增/修改文件列表；
2. 密码环境变量统一结果；
3. credentials file 写入规则；
4. bootstrap mode 支持情况；
5. 默认命令行为；
6. 各 mode 运行结果；
7. export 运行结果；
8. 输出文件路径；
9. 发行版打包验证结果；
10. commit id；
11. push result；
12. git status --short。
