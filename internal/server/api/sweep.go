package api

import (
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"
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

	// Expired tasks → timed_out.
	r1, _ := database.Exec(
		`UPDATE agent_tasks SET status = ?, finished_at = ?, updated_at = ?
		 WHERE status IN (?, ?, ?)
		 AND (julianday(?) - julianday(created_at)) * 86400 > timeout_seconds`,
		TaskStatusTimedOut, now, now,
		TaskStatusPending, TaskStatusClaimed, TaskStatusInProgress,
		now,
	)
	n1, _ := r1.RowsAffected()

	// Instances for timed-out tasks → failed.
	r2, _ := database.Exec(
		`UPDATE model_instances SET actual_state = ?, updated_at = ?
		 WHERE id IN (SELECT instance_id FROM agent_tasks WHERE status = ? AND instance_id != '')`,
		InstanceStateFailed, now, TaskStatusTimedOut,
	)
	n2, _ := r2.RowsAffected()

	// Expired reserved/active leases → failed.
	r3, _ := database.Exec(
		`UPDATE gpu_leases SET status = ?, updated_at = ?
		 WHERE expires_at IS NOT NULL AND expires_at < ? AND status IN (?, ?)`,
		LeaseFailed, now, LeaseReserved, LeaseActive,
	)
	n3, _ := r3.RowsAffected()

	// Deployments for failed instances → failed.
	r4, _ := database.Exec(
		`UPDATE model_deployments SET status = 'failed', updated_at = ?
		 WHERE id IN (SELECT deployment_id FROM model_instances WHERE actual_state = ?)`,
		now, InstanceStateFailed,
	)
	n4, _ := r4.RowsAffected()

	if n1+n2+n3+n4 > 0 {
		log.Info("sweep: cleaned up stale state",
			"timed_out_tasks", n1,
			"failed_instances", n2,
			"expired_leases", n3,
			"failed_deployments", n4,
		)
	}
}
