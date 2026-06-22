# 35 — Production Full-Flow E2E (Batch 4) Closeout

> Status: **REVISED** — Real runtime evidence for all 6 model types
> Baseline: commit `5cb70cf` (original programmatic)
> Date: 2026-06-23

## 1. Correction from Previous Version

Commit `5cb70cf` provided only programmatic validation (scan + compat JSON). That was NOT Production Full-Flow E2E. This revision adds **real container deployment, endpoint testing, logs, and cleanup** evidence.

## 2. Environment

| Item | Value |
|------|-------|
| Hostname | KZ-LAPTOP |
| GPU | NVIDIA GeForce RTX 5090 Laptop GPU (24 GiB) |
| Docker | 29.5.3 |
| vLLM image | vllm/vllm-openai:latest (31.8 GB) |
| SGLang image | lmsysorg/sglang:latest (41.6 GB) |
| llama.cpp image | ghcr.io/ggml-org/llama.cpp:server-cuda13 (5.25 GB) |

## 3. E2E Results — Final Status

| E2E | Model | Backend | Container | Endpoint | Status |
|-----|-------|---------|-----------|----------|--------|
| 1 | Qwen3-0.6B-Instruct-2512 (HF Chat) | vLLM | ✅ Started | ✅ /v1/models + /v1/chat/completions | **PASS** |
| 2 | Qwen3-0.6B-Instruct-2512 (HF Chat) | SGLang | ✅ Started | ✅ /v1/models + /v1/chat/completions | **PASS** |
| 3 | Qwen3.5-9B-Q4 (GGUF) | llama.cpp | ✅ Started | ✅ /v1/models + /v1/chat/completions | **PASS** |
| 4 | bge-small-zh-v1.5 (Embedding) | vLLM | ✅ Started | ✅ /v1/embeddings (vector returned) | **PASS** |
| 5 | bge-reranker-base (Reranker) | vLLM | ✅ Started | ✅ /v1/rerank (scores: 0.999 vs 0.001) | **PASS** |
| 6 | InternVL2_5-1B (VLM) | vLLM | ❌ Failed | ❌ Tokenizer error | **BACKEND_CAPABILITY_BLOCKED** |
| 7 | Wrong Combos | N/A | N/A | N/A | **PASS** (11/11) |

## 4. Runtime Evidence Details

### E2E-1: vLLM + HF Chat → PASS
- **Command**: `docker run -d --ipc host --shm-size 16gb --gpus "device=0" -v .../Qwen3-0.6B-Instruct-2512:/models/... -p 18001:8000 vllm/vllm-openai:latest --model /models/... --served-model-name Qwen3-0.6B-Instruct-2512`
- **Container**: Started, model loaded in ~30s, CUDA graphs compiled
- **/v1/models**: Returned Qwen3-0.6B-Instruct-2512
- **Chat**: "Hello! How can" (4-token response), finish_reason=length
- **Evidence**: e2e-1-docker-command.txt, e2e-1-v1-models.json, e2e-1-chat-response.json, e2e-1-logs.txt

### E2E-2: SGLang + HF Chat → PASS
- **Command**: `docker run -d --ipc host --shm-size 32gb --gpus "device=0" -v .../Qwen3-0.6B-Instruct-2512:/models/... -p 18002:30000 lmsysorg/sglang:latest python3 -m sglang.launch_server --model-path /models/...`
- **Container**: Started, model loaded in ~40s
- **/v1/models**: Returned model with id `/models/Qwen3-0.6B-Instruct-2512`
- **Chat**: "Hello! How can" — same 4-token response
- **Evidence**: e2e-2-docker-command.txt, e2e-2-model-endpoint.json, e2e-2-chat-or-completion-response.json

### E2E-3: llama.cpp + GGUF → PASS
- **Command**: `docker run -d --ipc host --shm-size 8gb --gpus "device=0" -v .../Qwen3.5-9B-Q4:/models/... -p 18004:8080 ghcr.io/ggml-org/llama.cpp:server-cuda13 -m /models/.../Qwen3.5-9B-Q4_K_M.gguf`
- **Key assertion**: `-m` points to concrete `.gguf` file, NOT directory
- **Container**: Started in <10s
- **/v1/models**: Returned Qwen3.5-9B-Q4_K_M.gguf
- **Chat**: Response received (8 tokens at 120 tok/s)
- **Evidence**: e2e-3-docker-command.txt, e2e-3-models.json, e2e-3-chat-response.json

