# Core Abstractions V2 — Repair Plan V2

> Date: 2026-06-23
> Purpose: Redefine key abstractions for current repairs and future extensibility

---

## 1. Tenant Authorizer / Resource Ownership Resolver

### Problem

Current tenant checks are scattered across handlers. Some check tenant, some don't. No unified pattern for:
- Single resource ownership
- Parent-child resource consistency
- Platform admin bypass
- API key / service account actors

### Design

```go
// internal/server/authz/authorizer.go

type ResourceType string

const (
    ResourceNode            ResourceType = "node"
    ResourceGPU             ResourceType = "gpu"
    ResourceBackend         ResourceType = "backend"
    ResourceBackendVersion  ResourceType = "backend_version"
    ResourceBackendRuntime  ResourceType = "backend_runtime"
    ResourceNodeBackendRuntime ResourceType = "node_backend_runtime"
    ResourceArtifact        ResourceType = "artifact"
    ResourceModelRoot       ResourceType = "model_root"
    ResourceModelLocation   ResourceType = "model_location"
    ResourceDeployment      ResourceType = "deployment"
    ResourceInstance        ResourceType = "instance"
    ResourceRunPlan         ResourceType = "run_plan"
    ResourceGPULease        ResourceType = "gpu_lease"
    ResourceAPIKey          ResourceType = "api_key"
    ResourceUsageEvent      ResourceType = "usage_event"
)

type ActorType string

const (
    ActorSessionUser   ActorType = "session_user"
    ActorAPIKey        ActorType = "api_key"
    ActorServiceAccount ActorType = "service_account"
    ActorSystem        ActorType = "system"
    ActorAgent         ActorType = "agent"
)

type Actor struct {
    Type     ActorType
    ID       string // user_id, api_key_id, service_account_id
    TenantID string
    IsAdmin  bool
}

type Authorizer interface {
    // CheckTenantOwnership verifies the resource belongs to the actor's tenant.
    // Returns nil if authorized, error if not.
    CheckTenantOwnership(ctx context.Context, actor Actor, resourceType ResourceType, resourceID string) error

    // CheckParentConsistency verifies parent-child tenant consistency.
    // e.g., node belongs to same tenant as deployment
    CheckParentConsistency(ctx context.Context, actor Actor, parentType ResourceType, parentID string, childType ResourceType, childID string) error

    // RequireAction verifies the actor has permission for the action on the resource.
    RequireAction(ctx context.Context, actor Actor, action string, resourceType ResourceType, resourceID string) error
}
```

### Usage Patterns

**Handler-level (explicit)**:
```go
func HandlePatchNodeBackendRuntime(w http.ResponseWriter, r *http.Request) {
    actor := auth.ActorFromContext(r.Context())
    nbrID := r.PathValue("id")

    // Check NBR belongs to actor's tenant
    if err := authz.CheckTenantOwnership(r.Context(), actor, authz.ResourceNodeBackendRuntime, nbrID); err != nil {
        http.Error(w, "not found", http.StatusNotFound) // 404, not 403
        return
    }

    // ... proceed ...
}
```

**Middleware-level (wrapping)**:
```go
router.HandleFunc("PATCH /api/node-backend-runtimes/{id}",
    authz.RequireTenantOwnership(authz.ResourceNodeBackendRuntime, "id")(handlePatchNBR))
```

### Key Decisions

1. **404 vs 403**: Cross-tenant access returns 404 (not 403) to avoid information leakage about resource existence.

2. **Admin bypass**: `Actor.IsAdmin` skips tenant check but not action permission check.

3. **Actor model**: All audit/usage logs record `Actor.Type` + `Actor.ID`, not just `user_id`.

4. **Parent-child consistency**: When creating a deployment on a node, verify both belong to the same tenant.

---

## 2. Agent Endpoint Registry + Address Policy + AgentClient

### Problem

Current server→agent calls use bare `http.Get()` with no SSRF protection, no timeout, inconsistent URL encoding.

### Design

