package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"lightai-go/internal/common/log"
	"lightai-go/internal/server/auth"
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
	AgentID      string             `json:"agent_id"`
	System       *SystemSnapshotReq `json:"system"`
	GPUResources []GPUResourceReq   `json:"gpu_resources"`
	Diagnostics  []DiagnosisReq     `json:"diagnostics"`
	CollectedAt  string             `json:"collected_at"`
}

type SystemSnapshotReq struct {
	Hostname          string                        `json:"hostname"`
	OS                string                        `json:"os"`
	OSVersion         string                        `json:"os_version"`
	KernelVersion     string                        `json:"kernel_version"`
	CPUModel          string                        `json:"cpu_model"`
	CPUCores          int                           `json:"cpu_cores"`
	CPUUtilization    float64                       `json:"cpu_utilization_percent"`
	MemoryTotalBytes  uint64                        `json:"memory_total_bytes"`
	MemoryUsedBytes   uint64                        `json:"memory_used_bytes"`
	SwapTotalBytes    uint64                        `json:"swap_total_bytes"`
	SwapUsedBytes     uint64                        `json:"swap_used_bytes"`
	Load1             float64                       `json:"load1"`
	Load5             float64                       `json:"load5"`
	Load15            float64                       `json:"load15"`
	UptimeSeconds     uint64                        `json:"uptime_seconds"`
	Filesystems       []FilesystemSnapshotReq       `json:"filesystems"`
	NetworkInterfaces []NetworkInterfaceSnapshotReq `json:"network_interfaces"`
	CollectedAt       string                        `json:"collected_at"`
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

// GPUResourceReq is the unified GPU resource in an agent report.
// GPUResourceReq is the unified GPU resource in an agent report (RC1 model).
// GPUDeviceInfo / GPUMetricInfo are parser-only raw records — never used here.
type GPUResourceReq struct {
	Vendor           string   `json:"vendor"`
	Index            int      `json:"index"`
	UUID             string   `json:"uuid"`
	Name             string   `json:"name"`
	PCIBusID         string   `json:"pci_bus_id"`
	DriverVersion    string   `json:"driver_version"`
	MemoryTotalBytes uint64   `json:"memory_total_bytes"`
	MemoryUsedBytes  uint64   `json:"memory_used_bytes"`
	MemoryFreeBytes  uint64   `json:"memory_free_bytes"`
	GPUUtilization   *float64 `json:"gpu_utilization_percent,omitempty"`
	MemUtilization   *float64 `json:"memory_utilization_percent,omitempty"`
	Temperature      *float64 `json:"temperature_celsius,omitempty"`
	PowerDraw        *float64 `json:"power_draw_watts,omitempty"`
	Health           string   `json:"health"`
	Status           string   `json:"status"`
	CollectedAt      string   `json:"collected_at"`
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
			if _, ferr := tx.Exec(
				`INSERT OR REPLACE INTO node_filesystem_snapshots
				 (node_id, mount_point, device, fs_type, total_bytes, used_bytes, free_bytes, used_percent, collected_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				nodeID, fs.MountPoint, fs.Device, fs.FSType,
				fs.TotalBytes, fs.UsedBytes, fs.FreeBytes,
				fmt.Sprintf("%.1f", fs.UsedPercent), now,
			); ferr != nil {
				log.Error("save filesystem snapshot error", "node_id", nodeID, "mount_point", fs.MountPoint, "error", ferr)
			}
		}

		// Save network interface snapshots.
		for _, net := range sys.NetworkInterfaces {
			if _, nerr := tx.Exec(
				`INSERT OR REPLACE INTO node_network_snapshots
				 (node_id, interface_name, up, bytes_recv, bytes_sent, collected_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				nodeID, net.Name, boolToInt(net.Up), net.BytesRecv, net.BytesSent, now,
			); nerr != nil {
				// AUD-006: Log non-fatal network write error.
				log.Error("save network snapshot error", "node_id", nodeID, "interface", net.Name, "error", nerr)
			}
		}

		// Update node info (timestamp + enrich from system snapshot).
		// P2-001: A successful resource report is also proof of Agent life.
		// Update last_heartbeat_at and status so the node stays online even
		// if heartbeat requests fail while resource reports succeed.
		if sys.OS != "" || sys.KernelVersion != "" {
			if _, uerr := tx.Exec(`UPDATE nodes SET
				os = CASE WHEN ? != '' THEN ? ELSE os END,
				kernel = CASE WHEN ? != '' THEN ? ELSE kernel END,
				last_heartbeat_at = ?, status = 'online', updated_at = ?
				WHERE id = ?`,
				sys.OS, sys.OS,
				sys.KernelVersion, sys.KernelVersion,
				now, now, nodeID,
			); uerr != nil {
				log.Error("update node info error", "node_id", nodeID, "error", uerr)
			}
		} else {
			if _, uerr := tx.Exec(`UPDATE nodes SET last_heartbeat_at = ?, status = 'online', updated_at = ? WHERE id = ?`,
				now, now, nodeID); uerr != nil {
				log.Error("update node heartbeat error", "node_id", nodeID, "error", uerr)
			}
		}
	}

	// P0-008: Process unified GPU resources (RC1 model).
	// No more separate GPUDevices/GPUMetrics merge — the Agent sends pre-normalized GPUResources.
	reportedUUIDs := make(map[string]bool)
	if req.GPUResources != nil {
		for _, g := range req.GPUResources {
			key := g.Vendor + ":" + g.UUID
			if g.UUID == "" {
				key = g.Vendor + ":" + fmt.Sprintf("%d", g.Index)
			}
			reportedUUIDs[key] = true

			// Fallback: if total is 0 but used+free > 0, derive total.
			if g.MemoryTotalBytes == 0 && g.MemoryUsedBytes+g.MemoryFreeBytes > 0 {
				g.MemoryTotalBytes = g.MemoryUsedBytes + g.MemoryFreeBytes
			}

			// P2-001: Use server receive time for collected_at, not Agent payload time.
			// Agent payload time can be stale when GPU collection fails and
			// the Agent falls back to cached data with the original timestamp.
			// Server now reflects when data was last successfully received.
			collectedAt := now

			var existingID string
			err := tx.QueryRow(
				`SELECT id FROM gpu_devices WHERE node_id = ? AND uuid = ?`,
				nodeID, g.UUID,
			).Scan(&existingID)

			if err == sql.ErrNoRows {
				gpuID := uuid.NewString()
				// Inherit tenant_id from the node, not hardcoded default.
				nodeTenantID := h.DB.DefaultTenantID()
				var ntid string
				if err := tx.QueryRow(`SELECT tenant_id FROM nodes WHERE id = ?`, nodeID).Scan(&ntid); err == nil && ntid != "" {
					nodeTenantID = ntid
				}
				_, err = tx.Exec(
					`INSERT INTO gpu_devices (id, node_id, vendor, index_num, name, uuid, pci_bus_id,
					 driver_version, memory_total_bytes, memory_used_bytes, memory_free_bytes,
					 gpu_utilization_percent, memory_utilization_percent,
					 temperature_celsius, power_draw_watts,
					 health, status, tenant_id, collected_at, created_at, updated_at)
					 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					gpuID, nodeID, g.Vendor, g.Index, g.Name, g.UUID, g.PCIBusID,
					g.DriverVersion,
					g.MemoryTotalBytes, g.MemoryUsedBytes, g.MemoryFreeBytes,
					g.GPUUtilization, g.MemUtilization,
					g.Temperature, g.PowerDraw,
					g.Health, g.Status, nodeTenantID, collectedAt, now, now,
				)
				if err != nil {
					log.Error("create gpu error", "error", err)
					http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
					return
				}
			} else if err == nil {
				_, err = tx.Exec(
					`UPDATE gpu_devices SET vendor = ?, index_num = ?, name = ?, driver_version = ?,
					 memory_total_bytes = ?, memory_used_bytes = ?, memory_free_bytes = ?,
					 gpu_utilization_percent = ?, memory_utilization_percent = ?,
					 temperature_celsius = ?, power_draw_watts = ?,
					 health = ?, status = ?, collected_at = ?, updated_at = ?
					 WHERE id = ?`,
					g.Vendor, g.Index, g.Name, g.DriverVersion,
					g.MemoryTotalBytes, g.MemoryUsedBytes, g.MemoryFreeBytes,
					g.GPUUtilization, g.MemUtilization,
					g.Temperature, g.PowerDraw,
					g.Health, g.Status, collectedAt, now, existingID,
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

	// Mark GPUs that disappeared since last report as unavailable.
	if len(reportedUUIDs) > 0 {
		rows, err := tx.Query(`SELECT vendor, uuid FROM gpu_devices WHERE node_id = ? AND status != 'unavailable'`, nodeID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var vendor, gpuUUID string
				if err := rows.Scan(&vendor, &gpuUUID); err != nil {
					continue
				}
				key := vendor + ":" + gpuUUID
				if !reportedUUIDs[key] {
					tx.Exec(`UPDATE gpu_devices SET status = 'unavailable', updated_at = ? WHERE node_id = ? AND uuid = ?`,
						now, nodeID, gpuUUID)
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

	// Query GPUs before update for state transition logging.
	rows, err := h.DB.Query(
		`SELECT id, gpu_index, vendor, health, status FROM gpu_devices
		 WHERE node_id = ? AND status != 'unavailable' AND collected_at < ?`,
		nodeID, cutoff,
	)
	if err != nil {
		return 0, err
	}
	type gpuInfo struct{ id, gpuIndex, vendor, health, status string }
	var gpus []gpuInfo
	for rows.Next() {
		var g gpuInfo
		rows.Scan(&g.id, &g.gpuIndex, &g.vendor, &g.health, &g.status)
		gpus = append(gpus, g)
	}
	rows.Close()

	result, err := h.DB.Exec(
		`UPDATE gpu_devices SET status = 'unavailable', updated_at = datetime('now')
		 WHERE node_id = ? AND status != 'unavailable' AND collected_at < ?`,
		nodeID, cutoff,
	)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()

	for _, g := range gpus {
		log.StateTransition(context.Background(), "gpu.stale_check", "gpu", g.id, g.status, "unavailable",
			"node_id", nodeID, "gpu_index", g.gpuIndex, "vendor", g.vendor, "health", g.health)
	}
	if n > 0 {
		log.Info("gpu.stale.marked", "node_id", nodeID, "count", n, "threshold", threshold.String())
	}
	return int(n), nil
}

// HandleListGPUs handles GET /api/gpus.
// P0-002/CODEX: Scoped to current session tenant via join with nodes.
func (h *ResourceHandler) HandleListGPUs(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	vendor := r.URL.Query().Get("vendor")
	info := auth.SessionInfoFromContext(r.Context())

	query := `SELECT g.id, g.node_id, g.vendor, g.index_num, g.name, g.uuid, g.pci_bus_id, g.driver_version,
		g.memory_total_bytes, g.memory_used_bytes, g.memory_free_bytes,
		g.gpu_utilization_percent, g.memory_utilization_percent,
		g.temperature_celsius, g.power_draw_watts,
		g.health, g.status, g.collected_at, g.created_at, g.updated_at
		FROM gpu_devices g`
	args := []interface{}{}
	// P0-002: Join nodes for tenant scoping.
	// Filter by GPU own tenant_id (inherited from node). Platform admin sees all.
	if info == nil || !info.IsPlatformAdmin {
		tid := h.DB.DefaultTenantID()
		if info != nil && info.TenantID != "" {
			tid = info.TenantID
		}
		query += " WHERE g.tenant_id = ?"
		args = append(args, tid)
	} else {
		query += " WHERE 1=1"
	}

	if nodeID != "" {
		query += " AND g.node_id = ?"
		args = append(args, nodeID)
	}
	if vendor != "" {
		query += " AND g.vendor = ?"
		args = append(args, vendor)
	}

	query += " ORDER BY g.node_id, g.index_num"

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
// REVIEW-002: Tenant-scoped access — non-admin users can only access GPUs in their tenant.
func (h *ResourceHandler) HandleGetGPU(w http.ResponseWriter, r *http.Request) {
	gpuID := r.PathValue("id")
	// Include tenant_id in the query for tenant scope check.
	var tid string
	gpu := scanGPUFromRowWithTenant(h.DB.QueryRow(
		`SELECT id, node_id, vendor, index_num, name, uuid, pci_bus_id, driver_version,
		memory_total_bytes, memory_used_bytes, memory_free_bytes,
		gpu_utilization_percent, memory_utilization_percent,
		temperature_celsius, power_draw_watts,
		health, status, tenant_id, collected_at, created_at, updated_at
		FROM gpu_devices WHERE id = ?`, gpuID,
	), &tid)
	if gpu == nil {
		http.Error(w, `{"error":"gpu not found"}`, http.StatusNotFound)
		return
	}

	// Tenant scope check: platform admin bypasses, others must match.
	info := auth.SessionInfoFromContext(r.Context())
	if info != nil && !info.IsPlatformAdmin && info.TenantID != "" && tid != info.TenantID {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
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

// scanGPUFromRowWithTenant is like scanGPUFromRow but also captures tenant_id
// into the provided *string for tenant scope checks (REVIEW-002).
func scanGPUFromRowWithTenant(row *sql.Row, tid *string) map[string]interface{} {
	var id, nodeID, vendor, name, uuid, pciBusID, driverVersion, health, status, tenantID string
	var indexNum int
	var memTotal, memUsed, memFree uint64
	var gpuUtil, memUtil, temp, power sql.NullFloat64
	var collectedAt, createdAt, updatedAt sql.NullString

	err := row.Scan(&id, &nodeID, &vendor, &indexNum, &name, &uuid, &pciBusID, &driverVersion,
		&memTotal, &memUsed, &memFree,
		&gpuUtil, &memUtil, &temp, &power,
		&health, &status, &tenantID, &collectedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil
	}

	if tid != nil {
		*tid = tenantID
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

	// P0-002/CODEX: Verify node belongs to current tenant.
	info := auth.SessionInfoFromContext(r.Context())
	if info != nil && info.TenantID != "" {
		var nodeTenant string
		if err := h.DB.QueryRow(`SELECT tenant_id FROM nodes WHERE id = ?`, nodeID).Scan(&nodeTenant); err == nil {
			if nodeTenant != info.TenantID {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
		}
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
		CPUUtilization string    `json:"cpu_utilization_percent"`
		MemoryTotal    uint64    `json:"memory_total_bytes"`
		MemoryUsed     uint64    `json:"memory_used_bytes"`
		SwapTotal      uint64    `json:"swap_total_bytes"`
		SwapUsed       uint64    `json:"swap_used_bytes"`
		UptimeSeconds  string    `json:"uptime_seconds"`
		CPUCores       int       `json:"cpu_cores"`
		Load1          string    `json:"load1"`
		Load5          string    `json:"load5"`
		Load15         string    `json:"load15"`
		CollectedAt    string    `json:"collected_at"`
		Filesystems    []FsInfo  `json:"filesystems"`
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
