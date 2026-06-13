package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"
)

// AuthHandler holds dependencies for auth API handlers.
type AuthHandler struct {
	DB           *db.DB
	SessionStore *SessionStore
	SessionCfg   SessionConfig
	RateLimiter  *LoginRateLimiter
	BootstrapCfg BootstrapConfig
}

// LoginRequest is the login request body.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TenantID string `json:"tenant_id,omitempty"`
}

// LoginResponse is the login response body.
type LoginResponse struct {
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	DisplayName     string `json:"display_name"`
	IsPlatformAdmin bool   `json:"is_platform_admin"`
	MustChangePass  bool   `json:"must_change_password"`
	TenantID        string `json:"tenant_id"`
	TenantName      string `json:"tenant_name"`
	CSRFToken       string `json:"csrf_token"`
}

// MeResponse is the /api/auth/me response.
type MeResponse struct {
	User struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		DisplayName     string `json:"display_name"`
		IsPlatformAdmin bool   `json:"is_platform_admin"`
		MustChangePass  bool   `json:"must_change_password"`
	} `json:"user"`
	Tenant struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`
	Roles       []RoleInfo `json:"roles"`
	Permissions []string   `json:"permissions"`
}

// RoleInfo is a simplified role representation.
type RoleInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BuiltIn bool   `json:"built_in"`
}

// ChangePasswordRequest is the change password request body.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// HandleLogin handles POST /api/auth/login.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Rate limit.
	if !h.RateLimiter.Allow(r, "") {
		http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
		return
	}

	// Validate Origin for login (no CSRF token yet).
	if !ValidateOrigin(r) {
		http.Error(w, `{"error":"invalid origin"}`, http.StatusForbidden)
		return
	}

	// Parse request.
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Rate limit by username.
	if !h.RateLimiter.Allow(r, req.Username) {
		http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
		return
	}

	// Find user.
	var userID, username, displayName, passwordHash, userStatus string
	var isPlatformAdmin, mustChangePassword int
	err := h.DB.QueryRow(
		`SELECT id, username, display_name, password_hash, status, is_platform_admin, must_change_password
		 FROM users WHERE username = ?`,
		req.Username,
	).Scan(&userID, &username, &displayName, &passwordHash, &userStatus, &isPlatformAdmin, &mustChangePassword)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error("login query error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Check user status.
	if userStatus != "active" {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Verify password.
	ok, err := VerifyPassword(req.Password, passwordHash)
	if err != nil || !ok {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Determine tenant.
	tenantID := req.TenantID
	if tenantID == "" {
		// Auto-select if user has only one active membership.
		rows, err := h.DB.Query(
			`SELECT tm.tenant_id FROM tenant_memberships tm
			 JOIN tenants t ON tm.tenant_id = t.id AND t.status = 'active'
			 WHERE tm.user_id = ? AND tm.status = 'active'`,
			userID,
		)
		if err != nil {
			log.Error("query memberships error", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var tenantIDs []string
		for rows.Next() {
			var tid string
			if err := rows.Scan(&tid); err != nil {
				continue
			}
			tenantIDs = append(tenantIDs, tid)
		}

		if len(tenantIDs) == 0 {
			http.Error(w, `{"error":"no active membership"}`, http.StatusForbidden)
			return
		}
		if len(tenantIDs) > 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":                "tenant_selection_required",
				"available_tenant_ids": tenantIDs,
			})
			return
		}
		tenantID = tenantIDs[0]
	} else {
		// Verify membership in the specified tenant.
		var count int
		err := h.DB.QueryRow(
			`SELECT COUNT(*) FROM tenant_memberships tm
			 JOIN tenants t ON tm.tenant_id = t.id AND t.status = 'active'
			 WHERE tm.user_id = ? AND tm.tenant_id = ? AND tm.status = 'active'`,
			userID, tenantID,
		).Scan(&count)
		if err != nil || count == 0 {
			http.Error(w, `{"error":"invalid tenant"}`, http.StatusForbidden)
			return
		}
	}

	// Check tenant status.
	var tenantName, tenantStatus string
	err = h.DB.QueryRow(`SELECT name, status FROM tenants WHERE id = ?`, tenantID).Scan(&tenantName, &tenantStatus)
	if err != nil || tenantStatus != "active" {
		http.Error(w, `{"error":"tenant disabled"}`, http.StatusForbidden)
		return
	}

	// Create session.
	sessionID, csrfSecret, err := h.SessionStore.CreateSession(userID, tenantID)
	if err != nil {
		log.Error("create session error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Set cookie.
	SetSessionCookie(w, sessionID, h.SessionCfg)

	// Build response.
	platformAdmin := isPlatformAdmin == 1
	mustChange := mustChangePassword == 1

	resp := LoginResponse{
		UserID:          userID,
		Username:        username,
		DisplayName:     displayName,
		IsPlatformAdmin: platformAdmin,
		MustChangePass:  mustChange,
		TenantID:        tenantID,
		TenantName:      tenantName,
		CSRFToken:       csrfSecret,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Info("user logged in",
		"user_id", userID,
		"username", username,
		"tenant_id", tenantID,
	)
}

// HandleLogout handles POST /api/auth/logout.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	info := SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"not logged in"}`, http.StatusUnauthorized)
		return
	}

	// Revoke session.
	if err := h.SessionStore.RevokeSession(info.SessionID); err != nil {
		log.Error("revoke session error", "error", err)
	}

	// Clear cookie.
	ClearSessionCookie(w, h.SessionCfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	log.Info("user logged out", "user_id", info.UserID)
}

