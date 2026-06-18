> Status: REFERENCE
> Last reviewed: 2026-06-18
> Scope: Reference document; not the current entrypoint
> Read order: See `docs/CURRENT.md`

# LightAI Go Full Project Review Plan

> Date: 2026-06-16
> Reviewer: Claude
> Scope: Architecture, code quality, tests, docs, security, observability, Web UI, GPU management, model runtime
> Constraint: Review only; no code changes; no commits

## Modules to Inspect

1. **Backend Server** (`cmd/server`, `internal/server`)
   - Router, middleware, RBAC, handlers, DB migrations, models, session/CSRF, rate limiter
   - Model artifact, runtime env, deployment, instance, lease, sweep, task, audit
   - Agent registration, heartbeat, resource reporting, node transfer

2. **Agent / GPU Collectors** (`cmd/agent`, `internal/agent`)
   - Collector interface, registry, external command, NVIDIA native, probe
   - Metrics snapshot, Prometheus exposure, registration/heartbeat
   - Docker runtime driver, task execution
   - GPU discovery scripts (NVIDIA, MetaX, AMD)

3. **Web / UI** (`web/src`)
   - Router, pages (all 19), API client, stores, composables, components, i18n
   - Tenant switching, auth flow, model serving pages, observability pages

4. **Common** (`internal/common`): config, errors, log, types, version

5. **Packaging / Deployment** (`scripts/`, `deploy/`, `configs/`)
   - Release scripts, patch system, install/start/stop, observability scripts
   - Prometheus/Grafana configs, dashboards, alert rules
   - systemd units

6. **Documentation** (`docs/`): design, plan, ops, review, README, release notes

7. **Tests**: unit, integration, E2E, shell, web

## GPUStack Comparison Areas

1. Architecture layers (server/worker, scheduler, detectors, controllers)
2. GPU device discovery and vendor abstraction
3. GPU/host monitoring and metrics exposure
4. Model deployment and instance lifecycle
5. Web UI information architecture and UX patterns
6. Code organization and patterns
7. Deployment and upgrade experience

## Read-Only Commands to Execute

```bash
go test ./...
go vet ./...
go build ./cmd/server && go build ./cmd/agent
bash -n scripts/*.sh
cd web && node tests/i18nKeys.test.mjs
cd web && node tests/apiClientPaths.test.mjs
cd web && node tests/formatters.test.mjs
cd web && npx vitest run
git diff --check
grep -rn "TODO\|FIXME" --include='*.go' internal/ cmd/
grep -rn "TODO\|FIXME" --include='*.vue' --include='*.ts' web/src/
find . -type f -name '*.go' ! -path './.cache/*' ! -path './web/*' | xargs wc -l | tail -1
```

## Commands NOT to Execute (Default Skip)

- `scripts/e2e-model-runtime-local.sh` (requires Docker + GPU + model files)
- `scripts/package-release-docker.sh --no-bump` (time-consuming build)
- `docker ...` (no Docker context available or needed)
- `nvidia-smi`, `mx-smi` (no GPU hardware in dev)
- Any command requiring GPU hardware or long-running service

## Key Questions to Answer

1. Is LightAI Go in a healthy state for its current phase?
2. Is the architecture suitable for "small/medium customers, handful of GPU servers"?
3. What does LightAI Go do well vs GPUStack? Where is it over/under-engineered?
4. What are the correctness, security, and reliability risks?
5. What are the UX, docs, deployment, and monitoring gaps?
6. What should Phase 2G/2H/3 do, and what should be deferred?
7. Are there "doc says done, code not done" or "code done, doc not updated" issues?
8. What foundation work is needed before model gateway/API key/billing?

## Final Report Path

```
docs/review/claude-full-project-review-20260616.md
```
