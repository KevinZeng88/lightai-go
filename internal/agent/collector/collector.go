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

// GPUDeviceInfo holds GPU device discovery information.
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

// GPUMetricInfo holds GPU metric information.
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

// ResourceReport is the full resource report sent to the server.
type ResourceReport struct {
	AgentID     string               `json:"agent_id"`
	System      *SystemSnapshot      `json:"system"`
	GPUDevices  []GPUDeviceInfo      `json:"gpu_devices"`
	GPUMetrics  []GPUMetricInfo      `json:"gpu_metrics"`
	Diagnostics []CollectorDiagnosis `json:"diagnostics"`
	CollectedAt time.Time            `json:"collected_at"`
}
