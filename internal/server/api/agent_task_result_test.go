package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"lightai-go/internal/server/db"
)

func insertStartTaskResultFixture(t *testing.T, database *db.DB, suffix string) (taskID, instanceID, deploymentID, runPlanID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	nodeID := "node-task-" + suffix
	deploymentID = "dep-task-" + suffix
	instanceID = "inst-task-" + suffix
	runPlanID = "runplan-task-" + suffix
	taskID = "task-start-" + suffix

	runtimeBoundaryInsertOnlineNode(t, database, nodeID)
	runtimeBoundaryInsertDeployment(t, database, deploymentID)
	if _, err := database.Exec(`INSERT INTO model_instances
		(id, deployment_id, tenant_id, replica_index, node_id, agent_id, current_run_plan_id, actual_state, desired_state, container_id, host_port, created_at, updated_at)
		VALUES (?, ?, '', 0, ?, 'agent-task', ?, 'pending', 'running', '', 8010, ?, ?)`,
		instanceID, deploymentID, nodeID, runPlanID, now, now); err != nil {
		t.Fatalf("insert instance: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO resolved_run_plans
		(id, deployment_id, instance_id, tenant_id, backend_runtime_id, node_backend_runtime_id, plan_json, docker_preview, input_hash, plan_hash, created_at)
		VALUES (?, ?, ?, '', ?, 'nbr-task', '{}', 'docker run test', 'ih', 'ph', ?)`,
		runPlanID, deploymentID, instanceID, "rt-"+deploymentID, now); err != nil {
		t.Fatalf("insert run plan: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO agent_tasks
		(id, task_type, status, tenant_id, deployment_id, instance_id, node_id, payload, timeout_seconds, operation_id, lease_owner, generation, created_at, updated_at)
		VALUES (?, 'model_instance_start', 'in_progress', '', ?, ?, ?, '{}', 300, ?, 'agent-task', 1, ?, ?)`,
		taskID, deploymentID, instanceID, nodeID, "op-"+suffix, now, now); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	return taskID, instanceID, deploymentID, runPlanID
}

func postTaskResult(t *testing.T, h *AgentHandler, taskID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := newReq("POST", "/api/v1/agent/tasks/"+taskID+"/result", body, nil, map[string]string{"id": taskID})
	w := httptest.NewRecorder()
	h.HandleTaskResult(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("task result code=%d body=%s", w.Code, w.Body.String())
	}
	return w
}

func readInstanceState(t *testing.T, database *db.DB, instanceID string) (state, containerID, lastError string) {
	t.Helper()
	if err := database.QueryRow(`SELECT actual_state, COALESCE(container_id,''), COALESCE(last_error,'') FROM model_instances WHERE id=?`, instanceID).Scan(&state, &containerID, &lastError); err != nil {
		t.Fatalf("read instance: %v", err)
	}
	return state, containerID, lastError
}

func auditDetailForAction(t *testing.T, database *db.DB, action, entityID string) string {
	t.Helper()
	var detail string
	if err := database.QueryRow(`SELECT detail FROM audit_logs WHERE action=? AND entity_id=? ORDER BY created_at DESC LIMIT 1`, action, entityID).Scan(&detail); err != nil {
		t.Fatalf("audit %s for %s not found: %v", action, entityID, err)
	}
	return detail
}

func TestHandleTaskResultStartSuccessStoresStateAndAudit(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	taskID, instanceID, deploymentID, runPlanID := insertStartTaskResultFixture(t, database, "success")

	postTaskResult(t, h, taskID, `{
		"status":"completed",
		"success":true,
		"agent_id":"agent-task",
		"instance_id":"`+instanceID+`",
		"deployment_id":"`+deploymentID+`",
		"container_id":"cid-success",
		"runtime_state":"running"
	}`)

	state, containerID, lastErr := readInstanceState(t, database, instanceID)
	if state != "running" {
		t.Fatalf("state=%q want running", state)
	}
	if containerID != "cid-success" {
		t.Fatalf("container_id=%q", containerID)
	}
	if lastErr != "" {
		t.Fatalf("last_error=%q want empty", lastErr)
	}
	detail := auditDetailForAction(t, database, "instance.start.succeeded", instanceID)
	for _, want := range []string{deploymentID, runPlanID, "cid-success", "node-task-success", "agent-task"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("audit detail missing %q: %s", want, detail)
		}
	}
}

func TestHandleTaskResultStartFailureStoresDiagnosticsAndAudit(t *testing.T) {
	for _, tc := range []struct {
		name       string
		reasonCode string
		exitCode   int
	}{
		{name: "container-exited", reasonCode: "container_exited", exitCode: 2},
		{name: "health-check", reasonCode: "health_check_failed", exitCode: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database := setupTestDB(t)
			h := NewAgentHandler(database, nil)
			taskID, instanceID, deploymentID, runPlanID := insertStartTaskResultFixture(t, database, tc.name)

			postTaskResult(t, h, taskID, `{
				"status":"failed",
				"success":false,
				"agent_id":"agent-task",
				"instance_id":"`+instanceID+`",
				"deployment_id":"`+deploymentID+`",
				"container_id":"cid-`+tc.name+`",
				"failure_reason_code":"`+tc.reasonCode+`",
				"exit_code":`+strconv.Itoa(tc.exitCode)+`,
				"stdout_tail_preview":"boot line",
				"stderr_tail_preview":"fatal line",
				"error_message":"start failed"
			}`)

			state, containerID, lastErr := readInstanceState(t, database, instanceID)
			if state != "failed" {
				t.Fatalf("state=%q want failed", state)
			}
			if containerID != "cid-"+tc.name {
				t.Fatalf("container_id=%q", containerID)
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(lastErr), &parsed); err != nil {
				t.Fatalf("last_error is not JSON: %s", lastErr)
			}
			if parsed["failure_reason_code"] != tc.reasonCode {
				t.Fatalf("failure_reason_code=%v want %s in %s", parsed["failure_reason_code"], tc.reasonCode, lastErr)
			}
			if parsed["stdout_tail_preview"] != "boot line" || parsed["stderr_tail_preview"] != "fatal line" {
				t.Fatalf("last_error missing previews: %s", lastErr)
			}
			detail := auditDetailForAction(t, database, "instance.start.failed", instanceID)
			for _, want := range []string{deploymentID, runPlanID, "cid-" + tc.name, tc.reasonCode, "node-task-" + tc.name, "agent-task"} {
				if !strings.Contains(detail, want) {
					t.Fatalf("audit detail missing %q: %s", want, detail)
				}
			}
		})
	}
}

func TestHandleTaskResultStartFailureUsesFallbackDiagnostics(t *testing.T) {
	database := setupTestDB(t)
	h := NewAgentHandler(database, nil)
	taskID, instanceID, _, _ := insertStartTaskResultFixture(t, database, "fallback")

	postTaskResult(t, h, taskID, `{
		"status":"failed",
		"success":false,
		"agent_id":"agent-task",
		"instance_id":"`+instanceID+`",
		"error_message":"docker client unavailable"
	}`)

	_, _, lastErr := readInstanceState(t, database, instanceID)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(lastErr), &parsed); err != nil {
		t.Fatalf("last_error is not JSON: %s", lastErr)
	}
	if parsed["failure_reason_code"] != "task_failed" {
		t.Fatalf("failure_reason_code=%v want task_failed", parsed["failure_reason_code"])
	}
	if parsed["exit_code"].(float64) != -1 {
		t.Fatalf("exit_code=%v want -1", parsed["exit_code"])
	}
}
