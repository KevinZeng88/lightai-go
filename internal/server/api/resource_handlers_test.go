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
		t.Fatalf("migrate: %v", err)
	}

	// Register a test node.
	database.Exec(
		`INSERT INTO nodes (id, agent_id, hostname, status, last_heartbeat_at, created_at, updated_at)
		 VALUES ('node-test', 'agent-test-001', 'k8s-master1', 'online', datetime('now'), datetime('now'), datetime('now'))`,
	)

	handler := NewResourceHandler(database, nil)

	// Step 1: POST /api/agent/resources/report — ingest the report.
	req := httptest.NewRequest("POST", "/api/agent/resources/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleResourceReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("report ingest: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Step 2: GET /api/gpus — verify 8 GPUs returned.
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

	database.Exec(
		`INSERT INTO nodes (id, agent_id, hostname, status, last_heartbeat_at, created_at, updated_at)
		 VALUES ('node-test', 'agent-test', 'host1', 'online', datetime('now'), datetime('now'), datetime('now'))`,
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
