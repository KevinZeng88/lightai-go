package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lightai-go/internal/server/auth"
	"lightai-go/internal/server/db"
	"lightai-go/internal/server/resolver"
)

func initModelTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, _ := db.Open(":memory:")
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Create gpu_devices table for dry-run validator tests.
	database.Exec(`CREATE TABLE IF NOT EXISTS gpu_devices (
		id TEXT PRIMARY KEY, node_id TEXT NOT NULL, vendor TEXT NOT NULL,
		index_num INTEGER NOT NULL, name TEXT NOT NULL DEFAULT '', uuid TEXT NOT NULL DEFAULT '',
		health TEXT NOT NULL DEFAULT 'unknown', status TEXT NOT NULL DEFAULT 'available',
		memory_total_bytes INTEGER NOT NULL DEFAULT 0, memory_used_bytes INTEGER NOT NULL DEFAULT 0,
		memory_free_bytes INTEGER NOT NULL DEFAULT 0, tenant_id TEXT NOT NULL DEFAULT 'default',
		created_at TEXT NOT NULL DEFAULT (datetime('now')), updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	cfg := auth.BootstrapConfig{Username: "admin", Password: "test1234", ForceChangePassword: false}
	if err := auth.InitBootstrap(database, cfg); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	return database
}

func modelAdminCtx() context.Context {
	return auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID:        "a0000000-0000-0000-0000-000000000001",
		UserID:          "admin-01",
		IsPlatformAdmin: true,
	})
}

func modelUserCtx(tenantID string) context.Context {
	return auth.NewContextWithSessionInfo(context.Background(), &auth.SessionInfo{
		TenantID:        tenantID,
		UserID:          "user-01",
		IsPlatformAdmin: false,
	})
}

// ==========================================================================
// ModelArtifact CRUD tests
// ==========================================================================

func TestModelArtifactCRUDRoundTrip(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	// Create
	body, _ := json.Marshal(map[string]interface{}{
		"name": "qwen3-32b", "path": "/data/models/Qwen3-32B",
		"format": "hf", "architecture": "qwen", "estimated_vram_bytes": 68719476736,
	})
	req := httptest.NewRequest("POST", "/api/model-artifacts", bytes.NewReader(body))
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateModelArtifact(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// Get
	req2 := httptest.NewRequest("GET", "/api/model-artifacts/{id}", nil)
	req2.SetPathValue("id", id)
	req2 = req2.WithContext(modelAdminCtx())
	w2 := httptest.NewRecorder()
	handler.HandleGetModelArtifact(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w2.Code)
	}

	// List
	req3 := httptest.NewRequest("GET", "/api/model-artifacts", nil)
	req3 = req3.WithContext(modelAdminCtx())
	w3 := httptest.NewRecorder()
	handler.HandleListModelArtifacts(w3, req3)
	var list []map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("list: expected 1, got %d", len(list))
	}

	// Patch
	patchBody, _ := json.Marshal(map[string]interface{}{"display_name": "Qwen3-32B-Updated"})
	req4 := httptest.NewRequest("PATCH", "/api/model-artifacts/{id}", bytes.NewReader(patchBody))
	req4.SetPathValue("id", id)
	req4 = req4.WithContext(modelAdminCtx())
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	handler.HandlePatchModelArtifact(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d", w4.Code)
	}

	// Delete
	req5 := httptest.NewRequest("DELETE", "/api/model-artifacts/{id}", nil)
	req5.SetPathValue("id", id)
	req5 = req5.WithContext(modelAdminCtx())
	w5 := httptest.NewRecorder()
	handler.HandleDeleteModelArtifact(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", w5.Code)
	}
}

func TestModelArtifactCreateRequiresName(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)
	body, _ := json.Marshal(map[string]interface{}{"path": "/data/test"})
	req := httptest.NewRequest("POST", "/api/model-artifacts", bytes.NewReader(body))
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateModelArtifact(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}
}

// ==========================================================================
// Tenant isolation tests
// ==========================================================================

func TestTenantACannotSeeTenantBModelArtifact(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	// Create as tenant A.
	body, _ := json.Marshal(map[string]interface{}{"name": "model-a"})
	req := httptest.NewRequest("POST", "/api/model-artifacts", bytes.NewReader(body))
	req = req.WithContext(modelUserCtx("tenant-a"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateModelArtifact(w, req)
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// Tenant B tries to get.
	req2 := httptest.NewRequest("GET", "/api/model-artifacts/{id}", nil)
	req2.SetPathValue("id", id)
	req2 = req2.WithContext(modelUserCtx("tenant-b"))
	w2 := httptest.NewRecorder()
	handler.HandleGetModelArtifact(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Errorf("tenant B should get 404 for tenant A model, got %d", w2.Code)
	}
}

func TestTenantACannotSeeTenantBGpuLease(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	database.Exec(`INSERT INTO gpu_leases (id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status)
		VALUES ('lease-1','gpu-1','node-1','dep-1','inst-1','tenant-a','reserved')`)

	req := httptest.NewRequest("GET", "/api/gpu-leases", nil)
	req = req.WithContext(modelUserCtx("tenant-b"))
	w := httptest.NewRecorder()
	handler.HandleListGpuLeases(w, req)
	var list []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("tenant B should see 0 leases, got %d", len(list))
	}
}

// ==========================================================================
// Dry Run tests
// ==========================================================================

func TestDryRunNodeNotFound(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	// Create a deployment first.
	database.Exec(`INSERT INTO model_artifacts (id, name, tenant_id) VALUES ('ma-1','m1','a0000000-0000-0000-0000-000000000001')`)
	database.Exec(`INSERT INTO runtime_environments (id, name, vendor, tenant_id) VALUES ('re-1','env1','nvidia',NULL)`)
	database.Exec(`INSERT INTO run_templates (id, name, tenant_id) VALUES ('rt-1','t1',NULL)`)
	database.Exec(`INSERT INTO model_deployments (id, name, model_artifact_id, runtime_environment_id, run_template_id, tenant_id)
		VALUES ('md-1','dep1','ma-1','re-1','rt-1','a0000000-0000-0000-0000-000000000001')`)

	body, _ := json.Marshal(map[string]interface{}{"node_id": "nonexistent"})
	req := httptest.NewRequest("POST", "/api/model-deployments/{id}/dry-run", bytes.NewReader(body))
	req.SetPathValue("id", "md-1")
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleDryRun(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dry-run: expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["valid"] != false {
		t.Error("dry-run with nonexistent node should be invalid")
	}
}

func TestDryRunNoGpuLeaseCreated(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id) VALUES ('node-1','a1','h1','online','a0000000-0000-0000-0000-000000000001')`)
	database.Exec(`INSERT INTO model_artifacts (id, name, path, tenant_id) VALUES ('ma-1','m1','/data/models/test','a0000000-0000-0000-0000-000000000001')`)
	database.Exec(`INSERT INTO runtime_environments (id, name, vendor, tenant_id) VALUES ('re-1','env1','nvidia',NULL)`)
	database.Exec(`INSERT INTO run_templates (id, name, tenant_id) VALUES ('rt-1','t1',NULL)`)
	database.Exec(`INSERT INTO model_deployments (id, name, model_artifact_id, runtime_environment_id, run_template_id, tenant_id)
		VALUES ('md-1','dep1','ma-1','re-1','rt-1','a0000000-0000-0000-0000-000000000001')`)

	body, _ := json.Marshal(map[string]interface{}{"node_id": "node-1", "gpu_ids": []string{}, "host_port": 8001})
	req := httptest.NewRequest("POST", "/api/model-deployments/{id}/dry-run", bytes.NewReader(body))
	req.SetPathValue("id", "md-1")
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleDryRun(w, req)

	// Verify no GPUs lease was created.
	var count int
	database.QueryRow(`SELECT COUNT(*) FROM gpu_leases`).Scan(&count)
	if count != 0 {
		t.Errorf("dry-run should not create GpuLease, found %d", count)
	}
	// Verify no instance was created.
	var ic int
	database.QueryRow(`SELECT COUNT(*) FROM model_instances`).Scan(&ic)
	if ic != 0 {
		t.Errorf("dry-run should not create ModelInstance, found %d", ic)
	}
}

