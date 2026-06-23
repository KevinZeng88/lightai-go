package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func insertRunPlanLogsFixture(t *testing.T, status string) (*AgentHandler, string) {
	t.Helper()
	database := setupTestDB(t)
	database.SetMaxOpenConns(1)
	tid := database.DefaultTenantID()
	h := NewAgentHandler(database, nil)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		VALUES ('node-logs','agent-logs','host-logs',?,?,?,?,?)`, status, tid, now, now, now)
	if err != nil {
		t.Fatalf("insert node: %v", err)
	}
	_, err = database.Exec(`INSERT INTO model_artifacts (id, name, display_name, format, task_type, tenant_id, created_at, updated_at)
		VALUES ('artifact-logs','artifact-logs','artifact-logs','huggingface','chat',?,?,?)`, tid, now, now)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}
	_, err = database.Exec(`INSERT INTO model_deployments (id, name, model_artifact_id, backend_runtime_id, tenant_id, created_at, updated_at)
		VALUES ('deploy-logs','deploy-logs','artifact-logs','runtime.vllm.nvidia-docker',?,?,?)`, tid, now, now)
	if err != nil {
		t.Fatalf("insert deployment: %v", err)
	}
	_, err = database.Exec(`INSERT INTO model_instances (id, deployment_id, tenant_id, replica_index, node_id, agent_id, current_run_plan_id, actual_state, desired_state, container_id, created_at, updated_at)
		VALUES ('inst-logs','deploy-logs',?,0,'node-logs','agent-logs','runplan-logs','failed','running','container-logs',?,?)`, tid, now, now)
	if err != nil {
		t.Fatalf("insert instance: %v", err)
	}
	_, err = database.Exec(`INSERT INTO resolved_run_plans (id, deployment_id, instance_id, tenant_id, backend_runtime_id, node_backend_runtime_id, plan_json, docker_preview, input_hash, plan_hash, created_at)
		VALUES ('runplan-logs','deploy-logs','inst-logs',?,'runtime.vllm.nvidia-docker','nbr-logs','{}','docker run test','ih','ph',?)`, tid, now)
	if err != nil {
		t.Fatalf("insert runplan: %v", err)
	}
	return h, tid
}

func TestNodeRunPlanLogsProxiesThroughAgentTask(t *testing.T) {
	h, _ := insertRunPlanLogsFixture(t, "online")

	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			var taskID string
			err := h.DB.QueryRow(`SELECT id FROM agent_tasks WHERE task_type='model_instance_logs' AND status='pending' ORDER BY created_at DESC LIMIT 1`).Scan(&taskID)
			if err == nil {
				req := newReq("POST", "/api/v1/agent/tasks/"+taskID+"/result", `{
					"status":"completed",
					"success":true,
					"instance_id":"inst-logs",
					"container_id":"container-logs",
					"runtime_state":"ok",
					"stdout":"ready\nAPI_KEY=super-secret\n",
					"stderr":"warn\n",
					"logs":"ready\nwarn\nAPI_KEY=super-secret\n"
				}`, nil, map[string]string{"id": taskID})
				h.HandleTaskResult(httptest.NewRecorder(), req)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	req := newReq("GET", "/api/v1/node-run-plans/runplan-logs/logs?tail=123", "", adminSession(), map[string]string{"id": "runplan-logs"})
	w := httptest.NewRecorder()
	h.HandleGetNodeRunPlanLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	<-done

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] == "DOCUMENTED_BLOCKER" {
		t.Fatalf("logs handler still returns blocker: %v", resp)
	}
	if got := resp["logs"].(string); !strings.Contains(got, "ready") || !strings.Contains(got, "warn") {
		t.Fatalf("logs missing stdout/stderr content: %q", got)
	}
	if strings.Contains(resp["logs"].(string), "super-secret") {
		t.Fatalf("sensitive env value was not redacted: %q", resp["logs"])
	}
	if resp["tail"].(float64) != 123 {
		t.Fatalf("tail=%v, want 123", resp["tail"])
	}
}

func TestNodeRunPlanLogsRejectsOfflineNode(t *testing.T) {
	h, _ := insertRunPlanLogsFixture(t, "offline")

	req := newReq("GET", "/api/v1/node-run-plans/runplan-logs/logs", "", adminSession(), map[string]string{"id": "runplan-logs"})
	w := httptest.NewRecorder()
	h.HandleGetNodeRunPlanLogs(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "offline") {
		t.Fatalf("offline error should be explicit: %s", w.Body.String())
	}
}

func TestNodeRunPlanLogsClassifiesLogEvents(t *testing.T) {
	h, _ := insertRunPlanLogsFixture(t, "online")

	done := make(chan struct{})
	go func() {
		defer close(done)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			var taskID string
			err := h.DB.QueryRow(`SELECT id FROM agent_tasks WHERE task_type='model_instance_logs' AND status='pending' ORDER BY created_at DESC LIMIT 1`).Scan(&taskID)
			if err == nil {
				// Return logs containing known warning patterns.
				req := newReq("POST", "/api/v1/agent/tasks/"+taskID+"/result", `{
					"status":"completed",
					"success":true,
					"instance_id":"inst-logs",
					"container_id":"container-logs",
					"runtime_state":"ok",
					"stdout":"",
					"stderr":"",
					"logs":"Attention backend not specified. Use flashinfer backend by default.\nwarn: LLAMA_ARG_HOST environment variable is set, but will be overwritten by command line argument --host\n"
				}`, nil, map[string]string{"id": taskID})
				h.HandleTaskResult(httptest.NewRecorder(), req)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	req := newReq("GET", "/api/v1/node-run-plans/runplan-logs/logs?tail=123", "", adminSession(), map[string]string{"id": "runplan-logs"})
	w := httptest.NewRecorder()
	h.HandleGetNodeRunPlanLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	<-done

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// Verify classified_log_events is present.
	eventsRaw, ok := resp["classified_log_events"]
	if !ok {
		t.Fatal("response missing classified_log_events")
	}
	events, ok := eventsRaw.([]interface{})
	if !ok {
		t.Fatalf("classified_log_events is not an array: %T", eventsRaw)
	}
	if len(events) == 0 {
		t.Fatal("classified_log_events is empty, expected at least 1 event")
	}

	// Verify at least one known rule matched.
	ruleIDs := make(map[string]bool)
	for _, ev := range events {
		if m, ok := ev.(map[string]interface{}); ok {
			if rid, ok := m["rule_id"].(string); ok {
				ruleIDs[rid] = true
			}
			// Verify required fields exist.
			if _, ok := m["severity"]; !ok {
				t.Error("event missing severity")
			}
			if _, ok := m["category"]; !ok {
				t.Error("event missing category")
			}
			if _, ok := m["message"]; !ok {
				t.Error("event missing message")
			}
		}
	}
	if !ruleIDs["sglang.attention_backend.default"] {
		t.Errorf("expected sglang.attention_backend.default rule, got rules: %v", ruleIDs)
	}
	if !ruleIDs["llamacpp.env_overwritten.host"] {
		t.Errorf("expected llamacpp.env_overwritten.host rule, got rules: %v", ruleIDs)
	}
}
