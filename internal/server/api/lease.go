package api

import (
	"fmt"
	"time"

	"lightai-go/internal/server/db"

	"github.com/google/uuid"
)

// LeaseStatus values.
const (
	LeaseReserved = "reserved"
	LeaseActive   = "active"
	LeaseReleased = "released"
	LeaseFailed   = "failed"
	LeaseExpired  = "expired"
)

// CreateLeases creates reserved GPU leases for the given GPU IDs within a
// transaction. It checks for conflicting active/reserved leases first.
// Returns the created lease IDs.
func CreateLeases(database *db.DB, gpuIDs []string, nodeID, deploymentID, instanceID, tenantID string) ([]string, error) {
	tx, err := database.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var leaseIDs []string
	now := time.Now().UTC().Format(time.RFC3339)

	for _, gpuID := range gpuIDs {
		// Check for existing active or reserved lease on this GPU.
		var existingID string
		err := tx.QueryRow(
			`SELECT id FROM gpu_leases WHERE gpu_id = ? AND status IN (?, ?) LIMIT 1`,
			gpuID, LeaseReserved, LeaseActive,
		).Scan(&existingID)
		if err == nil {
			return nil, fmt.Errorf("GPU %s is already reserved/active (lease %s)", gpuID, existingID)
		}

		leaseID := uuid.NewString()
		expiresAt := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)
		_, err = tx.Exec(
			`INSERT INTO gpu_leases (id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status, reserved_at, expires_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			leaseID, gpuID, nodeID, deploymentID, instanceID, tenantID, LeaseReserved, now, expiresAt, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("insert lease for gpu %s: %w", gpuID, err)
		}
		leaseIDs = append(leaseIDs, leaseID)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit leases: %w", err)
	}
	return leaseIDs, nil
}

// ActivateLeases transitions leases from reserved to active.
// Returns error if any lease is not in reserved state.
func ActivateLeases(database *db.DB, leaseIDs []string) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, leaseID := range leaseIDs {
		result, err := tx.Exec(
			`UPDATE gpu_leases SET status = ?, activated_at = ?, updated_at = ? WHERE id = ? AND status = ?`,
			LeaseActive, now, now, leaseID, LeaseReserved,
		)
		if err != nil {
			return fmt.Errorf("activate lease %s: %w", leaseID, err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return fmt.Errorf("lease %s is not in reserved state, cannot activate", leaseID)
		}
	}
	return tx.Commit()
}

// ReleaseLeases transitions leases to released.
// Idempotent: already-released leases are silently skipped.
func ReleaseLeases(database *db.DB, leaseIDs []string) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, leaseID := range leaseIDs {
		_, err := tx.Exec(
			`UPDATE gpu_leases SET status = ?, released_at = ?, updated_at = ?
			 WHERE id = ? AND status IN (?, ?)`,
			LeaseReleased, now, now, leaseID, LeaseReserved, LeaseActive,
		)
		if err != nil {
			return fmt.Errorf("release lease %s: %w", leaseID, err)
		}
	}
	return tx.Commit()
}

// FailLeases transitions leases to failed.
func FailLeases(database *db.DB, leaseIDs []string) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, leaseID := range leaseIDs {
		_, err := tx.Exec(
			`UPDATE gpu_leases SET status = ?, updated_at = ? WHERE id = ?`,
			LeaseFailed, now, leaseID,
		)
		if err != nil {
			return fmt.Errorf("fail lease %s: %w", leaseID, err)
		}
	}
	return tx.Commit()
}
