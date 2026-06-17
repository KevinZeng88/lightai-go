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

	// Build WHERE clause for both COUNT and SELECT queries.
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	selectQuery := `SELECT id, action, entity_type, entity_id, detail, operator_user_id, created_at FROM audit_logs WHERE 1=1`
	var countArgs []interface{}

	// Tenant scope: platform_admin sees all; others see their tenant's logs.
	if !info.IsPlatformAdmin {
		countQuery += ` AND operator_user_id IN (SELECT user_id FROM tenant_memberships WHERE tenant_id = ?)`
		selectQuery += ` AND operator_user_id IN (SELECT user_id FROM tenant_memberships WHERE tenant_id = ?)`
		countArgs = append(countArgs, info.TenantID)
	}

	if action != "" {
		countQuery += ` AND action = ?`
		selectQuery += ` AND action = ?`
		countArgs = append(countArgs, action)
	}
	if entityType != "" {
		countQuery += ` AND entity_type = ?`
		selectQuery += ` AND entity_type = ?`
		countArgs = append(countArgs, entityType)
	}
	if entityID != "" {
		countQuery += ` AND entity_id = ?`
		selectQuery += ` AND entity_id = ?`
		countArgs = append(countArgs, entityID)
	}

	// AUD-010: Run separate COUNT query for correct total (not len(page)).
	var total int
	if err := h.DB.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		log.Error("audit count query failed", "error", err)
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	selectQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	selectArgs := append(countArgs, limit, offset)

	rows, err := h.DB.Query(selectQuery, selectArgs...)
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
		"total":   total,
	})
}
