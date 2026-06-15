# LightAI Go — Model Runtime Local End-to-End Verification

## Prerequisites

- NVIDIA GPU with driver (verified: RTX 5090 Laptop, driver 610.47)
- Docker daemon with `nvidia-container-toolkit` installed
- `ghcr.io/ggml-org/llama.cpp:server-cuda13` Docker image available
- GGUF model file (verified: `/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf`)
- Port 8002 available on host

## Quick Start

```bash
# From project root:
scripts/e2e-model-runtime-local.sh
```

Or run manually:

```bash
scripts/e2e-model-runtime-local.sh 8002 /home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf
```

## Environment Checks

```bash
# GPU
nvidia-smi --query-gpu=name,memory.total --format=csv

# Docker GPU runtime
docker run --rm --gpus all nvidia/cuda:13.1.1-base-ubuntu24.04 nvidia-smi

# Docker socket
ls -la /var/run/docker.sock

# Model file
ls -la ~/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf

# Port availability
ss -tln | grep 8002
```

## Manual Verification Steps

### 1. Start Server

```bash
cat > /tmp/e2e-config.yaml << 'EOF'
host: 127.0.0.1
port: 18080
db_path: /tmp/lightai-e2e.db
log_level: info
agent_token: lightai-agent-token-change-me
node_offline_threshold: 300s
EOF

# Terminal 1: Server
/tmp/lightai-server --config /tmp/e2e-config.yaml

# Reset password (separate terminal or before start)
/tmp/lightai-server --config /tmp/e2e-config.yaml --reset-admin-password test1234
```

### 2. Start Agent

```bash
cat > /tmp/agent-e2e.yaml << 'EOF'
server_url: http://127.0.0.1:18080
agent_id: agent-e2e
agent_token: lightai-agent-token-change-me
advertised_address: 127.0.0.1
primary_ip: 127.0.0.1
identity_dir: /tmp/lightai-runtime
gpu:
  profile: production
  collector_mode: auto
metrics:
  enabled: false
heartbeat:
  interval: 2s
collectors:
  system:
    enabled: false
  report_interval: 10s
logging:
  level: info
  stdout: true
  file_enabled: false
EOF

# Terminal 2: Agent (foreground to see logs)
/tmp/lightai-agent --config /tmp/agent-e2e.yaml
```

### 3. Login

```bash
API="http://127.0.0.1:18080/api/v1"
curl -c /tmp/cookies.txt -X POST "$API/auth/login" \
  -H "Content-Type: application/json" -H "Origin: http://127.0.0.1:18080" \
  -d '{"username":"admin","password":"test1234"}'
```

### 4. Create Objects

```bash
# ModelArtifact
curl -b /tmp/cookies.txt -X POST "$API/model-artifacts" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"name":"Qwen3.5-9B-Q4_K_M","path":"/home/kzeng/models/Qwen3.5-9B-Q4/Qwen3.5-9B-Q4_K_M.gguf","format":"gguf","task_type":"chat","architecture":"qwen","size_label":"9B","quantization":"int4"}'

# RuntimeEnvironment (bridge network, NOT host)
curl -b /tmp/cookies.txt -X POST "$API/runtime-environments" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"name":"llama-cpp-cuda13","runtime_type":"docker","backend_type":"llama_cpp","vendor":"nvidia","default_port":8000,"docker":{"image":"ghcr.io/ggml-org/llama.cpp:server-cuda13","ipc_mode":{"enabled":true,"value":"host"},"shm_size":{"enabled":true,"value":"8gb"}}}'

# RunTemplate (use ${MODEL_PATH} which resolves to CONTAINER path)
curl -b /tmp/cookies.txt -X POST "$API/run-templates" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" \
  -d '{"name":"llama-cpp-server","runtime_type":"docker","vendor":"nvidia","backend_type":"llama_cpp","required_variables":["MODEL_PATH","CONTAINER_PORT"],"args_template":["-m","${MODEL_PATH}","--host","0.0.0.0","--port","${CONTAINER_PORT}"],"volume_mappings":{"enabled":true,"value":[{"host_path":"/home/kzeng/models","container_path":"/models","readonly":true}]}}'
```

### 5. Create Deployment + Start

```bash
# Get node and GPU IDs
NODE_ID=$(curl -s -b /tmp/cookies.txt "$API/nodes" | jq -r '.[] | select(.status=="online") | .id')
GPU_ID=$(curl -s -b /tmp/cookies.txt "$API/gpus" | jq -r '.[] | select(.health=="healthy") | .id')

# Create deployment
DID=$(curl -s -b /tmp/cookies.txt -X POST "$API/model-deployments" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" \
  -d "{\"name\":\"e2e-test\",\"model_artifact_id\":\"$AID\",\"runtime_environment_id\":\"$RID\",\"run_template_id\":\"$TID\",\"node_id\":\"$NODE_ID\",\"gpu_ids\":[\"$GPU_ID\"],\"host_port\":8002}" | jq -r '.id')

# Dry-run
curl -s -b /tmp/cookies.txt -X POST "$API/model-deployments/$DID/dry-run" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" -d '{}'

# Start
curl -s -b /tmp/cookies.txt -X POST "$API/model-deployments/$DID/start" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" -d '{}'
```

### 6. Verify

```bash
# Container
docker ps --format '{{.ID}} {{.Image}} {{.Names}} {{.Status}} {{.Ports}}' | grep lightai

# Model API
curl http://127.0.0.1:8002/v1/models

# Instance status
curl -s -b /tmp/cookies.txt "$API/model-instances"

# Logs
curl -s -b /tmp/cookies.txt "$API/model-instances/$IID/logs"

# Stop
curl -s -b /tmp/cookies.txt -X POST "$API/model-deployments/$DID/stop" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -H "Origin: http://127.0.0.1:18080" -d '{}'
```

### 7. Cleanup

```bash
pkill -f lightai-server
pkill -f lightai-agent
rm -f /tmp/lightai-e2e.db /tmp/lightai-e2e.db*
rm -f /tmp/lightai-runtime/agent-identity.json
```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid credentials` | Password changed or DB recreated | Run `--reset-admin-password test1234` |
| `heartbeat response too large` | Task payload exceeds 1MB limit | Reduce command preview size |
| `docker daemon unavailable` | Docker not running or socket permissions | `sudo usermod -aG docker $USER` |
| Port conflict | Port 8002 in use | Use different port |
| `lease conflict` / GPU already reserved | Previous test not cleaned up | Delete old DB, restart |
