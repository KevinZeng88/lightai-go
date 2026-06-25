# LightAI Bootstrap 工程设计索引

状态：`READY_FOR_CLAUDE_REVIEW`

本目录定义 LightAI Go 的环境自动初始化与环境导出能力。该能力不是一次性 repair，而是长期工程基础设施，用于开发机、测试环境、安装交付、发行版预配置和后续 Browser/API E2E。

## 背景

LightAI Go 当前开发验证存在几个重复成本：

1. 代码更新或 DB 重置后，需要手工重新登录、改密、注册模型、配置模型路径、启用 vLLM / SGLang / llama.cpp 运行环境。
2. 真实开发机已有可用模型和镜像，但这些环境信息没有可复用 profile。
3. 后续 Browser E2E 如果没有稳定的初始化数据，每次都要在 UI 或 API 中重复造数。
4. 管理员初始密码、最终密码、runtime initial credentials file、旧环境变量之间容易混用。
5. 安装交付时，如果没有统一 bootstrap 工具，客户环境初始化仍依赖人工步骤。

因此需要建立一套统一的 `lightai-bootstrap.sh` 能力：

- reset DB 后自动恢复本机开发测试环境；
- 从已配置环境导出 bootstrap profile；
- 安装后读取 YAML profile 自动初始化；
- 输出状态文件给 Browser E2E / API E2E 复用；
- 进入发行版，成为交付工具。

## 文档清单

| 文件 | 目的 |
|---|---|
| `00-index.md` | 总览、目录、阅读顺序、执行状态 |
| `01-password-contract-and-credentials-file.md` | 管理员初始密码、目标密码、credentials file 契约 |
| `02-bootstrap-tool-design.md` | `scripts/lightai-bootstrap.sh` 的命令、mode、参数、输出、幂等设计 |
| `03-profile-schema.md` | bootstrap profile schema、local-kz-laptop 示例、安装 profile 示例 |
| `04-export-mode-design.md` | 从当前环境导出 profile 的数据来源、安全规则、输出规则 |
| `05-packaging-and-installation.md` | 打包进入发行版、安装后初始化流程 |
| `06-acceptance-criteria.md` | 密码、auth、catalog、models、runtimes、dry-run、full、export、packaging 验收标准 |
| `07-execution-plan.md` | 分 batch 执行计划、测试命令、commit 条件、停止条件 |
| `08-closeout-template.md` | 最终 closeout 模板 |
| `CLAUDE_REVIEW_PROMPT.md` | 给 Claude 的审核入口 |

## 推荐阅读顺序

1. `00-index.md`
2. `01-password-contract-and-credentials-file.md`
3. `02-bootstrap-tool-design.md`
4. `03-profile-schema.md`
5. `04-export-mode-design.md`
6. `05-packaging-and-installation.md`
7. `06-acceptance-criteria.md`
8. `07-execution-plan.md`
9. `08-closeout-template.md`

Claude 审核时建议重点看：

- 密码环境变量契约是否与当前代码一致；
- credentials file 的写入规则是否安全、可审计、不覆盖真实状态；
- 各 mode 的边界是否清晰；
- profile schema 是否能覆盖当前本机 vLLM / SGLang / llama.cpp 场景；
- export 是否有足够安全保护；
- 打包脚本是否需要修改；
- 验收标准是否能防止“脚本看似成功但环境没有恢复”的问题。

## 推荐执行顺序

1. 文档审核与修订。
2. 密码契约与 credentials file server 侧修正。
3. `lightai-bootstrap.sh` 基础框架和短命令默认值。
4. `auth-only`。
5. `catalog-only`。
6. `models-only`。
7. `runtimes-only`。
8. `dry-run`。
9. `full`。
10. `export`。
11. 发行版打包集成。
12. 全量验收、evidence、closeout、commit、push。

## 当前范围

本设计要求最终实现所有 mode：

- `auth-only`
- `catalog-only`
- `models-only`
- `runtimes-only`
- `dry-run`
- `full`
- `export`

其中 `full` 必须同时满足 profile 配置和命令行显式允许才可以真实启动容器。

## 非目标

本专题不直接实现 Browser E2E。Browser E2E 后续读取本工具输出的：

```text
/tmp/lightai/e2e/bootstrap/bootstrap-state.json
```

本专题先解决可复用环境初始化、环境导出和安装初始化。

## 当前状态

**CLOSED** (2026-06-25)

所有 Batch 1-10 已完成。所有 7 个 mode 已实现并通过验证。详见 `bootstrap-final-closeout.md`。
