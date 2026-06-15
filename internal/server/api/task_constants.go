package api

// Task status values.
const (
	TaskStatusPending    = "pending"
	TaskStatusClaimed    = "claimed"
	TaskStatusInProgress = "in_progress"
	TaskStatusSucceeded  = "succeeded"
	TaskStatusFailed     = "failed"
	TaskStatusTimedOut   = "timed_out"
	TaskStatusCancelled  = "cancelled"
)

// IsTaskTerminal returns true if the task status is a final state.
func IsTaskTerminal(status string) bool {
	return status == TaskStatusSucceeded || status == TaskStatusFailed || status == TaskStatusTimedOut || status == TaskStatusCancelled
}
