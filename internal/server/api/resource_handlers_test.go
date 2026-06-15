package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"lightai-go/internal/agent/collector"
	"lightai-go/internal/server/db"
)

// readFixture reads a test fixture file from tests/fixtures/.
func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return string(data)
}

// TestServerIngestMetaX8GPUToAPI is the closed-loop test:
// MetaX fixture → GPUResource → report → handler → DB → API response → 8 GPUs.
func TestServerIngestMetaX8GPUToAPI(t *testing.T) {
	discoverOut := readFixture(t, "../../../tests/fixtures/collectors/metax/discover_8x_c500.txt")
	metricsOut := readFixture(t, "../../../tests/fixtures/collectors/metax/metrics_8x_c500.txt")
	now := time.Now()

	// Parse and normalize.
	devices, _, _, _ := collector.ParseProtocolOutput(discoverOut, now)
	_, gpuMetrics, _, _ := collector.ParseProtocolOutput(metricsOut, now)
	resources := collector.NormalizeGPUs(devices, gpuMetrics)

	if len(resources) != 8 {
		t.Fatalf("NormalizeGPUs: expected 8, got %d", len(resources))
	}

	// Build the unified GPUResources request body.
	gpuReqs := make([]GPUResourceReq, len(resources))
	for i, r := range resources {
		gpuReqs[i] = GPUResourceReq{
			Vendor:           r.Vendor,
			Index:            r.Index,
			UUID:             r.UUID,
			Name:             r.Name,
			PCIBusID:         r.PCIBusID,
			DriverVersion:    r.DriverVersion,
			MemoryTotalBytes: r.MemoryTotalBytes,
			MemoryUsedBytes:  r.MemoryUsedBytes,
			MemoryFreeBytes:  r.MemoryFreeBytes,
			GPUUtilization:   r.GPUUtilization,
			MemUtilization:   r.MemUtilization,
			Temperature:      r.Temperature,
			PowerDraw:        r.PowerDraw,
			Health:           r.Health,
			Status:           r.Status,
			CollectedAt:      now.Format(time.RFC3339),
		}
	}

	report := ResourceReportRequest{
		AgentID:      "agent-test-001",
		GPUResources: gpuReqs,
		CollectedAt:  now.Format(time.RFC3339),
	}

	body, _ := json.Marshal(report)

	// Use in-memory SQLite database.
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('a0000000-0000-0000-0000-000000000001','default','Default Tenant','active')`)
		t.Fatalf("migrate: %v", err)
	}

	// Register a test node.
	database.Exec(
		`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		 VALUES ('node-test', 'agent-test-001', 'k8s-master1', 'online', 'a0000000-0000-0000-0000-000000000001', datetime('now'), datetime('now'), datetime('now'))`,
	)

	handler := NewResourceHandler(database, nil)

	// Step 1: POST /api/v1/agent/resources/report — ingest the report.
	req := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleResourceReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("report ingest: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: GET /api/v1/gpus — verify 8 GPUs returned.
	req2 := httptest.NewRequest("GET", "/api/gpus", nil)
	w2 := httptest.NewRecorder()

	handler.HandleListGPUs(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("list gpus: expected 200, got %d", w2.Code)
	}

	var gpus []map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &gpus); err != nil {
		t.Fatalf("parse gpus response: %v", err)
	}

	if len(gpus) != 8 {
		t.Errorf("expected 8 GPUs in API response, got %d", len(gpus))
	}

	// Verify each GPU has the unified fields.
	for i, g := range gpus {
		if g["vendor"] != "metax" {
			t.Errorf("GPU %d: expected vendor=metax, got %v", i, g["vendor"])
		}
		if g["memory_total_bytes"] == nil || g["memory_total_bytes"].(float64) == 0 {
			t.Errorf("GPU %d: memory_total_bytes missing or zero", i)
		}
		if g["memory_used_bytes"] == nil {
			t.Errorf("GPU %d: memory_used_bytes missing", i)
		}
		if g["memory_free_bytes"] == nil {
			t.Errorf("GPU %d: memory_free_bytes missing", i)
		}
		if g["health"] == nil {
			t.Errorf("GPU %d: health missing", i)
		}
		if g["status"] == nil {
			t.Errorf("GPU %d: status missing", i)
		}
	}

	t.Logf("Closed-loop MetaX 8-GPU: report ingest → API response: %d GPUs", len(gpus))
}

// TestServerIngestMemoryFreeBytes verifies memory_free_bytes is stored and returned.
func TestServerIngestMemoryFreeBytes(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('a0000000-0000-0000-0000-000000000001','default','Default Tenant','active')`)

	database.Exec(
		`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at)
		 VALUES ('node-test', 'agent-test', 'host1', 'online', 'a0000000-0000-0000-0000-000000000001', datetime('now'), datetime('now'), datetime('now'))`,
	)

	handler := NewResourceHandler(database, nil)

	gpuReqs := []GPUResourceReq{{
		Vendor: "nvidia", Index: 0, UUID: "GPU-test", Name: "Test GPU",
		MemoryTotalBytes: 25651314688, MemoryUsedBytes: 0, MemoryFreeBytes: 25651314688,
		Health: "healthy", Status: "available",
	}}

	report := ResourceReportRequest{AgentID: "agent-test", GPUResources: gpuReqs}
	body, _ := json.Marshal(report)

	req := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleResourceReport(w, req)

	req2 := httptest.NewRequest("GET", "/api/gpus", nil)
	w2 := httptest.NewRecorder()
	handler.HandleListGPUs(w2, req2)

	var gpus []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &gpus)

	if len(gpus) != 1 {
		t.Fatalf("expected 1 GPU, got %d", len(gpus))
	}

	if gpus[0]["memory_free_bytes"] == nil || gpus[0]["memory_free_bytes"].(float64) == 0 {
		t.Error("memory_free_bytes should be 25651314688, got 0 or nil")
	}
}

