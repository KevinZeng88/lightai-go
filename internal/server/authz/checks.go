package authz

import (
	"database/sql"
	"net/http"

	"lightai-go/internal/server/auth"
)

// CheckNodeTenant verifies the node belongs to the caller's tenant.
// Returns true if authorized (including platform admin bypass).
func CheckNodeTenant(r *http.Request, db *sql.DB, nodeID string) bool {
	if isPlatformAdmin(r) {
		return true
	}
	var tid string
	err := db.QueryRow("SELECT tenant_id FROM nodes WHERE id=?", nodeID).Scan(&tid)
	if err != nil {
		return false
	}
	return tid == tenantID(r)
}

// CheckNBRTenant verifies the NodeBackendRuntime belongs to the caller's tenant.
func CheckNBRTenant(r *http.Request, db *sql.DB, nbrID string) bool {
	if isPlatformAdmin(r) {
		return true
	}
	var tid string
	err := db.QueryRow("SELECT tenant_id FROM node_backend_runtimes WHERE id=?", nbrID).Scan(&tid)
	if err != nil {
		return false
	}
	return tid == tenantID(r)
}

// CheckModelRootTenant verifies the model root belongs to the caller's tenant.
func CheckModelRootTenant(r *http.Request, db *sql.DB, rootID string) bool {
	if isPlatformAdmin(r) {
		return true
	}
	var tid string
	err := db.QueryRow("SELECT tenant_id FROM node_model_roots WHERE id=?", rootID).Scan(&tid)
	if err != nil {
		return false
	}
	return tid == tenantID(r)
}

// CheckModelLocationTenant verifies the model location belongs to the caller's tenant.
func CheckModelLocationTenant(r *http.Request, db *sql.DB, locationID string) bool {
	if isPlatformAdmin(r) {
		return true
	}
	var tid string
	err := db.QueryRow("SELECT tenant_id FROM model_locations WHERE id=?", locationID).Scan(&tid)
	if err != nil {
		return false
	}
	return tid == tenantID(r)
}

func tenantID(r *http.Request) string {
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		return ""
	}
	return info.TenantID
}

func isPlatformAdmin(r *http.Request) bool {
	info := auth.SessionInfoFromContext(r.Context())
	return info != nil && info.IsPlatformAdmin
}
