package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"

	"github.com/google/uuid"
)

// BootstrapConfig holds bootstrap admin configuration.
type BootstrapConfig struct {
	Username            string
	Password            string
	PasswordEnv         string
	ForceChangePassword bool
}

// PermissionDef defines a permission code for initialization.
type PermissionDef struct {
	Code        string
	Scope       string // tenant / platform
	Description string
}

// BuiltinRoleDef defines a built-in role with its permissions.
type BuiltinRoleDef struct {
	Name        string
	DisplayName string
	Description string
	Permissions []string // permission codes
}

// PermissionCatalog returns the full permission catalog.
func PermissionCatalog() []PermissionDef {
	return []PermissionDef{
		// Read-only permissions.
		{Code: "dashboard:read", Scope: "tenant", Description: "View dashboard"},
		{Code: "node:read", Scope: "tenant", Description: "View nodes"},
		{Code: "node:transfer", Scope: "tenant", Description: "Transfer node to another tenant"},
		{Code: "gpu:read", Scope: "tenant", Description: "View GPUs"},
		{Code: "monitoring:read", Scope: "tenant", Description: "View monitoring"},
		{Code: "log:read", Scope: "tenant", Description: "View logs"},

		// Runtime permissions.
		{Code: "runtime:read", Scope: "tenant", Description: "View runtime environments"},
		{Code: "runtime:write", Scope: "tenant", Description: "Manage runtime environments"},

		// Model permissions.
		{Code: "model:read", Scope: "tenant", Description: "View models"},
		{Code: "model:write", Scope: "tenant", Description: "Manage models"},

		// Deployment permissions.
		{Code: "deployment:read", Scope: "tenant", Description: "View model deployments"},
		{Code: "deployment:write", Scope: "tenant", Description: "Manage model deployments"},

		// Instance permissions.
		{Code: "instance:read", Scope: "tenant", Description: "View instances"},
		{Code: "instance:write", Scope: "tenant", Description: "Manage instances"},
		{Code: "instance:operate", Scope: "tenant", Description: "Start/stop/restart instances"},

		// Task permissions.
		{Code: "task:read", Scope: "tenant", Description: "View tasks"},
		{Code: "audit:read", Scope: "tenant", Description: "View audit logs"},

		// Membership permissions.
		{Code: "membership:read", Scope: "tenant", Description: "View memberships"},
		{Code: "membership:write", Scope: "tenant", Description: "Manage memberships"},

		// Role permissions.
		{Code: "role:read", Scope: "tenant", Description: "View roles"},
		{Code: "role:write", Scope: "tenant", Description: "Manage custom roles"},

		// Tenant settings.
		{Code: "tenant:settings:write", Scope: "tenant", Description: "Manage tenant settings"},

		// Platform permissions.
		{Code: "platform:user:manage", Scope: "platform", Description: "Manage global users"},
		{Code: "platform:tenant:manage", Scope: "platform", Description: "Manage tenants"},
		{Code: "platform:settings:write", Scope: "platform", Description: "Manage platform settings"},
	}
}

// BuiltinRoles returns the built-in role definitions.
func BuiltinRoles() []BuiltinRoleDef {
	return []BuiltinRoleDef{
		{
			Name:        "admin",
			DisplayName: "Admin",
			Description: "Tenant administrator with full tenant scope access",
			Permissions: []string{
				// Viewer permissions.
				"dashboard:read", "node:read", "gpu:read", "monitoring:read", "log:read",
				"runtime:read", "model:read", "instance:read", "deployment:read", "task:read",
				// Operator permissions.
				"runtime:write", "model:write", "instance:write", "instance:operate", "deployment:write",
				// Admin permissions.
				"membership:read", "membership:write", "role:read", "role:write",
				"node:transfer", "tenant:settings:write", "audit:read",
			},
		},
		{
			Name:        "operator",
			DisplayName: "Operator",
			Description: "Tenant operator with resource management access",
			Permissions: []string{
				// Viewer permissions.
				"dashboard:read", "node:read", "gpu:read", "monitoring:read", "log:read",
				"runtime:read", "model:read", "instance:read", "deployment:read", "task:read",
				// Operator permissions.
				"runtime:write", "model:write", "instance:write", "instance:operate", "deployment:write",
			},
		},
		{
			Name:        "viewer",
			DisplayName: "Viewer",
			Description: "Read-only tenant access",
			Permissions: []string{
				"dashboard:read", "node:read", "gpu:read", "monitoring:read", "log:read",
				"runtime:read", "model:read", "instance:read", "deployment:read", "task:read",
			},
		},
	}
}

