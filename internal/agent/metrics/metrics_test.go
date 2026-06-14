package metrics

import (
	"os"
	"strings"
	"testing"
	"time"

	"lightai-go/internal/agent/collector"

	"github.com/prometheus/client_golang/prometheus"
)

// readFixture reads a test fixture file.
func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return string(data)
}

// TestMetaX8GPUParseDiscover verifies parsing the 8-card MetaX discover output.
func TestMetaX8GPUParseDiscover(t *testing.T) {
	output := readFixture(t, "../../../tests/fixtures/collectors/metax/discover_8x_c500.txt")
	now := time.Now()

	devices, metrics, diag, err := collector.ParseProtocolOutput(output, now)
	if err != nil {
		t.Fatalf("ParseProtocolOutput failed: %v", err)
	}

	if len(devices) != 8 {
		t.Errorf("expected 8 devices, got %d", len(devices))
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics from discover, got %d", len(metrics))
	}
	if !diag.Available {
		t.Errorf("expected diag.Available=true, got false")
	}

	for i, dev := range devices {
		if dev.MemoryTotalBytes != 0 {
			t.Errorf("GPU %d: expected memory_total_bytes=0 from null, got %d", i, dev.MemoryTotalBytes)
		}
		if dev.UUID == "" {
			t.Errorf("GPU %d: missing uuid", i)
		}
	}
}

// TestMetaX8GPUParseMetrics verifies parsing the 8-card MetaX metrics output.
func TestMetaX8GPUParseMetrics(t *testing.T) {
	output := readFixture(t, "../../../tests/fixtures/collectors/metax/metrics_8x_c500.txt")
	now := time.Now()

	devices, metrics, diag, err := collector.ParseProtocolOutput(output, now)
	if err != nil {
		t.Fatalf("ParseProtocolOutput failed: %v", err)
	}

	if len(metrics) != 8 {
		t.Errorf("expected 8 metrics, got %d", len(metrics))
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices from metrics, got %d", len(devices))
	}
	if !diag.Available {
		t.Errorf("expected diag.Available=true")
	}

	for i, m := range metrics {
		if m.Vendor != "metax" {
			t.Errorf("METRIC %d: expected vendor=metax, got %s", i, m.Vendor)
		}
		if m.MemoryTotalBytes == 0 {
			t.Errorf("METRIC %d: expected non-zero memory_total_bytes", i)
		}
	}
}

// TestMetaX8GPUNormalize verifies NormalizeGPUs produces 8 unified GPUResources.
func TestMetaX8GPUNormalize(t *testing.T) {
	discoverOut := readFixture(t, "../../../tests/fixtures/collectors/metax/discover_8x_c500.txt")
	metricsOut := readFixture(t, "../../../tests/fixtures/collectors/metax/metrics_8x_c500.txt")
	now := time.Now()

	devices, _, _, _ := collector.ParseProtocolOutput(discoverOut, now)
	_, gpuMetrics, _, _ := collector.ParseProtocolOutput(metricsOut, now)

	resources := collector.NormalizeGPUs(devices, gpuMetrics)
	if len(resources) != 8 {
		t.Fatalf("expected 8 GPUResources, got %d", len(resources))
	}

	for i, g := range resources {
		if g.Vendor != "metax" {
			t.Errorf("GPU %d: expected vendor=metax, got %s", i, g.Vendor)
		}
		if g.MemoryTotalBytes == 0 {
			t.Errorf("GPU %d: memory_total_bytes should be filled from metrics, got 0", i)
		}
		if g.MemoryFreeBytes == 0 && g.MemoryUsedBytes != g.MemoryTotalBytes {
			t.Errorf("GPU %d: memory_free_bytes=0 but used < total", i)
		}
		if g.PCIBusID == "" {
			t.Errorf("GPU %d: pci_bus_id should come from discover", i)
		}
		if g.DriverVersion == "" {
			t.Errorf("GPU %d: driver_version should come from discover", i)
		}
		t.Logf("GPU %d: vendor=%s uuid=%s name=%s mem_total=%d mem_used=%d mem_free=%d health=%s status=%s",
			i, g.Vendor, g.UUID, g.Name, g.MemoryTotalBytes, g.MemoryUsedBytes, g.MemoryFreeBytes, g.Health, g.Status)
	}
}

