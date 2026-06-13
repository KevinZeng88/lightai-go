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
	// Auth endpoints (no session required for login/CSRF).
	mux.HandleFunc("POST /api/auth/login", cfg.AuthHandler.HandleLogin)
	mux.HandleFunc("GET /api/auth/csrf-token", cfg.AuthHandler.HandleCSRFToken)

	// Session-protected auth routes.
	mux.Handle("POST /api/auth/logout", sessionChain(cfg, cfg.AuthHandler.HandleLogout))
	mux.Handle("POST /api/auth/change-password", sessionChain(cfg, cfg.AuthHandler.HandleChangePassword))
	mux.Handle("GET /api/auth/me", sessionChain(cfg, cfg.AuthHandler.HandleMe))

	// Platform admin routes.
	mux.Handle("GET /api/users", platformChain(cfg, cfg.RBACHandler.HandleListUsers))
	mux.Handle("POST /api/users", platformChain(cfg, cfg.RBACHandler.HandleCreateUser))
	mux.Handle("GET /api/users/{id}", platformChain(cfg, cfg.RBACHandler.HandleGetUser))
	mux.Handle("PUT /api/users/{id}", platformChain(cfg, cfg.RBACHandler.HandleUpdateUser))
	mux.Handle("POST /api/users/{id}/disable", platformChain(cfg, cfg.RBACHandler.HandleDisableUser))
	mux.Handle("POST /api/users/{id}/reset-password", platformChain(cfg, cfg.RBACHandler.HandleResetPassword))

	mux.Handle("GET /api/tenants", platformChain(cfg, cfg.RBACHandler.HandleListTenants))
	mux.Handle("POST /api/tenants", platformChain(cfg, cfg.RBACHandler.HandleCreateTenant))
	mux.Handle("GET /api/tenants/{id}", platformChain(cfg, cfg.RBACHandler.HandleGetTenant))
	mux.Handle("PUT /api/tenants/{id}", platformChain(cfg, cfg.RBACHandler.HandleUpdateTenant))
	mux.Handle("POST /api/tenants/{id}/disable", platformChain(cfg, cfg.RBACHandler.HandleDisableTenant))

	// Tenant admin routes.
	mux.Handle("GET /api/tenant-memberships", tenantChain(cfg, cfg.RBACHandler.HandleListMemberships, "membership:read"))
	mux.Handle("POST /api/tenant-memberships", tenantChain(cfg, cfg.RBACHandler.HandleCreateMembership, "membership:write"))
	mux.Handle("POST /api/tenant-memberships/{id}/disable", tenantChain(cfg, cfg.RBACHandler.HandleDisableMembership, "membership:write"))
	mux.Handle("POST /api/tenant-memberships/{id}/roles", tenantChain(cfg, cfg.RBACHandler.HandleAddMembershipRole, "membership:write"))
	mux.Handle("DELETE /api/tenant-memberships/{id}/roles/{role_id}", tenantChain(cfg, cfg.RBACHandler.HandleRemoveMembershipRole, "membership:write"))

	mux.Handle("GET /api/roles", tenantChain(cfg, cfg.RBACHandler.HandleListRoles, "role:read"))
	mux.Handle("POST /api/roles", tenantChain(cfg, cfg.RBACHandler.HandleCreateRole, "role:write"))
	mux.Handle("DELETE /api/roles/{id}", tenantChain(cfg, cfg.RBACHandler.HandleDeleteRole, "role:write"))
	mux.Handle("PUT /api/roles/{id}/permissions", tenantChain(cfg, cfg.RBACHandler.HandleUpdateRolePermissions, "role:write"))

	mux.Handle("GET /api/permissions", tenantChain(cfg, cfg.RBACHandler.HandleListPermissions, "role:read"))

	// Observability status (no auth — used by Web frontend to avoid CORS).
	mux.HandleFunc("GET /api/observability/status", HandleObservabilityStatus)

	// Agent API routes (use agent token, not session).
	agentMW := auth.AgentAuthMiddleware(cfg.AgentToken)
	mux.Handle("POST /api/agent/register", agentMW(http.HandlerFunc(cfg.AgentHandler.HandleRegister)))
	mux.Handle("POST /api/agent/heartbeat", agentMW(http.HandlerFunc(cfg.AgentHandler.HandleHeartbeat)))
	mux.Handle("POST /api/agent/resources/report", agentMW(http.HandlerFunc(cfg.ResourceHandler.HandleResourceReport)))

	// Resource routes (node:read permission).
	resourceChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("node:read"),
	)
	mux.Handle("GET /api/nodes", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleListNodes)))
	mux.Handle("GET /api/nodes/{id}", resourceChain(http.HandlerFunc(cfg.AgentHandler.HandleGetNode)))

	// GPU routes (gpu:read permission).
	gpuChain := chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.RequirePermission("gpu:read"),
	)
	mux.Handle("GET /api/gpus", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleListGPUs)))
	mux.Handle("GET /api/gpus/{id}", gpuChain(http.HandlerFunc(cfg.ResourceHandler.HandleGetGPU)))
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
	return sessionMW(csrfMW(auth.RequirePlatformAdmin(http.HandlerFunc(h))))
}

func tenantChain(cfg RouterConfig, h http.HandlerFunc, permission string) http.Handler {
	return chain(
		auth.SessionMiddleware(cfg.SessionStore, cfg.DB, cfg.SessionCfg),
		auth.CSRFMiddleware(cfg.SessionCfg),
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
