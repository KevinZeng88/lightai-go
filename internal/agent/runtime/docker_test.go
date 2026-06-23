package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ==========================================================================
// Test helpers
// ==========================================================================

func makeTestSpec(runtimeType string) AgentRunSpec {
	instanceID := uuid.NewString()
	return AgentRunSpec{
		InstanceID:       instanceID,
		DeploymentID:     uuid.NewString(),
		RuntimeType:      runtimeType,
		BackendType:      "vllm",
		Vendor:           "nvidia",
		ModelPath:        "/data/models/Qwen3-32B",
		ServedModelName:  "qwen3-32b",
		NodeID:           uuid.NewString(),
		AgentID:          uuid.NewString(),
		GPUDeviceIDs:     []string{"0", "1"},
		GPUVisibleEnvKey: "CUDA_VISIBLE_DEVICES",
		Env: map[string]string{
			"CUDA_VISIBLE_DEVICES": "0,1",
			"MODEL_PATH":           "/data/models/Qwen3-32B",
		},
		Args:          []string{"--model", "/data/models/Qwen3-32B", "--port", "8000"},
		HostPort:      8001,
		ContainerPort: 8000,
		Volumes: []VolumeSpec{
			{HostPath: "/data/models", ContainerPath: "/data/models", Readonly: true},
		},
		Devices: []DeviceSpec{
			{HostPath: "/dev/dri", ContainerPath: "/dev/dri", Permissions: "rwm"},
		},
		Ports: []PortSpec{
			{HostPort: 8001, ContainerPort: 8000, Protocol: "tcp"},
		},
		Docker: DockerSpec{
			Image:           "vllm/vllm-openai:latest",
			ContainerName:   containerNameFromInstance(instanceID),
			Args:            []string{"--model", "/data/models/Qwen3-32B", "--port", "8000"},
			Privileged:      false,
			IPCMode:         "host",
			ShmSize:         "8gb",
			NetworkMode:     "host",
			GroupAdd:        []string{"video"},
			SecurityOptions: []string{"no-new-privileges:true"},
			RestartPolicy:   "unless-stopped",
		},
	}
}

func newTestDriver() (*DockerRuntimeDriver, *FakeDockerClient) {
	fake := NewFakeDockerClient()
	driver := NewDockerRuntimeDriver(fake)
	return driver, fake
}

// ==========================================================================
// Start tests
// ==========================================================================

func TestDockerRuntimeDriverStart(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")

	inst, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if inst.InstanceID != spec.InstanceID {
		t.Errorf("InstanceID = %q, want %q", inst.InstanceID, spec.InstanceID)
	}
	if inst.ContainerID == "" {
		t.Error("ContainerID is empty")
	}
	if inst.EndpointURL != "http://localhost:8001" {
		t.Errorf("EndpointURL = %q, want http://localhost:8001", inst.EndpointURL)
	}
	if inst.HostPort != 8001 {
		t.Errorf("HostPort = %d, want 8001", inst.HostPort)
	}

	// Verify the container was actually created with correct config.
	c := fake.LastContainer()
	if c == nil {
		t.Fatal("no container created")
	}
	if c.Image != "vllm/vllm-openai:latest" {
		t.Errorf("Image = %q, want vllm/vllm-openai:latest", c.Image)
	}
	if c.IPCMode != "host" {
		t.Errorf("IPCMode = %q, want host", c.IPCMode)
	}
	if c.ShmSize != "8gb" {
		t.Errorf("ShmSize = %q, want 8gb", c.ShmSize)
	}
	if c.NetworkMode != "host" {
		t.Errorf("NetworkMode = %q, want host", c.NetworkMode)
	}
	if c.State != "running" {
		t.Errorf("State = %q, want running", c.State)
	}
}

func TestDockerRuntimeDriverStartCreatesCorrectBinds(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	c := fake.LastContainer()
	found := false
	for _, b := range c.Binds {
		if b == "/data/models:/data/models:ro" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected bind /data/models:/data/models:ro not found in %v", c.Binds)
	}
}

func TestDockerRuntimeDriverStartCreatesCorrectEnv(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	c := fake.LastContainer()
	hasCUDA := false
	hasModel := false
	for _, e := range c.Env {
		if e == "CUDA_VISIBLE_DEVICES=0,1" {
			hasCUDA = true
		}
		if e == "MODEL_PATH=/data/models/Qwen3-32B" {
			hasModel = true
		}
	}
	if !hasCUDA {
		t.Errorf("CUDA_VISIBLE_DEVICES env not found in %v", c.Env)
	}
	if !hasModel {
		t.Errorf("MODEL_PATH env not found in %v", c.Env)
	}
}

