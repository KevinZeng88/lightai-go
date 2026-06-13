package collector

import (
	"testing"
	"time"
)

func TestParseProtocolOutput_Full(t *testing.T) {
	input := `STATUS vendor=nvidia ok=true message=ok
DEVICE vendor=nvidia index=0 uuid=GPU-XXX name="NVIDIA GeForce RTX 5090 Laptop GPU" pci_bus_id=00000000:02:00.0 driver_version=610.47 memory_total_bytes=25651314688
METRIC vendor=nvidia index=0 uuid=GPU-XXX memory_total_bytes=25651314688 memory_used_bytes=0 memory_free_bytes=25309478912 gpu_utilization_percent=0 memory_utilization_percent=0 temperature_celsius=43 power_draw_watts=13.39 health=healthy status=available`

	now := time.Now()
	devices, metrics, diag, err := ParseProtocolOutput(input, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !diag.Available {
		t.Error("expected available=true")
	}
	if diag.Vendor != "nvidia" {
		t.Errorf("expected vendor=nvidia, got %s", diag.Vendor)
	}

	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	d := devices[0]
	if d.Vendor != "nvidia" {
		t.Errorf("expected nvidia, got %s", d.Vendor)
	}
	if d.Index != 0 {
		t.Errorf("expected index 0, got %d", d.Index)
	}
	if d.Name != "NVIDIA GeForce RTX 5090 Laptop GPU" {
		t.Errorf("unexpected name: %s", d.Name)
	}
	if d.UUID != "GPU-XXX" {
		t.Errorf("unexpected uuid: %s", d.UUID)
	}
	if d.MemoryTotalBytes != 25651314688 {
		t.Errorf("unexpected memory: %d", d.MemoryTotalBytes)
	}

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	m := metrics[0]
	if m.MemoryUsedBytes != 0 {
		t.Errorf("expected 0 used, got %d", m.MemoryUsedBytes)
	}
	if m.MemoryFreeBytes != 25309478912 {
		t.Errorf("unexpected free: %d", m.MemoryFreeBytes)
	}
	if m.Temperature == nil || *m.Temperature != 43 {
		t.Error("expected temperature 43")
	}
	if m.PowerDraw == nil || *m.PowerDraw != 13.39 {
		t.Error("expected power.draw 13.39")
	}
}

func TestParseProtocolOutput_NullValues(t *testing.T) {
	input := `STATUS vendor=nvidia ok=true message=ok
DEVICE vendor=nvidia index=0 uuid=GPU-XXX name="Test GPU"
METRIC vendor=nvidia index=0 uuid=GPU-XXX memory_used_bytes=0 memory_free_bytes=0 gpu_utilization_percent=null memory_utilization_percent=N/A temperature_celsius= power_draw_watts=Unknown health=healthy`

	now := time.Now()
	_, metrics, _, err := ParseProtocolOutput(input, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := metrics[0]
	if m.GPUUtilization != nil {
		t.Error("expected nil GPUUtilization for null value")
	}
	if m.MemoryUtilization != nil {
		t.Error("expected nil MemoryUtilization for N/A")
	}
	if m.Temperature != nil {
		t.Error("expected nil Temperature for empty")
	}
	if m.PowerDraw != nil {
		t.Error("expected nil PowerDraw for Unknown")
	}
}

func TestParseProtocolOutput_NotAvailable(t *testing.T) {
	input := `STATUS vendor=nvidia ok=false message="nvidia-smi not found"`

	now := time.Now()
	devices, metrics, diag, err := ParseProtocolOutput(input, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diag.Available {
		t.Error("expected not available")
	}
	if diag.Error != "nvidia-smi not found" {
		t.Errorf("unexpected error: %s", diag.Error)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(metrics))
	}
}

func TestParseProtocolOutput_MissingRequired(t *testing.T) {
	input := `DEVICE vendor=nvidia index=0 name="GPU"`
	_, _, _, err := ParseProtocolOutput(input, time.Now())
	if err == nil {
		t.Error("expected error for missing uuid")
	}
}

func TestParseProtocolOutput_Malformed(t *testing.T) {
	input := `GARBAGE line without proper format`
	_, _, _, err := ParseProtocolOutput(input, time.Now())
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestParseProtocolOutput_Empty(t *testing.T) {
	devices, metrics, _, err := ParseProtocolOutput("", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(metrics))
	}
}

func TestParseProtocolOutput_Comments(t *testing.T) {
	input := `# This is a comment
STATUS vendor=test ok=true
# Another comment
DEVICE vendor=test index=0 uuid=U-1 name="Test"`
	devices, _, _, err := ParseProtocolOutput(input, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(devices))
	}
}

func TestParseKeyValues(t *testing.T) {
	kv, err := parseKeyValues(`key1=value1 key2="value with spaces" key3=123`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv["key1"] != "value1" {
		t.Errorf("expected value1, got %s", kv["key1"])
	}
	if kv["key2"] != "value with spaces" {
		t.Errorf("expected 'value with spaces', got %s", kv["key2"])
	}
	if kv["key3"] != "123" {
		t.Errorf("expected 123, got %s", kv["key3"])
	}
}

func TestParseKeyValues_EqualsInValue(t *testing.T) {
	kv, err := parseKeyValues(`url=http://example.com?x=1 name=test`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv["url"] != "http://example.com?x=1" {
		t.Errorf("unexpected url: %s", kv["url"])
	}
}
