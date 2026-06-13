package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// SystemCollectorImpl collects OS-level system information using gopsutil.
type SystemCollectorImpl struct{}

// NewSystemCollector creates a new system collector.
func NewSystemCollector() *SystemCollectorImpl {
	return &SystemCollectorImpl{}
}

// Name returns the collector name.
func (s *SystemCollectorImpl) Name() string {
	return "system"
}

// Collect gathers current system resource information.
func (s *SystemCollectorImpl) Collect(ctx context.Context) (*SystemSnapshot, *CollectorDiagnosis) {
	collectedAt := time.Now()
	diag := &CollectorDiagnosis{
		Name:      "system",
		Type:      "system",
		Available: true,
		CheckedAt: collectedAt,
	}

	snapshot := &SystemSnapshot{
		CollectedAt: collectedAt,
	}

	// Host info.
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		diag.Available = false
		diag.Error = fmt.Sprintf("host info: %v", err)
		return nil, diag
	}
	snapshot.Hostname = hostInfo.Hostname
	snapshot.OS = hostInfo.OS
	snapshot.OSVersion = hostInfo.PlatformVersion
	snapshot.KernelVersion = hostInfo.KernelVersion

	// CPU info.
	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err == nil && len(cpuInfo) > 0 {
		snapshot.CPUModel = cpuInfo[0].ModelName
	}
	snapshot.CPUCores, _ = cpu.CountsWithContext(ctx, true)

	cpuPercents, err := cpu.PercentWithContext(ctx, 0, false)
	if err == nil && len(cpuPercents) > 0 {
		snapshot.CPUUtilization = cpuPercents[0]
	}

	// Memory.
	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		snapshot.MemoryTotalBytes = memInfo.Total
		snapshot.MemoryUsedBytes = memInfo.Used
	}

	// Swap.
	swapInfo, err := mem.SwapMemoryWithContext(ctx)
	if err == nil {
		snapshot.SwapTotalBytes = swapInfo.Total
		snapshot.SwapUsedBytes = swapInfo.Used
	}

	// Filesystems.
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err == nil {
		for _, p := range partitions {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			snapshot.Filesystems = append(snapshot.Filesystems, FilesystemSnapshot{
				MountPoint:  p.Mountpoint,
				Device:      p.Device,
				FSType:      p.Fstype,
				TotalBytes:  usage.Total,
				UsedBytes:   usage.Used,
				FreeBytes:   usage.Free,
				UsedPercent: usage.UsedPercent,
			})
		}
	}

	// Network interfaces.
	interfaces, err := net.InterfacesWithContext(ctx)
	if err == nil {
		for _, iface := range interfaces {
			addrs := make([]string, 0)
			for _, addr := range iface.Addrs {
				addrs = append(addrs, addr.Addr)
			}
			counters, _ := net.IOCountersWithContext(ctx, false)
			var bytesRecv, bytesSent uint64
			for _, c := range counters {
				if c.Name == iface.Name {
					bytesRecv = c.BytesRecv
					bytesSent = c.BytesSent
					break
				}
			}
			// Check if interface is up by looking for "up" in flags.
			isUp := false
			for _, flag := range iface.Flags {
				if flag == "up" {
					isUp = true
					break
				}
			}
			snapshot.NetworkInterfaces = append(snapshot.NetworkInterfaces, NetworkInterfaceSnapshot{
				Name:      iface.Name,
				Addresses: addrs,
				Up:        isUp,
				BytesRecv: bytesRecv,
				BytesSent: bytesSent,
			})
		}
	}

	return snapshot, diag
}

// Ensure interface satisfaction.
var _ SystemCollector = (*SystemCollectorImpl)(nil)