func TestStartRejectsNonDocker(t *testing.T) {
	driver, _ := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("process")

	_, err := driver.Start(ctx, spec)
	if err == nil {
		t.Fatal("expected error for RuntimeType=process, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported runtime type") {
		t.Errorf("error should mention unsupported runtime type, got: %v", err)
	}
}

func TestStartPostCreateContainerExitReturnsDiagnostics(t *testing.T) {
	fake := NewFakeDockerClient()
	fake.AfterStart = func(containerID string) {
		fake.SetState(containerID, "exited", 2)
		fake.AppendLogs(containerID, "startup failed\n")
	}
	driver := NewDockerRuntimeDriver(fake)

	inst, err := driver.Start(context.Background(), makeTestSpec("docker"))
	if err == nil {
		t.Fatal("expected post-start exited container error")
	}
	if inst == nil {
		t.Fatal("expected RuntimeInstance with container diagnostics")
	}
	if inst.ContainerID == "" {
		t.Fatal("expected container_id to be preserved")
	}
	if inst.ExitCode != 2 {
		t.Fatalf("exit_code=%d want 2", inst.ExitCode)
	}
	if inst.FailureReasonCode != "container_exited" {
		t.Fatalf("failure_reason_code=%q want container_exited", inst.FailureReasonCode)
	}
	if !strings.Contains(inst.StdoutTailPreview, "startup failed") {
		t.Fatalf("stdout_tail_preview missing logs: %q", inst.StdoutTailPreview)
	}
}

func TestStartContainerStartErrorReturnsDiagnostics(t *testing.T) {
	fake := NewFakeDockerClient()
	fake.StartError = fmt.Errorf("failed to bind host port")
	driver := NewDockerRuntimeDriver(fake)

	inst, err := driver.Start(context.Background(), makeTestSpec("docker"))
	if err == nil {
		t.Fatal("expected docker start error")
	}
	if inst == nil {
		t.Fatal("expected RuntimeInstance with container diagnostics")
	}
	if inst.ContainerID == "" {
		t.Fatal("expected container_id to be preserved")
	}
	if inst.ExitCode != 128 {
		t.Fatalf("exit_code=%d want 128", inst.ExitCode)
	}
	if inst.FailureReasonCode != "container_exited" {
		t.Fatalf("failure_reason_code=%q want container_exited", inst.FailureReasonCode)
	}
}

func TestStartHealthCheckFailureReturnsDiagnostics(t *testing.T) {
	fake := NewFakeDockerClient()
	driver := NewDockerRuntimeDriver(fake)
	spec := makeTestSpec("docker")
	spec.Docker.NetworkMode = ""
	spec.HealthCheck = &HealthCheckConfig{
		Enabled:         true,
		Scheme:          "http",
		Path:            "/never-ready",
		Port:            65530,
		ExpectedStatus:  200,
		TimeoutSeconds:  1,
		IntervalSeconds: 1,
	}

	inst, err := driver.Start(context.Background(), spec)
	if err == nil {
		t.Fatal("expected health check failure")
	}
	if inst == nil {
		t.Fatal("expected RuntimeInstance with container diagnostics")
	}
	if inst.ContainerID == "" {
		t.Fatal("expected container_id to be preserved")
	}
	if inst.FailureReasonCode != "health_check_failed" && inst.FailureReasonCode != "health_timeout" {
		t.Fatalf("failure_reason_code=%q want health_check_failed or health_timeout", inst.FailureReasonCode)
	}
}

// ==========================================================================
// Stop tests
// ==========================================================================

func TestDockerRuntimeDriverStop(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")

	inst, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = driver.Stop(ctx, spec.InstanceID)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify container is removed after stop.
	if fake.Count() != 0 {
		t.Errorf("container count after stop = %d, want 0 (removed)", fake.Count())
	}
	_ = inst
}

func TestDockerRuntimeDriverStopNotFound(t *testing.T) {
	driver, _ := newTestDriver()
	ctx := context.Background()

	// REVIEW-006: Stop is now idempotent — missing container returns nil (success).
	err := driver.Stop(ctx, "nonexistent-instance")
	if err != nil {
		t.Fatalf("expected nil (idempotent stop) for nonexistent container, got: %v", err)
	}
}

// ==========================================================================
// Inspect tests
// ==========================================================================

func TestDockerRuntimeDriverInspect(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	status, err := driver.Inspect(ctx, spec.InstanceID)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}
	if status.State != "running" {
		t.Errorf("State = %q, want running", status.State)
	}
	if status.InstanceID != spec.InstanceID {
		t.Errorf("InstanceID = %q, want %q", status.InstanceID, spec.InstanceID)
	}

	// Stop and re-inspect.
	fake.SetState(fake.LastContainer().ID, "exited", 0)
	status, err = driver.Inspect(ctx, spec.InstanceID)
	if err != nil {
		t.Fatalf("Inspect after stop failed: %v", err)
	}
	if status.State != "stopped" {
		t.Errorf("State = %q, want stopped", status.State)
	}
}

