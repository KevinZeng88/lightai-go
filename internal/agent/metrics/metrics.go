// Package metrics provides LightAI custom Prometheus metrics for Agent.
// /metrics reads from the latest snapshot only — never triggers real collection.
package metrics

import (
	"sync"
	"time"

	"lightai-go/internal/agent/collector"

	"github.com/prometheus/client_golang/prometheus"
)

// Snapshot holds the latest collected data for /metrics scraping.
// GPUResources is the single source of truth for both /metrics and report payload.
type Snapshot struct {
	mu sync.RWMutex

	GPUResources []collector.GPUResource // unified, vendor-neutral
	System       *collector.SystemSnapshot
	Diagnostics  []collector.CollectorDiagnosis

	CollectErrors int64
	ReportSuccess int64
	ReportErrors  int64
	LastSuccessAt time.Time
	NodeOnline    int // 1 = online

	// Agent context for metric labels.
	NodeID   string
	AgentID  string
	Hostname string
}

// NewSnapshot creates a new metrics snapshot.
func NewSnapshot(nodeID, agentID, hostname string) *Snapshot {
	return &Snapshot{NodeID: nodeID, AgentID: agentID, Hostname: hostname}
}

// SetGPUResources atomically replaces the current GPU resource snapshot.
// This is the single entry point for both /metrics and report payload.
func (s *Snapshot) SetGPUResources(gpus []collector.GPUResource) {
	s.mu.Lock()
	s.GPUResources = gpus
	s.LastSuccessAt = time.Now()
	s.mu.Unlock()
}

func (s *Snapshot) SetSystem(sys *collector.SystemSnapshot) {
	s.mu.Lock()
	s.System = sys
	s.mu.Unlock()
}

func (s *Snapshot) IncCollectErrors() {
	s.mu.Lock()
	s.CollectErrors++
	s.mu.Unlock()
}

func (s *Snapshot) IncReportSuccess() {
	s.mu.Lock()
	s.ReportSuccess++
	s.mu.Unlock()
}

func (s *Snapshot) IncReportErrors() {
	s.mu.Lock()
	s.ReportErrors++
	s.mu.Unlock()
}

func (s *Snapshot) SetOnline(v bool) {
	s.mu.Lock()
	if v {
		s.NodeOnline = 1
	} else {
		s.NodeOnline = 0
	}
	s.mu.Unlock()
}

// LastSuccessTime returns the timestamp of the last successful collection.
// P1-001: Used for data staleness detection.
func (s *Snapshot) LastSuccessTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastSuccessAt
}

// SetNodeID updates the node_id in the snapshot (called after registration).
func (s *Snapshot) SetNodeID(nodeID string) {
	s.mu.Lock()
	s.NodeID = nodeID
	s.mu.Unlock()
}

// Register registers all LightAI Agent custom metrics on the given registry.
func Register(reg *prometheus.Registry, snap *Snapshot) {
	reg.MustRegister(newGPUCollector(snap))
	reg.MustRegister(newAgentCollector(snap))
	reg.MustRegister(newHostCollector(snap))
}

// --- GPU metrics collector ---

type gpuCollector struct {
	snap *Snapshot

	memTotalDesc *prometheus.Desc
	memUsedDesc  *prometheus.Desc
	memFreeDesc  *prometheus.Desc
	gpuUtilDesc  *prometheus.Desc
	memUtilDesc  *prometheus.Desc
	tempDesc     *prometheus.Desc
	powerDesc    *prometheus.Desc
	healthDesc   *prometheus.Desc
	statusDesc   *prometheus.Desc
}

