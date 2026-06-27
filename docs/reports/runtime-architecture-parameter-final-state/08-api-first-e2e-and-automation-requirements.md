# API-first E2E and Automation Requirements

## 1. 定位

API-first E2E 是本阶段的验收方式。目标是证明 Runtime 架构与参数体系可以通过 API 自动完成完整运行链路。

## 2. 基本要求

E2E 脚本必须做到：

1. 可在 fresh DB 下运行；
2. 自动启动 server；
3. 自动启动 agent；
4. 自动登录；
5. 自动处理 CSRF；
6. 自动准备 BackendRuntime；
7. 自动 enable NodeBackendRuntime；
8. 自动执行 check-request；
9. 自动扫描或创建模型；
10. 自动创建 Deployment；
11. 自动执行 Preflight；
12. 自动获取 RunPlan preview；
13. 自动启动实例；
14. 自动等待 health；
15. 自动获取 logs；
16. 自动停止实例；
17. 自动验证 final state；
18. 自动保存 evidence；
19. 任一步失败退出非零。

## 3. 建议脚本路径

```text
scripts/e2e/e2e-runtime-architecture-parameter-full-chain.sh
scripts/e2e/e2e-runtime-parameter-vllm.sh
scripts/e2e/e2e-runtime-parameter-sglang.sh
scripts/e2e/e2e-runtime-parameter-llamacpp.sh
```

## 4. 证据目录

所有证据保存到：

```text
docs/reports/runtime-architecture-parameter-final-state/evidence/
```

建议每次运行创建子目录：

```text
evidence/YYYYMMDD-HHMMSS-backend-name/
```

## 5. 必备证据文件

每次 E2E 至少保存：

```text
summary.md
server.log
agent.log
api-requests.jsonl
api-responses.jsonl
backend-runtime.json
node-backend-runtime-before-check.json
check-request.json
node-backend-runtime-after-check.json
model-artifact.json
model-location.json
deployment-created.json
preflight.json
runplan-preview.json
docker-create-spec.json
runplan-docker-diff.json
instance-running.json
health-check.json
container-logs.txt
stop-response.json
instance-final.json
git-status.txt
```

## 6. Full-chain 场景

### 6.1 vLLM

必须验证：

1. image inspect；
2. HuggingFace model path；
3. parameter override；
4. `--gpu-memory-utilization`；
5. `--max-model-len`；
6. `/v1/models`；
7. `/v1/chat/completions`；
8. logs；
9. stop；
10. RunPlan vs Docker spec。

### 6.2 SGLang

必须验证：

1. image inspect；
2. HuggingFace model path；
3. `--mem-fraction-static`；
4. `--context-length`；
5. `/v1/models`；
6. `/v1/chat/completions`；
7. logs；
8. stop；
9. RunPlan vs Docker spec。

如果本机镜像或模型导致真实运行失败，仍需保存失败 evidence，且区分代码缺陷与环境限制。

### 6.3 llama.cpp

必须验证：

1. image inspect；
2. GGUF model path；
3. `--ctx-size`；
4. `--n-gpu-layers`；
5. `/v1/models`；
6. `/v1/chat/completions`；
7. logs；
8. stop；
9. RunPlan vs Docker spec。

## 7. Negative cases

必须覆盖：

1. missing image；
2. missing model path；
3. invalid parameter type；
4. invalid parameter range；
5. port conflict；
6. unsupported model format；
7. NBR needs_check；
8. NBR missing_image；
9. NBR failed；
10. backend_runtime_id 被拒绝。

## 8. ready_with_warnings

必须覆盖：

1. NBR 状态 ready_with_warnings；
2. UI/API 可选择部署；
3. Preflight 继续执行；
4. Deployment 可创建；
5. warnings 保存；
6. evidence 保存。

## 9. RunPlan vs Docker spec 对比

脚本必须获取：

1. API RunPlan preview；
2. Agent Docker create spec evidence；
3. diff result。

必须对比字段：

1. image；
2. args；
3. env；
4. ports；
5. mounts；
6. devices；
7. gpus；
8. health check；
9. labels。

允许忽略：

1. Docker SDK 自动生成的 id；
2. 创建时间；
3. 默认 network 字段；
4. 空字段格式差异。

忽略规则必须写入 `runplan-docker-diff.json`。

## 10. API request 记录

每个 API 调用记录：

```json
{
  "time": "2026-06-27T00:00:00Z",
  "method": "POST",
  "path": "/api/v1/...",
  "request_body_file": "request-001.json",
  "response_body_file": "response-001.json",
  "status": 200
}
```

## 11. 最终 summary.md

每次 E2E summary 必须包含：

1. backend；
2. model path；
3. image；
4. node；
5. NBR id；
6. deployment id；
7. instance id；
8. container id；
9. endpoint；
10. health result；
11. chat result；
12. logs result；
13. stop result；
14. RunPlan/Docker diff result；
15. PASS/FAIL；
16. failure reason。
