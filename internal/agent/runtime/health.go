// Package runtime — endpoint health check for container readiness verification.
package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lightai-go/internal/common/log"
)

// ContainerInspectFunc inspects a container and returns its state and exit code.
// Returned state is Docker state: "running", "exited", "dead", etc.
type ContainerInspectFunc func(ctx context.Context) (state string, exitCode int, err error)

// resolveDefaults fills in default values for missing health check config fields.
func (c *HealthCheckConfig) resolveDefaults(hostPort int) {
	if !c.Enabled {
		return
	}
	if c.Scheme == "" {
		c.Scheme = "http"
	}
	if c.Path == "" {
		c.Path = "/v1/models"
	}
	if c.Port == 0 {
		if hostPort > 0 {
			c.Port = hostPort
		} else {
			c.Port = 8080 // fallback to common default
		}
	}
	if c.ExpectedStatus == 0 {
		c.ExpectedStatus = 200
	}
	if c.TimeoutSeconds == 0 {
		c.TimeoutSeconds = 30
	}
	if c.IntervalSeconds == 0 {
		c.IntervalSeconds = 2
	}
}

// endpointURL builds the health check URL from config.
func (c *HealthCheckConfig) endpointURL() string {
	return fmt.Sprintf("%s://127.0.0.1:%d%s", c.Scheme, c.Port, c.Path)
}

// CheckEndpointReady polls the health endpoint until it returns the expected
// status or the timeout is reached. Returns nil on success, or an error
// describing the failure.
//
// If inspect is non-nil, it is called on each connection-refused error to
// check whether the container has exited. If the container is no longer
// running, the health check aborts immediately instead of waiting for timeout.
func CheckEndpointReady(ctx context.Context, cfg *HealthCheckConfig, instanceID, containerID, containerName string, inspect ContainerInspectFunc) error {
	if cfg == nil || !cfg.Enabled {
		log.InfoContext(ctx, "health_check.skipped",
			"reason", "no_health_config",
			"instance_id", instanceID,
			"container_id", containerID,
		)
		return nil
	}

	url := cfg.endpointURL()
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	progressInterval := 10 * time.Second

	log.InfoContext(ctx, "health_check.started",
		"instance_id", instanceID,
		"container_id", containerID,
		"container_name", containerName,
		"endpoint_url", url,
		"health_check_port", cfg.Port,
		"health_check_path", cfg.Path,
		"expected_status", cfg.ExpectedStatus,
		"timeout_sec", cfg.TimeoutSeconds,
	)
	log.WaitStarted(ctx, "health_check", "endpoint_ready", cfg.TimeoutSeconds,
		"instance_id", instanceID,
		"container_id", containerID,
		"endpoint_url", url,
	)

	startTime := time.Now()
	deadline := startTime.Add(timeout)
	lastProgress := time.Time{}
	attempt := 0
	consecutiveRefused := 0
	var lastStatus int
	var lastBody string
	var lastError string
	httpClient := &http.Client{Timeout: 5 * time.Second}

	for {
		attempt++
		elapsed := time.Since(startTime)

		if time.Now().After(deadline) {
			// Final re-inspect before reporting timeout.
			if inspect != nil {
				if state, exitCode, inspErr := inspect(ctx); inspErr == nil && state != "running" {
					log.ErrorContext(ctx, "health_check.container_exited_on_timeout",
						"instance_id", instanceID,
						"container_id", containerID,
						"container_state", state,
						"exit_code", exitCode,
						"elapsed_ms", elapsed.Milliseconds(),
						"attempts", attempt,
						"last_http_status", lastStatus,
						"last_error", lastError,
					)
					return fmt.Errorf("health check aborted: container %s (exit_code=%d) during endpoint wait after %d attempts", state, exitCode, attempt)
				}
			}
			log.WaitTimeout(ctx, "health_check", "endpoint_ready", startTime, timeout.Milliseconds(),
				fmt.Sprintf("attempt=%d last_http_status=%d", attempt, lastStatus), lastError,
				"instance_id", instanceID,
				"container_id", containerID,
				"endpoint_url", url,
			)
			log.ErrorContext(ctx, "health_check.timeout",
				"instance_id", instanceID,
				"container_id", containerID,
				"endpoint_url", url,
				"attempts", attempt,
				"elapsed_ms", elapsed.Milliseconds(),
				"last_http_status", lastStatus,
				"last_body", truncateBody(lastBody, 500),
				"last_error", lastError,
			)
			return fmt.Errorf("health check timeout after %d attempts: last status=%d, error=%s", attempt, lastStatus, lastError)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastError = err.Error()
		} else {
			resp, err := httpClient.Do(req)
			if err != nil {
				lastError = err.Error()
				lastStatus = 0
				consecutiveRefused++

				// Re-inspect container on connection refused: if exited, abort immediately.
				if inspect != nil && consecutiveRefused >= 2 {
					if state, exitCode, inspErr := inspect(ctx); inspErr == nil && state != "running" {
						log.ErrorContext(ctx, "health_check.container_exited",
							"instance_id", instanceID,
							"container_id", containerID,
							"container_state", state,
							"exit_code", exitCode,
							"elapsed_ms", elapsed.Milliseconds(),
							"attempts", attempt,
							"last_error", lastError,
						)
						return fmt.Errorf("health check aborted: container %s (exit_code=%d) after %d attempts", state, exitCode, attempt)
					}
					consecutiveRefused = 0 // reset after inspect
				}
			} else {
				consecutiveRefused = 0
				lastStatus = resp.StatusCode
				bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
				resp.Body.Close()
				lastBody = string(bodyBytes)
				lastError = ""

				if resp.StatusCode == cfg.ExpectedStatus {
					log.WaitCompleted(ctx, "health_check", "endpoint_ready", startTime,
						"instance_id", instanceID,
						"container_id", containerID,
						"endpoint_url", url,
						"attempts", attempt,
						"http_status", resp.StatusCode,
					)
					log.InfoContext(ctx, "runtime.health_check.succeeded",
						"instance_id", instanceID,
						"container_id", containerID,
						"endpoint_url", url,
						"attempts", attempt,
						"elapsed_ms", elapsed.Milliseconds(),
						"http_status", resp.StatusCode,
					)
					return nil
				}
				lastError = fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, cfg.ExpectedStatus)
			}
		}

		// Rate-limited progress logging.
		if time.Since(lastProgress) >= progressInterval {
			log.WaitProgress(ctx, "health_check", "endpoint_ready", elapsed.Milliseconds(), timeout.Milliseconds(),
				fmt.Sprintf("attempt=%d last_status=%d", attempt, lastStatus),
				"instance_id", instanceID,
				"container_id", containerID,
				"last_error", lastError,
			)
			lastProgress = time.Now()
		}

		select {
		case <-ctx.Done():
			log.ErrorContext(ctx, "health_check.cancelled",
				"instance_id", instanceID,
				"container_id", containerID,
				"attempts", attempt,
				"elapsed_ms", elapsed.Milliseconds(),
				"last_status", lastStatus,
				"error", ctx.Err(),
			)
			return fmt.Errorf("health check cancelled: %w", ctx.Err())
		case <-time.After(interval):
		}
	}
}

// truncateBody truncates a string to maxLen characters, appending "...truncated" if needed.
func truncateBody(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...truncated"
}

// resolveHealthCheckConfig resolves effective health check config with defaults.
func resolveHealthCheckConfig(cfg *HealthCheckConfig, hostPort int) *HealthCheckConfig {
	if cfg == nil {
		return nil
	}
	copy := *cfg
	copy.resolveDefaults(hostPort)
	return &copy
}
