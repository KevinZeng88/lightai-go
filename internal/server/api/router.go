// Package api provides HTTP routing setup for the LightAI Server.
package api

import (
	"net/http"

	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
	srvmetrics "lightai-go/internal/server/metrics"
	"lightai-go/internal/server/rbac"
)

// RouterConfig holds all dependencies needed to set up routes.
type RouterConfig struct {
	DB              *db.DB
	AgentToken      string
	SessionStore    *auth.SessionStore
	SessionCfg      auth.SessionConfig
	AuthHandler     *auth.AuthHandler
	RBACHandler     *rbac.Handler
	AgentHandler    *AgentHandler
	ResourceHandler *ResourceHandler
	ModelHandler    *ModelHandler
	ServerMetrics   *srvmetrics.ServerMetrics
}

// SetupRoutes registers all API routes on the given mux.
func SetupRoutes(mux *http.ServeMux, cfg RouterConfig) {
	// Auth endpoints (no session required for login/CSRF).
	mux.HandleFunc("POST /api/v1/auth/login", cfg.AuthHandler.HandleLogin)
	mux.HandleFunc("GET /api/v1/auth/csrf-token", cfg.AuthHandler.HandleCSRFToken)

	// Session-protected auth routes.
	mux.Handle("POST /api/v1/auth/logout", sessionChain(cfg, cfg.AuthHandler.HandleLogout))
	mux.Handle("POST /api/v1/auth/change-password", sessionChain(cfg, cfg.AuthHandler.HandleChangePassword))
	mux.Handle("GET /api/v1/auth/me", sessionChain(cfg, cfg.AuthHandler.HandleMe))

	// Platform admin routes.
	mux.Handle("GET /api/v1/users", platformChain(cfg, cfg.RBACHandler.HandleListUsers))
	mux.Handle("POST /api/v1/users", platformChain(cfg, cfg.RBACHandler.HandleCreateUser))
	mux.Handle("GET /api/v1/users/{id}", platformChain(cfg, cfg.RBACHandler.HandleGetUser))
	mux.Handle("PUT /api/v1/users/{id}", platformChain(cfg, cfg.RBACHandler.HandleUpdateUser))
	mux.Handle("POST /api/v1/users/{id}/disable", platformChain(cfg, cfg.RBACHandler.HandleDisableUser))
	mux.Handle("POST /api/v1/users/{id}/reset-password", platformChain(cfg, cfg.RBACHandler.HandleResetPassword))

	mux.Handle("GET /api/v1/tenants", platformChain(cfg, cfg.RBACHandler.HandleListTenants))
	mux.Handle("POST /api/v1/tenants", platformChain(cfg, cfg.RBACHandler.HandleCreateTenant))
	mux.Handle("GET /api/v1/tenants/{id}", platformChain(cfg, cfg.RBACHandler.HandleGetTenant))
	mux.Handle("PUT /api/v1/tenants/{id}", platformChain(cfg, cfg.RBACHandler.HandleUpdateTenant))
	mux.Handle("POST /api/v1/tenants/{id}/disable", platformChain(cfg, cfg.RBACHandler.HandleDisableTenant))

	// Tenant admin routes.
	mux.Handle("GET /api/v1/tenant-memberships", tenantChain(cfg, cfg.RBACHandler.HandleListMemberships, "membership:read"))
	mux.Handle("POST /api/v1/tenant-memberships", tenantChain(cfg, cfg.RBACHandler.HandleCreateMembership, "membership:write"))
	mux.Handle("POST /api/v1/tenant-memberships/{id}/disable", tenantChain(cfg, cfg.RBACHandler.HandleDisableMembership, "membership:write"))
	mux.Handle("POST /api/v1/tenant-memberships/{id}/roles", tenantChain(cfg, cfg.RBACHandler.HandleAddMembershipRole, "membership:write"))
	mux.Handle("DELETE /api/v1/tenant-memberships/{id}/roles/{role_id}", tenantChain(cfg, cfg.RBACHandler.HandleRemoveMembershipRole, "membership:write"))

	mux.Handle("GET /api/v1/roles", tenantChain(cfg, cfg.RBACHandler.HandleListRoles, "role:read"))
	mux.Handle("POST /api/v1/roles", tenantChain(cfg, cfg.RBACHandler.HandleCreateRole, "role:write"))
	mux.Handle("DELETE /api/v1/roles/{id}", tenantChain(cfg, cfg.RBACHandler.HandleDeleteRole, "role:write"))
	mux.Handle("PUT /api/v1/roles/{id}/permissions", tenantChain(cfg, cfg.RBACHandler.HandleUpdateRolePermissions, "role:write"))

	mux.Handle("GET /api/v1/permissions", tenantChain(cfg, cfg.RBACHandler.HandleListPermissions, "role:read"))

	// Observability status (no auth — used by Web frontend to avoid CORS).
	mux.HandleFunc("GET /api/v1/observability/status", HandleObservabilityStatus)

	// Agent API routes (use agent token, not session).
	agentMW := auth.AgentAuthMiddleware(cfg.AgentToken)
	mux.Handle("POST /api/v1/agent/register", agentMW(http.HandlerFunc(cfg.AgentHandler.HandleRegister)))
	mux.Handle("POST /api/v1/agent/heartbeat", agentMW(http.HandlerFunc(cfg.AgentHandler.HandleHeartbeat)))
	mux.Handle("POST /api/v1/agent/resources/report", agentMW(http.HandlerFunc(cfg.ResourceHandler.HandleResourceReport)))
		th := NewTaskHandler(cfg.DB)
		mux.Handle("POST /api/v1/agent/tasks/{id}/result", agentMW(http.HandlerFunc(th.HandleTaskResult)))

	// Resource routes (node:read permission).
	resourceChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("node:read"),
	)
	mux.Handle("GET /api/v1/nodes", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleListNodes)))
	mux.Handle("GET /api/v1/nodes/{id}", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNode)))
	// PATCH /api/v1/nodes/{id}/tenant — platform admin only.
	mux.Handle("PATCH /api/v1/nodes/{id}/tenant", platformChain(cfg, cfg.AgentHandler.HandlePatchNodeTenant))
	// P1-004: Host system snapshot endpoint.
	mux.Handle("GET /api/v1/nodes/{id}/system", resourceChain(http.HandlerFunc(cfg.ResourceHandler.HandleGetNodeSystem)))

	// GPU routes (gpu:read permission).
	gpuChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("gpu:read"),
	)
	mux.Handle("GET /api/v1/gpus", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleListGPUs)))
	mux.Handle("GET /api/v1/gpus/{id}", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleGetGPU)))

	// Phase 1: Model runtime serving APIs.
	mh := cfg.ModelHandler

	// ModelArtifact CRUD (model:read / model:write).
	modelReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("model:read"),
	)
	modelWriteChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
		auth.RequirePermission("model:write"),
	)
	mux.Handle("GET /api/v1/model-artifacts", modelReadChain(http.HandlerFunc(mh.HandleListModelArtifacts)))
	mux.Handle("POST /api/v1/model-artifacts", modelWriteChain(http.HandlerFunc(mh.HandleCreateModelArtifact)))
	mux.Handle("GET /api/v1/model-artifacts/{id}", modelReadChain(http.HandlerFunc(mh.HandleGetModelArtifact)))
	mux.Handle("PATCH /api/v1/model-artifacts/{id}", modelWriteChain(http.HandlerFunc(mh.HandlePatchModelArtifact)))
	mux.Handle("DELETE /api/v1/model-artifacts/{id}", modelWriteChain(http.HandlerFunc(mh.HandleDeleteModelArtifact)))

	// RuntimeEnvironment CRUD (runtime:read / runtime:write).
	runtimeReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("runtime:read"),
	)
	runtimeWriteChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
		auth.RequirePermission("runtime:write"),
	)
	mux.Handle("GET /api/v1/runtime-environments", runtimeReadChain(http.HandlerFunc(mh.HandleListRuntimeEnvironments)))
	mux.Handle("POST /api/v1/runtime-environments", runtimeWriteChain(http.HandlerFunc(mh.HandleCreateRuntimeEnvironment)))
	mux.Handle("GET /api/v1/runtime-environments/{id}", runtimeReadChain(http.HandlerFunc(mh.HandleGetRuntimeEnvironment)))
	mux.Handle("PATCH /api/v1/runtime-environments/{id}", runtimeWriteChain(http.HandlerFunc(mh.HandlePatchRuntimeEnvironment)))
	mux.Handle("DELETE /api/v1/runtime-environments/{id}", runtimeWriteChain(http.HandlerFunc(mh.HandleDeleteRuntimeEnvironment)))

	// RunTemplate CRUD (runtime:read / runtime:write).
	mux.Handle("GET /api/v1/run-templates", runtimeReadChain(http.HandlerFunc(mh.HandleListRunTemplates)))
	mux.Handle("POST /api/v1/run-templates", runtimeWriteChain(http.HandlerFunc(mh.HandleCreateRunTemplate)))
	mux.Handle("GET /api/v1/run-templates/{id}", runtimeReadChain(http.HandlerFunc(mh.HandleGetRunTemplate)))
	mux.Handle("PATCH /api/v1/run-templates/{id}", runtimeWriteChain(http.HandlerFunc(mh.HandlePatchRunTemplate)))
	mux.Handle("DELETE /api/v1/run-templates/{id}", runtimeWriteChain(http.HandlerFunc(mh.HandleDeleteRunTemplate)))
	mux.Handle("POST /api/v1/run-templates/{id}/render-preview", runtimeReadChain(http.HandlerFunc(mh.HandleRenderPreview)))

	// ModelDeployment CRUD (deployment:read / deployment:write).
	deployReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("deployment:read"),
	)
	deployWriteChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
		auth.RequirePermission("deployment:write"),
	)
	mux.Handle("GET /api/v1/model-deployments", deployReadChain(http.HandlerFunc(mh.HandleListModelDeployments)))
	mux.Handle("POST /api/v1/model-deployments", deployWriteChain(http.HandlerFunc(mh.HandleCreateModelDeployment)))
	mux.Handle("GET /api/v1/model-deployments/{id}", deployReadChain(http.HandlerFunc(mh.HandleGetModelDeployment)))
	mux.Handle("PATCH /api/v1/model-deployments/{id}", deployWriteChain(http.HandlerFunc(mh.HandlePatchModelDeployment)))
	mux.Handle("DELETE /api/v1/model-deployments/{id}", deployWriteChain(http.HandlerFunc(mh.HandleDeleteModelDeployment)))
	mux.Handle("POST /api/v1/model-deployments/{id}/dry-run", deployWriteChain(http.HandlerFunc(mh.HandleDryRun)))
		mux.Handle("POST /api/v1/model-deployments/{id}/start", deployWriteChain(http.HandlerFunc(mh.HandleStartDeployment)))
		mux.Handle("POST /api/v1/model-deployments/{id}/stop", deployWriteChain(http.HandlerFunc(mh.HandleStopDeployment)))

	// ModelInstance read-only (instance:read).
	instanceReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("instance:read"),
	)
	mux.Handle("GET /api/v1/model-instances", instanceReadChain(http.HandlerFunc(mh.HandleListModelInstances)))
	mux.Handle("GET /api/v1/model-instances/{id}", instanceReadChain(http.HandlerFunc(mh.HandleGetModelInstance)))
		mux.Handle("GET /api/v1/model-instances/{id}/logs", instanceReadChain(http.HandlerFunc(mh.HandleGetInstanceLogs)))

	// GpuLease read-only (gpu:read).
	mux.Handle("GET /api/v1/gpu-leases", gpuChain(http.HandlerFunc(mh.HandleListGpuLeases)))
	mux.Handle("GET /api/v1/gpu-leases/{id}", gpuChain(http.HandlerFunc(mh.HandleGetGpuLease)))

		// Audit logs (platform_admin or audit:read).
		auditChain := chain(
			auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
			auth.RequirePermission("audit:read"),
		)
		ah := NewAuditHandler(cfg.DB)
		mux.Handle("GET /api/v1/audit-logs", auditChain(http.HandlerFunc(ah.HandleListAuditLogs)))
}

func sessionChain(cfg RouterConfig, h http.HandlerFunc) http.Handler {
	return chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
	)(h)
}

func platformChain(cfg RouterConfig, h http.HandlerFunc) http.Handler {
	sessionMW := auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg)
	csrfMW := auth.CSRFMiddleware(cfg.SessionCfg)
	pwdMW := auth.RequirePasswordNotExpired(cfg.DB)
	return sessionMW(csrfMW(pwdMW(auth.RequirePlatformAdmin(http.HandlerFunc(h)))))
}

func tenantChain(cfg RouterConfig, h http.HandlerFunc, permission string) http.Handler {
	return chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
		auth.RequirePasswordNotExpired(cfg.DB),
		auth.RequirePermission(permission),
	)(h)
}

func chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

func handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}