// InitBootstrap initializes the permission catalog, built-in roles,
// default tenant, and bootstrap admin user. All operations are idempotent.
func InitBootstrap(database *db.DB, cfg BootstrapConfig) error {
	// Start transaction.
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)

	// 1. Initialize permission catalog.
	permMap := make(map[string]string) // code -> id
	for _, p := range PermissionCatalog() {
		permID := uuid.NewString()
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO permissions (id, code, scope, description, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			permID, p.Code, p.Scope, p.Description, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert permission %s: %w", p.Code, err)
		}
		// Fetch the actual ID (may already exist).
		row := tx.QueryRow(`SELECT id FROM permissions WHERE code = ?`, p.Code)
		var actualID string
		if err := row.Scan(&actualID); err != nil {
			return fmt.Errorf("query permission %s: %w", p.Code, err)
		}
		permMap[p.Code] = actualID
	}

	// 2. Initialize built-in roles and their permissions.
	for _, role := range BuiltinRoles() {
		roleID := uuid.NewString()
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO roles (id, tenant_id, name, display_name, description, built_in, status, created_at, updated_at)
			 VALUES (?, NULL, ?, ?, ?, 1, 'active', ?, ?)`,
			roleID, role.Name, role.DisplayName, role.Description, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert role %s: %w", role.Name, err)
		}
		// Fetch actual role ID.
		row := tx.QueryRow(`SELECT id FROM roles WHERE tenant_id IS NULL AND name = ?`, role.Name)
		var actualRoleID string
		if err := row.Scan(&actualRoleID); err != nil {
			return fmt.Errorf("query role %s: %w", role.Name, err)
		}

		// Insert role permissions.
		for _, permCode := range role.Permissions {
			permID, ok := permMap[permCode]
			if !ok {
				return fmt.Errorf("permission %s not found for role %s", permCode, role.Name)
			}
			rpID := uuid.NewString()
			_, err := tx.Exec(
				`INSERT OR IGNORE INTO role_permissions (id, role_id, permission_id, created_at)
				 VALUES (?, ?, ?, ?)`,
				rpID, actualRoleID, permID, now,
			)
			if err != nil {
				return fmt.Errorf("insert role_permission %s/%s: %w", role.Name, permCode, err)
			}
		}
	}

	// 3. Initialize default tenant (idempotent: check by slug first).
	var actualTenantID string
	row := tx.QueryRow(`SELECT id FROM tenants WHERE slug = 'default'`)
	err = row.Scan(&actualTenantID)
	if err == sql.ErrNoRows {
		// Create default tenant with deterministic UUID.
		tenantID := "a0000000-0000-0000-0000-000000000001"
		_, err = tx.Exec(
			`INSERT INTO tenants (id, slug, name, status, created_at, updated_at)
			 VALUES (?, 'default', 'Default Tenant', 'active', ?, ?)`,
			tenantID, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert default tenant: %w", err)
		}
		actualTenantID = tenantID
	} else if err != nil {
		return fmt.Errorf("query default tenant: %w", err)
	}

	// 4. Determine bootstrap admin password.
	password := cfg.Password
	if cfg.PasswordEnv != "" {
		if envPass := os.Getenv(cfg.PasswordEnv); envPass != "" {
			password = envPass
		}
	}
	autoGenerated := false
	if password == "" {
		// Generate random password.
		pwBytes := make([]byte, 16)
		if _, err := rand.Read(pwBytes); err != nil {
			return fmt.Errorf("generate password: %w", err)
		}
		password = hex.EncodeToString(pwBytes)
		autoGenerated = true
	}

	// 5. Hash password.
	passwordHash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// 6. Create bootstrap admin user.
	userID := uuid.NewString()
	mustChange := 0
	if cfg.ForceChangePassword {
		mustChange = 1
	}
	userCreated := false
	result, err := tx.Exec(
		`INSERT OR IGNORE INTO users (id, username, display_name, password_hash, status, is_platform_admin, must_change_password, created_at, updated_at)
		 VALUES (?, ?, 'Administrator', ?, 'active', 1, ?, ?, ?)`,
		userID, cfg.Username, passwordHash, mustChange, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert bootstrap user: %w", err)
	}
	if n, _ := result.RowsAffected(); n > 0 {
		userCreated = true
	}

	// Fetch actual user ID.
	row = tx.QueryRow(`SELECT id FROM users WHERE username = ?`, cfg.Username)
	var actualUserID string
	if err := row.Scan(&actualUserID); err != nil {
		return fmt.Errorf("query bootstrap user: %w", err)
	}

	// 7. Create membership for bootstrap user in default tenant.
	membershipID := uuid.NewString()
	_, err = tx.Exec(
		`INSERT OR IGNORE INTO tenant_memberships (id, tenant_id, user_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'active', ?, ?)`,
		membershipID, actualTenantID, actualUserID, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert membership: %w", err)
	}
	// Fetch actual membership ID.
	row = tx.QueryRow(`SELECT id FROM tenant_memberships WHERE tenant_id = ? AND user_id = ?`, actualTenantID, actualUserID)
	var actualMembershipID string
	if err := row.Scan(&actualMembershipID); err != nil {
		return fmt.Errorf("query membership: %w", err)
	}

	// 8. Bind built-in admin role to membership.
	row = tx.QueryRow(`SELECT id FROM roles WHERE tenant_id IS NULL AND name = 'admin'`)
	var adminRoleID string
	if err := row.Scan(&adminRoleID); err != nil {
		return fmt.Errorf("query admin role: %w", err)
	}
	tmrID := uuid.NewString()
	_, err = tx.Exec(
		`INSERT OR IGNORE INTO tenant_membership_roles (id, membership_id, role_id, created_at)
		 VALUES (?, ?, ?, ?)`,
		tmrID, actualMembershipID, adminRoleID, now,
	)
	if err != nil {
		return fmt.Errorf("insert membership role: %w", err)
	}

	// Commit.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap: %w", err)
	}

	if userCreated {
		log.Info("bootstrap initialization complete",
			"user_created", true,
			"username", cfg.Username,
			"auto_generated", autoGenerated,
		)

		// Write initial credentials file (idempotent — won't overwrite).
		credPath := "runtime/initial-credentials.txt"
		if err := writeInitialCredentials(credPath, cfg.Username, password); err != nil {
			log.Warn("failed to write initial credentials file",
				"path", credPath,
				"error", err,
			)
		} else {
			fmt.Fprintf(os.Stderr, "Initial credentials written to %s\n", credPath)
		}
	} else {
		log.Info("bootstrap initialization skipped (already exists)")
	}

	return nil
}

// writeInitialCredentials writes the initial admin credentials to path.
// If the file already exists it is NOT overwritten (idempotent).
// The file is created with 0600 permissions.
func writeInitialCredentials(path, username, password string) error {
	// Do not overwrite existing credentials.
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create runtime dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	content := fmt.Sprintf(`============================================
LightAI Go - Initial Credentials
Generated: %s
============================================

[Web/Admin]
Username: %s
Password: %s
Note: Change this password after first login.

[Grafana]
See: runtime/initial-credentials.txt (appended by start-observability.sh)
Note: If Grafana is in use, start observability to record its password.
`, time.Now().Format(time.RFC3339), username, password)

	if _, err := f.WriteString(content); err != nil {
		return err
	}
	return nil
}
