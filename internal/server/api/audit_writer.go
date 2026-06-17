package api

import (
	"context"
	"database/sql"
	"time"

	"lightai-go/internal/common/log"

	"github.com/google/uuid"
)

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	TenantID     string `json:"tenant_id"`
	ActorID      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Result       string `json:"result"`            // "success", "failure"
	Detail       string `json:"detail,omitempty"`  // JSON string with extra context
	RequestID    string `json:"request_id,omitempty"`
	OperationID  string `json:"operation_id,omitempty"`
	Error        string `json:"error,omitempty"`
}

// WriteAudit inserts an audit log entry into the database.
// It never fails the caller — errors are logged but not returned.
// It also logs at INFO so structured logs capture what the audit table missed.
func WriteAudit(ctx context.Context, db *sql.DB, entry AuditEntry) {
	if db == nil {
		return
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)

	// Build detail string with all correlation context, result, and error.
	fullDetail := entry.Detail
	if entry.OperationID != "" {
		if fullDetail != "" {
			fullDetail += " "
		}
		fullDetail += "operation_id=" + entry.OperationID
	}
	if entry.RequestID != "" {
		if fullDetail != "" {
			fullDetail += " "
		}
		fullDetail += "request_id=" + entry.RequestID
	}
	if entry.Result != "" {
		if fullDetail != "" {
			fullDetail += " "
		}
		fullDetail += "result=" + entry.Result
	}
	if entry.Error != "" {
		if fullDetail != "" {
			fullDetail += " "
		}
		fullDetail += "error=" + entry.Error
	}

	_, err := db.Exec(
		`INSERT INTO audit_logs (id, tenant_id, action, entity_type, entity_id, detail, operator_user_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		safeStr(entry.TenantID),
		safeStr(entry.Action),
		safeStr(entry.ResourceType),
		safeStr(entry.ResourceID),
		fullDetail,
		safeStr(entry.ActorID),
		now,
	)
	if err != nil {
		log.Error("audit.write.failed",
			"action", entry.Action,
			"resource_type", entry.ResourceType,
			"resource_id", entry.ResourceID,
			"error", err,
		)
		return
	}

	log.Info("audit.recorded",
		"action", entry.Action,
		"resource_type", entry.ResourceType,
		"resource_id", entry.ResourceID,
		"result", entry.Result,
		"tenant_id", entry.TenantID,
		"actor_id", entry.ActorID,
		"operation_id", entry.OperationID,
		"request_id", entry.RequestID,
	)
}

// safeStr returns s if non-empty, otherwise returns "unknown".
func safeStr(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
