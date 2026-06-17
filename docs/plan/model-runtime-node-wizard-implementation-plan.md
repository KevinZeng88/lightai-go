# Model Runtime Node Wizard Implementation Plan

**Phase:** 4 — Model Runtime Wizards
**Branch:** `phase-4-model-runtime-wizards`
**Date:** 2026-06-18

## Phase 1: Agent Capabilities (Backend)

### 1.1 Agent File Browser
- **Goal:** Agent exposes `GET /files` on its metrics port for controlled directory browsing
- **Files:** `cmd/agent/main.go`, `configs/agent.yaml`
- **Config:** Add `model_browser.allowed_roots`, `max_entries`, `max_scan_depth`
- **Security:** Path traversal prevention, root-only browsing
- **API:** `GET /files?root=X&path=Y&limit=200&cursor=Z`

### 1.2 Agent Model Scanner
- **Goal:** Agent scans model directories for metadata
- **Files:** `internal/agent/collector/` (new `model_scanner.go`)
- **Capability:** Read config.json, detect safetensors, GGUF detection, estimate parameters
- **API:** `POST /model-paths/scan` with `{root, relative_path}`

### 1.3 Agent Docker Image Enhancement
- **Goal:** Return full image metadata (id, digest, created, repo_tags)
- **Files:** `cmd/agent/main.go` (enhance existing handler)
- **API:** Enhance `GET /docker-images` with search, pagination, richer fields

## Phase 2: Server API (Backend)

### 2.1 ModelLocation Management
- **Goal:** PATCH, DELETE for ModelLocation
- **Files:** `internal/server/api/artifact_handlers.go`, `router.go`

### 2.2 NodeBackendRuntime Management
- **Goal:** PATCH, DELETE for NodeBackendRuntime
- **Files:** `internal/server/api/runtime_handlers.go`, `router.go`

### 2.3 BackendRuntime Clone
- **Goal:** `POST /api/v1/backend-runtimes/{id}/clone`
- **Files:** `internal/server/api/runtime_handlers.go`, `router.go`

### 2.4 Agent Proxy Endpoints
- **Goal:** Server proxies file browsing and model scanning to agent
- **Files:** `internal/server/api/agent_handlers.go`, `router.go`

### 2.5 Standalone Preflight
- **Goal:** `POST /api/v1/deployments/preflight` computes candidate nodes
- **Files:** `internal/server/api/deployment_lifecycle_handlers.go`, `router.go`

## Phase 3: Web Wizards (Frontend)

### 3.1 File Browser Component
### 3.2 Docker Image Browser Component
### 3.3 Model Creation Wizard
### 3.4 Runtime Creation Wizard
### 3.5 Instance Start Wizard
### 3.6 i18n Keys