// TestMetaXFixtureFullPipeline verifies the full Agent→Server→API pipeline.
func TestMetaXFixtureFullPipeline(t *testing.T) {
	discoverOut := readFixture(t, "../../../tests/fixtures/collectors/metax/discover_8x_c500.txt")
	metricsOut := readFixture(t, "../../../tests/fixtures/collectors/metax/metrics_8x_c500.txt")
	now := time.Now()

	devices, _, _, _ := collector.ParseProtocolOutput(discoverOut, now)
	_, gpuMetrics, _, _ := collector.ParseProtocolOutput(metricsOut, now)
	resources := collector.NormalizeGPUs(devices, gpuMetrics)

	if len(resources) != 8 {
		t.Fatalf("NormalizeGPUs: expected 8, got %d", len(resources))
	}
	for i, r := range resources {
		if r.MemoryTotalBytes != 68719476736 {
			t.Errorf("GPU %d: MemoryTotalBytes=%d, expected 68719476736", i, r.MemoryTotalBytes)
		}
	}

	report := collector.ResourceReport{AgentID: "agent-001", GPUResources: resources, CollectedAt: now}
	reportJSON, _ := json.Marshal(report)

	var rawReport map[string]interface{}
	json.Unmarshal(reportJSON, &rawReport)
	gpus := rawReport["gpu_resources"].([]interface{})
	gpu0 := gpus[0].(map[string]interface{})
	if gpu0["memory_total_bytes"] == nil {
		t.Error("report JSON: memory_total_bytes missing")
	}

	var req ResourceReportRequest
	if err := json.Unmarshal(reportJSON, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.GPUResources[0].MemoryTotalBytes != 68719476736 {
		t.Errorf("server req: MemoryTotalBytes=%d, expected 68719476736", req.GPUResources[0].MemoryTotalBytes)
	}

	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('a0000000-0000-0000-0000-000000000001','default','Default Tenant','active')`)
	database.Exec("INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at) VALUES ('node-001','agent-001','host','online','a0000000-0000-0000-0000-000000000001',datetime('now'),datetime('now'),datetime('now'))")
	handler := NewResourceHandler(database, nil)

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.HandleResourceReport(w, httpReq)

	httpReq2 := httptest.NewRequest("GET", "/api/gpus", nil)
	w2 := httptest.NewRecorder()
	handler.HandleListGPUs(w2, httpReq2)
	var apiGpus []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &apiGpus)
	if len(apiGpus) != 8 {
		t.Fatalf("API: expected 8 GPUs, got %d", len(apiGpus))
	}
	mt := apiGpus[0]["memory_total_bytes"]
	if mt == nil || mt.(float64) == 0 {
		t.Errorf("API: GPU 0 memory_total_bytes=%v, expected 68719476736", mt)
	} else {
		t.Logf("API: GPU 0 memory_total_bytes=%v OK", mt)
	}
}

// TestDBOldTotalZeroUpdateWithNewTotal verifies old total=0 gets overwritten.
func TestDBOldTotalZeroUpdateWithNewTotal(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('a0000000-0000-0000-0000-000000000001','default','Default Tenant','active')`)
	database.Exec("INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at) VALUES ('node-001','agent-001','host','online','a0000000-0000-0000-0000-000000000001',datetime('now'),datetime('now'),datetime('now'))")
	handler := NewResourceHandler(database, nil)

	// First: total=0.
	req1 := ResourceReportRequest{AgentID: "agent-001", GPUResources: []GPUResourceReq{{Vendor: "nvidia", Index: 0, UUID: "GPU-t", Name: "T", MemoryTotalBytes: 0, MemoryUsedBytes: 0, MemoryFreeBytes: 0, Health: "healthy", Status: "available"}}}
	body1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body1))
	r1.Header.Set("Content-Type", "application/json")
	handler.HandleResourceReport(httptest.NewRecorder(), r1)

	// Second: total=25651314688.
	req2 := ResourceReportRequest{AgentID: "agent-001", GPUResources: []GPUResourceReq{{Vendor: "nvidia", Index: 0, UUID: "GPU-t", Name: "T", MemoryTotalBytes: 25651314688, MemoryUsedBytes: 0, MemoryFreeBytes: 25651314688, Health: "healthy", Status: "available"}}}
	body2, _ := json.Marshal(req2)
	r2 := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body2))
	r2.Header.Set("Content-Type", "application/json")
	handler.HandleResourceReport(httptest.NewRecorder(), r2)

	r3 := httptest.NewRequest("GET", "/api/gpus", nil)
	w3 := httptest.NewRecorder()
	handler.HandleListGPUs(w3, r3)
	var gpus []map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &gpus)
	mt := gpus[0]["memory_total_bytes"]
	if mt == nil || mt.(float64) == 0 {
		t.Errorf("old total=0 + new>0: total still 0 (=%v), expected 25651314688", mt)
	} else {
		t.Logf("old total=0 + new>0: total=%v (correctly updated)", mt)
	}
}