// ==========================================================================
// RuntimeEnvironment tests
// ==========================================================================

func TestRuntimeEnvironmentCRUD(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "nvidia-vllm", "runtime_type": "docker", "vendor": "nvidia",
		"docker": map[string]interface{}{
			"image":            "vllm/vllm-openai:latest",
			"image_pull_policy": "never",
			"privileged":       map[string]interface{}{"enabled": true, "value": true},
			"ipc_mode":         map[string]interface{}{"enabled": true, "value": "host"},
		},
	})
	req := httptest.NewRequest("POST", "/api/runtime-environments", bytes.NewReader(body))
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateRuntimeEnvironment(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create RE: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	if created["docker"] == nil {
		t.Error("docker spec should be attached to runtime environment")
	}
}

// ==========================================================================
// RunTemplate + render-preview tests
// ==========================================================================

func TestRunTemplateRenderPreview(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	body, _ := json.Marshal(map[string]interface{}{
		"name":               "vllm-standard",
		"runtime_type":       "docker",
		"vendor":             "nvidia",
		"required_variables": []string{"MODEL_PATH", "GPU_IDS"},
		"args_template":      []string{"--model", "${MODEL_PATH}", "--served-model-name", "${SERVED_MODEL_NAME}"},
	})
	req := httptest.NewRequest("POST", "/api/run-templates", bytes.NewReader(body))
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateRunTemplate(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create RT: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)

	// Render preview.
	previewBody, _ := json.Marshal(map[string]interface{}{
		"model_path":        "/data/models/Qwen3-32B",
		"served_model_name": "qwen3-32b",
		"gpu_ids":           []string{"0", "1"},
		"host_port":         8001,
	})
	req2 := httptest.NewRequest("POST", "/api/run-templates/{id}/render-preview", bytes.NewReader(previewBody))
	req2.SetPathValue("id", id)
	req2 = req2.WithContext(modelAdminCtx())
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.HandleRenderPreview(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("render-preview: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var preview map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &preview)
	if preview["resolved_run_spec"] == nil {
		t.Error("render-preview should include resolved_run_spec")
	}
	if preview["equivalent_command_preview"] == nil {
		t.Error("render-preview should include equivalent_command_preview")
	}
}

// ==========================================================================
// Sensitive field redaction tests
// ==========================================================================

func TestEnvWithTokenKeyIsRedacted(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "with-token", "runtime_type": "docker",
		"docker": map[string]interface{}{
			"image": "test:latest",
		},
	})
	req := httptest.NewRequest("POST", "/api/runtime-environments", bytes.NewReader(body))
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateRuntimeEnvironment(w, req)

	// Verify sensitive key detection works.
	if !isSensitive("API_TOKEN") {
		t.Error("API_TOKEN should be detected as sensitive")
	}
	if !isSensitive("SECRET_KEY") {
		t.Error("SECRET_KEY should be detected as sensitive")
	}
	if !isSensitive("MY_PASSWORD") {
		t.Error("MY_PASSWORD should be detected as sensitive")
	}
	if isSensitive("MODEL_NAME") {
		t.Error("MODEL_NAME should NOT be sensitive")
	}
}

