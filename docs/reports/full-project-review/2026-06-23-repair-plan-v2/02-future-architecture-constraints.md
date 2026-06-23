# Future Architecture Constraints — Repair Plan V2

> Date: 2026-06-23
> Purpose: Document architectural constraints that current repairs must not violate

---

## 1. Why This Document Exists

Current repair plan V1 focuses on fixing immediate bugs (SSRF, tenant isolation, container cleanup, race conditions). These fixes are necessary but may inadvertently create patterns that block future capabilities:

- Multi-server deployment
- Multi-replica model serving
- Automatic scheduling
- Unified model API / inference gateway
- API key management
- Usage metering and billing
- Multi-vendor GPU support (NVIDIA, MetaX, Huawei)

This document defines constraints that all repairs must satisfy, even if the features themselves are not implemented now.

---

## 2. Constraint Categories

### 2.1 Deployment Must Not Be Single-Instance

**Current assumption**: One deployment → one instance → one container → one node.

**Future reality**: One deployment → N replicas → N instances → N containers → potentially multiple nodes.

**Constraint**: All code that touches deployment, instance, container, or task must distinguish between:
- `DeploymentDesiredState` (what the user wants)
- `ModelInstanceActualState` (what exists on a node)
- `ContainerState` (Docker-level state on an agent)

**Current violation**: `deployment_lifecycle_handlers.go` waits synchronously for a single agent task result. This pattern cannot scale to multi-replica.

### 2.2 Instance Must Not Assume Single Container

**Current assumption**: One instance = one container.

**Future reality**: One instance might be:
- A single container (current)
- A pod of containers (sidecar pattern)
- A process managed by systemd (non-Docker runtime)

**Constraint**: `ModelInstance` must not embed Docker-specific fields. Docker details belong in `ContainerRuntime` or `AgentRunSpec`.

### 2.3 Gateway Must Not Be Tied to Single Endpoint

**Current assumption**: One deployment = one service endpoint = one node.

**Future reality**: One deployment → multiple instance endpoints → load balanced by gateway.

**Constraint**: Service endpoint discovery must be derived from instance state, not hardcoded to a single node/port.

### 2.4 API Must Not Be OpenAI-Only

**Current assumption**: Model serving follows OpenAI-compatible API (`/v1/models`, `/v1/chat/completions`).

**Future reality**: Multiple protocol profiles:
- OpenAI-compatible (first priority)
- Internal lightweight API
- Vendor-specific API (MetaX, Huawei)
- Batch inference API
- Embedding-only API
- Rerank API
- Multimodal API

**Constraint**: All API routing, usage extraction, and error mapping must be behind a `ProtocolAdapter` abstraction. OpenAI-compatible is a profile, not the only protocol.

### 2.5 Auth Must Not Be Session-Only

**Current assumption**: All actors are console users with session-based auth.

**Future reality**: Multiple actor types:
- Console user (session)
- API key (bearer token)
- Service account (bearer token)
- System controller (internal)
- Agent (agent token)

**Constraint**: All audit logs, usage events, and permission checks must record `actor_type` + `actor_id`, not just `user_id`.

### 2.6 Usage Must Not Be Token-Only

**Current assumption**: No usage metering exists.

**Future reality**: Multiple usage dimensions:
- Token count (prompt + completion)
- Request count
- Input/output bytes
- Image units
- Embedding vectors
- Rerank pairs
- Audio seconds
- GPU seconds
- Instance runtime hours
- Custom billing units

**Constraint**: Usage event schema must be generic. Token counting is one extractor, not the only metric.

### 2.7 Tenant Must Not Be Consumer-Only

**Current assumption**: Tenant is a resource consumer (uses nodes, deployments, APIs).

**Future reality**: Tenant can be:
- Resource owner (owns GPUs, models, runtimes)
- Resource consumer (uses APIs, deployments)
- Both (owns infrastructure and consumes services)

**Constraint**: Tenant scope checks must support ownership resolution, not just membership.

### 2.8 Private IPs Must Not Be Universally Blocked

**Current assumption**: SSRF protection might block all private IPs.

**Future reality**: Multi-server deployments typically run on private networks (10.x, 192.168.x). Agents and servers communicate over LAN.

**Constraint**: SSRF protection must be mode-aware:
- `dev/single-node`: Allow localhost + private
- `lan/private`: Allow configured private CIDRs
- `production/cloud`: Block metadata (169.254.169.254), link-local, unspecified; allow configured CIDRs

### 2.9 RunPlan Must Not Be Docker Args Only

**Current assumption**: RunPlan produces Docker CLI args.

**Future reality**: RunPlan must express:
- Resource requirements (GPU memory, CPU, disk)
- Runtime requirements (CUDA version, driver version)
- Security policy requirements (privileged, IPC, devices)
- API compatibility profile
- Replica count
- Placement policy

**Constraint**: RunPlan must have three layers:
1. `DeploymentRunPlan` — desired state, hardware-agnostic
2. `NodeRunPlan` — resolved to specific node, GPU, runtime
3. `AgentRunSpec` — agent-local execution details

### 2.10 Scheduler Must Not Be Implicit

**Current assumption**: No scheduler exists. User manually selects node and GPU.

**Future reality**: Automatic scheduling based on:
- Resource availability
- GPU memory/capability
- Runtime compatibility
- Agent security policy
- Tenant quota
- Existing GPU leases
- Model location proximity

**Constraint**: All resource decisions must go through a scheduler interface, even if the current implementation is "manual placement."

---

## 3. Specific "Must Not Write Down" Rules

These are design decisions that current repairs must NOT cement:

| Current Pattern | Why It Blocks Future | Constraint |
|----------------|---------------------|------------|
| `deployment.InstanceID` (single) | Multi-replica needs `InstanceSet` | Use `DeploymentID` → query instances |
| `HandleStopDeployment` waits for 1 task | Multi-replica needs async reconciliation | Use controller/reconciler pattern |
| `waitForAgentTaskResult` polling | Blocks server goroutine | Use async task + callback/webhook |
| `ResolvedRunPlan` has Docker args | Agent needs its own spec layer | Split into NodeRunPlan + AgentRunSpec |
| `agent_handlers.go` parses Docker state | Server shouldn't know Docker internals | Agent reports `InstanceActualState` |
| Frontend only checks `is_platform_admin` | No fine-grained RBAC | Wire `hasPermission()` checks |
| Audit log uses `user_id` only | API keys have no user | Use `actor_type` + `actor_id` |
| `redactDetailString` substring replace | Corrupts non-sensitive text | Parse JSON key-value pairs |
| E2E scripts require real GPU | No CI coverage | Add mock E2E framework |
| `DockerClient` interface missing `ContainerRemove` | Cleanup impossible | Add to interface |

---

## 4. Abstraction Readiness Checklist

Before finalizing any repair, verify:

- [ ] Does it assume one instance per deployment? If yes, redesign.
- [ ] Does it assume one container per instance? If yes, abstract.
- [ ] Does it assume session-only auth? If yes, add actor model.
- [ ] Does it assume OpenAI-only API? If yes, add protocol adapter.
- [ ] Does it assume single-node? If yes, make node-aware.
- [ ] Does it block all private IPs? If yes, add mode-aware policy.
- [ ] Does it hardcode Docker specifics in server? If yes, move to agent.
- [ ] Does it use `fmt.Sprintf` for JSON? If yes, use `json.Marshal`.
- [ ] Does it assume token-only usage? If yes, make generic.
- [ ] Does it assume tenant is consumer-only? If yes, add ownership.