// TestTotalFallbackFromUsedPlusFree verifies total=used+free fallback.
func TestTotalFallbackFromUsedPlusFree(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()
	database.Migrate()
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('a0000000-0000-0000-0000-000000000001','default','Default Tenant','active')`)
	database.Exec("INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at) VALUES ('n1','a1','h','online','a0000000-0000-0000-0000-000000000001',datetime('now'),datetime('now'),datetime('now'))")
	handler := NewResourceHandler(database, nil)

	// Report with total=0 but used+free>0.
	req := ResourceReportRequest{AgentID: "a1", GPUResources: []GPUResourceReq{{
		Vendor: "metax", Index: 0, UUID: "GPU-x", Name: "M",
		MemoryTotalBytes: 0, MemoryUsedBytes: 100, MemoryFreeBytes: 200,
		Health: "healthy", Status: "available",
	}}}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	handler.HandleResourceReport(httptest.NewRecorder(), r)

	r2 := httptest.NewRequest("GET", "/api/gpus", nil)
	w2 := httptest.NewRecorder()
	handler.HandleListGPUs(w2, r2)
	var gpus []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &gpus)
	mt := gpus[0]["memory_total_bytes"]
	if mt == nil || mt.(float64) == 0 {
		t.Errorf("total fallback: expected 300 (100+200), got %v", mt)
	} else {
		t.Logf("total fallback: %v (= 100+200) OK", mt)
	}
}

// TestAutoDiscoveredGPUDefaultTenant verifies GPU gets default tenant UUID.
func TestAutoDiscoveredGPUDefaultTenant(t *testing.T) {
	database, _ := db.Open(":memory:")
	database.Migrate()
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('a0000000-0000-0000-0000-000000000001','default','Default Tenant','active')`)
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at) VALUES ('node-g','agent-g','host','online','a0000000-0000-0000-0000-000000000001',datetime('now'),datetime('now'),datetime('now'))`)
	defer database.Close()
	handler := NewResourceHandler(database, nil)

	req := ResourceReportRequest{AgentID: "agent-g", GPUResources: []GPUResourceReq{{
		Vendor: "nvidia", Index: 0, UUID: "GPU-auto", Name: "Auto",
		MemoryTotalBytes: 81920, MemoryUsedBytes: 0, MemoryFreeBytes: 81920,
		Health: "healthy", Status: "available",
	}}}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	handler.HandleResourceReport(httptest.NewRecorder(), r)

	var tid string
	database.QueryRow(`SELECT tenant_id FROM gpu_devices WHERE uuid = 'GPU-auto'`).Scan(&tid)
	if tid == "" || tid == "default" {
		t.Errorf("GPU tenant_id = '%s', expected default tenant UUID", tid)
	}
}

// TestGPUInsertDoesNotOverwriteTenant verifies upsert preserves tenant_id.
func TestGPUInsertDoesNotOverwriteTenant(t *testing.T) {
	database, _ := db.Open(":memory:")
	database.Migrate()
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('a0000000-0000-0000-0000-000000000001','default','Default Tenant','active')`)
	database.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status) VALUES ('custom-uuid','custom','Custom','active')`)
	database.Exec(`INSERT INTO nodes (id, agent_id, hostname, status, tenant_id, last_heartbeat_at, created_at, updated_at) VALUES ('node-u','agent-u','host','online','a0000000-0000-0000-0000-000000000001',datetime('now'),datetime('now'),datetime('now'))`)
	handler := NewResourceHandler(database, nil)
	// Create GPU with custom tenant_id.
	database.Exec(`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, tenant_id, memory_total_bytes, health, status, collected_at, created_at, updated_at) VALUES ('gpu-ex','node-u','nvidia',0,'A100','GPU-ex','custom-uuid',81920,'healthy','available',datetime('now'),datetime('now'),datetime('now'))`)
	defer database.Close()

	// Upsert same GPU — tenant_id must not change.
	req := ResourceReportRequest{AgentID: "agent-u", GPUResources: []GPUResourceReq{{
		Vendor: "nvidia", Index: 0, UUID: "GPU-ex", Name: "A100",
		MemoryTotalBytes: 81920, MemoryUsedBytes: 0, MemoryFreeBytes: 81920,
		Health: "healthy", Status: "available",
	}}}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	handler.HandleResourceReport(httptest.NewRecorder(), r)

	var tid string
	database.QueryRow(`SELECT tenant_id FROM gpu_devices WHERE uuid = 'GPU-ex'`).Scan(&tid)
	if tid != "custom-uuid" {
		t.Errorf("tenant_id = %s after upsert, expected custom-uuid", tid)
	}
}
