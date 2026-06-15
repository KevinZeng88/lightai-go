package auth

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"
)

type contextKey string

const (
	ctxKeySessionInfo contextKey = "session_info"
	ctxKeyPermissions contextKey = "permissions"
	ctxKeyRoleIDs     contextKey = "role_ids"
)

// SessionInfoFromContext extracts session info from the request context.
func SessionInfoFromContext(ctx context.Context) *SessionInfo {
	info, _ := ctx.Value(ctxKeySessionInfo).(*SessionInfo)
	return info
}

// NewContextWithSessionInfo returns a context with session info injected.
// Exported for test usage (tenant isolation tests, etc.).
func NewContextWithSessionInfo(ctx context.Context, info *SessionInfo) context.Context {
	return context.WithValue(ctx, ctxKeySessionInfo, info)
}

// PermissionsFromContext extracts permission codes from the request context.

// NewContextWithPermissions returns a context with the given permissions set.
// This is intended for test use only.
func NewContextWithPermissions(ctx context.Context, perms []string) context.Context {
	return context.WithValue(ctx, ctxKeyPermissions, perms)
}
func PermissionsFromContext(ctx context.Context) []string {
	perms, _ := ctx.Value(ctxKeyPermissions).([]string)
	return perms
}

// RoleIDsFromContext extracts role IDs from the request context.
func RoleIDsFromContext(ctx context.Context) []string {
	ids, _ := ctx.Value(ctxKeyRoleIDs).([]string)
	return ids
}

// SessionMiddleware extracts the session from the cookie, validates it,
// and resolves roles and permissions for the request.
func SessionMiddleware(sessionStore *SessionStore, database *db.DB, sessionCfg SessionConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract session cookie.
			cookie, err := r.Cookie(sessionCfg.CookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Validate session.
			info, err := sessionStore.ValidateSession(cookie.Value)
			if err != nil {
				log.Warn("session validation error", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if info == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Resolve roles and permissions.
			roleIDs, permissions, err := resolvePermissions(database, info.UserID, info.TenantID, info.IsPlatformAdmin)
			if err != nil {
				log.Error("failed to resolve permissions", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// Store in context.
			ctx := context.WithValue(r.Context(), ctxKeySessionInfo, info)
			ctx = context.WithValue(ctx, ctxKeyPermissions, permissions)
			ctx = context.WithValue(ctx, ctxKeyRoleIDs, roleIDs)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission middleware checks that the request has the required permission code.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := SessionInfoFromContext(r.Context())
			if info == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			perms := PermissionsFromContext(r.Context())
			if !hasPermission(perms, permission) {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePlatformAdmin middleware checks that the request is from a platform admin.
func RequirePlatformAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := SessionInfoFromContext(r.Context())
		if info == nil || !info.IsPlatformAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePasswordNotExpired middleware rejects state-changing requests
// when the user has must_change_password set (unless the request is
// to change password itself). P0-007: backend enforcement of password expiry.
func RequirePasswordNotExpired(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only enforce on state-changing methods.
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// Allow change-password and logout to proceed even with expired password.
			if r.URL.Path == "/api/auth/change-password" || r.URL.Path == "/api/auth/logout" {
				next.ServeHTTP(w, r)
				return
			}

			info := SessionInfoFromContext(r.Context())
			if info == nil {
				next.ServeHTTP(w, r)
				return
			}

			var mustChange int
			err := database.QueryRow(
				`SELECT must_change_password FROM users WHERE id = ?`,
				info.UserID,
			).Scan(&mustChange)
			if err == nil && mustChange == 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"password change required","code":"must_change_password"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AgentAuthMiddleware validates the agent bootstrap token.
func AgentAuthMiddleware(agentToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != agentToken {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRFMiddleware validates the CSRF token for state-changing requests.
func CSRFMiddleware(sessionCfg SessionConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check state-changing methods.
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			info := SessionInfoFromContext(r.Context())
			if info == nil {
				// No session - let auth middleware handle it.
				next.ServeHTTP(w, r)
				return
			}

			if !ValidateCSRF(r, info.CSRFSecretHash) {
				http.Error(w, `{"error":"invalid csrf token"}`, http.StatusForbidden)
				return
			}

			if !ValidateOrigin(r) {
				http.Error(w, `{"error":"invalid origin"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// resolvePermissions queries the database for all roles and permissions
// for a given user in a given tenant.
func resolvePermissions(database *db.DB, userID, tenantID string, isPlatformAdmin bool) (roleIDs []string, permissions []string, err error) {
	// Get membership.
	var membershipID string
	err = database.QueryRow(
		`SELECT id FROM tenant_memberships WHERE user_id = ? AND tenant_id = ? AND status = 'active'`,
		userID, tenantID,
	).Scan(&membershipID)
	if err == sql.ErrNoRows {
		// No membership - no tenant roles.
		membershipID = ""
	} else if err != nil {
		return nil, nil, err
	}

	permSet := make(map[string]bool)

	if membershipID != "" {
		// Get roles via membership.
		rows, err := database.Query(
			`SELECT r.id, p.code
			 FROM tenant_membership_roles tmr
			 JOIN roles r ON tmr.role_id = r.id AND r.status = 'active'
			 JOIN role_permissions rp ON r.id = rp.role_id
			 JOIN permissions p ON rp.permission_id = p.id
			 WHERE tmr.membership_id = ?`,
			membershipID,
		)
		if err != nil {
			return nil, nil, err
		}
		defer rows.Close()

		roleSet := make(map[string]bool)
		for rows.Next() {
			var roleID, permCode string
			if err := rows.Scan(&roleID, &permCode); err != nil {
				return nil, nil, err
			}
			roleSet[roleID] = true
			permSet[permCode] = true
		}
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}

		for rid := range roleSet {
			roleIDs = append(roleIDs, rid)
		}
	}

	// Add platform admin permissions.
	if isPlatformAdmin {
		permSet["platform:user:manage"] = true
		permSet["platform:tenant:manage"] = true
		permSet["platform:settings:write"] = true
	}

	for p := range permSet {
		permissions = append(permissions, p)
	}

	return roleIDs, permissions, nil
}

func hasPermission(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == required {
			return true
		}
	}
	return false
}
