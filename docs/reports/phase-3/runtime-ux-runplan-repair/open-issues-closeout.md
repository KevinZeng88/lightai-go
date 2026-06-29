# Runtime UX / RunPlan Repair Open Issues Closeout

| ID | Issue | Evidence | Impact | Status | Fix Location | Verification | Final Decision |
| -- | ----- | -------- | ------ | ------ | ------------ | ------------ | -------------- |
| RUR-001 | Real API start smoke for vLLM/SGLang/llama.cpp was not executed in this run because the local LightAI server was not running on `127.0.0.1:18080`. | `curl -sS -m 2 http://127.0.0.1:18080/api/v1/health` returned connection refused. Docker 29.6.1, required images, and model paths were present. | Real container start remains unverified in this local session, but shared start-equivalent resolver output and command rendering are verified for all three backends. | DOCUMENTED_BLOCKER | Runtime environment, not source code. Start API uses `internal/server/api/deployment_lifecycle_handlers.go` and agent Docker execution uses `internal/agent/runtime/docker.go`. | Start server + agent, then run `bash scripts/e2e-real-smoke-all-three.sh`. | DOCUMENTED_BLOCKER |

