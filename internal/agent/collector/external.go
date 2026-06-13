package collector

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"lightai-go/internal/common/log"
)

// ExternalCommandConfig configures an external GPU collector script.
type ExternalCommandConfig struct {
	Name        string
	Vendor      string
	Enabled     bool
	DiscoverCmd string
	MetricsCmd  string
	Timeout     time.Duration
	WorkDir     string
	Env         []string
}

// ExternalCommandCollector runs external scripts that output
// LightAI GPU Collector Protocol.
type ExternalCommandCollector struct {
	cfg     ExternalCommandConfig
	cmdPath string // resolved discover command path
}

// NewExternalCommandCollector creates a new external command collector.
func NewExternalCommandCollector(cfg ExternalCommandConfig) *ExternalCommandCollector {
	return &ExternalCommandCollector{cfg: cfg}
}

// Name returns the collector name.
func (e *ExternalCommandCollector) Name() string {
	return e.cfg.Name
}

// Vendor returns the GPU vendor.
func (e *ExternalCommandCollector) Vendor() string {
	return e.cfg.Vendor
}

// Discover runs the discover command and parses protocol output.
func (e *ExternalCommandCollector) Discover(ctx context.Context) ([]GPUDeviceInfo, *CollectorDiagnosis) {
	start := time.Now()
	diag := &CollectorDiagnosis{
		Name:      e.cfg.Name,
		Type:      "gpu",
		Vendor:    e.cfg.Vendor,
		Available: true,
		CheckedAt: start,
	}

	log.Debug("external collector discover start",
		"collector", e.cfg.Name,
		"vendor", e.cfg.Vendor,
		"mode", "external",
		"discover_cmd", e.cfg.DiscoverCmd,
	)

	stdout, stderr, exitCode, err := e.runCmd(ctx, e.cfg.DiscoverCmd)
	duration := time.Since(start)

	if err != nil {
		log.Error("external collector discover failed",
			"collector", e.cfg.Name,
			"vendor", e.cfg.Vendor,
			"exit_code", exitCode,
			"duration_ms", duration.Milliseconds(),
			"stderr", truncateStr(stderr, 200),
			"error", err,
		)
		return e.handleError(exitCode, stderr, err, diag)
	}

	log.Debug("external collector discover done",
		"collector", e.cfg.Name,
		"exit_code", exitCode,
		"duration_ms", duration.Milliseconds(),
		"stdout_bytes", len(stdout),
	)

	devices, metrics, protoDiag, parseErr := ParseProtocolOutput(stdout, start)
	if parseErr != nil {
		log.Error("external collector discover parse failed",
			"collector", e.cfg.Name,
			"error", parseErr,
			"stdout", truncateStr(stdout, 500),
		)
		diag.Available = false
		diag.Error = fmt.Sprintf("parse error: %v", parseErr)
		return nil, diag
	}

	diag.Name = protoDiag.Name
	diag.Vendor = protoDiag.Vendor
	diag.Available = protoDiag.Available
	diag.Error = protoDiag.Error

	_ = metrics // Metrics are collected separately via Metrics().

	log.Info("external collector discover success",
		"collector", e.cfg.Name,
		"vendor", e.cfg.Vendor,
		"parsed_device_count", len(devices),
		"duration_ms", duration.Milliseconds(),
	)

	return devices, diag
}

// Metrics runs the metrics command and parses protocol output.
func (e *ExternalCommandCollector) Metrics(ctx context.Context) ([]GPUMetricInfo, *CollectorDiagnosis) {
	start := time.Now()
	diag := &CollectorDiagnosis{
		Name:      e.cfg.Name,
		Type:      "gpu",
		Vendor:    e.cfg.Vendor,
		Available: true,
		CheckedAt: start,
	}

	stdout, stderr, exitCode, err := e.runCmd(ctx, e.cfg.MetricsCmd)
	duration := time.Since(start)

	if err != nil {
		log.Error("external collector metrics failed",
			"collector", e.cfg.Name,
			"exit_code", exitCode,
			"duration_ms", duration.Milliseconds(),
			"stderr", truncateStr(stderr, 200),
		)
		return e.handleMetricError(exitCode, stderr, err, diag)
	}

	_, metrics, protoDiag, parseErr := ParseProtocolOutput(stdout, start)
	if parseErr != nil {
		log.Error("external collector metrics parse failed",
			"collector", e.cfg.Name,
			"error", parseErr,
		)
		diag.Available = false
		diag.Error = fmt.Sprintf("parse error: %v", parseErr)
		return nil, diag
	}

	diag.Available = protoDiag.Available
	diag.Error = protoDiag.Error

	log.Info("external collector metrics success",
		"collector", e.cfg.Name,
		"vendor", e.cfg.Vendor,
		"parsed_metric_count", len(metrics),
		"duration_ms", duration.Milliseconds(),
	)

	return metrics, diag
}

func (e *ExternalCommandCollector) runCmd(ctx context.Context, cmdStr string) (stdout, stderr string, exitCode int, err error) {
	if e.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.cfg.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	if e.cfg.WorkDir != "" {
		cmd.Dir = e.cfg.WorkDir
	}
	if len(e.cfg.Env) > 0 {
		cmd.Env = e.cfg.Env
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		return stdout, stderr, exitCode, err
	}

	return stdout, stderr, 0, nil
}

func (e *ExternalCommandCollector) handleError(exitCode int, stderr string, err error, diag *CollectorDiagnosis) ([]GPUDeviceInfo, *CollectorDiagnosis) {
	switch {
	case exitCode == 10:
		diag.Available = false
		diag.Error = fmt.Sprintf("not_available: %s", truncateStr(stderr, 200))
		log.Info("collector not_available",
			"collector", e.cfg.Name,
			"reason", diag.Error,
		)
	case exitCode == 20:
		diag.Error = fmt.Sprintf("partial_success: %s", truncateStr(stderr, 200))
		log.Warn("collector partial_success",
			"collector", e.cfg.Name,
			"stderr", truncateStr(stderr, 200),
		)
	default:
		diag.Available = false
		diag.Error = fmt.Sprintf("command_failed (exit=%d): %s", exitCode, truncateStr(stderr, 200))
	}
	return nil, diag
}

func (e *ExternalCommandCollector) handleMetricError(exitCode int, stderr string, err error, diag *CollectorDiagnosis) ([]GPUMetricInfo, *CollectorDiagnosis) {
	switch {
	case exitCode == 10:
		diag.Available = false
		diag.Error = fmt.Sprintf("not_available: %s", truncateStr(stderr, 200))
	case exitCode == 20:
		diag.Error = fmt.Sprintf("partial_success: %s", truncateStr(stderr, 200))
	default:
		diag.Available = false
		diag.Error = fmt.Sprintf("command_failed (exit=%d): %s", exitCode, truncateStr(stderr, 200))
	}
	return nil, diag
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Ensure interface satisfaction.
var _ GPUCollector = (*ExternalCommandCollector)(nil)
