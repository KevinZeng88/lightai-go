# LightAI Bootstrap Closeout Template

状态：`TEMPLATE`

## Final Status

可选值：

- `CLOSED`
- `PARTIAL`
- `BLOCKED`
- `REOPENED`

当前状态：`<填入状态>`

## 修复 / 实现摘要

| 项目 | 结果 | 说明 |
|---|---|---|
| 密码契约统一 | PASS / FAIL / PARTIAL |  |
| credentials file 写入规则 | PASS / FAIL / PARTIAL |  |
| lightai-bootstrap.sh | PASS / FAIL / PARTIAL |  |
| profile schema | PASS / FAIL / PARTIAL |  |
| auth-only | PASS / FAIL / PARTIAL |  |
| catalog-only | PASS / FAIL / PARTIAL |  |
| models-only | PASS / FAIL / PARTIAL |  |
| runtimes-only | PASS / FAIL / PARTIAL |  |
| dry-run | PASS / FAIL / PARTIAL |  |
| full | PASS / FAIL / PARTIAL / NOT_RUN |  |
| export | PASS / FAIL / PARTIAL |  |
| packaging | PASS / FAIL / PARTIAL |  |

## 密码环境变量最终契约

| 变量 | 用途 | 状态 |
|---|---|---|
| `LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD` | 初始管理员密码 |  |
| `LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD` | 最终管理员密码 |  |

旧变量 / legacy fallback：

| 变量 | 处理 | WARN | 说明 |
|---|---|---|---|
|  |  |  |  |

## Credentials File 规则确认

| 验收项 | 结果 | Evidence |
|---|---|---|
| clean DB 初始化写入实际 initial password |  |  |
| env initial password 写入 credentials file |  |  |
| legacy fallback 写入 credentials file 并 WARN |  |  |
| existing credentials file 可复用 |  |  |
| DB/admin 已存在不覆盖 credentials file |  |  |
| 文件权限 0600 |  |  |
| 日志不泄露密码 |  |  |

实际路径：

```text
<runtime initial credentials file path>
```

实际格式：

```text
<credentials file format>
```

## Mode 验收表

| Mode | 命令 | 结果 | 输出文件 | 说明 |
|---|---|---|---|---|
| auth-only | `bash scripts/lightai-bootstrap.sh --mode auth-only` |  |  |  |
| catalog-only | `bash scripts/lightai-bootstrap.sh --mode catalog-only` |  |  |  |
| models-only | `bash scripts/lightai-bootstrap.sh --mode models-only` |  |  |  |
| runtimes-only | `bash scripts/lightai-bootstrap.sh --mode runtimes-only` |  |  |  |
| dry-run | `bash scripts/lightai-bootstrap.sh --mode dry-run` |  |  |  |
| full | `bash scripts/lightai-bootstrap.sh --mode full --allow-real-start` |  |  |  |
| export | `bash scripts/lightai-bootstrap.sh --mode export` |  |  |  |

## 输出文件索引

默认目录：

```text
/tmp/lightai/e2e/bootstrap/
```

| 文件 | 是否生成 | 说明 |
|---|---|---|
| bootstrap.log |  |  |
| effective-config.json |  |  |
| bootstrap-state.json |  |  |
| auth.json |  |  |
| runtime-dir-check.json |  |  |
| catalog.json |  |  |
| nodes.json |  |  |
| models.json |  |  |
| model-locations.json |  |  |
| backend-runtimes.json |  |  |
| node-backend-runtimes.json |  |  |
| preflight-results.json |  |  |
| full-results.json |  |  |
| export-summary.json |  |  |
| export-resource-map.json |  |  |
| export-warnings.json |  |  |
| errors.json |  |  |

## Export 结果

输出 profile：

```text
<output profile path>
```

备份文件：

```text
<backup path>
```

资源数量：

| 类型 | 数量 |
|---|---:|
| tenants |  |
| nodes |  |
| models |  |
| model locations |  |
| backend runtimes |  |
| node backend runtimes |  |
| deployments |  |
| warnings |  |

安全检查：

| 检查项 | 结果 |
|---|---|
| 不包含密码 |  |
| 不包含 token |  |
| 不包含 cookie |  |
| 不包含 CSRF |  |
| env_json 脱敏 |  |

## Packaging 结果

打包命令：

```bash
<package command>
```

artifact：

```text
<artifact path>
```

包含检查：

| 文件 | 结果 |
|---|---|
| `scripts/lightai-bootstrap.sh` |  |
| `configs/bootstrap/bootstrap-profile.example.yaml` |  |
| `configs/bootstrap/local-kz-laptop.yaml` |  |
| `docs/engineering/bootstrap/lightai-bootstrap.md` |  |

## 测试结果

| 命令 | 结果 | 说明 |
|---|---|---|
| `go build ./cmd/server/` |  |  |
| `go build ./cmd/agent/` |  |  |
| `go test ./internal/...` |  |  |
| `cd web && npm test` |  |  |
| `cd web && npm run build` |  |  |
| `bash scripts/lightai-bootstrap.sh --mode auth-only` |  |  |
| `bash scripts/lightai-bootstrap.sh --mode catalog-only` |  |  |
| `bash scripts/lightai-bootstrap.sh --mode models-only` |  |  |
| `bash scripts/lightai-bootstrap.sh --mode runtimes-only` |  |  |
| `bash scripts/lightai-bootstrap.sh --mode dry-run` |  |  |
| `bash scripts/lightai-bootstrap.sh --mode export` |  |  |
| `bash scripts/lightai-bootstrap.sh --mode full --allow-real-start` |  |  |

## 未关闭问题

| 问题 | 影响 | 处理建议 | 状态 |
|---|---|---|---|
|  |  |  |  |

## Deferred Items

| Item | 原因 | 风险 | 后续触发条件 |
|---|---|---|---|
|  |  |  |  |

## Commit 列表

| Commit | Message | 说明 |
|---|---|---|
|  |  |  |

## Push 结果

```text
<push result>
```

## git status

```text
<git status --short>
```

## 最终结论

```text
FINAL_STATUS=<CLOSED|PARTIAL|BLOCKED|REOPENED>
```
