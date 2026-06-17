// Package log — operation lifecycle logging wrapper for LightAI Go.
//
// This file provides the structured operation logging API. All functions accept
// context.Context and automatically inject request_id / operation_id from it.
//
// Backed by log/slog — no third-party logging frameworks.
package log

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// StartOperation logs an operation_started event and returns the start time.
// The returned context carries operation_id for downstream use.
//
// Usage:
//
//	ctx, start := log.StartOperation(ctx, "deployment.start",
//	    "deployment_id", deployID)
//	defer log.OperationCompleted(ctx, "deployment.start", start,
//	    "deployment_id", deployID)
func StartOperation(ctx context.Context, operation string, fields ...any) (context.Context, time.Time) {
	start := time.Now()
	all := make([]any, 0, len(fields)+4)
	all = append(all, "operation", operation)
	all = append(all, "stage", "operation_started")
	all = append(all, fields...)
	InfoContext(ctx, "operation_started", all...)
	return ctx, start
}

// OperationCompleted logs operation_completed with duration.
// Call with defer after StartOperation.
func OperationCompleted(ctx context.Context, operation string, startedAt time.Time, fields ...any) {
	durationMs := time.Since(startedAt).Milliseconds()
	all := make([]any, 0, len(fields)+6)
	all = append(all, "operation", operation)
	all = append(all, "stage", "operation_completed")
	all = append(all, "duration_ms", durationMs)
	all = append(all, fields...)
	InfoContext(ctx, "operation_completed", all...)
}

// OperationFailed logs operation_failed with error, failed stage, and duration.
func OperationFailed(ctx context.Context, operation string, failedStage string, startedAt time.Time, err error, fields ...any) {
	durationMs := time.Since(startedAt).Milliseconds()
	all := make([]any, 0, len(fields)+10)
	all = append(all, "operation", operation)
	all = append(all, "stage", "operation_failed")
	all = append(all, "error", err.Error())
	all = append(all, "failed_stage", failedStage)
	all = append(all, "duration_ms", durationMs)
	all = append(all, fields...)
	ErrorContext(ctx, "operation_failed", all...)
}

// StageCompleted logs a stage completion with duration.
func StageCompleted(ctx context.Context, operation string, stage string, startedAt time.Time, fields ...any) {
	durationMs := time.Since(startedAt).Milliseconds()
	all := make([]any, 0, len(fields)+6)
	all = append(all, "operation", operation)
	all = append(all, "stage", stage)
	all = append(all, "duration_ms", durationMs)
	all = append(all, fields...)
	InfoContext(ctx, "stage_completed", all...)
}

// StageFailed logs a stage failure.
func StageFailed(ctx context.Context, operation string, stage string, startedAt time.Time, err error, fields ...any) {
	durationMs := time.Since(startedAt).Milliseconds()
	all := make([]any, 0, len(fields)+8)
	all = append(all, "operation", operation)
	all = append(all, "stage", stage+"_failed")
	all = append(all, "error", err.Error())
	all = append(all, "duration_ms", durationMs)
	all = append(all, fields...)
	ErrorContext(ctx, "stage_failed", all...)
}

// SlowOperation logs a slow_operation warning when an operation exceeds its threshold.
func SlowOperation(ctx context.Context, operation string, stage string, durationMs, thresholdMs int64, fields ...any) {
	all := make([]any, 0, len(fields)+8)
	all = append(all, "operation", operation)
	all = append(all, "stage", "slow_operation")
	all = append(all, "slow_stage", stage)
	all = append(all, "duration_ms", durationMs)
	all = append(all, "threshold_ms", thresholdMs)
	all = append(all, fields...)
	WarnContext(ctx, "slow_operation", all...)
}

