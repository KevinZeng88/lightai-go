package main

import (
	"errors"
	"testing"

	"lightai-go/internal/agent/register"
	agentruntime "lightai-go/internal/agent/runtime"
)

func TestApplyStartFailureDiagnosticsPreservesContainerInfo(t *testing.T) {
	result := &register.TaskResult{}
	inst := &agentruntime.RuntimeInstance{
		ContainerID:       "cid-start-failed",
		ExitCode:          2,
		FailureReasonCode: "container_exited",
		StdoutTailPreview: "boot line",
		StderrTailPreview: "fatal line",
	}

	applyStartFailureDiagnostics(result, inst, errors.New("container exited"))

	if result.Success {
		t.Fatal("failed start result should not be success")
	}
	if result.Status != "failed" {
		t.Fatalf("status=%q want failed", result.Status)
	}
	if result.ContainerID != "cid-start-failed" {
		t.Fatalf("container_id=%q", result.ContainerID)
	}
	if result.ExitCode != 2 {
		t.Fatalf("exit_code=%d want 2", result.ExitCode)
	}
	if result.FailureReasonCode != "container_exited" {
		t.Fatalf("failure_reason_code=%q", result.FailureReasonCode)
	}
	if result.StdoutTailPreview != "boot line" {
		t.Fatalf("stdout_tail_preview=%q", result.StdoutTailPreview)
	}
	if result.StderrTailPreview != "fatal line" {
		t.Fatalf("stderr_tail_preview=%q", result.StderrTailPreview)
	}
}

func TestApplyStartFailureDiagnosticsUsesTaskFailedFallback(t *testing.T) {
	result := &register.TaskResult{}

	applyStartFailureDiagnostics(result, nil, errors.New("docker client unavailable"))

	if result.Status != "failed" {
		t.Fatalf("status=%q want failed", result.Status)
	}
	if result.FailureReasonCode != "task_failed" {
		t.Fatalf("failure_reason_code=%q want task_failed", result.FailureReasonCode)
	}
	if result.ExitCode != -1 {
		t.Fatalf("exit_code=%d want -1", result.ExitCode)
	}
}

func TestApplyDefaultTaskResultStatus(t *testing.T) {
	success := &register.TaskResult{Success: true}
	applyDefaultTaskResultStatus(success)
	if success.Status != "completed" {
		t.Fatalf("success status=%q want completed", success.Status)
	}

	failure := &register.TaskResult{Success: false}
	applyDefaultTaskResultStatus(failure)
	if failure.Status != "failed" {
		t.Fatalf("failure status=%q want failed", failure.Status)
	}

	explicit := &register.TaskResult{Success: true, Status: "ok"}
	applyDefaultTaskResultStatus(explicit)
	if explicit.Status != "ok" {
		t.Fatalf("explicit status overwritten: %q", explicit.Status)
	}
}