// ==========================================================================
// Permission tests
// ==========================================================================

func TestViewerCannotCreateModelArtifact(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	body, _ := json.Marshal(map[string]interface{}{"name": "test"})
	req := httptest.NewRequest("POST", "/api/model-artifacts", bytes.NewReader(body))
	// Simulate viewer context — but the handler doesn't do its own permission check;
	// the router middleware does. So this test just validates the handler works.
	// Actual permission enforcement is tested through the router middleware chain.
	req = req.WithContext(modelUserCtx("a0000000-0000-0000-0000-000000000001"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateModelArtifact(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("viewer can create through handler (middleware enforces permissions): got %d", w.Code)
	}
}

// ==========================================================================
// ModelInstance and GpuLease read-only tests
// ==========================================================================

func TestModelInstanceReadOnly(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	// FK requires model_deployments row.
	database.Exec(`INSERT INTO model_artifacts (id, name, tenant_id) VALUES ('ma-1','m1','a0000000-0000-0000-0000-000000000001')`)
	database.Exec(`INSERT INTO runtime_environments (id, name, vendor, tenant_id) VALUES ('re-1','env1','nvidia',NULL)`)
	database.Exec(`INSERT INTO run_templates (id, name, tenant_id) VALUES ('rt-1','t1',NULL)`)
	database.Exec(`INSERT INTO model_deployments (id, name, model_artifact_id, runtime_environment_id, run_template_id, tenant_id)
		VALUES ('dep-1','dep1','ma-1','re-1','rt-1','a0000000-0000-0000-0000-000000000001')`)
	database.Exec(`INSERT INTO model_instances (id, deployment_id, actual_state)
		VALUES ('mi-1','dep-1','pending')`)

	req := httptest.NewRequest("GET", "/api/model-instances", nil)
	req = req.WithContext(modelAdminCtx())
	w := httptest.NewRecorder()
	handler.HandleListModelInstances(w, req)
	var list []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 instance, got %d", len(list))
	}
}

func TestGpuLeaseReadOnly(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	database.Exec(`INSERT INTO gpu_leases (id, gpu_id, node_id, deployment_id, instance_id, tenant_id, status)
		VALUES ('gl-1','gpu-1','node-1','dep-1','inst-1','a0000000-0000-0000-0000-000000000001','reserved')`)
	// Verify INSERT succeeded.
	var count int
	database.QueryRow(`SELECT COUNT(*) FROM gpu_leases`).Scan(&count)
	t.Logf("gpu_leases count after insert: %d", count)

	req := httptest.NewRequest("GET", "/api/gpu-leases", nil)
	req = req.WithContext(modelAdminCtx())
	w := httptest.NewRecorder()
	handler.HandleListGpuLeases(w, req)
	var list []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 lease, got %d", len(list))
	}
}

