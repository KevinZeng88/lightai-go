// Package log — high-frequency summary / sampling / rate-limit helpers.
package log

import (
	"sync"
	"time"
)

// SummaryConfig holds configurable thresholds for summary logging.
// These are defaults; individual callers can override.
var SummaryConfig = struct {
	HeartbeatInterval     time.Duration // default 60s between heartbeat summaries
	TaskPollInterval      time.Duration // default 60s between task poll summaries
	MetricsInterval       time.Duration // default 60s between metrics summaries
	WaitProgressInterval  time.Duration // default 10s between wait progress logs
	SlowAPIThresholdMs    int64         // default 1000ms
	SlowDBThresholdMs     int64         // default 500ms
	SlowDockerThresholdMs int64         // default 5000ms
	SlowHealthThresholdMs int64         // default 30000ms
	SlowScriptThresholdMs int64         // default 30000ms
}{
	HeartbeatInterval:     60 * time.Second,
	TaskPollInterval:      60 * time.Second,
	MetricsInterval:       60 * time.Second,
	WaitProgressInterval:  10 * time.Second,
	SlowAPIThresholdMs:    1000,
	SlowDBThresholdMs:     500,
	SlowDockerThresholdMs: 5000,
	SlowHealthThresholdMs: 30000,
	SlowScriptThresholdMs: 30000,
}

// RateLimiter is a simple time-based rate limiter for log events.
// It allows at most one event per interval.
type RateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	last     time.Time
}

// NewRateLimiter creates a rate limiter with the given interval.
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{interval: interval}
}

// Allow returns true if enough time has passed since the last Allow() call.
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if now.Sub(r.last) >= r.interval {
		r.last = now
		return true
	}
	return false
}

// PeriodicSummary tracks counts for periodic summary logging.
type PeriodicSummary struct {
	mu           sync.Mutex
	name         string
	interval     time.Duration
	lastSummary  time.Time
	successCount int64
	failureCount int64
	lastValue    int64
	maxValue     int64
	totalValue   int64
	count        int64
}

// NewPeriodicSummary creates a new periodic summary tracker.
func NewPeriodicSummary(name string, interval time.Duration) *PeriodicSummary {
	return &PeriodicSummary{
		name:     name,
		interval: interval,
	}
}

// RecordSuccess records a successful event with a value (e.g., latency_ms).
func (s *PeriodicSummary) RecordSuccess(value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.successCount++
	s.lastValue = value
	if value > s.maxValue {
		s.maxValue = value
	}
	s.totalValue += value
	s.count++
}

// RecordFailure records a failed event.
func (s *PeriodicSummary) RecordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureCount++
	s.count++
}

// ShouldSummarize returns true if it's time to emit a summary. When true,
// the counters are reset and the caller receives the summary data.
func (s *PeriodicSummary) ShouldSummarize() (bool, string, map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if now.Sub(s.lastSummary) < s.interval {
		return false, "", nil
	}

	s.lastSummary = now
	avgValue := int64(0)
	if s.count > 0 {
		avgValue = s.totalValue / s.count
	}

	args := map[string]any{
		"success_count": s.successCount,
		"failure_count": s.failureCount,
		"last_value":    s.lastValue,
		"max_value":     s.maxValue,
		"avg_value":     avgValue,
		"total_count":   s.count,
	}

	// Reset counters.
	s.successCount = 0
	s.failureCount = 0
	s.lastValue = 0
	s.maxValue = 0
	s.totalValue = 0
	s.count = 0

	return true, s.name, args
}

// ChangedTracker tracks whether a value has changed since last check.
// Used for state-change-triggered logging.
type ChangedTracker struct {
	mu        sync.Mutex
	lastValue interface{}
}

// NewChangedTracker creates a new change tracker.
func NewChangedTracker() *ChangedTracker {
	return &ChangedTracker{}
}

// Changed returns true if the value differs from the last seen value.
// Always returns true on first call (initial state is a change).
func (c *ChangedTracker) Changed(newValue interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lastValue == nil {
		c.lastValue = newValue
		return true
	}
	// Simple comparison — works for basic types and strings.
	if c.lastValue != newValue {
		c.lastValue = newValue
		return true
	}
	return false
}

// ChangedInt is a typed helper for int values.
func (c *ChangedTracker) ChangedInt(newValue int) bool {
	return c.Changed(newValue)
}

// ChangedString is a typed helper for string values.
func (c *ChangedTracker) ChangedString(newValue string) bool {
	return c.Changed(newValue)
}

// ChangedBool is a typed helper for bool values.
func (c *ChangedTracker) ChangedBool(newValue bool) bool {
	return c.Changed(newValue)
}