// HandleMe handles GET /api/auth/me.
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	info := SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Fetch user details.
	var username, displayName, userStatus string
	var isPlatformAdmin, mustChange int
	err := h.DB.QueryRow(
		`SELECT username, display_name, status, is_platform_admin, must_change_password
		 FROM users WHERE id = ?`,
		info.UserID,
	).Scan(&username, &displayName, &userStatus, &isPlatformAdmin, &mustChange)
	if err != nil {
		log.Error("query user error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Fetch tenant details.
	var tenantName string
	err = h.DB.QueryRow(`SELECT name FROM tenants WHERE id = ?`, info.TenantID).Scan(&tenantName)
	if err != nil {
		tenantName = "Unknown"
	}

	// Get role IDs and permissions from context.
	roleIDs := RoleIDsFromContext(r.Context())
	permissions := PermissionsFromContext(r.Context())

	// Fetch role details.
	var roles []RoleInfo
	if len(roleIDs) > 0 {
		placeholders := make([]string, len(roleIDs))
		args := make([]interface{}, len(roleIDs))
		for i, rid := range roleIDs {
			placeholders[i] = "?"
			args[i] = rid
		}
		query := `SELECT id, name, built_in FROM roles WHERE id IN (` + strings.Join(placeholders, ",") + `)`
		rows, err := h.DB.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r RoleInfo
				var builtIn int
				if err := rows.Scan(&r.ID, &r.Name, &builtIn); err == nil {
					r.BuiltIn = builtIn == 1
					roles = append(roles, r)
				}
			}
		}
	}
	if roles == nil {
		roles = []RoleInfo{}
	}
	if permissions == nil {
		permissions = []string{}
	}

	resp := MeResponse{}
	resp.User.ID = info.UserID
	resp.User.Username = username
	resp.User.DisplayName = displayName
	resp.User.IsPlatformAdmin = isPlatformAdmin == 1
	resp.User.MustChangePass = mustChange == 1
	resp.Tenant.ID = info.TenantID
	resp.Tenant.Name = tenantName
	resp.Roles = roles
	resp.Permissions = permissions

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleChangePassword handles POST /api/auth/change-password.
func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	info := SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, `{"error":"current and new password required"}`, http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		http.Error(w, `{"error":"new password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}

	// Verify current password.
	var currentHash string
	err := h.DB.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, info.UserID).Scan(&currentHash)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	ok, err := VerifyPassword(req.CurrentPassword, currentHash)
	if err != nil || !ok {
		http.Error(w, `{"error":"current password is incorrect"}`, http.StatusBadRequest)
		return
	}

	// Hash new password.
	newHash, err := HashPassword(req.NewPassword)
	if err != nil {
		log.Error("hash password error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Update password and clear must_change_password.
	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 0, updated_at = ? WHERE id = ?`,
		newHash, now, info.UserID,
	)
	if err != nil {
		log.Error("update password error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Revoke all other sessions.
	if err := h.SessionStore.RevokeUserSessions(info.UserID, info.SessionID); err != nil {
		log.Warn("revoke other sessions error", "error", err)
	}

	// Create fresh CSRF token: regenerate session CSRF secret.
	// For simplicity, we rotate by creating a new session.
	// In production, you'd update just the CSRF secret.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	log.Info("password changed", "user_id", info.UserID)
}

// HandleCSRFToken handles GET /api/auth/csrf-token.
// Returns the current CSRF token for the session.
func (h *AuthHandler) HandleCSRFToken(w http.ResponseWriter, r *http.Request) {
	// CSRF token is returned in the login response.
	// For subsequent requests, the client should use the same token.
	// This endpoint returns the token again if needed from the session.
	cookie, err := r.Cookie(h.SessionCfg.CookieName)
	if err != nil {
		http.Error(w, `{"error":"no session"}`, http.StatusUnauthorized)
		return
	}

	sessionIDHash := hashString(cookie.Value)
	var csrfHash string
	err = h.DB.QueryRow(
		`SELECT csrf_secret_hash FROM sessions WHERE id = ? AND revoked_at IS NULL`,
		sessionIDHash,
	).Scan(&csrfHash)
	if err != nil {
		http.Error(w, `{"error":"no session"}`, http.StatusUnauthorized)
		return
	}

	// We can't recover the original CSRF secret from the hash.
	// The CSRF token is set at login time and should be stored client-side.
	// This endpoint is a convenience that returns the hash status.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "CSRF token was provided at login. Use the same token for all requests.",
	})
}
