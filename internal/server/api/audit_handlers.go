package api

import (
	"net/http"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
)

// AuditHandler serves audit log queries.
type AuditHandler struct {
	DB *db.DB
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(database *db.DB) *AuditHandler {
	return &AuditHandler{DB: database}
}

// HandleListAuditLogs returns paginated audit log entries.
// GET /api/v1/audit-logs?action=&entity_type=&entity_id=&limit=50&offset=0
func (h *AuditHandler) HandleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	info := auth.SessionInfoFromContext(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	limit := intVal(map[string]interface{}{"limit": q.Get("limit")}, "limit", 50)
	offset := intVal(map[string]interface{}{"offset": q.Get("offset")}, "offset", 0)
	action := q.Get("action")
	entityType := q.Get("entity_type")
	entityID := q.Get("entity_id")

	if limit > 200 {
		limit = 200
	}

	query := `SELECT id, action, entity_type, entity_id, detail, operator_user_id, created_at FROM audit_logs WHERE 1=1`
	var args []interface{}

	// Tenant scope: platform_admin sees all; others see their tenant's logs.
	if !info.IsPlatformAdmin {
		query += ` AND operator_user_id IN (SELECT user_id FROM tenant_memberships WHERE tenant_id = ?)`
		args = append(args, info.TenantID)
	}

	if action != "" {
		query += ` AND action = ?`
		args = append(args, action)
	}
	if entityType != "" {
		query += ` AND entity_type = ?`
		args = append(args, entityType)
	}
	if entityID != "" {
		query += ` AND entity_id = ?`
		args = append(args, entityID)
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Error("audit list query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id, act, etype, eid, detail, opUID, createdAt string
		if err := rows.Scan(&id, &act, &etype, &eid, &detail, &opUID, &createdAt); err != nil {
			continue
		}
		// Redact sensitive content in detail.
		detail = redactDetailString(detail)
		entries = append(entries, map[string]interface{}{
			"id":               id,
			"action":           act,
			"entity_type":      etype,
			"entity_id":        eid,
			"detail":           detail,
			"operator_user_id": opUID,
			"created_at":       createdAt,
		})
	}
	if entries == nil {
		entries = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries": entries,
		"total":   len(entries),
	})
}

