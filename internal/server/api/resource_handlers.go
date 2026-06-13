package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/db"
	srvmetrics "lightai-go/internal/server/metrics"

	"github.com/google/uuid"
)

// ResourceHandler handles resource reporting and GPU query APIs.
type ResourceHandler struct {
	DB      *db.DB
	Metrics *srvmetrics.ServerMetrics
}

// NewResourceHandler creates a new ResourceHandler.
func NewResourceHandler(database *db.DB, m *srvmetrics.ServerMetrics) *ResourceHandler {
	// Ensure GPU table exists.
	database.Exec(`
		CREATE TABLE IF NOT EXISTS gpu_devices (
			id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL,
			vendor TEXT NOT NULL,
			index_num INTEGER NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			uuid TEXT NOT NULL DEFAULT '',
			pci_bus_id TEXT NOT NULL DEFAULT '',
			driver_version TEXT NOT NULL DEFAULT '',
			memory_total_bytes INTEGER NOT NULL DEFAULT 0,
			memory_used_bytes INTEGER NOT NULL DEFAULT 0,
			memory_free_bytes INTEGER NOT NULL DEFAULT 0,
			gpu_utilization_percent REAL,
			memory_utilization_percent REAL,
			temperature_celsius REAL,
			power_draw_watts REAL,
			health TEXT NOT NULL DEFAULT 'unknown',
			status TEXT NOT NULL DEFAULT 'available',
			collected_at TEXT,
			tenant_id TEXT NOT NULL DEFAULT 'default',
			owner_id TEXT,
			created_by TEXT NOT NULL DEFAULT 'system',
			updated_by TEXT NOT NULL DEFAULT 'system',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	return &ResourceHandler{DB: database, Metrics: m}
}

// ResourceReportRequest is the resource report from an agent.
type ResourceReportRequest struct {
	AgentID     string             `json:"agent_id"`
	System      *SystemSnapshotReq `json:"system"`
	GPUDevices  []GPUDeviceReq     `json:"gpu_devices"`
	GPUMetrics  []GPUMetricReq     `json:"gpu_metrics"`
	Diagnostics []DiagnosisReq     `json:"diagnostics"`
	CollectedAt string             `json:"collected_at"`
}

type SystemSnapshotReq struct {
	Hostname         string  `json:"hostname"`
	OS               string  `json:"os"`
	OSVersion        string  `json:"os_version"`
	KernelVersion    string  `json:"kernel_version"`
	CPUModel         string  `json:"cpu_model"`
	CPUCores         int     `json:"cpu_cores"`
	CPUUtilization   float64 `json:"cpu_utilization_percent"`
	MemoryTotalBytes uint64  `json:"memory_total_bytes"`
	MemoryUsedBytes  uint64  `json:"memory_used_bytes"`
	SwapTotalBytes   uint64  `json:"swap_total_bytes"`
	SwapUsedBytes    uint64  `json:"swap_used_bytes"`
	CollectedAt      string  `json:"collected_at"`
}

type GPUDeviceReq struct {
	Vendor           string `json:"vendor"`
	Index            int    `json:"index"`
	Name             string `json:"name"`
	UUID             string `json:"uuid"`
	PCIBusID         string `json:"pci_bus_id"`
	DriverVersion    string `json:"driver_version"`
	MemoryTotalBytes uint64 `json:"memory_total_bytes"`
	Status           string `json:"status"`
	CollectedAt      string `json:"collected_at"`
}

type GPUMetricReq struct {
	Vendor            string   `json:"vendor"`
	Index             int      `json:"index"`
	UUID              string   `json:"uuid"`
	MemoryUsedBytes   uint64   `json:"memory_used_bytes"`
	MemoryFreeBytes   uint64   `json:"memory_free_bytes"`
	GPUUtilization    *float64 `json:"gpu_utilization_percent,omitempty"`
	MemoryUtilization *float64 `json:"memory_utilization_percent,omitempty"`
	Temperature       *float64 `json:"temperature_celsius,omitempty"`
	PowerDraw         *float64 `json:"power_draw_watts,omitempty"`
	Health            string   `json:"health"`
	CollectedAt       string   `json:"collected_at"`
}

type DiagnosisReq struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Vendor    string `json:"vendor,omitempty"`
	Available bool   `json:"available"`
	ToolPath  string `json:"tool_path,omitempty"`
	Error     string `json:"error,omitempty"`
	CheckedAt string `json:"checked_at"`
}

// HandleResourceReport handles POST /api/agent/resources/report.
func (h *ResourceHandler) HandleResourceReport(w http.ResponseWriter, r *http.Request) {
	var req ResourceReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		http.Error(w, `{"error":"agent_id required"}`, http.StatusBadRequest)
		return
	}

	// Find node.
	var nodeID string
	err := h.DB.QueryRow(`SELECT id FROM nodes WHERE agent_id = ?`, req.AgentID).Scan(&nodeID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"agent not registered"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("query node error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	now := time.Now().Format(time.RFC3339)

	// Update node with system snapshot info.
	if req.System != nil {
		h.DB.Exec(`UPDATE nodes SET updated_at = ? WHERE id = ?`, now, nodeID)
	}

	// Update GPU devices.
	if req.GPUDevices != nil {
		for _, dev := range req.GPUDevices {
			// Check if GPU already exists (by node_id + uuid).
			var existingID string
			err := h.DB.QueryRow(
				`SELECT id FROM gpu_devices WHERE node_id = ? AND uuid = ?`,
				nodeID, dev.UUID,
			).Scan(&existingID)

			collectedAt := dev.CollectedAt
			if collectedAt == "" {
				collectedAt = now
			}

			if err == sql.ErrNoRows {
				// Create new GPU.
				gpuID := uuid.NewString()
				h.DB.Exec(
					`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, pci_bus_id,
					 driver_version, memory_total_bytes, status, collected_at, created_at, updated_at)
					 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					gpuID, nodeID, dev.Vendor, dev.Index, dev.Name, dev.UUID, dev.PCIBusID,
					dev.DriverVersion, dev.MemoryTotalBytes, dev.Status, collectedAt, now, now,
				)
			} else if err == nil {
				// Update existing GPU.
				h.DB.Exec(
					`UPDATE gpu_devices SET vendor = ?, index_num = ?, name = ?, driver_version = ?,
					 memory_total_bytes = ?, status = ?, collected_at = ?, updated_at = ?
					 WHERE id = ?`,
					dev.Vendor, dev.Index, dev.Name, dev.DriverVersion,
					dev.MemoryTotalBytes, dev.Status, collectedAt, now, existingID,
				)
			}
		}
	}

	// Update GPU metrics.
	if req.GPUMetrics != nil {
		for _, m := range req.GPUMetrics {
			collectedAt := m.CollectedAt
			if collectedAt == "" {
				collectedAt = now
			}

			h.DB.Exec(
				`UPDATE gpu_devices SET
				 memory_used_bytes = ?, memory_free_bytes = ?,
				 gpu_utilization_percent = ?, memory_utilization_percent = ?,
				 temperature_celsius = ?, power_draw_watts = ?,
				 health = ?, collected_at = ?, updated_at = ?
				 WHERE node_id = ? AND uuid = ?`,
				m.MemoryUsedBytes, m.MemoryFreeBytes,
				m.GPUUtilization, m.MemoryUtilization,
				m.Temperature, m.PowerDraw,
				m.Health, collectedAt, now,
				nodeID, m.UUID,
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	if h.Metrics != nil {
		h.Metrics.AgentReports.Inc()
	}
}

// HandleListGPUs handles GET /api/gpus.
func (h *ResourceHandler) HandleListGPUs(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	vendor := r.URL.Query().Get("vendor")

	query := `SELECT id, node_id, vendor, index_num, name, uuid, pci_bus_id, driver_version,
		memory_total_bytes, memory_used_bytes, memory_free_bytes,
		gpu_utilization_percent, memory_utilization_percent,
		temperature_celsius, power_draw_watts,
		health, status, collected_at, created_at, updated_at
		FROM gpu_devices WHERE 1=1`
	args := []interface{}{}

	if nodeID != "" {
		query += " AND node_id = ?"
		args = append(args, nodeID)
	}
	if vendor != "" {
		query += " AND vendor = ?"
		args = append(args, vendor)
	}

	query += " ORDER BY node_id, index_num"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Error("list gpus error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var gpus []map[string]interface{}
	for rows.Next() {
		gpu := scanGPU(rows)
		if gpu != nil {
			gpus = append(gpus, gpu)
		}
	}
	if gpus == nil {
		gpus = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gpus)
}

// HandleGetGPU handles GET /api/gpus/{id}.
func (h *ResourceHandler) HandleGetGPU(w http.ResponseWriter, r *http.Request) {
	gpuID := r.PathValue("id")
	row := h.DB.QueryRow(
		`SELECT id, node_id, vendor, index_num, name, uuid, pci_bus_id, driver_version,
		memory_total_bytes, memory_used_bytes, memory_free_bytes,
		gpu_utilization_percent, memory_utilization_percent,
		temperature_celsius, power_draw_watts,
		health, status, collected_at, created_at, updated_at
		FROM gpu_devices WHERE id = ?`, gpuID,
	)

	rowData := &sql.Row{}
	_ = rowData
	// Hack: use the same scan pattern.
	gpu := scanGPUFromRow(row)
	if gpu == nil {
		http.Error(w, `{"error":"gpu not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gpu)
}

func scanGPU(rows *sql.Rows) map[string]interface{} {
	var id, nodeID, vendor, name, uuid, pciBusID, driverVersion, health, status string
	var indexNum int
	var memTotal, memUsed, memFree uint64
	var gpuUtil, memUtil, temp, power sql.NullFloat64
	var collectedAt, createdAt, updatedAt sql.NullString

	err := rows.Scan(&id, &nodeID, &vendor, &indexNum, &name, &uuid, &pciBusID, &driverVersion,
		&memTotal, &memUsed, &memFree,
		&gpuUtil, &memUtil, &temp, &power,
		&health, &status, &collectedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil
	}

	gpu := map[string]interface{}{
		"id":                 id,
		"node_id":            nodeID,
		"vendor":             vendor,
		"index":              indexNum,
		"name":               name,
		"uuid":               uuid,
		"pci_bus_id":         pciBusID,
		"driver_version":     driverVersion,
		"memory_total_bytes": memTotal,
		"memory_used_bytes":  memUsed,
		"memory_free_bytes":  memFree,
		"health":             health,
		"status":             status,
	}
	if gpuUtil.Valid {
		gpu["gpu_utilization_percent"] = gpuUtil.Float64
	}
	if memUtil.Valid {
		gpu["memory_utilization_percent"] = memUtil.Float64
	}
	if temp.Valid {
		gpu["temperature_celsius"] = temp.Float64
	}
	if power.Valid {
		gpu["power_draw_watts"] = power.Float64
	}
	if collectedAt.Valid {
		gpu["collected_at"] = collectedAt.String
	}
	if createdAt.Valid {
		gpu["created_at"] = createdAt.String
	}
	if updatedAt.Valid {
		gpu["updated_at"] = updatedAt.String
	}
	return gpu
}

func scanGPUFromRow(row *sql.Row) map[string]interface{} {
	var id, nodeID, vendor, name, uuid, pciBusID, driverVersion, health, status string
	var indexNum int
	var memTotal, memUsed, memFree uint64
	var gpuUtil, memUtil, temp, power sql.NullFloat64
	var collectedAt, createdAt, updatedAt sql.NullString

	err := row.Scan(&id, &nodeID, &vendor, &indexNum, &name, &uuid, &pciBusID, &driverVersion,
		&memTotal, &memUsed, &memFree,
		&gpuUtil, &memUtil, &temp, &power,
		&health, &status, &collectedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil
	}

	gpu := map[string]interface{}{
		"id":                 id,
		"node_id":            nodeID,
		"vendor":             vendor,
		"index":              indexNum,
		"name":               name,
		"uuid":               uuid,
		"pci_bus_id":         pciBusID,
		"driver_version":     driverVersion,
		"memory_total_bytes": memTotal,
		"memory_used_bytes":  memUsed,
		"memory_free_bytes":  memFree,
		"health":             health,
		"status":             status,
	}
	if gpuUtil.Valid {
		gpu["gpu_utilization_percent"] = gpuUtil.Float64
	}
	if memUtil.Valid {
		gpu["memory_utilization_percent"] = memUtil.Float64
	}
	if temp.Valid {
		gpu["temperature_celsius"] = temp.Float64
	}
	if power.Valid {
		gpu["power_draw_watts"] = power.Float64
	}
	if collectedAt.Valid {
		gpu["collected_at"] = collectedAt.String
	}
	if createdAt.Valid {
		gpu["created_at"] = createdAt.String
	}
	if updatedAt.Valid {
		gpu["updated_at"] = updatedAt.String
	}
	return gpu
}
