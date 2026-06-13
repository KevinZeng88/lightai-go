package collector

import (
	"testing"
	"time"
)

func TestParseNvidiaCSV_Success(t *testing.T) {
	input := "0, NVIDIA GeForce RTX 5090 Laptop GPU, GPU-XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX, 00000000:02:00.0, 610.47, 24463, 0, 24137, 0, 0, 43, 15.19"
	now := time.Now()

	devices, err := parseNvidiaCSV(input, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}

	d := devices[0]
	if d.Vendor != "nvidia" {
		t.Errorf("expected vendor nvidia, got %s", d.Vendor)
	}
	if d.Index != 0 {
		t.Errorf("expected index 0, got %d", d.Index)
	}
	if d.Name != "NVIDIA GeForce RTX 5090 Laptop GPU" {
		t.Errorf("unexpected name: %s", d.Name)
	}

	// 24463 MB * 1024 * 1024 = 25,650,495,488 bytes
	expectedBytes := uint64(24463) * 1024 * 1024
	if d.MemoryTotalBytes != expectedBytes {
		t.Errorf("expected %d bytes, got %d", expectedBytes, d.MemoryTotalBytes)
	}
	if d.Status != "available" {
		t.Errorf("expected status available, got %s", d.Status)
	}
}

func TestParseNvidiaMetricsCSV_Success(t *testing.T) {
	input := "0, NVIDIA GeForce RTX 5090 Laptop GPU, GPU-XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX, 00000000:02:00.0, 610.47, 24463, 0, 24137, 0, 0, 43, 15.19"
	now := time.Now()

	metrics, err := parseNvidiaMetricsCSV(input, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	m := metrics[0]
	if m.Vendor != "nvidia" {
		t.Errorf("expected vendor nvidia, got %s", m.Vendor)
	}

	// memory.used: 0 MB → 0 bytes
	if m.MemoryUsedBytes != 0 {
		t.Errorf("expected 0 used bytes, got %d", m.MemoryUsedBytes)
	}

	// memory.free: 24137 MB → 24137 * 1024 * 1024 bytes
	expectedFree := uint64(24137) * 1024 * 1024
	if m.MemoryFreeBytes != expectedFree {
		t.Errorf("expected %d free bytes, got %d", expectedFree, m.MemoryFreeBytes)
	}

	// utilization.gpu: 0%
	if m.GPUUtilization == nil || *m.GPUUtilization != 0 {
		t.Error("expected GPU utilization 0")
	}

	// temperature: 43
	if m.Temperature == nil || *m.Temperature != 43 {
		t.Error("expected temperature 43")
	}

	// power.draw: 15.19
	if m.PowerDraw == nil || *m.PowerDraw != 15.19 {
		t.Error("expected power.draw 15.19")
	}
}

func TestParseNvidiaCSV_Empty(t *testing.T) {
	now := time.Now()
	devices, err := parseNvidiaCSV("", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestParseNvidiaCSV_PowerNA(t *testing.T) {
	// power.draw = N/A
	input := "0, Test GPU, GPU-XXX, 0000:00:00.0, 1.0, 8192, 1024, 7168, 50, 30, 60, N/A"
	now := time.Now()

	metrics, err := parseNvidiaMetricsCSV(input, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].PowerDraw != nil {
		t.Error("expected nil power.draw for N/A value")
	}
}

func TestParseFloatOrNil_NA(t *testing.T) {
	v, err := parseFloatOrNil("N/A")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if v != nil {
		t.Error("expected nil for N/A")
	}
}

func TestParseFloatOrNil_Empty(t *testing.T) {
	v, err := parseFloatOrNil("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if v != nil {
		t.Error("expected nil for empty string")
	}
}

func TestParseUintOrZero_NA(t *testing.T) {
	v, err := parseUintOrZero("N/A")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if v != 0 {
		t.Errorf("expected 0 for N/A, got %d", v)
	}
}
