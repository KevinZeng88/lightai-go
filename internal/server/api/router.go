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
	nodeModelRootWriteChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
		auth.RequirePermission("node_model_root:write"),
	)
	nodeFileReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("node_file:read"),
	)
	mux.Handle("GET /api/v1/nodes", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleListNodes)))
	mux.Handle("GET /api/v1/nodes/{id}", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNode)))
	// PATCH /api/v1/nodes/{id}/tenant — platform admin only.
	mux.Handle("PATCH /api/v1/nodes/{id}/tenant", platformChain(cfg, cfg.AgentHandler.HandlePatchNodeTenant))
	mux.Handle("GET /api/v1/nodes/{id}/system", resourceChain(http.HandlerFunc(cfg.ResourceHandler.HandleGetNodeSystem)))
	mux.Handle("GET /api/v1/nodes/{id}/model-browser/roots", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleListNodeModelBrowserRoots)))
	mux.Handle("POST /api/v1/nodes/{id}/model-browser/roots", nodeModelRootWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleAddNodeModelBrowserRoot)))
	mux.Handle("DELETE /api/v1/nodes/{id}/model-browser/roots", nodeModelRootWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteNodeModelBrowserRoot)))
	mux.Handle("GET /api/v1/nodes/{id}/model-roots", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleListNodeModelRoots)))
	mux.Handle("POST /api/v1/nodes/{id}/model-roots", nodeModelRootWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleAddNodeModelRoot)))
	mux.Handle("PATCH /api/v1/nodes/{id}/model-roots/{root_id}", nodeModelRootWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchNodeModelRoot)))
	mux.Handle("DELETE /api/v1/nodes/{id}/model-roots/{root_id}", nodeModelRootWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteNodeModelRoot)))

	mux.Handle("GET /api/v1/nodes/{id}/docker-images", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNodeDockerImages)))
	mux.Handle("GET /api/v1/nodes/{id}/docker-image-inspect", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNodeDockerImageInspect)))
	mux.Handle("GET /api/v1/nodes/{id}/files", nodeFileReadChain(http.HandlerFunc(cfg.AgentHandler.HandleProxyNodeFiles)))
	mux.Handle("POST /api/v1/nodes/{id}/model-paths/scan", nodeFileReadChain(http.HandlerFunc(cfg.AgentHandler.HandleProxyNodeModelScan)))

	// GPU routes (gpu:read permission).
	gpuChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("gpu:read"),
	)
	mux.Handle("GET /api/v1/gpus", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleListGPUs)))
	mux.Handle("GET /api/v1/gpus/{id}", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleGetGPU)))

	// Phase 4: Model runtime serving APIs.
	// Backend / BackendVersion.
	backendReadChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("backend:read"),
	)
	backendWriteChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
		auth.RequirePermission("backend_runtime:write"),
	)
	mux.Handle("GET /api/v1/backends", backendReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListBackends)))
	mux.Handle("GET /api/v1/backends/{id}", backendReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetBackend)))
	mux.Handle("GET /api/v1/backends/{id}/versions", backendReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListBackendVersions)))
	mux.Handle("POST /api/v1/backends/{id}/versions", backendWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateBackendVersion)))
	mux.Handle("GET /api/v1/backend-versions", backendReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListAllBackendVersions)))
	mux.Handle("PATCH /api/v1/backend-versions/{version_id}", backendWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchBackendVersion)))
	mux.Handle("POST /api/v1/backend-versions/{version_id}/clone", backendWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCloneBackendVersion)))
	mux.Handle("DELETE /api/v1/backend-versions/{version_id}", backendWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteBackendVersion)))
	mux.Handle("POST /api/v1/backend-catalog/reload", backendWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleReloadBackendCatalog)))

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
	mux.Handle("POST /api/v1/backend-runtimes", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateBackendRuntimeFromTemplate)))
	mux.Handle("POST /api/v1/backend-runtimes/from-template", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateBackendRuntimeFromTemplate)))
	mux.Handle("GET /api/v1/backend-runtimes/{id}", brReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetBackendRuntime)))
	mux.Handle("PATCH /api/v1/backend-runtimes/{id}", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchBackendRuntime)))
	mux.Handle("DELETE /api/v1/backend-runtimes/{id}", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteBackendRuntime)))
	mux.Handle("GET /api/v1/nodes/{id}/backend-runtimes", brReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListNodeBackendRuntimes)))
	mux.Handle("POST /api/v1/nodes/{id}/backend-runtimes/enable", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleEnableNodeBackendRuntime)))
	mux.Handle("POST /api/v1/nodes/{id}/backend-runtimes/check", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCheckNodeBackendRuntime)))
	mux.Handle("PATCH /api/v1/nodes/{id}/backend-runtimes/{nbr_id}", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchNodeBackendRuntime)))
	mux.Handle("POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/check-request", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleRequestNodeBackendRuntimeCheck)))
	mux.Handle("POST /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleProbeNodeBackendRuntime)))
	mux.Handle("GET /api/v1/nodes/{id}/backend-runtimes/{nbr_id}/probe", brReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNodeBackendRuntimeProbe)))
	mux.Handle("DELETE /api/v1/nodes/{id}/backend-runtimes/{nbr_id}", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteNodeBackendRuntime)))

	// BackendRuntime clone.
	mux.Handle("POST /api/v1/backend-runtimes/{id}/clone", brWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCloneBackendRuntime)))

	// ModelArtifact CRUD.
	maReadChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.RequirePermission("model_artifact:read"))
	maWriteChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_artifact:write"))
	mux.Handle("GET /api/v1/model-artifacts", maReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListArtifacts)))
	mux.Handle("POST /api/v1/model-artifacts", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateArtifact)))
	mux.Handle("POST /api/v1/model-artifacts/discover", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDiscoverArtifact)))
	mux.Handle("GET /api/v1/model-artifacts/{id}", maReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetArtifact)))
	mux.Handle("PATCH /api/v1/model-artifacts/{id}", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchArtifact)))
	mux.Handle("DELETE /api/v1/model-artifacts/{id}", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteArtifact)))
	mux.Handle("POST /api/v1/model-artifacts/{id}/locations", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateModelLocation)))
	mux.Handle("POST /api/v1/model-artifacts/{id}/locations/{location_id}/rescan", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleRescanModelLocation)))
	mux.Handle("GET /api/v1/model-capabilities", maReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetModelCapabilityEnums)))
	mux.Handle("POST /api/v1/model-artifacts/{id}/locations/{location_id}/attest", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleAttestModelLocation)))
	mux.Handle("PATCH /api/v1/model-artifacts/{id}/locations/{location_id}", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchModelLocation)))
	mux.Handle("DELETE /api/v1/model-artifacts/{id}/locations/{location_id}", maWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteModelLocation)))

	// ModelDeployment CRUD + lifecycle.
	mdReadChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.RequirePermission("model_deployment:read"))
	mdWriteChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_deployment:write"))
	mdStartChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_deployment:start"))
	mdStopChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.CSRFMiddleware(cfg.SessionCfg), auth.RequirePermission("model_deployment:stop"))
	mux.Handle("GET /api/v1/deployments", mdReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListDeployments)))
	mux.Handle("POST /api/v1/deployments", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleCreateDeployment)))
	mux.Handle("GET /api/v1/deployments/{id}", mdReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetDeployment)))
	mux.Handle("PATCH /api/v1/deployments/{id}", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePatchDeployment)))
	mux.Handle("DELETE /api/v1/deployments/{id}", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeleteDeployment)))
	mux.Handle("POST /api/v1/deployments/{id}/dry-run", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeploymentDryRun)))
	mux.Handle("POST /api/v1/deployments/{id}/template-sync/preview", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeploymentTemplateSyncPreview)))
	mux.Handle("POST /api/v1/deployments/{id}/template-sync/apply", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandleDeploymentTemplateSyncApply)))
	mux.Handle("POST /api/v1/deployments/preflight", mdWriteChain(http.HandlerFunc(cfg.AgentHandler.HandlePreflightDeployments)))
	mux.Handle("POST /api/v1/deployments/{id}/start", mdStartChain(http.HandlerFunc(cfg.AgentHandler.HandleStartDeployment)))
	mux.Handle("POST /api/v1/deployments/{id}/stop", mdStopChain(http.HandlerFunc(cfg.AgentHandler.HandleStopDeployment)))
	mux.Handle("GET /api/v1/deployments/{id}/run-plan-groups", mdReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListRunPlanGroups)))

	// ModelInstance read.
	miReadChain := chain(auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg), auth.RequirePermission("model_instance:read"))
	mux.Handle("GET /api/v1/model-instances", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleListInstances)))
	mux.Handle("GET /api/v1/model-instances/{id}", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetInstance)))
	mux.Handle("POST /api/v1/model-instances/{id}/test", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleModelInstanceTest)))
	mux.Handle("GET /api/v1/node-run-plans/{id}", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNodeRunPlan)))
	mux.Handle("GET /api/v1/node-run-plans/{id}/command-preview", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNodeRunPlanPreview)))
	mux.Handle("GET /api/v1/node-run-plans/{id}/logs", miReadChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNodeRunPlanLogs)))

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
