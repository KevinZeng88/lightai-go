package api

// Task status constants.
const (
	TaskStatusPending    = "pending"
	TaskStatusClaimed    = "claimed"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
	TaskStatusTimedOut   = "timed_out"
)

// Instance state constants.
const (
	InstanceStatePending      = "pending"
	InstanceStateStarting     = "starting"
	InstanceStateRunning      = "running"
	InstanceStateStopped      = "stopped"
	InstanceStateFailed       = "failed"
	InstanceStateInitializing = "initializing"
)

// GpuLease status and state constants.
const (
	LeaseReserved = "reserved"
	LeaseActive   = "active"
	LeaseReleased = "released"
	LeaseFailed   = "failed"
	LeaseExpired  = "expired"
)
