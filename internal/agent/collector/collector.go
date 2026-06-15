// Package collector defines the collector interfaces and registry for LightAI Agent.
package collector

import (
	"context"
	"time"
)

// CollectedAt is the timestamp type for collection snapshots.
type CollectedAt = time.Time

// SystemSnapshot holds OS-level resource information.
type SystemSnapshot struct {
	Hostname          string                     `json:"hostname"`
	OS                string                     `json:"os"`
	OSVersion         string                     `json:"os_version"`
	KernelVersion     string                     `json:"kernel_version"`
	CPUModel          string                     `json:"cpu_model"`
	CPUCores          int                        `json:"cpu_cores"`
	CPUUtilization    float64                    `json:"cpu_utilization_percent"` // 0-100
	Load1             float64                    `json:"load1"`
	Load5             float64                    `json:"load5"`
	Load15            float64                    `json:"load15"`
	MemoryTotalBytes  uint64                     `json:"memory_total_bytes"`
	MemoryUsedBytes   uint64                     `json:"memory_used_bytes"`
	SwapTotalBytes    uint64                     `json:"swap_total_bytes"`
	SwapUsedBytes     uint64                     `json:"swap_used_bytes"`
	UptimeSeconds     uint64                     `json:"uptime_seconds"`
	Filesystems       []FilesystemSnapshot       `json:"filesystems"`
	NetworkInterfaces []NetworkInterfaceSnapshot `json:"network_interfaces"`
	CollectedAt       time.Time                  `json:"collected_at"`
}

// FilesystemSnapshot holds filesystem information.
type FilesystemSnapshot struct {
	MountPoint  string  `json:"mount_point"`
	Device      string  `json:"device"`
	FSType      string  `json:"fs_type"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"` // 0-100
}

// NetworkInterfaceSnapshot holds network interface information.
type NetworkInterfaceSnapshot struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
	Up        bool     `json:"up"`
	BytesRecv uint64   `json:"bytes_recv"`
	BytesSent uint64   `json:"bytes_sent"`
}

// GPUDeviceInfo is a parser-only raw record from vendor collector scripts.
// Do NOT use in report payload, server ingest, API response, Web, or Prometheus exporter.
// All downstream consumers must use the unified GPUResource model.
type GPUDeviceInfo struct {
	Vendor           string    `json:"vendor"`
	Index            int       `json:"index"`
	Name             string    `json:"name"`
	UUID             string    `json:"uuid"`
	PCIBusID         string    `json:"pci_bus_id"`
	DriverVersion    string    `json:"driver_version"`
	MemoryTotalBytes uint64    `json:"memory_total_bytes"`
	Status           string    `json:"status"` // available / in_use / error
	CollectedAt      time.Time `json:"collected_at"`
}

// GPUMetricInfo is a parser-only raw record from vendor collector scripts.
// Do NOT use in report payload, server ingest, API response, Web, or Prometheus exporter.
// All downstream consumers must use the unified GPUResource model.
type GPUMetricInfo struct {
	Vendor            string    `json:"vendor"`
	Index             int       `json:"index"`
	Name              string    `json:"name"`
	UUID              string    `json:"uuid"`
	MemoryTotalBytes  uint64    `json:"memory_total_bytes"`
	MemoryUsedBytes   uint64    `json:"memory_used_bytes"`
	MemoryFreeBytes   uint64    `json:"memory_free_bytes"`
	GPUUtilization    *float64  `json:"gpu_utilization_percent,omitempty"`    // 0-100
	MemoryUtilization *float64  `json:"memory_utilization_percent,omitempty"` // 0-100
	Temperature       *float64  `json:"temperature_celsius,omitempty"`
	PowerDraw         *float64  `json:"power_draw_watts,omitempty"`
	Health            string    `json:"health"` // healthy / warning / error / unknown
	CollectedAt       time.Time `json:"collected_at"`
}

