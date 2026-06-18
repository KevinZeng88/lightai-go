package runplan

import (
	"strings"
	"testing"
)

func TestResolveMetaXRunPlanUsesRuntimeDockerOptions(t *testing.T) {
	in := ResolveInput{
		Backend: &BackendInfo{Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			Version:              "openai-latest",
			DefaultEntrypoint:    []string{"vllm", "serve"},
			DefaultArgs:          []string{"{{model_container_path}}", "--host", "0.0.0.0", "--port", "{{container_port}}"},
			HealthCheck:          HealthCheckInput{Path: "/v1/models", ExpectedStatus: 200},
			DefaultContainerPort: 8000,
			DefaultImages:        map[string]string{"metax": "0d307f1665d3"},
		},
		BackendRuntime: &RuntimeInfo{
			ID:          "runtime.vllm.metax-docker",
			Vendor:      "metax",
			RuntimeType: "docker",
			DefaultEnv: map[string]string{
				"CUDA_VISIBLE_DEVICES": "{{vendor_visible_devices}}",
			},
			Docker: DockerSpecInfo{
				Privileged:       true,
				IPCMode:          "host",
				UTSMode:          "host",
				ShmSize:          "100gb",
				GPUVisibleEnvKey: "CUDA_VISIBLE_DEVICES",
				Devices: []DeviceMapping{
					{HostPath: "/dev/dri", ContainerPath: "/dev/dri"},
					{HostPath: "/dev/mxcd", ContainerPath: "/dev/mxcd"},
					{HostPath: "/dev/infiniband", ContainerPath: "/dev/infiniband"},
				},
				GroupAdd:        []string{"video"},
				SecurityOptions: []string{"seccomp=unconfined", "apparmor=unconfined"},
				Ulimits:         map[string]string{"memlock": "-1"},
			},
			ModelMount: ModelMountInfo{ContainerPath: "/models", Readonly: true},
		},
		Artifact: &ArtifactInfo{
			Name:         "Qwen3",
			Path:         "/data/part2/MX-C500/model/Qwen3",
			ModelRoot:    "/data/part2/MX-C500/model",
			RelativePath: "Qwen3",
		},
		Deployment:   &DeploymentInfo{ID: "deploy-metax", Name: "metax", Service: ServiceInfo{HostPort: 8001}},
		InstanceID:   "inst-metax-001",
		Node:         &NodeInfo{ID: "node-metax", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 6, Vendor: "metax"}, {Index: 7, Vendor: "metax"}},
	}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if got := plan.Env["CUDA_VISIBLE_DEVICES"]; got != "6,7" {
		t.Fatalf("CUDA_VISIBLE_DEVICES = %q, want 6,7", got)
	}
	if plan.UTSMode != "host" {
		t.Fatalf("UTSMode = %q, want host", plan.UTSMode)
	}
	if len(plan.GroupAdd) != 1 || plan.GroupAdd[0] != "video" {
		t.Fatalf("GroupAdd = %#v, want [video]", plan.GroupAdd)
	}
	if len(plan.Devices) != 3 {
		t.Fatalf("devices = %#v, want 3 entries", plan.Devices)
	}
	preview := EquivalentCommandPreview(plan)
	for _, want := range []string{
		"--device /dev/dri:/dev/dri",
		"--device /dev/mxcd:/dev/mxcd",
		"--device /dev/infiniband:/dev/infiniband",
		"--group-add video",
		"--uts host",
		"--ipc host",
		"--privileged",
		"--security-opt seccomp=unconfined",
		"--security-opt apparmor=unconfined",
		"--shm-size 100gb",
		"--ulimit memlock=-1",
		"-v /data/part2/MX-C500/model/Qwen3:/models/Qwen3:ro",
		"-e CUDA_VISIBLE_DEVICES=6,7",
	} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q:\n%s", want, preview)
		}
	}
	if strings.Contains(preview, "/dev/mem") {
		t.Fatalf("preview must not include optional high-risk /dev/mem by default:\n%s", preview)
	}
}

func TestResolveHuaweiRunPlanUsesAscendVisibleDevices(t *testing.T) {
	in := ResolveInput{
		Backend: &BackendInfo{Name: "vllm", DefaultEnv: map[string]string{}},
		BackendVersion: &VersionInfo{
			Version:              "openai-latest",
			DefaultArgs:          []string{"{{model_container_path}}"},
			DefaultContainerPort: 8000,
			DefaultImages:        map[string]string{"huawei": "template-only"},
		},
		BackendRuntime: &RuntimeInfo{
			ID:          "runtime.vllm.huawei-docker",
			Vendor:      "huawei",
			RuntimeType: "docker",
			DefaultEnv: map[string]string{
				"ASCEND_VISIBLE_DEVICES": "{{vendor_visible_devices}}",
			},
			Docker: DockerSpecInfo{
				GPUVisibleEnvKey: "ASCEND_VISIBLE_DEVICES",
				Devices: []DeviceMapping{
					{HostPath: "/dev/davinci_manager", ContainerPath: "/dev/davinci_manager"},
					{HostPath: "/dev/devmm_svm", ContainerPath: "/dev/devmm_svm"},
					{HostPath: "/dev/hisi_hdc", ContainerPath: "/dev/hisi_hdc"},
				},
			},
			ModelMount: ModelMountInfo{ContainerPath: "/models", Readonly: true},
		},
		Artifact:     &ArtifactInfo{Name: "Qwen3", Path: "/data/models/Qwen3"},
		Deployment:   &DeploymentInfo{ID: "deploy-huawei", Name: "huawei", Service: ServiceInfo{HostPort: 8005}},
		InstanceID:   "inst-huawei-001",
		Node:         &NodeInfo{ID: "node-huawei", IP: "127.0.0.1"},
		AssignedGPUs: []GPUInfo{{Index: 0, Vendor: "huawei"}},
	}

	plan, errs, _ := Resolve(in)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if got := plan.GPUVisibleEnvKey; got != "ASCEND_VISIBLE_DEVICES" {
		t.Fatalf("GPUVisibleEnvKey = %q, want ASCEND_VISIBLE_DEVICES", got)
	}
	if got := plan.Env["ASCEND_VISIBLE_DEVICES"]; got != "0" {
		t.Fatalf("ASCEND_VISIBLE_DEVICES = %q, want 0", got)
	}
}
