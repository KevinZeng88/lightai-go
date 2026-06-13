package collector

import (
	"context"
	"time"
)

// MockGPUCollector returns simulated GPU data for development and testing.
// It is only enabled in development/test profiles, never in production.
type MockGPUCollector struct{}

// NewMockGPUCollector creates a new mock GPU collector.
func NewMockGPUCollector() *MockGPUCollector {
	return &MockGPUCollector{}
}

// Name returns the collector name.
func (m *MockGPUCollector) Name() string {
	return "mock_gpu"
}

// Vendor returns the GPU vendor name.
func (m *MockGPUCollector) Vendor() string {
	return "mock"
}

// Discover returns a fixed simulated GPU device.
func (m *MockGPUCollector) Discover(ctx context.Context) ([]GPUDeviceInfo, *CollectorDiagnosis) {
	now := time.Now()
	diag := &CollectorDiagnosis{
		Name:      "mock_gpu",
		Type:      "gpu",
		Vendor:    "mock",
		Available: true,
		CheckedAt: now,
	}

	devices := []GPUDeviceInfo{
		{
			Vendor:           "mock",
			Index:            0,
			Name:             "Mock GPU 24GB",
			UUID:             "GPU-mock-00000000-0000-0000-0000-000000000000",
			PCIBusID:         "0000:00:00.0",
			DriverVersion:    "mock.0.1",
			MemoryTotalBytes: 24 * 1024 * 1024 * 1024, // 24 GB
			Status:           "available",
			CollectedAt:      now,
		},
	}

	return devices, diag
}

// Metrics returns fixed simulated GPU metrics.
func (m *MockGPUCollector) Metrics(ctx context.Context) ([]GPUMetricInfo, *CollectorDiagnosis) {
	now := time.Now()
	diag := &CollectorDiagnosis{
		Name:      "mock_gpu",
		Type:      "gpu",
		Vendor:    "mock",
		Available: true,
		CheckedAt: now,
	}

	gpuUtil := 45.0
	memUtil := 30.0
	temp := 55.0
	power := 120.0

	metrics := []GPUMetricInfo{
		{
			Vendor:            "mock",
			Index:             0,
			UUID:              "GPU-mock-00000000-0000-0000-0000-000000000000",
			MemoryUsedBytes:   8 * 1024 * 1024 * 1024,  // 8 GB used
			MemoryFreeBytes:   16 * 1024 * 1024 * 1024, // 16 GB free
			GPUUtilization:    &gpuUtil,
			MemoryUtilization: &memUtil,
			Temperature:       &temp,
			PowerDraw:         &power,
			Health:            "healthy",
			CollectedAt:       now,
		},
	}

	return metrics, diag
}

// Ensure interface satisfaction.
var _ GPUCollector = (*MockGPUCollector)(nil)
