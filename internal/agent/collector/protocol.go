// Package collector - LightAI GPU Collector Protocol parser.
//
// All vendor GPU collector scripts must output this protocol.
// Go Agent only parses this protocol, never vendor-specific formats.
package collector

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Protocol line types.
const (
	lineStatus = "STATUS"
	lineDevice = "DEVICE"
	lineMetric = "METRIC"
)

// ParseProtocolOutput parses LightAI GPU Collector Protocol output into
// GPUDeviceInfo and GPUMetricInfo slices.
func ParseProtocolOutput(output string, collectedAt time.Time) ([]GPUDeviceInfo, []GPUMetricInfo, *CollectorDiagnosis, error) {
	diag := &CollectorDiagnosis{
		Type:      "gpu",
		CheckedAt: collectedAt,
	}

	var devices []GPUDeviceInfo
	var metrics []GPUMetricInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			return nil, nil, diag, fmt.Errorf("line %d: invalid format, expected TYPE key=value...", lineNo)
		}

		lineType := parts[0]
		kvString := parts[1]

		kv, err := parseKeyValues(kvString)
		if err != nil {
			return nil, nil, diag, fmt.Errorf("line %d: %w", lineNo, err)
		}

		switch lineType {
		case lineStatus:
			diag.Name = kv["name"]
			diag.Vendor = kv["vendor"]
			diag.Available = kv["ok"] == "true"
			if msg, ok := kv["message"]; ok {
				if !diag.Available {
					diag.Error = msg
				}
			}
		case lineDevice:
			dev, err := parseDeviceLine(kv, collectedAt)
			if err != nil {
				return nil, nil, diag, fmt.Errorf("line %d: %w", lineNo, err)
			}
			devices = append(devices, *dev)
		case lineMetric:
			m, err := parseMetricLine(kv, collectedAt)
			if err != nil {
				return nil, nil, diag, fmt.Errorf("line %d: %w", lineNo, err)
			}
			metrics = append(metrics, *m)
		default:
			return nil, nil, diag, fmt.Errorf("line %d: unknown type %q", lineNo, lineType)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, diag, fmt.Errorf("read output: %w", err)
	}

	return devices, metrics, diag, nil
}

// parseKeyValues parses "key1=value1 key2=value2 key3=\"value with spaces\""
func parseKeyValues(s string) (map[string]string, error) {
	result := make(map[string]string)
	i := 0
	runes := []rune(s)
	n := len(runes)

	for i < n {
		// Skip whitespace.
		for i < n && (runes[i] == ' ' || runes[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}

		// Read key.
		keyStart := i
		for i < n && runes[i] != '=' {
			i++
		}
		if i >= n || runes[i] != '=' {
			return nil, fmt.Errorf("expected '=' after key at position %d", i)
		}
		key := string(runes[keyStart:i])
		i++ // skip '='

		// Read value.
		if i >= n {
			return nil, fmt.Errorf("missing value for key %q", key)
		}

		var value string
		if runes[i] == '"' {
			// Quoted value.
			i++ // skip opening quote
			valStart := i
			for i < n && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < n {
					i += 2 // skip escaped char
					continue
				}
				i++
			}
			if i >= n {
				return nil, fmt.Errorf("unterminated quoted value for key %q", key)
			}
			value = string(runes[valStart:i])
			i++ // skip closing quote
		} else {
			// Unquoted value.
			valStart := i
			for i < n && runes[i] != ' ' && runes[i] != '\t' {
				i++
			}
			value = string(runes[valStart:i])
		}

		result[key] = value
	}

	return result, nil
}

func parseDeviceLine(kv map[string]string, collectedAt time.Time) (*GPUDeviceInfo, error) {
	required := []string{"vendor", "index", "name", "uuid"}
	for _, k := range required {
		if _, ok := kv[k]; !ok {
			return nil, fmt.Errorf("DEVICE missing required field %q", k)
		}
	}

	index, err := strconv.Atoi(kv["index"])
	if err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}

	memTotal := uint64(0)
	if v, ok := kv["memory_total_bytes"]; ok {
		memTotal, _ = strconv.ParseUint(v, 10, 64)
	}

	status := "available"
	if v, ok := kv["status"]; ok {
		status = v
	}

	return &GPUDeviceInfo{
		Vendor:           kv["vendor"],
		Index:            index,
		Name:             kv["name"],
		UUID:             kv["uuid"],
		PCIBusID:         kv["pci_bus_id"],
		DriverVersion:    kv["driver_version"],
		MemoryTotalBytes: memTotal,
		Status:           status,
		CollectedAt:      collectedAt,
	}, nil
}

func parseMetricLine(kv map[string]string, collectedAt time.Time) (*GPUMetricInfo, error) {
	required := []string{"vendor", "index", "uuid"}
	for _, k := range required {
		if _, ok := kv[k]; !ok {
			return nil, fmt.Errorf("METRIC missing required field %q", k)
		}
	}

	index, err := strconv.Atoi(kv["index"])
	if err != nil {
		return nil, fmt.Errorf("invalid index: %w", err)
	}

	metric := &GPUMetricInfo{
		Vendor:      kv["vendor"],
		Index:       index,
		Name:        stringOr(kv, "name", ""),
		UUID:        kv["uuid"],
		Health:      stringOr(kv, "health", "unknown"),
		CollectedAt: collectedAt,
	}

	if v, ok := kv["memory_total_bytes"]; ok {
		metric.MemoryTotalBytes, _ = strconv.ParseUint(v, 10, 64)
	}
	if v, ok := kv["memory_used_bytes"]; ok {
		metric.MemoryUsedBytes, _ = strconv.ParseUint(v, 10, 64)
	}
	if v, ok := kv["memory_free_bytes"]; ok {
		metric.MemoryFreeBytes, _ = strconv.ParseUint(v, 10, 64)
	}
	if v, ok := kv["gpu_utilization_percent"]; ok {
		metric.GPUUtilization = parseFloatOrNil(v)
	}
	if v, ok := kv["memory_utilization_percent"]; ok {
		metric.MemoryUtilization = parseFloatOrNil(v)
	}
	if v, ok := kv["temperature_celsius"]; ok {
		metric.Temperature = parseFloatOrNil(v)
	}
	if v, ok := kv["power_draw_watts"]; ok {
		metric.PowerDraw = parseFloatOrNil(v)
	}

	return metric, nil
}

func stringOr(kv map[string]string, key, def string) string {
	if v, ok := kv[key]; ok {
		return v
	}
	return def
}

// parseFloatOrNil parses a string to *float64, returning nil for null/N/A/empty/error.
func parseFloatOrNil(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" || s == "N/A" || s == "[N/A]" || s == "Unknown" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// parseUintOrZero parses a string to uint64, returning 0 for empty or invalid.
func parseUintOrZero(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" || s == "[N/A]" {
		return 0
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}