func newGPUCollector(snap *Snapshot) *gpuCollector {
	return &gpuCollector{
		snap: snap,
		memTotalDesc: prometheus.NewDesc(
			"lightai_gpu_memory_total_bytes",
			"Total GPU memory in bytes.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		memUsedDesc: prometheus.NewDesc(
			"lightai_gpu_memory_used_bytes",
			"Used GPU memory in bytes.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		memFreeDesc: prometheus.NewDesc(
			"lightai_gpu_memory_free_bytes",
			"Free GPU memory in bytes.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		gpuUtilDesc: prometheus.NewDesc(
			"lightai_gpu_utilization_percent",
			"GPU utilization percent (0-100).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		memUtilDesc: prometheus.NewDesc(
			"lightai_gpu_memory_utilization_percent",
			"GPU memory utilization percent (0-100).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		tempDesc: prometheus.NewDesc(
			"lightai_gpu_temperature_celsius",
			"GPU temperature in Celsius.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		powerDesc: prometheus.NewDesc(
			"lightai_gpu_power_draw_watts",
			"GPU power draw in Watts.",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		healthDesc: prometheus.NewDesc(
			"lightai_gpu_health_status",
			"GPU health status (1=healthy, 0=not healthy).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
		statusDesc: prometheus.NewDesc(
			"lightai_gpu_available_status",
			"GPU available status (1=available, 0=not available).",
			[]string{"vendor", "uuid", "gpu_index", "gpu_name", "node_id", "agent_id", "hostname"}, nil,
		),
	}
}

func (c *gpuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.memTotalDesc
	ch <- c.memUsedDesc
	ch <- c.memFreeDesc
	ch <- c.gpuUtilDesc
	ch <- c.memUtilDesc
	ch <- c.tempDesc
	ch <- c.powerDesc
	ch <- c.healthDesc
	ch <- c.statusDesc
}

func (c *gpuCollector) Collect(ch chan<- prometheus.Metric) {
	c.snap.mu.RLock()
	defer c.snap.mu.RUnlock()

	// Single loop over unified GPUResource — one emission per GPU per metric.
	// No separate GPUMetrics/GPUDevices loops; no duplicate time series possible.
	for _, g := range c.snap.GPUResources {
		lbls := []string{g.Vendor, g.UUID, itoa(g.Index), g.Name, c.snap.NodeID, c.snap.AgentID, c.snap.Hostname}

		ch <- prometheus.MustNewConstMetric(c.memTotalDesc, prometheus.GaugeValue, float64(g.MemoryTotalBytes), lbls...)
		ch <- prometheus.MustNewConstMetric(c.memUsedDesc, prometheus.GaugeValue, float64(g.MemoryUsedBytes), lbls...)
		ch <- prometheus.MustNewConstMetric(c.memFreeDesc, prometheus.GaugeValue, float64(g.MemoryFreeBytes), lbls...)

		if g.GPUUtilization != nil {
			ch <- prometheus.MustNewConstMetric(c.gpuUtilDesc, prometheus.GaugeValue, *g.GPUUtilization, lbls...)
		}
		if g.MemUtilization != nil {
			ch <- prometheus.MustNewConstMetric(c.memUtilDesc, prometheus.GaugeValue, *g.MemUtilization, lbls...)
		}
		if g.Temperature != nil {
			ch <- prometheus.MustNewConstMetric(c.tempDesc, prometheus.GaugeValue, *g.Temperature, lbls...)
		}
		if g.PowerDraw != nil {
			ch <- prometheus.MustNewConstMetric(c.powerDesc, prometheus.GaugeValue, *g.PowerDraw, lbls...)
		}

		healthVal := 0.0
		if g.Health == "healthy" {
			healthVal = 1.0
		}
		ch <- prometheus.MustNewConstMetric(c.healthDesc, prometheus.GaugeValue, healthVal, lbls...)

		availVal := 0.0
		if g.Status == "available" {
			availVal = 1.0
		}
		ch <- prometheus.MustNewConstMetric(c.statusDesc, prometheus.GaugeValue, availVal, lbls...)
	}
}

// --- Agent collector metrics ---

type agentCollector struct {
	snap *Snapshot

	lastSuccessDesc   *prometheus.Desc
	errorsDesc        *prometheus.Desc
	reportSuccessDesc *prometheus.Desc
	reportErrorsDesc  *prometheus.Desc
	nodeOnlineDesc    *prometheus.Desc
}

func newAgentCollector(snap *Snapshot) *agentCollector {
	return &agentCollector{
		snap: snap,
		lastSuccessDesc: prometheus.NewDesc(
			"lightai_agent_collector_last_success_timestamp_seconds",
			"Unix timestamp of last successful collector run.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		errorsDesc: prometheus.NewDesc(
			"lightai_agent_collector_errors_total",
			"Total number of collector errors.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		reportSuccessDesc: prometheus.NewDesc(
			"lightai_agent_report_success_total",
			"Total number of successful resource reports.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		reportErrorsDesc: prometheus.NewDesc(
			"lightai_agent_report_errors_total",
			"Total number of resource report errors.",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
		nodeOnlineDesc: prometheus.NewDesc(
			"lightai_node_online",
			"Node online status (1=online, 0=offline).",
			[]string{"node_id", "agent_id", "hostname"}, nil,
		),
	}
}

func (c *agentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.lastSuccessDesc
	ch <- c.errorsDesc
	ch <- c.reportSuccessDesc
	ch <- c.reportErrorsDesc
	ch <- c.nodeOnlineDesc
}

func (c *agentCollector) Collect(ch chan<- prometheus.Metric) {
	c.snap.mu.RLock()
	defer c.snap.mu.RUnlock()

	lbls := []string{c.snap.NodeID, c.snap.AgentID, c.snap.Hostname}
	ts := float64(c.snap.LastSuccessAt.Unix())
	if ts > 0 {
		ch <- prometheus.MustNewConstMetric(c.lastSuccessDesc, prometheus.GaugeValue, ts, lbls...)
	}
	ch <- prometheus.MustNewConstMetric(c.errorsDesc, prometheus.CounterValue, float64(c.snap.CollectErrors), lbls...)
	ch <- prometheus.MustNewConstMetric(c.reportSuccessDesc, prometheus.CounterValue, float64(c.snap.ReportSuccess), lbls...)
	ch <- prometheus.MustNewConstMetric(c.reportErrorsDesc, prometheus.CounterValue, float64(c.snap.ReportErrors), lbls...)
	ch <- prometheus.MustNewConstMetric(c.nodeOnlineDesc, prometheus.GaugeValue, float64(c.snap.NodeOnline), lbls...)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	n := i
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}


// --- Host metrics collector ---

type hostCollector struct {
	snap *Snapshot
	infoDesc, uptimeDesc, cpuCoresDesc, cpuUsageDesc *prometheus.Desc
	load1Desc, load5Desc, load15Desc *prometheus.Desc
	memTotalDesc, memUsedDesc, memUsedRatioDesc *prometheus.Desc
	swapTotalDesc, swapUsedDesc *prometheus.Desc
	fsTotalDesc, fsUsedDesc, fsAvailDesc, fsUsedRatioDesc *prometheus.Desc
}

func newHostCollector(snap *Snapshot) *hostCollector {
	lbls := []string{"node_id", "agent_id", "hostname"}
	fsLbls := []string{"node_id", "agent_id", "hostname", "mountpoint"}
	return &hostCollector{
		snap: snap,
		infoDesc: prometheus.NewDesc("lightai_host_info", "Host information.", lbls, nil),
		uptimeDesc: prometheus.NewDesc("lightai_host_uptime_seconds", "Host uptime.", lbls, nil),
		cpuCoresDesc: prometheus.NewDesc("lightai_host_cpu_cores", "CPU cores.", lbls, nil),
		cpuUsageDesc: prometheus.NewDesc("lightai_host_cpu_usage_ratio", "CPU usage ratio 0-1.", lbls, nil),
		load1Desc: prometheus.NewDesc("lightai_host_load1", "Load average 1min.", lbls, nil),
		load5Desc: prometheus.NewDesc("lightai_host_load5", "Load average 5min.", lbls, nil),
		load15Desc: prometheus.NewDesc("lightai_host_load15", "Load average 15min.", lbls, nil),
		memTotalDesc: prometheus.NewDesc("lightai_host_memory_total_bytes", "Total memory bytes.", lbls, nil),
		memUsedDesc: prometheus.NewDesc("lightai_host_memory_used_bytes", "Used memory bytes.", lbls, nil),
		memUsedRatioDesc: prometheus.NewDesc("lightai_host_memory_used_ratio", "Memory used ratio 0-1.", lbls, nil),
		swapTotalDesc: prometheus.NewDesc("lightai_host_swap_total_bytes", "Total swap bytes.", lbls, nil),
		swapUsedDesc: prometheus.NewDesc("lightai_host_swap_used_bytes", "Used swap bytes.", lbls, nil),
		fsTotalDesc: prometheus.NewDesc("lightai_host_filesystem_total_bytes", "FS total bytes.", fsLbls, nil),
		fsUsedDesc: prometheus.NewDesc("lightai_host_filesystem_used_bytes", "FS used bytes.", fsLbls, nil),
		fsAvailDesc: prometheus.NewDesc("lightai_host_filesystem_available_bytes", "FS available bytes.", fsLbls, nil),
		fsUsedRatioDesc: prometheus.NewDesc("lightai_host_filesystem_used_ratio", "FS used ratio 0-1.", fsLbls, nil),
	}
}

func (c *hostCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc; ch <- c.uptimeDesc; ch <- c.cpuCoresDesc; ch <- c.cpuUsageDesc
	ch <- c.load1Desc; ch <- c.load5Desc; ch <- c.load15Desc
	ch <- c.memTotalDesc; ch <- c.memUsedDesc; ch <- c.memUsedRatioDesc
	ch <- c.swapTotalDesc; ch <- c.swapUsedDesc
	ch <- c.fsTotalDesc; ch <- c.fsUsedDesc; ch <- c.fsAvailDesc; ch <- c.fsUsedRatioDesc
}

func (c *hostCollector) Collect(ch chan<- prometheus.Metric) {
	c.snap.mu.RLock()
	defer c.snap.mu.RUnlock()
	s := c.snap.System
	if s == nil { return }
	lbls := []string{c.snap.NodeID, c.snap.AgentID, c.snap.Hostname}

	ch <- prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue, 1, lbls...)
	ch <- prometheus.MustNewConstMetric(c.uptimeDesc, prometheus.GaugeValue, float64(s.UptimeSeconds), lbls...)
	ch <- prometheus.MustNewConstMetric(c.cpuCoresDesc, prometheus.GaugeValue, float64(s.CPUCores), lbls...)
	ch <- prometheus.MustNewConstMetric(c.cpuUsageDesc, prometheus.GaugeValue, s.CPUUtilization/100.0, lbls...)
	if s.Load1 > 0 || s.Load5 > 0 || s.Load15 > 0 {
		ch <- prometheus.MustNewConstMetric(c.load1Desc, prometheus.GaugeValue, s.Load1, lbls...)
		ch <- prometheus.MustNewConstMetric(c.load5Desc, prometheus.GaugeValue, s.Load5, lbls...)
		ch <- prometheus.MustNewConstMetric(c.load15Desc, prometheus.GaugeValue, s.Load15, lbls...)
	}
	ch <- prometheus.MustNewConstMetric(c.memTotalDesc, prometheus.GaugeValue, float64(s.MemoryTotalBytes), lbls...)
	ch <- prometheus.MustNewConstMetric(c.memUsedDesc, prometheus.GaugeValue, float64(s.MemoryUsedBytes), lbls...)
	if s.MemoryTotalBytes > 0 {
		ch <- prometheus.MustNewConstMetric(c.memUsedRatioDesc, prometheus.GaugeValue, float64(s.MemoryUsedBytes)/float64(s.MemoryTotalBytes), lbls...)
	}
	if s.SwapTotalBytes > 0 {
		ch <- prometheus.MustNewConstMetric(c.swapTotalDesc, prometheus.GaugeValue, float64(s.SwapTotalBytes), lbls...)
		ch <- prometheus.MustNewConstMetric(c.swapUsedDesc, prometheus.GaugeValue, float64(s.SwapUsedBytes), lbls...)
	}
	for _, fs := range s.Filesystems {
		if fs.MountPoint == "" { continue }
		fsl := []string{c.snap.NodeID, c.snap.AgentID, c.snap.Hostname, fs.MountPoint}
		ch <- prometheus.MustNewConstMetric(c.fsTotalDesc, prometheus.GaugeValue, float64(fs.TotalBytes), fsl...)
		ch <- prometheus.MustNewConstMetric(c.fsUsedDesc, prometheus.GaugeValue, float64(fs.UsedBytes), fsl...)
		ch <- prometheus.MustNewConstMetric(c.fsAvailDesc, prometheus.GaugeValue, float64(fs.FreeBytes), fsl...)
		if fs.TotalBytes > 0 {
			ch <- prometheus.MustNewConstMetric(c.fsUsedRatioDesc, prometheus.GaugeValue, fs.UsedPercent/100.0, fsl...)
		}
	}
}