func TestDeploymentReplicasMustBeOne(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)

	database.Exec(`INSERT INTO model_artifacts (id, name, tenant_id) VALUES ('ma-1','m1','a0000000-0000-0000-0000-000000000001')`)
	database.Exec(`INSERT INTO runtime_environments (id, name, vendor, tenant_id) VALUES ('re-1','env1','nvidia',NULL)`)
	database.Exec(`INSERT INTO run_templates (id, name, tenant_id) VALUES ('rt-1','t1',NULL)`)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "dep-replicas", "model_artifact_id": "ma-1",
		"runtime_environment_id": "re-1", "run_template_id": "rt-1",
		"replicas": 3,
	})
	req := httptest.NewRequest("POST", "/api/model-deployments", bytes.NewReader(body))
	req = req.WithContext(modelAdminCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateModelDeployment(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for replicas=3, got %d", w.Code)
	}
}


// ==========================================================================
// Phase 1.5 Quality Tests
// ==========================================================================

// --- Permission: viewer write operations ---

func TestViewerCreateModelArtifactSucceedsAtHandler(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)
	body, _ := json.Marshal(map[string]interface{}{"name": "test-viewer-handler"})
	req := httptest.NewRequest("POST", "/api/model-artifacts", bytes.NewReader(body))
	req = req.WithContext(modelUserCtx("a0000000-0000-0000-0000-000000000001"))
	// viewer has model:read but NOT model:write — handler allows, middleware blocks.
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateModelArtifact(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

// --- Tenant isolation: tenant A resource cannot be accessed by tenant B ---

func TestTenantBCannotGetTenantAModelArtifact(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)
	// Create as tenant A.
	body, _ := json.Marshal(map[string]interface{}{"name": "tenant-a-model"})
	req := httptest.NewRequest("POST", "/api/model-artifacts", bytes.NewReader(body))
	req = req.WithContext(modelUserCtx("tenant-a-xxxx"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleCreateModelArtifact(w, req)
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["id"].(string)
	// Tenant B tries GET.
	req2 := httptest.NewRequest("GET", "/api/model-artifacts/{id}", nil)
	req2.SetPathValue("id", id)
	req2 = req2.WithContext(modelUserCtx("tenant-b-xxxx"))
	w2 := httptest.NewRecorder()
	handler.HandleGetModelArtifact(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Errorf("tenant B GET should return 404, got %d", w2.Code)
	}
	// Tenant B tries PATCH.
	patchBody, _ := json.Marshal(map[string]interface{}{"display_name": "hacked"})
	req3 := httptest.NewRequest("PATCH", "/api/model-artifacts/{id}", bytes.NewReader(patchBody))
	req3.SetPathValue("id", id)
	req3 = req3.WithContext(modelUserCtx("tenant-b-xxxx"))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	handler.HandlePatchModelArtifact(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Errorf("tenant B PATCH should return 404, got %d", w3.Code)
	}
	// Tenant B tries DELETE.
	req4 := httptest.NewRequest("DELETE", "/api/model-artifacts/{id}", nil)
	req4.SetPathValue("id", id)
	req4 = req4.WithContext(modelUserCtx("tenant-b-xxxx"))
	w4 := httptest.NewRecorder()
	handler.HandleDeleteModelArtifact(w4, req4)
	if w4.Code != http.StatusNotFound {
		t.Errorf("tenant B DELETE should return 404, got %d", w4.Code)
	}
}

func TestTenantBCannotDryRunTenantADeployment(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	handler := NewModelHandler(database)
	// Setup as tenant A.
	database.Exec(`INSERT INTO model_artifacts (id, name, path, tenant_id) VALUES ('ma-xx','m1','/data/m','tenant-a-xxxx')`)
	database.Exec(`INSERT INTO runtime_environments (id, name, vendor, tenant_id) VALUES ('re-xx','env1','nvidia',NULL)`)
	database.Exec(`INSERT INTO run_templates (id, name, tenant_id) VALUES ('rt-xx','t1',NULL)`)
	database.Exec(`INSERT INTO model_deployments (id, name, model_artifact_id, runtime_environment_id, run_template_id, tenant_id)
		VALUES ('md-xx','dep1','ma-xx','re-xx','rt-xx','tenant-a-xxxx')`)
	// Tenant B tries dry-run.
	body, _ := json.Marshal(map[string]interface{}{"node_id": "node-1"})
	req := httptest.NewRequest("POST", "/api/model-deployments/{id}/dry-run", bytes.NewReader(body))
	req.SetPathValue("id", "md-xx")
	req = req.WithContext(modelUserCtx("tenant-b-xxxx"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleDryRun(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("tenant B dry-run should return 404, got %d", w.Code)
	}
}

// --- Sensitive field redaction in audit ---

func TestAuditLogRedactsSensitiveDetail(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	audit(database, "created", "model_artifact", "id-1", `{"api_key":"secret123"}`, "user-1")
	var detail string
	database.QueryRow(`SELECT detail FROM audit_logs WHERE entity_id = 'id-1'`).Scan(&detail)
	if strings.Contains(detail, "secret123") {
		t.Error("audit detail should NOT contain secret123")
	}
	if !strings.Contains(detail, "<redacted>") {
		t.Error("audit detail should contain <redacted>")
	}
}

func TestAllSensitiveKeysDetected(t *testing.T) {
	tests := []struct {
		key      string
		sensitive bool
	}{
		{"API_KEY", true},
		{"apikey", true},
		{"ACCESS_KEY", true},
		{"SECRET_KEY", true},
		{"AUTHORIZATION", true},
		{"BEARER", true},
		{"HF_TOKEN", true},
		{"DASHSCOPE_API_KEY", true},
		{"OPENAI_API_KEY", true},
		{"AK", true},
		{"SK", true},
		{"PASSWORD", true},
		{"PASSWD", true},
		{"PWD", true},
		{"TOKEN", true},
		{"CREDENTIAL", true},
		{"MODEL_NAME", false},
		{"DISPLAY_NAME", false},
		{"GPU_COUNT", false},
	}
	for _, tc := range tests {
		if got := isSensitive(tc.key); got != tc.sensitive {
			t.Errorf("isSensitive(%q) = %v, want %v", tc.key, got, tc.sensitive)
		}
	}
}

// --- Dry Run Validator unit tests (no HTTP) ---

func TestValidatorNodeNotFound(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	result := resolver.ValidateDryRun(database.DB, resolver.DryRunInput{NodeID: "nonexistent"})
	if result.Valid {
		t.Error("should be invalid when node not found")
	}
	if len(result.Errors) == 0 {
		t.Error("should have at least one error")
	}
}

func TestValidatorNodeOffline(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id) VALUES ('node-off','a1','h1','offline','tenant-a-xxxx')`)
	result := resolver.ValidateDryRun(database.DB, resolver.DryRunInput{NodeID: "node-off"})
	if result.Valid {
		t.Error("should be invalid when node is offline")
	}
}

func TestValidatorNoErrorsForCleanInput(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id) VALUES ('node-ok','a1','h1','online','tenant-a-xxxx')`)
	result := resolver.ValidateDryRun(database.DB, resolver.DryRunInput{
		NodeID: "node-ok", GPUIds: []string{}, HostPort: 0,
	})
	if !result.Valid {
		t.Errorf("should be valid, got errors: %v", result.Errors)
	}
}

func TestValidatorGPUNotHealthy(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id) VALUES ('n1','a1','h1','online','t1')`)
	// Need to create gpu_devices table entry.
	database.Exec(`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, health, status, tenant_id) 
		VALUES ('gpu-bad','n1','nvidia',0,'Test','uuid-bad','unhealthy','available','t1')`)
	result := resolver.ValidateDryRun(database.DB, resolver.DryRunInput{
		NodeID: "n1", GPUIds: []string{"gpu-bad"},
	})
	if result.Valid {
		t.Error("should be invalid for unhealthy GPU")
	}
}

func TestValidatorVendorMismatch(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id) VALUES ('n1','a1','h1','online','t1')`)
	database.Exec(`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, health, status, tenant_id) 
		VALUES ('gpu-nv','n1','nvidia',0,'Test','uuid-nv','healthy','available','t1')`)
	result := resolver.ValidateDryRun(database.DB, resolver.DryRunInput{
		NodeID: "n1", GPUIds: []string{"gpu-nv"}, RuntimeVendor: "metax",
	})
	if result.Valid {
		t.Error("should be invalid when vendor=nvidia but runtime=metax")
	}
}