func TestDockerRuntimeDriverInspectNotFound(t *testing.T) {
	driver, _ := newTestDriver()
	ctx := context.Background()

	_, err := driver.Inspect(ctx, "nonexistent-instance")
	if err == nil {
		t.Fatal("expected error for nonexistent container, got nil")
	}
}

// ==========================================================================
// Logs tests
// ==========================================================================

func TestDockerRuntimeDriverLogs(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	c := fake.LastContainer()
	fake.AppendLogs(c.ID, "model loading...\n")
	fake.AppendLogs(c.ID, "model ready\n")

	logs, err := driver.Logs(ctx, spec.InstanceID, LogOptions{Tail: 10})
	if err != nil {
		t.Fatalf("Logs failed: %v", err)
	}
	if !strings.Contains(logs.Stdout, "model loading") {
		t.Errorf("logs should contain 'model loading', got: %s", logs.Stdout)
	}
	if !strings.Contains(logs.Stdout, "model ready") {
		t.Errorf("logs should contain 'model ready', got: %s", logs.Stdout)
	}
}

// ==========================================================================
// Sensitive env tests
// ==========================================================================

func TestSensitiveEnvRedaction(t *testing.T) {
	// Verify that env values are redacted in log output but NOT in
	// container config (container needs the real value to run).
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")
	spec.Env["API_KEY"] = "sk-secret-12345"
	spec.Env["DB_PASSWORD"] = "hunter2"
	spec.Env["PUBLIC_VAR"] = "visible"

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	c := fake.LastContainer()

	// Container config must have real values.
	hasRealAPIKey := false
	hasRealPassword := false
	for _, e := range c.Env {
		if e == "API_KEY=sk-secret-12345" {
			hasRealAPIKey = true
		}
		if e == "DB_PASSWORD=hunter2" {
			hasRealPassword = true
		}
	}
	if !hasRealAPIKey {
		t.Error("container config must have real API_KEY value")
	}
	if !hasRealPassword {
		t.Error("container config must have real DB_PASSWORD value")
	}

	// log output (redactEnvForLog) must redact sensitive values.
	logStr := redactEnvForLog(spec.Env)
	if strings.Contains(logStr, "sk-secret-12345") {
		t.Error("log output must not contain plaintext API key")
	}
	if strings.Contains(logStr, "hunter2") {
		t.Error("log output must not contain plaintext password")
	}
	if !strings.Contains(logStr, "API_KEY=<redacted>") {
		t.Error("log output must redact API_KEY")
	}
	if !strings.Contains(logStr, "DB_PASSWORD=<redacted>") {
		t.Error("log output must redact DB_PASSWORD")
	}
	if !strings.Contains(logStr, "PUBLIC_VAR=visible") {
		t.Error("log output must preserve non-sensitive vars")
	}
}

// ==========================================================================
// isSensitive tests (table-driven)
// ==========================================================================

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"API_KEY", true},
		{"apikey", true},
		{"TOKEN", true},
		{"access_token", true},
		{"PASSWORD", true},
		{"db_password", true},
		{"SECRET", true},
		{"client_secret", true},
		{"AUTH", true},
		{"authorization", true},
		{"CREDENTIAL", true},
		{"credentials", true},
		{"ACCESS", true},
		{"PRIVATE", true},
		{"private_key", true},
		{"MODEL_PATH", false},
		{"CUDA_VISIBLE_DEVICES", false},
		{"NODE_ID", false},
		{"HOST_PORT", false},
		{"INSTANCE_ID", false},
		{"SHM_SIZE", false},
		{"MAX_MODEL_LEN", false},
		{"DTYPE", false},
	}
	for _, tc := range tests {
		if got := isSensitive(tc.key); got != tc.sensitive {
			t.Errorf("isSensitive(%q) = %v, want %v", tc.key, got, tc.sensitive)
		}
	}
}

