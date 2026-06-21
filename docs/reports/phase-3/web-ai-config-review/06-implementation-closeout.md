# 06 - Implementation Closeout

> Status: CURRENT
> Scope: Web AI workflow presentation-only implementation closeout
> Date: 2026-06-21

## 1. Data Structure Boundary

- Database schema modified: no.
- Migration added: no.
- New persisted data structure added: no.
- Backend / BackendVersion / BackendRuntime / NodeBackendRuntime / ModelDeployment / ModelInstance core semantics changed: no.

The only backend behavior change is the existing model instance smoke-test endpoint honoring the already submitted `mode` and `prompt` request fields so the Web test entry can explicitly run Chat Completion or Text Completion. No schema or persistence contract changed.

## 2. Navigation Result

The Web sidebar now presents a user workflow first:

```text
模型运行 / Model Runtime
- 模型库 / Model Library
- 运行配置 / Runtime Configs
- 模型部署 / Model Deployments
- 模型实例 / Model Instances
- 测试与诊断 / Test & Diagnostics
```

Low-frequency system configuration entries moved under:

```text
配置 / Configuration
- 推理后端 / Inference Backends
- 运行模板 / Runtime Templates
```

No route or page was deleted. Existing `/backends`, `/runtimes`, `/runner-configs`, `/models/artifacts`, `/models/deployments`, and `/models/instances` routes remain available. `/models/test-diagnostics` is an additional workflow entry using the existing instance/test page.

## 3. Model Capability Display

Implemented `web/src/utils/modelCapabilities.js` and connected it to the model library and model scan wizard.

Capability display uses only existing fields:

- model name, display name, path, task type, format, architecture;
- `discovered_metadata_json` from model locations;
- optional existing `capabilities` / `capabilities_json` if a response includes them.

Inference rules implemented:

- explicit capabilities containing chat/completion/embedding/rerank/vision/tool/structured output are high confidence;
- tokenizer `chat_template` means Chat with high confidence;
- model name containing Instruct/Chat means Chat with medium confidence;
- CausalLM metadata means Completion;
- embedding/bge/e5/gte/sentence-transformers patterns mean Embedding;
- rerank/reranker/cross-encoder patterns mean Rerank;
- vision/vlm/multimodal patterns mean Vision.

Capability editing persistence is not implemented because no current ModelArtifact capability override API exists. This is formally tracked as `WEB-AI-FU-001` in `open-issues-closeout.md`.

## 4. Qwen3 Test Default

`Qwen3-0.6B-Instruct-2512` defaults to Chat Completion in frontend inference. The covered test expects:

```text
recommendedTestMode(Qwen3-0.6B-Instruct-2512) == chat
endpoint == /v1/chat/completions
```

The instance test dialog supports Auto, Chat Completion, and Text Completion. The server test endpoint now honors explicit `mode=chat` and `mode=completion`; Auto preserves Chat-first fallback behavior.

Error messages now include endpoint/status/summary via the model capability helper, for example:

```text
Chat Completion 请求失败：接口 /v1/chat/completions，HTTP 状态 404，错误摘要 ...
实例未运行：当前状态 stopped
模型未加载完成：/v1/models 未返回目标模型
```

## 5. NBR Structured Runtime Config

`RunnerConfigsPage.vue` now presents NodeBackendRuntime config by sections:

- 基础信息 / Basic info
- 镜像与命令 / Image & command
- 环境变量 / Environment
- 卷映射与端口 / Volumes & ports
- 设备与权限 / Devices & permissions
- 健康检查与预览 / Health check & preview
- 高级诊断 JSON / Advanced diagnostic JSON

Editable fields still save through existing API fields only:

- `display_name`
- `image_ref`
- `config_snapshot_json`

High-risk fields (`privileged`, host IPC, security options) show a warning. JSON remains available as a collapsed advanced diagnostic view, not the primary entry.

## 6. Deployment Page Enhancements

Deployment list now shows additional context using existing APIs:

- deployment name/status;
- model id;
- inference backend;
- backend version;
- node runtime config;
- image;
- node;
- endpoint;
- recent error.

RunPlan/dry-run dialogs now show a readable summary before advanced JSON:

- image;
- command;
- environment;
- volumes;
- ports;
- devices;
- health check;
- command preview.

Deployment-level overrides implemented only for existing fields:

- `service_json` host/container/app port fields;
- `env_overrides_json` key/value editor;
- `parameters_json` JSON editor.

Not implemented because it needs a first-class contract: extra volumes, arbitrary port list, endpoint alias, served model alias. These are formally tracked as `WEB-AI-FU-002` through `WEB-AI-FU-004`.