// CollectorDiagnosis holds diagnostic information for a collector.
type CollectorDiagnosis struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"` // system / gpu
	Vendor    string    `json:"vendor,omitempty"`
	Available bool      `json:"available"`
	ToolPath  string    `json:"tool_path,omitempty"`
	Error     string    `json:"error,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// SystemCollector collects OS-level system information.
type SystemCollector interface {
	Name() string
	Collect(ctx context.Context) (*SystemSnapshot, *CollectorDiagnosis)
}

// GPUCollector collects GPU device and metric information.
type GPUCollector interface {
	Name() string
	Vendor() string
	Discover(ctx context.Context) ([]GPUDeviceInfo, *CollectorDiagnosis)
	Metrics(ctx context.Context) ([]GPUMetricInfo, *CollectorDiagnosis)
}

// GPUResource is the unified, vendor-neutral GPU model.
// All GPU collectors must normalize into this struct — it is the single
// data model for Agent /metrics, Agent report, Server storage, API, and Web.
// Future API/SDK/daemon-based collectors output the same type.
type GPUResource struct {
	Vendor           string    `json:"vendor"` // Currently supported: nvidia, metax. Future/unsupported: ascend, cambricon, hygon, intel, amd, unknown
	Index            int       `json:"index"`  // physical GPU index on the node
	UUID             string    `json:"uuid"`   // primary unique identifier
	Name             string    `json:"name"`   // human-readable GPU name
	PCIBusID         string    `json:"pci_bus_id"`
	DriverVersion    string    `json:"driver_version"`
	MemoryTotalBytes uint64    `json:"memory_total_bytes"`
	MemoryUsedBytes  uint64    `json:"memory_used_bytes"`
	MemoryFreeBytes  uint64    `json:"memory_free_bytes"`
	GPUUtilization   *float64  `json:"gpu_utilization_percent,omitempty"`
	MemUtilization   *float64  `json:"memory_utilization_percent,omitempty"`
	Temperature      *float64  `json:"temperature_celsius,omitempty"`
	PowerDraw        *float64  `json:"power_draw_watts,omitempty"`
	Health           string    `json:"health"` // healthy / degraded / error / unknown
	Status           string    `json:"status"` // available / unavailable
	CollectedAt      time.Time `json:"collected_at"`
}

// UniqueKey returns a stable identifier: vendor+uuid, or vendor+index if uuid is empty.
func (g *GPUResource) UniqueKey() string {
	if g.UUID != "" {
		return g.Vendor + ":" + g.UUID
	}
	return g.Vendor + ":" + itoa(g.Index)
}

// NormalizeGPUs merges discover devices and metrics into a unified GPUResource slice.
// Devices provide identity (vendor, index, uuid, name, pci, driver).
// Metrics provide current values (memory used/free, utilization, temperature, power, health).
// Merge key: vendor + uuid (or vendor + index if uuid is empty).
// Missing values from discover (e.g. memory_total_bytes=null) are filled from metrics.
func NormalizeGPUs(devices []GPUDeviceInfo, metrics []GPUMetricInfo) []GPUResource {
	// Index metrics by key.
	metricByKey := make(map[string]*GPUMetricInfo)
	for i := range metrics {
		m := &metrics[i]
		key := m.Vendor + ":" + m.UUID
		if m.UUID == "" {
			key = m.Vendor + ":" + itoa(m.Index)
		}
		metricByKey[key] = m
	}

	seen := make(map[string]bool)
	var result []GPUResource

	// Merge devices with their matching metrics.
	for _, d := range devices {
		key := d.Vendor + ":" + d.UUID
		if d.UUID == "" {
			key = d.Vendor + ":" + itoa(d.Index)
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		r := GPUResource{
			Vendor:        d.Vendor,
			Index:         d.Index,
			UUID:          d.UUID,
			Name:          d.Name,
			PCIBusID:      d.PCIBusID,
			DriverVersion: d.DriverVersion,
			Status:        d.Status,
			CollectedAt:   d.CollectedAt,
		}

		if m, ok := metricByKey[key]; ok {
			// Prefer metric fields over device fields for runtime values.
			if m.Name != "" {
				r.Name = m.Name
			}
			if m.MemoryTotalBytes > 0 {
				r.MemoryTotalBytes = m.MemoryTotalBytes
			} else {
				r.MemoryTotalBytes = d.MemoryTotalBytes
			}
			r.MemoryUsedBytes = m.MemoryUsedBytes
			r.MemoryFreeBytes = m.MemoryFreeBytes
			r.GPUUtilization = m.GPUUtilization
			r.MemUtilization = m.MemoryUtilization
			r.Temperature = m.Temperature
			r.PowerDraw = m.PowerDraw
			r.Health = m.Health
			if m.CollectedAt.After(r.CollectedAt) {
				r.CollectedAt = m.CollectedAt
			}
		} else {
			// No matching metric — use discover data only.
			r.MemoryTotalBytes = d.MemoryTotalBytes
			r.Health = "unknown"
			if r.Status == "" {
				r.Status = "available"
			}
		}
		if r.Status == "" {
			r.Status = "available"
		}
		if r.Health == "" {
			r.Health = "unknown"
		}

		result = append(result, r)
	}

	// Add metrics that have no matching device (rare, but handle gracefully).
	for _, m := range metrics {
		key := m.Vendor + ":" + m.UUID
		if m.UUID == "" {
			key = m.Vendor + ":" + itoa(m.Index)
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		r := GPUResource{
			Vendor:           m.Vendor,
			Index:            m.Index,
			UUID:             m.UUID,
			Name:             m.Name,
			MemoryTotalBytes: m.MemoryTotalBytes,
			MemoryUsedBytes:  m.MemoryUsedBytes,
			MemoryFreeBytes:  m.MemoryFreeBytes,
			GPUUtilization:   m.GPUUtilization,
			MemUtilization:   m.MemoryUtilization,
			Temperature:      m.Temperature,
			PowerDraw:        m.PowerDraw,
			Health:           m.Health,
			Status:           "available",
			CollectedAt:      m.CollectedAt,
		}
		if r.Health == "" {
			r.Health = "unknown"
		}
		result = append(result, r)
	}

	return result
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	n := i
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// ResourceReport is the full resource report sent to the server.
type ResourceReport struct {
	AgentID      string               `json:"agent_id"`
	System       *SystemSnapshot      `json:"system"`
	GPUDevices   []GPUDeviceInfo      `json:"gpu_devices"`
	GPUMetrics   []GPUMetricInfo      `json:"gpu_metrics"`
	GPUResources []GPUResource        `json:"gpu_resources"` // unified, vendor-neutral
	Diagnostics  []CollectorDiagnosis `json:"diagnostics"`
	CollectedAt  time.Time            `json:"collected_at"`
}
