# LightAI Go ConfigSet 重构交接包

本目录用于交给 Codex 执行 LightAI Go 配置模型重构。

## 目标

将 LightAI Go 现有 Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / Deployment / RunPlan 链路，统一改造成 `ConfigSet / ConfigItem` 模型，彻底去除 backend catalog 硬编码、旧字段、旧 API、旧兼容逻辑和过时文档口径。

## 文件

| 文件 | 用途 |
|---|---|
| `01-current-code-findings.md` | 基于 GitHub 当前代码的关键问题与证据 |
| `02-configset-configitem-design.md` | 正式设计文档 |
| `03-implementation-plan.md` | 分阶段实施计划 |
| `04-validation-and-acceptance.md` | 验证矩阵与验收标准 |
| `05-codex-execution-prompt.md` | 可直接交给 Codex 的执行 prompt |
| `manifest.json` | 文件清单 |

## 关键原则

1. 不兼容旧 DB / 旧 API / 旧 payload / 旧 UI / 旧文档。
2. 旧结构、旧字段、旧 seed、旧 route、旧文档口径必须删除或归档。
3. `configs/config-registry/` + `configs/backend-catalog/` 是唯一权威来源。
4. `internal/server/db/db.go` 不再维护 vLLM / SGLang / llama.cpp 的 catalog literal。
5. 所有层级统一保存 `config_set_json` + `source_metadata_json`。
6. RunPlan / AgentRunSpec / DockerSpec 只从最终 ConfigSet snapshot 渲染。
7. 常用参数结构化，未知参数走 `backend.extra_args` / `runtime.extra_env` / `launcher.extra_options`。
8. 校验偏宽松，尽量跑通；重复结构化参数、模型路径缺失、镜像缺失等明确无法运行问题才 error。