```go
// internal/server/agentclient/client.go

type AddressPolicy struct {
    Mode        AddressMode // Dev, LAN, Production
    AllowedCIDRs []*net.IPNet
    DeniedCIDRs  []*net.IPNet // always denied (metadata, link-local, unspecified, multicast)
    AllowHostname bool
    RequireAgentAuth bool
}

type AddressMode int

const (
    AddressModeDev       AddressMode = iota // localhost + private allowed
    AddressModeLAN                          // configured private CIDRs allowed
    AddressModeProduction                   // only registered agent IPs allowed
)

type AgentClient struct {
    httpClient    *http.Client
    addressPolicy AddressPolicy
    agentRegistry AgentEndpointRegistry
    agentToken    string
}

type AgentEndpointRegistry interface {
    // GetEndpoint returns the validated address for an agent.
    // Returns error if agent not registered or address not approved.
    GetEndpoint(ctx context.Context, agentID string) (string, error)

    // ValidateAddress checks if an address is allowed by the policy.
    ValidateAddress(addr string) error
}
```

### Address Validation Rules

| Address Type | Dev | LAN | Production |
|-------------|-----|-----|------------|
| 127.0.0.1 / ::1 | ALLOW | ALLOW | ALLOW (local agent) |
| 10.0.0.0/8 | ALLOW | CONFIGURABLE | DENY (unless registered) |
| 172.16.0.0/12 | ALLOW | CONFIGURABLE | DENY (unless registered) |
| 192.168.0.0/16 | ALLOW | CONFIGURABLE | DENY (unless registered) |
| 169.254.0.0/16 | DENY | DENY | DENY |
| 0.0.0.0 / :: | DENY | DENY | DENY |
| 224.0.0.0/4 (multicast) | DENY | DENY | DENY |
| Hostname | ALLOW | ALLOW | CONFIGURABLE |

### Key Decisions

1. **Private IPs not universally blocked**: Multi-server deployments run on LAN. SSRF protection is mode-aware, not blanket-deny.

2. **Registered agent IPs always allowed**: If an agent is registered and its address is approved by admin, it's allowed regardless of CIDR.

3. **Hostname support**: Production mode can optionally allow hostnames with DNS rebinding protection (resolve once, cache, verify IP matches).

4. **Agent auth token**: All agent requests include `Authorization: Bearer <agent_token>`.

5. **Response body limit**: AgentClient limits response body to 100MB by default.

6. **Timeout**: Default 30s, configurable per endpoint type.

---

## 3. Agent Endpoint Protection / NBR Execution Boundary (Converged)

### Problem

Agent management endpoints (file browse, model scan, Docker inspect) are exposed without authentication. NBR-defined Docker parameters need to flow through without agent-side blocking.

### Design Decision

**NBR is the source of truth for runtime parameters.** Agent executes what NBR specifies. No vendor-specific policy engine. No privileged manual approval.

**Agent endpoint protection** is the real fix: require auth for management endpoints.

### Agent Endpoint Auth

```go
// cmd/agent/main.go — auth middleware
func agentAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if !strings.HasPrefix(token, "Bearer ") || strings.TrimPrefix(token, "Bearer ") != agentToken {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}
```

| Endpoint | Auth |
|----------|------|
| `/healthz` | None (load balancer) |
| `/metrics` | None (Prometheus) |
| `/docker-images` | Required |
| `/docker-image-inspect` | Required |
| `/files` | Required |
| `/model-paths/scan` | Required |

### NBR Execution Boundary

NBR-defined parameters flow through to RunPlan and AgentRunSpec without agent-side blocking:

- `/dev/mxcd` (MetaX) → flows through
- `/dev/dri` → flows through
- `--privileged` → flows through (admin configured in NBR)
- `--ipc=host` → flows through
- `--security-opt` → flows through
- `--group-add` → flows through

**Audit logging**: High-risk parameters are logged in audit detail when deployment starts. This provides traceability without blocking.

**Responsibility**: Admin who configures NBR is responsible for parameter safety. Platform ensures only authorized users can modify NBR.

### Future Enhancement (Not This Cycle)

- Agent security policy as configurable allowlist (not vendor-specific engine)
- Scheduler preflight check against agent capabilities
- Operator approval workflow for high-risk capabilities

---

## 4. Docker Lifecycle / Instance Controller

### Problem

No lifecycle state machine. `Stop()` doesn't remove containers. Failed starts leave orphans.

### State Machine

