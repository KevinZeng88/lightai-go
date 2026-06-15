# LightAI Go — Getting Started (Production)

This guide helps you deploy LightAI Go for the first time.

## Prerequisites

- Linux (x86_64) with glibc >= 2.28
- Docker (for model runtime)
- NVIDIA GPU + drivers (for GPU management)
- At least 256MB RAM for Server, 128MB for Agent

## Quick Install

```bash
# 1. Extract the release tarball
tar xzf lightai-go-0.1.14-linux-amd64.tar.gz -C /opt/
cd /opt/lightai-go-0.1.14-linux-amd64

# 2. Install systemd units (recommended for production)
sudo cp deploy/systemd/lightai-server.service /etc/systemd/system/
sudo cp deploy/systemd/lightai-agent.service /etc/systemd/system/
sudo systemctl daemon-reload

# 3. Start Server
sudo systemctl start lightai-server

# 4. Check Server health
curl http://localhost:18080/healthz

# 5. Start Agent (on each GPU node)
sudo systemctl start lightai-agent

# 6. Verify agent registration
curl http://localhost:18080/api/v1/nodes
```

## Access

- **Web Console**: http://localhost:18080
- **Default admin**: `admin` / password from `runtime/initial-credentials.txt`
- **Grafana**: http://localhost:13000
- **Prometheus**: http://localhost:19090

## Configuration

Edit `configs/server.yaml` and `configs/agent.yaml` before starting.

Key settings:
- `agent.server_url`: Server address (change from `127.0.0.1` for multi-node)
- `agent.token`: Shared secret for agent authentication
- `server.listen`: Server bind address

## Multi-Node

1. Start Server on one machine
2. Edit `configs/agent.yaml` on each GPU node:
   - Set `server_url` to the Server's IP
3. Start Agent on each GPU node

## Model Runtime

1. Open Web Console → Model Artifacts → Create (add model path)
2. Open Runtime Environments → Create (Docker config)
3. Open Run Templates → Create (command template)
4. Open Model Deployments → Create → Start

## Troubleshooting

- Server logs: `logs/lightai-server.log`
- Agent logs: `logs/lightai-agent.log`
- Credentials: `runtime/initial-credentials.txt`
- Reset password: `bash scripts/reset-password.sh`

See `docs/ops/model-runtime-troubleshooting.md` for model runtime issues.
