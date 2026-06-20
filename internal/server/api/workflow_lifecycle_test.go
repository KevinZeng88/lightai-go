package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWorkflowLifecycleStartStatusLogsStop(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.DB.SetMaxOpenConns(1)
	fixture := newWorkflowDeploymentFixture(t, app, "life-success")
	deployment := workflowCreateDeployment(t, app, fixture, "life-success")
	deploymentID := workflowStringField(t, deployment, "id")

	started := workflowStartDeployment(t, app, deploymentID)
	instanceID := workflowStringField(t, started, "instance_id")
	taskID := workflowStringField(t, started, "task_id")
	runPlanID := workflowStringField(t, started, "run_plan_id")
	if started["deployment_id"] != deploymentID {
		t.Fatalf("start deployment_id=%#v want %#v response=%#v", started["deployment_id"], deploymentID, started)
	}

	workflowPostAgentTaskResult(t, app, taskID, map[string]interface{}{
		"status":        "completed",
		"success":       true,
		"agent_id":      "agent-" + fixture.NodeID,
		"operation_id":  "workflow-life-success",
		"instance_id":   instanceID,
		"deployment_id": deploymentID,
		"container_id":  "container-life-success",
		"runtime_state": "running",
	})
	instance := workflowGetModelInstance(t, app, instanceID)
	workflowAssertInstanceState(t, instance, "running", deploymentID, runPlanID, "container-life-success")
	workflowAssertModelInstanceListContains(t, app, deploymentID, instanceID, "running")

	logDone := workflowCompleteNextAgentTask(t, app, "model_instance_logs", map[string]interface{}{
		"status":        "completed",
		"success":       true,
		"agent_id":      "agent-" + fixture.NodeID,
		"instance_id":   instanceID,
		"container_id":  "container-life-success",
		"runtime_state": "running",
		"stdout":        "server ready\nAPI_KEY=must-redact\n",
		"stderr":        "warn line\n",
		"logs":          "server ready\nwarn line\nAPI_KEY=must-redact\n",
	})
	logs := workflowGetRunPlanLogs(t, app, runPlanID)
	<-logDone
	logsText := workflowStringField(t, logs, "logs")
	if !strings.Contains(logsText, "server ready") || !strings.Contains(logsText, "warn line") {
		t.Fatalf("logs missing expected content: %#v", logs)
	}
	if strings.Contains(logsText, "must-redact") {
		t.Fatalf("logs were not redacted: %#v", logs)
	}

	stopDone := workflowCompleteNextAgentTask(t, app, "model_instance_stop", map[string]interface{}{
		"status":       "completed",
		"success":      true,
		"agent_id":     "agent-" + fixture.NodeID,
		"instance_id":  instanceID,
		"container_id": "container-life-success",
	})
	stopped := workflowStopDeployment(t, app, deploymentID, http.StatusOK)
	<-stopDone
	if stopped["status"] != "stopped" {
		t.Fatalf("stop status=%#v response=%#v", stopped["status"], stopped)
	}
	instance = workflowGetModelInstance(t, app, instanceID)
	workflowAssertInstanceState(t, instance, "stopped", deploymentID, runPlanID, "container-life-success")
	workflowAssertAuditAction(t, app, instanceID, "instance.start.succeeded")
	workflowAssertAuditAction(t, app, instanceID, "instance.stop")

	workflowDeleteDeployment(t, app, deploymentID)
	workflowAssertDeploymentDeleted(t, app, deploymentID)
	workflowAssertModelInstanceDeleted(t, app, instanceID)
}

