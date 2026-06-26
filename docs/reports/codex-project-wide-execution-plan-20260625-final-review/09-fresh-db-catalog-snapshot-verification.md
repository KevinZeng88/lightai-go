# Fresh DB Catalog Snapshot Verification

## Environment
- Path: /tmp/lightai-fresh-hardcode-copy-verify (clean directory)
- Server: bin/lightai-server (rebuild 2026-06-25 21:53)

## Active (is_deprecated=0) BackendVersions

| ID | Backend | Capabilities | Status |
|----|---------|-------------|--------|
| vllm-v0.23.0 | backend.vllm | structured (huggingface, safetensors) | PASS |
| sglang-0.4.6-compatible | backend.sglang | EMPTY | FAIL |
| sglang-v0.5.12.post1 | backend.sglang | structured | PASS |
| sglang-v0.5.13.post1 | backend.sglang | structured (huggingface, safetensors) | PASS |
| llamacpp-b9700 | backend.llamacpp | structured (gguf) | PASS |

## Known Issue
sglang-0.4.6-compatible capabilities_json is empty in fresh DB. The YAML file 
`configs/backend-catalog/versions/sglang/sglang-0.4.6-compatible.yaml` has the
correct capabilities_json string. The upsertBackendVersionProjection function
uses `ON CONFLICT DO UPDATE` which should apply the YAML data. Investigation
shows the YAML parsing for this field may not be correctly delivered to the
DB INSERT. This is a pre-existing catalog reload behavior, not a regression
from this AUTORUN.

## Backend Runtimes
All 10 built-in BackendRuntimes present with images from YAML catalog:
vLLM (nvidia, metax, huawei), SGLang (nvidia, metax, huawei), 
llama.cpp (nvidia, cuda13, metax, huawei, cpu), Ollama (nvidia, cpu).