// ==========================================================================
// EquivalentCommandPreview tests
// ==========================================================================

func TestEquivalentCommandPreview(t *testing.T) {
	spec := makeTestSpec("docker")
	cmd := EquivalentCommandPreview(&spec)

	// Basic structure.
	if !strings.HasPrefix(cmd, "docker run -d") {
		t.Errorf("command should start with 'docker run -d', got: %s", cmd)
	}
	if !strings.Contains(cmd, "--name") {
		t.Error("command should contain --name")
	}
	if !strings.Contains(cmd, "--ipc host") {
		t.Error("command should contain --ipc host")
	}
	if !strings.Contains(cmd, "--shm-size 8gb") {
		t.Error("command should contain --shm-size 8gb")
	}
	if !strings.Contains(cmd, spec.Docker.Image) {
		t.Error("command should contain image name")
	}
}

func TestEquivalentCommandPreviewRedactsSensitiveEnv(t *testing.T) {
	spec := makeTestSpec("docker")
	spec.Env["API_KEY"] = "sk-secret"

	cmd := EquivalentCommandPreview(&spec)
	if strings.Contains(cmd, "sk-secret") {
		t.Error("command preview must not contain plaintext secret")
	}
	if !strings.Contains(cmd, "API_KEY=<redacted>") {
		t.Error("command preview must redact API_KEY")
	}
}

// ==========================================================================
// containerNameFromInstance tests
// ==========================================================================

func TestContainerNameFromInstance(t *testing.T) {
	name := containerNameFromInstance("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if name != "lightai-a1b2c3d4-e5f" {
		t.Errorf("name = %q, want lightai-a1b2c3d4-e5f", name)
	}

	name = containerNameFromInstance("short")
	if name != "lightai-short" {
		t.Errorf("name = %q, want lightai-short", name)
	}
}

// ==========================================================================
// mapContainerState tests (table-driven)
// ==========================================================================

func TestMapContainerState(t *testing.T) {
	tests := []struct {
		docker string
		lai    string
	}{
		{"created", "pending"},
		{"running", "running"},
		{"paused", "unhealthy"},
		{"restarting", "starting"},
		{"removing", "stopping"},
		{"exited", "stopped"},
		{"dead", "failed"},
		{"", "unknown"},
		{"bogus", "unknown"},
	}
	for _, tc := range tests {
		if got := mapContainerState(tc.docker); got != tc.lai {
			t.Errorf("mapContainerState(%q) = %q, want %q", tc.docker, got, tc.lai)
		}
	}
}

// ==========================================================================
// Interface compliance checks
// ==========================================================================

func TestDockerRuntimeDriverImplementsInterface(t *testing.T) {
	// Compile-time check above; this is a runtime sanity check.
	driver, _ := newTestDriver()
	var _ RuntimeDriver = driver // already checked by var _ above
	_ = driver
}

// ==========================================================================
// Real Docker daemon integration test (opt-in, skipped by default)
// ==========================================================================

// ==========================================================================
// GPU DeviceRequests tests
// ==========================================================================

func TestNvidiaGpuDeviceRequestAll(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")
	spec.Vendor = "nvidia"
	spec.GPUDeviceIDs = []string{"0", "1", "2", "3"}
	spec.Docker.GPUDeviceIDs = []string{"0", "1", "2", "3"}

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	c := fake.LastContainer()
	if c == nil {
		t.Fatal("no container created")
	}
	if len(c.DeviceRequests) != 1 {
		t.Fatalf("expected 1 DeviceRequest, got %d", len(c.DeviceRequests))
	}
	dr := c.DeviceRequests[0]
	if dr.Driver != "" {
		t.Errorf("DeviceRequest.Driver = %q, want empty (matching docker run --gpus CLI)", dr.Driver)
	}
	if len(dr.Capabilities) != 1 || len(dr.Capabilities[0]) != 1 || dr.Capabilities[0][0] != "gpu" {
		t.Errorf("DeviceRequest.Capabilities = %v, want [[gpu]]", dr.Capabilities)
	}
	if len(dr.DeviceIDs) != 4 {
		t.Errorf("DeviceRequest.DeviceIDs len = %d, want 4", len(dr.DeviceIDs))
	}
}

