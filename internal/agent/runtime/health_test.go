package runtime

import (
	"context"
	"strings"
	"testing"
)

func TestHealthCheckConfigDefaults(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled: true,
	}
	cfg.resolveDefaults(8080)

	if cfg.Scheme != "http" {
		t.Errorf("expected scheme=http, got %s", cfg.Scheme)
	}
	if cfg.Path != "/v1/models" {
		t.Errorf("expected path=/v1/models, got %s", cfg.Path)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port=8080, got %d", cfg.Port)
	}
	if cfg.ExpectedStatus != 200 {
		t.Errorf("expected status=200, got %d", cfg.ExpectedStatus)
	}
	if cfg.TimeoutSeconds != 30 {
		t.Errorf("expected timeout=30, got %d", cfg.TimeoutSeconds)
	}
	if cfg.IntervalSeconds != 2 {
		t.Errorf("expected interval=2, got %d", cfg.IntervalSeconds)
	}
}

func TestHealthCheckConfigEndpointURL(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled: true,
		Scheme:  "http",
		Path:    "/health",
		Port:    8000,
	}
	url := cfg.endpointURL()
	if url != "http://127.0.0.1:8000/health" {
		t.Errorf("expected http://127.0.0.1:8000/health, got %s", url)
	}
}

func TestHealthCheckConfigDisabled(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled: false,
	}
	cfg.resolveDefaults(0)
	if cfg.Enabled {
		t.Error("expected disabled")
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 500, "short"},
		{"", 500, ""},
	}
	for _, tt := range tests {
		got := truncateBody(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateBody(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}

	long := ""
	for i := 0; i < 600; i++ {
		long += "x"
	}
	got := truncateBody(long, 500)
	if len(got) > 500+len("...truncated") {
		t.Errorf("truncateBody(long, 500) too long: %d chars", len(got))
	}
	if got[len(got)-len("...truncated"):] != "...truncated" {
		t.Errorf("truncateBody should end with ...truncated, got %q", got[len(got)-20:])
	}
}

func TestCheckEndpointReadyContainerExitedAbortsEarly(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:         true,
		Scheme:          "http",
		Path:            "/health",
		Port:            19999, // port where nothing listens
		ExpectedStatus:  200,
		TimeoutSeconds:  30,
		IntervalSeconds: 1,
	}
	// Simulate container exited after 2 connection refused.
	callCount := 0
	inspect := func(ctx context.Context) (string, int, error) {
		callCount++
		if callCount >= 1 {
			return "exited", 1, nil
		}
		return "running", 0, nil
	}
	ctx := context.Background()
	err := CheckEndpointReady(ctx, cfg, "inst-1", "cont-1", "test-container", inspect)
	if err == nil {
		t.Error("expected error when container exits during health check")
	}
	if !strings.Contains(err.Error(), "exited") {
		t.Errorf("error should mention container exited, got: %v", err)
	}
}

func TestCheckEndpointReadyNoInspect(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:         true,
		Scheme:          "http",
		Path:            "/health",
		Port:            19998,
		ExpectedStatus:  200,
		TimeoutSeconds:  2,
		IntervalSeconds: 1,
	}
	ctx := context.Background()
	// nil inspect function — should work without crash, just timeout
	err := CheckEndpointReady(ctx, cfg, "inst-2", "cont-2", "test", nil)
	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "connection refused") {
		t.Logf("health check error (expected): %v", err)
	}
}

func TestResolveHealthCheckConfig(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled: true,
		Path:    "/custom",
		Port:    9000,
	}
	result := resolveHealthCheckConfig(cfg, 8080)
	if result.Path != "/custom" {
		t.Errorf("expected custom path preserved, got %s", result.Path)
	}
	if result.Port != 9000 {
		t.Errorf("expected explicit port preserved, got %d", result.Port)
	}
	if result.IntervalSeconds != 2 {
		t.Errorf("expected default interval, got %d", result.IntervalSeconds)
	}

	// nil config returns nil.
	if resolveHealthCheckConfig(nil, 8080) != nil {
		t.Error("expected nil for nil config")
	}
}
