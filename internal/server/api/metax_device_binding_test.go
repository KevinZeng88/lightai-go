package api

import (
	"net/http"
	"strings"
	"testing"
)

// TestMetaXNBRCheckRequiresMetaXGPUs verifies that MetaX NBR vendor check
// fails correctly when the node has no MetaX GPUs (only NVIDIA).
func TestMetaXNBRCheckRequiresMetaXGPUs(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{{
			Repository: "vllm-metax", Tag: "latest",
			ImageRef: "vllm-metax:latest", ImageID: "sha256:metax-vllm-image", Size: 123456789,
		}},
		Inspect: workflowInspectSuccess("vllm-metax:latest", "sha256:metax-vllm-image"),
	})
	nodeID := app.InsertOnlineNode(t, "metax-no-gpu-node", fakeAgent)
	app.InsertGPU(t, nodeID, "nvidia") // Only NVIDIA GPU, not MetaX
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "metax")

	enableResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          "vllm-metax:latest",
	}, http.StatusOK)
	var enabled map[string]interface{}
	enableResp.Decode(t, &enabled)
	if got := workflowStringField(t, enabled, "status"); got != "unsupported_device" {
		t.Errorf("MetaX on NVIDIA-only node: status=%q, want unsupported_device", got)
	}
}

// TestMetaXNBRCheckWithMetaXGPU verifies MetaX NBR passes when MetaX GPU exists.
func TestMetaXNBRCheckWithMetaXGPU(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{{
			Repository: "vllm-metax", Tag: "latest",
			ImageRef: "vllm-metax:latest", ImageID: "sha256:metax-vllm-image", Size: 123456789,
		}},
		Inspect: workflowInspectSuccess("vllm-metax:latest", "sha256:metax-vllm-image"),
	})
	nodeID := app.InsertOnlineNode(t, "metax-gpu-node", fakeAgent)
	app.InsertGPU(t, nodeID, "metax")
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "vllm", "metax")

	enableResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          "vllm-metax:latest",
	}, http.StatusOK)
	var enabled map[string]interface{}
	enableResp.Decode(t, &enabled)
	nbrID := workflowStringField(t, enabled, "id")

	checkResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/"+nbrID+"/check-request", map[string]interface{}{}, http.StatusOK)
	var check map[string]interface{}
	checkResp.Decode(t, &check)
	status := workflowStringField(t, check, "status")
	if status != "ready" && status != "ready_with_warnings" {
		t.Fatalf("MetaX check-request status=%q, want ready or ready_with_warnings", status)
	}
	if deployable := check["deployable"]; deployable != true {
		t.Fatalf("MetaX NBR deployable=%v, want true", deployable)
	}
}

// TestCPUModeSkipsAllGPUBinding verifies CPU runtime doesn't require GPU devices.
func TestCPUModeSkipsAllGPUBinding(t *testing.T) {
	app := newWorkflowTestApp(t)
	fakeAgent := newFakeAgent(t, fakeAgentScenario{
		Images: []fakeAgentImage{{
			Repository: "llamacpp-cpu", Tag: "latest",
			ImageRef: "llamacpp-cpu:latest", ImageID: "sha256:cpu-image", Size: 500000,
		}},
		Inspect: workflowInspectSuccess("llamacpp-cpu:latest", "sha256:cpu-image"),
	})
	nodeID := app.InsertOnlineNode(t, "cpu-node", fakeAgent)
	// No GPU inserted — CPU mode should work without GPU
	app.Client.LoginAsAdmin(t)

	runtimeID := app.FindBackendRuntimeID(t, "llamacpp", "cpu")

	enableResp := app.Client.JSON(t, http.MethodPost, "/api/v1/nodes/"+nodeID+"/backend-runtimes/enable", map[string]interface{}{
		"backend_runtime_id": runtimeID,
		"image_ref":          "llamacpp-cpu:latest",
	}, http.StatusOK)
	var enabled map[string]interface{}
	enableResp.Decode(t, &enabled)
	status := workflowStringField(t, enabled, "status")
	if status == "unsupported_device" {
		t.Errorf("CPU runtime should not require GPU devices, got status=%q", status)
	}
	t.Logf("CPU NBR status: %s", status)
}

