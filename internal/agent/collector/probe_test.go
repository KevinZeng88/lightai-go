package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTempScript writes a shell script to a temp file and returns its path.
func writeTempScript(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sh")
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write temp script: %v", err)
	}
	return path
}

// TestProbeNvidiaAvailable verifies: DEVICE output → available=true.
func TestProbeNvidiaAvailable(t *testing.T) {
	script := writeTempScript(t, `#!/bin/sh
echo 'STATUS vendor=nvidia name=nvidia ok=true'
echo 'DEVICE vendor=nvidia index=0 name="NVIDIA RTX 5090" uuid=GPU-abc'
echo 'DEVICE vendor=nvidia index=1 name="NVIDIA RTX 5090" uuid=GPU-def'
exit 0
`)
	def := ProbeDef{
		Name:        "nvidia",
		Vendor:      "nvidia",
		DiscoverCmd: "sh " + script,
		MetricsCmd:  "echo 'STATUS vendor=nvidia ok=true'",
		Timeout:     5 * time.Second,
	}
	result := Probe(context.Background(), def)
	if !result.Available {
		t.Errorf("expected available=true, got false: %s", result.Error)
	}
	if result.DeviceCount != 2 {
		t.Errorf("expected 2 devices, got %d", result.DeviceCount)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
}

// TestProbeMetaxAvailable verifies MetaX discover → available=true.
func TestProbeMetaxAvailable(t *testing.T) {
	script := writeTempScript(t, `#!/bin/sh
echo 'STATUS vendor=metax name=metax ok=true'
echo 'DEVICE vendor=metax index=0 name="MetaX C500" uuid=MX-abc'
exit 0
`)
	def := ProbeDef{
		Name:        "metax",
		Vendor:      "metax",
		DiscoverCmd: "sh " + script,
		MetricsCmd:  "echo 'STATUS vendor=metax ok=true'",
		Timeout:     5 * time.Second,
	}
	result := Probe(context.Background(), def)
	if !result.Available {
		t.Errorf("expected available=true, got false: %s", result.Error)
	}
	if result.DeviceCount != 1 {
		t.Errorf("expected 1 device, got %d", result.DeviceCount)
	}
}

// TestProbeBothVendorsAvailable verifies multi-vendor probes both succeed.
func TestProbeBothVendorsAvailable(t *testing.T) {
	nvidiaScript := writeTempScript(t, `#!/bin/sh
echo 'STATUS vendor=nvidia name=nvidia ok=true'
echo 'DEVICE vendor=nvidia index=0 name="RTX 5090" uuid=GPU-1'
exit 0
`)
	metaxScript := writeTempScript(t, `#!/bin/sh
echo 'STATUS vendor=metax name=metax ok=true'
echo 'DEVICE vendor=metax index=0 name="C500" uuid=MX-1'
exit 0
`)

	nvidia := ProbeDef{
		Name: "nvidia", Vendor: "nvidia",
		DiscoverCmd: "sh " + nvidiaScript,
		MetricsCmd:  "echo ok",
		Timeout:     5 * time.Second,
	}
	metax := ProbeDef{
		Name: "metax", Vendor: "metax",
		DiscoverCmd: "sh " + metaxScript,
		MetricsCmd:  "echo ok",
		Timeout:     5 * time.Second,
	}

	r1 := Probe(context.Background(), nvidia)
	r2 := Probe(context.Background(), metax)

	if !r1.Available {
		t.Errorf("nvidia should be available: %s", r1.Error)
	}
	if !r2.Available {
		t.Errorf("metax should be available: %s", r2.Error)
	}
}

// TestProbeExit10NotAvailable verifies exit 10 → skip, no error.
func TestProbeExit10NotAvailable(t *testing.T) {
	script := writeTempScript(t, `#!/bin/sh
echo 'STATUS vendor=nvidia name=nvidia ok=false message="No NVIDIA GPU found"'
exit 10
`)
	def := ProbeDef{
		Name:        "nvidia",
		Vendor:      "nvidia",
		DiscoverCmd: "sh " + script,
		MetricsCmd:  "echo ok",
		Timeout:     5 * time.Second,
	}
	result := Probe(context.Background(), def)
	if result.Available {
		t.Error("expected available=false for exit 10")
	}
	if result.ExitCode != 10 {
		t.Errorf("expected exit 10, got %d", result.ExitCode)
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
}

// TestProbeExit30ProbeFailed verifies exit >=30 → warn, not error.
func TestProbeExit30ProbeFailed(t *testing.T) {
	script := writeTempScript(t, `#!/bin/sh
echo "some error" >&2
exit 30
`)
	def := ProbeDef{
		Name:        "nvidia",
		Vendor:      "nvidia",
		DiscoverCmd: "sh " + script,
		MetricsCmd:  "echo ok",
		Timeout:     5 * time.Second,
	}
	result := Probe(context.Background(), def)
	if result.Available {
		t.Error("expected available=false for exit 30")
	}
	if result.ExitCode != 30 {
		t.Errorf("expected exit 30, got %d", result.ExitCode)
	}
}

// TestProbeNoDevices verifies exit 0 but no DEVICE → not available.
func TestProbeNoDevices(t *testing.T) {
	script := writeTempScript(t, `#!/bin/sh
echo 'STATUS vendor=nvidia name=nvidia ok=true'
exit 0
`)
	def := ProbeDef{
		Name:        "nvidia",
		Vendor:      "nvidia",
		DiscoverCmd: "sh " + script,
		MetricsCmd:  "echo ok",
		Timeout:     5 * time.Second,
	}
	result := Probe(context.Background(), def)
	if result.Available {
		t.Error("expected available=false when no DEVICE output")
	}
	if result.DeviceCount != 0 {
		t.Errorf("expected 0 devices, got %d", result.DeviceCount)
	}
}
