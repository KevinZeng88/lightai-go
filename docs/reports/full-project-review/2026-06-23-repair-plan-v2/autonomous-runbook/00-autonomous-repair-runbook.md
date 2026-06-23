# Autonomous Repair Runbook

> Date: 2026-06-23
> Purpose: Self-contained execution guide for MiMo autonomous repair
> **User approves this runbook → MiMo executes all batches automatically**

---

## 1. Product Stage

LightAI Go is a **lightweight GPU/model management platform for internal network environments**. NOT a public cloud, NOT Kubernetes, NOT SaaS.

**Implications**:
- No over-engineering
- No vendor policy engine
- No privileged approval
- No mTLS / secret manager / full billing / full gateway
- No backward compatibility with old configs/schemas/fallbacks
- Fix real issues that affect current operation
- Current working flows must continue working

---

## 2. Execution Order

```
Batch 1A (Tenant Scope) ───┐
Batch 1B (AgentClient)  ───┼──→ Batch 1C (Agent Endpoint) ──→ Batch 2 (Docker Lifecycle)
                            │                                      │
                            │                                      ├──→ Batch 3 (I/O Safety)
                            │                                      │
                            │                                      ├──→ Batch 4 (RunPlan)
                            │                                      │
                            │                                      ├──→ Batch 6 (Web UI)
                            │                                      │
                            │                                      └──→ Batch 7 (Tests)
                            │
                            └──→ (1A/1B new packages can parallel)
```

**Batch 5**: Paused. Future constraint only. No standalone execution.

---

## 3. Per-Batch Execution Rules

For each batch:

1. **Read** the batch plan document
2. **Run** baseline commands (go build, go test, npm test)
3. **Create** new files per plan
4. **Modify** existing files per plan
5. **Run** unit tests after each commit
6. **Run** integration tests after all commits
7. **Run** golden path checks
8. **Generate** closeout document
9. **Commit** with descriptive message
10. **Verify** git status clean

---

## 4. Automatic Continue Rules

MiMo continues to next batch when:

- All tests pass
- Golden path checks pass
- No Stop Conditions triggered
- Closeout document generated
- Commits clean

MiMo does NOT ask for user confirmation between batches.

---

## 5. Commit Rules

- One commit per logical change (per batch plan commit boundary)
- Message format: `fix(scope): description` or `feat(scope): description`
- Never commit secrets, credentials, tokens
- Never use `git add -A` — stage specific files
- Verify `git status` before each commit

---

## 6. Stop Conditions

See `12-stop-conditions-and-recovery.md` for full list. Key stops:

1. Golden Path broken, auto-recovery failed twice
2. NBR-defined parameters blocked
3. Default tenant/admin flow blocked
4. Server cannot reach local agent
5. Unrelated working tree changes appear
6. Tests fail for unclear reason after two fix attempts
7. Environment dependency missing (Docker, GPU, etc.)
8. Planned fix contradicts current decisions

---

## 7. Batch Summary

| Batch | Plan Doc | Key Files | Commits | Tests |
|-------|----------|-----------|---------|-------|
| 1A | 03-batch-1a-tenant-scope-plan.md | authz/, 6 handler files | 3 | authz + tenant isolation |
| 1B | 04-batch-1b-agentclient-plan.md | agentclient/, 4 handler files | 3 | agentclient + proxy |
| 1C | 05-batch-1c-agent-endpoint-protection-plan.md | cmd/agent/main.go, collector | 2 | endpoint auth |
| 2 | 06-batch-2-docker-lifecycle-plan.md | docker_client.go, docker.go, main.go | 5 | lifecycle + race |
| 3 | 07-batch-3-io-audit-log-safety-plan.md | middleware, helpers, docker_real | 4 | body limit + redaction |
| 4 | 08-batch-4-runplan-runtime-catalog-plan.md | resolver.go, catalog YAML | 4 | resolver tests |
| 6 | 09-batch-6-web-i18n-permission-plan.md | router, pages, stores, locales | 4 | frontend tests |
| 7 | 10-batch-7-test-infrastructure-plan.md | test files, scripts | 3 | all tests |

**Total**: ~28 commits, ~14-19 days equivalent work

---

## 8. Closeout Requirements

Each batch generates: `docs/reports/full-project-review/2026-06-23-repair-plan-v2/autonomous-runbook/batch-{X}-closeout.md`

See `11-autonomous-closeout-template.md` for format.