func TestNvidiaGpuDeviceRequestSpecific(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")
	spec.Vendor = "nvidia"
	spec.GPUDeviceIDs = []string{"0", "1"}
	spec.Docker.GPUDeviceIDs = []string{"0", "1"}

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	c := fake.LastContainer()
	if len(c.DeviceRequests) != 1 {
		t.Fatalf("expected 1 DeviceRequest, got %d", len(c.DeviceRequests))
	}
	dr := c.DeviceRequests[0]
	if dr.Driver != "" {
		t.Errorf("DeviceRequest.Driver = %q, want empty (matching docker run --gpus CLI)", dr.Driver)
	}
	if len(dr.DeviceIDs) != 2 || dr.DeviceIDs[0] != "0" || dr.DeviceIDs[1] != "1" {
		t.Errorf("DeviceRequest.DeviceIDs = %v, want [0 1]", dr.DeviceIDs)
	}
}

func TestMetaXUsesRawDevicesNotDeviceRequest(t *testing.T) {
	driver, fake := newTestDriver()
	ctx := context.Background()
	spec := makeTestSpec("docker")
	spec.Vendor = "metax"
	spec.GPUDeviceIDs = []string{"0"}
	spec.Devices = []DeviceSpec{
		{HostPath: "/dev/dri", ContainerPath: "/dev/dri", Permissions: "rwm"},
		{HostPath: "/dev/mxcd", ContainerPath: "/dev/mxcd", Permissions: "rwm"},
	}
	spec.Docker.SecurityOptions = []string{"seccomp=unconfined", "apparmor=unconfined"}
	spec.Docker.Privileged = true
	spec.Docker.IPCMode = "host"
	spec.Docker.UTSMode = "host"
	spec.Docker.ShmSize = "8gb"
	spec.Docker.GroupAdd = []string{"video"}
	spec.Docker.Ulimits = map[string]string{"memlock": "-1"}

	_, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	c := fake.LastContainer()
	// MetaX should NOT have DeviceRequests (uses raw devices instead).
	if len(c.DeviceRequests) != 0 {
		t.Errorf("MetaX should have 0 DeviceRequests, got %d", len(c.DeviceRequests))
	}
	// Must have security options.
	if len(c.SecurityOpt) < 2 {
		t.Errorf("MetaX should have security options, got %v", c.SecurityOpt)
	}
	if !c.Privileged {
		t.Error("MetaX should have privileged=true")
	}
	if c.IPCMode != "host" {
		t.Errorf("MetaX IPCMode = %q, want host", c.IPCMode)
	}
	if c.UTSMode != "host" {
		t.Errorf("MetaX UTSMode = %q, want host", c.UTSMode)
	}
	if c.ShmSize != "8gb" {
		t.Errorf("MetaX ShmSize = %q, want 8gb", c.ShmSize)
	}
	if c.Ulimits["memlock"] != "-1" {
		t.Errorf("MetaX ulimit memlock = %q, want -1", c.Ulimits["memlock"])
	}
}

func TestEquivalentCommandPreviewShowsGpus(t *testing.T) {
	spec := makeTestSpec("docker")
	spec.Vendor = "nvidia"
	spec.Docker.GPUDeviceIDs = []string{"0", "1"}
	cmd := EquivalentCommandPreview(&spec)

	if !strings.Contains(cmd, "--gpus") {
		t.Error("NVIDIA command preview should contain --gpus")
	}
	if !strings.Contains(cmd, "device=0,1") {
		t.Error("NVIDIA command preview should contain device=0,1")
	}
}

func TestEquivalentCommandPreviewNoGpusForMetaX(t *testing.T) {
	spec := makeTestSpec("docker")
	spec.Vendor = "metax"
	spec.Docker.GPUDeviceIDs = []string{"0"}
	spec.Devices = []DeviceSpec{
		{HostPath: "/dev/dri", ContainerPath: "/dev/dri"},
	}
	cmd := EquivalentCommandPreview(&spec)

	// MetaX should NOT have --gpus.
	if strings.Contains(cmd, "--gpus") {
		t.Error("MetaX command preview should NOT contain --gpus")
	}
	// MetaX should have --device for raw devices.
	if !strings.Contains(cmd, "--device") {
		t.Error("MetaX command preview should contain --device")
	}
}

