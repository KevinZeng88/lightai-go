# Reliability & Observability Status

## GPU Lease Concurrency
- GPU leases created in transaction with instance creation
- Released on deployment stop/delete
- Statuses: reserved → active → released
- Concurrent start handled by lease availability check

## Deployment Lifecycle
- Stop: idempotent (checks current state before stopping)
- Delete: releases GPU leases, stops instances, cleans up tasks
- Start: requires NBR ready + preflight pass

## Observability
- Server-managed Prometheus/Grafana: external binaries, not Go-supervised
- Status: documented limitation, not auto-recovery
- Logs endpoint: GET /api/v1/node-run-plans/{id}/logs with tail/bytes support
- Key operation_id tracked in deployment start/stop/dry-run lifecycle

## R-013 Status
CLOSED — observability implementation is documented and matches actual capabilities.
