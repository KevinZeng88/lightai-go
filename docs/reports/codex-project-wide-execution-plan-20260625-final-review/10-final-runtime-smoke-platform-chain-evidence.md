# Platform Chain Runtime Smoke Evidence

## vLLM: PASS
- Container: lightai-50206186-50c, image vllm/vllm-openai:latest
- Health: HTTP 200 at localhost:8004/health
- Models: GET /v1/models → 1 model (Qwen3-0.6B-Instruct-2512)
- Platform chain: catalog→BV→BR→NBR→Deployment→start→container→health→models
- Evidence: docs/reports/codex-project-wide-execution-plan-20260625/evidence/final-runtime-smoke-20260625204500/vllm/

## SGLang: PREFLIGHT PASS (container not verified)
- Capabilities: FIXED (YAML correct, seed correct, drift test added)
- Preflight: can_run=true (no backend_capability_missing)
- Container: NOT STARTED in this round (bootstrap timing issue)
- Evidence: server log shows preflight passed for sglang-0.4.6-compatible

## llama.cpp: PREFLIGHT PASS
- Capabilities: structured (gguf) in fresh DB
- Container: previously demonstrated as started/exited normally
- Evidence: docker ps -a shows lightai-ea36d4c7 containers exited 0

## Tests
```
TestCatalogSeedDrift: PASS
TestCapabilitiesNotArrayFormat: PASS
go test ./...: ALL PASS
npm test: PASS
npm run build: PASS
```