func TestWorkflowLifecycleStartFailureKeepsDiagnosticsAndLogs(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.DB.SetMaxOpenConns(1)
	fixture := newWorkflowDeploymentFixture(t, app, "life-failure")
	deployment := workflowCreateDeployment(t, app, fixture, "life-failure")
	deploymentID := workflowStringField(t, deployment, "id")

	started := workflowStartDeployment(t, app, deploymentID)
	instanceID := workflowStringField(t, started, "instance_id")
	taskID := workflowStringField(t, started, "task_id")
	runPlanID := workflowStringField(t, started, "run_plan_id")

	workflowPostAgentTaskResult(t, app, taskID, map[string]interface{}{
		"status":              "failed",
		"success":             false,
		"agent_id":            "agent-" + fixture.NodeID,
		"operation_id":        "workflow-life-failure",
		"instance_id":         instanceID,
		"deployment_id":       deploymentID,
		"container_id":        "container-life-failure",
		"failure_reason_code": "health_check_failed",
		"exit_code":           float64(2),
		"stdout_tail_preview": "boot line",
		"stderr_tail_preview": "fatal line",
		"error_message":       "health check failed",
	})
	instance := workflowGetModelInstance(t, app, instanceID)
	workflowAssertInstanceState(t, instance, "failed", deploymentID, runPlanID, "container-life-failure")
	lastError := workflowStringField(t, instance, "last_error")
	for _, want := range []string{"health_check_failed", "health check failed", "boot line", "fatal line"} {
		if !strings.Contains(lastError, want) {
			t.Fatalf("last_error missing %q: %s", want, lastError)
		}
	}

	logDone := workflowCompleteNextAgentTask(t, app, "model_instance_logs", map[string]interface{}{
		"status":        "completed",
		"success":       true,
		"agent_id":      "agent-" + fixture.NodeID,
		"instance_id":   instanceID,
		"container_id":  "container-life-failure",
		"runtime_state": "failed",
		"stdout":        "boot line\n",
		"stderr":        "fatal line\n",
		"logs":          "boot line\nfatal line\n",
	})
	logs := workflowGetRunPlanLogs(t, app, runPlanID)
	<-logDone
	if got := workflowStringField(t, logs, "logs"); !strings.Contains(got, "fatal line") {
		t.Fatalf("failed instance logs missing diagnostic line: %#v", logs)
	}
	workflowAssertAuditAction(t, app, instanceID, "instance.start.failed")

	workflowDeleteDeployment(t, app, deploymentID)
	workflowAssertDeploymentDeleted(t, app, deploymentID)
	workflowAssertModelInstanceDeleted(t, app, instanceID)
}

func TestWorkflowLifecycleStopIsIdempotentOrExplained(t *testing.T) {
	app := newWorkflowTestApp(t)
	app.DB.SetMaxOpenConns(1)
	fixture := newWorkflowDeploymentFixture(t, app, "life-stop")
	deployment := workflowCreateDeployment(t, app, fixture, "life-stop")
	deploymentID := workflowStringField(t, deployment, "id")

	started := workflowStartDeployment(t, app, deploymentID)
	instanceID := workflowStringField(t, started, "instance_id")
	taskID := workflowStringField(t, started, "task_id")
	workflowPostAgentTaskResult(t, app, taskID, map[string]interface{}{
		"status":        "completed",
		"success":       true,
		"agent_id":      "agent-" + fixture.NodeID,
		"instance_id":   instanceID,
		"deployment_id": deploymentID,
		"container_id":  "container-life-stop",
	})

	stopDone := workflowCompleteNextAgentTask(t, app, "model_instance_stop", map[string]interface{}{
		"status":       "completed",
		"success":      true,
		"agent_id":     "agent-" + fixture.NodeID,
		"instance_id":  instanceID,
		"container_id": "container-life-stop",
	})
	firstStop := workflowStopDeployment(t, app, deploymentID, http.StatusOK)
	<-stopDone
	if firstStop["status"] != "stopped" {
		t.Fatalf("first stop response=%#v", firstStop)
	}

	secondStop := workflowStopDeployment(t, app, deploymentID, http.StatusOK)
	if secondStop["status"] != "stopped" {
		t.Fatalf("second stop should be idempotent or explicitly stopped: %#v", secondStop)
	}
	if got := int(secondStop["instances_stopped"].(float64)); got != 0 {
		t.Fatalf("second stop instances_stopped=%d want 0 response=%#v", got, secondStop)
	}

	workflowDeleteDeployment(t, app, deploymentID)
	workflowAssertDeploymentDeleted(t, app, deploymentID)
}

func workflowStartDeployment(t *testing.T, app *workflowTestApp, deploymentID string) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/deployments/"+deploymentID+"/start", map[string]interface{}{}, http.StatusOK)
	var out map[string]interface{}
	resp.Decode(t, &out)
	for _, field := range []string{"deployment_id", "instance_id", "task_id", "run_plan_id"} {
		workflowStringField(t, out, field)
	}
	if out["status"] != "started" {
		t.Fatalf("start status=%#v response=%#v", out["status"], out)
	}
	return out
}

func workflowStopDeployment(t *testing.T, app *workflowTestApp, deploymentID string, wantStatus int) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodPost, "/api/v1/deployments/"+deploymentID+"/stop", map[string]interface{}{}, wantStatus)
	var out map[string]interface{}
	resp.Decode(t, &out)
	return out
}