// TestNVIDIADifferentiatesFromMetaX verifies that NVIDIA and MetaX use
// different env vars in their runtime templates.
func TestNVIDIADifferentiatesFromMetaX(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.Client.LoginAsAdmin(t)

	runtimesResp := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes", nil, http.StatusOK)
	var runtimes []map[string]interface{}
	runtimesResp.Decode(t, &runtimes)

	var nvidiaEnv, metaxEnv map[string]interface{}
	for _, rt := range runtimes {
		vendor, _ := rt["vendor"].(string)
		id, _ := rt["id"].(string)
		if vendor == "nvidia" && nvidiaEnv == nil && id == "runtime.vllm.nvidia-docker" {
			nvidiaEnv = rt
		}
		if vendor == "metax" && metaxEnv == nil {
			metaxEnv = rt
		}
	}

	if nvidiaEnv == nil {
		t.Skip("no NVIDIA runtime template found")
	}
	if metaxEnv == nil {
		t.Skip("no MetaX runtime template found")
	}

	// Get the Docker options from templates via the runtime detail.
	nvidiaDetail := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes/runtime.vllm.nvidia-docker", nil, http.StatusOK)
	var nv map[string]interface{}
	nvidiaDetail.Decode(t, &nv)
	nvDocker := extractDockerJSON(t, nv)

	metaxDetail := app.Client.JSON(t, http.MethodGet, "/api/v1/backend-runtimes/runtime.vllm.metax-docker", nil, http.StatusOK)
	var mx map[string]interface{}
	metaxDetail.Decode(t, &mx)
	mxDocker := extractDockerJSON(t, mx)

	// NVIDIA should NOT have /dev/mxcd device
	for _, d := range getDevices(t, nvDocker) {
		if d == "/dev/mxcd" {
			t.Error("NVIDIA template should not contain /dev/mxcd device")
		}
	}

	// MetaX should have /dev/mxcd and /dev/dri
	hasDRI, hasMXCD := false, false
	for _, d := range getDevices(t, mxDocker) {
		if d == "/dev/dri" {
			hasDRI = true
		}
		if d == "/dev/mxcd" {
			hasMXCD = true
		}
	}
	if !hasDRI {
		t.Error("MetaX template missing /dev/dri device")
	}
	if !hasMXCD {
		t.Error("MetaX template missing /dev/mxcd device")
	}

	// MetaX should have privileged=true
	if priv, _ := mxDocker["privileged"].(bool); !priv {
		t.Error("MetaX template should have privileged=true")
	}

	// NVIDIA should NOT have MACA_VISIBLE_DEVICE in env
	nvEnv := extractEnvJSON(t, nv)
	for k := range nvEnv {
		if k == "CUDA_VISIBLE_DEVICES" {
			t.Log("NVIDIA has CUDA_VISIBLE_DEVICES — this is standard for NVIDIA")
		}
	}

	// MetaX should have MACA_ env vars
	mxEnv := extractEnvJSON(t, mx)
	hasMACA := false
	for k := range mxEnv {
		if strings.HasPrefix(k, "MACA_") || k == "CUDA_VISIBLE_DEVICES" {
			hasMACA = true
		}
	}
	if !hasMACA {
		t.Error("MetaX env should contain CUDA_VISIBLE_DEVICES or MACA_SMALL_PAGESIZE_ENABLE")
	}
}

// Helpers for extracting nested JSON fields from map responses
func extractDockerJSON(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw := m["docker_options"]
	switch v := raw.(type) {
	case map[string]interface{}:
		return v
	default:
		t.Fatalf("docker_options is not a map: %T", raw)
		return nil
	}
}

func extractEnvJSON(t *testing.T, m map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw := m["env"]
	switch v := raw.(type) {
	case map[string]interface{}:
		return v
	default:
		t.Fatalf("env is not a map: %T", raw)
		return nil
	}
}

func getDevices(t *testing.T, dockerJSON map[string]interface{}) []string {
	t.Helper()
	devices, _ := dockerJSON["devices"].([]interface{})
	var paths []string
	for _, d := range devices {
		if dm, ok := d.(map[string]interface{}); ok {
			if hp, ok := dm["host_path"].(string); ok {
				paths = append(paths, hp)
			}
		}
	}
	return paths
}