// TestMetaX8GPUSnapshotNoAccumulation verifies 3 rounds still produce 8 GPUs.
func TestMetaX8GPUSnapshotNoAccumulation(t *testing.T) {
	discoverOut := readFixture(t, "../../../tests/fixtures/collectors/metax/discover_8x_c500.txt")
	metricsOut := readFixture(t, "../../../tests/fixtures/collectors/metax/metrics_8x_c500.txt")
	now := time.Now()

	devices, _, _, _ := collector.ParseProtocolOutput(discoverOut, now)
	_, gpuMetrics, _, _ := collector.ParseProtocolOutput(metricsOut, now)
	resources := collector.NormalizeGPUs(devices, gpuMetrics)

	snap := NewSnapshot("node-01", "agent-01", "k8s-master1")
	for round := 0; round < 3; round++ {
		snap.SetGPUResources(resources)
		snap.mu.RLock()
		count := len(snap.GPUResources)
		snap.mu.RUnlock()
		if count != 8 {
			t.Errorf("round %d: expected 8 GPUs, got %d", round, count)
		}
	}
}

// TestMetaX8GPUPrometheusNoDuplicates verifies no duplicate time series.
func TestMetaX8GPUPrometheusNoDuplicates(t *testing.T) {
	discoverOut := readFixture(t, "../../../tests/fixtures/collectors/metax/discover_8x_c500.txt")
	metricsOut := readFixture(t, "../../../tests/fixtures/collectors/metax/metrics_8x_c500.txt")
	now := time.Now()

	devices, _, _, _ := collector.ParseProtocolOutput(discoverOut, now)
	_, gpuMetrics, _, _ := collector.ParseProtocolOutput(metricsOut, now)
	resources := collector.NormalizeGPUs(devices, gpuMetrics)

	snap := NewSnapshot("node-860947cd-1165-490e-b8c2-54d0426ee547", "agent-01", "k8s-master1")
	snap.SetGPUResources(resources)

	reg := prometheus.NewRegistry()
	reg.MustRegister(newGPUCollector(snap))

	metricFamilies, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed (this was the MetaX 8-card bug!): %v", err)
	}

	counts := make(map[string]int)
	for _, mf := range metricFamilies {
		name := mf.GetName()
		counts[name] += len(mf.GetMetric())
	}

	expected8 := []string{
		"lightai_gpu_memory_total_bytes",
		"lightai_gpu_memory_used_bytes",
		"lightai_gpu_memory_free_bytes",
		"lightai_gpu_available_status",
		"lightai_gpu_health_status",
	}
	for _, name := range expected8 {
		if counts[name] != 8 {
			t.Errorf("expected 8 x %s, got %d", name, counts[name])
		}
	}
	t.Logf("All GPU metric counts: %v", counts)
}

// TestLegalZeroValues verifies 0 values are preserved.
func TestLegalZeroValues(t *testing.T) {
	util0 := 0.0
	memUtil0 := 0.0
	tempOk := 42.0
	powerZero := 0.0

	resources := []collector.GPUResource{
		{Vendor: "nvidia", Index: 0, UUID: "GPU-test", Name: "Test GPU",
			MemoryTotalBytes: 25651314688, MemoryUsedBytes: 0, MemoryFreeBytes: 25651314688,
			GPUUtilization: &util0, MemUtilization: &memUtil0,
			Temperature: &tempOk, PowerDraw: &powerZero,
			Health: "healthy", Status: "available"},
	}

	snap := NewSnapshot("node-01", "agent-01", "host-01")
	snap.SetGPUResources(resources)

	reg := prometheus.NewRegistry()
	reg.MustRegister(newGPUCollector(snap))

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}

	for _, mf := range mfs {
		name := mf.GetName()
		for _, m := range mf.GetMetric() {
			val := m.GetGauge().GetValue()
			switch name {
			case "lightai_gpu_memory_used_bytes":
				if val != 0 {
					t.Errorf("memory_used_bytes: expected 0, got %f", val)
				}
			case "lightai_gpu_utilization_percent":
				if val != 0 {
					t.Errorf("gpu_utilization: expected 0, got %f", val)
				}
			case "lightai_gpu_power_draw_watts":
				if val != 0 {
					t.Errorf("power_draw: expected 0, got %f", val)
				}
			case "lightai_gpu_available_status":
				if val != 1 {
					t.Errorf("available: expected 1, got %f", val)
				}
			}
		}
	}
}