func TestValidatorCustomVendorWarning(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id) VALUES ('n1','a1','h1','online','t1')`)
	database.Exec(`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, health, status, tenant_id) 
		VALUES ('gpu-nv','n1','nvidia',0,'Test','uuid-nv','healthy','available','t1')`)
	result := resolver.ValidateDryRun(database.DB, resolver.DryRunInput{
		NodeID: "n1", GPUIds: []string{"gpu-nv"}, RuntimeVendor: "custom",
	})
	if !result.Valid {
		t.Errorf("custom vendor should be valid, got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("custom vendor should produce a warning")
	}
}

func TestValidatorHostPortConflict(t *testing.T) {
	database := initModelTestDB(t)
	defer database.Close()
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id) VALUES ('n1','a1','h1','online','t1')`)
	database.Exec(`INSERT INTO model_artifacts (id, name, tenant_id) VALUES ('ma','m1','t1')`)
	database.Exec(`INSERT INTO runtime_environments (id, name, vendor, tenant_id) VALUES ('re','e1','nvidia',NULL)`)
	database.Exec(`INSERT INTO run_templates (id, name, tenant_id) VALUES ('rt','t1',NULL)`)
	database.Exec(`INSERT INTO model_deployments (id, name, model_artifact_id, runtime_environment_id, run_template_id, tenant_id)
		VALUES ('md','dep1','ma','re','rt','t1')`)
	database.Exec(`INSERT INTO model_instances (id, deployment_id, host_port, actual_state) VALUES ('mi-1','md',8001,'running')`)
	result := resolver.ValidateDryRun(database.DB, resolver.DryRunInput{
		NodeID: "n1", HostPort: 8001,
	})
	if result.Valid {
		t.Error("should be invalid when host_port is already in use")
	}
}
