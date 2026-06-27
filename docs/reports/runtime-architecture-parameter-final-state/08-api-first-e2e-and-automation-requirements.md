# API-first E2E and Automation Requirements

## 1. 定位

API-first E2E 是本专题验收方式，用于证明 Runtime 架构、参数体系、RunPlan、Preflight、UI/API 行为已经形成可执行闭环。

自动化不是阶段主目标，验收必须覆盖架构和参数语义。

## 2. 基础链路

E2E 应覆盖：

1. clean runtime dir / fresh DB；
2. start server；
3. start agent；
4. login；
5. CSRF；
6. create or clone BackendRuntime；
7. enable NodeBackendRuntime；
8. check-request；
9. model scan；
10. create ModelArtifact / ModelLocation；
11. create Deployment；
12. Preflight；
13. RunPlan preview；
14. start deployment；
15. agent claim task；
16. Docker create/start；
17. health check；
18. instance running；
19. call OpenAI compatible endpoint；
20. fetch logs；
21. stop；
22. verify final state；
23. collect evidence；
24. non-zero failure on assertion failure。

## 3. 后端覆盖

至少覆盖：

1. vLLM；
2. SGLang；
3. llama.cpp；
4. NVIDIA real smoke；
5. MetaX dry-run / structure check；
6. missing image；
7. missing model path；
8. invalid parameter；
9. ready_with_warnings；
10. preview/spec consistency。

## 4. 参数语义 E2E

必须新增或增强参数语义断言：

1. 一个参数只有一个 schema definition；
2. override 引用 definition，不复制 schema；
3. default value 不导致 enabled=true；
4. required 参数不显示为用户 checked；
5. optional 参数默认不 checked；
6. advanced 参数默认折叠；
7. disabled input 仍有值；
8. unchecked optional 不进入当前层 override；
9. unchecked optional 不进入最终 args，除非 schema/resolver 明确 default-applied；
10. Deployment override 生效；
11. Deployment override 不复制 schema；
12. clone 保留 owner/key/value/enabled/source；
13. clone 不扩大 checked 范围；
14. copy-on-create 后上层修改不污染已有下层；
15. copy-on-create 后下层修改不污染上层；
16. RunPlan preview 显示 source map；
17. final args/env/mounts/ports/devices 带来源；
18. preview 与 Docker create spec 一致。

## 5. 页面/API 边界 E2E

必须断言：

1. Model API / 页面不提供 Docker 参数编辑；
2. Backend / BackendVersion API 不混入节点状态；
3. BackendRuntime API 提供模板参数；
4. NodeBackendRuntime API 提供节点配置和 evidence；
5. Deployment API 提供可覆盖参数和 RunPlan preview；
6. Instance API 只提供运行事实，不提供运行参数编辑；
7. ready_with_warnings 可部署；
8. needs_check / missing_image / failed / disabled 不可部署。

## 6. Preflight E2E

必须覆盖：

1. image inspect evidence；
2. model path evidence；
3. parameter validation；
4. device availability；
5. port availability；
6. mount validity；
7. health check validity；
8. warnings；
9. blocking errors；
10. API/UI error consistency。

## 7. Evidence 要求

证据统一进入：

```text
docs/reports/runtime-architecture-parameter-final-state/evidence/
```

至少保存：

```text
server.log
agent.log
api-requests.jsonl
api-responses.jsonl
preflight.json
check-request.json
runplan-preview.json
docker-create-spec.json
parameter-source-map.json
health-check.json
instance-final.json
container-logs.txt
summary.md
```

## 8. 建议脚本

建议新增或修复：

```text
scripts/e2e/e2e-runtime-architecture-parameter-full-chain.sh
scripts/e2e/e2e-runtime-parameter-ownership.sh
scripts/e2e/e2e-runtime-parameter-vllm.sh
scripts/e2e/e2e-runtime-parameter-sglang.sh
scripts/e2e/e2e-runtime-parameter-llamacpp.sh
```

脚本要求：

1. 可重复执行；
2. 失败非零退出；
3. 自动采集证据；
4. 不依赖手工 Docker 判断；
5. 不依赖 UI 手工操作。

## 9. 最终验收命令

基础命令：

```bash
go test ./internal/server/...
go test ./internal/agent/...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm run build
cd web && npm test
```

E2E 命令以实际脚本为准，必须在 closeout 中记录命令、结果和 evidence 路径。