// TestGPUResourceUniqueKey verifies the unique key logic.
func TestGPUResourceUniqueKey(t *testing.T) {
	a := collector.GPUResource{Vendor: "nvidia", UUID: "GPU-aaa", Index: 0}
	b := collector.GPUResource{Vendor: "nvidia", UUID: "GPU-aaa", Index: 1}
	c := collector.GPUResource{Vendor: "metax", UUID: "", Index: 0}
	d := collector.GPUResource{Vendor: "metax", UUID: "", Index: 1}

	if a.UniqueKey() != b.UniqueKey() {
		t.Error("same vendor+uuid should have same key")
	}
	if c.UniqueKey() == d.UniqueKey() {
		t.Error("different index should have different key when uuid is empty")
	}
	if a.UniqueKey() == c.UniqueKey() {
		t.Error("different vendor should have different key")
	}
}

// TestGPUPrometheusDuplicateDetection ensures duplicate GPUResources cause Gather failure.
func TestGPUPrometheusDuplicateDetection(t *testing.T) {
	resources := []collector.GPUResource{
		{Vendor: "metax", Index: 0, UUID: "GPU-aaa", Name: "MetaX C500",
			MemoryTotalBytes: 68719476736, MemoryUsedBytes: 0, MemoryFreeBytes: 68719476736,
			Health: "healthy", Status: "available"},
		{Vendor: "metax", Index: 0, UUID: "GPU-aaa", Name: "MetaX C500", // duplicate
			MemoryTotalBytes: 68719476736, MemoryUsedBytes: 0, MemoryFreeBytes: 68719476736,
			Health: "healthy", Status: "available"},
	}

	snap := NewSnapshot("node-01", "agent-01", "host-01")
	snap.SetGPUResources(resources)

	reg := prometheus.NewRegistry()
	reg.MustRegister(newGPUCollector(snap))

	_, err := reg.Gather()
	if err == nil {
		t.Error("expected Gather to fail with duplicate resources, but it succeeded")
	} else {
		t.Logf("Correctly rejected duplicates: %v", err)
	}

	// Non-duplicate case must succeed.
	nonDup := []collector.GPUResource{
		{Vendor: "metax", Index: 0, UUID: "GPU-aaa", Name: "MetaX C500",
			MemoryTotalBytes: 68719476736, Health: "healthy", Status: "available"},
	}
	snap2 := NewSnapshot("node-01", "agent-01", "host-01")
	snap2.SetGPUResources(nonDup)

	reg2 := prometheus.NewRegistry()
	reg2.MustRegister(newGPUCollector(snap2))

	_, err = reg2.Gather()
	if err != nil {
		t.Fatalf("non-duplicate Gather should succeed: %v", err)
	}
}

// TestVendorNeutral verifies all known vendors produce valid GPUResource.
func TestVendorNeutral(t *testing.T) {
	vendors := []string{"nvidia", "metax", "ascend", "cambricon", "hygon", "intel", "unknown"}
	for _, vendor := range vendors {
		r := collector.GPUResource{
			Vendor: vendor, Index: 0, UUID: "GPU-" + vendor,
			Name: vendor + " GPU", Health: "healthy", Status: "available",
		}
		key := r.UniqueKey()
		if key == "" || key == ":" {
			t.Errorf("vendor=%s: empty UniqueKey", vendor)
		}
	}
}

// Ensure dto import is used (for prometheus.Gather return type processing).
var _ = strings.TrimSpace