```
                    ┌──────────────────────────────────────────┐
                    │           Deployment Desired State        │
                    │  desired_replicas, placement, resources   │
                    └──────────────┬───────────────────────────┘
                                   │
                    ┌──────────────▼───────────────────────────┐
                    │         Instance Controller               │
                    │  reconcile(desired, actual) → actions     │
                    └──────────────┬───────────────────────────┘
                                   │
                    ┌──────────────▼───────────────────────────┐
                    │         ModelInstance Actual State         │
                    │                                           │
                    │  pending → starting → running → stopping  │
                    │     │        │          │          │       │
                    │     │        ▼          │          ▼       │
                    │     │    failed         │       stopped    │
                    │     │        │          │          │       │
                    │     ▼        ▼          ▼          ▼       │
                    │  ┌─────────────────────────────────────┐  │
                    │  │           Container State            │  │
                    │  │  (agent-local, not in server DB)     │  │
                    │  └─────────────────────────────────────┘  │
                    └───────────────────────────────────────────┘
```

### Cleanup Semantics

| Scenario | Container Action | Evidence Retention |
|----------|-----------------|-------------------|
| Start fails (create succeeds, start fails) | Remove immediately | Logs captured before remove |
| Start fails (health check fails) | Remove immediately | Logs captured before remove |
| Stop requested | Stop → capture logs → Remove | Logs sent to server |
| Restart policy conflict | Remove after stop | Logs sent to server |
| Agent crash | Reconciliation detects orphan → remove | Logs may be lost |
| Server crash | Agent reconciliation cleans up | Logs in agent local storage |

### API Sync/Async Boundary

**Current (broken)**: `HandleStopDeployment` waits synchronously for agent task result.

**Correct design**:
1. Server sets `Deployment.DesiredState = "stopped"`
2. Instance controller detects mismatch with actual state
3. Controller dispatches stop task to agent
4. Agent executes stop, reports result via heartbeat
5. Controller updates instance state
6. API returns immediately with 202 Accepted

---

## 5. RunPlan / NodeRunPlan / AgentRunSpec

### Three-Layer Design

```
DeploymentRunPlan (server, hardware-agnostic)
    │
    │  Scheduler resolves to specific node/GPU
    ▼
NodeRunPlan (server, node-specific)
    │
    │  Agent resolves to local execution details
    ▼
AgentRunSpec (agent, execution-ready)
```

### DeploymentRunPlan

```go
type DeploymentRunPlan struct {
    // What to run
    ArtifactPath      string
    BackendName       string
    BackendVersion    string
    RuntimeRequirements RuntimeRequirements

    // How to run
    ParameterProfile  map[string]interface{}
    EnvOverrides      map[string]string
    ArgsOverride      []string

    // Where to run
    ReplicaCount      int
    PlacementPolicy   PlacementPolicy
    ResourceRequest   ResourceRequest

    // How to expose
    ServiceConfig     ServiceConfig
    APICompatibility  APICompatibilityProfile
}
```

### NodeRunPlan

```go
type NodeRunPlan struct {
    // Resolved node/GPU
    NodeID            string
    AssignedGPUs      []string
    ModelLocationPath string
    NodeBackendRuntimeID string

    // Resolved config
    ResolvedImage     string
    ResolvedArgs      []string
    ResolvedEnv       map[string]string
    ResolvedMounts    []Mount
    ResolvedDevices   []DeviceMapping
    ResolvedPorts     PortConfig

    // Policy check result (future — not implemented this cycle)
    PreflightOK       bool
    PreflightWarnings []string
}
```

### AgentRunSpec

```go
type AgentRunSpec struct {
    // Agent-local execution details only
    ContainerName  string
    Image          string
    Entrypoint     []string
    Command        []string
    Env            []string
    Binds          []string
    Devices        []DeviceMapping
    PortBindings   map[string][]PortBinding
    Privileged     bool
    IPCMode        string
    UTSMode        string
    NetworkMode    string
    SecurityOpt    []string
    GroupAdd       []string
    RestartPolicy  string
    HealthCheck    *HealthCheckConfig
}
```

---

## 6. RuntimeRequirements / BackendCapabilityProfile

### RuntimeRequirements (what the model needs)

```go
type RuntimeRequirements struct {
    // GPU requirements
    MinGPUMemory    int64    // bytes
    GPUVendor       string   // "nvidia", "metax", "huawei", ""
    MinGPUs         int
    GPUFeatures     []string // ["bf16", "fp16", "tensorcore"]

    // Runtime requirements
    MinCUDAVersion  string
    MinDriverVersion string
    RequiredDevices []string // ["/dev/dri", "/dev/nvidia"]

    // Capability requirements
    SupportsChat      bool
    SupportsCompletion bool
    SupportsEmbedding  bool
    SupportsStreaming  bool
    SupportsToolCalling bool
}
```

### BackendCapabilityProfile (what the backend provides)

