// Package types defines shared types used across LightAI Go.
package types

import "time"

// HealthResponse is the standard health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// MetricTarget represents a Prometheus scrape target.
type MetricTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// AuditFields holds common audit information for database entities.
type AuditFields struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
}

// ResourceOwnership holds tenant and owner information.
type ResourceOwnership struct {
	TenantID string  `json:"tenant_id"`
	OwnerID  *string `json:"owner_id,omitempty"`
}
