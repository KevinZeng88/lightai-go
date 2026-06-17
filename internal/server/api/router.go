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
	ServerMetrics   *srvmetrics.ServerMetrics
}

// SetupRoutes registers all API routes on the given mux.
func SetupRoutes(mux *http.ServeMux, cfg RouterConfig) {
	// Wire login metrics to AuthHandler (C3 fix: login counter was TODO).
	if cfg.AuthHandler != nil && cfg.ServerMetrics != nil {
		cfg.AuthHandler.Metrics = cfg.ServerMetrics
	}

	// Auth endpoints (no session required for login/CSRF).
	mux.HandleFunc("POST /api/v1/auth/login", cfg.AuthHandler.HandleLogin)
	mux.HandleFunc("GET /api/v1/auth/csrf-token", cfg.AuthHandler.HandleCSRFToken)

	// Session-protected auth routes.
	mux.Handle("POST /api/v1/auth/logout", sessionChain(cfg, cfg.AuthHandler.HandleLogout))
	mux.Handle("POST /api/v1/auth/change-password", sessionChain(cfg, cfg.AuthHandler.HandleChangePassword))
	mux.Handle("GET /api/v1/auth/me", sessionChain(cfg, cfg.AuthHandler.HandleMe))
	mux.Handle("POST /api/v1/session/switch-tenant", sessionChain(cfg, cfg.AuthHandler.HandleSwitchTenant))

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

	// Resource routes (node:read permission).
	resourceChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("node:read"),
	)
	mux.Handle("GET /api/v1/nodes", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleListNodes)))
	mux.Handle("GET /api/v1/nodes/{id}", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNode)))
	// PATCH /api/v1/nodes/{id}/tenant — platform admin only.
	mux.Handle("PATCH /api/v1/nodes/{id}/tenant", platformChain(cfg, cfg.AgentHandler.HandlePatchNodeTenant))
	mux.Handle("GET /api/v1/nodes/{id}/system", resourceChain(http.HandlerFunc(cfg.ResourceHandler.HandleGetNodeSystem)))
	mux.Handle("GET /api/v1/nodes/{id}/docker-images", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNodeDockerImages)))

	// GPU routes (gpu:read permission).
	gpuChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("gpu:read"),
	)
	mux.Handle("GET /api/v1/gpus", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleListGPUs)))
	mux.Handle("GET /api/v1/gpus/{id}", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleGetGPU)))

	// Phase 4: Model runtime serving APIs.
	// Backend / BackendVersion (read-only).
	backendReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("backend:read"),
	)
	mux.Handle("GET /api/v1/inference-backends", backendReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListBackends)))
	mux.Handle("GET /api/v1/inference-backends/{id}", backendReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetBackend)))
	mux.Handle("GET /api/v1/inference-backends/{id}/versions", backendReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListBackendVersions)))

	// BackendRuntimeTemplate (read-only from config files).
	mux.Handle("GET /api/v1/backend-runtime-templates", backendReadChain(http.HandlerFunc(HandleListRuntimeTemplates)))
	mux.Handle("GET /api/v1/backend-runtime-templates/{name}", backendReadChain(http.HandlerFunc(HandleGetRuntimeTemplate)))

	// BackendRuntime CRUD (backend_runtime:read / backend_runtime:write).
	brReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("backend_runtime:read"),
	)
	brWriteChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
		auth.RequirePermission("backend_runtime:write"),
	)
	mux.Handle("GET /api/v1/backend-runtimes", brReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListBackendRuntimes)))
	mux.Handle("POST /api/v1/backend-runtimes/from-template", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateBackendRuntimeFromTemplate)))
	mux.Handle("GET /api/v1/backend-runtimes/{id}", brReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetBackendRuntime)))
	mux.Handle("PATCH /api/v1/backend-runtimes/{id}", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchBackendRuntime)))
	mux.Handle("DELETE /api/v1/backend-runtimes/{id}", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteBackendRuntime)))

	// ModelArtifact CRUD.
	maReadChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.RequirePermission("model_artifact:read"))
	maWriteChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_artifact:write"))
	mux.Handle("GET /api/v1/model-artifacts", maReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListArtifacts)))
	mux.Handle("POST /api/v1/model-artifacts", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateArtifact)))
	mux.Handle("GET /api/v1/model-artifacts/{id}", maReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetArtifact)))
	mux.Handle("PATCH /api/v1/model-artifacts/{id}", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchArtifact)))
	mux.Handle("DELETE /api/v1/model-artifacts/{id}", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteArtifact)))

	// ModelDeployment CRUD + lifecycle.
	mdReadChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.RequirePermission("model_deployment:read"))
	mdWriteChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_deployment:write"))
	mdStartChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_deployment:start"))
	mdStopChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_deployment:stop"))
	mux.Handle("GET /api/v1/model-deployments", mdReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListDeployments)))
	mux.Handle("POST /api/v1/model-deployments", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateDeployment)))
	mux.Handle("GET /api/v1/model-deployments/{id}", mdReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetDeployment)))
	mux.Handle("PATCH /api/v1/model-deployments/{id}", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchDeployment)))
	mux.Handle("DELETE /api/v1/model-deployments/{id}", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteDeployment)))
	mux.Handle("POST /api/v1/model-deployments/{id}/dry-run", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeploymentDryRun)))
	mux.Handle("POST /api/v1/model-deployments/{id}/start", mdStartChain(http.HandlerFunc(cfg.AgentHandler.HandleStartDeployment)))
	mux.Handle("POST /api/v1/model-deployments/{id}/stop", mdStopChain(http.HandlerFunc(cfg.AgentHandler.HandleStopDeployment)))

	// ModelInstance read.
	miReadChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.RequirePermission("model_instance:read"))
	mux.Handle("GET /api/v1/model-instances", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListInstances)))
	mux.Handle("GET /api/v1/model-instances/{id}", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetInstance)))

	// Agent task result.
	mux.Handle("POST /api/v1/agent/tasks/{id}/result", agentMW(http.HandlerFunc(cfg.AgentHandler.HandleTaskResult)))

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
