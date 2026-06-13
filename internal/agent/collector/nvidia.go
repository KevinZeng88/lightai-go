package collector

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	nvidiaSmiCmd   = "nvidia-smi"
	nvidiaSmiQuery = "index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw"
)

// NvidiaCollector collects NVIDIA GPU information using nvidia-smi.
type NvidiaCollector struct {
	toolPath string
}

// NewNvidiaCollector creates a new NVIDIA collector.
// It auto-detects the nvidia-smi binary path.
func NewNvidiaCollector() *NvidiaCollector {
	path := nvidiaSmiCmd
	if p, err := exec.LookPath(nvidiaSmiCmd); err == nil {
		path = p
	}
	return &NvidiaCollector{toolPath: path}
}

// Name returns the collector name.
func (n *NvidiaCollector) Name() string {
	return "nvidia"
}

// Vendor returns the GPU vendor name.
func (n *NvidiaCollector) Vendor() string {
	return "nvidia"
}

// Discover executes nvidia-smi --query-gpu to discover GPU devices.
func (n *NvidiaCollector) Discover(ctx context.Context) ([]GPUDeviceInfo, *CollectorDiagnosis) {
	now := time.Now()
	diag := &CollectorDiagnosis{
		Name:      "nvidia",
		Type:      "gpu",
		Vendor:    "nvidia",
		ToolPath:  n.toolPath,
		Available: true,
		CheckedAt: now,
	}

	// Check tool availability.
	if _, err := exec.LookPath(nvidiaSmiCmd); err != nil {
		diag.Available = false
		diag.Error = fmt.Sprintf("nvidia-smi not found: %v", err)
		return nil, diag
	}

	// Execute query.
	args := []string{
		"--query-gpu=" + nvidiaSmiQuery,
		"--format=csv,noheader,nounits",
	}
	cmd := exec.CommandContext(ctx, nvidiaSmiCmd, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		diag.Available = false
		diag.Error = fmt.Sprintf("nvidia-smi failed: %v (stderr: %s)", err, stderr.String())
		return nil, diag
	}

	// Parse output.
	devices, err := parseNvidiaCSV(stdout.String(), now)
	if err != nil {
		diag.Available = false
		diag.Error = fmt.Sprintf("parse nvidia-smi output: %v", err)
		return nil, diag
	}

	return devices, diag
}

// Metrics executes nvidia-smi to collect GPU metrics.
func (n *NvidiaCollector) Metrics(ctx context.Context) ([]GPUMetricInfo, *CollectorDiagnosis) {
	now := time.Now()
	diag := &CollectorDiagnosis{
		Name:      "nvidia",
		Type:      "gpu",
		Vendor:    "nvidia",
		ToolPath:  n.toolPath,
		Available: true,
		CheckedAt: now,
	}

	if _, err := exec.LookPath(nvidiaSmiCmd); err != nil {
		diag.Available = false
		diag.Error = fmt.Sprintf("nvidia-smi not found: %v", err)
		return nil, diag
	}

	args := []string{
		"--query-gpu=" + nvidiaSmiQuery,
		"--format=csv,noheader,nounits",
	}
	cmd := exec.CommandContext(ctx, nvidiaSmiCmd, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		diag.Available = false
		diag.Error = fmt.Sprintf("nvidia-smi failed: %v (stderr: %s)", err, stderr.String())
		return nil, diag
	}

	metrics, err := parseNvidiaMetricsCSV(stdout.String(), now)
	if err != nil {
		diag.Available = false
		diag.Error = fmt.Sprintf("parse nvidia-smi output: %v", err)
		return nil, diag
	}

	return metrics, diag
}

// parseNvidiaCSV parses nvidia-smi CSV output into GPUDeviceInfo.
// Expected format: index,name,uuid,pci.bus_id,driver_version,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw
func parseNvidiaCSV(output string, collectedAt time.Time) ([]GPUDeviceInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return []GPUDeviceInfo{}, nil // Empty list is valid (no GPUs).
	}

	var devices []GPUDeviceInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 8 {
			return nil, fmt.Errorf("expected at least 8 fields, got %d in line: %s", len(fields), line)
		}

		// Trim whitespace from each field.
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		index, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("invalid index '%s': %w", fields[0], err)
		}

		// memory.total is in MB, convert to bytes.
		memTotalMB, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid memory.total '%s': %w", fields[5], err)
		}
		memTotalBytes := memTotalMB * 1024 * 1024

		devices = append(devices, GPUDeviceInfo{
			Vendor:           "nvidia",
			Index:            index,
			Name:             fields[1],
			UUID:             fields[2],
			PCIBusID:         fields[3],
			DriverVersion:    fields[4],
			MemoryTotalBytes: memTotalBytes,
			Status:           "available",
			CollectedAt:      collectedAt,
		})
	}

	return devices, nil
}

// parseNvidiaMetricsCSV parses nvidia-smi CSV output into GPUMetricInfo.
func parseNvidiaMetricsCSV(output string, collectedAt time.Time) ([]GPUMetricInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return []GPUMetricInfo{}, nil
	}

	var metrics []GPUMetricInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 8 {
			return nil, fmt.Errorf("expected at least 8 fields, got %d in line: %s", len(fields), line)
		}

		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		index, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("invalid index '%s': %w", fields[0], err)
		}

		// memory.used and memory.free are in MB, convert to bytes.
		memUsedMB, err := parseUintOrZero(fields[6])
		if err != nil {
			return nil, fmt.Errorf("invalid memory.used '%s': %w", fields[6], err)
		}
		memFreeMB, err := parseUintOrZero(fields[7])
		if err != nil {
			return nil, fmt.Errorf("invalid memory.free '%s': %w", fields[7], err)
		}
		memUsedBytes := memUsedMB * 1024 * 1024
		memFreeBytes := memFreeMB * 1024 * 1024

		metric := GPUMetricInfo{
			Vendor:          "nvidia",
			Index:           index,
			UUID:            fields[2],
			MemoryUsedBytes: memUsedBytes,
			MemoryFreeBytes: memFreeBytes,
			Health:          "healthy",
			CollectedAt:     collectedAt,
		}

		// utilization.gpu (field 8) - 0-100 percent.
		if len(fields) > 8 {
			if v, err := parseFloatOrNil(fields[8]); err == nil && v != nil {
				metric.GPUUtilization = v
			}
		}

		// utilization.memory (field 9) - 0-100 percent.
		if len(fields) > 9 {
			if v, err := parseFloatOrNil(fields[9]); err == nil && v != nil {
				metric.MemoryUtilization = v
			}
		}

		// temperature.gpu (field 10) - celsius.
		if len(fields) > 10 {
			if v, err := parseFloatOrNil(fields[10]); err == nil && v != nil {
				metric.Temperature = v
			}
		}

		// power.draw (field 11) - watts. May be "N/A" or empty.
		if len(fields) > 11 {
			if v, err := parseFloatOrNil(fields[11]); err == nil && v != nil {
				metric.PowerDraw = v
			}
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// parseUintOrZero parses a string to uint64, returning 0 for empty or invalid input.
func parseUintOrZero(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" || s == "[N/A]" {
		return 0, nil
	}
	return strconv.ParseUint(s, 10, 64)
}

// parseFloatOrNil parses a string to float64, returning nil for empty, N/A, or invalid input.
func parseFloatOrNil(s string) (*float64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" || s == "[N/A]" || s == "Unknown" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, nil // Treat parse errors as nil (unknown).
	}
	return &v, nil
}

// Ensure interface satisfaction.
var _ GPUCollector = (*NvidiaCollector)(nil)