// OperationTimeout logs an operation_timeout with full context.
func OperationTimeout(ctx context.Context, operation string, waitCondition string, startedAt time.Time, timeoutMs int64, lastState string, lastError string, fields ...any) {
	elapsedMs := time.Since(startedAt).Milliseconds()
	all := make([]any, 0, len(fields)+14)
	all = append(all, "operation", operation)
	all = append(all, "stage", "timeout")
	all = append(all, "wait_condition", waitCondition)
	all = append(all, "timeout_ms", timeoutMs)
	all = append(all, "elapsed_ms", elapsedMs)
	all = append(all, "last_state", lastState)
	if lastError != "" {
		all = append(all, "last_error", lastError)
	}
	all = append(all, fields...)
	ErrorContext(ctx, "operation_timeout", all...)
}

// StateTransition logs a state change.
func StateTransition(ctx context.Context, operation string, entityType string, entityID string, stateFrom, stateTo string, fields ...any) {
	all := make([]any, 0, len(fields)+10)
	all = append(all, "operation", operation)
	all = append(all, "stage", "state_transition")
	all = append(all, "entity_type", entityType)
	all = append(all, "entity_id", entityID)
	all = append(all, "state_from", stateFrom)
	all = append(all, "state_to", stateTo)
	all = append(all, fields...)
	InfoContext(ctx, "state_transition", all...)
}

// WaitStarted logs a wait_started event.
func WaitStarted(ctx context.Context, operation string, waitCondition string, timeoutSec int, fields ...any) {
	all := make([]any, 0, len(fields)+8)
	all = append(all, "operation", operation)
	all = append(all, "stage", "wait_started")
	all = append(all, "wait_condition", waitCondition)
	all = append(all, "timeout_sec", timeoutSec)
	all = append(all, fields...)
	InfoContext(ctx, "wait_started", all...)
}

// WaitProgress logs wait progress (caller should rate-limit).
func WaitProgress(ctx context.Context, operation string, waitCondition string, elapsedMs int64, timeoutMs int64, currentState string, fields ...any) {
	all := make([]any, 0, len(fields)+10)
	all = append(all, "operation", operation)
	all = append(all, "stage", "wait_progress")
	all = append(all, "wait_condition", waitCondition)
	all = append(all, "elapsed_ms", elapsedMs)
	all = append(all, "timeout_ms", timeoutMs)
	all = append(all, "current_state", currentState)
	all = append(all, fields...)
	DebugContext(ctx, "wait_progress", all...)
}

// WaitCompleted logs a wait_completed event.
func WaitCompleted(ctx context.Context, operation string, waitCondition string, startedAt time.Time, fields ...any) {
	elapsedMs := time.Since(startedAt).Milliseconds()
	all := make([]any, 0, len(fields)+8)
	all = append(all, "operation", operation)
	all = append(all, "stage", "wait_completed")
	all = append(all, "wait_condition", waitCondition)
	all = append(all, "elapsed_ms", elapsedMs)
	all = append(all, fields...)
	InfoContext(ctx, "wait_completed", all...)
}

// WaitTimeout logs a wait timeout (convenience over OperationTimeout).
func WaitTimeout(ctx context.Context, operation string, waitCondition string, startedAt time.Time, timeoutMs int64, lastState string, lastError string, fields ...any) {
	OperationTimeout(ctx, operation, waitCondition, startedAt, timeoutMs, lastState, lastError, fields...)
}

// DurationMs computes milliseconds since start.
func DurationMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

// NewRequestID generates a new request_id (UUIDv4).
func NewRequestID() string {
	return newUUID()
}

// NewOperationID generates a new operation_id (UUIDv4).
func NewOperationID() string {
	return newUUID()
}

// OpWarn logs a warning with operation context.
func OpWarn(operation string, stage string, args ...any) {
	all := make([]any, 0, len(args)+4)
	all = append(all, "operation", operation)
	all = append(all, "stage", stage)
	all = append(all, args...)
	Warn("operation_warning", all...)
}

// SafeDockerCommandPreview returns a redacted Docker command string for logging.
func SafeDockerCommandPreview(image string, containerName string, envKeys []string, ports string, volumesCount int, devicesCount int, gpuIDs []string) string {
	return "docker run -d --name " + containerName + " " + image
}

// newUUID generates a random UUIDv4 string without external dependencies.
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp-based pseudo-unique string.
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return hex.EncodeToString(b)
}