### E2E-4: vLLM + Embedding → PASS
- **Command**: `docker run -d --ipc host --shm-size 8gb --gpus "device=0" -v .../bge-small-zh-v1.5:/models/... -p 18004:8000 vllm/vllm-openai:latest /models/bge-small-zh-v1.5 --served-model-name bge-small-zh-v1.5`
- **Note**: `--task embedding` is NOT supported in current vLLM; positional model arg used
- **/v1/embeddings**: Returned valid 512-dim embedding vector for "hello world"
- **Evidence**: e2e-4-docker-command.txt, e2e-4-embedding-response.json (11KB embedding)

### E2E-5: vLLM + Reranker → PASS
- **Command**: `docker run -d --ipc host --shm-size 8gb --gpus "device=0" -v .../bge-reranker-base:/models/... -p 18005:8000 vllm/vllm-openai:latest /models/bge-reranker-base --served-model-name bge-reranker-base`
- **/v1/rerank**: Returned relevance_score 0.999 for GPU doc, 0.001 for DB doc
- **Evidence**: e2e-5-docker-command.txt, e2e-5-rerank-response.json

### E2E-6: vLLM + VLM → BACKEND_CAPABILITY_BLOCKED
- **Reason**: vLLM image lacks sentencepiece/tiktoken dependency for InternVL2_5-1B tokenizer
- **Error**: `ValueError: Couldn't instantiate the backend tokenizer... You need to have sentencepiece or tiktoken installed`
- **Not a code defect**: This is an infrastructure/container image capability issue
- **Evidence**: e2e-6-chat-response-or-blocker.txt, e2e-6-logs.txt

### E2E-7: Wrong Combinations → PASS (11/11)
- 6 blocking combinations verified (format_mismatch, not_deployable)
- 5 passing combinations verified
- Evidence: e2e-7-wrong-combinations.json

## 5. Batch 1/2/3 Regression

| Gate | Status |
|------|--------|
| All agent/collector tests | ✅ PASS |
| All server/api tests | ✅ PASS |
| All server/runplan tests | ✅ PASS |
| go vet | ✅ CLEAN |
| npm test | ✅ PASS |
| npm run build | ✅ ✓ built |
| git diff --check | ✅ CLEAN |

## 6. Evidence Files

```
evidence/batch4-full-flow-e2e/
├── e2e-1-docker-command.txt
├── e2e-1-v1-models.json
├── e2e-1-chat-response.json
├── e2e-1-logs.txt
├── e2e-1-stop-cleanup.txt
├── e2e-2-docker-command.txt
├── e2e-2-model-endpoint.json
├── e2e-2-chat-or-completion-response.json
├── e2e-2-logs.txt
├── e2e-2-stop-cleanup.txt
├── e2e-3-docker-command.txt
├── e2e-3-models.json
├── e2e-3-chat-response.json
├── e2e-3-logs.txt
├── e2e-3-stop-cleanup.txt
├── e2e-4-docker-command.txt
├── e2e-4-embedding-response.json
├── e2e-4-logs.txt
├── e2e-4-stop-cleanup.txt
├── e2e-5-docker-command.txt
├── e2e-5-rerank-response.json
├── e2e-5-logs.txt
├── e2e-5-stop-cleanup.txt
├── e2e-6-chat-response-or-blocker.txt
├── e2e-6-logs.txt
├── e2e-6-stop-cleanup.txt
├── e2e-7-wrong-combinations.json
├── e2e-1-scan.json to e2e-6-scan.json (from programmatic phase)
```

## 7. Final Status

| Count | Status |
|-------|--------|
| 5 | ✅ PASS (real container + endpoint evidence) |
| 1 | BACKEND_CAPABILITY_BLOCKED (InternVL tokenizer) |
| 1 | PASS (E2E-7: compatibility checks) |
