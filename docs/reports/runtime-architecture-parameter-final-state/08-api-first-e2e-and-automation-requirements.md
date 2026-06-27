# API-first E2E and Automation Requirements

## 1. 目的

自动化是验收方式，不是主目标。本阶段验证 ConfigSetBundle 最终模型。

## 2. Go/API contract tests

必须覆盖 ConfigSetBundle copy-on-create、schema/snapshot 只读、value/state 当前层可修改、owner 不因 copy 改变、default != enabled、required != checked、inherited != checked、optional unchecked 不进入 RunPlan、Docker optional unchecked 不进入 HostConfig、local_edits 记录、provenance/source_chain 记录、parent 修改不污染 child、child 修改不污染 parent、clone 不扩大 checked/enabled、shared RunPlan builder 被 preview/preflight/start 共用。

## 3. Web tests

Vitest/component tests 覆盖 ConfigSetRenderer sections、child_slots 调用 child ConfigSet view、required/common/advanced 分组、advanced 默认折叠、local edits summary、inherited/local/effective value 展示、checked/enabled 只表示当前层 local edit、custom renderer obeys ConfigItem contract、Model 页面不展示 Docker 参数、Instance 页面不编辑 ConfigSet。

## 4. API-first E2E

E2E 至少覆盖 fresh DB、server/agent start、login/CSRF、BackendVersion seed、BackendRuntime create、NodeBackendRuntime enable + check、ModelArtifact/ModelLocation、Deployment create、RunPlan preview、Preflight、start、health check、logs、stop、evidence、non-zero failure。

## 5. RunPlan evidence

每次 E2E 必须保存 resolved_run_plan.json、parameter_source_map.json、docker_spec_expected.json、docker_spec_actual.json、preflight.json、health.json、logs_excerpt.txt。

## 6. Fresh DB 策略

本阶段允许删除并重建 `/tmp/lightai/data/lightai.db`。旧 DB、旧 snapshot、旧 API 字段不做兼容。catalog/registry 如继续存在，必须按最终 ConfigSetBundle / ConfigSet / ConfigItem 模型重建，不作为旧兼容层。

## 7. 不允许的验收方式

不只靠 UI 手工截图；不只靠 Docker 命令手动观察；不用前端传入 image_present 作为权威；不让 preview 和 start 走不同构建逻辑；不让 source_map 只是 UI 假数据；不保留“以后处理”的可修复问题。
