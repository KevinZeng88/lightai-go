# Runtime Architecture and Parameter Final-State Closeout

## 1. Final Status

- Status:
- Final commit:
- Push result:
- git status:

## 2. Scope Completed

- Runtime domain contract:
- Parameter ownership:
- Copy-on-create snapshot chain:
- RuntimeRequirements:
- BackendCapabilityProfile:
- RunPlan / Preflight:
- UI/API:
- API-first E2E:

## 3. Runtime Domain Contract Result

记录：

1. Backend / BackendVersion 硬件无关结果；
2. ModelArtifact / ModelLocation 边界；
3. BackendRuntime / NodeBackendRuntime 边界；
4. Deployment 边界；
5. Instance 运行事实边界；
6. NodeBackendRuntime 唯一部署入口验证。

## 4. Parameter Ownership Final Checklist

必须逐项填写结果和证据：

| Item | Result | Evidence |
|---|---|---|
| 一个参数只有一个 owner |  |  |
| 一个参数只有一个 schema 定义位置 |  |  |
| Override 引用 definition |  |  |
| Deployment 只保存 override |  |  |
| UI 不复制 schema |  |  |
| copy-on-create 上层到下层 |  |  |
| 上层修改不污染已有下层 |  |  |
| 下层修改不污染上层 |  |  |
| clone 不扩大 checked |  |  |
| RunPlan source map 存在 |  |  |

## 5. Parameter Display Checklist

| Item | Result | Evidence |
|---|---|---|
| Model 页面不展示 Docker 参数 |  |  |
| Backend 页面不展示节点状态 |  |  |
| BackendRuntime 页面只展示模板参数 |  |  |
| NodeBackendRuntime 页面只展示节点运行环境和 evidence |  |  |
| Deployment 页面展示 override 和 RunPlan preview |  |  |
| Instance 页面只展示运行事实 |  |  |
| 参数按 category 分组 |  |  |
| advanced 默认折叠 |  |  |
| disabled input 显示值 |  |  |

## 6. checked / enabled Checklist

| Item | Result | Evidence |
|---|---|---|
| default value 不等于 enabled |  |  |
| required 不等于用户 checked |  |  |
| optional 默认不 checked |  |  |
| advanced 默认不 checked |  |  |
| unchecked optional 不进入 override |  |  |
| unchecked optional 不进入 final args |  |  |
| current-layer override 才 checked |  |  |

## 7. RuntimeRequirements Result

记录：

1. image requirements；
2. Docker runtime requirements；
3. model path requirements；
4. device binding requirements；
5. port/mount/env requirements；
6. health check requirements；
7. warning/blocking error；
8. vLLM/SGLang/llama.cpp coverage。

## 8. BackendCapabilityProfile Result

记录：

1. model formats；
2. protocols；
3. endpoints；
4. parameter capabilities；
5. resource controls；
6. health checks；
7. device binding abstraction；
8. NVIDIA/MetaX/Huawei boundary。

## 9. RunPlan / Preflight Result

记录：

1. Preflight errors/warnings；
2. check-request evidence；
3. parameter_source_map；
4. final args/env/mounts/ports/devices；
5. preview 与 Docker spec 对比；
6. health check 与端口一致性；
7. vLLM/SGLang/llama.cpp mapping。

## 10. UI/API Result

记录：

1. changed pages/components；
2. API response changes；
3. RuntimeParameterEditor behavior；
4. RunnerConfigsPage cleanup；
5. Deployment page behavior；
6. Instance logs/status behavior。

## 11. API-first E2E Evidence

Evidence directory:

```text
docs/reports/runtime-architecture-parameter-final-state/evidence/
```

Files:

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

## 12. Test Results

Commands and results:

```bash
go test ./internal/server/...
go test ./internal/agent/...
go build ./cmd/server/...
go build ./cmd/agent/...
cd web && npm run build
cd web && npm test
```

E2E commands:

```bash
# fill actual commands
```

## 13. Deleted Legacy Logic

List removed fields, compatibility branches, old UI entry points, old fallback logic.

## 14. Open Issues

Only include issues that are currently not verifiable or out of scope due to external dependencies.

Each item must include:

1. issue；
2. reason not fixed；
3. impact；
4. verification condition；
5. suggested next action。

## 15. Commits

```text
<commit> <message>
```

## 16. Final git status

```bash
git status --short
```

## 17. Final Conclusion

State whether Runtime architecture and parameter final-state is closed.