## 7. Instance Page Result

`ModelInstancesPage.vue` now:

- defaults the main list to hide `stopped`;
- provides a “显示已停止实例 / Show stopped instances” filter;
- keeps `failed` and other diagnostic states visible;
- keeps logs accessible through `current_run_plan_id`;
- replaces raw key/value detail with customer-readable sections:
  - 基础信息 / Basic info;
  - 运行信息 / Runtime info;
  - 诊断 / Diagnostics;
  - advanced JSON collapsed.

Raw JSON is no longer the main detail body.

## 8. i18n And Leakage Check

Fixed and covered:

- navigation keys for the new model workflow/config layout;
- model capability labels and source/confidence text;
- NBR section labels;
- deployment context/override labels;
- instance detail/test labels;
- GGUF/HuggingFace metadata headings and head-count labels;
- raw JSON moved to advanced collapsed areas.

`npm --prefix web test` includes i18n key consistency/missing-key checks and Web AI workflow leakage guards.

## 9. Formal Problem Closure

Unresolved problems remain only where current schema/API lacks a safe contract. All such items are in:

```text
docs/reports/phase-3/web-ai-config-review/open-issues-closeout.md
```

Statuses:

- `WEB-AI-FU-001`: DOCUMENTED_BLOCKER
- `WEB-AI-FU-002`: DOCUMENTED_BLOCKER
- `WEB-AI-FU-003`: DOCUMENTED_BLOCKER
- `WEB-AI-FU-004`: DOCUMENTED_BLOCKER
- `WEB-AI-FU-005`: DOCUMENTED_BLOCKER

No problem exists only in chat. No undocumented blocker is used to close this work.

## 10. Modified Files

Primary implementation files:

- `internal/server/api/deployment_lifecycle_handlers.go`
- `internal/server/api/ui_persistence_runplan_test.go`
- `web/src/utils/modelCapabilities.js`
- `web/tests/modelCapabilities.test.mjs`
- `web/tests/runtimeBoundaryUi.test.mjs`
- `web/src/layouts/ConsoleLayout.vue`
- `web/src/router/index.ts`
- `web/src/pages/ModelArtifactsPage.vue`
- `web/src/pages/RunnerConfigsPage.vue`
- `web/src/pages/ModelDeploymentsPage.vue`
- `web/src/pages/ModelInstancesPage.vue`
- `web/src/locales/zh-CN.ts`
- `web/src/locales/en-US.ts`
- `web/typings.d.ts`
- `web/package.json`

Documents:

- `docs/reports/phase-3/web-ai-config-review/05-implementation-review-findings.md`
- `docs/reports/phase-3/web-ai-config-review/open-issues-closeout.md`
- `docs/reports/phase-3/web-ai-config-review/06-implementation-closeout.md`

## 11. Verification Results

Executed:

```bash
gofmt -w cmd/ internal/
gofmt -w internal/server/api/deployment_lifecycle_handlers.go internal/server/api/ui_persistence_runplan_test.go
go test ./internal/server/api/...
go test ./internal/server/runplan/...
go vet ./...
npm --prefix web test
npm --prefix web build
npm --prefix web run build
bash -n scripts/e2e/*.sh scripts/e2e/lib/*.sh
bash -n scripts/e2e/lib/*.sh
git diff --check
git status --short
```

Results:

- `go test ./internal/server/api/...`: PASS.
- `go test ./internal/server/runplan/...`: PASS.
- `go vet ./...`: PASS.
- `npm --prefix web test`: PASS.
- `npm --prefix web build`: command syntax failed in current npm; npm reported `Unknown command: "build"` and suggested `npm run build`.
- `npm --prefix web run build`: PASS.
- `bash -n scripts/e2e/*.sh scripts/e2e/lib/*.sh`: failed because `scripts/e2e/*.sh` has no matching top-level files.
- `bash -n scripts/e2e/lib/*.sh`: PASS for existing E2E shell scripts.
- `git diff --check`: PASS.

Real hardware/model E2E was not run in this environment. The replacement validation is frontend tests, backend API tests, RunPlan tests, build, vet, and shell syntax checks. The blocked real-hardware style work is not introduced by this round.

## 12. Commit And Push

Implementation commit id: `ac45ae3`.

This closeout metadata update is committed separately because a commit cannot contain its own final hash.

`git status --short` before metadata update commit:

```text
 M docs/reports/phase-3/web-ai-config-review/06-implementation-closeout.md
```