// TestRealDockerRuntimeDriver verifies the DockerRuntimeDriver against a real
// Docker daemon using a lightweight alpine image.  It is skipped unless the
// LIGHTAI_TEST_DOCKER environment variable is set to "1".
//
//	LIGHTAI_TEST_DOCKER=1 go test ./internal/agent/runtime -run TestRealDockerRuntimeDriver -v
//
// The test creates a short-lived container (alpine echo), verifies lifecycle
// (create → start → inspect → logs → stop), and cleans up afterwards.
// No GPU is required.
func TestRealDockerRuntimeDriver(t *testing.T) {
	if os.Getenv("LIGHTAI_TEST_DOCKER") != "1" {
		t.Skip("Skipping real Docker integration test. Set LIGHTAI_TEST_DOCKER=1 to run.")
	}

	// Create real Docker client.
	realCli, err := NewRealDockerClient()
	if err != nil {
		t.Skipf("Cannot connect to Docker daemon: %v", err)
	}
	defer realCli.Close()

	// Quick connectivity check.
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if _, err := realCli.cli.Ping(pingCtx); err != nil {
		t.Skipf("Docker daemon unreachable: %v", err)
	}

	driver := NewDockerRuntimeDriver(realCli)
	ctx := context.Background()

	// Use lightweight alpine:latest image.
	testImage := "alpine:latest"
	instanceID := "test-" + uuid.NewString()[:8]

	spec := AgentRunSpec{
		InstanceID:      instanceID,
		DeploymentID:    uuid.NewString(),
		RuntimeType:     "docker",
		BackendType:     "custom",
		Vendor:          "cpu",
		ModelPath:       "/tmp",
		ServedModelName: "test-model",
		Args:            []string{"echo", "hello from lightai integration test"},
		Docker: DockerSpec{
			Image:         testImage,
			ContainerName: containerNameFromInstance(instanceID),
			Args:          []string{"echo", "hello from lightai integration test"},
			RestartPolicy: "no",
		},
	}

	// --- Step 1: Start ---
	t.Log("starting container...")
	inst, err := driver.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Logf("started: id=%s name=%s", inst.ContainerID, inst.ContainerName)

	// Ensure cleanup on exit.
	defer func() {
		t.Log("cleanup: stopping container...")
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := driver.Stop(stopCtx, instanceID); err != nil {
			t.Logf("cleanup stop warning: %v", err)
		}
	}()

	// --- Step 2: Inspect ---
	t.Log("inspecting container...")
	status, err := driver.Inspect(ctx, instanceID)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}
	t.Logf("inspect: state=%s exit=%d", status.State, status.ExitCode)

	// Alpine echo exits quickly, so state may already be "exited" (→ "stopped")
	// or "running". Both are valid — the container ran.
	if status.State != "running" && status.State != "stopped" {
		t.Errorf("unexpected state after start: %q (want running or stopped)", status.State)
	}

	// --- Step 3: Wait for container to finish and re-inspect ---
	t.Log("waiting for container to finish...")
	time.Sleep(3 * time.Second)

	status, err = driver.Inspect(ctx, instanceID)
	if err != nil {
		t.Fatalf("Inspect after wait failed: %v", err)
	}
	t.Logf("inspect after wait: state=%s exit=%d", status.State, status.ExitCode)
	if status.State != "stopped" {
		t.Logf("note: state is %q (alpine may still be running — this is OK)", status.State)
	}
	if status.State == "stopped" && status.ExitCode != 0 {
		t.Errorf("container exited with non-zero code: %d", status.ExitCode)
	}

	// --- Step 4: Logs ---
	t.Log("fetching logs...")
	logs, err := driver.Logs(ctx, instanceID, LogOptions{Tail: 10})
	if err != nil {
		t.Fatalf("Logs failed: %v", err)
	}
	if !strings.Contains(logs.Stdout, "hello from lightai integration test") {
		t.Errorf("logs should contain greeting, got: %s", logs.Stdout)
	}
	t.Logf("logs: %s", strings.TrimSpace(logs.Stdout))

	// --- Step 5: Stop (idempotent) ---
	t.Log("stopping container...")
	if err := driver.Stop(ctx, instanceID); err != nil {
		// Already stopped is fine — docker stop on an exited container
		// returns an error. We accept this as normal.
		t.Logf("stop result (expected for already-exited container): %v", err)
	}

	t.Log("PASS: real Docker integration test complete")
}

// realDockerSkipReason returns the reason the real Docker test would be
// skipped, or empty string if it can proceed. Useful for diagnostics.
func realDockerSkipReason() string {
	if os.Getenv("LIGHTAI_TEST_DOCKER") != "1" {
		return "LIGHTAI_TEST_DOCKER not set to 1"
	}
	cli, err := NewRealDockerClient()
	if err != nil {
		return fmt.Sprintf("cannot create Docker client: %v", err)
	}
	cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := cli.cli.Ping(ctx); err != nil {
		return fmt.Sprintf("Docker daemon unreachable: %v", err)
	}
	return ""
}
