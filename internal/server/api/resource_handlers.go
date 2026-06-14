package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	// P1-004: Host system snapshot tables.
	database.Exec(`
		CREATE TABLE IF NOT EXISTS node_system_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT NOT NULL,
			cpu_utilization_percent TEXT NOT NULL DEFAULT '0',
			memory_total_bytes INTEGER NOT NULL DEFAULT 0,
			memory_used_bytes INTEGER NOT NULL DEFAULT 0,
			swap_total_bytes INTEGER NOT NULL DEFAULT 0,
			swap_used_bytes INTEGER NOT NULL DEFAULT 0,
			uptime_seconds TEXT NOT NULL DEFAULT '0',
			cpu_cores INTEGER NOT NULL DEFAULT 0,
			load1 TEXT NOT NULL DEFAULT '0',
			load5 TEXT NOT NULL DEFAULT '0',
			load15 TEXT NOT NULL DEFAULT '0',
			collected_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	database.Exec(`
		CREATE TABLE IF NOT EXISTS node_filesystem_snapshots (
			node_id TEXT NOT NULL,
			mount_point TEXT NOT NULL,
			device TEXT NOT NULL DEFAULT '',
			fs_type TEXT NOT NULL DEFAULT '',
			total_bytes INTEGER NOT NULL DEFAULT 0,
			used_bytes INTEGER NOT NULL DEFAULT 0,
			free_bytes INTEGER NOT NULL DEFAULT 0,
			used_percent TEXT NOT NULL DEFAULT '0',
			collected_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (node_id, mount_point)
		)
	`)
	database.Exec(`
		CREATE TABLE IF NOT EXISTS node_network_snapshots (
			node_id TEXT NOT NULL,
			interface_name TEXT NOT NULL,
			up INTEGER NOT NULL DEFAULT 0,
			bytes_recv INTEGER NOT NULL DEFAULT 0,
			bytes_sent INTEGER NOT NULL DEFAULT 0,
			collected_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (node_id, interface_name)
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
	Hostname           string                    `json:"hostname"`
	OS                 string                    `json:"os"`
	OSVersion          string                    `json:"os_version"`
	KernelVersion      string                    `json:"kernel_version"`
	CPUModel           string                    `json:"cpu_model"`
	CPUCores           int                       `json:"cpu_cores"`
	CPUUtilization     float64                   `json:"cpu_utilization_percent"`
	MemoryTotalBytes   uint64                    `json:"memory_total_bytes"`
	MemoryUsedBytes    uint64                    `json:"memory_used_bytes"`
	SwapTotalBytes     uint64                    `json:"swap_total_bytes"`
	SwapUsedBytes      uint64                    `json:"swap_used_bytes"`
	Load1              float64                   `json:"load1"`
	Load5              float64                   `json:"load5"`
	Load15             float64                   `json:"load15"`
	UptimeSeconds      uint64                    `json:"uptime_seconds"`
	Filesystems        []FilesystemSnapshotReq   `json:"filesystems"`
	NetworkInterfaces  []NetworkInterfaceSnapshotReq `json:"network_interfaces"`
	CollectedAt        string                    `json:"collected_at"`
}

type FilesystemSnapshotReq struct {
	MountPoint  string  `json:"mount_point"`
	Device      string  `json:"device"`
	FSType      string  `json:"fs_type"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type NetworkInterfaceSnapshotReq struct {
	Name      string `json:"name"`
	Up        bool   `json:"up"`
	BytesRecv uint64 `json:"bytes_recv"`
	BytesSent uint64 `json:"bytes_sent"`
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

	// P0-008: Use transaction for atomic resource write.
	tx, err := h.DB.Begin()
	if err != nil {
		log.Error("begin tx error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// P1-004: Save system snapshot (host CPU/memory/disk/network/uptime).
	if req.System != nil {
		sys := req.System
		cpuStr := fmt.Sprintf("%.1f", sys.CPUUtilization)
		_, err := tx.Exec(
			`INSERT INTO node_system_snapshots
			 (node_id, cpu_utilization_percent, memory_total_bytes, memory_used_bytes,
			  swap_total_bytes, swap_used_bytes, uptime_seconds, cpu_cores,
			  load1, load5, load15, collected_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			nodeID, cpuStr, sys.MemoryTotalBytes, sys.MemoryUsedBytes,
			sys.SwapTotalBytes, sys.SwapUsedBytes, fmt.Sprintf("%d", sys.UptimeSeconds), sys.CPUCores,
			fmt.Sprintf("%.2f", sys.Load1), fmt.Sprintf("%.2f", sys.Load5), fmt.Sprintf("%.2f", sys.Load15),
			now,
		)
		if err != nil {
			log.Error("save system snapshot error", "error", err)
			// Non-fatal — GPU data should still be saved.
		}

		// Save filesystem snapshots.
		for _, fs := range sys.Filesystems {
			if fs.MountPoint == "" {
				continue
			}
			tx.Exec(
				`INSERT OR REPLACE INTO node_filesystem_snapshots
				 (node_id, mount_point, device, fs_type, total_bytes, used_bytes, free_bytes, used_percent, collected_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				nodeID, fs.MountPoint, fs.Device, fs.FSType,
				fs.TotalBytes, fs.UsedBytes, fs.FreeBytes,
				fmt.Sprintf("%.1f", fs.UsedPercent), now,
			)
		}

		// Save network interface snapshots.
		for _, net := range sys.NetworkInterfaces {
			tx.Exec(
				`INSERT OR REPLACE INTO node_network_snapshots
				 (node_id, interface_name, up, bytes_recv, bytes_sent, collected_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				nodeID, net.Name, boolToInt(net.Up), net.BytesRecv, net.BytesSent, now,
			)
		}

		// Update node info.
		tx.Exec(`UPDATE nodes SET updated_at = ? WHERE id = ?`, now, nodeID)
	}

	// Collect UUIDs from this report for GPU staleness detection.
	reportedUUIDs := make(map[string]bool)

	// Update GPU devices.
	if req.GPUDevices != nil {
		for _, dev := range req.GPUDevices {
			reportedUUIDs[dev.UUID] = true

			collectedAt := dev.CollectedAt
			if collectedAt == "" {
				collectedAt = now
			}

			var existingID string
			err := tx.QueryRow(
				`SELECT id FROM gpu_devices WHERE node_id = ? AND uuid = ?`,
				nodeID, dev.UUID,
			).Scan(&existingID)

			if err == sql.ErrNoRows {
				// Create new GPU.
				gpuID := uuid.NewString()
				_, err = tx.Exec(
					`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, pci_bus_id,
					 driver_version, memory_total_bytes, status, collected_at, created_at, updated_at)
					 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					gpuID, nodeID, dev.Vendor, dev.Index, dev.Name, dev.UUID, dev.PCIBusID,
					dev.DriverVersion, dev.MemoryTotalBytes, dev.Status, collectedAt, now, now,
				)
				if err != nil {
					log.Error("create gpu error", "error", err)
					http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
					return
				}
			} else if err == nil {
				// Update existing GPU.
				_, err = tx.Exec(
					`UPDATE gpu_devices SET vendor = ?, index_num = ?, name = ?, driver_version = ?,
					 memory_total_bytes = ?, status = ?, collected_at = ?, updated_at = ?
					 WHERE id = ?`,
					dev.Vendor, dev.Index, dev.Name, dev.DriverVersion,
					dev.MemoryTotalBytes, dev.Status, collectedAt, now, existingID,
				)
				if err != nil {
					log.Error("update gpu error", "error", err)
					http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
					return
				}
			} else {
				log.Error("query gpu error", "error", err)
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}
		}
	}

	// Update GPU metrics.
	if req.GPUMetrics != nil {
		for _, m := range req.GPUMetrics {
			reportedUUIDs[m.UUID] = true

			collectedAt := m.CollectedAt
			if collectedAt == "" {
				collectedAt = now
			}

			_, err := tx.Exec(
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
			if err != nil {
				log.Error("update gpu metrics error", "error", err)
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}
		}
	}

	// P0-008: Mark GPUs that disappeared since last report as invalid.
	// GPUs for this node not in the current report should be marked unavailable.
	if len(reportedUUIDs) > 0 {
		rows, err := tx.Query(`SELECT uuid FROM gpu_devices WHERE node_id = ? AND status != 'unavailable'`, nodeID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var uuid string
				if err := rows.Scan(&uuid); err != nil {
					continue
				}
				if !reportedUUIDs[uuid] {
					// GPU was previously reported but not in this report → mark unavailable.
					tx.Exec(`UPDATE gpu_devices SET status = 'unavailable', updated_at = ? WHERE node_id = ? AND uuid = ?`,
						now, nodeID, uuid)
				}
			}
		}
	}

	// Persist diagnostics if provided (P0-008).
	if req.Diagnostics != nil {
		for _, d := range req.Diagnostics {
			// Store diagnostics in a simple log or metrics.
			// For now, log them at debug level.
			log.Debug("agent diagnostic",
				"agent_id", req.AgentID,
				"name", d.Name,
				"type", d.Type,
				"available", d.Available,
				"error", d.Error,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("commit tx error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	if h.Metrics != nil {
		h.Metrics.AgentReports.Inc()
	}
}

// MarkStaleGPUs marks GPU devices as unavailable if not updated within the threshold.
// Called periodically by the node health checker.
func (h *ResourceHandler) MarkStaleGPUs(nodeID string, threshold time.Duration) (int, error) {
	cutoff := time.Now().Add(-threshold).Format(time.RFC3339)
	result, err := h.DB.Exec(
		`UPDATE gpu_devices SET status = 'unavailable', updated_at = datetime('now')
		 WHERE node_id = ? AND status != 'unavailable' AND collected_at < ?`,
		nodeID, cutoff,
	)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
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
	gpu := scanGPUFromRow(h.DB.QueryRow(
		`SELECT id, node_id, vendor, index_num, name, uuid, pci_bus_id, driver_version,
		memory_total_bytes, memory_used_bytes, memory_free_bytes,
		gpu_utilization_percent, memory_utilization_percent,
		temperature_celsius, power_draw_watts,
		health, status, collected_at, created_at, updated_at
		FROM gpu_devices WHERE id = ?`, gpuID,
	))
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

// HandleGetNodeSystem handles GET /api/nodes/{id}/system.
// P1-004: Returns the latest host system snapshot for a node.
func (h *ResourceHandler) HandleGetNodeSystem(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	if nodeID == "" {
		http.Error(w, `{"error":"node id required"}`, http.StatusBadRequest)
		return
	}

	type FsInfo struct {
		MountPoint  string `json:"mount_point"`
		Device      string `json:"device"`
		FSType      string `json:"fs_type"`
		TotalBytes  uint64 `json:"total_bytes"`
		UsedBytes   uint64 `json:"used_bytes"`
		FreeBytes   uint64 `json:"free_bytes"`
		UsedPercent string `json:"used_percent"`
	}
	type NetInfo struct {
		Name      string `json:"name"`
		Up        bool   `json:"up"`
		BytesRecv uint64 `json:"bytes_recv"`
		BytesSent uint64 `json:"bytes_sent"`
	}
	type SysInfo struct {
		CPUUtilization string  `json:"cpu_utilization_percent"`
		MemoryTotal    uint64  `json:"memory_total_bytes"`
		MemoryUsed     uint64  `json:"memory_used_bytes"`
		SwapTotal      uint64  `json:"swap_total_bytes"`
		SwapUsed       uint64  `json:"swap_used_bytes"`
		UptimeSeconds  string  `json:"uptime_seconds"`
		CPUCores       int     `json:"cpu_cores"`
		Load1          string  `json:"load1"`
		Load5          string  `json:"load5"`
		Load15         string  `json:"load15"`
		CollectedAt    string  `json:"collected_at"`
		Filesystems    []FsInfo `json:"filesystems"`
		Networks       []NetInfo `json:"networks"`
	}

	resp := SysInfo{
		Filesystems: []FsInfo{},
		Networks:    []NetInfo{},
	}

	// Query latest system snapshot.
	row := h.DB.QueryRow(
		`SELECT cpu_utilization_percent, memory_total_bytes, memory_used_bytes,
		        swap_total_bytes, swap_used_bytes, uptime_seconds, cpu_cores,
		        load1, load5, load15, collected_at
		 FROM node_system_snapshots WHERE node_id = ?
		 ORDER BY collected_at DESC LIMIT 1`, nodeID)
	err := row.Scan(
		&resp.CPUUtilization, &resp.MemoryTotal, &resp.MemoryUsed,
		&resp.SwapTotal, &resp.SwapUsed, &resp.UptimeSeconds, &resp.CPUCores,
		&resp.Load1, &resp.Load5, &resp.Load15, &resp.CollectedAt,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Error("query system snapshot error", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Query filesystem snapshots.
	fsRows, err := h.DB.Query(
		`SELECT mount_point, device, fs_type, total_bytes, used_bytes, free_bytes, used_percent
		 FROM node_filesystem_snapshots WHERE node_id = ?`, nodeID)
	if err == nil {
		defer fsRows.Close()
		for fsRows.Next() {
			var fs FsInfo
			if err := fsRows.Scan(&fs.MountPoint, &fs.Device, &fs.FSType,
				&fs.TotalBytes, &fs.UsedBytes, &fs.FreeBytes, &fs.UsedPercent); err == nil {
				resp.Filesystems = append(resp.Filesystems, fs)
			}
		}
	}

	// Query network snapshots.
	netRows, err := h.DB.Query(
		`SELECT interface_name, up, bytes_recv, bytes_sent
		 FROM node_network_snapshots WHERE node_id = ?`, nodeID)
	if err == nil {
		defer netRows.Close()
		for netRows.Next() {
			var net NetInfo
			var upInt int
			if err := netRows.Scan(&net.Name, &upInt, &net.BytesRecv, &net.BytesSent); err == nil {
				net.Up = upInt == 1
				resp.Networks = append(resp.Networks, net)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
