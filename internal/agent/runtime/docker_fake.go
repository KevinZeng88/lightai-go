package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// compile-time interface check
var _ DockerClient = (*FakeDockerClient)(nil)

// fakeContainer holds the in-memory state for a single container.
type fakeContainer struct {
	ID             string
	Name           string
	Image          string
	Env            []string
	Args           []string
	Binds          []string
	PortBindings   map[string][]PortBinding
	Privileged     bool
	IPCMode        string
	UTSMode        string
	ShmSize        string
	NetworkMode    string
	GroupAdd       []string
	SecurityOpt    []string
	Ulimits        map[string]string
	RestartPolicy  string
	DeviceRequests []DeviceRequest
	State          string // created, running, exited
	ExitCode       int
	StartedAt      time.Time
	FinishedAt     time.Time
	Logs           strings.Builder
}

// FakeDockerClient implements DockerClient with an in-memory container
// store. It is intended for unit tests only.
type FakeDockerClient struct {
	mu         sync.Mutex
	containers map[string]*fakeContainer // keyed by container ID
	nameIndex  map[string]string         // container name → container ID
}

// NewFakeDockerClient returns a ready-to-use fake Docker client.
func NewFakeDockerClient() *FakeDockerClient {
	return &FakeDockerClient{
		containers: make(map[string]*fakeContainer),
		nameIndex:  make(map[string]string),
	}
}

func (f *FakeDockerClient) ContainerCreate(ctx context.Context, opts ContainerCreateOptions) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := uuid.NewString()[:12]
	name := opts.ContainerName
	if name == "" {
		name = "lightai-" + id
	}

	// Reject duplicate names.
	if _, exists := f.nameIndex[name]; exists {
		return "", fmt.Errorf("container name %q already in use", name)
	}

	c := &fakeContainer{
		ID:             id,
		Name:           name,
		Image:          opts.Image,
		Env:            opts.Env,
		Args:           opts.Command,
		Binds:          opts.Binds,
		PortBindings:   opts.PortBindings,
		Privileged:     opts.Privileged,
		IPCMode:        opts.IPCMode,
		UTSMode:        opts.UTSMode,
		ShmSize:        opts.ShmSize,
		NetworkMode:    opts.NetworkMode,
		GroupAdd:       opts.GroupAdd,
		SecurityOpt:    opts.SecurityOpt,
		Ulimits:        opts.Ulimits,
		RestartPolicy:  opts.RestartPolicy,
		DeviceRequests: opts.DeviceRequests,
		State:          "created",
	}
	f.containers[id] = c
	f.nameIndex[name] = id
	return id, nil
}

func (f *FakeDockerClient) ContainerStart(ctx context.Context, containerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[containerID]
	if !ok {
		return fmt.Errorf("container %s not found", containerID)
	}
	if c.State != "created" {
		return fmt.Errorf("container %s is in state %q, cannot start", containerID, c.State)
	}
	c.State = "running"
	c.StartedAt = time.Now()
	return nil
}

func (f *FakeDockerClient) ContainerStop(ctx context.Context, containerID string, timeoutSeconds int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[containerID]
	if !ok {
		// Docker stop on nonexistent container returns an error.
		return fmt.Errorf("container %s not found", containerID)
	}
	if c.State == "running" {
		c.State = "exited"
		c.ExitCode = 0
		c.FinishedAt = time.Now()
	}
	return nil
}

func (f *FakeDockerClient) ContainerInspect(ctx context.Context, containerID string) (*InspectResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Also support lookup by name.
	c, ok := f.containers[containerID]
	if !ok {
		if cid, exists := f.nameIndex[containerID]; exists {
			c = f.containers[cid]
		}
	}
	if c == nil {
		return nil, fmt.Errorf("container %s not found", containerID)
	}

	startedAt := ""
	if !c.StartedAt.IsZero() {
		startedAt = c.StartedAt.Format(time.RFC3339)
	}
	finishedAt := ""
	if !c.FinishedAt.IsZero() {
		finishedAt = c.FinishedAt.Format(time.RFC3339)
	}

	return &InspectResult{
		ID:         c.ID,
		Name:       c.Name,
		State:      c.State,
		ExitCode:   c.ExitCode,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}, nil
}

func (f *FakeDockerClient) ContainerLogs(ctx context.Context, containerID string, opts LogFetchOptions) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	c, ok := f.containers[containerID]
	if !ok {
		if cid, exists := f.nameIndex[containerID]; exists {
			c = f.containers[cid]
		}
	}
	if c == nil {
		return "", "", fmt.Errorf("container %s not found", containerID)
	}

	// Simulate log appending for running containers.
	if c.State == "running" {
		c.Logs.WriteString(fmt.Sprintf("[%s] container %s running\n", time.Now().Format(time.RFC3339), c.Name))
	}

	// Fake client stores logs in a single buffer; return as stdout.
	return c.Logs.String(), "", nil
}

// SetState allows tests to directly manipulate container state.
func (f *FakeDockerClient) SetState(containerID string, state string, exitCode int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.containers[containerID]; ok {
		c.State = state
		c.ExitCode = exitCode
		if state == "exited" {
			c.FinishedAt = time.Now()
		}
	}
}

// AppendLogs adds log output to a container (for test setup).
func (f *FakeDockerClient) AppendLogs(containerID string, line string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.containers[containerID]; ok {
		c.Logs.WriteString(line)
	}
}

// LastContainer returns the ID of the most recently created container, or
// empty string if none exist. Useful for tests that need to inspect the
// result of a Create call.
func (f *FakeDockerClient) LastContainer() *fakeContainer {
	f.mu.Lock()
	defer f.mu.Unlock()
	var last *fakeContainer
	for _, c := range f.containers {
		last = c
	}
	return last
}

// Count returns the number of tracked containers.
func (f *FakeDockerClient) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.containers)
}
