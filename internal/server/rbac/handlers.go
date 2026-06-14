// Package rbac provides RBAC management API handlers.
package rbac

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"

	"github.com/google/uuid"
)

// Handler holds dependencies for RBAC API handlers.
type Handler struct {
	DB *db.DB
}

// NewHandler creates a new RBAC handler.
func NewHandler(database *db.DB) *Handler {
	return &Handler{DB: database}
}

// --- User Management (Platform Admin) ---

// HandleListUsers handles GET /api/users.
func (h *Handler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(
		`SELECT id, username, display_name, status, is_platform_admin, must_change_password, created_at, updated_at
		 FROM users ORDER BY username`,
	)
	if err != nil {
		log.Error("list users error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, username, displayName, status, createdAt, updatedAt string
		var isPA, mustChange int
		if err := rows.Scan(&id, &username, &displayName, &status, &isPA, &mustChange, &createdAt, &updatedAt); err != nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"id":                   id,
			"username":             username,
			"display_name":         displayName,
			"status":               status,
			"is_platform_admin":    isPA == 1,
			"must_change_password": mustChange == 1,
			"created_at":           createdAt,
			"updated_at":           updatedAt,
		})
	}
	if users == nil {
		users = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// HandleCreateUser handles POST /api/users.
func (h *Handler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username        string `json:"username"`
		Password        string `json:"password"`
		DisplayName     string `json:"display_name"`
		IsPlatformAdmin *bool  `json:"is_platform_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Error("hash password error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	isPA := 0
	if req.IsPlatformAdmin != nil && *req.IsPlatformAdmin {
		isPA = 1
	}

	now := time.Now().Format(time.RFC3339)
	userID := uuid.NewString()

	_, err = h.DB.Exec(
		`INSERT INTO users (id, username, display_name, password_hash, status, is_platform_admin, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'active', ?, ?, ?)`,
		userID, req.Username, displayName, passwordHash, isPA, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
			return
		}
		log.Error("create user error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":                userID,
		"username":          req.Username,
		"display_name":      displayName,
		"is_platform_admin": isPA == 1,
		"status":            "active",
	})

	log.Info("user created", "user_id", userID, "username", req.Username)
}

// HandleGetUser handles GET /api/users/{id}.
func (h *Handler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, `{"error":"user id required"}`, http.StatusBadRequest)
		return
	}

	var username, displayName, status, createdAt, updatedAt string
	var isPA, mustChange int
	err := h.DB.QueryRow(
		`SELECT username, display_name, status, is_platform_admin, must_change_password, created_at, updated_at
		 FROM users WHERE id = ?`, userID,
	).Scan(&username, &displayName, &status, &isPA, &mustChange, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("get user error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":                   userID,
		"username":             username,
		"display_name":         displayName,
		"status":               status,
		"is_platform_admin":    isPA == 1,
		"must_change_password": mustChange == 1,
		"created_at":           createdAt,
		"updated_at":           updatedAt,
	})
}

// HandleUpdateUser handles PUT /api/users/{id}.
func (h *Handler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, `{"error":"user id required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		DisplayName     *string `json:"display_name"`
		IsPlatformAdmin *bool   `json:"is_platform_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)

	if req.DisplayName != nil {
		_, err := h.DB.Exec(`UPDATE users SET display_name = ?, updated_at = ? WHERE id = ?`,
			*req.DisplayName, now, userID)
		if err != nil {
			log.Error("update user error", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
	}

	if req.IsPlatformAdmin != nil {
		if !*req.IsPlatformAdmin {
			// Check we're not removing the last platform admin.
			var count int
			h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_platform_admin = 1 AND status = 'active' AND id != ?`, userID).Scan(&count)
			if count == 0 {
				http.Error(w, `{"error":"cannot remove last active platform admin"}`, http.StatusBadRequest)
				return
			}
		}
		isPA := 0
		if *req.IsPlatformAdmin {
			isPA = 1
		}
		_, err := h.DB.Exec(`UPDATE users SET is_platform_admin = ?, updated_at = ? WHERE id = ?`,
			isPA, now, userID)
		if err != nil {
			log.Error("update user error", "error", err)
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleDisableUser handles POST /api/users/{id}/disable.
func (h *Handler) HandleDisableUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, `{"error":"user id required"}`, http.StatusBadRequest)
		return
	}

	// Check not disabling the last platform admin.
	var isPA int
	h.DB.QueryRow(`SELECT is_platform_admin FROM users WHERE id = ?`, userID).Scan(&isPA)
	if isPA == 1 {
		var count int
		h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_platform_admin = 1 AND status = 'active' AND id != ?`, userID).Scan(&count)
		if count == 0 {
			http.Error(w, `{"error":"cannot disable last active platform admin"}`, http.StatusBadRequest)
			return
		}
	}

	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`UPDATE users SET status = 'disabled', updated_at = ? WHERE id = ?`, now, userID)
	if err != nil {
		log.Error("disable user error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Revoke all sessions for this user.
	h.DB.Exec(`UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, now, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleResetPassword handles POST /api/users/{id}/reset-password.
func (h *Handler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		log.Error("hash password error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 1, updated_at = ? WHERE id = ?`,
		passwordHash, now, userID,
	)
	if err != nil {
		log.Error("reset password error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Revoke all sessions.
	h.DB.Exec(`UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, now, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- Tenant Management (Platform Admin) ---

// HandleListTenants handles GET /api/tenants.
func (h *Handler) HandleListTenants(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, name, status, created_at, updated_at FROM tenants ORDER BY name`)
	if err != nil {
		log.Error("list tenants error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tenants []map[string]interface{}
	for rows.Next() {
		var id, name, status, createdAt, updatedAt string
		if err := rows.Scan(&id, &name, &status, &createdAt, &updatedAt); err != nil {
			continue
		}
		tenants = append(tenants, map[string]interface{}{
			"id": id, "name": name, "status": status,
			"created_at": createdAt, "updated_at": updatedAt,
		})
	}
	if tenants == nil {
		tenants = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

// HandleCreateTenant handles POST /api/tenants.
func (h *Handler) HandleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		AdminUsername string `json:"admin_username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.AdminUsername == "" {
		http.Error(w, `{"error":"name and admin_username required"}`, http.StatusBadRequest)
		return
	}

	// Find admin user.
	var adminUserID string
	err := h.DB.QueryRow(`SELECT id FROM users WHERE username = ? AND status = 'active'`, req.AdminUsername).Scan(&adminUserID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"admin user not found or not active"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	now := time.Now().Format(time.RFC3339)
	tenantID := uuid.NewString()

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO tenants (id, name, status, created_at, updated_at) VALUES (?, ?, 'active', ?, ?)`,
		tenantID, req.Name, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, `{"error":"tenant name already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Create membership for admin user.
	membershipID := uuid.NewString()
	_, err = tx.Exec(
		`INSERT INTO tenant_memberships (id, tenant_id, user_id, status, created_at, updated_at) VALUES (?, ?, ?, 'active', ?, ?)`,
		membershipID, tenantID, adminUserID, now, now,
	)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Bind built-in admin role.
	var adminRoleID string
	err = tx.QueryRow(`SELECT id FROM roles WHERE tenant_id IS NULL AND name = 'admin'`).Scan(&adminRoleID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	tmrID := uuid.NewString()
	_, err = tx.Exec(
		`INSERT INTO tenant_membership_roles (id, membership_id, role_id, created_at) VALUES (?, ?, ?, ?)`,
		tmrID, membershipID, adminRoleID, now,
	)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": tenantID, "name": req.Name, "status": "active",
	})
}

// HandleGetTenant handles GET /api/tenants/{id}.
func (h *Handler) HandleGetTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	var name, status, createdAt, updatedAt string
	err := h.DB.QueryRow(
		`SELECT name, status, created_at, updated_at FROM tenants WHERE id = ?`, tenantID,
	).Scan(&name, &status, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"tenant not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": tenantID, "name": name, "status": status,
		"created_at": createdAt, "updated_at": updatedAt,
	})
}

// HandleUpdateTenant handles PUT /api/tenants/{id}.
func (h *Handler) HandleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	var req struct {
		Name *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		now := time.Now().Format(time.RFC3339)
		_, err := h.DB.Exec(`UPDATE tenants SET name = ?, updated_at = ? WHERE id = ?`, *req.Name, now, tenantID)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleDisableTenant handles POST /api/tenants/{id}/disable.
func (h *Handler) HandleDisableTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	now := time.Now().Format(time.RFC3339)
	_, err := h.DB.Exec(`UPDATE tenants SET status = 'disabled', updated_at = ? WHERE id = ?`, now, tenantID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Revoke sessions for this tenant.
	h.DB.Exec(`UPDATE sessions SET revoked_at = ? WHERE current_tenant_id = ? AND revoked_at IS NULL`, now, tenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- Tenant Membership Management (Tenant Admin) ---

// HandleListMemberships handles GET /api/tenant-memberships.
func (h *Handler) HandleListMemberships(w http.ResponseWriter, r *http.Request) {
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rows, err := h.DB.Query(
		`SELECT tm.id, tm.tenant_id, tm.user_id, u.username, tm.status, tm.created_at, tm.updated_at
		 FROM tenant_memberships tm
		 JOIN users u ON tm.user_id = u.id
		 WHERE tm.tenant_id = ?
		 ORDER BY u.username`,
		info.TenantID,
	)
	if err != nil {
		log.Error("list memberships error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var memberships []map[string]interface{}
	for rows.Next() {
		var id, tenantID, userID, username, status, createdAt, updatedAt string
		if err := rows.Scan(&id, &tenantID, &userID, &username, &status, &createdAt, &updatedAt); err != nil {
			continue
		}
		memberships = append(memberships, map[string]interface{}{
			"id": id, "tenant_id": tenantID, "user_id": userID, "username": username,
			"status": status, "created_at": createdAt, "updated_at": updatedAt,
		})
	}
	if memberships == nil {
		memberships = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memberships)
}

// HandleCreateMembership handles POST /api/tenant-memberships.
func (h *Handler) HandleCreateMembership(w http.ResponseWriter, r *http.Request) {
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Username string   `json:"username"`
		RoleIDs  []string `json:"role_ids"` // At least one role required
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || len(req.RoleIDs) == 0 {
		http.Error(w, `{"error":"username and at least one role_id required"}`, http.StatusBadRequest)
		return
	}

	// Find user.
	var userID string
	err := h.DB.QueryRow(`SELECT id FROM users WHERE username = ? AND status = 'active'`, req.Username).Scan(&userID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"user not found or not active"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Check for existing membership.
	var existingID string
	err = h.DB.QueryRow(
		`SELECT id FROM tenant_memberships WHERE tenant_id = ? AND user_id = ?`,
		info.TenantID, userID,
	).Scan(&existingID)
	if err == nil {
		http.Error(w, `{"error":"user already has a membership in this tenant"}`, http.StatusConflict)
		return
	}

	// Validate roles.
	for _, roleID := range req.RoleIDs {
		var roleTenantID sql.NullString
		err := h.DB.QueryRow(`SELECT tenant_id FROM roles WHERE id = ? AND status = 'active'`, roleID).Scan(&roleTenantID)
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"role not found: `+roleID+`"}`, http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
		// Built-in roles have null tenant_id; custom roles must belong to this tenant.
		if roleTenantID.Valid && roleTenantID.String != info.TenantID {
			http.Error(w, `{"error":"role does not belong to this tenant"}`, http.StatusBadRequest)
			return
		}
	}

	now := time.Now().Format(time.RFC3339)
	membershipID := uuid.NewString()

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO tenant_memberships (id, tenant_id, user_id, status, created_at, updated_at) VALUES (?, ?, ?, 'active', ?, ?)`,
		membershipID, info.TenantID, userID, now, now,
	)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	for _, roleID := range req.RoleIDs {
		tmrID := uuid.NewString()
		_, err = tx.Exec(
			`INSERT INTO tenant_membership_roles (id, membership_id, role_id, created_at) VALUES (?, ?, ?, ?)`,
			tmrID, membershipID, roleID, now,
		)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": membershipID, "tenant_id": info.TenantID, "user_id": userID, "status": "active",
	})
}

// HandleDisableMembership handles POST /api/tenant-memberships/{id}/disable.
func (h *Handler) HandleDisableMembership(w http.ResponseWriter, r *http.Request) {
	membershipID := r.PathValue("id")
	info := auth.SessionInfoFromContext(r.Context())

	// Verify membership belongs to current tenant.
	var tenantID string
	err := h.DB.QueryRow(`SELECT tenant_id FROM tenant_memberships WHERE id = ?`, membershipID).Scan(&tenantID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"membership not found"}`, http.StatusNotFound)
		return
	}
	if tenantID != info.TenantID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(`UPDATE tenant_memberships SET status = 'disabled', updated_at = ? WHERE id = ?`, now, membershipID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleAddMembershipRole handles POST /api/tenant-memberships/{id}/roles.
func (h *Handler) HandleAddMembershipRole(w http.ResponseWriter, r *http.Request) {
	membershipID := r.PathValue("id")
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	// P0-006: Verify the membership belongs to the current tenant.
	var membershipTenantID string
	err := h.DB.QueryRow(`SELECT tenant_id FROM tenant_memberships WHERE id = ? AND status = 'active'`, membershipID).Scan(&membershipTenantID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"membership not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("query membership error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if membershipTenantID != info.TenantID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	// Verify role belongs to this tenant (or is built-in).
	var roleTenantID sql.NullString
	err = h.DB.QueryRow(`SELECT tenant_id FROM roles WHERE id = ? AND status = 'active'`, req.RoleID).Scan(&roleTenantID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"role not found"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Error("query role error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if roleTenantID.Valid && roleTenantID.String != info.TenantID {
		http.Error(w, `{"error":"role does not belong to this tenant"}`, http.StatusForbidden)
		return
	}

	now := time.Now().Format(time.RFC3339)
	tmrID := uuid.NewString()
	_, err = h.DB.Exec(
		`INSERT INTO tenant_membership_roles (id, membership_id, role_id, created_at) VALUES (?, ?, ?, ?)`,
		tmrID, membershipID, req.RoleID, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, `{"error":"role already assigned"}`, http.StatusConflict)
			return
		}
		log.Error("add membership role error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleRemoveMembershipRole handles DELETE /api/tenant-memberships/{id}/roles/{role_id}.
func (h *Handler) HandleRemoveMembershipRole(w http.ResponseWriter, r *http.Request) {
	membershipID := r.PathValue("id")
	roleID := r.PathValue("role_id")
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// P0-006: Verify the membership belongs to the current tenant.
	var membershipTenantID string
	var status string
	err := h.DB.QueryRow(`SELECT tenant_id, status FROM tenant_memberships WHERE id = ?`, membershipID).Scan(&membershipTenantID, &status)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"membership not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("query membership error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if membershipTenantID != info.TenantID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if status == "active" {
		var roleCount int
		h.DB.QueryRow(
			`SELECT COUNT(*) FROM tenant_membership_roles WHERE membership_id = ?`,
			membershipID,
		).Scan(&roleCount)
		if roleCount <= 1 {
			http.Error(w, `{"error":"cannot remove last role from active membership"}`, http.StatusBadRequest)
			return
		}
	}

	_, err = h.DB.Exec(
		`DELETE FROM tenant_membership_roles WHERE membership_id = ? AND role_id = ?`,
		membershipID, roleID,
	)
	if err != nil {
		log.Error("remove membership role error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- Role Management (Tenant Admin) ---

// HandleListRoles handles GET /api/roles.
func (h *Handler) HandleListRoles(w http.ResponseWriter, r *http.Request) {
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Return built-in roles + current tenant custom roles.
	rows, err := h.DB.Query(
		`SELECT id, tenant_id, name, display_name, description, built_in, status, created_at, updated_at
		 FROM roles
		 WHERE (tenant_id IS NULL) OR (tenant_id = ?)
		 ORDER BY built_in DESC, name`,
		info.TenantID,
	)
	if err != nil {
		log.Error("list roles error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var roles []map[string]interface{}
	for rows.Next() {
		var id, name, displayName, description, status, createdAt, updatedAt string
		var tenantID sql.NullString
		var builtIn int
		if err := rows.Scan(&id, &tenantID, &name, &displayName, &description, &builtIn, &status, &createdAt, &updatedAt); err != nil {
			continue
		}
		role := map[string]interface{}{
			"id": id, "name": name, "display_name": displayName, "description": description,
			"built_in": builtIn == 1, "status": status,
			"created_at": createdAt, "updated_at": updatedAt,
		}
		if tenantID.Valid {
			role["tenant_id"] = tenantID.String
		}
		roles = append(roles, role)
	}
	if roles == nil {
		roles = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

// HandleCreateRole handles POST /api/roles.
func (h *Handler) HandleCreateRole(w http.ResponseWriter, r *http.Request) {
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)
	roleID := uuid.NewString()
	_, err := h.DB.Exec(
		`INSERT INTO roles (id, tenant_id, name, display_name, description, built_in, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 0, 'active', ?, ?)`,
		roleID, info.TenantID, req.Name, req.DisplayName, req.Description, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, `{"error":"role name already exists in this tenant"}`, http.StatusConflict)
			return
		}
		log.Error("create role error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": roleID, "name": req.Name, "display_name": req.DisplayName,
		"tenant_id": info.TenantID, "built_in": false, "status": "active",
	})
}

// HandleDeleteRole handles DELETE /api/roles/{id}.
func (h *Handler) HandleDeleteRole(w http.ResponseWriter, r *http.Request) {
	roleID := r.PathValue("id")
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// P0-006: Fetch role and verify it belongs to the current tenant.
	var tenantID sql.NullString
	var builtIn int
	err := h.DB.QueryRow(`SELECT tenant_id, built_in FROM roles WHERE id = ?`, roleID).Scan(&tenantID, &builtIn)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"role not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("query role error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if builtIn == 1 {
		http.Error(w, `{"error":"cannot delete built-in role"}`, http.StatusForbidden)
		return
	}
	// P0-006: Ensure tenant-scoped role belongs to the current tenant.
	if !tenantID.Valid || tenantID.String != info.TenantID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	// Check not assigned to any active membership.
	var count int
	h.DB.QueryRow(`SELECT COUNT(*) FROM tenant_membership_roles WHERE role_id = ?`, roleID).Scan(&count)
	if count > 0 {
		http.Error(w, `{"error":"role is assigned to memberships, unassign first"}`, http.StatusConflict)
		return
	}

	_, err = h.DB.Exec(`DELETE FROM roles WHERE id = ?`, roleID)
	if err != nil {
		log.Error("delete role error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleUpdateRolePermissions handles PUT /api/roles/{id}/permissions.
func (h *Handler) HandleUpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := r.PathValue("id")
	info := auth.SessionInfoFromContext(r.Context())

	// Check role.
	var tenantID sql.NullString
	var builtIn int
	err := h.DB.QueryRow(`SELECT tenant_id, built_in FROM roles WHERE id = ?`, roleID).Scan(&tenantID, &builtIn)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"role not found"}`, http.StatusNotFound)
		return
	}
	if builtIn == 1 {
		http.Error(w, `{"error":"cannot modify built-in role permissions"}`, http.StatusForbidden)
		return
	}
	if !tenantID.Valid || tenantID.String != info.TenantID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	var req struct {
		PermissionIDs []string `json:"permission_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Remove existing permissions.
	_, err = tx.Exec(`DELETE FROM role_permissions WHERE role_id = ?`, roleID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Add new permissions (only tenant-scope permissions).
	for _, permID := range req.PermissionIDs {
		var scope string
		err := tx.QueryRow(`SELECT scope FROM permissions WHERE id = ?`, permID).Scan(&scope)
		if err != nil {
			http.Error(w, `{"error":"permission not found: `+permID+`"}`, http.StatusBadRequest)
			return
		}
		if scope == "platform" {
			http.Error(w, `{"error":"cannot assign platform permission to tenant role"}`, http.StatusBadRequest)
			return
		}
		rpID := uuid.NewString()
		_, err = tx.Exec(
			`INSERT INTO role_permissions (id, role_id, permission_id, created_at) VALUES (?, ?, ?, ?)`,
			rpID, roleID, permID, now,
		)
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleListPermissions handles GET /api/permissions.
func (h *Handler) HandleListPermissions(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(
		`SELECT id, code, scope, description FROM permissions ORDER BY code`,
	)
	if err != nil {
		log.Error("list permissions error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var perms []map[string]interface{}
	for rows.Next() {
		var id, code, scope, desc string
		if err := rows.Scan(&id, &code, &scope, &desc); err != nil {
			continue
		}
		perms = append(perms, map[string]interface{}{
			"id": id, "code": code, "scope": scope, "description": desc,
		})
	}
	if perms == nil {
		perms = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perms)
}