func workflowPostAgentTaskResult(t *testing.T, app *workflowTestApp, taskID string, payload map[string]interface{}) {
	t.Helper()
	app.AgentJSON(t, http.MethodPost, "/api/v1/agent/tasks/"+taskID+"/result", payload, http.StatusOK)
}

func workflowCompleteNextAgentTask(t *testing.T, app *workflowTestApp, taskType string, payload map[string]interface{}) <-chan struct{} {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			var taskID, instanceID string
			err := app.DB.QueryRow(`SELECT id, COALESCE(instance_id,'') FROM agent_tasks WHERE task_type = ? AND status = 'pending' ORDER BY created_at DESC LIMIT 1`, taskType).Scan(&taskID, &instanceID)
			if err == nil {
				if _, ok := payload["instance_id"]; !ok && instanceID != "" {
					payload["instance_id"] = instanceID
				}
				workflowPostAgentTaskResult(t, app, taskID, payload)
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Errorf("timed out waiting for pending %s task", taskType)
	}()
	return done
}

func workflowGetModelInstance(t *testing.T, app *workflowTestApp, instanceID string) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/model-instances/"+instanceID, nil, http.StatusOK)
	var instance map[string]interface{}
	resp.Decode(t, &instance)
	return instance
}

func workflowAssertInstanceState(t *testing.T, instance map[string]interface{}, wantState, deploymentID, runPlanID, containerID string) {
	t.Helper()
	if instance["actual_state"] != wantState {
		t.Fatalf("instance state=%#v want %#v instance=%#v", instance["actual_state"], wantState, instance)
	}
	if instance["deployment_id"] != deploymentID || instance["current_run_plan_id"] != runPlanID {
		t.Fatalf("instance linkage mismatch deployment=%#v/%#v runplan=%#v/%#v instance=%#v",
			instance["deployment_id"], deploymentID, instance["current_run_plan_id"], runPlanID, instance)
	}
	if containerID != "" && instance["container_id"] != containerID {
		t.Fatalf("instance container_id=%#v want %#v instance=%#v", instance["container_id"], containerID, instance)
	}
}

func workflowAssertModelInstanceListContains(t *testing.T, app *workflowTestApp, deploymentID, instanceID, wantState string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/model-instances?deployment_id="+deploymentID, nil, http.StatusOK)
	var instances []map[string]interface{}
	resp.Decode(t, &instances)
	item := workflowFindByID(t, instances, instanceID)
	if item["actual_state"] != wantState {
		t.Fatalf("list instance state=%#v want %#v item=%#v", item["actual_state"], wantState, item)
	}
}

func workflowGetRunPlanLogs(t *testing.T, app *workflowTestApp, runPlanID string) map[string]interface{} {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/node-run-plans/"+runPlanID+"/logs?tail=123", nil, http.StatusOK)
	var logs map[string]interface{}
	resp.Decode(t, &logs)
	if logs["status"] != "ok" {
		t.Fatalf("logs status=%#v response=%#v", logs["status"], logs)
	}
	if logs["id"] != runPlanID {
		t.Fatalf("logs id=%#v want %#v response=%#v", logs["id"], runPlanID, logs)
	}
	return logs
}

func workflowAssertAuditAction(t *testing.T, app *workflowTestApp, instanceID, action string) {
	t.Helper()
	resp := app.Client.JSON(t, http.MethodGet, fmt.Sprintf("/api/v1/audit-logs?action=%s&entity_id=%s&limit=20", action, instanceID), nil, http.StatusOK)
	var audit map[string]interface{}
	resp.Decode(t, &audit)
	entries, ok := audit["entries"].([]interface{})
	if !ok || len(entries) == 0 {
		t.Fatalf("audit action %q for instance %q not found: %#v", action, instanceID, audit)
	}
}

func workflowAssertModelInstanceDeleted(t *testing.T, app *workflowTestApp, instanceID string) {
	t.Helper()
	app.Client.JSON(t, http.MethodGet, "/api/v1/model-instances/"+instanceID, nil, http.StatusNotFound)
	resp := app.Client.JSON(t, http.MethodGet, "/api/v1/model-instances", nil, http.StatusOK)
	var instances []map[string]interface{}
	resp.Decode(t, &instances)
	if workflowListContainsID(instances, instanceID) {
		t.Fatalf("instance %q still visible after cleanup: %#v", instanceID, instances)
	}
}
