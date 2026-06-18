> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# Reverse Proxy and TLS Deployment Guide

## Summary

LightAI Go Server and Agent communicate over HTTP. For production deployments
exposed beyond localhost, use a reverse proxy with TLS termination.

## Recommended Architecture

```
Client Browser (HTTPS)
    │
    ▼
Reverse Proxy (TLS termination) ─ e.g. nginx, Caddy, HAProxy
    │
    ▼
LightAI Go Server (HTTP, localhost:18080)
    │
    ▼
Agent (HTTP heartbeat to Server)
```

## Quick Start with Caddy

```caddyfile
lightai.example.com {
    reverse_proxy 127.0.0.1:18080
}
```

## Quick Start with nginx

```nginx
server {
    listen 443 ssl;
    server_name lightai.example.com;

    ssl_certificate     /etc/ssl/certs/lightai.crt;
    ssl_certificate_key /etc/ssl/private/lightai.key;

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Server Configuration

Set the server to bind localhost only (default in release config):

```yaml
host: "127.0.0.1"
port: 18080
```

## Agent Configuration

Agent communicates with Server over the loopback or internal network.
The agent token must be a secure random value:

```bash
export LIGHTAI_AGENT_TOKEN=$(openssl rand -hex 32)
```

## Cookie Security

When deploying behind TLS:
- Set `secure: true` in the session cookie config.
- This ensures session cookies are only sent over HTTPS.

## Observability Endpoints

Prometheus (19090) and Grafana (13000) should NOT be exposed on public
interfaces. Bind them to localhost or place them behind the same reverse
proxy with authentication.

## Current Limitations

- Native TLS in the Go server is not yet implemented.
- Use a reverse proxy for TLS termination until native support is added.
- This is a documented known limitation per `docs/PHASE-STATUS.md`.
