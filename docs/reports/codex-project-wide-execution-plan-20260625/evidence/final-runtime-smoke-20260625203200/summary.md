# Runtime Smoke Summary

## vLLM
- container: started (lightai-50206186-50c)
- health: PASS (http://localhost:8004/health → 200)
- /v1/models: PASS (Qwen3-0.6B-Instruct-2512)
- inference: PASS (warmup delay ~2min)
- stop: PASS

## SGLang
- container: started then stopped (keep=false default)
- deployment start task claimed by agent
- stop: PASS (automatic cleanup)

## llama.cpp
- container: started then stopped (keep=false default)
- deployment start task claimed by agent
- stop: PASS (automatic cleanup)

## Post-Smoke
- All containers stopped
- No residual instances
