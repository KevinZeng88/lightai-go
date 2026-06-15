package collector

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"lightai-go/internal/common/log"
)

// ProbeResult is the outcome of probing a GPU vendor collector.
type ProbeResult struct {
	Vendor      string
	Available   bool   // true if DISCOVER.sh exited 0 and returned DEVICE lines
	DeviceCount int    // number of DEVICE lines parsed
	Error       string // human-readable error or not_available reason
	ExitCode    int
}

// ProbeDef defines how to probe a vendor.
type ProbeDef struct {
	Name        string
	Vendor      string
	DiscoverCmd string
	MetricsCmd  string
	Timeout     time.Duration
}

// DefaultProbes returns the built-in list of vendor probes.
func DefaultProbes() []ProbeDef {
	return []ProbeDef{
		{
			Name:        "nvidia",
			Vendor:      "nvidia",
			DiscoverCmd: "deploy/collectors/gpu/nvidia/discover.sh",
			MetricsCmd:  "deploy/collectors/gpu/nvidia/metrics.sh",
		},
		{
			Name:        "metax",
			Vendor:      "metax",
			DiscoverCmd: "deploy/collectors/gpu/metax/discover.sh",
			MetricsCmd:  "deploy/collectors/gpu/metax/metrics.sh",
		},
	}
}

// Probe runs a discover command and checks whether the vendor has GPUs.
// Exit 0 + DEVICE lines → available
// Exit 0 + no DEVICE → not available (record diagnostic)
// Exit 10 → not_available (no error)
// Exit >=30 → probe failed (warn)
func Probe(ctx context.Context, def ProbeDef) ProbeResult {
	start := time.Now()
	result := ProbeResult{Vendor: def.Vendor}

	cmd := exec.CommandContext(ctx, "sh", "-c", def.DiscoverCmd)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout := outBuf.String()
	stderr := errBuf.String()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	result.ExitCode = exitCode
	duration := time.Since(start)

	switch {
	case exitCode == 0:
		// Parse protocol to count DEVICE lines.
		devices, _, _, parseErr := ParseProtocolOutput(stdout, start)
		if parseErr != nil {
			result.Available = false
			result.Error = fmt.Sprintf("parse error: %v", parseErr)
			log.Warn("auto-detect probe parse failed",
				"vendor", def.Vendor,
				"error", parseErr,
				"duration_ms", duration.Milliseconds(),
			)
			return result
		}
		if len(devices) > 0 {
			result.Available = true
			result.DeviceCount = len(devices)
			log.Info("auto-detect probe found GPUs",
				"vendor", def.Vendor,
				"device_count", len(devices),
				"duration_ms", duration.Milliseconds(),
			)
		} else {
			result.Available = false
			result.Error = "no DEVICE output from discover"
			log.Debug("auto-detect probe: no GPUs",
				"vendor", def.Vendor,
				"duration_ms", duration.Milliseconds(),
			)
		}

	case exitCode == 10:
		result.Available = false
		result.Error = fmt.Sprintf("not_available: %s", trimStr(stderr, 200))
		log.Info("auto-detect probe: vendor not available",
			"vendor", def.Vendor,
			"reason", result.Error,
			"duration_ms", duration.Milliseconds(),
		)

	default:
		result.Available = false
		result.Error = fmt.Sprintf("probe failed (exit=%d): %s", exitCode, trimStr(stderr, 200))
		log.Warn("auto-detect probe failed",
			"vendor", def.Vendor,
			"exit_code", exitCode,
			"stderr", trimStr(stderr, 200),
			"duration_ms", duration.Milliseconds(),
		)
	}

	return result
}

func trimStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
