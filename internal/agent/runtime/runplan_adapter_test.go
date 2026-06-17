package runtime

import (
	"testing"
)

func TestConvertRunplanToAgentSpec(t *testing.T) {
	input := PlanInput{
		InstanceID:       "inst-000000000001",
		DeploymentID:     "deploy-1",
		NodeID:           "node-1",
		AgentID:          "agent-1",
		ModelPath:        "/data/models/Qwen3-32B",
		ServedModelName:  "qwen3-32b",
		Image:            "vllm/vllm-openai:v0.8.5",
		ContainerName:    "lightai-inst-0001",
		Entrypoint:       []string{"vllm", "serve"},
		Args:             []string{"/models/Qwen3-32B", "--host", "0.0.0.0", "--port", "8000"},
		Env:              map[string]string{"CUDA_VISIBLE_DEVICES": "0,1"},
		Privileged:       true,
		IPCMode:          "host",
		ShmSize:          "10g",
		HostPort:         8001,
		ContainerPort:    8000,
		GPUDeviceIDs:     []string{"0", "1"},
		GPUVisibleEnvKey: "CUDA_VISIBLE_DEVICES",
		Devices: []PlanDevice{
			{HostPath: "/dev/dri", ContainerPath: "/dev/dri"},
		},
		Mounts: []PlanMount{
			{HostPath: "/data/models/Qwen3-32B", ContainerPath: "/models/Qwen3-32B", Readonly: true},
		},
	}

	spec := ConvertRunplanToAgentSpec(input)

	if spec.InstanceID != "inst-000000000001" {
		t.Errorf("instance_id mismatch: %s", spec.InstanceID)
	}
	if spec.RuntimeType != "docker" {
		t.Errorf("runtime_type should be docker: %s", spec.RuntimeType)
	}
	if spec.Docker.Image != "vllm/vllm-openai:v0.8.5" {
		t.Errorf("image mismatch: %s", spec.Docker.Image)
	}
	if !spec.Docker.Privileged {
		t.Error("privileged should be true")
	}
	if spec.Docker.IPCMode != "host" {
		t.Errorf("ipc_mode mismatch: %s", spec.Docker.IPCMode)
	}
	if spec.Docker.ShmSize != "10g" {
		t.Errorf("shm_size mismatch: %s", spec.Docker.ShmSize)
	}
	if spec.HostPort != 8001 {
		t.Errorf("host_port mismatch: %d", spec.HostPort)
	}
	if len(spec.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(spec.Volumes))
	}
	if len(spec.Devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(spec.Devices))
	}
	if len(spec.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(spec.Ports))
	}
	if spec.Env["CUDA_VISIBLE_DEVICES"] != "0,1" {
		t.Errorf("env mismatch: %v", spec.Env)
	}
	if len(spec.Docker.Args) != 5 {
		t.Errorf("expected 5 args, got %d", len(spec.Docker.Args))
	}
}

func TestConvertRunplanToAgentSpecNoGPU(t *testing.T) {
	input := PlanInput{
		InstanceID:   "inst-2",
		DeploymentID: "deploy-2",
		Image:        "vllm/vllm-openai:v0.8.5",
	}
	spec := ConvertRunplanToAgentSpec(input)
	if len(spec.GPUDeviceIDs) != 0 {
		t.Error("expected no GPU devices")
	}
	if len(spec.Ports) != 0 {
		t.Error("expected no ports for host_port=0")
	}
}
