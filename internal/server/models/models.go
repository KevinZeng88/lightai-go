// Package models defines all database models for LightAI Go.
package models

import "time"

// Tenant represents a tenant boundary for resources and members.
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // active / disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents a global local user account.
type User struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	DisplayName        string    `json:"display_name"`
	PasswordHash       string    `json:"-"`
	Status             string    `json:"status"` // active / disabled
	IsPlatformAdmin    bool      `json:"is_platform_admin"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TenantMembership represents a user's membership in a tenant.
type TenantMembership struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"` // active / disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Role represents a built-in or custom role.
type Role struct {
	ID          string    `json:"id"`
	TenantID    *string   `json:"tenant_id,omitempty"` // nil for built-in
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	BuiltIn     bool      `json:"built_in"`
	Status      string    `json:"status"` // active / disabled
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a system-readonly permission code.
type Permission struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Scope       string    `json:"scope"` // tenant / platform
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RolePermission binds a role to a permission.
type RolePermission struct {
	ID           string    `json:"id"`
	RoleID       string    `json:"role_id"`
	PermissionID string    `json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// TenantMembershipRole binds a membership to a role.
type TenantMembershipRole struct {
	ID           string    `json:"id"`
	MembershipID string    `json:"membership_id"`
	RoleID       string    `json:"role_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents a server-side user session.
type Session struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	CurrentTenantID string     `json:"current_tenant_id"`
	CSRFSecretHash  string     `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked returns true if the session has been revoked.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsValid returns true if the session is neither expired nor revoked.
func (s *Session) IsValid() bool {
	return !s.IsExpired() && !s.IsRevoked()
}

// Model runtime structs have been moved to separate files:
// - backend.go: InferenceBackend, BackendVersion
// - runtime.go: BackendRuntimeTemplate, BackendRuntime, NodeRuntimeOverride
// - artifact.go: ModelArtifact
// - deployment.go: ModelDeployment
// - instance.go: ModelInstance, GpuLease, AgentTask
// - runplan.go: ResolvedRunPlan
