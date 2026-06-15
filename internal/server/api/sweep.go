package api

import (
	"fmt"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"

	"github.com/google/uuid"
)

// RunSweepLoop periodically sweeps expired tasks and leases.
// It runs independently of heartbeat-triggered sweeps, ensuring cleanup
// even when all agents are offline.
func RunSweepLoop(database *db.DB, interval time.Duration, stop <-chan struct{}) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	log.Info("sweep loop started", "interval", interval.String())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Info("sweep loop stopped")
			return
		case <-ticker.C:
			sweepOnce(database)
		}
	}
}

func sweepOnce(database *db.DB) {
	now := time.Now().UTC().Format(time.RFC3339)

	// 1. Expired pending/claimed tasks → timed_out.
	r1, _ := database.Exec(
		`UPDATE agent_tasks SET status = ?, finished_at = ?, updated_at = ?
		 WHERE status IN (?, ?, ?)
		 AND (julianday(?) - julianday(created_at)) * 86400 > timeout_seconds`,
		TaskStatusTimedOut, now, now,
		TaskStatusPending, TaskStatusClaimed, TaskStatusInProgress,
		now,
	)
	n1, _ := r1.RowsAffected()

	// 2. Instances with timed-out tasks: if still pending/starting → unknown.
	//    Don't force 'failed' — agent may still be loading the model.
	r2, _ := database.Exec(
		`UPDATE model_instances SET actual_state = ?, updated_at = ?
		 WHERE id IN (SELECT instance_id FROM agent_tasks WHERE status = ? AND instance_id != '')
		 AND actual_state IN (?, ?)`,
		InstanceStateUnknown, now, TaskStatusTimedOut,
		InstanceStatePending, InstanceStateStarting,
	)
	n2, _ := r2.RowsAffected()

	// 3. Expired reserved leases → failed (safe: no container was started).
	r3, _ := database.Exec(
		`UPDATE gpu_leases SET status = ?, updated_at = ?
		 WHERE expires_at IS NOT NULL AND expires_at < ? AND status = ?`,
		LeaseFailed, now, LeaseReserved,
	)
	n3, _ := r3.RowsAffected()

	// 4. Active leases past grace period (2x timeout) without agent heartbeat → failed.
	//    Active leases are NOT immediately failed on expiry — the agent may
	//    still be running the container even if a task timed out.
	//    Only mark active leases as failed if the node has been offline.
	r4, _ := database.Exec(
		`UPDATE gpu_leases SET status = ?, updated_at = ?
		 WHERE expires_at IS NOT NULL AND expires_at < ? AND status = ?
		 AND node_id IN (SELECT id FROM nodes WHERE status = 'offline')`,
		LeaseFailed, now, LeaseActive,
	)
	n4, _ := r4.RowsAffected()

	if n1+n2+n3+n4 > 0 {
		log.Info("sweep: cleaned up stale state",
			"timed_out_tasks", n1,
			"unknown_instances", n2,
			"expired_reserved_leases", n3,
			"expired_active_leases_offline", n4,
		)
		// Audit significant state changes.
		if n2 > 0 {
			auditSweep(database, "instance", InstanceStateUnknown, now, "timed_out_task")
		}
		if n3 > 0 {
			auditSweep(database, "lease", LeaseFailed, now, "expired_reserved")
		}
	}
}

// auditSweep writes a sweep-triggered state change to audit_logs.
func auditSweep(database *db.DB, entityType, newState, now, reason string) {
	database.Exec(
		`INSERT INTO audit_logs (id, action, entity_type, entity_id, detail, operator_user_id, created_at)
		 VALUES (?, 'sweep', ?, '', ?, 'system', ?)`,
		uuid.NewString(), entityType, fmt.Sprintf(`{"new_state":"%s","reason":"%s"}`, newState, reason), now,
	)
}