```go
type BackendCapabilityProfile struct {
    BackendName    string
    Version        string
    RuntimeID      string

    // GPU capabilities
    SupportedGPUVendors []string
    MinGPUMemory        int64
    RequiredDevices     []string

    // API capabilities
    SupportsChat        bool
    SupportsCompletion  bool
    SupportsEmbedding   bool
    SupportsStreaming   bool
    SupportsToolCalling bool

    // Process capabilities
    DefaultEntrypoint   string
    DefaultArgs         []string
    HealthCheckEndpoint string
}
```

### Preflight Check (Future — Not This Cycle)

```go
func PreflightCheck(reqs RuntimeRequirements, caps BackendCapabilityProfile, node NodeResources) PreflightResult {
    // Current cycle: GPU memory check, device availability check
    // Future cycle: may add agent security policy check
    return PreflightResult{OK: true, Warnings: []string{}}
}
```

---

## 7. Scheduler / Preflight Minimal Model

### SchedulerInput

```go
type SchedulerInput struct {
    Deployment      Deployment
    RunPlan         DeploymentRunPlan
    RuntimeReqs     RuntimeRequirements
    CapProfile      BackendCapabilityProfile
    APIProfile      APICompatibilityProfile
    TenantID        string
    TenantQuota     QuotaPolicy
    ExistingLeases  []GPULease
}
```

### CandidateNode

```go
type CandidateNode struct {
    NodeID          string
    TenantID        string
    AvailableGPUs   []GPUInfo
    ModelLocations  []ModelLocation
    NodeBackendRuntimes []NodeBackendRuntime
    Score           float64 // higher is better
}
```

### SchedulerDecision

```go
type SchedulerDecision struct {
    SelectedNode    *CandidateNode
    AssignedGPUs    []string
    SelectedRuntime *NodeBackendRuntime
    SelectedLocation *ModelLocation
    UnschedulableReason string // empty if schedulable
}
```

### Relationship to Existing Types

| New Abstraction | Maps To | Notes |
|----------------|---------|-------|
| `DeploymentRunPlan` | `ResolvedRunPlan` (current) | Split from single plan to deployment-level |
| `NodeRunPlan` | Part of `ResolvedRunPlan` | Node-specific resolution |
| `AgentRunSpec` | `AgentRunSpec` (current) | Already exists, just clarify boundary |
| `RuntimeRequirements` | `RuntimeRequirements` (current) | Already exists, needs extension |
| `BackendCapabilityProfile` | `capabilities_json` in YAML | Needs structured Go type |
| `APICompatibilityProfile` | New | Describes protocol support |
| `SchedulerInput` | New | Aggregates all scheduling inputs |
| `CandidateNode` | New | Node + available resources |
| `SchedulerDecision` | New | Output of scheduler |

---

## 8. Unified Model API / Inference Gateway — FUTURE CONSTRAINT ONLY

**Status**: Not implemented this cycle. Documented as future constraint.

### Future Constraints (Must Not Write Down)

- Current priority: OpenAI-compatible API entry
- Future: must not hardcode OpenAI-only
- Don't hardcode deployment name as API model name
- Don't hardcode actor as session user
- Don't hardcode usage as token-only

### Architecture Direction (Future)

```
Control Plane (Server) → Data Plane (Gateway) → Instance Endpoints
```

Gateway will:
1. Receive client request (e.g., `POST /v1/chat/completions`)
2. Resolve model name → deployment → running instances
3. Select instance (load balancing)
4. Proxy request to instance endpoint
5. Extract usage from response
6. Return response to client

### Protocol Adapter (Future)

OpenAI-compatible is first profile. Future profiles: internal, vendor-specific, batch, embedding-only, rerank, multimodal.

---

## 9. Usage / Audit / Billing Model — FUTURE CONSTRAINT ONLY

**Status**: Not implemented this cycle. Documented as future constraint.

### Future Constraints (Must Not Write Down)

- Usage event schema must be generic (not token-only)
- Actor model must support API key / service account (not session-user-only)
- Audit log must use structured JSON (not fmt.Sprintf)
- Prompt/completion content: default NOT stored

### Audit Log (Future)

Records management actions: login, create/deploy, start/stop, revoke key, change quota.

### Usage Event (Future)

Records API calls: request_id, actor, model, tokens, bytes, latency, status.

### Billing (Future)

Based on: token usage, request count, GPU hours, instance runtime, custom units.
